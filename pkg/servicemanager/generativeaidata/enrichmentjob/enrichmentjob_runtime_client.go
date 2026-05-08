/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package enrichmentjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaidatasdk "github.com/oracle/oci-go-sdk/v65/generativeaidata"
	generativeaidatav1beta1 "github.com/oracle/oci-service-operator/api/generativeaidata/v1beta1"
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

const enrichmentJobRequeueDuration = time.Minute

type enrichmentJobOCIClient interface {
	GenerateEnrichmentJob(context.Context, generativeaidatasdk.GenerateEnrichmentJobRequest) (generativeaidatasdk.GenerateEnrichmentJobResponse, error)
	GetEnrichmentJob(context.Context, generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error)
	ListEnrichmentJobs(context.Context, generativeaidatasdk.ListEnrichmentJobsRequest) (generativeaidatasdk.ListEnrichmentJobsResponse, error)
	CancelEnrichmentJob(context.Context, generativeaidatasdk.CancelEnrichmentJobRequest) (generativeaidatasdk.CancelEnrichmentJobResponse, error)
}

type enrichmentJobRuntimeClient struct {
	delegate EnrichmentJobServiceClient
	client   enrichmentJobOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

type enrichmentJobIdentity struct {
	SemanticStoreID string
	CompartmentID   string
	DisplayName     string
}

var _ EnrichmentJobServiceClient = (*enrichmentJobRuntimeClient)(nil)

func (c EnrichmentJobSDKClients) GenerateEnrichmentJob(
	ctx context.Context,
	request generativeaidatasdk.GenerateEnrichmentJobRequest,
) (generativeaidatasdk.GenerateEnrichmentJobResponse, error) {
	return c.generateEnrichmentJobClient.GenerateEnrichmentJob(ctx, request)
}

func (c EnrichmentJobSDKClients) GetEnrichmentJob(
	ctx context.Context,
	request generativeaidatasdk.GetEnrichmentJobRequest,
) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
	return c.getEnrichmentJobClient.GetEnrichmentJob(ctx, request)
}

func (c EnrichmentJobSDKClients) ListEnrichmentJobs(
	ctx context.Context,
	request generativeaidatasdk.ListEnrichmentJobsRequest,
) (generativeaidatasdk.ListEnrichmentJobsResponse, error) {
	return c.listEnrichmentJobsClient.ListEnrichmentJobs(ctx, request)
}

func (c EnrichmentJobSDKClients) CancelEnrichmentJob(
	ctx context.Context,
	request generativeaidatasdk.CancelEnrichmentJobRequest,
) (generativeaidatasdk.CancelEnrichmentJobResponse, error) {
	return c.cancelEnrichmentJobClient.CancelEnrichmentJob(ctx, request)
}

