/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuesdk "github.com/oracle/oci-go-sdk/v65/queue"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const queueRequeueDuration = time.Minute

type queueOCIClient interface {
	CreateQueue(ctx context.Context, request queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error)
	GetQueue(ctx context.Context, request queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error)
	ListQueues(ctx context.Context, request queuesdk.ListQueuesRequest) (queuesdk.ListQueuesResponse, error)
	UpdateQueue(ctx context.Context, request queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error)
	DeleteQueue(ctx context.Context, request queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error)
	GetWorkRequest(ctx context.Context, request queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error)
}

type queueRuntimeClient struct {
	manager *QueueServiceManager
	client  queueOCIClient
	initErr error
}

type queueWorkRequestPhase string

const (
	queueWorkRequestPhaseCreate queueWorkRequestPhase = "create"
	queueWorkRequestPhaseUpdate queueWorkRequestPhase = "update"
	queueWorkRequestPhaseDelete queueWorkRequestPhase = "delete"
)

var queueServiceClientDecorator = func(_ *QueueServiceManager, delegate QueueServiceClient) QueueServiceClient {
	return delegate
}

func init() {
	newQueueServiceClient = func(manager *QueueServiceManager) QueueServiceClient {
		sdkClient, err := queuesdk.NewQueueAdminClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &queueRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize Queue OCI client: %w", err)
		}
		return queueServiceClientDecorator(manager, runtimeClient)
	}
}

func (c *queueRuntimeClient) CreateOrUpdate(ctx context.Context, resource *queuev1beta1.Queue, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	if workRequestID := strings.TrimSpace(resource.Status.CreateWorkRequestId); workRequestID != "" {
		return c.resumeCreate(ctx, resource, workRequestID)
	}

	trackedID := currentQueueID(resource)
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.getQueue(ctx, trackedID)
	if err != nil {
		if isQueueReadNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeQueueOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	if workRequestID := strings.TrimSpace(resource.Status.UpdateWorkRequestId); workRequestID != "" {
		return c.resumeUpdate(ctx, resource, current, workRequestID)
	}

	switch current.LifecycleState {
	case queuesdk.QueueLifecycleStateCreating, queuesdk.QueueLifecycleStateUpdating, queuesdk.QueueLifecycleStateDeleting, queuesdk.QueueLifecycleStateFailed:
		return c.finishWithLifecycle(resource, current), nil
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.finishWithLifecycle(resource, current), nil
	}

	response, err := c.client.UpdateQueue(ctx, updateRequest)
	if err != nil {
		return c.fail(resource, normalizeQueueOCIError(err))
	}

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return c.fail(resource, fmt.Errorf("Queue update did not return an opc-work-request-id"))
	}
	resource.Status.UpdateWorkRequestId = workRequestID
	return c.resumeUpdate(ctx, resource, current, workRequestID)
}

func (c *queueRuntimeClient) Delete(ctx context.Context, resource *queuev1beta1.Queue) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	workRequestID := strings.TrimSpace(resource.Status.DeleteWorkRequestId)
	trackedID := currentQueueID(resource)
	if workRequestID != "" {
		return c.resumeDelete(ctx, resource, trackedID, workRequestID)
	}

	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	response, err := c.client.DeleteQueue(ctx, queuesdk.DeleteQueueRequest{
		QueueId: common.String(trackedID),
	})
	if err != nil {
		if isQueueDeleteNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeQueueOCIError(err)
	}

	workRequestID = strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return false, fmt.Errorf("Queue delete did not return an opc-work-request-id")
	}
	resource.Status.DeleteWorkRequestId = workRequestID
	return c.resumeDelete(ctx, resource, trackedID, workRequestID)
}

func (c *queueRuntimeClient) create(ctx context.Context, resource *queuev1beta1.Queue) (servicemanager.OSOKResponse, error) {
	response, err := c.client.CreateQueue(ctx, queuesdk.CreateQueueRequest{
		CreateQueueDetails: buildCreateQueueDetails(resource.Spec),
	})
	if err != nil {
		return c.fail(resource, normalizeQueueOCIError(err))
	}

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return c.fail(resource, fmt.Errorf("Queue create did not return an opc-work-request-id"))
	}
	resource.Status.CreateWorkRequestId = workRequestID
	return c.resumeCreate(ctx, resource, workRequestID)
}

