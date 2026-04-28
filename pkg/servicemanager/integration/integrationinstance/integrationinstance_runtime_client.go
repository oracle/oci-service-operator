/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package integrationinstance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	integrationsdk "github.com/oracle/oci-go-sdk/v65/integration"
	integrationv1beta1 "github.com/oracle/oci-service-operator/api/integration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerIntegrationInstanceRuntimeHooksMutator(func(manager *IntegrationInstanceServiceManager, hooks *IntegrationInstanceRuntimeHooks) {
		applyIntegrationInstanceRuntimeHooks(hooks)
		wrapIntegrationInstanceNetworkEndpointAction(manager, hooks)
	})
}

func applyIntegrationInstanceRuntimeHooks(hooks *IntegrationInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newIntegrationInstanceRuntimeSemantics()
	hooks.Identity = generatedruntime.IdentityHooks[*integrationv1beta1.IntegrationInstance]{
		GuardExistingBeforeCreate: guardIntegrationInstanceExistingBeforeCreate,
	}
	hooks.BuildCreateBody = buildIntegrationInstanceCreateBody
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *integrationv1beta1.IntegrationInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildIntegrationInstanceUpdateBody(resource, currentResponse)
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeIntegrationInstanceDesiredState
	wrapIntegrationInstanceListPages(hooks)
	hooks.DeleteHooks.HandleError = handleIntegrationInstanceDeleteError
	wrapIntegrationInstanceDeleteConfirmation(hooks)
}

func newIntegrationInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "integration",
		FormalSlug:    "integrationinstance",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(integrationsdk.IntegrationInstanceLifecycleStateCreating)},
			UpdatingStates:     []string{string(integrationsdk.IntegrationInstanceLifecycleStateUpdating)},
			ActiveStates: []string{
				string(integrationsdk.IntegrationInstanceLifecycleStateActive),
				string(integrationsdk.IntegrationInstanceLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(integrationsdk.IntegrationInstanceLifecycleStateDeleting)},
			TerminalStates: []string{string(integrationsdk.IntegrationInstanceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"integrationInstanceType",
				"isByol",
				"messagePacks",
				"freeformTags",
				"definedTags",
				"securityAttributes",
				"isFileServerEnabled",
				"isVisualBuilderEnabled",
				"customEndpoint",
				"alternateCustomEndpoints",
				"networkEndpointDetails",
			},
			ForceNew: []string{
				"compartmentId",
				"idcsAt",
				"consumptionModel",
				"isDisasterRecoveryEnabled",
				"shape",
				"domainId",
			},
			ConflictsWith: map[string][]string{
				"idcsAt":   {"domainId"},
				"domainId": {"idcsAt"},
			},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IntegrationInstance", Action: "CreateIntegrationInstance"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IntegrationInstance", Action: "UpdateIntegrationInstance"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IntegrationInstance", Action: "DeleteIntegrationInstance"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IntegrationInstance", Action: "ListIntegrationInstances"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IntegrationInstance", Action: "GetIntegrationInstance"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IntegrationInstance", Action: "GetIntegrationInstance"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func guardIntegrationInstanceExistingBeforeCreate(_ context.Context, resource *integrationv1beta1.IntegrationInstance) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("IntegrationInstance resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapIntegrationInstanceListPages(hooks *IntegrationInstanceRuntimeHooks) {
	if hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request integrationsdk.ListIntegrationInstancesRequest) (integrationsdk.ListIntegrationInstancesResponse, error) {
		return listIntegrationInstancePages(ctx, call, request)
	}
}

func listIntegrationInstancePages(
	ctx context.Context,
	call func(context.Context, integrationsdk.ListIntegrationInstancesRequest) (integrationsdk.ListIntegrationInstancesResponse, error),
	request integrationsdk.ListIntegrationInstancesRequest,
) (integrationsdk.ListIntegrationInstancesResponse, error) {
	var combined integrationsdk.ListIntegrationInstancesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func handleIntegrationInstanceDeleteError(resource *integrationv1beta1.IntegrationInstance, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("IntegrationInstance delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapIntegrationInstanceDeleteConfirmation(hooks *IntegrationInstanceRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getIntegrationInstance := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate IntegrationInstanceServiceClient) IntegrationInstanceServiceClient {
		return integrationInstanceDeleteConfirmationClient{
			delegate:               delegate,
			getIntegrationInstance: getIntegrationInstance,
		}
	})
}

func wrapIntegrationInstanceNetworkEndpointAction(manager *IntegrationInstanceServiceManager, hooks *IntegrationInstanceRuntimeHooks) {
	if manager == nil || hooks == nil || hooks.Get.Call == nil {
		return
	}

	sdkClient, initErr := integrationsdk.NewIntegrationInstanceClientWithConfigurationProvider(manager.Provider)
	changeNetworkEndpoint := func(
		ctx context.Context,
		request integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest,
	) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error) {
		if initErr != nil {
			return integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse{}, fmt.Errorf("initialize IntegrationInstance OCI client for ChangeIntegrationInstanceNetworkEndpoint: %w", initErr)
		}
		return sdkClient.ChangeIntegrationInstanceNetworkEndpoint(ctx, request)
	}
	getIntegrationInstance := hooks.Get.Call

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate IntegrationInstanceServiceClient) IntegrationInstanceServiceClient {
		return newIntegrationInstanceNetworkEndpointActionClient(delegate, getIntegrationInstance, changeNetworkEndpoint, manager.Log)
	})
}

