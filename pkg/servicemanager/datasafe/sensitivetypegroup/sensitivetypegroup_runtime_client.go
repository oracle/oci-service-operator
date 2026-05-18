/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetypegroup

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
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

const sensitiveTypeGroupKind = "SensitiveTypeGroup"

var sensitiveTypeGroupWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(datasafesdk.WorkRequestStatusAccepted),
		string(datasafesdk.WorkRequestStatusInProgress),
		string(datasafesdk.WorkRequestStatusCanceling),
		string(datasafesdk.WorkRequestStatusSuspending),
	},
	SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(datasafesdk.WorkRequestStatusSuspended)},
	CreateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeCreateSensitiveTypeGroup),
		string(datasafesdk.WorkRequestResourceActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeUpdateSensitiveTypeGroup),
		string(datasafesdk.WorkRequestResourceActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeDeleteSensitiveTypeGroup),
		string(datasafesdk.WorkRequestResourceActionTypeDeleted),
	},
}

type sensitiveTypeGroupWorkRequestClient interface {
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type sensitiveTypeGroupIdentity struct {
	compartmentID string
	displayName   string
}

type sensitiveTypeGroupRuntimeClient struct {
	delegate SensitiveTypeGroupServiceClient
	get      func(context.Context, datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error)
}

type sensitiveTypeGroupMutationRecorder struct {
	phase         shared.OSOKAsyncPhase
	workRequestID string
	opcRequestID  string
}

type sensitiveTypeGroupMutationRecorderKey struct{}

type sensitiveTypeGroupAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e sensitiveTypeGroupAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e sensitiveTypeGroupAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSensitiveTypeGroupRuntimeHooksMutator(func(manager *SensitiveTypeGroupServiceManager, hooks *SensitiveTypeGroupRuntimeHooks) {
		client, initErr := newSensitiveTypeGroupWorkRequestClient(manager)
		applySensitiveTypeGroupRuntimeHooks(hooks, client, initErr)
	})
}

func newSensitiveTypeGroupWorkRequestClient(manager *SensitiveTypeGroupServiceManager) (sensitiveTypeGroupWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", sensitiveTypeGroupKind)
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySensitiveTypeGroupRuntimeHooks(
	hooks *SensitiveTypeGroupRuntimeHooks,
	client sensitiveTypeGroupWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = sensitiveTypeGroupRuntimeSemantics()
	hooks.BuildCreateBody = buildSensitiveTypeGroupCreateBody
	hooks.BuildUpdateBody = buildSensitiveTypeGroupUpdateBody
	hooks.Identity.Resolve = resolveSensitiveTypeGroupIdentity
	hooks.Identity.RecordPath = recordSensitiveTypeGroupPathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardSensitiveTypeGroupExistingBeforeCreate
	hooks.Create.Fields = sensitiveTypeGroupCreateFields()
	hooks.Get.Fields = sensitiveTypeGroupGetFields()
	hooks.List.Fields = sensitiveTypeGroupListFields()
	hooks.Update.Fields = sensitiveTypeGroupUpdateFields()
	hooks.Delete.Fields = sensitiveTypeGroupDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listSensitiveTypeGroupsAllPages(hooks.List.Call)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedSensitiveTypeGroupIdentity
	hooks.StatusHooks.ProjectStatus = projectSensitiveTypeGroupStatus
	hooks.StatusHooks.MarkTerminating = markSensitiveTypeGroupTerminating
	hooks.DeleteHooks.HandleError = handleSensitiveTypeGroupDeleteError
	hooks.Async.Adapter = sensitiveTypeGroupWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getSensitiveTypeGroupWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveSensitiveTypeGroupWorkRequestAction
	hooks.Async.RecoverResourceID = recoverSensitiveTypeGroupIDFromWorkRequest
	hooks.Async.Message = sensitiveTypeGroupWorkRequestMessage
	wrapSensitiveTypeGroupMutationRecorders(hooks)
	wrapSensitiveTypeGroupDeletePreRead(hooks)
}

func sensitiveTypeGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "sensitivetypegroup",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.SensitiveTypeGroupLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.SensitiveTypeGroupLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.SensitiveTypeGroupLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.SensitiveTypeGroupLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.SensitiveTypeGroupLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: sensitiveTypeGroupKind, Action: "CreateSensitiveTypeGroup"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: sensitiveTypeGroupKind, Action: "UpdateSensitiveTypeGroup"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: sensitiveTypeGroupKind, Action: "DeleteSensitiveTypeGroup"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetSensitiveTypeGroup"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetSensitiveTypeGroup"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func sensitiveTypeGroupCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSensitiveTypeGroupDetails", RequestName: "CreateSensitiveTypeGroupDetails", Contribution: "body"},
	}
}

func sensitiveTypeGroupGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SensitiveTypeGroupId", RequestName: "sensitiveTypeGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func sensitiveTypeGroupListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "SensitiveTypeGroupId", RequestName: "sensitiveTypeGroupId", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
	}
}

func sensitiveTypeGroupUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SensitiveTypeGroupId", RequestName: "sensitiveTypeGroupId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSensitiveTypeGroupDetails", RequestName: "UpdateSensitiveTypeGroupDetails", Contribution: "body"},
	}
}

func sensitiveTypeGroupDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SensitiveTypeGroupId", RequestName: "sensitiveTypeGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func buildSensitiveTypeGroupCreateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveTypeGroup,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", sensitiveTypeGroupKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("%s requires spec.compartmentId", sensitiveTypeGroupKind)
	}

	details := datasafesdk.CreateSensitiveTypeGroupDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   sensitiveTypeGroupOptionalString(resource.Spec.DisplayName),
		Description:   sensitiveTypeGroupOptionalString(resource.Spec.Description),
		FreeformTags:  sensitiveTypeGroupStringMap(resource.Spec.FreeformTags),
		DefinedTags:   sensitiveTypeGroupDefinedTags(resource.Spec.DefinedTags),
	}
	return details, nil
}

func buildSensitiveTypeGroupUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveTypeGroup,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", sensitiveTypeGroupKind)
	}
	current, ok := sensitiveTypeGroupStatusProjectionFromResponse(currentResponse)
	if !ok {
		current = sensitiveTypeGroupStatusProjectionFromResource(resource)
	}

	details := datasafesdk.UpdateSensitiveTypeGroupDetails{}
	updateNeeded := false
	updateNeeded = applySensitiveTypeGroupStringUpdates(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applySensitiveTypeGroupTagUpdates(&details, resource.Spec, current) || updateNeeded
	if !updateNeeded {
		return datasafesdk.UpdateSensitiveTypeGroupDetails{}, false, nil
	}
	return details, true, nil
}

func applySensitiveTypeGroupStringUpdates(
	details *datasafesdk.UpdateSensitiveTypeGroupDetails,
	spec datasafev1beta1.SensitiveTypeGroupSpec,
	current sensitiveTypeGroupStatusProjection,
) bool {
	updateNeeded := false
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" && displayName != current.DisplayName {
		details.DisplayName = common.String(displayName)
		updateNeeded = true
	}
	if description := strings.TrimSpace(spec.Description); description != "" && description != current.Description {
		details.Description = common.String(description)
		updateNeeded = true
	}
	return updateNeeded
}

func applySensitiveTypeGroupTagUpdates(
	details *datasafesdk.UpdateSensitiveTypeGroupDetails,
	spec datasafev1beta1.SensitiveTypeGroupSpec,
	current sensitiveTypeGroupStatusProjection,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = sensitiveTypeGroupStringMap(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil && !reflect.DeepEqual(sensitiveTypeGroupDefinedTags(spec.DefinedTags), sensitiveTypeGroupDefinedTags(current.DefinedTags)) {
		details.DefinedTags = sensitiveTypeGroupDefinedTags(spec.DefinedTags)
		updateNeeded = true
	}
	return updateNeeded
}

func resolveSensitiveTypeGroupIdentity(resource *datasafev1beta1.SensitiveTypeGroup) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", sensitiveTypeGroupKind)
	}
	return sensitiveTypeGroupIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}, nil
}

