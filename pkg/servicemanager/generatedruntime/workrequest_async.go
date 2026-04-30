/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c ServiceClient[T]) workRequestLegacyBridge() servicemanager.WorkRequestLegacyBridge {
	workRequest := c.generatedWorkRequestSemantics()
	if workRequest == nil || workRequest.LegacyFieldBridge == nil {
		return servicemanager.WorkRequestLegacyBridge{}
	}
	return servicemanager.WorkRequestLegacyBridge{
		Create: workRequest.LegacyFieldBridge.Create,
		Update: workRequest.LegacyFieldBridge.Update,
		Delete: workRequest.LegacyFieldBridge.Delete,
	}
}

func (c ServiceClient[T]) generatedWorkRequestSemantics() *WorkRequestSemantics {
	if c.config.Semantics == nil || c.config.Semantics.Async == nil {
		return nil
	}
	async := c.config.Semantics.Async
	if strings.TrimSpace(async.Strategy) != asyncStrategyWorkRequest ||
		strings.TrimSpace(async.Runtime) != asyncRuntimeGeneratedRuntime {
		return nil
	}
	return async.WorkRequest
}

func (c ServiceClient[T]) generatedWorkRequestAsyncEnabled() bool {
	return c.generatedWorkRequestSemantics() != nil
}

func (c ServiceClient[T]) generatedWorkRequestPhaseEnabled(phase shared.OSOKAsyncPhase) bool {
	workRequest := c.generatedWorkRequestSemantics()
	if workRequest == nil || phase == "" {
		return false
	}
	for _, rawPhase := range workRequest.Phases {
		if strings.TrimSpace(rawPhase) == string(phase) {
			return true
		}
	}
	return false
}

func (c ServiceClient[T]) defaultConfiguredWorkRequestPhase() shared.OSOKAsyncPhase {
	workRequest := c.generatedWorkRequestSemantics()
	if workRequest == nil || len(workRequest.Phases) != 1 {
		return ""
	}
	return shared.OSOKAsyncPhase(strings.TrimSpace(workRequest.Phases[0]))
}

func (c ServiceClient[T]) currentGeneratedWorkRequest(resource T) (string, shared.OSOKAsyncPhase) {
	if !c.generatedWorkRequestAsyncEnabled() {
		return "", ""
	}

	status, err := osokStatus(resource)
	if err != nil {
		return "", ""
	}
	workRequestID, phase := servicemanager.ResolveTrackedWorkRequest(
		status,
		resource,
		c.workRequestLegacyBridge(),
		c.defaultConfiguredWorkRequestPhase(),
	)
	if workRequestID == "" || phase == "" {
		return "", ""
	}
	return workRequestID, phase
}

func (c ServiceClient[T]) applyAsyncOperation(status *shared.OSOKStatus, resource T, current *shared.OSOKAsyncOperation) servicemanager.AsyncProjection {
	if c.generatedWorkRequestAsyncEnabled() {
		return servicemanager.ApplyAsyncOperationWithLegacyBridge(status, resource, c.workRequestLegacyBridge(), current, c.config.Log)
	}
	return servicemanager.ApplyAsyncOperation(status, current, c.config.Log)
}

func (c ServiceClient[T]) clearAsyncOperation(status *shared.OSOKStatus, resource T) {
	if c.generatedWorkRequestAsyncEnabled() {
		servicemanager.ClearAsyncOperationWithLegacyBridge(status, resource, c.workRequestLegacyBridge())
		return
	}
	servicemanager.ClearAsyncOperation(status)
}

