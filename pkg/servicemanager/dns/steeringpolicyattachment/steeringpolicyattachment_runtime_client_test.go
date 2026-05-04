/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package steeringpolicyattachment

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSteeringPolicyAttachmentID = "ocid1.steeringpolicyattachment.oc1..example"
	testSteeringPolicyID           = "ocid1.steeringpolicy.oc1..example"
	testZoneID                     = "ocid1.dns-zone.oc1..example"
	testCompartmentID              = "ocid1.compartment.oc1..example"
	testDomainName                 = "www.example.com"
	testAttachmentDisplayName      = "sample attachment"
)

type fakeSteeringPolicyAttachmentOCIClient struct {
	createFn            func(context.Context, dnssdk.CreateSteeringPolicyAttachmentRequest) (dnssdk.CreateSteeringPolicyAttachmentResponse, error)
	getFn               func(context.Context, dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error)
	listFn              func(context.Context, dnssdk.ListSteeringPolicyAttachmentsRequest) (dnssdk.ListSteeringPolicyAttachmentsResponse, error)
	updateFn            func(context.Context, dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error)
	deleteFn            func(context.Context, dnssdk.DeleteSteeringPolicyAttachmentRequest) (dnssdk.DeleteSteeringPolicyAttachmentResponse, error)
	getSteeringPolicyFn func(context.Context, dnssdk.GetSteeringPolicyRequest) (dnssdk.GetSteeringPolicyResponse, error)
}

func (f *fakeSteeringPolicyAttachmentOCIClient) CreateSteeringPolicyAttachment(ctx context.Context, req dnssdk.CreateSteeringPolicyAttachmentRequest) (dnssdk.CreateSteeringPolicyAttachmentResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return dnssdk.CreateSteeringPolicyAttachmentResponse{}, nil
}

func (f *fakeSteeringPolicyAttachmentOCIClient) GetSteeringPolicyAttachment(ctx context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return dnssdk.GetSteeringPolicyAttachmentResponse{}, nil
}

func (f *fakeSteeringPolicyAttachmentOCIClient) ListSteeringPolicyAttachments(ctx context.Context, req dnssdk.ListSteeringPolicyAttachmentsRequest) (dnssdk.ListSteeringPolicyAttachmentsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return dnssdk.ListSteeringPolicyAttachmentsResponse{}, nil
}

func (f *fakeSteeringPolicyAttachmentOCIClient) UpdateSteeringPolicyAttachment(ctx context.Context, req dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return dnssdk.UpdateSteeringPolicyAttachmentResponse{}, nil
}

func (f *fakeSteeringPolicyAttachmentOCIClient) DeleteSteeringPolicyAttachment(ctx context.Context, req dnssdk.DeleteSteeringPolicyAttachmentRequest) (dnssdk.DeleteSteeringPolicyAttachmentResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return dnssdk.DeleteSteeringPolicyAttachmentResponse{}, nil
}

func (f *fakeSteeringPolicyAttachmentOCIClient) GetSteeringPolicy(ctx context.Context, req dnssdk.GetSteeringPolicyRequest) (dnssdk.GetSteeringPolicyResponse, error) {
	if f.getSteeringPolicyFn != nil {
		return f.getSteeringPolicyFn(ctx, req)
	}
	return dnssdk.GetSteeringPolicyResponse{
		SteeringPolicy: dnssdk.SteeringPolicy{
			Id:            common.String(testSteeringPolicyID),
			CompartmentId: common.String(testCompartmentID),
		},
	}, nil
}

func testSteeringPolicyAttachmentClient(fake *fakeSteeringPolicyAttachmentOCIClient) SteeringPolicyAttachmentServiceClient {
	if fake == nil {
		fake = &fakeSteeringPolicyAttachmentOCIClient{}
	}

	manager := &SteeringPolicyAttachmentServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
	hooks := newSteeringPolicyAttachmentDefaultRuntimeHooks(dnssdk.DnsClient{})
	applySteeringPolicyAttachmentRuntimeHooks(&hooks, fake, nil)
	return defaultSteeringPolicyAttachmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.SteeringPolicyAttachment](
			buildSteeringPolicyAttachmentGeneratedRuntimeConfig(manager, hooks),
		),
	}
}

