/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const configKind = "Config"

type configIdentity struct {
	compartmentID string
	configType    string
	resourceType  string
}

type configAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type configDeletePreflightClient struct {
	delegate ConfigServiceClient
	get      func(context.Context, stackmonitoringsdk.GetConfigRequest) (stackmonitoringsdk.GetConfigResponse, error)
	list     func(context.Context, stackmonitoringsdk.ListConfigsRequest) (stackmonitoringsdk.ListConfigsResponse, error)
}

type configResourceContextKey struct{}

type configResourceContextClient struct {
	delegate ConfigServiceClient
}

func (e configAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e configAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerConfigRuntimeHooksMutator(func(_ *ConfigServiceManager, hooks *ConfigRuntimeHooks) {
		applyConfigRuntimeHooks(hooks)
	})
}

func applyConfigRuntimeHooks(hooks *ConfigRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newConfigRuntimeSemantics()
	hooks.BuildCreateBody = buildConfigCreateBody
	hooks.BuildUpdateBody = buildConfigUpdateBody
	hooks.Identity.Resolve = func(resource *stackmonitoringv1beta1.Config) (any, error) {
		return resolveConfigIdentity(resource)
	}
	hooks.Create.Fields = configCreateFields()
	hooks.Update.Fields = configUpdateFields()
	wrapConfigPolymorphicCalls(hooks)
	hooks.List.Fields = configListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listConfigsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleConfigDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapConfigResourceContextClient)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ConfigServiceClient) ConfigServiceClient {
		return configDeletePreflightClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newConfigRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "config",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(stackmonitoringsdk.ConfigLifecycleStateCreating)},
			UpdatingStates:     []string{string(stackmonitoringsdk.ConfigLifecycleStateUpdating)},
			ActiveStates:       []string{string(stackmonitoringsdk.ConfigLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(stackmonitoringsdk.ConfigLifecycleStateDeleting)},
			TerminalStates: []string{string(stackmonitoringsdk.ConfigLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "configType", "resourceType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"freeformTags",
				"definedTags",
				"isEnabled",
				"license",
				"isManuallyOnboarded",
				"version",
				"policyNames",
				"dynamicGroups",
				"userGroups",
				"additionalConfigurations",
			},
			ForceNew: []string{
				"compartmentId",
				"configType",
				"resourceType",
				"jsonData",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: configKind, Action: "CreateConfig"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: configKind, Action: "UpdateConfig"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: configKind, Action: "DeleteConfig"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: configKind, Action: "GetConfig"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: configKind, Action: "GetConfig"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: configKind, Action: "GetConfig"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func configCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpcRequestId", RequestName: "opc-request-id", Contribution: "header"},
	}
}

func configUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ConfigId", RequestName: "configId", Contribution: "path", PreferResourceID: true},
		{FieldName: "OpcRequestId", RequestName: "opc-request-id", Contribution: "header"},
	}
}

func configListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "Type",
			RequestName:  "type",
			Contribution: "query",
			LookupPaths:  []string{"status.configType", "spec.configType", "configType"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func wrapConfigPolymorphicCalls(hooks *ConfigRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.Create.Call = wrapConfigCreateCall(hooks.Create.Call)
	hooks.Update.Call = wrapConfigUpdateCall(hooks.Update.Call)
}

func wrapConfigCreateCall(
	createCall func(context.Context, stackmonitoringsdk.CreateConfigRequest) (stackmonitoringsdk.CreateConfigResponse, error),
) func(context.Context, stackmonitoringsdk.CreateConfigRequest) (stackmonitoringsdk.CreateConfigResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.CreateConfigRequest) (stackmonitoringsdk.CreateConfigResponse, error) {
		if createCall == nil {
			return stackmonitoringsdk.CreateConfigResponse{}, fmt.Errorf("config create call is not configured")
		}
		details, err := configCreateDetailsFromContext(ctx)
		if err != nil {
			return stackmonitoringsdk.CreateConfigResponse{}, err
		}
		request.CreateConfigDetails = details
		return createCall(ctx, request)
	}
}

