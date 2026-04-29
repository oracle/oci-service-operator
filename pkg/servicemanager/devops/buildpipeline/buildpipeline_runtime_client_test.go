/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package buildpipeline

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testBuildPipelineID = "ocid1.devopsbuildpipeline.oc1..example"
	testProjectID       = "ocid1.devopsproject.oc1..example"
	testCompartmentID   = "ocid1.compartment.oc1..example"
)

type fakeBuildPipelineOCIClient struct {
	createBuildPipelineFn func(context.Context, devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error)
	getBuildPipelineFn    func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error)
	listBuildPipelinesFn  func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error)
	updateBuildPipelineFn func(context.Context, devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error)
	deleteBuildPipelineFn func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error)
	getWorkRequestFn      func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

func (f *fakeBuildPipelineOCIClient) CreateBuildPipeline(ctx context.Context, req devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
	if f.createBuildPipelineFn != nil {
		return f.createBuildPipelineFn(ctx, req)
	}
	return devopssdk.CreateBuildPipelineResponse{}, nil
}

func (f *fakeBuildPipelineOCIClient) GetBuildPipeline(ctx context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
	if f.getBuildPipelineFn != nil {
		return f.getBuildPipelineFn(ctx, req)
	}
	return devopssdk.GetBuildPipelineResponse{}, nil
}

func (f *fakeBuildPipelineOCIClient) ListBuildPipelines(ctx context.Context, req devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
	if f.listBuildPipelinesFn != nil {
		return f.listBuildPipelinesFn(ctx, req)
	}
	return devopssdk.ListBuildPipelinesResponse{}, nil
}

func (f *fakeBuildPipelineOCIClient) UpdateBuildPipeline(ctx context.Context, req devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
	if f.updateBuildPipelineFn != nil {
		return f.updateBuildPipelineFn(ctx, req)
	}
	return devopssdk.UpdateBuildPipelineResponse{}, nil
}

func (f *fakeBuildPipelineOCIClient) DeleteBuildPipeline(ctx context.Context, req devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
	if f.deleteBuildPipelineFn != nil {
		return f.deleteBuildPipelineFn(ctx, req)
	}
	return devopssdk.DeleteBuildPipelineResponse{}, nil
}

func (f *fakeBuildPipelineOCIClient) GetWorkRequest(ctx context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return devopssdk.GetWorkRequestResponse{}, nil
}

