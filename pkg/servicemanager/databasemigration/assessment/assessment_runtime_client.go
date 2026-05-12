/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package assessment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

var assessmentWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(databasemigrationsdk.OperationStatusAccepted),
		string(databasemigrationsdk.OperationStatusInProgress),
		string(databasemigrationsdk.OperationStatusWaiting),
		string(databasemigrationsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(databasemigrationsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(databasemigrationsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(databasemigrationsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(databasemigrationsdk.OperationTypesCreateAssessment)},
	UpdateActionTokens:    []string{string(databasemigrationsdk.OperationTypesUpdateAssessment)},
	DeleteActionTokens:    []string{string(databasemigrationsdk.OperationTypesDeleteAssessment)},
}

var (
	pendingAssessmentCreateBodies sync.Map
	pendingAssessmentUpdateBodies sync.Map
	assessmentRequestBodySequence atomic.Uint64
)

type assessmentRequestBodyContextKey struct{}

type assessmentOCIClient interface {
	CreateAssessment(context.Context, databasemigrationsdk.CreateAssessmentRequest) (databasemigrationsdk.CreateAssessmentResponse, error)
	GetAssessment(context.Context, databasemigrationsdk.GetAssessmentRequest) (databasemigrationsdk.GetAssessmentResponse, error)
	ListAssessments(context.Context, databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error)
	UpdateAssessment(context.Context, databasemigrationsdk.UpdateAssessmentRequest) (databasemigrationsdk.UpdateAssessmentResponse, error)
	DeleteAssessment(context.Context, databasemigrationsdk.DeleteAssessmentRequest) (databasemigrationsdk.DeleteAssessmentResponse, error)
	GetWorkRequest(context.Context, databasemigrationsdk.GetWorkRequestRequest) (databasemigrationsdk.GetWorkRequestResponse, error)
}

type assessmentLookupIdentity struct {
	CompartmentID        string
	DisplayName          string
	DatabaseCombination  string
	ExactMatchFieldsJSON map[string]any
}

func init() {
	registerAssessmentRuntimeHooksMutator(func(manager *AssessmentServiceManager, hooks *AssessmentRuntimeHooks) {
		client, initErr := newAssessmentSDKClient(manager)
		applyAssessmentRuntimeHooks(hooks, client, initErr)
	})

	newAssessmentServiceClient = func(manager *AssessmentServiceManager) AssessmentServiceClient {
		config, hooks := newAssessmentGeneratedRuntimeConfig(manager, nil)
		delegate := defaultAssessmentServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*databasemigrationv1beta1.Assessment](config),
		}
		return wrapAssessmentGeneratedClient(hooks, delegate)
	}
}

func newAssessmentSDKClient(manager *AssessmentServiceManager) (assessmentOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Assessment service manager is nil")
	}

	client, err := databasemigrationsdk.NewDatabaseMigrationClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newAssessmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client assessmentOCIClient,
) AssessmentServiceClient {
	manager := &AssessmentServiceManager{Log: log}
	config, hooks := newAssessmentGeneratedRuntimeConfig(manager, client)
	delegate := defaultAssessmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*databasemigrationv1beta1.Assessment](config),
	}
	return wrapAssessmentGeneratedClient(hooks, delegate)
}

func newAssessmentGeneratedRuntimeConfig(
	manager *AssessmentServiceManager,
	client assessmentOCIClient,
) (generatedruntime.Config[*databasemigrationv1beta1.Assessment], AssessmentRuntimeHooks) {
	var initErr error
	if client == nil {
		client, initErr = newAssessmentSDKClient(manager)
	}

	hooks := newAssessmentRuntimeHooksWithOCIClient(client)
	applyAssessmentRuntimeHooks(&hooks, client, initErr)

	config := buildAssessmentGeneratedRuntimeConfig(manager, hooks)
	// Assessment list summaries do not expose enough shape to support the shared
	// generic bind-before-create matcher without risking false matches.
	config.List = nil
	if initErr != nil {
		config.InitError = fmt.Errorf("initialize Assessment OCI client: %w", initErr)
	}
	return config, hooks
}

func applyAssessmentRuntimeHooks(
	hooks *AssessmentRuntimeHooks,
	client assessmentOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedAssessmentRuntimeSemantics()
	hooks.Identity.Resolve = resolveAssessmentIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardAssessmentExistingBeforeCreate
	hooks.Identity.LookupExisting = func(
		ctx context.Context,
		resource *databasemigrationv1beta1.Assessment,
		identity any,
	) (any, error) {
		return lookupExistingAssessment(ctx, client, initErr, resource, identity)
	}
	hooks.Create.Fields = assessmentCreateFields()
	hooks.Get.Fields = assessmentGetFields()
	hooks.List.Fields = assessmentListFields()
	hooks.Update.Fields = assessmentUpdateFields()
	hooks.Delete.Fields = assessmentDeleteFields()
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *databasemigrationv1beta1.Assessment,
		namespace string,
	) (any, error) {
		return buildAssessmentCreateBody(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *databasemigrationv1beta1.Assessment,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildAssessmentUpdateBody(ctx, resource, namespace, currentResponse)
	}
	hooks.Async.Adapter = assessmentWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getAssessmentWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveAssessmentGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveAssessmentGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverAssessmentIDFromGeneratedWorkRequest
	hooks.Async.Message = assessmentGeneratedWorkRequestMessage
	wrapAssessmentRequestBodyContext(hooks)
	wrapAssessmentRequestBodies(hooks)
}

