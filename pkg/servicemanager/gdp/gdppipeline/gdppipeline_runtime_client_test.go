/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package gdppipeline

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	gdpsdk "github.com/oracle/oci-go-sdk/v65/gdp"
	gdpv1beta1 "github.com/oracle/oci-service-operator/api/gdp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeGdpPipelineOCIClient struct {
	createFn      func(context.Context, gdpsdk.CreateGdpPipelineRequest) (gdpsdk.CreateGdpPipelineResponse, error)
	getFn         func(context.Context, gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error)
	listFn        func(context.Context, gdpsdk.ListGdpPipelinesRequest) (gdpsdk.ListGdpPipelinesResponse, error)
	updateFn      func(context.Context, gdpsdk.UpdateGdpPipelineRequest) (gdpsdk.UpdateGdpPipelineResponse, error)
	deleteFn      func(context.Context, gdpsdk.DeleteGdpPipelineRequest) (gdpsdk.DeleteGdpPipelineResponse, error)
	workRequestFn func(context.Context, gdpsdk.GetGdpWorkRequestRequest) (gdpsdk.GetGdpWorkRequestResponse, error)
}

func (f *fakeGdpPipelineOCIClient) CreateGdpPipeline(
	ctx context.Context,
	req gdpsdk.CreateGdpPipelineRequest,
) (gdpsdk.CreateGdpPipelineResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return gdpsdk.CreateGdpPipelineResponse{}, nil
}

func (f *fakeGdpPipelineOCIClient) GetGdpPipeline(
	ctx context.Context,
	req gdpsdk.GetGdpPipelineRequest,
) (gdpsdk.GetGdpPipelineResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return gdpsdk.GetGdpPipelineResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeGdpPipelineOCIClient) ListGdpPipelines(
	ctx context.Context,
	req gdpsdk.ListGdpPipelinesRequest,
) (gdpsdk.ListGdpPipelinesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return gdpsdk.ListGdpPipelinesResponse{}, nil
}

func (f *fakeGdpPipelineOCIClient) UpdateGdpPipeline(
	ctx context.Context,
	req gdpsdk.UpdateGdpPipelineRequest,
) (gdpsdk.UpdateGdpPipelineResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return gdpsdk.UpdateGdpPipelineResponse{}, nil
}

func (f *fakeGdpPipelineOCIClient) DeleteGdpPipeline(
	ctx context.Context,
	req gdpsdk.DeleteGdpPipelineRequest,
) (gdpsdk.DeleteGdpPipelineResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return gdpsdk.DeleteGdpPipelineResponse{}, nil
}

func (f *fakeGdpPipelineOCIClient) GetGdpWorkRequest(
	ctx context.Context,
	req gdpsdk.GetGdpWorkRequestRequest,
) (gdpsdk.GetGdpWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return gdpsdk.GetGdpWorkRequestResponse{}, nil
}

type gdpPipelineRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func newTestGdpPipelineClient(fake *fakeGdpPipelineOCIClient) GdpPipelineServiceClient {
	return newGdpPipelineServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func TestReviewedGdpPipelineRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedGdpPipelineRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedGdpPipelineRuntimeSemantics() = nil")
	}

	if got.FormalService != "gdp" {
		t.Fatalf("FormalService = %q, want gdp", got.FormalService)
	}
	if got.FormalSlug != "gdppipeline" {
		t.Fatalf("FormalSlug = %q, want gdppipeline", got.FormalSlug)
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
	assertGdpPipelineStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertGdpPipelineStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertGdpPipelineStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertGdpPipelineStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertGdpPipelineStringSliceEqual(
		t,
		"List.MatchFields",
		got.List.MatchFields,
		[]string{"bucketDetails", "compartmentId", "displayName", "id", "peeringRegion", "pipelineType"},
	)
	assertGdpPipelineStringSliceEqual(
		t,
		"Mutation.Mutable",
		got.Mutation.Mutable,
		[]string{
			"approvalKeyVaultId",
			"authorizationDetails",
			"definedTags",
			"description",
			"displayName",
			"fileTypes",
			"freeformTags",
			"isApprovalNeeded",
			"isChunkingEnabled",
			"isFileOverrideInDestinationEnabled",
			"isScanningEnabled",
			"serviceLogGroupId",
		},
	)
	assertGdpPipelineStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"bucketDetails", "compartmentId", "peeringRegion", "pipelineType"})
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetGdpPipeline" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want workrequest-backed follow-up", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetGdpPipeline" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want workrequest-backed follow-up", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetGdpPipeline/ListGdpPipelines confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
}

func TestGuardGdpPipelineExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeGdpPipelineResource()

	resource.Spec.DisplayName = ""
	decision, err := guardGdpPipelineExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty displayName) = %q, want skip", decision)
	}

	resource = makeGdpPipelineResource()
	resource.Spec.PipelineType = ""
	decision, err = guardGdpPipelineExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty pipelineType) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty pipelineType) = %q, want skip", decision)
	}

	resource = makeGdpPipelineResource()
	resource.Spec.PeeringRegion = ""
	decision, err = guardGdpPipelineExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty peeringRegion) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty peeringRegion) = %q, want skip", decision)
	}

	resource = makeGdpPipelineResource()
	resource.Spec.BucketDetails = nil
	decision, err = guardGdpPipelineExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty bucketDetails) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(empty bucketDetails) = %q, want skip", decision)
	}

	resource = makeGdpPipelineResource()
	decision, err = guardGdpPipelineExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(valid identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardGdpPipelineExistingBeforeCreate(valid identity) = %q, want allow", decision)
	}
}

func TestBuildGdpPipelineCreateDetailsPreservesExplicitFalseBooleans(t *testing.T) {
	t.Parallel()

	resource := makeGdpPipelineResource()
	resource.Spec.IsFileOverrideInDestinationEnabled = false
	resource.Spec.IsScanningEnabled = false
	resource.Spec.IsChunkingEnabled = false
	resource.Spec.IsApprovalNeeded = false

	details, err := buildGdpPipelineCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildGdpPipelineCreateDetails() error = %v", err)
	}

	requireGdpPipelineStringPtr(t, "compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireGdpPipelineStringPtr(t, "displayName", details.DisplayName, resource.Spec.DisplayName)
	if got := details.PipelineType; got != gdpsdk.GdpPipelinePipelineTypeEnum(resource.Spec.PipelineType) {
		t.Fatalf("PipelineType = %q, want %q", got, resource.Spec.PipelineType)
	}
	if len(details.BucketDetails) != len(resource.Spec.BucketDetails) {
		t.Fatalf("BucketDetails len = %d, want %d", len(details.BucketDetails), len(resource.Spec.BucketDetails))
	}
	requireGdpPipelineBoolPtr(t, "isFileOverrideInDestinationEnabled", details.IsFileOverrideInDestinationEnabled, false)
	requireGdpPipelineBoolPtr(t, "isScanningEnabled", details.IsScanningEnabled, false)
	requireGdpPipelineBoolPtr(t, "isChunkingEnabled", details.IsChunkingEnabled, false)
	requireGdpPipelineBoolPtr(t, "isApprovalNeeded", details.IsApprovalNeeded, false)

	body := gdpPipelineSerializedRequestBody(
		t,
		gdpsdk.CreateGdpPipelineRequest{CreateGdpPipelineDetails: details},
		http.MethodPost,
		"/gdpPipelines",
	)
	for _, want := range []string{
		`"bucketType":"SOURCE"`,
		`"isFileOverrideInDestinationEnabled":false`,
		`"isScanningEnabled":false`,
		`"isChunkingEnabled":false`,
		`"isApprovalNeeded":false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestBuildGdpPipelineUpdateBodyPreservesClearSemantics(t *testing.T) {
	t.Parallel()

	resource := makeGdpPipelineResource()
	resource.Spec.Description = ""
	resource.Spec.ServiceLogGroupId = ""
	resource.Spec.FileTypes = []string{}
	resource.Spec.AuthorizationDetails = ""
	resource.Spec.IsFileOverrideInDestinationEnabled = false
	resource.Spec.IsScanningEnabled = false
	resource.Spec.IsChunkingEnabled = false
	resource.Spec.IsApprovalNeeded = false
	resource.Spec.ApprovalKeyVaultId = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := makeSDKGdpPipeline("ocid1.gdppipeline.oc1..existing", makeGdpPipelineResource(), gdpsdk.GdpPipelineLifecycleStateActive)

	details, updateNeeded, err := buildGdpPipelineUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildGdpPipelineUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildGdpPipelineUpdateBody() updateNeeded = false, want true")
	}

	requireGdpPipelineStringPtr(t, "description", details.Description, "")
	requireGdpPipelineStringPtr(t, "serviceLogGroupId", details.ServiceLogGroupId, "")
	requireGdpPipelineStringPtr(t, "authorizationDetails", details.AuthorizationDetails, "")
	requireGdpPipelineStringPtr(t, "approvalKeyVaultId", details.ApprovalKeyVaultId, "")
	if details.FileTypes != nil {
		t.Fatalf("FileTypes = %#v, want nil because empty-slice clear is not published", details.FileTypes)
	}
	requireGdpPipelineBoolPtr(t, "isFileOverrideInDestinationEnabled", details.IsFileOverrideInDestinationEnabled, false)
	requireGdpPipelineBoolPtr(t, "isScanningEnabled", details.IsScanningEnabled, false)
	requireGdpPipelineBoolPtr(t, "isChunkingEnabled", details.IsChunkingEnabled, false)
	requireGdpPipelineBoolPtr(t, "isApprovalNeeded", details.IsApprovalNeeded, false)
	if len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want empty map clear", details.FreeformTags)
	}
	if len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want empty map clear", details.DefinedTags)
	}

	body := gdpPipelineSerializedRequestBody(
		t,
		gdpsdk.UpdateGdpPipelineRequest{
			GdpPipelineId:            common.String("ocid1.gdppipeline.oc1..existing"),
			UpdateGdpPipelineDetails: details,
		},
		http.MethodPut,
		"/gdpPipelines/ocid1.gdppipeline.oc1..existing",
	)
	for _, want := range []string{
		`"description":""`,
		`"serviceLogGroupId":""`,
		`"authorizationDetails":""`,
		`"isScanningEnabled":false`,
		`"approvalKeyVaultId":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestGdpPipelineCreateOrUpdateReusesExistingPipelineWithExactBucketDetails(t *testing.T) {
	t.Parallel()

	resource := makeGdpPipelineResource()
	existing := makeSDKGdpPipeline("ocid1.gdppipeline.oc1..existing", resource, gdpsdk.GdpPipelineLifecycleStateActive)

	listCalls := 0
	getCalls := 0
	createCalls := 0

	client := newTestGdpPipelineClient(&fakeGdpPipelineOCIClient{
		listFn: func(_ context.Context, req gdpsdk.ListGdpPipelinesRequest) (gdpsdk.ListGdpPipelinesResponse, error) {
			listCalls++
			requireGdpPipelineStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireGdpPipelineStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.GdpPipelineId != nil {
				t.Fatalf("list gdpPipelineId = %#v, want nil for pre-create lookup", req.GdpPipelineId)
			}
			return gdpsdk.ListGdpPipelinesResponse{
				GdpPipelineCollection: gdpsdk.GdpPipelineCollection{
					Items: []gdpsdk.GdpPipelineSummary{makeGdpPipelineSummary(existing)},
				},
			}, nil
		},
		getFn: func(_ context.Context, req gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error) {
			getCalls++
			requireGdpPipelineStringPtr(t, "get gdpPipelineId", req.GdpPipelineId, "ocid1.gdppipeline.oc1..existing")
			return gdpsdk.GetGdpPipelineResponse{GdpPipeline: existing}, nil
		},
		createFn: func(_ context.Context, _ gdpsdk.CreateGdpPipelineRequest) (gdpsdk.CreateGdpPipelineResponse, error) {
			createCalls++
			return gdpsdk.CreateGdpPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful settled reuse", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListGdpPipelines() calls = %d, want 1", listCalls)
	}
	if getCalls < 1 {
		t.Fatalf("GetGdpPipeline() calls = %d, want at least 1", getCalls)
	}
	if createCalls != 0 {
		t.Fatalf("CreateGdpPipeline() calls = %d, want 0", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.gdppipeline.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing id", got)
	}
}

func TestGdpPipelineCreateOrUpdateCreatesWhenBucketDetailsDoNotMatchExpandedCandidate(t *testing.T) {
	t.Parallel()

	resource := makeGdpPipelineResource()
	mismatched := makeSDKGdpPipeline("ocid1.gdppipeline.oc1..existing", resource, gdpsdk.GdpPipelineLifecycleStateActive)
	mismatched.BucketDetails = []gdpsdk.BucketDetailsDefinition{
		{
			BucketType: gdpsdk.BucketDetailsDefinitionBucketTypeDestination,
			Namespace:  common.String("other-namespace"),
			Name:       common.String("other-bucket"),
			Id:         common.String("ocid1.bucket.oc1..other"),
		},
	}

	createCalls := 0

	client := newTestGdpPipelineClient(&fakeGdpPipelineOCIClient{
		listFn: func(_ context.Context, _ gdpsdk.ListGdpPipelinesRequest) (gdpsdk.ListGdpPipelinesResponse, error) {
			return gdpsdk.ListGdpPipelinesResponse{
				GdpPipelineCollection: gdpsdk.GdpPipelineCollection{
					Items: []gdpsdk.GdpPipelineSummary{makeGdpPipelineSummary(mismatched)},
				},
			}, nil
		},
		getFn: func(_ context.Context, req gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error) {
			requireGdpPipelineStringPtr(t, "get gdpPipelineId", req.GdpPipelineId, "ocid1.gdppipeline.oc1..existing")
			return gdpsdk.GetGdpPipelineResponse{GdpPipeline: mismatched}, nil
		},
		createFn: func(_ context.Context, req gdpsdk.CreateGdpPipelineRequest) (gdpsdk.CreateGdpPipelineResponse, error) {
			createCalls++
			requireGdpPipelineStringPtr(t, "create compartmentId", req.CreateGdpPipelineDetails.CompartmentId, resource.Spec.CompartmentId)
			return gdpsdk.CreateGdpPipelineResponse{
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req gdpsdk.GetGdpWorkRequestRequest) (gdpsdk.GetGdpWorkRequestResponse, error) {
			requireGdpPipelineStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-1")
			return gdpsdk.GetGdpWorkRequestResponse{
				GdpWorkRequest: makeGdpPipelineWorkRequest(
					"wr-create-1",
					gdpsdk.GdpOperationTypeCreateGdpPipeline,
					gdpsdk.OperationStatusInProgress,
					gdpsdk.ActionTypeCreated,
					"ocid1.gdppipeline.oc1..created",
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
		t.Fatalf("CreateGdpPipeline() calls = %d, want 1", createCalls)
	}
	requireGdpPipelineAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", shared.OSOKAsyncClassPending)
}

func TestGdpPipelineCreateOrUpdateUsesTrackedIDListFilterOnGetFallback(t *testing.T) {
	t.Parallel()

	resource := makeGdpPipelineResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.gdppipeline.oc1..tracked")

	var listRequests []gdpsdk.ListGdpPipelinesRequest

	client := newTestGdpPipelineClient(&fakeGdpPipelineOCIClient{
		getFn: func(_ context.Context, req gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error) {
			requireGdpPipelineStringPtr(t, "get gdpPipelineId", req.GdpPipelineId, "ocid1.gdppipeline.oc1..tracked")
			return gdpsdk.GetGdpPipelineResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
		},
		listFn: func(_ context.Context, req gdpsdk.ListGdpPipelinesRequest) (gdpsdk.ListGdpPipelinesResponse, error) {
			listRequests = append(listRequests, req)
			return gdpsdk.ListGdpPipelinesResponse{}, nil
		},
		createFn: func(_ context.Context, _ gdpsdk.CreateGdpPipelineRequest) (gdpsdk.CreateGdpPipelineResponse, error) {
			return gdpsdk.CreateGdpPipelineResponse{
				OpcWorkRequestId: common.String("wr-create-2"),
				OpcRequestId:     common.String("opc-create-2"),
			}, nil
		},
		workRequestFn: func(_ context.Context, _ gdpsdk.GetGdpWorkRequestRequest) (gdpsdk.GetGdpWorkRequestResponse, error) {
			return gdpsdk.GetGdpWorkRequestResponse{
				GdpWorkRequest: makeGdpPipelineWorkRequest(
					"wr-create-2",
					gdpsdk.GdpOperationTypeCreateGdpPipeline,
					gdpsdk.OperationStatusInProgress,
					gdpsdk.ActionTypeCreated,
					"ocid1.gdppipeline.oc1..created",
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
	foundTrackedLookup := false
	for _, req := range listRequests {
		if req.GdpPipelineId != nil && *req.GdpPipelineId == "ocid1.gdppipeline.oc1..tracked" {
			foundTrackedLookup = true
			break
		}
	}
	if !foundTrackedLookup {
		t.Fatalf("ListGdpPipelines() requests = %#v, want one request with gdpPipelineId set to the tracked OCID", listRequests)
	}
}

func TestGdpPipelineLifecycleClassificationHandlesInactiveAndNeedsAttention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		lifecycle     gdpsdk.GdpPipelineLifecycleStateEnum
		wantSuccess   bool
		wantRequeue   bool
		wantCondition string
	}{
		{
			name:          "inactive is steady state success",
			lifecycle:     gdpsdk.GdpPipelineLifecycleStateInactive,
			wantSuccess:   true,
			wantRequeue:   false,
			wantCondition: string(shared.Active),
		},
		{
			name:          "needs attention is terminal failure",
			lifecycle:     gdpsdk.GdpPipelineLifecycleStateNeedsAttention,
			wantSuccess:   false,
			wantRequeue:   false,
			wantCondition: string(shared.Failed),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := makeGdpPipelineResource()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.gdppipeline.oc1..tracked")

			client := newTestGdpPipelineClient(&fakeGdpPipelineOCIClient{
				getFn: func(_ context.Context, req gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error) {
					requireGdpPipelineStringPtr(t, "get gdpPipelineId", req.GdpPipelineId, "ocid1.gdppipeline.oc1..tracked")
					return gdpsdk.GetGdpPipelineResponse{
						GdpPipeline: makeSDKGdpPipeline("ocid1.gdppipeline.oc1..tracked", resource, tc.lifecycle),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccess {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccess)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}

			condition := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1]
			if condition.Type != shared.OSOKConditionType(tc.wantCondition) {
				t.Fatalf("latest condition = %q, want %q", condition.Type, tc.wantCondition)
			}
			if resource.Status.LifecycleState != string(tc.lifecycle) {
				t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.lifecycle)
			}
		})
	}
}

func makeGdpPipelineResource() *gdpv1beta1.GdpPipeline {
	return &gdpv1beta1.GdpPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gdppipeline-sample",
			Namespace: "default",
		},
		Spec: gdpv1beta1.GdpPipelineSpec{
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
			DisplayName:   "osok-gdp-pipeline",
			PipelineType:  "SENDER",
			BucketDetails: []gdpv1beta1.GdpPipelineBucketDetail{
				{
					BucketType: "SOURCE",
					Namespace:  "namespace-a",
					Name:       "source-bucket",
					Id:         "ocid1.bucket.oc1..source",
				},
				{
					BucketType: "TRANSFER",
					Namespace:  "namespace-a",
					Name:       "transfer-bucket",
					Id:         "ocid1.bucket.oc1..transfer",
				},
			},
			PeeringRegion:                      "us-phoenix-1",
			Description:                        "pipeline description",
			ServiceLogGroupId:                  "ocid1.loggroup.oc1..exampleuniqueID",
			FileTypes:                          []string{".pdf", ".xml"},
			AuthorizationDetails:               "authorization details",
			IsFileOverrideInDestinationEnabled: true,
			IsScanningEnabled:                  true,
			IsChunkingEnabled:                  true,
			IsApprovalNeeded:                   true,
			ApprovalKeyVaultId:                 "ocid1.vault.oc1..exampleuniqueID",
			FreeformTags: map[string]string{
				"managed-by": "oci-service-operator",
			},
			DefinedTags: map[string]shared.MapValue{
				"operations": {
					"owner": "team-a",
				},
			},
		},
	}
}

func makeSDKGdpPipeline(
	id string,
	resource *gdpv1beta1.GdpPipeline,
	lifecycle gdpsdk.GdpPipelineLifecycleStateEnum,
) gdpsdk.GdpPipeline {
	if resource == nil {
		resource = makeGdpPipelineResource()
	}

	created := common.SDKTime{Time: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 5, 6, 13, 0, 0, 0, time.UTC)}
	return gdpsdk.GdpPipeline{
		Id:                                 common.String(id),
		CompartmentId:                      common.String(resource.Spec.CompartmentId),
		LifecycleState:                     lifecycle,
		DisplayName:                        common.String(resource.Spec.DisplayName),
		PipelineType:                       gdpsdk.GdpPipelinePipelineTypeEnum(resource.Spec.PipelineType),
		PeeringRegion:                      common.String(resource.Spec.PeeringRegion),
		TimeCreated:                        &created,
		TimeUpdated:                        &updated,
		FreeformTags:                       cloneGdpPipelineStringMap(resource.Spec.FreeformTags),
		DefinedTags:                        gdpPipelineDefinedTagsFromSpec(resource.Spec.DefinedTags),
		LifecycleDetails:                   common.String("lifecycle details"),
		Description:                        common.String(resource.Spec.Description),
		ServiceLogGroupId:                  common.String(resource.Spec.ServiceLogGroupId),
		FileTypes:                          append([]string(nil), resource.Spec.FileTypes...),
		AuthorizationDetails:               common.String(resource.Spec.AuthorizationDetails),
		BucketDetails:                      sdkBucketDetailsFromSpec(resource.Spec.BucketDetails),
		PeeredGdpPipelineId:                common.String("ocid1.gdppipeline.oc1..peer"),
		IsFileOverrideInDestinationEnabled: common.Bool(resource.Spec.IsFileOverrideInDestinationEnabled),
		IsScanningEnabled:                  common.Bool(resource.Spec.IsScanningEnabled),
		IsChunkingEnabled:                  common.Bool(resource.Spec.IsChunkingEnabled),
		IsApprovalNeeded:                   common.Bool(resource.Spec.IsApprovalNeeded),
		ApprovalKeyVaultId:                 common.String(resource.Spec.ApprovalKeyVaultId),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeGdpPipelineSummary(pipeline gdpsdk.GdpPipeline) gdpsdk.GdpPipelineSummary {
	return gdpsdk.GdpPipelineSummary{
		Id:                                 pipeline.Id,
		CompartmentId:                      pipeline.CompartmentId,
		LifecycleState:                     pipeline.LifecycleState,
		DisplayName:                        pipeline.DisplayName,
		PipelineType:                       pipeline.PipelineType,
		PeeringRegion:                      pipeline.PeeringRegion,
		TimeCreated:                        pipeline.TimeCreated,
		TimeUpdated:                        pipeline.TimeUpdated,
		FreeformTags:                       pipeline.FreeformTags,
		DefinedTags:                        pipeline.DefinedTags,
		Description:                        pipeline.Description,
		ServiceLogGroupId:                  pipeline.ServiceLogGroupId,
		AuthorizationDetails:               pipeline.AuthorizationDetails,
		IsFileOverrideInDestinationEnabled: pipeline.IsFileOverrideInDestinationEnabled,
		IsScanningEnabled:                  pipeline.IsScanningEnabled,
		IsChunkingEnabled:                  pipeline.IsChunkingEnabled,
		IsApprovalNeeded:                   pipeline.IsApprovalNeeded,
		SystemTags:                         pipeline.SystemTags,
	}
}

func makeGdpPipelineWorkRequest(
	workRequestID string,
	operationType gdpsdk.GdpOperationTypeEnum,
	status gdpsdk.OperationStatusEnum,
	action gdpsdk.ActionTypeEnum,
	resourceID string,
) gdpsdk.GdpWorkRequest {
	accepted := common.SDKTime{Time: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	percent := float32(42)
	return gdpsdk.GdpWorkRequest{
		OperationType: operationType,
		Status:        status,
		Id:            common.String(workRequestID),
		CompartmentId: common.String("ocid1.compartment.oc1..exampleuniqueID"),
		Resources: []gdpsdk.WorkRequestResource{
			{
				EntityType: common.String("GdpPipeline"),
				ActionType: action,
				Identifier: common.String(resourceID),
			},
		},
		PercentComplete: &percent,
		TimeAccepted:    &accepted,
	}
}

func sdkBucketDetailsFromSpec(spec []gdpv1beta1.GdpPipelineBucketDetail) []gdpsdk.BucketDetailsDefinition {
	if spec == nil {
		return nil
	}
	out := make([]gdpsdk.BucketDetailsDefinition, 0, len(spec))
	for _, detail := range spec {
		out = append(out, gdpsdk.BucketDetailsDefinition{
			BucketType: gdpsdk.BucketDetailsDefinitionBucketTypeEnum(detail.BucketType),
			Namespace:  common.String(detail.Namespace),
			Name:       common.String(detail.Name),
			Id:         common.String(detail.Id),
		})
	}
	return out
}

func gdpPipelineSerializedRequestBody(
	t *testing.T,
	builder gdpPipelineRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	req, err := builder.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}
	if req.Body == nil {
		return ""
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(body)
}

func requireGdpPipelineStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %#v, want %q", label, got, want)
	}
}

func requireGdpPipelineBoolPtr(t *testing.T, label string, got *bool, want bool) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %#v, want %t", label, got, want)
	}
}

func requireGdpPipelineAsyncCurrent(
	t *testing.T,
	resource *gdpv1beta1.GdpPipeline,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
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

func assertGdpPipelineStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("%s = %#v, want %#v", label, got, want)
		}
	}
}
