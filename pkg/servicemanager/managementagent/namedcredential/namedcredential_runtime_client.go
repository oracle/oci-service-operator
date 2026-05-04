/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package namedcredential

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

var namedCredentialWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(managementagentsdk.OperationStatusCreated),
		string(managementagentsdk.OperationStatusAccepted),
		string(managementagentsdk.OperationStatusInProgress),
		string(managementagentsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(managementagentsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(managementagentsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(managementagentsdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(managementagentsdk.OperationTypesCreateNamedcredentials),
		string(managementagentsdk.ActionTypesCreated),
	},
	UpdateActionTokens: []string{
		string(managementagentsdk.OperationTypesUpdateNamedcredentials),
		string(managementagentsdk.ActionTypesUpdated),
	},
	DeleteActionTokens: []string{
		string(managementagentsdk.OperationTypesDeleteNamedcredentials),
		string(managementagentsdk.ActionTypesDeleted),
	},
}

type namedCredentialOCIClient interface {
	CreateNamedCredential(context.Context, managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error)
	GetNamedCredential(context.Context, managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error)
	ListNamedCredentials(context.Context, managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error)
	UpdateNamedCredential(context.Context, managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error)
	DeleteNamedCredential(context.Context, managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error)
	GetWorkRequest(context.Context, managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error)
}

type namedCredentialListCall func(context.Context, managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error)

type namedCredentialAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e namedCredentialAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e namedCredentialAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerNamedCredentialRuntimeHooksMutator(func(manager *NamedCredentialServiceManager, hooks *NamedCredentialRuntimeHooks) {
		client, initErr := newNamedCredentialOCIClient(manager)
		applyNamedCredentialRuntimeHooks(hooks, client, initErr)
	})
}

func newNamedCredentialOCIClient(manager *NamedCredentialServiceManager) (namedCredentialOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("named credential service manager is nil")
	}
	client, err := managementagentsdk.NewManagementAgentClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyNamedCredentialRuntimeHooks(
	hooks *NamedCredentialRuntimeHooks,
	client namedCredentialOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newNamedCredentialRuntimeSemantics()
	hooks.BuildCreateBody = buildNamedCredentialCreateBody
	hooks.BuildUpdateBody = buildNamedCredentialUpdateBody
	hooks.List.Fields = namedCredentialListFields()
	hooks.Async.Adapter = namedCredentialWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getNamedCredentialWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveNamedCredentialGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveNamedCredentialGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverNamedCredentialIDFromGeneratedWorkRequest
	hooks.Async.Message = namedCredentialGeneratedWorkRequestMessage
	hooks.DeleteHooks.HandleError = handleNamedCredentialDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapNamedCredentialDeleteSafety(client, initErr))
	if hooks.List.Call != nil {
		hooks.List.Call = listNamedCredentialsAllPages(hooks.List.Call)
	}
}

func newNamedCredentialRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "managementagent",
		FormalSlug:    "namedcredential",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(managementagentsdk.NamedCredentialLifecycleStateCreating)},
			UpdatingStates:     []string{string(managementagentsdk.NamedCredentialLifecycleStateUpdating)},
			ActiveStates:       []string{string(managementagentsdk.NamedCredentialLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(managementagentsdk.NamedCredentialLifecycleStateDeleting)},
			TerminalStates: []string{string(managementagentsdk.NamedCredentialLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"managementAgentId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"properties",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"managementAgentId",
				"name",
				"type",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "NamedCredential", Action: "CreateNamedCredential"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "NamedCredential", Action: "UpdateNamedCredential"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "NamedCredential", Action: "DeleteNamedCredential"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "NamedCredential", Action: "GetNamedCredential"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "NamedCredential", Action: "GetNamedCredential"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "NamedCredential", Action: "GetNamedCredential"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func namedCredentialListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ManagementAgentId",
			RequestName:  "managementAgentId",
			Contribution: "query",
			LookupPaths:  []string{"status.managementAgentId", "spec.managementAgentId", "managementAgentId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func newNamedCredentialServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client namedCredentialOCIClient,
) NamedCredentialServiceClient {
	manager := &NamedCredentialServiceManager{Log: log}
	hooks := newNamedCredentialRuntimeHooksWithOCIClient(client)
	applyNamedCredentialRuntimeHooks(&hooks, client, nil)
	delegate := defaultNamedCredentialServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*managementagentv1beta1.NamedCredential](
			buildNamedCredentialGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapNamedCredentialGeneratedClient(hooks, delegate)
}

func newNamedCredentialRuntimeHooksWithOCIClient(client namedCredentialOCIClient) NamedCredentialRuntimeHooks {
	return NamedCredentialRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*managementagentv1beta1.NamedCredential]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*managementagentv1beta1.NamedCredential]{},
		StatusHooks:     generatedruntime.StatusHooks[*managementagentv1beta1.NamedCredential]{},
		ParityHooks:     generatedruntime.ParityHooks[*managementagentv1beta1.NamedCredential]{},
		Async:           generatedruntime.AsyncHooks[*managementagentv1beta1.NamedCredential]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*managementagentv1beta1.NamedCredential]{},
		Create: runtimeOperationHooks[managementagentsdk.CreateNamedCredentialRequest, managementagentsdk.CreateNamedCredentialResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateNamedCredentialDetails", RequestName: "CreateNamedCredentialDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request managementagentsdk.CreateNamedCredentialRequest) (managementagentsdk.CreateNamedCredentialResponse, error) {
				if client == nil {
					return managementagentsdk.CreateNamedCredentialResponse{}, fmt.Errorf("named credential OCI client is nil")
				}
				return client.CreateNamedCredential(ctx, request)
			},
		},
		Get: runtimeOperationHooks[managementagentsdk.GetNamedCredentialRequest, managementagentsdk.GetNamedCredentialResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NamedCredentialId", RequestName: "namedCredentialId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request managementagentsdk.GetNamedCredentialRequest) (managementagentsdk.GetNamedCredentialResponse, error) {
				if client == nil {
					return managementagentsdk.GetNamedCredentialResponse{}, fmt.Errorf("named credential OCI client is nil")
				}
				return client.GetNamedCredential(ctx, request)
			},
		},
		List: runtimeOperationHooks[managementagentsdk.ListNamedCredentialsRequest, managementagentsdk.ListNamedCredentialsResponse]{
			Fields: namedCredentialListFields(),
			Call: func(ctx context.Context, request managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error) {
				if client == nil {
					return managementagentsdk.ListNamedCredentialsResponse{}, fmt.Errorf("named credential OCI client is nil")
				}
				return client.ListNamedCredentials(ctx, request)
			},
		},
		Update: runtimeOperationHooks[managementagentsdk.UpdateNamedCredentialRequest, managementagentsdk.UpdateNamedCredentialResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "NamedCredentialId", RequestName: "namedCredentialId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateNamedCredentialDetails", RequestName: "UpdateNamedCredentialDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request managementagentsdk.UpdateNamedCredentialRequest) (managementagentsdk.UpdateNamedCredentialResponse, error) {
				if client == nil {
					return managementagentsdk.UpdateNamedCredentialResponse{}, fmt.Errorf("named credential OCI client is nil")
				}
				return client.UpdateNamedCredential(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[managementagentsdk.DeleteNamedCredentialRequest, managementagentsdk.DeleteNamedCredentialResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NamedCredentialId", RequestName: "namedCredentialId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request managementagentsdk.DeleteNamedCredentialRequest) (managementagentsdk.DeleteNamedCredentialResponse, error) {
				if client == nil {
					return managementagentsdk.DeleteNamedCredentialResponse{}, fmt.Errorf("named credential OCI client is nil")
				}
				return client.DeleteNamedCredential(ctx, request)
			},
		},
		WrapGeneratedClient: []func(NamedCredentialServiceClient) NamedCredentialServiceClient{},
	}
}

