/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"testing"

	aivisionsdk "github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/common"
	aivisionv1beta1 "github.com/oracle/oci-service-operator/api/aivision/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeProjectOCIClient struct {
	createFn func(context.Context, aivisionsdk.CreateProjectRequest) (aivisionsdk.CreateProjectResponse, error)
	getFn    func(context.Context, aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error)
	listFn   func(context.Context, aivisionsdk.ListProjectsRequest) (aivisionsdk.ListProjectsResponse, error)
	updateFn func(context.Context, aivisionsdk.UpdateProjectRequest) (aivisionsdk.UpdateProjectResponse, error)
	deleteFn func(context.Context, aivisionsdk.DeleteProjectRequest) (aivisionsdk.DeleteProjectResponse, error)
}

func (f *fakeProjectOCIClient) CreateProject(ctx context.Context, req aivisionsdk.CreateProjectRequest) (aivisionsdk.CreateProjectResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return aivisionsdk.CreateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) GetProject(ctx context.Context, req aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return aivisionsdk.GetProjectResponse{}, fakeProjectServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "missing",
	}
}

func (f *fakeProjectOCIClient) ListProjects(ctx context.Context, req aivisionsdk.ListProjectsRequest) (aivisionsdk.ListProjectsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return aivisionsdk.ListProjectsResponse{}, nil
}

func (f *fakeProjectOCIClient) UpdateProject(ctx context.Context, req aivisionsdk.UpdateProjectRequest) (aivisionsdk.UpdateProjectResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return aivisionsdk.UpdateProjectResponse{}, nil
}

func (f *fakeProjectOCIClient) DeleteProject(ctx context.Context, req aivisionsdk.DeleteProjectRequest) (aivisionsdk.DeleteProjectResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return aivisionsdk.DeleteProjectResponse{}, nil
}

type fakeProjectServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeProjectServiceError) Error() string          { return f.message }
func (f fakeProjectServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeProjectServiceError) GetMessage() string     { return f.message }
func (f fakeProjectServiceError) GetCode() string        { return f.code }
func (f fakeProjectServiceError) GetOpcRequestID() string {
	return ""
}

func newTestProjectServiceClient(
	manager *ProjectServiceManager,
	client projectOCIClient,
) ProjectServiceClient {
	if client == nil {
		client = &fakeProjectOCIClient{}
	}
	return defaultProjectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*aivisionv1beta1.Project](
			newProjectRuntimeConfig(manager.Log, client),
		),
	}
}

func newTestProjectManager(client projectOCIClient) *ProjectServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewProjectServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	return manager.WithClient(newTestProjectServiceClient(manager, client))
}