func init() {
	registerEnrichmentJobRuntimeHooksMutator(func(manager *EnrichmentJobServiceManager, hooks *EnrichmentJobRuntimeHooks) {
		client, initErr := newEnrichmentJobSDKClient(manager)
		applyEnrichmentJobRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newEnrichmentJobSDKClient(manager *EnrichmentJobServiceManager) (enrichmentJobOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("EnrichmentJob service manager is nil")
	}
	return newEnrichmentJobSDKClients(manager)
}

func applyEnrichmentJobRuntimeHooks(
	manager *EnrichmentJobServiceManager,
	hooks *EnrichmentJobRuntimeHooks,
	client enrichmentJobOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	runtimeClient := newEnrichmentJobRuntimeClient(manager, nil, client, initErr)
	hooks.Semantics = reviewedEnrichmentJobRuntimeSemantics()
	hooks.Create.Fields = enrichmentJobCreateFields()
	hooks.Get.Fields = enrichmentJobGetFields()
	hooks.List.Fields = enrichmentJobListFields()
	hooks.Delete.Fields = enrichmentJobDeleteFields()
	hooks.Identity.Resolve = resolveEnrichmentJobIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardEnrichmentJobExistingBeforeCreate
	hooks.Identity.LookupExisting = runtimeClient.lookupExisting
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateEnrichmentJobCreateOnlyDrift
	hooks.StatusHooks.MarkDeleted = runtimeClient.markDeleted
	hooks.DeleteHooks.ConfirmRead = runtimeClient.confirmDeleteRead
	hooks.DeleteHooks.HandleError = runtimeClient.handleDeleteError
	hooks.DeleteHooks.ApplyOutcome = runtimeClient.applyDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate EnrichmentJobServiceClient) EnrichmentJobServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func newEnrichmentJobRuntimeClient(
	manager *EnrichmentJobServiceManager,
	delegate EnrichmentJobServiceClient,
	client enrichmentJobOCIClient,
	initErr error,
) *enrichmentJobRuntimeClient {
	runtimeClient := &enrichmentJobRuntimeClient{
		delegate: delegate,
		client:   client,
		initErr:  initErr,
	}
	if manager != nil {
		runtimeClient.log = manager.Log
	}
	return runtimeClient
}

func newEnrichmentJobServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client enrichmentJobOCIClient,
) EnrichmentJobServiceClient {
	manager := &EnrichmentJobServiceManager{Log: log}
	hooks := newEnrichmentJobRuntimeHooksWithOCIClient(client)
	applyEnrichmentJobRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultEnrichmentJobServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*generativeaidatav1beta1.EnrichmentJob](
			buildEnrichmentJobGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapEnrichmentJobGeneratedClient(hooks, delegate)
}

func newEnrichmentJobRuntimeHooksWithOCIClient(client enrichmentJobOCIClient) EnrichmentJobRuntimeHooks {
	return EnrichmentJobRuntimeHooks{
		Semantics:       reviewedEnrichmentJobRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*generativeaidatav1beta1.EnrichmentJob]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*generativeaidatav1beta1.EnrichmentJob]{},
		StatusHooks:     generatedruntime.StatusHooks[*generativeaidatav1beta1.EnrichmentJob]{},
		ParityHooks:     generatedruntime.ParityHooks[*generativeaidatav1beta1.EnrichmentJob]{},
		Async:           generatedruntime.AsyncHooks[*generativeaidatav1beta1.EnrichmentJob]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*generativeaidatav1beta1.EnrichmentJob]{},
		Create: runtimeOperationHooks[generativeaidatasdk.GenerateEnrichmentJobRequest, generativeaidatasdk.GenerateEnrichmentJobResponse]{
			Fields: enrichmentJobCreateFields(),
			Call: func(ctx context.Context, request generativeaidatasdk.GenerateEnrichmentJobRequest) (generativeaidatasdk.GenerateEnrichmentJobResponse, error) {
				return client.GenerateEnrichmentJob(ctx, request)
			},
		},
		Get: runtimeOperationHooks[generativeaidatasdk.GetEnrichmentJobRequest, generativeaidatasdk.GetEnrichmentJobResponse]{
			Fields: enrichmentJobGetFields(),
			Call: func(ctx context.Context, request generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
				return client.GetEnrichmentJob(ctx, request)
			},
		},
		List: runtimeOperationHooks[generativeaidatasdk.ListEnrichmentJobsRequest, generativeaidatasdk.ListEnrichmentJobsResponse]{
			Fields: enrichmentJobListFields(),
			Call: func(ctx context.Context, request generativeaidatasdk.ListEnrichmentJobsRequest) (generativeaidatasdk.ListEnrichmentJobsResponse, error) {
				return client.ListEnrichmentJobs(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[generativeaidatasdk.CancelEnrichmentJobRequest, generativeaidatasdk.CancelEnrichmentJobResponse]{
			Fields: enrichmentJobDeleteFields(),
			Call: func(ctx context.Context, request generativeaidatasdk.CancelEnrichmentJobRequest) (generativeaidatasdk.CancelEnrichmentJobResponse, error) {
				return client.CancelEnrichmentJob(ctx, request)
			},
		},
	}
}

func reviewedEnrichmentJobRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "generativeaidata",
		FormalSlug:    "enrichmentjob",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{
				string(generativeaidatasdk.LifecycleStateAccepted),
				string(generativeaidatasdk.LifecycleStateInProgress),
			},
			ActiveStates: []string{
				string(generativeaidatasdk.LifecycleStateSucceeded),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(generativeaidatasdk.LifecycleStateCanceling),
			},
			TerminalStates: []string{
				string(generativeaidatasdk.LifecycleStateCanceled),
				string(generativeaidatasdk.LifecycleStateFailed),
				"NOT_FOUND",
				string(generativeaidatasdk.LifecycleStateSucceeded),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"displayName", "semanticStoreId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{},
			ForceNew: []string{"compartmentId", "semanticStoreId"},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "none",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func enrichmentJobCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SemanticStoreId",
			RequestName:  "semanticStoreId",
			Contribution: "path",
			LookupPaths:  []string{"status.semanticStoreId", "spec.semanticStoreId", "semanticStoreId"},
		},
		{FieldName: "GenerateEnrichmentJobDetails", RequestName: "GenerateEnrichmentJobDetails", Contribution: "body"},
	}
}

func enrichmentJobGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SemanticStoreId",
			RequestName:  "semanticStoreId",
			Contribution: "path",
			LookupPaths:  []string{"status.semanticStoreId", "spec.semanticStoreId", "semanticStoreId"},
		},
		{FieldName: "EnrichmentJobId", RequestName: "enrichmentJobId", Contribution: "path", PreferResourceID: true},
	}
}

func enrichmentJobListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SemanticStoreId",
			RequestName:  "semanticStoreId",
			Contribution: "path",
			LookupPaths:  []string{"status.semanticStoreId", "spec.semanticStoreId", "semanticStoreId"},
		},
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

func enrichmentJobDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SemanticStoreId",
			RequestName:  "semanticStoreId",
			Contribution: "path",
			LookupPaths:  []string{"status.semanticStoreId", "spec.semanticStoreId", "semanticStoreId"},
		},
		{FieldName: "EnrichmentJobId", RequestName: "enrichmentJobId", Contribution: "path", PreferResourceID: true},
	}
}

func guardEnrichmentJobExistingBeforeCreate(
	_ context.Context,
	resource *generativeaidatav1beta1.EnrichmentJob,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("EnrichmentJob resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveEnrichmentJobIdentity(resource *generativeaidatav1beta1.EnrichmentJob) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("EnrichmentJob resource is nil")
	}
	return enrichmentJobIdentity{
		SemanticStoreID: strings.TrimSpace(resource.Spec.SemanticStoreId),
		CompartmentID:   strings.TrimSpace(resource.Spec.CompartmentId),
		DisplayName:     strings.TrimSpace(resource.Spec.DisplayName),
	}, nil
}

func (c *enrichmentJobRuntimeClient) lookupExisting(
	ctx context.Context,
	resource *generativeaidatav1beta1.EnrichmentJob,
	_ any,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("EnrichmentJob resource is nil")
	}
	if c.client == nil {
		if c.initErr != nil {
			return nil, c.initErr
		}
		return nil, fmt.Errorf("EnrichmentJob OCI client is not configured")
	}

	semanticStoreID := currentEnrichmentJobSemanticStoreID(resource)
	compartmentID := currentEnrichmentJobCompartmentID(resource)
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if semanticStoreID == "" || compartmentID == "" || displayName == "" {
		return nil, nil
	}

	response, err := c.client.ListEnrichmentJobs(ctx, generativeaidatasdk.ListEnrichmentJobsRequest{
		SemanticStoreId: common.String(semanticStoreID),
		CompartmentId:   common.String(compartmentID),
		DisplayName:     common.String(displayName),
	})
	if err != nil {
		return nil, normalizeEnrichmentJobOCIError(err)
	}

	matches := matchingEnrichmentJobSummaries(response.Items, currentEnrichmentJobID(resource), displayName)
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("EnrichmentJob list response returned multiple exact displayName matches in semantic store %q and compartment %q", semanticStoreID, compartmentID)
	}
}

func matchingEnrichmentJobSummaries(
	items []generativeaidatasdk.EnrichmentJobSummary,
	currentID string,
	displayName string,
) []generativeaidatasdk.EnrichmentJobSummary {
	var matches []generativeaidatasdk.EnrichmentJobSummary
	for _, item := range items {
		itemID := strings.TrimSpace(enrichmentJobStringValue(item.Id))
		itemDisplayName := strings.TrimSpace(enrichmentJobStringValue(item.DisplayName))
		switch {
		case currentID != "" && itemID == currentID:
			matches = append(matches, item)
		case displayName != "" && itemDisplayName == displayName:
			matches = append(matches, item)
		}
	}
	return matches
}

func enrichmentJobStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func validateEnrichmentJobCreateOnlyDrift(
	resource *generativeaidatav1beta1.EnrichmentJob,
	currentResponse any,
) error {
	if resource == nil || currentResponse == nil {
		return nil
	}

	status := resource.Status
	var drift []string
	if status.CompartmentId != "" && strings.TrimSpace(resource.Spec.CompartmentId) != strings.TrimSpace(status.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if status.SemanticStoreId != "" && strings.TrimSpace(resource.Spec.SemanticStoreId) != strings.TrimSpace(status.SemanticStoreId) {
		drift = append(drift, "semanticStoreId")
	}
	if resource.Spec.EnrichmentJobType != "" && resource.Spec.EnrichmentJobType != status.EnrichmentJobType {
		drift = append(drift, "enrichmentJobType")
	}
	if strings.TrimSpace(resource.Spec.Description) != strings.TrimSpace(status.Description) {
		drift = append(drift, "description")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != strings.TrimSpace(status.DisplayName) {
		drift = append(drift, "displayName")
	}
	if !enrichmentJobJSONEqual(resource.Spec.FreeformTags, status.FreeformTags) {
		drift = append(drift, "freeformTags")
	}
	if !enrichmentJobJSONEqual(resource.Spec.DefinedTags, status.DefinedTags) {
		drift = append(drift, "definedTags")
	}
	if !enrichmentJobConfigurationEqual(resource.Spec.EnrichmentJobConfiguration, status.EnrichmentJobConfiguration) {
		drift = append(drift, "enrichmentJobConfiguration")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("EnrichmentJob runtime rejects create-only drift for %s", strings.Join(drift, ", "))
}

func enrichmentJobConfigurationEqual(
	spec generativeaidatav1beta1.EnrichmentJobConfiguration,
	status generativeaidatav1beta1.EnrichmentJobConfiguration,
) bool {
	return enrichmentJobJSONEqual(
		normalizeEnrichmentJobConfiguration(spec),
		normalizeEnrichmentJobConfiguration(status),
	)
}

func normalizeEnrichmentJobConfiguration(
	config generativeaidatav1beta1.EnrichmentJobConfiguration,
) map[string]any {
	payload, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil
	}
	delete(values, "jsonData")
	return values
}

func enrichmentJobJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func (c *enrichmentJobRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaidatav1beta1.EnrichmentJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("EnrichmentJob generated runtime delegate is not configured")
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if resource != nil {
		persistEnrichmentJobLookupStatus(resource)
	}
	if err != nil {
		return response, err
	}

	switch lifecycleState := normalizedEnrichmentJobLifecycle(resource.Status.LifecycleState); lifecycleState {
	case string(generativeaidatasdk.LifecycleStateSucceeded):
		return c.markSucceeded(resource), nil
	case string(generativeaidatasdk.LifecycleStateFailed):
		return c.applyLifecycleOverride(
			resource,
			enrichmentJobFailurePhase(resource),
			shared.OSOKAsyncClassFailed,
			lifecycleState,
			enrichmentJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI EnrichmentJob failed"),
		), nil
	case string(generativeaidatasdk.LifecycleStateCanceling):
		return c.applyLifecycleOverride(
			resource,
			shared.OSOKAsyncPhaseDelete,
			shared.OSOKAsyncClassPending,
			lifecycleState,
			enrichmentJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI EnrichmentJob cancellation is in progress"),
		), nil
	case string(generativeaidatasdk.LifecycleStateCanceled):
		return c.applyLifecycleOverride(
			resource,
			enrichmentJobFailurePhase(resource),
			shared.OSOKAsyncClassCanceled,
			lifecycleState,
			enrichmentJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI EnrichmentJob was canceled"),
		), nil
	default:
		return response, nil
	}
}

func (c *enrichmentJobRuntimeClient) Delete(
	ctx context.Context,
	resource *generativeaidatav1beta1.EnrichmentJob,
) (bool, error) {
	if resource != nil && currentEnrichmentJobID(resource) == "" {
		c.markDeleted(resource, "OCI EnrichmentJob identifier is not recorded")
		return true, nil
	}
	if c.delegate == nil {
		if c.initErr != nil {
			return false, c.initErr
		}
		return false, fmt.Errorf("EnrichmentJob generated runtime delegate is not configured")
	}
	deleted, err := c.delegate.Delete(ctx, resource)
	if resource != nil {
		persistEnrichmentJobLookupStatus(resource)
	}
	return deleted, err
}

func (c *enrichmentJobRuntimeClient) confirmDeleteRead(
	ctx context.Context,
	resource *generativeaidatav1beta1.EnrichmentJob,
	currentID string,
) (any, error) {
	if c.client == nil {
		if c.initErr != nil {
			return nil, c.initErr
		}
		return nil, fmt.Errorf("EnrichmentJob OCI client is not configured")
	}
	if strings.TrimSpace(currentID) == "" {
		currentID = currentEnrichmentJobID(resource)
	}
	if currentID == "" {
		return nil, fmt.Errorf("EnrichmentJob OCI resource identifier is not recorded")
	}
	semanticStoreID := currentEnrichmentJobSemanticStoreID(resource)
	if semanticStoreID == "" {
		return nil, fmt.Errorf("EnrichmentJob semanticStoreId is not recorded")
	}
	return c.client.GetEnrichmentJob(ctx, generativeaidatasdk.GetEnrichmentJobRequest{
		SemanticStoreId: common.String(semanticStoreID),
		EnrichmentJobId: common.String(currentID),
	})
}

func (c *enrichmentJobRuntimeClient) handleDeleteError(
	resource *generativeaidatav1beta1.EnrichmentJob,
	err error,
) error {
	normalizedErr := normalizeEnrichmentJobOCIError(err)
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
		persistEnrichmentJobLookupStatus(resource)
	}
	return normalizedErr
}

func (c *enrichmentJobRuntimeClient) applyDeleteOutcome(
	resource *generativeaidatav1beta1.EnrichmentJob,
	_ any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	lifecycleState := normalizedEnrichmentJobLifecycle(resource.Status.LifecycleState)

	switch stage {
	case generatedruntime.DeleteConfirmStageAlreadyPending:
		switch {
		case enrichmentJobDeleteTerminal(lifecycleState):
			c.markDeleted(resource, enrichmentJobDeleteTerminalMessage(lifecycleState))
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
		case enrichmentJobDeleteAlreadyPending(resource, lifecycleState):
			c.markDeletePending(resource, lifecycleState)
			return generatedruntime.DeleteOutcome{Handled: true}, nil
		default:
			return generatedruntime.DeleteOutcome{}, nil
		}
	case generatedruntime.DeleteConfirmStageAfterRequest:
		switch {
		case enrichmentJobDeleteTerminal(lifecycleState):
			c.markDeleted(resource, enrichmentJobDeleteTerminalMessage(lifecycleState))
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
		case enrichmentJobDeletePendingAfterRequest(lifecycleState):
			c.markDeletePending(resource, lifecycleState)
			return generatedruntime.DeleteOutcome{Handled: true}, nil
		default:
			c.markDeleteRetry(resource, lifecycleState)
			return generatedruntime.DeleteOutcome{Handled: true}, nil
		}
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func (c *enrichmentJobRuntimeClient) applyLifecycleOverride(
	resource *generativeaidatav1beta1.EnrichmentJob,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	lifecycleState string,
	message string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	if currentID := currentEnrichmentJobID(resource); currentID != "" && status.Ocid == "" {
		status.Ocid = shared.OCID(currentID)
	}
	persistEnrichmentJobLookupStatus(resource)

	now := metav1.Now()
	projection := servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       lifecycleState,
		NormalizedClass: class,
		Message:         message,
		UpdatedAt:       &now,
	}, c.log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: enrichmentJobRequeueDuration,
	}
}

func (c *enrichmentJobRuntimeClient) markDeletePending(
	resource *generativeaidatav1beta1.EnrichmentJob,
	lifecycleState string,
) {
	_ = c.applyLifecycleOverride(
		resource,
		shared.OSOKAsyncPhaseDelete,
		shared.OSOKAsyncClassPending,
		lifecycleState,
		enrichmentJobDeletePendingMessage(lifecycleState),
	)
}

func (c *enrichmentJobRuntimeClient) markDeleteRetry(
	resource *generativeaidatav1beta1.EnrichmentJob,
	lifecycleState string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = enrichmentJobDeleteRetryMessage(lifecycleState)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.log)
	persistEnrichmentJobLookupStatus(resource)
}

