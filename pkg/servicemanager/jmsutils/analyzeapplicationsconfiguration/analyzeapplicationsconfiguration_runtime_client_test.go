/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package analyzeapplicationsconfiguration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmsutilssdk "github.com/oracle/oci-go-sdk/v65/jmsutils"
	jmsutilsv1beta1 "github.com/oracle/oci-service-operator/api/jmsutils/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeAnalyzeApplicationsConfigurationOCIClient struct {
	getFn    func(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error)
	updateFn func(context.Context, jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error)

	getCalls    int
	updateCalls int
}

func (f *fakeAnalyzeApplicationsConfigurationOCIClient) GetAnalyzeApplicationsConfiguration(
	ctx context.Context,
	req jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest,
) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
	f.getCalls++
	if f.getFn == nil {
		return jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse{}, fmt.Errorf("unexpected GetAnalyzeApplicationsConfiguration call")
	}
	return f.getFn(ctx, req)
}

func (f *fakeAnalyzeApplicationsConfigurationOCIClient) UpdateAnalyzeApplicationsConfiguration(
	ctx context.Context,
	req jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest,
) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error) {
	f.updateCalls++
	if f.updateFn == nil {
		return jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse{}, fmt.Errorf("unexpected UpdateAnalyzeApplicationsConfiguration call")
	}
	return f.updateFn(ctx, req)
}

type unsupportedAnalyzeApplicationsConfigurationDelegate struct{}

func (unsupportedAnalyzeApplicationsConfigurationDelegate) CreateOrUpdate(
	context.Context,
	*jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{}, fmt.Errorf("delegate CreateOrUpdate should not be called")
}

func (unsupportedAnalyzeApplicationsConfigurationDelegate) Delete(
	context.Context,
	*jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
) (bool, error) {
	return false, fmt.Errorf("delegate Delete should not be called")
}

func TestApplyAnalyzeApplicationsConfigurationRuntimeHooksWrapsGeneratedClient(t *testing.T) {
	hooks := newAnalyzeApplicationsConfigurationDefaultRuntimeHooks(jmsutilssdk.JmsUtilsClient{})
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{}
	manager := &AnalyzeApplicationsConfigurationServiceManager{
		Provider: common.NewRawConfigurationProvider("ocid1.tenancy.oc1..test", "user", "us-ashburn-1", "fingerprint", "private-key", nil),
	}
	applyAnalyzeApplicationsConfigurationRuntimeHooks(manager, &hooks, client, nil)

	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}
	wrapped := hooks.WrapGeneratedClient[0](unsupportedAnalyzeApplicationsConfigurationDelegate{})
	runtimeClient, ok := wrapped.(*analyzeApplicationsConfigurationRuntimeClient)
	if !ok {
		t.Fatalf("wrapped client type = %T, want *analyzeApplicationsConfigurationRuntimeClient", wrapped)
	}
	requireStringPtr(t, "wrapped compartmentId", runtimeClient.compartmentID, "ocid1.tenancy.oc1..test")
}

func TestCreateOrUpdateBindsExistingConfigurationWhenSpecEmpty(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{}
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{
		getFn: func(_ context.Context, req jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
			requireStringPtr(t, "GetAnalyzeApplicationsConfiguration compartmentId", req.CompartmentId, "ocid1.tenancy.oc1..test")
			return getAnalyzeApplicationsConfigurationResponse("oci-namespace", "analysis-bucket", "opc-get"), nil
		},
		updateFn: func(context.Context, jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error) {
			t.Fatal("UpdateAnalyzeApplicationsConfiguration should not be called for empty desired spec")
			return jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse{}, nil
		},
	}

	response, err := newTestAnalyzeApplicationsConfigurationRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if client.getCalls != 1 {
		t.Fatalf("GetAnalyzeApplicationsConfiguration() calls = %d, want 1", client.getCalls)
	}
	if client.updateCalls != 0 {
		t.Fatalf("UpdateAnalyzeApplicationsConfiguration() calls = %d, want 0", client.updateCalls)
	}
	if resource.Status.NamespaceName != "oci-namespace" {
		t.Fatalf("status.namespaceName = %q, want oci-namespace", resource.Status.NamespaceName)
	}
	if resource.Status.BucketName != "analysis-bucket" {
		t.Fatalf("status.bucketName = %q, want analysis-bucket", resource.Status.BucketName)
	}
	if resource.Status.OsokStatus.OpcRequestID != "" {
		t.Fatalf("status.status.opcRequestId = %q, want empty because no mutating OCI response occurred", resource.Status.OsokStatus.OpcRequestID)
	}
	assertLastCondition(t, resource, shared.Active)
}