func makeSteeringPolicyAttachmentResource() *dnsv1beta1.SteeringPolicyAttachment {
	return &dnsv1beta1.SteeringPolicyAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "steering-policy-attachment-sample",
			Namespace: "default",
		},
		Spec: dnsv1beta1.SteeringPolicyAttachmentSpec{
			SteeringPolicyId: testSteeringPolicyID,
			ZoneId:           testZoneID,
			DomainName:       testDomainName,
			DisplayName:      testAttachmentDisplayName,
		},
	}
}

func makeTrackedSteeringPolicyAttachmentResource() *dnsv1beta1.SteeringPolicyAttachment {
	resource := makeSteeringPolicyAttachmentResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSteeringPolicyAttachmentID)
	resource.Status.Id = testSteeringPolicyAttachmentID
	resource.Status.CompartmentId = testCompartmentID
	resource.Status.SteeringPolicyId = testSteeringPolicyID
	resource.Status.ZoneId = testZoneID
	resource.Status.DomainName = testDomainName
	resource.Status.DisplayName = testAttachmentDisplayName
	resource.Status.LifecycleState = string(dnssdk.SteeringPolicyAttachmentLifecycleStateActive)
	return resource
}

func makeSDKSteeringPolicyAttachment(id, displayName, steeringPolicyID, zoneID, domainName string, state dnssdk.SteeringPolicyAttachmentLifecycleStateEnum) dnssdk.SteeringPolicyAttachment {
	return dnssdk.SteeringPolicyAttachment{
		Id:               common.String(id),
		SteeringPolicyId: common.String(steeringPolicyID),
		ZoneId:           common.String(zoneID),
		DomainName:       common.String(domainName),
		DisplayName:      common.String(displayName),
		Rtypes:           []string{"A"},
		CompartmentId:    common.String(testCompartmentID),
		Self:             common.String("https://example.invalid/steeringPolicyAttachments/" + id),
		TimeCreated:      &common.SDKTime{Time: time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC)},
		LifecycleState:   state,
	}
}

func makeSDKSteeringPolicyAttachmentSummary(id, displayName string, state dnssdk.SteeringPolicyAttachmentSummaryLifecycleStateEnum) dnssdk.SteeringPolicyAttachmentSummary {
	return dnssdk.SteeringPolicyAttachmentSummary{
		Id:               common.String(id),
		SteeringPolicyId: common.String(testSteeringPolicyID),
		ZoneId:           common.String(testZoneID),
		DomainName:       common.String(testDomainName),
		DisplayName:      common.String(displayName),
		Rtypes:           []string{"A"},
		CompartmentId:    common.String(testCompartmentID),
		Self:             common.String("https://example.invalid/steeringPolicyAttachments/" + id),
		TimeCreated:      &common.SDKTime{Time: time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC)},
		LifecycleState:   state,
	}
}

type createAndProjectActiveStatusTestState struct {
	t                      *testing.T
	resource               *dnsv1beta1.SteeringPolicyAttachment
	createRequest          dnssdk.CreateSteeringPolicyAttachmentRequest
	getCalls               int
	listCalls              int
	getSteeringPolicyCalls int
}

func newCreateAndProjectActiveStatusTestState(
	t *testing.T,
	resource *dnsv1beta1.SteeringPolicyAttachment,
) (*createAndProjectActiveStatusTestState, SteeringPolicyAttachmentServiceClient) {
	t.Helper()

	state := &createAndProjectActiveStatusTestState{
		t:        t,
		resource: resource,
	}
	client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
		getSteeringPolicyFn: state.getSteeringPolicy,
		listFn:              state.listSteeringPolicyAttachments,
		createFn:            state.createSteeringPolicyAttachment,
		getFn:               state.getSteeringPolicyAttachment,
	})
	return state, client
}