func (c *enrichmentJobRuntimeClient) markSucceeded(
	resource *generativeaidatav1beta1.EnrichmentJob,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = enrichmentJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI EnrichmentJob is ready")
	status.Reason = string(shared.Active)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", status.Message, c.log)
	persistEnrichmentJobLookupStatus(resource)
	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *enrichmentJobRuntimeClient) markDeleted(
	resource *generativeaidatav1beta1.EnrichmentJob,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.log)
	persistEnrichmentJobLookupStatus(resource)
}

func persistEnrichmentJobLookupStatus(resource *generativeaidatav1beta1.EnrichmentJob) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(resource.Status.CompartmentId) == "" {
		resource.Status.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	}
	if strings.TrimSpace(resource.Status.SemanticStoreId) == "" {
		resource.Status.SemanticStoreId = strings.TrimSpace(resource.Spec.SemanticStoreId)
	}
	if resource.Status.OsokStatus.Ocid == "" && strings.TrimSpace(resource.Status.Id) != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resource.Status.Id))
	}
}

func currentEnrichmentJobID(resource *generativeaidatav1beta1.EnrichmentJob) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentEnrichmentJobSemanticStoreID(resource *generativeaidatav1beta1.EnrichmentJob) string {
	if resource == nil {
		return ""
	}
	if strings.TrimSpace(resource.Status.SemanticStoreId) != "" {
		return strings.TrimSpace(resource.Status.SemanticStoreId)
	}
	return strings.TrimSpace(resource.Spec.SemanticStoreId)
}

