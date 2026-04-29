/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package governanceinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	accessgovernancecpsdk "github.com/oracle/oci-go-sdk/v65/accessgovernancecp"
	"github.com/oracle/oci-go-sdk/v65/common"
	accessgovernancecpv1beta1 "github.com/oracle/oci-service-operator/api/accessgovernancecp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type governanceInstanceOCIClient interface {
	CreateGovernanceInstance(context.Context, accessgovernancecpsdk.CreateGovernanceInstanceRequest) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error)
	GetGovernanceInstance(context.Context, accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error)
	ListGovernanceInstances(context.Context, accessgovernancecpsdk.ListGovernanceInstancesRequest) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error)
	UpdateGovernanceInstance(context.Context, accessgovernancecpsdk.UpdateGovernanceInstanceRequest) (accessgovernancecpsdk.UpdateGovernanceInstanceResponse, error)
	DeleteGovernanceInstance(context.Context, accessgovernancecpsdk.DeleteGovernanceInstanceRequest) (accessgovernancecpsdk.DeleteGovernanceInstanceResponse, error)
}

type governanceInstanceIdentity struct {
	compartmentID    string
	displayName      string
	licenseType      string
	tenancyNamespace string
}

func init() {
	registerGovernanceInstanceRuntimeHooksMutator(func(manager *GovernanceInstanceServiceManager, hooks *GovernanceInstanceRuntimeHooks) {
		client, initErr := newGovernanceInstanceSDKClient(manager)
		applyGovernanceInstanceRuntimeHooks(hooks, client, initErr)
	})
}

func newGovernanceInstanceSDKClient(manager *GovernanceInstanceServiceManager) (governanceInstanceOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("GovernanceInstance service manager is nil")
	}

	client, err := accessgovernancecpsdk.NewAccessGovernanceCPClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyGovernanceInstanceRuntimeHooks(
	hooks *GovernanceInstanceRuntimeHooks,
	client governanceInstanceOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedGovernanceInstanceRuntimeSemantics()
	hooks.ParityHooks.NormalizeDesiredState = normalizeGovernanceInstanceDesiredState
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *accessgovernancecpv1beta1.GovernanceInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildGovernanceInstanceUpdateBody(resource, currentResponse)
	}
	hooks.Identity.Resolve = func(resource *accessgovernancecpv1beta1.GovernanceInstance) (any, error) {
		return resolveGovernanceInstanceIdentity(resource)
	}
	hooks.Identity.LookupExisting = func(
		ctx context.Context,
		_ *accessgovernancecpv1beta1.GovernanceInstance,
		identity any,
	) (any, error) {
		governanceIdentity, ok := identity.(governanceInstanceIdentity)
		if !ok {
			return nil, fmt.Errorf("unexpected GovernanceInstance identity type %T", identity)
		}
		return lookupExistingGovernanceInstance(ctx, client, initErr, governanceIdentity)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardGovernanceInstanceExistingBeforeCreate
	hooks.Create.Fields = governanceInstanceCreateFields()
	hooks.Get.Fields = governanceInstanceGetFields()
	hooks.List.Fields = governanceInstanceListFields()
	hooks.Update.Fields = governanceInstanceUpdateFields()
	hooks.Delete.Fields = governanceInstanceDeleteFields()
}

func newGovernanceInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client governanceInstanceOCIClient,
) GovernanceInstanceServiceClient {
	return defaultGovernanceInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*accessgovernancecpv1beta1.GovernanceInstance](
			newGovernanceInstanceRuntimeConfig(log, client),
		),
	}
}

func newGovernanceInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client governanceInstanceOCIClient,
) generatedruntime.Config[*accessgovernancecpv1beta1.GovernanceInstance] {
	hooks := newGovernanceInstanceRuntimeHooksWithOCIClient(client)
	applyGovernanceInstanceRuntimeHooks(&hooks, client, nil)
	return buildGovernanceInstanceGeneratedRuntimeConfig(&GovernanceInstanceServiceManager{Log: log}, hooks)
}