func (s *createAndProjectActiveStatusTestState) getSteeringPolicy(_ context.Context, req dnssdk.GetSteeringPolicyRequest) (dnssdk.GetSteeringPolicyResponse, error) {
	s.getSteeringPolicyCalls++
	requireSteeringPolicyAttachmentStringPtr(s.t, "get steering policy id", req.SteeringPolicyId, testSteeringPolicyID)
	if req.Scope != dnssdk.GetSteeringPolicyScopeGlobal {
		s.t.Fatalf("get steering policy scope = %q, want GLOBAL", req.Scope)
	}
	return dnssdk.GetSteeringPolicyResponse{
		SteeringPolicy: dnssdk.SteeringPolicy{
			Id:            common.String(testSteeringPolicyID),
			CompartmentId: common.String(testCompartmentID),
		},
	}, nil
}

func (s *createAndProjectActiveStatusTestState) listSteeringPolicyAttachments(_ context.Context, req dnssdk.ListSteeringPolicyAttachmentsRequest) (dnssdk.ListSteeringPolicyAttachmentsResponse, error) {
	s.listCalls++
	requireSteeringPolicyAttachmentStringPtr(s.t, "list compartmentId", req.CompartmentId, testCompartmentID)
	requireSteeringPolicyAttachmentStringPtr(s.t, "list steeringPolicyId", req.SteeringPolicyId, testSteeringPolicyID)
	requireSteeringPolicyAttachmentStringPtr(s.t, "list zoneId", req.ZoneId, testZoneID)
	requireSteeringPolicyAttachmentStringPtr(s.t, "list domain", req.Domain, testDomainName)
	if req.DisplayName != nil {
		s.t.Fatalf("list displayName = %v, want nil so mutable drift does not hide existing attachments", req.DisplayName)
	}
	if req.Scope != dnssdk.ListSteeringPolicyAttachmentsScopeGlobal {
		s.t.Fatalf("list scope = %q, want GLOBAL", req.Scope)
	}
	return dnssdk.ListSteeringPolicyAttachmentsResponse{}, nil
}

func (s *createAndProjectActiveStatusTestState) createSteeringPolicyAttachment(_ context.Context, req dnssdk.CreateSteeringPolicyAttachmentRequest) (dnssdk.CreateSteeringPolicyAttachmentResponse, error) {
	s.createRequest = req
	return dnssdk.CreateSteeringPolicyAttachmentResponse{
		SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
			testSteeringPolicyAttachmentID,
			s.resource.Spec.DisplayName,
			testSteeringPolicyID,
			testZoneID,
			testDomainName,
			dnssdk.SteeringPolicyAttachmentLifecycleStateCreating,
		),
		OpcRequestId:     common.String("opc-create-1"),
		OpcWorkRequestId: common.String("wr-create-1"),
	}, nil
}

func (s *createAndProjectActiveStatusTestState) getSteeringPolicyAttachment(_ context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
	s.getCalls++
	requireSteeringPolicyAttachmentStringPtr(s.t, "get attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
	if req.Scope != dnssdk.GetSteeringPolicyAttachmentScopeGlobal {
		s.t.Fatalf("get scope = %q, want GLOBAL", req.Scope)
	}
	return dnssdk.GetSteeringPolicyAttachmentResponse{
		SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
			testSteeringPolicyAttachmentID,
			s.resource.Spec.DisplayName,
			testSteeringPolicyID,
			testZoneID,
			testDomainName,
			dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
		),
	}, nil
}