func (c *queueRuntimeClient) resumeCreate(ctx context.Context, resource *queuev1beta1.Queue, workRequestID string) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, normalizeQueueOCIError(err))
	}

	switch workRequest.Status {
	case queuesdk.OperationStatusAccepted, queuesdk.OperationStatusInProgress, queuesdk.OperationStatusCanceling:
		return c.markWorkRequestProgress(resource, shared.Provisioning, queueWorkRequestMessage(workRequest, queueWorkRequestPhaseCreate)), nil
	case queuesdk.OperationStatusFailed, queuesdk.OperationStatusCanceled:
		return c.fail(resource, fmt.Errorf("Queue create work request %s finished with status %s", workRequestID, workRequest.Status))
	case queuesdk.OperationStatusSucceeded:
		queueID, err := resolveQueueIDFromWorkRequest(workRequest, queuesdk.ActionTypeCreated)
		if err != nil {
			return c.fail(resource, err)
		}
		current, err := c.getQueue(ctx, queueID)
		if err != nil {
			if isQueueReadNotFoundOCI(err) {
				return c.markWorkRequestProgress(resource, shared.Provisioning, fmt.Sprintf("Queue create work request %s succeeded; waiting for Queue %s to become readable", workRequestID, queueID)), nil
			}
			return c.fail(resource, normalizeQueueOCIError(err))
		}
		resource.Status.CreateWorkRequestId = ""
		if err := c.projectStatus(resource, current); err != nil {
			return c.fail(resource, err)
		}
		return c.finishWithLifecycle(resource, current), nil
	default:
		return c.fail(resource, fmt.Errorf("Queue create work request %s returned unmodeled status %s", workRequestID, workRequest.Status))
	}
}

func (c *queueRuntimeClient) resumeUpdate(ctx context.Context, resource *queuev1beta1.Queue, current queuesdk.Queue, workRequestID string) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, normalizeQueueOCIError(err))
	}

	switch workRequest.Status {
	case queuesdk.OperationStatusAccepted, queuesdk.OperationStatusInProgress, queuesdk.OperationStatusCanceling:
		return c.markWorkRequestProgress(resource, shared.Updating, queueWorkRequestMessage(workRequest, queueWorkRequestPhaseUpdate)), nil
	case queuesdk.OperationStatusFailed, queuesdk.OperationStatusCanceled:
		return c.fail(resource, fmt.Errorf("Queue update work request %s finished with status %s", workRequestID, workRequest.Status))
	case queuesdk.OperationStatusSucceeded:
		current, err = c.getQueue(ctx, currentQueueID(resource))
		if err != nil {
			if isQueueReadNotFoundOCI(err) {
				return c.fail(resource, fmt.Errorf("Queue update work request %s succeeded but Queue %s is no longer readable", workRequestID, currentQueueID(resource)))
			}
			return c.fail(resource, normalizeQueueOCIError(err))
		}
		resource.Status.UpdateWorkRequestId = ""
		if err := c.projectStatus(resource, current); err != nil {
			return c.fail(resource, err)
		}
		return c.finishWithLifecycle(resource, current), nil
	default:
		return c.fail(resource, fmt.Errorf("Queue update work request %s returned unmodeled status %s", workRequestID, workRequest.Status))
	}
}

func (c *queueRuntimeClient) resumeDelete(ctx context.Context, resource *queuev1beta1.Queue, trackedID string, workRequestID string) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		if trackedID != "" && isQueueDeleteNotFoundOCI(err) {
			current, readErr := c.getQueue(ctx, trackedID)
			if readErr != nil {
				if isQueueDeleteNotFoundOCI(readErr) {
					c.markDeleted(resource, "OCI Queue deleted")
					return true, nil
				}
				return false, normalizeQueueOCIError(readErr)
			}
			if current.LifecycleState == queuesdk.QueueLifecycleStateDeleted {
				c.markDeleted(resource, "OCI Queue deleted")
				return true, nil
			}
			c.markDeleteProgress(resource, fmt.Sprintf("Queue delete work request %s is no longer readable; waiting for Queue %s to disappear", workRequestID, trackedID))
			return false, nil
		}
		return false, normalizeQueueOCIError(err)
	}

	switch workRequest.Status {
	case queuesdk.OperationStatusAccepted, queuesdk.OperationStatusInProgress, queuesdk.OperationStatusCanceling:
		c.markDeleteProgress(resource, queueWorkRequestMessage(workRequest, queueWorkRequestPhaseDelete))
		return false, nil
	case queuesdk.OperationStatusFailed, queuesdk.OperationStatusCanceled:
		return false, fmt.Errorf("Queue delete work request %s finished with status %s", workRequestID, workRequest.Status)
	case queuesdk.OperationStatusSucceeded:
		if trackedID == "" {
			if _, err := resolveQueueIDFromWorkRequest(workRequest, queuesdk.ActionTypeDeleted); err != nil {
				c.markDeleted(resource, "OCI Queue delete work request completed")
				return true, nil
			}
			c.markDeleted(resource, "OCI Queue delete work request completed")
			return true, nil
		}

		current, err := c.getQueue(ctx, trackedID)
		if err != nil {
			if isQueueDeleteNotFoundOCI(err) {
				c.markDeleted(resource, "OCI Queue deleted")
				return true, nil
			}
			return false, normalizeQueueOCIError(err)
		}
		if current.LifecycleState == queuesdk.QueueLifecycleStateDeleted {
			c.markDeleted(resource, "OCI Queue deleted")
			return true, nil
		}

		c.markDeleteProgress(resource, fmt.Sprintf("Queue delete work request %s succeeded; waiting for Queue %s to disappear", workRequestID, trackedID))
		return false, nil
	default:
		return false, fmt.Errorf("Queue delete work request %s returned unmodeled status %s", workRequestID, workRequest.Status)
	}
}