func makeSpecProject() *aivisionv1beta1.Project {
	return &aivisionv1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "vision-project"},
		Spec: aivisionv1beta1.ProjectSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "vision-project",
			Description:   "training set",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKProject(id, displayName string, lifecycleState aivisionsdk.ProjectLifecycleStateEnum) aivisionsdk.Project {
	return aivisionsdk.Project{
		Id:               common.String(id),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		LifecycleState:   lifecycleState,
		DisplayName:      common.String(displayName),
		Description:      common.String("training set"),
		LifecycleDetails: common.String("project ready"),
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKProjectSummary(id, displayName string, lifecycleState aivisionsdk.ProjectLifecycleStateEnum) aivisionsdk.ProjectSummary {
	return aivisionsdk.ProjectSummary{
		Id:               common.String(id),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		LifecycleState:   lifecycleState,
		DisplayName:      common.String(displayName),
		LifecycleDetails: common.String("project delete in progress"),
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func TestProjectCreateOrUpdate_CreateProjectsAndProjectsStatus(t *testing.T) {
	var createReq aivisionsdk.CreateProjectRequest
	getCalls := 0
	manager := newTestProjectManager(&fakeProjectOCIClient{
		createFn: func(_ context.Context, req aivisionsdk.CreateProjectRequest) (aivisionsdk.CreateProjectResponse, error) {
			createReq = req
			return aivisionsdk.CreateProjectResponse{
				Project: makeSDKProject("ocid1.project.oc1..create", "vision-project", aivisionsdk.ProjectLifecycleStateCreating),
			}, nil
		},
		getFn: func(_ context.Context, req aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error) {
			getCalls++
			if assert.NotNil(t, req.ProjectId) {
				assert.Equal(t, "ocid1.project.oc1..create", *req.ProjectId)
			}
			return aivisionsdk.GetProjectResponse{
				Project: makeSDKProject("ocid1.project.oc1..create", "vision-project", aivisionsdk.ProjectLifecycleStateActive),
			}, nil
		},
	})

	resource := makeSpecProject()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	if assert.NotNil(t, createReq.CompartmentId) {
		assert.Equal(t, "ocid1.compartment.oc1..example", *createReq.CompartmentId)
	}
	if assert.NotNil(t, createReq.DisplayName) {
		assert.Equal(t, "vision-project", *createReq.DisplayName)
	}
	if assert.NotNil(t, createReq.Description) {
		assert.Equal(t, "training set", *createReq.Description)
	}
	assert.Equal(t, map[string]string{"env": "dev"}, createReq.FreeformTags)
	assert.Equal(t, "42", createReq.DefinedTags["Operations"]["CostCenter"])
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, "ocid1.project.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.project.oc1..create", resource.Status.Id)
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
	assert.Equal(t, "vision-project", resource.Status.DisplayName)
	assert.Equal(t, "training set", resource.Status.Description)
}

func TestProjectCreateOrUpdate_NoMutableDriftSkipsUpdate(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newTestProjectManager(&fakeProjectOCIClient{
		getFn: func(_ context.Context, req aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error) {
			getCalls++
			if assert.NotNil(t, req.ProjectId) {
				assert.Equal(t, "ocid1.project.oc1..existing", *req.ProjectId)
			}
			return aivisionsdk.GetProjectResponse{
				Project: makeSDKProject("ocid1.project.oc1..existing", "vision-project", aivisionsdk.ProjectLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, _ aivisionsdk.UpdateProjectRequest) (aivisionsdk.UpdateProjectResponse, error) {
			updateCalls++
			return aivisionsdk.UpdateProjectResponse{}, nil
		},
	})

	resource := makeSpecProject()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.project.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
	assert.Equal(t, "vision-project", resource.Status.DisplayName)
}

func TestProjectDelete_FallbackListUsesIDQueryAndMarksDeletePending(t *testing.T) {
	var deleteReq aivisionsdk.DeleteProjectRequest
	var listReq aivisionsdk.ListProjectsRequest
	getCalls := 0
	manager := newTestProjectManager(&fakeProjectOCIClient{
		deleteFn: func(_ context.Context, req aivisionsdk.DeleteProjectRequest) (aivisionsdk.DeleteProjectResponse, error) {
			deleteReq = req
			return aivisionsdk.DeleteProjectResponse{}, nil
		},
		getFn: func(_ context.Context, req aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error) {
			getCalls++
			if assert.NotNil(t, req.ProjectId) {
				assert.Equal(t, "ocid1.project.oc1..existing", *req.ProjectId)
			}
			if getCalls == 1 {
				return aivisionsdk.GetProjectResponse{
					Project: makeSDKProject("ocid1.project.oc1..existing", "vision-project", aivisionsdk.ProjectLifecycleStateActive),
				}, nil
			}
			return aivisionsdk.GetProjectResponse{}, fakeProjectServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "missing",
			}
		},
		listFn: func(_ context.Context, req aivisionsdk.ListProjectsRequest) (aivisionsdk.ListProjectsResponse, error) {
			listReq = req
			if assert.NotNil(t, req.Id) {
				assert.Equal(t, "ocid1.project.oc1..existing", *req.Id)
			}
			if assert.NotNil(t, req.CompartmentId) {
				assert.Equal(t, "ocid1.compartment.oc1..example", *req.CompartmentId)
			}
			return aivisionsdk.ListProjectsResponse{
				ProjectCollection: aivisionsdk.ProjectCollection{
					Items: []aivisionsdk.ProjectSummary{
						makeSDKProjectSummary("ocid1.project.oc1..existing", "vision-project", aivisionsdk.ProjectLifecycleStateDeleting),
					},
				},
			}, nil
		},
	})

	resource := makeSpecProject()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.project.oc1..existing")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	if assert.NotNil(t, deleteReq.ProjectId) {
		assert.Equal(t, "ocid1.project.oc1..existing", *deleteReq.ProjectId)
	}
	if assert.NotNil(t, listReq.Id) {
		assert.Equal(t, "ocid1.project.oc1..existing", *listReq.Id)
	}
	assert.Equal(t, "DELETING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "OCI resource delete is in progress", resource.Status.OsokStatus.Message)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncPhaseDelete, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, shared.OSOKAsyncSourceLifecycle, resource.Status.OsokStatus.Async.Current.Source)
	}
}
