/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package protectionpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
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

const (
	protectionPolicyDeletePendingMessage      = "OCI ProtectionPolicy delete is in progress"
	protectionPolicyPendingWriteDeleteMessage = "OCI ProtectionPolicy create or update is still in progress; retaining finalizer"
)

var protectionPolicyWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(recoverysdk.OperationStatusAccepted),
		string(recoverysdk.OperationStatusWaiting),
		string(recoverysdk.OperationStatusInProgress),
		string(recoverysdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(recoverysdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(recoverysdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(recoverysdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(recoverysdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(recoverysdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(recoverysdk.ActionTypeDeleted)},
}

type protectionPolicyOCIClient interface {
	CreateProtectionPolicy(context.Context, recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error)
	GetProtectionPolicy(context.Context, recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error)
	ListProtectionPolicies(context.Context, recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error)
	UpdateProtectionPolicy(context.Context, recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error)
	DeleteProtectionPolicy(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error)
	GetWorkRequest(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

type protectionPolicyListCall func(context.Context, recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error)

type protectionPolicyAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e protectionPolicyAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e protectionPolicyAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerProtectionPolicyRuntimeHooksMutator(func(manager *ProtectionPolicyServiceManager, hooks *ProtectionPolicyRuntimeHooks) {
		client, initErr := newProtectionPolicyOCIClient(manager)
		applyProtectionPolicyRuntimeHooks(hooks, client, initErr)
	})
}

func newProtectionPolicyOCIClient(manager *ProtectionPolicyServiceManager) (protectionPolicyOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("protectionPolicy service manager is nil")
	}
	client, err := recoverysdk.NewDatabaseRecoveryClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyProtectionPolicyRuntimeHooks(
	hooks *ProtectionPolicyRuntimeHooks,
	client protectionPolicyOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newProtectionPolicyRuntimeSemantics()
	hooks.BuildCreateBody = buildProtectionPolicyCreateBody
	hooks.BuildUpdateBody = buildProtectionPolicyUpdateBody
	hooks.List.Fields = protectionPolicyListFields()
	hooks.List.Call = paginatedProtectionPolicyListCall(hooks.List.Call)
	hooks.Get.Call = conservativeProtectionPolicyGetCall(hooks.Get.Call)
	hooks.Async.Adapter = protectionPolicyWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getProtectionPolicyWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveProtectionPolicyGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveProtectionPolicyGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverProtectionPolicyIDFromGeneratedWorkRequest
	hooks.Async.Message = protectionPolicyGeneratedWorkRequestMessage
	hooks.DeleteHooks.HandleError = handleProtectionPolicyDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyProtectionPolicyDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapProtectionPolicyDeleteGuard(client, initErr))
}

func newProtectionPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "recovery",
		FormalSlug:    "protectionpolicy",
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
			ProvisioningStates: []string{string(recoverysdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(recoverysdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(recoverysdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(recoverysdk.LifecycleStateDeleteScheduled),
				string(recoverysdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{string(recoverysdk.LifecycleStateDeleted)},
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
				"backupRetentionPeriodInDays",
				"policyLockedDateTime",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"mustEnforceCloudLocality",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ProtectionPolicy", Action: "CreateProtectionPolicy"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ProtectionPolicy", Action: "UpdateProtectionPolicy"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ProtectionPolicy", Action: "DeleteProtectionPolicy"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "ProtectionPolicy", Action: "CREATED"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "ProtectionPolicy", Action: "UPDATED"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "ProtectionPolicy", Action: "DELETED"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func protectionPolicyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "ProtectionPolicyId", RequestName: "protectionPolicyId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildProtectionPolicyCreateBody(
	_ context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	_ string,
) (any, error) {
	if resource == nil {
		return recoverysdk.CreateProtectionPolicyDetails{}, fmt.Errorf("ProtectionPolicy resource is nil")
	}

	details := recoverysdk.CreateProtectionPolicyDetails{
		DisplayName:                 stringPtr(resource.Spec.DisplayName),
		BackupRetentionPeriodInDays: intPtr(resource.Spec.BackupRetentionPeriodInDays),
		CompartmentId:               stringPtr(resource.Spec.CompartmentId),
		MustEnforceCloudLocality:    boolPtr(resource.Spec.MustEnforceCloudLocality),
		FreeformTags:                cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:                 protectionPolicyDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
	if strings.TrimSpace(resource.Spec.PolicyLockedDateTime) != "" {
		details.PolicyLockedDateTime = stringPtr(resource.Spec.PolicyLockedDateTime)
	}
	return details, nil
}

func buildProtectionPolicyUpdateBody(
	_ context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return recoverysdk.UpdateProtectionPolicyDetails{}, false, fmt.Errorf("ProtectionPolicy resource is nil")
	}
	current, err := protectionPolicyFromResponse(currentResponse)
	if err != nil {
		return recoverysdk.UpdateProtectionPolicyDetails{}, false, err
	}

	details := recoverysdk.UpdateProtectionPolicyDetails{}
	updateNeeded := setProtectionPolicyScalarUpdates(&details, resource, current)
	if desired, ok := desiredProtectionPolicyFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := desiredProtectionPolicyDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if !updateNeeded {
		return recoverysdk.UpdateProtectionPolicyDetails{}, false, nil
	}
	return details, true, nil
}

func setProtectionPolicyScalarUpdates(
	details *recoverysdk.UpdateProtectionPolicyDetails,
	resource *recoveryv1beta1.ProtectionPolicy,
	current recoverysdk.ProtectionPolicy,
) bool {
	updateNeeded := false
	if resource.Spec.DisplayName != stringPtrValue(current.DisplayName) {
		details.DisplayName = stringPtr(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.BackupRetentionPeriodInDays != intPtrValue(current.BackupRetentionPeriodInDays) {
		details.BackupRetentionPeriodInDays = intPtr(resource.Spec.BackupRetentionPeriodInDays)
		updateNeeded = true
	}
	if desired := strings.TrimSpace(resource.Spec.PolicyLockedDateTime); desired != "" && desired != stringPtrValue(current.PolicyLockedDateTime) {
		details.PolicyLockedDateTime = stringPtr(resource.Spec.PolicyLockedDateTime)
		updateNeeded = true
	}
	return updateNeeded
}

func paginatedProtectionPolicyListCall(call protectionPolicyListCall) protectionPolicyListCall {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
		request.Owner = recoverysdk.ListProtectionPoliciesOwnerCustomer
		var combined recoverysdk.ListProtectionPoliciesResponse
		seenPages := map[string]bool{}
		nextPage := request.Page
		for {
			response, err := fetchProtectionPolicyListPage(ctx, call, request, nextPage, seenPages)
			if err != nil {
				return combined, err
			}
			mergeProtectionPolicyListPage(&combined, response)
			if stringPtrValue(response.OpcNextPage) == "" {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func fetchProtectionPolicyListPage(
	ctx context.Context,
	call protectionPolicyListCall,
	request recoverysdk.ListProtectionPoliciesRequest,
	nextPage *string,
	seenPages map[string]bool,
) (recoverysdk.ListProtectionPoliciesResponse, error) {
	pageRequest := request
	pageRequest.Page = nextPage
	if err := recordProtectionPolicyListPage(pageRequest.Page, seenPages); err != nil {
		return recoverysdk.ListProtectionPoliciesResponse{}, err
	}
	return call(ctx, pageRequest)
}

func recordProtectionPolicyListPage(pageToken *string, seenPages map[string]bool) error {
	page := stringPtrValue(pageToken)
	if page == "" {
		return nil
	}
	if seenPages[page] {
		return fmt.Errorf("ProtectionPolicy list pagination repeated page token %q", page)
	}
	seenPages[page] = true
	return nil
}

func mergeProtectionPolicyListPage(
	combined *recoverysdk.ListProtectionPoliciesResponse,
	response recoverysdk.ListProtectionPoliciesResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.OpcNextPage = response.OpcNextPage
	combined.Items = append(combined.Items, response.Items...)
}

func protectionPolicyFromResponse(currentResponse any) (recoverysdk.ProtectionPolicy, error) {
	if current, ok, err := protectionPolicyFromModelResponse(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := protectionPolicyFromOperationResponse(currentResponse); ok || err != nil {
		return current, err
	}
	return recoverysdk.ProtectionPolicy{}, fmt.Errorf("unexpected current ProtectionPolicy response type %T", currentResponse)
}

func protectionPolicyFromModelResponse(currentResponse any) (recoverysdk.ProtectionPolicy, bool, error) {
	switch current := currentResponse.(type) {
	case recoverysdk.ProtectionPolicy:
		return current, true, nil
	case *recoverysdk.ProtectionPolicy:
		if current == nil {
			return recoverysdk.ProtectionPolicy{}, true, fmt.Errorf("current ProtectionPolicy response is nil")
		}
		return *current, true, nil
	case recoverysdk.ProtectionPolicySummary:
		return protectionPolicyFromSummary(current), true, nil
	case *recoverysdk.ProtectionPolicySummary:
		if current == nil {
			return recoverysdk.ProtectionPolicy{}, true, fmt.Errorf("current ProtectionPolicy response is nil")
		}
		return protectionPolicyFromSummary(*current), true, nil
	default:
		return recoverysdk.ProtectionPolicy{}, false, nil
	}
}

func protectionPolicyFromOperationResponse(currentResponse any) (recoverysdk.ProtectionPolicy, bool, error) {
	switch current := currentResponse.(type) {
	case recoverysdk.CreateProtectionPolicyResponse:
		return current.ProtectionPolicy, true, nil
	case *recoverysdk.CreateProtectionPolicyResponse:
		if current == nil {
			return recoverysdk.ProtectionPolicy{}, true, fmt.Errorf("current ProtectionPolicy response is nil")
		}
		return current.ProtectionPolicy, true, nil
	case recoverysdk.GetProtectionPolicyResponse:
		return current.ProtectionPolicy, true, nil
	case *recoverysdk.GetProtectionPolicyResponse:
		if current == nil {
			return recoverysdk.ProtectionPolicy{}, true, fmt.Errorf("current ProtectionPolicy response is nil")
		}
		return current.ProtectionPolicy, true, nil
	default:
		return recoverysdk.ProtectionPolicy{}, false, nil
	}
}

func protectionPolicyFromSummary(summary recoverysdk.ProtectionPolicySummary) recoverysdk.ProtectionPolicy {
	return recoverysdk.ProtectionPolicy(summary)
}

func desiredProtectionPolicyFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return cloneStringMap(spec), true
}

func desiredProtectionPolicyDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := protectionPolicyDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if protectionPolicyJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func protectionPolicyDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func conservativeProtectionPolicyGetCall(
	call func(context.Context, recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error),
) func(context.Context, recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
		response, err := call(ctx, request)
		return response, conservativeProtectionPolicyNotFoundError(err, "read")
	}
}

func conservativeProtectionPolicyNotFoundError(err error, operation string) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return protectionPolicyAmbiguousNotFoundError{
		message:      fmt.Sprintf("ProtectionPolicy %s returned ambiguous 404 NotAuthorizedOrNotFound; retaining finalizer: %v", strings.TrimSpace(operation), err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func protectionPolicyJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func handleProtectionPolicyDeleteError(resource *recoveryv1beta1.ProtectionPolicy, err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}

	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("ProtectionPolicy delete returned ambiguous %s %s; retaining finalizer",
		classification.HTTPStatusCodeString(), classification.ErrorCodeString())
}

type protectionPolicyDeleteGuardClient struct {
	delegate ProtectionPolicyServiceClient
	client   protectionPolicyOCIClient
	initErr  error
}

func wrapProtectionPolicyDeleteGuard(
	client protectionPolicyOCIClient,
	initErr error,
) func(ProtectionPolicyServiceClient) ProtectionPolicyServiceClient {
	return func(delegate ProtectionPolicyServiceClient) ProtectionPolicyServiceClient {
		return protectionPolicyDeleteGuardClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	}
}

func (c protectionPolicyDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c protectionPolicyDeleteGuardClient) Delete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
) (bool, error) {
	if handled, deleted, err := c.handleTrackedWorkRequestForDelete(ctx, resource); handled {
		return deleted, err
	}
	if handled, deleted, err := c.deleteWithPreDeleteConfirmGuard(ctx, resource); handled {
		return deleted, err
	}
	deleted, err := c.delegate.Delete(ctx, resource)
	if err != nil && protectionPolicyDeleteWorkRequestReadbackStillLive(resource) {
		markProtectionPolicyTerminating(resource, protectionPolicyDeletePendingMessage)
		return false, nil
	}
	return deleted, err
}

func (c protectionPolicyDeleteGuardClient) deleteWithPreDeleteConfirmGuard(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
) (bool, bool, error) {
	protectionPolicyID := recordedProtectionPolicyID(resource)
	if protectionPolicyID == "" {
		return false, false, nil
	}

	response, err := c.readProtectionPolicyForPreDeleteGuard(ctx, protectionPolicyID)
	if err != nil {
		return c.handlePreDeleteConfirmReadError(resource, err)
	}
	if shouldMarkProtectionPolicyDeleted(response) {
		markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy deleted")
		return true, true, nil
	}
	if shouldRetainProtectionPolicyDeleteBeforeRequest(resource, response) {
		recordProtectionPolicyDeleteReadback(resource, response)
		markProtectionPolicyTerminating(resource, protectionPolicyDeletePendingMessage)
		return true, false, nil
	}

	deleted, err := c.invokeProtectionPolicyDeleteAfterPreRead(ctx, resource, protectionPolicyID)
	return true, deleted, err
}

func recordedProtectionPolicyID(resource *recoveryv1beta1.ProtectionPolicy) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
		return id
	}
	return ""
}

func (c protectionPolicyDeleteGuardClient) readProtectionPolicyForPreDeleteGuard(
	ctx context.Context,
	protectionPolicyID string,
) (recoverysdk.GetProtectionPolicyResponse, error) {
	if c.initErr != nil {
		return recoverysdk.GetProtectionPolicyResponse{}, fmt.Errorf("initialize ProtectionPolicy OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return recoverysdk.GetProtectionPolicyResponse{}, fmt.Errorf("ProtectionPolicy OCI client is not configured")
	}
	return c.client.GetProtectionPolicy(ctx, recoverysdk.GetProtectionPolicyRequest{
		ProtectionPolicyId: stringPtr(protectionPolicyID),
	})
}

func (c protectionPolicyDeleteGuardClient) handlePreDeleteConfirmReadError(
	resource *recoveryv1beta1.ProtectionPolicy,
	err error,
) (bool, bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if classification.IsUnambiguousNotFound() {
		markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy deleted")
		return true, true, nil
	}
	if classification.IsAuthShapedNotFound() {
		message := fmt.Sprintf("ProtectionPolicy pre-delete confirm read returned ambiguous %s %s; retaining finalizer",
			classification.HTTPStatusCodeString(), classification.ErrorCodeString())
		markProtectionPolicyTerminating(resource, message)
		return true, false, fmt.Errorf("%s: %w", message, err)
	}
	return true, false, err
}

func shouldRetainProtectionPolicyDeleteBeforeRequest(
	resource *recoveryv1beta1.ProtectionPolicy,
	response recoverysdk.GetProtectionPolicyResponse,
) bool {
	if shouldHandleProtectionPolicyDeleteOutcome(resource, response, generatedruntime.DeleteConfirmStageAlreadyPending) {
		return true
	}
	return isProtectionPolicyDeletePendingState(protectionPolicyLifecycleStateFromResponse(response))
}

func isProtectionPolicyDeletePendingState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case string(recoverysdk.LifecycleStateDeleteScheduled), string(recoverysdk.LifecycleStateDeleting):
		return true
	default:
		return false
	}
}

func (c protectionPolicyDeleteGuardClient) invokeProtectionPolicyDeleteAfterPreRead(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	protectionPolicyID string,
) (bool, error) {
	if c.initErr != nil {
		return false, fmt.Errorf("initialize ProtectionPolicy OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return false, fmt.Errorf("ProtectionPolicy OCI client is not configured")
	}
	response, err := c.client.DeleteProtectionPolicy(ctx, recoverysdk.DeleteProtectionPolicyRequest{
		ProtectionPolicyId: stringPtr(protectionPolicyID),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy no longer exists")
			return true, nil
		}
		return false, handleProtectionPolicyDeleteError(resource, err)
	}

	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	workRequestID := stringPtrValue(response.OpcWorkRequestId)
	if workRequestID != "" {
		seedProtectionPolicyDeleteWorkRequest(resource, workRequestID)
		return c.handleDeleteWorkRequestAfterDeleteRequest(ctx, resource, workRequestID)
	}
	return c.confirmDeleteRequestAccepted(ctx, resource, protectionPolicyID)
}

func seedProtectionPolicyDeleteWorkRequest(resource *recoveryv1beta1.ProtectionPolicy, workRequestID string) {
	if resource == nil || strings.TrimSpace(workRequestID) == "" {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         fmt.Sprintf("ProtectionPolicy delete work request %s is in progress", strings.TrimSpace(workRequestID)),
		UpdatedAt:       &now,
	}
}

func (c protectionPolicyDeleteGuardClient) handleDeleteWorkRequestAfterDeleteRequest(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequestID string,
) (bool, error) {
	workRequest, currentAsync, err := c.currentWorkRequestOperation(ctx, resource, workRequestID, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("ProtectionPolicy delete work request %s is still in progress", workRequestID)
		applyProtectionPolicyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.confirmSucceededDeleteWorkRequest(ctx, resource, workRequest)
	default:
		_, _, err := c.handleTerminalWorkRequestForDelete(resource, currentAsync)
		return false, err
	}
}

func (c protectionPolicyDeleteGuardClient) confirmDeleteRequestAccepted(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	protectionPolicyID string,
) (bool, error) {
	response, err := c.readProtectionPolicyForDeleteConfirmation(ctx, protectionPolicyID)
	if err != nil {
		return handleProtectionPolicyDeleteConfirmationReadError(resource, err)
	}
	if shouldMarkProtectionPolicyDeleted(response) {
		markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy deleted")
		return true, nil
	}
	recordProtectionPolicyDeleteReadback(resource, response)
	markProtectionPolicyTerminating(resource, protectionPolicyDeletePendingMessage)
	return false, nil
}

func protectionPolicyDeleteWorkRequestReadbackStillLive(resource *recoveryv1beta1.ProtectionPolicy) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil ||
		current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		strings.TrimSpace(current.WorkRequestID) == "" {
		return false
	}

	switch strings.ToUpper(strings.TrimSpace(resource.Status.LifecycleState)) {
	case string(recoverysdk.LifecycleStateCreating),
		string(recoverysdk.LifecycleStateUpdating),
		string(recoverysdk.LifecycleStateActive):
		return true
	default:
		return false
	}
}

func (c protectionPolicyDeleteGuardClient) handleTrackedWorkRequestForDelete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
) (bool, bool, error) {
	workRequestID, phase := currentProtectionPolicyWorkRequest(resource)
	if workRequestID == "" {
		if protectionPolicyWriteAlreadyPending(resource) {
			markProtectionPolicyWritePendingDeleteGuard(resource)
			return true, false, nil
		}
		return false, false, nil
	}

	switch phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return c.handlePendingWriteWorkRequestForDelete(ctx, resource, workRequestID, phase)
	case shared.OSOKAsyncPhaseDelete:
		return c.handlePendingDeleteWorkRequestForDelete(ctx, resource, workRequestID)
	default:
		err := fmt.Errorf("ProtectionPolicy delete cannot resume %s work request %s from delete path", phase, workRequestID)
		markProtectionPolicyFailed(resource, err)
		return true, false, err
	}
}

func (c protectionPolicyDeleteGuardClient) handlePendingWriteWorkRequestForDelete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, bool, error) {
	workRequest, currentAsync, err := c.currentWorkRequestOperation(ctx, resource, workRequestID, phase)
	if err != nil {
		return true, false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("ProtectionPolicy %s work request %s is still in progress; waiting before delete", phase, workRequestID)
		applyProtectionPolicyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
		return true, false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.handleSucceededWriteWorkRequestForDelete(resource, workRequest, currentAsync)
	default:
		return c.handleTerminalWorkRequestForDelete(resource, currentAsync)
	}
}

func (c protectionPolicyDeleteGuardClient) handlePendingDeleteWorkRequestForDelete(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequestID string,
) (bool, bool, error) {
	workRequest, currentAsync, err := c.currentWorkRequestOperation(ctx, resource, workRequestID, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return true, false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("ProtectionPolicy delete work request %s is still in progress", workRequestID)
		applyProtectionPolicyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
		return true, false, nil
	case shared.OSOKAsyncClassSucceeded:
		deleted, err := c.confirmSucceededDeleteWorkRequest(ctx, resource, workRequest)
		return true, deleted, err
	default:
		return c.handleTerminalWorkRequestForDelete(resource, currentAsync)
	}
}

func (c protectionPolicyDeleteGuardClient) currentWorkRequestOperation(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (any, *shared.OSOKAsyncOperation, error) {
	workRequest, err := getProtectionPolicyWorkRequest(ctx, c.client, c.initErr, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markProtectionPolicyFailed(resource, err)
		return nil, nil, err
	}
	currentAsync, err := buildProtectionPolicyWorkRequestOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		markProtectionPolicyFailed(resource, err)
		return nil, nil, err
	}
	return workRequest, currentAsync, nil
}

func (c protectionPolicyDeleteGuardClient) handleSucceededWriteWorkRequestForDelete(
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequest any,
	currentAsync *shared.OSOKAsyncOperation,
) (bool, bool, error) {
	if err := recordProtectionPolicyIDFromSucceededWorkRequest(resource, workRequest, currentAsync.Phase); err != nil {
		markProtectionPolicyFailed(resource, err)
		return true, false, err
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return false, false, nil
}

func (c protectionPolicyDeleteGuardClient) handleTerminalWorkRequestForDelete(
	resource *recoveryv1beta1.ProtectionPolicy,
	currentAsync *shared.OSOKAsyncOperation,
) (bool, bool, error) {
	workRequestID := ""
	phase := shared.OSOKAsyncPhase("")
	rawStatus := ""
	if currentAsync != nil {
		workRequestID = currentAsync.WorkRequestID
		phase = currentAsync.Phase
		rawStatus = currentAsync.RawStatus
	}
	err := fmt.Errorf("ProtectionPolicy %s work request %s finished with status %s", phase, workRequestID, rawStatus)
	applyProtectionPolicyWorkRequestOperation(resource, currentAsync)
	return true, false, err
}

func (c protectionPolicyDeleteGuardClient) confirmSucceededDeleteWorkRequest(
	ctx context.Context,
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequest any,
) (bool, error) {
	protectionPolicyID, err := protectionPolicyDeleteConfirmationID(resource, workRequest)
	if err != nil {
		markProtectionPolicyFailed(resource, err)
		return false, err
	}
	if protectionPolicyID == "" {
		markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy delete work request completed")
		return true, nil
	}

	response, err := c.readProtectionPolicyForDeleteConfirmation(ctx, protectionPolicyID)
	if err != nil {
		return handleProtectionPolicyDeleteConfirmationReadError(resource, err)
	}
	if shouldMarkProtectionPolicyDeleted(response) {
		markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy deleted")
		return true, nil
	}
	recordProtectionPolicyDeleteReadback(resource, response)
	markProtectionPolicyTerminating(resource, protectionPolicyDeletePendingMessage)
	return false, nil
}

func recordProtectionPolicyIDFromSucceededWorkRequest(
	resource *recoveryv1beta1.ProtectionPolicy,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) error {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return nil
	}
	protectionPolicyID, err := recoverProtectionPolicyIDFromGeneratedWorkRequest(resource, workRequest, phase)
	if err != nil {
		return err
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(protectionPolicyID)
	resource.Status.Id = protectionPolicyID
	return nil
}

func protectionPolicyDeleteConfirmationID(resource *recoveryv1beta1.ProtectionPolicy, workRequest any) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("ProtectionPolicy resource is nil")
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id, nil
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
		return id, nil
	}
	id, err := recoverProtectionPolicyIDFromGeneratedWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return "", nil
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.Id = id
	return id, nil
}

func (c protectionPolicyDeleteGuardClient) readProtectionPolicyForDeleteConfirmation(
	ctx context.Context,
	protectionPolicyID string,
) (recoverysdk.GetProtectionPolicyResponse, error) {
	if c.initErr != nil {
		return recoverysdk.GetProtectionPolicyResponse{}, fmt.Errorf("initialize ProtectionPolicy OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return recoverysdk.GetProtectionPolicyResponse{}, fmt.Errorf("ProtectionPolicy OCI client is not configured")
	}
	response, err := c.client.GetProtectionPolicy(ctx, recoverysdk.GetProtectionPolicyRequest{
		ProtectionPolicyId: stringPtr(protectionPolicyID),
	})
	return response, conservativeProtectionPolicyNotFoundError(err, "delete confirmation")
}

func handleProtectionPolicyDeleteConfirmationReadError(
	resource *recoveryv1beta1.ProtectionPolicy,
	err error,
) (bool, error) {
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		if strings.TrimSpace(resource.Status.OsokStatus.OpcRequestID) == "" {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		markProtectionPolicyDeleted(resource, "OCI ProtectionPolicy deleted")
		return true, nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return false, err
}

func shouldMarkProtectionPolicyDeleted(response recoverysdk.GetProtectionPolicyResponse) bool {
	return protectionPolicyLifecycleStateFromResponse(response) == string(recoverysdk.LifecycleStateDeleted)
}

func recordProtectionPolicyDeleteReadback(
	resource *recoveryv1beta1.ProtectionPolicy,
	response recoverysdk.GetProtectionPolicyResponse,
) {
	if resource == nil {
		return
	}
	current := response.ProtectionPolicy
	if id := stringPtrValue(current.Id); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	resource.Status.LifecycleState = strings.ToUpper(strings.TrimSpace(string(current.LifecycleState)))
}

func markProtectionPolicyDeleted(resource *recoveryv1beta1.ProtectionPolicy, message string) {
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

func applyProtectionPolicyDeleteOutcome(
	resource *recoveryv1beta1.ProtectionPolicy,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if response == nil || !shouldHandleProtectionPolicyDeleteOutcome(resource, response, stage) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	markProtectionPolicyTerminating(resource, protectionPolicyDeletePendingMessage)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func shouldHandleProtectionPolicyDeleteOutcome(
	resource *recoveryv1beta1.ProtectionPolicy,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) bool {
	switch stage {
	case generatedruntime.DeleteConfirmStageAlreadyPending:
		if !protectionPolicyDeleteAlreadyPending(resource) {
			return false
		}
	case generatedruntime.DeleteConfirmStageAfterRequest:
	default:
		return false
	}

	state := protectionPolicyLifecycleStateFromResponse(response)
	switch state {
	case "", string(recoverysdk.LifecycleStateDeleted), string(recoverysdk.LifecycleStateFailed):
		return false
	default:
		return true
	}
}

func protectionPolicyLifecycleStateFromResponse(response any) string {
	current, err := protectionPolicyFromResponse(response)
	if err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(string(current.LifecycleState)))
}

func protectionPolicyDeleteAlreadyPending(resource *recoveryv1beta1.ProtectionPolicy) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func protectionPolicyWriteAlreadyPending(resource *recoveryv1beta1.ProtectionPolicy) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func markProtectionPolicyWritePendingDeleteGuard(resource *recoveryv1beta1.ProtectionPolicy) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = protectionPolicyPendingWriteDeleteMessage
	if status.Async.Current != nil && status.Async.Current.UpdatedAt == nil {
		status.Async.Current.UpdatedAt = &now
	}
}

func markProtectionPolicyTerminating(resource *recoveryv1beta1.ProtectionPolicy, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	source := shared.OSOKAsyncSourceLifecycle
	workRequestID := ""
	if current := status.Async.Current; current != nil && current.Phase == shared.OSOKAsyncPhaseDelete {
		source = current.Source
		workRequestID = current.WorkRequestID
	}
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          source,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   workRequestID,
		RawStatus:       strings.TrimSpace(resource.Status.LifecycleState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func getProtectionPolicyWorkRequest(
	ctx context.Context,
	client protectionPolicyOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize ProtectionPolicy OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("ProtectionPolicy OCI client is not configured")
	}
	trimmedWorkRequestID := strings.TrimSpace(workRequestID)
	if trimmedWorkRequestID == "" {
		return nil, fmt.Errorf("ProtectionPolicy work request id is empty")
	}
	response, err := client.GetWorkRequest(ctx, recoverysdk.GetWorkRequestRequest{
		WorkRequestId: stringPtr(trimmedWorkRequestID),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveProtectionPolicyGeneratedWorkRequestAction(workRequest any) (string, error) {
	protectionPolicyWorkRequest, err := protectionPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveProtectionPolicyWorkRequestAction(protectionPolicyWorkRequest)
}

func resolveProtectionPolicyGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	protectionPolicyWorkRequest, err := protectionPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := protectionPolicyWorkRequestPhaseFromOperationType(protectionPolicyWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverProtectionPolicyIDFromGeneratedWorkRequest(
	_ *recoveryv1beta1.ProtectionPolicy,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	protectionPolicyWorkRequest, err := protectionPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveProtectionPolicyIDFromWorkRequest(protectionPolicyWorkRequest, protectionPolicyWorkRequestActionForPhase(phase))
}

func protectionPolicyGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	protectionPolicyWorkRequest, err := protectionPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ProtectionPolicy %s work request %s is %s", phase, stringPtrValue(protectionPolicyWorkRequest.Id), protectionPolicyWorkRequest.Status)
}

func protectionPolicyWorkRequestFromAny(workRequest any) (recoverysdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case recoverysdk.WorkRequest:
		return current, nil
	case *recoverysdk.WorkRequest:
		if current == nil {
			return recoverysdk.WorkRequest{}, fmt.Errorf("ProtectionPolicy work request is nil")
		}
		return *current, nil
	default:
		return recoverysdk.WorkRequest{}, fmt.Errorf("unexpected ProtectionPolicy work request type %T", workRequest)
	}
}

func resolveProtectionPolicyWorkRequestAction(workRequest recoverysdk.WorkRequest) (string, error) {
	action := ""
	for _, resource := range workRequest.Resources {
		if !isProtectionPolicyWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" ||
			strings.EqualFold(candidate, string(recoverysdk.ActionTypeInProgress)) ||
			strings.EqualFold(candidate, string(recoverysdk.ActionTypeRelated)) ||
			strings.EqualFold(candidate, string(recoverysdk.ActionTypeFailed)) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("ProtectionPolicy work request %s exposes conflicting ProtectionPolicy action types %q and %q", stringPtrValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func protectionPolicyWorkRequestPhaseFromOperationType(operationType recoverysdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case recoverysdk.OperationTypeCreateProtectionPolicy:
		return shared.OSOKAsyncPhaseCreate, true
	case recoverysdk.OperationTypeUpdateProtectionPolicy:
		return shared.OSOKAsyncPhaseUpdate, true
	case recoverysdk.OperationTypeDeleteProtectionPolicy:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func protectionPolicyWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) recoverysdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return recoverysdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return recoverysdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return recoverysdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveProtectionPolicyIDFromWorkRequest(
	workRequest recoverysdk.WorkRequest,
	action recoverysdk.ActionTypeEnum,
) (string, error) {
	if id, ok := resolveProtectionPolicyIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveProtectionPolicyIDFromResources(workRequest.Resources, "", true); ok {
		return id, nil
	}
	if id, ok := resolveProtectionPolicyIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("ProtectionPolicy work request %s does not expose a ProtectionPolicy identifier", stringPtrValue(workRequest.Id))
}

func resolveProtectionPolicyIDFromResources(
	resources []recoverysdk.WorkRequestResource,
	action recoverysdk.ActionTypeEnum,
	preferProtectionPolicyOnly bool,
) (string, bool) {
	candidate := ""
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferProtectionPolicyOnly && !isProtectionPolicyWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringPtrValue(resource.Identifier))
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

func isProtectionPolicyWorkRequestResource(resource recoverysdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringPtrValue(resource.EntityType)))
	normalizedEntityType := strings.NewReplacer("_", "", "-", "", " ", "").Replace(entityType)
	if strings.Contains(normalizedEntityType, "protectionpolicy") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringPtrValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/protectionpolicies/")
}

func currentProtectionPolicyWorkRequest(resource *recoveryv1beta1.ProtectionPolicy) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func buildProtectionPolicyWorkRequestOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	protectionPolicyWorkRequest, err := protectionPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	action, err := resolveProtectionPolicyWorkRequestAction(protectionPolicyWorkRequest)
	if err != nil {
		return nil, err
	}
	if phase, ok := protectionPolicyWorkRequestPhaseFromOperationType(protectionPolicyWorkRequest.OperationType); ok {
		if fallbackPhase != "" && fallbackPhase != phase {
			return nil, fmt.Errorf(
				"ProtectionPolicy work request %s exposes phase %q while reconcile expected %q",
				stringPtrValue(protectionPolicyWorkRequest.Id),
				phase,
				fallbackPhase,
			)
		}
		fallbackPhase = phase
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, protectionPolicyWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(protectionPolicyWorkRequest.Status),
		RawAction:        action,
		RawOperationType: string(protectionPolicyWorkRequest.OperationType),
		WorkRequestID:    stringPtrValue(protectionPolicyWorkRequest.Id),
		PercentComplete:  protectionPolicyWorkRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func applyProtectionPolicyWorkRequestOperation(
	resource *recoveryv1beta1.ProtectionPolicy,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	if current.WorkRequestID == "" && resource.Status.OsokStatus.Async.Current != nil {
		current.WorkRequestID = resource.Status.OsokStatus.Async.Current.WorkRequestID
	}
	now := metav1.Now()
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: 0,
	}
}

func applyProtectionPolicyWorkRequestOperationAs(
	resource *recoveryv1beta1.ProtectionPolicy,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	if current == nil {
		return applyProtectionPolicyWorkRequestOperation(resource, current)
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	return applyProtectionPolicyWorkRequestOperation(resource, &next)
}

func markProtectionPolicyFailed(resource *recoveryv1beta1.ProtectionPolicy, err error) {
	if resource == nil || err == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	return maps.Clone(value)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func intPtrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func newProtectionPolicyServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client protectionPolicyOCIClient,
) ProtectionPolicyServiceClient {
	hooks := newProtectionPolicyRuntimeHooksWithOCIClient(client)
	applyProtectionPolicyRuntimeHooks(&hooks, client, nil)
	delegate := defaultProtectionPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*recoveryv1beta1.ProtectionPolicy](
			buildProtectionPolicyGeneratedRuntimeConfig(&ProtectionPolicyServiceManager{Log: log}, hooks),
		),
	}
	return wrapProtectionPolicyGeneratedClient(hooks, delegate)
}

func newProtectionPolicyRuntimeHooksWithOCIClient(client protectionPolicyOCIClient) ProtectionPolicyRuntimeHooks {
	return ProtectionPolicyRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*recoveryv1beta1.ProtectionPolicy]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*recoveryv1beta1.ProtectionPolicy]{},
		StatusHooks:     generatedruntime.StatusHooks[*recoveryv1beta1.ProtectionPolicy]{},
		ParityHooks:     generatedruntime.ParityHooks[*recoveryv1beta1.ProtectionPolicy]{},
		Async:           generatedruntime.AsyncHooks[*recoveryv1beta1.ProtectionPolicy]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*recoveryv1beta1.ProtectionPolicy]{},
		Create: runtimeOperationHooks[recoverysdk.CreateProtectionPolicyRequest, recoverysdk.CreateProtectionPolicyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateProtectionPolicyDetails", RequestName: "CreateProtectionPolicyDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error) {
				return client.CreateProtectionPolicy(ctx, request)
			},
		},
		Get: runtimeOperationHooks[recoverysdk.GetProtectionPolicyRequest, recoverysdk.GetProtectionPolicyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProtectionPolicyId", RequestName: "protectionPolicyId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
				return client.GetProtectionPolicy(ctx, request)
			},
		},
		List: runtimeOperationHooks[recoverysdk.ListProtectionPoliciesRequest, recoverysdk.ListProtectionPoliciesResponse]{
			Fields: protectionPolicyListFields(),
			Call: func(ctx context.Context, request recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
				return client.ListProtectionPolicies(ctx, request)
			},
		},
		Update: runtimeOperationHooks[recoverysdk.UpdateProtectionPolicyRequest, recoverysdk.UpdateProtectionPolicyResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ProtectionPolicyId", RequestName: "protectionPolicyId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateProtectionPolicyDetails", RequestName: "UpdateProtectionPolicyDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error) {
				return client.UpdateProtectionPolicy(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[recoverysdk.DeleteProtectionPolicyRequest, recoverysdk.DeleteProtectionPolicyResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProtectionPolicyId", RequestName: "protectionPolicyId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
				return client.DeleteProtectionPolicy(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ProtectionPolicyServiceClient) ProtectionPolicyServiceClient{},
	}
}