func testBuildPipelineClient(fake *fakeBuildPipelineOCIClient) BuildPipelineServiceClient {
	return newBuildPipelineServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeBuildPipelineResource() *devopsv1beta1.BuildPipeline {
	return &devopsv1beta1.BuildPipeline{
		Spec: devopsv1beta1.BuildPipelineSpec{
			ProjectId:   testProjectID,
			DisplayName: "pipeline-alpha",
			Description: "desired description",
			BuildPipelineParameters: devopsv1beta1.BuildPipelineParameters{
				Items: []devopsv1beta1.BuildPipelineParametersItem{
					{Name: "IMAGE_TAG", DefaultValue: "latest", Description: "image tag"},
				},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKBuildPipeline(
	id string,
	projectID string,
	displayName string,
	description string,
	state devopssdk.BuildPipelineLifecycleStateEnum,
) devopssdk.BuildPipeline {
	buildPipeline := devopssdk.BuildPipeline{
		Id:             common.String(id),
		CompartmentId:  common.String(testCompartmentID),
		ProjectId:      common.String(projectID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		BuildPipelineParameters: &devopssdk.BuildPipelineParameterCollection{Items: []devopssdk.BuildPipelineParameter{
			{Name: common.String("IMAGE_TAG"), DefaultValue: common.String("latest"), Description: common.String("image tag")},
		}},
		FreeformTags: map[string]string{"env": "dev"},
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:   map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
	if description != "" {
		buildPipeline.Description = common.String(description)
	}
	return buildPipeline
}

func makeSDKBuildPipelineSummary(
	id string,
	projectID string,
	displayName string,
	description string,
	state devopssdk.BuildPipelineLifecycleStateEnum,
) devopssdk.BuildPipelineSummary {
	summary := devopssdk.BuildPipelineSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(testCompartmentID),
		ProjectId:      common.String(projectID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		BuildPipelineParameters: &devopssdk.BuildPipelineParameterCollection{Items: []devopssdk.BuildPipelineParameter{
			{Name: common.String("IMAGE_TAG"), DefaultValue: common.String("latest"), Description: common.String("image tag")},
		}},
		FreeformTags: map[string]string{"env": "dev"},
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	if description != "" {
		summary.Description = common.String(description)
	}
	return summary
}

func makeBuildPipelineWorkRequest(
	id string,
	operationType devopssdk.OperationTypeEnum,
	status devopssdk.OperationStatusEnum,
	action devopssdk.ActionTypeEnum,
	buildPipelineID string,
) devopssdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := devopssdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if buildPipelineID != "" {
		workRequest.Resources = []devopssdk.WorkRequestResource{
			{
				EntityType: common.String("buildPipeline"),
				ActionType: action,
				Identifier: common.String(buildPipelineID),
				EntityUri:  common.String("/20210630/buildPipelines/" + buildPipelineID),
			},
		}
	}
	return workRequest
}

func TestBuildPipelineRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedBuildPipelineRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedBuildPipelineRuntimeSemantics() = nil")
	}
	if got.FormalService != "devops" || got.FormalSlug != "buildpipeline" {
		t.Fatalf("formal binding = %s/%s, want devops/buildpipeline", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertBuildPipelineStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"projectId", "displayName", "id"})
	assertBuildPipelineStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"description", "displayName", "buildPipelineParameters", "freeformTags", "definedTags"})
	assertBuildPipelineStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"projectId"})
}

func TestBuildPipelineServiceClientCreatesThroughWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	var createRequest devopssdk.CreateBuildPipelineRequest
	listCalls := 0
	createCalls := 0
	getWorkRequestCalls := 0
	getCalls := 0

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(_ context.Context, req devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			listCalls++
			requireStringPtr(t, "list projectId", req.ProjectId, testProjectID)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return devopssdk.ListBuildPipelinesResponse{}, nil
		},
		createBuildPipelineFn: func(_ context.Context, req devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
			createCalls++
			createRequest = req
			return devopssdk.CreateBuildPipelineResponse{
				BuildPipeline:    makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeBuildPipelineWorkRequest("wr-create-1", devopssdk.OperationTypeCreateBuildPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, testBuildPipelineID),
			}, nil
		},
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 1 || createCalls != 1 || getWorkRequestCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/getWR/get = %d/%d/%d/%d, want 1/1/1/1", listCalls, createCalls, getWorkRequestCalls, getCalls)
	}
	assertBuildPipelineCreateRequest(t, createRequest, resource)
	assertBuildPipelineCreatedStatus(t, resource)
}

func assertBuildPipelineCreateRequest(
	t *testing.T,
	createRequest devopssdk.CreateBuildPipelineRequest,
	resource *devopsv1beta1.BuildPipeline,
) {
	t.Helper()

	requireStringPtr(t, "create projectId", createRequest.ProjectId, testProjectID)
	requireStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	items := requireBuildPipelineParameterItems(t, "create buildPipelineParameters", createRequest.BuildPipelineParameters)
	requireStringPtr(t, "create parameter name", items[0].Name, "IMAGE_TAG")
}

func assertBuildPipelineCreatedStatus(t *testing.T, resource *devopsv1beta1.BuildPipeline) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != testBuildPipelineID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testBuildPipelineID)
	}
	if got := resource.Status.Id; got != testBuildPipelineID {
		t.Fatalf("status.id = %q, want %q", got, testBuildPipelineID)
	}
	if got := resource.Status.BuildPipelineParameters.Items[0].DefaultValue; got != "latest" {
		t.Fatalf("status.buildPipelineParameters.items[0].defaultValue = %q, want latest", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed create", resource.Status.OsokStatus.Async.Current)
	}
	if got := lastBuildPipelineConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestBuildPipelineServiceClientTracksPendingCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			return devopssdk.ListBuildPipelinesResponse{}, nil
		},
		createBuildPipelineFn: func(context.Context, devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
			return devopssdk.CreateBuildPipelineResponse{
				BuildPipeline:    makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-pending"),
				OpcRequestId:     common.String("opc-create-pending"),
			}, nil
		},
		getWorkRequestFn: func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeBuildPipelineWorkRequest("wr-create-pending", devopssdk.OperationTypeCreateBuildPipeline, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, testBuildPipelineID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	current := requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-pending")
	if current.RawStatus != string(devopssdk.OperationStatusInProgress) {
		t.Fatalf("status.async.current.rawStatus = %q, want IN_PROGRESS", current.RawStatus)
	}
	if current.RawOperationType != string(devopssdk.OperationTypeCreateBuildPipeline) {
		t.Fatalf("status.async.current.rawOperationType = %q, want CREATE_BUILD_PIPELINE", current.RawOperationType)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-pending" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-pending", got)
	}
}