func (c *queueRuntimeClient) getQueue(ctx context.Context, queueID string) (queuesdk.Queue, error) {
	response, err := c.client.GetQueue(ctx, queuesdk.GetQueueRequest{
		QueueId: common.String(queueID),
	})
	if err != nil {
		return queuesdk.Queue{}, err
	}
	return response.Queue, nil
}

func (c *queueRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (queuesdk.WorkRequest, error) {
	response, err := c.client.GetWorkRequest(ctx, queuesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return queuesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c *queueRuntimeClient) buildUpdateRequest(resource *queuev1beta1.Queue, current queuesdk.Queue) (queuesdk.UpdateQueueRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return queuesdk.UpdateQueueRequest{}, false, fmt.Errorf("current Queue does not expose an OCI identifier")
	}
	if err := validateQueueCreateOnlyDrift(resource.Spec, current); err != nil {
		return queuesdk.UpdateQueueRequest{}, false, err
	}

	updateDetails := queuesdk.UpdateQueueDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if !intPtrEqual(current.VisibilityInSeconds, resource.Spec.VisibilityInSeconds) {
		updateDetails.VisibilityInSeconds = common.Int(resource.Spec.VisibilityInSeconds)
		updateNeeded = true
	}
	if !intPtrEqual(current.TimeoutInSeconds, resource.Spec.TimeoutInSeconds) {
		updateDetails.TimeoutInSeconds = common.Int(resource.Spec.TimeoutInSeconds)
		updateNeeded = true
	}
	if !intPtrEqual(current.ChannelConsumptionLimit, resource.Spec.ChannelConsumptionLimit) {
		updateDetails.ChannelConsumptionLimit = common.Int(resource.Spec.ChannelConsumptionLimit)
		updateNeeded = true
	}
	if !intPtrEqual(current.DeadLetterQueueDeliveryCount, resource.Spec.DeadLetterQueueDeliveryCount) {
		updateDetails.DeadLetterQueueDeliveryCount = common.Int(resource.Spec.DeadLetterQueueDeliveryCount)
		updateNeeded = true
	}
	if !stringPtrEqual(current.CustomEncryptionKeyId, resource.Spec.CustomEncryptionKeyId) {
		updateDetails.CustomEncryptionKeyId = common.String(resource.Spec.CustomEncryptionKeyId)
		updateNeeded = true
	}

	desiredFreeformTags := desiredQueueFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}
	desiredDefinedTags := desiredQueueDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return queuesdk.UpdateQueueRequest{}, false, nil
	}

	return queuesdk.UpdateQueueRequest{
		QueueId:            current.Id,
		UpdateQueueDetails: updateDetails,
	}, true, nil
}

func buildCreateQueueDetails(spec queuev1beta1.QueueSpec) queuesdk.CreateQueueDetails {
	createDetails := queuesdk.CreateQueueDetails{
		DisplayName:   common.String(spec.DisplayName),
		CompartmentId: common.String(spec.CompartmentId),
	}

	if spec.RetentionInSeconds != 0 {
		createDetails.RetentionInSeconds = common.Int(spec.RetentionInSeconds)
	}
	if spec.VisibilityInSeconds != 0 {
		createDetails.VisibilityInSeconds = common.Int(spec.VisibilityInSeconds)
	}
	if spec.TimeoutInSeconds != 0 {
		createDetails.TimeoutInSeconds = common.Int(spec.TimeoutInSeconds)
	}
	if spec.ChannelConsumptionLimit != 0 {
		createDetails.ChannelConsumptionLimit = common.Int(spec.ChannelConsumptionLimit)
	}
	if spec.DeadLetterQueueDeliveryCount != 0 {
		createDetails.DeadLetterQueueDeliveryCount = common.Int(spec.DeadLetterQueueDeliveryCount)
	}
	if spec.CustomEncryptionKeyId != "" {
		createDetails.CustomEncryptionKeyId = common.String(spec.CustomEncryptionKeyId)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}

	return createDetails
}

