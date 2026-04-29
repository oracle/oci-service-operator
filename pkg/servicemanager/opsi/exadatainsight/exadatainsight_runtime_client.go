/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package exadatainsight

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const exadataInsightCreateOnlyFingerprintKey = "osokExadataInsightCreateOnlySHA256="

var exadataInsightWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(opsisdk.OperationTypeCreateExadataInsight),
		string(opsisdk.ActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(opsisdk.OperationTypeUpdateExadataInsight),
		string(opsisdk.ActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(opsisdk.OperationTypeDeleteExadataInsight),
		string(opsisdk.ActionTypeDeleted),
	},
}

type exadataInsightOCIClient interface {
	CreateExadataInsight(context.Context, opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error)
	GetExadataInsight(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error)
	ListExadataInsights(context.Context, opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error)
	UpdateExadataInsight(context.Context, opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error)
	DeleteExadataInsight(context.Context, opsisdk.DeleteExadataInsightRequest) (opsisdk.DeleteExadataInsightResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type exadataInsightListCall func(context.Context, opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error)
type exadataInsightGetCall func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error)

type exadataInsightAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e exadataInsightAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e exadataInsightAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerExadataInsightRuntimeHooksMutator(func(manager *ExadataInsightServiceManager, hooks *ExadataInsightRuntimeHooks) {
		client, initErr := newExadataInsightOCIClient(manager)
		applyExadataInsightRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newExadataInsightOCIClient(manager *ExadataInsightServiceManager) (exadataInsightOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("exadata insight service manager is nil")
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyExadataInsightRuntimeHooks(
	manager *ExadataInsightServiceManager,
	hooks *ExadataInsightRuntimeHooks,
	client exadataInsightOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	bodyStore := &exadataInsightRequestBodyStore{}
	hooks.Semantics = exadataInsightRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *opsiv1beta1.ExadataInsight, namespace string) (any, error) {
		body, err := buildExadataInsightCreateDetails(ctx, manager, resource, namespace)
		if err == nil {
			bodyStore.storeCreate(resource, body)
		}
		return body, err
	}
	hooks.BuildUpdateBody = func(ctx context.Context, resource *opsiv1beta1.ExadataInsight, namespace string, currentResponse any) (any, bool, error) {
		body, ok, err := buildExadataInsightUpdateDetails(ctx, manager, resource, namespace, currentResponse)
		if err == nil && ok {
			updateBody, implements := body.(opsisdk.UpdateExadataInsightDetails)
			if !implements {
				return nil, false, fmt.Errorf("exadata insight update body %T does not implement UpdateExadataInsightDetails", body)
			}
			bodyStore.storeUpdate(resource, currentResponse, updateBody)
		}
		return body, ok, err
	}
	hooks.StatusHooks.ProjectStatus = projectExadataInsightStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateExadataInsightCreateOnlyDrift
	hooks.Async.Adapter = exadataInsightWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getExadataInsightWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveExadataInsightWorkRequestAction
	hooks.Async.ResolvePhase = resolveExadataInsightWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverExadataInsightIDFromWorkRequest
	hooks.Async.Message = exadataInsightWorkRequestMessage
	hooks.Create.Fields = exadataInsightCreateFields()
	hooks.List.Fields = exadataInsightListFields()
	hooks.Update.Fields = exadataInsightUpdateFields()
	hooks.DeleteHooks.HandleError = handleExadataInsightDeleteError

	wrapExadataInsightPolymorphicBodyCalls(hooks, bodyStore)
	wrapExadataInsightStatusResponses(hooks)
	if hooks.List.Call != nil {
		hooks.List.Call = listExadataInsightsAllPages(hooks.List.Call)
	}
	rawGet := hooks.Get.Call
	if hooks.Get.Call != nil {
		hooks.Get.Call = getExadataInsightConservatively(hooks.Get.Call)
	}
	wrapExadataInsightDeleteConfirmation(hooks, rawGet)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ExadataInsightServiceClient) ExadataInsightServiceClient {
		return exadataInsightCreateOnlyTrackingClient{delegate: delegate}
	})
}

func exadataInsightRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "exadatainsight",
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
			ProvisioningStates: []string{string(opsisdk.ExadataInsightLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.ExadataInsightLifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.ExadataInsightLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.ExadataInsightLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.ExadataInsightLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"entitySource",
				"enterpriseManagerBridgeId",
				"enterpriseManagerEntityIdentifier",
				"exadataInfraId",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"freeformTags", "definedTags", "isAutoSyncEnabled"},
			Mutable:         []string{"freeformTags", "definedTags", "isAutoSyncEnabled"},
			ForceNew: []string{
				"compartmentId",
				"entitySource",
				"enterpriseManagerIdentifier",
				"enterpriseManagerBridgeId",
				"enterpriseManagerEntityIdentifier",
				"exadataInfraId",
				"jsonData",
				"memberEntityDetails",
				"memberVmClusterDetails",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func exadataInsightCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpcRetryToken", RequestName: "opcRetryToken", Contribution: "header"},
	}
}

func exadataInsightListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "EnterpriseManagerBridgeId", RequestName: "enterpriseManagerBridgeId", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func exadataInsightUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ExadataInsightId", RequestName: "exadataInsightId", Contribution: "path", PreferResourceID: true},
	}
}

func newExadataInsightServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client exadataInsightOCIClient,
) ExadataInsightServiceClient {
	manager := &ExadataInsightServiceManager{Log: log}
	hooks := newExadataInsightRuntimeHooksWithOCIClient(client)
	applyExadataInsightRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultExadataInsightServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.ExadataInsight](
			buildExadataInsightGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapExadataInsightGeneratedClient(hooks, delegate)
}