func newAssessmentRuntimeHooksWithOCIClient(client assessmentOCIClient) AssessmentRuntimeHooks {
	return AssessmentRuntimeHooks{
		Semantics:       reviewedAssessmentRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*databasemigrationv1beta1.Assessment]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*databasemigrationv1beta1.Assessment]{},
		StatusHooks:     generatedruntime.StatusHooks[*databasemigrationv1beta1.Assessment]{},
		ParityHooks:     generatedruntime.ParityHooks[*databasemigrationv1beta1.Assessment]{},
		Async:           generatedruntime.AsyncHooks[*databasemigrationv1beta1.Assessment]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*databasemigrationv1beta1.Assessment]{},
		Create: runtimeOperationHooks[databasemigrationsdk.CreateAssessmentRequest, databasemigrationsdk.CreateAssessmentResponse]{
			Fields: assessmentCreateFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.CreateAssessmentRequest) (databasemigrationsdk.CreateAssessmentResponse, error) {
				if client == nil {
					return databasemigrationsdk.CreateAssessmentResponse{}, fmt.Errorf("Assessment OCI client is not configured")
				}
				return client.CreateAssessment(ctx, request)
			},
		},
		Get: runtimeOperationHooks[databasemigrationsdk.GetAssessmentRequest, databasemigrationsdk.GetAssessmentResponse]{
			Fields: assessmentGetFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.GetAssessmentRequest) (databasemigrationsdk.GetAssessmentResponse, error) {
				if client == nil {
					return databasemigrationsdk.GetAssessmentResponse{}, fmt.Errorf("Assessment OCI client is not configured")
				}
				return client.GetAssessment(ctx, request)
			},
		},
		List: runtimeOperationHooks[databasemigrationsdk.ListAssessmentsRequest, databasemigrationsdk.ListAssessmentsResponse]{
			Fields: assessmentListFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.ListAssessmentsRequest) (databasemigrationsdk.ListAssessmentsResponse, error) {
				if client == nil {
					return databasemigrationsdk.ListAssessmentsResponse{}, fmt.Errorf("Assessment OCI client is not configured")
				}
				return client.ListAssessments(ctx, request)
			},
		},
		Update: runtimeOperationHooks[databasemigrationsdk.UpdateAssessmentRequest, databasemigrationsdk.UpdateAssessmentResponse]{
			Fields: assessmentUpdateFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.UpdateAssessmentRequest) (databasemigrationsdk.UpdateAssessmentResponse, error) {
				if client == nil {
					return databasemigrationsdk.UpdateAssessmentResponse{}, fmt.Errorf("Assessment OCI client is not configured")
				}
				return client.UpdateAssessment(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[databasemigrationsdk.DeleteAssessmentRequest, databasemigrationsdk.DeleteAssessmentResponse]{
			Fields: assessmentDeleteFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.DeleteAssessmentRequest) (databasemigrationsdk.DeleteAssessmentResponse, error) {
				if client == nil {
					return databasemigrationsdk.DeleteAssessmentResponse{}, fmt.Errorf("Assessment OCI client is not configured")
				}
				return client.DeleteAssessment(ctx, request)
			},
		},
		WrapGeneratedClient: []func(AssessmentServiceClient) AssessmentServiceClient{},
	}
}

func reviewedAssessmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "databasemigration",
		FormalSlug:    "assessment",
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
			ProvisioningStates: []string{
				string(databasemigrationsdk.AssessmentLifecycleStatesCreating),
				string(databasemigrationsdk.AssessmentLifecycleStatesInProgress),
			},
			UpdatingStates: []string{string(databasemigrationsdk.AssessmentLifecycleStatesUpdating)},
			ActiveStates: []string{
				string(databasemigrationsdk.AssessmentLifecycleStatesActive),
				string(databasemigrationsdk.AssessmentLifecycleStatesSucceeded),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(databasemigrationsdk.AssessmentLifecycleStatesDeleting)},
			TerminalStates: []string{string(databasemigrationsdk.AssessmentLifecycleStatesDeleted)},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"acceptableDowntime",
				"creationType",
				"databaseDataSize",
				"ddlExpectation",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"networkSpeedMegabitPerSecond",
				"sourceDatabaseConnection",
				"targetDatabaseConnection",
			},
			ForceNew: []string{
				"bulkIncludeExcludeData",
				"compartmentId",
				"databaseCombination",
				"excludeObjects",
				"includeObjects",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "assessment", Action: "CREATED"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "assessment", Action: "UPDATED"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "assessment", Action: "DELETED"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAssessment",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "assessment", Action: "CREATED"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAssessment",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "assessment", Action: "UPDATED"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAssessment/ListAssessments confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "assessment", Action: "DELETED"}},
		},
		AuxiliaryOperations: nil,
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func assessmentCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateAssessmentDetails", RequestName: "CreateAssessmentDetails", Contribution: "body"},
	}
}

func assessmentGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AssessmentId", RequestName: "assessmentId", Contribution: "path", PreferResourceID: true},
	}
}

func assessmentListFields() []generatedruntime.RequestField {
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
	}
}

func assessmentUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AssessmentId", RequestName: "assessmentId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateAssessmentDetails", RequestName: "UpdateAssessmentDetails", Contribution: "body"},
	}
}

func assessmentDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AssessmentId", RequestName: "assessmentId", Contribution: "path", PreferResourceID: true},
	}
}

