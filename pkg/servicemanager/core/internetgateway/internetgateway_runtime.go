/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package internetgateway

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const internetGatewayRequeueDuration = time.Minute

type internetGatewayOCIClient interface {
	CreateInternetGateway(ctx context.Context, request coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error)
	GetInternetGateway(ctx context.Context, request coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error)
	ListInternetGateways(ctx context.Context, request coresdk.ListInternetGatewaysRequest) (coresdk.ListInternetGatewaysResponse, error)
	UpdateInternetGateway(ctx context.Context, request coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error)
	DeleteInternetGateway(ctx context.Context, request coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error)
}

type internetGatewayGeneratedParityClient struct {
	manager  *InternetGatewayServiceManager
	delegate InternetGatewayServiceClient
	client   internetGatewayOCIClient
	initErr  error
}

func init() {
	generatedFactory := newInternetGatewayServiceClient
	newInternetGatewayServiceClient = func(manager *InternetGatewayServiceManager) InternetGatewayServiceClient {
		delegate := generatedFactory(manager)
		sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
		parityClient := &internetGatewayGeneratedParityClient{
			manager:  manager,
			delegate: delegate,
			client:   sdkClient,
		}
		if err != nil {
			parityClient.initErr = fmt.Errorf("initialize InternetGateway OCI client: %w", err)
		}
		return parityClient
	}
}

func (c *internetGatewayGeneratedParityClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.InternetGateway, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("internet gateway parity delegate is not configured")
	}

	trackedID := currentInternetGatewayID(resource)
	explicitRecreate := false
	if trackedID != "" {
		if c.initErr != nil {
			return c.fail(resource, c.initErr)
		}

		current, err := c.get(ctx, trackedID)
		if err != nil {
			if isInternetGatewayReadNotFoundOCI(err) {
				c.clearTrackedIdentity(resource)
				explicitRecreate = true
			} else {
				return c.fail(resource, normalizeInternetGatewayOCIError(err))
			}
		} else if internetGatewayLifecycleIsRetryable(current.LifecycleState) {
			if err := c.projectStatus(resource, current); err != nil {
				return c.fail(resource, err)
			}
			return c.applyLifecycle(resource, current)
		} else if requiresManualInternetGatewayUpdate(resource.Spec, current) {
			return c.update(ctx, resource, current)
		}
	}

	previousStatus := resource.Status
	c.clearProjectedStatus(resource)

	delegateCtx := ctx
	if explicitRecreate {
		delegateCtx = generatedruntime.WithSkipExistingBeforeCreate(delegateCtx)
	}
	response, err := c.delegate.CreateOrUpdate(delegateCtx, resource, req)
	if err != nil {
		c.restoreStatus(resource, previousStatus)
	}
	return response, err
}

func (c *internetGatewayGeneratedParityClient) Delete(ctx context.Context, resource *corev1beta1.InternetGateway) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("internet gateway parity delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *internetGatewayGeneratedParityClient) clearProjectedStatus(resource *corev1beta1.InternetGateway) {
	if resource == nil {
		return
	}

	resource.Status = corev1beta1.InternetGatewayStatus{
		OsokStatus: resource.Status.OsokStatus,
		Id:         resource.Status.Id,
	}
}

func (c *internetGatewayGeneratedParityClient) restoreStatus(resource *corev1beta1.InternetGateway, previous corev1beta1.InternetGatewayStatus) {
	if resource == nil {
		return
	}

	failedStatus := resource.Status.OsokStatus
	resource.Status = previous
	resource.Status.OsokStatus = failedStatus
}

func (c *internetGatewayGeneratedParityClient) create(ctx context.Context, resource *corev1beta1.InternetGateway) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateInternetGatewayRequest{
		CreateInternetGatewayDetails: buildCreateInternetGatewayDetails(resource.Spec),
	}

	response, err := c.client.CreateInternetGateway(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeInternetGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, response.InternetGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.InternetGateway)
}

func (c *internetGatewayGeneratedParityClient) get(ctx context.Context, ocid string) (coresdk.InternetGateway, error) {
	response, err := c.client.GetInternetGateway(ctx, coresdk.GetInternetGatewayRequest{
		IgId: common.String(ocid),
	})
	if err != nil {
		return coresdk.InternetGateway{}, err
	}
	return response.InternetGateway, nil
}

func (c *internetGatewayGeneratedParityClient) update(ctx context.Context, resource *corev1beta1.InternetGateway, current coresdk.InternetGateway) (servicemanager.OSOKResponse, error) {
	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		if err := c.projectStatus(resource, current); err != nil {
			return c.fail(resource, err)
		}
		return c.applyLifecycle(resource, current)
	}

	response, err := c.client.UpdateInternetGateway(ctx, updateRequest)
	if err != nil {
		return c.fail(resource, normalizeInternetGatewayOCIError(err))
	}
	if err := c.projectStatus(resource, response.InternetGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.InternetGateway)
}