func recordSensitiveTypeGroupPathIdentity(resource *datasafev1beta1.SensitiveTypeGroup, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(sensitiveTypeGroupIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if typed.displayName != "" {
		resource.Status.DisplayName = typed.displayName
	}
}

func guardSensitiveTypeGroupExistingBeforeCreate(_ context.Context, resource *datasafev1beta1.SensitiveTypeGroup) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", sensitiveTypeGroupKind)
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func listSensitiveTypeGroupsAllPages(
	call func(context.Context, datasafesdk.ListSensitiveTypeGroupsRequest) (datasafesdk.ListSensitiveTypeGroupsResponse, error),
) func(context.Context, datasafesdk.ListSensitiveTypeGroupsRequest) (datasafesdk.ListSensitiveTypeGroupsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListSensitiveTypeGroupsRequest) (datasafesdk.ListSensitiveTypeGroupsResponse, error) {
		var combined datasafesdk.ListSensitiveTypeGroupsResponse
		seenPages := map[string]struct{}{}
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			appendSensitiveTypeGroupListPage(&combined, response)
			nextPage, err := nextSensitiveTypeGroupPage(response, seenPages)
			if err != nil {
				return datasafesdk.ListSensitiveTypeGroupsResponse{}, err
			}
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func appendSensitiveTypeGroupListPage(
	combined *datasafesdk.ListSensitiveTypeGroupsResponse,
	response datasafesdk.ListSensitiveTypeGroupsResponse,
) {
	combined.RawResponse = response.RawResponse
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	if len(response.Items) > 0 {
		combined.Items = append(combined.Items, response.Items...)
	}
}

func nextSensitiveTypeGroupPage(
	response datasafesdk.ListSensitiveTypeGroupsResponse,
	seenPages map[string]struct{},
) (string, error) {
	nextPage := sensitiveTypeGroupStringValue(response.OpcNextPage)
	if nextPage == "" {
		return "", nil
	}
	if _, ok := seenPages[nextPage]; ok {
		return "", fmt.Errorf("%s list pagination repeated page token %q", sensitiveTypeGroupKind, nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nextPage, nil
}

func clearTrackedSensitiveTypeGroupIdentity(resource *datasafev1beta1.SensitiveTypeGroup) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = ""
}

func handleSensitiveTypeGroupDeleteError(resource *datasafev1beta1.SensitiveTypeGroup, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := sensitiveTypeGroupAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func wrapSensitiveTypeGroupMutationRecorders(hooks *SensitiveTypeGroupRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Update.Call != nil {
		update := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
			response, err := update(ctx, request)
			if err == nil {
				recordSensitiveTypeGroupMutation(ctx, shared.OSOKAsyncPhaseUpdate, response.OpcWorkRequestId, response.OpcRequestId)
			}
			return response, err
		}
	}
	if hooks.Delete.Call != nil {
		deleteCall := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteSensitiveTypeGroupRequest) (datasafesdk.DeleteSensitiveTypeGroupResponse, error) {
			response, err := deleteCall(ctx, request)
			if err == nil {
				recordSensitiveTypeGroupMutation(ctx, shared.OSOKAsyncPhaseDelete, response.OpcWorkRequestId, response.OpcRequestId)
			}
			return response, err
		}
	}
}

func recordSensitiveTypeGroupMutation(ctx context.Context, phase shared.OSOKAsyncPhase, workRequestID *string, opcRequestID *string) {
	recorder, _ := ctx.Value(sensitiveTypeGroupMutationRecorderKey{}).(*sensitiveTypeGroupMutationRecorder)
	if recorder == nil {
		return
	}
	if id := sensitiveTypeGroupStringValue(workRequestID); id != "" {
		recorder.phase = phase
		recorder.workRequestID = id
	}
	if requestID := sensitiveTypeGroupStringValue(opcRequestID); requestID != "" {
		recorder.opcRequestID = requestID
	}
}

func wrapSensitiveTypeGroupDeletePreRead(hooks *SensitiveTypeGroupRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getSensitiveTypeGroup := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SensitiveTypeGroupServiceClient) SensitiveTypeGroupServiceClient {
		return sensitiveTypeGroupRuntimeClient{
			delegate: delegate,
			get:      getSensitiveTypeGroup,
		}
	})
}

