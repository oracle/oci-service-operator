/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"

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
	registerProjectRuntimeHooksMutator(func(_ *ProjectServiceManager, hooks *ProjectRuntimeHooks) {
		applyProjectRuntimeHooks(hooks)
	})
}

func applyProjectRuntimeHooks(hooks *ProjectRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = projectRuntimeSemantics()
	hooks.List.Fields = projectListFields()
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
	hooks := newProjectRuntimeHooksWithOCIClient(sdkClient)
	applyProjectRuntimeHooks(&hooks)
	return buildProjectGeneratedRuntimeConfig(&ProjectServiceManager{Log: log}, hooks)
}

func newProjectRuntimeHooksWithOCIClient(client projectOCIClient) ProjectRuntimeHooks {
	return ProjectRuntimeHooks{
		Semantics: newProjectRuntimeSemantics(),
		Create: runtimeOperationHooks[aivisionsdk.CreateProjectRequest, aivisionsdk.CreateProjectResponse]{
			Fields: projectCreateFields(),
			Call: func(ctx context.Context, request aivisionsdk.CreateProjectRequest) (aivisionsdk.CreateProjectResponse, error) {
				return client.CreateProject(ctx, request)
			},
		},
		Get: runtimeOperationHooks[aivisionsdk.GetProjectRequest, aivisionsdk.GetProjectResponse]{
			Fields: projectGetFields(),
			Call: func(ctx context.Context, request aivisionsdk.GetProjectRequest) (aivisionsdk.GetProjectResponse, error) {
				return client.GetProject(ctx, request)
			},
		},
		List: runtimeOperationHooks[aivisionsdk.ListProjectsRequest, aivisionsdk.ListProjectsResponse]{
			Fields: projectListFields(),
			Call: func(ctx context.Context, request aivisionsdk.ListProjectsRequest) (aivisionsdk.ListProjectsResponse, error) {
				return client.ListProjects(ctx, request)
			},
		},
		Update: runtimeOperationHooks[aivisionsdk.UpdateProjectRequest, aivisionsdk.UpdateProjectResponse]{
			Fields: projectUpdateFields(),
			Call: func(ctx context.Context, request aivisionsdk.UpdateProjectRequest) (aivisionsdk.UpdateProjectResponse, error) {
				return client.UpdateProject(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[aivisionsdk.DeleteProjectRequest, aivisionsdk.DeleteProjectResponse]{
			Fields: projectDeleteFields(),
			Call: func(ctx context.Context, request aivisionsdk.DeleteProjectRequest) (aivisionsdk.DeleteProjectResponse, error) {
				return client.DeleteProject(ctx, request)
			},
		},
	}
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
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query", LookupPaths: []string{"id", "ocid"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
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
