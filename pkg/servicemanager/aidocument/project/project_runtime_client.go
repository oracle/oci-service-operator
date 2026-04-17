/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"fmt"

	aidocumentsdk "github.com/oracle/oci-go-sdk/v65/aidocument"
	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type projectOCIClient interface {
	CreateProject(context.Context, aidocumentsdk.CreateProjectRequest) (aidocumentsdk.CreateProjectResponse, error)
	GetProject(context.Context, aidocumentsdk.GetProjectRequest) (aidocumentsdk.GetProjectResponse, error)
	ListProjects(context.Context, aidocumentsdk.ListProjectsRequest) (aidocumentsdk.ListProjectsResponse, error)
	UpdateProject(context.Context, aidocumentsdk.UpdateProjectRequest) (aidocumentsdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, aidocumentsdk.DeleteProjectRequest) (aidocumentsdk.DeleteProjectResponse, error)
}

func init() {
	newProjectServiceClient = func(manager *ProjectServiceManager) ProjectServiceClient {
		sdkClient, err := aidocumentsdk.NewAIServiceDocumentClientWithConfigurationProvider(manager.Provider)
		config := newProjectRuntimeConfig(manager.Log, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize Project OCI client: %w", err)
		}
		return defaultProjectServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*aidocumentv1beta1.Project](config),
		}
	}
}

func newProjectServiceClientWithOCIClient(log loggerutil.OSOKLogger, client projectOCIClient) ProjectServiceClient {
	return defaultProjectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*aidocumentv1beta1.Project](
			newProjectRuntimeConfig(log, client),
		),
	}
}

func newProjectRuntimeConfig(
	log loggerutil.OSOKLogger,
	client projectOCIClient,
) generatedruntime.Config[*aidocumentv1beta1.Project] {
	return generatedruntime.Config[*aidocumentv1beta1.Project]{
		Kind:      "Project",
		SDKName:   "Project",
		Log:       log,
		Semantics: newProjectRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &aidocumentsdk.CreateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateProject(ctx, *request.(*aidocumentsdk.CreateProjectRequest))
			},
			Fields: projectCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &aidocumentsdk.GetProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetProject(ctx, *request.(*aidocumentsdk.GetProjectRequest))
			},
			Fields: projectGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &aidocumentsdk.ListProjectsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListProjects(ctx, *request.(*aidocumentsdk.ListProjectsRequest))
			},
			Fields: projectListFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &aidocumentsdk.UpdateProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateProject(ctx, *request.(*aidocumentsdk.UpdateProjectRequest))
			},
			Fields: projectUpdateFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &aidocumentsdk.DeleteProjectRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteProject(ctx, *request.(*aidocumentsdk.DeleteProjectRequest))
			},
			Fields: projectDeleteFields(),
		},
	}
}

func newProjectRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "aidocument",
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
			MatchFields:        []string{"compartmentId", "displayName", "id"},
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