func newIntegrationInstanceNetworkEndpointActionClient(
	delegate IntegrationInstanceServiceClient,
	getIntegrationInstance func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error),
	changeNetworkEndpoint func(context.Context, integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error),
	log loggerutil.OSOKLogger,
) IntegrationInstanceServiceClient {
	if delegate == nil || getIntegrationInstance == nil || changeNetworkEndpoint == nil {
		return delegate
	}
	return integrationInstanceNetworkEndpointActionClient{
		delegate:               delegate,
		getIntegrationInstance: getIntegrationInstance,
		changeNetworkEndpoint:  changeNetworkEndpoint,
		log:                    log,
	}
}

type integrationInstanceNetworkEndpointActionClient struct {
	delegate               IntegrationInstanceServiceClient
	getIntegrationInstance func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error)
	changeNetworkEndpoint  func(context.Context, integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error)
	log                    loggerutil.OSOKLogger
}

func (c integrationInstanceNetworkEndpointActionClient) CreateOrUpdate(
	ctx context.Context,
	resource *integrationv1beta1.IntegrationInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	desiredEndpoint, err := integrationInstanceNetworkEndpointDetails(resource.Spec.NetworkEndpointDetails)
	if err != nil {
		return c.fail(resource, fmt.Errorf("build IntegrationInstance network endpoint change body: %w", err))
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || response.ShouldRequeue || trackedIntegrationInstanceID(resource) == "" {
		return response, err
	}

	normalizeIntegrationInstanceStatusNetworkEndpoint(resource)
	currentEndpoint, err := integrationInstanceCurrentNetworkEndpointDetails(resource.Status)
	if err != nil {
		return c.fail(resource, err)
	}
	matches, err := integrationInstanceNetworkEndpointMatchesObserved(desiredEndpoint, currentEndpoint)
	if err != nil {
		return c.fail(resource, err)
	}
	if matches {
		return response, nil
	}

	return c.changeEndpoint(ctx, resource, desiredEndpoint)
}

func (c integrationInstanceNetworkEndpointActionClient) Delete(
	ctx context.Context,
	resource *integrationv1beta1.IntegrationInstance,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c integrationInstanceNetworkEndpointActionClient) changeEndpoint(
	ctx context.Context,
	resource *integrationv1beta1.IntegrationInstance,
	desiredEndpoint integrationsdk.NetworkEndpointDetails,
) (servicemanager.OSOKResponse, error) {
	integrationInstanceID := trackedIntegrationInstanceID(resource)
	changeResponse, err := c.changeNetworkEndpoint(ctx, integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest{
		IntegrationInstanceId: common.String(integrationInstanceID),
		ChangeIntegrationInstanceNetworkEndpointDetails: integrationsdk.ChangeIntegrationInstanceNetworkEndpointDetails{
			NetworkEndpointDetails: desiredEndpoint,
		},
		OpcRetryToken: integrationInstanceNetworkEndpointRetryToken(resource),
	})
	if err != nil {
		return c.fail(resource, err)
	}

	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, changeResponse)
	c.markPendingNetworkEndpointChange(resource, changeResponse, "OCI IntegrationInstance network endpoint change accepted")

	refreshed, err := c.getIntegrationInstance(ctx, integrationsdk.GetIntegrationInstanceRequest{
		IntegrationInstanceId: common.String(integrationInstanceID),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	if err := projectIntegrationInstanceStatus(resource, refreshed.IntegrationInstance); err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, changeResponse)

	return c.finishNetworkEndpointChange(resource, refreshed.IntegrationInstance, desiredEndpoint, changeResponse)
}

func (c integrationInstanceNetworkEndpointActionClient) finishNetworkEndpointChange(
	resource *integrationv1beta1.IntegrationInstance,
	refreshed integrationsdk.IntegrationInstance,
	desiredEndpoint integrationsdk.NetworkEndpointDetails,
	changeResponse integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse,
) (servicemanager.OSOKResponse, error) {
	message := integrationInstanceLifecycleMessage(refreshed)
	workRequestID := integrationInstanceStringValue(changeResponse.OpcWorkRequestId)
	current := servicemanager.NewLifecycleAsyncOperation(
		&resource.Status.OsokStatus,
		string(refreshed.LifecycleState),
		message,
		shared.OSOKAsyncPhaseUpdate,
	)
	if current != nil {
		if workRequestID != "" {
			current.WorkRequestID = workRequestID
		}
		now := metav1.Now()
		current.UpdatedAt = &now
		projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: time.Minute,
		}, nil
	}

	matches, err := integrationInstanceNetworkEndpointMatchesObserved(desiredEndpoint, refreshed.NetworkEndpointDetails)
	if err != nil {
		return c.fail(resource, err)
	}
	if !matches {
		return c.markPendingNetworkEndpointChange(resource, changeResponse, "OCI IntegrationInstance network endpoint change is pending"), nil
	}

	return c.markCondition(resource, shared.Active, "IntegrationInstance network endpoint is current"), nil
}

func (c integrationInstanceNetworkEndpointActionClient) markPendingNetworkEndpointChange(
	resource *integrationv1beta1.IntegrationInstance,
	response integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse,
	message string,
) servicemanager.OSOKResponse {
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   integrationInstanceStringValue(response.OpcWorkRequestId),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func (c integrationInstanceNetworkEndpointActionClient) markCondition(
	resource *integrationv1beta1.IntegrationInstance,
	condition shared.OSOKConditionType,
	message string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatusForIntegrationInstance(condition), "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating,
		RequeueDuration: time.Minute,
	}
}

