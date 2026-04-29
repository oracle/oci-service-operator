/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/util"
)

type repositoryOCIClient interface {
	CreateRepository(context.Context, artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error)
	GetRepository(context.Context, artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error)
	ListRepositories(context.Context, artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error)
	UpdateRepository(context.Context, artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error)
	DeleteRepository(context.Context, artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error)
}

type createGenericRepositoryRequest struct {
	CreateRepositoryDetails artifactssdk.CreateGenericRepositoryDetails `contributesTo:"body"`
	OpcRequestId            *string                                     `mandatory:"false" contributesTo:"header" name:"opc-request-id"`
	OpcRetryToken           *string                                     `mandatory:"false" contributesTo:"header" name:"opc-retry-token"`
	RequestMetadata         common.RequestMetadata
}

type updateGenericRepositoryRequest struct {
	RepositoryId            *string                                     `mandatory:"true" contributesTo:"path" name:"repositoryId"`
	UpdateRepositoryDetails artifactssdk.UpdateGenericRepositoryDetails `contributesTo:"body"`
	IfMatch                 *string                                     `mandatory:"false" contributesTo:"header" name:"if-match"`
	OpcRequestId            *string                                     `mandatory:"false" contributesTo:"header" name:"opc-request-id"`
	RequestMetadata         common.RequestMetadata
}

type ambiguousRepositoryNotFoundError struct {
	message      string
	opcRequestID string
}

type repositoryIdentity struct {
	compartmentID string
	displayName   string
}

func (e ambiguousRepositoryNotFoundError) Error() string {
	return e.message
}

func (e ambiguousRepositoryNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerRepositoryRuntimeHooksMutator(func(manager *RepositoryServiceManager, hooks *RepositoryRuntimeHooks) {
		applyRepositoryRuntimeHooks(manager, hooks)
	})
}

func applyRepositoryRuntimeHooks(manager *RepositoryServiceManager, hooks *RepositoryRuntimeHooks) {
	if hooks == nil {
		return
	}

	applyRepositoryRuntimeHookSettings(hooks)
	if manager == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(RepositoryServiceClient) RepositoryServiceClient {
		sdkClient, err := artifactssdk.NewArtifactsClientWithConfigurationProvider(manager.Provider)
		customHooks := newRepositoryDefaultRuntimeHooks(sdkClient)
		applyRepositoryRuntimeHookSettings(&customHooks)
		config := buildRepositoryConcreteGeneratedRuntimeConfig(manager, customHooks)
		if err != nil {
			config.InitError = fmt.Errorf("initialize repository OCI client: %w", err)
		}
		return defaultRepositoryServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*artifactsv1beta1.Repository](config),
		}
	})
}

func applyRepositoryRuntimeHookSettings(hooks *RepositoryRuntimeHooks) {
	hooks.Semantics = reviewedRepositoryRuntimeSemantics()
	hooks.BuildCreateBody = buildRepositoryCreateBody
	hooks.BuildUpdateBody = buildRepositoryUpdateBody
	hooks.Identity.Resolve = func(resource *artifactsv1beta1.Repository) (any, error) {
		return resolveRepositoryIdentity(resource)
	}
	hooks.Identity.RecordPath = recordRepositoryPathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardRepositoryExistingBeforeCreate
	hooks.List.Fields = repositoryListFields()
	wrapRepositoryReadAndDeleteCalls(hooks)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedRepositoryIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateRepositoryCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *artifactsv1beta1.Repository, currentID string) (any, error) {
		return confirmRepositoryDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleRepositoryDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyRepositoryDeleteConfirmOutcome
}