func wrapConfigUpdateCall(
	updateCall func(context.Context, stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error),
) func(context.Context, stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.UpdateConfigRequest) (stackmonitoringsdk.UpdateConfigResponse, error) {
		if updateCall == nil {
			return stackmonitoringsdk.UpdateConfigResponse{}, fmt.Errorf("config update call is not configured")
		}
		details, err := configUpdateDetailsFromContext(ctx)
		if err != nil {
			return stackmonitoringsdk.UpdateConfigResponse{}, err
		}
		request.UpdateConfigDetails = details
		return updateCall(ctx, request)
	}
}

func configCreateDetailsFromContext(ctx context.Context) (stackmonitoringsdk.CreateConfigDetails, error) {
	resource, err := configResourceFromContext(ctx)
	if err != nil {
		return nil, err
	}
	body, err := buildConfigCreateBody(ctx, resource, "")
	if err != nil {
		return nil, err
	}
	details, ok := body.(stackmonitoringsdk.CreateConfigDetails)
	if !ok {
		return nil, fmt.Errorf("resolved Config create body %T does not implement stackmonitoring.CreateConfigDetails", body)
	}
	return details, nil
}

func configUpdateDetailsFromContext(ctx context.Context) (stackmonitoringsdk.UpdateConfigDetails, error) {
	resource, err := configResourceFromContext(ctx)
	if err != nil {
		return nil, err
	}
	body, _, err := buildConfigUpdateDetails(resource.Spec)
	if err != nil {
		return nil, err
	}
	details, ok := body.(stackmonitoringsdk.UpdateConfigDetails)
	if !ok {
		return nil, fmt.Errorf("resolved Config update body %T does not implement stackmonitoring.UpdateConfigDetails", body)
	}
	return details, nil
}

func wrapConfigResourceContextClient(delegate ConfigServiceClient) ConfigServiceClient {
	return configResourceContextClient{delegate: delegate}
}

func (c configResourceContextClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.Config,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("config runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(context.WithValue(ctx, configResourceContextKey{}, resource), resource, req)
}

func (c configResourceContextClient) Delete(ctx context.Context, resource *stackmonitoringv1beta1.Config) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("config runtime client is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func configResourceFromContext(ctx context.Context) (*stackmonitoringv1beta1.Config, error) {
	resource, ok := ctx.Value(configResourceContextKey{}).(*stackmonitoringv1beta1.Config)
	if !ok || resource == nil {
		return nil, fmt.Errorf("config resource is not available in request context")
	}
	return resource, nil
}

func resolveConfigIdentity(resource *stackmonitoringv1beta1.Config) (configIdentity, error) {
	if resource == nil {
		return configIdentity{}, fmt.Errorf("config resource is nil")
	}
	configType, err := normalizedConfigType(resource.Spec.ConfigType)
	if err != nil {
		return configIdentity{}, err
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return configIdentity{}, fmt.Errorf("%s spec is missing required field: compartmentId", configKind)
	}
	resourceType, err := normalizedAutoPromoteResourceType(resource.Spec.ResourceType)
	if err != nil && configType == stackmonitoringsdk.ConfigConfigTypeAutoPromote {
		return configIdentity{}, err
	}
	if configType != stackmonitoringsdk.ConfigConfigTypeAutoPromote {
		resourceType = ""
	}
	return configIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		configType:    string(configType),
		resourceType:  string(resourceType),
	}, nil
}

func buildConfigCreateBody(_ context.Context, resource *stackmonitoringv1beta1.Config, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("config resource is nil")
	}
	if err := validateConfigSpec(resource.Spec); err != nil {
		return nil, err
	}
	configType, err := normalizedConfigType(resource.Spec.ConfigType)
	if err != nil {
		return nil, err
	}

	commonFields := configCreateCommonFields(resource.Spec)
	return buildConfigCreateDetails(resource.Spec, configType, commonFields)
}