func TestBuildPipelineServiceClientSkipsPreCreateBindWithoutDisplayName(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Spec.DisplayName = ""
	listCalled := false
	createCalled := false

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			listCalled = true
			return devopssdk.ListBuildPipelinesResponse{}, nil
		},
		createBuildPipelineFn: func(context.Context, devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
			createCalled = true
			return devopssdk.CreateBuildPipelineResponse{
				BuildPipeline:    makeSDKBuildPipeline(testBuildPipelineID, testProjectID, "", resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-no-display-name"),
			}, nil
		},
		getWorkRequestFn: func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeBuildPipelineWorkRequest("wr-create-no-display-name", devopssdk.OperationTypeCreateBuildPipeline, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, testBuildPipelineID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if listCalled {
		t.Fatal("ListBuildPipelines() was called without a stable displayName identity")
	}
	if !createCalled {
		t.Fatal("CreateBuildPipeline() was not called")
	}
}

func TestBuildPipelineServiceClientBindsExistingFromPagedList(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	pages := []string{}
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(_ context.Context, req devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			pages = append(pages, stringValue(req.Page))
			requireStringPtr(t, "list projectId", req.ProjectId, testProjectID)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.Page == nil {
				return devopssdk.ListBuildPipelinesResponse{
					BuildPipelineCollection: devopssdk.BuildPipelineCollection{Items: []devopssdk.BuildPipelineSummary{
						makeSDKBuildPipelineSummary("ocid1.devopsbuildpipeline.oc1..other", testProjectID, "other-pipeline", "other", devopssdk.BuildPipelineLifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return devopssdk.ListBuildPipelinesResponse{
				BuildPipelineCollection: devopssdk.BuildPipelineCollection{Items: []devopssdk.BuildPipelineSummary{
					makeSDKBuildPipelineSummary(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
				}},
			}, nil
		},
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		},
		createBuildPipelineFn: func(context.Context, devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
			createCalled = true
			return devopssdk.CreateBuildPipelineResponse{}, nil
		},
		updateBuildPipelineFn: func(context.Context, devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
			updateCalled = true
			return devopssdk.UpdateBuildPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalled || updateCalled {
		t.Fatalf("create/update called = %v/%v, want neither for bind", createCalled, updateCalled)
	}
	if !reflect.DeepEqual(pages, []string{"", "page-2"}) {
		t.Fatalf("list pages = %#v, want first page then page-2", pages)
	}
	if getCalls != 1 {
		t.Fatalf("GetBuildPipeline() calls = %d, want 1 live mutation assessment read", getCalls)
	}
	if got := resource.Status.Id; got != testBuildPipelineID {
		t.Fatalf("status.id = %q, want %q", got, testBuildPipelineID)
	}
}

func TestBuildPipelineServiceClientNoopsWhenTrackedStateMatchesSpec(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)
	getCalls := 0
	updateCalled := false

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		},
		updateBuildPipelineFn: func(context.Context, devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
			updateCalled = true
			return devopssdk.UpdateBuildPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue no-op", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetBuildPipeline() calls = %d, want one observe read", getCalls)
	}
	if updateCalled {
		t.Fatal("UpdateBuildPipeline() was called for matching observed state")
	}
	if got := resource.Status.Id; got != testBuildPipelineID {
		t.Fatalf("status.id = %q, want %q", got, testBuildPipelineID)
	}
	if got := lastBuildPipelineConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestBuildPipelineServiceClientPreservesOmittedOptionalUpdateFields(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)
	resource.Spec.Description = ""
	resource.Spec.DisplayName = ""
	resource.Spec.BuildPipelineParameters = devopsv1beta1.BuildPipelineParameters{}
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil

	getCalls := 0
	updateCalled := false
	current := makeSDKBuildPipeline(testBuildPipelineID, testProjectID, "existing-display", "existing description", devopssdk.BuildPipelineLifecycleStateActive)
	current.FreeformTags = map[string]string{"owner": "platform"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{BuildPipeline: current}, nil
		},
		updateBuildPipelineFn: func(context.Context, devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
			updateCalled = true
			t.Fatal("UpdateBuildPipeline() was called when optional fields and tags were omitted from the spec")
			return devopssdk.UpdateBuildPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue no-op", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetBuildPipeline() calls = %d, want one observe read", getCalls)
	}
	if updateCalled {
		t.Fatal("UpdateBuildPipeline() was called for omitted optional fields")
	}
	if resource.Status.Description != "existing description" {
		t.Fatalf("status.description = %q, want existing description", resource.Status.Description)
	}
	if resource.Status.DisplayName != "existing-display" {
		t.Fatalf("status.displayName = %q, want existing-display", resource.Status.DisplayName)
	}
	if !reflect.DeepEqual(resource.Status.FreeformTags, map[string]string{"owner": "platform"}) {
		t.Fatalf("status.freeformTags = %#v, want owner=platform", resource.Status.FreeformTags)
	}
	if got := resource.Status.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("status.definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func TestBuildPipelineServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)
	resource.Spec.Description = "new description"
	resource.Spec.DisplayName = "pipeline-renamed"
	resource.Spec.BuildPipelineParameters.Items = []devopsv1beta1.BuildPipelineParametersItem{
		{Name: "IMAGE_TAG", DefaultValue: "stable", Description: "updated tag"},
	}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	var updateRequest devopssdk.UpdateBuildPipelineRequest
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getBuildPipelineFn: mutableBuildPipelineReadback(t, resource, &getCalls),
		updateBuildPipelineFn: func(_ context.Context, req devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
			updateRequest = req
			return devopssdk.UpdateBuildPipelineResponse{
				BuildPipeline:    makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateUpdating),
				OpcWorkRequestId: common.String("wr-update-1"),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-update-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeBuildPipelineWorkRequest("wr-update-1", devopssdk.OperationTypeUpdateBuildPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeUpdated, testBuildPipelineID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update", response)
	}
	if getCalls != 2 {
		t.Fatalf("GetBuildPipeline() calls = %d, want pre-update read and post-work-request read", getCalls)
	}
	requireStringPtr(t, "update buildPipelineId", updateRequest.BuildPipelineId, testBuildPipelineID)
	assertBuildPipelineUpdateRequest(t, updateRequest, resource)
	assertBuildPipelineUpdatedStatus(t, resource)
}

func mutableBuildPipelineReadback(
	t *testing.T,
	resource *devopsv1beta1.BuildPipeline,
	getCalls *int,
) func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
	t.Helper()

	return func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
		(*getCalls)++
		requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
		if *getCalls == 1 {
			current := makeSDKBuildPipeline(testBuildPipelineID, testProjectID, "pipeline-alpha", "old description", devopssdk.BuildPipelineLifecycleStateActive)
			current.FreeformTags = map[string]string{"env": "dev"}
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
			return devopssdk.GetBuildPipelineResponse{BuildPipeline: current}, nil
		}

		current := makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive)
		current.BuildPipelineParameters = &devopssdk.BuildPipelineParameterCollection{Items: []devopssdk.BuildPipelineParameter{
			{Name: common.String("IMAGE_TAG"), DefaultValue: common.String("stable"), Description: common.String("updated tag")},
		}}
		current.FreeformTags = map[string]string{"env": "prod"}
		current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
		return devopssdk.GetBuildPipelineResponse{BuildPipeline: current}, nil
	}
}

func assertBuildPipelineUpdateRequest(
	t *testing.T,
	updateRequest devopssdk.UpdateBuildPipelineRequest,
	resource *devopsv1beta1.BuildPipeline,
) {
	t.Helper()

	requireStringPtr(t, "update description", updateRequest.Description, resource.Spec.Description)
	requireStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	items := requireBuildPipelineParameterItems(t, "update buildPipelineParameters", updateRequest.BuildPipelineParameters)
	requireStringPtr(t, "update parameter defaultValue", items[0].DefaultValue, "stable")
	if !reflect.DeepEqual(updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
}

func assertBuildPipelineUpdatedStatus(t *testing.T, resource *devopsv1beta1.BuildPipeline) {
	t.Helper()

	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.BuildPipelineParameters.Items[0].DefaultValue; got != "stable" {
		t.Fatalf("status.buildPipelineParameters.items[0].defaultValue = %q, want stable", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestBuildPipelineServiceClientRejectsForceNewDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)
	resource.Spec.ProjectId = "ocid1.devopsproject.oc1..other"
	updateCalled := false

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getBuildPipelineFn: func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		},
		updateBuildPipelineFn: func(context.Context, devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
			updateCalled = true
			return devopssdk.UpdateBuildPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want force-new drift error")
	}
	if updateCalled {
		t.Fatal("UpdateBuildPipeline() was called for force-new projectId drift")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "projectId") {
		t.Fatalf("CreateOrUpdate() error = %v, want projectId drift detail", err)
	}
	if got := lastBuildPipelineConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestBuildPipelineServiceClientDeleteRetainsFinalizerUntilReadbackConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)
	getCalls := 0
	deleteCalls := 0

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getBuildPipelineFn: deleteBuildPipelineReadback(t, resource, &getCalls),
		deleteBuildPipelineFn: func(_ context.Context, req devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.DeleteBuildPipelineResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeBuildPipelineWorkRequest("wr-delete-1", devopssdk.OperationTypeDeleteBuildPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeDeleted, testBuildPipelineID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertBuildPipelineDeletePending(t, deleted, resource)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	assertBuildPipelineDeleteConfirmed(t, deleted, resource)
	if deleteCalls != 1 {
		t.Fatalf("DeleteBuildPipeline() calls = %d, want 1", deleteCalls)
	}
}

type buildPipelineWriteWorkRequestDeleteCase struct {
	name   string
	phase  shared.OSOKAsyncPhase
	op     devopssdk.OperationTypeEnum
	action devopssdk.ActionTypeEnum
}

func buildPipelineWriteWorkRequestDeleteCases() []buildPipelineWriteWorkRequestDeleteCase {
	return []buildPipelineWriteWorkRequestDeleteCase{
		{
			name:   "create",
			phase:  shared.OSOKAsyncPhaseCreate,
			op:     devopssdk.OperationTypeCreateBuildPipeline,
			action: devopssdk.ActionTypeCreated,
		},
		{
			name:   "update",
			phase:  shared.OSOKAsyncPhaseUpdate,
			op:     devopssdk.OperationTypeUpdateBuildPipeline,
			action: devopssdk.ActionTypeUpdated,
		},
	}
}

func (tc buildPipelineWriteWorkRequestDeleteCase) workRequestID() string {
	return "wr-" + tc.name
}

func (tc buildPipelineWriteWorkRequestDeleteCase) pendingWorkRequest() devopssdk.WorkRequest {
	return makeBuildPipelineWorkRequest(
		tc.workRequestID(),
		tc.op,
		devopssdk.OperationStatusInProgress,
		devopssdk.ActionTypeInProgress,
		testBuildPipelineID,
	)
}

func (tc buildPipelineWriteWorkRequestDeleteCase) succeededWorkRequest() devopssdk.WorkRequest {
	return makeBuildPipelineWorkRequest(
		tc.workRequestID(),
		tc.op,
		devopssdk.OperationStatusSucceeded,
		tc.action,
		testBuildPipelineID,
	)
}

func TestBuildPipelineServiceClientDeleteWaitsForPendingCreateOrUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range buildPipelineWriteWorkRequestDeleteCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireBuildPipelineDeleteWaitsForPendingWriteWorkRequest(t, tc)
		})
	}
}

