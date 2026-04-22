/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package transcriptionjob

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/common"
	aispeechv1beta1 "github.com/oracle/oci-service-operator/api/aispeech/v1beta1"
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

const transcriptionJobRequeueDuration = time.Minute

type transcriptionJobOCIClient interface {
	CreateTranscriptionJob(context.Context, aispeech.CreateTranscriptionJobRequest) (aispeech.CreateTranscriptionJobResponse, error)
	GetTranscriptionJob(context.Context, aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error)
	ListTranscriptionJobs(context.Context, aispeech.ListTranscriptionJobsRequest) (aispeech.ListTranscriptionJobsResponse, error)
	UpdateTranscriptionJob(context.Context, aispeech.UpdateTranscriptionJobRequest) (aispeech.UpdateTranscriptionJobResponse, error)
	DeleteTranscriptionJob(context.Context, aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error)
}

type transcriptionJobRuntimeClient struct {
	delegate TranscriptionJobServiceClient
	client   transcriptionJobOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

var _ TranscriptionJobServiceClient = (*transcriptionJobRuntimeClient)(nil)

func init() {
	registerTranscriptionJobRuntimeHooksMutator(func(manager *TranscriptionJobServiceManager, hooks *TranscriptionJobRuntimeHooks) {
		client, initErr := newTranscriptionJobSDKClient(manager)
		applyTranscriptionJobRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newTranscriptionJobSDKClient(manager *TranscriptionJobServiceManager) (transcriptionJobOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("TranscriptionJob service manager is nil")
	}
	client, err := aispeech.NewAIServiceSpeechClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTranscriptionJobRuntimeHooks(
	manager *TranscriptionJobServiceManager,
	hooks *TranscriptionJobRuntimeHooks,
	client transcriptionJobOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	runtimeClient := newTranscriptionJobRuntimeClient(manager, nil, client, initErr)
	hooks.Semantics = reviewedTranscriptionJobRuntimeSemantics()
	hooks.Create.Fields = transcriptionJobCreateFields()
	hooks.Get.Fields = transcriptionJobGetFields()
	hooks.List.Fields = transcriptionJobListFields()
	hooks.Update.Fields = transcriptionJobUpdateFields()
	hooks.Delete.Fields = transcriptionJobDeleteFields()
	hooks.DeleteHooks.ConfirmRead = runtimeClient.confirmDeleteRead
	hooks.DeleteHooks.HandleError = runtimeClient.handleDeleteError
	hooks.DeleteHooks.ApplyOutcome = runtimeClient.applyDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TranscriptionJobServiceClient) TranscriptionJobServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func newTranscriptionJobRuntimeClient(
	manager *TranscriptionJobServiceManager,
	delegate TranscriptionJobServiceClient,
	client transcriptionJobOCIClient,
	initErr error,
) *transcriptionJobRuntimeClient {
	runtimeClient := &transcriptionJobRuntimeClient{
		delegate: delegate,
		client:   client,
		initErr:  initErr,
	}
	if manager != nil {
		runtimeClient.log = manager.Log
	}
	return runtimeClient
}

func newTranscriptionJobServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client transcriptionJobOCIClient,
) TranscriptionJobServiceClient {
	manager := &TranscriptionJobServiceManager{Log: log}
	hooks := newTranscriptionJobRuntimeHooksWithOCIClient(client)
	applyTranscriptionJobRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultTranscriptionJobServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*aispeechv1beta1.TranscriptionJob](
			buildTranscriptionJobGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapTranscriptionJobGeneratedClient(hooks, delegate)
}

func newTranscriptionJobRuntimeHooksWithOCIClient(client transcriptionJobOCIClient) TranscriptionJobRuntimeHooks {
	return TranscriptionJobRuntimeHooks{
		Semantics: reviewedTranscriptionJobRuntimeSemantics(),
		Create: runtimeOperationHooks[aispeech.CreateTranscriptionJobRequest, aispeech.CreateTranscriptionJobResponse]{
			Fields: transcriptionJobCreateFields(),
			Call: func(ctx context.Context, request aispeech.CreateTranscriptionJobRequest) (aispeech.CreateTranscriptionJobResponse, error) {
				return client.CreateTranscriptionJob(ctx, request)
			},
		},
		Get: runtimeOperationHooks[aispeech.GetTranscriptionJobRequest, aispeech.GetTranscriptionJobResponse]{
			Fields: transcriptionJobGetFields(),
			Call: func(ctx context.Context, request aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
				return client.GetTranscriptionJob(ctx, request)
			},
		},
		List: runtimeOperationHooks[aispeech.ListTranscriptionJobsRequest, aispeech.ListTranscriptionJobsResponse]{
			Fields: transcriptionJobListFields(),
			Call: func(ctx context.Context, request aispeech.ListTranscriptionJobsRequest) (aispeech.ListTranscriptionJobsResponse, error) {
				return client.ListTranscriptionJobs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[aispeech.UpdateTranscriptionJobRequest, aispeech.UpdateTranscriptionJobResponse]{
			Fields: transcriptionJobUpdateFields(),
			Call: func(ctx context.Context, request aispeech.UpdateTranscriptionJobRequest) (aispeech.UpdateTranscriptionJobResponse, error) {
				return client.UpdateTranscriptionJob(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[aispeech.DeleteTranscriptionJobRequest, aispeech.DeleteTranscriptionJobResponse]{
			Fields: transcriptionJobDeleteFields(),
			Call: func(ctx context.Context, request aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
				return client.DeleteTranscriptionJob(ctx, request)
			},
		},
	}
}

// Delete confirmation stays on DeleteHooks so the steady-state observe path can
// keep CANCELED distinct from a completed delete.
func reviewedTranscriptionJobRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "aispeech",
		FormalSlug:    "transcriptionjob",
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
				string(aispeech.TranscriptionJobLifecycleStateAccepted),
				string(aispeech.TranscriptionJobLifecycleStateInProgress),
			},
			ActiveStates: []string{
				string(aispeech.TranscriptionJobLifecycleStateSucceeded),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{"NOT_FOUND"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"definedTags", "description", "displayName", "freeformTags"},
			ForceNew:      []string{"compartmentId"},
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

func transcriptionJobCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateTranscriptionJobDetails", RequestName: "CreateTranscriptionJobDetails", Contribution: "body"},
	}
}

func transcriptionJobGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TranscriptionJobId", RequestName: "transcriptionJobId", Contribution: "path", PreferResourceID: true},
	}
}

func transcriptionJobListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", LookupPaths: []string{"status.id", "status.ocid", "id", "ocid"}},
	}
}

func transcriptionJobUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TranscriptionJobId", RequestName: "transcriptionJobId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateTranscriptionJobDetails", RequestName: "UpdateTranscriptionJobDetails", Contribution: "body"},
	}
}

func transcriptionJobDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TranscriptionJobId", RequestName: "transcriptionJobId", Contribution: "path", PreferResourceID: true},
	}
}

func (c *transcriptionJobRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *aispeechv1beta1.TranscriptionJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("TranscriptionJob generated runtime delegate is not configured")
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		return response, err
	}

	switch lifecycleState := normalizedTranscriptionJobLifecycle(resource.Status.LifecycleState); lifecycleState {
	case string(aispeech.TranscriptionJobLifecycleStateFailed):
		return c.applyLifecycleOverride(
			resource,
			transcriptionJobFailurePhase(resource),
			shared.OSOKAsyncClassFailed,
			lifecycleState,
			transcriptionJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI TranscriptionJob failed"),
		), nil
	case string(aispeech.TranscriptionJobLifecycleStateCanceling):
		return c.applyLifecycleOverride(
			resource,
			shared.OSOKAsyncPhaseDelete,
			shared.OSOKAsyncClassPending,
			lifecycleState,
			transcriptionJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI TranscriptionJob cancellation is in progress"),
		), nil
	case string(aispeech.TranscriptionJobLifecycleStateCanceled):
		return c.applyLifecycleOverride(
			resource,
			transcriptionJobFailurePhase(resource),
			shared.OSOKAsyncClassCanceled,
			lifecycleState,
			transcriptionJobLifecycleMessage(resource.Status.LifecycleDetails, "OCI TranscriptionJob was canceled"),
		), nil
	default:
		return response, nil
	}
}

