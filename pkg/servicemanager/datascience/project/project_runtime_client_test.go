/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	datasciencesdk "github.com/oracle/oci-go-sdk/v65/datascience"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeProjectOCIClient struct {
	createProjectFn func(context.Context, datasciencesdk.CreateProjectRequest) (datasciencesdk.CreateProjectResponse, error)
	getProjectFn    func(context.Context, datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error)
	listProjectsFn  func(context.Context, datasciencesdk.ListProjectsRequest) (datasciencesdk.ListProjectsResponse, error)
	updateProjectFn func(context.Context, datasciencesdk.UpdateProjectRequest) (datasciencesdk.UpdateProjectResponse, error)
	deleteProjectFn func(context.Context, datasciencesdk.DeleteProjectRequest) (datasciencesdk.DeleteProjectResponse, error)
}

func (f *fakeProjectOCIClient) CreateProject(ctx context.Context, req datasciencesdk.CreateProjectRequest) (datasciencesdk.CreateProjectResponse, error) {
	if f.createProjectFn != nil {
		return f.createProjectFn(ctx, req)
	}
	return datasciencesdk.CreateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetProject(ctx context.Context, req datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
	if f.getProjectFn != nil {
		return f.getProjectFn(ctx, req)
	}
	return datasciencesdk.GetProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) ListProjects(ctx context.Context, req datasciencesdk.ListProjectsRequest) (datasciencesdk.ListProjectsResponse, error) {
	if f.listProjectsFn != nil {
		return f.listProjectsFn(ctx, req)
	}
	return datasciencesdk.ListProjectsResponse{}, nil
}

func (f *fakeProjectOCIClient) UpdateProject(ctx context.Context, req datasciencesdk.UpdateProjectRequest) (datasciencesdk.UpdateProjectResponse, error) {
	if f.updateProjectFn != nil {
		return f.updateProjectFn(ctx, req)
	}
	return datasciencesdk.UpdateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) DeleteProject(ctx context.Context, req datasciencesdk.DeleteProjectRequest) (datasciencesdk.DeleteProjectResponse, error) {
	if f.deleteProjectFn != nil {
		return f.deleteProjectFn(ctx, req)
	}
	return datasciencesdk.DeleteProjectResponse{}, nil
}

func testProjectClient(fake *fakeProjectOCIClient) ProjectServiceClient {
	return newProjectServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeProjectResource() *datasciencev1beta1.Project {
	return &datasciencev1beta1.Project{
		Spec: datasciencev1beta1.ProjectSpec{
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
	state datasciencesdk.ProjectLifecycleStateEnum,
) datasciencesdk.Project {
	project := datasciencesdk.Project{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
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
	state datasciencesdk.ProjectLifecycleStateEnum,
) datasciencesdk.ProjectSummary {
	project := datasciencesdk.ProjectSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	return project
}

func TestProjectServiceClientCreateOrUpdateCreatesProjectAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	var createRequest datasciencesdk.CreateProjectRequest
	var getRequest datasciencesdk.GetProjectRequest

	client := testProjectClient(&fakeProjectOCIClient{
		createProjectFn: func(_ context.Context, req datasciencesdk.CreateProjectRequest) (datasciencesdk.CreateProjectResponse, error) {
			createRequest = req
			return datasciencesdk.CreateProjectResponse{
				OpcRequestId: common.String("opc-create-1"),
				Project: makeSDKProject(
					"ocid1.project.oc1..created",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					datasciencesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		getProjectFn: func(_ context.Context, req datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
			getRequest = req
			return datasciencesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..created",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					datasciencesdk.ProjectLifecycleStateActive,
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
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if createRequest.CreateProjectDetails.CompartmentId == nil || *createRequest.CreateProjectDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateProjectDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateProjectDetails.DisplayName == nil || *createRequest.CreateProjectDetails.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %v, want %q", createRequest.CreateProjectDetails.DisplayName, resource.Spec.DisplayName)
	}
	if getRequest.ProjectId == nil || *getRequest.ProjectId != "ocid1.project.oc1..created" {
		t.Fatalf("get projectId = %v, want created project ID", getRequest.ProjectId)
	}
	if resource.Status.Id != "ocid1.project.oc1..created" {
		t.Fatalf("status.id = %q, want created project ID", resource.Status.Id)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.project.oc1..created" {
		t.Fatalf("status.ocid = %q, want created project ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
	if resource.Status.DisplayName != "project-alpha" {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, "project-alpha")
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for synchronous create", resource.Status.OsokStatus.Async.Current)
	}
}

func TestProjectServiceClientCreateOrUpdateResolvesExistingProjectBeforeCreate(t *testing.T) {
	t.Parallel()

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		createProjectFn: func(_ context.Context, _ datasciencesdk.CreateProjectRequest) (datasciencesdk.CreateProjectResponse, error) {
			createCalls++
			return datasciencesdk.CreateProjectResponse{}, nil
		},
		listProjectsFn: func(_ context.Context, req datasciencesdk.ListProjectsRequest) (datasciencesdk.ListProjectsResponse, error) {
			listCalls++
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil before the project is tracked", req.Id)
			}
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..example" {
				t.Fatalf("list compartmentId = %v, want spec compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "project-alpha" {
				t.Fatalf("list displayName = %v, want spec displayName", req.DisplayName)
			}
			if req.LifecycleState != "" {
				t.Fatalf("list lifecycleState = %q, want empty because runtime matches reusability from the response", req.LifecycleState)
			}
			if req.CreatedBy != nil {
				t.Fatalf("list createdBy = %v, want nil because runtime does not bind identity from provider-only filters", req.CreatedBy)
			}
			return datasciencesdk.ListProjectsResponse{
				Items: []datasciencesdk.ProjectSummary{
					makeSDKProjectSummary(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						datasciencesdk.ProjectLifecycleStateActive,
					),
				},
			}, nil
		},
		getProjectFn: func(_ context.Context, req datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want resolved project ID", req.ProjectId)
			}
			return datasciencesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					datasciencesdk.ProjectLifecycleStateActive,
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
		t.Fatal("CreateOrUpdate() should not requeue when the resolved project is ACTIVE")
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
}

func TestProjectServiceClientCreateOrUpdateUpdatesMutableDrift(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest datasciencesdk.UpdateProjectRequest

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			switch getCalls {
			case 1:
				return datasciencesdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						"stale description",
						datasciencesdk.ProjectLifecycleStateActive,
					),
				}, nil
			case 2:
				return datasciencesdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						"desired description",
						datasciencesdk.ProjectLifecycleStateActive,
					),
				}, nil
			default:
				t.Fatalf("unexpected GetProject() call %d", getCalls)
				return datasciencesdk.GetProjectResponse{}, nil
			}
		},
		updateProjectFn: func(_ context.Context, req datasciencesdk.UpdateProjectRequest) (datasciencesdk.UpdateProjectResponse, error) {
			updateRequest = req
			return datasciencesdk.UpdateProjectResponse{
				OpcRequestId: common.String("opc-update-1"),
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					datasciencesdk.ProjectLifecycleStateActive,
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
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once follow-up read sees ACTIVE")
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
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after synchronous update", resource.Status.OsokStatus.Async.Current)
	}
}

func TestProjectServiceClientCreateOrUpdateRejectsCompartmentDriftAgainstLiveProject(t *testing.T) {
	t.Parallel()

	updateCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return datasciencesdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..live",
					"project-alpha",
					"desired description",
					datasciencesdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		updateProjectFn: func(_ context.Context, _ datasciencesdk.UpdateProjectRequest) (datasciencesdk.UpdateProjectResponse, error) {
			updateCalls++
			t.Fatal("UpdateProject() should not be called when compartment drift requires replacement")
			return datasciencesdk.UpdateProjectResponse{}, nil
		},
	})

	resource := makeProjectResource()
	resource.Status.Id = "ocid1.project.oc1..existing"
	resource.Status.CompartmentId = resource.Spec.CompartmentId

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement failure for compartmentId drift", err)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateProject() calls = %d, want 0", updateCalls)
	}
	if resource.Status.CompartmentId != "ocid1.compartment.oc1..live" {
		t.Fatalf("status.compartmentId = %q, want live compartment after force-new validation", resource.Status.CompartmentId)
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE from live project read", resource.Status.LifecycleState)
	}
}

func TestProjectServiceClientDeleteFallsBackToListUsingTrackedIdentity(t *testing.T) {
	t.Parallel()

	notFound := errortest.NewServiceError(404, "NotFound", "")
	getCalls := 0
	deleteCalls := 0
	listCalls := 0

	resource := makeProjectResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"
	resource.Spec.DisplayName = "desired-name"
	resource.Status.Id = "ocid1.project.oc1..existing"
	resource.Status.CompartmentId = "ocid1.compartment.oc1..observed"
	resource.Status.DisplayName = "observed-name"

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			if getCalls == 1 {
				return datasciencesdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..observed",
						"observed-name",
						"desired description",
						datasciencesdk.ProjectLifecycleStateActive,
					),
				}, nil
			}
			return datasciencesdk.GetProjectResponse{}, notFound
		},
		deleteProjectFn: func(_ context.Context, req datasciencesdk.DeleteProjectRequest) (datasciencesdk.DeleteProjectResponse, error) {
			deleteCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("delete projectId = %v, want tracked project ID", req.ProjectId)
			}
			return datasciencesdk.DeleteProjectResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
		listProjectsFn: func(_ context.Context, req datasciencesdk.ListProjectsRequest) (datasciencesdk.ListProjectsResponse, error) {
			listCalls++
			if req.Id == nil || *req.Id != "ocid1.project.oc1..existing" {
				t.Fatalf("list id = %v, want tracked project ID", req.Id)
			}
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..observed" {
				t.Fatalf("list compartmentId = %v, want observed compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "observed-name" {
				t.Fatalf("list displayName = %v, want observed displayName", req.DisplayName)
			}
			if req.LifecycleState != "" {
				t.Fatalf("list lifecycleState = %q, want empty because delete fallback reuses the reviewed identity shape", req.LifecycleState)
			}
			if req.CreatedBy != nil {
				t.Fatalf("list createdBy = %v, want nil because delete fallback does not bind provider-only filters", req.CreatedBy)
			}
			return datasciencesdk.ListProjectsResponse{
				Items: []datasciencesdk.ProjectSummary{
					makeSDKProjectSummary(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..observed",
						"observed-name",
						datasciencesdk.ProjectLifecycleStateDeleting,
					),
				},
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer while delete is in progress")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteProject() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetProject() calls = %d, want 2", getCalls)
	}
	if listCalls != 1 {
		t.Fatalf("ListProjects() calls = %d, want 1", listCalls)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want delete tracker")
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
	}
	if resource.Status.OsokStatus.Async.Current.WorkRequestID != "wr-delete-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", resource.Status.OsokStatus.Async.Current.WorkRequestID, "wr-delete-1")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	}
}
