/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package transcriptionjob

import (
	"context"
	"encoding/json"
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

// Speech jobs need a narrow handwritten layer because CANCELING/CANCELED have
// different meanings during normal status observation and delete confirmation.
// Generatedruntime still owns the CRUD request projection, lifecycle rereads,
// and update-drift enforcement for the steady-state create/update path.
type transcriptionJobRuntimeClient struct {
	generated generatedruntime.ServiceClient[*aispeechv1beta1.TranscriptionJob]
	client    transcriptionJobOCIClient
	log       loggerutil.OSOKLogger
	initErr   error
}

func init() {
	newTranscriptionJobServiceClient = func(manager *TranscriptionJobServiceManager) TranscriptionJobServiceClient {
		sdkClient, err := aispeech.NewAIServiceSpeechClientWithConfigurationProvider(manager.Provider)
		config := newTranscriptionJobRuntimeConfig(manager.Log, sdkClient)
		initErr := err
		if err != nil {
			config.InitError = fmt.Errorf("initialize TranscriptionJob OCI client: %w", err)
		}
		return &transcriptionJobRuntimeClient{
			generated: generatedruntime.NewServiceClient[*aispeechv1beta1.TranscriptionJob](config),
			client:    sdkClient,
			log:       manager.Log,
			initErr:   initErr,
		}
	}
}

func newTranscriptionJobServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client transcriptionJobOCIClient,
) TranscriptionJobServiceClient {
	return &transcriptionJobRuntimeClient{
		generated: generatedruntime.NewServiceClient[*aispeechv1beta1.TranscriptionJob](
			newTranscriptionJobRuntimeConfig(log, client),
		),
		client: client,
		log:    log,
	}
}

func newTranscriptionJobRuntimeConfig(
	log loggerutil.OSOKLogger,
	sdkClient transcriptionJobOCIClient,
) generatedruntime.Config[*aispeechv1beta1.TranscriptionJob] {
	return generatedruntime.Config[*aispeechv1beta1.TranscriptionJob]{
		Kind:      "TranscriptionJob",
		SDKName:   "TranscriptionJob",
		Log:       log,
		Semantics: transcriptionJobRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &aispeech.CreateTranscriptionJobRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.CreateTranscriptionJob(ctx, *request.(*aispeech.CreateTranscriptionJobRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateTranscriptionJobDetails", RequestName: "CreateTranscriptionJobDetails", Contribution: "body"},
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &aispeech.GetTranscriptionJobRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.GetTranscriptionJob(ctx, *request.(*aispeech.GetTranscriptionJobRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "TranscriptionJobId", RequestName: "transcriptionJobId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &aispeech.ListTranscriptionJobsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.ListTranscriptionJobs(ctx, *request.(*aispeech.ListTranscriptionJobsRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
				{FieldName: "Id", RequestName: "id", Contribution: "query", LookupPaths: []string{"status.id", "status.ocid", "id", "ocid"}},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &aispeech.UpdateTranscriptionJobRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.UpdateTranscriptionJob(ctx, *request.(*aispeech.UpdateTranscriptionJobRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "TranscriptionJobId", RequestName: "transcriptionJobId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateTranscriptionJobDetails", RequestName: "UpdateTranscriptionJobDetails", Contribution: "body"},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &aispeech.DeleteTranscriptionJobRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.DeleteTranscriptionJob(ctx, *request.(*aispeech.DeleteTranscriptionJobRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "TranscriptionJobId", RequestName: "transcriptionJobId", Contribution: "path", PreferResourceID: true},
			},
		},
	}
}

func transcriptionJobRuntimeSemantics() *generatedruntime.Semantics {
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
		// Delete confirmation stays in the handwritten layer so normal observe
		// does not misclassify CANCELED as a completed delete.
		Delete: generatedruntime.DeleteSemantics{
			Policy: "not-supported",
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
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func (c *transcriptionJobRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *aispeechv1beta1.TranscriptionJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.generated.CreateOrUpdate(ctx, resource, req)
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
	if c.initErr != nil {
		return false, c.initErr
	}

	currentID := currentTranscriptionJobID(resource)
	if currentID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	current, err := c.getTranscriptionJob(ctx, currentID)
	if err != nil {
		if isTranscriptionJobDeleteNotFoundOCI(err) {
			normalizedErr := normalizeTranscriptionJobOCIError(err)
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}

		normalizedErr := normalizeTranscriptionJobOCIError(err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
		return false, normalizedErr
	}
	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}

	lifecycleState := normalizedTranscriptionJobLifecycle(resource.Status.LifecycleState)
	if transcriptionJobDeleteInProgress(lifecycleState) {
		c.markDeletePending(resource, lifecycleState)
		return false, nil
	}
	if transcriptionJobDeletePending(resource) {
		c.markDeleteRetry(resource, lifecycleState)
	}

	response, err := c.client.DeleteTranscriptionJob(ctx, aispeech.DeleteTranscriptionJobRequest{
		TranscriptionJobId: common.String(currentID),
	})
	if err != nil {
		if isTranscriptionJobDeleteNotFoundOCI(err) {
			normalizedErr := normalizeTranscriptionJobOCIError(err)
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}

		retryableConflict := isRetryableTranscriptionJobDeleteConflict(err)
		normalizedErr := normalizeTranscriptionJobOCIError(err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
		if !retryableConflict {
			return false, normalizedErr
		}
	} else {
		servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	}

	return c.confirmDelete(ctx, resource, currentID)
}

func (c *transcriptionJobRuntimeClient) confirmDelete(
	ctx context.Context,
	resource *aispeechv1beta1.TranscriptionJob,
	currentID string,
) (bool, error) {
	current, err := c.getTranscriptionJob(ctx, currentID)
	if err != nil {
		if isTranscriptionJobDeleteNotFoundOCI(err) {
			normalizedErr := normalizeTranscriptionJobOCIError(err)
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}

		normalizedErr := normalizeTranscriptionJobOCIError(err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalizedErr)
		return false, normalizedErr
	}
	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}

	lifecycleState := normalizedTranscriptionJobLifecycle(resource.Status.LifecycleState)
	if transcriptionJobDeleteInProgress(lifecycleState) {
		c.markDeletePending(resource, lifecycleState)
		return false, nil
	}

	c.markDeleteRetry(resource, lifecycleState)
	return false, nil
}

func (c *transcriptionJobRuntimeClient) getTranscriptionJob(
	ctx context.Context,
	currentID string,
) (aispeech.TranscriptionJob, error) {
	response, err := c.client.GetTranscriptionJob(ctx, aispeech.GetTranscriptionJobRequest{
		TranscriptionJobId: common.String(currentID),
	})
	if err != nil {
		return aispeech.TranscriptionJob{}, err
	}
	return response.TranscriptionJob, nil
}

func (c *transcriptionJobRuntimeClient) projectStatus(
	resource *aispeechv1beta1.TranscriptionJob,
	current aispeech.TranscriptionJob,
) error {
	payload, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("marshal TranscriptionJob response body: %w", err)
	}
	if err := json.Unmarshal(payload, &resource.Status); err != nil {
		return fmt.Errorf("project TranscriptionJob response body into status: %w", err)
	}
	if id := strings.TrimSpace(stringValue(current.Id)); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	return nil
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

func isTranscriptionJobDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func isRetryableTranscriptionJobDeleteConflict(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsConflict()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
