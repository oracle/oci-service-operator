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
	createProjectFn func(context.Context, ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error)
	getProjectFn    func(context.Context, ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error)
	listProjectsFn  func(context.Context, ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error)
	updateProjectFn func(context.Context, ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error)
	deleteProjectFn func(context.Context, ailanguagesdk.DeleteProjectRequest) (ailanguagesdk.DeleteProjectResponse, error)
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
		},
	}
}

func makeSDKProject(id string, compartmentID string, displayName string, description string, state ailanguagesdk.ProjectLifecycleStateEnum) ailanguagesdk.Project {
	project := ailanguagesdk.Project{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
	}
	if description != "" {
		project.Description = common.String(description)
	}
	return project
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

func TestProjectServiceClientCreateOrUpdateCreatesAndRequeuesWhileCreating(t *testing.T) {
	t.Parallel()

	var createRequest ailanguagesdk.CreateProjectRequest
	var getRequest ailanguagesdk.GetProjectRequest

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
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			getRequest = req
			return ailanguagesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..created",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					ailanguagesdk.ProjectLifecycleStateCreating,
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
		t.Fatal("CreateOrUpdate() should report success while create is in progress")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the project remains CREATING")
	}
	if createRequest.CreateProjectDetails.CompartmentId == nil || *createRequest.CreateProjectDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateProjectDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if getRequest.ProjectId == nil || *getRequest.ProjectId != "ocid1.project.oc1..created" {
		t.Fatalf("get projectId = %v, want created project ID", getRequest.ProjectId)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.project.oc1..created" {
		t.Fatalf("status.ocid = %q, want created project ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "CREATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "CREATING")
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

func TestProjectServiceClientCreateOrUpdateUpdatesMutableDrift(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest ailanguagesdk.UpdateProjectRequest

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			switch getCalls {
			case 1:
				return ailanguagesdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						"stale description",
						ailanguagesdk.ProjectLifecycleStateActive,
					),
				}, nil
			case 2:
				return ailanguagesdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						"desired description",
						ailanguagesdk.ProjectLifecycleStateActive,
					),
				}, nil
			default:
				t.Fatalf("unexpected GetProject() call %d", getCalls)
				return ailanguagesdk.GetProjectResponse{}, nil
			}
		},
		updateProjectFn: func(_ context.Context, req ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error) {
			updateRequest = req
			return ailanguagesdk.UpdateProjectResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
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
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once the follow-up read sees ACTIVE state")
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
	if resource.Status.Description != "desired description" {
		t.Fatalf("status.description = %q, want %q", resource.Status.Description, "desired description")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
}

func TestProjectServiceClientDeleteFallsBackToListUsingProjectID(t *testing.T) {
	t.Parallel()

	notFound := errortest.NewServiceError(404, "NotFound", "")
	getCalls := 0
	listCalls := 0
	deleteCalls := 0

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
						{
							Id:             common.String("ocid1.project.oc1..existing"),
							CompartmentId:  common.String("ocid1.compartment.oc1..observed"),
							DisplayName:    common.String("observed-name"),
							LifecycleState: state,
						},
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
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "DELETING")
	}
}
