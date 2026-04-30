/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inclusioncriterion

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	governancerulescontrolplanesdk "github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testInclusionCriterionID      = "ocid1.inclusioncriterion.oc1..example"
	testInclusionCriterionOtherID = "ocid1.inclusioncriterion.oc1..other"
	testGovernanceRuleID          = "ocid1.governancerule.oc1..example"
	testGovernanceRuleOtherID     = "ocid1.governancerule.oc1..other"
	testTenancyID                 = "ocid1.tenancy.oc1..example"
	testTenancyOtherID            = "ocid1.tenancy.oc1..other"
	testInclusionCriterionUID     = "inclusion-criterion-uid"
)

type inclusionCriterionOCIClient interface {
	CreateInclusionCriterion(context.Context, governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error)
	GetInclusionCriterion(context.Context, governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error)
	DeleteInclusionCriterion(context.Context, governancerulescontrolplanesdk.DeleteInclusionCriterionRequest) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error)
}

type fakeInclusionCriterionOCIClient struct {
	createFn func(context.Context, governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error)
	getFn    func(context.Context, governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error)
	deleteFn func(context.Context, governancerulescontrolplanesdk.DeleteInclusionCriterionRequest) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error)
}

func (f *fakeInclusionCriterionOCIClient) CreateInclusionCriterion(
	ctx context.Context,
	request governancerulescontrolplanesdk.CreateInclusionCriterionRequest,
) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return governancerulescontrolplanesdk.CreateInclusionCriterionResponse{}, nil
}

func (f *fakeInclusionCriterionOCIClient) GetInclusionCriterion(
	ctx context.Context,
	request governancerulescontrolplanesdk.GetInclusionCriterionRequest,
) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return governancerulescontrolplanesdk.GetInclusionCriterionResponse{}, nil
}

func (f *fakeInclusionCriterionOCIClient) DeleteInclusionCriterion(
	ctx context.Context,
	request governancerulescontrolplanesdk.DeleteInclusionCriterionRequest,
) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return governancerulescontrolplanesdk.DeleteInclusionCriterionResponse{}, nil
}

func testInclusionCriterionClient(fake *fakeInclusionCriterionOCIClient) InclusionCriterionServiceClient {
	manager := &InclusionCriterionServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newInclusionCriterionRuntimeHooksWithOCIClient(fake)
	applyInclusionCriterionRuntimeHooks(&hooks)
	delegate := defaultInclusionCriterionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*governancerulescontrolplanev1beta1.InclusionCriterion](
			buildInclusionCriterionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapInclusionCriterionGeneratedClient(hooks, delegate)
}

func newInclusionCriterionRuntimeHooksWithOCIClient(client inclusionCriterionOCIClient) InclusionCriterionRuntimeHooks {
	return InclusionCriterionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*governancerulescontrolplanev1beta1.InclusionCriterion]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*governancerulescontrolplanev1beta1.InclusionCriterion]{},
		StatusHooks:     generatedruntime.StatusHooks[*governancerulescontrolplanev1beta1.InclusionCriterion]{},
		ParityHooks:     generatedruntime.ParityHooks[*governancerulescontrolplanev1beta1.InclusionCriterion]{},
		Async:           generatedruntime.AsyncHooks[*governancerulescontrolplanev1beta1.InclusionCriterion]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*governancerulescontrolplanev1beta1.InclusionCriterion]{},
		Create: runtimeOperationHooks[governancerulescontrolplanesdk.CreateInclusionCriterionRequest, governancerulescontrolplanesdk.CreateInclusionCriterionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateInclusionCriterionDetails", RequestName: "CreateInclusionCriterionDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error) {
				return client.CreateInclusionCriterion(ctx, request)
			},
		},
		Get: runtimeOperationHooks[governancerulescontrolplanesdk.GetInclusionCriterionRequest, governancerulescontrolplanesdk.GetInclusionCriterionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "InclusionCriterionId", RequestName: "inclusionCriterionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
				return client.GetInclusionCriterion(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[governancerulescontrolplanesdk.DeleteInclusionCriterionRequest, governancerulescontrolplanesdk.DeleteInclusionCriterionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "InclusionCriterionId", RequestName: "inclusionCriterionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.DeleteInclusionCriterionRequest) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error) {
				return client.DeleteInclusionCriterion(ctx, request)
			},
		},
		WrapGeneratedClient: []func(InclusionCriterionServiceClient) InclusionCriterionServiceClient{},
	}
}

