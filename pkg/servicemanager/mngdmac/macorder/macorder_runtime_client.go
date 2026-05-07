/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package macorder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	mngdmacsdk "github.com/oracle/oci-go-sdk/v65/mngdmac"
	mngdmacv1beta1 "github.com/oracle/oci-service-operator/api/mngdmac/v1beta1"
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

const macOrderKind = "MacOrder"

type macOrderOCIClient interface {
	CreateMacOrder(context.Context, mngdmacsdk.CreateMacOrderRequest) (mngdmacsdk.CreateMacOrderResponse, error)
	GetMacOrder(context.Context, mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error)
	ListMacOrders(context.Context, mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error)
	UpdateMacOrder(context.Context, mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error)
	CancelMacOrder(context.Context, mngdmacsdk.CancelMacOrderRequest) (mngdmacsdk.CancelMacOrderResponse, error)
	GetWorkRequest(context.Context, mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error)
}

type macOrderListCall func(context.Context, mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error)

var macOrderWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(mngdmacsdk.OperationStatusAccepted),
		string(mngdmacsdk.OperationStatusInProgress),
		string(mngdmacsdk.OperationStatusWaiting),
		string(mngdmacsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(mngdmacsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(mngdmacsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(mngdmacsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(mngdmacsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(mngdmacsdk.OperationTypeCreateMacOrder)},
	UpdateActionTokens:    []string{string(mngdmacsdk.OperationTypeUpdateMacOrder)},
	DeleteActionTokens: []string{
		string(mngdmacsdk.OperationTypeCancelMacOrder),
		string(mngdmacsdk.OperationTypeDeleteMacOrder),
	},
}

type macOrderRuntimeClient struct {
	delegate MacOrderServiceClient
	client   macOrderOCIClient
	initErr  error
}

func init() {
	newMacOrderServiceClient = newMacOrderRuntimeServiceClient
}

func newMacOrderRuntimeServiceClient(manager *MacOrderServiceManager) MacOrderServiceClient {
	var (
		sdkClient mngdmacsdk.MacOrderClient
		err       error
	)
	if manager == nil {
		err = fmt.Errorf("%s service manager is nil", macOrderKind)
		manager = &MacOrderServiceManager{}
	} else {
		sdkClient, err = mngdmacsdk.NewMacOrderClientWithConfigurationProvider(manager.Provider)
	}

	hooks := newMacOrderRuntimeHooksWithOCIClient(sdkClient)
	applyMacOrderRuntimeHooks(&hooks, sdkClient, err)

	config := buildMacOrderGeneratedRuntimeConfig(manager, hooks)
	configureMacOrderDeleteOperation(&config, sdkClient)
	if err != nil {
		config.InitError = fmt.Errorf("initialize %s OCI client: %w", macOrderKind, err)
	}

	delegate := defaultMacOrderServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*mngdmacv1beta1.MacOrder](config),
	}
	return &macOrderRuntimeClient{
		delegate: wrapMacOrderGeneratedClient(hooks, delegate),
		client:   sdkClient,
		initErr:  err,
	}
}

func newMacOrderServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client macOrderOCIClient,
) MacOrderServiceClient {
	manager := &MacOrderServiceManager{Log: log}
	hooks := newMacOrderRuntimeHooksWithOCIClient(client)
	applyMacOrderRuntimeHooks(&hooks, client, nil)

	config := buildMacOrderGeneratedRuntimeConfig(manager, hooks)
	configureMacOrderDeleteOperation(&config, client)

	delegate := defaultMacOrderServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*mngdmacv1beta1.MacOrder](config),
	}
	return &macOrderRuntimeClient{
		delegate: wrapMacOrderGeneratedClient(hooks, delegate),
		client:   client,
	}
}

