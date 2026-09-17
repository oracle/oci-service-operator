/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package stack

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"

	resourcemanagersdk "github.com/oracle/oci-go-sdk/v65/resourcemanager"
	resourcemanagerv1beta1 "github.com/oracle/oci-service-operator/api/resourcemanager/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func mockStackZip(t *testing.T) string {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("main.tf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("terraform { required_version = \">= 1.0\" }\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func newMockStackClient(sdkClient resourcemanagersdk.ResourceManagerClient) StackServiceClient {
	manager := &StackServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newStackRuntimeHooks(manager, sdkClient)
	delegate := defaultStackServiceClient{ServiceClient: generatedruntime.NewServiceClient[*resourcemanagerv1beta1.Stack](buildStackGeneratedRuntimeConfig(manager, hooks))}
	return wrapStackGeneratedClient(hooks, delegate)
}