func buildRepositoryConcreteGeneratedRuntimeConfig(
	manager *RepositoryServiceManager,
	hooks RepositoryRuntimeHooks,
) generatedruntime.Config[*artifactsv1beta1.Repository] {
	return generatedruntime.Config[*artifactsv1beta1.Repository]{
		Kind:            "Repository",
		SDKName:         "Repository",
		Log:             manager.Log,
		Semantics:       hooks.Semantics,
		Identity:        hooks.Identity,
		Read:            hooks.Read,
		TrackedRecreate: hooks.TrackedRecreate,
		StatusHooks:     hooks.StatusHooks,
		ParityHooks:     hooks.ParityHooks,
		Async:           hooks.Async,
		DeleteHooks:     hooks.DeleteHooks,
		BuildCreateBody: hooks.BuildCreateBody,
		BuildUpdateBody: hooks.BuildUpdateBody,
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &createGenericRepositoryRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Create.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				typed := request.(*createGenericRepositoryRequest)
				return hooks.Create.Call(ctx, artifactssdk.CreateRepositoryRequest{
					CreateRepositoryDetails: typed.CreateRepositoryDetails,
					OpcRequestId:            typed.OpcRequestId,
					OpcRetryToken:           typed.OpcRetryToken,
					RequestMetadata:         typed.RequestMetadata,
				})
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &artifactssdk.GetRepositoryRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Get.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				return hooks.Get.Call(ctx, *request.(*artifactssdk.GetRepositoryRequest))
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &artifactssdk.ListRepositoriesRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.List.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				return hooks.List.Call(ctx, *request.(*artifactssdk.ListRepositoriesRequest))
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &updateGenericRepositoryRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Update.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				typed := request.(*updateGenericRepositoryRequest)
				return hooks.Update.Call(ctx, artifactssdk.UpdateRepositoryRequest{
					RepositoryId:            typed.RepositoryId,
					UpdateRepositoryDetails: typed.UpdateRepositoryDetails,
					IfMatch:                 typed.IfMatch,
					OpcRequestId:            typed.OpcRequestId,
					RequestMetadata:         typed.RequestMetadata,
				})
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &artifactssdk.DeleteRepositoryRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Delete.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				return hooks.Delete.Call(ctx, *request.(*artifactssdk.DeleteRepositoryRequest))
			},
		},
	}
}

func wrapRepositoryReadAndDeleteCalls(hooks *RepositoryRuntimeHooks) {
	getCall := hooks.Get.Call
	if getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeRepositoryNotFoundError(err, "read")
		}
	}

	listCall := hooks.List.Call
	if listCall != nil {
		hooks.List.Call = func(ctx context.Context, request artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			return listRepositoriesAllPages(ctx, listCall, request)
		}
	}

	deleteCall := hooks.Delete.Call
	if deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeRepositoryNotFoundError(err, "delete")
		}
	}
}

func newRepositoryServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client repositoryOCIClient,
) RepositoryServiceClient {
	hooks := newRepositoryRuntimeHooksWithOCIClient(client)
	applyRepositoryRuntimeHookSettings(&hooks)
	manager := &RepositoryServiceManager{Log: log}
	return defaultRepositoryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*artifactsv1beta1.Repository](
			buildRepositoryConcreteGeneratedRuntimeConfig(manager, hooks),
		),
	}
}

func newRepositoryRuntimeHooksWithOCIClient(client repositoryOCIClient) RepositoryRuntimeHooks {
	return RepositoryRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*artifactsv1beta1.Repository]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*artifactsv1beta1.Repository]{},
		StatusHooks:     generatedruntime.StatusHooks[*artifactsv1beta1.Repository]{},
		ParityHooks:     generatedruntime.ParityHooks[*artifactsv1beta1.Repository]{},
		Async:           generatedruntime.AsyncHooks[*artifactsv1beta1.Repository]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*artifactsv1beta1.Repository]{},
		Create: runtimeOperationHooks[artifactssdk.CreateRepositoryRequest, artifactssdk.CreateRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateRepositoryDetails", RequestName: "CreateRepositoryDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error) {
				return client.CreateRepository(ctx, request)
			},
		},
		Get: runtimeOperationHooks[artifactssdk.GetRepositoryRequest, artifactssdk.GetRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
				return client.GetRepository(ctx, request)
			},
		},
		List: runtimeOperationHooks[artifactssdk.ListRepositoriesRequest, artifactssdk.ListRepositoriesResponse]{
			Fields: repositoryListFields(),
			Call: func(ctx context.Context, request artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
				return client.ListRepositories(ctx, request)
			},
		},
		Update: runtimeOperationHooks[artifactssdk.UpdateRepositoryRequest, artifactssdk.UpdateRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateRepositoryDetails", RequestName: "UpdateRepositoryDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
				return client.UpdateRepository(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[artifactssdk.DeleteRepositoryRequest, artifactssdk.DeleteRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
				return client.DeleteRepository(ctx, request)
			},
		},
		WrapGeneratedClient: []func(RepositoryServiceClient) RepositoryServiceClient{},
	}
}

func reviewedRepositoryRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "artifacts",
		FormalSlug:    "repository",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(artifactssdk.RepositoryLifecycleStateAvailable)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(artifactssdk.RepositoryLifecycleStateDeleting)},
			TerminalStates: []string{string(artifactssdk.RepositoryLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "isImmutable", "repositoryType"},
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

func repositoryListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildRepositoryCreateBody(
	_ context.Context,
	resource *artifactsv1beta1.Repository,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("repository resource is nil")
	}
	return repositoryCreateDetailsFromSpec(resource.Spec)
}

func buildRepositoryUpdateBody(
	_ context.Context,
	resource *artifactsv1beta1.Repository,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return artifactssdk.UpdateGenericRepositoryDetails{}, false, fmt.Errorf("repository resource is nil")
	}
	current, ok := repositoryFromResponse(currentResponse)
	if !ok {
		return artifactssdk.UpdateGenericRepositoryDetails{}, false, fmt.Errorf("current repository response does not expose a repository body")
	}
	if err := validateRepositoryCreateOnlyDrift(resource.Spec, current); err != nil {
		return artifactssdk.UpdateGenericRepositoryDetails{}, false, err
	}

	desired, err := repositoryUpdateDetailsFromSpec(resource.Spec)
	if err != nil {
		return artifactssdk.UpdateGenericRepositoryDetails{}, false, err
	}

	updateDetails := artifactssdk.UpdateGenericRepositoryDetails{}
	updateNeeded := false

	updateNeeded = applyRepositoryStringUpdates(&updateDetails, desired, current) || updateNeeded
	updateNeeded = applyRepositoryTagUpdates(&updateDetails, desired, current) || updateNeeded
	if !updateNeeded {
		return artifactssdk.UpdateGenericRepositoryDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func applyRepositoryStringUpdates(
	updateDetails *artifactssdk.UpdateGenericRepositoryDetails,
	desired artifactssdk.UpdateGenericRepositoryDetails,
	current artifactssdk.Repository,
) bool {
	updateNeeded := false
	if desired.DisplayName != nil && !repositoryStringPtrEqual(current.GetDisplayName(), *desired.DisplayName) {
		updateDetails.DisplayName = common.String(strings.TrimSpace(*desired.DisplayName))
		updateNeeded = true
	}
	if desired.Description != nil && !repositoryStringPtrEqual(current.GetDescription(), *desired.Description) {
		updateDetails.Description = common.String(strings.TrimSpace(*desired.Description))
		updateNeeded = true
	}
	return updateNeeded
}

func applyRepositoryTagUpdates(
	updateDetails *artifactssdk.UpdateGenericRepositoryDetails,
	desired artifactssdk.UpdateGenericRepositoryDetails,
	current artifactssdk.Repository,
) bool {
	updateNeeded := false
	if desired.FreeformTags != nil && !reflect.DeepEqual(current.GetFreeformTags(), desired.FreeformTags) {
		updateDetails.FreeformTags = cloneRepositoryStringMap(desired.FreeformTags)
		updateNeeded = true
	}

	if desired.DefinedTags != nil && !reflect.DeepEqual(current.GetDefinedTags(), desired.DefinedTags) {
		updateDetails.DefinedTags = cloneRepositoryDefinedTags(desired.DefinedTags)
		updateNeeded = true
	}
	return updateNeeded
}

func repositoryCreateDetailsFromSpec(spec artifactsv1beta1.RepositorySpec) (artifactssdk.CreateGenericRepositoryDetails, error) {
	if err := validateRepositorySpec(spec); err != nil {
		return artifactssdk.CreateGenericRepositoryDetails{}, err
	}

	body := artifactssdk.CreateGenericRepositoryDetails{}
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		var err error
		body, err = repositoryCreateDetailsFromJSON(raw)
		if err != nil {
			return artifactssdk.CreateGenericRepositoryDetails{}, err
		}
		if err := validateRepositoryCreateIdentityOverrides(spec, body); err != nil {
			return artifactssdk.CreateGenericRepositoryDetails{}, err
		}
	}

	applyRepositoryCreateSpecDefaults(&body, spec)
	return body, nil
}

