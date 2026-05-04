/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operationsinsightswarehouse

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
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

type operationsInsightsWarehouseOCIClient interface {
	CreateOperationsInsightsWarehouse(context.Context, opsisdk.CreateOperationsInsightsWarehouseRequest) (opsisdk.CreateOperationsInsightsWarehouseResponse, error)
	GetOperationsInsightsWarehouse(context.Context, opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error)
	ListOperationsInsightsWarehouses(context.Context, opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error)
	UpdateOperationsInsightsWarehouse(context.Context, opsisdk.UpdateOperationsInsightsWarehouseRequest) (opsisdk.UpdateOperationsInsightsWarehouseResponse, error)
	DeleteOperationsInsightsWarehouse(context.Context, opsisdk.DeleteOperationsInsightsWarehouseRequest) (opsisdk.DeleteOperationsInsightsWarehouseResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

var operationsInsightsWarehouseWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opsisdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(opsisdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(opsisdk.ActionTypeDeleted)},
}

func init() {
	registerOperationsInsightsWarehouseRuntimeHooksMutator(func(manager *OperationsInsightsWarehouseServiceManager, hooks *OperationsInsightsWarehouseRuntimeHooks) {
		client, initErr := newOperationsInsightsWarehouseOperationsInsightsClient(manager)
		applyOperationsInsightsWarehouseRuntimeHooks(hooks, client, initErr, operationsInsightsWarehouseLog(manager))
	})
}

func newOperationsInsightsWarehouseOperationsInsightsClient(manager *OperationsInsightsWarehouseServiceManager) (operationsInsightsWarehouseOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("operationsinsightswarehouse service manager is nil")
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func operationsInsightsWarehouseLog(manager *OperationsInsightsWarehouseServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func applyOperationsInsightsWarehouseRuntimeHooks(
	hooks *OperationsInsightsWarehouseRuntimeHooks,
	client operationsInsightsWarehouseOCIClient,
	initErr error,
	log loggerutil.OSOKLogger,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = operationsInsightsWarehouseRuntimeSemantics()
	hooks.BuildCreateBody = buildOperationsInsightsWarehouseCreateBody
	hooks.BuildUpdateBody = buildOperationsInsightsWarehouseUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardOperationsInsightsWarehouseExistingBeforeCreate
	hooks.List.Fields = operationsInsightsWarehouseListFields()
	wrapOperationsInsightsWarehouseListCalls(hooks)
	installOperationsInsightsWarehouseProjectedReadOperations(hooks)
	hooks.StatusHooks.ProjectStatus = projectOperationsInsightsWarehouseStatus
	hooks.DeleteHooks.HandleError = handleOperationsInsightsWarehouseDeleteError
	hooks.Async.Adapter = operationsInsightsWarehouseWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOperationsInsightsWarehouseWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveOperationsInsightsWarehouseWorkRequestAction
	hooks.Async.ResolvePhase = resolveOperationsInsightsWarehouseWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverOperationsInsightsWarehouseIDFromWorkRequest
	hooks.Async.Message = operationsInsightsWarehouseWorkRequestMessage
	wrapOperationsInsightsWarehouseDeleteGuardClient(hooks, log)
}

func newOperationsInsightsWarehouseServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client operationsInsightsWarehouseOCIClient,
) OperationsInsightsWarehouseServiceClient {
	hooks := newOperationsInsightsWarehouseRuntimeHooksWithOCIClient(client)
	applyOperationsInsightsWarehouseRuntimeHooks(&hooks, client, nil, log)
	delegate := defaultOperationsInsightsWarehouseServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.OperationsInsightsWarehouse](
			buildOperationsInsightsWarehouseGeneratedRuntimeConfig(&OperationsInsightsWarehouseServiceManager{Log: log}, hooks),
		),
	}
	return wrapOperationsInsightsWarehouseGeneratedClient(hooks, delegate)
}