func buildConfigCreateDetails(
	spec stackmonitoringv1beta1.ConfigSpec,
	configType stackmonitoringsdk.ConfigConfigTypeEnum,
	commonFields configCreateCommon,
) (any, error) {
	switch configType {
	case stackmonitoringsdk.ConfigConfigTypeAutoPromote:
		resourceType, err := normalizedAutoPromoteResourceType(spec.ResourceType)
		if err != nil {
			return nil, err
		}
		return stackmonitoringsdk.CreateAutoPromoteConfigDetails{
			CompartmentId: commonFields.compartmentID,
			DisplayName:   commonFields.displayName,
			FreeformTags:  commonFields.freeformTags,
			DefinedTags:   commonFields.definedTags,
			IsEnabled:     common.Bool(spec.IsEnabled),
			ResourceType:  resourceType,
		}, nil
	case stackmonitoringsdk.ConfigConfigTypeComputeAutoActivatePlugin:
		return stackmonitoringsdk.CreateComputeAutoActivatePluginConfigDetails{
			CompartmentId: commonFields.compartmentID,
			DisplayName:   commonFields.displayName,
			FreeformTags:  commonFields.freeformTags,
			DefinedTags:   commonFields.definedTags,
			IsEnabled:     common.Bool(spec.IsEnabled),
		}, nil
	case stackmonitoringsdk.ConfigConfigTypeLicenseAutoAssign:
		license, err := normalizedConfigLicense(spec.License)
		if err != nil {
			return nil, err
		}
		return stackmonitoringsdk.CreateLicenseAutoAssignConfigDetails{
			CompartmentId: commonFields.compartmentID,
			DisplayName:   commonFields.displayName,
			FreeformTags:  commonFields.freeformTags,
			DefinedTags:   commonFields.definedTags,
			License:       license,
		}, nil
	case stackmonitoringsdk.ConfigConfigTypeLicenseEnterpriseExtensibility:
		return stackmonitoringsdk.CreateLicenseEnterpriseExtensibilityConfigDetails{
			CompartmentId: commonFields.compartmentID,
			DisplayName:   commonFields.displayName,
			FreeformTags:  commonFields.freeformTags,
			DefinedTags:   commonFields.definedTags,
			IsEnabled:     common.Bool(spec.IsEnabled),
		}, nil
	case stackmonitoringsdk.ConfigConfigTypeOnboard:
		body, err := buildOnboardCreateBody(spec, commonFields)
		if err != nil {
			return nil, err
		}
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported Config configType %q", spec.ConfigType)
	}
}

func buildConfigUpdateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.Config,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("config resource is nil")
	}
	if err := validateConfigSpec(resource.Spec); err != nil {
		return nil, false, err
	}

	current, err := configResponseMap(currentResponse)
	if err != nil {
		return nil, false, err
	}
	configType, err := normalizedConfigType(resource.Spec.ConfigType)
	if err != nil {
		return nil, false, err
	}
	desiredType := string(configType)
	if currentType := configString(current, "configType"); currentType != "" && !strings.EqualFold(currentType, desiredType) {
		return nil, false, fmt.Errorf("%s formal semantics require replacement when configType changes", configKind)
	}

	body, desired, err := buildConfigUpdateDetails(resource.Spec)
	if err != nil {
		return nil, false, err
	}
	return body, configUpdateNeeded(desired, current), nil
}

func buildConfigUpdateDetails(spec stackmonitoringv1beta1.ConfigSpec) (any, map[string]any, error) {
	configType, err := normalizedConfigType(spec.ConfigType)
	if err != nil {
		return nil, nil, err
	}
	switch configType {
	case stackmonitoringsdk.ConfigConfigTypeAutoPromote:
		body, desired := buildAutoPromoteUpdateBody(spec)
		return body, desired, nil
	case stackmonitoringsdk.ConfigConfigTypeComputeAutoActivatePlugin:
		body, desired := buildComputeAutoActivatePluginUpdateBody(spec)
		return body, desired, nil
	case stackmonitoringsdk.ConfigConfigTypeLicenseAutoAssign:
		body, desired, err := buildLicenseAutoAssignUpdateBody(spec)
		if err != nil {
			return nil, nil, err
		}
		return body, desired, nil
	case stackmonitoringsdk.ConfigConfigTypeLicenseEnterpriseExtensibility:
		body, desired := buildLicenseEnterpriseExtensibilityUpdateBody(spec)
		return body, desired, nil
	case stackmonitoringsdk.ConfigConfigTypeOnboard:
		body, desired, err := buildOnboardUpdateBody(spec)
		if err != nil {
			return nil, nil, err
		}
		return body, desired, nil
	default:
		return nil, nil, fmt.Errorf("unsupported Config configType %q", spec.ConfigType)
	}
}