func validateRepositoryCreateIdentityOverrides(
	spec artifactsv1beta1.RepositorySpec,
	body artifactssdk.CreateGenericRepositoryDetails,
) error {
	var conflicts []string
	if body.CompartmentId != nil && !repositoryStringPtrEqual(body.CompartmentId, spec.CompartmentId) {
		conflicts = append(conflicts, "compartmentId")
	}
	if body.DisplayName != nil && strings.TrimSpace(spec.DisplayName) != "" &&
		!repositoryStringPtrEqual(body.DisplayName, spec.DisplayName) {
		conflicts = append(conflicts, "displayName")
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("repository jsonData identity conflicts with spec field(s): %s", strings.Join(conflicts, ", "))
}

func applyRepositoryCreateSpecDefaults(
	body *artifactssdk.CreateGenericRepositoryDetails,
	spec artifactsv1beta1.RepositorySpec,
) {
	applyRepositoryCreateIdentityDefaults(body, spec)
	applyRepositoryCreateMutableDefaults(body, spec)
	applyRepositoryCreateTagDefaults(body, spec)
}

func applyRepositoryCreateIdentityDefaults(
	body *artifactssdk.CreateGenericRepositoryDetails,
	spec artifactsv1beta1.RepositorySpec,
) {
	if body.CompartmentId == nil {
		body.CompartmentId = common.String(strings.TrimSpace(spec.CompartmentId))
	}
	if body.IsImmutable == nil {
		body.IsImmutable = common.Bool(spec.IsImmutable)
	}
}

func applyRepositoryCreateMutableDefaults(
	body *artifactssdk.CreateGenericRepositoryDetails,
	spec artifactsv1beta1.RepositorySpec,
) {
	if body.DisplayName == nil && strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	}
	if body.Description == nil && strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(strings.TrimSpace(spec.Description))
	}
}

func applyRepositoryCreateTagDefaults(
	body *artifactssdk.CreateGenericRepositoryDetails,
	spec artifactsv1beta1.RepositorySpec,
) {
	if body.FreeformTags == nil && spec.FreeformTags != nil {
		body.FreeformTags = cloneRepositoryStringMap(spec.FreeformTags)
	}
	if body.DefinedTags == nil && spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
}

func repositoryUpdateDetailsFromSpec(spec artifactsv1beta1.RepositorySpec) (artifactssdk.UpdateGenericRepositoryDetails, error) {
	if err := validateRepositorySpec(spec); err != nil {
		return artifactssdk.UpdateGenericRepositoryDetails{}, err
	}

	body := artifactssdk.UpdateGenericRepositoryDetails{}
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		var err error
		body, err = repositoryUpdateDetailsFromJSON(raw)
		if err != nil {
			return artifactssdk.UpdateGenericRepositoryDetails{}, err
		}
	}

	applyRepositoryUpdateSpecDefaults(&body, spec)
	return body, nil
}

func applyRepositoryUpdateSpecDefaults(
	body *artifactssdk.UpdateGenericRepositoryDetails,
	spec artifactsv1beta1.RepositorySpec,
) {
	if body.DisplayName == nil && strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	}
	if body.Description == nil && strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(strings.TrimSpace(spec.Description))
	}
	if body.FreeformTags == nil && spec.FreeformTags != nil {
		body.FreeformTags = cloneRepositoryStringMap(spec.FreeformTags)
	}
	if body.DefinedTags == nil && spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
}