func (c sensitiveTypeGroupRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveTypeGroup,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", sensitiveTypeGroupKind)
	}
	pendingBefore := currentSensitiveTypeGroupWorkRequest(resource, shared.OSOKAsyncPhaseUpdate)
	recorder := &sensitiveTypeGroupMutationRecorder{}
	response, err := c.delegate.CreateOrUpdate(context.WithValue(ctx, sensitiveTypeGroupMutationRecorderKey{}, recorder), resource, req)
	if err != nil {
		return response, err
	}
	if guarded, handled := sensitiveTypeGroupStaleUpdateGuard(resource, response, pendingBefore, recorder); handled {
		return guarded, nil
	}
	return response, nil
}

func sensitiveTypeGroupStaleUpdateGuard(
	resource *datasafev1beta1.SensitiveTypeGroup,
	response servicemanager.OSOKResponse,
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveTypeGroupMutationRecorder,
) (servicemanager.OSOKResponse, bool) {
	if pendingBefore != nil && !response.ShouldRequeue && sensitiveTypeGroupDesiredUpdatePending(resource) {
		return markSensitiveTypeGroupWorkRequestPending(resource, pendingBefore.Phase, pendingBefore.WorkRequestID, "accepted update is not reflected by readback"), true
	}
	if recorder.phase == shared.OSOKAsyncPhaseUpdate && recorder.workRequestID != "" && !response.ShouldRequeue && sensitiveTypeGroupDesiredUpdatePending(resource) {
		if recorder.opcRequestID != "" {
			resource.Status.OsokStatus.OpcRequestID = recorder.opcRequestID
		}
		return markSensitiveTypeGroupWorkRequestPending(resource, recorder.phase, recorder.workRequestID, "accepted update is not reflected by readback"), true
	}
	return servicemanager.OSOKResponse{}, false
}

func (c sensitiveTypeGroupRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.SensitiveTypeGroup) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", sensitiveTypeGroupKind)
	}
	if currentSensitiveTypeGroupID(resource) == "" && sensitiveTypeGroupDisplayNameForDelete(resource) == "" {
		markSensitiveTypeGroupDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	return c.deleteWithSensitiveTypeGroupWorkRequestGuard(ctx, resource)
}

func (c sensitiveTypeGroupRuntimeClient) deleteWithSensitiveTypeGroupWorkRequestGuard(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveTypeGroup,
) (bool, error) {
	currentID := currentSensitiveTypeGroupID(resource)
	if currentID != "" {
		if err := c.guardSensitiveTypeGroupPreDeleteGet(ctx, resource, currentID); err != nil {
			return false, err
		}
	}
	pendingBefore := currentSensitiveTypeGroupWorkRequest(resource, shared.OSOKAsyncPhaseDelete)
	recorder := &sensitiveTypeGroupMutationRecorder{}
	deleted, err := c.delegate.Delete(context.WithValue(ctx, sensitiveTypeGroupMutationRecorderKey{}, recorder), resource)
	return sensitiveTypeGroupDeleteGuardResult(resource, pendingBefore, recorder, deleted, err)
}

func sensitiveTypeGroupDeleteGuardResult(
	resource *datasafev1beta1.SensitiveTypeGroup,
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveTypeGroupMutationRecorder,
	deleted bool,
	err error,
) (bool, error) {
	if err != nil {
		if workRequestID := sensitiveTypeGroupDeleteWorkRequestID(pendingBefore, recorder); workRequestID != "" && sensitiveTypeGroupDeleteReadbackStillActive(err) {
			markSensitiveTypeGroupWorkRequestPending(resource, shared.OSOKAsyncPhaseDelete, workRequestID, "accepted delete is not reflected by readback")
			return false, nil
		}
		return deleted, err
	}
	if !deleted && recorder.phase == shared.OSOKAsyncPhaseDelete && recorder.workRequestID != "" {
		if recorder.opcRequestID != "" {
			resource.Status.OsokStatus.OpcRequestID = recorder.opcRequestID
		}
		current := currentSensitiveTypeGroupWorkRequest(resource, recorder.phase)
		if current == nil || strings.TrimSpace(current.WorkRequestID) != recorder.workRequestID {
			markSensitiveTypeGroupWorkRequestPending(resource, recorder.phase, recorder.workRequestID, "accepted delete is not reflected by readback")
		}
	}
	return deleted, nil
}

