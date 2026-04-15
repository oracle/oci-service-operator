/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const functionsRequeueDuration = 30 * time.Second

func functionsBadRequestCode(err error) (string, bool) {
	var badRequest errorutil.BadRequestOciError
	if !errors.As(err, &badRequest) {
		return "", false
	}
	return badRequest.ErrorCode, true
}

func safeFunctionsString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applyFunctionsCreateFailure(status *shared.OSOKStatus, err error, log loggerutil.OSOKLogger, kind string) {
	now := metav1.Now()
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), log)
	if code, ok := functionsBadRequestCode(err); ok {
		status.Message = code
		log.ErrorLog(err, fmt.Sprintf("Create %s bad request", kind))
		return
	}
	log.ErrorLog(err, fmt.Sprintf("Create %s failed", kind))
}

func reconcileFunctionsApplicationLifecycle(
	status *shared.OSOKStatus,
	instance *ocifunctions.Application,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	if instance == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}

	return markFunctionsLifecycle(
		status,
		shared.OCID(safeFunctionsString(instance.Id)),
		safeFunctionsString(instance.DisplayName),
		string(instance.LifecycleState),
		log,
		"Application",
	)
}

func reconcileFunctionsFunctionLifecycle(
	status *shared.OSOKStatus,
	instance *ocifunctions.Function,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	if instance == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}

	return markFunctionsLifecycle(
		status,
		shared.OCID(safeFunctionsString(instance.Id)),
		safeFunctionsString(instance.DisplayName),
		string(instance.LifecycleState),
		log,
		"Function",
	)
}

func markFunctionsLifecycle(
	status *shared.OSOKStatus,
	resourceID shared.OCID,
	displayName string,
	lifecycleState string,
	log loggerutil.OSOKLogger,
	kind string,
) servicemanager.OSOKResponse {
	condition := shared.Provisioning
	conditionStatus := v1.ConditionTrue
	shouldRequeue := true

	switch strings.ToUpper(strings.TrimSpace(lifecycleState)) {
	case "ACTIVE":
		condition = shared.Active
		shouldRequeue = false
	case "UPDATING":
		condition = shared.Updating
	case "FAILED", "DELETED":
		condition = shared.Failed
		conditionStatus = v1.ConditionFalse
		shouldRequeue = false
	}

	now := metav1.Now()
	if resourceID != "" {
		status.Ocid = resourceID
		if status.CreatedAt == nil && condition != shared.Failed {
			status.CreatedAt = &now
		}
	}
	status.UpdatedAt = &now
	status.Message = fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState)
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", status.Message, log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: functionsRequeueDuration,
	}
}

func buildFunctionsDetails[T any](
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource interface {
		GetNamespace() string
	},
) (T, error) {
	var details T

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, credentialClient, resource.GetNamespace())
	if err != nil {
		return details, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return details, fmt.Errorf("marshal resolved functions spec: %w", err)
	}
	if err := json.Unmarshal(payload, &details); err != nil {
		return details, fmt.Errorf("decode functions request body: %w", err)
	}
	return details, nil
}

func trimUnchangedFunctionsDetails[T any](details *T, existing any) bool {
	desiredValue := reflect.ValueOf(details)
	if !desiredValue.IsValid() || desiredValue.Kind() != reflect.Pointer || desiredValue.IsNil() {
		return false
	}

	desiredStruct := desiredValue.Elem()
	if desiredStruct.Kind() != reflect.Struct {
		return !desiredStruct.IsZero()
	}

	existingStruct, ok := indirectStructValue(reflect.ValueOf(existing))
	if !ok {
		return !desiredStruct.IsZero()
	}

	changed := false
	for i := 0; i < desiredStruct.NumField(); i++ {
		field := desiredStruct.Field(i)
		fieldType := desiredStruct.Type().Field(i)
		if !fieldType.IsExported() || field.IsZero() {
			continue
		}

		existingField := existingStruct.FieldByName(fieldType.Name)
		if existingField.IsValid() && jsonEquivalent(field.Interface(), existingField.Interface()) {
			field.Set(reflect.Zero(field.Type()))
			continue
		}

		changed = true
	}

	return changed
}