func (c integrationInstanceNetworkEndpointActionClient) fail(
	resource *integrationv1beta1.IntegrationInstance,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil && err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markCondition(resource, shared.Failed, err.Error())
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

type integrationInstanceDeleteConfirmationClient struct {
	delegate               IntegrationInstanceServiceClient
	getIntegrationInstance func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error)
}

func (c integrationInstanceDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *integrationv1beta1.IntegrationInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c integrationInstanceDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *integrationv1beta1.IntegrationInstance,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c integrationInstanceDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *integrationv1beta1.IntegrationInstance,
) error {
	if c.getIntegrationInstance == nil || resource == nil {
		return nil
	}
	integrationInstanceID := trackedIntegrationInstanceID(resource)
	if integrationInstanceID == "" {
		return nil
	}
	_, err := c.getIntegrationInstance(ctx, integrationsdk.GetIntegrationInstanceRequest{
		IntegrationInstanceId: common.String(integrationInstanceID),
	})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("IntegrationInstance delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedIntegrationInstanceID(resource *integrationv1beta1.IntegrationInstance) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func projectIntegrationInstanceStatus(
	resource *integrationv1beta1.IntegrationInstance,
	observed integrationsdk.IntegrationInstance,
) error {
	if resource == nil {
		return fmt.Errorf("IntegrationInstance resource is nil")
	}

	payload, err := json.Marshal(observed)
	if err != nil {
		return fmt.Errorf("marshal IntegrationInstance readback: %w", err)
	}
	if err := json.Unmarshal(payload, &resource.Status); err != nil {
		return fmt.Errorf("project IntegrationInstance readback into status: %w", err)
	}
	if observed.NetworkEndpointDetails == nil {
		resource.Status.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{}
	}
	normalizeIntegrationInstanceStatusNetworkEndpoint(resource)
	if id := integrationInstanceStringValue(observed.Id); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
		if resource.Status.OsokStatus.CreatedAt == nil {
			now := metav1.Now()
			resource.Status.OsokStatus.CreatedAt = &now
		}
	}
	return nil
}

func integrationInstanceLifecycleMessage(observed integrationsdk.IntegrationInstance) string {
	if message := strings.TrimSpace(integrationInstanceStringValue(observed.StateMessage)); message != "" {
		return message
	}
	return strings.TrimSpace(integrationInstanceStringValue(observed.LifecycleDetails))
}

func integrationInstanceNetworkEndpointRetryToken(resource *integrationv1beta1.IntegrationInstance) *string {
	if resource == nil {
		return nil
	}
	base := strings.TrimSpace(string(resource.UID))
	if base == "" {
		namespace := strings.TrimSpace(resource.Namespace)
		name := strings.TrimSpace(resource.Name)
		if namespace == "" && name == "" {
			return nil
		}
		sum := sha256.Sum256([]byte(namespace + "/" + name))
		base = fmt.Sprintf("%x", sum[:16])
	}
	return common.String(base + "-network-endpoint")
}

func conditionStatusForIntegrationInstance(condition shared.OSOKConditionType) v1.ConditionStatus {
	if condition == shared.Failed {
		return v1.ConditionFalse
	}
	return v1.ConditionTrue
}

func buildIntegrationInstanceCreateBody(
	_ context.Context,
	resource *integrationv1beta1.IntegrationInstance,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("IntegrationInstance resource is nil")
	}

	customEndpoint, err := integrationInstanceCreateCustomEndpoint(resource.Spec.CustomEndpoint)
	if err != nil {
		return nil, err
	}
	alternateCustomEndpoints, err := integrationInstanceCreateAlternateCustomEndpoints(resource.Spec.AlternateCustomEndpoints)
	if err != nil {
		return nil, err
	}
	networkEndpoint, err := integrationInstanceNetworkEndpointDetails(resource.Spec.NetworkEndpointDetails)
	if err != nil {
		return nil, err
	}

	return integrationsdk.CreateIntegrationInstanceDetails{
		DisplayName:               common.String(resource.Spec.DisplayName),
		CompartmentId:             common.String(resource.Spec.CompartmentId),
		IntegrationInstanceType:   integrationsdk.CreateIntegrationInstanceDetailsIntegrationInstanceTypeEnum(resource.Spec.IntegrationInstanceType),
		IsByol:                    common.Bool(resource.Spec.IsByol),
		MessagePacks:              common.Int(resource.Spec.MessagePacks),
		FreeformTags:              maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:               integrationInstanceMapValueMap(resource.Spec.DefinedTags),
		SecurityAttributes:        integrationInstanceMapValueMap(resource.Spec.SecurityAttributes),
		IdcsAt:                    integrationInstanceStringPtr(resource.Spec.IdcsAt),
		IsVisualBuilderEnabled:    integrationInstanceOptionalBoolPtr(resource.Spec.IsVisualBuilderEnabled),
		CustomEndpoint:            customEndpoint,
		AlternateCustomEndpoints:  alternateCustomEndpoints,
		ConsumptionModel:          integrationsdk.CreateIntegrationInstanceDetailsConsumptionModelEnum(resource.Spec.ConsumptionModel),
		IsFileServerEnabled:       integrationInstanceOptionalBoolPtr(resource.Spec.IsFileServerEnabled),
		IsDisasterRecoveryEnabled: integrationInstanceOptionalBoolPtr(resource.Spec.IsDisasterRecoveryEnabled),
		NetworkEndpointDetails:    networkEndpoint,
		Shape:                     integrationsdk.CreateIntegrationInstanceDetailsShapeEnum(resource.Spec.Shape),
		DomainId:                  integrationInstanceStringPtr(resource.Spec.DomainId),
	}, nil
}

