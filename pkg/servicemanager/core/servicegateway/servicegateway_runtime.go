/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicegateway

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
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

const serviceGatewayRequeueDuration = time.Minute

type serviceGatewayOCIClient interface {
	CreateServiceGateway(ctx context.Context, request coresdk.CreateServiceGatewayRequest) (coresdk.CreateServiceGatewayResponse, error)
	GetServiceGateway(ctx context.Context, request coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error)
	ListServiceGateways(ctx context.Context, request coresdk.ListServiceGatewaysRequest) (coresdk.ListServiceGatewaysResponse, error)
	UpdateServiceGateway(ctx context.Context, request coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error)
	DeleteServiceGateway(ctx context.Context, request coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error)
}

type serviceGatewayRuntimeClient struct {
	manager  *ServiceGatewayServiceManager
	delegate ServiceGatewayServiceClient
	client   serviceGatewayOCIClient
	initErr  error
}

type normalizedServiceGatewayService struct {
	serviceID string
}

func init() {
	registerServiceGatewayRuntimeHooksMutator(func(manager *ServiceGatewayServiceManager, hooks *ServiceGatewayRuntimeHooks) {
		applyServiceGatewayRuntimeHooks(manager, hooks, nil)
	})
}

func applyServiceGatewayRuntimeHooks(
	manager *ServiceGatewayServiceManager,
	hooks *ServiceGatewayRuntimeHooks,
	client serviceGatewayOCIClient,
) {
	if hooks == nil {
		return
	}

	if hooks.Semantics != nil {
		semantics := *hooks.Semantics
		mutation := semantics.Mutation
		mutation.ForceNew = nil
		semantics.Mutation = mutation
		hooks.Semantics = &semantics
	}

	runtimeClient := newServiceGatewayRuntimeClient(manager, nil, client)

	hooks.BuildCreateBody = func(_ context.Context, resource *corev1beta1.ServiceGateway, _ string) (any, error) {
		return buildCreateServiceGatewayDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *corev1beta1.ServiceGateway, _ string, currentResponse any) (any, bool, error) {
		current, ok := serviceGatewayFromResponse(currentResponse)
		if !ok {
			return nil, false, fmt.Errorf("unexpected ServiceGateway current response type %T", currentResponse)
		}
		request, updateNeeded, err := runtimeClient.buildUpdateRequest(resource, current)
		if err != nil {
			return nil, false, err
		}
		return request.UpdateServiceGatewayDetails, updateNeeded, nil
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = runtimeClient.clearTrackedIdentity
	hooks.StatusHooks.ProjectStatus = func(resource *corev1beta1.ServiceGateway, response any) error {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected ServiceGateway status response type %T", response)
		}
		return runtimeClient.projectStatus(resource, current)
	}
	hooks.StatusHooks.ApplyLifecycle = func(resource *corev1beta1.ServiceGateway, response any) (servicemanager.OSOKResponse, error) {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return runtimeClient.fail(resource, fmt.Errorf("unexpected ServiceGateway lifecycle response type %T", response))
		}
		return runtimeClient.applyLifecycle(resource, current)
	}
	hooks.StatusHooks.MarkDeleted = runtimeClient.markDeleted
	hooks.StatusHooks.MarkTerminating = func(resource *corev1beta1.ServiceGateway, response any) {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return
		}
		runtimeClient.markTerminating(resource, current)
	}
	hooks.ParityHooks.NormalizeDesiredState = func(resource *corev1beta1.ServiceGateway, response any) {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return
		}
		runtimeClient.normalizeEquivalentServices(resource, current)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = func(resource *corev1beta1.ServiceGateway, response any) error {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected ServiceGateway current response type %T", response)
		}
		return validateServiceGatewayCreateOnlyDrift(resource.Spec, current)
	}
	hooks.ParityHooks.RequiresParityHandling = func(resource *corev1beta1.ServiceGateway, response any) bool {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return false
		}
		return serviceGatewayRequiresLocalParity(resource, current)
	}
	hooks.ParityHooks.ApplyParityUpdate = func(ctx context.Context, resource *corev1beta1.ServiceGateway, response any) (servicemanager.OSOKResponse, error) {
		current, ok := serviceGatewayFromResponse(response)
		if !ok {
			return runtimeClient.fail(resource, fmt.Errorf("unexpected ServiceGateway parity response type %T", response))
		}
		return runtimeClient.update(ctx, resource, current)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ServiceGatewayServiceClient) ServiceGatewayServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func newServiceGatewayRuntimeClient(
	manager *ServiceGatewayServiceManager,
	delegate ServiceGatewayServiceClient,
	client serviceGatewayOCIClient,
) *serviceGatewayRuntimeClient {
	runtimeClient := &serviceGatewayRuntimeClient{
		manager:  manager,
		delegate: delegate,
		client:   client,
	}
	if runtimeClient.client != nil {
		return runtimeClient
	}

	sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
	runtimeClient.client = sdkClient
	if err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize ServiceGateway OCI client: %w", err)
	}
	return runtimeClient
}

