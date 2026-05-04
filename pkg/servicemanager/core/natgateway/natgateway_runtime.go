/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package natgateway

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

const natGatewayRequeueDuration = time.Minute

const (
	natGatewayPublicIPIDCreateIntentExplicit = "Explicit"
	natGatewayPublicIPIDCreateIntentOmitted  = "Omitted"
)

type natGatewayOCIClient interface {
	CreateNatGateway(ctx context.Context, request coresdk.CreateNatGatewayRequest) (coresdk.CreateNatGatewayResponse, error)
	GetNatGateway(ctx context.Context, request coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error)
	ListNatGateways(ctx context.Context, request coresdk.ListNatGatewaysRequest) (coresdk.ListNatGatewaysResponse, error)
	UpdateNatGateway(ctx context.Context, request coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error)
	DeleteNatGateway(ctx context.Context, request coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error)
}

type natGatewayRuntimeClient struct {
	manager  *NatGatewayServiceManager
	delegate NatGatewayServiceClient
	client   natGatewayOCIClient
	initErr  error
}

func init() {
	registerNatGatewayRuntimeHooksMutator(func(manager *NatGatewayServiceManager, hooks *NatGatewayRuntimeHooks) {
		applyNatGatewayRuntimeHooks(manager, hooks, nil)
	})
}

func applyNatGatewayRuntimeHooks(
	manager *NatGatewayServiceManager,
	hooks *NatGatewayRuntimeHooks,
	client natGatewayOCIClient,
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

	runtimeClient := newNatGatewayRuntimeClient(manager, nil, client)

	hooks.BuildCreateBody = func(_ context.Context, resource *corev1beta1.NatGateway, _ string) (any, error) {
		return buildCreateNatGatewayDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *corev1beta1.NatGateway, _ string, currentResponse any) (any, bool, error) {
		current, ok := natGatewayFromResponse(currentResponse)
		if !ok {
			return nil, false, fmt.Errorf("unexpected NatGateway current response type %T", currentResponse)
		}
		request, updateNeeded, err := runtimeClient.buildUpdateRequest(resource, current)
		if err != nil {
			return nil, false, err
		}
		return request.UpdateNatGatewayDetails, updateNeeded, nil
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = runtimeClient.clearTrackedIdentity
	hooks.StatusHooks.ProjectStatus = func(resource *corev1beta1.NatGateway, response any) error {
		current, ok := natGatewayFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected NatGateway status response type %T", response)
		}
		return runtimeClient.projectStatus(resource, current)
	}
	hooks.StatusHooks.ApplyLifecycle = func(resource *corev1beta1.NatGateway, response any) (servicemanager.OSOKResponse, error) {
		current, ok := natGatewayFromResponse(response)
		if !ok {
			return runtimeClient.fail(resource, fmt.Errorf("unexpected NatGateway lifecycle response type %T", response))
		}
		return runtimeClient.applyLifecycle(resource, current)
	}
	hooks.StatusHooks.MarkDeleted = runtimeClient.markDeleted
	hooks.StatusHooks.MarkTerminating = func(resource *corev1beta1.NatGateway, response any) {
		current, ok := natGatewayFromResponse(response)
		if !ok {
			return
		}
		runtimeClient.markTerminating(resource, current)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = func(resource *corev1beta1.NatGateway, response any) error {
		current, ok := natGatewayFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected NatGateway current response type %T", response)
		}
		return validateNatGatewayCreateOnlyDrift(resource, current)
	}
	hooks.ParityHooks.RequiresParityHandling = func(resource *corev1beta1.NatGateway, response any) bool {
		current, ok := natGatewayFromResponse(response)
		if !ok {
			return false
		}
		return natGatewayRequiresParityUpdate(resource, current)
	}
	hooks.ParityHooks.ApplyParityUpdate = func(ctx context.Context, resource *corev1beta1.NatGateway, response any) (servicemanager.OSOKResponse, error) {
		current, ok := natGatewayFromResponse(response)
		if !ok {
			return runtimeClient.fail(resource, fmt.Errorf("unexpected NatGateway parity response type %T", response))
		}
		return runtimeClient.update(ctx, resource, current)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate NatGatewayServiceClient) NatGatewayServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func newNatGatewayRuntimeClient(
	manager *NatGatewayServiceManager,
	delegate NatGatewayServiceClient,
	client natGatewayOCIClient,
) *natGatewayRuntimeClient {
	runtimeClient := &natGatewayRuntimeClient{
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
		runtimeClient.initErr = fmt.Errorf("initialize NatGateway OCI client: %w", err)
	}
	return runtimeClient
}

func (c *natGatewayRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.NatGateway, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("natgateway parity delegate is not configured")
	}

	trackedID := currentNatGatewayID(resource)
	seedNatGatewayTrackedOCID(resource, trackedID)
	explicitRecreate := false
	if trackedID != "" {
		if c.initErr != nil {
			return c.fail(resource, c.initErr)
		}
		if c.client == nil {
			return c.fail(resource, fmt.Errorf("natgateway parity OCI client is not configured"))
		}

		_, err := c.get(ctx, trackedID)
		if err != nil {
			if isNatGatewayReadNotFoundOCI(err) {
				c.clearTrackedIdentity(resource)
				explicitRecreate = true
			} else {
				return c.fail(resource, normalizeNatGatewayOCIError(err))
			}
		}
	}

	publicIPIDCreateIntent := natGatewayCreateIntentForReconcile(resource, trackedID, explicitRecreate)
	previousStatus := resource.Status
	c.clearProjectedStatus(resource)

	delegateCtx := ctx
	if explicitRecreate {
		delegateCtx = generatedruntime.WithSkipExistingBeforeCreate(delegateCtx)
	}

	response, err := c.delegate.CreateOrUpdate(delegateCtx, resource, req)
	if err != nil {
		c.restoreStatus(resource, previousStatus)
	} else {
		resource.Status.PublicIpIdCreateIntent = publicIPIDCreateIntent
	}
	return response, err
}