type configCreateCommon struct {
	compartmentID *string
	displayName   *string
	freeformTags  map[string]string
	definedTags   map[string]map[string]interface{}
}

func configCreateCommonFields(spec stackmonitoringv1beta1.ConfigSpec) configCreateCommon {
	return configCreateCommon{
		compartmentID: common.String(strings.TrimSpace(spec.CompartmentId)),
		displayName:   configOptionalString(spec.DisplayName),
		freeformTags:  cloneConfigStringMap(spec.FreeformTags),
		definedTags:   configDefinedTags(spec.DefinedTags),
	}
}

func buildOnboardCreateBody(
	spec stackmonitoringv1beta1.ConfigSpec,
	commonFields configCreateCommon,
) (stackmonitoringsdk.CreateOnboardConfigDetails, error) {
	dynamicGroups, err := configDynamicGroups(spec.DynamicGroups)
	if err != nil {
		return stackmonitoringsdk.CreateOnboardConfigDetails{}, err
	}
	userGroups, err := configUserGroups(spec.UserGroups)
	if err != nil {
		return stackmonitoringsdk.CreateOnboardConfigDetails{}, err
	}
	return stackmonitoringsdk.CreateOnboardConfigDetails{
		CompartmentId:            commonFields.compartmentID,
		DisplayName:              commonFields.displayName,
		FreeformTags:             commonFields.freeformTags,
		DefinedTags:              commonFields.definedTags,
		IsManuallyOnboarded:      common.Bool(spec.IsManuallyOnboarded),
		Version:                  configOptionalString(spec.Version),
		PolicyNames:              cloneConfigStringSlice(spec.PolicyNames),
		DynamicGroups:            dynamicGroups,
		UserGroups:               userGroups,
		AdditionalConfigurations: configAdditionalConfigurations(spec.AdditionalConfigurations),
	}, nil
}

func buildAutoPromoteUpdateBody(spec stackmonitoringv1beta1.ConfigSpec) (stackmonitoringsdk.UpdateAutoPromoteConfigDetails, map[string]any) {
	body := stackmonitoringsdk.UpdateAutoPromoteConfigDetails{
		IsEnabled: common.Bool(spec.IsEnabled),
	}
	desired := map[string]any{"isEnabled": spec.IsEnabled}
	applyConfigCommonUpdateFields(spec, &body.DisplayName, &body.FreeformTags, &body.DefinedTags, desired)
	return body, desired
}

func buildComputeAutoActivatePluginUpdateBody(
	spec stackmonitoringv1beta1.ConfigSpec,
) (stackmonitoringsdk.UpdateComputeAutoActivatePluginConfigDetails, map[string]any) {
	body := stackmonitoringsdk.UpdateComputeAutoActivatePluginConfigDetails{
		IsEnabled: common.Bool(spec.IsEnabled),
	}
	desired := map[string]any{"isEnabled": spec.IsEnabled}
	applyConfigCommonUpdateFields(spec, &body.DisplayName, &body.FreeformTags, &body.DefinedTags, desired)
	return body, desired
}

func buildLicenseAutoAssignUpdateBody(
	spec stackmonitoringv1beta1.ConfigSpec,
) (stackmonitoringsdk.UpdateLicenseAutoAssignConfigDetails, map[string]any, error) {
	body := stackmonitoringsdk.UpdateLicenseAutoAssignConfigDetails{}
	desired := map[string]any{}
	if strings.TrimSpace(spec.License) != "" {
		license, err := normalizedConfigLicense(spec.License)
		if err != nil {
			return stackmonitoringsdk.UpdateLicenseAutoAssignConfigDetails{}, nil, err
		}
		body.License = license
		desired["license"] = string(license)
	}
	applyConfigCommonUpdateFields(spec, &body.DisplayName, &body.FreeformTags, &body.DefinedTags, desired)
	return body, desired, nil
}

func buildLicenseEnterpriseExtensibilityUpdateBody(
	spec stackmonitoringv1beta1.ConfigSpec,
) (stackmonitoringsdk.UpdateLicenseEnterpriseExtensibilityConfigDetails, map[string]any) {
	body := stackmonitoringsdk.UpdateLicenseEnterpriseExtensibilityConfigDetails{
		IsEnabled: common.Bool(spec.IsEnabled),
	}
	desired := map[string]any{"isEnabled": spec.IsEnabled}
	applyConfigCommonUpdateFields(spec, &body.DisplayName, &body.FreeformTags, &body.DefinedTags, desired)
	return body, desired
}

