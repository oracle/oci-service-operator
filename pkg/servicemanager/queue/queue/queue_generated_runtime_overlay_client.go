/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuesdk "github.com/oracle/oci-go-sdk/v65/queue"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type queueGeneratedRuntimeOverlayClient struct {
	manager  *QueueServiceManager
	delegate QueueServiceClient
	client   queueOCIClient
	initErr  error
}

var _ QueueServiceClient = queueGeneratedRuntimeOverlayClient{}

func appendQueueGeneratedRuntimeOverlay(
	manager *QueueServiceManager,
	hooks *QueueRuntimeHooks,
	client queueOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate QueueServiceClient) QueueServiceClient {
		return newQueueGeneratedRuntimeOverlayClient(manager, delegate, client, initErr)
	})
}

func appendQueueEndpointSecretWrapper(manager *QueueServiceManager, hooks *QueueRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate QueueServiceClient) QueueServiceClient {
		return newQueueEndpointSecretClient(manager, delegate)
	})
}

func newQueueGeneratedRuntimeOverlayClient(
	manager *QueueServiceManager,
	delegate QueueServiceClient,
	client queueOCIClient,
	initErr error,
) QueueServiceClient {
	overlay := queueGeneratedRuntimeOverlayClient{
		manager:  manager,
		delegate: delegate,
		client:   client,
		initErr:  initErr,
	}
	if overlay.client != nil || manager == nil {
		return overlay
	}

	sdkClient, err := queuesdk.NewQueueAdminClientWithConfigurationProvider(manager.Provider)
	overlay.client = sdkClient
	if err != nil && overlay.initErr == nil {
		overlay.initErr = fmt.Errorf("initialize Queue OCI client: %w", err)
	}
	return overlay
}

func (c queueGeneratedRuntimeOverlayClient) CreateOrUpdate(
	ctx context.Context,
	resource *queuev1beta1.Queue,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Queue generated runtime delegate is not configured")
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || resource == nil {
		return response, err
	}
	if !queueObservedLifecycleFailed(resource) {
		return response, nil
	}
	return queueFinishWithLifecycle(resource, queueFromStatus(resource), "", c.log()), nil
}

func (c queueGeneratedRuntimeOverlayClient) Delete(ctx context.Context, resource *queuev1beta1.Queue) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("Queue generated runtime delegate is not configured")
	}

	deleted, err := c.delegate.Delete(ctx, resource)
	if err == nil || deleted || resource == nil {
		return deleted, err
	}

	trackedID := currentQueueID(resource)
	workRequestID := strings.TrimSpace(resource.Status.DeleteWorkRequestId)
	if trackedID == "" ||
		workRequestID == "" ||
		c.initErr != nil ||
		c.client == nil ||
		(!isQueueDeleteNotFoundOCI(err) && !isQueueDeleteConfirmUnexpectedLifecycle(err)) {
		return deleted, err
	}

	current, readErr := c.getQueue(ctx, trackedID)
	if readErr != nil {
		if isQueueDeleteNotFoundOCI(readErr) {
			queueRecordErrorRequestID(resource, readErr)
			queueMarkDeleted(resource, "OCI Queue deleted", c.log())
			return true, nil
		}
		normalized := normalizeQueueOCIError(readErr)
		queueRecordErrorRequestID(resource, normalized)
		return false, normalized
	}
	if current.LifecycleState == queuesdk.QueueLifecycleStateDeleted {
		queueMarkDeleted(resource, "OCI Queue deleted", c.log())
		return true, nil
	}
	if err := projectQueueStatus(resource, current); err != nil {
		return false, err
	}

	queueMarkDeleteProgress(
		resource,
		queueDeleteProgressMessage(workRequestID, trackedID, err),
		c.log(),
	)
	return false, nil
}

func (c queueGeneratedRuntimeOverlayClient) getQueue(ctx context.Context, queueID string) (queuesdk.Queue, error) {
	if c.initErr != nil {
		return queuesdk.Queue{}, c.initErr
	}
	if c.client == nil {
		return queuesdk.Queue{}, fmt.Errorf("Queue OCI client is not configured")
	}

	response, err := c.client.GetQueue(ctx, queuesdk.GetQueueRequest{
		QueueId: common.String(queueID),
	})
	if err != nil {
		return queuesdk.Queue{}, err
	}
	return response.Queue, nil
}

func (c queueGeneratedRuntimeOverlayClient) log() loggerutil.OSOKLogger {
	if c.manager != nil {
		return c.manager.Log
	}
	return loggerutil.OSOKLogger{}
}

func queueObservedLifecycleFailed(resource *queuev1beta1.Queue) bool {
	if resource == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(resource.Status.LifecycleState), string(queuesdk.QueueLifecycleStateFailed))
}

