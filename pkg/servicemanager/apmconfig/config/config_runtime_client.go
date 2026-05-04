/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apmconfigsdk "github.com/oracle/oci-go-sdk/v65/apmconfig"
	apmconfigv1beta1 "github.com/oracle/oci-service-operator/api/apmconfig/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type configOCIClient interface {
	CreateConfig(context.Context, apmconfigsdk.CreateConfigRequest) (apmconfigsdk.CreateConfigResponse, error)
	GetConfig(context.Context, apmconfigsdk.GetConfigRequest) (apmconfigsdk.GetConfigResponse, error)
	ListConfigs(context.Context, apmconfigsdk.ListConfigsRequest) (apmconfigsdk.ListConfigsResponse, error)
	UpdateConfig(context.Context, apmconfigsdk.UpdateConfigRequest) (apmconfigsdk.UpdateConfigResponse, error)
	DeleteConfig(context.Context, apmconfigsdk.DeleteConfigRequest) (apmconfigsdk.DeleteConfigResponse, error)
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

	hooks.Semantics = reviewedConfigRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardConfigExistingBeforeCreate
	hooks.Create.Fields = configCreateFields()
	hooks.Get.Fields = configGetFields()
	hooks.List.Fields = configListFields()
	hooks.Update.Fields = configUpdateFields()
	hooks.Delete.Fields = configDeleteFields()
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *apmconfigv1beta1.Config,
		namespace string,
	) (any, error) {
		return buildConfigCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *apmconfigv1beta1.Config,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildConfigUpdateDetails(ctx, resource, namespace, currentResponse)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapConfigStatusMirrorClient)
}

func newConfigServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client configOCIClient,
) ConfigServiceClient {
	hooks := newConfigRuntimeHooksWithOCIClient(client)
	applyConfigRuntimeHooks(&hooks)
	delegate := defaultConfigServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apmconfigv1beta1.Config](
			buildConfigGeneratedRuntimeConfig(&ConfigServiceManager{Log: log}, hooks),
		),
	}
	return wrapConfigGeneratedClient(hooks, delegate)
}

func newConfigRuntimeHooksWithOCIClient(client configOCIClient) ConfigRuntimeHooks {
	return ConfigRuntimeHooks{
		Semantics: reviewedConfigRuntimeSemantics(),
		Create: runtimeOperationHooks[apmconfigsdk.CreateConfigRequest, apmconfigsdk.CreateConfigResponse]{
			Fields: configCreateFields(),
			Call: func(ctx context.Context, request apmconfigsdk.CreateConfigRequest) (apmconfigsdk.CreateConfigResponse, error) {
				return client.CreateConfig(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apmconfigsdk.GetConfigRequest, apmconfigsdk.GetConfigResponse]{
			Fields: configGetFields(),
			Call: func(ctx context.Context, request apmconfigsdk.GetConfigRequest) (apmconfigsdk.GetConfigResponse, error) {
				return client.GetConfig(ctx, request)
			},
		},
		List: runtimeOperationHooks[apmconfigsdk.ListConfigsRequest, apmconfigsdk.ListConfigsResponse]{
			Fields: configListFields(),
			Call: func(ctx context.Context, request apmconfigsdk.ListConfigsRequest) (apmconfigsdk.ListConfigsResponse, error) {
				return client.ListConfigs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apmconfigsdk.UpdateConfigRequest, apmconfigsdk.UpdateConfigResponse]{
			Fields: configUpdateFields(),
			Call: func(ctx context.Context, request apmconfigsdk.UpdateConfigRequest) (apmconfigsdk.UpdateConfigResponse, error) {
				return client.UpdateConfig(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apmconfigsdk.DeleteConfigRequest, apmconfigsdk.DeleteConfigResponse]{
			Fields: configDeleteFields(),
			Call: func(ctx context.Context, request apmconfigsdk.DeleteConfigRequest) (apmconfigsdk.DeleteConfigResponse, error) {
				return client.DeleteConfig(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ConfigServiceClient) ConfigServiceClient{},
	}
}

func reviewedConfigRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newConfigRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{}
	semantics.Delete = generatedruntime.DeleteSemantics{
		Policy:         "required",
		PendingStates:  []string{},
		TerminalStates: []string{"DELETED"},
	}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields: []string{
			"configType",
			"displayName",
			"group",
			"filterId",
			"filterText",
			"namespace",
			"managementAgentId",
			"matchAgentsWithAttributeValue",
			"serviceName",
		},
	}
	semantics.Mutation = generatedruntime.MutationSemantics{
		Mutable: []string{
			"agentVersion",
			"attachInstallDir",
			"config",
			"definedTags",
			"description",
			"dimensions",
			"displayName",
			"filterId",
			"filterText",
			"freeformTags",
			"group",
			"metrics",
			"namespace",
			"options",
			"overrides",
			"processFilter",
			"rules",
			"runAsUser",
			"serviceName",
		},
		ForceNew: []string{
			"apmDomainId",
			"configType",
			"managementAgentId",
			"matchAgentsWithAttributeValue",
		},
		ConflictsWith: map[string][]string{},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "read-after-write",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "read-after-write",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "confirm-delete",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
	}
	semantics.AuxiliaryOperations = nil
	semantics.Unsupported = nil
	return semantics
}

func configCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "CreateConfigDetails", RequestName: "CreateConfigDetails", Contribution: "body"},
	}
}

func configGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ConfigId", RequestName: "configId", Contribution: "path", PreferResourceID: true},
	}
}

func configListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{
			FieldName:    "ConfigType",
			RequestName:  "configType",
			Contribution: "query",
			LookupPaths:  []string{"status.configType", "spec.configType", "configType"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{
			FieldName:    "OptionsGroup",
			RequestName:  "optionsGroup",
			Contribution: "query",
			LookupPaths:  []string{"status.group", "spec.group", "group"},
		},
	}
}

func configUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ConfigId", RequestName: "configId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateConfigDetails", RequestName: "UpdateConfigDetails", Contribution: "body"},
	}
}

func configDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ConfigId", RequestName: "configId", Contribution: "path", PreferResourceID: true},
	}
}

func guardConfigExistingBeforeCreate(
	_ context.Context,
	resource *apmconfigv1beta1.Config,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Config resource is nil")
	}
	if strings.TrimSpace(resource.Spec.ApmDomainId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Config spec.apmDomainId is required")
	}

	configType, err := configTypeForResource(resource, nil)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}

	switch configType {
	case string(apmconfigsdk.ConfigTypesAgent):
		if strings.TrimSpace(resource.Spec.MatchAgentsWithAttributeValue) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	case string(apmconfigsdk.ConfigTypesApdex):
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	case string(apmconfigsdk.ConfigTypesMacsApmExtension):
		if strings.TrimSpace(resource.Spec.ManagementAgentId) == "" || strings.TrimSpace(resource.Spec.ServiceName) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	case string(apmconfigsdk.ConfigTypesMetricGroup):
		if strings.TrimSpace(resource.Spec.DisplayName) == "" || strings.TrimSpace(resource.Spec.FilterId) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	case string(apmconfigsdk.ConfigTypesOptions):
		if strings.TrimSpace(resource.Spec.DisplayName) == "" && strings.TrimSpace(resource.Spec.Group) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	case string(apmconfigsdk.ConfigTypesSpanFilter):
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	}

	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildConfigCreateDetails(
	ctx context.Context,
	resource *apmconfigv1beta1.Config,
	namespace string,
) (apmconfigsdk.CreateConfigDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("Config resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return nil, err
	}
	patchConfigExplicitCollectionIntent(resource, resolvedSpec)

	configType, err := configTypeForResource(resource, nil)
	if err != nil {
		return nil, err
	}

	specValues, err := configJSONMap(resolvedSpec)
	if err != nil {
		return nil, fmt.Errorf("project Config create spec: %w", err)
	}
	if err := validateConfigCreateRequiredFields(configType, specValues); err != nil {
		return nil, err
	}
	if err := validateConfigTypeSpecificFields(configType, specValues); err != nil {
		return nil, err
	}

	return decodeConfigCreateDetails(configType, resolvedSpec)
}

func buildConfigUpdateDetails(
	ctx context.Context,
	resource *apmconfigv1beta1.Config,
	namespace string,
	currentResponse any,
) (apmconfigsdk.UpdateConfigDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("Config resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return nil, false, err
	}
	patchConfigExplicitCollectionIntent(resource, resolvedSpec)

	configType, err := configTypeForResource(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}

	specValues, err := configJSONMap(resolvedSpec)
	if err != nil {
		return nil, false, fmt.Errorf("project Config update spec: %w", err)
	}
	if err := validateConfigTypeSpecificFields(configType, specValues); err != nil {
		return nil, false, err
	}

	desired, err := decodeConfigUpdateDetails(configType, resolvedSpec)
	if err != nil {
		return nil, false, err
	}
	desiredValues, err := configJSONMap(desired)
	if err != nil {
		return nil, false, fmt.Errorf("project desired Config update body: %w", err)
	}
	if len(desiredValues) == 0 {
		return desired, false, nil
	}

	currentDetails, err := configCurrentUpdateDetails(configType, currentResponse)
	if err != nil {
		return nil, false, err
	}
	currentValues, err := configJSONMap(currentDetails)
	if err != nil {
		return nil, false, fmt.Errorf("project current Config update body: %w", err)
	}

	return desired, !configMapSubsetEqual(desiredValues, currentValues), nil
}

