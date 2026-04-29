/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apmdomain

import (
	"context"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	apmcontrolplanesdk "github.com/oracle/oci-go-sdk/v65/apmcontrolplane"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmcontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/apmcontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeApmDomainOCIClient struct {
	createFn      func(context.Context, apmcontrolplanesdk.CreateApmDomainRequest) (apmcontrolplanesdk.CreateApmDomainResponse, error)
	getFn         func(context.Context, apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error)
	listFn        func(context.Context, apmcontrolplanesdk.ListApmDomainsRequest) (apmcontrolplanesdk.ListApmDomainsResponse, error)
	updateFn      func(context.Context, apmcontrolplanesdk.UpdateApmDomainRequest) (apmcontrolplanesdk.UpdateApmDomainResponse, error)
	deleteFn      func(context.Context, apmcontrolplanesdk.DeleteApmDomainRequest) (apmcontrolplanesdk.DeleteApmDomainResponse, error)
	workRequestFn func(context.Context, apmcontrolplanesdk.GetWorkRequestRequest) (apmcontrolplanesdk.GetWorkRequestResponse, error)
}

func (f *fakeApmDomainOCIClient) CreateApmDomain(
	ctx context.Context,
	req apmcontrolplanesdk.CreateApmDomainRequest,
) (apmcontrolplanesdk.CreateApmDomainResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return apmcontrolplanesdk.CreateApmDomainResponse{}, nil
}

func (f *fakeApmDomainOCIClient) GetApmDomain(
	ctx context.Context,
	req apmcontrolplanesdk.GetApmDomainRequest,
) (apmcontrolplanesdk.GetApmDomainResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return apmcontrolplanesdk.GetApmDomainResponse{}, nil
}

func (f *fakeApmDomainOCIClient) ListApmDomains(
	ctx context.Context,
	req apmcontrolplanesdk.ListApmDomainsRequest,
) (apmcontrolplanesdk.ListApmDomainsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return apmcontrolplanesdk.ListApmDomainsResponse{}, nil
}

func (f *fakeApmDomainOCIClient) UpdateApmDomain(
	ctx context.Context,
	req apmcontrolplanesdk.UpdateApmDomainRequest,
) (apmcontrolplanesdk.UpdateApmDomainResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return apmcontrolplanesdk.UpdateApmDomainResponse{}, nil
}

func (f *fakeApmDomainOCIClient) DeleteApmDomain(
	ctx context.Context,
	req apmcontrolplanesdk.DeleteApmDomainRequest,
) (apmcontrolplanesdk.DeleteApmDomainResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return apmcontrolplanesdk.DeleteApmDomainResponse{}, nil
}

func (f *fakeApmDomainOCIClient) GetWorkRequest(
	ctx context.Context,
	req apmcontrolplanesdk.GetWorkRequestRequest,
) (apmcontrolplanesdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return apmcontrolplanesdk.GetWorkRequestResponse{}, nil
}

type apmDomainRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func TestReviewedApmDomainRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedApmDomainRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedApmDomainRuntimeSemantics() = nil")
	}

	if got.FormalService != "apmcontrolplane" {
		t.Fatalf("FormalService = %q, want apmcontrolplane", got.FormalService)
	}
	if got.FormalSlug != "apmdomain" {
		t.Fatalf("FormalSlug = %q, want apmdomain", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertApmDomainStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertApmDomainStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertApmDomainStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertApmDomainStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertApmDomainStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertApmDomainStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertApmDomainStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertApmDomainStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags"})
	assertApmDomainStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "isFreeTier"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetApmDomain" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetApmDomain", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetApmDomain" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetApmDomain", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetApmDomain/ListApmDomains confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardApmDomainExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeApmDomainResource()
	resource.Spec.DisplayName = ""

	decision, err := guardApmDomainExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardApmDomainExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardApmDomainExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "apm-domain"
	decision, err = guardApmDomainExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardApmDomainExistingBeforeCreate(non-empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardApmDomainExistingBeforeCreate(non-empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildApmDomainUpdateBodyPreservesClears(t *testing.T) {
	t.Parallel()

	currentResource := makeApmDomainResource()
	desired := makeApmDomainResource()
	desired.Spec.DisplayName = "apm-domain-updated"
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildApmDomainUpdateBody(
		desired,
		apmcontrolplanesdk.GetApmDomainResponse{
			ApmDomain: makeSDKApmDomain(
				"ocid1.apmdomain.oc1..existing",
				currentResource,
				apmcontrolplanesdk.LifecycleStatesActive,
			),
		},
	)
	if err != nil {
		t.Fatalf("buildApmDomainUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildApmDomainUpdateBody() updateNeeded = false, want true")
	}

	requireApmDomainStringPtr(t, "details.displayName", body.DisplayName, desired.Spec.DisplayName)
	requireApmDomainStringPtr(t, "details.description", body.Description, "")
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}

	requestBody := apmDomainSerializedRequestBody(t, apmcontrolplanesdk.UpdateApmDomainRequest{
		ApmDomainId:            common.String("ocid1.apmdomain.oc1..existing"),
		UpdateApmDomainDetails: body,
	}, http.MethodPut, "/apmDomains/ocid1.apmdomain.oc1..existing")
	for _, want := range []string{
		`"displayName":"apm-domain-updated"`,
		`"description":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(requestBody, want) {
			t.Fatalf("request body %s does not contain %s", requestBody, want)
		}
	}
}

func TestApmDomainCreateOrUpdateRejectsAmbiguousDisplayNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeApmDomainResource()
	createCalls := 0

	client := newTestApmDomainClient(&fakeApmDomainOCIClient{
		listFn: func(_ context.Context, req apmcontrolplanesdk.ListApmDomainsRequest) (apmcontrolplanesdk.ListApmDomainsResponse, error) {
			requireApmDomainStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireApmDomainStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.LifecycleState != "" {
				t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", req.LifecycleState)
			}
			return apmcontrolplanesdk.ListApmDomainsResponse{
				Items: []apmcontrolplanesdk.ApmDomainSummary{
					makeSDKApmDomainSummary("ocid1.apmdomain.oc1..first", resource, apmcontrolplanesdk.LifecycleStatesActive),
					makeSDKApmDomainSummary("ocid1.apmdomain.oc1..second", resource, apmcontrolplanesdk.LifecycleStatesUpdating),
				},
			}, nil
		},
		createFn: func(_ context.Context, _ apmcontrolplanesdk.CreateApmDomainRequest) (apmcontrolplanesdk.CreateApmDomainResponse, error) {
			createCalls++
			return apmcontrolplanesdk.CreateApmDomainResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous list match failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful result", response)
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match failure", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreateApmDomain() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestApmDomainCreateOrUpdateRejectsUnsupportedIsFreeTierDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.apmdomain.oc1..existing"

	resource := newExistingApmDomainResource(existingID)
	resource.Spec.IsFreeTier = false

	current := makeSDKApmDomain(existingID, makeApmDomainResource(), apmcontrolplanesdk.LifecycleStatesActive)

	client := newTestApmDomainClient(&fakeApmDomainOCIClient{
		getFn: func(_ context.Context, _ apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error) {
			return apmcontrolplanesdk.GetApmDomainResponse{ApmDomain: current}, nil
		},
		updateFn: func(_ context.Context, _ apmcontrolplanesdk.UpdateApmDomainRequest) (apmcontrolplanesdk.UpdateApmDomainResponse, error) {
			t.Fatal("UpdateApmDomain() should not be called for unsupported isFreeTier drift")
			return apmcontrolplanesdk.UpdateApmDomainResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "replacement when isFreeTier changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported isFreeTier drift failure", err)
	}
}

func TestApmDomainCreateOrUpdateProjectsStatusFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.apmdomain.oc1..existing"

	resource := newExistingApmDomainResource(existingID)
	updateCalled := false

	client := newTestApmDomainClient(&fakeApmDomainOCIClient{
		getFn: func(_ context.Context, req apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error) {
			requireApmDomainStringPtr(t, "get apmDomainId", req.ApmDomainId, existingID)
			return apmcontrolplanesdk.GetApmDomainResponse{
				ApmDomain: makeSDKApmDomain(existingID, resource, apmcontrolplanesdk.LifecycleStatesActive),
			}, nil
		},
		updateFn: func(_ context.Context, _ apmcontrolplanesdk.UpdateApmDomainRequest) (apmcontrolplanesdk.UpdateApmDomainResponse, error) {
			updateCalled = true
			t.Fatal("UpdateApmDomain() should not be called when status already matches")
			return apmcontrolplanesdk.UpdateApmDomainResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for steady ACTIVE state")
	}
	if updateCalled {
		t.Fatal("UpdateApmDomain() was called unexpectedly")
	}
	if resource.Status.DataUploadEndpoint != "https://upload.apm.example.com" {
		t.Fatalf("status.dataUploadEndpoint = %q, want upload endpoint", resource.Status.DataUploadEndpoint)
	}
	if !resource.Status.IsFreeTier {
		t.Fatal("status.isFreeTier = false, want true")
	}
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
	if resource.Status.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", resource.Status.CompartmentId, resource.Spec.CompartmentId)
	}
}

func TestApmDomainServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.apmdomain.oc1..created"
		workRequestID = "wr-apmdomain-create"
	)

	resource := makeApmDomainResource()
	workRequests := map[string]apmcontrolplanesdk.WorkRequest{
		workRequestID: makeApmDomainWorkRequest(
			workRequestID,
			apmcontrolplanesdk.OperationTypesCreateApmDomain,
			apmcontrolplanesdk.OperationStatusInProgress,
			apmcontrolplanesdk.ActionTypesInProgress,
			"",
		),
	}

	var createRequest apmcontrolplanesdk.CreateApmDomainRequest
	var listRequest apmcontrolplanesdk.ListApmDomainsRequest
	getCalls := 0

	client := newTestApmDomainClient(&fakeApmDomainOCIClient{
		listFn: func(_ context.Context, req apmcontrolplanesdk.ListApmDomainsRequest) (apmcontrolplanesdk.ListApmDomainsResponse, error) {
			listRequest = req
			return apmcontrolplanesdk.ListApmDomainsResponse{}, nil
		},
		createFn: func(_ context.Context, req apmcontrolplanesdk.CreateApmDomainRequest) (apmcontrolplanesdk.CreateApmDomainResponse, error) {
			createRequest = req
			return apmcontrolplanesdk.CreateApmDomainResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-apmdomain"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req apmcontrolplanesdk.GetWorkRequestRequest) (apmcontrolplanesdk.GetWorkRequestResponse, error) {
			requireApmDomainStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return apmcontrolplanesdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error) {
			getCalls++
			requireApmDomainStringPtr(t, "get apmDomainId", req.ApmDomainId, createdID)
			return apmcontrolplanesdk.GetApmDomainResponse{
				ApmDomain: makeSDKApmDomain(createdID, resource, apmcontrolplanesdk.LifecycleStatesActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	requireApmDomainStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireApmDomainStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if listRequest.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", listRequest.LifecycleState)
	}
	requireApmDomainStringPtr(t, "create compartmentId", createRequest.CreateApmDomainDetails.CompartmentId, resource.Spec.CompartmentId)
	requireApmDomainStringPtr(t, "create displayName", createRequest.CreateApmDomainDetails.DisplayName, resource.Spec.DisplayName)
	requireApmDomainBoolPtr(t, "create isFreeTier", createRequest.CreateApmDomainDetails.IsFreeTier, true)
	if getCalls != 0 {
		t.Fatalf("GetApmDomain() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireApmDomainAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-apmdomain" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-apmdomain", got)
	}

	workRequests[workRequestID] = makeApmDomainWorkRequest(
		workRequestID,
		apmcontrolplanesdk.OperationTypesCreateApmDomain,
		apmcontrolplanesdk.OperationStatusSucceeded,
		apmcontrolplanesdk.ActionTypesCreated,
		createdID,
	)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetApmDomain() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(apmcontrolplanesdk.LifecycleStatesActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func TestApmDomainDeleteStartsWorkRequestAndWaitsForConfirmation(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.apmdomain.oc1..existing"
		workRequestID = "wr-apmdomain-delete"
	)

	resource := newExistingApmDomainResource(existingID)
	getCalls := 0
	var deleteRequest apmcontrolplanesdk.DeleteApmDomainRequest

	client := newTestApmDomainClient(&fakeApmDomainOCIClient{
		getFn: func(_ context.Context, req apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error) {
			getCalls++
			requireApmDomainStringPtr(t, "get apmDomainId", req.ApmDomainId, existingID)
			return apmcontrolplanesdk.GetApmDomainResponse{
				ApmDomain: makeSDKApmDomain(existingID, resource, apmcontrolplanesdk.LifecycleStatesActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req apmcontrolplanesdk.DeleteApmDomainRequest) (apmcontrolplanesdk.DeleteApmDomainResponse, error) {
			deleteRequest = req
			return apmcontrolplanesdk.DeleteApmDomainResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-delete-apmdomain"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req apmcontrolplanesdk.GetWorkRequestRequest) (apmcontrolplanesdk.GetWorkRequestResponse, error) {
			requireApmDomainStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return apmcontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeApmDomainWorkRequest(
					workRequestID,
					apmcontrolplanesdk.OperationTypesDeleteApmDomain,
					apmcontrolplanesdk.OperationStatusInProgress,
					apmcontrolplanesdk.ActionTypesInProgress,
					existingID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending delete confirmation")
	}
	requireApmDomainStringPtr(t, "delete apmDomainId", deleteRequest.ApmDomainId, existingID)
	requireApmDomainAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-apmdomain" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-apmdomain", got)
	}
	if getCalls != 0 {
		t.Fatalf("GetApmDomain() calls = %d, want 0 when the tracked OCID is already available", getCalls)
	}
}

func newTestApmDomainClient(client *fakeApmDomainOCIClient) ApmDomainServiceClient {
	if client == nil {
		client = &fakeApmDomainOCIClient{}
	}
	return newApmDomainServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
	)
}

func makeApmDomainResource() *apmcontrolplanev1beta1.ApmDomain {
	return &apmcontrolplanev1beta1.ApmDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apmdomain-sample",
			Namespace: "default",
		},
		Spec: apmcontrolplanev1beta1.ApmDomainSpec{
			DisplayName:   "apm-domain",
			CompartmentId: "ocid1.compartment.oc1..apmdomainexample",
			Description:   "apm domain description",
			FreeformTags: map[string]string{
				"environment": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			IsFreeTier: true,
		},
	}
}

func newExistingApmDomainResource(existingID string) *apmcontrolplanev1beta1.ApmDomain {
	resource := makeApmDomainResource()
	resource.Status = apmcontrolplanev1beta1.ApmDomainStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func makeSDKApmDomain(
	id string,
	resource *apmcontrolplanev1beta1.ApmDomain,
	state apmcontrolplanesdk.LifecycleStatesEnum,
) apmcontrolplanesdk.ApmDomain {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return apmcontrolplanesdk.ApmDomain{
		Id:                 common.String(id),
		DisplayName:        common.String(resource.Spec.DisplayName),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		Description:        common.String(resource.Spec.Description),
		LifecycleState:     state,
		IsFreeTier:         common.Bool(resource.Spec.IsFreeTier),
		TimeCreated:        &now,
		TimeUpdated:        &now,
		FreeformTags:       maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:        sdkApmDomainDefinedTags(resource.Spec.DefinedTags),
		DataUploadEndpoint: common.String("https://upload.apm.example.com"),
	}
}

func makeSDKApmDomainSummary(
	id string,
	resource *apmcontrolplanev1beta1.ApmDomain,
	state apmcontrolplanesdk.LifecycleStatesEnum,
) apmcontrolplanesdk.ApmDomainSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return apmcontrolplanesdk.ApmDomainSummary{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		Description:    common.String(resource.Spec.Description),
		LifecycleState: state,
		IsFreeTier:     common.Bool(resource.Spec.IsFreeTier),
		TimeCreated:    &now,
		TimeUpdated:    &now,
		FreeformTags:   maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:    sdkApmDomainDefinedTags(resource.Spec.DefinedTags),
	}
}

func makeApmDomainWorkRequest(
	id string,
	operation apmcontrolplanesdk.OperationTypesEnum,
	status apmcontrolplanesdk.OperationStatusEnum,
	action apmcontrolplanesdk.ActionTypesEnum,
	resourceID string,
) apmcontrolplanesdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return apmcontrolplanesdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..apmdomainexample"),
		Resources:       []apmcontrolplanesdk.WorkRequestResource{{EntityType: common.String("ApmDomain"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func sdkApmDomainDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		out[namespace] = converted
	}
	return out
}

func apmDomainSerializedRequestBody(
	t *testing.T,
	request apmDomainRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request body) error = %v", err)
	}
	return string(body)
}

func assertApmDomainStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireApmDomainStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireApmDomainBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %t", name, got, want)
	}
}

func requireApmDomainAsyncCurrent(
	t *testing.T,
	resource *apmcontrolplanev1beta1.ApmDomain,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}
