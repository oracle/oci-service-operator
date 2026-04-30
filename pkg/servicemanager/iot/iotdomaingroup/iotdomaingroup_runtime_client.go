/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package iotdomaingroup

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
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

const (
	iotDomainGroupDeletePendingMessage      = "OCI IotDomainGroup delete is in progress"
	iotDomainGroupPendingWriteDeleteMessage = "OCI IotDomainGroup create or update is in progress; delete is waiting"
)

type iotDomainGroupOCIClient interface {
	CreateIotDomainGroup(context.Context, iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error)
	GetIotDomainGroup(context.Context, iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error)
	ListIotDomainGroups(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error)
	UpdateIotDomainGroup(context.Context, iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error)
	DeleteIotDomainGroup(context.Context, iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error)
}

type iotDomainGroupAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e iotDomainGroupAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e iotDomainGroupAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerIotDomainGroupRuntimeHooksMutator(func(_ *IotDomainGroupServiceManager, hooks *IotDomainGroupRuntimeHooks) {
		applyIotDomainGroupRuntimeHooks(hooks)
	})
}

func applyIotDomainGroupRuntimeHooks(hooks *IotDomainGroupRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = iotDomainGroupRuntimeSemantics()
	hooks.BuildCreateBody = buildIotDomainGroupCreateBody
	hooks.BuildUpdateBody = buildIotDomainGroupUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardIotDomainGroupExistingBeforeCreate
	hooks.List.Fields = iotDomainGroupListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listIotDomainGroupsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeIotDomainGroupDesiredState
	hooks.DeleteHooks.HandleError = handleIotDomainGroupDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyIotDomainGroupDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markIotDomainGroupTerminating
	wrapIotDomainGroupDeleteConfirmation(hooks)
}

func newIotDomainGroupServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client iotDomainGroupOCIClient,
) IotDomainGroupServiceClient {
	manager := &IotDomainGroupServiceManager{Log: log}
	hooks := newIotDomainGroupRuntimeHooksWithOCIClient(client)
	applyIotDomainGroupRuntimeHooks(&hooks)
	delegate := defaultIotDomainGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*iotv1beta1.IotDomainGroup](
			buildIotDomainGroupGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapIotDomainGroupGeneratedClient(hooks, delegate)
}