func configCurrentUpdateDetails(
	configType string,
	currentResponse any,
) (apmconfigsdk.UpdateConfigDetails, error) {
	body, err := configRuntimeBody(currentResponse)
	if err != nil {
		return nil, err
	}
	return decodeConfigUpdateDetails(configType, body)
}

func configTypeForResource(
	resource *apmconfigv1beta1.Config,
	currentResponse any,
) (string, error) {
	if resource != nil {
		if configType, ok := normalizeConfigType(resource.Spec.ConfigType); ok {
			return configType, nil
		}
	}

	if currentResponse != nil {
		body, err := configRuntimeBody(currentResponse)
		if err != nil {
			return "", err
		}
		if configType, ok := normalizeConfigType(configTypeFromRuntimeBody(body)); ok {
			return configType, nil
		}
	}

	return "", fmt.Errorf(
		"Config spec.configType must be set to one of %s",
		strings.Join(apmconfigsdk.GetConfigTypesEnumStringValues(), ", "),
	)
}

func normalizeConfigType(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	configType, ok := apmconfigsdk.GetMappingConfigTypesEnum(raw)
	if !ok {
		return "", false
	}
	return string(configType), true
}

func decodeConfigCreateDetails(
	configType string,
	raw any,
) (apmconfigsdk.CreateConfigDetails, error) {
	switch configType {
	case string(apmconfigsdk.ConfigTypesAgent):
		details, err := decodeConfigConcrete[apmconfigsdk.CreateAgentConfigDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesApdex):
		details, err := decodeConfigConcrete[apmconfigsdk.CreateApdexRulesDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesMacsApmExtension):
		details, err := decodeConfigConcrete[apmconfigsdk.CreateMacsApmExtensionDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesMetricGroup):
		details, err := decodeConfigConcrete[apmconfigsdk.CreateMetricGroupDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesOptions):
		details, err := decodeConfigConcrete[apmconfigsdk.CreateOptionsDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesSpanFilter):
		details, err := decodeConfigConcrete[apmconfigsdk.CreateSpanFilterDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported Config create type %q", configType)
	}
}

func decodeConfigUpdateDetails(
	configType string,
	raw any,
) (apmconfigsdk.UpdateConfigDetails, error) {
	switch configType {
	case string(apmconfigsdk.ConfigTypesAgent):
		details, err := decodeConfigConcrete[apmconfigsdk.UpdateAgentConfigDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesApdex):
		details, err := decodeConfigConcrete[apmconfigsdk.UpdateApdexRulesDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesMacsApmExtension):
		details, err := decodeConfigConcrete[apmconfigsdk.UpdateMacsApmExtensionDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesMetricGroup):
		details, err := decodeConfigConcrete[apmconfigsdk.UpdateMetricGroupDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesOptions):
		details, err := decodeConfigConcrete[apmconfigsdk.UpdateOptionsDetails](raw)
		return details, err
	case string(apmconfigsdk.ConfigTypesSpanFilter):
		details, err := decodeConfigConcrete[apmconfigsdk.UpdateSpanFilterDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported Config update type %q", configType)
	}
}

func decodeConfigConcrete[T any](raw any) (T, error) {
	var decoded T

	payload, err := json.Marshal(raw)
	if err != nil {
		return decoded, fmt.Errorf("marshal Config payload: %w", err)
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return decoded, fmt.Errorf("unmarshal Config payload: %w", err)
	}
	return decoded, nil
}

func configRuntimeBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case apmconfigsdk.CreateConfigResponse:
		if current.Config == nil {
			return nil, fmt.Errorf("current Config create response body is nil")
		}
		return current.Config, nil
	case *apmconfigsdk.CreateConfigResponse:
		if current == nil {
			return nil, fmt.Errorf("current Config create response is nil")
		}
		return configRuntimeBody(*current)
	case apmconfigsdk.GetConfigResponse:
		if current.Config == nil {
			return nil, fmt.Errorf("current Config get response body is nil")
		}
		return current.Config, nil
	case *apmconfigsdk.GetConfigResponse:
		if current == nil {
			return nil, fmt.Errorf("current Config get response is nil")
		}
		return configRuntimeBody(*current)
	case apmconfigsdk.UpdateConfigResponse:
		if current.Config == nil {
			return nil, fmt.Errorf("current Config update response body is nil")
		}
		return current.Config, nil
	case *apmconfigsdk.UpdateConfigResponse:
		if current == nil {
			return nil, fmt.Errorf("current Config update response is nil")
		}
		return configRuntimeBody(*current)
	case apmconfigsdk.Config:
		if current == nil {
			return nil, fmt.Errorf("current Config body is nil")
		}
		return current, nil
	case apmconfigsdk.ConfigSummary:
		if current == nil {
			return nil, fmt.Errorf("current Config summary is nil")
		}
		return current, nil
	default:
		return nil, fmt.Errorf("unsupported current Config payload type %T", currentResponse)
	}
}