func resolveAssessmentIdentity(resource *databasemigrationv1beta1.Assessment) (any, error) {
	if resource == nil {
		return nil, nil
	}

	identity := assessmentLookupIdentity{
		CompartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		DisplayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
	if combination, ok := normalizeAssessmentDatabaseCombination(resource.Spec.DatabaseCombination); ok {
		identity.DatabaseCombination = combination
	}

	exactMatchFields := map[string]any{}
	if identity.CompartmentID != "" {
		exactMatchFields["compartmentId"] = identity.CompartmentID
	}
	if identity.DisplayName != "" {
		exactMatchFields["displayName"] = identity.DisplayName
	}
	if identity.DatabaseCombination != "" {
		exactMatchFields["databaseCombination"] = identity.DatabaseCombination
	}
	if sourceID := strings.TrimSpace(resource.Spec.SourceDatabaseConnection.Id); sourceID != "" {
		exactMatchFields["sourceDatabaseConnection"] = map[string]any{"id": sourceID}
	}
	if targetValues, err := assessmentJSONMap(resource.Spec.TargetDatabaseConnection); err != nil {
		return nil, fmt.Errorf("project Assessment target identity: %w", err)
	} else if len(targetValues) > 0 {
		exactMatchFields["targetDatabaseConnection"] = targetValues
	}
	if value := strings.TrimSpace(resource.Spec.NetworkSpeedMegabitPerSecond); value != "" {
		exactMatchFields["networkSpeedMegabitPerSecond"] = value
	}
	if value := strings.TrimSpace(resource.Spec.AcceptableDowntime); value != "" {
		exactMatchFields["acceptableDowntime"] = value
	}
	if value := strings.TrimSpace(resource.Spec.DatabaseDataSize); value != "" {
		exactMatchFields["databaseDataSize"] = value
	}
	if value := strings.TrimSpace(resource.Spec.DdlExpectation); value != "" {
		exactMatchFields["ddlExpectation"] = value
	}
	if value := strings.TrimSpace(resource.Spec.CreationType); value != "" {
		exactMatchFields["creationType"] = value
	}
	identity.ExactMatchFieldsJSON = exactMatchFields
	return identity, nil
}

func guardAssessmentExistingBeforeCreate(
	_ context.Context,
	resource *databasemigrationv1beta1.Assessment,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if _, ok := normalizeAssessmentDatabaseCombination(resource.Spec.DatabaseCombination); !ok {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupExistingAssessment(
	ctx context.Context,
	client assessmentOCIClient,
	initErr error,
	resource *databasemigrationv1beta1.Assessment,
	identity any,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("Assessment resource is nil")
	}
	if initErr != nil {
		return nil, fmt.Errorf("initialize Assessment OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Assessment OCI client is not configured")
	}

	lookupIdentity, ok := identity.(assessmentLookupIdentity)
	if !ok {
		return nil, fmt.Errorf("unexpected Assessment identity payload %T", identity)
	}
	if lookupIdentity.CompartmentID == "" || lookupIdentity.DisplayName == "" || lookupIdentity.DatabaseCombination == "" {
		return nil, nil
	}

	summaries, err := listAssessmentsAllPages(ctx, client, databasemigrationsdk.ListAssessmentsRequest{
		CompartmentId: common.String(lookupIdentity.CompartmentID),
		DisplayName:   common.String(lookupIdentity.DisplayName),
	})
	if err != nil {
		return nil, err
	}

	var exactMatch any
	for _, summary := range summaries {
		if !assessmentSummaryMatchesLookup(summary, lookupIdentity) {
			continue
		}

		assessmentID := assessmentIDFromAny(summary)
		if assessmentID == "" {
			continue
		}

		response, err := client.GetAssessment(ctx, databasemigrationsdk.GetAssessmentRequest{
			AssessmentId: common.String(assessmentID),
		})
		if err != nil {
			return nil, err
		}
		if !assessmentMatchesLookupIdentity(response.Assessment, lookupIdentity) {
			continue
		}
		if exactMatch != nil {
			return nil, fmt.Errorf("Assessment pre-create lookup found multiple exact matches for displayName %q", lookupIdentity.DisplayName)
		}
		exactMatch = response
	}

	return exactMatch, nil
}

func listAssessmentsAllPages(
	ctx context.Context,
	client assessmentOCIClient,
	request databasemigrationsdk.ListAssessmentsRequest,
) ([]databasemigrationsdk.AssessmentSummary, error) {
	var items []databasemigrationsdk.AssessmentSummary
	for {
		response, err := client.ListAssessments(ctx, request)
		if err != nil {
			return nil, err
		}
		items = append(items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return items, nil
		}
		request.Page = response.OpcNextPage
	}
}

func buildAssessmentCreateDetails(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
) (databasemigrationsdk.CreateAssessmentDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("Assessment resource is nil")
	}
	if err := validateAssessmentUnsupportedSpec(resource); err != nil {
		return nil, err
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return nil, err
	}

	combination, err := assessmentDatabaseCombinationForResource(resource, nil)
	if err != nil {
		return nil, err
	}
	if err := validateAssessmentRequiredFields(resource); err != nil {
		return nil, err
	}

	return decodeAssessmentCreateDetails(combination, resolvedSpec)
}

func buildAssessmentCreateBody(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
) (any, error) {
	details, err := buildAssessmentCreateDetails(ctx, resource, namespace)
	if err != nil {
		return nil, err
	}
	if err := stashAssessmentCreateBody(ctx, resource, details); err != nil {
		return nil, err
	}
	return nil, nil
}

func buildAssessmentUpdateDetails(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
	currentResponse any,
) (databasemigrationsdk.UpdateAssessmentDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("Assessment resource is nil")
	}
	if err := validateAssessmentUnsupportedSpec(resource); err != nil {
		return nil, false, err
	}

	combination, err := assessmentDatabaseCombinationForResource(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}

	currentDetails, err := assessmentCurrentUpdateDetails(combination, currentResponse)
	if err != nil {
		return nil, false, err
	}
	currentState, err := assessmentCurrentUpdateState(currentDetails)
	if err != nil {
		return nil, false, err
	}

	desiredState, updateNeeded, err := desiredAssessmentUpdateState(ctx, resource, namespace, currentState)
	if err != nil {
		return nil, false, err
	}
	desiredDetails, err := assessmentUpdateDetailsFromState(combination, desiredState)
	if err != nil {
		return nil, false, err
	}
	return desiredDetails, updateNeeded, nil
}

