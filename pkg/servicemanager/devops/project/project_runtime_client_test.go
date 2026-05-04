/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

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
	testProjectID     = "ocid1.devopsproject.oc1..example"
	testCompartmentID = "ocid1.compartment.oc1..example"
	testTopicID       = "ocid1.onstopic.oc1..example"
)

type fakeProjectOCIClient struct {
	createProjectFn  func(context.Context, devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error)
	getProjectFn     func(context.Context, devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error)
	listProjectsFn   func(context.Context, devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error)
	updateProjectFn  func(context.Context, devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error)
	deleteProjectFn  func(context.Context, devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error)
	getWorkRequestFn func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

func (f *fakeProjectOCIClient) CreateProject(ctx context.Context, req devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error) {
	if f.createProjectFn != nil {
		return f.createProjectFn(ctx, req)
	}
	return devopssdk.CreateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetProject(ctx context.Context, req devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
	if f.getProjectFn != nil {
		return f.getProjectFn(ctx, req)
	}
	return devopssdk.GetProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) ListProjects(ctx context.Context, req devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
	if f.listProjectsFn != nil {
		return f.listProjectsFn(ctx, req)
	}
	return devopssdk.ListProjectsResponse{}, nil
}

func (f *fakeProjectOCIClient) UpdateProject(ctx context.Context, req devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error) {
	if f.updateProjectFn != nil {
		return f.updateProjectFn(ctx, req)
	}
	return devopssdk.UpdateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) DeleteProject(ctx context.Context, req devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error) {
	if f.deleteProjectFn != nil {
		return f.deleteProjectFn(ctx, req)
	}
	return devopssdk.DeleteProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetWorkRequest(ctx context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return devopssdk.GetWorkRequestResponse{}, nil
}

func testProjectClient(fake *fakeProjectOCIClient) ProjectServiceClient {
	return newProjectServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeProjectResource() *devopsv1beta1.Project {
	return &devopsv1beta1.Project{
		Spec: devopsv1beta1.ProjectSpec{
			Name:          "project-alpha",
			CompartmentId: testCompartmentID,
			NotificationConfig: devopsv1beta1.ProjectNotificationConfig{
				TopicId: testTopicID,
			},
			Description:  "desired description",
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKProject(
	id string,
	compartmentID string,
	name string,
	topicID string,
	description string,
	state devopssdk.ProjectLifecycleStateEnum,
) devopssdk.Project {
	project := devopssdk.Project{
		Id:                 common.String(id),
		Name:               common.String(name),
		CompartmentId:      common.String(compartmentID),
		NotificationConfig: &devopssdk.NotificationConfig{TopicId: common.String(topicID)},
		LifecycleState:     state,
		FreeformTags:       map[string]string{"env": "dev"},
		DefinedTags:        map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:         map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
	if description != "" {
		project.Description = common.String(description)
	}
	return project
}

func makeSDKProjectSummary(
	id string,
	compartmentID string,
	name string,
	topicID string,
	description string,
	state devopssdk.ProjectLifecycleStateEnum,
) devopssdk.ProjectSummary {
	summary := devopssdk.ProjectSummary{
		Id:                 common.String(id),
		Name:               common.String(name),
		CompartmentId:      common.String(compartmentID),
		NotificationConfig: &devopssdk.NotificationConfig{TopicId: common.String(topicID)},
		LifecycleState:     state,
		FreeformTags:       map[string]string{"env": "dev"},
		DefinedTags:        map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	if description != "" {
		summary.Description = common.String(description)
	}
	return summary
}

func makeProjectWorkRequest(
	id string,
	operationType devopssdk.OperationTypeEnum,
	status devopssdk.OperationStatusEnum,
	action devopssdk.ActionTypeEnum,
	projectID string,
) devopssdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := devopssdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if projectID != "" {
		workRequest.Resources = []devopssdk.WorkRequestResource{
			{
				EntityType: common.String("project"),
				ActionType: action,
				Identifier: common.String(projectID),
				EntityUri:  common.String("/20210630/projects/" + projectID),
			},
		}
	}
	return workRequest
}

func TestProjectRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedProjectRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedProjectRuntimeSemantics() = nil")
	}
	if got.FormalService != "devops" || got.FormalSlug != "project" {
		t.Fatalf("formal binding = %s/%s, want devops/project", got.FormalService, got.FormalSlug)
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
	assertProjectStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "id"})
	assertProjectStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"description", "notificationConfig", "freeformTags", "definedTags"})
	assertProjectStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "name"})
}

func TestProjectServiceClientCreatesThroughWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	var createRequest devopssdk.CreateProjectRequest
	listCalls := 0
	createCalls := 0
	getWorkRequestCalls := 0
	getCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		listProjectsFn: func(_ context.Context, req devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
			listCalls++
			requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list name", req.Name, resource.Spec.Name)
			return devopssdk.ListProjectsResponse{}, nil
		},
		createProjectFn: func(_ context.Context, req devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error) {
			createCalls++
			createRequest = req
			return devopssdk.CreateProjectResponse{
				Project:          makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest("wr-create-1", devopssdk.OperationTypeCreateProject, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, testProjectID),
			}, nil
		},
		getProjectFn: func(_ context.Context, req devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
			getCalls++
			requireStringPtr(t, "get projectId", req.ProjectId, testProjectID)
			return devopssdk.GetProjectResponse{
				Project: makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive),
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
	requireStringPtr(t, "create name", createRequest.CreateProjectDetails.Name, resource.Spec.Name)
	requireStringPtr(t, "create compartmentId", createRequest.CreateProjectDetails.CompartmentId, testCompartmentID)
	requireStringPtr(t, "create topicId", createRequest.CreateProjectDetails.NotificationConfig.TopicId, testTopicID)
	if got := string(resource.Status.OsokStatus.Ocid); got != testProjectID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProjectID)
	}
	if got := resource.Status.Id; got != testProjectID {
		t.Fatalf("status.id = %q, want %q", got, testProjectID)
	}
	if got := resource.Status.NotificationConfig.TopicId; got != testTopicID {
		t.Fatalf("status.notificationConfig.topicId = %q, want %q", got, testTopicID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed create", resource.Status.OsokStatus.Async.Current)
	}
	if got := lastProjectConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestProjectServiceClientTracksPendingCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	client := testProjectClient(&fakeProjectOCIClient{
		listProjectsFn: func(context.Context, devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
			return devopssdk.ListProjectsResponse{}, nil
		},
		createProjectFn: func(context.Context, devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error) {
			return devopssdk.CreateProjectResponse{
				Project:          makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-pending"),
				OpcRequestId:     common.String("opc-create-pending"),
			}, nil
		},
		getWorkRequestFn: func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest("wr-create-pending", devopssdk.OperationTypeCreateProject, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, testProjectID),
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
	if current.RawOperationType != string(devopssdk.OperationTypeCreateProject) {
		t.Fatalf("status.async.current.rawOperationType = %q, want CREATE_PROJECT", current.RawOperationType)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-pending" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-pending", got)
	}
}

func TestProjectServiceClientBindsExistingFromPagedList(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	pages := []string{}
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		listProjectsFn: func(_ context.Context, req devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
			pages = append(pages, stringValue(req.Page))
			requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list name", req.Name, resource.Spec.Name)
			if req.Page == nil {
				return devopssdk.ListProjectsResponse{
					ProjectCollection: devopssdk.ProjectCollection{Items: []devopssdk.ProjectSummary{
						makeSDKProjectSummary("ocid1.devopsproject.oc1..other", testCompartmentID, "other-project", testTopicID, "other", devopssdk.ProjectLifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return devopssdk.ListProjectsResponse{
				ProjectCollection: devopssdk.ProjectCollection{Items: []devopssdk.ProjectSummary{
					makeSDKProjectSummary(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive),
				}},
			}, nil
		},
		getProjectFn: func(_ context.Context, req devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
			getCalls++
			requireStringPtr(t, "get projectId", req.ProjectId, testProjectID)
			return devopssdk.GetProjectResponse{
				Project: makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive),
			}, nil
		},
		createProjectFn: func(context.Context, devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error) {
			createCalled = true
			return devopssdk.CreateProjectResponse{}, nil
		},
		updateProjectFn: func(context.Context, devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error) {
			updateCalled = true
			return devopssdk.UpdateProjectResponse{}, nil
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
		t.Fatalf("GetProject() calls = %d, want 1 live mutation assessment read", getCalls)
	}
	if got := resource.Status.Id; got != testProjectID {
		t.Fatalf("status.id = %q, want %q", got, testProjectID)
	}
}

func TestProjectServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	resource.Status.Id = testProjectID
	resource.Status.OsokStatus.Ocid = shared.OCID(testProjectID)
	resource.Spec.Description = "new description"
	resource.Spec.NotificationConfig.TopicId = "ocid1.onstopic.oc1..updated"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	var updateRequest devopssdk.UpdateProjectRequest
	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
			getCalls++
			requireStringPtr(t, "get projectId", req.ProjectId, testProjectID)
			if getCalls == 1 {
				current := makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, "old description", devopssdk.ProjectLifecycleStateActive)
				current.FreeformTags = map[string]string{"env": "dev"}
				current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
				return devopssdk.GetProjectResponse{Project: current}, nil
			}
			current := makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, resource.Spec.NotificationConfig.TopicId, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive)
			current.FreeformTags = map[string]string{"env": "prod"}
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			return devopssdk.GetProjectResponse{Project: current}, nil
		},
		updateProjectFn: func(_ context.Context, req devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error) {
			updateRequest = req
			return devopssdk.UpdateProjectResponse{
				Project:          makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, resource.Spec.NotificationConfig.TopicId, resource.Spec.Description, devopssdk.ProjectLifecycleStateUpdating),
				OpcWorkRequestId: common.String("wr-update-1"),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-update-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest("wr-update-1", devopssdk.OperationTypeUpdateProject, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeUpdated, testProjectID),
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
		t.Fatalf("GetProject() calls = %d, want pre-update read and post-work-request read", getCalls)
	}
	requireStringPtr(t, "update projectId", updateRequest.ProjectId, testProjectID)
	requireStringPtr(t, "update description", updateRequest.UpdateProjectDetails.Description, resource.Spec.Description)
	requireStringPtr(t, "update topicId", updateRequest.UpdateProjectDetails.NotificationConfig.TopicId, resource.Spec.NotificationConfig.TopicId)
	if !reflect.DeepEqual(updateRequest.UpdateProjectDetails.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.UpdateProjectDetails.FreeformTags)
	}
	if got := updateRequest.UpdateProjectDetails.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
	if got := resource.Status.NotificationConfig.TopicId; got != resource.Spec.NotificationConfig.TopicId {
		t.Fatalf("status.notificationConfig.topicId = %q, want %q", got, resource.Spec.NotificationConfig.TopicId)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestProjectServiceClientRejectsForceNewDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	resource.Status.Id = testProjectID
	resource.Status.OsokStatus.Ocid = shared.OCID(testProjectID)
	resource.Spec.Name = "project-renamed"
	updateCalled := false

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(context.Context, devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
			return devopssdk.GetProjectResponse{
				Project: makeSDKProject(testProjectID, testCompartmentID, "project-alpha", testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive),
			}, nil
		},
		updateProjectFn: func(context.Context, devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error) {
			updateCalled = true
			return devopssdk.UpdateProjectResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want force-new drift error")
	}
	if updateCalled {
		t.Fatal("UpdateProject() was called for force-new name drift")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("CreateOrUpdate() error = %v, want name drift detail", err)
	}
	if got := lastProjectConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestProjectServiceClientDeleteRetainsFinalizerUntilReadbackConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	resource.Status.Id = testProjectID
	resource.Status.OsokStatus.Ocid = shared.OCID(testProjectID)
	getCalls := 0
	deleteCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
			getCalls++
			requireStringPtr(t, "get projectId", req.ProjectId, testProjectID)
			switch getCalls {
			case 1:
				return devopssdk.GetProjectResponse{
					Project: makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive),
				}, nil
			case 2:
				return devopssdk.GetProjectResponse{
					Project: makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateDeleting),
				}, nil
			default:
				return devopssdk.GetProjectResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "Project deleted")
			}
		},
		deleteProjectFn: func(_ context.Context, req devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete projectId", req.ProjectId, testProjectID)
			return devopssdk.DeleteProjectResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest("wr-delete-1", devopssdk.OperationTypeDeleteProject, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeDeleted, testProjectID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback still reports DELETING")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil before confirmed deletion", resource.Status.OsokStatus.DeletedAt)
	}
	if got := lastProjectConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want true after readback NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteProject() calls = %d, want 1", deleteCalls)
	}
}

func TestProjectServiceClientDeleteTreatsAuthShapedNotFoundAsError(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	resource.Status.Id = testProjectID
	resource.Status.OsokStatus.Ocid = shared.OCID(testProjectID)

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(context.Context, devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
			return devopssdk.GetProjectResponse{
				Project: makeSDKProject(testProjectID, testCompartmentID, resource.Spec.Name, testTopicID, resource.Spec.Description, devopssdk.ProjectLifecycleStateActive),
			}, nil
		},
		deleteProjectFn: func(context.Context, devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error) {
			return devopssdk.DeleteProjectResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
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

func TestProjectServiceClientRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := makeProjectResource()
	client := testProjectClient(&fakeProjectOCIClient{
		listProjectsFn: func(context.Context, devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
			return devopssdk.ListProjectsResponse{}, nil
		},
		createProjectFn: func(context.Context, devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error) {
			return devopssdk.CreateProjectResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
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
	if got := lastProjectConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func requireAsyncCurrent(t *testing.T, resource *devopsv1beta1.Project, phase shared.OSOKAsyncPhase, workRequestID string) *shared.OSOKAsyncOperation {
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

func assertProjectStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func lastProjectConditionType(resource *devopsv1beta1.Project) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}
