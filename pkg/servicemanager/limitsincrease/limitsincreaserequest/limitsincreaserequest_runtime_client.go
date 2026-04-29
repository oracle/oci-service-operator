/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package limitsincreaserequest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	limitsincreasesdk "github.com/oracle/oci-go-sdk/v65/limitsincrease"
	limitsincreasev1beta1 "github.com/oracle/oci-service-operator/api/limitsincrease/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const limitsIncreaseRequestDeletePendingMessage = "OCI LimitsIncreaseRequest delete is in progress"
const limitsIncreaseRequestPendingWriteDeleteMessage = "OCI LimitsIncreaseRequest create or update is still in progress; retaining finalizer"

type limitsIncreaseRequestListFunc func(context.Context, limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error)
type limitsIncreaseRequestGetFunc func(context.Context, limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error)

func init() {
	registerLimitsIncreaseRequestRuntimeHooksMutator(func(_ *LimitsIncreaseRequestServiceManager, hooks *LimitsIncreaseRequestRuntimeHooks) {
		applyLimitsIncreaseRequestRuntimeHooks(hooks)
	})
}

func applyLimitsIncreaseRequestRuntimeHooks(hooks *LimitsIncreaseRequestRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newLimitsIncreaseRequestRuntimeSemantics()
	hooks.BuildUpdateBody = buildLimitsIncreaseRequestUpdateBody
	hooks.List.Fields = limitsIncreaseRequestListFields()
	hooks.List.Call = paginatedLimitsIncreaseRequestListCall(hooks.List.Call)
	hooks.Create.Call = normalizeLimitsIncreaseRequestCreateCall(hooks.Create.Call)
	hooks.Get.Call = normalizeLimitsIncreaseRequestGetCall(hooks.Get.Call)
	hooks.Update.Call = normalizeLimitsIncreaseRequestUpdateCall(hooks.Update.Call)
	hooks.ParityHooks.NormalizeDesiredState = normalizeLimitsIncreaseRequestDesiredState
	hooks.DeleteHooks.HandleError = handleLimitsIncreaseRequestDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyLimitsIncreaseRequestDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapLimitsIncreaseRequestDeleteGuard(hooks.Get.Call))
}

func newLimitsIncreaseRequestRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "limitsincrease",
		FormalSlug:    "limitsincreaserequest",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"ACCEPTED", "IN_PROGRESS"},
			ActiveStates:       []string{"SUCCEEDED", "PARTIALLY_SUCCEEDED"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "justification", "subscriptionId", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "freeformTags"},
			Mutable:         []string{"definedTags", "freeformTags"},
			ForceNew: []string{
				"compartmentId",
				"displayName",
				"justification",
				"limitsIncreaseItemRequests",
				"subscriptionId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func limitsIncreaseRequestListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func paginatedLimitsIncreaseRequestListCall(call limitsIncreaseRequestListFunc) limitsIncreaseRequestListFunc {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error) {
		var combined limitsincreasesdk.ListLimitsIncreaseRequestsResponse
		seenPages := map[string]bool{}
		nextPage := request.Page
		for {
			response, err := fetchLimitsIncreaseRequestListPage(ctx, call, request, nextPage, seenPages)
			if err != nil {
				return combined, err
			}
			mergeLimitsIncreaseRequestListPage(&combined, response)
			if stringPtrValue(response.OpcNextPage) == "" {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func fetchLimitsIncreaseRequestListPage(
	ctx context.Context,
	call limitsIncreaseRequestListFunc,
	request limitsincreasesdk.ListLimitsIncreaseRequestsRequest,
	nextPage *string,
	seenPages map[string]bool,
) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error) {
	pageRequest := request
	pageRequest.Page = nextPage
	if err := recordLimitsIncreaseRequestListPage(pageRequest.Page, seenPages); err != nil {
		return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{}, err
	}
	return call(ctx, pageRequest)
}

func recordLimitsIncreaseRequestListPage(pageToken *string, seenPages map[string]bool) error {
	page := stringPtrValue(pageToken)
	if page == "" {
		return nil
	}
	if seenPages[page] {
		return fmt.Errorf("LimitsIncreaseRequest list pagination repeated page token %q", page)
	}
	seenPages[page] = true
	return nil
}

func mergeLimitsIncreaseRequestListPage(combined *limitsincreasesdk.ListLimitsIncreaseRequestsResponse, response limitsincreasesdk.ListLimitsIncreaseRequestsResponse) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func normalizeLimitsIncreaseRequestCreateCall(
	call func(context.Context, limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error),
) func(context.Context, limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		response.LimitsIncreaseRequest = normalizeLimitsIncreaseRequestModel(response.LimitsIncreaseRequest)
		return response, nil
	}
}

func normalizeLimitsIncreaseRequestGetCall(
	call func(context.Context, limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error),
) func(context.Context, limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		response.LimitsIncreaseRequest = normalizeLimitsIncreaseRequestModel(response.LimitsIncreaseRequest)
		return response, nil
	}
}

