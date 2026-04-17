/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"fmt"

	ailanguagesdk "github.com/oracle/oci-go-sdk/v65/ailanguage"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type projectOCIClient interface {
	CreateProject(context.Context, ailanguagesdk.CreateProjectRequest) (ailanguagesdk.CreateProjectResponse, error)
	GetProject(context.Context, ailanguagesdk.GetProjectRequest) (ailanguagesdk.GetProjectResponse, error)
	ListProjects(context.Context, ailanguagesdk.ListProjectsRequest) (ailanguagesdk.ListProjectsResponse, error)
	UpdateProject(context.Context, ailanguagesdk.UpdateProjectRequest) (ailanguagesdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, ailanguagesdk.DeleteProjectRequest) (ailanguagesdk.DeleteProjectResponse, error)
}

func init() {
	newProjectServiceClient = func(manager *ProjectServiceManager) ProjectServiceClient {
		sdkClient, err := ailanguagesdk.NewAIServiceLanguageClientWithConfigurationProvider(manager.Provider)
		config := newProjectRuntimeConfig(manager.Log, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize Project OCI client: %w", err)
		}
		return defaultProjectServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*ailanguagev1beta1.Project](config),
		}
	}
}

func newProjectServiceClientWithOCIClient(log loggerutil.OSOKLogger, client projectOCIClient) ProjectServiceClient {
	return defaultProjectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*ailanguagev1beta1.Project](newProjectRuntimeConfig(log, client)),
	}
}

func newProjectRuntimeConfig(
	log loggerutil.OSOKLogger,
	client projectOCIClient,
) generatedruntime.Config[*ailanguagev1beta1.Project] {
	return generatedruntime.Config[*ailanguagev1beta1.Project]{
		Kind:      "Project",
		SDKName:   "Project",
		Log:       log,
		Semantics: reviewedProjectRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ailanguagesdk.CreateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateProject(ctx, *request.(*ailanguagesdk.CreateProjectRequest))
			},
			Fields: projectCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ailanguagesdk.GetProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetProject(ctx, *request.(*ailanguagesdk.GetProjectRequest))
			},
			Fields: projectGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ailanguagesdk.ListProjectsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListProjects(ctx, *request.(*ailanguagesdk.ListProjectsRequest))
			},
			Fields: projectListFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ailanguagesdk.UpdateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateProject(ctx, *request.(*ailanguagesdk.UpdateProjectRequest))
			},
			Fields: projectUpdateFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &ailanguagesdk.DeleteProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteProject(ctx, *request.(*ailanguagesdk.DeleteProjectRequest))
			},
			Fields: projectDeleteFields(),
		},
	}
}

func reviewedProjectRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newProjectRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "projectId"},
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
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "query", PreferResourceID: true},
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