func newOperationsInsightsWarehouseRuntimeHooksWithOCIClient(client operationsInsightsWarehouseOCIClient) OperationsInsightsWarehouseRuntimeHooks {
	return OperationsInsightsWarehouseRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.OperationsInsightsWarehouse]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.OperationsInsightsWarehouse]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.OperationsInsightsWarehouse]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.OperationsInsightsWarehouse]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.OperationsInsightsWarehouse]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.OperationsInsightsWarehouse]{},
		Create: runtimeOperationHooks[opsisdk.CreateOperationsInsightsWarehouseRequest, opsisdk.CreateOperationsInsightsWarehouseResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOperationsInsightsWarehouseDetails", RequestName: "CreateOperationsInsightsWarehouseDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.CreateOperationsInsightsWarehouseRequest) (opsisdk.CreateOperationsInsightsWarehouseResponse, error) {
				return client.CreateOperationsInsightsWarehouse(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetOperationsInsightsWarehouseRequest, opsisdk.GetOperationsInsightsWarehouseResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsWarehouseId", RequestName: "operationsInsightsWarehouseId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
				return client.GetOperationsInsightsWarehouse(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListOperationsInsightsWarehousesRequest, opsisdk.ListOperationsInsightsWarehousesResponse]{
			Fields: operationsInsightsWarehouseListFields(),
			Call: func(ctx context.Context, request opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error) {
				return client.ListOperationsInsightsWarehouses(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateOperationsInsightsWarehouseRequest, opsisdk.UpdateOperationsInsightsWarehouseResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "OperationsInsightsWarehouseId", RequestName: "operationsInsightsWarehouseId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateOperationsInsightsWarehouseDetails", RequestName: "UpdateOperationsInsightsWarehouseDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request opsisdk.UpdateOperationsInsightsWarehouseRequest) (opsisdk.UpdateOperationsInsightsWarehouseResponse, error) {
				return client.UpdateOperationsInsightsWarehouse(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteOperationsInsightsWarehouseRequest, opsisdk.DeleteOperationsInsightsWarehouseResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsWarehouseId", RequestName: "operationsInsightsWarehouseId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteOperationsInsightsWarehouseRequest) (opsisdk.DeleteOperationsInsightsWarehouseResponse, error) {
				return client.DeleteOperationsInsightsWarehouse(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OperationsInsightsWarehouseServiceClient) OperationsInsightsWarehouseServiceClient{},
	}
}

func operationsInsightsWarehouseRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "opsi",
		FormalSlug:    "operationsinsightswarehouse",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(opsisdk.OperationsInsightsWarehouseLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.OperationsInsightsWarehouseLifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.OperationsInsightsWarehouseLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.OperationsInsightsWarehouseLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.OperationsInsightsWarehouseLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "cpuAllocated", "computeModel", "storageAllocatedInGBs", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "cpuAllocated", "computeModel", "storageAllocatedInGBs", "freeformTags", "definedTags"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "operationsInsightsWarehouse", Action: string(opsisdk.OperationTypeCreateOpsiWarehouse)},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "operationsInsightsWarehouse", Action: string(opsisdk.OperationTypeUpdateOpsiWarehouse)},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "operationsInsightsWarehouse", Action: string(opsisdk.OperationTypeDeleteOpsiWarehouse)},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOperationsInsightsWarehouse",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "operationsInsightsWarehouse", Action: string(opsisdk.OperationTypeCreateOpsiWarehouse)},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOperationsInsightsWarehouse",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "operationsInsightsWarehouse", Action: string(opsisdk.OperationTypeUpdateOpsiWarehouse)},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "operationsInsightsWarehouse", Action: string(opsisdk.OperationTypeDeleteOpsiWarehouse)},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func operationsInsightsWarehouseListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true, LookupPaths: []string{"status.id", "status.ocid"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func guardOperationsInsightsWarehouseExistingBeforeCreate(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("operationsinsightswarehouse resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildOperationsInsightsWarehouseCreateBody(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("operationsinsightswarehouse resource is nil")
	}
	if err := validateOperationsInsightsWarehouseSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := opsisdk.CreateOperationsInsightsWarehouseDetails{
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(spec.DisplayName)),
		CpuAllocated:  common.Float64(spec.CpuAllocated),
	}
	if computeModel, ok, err := operationsInsightsWarehouseComputeModel(spec.ComputeModel); err != nil {
		return nil, err
	} else if ok {
		body.ComputeModel = computeModel
	}
	if spec.StorageAllocatedInGBs > 0 {
		body.StorageAllocatedInGBs = common.Float64(spec.StorageAllocatedInGBs)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneOperationsInsightsWarehouseStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = operationsInsightsWarehouseDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildOperationsInsightsWarehouseUpdateBody(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return opsisdk.UpdateOperationsInsightsWarehouseDetails{}, false, fmt.Errorf("operationsinsightswarehouse resource is nil")
	}
	if err := validateOperationsInsightsWarehouseSpec(resource.Spec); err != nil {
		return opsisdk.UpdateOperationsInsightsWarehouseDetails{}, false, err
	}

	current, ok := operationsInsightsWarehouseFromResponse(currentResponse)
	if !ok {
		return opsisdk.UpdateOperationsInsightsWarehouseDetails{}, false, fmt.Errorf("current OperationsInsightsWarehouse response does not expose an OperationsInsightsWarehouse body")
	}

	builder := operationsInsightsWarehouseUpdateBuilder{}
	if err := builder.apply(resource.Spec, current); err != nil {
		return opsisdk.UpdateOperationsInsightsWarehouseDetails{}, false, err
	}
	if !builder.changed {
		return opsisdk.UpdateOperationsInsightsWarehouseDetails{}, false, nil
	}
	return builder.details, true, nil
}

type operationsInsightsWarehouseUpdateBuilder struct {
	details opsisdk.UpdateOperationsInsightsWarehouseDetails
	changed bool
}

func (b *operationsInsightsWarehouseUpdateBuilder) apply(
	spec opsiv1beta1.OperationsInsightsWarehouseSpec,
	current opsisdk.OperationsInsightsWarehouse,
) error {
	b.setDisplayName(spec.DisplayName, current.DisplayName)
	b.setCpuAllocated(spec.CpuAllocated, current.CpuAllocated)
	if err := b.setComputeModel(spec.ComputeModel, current.ComputeModel); err != nil {
		return err
	}
	b.setStorageAllocated(spec.StorageAllocatedInGBs, current.StorageAllocatedInGBs)
	b.setFreeformTags(spec.FreeformTags, current.FreeformTags)
	b.setDefinedTags(spec.DefinedTags, current.DefinedTags)
	return nil
}

func (b *operationsInsightsWarehouseUpdateBuilder) setDisplayName(spec string, current *string) {
	if displayName, ok := desiredOperationsInsightsWarehouseStringForUpdate(spec, current); ok {
		b.details.DisplayName = displayName
		b.changed = true
	}
}

func (b *operationsInsightsWarehouseUpdateBuilder) setCpuAllocated(spec float64, current *float64) {
	if cpuAllocated, ok := desiredOperationsInsightsWarehouseFloatForUpdate(spec, current, true); ok {
		b.details.CpuAllocated = cpuAllocated
		b.changed = true
	}
}

func (b *operationsInsightsWarehouseUpdateBuilder) setComputeModel(
	spec string,
	current opsisdk.OperationsInsightsWarehouseComputeModelEnum,
) error {
	computeModel, ok, err := desiredOperationsInsightsWarehouseComputeModelForUpdate(spec, current)
	if err != nil || !ok {
		return err
	}
	b.details.ComputeModel = computeModel
	b.changed = true
	return nil
}

func (b *operationsInsightsWarehouseUpdateBuilder) setStorageAllocated(spec float64, current *float64) {
	if storageAllocated, ok := desiredOperationsInsightsWarehouseFloatForUpdate(spec, current, false); ok {
		b.details.StorageAllocatedInGBs = storageAllocated
		b.changed = true
	}
}

func (b *operationsInsightsWarehouseUpdateBuilder) setFreeformTags(spec map[string]string, current map[string]string) {
	if freeformTags, ok := desiredOperationsInsightsWarehouseFreeformTagsForUpdate(spec, current); ok {
		b.details.FreeformTags = freeformTags
		b.changed = true
	}
}

func (b *operationsInsightsWarehouseUpdateBuilder) setDefinedTags(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) {
	if definedTags, ok := desiredOperationsInsightsWarehouseDefinedTagsForUpdate(spec, current); ok {
		b.details.DefinedTags = definedTags
		b.changed = true
	}
}

func validateOperationsInsightsWarehouseSpec(spec opsiv1beta1.OperationsInsightsWarehouseSpec) error {
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("operationsinsightswarehouse spec.compartmentId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		return fmt.Errorf("operationsinsightswarehouse spec.displayName is required")
	}
	if spec.CpuAllocated <= 0 {
		return fmt.Errorf("operationsinsightswarehouse spec.cpuAllocated must be greater than zero")
	}
	if spec.StorageAllocatedInGBs < 0 {
		return fmt.Errorf("operationsinsightswarehouse spec.storageAllocatedInGBs must not be negative")
	}
	if _, _, err := operationsInsightsWarehouseComputeModel(spec.ComputeModel); err != nil {
		return err
	}
	return nil
}

func operationsInsightsWarehouseComputeModel(
	value string,
) (opsisdk.OperationsInsightsWarehouseComputeModelEnum, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, nil
	}
	computeModel, ok := opsisdk.GetMappingOperationsInsightsWarehouseComputeModelEnum(value)
	if !ok {
		return "", false, fmt.Errorf("operationsinsightswarehouse spec.computeModel %q is unsupported", value)
	}
	return computeModel, true, nil
}

