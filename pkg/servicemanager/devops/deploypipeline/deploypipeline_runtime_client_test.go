/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package deploypipeline

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
	testDeployPipelineID = "ocid1.devopsdeploypipeline.oc1..example"
	testProjectID        = "ocid1.devopsproject.oc1..example"
	testCompartmentID    = "ocid1.compartment.oc1..example"
)

type fakeDeployPipelineOCIClient struct {
	createFn         func(context.Context, devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error)
	getFn            func(context.Context, devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error)
	listFn           func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error)
	updateFn         func(context.Context, devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error)
	deleteFn         func(context.Context, devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error)
	getWorkRequestFn func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

func (f *fakeDeployPipelineOCIClient) CreateDeployPipeline(ctx context.Context, req devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return devopssdk.CreateDeployPipelineResponse{}, nil
}

func (f *fakeDeployPipelineOCIClient) GetDeployPipeline(ctx context.Context, req devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return devopssdk.GetDeployPipelineResponse{}, nil
}

func (f *fakeDeployPipelineOCIClient) ListDeployPipelines(ctx context.Context, req devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return devopssdk.ListDeployPipelinesResponse{}, nil
}

func (f *fakeDeployPipelineOCIClient) UpdateDeployPipeline(ctx context.Context, req devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return devopssdk.UpdateDeployPipelineResponse{}, nil
}

func (f *fakeDeployPipelineOCIClient) DeleteDeployPipeline(ctx context.Context, req devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return devopssdk.DeleteDeployPipelineResponse{}, nil
}

func (f *fakeDeployPipelineOCIClient) GetWorkRequest(ctx context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return devopssdk.GetWorkRequestResponse{}, nil
}

func testDeployPipelineClient(fake *fakeDeployPipelineOCIClient) DeployPipelineServiceClient {
	return newDeployPipelineServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeDeployPipelineResource() *devopsv1beta1.DeployPipeline {
	return &devopsv1beta1.DeployPipeline{
		Spec: devopsv1beta1.DeployPipelineSpec{
			ProjectId:   testProjectID,
			Description: "desired description",
			DisplayName: "pipeline-alpha",
			DeployPipelineParameters: devopsv1beta1.DeployPipelineParameters{
				Items: []devopsv1beta1.DeployPipelineParametersItem{
					{
						Name:         "release_version",
						DefaultValue: "1.0.0",
						Description:  "release version",
					},
				},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKDeployPipeline(
	id string,
	projectID string,
	displayName string,
	description string,
	state devopssdk.DeployPipelineLifecycleStateEnum,
) devopssdk.DeployPipeline {
	pipeline := devopssdk.DeployPipeline{
		Id:            common.String(id),
		ProjectId:     common.String(projectID),
		CompartmentId: common.String(testCompartmentID),
		DisplayName:   common.String(displayName),
		Description:   common.String(description),
		DeployPipelineParameters: &devopssdk.DeployPipelineParameterCollection{
			Items: []devopssdk.DeployPipelineParameter{
				{
					Name:         common.String("release_version"),
					DefaultValue: common.String("1.0.0"),
					Description:  common.String("release version"),
				},
			},
		},
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
	return pipeline
}

func makeSDKDeployPipelineSummary(
	id string,
	projectID string,
	displayName string,
	description string,
	state devopssdk.DeployPipelineLifecycleStateEnum,
) devopssdk.DeployPipelineSummary {
	summary := devopssdk.DeployPipelineSummary{
		Id:             common.String(id),
		ProjectId:      common.String(projectID),
		CompartmentId:  common.String(testCompartmentID),
		DisplayName:    common.String(displayName),
		Description:    common.String(description),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	return summary
}

func makeDeployPipelineWorkRequest(
	id string,
	operationType devopssdk.OperationTypeEnum,
	status devopssdk.OperationStatusEnum,
	action devopssdk.ActionTypeEnum,
	pipelineID string,
) devopssdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := devopssdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if pipelineID != "" {
		workRequest.Resources = []devopssdk.WorkRequestResource{
			{
				EntityType: common.String("deployPipeline"),
				ActionType: action,
				Identifier: common.String(pipelineID),
				EntityUri:  common.String("/20210630/deployPipelines/" + pipelineID),
			},
		}
	}
	return workRequest
}

func TestDeployPipelineRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedDeployPipelineRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedDeployPipelineRuntimeSemantics() = nil")
	}
	if got.FormalService != "devops" || got.FormalSlug != "deploypipeline" {
		t.Fatalf("formal binding = %s/%s, want devops/deploypipeline", got.FormalService, got.FormalSlug)
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
	assertDeployPipelineStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"projectId", "displayName", "id"})
	assertDeployPipelineStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"description", "displayName", "deployPipelineParameters", "freeformTags", "definedTags"})
	assertDeployPipelineStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"projectId"})
}