func buildOnboardUpdateBody(
	spec stackmonitoringv1beta1.ConfigSpec,
) (stackmonitoringsdk.UpdateOnboardConfigDetails, map[string]any, error) {
	dynamicGroups, err := configDynamicGroups(spec.DynamicGroups)
	if err != nil {
		return stackmonitoringsdk.UpdateOnboardConfigDetails{}, nil, err
	}
	userGroups, err := configUserGroups(spec.UserGroups)
	if err != nil {
		return stackmonitoringsdk.UpdateOnboardConfigDetails{}, nil, err
	}

	body := stackmonitoringsdk.UpdateOnboardConfigDetails{
		IsManuallyOnboarded: common.Bool(spec.IsManuallyOnboarded),
	}
	desired := map[string]any{"isManuallyOnboarded": spec.IsManuallyOnboarded}
	applyConfigCommonUpdateFields(spec, &body.DisplayName, &body.FreeformTags, &body.DefinedTags, desired)
	if strings.TrimSpace(spec.Version) != "" {
		body.Version = common.String(spec.Version)
		desired["version"] = spec.Version
	}
	if spec.PolicyNames != nil {
		body.PolicyNames = cloneConfigStringSlice(spec.PolicyNames)
		desired["policyNames"] = body.PolicyNames
	}
	if spec.DynamicGroups != nil {
		body.DynamicGroups = dynamicGroups
		desired["dynamicGroups"] = dynamicGroups
	}
	if spec.UserGroups != nil {
		body.UserGroups = userGroups
		desired["userGroups"] = userGroups
	}
	if spec.AdditionalConfigurations.PropertiesMap != nil {
		body.AdditionalConfigurations = configAdditionalConfigurations(spec.AdditionalConfigurations)
		desired["additionalConfigurations"] = body.AdditionalConfigurations
	}
	return body, desired, nil
}

func applyConfigCommonUpdateFields(
	spec stackmonitoringv1beta1.ConfigSpec,
	displayName **string,
	freeformTags *map[string]string,
	definedTags *map[string]map[string]interface{},
	desired map[string]any,
) {
	if strings.TrimSpace(spec.DisplayName) != "" {
		*displayName = common.String(spec.DisplayName)
		desired["displayName"] = spec.DisplayName
	}
	if spec.FreeformTags != nil {
		*freeformTags = cloneConfigStringMap(spec.FreeformTags)
		desired["freeformTags"] = *freeformTags
	}
	if spec.DefinedTags != nil {
		*definedTags = configDefinedTags(spec.DefinedTags)
		desired["definedTags"] = *definedTags
	}
}

func validateConfigSpec(spec stackmonitoringv1beta1.ConfigSpec) error {
	configType, err := normalizedConfigType(spec.ConfigType)
	if err != nil {
		return err
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("%s spec is missing required field: compartmentId", configKind)
	}
	if strings.TrimSpace(spec.JsonData) != "" {
		return fmt.Errorf("%s spec.jsonData is not supported; use the typed Config spec fields", configKind)
	}
	if err := validateConfigTypeSpecificFields(spec, configType); err != nil {
		return err
	}
	return validateConfigIncompatibleFields(spec, configType)
}