func newIotDomainGroupRuntimeHooksWithOCIClient(client iotDomainGroupOCIClient) IotDomainGroupRuntimeHooks {
	hooks := newIotDomainGroupDefaultRuntimeHooks(iotsdk.IotClient{})
	hooks.Create.Call = func(ctx context.Context, request iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error) {
		if client == nil {
			return iotsdk.CreateIotDomainGroupResponse{}, fmt.Errorf("IotDomainGroup OCI client is not configured")
		}
		return client.CreateIotDomainGroup(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
		if client == nil {
			return iotsdk.GetIotDomainGroupResponse{}, fmt.Errorf("IotDomainGroup OCI client is not configured")
		}
		return client.GetIotDomainGroup(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
		if client == nil {
			return iotsdk.ListIotDomainGroupsResponse{}, fmt.Errorf("IotDomainGroup OCI client is not configured")
		}
		return client.ListIotDomainGroups(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
		if client == nil {
			return iotsdk.UpdateIotDomainGroupResponse{}, fmt.Errorf("IotDomainGroup OCI client is not configured")
		}
		return client.UpdateIotDomainGroup(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
		if client == nil {
			return iotsdk.DeleteIotDomainGroupResponse{}, fmt.Errorf("IotDomainGroup OCI client is not configured")
		}
		return client.DeleteIotDomainGroup(ctx, request)
	}
	return hooks
}

func iotDomainGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "iot",
		FormalSlug:    "iotdomaingroup",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(iotsdk.IotDomainGroupLifecycleStateCreating)},
			UpdatingStates:     []string{string(iotsdk.IotDomainGroupLifecycleStateUpdating)},
			ActiveStates:       []string{string(iotsdk.IotDomainGroupLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(iotsdk.IotDomainGroupLifecycleStateDeleting)},
			TerminalStates: []string{string(iotsdk.IotDomainGroupLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "type", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "type"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IotDomainGroup", Action: "CreateIotDomainGroup"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IotDomainGroup", Action: "UpdateIotDomainGroup"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IotDomainGroup", Action: "DeleteIotDomainGroup"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IotDomainGroup", Action: "GetIotDomainGroup"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IotDomainGroup", Action: "GetIotDomainGroup"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IotDomainGroup", Action: "GetIotDomainGroup"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func iotDomainGroupListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{
			FieldName:    "Type",
			RequestName:  "type",
			Contribution: "query",
			LookupPaths:  []string{"status.type", "spec.type", "type"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func guardIotDomainGroupExistingBeforeCreate(
	_ context.Context,
	resource *iotv1beta1.IotDomainGroup,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil || trackedIotDomainGroupID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildIotDomainGroupCreateBody(_ context.Context, resource *iotv1beta1.IotDomainGroup, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("IotDomainGroup resource is nil")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return nil, fmt.Errorf("compartmentId is required")
	}

	body := iotsdk.CreateIotDomainGroupDetails{
		CompartmentId: common.String(compartmentID),
	}
	if value := strings.TrimSpace(resource.Spec.Type); value != "" {
		domainType, ok := iotsdk.GetMappingCreateIotDomainGroupDetailsTypeEnum(value)
		if !ok {
			return nil, fmt.Errorf("unsupported type %q", resource.Spec.Type)
		}
		body.Type = domainType
	}
	setIotDomainGroupMutableCreateFields(&body, resource.Spec)
	return body, nil
}

func setIotDomainGroupMutableCreateFields(
	body *iotsdk.CreateIotDomainGroupDetails,
	spec iotv1beta1.IotDomainGroupSpec,
) {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.Description); value != "" {
		body.Description = common.String(value)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneIotDomainGroupStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = iotDomainGroupDefinedTags(spec.DefinedTags)
	}
}

func buildIotDomainGroupUpdateBody(
	_ context.Context,
	resource *iotv1beta1.IotDomainGroup,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("IotDomainGroup resource is nil")
	}
	current, ok := iotDomainGroupBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current IotDomainGroup response does not expose an IotDomainGroup body")
	}

	body := iotsdk.UpdateIotDomainGroupDetails{}
	updateNeeded := false
	setIotDomainGroupStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	setIotDomainGroupStringUpdate(&body.Description, &updateNeeded, resource.Spec.Description, current.Description)
	if resource.Spec.FreeformTags != nil {
		desired := cloneIotDomainGroupStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := iotDomainGroupDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func setIotDomainGroupStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func normalizeIotDomainGroupDesiredState(resource *iotv1beta1.IotDomainGroup, _ any) {
	if resource == nil {
		return
	}
	if domainType, ok := iotsdk.GetMappingIotDomainGroupTypeEnum(strings.TrimSpace(resource.Spec.Type)); ok {
		resource.Spec.Type = string(domainType)
	}
}

func iotDomainGroupBodyFromResponse(response any) (iotsdk.IotDomainGroup, bool) {
	switch current := response.(type) {
	case iotsdk.CreateIotDomainGroupResponse:
		return current.IotDomainGroup, true
	case *iotsdk.CreateIotDomainGroupResponse:
		return iotDomainGroupCreateResponseBody(current)
	case iotsdk.GetIotDomainGroupResponse:
		return current.IotDomainGroup, true
	case *iotsdk.GetIotDomainGroupResponse:
		return iotDomainGroupGetResponseBody(current)
	case iotsdk.IotDomainGroup:
		return current, true
	case *iotsdk.IotDomainGroup:
		return iotDomainGroupPointerBody(current)
	case iotsdk.IotDomainGroupSummary:
		return iotDomainGroupFromSummary(current), true
	case *iotsdk.IotDomainGroupSummary:
		return iotDomainGroupSummaryPointerBody(current)
	default:
		return iotsdk.IotDomainGroup{}, false
	}
}

func iotDomainGroupCreateResponseBody(
	response *iotsdk.CreateIotDomainGroupResponse,
) (iotsdk.IotDomainGroup, bool) {
	if response == nil {
		return iotsdk.IotDomainGroup{}, false
	}
	return response.IotDomainGroup, true
}

func iotDomainGroupGetResponseBody(
	response *iotsdk.GetIotDomainGroupResponse,
) (iotsdk.IotDomainGroup, bool) {
	if response == nil {
		return iotsdk.IotDomainGroup{}, false
	}
	return response.IotDomainGroup, true
}

func iotDomainGroupPointerBody(response *iotsdk.IotDomainGroup) (iotsdk.IotDomainGroup, bool) {
	if response == nil {
		return iotsdk.IotDomainGroup{}, false
	}
	return *response, true
}

func iotDomainGroupSummaryPointerBody(response *iotsdk.IotDomainGroupSummary) (iotsdk.IotDomainGroup, bool) {
	if response == nil {
		return iotsdk.IotDomainGroup{}, false
	}
	return iotDomainGroupFromSummary(*response), true
}

func iotDomainGroupFromSummary(summary iotsdk.IotDomainGroupSummary) iotsdk.IotDomainGroup {
	return iotsdk.IotDomainGroup{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		Type:           iotsdk.IotDomainGroupTypeEnum(summary.Type),
		DisplayName:    summary.DisplayName,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
		Description:    summary.Description,
		FreeformTags:   cloneIotDomainGroupStringMap(summary.FreeformTags),
		DefinedTags:    cloneIotDomainGroupDefinedTags(summary.DefinedTags),
		SystemTags:     cloneIotDomainGroupDefinedTags(summary.SystemTags),
		TimeUpdated:    summary.TimeUpdated,
	}
}

func listIotDomainGroupsAllPages(
	call func(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error),
) func(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
	return func(ctx context.Context, request iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
		var combined iotsdk.ListIotDomainGroupsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return iotsdk.ListIotDomainGroupsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleIotDomainGroupDeleteError(resource *iotv1beta1.IotDomainGroup, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return iotDomainGroupAmbiguousNotFoundError{
		message:      "IotDomainGroup delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapIotDomainGroupDeleteConfirmation(hooks *IotDomainGroupRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getIotDomainGroup := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate IotDomainGroupServiceClient) IotDomainGroupServiceClient {
		return iotDomainGroupDeleteConfirmationClient{
			delegate:          delegate,
			getIotDomainGroup: getIotDomainGroup,
		}
	})
}

type iotDomainGroupDeleteConfirmationClient struct {
	delegate          IotDomainGroupServiceClient
	getIotDomainGroup func(context.Context, iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error)
}

func (c iotDomainGroupDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *iotv1beta1.IotDomainGroup,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c iotDomainGroupDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *iotv1beta1.IotDomainGroup,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c iotDomainGroupDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *iotv1beta1.IotDomainGroup,
) error {
	if c.getIotDomainGroup == nil || resource == nil {
		return nil
	}
	groupID := trackedIotDomainGroupID(resource)
	if groupID == "" {
		return nil
	}
	_, err := c.getIotDomainGroup(ctx, iotsdk.GetIotDomainGroupRequest{IotDomainGroupId: common.String(groupID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("IotDomainGroup delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedIotDomainGroupID(resource *iotv1beta1.IotDomainGroup) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func applyIotDomainGroupDeleteOutcome(
	resource *iotv1beta1.IotDomainGroup,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := strings.ToUpper(iotDomainGroupLifecycleState(response))
	if iotDomainGroupDeleteConfirmedState(state) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if iotDomainGroupShouldGuardPendingWriteDelete(resource, state, stage) {
		markIotDomainGroupPendingWriteDeleteGuard(resource, response, state)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	if !iotDomainGroupShouldMarkTerminating(resource, state, stage) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markIotDomainGroupTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func iotDomainGroupDeleteConfirmedState(state string) bool {
	return state == "" || state == string(iotsdk.IotDomainGroupLifecycleStateDeleted)
}

func iotDomainGroupShouldGuardPendingWriteDelete(
	resource *iotv1beta1.IotDomainGroup,
	state string,
	stage generatedruntime.DeleteConfirmStage,
) bool {
	return stage == generatedruntime.DeleteConfirmStageAlreadyPending &&
		iotDomainGroupPendingWriteState(state) &&
		!iotDomainGroupDeleteAlreadyPending(resource)
}

func iotDomainGroupShouldMarkTerminating(
	resource *iotv1beta1.IotDomainGroup,
	state string,
	stage generatedruntime.DeleteConfirmStage,
) bool {
	if !iotDomainGroupRetainFinalizerState(state) {
		return false
	}
	return stage != generatedruntime.DeleteConfirmStageAlreadyPending ||
		iotDomainGroupDeleteAlreadyPending(resource)
}

func iotDomainGroupRetainFinalizerState(state string) bool {
	return state == string(iotsdk.IotDomainGroupLifecycleStateActive) ||
		state == string(iotsdk.IotDomainGroupLifecycleStateCreating) ||
		state == string(iotsdk.IotDomainGroupLifecycleStateUpdating)
}

func iotDomainGroupDeleteAlreadyPending(resource *iotv1beta1.IotDomainGroup) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func iotDomainGroupPendingWriteState(state string) bool {
	return state == string(iotsdk.IotDomainGroupLifecycleStateCreating) ||
		state == string(iotsdk.IotDomainGroupLifecycleStateUpdating)
}

func markIotDomainGroupPendingWriteDeleteGuard(
	resource *iotv1beta1.IotDomainGroup,
	response any,
	state string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	phase := shared.OSOKAsyncPhaseUpdate
	if state == string(iotsdk.IotDomainGroupLifecycleStateCreating) {
		phase = shared.OSOKAsyncPhaseCreate
	}
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = iotDomainGroupPendingWriteDeleteMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       iotDomainGroupLifecycleState(response),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         iotDomainGroupPendingWriteDeleteMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		iotDomainGroupPendingWriteDeleteMessage,
		loggerutil.OSOKLogger{},
	)
}

func markIotDomainGroupTerminating(resource *iotv1beta1.IotDomainGroup, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = iotDomainGroupDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := iotDomainGroupLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         iotDomainGroupDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		iotDomainGroupDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func iotDomainGroupLifecycleState(response any) string {
	current, ok := iotDomainGroupBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneIotDomainGroupStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneIotDomainGroupDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		clone[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			clone[namespace][key] = value
		}
	}
	return clone
}

func iotDomainGroupDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}