func TestDeployPipelineServiceClientCreatesThroughWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	var createRequest devopssdk.CreateDeployPipelineRequest
	listCalls := 0
	createCalls := 0
	getWorkRequestCalls := 0
	getCalls := 0

	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		listFn: func(_ context.Context, req devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
			listCalls++
			requireStringPtr(t, "list projectId", req.ProjectId, testProjectID)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return devopssdk.ListDeployPipelinesResponse{}, nil
		},
		createFn: func(_ context.Context, req devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error) {
			createCalls++
			createRequest = req
			return devopssdk.CreateDeployPipelineResponse{
				DeployPipeline:   makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeDeployPipelineWorkRequest("wr-create-1", devopssdk.OperationTypeCreateDeployPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, testDeployPipelineID),
			}, nil
		},
		getFn: func(_ context.Context, req devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get deployPipelineId", req.DeployPipelineId, testDeployPipelineID)
			return devopssdk.GetDeployPipelineResponse{
				DeployPipeline: makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive),
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
	requireCreateDeployPipelineRequest(t, createRequest, resource)
	requireCompletedCreateDeployPipelineStatus(t, resource)
}

func TestDeployPipelineServiceClientTracksPendingCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		listFn: func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
			return devopssdk.ListDeployPipelinesResponse{}, nil
		},
		createFn: func(context.Context, devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error) {
			return devopssdk.CreateDeployPipelineResponse{
				DeployPipeline:   makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-pending"),
				OpcRequestId:     common.String("opc-create-pending"),
			}, nil
		},
		getWorkRequestFn: func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeDeployPipelineWorkRequest("wr-create-pending", devopssdk.OperationTypeCreateDeployPipeline, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, testDeployPipelineID),
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
	if current.RawOperationType != string(devopssdk.OperationTypeCreateDeployPipeline) {
		t.Fatalf("status.async.current.rawOperationType = %q, want CREATE_DEPLOY_PIPELINE", current.RawOperationType)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-pending" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-pending", got)
	}
}