func (c sensitiveTypeGroupRuntimeClient) guardSensitiveTypeGroupPreDeleteGet(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveTypeGroup,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	response, err := c.get(ctx, datasafesdk.GetSensitiveTypeGroupRequest{SensitiveTypeGroupId: common.String(currentID)})
	if ambiguous := sensitiveTypeGroupAmbiguousDeleteError(resource, err, "pre-delete get"); ambiguous != nil {
		return ambiguous
	}
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil
	}
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return fmt.Errorf("%s pre-delete get failed; refusing to call delete: %w", sensitiveTypeGroupKind, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return nil
}

func sensitiveTypeGroupAmbiguousDeleteError(
	resource *datasafev1beta1.SensitiveTypeGroup,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return sensitiveTypeGroupAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", sensitiveTypeGroupKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func projectSensitiveTypeGroupStatus(resource *datasafev1beta1.SensitiveTypeGroup, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", sensitiveTypeGroupKind)
	}
	projected, ok := sensitiveTypeGroupStatusProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.SensitiveTypeGroupStatus{
		OsokStatus:         osokStatus,
		Id:                 projected.Id,
		DisplayName:        projected.DisplayName,
		CompartmentId:      projected.CompartmentId,
		LifecycleState:     projected.LifecycleState,
		TimeCreated:        projected.TimeCreated,
		SensitiveTypeCount: projected.SensitiveTypeCount,
		Description:        projected.Description,
		TimeUpdated:        projected.TimeUpdated,
		FreeformTags:       sensitiveTypeGroupStringMap(projected.FreeformTags),
		DefinedTags:        sensitiveTypeGroupCloneSharedTags(projected.DefinedTags),
		SystemTags:         sensitiveTypeGroupCloneSharedTags(projected.SystemTags),
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

func markSensitiveTypeGroupTerminating(resource *datasafev1beta1.SensitiveTypeGroup, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = "OCI resource delete is in progress"
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       sensitiveTypeGroupLifecycleState(response),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         status.Message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, loggerutil.OSOKLogger{})
}

type sensitiveTypeGroupStatusProjection struct {
	Id                 string
	DisplayName        string
	CompartmentId      string
	LifecycleState     string
	TimeCreated        string
	SensitiveTypeCount int
	Description        string
	TimeUpdated        string
	FreeformTags       map[string]string
	DefinedTags        map[string]shared.MapValue
	SystemTags         map[string]shared.MapValue
}

func sensitiveTypeGroupStatusProjectionFromResponse(response any) (sensitiveTypeGroupStatusProjection, bool) {
	if current, ok := sensitiveTypeGroupFromResponse(response); ok {
		return sensitiveTypeGroupStatusProjectionFromSDK(current), true
	}
	if summary, ok := sensitiveTypeGroupSummaryFromResponse(response); ok {
		return sensitiveTypeGroupStatusProjectionFromSummary(summary), true
	}
	return sensitiveTypeGroupStatusProjection{}, false
}

func sensitiveTypeGroupLifecycleState(response any) string {
	projected, ok := sensitiveTypeGroupStatusProjectionFromResponse(response)
	if !ok {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(projected.LifecycleState))
}

func sensitiveTypeGroupStatusProjectionFromResource(resource *datasafev1beta1.SensitiveTypeGroup) sensitiveTypeGroupStatusProjection {
	if resource == nil {
		return sensitiveTypeGroupStatusProjection{}
	}
	return sensitiveTypeGroupStatusProjection{
		Id:                 strings.TrimSpace(resource.Status.Id),
		DisplayName:        strings.TrimSpace(resource.Status.DisplayName),
		CompartmentId:      strings.TrimSpace(resource.Status.CompartmentId),
		LifecycleState:     strings.TrimSpace(resource.Status.LifecycleState),
		TimeCreated:        resource.Status.TimeCreated,
		SensitiveTypeCount: resource.Status.SensitiveTypeCount,
		Description:        strings.TrimSpace(resource.Status.Description),
		TimeUpdated:        resource.Status.TimeUpdated,
		FreeformTags:       sensitiveTypeGroupStringMap(resource.Status.FreeformTags),
		DefinedTags:        sensitiveTypeGroupCloneSharedTags(resource.Status.DefinedTags),
		SystemTags:         sensitiveTypeGroupCloneSharedTags(resource.Status.SystemTags),
	}
}