func (c *serviceGatewayRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.ServiceGateway, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("servicegateway parity delegate is not configured")
	}

	trackedID := currentServiceGatewayID(resource)
	seedServiceGatewayTrackedOCID(resource, trackedID)
	explicitRecreate := false
	if trackedID != "" {
		if c.initErr != nil {
			return c.fail(resource, c.initErr)
		}
		if c.client == nil {
			return c.fail(resource, fmt.Errorf("servicegateway parity OCI client is not configured"))
		}

		current, err := c.get(ctx, trackedID)
		if err != nil {
			if isServiceGatewayReadNotFoundOCI(err) {
				c.clearTrackedIdentity(resource)
				explicitRecreate = true
			} else {
				return c.fail(resource, normalizeServiceGatewayOCIError(err))
			}
		} else {
			c.normalizeEquivalentServices(resource, current)

			if !serviceGatewayLifecycleIsRetryable(current.LifecycleState) {
				updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
				if err != nil {
					return c.fail(resource, err)
				}
				if serviceGatewayRequiresLocalParity(resource, current) {
					if updateNeeded {
						return c.updateWithParity(ctx, resource, updateRequest)
					}
					if err := c.projectStatus(resource, current); err != nil {
						return c.fail(resource, err)
					}
					return c.applyLifecycle(resource, current)
				}
			}
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
		return response, err
	}
	if trackedID == "" || explicitRecreate {
		if postCreateResponse, handled, err := c.applyPostCreateUpdateOnlyParity(ctx, resource); handled || err != nil {
			return postCreateResponse, err
		}
	}
	return response, nil
}

func (c *serviceGatewayRuntimeClient) update(
	ctx context.Context,
	resource *corev1beta1.ServiceGateway,
	current coresdk.ServiceGateway,
) (servicemanager.OSOKResponse, error) {
	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.applyLifecycle(resource, current)
	}

	return c.updateWithParity(ctx, resource, updateRequest)
}

func (c *serviceGatewayRuntimeClient) updateWithParity(
	ctx context.Context,
	resource *corev1beta1.ServiceGateway,
	request coresdk.UpdateServiceGatewayRequest,
) (servicemanager.OSOKResponse, error) {
	response, err := c.client.UpdateServiceGateway(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeServiceGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, response.ServiceGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.ServiceGateway)
}

func (c *serviceGatewayRuntimeClient) applyPostCreateUpdateOnlyParity(
	ctx context.Context,
	resource *corev1beta1.ServiceGateway,
) (servicemanager.OSOKResponse, bool, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{}, false, nil
	}

	current := serviceGatewayFromProjectedStatus(resource.Status)
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if serviceGatewayLifecycleIsRetryable(current.LifecycleState) {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if boolPtrEqual(current.BlockTraffic, resource.Spec.BlockTraffic) {
		return servicemanager.OSOKResponse{}, false, nil
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		response, failErr := c.fail(resource, err)
		return response, true, failErr
	}
	if !updateNeeded {
		return servicemanager.OSOKResponse{}, false, nil
	}

	response, err := c.updateWithParity(ctx, resource, updateRequest)
	return response, true, err
}

