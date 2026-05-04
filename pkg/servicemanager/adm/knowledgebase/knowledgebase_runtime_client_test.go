/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package knowledgebase

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	admsdk "github.com/oracle/oci-go-sdk/v65/adm"
	"github.com/oracle/oci-go-sdk/v65/common"
	admv1beta1 "github.com/oracle/oci-service-operator/api/adm/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeKnowledgeBaseOCIClient struct {
	createFn      func(context.Context, admsdk.CreateKnowledgeBaseRequest) (admsdk.CreateKnowledgeBaseResponse, error)
	getFn         func(context.Context, admsdk.GetKnowledgeBaseRequest) (admsdk.GetKnowledgeBaseResponse, error)
	listFn        func(context.Context, admsdk.ListKnowledgeBasesRequest) (admsdk.ListKnowledgeBasesResponse, error)
	updateFn      func(context.Context, admsdk.UpdateKnowledgeBaseRequest) (admsdk.UpdateKnowledgeBaseResponse, error)
	deleteFn      func(context.Context, admsdk.DeleteKnowledgeBaseRequest) (admsdk.DeleteKnowledgeBaseResponse, error)
	workRequestFn func(context.Context, admsdk.GetWorkRequestRequest) (admsdk.GetWorkRequestResponse, error)
}

func (f *fakeKnowledgeBaseOCIClient) CreateKnowledgeBase(
	ctx context.Context,
	req admsdk.CreateKnowledgeBaseRequest,
) (admsdk.CreateKnowledgeBaseResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return admsdk.CreateKnowledgeBaseResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) GetKnowledgeBase(
	ctx context.Context,
	req admsdk.GetKnowledgeBaseRequest,
) (admsdk.GetKnowledgeBaseResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return admsdk.GetKnowledgeBaseResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeKnowledgeBaseOCIClient) ListKnowledgeBases(
	ctx context.Context,
	req admsdk.ListKnowledgeBasesRequest,
) (admsdk.ListKnowledgeBasesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return admsdk.ListKnowledgeBasesResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) UpdateKnowledgeBase(
	ctx context.Context,
	req admsdk.UpdateKnowledgeBaseRequest,
) (admsdk.UpdateKnowledgeBaseResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return admsdk.UpdateKnowledgeBaseResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) DeleteKnowledgeBase(
	ctx context.Context,
	req admsdk.DeleteKnowledgeBaseRequest,
) (admsdk.DeleteKnowledgeBaseResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return admsdk.DeleteKnowledgeBaseResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) GetWorkRequest(
	ctx context.Context,
	req admsdk.GetWorkRequestRequest,
) (admsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return admsdk.GetWorkRequestResponse{}, nil
}

func TestReviewedKnowledgeBaseRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedKnowledgeBaseRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedKnowledgeBaseRuntimeSemantics() = nil")
	}

	if got.FormalService != "adm" {
		t.Fatalf("FormalService = %q, want adm", got.FormalService)
	}
	if got.FormalSlug != "knowledgebase" {
		t.Fatalf("FormalSlug = %q, want knowledgebase", got.FormalSlug)
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
	assertKnowledgeBaseStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertKnowledgeBaseStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertKnowledgeBaseStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertKnowledgeBaseStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertKnowledgeBaseStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertKnowledgeBaseStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertKnowledgeBaseStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	assertKnowledgeBaseStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "displayName", "freeformTags"})
	assertKnowledgeBaseStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetKnowledgeBase" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetKnowledgeBase", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetKnowledgeBase" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetKnowledgeBase", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetKnowledgeBase/ListKnowledgeBases confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardKnowledgeBaseExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeKnowledgeBaseResource()
	resource.Spec.DisplayName = ""

	decision, err := guardKnowledgeBaseExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardKnowledgeBaseExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardKnowledgeBaseExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "knowledge-base"
	decision, err = guardKnowledgeBaseExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardKnowledgeBaseExistingBeforeCreate(non-empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardKnowledgeBaseExistingBeforeCreate(non-empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildKnowledgeBaseUpdateBodyPreservesTagClears(t *testing.T) {
	t.Parallel()

	currentResource := makeKnowledgeBaseResource()
	desired := makeKnowledgeBaseResource()
	desired.Spec.DisplayName = "knowledge-base-updated"
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildKnowledgeBaseUpdateBody(
		desired,
		admsdk.GetKnowledgeBaseResponse{
			KnowledgeBase: makeSDKKnowledgeBase("ocid1.knowledgebase.oc1..existing", currentResource, admsdk.KnowledgeBaseLifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildKnowledgeBaseUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildKnowledgeBaseUpdateBody() updateNeeded = false, want true")
	}

	details := body
	requireKnowledgeBaseStringPtr(t, "details.displayName", details.DisplayName, desired.Spec.DisplayName)
	if len(details.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", details.FreeformTags)
	}
	if len(details.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", details.DefinedTags)
	}
}

func TestKnowledgeBaseCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	resource := makeKnowledgeBaseResource()
	resource.Spec.DisplayName = ""

	listCalls := 0
	createCalls := 0

	client := newTestKnowledgeBaseClient(&fakeKnowledgeBaseOCIClient{
		listFn: func(_ context.Context, _ admsdk.ListKnowledgeBasesRequest) (admsdk.ListKnowledgeBasesResponse, error) {
			listCalls++
			return admsdk.ListKnowledgeBasesResponse{}, nil
		},
		createFn: func(_ context.Context, req admsdk.CreateKnowledgeBaseRequest) (admsdk.CreateKnowledgeBaseResponse, error) {
			createCalls++
			requireKnowledgeBaseStringPtr(t, "create compartmentId", req.CreateKnowledgeBaseDetails.CompartmentId, resource.Spec.CompartmentId)
			if req.CreateKnowledgeBaseDetails.DisplayName != nil {
				t.Fatalf("create displayName = %v, want nil when spec.displayName is empty", req.CreateKnowledgeBaseDetails.DisplayName)
			}
			return admsdk.CreateKnowledgeBaseResponse{
				OpcWorkRequestId: common.String("wr-create-empty-name"),
				OpcRequestId:     common.String("opc-create-empty-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req admsdk.GetWorkRequestRequest) (admsdk.GetWorkRequestResponse, error) {
			requireKnowledgeBaseStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-empty-name")
			return admsdk.GetWorkRequestResponse{
				WorkRequest: makeKnowledgeBaseWorkRequest(
					"wr-create-empty-name",
					admsdk.OperationTypeCreateKnowledgeBase,
					admsdk.OperationStatusInProgress,
					admsdk.ActionTypeInProgress,
					"",
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
		t.Fatalf("ListKnowledgeBases() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateKnowledgeBase() calls = %d, want 1", createCalls)
	}
	requireKnowledgeBaseAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-empty-name", shared.OSOKAsyncClassPending)
}

func TestKnowledgeBaseCreateOrUpdateRejectsAmbiguousDisplayNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeKnowledgeBaseResource()
	createCalls := 0

	client := newTestKnowledgeBaseClient(&fakeKnowledgeBaseOCIClient{
		listFn: func(_ context.Context, req admsdk.ListKnowledgeBasesRequest) (admsdk.ListKnowledgeBasesResponse, error) {
			requireKnowledgeBaseStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireKnowledgeBaseStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return admsdk.ListKnowledgeBasesResponse{
				KnowledgeBaseCollection: admsdk.KnowledgeBaseCollection{
					Items: []admsdk.KnowledgeBaseSummary{
						makeSDKKnowledgeBaseSummary("ocid1.knowledgebase.oc1..first", resource, admsdk.KnowledgeBaseLifecycleStateActive),
						makeSDKKnowledgeBaseSummary("ocid1.knowledgebase.oc1..second", resource, admsdk.KnowledgeBaseLifecycleStateUpdating),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ admsdk.CreateKnowledgeBaseRequest) (admsdk.CreateKnowledgeBaseResponse, error) {
			createCalls++
			return admsdk.CreateKnowledgeBaseResponse{}, nil
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
		t.Fatalf("CreateKnowledgeBase() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestKnowledgeBaseServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.knowledgebase.oc1..created"
		workRequestID = "wr-knowledgebase-create"
	)

	resource := makeKnowledgeBaseResource()
	workRequests := map[string]admsdk.WorkRequest{
		workRequestID: makeKnowledgeBaseWorkRequest(
			workRequestID,
			admsdk.OperationTypeCreateKnowledgeBase,
			admsdk.OperationStatusInProgress,
			admsdk.ActionTypeInProgress,
			"",
		),
	}

	var createRequest admsdk.CreateKnowledgeBaseRequest
	getCalls := 0

	client := newTestKnowledgeBaseClient(&fakeKnowledgeBaseOCIClient{
		listFn: func(_ context.Context, req admsdk.ListKnowledgeBasesRequest) (admsdk.ListKnowledgeBasesResponse, error) {
			requireKnowledgeBaseStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireKnowledgeBaseStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return admsdk.ListKnowledgeBasesResponse{}, nil
		},
		createFn: func(_ context.Context, req admsdk.CreateKnowledgeBaseRequest) (admsdk.CreateKnowledgeBaseResponse, error) {
			createRequest = req
			return admsdk.CreateKnowledgeBaseResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-knowledgebase"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req admsdk.GetWorkRequestRequest) (admsdk.GetWorkRequestResponse, error) {
			requireKnowledgeBaseStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return admsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req admsdk.GetKnowledgeBaseRequest) (admsdk.GetKnowledgeBaseResponse, error) {
			getCalls++
			requireKnowledgeBaseStringPtr(t, "get knowledgeBaseId", req.KnowledgeBaseId, createdID)
			return admsdk.GetKnowledgeBaseResponse{
				KnowledgeBase: makeSDKKnowledgeBase(createdID, resource, admsdk.KnowledgeBaseLifecycleStateActive),
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
	requireKnowledgeBaseStringPtr(t, "create compartmentId", createRequest.CreateKnowledgeBaseDetails.CompartmentId, resource.Spec.CompartmentId)
	requireKnowledgeBaseStringPtr(t, "create displayName", createRequest.CreateKnowledgeBaseDetails.DisplayName, resource.Spec.DisplayName)
	if getCalls != 0 {
		t.Fatalf("GetKnowledgeBase() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireKnowledgeBaseAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-knowledgebase" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-knowledgebase", got)
	}

	workRequests[workRequestID] = makeKnowledgeBaseWorkRequest(
		workRequestID,
		admsdk.OperationTypeCreateKnowledgeBase,
		admsdk.OperationStatusSucceeded,
		admsdk.ActionTypeCreated,
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
		t.Fatalf("GetKnowledgeBase() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(admsdk.KnowledgeBaseLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func newTestKnowledgeBaseClient(client *fakeKnowledgeBaseOCIClient) KnowledgeBaseServiceClient {
	if client == nil {
		client = &fakeKnowledgeBaseOCIClient{}
	}
	return newKnowledgeBaseServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeKnowledgeBaseResource() *admv1beta1.KnowledgeBase {
	return &admv1beta1.KnowledgeBase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knowledge-base-sample",
			Namespace: "default",
		},
		Spec: admv1beta1.KnowledgeBaseSpec{
			CompartmentId: "ocid1.compartment.oc1..knowledgebaseexample",
			DisplayName:   "knowledge-base-sample",
			FreeformTags: map[string]string{
				"environment": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeSDKKnowledgeBase(
	id string,
	resource *admv1beta1.KnowledgeBase,
	state admsdk.KnowledgeBaseLifecycleStateEnum,
) admsdk.KnowledgeBase {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return admsdk.KnowledgeBase{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		TimeCreated:    &now,
		TimeUpdated:    &now,
		LifecycleState: state,
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		FreeformTags:   maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:    sdkDefinedTags(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKKnowledgeBaseSummary(
	id string,
	resource *admv1beta1.KnowledgeBase,
	state admsdk.KnowledgeBaseLifecycleStateEnum,
) admsdk.KnowledgeBaseSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return admsdk.KnowledgeBaseSummary{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		TimeCreated:    &now,
		TimeUpdated:    &now,
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		LifecycleState: state,
		FreeformTags:   maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:    sdkDefinedTags(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeKnowledgeBaseWorkRequest(
	id string,
	operation admsdk.OperationTypeEnum,
	status admsdk.OperationStatusEnum,
	action admsdk.ActionTypeEnum,
	resourceID string,
) admsdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return admsdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..knowledgebaseexample"),
		Resources:       []admsdk.WorkRequestResource{{EntityType: common.String("KnowledgeBase"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func sdkDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
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

func assertKnowledgeBaseStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireKnowledgeBaseStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireKnowledgeBaseAsyncCurrent(
	t *testing.T,
	resource *admv1beta1.KnowledgeBase,
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
