/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
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
		return buildCreateQueueDetails(resource.Spec)
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

	runtimeClient, _ := workRequestClient.(queueOCIClient)
	appendQueueGeneratedRuntimeOverlay(manager, hooks, runtimeClient, initErr)
	appendQueueEndpointSecretWrapper(manager, hooks)
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
	desiredCapabilities, capabilitiesChanged, err := desiredQueueCapabilitiesForUpdate(
		resource.Spec.Capabilities,
		current.Capabilities,
		resource.Status.Capabilities,
	)
	if err != nil {
		return queuesdk.UpdateQueueDetails{}, false, err
	}
	if capabilitiesChanged {
		updateDetails.Capabilities = desiredCapabilities
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
	projectedCapabilities, err := projectQueueCapabilities(current.Capabilities)
	if err != nil {
		return err
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
		Capabilities:                 projectedCapabilities,
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
	capabilities := queueCapabilitySummariesToDetails(summary.Capabilities)
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
		Capabilities:     capabilities,
	}
}

func buildCreateQueueDetails(spec queuev1beta1.QueueSpec) (queuesdk.CreateQueueDetails, error) {
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
	capabilities, err := buildQueueCapabilities(spec.Capabilities)
	if err != nil {
		return queuesdk.CreateQueueDetails{}, err
	}
	if len(capabilities) > 0 {
		createDetails.Capabilities = capabilities
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}

	return createDetails, nil
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

func buildQueueCapabilities(specs []queuev1beta1.QueueCapability) ([]queuesdk.CapabilityDetails, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	capabilities := make([]queuesdk.CapabilityDetails, 0, len(specs))
	seenTypes := make(map[string]struct{}, len(specs))
	for index, spec := range specs {
		capability, capabilityType, err := buildQueueCapability(spec)
		if err != nil {
			return nil, fmt.Errorf("build capabilities[%d]: %w", index, err)
		}
		if _, exists := seenTypes[capabilityType]; exists {
			return nil, fmt.Errorf("build capabilities[%d]: duplicate capability type %q", index, capabilityType)
		}
		seenTypes[capabilityType] = struct{}{}
		capabilities = append(capabilities, capability)
	}
	return capabilities, nil
}

func buildQueueCapability(spec queuev1beta1.QueueCapability) (queuesdk.CapabilityDetails, string, error) {
	capabilityType, err := queueCapabilityTypeFromSpec(spec)
	if err != nil {
		return nil, "", err
	}

	if rawJSON := strings.TrimSpace(spec.JsonData); rawJSON != "" {
		capability, rawType, err := queueCapabilityFromJSON(rawJSON, capabilityType)
		if err != nil {
			return nil, "", err
		}
		return capability, rawType, nil
	}

	return queueCapabilityFromTypedFields(spec, capabilityType), capabilityType, nil
}

func queueCapabilityFromTypedFields(
	spec queuev1beta1.QueueCapability,
	capabilityType string,
) queuesdk.CapabilityDetails {
	payload := map[string]any{
		"type": capabilityType,
	}

	if capabilityType != string(queuesdk.QueueCapabilityConsumerGroups) {
		return payload
	}

	if spec.IsPrimaryConsumerGroupEnabled {
		payload["isPrimaryConsumerGroupEnabled"] = true
	}
	if value := spec.PrimaryConsumerGroupDisplayName; strings.TrimSpace(value) != "" {
		payload["primaryConsumerGroupDisplayName"] = value
	}
	if value := spec.PrimaryConsumerGroupFilter; strings.TrimSpace(value) != "" {
		payload["primaryConsumerGroupFilter"] = value
	}
	if spec.PrimaryConsumerGroupDeadLetterQueueDeliveryCount != 0 {
		payload["primaryConsumerGroupDeadLetterQueueDeliveryCount"] = spec.PrimaryConsumerGroupDeadLetterQueueDeliveryCount
	}

	return payload
}

func queueCapabilityFromJSON(rawJSON string, fallbackType string) (queuesdk.CapabilityDetails, string, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, "", fmt.Errorf("decode jsonData: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}

	rawType, err := queueCapabilityTypeFromJSON([]byte(rawJSON))
	if err != nil {
		return nil, "", fmt.Errorf("parse jsonData type: %w", err)
	}
	if fallbackType == "" {
		fallbackType = rawType
	}
	if fallbackType == "" {
		return nil, "", fmt.Errorf("type is required")
	}
	if rawType != "" && rawType != fallbackType {
		return nil, "", fmt.Errorf("type %q does not match jsonData type %q", fallbackType, rawType)
	}
	payload["type"] = fallbackType
	return payload, fallbackType, nil
}

func desiredQueueCapabilitiesForUpdate(
	specs []queuev1beta1.QueueCapability,
	current []queuesdk.CapabilityDetails,
	currentStatus []queuev1beta1.QueueCapability,
) ([]queuesdk.CapabilityDetails, bool, error) {
	desiredTypes, err := queueCapabilityTypeSetFromSpec(specs)
	if err != nil {
		return nil, false, err
	}
	currentTypes, err := queueCapabilityTypeSetFromSDK(current)
	if err != nil {
		return nil, false, err
	}
	if reflect.DeepEqual(currentTypes, desiredTypes) {
		desiredParity, err := queueControllerOwnedCapabilityParityFromSpec(specs)
		if err != nil {
			return nil, false, err
		}
		statusParity, err := queueControllerOwnedCapabilityParityFromStatus(currentStatus, queueCapabilityTypeMembership(desiredTypes))
		if err != nil {
			return nil, false, err
		}
		if reflect.DeepEqual(statusParity, desiredParity) {
			return nil, false, nil
		}
	}
	if len(specs) == 0 {
		return []queuesdk.CapabilityDetails{}, true, nil
	}
	desiredCapabilities, err := buildQueueCapabilities(specs)
	if err != nil {
		return nil, false, err
	}
	return desiredCapabilities, true, nil
}

func queueControllerOwnedCapabilityParityFromSpec(specs []queuev1beta1.QueueCapability) (map[string]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	snapshots := make(map[string]string)
	for index, spec := range specs {
		payloadJSON, capabilityType, err := queueCapabilityCanonicalPayloadJSON(spec)
		if err != nil {
			return nil, fmt.Errorf("build desired capabilities[%d] controller-owned parity: %w", index, err)
		}
		if !queueCapabilityRequiresControllerOwnedParity(capabilityType) {
			continue
		}
		if _, exists := snapshots[capabilityType]; exists {
			return nil, fmt.Errorf("build desired capabilities[%d] controller-owned parity: duplicate capability type %q", index, capabilityType)
		}
		snapshots[capabilityType] = payloadJSON
	}
	if len(snapshots) == 0 {
		return nil, nil
	}
	return snapshots, nil
}

func queueControllerOwnedCapabilityParityFromStatus(
	capabilities []queuev1beta1.QueueCapability,
	allowedTypes map[string]struct{},
) (map[string]string, error) {
	if len(capabilities) == 0 || len(allowedTypes) == 0 {
		return nil, nil
	}

	snapshots := make(map[string]string)
	for index, capability := range capabilities {
		payloadJSON, capabilityType, err := queueCapabilityCanonicalPayloadJSON(capability)
		if err != nil {
			return nil, fmt.Errorf("build status capabilities[%d] controller-owned parity: %w", index, err)
		}
		if !queueCapabilityRequiresControllerOwnedParity(capabilityType) {
			continue
		}
		if _, allowed := allowedTypes[capabilityType]; !allowed {
			continue
		}
		if _, exists := snapshots[capabilityType]; exists {
			return nil, fmt.Errorf("build status capabilities[%d] controller-owned parity: duplicate capability type %q", index, capabilityType)
		}
		snapshots[capabilityType] = payloadJSON
	}
	if len(snapshots) == 0 {
		return nil, nil
	}
	return snapshots, nil
}

func queueCapabilityCanonicalPayloadJSON(spec queuev1beta1.QueueCapability) (string, string, error) {
	capability, capabilityType, err := buildQueueCapability(spec)
	if err != nil {
		return "", "", err
	}

	rawJSON, err := json.Marshal(capability)
	if err != nil {
		return "", "", fmt.Errorf("marshal capability: %w", err)
	}
	return string(rawJSON), capabilityType, nil
}

func queueCapabilityTypeMembership(capabilityTypes []string) map[string]struct{} {
	if len(capabilityTypes) == 0 {
		return nil
	}

	membership := make(map[string]struct{}, len(capabilityTypes))
	for _, capabilityType := range capabilityTypes {
		membership[capabilityType] = struct{}{}
	}
	return membership
}

func rememberQueueControllerOwnedCapabilityParityStatus(resource *queuev1beta1.Queue) error {
	if !shouldRememberQueueControllerOwnedCapabilityParity(resource) {
		return nil
	}

	remembered, err := queueStatusCapabilitiesWithControllerOwnedParity(
		resource.Spec.Capabilities,
		resource.Status.Capabilities,
	)
	if err != nil {
		return err
	}
	resource.Status.Capabilities = remembered
	return nil
}

func shouldRememberQueueControllerOwnedCapabilityParity(resource *queuev1beta1.Queue) bool {
	if resource == nil {
		return false
	}
	if currentQueueAsyncPhase(resource, "") != "" {
		return false
	}
	return strings.EqualFold(
		strings.TrimSpace(resource.Status.LifecycleState),
		string(queuesdk.QueueLifecycleStateActive),
	)
}

func queueStatusCapabilitiesWithControllerOwnedParity(
	specs []queuev1beta1.QueueCapability,
	projected []queuev1beta1.QueueCapability,
) ([]queuev1beta1.QueueCapability, error) {
	if len(projected) == 0 {
		return nil, nil
	}

	controllerOwned := make(map[string]queuev1beta1.QueueCapability)
	for index, spec := range specs {
		statusCapability, capabilityType, err := queueControllerOwnedStatusCapability(spec)
		if err != nil {
			return nil, fmt.Errorf("build controller-owned status capabilities[%d]: %w", index, err)
		}
		if !queueCapabilityRequiresControllerOwnedParity(capabilityType) {
			continue
		}
		if _, exists := controllerOwned[capabilityType]; exists {
			return nil, fmt.Errorf("build controller-owned status capabilities[%d]: duplicate capability type %q", index, capabilityType)
		}
		controllerOwned[capabilityType] = statusCapability
	}
	if len(controllerOwned) == 0 {
		return projected, nil
	}

	remembered := append([]queuev1beta1.QueueCapability(nil), projected...)
	for index, capability := range remembered {
		capabilityType, err := queueCapabilityTypeFromSpec(capability)
		if err != nil {
			return nil, fmt.Errorf("read projected capabilities[%d] type: %w", index, err)
		}
		if snapshot, ok := controllerOwned[capabilityType]; ok {
			remembered[index] = snapshot
		}
	}
	return remembered, nil
}

func queueControllerOwnedStatusCapability(
	spec queuev1beta1.QueueCapability,
) (queuev1beta1.QueueCapability, string, error) {
	payloadJSON, capabilityType, err := queueCapabilityCanonicalPayloadJSON(spec)
	if err != nil {
		return queuev1beta1.QueueCapability{}, "", err
	}

	statusCapability := queuev1beta1.QueueCapability{
		Type:     capabilityType,
		JsonData: payloadJSON,
	}
	if err := json.Unmarshal([]byte(payloadJSON), &statusCapability); err != nil {
		return queuev1beta1.QueueCapability{}, "", fmt.Errorf("decode capability parity payload: %w", err)
	}
	statusCapability.Type = capabilityType
	statusCapability.JsonData = payloadJSON
	return statusCapability, capabilityType, nil
}

func queueCapabilityTypeSetFromSpec(specs []queuev1beta1.QueueCapability) ([]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	capabilityTypes := make([]string, 0, len(specs))
	seenTypes := make(map[string]struct{}, len(specs))
	for index, spec := range specs {
		capabilityType, err := queueCapabilityTypeFromSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("resolve capabilities[%d] type: %w", index, err)
		}
		if _, exists := seenTypes[capabilityType]; exists {
			return nil, fmt.Errorf("resolve capabilities[%d] type: duplicate capability type %q", index, capabilityType)
		}
		seenTypes[capabilityType] = struct{}{}
		capabilityTypes = append(capabilityTypes, capabilityType)
	}
	sort.Strings(capabilityTypes)
	return capabilityTypes, nil
}

func queueCapabilityTypeSetFromSDK(capabilities []queuesdk.CapabilityDetails) ([]string, error) {
	if len(capabilities) == 0 {
		return nil, nil
	}

	capabilityTypes := make([]string, 0, len(capabilities))
	seenTypes := make(map[string]struct{}, len(capabilities))
	for index, capability := range capabilities {
		capabilityType, err := queueCapabilityTypeFromAny(capability)
		if err != nil {
			return nil, fmt.Errorf("read capabilities[%d] type: %w", index, err)
		}
		if capabilityType == "" {
			return nil, fmt.Errorf("read capabilities[%d] type: type is empty", index)
		}
		if _, exists := seenTypes[capabilityType]; exists {
			continue
		}
		seenTypes[capabilityType] = struct{}{}
		capabilityTypes = append(capabilityTypes, capabilityType)
	}
	sort.Strings(capabilityTypes)
	return capabilityTypes, nil
}

func queueCapabilityTypeFromSpec(spec queuev1beta1.QueueCapability) (string, error) {
	capabilityType := canonicalQueueCapabilityType(spec.Type)
	rawJSON := strings.TrimSpace(spec.JsonData)
	if rawJSON == "" {
		if capabilityType == "" {
			return "", fmt.Errorf("type is required")
		}
		return capabilityType, nil
	}

	rawType, err := queueCapabilityTypeFromJSON([]byte(rawJSON))
	if err != nil {
		return "", fmt.Errorf("parse jsonData type: %w", err)
	}
	if capabilityType == "" {
		capabilityType = rawType
	} else if rawType != "" && rawType != capabilityType {
		return "", fmt.Errorf("type %q does not match jsonData type %q", spec.Type, rawType)
	}
	if capabilityType == "" {
		return "", fmt.Errorf("type is required")
	}
	return capabilityType, nil
}

func queueCapabilityTypeFromAny(capability any) (string, error) {
	rawJSON, err := json.Marshal(capability)
	if err != nil {
		return "", fmt.Errorf("marshal capability: %w", err)
	}
	return queueCapabilityTypeFromJSON(rawJSON)
}

func queueCapabilityTypeFromJSON(rawJSON []byte) (string, error) {
	var discriminator struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawJSON, &discriminator); err != nil {
		return "", err
	}
	return canonicalQueueCapabilityType(discriminator.Type), nil
}

