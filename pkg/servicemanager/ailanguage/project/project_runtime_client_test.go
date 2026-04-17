/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	ailanguagesdk "github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/common"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeProjectOCIClient struct {
	createProjectFn  func(context.Context, ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error)
	getProjectFn     func(context.Context, ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error)
	listProjectsFn   func(context.Context, ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error)
	updateProjectFn  func(context.Context, ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error)
	deleteProjectFn  func(context.Context, ailanguagesdk.DeleteProjectRequest) (ailanguagesdk.DeleteProjectResponse, error)
	getWorkRequestFn func(context.Context, ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error)
}

func (f *fakeProjectOCIClient) CreateProject(ctx context.Context, req ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error) {
	if f.createProjectFn != nil {
		return f.createProjectFn(ctx, req)
	}
	return ailanguagesdk.CreateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetProject(ctx context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
	if f.getProjectFn != nil {
		return f.getProjectFn(ctx, req)
	}
	return ailanguagesdk.GetProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) ListProjects(ctx context.Context, req ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error) {
	if f.listProjectsFn != nil {
		return f.listProjectsFn(ctx, req)
	}
	return ailanguagesdk.ListProjectsResponse{}, nil
}

func (f *fakeProjectOCIClient) UpdateProject(ctx context.Context, req ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error) {
	if f.updateProjectFn != nil {
		return f.updateProjectFn(ctx, req)
	}
	return ailanguagesdk.UpdateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) DeleteProject(ctx context.Context, req ailanguagesdk.DeleteProjectRequest) (ailanguagesdk.DeleteProjectResponse, error) {
	if f.deleteProjectFn != nil {
		return f.deleteProjectFn(ctx, req)
	}
	return ailanguagesdk.DeleteProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetWorkRequest(ctx context.Context, req ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return ailanguagesdk.GetWorkRequestResponse{}, nil
}

func testProjectClient(fake *fakeProjectOCIClient) ProjectServiceClient {
	return newProjectServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeProjectResource() *ailanguagev1beta1.Project {
	return &ailanguagev1beta1.Project{
		Spec: ailanguagev1beta1.ProjectSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "project-alpha",
			Description:   "desired description",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKProject(
	id string,
	compartmentID string,
	displayName string,
	description string,
	state ailanguagesdk.ProjectLifecycleStateEnum,
) ailanguagesdk.Project {
	project := ailanguagesdk.Project{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
	if description != "" {
		project.Description = common.String(description)
	}
	return project
}

func makeSDKProjectSummary(
	id string,
	compartmentID string,
	displayName string,
	description string,
	state ailanguagesdk.ProjectLifecycleStateEnum,
) ailanguagesdk.ProjectSummary {
	summary := ailanguagesdk.ProjectSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
	if description != "" {
		summary.Description = common.String(description)
	}
	return summary
}

func makeProjectWorkRequest(
	id string,
	operationType ailanguagesdk.OperationTypeEnum,
	status ailanguagesdk.OperationStatusEnum,
	action ailanguagesdk.ActionTypeEnum,
	projectID string,
) ailanguagesdk.WorkRequest {
	percentComplete := float32(42)

	workRequest := ailanguagesdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if projectID != "" {
		workRequest.Resources = []ailanguagesdk.WorkRequestResource{
			{
				EntityType: common.String("project"),
				ActionType: action,
				Identifier: common.String(projectID),
				EntityUri:  common.String("/20220101/projects/" + projectID),
			},
		}
	}
	return workRequest
}

func requireAsyncCurrent(t *testing.T, resource *ailanguagev1beta1.Project, phase shared.OSOKAsyncPhase, workRequestID string) {
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
}

func TestProjectServiceClientCreateOrUpdateCreatesAndPollsWorkRequest(t *testing.T) {
	t.Parallel()

	var createRequest ailanguagesdk.CreateProjectRequest
	var getWorkRequestRequest ailanguagesdk.GetWorkRequestRequest

	client := testProjectClient(&fakeProjectOCIClient{
		createProjectFn: func(_ context.Context, req ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error) {
			createRequest = req
			return ailanguagesdk.CreateProjectResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				Project: makeSDKProject(
					"ocid1.project.oc1..created",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					ailanguagesdk.ProjectLifecycleStateCreating,
				),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error) {
			getWorkRequestRequest = req
			return ailanguagesdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest(
					"wr-create-1",
					ailanguagesdk.OperationTypeCreateProject,
					ailanguagesdk.OperationStatusInProgress,
					ailanguagesdk.ActionTypeInProgress,
					"ocid1.project.oc1..created",
				),
			}, nil
		},
	})

	resource := makeProjectResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create work request is in progress")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the create work request remains in progress")
	}
	if createRequest.CreateProjectDetails.CompartmentId == nil || *createRequest.CreateProjectDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateProjectDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if getWorkRequestRequest.WorkRequestId == nil || *getWorkRequestRequest.WorkRequestId != "wr-create-1" {
		t.Fatalf("getWorkRequest workRequestId = %v, want %q", getWorkRequestRequest.WorkRequestId, "wr-create-1")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.project.oc1..created" {
		t.Fatalf("status.ocid = %q, want created project ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
	if resource.Status.LifecycleState != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "CREATING")
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "IN_PROGRESS" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "IN_PROGRESS")
	}
}

func TestProjectServiceClientCreateOrUpdateResolvesExistingUsingListMatchFields(t *testing.T) {
	t.Parallel()

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		createProjectFn: func(_ context.Context, _ ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error) {
			createCalls++
			return ailanguagesdk.CreateProjectResponse{}, nil
		},
		listProjectsFn: func(_ context.Context, req ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error) {
			listCalls++
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..example" {
				t.Fatalf("list compartmentId = %v, want spec compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "project-alpha" {
				t.Fatalf("list displayName = %v, want spec displayName", req.DisplayName)
			}
			if req.ProjectId != nil {
				t.Fatalf("list projectId = %v, want nil when no tracked identity exists", req.ProjectId)
			}
			return ailanguagesdk.ListProjectsResponse{
				ProjectCollection: ailanguagesdk.ProjectCollection{
					Items: []ailanguagesdk.ProjectSummary{
						makeSDKProjectSummary(
							"ocid1.project.oc1..existing",
							"ocid1.compartment.oc1..example",
							"project-alpha",
							"desired description",
							ailanguagesdk.ProjectLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want resolved project ID", req.ProjectId)
			}
			return ailanguagesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					ailanguagesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makeProjectResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for steady ACTIVE lifecycle")
	}
	if createCalls != 0 {
		t.Fatalf("CreateProject() calls = %d, want 0", createCalls)
	}
	if listCalls != 1 {
		t.Fatalf("ListProjects() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetProject() calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != "ocid1.project.oc1..existing" {
		t.Fatalf("status.id = %q, want resolved project ID", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for steady ACTIVE lifecycle", resource.Status.OsokStatus.Async.Current)
	}
}

func TestProjectServiceClientCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					ailanguagesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		updateProjectFn: func(_ context.Context, req ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error) {
			t.Fatalf("UpdateProject() should not be called when mutable fields already match: %+v", req)
			return ailanguagesdk.UpdateProjectResponse{}, nil
		},
	})

	resource := makeProjectResource()
	resource.Status.Id = "ocid1.project.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if getCalls != 1 {
		t.Fatalf("GetProject() calls = %d, want 1", getCalls)
	}
	if resource.Status.DisplayName != "project-alpha" {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, "project-alpha")
	}
	if resource.Status.Description != "desired description" {
		t.Fatalf("status.description = %q, want %q", resource.Status.Description, "desired description")
	}
}

func TestProjectServiceClientCreateOrUpdateStartsUpdateWorkRequestWhenMutableDriftExists(t *testing.T) {
	t.Parallel()

	var updateRequest ailanguagesdk.UpdateProjectRequest
	var getWorkRequestRequest ailanguagesdk.GetWorkRequestRequest

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"stale description",
					ailanguagesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		updateProjectFn: func(_ context.Context, req ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error) {
			updateRequest = req
			return ailanguagesdk.UpdateProjectResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error) {
			getWorkRequestRequest = req
			return ailanguagesdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest(
					"wr-update-1",
					ailanguagesdk.OperationTypeUpdateProject,
					ailanguagesdk.OperationStatusInProgress,
					ailanguagesdk.ActionTypeInProgress,
					"ocid1.project.oc1..existing",
				),
			}, nil
		},
	})

	resource := makeProjectResource()
	resource.Status.Id = "ocid1.project.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the update work request remains in progress")
	}
	if updateRequest.ProjectId == nil || *updateRequest.ProjectId != "ocid1.project.oc1..existing" {
		t.Fatalf("update projectId = %v, want tracked project ID", updateRequest.ProjectId)
	}
	if updateRequest.UpdateProjectDetails.Description == nil || *updateRequest.UpdateProjectDetails.Description != "desired description" {
		t.Fatalf("update description = %v, want %q", updateRequest.UpdateProjectDetails.Description, "desired description")
	}
	if updateRequest.UpdateProjectDetails.DisplayName != nil {
		t.Fatalf("update displayName = %v, want nil when displayName did not drift", updateRequest.UpdateProjectDetails.DisplayName)
	}
	if getWorkRequestRequest.WorkRequestId == nil || *getWorkRequestRequest.WorkRequestId != "wr-update-1" {
		t.Fatalf("getWorkRequest workRequestId = %v, want %q", getWorkRequestRequest.WorkRequestId, "wr-update-1")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "IN_PROGRESS" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "IN_PROGRESS")
	}
}