func configTypeFromRuntimeBody(body any) string {
	values, err := configJSONMap(body)
	if err != nil {
		return ""
	}
	rawType, ok := values["configType"].(string)
	if !ok {
		return ""
	}
	return rawType
}

func validateConfigCreateRequiredFields(configType string, specValues map[string]any) error {
	required := []string{"apmDomainId", "configType"}
	switch configType {
	case string(apmconfigsdk.ConfigTypesAgent):
		required = append(required, "matchAgentsWithAttributeValue")
	case string(apmconfigsdk.ConfigTypesApdex):
		required = append(required, "displayName", "rules")
	case string(apmconfigsdk.ConfigTypesMacsApmExtension):
		required = append(required, "managementAgentId", "processFilter", "runAsUser", "serviceName", "agentVersion", "attachInstallDir")
	case string(apmconfigsdk.ConfigTypesMetricGroup):
		required = append(required, "displayName", "filterId", "metrics")
	case string(apmconfigsdk.ConfigTypesSpanFilter):
		required = append(required, "displayName", "filterText")
	}

	var missing []string
	for _, field := range required {
		if !configHasMeaningfulValue(specValues, field) {
			missing = append(missing, "spec."+field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("Config requires %s for %s resources", strings.Join(missing, ", "), configType)
	}
	return nil
}

func validateConfigTypeSpecificFields(configType string, specValues map[string]any) error {
	if len(specValues) == 0 {
		return nil
	}

	var unsupported []string
	switch configType {
	case string(apmconfigsdk.ConfigTypesAgent):
		unsupported = configUnsupportedFields(
			specValues,
			"displayName",
			"filterText",
			"description",
			"filterId",
			"metrics",
			"namespace",
			"dimensions",
			"options",
			"group",
			"managementAgentId",
			"processFilter",
			"runAsUser",
			"serviceName",
			"agentVersion",
			"attachInstallDir",
			"rules",
		)
	case string(apmconfigsdk.ConfigTypesApdex):
		unsupported = configUnsupportedFields(
			specValues,
			"filterText",
			"description",
			"filterId",
			"metrics",
			"namespace",
			"dimensions",
			"matchAgentsWithAttributeValue",
			"config",
			"overrides",
			"options",
			"group",
			"managementAgentId",
			"processFilter",
			"runAsUser",
			"serviceName",
			"agentVersion",
			"attachInstallDir",
		)
	case string(apmconfigsdk.ConfigTypesMacsApmExtension):
		unsupported = configUnsupportedFields(
			specValues,
			"filterText",
			"description",
			"filterId",
			"metrics",
			"namespace",
			"dimensions",
			"matchAgentsWithAttributeValue",
			"config",
			"overrides",
			"options",
			"group",
			"rules",
		)
	case string(apmconfigsdk.ConfigTypesMetricGroup):
		unsupported = configUnsupportedFields(
			specValues,
			"filterText",
			"description",
			"matchAgentsWithAttributeValue",
			"config",
			"overrides",
			"options",
			"group",
			"managementAgentId",
			"processFilter",
			"runAsUser",
			"serviceName",
			"agentVersion",
			"attachInstallDir",
			"rules",
		)
	case string(apmconfigsdk.ConfigTypesOptions):
		unsupported = configUnsupportedFields(
			specValues,
			"filterText",
			"filterId",
			"metrics",
			"namespace",
			"dimensions",
			"matchAgentsWithAttributeValue",
			"config",
			"overrides",
			"managementAgentId",
			"processFilter",
			"runAsUser",
			"serviceName",
			"agentVersion",
			"attachInstallDir",
			"rules",
		)
	case string(apmconfigsdk.ConfigTypesSpanFilter):
		unsupported = configUnsupportedFields(
			specValues,
			"filterId",
			"metrics",
			"namespace",
			"dimensions",
			"matchAgentsWithAttributeValue",
			"config",
			"overrides",
			"options",
			"group",
			"managementAgentId",
			"processFilter",
			"runAsUser",
			"serviceName",
			"agentVersion",
			"attachInstallDir",
			"rules",
		)
	default:
		return fmt.Errorf("unsupported Config type %q", configType)
	}
	if len(unsupported) == 0 {
		return nil
	}

	return fmt.Errorf("Config type %s does not support %s", configType, strings.Join(unsupported, ", "))
}

func configUnsupportedFields(specValues map[string]any, fieldNames ...string) []string {
	var unsupported []string
	for _, fieldName := range fieldNames {
		if configHasMeaningfulValue(specValues, fieldName) {
			unsupported = append(unsupported, "spec."+fieldName)
		}
	}
	return unsupported
}

func configHasMeaningfulValue(specValues map[string]any, field string) bool {
	value, ok := specValues[field]
	if !ok || value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case float64:
		return concrete != 0
	case bool:
		return concrete
	case []any:
		return len(concrete) > 0
	case map[string]any:
		return len(concrete) > 0
	default:
		return true
	}
}

func configJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func configMapSubsetEqual(desired map[string]any, current map[string]any) bool {
	for key, desiredValue := range desired {
		currentValue, ok := current[key]
		if !ok || !configValuesEqual(desiredValue, currentValue) {
			return false
		}
	}
	return true
}

func configValuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return string(leftPayload) == string(rightPayload)
}