func buildAssessmentUpdateBody(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
	currentResponse any,
) (any, bool, error) {
	details, updateNeeded, err := buildAssessmentUpdateDetails(ctx, resource, namespace, currentResponse)
	if err != nil {
		return nil, false, err
	}
	if !updateNeeded {
		return nil, false, nil
	}
	if err := stashAssessmentUpdateBody(ctx, resource, details); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

func assessmentCurrentUpdateDetails(
	databaseCombination string,
	currentResponse any,
) (databasemigrationsdk.UpdateAssessmentDetails, error) {
	body, err := assessmentRuntimeBody(currentResponse)
	if err != nil {
		return nil, err
	}
	return decodeAssessmentUpdateDetails(databaseCombination, body)
}

type assessmentUpdateState struct {
	Description                  *string
	DisplayName                  *string
	SourceDatabaseConnection     *databasemigrationsdk.SourceAssessmentConnection
	TargetDatabaseConnection     *databasemigrationsdk.TargetAssessmentConnection
	FreeformTags                 map[string]string
	DefinedTags                  map[string]map[string]interface{}
	NetworkSpeedMegabitPerSecond databasemigrationsdk.NetworkSpeedMegabitPerSecondEnum
	AcceptableDowntime           databasemigrationsdk.AcceptableDowntimeEnum
	DatabaseDataSize             databasemigrationsdk.DatabaseDataSizeEnum
	DdlExpectation               databasemigrationsdk.DdlExpectationEnum
	CreationType                 databasemigrationsdk.CreationTypeEnum
}

func assessmentCurrentUpdateState(
	details databasemigrationsdk.UpdateAssessmentDetails,
) (assessmentUpdateState, error) {
	switch current := details.(type) {
	case databasemigrationsdk.UpdateMySqlAssessmentDetails:
		return assessmentUpdateState{
			Description:                  current.Description,
			DisplayName:                  current.DisplayName,
			SourceDatabaseConnection:     current.SourceDatabaseConnection,
			TargetDatabaseConnection:     current.TargetDatabaseConnection,
			FreeformTags:                 cloneAssessmentStringMap(current.FreeformTags),
			DefinedTags:                  cloneAssessmentDefinedTags(current.DefinedTags),
			NetworkSpeedMegabitPerSecond: current.NetworkSpeedMegabitPerSecond,
			AcceptableDowntime:           current.AcceptableDowntime,
			DatabaseDataSize:             current.DatabaseDataSize,
			DdlExpectation:               current.DdlExpectation,
			CreationType:                 current.CreationType,
		}, nil
	case databasemigrationsdk.UpdateOracleAssessmentDetails:
		return assessmentUpdateState{
			Description:                  current.Description,
			DisplayName:                  current.DisplayName,
			SourceDatabaseConnection:     current.SourceDatabaseConnection,
			TargetDatabaseConnection:     current.TargetDatabaseConnection,
			FreeformTags:                 cloneAssessmentStringMap(current.FreeformTags),
			DefinedTags:                  cloneAssessmentDefinedTags(current.DefinedTags),
			NetworkSpeedMegabitPerSecond: current.NetworkSpeedMegabitPerSecond,
			AcceptableDowntime:           current.AcceptableDowntime,
			DatabaseDataSize:             current.DatabaseDataSize,
			DdlExpectation:               current.DdlExpectation,
			CreationType:                 current.CreationType,
		}, nil
	default:
		return assessmentUpdateState{}, fmt.Errorf("unexpected Assessment update details type %T", details)
	}
}

func desiredAssessmentUpdateState(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
	current assessmentUpdateState,
) (assessmentUpdateState, bool, error) {
	desired := assessmentUpdateState{}
	updateNeeded := false

	if next, ok := desiredAssessmentStringUpdate(resource.Spec.Description, current.Description); ok {
		desired.Description = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		desired.DisplayName = next
		updateNeeded = true
	}
	if next, ok, err := desiredAssessmentSourceConnectionUpdate(ctx, resource, namespace, current.SourceDatabaseConnection); err != nil {
		return assessmentUpdateState{}, false, err
	} else if ok {
		desired.SourceDatabaseConnection = next
		updateNeeded = true
	}
	if next, ok, err := desiredAssessmentTargetConnectionUpdate(ctx, resource, namespace, current.TargetDatabaseConnection); err != nil {
		return assessmentUpdateState{}, false, err
	} else if ok {
		desired.TargetDatabaseConnection = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		desired.FreeformTags = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		desired.DefinedTags = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentNetworkSpeedUpdate(resource.Spec.NetworkSpeedMegabitPerSecond, current.NetworkSpeedMegabitPerSecond); ok {
		desired.NetworkSpeedMegabitPerSecond = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentAcceptableDowntimeUpdate(resource.Spec.AcceptableDowntime, current.AcceptableDowntime); ok {
		desired.AcceptableDowntime = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentDatabaseDataSizeUpdate(resource.Spec.DatabaseDataSize, current.DatabaseDataSize); ok {
		desired.DatabaseDataSize = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentDdlExpectationUpdate(resource.Spec.DdlExpectation, current.DdlExpectation); ok {
		desired.DdlExpectation = next
		updateNeeded = true
	}
	if next, ok := desiredAssessmentCreationTypeUpdate(resource.Spec.CreationType, current.CreationType); ok {
		desired.CreationType = next
		updateNeeded = true
	}
	return desired, updateNeeded, nil
}

func assessmentUpdateDetailsFromState(
	databaseCombination string,
	state assessmentUpdateState,
) (databasemigrationsdk.UpdateAssessmentDetails, error) {
	switch databaseCombination {
	case "MYSQL":
		return databasemigrationsdk.UpdateMySqlAssessmentDetails{
			Description:                  state.Description,
			DisplayName:                  state.DisplayName,
			SourceDatabaseConnection:     state.SourceDatabaseConnection,
			TargetDatabaseConnection:     state.TargetDatabaseConnection,
			FreeformTags:                 cloneAssessmentStringMap(state.FreeformTags),
			DefinedTags:                  cloneAssessmentDefinedTags(state.DefinedTags),
			NetworkSpeedMegabitPerSecond: state.NetworkSpeedMegabitPerSecond,
			AcceptableDowntime:           state.AcceptableDowntime,
			DatabaseDataSize:             state.DatabaseDataSize,
			DdlExpectation:               state.DdlExpectation,
			CreationType:                 state.CreationType,
		}, nil
	case "ORACLE":
		return databasemigrationsdk.UpdateOracleAssessmentDetails{
			Description:                  state.Description,
			DisplayName:                  state.DisplayName,
			SourceDatabaseConnection:     state.SourceDatabaseConnection,
			TargetDatabaseConnection:     state.TargetDatabaseConnection,
			FreeformTags:                 cloneAssessmentStringMap(state.FreeformTags),
			DefinedTags:                  cloneAssessmentDefinedTags(state.DefinedTags),
			NetworkSpeedMegabitPerSecond: state.NetworkSpeedMegabitPerSecond,
			AcceptableDowntime:           state.AcceptableDowntime,
			DatabaseDataSize:             state.DatabaseDataSize,
			DdlExpectation:               state.DdlExpectation,
			CreationType:                 state.CreationType,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported Assessment databaseCombination %q", databaseCombination)
	}
}

func desiredAssessmentStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func desiredAssessmentSourceConnectionUpdate(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
	current *databasemigrationsdk.SourceAssessmentConnection,
) (*databasemigrationsdk.SourceAssessmentConnection, bool, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return nil, false, err
	}
	specMap, err := assessmentJSONMap(resolvedSpec)
	if err != nil {
		return nil, false, err
	}
	rawSource, ok := specMap["sourceDatabaseConnection"]
	if !ok {
		return nil, false, nil
	}

	var desired databasemigrationsdk.SourceAssessmentConnection
	if err := assessmentDecodeJSONObject(rawSource, &desired); err != nil {
		return nil, false, fmt.Errorf("decode Assessment sourceDatabaseConnection: %w", err)
	}

	currentMap, err := assessmentJSONMap(current)
	if err != nil {
		return nil, false, err
	}
	desiredMap, err := assessmentJSONMap(desired)
	if err != nil {
		return nil, false, err
	}
	if assessmentMapSubsetEqual(desiredMap, currentMap) {
		return nil, false, nil
	}
	return &desired, true, nil
}

func desiredAssessmentTargetConnectionUpdate(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	namespace string,
	current *databasemigrationsdk.TargetAssessmentConnection,
) (*databasemigrationsdk.TargetAssessmentConnection, bool, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return nil, false, err
	}
	specMap, err := assessmentJSONMap(resolvedSpec)
	if err != nil {
		return nil, false, err
	}
	rawTarget, ok := specMap["targetDatabaseConnection"]
	if !ok {
		return nil, false, nil
	}

	var desired databasemigrationsdk.TargetAssessmentConnection
	if err := assessmentDecodeJSONObject(rawTarget, &desired); err != nil {
		return nil, false, fmt.Errorf("decode Assessment targetDatabaseConnection: %w", err)
	}

	currentMap, err := assessmentJSONMap(current)
	if err != nil {
		return nil, false, err
	}
	desiredMap, err := assessmentJSONMap(desired)
	if err != nil {
		return nil, false, err
	}
	if assessmentMapSubsetEqual(desiredMap, currentMap) {
		return nil, false, nil
	}
	return &desired, true, nil
}

func desiredAssessmentFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if assessmentValuesEqual(spec, current) {
		return nil, false
	}
	return cloneAssessmentStringMap(spec), true
}

func desiredAssessmentDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := assessmentDefinedTagsFromSpec(spec)
	if assessmentValuesEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func desiredAssessmentNetworkSpeedUpdate(
	spec string,
	current databasemigrationsdk.NetworkSpeedMegabitPerSecondEnum,
) (databasemigrationsdk.NetworkSpeedMegabitPerSecondEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return databasemigrationsdk.NetworkSpeedMegabitPerSecondEnum(spec), true
}

func desiredAssessmentAcceptableDowntimeUpdate(
	spec string,
	current databasemigrationsdk.AcceptableDowntimeEnum,
) (databasemigrationsdk.AcceptableDowntimeEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return databasemigrationsdk.AcceptableDowntimeEnum(spec), true
}

func desiredAssessmentDatabaseDataSizeUpdate(
	spec string,
	current databasemigrationsdk.DatabaseDataSizeEnum,
) (databasemigrationsdk.DatabaseDataSizeEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return databasemigrationsdk.DatabaseDataSizeEnum(spec), true
}

func desiredAssessmentDdlExpectationUpdate(
	spec string,
	current databasemigrationsdk.DdlExpectationEnum,
) (databasemigrationsdk.DdlExpectationEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return databasemigrationsdk.DdlExpectationEnum(spec), true
}

func desiredAssessmentCreationTypeUpdate(
	spec string,
	current databasemigrationsdk.CreationTypeEnum,
) (databasemigrationsdk.CreationTypeEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return databasemigrationsdk.CreationTypeEnum(spec), true
}

func validateAssessmentUnsupportedSpec(resource *databasemigrationv1beta1.Assessment) error {
	if resource == nil {
		return fmt.Errorf("Assessment resource is nil")
	}

	var unsupported []string
	if len(resource.Spec.ExcludeObjects) > 0 {
		unsupported = append(unsupported, "spec.excludeObjects")
	}
	if len(resource.Spec.IncludeObjects) > 0 {
		unsupported = append(unsupported, "spec.includeObjects")
	}
	if strings.TrimSpace(resource.Spec.BulkIncludeExcludeData) != "" {
		unsupported = append(unsupported, "spec.bulkIncludeExcludeData")
	}
	if len(unsupported) == 0 {
		return nil
	}

	return fmt.Errorf(
		"Assessment object helper fields are out of scope for the published runtime: %s",
		strings.Join(unsupported, ", "),
	)
}

func validateAssessmentRequiredFields(resource *databasemigrationv1beta1.Assessment) error {
	var missing []string
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		missing = append(missing, "spec.compartmentId")
	}
	if strings.TrimSpace(resource.Spec.DatabaseCombination) == "" {
		missing = append(missing, "spec.databaseCombination")
	}
	if strings.TrimSpace(resource.Spec.NetworkSpeedMegabitPerSecond) == "" {
		missing = append(missing, "spec.networkSpeedMegabitPerSecond")
	}
	if strings.TrimSpace(resource.Spec.AcceptableDowntime) == "" {
		missing = append(missing, "spec.acceptableDowntime")
	}
	if strings.TrimSpace(resource.Spec.DatabaseDataSize) == "" {
		missing = append(missing, "spec.databaseDataSize")
	}
	if strings.TrimSpace(resource.Spec.DdlExpectation) == "" {
		missing = append(missing, "spec.ddlExpectation")
	}
	if strings.TrimSpace(resource.Spec.SourceDatabaseConnection.Id) == "" {
		missing = append(missing, "spec.sourceDatabaseConnection.id")
	}
	if values, err := assessmentJSONMap(resource.Spec.TargetDatabaseConnection); err != nil {
		return fmt.Errorf("project Assessment target connection: %w", err)
	} else if len(values) == 0 {
		missing = append(missing, "spec.targetDatabaseConnection")
	}
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("Assessment requires %s", strings.Join(missing, ", "))
}

func assessmentDatabaseCombinationForResource(
	resource *databasemigrationv1beta1.Assessment,
	currentResponse any,
) (string, error) {
	if resource != nil {
		if combination, ok := normalizeAssessmentDatabaseCombination(resource.Spec.DatabaseCombination); ok {
			return combination, nil
		}
	}
	if currentResponse != nil {
		body, err := assessmentRuntimeBody(currentResponse)
		if err != nil {
			return "", err
		}
		if combination, ok := normalizeAssessmentDatabaseCombination(assessmentDatabaseCombinationFromRuntimeBody(body)); ok {
			return combination, nil
		}
	}
	return "", fmt.Errorf("Assessment spec.databaseCombination must be set to MYSQL or ORACLE")
}

func normalizeAssessmentDatabaseCombination(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "MYSQL":
		return "MYSQL", true
	case "ORACLE":
		return "ORACLE", true
	default:
		return "", false
	}
}

func decodeAssessmentCreateDetails(
	databaseCombination string,
	raw any,
) (databasemigrationsdk.CreateAssessmentDetails, error) {
	switch databaseCombination {
	case "MYSQL":
		details, err := decodeAssessmentConcrete[databasemigrationsdk.CreateMySqlAssessmentDetails](raw)
		return details, err
	case "ORACLE":
		details, err := decodeAssessmentConcrete[databasemigrationsdk.CreateOracleAssessmentDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported Assessment databaseCombination %q", databaseCombination)
	}
}

func decodeAssessmentUpdateDetails(
	databaseCombination string,
	raw any,
) (databasemigrationsdk.UpdateAssessmentDetails, error) {
	switch databaseCombination {
	case "MYSQL":
		details, err := decodeAssessmentConcrete[databasemigrationsdk.UpdateMySqlAssessmentDetails](raw)
		return details, err
	case "ORACLE":
		details, err := decodeAssessmentConcrete[databasemigrationsdk.UpdateOracleAssessmentDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported Assessment databaseCombination %q", databaseCombination)
	}
}

func decodeAssessmentConcrete[T any](raw any) (T, error) {
	var decoded T

	payload, err := json.Marshal(raw)
	if err != nil {
		return decoded, fmt.Errorf("marshal Assessment payload: %w", err)
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return decoded, fmt.Errorf("unmarshal Assessment payload: %w", err)
	}
	return decoded, nil
}

func assessmentRuntimeBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case databasemigrationsdk.CreateAssessmentResponse:
		return current.Assessment, nil
	case *databasemigrationsdk.CreateAssessmentResponse:
		if current == nil {
			return nil, fmt.Errorf("current Assessment response is nil")
		}
		return current.Assessment, nil
	case databasemigrationsdk.GetAssessmentResponse:
		return current.Assessment, nil
	case *databasemigrationsdk.GetAssessmentResponse:
		if current == nil {
			return nil, fmt.Errorf("current Assessment response is nil")
		}
		return current.Assessment, nil
	case databasemigrationsdk.Assessment:
		if current == nil {
			return nil, fmt.Errorf("current Assessment body is nil")
		}
		return current, nil
	case databasemigrationsdk.AssessmentSummary:
		if current == nil {
			return nil, fmt.Errorf("current Assessment summary is nil")
		}
		return current, nil
	case databasemigrationsdk.MySqlAssessment,
		databasemigrationsdk.MySqlAssessmentSummary,
		databasemigrationsdk.OracleAssessment,
		databasemigrationsdk.OracleAssessmentSummary:
		return current, nil
	default:
		return nil, fmt.Errorf("unsupported current Assessment payload type %T", currentResponse)
	}
}

func assessmentDatabaseCombinationFromRuntimeBody(body any) string {
	switch body.(type) {
	case databasemigrationsdk.MySqlAssessment, databasemigrationsdk.MySqlAssessmentSummary:
		return "MYSQL"
	case databasemigrationsdk.OracleAssessment, databasemigrationsdk.OracleAssessmentSummary:
		return "ORACLE"
	default:
		values, err := assessmentJSONMap(body)
		if err != nil {
			return ""
		}
		raw, _ := values["databaseCombination"].(string)
		return raw
	}
}

func assessmentSummaryMatchesLookup(
	summary databasemigrationsdk.AssessmentSummary,
	identity assessmentLookupIdentity,
) bool {
	if summary == nil {
		return false
	}
	if strings.TrimSpace(assessmentStringPtr(summary.GetDisplayName())) != identity.DisplayName {
		return false
	}
	if strings.TrimSpace(assessmentStringPtr(summary.GetCompartmentId())) != identity.CompartmentID {
		return false
	}
	if combination, ok := normalizeAssessmentDatabaseCombination(assessmentDatabaseCombinationFromRuntimeBody(summary)); !ok || combination != identity.DatabaseCombination {
		return false
	}
	return true
}

func assessmentMatchesLookupIdentity(body any, identity assessmentLookupIdentity) bool {
	values, err := assessmentJSONMap(body)
	if err != nil {
		return false
	}
	return assessmentMapSubsetEqual(identity.ExactMatchFieldsJSON, values)
}

func assessmentIDFromAny(body any) string {
	values, err := assessmentJSONMap(body)
	if err != nil {
		return ""
	}
	raw, _ := values["id"].(string)
	return strings.TrimSpace(raw)
}

func assessmentJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func assessmentMapSubsetEqual(desired map[string]any, current map[string]any) bool {
	for key, desiredValue := range desired {
		currentValue, ok := current[key]
		if !ok || !assessmentValueSubsetEqual(desiredValue, currentValue) {
			return false
		}
	}
	return true
}

func assessmentValueSubsetEqual(desired any, current any) bool {
	desiredMap, desiredIsMap := desired.(map[string]any)
	currentMap, currentIsMap := current.(map[string]any)
	if desiredIsMap || currentIsMap {
		if !desiredIsMap || !currentIsMap {
			return false
		}
		return assessmentMapSubsetEqual(desiredMap, currentMap)
	}

	desiredSlice, desiredIsSlice := desired.([]any)
	currentSlice, currentIsSlice := current.([]any)
	if desiredIsSlice || currentIsSlice {
		if !desiredIsSlice || !currentIsSlice || len(desiredSlice) != len(currentSlice) {
			return false
		}
		for i := range desiredSlice {
			if !assessmentValueSubsetEqual(desiredSlice[i], currentSlice[i]) {
				return false
			}
		}
		return true
	}

	return assessmentValuesEqual(desired, current)
}

func assessmentValuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return string(leftPayload) == string(rightPayload)
}

