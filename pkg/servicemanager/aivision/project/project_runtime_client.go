/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"fmt"

	aivisionsdk "github.com/oracle/oci-go-sdk/v65/aivision"
	aivisionv1beta1 "github.com/oracle/oci-service-operator/api/aivision/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type projectOCIClient interface {
	CreateProject(context.Context, aivisionsdk.CreateProjectRequest) (aivisionsdk.CreateProjectResponse, error)
	GetProject(context.Context, aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error)
	ListProjects(context.Context, aivisionsdk.ListProjectsRequest) (aivisionsdk.ListProjectsResponse, error)
	UpdateProject(context.Context, aivisionsdk.UpdateProjectRequest) (aivisionsdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, aivisionsdk.DeleteProjectRequest) (aivisionsdk.DeleteProjectResponse, error)
}

func init() {
	newProjectServiceClient = func(manager *ProjectServiceManager) ProjectServiceClient {
		sdkClient, err := aivisionsdk.NewAIServiceVisionClientWithConfigurationProvider(manager.Provider)
		config := newProjectRuntimeConfig(manager.Log, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize Project OCI client: %w", err)
		}
		return defaultProjectServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*aivisionv1beta1.Project](config),
		}
	}
}

func projectRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "aivision",
		FormalSlug:    "project",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id", "lifecycleState"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"definedTags", "description", "displayName", "freeformTags"},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func newProjectRuntimeConfig(
	log loggerutil.OSOKLogger,
	sdkClient projectOCIClient,
) generatedruntime.Config[*aivisionv1beta1.Project] {
	return generatedruntime.Config[*aivisionv1beta1.Project]{
		Kind:      "Project",
		SDKName:   "Project",
		Log:       log,
		Semantics: projectRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &aivisionsdk.CreateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.CreateProject(ctx, *request.(*aivisionsdk.CreateProjectRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateProjectDetails", RequestName: "CreateProjectDetails", Contribution: "body"},
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &aivisionsdk.GetProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.GetProject(ctx, *request.(*aivisionsdk.GetProjectRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &aivisionsdk.ListProjectsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.ListProjects(ctx, *request.(*aivisionsdk.ListProjectsRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query", LookupPaths: []string{"id", "ocid"}},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &aivisionsdk.UpdateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.UpdateProject(ctx, *request.(*aivisionsdk.UpdateProjectRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateProjectDetails", RequestName: "UpdateProjectDetails", Contribution: "body"},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &aivisionsdk.DeleteProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.DeleteProject(ctx, *request.(*aivisionsdk.DeleteProjectRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true},
			},
		},
	}
}
