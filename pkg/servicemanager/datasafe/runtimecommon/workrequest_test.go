/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package runtimecommon

import (
	"context"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type fakeWorkRequestClient struct {
	response datasafesdk.GetWorkRequestResponse
	request  datasafesdk.GetWorkRequestRequest
}

func (f *fakeWorkRequestClient) GetWorkRequest(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
	f.request = request
	return f.response, nil
}

func TestConfigureWorkRequest(t *testing.T) {
	want := datasafesdk.WorkRequest{Id: common.String("wr-1"), Status: datasafesdk.WorkRequestStatusSucceeded}
	client := &fakeWorkRequestClient{response: datasafesdk.GetWorkRequestResponse{WorkRequest: want}}
	hooks := generatedruntime.AsyncHooks[struct{}]{}
	ConfigureWorkRequest(&hooks, client, nil, "Thing")

	got, err := hooks.GetWorkRequest(context.Background(), " wr-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetWorkRequest() = %#v, want %#v", got, want)
	}
	if client.request.WorkRequestId == nil || *client.request.WorkRequestId != "wr-1" {
		t.Fatalf("GetWorkRequest request = %#v", client.request)
	}
	if class, err := hooks.Adapter.Normalize(string(datasafesdk.WorkRequestStatusInProgress)); err != nil || class == "" {
		t.Fatalf("Normalize(IN_PROGRESS) = %q, %v", class, err)
	}
}

func TestConfigureWorkRequestPreservesInitializationError(t *testing.T) {
	hooks := generatedruntime.AsyncHooks[struct{}]{}
	ConfigureWorkRequest(&hooks, nil, context.Canceled, "Thing")
	if _, err := hooks.GetWorkRequest(context.Background(), "wr-1"); err == nil {
		t.Fatal("GetWorkRequest() error = nil, want initialization failure")
	}
}