func queueFromStatus(resource *queuev1beta1.Queue) queuesdk.Queue {
	if resource == nil {
		return queuesdk.Queue{}
	}

	var queueID *string
	if currentID := strings.TrimSpace(currentQueueID(resource)); currentID != "" {
		queueID = common.String(currentID)
	}

	var displayName *string
	if name := strings.TrimSpace(resource.Status.DisplayName); name != "" {
		displayName = common.String(name)
	}

	return queuesdk.Queue{
		Id:             queueID,
		DisplayName:    displayName,
		LifecycleState: queuesdk.QueueLifecycleStateEnum(strings.TrimSpace(resource.Status.LifecycleState)),
	}
}

func queueFinishWithLifecycle(
	resource *queuev1beta1.Queue,
	current queuesdk.Queue,
	explicitPhase shared.OSOKAsyncPhase,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	condition, shouldRequeue := classifyQueueLifecycle(current.LifecycleState)
	message := queueLifecycleMessage(current)
	if asyncCurrent := queueLifecycleAsyncOperation(resource, current, message, explicitPhase); asyncCurrent != nil {
		return queueMarkAsyncOperation(resource, asyncCurrent, log)
	}
	return queueMarkCondition(resource, condition, message, shouldRequeue, log)
}

func queueMarkCondition(
	resource *queuev1beta1.Queue,
	condition shared.OSOKConditionType,
	message string,
	shouldRequeue bool,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if currentID := currentQueueID(resource); currentID != "" {
		status.Ocid = shared.OCID(currentID)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active || (condition == shared.Failed && currentQueueAsyncPhase(resource, "") == "") {
		servicemanager.ClearAsyncOperation(status)
	}
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: queueRequeueDuration,
	}
}

func queueMarkAsyncOperation(
	resource *queuev1beta1.Queue,
	current *shared.OSOKAsyncOperation,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if currentID := currentQueueID(resource); currentID != "" {
		status.Ocid = shared.OCID(currentID)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: queueRequeueDuration,
	}
}

func queueMarkDeleteProgress(resource *queuev1beta1.Queue, message string, log loggerutil.OSOKLogger) {
	_ = queueMarkAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           currentQueueAsyncPhase(resource, shared.OSOKAsyncPhaseDelete),
		WorkRequestID:   strings.TrimSpace(resource.Status.DeleteWorkRequestId),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
	}, log)
}

func queueRecordErrorRequestID(resource *queuev1beta1.Queue, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func queueMarkDeleted(resource *queuev1beta1.Queue, message string, log loggerutil.OSOKLogger) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.CreateWorkRequestId = ""
	resource.Status.UpdateWorkRequestId = ""
	resource.Status.DeleteWorkRequestId = ""
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, log)
}

func queueLifecycleMessage(current queuesdk.Queue) string {
	name := strings.TrimSpace(stringValue(current.DisplayName))
	if name == "" {
		name = strings.TrimSpace(stringValue(current.Id))
	}
	if name == "" {
		name = "Queue"
	}
	return fmt.Sprintf("Queue %s is %s", name, current.LifecycleState)
}

func classifyQueueLifecycle(state queuesdk.QueueLifecycleStateEnum) (shared.OSOKConditionType, bool) {
	switch state {
	case queuesdk.QueueLifecycleStateCreating:
		return shared.Provisioning, true
	case queuesdk.QueueLifecycleStateUpdating:
		return shared.Updating, true
	case queuesdk.QueueLifecycleStateDeleting:
		return shared.Terminating, true
	case queuesdk.QueueLifecycleStateFailed:
		return shared.Failed, false
	default:
		return shared.Active, false
	}
}

func queueLifecycleAsyncOperation(
	resource *queuev1beta1.Queue,
	current queuesdk.Queue,
	message string,
	explicitPhase shared.OSOKAsyncPhase,
) *shared.OSOKAsyncOperation {
	switch current.LifecycleState {
	case queuesdk.QueueLifecycleStateCreating:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseCreate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case queuesdk.QueueLifecycleStateUpdating:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case queuesdk.QueueLifecycleStateDeleting:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case queuesdk.QueueLifecycleStateFailed:
		phase := currentQueueAsyncPhase(resource, explicitPhase)
		if phase == "" {
			return nil
		}
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
		}
	default:
		return nil
	}
}

func normalizeQueueOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isQueueDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func isQueueDeleteConfirmUnexpectedLifecycle(err error) bool {
	return err != nil && strings.Contains(err.Error(), "delete confirmation returned unexpected lifecycle state")
}

func queueDeleteProgressMessage(workRequestID, queueID string, cause error) string {
	if isQueueDeleteNotFoundOCI(cause) {
		return fmt.Sprintf(
			"Queue delete work request %s is no longer readable; waiting for Queue %s to disappear",
			workRequestID,
			queueID,
		)
	}
	return fmt.Sprintf(
		"Queue delete work request %s succeeded; waiting for Queue %s to disappear",
		workRequestID,
		queueID,
	)
}

func currentQueueID(resource *queuev1beta1.Queue) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}