func (c *queueRuntimeClient) finishWithLifecycle(resource *queuev1beta1.Queue, current queuesdk.Queue) servicemanager.OSOKResponse {
	condition, shouldRequeue := classifyQueueLifecycle(current.LifecycleState)
	return c.markCondition(resource, condition, queueLifecycleMessage(current), shouldRequeue)
}

func (c *queueRuntimeClient) markCondition(resource *queuev1beta1.Queue, condition shared.OSOKConditionType, message string, shouldRequeue bool) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.manager.Log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: queueRequeueDuration,
	}
}

func (c *queueRuntimeClient) markWorkRequestProgress(resource *queuev1beta1.Queue, condition shared.OSOKConditionType, message string) servicemanager.OSOKResponse {
	return c.markCondition(resource, condition, message, true)
}

func (c *queueRuntimeClient) markDeleteProgress(resource *queuev1beta1.Queue, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *queueRuntimeClient) fail(resource *queuev1beta1.Queue, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *queueRuntimeClient) markDeleted(resource *queuev1beta1.Queue, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.CreateWorkRequestId = ""
	resource.Status.UpdateWorkRequestId = ""
	resource.Status.DeleteWorkRequestId = ""
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *queueRuntimeClient) clearTrackedIdentity(resource *queuev1beta1.Queue) {
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func (c *queueRuntimeClient) projectStatus(resource *queuev1beta1.Queue, current queuesdk.Queue) error {
	resource.Status = queuev1beta1.QueueStatus{
		OsokStatus:                   resource.Status.OsokStatus,
		Id:                           stringValue(current.Id),
		CompartmentId:                stringValue(current.CompartmentId),
		TimeCreated:                  sdkTimeString(current.TimeCreated),
		TimeUpdated:                  sdkTimeString(current.TimeUpdated),
		LifecycleState:               string(current.LifecycleState),
		MessagesEndpoint:             stringValue(current.MessagesEndpoint),
		RetentionInSeconds:           intValue(current.RetentionInSeconds),
		VisibilityInSeconds:          intValue(current.VisibilityInSeconds),
		TimeoutInSeconds:             intValue(current.TimeoutInSeconds),
		DeadLetterQueueDeliveryCount: intValue(current.DeadLetterQueueDeliveryCount),
		DisplayName:                  stringValue(current.DisplayName),
		LifecycleDetails:             stringValue(current.LifecycleDetails),
		CustomEncryptionKeyId:        stringValue(current.CustomEncryptionKeyId),
		FreeformTags:                 cloneStringMap(current.FreeformTags),
		DefinedTags:                  convertOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:                   convertOCIToStatusDefinedTags(current.SystemTags),
		ChannelConsumptionLimit:      intValue(current.ChannelConsumptionLimit),
		CreateWorkRequestId:          resource.Status.CreateWorkRequestId,
		UpdateWorkRequestId:          resource.Status.UpdateWorkRequestId,
		DeleteWorkRequestId:          resource.Status.DeleteWorkRequestId,
	}
	return nil
}

func validateQueueCreateOnlyDrift(spec queuev1beta1.QueueSpec, current queuesdk.Queue) error {
	var unsupported []string

	if !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if spec.RetentionInSeconds != 0 && !intPtrEqual(current.RetentionInSeconds, spec.RetentionInSeconds) {
		unsupported = append(unsupported, "retentionInSeconds")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("Queue create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func resolveQueueIDFromWorkRequest(workRequest queuesdk.WorkRequest, action queuesdk.ActionTypeEnum) (string, error) {
	for _, resource := range workRequest.Resources {
		if !strings.EqualFold(strings.TrimSpace(stringValue(resource.EntityType)), "queue") {
			continue
		}
		if action != "" && resource.ActionType != action {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}

	for _, resource := range workRequest.Resources {
		if !strings.EqualFold(strings.TrimSpace(stringValue(resource.EntityType)), "queue") {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("Queue work request %s does not expose a Queue identifier", stringValue(workRequest.Id))
}

func queueWorkRequestMessage(workRequest queuesdk.WorkRequest, phase queueWorkRequestPhase) string {
	return fmt.Sprintf("Queue %s work request %s is %s", phase, stringValue(workRequest.Id), workRequest.Status)
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

func isQueueReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isQueueDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
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

func desiredQueueFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredQueueDefinedTagsForUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func intPtrEqual(actual *int, expected int) bool {
	if actual == nil {
		return expected == 0
	}
	return *actual == expected
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(input) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if len(values) == 0 {
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	if len(converted) == 0 {
		return nil
	}
	return converted
}