func requireBuildPipelineDeleteWaitsForPendingWriteWorkRequest(
	t *testing.T,
	tc buildPipelineWriteWorkRequestDeleteCase,
) {
	t.Helper()

	resource := makeBuildPipelineResource()
	seedBuildPipelineCurrentWorkRequest(resource, tc.phase, tc.workRequestID())
	deleteCalled := false
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getWorkRequestFn: buildPipelineWorkRequestLookup(t, map[string]devopssdk.WorkRequest{
			tc.workRequestID(): tc.pendingWorkRequest(),
		}),
		deleteBuildPipelineFn: func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteBuildPipeline() should not be called before the write work request finishes")
			return devopssdk.DeleteBuildPipelineResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while write work request is pending")
	}
	if deleteCalled {
		t.Fatal("DeleteBuildPipeline() was called before the write work request finished")
	}
	current := requireAsyncCurrent(t, resource, tc.phase, tc.workRequestID())
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if !strings.Contains(current.Message, "waiting before delete") {
		t.Fatalf("status.async.current.message = %q, want waiting-before-delete detail", current.Message)
	}
}

func TestBuildPipelineServiceClientDeleteStartsAfterCreateOrUpdateWorkRequestSucceeds(t *testing.T) {
	t.Parallel()

	for _, tc := range buildPipelineWriteWorkRequestDeleteCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireBuildPipelineDeleteStartsAfterSucceededWriteWorkRequest(t, tc)
		})
	}
}

