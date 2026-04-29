/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package governanceinstance

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	accessgovernancecpsdk "github.com/oracle/oci-go-sdk/v65/accessgovernancecp"
	"github.com/oracle/oci-go-sdk/v65/common"
	accessgovernancecpv1beta1 "github.com/oracle/oci-service-operator/api/accessgovernancecp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeGovernanceInstanceOCIClient struct {
	createFn func(context.Context, accessgovernancecpsdk.CreateGovernanceInstanceRequest) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error)
	getFn    func(context.Context, accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error)
	listFn   func(context.Context, accessgovernancecpsdk.ListGovernanceInstancesRequest) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error)
	updateFn func(context.Context, accessgovernancecpsdk.UpdateGovernanceInstanceRequest) (accessgovernancecpsdk.UpdateGovernanceInstanceResponse, error)
	deleteFn func(context.Context, accessgovernancecpsdk.DeleteGovernanceInstanceRequest) (accessgovernancecpsdk.DeleteGovernanceInstanceResponse, error)
}

func (f *fakeGovernanceInstanceOCIClient) CreateGovernanceInstance(
	ctx context.Context,
	req accessgovernancecpsdk.CreateGovernanceInstanceRequest,
) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return accessgovernancecpsdk.CreateGovernanceInstanceResponse{}, nil
}

func (f *fakeGovernanceInstanceOCIClient) GetGovernanceInstance(
	ctx context.Context,
	req accessgovernancecpsdk.GetGovernanceInstanceRequest,
) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return accessgovernancecpsdk.GetGovernanceInstanceResponse{}, errortest.NewServiceError(404, "NotFound", "missing GovernanceInstance")
}

func (f *fakeGovernanceInstanceOCIClient) ListGovernanceInstances(
	ctx context.Context,
	req accessgovernancecpsdk.ListGovernanceInstancesRequest,
) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return accessgovernancecpsdk.ListGovernanceInstancesResponse{}, nil
}

func (f *fakeGovernanceInstanceOCIClient) UpdateGovernanceInstance(
	ctx context.Context,
	req accessgovernancecpsdk.UpdateGovernanceInstanceRequest,
) (accessgovernancecpsdk.UpdateGovernanceInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return accessgovernancecpsdk.UpdateGovernanceInstanceResponse{}, nil
}

func (f *fakeGovernanceInstanceOCIClient) DeleteGovernanceInstance(
	ctx context.Context,
	req accessgovernancecpsdk.DeleteGovernanceInstanceRequest,
) (accessgovernancecpsdk.DeleteGovernanceInstanceResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return accessgovernancecpsdk.DeleteGovernanceInstanceResponse{}, nil
}

func TestReviewedGovernanceInstanceRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := reviewedGovernanceInstanceRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedGovernanceInstanceRuntimeSemantics() = nil")
	}

	if got.FormalService != "accessgovernancecp" {
		t.Fatalf("FormalService = %q, want accessgovernancecp", got.FormalService)
	}
	if got.FormalSlug != "governanceinstance" {
		t.Fatalf("FormalSlug = %q, want governanceinstance", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want lifecycle semantics")
	}
	if got.Async.Strategy != "lifecycle" {
		t.Fatalf("Async.Strategy = %q, want lifecycle", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.FormalClassification != "lifecycle" {
		t.Fatalf("Async.FormalClassification = %q, want lifecycle", got.Async.FormalClassification)
	}
	if !slices.Equal(got.Lifecycle.ProvisioningStates, []string{"CREATING"}) {
		t.Fatalf("Lifecycle.ProvisioningStates = %#v, want [CREATING]", got.Lifecycle.ProvisioningStates)
	}
	if len(got.Lifecycle.UpdatingStates) != 0 {
		t.Fatalf("Lifecycle.UpdatingStates = %#v, want none", got.Lifecycle.UpdatingStates)
	}
	if !slices.Equal(got.Lifecycle.ActiveStates, []string{"ACTIVE"}) {
		t.Fatalf("Lifecycle.ActiveStates = %#v, want [ACTIVE]", got.Lifecycle.ActiveStates)
	}
	if !slices.Equal(got.Delete.PendingStates, []string{"DELETING"}) {
		t.Fatalf("Delete.PendingStates = %#v, want [DELETING]", got.Delete.PendingStates)
	}
	if !slices.Equal(got.Delete.TerminalStates, []string{"DELETED"}) {
		t.Fatalf("Delete.TerminalStates = %#v, want [DELETED]", got.Delete.TerminalStates)
	}
	if got.List == nil {
		t.Fatal("List = nil, want explicit match fields")
	}
	if !slices.Equal(got.List.MatchFields, []string{"compartmentId", "displayName", "licenseType", "tenancyNamespace", "id"}) {
		t.Fatalf("List.MatchFields = %#v, want exact reviewed lookup keys", got.List.MatchFields)
	}
	if !slices.Equal(got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags", "licenseType"}) {
		t.Fatalf("Mutation.Mutable = %#v, want reviewed mutable fields", got.Mutation.Mutable)
	}
	if !slices.Equal(got.Mutation.ForceNew, []string{"compartmentId", "idcsAccessToken", "systemTags", "tenancyNamespace"}) {
		t.Fatalf("Mutation.ForceNew = %#v, want reviewed force-new fields", got.Mutation.ForceNew)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardGovernanceInstanceExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := newGovernanceInstanceTestResource()
	resource.Spec.TenancyNamespace = ""

	decision, err := guardGovernanceInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGovernanceInstanceExistingBeforeCreate(empty tenancyNamespace) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardGovernanceInstanceExistingBeforeCreate(empty tenancyNamespace) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.TenancyNamespace = "tenant-namespace"
	decision, err = guardGovernanceInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGovernanceInstanceExistingBeforeCreate(complete identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardGovernanceInstanceExistingBeforeCreate(complete identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildGovernanceInstanceUpdateBodySupportsClearsAndRename(t *testing.T) {
	t.Parallel()

	currentResource := newGovernanceInstanceTestResource()
	desired := newGovernanceInstanceTestResource()
	desired.Spec.DisplayName = "governance-updated"
	desired.Spec.Description = ""
	desired.Spec.LicenseType = string(accessgovernancecpsdk.LicenseTypeAgOci)
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildGovernanceInstanceUpdateBody(
		desired,
		accessgovernancecpsdk.GetGovernanceInstanceResponse{
			GovernanceInstance: observedGovernanceInstanceFromSpec(
				"ocid1.governanceinstance.oc1..existing",
				currentResource.Spec,
				"ACTIVE",
			),
		},
	)
	if err != nil {
		t.Fatalf("buildGovernanceInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildGovernanceInstanceUpdateBody() updateNeeded = false, want true")
	}

	requireGovernanceInstanceStringPtr(t, "details.displayName", body.DisplayName, desired.Spec.DisplayName)
	requireGovernanceInstanceStringPtr(t, "details.description", body.Description, "")
	if body.LicenseType != accessgovernancecpsdk.LicenseTypeAgOci {
		t.Fatalf("details.LicenseType = %q, want %q", body.LicenseType, accessgovernancecpsdk.LicenseTypeAgOci)
	}
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map clear", body.DefinedTags)
	}
}

func TestGovernanceInstanceCreateOrUpdateSkipsReuseWhenLookupCriteriaMissing(t *testing.T) {
	t.Parallel()

	resource := newGovernanceInstanceTestResource()
	resource.Spec.TenancyNamespace = ""

	listCalls := 0
	createCalls := 0

	client := newTestGovernanceInstanceClient(&fakeGovernanceInstanceOCIClient{
		listFn: func(_ context.Context, _ accessgovernancecpsdk.ListGovernanceInstancesRequest) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error) {
			listCalls++
			return accessgovernancecpsdk.ListGovernanceInstancesResponse{}, nil
		},
		getFn: func(_ context.Context, req accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
			requireGovernanceInstanceStringPtr(t, "get governanceInstanceId", req.GovernanceInstanceId, "ocid1.governanceinstance.oc1..created")
			return accessgovernancecpsdk.GetGovernanceInstanceResponse{
				GovernanceInstance: observedGovernanceInstanceFromSpec(
					"ocid1.governanceinstance.oc1..created",
					resource.Spec,
					"ACTIVE",
				),
			}, nil
		},
		createFn: func(_ context.Context, req accessgovernancecpsdk.CreateGovernanceInstanceRequest) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error) {
			createCalls++
			requireGovernanceInstanceStringPtr(t, "create compartmentId", req.CreateGovernanceInstanceDetails.CompartmentId, resource.Spec.CompartmentId)
			return accessgovernancecpsdk.CreateGovernanceInstanceResponse{
				GovernanceInstance: observedGovernanceInstanceFromSpec(
					"ocid1.governanceinstance.oc1..created",
					resource.Spec,
					"ACTIVE",
				),
				OpcRequestId: common.String("opc-create-missing-lookup"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful settled create", response)
	}
	if listCalls != 0 {
		t.Fatalf("ListGovernanceInstances() calls = %d, want 0 when pre-create lookup criteria are incomplete", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateGovernanceInstance() calls = %d, want 1", createCalls)
	}
}

func TestGovernanceInstanceCreateOrUpdateReusesUniqueTenantScopedInstance(t *testing.T) {
	t.Parallel()

	const wrongID = "ocid1.governanceinstance.oc1..wrong"
	const rightID = "ocid1.governanceinstance.oc1..right"

	resource := newGovernanceInstanceTestResource()
	createCalled := false

	client := newTestGovernanceInstanceClient(&fakeGovernanceInstanceOCIClient{
		listFn: func(_ context.Context, req accessgovernancecpsdk.ListGovernanceInstancesRequest) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error) {
			requireGovernanceInstanceStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireGovernanceInstanceStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return accessgovernancecpsdk.ListGovernanceInstancesResponse{
				GovernanceInstanceCollection: accessgovernancecpsdk.GovernanceInstanceCollection{
					Items: []accessgovernancecpsdk.GovernanceInstanceSummary{
						observedGovernanceInstanceSummaryFromSpec(wrongID, resource.Spec, "ACTIVE"),
						observedGovernanceInstanceSummaryFromSpec(rightID, resource.Spec, "ACTIVE"),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
			resourceID := ""
			if req.GovernanceInstanceId != nil {
				resourceID = *req.GovernanceInstanceId
			}
			switch resourceID {
			case wrongID:
				wrongSpec := resource.Spec
				wrongSpec.TenancyNamespace = "other-namespace"
				return accessgovernancecpsdk.GetGovernanceInstanceResponse{
					GovernanceInstance: observedGovernanceInstanceFromSpec(wrongID, wrongSpec, "ACTIVE"),
				}, nil
			case rightID:
				return accessgovernancecpsdk.GetGovernanceInstanceResponse{
					GovernanceInstance: observedGovernanceInstanceFromSpec(rightID, resource.Spec, "ACTIVE"),
				}, nil
			default:
				t.Fatalf("unexpected GetGovernanceInstanceRequest.GovernanceInstanceId = %q", resourceID)
				return accessgovernancecpsdk.GetGovernanceInstanceResponse{}, nil
			}
		},
		createFn: func(_ context.Context, _ accessgovernancecpsdk.CreateGovernanceInstanceRequest) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error) {
			createCalled = true
			t.Fatal("CreateGovernanceInstance() should not be called when a unique tenant-scoped match exists")
			return accessgovernancecpsdk.CreateGovernanceInstanceResponse{}, nil
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
		t.Fatal("CreateOrUpdate() should not requeue for reusable ACTIVE GovernanceInstance")
	}
	if createCalled {
		t.Fatal("CreateGovernanceInstance() was called unexpectedly")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != rightID {
		t.Fatalf("status.ocid = %q, want %q", got, rightID)
	}
}

func TestGovernanceInstanceCreateOrUpdateIgnoresUnobservedIdcsAccessTokenAfterCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.governanceinstance.oc1..existing"

	resource := newExistingGovernanceInstanceTestResource(existingID)
	resource.Spec.IdcsAccessToken = "rotated-access-token"
	updateCalled := false

	client := newTestGovernanceInstanceClient(&fakeGovernanceInstanceOCIClient{
		getFn: func(_ context.Context, req accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
			requireGovernanceInstanceStringPtr(t, "get governanceInstanceId", req.GovernanceInstanceId, existingID)
			return accessgovernancecpsdk.GetGovernanceInstanceResponse{
				GovernanceInstance: observedGovernanceInstanceFromSpec(
					existingID,
					newGovernanceInstanceTestResource().Spec,
					"ACTIVE",
				),
			}, nil
		},
		updateFn: func(_ context.Context, _ accessgovernancecpsdk.UpdateGovernanceInstanceRequest) (accessgovernancecpsdk.UpdateGovernanceInstanceResponse, error) {
			updateCalled = true
			t.Fatal("UpdateGovernanceInstance() should not be called for create-only idcsAccessToken drift")
			return accessgovernancecpsdk.UpdateGovernanceInstanceResponse{}, nil
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
		t.Fatal("UpdateGovernanceInstance() was called unexpectedly")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
}

func TestGovernanceInstanceCreatePendingProjectsSharedAsyncBreadcrumbs(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.governanceinstance.oc1..created"

	resource := newGovernanceInstanceTestResource()

	client := newTestGovernanceInstanceClient(&fakeGovernanceInstanceOCIClient{
		createFn: func(_ context.Context, _ accessgovernancecpsdk.CreateGovernanceInstanceRequest) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error) {
			return accessgovernancecpsdk.CreateGovernanceInstanceResponse{
				GovernanceInstance: observedGovernanceInstanceFromSpec(createdID, resource.Spec, "CREATING"),
				OpcRequestId:       common.String("opc-create-1"),
				OpcWorkRequestId:   common.String("wr-create-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create response stays CREATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should keep requeueing while create response stays CREATING")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Provisioning) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Provisioning)
	}
	requireGovernanceInstanceOpcRequestID(t, resource, "opc-create-1")
	requireGovernanceInstanceAsyncCurrent(
		t,
		resource,
		shared.OSOKAsyncPhaseCreate,
		"CREATING",
		shared.OSOKAsyncClassPending,
		"wr-create-1",
	)
}

func TestGovernanceInstanceCreateOrUpdateClassifiesReviewedLifecycleStates(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.governanceinstance.oc1..existing"

	testCases := []struct {
		name           string
		lifecycle      string
		wantSuccessful bool
		wantReason     string
		wantRequeue    bool
	}{
		{
			name:           "active settles success",
			lifecycle:      "ACTIVE",
			wantSuccessful: true,
			wantReason:     string(shared.Active),
			wantRequeue:    false,
		},
		{
			name:           "needs attention is terminal failure",
			lifecycle:      "NEEDS_ATTENTION",
			wantSuccessful: false,
			wantReason:     string(shared.Failed),
			wantRequeue:    false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := newExistingGovernanceInstanceTestResource(existingID)

			client := newTestGovernanceInstanceClient(&fakeGovernanceInstanceOCIClient{
				getFn: func(_ context.Context, req accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
					requireGovernanceInstanceStringPtr(t, "get governanceInstanceId", req.GovernanceInstanceId, existingID)
					return accessgovernancecpsdk.GetGovernanceInstanceResponse{
						GovernanceInstance: observedGovernanceInstanceFromSpec(existingID, resource.Spec, tc.lifecycle),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccessful)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if resource.Status.OsokStatus.Reason != tc.wantReason {
				t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, tc.wantReason)
			}
			if resource.Status.LifecycleState != tc.lifecycle {
				t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.lifecycle)
			}
			if resource.Status.OsokStatus.Async.Current != nil {
				t.Fatalf("status.async.current = %#v, want nil for non-pending lifecycle state", resource.Status.OsokStatus.Async.Current)
			}
		})
	}
}

func TestGovernanceInstanceDeletePendingProjectsSharedAsyncBreadcrumbs(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.governanceinstance.oc1..existing"

	resource := newExistingGovernanceInstanceTestResource(existingID)
	getCalls := 0

	client := newTestGovernanceInstanceClient(&fakeGovernanceInstanceOCIClient{
		getFn: func(_ context.Context, _ accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
			getCalls++
			lifecycle := "ACTIVE"
			if getCalls > 1 {
				lifecycle = "DELETING"
			}
			return accessgovernancecpsdk.GetGovernanceInstanceResponse{
				GovernanceInstance: observedGovernanceInstanceFromSpec(existingID, resource.Spec, lifecycle),
			}, nil
		},
		deleteFn: func(_ context.Context, req accessgovernancecpsdk.DeleteGovernanceInstanceRequest) (accessgovernancecpsdk.DeleteGovernanceInstanceResponse, error) {
			requireGovernanceInstanceStringPtr(t, "delete governanceInstanceId", req.GovernanceInstanceId, existingID)
			return accessgovernancecpsdk.DeleteGovernanceInstanceResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending delete confirmation while lifecycle is DELETING")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "DELETING")
	}
	requireGovernanceInstanceOpcRequestID(t, resource, "opc-delete-1")
	requireGovernanceInstanceAsyncCurrent(
		t,
		resource,
		shared.OSOKAsyncPhaseDelete,
		"",
		shared.OSOKAsyncClassPending,
		"wr-delete-1",
	)
}

func newTestGovernanceInstanceClient(client governanceInstanceOCIClient) GovernanceInstanceServiceClient {
	return newGovernanceInstanceServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func newGovernanceInstanceTestResource() *accessgovernancecpv1beta1.GovernanceInstance {
	return &accessgovernancecpv1beta1.GovernanceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "governanceinstance-sample",
			Namespace: "default",
		},
		Spec: accessgovernancecpv1beta1.GovernanceInstanceSpec{
			DisplayName:      "governance-sample",
			LicenseType:      string(accessgovernancecpsdk.LicenseTypeNewLicense),
			TenancyNamespace: "tenant-namespace",
			CompartmentId:    "ocid1.compartment.oc1..exampleuniqueID",
			IdcsAccessToken:  "opaque-access-token",
			Description:      "governance description",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"costCenter": "42",
				},
			},
		},
	}
}

func newExistingGovernanceInstanceTestResource(existingID string) *accessgovernancecpv1beta1.GovernanceInstance {
	resource := newGovernanceInstanceTestResource()
	resource.Status = accessgovernancecpv1beta1.GovernanceInstanceStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedGovernanceInstanceFromSpec(
	id string,
	spec accessgovernancecpv1beta1.GovernanceInstanceSpec,
	lifecycle string,
) accessgovernancecpsdk.GovernanceInstance {
	return accessgovernancecpsdk.GovernanceInstance{
		Id:               common.String(id),
		DisplayName:      common.String(spec.DisplayName),
		CompartmentId:    common.String(spec.CompartmentId),
		Description:      common.String(spec.Description),
		LicenseType:      accessgovernancecpsdk.LicenseTypeEnum(spec.LicenseType),
		TenancyNamespace: common.String(spec.TenancyNamespace),
		InstanceUrl:      common.String("https://access-governance.example"),
		LifecycleState:   accessgovernancecpsdk.InstanceLifecycleStateEnum(lifecycle),
		DefinedTags:      governanceInstanceDefinedTagsFromSpec(spec.DefinedTags),
		FreeformTags:     maps.Clone(spec.FreeformTags),
	}
}

func observedGovernanceInstanceSummaryFromSpec(
	id string,
	spec accessgovernancecpv1beta1.GovernanceInstanceSpec,
	lifecycle string,
) accessgovernancecpsdk.GovernanceInstanceSummary {
	return accessgovernancecpsdk.GovernanceInstanceSummary{
		Id:             common.String(id),
		DisplayName:    common.String(spec.DisplayName),
		CompartmentId:  common.String(spec.CompartmentId),
		Description:    common.String(spec.Description),
		LicenseType:    accessgovernancecpsdk.LicenseTypeEnum(spec.LicenseType),
		InstanceUrl:    common.String("https://access-governance.example"),
		LifecycleState: accessgovernancecpsdk.InstanceLifecycleStateEnum(lifecycle),
		DefinedTags:    governanceInstanceDefinedTagsFromSpec(spec.DefinedTags),
		FreeformTags:   maps.Clone(spec.FreeformTags),
	}
}

func requireGovernanceInstanceStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}

func requireGovernanceInstanceOpcRequestID(
	t *testing.T,
	resource *accessgovernancecpv1beta1.GovernanceInstance,
	want string,
) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func requireGovernanceInstanceAsyncCurrent(
	t *testing.T,
	resource *accessgovernancecpv1beta1.GovernanceInstance,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func TestLookupExistingGovernanceInstanceRejectsAmbiguousTenantScopedMatches(t *testing.T) {
	t.Parallel()

	identity := governanceInstanceIdentity{
		compartmentID:    "ocid1.compartment.oc1..exampleuniqueID",
		displayName:      "governance-sample",
		licenseType:      string(accessgovernancecpsdk.LicenseTypeNewLicense),
		tenancyNamespace: "tenant-namespace",
	}

	client := &fakeGovernanceInstanceOCIClient{
		listFn: func(_ context.Context, _ accessgovernancecpsdk.ListGovernanceInstancesRequest) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error) {
			spec := newGovernanceInstanceTestResource().Spec
			return accessgovernancecpsdk.ListGovernanceInstancesResponse{
				GovernanceInstanceCollection: accessgovernancecpsdk.GovernanceInstanceCollection{
					Items: []accessgovernancecpsdk.GovernanceInstanceSummary{
						observedGovernanceInstanceSummaryFromSpec("ocid1.governanceinstance.oc1..one", spec, "ACTIVE"),
						observedGovernanceInstanceSummaryFromSpec("ocid1.governanceinstance.oc1..two", spec, "CREATING"),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
			spec := newGovernanceInstanceTestResource().Spec
			return accessgovernancecpsdk.GetGovernanceInstanceResponse{
				GovernanceInstance: observedGovernanceInstanceFromSpec(*req.GovernanceInstanceId, spec, "ACTIVE"),
			}, nil
		},
	}

	_, err := lookupExistingGovernanceInstance(context.Background(), client, nil, identity)
	if err == nil || !strings.Contains(err.Error(), "multiple exact matches") {
		t.Fatalf("lookupExistingGovernanceInstance() error = %v, want multiple exact matches failure", err)
	}
}