func canonicalQueueCapabilityType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if mapped, ok := queuesdk.GetMappingQueueCapabilityEnum(value); ok {
		return string(mapped)
	}
	return strings.ToUpper(value)
}

func projectQueueCapabilities(capabilities []queuesdk.CapabilityDetails) ([]queuev1beta1.QueueCapability, error) {
	if len(capabilities) == 0 {
		return nil, nil
	}

	projected := make([]queuev1beta1.QueueCapability, 0, len(capabilities))
	seenTypes := make(map[string]struct{}, len(capabilities))
	for index, capability := range capabilities {
		projectedCapability, err := projectQueueCapability(capability)
		if err != nil {
			return nil, fmt.Errorf("project capabilities[%d]: %w", index, err)
		}
		if _, exists := seenTypes[projectedCapability.Type]; exists {
			continue
		}
		seenTypes[projectedCapability.Type] = struct{}{}
		projected = append(projected, projectedCapability)
	}
	if len(projected) == 0 {
		return nil, nil
	}
	return projected, nil
}

func projectQueueCapability(capability any) (queuev1beta1.QueueCapability, error) {
	rawJSON, err := json.Marshal(capability)
	if err != nil {
		return queuev1beta1.QueueCapability{}, fmt.Errorf("marshal capability: %w", err)
	}

	var projected queuev1beta1.QueueCapability
	if err := json.Unmarshal(rawJSON, &projected); err != nil {
		return queuev1beta1.QueueCapability{}, fmt.Errorf("decode capability: %w", err)
	}
	projected.Type = canonicalQueueCapabilityType(projected.Type)
	if projected.Type == "" {
		projected.Type, err = queueCapabilityTypeFromJSON(rawJSON)
		if err != nil {
			return queuev1beta1.QueueCapability{}, fmt.Errorf("decode capability type: %w", err)
		}
	}
	if projected.Type == "" {
		return queuev1beta1.QueueCapability{}, fmt.Errorf("type is required")
	}
	if !queueCapabilityUsesTypedProjection(projected.Type) {
		projected.JsonData = string(rawJSON)
	}
	return projected, nil
}

func queueCapabilityUsesTypedProjection(capabilityType string) bool {
	switch capabilityType {
	case string(queuesdk.QueueCapabilityConsumerGroups), string(queuesdk.QueueCapabilityLargeMessages):
		return true
	default:
		return false
	}
}

func queueCapabilityRequiresControllerOwnedParity(capabilityType string) bool {
	return capabilityType == string(queuesdk.QueueCapabilityConsumerGroups)
}

func queueCapabilitySummariesToDetails(capabilities []queuesdk.QueueCapabilityEnum) []queuesdk.CapabilityDetails {
	if len(capabilities) == 0 {
		return nil
	}

	projected := make([]queuesdk.CapabilityDetails, 0, len(capabilities))
	for _, capability := range capabilities {
		capabilityType := canonicalQueueCapabilityType(string(capability))
		if capabilityType == "" {
			continue
		}
		projected = append(projected, map[string]any{"type": capabilityType})
	}
	if len(projected) == 0 {
		return nil
	}
	return projected
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