func buildIntegrationInstanceUpdateBody(
	resource *integrationv1beta1.IntegrationInstance,
	currentResponse any,
) (integrationsdk.UpdateIntegrationInstanceDetails, bool, error) {
	if resource == nil {
		return integrationsdk.UpdateIntegrationInstanceDetails{}, false, fmt.Errorf("IntegrationInstance resource is nil")
	}

	current, err := integrationInstanceRuntimeBody(currentResponse)
	if err != nil {
		return integrationsdk.UpdateIntegrationInstanceDetails{}, false, err
	}

	details := integrationsdk.UpdateIntegrationInstanceDetails{}
	updateNeeded := integrationInstanceApplyScalarUpdates(&details, resource.Spec, current)
	updateNeeded = integrationInstanceApplyTagUpdates(&details, resource.Spec, current) || updateNeeded
	endpointsUpdated, err := integrationInstanceApplyEndpointUpdates(&details, resource.Spec, current)
	if err != nil {
		return integrationsdk.UpdateIntegrationInstanceDetails{}, false, err
	}
	return details, updateNeeded || endpointsUpdated, nil
}

func integrationInstanceApplyScalarUpdates(
	details *integrationsdk.UpdateIntegrationInstanceDetails,
	spec integrationv1beta1.IntegrationInstanceSpec,
	current integrationsdk.IntegrationInstance,
) bool {
	updateNeeded := false
	if desired, ok := integrationInstanceDesiredStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredTypeUpdate(spec.IntegrationInstanceType, current.IntegrationInstanceType); ok {
		details.IntegrationInstanceType = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredRequiredBoolUpdate(spec.IsByol, current.IsByol); ok {
		details.IsByol = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredRequiredIntUpdate(spec.MessagePacks, current.MessagePacks); ok {
		details.MessagePacks = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredOptionalBoolUpdate(spec.IsFileServerEnabled, current.IsFileServerEnabled); ok {
		details.IsFileServerEnabled = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredOptionalBoolUpdate(spec.IsVisualBuilderEnabled, current.IsVisualBuilderEnabled); ok {
		details.IsVisualBuilderEnabled = desired
		updateNeeded = true
	}
	return updateNeeded
}

func integrationInstanceApplyTagUpdates(
	details *integrationsdk.UpdateIntegrationInstanceDetails,
	spec integrationv1beta1.IntegrationInstanceSpec,
	current integrationsdk.IntegrationInstance,
) bool {
	updateNeeded := false
	if desired, ok := integrationInstanceDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredMapValueUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if desired, ok := integrationInstanceDesiredMapValueUpdate(spec.SecurityAttributes, current.SecurityAttributes); ok {
		details.SecurityAttributes = desired
		updateNeeded = true
	}
	return updateNeeded
}

func integrationInstanceApplyEndpointUpdates(
	details *integrationsdk.UpdateIntegrationInstanceDetails,
	spec integrationv1beta1.IntegrationInstanceSpec,
	current integrationsdk.IntegrationInstance,
) (bool, error) {
	updateNeeded := false
	if desired, ok, err := integrationInstanceDesiredCustomEndpointUpdate(spec.CustomEndpoint, current.CustomEndpoint); err != nil {
		return false, err
	} else if ok {
		details.CustomEndpoint = desired
		updateNeeded = true
	}
	if desired, ok, err := integrationInstanceDesiredAlternateCustomEndpointsUpdate(spec.AlternateCustomEndpoints, current.AlternateCustomEndpoints); err != nil {
		return false, err
	} else if ok {
		details.AlternateCustomEndpoints = desired
		updateNeeded = true
	}
	return updateNeeded, nil
}

func integrationInstanceRuntimeBody(currentResponse any) (integrationsdk.IntegrationInstance, error) {
	switch current := currentResponse.(type) {
	case integrationsdk.IntegrationInstance:
		return current, nil
	case *integrationsdk.IntegrationInstance:
		if current == nil {
			return integrationsdk.IntegrationInstance{}, fmt.Errorf("current IntegrationInstance response is nil")
		}
		return *current, nil
	case integrationsdk.IntegrationInstanceSummary:
		return integrationInstanceFromSummary(current), nil
	case *integrationsdk.IntegrationInstanceSummary:
		if current == nil {
			return integrationsdk.IntegrationInstance{}, fmt.Errorf("current IntegrationInstance response is nil")
		}
		return integrationInstanceFromSummary(*current), nil
	case integrationsdk.GetIntegrationInstanceResponse:
		return current.IntegrationInstance, nil
	case *integrationsdk.GetIntegrationInstanceResponse:
		if current == nil {
			return integrationsdk.IntegrationInstance{}, fmt.Errorf("current IntegrationInstance response is nil")
		}
		return current.IntegrationInstance, nil
	default:
		return integrationsdk.IntegrationInstance{}, fmt.Errorf("unexpected current IntegrationInstance response type %T", currentResponse)
	}
}

func integrationInstanceFromSummary(summary integrationsdk.IntegrationInstanceSummary) integrationsdk.IntegrationInstance {
	return integrationsdk.IntegrationInstance{
		Id:                                summary.Id,
		DisplayName:                       summary.DisplayName,
		CompartmentId:                     summary.CompartmentId,
		IntegrationInstanceType:           integrationsdk.IntegrationInstanceIntegrationInstanceTypeEnum(summary.IntegrationInstanceType),
		IsByol:                            summary.IsByol,
		InstanceUrl:                       summary.InstanceUrl,
		MessagePacks:                      summary.MessagePacks,
		TimeCreated:                       summary.TimeCreated,
		TimeUpdated:                       summary.TimeUpdated,
		LifecycleState:                    integrationsdk.IntegrationInstanceLifecycleStateEnum(summary.LifecycleState),
		LifecycleDetails:                  summary.LifecycleDetails,
		StateMessage:                      summary.StateMessage,
		FreeformTags:                      maps.Clone(summary.FreeformTags),
		DefinedTags:                       integrationInstanceCloneSDKMap(summary.DefinedTags),
		SystemTags:                        integrationInstanceCloneSDKMap(summary.SystemTags),
		SecurityAttributes:                integrationInstanceCloneSDKMap(summary.SecurityAttributes),
		InstanceDesignTimeUrl:             summary.InstanceDesignTimeUrl,
		IsFileServerEnabled:               summary.IsFileServerEnabled,
		IsVisualBuilderEnabled:            summary.IsVisualBuilderEnabled,
		CustomEndpoint:                    summary.CustomEndpoint,
		AlternateCustomEndpoints:          append([]integrationsdk.CustomEndpointDetails(nil), summary.AlternateCustomEndpoints...),
		ConsumptionModel:                  integrationsdk.IntegrationInstanceConsumptionModelEnum(summary.ConsumptionModel),
		NetworkEndpointDetails:            summary.NetworkEndpointDetails,
		Shape:                             integrationsdk.IntegrationInstanceShapeEnum(summary.Shape),
		PrivateEndpointOutboundConnection: summary.PrivateEndpointOutboundConnection,
		IsDisasterRecoveryEnabled:         summary.IsDisasterRecoveryEnabled,
		DataRetentionPeriod:               integrationsdk.IntegrationInstanceDataRetentionPeriodEnum(summary.DataRetentionPeriod),
		LogGroupId:                        summary.LogGroupId,
	}
}

func integrationInstanceCreateCustomEndpoint(
	spec integrationv1beta1.IntegrationInstanceCustomEndpoint,
) (*integrationsdk.CreateCustomEndpointDetails, error) {
	if !integrationInstanceCustomEndpointMeaningful(spec) {
		return nil, nil
	}
	if strings.TrimSpace(spec.Hostname) == "" {
		return nil, fmt.Errorf("customEndpoint.hostname is required when customEndpoint is configured")
	}
	return &integrationsdk.CreateCustomEndpointDetails{
		Hostname:            common.String(spec.Hostname),
		CertificateSecretId: integrationInstanceStringPtr(spec.CertificateSecretId),
	}, nil
}

func integrationInstanceUpdateCustomEndpoint(
	spec integrationv1beta1.IntegrationInstanceCustomEndpoint,
) (*integrationsdk.UpdateCustomEndpointDetails, error) {
	if !integrationInstanceCustomEndpointMeaningful(spec) {
		return nil, nil
	}
	if strings.TrimSpace(spec.Hostname) == "" {
		return nil, fmt.Errorf("customEndpoint.hostname is required when customEndpoint is configured")
	}
	return &integrationsdk.UpdateCustomEndpointDetails{
		Hostname:            common.String(spec.Hostname),
		CertificateSecretId: integrationInstanceStringPtr(spec.CertificateSecretId),
	}, nil
}

func integrationInstanceCreateAlternateCustomEndpoints(
	spec []integrationv1beta1.IntegrationInstanceAlternateCustomEndpoint,
) ([]integrationsdk.CreateCustomEndpointDetails, error) {
	if len(spec) == 0 {
		return nil, nil
	}

	endpoints := make([]integrationsdk.CreateCustomEndpointDetails, 0, len(spec))
	for index, endpoint := range spec {
		if strings.TrimSpace(endpoint.Hostname) == "" {
			return nil, fmt.Errorf("alternateCustomEndpoints[%d].hostname is required", index)
		}
		endpoints = append(endpoints, integrationsdk.CreateCustomEndpointDetails{
			Hostname:            common.String(endpoint.Hostname),
			CertificateSecretId: integrationInstanceStringPtr(endpoint.CertificateSecretId),
		})
	}
	return endpoints, nil
}

func integrationInstanceUpdateAlternateCustomEndpoints(
	spec []integrationv1beta1.IntegrationInstanceAlternateCustomEndpoint,
) ([]integrationsdk.UpdateCustomEndpointDetails, error) {
	endpoints := make([]integrationsdk.UpdateCustomEndpointDetails, 0, len(spec))
	for index, endpoint := range spec {
		if strings.TrimSpace(endpoint.Hostname) == "" {
			return nil, fmt.Errorf("alternateCustomEndpoints[%d].hostname is required", index)
		}
		endpoints = append(endpoints, integrationsdk.UpdateCustomEndpointDetails{
			Hostname:            common.String(endpoint.Hostname),
			CertificateSecretId: integrationInstanceStringPtr(endpoint.CertificateSecretId),
		})
	}
	return endpoints, nil
}

func integrationInstanceNetworkEndpointDetails(
	spec integrationv1beta1.IntegrationInstanceNetworkEndpointDetails,
) (integrationsdk.NetworkEndpointDetails, error) {
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		return integrationInstanceNetworkEndpointFromJSONData(raw)
	}
	if !integrationInstanceNetworkEndpointMeaningful(spec) {
		return nil, nil
	}

	endpointType := strings.ToUpper(strings.TrimSpace(spec.NetworkEndpointType))
	if endpointType == "" {
		endpointType = string(integrationsdk.NetworkEndpointTypePublic)
	}
	if endpointType != string(integrationsdk.NetworkEndpointTypePublic) {
		return nil, fmt.Errorf("networkEndpointDetails.networkEndpointType %q is not supported", spec.NetworkEndpointType)
	}

	return integrationsdk.PublicEndpointDetails{
		AllowlistedHttpIps:          append([]string(nil), spec.AllowlistedHttpIps...),
		AllowlistedHttpVcns:         integrationInstanceVirtualCloudNetworks(spec.AllowlistedHttpVcns),
		Runtime:                     integrationInstanceComponentAllowList(spec.Runtime),
		DesignTime:                  integrationInstanceDesignTimeComponentAllowList(spec.DesignTime),
		IsIntegrationVcnAllowlisted: integrationInstanceOptionalBoolPtr(spec.IsIntegrationVcnAllowlisted),
	}, nil
}

func integrationInstanceNetworkEndpointFromJSONData(raw string) (integrationsdk.NetworkEndpointDetails, error) {
	var discriminator struct {
		NetworkEndpointType string `json:"networkEndpointType"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode networkEndpointDetails jsonData discriminator: %w", err)
	}

	endpointType := strings.ToUpper(strings.TrimSpace(discriminator.NetworkEndpointType))
	if endpointType == "" {
		endpointType = string(integrationsdk.NetworkEndpointTypePublic)
	}
	if endpointType != string(integrationsdk.NetworkEndpointTypePublic) {
		return nil, fmt.Errorf("networkEndpointDetails.networkEndpointType %q is not supported", discriminator.NetworkEndpointType)
	}

	var details integrationsdk.PublicEndpointDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return nil, fmt.Errorf("decode PublicEndpointDetails jsonData: %w", err)
	}
	return details, nil
}

func integrationInstanceVirtualCloudNetworks(
	spec []integrationv1beta1.IntegrationInstanceNetworkEndpointDetailsAllowlistedHttpVcn,
) []integrationsdk.VirtualCloudNetwork {
	if len(spec) == 0 {
		return nil
	}
	vcns := make([]integrationsdk.VirtualCloudNetwork, 0, len(spec))
	for _, vcn := range spec {
		vcns = append(vcns, integrationsdk.VirtualCloudNetwork{
			Id:             integrationInstanceStringPtr(vcn.Id),
			AllowlistedIps: append([]string(nil), vcn.AllowlistedIps...),
		})
	}
	return vcns
}

func integrationInstanceRuntimeVirtualCloudNetworks(
	spec []integrationv1beta1.IntegrationInstanceNetworkEndpointDetailsRuntimeAllowlistedHttpVcn,
) []integrationsdk.VirtualCloudNetwork {
	if len(spec) == 0 {
		return nil
	}
	vcns := make([]integrationsdk.VirtualCloudNetwork, 0, len(spec))
	for _, vcn := range spec {
		vcns = append(vcns, integrationsdk.VirtualCloudNetwork{
			Id:             integrationInstanceStringPtr(vcn.Id),
			AllowlistedIps: append([]string(nil), vcn.AllowlistedIps...),
		})
	}
	return vcns
}

func integrationInstanceDesignTimeVirtualCloudNetworks(
	spec []integrationv1beta1.IntegrationInstanceNetworkEndpointDetailsDesignTimeAllowlistedHttpVcn,
) []integrationsdk.VirtualCloudNetwork {
	if len(spec) == 0 {
		return nil
	}
	vcns := make([]integrationsdk.VirtualCloudNetwork, 0, len(spec))
	for _, vcn := range spec {
		vcns = append(vcns, integrationsdk.VirtualCloudNetwork{
			Id:             integrationInstanceStringPtr(vcn.Id),
			AllowlistedIps: append([]string(nil), vcn.AllowlistedIps...),
		})
	}
	return vcns
}

func integrationInstanceComponentAllowList(
	spec integrationv1beta1.IntegrationInstanceNetworkEndpointDetailsRuntime,
) *integrationsdk.ComponentAllowListDetails {
	if len(spec.AllowlistedHttpIps) == 0 && len(spec.AllowlistedHttpVcns) == 0 {
		return nil
	}
	return &integrationsdk.ComponentAllowListDetails{
		AllowlistedHttpIps:  append([]string(nil), spec.AllowlistedHttpIps...),
		AllowlistedHttpVcns: integrationInstanceRuntimeVirtualCloudNetworks(spec.AllowlistedHttpVcns),
	}
}

func integrationInstanceDesignTimeComponentAllowList(
	spec integrationv1beta1.IntegrationInstanceNetworkEndpointDetailsDesignTime,
) *integrationsdk.ComponentAllowListDetails {
	if len(spec.AllowlistedHttpIps) == 0 && len(spec.AllowlistedHttpVcns) == 0 {
		return nil
	}
	return &integrationsdk.ComponentAllowListDetails{
		AllowlistedHttpIps:  append([]string(nil), spec.AllowlistedHttpIps...),
		AllowlistedHttpVcns: integrationInstanceDesignTimeVirtualCloudNetworks(spec.AllowlistedHttpVcns),
	}
}

func integrationInstanceDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func integrationInstanceDesiredTypeUpdate(
	spec string,
	current integrationsdk.IntegrationInstanceIntegrationInstanceTypeEnum,
) (integrationsdk.UpdateIntegrationInstanceDetailsIntegrationInstanceTypeEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return integrationsdk.UpdateIntegrationInstanceDetailsIntegrationInstanceTypeEnum(spec), true
}

func integrationInstanceDesiredRequiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Bool(spec), true
}

func integrationInstanceDesiredOptionalBoolUpdate(spec bool, current *bool) (*bool, bool) {
	if current == nil && !spec {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Bool(spec), true
}

func integrationInstanceDesiredRequiredIntUpdate(spec int, current *int) (*int, bool) {
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Int(spec), true
}

func integrationInstanceDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func integrationInstanceDesiredMapValueUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := integrationInstanceMapValueMap(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if integrationInstanceJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func integrationInstanceDesiredCustomEndpointUpdate(
	spec integrationv1beta1.IntegrationInstanceCustomEndpoint,
	current *integrationsdk.CustomEndpointDetails,
) (*integrationsdk.UpdateCustomEndpointDetails, bool, error) {
	if !integrationInstanceCustomEndpointMeaningful(spec) {
		return nil, false, nil
	}
	if current != nil && integrationInstanceCustomEndpointMatchesObserved(spec, *current) {
		return nil, false, nil
	}
	endpoint, err := integrationInstanceUpdateCustomEndpoint(spec)
	if err != nil {
		return nil, false, err
	}
	return endpoint, true, nil
}

func integrationInstanceDesiredAlternateCustomEndpointsUpdate(
	spec []integrationv1beta1.IntegrationInstanceAlternateCustomEndpoint,
	current []integrationsdk.CustomEndpointDetails,
) ([]integrationsdk.UpdateCustomEndpointDetails, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if integrationInstanceAlternateCustomEndpointsMatchObserved(spec, current) {
		return nil, false, nil
	}
	endpoints, err := integrationInstanceUpdateAlternateCustomEndpoints(spec)
	if err != nil {
		return nil, false, err
	}
	return endpoints, true, nil
}

func normalizeIntegrationInstanceDesiredState(resource *integrationv1beta1.IntegrationInstance, currentResponse any) {
	if resource == nil || strings.TrimSpace(resource.Spec.NetworkEndpointDetails.JsonData) == "" {
		return
	}

	endpoint, err := integrationInstanceNetworkEndpointFromJSONData(resource.Spec.NetworkEndpointDetails.JsonData)
	if err != nil {
		return
	}
	if current, err := integrationInstanceCurrentNetworkEndpointDetails(currentResponse); err == nil && current != nil {
		endpoint = current
	}

	publicEndpoint, ok := endpoint.(integrationsdk.PublicEndpointDetails)
	if !ok {
		return
	}

	normalized, err := integrationInstanceSpecNetworkEndpointFromPublicEndpoint(publicEndpoint)
	if err != nil {
		return
	}
	normalized.JsonData = resource.Spec.NetworkEndpointDetails.JsonData
	resource.Spec.NetworkEndpointDetails = normalized
}

func integrationInstanceSpecNetworkEndpointFromPublicEndpoint(
	endpoint integrationsdk.PublicEndpointDetails,
) (integrationv1beta1.IntegrationInstanceNetworkEndpointDetails, error) {
	payload, err := json.Marshal(endpoint)
	if err != nil {
		return integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{}, err
	}
	var spec integrationv1beta1.IntegrationInstanceNetworkEndpointDetails
	if err := json.Unmarshal(payload, &spec); err != nil {
		return integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{}, err
	}
	return spec, nil
}

func integrationInstanceCurrentNetworkEndpointDetails(currentResponse any) (integrationsdk.NetworkEndpointDetails, error) {
	current, err := integrationInstanceRuntimeBody(currentResponse)
	if err == nil {
		return current.NetworkEndpointDetails, nil
	}

	switch current := currentResponse.(type) {
	case integrationv1beta1.IntegrationInstanceStatus:
		return integrationInstanceNetworkEndpointDetails(current.NetworkEndpointDetails)
	case *integrationv1beta1.IntegrationInstanceStatus:
		if current == nil {
			return nil, fmt.Errorf("current IntegrationInstance status is nil")
		}
		return integrationInstanceNetworkEndpointDetails(current.NetworkEndpointDetails)
	default:
		return nil, fmt.Errorf("current IntegrationInstance response does not expose networkEndpointDetails: %T", currentResponse)
	}
}

func integrationInstanceNetworkEndpointMatchesObserved(
	desired integrationsdk.NetworkEndpointDetails,
	current integrationsdk.NetworkEndpointDetails,
) (bool, error) {
	desiredValue, desiredOK, err := integrationInstanceComparableNetworkEndpoint(desired)
	if err != nil {
		return false, fmt.Errorf("normalize desired networkEndpointDetails: %w", err)
	}
	currentValue, currentOK, err := integrationInstanceComparableNetworkEndpoint(current)
	if err != nil {
		return false, fmt.Errorf("normalize observed networkEndpointDetails: %w", err)
	}
	if !desiredOK || !currentOK {
		return desiredOK == currentOK, nil
	}
	return integrationInstanceJSONEqual(desiredValue, currentValue), nil
}

func integrationInstanceComparableNetworkEndpoint(details integrationsdk.NetworkEndpointDetails) (any, bool, error) {
	if details == nil {
		return nil, false, nil
	}

	payload, err := json.Marshal(details)
	if err != nil {
		return nil, false, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false, err
	}
	pruned, ok := integrationInstancePruneComparableValue(decoded)
	return pruned, ok, nil
}

func normalizeIntegrationInstanceStatusNetworkEndpoint(resource *integrationv1beta1.IntegrationInstance) {
	if resource == nil {
		return
	}
	endpoint, err := integrationInstanceNetworkEndpointDetails(resource.Status.NetworkEndpointDetails)
	if err != nil {
		return
	}
	_, ok, err := integrationInstanceComparableNetworkEndpoint(endpoint)
	if err != nil || ok {
		return
	}
	resource.Status.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{}
}

func integrationInstancePruneComparableValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case map[string]any:
		return integrationInstancePruneComparableMap(concrete)
	case []any:
		return integrationInstancePruneComparableSlice(concrete)
	default:
		if !integrationInstanceMeaningfulComparableValue(concrete) {
			return nil, false
		}
		return concrete, true
	}
}

func integrationInstancePruneComparableMap(value map[string]any) (any, bool) {
	pruned := make(map[string]any, len(value))
	for key, child := range value {
		if integrationInstanceDefaultNetworkEndpointDiscriminator(key, child) {
			continue
		}
		prunedChild, ok := integrationInstancePruneComparableValue(child)
		if !ok {
			continue
		}
		pruned[key] = prunedChild
	}
	if len(pruned) == 0 {
		return nil, false
	}
	return pruned, true
}

func integrationInstancePruneComparableSlice(value []any) (any, bool) {
	pruned := make([]any, 0, len(value))
	for _, child := range value {
		prunedChild, ok := integrationInstancePruneComparableValue(child)
		if !ok {
			continue
		}
		pruned = append(pruned, prunedChild)
	}
	if len(pruned) == 0 {
		return nil, false
	}
	return pruned, true
}

func integrationInstanceMeaningfulComparableValue(value any) bool {
	if value == nil {
		return false
	}
	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case bool:
		return concrete
	case float64:
		return concrete != 0
	default:
		return true
	}
}

func integrationInstanceDefaultNetworkEndpointDiscriminator(key string, value any) bool {
	if key != "networkEndpointType" {
		return false
	}
	endpointType, ok := value.(string)
	return ok && strings.EqualFold(strings.TrimSpace(endpointType), string(integrationsdk.NetworkEndpointTypePublic))
}

func integrationInstanceCustomEndpointMeaningful(spec integrationv1beta1.IntegrationInstanceCustomEndpoint) bool {
	return strings.TrimSpace(spec.Hostname) != "" || strings.TrimSpace(spec.CertificateSecretId) != ""
}

func integrationInstanceNetworkEndpointMeaningful(spec integrationv1beta1.IntegrationInstanceNetworkEndpointDetails) bool {
	return strings.TrimSpace(spec.JsonData) != "" ||
		strings.TrimSpace(spec.NetworkEndpointType) != "" ||
		len(spec.AllowlistedHttpIps) > 0 ||
		len(spec.AllowlistedHttpVcns) > 0 ||
		len(spec.Runtime.AllowlistedHttpIps) > 0 ||
		len(spec.Runtime.AllowlistedHttpVcns) > 0 ||
		len(spec.DesignTime.AllowlistedHttpIps) > 0 ||
		len(spec.DesignTime.AllowlistedHttpVcns) > 0 ||
		spec.IsIntegrationVcnAllowlisted
}

func integrationInstanceCustomEndpointMatchesObserved(
	spec integrationv1beta1.IntegrationInstanceCustomEndpoint,
	current integrationsdk.CustomEndpointDetails,
) bool {
	return spec.Hostname == integrationInstanceStringValue(current.Hostname) &&
		spec.CertificateSecretId == integrationInstanceStringValue(current.CertificateSecretId)
}

func integrationInstanceAlternateCustomEndpointsMatchObserved(
	spec []integrationv1beta1.IntegrationInstanceAlternateCustomEndpoint,
	current []integrationsdk.CustomEndpointDetails,
) bool {
	if len(spec) != len(current) {
		return false
	}
	for index := range spec {
		if spec[index].Hostname != integrationInstanceStringValue(current[index].Hostname) ||
			spec[index].CertificateSecretId != integrationInstanceStringValue(current[index].CertificateSecretId) {
			return false
		}
	}
	return true
}

func integrationInstanceMapValueMap(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func integrationInstanceCloneSDKMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}

	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		cloned[namespace] = maps.Clone(values)
	}
	return cloned
}

func integrationInstanceStringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func integrationInstanceStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func integrationInstanceOptionalBoolPtr(value bool) *bool {
	if !value {
		return nil
	}
	return common.Bool(value)
}

func integrationInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftPayload) == string(rightPayload)
}
