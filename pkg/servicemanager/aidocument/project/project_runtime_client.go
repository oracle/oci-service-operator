/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"strings"
	"time"

	aidocumentsdk "github.com/oracle/oci-go-sdk/v65/aidocument"
	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type projectOCIClient interface {
	CreateProject(context.Context, aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error)
	GetProject(context.Context, aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error)
	ListProjects(context.Context, aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error)
	UpdateProject(context.Context, aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, aidocumentsdk.DeleteProjectRequest) (aidocumentsdk.DeleteProjectResponse, error)
}

func init() {
	registerProjectRuntimeHooksMutator(func(_ *ProjectServiceManager, hooks *ProjectRuntimeHooks) {
		applyProjectRuntimeHooks(hooks)
	})
}

func applyProjectRuntimeHooks(hooks *ProjectRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedProjectRuntimeSemantics()
	hooks.Create.Fields = projectCreateFields()
	hooks.Get.Fields = projectGetFields()
	hooks.List.Fields = projectListFields()
	hooks.Update.Fields = projectUpdateFields()
	hooks.Delete.Fields = projectDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProjectServiceClient) ProjectServiceClient {
		return projectConvergenceClient{delegate: delegate}
	})
}

func newProjectServiceClientWithOCIClient(log loggerutil.OSOKLogger, client projectOCIClient) ProjectServiceClient {
	manager := &ProjectServiceManager{Log: log}
	hooks := newProjectRuntimeHooksWithOCIClient(client)
	applyProjectRuntimeHooks(&hooks)
	delegate := defaultProjectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*aidocumentv1beta1.Project](
			buildProjectGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapProjectGeneratedClient(hooks, delegate)
}

type projectConvergenceClient struct {
	delegate ProjectServiceClient
}

func (c projectConvergenceClient) CreateOrUpdate(
	ctx context.Context,
	resource *aidocumentv1beta1.Project,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil && projectUpdateStillInProgress(err) {
		response.IsSuccessful = true
		response.ShouldRequeue = true
		response.RequeueDuration = 5 * time.Second
		return response, nil
	}
	if err == nil && response.IsSuccessful && !response.ShouldRequeue && projectMutableStatusDrift(resource) {
		response.ShouldRequeue = true
		response.RequeueDuration = 5 * time.Second
	}
	return response, err
}

func projectUpdateStillInProgress(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.HTTPStatusCode == 409 && strings.Contains(strings.ToLower(err.Error()), "currently being modified")
}

func (c projectConvergenceClient) Delete(ctx context.Context, resource *aidocumentv1beta1.Project) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func projectMutableStatusDrift(resource *aidocumentv1beta1.Project) bool {
	if resource == nil {
		return false
	}
	if resource.Status.DisplayName != resource.Spec.DisplayName || resource.Status.Description != resource.Spec.Description {
		return true
	}
	return false
}

func newProjectRuntimeConfig(
	log loggerutil.OSOKLogger,
	client projectOCIClient,
) generatedruntime.Config[*aidocumentv1beta1.Project] {
	hooks := newProjectRuntimeHooksWithOCIClient(client)
	applyProjectRuntimeHooks(&hooks)
	return buildProjectGeneratedRuntimeConfig(&ProjectServiceManager{Log: log}, hooks)
}

func newProjectRuntimeHooksWithOCIClient(client projectOCIClient) ProjectRuntimeHooks {
	return ProjectRuntimeHooks{
		Semantics: newProjectRuntimeSemantics(),
		Create: runtimeOperationHooks[aidocumentsdk.CreateProjectRequest, aidocumentsdk.CreateProjectResponse]{
			Fields: projectCreateFields(),
			Call: func(ctx context.Context, request aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error) {
				return client.CreateProject(ctx, request)
			},
		},
		Get: runtimeOperationHooks[aidocumentsdk.GetProjectRequest, aidocumentsdk.GetProjectResponse]{
			Fields: projectGetFields(),
			Call: func(ctx context.Context, request aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error) {
				return client.GetProject(ctx, request)
			},
		},
		List: runtimeOperationHooks[aidocumentsdk.ListProjectsRequest, aidocumentsdk.ListProjectsResponse]{
			Fields: projectListFields(),
			Call: func(ctx context.Context, request aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error) {
				return client.ListProjects(ctx, request)
			},
		},
		Update: runtimeOperationHooks[aidocumentsdk.UpdateProjectRequest, aidocumentsdk.UpdateProjectResponse]{
			Fields: projectUpdateFields(),
			Call: func(ctx context.Context, request aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error) {
				return client.UpdateProject(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[aidocumentsdk.DeleteProjectRequest, aidocumentsdk.DeleteProjectResponse]{
			Fields: projectDeleteFields(),
			Call: func(ctx context.Context, request aidocumentsdk.DeleteProjectRequest) (aidocumentsdk.DeleteProjectResponse, error) {
				return client.DeleteProject(ctx, request)
			},
		},
	}
}

func reviewedProjectRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newProjectRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "id"},
	}
	semantics.Mutation.Mutable = []string{"description", "displayName"}
	semantics.Mutation.ForceNew = []string{"compartmentId", "definedTags", "freeformTags"}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func projectCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateProjectDetails", RequestName: "CreateProjectDetails", Contribution: "body"},
	}
}

func projectGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true},
	}
}

func projectListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func projectUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateProjectDetails", RequestName: "UpdateProjectDetails", Contribution: "body"},
	}
}

func projectDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true},
	}
}
