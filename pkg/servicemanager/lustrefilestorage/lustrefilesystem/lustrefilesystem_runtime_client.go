/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lustrefilesystem

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	lustrefilestoragev1beta1 "github.com/oracle/oci-service-operator/api/lustrefilestorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var lustreFileSystemWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(lustrefilestoragesdk.OperationStatusAccepted),
		string(lustrefilestoragesdk.OperationStatusInProgress),
		string(lustrefilestoragesdk.OperationStatusWaiting),
		string(lustrefilestoragesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(lustrefilestoragesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(lustrefilestoragesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(lustrefilestoragesdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(lustrefilestoragesdk.OperationStatusNeedsAttention)},
	CreateActionTokens: []string{
		string(lustrefilestoragesdk.OperationTypeCreateLustreFileSystem),
		string(lustrefilestoragesdk.ActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(lustrefilestoragesdk.OperationTypeUpdateLustreFileSystem),
		string(lustrefilestoragesdk.ActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(lustrefilestoragesdk.OperationTypeDeleteLustreFileSystem),
		string(lustrefilestoragesdk.ActionTypeDeleted),
	},
}

type lustreFileSystemWorkRequestClient interface {
	GetWorkRequest(context.Context, lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error)
}

func init() {
	registerLustreFileSystemRuntimeHooksMutator(func(manager *LustreFileSystemServiceManager, hooks *LustreFileSystemRuntimeHooks) {
		workRequestClient, initErr := newLustreFileSystemWorkRequestClient(manager)
		applyLustreFileSystemRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newLustreFileSystemWorkRequestClient(manager *LustreFileSystemServiceManager) (lustreFileSystemWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("LustreFileSystem service manager is nil")
	}
	client, err := lustrefilestoragesdk.NewLustreFileStorageClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return lustrefilestoragesdk.LustreFileStorageClient{}, err
	}
	return client, nil
}

func applyLustreFileSystemRuntimeHooks(
	hooks *LustreFileSystemRuntimeHooks,
	workRequestClient lustreFileSystemWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Async.Adapter = lustreFileSystemWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getLustreFileSystemWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveLustreFileSystemGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveLustreFileSystemGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverLustreFileSystemIDFromGeneratedWorkRequest
	hooks.Async.Message = lustreFileSystemGeneratedWorkRequestMessage
}

func newLustreFileSystemRuntimeHooksWithWorkRequestClient(
	workRequestClient lustreFileSystemWorkRequestClient,
) LustreFileSystemRuntimeHooks {
	hooks := newLustreFileSystemDefaultRuntimeHooks(lustrefilestoragesdk.LustreFileStorageClient{})
	applyLustreFileSystemRuntimeHooks(&hooks, workRequestClient, nil)
	return hooks
}

func getLustreFileSystemWorkRequest(
	ctx context.Context,
	workRequestClient lustreFileSystemWorkRequestClient,
	initErr error,
	workRequestID string,
) (lustrefilestoragesdk.WorkRequest, error) {
	if initErr != nil {
		return lustrefilestoragesdk.WorkRequest{}, fmt.Errorf("initialize LustreFileSystem OCI client: %w", initErr)
	}
	if workRequestClient == nil {
		return lustrefilestoragesdk.WorkRequest{}, fmt.Errorf("LustreFileSystem work request client is nil")
	}
	if strings.TrimSpace(workRequestID) == "" {
		return lustrefilestoragesdk.WorkRequest{}, fmt.Errorf("LustreFileSystem work request id is empty")
	}

	response, err := workRequestClient.GetWorkRequest(ctx, lustrefilestoragesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return lustrefilestoragesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveLustreFileSystemGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := lustreFileSystemWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveLustreFileSystemGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := lustreFileSystemWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := lustreFileSystemWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverLustreFileSystemIDFromGeneratedWorkRequest(
	_ *lustrefilestoragev1beta1.LustreFileSystem,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := lustreFileSystemWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := lustreFileSystemWorkRequestActionForPhase(phase)
	if id, ok := resolveLustreFileSystemIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveLustreFileSystemIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("LustreFileSystem work request %s does not expose a LustreFileSystem identifier", lustreFileSystemStringValue(current.Id))
}

func lustreFileSystemGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := lustreFileSystemWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("LustreFileSystem %s work request %s is %s", phase, lustreFileSystemStringValue(current.Id), current.Status)
}

func lustreFileSystemWorkRequestFromAny(workRequest any) (lustrefilestoragesdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case lustrefilestoragesdk.WorkRequest:
		return current, nil
	case *lustrefilestoragesdk.WorkRequest:
		if current == nil {
			return lustrefilestoragesdk.WorkRequest{}, fmt.Errorf("LustreFileSystem work request is nil")
		}
		return *current, nil
	default:
		return lustrefilestoragesdk.WorkRequest{}, fmt.Errorf("unexpected LustreFileSystem work request type %T", workRequest)
	}
}

func lustreFileSystemWorkRequestPhaseFromOperationType(
	operationType lustrefilestoragesdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case lustrefilestoragesdk.OperationTypeCreateLustreFileSystem:
		return shared.OSOKAsyncPhaseCreate, true
	case lustrefilestoragesdk.OperationTypeUpdateLustreFileSystem:
		return shared.OSOKAsyncPhaseUpdate, true
	case lustrefilestoragesdk.OperationTypeDeleteLustreFileSystem:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func lustreFileSystemWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) lustrefilestoragesdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return lustrefilestoragesdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return lustrefilestoragesdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return lustrefilestoragesdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveLustreFileSystemIDFromResources(
	resources []lustrefilestoragesdk.WorkRequestResource,
	action lustrefilestoragesdk.ActionTypeEnum,
	preferLustreFileSystemOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferLustreFileSystemOnly && !isLustreFileSystemWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(lustreFileSystemStringValue(resource.Identifier))
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

func isLustreFileSystemWorkRequestResource(resource lustrefilestoragesdk.WorkRequestResource) bool {
	if normalizeLustreFileSystemWorkRequestToken(lustreFileSystemStringValue(resource.EntityType)) == "lustrefilesystem" {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(lustreFileSystemStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/lustrefilesystems/") ||
		strings.Contains(entityURI, "/lustre-file-systems/")
}

func normalizeLustreFileSystemWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func lustreFileSystemStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