func getAssessmentWorkRequest(
	ctx context.Context,
	client assessmentOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Assessment OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Assessment OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, databasemigrationsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveAssessmentGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := assessmentWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveAssessmentGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := assessmentWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := assessmentWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverAssessmentIDFromGeneratedWorkRequest(
	_ *databasemigrationv1beta1.Assessment,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := assessmentWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := assessmentWorkRequestActionForPhase(phase)
	if id, ok := resolveAssessmentIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveAssessmentIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Assessment work request %s does not expose an assessment identifier", assessmentStringValue(current.Id))
}

func assessmentGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := assessmentWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Assessment %s work request %s is %s", phase, assessmentStringValue(current.Id), current.Status)
}

func assessmentWorkRequestFromAny(workRequest any) (databasemigrationsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case databasemigrationsdk.WorkRequest:
		return current, nil
	case *databasemigrationsdk.WorkRequest:
		if current == nil {
			return databasemigrationsdk.WorkRequest{}, fmt.Errorf("Assessment work request is nil")
		}
		return *current, nil
	default:
		return databasemigrationsdk.WorkRequest{}, fmt.Errorf("unexpected Assessment work request type %T", workRequest)
	}
}

func assessmentWorkRequestPhaseFromOperationType(
	operationType databasemigrationsdk.OperationTypesEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case databasemigrationsdk.OperationTypesCreateAssessment:
		return shared.OSOKAsyncPhaseCreate, true
	case databasemigrationsdk.OperationTypesUpdateAssessment:
		return shared.OSOKAsyncPhaseUpdate, true
	case databasemigrationsdk.OperationTypesDeleteAssessment:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func assessmentWorkRequestActionForPhase(
	phase shared.OSOKAsyncPhase,
) databasemigrationsdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return databasemigrationsdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return databasemigrationsdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return databasemigrationsdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func resolveAssessmentIDFromResources(
	resources []databasemigrationsdk.WorkRequestResource,
	action databasemigrationsdk.WorkRequestResourceActionTypeEnum,
	preferAssessmentOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferAssessmentOnly && !isAssessmentWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(assessmentStringValue(resource.Identifier))
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

func isAssessmentWorkRequestResource(resource databasemigrationsdk.WorkRequestResource) bool {
	return normalizeAssessmentWorkRequestToken(assessmentStringValue(resource.EntityType)) == "assessment"
}

func normalizeAssessmentWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func assessmentStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func assessmentStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func wrapAssessmentRequestBodyContext(hooks *AssessmentRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AssessmentServiceClient) AssessmentServiceClient {
		return assessmentRequestBodyContextClient{delegate: delegate}
	})
}