func wrapOperationsInsightsWarehouseListCalls(hooks *OperationsInsightsWarehouseRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error) {
		return listOperationsInsightsWarehousesAllPages(ctx, call, request)
	}
}

func installOperationsInsightsWarehouseProjectedReadOperations(hooks *OperationsInsightsWarehouseRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Get.Call != nil {
		getFields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
		hooks.Read.Get = &generatedruntime.Operation{
			NewRequest: func() any { return &opsisdk.GetOperationsInsightsWarehouseRequest{} },
			Fields:     getFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*opsisdk.GetOperationsInsightsWarehouseRequest))
				if err != nil {
					return nil, err
				}
				return operationsInsightsWarehouseProjectedResponseFromSDK(response.OperationsInsightsWarehouse, response.OpcRequestId), nil
			},
		}
	}
	if hooks.List.Call != nil {
		listFields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
		hooks.Read.List = &generatedruntime.Operation{
			NewRequest: func() any { return &opsisdk.ListOperationsInsightsWarehousesRequest{} },
			Fields:     listFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*opsisdk.ListOperationsInsightsWarehousesRequest))
				if err != nil {
					return nil, err
				}
				return operationsInsightsWarehouseProjectedListResponseFromSDK(response), nil
			},
		}
	}
}

func listOperationsInsightsWarehousesAllPages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error),
	request opsisdk.ListOperationsInsightsWarehousesRequest,
) (opsisdk.ListOperationsInsightsWarehousesResponse, error) {
	var combined opsisdk.ListOperationsInsightsWarehousesResponse
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
		combined.OpcNextPage = response.OpcNextPage
	}
}