func (c ServiceClient[T]) startGeneratedWorkRequest(resource T, response any, phase shared.OSOKAsyncPhase, identity any) (string, error) {
	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return "", err
	}
	if resourceID := responseID(response); resourceID != "" {
		if c.config.Identity.RecordTracked != nil {
			c.recordTrackedIdentity(resource, identity, resourceID)
		} else if status, err := osokStatus(resource); err == nil {
			status.Ocid = shared.OCID(resourceID)
		}
	}
	c.seedOpeningWorkRequestID(resource, response, phase)
	workRequestID, resolvedPhase := c.currentGeneratedWorkRequest(resource)
	if workRequestID != "" && resolvedPhase == phase {
		return workRequestID, nil
	}
	if workRequestID == "" {
		return "", fmt.Errorf("%s %s did not return an opc-work-request-id", c.config.Kind, phase)
	}
	return "", fmt.Errorf("%s %s returned work request %s for unexpected phase %q", c.config.Kind, phase, workRequestID, resolvedPhase)
}

func (c ServiceClient[T]) fetchGeneratedWorkRequest(ctx context.Context, workRequestID string) (any, error) {
	if c.config.Async.GetWorkRequest == nil {
		return nil, fmt.Errorf("%s workrequest async hooks require GetWorkRequest", c.config.Kind)
	}
	workRequest, err := c.config.Async.GetWorkRequest(ctx, strings.TrimSpace(workRequestID))
	if err != nil {
		return nil, normalizeOCIError(err)
	}
	if workRequest == nil {
		return nil, fmt.Errorf("%s work request %s did not return a body payload", c.config.Kind, workRequestID)
	}
	return workRequest, nil
}

func (c ServiceClient[T]) buildGeneratedWorkRequestAsyncOperation(resource T, workRequest any, explicitPhase shared.OSOKAsyncPhase) (*shared.OSOKAsyncOperation, error) {
	status, err := osokStatus(resource)
	if err != nil {
		return nil, err
	}

	rawAction := ""
	if c.config.Async.ResolveAction != nil {
		rawAction, err = c.config.Async.ResolveAction(workRequest)
		if err != nil {
			return nil, err
		}
	}

	fallbackPhase := explicitPhase
	if fallbackPhase == "" {
		_, fallbackPhase = c.currentGeneratedWorkRequest(resource)
	}
	if fallbackPhase == "" {
		fallbackPhase = c.defaultConfiguredWorkRequestPhase()
	}
	if c.config.Async.ResolvePhase != nil {
		derivedPhase, ok, err := c.config.Async.ResolvePhase(workRequest)
		if err != nil {
			return nil, err
		}
		if ok {
			if fallbackPhase != "" && fallbackPhase != derivedPhase {
				return nil, fmt.Errorf(
					"%s work request %s exposes phase %q while reconcile expected %q",
					c.config.Kind,
					workRequestStringField(workRequest, "Id"),
					derivedPhase,
					fallbackPhase,
				)
			}
			fallbackPhase = derivedPhase
		}
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, c.config.Async.Adapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        workRequestStringField(workRequest, "Status"),
		RawAction:        rawAction,
		RawOperationType: workRequestStringField(workRequest, "OperationType"),
		WorkRequestID:    workRequestStringField(workRequest, "Id"),
		PercentComplete:  workRequestFloat32Field(workRequest, "PercentComplete"),
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}

	message := strings.TrimSpace(c.generatedWorkRequestMessage(current.Phase, workRequest))
	if message != "" {
		current.Message = message
	}
	return current, nil
}

func (c ServiceClient[T]) generatedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	if c.config.Async.Message != nil {
		if message := strings.TrimSpace(c.config.Async.Message(phase, workRequest)); message != "" {
			return message
		}
	}
	workRequestID := workRequestStringField(workRequest, "Id")
	rawStatus := workRequestStringField(workRequest, "Status")
	switch {
	case phase != "" && workRequestID != "" && rawStatus != "":
		return fmt.Sprintf("%s %s work request %s is %s", c.config.Kind, phase, workRequestID, rawStatus)
	case phase != "" && rawStatus != "":
		return fmt.Sprintf("%s %s work request is %s", c.config.Kind, phase, rawStatus)
	default:
		return ""
	}
}

