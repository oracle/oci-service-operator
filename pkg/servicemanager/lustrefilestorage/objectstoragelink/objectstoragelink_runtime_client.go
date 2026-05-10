/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstoragelink

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
)

var objectStorageLinkWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
		string(lustrefilestoragesdk.OperationTypeDeleteObjectStorageLink),
	},
}

type objectStorageLinkWorkRequestClient interface {
	GetWorkRequest(context.Context, lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error)
}

func init() {
	registerObjectStorageLinkRuntimeHooksMutator(func(manager *ObjectStorageLinkServiceManager, hooks *ObjectStorageLinkRuntimeHooks) {
		workRequestClient, initErr := newObjectStorageLinkWorkRequestClient(manager)
		applyObjectStorageLinkRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newObjectStorageLinkWorkRequestClient(manager *ObjectStorageLinkServiceManager) (objectStorageLinkWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ObjectStorageLink service manager is nil")
	}
	client, err := lustrefilestoragesdk.NewLustreFileStorageClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return lustrefilestoragesdk.LustreFileStorageClient{}, err
	}
	return client, nil
}

func applyObjectStorageLinkRuntimeHooks(
	hooks *ObjectStorageLinkRuntimeHooks,
	workRequestClient objectStorageLinkWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Async.Adapter = objectStorageLinkWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getObjectStorageLinkWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
}

func newObjectStorageLinkRuntimeHooksWithWorkRequestClient(
	workRequestClient objectStorageLinkWorkRequestClient,
) ObjectStorageLinkRuntimeHooks {
	hooks := newObjectStorageLinkDefaultRuntimeHooks(lustrefilestoragesdk.LustreFileStorageClient{})
	applyObjectStorageLinkRuntimeHooks(&hooks, workRequestClient, nil)
	return hooks
}

func getObjectStorageLinkWorkRequest(
	ctx context.Context,
	workRequestClient objectStorageLinkWorkRequestClient,
	initErr error,
	workRequestID string,
) (lustrefilestoragesdk.WorkRequest, error) {
	if initErr != nil {
		return lustrefilestoragesdk.WorkRequest{}, initErr
	}
	if workRequestClient == nil {
		return lustrefilestoragesdk.WorkRequest{}, fmt.Errorf("ObjectStorageLink work request client is nil")
	}

	response, err := workRequestClient.GetWorkRequest(ctx, lustrefilestoragesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return lustrefilestoragesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}