func getOperationsInsightsWarehouseWorkRequest(
	ctx context.Context,
	client operationsInsightsWarehouseOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize OperationsInsightsWarehouse OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("operationsinsightswarehouse OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveOperationsInsightsWarehouseWorkRequestAction(workRequest any) (string, error) {
	current, err := operationsInsightsWarehouseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	var action string
	for _, resource := range current.Resources {
		if !isOperationsInsightsWarehouseWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		switch opsisdk.ActionTypeEnum(candidate) {
		case "", opsisdk.ActionTypeInProgress, opsisdk.ActionTypeRelated, opsisdk.ActionTypeFailed:
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("operationsinsightswarehouse work request %s exposes conflicting OperationsInsightsWarehouse action types %q and %q", operationsInsightsWarehouseStringValue(current.Id), action, candidate)
		}
	}
	return action, nil
}

func resolveOperationsInsightsWarehouseWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := operationsInsightsWarehouseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case opsisdk.OperationTypeCreateOpsiWarehouse:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case opsisdk.OperationTypeUpdateOpsiWarehouse:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case opsisdk.OperationTypeDeleteOpsiWarehouse:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverOperationsInsightsWarehouseIDFromWorkRequest(
	_ *opsiv1beta1.OperationsInsightsWarehouse,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := operationsInsightsWarehouseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	expectedAction := operationsInsightsWarehouseWorkRequestActionForPhase(phase)
	for _, resource := range current.Resources {
		if !isOperationsInsightsWarehouseWorkRequestResource(resource) {
			continue
		}
		if expectedAction != "" && resource.ActionType != expectedAction && resource.ActionType != opsisdk.ActionTypeInProgress {
			continue
		}
		if id := strings.TrimSpace(operationsInsightsWarehouseStringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func operationsInsightsWarehouseWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return opsisdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return opsisdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return opsisdk.ActionTypeDeleted
	default:
		return ""
	}
}

func operationsInsightsWarehouseWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := operationsInsightsWarehouseWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("OperationsInsightsWarehouse %s work request %s is %s", phase, operationsInsightsWarehouseStringValue(current.Id), current.Status)
}

func operationsInsightsWarehouseWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("operationsinsightswarehouse work request is nil")
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("operationsinsightswarehouse work request has unexpected type %T", workRequest)
	}
}

func isOperationsInsightsWarehouseWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	switch normalizeOperationsInsightsWarehouseWorkRequestToken(operationsInsightsWarehouseStringValue(resource.EntityType)) {
	case "operationsinsightswarehouse", "opsiwarehouse", "warehouse":
		return true
	default:
		return false
	}
}

func normalizeOperationsInsightsWarehouseWorkRequestToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func handleOperationsInsightsWarehouseDeleteError(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("operationsinsightswarehouse delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapOperationsInsightsWarehouseDeleteGuardClient(hooks *OperationsInsightsWarehouseRuntimeHooks, log loggerutil.OSOKLogger) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OperationsInsightsWarehouseServiceClient) OperationsInsightsWarehouseServiceClient {
		return operationsInsightsWarehouseDeleteGuardClient{
			delegate:       delegate,
			getWorkRequest: hooks.Async.GetWorkRequest,
			getWarehouse:   hooks.Get.Call,
			log:            log,
		}
	})
}

type operationsInsightsWarehouseDeleteGuardClient struct {
	delegate       OperationsInsightsWarehouseServiceClient
	getWorkRequest func(context.Context, string) (any, error)
	getWarehouse   func(context.Context, opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error)
	log            loggerutil.OSOKLogger
}

func (c operationsInsightsWarehouseDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("operationsinsightswarehouse runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c operationsInsightsWarehouseDeleteGuardClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("operationsinsightswarehouse runtime client is not configured")
	}
	if resource == nil {
		return false, fmt.Errorf("operationsinsightswarehouse resource is nil")
	}

	workRequestID, phase := currentOperationsInsightsWarehouseWorkRequest(resource)
	if workRequestID == "" {
		if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
			return false, err
		}
		return c.delegate.Delete(ctx, resource)
	}

	switch phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return c.resumeOperationsInsightsWarehouseWriteWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
	case shared.OSOKAsyncPhaseDelete:
		return c.resumeOperationsInsightsWarehouseDeleteWorkRequest(ctx, resource, workRequestID)
	default:
		return false, fmt.Errorf("operationsinsightswarehouse delete cannot resume unsupported work request phase %q", phase)
	}
}

func (c operationsInsightsWarehouseDeleteGuardClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
) error {
	if c.getWarehouse == nil {
		return nil
	}
	currentID := currentOperationsInsightsWarehouseID(resource)
	if currentID == "" {
		return nil
	}
	_, err := c.getWarehouse(ctx, opsisdk.GetOperationsInsightsWarehouseRequest{
		OperationsInsightsWarehouseId: common.String(currentID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("operationsinsightswarehouse delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func currentOperationsInsightsWarehouseWorkRequest(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || strings.TrimSpace(current.WorkRequestID) == "" {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func (c operationsInsightsWarehouseDeleteGuardClient) resumeOperationsInsightsWarehouseWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	workRequest, current, err := c.pollOperationsInsightsWarehouseWorkRequest(ctx, resource, workRequestID, phase)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markOperationsInsightsWarehouseWorkRequestOperation(resource, current, operationsInsightsWarehouseWorkRequestMessage(current.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		resourceID, err := recoverOperationsInsightsWarehouseIDFromWorkRequest(resource, workRequest, current.Phase)
		if err != nil {
			c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
			return false, err
		}
		if strings.TrimSpace(resourceID) == "" {
			resourceID = currentOperationsInsightsWarehouseID(resource)
		}
		if strings.TrimSpace(resourceID) == "" {
			err := fmt.Errorf("operationsinsightswarehouse %s work request %s did not expose an OperationsInsightsWarehouse identifier before delete", current.Phase, workRequestID)
			c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
			return false, err
		}
		recordOperationsInsightsWarehouseID(resource, resourceID)
		servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
		return c.delegate.Delete(ctx, resource)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("operationsinsightswarehouse %s work request %s finished with status %s before delete", current.Phase, workRequestID, current.RawStatus)
		c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
		return false, err
	default:
		err := fmt.Errorf("operationsinsightswarehouse %s work request %s projected unsupported async class %s before delete", current.Phase, workRequestID, current.NormalizedClass)
		c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
		return false, err
	}
}

func (c operationsInsightsWarehouseDeleteGuardClient) resumeOperationsInsightsWarehouseDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	workRequestID string,
) (bool, error) {
	workRequest, current, err := c.pollOperationsInsightsWarehouseWorkRequest(ctx, resource, workRequestID, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markOperationsInsightsWarehouseWorkRequestOperation(resource, current, operationsInsightsWarehouseWorkRequestMessage(current.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.confirmSucceededOperationsInsightsWarehouseDeleteWorkRequest(ctx, resource, workRequest, current)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("operationsinsightswarehouse delete work request %s finished with status %s", workRequestID, current.RawStatus)
		c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
		return false, err
	default:
		err := fmt.Errorf("operationsinsightswarehouse delete work request %s projected unsupported async class %s", workRequestID, current.NormalizedClass)
		c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
		return false, err
	}
}

func (c operationsInsightsWarehouseDeleteGuardClient) pollOperationsInsightsWarehouseWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (any, *shared.OSOKAsyncOperation, error) {
	if c.getWorkRequest == nil {
		return nil, nil, fmt.Errorf("operationsinsightswarehouse work request polling is not configured")
	}
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return nil, nil, err
	}
	current, err := buildOperationsInsightsWarehouseAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return nil, nil, err
	}
	return workRequest, current, nil
}

func buildOperationsInsightsWarehouseAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	rawAction, err := resolveOperationsInsightsWarehouseWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	if derivedPhase, ok, err := resolveOperationsInsightsWarehouseWorkRequestPhase(workRequest); err != nil {
		return nil, err
	} else if ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf("operationsinsightswarehouse work request exposes phase %q while reconcile expected %q", derivedPhase, fallbackPhase)
		}
		fallbackPhase = derivedPhase
	}

	current, err := operationsInsightsWarehouseWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	asyncOperation, err := servicemanager.BuildWorkRequestAsyncOperation(status, operationsInsightsWarehouseWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        rawAction,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    operationsInsightsWarehouseStringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := strings.TrimSpace(operationsInsightsWarehouseWorkRequestMessage(asyncOperation.Phase, workRequest)); message != "" {
		asyncOperation.Message = message
	}
	return asyncOperation, nil
}