func TestProjectServiceClientCreateOrUpdateResumesSucceededUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	getProjectCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		getWorkRequestFn: func(_ context.Context, req ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error) {
			if req.WorkRequestId == nil || *req.WorkRequestId != "wr-update-1" {
				t.Fatalf("getWorkRequest workRequestId = %v, want %q", req.WorkRequestId, "wr-update-1")
			}
			return ailanguagesdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest(
					"wr-update-1",
					ailanguagesdk.OperationTypeUpdateProject,
					ailanguagesdk.OperationStatusSucceeded,
					ailanguagesdk.ActionTypeUpdated,
					"ocid1.project.oc1..existing",
				),
			}, nil
		},
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			getProjectCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					ailanguagesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makeProjectResource()
	resource.Status.Id = "ocid1.project.oc1..existing"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   "wr-update-1",
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         "update in progress",
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after the update work request succeeds and Project returns ACTIVE")
	}
	if getProjectCalls != 1 {
		t.Fatalf("GetProject() calls = %d, want 1", getProjectCalls)
	}
	if resource.Status.Description != "desired description" {
		t.Fatalf("status.description = %q, want %q", resource.Status.Description, "desired description")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after ACTIVE reconciliation", resource.Status.OsokStatus.Async.Current)
	}
}

func TestProjectServiceClientCreateOrUpdateConflictCurrentlyBeingModifiedRequeuesInsteadOfFailing(t *testing.T) {
	t.Parallel()

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"stale description",
					ailanguagesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		updateProjectFn: func(_ context.Context, req ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error) {
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("update projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.UpdateProjectResponse{}, errortest.NewServiceError(409, "IncorrectState", "Project is currently being modified")
		},
	})

	resource := makeProjectResource()
	resource.Status.Id = "ocid1.project.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want nil for retryable conflict", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success for retryable update conflict")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue for retryable update conflict")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Updating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Updating)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want pending update tracker")
	}
	if current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseUpdate)
	}
	if current.WorkRequestID != "" {
		t.Fatalf("status.async.current.workRequestId = %q, want empty when OCI did not return a work request ID", current.WorkRequestID)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func TestProjectServiceClientDeleteFallsBackToListUsingProjectIDAndPollsDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	notFound := errortest.NewServiceError(404, "NotFound", "")
	getCalls := 0
	listCalls := 0
	deleteCalls := 0
	getWorkRequestCalls := 0

	resource := makeProjectResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"
	resource.Spec.DisplayName = "desired-name"
	resource.Status.Id = "ocid1.project.oc1..existing"
	resource.Status.CompartmentId = "ocid1.compartment.oc1..observed"
	resource.Status.DisplayName = "observed-name"

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.GetProjectResponse{}, notFound
		},
		listProjectsFn: func(_ context.Context, req ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error) {
			listCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("list projectId = %v, want tracked project ID", req.ProjectId)
			}
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..observed" {
				t.Fatalf("list compartmentId = %v, want observed status compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "observed-name" {
				t.Fatalf("list displayName = %v, want observed status displayName", req.DisplayName)
			}

			state := ailanguagesdk.ProjectLifecycleStateActive
			if listCalls > 1 {
				state = ailanguagesdk.ProjectLifecycleStateDeleting
			}
			return ailanguagesdk.ListProjectsResponse{
				ProjectCollection: ailanguagesdk.ProjectCollection{
					Items: []ailanguagesdk.ProjectSummary{
						makeSDKProjectSummary(
							"ocid1.project.oc1..existing",
							"ocid1.compartment.oc1..observed",
							"observed-name",
							"desired description",
							state,
						),
					},
				},
			}, nil
		},
		deleteProjectFn: func(_ context.Context, req ailanguagesdk.DeleteProjectRequest) (ailanguagesdk.DeleteProjectResponse, error) {
			deleteCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("delete projectId = %v, want tracked project ID", req.ProjectId)
			}
			return ailanguagesdk.DeleteProjectResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req ailanguagesdk.GetWorkRequestRequest) (ailanguagesdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			if req.WorkRequestId == nil || *req.WorkRequestId != "wr-delete-1" {
				t.Fatalf("getWorkRequest workRequestId = %v, want %q", req.WorkRequestId, "wr-delete-1")
			}
			return ailanguagesdk.GetWorkRequestResponse{
				WorkRequest: makeProjectWorkRequest(
					"wr-delete-1",
					ailanguagesdk.OperationTypeDeleteProject,
					ailanguagesdk.OperationStatusSucceeded,
					ailanguagesdk.ActionTypeDeleted,
					"ocid1.project.oc1..existing",
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should report in-progress delete while OCI still returns DELETING")
	}
	if getCalls != 2 {
		t.Fatalf("GetProject() calls = %d, want 2", getCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListProjects() calls = %d, want 2", listCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteProject() calls = %d, want 1", deleteCalls)
	}
	if getWorkRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", getWorkRequestCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "DELETING")
	}
}