func (c *transcriptionJobRuntimeClient) Delete(ctx context.Context, resource *aispeechv1beta1.TranscriptionJob) (bool, error) {
	if resource != nil && currentTranscriptionJobID(resource) == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	if c.delegate == nil {
		if c.initErr != nil {
			return false, c.initErr
		}
		return false, fmt.Errorf("TranscriptionJob generated runtime delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *transcriptionJobRuntimeClient) confirmDeleteRead(
	ctx context.Context,
	resource *aispeechv1beta1.TranscriptionJob,
	currentID string,
) (any, error) {
	if c.client == nil {
		if c.initErr != nil {
			return nil, c.initErr
		}
		return nil, fmt.Errorf("TranscriptionJob OCI client is not configured")
	}
	if strings.TrimSpace(currentID) == "" {
		currentID = currentTranscriptionJobID(resource)
	}
	if currentID == "" {
		return nil, fmt.Errorf("TranscriptionJob OCI resource identifier is not recorded")
	}
	return c.client.GetTranscriptionJob(ctx, aispeech.GetTranscriptionJobRequest{
		TranscriptionJobId: common.String(currentID),
	})
}

func (c *transcriptionJobRuntimeClient) handleDeleteError(
	resource *aispeechv1beta1.TranscriptionJob,
	err error,
) error {
	normalizedErr := normalizeTranscriptionJobOCIError(err)
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
	return normalizedErr
}

func (c *transcriptionJobRuntimeClient) applyDeleteOutcome(
	resource *aispeechv1beta1.TranscriptionJob,
	_ any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	lifecycleState := normalizedTranscriptionJobLifecycle(resource.Status.LifecycleState)

	switch stage {
	case generatedruntime.DeleteConfirmStageAlreadyPending:
		if transcriptionJobDeletePending(resource) {
			if transcriptionJobDeleteInProgress(lifecycleState) {
				c.markDeletePending(resource, lifecycleState)
				return generatedruntime.DeleteOutcome{Handled: true}, nil
			}
			c.markDeleteRetry(resource, lifecycleState)
		}
		if lifecycleState == string(aispeech.TranscriptionJobLifecycleStateCanceling) {
			c.markDeleteWaitingForCancel(resource, lifecycleState)
			return generatedruntime.DeleteOutcome{Handled: true}, nil
		}
		return generatedruntime.DeleteOutcome{}, nil
	case generatedruntime.DeleteConfirmStageAfterRequest:
		if transcriptionJobDeleteInProgress(lifecycleState) {
			c.markDeletePending(resource, lifecycleState)
			return generatedruntime.DeleteOutcome{Handled: true}, nil
		}
		c.markDeleteRetry(resource, lifecycleState)
		return generatedruntime.DeleteOutcome{Handled: true}, nil
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func (c *transcriptionJobRuntimeClient) applyLifecycleOverride(
	resource *aispeechv1beta1.TranscriptionJob,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	lifecycleState string,
	message string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	if currentID := currentTranscriptionJobID(resource); currentID != "" && status.Ocid == "" {
		status.Ocid = shared.OCID(currentID)
	}

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
		RequeueDuration: transcriptionJobRequeueDuration,
	}
}

func (c *transcriptionJobRuntimeClient) markDeletePending(
	resource *aispeechv1beta1.TranscriptionJob,
	lifecycleState string,
) {
	_ = c.applyLifecycleOverride(
		resource,
		shared.OSOKAsyncPhaseDelete,
		shared.OSOKAsyncClassPending,
		lifecycleState,
		transcriptionJobDeleteMessage(lifecycleState),
	)
}

func (c *transcriptionJobRuntimeClient) markDeleteWaitingForCancel(
	resource *aispeechv1beta1.TranscriptionJob,
	lifecycleState string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = transcriptionJobDeleteWaitMessage(lifecycleState)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.log)
}

func (c *transcriptionJobRuntimeClient) markDeleteRetry(
	resource *aispeechv1beta1.TranscriptionJob,
	lifecycleState string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = transcriptionJobDeleteRetryMessage(lifecycleState)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.log)
}

func (c *transcriptionJobRuntimeClient) markDeleted(
	resource *aispeechv1beta1.TranscriptionJob,
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
}

func transcriptionJobDeletePending(resource *aispeechv1beta1.TranscriptionJob) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}

	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func transcriptionJobDeleteInProgress(lifecycleState string) bool {
	switch normalizedTranscriptionJobLifecycle(lifecycleState) {
	case string(aispeech.TranscriptionJobLifecycleStateCanceling),
		string(aispeech.TranscriptionJobLifecycleStateCanceled):
		return true
	default:
		return false
	}
}

func currentTranscriptionJobID(resource *aispeechv1beta1.TranscriptionJob) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func transcriptionJobFailurePhase(resource *aispeechv1beta1.TranscriptionJob) shared.OSOKAsyncPhase {
	if resource != nil && resource.Status.OsokStatus.Async.Current != nil {
		phase := resource.Status.OsokStatus.Async.Current.Phase
		if phase != "" && phase != shared.OSOKAsyncPhaseDelete {
			return phase
		}
	}
	return shared.OSOKAsyncPhaseCreate
}

func normalizedTranscriptionJobLifecycle(lifecycleState string) string {
	return strings.ToUpper(strings.TrimSpace(lifecycleState))
}

func transcriptionJobLifecycleMessage(lifecycleDetails string, fallback string) string {
	if details := strings.TrimSpace(lifecycleDetails); details != "" {
		return details
	}
	return fallback
}

func transcriptionJobDeleteMessage(lifecycleState string) string {
	switch normalizedTranscriptionJobLifecycle(lifecycleState) {
	case string(aispeech.TranscriptionJobLifecycleStateCanceling):
		return "OCI TranscriptionJob delete is in progress"
	case string(aispeech.TranscriptionJobLifecycleStateCanceled):
		return "OCI TranscriptionJob delete is awaiting final not-found confirmation"
	default:
		return "OCI TranscriptionJob delete is awaiting final not-found confirmation"
	}
}

func transcriptionJobDeleteWaitMessage(lifecycleState string) string {
	if normalizedTranscriptionJobLifecycle(lifecycleState) == string(aispeech.TranscriptionJobLifecycleStateCanceling) {
		return "OCI TranscriptionJob cancellation is still in progress; waiting to issue delete"
	}
	return transcriptionJobDeleteRetryMessage(lifecycleState)
}

func transcriptionJobDeleteRetryMessage(lifecycleState string) string {
	lifecycleState = normalizedTranscriptionJobLifecycle(lifecycleState)
	if lifecycleState == "" {
		return "OCI TranscriptionJob delete has not started yet; retrying delete"
	}
	return fmt.Sprintf("OCI TranscriptionJob delete is not in progress yet; current lifecycle state is %s; retrying delete", lifecycleState)
}

func normalizeTranscriptionJobOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}
