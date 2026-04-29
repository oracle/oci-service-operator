/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loggroup

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var logGroupWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(loggingsdk.OperationStatusAccepted),
		string(loggingsdk.OperationStatusInProgress),
		string(loggingsdk.OperationStatusCancelling),
	},
	SucceededStatusTokens: []string{string(loggingsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(loggingsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(loggingsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(loggingsdk.OperationTypesCreateLogGroup)},
	UpdateActionTokens:    []string{string(loggingsdk.OperationTypesUpdateLogGroup)},
	DeleteActionTokens:    []string{string(loggingsdk.OperationTypesDeleteLogGroup)},
}

type logGroupWorkRequestClient interface {
	GetWorkRequest(context.Context, loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error)
}

func init() {
	registerLogGroupRuntimeHooksMutator(func(manager *LogGroupServiceManager, hooks *LogGroupRuntimeHooks) {
		workRequestClient, initErr := newLogGroupWorkRequestClient(manager)
		applyLogGroupRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newLogGroupWorkRequestClient(manager *LogGroupServiceManager) (logGroupWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("LogGroup service manager is nil")
	}
	client, err := loggingsdk.NewLoggingManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyLogGroupRuntimeHooks(
	hooks *LogGroupRuntimeHooks,
	workRequestClient logGroupWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newLogGroupRuntimeSemantics()
	hooks.Async.Adapter = logGroupWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getLogGroupWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveLogGroupGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveLogGroupGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverLogGroupIDFromGeneratedWorkRequest
	hooks.Async.Message = logGroupGeneratedWorkRequestMessage
}

func newLogGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "logging",
		FormalSlug:    "loggroup",
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
			ProvisioningStates: []string{string(loggingsdk.LogGroupLifecycleStateCreating)},
			UpdatingStates:     []string{string(loggingsdk.LogGroupLifecycleStateUpdating)},
			ActiveStates:       []string{string(loggingsdk.LogGroupLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(loggingsdk.LogGroupLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
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

func getLogGroupWorkRequest(
	ctx context.Context,
	client logGroupWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize LogGroup OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("LogGroup OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, loggingsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveLogGroupGeneratedWorkRequestAction(workRequest any) (string, error) {
	logGroupWorkRequest, err := logGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(logGroupWorkRequest.OperationType), nil
}

func resolveLogGroupGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	logGroupWorkRequest, err := logGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := logGroupWorkRequestPhaseFromOperationType(logGroupWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverLogGroupIDFromGeneratedWorkRequest(
	_ *loggingv1beta1.LogGroup,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	logGroupWorkRequest, err := logGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveLogGroupIDFromWorkRequest(logGroupWorkRequest, logGroupWorkRequestActionForPhase(phase))
}

func logGroupWorkRequestFromAny(workRequest any) (loggingsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case loggingsdk.WorkRequest:
		return current, nil
	case *loggingsdk.WorkRequest:
		if current == nil {
			return loggingsdk.WorkRequest{}, fmt.Errorf("LogGroup work request is nil")
		}
		return *current, nil
	default:
		return loggingsdk.WorkRequest{}, fmt.Errorf("unexpected LogGroup work request type %T", workRequest)
	}
}

func logGroupWorkRequestPhaseFromOperationType(operationType loggingsdk.OperationTypesEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case loggingsdk.OperationTypesCreateLogGroup:
		return shared.OSOKAsyncPhaseCreate, true
	case loggingsdk.OperationTypesUpdateLogGroup:
		return shared.OSOKAsyncPhaseUpdate, true
	case loggingsdk.OperationTypesDeleteLogGroup:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func logGroupWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) loggingsdk.ActionTypesEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return loggingsdk.ActionTypesCreated
	case shared.OSOKAsyncPhaseUpdate:
		return loggingsdk.ActionTypesUpdated
	case shared.OSOKAsyncPhaseDelete:
		return loggingsdk.ActionTypesDeleted
	default:
		return ""
	}
}

func resolveLogGroupIDFromWorkRequest(workRequest loggingsdk.WorkRequest, action loggingsdk.ActionTypesEnum) (string, error) {
	if id, ok := resolveLogGroupIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveLogGroupIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("LogGroup work request %s does not expose a log group identifier", stringValue(workRequest.Id))
}

func resolveLogGroupIDFromResources(
	resources []loggingsdk.WorkRequestResource,
	action loggingsdk.ActionTypesEnum,
	preferLogGroupOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferLogGroupOnly && !isLogGroupWorkRequestResource(resource) {
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

func logGroupGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	logGroupWorkRequest, err := logGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("LogGroup %s work request %s is %s", phase, stringValue(logGroupWorkRequest.Id), logGroupWorkRequest.Status)
}

func isLogGroupWorkRequestResource(resource loggingsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "loggroup", "log_group", "loggroups", "log_groups":
		return true
	}
	if strings.Contains(entityType, "loggroup") || strings.Contains(entityType, "log_group") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/loggroups/")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