func (c ServiceClient[T]) markWorkRequestOperation(resource T, current *shared.OSOKAsyncOperation) servicemanager.OSOKResponse {
	status, err := osokStatus(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}

	now := metav1.Now()
	if currentID := c.currentID(resource); currentID != "" {
		status.Ocid = shared.OCID(currentID)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current != nil && current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}

	projection := c.applyAsyncOperation(status, resource, current)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: defaultRequeueDuration,
	}
}

func (c ServiceClient[T]) setWorkRequestOperation(resource T, current *shared.OSOKAsyncOperation, class shared.OSOKAsyncNormalizedClass, message string) servicemanager.OSOKResponse {
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	return c.markWorkRequestOperation(resource, &next)
}

func (c ServiceClient[T]) failWorkRequestOperation(resource T, current *shared.OSOKAsyncOperation, err error) (servicemanager.OSOKResponse, error) {
	if current == nil {
		return c.failCreateOrUpdate(resource, err)
	}
	c.recordErrorRequestID(resource, err)

	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}

	return c.setWorkRequestOperation(resource, current, class, err.Error()), err
}

func (c ServiceClient[T]) resumeGeneratedWorkRequestCreateOrUpdate(
	ctx context.Context,
	resource T,
	identity any,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.fetchGeneratedWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	currentAsync, err := c.buildGeneratedWorkRequestAsyncOperation(resource, workRequest, phase)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.setWorkRequestOperation(resource, currentAsync, shared.OSOKAsyncClassPending, c.generatedWorkRequestMessage(currentAsync.Phase, workRequest)), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s finished with status %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.RawStatus),
		)
	case shared.OSOKAsyncClassSucceeded:
		return c.completeGeneratedWorkRequestWrite(ctx, resource, identity, workRequest, currentAsync)
	default:
		return c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s projected unsupported async class %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.NormalizedClass),
		)
	}
}

func (c ServiceClient[T]) completeGeneratedWorkRequestWrite(
	ctx context.Context,
	resource T,
	identity any,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, error) {
	resourceID, err := c.resolveGeneratedWorkRequestResourceID(resource, workRequest, current.Phase)
	if err != nil {
		return c.failWorkRequestOperation(resource, current, err)
	}

	response, err := c.readResource(ctx, resource, resourceID, phaseReadPhase(string(current.Phase)))
	if err != nil {
		if current.Phase == shared.OSOKAsyncPhaseCreate && errors.Is(err, errResourceNotFound) {
			return c.setWorkRequestOperation(
				resource,
				current,
				shared.OSOKAsyncClassPending,
				fmt.Sprintf("%s create work request %s succeeded; waiting for %s %s to become readable", c.config.Kind, current.WorkRequestID, c.config.Kind, resourceID),
			), nil
		}
		return c.failWorkRequestOperation(resource, current, c.generatedWorkRequestReadError(current, resourceID, err))
	}

	return c.applySuccessWithIdentity(resource, response, fallbackConditionForAsyncPhase(current.Phase), identity)
}

func (c ServiceClient[T]) generatedWorkRequestReadError(current *shared.OSOKAsyncOperation, resourceID string, err error) error {
	if !errors.Is(err, errResourceNotFound) {
		return err
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseUpdate:
		return fmt.Errorf("%s update work request %s succeeded but %s %s is no longer readable", c.config.Kind, current.WorkRequestID, c.config.Kind, resourceID)
	case shared.OSOKAsyncPhaseDelete:
		return fmt.Errorf("%s delete work request %s succeeded but %s %s is still unresolved", c.config.Kind, current.WorkRequestID, c.config.Kind, resourceID)
	default:
		return err
	}
}