func (c operationsInsightsWarehouseDeleteGuardClient) confirmSucceededOperationsInsightsWarehouseDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, error) {
	currentID := currentOperationsInsightsWarehouseID(resource)
	if currentID == "" {
		recoveredID, err := recoverOperationsInsightsWarehouseIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err != nil {
			c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
			return false, err
		}
		currentID = strings.TrimSpace(recoveredID)
	}
	if currentID == "" {
		markOperationsInsightsWarehouseDeleted(resource, "OCI OperationsInsightsWarehouse delete work request completed", c.log)
		return true, nil
	}

	response, err := c.readOperationsInsightsWarehouseForDelete(ctx, currentID)
	if err != nil {
		err = handleOperationsInsightsWarehouseDeleteError(resource, err)
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markOperationsInsightsWarehouseDeleted(resource, "OCI resource deleted", c.log)
			return true, nil
		}
		c.markOperationsInsightsWarehouseWorkRequestFailure(resource, current, err)
		return false, err
	}
	if err := projectOperationsInsightsWarehouseStatus(resource, response); err != nil {
		return false, err
	}

	currentWarehouse, ok := operationsInsightsWarehouseFromResponse(response)
	if !ok {
		return false, fmt.Errorf("operationsinsightswarehouse delete confirmation response %T does not expose an OperationsInsightsWarehouse body", response)
	}
	switch currentWarehouse.LifecycleState {
	case opsisdk.OperationsInsightsWarehouseLifecycleStateDeleted:
		markOperationsInsightsWarehouseDeleted(resource, "OCI resource deleted", c.log)
		return true, nil
	case "", opsisdk.OperationsInsightsWarehouseLifecycleStateDeleting:
		markOperationsInsightsWarehouseTerminating(resource, "OCI resource delete is in progress", c.log)
		return false, nil
	default:
		return false, fmt.Errorf("operationsinsightswarehouse delete confirmation returned unexpected lifecycle state %q", currentWarehouse.LifecycleState)
	}
}

func (c operationsInsightsWarehouseDeleteGuardClient) readOperationsInsightsWarehouseForDelete(
	ctx context.Context,
	resourceID string,
) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
	if c.getWarehouse == nil {
		return opsisdk.GetOperationsInsightsWarehouseResponse{}, fmt.Errorf("operationsinsightswarehouse delete requires a readable OCI operation")
	}
	return c.getWarehouse(ctx, opsisdk.GetOperationsInsightsWarehouseRequest{
		OperationsInsightsWarehouseId: common.String(strings.TrimSpace(resourceID)),
	})
}