func requireBuildPipelineDeleteStartsAfterSucceededWriteWorkRequest(
	t *testing.T,
	tc buildPipelineWriteWorkRequestDeleteCase,
) {
	t.Helper()

	resource := makeBuildPipelineResource()
	seedBuildPipelineCurrentWorkRequest(resource, tc.phase, tc.workRequestID())
	deleteCalls := 0
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getWorkRequestFn: buildPipelineWorkRequestLookup(t, map[string]devopssdk.WorkRequest{
			tc.workRequestID(): tc.succeededWorkRequest(),
			"wr-delete-1":      makeBuildPipelineWorkRequest("wr-delete-1", devopssdk.OperationTypeDeleteBuildPipeline, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, testBuildPipelineID),
		}),
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		},
		deleteBuildPipelineFn: func(_ context.Context, req devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.DeleteBuildPipelineResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteBuildPipeline() calls = %d, want 1 after write readback succeeds", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
	current := requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("delete status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
}

func TestBuildPipelineServiceClientDeleteWaitsWhenSucceededWriteReadbackIsNotVisible(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	seedBuildPipelineCurrentWorkRequest(resource, shared.OSOKAsyncPhaseCreate, "wr-create")
	deleteCalled := false
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getWorkRequestFn: buildPipelineWorkRequestLookup(t, map[string]devopssdk.WorkRequest{
			"wr-create": makeBuildPipelineWorkRequest("wr-create", devopssdk.OperationTypeCreateBuildPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, testBuildPipelineID),
		}),
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "build pipeline not yet readable")
		},
		deleteBuildPipelineFn: func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteBuildPipeline() should not be called before succeeded write readback is visible")
			return devopssdk.DeleteBuildPipelineResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while write readback is not visible")
	}
	if deleteCalled {
		t.Fatal("DeleteBuildPipeline() was called before write readback was visible")
	}
	current := requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create")
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testBuildPipelineID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testBuildPipelineID)
	}
}