type assessmentRequestBodyContextClient struct {
	delegate AssessmentServiceClient
}

func (c assessmentRequestBodyContextClient) CreateOrUpdate(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	ctx = withAssessmentRequestBodyToken(ctx)
	defer clearAssessmentRequestBodies(ctx)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c assessmentRequestBodyContextClient) Delete(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func withAssessmentRequestBodyToken(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if assessmentRequestBodyToken(ctx) != "" {
		return ctx
	}
	token := fmt.Sprintf("assessment-request-%d", assessmentRequestBodySequence.Add(1))
	return context.WithValue(ctx, assessmentRequestBodyContextKey{}, token)
}

func assessmentRequestBodyToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(assessmentRequestBodyContextKey{}).(string)
	return strings.TrimSpace(value)
}

func clearAssessmentRequestBodies(ctx context.Context) {
	key := assessmentRequestBodyToken(ctx)
	if key == "" {
		return
	}
	pendingAssessmentCreateBodies.Delete(key)
	pendingAssessmentUpdateBodies.Delete(key)
}

func wrapAssessmentRequestBodies(hooks *AssessmentRuntimeHooks) {
	if hooks == nil {
		return
	}

	createCall := hooks.Create.Call
	if createCall != nil {
		hooks.Create.Call = func(ctx context.Context, request databasemigrationsdk.CreateAssessmentRequest) (databasemigrationsdk.CreateAssessmentResponse, error) {
			if request.CreateAssessmentDetails == nil {
				body, err := takeAssessmentCreateBody(ctx, request.OpcRetryToken)
				if err != nil {
					return databasemigrationsdk.CreateAssessmentResponse{}, err
				}
				request.CreateAssessmentDetails = body
			}
			return createCall(ctx, request)
		}
	}

	updateCall := hooks.Update.Call
	if updateCall != nil {
		hooks.Update.Call = func(ctx context.Context, request databasemigrationsdk.UpdateAssessmentRequest) (databasemigrationsdk.UpdateAssessmentResponse, error) {
			if request.UpdateAssessmentDetails == nil {
				body, err := takeAssessmentUpdateBody(ctx, request.AssessmentId)
				if err != nil {
					return databasemigrationsdk.UpdateAssessmentResponse{}, err
				}
				request.UpdateAssessmentDetails = body
			}
			return updateCall(ctx, request)
		}
	}
}