func buildNamedCredentialCreateBody(_ context.Context, resource *managementagentv1beta1.NamedCredential, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("named credential resource is nil")
	}
	if err := validateNamedCredentialSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := managementagentsdk.CreateNamedCredentialDetails{
		Name:              common.String(strings.TrimSpace(resource.Spec.Name)),
		Type:              common.String(strings.TrimSpace(resource.Spec.Type)),
		ManagementAgentId: common.String(strings.TrimSpace(resource.Spec.ManagementAgentId)),
		Properties:        namedCredentialSDKProperties(resource.Spec.Properties),
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneNamedCredentialStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = namedCredentialDefinedTags(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildNamedCredentialUpdateBody(
	_ context.Context,
	resource *managementagentv1beta1.NamedCredential,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("named credential resource is nil")
	}
	if err := validateNamedCredentialSpec(resource.Spec); err != nil {
		return nil, false, err
	}

	current, ok := namedCredentialBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current NamedCredential response does not expose a NamedCredential body")
	}

	desiredProperties := namedCredentialSDKProperties(resource.Spec.Properties)
	body := managementagentsdk.UpdateNamedCredentialDetails{Properties: desiredProperties}
	updateNeeded := !namedCredentialPropertiesEqual(current.Properties, desiredProperties)

	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
		if !stringPtrEqual(current.Description, description) {
			updateNeeded = true
		}
	}
	if resource.Spec.FreeformTags != nil {
		desired := cloneNamedCredentialStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := namedCredentialDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateNamedCredentialSpec(spec managementagentv1beta1.NamedCredentialSpec) error {
	var missing []string
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.Type) == "" {
		missing = append(missing, "type")
	}
	if strings.TrimSpace(spec.ManagementAgentId) == "" {
		missing = append(missing, "managementAgentId")
	}
	if len(spec.Properties) == 0 {
		missing = append(missing, "properties")
	}
	if len(missing) > 0 {
		return fmt.Errorf("named credential spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	for _, property := range spec.Properties {
		if _, ok := managementagentsdk.GetMappingValueCategoryTypeEnum(property.ValueCategory); !ok {
			return fmt.Errorf("named credential property %q has unsupported valueCategory %q", property.Name, property.ValueCategory)
		}
	}
	return nil
}

func namedCredentialSDKProperties(properties []managementagentv1beta1.NamedCredentialProperty) []managementagentsdk.NamedCredentialProperty {
	if properties == nil {
		return nil
	}
	converted := make([]managementagentsdk.NamedCredentialProperty, 0, len(properties))
	for _, property := range properties {
		converted = append(converted, managementagentsdk.NamedCredentialProperty{
			Name:          common.String(strings.TrimSpace(property.Name)),
			Value:         common.String(property.Value),
			ValueCategory: managementagentsdk.ValueCategoryTypeEnum(strings.TrimSpace(property.ValueCategory)),
		})
	}
	return converted
}

func namedCredentialPropertiesEqual(current []managementagentsdk.NamedCredentialProperty, desired []managementagentsdk.NamedCredentialProperty) bool {
	currentComparable := comparableNamedCredentialProperties(current)
	desiredComparable := comparableNamedCredentialProperties(desired)
	if len(currentComparable) != len(desiredComparable) {
		return false
	}
	for index := range currentComparable {
		if currentComparable[index] != desiredComparable[index] {
			return false
		}
	}
	return true
}

type comparableNamedCredentialProperty struct {
	name          string
	value         string
	valueCategory string
}

func comparableNamedCredentialProperties(properties []managementagentsdk.NamedCredentialProperty) []comparableNamedCredentialProperty {
	comparable := make([]comparableNamedCredentialProperty, 0, len(properties))
	for _, property := range properties {
		comparable = append(comparable, comparableNamedCredentialProperty{
			name:          stringValue(property.Name),
			value:         stringValue(property.Value),
			valueCategory: string(property.ValueCategory),
		})
	}
	sort.Slice(comparable, func(i int, j int) bool {
		if comparable[i].name != comparable[j].name {
			return comparable[i].name < comparable[j].name
		}
		if comparable[i].valueCategory != comparable[j].valueCategory {
			return comparable[i].valueCategory < comparable[j].valueCategory
		}
		return comparable[i].value < comparable[j].value
	})
	return comparable
}

func namedCredentialBodyFromResponse(response any) (managementagentsdk.NamedCredential, bool) {
	if body, ok := namedCredentialBodyFromOperationResponse(response); ok {
		return body, true
	}
	return namedCredentialBodyFromResource(response)
}

func namedCredentialBodyFromOperationResponse(response any) (managementagentsdk.NamedCredential, bool) {
	switch current := response.(type) {
	case managementagentsdk.CreateNamedCredentialResponse:
		return current.NamedCredential, true
	case *managementagentsdk.CreateNamedCredentialResponse:
		if current == nil {
			return managementagentsdk.NamedCredential{}, false
		}
		return current.NamedCredential, true
	case managementagentsdk.GetNamedCredentialResponse:
		return current.NamedCredential, true
	case *managementagentsdk.GetNamedCredentialResponse:
		if current == nil {
			return managementagentsdk.NamedCredential{}, false
		}
		return current.NamedCredential, true
	case managementagentsdk.UpdateNamedCredentialResponse:
		return current.NamedCredential, true
	case *managementagentsdk.UpdateNamedCredentialResponse:
		if current == nil {
			return managementagentsdk.NamedCredential{}, false
		}
		return current.NamedCredential, true
	default:
		return managementagentsdk.NamedCredential{}, false
	}
}

func namedCredentialBodyFromResource(response any) (managementagentsdk.NamedCredential, bool) {
	switch current := response.(type) {
	case managementagentsdk.NamedCredential:
		return current, true
	case *managementagentsdk.NamedCredential:
		if current == nil {
			return managementagentsdk.NamedCredential{}, false
		}
		return *current, true
	case managementagentsdk.NamedCredentialSummary:
		return namedCredentialFromSummary(current), true
	case *managementagentsdk.NamedCredentialSummary:
		if current == nil {
			return managementagentsdk.NamedCredential{}, false
		}
		return namedCredentialFromSummary(*current), true
	default:
		return managementagentsdk.NamedCredential{}, false
	}
}

func namedCredentialFromSummary(summary managementagentsdk.NamedCredentialSummary) managementagentsdk.NamedCredential {
	return managementagentsdk.NamedCredential{
		Id:                summary.Id,
		Name:              summary.Name,
		Type:              summary.Type,
		ManagementAgentId: summary.ManagementAgentId,
		Properties:        summary.Properties,
		FreeformTags:      cloneNamedCredentialStringMap(summary.FreeformTags),
		DefinedTags:       cloneNamedCredentialDefinedTagMap(summary.DefinedTags),
		Description:       summary.Description,
		TimeCreated:       summary.TimeCreated,
		TimeUpdated:       summary.TimeUpdated,
		LifecycleState:    summary.LifecycleState,
		SystemTags:        cloneNamedCredentialDefinedTagMap(summary.SystemTags),
	}
}

func getNamedCredentialWorkRequest(
	ctx context.Context,
	client namedCredentialOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize named credential OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("named credential OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, managementagentsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveNamedCredentialGeneratedWorkRequestAction(workRequest any) (string, error) {
	namedCredentialWorkRequest, err := namedCredentialWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(namedCredentialWorkRequest.OperationType), nil
}

func resolveNamedCredentialGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	namedCredentialWorkRequest, err := namedCredentialWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := namedCredentialWorkRequestPhaseFromOperationType(namedCredentialWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverNamedCredentialIDFromGeneratedWorkRequest(
	_ *managementagentv1beta1.NamedCredential,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	namedCredentialWorkRequest, err := namedCredentialWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveNamedCredentialIDFromWorkRequest(namedCredentialWorkRequest, namedCredentialWorkRequestActionForPhase(phase))
}

func namedCredentialWorkRequestFromAny(workRequest any) (managementagentsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case managementagentsdk.WorkRequest:
		return current, nil
	case *managementagentsdk.WorkRequest:
		if current == nil {
			return managementagentsdk.WorkRequest{}, fmt.Errorf("named credential work request is nil")
		}
		return *current, nil
	default:
		return managementagentsdk.WorkRequest{}, fmt.Errorf("unexpected named credential work request type %T", workRequest)
	}
}

func namedCredentialWorkRequestPhaseFromOperationType(operationType managementagentsdk.OperationTypesEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case managementagentsdk.OperationTypesCreateNamedcredentials:
		return shared.OSOKAsyncPhaseCreate, true
	case managementagentsdk.OperationTypesUpdateNamedcredentials:
		return shared.OSOKAsyncPhaseUpdate, true
	case managementagentsdk.OperationTypesDeleteNamedcredentials:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func namedCredentialWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) managementagentsdk.ActionTypesEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return managementagentsdk.ActionTypesCreated
	case shared.OSOKAsyncPhaseUpdate:
		return managementagentsdk.ActionTypesUpdated
	case shared.OSOKAsyncPhaseDelete:
		return managementagentsdk.ActionTypesDeleted
	default:
		return ""
	}
}

