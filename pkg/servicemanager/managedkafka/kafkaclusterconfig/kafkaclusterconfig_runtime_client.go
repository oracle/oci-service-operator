/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kafkaclusterconfig

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	managedkafkasdk "github.com/oracle/oci-go-sdk/v65/managedkafka"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/util"
)

type kafkaClusterConfigOCIClient interface {
	CreateKafkaClusterConfig(context.Context, managedkafkasdk.CreateKafkaClusterConfigRequest) (managedkafkasdk.CreateKafkaClusterConfigResponse, error)
	GetKafkaClusterConfig(context.Context, managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error)
	ListKafkaClusterConfigs(context.Context, managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error)
	UpdateKafkaClusterConfig(context.Context, managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error)
	DeleteKafkaClusterConfig(context.Context, managedkafkasdk.DeleteKafkaClusterConfigRequest) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error)
}

func init() {
	registerKafkaClusterConfigRuntimeHooksMutator(func(_ *KafkaClusterConfigServiceManager, hooks *KafkaClusterConfigRuntimeHooks) {
		applyKafkaClusterConfigRuntimeHooks(hooks)
	})
}

func applyKafkaClusterConfigRuntimeHooks(hooks *KafkaClusterConfigRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newKafkaClusterConfigRuntimeSemantics()
	hooks.BuildCreateBody = buildKafkaClusterConfigCreateBody
	hooks.BuildUpdateBody = buildKafkaClusterConfigUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardKafkaClusterConfigExistingBeforeCreate
	hooks.List.Fields = kafkaClusterConfigListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedKafkaClusterConfigIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateKafkaClusterConfigCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleKafkaClusterConfigDeleteError
	wrapKafkaClusterConfigReadAndDeleteCalls(hooks)
}

func newKafkaClusterConfigServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client kafkaClusterConfigOCIClient,
) KafkaClusterConfigServiceClient {
	hooks := newKafkaClusterConfigRuntimeHooksWithOCIClient(client)
	applyKafkaClusterConfigRuntimeHooks(&hooks)
	manager := &KafkaClusterConfigServiceManager{Log: log}
	return defaultKafkaClusterConfigServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*managedkafkav1beta1.KafkaClusterConfig](
			buildKafkaClusterConfigGeneratedRuntimeConfig(manager, hooks),
		),
	}
}