func TestCreateOrUpdateSkipsUpdateWhenDesiredMatchesReadback(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{
		Spec: jmsutilsv1beta1.AnalyzeApplicationsConfigurationSpec{
			NamespaceName: "oci-namespace",
			BucketName:    "analysis-bucket",
		},
	}
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{
		getFn: func(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
			return getAnalyzeApplicationsConfigurationResponse("oci-namespace", "analysis-bucket", "opc-get"), nil
		},
		updateFn: func(context.Context, jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error) {
			t.Fatal("UpdateAnalyzeApplicationsConfiguration should not be called when readback matches desired state")
			return jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse{}, nil
		},
	}

	response, err := newTestAnalyzeApplicationsConfigurationRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if client.updateCalls != 0 {
		t.Fatalf("UpdateAnalyzeApplicationsConfiguration() calls = %d, want 0", client.updateCalls)
	}
	assertLastCondition(t, resource, shared.Active)
}

func TestCreateOrUpdateUpdatesChangedFieldsWithIfMatch(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{
		Spec: jmsutilsv1beta1.AnalyzeApplicationsConfigurationSpec{
			NamespaceName: "new-namespace",
			BucketName:    "new-bucket",
		},
	}
	var updateRequest jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{}
	client.getFn = func(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
		switch client.getCalls {
		case 1:
			response := getAnalyzeApplicationsConfigurationResponse("old-namespace", "old-bucket", "opc-get")
			response.Etag = common.String("etag-1")
			return response, nil
		case 2:
			return getAnalyzeApplicationsConfigurationResponse("new-namespace", "new-bucket", "opc-refresh"), nil
		default:
			return jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse{}, fmt.Errorf("unexpected GetAnalyzeApplicationsConfiguration call %d", client.getCalls)
		}
	}
	client.updateFn = func(_ context.Context, req jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error) {
		updateRequest = req
		return jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse{OpcRequestId: common.String("opc-update")}, nil
	}

	response, err := newTestAnalyzeApplicationsConfigurationRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if client.updateCalls != 1 {
		t.Fatalf("UpdateAnalyzeApplicationsConfiguration() calls = %d, want 1", client.updateCalls)
	}
	if client.getCalls != 2 {
		t.Fatalf("GetAnalyzeApplicationsConfiguration() calls = %d, want initial read and post-update readback", client.getCalls)
	}
	requireStringPtr(t, "update namespaceName", updateRequest.NamespaceName, "new-namespace")
	requireStringPtr(t, "update bucketName", updateRequest.BucketName, "new-bucket")
	requireStringPtr(t, "update ifMatch", updateRequest.IfMatch, "etag-1")
	if resource.Status.NamespaceName != "new-namespace" {
		t.Fatalf("status.namespaceName = %q, want new-namespace", resource.Status.NamespaceName)
	}
	if resource.Status.BucketName != "new-bucket" {
		t.Fatalf("status.bucketName = %q, want new-bucket", resource.Status.BucketName)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	assertLastCondition(t, resource, shared.Active)
}

func TestCreateOrUpdateSendsOnlyChangedDesiredFields(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{
		Spec: jmsutilsv1beta1.AnalyzeApplicationsConfigurationSpec{
			NamespaceName: "new-namespace",
			BucketName:    "current-bucket",
		},
	}
	var updateRequest jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{}
	client.getFn = func(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
		switch client.getCalls {
		case 1:
			return getAnalyzeApplicationsConfigurationResponse("current-namespace", "current-bucket", "opc-get"), nil
		case 2:
			return getAnalyzeApplicationsConfigurationResponse("new-namespace", "current-bucket", "opc-refresh"), nil
		default:
			return jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse{}, fmt.Errorf("unexpected GetAnalyzeApplicationsConfiguration call %d", client.getCalls)
		}
	}
	client.updateFn = func(_ context.Context, req jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error) {
		updateRequest = req
		return jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse{OpcRequestId: common.String("opc-update")}, nil
	}

	if _, err := newTestAnalyzeApplicationsConfigurationRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireStringPtr(t, "update namespaceName", updateRequest.NamespaceName, "new-namespace")
	if updateRequest.BucketName != nil {
		t.Fatalf("update bucketName = %q, want nil for unchanged desired field", *updateRequest.BucketName)
	}
}

func TestCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{
		Spec: jmsutilsv1beta1.AnalyzeApplicationsConfigurationSpec{
			NamespaceName: "new-namespace",
		},
	}
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{
		getFn: func(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
			return getAnalyzeApplicationsConfigurationResponse("old-namespace", "bucket", "opc-get"), nil
		},
		updateFn: func(context.Context, jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error) {
			return jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse{}, errortest.NewServiceError(409, "Conflict", "configuration changed")
		},
	}

	response, err := newTestAnalyzeApplicationsConfigurationRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want update error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
	assertLastCondition(t, resource, shared.Failed)
}