func validateConfigTypeSpecificFields(spec stackmonitoringv1beta1.ConfigSpec, configType stackmonitoringsdk.ConfigConfigTypeEnum) error {
	switch configType {
	case stackmonitoringsdk.ConfigConfigTypeAutoPromote:
		if _, err := normalizedAutoPromoteResourceType(spec.ResourceType); err != nil {
			return err
		}
	case stackmonitoringsdk.ConfigConfigTypeLicenseAutoAssign:
		if _, err := normalizedConfigLicense(spec.License); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigIncompatibleFields(
	spec stackmonitoringv1beta1.ConfigSpec,
	configType stackmonitoringsdk.ConfigConfigTypeEnum,
) error {
	fields := configIncompatibleFields(spec, configType)
	if len(fields) == 0 {
		return nil
	}
	return fmt.Errorf("%s configType %s does not support spec field(s): %s", configKind, configType, strings.Join(fields, ", "))
}

func configIncompatibleFields(
	spec stackmonitoringv1beta1.ConfigSpec,
	configType stackmonitoringsdk.ConfigConfigTypeEnum,
) []string {
	switch configType {
	case stackmonitoringsdk.ConfigConfigTypeAutoPromote:
		return configIncompatibleCommonNonAutoPromoteFields(spec)
	case stackmonitoringsdk.ConfigConfigTypeComputeAutoActivatePlugin,
		stackmonitoringsdk.ConfigConfigTypeLicenseEnterpriseExtensibility:
		return configIncompatibleNonAutoPromoteFields(spec)
	case stackmonitoringsdk.ConfigConfigTypeLicenseAutoAssign:
		return configIncompatibleLicenseAutoAssignFields(spec)
	case stackmonitoringsdk.ConfigConfigTypeOnboard:
		return configIncompatibleOnboardConfigFields(spec)
	}
	return nil
}

func configIncompatibleNonAutoPromoteFields(spec stackmonitoringv1beta1.ConfigSpec) []string {
	fields := configIncompatibleCommonNonAutoPromoteFields(spec)
	return appendConfigFieldIfSet(fields, "resourceType", strings.TrimSpace(spec.ResourceType) != "")
}

func configIncompatibleLicenseAutoAssignFields(spec stackmonitoringv1beta1.ConfigSpec) []string {
	var fields []string
	fields = appendConfigFieldIfSet(fields, "resourceType", strings.TrimSpace(spec.ResourceType) != "")
	fields = appendConfigFieldIfSet(fields, "isEnabled", spec.IsEnabled)
	return append(fields, configIncompatibleOnboardFields(spec)...)
}

func configIncompatibleOnboardConfigFields(spec stackmonitoringv1beta1.ConfigSpec) []string {
	var fields []string
	fields = appendConfigFieldIfSet(fields, "resourceType", strings.TrimSpace(spec.ResourceType) != "")
	fields = appendConfigFieldIfSet(fields, "license", strings.TrimSpace(spec.License) != "")
	return appendConfigFieldIfSet(fields, "isEnabled", spec.IsEnabled)
}

func configIncompatibleCommonNonAutoPromoteFields(spec stackmonitoringv1beta1.ConfigSpec) []string {
	fields := configIncompatibleOnboardFields(spec)
	if strings.TrimSpace(spec.License) != "" {
		fields = append(fields, "license")
	}
	return fields
}

func appendConfigFieldIfSet(fields []string, name string, set bool) []string {
	if set {
		return append(fields, name)
	}
	return fields
}

func configIncompatibleOnboardFields(spec stackmonitoringv1beta1.ConfigSpec) []string {
	var fields []string
	if spec.IsManuallyOnboarded {
		fields = append(fields, "isManuallyOnboarded")
	}
	if strings.TrimSpace(spec.Version) != "" {
		fields = append(fields, "version")
	}
	if spec.PolicyNames != nil {
		fields = append(fields, "policyNames")
	}
	if spec.DynamicGroups != nil {
		fields = append(fields, "dynamicGroups")
	}
	if spec.UserGroups != nil {
		fields = append(fields, "userGroups")
	}
	if spec.AdditionalConfigurations.PropertiesMap != nil {
		fields = append(fields, "additionalConfigurations")
	}
	return fields
}

func normalizedConfigType(raw string) (stackmonitoringsdk.ConfigConfigTypeEnum, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%s spec is missing required field: configType", configKind)
	}
	configType, ok := stackmonitoringsdk.GetMappingConfigConfigTypeEnum(value)
	if !ok {
		return "", fmt.Errorf("unsupported Config configType %q; supported values: %s", raw, strings.Join(stackmonitoringsdk.GetConfigConfigTypeEnumStringValues(), ", "))
	}
	return configType, nil
}

func normalizedAutoPromoteResourceType(raw string) (stackmonitoringsdk.CreateAutoPromoteConfigDetailsResourceTypeEnum, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%s AUTO_PROMOTE spec is missing required field: resourceType", configKind)
	}
	resourceType, ok := stackmonitoringsdk.GetMappingCreateAutoPromoteConfigDetailsResourceTypeEnum(value)
	if !ok {
		return "", fmt.Errorf("unsupported Config AUTO_PROMOTE resourceType %q; supported values: %s", raw, strings.Join(stackmonitoringsdk.GetCreateAutoPromoteConfigDetailsResourceTypeEnumStringValues(), ", "))
	}
	return resourceType, nil
}