func sensitiveTypeGroupFromResponse(response any) (datasafesdk.SensitiveTypeGroup, bool) {
	switch current := response.(type) {
	case datasafesdk.GetSensitiveTypeGroupResponse:
		return current.SensitiveTypeGroup, true
	case *datasafesdk.GetSensitiveTypeGroupResponse:
		if current == nil {
			return datasafesdk.SensitiveTypeGroup{}, false
		}
		return current.SensitiveTypeGroup, true
	case datasafesdk.CreateSensitiveTypeGroupResponse:
		return current.SensitiveTypeGroup, true
	case *datasafesdk.CreateSensitiveTypeGroupResponse:
		if current == nil {
			return datasafesdk.SensitiveTypeGroup{}, false
		}
		return current.SensitiveTypeGroup, true
	case datasafesdk.SensitiveTypeGroup:
		return current, true
	case *datasafesdk.SensitiveTypeGroup:
		if current == nil {
			return datasafesdk.SensitiveTypeGroup{}, false
		}
		return *current, true
	default:
		return datasafesdk.SensitiveTypeGroup{}, false
	}
}

func sensitiveTypeGroupSummaryFromResponse(response any) (datasafesdk.SensitiveTypeGroupSummary, bool) {
	switch current := response.(type) {
	case datasafesdk.SensitiveTypeGroupSummary:
		return current, true
	case *datasafesdk.SensitiveTypeGroupSummary:
		if current == nil {
			return datasafesdk.SensitiveTypeGroupSummary{}, false
		}
		return *current, true
	default:
		return datasafesdk.SensitiveTypeGroupSummary{}, false
	}
}

func sensitiveTypeGroupStatusProjectionFromSDK(current datasafesdk.SensitiveTypeGroup) sensitiveTypeGroupStatusProjection {
	return sensitiveTypeGroupStatusProjection{
		Id:                 sensitiveTypeGroupStringValue(current.Id),
		DisplayName:        sensitiveTypeGroupStringValue(current.DisplayName),
		CompartmentId:      sensitiveTypeGroupStringValue(current.CompartmentId),
		LifecycleState:     string(current.LifecycleState),
		TimeCreated:        sensitiveTypeGroupSDKTimeString(current.TimeCreated),
		SensitiveTypeCount: sensitiveTypeGroupIntValue(current.SensitiveTypeCount),
		Description:        sensitiveTypeGroupStringValue(current.Description),
		TimeUpdated:        sensitiveTypeGroupSDKTimeString(current.TimeUpdated),
		FreeformTags:       sensitiveTypeGroupStringMap(current.FreeformTags),
		DefinedTags:        sensitiveTypeGroupSharedTags(current.DefinedTags),
		SystemTags:         sensitiveTypeGroupSharedTags(current.SystemTags),
	}
}

func sensitiveTypeGroupStatusProjectionFromSummary(current datasafesdk.SensitiveTypeGroupSummary) sensitiveTypeGroupStatusProjection {
	return sensitiveTypeGroupStatusProjection{
		Id:                 sensitiveTypeGroupStringValue(current.Id),
		DisplayName:        sensitiveTypeGroupStringValue(current.DisplayName),
		CompartmentId:      sensitiveTypeGroupStringValue(current.CompartmentId),
		LifecycleState:     string(current.LifecycleState),
		TimeCreated:        sensitiveTypeGroupSDKTimeString(current.TimeCreated),
		SensitiveTypeCount: sensitiveTypeGroupIntValue(current.SensitiveTypeCount),
		Description:        sensitiveTypeGroupStringValue(current.Description),
		TimeUpdated:        sensitiveTypeGroupSDKTimeString(current.TimeUpdated),
		FreeformTags:       sensitiveTypeGroupStringMap(current.FreeformTags),
		DefinedTags:        sensitiveTypeGroupSharedTags(current.DefinedTags),
		SystemTags:         sensitiveTypeGroupSharedTags(current.SystemTags),
	}
}