func newMacOrderRuntimeHooksWithOCIClient(client macOrderOCIClient) MacOrderRuntimeHooks {
	hooks := newMacOrderDefaultRuntimeHooks(mngdmacsdk.MacOrderClient{})
	hooks.Create.Call = func(ctx context.Context, request mngdmacsdk.CreateMacOrderRequest) (mngdmacsdk.CreateMacOrderResponse, error) {
		return client.CreateMacOrder(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
		return client.GetMacOrder(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error) {
		return client.ListMacOrders(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error) {
		return client.UpdateMacOrder(ctx, request)
	}
	return hooks
}

func configureMacOrderDeleteOperation(
	config *generatedruntime.Config[*mngdmacv1beta1.MacOrder],
	client macOrderOCIClient,
) {
	if config == nil {
		return
	}
	config.Delete = &generatedruntime.Operation{
		NewRequest: func() any { return &mngdmacsdk.CancelMacOrderRequest{} },
		Fields:     macOrderDeleteFields(),
		Call: func(ctx context.Context, request any) (any, error) {
			return client.CancelMacOrder(ctx, *request.(*mngdmacsdk.CancelMacOrderRequest))
		},
	}
}

func (c *macOrderRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *mngdmacv1beta1.MacOrder,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s generated runtime delegate is not configured", macOrderKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *macOrderRuntimeClient) Delete(ctx context.Context, resource *mngdmacv1beta1.MacOrder) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", macOrderKind)
	}
	if c == nil {
		return false, fmt.Errorf("%s runtime client is nil", macOrderKind)
	}
	if c.initErr != nil {
		return false, fmt.Errorf("initialize %s OCI client: %w", macOrderKind, c.initErr)
	}
	if c.client == nil {
		return false, fmt.Errorf("%s OCI client is not configured", macOrderKind)
	}

	currentID := trackedMacOrderID(resource)
	if currentID == "" {
		response, err := confirmMacOrderDeleteRead(ctx, resource, "", c.client, nil)
		if err != nil {
			if macOrderDeleteNotFound(err) {
				markMacOrderDeleted(resource, "OCI resource deleted")
				return true, nil
			}
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, err
		}
		if err := applyMacOrderStatus(resource, response); err != nil {
			return false, err
		}
		if macOrderDeleteConfirmed(response) {
			markMacOrderDeleted(resource, "OCI MacOrder cancellation confirmed")
			return true, nil
		}
		currentID = trackedMacOrderID(resource)
	}

	if currentID == "" {
		return false, fmt.Errorf("%s delete requires a tracked id or an exact compartmentId + displayName match", macOrderKind)
	}

	response, err := confirmMacOrderDeleteRead(ctx, resource, currentID, c.client, nil)
	if err != nil {
		if macOrderDeleteNotFound(err) {
			markMacOrderDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	if err := applyMacOrderStatus(resource, response); err != nil {
		return false, err
	}
	if macOrderDeleteConfirmed(response) {
		markMacOrderDeleted(resource, "OCI MacOrder cancellation confirmed")
		return true, nil
	}
	if macOrderDeletePending(response) {
		markMacOrderDeletePending(resource, "OCI MacOrder cancellation is in progress")
		return false, nil
	}

	cancelResponse, err := c.client.CancelMacOrder(ctx, mngdmacsdk.CancelMacOrderRequest{
		MacOrderId: common.String(currentID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if macOrderDeleteNotFound(err) {
			markMacOrderDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, cancelResponse)

	workRequestID := stringValue(cancelResponse.OpcWorkRequestId)
	if workRequestID == "" {
		return false, fmt.Errorf("%s delete did not return an opc-work-request-id", macOrderKind)
	}

	workRequest, err := getMacOrderWorkRequest(ctx, c.client, nil, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	currentAsync, err := buildMacOrderDeleteAsyncOperation(resource, workRequest)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		applyMacOrderAsync(resource, currentAsync)
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		applyMacOrderAsync(resource, currentAsync)
		return false, fmt.Errorf("%s delete work request %s finished with status %s", macOrderKind, workRequestID, currentAsync.RawStatus)
	case shared.OSOKAsyncClassSucceeded:
		response, err := confirmMacOrderDeleteRead(ctx, resource, currentID, c.client, nil)
		if err != nil {
			if macOrderDeleteNotFound(err) {
				markMacOrderDeleted(resource, "OCI resource deleted")
				return true, nil
			}
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, err
		}
		if err := applyMacOrderStatus(resource, response); err != nil {
			return false, err
		}
		if macOrderDeleteConfirmed(response) {
			markMacOrderDeleted(resource, "OCI MacOrder cancellation confirmed")
			return true, nil
		}
		if macOrderDeletePending(response) {
			markMacOrderDeletePending(resource, "OCI MacOrder cancellation is in progress")
			return false, nil
		}
		return false, fmt.Errorf("%s delete confirmation returned unexpected lifecycle state %q", macOrderKind, resource.Status.LifecycleState)
	default:
		return false, fmt.Errorf("%s delete work request %s projected unsupported async class %s", macOrderKind, workRequestID, currentAsync.NormalizedClass)
	}
}

func applyMacOrderRuntimeHooks(
	hooks *MacOrderRuntimeHooks,
	client macOrderOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedMacOrderRuntimeSemantics()
	hooks.BuildCreateBody = buildMacOrderCreateBody
	hooks.BuildUpdateBody = buildMacOrderUpdateBody
	hooks.List.Fields = macOrderListFields()
	hooks.List.Call = listMacOrdersAllPages(hooks.List.Call)
	hooks.Identity.GuardExistingBeforeCreate = guardMacOrderExistingBeforeCreate
	hooks.Async.Adapter = macOrderWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getMacOrderWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveMacOrderWorkRequestAction
	hooks.Async.ResolvePhase = resolveMacOrderWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverMacOrderIDFromWorkRequest
	hooks.Async.Message = macOrderWorkRequestMessage
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *mngdmacv1beta1.MacOrder, currentID string) (any, error) {
		return confirmMacOrderDeleteRead(ctx, resource, currentID, client, initErr)
	}
}

func reviewedMacOrderRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "mngdmac",
		FormalSlug:    "macorder",
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
			ProvisioningStates: []string{string(mngdmacsdk.MacOrderLifecycleStateCreating)},
			UpdatingStates:     []string{string(mngdmacsdk.MacOrderLifecycleStateUpdating)},
			ActiveStates:       []string{string(mngdmacsdk.MacOrderLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(mngdmacsdk.MacOrderLifecycleStateDeleting)},
			TerminalStates: []string{string(mngdmacsdk.MacOrderLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "ipRange", "orderDescription", "orderSize", "shape"},
			ForceNew:      []string{"commitmentTerm", "compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: macOrderKind, Action: "CreateMacOrder"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: macOrderKind, Action: "UpdateMacOrder"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: macOrderKind, Action: "CancelMacOrder"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetMacOrder",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource", EntityType: macOrderKind, Action: "GetWorkRequest"},
				{Helper: "tfresource.CreateResource", EntityType: macOrderKind, Action: "GetMacOrder"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetMacOrder",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource", EntityType: macOrderKind, Action: "GetWorkRequest"},
				{Helper: "tfresource.UpdateResource", EntityType: macOrderKind, Action: "GetMacOrder"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> CancelMacOrder/GetMacOrder/ListMacOrders confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource", EntityType: macOrderKind, Action: "GetWorkRequest"},
				{Helper: "tfresource.DeleteResource", EntityType: macOrderKind, Action: "GetMacOrder"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func macOrderListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func macOrderDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MacOrderId", RequestName: "macOrderId", Contribution: "path", PreferResourceID: true},
	}
}

func guardMacOrderExistingBeforeCreate(
	_ context.Context,
	resource *mngdmacv1beta1.MacOrder,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", macOrderKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildMacOrderCreateBody(
	_ context.Context,
	resource *mngdmacv1beta1.MacOrder,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", macOrderKind)
	}

	details := mngdmacsdk.CreateMacOrderDetails{
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		OrderDescription: common.String(resource.Spec.OrderDescription),
		OrderSize:        common.Int(resource.Spec.OrderSize),
		Shape:            mngdmacsdk.MacOrderShapeEnum(resource.Spec.Shape),
		CommitmentTerm:   mngdmacsdk.MacOrderCommitmentTermEnum(resource.Spec.CommitmentTerm),
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		details.DisplayName = common.String(displayName)
	}
	if ipRange := strings.TrimSpace(resource.Spec.IpRange); ipRange != "" {
		details.IpRange = common.String(ipRange)
	}
	return details, nil
}

func buildMacOrderUpdateBody(
	_ context.Context,
	resource *mngdmacv1beta1.MacOrder,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return mngdmacsdk.UpdateMacOrderDetails{}, false, fmt.Errorf("%s resource is nil", macOrderKind)
	}

	current, err := macOrderFromResponse(currentResponse)
	if err != nil {
		return mngdmacsdk.UpdateMacOrderDetails{}, false, err
	}

	body := mngdmacsdk.UpdateMacOrderDetails{}
	updateNeeded := false
	if desired, ok := macOrderDesiredOptionalStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		body.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := macOrderDesiredRequiredStringUpdate(resource.Spec.OrderDescription, current.OrderDescription); ok {
		body.OrderDescription = desired
		updateNeeded = true
	}
	if desired, ok := macOrderDesiredIntUpdate(resource.Spec.OrderSize, current.OrderSize); ok {
		body.OrderSize = desired
		updateNeeded = true
	}
	if desired, ok := macOrderDesiredShapeUpdate(resource.Spec.Shape, current.Shape); ok {
		body.Shape = desired
		updateNeeded = true
	}
	if desired, ok := macOrderDesiredOptionalStringUpdate(resource.Spec.IpRange, current.IpRange); ok {
		body.IpRange = desired
		updateNeeded = true
	}

	return body, updateNeeded, nil
}

func macOrderDesiredRequiredStringUpdate(spec string, current *string) (*string, bool) {
	if strings.TrimSpace(spec) == "" {
		return nil, false
	}
	return macOrderDesiredOptionalStringUpdate(spec, current)
}

func macOrderDesiredOptionalStringUpdate(spec string, current *string) (*string, bool) {
	if strings.TrimSpace(spec) == "" {
		return nil, false
	}

	currentValue := strings.TrimSpace(stringValue(current))
	if spec == currentValue {
		return nil, false
	}
	return common.String(spec), true
}

func macOrderDesiredIntUpdate(spec int, current *int) (*int, bool) {
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Int(spec), true
}

func macOrderDesiredShapeUpdate(
	spec string,
	current mngdmacsdk.MacOrderShapeEnum,
) (mngdmacsdk.MacOrderShapeEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return mngdmacsdk.MacOrderShapeEnum(spec), true
}

func getMacOrderWorkRequest(
	ctx context.Context,
	client macOrderOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", macOrderKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", macOrderKind)
	}

	response, err := client.GetWorkRequest(ctx, mngdmacsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveMacOrderWorkRequestAction(workRequest any) (string, error) {
	current, err := macOrderWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveMacOrderWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := macOrderWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := macOrderWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverMacOrderIDFromWorkRequest(
	_ *mngdmacv1beta1.MacOrder,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := macOrderWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveMacOrderIDFromWorkRequest(current, macOrderWorkRequestActionForPhase(phase))
}

func macOrderWorkRequestFromAny(workRequest any) (mngdmacsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case mngdmacsdk.WorkRequest:
		return current, nil
	case *mngdmacsdk.WorkRequest:
		if current == nil {
			return mngdmacsdk.WorkRequest{}, fmt.Errorf("%s work request is nil", macOrderKind)
		}
		return *current, nil
	default:
		return mngdmacsdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", macOrderKind, workRequest)
	}
}

func macOrderWorkRequestPhaseFromOperationType(
	operationType mngdmacsdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case mngdmacsdk.OperationTypeCreateMacOrder:
		return shared.OSOKAsyncPhaseCreate, true
	case mngdmacsdk.OperationTypeUpdateMacOrder:
		return shared.OSOKAsyncPhaseUpdate, true
	case mngdmacsdk.OperationTypeCancelMacOrder, mngdmacsdk.OperationTypeDeleteMacOrder:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func macOrderWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) mngdmacsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return mngdmacsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return mngdmacsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return mngdmacsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveMacOrderIDFromWorkRequest(
	workRequest mngdmacsdk.WorkRequest,
	action mngdmacsdk.ActionTypeEnum,
) (string, error) {
	if id, ok := resolveMacOrderIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveMacOrderIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a MacOrder identifier", macOrderKind, stringValue(workRequest.Id))
}

func resolveMacOrderIDFromResources(
	resources []mngdmacsdk.WorkRequestResource,
	action mngdmacsdk.ActionTypeEnum,
	preferMacOrderOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferMacOrderOnly && !isMacOrderWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func isMacOrderWorkRequestResource(resource mngdmacsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "macorder", "mac_order", "macorders", "mac_orders":
		return true
	}
	if strings.Contains(entityType, "macorder") || strings.Contains(entityType, "mac_order") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/macorders/")
}

func macOrderWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := macOrderWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", macOrderKind, phase, stringValue(current.Id), current.Status)
}

func confirmMacOrderDeleteRead(
	ctx context.Context,
	resource *mngdmacv1beta1.MacOrder,
	currentID string,
	client macOrderOCIClient,
	initErr error,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", macOrderKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", macOrderKind)
	}

	currentID = strings.TrimSpace(currentID)
	if currentID != "" {
		response, err := client.GetMacOrder(ctx, mngdmacsdk.GetMacOrderRequest{
			MacOrderId: common.String(currentID),
		})
		if err != nil {
			return nil, err
		}
		return normalizeMacOrderDeleteConfirmResponse(response), nil
	}

	summary, err := findMacOrderDeleteSummary(ctx, resource, client)
	if err != nil {
		return nil, err
	}

	id := strings.TrimSpace(stringValue(summary.Id))
	if id == "" {
		return normalizeMacOrderDeleteConfirmSummary(summary), nil
	}

	response, err := client.GetMacOrder(ctx, mngdmacsdk.GetMacOrderRequest{
		MacOrderId: common.String(id),
	})
	if err != nil {
		return nil, err
	}
	return normalizeMacOrderDeleteConfirmResponse(response), nil
}

func findMacOrderDeleteSummary(
	ctx context.Context,
	resource *mngdmacv1beta1.MacOrder,
	client macOrderOCIClient,
) (mngdmacsdk.MacOrderSummary, error) {
	compartmentID := macOrderDeleteCompartmentID(resource)
	displayName := macOrderDeleteDisplayName(resource)
	if compartmentID == "" || displayName == "" {
		return mngdmacsdk.MacOrderSummary{}, fmt.Errorf("%s delete confirmation requires a tracked id or exact spec/status compartmentId + displayName", macOrderKind)
	}

	response, err := listMacOrdersAllPages(client.ListMacOrders)(ctx, mngdmacsdk.ListMacOrdersRequest{
		CompartmentId: common.String(compartmentID),
		DisplayName:   common.String(displayName),
	})
	if err != nil {
		return mngdmacsdk.MacOrderSummary{}, err
	}

	matches := make([]mngdmacsdk.MacOrderSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if strings.TrimSpace(stringValue(item.CompartmentId)) != compartmentID {
			continue
		}
		if strings.TrimSpace(stringValue(item.DisplayName)) != displayName {
			continue
		}
		matches = append(matches, item)
	}

	switch len(matches) {
	case 0:
		return mngdmacsdk.MacOrderSummary{}, errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			Description:    fmt.Sprintf("%s delete confirmation could not find a matching MacOrder", macOrderKind),
		}
	case 1:
		return matches[0], nil
	default:
		return mngdmacsdk.MacOrderSummary{}, fmt.Errorf("%s delete confirmation matched multiple MacOrders for compartmentId %q and displayName %q", macOrderKind, compartmentID, displayName)
	}
}

func normalizeMacOrderDeleteConfirmResponse(
	response mngdmacsdk.GetMacOrderResponse,
) mngdmacsdk.GetMacOrderResponse {
	response.MacOrder = normalizeMacOrderDeleteConfirmBody(response.MacOrder)
	return response
}

func normalizeMacOrderDeleteConfirmSummary(
	summary mngdmacsdk.MacOrderSummary,
) mngdmacsdk.GetMacOrderResponse {
	return normalizeMacOrderDeleteConfirmResponse(mngdmacsdk.GetMacOrderResponse{
		MacOrder: mngdmacsdk.MacOrder{
			Id:                 summary.Id,
			CompartmentId:      summary.CompartmentId,
			OrderDescription:   summary.OrderDescription,
			OrderSize:          summary.OrderSize,
			IsDocusigned:       summary.IsDocusigned,
			Shape:              summary.Shape,
			TimeCreated:        summary.TimeCreated,
			CommitmentTerm:     summary.CommitmentTerm,
			OrderStatus:        summary.OrderStatus,
			LifecycleState:     summary.LifecycleState,
			DisplayName:        summary.DisplayName,
			IpRange:            summary.IpRange,
			TimeUpdated:        summary.TimeUpdated,
			TimeBillingStarted: summary.TimeBillingStarted,
			TimeBillingEnded:   summary.TimeBillingEnded,
			LifecycleDetails:   summary.LifecycleDetails,
		},
	})
}

func normalizeMacOrderDeleteConfirmBody(body mngdmacsdk.MacOrder) mngdmacsdk.MacOrder {
	if macOrderIsCanceled(body.OrderStatus, body.TimeCanceled) {
		body.LifecycleState = mngdmacsdk.MacOrderLifecycleStateDeleted
	}
	return body
}

func macOrderIsCanceled(
	orderStatus mngdmacsdk.MacOrderOrderStatusEnum,
	timeCanceled *common.SDKTime,
) bool {
	return orderStatus == mngdmacsdk.MacOrderOrderStatusCanceled || timeCanceled != nil
}

func macOrderDeleteCompartmentID(resource *mngdmacv1beta1.MacOrder) string {
	if resource == nil {
		return ""
	}
	if compartmentID := strings.TrimSpace(resource.Status.CompartmentId); compartmentID != "" {
		return compartmentID
	}
	return strings.TrimSpace(resource.Spec.CompartmentId)
}

func macOrderDeleteDisplayName(resource *mngdmacv1beta1.MacOrder) string {
	if resource == nil {
		return ""
	}
	if displayName := strings.TrimSpace(resource.Status.DisplayName); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(resource.Spec.DisplayName)
}

func trackedMacOrderID(resource *mngdmacv1beta1.MacOrder) string {
	if resource == nil {
		return ""
	}
	if resource.Status.Id != "" {
		return strings.TrimSpace(resource.Status.Id)
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func listMacOrdersAllPages(call macOrderListCall) macOrderListCall {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error) {
		var combined mngdmacsdk.ListMacOrdersResponse
		firstPage := true
		for {
			response, err := call(ctx, request)
			if err != nil {
				return mngdmacsdk.ListMacOrdersResponse{}, err
			}
			if firstPage {
				combined = response
				combined.Items = append([]mngdmacsdk.MacOrderSummary(nil), response.Items...)
				firstPage = false
			} else {
				combined.Items = append(combined.Items, response.Items...)
			}
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func macOrderFromResponse(response any) (mngdmacsdk.MacOrder, error) {
	switch current := response.(type) {
	case mngdmacsdk.GetMacOrderResponse:
		return current.MacOrder, nil
	case *mngdmacsdk.GetMacOrderResponse:
		if current == nil {
			return mngdmacsdk.MacOrder{}, fmt.Errorf("%s response is nil", macOrderKind)
		}
		return current.MacOrder, nil
	case mngdmacsdk.CreateMacOrderResponse:
		return current.MacOrder, nil
	case *mngdmacsdk.CreateMacOrderResponse:
		if current == nil {
			return mngdmacsdk.MacOrder{}, fmt.Errorf("%s response is nil", macOrderKind)
		}
		return current.MacOrder, nil
	case mngdmacsdk.MacOrder:
		return current, nil
	case *mngdmacsdk.MacOrder:
		if current == nil {
			return mngdmacsdk.MacOrder{}, fmt.Errorf("%s body is nil", macOrderKind)
		}
		return *current, nil
	default:
		return mngdmacsdk.MacOrder{}, fmt.Errorf("%s response does not expose a MacOrder body", macOrderKind)
	}
}

func macOrderDeleteConfirmed(response any) bool {
	order, err := macOrderFromResponse(response)
	if err != nil {
		return false
	}
	lifecycleState := strings.ToUpper(string(order.LifecycleState))
	return lifecycleState == string(mngdmacsdk.MacOrderLifecycleStateDeleted)
}

func macOrderDeletePending(response any) bool {
	order, err := macOrderFromResponse(response)
	if err != nil {
		return false
	}
	lifecycleState := strings.ToUpper(string(order.LifecycleState))
	return lifecycleState == string(mngdmacsdk.MacOrderLifecycleStateDeleting)
}

func macOrderDeleteNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func buildMacOrderDeleteAsyncOperation(
	resource *mngdmacv1beta1.MacOrder,
	workRequest any,
) (*shared.OSOKAsyncOperation, error) {
	current, err := macOrderWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	rawAction, err := resolveMacOrderWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(
		&resource.Status.OsokStatus,
		macOrderWorkRequestAsyncAdapter,
		servicemanager.WorkRequestAsyncInput{
			RawStatus:        string(current.Status),
			RawAction:        rawAction,
			RawOperationType: string(current.OperationType),
			WorkRequestID:    stringValue(current.Id),
			PercentComplete:  current.PercentComplete,
			Message:          macOrderWorkRequestMessage(shared.OSOKAsyncPhaseDelete, current),
			FallbackPhase:    shared.OSOKAsyncPhaseDelete,
		},
	)
}

func applyMacOrderAsync(resource *mngdmacv1beta1.MacOrder, current *shared.OSOKAsyncOperation) {
	if resource == nil || current == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
}

func applyMacOrderStatus(resource *mngdmacv1beta1.MacOrder, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", macOrderKind)
	}

	order, err := macOrderFromResponse(response)
	if err != nil {
		return err
	}

	status := resource.Status
	payload, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal %s observed state: %w", macOrderKind, err)
	}
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("decode %s observed state into status: %w", macOrderKind, err)
	}
	status.OsokStatus = resource.Status.OsokStatus
	if id := stringValue(order.Id); id != "" {
		status.Id = id
		status.OsokStatus.Ocid = shared.OCID(id)
	}
	resource.Status = status
	return nil
}

func markMacOrderDeletePending(resource *mngdmacv1beta1.MacOrder, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.UpdatedAt = &now
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       resource.Status.LifecycleState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
}

func markMacOrderDeleted(resource *mngdmacv1beta1.MacOrder, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.DeletedAt = &now
	resource.Status.OsokStatus.UpdatedAt = &now
	resource.Status.OsokStatus.Message = message
	resource.Status.OsokStatus.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		message,
		loggerutil.OSOKLogger{},
	)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