func patchConfigExplicitCollectionIntent(resource *apmconfigv1beta1.Config, raw any) {
	if resource == nil {
		return
	}

	values, ok := raw.(map[string]any)
	if !ok {
		return
	}

	if resource.Spec.FreeformTags != nil && len(resource.Spec.FreeformTags) == 0 {
		values["freeformTags"] = map[string]string{}
	}
	if resource.Spec.DefinedTags != nil && len(resource.Spec.DefinedTags) == 0 {
		values["definedTags"] = map[string]any{}
	}
	if resource.Spec.ProcessFilter != nil && len(resource.Spec.ProcessFilter) == 0 {
		values["processFilter"] = []any{}
	}
	if resource.Spec.Dimensions != nil && len(resource.Spec.Dimensions) == 0 {
		values["dimensions"] = []any{}
	}
	if resource.Spec.Metrics != nil && len(resource.Spec.Metrics) == 0 {
		values["metrics"] = []any{}
	}
	if resource.Spec.Rules != nil && len(resource.Spec.Rules) == 0 {
		values["rules"] = []any{}
	}
	if resource.Spec.Config.ConfigMap != nil && len(resource.Spec.Config.ConfigMap) == 0 {
		ensureConfigMapChild(values, "config")["configMap"] = map[string]any{}
	}
	if resource.Spec.Overrides.OverrideList != nil && len(resource.Spec.Overrides.OverrideList) == 0 {
		ensureConfigMapChild(values, "overrides")["overrideList"] = []any{}
	}
	if len(resource.Spec.Overrides.OverrideList) > 0 {
		overrides := ensureConfigMapChild(values, "overrides")
		rawList, ok := overrides["overrideList"].([]any)
		if !ok {
			return
		}
		for i, item := range resource.Spec.Overrides.OverrideList {
			if item.OverrideMap == nil || len(item.OverrideMap) != 0 || i >= len(rawList) {
				continue
			}
			itemMap, ok := rawList[i].(map[string]any)
			if !ok {
				itemMap = map[string]any{}
				rawList[i] = itemMap
			}
			itemMap["overrideMap"] = map[string]any{}
		}
		overrides["overrideList"] = rawList
	}
}

func ensureConfigMapChild(values map[string]any, key string) map[string]any {
	child, ok := values[key].(map[string]any)
	if ok {
		return child
	}
	child = map[string]any{}
	values[key] = child
	return child
}

type configStatusMirrorClient struct {
	delegate ConfigServiceClient
}

func wrapConfigStatusMirrorClient(delegate ConfigServiceClient) ConfigServiceClient {
	return configStatusMirrorClient{delegate: delegate}
}

func (c configStatusMirrorClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmconfigv1beta1.Config,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful {
		projectConfigRequestContext(resource)
	}
	return response, err
}

func (c configStatusMirrorClient) Delete(ctx context.Context, resource *apmconfigv1beta1.Config) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func projectConfigRequestContext(resource *apmconfigv1beta1.Config) {
	if resource == nil {
		return
	}

	resource.Status.ApmDomainId = strings.TrimSpace(resource.Spec.ApmDomainId)
	if strings.TrimSpace(resource.Status.ConfigType) == "" {
		resource.Status.ConfigType = strings.TrimSpace(resource.Spec.ConfigType)
	}
}