func validateRepositorySpec(spec artifactsv1beta1.RepositorySpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if _, err := normalizedRepositoryType(spec.RepositoryType, spec.JsonData); err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("repository spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func repositoryCreateDetailsFromJSON(raw string) (artifactssdk.CreateGenericRepositoryDetails, error) {
	if _, err := normalizedRepositoryType("", raw); err != nil {
		return artifactssdk.CreateGenericRepositoryDetails{}, err
	}
	var details artifactssdk.CreateGenericRepositoryDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return artifactssdk.CreateGenericRepositoryDetails{}, fmt.Errorf("decode repository jsonData: %w", err)
	}
	return details, nil
}

func repositoryUpdateDetailsFromJSON(raw string) (artifactssdk.UpdateGenericRepositoryDetails, error) {
	if _, err := normalizedRepositoryType("", raw); err != nil {
		return artifactssdk.UpdateGenericRepositoryDetails{}, err
	}
	var details artifactssdk.UpdateGenericRepositoryDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return artifactssdk.UpdateGenericRepositoryDetails{}, fmt.Errorf("decode repository jsonData: %w", err)
	}
	return details, nil
}

func normalizedRepositoryType(specValue string, rawJSON string) (string, error) {
	repositoryType := strings.TrimSpace(specValue)
	if repositoryType == "" && strings.TrimSpace(rawJSON) != "" {
		var discriminator struct {
			RepositoryType string `json:"repositoryType"`
		}
		if err := json.Unmarshal([]byte(rawJSON), &discriminator); err != nil {
			return "", fmt.Errorf("decode repository jsonData discriminator: %w", err)
		}
		repositoryType = discriminator.RepositoryType
	}
	if repositoryType == "" {
		return string(artifactssdk.RepositoryRepositoryTypeGeneric), nil
	}
	if strings.EqualFold(strings.TrimSpace(repositoryType), string(artifactssdk.RepositoryRepositoryTypeGeneric)) {
		return string(artifactssdk.RepositoryRepositoryTypeGeneric), nil
	}
	return "", fmt.Errorf("unsupported repositoryType %q", repositoryType)
}

func guardRepositoryExistingBeforeCreate(
	_ context.Context,
	resource *artifactsv1beta1.Repository,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveRepositoryIdentity(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if identity.displayName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveRepositoryIdentity(resource *artifactsv1beta1.Repository) (repositoryIdentity, error) {
	if resource == nil {
		return repositoryIdentity{}, fmt.Errorf("repository resource is nil")
	}
	createDetails, err := repositoryCreateDetailsFromSpec(resource.Spec)
	if err != nil {
		return repositoryIdentity{}, err
	}
	return repositoryIdentity{
		compartmentID: strings.TrimSpace(repositoryStringValue(createDetails.CompartmentId)),
		displayName:   strings.TrimSpace(repositoryStringValue(createDetails.DisplayName)),
	}, nil
}

func recordRepositoryPathIdentity(resource *artifactsv1beta1.Repository, identity any) {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	typed, ok := identity.(repositoryIdentity)
	if !ok {
		return
	}
	resource.Status.CompartmentId = typed.compartmentID
	resource.Status.DisplayName = typed.displayName
}

func validateRepositoryCreateOnlyDriftForResponse(
	resource *artifactsv1beta1.Repository,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("repository resource is nil")
	}
	current, ok := repositoryFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current repository response does not expose a repository body")
	}
	return validateRepositoryCreateOnlyDrift(resource.Spec, current)
}