func (c operationsInsightsWarehouseDeleteGuardClient) markOperationsInsightsWarehouseWorkRequestOperation(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	current *shared.OSOKAsyncOperation,
	message string,
) {
	if current == nil {
		return
	}
	next := *current
	next.Message = strings.TrimSpace(message)
	now := metav1.Now()
	next.UpdatedAt = &now
	if currentID := currentOperationsInsightsWarehouseID(resource); currentID != "" {
		recordOperationsInsightsWarehouseID(resource, currentID)
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func (c operationsInsightsWarehouseDeleteGuardClient) markOperationsInsightsWarehouseWorkRequestFailure(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	current *shared.OSOKAsyncOperation,
	err error,
) {
	if err == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if current == nil {
		resource.Status.OsokStatus.Message = err.Error()
		resource.Status.OsokStatus.Reason = string(shared.Failed)
		now := metav1.Now()
		resource.Status.OsokStatus.UpdatedAt = &now
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
		return
	}
	next := *current
	switch next.NormalizedClass {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		next.NormalizedClass = shared.OSOKAsyncClassFailed
	}
	next.Message = err.Error()
	now := metav1.Now()
	next.UpdatedAt = &now
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func operationsInsightsWarehouseFromResponse(response any) (opsisdk.OperationsInsightsWarehouse, bool) {
	if projection, ok := operationsInsightsWarehouseProjectionFromResponse(response); ok {
		return operationsInsightsWarehouseFromProjection(projection), true
	}
	if current, ok := operationsInsightsWarehouseFromDirectResponse(response); ok {
		return current, true
	}
	return operationsInsightsWarehouseFromWrappedResponse(response)
}

func operationsInsightsWarehouseFromDirectResponse(response any) (opsisdk.OperationsInsightsWarehouse, bool) {
	switch current := response.(type) {
	case opsisdk.OperationsInsightsWarehouse:
		return current, true
	case *opsisdk.OperationsInsightsWarehouse:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouse{}, false
		}
		return *current, true
	case opsisdk.OperationsInsightsWarehouseSummary:
		return operationsInsightsWarehouseFromSummary(current), true
	case *opsisdk.OperationsInsightsWarehouseSummary:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouse{}, false
		}
		return operationsInsightsWarehouseFromSummary(*current), true
	default:
		return opsisdk.OperationsInsightsWarehouse{}, false
	}
}

func operationsInsightsWarehouseFromWrappedResponse(response any) (opsisdk.OperationsInsightsWarehouse, bool) {
	switch current := response.(type) {
	case opsisdk.CreateOperationsInsightsWarehouseResponse:
		return current.OperationsInsightsWarehouse, true
	case *opsisdk.CreateOperationsInsightsWarehouseResponse:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouse{}, false
		}
		return current.OperationsInsightsWarehouse, true
	case opsisdk.GetOperationsInsightsWarehouseResponse:
		return current.OperationsInsightsWarehouse, true
	case *opsisdk.GetOperationsInsightsWarehouseResponse:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouse{}, false
		}
		return current.OperationsInsightsWarehouse, true
	default:
		return opsisdk.OperationsInsightsWarehouse{}, false
	}
}

func operationsInsightsWarehouseFromSummary(summary opsisdk.OperationsInsightsWarehouseSummary) opsisdk.OperationsInsightsWarehouse {
	return opsisdk.OperationsInsightsWarehouse{
		Id:                          summary.Id,
		CompartmentId:               summary.CompartmentId,
		DisplayName:                 summary.DisplayName,
		CpuAllocated:                summary.CpuAllocated,
		TimeCreated:                 summary.TimeCreated,
		LifecycleState:              summary.LifecycleState,
		ComputeModel:                summary.ComputeModel,
		CpuUsed:                     summary.CpuUsed,
		StorageAllocatedInGBs:       summary.StorageAllocatedInGBs,
		StorageUsedInGBs:            summary.StorageUsedInGBs,
		DynamicGroupId:              summary.DynamicGroupId,
		OperationsInsightsTenancyId: summary.OperationsInsightsTenancyId,
		TimeLastWalletRotated:       summary.TimeLastWalletRotated,
		FreeformTags:                summary.FreeformTags,
		DefinedTags:                 summary.DefinedTags,
		SystemTags:                  summary.SystemTags,
		TimeUpdated:                 summary.TimeUpdated,
		LifecycleDetails:            summary.LifecycleDetails,
	}
}

type operationsInsightsWarehouseStatusProjection struct {
	Id                          string                     `json:"id,omitempty"`
	CompartmentId               string                     `json:"compartmentId,omitempty"`
	DisplayName                 string                     `json:"displayName,omitempty"`
	CpuAllocated                float64                    `json:"cpuAllocated,omitempty"`
	TimeCreated                 string                     `json:"timeCreated,omitempty"`
	LifecycleState              string                     `json:"lifecycleState,omitempty"`
	ComputeModel                string                     `json:"computeModel,omitempty"`
	CpuUsed                     float64                    `json:"cpuUsed,omitempty"`
	StorageAllocatedInGBs       float64                    `json:"storageAllocatedInGBs,omitempty"`
	StorageUsedInGBs            float64                    `json:"storageUsedInGBs,omitempty"`
	DynamicGroupId              string                     `json:"dynamicGroupId,omitempty"`
	OperationsInsightsTenancyId string                     `json:"operationsInsightsTenancyId,omitempty"`
	TimeLastWalletRotated       string                     `json:"timeLastWalletRotated,omitempty"`
	FreeformTags                map[string]string          `json:"freeformTags,omitempty"`
	DefinedTags                 map[string]shared.MapValue `json:"definedTags,omitempty"`
	SystemTags                  map[string]shared.MapValue `json:"systemTags,omitempty"`
	TimeUpdated                 string                     `json:"timeUpdated,omitempty"`
	LifecycleDetails            string                     `json:"lifecycleDetails,omitempty"`
}

type operationsInsightsWarehouseProjectedResponse struct {
	OperationsInsightsWarehouse operationsInsightsWarehouseStatusProjection `presentIn:"body"`
	OpcRequestId                *string                                     `presentIn:"header" name:"opc-request-id"`
}

type operationsInsightsWarehouseProjectedCollection struct {
	Items []operationsInsightsWarehouseStatusProjection `json:"items,omitempty"`
}

