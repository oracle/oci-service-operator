/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetalertpolicyassociation

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
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const targetAlertPolicyAssociationKind = "TargetAlertPolicyAssociation"

var targetAlertPolicyAssociationWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
	CreateActionTokens:    []string{string(datasafesdk.WorkRequestResourceActionTypeCreated)},
	UpdateActionTokens: []string{
		string(datasafesdk.WorkRequestResourceActionTypeUpdated),
		string(datasafesdk.WorkRequestOperationTypePatchTargetAlertPolicyAssociation),
	},
	DeleteActionTokens: []string{string(datasafesdk.WorkRequestResourceActionTypeDeleted)},
}

type targetAlertPolicyAssociationOCIClient interface {
	CreateTargetAlertPolicyAssociation(context.Context, datasafesdk.CreateTargetAlertPolicyAssociationRequest) (datasafesdk.CreateTargetAlertPolicyAssociationResponse, error)
	GetTargetAlertPolicyAssociation(context.Context, datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error)
	ListTargetAlertPolicyAssociations(context.Context, datasafesdk.ListTargetAlertPolicyAssociationsRequest) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error)
	UpdateTargetAlertPolicyAssociation(context.Context, datasafesdk.UpdateTargetAlertPolicyAssociationRequest) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error)
	DeleteTargetAlertPolicyAssociation(context.Context, datasafesdk.DeleteTargetAlertPolicyAssociationRequest) (datasafesdk.DeleteTargetAlertPolicyAssociationResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type targetAlertPolicyAssociationIdentity struct {
	compartmentID string
	policyID      string
	targetID      string
}

type targetAlertPolicyAssociationRuntimeClient struct {
	delegate       TargetAlertPolicyAssociationServiceClient
	get            func(context.Context, datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error)
	getWorkRequest func(context.Context, string) (datasafesdk.WorkRequest, error)
}

type targetAlertPolicyAssociationAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e targetAlertPolicyAssociationAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e targetAlertPolicyAssociationAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerTargetAlertPolicyAssociationRuntimeHooksMutator(func(manager *TargetAlertPolicyAssociationServiceManager, hooks *TargetAlertPolicyAssociationRuntimeHooks) {
		client, initErr := newTargetAlertPolicyAssociationOCIClient(manager)
		applyTargetAlertPolicyAssociationRuntimeHooks(hooks, client, initErr)
	})
}

func newTargetAlertPolicyAssociationOCIClient(manager *TargetAlertPolicyAssociationServiceManager) (targetAlertPolicyAssociationOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", targetAlertPolicyAssociationKind)
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTargetAlertPolicyAssociationRuntimeHooks(
	hooks *TargetAlertPolicyAssociationRuntimeHooks,
	client targetAlertPolicyAssociationOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = targetAlertPolicyAssociationRuntimeSemantics()
	hooks.BuildCreateBody = buildTargetAlertPolicyAssociationCreateBody
	hooks.BuildUpdateBody = buildTargetAlertPolicyAssociationUpdateBody
	hooks.Identity.Resolve = resolveTargetAlertPolicyAssociationIdentity
	hooks.Identity.RecordPath = recordTargetAlertPolicyAssociationPathIdentity
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedTargetAlertPolicyAssociationIdentity
	hooks.Create.Fields = targetAlertPolicyAssociationCreateFields()
	hooks.Get.Fields = targetAlertPolicyAssociationGetFields()
	hooks.List.Fields = targetAlertPolicyAssociationListFields()
	hooks.Update.Fields = targetAlertPolicyAssociationUpdateFields()
	hooks.Delete.Fields = targetAlertPolicyAssociationDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listTargetAlertPolicyAssociationsAllPages(hooks.List.Call)
	}
	hooks.StatusHooks.ClearProjectedStatus = clearTargetAlertPolicyAssociationProjectedStatus
	hooks.StatusHooks.RestoreStatus = restoreTargetAlertPolicyAssociationProjectedStatus
	hooks.StatusHooks.ProjectStatus = projectTargetAlertPolicyAssociationStatus
	hooks.DeleteHooks.HandleError = handleTargetAlertPolicyAssociationDeleteError
	hooks.Async.Adapter = targetAlertPolicyAssociationWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getTargetAlertPolicyAssociationWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveTargetAlertPolicyAssociationWorkRequestAction
	hooks.Async.RecoverResourceID = recoverTargetAlertPolicyAssociationIDFromWorkRequest
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TargetAlertPolicyAssociationServiceClient) TargetAlertPolicyAssociationServiceClient {
		return targetAlertPolicyAssociationRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			getWorkRequest: func(ctx context.Context, workRequestID string) (datasafesdk.WorkRequest, error) {
				return getTargetAlertPolicyAssociationWorkRequest(ctx, client, initErr, workRequestID)
			},
		}
	})
}

func targetAlertPolicyAssociationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "targetalertpolicyassociation",
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
			ProvisioningStates: []string{string(datasafesdk.AlertPolicyLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.AlertPolicyLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.AlertPolicyLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.AlertPolicyLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.AlertPolicyLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"policyId",
				"targetId",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"isEnabled",
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"policyId",
				"targetId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: targetAlertPolicyAssociationKind, Action: "CreateTargetAlertPolicyAssociation"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: targetAlertPolicyAssociationKind, Action: "UpdateTargetAlertPolicyAssociation"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: targetAlertPolicyAssociationKind, Action: "DeleteTargetAlertPolicyAssociation"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetTargetAlertPolicyAssociation"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetTargetAlertPolicyAssociation"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func targetAlertPolicyAssociationCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateTargetAlertPolicyAssociationDetails", RequestName: "CreateTargetAlertPolicyAssociationDetails", Contribution: "body"},
	}
}

func targetAlertPolicyAssociationGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TargetAlertPolicyAssociationId", RequestName: "targetAlertPolicyAssociationId", Contribution: "path", PreferResourceID: true},
	}
}

func targetAlertPolicyAssociationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "TargetAlertPolicyAssociationId", RequestName: "targetAlertPolicyAssociationId", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "AlertPolicyId",
			RequestName:  "alertPolicyId",
			Contribution: "query",
			LookupPaths:  []string{"status.policyId", "spec.policyId", "policyId"},
		},
		{
			FieldName:    "TargetId",
			RequestName:  "targetId",
			Contribution: "query",
			LookupPaths:  []string{"status.targetId", "spec.targetId", "targetId"},
		},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "TimeCreatedGreaterThanOrEqualTo", RequestName: "timeCreatedGreaterThanOrEqualTo", Contribution: "query"},
		{FieldName: "TimeCreatedLessThan", RequestName: "timeCreatedLessThan", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
	}
}

func targetAlertPolicyAssociationUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TargetAlertPolicyAssociationId", RequestName: "targetAlertPolicyAssociationId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateTargetAlertPolicyAssociationDetails", RequestName: "UpdateTargetAlertPolicyAssociationDetails", Contribution: "body"},
	}
}

func targetAlertPolicyAssociationDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TargetAlertPolicyAssociationId", RequestName: "targetAlertPolicyAssociationId", Contribution: "path", PreferResourceID: true},
	}
}

func buildTargetAlertPolicyAssociationCreateBody(
	_ context.Context,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", targetAlertPolicyAssociationKind)
	}
	if err := validateTargetAlertPolicyAssociationCreateIdentity(resource); err != nil {
		return nil, err
	}

	return datasafesdk.CreateTargetAlertPolicyAssociationDetails{
		PolicyId:      common.String(strings.TrimSpace(resource.Spec.PolicyId)),
		TargetId:      common.String(strings.TrimSpace(resource.Spec.TargetId)),
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		IsEnabled:     common.Bool(resource.Spec.IsEnabled),
		DisplayName:   targetAlertPolicyAssociationOptionalString(resource.Spec.DisplayName),
		Description:   targetAlertPolicyAssociationOptionalString(resource.Spec.Description),
		FreeformTags:  targetAlertPolicyAssociationStringMap(resource.Spec.FreeformTags),
		DefinedTags:   targetAlertPolicyAssociationDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func buildTargetAlertPolicyAssociationUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", targetAlertPolicyAssociationKind)
	}
	current, ok := targetAlertPolicyAssociationStatusProjectionFromResponse(currentResponse)
	if !ok {
		current = targetAlertPolicyAssociationStatusProjectionFromResource(resource)
	}

	details := datasafesdk.UpdateTargetAlertPolicyAssociationDetails{}
	updateNeeded := false
	if resource.Spec.IsEnabled != current.IsEnabled {
		details.IsEnabled = common.Bool(resource.Spec.IsEnabled)
		updateNeeded = true
	}
	updateNeeded = applyTargetAlertPolicyAssociationStringUpdates(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyTargetAlertPolicyAssociationTagUpdates(&details, resource.Spec, current) || updateNeeded
	if !updateNeeded {
		return datasafesdk.UpdateTargetAlertPolicyAssociationDetails{}, false, nil
	}
	return details, true, nil
}