func TestInclusionCriterionRuntimeSemantics(t *testing.T) {
	semantics := inclusionCriterionRuntimeSemantics()
	requireInclusionCriterionRuntimeSemantics(t, semantics)
	requireInclusionCriterionMutationSemantics(t, semantics)
	requireInclusionCriterionDeleteSemantics(t, semantics)
	requireInclusionCriterionFollowUpSemantics(t, semantics)
}

func requireInclusionCriterionRuntimeSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if semantics == nil {
		t.Fatal("inclusionCriterionRuntimeSemantics() = nil")
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("async strategy = %#v, want lifecycle", semantics.Async)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q", semantics.FinalizerPolicy)
	}
}

func requireInclusionCriterionMutationSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if len(semantics.Mutation.ForceNew) != 1 || semantics.Mutation.ForceNew[0] != "governanceRuleId" {
		t.Fatalf("force-new fields = %#v, want governanceRuleId", semantics.Mutation.ForceNew)
	}
}

func requireInclusionCriterionDeleteSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if len(semantics.Delete.TerminalStates) != 1 ||
		semantics.Delete.TerminalStates[0] != string(governancerulescontrolplanesdk.InclusionCriterionLifecycleStateDeleted) {
		t.Fatalf("delete terminal states = %#v, want DELETED", semantics.Delete.TerminalStates)
	}
}

func requireInclusionCriterionFollowUpSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if semantics.CreateFollowUp.Strategy != "read-after-write" || semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("follow-up strategies = create:%q delete:%q", semantics.CreateFollowUp.Strategy, semantics.DeleteFollowUp.Strategy)
	}
}

func TestInclusionCriterionCreateBodyBuildsTenancyAssociation(t *testing.T) {
	resource := makeInclusionCriterionResource()

	body, err := buildInclusionCriterionCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildInclusionCriterionCreateBody() error = %v", err)
	}
	details, ok := body.(governancerulescontrolplanesdk.CreateInclusionCriterionDetails)
	if !ok {
		t.Fatalf("body type = %T, want CreateInclusionCriterionDetails", body)
	}
	if got := inclusionCriterionStringValue(details.GovernanceRuleId); got != testGovernanceRuleID {
		t.Fatalf("governanceRuleId = %q, want %q", got, testGovernanceRuleID)
	}
	if details.Type != governancerulescontrolplanesdk.InclusionCriterionTypeTenancy {
		t.Fatalf("type = %q, want TENANCY", details.Type)
	}
	tenancy := requireTenancyAssociation(t, details.Association)
	if got := inclusionCriterionStringValue(tenancy.TenancyId); got != testTenancyID {
		t.Fatalf("association.tenancyId = %q, want %q", got, testTenancyID)
	}
}

func TestInclusionCriterionCreateBodyRejectsAllAssociation(t *testing.T) {
	resource := makeInclusionCriterionResource()
	resource.Spec.Type = string(governancerulescontrolplanesdk.InclusionCriterionTypeAll)

	_, err := buildInclusionCriterionCreateBody(context.Background(), resource, "default")
	if err == nil {
		t.Fatal("buildInclusionCriterionCreateBody() error = nil, want ALL association rejection")
	}
	if !strings.Contains(err.Error(), "ALL") || !strings.Contains(err.Error(), "association") {
		t.Fatalf("error = %q, want ALL association rejection", err.Error())
	}
}

