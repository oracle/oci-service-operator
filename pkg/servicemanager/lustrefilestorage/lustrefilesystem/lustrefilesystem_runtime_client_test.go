/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lustrefilesystem

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
)

type fakeLustreFileSystemWorkRequestClient struct {
	request lustrefilestoragesdk.GetWorkRequestRequest
}

func (f *fakeLustreFileSystemWorkRequestClient) GetWorkRequest(
	_ context.Context,
	request lustrefilestoragesdk.GetWorkRequestRequest,
) (lustrefilestoragesdk.GetWorkRequestResponse, error) {
	f.request = request
	return lustrefilestoragesdk.GetWorkRequestResponse{WorkRequest: lustrefilestoragesdk.WorkRequest{Id: common.String("wr-1")}}, nil
}

func TestApplyLustreFileSystemRuntimeHooksProvidesWorkRequestClient(t *testing.T) {
	hooks := newLustreFileSystemDefaultRuntimeHooks(lustrefilestoragesdk.LustreFileStorageClient{})
	client := &fakeLustreFileSystemWorkRequestClient{}
	applyLustreFileSystemRuntimeHooks(&hooks, client, nil)
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("Async.GetWorkRequest is nil")
	}
	result, err := hooks.Async.GetWorkRequest(context.Background(), " wr-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if client.request.WorkRequestId == nil || *client.request.WorkRequestId != "wr-1" {
		t.Fatalf("work request ID = %v", client.request.WorkRequestId)
	}
	if result.(lustrefilestoragesdk.WorkRequest).Id == nil {
		t.Fatalf("work request = %+v", result)
	}
}

func TestApplyLustreFileSystemRuntimeHooksPreservesInitializationError(t *testing.T) {
	hooks := newLustreFileSystemDefaultRuntimeHooks(lustrefilestoragesdk.LustreFileStorageClient{})
	applyLustreFileSystemRuntimeHooks(&hooks, nil, errors.New("provider unavailable"))
	_, err := hooks.Async.GetWorkRequest(context.Background(), "wr-1")
	if err == nil || !strings.Contains(err.Error(), "provider unavailable") {
		t.Fatalf("error = %v", err)
	}
}