func normalizeLimitsIncreaseRequestUpdateCall(
	call func(context.Context, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error),
) func(context.Context, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		response.LimitsIncreaseRequest = normalizeLimitsIncreaseRequestModel(response.LimitsIncreaseRequest)
		return response, nil
	}
}

func normalizeLimitsIncreaseRequestDesiredState(resource *limitsincreasev1beta1.LimitsIncreaseRequest, _ any) {
	if resource == nil {
		return
	}
	sort.SliceStable(resource.Spec.LimitsIncreaseItemRequests, func(i, j int) bool {
		return limitsIncreaseRequestSpecItemKey(resource.Spec.LimitsIncreaseItemRequests[i]) <
			limitsIncreaseRequestSpecItemKey(resource.Spec.LimitsIncreaseItemRequests[j])
	})
	for i := range resource.Spec.LimitsIncreaseItemRequests {
		sort.SliceStable(resource.Spec.LimitsIncreaseItemRequests[i].QuestionnaireResponse, func(j, k int) bool {
			return limitsIncreaseRequestSpecQuestionKey(resource.Spec.LimitsIncreaseItemRequests[i].QuestionnaireResponse[j]) <
				limitsIncreaseRequestSpecQuestionKey(resource.Spec.LimitsIncreaseItemRequests[i].QuestionnaireResponse[k])
		})
	}
}

func normalizeLimitsIncreaseRequestModel(model limitsincreasesdk.LimitsIncreaseRequest) limitsincreasesdk.LimitsIncreaseRequest {
	model.LimitsIncreaseItemRequests = normalizeLimitsIncreaseRequestItems(model.LimitsIncreaseItemRequests)
	return model
}

func buildLimitsIncreaseRequestUpdateBody(
	_ context.Context,
	resource *limitsincreasev1beta1.LimitsIncreaseRequest,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return limitsincreasesdk.UpdateLimitsIncreaseRequestDetails{}, false, fmt.Errorf("LimitsIncreaseRequest resource is nil")
	}
	current, err := limitsIncreaseRequestFromResponse(currentResponse)
	if err != nil {
		return limitsincreasesdk.UpdateLimitsIncreaseRequestDetails{}, false, err
	}

	details := limitsincreasesdk.UpdateLimitsIncreaseRequestDetails{}
	updateNeeded := false
	if desired, ok := limitsIncreaseRequestDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := limitsIncreaseRequestDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if !updateNeeded {
		return limitsincreasesdk.UpdateLimitsIncreaseRequestDetails{}, false, nil
	}
	return details, true, nil
}