func TestInclusionCriterionServiceClientCreatesAndProjectsStatus(t *testing.T) {
	resource := makeInclusionCriterionResource()
	var createCalls int
	var getCalls int
	var capturedCreate governancerulescontrolplanesdk.CreateInclusionCriterionRequest
	fake := &fakeInclusionCriterionOCIClient{
		createFn: func(_ context.Context, request governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error) {
			createCalls++
			capturedCreate = request
			return governancerulescontrolplanesdk.CreateInclusionCriterionResponse{
				InclusionCriterion: makeSDKInclusionCriterion(
					testInclusionCriterionID,
					testGovernanceRuleID,
					governancerulescontrolplanesdk.InclusionCriterionTypeTenancy,
					testTenancyAssociation(testTenancyID),
					governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive,
				),
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
			getCalls++
			if got := inclusionCriterionStringValue(request.InclusionCriterionId); got != testInclusionCriterionID {
				t.Fatalf("get inclusionCriterionId = %q, want %q", got, testInclusionCriterionID)
			}
			return governancerulescontrolplanesdk.GetInclusionCriterionResponse{
				InclusionCriterion: makeSDKInclusionCriterion(
					testInclusionCriterionID,
					testGovernanceRuleID,
					governancerulescontrolplanesdk.InclusionCriterionTypeTenancy,
					testTenancyAssociation(testTenancyID),
					governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive,
				),
			}, nil
		},
	}

	response, err := testInclusionCriterionClient(fake).CreateOrUpdate(context.Background(), resource, inclusionCriterionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if createCalls != 1 || getCalls != 1 {
		t.Fatalf("createCalls = %d, getCalls = %d, want 1/1", createCalls, getCalls)
	}
	if got := inclusionCriterionStringValue(capturedCreate.OpcRetryToken); got != testInclusionCriterionUID {
		t.Fatalf("opcRetryToken = %q, want %q", got, testInclusionCriterionUID)
	}
	requireTenancyAssociation(t, capturedCreate.Association)
	requireInclusionCriterionProjectedStatus(t, resource, testInclusionCriterionID, testGovernanceRuleID, testTenancyID)
	requireInclusionCriterionOpcRequestID(t, resource, "opc-create")
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil", current)
	}
}

func TestInclusionCriterionServiceClientBindsTrackedIDWithoutCreate(t *testing.T) {
	resource := makeInclusionCriterionResource()
	resource.Status.Id = testInclusionCriterionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testInclusionCriterionID)
	var getCalls int
	fake := &fakeInclusionCriterionOCIClient{
		createFn: func(context.Context, governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error) {
			t.Fatal("CreateInclusionCriterion should not be called for tracked resource")
			return governancerulescontrolplanesdk.CreateInclusionCriterionResponse{}, nil
		},
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
			getCalls++
			if got := inclusionCriterionStringValue(request.InclusionCriterionId); got != testInclusionCriterionID {
				t.Fatalf("get inclusionCriterionId = %q, want %q", got, testInclusionCriterionID)
			}
			return governancerulescontrolplanesdk.GetInclusionCriterionResponse{
				InclusionCriterion: makeSDKInclusionCriterion(
					testInclusionCriterionID,
					testGovernanceRuleID,
					governancerulescontrolplanesdk.InclusionCriterionTypeTenancy,
					testTenancyAssociation(testTenancyID),
					governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive,
				),
			}, nil
		},
	}

	response, err := testInclusionCriterionClient(fake).CreateOrUpdate(context.Background(), resource, inclusionCriterionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", getCalls)
	}
	requireInclusionCriterionProjectedStatus(t, resource, testInclusionCriterionID, testGovernanceRuleID, testTenancyID)
}

func TestInclusionCriterionServiceClientRejectsNoUpdateDrift(t *testing.T) {
	resource := makeInclusionCriterionResource()
	resource.Status.Id = testInclusionCriterionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testInclusionCriterionID)
	fake := &fakeInclusionCriterionOCIClient{
		createFn: func(context.Context, governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error) {
			t.Fatal("CreateInclusionCriterion should not be called for drifted tracked resource")
			return governancerulescontrolplanesdk.CreateInclusionCriterionResponse{}, nil
		},
		getFn: func(context.Context, governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
			return governancerulescontrolplanesdk.GetInclusionCriterionResponse{
				InclusionCriterion: makeSDKInclusionCriterion(
					testInclusionCriterionID,
					testGovernanceRuleID,
					governancerulescontrolplanesdk.InclusionCriterionTypeAll,
					nil,
					governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive,
				),
			}, nil
		},
	}

	response, err := testInclusionCriterionClient(fake).CreateOrUpdate(context.Background(), resource, inclusionCriterionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want no-update drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "no OCI update operation") || !strings.Contains(err.Error(), "type") {
		t.Fatalf("error = %q, want no-update type drift rejection", err.Error())
	}
}

func TestInclusionCriterionDeleteWaitsForActiveReadbackAndAvoidsDuplicateDelete(t *testing.T) {
	resource := makeInclusionCriterionResource()
	resource.Status.Id = testInclusionCriterionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testInclusionCriterionID)
	client, calls := activeReadbackDeleteClient(t)

	deleted, err := client.Delete(context.Background(), resource)
	requireInclusionCriterionDeletePendingResult(t, "Delete", deleted, err)
	requireInclusionCriterionDeleteCallCounts(t, calls, 2, 1, "after first delete")
	requireInclusionCriterionAsyncDeletePending(t, resource)
	requireInclusionCriterionOpcRequestID(t, resource, "opc-delete")

	deleted, err = client.Delete(context.Background(), resource)
	requireInclusionCriterionDeletePendingResult(t, "second Delete", deleted, err)
	requireInclusionCriterionDeleteCallCounts(t, calls, 3, 1, "after second delete")
	requireInclusionCriterionAsyncDeletePending(t, resource)
}

