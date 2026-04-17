/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"fmt"

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
	newProjectServiceClient = func(manager *ProjectServiceManager) ProjectServiceClient {
		sdkClient, err := datasciencesdk.NewDataScienceClientWithConfigurationProvider(manager.Provider)
		config := newProjectRuntimeConfig(manager.Log, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize Project OCI client: %w", err)
		}
		return defaultProjectServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*datasciencev1beta1.Project](config),
		}
	}
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
	return generatedruntime.Config[*datasciencev1beta1.Project]{
		Kind:      "Project",
		SDKName:   "Project",
		Log:       log,
		Semantics: reviewedProjectRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &datasciencesdk.CreateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateProject(ctx, *request.(*datasciencesdk.CreateProjectRequest))
			},
			Fields: projectCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &datasciencesdk.GetProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetProject(ctx, *request.(*datasciencesdk.GetProjectRequest))
			},
			Fields: projectGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &datasciencesdk.ListProjectsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListProjects(ctx, *request.(*datasciencesdk.ListProjectsRequest))
			},
			Fields: projectListFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &datasciencesdk.UpdateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateProject(ctx, *request.(*datasciencesdk.UpdateProjectRequest))
			},
			Fields: projectUpdateFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &datasciencesdk.DeleteProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteProject(ctx, *request.(*datasciencesdk.DeleteProjectRequest))
			},
			Fields: projectDeleteFields(),
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