func newExadataInsightRuntimeHooksWithOCIClient(client exadataInsightOCIClient) ExadataInsightRuntimeHooks {
	return ExadataInsightRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.ExadataInsight]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.ExadataInsight]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.ExadataInsight]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.ExadataInsight]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.ExadataInsight]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.ExadataInsight]{},
		Create: runtimeOperationHooks[opsisdk.CreateExadataInsightRequest, opsisdk.CreateExadataInsightResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateExadataInsightDetails", RequestName: "CreateExadataInsightDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error) {
				if client == nil {
					return opsisdk.CreateExadataInsightResponse{}, fmt.Errorf("exadata insight OCI client is nil")
				}
				return client.CreateExadataInsight(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetExadataInsightRequest, opsisdk.GetExadataInsightResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ExadataInsightId", RequestName: "exadataInsightId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
				if client == nil {
					return opsisdk.GetExadataInsightResponse{}, fmt.Errorf("exadata insight OCI client is nil")
				}
				return client.GetExadataInsight(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListExadataInsightsRequest, opsisdk.ListExadataInsightsResponse]{
			Fields: exadataInsightListFields(),
			Call: func(ctx context.Context, request opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error) {
				if client == nil {
					return opsisdk.ListExadataInsightsResponse{}, fmt.Errorf("exadata insight OCI client is nil")
				}
				return client.ListExadataInsights(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateExadataInsightRequest, opsisdk.UpdateExadataInsightResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ExadataInsightId", RequestName: "exadataInsightId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateExadataInsightDetails", RequestName: "UpdateExadataInsightDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error) {
				if client == nil {
					return opsisdk.UpdateExadataInsightResponse{}, fmt.Errorf("exadata insight OCI client is nil")
				}
				return client.UpdateExadataInsight(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteExadataInsightRequest, opsisdk.DeleteExadataInsightResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ExadataInsightId", RequestName: "exadataInsightId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteExadataInsightRequest) (opsisdk.DeleteExadataInsightResponse, error) {
				if client == nil {
					return opsisdk.DeleteExadataInsightResponse{}, fmt.Errorf("exadata insight OCI client is nil")
				}
				return client.DeleteExadataInsight(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ExadataInsightServiceClient) ExadataInsightServiceClient{},
	}
}

type exadataInsightRequestBodyStore struct {
	create sync.Map
	update sync.Map
}

func (s *exadataInsightRequestBodyStore) storeCreate(
	resource *opsiv1beta1.ExadataInsight,
	body opsisdk.CreateExadataInsightDetails,
) {
	if s == nil || body == nil {
		return
	}
	s.create.Store(exadataInsightResourceRetryToken(resource), body)
}

func (s *exadataInsightRequestBodyStore) loadCreate(retryToken string) (opsisdk.CreateExadataInsightDetails, bool) {
	if s == nil {
		return nil, false
	}
	value, ok := s.create.LoadAndDelete(strings.TrimSpace(retryToken))
	if !ok {
		return nil, false
	}
	body, ok := value.(opsisdk.CreateExadataInsightDetails)
	return body, ok
}

func (s *exadataInsightRequestBodyStore) storeUpdate(
	resource *opsiv1beta1.ExadataInsight,
	currentResponse any,
	body opsisdk.UpdateExadataInsightDetails,
) {
	if s == nil || body == nil {
		return
	}
	if id := exadataInsightResponseID(currentResponse); id != "" {
		s.update.Store(id, body)
		return
	}
	if id := trackedExadataInsightID(resource); id != "" {
		s.update.Store(id, body)
	}
}

func (s *exadataInsightRequestBodyStore) loadUpdate(exadataInsightID string) (opsisdk.UpdateExadataInsightDetails, bool) {
	if s == nil {
		return nil, false
	}
	value, ok := s.update.LoadAndDelete(strings.TrimSpace(exadataInsightID))
	if !ok {
		return nil, false
	}
	body, ok := value.(opsisdk.UpdateExadataInsightDetails)
	return body, ok
}

func wrapExadataInsightPolymorphicBodyCalls(hooks *ExadataInsightRuntimeHooks, store *exadataInsightRequestBodyStore) {
	if hooks == nil || store == nil {
		return
	}
	if hooks.Create.Call != nil {
		create := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error) {
			body, ok := store.loadCreate(stringValue(request.OpcRetryToken))
			if !ok {
				return opsisdk.CreateExadataInsightResponse{}, fmt.Errorf("exadata insight create body was not prepared")
			}
			request.CreateExadataInsightDetails = body
			return create(ctx, request)
		}
	}
	if hooks.Update.Call != nil {
		update := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error) {
			body, ok := store.loadUpdate(stringValue(request.ExadataInsightId))
			if !ok {
				return opsisdk.UpdateExadataInsightResponse{}, fmt.Errorf("exadata insight update body was not prepared")
			}
			request.UpdateExadataInsightDetails = body
			return update(ctx, request)
		}
	}
}

func exadataInsightResourceRetryToken(resource *opsiv1beta1.ExadataInsight) string {
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
	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func buildExadataInsightCreateDetails(
	ctx context.Context,
	manager *ExadataInsightServiceManager,
	resource *opsiv1beta1.ExadataInsight,
	namespace string,
) (opsisdk.CreateExadataInsightDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("exadata insight resource is nil")
	}
	if err := validateExadataInsightSourceSpecificSpec(resource); err != nil {
		return nil, err
	}

	desired, err := exadataInsightDesiredValues(ctx, manager, resource, namespace)
	if err != nil {
		return nil, err
	}
	entitySource, err := normalizedExadataInsightEntitySource(resource.Spec.EntitySource)
	if err != nil {
		return nil, err
	}
	desired["entitySource"] = entitySource

	return exadataInsightCreateDetailsFromDesired(entitySource, desired, resource)
}

func exadataInsightCreateDetailsFromDesired(
	entitySource string,
	desired map[string]any,
	resource *opsiv1beta1.ExadataInsight,
) (opsisdk.CreateExadataInsightDetails, error) {
	switch entitySource {
	case string(opsisdk.ExadataEntitySourceEmManagedExternalExadata):
		return buildExadataInsightCreateEMDetails(desired)
	case string(opsisdk.ExadataEntitySourcePeComanagedExadata):
		return buildExadataInsightCreatePEDetails(desired, resource)
	case string(opsisdk.ExadataEntitySourceMacsManagedCloudExadata):
		return buildExadataInsightCreateMACSDetails(desired)
	default:
		return nil, fmt.Errorf("unsupported exadata insight entitySource %q", entitySource)
	}
}

func buildExadataInsightCreateEMDetails(desired map[string]any) (opsisdk.CreateExadataInsightDetails, error) {
	var details opsisdk.CreateEmManagedExternalExadataInsightDetails
	if err := decodeExadataInsightDetails(desired, &details); err != nil {
		return nil, err
	}
	return details, validateExadataInsightCreateEM(details)
}

func buildExadataInsightCreatePEDetails(
	desired map[string]any,
	resource *opsiv1beta1.ExadataInsight,
) (opsisdk.CreateExadataInsightDetails, error) {
	var details opsisdk.CreatePeComanagedExadataInsightDetails
	if err := decodeExadataInsightDetails(desired, &details); err != nil {
		return nil, err
	}
	memberDetails, err := buildExadataInsightPEMemberVmClusterDetails(resource, details.MemberVmClusterDetails)
	if err != nil {
		return nil, err
	}
	details.MemberVmClusterDetails = memberDetails
	return details, validateExadataInsightCreatePE(details)
}

func buildExadataInsightCreateMACSDetails(desired map[string]any) (opsisdk.CreateExadataInsightDetails, error) {
	var details opsisdk.CreateMacsManagedCloudExadataInsightDetails
	if err := decodeExadataInsightDetails(desired, &details); err != nil {
		return nil, err
	}
	return details, validateExadataInsightCreateMACS(details)
}

func buildExadataInsightUpdateDetails(
	ctx context.Context,
	manager *ExadataInsightServiceManager,
	resource *opsiv1beta1.ExadataInsight,
	namespace string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("exadata insight resource is nil")
	}
	if err := validateExadataInsightSourceSpecificSpec(resource); err != nil {
		return nil, false, err
	}
	if err := validateExadataInsightCreateOnlyDrift(resource, currentResponse); err != nil {
		return nil, false, err
	}

	desired, err := exadataInsightDesiredValues(ctx, manager, resource, namespace)
	if err != nil {
		return nil, false, err
	}
	observed := observedExadataInsightState(resource, currentResponse)
	entitySource, err := updateExadataInsightEntitySource(resource, observed)
	if err != nil {
		return nil, false, err
	}
	mutable, err := exadataInsightMutableUpdateFromDesired(resource, desired)
	if err != nil {
		return nil, false, err
	}

	return exadataInsightUpdateDetailsForSource(entitySource, mutable, observed)
}

func updateExadataInsightEntitySource(
	resource *opsiv1beta1.ExadataInsight,
	observed exadataInsightObservedState,
) (string, error) {
	if observed.entitySource != "" {
		return observed.entitySource, nil
	}
	return normalizedExadataInsightEntitySource(resource.Spec.EntitySource)
}

type exadataInsightMutableUpdate struct {
	freeformTags      map[string]string
	hasFreeformTags   bool
	definedTags       map[string]map[string]interface{}
	hasDefinedTags    bool
	isAutoSyncEnabled bool
	hasAutoSync       bool
}

func exadataInsightMutableUpdateFromDesired(
	resource *opsiv1beta1.ExadataInsight,
	desired map[string]any,
) (exadataInsightMutableUpdate, error) {
	var mutable exadataInsightMutableUpdate
	var err error
	mutable.freeformTags, mutable.hasFreeformTags, err = desiredStringMap(desired, "freeformTags")
	if err != nil {
		return exadataInsightMutableUpdate{}, err
	}
	mutable.definedTags, mutable.hasDefinedTags, err = desiredDefinedTags(desired, "definedTags")
	if err != nil {
		return exadataInsightMutableUpdate{}, err
	}
	mutable.isAutoSyncEnabled, mutable.hasAutoSync, err = desiredBool(desired, "isAutoSyncEnabled")
	if err != nil {
		return exadataInsightMutableUpdate{}, err
	}
	if resource != nil && resource.Spec.FreeformTags != nil {
		mutable.freeformTags = cloneStringMap(resource.Spec.FreeformTags)
		mutable.hasFreeformTags = true
	}
	if resource != nil && resource.Spec.DefinedTags != nil {
		mutable.definedTags = sharedMapToInterfaceMap(resource.Spec.DefinedTags)
		mutable.hasDefinedTags = true
	}
	return mutable, nil
}

func exadataInsightUpdateDetailsForSource(
	entitySource string,
	mutable exadataInsightMutableUpdate,
	observed exadataInsightObservedState,
) (opsisdk.UpdateExadataInsightDetails, bool, error) {
	switch entitySource {
	case string(opsisdk.ExadataEntitySourceEmManagedExternalExadata):
		return buildExadataInsightEMUpdate(mutable, observed)
	case string(opsisdk.ExadataEntitySourcePeComanagedExadata):
		return buildExadataInsightPEUpdate(mutable, observed)
	case string(opsisdk.ExadataEntitySourceMacsManagedCloudExadata):
		return buildExadataInsightMACSUpdate(mutable, observed)
	default:
		return nil, false, fmt.Errorf("unsupported exadata insight entitySource %q", entitySource)
	}
}

func buildExadataInsightEMUpdate(
	mutable exadataInsightMutableUpdate,
	observed exadataInsightObservedState,
) (opsisdk.UpdateExadataInsightDetails, bool, error) {
	update := opsisdk.UpdateEmManagedExternalExadataInsightDetails{}
	updateNeeded := false
	if mutable.hasFreeformTags && !jsonEqual(mutable.freeformTags, observed.freeformTags) {
		update.FreeformTags = mutable.freeformTags
		updateNeeded = true
	}
	if mutable.hasDefinedTags && !jsonEqual(mutable.definedTags, observed.definedTags) {
		update.DefinedTags = mutable.definedTags
		updateNeeded = true
	}
	if mutable.hasAutoSync && (observed.isAutoSyncEnabled == nil || *observed.isAutoSyncEnabled != mutable.isAutoSyncEnabled) {
		update.IsAutoSyncEnabled = common.Bool(mutable.isAutoSyncEnabled)
		updateNeeded = true
	}
	return update, updateNeeded, nil
}

func buildExadataInsightPEUpdate(
	mutable exadataInsightMutableUpdate,
	observed exadataInsightObservedState,
) (opsisdk.UpdateExadataInsightDetails, bool, error) {
	if err := rejectUnsupportedExadataInsightAutoSync(mutable); err != nil {
		return nil, false, err
	}
	update := opsisdk.UpdatePeComanagedExadataInsightDetails{}
	updateNeeded := false
	if mutable.hasFreeformTags && !jsonEqual(mutable.freeformTags, observed.freeformTags) {
		update.FreeformTags = mutable.freeformTags
		updateNeeded = true
	}
	if mutable.hasDefinedTags && !jsonEqual(mutable.definedTags, observed.definedTags) {
		update.DefinedTags = mutable.definedTags
		updateNeeded = true
	}
	return update, updateNeeded, nil
}

func buildExadataInsightMACSUpdate(
	mutable exadataInsightMutableUpdate,
	observed exadataInsightObservedState,
) (opsisdk.UpdateExadataInsightDetails, bool, error) {
	if err := rejectUnsupportedExadataInsightAutoSync(mutable); err != nil {
		return nil, false, err
	}
	update := opsisdk.UpdateMacsManagedCloudExadataInsightDetails{}
	updateNeeded := false
	if mutable.hasFreeformTags && !jsonEqual(mutable.freeformTags, observed.freeformTags) {
		update.FreeformTags = mutable.freeformTags
		updateNeeded = true
	}
	if mutable.hasDefinedTags && !jsonEqual(mutable.definedTags, observed.definedTags) {
		update.DefinedTags = mutable.definedTags
		updateNeeded = true
	}
	return update, updateNeeded, nil
}

func rejectUnsupportedExadataInsightAutoSync(mutable exadataInsightMutableUpdate) error {
	if mutable.hasAutoSync && mutable.isAutoSyncEnabled {
		return fmt.Errorf("exadata insight isAutoSyncEnabled is only supported for entitySource %s", opsisdk.ExadataEntitySourceEmManagedExternalExadata)
	}
	return nil
}

func exadataInsightDesiredValues(
	ctx context.Context,
	manager *ExadataInsightServiceManager,
	resource *opsiv1beta1.ExadataInsight,
	namespace string,
) (map[string]any, error) {
	if manager == nil {
		return nil, fmt.Errorf("exadata insight service manager is nil")
	}
	resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, manager.CredentialClient, namespace)
	if err != nil {
		return nil, err
	}
	values, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("exadata insight spec resolved to %T, want map", resolved)
	}
	delete(values, "jsonData")
	return values, nil
}

