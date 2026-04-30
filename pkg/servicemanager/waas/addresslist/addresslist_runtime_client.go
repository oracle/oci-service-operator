/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package addresslist

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type addressListListRequest struct {
	CompartmentId *string
	Name          *string
	Id            *string
	Page          *string
}

type addressListCompartmentMoveClient interface {
	ChangeAddressListCompartment(context.Context, waassdk.ChangeAddressListCompartmentRequest) (waassdk.ChangeAddressListCompartmentResponse, error)
}

func init() {
	registerAddressListRuntimeHooksMutator(func(manager *AddressListServiceManager, hooks *AddressListRuntimeHooks) {
		moveClient, initErr := newAddressListCompartmentMoveClient(manager)
		applyAddressListRuntimeHooks(hooks, moveClient, initErr)
	})
}

func newAddressListCompartmentMoveClient(manager *AddressListServiceManager) (addressListCompartmentMoveClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("addressList manager is nil")
	}
	client, err := waassdk.NewWaasClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAddressListRuntimeHooks(
	hooks *AddressListRuntimeHooks,
	moveClient addressListCompartmentMoveClient,
	moveClientInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAddressListRuntimeSemantics()
	hooks.BuildCreateBody = buildAddressListCreateBody
	hooks.BuildUpdateBody = buildAddressListUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardAddressListExistingBeforeCreate
	hooks.Read.List = addressListPaginatedListReadOperation(hooks)
	hooks.ParityHooks.RequiresParityHandling = addressListRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *waasv1beta1.AddressList,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyAddressListCompartmentMove(
			ctx,
			resource,
			currentResponse,
			moveClient,
			hooks.Get.Call,
			hooks.Update.Call,
			moveClientInitErr,
		)
	}
	hooks.DeleteHooks.HandleError = handleAddressListDeleteError
	wrapAddressListDeleteConfirmation(hooks)
}

func newAddressListRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "waas",
		FormalSlug:        "addresslist",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(waassdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(waassdk.LifecycleStatesUpdating)},
			ActiveStates:       []string{string(waassdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(waassdk.LifecycleStatesDeleting)},
			TerminalStates: []string{string(waassdk.LifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"compartmentId", "displayName", "addresses", "freeformTags", "definedTags"},
			Mutable:         []string{"compartmentId", "displayName", "addresses", "freeformTags", "definedTags"},
			ForceNew:        []string{},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AddressList", Action: "CreateAddressList"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AddressList", Action: "UpdateAddressList"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AddressList", Action: "DeleteAddressList"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AddressList", Action: "GetAddressList"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AddressList", Action: "GetAddressList"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AddressList", Action: "GetAddressList"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func guardAddressListExistingBeforeCreate(
	_ context.Context,
	resource *waasv1beta1.AddressList,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("addressList resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildAddressListCreateBody(
	_ context.Context,
	resource *waasv1beta1.AddressList,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("addressList resource is nil")
	}
	if err := validateAddressListSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	details := waassdk.CreateAddressListDetails{
		CompartmentId: common.String(spec.CompartmentId),
		DisplayName:   common.String(spec.DisplayName),
		Addresses:     append([]string(nil), spec.Addresses...),
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = addressListDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details, nil
}

func buildAddressListUpdateBody(
	_ context.Context,
	resource *waasv1beta1.AddressList,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return waassdk.UpdateAddressListDetails{}, false, fmt.Errorf("addressList resource is nil")
	}
	if err := validateAddressListSpec(resource.Spec); err != nil {
		return waassdk.UpdateAddressListDetails{}, false, err
	}

	current, err := addressListFromResponse(currentResponse)
	if err != nil {
		return waassdk.UpdateAddressListDetails{}, false, err
	}

	details, updateNeeded := addressListUpdateDetails(resource.Spec, current)
	return details, updateNeeded, nil
}

func addressListUpdateDetails(
	spec waasv1beta1.AddressListSpec,
	current waassdk.AddressList,
) (waassdk.UpdateAddressListDetails, bool) {
	details, updateNeeded := addressListMutableUpdateDetails(spec, current)
	if addressListCompartmentNeedsMove(spec, current) {
		updateNeeded = true
	}
	return details, updateNeeded
}

func addressListMutableUpdateDetails(
	spec waasv1beta1.AddressListSpec,
	current waassdk.AddressList,
) (waassdk.UpdateAddressListDetails, bool) {
	details := waassdk.UpdateAddressListDetails{}
	updateNeeded := false
	if desired, ok := addressListStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if !slices.Equal(spec.Addresses, current.Addresses) {
		details.Addresses = append([]string(nil), spec.Addresses...)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil && !addressListDefinedTagsEqual(spec.DefinedTags, current.DefinedTags) {
		details.DefinedTags = addressListDefinedTagsFromSpec(spec.DefinedTags)
		updateNeeded = true
	}
	return details, updateNeeded
}

func addressListRequiresCompartmentMove(resource *waasv1beta1.AddressList, currentResponse any) bool {
	if resource == nil {
		return false
	}
	current, err := addressListFromResponse(currentResponse)
	if err != nil {
		return false
	}
	return addressListCompartmentNeedsMove(resource.Spec, current)
}

func addressListCompartmentNeedsMove(spec waasv1beta1.AddressListSpec, current waassdk.AddressList) bool {
	desired := strings.TrimSpace(spec.CompartmentId)
	observed := strings.TrimSpace(addressListStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyAddressListCompartmentMove(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
	currentResponse any,
	moveClient addressListCompartmentMoveClient,
	getAddressList func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error),
	updateAddressList func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error),
	initErr error,
) (servicemanager.OSOKResponse, error) {
	addressListID, compartmentID, err := addressListCompartmentMoveInputs(resource, currentResponse, moveClient, getAddressList, initErr)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if err := addressListChangeCompartment(ctx, resource, moveClient, addressListID, compartmentID); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	refreshed, err := addressListRefreshAfterMove(ctx, resource, getAddressList, addressListID)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return addressListApplyPendingMutableUpdate(ctx, resource, getAddressList, updateAddressList, addressListID, refreshed.AddressList)
}

func addressListCompartmentMoveInputs(
	resource *waasv1beta1.AddressList,
	currentResponse any,
	moveClient addressListCompartmentMoveClient,
	getAddressList func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error),
	initErr error,
) (string, string, error) {
	if resource == nil {
		return "", "", fmt.Errorf("addressList resource is nil")
	}
	if initErr != nil {
		return "", "", fmt.Errorf("initialize AddressList compartment move OCI client: %w", initErr)
	}
	if moveClient == nil {
		return "", "", fmt.Errorf("addressList compartment move OCI client is not configured")
	}
	if getAddressList == nil {
		return "", "", fmt.Errorf("addressList GetAddressList call is not configured")
	}

	current, err := addressListFromResponse(currentResponse)
	if err != nil {
		return "", "", err
	}
	addressListID := strings.TrimSpace(addressListStringValue(current.Id))
	if addressListID == "" {
		addressListID = trackedAddressListID(resource)
	}
	if addressListID == "" {
		return "", "", fmt.Errorf("addressList compartment move requires a tracked addressList id")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return "", "", fmt.Errorf("addressList compartment move requires spec.compartmentId")
	}
	return addressListID, compartmentID, nil
}

func addressListChangeCompartment(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
	moveClient addressListCompartmentMoveClient,
	addressListID string,
	compartmentID string,
) error {
	response, err := moveClient.ChangeAddressListCompartment(ctx, waassdk.ChangeAddressListCompartmentRequest{
		AddressListId: common.String(addressListID),
		ChangeAddressListCompartmentDetails: waassdk.ChangeAddressListCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
		OpcRetryToken: addressListCompartmentMoveRetryToken(resource),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return nil
}

func addressListRefreshAfterMove(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
	getAddressList func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error),
	addressListID string,
) (waassdk.GetAddressListResponse, error) {
	refreshed, err := getAddressList(ctx, waassdk.GetAddressListRequest{AddressListId: common.String(addressListID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return waassdk.GetAddressListResponse{}, fmt.Errorf("refresh AddressList after compartment move: %w", err)
	}
	return refreshed, nil
}

func addressListApplyPendingMutableUpdate(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
	getAddressList func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error),
	updateAddressList func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error),
	addressListID string,
	current waassdk.AddressList,
) (servicemanager.OSOKResponse, error) {
	if addressListShouldWaitForLifecycle(current) {
		return addressListProjectMoveStatus(resource, current), nil
	}

	details, updateNeeded := addressListMutableUpdateDetails(resource.Spec, current)
	if !updateNeeded {
		return addressListProjectMoveStatus(resource, current), nil
	}
	if updateAddressList == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("addressList UpdateAddressList call is not configured")
	}

	response, err := updateAddressList(ctx, waassdk.UpdateAddressListRequest{
		AddressListId:            common.String(addressListID),
		UpdateAddressListDetails: details,
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	refreshed, err := addressListRefreshAfterUpdate(ctx, resource, getAddressList, addressListID)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return addressListProjectMoveStatus(resource, refreshed.AddressList), nil
}

func addressListShouldWaitForLifecycle(current waassdk.AddressList) bool {
	state := current.LifecycleState
	return state == waassdk.LifecycleStatesCreating ||
		state == waassdk.LifecycleStatesUpdating ||
		state == waassdk.LifecycleStatesDeleting ||
		state == waassdk.LifecycleStatesFailed ||
		state == waassdk.LifecycleStatesDeleted
}

func addressListRefreshAfterUpdate(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
	getAddressList func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error),
	addressListID string,
) (waassdk.GetAddressListResponse, error) {
	refreshed, err := getAddressList(ctx, waassdk.GetAddressListRequest{AddressListId: common.String(addressListID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return waassdk.GetAddressListResponse{}, fmt.Errorf("refresh AddressList after mutable update: %w", err)
	}
	return refreshed, nil
}

func addressListCompartmentMoveRetryToken(resource *waasv1beta1.AddressList) *string {
	if resource == nil || resource.UID == "" {
		return nil
	}
	return common.String(string(resource.UID) + "-move-compartment")
}

func addressListProjectMoveStatus(
	resource *waasv1beta1.AddressList,
	current waassdk.AddressList,
) servicemanager.OSOKResponse {
	addressListProjectObservedStatus(resource, current)

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now

	state := current.LifecycleState
	condition, conditionStatus, shouldRequeue, message := addressListConditionForLifecycle(state, shared.Active)
	if condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating {
		projection := servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}, loggerutil.OSOKLogger{})
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: 0,
		}
	}

	servicemanager.ClearAsyncOperation(status)
	status.Message = message
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  condition != shared.Failed,
		ShouldRequeue: shouldRequeue,
	}
}

func addressListProjectObservedStatus(resource *waasv1beta1.AddressList, current waassdk.AddressList) {
	if resource == nil {
		return
	}
	resource.Status.Id = addressListStringValue(current.Id)
	resource.Status.CompartmentId = addressListStringValue(current.CompartmentId)
	resource.Status.DisplayName = addressListStringValue(current.DisplayName)
	if current.AddressCount != nil {
		resource.Status.AddressCount = *current.AddressCount
	}
	resource.Status.Addresses = append([]string(nil), current.Addresses...)
	resource.Status.FreeformTags = maps.Clone(current.FreeformTags)
	resource.Status.DefinedTags = addressListStatusDefinedTags(current.DefinedTags)
	resource.Status.LifecycleState = string(current.LifecycleState)
	if current.TimeCreated != nil {
		resource.Status.TimeCreated = current.TimeCreated.String()
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func addressListConditionForLifecycle(
	state waassdk.LifecycleStatesEnum,
	fallback shared.OSOKConditionType,
) (shared.OSOKConditionType, corev1.ConditionStatus, bool, string) {
	switch state {
	case waassdk.LifecycleStatesCreating:
		return shared.Provisioning, corev1.ConditionTrue, true, "OCI AddressList is creating"
	case waassdk.LifecycleStatesUpdating:
		return shared.Updating, corev1.ConditionTrue, true, "OCI AddressList update is in progress"
	case waassdk.LifecycleStatesDeleting:
		return shared.Terminating, corev1.ConditionTrue, true, "OCI AddressList delete is in progress"
	case waassdk.LifecycleStatesFailed:
		return shared.Failed, corev1.ConditionFalse, false, "OCI AddressList is failed"
	case waassdk.LifecycleStatesDeleted:
		return shared.Terminating, corev1.ConditionTrue, false, "OCI AddressList is deleted"
	case waassdk.LifecycleStatesActive:
		return shared.Active, corev1.ConditionTrue, false, "OCI AddressList is active"
	default:
		return fallback, corev1.ConditionTrue, fallback == shared.Provisioning || fallback == shared.Updating || fallback == shared.Terminating, "OCI AddressList state observed"
	}
}

func validateAddressListSpec(spec waasv1beta1.AddressListSpec) error {
	var problems []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		problems = append(problems, "displayName is required")
	}
	if len(spec.Addresses) == 0 {
		problems = append(problems, "addresses must contain at least one entry")
	}
	for index, address := range spec.Addresses {
		if strings.TrimSpace(address) == "" {
			problems = append(problems, fmt.Sprintf("addresses[%d] must not be empty", index))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid AddressList spec: %s", strings.Join(problems, "; "))
}

func addressListPaginatedListReadOperation(hooks *AddressListRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}

	listCall := hooks.List.Call
	return &generatedruntime.Operation{
		NewRequest: func() any { return &addressListListRequest{} },
		Fields: []generatedruntime.RequestField{
			{
				FieldName:    "CompartmentId",
				RequestName:  "compartmentId",
				Contribution: "query",
				LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
			},
			{
				FieldName:    "Name",
				RequestName:  "name",
				Contribution: "query",
				LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
			},
			{
				FieldName:        "Id",
				RequestName:      "id",
				Contribution:     "query",
				PreferResourceID: true,
				LookupPaths:      []string{"status.id", "status.status.ocid", "id", "ocid"},
			},
			{FieldName: "Page", RequestName: "page", Contribution: "query"},
		},
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*addressListListRequest)
			if !ok {
				return nil, fmt.Errorf("expected *addresslist.addressListListRequest, got %T", request)
			}
			return listAddressListPages(ctx, listCall, *typed)
		},
	}
}

func listAddressListPages(
	ctx context.Context,
	call func(context.Context, waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error),
	request addressListListRequest,
) (waassdk.ListAddressListsResponse, error) {
	if call == nil {
		return waassdk.ListAddressListsResponse{}, fmt.Errorf("addressList list operation is not configured")
	}

	sdkRequest := addressListSDKListRequest(request)
	seenPages := map[string]struct{}{}
	var combined waassdk.ListAddressListsResponse
	for {
		response, err := call(ctx, sdkRequest)
		if err != nil {
			return waassdk.ListAddressListsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := strings.TrimSpace(addressListStringValue(response.OpcNextPage))
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return waassdk.ListAddressListsResponse{}, fmt.Errorf("addressList list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		sdkRequest.Page = common.String(nextPage)
	}
}

func addressListSDKListRequest(request addressListListRequest) waassdk.ListAddressListsRequest {
	sdkRequest := waassdk.ListAddressListsRequest{
		CompartmentId: request.CompartmentId,
		Page:          request.Page,
	}
	if id := strings.TrimSpace(addressListStringValue(request.Id)); id != "" {
		sdkRequest.Id = []string{id}
	}
	if name := addressListStringValue(request.Name); name != "" {
		sdkRequest.Name = []string{name}
	}
	return sdkRequest
}

func handleAddressListDeleteError(resource *waasv1beta1.AddressList, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("addressList delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapAddressListDeleteConfirmation(hooks *AddressListRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}

	getAddressList := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AddressListServiceClient) AddressListServiceClient {
		return addressListDeleteConfirmationClient{
			delegate:       delegate,
			getAddressList: getAddressList,
		}
	})
}

type addressListDeleteConfirmationClient struct {
	delegate       AddressListServiceClient
	getAddressList func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error)
}

func (c addressListDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c addressListDeleteConfirmationClient) Delete(ctx context.Context, resource *waasv1beta1.AddressList) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c addressListDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *waasv1beta1.AddressList,
) error {
	if c.getAddressList == nil || resource == nil {
		return nil
	}
	addressListID := trackedAddressListID(resource)
	if addressListID == "" {
		return nil
	}

	_, err := c.getAddressList(ctx, waassdk.GetAddressListRequest{AddressListId: common.String(addressListID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("addressList delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedAddressListID(resource *waasv1beta1.AddressList) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func addressListFromResponse(currentResponse any) (waassdk.AddressList, error) {
	if currentResponse == nil {
		return waassdk.AddressList{}, fmt.Errorf("current AddressList response is nil")
	}
	if current, ok := addressListFromValueResponse(currentResponse); ok {
		return current, nil
	}
	if current, ok, err := addressListFromPointerEntityResponse(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := addressListFromPointerOperationResponse(currentResponse); ok || err != nil {
		return current, err
	}
	return waassdk.AddressList{}, fmt.Errorf("unexpected current AddressList response type %T", currentResponse)
}

func addressListFromValueResponse(currentResponse any) (waassdk.AddressList, bool) {
	switch current := currentResponse.(type) {
	case waassdk.AddressList:
		return current, true
	case waassdk.AddressListSummary:
		return addressListFromSummary(current), true
	case waassdk.CreateAddressListResponse:
		return current.AddressList, true
	case waassdk.GetAddressListResponse:
		return current.AddressList, true
	case waassdk.UpdateAddressListResponse:
		return current.AddressList, true
	default:
		return waassdk.AddressList{}, false
	}
}

func addressListFromPointerEntityResponse(currentResponse any) (waassdk.AddressList, bool, error) {
	switch current := currentResponse.(type) {
	case *waassdk.AddressList:
		if current == nil {
			return waassdk.AddressList{}, true, fmt.Errorf("current AddressList response is nil")
		}
		return *current, true, nil
	case *waassdk.AddressListSummary:
		if current == nil {
			return waassdk.AddressList{}, true, fmt.Errorf("current AddressList summary response is nil")
		}
		return addressListFromSummary(*current), true, nil
	default:
		return waassdk.AddressList{}, false, nil
	}
}

func addressListFromPointerOperationResponse(currentResponse any) (waassdk.AddressList, bool, error) {
	switch current := currentResponse.(type) {
	case *waassdk.CreateAddressListResponse:
		if current == nil {
			return waassdk.AddressList{}, true, fmt.Errorf("current CreateAddressList response is nil")
		}
		return current.AddressList, true, nil
	case *waassdk.GetAddressListResponse:
		if current == nil {
			return waassdk.AddressList{}, true, fmt.Errorf("current GetAddressList response is nil")
		}
		return current.AddressList, true, nil
	case *waassdk.UpdateAddressListResponse:
		if current == nil {
			return waassdk.AddressList{}, true, fmt.Errorf("current UpdateAddressList response is nil")
		}
		return current.AddressList, true, nil
	default:
		return waassdk.AddressList{}, false, nil
	}
}

func addressListFromSummary(summary waassdk.AddressListSummary) waassdk.AddressList {
	return waassdk.AddressList{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		AddressCount:   summary.AddressCount,
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
	}
}

func addressListStringUpdate(desired string, current *string) (*string, bool) {
	if desired == addressListStringValue(current) {
		return nil, false
	}
	return common.String(desired), true
}

func addressListStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func addressListDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		converted[namespace] = child
	}
	return converted
}

func addressListDefinedTagsEqual(spec map[string]shared.MapValue, current map[string]map[string]interface{}) bool {
	return reflect.DeepEqual(addressListDefinedTagsFromSpec(spec), addressListNormalizeDefinedTags(current))
}

func addressListStatusDefinedTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	normalized := addressListNormalizeDefinedTags(tags)
	if normalized == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(normalized))
	for namespace, values := range normalized {
		child := make(shared.MapValue, len(values))
		for key, value := range values {
			child[key] = fmt.Sprint(value)
		}
		converted[namespace] = child
	}
	return converted
}

func addressListNormalizeDefinedTags(tags map[string]map[string]interface{}) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	normalized := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			if value == nil {
				child[key] = ""
				continue
			}
			child[key] = fmt.Sprint(value)
		}
		normalized[namespace] = child
	}
	return normalized
}