func TestBuildPipelineServiceClientDeleteKeepsAuthShapedWriteReadbackFatal(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	seedBuildPipelineCurrentWorkRequest(resource, shared.OSOKAsyncPhaseCreate, "wr-create")
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getWorkRequestFn: buildPipelineWorkRequestLookup(t, map[string]devopssdk.WorkRequest{
			"wr-create": makeBuildPipelineWorkRequest("wr-create", devopssdk.OperationTypeCreateBuildPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, testBuildPipelineID),
		}),
		getBuildPipelineFn: func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
			return devopssdk.GetBuildPipelineResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteBuildPipelineFn: func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			t.Fatal("DeleteBuildPipeline() should not be called after auth-shaped write readback")
			return devopssdk.DeleteBuildPipelineResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 detail", err)
	}
}

func deleteBuildPipelineReadback(
	t *testing.T,
	resource *devopsv1beta1.BuildPipeline,
	getCalls *int,
) func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
	t.Helper()

	return func(_ context.Context, req devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
		(*getCalls)++
		requireStringPtr(t, "get buildPipelineId", req.BuildPipelineId, testBuildPipelineID)
		switch *getCalls {
		case 1:
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		case 2:
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateDeleting),
			}, nil
		default:
			return devopssdk.GetBuildPipelineResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "BuildPipeline deleted")
		}
	}
}

func assertBuildPipelineDeletePending(t *testing.T, deleted bool, resource *devopsv1beta1.BuildPipeline) {
	t.Helper()

	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback still reports DELETING")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil before confirmed deletion", resource.Status.OsokStatus.DeletedAt)
	}
	if got := lastBuildPipelineConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
}