type inclusionCriterionDeleteCallCounts struct {
	get    int
	delete int
}

func activeReadbackDeleteClient(t *testing.T) (InclusionCriterionServiceClient, *inclusionCriterionDeleteCallCounts) {
	t.Helper()
	calls := &inclusionCriterionDeleteCallCounts{}
	fake := &fakeInclusionCriterionOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
			calls.get++
			requireInclusionCriterionGetRequestID(t, request, testInclusionCriterionID)
			return activeInclusionCriterionReadback(), nil
		},
		deleteFn: func(_ context.Context, request governancerulescontrolplanesdk.DeleteInclusionCriterionRequest) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error) {
			calls.delete++
			requireInclusionCriterionDeleteRequestID(t, request, testInclusionCriterionID)
			return governancerulescontrolplanesdk.DeleteInclusionCriterionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}
	return testInclusionCriterionClient(fake), calls
}

func activeInclusionCriterionReadback() governancerulescontrolplanesdk.GetInclusionCriterionResponse {
	return governancerulescontrolplanesdk.GetInclusionCriterionResponse{
		InclusionCriterion: makeSDKInclusionCriterion(
			testInclusionCriterionID,
			testGovernanceRuleID,
			governancerulescontrolplanesdk.InclusionCriterionTypeTenancy,
			testTenancyAssociation(testTenancyID),
			governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive,
		),
	}
}

func requireInclusionCriterionGetRequestID(
	t *testing.T,
	request governancerulescontrolplanesdk.GetInclusionCriterionRequest,
	want string,
) {
	t.Helper()
	if got := inclusionCriterionStringValue(request.InclusionCriterionId); got != want {
		t.Fatalf("get inclusionCriterionId = %q, want %q", got, want)
	}
}

func requireInclusionCriterionDeleteRequestID(
	t *testing.T,
	request governancerulescontrolplanesdk.DeleteInclusionCriterionRequest,
	want string,
) {
	t.Helper()
	if got := inclusionCriterionStringValue(request.InclusionCriterionId); got != want {
		t.Fatalf("delete inclusionCriterionId = %q, want %q", got, want)
	}
}

func requireInclusionCriterionDeletePendingResult(t *testing.T, operation string, deleted bool, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s() error = %v", operation, err)
	}
	if deleted {
		t.Fatalf("%s() deleted = true, want false while readback is ACTIVE", operation)
	}
}

func requireInclusionCriterionDeleteCallCounts(
	t *testing.T,
	calls *inclusionCriterionDeleteCallCounts,
	wantGet int,
	wantDelete int,
	context string,
) {
	t.Helper()
	if calls.get != wantGet || calls.delete != wantDelete {
		t.Fatalf("%s getCalls = %d, deleteCalls = %d, want %d/%d", context, calls.get, calls.delete, wantGet, wantDelete)
	}
}

func TestInclusionCriterionDeleteConfirmsDeletedReadback(t *testing.T) {
	resource := makeInclusionCriterionResource()
	resource.Status.Id = testInclusionCriterionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testInclusionCriterionID)
	getCalls := 0
	fake := &fakeInclusionCriterionOCIClient{
		getFn: func(context.Context, governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
			getCalls++
			state := governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive
			if getCalls > 1 {
				state = governancerulescontrolplanesdk.InclusionCriterionLifecycleStateDeleted
			}
			return governancerulescontrolplanesdk.GetInclusionCriterionResponse{
				InclusionCriterion: makeSDKInclusionCriterion(
					testInclusionCriterionID,
					testGovernanceRuleID,
					governancerulescontrolplanesdk.InclusionCriterionTypeTenancy,
					testTenancyAssociation(testTenancyID),
					state,
				),
			}, nil
		},
		deleteFn: func(context.Context, governancerulescontrolplanesdk.DeleteInclusionCriterionRequest) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error) {
			return governancerulescontrolplanesdk.DeleteInclusionCriterionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testInclusionCriterionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatalf("Delete() deleted = false, want true after DELETED readback")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatalf("status.status.deletedAt = nil, want set")
	}
	requireInclusionCriterionOpcRequestID(t, resource, "opc-delete")
}

func TestInclusionCriterionDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := makeInclusionCriterionResource()
	resource.Status.Id = testInclusionCriterionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testInclusionCriterionID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-read"
	fake := &fakeInclusionCriterionOCIClient{
		getFn: func(context.Context, governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error) {
			return governancerulescontrolplanesdk.GetInclusionCriterionResponse{}, authErr
		},
		deleteFn: func(context.Context, governancerulescontrolplanesdk.DeleteInclusionCriterionRequest) (governancerulescontrolplanesdk.DeleteInclusionCriterionResponse, error) {
			t.Fatal("DeleteInclusionCriterion should not be called after auth-shaped confirm read")
			return governancerulescontrolplanesdk.DeleteInclusionCriterionResponse{}, nil
		},
	}

	deleted, err := testInclusionCriterionClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("error = %q, want ambiguous auth-shaped not-found", err.Error())
	}
	requireInclusionCriterionOpcRequestID(t, resource, "opc-auth-read")
}

func TestInclusionCriterionCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := makeInclusionCriterionResource()
	createErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeInclusionCriterionOCIClient{
		createFn: func(context.Context, governancerulescontrolplanesdk.CreateInclusionCriterionRequest) (governancerulescontrolplanesdk.CreateInclusionCriterionResponse, error) {
			return governancerulescontrolplanesdk.CreateInclusionCriterionResponse{}, createErr
		},
	}

	response, err := testInclusionCriterionClient(fake).CreateOrUpdate(context.Background(), resource, inclusionCriterionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	requireInclusionCriterionOpcRequestID(t, resource, "opc-create-error")
}

func makeInclusionCriterionResource() *governancerulescontrolplanev1beta1.InclusionCriterion {
	return &governancerulescontrolplanev1beta1.InclusionCriterion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "criterion-alpha",
			Namespace: "default",
			UID:       types.UID(testInclusionCriterionUID),
		},
		Spec: governancerulescontrolplanev1beta1.InclusionCriterionSpec{
			GovernanceRuleId: testGovernanceRuleID,
			Type:             string(governancerulescontrolplanesdk.InclusionCriterionTypeTenancy),
			Association: governancerulescontrolplanev1beta1.InclusionCriterionAssociation{
				TenancyId: testTenancyID,
			},
		},
	}
}

func makeSDKInclusionCriterion(
	id string,
	governanceRuleID string,
	criterionType governancerulescontrolplanesdk.InclusionCriterionTypeEnum,
	association governancerulescontrolplanesdk.Association,
	state governancerulescontrolplanesdk.InclusionCriterionLifecycleStateEnum,
) governancerulescontrolplanesdk.InclusionCriterion {
	timestamp := common.SDKTime{Time: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)}
	return governancerulescontrolplanesdk.InclusionCriterion{
		Id:               common.String(id),
		GovernanceRuleId: common.String(governanceRuleID),
		Type:             criterionType,
		LifecycleState:   state,
		TimeCreated:      &timestamp,
		TimeUpdated:      &timestamp,
		Association:      association,
	}
}

func testTenancyAssociation(tenancyID string) governancerulescontrolplanesdk.TenancyAssociation {
	return governancerulescontrolplanesdk.TenancyAssociation{TenancyId: common.String(tenancyID)}
}

func inclusionCriterionRequest(resource *governancerulescontrolplanev1beta1.InclusionCriterion) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func requireTenancyAssociation(
	t *testing.T,
	association governancerulescontrolplanesdk.Association,
) governancerulescontrolplanesdk.TenancyAssociation {
	t.Helper()
	tenancy, ok := association.(governancerulescontrolplanesdk.TenancyAssociation)
	if !ok {
		t.Fatalf("association type = %T, want TenancyAssociation", association)
	}
	return tenancy
}

func requireInclusionCriterionProjectedStatus(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	wantID string,
	wantGovernanceRuleID string,
	wantTenancyID string,
) {
	t.Helper()
	if got := resource.Status.Id; got != wantID {
		t.Fatalf("status.id = %q, want %q", got, wantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if got := resource.Status.GovernanceRuleId; got != wantGovernanceRuleID {
		t.Fatalf("status.governanceRuleId = %q, want %q", got, wantGovernanceRuleID)
	}
	if got := resource.Status.Type; got != string(governancerulescontrolplanesdk.InclusionCriterionTypeTenancy) {
		t.Fatalf("status.type = %q, want TENANCY", got)
	}
	if got := resource.Status.LifecycleState; got != string(governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.Association.Type; got != string(governancerulescontrolplanesdk.InclusionCriterionTypeTenancy) {
		t.Fatalf("status.association.type = %q, want TENANCY", got)
	}
	if got := resource.Status.Association.TenancyId; got != wantTenancyID {
		t.Fatalf("status.association.tenancyId = %q, want %q", got, wantTenancyID)
	}
}

func requireInclusionCriterionOpcRequestID(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	want string,
) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func requireInclusionCriterionAsyncDeletePending(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending ACTIVE", current)
	}
}