func normalizedConfigLicense(raw string) (stackmonitoringsdk.LicenseTypeEnum, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%s LICENSE_AUTO_ASSIGN spec is missing required field: license", configKind)
	}
	license, ok := stackmonitoringsdk.GetMappingLicenseTypeEnum(value)
	if !ok {
		return "", fmt.Errorf("unsupported Config license %q; supported values: %s", raw, strings.Join(stackmonitoringsdk.GetLicenseTypeEnumStringValues(), ", "))
	}
	return license, nil
}

func configDynamicGroups(groups []stackmonitoringv1beta1.ConfigDynamicGroup) ([]stackmonitoringsdk.DynamicGroupDetails, error) {
	if groups == nil {
		return nil, nil
	}
	converted := make([]stackmonitoringsdk.DynamicGroupDetails, 0, len(groups))
	for index, group := range groups {
		if strings.TrimSpace(group.Name) == "" {
			return nil, fmt.Errorf("%s dynamicGroups[%d] is missing required field: name", configKind, index)
		}
		assignment, ok := stackmonitoringsdk.GetMappingDynamicGroupDetailsStackMonitoringAssignmentEnum(group.StackMonitoringAssignment)
		if !ok {
			return nil, fmt.Errorf("unsupported Config dynamicGroups[%d].stackMonitoringAssignment %q; supported values: %s", index, group.StackMonitoringAssignment, strings.Join(stackmonitoringsdk.GetDynamicGroupDetailsStackMonitoringAssignmentEnumStringValues(), ", "))
		}
		converted = append(converted, stackmonitoringsdk.DynamicGroupDetails{
			Name:                      common.String(group.Name),
			StackMonitoringAssignment: assignment,
			Domain:                    configOptionalString(group.Domain),
		})
	}
	return converted, nil
}

func configUserGroups(groups []stackmonitoringv1beta1.ConfigUserGroup) ([]stackmonitoringsdk.GroupDetails, error) {
	if groups == nil {
		return nil, nil
	}
	converted := make([]stackmonitoringsdk.GroupDetails, 0, len(groups))
	for index, group := range groups {
		if strings.TrimSpace(group.Name) == "" {
			return nil, fmt.Errorf("%s userGroups[%d] is missing required field: name", configKind, index)
		}
		if strings.TrimSpace(group.StackMonitoringRole) == "" {
			return nil, fmt.Errorf("%s userGroups[%d] is missing required field: stackMonitoringRole", configKind, index)
		}
		converted = append(converted, stackmonitoringsdk.GroupDetails{
			Name:                common.String(group.Name),
			StackMonitoringRole: common.String(group.StackMonitoringRole),
			Domain:              configOptionalString(group.Domain),
		})
	}
	return converted, nil
}

func configAdditionalConfigurations(
	value stackmonitoringv1beta1.ConfigAdditionalConfigurations,
) *stackmonitoringsdk.AdditionalConfigurationDetails {
	if value.PropertiesMap == nil {
		return nil
	}
	return &stackmonitoringsdk.AdditionalConfigurationDetails{
		PropertiesMap: cloneConfigStringMap(value.PropertiesMap),
	}
}