func currentSensitiveTypeGroupID(resource *datasafev1beta1.SensitiveTypeGroup) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func sensitiveTypeGroupDisplayNameForDelete(resource *datasafev1beta1.SensitiveTypeGroup) string {
	if resource == nil {
		return ""
	}
	if displayName := strings.TrimSpace(resource.Status.DisplayName); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(resource.Spec.DisplayName)
}

func markSensitiveTypeGroupDeleted(resource *datasafev1beta1.SensitiveTypeGroup, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func markSensitiveTypeGroupWorkRequestPending(
	resource *datasafev1beta1.SensitiveTypeGroup,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	message string,
) servicemanager.OSOKResponse {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	current := &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            phase,
		WorkRequestID:    workRequestID,
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawOperationType: sensitiveTypeGroupWorkRequestOperationType(phase),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          sensitiveTypeGroupWorkRequestPendingMessage(phase, workRequestID, message),
		UpdatedAt:        &now,
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}
}

func currentSensitiveTypeGroupWorkRequest(
	resource *datasafev1beta1.SensitiveTypeGroup,
	phase shared.OSOKAsyncPhase,
) *shared.OSOKAsyncOperation {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return nil
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != "" && current.Source != shared.OSOKAsyncSourceWorkRequest {
		return nil
	}
	if current.Phase != phase || strings.TrimSpace(current.WorkRequestID) == "" {
		return nil
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassSucceeded, shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled:
		return nil
	default:
		return current.DeepCopy()
	}
}

func sensitiveTypeGroupDeleteWorkRequestID(
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveTypeGroupMutationRecorder,
) string {
	if recorder != nil && recorder.phase == shared.OSOKAsyncPhaseDelete {
		if workRequestID := strings.TrimSpace(recorder.workRequestID); workRequestID != "" {
			return workRequestID
		}
	}
	if pendingBefore != nil {
		return strings.TrimSpace(pendingBefore.WorkRequestID)
	}
	return ""
}

func sensitiveTypeGroupDeleteReadbackStillActive(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "delete confirmation returned unexpected lifecycle state") &&
		strings.Contains(message, string(datasafesdk.SensitiveTypeGroupLifecycleStateActive))
}

func sensitiveTypeGroupDesiredUpdatePending(resource *datasafev1beta1.SensitiveTypeGroup) bool {
	if resource == nil {
		return false
	}
	current := sensitiveTypeGroupStatusProjectionFromResource(resource)
	details := datasafesdk.UpdateSensitiveTypeGroupDetails{}
	return applySensitiveTypeGroupStringUpdates(&details, resource.Spec, current) ||
		applySensitiveTypeGroupTagUpdates(&details, resource.Spec, current)
}

func sensitiveTypeGroupWorkRequestPendingMessage(
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	reason string,
) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = fmt.Sprintf("waiting for %s readback to converge", phase)
	}
	return fmt.Sprintf("%s %s work request %s is accepted; %s", sensitiveTypeGroupKind, phase, strings.TrimSpace(workRequestID), reason)
}

func sensitiveTypeGroupWorkRequestOperationType(phase shared.OSOKAsyncPhase) string {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return string(datasafesdk.WorkRequestOperationTypeCreateSensitiveTypeGroup)
	case shared.OSOKAsyncPhaseUpdate:
		return string(datasafesdk.WorkRequestOperationTypeUpdateSensitiveTypeGroup)
	case shared.OSOKAsyncPhaseDelete:
		return string(datasafesdk.WorkRequestOperationTypeDeleteSensitiveTypeGroup)
	default:
		return ""
	}
}

func getSensitiveTypeGroupWorkRequest(
	ctx context.Context,
	client sensitiveTypeGroupWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI work request client: %w", sensitiveTypeGroupKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI work request client is not configured", sensitiveTypeGroupKind)
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveSensitiveTypeGroupWorkRequestAction(workRequest any) (string, error) {
	current, ok := sensitiveTypeGroupWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request response did not expose a Data Safe WorkRequest body", sensitiveTypeGroupKind)
	}
	return string(current.OperationType), nil
}