func (c *natGatewayRuntimeClient) update(
	ctx context.Context,
	resource *corev1beta1.NatGateway,
	current coresdk.NatGateway,
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

func (c *natGatewayRuntimeClient) updateWithParity(
	ctx context.Context,
	resource *corev1beta1.NatGateway,
	request coresdk.UpdateNatGatewayRequest,
) (servicemanager.OSOKResponse, error) {
	response, err := c.client.UpdateNatGateway(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeNatGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, response.NatGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.NatGateway)
}

func (c *natGatewayRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.NatGateway) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("natgateway parity delegate is not configured")
	}
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := currentNatGatewayID(resource)
	seedNatGatewayTrackedOCID(resource, trackedID)
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	return c.delegate.Delete(ctx, resource)
}

func (c *natGatewayRuntimeClient) create(ctx context.Context, resource *corev1beta1.NatGateway) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateNatGatewayRequest{
		CreateNatGatewayDetails: buildCreateNatGatewayDetails(resource.Spec),
	}

	response, err := c.client.CreateNatGateway(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeNatGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, response.NatGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.NatGateway)
}

func (c *natGatewayRuntimeClient) get(ctx context.Context, ocid string) (coresdk.NatGateway, error) {
	response, err := c.client.GetNatGateway(ctx, coresdk.GetNatGatewayRequest{
		NatGatewayId: common.String(ocid),
	})
	if err != nil {
		return coresdk.NatGateway{}, err
	}
	return response.NatGateway, nil
}

func (c *natGatewayRuntimeClient) buildUpdateRequest(resource *corev1beta1.NatGateway, current coresdk.NatGateway) (coresdk.UpdateNatGatewayRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateNatGatewayRequest{}, false, fmt.Errorf("current NatGateway does not expose an OCI identifier")
	}

	if err := validateNatGatewayCreateOnlyDrift(resource, current); err != nil {
		return coresdk.UpdateNatGatewayRequest{}, false, err
	}

	updateDetails := coresdk.UpdateNatGatewayDetails{}
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

	if !updateNeeded {
		return coresdk.UpdateNatGatewayRequest{}, false, nil
	}

	return coresdk.UpdateNatGatewayRequest{
		NatGatewayId:            current.Id,
		UpdateNatGatewayDetails: updateDetails,
	}, true, nil
}

