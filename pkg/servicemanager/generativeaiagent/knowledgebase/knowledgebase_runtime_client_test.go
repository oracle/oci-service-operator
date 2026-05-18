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

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaiagentsdk "github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	generativeaiagentv1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeKnowledgeBaseOCIClient struct {
	createFn      func(context.Context, generativeaiagentsdk.CreateKnowledgeBaseRequest) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error)
	getFn         func(context.Context, generativeaiagentsdk.GetKnowledgeBaseRequest) (generativeaiagentsdk.GetKnowledgeBaseResponse, error)
	listFn        func(context.Context, generativeaiagentsdk.ListKnowledgeBasesRequest) (generativeaiagentsdk.ListKnowledgeBasesResponse, error)
	updateFn      func(context.Context, generativeaiagentsdk.UpdateKnowledgeBaseRequest) (generativeaiagentsdk.UpdateKnowledgeBaseResponse, error)
	deleteFn      func(context.Context, generativeaiagentsdk.DeleteKnowledgeBaseRequest) (generativeaiagentsdk.DeleteKnowledgeBaseResponse, error)
	workRequestFn func(context.Context, generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error)
}

func (f *fakeKnowledgeBaseOCIClient) CreateKnowledgeBase(
	ctx context.Context,
	req generativeaiagentsdk.CreateKnowledgeBaseRequest,
) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return generativeaiagentsdk.CreateKnowledgeBaseResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) GetKnowledgeBase(
	ctx context.Context,
	req generativeaiagentsdk.GetKnowledgeBaseRequest,
) (generativeaiagentsdk.GetKnowledgeBaseResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return generativeaiagentsdk.GetKnowledgeBaseResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeKnowledgeBaseOCIClient) ListKnowledgeBases(
	ctx context.Context,
	req generativeaiagentsdk.ListKnowledgeBasesRequest,
) (generativeaiagentsdk.ListKnowledgeBasesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return generativeaiagentsdk.ListKnowledgeBasesResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) UpdateKnowledgeBase(
	ctx context.Context,
	req generativeaiagentsdk.UpdateKnowledgeBaseRequest,
) (generativeaiagentsdk.UpdateKnowledgeBaseResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return generativeaiagentsdk.UpdateKnowledgeBaseResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) DeleteKnowledgeBase(
	ctx context.Context,
	req generativeaiagentsdk.DeleteKnowledgeBaseRequest,
) (generativeaiagentsdk.DeleteKnowledgeBaseResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return generativeaiagentsdk.DeleteKnowledgeBaseResponse{}, nil
}

func (f *fakeKnowledgeBaseOCIClient) GetWorkRequest(
	ctx context.Context,
	req generativeaiagentsdk.GetWorkRequestRequest,
) (generativeaiagentsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return generativeaiagentsdk.GetWorkRequestResponse{}, nil
}

func TestReviewedKnowledgeBaseRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedKnowledgeBaseRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedKnowledgeBaseRuntimeSemantics() = nil")
	}

	if got.FormalService != "generativeaiagent" {
		t.Fatalf("FormalService = %q, want generativeaiagent", got.FormalService)
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
	assertKnowledgeBaseStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertKnowledgeBaseStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertKnowledgeBaseStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertKnowledgeBaseStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertKnowledgeBaseStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags", "indexConfig"})
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

func TestBuildKnowledgeBaseCreateDetailsPreservesConcreteIndexConfigAndFalseBool(t *testing.T) {
	t.Parallel()

	resource := makeKnowledgeBaseResource()

	details, err := buildKnowledgeBaseCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildKnowledgeBaseCreateDetails() error = %v", err)
	}

	requireKnowledgeBaseStringPtr(t, "details.compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireKnowledgeBaseStringPtr(t, "details.displayName", details.DisplayName, resource.Spec.DisplayName)
	requireKnowledgeBaseStringPtr(t, "details.description", details.Description, resource.Spec.Description)
	requireKnowledgeBaseDefaultIndexConfig(t, "details.indexConfig", details.IndexConfig, false)
}

func TestBuildKnowledgeBaseUpdateBodyPreservesClearsAndIndexConfigChanges(t *testing.T) {
	t.Parallel()

	currentResource := makeKnowledgeBaseResource()
	currentResource.Spec.IndexConfig.ShouldEnableHybridSearch = true
	currentResource.Spec.Description = "current description"

	desired := makeKnowledgeBaseResource()
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildKnowledgeBaseUpdateBody(
		context.Background(),
		desired,
		desired.Namespace,
		generativeaiagentsdk.GetKnowledgeBaseResponse{
			KnowledgeBase: makeSDKKnowledgeBase("ocid1.knowledgebase.oc1..existing", currentResource, generativeaiagentsdk.KnowledgeBaseLifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildKnowledgeBaseUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildKnowledgeBaseUpdateBody() updateNeeded = false, want true")
	}

	requireKnowledgeBaseStringPtr(t, "details.description", body.Description, "")
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
	requireKnowledgeBaseDefaultIndexConfig(t, "details.indexConfig", body.IndexConfig, false)
}

func TestKnowledgeBaseCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	resource := makeKnowledgeBaseResource()
	resource.Spec.DisplayName = ""

	const (
		createdID     = "ocid1.knowledgebase.oc1..created"
		workRequestID = "wr-create-empty-name"
	)

	listCalls := 0
	createCalls := 0

	client := newTestKnowledgeBaseClient(&fakeKnowledgeBaseOCIClient{
		listFn: func(_ context.Context, _ generativeaiagentsdk.ListKnowledgeBasesRequest) (generativeaiagentsdk.ListKnowledgeBasesResponse, error) {
			listCalls++
			return generativeaiagentsdk.ListKnowledgeBasesResponse{}, nil
		},
		createFn: func(_ context.Context, req generativeaiagentsdk.CreateKnowledgeBaseRequest) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error) {
			createCalls++
			requireKnowledgeBaseStringPtr(t, "create compartmentId", req.CreateKnowledgeBaseDetails.CompartmentId, resource.Spec.CompartmentId)
			if req.CreateKnowledgeBaseDetails.DisplayName != nil {
				t.Fatalf("create displayName = %v, want nil when spec.displayName is empty", req.CreateKnowledgeBaseDetails.DisplayName)
			}
			return generativeaiagentsdk.CreateKnowledgeBaseResponse{
				KnowledgeBase:    makeSDKKnowledgeBase(createdID, resource, generativeaiagentsdk.KnowledgeBaseLifecycleStateCreating),
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-empty-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error) {
			requireKnowledgeBaseStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return generativeaiagentsdk.GetWorkRequestResponse{
				WorkRequest: makeKnowledgeBaseWorkRequest(
					workRequestID,
					generativeaiagentsdk.OperationTypeCreateKnowledgeBase,
					generativeaiagentsdk.OperationStatusInProgress,
					generativeaiagentsdk.ActionTypeInProgress,
					createdID,
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
	requireKnowledgeBaseAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q from create response body", got, createdID)
	}
}

func TestKnowledgeBaseCreateOrUpdateRejectsAmbiguousDisplayNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeKnowledgeBaseResource()
	createCalls := 0

	client := newTestKnowledgeBaseClient(&fakeKnowledgeBaseOCIClient{
		listFn: func(_ context.Context, req generativeaiagentsdk.ListKnowledgeBasesRequest) (generativeaiagentsdk.ListKnowledgeBasesResponse, error) {
			requireKnowledgeBaseStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireKnowledgeBaseStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return generativeaiagentsdk.ListKnowledgeBasesResponse{
				KnowledgeBaseCollection: generativeaiagentsdk.KnowledgeBaseCollection{
					Items: []generativeaiagentsdk.KnowledgeBaseSummary{
						makeSDKKnowledgeBaseSummary("ocid1.knowledgebase.oc1..first", resource, generativeaiagentsdk.KnowledgeBaseLifecycleStateActive),
						makeSDKKnowledgeBaseSummary("ocid1.knowledgebase.oc1..second", resource, generativeaiagentsdk.KnowledgeBaseLifecycleStateInactive),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ generativeaiagentsdk.CreateKnowledgeBaseRequest) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error) {
			createCalls++
			return generativeaiagentsdk.CreateKnowledgeBaseResponse{}, nil
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
	workRequests := map[string]generativeaiagentsdk.WorkRequest{
		workRequestID: makeKnowledgeBaseWorkRequest(
			workRequestID,
			generativeaiagentsdk.OperationTypeCreateKnowledgeBase,
			generativeaiagentsdk.OperationStatusInProgress,
			generativeaiagentsdk.ActionTypeInProgress,
			createdID,
		),
	}

	var createRequest generativeaiagentsdk.CreateKnowledgeBaseRequest
	getCalls := 0

	client := newTestKnowledgeBaseClient(&fakeKnowledgeBaseOCIClient{
		listFn: func(_ context.Context, req generativeaiagentsdk.ListKnowledgeBasesRequest) (generativeaiagentsdk.ListKnowledgeBasesResponse, error) {
			requireKnowledgeBaseStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireKnowledgeBaseStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return generativeaiagentsdk.ListKnowledgeBasesResponse{}, nil
		},
		createFn: func(_ context.Context, req generativeaiagentsdk.CreateKnowledgeBaseRequest) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error) {
			createRequest = req
			return generativeaiagentsdk.CreateKnowledgeBaseResponse{
				KnowledgeBase:    makeSDKKnowledgeBase(createdID, resource, generativeaiagentsdk.KnowledgeBaseLifecycleStateCreating),
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-knowledgebase"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error) {
			requireKnowledgeBaseStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return generativeaiagentsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req generativeaiagentsdk.GetKnowledgeBaseRequest) (generativeaiagentsdk.GetKnowledgeBaseResponse, error) {
			getCalls++
			requireKnowledgeBaseStringPtr(t, "get knowledgeBaseId", req.KnowledgeBaseId, createdID)
			return generativeaiagentsdk.GetKnowledgeBaseResponse{
				KnowledgeBase: makeSDKKnowledgeBase(createdID, resource, generativeaiagentsdk.KnowledgeBaseLifecycleStateActive),
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
	requireKnowledgeBaseDefaultIndexConfig(t, "create indexConfig", createRequest.CreateKnowledgeBaseDetails.IndexConfig, false)
	if getCalls != 0 {
		t.Fatalf("GetKnowledgeBase() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireKnowledgeBaseAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q from create response body", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(generativeaiagentsdk.KnowledgeBaseLifecycleStateCreating) {
		t.Fatalf("status.lifecycleState = %q, want CREATING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-knowledgebase" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-knowledgebase", got)
	}

	workRequests[workRequestID] = makeKnowledgeBaseWorkRequest(
		workRequestID,
		generativeaiagentsdk.OperationTypeCreateKnowledgeBase,
		generativeaiagentsdk.OperationStatusSucceeded,
		generativeaiagentsdk.ActionTypeCreated,
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
	if got := resource.Status.LifecycleState; got != string(generativeaiagentsdk.KnowledgeBaseLifecycleStateActive) {
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

func makeKnowledgeBaseResource() *generativeaiagentv1beta1.KnowledgeBase {
	return &generativeaiagentv1beta1.KnowledgeBase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knowledge-base-sample",
			Namespace: "default",
		},
		Spec: generativeaiagentv1beta1.KnowledgeBaseSpec{
			CompartmentId: "ocid1.compartment.oc1..knowledgebaseexample",
			DisplayName:   "knowledge-base-sample",
			Description:   "knowledge-base description",
			IndexConfig: generativeaiagentv1beta1.KnowledgeBaseIndexConfig{
				IndexConfigType:          "DEFAULT_INDEX_CONFIG",
				ShouldEnableHybridSearch: false,
			},
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
	resource *generativeaiagentv1beta1.KnowledgeBase,
	state generativeaiagentsdk.KnowledgeBaseLifecycleStateEnum,
) generativeaiagentsdk.KnowledgeBase {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return generativeaiagentsdk.KnowledgeBase{
		Id:               common.String(id),
		DisplayName:      common.String(resource.Spec.DisplayName),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		IndexConfig:      makeSDKKnowledgeBaseIndexConfig(resource.Spec.IndexConfig),
		TimeCreated:      &now,
		LifecycleState:   state,
		FreeformTags:     maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:      sdkDefinedTags(resource.Spec.DefinedTags),
		Description:      common.String(resource.Spec.Description),
		TimeUpdated:      &now,
		LifecycleDetails: common.String("lifecycle detail"),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
		KnowledgeBaseStatistics: &generativeaiagentsdk.KnowledgeBaseStatistics{
			SizeInBytes:        common.Int64(128),
			TotalIngestedFiles: common.Int64(4),
		},
	}
}

func makeSDKKnowledgeBaseSummary(
	id string,
	resource *generativeaiagentv1beta1.KnowledgeBase,
	state generativeaiagentsdk.KnowledgeBaseLifecycleStateEnum,
) generativeaiagentsdk.KnowledgeBaseSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return generativeaiagentsdk.KnowledgeBaseSummary{
		Id:               common.String(id),
		DisplayName:      common.String(resource.Spec.DisplayName),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &now,
		LifecycleState:   state,
		FreeformTags:     maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:      sdkDefinedTags(resource.Spec.DefinedTags),
		Description:      common.String(resource.Spec.Description),
		TimeUpdated:      &now,
		LifecycleDetails: common.String("lifecycle detail"),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKKnowledgeBaseIndexConfig(
	spec generativeaiagentv1beta1.KnowledgeBaseIndexConfig,
) generativeaiagentsdk.IndexConfig {
	switch spec.IndexConfigType {
	case "DEFAULT_INDEX_CONFIG":
		return generativeaiagentsdk.DefaultIndexConfig{
			ShouldEnableHybridSearch: common.Bool(spec.ShouldEnableHybridSearch),
		}
	default:
		return nil
	}
}

func makeKnowledgeBaseWorkRequest(
	id string,
	operation generativeaiagentsdk.OperationTypeEnum,
	status generativeaiagentsdk.OperationStatusEnum,
	action generativeaiagentsdk.ActionTypeEnum,
	resourceID string,
) generativeaiagentsdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return generativeaiagentsdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..knowledgebaseexample"),
		Resources:       []generativeaiagentsdk.WorkRequestResource{{EntityType: common.String("KnowledgeBase"), ActionType: action, Identifier: common.String(resourceID)}},
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

func requireKnowledgeBaseDefaultIndexConfig(
	t *testing.T,
	name string,
	got generativeaiagentsdk.IndexConfig,
	wantHybrid bool,
) {
	t.Helper()
	config, ok := got.(generativeaiagentsdk.DefaultIndexConfig)
	if !ok {
		t.Fatalf("%s type = %T, want DefaultIndexConfig", name, got)
	}
	if config.ShouldEnableHybridSearch == nil || *config.ShouldEnableHybridSearch != wantHybrid {
		t.Fatalf("%s.ShouldEnableHybridSearch = %v, want %t", name, config.ShouldEnableHybridSearch, wantHybrid)
	}
}

func requireKnowledgeBaseAsyncCurrent(
	t *testing.T,
	resource *generativeaiagentv1beta1.KnowledgeBase,
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