func validateExadataInsightSourceSpecificSpec(resource *opsiv1beta1.ExadataInsight) error {
	entitySource, err := normalizedExadataInsightEntitySource(resource.Spec.EntitySource)
	if err != nil {
		return err
	}
	switch entitySource {
	case string(opsisdk.ExadataEntitySourceEmManagedExternalExadata):
		return validateExadataInsightEMSpec(resource, entitySource)
	case string(opsisdk.ExadataEntitySourcePeComanagedExadata),
		string(opsisdk.ExadataEntitySourceMacsManagedCloudExadata):
		return validateExadataInsightCoManagedSpec(resource, entitySource)
	default:
		return fmt.Errorf("unsupported exadata insight entitySource %q", entitySource)
	}
}

func validateExadataInsightEMSpec(resource *opsiv1beta1.ExadataInsight, entitySource string) error {
	if strings.TrimSpace(resource.Spec.ExadataInfraId) != "" || len(resource.Spec.MemberVmClusterDetails) > 0 {
		return fmt.Errorf("exadata insight entitySource %s cannot set exadataInfraId or memberVmClusterDetails", entitySource)
	}
	if strings.TrimSpace(resource.Spec.EnterpriseManagerIdentifier) == "" {
		return fmt.Errorf("exadata insight enterpriseManagerIdentifier is required for entitySource %s", entitySource)
	}
	if strings.TrimSpace(resource.Spec.EnterpriseManagerBridgeId) == "" {
		return fmt.Errorf("exadata insight enterpriseManagerBridgeId is required for entitySource %s", entitySource)
	}
	if strings.TrimSpace(resource.Spec.EnterpriseManagerEntityIdentifier) == "" {
		return fmt.Errorf("exadata insight enterpriseManagerEntityIdentifier is required for entitySource %s", entitySource)
	}
	return nil
}

func validateExadataInsightCoManagedSpec(resource *opsiv1beta1.ExadataInsight, entitySource string) error {
	if exadataInsightHasEnterpriseManagerFields(resource) {
		return fmt.Errorf("exadata insight entitySource %s cannot set Enterprise Manager identity fields", entitySource)
	}
	if resource.Spec.IsAutoSyncEnabled {
		return fmt.Errorf("exadata insight isAutoSyncEnabled is only supported for entitySource %s", opsisdk.ExadataEntitySourceEmManagedExternalExadata)
	}
	if strings.TrimSpace(resource.Spec.ExadataInfraId) == "" {
		return fmt.Errorf("exadata insight exadataInfraId is required for entitySource %s", entitySource)
	}
	return nil
}

func exadataInsightHasEnterpriseManagerFields(resource *opsiv1beta1.ExadataInsight) bool {
	return strings.TrimSpace(resource.Spec.EnterpriseManagerIdentifier) != "" ||
		strings.TrimSpace(resource.Spec.EnterpriseManagerBridgeId) != "" ||
		strings.TrimSpace(resource.Spec.EnterpriseManagerEntityIdentifier) != "" ||
		len(resource.Spec.MemberEntityDetails) > 0
}

func validateExadataInsightCreateEM(details opsisdk.CreateEmManagedExternalExadataInsightDetails) error {
	if stringValue(details.CompartmentId) == "" {
		return fmt.Errorf("exadata insight compartmentId is required")
	}
	if stringValue(details.EnterpriseManagerIdentifier) == "" {
		return fmt.Errorf("exadata insight enterpriseManagerIdentifier is required")
	}
	if stringValue(details.EnterpriseManagerBridgeId) == "" {
		return fmt.Errorf("exadata insight enterpriseManagerBridgeId is required")
	}
	if stringValue(details.EnterpriseManagerEntityIdentifier) == "" {
		return fmt.Errorf("exadata insight enterpriseManagerEntityIdentifier is required")
	}
	return nil
}

func validateExadataInsightCreatePE(details opsisdk.CreatePeComanagedExadataInsightDetails) error {
	if stringValue(details.CompartmentId) == "" {
		return fmt.Errorf("exadata insight compartmentId is required")
	}
	if stringValue(details.ExadataInfraId) == "" {
		return fmt.Errorf("exadata insight exadataInfraId is required")
	}
	return validateExadataInsightPEMemberClusters(details.MemberVmClusterDetails)
}