func newGovernanceInstanceRuntimeHooksWithOCIClient(client governanceInstanceOCIClient) GovernanceInstanceRuntimeHooks {
	return GovernanceInstanceRuntimeHooks{
		Semantics: reviewedGovernanceInstanceRuntimeSemantics(),
		Create: runtimeOperationHooks[accessgovernancecpsdk.CreateGovernanceInstanceRequest, accessgovernancecpsdk.CreateGovernanceInstanceResponse]{
			Fields: governanceInstanceCreateFields(),
			Call: func(ctx context.Context, request accessgovernancecpsdk.CreateGovernanceInstanceRequest) (accessgovernancecpsdk.CreateGovernanceInstanceResponse, error) {
				return client.CreateGovernanceInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[accessgovernancecpsdk.GetGovernanceInstanceRequest, accessgovernancecpsdk.GetGovernanceInstanceResponse]{
			Fields: governanceInstanceGetFields(),
			Call: func(ctx context.Context, request accessgovernancecpsdk.GetGovernanceInstanceRequest) (accessgovernancecpsdk.GetGovernanceInstanceResponse, error) {
				return client.GetGovernanceInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[accessgovernancecpsdk.ListGovernanceInstancesRequest, accessgovernancecpsdk.ListGovernanceInstancesResponse]{
			Fields: governanceInstanceListFields(),
			Call: func(ctx context.Context, request accessgovernancecpsdk.ListGovernanceInstancesRequest) (accessgovernancecpsdk.ListGovernanceInstancesResponse, error) {
				return client.ListGovernanceInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[accessgovernancecpsdk.UpdateGovernanceInstanceRequest, accessgovernancecpsdk.UpdateGovernanceInstanceResponse]{
			Fields: governanceInstanceUpdateFields(),
			Call: func(ctx context.Context, request accessgovernancecpsdk.UpdateGovernanceInstanceRequest) (accessgovernancecpsdk.UpdateGovernanceInstanceResponse, error) {
				return client.UpdateGovernanceInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[accessgovernancecpsdk.DeleteGovernanceInstanceRequest, accessgovernancecpsdk.DeleteGovernanceInstanceResponse]{
			Fields: governanceInstanceDeleteFields(),
			Call: func(ctx context.Context, request accessgovernancecpsdk.DeleteGovernanceInstanceRequest) (accessgovernancecpsdk.DeleteGovernanceInstanceResponse, error) {
				return client.DeleteGovernanceInstance(ctx, request)
			},
		},
	}
}

func reviewedGovernanceInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "accessgovernancecp",
		FormalSlug:        "governanceinstance",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(accessgovernancecpsdk.InstanceLifecycleStateCreating)},
			UpdatingStates:     []string{},
			ActiveStates:       []string{string(accessgovernancecpsdk.InstanceLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(accessgovernancecpsdk.InstanceLifecycleStateDeleting)},
			TerminalStates: []string{string(accessgovernancecpsdk.InstanceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "licenseType", "tenancyNamespace", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"definedTags", "description", "displayName", "freeformTags", "licenseType"},
			ForceNew:      []string{"compartmentId", "systemTags", "tenancyNamespace"},
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

func governanceInstanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateGovernanceInstanceDetails", RequestName: "CreateGovernanceInstanceDetails", Contribution: "body"},
	}
}

func governanceInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "GovernanceInstanceId", RequestName: "governanceInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func governanceInstanceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func governanceInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "GovernanceInstanceId", RequestName: "governanceInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateGovernanceInstanceDetails", RequestName: "UpdateGovernanceInstanceDetails", Contribution: "body"},
	}
}

func governanceInstanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "GovernanceInstanceId", RequestName: "governanceInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func normalizeGovernanceInstanceDesiredState(
	resource *accessgovernancecpv1beta1.GovernanceInstance,
	currentResponse any,
) {
	if resource == nil || currentResponse == nil {
		return
	}

	// OCI does not echo this create-time credential input back on GovernanceInstance.
	resource.Spec.IdcsAccessToken = ""
}

func resolveGovernanceInstanceIdentity(
	resource *accessgovernancecpv1beta1.GovernanceInstance,
) (governanceInstanceIdentity, error) {
	if resource == nil {
		return governanceInstanceIdentity{}, fmt.Errorf("GovernanceInstance resource is nil")
	}

	return governanceInstanceIdentity{
		compartmentID:    firstNonEmptyTrim(resource.Spec.CompartmentId, resource.Status.CompartmentId),
		displayName:      firstNonEmptyTrim(resource.Spec.DisplayName, resource.Status.DisplayName),
		licenseType:      firstNonEmptyTrim(resource.Spec.LicenseType, resource.Status.LicenseType),
		tenancyNamespace: firstNonEmptyTrim(resource.Spec.TenancyNamespace, resource.Status.TenancyNamespace),
	}, nil
}

func guardGovernanceInstanceExistingBeforeCreate(
	_ context.Context,
	resource *accessgovernancecpv1beta1.GovernanceInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("GovernanceInstance resource is nil")
	}

	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" ||
		strings.TrimSpace(resource.Spec.LicenseType) == "" ||
		strings.TrimSpace(resource.Spec.TenancyNamespace) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}

	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupExistingGovernanceInstance(
	ctx context.Context,
	client governanceInstanceOCIClient,
	initErr error,
	identity governanceInstanceIdentity,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("GovernanceInstance OCI client is nil")
	}

	response, err := client.ListGovernanceInstances(ctx, accessgovernancecpsdk.ListGovernanceInstancesRequest{
		CompartmentId: common.String(identity.compartmentID),
		DisplayName:   common.String(identity.displayName),
	})
	if err != nil {
		return nil, err
	}

	matches := make([]accessgovernancecpsdk.GetGovernanceInstanceResponse, 0, len(response.Items))
	for _, item := range response.Items {
		if !governanceInstanceSummaryMatchesIdentity(item, identity) {
			continue
		}

		resourceID := stringValue(item.Id)
		if resourceID == "" {
			continue
		}

		candidate, err := client.GetGovernanceInstance(ctx, accessgovernancecpsdk.GetGovernanceInstanceRequest{
			GovernanceInstanceId: common.String(resourceID),
		})
		if err != nil {
			return nil, err
		}
		if !governanceInstanceMatchesIdentity(candidate.GovernanceInstance, identity) {
			continue
		}
		if !governanceInstanceLifecycleReusable(candidate.GovernanceInstance.LifecycleState) {
			continue
		}

		matches = append(matches, candidate)
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("GovernanceInstance lookup returned multiple exact matches for compartmentId=%q displayName=%q licenseType=%q tenancyNamespace=%q", identity.compartmentID, identity.displayName, identity.licenseType, identity.tenancyNamespace)
	}
}

func governanceInstanceSummaryMatchesIdentity(
	current accessgovernancecpsdk.GovernanceInstanceSummary,
	identity governanceInstanceIdentity,
) bool {
	if !governanceInstanceLifecycleReusable(current.LifecycleState) {
		return false
	}

	return stringValue(current.CompartmentId) == identity.compartmentID &&
		stringValue(current.DisplayName) == identity.displayName &&
		string(current.LicenseType) == identity.licenseType
}

func governanceInstanceMatchesIdentity(
	current accessgovernancecpsdk.GovernanceInstance,
	identity governanceInstanceIdentity,
) bool {
	return stringValue(current.CompartmentId) == identity.compartmentID &&
		stringValue(current.DisplayName) == identity.displayName &&
		string(current.LicenseType) == identity.licenseType &&
		stringValue(current.TenancyNamespace) == identity.tenancyNamespace
}

func governanceInstanceLifecycleReusable(state accessgovernancecpsdk.InstanceLifecycleStateEnum) bool {
	switch state {
	case accessgovernancecpsdk.InstanceLifecycleStateActive, accessgovernancecpsdk.InstanceLifecycleStateCreating:
		return true
	default:
		return false
	}
}

func buildGovernanceInstanceUpdateBody(
	resource *accessgovernancecpv1beta1.GovernanceInstance,
	currentResponse any,
) (accessgovernancecpsdk.UpdateGovernanceInstanceDetails, bool, error) {
	if resource == nil {
		return accessgovernancecpsdk.UpdateGovernanceInstanceDetails{}, false, fmt.Errorf("GovernanceInstance resource is nil")
	}

	current, err := governanceInstanceFromResponse(currentResponse)
	if err != nil {
		return accessgovernancecpsdk.UpdateGovernanceInstanceDetails{}, false, err
	}

	details := accessgovernancecpsdk.UpdateGovernanceInstanceDetails{}
	updateNeeded := false

	if desired, ok := governanceInstanceDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := governanceInstanceDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := governanceInstanceDesiredLicenseTypeUpdate(resource.Spec.LicenseType, current.LicenseType); ok {
		details.LicenseType = desired
		updateNeeded = true
	}
	if desired, ok := governanceInstanceDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := governanceInstanceDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func governanceInstanceFromResponse(currentResponse any) (accessgovernancecpsdk.GovernanceInstance, error) {
	switch current := currentResponse.(type) {
	case accessgovernancecpsdk.GovernanceInstance:
		return current, nil
	case *accessgovernancecpsdk.GovernanceInstance:
		if current == nil {
			return accessgovernancecpsdk.GovernanceInstance{}, fmt.Errorf("current GovernanceInstance response is nil")
		}
		return *current, nil
	case accessgovernancecpsdk.GovernanceInstanceSummary:
		return accessgovernancecpsdk.GovernanceInstance{
			Id:             current.Id,
			DisplayName:    current.DisplayName,
			CompartmentId:  current.CompartmentId,
			TimeCreated:    current.TimeCreated,
			TimeUpdated:    current.TimeUpdated,
			LifecycleState: current.LifecycleState,
			Description:    current.Description,
			LicenseType:    current.LicenseType,
			InstanceUrl:    current.InstanceUrl,
			DefinedTags:    current.DefinedTags,
			FreeformTags:   current.FreeformTags,
			SystemTags:     current.SystemTags,
		}, nil
	case *accessgovernancecpsdk.GovernanceInstanceSummary:
		if current == nil {
			return accessgovernancecpsdk.GovernanceInstance{}, fmt.Errorf("current GovernanceInstance response is nil")
		}
		return governanceInstanceFromResponse(*current)
	case accessgovernancecpsdk.CreateGovernanceInstanceResponse:
		return current.GovernanceInstance, nil
	case *accessgovernancecpsdk.CreateGovernanceInstanceResponse:
		if current == nil {
			return accessgovernancecpsdk.GovernanceInstance{}, fmt.Errorf("current GovernanceInstance response is nil")
		}
		return current.GovernanceInstance, nil
	case accessgovernancecpsdk.GetGovernanceInstanceResponse:
		return current.GovernanceInstance, nil
	case *accessgovernancecpsdk.GetGovernanceInstanceResponse:
		if current == nil {
			return accessgovernancecpsdk.GovernanceInstance{}, fmt.Errorf("current GovernanceInstance response is nil")
		}
		return current.GovernanceInstance, nil
	case accessgovernancecpsdk.UpdateGovernanceInstanceResponse:
		return current.GovernanceInstance, nil
	case *accessgovernancecpsdk.UpdateGovernanceInstanceResponse:
		if current == nil {
			return accessgovernancecpsdk.GovernanceInstance{}, fmt.Errorf("current GovernanceInstance response is nil")
		}
		return current.GovernanceInstance, nil
	default:
		return accessgovernancecpsdk.GovernanceInstance{}, fmt.Errorf("unexpected current GovernanceInstance response type %T", currentResponse)
	}
}

func governanceInstanceDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func governanceInstanceDesiredLicenseTypeUpdate(
	spec string,
	current accessgovernancecpsdk.LicenseTypeEnum,
) (accessgovernancecpsdk.LicenseTypeEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return accessgovernancecpsdk.LicenseTypeEnum(spec), true
}

func governanceInstanceDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func governanceInstanceDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := governanceInstanceDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if governanceInstanceJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func governanceInstanceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func governanceInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
