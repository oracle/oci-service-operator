/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"

	datasciencesdk "github.com/oracle/oci-go-sdk/v65/datascience"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type projectOCIClient interface {
	CreateProject(context.Context, datasciencesdk.CreateProjectRequest) (datasciencesdk.CreateProjectResponse, error)
	GetProject(context.Context, datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error)
	ListProjects(context.Context, datasciencesdk.ListProjectsRequest) (datasciencesdk.ListProjectsResponse, error)
	UpdateProject(context.Context, datasciencesdk.UpdateProjectRequest) (datasciencesdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, datasciencesdk.DeleteProjectRequest) (datasciencesdk.DeleteProjectResponse, error)
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
	hooks.List.Fields = projectListFields()
}

func newProjectServiceClientWithOCIClient(log loggerutil.OSOKLogger, client projectOCIClient) ProjectServiceClient {
	return defaultProjectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasciencev1beta1.Project](
			newProjectRuntimeConfig(log, client),
		),
	}
}

func newProjectRuntimeConfig(
	log loggerutil.OSOKLogger,
	client projectOCIClient,
) generatedruntime.Config[*datasciencev1beta1.Project] {
	hooks := newProjectRuntimeHooksWithOCIClient(client)
	applyProjectRuntimeHooks(&hooks)
	return buildProjectGeneratedRuntimeConfig(&ProjectServiceManager{Log: log}, hooks)
}

func newProjectRuntimeHooksWithOCIClient(client projectOCIClient) ProjectRuntimeHooks {
	return ProjectRuntimeHooks{
		Semantics: newProjectRuntimeSemantics(),
		Create: runtimeOperationHooks[datasciencesdk.CreateProjectRequest, datasciencesdk.CreateProjectResponse]{
			Fields: projectCreateFields(),
			Call: func(ctx context.Context, request datasciencesdk.CreateProjectRequest) (datasciencesdk.CreateProjectResponse, error) {
				return client.CreateProject(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasciencesdk.GetProjectRequest, datasciencesdk.GetProjectResponse]{
			Fields: projectGetFields(),
			Call: func(ctx context.Context, request datasciencesdk.GetProjectRequest) (datasciencesdk.GetProjectResponse, error) {
				return client.GetProject(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasciencesdk.ListProjectsRequest, datasciencesdk.ListProjectsResponse]{
			Fields: projectListFields(),
			Call: func(ctx context.Context, request datasciencesdk.ListProjectsRequest) (datasciencesdk.ListProjectsResponse, error) {
				return client.ListProjects(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasciencesdk.UpdateProjectRequest, datasciencesdk.UpdateProjectResponse]{
			Fields: projectUpdateFields(),
			Call: func(ctx context.Context, request datasciencesdk.UpdateProjectRequest) (datasciencesdk.UpdateProjectResponse, error) {
				return client.UpdateProject(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasciencesdk.DeleteProjectRequest, datasciencesdk.DeleteProjectResponse]{
			Fields: projectDeleteFields(),
			Call: func(ctx context.Context, request datasciencesdk.DeleteProjectRequest) (datasciencesdk.DeleteProjectResponse, error) {
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
