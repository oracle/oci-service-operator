/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odainstanceattachment

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOdaInstanceID           = "ocid1.odainstance.oc1..test"
	testOdaInstanceAttachmentID = "ocid1.odainstanceattachment.oc1..test"
)

type fakeOdaInstanceAttachmentOCIClient struct {
	createFunc      func(context.Context, odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error)
	getFunc         func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error)
	listFunc        func(context.Context, odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error)
	updateFunc      func(context.Context, odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error)
	deleteFunc      func(context.Context, odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error)
	workRequestFunc func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error)

	createRequests      []odasdk.CreateOdaInstanceAttachmentRequest
	getRequests         []odasdk.GetOdaInstanceAttachmentRequest
	listRequests        []odasdk.ListOdaInstanceAttachmentsRequest
	updateRequests      []odasdk.UpdateOdaInstanceAttachmentRequest
	deleteRequests      []odasdk.DeleteOdaInstanceAttachmentRequest
	workRequestRequests []odasdk.GetWorkRequestRequest
}

func (f *fakeOdaInstanceAttachmentOCIClient) CreateOdaInstanceAttachment(ctx context.Context, request odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return odasdk.CreateOdaInstanceAttachmentResponse{}, nil
}

func (f *fakeOdaInstanceAttachmentOCIClient) GetOdaInstanceAttachment(ctx context.Context, request odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return odasdk.GetOdaInstanceAttachmentResponse{}, nil
}

func (f *fakeOdaInstanceAttachmentOCIClient) ListOdaInstanceAttachments(ctx context.Context, request odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return odasdk.ListOdaInstanceAttachmentsResponse{}, nil
}

func (f *fakeOdaInstanceAttachmentOCIClient) UpdateOdaInstanceAttachment(ctx context.Context, request odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return odasdk.UpdateOdaInstanceAttachmentResponse{}, nil
}

func (f *fakeOdaInstanceAttachmentOCIClient) DeleteOdaInstanceAttachment(ctx context.Context, request odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return odasdk.DeleteOdaInstanceAttachmentResponse{}, nil
}

func (f *fakeOdaInstanceAttachmentOCIClient) GetWorkRequest(ctx context.Context, request odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFunc != nil {
		return f.workRequestFunc(ctx, request)
	}
	return odasdk.GetWorkRequestResponse{}, nil
}

func TestOdaInstanceAttachmentRuntimeSemanticsEncodesWorkRequestLifecycleContract(t *testing.T) {
	hooks := newOdaInstanceAttachmentRuntimeHooksWithOCIClient(&fakeOdaInstanceAttachmentOCIClient{})
	applyOdaInstanceAttachmentRuntimeHooks(
		&OdaInstanceAttachmentServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
		&hooks,
		&fakeOdaInstanceAttachmentOCIClient{},
		nil,
	)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics is nil")
	}
	if got := hooks.Semantics.FormalService; got != "oda" {
		t.Fatalf("FormalService = %q, want oda", got)
	}
	if got := hooks.Semantics.FormalSlug; got != "odainstanceattachment" {
		t.Fatalf("FormalSlug = %q, want odainstanceattachment", got)
	}
	if got := hooks.Semantics.Async.Runtime; got != "handwritten" {
		t.Fatalf("Async.Runtime = %q, want handwritten", got)
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got)
	}
	assertStringSliceEqual(t, "Lifecycle.ProvisioningStates", hooks.Semantics.Lifecycle.ProvisioningStates, []string{"ATTACHING"})
	assertStringSliceEqual(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertStringSliceEqual(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, []string{"DETACHING"})
	assertStringSliceContains(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "attachmentMetadata")
	assertStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "attachToId")
	if len(hooks.Semantics.Unsupported) != 1 || hooks.Semantics.Unsupported[0].Category != "direct-generatedruntime-parent-and-workrequest-shape" {
		t.Fatalf("Unsupported = %#v, want direct-generatedruntime-parent-and-workrequest-shape entry", hooks.Semantics.Unsupported)
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}
}