func recoverSensitiveTypeGroupIDFromWorkRequest(
	resource *datasafev1beta1.SensitiveTypeGroup,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if id := currentSensitiveTypeGroupID(resource); id != "" {
		return id, nil
	}
	current, ok := sensitiveTypeGroupWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request response did not expose a Data Safe WorkRequest body", sensitiveTypeGroupKind)
	}
	for _, impacted := range current.Resources {
		if !sensitiveTypeGroupWorkRequestActionMatchesPhase(impacted.ActionType, phase) {
			continue
		}
		identifier := sensitiveTypeGroupStringValue(impacted.Identifier)
		if identifier == "" {
			continue
		}
		entityType := strings.ToLower(sensitiveTypeGroupStringValue(impacted.EntityType))
		if entityType == "" || strings.Contains(entityType, "sensitivetypegroup") || strings.Contains(entityType, "sensitive_type_group") {
			return identifier, nil
		}
	}
	return "", nil
}

func sensitiveTypeGroupWorkRequestActionMatchesPhase(
	action datasafesdk.WorkRequestResourceActionTypeEnum,
	phase shared.OSOKAsyncPhase,
) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == datasafesdk.WorkRequestResourceActionTypeCreated ||
			action == datasafesdk.WorkRequestResourceActionTypeInProgress
	case shared.OSOKAsyncPhaseUpdate:
		return action == datasafesdk.WorkRequestResourceActionTypeUpdated ||
			action == datasafesdk.WorkRequestResourceActionTypeInProgress
	case shared.OSOKAsyncPhaseDelete:
		return action == datasafesdk.WorkRequestResourceActionTypeDeleted ||
			action == datasafesdk.WorkRequestResourceActionTypeInProgress
	default:
		return false
	}
}

func sensitiveTypeGroupWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, ok := sensitiveTypeGroupWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	workRequestID := strings.TrimSpace(sensitiveTypeGroupStringValue(current.Id))
	status := strings.TrimSpace(string(current.Status))
	if workRequestID == "" || status == "" {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", sensitiveTypeGroupKind, phase, workRequestID, status)
}

func sensitiveTypeGroupWorkRequestFromAny(value any) (datasafesdk.WorkRequest, bool) {
	switch current := value.(type) {
	case datasafesdk.GetWorkRequestResponse:
		return current.WorkRequest, sensitiveTypeGroupWorkRequestPresent(current.WorkRequest)
	case *datasafesdk.GetWorkRequestResponse:
		if current == nil {
			return datasafesdk.WorkRequest{}, false
		}
		return current.WorkRequest, sensitiveTypeGroupWorkRequestPresent(current.WorkRequest)
	case datasafesdk.WorkRequest:
		return current, sensitiveTypeGroupWorkRequestPresent(current)
	case *datasafesdk.WorkRequest:
		if current == nil {
			return datasafesdk.WorkRequest{}, false
		}
		return *current, sensitiveTypeGroupWorkRequestPresent(*current)
	default:
		return datasafesdk.WorkRequest{}, false
	}
}

func sensitiveTypeGroupWorkRequestPresent(workRequest datasafesdk.WorkRequest) bool {
	return sensitiveTypeGroupStringValue(workRequest.Id) != "" || workRequest.Status != "" || workRequest.OperationType != ""
}

func sensitiveTypeGroupOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func sensitiveTypeGroupStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func sensitiveTypeGroupIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func sensitiveTypeGroupSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func sensitiveTypeGroupStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	return maps.Clone(source)
}

func sensitiveTypeGroupSharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = fmt.Sprint(value)
		}
		converted[namespace] = children
	}
	return converted
}

func sensitiveTypeGroupCloneSharedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = value
		}
		cloned[namespace] = children
	}
	return cloned
}

func sensitiveTypeGroupDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

var _ interface{ GetOpcRequestID() string } = sensitiveTypeGroupAmbiguousNotFoundError{}