func newKafkaClusterConfigRuntimeHooksWithOCIClient(client kafkaClusterConfigOCIClient) KafkaClusterConfigRuntimeHooks {
	return KafkaClusterConfigRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*managedkafkav1beta1.KafkaClusterConfig]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*managedkafkav1beta1.KafkaClusterConfig]{},
		StatusHooks:     generatedruntime.StatusHooks[*managedkafkav1beta1.KafkaClusterConfig]{},
		ParityHooks:     generatedruntime.ParityHooks[*managedkafkav1beta1.KafkaClusterConfig]{},
		Async:           generatedruntime.AsyncHooks[*managedkafkav1beta1.KafkaClusterConfig]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*managedkafkav1beta1.KafkaClusterConfig]{},
		Create: runtimeOperationHooks[managedkafkasdk.CreateKafkaClusterConfigRequest, managedkafkasdk.CreateKafkaClusterConfigResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateKafkaClusterConfigDetails", RequestName: "CreateKafkaClusterConfigDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request managedkafkasdk.CreateKafkaClusterConfigRequest) (managedkafkasdk.CreateKafkaClusterConfigResponse, error) {
				if client == nil {
					return managedkafkasdk.CreateKafkaClusterConfigResponse{}, fmt.Errorf("KafkaClusterConfig OCI client is nil")
				}
				return client.CreateKafkaClusterConfig(ctx, request)
			},
		},
		Get: runtimeOperationHooks[managedkafkasdk.GetKafkaClusterConfigRequest, managedkafkasdk.GetKafkaClusterConfigResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "KafkaClusterConfigId", RequestName: "kafkaClusterConfigId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
				if client == nil {
					return managedkafkasdk.GetKafkaClusterConfigResponse{}, fmt.Errorf("KafkaClusterConfig OCI client is nil")
				}
				return client.GetKafkaClusterConfig(ctx, request)
			},
		},
		List: runtimeOperationHooks[managedkafkasdk.ListKafkaClusterConfigsRequest, managedkafkasdk.ListKafkaClusterConfigsResponse]{
			Fields: kafkaClusterConfigListFields(),
			Call: func(ctx context.Context, request managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
				if client == nil {
					return managedkafkasdk.ListKafkaClusterConfigsResponse{}, fmt.Errorf("KafkaClusterConfig OCI client is nil")
				}
				return client.ListKafkaClusterConfigs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[managedkafkasdk.UpdateKafkaClusterConfigRequest, managedkafkasdk.UpdateKafkaClusterConfigResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "KafkaClusterConfigId", RequestName: "kafkaClusterConfigId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateKafkaClusterConfigDetails", RequestName: "UpdateKafkaClusterConfigDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request managedkafkasdk.UpdateKafkaClusterConfigRequest) (managedkafkasdk.UpdateKafkaClusterConfigResponse, error) {
				if client == nil {
					return managedkafkasdk.UpdateKafkaClusterConfigResponse{}, fmt.Errorf("KafkaClusterConfig OCI client is nil")
				}
				return client.UpdateKafkaClusterConfig(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[managedkafkasdk.DeleteKafkaClusterConfigRequest, managedkafkasdk.DeleteKafkaClusterConfigResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "KafkaClusterConfigId", RequestName: "kafkaClusterConfigId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request managedkafkasdk.DeleteKafkaClusterConfigRequest) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error) {
				if client == nil {
					return managedkafkasdk.DeleteKafkaClusterConfigResponse{}, fmt.Errorf("KafkaClusterConfig OCI client is nil")
				}
				return client.DeleteKafkaClusterConfig(ctx, request)
			},
		},
		WrapGeneratedClient: []func(KafkaClusterConfigServiceClient) KafkaClusterConfigServiceClient{},
	}
}

func newKafkaClusterConfigRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "managedkafka",
		FormalSlug:    "kafkaclusterconfig",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(managedkafkasdk.KafkaClusterConfigLifecycleStateCreating)},
			UpdatingStates:     []string{string(managedkafkasdk.KafkaClusterConfigLifecycleStateUpdating)},
			ActiveStates:       []string{string(managedkafkasdk.KafkaClusterConfigLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(managedkafkasdk.KafkaClusterConfigLifecycleStateCreating),
				string(managedkafkasdk.KafkaClusterConfigLifecycleStateUpdating),
			},
			TerminalStates: []string{string(managedkafkasdk.KafkaClusterConfigLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "latestConfig.properties", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "latestConfig.properties", "freeformTags", "definedTags"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "KafkaClusterConfig", Action: "CreateKafkaClusterConfig"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "KafkaClusterConfig", Action: "UpdateKafkaClusterConfig"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "KafkaClusterConfig", Action: "DeleteKafkaClusterConfig"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "KafkaClusterConfig", Action: "GetKafkaClusterConfig"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "KafkaClusterConfig", Action: "GetKafkaClusterConfig"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "KafkaClusterConfig", Action: "GetKafkaClusterConfig"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func kafkaClusterConfigListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardKafkaClusterConfigExistingBeforeCreate(
	_ context.Context,
	resource *managedkafkav1beta1.KafkaClusterConfig,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("KafkaClusterConfig resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildKafkaClusterConfigCreateBody(
	_ context.Context,
	resource *managedkafkav1beta1.KafkaClusterConfig,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("KafkaClusterConfig resource is nil")
	}
	if err := validateKafkaClusterConfigSpec(resource.Spec); err != nil {
		return nil, err
	}

	details := managedkafkasdk.CreateKafkaClusterConfigDetails{
		CompartmentId: commonString(strings.TrimSpace(resource.Spec.CompartmentId)),
		LatestConfig:  kafkaClusterConfigVersionFromSpec(resource.Spec.LatestConfig),
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		details.DisplayName = commonString(displayName)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildKafkaClusterConfigUpdateBody(
	_ context.Context,
	resource *managedkafkav1beta1.KafkaClusterConfig,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return managedkafkasdk.UpdateKafkaClusterConfigDetails{}, false, fmt.Errorf("KafkaClusterConfig resource is nil")
	}
	if err := validateKafkaClusterConfigSpec(resource.Spec); err != nil {
		return managedkafkasdk.UpdateKafkaClusterConfigDetails{}, false, err
	}
	current, ok := kafkaClusterConfigFromResponse(currentResponse)
	if !ok {
		return managedkafkasdk.UpdateKafkaClusterConfigDetails{}, false, fmt.Errorf("current KafkaClusterConfig response does not expose a KafkaClusterConfig body")
	}
	if err := validateKafkaClusterConfigCreateOnlyDrift(resource.Spec, current); err != nil {
		return managedkafkasdk.UpdateKafkaClusterConfigDetails{}, false, err
	}

	details := managedkafkasdk.UpdateKafkaClusterConfigDetails{}
	updateNeeded := false
	updateNeeded = applyKafkaClusterConfigDisplayNameUpdate(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyKafkaClusterConfigLatestConfigUpdate(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyKafkaClusterConfigFreeformTagsUpdate(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyKafkaClusterConfigDefinedTagsUpdate(&details, resource.Spec, current) || updateNeeded
	return details, updateNeeded, nil
}

func applyKafkaClusterConfigDisplayNameUpdate(
	details *managedkafkasdk.UpdateKafkaClusterConfigDetails,
	spec managedkafkav1beta1.KafkaClusterConfigSpec,
	current managedkafkasdk.KafkaClusterConfig,
) bool {
	desired, ok := kafkaClusterConfigStringUpdate(spec.DisplayName, current.DisplayName)
	if !ok {
		return false
	}
	details.DisplayName = desired
	return true
}

func applyKafkaClusterConfigLatestConfigUpdate(
	details *managedkafkasdk.UpdateKafkaClusterConfigDetails,
	spec managedkafkav1beta1.KafkaClusterConfigSpec,
	current managedkafkasdk.KafkaClusterConfig,
) bool {
	if kafkaClusterConfigLatestConfigPropertiesEqual(spec.LatestConfig, current.LatestConfig) {
		return false
	}
	details.LatestConfig = kafkaClusterConfigVersionFromSpec(spec.LatestConfig)
	return true
}

func applyKafkaClusterConfigFreeformTagsUpdate(
	details *managedkafkasdk.UpdateKafkaClusterConfigDetails,
	spec managedkafkav1beta1.KafkaClusterConfigSpec,
	current managedkafkasdk.KafkaClusterConfig,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	details.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func applyKafkaClusterConfigDefinedTagsUpdate(
	details *managedkafkasdk.UpdateKafkaClusterConfigDetails,
	spec managedkafkav1beta1.KafkaClusterConfigSpec,
	current managedkafkasdk.KafkaClusterConfig,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	if reflect.DeepEqual(desired, current.DefinedTags) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func validateKafkaClusterConfigSpec(spec managedkafkav1beta1.KafkaClusterConfigSpec) error {
	var problems []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if len(spec.LatestConfig.Properties) == 0 {
		problems = append(problems, "latestConfig.properties is required")
	}
	if strings.TrimSpace(spec.LatestConfig.ConfigId) != "" {
		problems = append(problems, "latestConfig.configId is service-assigned and cannot be set in spec")
	}
	if spec.LatestConfig.VersionNumber != 0 {
		problems = append(problems, "latestConfig.versionNumber is service-assigned and cannot be set in spec")
	}
	if strings.TrimSpace(spec.LatestConfig.TimeCreated) != "" {
		problems = append(problems, "latestConfig.timeCreated is service-assigned and cannot be set in spec")
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("KafkaClusterConfig spec is invalid: %s", strings.Join(problems, "; "))
}

func validateKafkaClusterConfigCreateOnlyDriftForResponse(
	resource *managedkafkav1beta1.KafkaClusterConfig,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("KafkaClusterConfig resource is nil")
	}
	current, ok := kafkaClusterConfigFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current KafkaClusterConfig response does not expose a KafkaClusterConfig body")
	}
	return validateKafkaClusterConfigCreateOnlyDrift(resource.Spec, current)
}

func validateKafkaClusterConfigCreateOnlyDrift(
	spec managedkafkav1beta1.KafkaClusterConfigSpec,
	current managedkafkasdk.KafkaClusterConfig,
) error {
	if stringPtrValue(current.CompartmentId) == strings.TrimSpace(spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("KafkaClusterConfig create-only field drift is not supported: compartmentId")
}

func kafkaClusterConfigVersionFromSpec(
	spec managedkafkav1beta1.KafkaClusterConfigLatestConfig,
) *managedkafkasdk.KafkaClusterConfigVersion {
	return &managedkafkasdk.KafkaClusterConfigVersion{
		Properties: maps.Clone(spec.Properties),
	}
}

func kafkaClusterConfigStringUpdate(spec string, current *string) (*string, bool) {
	desired := strings.TrimSpace(spec)
	if desired == "" {
		return nil, false
	}
	if desired == stringPtrValue(current) {
		return nil, false
	}
	return commonString(desired), true
}

func kafkaClusterConfigLatestConfigPropertiesEqual(
	spec managedkafkav1beta1.KafkaClusterConfigLatestConfig,
	current *managedkafkasdk.KafkaClusterConfigVersion,
) bool {
	if current == nil {
		return len(spec.Properties) == 0
	}
	return maps.Equal(spec.Properties, current.Properties)
}

func wrapKafkaClusterConfigReadAndDeleteCalls(hooks *KafkaClusterConfigRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request managedkafkasdk.GetKafkaClusterConfigRequest) (managedkafkasdk.GetKafkaClusterConfigResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeKafkaClusterConfigNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
			return listKafkaClusterConfigsAllPages(ctx, call, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request managedkafkasdk.DeleteKafkaClusterConfigRequest) (managedkafkasdk.DeleteKafkaClusterConfigResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeKafkaClusterConfigNotFoundError(err, "delete")
		}
	}
}

func listKafkaClusterConfigsAllPages(
	ctx context.Context,
	call func(context.Context, managedkafkasdk.ListKafkaClusterConfigsRequest) (managedkafkasdk.ListKafkaClusterConfigsResponse, error),
	request managedkafkasdk.ListKafkaClusterConfigsRequest,
) (managedkafkasdk.ListKafkaClusterConfigsResponse, error) {
	var combined managedkafkasdk.ListKafkaClusterConfigsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return managedkafkasdk.ListKafkaClusterConfigsResponse{}, conservativeKafkaClusterConfigNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == managedkafkasdk.KafkaClusterConfigLifecycleStateDeleted {
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

func handleKafkaClusterConfigDeleteError(resource *managedkafkav1beta1.KafkaClusterConfig, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

type kafkaClusterConfigAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e kafkaClusterConfigAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e kafkaClusterConfigAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func conservativeKafkaClusterConfigNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return kafkaClusterConfigAmbiguousNotFoundError{
		message:      fmt.Sprintf("KafkaClusterConfig %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func clearTrackedKafkaClusterConfigIdentity(resource *managedkafkav1beta1.KafkaClusterConfig) {
	if resource == nil {
		return
	}
	resource.Status = managedkafkav1beta1.KafkaClusterConfigStatus{}
}

func kafkaClusterConfigFromResponse(response any) (managedkafkasdk.KafkaClusterConfig, bool) {
	if current, ok := kafkaClusterConfigFromDirectResponse(response); ok {
		return current, true
	}
	return kafkaClusterConfigFromOperationResponse(response)
}

func kafkaClusterConfigFromDirectResponse(response any) (managedkafkasdk.KafkaClusterConfig, bool) {
	switch current := response.(type) {
	case managedkafkasdk.KafkaClusterConfig:
		return current, true
	case *managedkafkasdk.KafkaClusterConfig:
		return kafkaClusterConfigFromOptional(current, func(config managedkafkasdk.KafkaClusterConfig) managedkafkasdk.KafkaClusterConfig {
			return config
		})
	case managedkafkasdk.KafkaClusterConfigSummary:
		return kafkaClusterConfigFromSummary(current), true
	case *managedkafkasdk.KafkaClusterConfigSummary:
		return kafkaClusterConfigFromOptional(current, kafkaClusterConfigFromSummary)
	default:
		return managedkafkasdk.KafkaClusterConfig{}, false
	}
}

func kafkaClusterConfigFromOperationResponse(response any) (managedkafkasdk.KafkaClusterConfig, bool) {
	switch current := response.(type) {
	case managedkafkasdk.CreateKafkaClusterConfigResponse:
		return current.KafkaClusterConfig, true
	case *managedkafkasdk.CreateKafkaClusterConfigResponse:
		return kafkaClusterConfigFromOptional(current, kafkaClusterConfigFromCreateResponse)
	case managedkafkasdk.GetKafkaClusterConfigResponse:
		return current.KafkaClusterConfig, true
	case *managedkafkasdk.GetKafkaClusterConfigResponse:
		return kafkaClusterConfigFromOptional(current, kafkaClusterConfigFromGetResponse)
	case managedkafkasdk.UpdateKafkaClusterConfigResponse:
		return current.KafkaClusterConfig, true
	case *managedkafkasdk.UpdateKafkaClusterConfigResponse:
		return kafkaClusterConfigFromOptional(current, kafkaClusterConfigFromUpdateResponse)
	default:
		return managedkafkasdk.KafkaClusterConfig{}, false
	}
}

func kafkaClusterConfigFromOptional[T any](
	current *T,
	convert func(T) managedkafkasdk.KafkaClusterConfig,
) (managedkafkasdk.KafkaClusterConfig, bool) {
	if current == nil {
		return managedkafkasdk.KafkaClusterConfig{}, false
	}
	return convert(*current), true
}

func kafkaClusterConfigFromCreateResponse(response managedkafkasdk.CreateKafkaClusterConfigResponse) managedkafkasdk.KafkaClusterConfig {
	return response.KafkaClusterConfig
}

func kafkaClusterConfigFromGetResponse(response managedkafkasdk.GetKafkaClusterConfigResponse) managedkafkasdk.KafkaClusterConfig {
	return response.KafkaClusterConfig
}

func kafkaClusterConfigFromUpdateResponse(response managedkafkasdk.UpdateKafkaClusterConfigResponse) managedkafkasdk.KafkaClusterConfig {
	return response.KafkaClusterConfig
}

func kafkaClusterConfigFromSummary(summary managedkafkasdk.KafkaClusterConfigSummary) managedkafkasdk.KafkaClusterConfig {
	return managedkafkasdk.KafkaClusterConfig{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		CompartmentId:    summary.CompartmentId,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleDetails: summary.LifecycleDetails,
		SystemTags:       summary.SystemTags,
	}
}

func commonString(value string) *string {
	return &value
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ error = kafkaClusterConfigAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = kafkaClusterConfigAmbiguousNotFoundError{}