func limitsIncreaseRequestFromResponse(currentResponse any) (limitsincreasesdk.LimitsIncreaseRequest, error) {
	if current, ok, err := limitsIncreaseRequestFromModelResponse(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := limitsIncreaseRequestFromOperationResponse(currentResponse); ok || err != nil {
		return current, err
	}
	return limitsincreasesdk.LimitsIncreaseRequest{}, fmt.Errorf("unexpected current LimitsIncreaseRequest response type %T", currentResponse)
}

func limitsIncreaseRequestFromModelResponse(currentResponse any) (limitsincreasesdk.LimitsIncreaseRequest, bool, error) {
	switch current := currentResponse.(type) {
	case limitsincreasesdk.LimitsIncreaseRequest:
		return current, true, nil
	case *limitsincreasesdk.LimitsIncreaseRequest:
		if current == nil {
			return limitsincreasesdk.LimitsIncreaseRequest{}, true, fmt.Errorf("current LimitsIncreaseRequest response is nil")
		}
		return *current, true, nil
	case limitsincreasesdk.LimitsIncreaseRequestSummary:
		return limitsIncreaseRequestFromSummary(current), true, nil
	case *limitsincreasesdk.LimitsIncreaseRequestSummary:
		if current == nil {
			return limitsincreasesdk.LimitsIncreaseRequest{}, true, fmt.Errorf("current LimitsIncreaseRequest response is nil")
		}
		return limitsIncreaseRequestFromSummary(*current), true, nil
	default:
		return limitsincreasesdk.LimitsIncreaseRequest{}, false, nil
	}
}

func limitsIncreaseRequestFromOperationResponse(currentResponse any) (limitsincreasesdk.LimitsIncreaseRequest, bool, error) {
	switch current := currentResponse.(type) {
	case limitsincreasesdk.CreateLimitsIncreaseRequestResponse:
		return current.LimitsIncreaseRequest, true, nil
	case *limitsincreasesdk.CreateLimitsIncreaseRequestResponse:
		if current == nil {
			return limitsincreasesdk.LimitsIncreaseRequest{}, true, fmt.Errorf("current LimitsIncreaseRequest response is nil")
		}
		return current.LimitsIncreaseRequest, true, nil
	case limitsincreasesdk.GetLimitsIncreaseRequestResponse:
		return current.LimitsIncreaseRequest, true, nil
	case *limitsincreasesdk.GetLimitsIncreaseRequestResponse:
		if current == nil {
			return limitsincreasesdk.LimitsIncreaseRequest{}, true, fmt.Errorf("current LimitsIncreaseRequest response is nil")
		}
		return current.LimitsIncreaseRequest, true, nil
	case limitsincreasesdk.UpdateLimitsIncreaseRequestResponse:
		return current.LimitsIncreaseRequest, true, nil
	case *limitsincreasesdk.UpdateLimitsIncreaseRequestResponse:
		if current == nil {
			return limitsincreasesdk.LimitsIncreaseRequest{}, true, fmt.Errorf("current LimitsIncreaseRequest response is nil")
		}
		return current.LimitsIncreaseRequest, true, nil
	default:
		return limitsincreasesdk.LimitsIncreaseRequest{}, false, nil
	}
}

func limitsIncreaseRequestFromSummary(summary limitsincreasesdk.LimitsIncreaseRequestSummary) limitsincreasesdk.LimitsIncreaseRequest {
	return limitsincreasesdk.LimitsIncreaseRequest{
		Id:             summary.Id,
		DisplayName:    summary.DisplayName,
		CompartmentId:  summary.CompartmentId,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
		SubscriptionId: summary.SubscriptionId,
		Justification:  summary.Justification,
		SystemTags:     summary.SystemTags,
	}
}

func limitsIncreaseRequestDesiredFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
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

func limitsIncreaseRequestDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := limitsIncreaseRequestDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if limitsIncreaseRequestJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func limitsIncreaseRequestDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func limitsIncreaseRequestJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func normalizeLimitsIncreaseRequestItems(items []limitsincreasesdk.LimitsIncreaseItemRequest) []limitsincreasesdk.LimitsIncreaseItemRequest {
	normalized := make([]limitsincreasesdk.LimitsIncreaseItemRequest, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, limitsincreasesdk.LimitsIncreaseItemRequest{
			ServiceName:           item.ServiceName,
			LimitName:             item.LimitName,
			Region:                item.Region,
			Value:                 item.Value,
			Scope:                 item.Scope,
			QuestionnaireResponse: normalizeLimitsIncreaseRequestQuestions(item.QuestionnaireResponse),
		})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return limitsIncreaseRequestSDKItemKey(normalized[i]) < limitsIncreaseRequestSDKItemKey(normalized[j])
	})
	return normalized
}

func normalizeLimitsIncreaseRequestQuestions(questions []limitsincreasesdk.LimitsIncreaseItemQuestionResponse) []limitsincreasesdk.LimitsIncreaseItemQuestionResponse {
	normalized := make([]limitsincreasesdk.LimitsIncreaseItemQuestionResponse, 0, len(questions))
	for _, question := range questions {
		normalized = append(normalized, limitsincreasesdk.LimitsIncreaseItemQuestionResponse{
			Id:               question.Id,
			QuestionResponse: question.QuestionResponse,
		})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return limitsIncreaseRequestSDKQuestionKey(normalized[i]) < limitsIncreaseRequestSDKQuestionKey(normalized[j])
	})
	return normalized
}

func handleLimitsIncreaseRequestDeleteError(resource *limitsincreasev1beta1.LimitsIncreaseRequest, err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}

	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("LimitsIncreaseRequest delete returned ambiguous %s %s; retaining finalizer",
		classification.HTTPStatusCodeString(), classification.ErrorCodeString())
}

type limitsIncreaseRequestDeleteGuardClient struct {
	delegate                 LimitsIncreaseRequestServiceClient
	getLimitsIncreaseRequest limitsIncreaseRequestGetFunc
}