func currentEnrichmentJobCompartmentID(resource *generativeaidatav1beta1.EnrichmentJob) string {
	if resource == nil {
		return ""
	}
	if strings.TrimSpace(resource.Status.CompartmentId) != "" {
		return strings.TrimSpace(resource.Status.CompartmentId)
	}
	return strings.TrimSpace(resource.Spec.CompartmentId)
}

func enrichmentJobFailurePhase(resource *generativeaidatav1beta1.EnrichmentJob) shared.OSOKAsyncPhase {
	if resource != nil && resource.Status.OsokStatus.Async.Current != nil {
		phase := resource.Status.OsokStatus.Async.Current.Phase
		if phase != "" && phase != shared.OSOKAsyncPhaseDelete {
			return phase
		}
	}
	return shared.OSOKAsyncPhaseCreate
}

func enrichmentJobDeleteTracked(resource *generativeaidatav1beta1.EnrichmentJob) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func enrichmentJobDeleteAlreadyPending(resource *generativeaidatav1beta1.EnrichmentJob, lifecycleState string) bool {
	switch lifecycleState {
	case string(generativeaidatasdk.LifecycleStateCanceling):
		return true
	case string(generativeaidatasdk.LifecycleStateAccepted),
		string(generativeaidatasdk.LifecycleStateInProgress):
		return enrichmentJobDeleteTracked(resource)
	default:
		return false
	}
}

func enrichmentJobDeletePendingAfterRequest(lifecycleState string) bool {
	switch lifecycleState {
	case string(generativeaidatasdk.LifecycleStateAccepted),
		string(generativeaidatasdk.LifecycleStateInProgress),
		string(generativeaidatasdk.LifecycleStateCanceling):
		return true
	default:
		return false
	}
}

func enrichmentJobDeleteTerminal(lifecycleState string) bool {
	switch lifecycleState {
	case string(generativeaidatasdk.LifecycleStateCanceled),
		string(generativeaidatasdk.LifecycleStateFailed),
		string(generativeaidatasdk.LifecycleStateSucceeded):
		return true
	default:
		return false
	}
}

func normalizedEnrichmentJobLifecycle(lifecycleState string) string {
	return strings.ToUpper(strings.TrimSpace(lifecycleState))
}

func enrichmentJobLifecycleMessage(lifecycleDetails string, fallback string) string {
	if details := strings.TrimSpace(lifecycleDetails); details != "" {
		return details
	}
	return fallback
}

func enrichmentJobDeletePendingMessage(lifecycleState string) string {
	switch normalizedEnrichmentJobLifecycle(lifecycleState) {
	case string(generativeaidatasdk.LifecycleStateCanceling):
		return "OCI EnrichmentJob cancellation is in progress"
	default:
		return "OCI EnrichmentJob cancellation was requested; waiting for OCI to reflect the cancel state"
	}
}

func enrichmentJobDeleteRetryMessage(lifecycleState string) string {
	lifecycleState = normalizedEnrichmentJobLifecycle(lifecycleState)
	if lifecycleState == "" {
		return "OCI EnrichmentJob cancellation has not started yet; retrying cancel confirmation"
	}
	return fmt.Sprintf("OCI EnrichmentJob cancellation is not in progress yet; current lifecycle state is %s; retrying cancel confirmation", lifecycleState)
}

func enrichmentJobDeleteTerminalMessage(lifecycleState string) string {
	switch normalizedEnrichmentJobLifecycle(lifecycleState) {
	case string(generativeaidatasdk.LifecycleStateCanceled):
		return "OCI EnrichmentJob cancellation completed; Kubernetes finalizer can be removed"
	case string(generativeaidatasdk.LifecycleStateSucceeded):
		return "OCI EnrichmentJob already succeeded; Kubernetes finalizer can be removed without issuing cancel"
	case string(generativeaidatasdk.LifecycleStateFailed):
		return "OCI EnrichmentJob already failed; Kubernetes finalizer can be removed without issuing cancel"
	default:
		return "OCI EnrichmentJob reached a terminal lifecycle state; Kubernetes finalizer can be removed"
	}
}

func normalizeEnrichmentJobOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}