func applyTargetAlertPolicyAssociationStringUpdates(
	details *datasafesdk.UpdateTargetAlertPolicyAssociationDetails,
	spec datasafev1beta1.TargetAlertPolicyAssociationSpec,
	current targetAlertPolicyAssociationStatusProjection,
) bool {
	updateNeeded := false
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != current.DisplayName {
		details.DisplayName = common.String(displayName)
		updateNeeded = true
	}
	if description := strings.TrimSpace(spec.Description); description != current.Description {
		details.Description = common.String(description)
		updateNeeded = true
	}
	return updateNeeded
}

func applyTargetAlertPolicyAssociationTagUpdates(
	details *datasafesdk.UpdateTargetAlertPolicyAssociationDetails,
	spec datasafev1beta1.TargetAlertPolicyAssociationSpec,
	current targetAlertPolicyAssociationStatusProjection,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = targetAlertPolicyAssociationStringMap(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil && !reflect.DeepEqual(targetAlertPolicyAssociationDefinedTags(spec.DefinedTags), targetAlertPolicyAssociationDefinedTags(current.DefinedTags)) {
		details.DefinedTags = targetAlertPolicyAssociationDefinedTags(spec.DefinedTags)
		updateNeeded = true
	}
	return updateNeeded
}

func validateTargetAlertPolicyAssociationCreateIdentity(resource *datasafev1beta1.TargetAlertPolicyAssociation) error {
	switch {
	case strings.TrimSpace(resource.Spec.PolicyId) == "":
		return fmt.Errorf("%s requires spec.policyId", targetAlertPolicyAssociationKind)
	case strings.TrimSpace(resource.Spec.TargetId) == "":
		return fmt.Errorf("%s requires spec.targetId", targetAlertPolicyAssociationKind)
	case strings.TrimSpace(resource.Spec.CompartmentId) == "":
		return fmt.Errorf("%s requires spec.compartmentId", targetAlertPolicyAssociationKind)
	default:
		return nil
	}
}

func resolveTargetAlertPolicyAssociationIdentity(resource *datasafev1beta1.TargetAlertPolicyAssociation) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", targetAlertPolicyAssociationKind)
	}
	if err := validateTargetAlertPolicyAssociationCreateIdentity(resource); err != nil {
		return nil, err
	}
	return targetAlertPolicyAssociationIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		policyID:      strings.TrimSpace(resource.Spec.PolicyId),
		targetID:      strings.TrimSpace(resource.Spec.TargetId),
	}, nil
}