func TestDeployPipelineServiceClientBindsExistingFromPagedList(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	pages := []string{}
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		listFn: func(_ context.Context, req devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
			pages = append(pages, stringValue(req.Page))
			requireStringPtr(t, "list projectId", req.ProjectId, testProjectID)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.Page == nil {
				return devopssdk.ListDeployPipelinesResponse{
					DeployPipelineCollection: devopssdk.DeployPipelineCollection{Items: []devopssdk.DeployPipelineSummary{
						makeSDKDeployPipelineSummary("ocid1.devopsdeploypipeline.oc1..other", testProjectID, "other-pipeline", "other", devopssdk.DeployPipelineLifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return devopssdk.ListDeployPipelinesResponse{
				DeployPipelineCollection: devopssdk.DeployPipelineCollection{Items: []devopssdk.DeployPipelineSummary{
					makeSDKDeployPipelineSummary(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive),
				}},
			}, nil
		},
		getFn: func(_ context.Context, req devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get deployPipelineId", req.DeployPipelineId, testDeployPipelineID)
			return devopssdk.GetDeployPipelineResponse{
				DeployPipeline: makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error) {
			createCalled = true
			return devopssdk.CreateDeployPipelineResponse{}, nil
		},
		updateFn: func(context.Context, devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error) {
			updateCalled = true
			return devopssdk.UpdateDeployPipelineResponse{}, nil
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
		t.Fatalf("GetDeployPipeline() calls = %d, want 1 live mutation assessment read", getCalls)
	}
	if got := resource.Status.Id; got != testDeployPipelineID {
		t.Fatalf("status.id = %q, want %q", got, testDeployPipelineID)
	}
}

func TestDeployPipelineServiceClientNoopReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	resource.Status.Id = testDeployPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDeployPipelineID)
	updateCalled := false

	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		getFn: func(context.Context, devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			return devopssdk.GetDeployPipelineResponse{
				DeployPipeline: makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error) {
			updateCalled = true
			return devopssdk.UpdateDeployPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if updateCalled {
		t.Fatal("UpdateDeployPipeline() was called for matching observed state")
	}
	if got := lastDeployPipelineConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestDeployPipelineServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	resource.Status.Id = testDeployPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDeployPipelineID)
	resource.Spec.Description = "new description"
	resource.Spec.DisplayName = "pipeline-renamed"
	resource.Spec.DeployPipelineParameters.Items[0].DefaultValue = "2.0.0"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	var updateRequest devopssdk.UpdateDeployPipelineRequest
	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		getFn: func(_ context.Context, req devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get deployPipelineId", req.DeployPipelineId, testDeployPipelineID)
			if getCalls == 1 {
				current := makeSDKDeployPipeline(testDeployPipelineID, testProjectID, "pipeline-alpha", "old description", devopssdk.DeployPipelineLifecycleStateActive)
				current.FreeformTags = map[string]string{"env": "dev"}
				current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
				return devopssdk.GetDeployPipelineResponse{DeployPipeline: current}, nil
			}
			current := makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive)
			current.DeployPipelineParameters.Items[0].DefaultValue = common.String("2.0.0")
			current.FreeformTags = map[string]string{"env": "prod"}
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			return devopssdk.GetDeployPipelineResponse{DeployPipeline: current}, nil
		},
		updateFn: func(_ context.Context, req devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error) {
			updateRequest = req
			return devopssdk.UpdateDeployPipelineResponse{
				DeployPipeline:   makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateUpdating),
				OpcWorkRequestId: common.String("wr-update-1"),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-update-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeDeployPipelineWorkRequest("wr-update-1", devopssdk.OperationTypeUpdateDeployPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeUpdated, testDeployPipelineID),
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
		t.Fatalf("GetDeployPipeline() calls = %d, want pre-update read and post-work-request read", getCalls)
	}
	requireStringPtr(t, "update deployPipelineId", updateRequest.DeployPipelineId, testDeployPipelineID)
	requireMutableDeployPipelineUpdateRequest(t, updateRequest, resource)
	requireUpdatedDeployPipelineStatus(t, resource)
}

func TestDeployPipelineServiceClientRejectsProjectIDDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	resource.Status.Id = testDeployPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDeployPipelineID)
	resource.Spec.ProjectId = "ocid1.devopsproject.oc1..renamed"
	updateCalled := false

	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		getFn: func(context.Context, devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			return devopssdk.GetDeployPipelineResponse{
				DeployPipeline: makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error) {
			updateCalled = true
			return devopssdk.UpdateDeployPipelineResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want projectId drift error")
	}
	if updateCalled {
		t.Fatal("UpdateDeployPipeline() was called for projectId drift")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "projectId") {
		t.Fatalf("CreateOrUpdate() error = %v, want projectId drift detail", err)
	}
	if got := lastDeployPipelineConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestDeployPipelineServiceClientDeleteRetainsFinalizerUntilReadbackConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	resource.Status.Id = testDeployPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDeployPipelineID)
	getCalls := 0
	deleteCalls := 0

	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		getFn: func(_ context.Context, req devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			getCalls++
			requireStringPtr(t, "get deployPipelineId", req.DeployPipelineId, testDeployPipelineID)
			switch getCalls {
			case 1:
				return devopssdk.GetDeployPipelineResponse{
					DeployPipeline: makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateActive),
				}, nil
			case 2:
				return devopssdk.GetDeployPipelineResponse{
					DeployPipeline: makeSDKDeployPipeline(testDeployPipelineID, testProjectID, resource.Spec.DisplayName, resource.Spec.Description, devopssdk.DeployPipelineLifecycleStateDeleting),
				}, nil
			default:
				return devopssdk.GetDeployPipelineResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "DeployPipeline deleted")
			}
		},
		deleteFn: func(_ context.Context, req devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete deployPipelineId", req.DeployPipelineId, testDeployPipelineID)
			return devopssdk.DeleteDeployPipelineResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeDeployPipelineWorkRequest("wr-delete-1", devopssdk.OperationTypeDeleteDeployPipeline, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeDeleted, testDeployPipelineID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	requireDeployPipelineDeletePending(t, deleted, err, resource)

	deleted, err = client.Delete(context.Background(), resource)
	requireDeployPipelineDeleteConfirmed(t, deleted, err, resource)
	if deleteCalls != 1 {
		t.Fatalf("DeleteDeployPipeline() calls = %d, want 1", deleteCalls)
	}
}

func TestDeployPipelineDeleteTreatsPreDeleteAuthShapedNotFoundAsError(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	resource.Status.Id = testDeployPipelineID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDeployPipelineID)
	deleteCalled := false

	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		getFn: func(context.Context, devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
			return devopssdk.GetDeployPipelineResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error) {
			deleteCalled = true
			return devopssdk.DeleteDeployPipelineResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true for auth-shaped 404")
	}
	if deleteCalled {
		t.Fatal("DeleteDeployPipeline() was called after ambiguous pre-delete read")
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

func TestDeployPipelineServiceClientRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := makeDeployPipelineResource()
	client := testDeployPipelineClient(&fakeDeployPipelineOCIClient{
		listFn: func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
			return devopssdk.ListDeployPipelinesResponse{}, nil
		},
		createFn: func(context.Context, devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error) {
			return devopssdk.CreateDeployPipelineResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
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
	if got := lastDeployPipelineConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func requireCreateDeployPipelineRequest(
	t *testing.T,
	createRequest devopssdk.CreateDeployPipelineRequest,
	resource *devopsv1beta1.DeployPipeline,
) {
	t.Helper()

	requireStringPtr(t, "create projectId", createRequest.ProjectId, testProjectID)
	requireStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireDeployPipelineParameter(t, "create parameter", createRequest.DeployPipelineParameters, "release_version", "1.0.0")
}

func requireCompletedCreateDeployPipelineStatus(t *testing.T, resource *devopsv1beta1.DeployPipeline) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != testDeployPipelineID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDeployPipelineID)
	}
	if got := resource.Status.Id; got != testDeployPipelineID {
		t.Fatalf("status.id = %q, want %q", got, testDeployPipelineID)
	}
	if got := resource.Status.DeployPipelineParameters.Items[0].Name; got != "release_version" {
		t.Fatalf("status.deployPipelineParameters.items[0].name = %q, want release_version", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed create", resource.Status.OsokStatus.Async.Current)
	}
	if got := lastDeployPipelineConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func requireMutableDeployPipelineUpdateRequest(
	t *testing.T,
	updateRequest devopssdk.UpdateDeployPipelineRequest,
	resource *devopsv1beta1.DeployPipeline,
) {
	t.Helper()

	requireStringPtr(t, "update description", updateRequest.Description, resource.Spec.Description)
	requireStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireDeployPipelineParameter(t, "update parameter", updateRequest.DeployPipelineParameters, "release_version", "2.0.0")
	if !reflect.DeepEqual(updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
}

func requireUpdatedDeployPipelineStatus(t *testing.T, resource *devopsv1beta1.DeployPipeline) {
	t.Helper()

	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.DeployPipelineParameters.Items[0].DefaultValue; got != "2.0.0" {
		t.Fatalf("status.deployPipelineParameters.items[0].defaultValue = %q, want 2.0.0", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func requireDeployPipelineDeletePending(
	t *testing.T,
	deleted bool,
	err error,
	resource *devopsv1beta1.DeployPipeline,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback still reports DELETING")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil before confirmed deletion", resource.Status.OsokStatus.DeletedAt)
	}
	if got := lastDeployPipelineConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
}

func requireDeployPipelineDeleteConfirmed(
	t *testing.T,
	deleted bool,
	err error,
	resource *devopsv1beta1.DeployPipeline,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want true after readback NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func requireAsyncCurrent(t *testing.T, resource *devopsv1beta1.DeployPipeline, phase shared.OSOKAsyncPhase, workRequestID string) *shared.OSOKAsyncOperation {
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

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireDeployPipelineParameter(
	t *testing.T,
	name string,
	got *devopssdk.DeployPipelineParameterCollection,
	wantName string,
	wantDefault string,
) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want one parameter", name)
	}
	if len(got.Items) != 1 {
		t.Fatalf("%s items len = %d, want 1", name, len(got.Items))
	}
	requireStringPtr(t, name+" name", got.Items[0].Name, wantName)
	requireStringPtr(t, name+" defaultValue", got.Items[0].DefaultValue, wantDefault)
}

func assertDeployPipelineStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func lastDeployPipelineConditionType(resource *devopsv1beta1.DeployPipeline) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