func wrapLimitsIncreaseRequestDeleteGuard(getLimitsIncreaseRequest limitsIncreaseRequestGetFunc) func(LimitsIncreaseRequestServiceClient) LimitsIncreaseRequestServiceClient {
	return func(delegate LimitsIncreaseRequestServiceClient) LimitsIncreaseRequestServiceClient {
		return limitsIncreaseRequestDeleteGuardClient{
			delegate:                 delegate,
			getLimitsIncreaseRequest: getLimitsIncreaseRequest,
		}
	}
}

func (c limitsIncreaseRequestDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *limitsincreasev1beta1.LimitsIncreaseRequest,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c limitsIncreaseRequestDeleteGuardClient) Delete(
	ctx context.Context,
	resource *limitsincreasev1beta1.LimitsIncreaseRequest,
) (bool, error) {
	if limitsIncreaseRequestWriteAlreadyPending(resource) {
		markLimitsIncreaseRequestWritePendingDeleteGuard(resource)
		return false, nil
	}
	if err := c.rejectAlreadyPendingAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c limitsIncreaseRequestDeleteGuardClient) rejectAlreadyPendingAuthShapedConfirmRead(
	ctx context.Context,
	resource *limitsincreasev1beta1.LimitsIncreaseRequest,
) error {
	if !limitsIncreaseRequestDeleteAlreadyPending(resource) || c.getLimitsIncreaseRequest == nil {
		return nil
	}
	currentID := limitsIncreaseRequestTrackedID(resource)
	if currentID == "" {
		return nil
	}
	_, err := c.getLimitsIncreaseRequest(ctx, limitsincreasesdk.GetLimitsIncreaseRequestRequest{
		LimitsIncreaseRequestId: common.String(currentID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("LimitsIncreaseRequest delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound while delete was already pending; retaining finalizer")
}

func applyLimitsIncreaseRequestDeleteOutcome(
	resource *limitsincreasev1beta1.LimitsIncreaseRequest,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch stage {
	case generatedruntime.DeleteConfirmStageAlreadyPending:
		if !limitsIncreaseRequestDeleteAlreadyPending(resource) {
			return generatedruntime.DeleteOutcome{}, nil
		}
	case generatedruntime.DeleteConfirmStageAfterRequest:
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
	if response == nil {
		return generatedruntime.DeleteOutcome{}, nil
	}

	markLimitsIncreaseRequestTerminating(resource, limitsIncreaseRequestDeletePendingMessage)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func limitsIncreaseRequestDeleteAlreadyPending(resource *limitsincreasev1beta1.LimitsIncreaseRequest) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func limitsIncreaseRequestWriteAlreadyPending(resource *limitsincreasev1beta1.LimitsIncreaseRequest) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func markLimitsIncreaseRequestWritePendingDeleteGuard(resource *limitsincreasev1beta1.LimitsIncreaseRequest) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = limitsIncreaseRequestPendingWriteDeleteMessage
	if status.Async.Current != nil && status.Async.Current.UpdatedAt == nil {
		status.Async.Current.UpdatedAt = &now
	}
}

func markLimitsIncreaseRequestTerminating(resource *limitsincreasev1beta1.LimitsIncreaseRequest, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       strings.TrimSpace(resource.Status.LifecycleState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func limitsIncreaseRequestTrackedID(resource *limitsincreasev1beta1.LimitsIncreaseRequest) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func limitsIncreaseRequestSpecItemKey(item limitsincreasev1beta1.LimitsIncreaseRequestLimitsIncreaseItemRequest) string {
	return strings.Join([]string{
		item.ServiceName,
		item.LimitName,
		item.Region,
		item.Scope,
		fmt.Sprintf("%020d", item.Value),
	}, "\x00")
}

func limitsIncreaseRequestSDKItemKey(item limitsincreasesdk.LimitsIncreaseItemRequest) string {
	return strings.Join([]string{
		stringPtrValue(item.ServiceName),
		stringPtrValue(item.LimitName),
		stringPtrValue(item.Region),
		stringPtrValue(item.Scope),
		fmt.Sprintf("%020d", int64PtrValue(item.Value)),
	}, "\x00")
}

func limitsIncreaseRequestSpecQuestionKey(question limitsincreasev1beta1.LimitsIncreaseRequestLimitsIncreaseItemRequestQuestionnaireResponse) string {
	return strings.Join([]string{question.Id, question.QuestionResponse}, "\x00")
}

func limitsIncreaseRequestSDKQuestionKey(question limitsincreasesdk.LimitsIncreaseItemQuestionResponse) string {
	return strings.Join([]string{stringPtrValue(question.Id), stringPtrValue(question.QuestionResponse)}, "\x00")
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func int64PtrValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