type operationsInsightsWarehouseProjectedListResponse struct {
	OperationsInsightsWarehouseCollection operationsInsightsWarehouseProjectedCollection `presentIn:"body"`
	OpcRequestId                          *string                                        `presentIn:"header" name:"opc-request-id"`
	OpcNextPage                           *string                                        `presentIn:"header" name:"opc-next-page"`
}

func operationsInsightsWarehouseProjectedResponseFromSDK(
	current opsisdk.OperationsInsightsWarehouse,
	opcRequestID *string,
) operationsInsightsWarehouseProjectedResponse {
	return operationsInsightsWarehouseProjectedResponse{
		OperationsInsightsWarehouse: operationsInsightsWarehouseProjectionFromSDK(current),
		OpcRequestId:                opcRequestID,
	}
}

func operationsInsightsWarehouseProjectedListResponseFromSDK(
	response opsisdk.ListOperationsInsightsWarehousesResponse,
) operationsInsightsWarehouseProjectedListResponse {
	projected := operationsInsightsWarehouseProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.OperationsInsightsWarehouseCollection.Items = append(
			projected.OperationsInsightsWarehouseCollection.Items,
			operationsInsightsWarehouseProjectionFromSDK(operationsInsightsWarehouseFromSummary(item)),
		)
	}
	return projected
}

func operationsInsightsWarehouseProjectionFromSDK(
	current opsisdk.OperationsInsightsWarehouse,
) operationsInsightsWarehouseStatusProjection {
	return operationsInsightsWarehouseStatusProjection{
		Id:                          operationsInsightsWarehouseStringValue(current.Id),
		CompartmentId:               operationsInsightsWarehouseStringValue(current.CompartmentId),
		DisplayName:                 operationsInsightsWarehouseStringValue(current.DisplayName),
		CpuAllocated:                operationsInsightsWarehouseFloatValue(current.CpuAllocated),
		TimeCreated:                 operationsInsightsWarehouseSDKTimeString(current.TimeCreated),
		LifecycleState:              string(current.LifecycleState),
		ComputeModel:                string(current.ComputeModel),
		CpuUsed:                     operationsInsightsWarehouseFloatValue(current.CpuUsed),
		StorageAllocatedInGBs:       operationsInsightsWarehouseFloatValue(current.StorageAllocatedInGBs),
		StorageUsedInGBs:            operationsInsightsWarehouseFloatValue(current.StorageUsedInGBs),
		DynamicGroupId:              operationsInsightsWarehouseStringValue(current.DynamicGroupId),
		OperationsInsightsTenancyId: operationsInsightsWarehouseStringValue(current.OperationsInsightsTenancyId),
		TimeLastWalletRotated:       operationsInsightsWarehouseSDKTimeString(current.TimeLastWalletRotated),
		FreeformTags:                cloneOperationsInsightsWarehouseStringMap(current.FreeformTags),
		DefinedTags:                 operationsInsightsWarehouseStatusDefinedTags(current.DefinedTags),
		SystemTags:                  operationsInsightsWarehouseStatusDefinedTags(current.SystemTags),
		TimeUpdated:                 operationsInsightsWarehouseSDKTimeString(current.TimeUpdated),
		LifecycleDetails:            operationsInsightsWarehouseStringValue(current.LifecycleDetails),
	}
}

func projectOperationsInsightsWarehouseStatus(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("operationsinsightswarehouse resource is nil")
	}
	projected, ok := operationsInsightsWarehouseProjectionFromResponse(response)
	if !ok {
		return nil
	}
	resource.Status = operationsInsightsWarehouseStatusFromProjection(resource.Status.OsokStatus, projected)
	if projected.Id != "" {
		recordOperationsInsightsWarehouseID(resource, projected.Id)
	}
	return nil
}

func operationsInsightsWarehouseProjectionFromResponse(
	response any,
) (operationsInsightsWarehouseStatusProjection, bool) {
	switch current := response.(type) {
	case operationsInsightsWarehouseProjectedResponse:
		return current.OperationsInsightsWarehouse, true
	case *operationsInsightsWarehouseProjectedResponse:
		if current == nil {
			return operationsInsightsWarehouseStatusProjection{}, false
		}
		return current.OperationsInsightsWarehouse, true
	case operationsInsightsWarehouseStatusProjection:
		return current, true
	case *operationsInsightsWarehouseStatusProjection:
		if current == nil {
			return operationsInsightsWarehouseStatusProjection{}, false
		}
		return *current, true
	default:
		if warehouse, ok := operationsInsightsWarehouseFromDirectResponse(response); ok {
			return operationsInsightsWarehouseProjectionFromSDK(warehouse), true
		}
		if warehouse, ok := operationsInsightsWarehouseFromWrappedResponse(response); ok {
			return operationsInsightsWarehouseProjectionFromSDK(warehouse), true
		}
		return operationsInsightsWarehouseStatusProjection{}, false
	}
}