func validateRepositoryCreateOnlyDrift(
	spec artifactsv1beta1.RepositorySpec,
	current artifactssdk.Repository,
) error {
	desired, err := repositoryCreateDetailsFromSpec(spec)
	if err != nil {
		return err
	}
	desiredType, err := normalizedRepositoryType(spec.RepositoryType, spec.JsonData)
	if err != nil {
		return err
	}

	var drift []string
	if !repositoryStringPtrEqual(current.GetCompartmentId(), repositoryStringValue(desired.CompartmentId)) {
		drift = append(drift, "compartmentId")
	}
	if desired.IsImmutable != nil && !repositoryBoolPtrEqual(current.GetIsImmutable(), *desired.IsImmutable) {
		drift = append(drift, "isImmutable")
	}
	if currentType := repositoryTypeFromModel(current); currentType != "" && currentType != desiredType {
		drift = append(drift, "repositoryType")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("repository create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func listRepositoriesAllPages(
	ctx context.Context,
	list func(context.Context, artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error),
	request artifactssdk.ListRepositoriesRequest,
) (artifactssdk.ListRepositoriesResponse, error) {
	var combined artifactssdk.ListRepositoriesResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return artifactssdk.ListRepositoriesResponse{}, conservativeRepositoryNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item == nil || item.GetLifecycleState() == artifactssdk.RepositoryLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleRepositoryDeleteError(resource *artifactsv1beta1.Repository, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func confirmRepositoryDeleteRead(
	ctx context.Context,
	hooks *RepositoryRuntimeHooks,
	resource *artifactsv1beta1.Repository,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm repository delete: runtime hooks are nil")
	}
	if repositoryID := strings.TrimSpace(currentID); repositoryID != "" {
		return confirmRepositoryDeleteReadByID(ctx, hooks, repositoryID)
	}
	return confirmRepositoryDeleteReadByList(ctx, hooks, resource)
}

func confirmRepositoryDeleteReadByID(
	ctx context.Context,
	hooks *RepositoryRuntimeHooks,
	repositoryID string,
) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm repository delete: get hook is not configured")
	}
	response, err := hooks.Get.Call(ctx, artifactssdk.GetRepositoryRequest{RepositoryId: common.String(repositoryID)})
	return repositoryDeleteConfirmReadResponse(response, err)
}

func confirmRepositoryDeleteReadByList(
	ctx context.Context,
	hooks *RepositoryRuntimeHooks,
	resource *artifactsv1beta1.Repository,
) (any, error) {
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm repository delete: list hook is not configured")
	}
	request, identity, ok, err := repositoryDeleteListRequest(resource)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("confirm repository delete: repository identity is not recorded")
	}
	response, err := hooks.List.Call(ctx, request)
	if err != nil {
		return repositoryDeleteConfirmReadResponse(response, err)
	}
	for _, item := range response.Items {
		if repositorySummaryMatchesIdentity(identity, item) {
			return item, nil
		}
	}
	return nil, errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "Repository delete confirmation did not find a matching OCI repository",
	}
}

func repositoryDeleteListRequest(resource *artifactsv1beta1.Repository) (
	artifactssdk.ListRepositoriesRequest,
	repositoryIdentity,
	bool,
	error,
) {
	identity, err := resolveRepositoryIdentity(resource)
	if err != nil {
		return artifactssdk.ListRepositoriesRequest{}, repositoryIdentity{}, false, err
	}
	request := artifactssdk.ListRepositoriesRequest{}
	if identity.compartmentID != "" {
		request.CompartmentId = common.String(identity.compartmentID)
	}
	if identity.displayName != "" {
		request.DisplayName = common.String(identity.displayName)
	}
	return request, identity, request.CompartmentId != nil && request.DisplayName != nil, nil
}

func repositorySummaryMatchesIdentity(identity repositoryIdentity, summary artifactssdk.RepositorySummary) bool {
	if summary == nil || summary.GetLifecycleState() == artifactssdk.RepositoryLifecycleStateDeleted {
		return false
	}
	if identity.compartmentID != "" && !repositoryStringPtrEqual(summary.GetCompartmentId(), identity.compartmentID) {
		return false
	}
	if identity.displayName == "" || !repositoryStringPtrEqual(summary.GetDisplayName(), identity.displayName) {
		return false
	}
	return strings.TrimSpace(repositoryStringValue(summary.GetId())) != ""
}

func repositoryDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	convertedErr := conservativeRepositoryNotFoundError(err, "delete confirmation")
	if ambiguous, ok := asAmbiguousRepositoryNotFoundError(convertedErr); ok {
		return ambiguousRepositoryNotFoundError{
			message:      fmt.Sprintf("repository delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound: %s", err.Error()),
			opcRequestID: ambiguous.opcRequestID,
		}, nil
	}
	return nil, convertedErr
}

