/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateserviceaccess

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	psasdk "github.com/oracle/oci-go-sdk/v65/psa"
	psav1beta1 "github.com/oracle/oci-service-operator/api/psa/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakePrivateServiceAccessOCIClient struct {
	createFn      func(context.Context, psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error)
	getFn         func(context.Context, psasdk.GetPrivateServiceAccessRequest) (psasdk.GetPrivateServiceAccessResponse, error)
	listFn        func(context.Context, psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error)
	updateFn      func(context.Context, psasdk.UpdatePrivateServiceAccessRequest) (psasdk.UpdatePrivateServiceAccessResponse, error)
	deleteFn      func(context.Context, psasdk.DeletePrivateServiceAccessRequest) (psasdk.DeletePrivateServiceAccessResponse, error)
	workRequestFn func(context.Context, psasdk.GetPsaWorkRequestRequest) (psasdk.GetPsaWorkRequestResponse, error)
}

func (f *fakePrivateServiceAccessOCIClient) CreatePrivateServiceAccess(
	ctx context.Context,
	req psasdk.CreatePrivateServiceAccessRequest,
) (psasdk.CreatePrivateServiceAccessResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return psasdk.CreatePrivateServiceAccessResponse{}, nil
}

func (f *fakePrivateServiceAccessOCIClient) GetPrivateServiceAccess(
	ctx context.Context,
	req psasdk.GetPrivateServiceAccessRequest,
) (psasdk.GetPrivateServiceAccessResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return psasdk.GetPrivateServiceAccessResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakePrivateServiceAccessOCIClient) ListPrivateServiceAccesses(
	ctx context.Context,
	req psasdk.ListPrivateServiceAccessesRequest,
) (psasdk.ListPrivateServiceAccessesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return psasdk.ListPrivateServiceAccessesResponse{}, nil
}

func (f *fakePrivateServiceAccessOCIClient) UpdatePrivateServiceAccess(
	ctx context.Context,
	req psasdk.UpdatePrivateServiceAccessRequest,
) (psasdk.UpdatePrivateServiceAccessResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return psasdk.UpdatePrivateServiceAccessResponse{}, nil
}

func (f *fakePrivateServiceAccessOCIClient) DeletePrivateServiceAccess(
	ctx context.Context,
	req psasdk.DeletePrivateServiceAccessRequest,
) (psasdk.DeletePrivateServiceAccessResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return psasdk.DeletePrivateServiceAccessResponse{}, nil
}

func (f *fakePrivateServiceAccessOCIClient) GetPsaWorkRequest(
	ctx context.Context,
	req psasdk.GetPsaWorkRequestRequest,
) (psasdk.GetPsaWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return psasdk.GetPsaWorkRequestResponse{}, nil
}

func TestReviewedPrivateServiceAccessRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedPrivateServiceAccessRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedPrivateServiceAccessRuntimeSemantics() = nil")
	}

	if got.FormalService != "psa" {
		t.Fatalf("FormalService = %q, want psa", got.FormalService)
	}
	if got.FormalSlug != "privateserviceaccess" {
		t.Fatalf("FormalSlug = %q, want privateserviceaccess", got.FormalSlug)
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
	assertPrivateServiceAccessStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertPrivateServiceAccessStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertPrivateServiceAccessStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertPrivateServiceAccessStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertPrivateServiceAccessStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertPrivateServiceAccessStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertPrivateServiceAccessStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "serviceId", "subnetId", "ipv4Ip"})
	assertPrivateServiceAccessStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags", "nsgIds", "securityAttributes"})
	assertPrivateServiceAccessStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "ipv4Ip", "serviceId", "subnetId"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetPsaWorkRequest -> GetPrivateServiceAccess" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetPsaWorkRequest -> GetPrivateServiceAccess", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetPsaWorkRequest -> GetPrivateServiceAccess" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetPsaWorkRequest -> GetPrivateServiceAccess", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetPsaWorkRequest -> GetPrivateServiceAccess/ListPrivateServiceAccesses confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want GetPsaWorkRequest -> GetPrivateServiceAccess/ListPrivateServiceAccesses confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardPrivateServiceAccessExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makePrivateServiceAccessResource()
	resource.Spec.DisplayName = ""

	decision, err := guardPrivateServiceAccessExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardPrivateServiceAccessExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardPrivateServiceAccessExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "privateserviceaccess-sample"
	decision, err = guardPrivateServiceAccessExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardPrivateServiceAccessExistingBeforeCreate(complete criteria) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardPrivateServiceAccessExistingBeforeCreate(complete criteria) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildPrivateServiceAccessUpdateBodyPreservesClears(t *testing.T) {
	t.Parallel()

	currentResource := makePrivateServiceAccessResource()
	desired := makePrivateServiceAccessResource()
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}
	desired.Spec.SecurityAttributes = map[string]shared.MapValue{}
	desired.Spec.NsgIds = []string{}

	body, updateNeeded, err := buildPrivateServiceAccessUpdateBody(
		desired,
		psasdk.GetPrivateServiceAccessResponse{
			PrivateServiceAccess: makeSDKPrivateServiceAccess("ocid1.privateserviceaccess.oc1..existing", currentResource, psasdk.PrivateServiceAccessLifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildPrivateServiceAccessUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildPrivateServiceAccessUpdateBody() updateNeeded = false, want true")
	}

	requirePrivateServiceAccessStringPtr(t, "details.description", body.Description, "")
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
	if len(body.SecurityAttributes) != 0 {
		t.Fatalf("details.SecurityAttributes = %#v, want empty map for clear", body.SecurityAttributes)
	}
	if len(body.NsgIds) != 0 {
		t.Fatalf("details.NsgIds = %#v, want empty slice for clear", body.NsgIds)
	}
}

func TestPrivateServiceAccessCreateOrUpdateSkipsReuseWhenCriteriaMissing(t *testing.T) {
	t.Parallel()

	resource := makePrivateServiceAccessResource()
	resource.Spec.DisplayName = ""

	listCalls := 0
	createCalls := 0

	client := newTestPrivateServiceAccessClient(&fakePrivateServiceAccessOCIClient{
		listFn: func(_ context.Context, _ psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error) {
			listCalls++
			return psasdk.ListPrivateServiceAccessesResponse{}, nil
		},
		createFn: func(_ context.Context, req psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error) {
			createCalls++
			requirePrivateServiceAccessStringPtr(t, "create compartmentId", req.CreatePrivateServiceAccessDetails.CompartmentId, resource.Spec.CompartmentId)
			requirePrivateServiceAccessStringPtr(t, "create subnetId", req.CreatePrivateServiceAccessDetails.SubnetId, resource.Spec.SubnetId)
			requirePrivateServiceAccessStringPtr(t, "create serviceId", req.CreatePrivateServiceAccessDetails.ServiceId, resource.Spec.ServiceId)
			if req.CreatePrivateServiceAccessDetails.DisplayName != nil {
				t.Fatalf("create displayName = %#v, want nil when spec.displayName is empty", req.CreatePrivateServiceAccessDetails.DisplayName)
			}
			return psasdk.CreatePrivateServiceAccessResponse{
				PrivateServiceAccess: makeSDKPrivateServiceAccess("ocid1.privateserviceaccess.oc1..created", resource, psasdk.PrivateServiceAccessLifecycleStateCreating),
				OpcWorkRequestId:     common.String("wr-create-missing-display-name"),
				OpcRequestId:         common.String("opc-create-missing-display-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req psasdk.GetPsaWorkRequestRequest) (psasdk.GetPsaWorkRequestResponse, error) {
			requirePrivateServiceAccessStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-missing-display-name")
			return psasdk.GetPsaWorkRequestResponse{
				WorkRequest: makePrivateServiceAccessWorkRequest(
					"wr-create-missing-display-name",
					psasdk.OperationTypeCreatePrivateServiceAccess,
					psasdk.OperationStatusInProgress,
					psasdk.ActionTypeCreated,
					"ocid1.privateserviceaccess.oc1..created",
				),
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
	if listCalls != 0 {
		t.Fatalf("ListPrivateServiceAccesses() calls = %d, want 0 when pre-create criteria are incomplete", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreatePrivateServiceAccess() calls = %d, want 1", createCalls)
	}
	requirePrivateServiceAccessAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-missing-display-name", shared.OSOKAsyncClassPending)
}

func TestPrivateServiceAccessCreateOrUpdateBindsUniqueExactMatch(t *testing.T) {
	t.Parallel()

	resource := makePrivateServiceAccessResource()
	createCalls := 0
	listCalls := 0

	client := newTestPrivateServiceAccessClient(&fakePrivateServiceAccessOCIClient{
		listFn: func(_ context.Context, req psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error) {
			listCalls++
			requirePrivateServiceAccessStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requirePrivateServiceAccessStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			requirePrivateServiceAccessStringPtr(t, "list serviceId", req.ServiceId, resource.Spec.ServiceId)
			if req.VcnId != nil {
				t.Fatalf("list vcnId = %#v, want nil because pre-create reuse should not require a VCN lookup", req.VcnId)
			}
			return psasdk.ListPrivateServiceAccessesResponse{
				PrivateServiceAccessCollection: psasdk.PrivateServiceAccessCollection{
					Items: []psasdk.PrivateServiceAccessSummary{
						makeSDKPrivateServiceAccessSummary("ocid1.privateserviceaccess.oc1..matched", resource, psasdk.PrivateServiceAccessLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req psasdk.GetPrivateServiceAccessRequest) (psasdk.GetPrivateServiceAccessResponse, error) {
			requirePrivateServiceAccessStringPtr(t, "get privateServiceAccessId", req.PrivateServiceAccessId, "ocid1.privateserviceaccess.oc1..matched")
			return psasdk.GetPrivateServiceAccessResponse{
				PrivateServiceAccess: makeSDKPrivateServiceAccess("ocid1.privateserviceaccess.oc1..matched", resource, psasdk.PrivateServiceAccessLifecycleStateActive),
			}, nil
		},
		createFn: func(_ context.Context, _ psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error) {
			createCalls++
			return psasdk.CreatePrivateServiceAccessResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListPrivateServiceAccesses() calls = %d, want 1", listCalls)
	}
	if createCalls != 0 {
		t.Fatalf("CreatePrivateServiceAccess() calls = %d, want 0 after binding an exact match", createCalls)
	}
	if resource.Status.Id != "ocid1.privateserviceaccess.oc1..matched" {
		t.Fatalf("status.id = %q, want matched resource ID", resource.Status.Id)
	}
	if resource.Status.SubnetId != resource.Spec.SubnetId {
		t.Fatalf("status.subnetId = %q, want %q", resource.Status.SubnetId, resource.Spec.SubnetId)
	}
}

func TestPrivateServiceAccessCreateOrUpdateCreatesWhenListCandidateSubnetDiffers(t *testing.T) {
	t.Parallel()

	resource := makePrivateServiceAccessResource()
	createCalls := 0

	client := newTestPrivateServiceAccessClient(&fakePrivateServiceAccessOCIClient{
		listFn: func(_ context.Context, _ psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error) {
			mismatched := makeSDKPrivateServiceAccessSummary("ocid1.privateserviceaccess.oc1..other-subnet", resource, psasdk.PrivateServiceAccessLifecycleStateActive)
			mismatched.SubnetId = common.String("ocid1.subnet.oc1..different")
			return psasdk.ListPrivateServiceAccessesResponse{
				PrivateServiceAccessCollection: psasdk.PrivateServiceAccessCollection{
					Items: []psasdk.PrivateServiceAccessSummary{mismatched},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error) {
			createCalls++
			return psasdk.CreatePrivateServiceAccessResponse{
				PrivateServiceAccess: makeSDKPrivateServiceAccess("ocid1.privateserviceaccess.oc1..created-after-mismatch", resource, psasdk.PrivateServiceAccessLifecycleStateCreating),
				OpcWorkRequestId:     common.String("wr-create-subnet-mismatch"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req psasdk.GetPsaWorkRequestRequest) (psasdk.GetPsaWorkRequestResponse, error) {
			requirePrivateServiceAccessStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-subnet-mismatch")
			return psasdk.GetPsaWorkRequestResponse{
				WorkRequest: makePrivateServiceAccessWorkRequest(
					"wr-create-subnet-mismatch",
					psasdk.OperationTypeCreatePrivateServiceAccess,
					psasdk.OperationStatusInProgress,
					psasdk.ActionTypeCreated,
					"ocid1.privateserviceaccess.oc1..created-after-mismatch",
				),
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
	if createCalls != 1 {
		t.Fatalf("CreatePrivateServiceAccess() calls = %d, want 1 when list candidate subnet does not match", createCalls)
	}
}

func TestPrivateServiceAccessCreateOrUpdateFailsDuplicateExactMatches(t *testing.T) {
	t.Parallel()

	resource := makePrivateServiceAccessResource()
	createCalls := 0

	duplicateA := makeSDKPrivateServiceAccessSummary("ocid1.privateserviceaccess.oc1..match-a", resource, psasdk.PrivateServiceAccessLifecycleStateActive)
	duplicateB := makeSDKPrivateServiceAccessSummary("ocid1.privateserviceaccess.oc1..match-b", resource, psasdk.PrivateServiceAccessLifecycleStateActive)

	client := newTestPrivateServiceAccessClient(&fakePrivateServiceAccessOCIClient{
		listFn: func(_ context.Context, _ psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error) {
			return psasdk.ListPrivateServiceAccessesResponse{
				PrivateServiceAccessCollection: psasdk.PrivateServiceAccessCollection{
					Items: []psasdk.PrivateServiceAccessSummary{duplicateA, duplicateB},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error) {
			createCalls++
			return psasdk.CreatePrivateServiceAccessResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want duplicate pre-create match failure")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match message", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreatePrivateServiceAccess() calls = %d, want 0 after duplicate pre-create matches", createCalls)
	}
}

func newTestPrivateServiceAccessClient(client privateServiceAccessOCIClient) PrivateServiceAccessServiceClient {
	return newPrivateServiceAccessServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func makePrivateServiceAccessResource() *psav1beta1.PrivateServiceAccess {
	return &psav1beta1.PrivateServiceAccess{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "privateserviceaccess-sample",
			Namespace: "default",
		},
		Spec: psav1beta1.PrivateServiceAccessSpec{
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
			SubnetId:      "ocid1.subnet.oc1..exampleuniqueID",
			ServiceId:     "ocid1.psaservice.oc1..exampleuniqueID",
			DisplayName:   "privateserviceaccess-sample",
			Description:   "private service access",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"costCenter": "42",
				},
			},
			SecurityAttributes: map[string]shared.MapValue{
				"Oracle-DataSecurity-ZPR": {
					"mode": "audit",
				},
			},
			NsgIds: []string{
				"ocid1.nsg.oc1..examplefirst",
				"ocid1.nsg.oc1..examplesecond",
			},
			Ipv4Ip: "10.0.0.12",
		},
	}
}

func makeSDKPrivateServiceAccess(
	id string,
	resource *psav1beta1.PrivateServiceAccess,
	lifecycleState psasdk.PrivateServiceAccessLifecycleStateEnum,
) psasdk.PrivateServiceAccess {
	if resource == nil {
		resource = makePrivateServiceAccessResource()
	}

	return psasdk.PrivateServiceAccess{
		Id:                 common.String(id),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		DisplayName:        common.String(resource.Spec.DisplayName),
		VcnId:              common.String("ocid1.vcn.oc1..exampleuniqueID"),
		SubnetId:           common.String(resource.Spec.SubnetId),
		VnicId:             common.String("ocid1.vnic.oc1..exampleuniqueID"),
		LifecycleState:     lifecycleState,
		ServiceId:          common.String(resource.Spec.ServiceId),
		Fqdns:              []string{"service.example.oraclecloud.com"},
		DefinedTags:        privateServiceAccessNestedMapFromSpec(resource.Spec.DefinedTags),
		FreeformTags:       maps.Clone(resource.Spec.FreeformTags),
		SystemTags:         map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		SecurityAttributes: privateServiceAccessNestedMapFromSpec(resource.Spec.SecurityAttributes),
		Description:        common.String(resource.Spec.Description),
		NsgIds:             append([]string(nil), resource.Spec.NsgIds...),
		Ipv4Ip:             common.String(resource.Spec.Ipv4Ip),
	}
}

func makeSDKPrivateServiceAccessSummary(
	id string,
	resource *psav1beta1.PrivateServiceAccess,
	lifecycleState psasdk.PrivateServiceAccessLifecycleStateEnum,
) psasdk.PrivateServiceAccessSummary {
	if resource == nil {
		resource = makePrivateServiceAccessResource()
	}

	return psasdk.PrivateServiceAccessSummary{
		Id:                 common.String(id),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		DisplayName:        common.String(resource.Spec.DisplayName),
		VcnId:              common.String("ocid1.vcn.oc1..exampleuniqueID"),
		SubnetId:           common.String(resource.Spec.SubnetId),
		VnicId:             common.String("ocid1.vnic.oc1..exampleuniqueID"),
		LifecycleState:     lifecycleState,
		ServiceId:          common.String(resource.Spec.ServiceId),
		Fqdns:              []string{"service.example.oraclecloud.com"},
		DefinedTags:        privateServiceAccessNestedMapFromSpec(resource.Spec.DefinedTags),
		FreeformTags:       maps.Clone(resource.Spec.FreeformTags),
		SystemTags:         map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		SecurityAttributes: privateServiceAccessNestedMapFromSpec(resource.Spec.SecurityAttributes),
		Description:        common.String(resource.Spec.Description),
		NsgIds:             append([]string(nil), resource.Spec.NsgIds...),
		Ipv4Ip:             common.String(resource.Spec.Ipv4Ip),
	}
}

func makePrivateServiceAccessWorkRequest(
	workRequestID string,
	operationType psasdk.OperationTypeEnum,
	status psasdk.OperationStatusEnum,
	action psasdk.ActionTypeEnum,
	resourceID string,
) psasdk.WorkRequest {
	resources := []psasdk.WorkRequestResource{}
	if strings.TrimSpace(resourceID) != "" {
		resources = append(resources, psasdk.WorkRequestResource{
			EntityType: common.String("PrivateServiceAccess"),
			ActionType: action,
			Identifier: common.String(resourceID),
			EntityUri:  common.String("/20240301/privateServiceAccess/" + resourceID),
		})
	}
	return psasdk.WorkRequest{
		Id:            common.String(workRequestID),
		OperationType: operationType,
		Status:        status,
		Resources:     resources,
	}
}

func requirePrivateServiceAccessStringPtr(t *testing.T, field string, actual *string, want string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *actual != want {
		t.Fatalf("%s = %q, want %q", field, *actual, want)
	}
}

func requirePrivateServiceAccessAsyncCurrent(
	t *testing.T,
	resource *psav1beta1.PrivateServiceAccess,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func assertPrivateServiceAccessStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}