func recordTargetAlertPolicyAssociationPathIdentity(resource *datasafev1beta1.TargetAlertPolicyAssociation, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(targetAlertPolicyAssociationIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if typed.policyID != "" {
		resource.Status.PolicyId = typed.policyID
	}
	if typed.targetID != "" {
		resource.Status.TargetId = typed.targetID
	}
}

func listTargetAlertPolicyAssociationsAllPages(
	call func(context.Context, datasafesdk.ListTargetAlertPolicyAssociationsRequest) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error),
) func(context.Context, datasafesdk.ListTargetAlertPolicyAssociationsRequest) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListTargetAlertPolicyAssociationsRequest) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error) {
		var combined datasafesdk.ListTargetAlertPolicyAssociationsResponse
		seenPages := map[string]struct{}{}
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			appendTargetAlertPolicyAssociationListPage(&combined, response)
			nextPage, err := nextTargetAlertPolicyAssociationPage(response, seenPages)
			if err != nil {
				return datasafesdk.ListTargetAlertPolicyAssociationsResponse{}, err
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

func appendTargetAlertPolicyAssociationListPage(
	combined *datasafesdk.ListTargetAlertPolicyAssociationsResponse,
	response datasafesdk.ListTargetAlertPolicyAssociationsResponse,
) {
	combined.RawResponse = response.RawResponse
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	if len(response.Items) > 0 {
		combined.Items = append(combined.Items, response.Items...)
	}
}

func nextTargetAlertPolicyAssociationPage(
	response datasafesdk.ListTargetAlertPolicyAssociationsResponse,
	seenPages map[string]struct{},
) (string, error) {
	nextPage := targetAlertPolicyAssociationStringValue(response.OpcNextPage)
	if nextPage == "" {
		return "", nil
	}
	if _, ok := seenPages[nextPage]; ok {
		return "", fmt.Errorf("%s list pagination repeated page token %q", targetAlertPolicyAssociationKind, nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nextPage, nil
}

func clearTrackedTargetAlertPolicyAssociationIdentity(resource *datasafev1beta1.TargetAlertPolicyAssociation) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = ""
}

func handleTargetAlertPolicyAssociationDeleteError(resource *datasafev1beta1.TargetAlertPolicyAssociation, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := targetAlertPolicyAssociationAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func (c targetAlertPolicyAssociationRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", targetAlertPolicyAssociationKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c targetAlertPolicyAssociationRuntimeClient) Delete(
	ctx context.Context,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", targetAlertPolicyAssociationKind)
	}
	if currentID := currentTargetAlertPolicyAssociationID(resource); currentID != "" {
		if err := c.guardTargetAlertPolicyAssociationPreDeleteGet(ctx, resource, currentID); err != nil {
			return false, err
		}
	}
	if err := c.guardTargetAlertPolicyAssociationCompletedDeleteWorkRequest(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c targetAlertPolicyAssociationRuntimeClient) guardTargetAlertPolicyAssociationPreDeleteGet(
	ctx context.Context,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	response, err := c.get(ctx, datasafesdk.GetTargetAlertPolicyAssociationRequest{
		TargetAlertPolicyAssociationId: common.String(currentID),
	})
	if ambiguous := targetAlertPolicyAssociationAmbiguousDeleteError(resource, err, "pre-delete get"); ambiguous != nil {
		return ambiguous
	}
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil
	}
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return fmt.Errorf("%s pre-delete get failed; refusing to call delete: %w", targetAlertPolicyAssociationKind, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return nil
}

func (c targetAlertPolicyAssociationRuntimeClient) guardTargetAlertPolicyAssociationCompletedDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
) error {
	if c.get == nil || c.getWorkRequest == nil {
		return nil
	}
	workRequestID := trackedTargetAlertPolicyAssociationDeleteWorkRequestID(resource)
	if workRequestID == "" {
		return nil
	}
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil || workRequest.Status != datasafesdk.WorkRequestStatusSucceeded {
		return nil
	}
	currentID := currentTargetAlertPolicyAssociationID(resource)
	if currentID == "" {
		recoveredID, recoverErr := recoverTargetAlertPolicyAssociationIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if recoverErr != nil {
			return nil
		}
		currentID = strings.TrimSpace(recoveredID)
	}
	if currentID == "" {
		return nil
	}

	_, err = c.get(ctx, datasafesdk.GetTargetAlertPolicyAssociationRequest{
		TargetAlertPolicyAssociationId: common.String(currentID),
	})
	if ambiguous := targetAlertPolicyAssociationAmbiguousDeleteError(resource, err, "post-delete work request confirmation get"); ambiguous != nil {
		return ambiguous
	}
	return nil
}

func trackedTargetAlertPolicyAssociationDeleteWorkRequestID(resource *datasafev1beta1.TargetAlertPolicyAssociation) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func targetAlertPolicyAssociationAmbiguousDeleteError(
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return targetAlertPolicyAssociationAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", targetAlertPolicyAssociationKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func projectTargetAlertPolicyAssociationStatus(resource *datasafev1beta1.TargetAlertPolicyAssociation, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", targetAlertPolicyAssociationKind)
	}
	projected, ok := targetAlertPolicyAssociationStatusProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.TargetAlertPolicyAssociationStatus{
		OsokStatus:       osokStatus,
		Id:               projected.Id,
		CompartmentId:    projected.CompartmentId,
		TimeCreated:      projected.TimeCreated,
		TimeUpdated:      projected.TimeUpdated,
		LifecycleState:   projected.LifecycleState,
		DisplayName:      projected.DisplayName,
		Description:      projected.Description,
		PolicyId:         projected.PolicyId,
		TargetId:         projected.TargetId,
		IsEnabled:        projected.IsEnabled,
		LifecycleDetails: projected.LifecycleDetails,
		FreeformTags:     targetAlertPolicyAssociationStringMap(projected.FreeformTags),
		DefinedTags:      targetAlertPolicyAssociationCloneSharedTags(projected.DefinedTags),
		SystemTags:       targetAlertPolicyAssociationCloneSharedTags(projected.SystemTags),
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

type targetAlertPolicyAssociationStatusProjection struct {
	Id               string
	CompartmentId    string
	TimeCreated      string
	TimeUpdated      string
	LifecycleState   string
	DisplayName      string
	Description      string
	PolicyId         string
	TargetId         string
	IsEnabled        bool
	LifecycleDetails string
	FreeformTags     map[string]string
	DefinedTags      map[string]shared.MapValue
	SystemTags       map[string]shared.MapValue
}

func targetAlertPolicyAssociationStatusProjectionFromResponse(response any) (targetAlertPolicyAssociationStatusProjection, bool) {
	if current, ok := targetAlertPolicyAssociationFromResponse(response); ok {
		return targetAlertPolicyAssociationStatusProjectionFromSDK(current), true
	}
	if summary, ok := targetAlertPolicyAssociationSummaryFromResponse(response); ok {
		return targetAlertPolicyAssociationStatusProjectionFromSummary(summary), true
	}
	return targetAlertPolicyAssociationStatusProjection{}, false
}

func targetAlertPolicyAssociationStatusProjectionFromResource(resource *datasafev1beta1.TargetAlertPolicyAssociation) targetAlertPolicyAssociationStatusProjection {
	if resource == nil {
		return targetAlertPolicyAssociationStatusProjection{}
	}
	return targetAlertPolicyAssociationStatusProjection{
		Id:               strings.TrimSpace(resource.Status.Id),
		CompartmentId:    strings.TrimSpace(resource.Status.CompartmentId),
		TimeCreated:      resource.Status.TimeCreated,
		TimeUpdated:      resource.Status.TimeUpdated,
		LifecycleState:   strings.TrimSpace(resource.Status.LifecycleState),
		DisplayName:      strings.TrimSpace(resource.Status.DisplayName),
		Description:      strings.TrimSpace(resource.Status.Description),
		PolicyId:         strings.TrimSpace(resource.Status.PolicyId),
		TargetId:         strings.TrimSpace(resource.Status.TargetId),
		IsEnabled:        resource.Status.IsEnabled,
		LifecycleDetails: strings.TrimSpace(resource.Status.LifecycleDetails),
		FreeformTags:     targetAlertPolicyAssociationStringMap(resource.Status.FreeformTags),
		DefinedTags:      targetAlertPolicyAssociationCloneSharedTags(resource.Status.DefinedTags),
		SystemTags:       targetAlertPolicyAssociationCloneSharedTags(resource.Status.SystemTags),
	}
}

func clearTargetAlertPolicyAssociationProjectedStatus(resource *datasafev1beta1.TargetAlertPolicyAssociation) any {
	if resource == nil {
		return nil
	}
	baseline := resource.Status
	resource.Status = datasafev1beta1.TargetAlertPolicyAssociationStatus{OsokStatus: baseline.OsokStatus}
	return baseline
}

func restoreTargetAlertPolicyAssociationProjectedStatus(resource *datasafev1beta1.TargetAlertPolicyAssociation, baseline any) {
	if resource == nil {
		return
	}
	if status, ok := baseline.(datasafev1beta1.TargetAlertPolicyAssociationStatus); ok {
		osokStatus := resource.Status.OsokStatus
		resource.Status = status
		resource.Status.OsokStatus = osokStatus
	}
}

func targetAlertPolicyAssociationFromResponse(response any) (datasafesdk.TargetAlertPolicyAssociation, bool) {
	switch current := response.(type) {
	case datasafesdk.GetTargetAlertPolicyAssociationResponse:
		return current.TargetAlertPolicyAssociation, true
	case *datasafesdk.GetTargetAlertPolicyAssociationResponse:
		if current == nil {
			return datasafesdk.TargetAlertPolicyAssociation{}, false
		}
		return current.TargetAlertPolicyAssociation, true
	case datasafesdk.CreateTargetAlertPolicyAssociationResponse:
		return current.TargetAlertPolicyAssociation, true
	case *datasafesdk.CreateTargetAlertPolicyAssociationResponse:
		if current == nil {
			return datasafesdk.TargetAlertPolicyAssociation{}, false
		}
		return current.TargetAlertPolicyAssociation, true
	case datasafesdk.TargetAlertPolicyAssociation:
		return current, true
	case *datasafesdk.TargetAlertPolicyAssociation:
		if current == nil {
			return datasafesdk.TargetAlertPolicyAssociation{}, false
		}
		return *current, true
	default:
		return datasafesdk.TargetAlertPolicyAssociation{}, false
	}
}

func targetAlertPolicyAssociationSummaryFromResponse(response any) (datasafesdk.TargetAlertPolicyAssociationSummary, bool) {
	switch current := response.(type) {
	case datasafesdk.TargetAlertPolicyAssociationSummary:
		return current, true
	case *datasafesdk.TargetAlertPolicyAssociationSummary:
		if current == nil {
			return datasafesdk.TargetAlertPolicyAssociationSummary{}, false
		}
		return *current, true
	default:
		return datasafesdk.TargetAlertPolicyAssociationSummary{}, false
	}
}

func targetAlertPolicyAssociationStatusProjectionFromSDK(current datasafesdk.TargetAlertPolicyAssociation) targetAlertPolicyAssociationStatusProjection {
	return targetAlertPolicyAssociationStatusProjection{
		Id:               targetAlertPolicyAssociationStringValue(current.Id),
		CompartmentId:    targetAlertPolicyAssociationStringValue(current.CompartmentId),
		TimeCreated:      targetAlertPolicyAssociationSDKTimeString(current.TimeCreated),
		TimeUpdated:      targetAlertPolicyAssociationSDKTimeString(current.TimeUpdated),
		LifecycleState:   string(current.LifecycleState),
		DisplayName:      targetAlertPolicyAssociationStringValue(current.DisplayName),
		Description:      targetAlertPolicyAssociationStringValue(current.Description),
		PolicyId:         targetAlertPolicyAssociationStringValue(current.PolicyId),
		TargetId:         targetAlertPolicyAssociationStringValue(current.TargetId),
		IsEnabled:        targetAlertPolicyAssociationBoolValue(current.IsEnabled),
		LifecycleDetails: targetAlertPolicyAssociationStringValue(current.LifecycleDetails),
		FreeformTags:     targetAlertPolicyAssociationStringMap(current.FreeformTags),
		DefinedTags:      targetAlertPolicyAssociationSharedTags(current.DefinedTags),
		SystemTags:       targetAlertPolicyAssociationSharedTags(current.SystemTags),
	}
}

func targetAlertPolicyAssociationStatusProjectionFromSummary(current datasafesdk.TargetAlertPolicyAssociationSummary) targetAlertPolicyAssociationStatusProjection {
	return targetAlertPolicyAssociationStatusProjection{
		Id:               targetAlertPolicyAssociationStringValue(current.Id),
		CompartmentId:    targetAlertPolicyAssociationStringValue(current.CompartmentId),
		TimeCreated:      targetAlertPolicyAssociationSDKTimeString(current.TimeCreated),
		TimeUpdated:      targetAlertPolicyAssociationSDKTimeString(current.TimeUpdated),
		LifecycleState:   string(current.LifecycleState),
		DisplayName:      targetAlertPolicyAssociationStringValue(current.DisplayName),
		Description:      targetAlertPolicyAssociationStringValue(current.Description),
		PolicyId:         targetAlertPolicyAssociationStringValue(current.PolicyId),
		TargetId:         targetAlertPolicyAssociationStringValue(current.TargetId),
		IsEnabled:        targetAlertPolicyAssociationBoolValue(current.IsEnabled),
		LifecycleDetails: targetAlertPolicyAssociationStringValue(current.LifecycleDetails),
		FreeformTags:     targetAlertPolicyAssociationStringMap(current.FreeformTags),
		DefinedTags:      targetAlertPolicyAssociationSharedTags(current.DefinedTags),
	}
}

func getTargetAlertPolicyAssociationWorkRequest(
	ctx context.Context,
	client targetAlertPolicyAssociationOCIClient,
	initErr error,
	workRequestID string,
) (datasafesdk.WorkRequest, error) {
	if initErr != nil {
		return datasafesdk.WorkRequest{}, initErr
	}
	if client == nil {
		return datasafesdk.WorkRequest{}, fmt.Errorf("%s work request OCI client is not configured", targetAlertPolicyAssociationKind)
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return datasafesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveTargetAlertPolicyAssociationWorkRequestAction(workRequest any) (string, error) {
	current, ok := targetAlertPolicyAssociationWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", targetAlertPolicyAssociationKind, workRequest)
	}
	for _, resource := range current.Resources {
		if !isTargetAlertPolicyAssociationWorkRequestResource(resource) {
			continue
		}
		switch resource.ActionType {
		case datasafesdk.WorkRequestResourceActionTypeCreated,
			datasafesdk.WorkRequestResourceActionTypeUpdated,
			datasafesdk.WorkRequestResourceActionTypeDeleted:
			return string(resource.ActionType), nil
		}
	}
	if current.OperationType == datasafesdk.WorkRequestOperationTypePatchTargetAlertPolicyAssociation {
		return string(current.OperationType), nil
	}
	return "", nil
}

func recoverTargetAlertPolicyAssociationIDFromWorkRequest(
	_ *datasafev1beta1.TargetAlertPolicyAssociation,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, ok := targetAlertPolicyAssociationWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", targetAlertPolicyAssociationKind, workRequest)
	}
	for _, resource := range current.Resources {
		if !isTargetAlertPolicyAssociationWorkRequestResource(resource) {
			continue
		}
		if !targetAlertPolicyAssociationWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		if id := targetAlertPolicyAssociationStringValue(resource.Identifier); id != "" {
			return id, nil
		}
	}
	for _, resource := range current.Resources {
		if isTargetAlertPolicyAssociationWorkRequestResource(resource) {
			return targetAlertPolicyAssociationStringValue(resource.Identifier), nil
		}
	}
	return "", nil
}

func targetAlertPolicyAssociationWorkRequestFromAny(workRequest any) (datasafesdk.WorkRequest, bool) {
	switch current := workRequest.(type) {
	case datasafesdk.WorkRequest:
		return current, true
	case *datasafesdk.WorkRequest:
		if current == nil {
			return datasafesdk.WorkRequest{}, false
		}
		return *current, true
	default:
		return datasafesdk.WorkRequest{}, false
	}
}

func isTargetAlertPolicyAssociationWorkRequestResource(resource datasafesdk.WorkRequestResource) bool {
	entityType := normalizeTargetAlertPolicyAssociationToken(targetAlertPolicyAssociationStringValue(resource.EntityType))
	return entityType == "targetalertpolicyassociation" || strings.Contains(entityType, "targetalertpolicyassociation")
}

func targetAlertPolicyAssociationWorkRequestActionMatchesPhase(
	action datasafesdk.WorkRequestResourceActionTypeEnum,
	phase shared.OSOKAsyncPhase,
) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == datasafesdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return action == datasafesdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return action == datasafesdk.WorkRequestResourceActionTypeDeleted
	default:
		return false
	}
}

func normalizeTargetAlertPolicyAssociationToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", ".", "", " ", "")
	return replacer.Replace(value)
}

func currentTargetAlertPolicyAssociationID(resource *datasafev1beta1.TargetAlertPolicyAssociation) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func targetAlertPolicyAssociationOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func targetAlertPolicyAssociationStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func targetAlertPolicyAssociationBoolValue(value *bool) bool {
	return value != nil && *value
}

func targetAlertPolicyAssociationSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func targetAlertPolicyAssociationStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	return maps.Clone(source)
}

func targetAlertPolicyAssociationSharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
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

func targetAlertPolicyAssociationCloneSharedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
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

func targetAlertPolicyAssociationDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
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

var _ interface{ GetOpcRequestID() string } = targetAlertPolicyAssociationAmbiguousNotFoundError{}