func buildCreateNatGatewayDetails(spec corev1beta1.NatGatewaySpec) coresdk.CreateNatGatewayDetails {
	createDetails := coresdk.CreateNatGatewayDetails{
		CompartmentId: common.String(spec.CompartmentId),
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
	if spec.BlockTraffic {
		createDetails.BlockTraffic = common.Bool(spec.BlockTraffic)
	}
	if spec.PublicIpId != "" {
		createDetails.PublicIpId = common.String(spec.PublicIpId)
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

func validateNatGatewayCreateOnlyDrift(resource *corev1beta1.NatGateway, current coresdk.NatGateway) error {
	spec := resource.Spec
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !stringCreateOnlyMatches(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}
	if natGatewayShouldEnforcePublicIPIDDrift(resource) && !stringCreateOnlyMatches(current.PublicIpId, spec.PublicIpId) {
		unsupported = append(unsupported, "publicIpId")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("NatGateway create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func (c *natGatewayRuntimeClient) applyLifecycle(resource *corev1beta1.NatGateway, current coresdk.NatGateway) (servicemanager.OSOKResponse, error) {
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

	message := natGatewayLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.NatGatewayLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.NatGatewayLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: natGatewayRequeueDuration}, nil
	case coresdk.NatGatewayLifecycleStateTerminating, coresdk.NatGatewayLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: natGatewayRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("NatGateway lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *natGatewayRuntimeClient) fail(resource *corev1beta1.NatGateway, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *natGatewayRuntimeClient) markDeleted(resource *corev1beta1.NatGateway, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *natGatewayRuntimeClient) clearTrackedIdentity(resource *corev1beta1.NatGateway) {
	resource.Status = corev1beta1.NatGatewayStatus{}
}

func (c *natGatewayRuntimeClient) clearProjectedStatus(resource *corev1beta1.NatGateway) {
	if resource == nil {
		return
	}

	resource.Status = corev1beta1.NatGatewayStatus{
		OsokStatus:             resource.Status.OsokStatus,
		PublicIpIdCreateIntent: resource.Status.PublicIpIdCreateIntent,
	}
}

func (c *natGatewayRuntimeClient) restoreStatus(resource *corev1beta1.NatGateway, previous corev1beta1.NatGatewayStatus) {
	if resource == nil {
		return
	}

	failedStatus := resource.Status.OsokStatus
	resource.Status = previous
	resource.Status.OsokStatus = failedStatus
}

func (c *natGatewayRuntimeClient) markTerminating(resource *corev1beta1.NatGateway, current coresdk.NatGateway) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = natGatewayLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *natGatewayRuntimeClient) projectStatus(resource *corev1beta1.NatGateway, current coresdk.NatGateway) error {
	resource.Status = corev1beta1.NatGatewayStatus{
		OsokStatus:             resource.Status.OsokStatus,
		CompartmentId:          stringValue(current.CompartmentId),
		Id:                     stringValue(current.Id),
		BlockTraffic:           boolValue(current.BlockTraffic),
		LifecycleState:         string(current.LifecycleState),
		NatIp:                  stringValue(current.NatIp),
		TimeCreated:            sdkTimeString(current.TimeCreated),
		VcnId:                  stringValue(current.VcnId),
		DefinedTags:            convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:            stringValue(current.DisplayName),
		FreeformTags:           cloneStringMap(current.FreeformTags),
		PublicIpId:             stringValue(current.PublicIpId),
		RouteTableId:           stringValue(current.RouteTableId),
		PublicIpIdCreateIntent: natGatewayCurrentCreateIntent(resource),
	}
	return nil
}

func natGatewayFromResponse(response any) (coresdk.NatGateway, bool) {
	switch typed := response.(type) {
	case coresdk.NatGateway:
		return typed, true
	case coresdk.CreateNatGatewayResponse:
		return typed.NatGateway, true
	case coresdk.GetNatGatewayResponse:
		return typed.NatGateway, true
	case coresdk.UpdateNatGatewayResponse:
		return typed.NatGateway, true
	default:
		return coresdk.NatGateway{}, false
	}
}

func natGatewayLifecycleMessage(current coresdk.NatGateway) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "NatGateway"
	}
	return fmt.Sprintf("NatGateway %s is %s", name, current.LifecycleState)
}

func normalizeNatGatewayOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isNatGatewayReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isNatGatewayDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func currentNatGatewayID(resource *corev1beta1.NatGateway) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func seedNatGatewayTrackedOCID(resource *corev1beta1.NatGateway, trackedID string) {
	if resource == nil || strings.TrimSpace(trackedID) == "" {
		return
	}
	if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(trackedID)
}

func natGatewayLifecycleIsRetryable(state coresdk.NatGatewayLifecycleStateEnum) bool {
	switch state {
	case coresdk.NatGatewayLifecycleStateProvisioning,
		coresdk.NatGatewayLifecycleStateTerminating,
		coresdk.NatGatewayLifecycleStateTerminated:
		return true
	default:
		return false
	}
}

func natGatewayRequiresParityUpdate(resource *corev1beta1.NatGateway, current coresdk.NatGateway) bool {
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

	return false
}

func natGatewayCreateIntentForReconcile(
	resource *corev1beta1.NatGateway,
	trackedID string,
	explicitRecreate bool,
) string {
	if trackedID == "" || explicitRecreate {
		return natGatewayCreateIntentFromSpec(resource.Spec.PublicIpId)
	}
	return natGatewayCurrentCreateIntent(resource)
}

func natGatewayCreateIntentFromSpec(publicIPID string) string {
	if strings.TrimSpace(publicIPID) != "" {
		return natGatewayPublicIPIDCreateIntentExplicit
	}
	return natGatewayPublicIPIDCreateIntentOmitted
}

func natGatewayCurrentCreateIntent(resource *corev1beta1.NatGateway) string {
	if resource == nil {
		return ""
	}
	if resource.Status.PublicIpIdCreateIntent != "" {
		return resource.Status.PublicIpIdCreateIntent
	}
	if strings.TrimSpace(resource.Spec.PublicIpId) != "" {
		return natGatewayPublicIPIDCreateIntentExplicit
	}
	return ""
}

func natGatewayShouldEnforcePublicIPIDDrift(resource *corev1beta1.NatGateway) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.Spec.PublicIpId) != "" {
		return true
	}
	return natGatewayCurrentCreateIntent(resource) == natGatewayPublicIPIDCreateIntentExplicit
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