func (c *serviceGatewayRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.ServiceGateway) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("servicegateway parity delegate is not configured")
	}
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := currentServiceGatewayID(resource)
	seedServiceGatewayTrackedOCID(resource, trackedID)
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	return c.delegate.Delete(ctx, resource)
}

func (c *serviceGatewayRuntimeClient) create(ctx context.Context, resource *corev1beta1.ServiceGateway) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateServiceGatewayRequest{
		CreateServiceGatewayDetails: buildCreateServiceGatewayDetails(resource.Spec),
	}

	response, err := c.client.CreateServiceGateway(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeServiceGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, response.ServiceGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.ServiceGateway)
}

func (c *serviceGatewayRuntimeClient) get(ctx context.Context, ocid string) (coresdk.ServiceGateway, error) {
	response, err := c.client.GetServiceGateway(ctx, coresdk.GetServiceGatewayRequest{
		ServiceGatewayId: common.String(ocid),
	})
	if err != nil {
		return coresdk.ServiceGateway{}, err
	}
	return response.ServiceGateway, nil
}

func (c *serviceGatewayRuntimeClient) buildUpdateRequest(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) (coresdk.UpdateServiceGatewayRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateServiceGatewayRequest{}, false, fmt.Errorf("current ServiceGateway does not expose an OCI identifier")
	}

	if err := validateServiceGatewayCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateServiceGatewayRequest{}, false, err
	}

	updateDetails := coresdk.UpdateServiceGatewayDetails{}
	updateNeeded := false

	if !boolPtrEqual(current.BlockTraffic, resource.Spec.BlockTraffic) {
		updateDetails.BlockTraffic = common.Bool(resource.Spec.BlockTraffic)
		updateNeeded = true
	}
	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	desiredFreeformTags := desiredFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}
	desiredDefinedTags := desiredDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	if !stringPtrEqual(current.RouteTableId, resource.Spec.RouteTableId) {
		updateDetails.RouteTableId = common.String(resource.Spec.RouteTableId)
		updateNeeded = true
	}

	desiredServices := convertSpecServicesToOCI(resource.Spec.Services)
	if !normalizedServicesEqual(current.Services, desiredServices) {
		updateDetails.Services = desiredServices
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateServiceGatewayRequest{}, false, nil
	}

	return coresdk.UpdateServiceGatewayRequest{
		ServiceGatewayId:            current.Id,
		UpdateServiceGatewayDetails: updateDetails,
	}, true, nil
}