func assertBuildPipelineDeleteConfirmed(t *testing.T, deleted bool, resource *devopsv1beta1.BuildPipeline) {
	t.Helper()

	if !deleted {
		t.Fatal("second Delete() deleted = false, want true after readback NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestBuildPipelineServiceClientDeleteWithoutTrackedIDAndBlankDisplayNameConfirmsDeleted(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Spec.DisplayName = ""
	listCalled := false
	deleteCalled := false

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			listCalled = true
			t.Fatal("ListBuildPipelines() should not be called without a stable displayName identity")
			return devopssdk.ListBuildPipelinesResponse{}, nil
		},
		deleteBuildPipelineFn: func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteBuildPipeline() should not be called without a resolved build pipeline OCID")
			return devopssdk.DeleteBuildPipelineResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release when no OCI identity can be resolved")
	}
	if listCalled {
		t.Fatal("ListBuildPipelines() was called without a stable displayName identity")
	}
	if deleteCalled {
		t.Fatal("DeleteBuildPipeline() was called without a resolved build pipeline OCID")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if resource.Status.OsokStatus.Ocid != "" || resource.Status.Id != "" {
		t.Fatalf("tracked identity = status.ocid %q status.id %q, want both empty", resource.Status.OsokStatus.Ocid, resource.Status.Id)
	}
}

func TestBuildPipelineServiceClientDeleteWithoutTrackedIDRejectsDuplicateListMatches(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	deleteCalled := false

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(_ context.Context, req devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			requireStringPtr(t, "list projectId", req.ProjectId, testProjectID)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return devopssdk.ListBuildPipelinesResponse{
				BuildPipelineCollection: devopssdk.BuildPipelineCollection{Items: []devopssdk.BuildPipelineSummary{
					makeSDKBuildPipelineSummary("ocid1.devopsbuildpipeline.oc1..one", testProjectID, resource.Spec.DisplayName, "one", devopssdk.BuildPipelineLifecycleStateActive),
					makeSDKBuildPipelineSummary("ocid1.devopsbuildpipeline.oc1..two", testProjectID, resource.Spec.DisplayName, "two", devopssdk.BuildPipelineLifecycleStateActive),
				}},
			}, nil
		},
		deleteBuildPipelineFn: func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteBuildPipeline() should not be called when untracked list resolution is ambiguous")
			return devopssdk.DeleteBuildPipelineResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want duplicate-match error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true for ambiguous duplicate matches")
	}
	if deleteCalled {
		t.Fatal("DeleteBuildPipeline() was called for ambiguous duplicate matches")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil for ambiguous duplicate matches", resource.Status.OsokStatus.DeletedAt)
	}
	if !strings.Contains(err.Error(), "found 2 build pipelines") {
		t.Fatalf("Delete() error = %v, want duplicate-match detail", err)
	}
}

func TestBuildPipelineServiceClientDeleteTreatsAuthShapedNotFoundAsError(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)

	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		getBuildPipelineFn: func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
			return devopssdk.GetBuildPipelineResponse{
				BuildPipeline: makeSDKBuildPipeline(testBuildPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.BuildPipelineLifecycleStateActive),
			}, nil
		},
		deleteBuildPipelineFn: func(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
			return devopssdk.DeleteBuildPipelineResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
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
		t.Fatalf("status.status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 detail", err)
	}
}

func TestBuildPipelineServiceClientRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := makeBuildPipelineResource()
	client := testBuildPipelineClient(&fakeBuildPipelineOCIClient{
		listBuildPipelinesFn: func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
			return devopssdk.ListBuildPipelinesResponse{}, nil
		},
		createBuildPipelineFn: func(context.Context, devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
			return devopssdk.CreateBuildPipelineResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if got := lastBuildPipelineConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func requireAsyncCurrent(t *testing.T, resource *devopsv1beta1.BuildPipeline, phase shared.OSOKAsyncPhase, workRequestID string) *shared.OSOKAsyncOperation {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want active async tracker")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	return current
}

func seedBuildPipelineCurrentWorkRequest(
	resource *devopsv1beta1.BuildPipeline,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	if resource == nil {
		return
	}
	resource.Status.Id = testBuildPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testBuildPipelineID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func buildPipelineWorkRequestLookup(
	t *testing.T,
	workRequests map[string]devopssdk.WorkRequest,
) func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
	t.Helper()

	return func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
		workRequestID := stringValue(req.WorkRequestId)
		workRequest, ok := workRequests[workRequestID]
		if !ok {
			t.Fatalf("unexpected workRequestId %q", workRequestID)
		}
		return devopssdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireBuildPipelineParameterItems(
	t *testing.T,
	name string,
	params *devopssdk.BuildPipelineParameterCollection,
) []devopssdk.BuildPipelineParameter {
	t.Helper()
	if params == nil || len(params.Items) != 1 {
		t.Fatalf("%s = %#v, want one item", name, params)
	}
	return params.Items
}

func assertBuildPipelineStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func lastBuildPipelineConditionType(resource *devopsv1beta1.BuildPipeline) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}
