/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lustrefilesystem

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
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
	DeleteActionTokens: []string{
		string(lustrefilestoragesdk.ActionTypeDeleted),
		string(lustrefilestoragesdk.OperationTypeDeleteLustreFileSystem),
	},
}

type lustreFileSystemWorkRequestClient interface {
	GetWorkRequest(context.Context, lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error)
}

func init() {
	registerLustreFileSystemRuntimeHooksMutator(func(manager *LustreFileSystemServiceManager, hooks *LustreFileSystemRuntimeHooks) {
		client, initErr := newLustreFileSystemWorkRequestClient(manager)
		applyLustreFileSystemRuntimeHooks(hooks, client, initErr)
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
	client lustreFileSystemWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}
	hooks.Async.Adapter = lustreFileSystemWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		if client == nil {
			return nil, fmt.Errorf("LustreFileSystem work request client is nil")
		}
		response, err := client.GetWorkRequest(ctx, lustrefilestoragesdk.GetWorkRequestRequest{
			WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
		})
		if err != nil {
			return nil, err
		}
		return response.WorkRequest, nil
	}
}
