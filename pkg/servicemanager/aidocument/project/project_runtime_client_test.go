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
	"time"

	"github.com/go-logr/logr"
	aidocumentsdk "github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/common"
	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeProjectOCIClient struct {
	createProjectFn func(context.Context, aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error)
	getProjectFn    func(context.Context, aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error)
	listProjectsFn  func(context.Context, aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error)
	updateProjectFn func(context.Context, aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error)
	deleteProjectFn func(context.Context, aidocumentsdk.DeleteProjectRequest) (aidocumentsdk.DeleteProjectResponse, error)
}

func (f *fakeProjectOCIClient) CreateProject(ctx context.Context, req aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error) {
	if f.createProjectFn != nil {
		return f.createProjectFn(ctx, req)
	}
	return aidocumentsdk.CreateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetProject(ctx context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
	if f.getProjectFn != nil {
		return f.getProjectFn(ctx, req)
	}
	return aidocumentsdk.GetProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) ListProjects(ctx context.Context, req aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error) {
	if f.listProjectsFn != nil {
		return f.listProjectsFn(ctx, req)
	}
	return aidocumentsdk.ListProjectsResponse{}, nil
}

func (f *fakeProjectOCIClient) UpdateProject(ctx context.Context, req aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error) {
	if f.updateProjectFn != nil {
		return f.updateProjectFn(ctx, req)
	}
	return aidocumentsdk.UpdateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) DeleteProject(ctx context.Context, req aidocumentsdk.DeleteProjectRequest) (aidocumentsdk.DeleteProjectResponse, error) {
	if f.deleteProjectFn != nil {
		return f.deleteProjectFn(ctx, req)
	}
	return aidocumentsdk.DeleteProjectResponse{}, nil
}

func testProjectClient(fake *fakeProjectOCIClient) ProjectServiceClient {
	return newProjectServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeProjectResource() *aidocumentv1beta1.Project {
	return &aidocumentv1beta1.Project{
		Spec: aidocumentv1beta1.ProjectSpec{
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
	state aidocumentsdk.ProjectLifecycleStateEnum,
) aidocumentsdk.Project {
	project := aidocumentsdk.Project{
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
	state aidocumentsdk.ProjectLifecycleStateEnum,
) aidocumentsdk.ProjectSummary {
	return aidocumentsdk.ProjectSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
	}
}

func makeSDKProjectLock(
	lockType aidocumentsdk.ResourceLockTypeEnum,
	compartmentID string,
	relatedResourceID string,
	message string,
	createdAt time.Time,
) aidocumentsdk.ResourceLock {
	lock := aidocumentsdk.ResourceLock{
		Type:          lockType,
		CompartmentId: common.String(compartmentID),
	}
	if relatedResourceID != "" {
		lock.RelatedResourceId = common.String(relatedResourceID)
	}
	if message != "" {
		lock.Message = common.String(message)
	}
	if !createdAt.IsZero() {
		sdkTime := common.SDKTime{Time: createdAt.UTC()}
		lock.TimeCreated = &sdkTime
	}
	return lock
}

func requireAsyncCurrent(t *testing.T, resource *aidocumentv1beta1.Project, phase shared.OSOKAsyncPhase, workRequestID string) {
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

func TestProjectRuntimeHooksUseReviewedRequestFields(t *testing.T) {
	t.Parallel()

	hooks := newProjectRuntimeHooks(&ProjectServiceManager{}, aidocumentsdk.AIServiceDocumentClient{})

	if !reflect.DeepEqual(hooks.Create.Fields, projectCreateFields()) {
		t.Fatalf("create fields = %#v, want %#v", hooks.Create.Fields, projectCreateFields())
	}
	if !reflect.DeepEqual(hooks.Get.Fields, projectGetFields()) {
		t.Fatalf("get fields = %#v, want %#v", hooks.Get.Fields, projectGetFields())
	}
	if !reflect.DeepEqual(hooks.List.Fields, projectListFields()) {
		t.Fatalf("list fields = %#v, want %#v", hooks.List.Fields, projectListFields())
	}
	if !reflect.DeepEqual(hooks.Update.Fields, projectUpdateFields()) {
		t.Fatalf("update fields = %#v, want %#v", hooks.Update.Fields, projectUpdateFields())
	}
	if !reflect.DeepEqual(hooks.Delete.Fields, projectDeleteFields()) {
		t.Fatalf("delete fields = %#v, want %#v", hooks.Delete.Fields, projectDeleteFields())
	}
	if hooks.Semantics == nil {
		t.Fatal("semantics = nil, want reviewed semantics")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("semantics.auxiliaryOperations = %#v, want reviewed omission of ChangeProjectCompartment", hooks.Semantics.AuxiliaryOperations)
	}
}

func TestProjectServiceClientCreateOrUpdateCreatesAndRequeuesWhileCreating(t *testing.T) {
	t.Parallel()

	var createRequest aidocumentsdk.CreateProjectRequest
	var getRequest aidocumentsdk.GetProjectRequest
	lockCreatedAt := time.Date(2026, time.April, 28, 1, 2, 3, 0, time.UTC)

	client := testProjectClient(&fakeProjectOCIClient{
		createProjectFn: func(_ context.Context, req aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error) {
			createRequest = req
			return aidocumentsdk.CreateProjectResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				Project: makeSDKProject(
					"ocid1.project.oc1..created",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					aidocumentsdk.ProjectLifecycleStateCreating,
				),
			}, nil
		},
		getProjectFn: func(_ context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
			getRequest = req
			project := makeSDKProject(
				"ocid1.project.oc1..created",
				"ocid1.compartment.oc1..example",
				"project-alpha",
				"desired description",
				aidocumentsdk.ProjectLifecycleStateCreating,
			)
			project.Locks = []aidocumentsdk.ResourceLock{
				makeSDKProjectLock(
					aidocumentsdk.ResourceLockTypeDelete,
					"ocid1.compartment.oc1..example",
					"ocid1.model.oc1..example",
					"protect linked model",
					lockCreatedAt,
				),
			}
			return aidocumentsdk.GetProjectResponse{
				Project: project,
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
	if createRequest.CreateProjectDetails.DisplayName == nil || *createRequest.CreateProjectDetails.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %v, want %q", createRequest.CreateProjectDetails.DisplayName, resource.Spec.DisplayName)
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
	if len(resource.Status.Locks) != 1 {
		t.Fatalf("status.locks length = %d, want 1", len(resource.Status.Locks))
	}
	if resource.Status.Locks[0].Type != string(aidocumentsdk.ResourceLockTypeDelete) {
		t.Fatalf("status.locks[0].type = %q, want %q", resource.Status.Locks[0].Type, aidocumentsdk.ResourceLockTypeDelete)
	}
	if resource.Status.Locks[0].CompartmentId != "ocid1.compartment.oc1..example" {
		t.Fatalf("status.locks[0].compartmentId = %q, want example compartment", resource.Status.Locks[0].CompartmentId)
	}
	if resource.Status.Locks[0].RelatedResourceId != "ocid1.model.oc1..example" {
		t.Fatalf("status.locks[0].relatedResourceId = %q, want linked model ID", resource.Status.Locks[0].RelatedResourceId)
	}
	if resource.Status.Locks[0].Message != "protect linked model" {
		t.Fatalf("status.locks[0].message = %q, want lock message", resource.Status.Locks[0].Message)
	}
	if resource.Status.Locks[0].TimeCreated != lockCreatedAt.Format(time.RFC3339Nano) {
		t.Fatalf("status.locks[0].timeCreated = %q, want %q", resource.Status.Locks[0].TimeCreated, lockCreatedAt.Format(time.RFC3339Nano))
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "CREATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "CREATING")
	}
}

func TestProjectServiceClientCreateOrUpdateResolvesExistingProjectBeforeCreate(t *testing.T) {
	t.Parallel()

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := testProjectClient(&fakeProjectOCIClient{
		createProjectFn: func(_ context.Context, _ aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error) {
			createCalls++
			return aidocumentsdk.CreateProjectResponse{}, nil
		},
		listProjectsFn: func(_ context.Context, req aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error) {
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
			return aidocumentsdk.ListProjectsResponse{
				ProjectCollection: aidocumentsdk.ProjectCollection{
					Items: []aidocumentsdk.ProjectSummary{
						makeSDKProjectSummary(
							"ocid1.project.oc1..existing",
							"ocid1.compartment.oc1..example",
							"project-alpha",
							aidocumentsdk.ProjectLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getProjectFn: func(_ context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want resolved project ID", req.ProjectId)
			}
			return aidocumentsdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					aidocumentsdk.ProjectLifecycleStateActive,
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

func TestProjectServiceClientCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return aidocumentsdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..example",
					"project-alpha",
					"desired description",
					aidocumentsdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		updateProjectFn: func(_ context.Context, req aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error) {
			t.Fatalf("UpdateProject() should not be called when mutable fields already match: %+v", req)
			return aidocumentsdk.UpdateProjectResponse{}, nil
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
	var updateRequest aidocumentsdk.UpdateProjectRequest

	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			switch getCalls {
			case 1:
				return aidocumentsdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						"stale description",
						aidocumentsdk.ProjectLifecycleStateActive,
					),
				}, nil
			case 2:
				return aidocumentsdk.GetProjectResponse{
					Project: makeSDKProject(
						"ocid1.project.oc1..existing",
						"ocid1.compartment.oc1..example",
						"project-alpha",
						"desired description",
						aidocumentsdk.ProjectLifecycleStateUpdating,
					),
				}, nil
			default:
				t.Fatalf("unexpected GetProject() call %d", getCalls)
				return aidocumentsdk.GetProjectResponse{}, nil
			}
		},
		updateProjectFn: func(_ context.Context, req aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error) {
			updateRequest = req
			return aidocumentsdk.UpdateProjectResponse{
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
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the follow-up read reports UPDATING")
	}
	if updateRequest.ProjectId == nil || *updateRequest.ProjectId != "ocid1.project.oc1..existing" {
		t.Fatalf("update projectId = %v, want tracked project ID", updateRequest.ProjectId)
	}
	if updateRequest.IsLockOverride != nil {
		t.Fatalf("update isLockOverride = %#v, want reviewed hook field omission", updateRequest.IsLockOverride)
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
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "UPDATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "UPDATING")
	}
}

func TestProjectServiceClientCreateOrUpdateRejectsCompartmentDrift(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testProjectClient(&fakeProjectOCIClient{
		getProjectFn: func(_ context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return aidocumentsdk.GetProjectResponse{
				Project: makeSDKProject(
					"ocid1.project.oc1..existing",
					"ocid1.compartment.oc1..observed",
					"project-alpha",
					"desired description",
					aidocumentsdk.ProjectLifecycleStateActive,
				),
			}, nil
		},
		updateProjectFn: func(_ context.Context, req aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error) {
			updateCalls++
			return aidocumentsdk.UpdateProjectResponse{}, nil
		},
	})

	resource := makeProjectResource()
	resource.Status.Id = "ocid1.project.oc1..existing"
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want replacement validation error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when compartment drift requires replacement")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateProject() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "Project formal semantics require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want replacement message", err)
	}
}

func TestProjectServiceClientDeleteFallsBackToListUsingID(t *testing.T) {
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
		getProjectFn: func(_ context.Context, req aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
			getCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("get projectId = %v, want tracked project ID", req.ProjectId)
			}
			return aidocumentsdk.GetProjectResponse{}, notFound
		},
		listProjectsFn: func(_ context.Context, req aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error) {
			listCalls++
			if req.Id == nil || *req.Id != "ocid1.project.oc1..existing" {
				t.Fatalf("list id = %v, want tracked project ID", req.Id)
			}
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..observed" {
				t.Fatalf("list compartmentId = %v, want observed status compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "observed-name" {
				t.Fatalf("list displayName = %v, want observed status displayName", req.DisplayName)
			}

			state := aidocumentsdk.ProjectLifecycleStateActive
			if listCalls > 1 {
				state = aidocumentsdk.ProjectLifecycleStateDeleting
			}
			return aidocumentsdk.ListProjectsResponse{
				ProjectCollection: aidocumentsdk.ProjectCollection{
					Items: []aidocumentsdk.ProjectSummary{
						makeSDKProjectSummary(
							"ocid1.project.oc1..existing",
							"ocid1.compartment.oc1..observed",
							"observed-name",
							state,
						),
					},
				},
			}, nil
		},
		deleteProjectFn: func(_ context.Context, req aidocumentsdk.DeleteProjectRequest) (aidocumentsdk.DeleteProjectResponse, error) {
			deleteCalls++
			if req.ProjectId == nil || *req.ProjectId != "ocid1.project.oc1..existing" {
				t.Fatalf("delete projectId = %v, want tracked project ID", req.ProjectId)
			}
			if req.IsLockOverride != nil {
				t.Fatalf("delete isLockOverride = %#v, want reviewed hook field omission", req.IsLockOverride)
			}
			return aidocumentsdk.DeleteProjectResponse{
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