func validateExadataInsightPEMemberClusters(clusters []opsisdk.CreatePeComanagedExadataVmclusterDetails) error {
	for clusterIndex, cluster := range clusters {
		if err := validateExadataInsightPEMemberCluster(cluster, clusterIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateExadataInsightPEMemberCluster(
	cluster opsisdk.CreatePeComanagedExadataVmclusterDetails,
	clusterIndex int,
) error {
	if stringValue(cluster.VmclusterId) == "" {
		return fmt.Errorf("exadata insight memberVmClusterDetails[%d].vmclusterId is required", clusterIndex)
	}
	if stringValue(cluster.CompartmentId) == "" {
		return fmt.Errorf("exadata insight memberVmClusterDetails[%d].compartmentId is required", clusterIndex)
	}
	for databaseIndex, database := range cluster.MemberDatabaseDetails {
		if err := validateExadataInsightPEMemberDatabase(database, clusterIndex, databaseIndex); err != nil {
			return err
		}
	}
	for autonomousIndex, autonomous := range cluster.MemberAutonomousDetails {
		if err := validateExadataInsightPEMemberAutonomous(autonomous, clusterIndex, autonomousIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateExadataInsightPEMemberDatabase(
	database opsisdk.CreatePeComanagedDatabaseInsightDetails,
	clusterIndex int,
	databaseIndex int,
) error {
	field := fmt.Sprintf("memberVmClusterDetails[%d].memberDatabaseDetails[%d]", clusterIndex, databaseIndex)
	if err := validateExadataInsightPEMemberDatabaseRequiredFields(database, field); err != nil {
		return err
	}
	if database.CredentialDetails == nil {
		return fmt.Errorf("exadata insight %s.credentialDetails is required", field)
	}
	if database.ConnectionDetails == nil || len(database.ConnectionDetails.Hosts) == 0 {
		return fmt.Errorf("exadata insight %s.connectionDetails.hosts is required", field)
	}
	if stringValue(database.OpsiPrivateEndpointId) == "" && stringValue(database.DbmPrivateEndpointId) == "" {
		return fmt.Errorf("exadata insight %s requires spec.jsonData.%s.opsiPrivateEndpointId or dbmPrivateEndpointId because the generated CRD does not expose PE endpoint fields", field, field)
	}
	if database.DeploymentType == "" {
		return fmt.Errorf("exadata insight %s.deploymentType is required", field)
	}
	return nil
}

func validateExadataInsightPEMemberDatabaseRequiredFields(
	database opsisdk.CreatePeComanagedDatabaseInsightDetails,
	field string,
) error {
	required := []struct {
		name  string
		value string
	}{
		{name: "compartmentId", value: stringValue(database.CompartmentId)},
		{name: "databaseId", value: stringValue(database.DatabaseId)},
		{name: "databaseResourceType", value: stringValue(database.DatabaseResourceType)},
		{name: "serviceName", value: stringValue(database.ServiceName)},
	}
	for _, current := range required {
		if current.value == "" {
			return fmt.Errorf("exadata insight %s.%s is required", field, current.name)
		}
	}
	return nil
}

func validateExadataInsightPEMemberAutonomous(
	autonomous opsisdk.CreateAutonomousDatabaseInsightDetails,
	clusterIndex int,
	autonomousIndex int,
) error {
	field := fmt.Sprintf("memberVmClusterDetails[%d].memberAutonomousDetails[%d]", clusterIndex, autonomousIndex)
	if err := validateExadataInsightPEMemberAutonomousRequiredFields(autonomous, field); err != nil {
		return err
	}
	if autonomous.IsAdvancedFeaturesEnabled == nil {
		return fmt.Errorf("exadata insight %s.isAdvancedFeaturesEnabled is required in spec.jsonData", field)
	}
	if *autonomous.IsAdvancedFeaturesEnabled {
		if autonomous.CredentialDetails == nil {
			return fmt.Errorf("exadata insight %s.credentialDetails is required when isAdvancedFeaturesEnabled is true", field)
		}
		if autonomous.ConnectionDetails == nil {
			return fmt.Errorf("exadata insight %s.connectionDetails is required when isAdvancedFeaturesEnabled is true", field)
		}
	}
	return nil
}

func validateExadataInsightPEMemberAutonomousRequiredFields(
	autonomous opsisdk.CreateAutonomousDatabaseInsightDetails,
	field string,
) error {
	required := []struct {
		name  string
		value string
	}{
		{name: "compartmentId", value: stringValue(autonomous.CompartmentId)},
		{name: "databaseId", value: stringValue(autonomous.DatabaseId)},
		{name: "databaseResourceType", value: stringValue(autonomous.DatabaseResourceType)},
	}
	for _, current := range required {
		if current.value == "" {
			return fmt.Errorf("exadata insight %s.%s is required", field, current.name)
		}
	}
	return nil
}

func validateExadataInsightCreateMACS(details opsisdk.CreateMacsManagedCloudExadataInsightDetails) error {
	if stringValue(details.CompartmentId) == "" {
		return fmt.Errorf("exadata insight compartmentId is required")
	}
	if stringValue(details.ExadataInfraId) == "" {
		return fmt.Errorf("exadata insight exadataInfraId is required")
	}
	return nil
}

type exadataInsightPECreateOverlay struct {
	MemberVmClusterDetails []exadataInsightPEVmClusterOverlay `json:"memberVmClusterDetails,omitempty"`
}

type exadataInsightPEVmClusterOverlay struct {
	OpsiPrivateEndpointId   string                                    `json:"opsiPrivateEndpointId,omitempty"`
	DbmPrivateEndpointId    string                                    `json:"dbmPrivateEndpointId,omitempty"`
	VmClusterType           string                                    `json:"vmClusterType,omitempty"`
	MemberDatabaseDetails   []exadataInsightPEMemberDatabaseOverlay   `json:"memberDatabaseDetails,omitempty"`
	MemberAutonomousDetails []exadataInsightPEMemberAutonomousOverlay `json:"memberAutonomousDetails,omitempty"`
}

type exadataInsightPEMemberDatabaseOverlay struct {
	OpsiPrivateEndpointId string `json:"opsiPrivateEndpointId,omitempty"`
	DbmPrivateEndpointId  string `json:"dbmPrivateEndpointId,omitempty"`
}

type exadataInsightPEMemberAutonomousOverlay struct {
	OpsiPrivateEndpointId     string `json:"opsiPrivateEndpointId,omitempty"`
	IsAdvancedFeaturesEnabled *bool  `json:"isAdvancedFeaturesEnabled,omitempty"`
}

func buildExadataInsightPEMemberVmClusterDetails(
	resource *opsiv1beta1.ExadataInsight,
	decoded []opsisdk.CreatePeComanagedExadataVmclusterDetails,
) ([]opsisdk.CreatePeComanagedExadataVmclusterDetails, error) {
	if resource == nil || len(resource.Spec.MemberVmClusterDetails) == 0 {
		return decoded, nil
	}
	overlay, err := exadataInsightPECreateOverlayFromJSON(resource.Spec.JsonData)
	if err != nil {
		return nil, err
	}
	if len(overlay.MemberVmClusterDetails) > len(resource.Spec.MemberVmClusterDetails) {
		return nil, fmt.Errorf("exadata insight spec.jsonData.memberVmClusterDetails has %d entries, want at most %d", len(overlay.MemberVmClusterDetails), len(resource.Spec.MemberVmClusterDetails))
	}

	clusters := make([]opsisdk.CreatePeComanagedExadataVmclusterDetails, 0, len(resource.Spec.MemberVmClusterDetails))
	for index, member := range resource.Spec.MemberVmClusterDetails {
		cluster := decodedExadataInsightPEVmCluster(decoded, index)
		overlayCluster := exadataInsightPEVmClusterOverlayAt(overlay, index)
		built, err := buildExadataInsightPEMemberVmClusterDetail(member, overlayCluster, cluster)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, built)
	}
	return clusters, nil
}

func decodedExadataInsightPEVmCluster(
	decoded []opsisdk.CreatePeComanagedExadataVmclusterDetails,
	index int,
) opsisdk.CreatePeComanagedExadataVmclusterDetails {
	if index >= 0 && index < len(decoded) {
		return decoded[index]
	}
	return opsisdk.CreatePeComanagedExadataVmclusterDetails{}
}

func exadataInsightPEVmClusterOverlayAt(
	overlay exadataInsightPECreateOverlay,
	index int,
) exadataInsightPEVmClusterOverlay {
	if index >= 0 && index < len(overlay.MemberVmClusterDetails) {
		return overlay.MemberVmClusterDetails[index]
	}
	return exadataInsightPEVmClusterOverlay{}
}

func buildExadataInsightPEMemberVmClusterDetail(
	member opsiv1beta1.ExadataInsightMemberVmClusterDetail,
	overlay exadataInsightPEVmClusterOverlay,
	base opsisdk.CreatePeComanagedExadataVmclusterDetails,
) (opsisdk.CreatePeComanagedExadataVmclusterDetails, error) {
	base.VmclusterId = exadataInsightStringPointer(member.VmclusterId)
	base.CompartmentId = exadataInsightStringPointer(member.CompartmentId)
	base.OpsiPrivateEndpointId = exadataInsightStringPointer(overlay.OpsiPrivateEndpointId)
	base.DbmPrivateEndpointId = exadataInsightStringPointer(overlay.DbmPrivateEndpointId)
	if strings.TrimSpace(overlay.VmClusterType) != "" {
		vmClusterType, ok := opsisdk.GetMappingExadataVmClusterTypeEnum(overlay.VmClusterType)
		if !ok {
			return opsisdk.CreatePeComanagedExadataVmclusterDetails{}, fmt.Errorf("unsupported exadata insight spec.jsonData.memberVmClusterDetails.vmClusterType %q", overlay.VmClusterType)
		}
		base.VmClusterType = vmClusterType
	}
	memberDatabaseDetails, err := buildExadataInsightPEMemberDatabaseDetails(
		member.MemberDatabaseDetails,
		overlay.MemberDatabaseDetails,
	)
	if err != nil {
		return opsisdk.CreatePeComanagedExadataVmclusterDetails{}, err
	}
	base.MemberDatabaseDetails = memberDatabaseDetails
	memberAutonomousDetails, err := buildExadataInsightPEMemberAutonomousDetails(
		member.MemberAutonomousDetails,
		overlay.MemberAutonomousDetails,
	)
	if err != nil {
		return opsisdk.CreatePeComanagedExadataVmclusterDetails{}, err
	}
	base.MemberAutonomousDetails = memberAutonomousDetails
	return base, nil
}

func buildExadataInsightPEMemberDatabaseDetails(
	members []opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetail,
	overlays []exadataInsightPEMemberDatabaseOverlay,
) ([]opsisdk.CreatePeComanagedDatabaseInsightDetails, error) {
	if len(overlays) > len(members) {
		return nil, fmt.Errorf("exadata insight spec.jsonData.memberVmClusterDetails.memberDatabaseDetails has %d entries, want at most %d", len(overlays), len(members))
	}
	details := make([]opsisdk.CreatePeComanagedDatabaseInsightDetails, 0, len(members))
	for index, member := range members {
		database, err := buildExadataInsightPEMemberDatabaseDetail(member, exadataInsightPEMemberDatabaseOverlayAt(overlays, index))
		if err != nil {
			return nil, err
		}
		details = append(details, database)
	}
	return details, nil
}

func exadataInsightPEMemberDatabaseOverlayAt(
	overlays []exadataInsightPEMemberDatabaseOverlay,
	index int,
) exadataInsightPEMemberDatabaseOverlay {
	if index >= 0 && index < len(overlays) {
		return overlays[index]
	}
	return exadataInsightPEMemberDatabaseOverlay{}
}

func buildExadataInsightPEMemberDatabaseDetail(
	member opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetail,
	overlay exadataInsightPEMemberDatabaseOverlay,
) (opsisdk.CreatePeComanagedDatabaseInsightDetails, error) {
	connectionDetails, err := exadataInsightPEMemberDatabaseConnectionDetails(member.ConnectionDetails)
	if err != nil {
		return opsisdk.CreatePeComanagedDatabaseInsightDetails{}, err
	}
	credentialDetails, err := exadataInsightCredentialDetailsFromMemberDatabase(member.ConnectionCredentialDetails)
	if err != nil {
		return opsisdk.CreatePeComanagedDatabaseInsightDetails{}, err
	}
	deploymentType, err := exadataInsightPEMemberDatabaseDeploymentType(member.DeploymentType)
	if err != nil {
		return opsisdk.CreatePeComanagedDatabaseInsightDetails{}, err
	}

	return opsisdk.CreatePeComanagedDatabaseInsightDetails{
		CompartmentId:         exadataInsightStringPointer(member.CompartmentId),
		DatabaseId:            exadataInsightStringPointer(member.DatabaseId),
		DatabaseResourceType:  exadataInsightStringPointer(member.DatabaseResourceType),
		ServiceName:           exadataInsightStringPointer(member.ConnectionDetails.ServiceName),
		CredentialDetails:     credentialDetails,
		FreeformTags:          cloneStringMap(member.FreeformTags),
		DefinedTags:           sharedMapToInterfaceMap(member.DefinedTags),
		OpsiPrivateEndpointId: exadataInsightStringPointer(overlay.OpsiPrivateEndpointId),
		DbmPrivateEndpointId:  exadataInsightStringPointer(overlay.DbmPrivateEndpointId),
		ConnectionDetails:     connectionDetails,
		SystemTags:            sharedMapToInterfaceMap(member.SystemTags),
		DeploymentType:        deploymentType,
	}, nil
}

func buildExadataInsightPEMemberAutonomousDetails(
	members []opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail,
	overlays []exadataInsightPEMemberAutonomousOverlay,
) ([]opsisdk.CreateAutonomousDatabaseInsightDetails, error) {
	if len(overlays) > len(members) {
		return nil, fmt.Errorf("exadata insight spec.jsonData.memberVmClusterDetails.memberAutonomousDetails has %d entries, want at most %d", len(overlays), len(members))
	}
	details := make([]opsisdk.CreateAutonomousDatabaseInsightDetails, 0, len(members))
	for index, member := range members {
		autonomous, err := buildExadataInsightPEMemberAutonomousDetail(member, exadataInsightPEMemberAutonomousOverlayAt(overlays, index))
		if err != nil {
			return nil, err
		}
		details = append(details, autonomous)
	}
	return details, nil
}

func exadataInsightPEMemberAutonomousOverlayAt(
	overlays []exadataInsightPEMemberAutonomousOverlay,
	index int,
) exadataInsightPEMemberAutonomousOverlay {
	if index >= 0 && index < len(overlays) {
		return overlays[index]
	}
	return exadataInsightPEMemberAutonomousOverlay{}
}

func buildExadataInsightPEMemberAutonomousDetail(
	member opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail,
	overlay exadataInsightPEMemberAutonomousOverlay,
) (opsisdk.CreateAutonomousDatabaseInsightDetails, error) {
	connectionDetails, err := exadataInsightPEMemberAutonomousConnectionDetails(member.ConnectionDetails)
	if err != nil {
		return opsisdk.CreateAutonomousDatabaseInsightDetails{}, err
	}
	credentialDetails, err := exadataInsightCredentialDetailsFromMemberAutonomous(member.ConnectionCredentialDetails)
	if err != nil {
		return opsisdk.CreateAutonomousDatabaseInsightDetails{}, err
	}

	return opsisdk.CreateAutonomousDatabaseInsightDetails{
		CompartmentId:             exadataInsightStringPointer(member.CompartmentId),
		DatabaseId:                exadataInsightStringPointer(member.DatabaseId),
		DatabaseResourceType:      exadataInsightStringPointer(member.DatabaseResourceType),
		IsAdvancedFeaturesEnabled: overlay.IsAdvancedFeaturesEnabled,
		FreeformTags:              cloneStringMap(member.FreeformTags),
		DefinedTags:               sharedMapToInterfaceMap(member.DefinedTags),
		ConnectionDetails:         connectionDetails,
		CredentialDetails:         credentialDetails,
		OpsiPrivateEndpointId:     exadataInsightStringPointer(overlay.OpsiPrivateEndpointId),
		SystemTags:                sharedMapToInterfaceMap(member.SystemTags),
	}, nil
}

func exadataInsightPEMemberAutonomousConnectionDetails(
	details opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetailConnectionDetails,
) (*opsisdk.ConnectionDetails, error) {
	protocol, ok := opsisdk.GetMappingConnectionDetailsProtocolEnum(details.Protocol)
	if !ok {
		return nil, fmt.Errorf("unsupported exadata insight PE member autonomous connection protocol %q", details.Protocol)
	}
	return &opsisdk.ConnectionDetails{
		HostName:    exadataInsightStringPointer(details.HostName),
		Protocol:    protocol,
		Port:        exadataInsightIntPointer(details.Port),
		ServiceName: exadataInsightStringPointer(details.ServiceName),
	}, nil
}

func exadataInsightPEMemberDatabaseConnectionDetails(
	details opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionDetails,
) (*opsisdk.PeComanagedDatabaseConnectionDetails, error) {
	protocol, ok := opsisdk.GetMappingPeComanagedDatabaseConnectionDetailsProtocolEnum(details.Protocol)
	if !ok {
		return nil, fmt.Errorf("unsupported exadata insight PE member database connection protocol %q", details.Protocol)
	}
	return &opsisdk.PeComanagedDatabaseConnectionDetails{
		Hosts: []opsisdk.PeComanagedDatabaseHostDetails{{
			HostIp: exadataInsightStringPointer(details.HostName),
			Port:   exadataInsightIntPointer(details.Port),
		}},
		Protocol:    protocol,
		ServiceName: exadataInsightStringPointer(details.ServiceName),
	}, nil
}

func exadataInsightCredentialDetailsFromMemberAutonomous(
	details opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetailConnectionCredentialDetails,
) (opsisdk.CredentialDetails, error) {
	return exadataInsightCredentialDetailsFromMemberDatabase(
		opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails(details),
	)
}

func exadataInsightCredentialDetailsFromMemberDatabase(
	details opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails,
) (opsisdk.CredentialDetails, error) {
	normalized, err := normalizedExadataInsightCredentialDetails(details)
	if err != nil {
		return nil, err
	}
	credentialType, ok := opsisdk.GetMappingCredentialDetailsCredentialTypeEnum(normalized.CredentialType)
	if !ok {
		return nil, fmt.Errorf("unsupported exadata insight credentialType %q", normalized.CredentialType)
	}
	switch credentialType {
	case opsisdk.CredentialDetailsCredentialTypeSource:
		return opsisdk.CredentialsBySource{CredentialSourceName: exadataInsightStringPointer(normalized.CredentialSourceName)}, nil
	case opsisdk.CredentialDetailsCredentialTypeNamedCreds:
		return opsisdk.CredentialByNamedCredentials{
			CredentialSourceName: exadataInsightStringPointer(normalized.CredentialSourceName),
			NamedCredentialId:    exadataInsightStringPointer(normalized.NamedCredentialId),
		}, nil
	case opsisdk.CredentialDetailsCredentialTypeVault:
		role, err := exadataInsightCredentialByVaultRole(normalized.Role)
		if err != nil {
			return nil, err
		}
		return opsisdk.CredentialByVault{
			CredentialSourceName: exadataInsightStringPointer(normalized.CredentialSourceName),
			UserName:             exadataInsightStringPointer(normalized.UserName),
			PasswordSecretId:     exadataInsightStringPointer(normalized.PasswordSecretId),
			WalletSecretId:       exadataInsightStringPointer(normalized.WalletSecretId),
			Role:                 role,
		}, nil
	case opsisdk.CredentialDetailsCredentialTypeIam:
		return opsisdk.CredentialByIam{CredentialSourceName: exadataInsightStringPointer(normalized.CredentialSourceName)}, nil
	default:
		return nil, fmt.Errorf("unsupported exadata insight credentialType %q", normalized.CredentialType)
	}
}

func normalizedExadataInsightCredentialDetails(
	details opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails,
) (opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails, error) {
	raw := strings.TrimSpace(details.JsonData)
	if raw == "" {
		return details, nil
	}
	var overlay opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		return details, fmt.Errorf("decode exadata insight connectionCredentialDetails.jsonData: %w", err)
	}
	mergeExadataInsightCredentialDetails(&details, overlay)
	return details, nil
}

func mergeExadataInsightCredentialDetails(
	details *opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails,
	overlay opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails,
) {
	if strings.TrimSpace(details.CredentialSourceName) == "" {
		details.CredentialSourceName = overlay.CredentialSourceName
	}
	if strings.TrimSpace(details.CredentialType) == "" {
		details.CredentialType = overlay.CredentialType
	}
	if strings.TrimSpace(details.NamedCredentialId) == "" {
		details.NamedCredentialId = overlay.NamedCredentialId
	}
	if strings.TrimSpace(details.UserName) == "" {
		details.UserName = overlay.UserName
	}
	if strings.TrimSpace(details.PasswordSecretId) == "" {
		details.PasswordSecretId = overlay.PasswordSecretId
	}
	if strings.TrimSpace(details.WalletSecretId) == "" {
		details.WalletSecretId = overlay.WalletSecretId
	}
	if strings.TrimSpace(details.Role) == "" {
		details.Role = overlay.Role
	}
}

func exadataInsightCredentialByVaultRole(raw string) (opsisdk.CredentialByVaultRoleEnum, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	role, ok := opsisdk.GetMappingCredentialByVaultRoleEnum(raw)
	if !ok {
		return "", fmt.Errorf("unsupported exadata insight credential role %q", raw)
	}
	return role, nil
}

func exadataInsightPEMemberDatabaseDeploymentType(raw string) (opsisdk.CreatePeComanagedDatabaseInsightDetailsDeploymentTypeEnum, error) {
	deploymentType, ok := opsisdk.GetMappingCreatePeComanagedDatabaseInsightDetailsDeploymentTypeEnum(raw)
	if !ok {
		return "", fmt.Errorf("unsupported exadata insight PE member database deploymentType %q", raw)
	}
	return deploymentType, nil
}

func exadataInsightPECreateOverlayFromJSON(raw string) (exadataInsightPECreateOverlay, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return exadataInsightPECreateOverlay{}, nil
	}
	var overlay exadataInsightPECreateOverlay
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		return exadataInsightPECreateOverlay{}, fmt.Errorf("decode exadata insight PE jsonData: %w", err)
	}
	return overlay, nil
}

func exadataInsightStringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func exadataInsightIntPointer(value int) *int {
	if value == 0 {
		return nil
	}
	return common.Int(value)
}

func normalizedExadataInsightEntitySource(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("exadata insight entitySource is required")
	}
	value, ok := opsisdk.GetMappingExadataEntitySourceEnum(raw)
	if !ok {
		return "", fmt.Errorf("unsupported exadata insight entitySource %q", raw)
	}
	return string(value), nil
}

func decodeExadataInsightDetails(values map[string]any, target any) error {
	payload, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal exadata insight request body: %w", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode exadata insight request body: %w", err)
	}
	return nil
}