func stashAssessmentCreateBody(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	body databasemigrationsdk.CreateAssessmentDetails,
) error {
	key := assessmentRequestBodyToken(ctx)
	if key == "" {
		key = assessmentResourceBodyKey(resource)
	}
	if key == "" {
		return fmt.Errorf("Assessment create body cannot be keyed without resource namespace/name or uid")
	}
	pendingAssessmentCreateBodies.Store(key, body)
	return nil
}

func takeAssessmentCreateBody(
	ctx context.Context,
	retryToken *string,
) (databasemigrationsdk.CreateAssessmentDetails, error) {
	key := assessmentRequestBodyToken(ctx)
	if key == "" {
		key = assessmentStringValue(retryToken)
	}
	if key == "" {
		return nil, fmt.Errorf("Assessment create body is missing a request key")
	}
	value, ok := pendingAssessmentCreateBodies.LoadAndDelete(key)
	if !ok {
		return nil, fmt.Errorf("Assessment create body was not prepared for key %q", key)
	}
	body, ok := value.(databasemigrationsdk.CreateAssessmentDetails)
	if !ok {
		return nil, fmt.Errorf("prepared Assessment create body has unexpected type %T", value)
	}
	return body, nil
}

func stashAssessmentUpdateBody(
	ctx context.Context,
	resource *databasemigrationv1beta1.Assessment,
	body databasemigrationsdk.UpdateAssessmentDetails,
) error {
	key := assessmentRequestBodyToken(ctx)
	if key == "" {
		key = trackedAssessmentID(resource)
	}
	if key == "" {
		return fmt.Errorf("Assessment update body cannot be keyed without a tracked Assessment OCID")
	}
	pendingAssessmentUpdateBodies.Store(key, body)
	return nil
}