func (c ServiceClient[T]) resolveGeneratedWorkRequestResourceID(resource T, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
	if resourceID := strings.TrimSpace(c.currentID(resource)); resourceID != "" {
		return resourceID, nil
	}
	if c.config.Async.RecoverResourceID != nil {
		resourceID, err := c.config.Async.RecoverResourceID(resource, workRequest, phase)
		if err != nil {
			return "", err
		}
		if resourceID := strings.TrimSpace(resourceID); resourceID != "" {
			return resourceID, nil
		}
	}
	return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", c.config.Kind, phase, workRequestStringField(workRequest, "Id"), c.config.Kind)
}

func (c ServiceClient[T]) resumeGeneratedWorkRequestDelete(
	ctx context.Context,
	resource T,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.fetchGeneratedWorkRequest(ctx, workRequestID)
	if err != nil {
		c.recordErrorRequestID(resource, err)
		return false, err
	}

	currentAsync, err := c.buildGeneratedWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.setWorkRequestOperation(resource, currentAsync, shared.OSOKAsyncClassPending, c.generatedWorkRequestMessage(currentAsync.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, err := c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s finished with status %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.RawStatus),
		)
		return false, err
	case shared.OSOKAsyncClassSucceeded:
		return c.completeGeneratedWorkRequestDelete(ctx, resource, workRequest, currentAsync)
	default:
		_, err := c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s projected unsupported async class %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.NormalizedClass),
		)
		return false, err
	}
}

func (c ServiceClient[T]) completeGeneratedWorkRequestDelete(
	ctx context.Context,
	resource T,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, error) {
	currentID := strings.TrimSpace(c.currentID(resource))
	if currentID == "" && c.config.Async.RecoverResourceID != nil {
		recoveredID, err := c.config.Async.RecoverResourceID(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err == nil {
			currentID = strings.TrimSpace(recoveredID)
		}
	}
	if currentID == "" {
		c.markDeletedWithHooks(resource, fmt.Sprintf("OCI %s delete work request completed", c.config.Kind))
		return true, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.recordErrorRequestID(resource, err)
			c.markDeletedWithHooks(resource, "OCI resource deleted")
			return true, nil
		}
		_, deleteErr := c.failWorkRequestOperation(resource, current, err)
		return false, deleteErr
	}

	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return false, err
	}
	if semantics := c.config.Semantics; semantics != nil {
		deleteResult, err := c.applyDeletePolicy(resource, response, semantics, 0)
		return deleteResult.Deleted, err
	}
	if err := c.markTerminatingWithHooks(resource, response); err != nil {
		return false, err
	}
	return false, nil
}

func workRequestStringField(workRequest any, fieldName string) string {
	return stringFieldValue(workRequestFieldValue(workRequest, fieldName))
}

func workRequestFloat32Field(workRequest any, fieldName string) *float32 {
	value, ok := indirectValue(workRequestFieldValue(workRequest, fieldName))
	if !ok {
		return nil
	}

	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		percent := float32(value.Convert(reflect.TypeOf(float32(0))).Float())
		return &percent
	default:
		return nil
	}
}

func workRequestFieldValue(workRequest any, fieldName string) reflect.Value {
	if strings.TrimSpace(fieldName) == "" {
		return reflect.Value{}
	}

	value, ok := indirectValue(reflect.ValueOf(workRequest))
	if !ok || value.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return value.FieldByName(fieldName)
}

func validateGeneratedWorkRequestAsyncHooks[T any](cfg Config[T]) error {
	if cfg.Semantics == nil || cfg.Semantics.Async == nil {
		return nil
	}

	async := cfg.Semantics.Async
	if strings.TrimSpace(async.Strategy) != asyncStrategyWorkRequest ||
		strings.TrimSpace(async.Runtime) != asyncRuntimeGeneratedRuntime {
		return nil
	}

	var problems []string
	if cfg.Async.GetWorkRequest == nil {
		problems = append(problems, "workrequest async semantics require Async.GetWorkRequest")
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s workrequest async hooks blocked: %s", cfg.Kind, strings.Join(problems, "; "))
}