func buildCreateServiceGatewayDetails(spec corev1beta1.ServiceGatewaySpec) coresdk.CreateServiceGatewayDetails {
	createDetails := coresdk.CreateServiceGatewayDetails{
		CompartmentId: common.String(spec.CompartmentId),
		Services:      convertSpecServicesToOCI(spec.Services),
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

func desiredFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDefinedTagsForUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func validateServiceGatewayCreateOnlyDrift(spec corev1beta1.ServiceGatewaySpec, current coresdk.ServiceGateway) error {
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
	return fmt.Errorf("ServiceGateway create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func convertSpecServicesToOCI(services []corev1beta1.ServiceGatewayService) []coresdk.ServiceIdRequestDetails {
	converted := make([]coresdk.ServiceIdRequestDetails, 0, len(services))
	for _, service := range services {
		converted = append(converted, coresdk.ServiceIdRequestDetails{
			ServiceId: common.String(service.ServiceId),
		})
	}
	return converted
}

func convertOCIServicesToStatus(services []coresdk.ServiceIdResponseDetails) []corev1beta1.ServiceGatewayService {
	converted := make([]corev1beta1.ServiceGatewayService, 0, len(services))
	for _, service := range services {
		converted = append(converted, corev1beta1.ServiceGatewayService{
			ServiceId: stringValue(service.ServiceId),
		})
	}
	return converted
}

func convertStatusServicesToOCIResponse(services []corev1beta1.ServiceGatewayService) []coresdk.ServiceIdResponseDetails {
	converted := make([]coresdk.ServiceIdResponseDetails, 0, len(services))
	for _, service := range services {
		converted = append(converted, coresdk.ServiceIdResponseDetails{
			ServiceId: stringPointer(service.ServiceId),
		})
	}
	return converted
}

func normalizedServicesEqual(current []coresdk.ServiceIdResponseDetails, desired []coresdk.ServiceIdRequestDetails) bool {
	return reflect.DeepEqual(normalizeObservedServices(current), normalizeDesiredServices(desired))
}

func normalizeObservedServices(services []coresdk.ServiceIdResponseDetails) []normalizedServiceGatewayService {
	normalized := make([]normalizedServiceGatewayService, 0, len(services))
	for _, service := range services {
		normalized = append(normalized, normalizedServiceGatewayService{
			serviceID: strings.TrimSpace(stringValue(service.ServiceId)),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].serviceID < normalized[j].serviceID
	})
	return normalized
}

func normalizeDesiredServices(services []coresdk.ServiceIdRequestDetails) []normalizedServiceGatewayService {
	normalized := make([]normalizedServiceGatewayService, 0, len(services))
	for _, service := range services {
		normalized = append(normalized, normalizedServiceGatewayService{
			serviceID: strings.TrimSpace(stringValue(service.ServiceId)),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].serviceID < normalized[j].serviceID
	})
	return normalized
}

func (c *serviceGatewayRuntimeClient) applyLifecycle(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) (servicemanager.OSOKResponse, error) {
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

	message := serviceGatewayLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.ServiceGatewayLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.ServiceGatewayLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: serviceGatewayRequeueDuration}, nil
	case coresdk.ServiceGatewayLifecycleStateTerminating, coresdk.ServiceGatewayLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: serviceGatewayRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("ServiceGateway lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *serviceGatewayRuntimeClient) fail(resource *corev1beta1.ServiceGateway, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *serviceGatewayRuntimeClient) markDeleted(resource *corev1beta1.ServiceGateway, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *serviceGatewayRuntimeClient) clearTrackedIdentity(resource *corev1beta1.ServiceGateway) {
	resource.Status = corev1beta1.ServiceGatewayStatus{}
}

func (c *serviceGatewayRuntimeClient) clearProjectedStatus(resource *corev1beta1.ServiceGateway) {
	if resource == nil {
		return
	}

	resource.Status = corev1beta1.ServiceGatewayStatus{
		OsokStatus: resource.Status.OsokStatus,
	}
}

func (c *serviceGatewayRuntimeClient) restoreStatus(resource *corev1beta1.ServiceGateway, previous corev1beta1.ServiceGatewayStatus) {
	if resource == nil {
		return
	}

	failedStatus := resource.Status.OsokStatus
	resource.Status = previous
	resource.Status.OsokStatus = failedStatus
}

func (c *serviceGatewayRuntimeClient) normalizeEquivalentServices(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) {
	if resource == nil {
		return
	}
	if normalizedServicesEqual(current.Services, convertSpecServicesToOCI(resource.Spec.Services)) {
		resource.Spec.Services = convertOCIServicesToStatus(current.Services)
	}
}

func (c *serviceGatewayRuntimeClient) markTerminating(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = serviceGatewayLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *serviceGatewayRuntimeClient) projectStatus(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) error {
	resource.Status = corev1beta1.ServiceGatewayStatus{
		OsokStatus:     resource.Status.OsokStatus,
		BlockTraffic:   boolValue(current.BlockTraffic),
		CompartmentId:  stringValue(current.CompartmentId),
		Id:             stringValue(current.Id),
		LifecycleState: string(current.LifecycleState),
		Services:       convertOCIServicesToStatus(current.Services),
		VcnId:          stringValue(current.VcnId),
		DefinedTags:    convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:    stringValue(current.DisplayName),
		FreeformTags:   cloneStringMap(current.FreeformTags),
		RouteTableId:   stringValue(current.RouteTableId),
		TimeCreated:    sdkTimeString(current.TimeCreated),
	}
	return nil
}

func serviceGatewayFromResponse(response any) (coresdk.ServiceGateway, bool) {
	switch typed := response.(type) {
	case coresdk.ServiceGateway:
		return typed, true
	case coresdk.CreateServiceGatewayResponse:
		return typed.ServiceGateway, true
	case coresdk.GetServiceGatewayResponse:
		return typed.ServiceGateway, true
	case coresdk.UpdateServiceGatewayResponse:
		return typed.ServiceGateway, true
	default:
		return coresdk.ServiceGateway{}, false
	}
}

func serviceGatewayLifecycleMessage(current coresdk.ServiceGateway) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "ServiceGateway"
	}
	return fmt.Sprintf("ServiceGateway %s is %s", name, current.LifecycleState)
}

func normalizeServiceGatewayOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isServiceGatewayReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isServiceGatewayDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func currentServiceGatewayID(resource *corev1beta1.ServiceGateway) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func seedServiceGatewayTrackedOCID(resource *corev1beta1.ServiceGateway, trackedID string) {
	if resource == nil || strings.TrimSpace(trackedID) == "" {
		return
	}
	if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(trackedID)
}

func serviceGatewayLifecycleIsRetryable(state coresdk.ServiceGatewayLifecycleStateEnum) bool {
	switch state {
	case coresdk.ServiceGatewayLifecycleStateProvisioning,
		coresdk.ServiceGatewayLifecycleStateTerminating,
		coresdk.ServiceGatewayLifecycleStateTerminated:
		return true
	default:
		return false
	}
}

func serviceGatewayRequiresParityUpdate(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) bool {
	if resource == nil {
		return false
	}

	if !resource.Spec.BlockTraffic && !boolPtrEqual(current.BlockTraffic, resource.Spec.BlockTraffic) {
		return true
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" && !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		return true
	}
	if strings.TrimSpace(resource.Spec.RouteTableId) == "" && !stringPtrEqual(current.RouteTableId, resource.Spec.RouteTableId) {
		return true
	}

	desiredFreeformTags := desiredFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if len(desiredFreeformTags) == 0 && !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		return true
	}

	desiredDefinedTags := desiredDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if len(desiredDefinedTags) == 0 && !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		return true
	}

	desiredServices := convertSpecServicesToOCI(resource.Spec.Services)
	if len(desiredServices) == 0 && !normalizedServicesEqual(current.Services, desiredServices) {
		return true
	}

	return false
}

func serviceGatewayRequiresLocalParity(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) bool {
	if serviceGatewayRequiresParityUpdate(resource, current) {
		return true
	}
	return len(resource.Spec.Services) > 0 || len(current.Services) > 0
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

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
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

func serviceGatewayFromProjectedStatus(status corev1beta1.ServiceGatewayStatus) coresdk.ServiceGateway {
	current := coresdk.ServiceGateway{
		BlockTraffic:   common.Bool(status.BlockTraffic),
		CompartmentId:  stringPointer(status.CompartmentId),
		Id:             stringPointer(status.Id),
		LifecycleState: coresdk.ServiceGatewayLifecycleStateEnum(status.LifecycleState),
		Services:       convertStatusServicesToOCIResponse(status.Services),
		VcnId:          stringPointer(status.VcnId),
		DisplayName:    stringPointer(status.DisplayName),
		FreeformTags:   cloneStringMap(status.FreeformTags),
		RouteTableId:   stringPointer(status.RouteTableId),
	}
	if status.DefinedTags != nil {
		current.DefinedTags = *util.ConvertToOciDefinedTags(&status.DefinedTags)
	}
	return current
}