func TestDeleteReleasesFinalizerWithoutOCIClient(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{}
	deleted, err := (&analyzeApplicationsConfigurationRuntimeClient{}).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp")
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "delete is not supported") {
		t.Fatalf("status.status.message = %q, want unsupported delete message", resource.Status.OsokStatus.Message)
	}
	assertLastCondition(t, resource, shared.Terminating)
}

func TestCreateOrUpdateRejectsNilResourceBeforeOCI(t *testing.T) {
	client := &fakeAnalyzeApplicationsConfigurationOCIClient{
		getFn: func(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error) {
			t.Fatal("GetAnalyzeApplicationsConfiguration should not be called for nil resource")
			return jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse{}, nil
		},
	}
	_, err := newTestAnalyzeApplicationsConfigurationRuntimeClient(client).CreateOrUpdate(context.Background(), nil, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want nil resource error")
	}
	if !strings.Contains(err.Error(), "resource is nil") {
		t.Fatalf("CreateOrUpdate() error = %v, want nil resource detail", err)
	}
}

func newTestAnalyzeApplicationsConfigurationRuntimeClient(
	client analyzeApplicationsConfigurationOCIClient,
) *analyzeApplicationsConfigurationRuntimeClient {
	return &analyzeApplicationsConfigurationRuntimeClient{
		client:        client,
		compartmentID: common.String("ocid1.tenancy.oc1..test"),
	}
}

func getAnalyzeApplicationsConfigurationResponse(
	namespaceName string,
	bucketName string,
	opcRequestID string,
) jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse {
	return jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse{
		AnalyzeApplicationsConfiguration: jmsutilssdk.AnalyzeApplicationsConfiguration{
			NamespaceName: common.String(namespaceName),
			BucketName:    common.String(bucketName),
		},
		OpcRequestId: common.String(opcRequestID),
	}
}

func requireStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertLastCondition(
	t *testing.T,
	resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func TestAnalyzeApplicationsConfigurationUpdateDetailsNoopsForEmptyDesiredFields(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{}
	details, shouldUpdate := analyzeApplicationsConfigurationUpdateDetails(resource, jmsutilssdk.AnalyzeApplicationsConfiguration{
		NamespaceName: common.String("current-namespace"),
		BucketName:    common.String("current-bucket"),
	})
	if shouldUpdate {
		t.Fatalf("shouldUpdate = true with empty desired fields, details = %#v", details)
	}
	if details.NamespaceName != nil || details.BucketName != nil {
		t.Fatalf("details = %#v, want empty update details", details)
	}
}

func TestAnalyzeApplicationsConfigurationRuntimeClientReportsInitError(t *testing.T) {
	resource := &jmsutilsv1beta1.AnalyzeApplicationsConfiguration{}
	client := &analyzeApplicationsConfigurationRuntimeClient{
		client:  &fakeAnalyzeApplicationsConfigurationOCIClient{},
		initErr: errors.New("missing configuration provider"),
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want init error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "initialize AnalyzeApplicationsConfiguration OCI client") {
		t.Fatalf("CreateOrUpdate() error = %v, want init context", err)
	}
	assertLastCondition(t, resource, shared.Failed)
}