func resolveNamedCredentialIDFromWorkRequest(workRequest managementagentsdk.WorkRequest, action managementagentsdk.ActionTypesEnum) (string, error) {
	if id, ok := resolveNamedCredentialIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveNamedCredentialIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("named credential work request %s does not expose a named credential identifier", stringValue(workRequest.Id))
}

func resolveNamedCredentialIDFromResources(
	resources []managementagentsdk.WorkRequestResource,
	action managementagentsdk.ActionTypesEnum,
	preferNamedCredentialOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferNamedCredentialOnly && !isNamedCredentialWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func namedCredentialGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	namedCredentialWorkRequest, err := namedCredentialWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("NamedCredential %s work request %s is %s", phase, stringValue(namedCredentialWorkRequest.Id), namedCredentialWorkRequest.Status)
}

func isNamedCredentialWorkRequestResource(resource managementagentsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "namedcredential", "namedcredentials", "named_credential", "named_credentials":
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "namedcredential")
}

func listNamedCredentialsAllPages(call namedCredentialListCall) namedCredentialListCall {
	return func(ctx context.Context, request managementagentsdk.ListNamedCredentialsRequest) (managementagentsdk.ListNamedCredentialsResponse, error) {
		if call == nil {
			return managementagentsdk.ListNamedCredentialsResponse{}, fmt.Errorf("named credential list operation is not configured")
		}

		accumulator := newNamedCredentialListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return managementagentsdk.ListNamedCredentialsResponse{}, err
			}
			accumulator.append(response)

			nextPage := stringValue(response.OpcNextPage)
			if nextPage == "" {
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return managementagentsdk.ListNamedCredentialsResponse{}, err
			}
		}
	}
}

