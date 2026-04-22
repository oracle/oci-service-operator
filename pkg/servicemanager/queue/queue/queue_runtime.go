/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuesdk "github.com/oracle/oci-go-sdk/v65/queue"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

const queueRequeueDuration = time.Minute

var queueWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(queuesdk.OperationStatusAccepted),
		string(queuesdk.OperationStatusInProgress),
		string(queuesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(queuesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(queuesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(queuesdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(queuesdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(queuesdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(queuesdk.ActionTypeDeleted)},
}

type queueOCIClient interface {
	CreateQueue(ctx context.Context, request queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error)
	GetQueue(ctx context.Context, request queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error)
	ListQueues(ctx context.Context, request queuesdk.ListQueuesRequest) (queuesdk.ListQueuesResponse, error)
	UpdateQueue(ctx context.Context, request queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error)
	DeleteQueue(ctx context.Context, request queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error)
	GetWorkRequest(ctx context.Context, request queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error)
}

type queueWorkRequestClient interface {
	GetWorkRequest(ctx context.Context, request queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error)
}

func init() {
	registerQueueRuntimeHooksMutator(func(manager *QueueServiceManager, hooks *QueueRuntimeHooks) {
		workRequestClient, initErr := newQueueWorkRequestClient(manager)
		applyQueueRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newQueueWorkRequestClient(manager *QueueServiceManager) (queueWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Queue service manager is nil")
	}
	client, err := queuesdk.NewQueueAdminClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyQueueRuntimeHooks(
	manager *QueueServiceManager,
	hooks *QueueRuntimeHooks,
	workRequestClient queueWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.BuildCreateBody = func(_ context.Context, resource *queuev1beta1.Queue, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("Queue resource is nil")
		}
		return buildCreateQueueDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *queuev1beta1.Queue,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildQueueUpdateBody(resource, currentResponse)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedQueueIdentity
	hooks.StatusHooks.ProjectStatus = projectQueueStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateQueueCreateOnlyDriftForResponse
	hooks.Async.Adapter = queueWorkRequestAsyncAdapter
	hooks.Async.ResolveAction = resolveQueueGeneratedWorkRequestAction
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getQueueWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolvePhase = resolveQueueGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverQueueIDFromGeneratedWorkRequest
	hooks.Async.Message = queueGeneratedWorkRequestMessage
	if manager != nil {
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate QueueServiceClient) QueueServiceClient {
			return newQueueEndpointSecretClient(manager, delegate)
		})
	}
}

func buildQueueUpdateBody(
	resource *queuev1beta1.Queue,
	currentResponse any,
) (queuesdk.UpdateQueueDetails, bool, error) {
	if resource == nil {
		return queuesdk.UpdateQueueDetails{}, false, fmt.Errorf("Queue resource is nil")
	}

	current, ok := queueFromResponse(currentResponse)
	if !ok {
		return queuesdk.UpdateQueueDetails{}, false, fmt.Errorf("current Queue response does not expose a Queue body")
	}
	if err := validateQueueCreateOnlyDrift(resource.Spec, current); err != nil {
		return queuesdk.UpdateQueueDetails{}, false, err
	}

	// Queue updates must preserve explicit zero and empty-string intent. The
	// generic generatedruntime mutation filter still treats those values as
	// absent, so Queue keeps a local typed request builder here.
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
		return queuesdk.UpdateQueueDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func projectQueueStatus(resource *queuev1beta1.Queue, response any) error {
	if resource == nil {
		return fmt.Errorf("Queue resource is nil")
	}

	current, ok := queueFromResponse(response)
	if !ok {
		return nil
	}

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

func validateQueueCreateOnlyDriftForResponse(resource *queuev1beta1.Queue, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("Queue resource is nil")
	}

	current, ok := queueFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current Queue response does not expose a Queue body")
	}
	return validateQueueCreateOnlyDrift(resource.Spec, current)
}

func clearTrackedQueueIdentity(resource *queuev1beta1.Queue) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func getQueueWorkRequest(
	ctx context.Context,
	client queueWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Queue OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Queue OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, queuesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveQueueGeneratedWorkRequestAction(workRequest any) (string, error) {
	queueWorkRequest, err := queueWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveQueueWorkRequestAction(queueWorkRequest)
}

func resolveQueueGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	queueWorkRequest, err := queueWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := queueWorkRequestPhaseFromOperationType(queueWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverQueueIDFromGeneratedWorkRequest(
	_ *queuev1beta1.Queue,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	queueWorkRequest, err := queueWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveQueueIDFromWorkRequest(queueWorkRequest, queueWorkRequestActionForPhase(phase))
}

func queueGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	queueWorkRequest, err := queueWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return queueWorkRequestMessage(phase, queueWorkRequest)
}

func queueWorkRequestFromAny(workRequest any) (queuesdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case queuesdk.WorkRequest:
		return current, nil
	case *queuesdk.WorkRequest:
		if current == nil {
			return queuesdk.WorkRequest{}, fmt.Errorf("Queue work request is nil")
		}
		return *current, nil
	default:
		return queuesdk.WorkRequest{}, fmt.Errorf("unexpected Queue work request type %T", workRequest)
	}
}

func queueWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) queuesdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return queuesdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return queuesdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return queuesdk.ActionTypeDeleted
	default:
		return ""
	}
}

func queueWorkRequestPhaseFromOperationType(operationType queuesdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case queuesdk.OperationTypeCreateQueue:
		return shared.OSOKAsyncPhaseCreate, true
	case queuesdk.OperationTypeUpdateQueue:
		return shared.OSOKAsyncPhaseUpdate, true
	case queuesdk.OperationTypeDeleteQueue:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func queueFromResponse(response any) (queuesdk.Queue, bool) {
	switch current := response.(type) {
	case queuesdk.CreateQueueResponse:
		return queuesdk.Queue{}, false
	case *queuesdk.CreateQueueResponse:
		return queuesdk.Queue{}, false
	case queuesdk.GetQueueResponse:
		return current.Queue, true
	case *queuesdk.GetQueueResponse:
		if current == nil {
			return queuesdk.Queue{}, false
		}
		return current.Queue, true
	case queuesdk.Queue:
		return current, true
	case *queuesdk.Queue:
		if current == nil {
			return queuesdk.Queue{}, false
		}
		return *current, true
	case queuesdk.QueueSummary:
		return queueFromSummary(current), true
	case *queuesdk.QueueSummary:
		if current == nil {
			return queuesdk.Queue{}, false
		}
		return queueFromSummary(*current), true
	default:
		return queuesdk.Queue{}, false
	}
}

func queueFromSummary(summary queuesdk.QueueSummary) queuesdk.Queue {
	return queuesdk.Queue{
		Id:               summary.Id,
		CompartmentId:    summary.CompartmentId,
		TimeCreated:      summary.TimeCreated,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleState:   summary.LifecycleState,
		MessagesEndpoint: summary.MessagesEndpoint,
		DisplayName:      summary.DisplayName,
		LifecycleDetails: summary.LifecycleDetails,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		SystemTags:       summary.SystemTags,
	}
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
		if !isQueueWorkRequestResource(resource) {
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
		if !isQueueWorkRequestResource(resource) {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("Queue work request %s does not expose a Queue identifier", stringValue(workRequest.Id))
}

func resolveQueueWorkRequestAction(workRequest queuesdk.WorkRequest) (string, error) {
	var action string

	for _, resource := range workRequest.Resources {
		if !isQueueWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("Queue work request %s exposes conflicting Queue action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}

	return action, nil
}

func queueWorkRequestAsyncOperation(
	resource *queuev1beta1.Queue,
	workRequest queuesdk.WorkRequest,
	explicitPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}

	rawAction, err := resolveQueueWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}

	fallbackPhase := currentQueueAsyncPhase(resource, explicitPhase)
	if derivedPhase, ok := queueWorkRequestPhaseFromOperationType(workRequest.OperationType); ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf(
				"Queue work request %s exposes operation type %q for phase %q while reconcile expected phase %q",
				stringValue(workRequest.Id),
				workRequest.OperationType,
				derivedPhase,
				fallbackPhase,
			)
		}
		fallbackPhase = derivedPhase
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, queueWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        rawAction,
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    stringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}

	current.Message = queueWorkRequestMessage(current.Phase, workRequest)
	return current, nil
}

func queueWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest queuesdk.WorkRequest) string {
	return fmt.Sprintf("Queue %s work request %s is %s", phase, stringValue(workRequest.Id), workRequest.Status)
}

func isQueueWorkRequestResource(resource queuesdk.WorkRequestResource) bool {
	return strings.EqualFold(strings.TrimSpace(stringValue(resource.EntityType)), "queue")
}

func currentQueueAsyncPhase(resource *queuev1beta1.Queue, fallback shared.OSOKAsyncPhase) shared.OSOKAsyncPhase {
	if resource == nil {
		return fallback
	}
	if fallback != "" {
		return fallback
	}
	if legacy := queueLegacyWorkRequestPhase(resource); legacy != "" {
		return legacy
	}
	return servicemanager.ResolveAsyncPhase(&resource.Status.OsokStatus, "")
}

func queueLegacyWorkRequestPhase(resource *queuev1beta1.Queue) shared.OSOKAsyncPhase {
	if resource == nil {
		return ""
	}
	switch {
	case strings.TrimSpace(resource.Status.DeleteWorkRequestId) != "":
		return shared.OSOKAsyncPhaseDelete
	case strings.TrimSpace(resource.Status.UpdateWorkRequestId) != "":
		return shared.OSOKAsyncPhaseUpdate
	case strings.TrimSpace(resource.Status.CreateWorkRequestId) != "":
		return shared.OSOKAsyncPhaseCreate
	default:
		return ""
	}
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