func takeAssessmentUpdateBody(
	ctx context.Context,
	assessmentID *string,
) (databasemigrationsdk.UpdateAssessmentDetails, error) {
	key := assessmentRequestBodyToken(ctx)
	if key == "" {
		key = assessmentStringValue(assessmentID)
	}
	if key == "" {
		return nil, fmt.Errorf("Assessment update body is missing a resource key")
	}
	value, ok := pendingAssessmentUpdateBodies.LoadAndDelete(key)
	if !ok {
		return nil, fmt.Errorf("Assessment update body was not prepared for key %q", key)
	}
	body, ok := value.(databasemigrationsdk.UpdateAssessmentDetails)
	if !ok {
		return nil, fmt.Errorf("prepared Assessment update body has unexpected type %T", value)
	}
	return body, nil
}

func trackedAssessmentID(resource *databasemigrationv1beta1.Assessment) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func assessmentResourceBodyKey(resource *databasemigrationv1beta1.Assessment) string {
	if resource == nil {
		return ""
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return uid
	}
	namespace := strings.TrimSpace(resource.Namespace)
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return ""
	}
	return namespace + "/" + name
}

func assessmentDecodeJSONObject(raw any, target any) error {
	payload, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}

func cloneAssessmentStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func cloneAssessmentDefinedTags(values map[string]map[string]interface{}) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(values))
	for namespace, tags := range values {
		clone[namespace] = make(map[string]interface{}, len(tags))
		for key, value := range tags {
			clone[namespace][key] = value
		}
	}
	return clone
}

func assessmentDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
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