func TestOdaInstanceAttachmentRequiresOdaInstanceAnnotation(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Annotations = nil
	fake := &fakeOdaInstanceAttachmentOCIClient{
		createFunc: func(context.Context, odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error) {
			t.Fatal("CreateOdaInstanceAttachment should not be called without parent annotation")
			return odasdk.CreateOdaInstanceAttachmentResponse{}, nil
		},
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), odaInstanceAttachmentOdaInstanceIDAnnotation) {
		t.Fatalf("CreateOrUpdate error = %v, want missing annotation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestOdaInstanceAttachmentCreateTracksPendingWorkRequest(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error) {
		return odasdk.ListOdaInstanceAttachmentsResponse{}, nil
	}
	fake.createFunc = func(context.Context, odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error) {
		return odasdk.CreateOdaInstanceAttachmentResponse{
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("create-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateOdaInstanceAttachment,
				odasdk.WorkRequestStatusInProgress,
				"",
			),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending create work request", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-request" {
		t.Fatalf("opcRequestId = %q, want create-request", got)
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("get requests = %d, want 0 while work request is pending", len(fake.getRequests))
	}
}

func TestOdaInstanceAttachmentCreateProjectsStatusAfterSucceededWorkRequest(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error) {
		return odasdk.ListOdaInstanceAttachmentsResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request odasdk.CreateOdaInstanceAttachmentRequest) (odasdk.CreateOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("create odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.CreateOdaInstanceAttachmentDetails.AttachToId); got != resource.Spec.AttachToId {
			t.Fatalf("create attachToId = %q, want %q", got, resource.Spec.AttachToId)
		}
		if got := string(request.CreateOdaInstanceAttachmentDetails.AttachmentType); got != resource.Spec.AttachmentType {
			t.Fatalf("create attachmentType = %q, want %q", got, resource.Spec.AttachmentType)
		}
		if got := stringValue(request.CreateOdaInstanceAttachmentDetails.Owner.OwnerServiceName); got != resource.Spec.Owner.OwnerServiceName {
			t.Fatalf("create ownerServiceName = %q, want %q", got, resource.Spec.Owner.OwnerServiceName)
		}
		return odasdk.CreateOdaInstanceAttachmentResponse{
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("create-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateOdaInstanceAttachment,
				odasdk.WorkRequestStatusSucceeded,
				testOdaInstanceAttachmentID,
			),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("get odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AttachmentId); got != testOdaInstanceAttachmentID {
			t.Fatalf("get attachmentId = %q, want %q", got, testOdaInstanceAttachmentID)
		}
		if request.IncludeOwnerMetadata == nil || !*request.IncludeOwnerMetadata {
			t.Fatalf("get includeOwnerMetadata = %#v, want true", request.IncludeOwnerMetadata)
		}
		return odasdk.GetOdaInstanceAttachmentResponse{
			OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1", len(fake.listRequests))
	}
	if got := stringValue(fake.listRequests[0].OdaInstanceId); got != testOdaInstanceID {
		t.Fatalf("list odaInstanceId = %q, want %q", got, testOdaInstanceID)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	assertOdaInstanceAttachmentActiveStatus(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-request" {
		t.Fatalf("opcRequestId = %q, want create-request", got)
	}
}

func TestOdaInstanceAttachmentBindsExistingWithoutCreate(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListOdaInstanceAttachmentsRequest) (odasdk.ListOdaInstanceAttachmentsResponse, error) {
		return odasdk.ListOdaInstanceAttachmentsResponse{
			OdaInstanceAttachmentCollection: odasdk.OdaInstanceAttachmentCollection{
				Items: []odasdk.OdaInstanceAttachmentSummary{
					makeSDKOdaInstanceAttachmentSummary(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		return odasdk.GetOdaInstanceAttachmentResponse{
			OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	assertOdaInstanceAttachmentActiveStatus(t, resource)
}

func TestOdaInstanceAttachmentUpdatesSupportedMutableDrift(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Status.Id = testOdaInstanceAttachmentID
	resource.Status.InstanceId = testOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
	resource.Spec.AttachmentMetadata = "metadata-v2"
	resource.Spec.RestrictedOperations = []string{"UPDATE", "DELETE"}
	resource.Spec.Owner = odav1beta1.OdaInstanceAttachmentOwner{
		OwnerServiceName:    "owner-service-v2",
		OwnerServiceTenancy: "ocid1.tenancy.oc1..owner2",
	}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": shared.MapValue{"CostCenter": "42"}}

	current := makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive)
	current.AttachmentMetadata = common.String("metadata-v1")
	current.RestrictedOperations = []string{"DELETE"}
	current.Owner = &odasdk.OdaInstanceOwner{
		OwnerServiceName:    common.String("owner-service-v1"),
		OwnerServiceTenancy: common.String("ocid1.tenancy.oc1..owner1"),
	}
	current.FreeformTags = map[string]string{"env": "dev"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
	updated := makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive)

	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetOdaInstanceAttachmentResponse{OdaInstanceAttachment: current}, nil
		}
		return odasdk.GetOdaInstanceAttachmentResponse{OdaInstanceAttachment: updated}, nil
	}
	fake.updateFunc = func(_ context.Context, request odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("update odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AttachmentId); got != testOdaInstanceAttachmentID {
			t.Fatalf("update attachmentId = %q, want %q", got, testOdaInstanceAttachmentID)
		}
		return odasdk.UpdateOdaInstanceAttachmentResponse{
			OpcWorkRequestId: common.String("wr-update"),
			OpcRequestId:     common.String("update-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-update",
				odasdk.WorkRequestRequestActionUpdateOdaInstanceAttachment,
				odasdk.WorkRequestStatusSucceeded,
				testOdaInstanceAttachmentID,
			),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	details := fake.updateRequests[0].UpdateOdaInstanceAttachmentDetails
	if got := stringValue(details.AttachmentMetadata); got != resource.Spec.AttachmentMetadata {
		t.Fatalf("updated attachmentMetadata = %q, want %q", got, resource.Spec.AttachmentMetadata)
	}
	assertStringSliceEqual(t, "updated restrictedOperations", details.RestrictedOperations, resource.Spec.RestrictedOperations)
	if got := stringValue(details.Owner.OwnerServiceName); got != resource.Spec.Owner.OwnerServiceName {
		t.Fatalf("updated ownerServiceName = %q, want %q", got, resource.Spec.Owner.OwnerServiceName)
	}
	if got := details.FreeformTags["env"]; got != "prod" {
		t.Fatalf("updated freeform env = %q, want prod", got)
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("updated defined tag CostCenter = %#v, want 42", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "update-request" {
		t.Fatalf("opcRequestId = %q, want update-request", got)
	}
	assertOdaInstanceAttachmentActiveStatus(t, resource)
}

func TestOdaInstanceAttachmentRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Status.Id = testOdaInstanceAttachmentID
	resource.Status.InstanceId = testOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
	current := makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive)
	current.AttachToId = common.String("ocid1.target.oc1..different")
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		return odasdk.GetOdaInstanceAttachmentResponse{OdaInstanceAttachment: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error) {
		t.Fatal("UpdateOdaInstanceAttachment should not be called for create-only drift")
		return odasdk.UpdateOdaInstanceAttachmentResponse{}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "attachToId") {
		t.Fatalf("CreateOrUpdate error = %v, want create-only attachToId drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
}

func TestOdaInstanceAttachmentClassifiesLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		state         odasdk.OdaInstanceAttachmentLifecycleStateEnum
		wantReason    shared.OSOKConditionType
		wantRequeue   bool
		wantSuccess   bool
		wantAsync     bool
		wantAsyncNorm shared.OSOKAsyncNormalizedClass
	}{
		{name: "attaching", state: odasdk.OdaInstanceAttachmentLifecycleStateAttaching, wantReason: shared.Provisioning, wantRequeue: true, wantSuccess: true, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassPending},
		{name: "detaching", state: odasdk.OdaInstanceAttachmentLifecycleStateDetaching, wantReason: shared.Terminating, wantRequeue: true, wantSuccess: true, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassPending},
		{name: "active", state: odasdk.OdaInstanceAttachmentLifecycleStateActive, wantReason: shared.Active, wantRequeue: false, wantSuccess: true, wantAsync: false},
		{name: "failed", state: odasdk.OdaInstanceAttachmentLifecycleStateFailed, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassFailed},
		{name: "inactive", state: odasdk.OdaInstanceAttachmentLifecycleStateInactive, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := makeOdaInstanceAttachmentResource()
			resource.Status.Id = testOdaInstanceAttachmentID
			resource.Status.InstanceId = testOdaInstanceID
			resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
			fake := &fakeOdaInstanceAttachmentOCIClient{}
			fake.getFunc = func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
				return odasdk.GetOdaInstanceAttachmentResponse{
					OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, tt.state),
				}, nil
			}
			fake.updateFunc = func(context.Context, odasdk.UpdateOdaInstanceAttachmentRequest) (odasdk.UpdateOdaInstanceAttachmentResponse, error) {
				t.Fatal("UpdateOdaInstanceAttachment should not be called for lifecycle classification")
				return odasdk.UpdateOdaInstanceAttachmentResponse{}, nil
			}
			client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate returned error: %v", err)
			}
			if response.IsSuccessful != tt.wantSuccess {
				t.Fatalf("IsSuccessful = %v, want %v", response.IsSuccessful, tt.wantSuccess)
			}
			if response.ShouldRequeue != tt.wantRequeue {
				t.Fatalf("ShouldRequeue = %v, want %v", response.ShouldRequeue, tt.wantRequeue)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tt.wantReason) {
				t.Fatalf("status.reason = %q, want %q", got, tt.wantReason)
			}
			if tt.wantAsync {
				if resource.Status.OsokStatus.Async.Current == nil {
					t.Fatalf("async.current = nil, want %s", tt.wantAsyncNorm)
				}
				if got := resource.Status.OsokStatus.Async.Current.NormalizedClass; got != tt.wantAsyncNorm {
					t.Fatalf("async.normalizedClass = %q, want %q", got, tt.wantAsyncNorm)
				}
			} else if resource.Status.OsokStatus.Async.Current != nil {
				t.Fatalf("async.current = %#v, want nil", resource.Status.OsokStatus.Async.Current)
			}
		})
	}
}

func TestOdaInstanceAttachmentDeleteWaitsForWorkRequestConfirmation(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Status.Id = testOdaInstanceAttachmentID
	resource.Status.InstanceId = testOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		return odasdk.GetOdaInstanceAttachmentResponse{
			OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("delete odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AttachmentId); got != testOdaInstanceAttachmentID {
			t.Fatalf("delete attachmentId = %q, want %q", got, testOdaInstanceAttachmentID)
		}
		return odasdk.DeleteOdaInstanceAttachmentResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("delete-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-delete",
				odasdk.WorkRequestRequestActionDeleteOdaInstanceAttachment,
				odasdk.WorkRequestStatusInProgress,
				testOdaInstanceAttachmentID,
			),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want false while delete work request is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("DeletedAt = %#v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "delete-request" {
		t.Fatalf("opcRequestId = %q, want delete-request", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending delete work request", current)
	}
}

func TestOdaInstanceAttachmentDeleteUsesStatusParentWhenAnnotationMissing(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Annotations = nil
	resource.Status.Id = testOdaInstanceAttachmentID
	resource.Status.InstanceId = testOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.getFunc = func(_ context.Context, request odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("get odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AttachmentId); got != testOdaInstanceAttachmentID {
			t.Fatalf("get attachmentId = %q, want %q", got, testOdaInstanceAttachmentID)
		}
		return odasdk.GetOdaInstanceAttachmentResponse{
			OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("delete odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AttachmentId); got != testOdaInstanceAttachmentID {
			t.Fatalf("delete attachmentId = %q, want %q", got, testOdaInstanceAttachmentID)
		}
		return odasdk.DeleteOdaInstanceAttachmentResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("delete-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-delete",
				odasdk.WorkRequestRequestActionDeleteOdaInstanceAttachment,
				odasdk.WorkRequestStatusInProgress,
				testOdaInstanceAttachmentID,
			),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want false while delete work request is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("DeletedAt = %#v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending delete work request", current)
	}
}

func TestOdaInstanceAttachmentDeleteSucceededWorkRequestReadbackPresentDoesNotDuplicateDelete(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Annotations = nil
	resource.Status.Id = testOdaInstanceAttachmentID
	resource.Status.InstanceId = testOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.getFunc = func(_ context.Context, request odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("get odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AttachmentId); got != testOdaInstanceAttachmentID {
			t.Fatalf("get attachmentId = %q, want %q", got, testOdaInstanceAttachmentID)
		}
		return odasdk.GetOdaInstanceAttachmentResponse{
			OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error) {
		t.Fatal("DeleteOdaInstanceAttachment should not be called while delete work request confirmation is pending")
		return odasdk.DeleteOdaInstanceAttachmentResponse{}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-delete",
				odasdk.WorkRequestRequestActionDeleteOdaInstanceAttachment,
				odasdk.WorkRequestStatusSucceeded,
				testOdaInstanceAttachmentID,
			),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	for i := 0; i < 2; i++ {
		deleted, err := client.Delete(context.Background(), resource)
		if err != nil {
			t.Fatalf("Delete #%d returned error: %v", i+1, err)
		}
		if deleted {
			t.Fatalf("Delete #%d returned deleted=true, want false while readback still finds attachment", i+1)
		}
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 while waiting for readback confirmation", len(fake.deleteRequests))
	}
	if len(fake.workRequestRequests) != 2 {
		t.Fatalf("work request polls = %d, want 2", len(fake.workRequestRequests))
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("get requests = %d, want 2", len(fake.getRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("DeletedAt = %#v, want nil while readback still finds attachment", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete || current.WorkRequestID != "wr-delete" || current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		t.Fatalf("async.current = %#v, want succeeded delete work request to remain tracked", current)
	}
}

func TestOdaInstanceAttachmentDeleteConfirmsReadNotFoundAfterSucceededWorkRequest(t *testing.T) {
	resource := makeOdaInstanceAttachmentResource()
	resource.Status.Id = testOdaInstanceAttachmentID
	resource.Status.InstanceId = testOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOdaInstanceAttachmentID)
	fake := &fakeOdaInstanceAttachmentOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetOdaInstanceAttachmentRequest) (odasdk.GetOdaInstanceAttachmentResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetOdaInstanceAttachmentResponse{
				OdaInstanceAttachment: makeSDKOdaInstanceAttachment(testOdaInstanceAttachmentID, resource, odasdk.OdaInstanceAttachmentLifecycleStateActive),
			}, nil
		}
		return odasdk.GetOdaInstanceAttachmentResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "oda instance attachment not found")
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteOdaInstanceAttachmentRequest) (odasdk.DeleteOdaInstanceAttachmentResponse, error) {
		return odasdk.DeleteOdaInstanceAttachmentResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("delete-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKOdaInstanceAttachmentWorkRequest(
				"wr-delete",
				odasdk.WorkRequestRequestActionDeleteOdaInstanceAttachment,
				odasdk.WorkRequestStatusSucceeded,
				testOdaInstanceAttachmentID,
			),
		}, nil
	}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete returned deleted=false, want true after confirm read not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want timestamp after deletion confirmation")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("async.current = %#v, want nil after delete confirmation", current)
	}
}

func makeOdaInstanceAttachmentResource() *odav1beta1.OdaInstanceAttachment {
	return &odav1beta1.OdaInstanceAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-attachment",
			Namespace: "default",
			Annotations: map[string]string{
				odaInstanceAttachmentOdaInstanceIDAnnotation: testOdaInstanceID,
			},
		},
		Spec: odav1beta1.OdaInstanceAttachmentSpec{
			AttachToId:           "ocid1.target.oc1..test",
			AttachmentType:       string(odasdk.OdaInstanceAttachmentAttachmentTypeFusion),
			AttachmentMetadata:   "metadata-v1",
			RestrictedOperations: []string{"DELETE"},
			Owner: odav1beta1.OdaInstanceAttachmentOwner{
				OwnerServiceName:    "owner-service-v1",
				OwnerServiceTenancy: "ocid1.tenancy.oc1..owner1",
			},
		},
	}
}

func makeSDKOdaInstanceAttachment(id string, resource *odav1beta1.OdaInstanceAttachment, state odasdk.OdaInstanceAttachmentLifecycleStateEnum) odasdk.OdaInstanceAttachment {
	return odasdk.OdaInstanceAttachment{
		Id:                   common.String(id),
		InstanceId:           common.String(testOdaInstanceID),
		AttachToId:           common.String(resource.Spec.AttachToId),
		AttachmentType:       odasdk.OdaInstanceAttachmentAttachmentTypeEnum(resource.Spec.AttachmentType),
		LifecycleState:       state,
		AttachmentMetadata:   optionalString(resource.Spec.AttachmentMetadata),
		RestrictedOperations: append([]string(nil), resource.Spec.RestrictedOperations...),
		Owner: &odasdk.OdaInstanceOwner{
			OwnerServiceName:    common.String(resource.Spec.Owner.OwnerServiceName),
			OwnerServiceTenancy: common.String(resource.Spec.Owner.OwnerServiceTenancy),
		},
		FreeformTags: mapsClone(resource.Spec.FreeformTags),
		DefinedTags:  odaInstanceAttachmentDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKOdaInstanceAttachmentSummary(id string, resource *odav1beta1.OdaInstanceAttachment, state odasdk.OdaInstanceAttachmentLifecycleStateEnum) odasdk.OdaInstanceAttachmentSummary {
	return odasdk.OdaInstanceAttachmentSummary{
		Id:                   common.String(id),
		InstanceId:           common.String(testOdaInstanceID),
		AttachToId:           common.String(resource.Spec.AttachToId),
		AttachmentType:       odasdk.OdaInstanceAttachmentSummaryAttachmentTypeEnum(resource.Spec.AttachmentType),
		LifecycleState:       state,
		AttachmentMetadata:   optionalString(resource.Spec.AttachmentMetadata),
		RestrictedOperations: append([]string(nil), resource.Spec.RestrictedOperations...),
		Owner: &odasdk.OdaInstanceOwner{
			OwnerServiceName:    common.String(resource.Spec.Owner.OwnerServiceName),
			OwnerServiceTenancy: common.String(resource.Spec.Owner.OwnerServiceTenancy),
		},
		FreeformTags: mapsClone(resource.Spec.FreeformTags),
		DefinedTags:  odaInstanceAttachmentDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKOdaInstanceAttachmentWorkRequest(
	id string,
	action odasdk.WorkRequestRequestActionEnum,
	status odasdk.WorkRequestStatusEnum,
	resourceID string,
) odasdk.WorkRequest {
	workRequest := odasdk.WorkRequest{
		Id:              common.String(id),
		OdaInstanceId:   common.String(testOdaInstanceID),
		ResourceId:      optionalString(resourceID),
		RequestAction:   action,
		Status:          status,
		PercentComplete: common.Float32(50),
	}
	if resourceID != "" {
		workRequest.Resources = []odasdk.WorkRequestResource{
			{
				ResourceAction: workRequestResourceActionForRequestAction(action),
				ResourceId:     common.String(resourceID),
			},
		}
	}
	return workRequest
}

func workRequestResourceActionForRequestAction(action odasdk.WorkRequestRequestActionEnum) odasdk.WorkRequestResourceResourceActionEnum {
	switch action {
	case odasdk.WorkRequestRequestActionUpdateOdaInstanceAttachment:
		return odasdk.WorkRequestResourceResourceActionUpdateOdaInstanceAttachment
	case odasdk.WorkRequestRequestActionDeleteOdaInstanceAttachment:
		return odasdk.WorkRequestResourceResourceActionDeleteOdaInstanceAttachment
	default:
		return odasdk.WorkRequestResourceResourceActionCreateOdaInstanceAttachment
	}
}

func assertOdaInstanceAttachmentActiveStatus(t *testing.T, resource *odav1beta1.OdaInstanceAttachment) {
	t.Helper()
	if got := resource.Status.Id; got != testOdaInstanceAttachmentID {
		t.Fatalf("status.id = %q, want %q", got, testOdaInstanceAttachmentID)
	}
	if got := resource.Status.InstanceId; got != testOdaInstanceID {
		t.Fatalf("status.instanceId = %q, want %q", got, testOdaInstanceID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testOdaInstanceAttachmentID {
		t.Fatalf("status.ocid = %q, want %q", got, testOdaInstanceAttachmentID)
	}
	if got := resource.Status.LifecycleState; got != string(odasdk.OdaInstanceAttachmentLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Active)
	}
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("async.current = %#v, want nil for ACTIVE", current)
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func mapsClone(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	return cloneStringMap(input)
}

func assertStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}

func assertStringSliceContains(t *testing.T, name string, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %#v, want entry %q", name, got, want)
}