type namedCredentialListAccumulator struct {
	response  managementagentsdk.ListNamedCredentialsResponse
	seenPages map[string]struct{}
}

func newNamedCredentialListAccumulator() namedCredentialListAccumulator {
	return namedCredentialListAccumulator{seenPages: map[string]struct{}{}}
}

func (a *namedCredentialListAccumulator) append(response managementagentsdk.ListNamedCredentialsResponse) {
	if a.response.RawResponse == nil {
		a.response.RawResponse = response.RawResponse
	}
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.OpcNextPage = response.OpcNextPage
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *namedCredentialListAccumulator) advance(request *managementagentsdk.ListNamedCredentialsRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("named credential list pagination repeated page token %q", nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return nil
}

func wrapNamedCredentialDeleteSafety(
	client namedCredentialOCIClient,
	initErr error,
) func(NamedCredentialServiceClient) NamedCredentialServiceClient {
	return func(delegate NamedCredentialServiceClient) NamedCredentialServiceClient {
		return namedCredentialDeleteSafetyClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	}
}

type namedCredentialDeleteSafetyClient struct {
	delegate NamedCredentialServiceClient
	client   namedCredentialOCIClient
	initErr  error
}

func (c namedCredentialDeleteSafetyClient) CreateOrUpdate(
	ctx context.Context,
	resource *managementagentv1beta1.NamedCredential,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c namedCredentialDeleteSafetyClient) Delete(
	ctx context.Context,
	resource *managementagentv1beta1.NamedCredential,
) (bool, error) {
	if deleted, handled, err := c.handlePendingWriteWorkRequest(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedCompletedDelete(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c namedCredentialDeleteSafetyClient) handlePendingWriteWorkRequest(
	ctx context.Context,
	resource *managementagentv1beta1.NamedCredential,
) (bool, bool, error) {
	if c.initErr != nil || c.client == nil || resource == nil {
		return false, false, nil
	}
	current, ok := currentPendingWriteWorkRequest(resource)
	if !ok {
		return false, false, nil
	}

	workRequest, err := c.fetchDeleteSafetyWorkRequest(ctx, current.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, true, err
	}
	return pendingWriteDeleteOutcome(current, workRequest)
}

func currentPendingWriteWorkRequest(resource *managementagentv1beta1.NamedCredential) (*shared.OSOKAsyncOperation, bool) {
	if resource == nil {
		return nil, false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest {
		return nil, false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return current, true
	default:
		return nil, false
	}
}

func pendingWriteDeleteOutcome(
	current *shared.OSOKAsyncOperation,
	workRequest managementagentsdk.WorkRequest,
) (bool, bool, error) {
	class, err := namedCredentialWorkRequestAsyncAdapter.Normalize(string(workRequest.Status))
	if err != nil {
		return false, true, err
	}
	if class == shared.OSOKAsyncClassPending {
		return false, true, nil
	}
	if class != shared.OSOKAsyncClassSucceeded {
		return false, true, fmt.Errorf("named credential %s work request %s finished with status %s; refusing delete until the write is resolved", current.Phase, current.WorkRequestID, workRequest.Status)
	}
	return false, false, nil
}

func (c namedCredentialDeleteSafetyClient) rejectAuthShapedCompletedDelete(
	ctx context.Context,
	resource *managementagentv1beta1.NamedCredential,
) error {
	if c.initErr != nil || c.client == nil || resource == nil {
		return nil
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return nil
	}
	workRequest, err := c.fetchDeleteSafetyWorkRequest(ctx, current.WorkRequestID)
	if err != nil || !namedCredentialDeleteWorkRequestSucceeded(workRequest) {
		return nil
	}
	return c.rejectAuthShapedConfirmRead(ctx, resource)
}

func (c namedCredentialDeleteSafetyClient) fetchDeleteSafetyWorkRequest(
	ctx context.Context,
	workRequestID string,
) (managementagentsdk.WorkRequest, error) {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return managementagentsdk.WorkRequest{}, fmt.Errorf("named credential delete safety requires a work request ID")
	}
	response, err := c.client.GetWorkRequest(ctx, managementagentsdk.GetWorkRequestRequest{WorkRequestId: common.String(workRequestID)})
	if err != nil {
		return managementagentsdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c namedCredentialDeleteSafetyClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *managementagentv1beta1.NamedCredential,
) error {
	namedCredentialID := trackedNamedCredentialID(resource)
	if namedCredentialID == "" {
		return nil
	}
	_, err := c.client.GetNamedCredential(ctx, managementagentsdk.GetNamedCredentialRequest{
		NamedCredentialId: common.String(namedCredentialID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("named credential delete work request completed but confirmation returned authorization-shaped not found; refusing to confirm deletion: %w", err)
}

func namedCredentialDeleteWorkRequestSucceeded(workRequest managementagentsdk.WorkRequest) bool {
	if workRequest.OperationType != managementagentsdk.OperationTypesDeleteNamedcredentials {
		return false
	}
	class, err := namedCredentialWorkRequestAsyncAdapter.Normalize(string(workRequest.Status))
	return err == nil && class == shared.OSOKAsyncClassSucceeded
}

func handleNamedCredentialDeleteError(resource *managementagentv1beta1.NamedCredential, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return namedCredentialAmbiguousNotFoundError{
		message:      "named credential delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func trackedNamedCredentialID(resource *managementagentv1beta1.NamedCredential) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func cloneNamedCredentialStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneNamedCredentialDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func namedCredentialDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrEqual(got *string, want string) bool {
	return got != nil && *got == want
}