func (c *internetGatewayGeneratedParityClient) buildUpdateRequest(resource *corev1beta1.InternetGateway, current coresdk.InternetGateway) (coresdk.UpdateInternetGatewayRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateInternetGatewayRequest{}, false, fmt.Errorf("current InternetGateway does not expose an OCI identifier")
	}

	if err := validateInternetGatewayCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateInternetGatewayRequest{}, false, err
	}

	updateDetails := coresdk.UpdateInternetGatewayDetails{}
	updateNeeded := false

	if resource.Spec.DisplayName != "" && !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		updateDetails.FreeformTags = resource.Spec.FreeformTags
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	if !boolPtrEqual(current.IsEnabled, resource.Spec.IsEnabled) {
		updateDetails.IsEnabled = common.Bool(resource.Spec.IsEnabled)
		updateNeeded = true
	}
	if resource.Spec.RouteTableId != "" && !stringPtrEqual(current.RouteTableId, resource.Spec.RouteTableId) {
		updateDetails.RouteTableId = common.String(resource.Spec.RouteTableId)
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateInternetGatewayRequest{}, false, nil
	}

	return coresdk.UpdateInternetGatewayRequest{
		IgId:                         current.Id,
		UpdateInternetGatewayDetails: updateDetails,
	}, true, nil
}

func buildCreateInternetGatewayDetails(spec corev1beta1.InternetGatewaySpec) coresdk.CreateInternetGatewayDetails {
	createDetails := coresdk.CreateInternetGatewayDetails{
		CompartmentId: common.String(spec.CompartmentId),
		IsEnabled:     common.Bool(spec.IsEnabled),
		VcnId:         common.String(spec.VcnId),
	}

	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = spec.FreeformTags
	}
	if spec.RouteTableId != "" {
		createDetails.RouteTableId = common.String(spec.RouteTableId)
	}

	return createDetails
}

func validateInternetGatewayCreateOnlyDrift(spec corev1beta1.InternetGatewaySpec, current coresdk.InternetGateway) error {
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !stringCreateOnlyMatches(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("InternetGateway create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func (c *internetGatewayGeneratedParityClient) applyLifecycle(resource *corev1beta1.InternetGateway, current coresdk.InternetGateway) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	now := time.Now()
	if status.CreatedAt == nil && current.Id != nil && strings.TrimSpace(*current.Id) != "" {
		createdAt := metav1Time(now)
		status.CreatedAt = &createdAt
	}
	updatedAt := metav1Time(now)
	status.UpdatedAt = &updatedAt
	if current.Id != nil {
		status.Ocid = shared.OCID(*current.Id)
	}

	message := internetGatewayLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.InternetGatewayLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.InternetGatewayLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: internetGatewayRequeueDuration}, nil
	case coresdk.InternetGatewayLifecycleStateTerminating, coresdk.InternetGatewayLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: internetGatewayRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("InternetGateway lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *internetGatewayGeneratedParityClient) fail(resource *corev1beta1.InternetGateway, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *internetGatewayGeneratedParityClient) markDeleted(resource *corev1beta1.InternetGateway, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *internetGatewayGeneratedParityClient) clearTrackedIdentity(resource *corev1beta1.InternetGateway) {
	resource.Status = corev1beta1.InternetGatewayStatus{}
}

func (c *internetGatewayGeneratedParityClient) markTerminating(resource *corev1beta1.InternetGateway, current coresdk.InternetGateway) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = internetGatewayLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *internetGatewayGeneratedParityClient) projectStatus(resource *corev1beta1.InternetGateway, current coresdk.InternetGateway) error {
	resource.Status = corev1beta1.InternetGatewayStatus{
		OsokStatus:     resource.Status.OsokStatus,
		CompartmentId:  stringValue(current.CompartmentId),
		Id:             stringValue(current.Id),
		LifecycleState: string(current.LifecycleState),
		VcnId:          stringValue(current.VcnId),
		DefinedTags:    convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:    stringValue(current.DisplayName),
		FreeformTags:   cloneStringMap(current.FreeformTags),
		IsEnabled:      boolValue(current.IsEnabled),
		TimeCreated:    sdkTimeString(current.TimeCreated),
		RouteTableId:   stringValue(current.RouteTableId),
	}
	return nil
}

func internetGatewayLifecycleMessage(current coresdk.InternetGateway) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "InternetGateway"
	}
	return fmt.Sprintf("InternetGateway %s is %s", name, current.LifecycleState)
}

func normalizeInternetGatewayOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isInternetGatewayReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isInternetGatewayDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func currentInternetGatewayID(resource *corev1beta1.InternetGateway) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func internetGatewayLifecycleIsRetryable(state coresdk.InternetGatewayLifecycleStateEnum) bool {
	switch state {
	case coresdk.InternetGatewayLifecycleStateProvisioning,
		coresdk.InternetGatewayLifecycleStateTerminating,
		coresdk.InternetGatewayLifecycleStateTerminated:
		return true
	default:
		return false
	}
}

func requiresManualInternetGatewayUpdate(spec corev1beta1.InternetGatewaySpec, current coresdk.InternetGateway) bool {
	if !spec.IsEnabled && !boolPtrEqual(current.IsEnabled, spec.IsEnabled) {
		return true
	}
	if spec.FreeformTags != nil && len(spec.FreeformTags) == 0 && !reflect.DeepEqual(current.FreeformTags, spec.FreeformTags) {
		return true
	}
	if spec.DefinedTags != nil && len(spec.DefinedTags) == 0 {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			return true
		}
	}
	return false
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func stringCreateOnlyMatches(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func boolPtrEqual(actual *bool, expected bool) bool {
	if actual == nil {
		return !expected
	}
	return *actual == expected
}

func metav1Time(t time.Time) metav1.Time {
	return metav1.NewTime(t)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		convertedValues := make(shared.MapValue, len(values))
		for key, value := range values {
			convertedValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedValues
	}
	return converted
}