func wrapExadataInsightStatusResponses(hooks *ExadataInsightRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.Create.Call = wrapExadataInsightCreateStatusResponse(hooks.Create.Call)
	hooks.Get.Call = wrapExadataInsightGetStatusResponse(hooks.Get.Call)
	hooks.List.Call = wrapExadataInsightListStatusResponse(hooks.List.Call)
}

func wrapExadataInsightCreateStatusResponse(
	create func(context.Context, opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error),
) func(context.Context, opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error) {
	if create == nil {
		return nil
	}
	return func(ctx context.Context, request opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error) {
		response, err := create(ctx, request)
		if err != nil {
			return response, err
		}
		response.ExadataInsight = adaptExadataInsightBody(response.ExadataInsight)
		return response, nil
	}
}

func wrapExadataInsightGetStatusResponse(
	get func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error),
) func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
	if get == nil {
		return nil
	}
	return func(ctx context.Context, request opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		response, err := get(ctx, request)
		if err != nil {
			return response, err
		}
		response.ExadataInsight = adaptExadataInsightBody(response.ExadataInsight)
		return response, nil
	}
}

func wrapExadataInsightListStatusResponse(
	list func(context.Context, opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error),
) func(context.Context, opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error) {
	if list == nil {
		return nil
	}
	return func(ctx context.Context, request opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error) {
		response, err := list(ctx, request)
		if err != nil {
			return response, err
		}
		for i, item := range response.Items {
			response.Items[i] = adaptExadataInsightSummary(item)
		}
		return response, nil
	}
}