func listConfigsAllPages(
	call func(context.Context, stackmonitoringsdk.ListConfigsRequest) (stackmonitoringsdk.ListConfigsResponse, error),
) func(context.Context, stackmonitoringsdk.ListConfigsRequest) (stackmonitoringsdk.ListConfigsResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.ListConfigsRequest) (stackmonitoringsdk.ListConfigsResponse, error) {
		var combined stackmonitoringsdk.ListConfigsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.RawResponse = response.RawResponse
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func (c configDeletePreflightClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.Config,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("config runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c configDeletePreflightClient) Delete(ctx context.Context, resource *stackmonitoringv1beta1.Config) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("config runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c configDeletePreflightClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *stackmonitoringv1beta1.Config,
) error {
	if resource == nil {
		return nil
	}
	if currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c configDeletePreflightClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *stackmonitoringv1beta1.Config,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, stackmonitoringsdk.GetConfigRequest{ConfigId: common.String(currentID)})
	return configAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c configDeletePreflightClient) rejectAuthShapedList(
	ctx context.Context,
	resource *stackmonitoringv1beta1.Config,
) error {
	if c.list == nil {
		return nil
	}
	identity, err := resolveConfigIdentity(resource)
	if err != nil {
		return err
	}
	_, err = c.list(ctx, stackmonitoringsdk.ListConfigsRequest{
		CompartmentId: common.String(identity.compartmentID),
		Type:          stackmonitoringsdk.ConfigConfigTypeEnum(identity.configType),
	})
	return configAmbiguousDeleteError(resource, err, "pre-delete list")
}

func handleConfigDeleteError(resource *stackmonitoringv1beta1.Config, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := configAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func configAmbiguousDeleteError(resource *stackmonitoringv1beta1.Config, err error, operation string) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return configAmbiguousNotFoundError{
		message:      fmt.Sprintf("Config %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func configResponseMap(response any) (map[string]any, error) {
	body, ok := configResponseBody(response)
	if !ok || body == nil {
		return nil, fmt.Errorf("current Config response does not expose a Config body")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal current Config body: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode current Config body: %w", err)
	}
	return values, nil
}

func configResponseBody(response any) (any, bool) {
	switch concrete := response.(type) {
	case nil:
		return nil, false
	case stackmonitoringsdk.CreateConfigResponse:
		return configBody(concrete.Config)
	case *stackmonitoringsdk.CreateConfigResponse:
		return configBodyFromCreateResponse(concrete)
	case stackmonitoringsdk.GetConfigResponse:
		return configBody(concrete.Config)
	case *stackmonitoringsdk.GetConfigResponse:
		return configBodyFromGetResponse(concrete)
	case stackmonitoringsdk.UpdateConfigResponse:
		return configBody(concrete.Config)
	case *stackmonitoringsdk.UpdateConfigResponse:
		return configBodyFromUpdateResponse(concrete)
	case stackmonitoringsdk.Config:
		return configBody(concrete)
	default:
		return response, true
	}
}

func configBody(body stackmonitoringsdk.Config) (any, bool) {
	return body, body != nil
}

func configBodyFromCreateResponse(response *stackmonitoringsdk.CreateConfigResponse) (any, bool) {
	if response == nil {
		return nil, false
	}
	return configBody(response.Config)
}

func configBodyFromGetResponse(response *stackmonitoringsdk.GetConfigResponse) (any, bool) {
	if response == nil {
		return nil, false
	}
	return configBody(response.Config)
}

func configBodyFromUpdateResponse(response *stackmonitoringsdk.UpdateConfigResponse) (any, bool) {
	if response == nil {
		return nil, false
	}
	return configBody(response.Config)
}

func configUpdateNeeded(desired map[string]any, current map[string]any) bool {
	for key, desiredValue := range desired {
		currentValue, ok := configMapValue(current, key)
		if !ok || !reflect.DeepEqual(configComparableValue(desiredValue), configComparableValue(currentValue)) {
			return true
		}
	}
	return false
}

func configMapValue(values map[string]any, key string) (any, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}
	normalized := strings.ToLower(key)
	for candidate, value := range values {
		if strings.ToLower(candidate) == normalized {
			return value, true
		}
	}
	return nil, false
}

func configString(values map[string]any, key string) string {
	value, ok := configMapValue(values, key)
	if !ok || value == nil {
		return ""
	}
	if concrete, ok := value.(string); ok {
		return strings.TrimSpace(concrete)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func configComparableValue(value any) any {
	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var comparable any
	if err := json.Unmarshal(payload, &comparable); err != nil {
		return value
	}
	return comparable
}

func configOptionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func cloneConfigStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneConfigStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func configDefinedTags(values map[string]shared.MapValue) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(values))
	for namespace, entries := range values {
		converted[namespace] = make(map[string]interface{}, len(entries))
		for key, value := range entries {
			converted[namespace][key] = value
		}
	}
	return converted
}