func operationsInsightsWarehouseStatusFromProjection(
	osokStatus shared.OSOKStatus,
	projected operationsInsightsWarehouseStatusProjection,
) opsiv1beta1.OperationsInsightsWarehouseStatus {
	return opsiv1beta1.OperationsInsightsWarehouseStatus{
		OsokStatus:                  osokStatus,
		Id:                          projected.Id,
		CompartmentId:               projected.CompartmentId,
		DisplayName:                 projected.DisplayName,
		CpuAllocated:                projected.CpuAllocated,
		TimeCreated:                 projected.TimeCreated,
		LifecycleState:              projected.LifecycleState,
		ComputeModel:                projected.ComputeModel,
		CpuUsed:                     projected.CpuUsed,
		StorageAllocatedInGBs:       projected.StorageAllocatedInGBs,
		StorageUsedInGBs:            projected.StorageUsedInGBs,
		DynamicGroupId:              projected.DynamicGroupId,
		OperationsInsightsTenancyId: projected.OperationsInsightsTenancyId,
		TimeLastWalletRotated:       projected.TimeLastWalletRotated,
		FreeformTags:                cloneOperationsInsightsWarehouseStringMap(projected.FreeformTags),
		DefinedTags:                 cloneOperationsInsightsWarehouseStatusDefinedTags(projected.DefinedTags),
		SystemTags:                  cloneOperationsInsightsWarehouseStatusDefinedTags(projected.SystemTags),
		TimeUpdated:                 projected.TimeUpdated,
		LifecycleDetails:            projected.LifecycleDetails,
	}
}

func operationsInsightsWarehouseFromProjection(
	projected operationsInsightsWarehouseStatusProjection,
) opsisdk.OperationsInsightsWarehouse {
	return opsisdk.OperationsInsightsWarehouse{
		Id:                          operationsInsightsWarehouseStringPtr(projected.Id),
		CompartmentId:               operationsInsightsWarehouseStringPtr(projected.CompartmentId),
		DisplayName:                 operationsInsightsWarehouseStringPtr(projected.DisplayName),
		CpuAllocated:                operationsInsightsWarehouseFloatPtr(projected.CpuAllocated),
		LifecycleState:              opsisdk.OperationsInsightsWarehouseLifecycleStateEnum(projected.LifecycleState),
		ComputeModel:                opsisdk.OperationsInsightsWarehouseComputeModelEnum(projected.ComputeModel),
		CpuUsed:                     operationsInsightsWarehouseFloatPtr(projected.CpuUsed),
		StorageAllocatedInGBs:       operationsInsightsWarehouseFloatPtr(projected.StorageAllocatedInGBs),
		StorageUsedInGBs:            operationsInsightsWarehouseFloatPtr(projected.StorageUsedInGBs),
		DynamicGroupId:              operationsInsightsWarehouseStringPtr(projected.DynamicGroupId),
		OperationsInsightsTenancyId: operationsInsightsWarehouseStringPtr(projected.OperationsInsightsTenancyId),
		TimeLastWalletRotated:       nil,
		FreeformTags:                cloneOperationsInsightsWarehouseStringMap(projected.FreeformTags),
		DefinedTags:                 operationsInsightsWarehouseDefinedTagsFromStatus(projected.DefinedTags),
		SystemTags:                  operationsInsightsWarehouseDefinedTagsFromStatus(projected.SystemTags),
		TimeUpdated:                 nil,
		LifecycleDetails:            operationsInsightsWarehouseStringPtr(projected.LifecycleDetails),
	}
}

func markOperationsInsightsWarehouseDeleted(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	message string,
	log loggerutil.OSOKLogger,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, log)
}

func markOperationsInsightsWarehouseTerminating(
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	message string,
	log loggerutil.OSOKLogger,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
		UpdatedAt:       &now,
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, log)
}

func currentOperationsInsightsWarehouseID(resource *opsiv1beta1.OperationsInsightsWarehouse) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func recordOperationsInsightsWarehouseID(resource *opsiv1beta1.OperationsInsightsWarehouse, id string) {
	if resource == nil {
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	if resource.Status.OsokStatus.CreatedAt == nil {
		now := metav1.Now()
		resource.Status.OsokStatus.CreatedAt = &now
	}
}

func desiredOperationsInsightsWarehouseStringForUpdate(spec string, current *string) (*string, bool) {
	trimmedSpec := strings.TrimSpace(spec)
	if trimmedSpec == "" || trimmedSpec == strings.TrimSpace(operationsInsightsWarehouseStringValue(current)) {
		return nil, false
	}
	return common.String(trimmedSpec), true
}

func desiredOperationsInsightsWarehouseFloatForUpdate(spec float64, current *float64, required bool) (*float64, bool) {
	if !required && spec == 0 {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Float64(spec), true
}

func desiredOperationsInsightsWarehouseComputeModelForUpdate(
	spec string,
	current opsisdk.OperationsInsightsWarehouseComputeModelEnum,
) (opsisdk.OperationsInsightsWarehouseComputeModelEnum, bool, error) {
	computeModel, ok, err := operationsInsightsWarehouseComputeModel(spec)
	if err != nil || !ok {
		return "", false, err
	}
	if computeModel == current {
		return "", false, nil
	}
	return computeModel, true, nil
}

func desiredOperationsInsightsWarehouseFreeformTagsForUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	desired := cloneOperationsInsightsWarehouseStringMap(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func desiredOperationsInsightsWarehouseDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := operationsInsightsWarehouseDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func operationsInsightsWarehouseDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func operationsInsightsWarehouseDefinedTagsFromStatus(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func operationsInsightsWarehouseStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func cloneOperationsInsightsWarehouseStatusDefinedTags(input map[string]shared.MapValue) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		cloned[namespace] = tagValues
	}
	return cloned
}

func cloneOperationsInsightsWarehouseStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func operationsInsightsWarehouseStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func operationsInsightsWarehouseStringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func operationsInsightsWarehouseFloatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func operationsInsightsWarehouseFloatPtr(value float64) *float64 {
	if value == 0 {
		return nil
	}
	return common.Float64(value)
}

func operationsInsightsWarehouseSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}