func (s *createAndProjectActiveStatusTestState) requireCalls() {
	s.t.Helper()
	if s.getSteeringPolicyCalls != 1 {
		s.t.Fatalf("GetSteeringPolicy() calls = %d, want 1 compartment lookup", s.getSteeringPolicyCalls)
	}
	if s.listCalls != 1 {
		s.t.Fatalf("ListSteeringPolicyAttachments() calls = %d, want 1", s.listCalls)
	}
	if s.getCalls != 1 {
		s.t.Fatalf("GetSteeringPolicyAttachment() calls = %d, want 1 follow-up read", s.getCalls)
	}
}

func (s *createAndProjectActiveStatusTestState) requireCreateRequest() {
	s.t.Helper()
	requireSteeringPolicyAttachmentStringPtr(s.t, "create steeringPolicyId", s.createRequest.SteeringPolicyId, testSteeringPolicyID)
	requireSteeringPolicyAttachmentStringPtr(s.t, "create zoneId", s.createRequest.ZoneId, testZoneID)
	requireSteeringPolicyAttachmentStringPtr(s.t, "create domainName", s.createRequest.DomainName, testDomainName)
	requireSteeringPolicyAttachmentStringPtr(s.t, "create displayName", s.createRequest.DisplayName, s.resource.Spec.DisplayName)
	if s.createRequest.Scope != dnssdk.CreateSteeringPolicyAttachmentScopeGlobal {
		s.t.Fatalf("create scope = %q, want GLOBAL", s.createRequest.Scope)
	}
}

func TestSteeringPolicyAttachmentRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := steeringPolicyAttachmentRuntimeSemantics()
	if got == nil {
		t.Fatal("steeringPolicyAttachmentRuntimeSemantics() = nil")
	}
	if got.FormalService != "dns" || got.FormalSlug != "steeringpolicyattachment" {
		t.Fatalf("formal binding = %s/%s, want dns/steeringpolicyattachment", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	assertSteeringPolicyAttachmentStrings(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertSteeringPolicyAttachmentStrings(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertSteeringPolicyAttachmentStrings(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertSteeringPolicyAttachmentStrings(t, "List.MatchFields", got.List.MatchFields, []string{"steeringPolicyId", "zoneId", "domainName"})
	assertSteeringPolicyAttachmentStrings(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName"})
	assertSteeringPolicyAttachmentStrings(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"steeringPolicyId", "zoneId", "domainName"})
}

func TestSteeringPolicyAttachmentCreatesAndProjectsActiveStatus(t *testing.T) {
	t.Parallel()

	resource := makeSteeringPolicyAttachmentResource()
	state, client := newCreateAndProjectActiveStatusTestState(t, resource)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	state.requireCalls()
	state.requireCreateRequest()
	requireSteeringPolicyAttachmentStatus(t, resource, resource.Spec.DisplayName)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := lastSteeringPolicyAttachmentConditionType(t, resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestSteeringPolicyAttachmentBindsExistingAttachmentFromSecondPage(t *testing.T) {
	t.Parallel()

	resource := makeSteeringPolicyAttachmentResource()
	createCalled := false
	getCalls := 0
	var listPages []string

	client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
		listFn: func(_ context.Context, req dnssdk.ListSteeringPolicyAttachmentsRequest) (dnssdk.ListSteeringPolicyAttachmentsResponse, error) {
			page := ""
			if req.Page != nil {
				page = *req.Page
			}
			listPages = append(listPages, page)
			requireSteeringPolicyAttachmentStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			if page == "" {
				return dnssdk.ListSteeringPolicyAttachmentsResponse{OpcNextPage: common.String("page-2")}, nil
			}
			return dnssdk.ListSteeringPolicyAttachmentsResponse{
				Items: []dnssdk.SteeringPolicyAttachmentSummary{
					makeSDKSteeringPolicyAttachmentSummary(testSteeringPolicyAttachmentID, resource.Spec.DisplayName, dnssdk.SteeringPolicyAttachmentSummaryLifecycleStateActive),
				},
			}, nil
		},
		createFn: func(context.Context, dnssdk.CreateSteeringPolicyAttachmentRequest) (dnssdk.CreateSteeringPolicyAttachmentResponse, error) {
			createCalled = true
			return dnssdk.CreateSteeringPolicyAttachmentResponse{}, nil
		},
		getFn: func(_ context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
			getCalls++
			requireSteeringPolicyAttachmentStringPtr(t, "get attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
			return dnssdk.GetSteeringPolicyAttachmentResponse{
				SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
					testSteeringPolicyAttachmentID,
					resource.Spec.DisplayName,
					testSteeringPolicyID,
					testZoneID,
					testDomainName,
					dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind", response)
	}
	if createCalled {
		t.Fatal("CreateSteeringPolicyAttachment() called, want bind to existing attachment")
	}
	if getCalls != 1 {
		t.Fatalf("GetSteeringPolicyAttachment() calls = %d, want 1 live bind read", getCalls)
	}
	assertSteeringPolicyAttachmentStrings(t, "list pages", listPages, []string{"", "page-2"})
	requireSteeringPolicyAttachmentStatus(t, resource, resource.Spec.DisplayName)
}

func TestSteeringPolicyAttachmentNoOpReadbackDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSteeringPolicyAttachmentResource()
	updateCalled := false
	getCalls := 0

	client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
		getFn: func(_ context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
			getCalls++
			requireSteeringPolicyAttachmentStringPtr(t, "get attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
			return dnssdk.GetSteeringPolicyAttachmentResponse{
				SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
					testSteeringPolicyAttachmentID,
					resource.Spec.DisplayName,
					testSteeringPolicyID,
					testZoneID,
					testDomainName,
					dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error) {
			updateCalled = true
			return dnssdk.UpdateSteeringPolicyAttachmentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if updateCalled {
		t.Fatal("UpdateSteeringPolicyAttachment() called, want no-op readback")
	}
	if getCalls != 1 {
		t.Fatalf("GetSteeringPolicyAttachment() calls = %d, want 1", getCalls)
	}
	requireSteeringPolicyAttachmentStatus(t, resource, resource.Spec.DisplayName)
}

func TestSteeringPolicyAttachmentMutableDisplayNameUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSteeringPolicyAttachmentResource()
	resource.Spec.DisplayName = "updated attachment"
	var updateRequest dnssdk.UpdateSteeringPolicyAttachmentRequest
	getCalls := 0

	client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
		getFn: func(_ context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
			getCalls++
			requireSteeringPolicyAttachmentStringPtr(t, "get attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
			displayName := testAttachmentDisplayName
			if getCalls > 1 {
				displayName = resource.Spec.DisplayName
			}
			return dnssdk.GetSteeringPolicyAttachmentResponse{
				SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
					testSteeringPolicyAttachmentID,
					displayName,
					testSteeringPolicyID,
					testZoneID,
					testDomainName,
					dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, req dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error) {
			updateRequest = req
			return dnssdk.UpdateSteeringPolicyAttachmentResponse{
				SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
					testSteeringPolicyAttachmentID,
					resource.Spec.DisplayName,
					testSteeringPolicyID,
					testZoneID,
					testDomainName,
					dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update", response)
	}
	requireSteeringPolicyAttachmentStringPtr(t, "update attachment id", updateRequest.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
	requireSteeringPolicyAttachmentStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	if updateRequest.Scope != dnssdk.UpdateSteeringPolicyAttachmentScopeGlobal {
		t.Fatalf("update scope = %q, want GLOBAL", updateRequest.Scope)
	}
	if getCalls != 2 {
		t.Fatalf("GetSteeringPolicyAttachment() calls = %d, want pre-update and follow-up reads", getCalls)
	}
	requireSteeringPolicyAttachmentStatus(t, resource, resource.Spec.DisplayName)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestSteeringPolicyAttachmentRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(*dnsv1beta1.SteeringPolicyAttachment)
		wantErr string
	}{
		{
			name: "steering policy",
			mutate: func(resource *dnsv1beta1.SteeringPolicyAttachment) {
				resource.Spec.SteeringPolicyId = "ocid1.steeringpolicy.oc1..replacement"
			},
			wantErr: "require replacement when steeringPolicyId changes",
		},
		{
			name: "zone",
			mutate: func(resource *dnsv1beta1.SteeringPolicyAttachment) {
				resource.Spec.ZoneId = "ocid1.dns-zone.oc1..replacement"
			},
			wantErr: "require replacement when zoneId changes",
		},
		{
			name: "domain",
			mutate: func(resource *dnsv1beta1.SteeringPolicyAttachment) {
				resource.Spec.DomainName = "other.example.com"
			},
			wantErr: "require replacement when domainName changes",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedSteeringPolicyAttachmentResource()
			tc.mutate(resource)
			updateCalled := false

			client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
				getFn: func(_ context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
					requireSteeringPolicyAttachmentStringPtr(t, "get attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
					return dnssdk.GetSteeringPolicyAttachmentResponse{
						SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
							testSteeringPolicyAttachmentID,
							testAttachmentDisplayName,
							testSteeringPolicyID,
							testZoneID,
							testDomainName,
							dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
						),
					}, nil
				},
				updateFn: func(context.Context, dnssdk.UpdateSteeringPolicyAttachmentRequest) (dnssdk.UpdateSteeringPolicyAttachmentResponse, error) {
					updateCalled = true
					return dnssdk.UpdateSteeringPolicyAttachmentResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tc.wantErr)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
			}
			if updateCalled {
				t.Fatal("UpdateSteeringPolicyAttachment() called, want create-only drift rejection before update")
			}
		})
	}
}

type deleteWaitsForReadbackConfirmationTestState struct {
	t           *testing.T
	resource    *dnsv1beta1.SteeringPolicyAttachment
	getCalls    int
	deleteCalls int
}

func newDeleteWaitsForReadbackConfirmationTestState(
	t *testing.T,
	resource *dnsv1beta1.SteeringPolicyAttachment,
) (*deleteWaitsForReadbackConfirmationTestState, SteeringPolicyAttachmentServiceClient) {
	t.Helper()

	state := &deleteWaitsForReadbackConfirmationTestState{
		t:        t,
		resource: resource,
	}
	client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
		getFn:    state.getSteeringPolicyAttachment,
		deleteFn: state.deleteSteeringPolicyAttachment,
	})
	return state, client
}

func (s *deleteWaitsForReadbackConfirmationTestState) getSteeringPolicyAttachment(_ context.Context, req dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
	s.getCalls++
	requireSteeringPolicyAttachmentStringPtr(s.t, "get attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
	switch s.getCalls {
	case 1:
		return s.getResponse(dnssdk.SteeringPolicyAttachmentLifecycleStateActive), nil
	case 2:
		return s.getResponse(dnssdk.SteeringPolicyAttachmentLifecycleStateDeleting), nil
	default:
		return dnssdk.GetSteeringPolicyAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "attachment deleted")
	}
}

func (s *deleteWaitsForReadbackConfirmationTestState) getResponse(state dnssdk.SteeringPolicyAttachmentLifecycleStateEnum) dnssdk.GetSteeringPolicyAttachmentResponse {
	return dnssdk.GetSteeringPolicyAttachmentResponse{
		SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
			testSteeringPolicyAttachmentID,
			s.resource.Spec.DisplayName,
			testSteeringPolicyID,
			testZoneID,
			testDomainName,
			state,
		),
	}
}

func (s *deleteWaitsForReadbackConfirmationTestState) deleteSteeringPolicyAttachment(_ context.Context, req dnssdk.DeleteSteeringPolicyAttachmentRequest) (dnssdk.DeleteSteeringPolicyAttachmentResponse, error) {
	s.deleteCalls++
	requireSteeringPolicyAttachmentStringPtr(s.t, "delete attachment id", req.SteeringPolicyAttachmentId, testSteeringPolicyAttachmentID)
	if req.Scope != dnssdk.DeleteSteeringPolicyAttachmentScopeGlobal {
		s.t.Fatalf("delete scope = %q, want GLOBAL", req.Scope)
	}
	return dnssdk.DeleteSteeringPolicyAttachmentResponse{
		OpcRequestId:     common.String("opc-delete-1"),
		OpcWorkRequestId: common.String("wr-delete-1"),
	}, nil
}

func TestSteeringPolicyAttachmentDeleteWaitsForReadbackConfirmation(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSteeringPolicyAttachmentResource()
	state, client := newDeleteWaitsForReadbackConfirmationTestState(t, resource)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true before readback confirms absence")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set before delete confirmation")
	}
	if got := resource.Status.LifecycleState; got != string(dnssdk.SteeringPolicyAttachmentLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want true after NotFound confirmation")
	}
	if state.deleteCalls != 1 {
		t.Fatalf("DeleteSteeringPolicyAttachment() calls = %d, want one delete request", state.deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestSteeringPolicyAttachmentDeleteTreatsAuthShapedNotFoundAsFatal(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSteeringPolicyAttachmentResource()
	client := testSteeringPolicyAttachmentClient(&fakeSteeringPolicyAttachmentOCIClient{
		getFn: func(context.Context, dnssdk.GetSteeringPolicyAttachmentRequest) (dnssdk.GetSteeringPolicyAttachmentResponse, error) {
			return dnssdk.GetSteeringPolicyAttachmentResponse{
				SteeringPolicyAttachment: makeSDKSteeringPolicyAttachment(
					testSteeringPolicyAttachmentID,
					resource.Spec.DisplayName,
					testSteeringPolicyID,
					testZoneID,
					testDomainName,
					dnssdk.SteeringPolicyAttachmentLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, dnssdk.DeleteSteeringPolicyAttachmentRequest) (dnssdk.DeleteSteeringPolicyAttachmentResponse, error) {
			return dnssdk.DeleteSteeringPolicyAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for auth-shaped 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func requireSteeringPolicyAttachmentStatus(t *testing.T, resource *dnsv1beta1.SteeringPolicyAttachment, displayName string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testSteeringPolicyAttachmentID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSteeringPolicyAttachmentID)
	}
	if got := resource.Status.Id; got != testSteeringPolicyAttachmentID {
		t.Fatalf("status.id = %q, want %q", got, testSteeringPolicyAttachmentID)
	}
	if got := resource.Status.SteeringPolicyId; got != testSteeringPolicyID {
		t.Fatalf("status.steeringPolicyId = %q, want %q", got, testSteeringPolicyID)
	}
	if got := resource.Status.ZoneId; got != testZoneID {
		t.Fatalf("status.zoneId = %q, want %q", got, testZoneID)
	}
	if got := resource.Status.DomainName; got != testDomainName {
		t.Fatalf("status.domainName = %q, want %q", got, testDomainName)
	}
	if got := resource.Status.DisplayName; got != displayName {
		t.Fatalf("status.displayName = %q, want %q", got, displayName)
	}
	if got := resource.Status.CompartmentId; got != testCompartmentID {
		t.Fatalf("status.compartmentId = %q, want %q", got, testCompartmentID)
	}
	if got := resource.Status.LifecycleState; got != string(dnssdk.SteeringPolicyAttachmentLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func requireSteeringPolicyAttachmentStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertSteeringPolicyAttachmentStrings(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d (%v)", label, len(got), len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}

func lastSteeringPolicyAttachmentConditionType(t *testing.T, resource *dnsv1beta1.SteeringPolicyAttachment) shared.OSOKConditionType {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.status.conditions = nil, want at least one condition")
	}
	return conditions[len(conditions)-1].Type
}