func projectFunctionsResponseIntoStatus(resource any, response any) error {
	body, ok := functionsResponseBody(response)
	if !ok || body == nil {
		return nil
	}

	statusValue, err := functionsStatusValue(resource)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal OCI response body: %w", err)
	}
	if err := json.Unmarshal(payload, statusValue.Addr().Interface()); err != nil {
		return fmt.Errorf("project OCI response body into status: %w", err)
	}
	return nil
}

func clearTrackedFunctionsStatus(resource any) {
	statusValue, err := functionsStatusValue(resource)
	if err != nil {
		return
	}

	if idField := statusValue.FieldByName("Id"); idField.IsValid() && idField.CanSet() && idField.Kind() == reflect.String {
		idField.SetString("")
	}

	status, err := functionsOSOKStatus(resource)
	if err != nil {
		return
	}
	status.Ocid = ""
}

func functionsTrackedResourceID(resource any) shared.OCID {
	status, err := functionsOSOKStatus(resource)
	if err == nil && strings.TrimSpace(string(status.Ocid)) != "" {
		return status.Ocid
	}

	statusValue, err := functionsStatusValue(resource)
	if err != nil {
		return ""
	}
	if idField := statusValue.FieldByName("Id"); idField.IsValid() && idField.Kind() == reflect.String {
		return shared.OCID(strings.TrimSpace(idField.String()))
	}
	return ""
}

func functionsOSOKStatus(resource any) (*shared.OSOKStatus, error) {
	statusValue, err := functionsStatusValue(resource)
	if err != nil {
		return nil, err
	}

	osokField := statusValue.FieldByName("OsokStatus")
	if !osokField.IsValid() {
		return nil, fmt.Errorf("resource %T status does not contain OsokStatus", resource)
	}
	if !osokField.CanAddr() {
		return nil, fmt.Errorf("resource %T OsokStatus field is not addressable", resource)
	}

	status, ok := osokField.Addr().Interface().(*shared.OSOKStatus)
	if !ok {
		return nil, fmt.Errorf("resource %T OsokStatus field has unexpected type %T", resource, osokField.Interface())
	}
	return status, nil
}

func functionsStatusValue(resource any) (reflect.Value, error) {
	value := reflect.ValueOf(resource)
	if !value.IsValid() {
		return reflect.Value{}, fmt.Errorf("functions resource is invalid")
	}
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return reflect.Value{}, fmt.Errorf("functions resource must be a non-nil pointer, got %T", resource)
	}

	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("functions resource must point to a struct, got %T", resource)
	}

	statusField := elem.FieldByName("Status")
	if !statusField.IsValid() {
		return reflect.Value{}, fmt.Errorf("functions resource %T does not contain a Status field", resource)
	}
	if statusField.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("functions resource %T Status field is %s, want struct", resource, statusField.Kind())
	}
	if !statusField.CanAddr() {
		return reflect.Value{}, fmt.Errorf("functions resource %T Status field is not addressable", resource)
	}

	return statusField, nil
}

func functionsResponseBody(response any) (any, bool) {
	if response == nil {
		return nil, false
	}

	value := reflect.ValueOf(response)
	if !value.IsValid() {
		return nil, false
	}
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, false
	}

	return response, true
}

func applicationStatusID(resource *functionsv1beta1.Application) shared.OCID {
	if resource == nil {
		return ""
	}
	return functionsTrackedResourceID(resource)
}

func functionStatusID(resource *functionsv1beta1.Function) shared.OCID {
	if resource == nil {
		return ""
	}
	return functionsTrackedResourceID(resource)
}

func indirectStructValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	return value, true
}

func jsonEquivalent(left any, right any) bool {
	leftPayload, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightPayload, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return reflect.DeepEqual(leftPayload, rightPayload)
}
