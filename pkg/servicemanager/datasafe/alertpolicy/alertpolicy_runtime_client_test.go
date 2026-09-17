/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alertpolicy

import (
	"context"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
)

type fakeAlertPolicyWorkRequestClient struct {
	response datasafesdk.GetWorkRequestResponse
	err      error
	request  datasafesdk.GetWorkRequestRequest
}

func (f *fakeAlertPolicyWorkRequestClient) GetWorkRequest(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
	f.request = request
	return f.response, f.err
}

func TestAlertPolicyRuntimeHooksConfigureWorkRequestLifecycle(t *testing.T) {
	workRequest := datasafesdk.WorkRequest{Id: common.String("wr-1"), Status: datasafesdk.WorkRequestStatusSucceeded}
	client := &fakeAlertPolicyWorkRequestClient{response: datasafesdk.GetWorkRequestResponse{WorkRequest: workRequest}}
	hooks := newAlertPolicyDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyAlertPolicyRuntimeHooks(&hooks, client, nil)

	if hooks.Semantics == nil || hooks.Semantics.Async == nil || hooks.Semantics.Async.WorkRequest == nil {
		t.Fatal("AlertPolicy work-request semantics were not configured")
	}
	if got, want := hooks.Semantics.Async.WorkRequest.Phases, []string{"create", "update", "delete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("work-request phases = %v, want %v", got, want)
	}
	if got, want := hooks.Semantics.Mutation.Mutable, []string{"displayName", "description", "severity", "freeformTags", "definedTags"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mutable fields = %v, want %v", got, want)
	}
	observed, err := hooks.Async.GetWorkRequest(context.Background(), " wr-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(observed, workRequest) {
		t.Fatalf("GetWorkRequest() = %#v, want %#v", observed, workRequest)
	}
	if client.request.WorkRequestId == nil || *client.request.WorkRequestId != "wr-1" {
		t.Fatalf("GetWorkRequest request = %#v", client.request)
	}
}

func TestAlertPolicyRuntimeHooksPreserveInitializationError(t *testing.T) {
	hooks := newAlertPolicyDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyAlertPolicyRuntimeHooks(&hooks, nil, context.Canceled)
	if _, err := hooks.Async.GetWorkRequest(context.Background(), "wr-1"); err == nil {
		t.Fatal("GetWorkRequest() error = nil, want initialization failure")
	}
}