func applyRepositoryDeleteConfirmOutcome(
	resource *artifactsv1beta1.Repository,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	ambiguous, ok := response.(ambiguousRepositoryNotFoundError)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
	}
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
}

func asAmbiguousRepositoryNotFoundError(err error) (ambiguousRepositoryNotFoundError, bool) {
	var ambiguous ambiguousRepositoryNotFoundError
	if errors.As(err, &ambiguous) {
		return ambiguous, true
	}
	return ambiguousRepositoryNotFoundError{}, false
}

func conservativeRepositoryNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("repository %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousRepositoryNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousRepositoryNotFoundError{message: message}
}

func clearTrackedRepositoryIdentity(resource *artifactsv1beta1.Repository) {
	if resource == nil {
		return
	}
	resource.Status = artifactsv1beta1.RepositoryStatus{}
}

func repositoryFromResponse(response any) (artifactssdk.Repository, bool) {
	if current, ok := repositoryFromWriteResponse(response); ok {
		return current, true
	}
	if current, ok := repositoryFromReadResponse(response); ok {
		return current, true
	}
	return repositoryFromListItem(response)
}

func repositoryFromWriteResponse(response any) (artifactssdk.Repository, bool) {
	switch current := response.(type) {
	case artifactssdk.CreateRepositoryResponse:
		return repositoryFromWriteBody(current.Repository)
	case *artifactssdk.CreateRepositoryResponse:
		if current == nil {
			return nil, false
		}
		return repositoryFromWriteBody(current.Repository)
	case artifactssdk.UpdateRepositoryResponse:
		return repositoryFromWriteBody(current.Repository)
	case *artifactssdk.UpdateRepositoryResponse:
		if current == nil {
			return nil, false
		}
		return repositoryFromWriteBody(current.Repository)
	default:
		return nil, false
	}
}

func repositoryFromWriteBody(current artifactssdk.Repository) (artifactssdk.Repository, bool) {
	if current == nil {
		return nil, false
	}
	return current, true
}

func repositoryFromReadResponse(response any) (artifactssdk.Repository, bool) {
	switch current := response.(type) {
	case artifactssdk.GetRepositoryResponse:
		if current.Repository == nil {
			return nil, false
		}
		return current.Repository, true
	case *artifactssdk.GetRepositoryResponse:
		if current == nil || current.Repository == nil {
			return nil, false
		}
		return current.Repository, true
	case artifactssdk.Repository:
		if current == nil {
			return nil, false
		}
		return current, true
	default:
		return nil, false
	}
}

func repositoryFromListItem(response any) (artifactssdk.Repository, bool) {
	switch current := response.(type) {
	case artifactssdk.RepositorySummary:
		if current == nil {
			return nil, false
		}
		return repositoryFromSummary(current), true
	default:
		return nil, false
	}
}

func repositoryFromSummary(summary artifactssdk.RepositorySummary) artifactssdk.GenericRepository {
	return artifactssdk.GenericRepository{
		Id:             summary.GetId(),
		DisplayName:    summary.GetDisplayName(),
		CompartmentId:  summary.GetCompartmentId(),
		Description:    summary.GetDescription(),
		IsImmutable:    summary.GetIsImmutable(),
		LifecycleState: summary.GetLifecycleState(),
		FreeformTags:   cloneRepositoryStringMap(summary.GetFreeformTags()),
		DefinedTags:    cloneRepositoryDefinedTags(summary.GetDefinedTags()),
		TimeCreated:    summary.GetTimeCreated(),
	}
}

func repositoryTypeFromModel(model any) string {
	payload, err := json.Marshal(model)
	if err != nil {
		return ""
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return ""
	}
	raw, ok := values["repositoryType"]
	if !ok {
		return string(artifactssdk.RepositoryRepositoryTypeGeneric)
	}
	return strings.ToUpper(strings.TrimSpace(fmt.Sprint(raw)))
}

func cloneRepositoryStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneRepositoryDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func repositoryBoolPtrEqual(current *bool, desired bool) bool {
	return current != nil && *current == desired
}

func repositoryStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(repositoryStringValue(current)) == strings.TrimSpace(desired)
}

func repositoryStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