type exadataInsightStatusAdapter struct {
	opsisdk.ExadataInsight
}

func (a exadataInsightStatusAdapter) MarshalJSON() ([]byte, error) {
	return exadataInsightPayloadWithSDKStatus(a.ExadataInsight)
}

type exadataInsightSummaryStatusAdapter struct {
	opsisdk.ExadataInsightSummary
}

func (a exadataInsightSummaryStatusAdapter) MarshalJSON() ([]byte, error) {
	return exadataInsightPayloadWithSDKStatus(a.ExadataInsightSummary)
}

func adaptExadataInsightBody(body opsisdk.ExadataInsight) opsisdk.ExadataInsight {
	if body == nil {
		return nil
	}
	return exadataInsightStatusAdapter{ExadataInsight: body}
}

func adaptExadataInsightSummary(body opsisdk.ExadataInsightSummary) opsisdk.ExadataInsightSummary {
	if body == nil {
		return nil
	}
	return exadataInsightSummaryStatusAdapter{ExadataInsightSummary: body}
}

func exadataInsightPayloadWithSDKStatus(body any) ([]byte, error) {
	values, err := jsonMapFromAny(body)
	if err != nil {
		return nil, err
	}
	if sdkStatus, ok := values["status"]; ok {
		values["sdkStatus"] = sdkStatus
		delete(values, "status")
	}
	delete(values, "jsonData")
	return json.Marshal(values)
}

func projectExadataInsightStatus(resource *opsiv1beta1.ExadataInsight, response any) error {
	if resource == nil {
		return fmt.Errorf("exadata insight resource is nil")
	}
	payload, ok, err := exadataInsightResponsePayload(response)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if sdkStatus, ok := payload["status"]; ok {
		payload["sdkStatus"] = sdkStatus
		delete(payload, "status")
	}
	delete(payload, "jsonData")
	projected := resource.Status
	projected.OsokStatus = resource.Status.OsokStatus
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal exadata insight status payload: %w", err)
	}
	if err := json.Unmarshal(raw, &projected); err != nil {
		return fmt.Errorf("project exadata insight status: %w", err)
	}
	projected.OsokStatus = resource.Status.OsokStatus
	resource.Status = projected
	return nil
}

type exadataInsightObservedState struct {
	entitySource      string
	freeformTags      map[string]string
	definedTags       map[string]map[string]interface{}
	isAutoSyncEnabled *bool
}

func observedExadataInsightState(resource *opsiv1beta1.ExadataInsight, response any) exadataInsightObservedState {
	state := observedExadataInsightStateFromResource(resource)
	payload, ok, _ := exadataInsightResponsePayload(response)
	if !ok {
		return state
	}
	applyExadataInsightPayloadState(&state, payload)
	return state
}

func observedExadataInsightStateFromResource(resource *opsiv1beta1.ExadataInsight) exadataInsightObservedState {
	state := exadataInsightObservedState{}
	if resource == nil {
		return state
	}
	if entitySource, err := normalizedExadataInsightEntitySource(resource.Status.EntitySource); err == nil {
		state.entitySource = entitySource
	}
	state.freeformTags = cloneStringMap(resource.Status.FreeformTags)
	state.definedTags = sharedMapToInterfaceMap(resource.Status.DefinedTags)
	if state.entitySource == string(opsisdk.ExadataEntitySourceEmManagedExternalExadata) {
		value := resource.Status.IsAutoSyncEnabled
		state.isAutoSyncEnabled = &value
	}
	return state
}

func applyExadataInsightPayloadState(state *exadataInsightObservedState, payload map[string]any) {
	if raw, ok := payload["entitySource"].(string); ok {
		if entitySource, err := normalizedExadataInsightEntitySource(raw); err == nil {
			state.entitySource = entitySource
		}
	}
	if tags, ok, err := mapStringFromAny(payload["freeformTags"]); err == nil && ok {
		state.freeformTags = tags
	}
	if tags, ok, err := mapStringInterfaceFromAny(payload["definedTags"]); err == nil && ok {
		state.definedTags = tags
	}
	if raw, ok := payload["isAutoSyncEnabled"]; ok {
		if value, ok := raw.(bool); ok {
			state.isAutoSyncEnabled = &value
		}
	}
}

func exadataInsightResponsePayload(response any) (map[string]any, bool, error) {
	body := exadataInsightResponseBody(response)
	if body == nil {
		return nil, false, nil
	}
	values, err := jsonMapFromAny(body)
	if err != nil {
		return nil, false, fmt.Errorf("decode exadata insight response payload: %w", err)
	}
	return values, true, nil
}

func exadataInsightResponseBody(response any) any {
	switch concrete := response.(type) {
	case opsisdk.GetExadataInsightResponse:
		return concrete.ExadataInsight
	case *opsisdk.GetExadataInsightResponse:
		if concrete != nil {
			return concrete.ExadataInsight
		}
	case opsisdk.CreateExadataInsightResponse:
		return concrete.ExadataInsight
	case *opsisdk.CreateExadataInsightResponse:
		if concrete != nil {
			return concrete.ExadataInsight
		}
	case opsisdk.ExadataInsight:
		return concrete
	case opsisdk.ExadataInsightSummary:
		return concrete
	}
	return nil
}

func exadataInsightResponseID(response any) string {
	payload, ok, err := exadataInsightResponsePayload(response)
	if err != nil || !ok {
		return ""
	}
	return firstNonEmptyString(payload, "id", "ocid")
}

func getExadataInsightWorkRequest(
	ctx context.Context,
	client exadataInsightOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize exadata insight OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("exadata insight OCI client is nil")
	}
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return nil, fmt.Errorf("exadata insight work request id is empty")
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveExadataInsightWorkRequestAction(workRequest any) (string, error) {
	exadataInsightWorkRequest, err := exadataInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(exadataInsightWorkRequest.OperationType), nil
}

func resolveExadataInsightWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	exadataInsightWorkRequest, err := exadataInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := exadataInsightWorkRequestPhaseFromOperationType(exadataInsightWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverExadataInsightIDFromWorkRequest(
	_ *opsiv1beta1.ExadataInsight,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	exadataInsightWorkRequest, err := exadataInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveExadataInsightIDFromWorkRequest(exadataInsightWorkRequest, exadataInsightWorkRequestActionForPhase(phase))
}

func exadataInsightWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("exadata insight work request is nil")
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("unexpected exadata insight work request type %T", workRequest)
	}
}

func exadataInsightWorkRequestPhaseFromOperationType(operationType opsisdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case opsisdk.OperationTypeCreateExadataInsight:
		return shared.OSOKAsyncPhaseCreate, true
	case opsisdk.OperationTypeUpdateExadataInsight:
		return shared.OSOKAsyncPhaseUpdate, true
	case opsisdk.OperationTypeDeleteExadataInsight:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func exadataInsightWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
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

func resolveExadataInsightIDFromWorkRequest(workRequest opsisdk.WorkRequest, action opsisdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveExadataInsightIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveExadataInsightIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("exadata insight work request %s does not expose an ExadataInsight identifier", stringValue(workRequest.Id))
}

func resolveExadataInsightIDFromResources(
	resources []opsisdk.WorkRequestResource,
	action opsisdk.ActionTypeEnum,
	preferExadataInsightOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferExadataInsightOnly && !isExadataInsightWorkRequestResource(resource) {
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

func isExadataInsightWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "exadatainsight", "exadata_insight", "exadatainsights", "exadata_insights":
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/exadatainsights/")
}

func exadataInsightWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	exadataInsightWorkRequest, err := exadataInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ExadataInsight %s work request %s is %s", phase, stringValue(exadataInsightWorkRequest.Id), exadataInsightWorkRequest.Status)
}

func listExadataInsightsAllPages(call exadataInsightListCall) exadataInsightListCall {
	return func(ctx context.Context, request opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error) {
		if call == nil {
			return opsisdk.ListExadataInsightsResponse{}, fmt.Errorf("exadata insight list operation is not configured")
		}
		accumulator := newExadataInsightListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return opsisdk.ListExadataInsightsResponse{}, err
			}
			accumulator.append(response)

			nextPage := stringValue(response.OpcNextPage)
			if nextPage == "" {
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return opsisdk.ListExadataInsightsResponse{}, err
			}
		}
	}
}

type exadataInsightListAccumulator struct {
	response  opsisdk.ListExadataInsightsResponse
	seenPages map[string]struct{}
}

func newExadataInsightListAccumulator() exadataInsightListAccumulator {
	return exadataInsightListAccumulator{seenPages: map[string]struct{}{}}
}

func (a *exadataInsightListAccumulator) append(response opsisdk.ListExadataInsightsResponse) {
	if a.response.RawResponse == nil {
		a.response.RawResponse = response.RawResponse
	}
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.OpcNextPage = response.OpcNextPage
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *exadataInsightListAccumulator) advance(request *opsisdk.ListExadataInsightsRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("exadata insight list pagination repeated page token %q", nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return nil
}

func getExadataInsightConservatively(call exadataInsightGetCall) exadataInsightGetCall {
	return func(ctx context.Context, request opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return response, ambiguousExadataInsightNotFound("read", err)
		}
		return response, nil
	}
}

func handleExadataInsightDeleteError(resource *opsiv1beta1.ExadataInsight, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return exadataInsightAmbiguousNotFoundError{
		message:      "ExadataInsight delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func ambiguousExadataInsightNotFound(operation string, err error) error {
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	return exadataInsightAmbiguousNotFoundError{
		message:      fmt.Sprintf("ExadataInsight %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: requestID,
	}
}

func wrapExadataInsightDeleteConfirmation(hooks *ExadataInsightRuntimeHooks, getExadataInsight exadataInsightGetCall) {
	if hooks == nil || getExadataInsight == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ExadataInsightServiceClient) ExadataInsightServiceClient {
		return exadataInsightDeleteConfirmationClient{
			delegate:          delegate,
			getExadataInsight: getExadataInsight,
		}
	})
}

type exadataInsightDeleteConfirmationClient struct {
	delegate          ExadataInsightServiceClient
	getExadataInsight exadataInsightGetCall
}

func (c exadataInsightDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.ExadataInsight,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c exadataInsightDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.ExadataInsight,
) (bool, error) {
	if exadataInsightHasTrackedDeleteWorkRequest(resource) {
		return c.delegate.Delete(ctx, resource)
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func exadataInsightHasTrackedDeleteWorkRequest(resource *opsiv1beta1.ExadataInsight) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		strings.TrimSpace(current.WorkRequestID) != ""
}

func (c exadataInsightDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *opsiv1beta1.ExadataInsight,
) error {
	if c.getExadataInsight == nil || resource == nil {
		return nil
	}
	exadataInsightID := trackedExadataInsightID(resource)
	if exadataInsightID == "" {
		return nil
	}
	_, err := c.getExadataInsight(ctx, opsisdk.GetExadataInsightRequest{ExadataInsightId: common.String(exadataInsightID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("exadata insight delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

type exadataInsightCreateOnlyTrackingClient struct {
	delegate ExadataInsightServiceClient
}

func (c exadataInsightCreateOnlyTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.ExadataInsight,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	recorded, hasRecorded := exadataInsightRecordedCreateOnlyFingerprint(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	switch {
	case hasRecorded && exadataInsightHasTrackedIdentity(resource):
		setExadataInsightCreateOnlyFingerprint(resource, recorded)
	case err == nil && response.IsSuccessful && exadataInsightHasTrackedIdentity(resource):
		_ = recordExadataInsightCreateOnlyFingerprint(resource)
	}
	return response, err
}

func (c exadataInsightCreateOnlyTrackingClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.ExadataInsight,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func validateExadataInsightCreateOnlyDrift(resource *opsiv1beta1.ExadataInsight, _ any) error {
	if resource == nil {
		return fmt.Errorf("exadata insight resource is nil")
	}
	recorded, ok := exadataInsightRecordedCreateOnlyFingerprint(resource)
	if !ok {
		if exadataInsightHasTrackedIdentity(resource) && resource.Status.OsokStatus.CreatedAt != nil {
			return fmt.Errorf("exadata insight create-only fields cannot be validated because the original fingerprint is missing; recreate the resource instead of changing create-only fields")
		}
		return nil
	}
	desired, err := exadataInsightCreateOnlyFingerprint(resource)
	if err != nil {
		return err
	}
	if desired != recorded {
		return fmt.Errorf("exadata insight formal semantics require replacement when create-only fields change")
	}
	return nil
}

func recordExadataInsightCreateOnlyFingerprint(resource *opsiv1beta1.ExadataInsight) error {
	fingerprint, err := exadataInsightCreateOnlyFingerprint(resource)
	if err != nil {
		return err
	}
	setExadataInsightCreateOnlyFingerprint(resource, fingerprint)
	return nil
}

func setExadataInsightCreateOnlyFingerprint(resource *opsiv1beta1.ExadataInsight, fingerprint string) {
	if resource == nil {
		return
	}
	base := stripExadataInsightCreateOnlyFingerprint(resource.Status.OsokStatus.Message)
	marker := exadataInsightCreateOnlyFingerprintKey + fingerprint
	if base == "" {
		resource.Status.OsokStatus.Message = marker
		return
	}
	resource.Status.OsokStatus.Message = base + "; " + marker
}

func exadataInsightRecordedCreateOnlyFingerprint(resource *opsiv1beta1.ExadataInsight) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, exadataInsightCreateOnlyFingerprintKey)
	if index < 0 {
		return "", false
	}
	start := index + len(exadataInsightCreateOnlyFingerprintKey)
	end := start
	for end < len(raw) && isExadataInsightHexDigit(raw[end]) {
		end++
	}
	fingerprint := raw[start:end]
	if len(fingerprint) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", false
	}
	return fingerprint, true
}

func stripExadataInsightCreateOnlyFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, exadataInsightCreateOnlyFingerprintKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(exadataInsightCreateOnlyFingerprintKey)
	end := start
	for end < len(raw) && isExadataInsightHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func exadataInsightCreateOnlyFingerprint(resource *opsiv1beta1.ExadataInsight) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("exadata insight resource is nil")
	}
	entitySource, err := normalizedExadataInsightEntitySource(resource.Spec.EntitySource)
	if err != nil {
		return "", err
	}
	payload := struct {
		CompartmentId                     string                                            `json:"compartmentId"`
		EntitySource                      string                                            `json:"entitySource"`
		EnterpriseManagerIdentifier       string                                            `json:"enterpriseManagerIdentifier,omitempty"`
		EnterpriseManagerBridgeId         string                                            `json:"enterpriseManagerBridgeId,omitempty"`
		EnterpriseManagerEntityIdentifier string                                            `json:"enterpriseManagerEntityIdentifier,omitempty"`
		ExadataInfraId                    string                                            `json:"exadataInfraId,omitempty"`
		JsonData                          string                                            `json:"jsonData,omitempty"`
		MemberEntityDetails               []opsiv1beta1.ExadataInsightMemberEntityDetail    `json:"memberEntityDetails,omitempty"`
		MemberVmClusterDetails            []opsiv1beta1.ExadataInsightMemberVmClusterDetail `json:"memberVmClusterDetails,omitempty"`
	}{
		CompartmentId:                     strings.TrimSpace(resource.Spec.CompartmentId),
		EntitySource:                      entitySource,
		EnterpriseManagerIdentifier:       strings.TrimSpace(resource.Spec.EnterpriseManagerIdentifier),
		EnterpriseManagerBridgeId:         strings.TrimSpace(resource.Spec.EnterpriseManagerBridgeId),
		EnterpriseManagerEntityIdentifier: strings.TrimSpace(resource.Spec.EnterpriseManagerEntityIdentifier),
		ExadataInfraId:                    strings.TrimSpace(resource.Spec.ExadataInfraId),
		JsonData:                          normalizedExadataInsightCreateOnlyJSONData(resource.Spec.JsonData),
		MemberEntityDetails:               resource.Spec.MemberEntityDetails,
		MemberVmClusterDetails:            resource.Spec.MemberVmClusterDetails,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal exadata insight create-only fingerprint: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func normalizedExadataInsightCreateOnlyJSONData(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return raw
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return raw
	}
	return string(normalized)
}

func exadataInsightHasTrackedIdentity(resource *opsiv1beta1.ExadataInsight) bool {
	return trackedExadataInsightID(resource) != ""
}

func trackedExadataInsightID(resource *opsiv1beta1.ExadataInsight) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func isExadataInsightHexDigit(value byte) bool {
	return ('0' <= value && value <= '9') ||
		('a' <= value && value <= 'f') ||
		('A' <= value && value <= 'F')
}

func desiredStringMap(values map[string]any, key string) (map[string]string, bool, error) {
	raw, ok := values[key]
	if !ok {
		return nil, false, nil
	}
	return mapStringFromAny(raw)
}

func desiredDefinedTags(values map[string]any, key string) (map[string]map[string]interface{}, bool, error) {
	raw, ok := values[key]
	if !ok {
		return nil, false, nil
	}
	return mapStringInterfaceFromAny(raw)
}

func desiredBool(values map[string]any, key string) (bool, bool, error) {
	raw, ok := values[key]
	if !ok {
		return false, false, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, false, fmt.Errorf("exadata insight %s resolved to %T, want bool", key, raw)
	}
	return value, true, nil
}

func mapStringFromAny(raw any) (map[string]string, bool, error) {
	if raw == nil {
		return nil, true, nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, false, err
	}
	var values map[string]string
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, false, err
	}
	return values, true, nil
}

func mapStringInterfaceFromAny(raw any) (map[string]map[string]interface{}, bool, error) {
	if raw == nil {
		return nil, true, nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, false, err
	}
	var values map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, false, err
	}
	return values, true, nil
}

func sharedMapToInterfaceMap(values map[string]shared.MapValue) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(values))
	for namespace, tags := range values {
		converted[namespace] = make(map[string]interface{}, len(tags))
		for key, value := range tags {
			converted[namespace][key] = value
		}
	}
	return converted
}

func jsonMapFromAny(value any) (map[string]any, error) {
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

func firstNonEmptyString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func jsonEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprintf("%#v", left) == fmt.Sprintf("%#v", right)
	}
	return string(leftPayload) == string(rightPayload)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
