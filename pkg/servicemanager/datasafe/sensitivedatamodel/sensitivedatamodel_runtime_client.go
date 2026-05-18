/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivedatamodel

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const sensitiveDataModelKind = "SensitiveDataModel"

type sensitiveDataModelOCIClient interface {
	CreateSensitiveDataModel(context.Context, datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error)
	GetSensitiveDataModel(context.Context, datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error)
	ListSensitiveDataModels(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error)
	UpdateSensitiveDataModel(context.Context, datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error)
	DeleteSensitiveDataModel(context.Context, datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error)
}

type sensitiveDataModelRuntimeClient struct {
	delegate SensitiveDataModelServiceClient
	get      func(context.Context, datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error)
	list     func(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error)
}

type sensitiveDataModelMutationRecorder struct {
	phase         shared.OSOKAsyncPhase
	workRequestID string
	opcRequestID  string
}

type sensitiveDataModelMutationRecorderKey struct{}

type sensitiveDataModelAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e sensitiveDataModelAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e sensitiveDataModelAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSensitiveDataModelRuntimeHooksMutator(func(_ *SensitiveDataModelServiceManager, hooks *SensitiveDataModelRuntimeHooks) {
		applySensitiveDataModelRuntimeHooks(hooks)
	})
}

func applySensitiveDataModelRuntimeHooks(hooks *SensitiveDataModelRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = sensitiveDataModelRuntimeSemantics()
	hooks.BuildCreateBody = buildSensitiveDataModelCreateBody
	hooks.BuildUpdateBody = buildSensitiveDataModelUpdateBody
	hooks.Create.Fields = sensitiveDataModelCreateFields()
	hooks.Get.Fields = sensitiveDataModelGetFields()
	hooks.List.Fields = sensitiveDataModelListFields()
	hooks.Update.Fields = sensitiveDataModelUpdateFields()
	hooks.Delete.Fields = sensitiveDataModelDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listSensitiveDataModelsAllPages(hooks.List.Call)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedSensitiveDataModelIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateSensitiveDataModelCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleSensitiveDataModelDeleteError
	hooks.DeleteHooks.ApplyOutcome = applySensitiveDataModelDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectSensitiveDataModelStatus
	hooks.StatusHooks.MarkTerminating = markSensitiveDataModelTerminating
	hooks.Identity.GuardExistingBeforeCreate = guardSensitiveDataModelExistingBeforeCreate
	wrapSensitiveDataModelMutationRecorders(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SensitiveDataModelServiceClient) SensitiveDataModelServiceClient {
		return sensitiveDataModelRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newSensitiveDataModelServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client sensitiveDataModelOCIClient,
) SensitiveDataModelServiceClient {
	hooks := newSensitiveDataModelRuntimeHooksWithOCIClient(client)
	applySensitiveDataModelRuntimeHooks(&hooks)
	manager := &SensitiveDataModelServiceManager{Log: log}
	delegate := defaultSensitiveDataModelServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveDataModel](
			buildSensitiveDataModelGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSensitiveDataModelGeneratedClient(hooks, delegate)
}

func newSensitiveDataModelRuntimeHooksWithOCIClient(client sensitiveDataModelOCIClient) SensitiveDataModelRuntimeHooks {
	return SensitiveDataModelRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.SensitiveDataModel]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.SensitiveDataModel]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.SensitiveDataModel]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.SensitiveDataModel]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.SensitiveDataModel]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.SensitiveDataModel]{},
		Create: runtimeOperationHooks[datasafesdk.CreateSensitiveDataModelRequest, datasafesdk.CreateSensitiveDataModelResponse]{
			Fields: sensitiveDataModelCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
				if client == nil {
					return datasafesdk.CreateSensitiveDataModelResponse{}, fmt.Errorf("%s OCI client is nil", sensitiveDataModelKind)
				}
				return client.CreateSensitiveDataModel(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetSensitiveDataModelRequest, datasafesdk.GetSensitiveDataModelResponse]{
			Fields: sensitiveDataModelGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
				if client == nil {
					return datasafesdk.GetSensitiveDataModelResponse{}, fmt.Errorf("%s OCI client is nil", sensitiveDataModelKind)
				}
				return client.GetSensitiveDataModel(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListSensitiveDataModelsRequest, datasafesdk.ListSensitiveDataModelsResponse]{
			Fields: sensitiveDataModelListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error) {
				if client == nil {
					return datasafesdk.ListSensitiveDataModelsResponse{}, fmt.Errorf("%s OCI client is nil", sensitiveDataModelKind)
				}
				return client.ListSensitiveDataModels(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateSensitiveDataModelRequest, datasafesdk.UpdateSensitiveDataModelResponse]{
			Fields: sensitiveDataModelUpdateFields(),
			Call: func(ctx context.Context, request datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
				if client == nil {
					return datasafesdk.UpdateSensitiveDataModelResponse{}, fmt.Errorf("%s OCI client is nil", sensitiveDataModelKind)
				}
				return client.UpdateSensitiveDataModel(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteSensitiveDataModelRequest, datasafesdk.DeleteSensitiveDataModelResponse]{
			Fields: sensitiveDataModelDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
				if client == nil {
					return datasafesdk.DeleteSensitiveDataModelResponse{}, fmt.Errorf("%s OCI client is nil", sensitiveDataModelKind)
				}
				return client.DeleteSensitiveDataModel(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SensitiveDataModelServiceClient) SensitiveDataModelServiceClient{},
	}
}

func sensitiveDataModelRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "sensitivedatamodel",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.DiscoveryLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.DiscoveryLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.DiscoveryLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.DiscoveryLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.DiscoveryLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"targetId",
				"displayName",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"targetId",
				"appSuiteName",
				"description",
				"schemasForDiscovery",
				"tablesForDiscovery",
				"sensitiveTypeIdsForDiscovery",
				"sensitiveTypeGroupIdsForDiscovery",
				"isSampleDataCollectionEnabled",
				"isAppDefinedRelationDiscoveryEnabled",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"isIncludeAllSchemas",
				"isIncludeAllSensitiveTypes",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: sensitiveDataModelKind, Action: "CreateSensitiveDataModel"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: sensitiveDataModelKind, Action: "UpdateSensitiveDataModel"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: sensitiveDataModelKind, Action: "DeleteSensitiveDataModel"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func sensitiveDataModelCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSensitiveDataModelDetails", RequestName: "CreateSensitiveDataModelDetails", Contribution: "body"},
	}
}

func sensitiveDataModelGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SensitiveDataModelId", RequestName: "sensitiveDataModelId", Contribution: "path", PreferResourceID: true},
	}
}

func sensitiveDataModelListFields() []generatedruntime.RequestField {
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
		{
			FieldName:    "TargetId",
			RequestName:  "targetId",
			Contribution: "query",
			LookupPaths:  []string{"status.targetId", "spec.targetId", "targetId"},
		},
		{FieldName: "SensitiveDataModelId", RequestName: "sensitiveDataModelId", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "TimeCreatedGreaterThanOrEqualTo", RequestName: "timeCreatedGreaterThanOrEqualTo", Contribution: "query"},
		{FieldName: "TimeCreatedLessThan", RequestName: "timeCreatedLessThan", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
	}
}

func sensitiveDataModelUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SensitiveDataModelId", RequestName: "sensitiveDataModelId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSensitiveDataModelDetails", RequestName: "UpdateSensitiveDataModelDetails", Contribution: "body"},
	}
}

func sensitiveDataModelDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SensitiveDataModelId", RequestName: "sensitiveDataModelId", Contribution: "path", PreferResourceID: true},
	}
}

func buildSensitiveDataModelCreateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", sensitiveDataModelKind)
	}
	if err := validateSensitiveDataModelSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := datasafesdk.CreateSensitiveDataModelDetails{
		CompartmentId:                        common.String(strings.TrimSpace(spec.CompartmentId)),
		TargetId:                             common.String(strings.TrimSpace(spec.TargetId)),
		IsSampleDataCollectionEnabled:        common.Bool(spec.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: common.Bool(spec.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  common.Bool(spec.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           common.Bool(spec.IsIncludeAllSensitiveTypes),
		SchemasForDiscovery:                  cloneSensitiveDataModelStringSlice(spec.SchemasForDiscovery),
		TablesForDiscovery:                   sensitiveDataModelTablesForDiscoveryFromSpec(spec.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         cloneSensitiveDataModelStringSlice(spec.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    cloneSensitiveDataModelStringSlice(spec.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         maps.Clone(spec.FreeformTags),
		DefinedTags:                          sensitiveDataModelDefinedTagsFromSpec(spec.DefinedTags),
	}
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.AppSuiteName); value != "" {
		body.AppSuiteName = common.String(value)
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	return body, nil
}

func buildSensitiveDataModelUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", sensitiveDataModelKind)
	}
	if err := validateSensitiveDataModelSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := sensitiveDataModelFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a full %s body", sensitiveDataModelKind, sensitiveDataModelKind)
	}
	if err := validateSensitiveDataModelCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}

	body, updateNeeded := sensitiveDataModelMutableUpdateBody(resource.Spec, current)
	return body, updateNeeded, nil
}

func sensitiveDataModelMutableUpdateBody(
	spec datasafev1beta1.SensitiveDataModelSpec,
	current datasafesdk.SensitiveDataModel,
) (datasafesdk.UpdateSensitiveDataModelDetails, bool) {
	body := datasafesdk.UpdateSensitiveDataModelDetails{}
	updateNeeded := false
	for _, apply := range sensitiveDataModelUpdateMutators {
		if apply(&body, current, spec) {
			updateNeeded = true
		}
	}
	return body, updateNeeded
}

type sensitiveDataModelUpdateMutator func(
	*datasafesdk.UpdateSensitiveDataModelDetails,
	datasafesdk.SensitiveDataModel,
	datasafev1beta1.SensitiveDataModelSpec,
) bool

var sensitiveDataModelUpdateMutators = []sensitiveDataModelUpdateMutator{
	setSensitiveDataModelDisplayName,
	setSensitiveDataModelTargetID,
	setSensitiveDataModelAppSuiteName,
	setSensitiveDataModelDescription,
	setSensitiveDataModelSchemasForDiscovery,
	setSensitiveDataModelTablesForDiscovery,
	setSensitiveDataModelSensitiveTypeIDs,
	setSensitiveDataModelSensitiveTypeGroupIDs,
	setSensitiveDataModelSampleDataCollection,
	setSensitiveDataModelAppDefinedRelationDiscovery,
	setSensitiveDataModelFreeformTags,
	setSensitiveDataModelDefinedTags,
}

func setSensitiveDataModelDisplayName(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	return setSensitiveDataModelOptionalTrimmedString(&body.DisplayName, current.DisplayName, spec.DisplayName)
}

func setSensitiveDataModelTargetID(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if sensitiveDataModelStringPtrEqual(current.TargetId, spec.TargetId) {
		return false
	}
	body.TargetId = common.String(strings.TrimSpace(spec.TargetId))
	return true
}

func setSensitiveDataModelAppSuiteName(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	return setSensitiveDataModelOptionalTrimmedString(&body.AppSuiteName, current.AppSuiteName, spec.AppSuiteName)
}

func setSensitiveDataModelDescription(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if strings.TrimSpace(spec.Description) == "" || sensitiveDataModelStringPtrEqual(current.Description, spec.Description) {
		return false
	}
	body.Description = common.String(spec.Description)
	return true
}

func setSensitiveDataModelOptionalTrimmedString(field **string, current *string, desired string) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" || sensitiveDataModelStringPtrEqual(current, desired) {
		return false
	}
	*field = common.String(desired)
	return true
}

func setSensitiveDataModelSchemasForDiscovery(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if spec.SchemasForDiscovery == nil || reflect.DeepEqual(spec.SchemasForDiscovery, current.SchemasForDiscovery) {
		return false
	}
	body.SchemasForDiscovery = cloneSensitiveDataModelStringSlice(spec.SchemasForDiscovery)
	return true
}

func setSensitiveDataModelTablesForDiscovery(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	desired := sensitiveDataModelTablesForDiscoveryFromSpec(spec.TablesForDiscovery)
	if spec.TablesForDiscovery == nil || sensitiveDataModelTablesForDiscoveryEqual(desired, current.TablesForDiscovery) {
		return false
	}
	body.TablesForDiscovery = desired
	return true
}

func setSensitiveDataModelSensitiveTypeIDs(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if spec.SensitiveTypeIdsForDiscovery == nil ||
		reflect.DeepEqual(spec.SensitiveTypeIdsForDiscovery, current.SensitiveTypeIdsForDiscovery) {
		return false
	}
	body.SensitiveTypeIdsForDiscovery = cloneSensitiveDataModelStringSlice(spec.SensitiveTypeIdsForDiscovery)
	return true
}

func setSensitiveDataModelSensitiveTypeGroupIDs(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if spec.SensitiveTypeGroupIdsForDiscovery == nil ||
		reflect.DeepEqual(spec.SensitiveTypeGroupIdsForDiscovery, current.SensitiveTypeGroupIdsForDiscovery) {
		return false
	}
	body.SensitiveTypeGroupIdsForDiscovery = cloneSensitiveDataModelStringSlice(spec.SensitiveTypeGroupIdsForDiscovery)
	return true
}

func setSensitiveDataModelSampleDataCollection(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if sensitiveDataModelBoolPtrValue(current.IsSampleDataCollectionEnabled) == spec.IsSampleDataCollectionEnabled {
		return false
	}
	body.IsSampleDataCollectionEnabled = common.Bool(spec.IsSampleDataCollectionEnabled)
	return true
}

func setSensitiveDataModelAppDefinedRelationDiscovery(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if sensitiveDataModelBoolPtrValue(current.IsAppDefinedRelationDiscoveryEnabled) == spec.IsAppDefinedRelationDiscoveryEnabled {
		return false
	}
	body.IsAppDefinedRelationDiscoveryEnabled = common.Bool(spec.IsAppDefinedRelationDiscoveryEnabled)
	return true
}

func setSensitiveDataModelFreeformTags(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	body.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func setSensitiveDataModelDefinedTags(
	body *datasafesdk.UpdateSensitiveDataModelDetails,
	current datasafesdk.SensitiveDataModel,
	spec datasafev1beta1.SensitiveDataModelSpec,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := sensitiveDataModelDefinedTagsFromSpec(spec.DefinedTags)
	if sensitiveDataModelJSONEqual(desired, current.DefinedTags) {
		return false
	}
	body.DefinedTags = desired
	return true
}

func validateSensitiveDataModelSpec(spec datasafev1beta1.SensitiveDataModelSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.TargetId) == "" {
		missing = append(missing, "targetId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", sensitiveDataModelKind, strings.Join(missing, ", "))
}

func validateSensitiveDataModelCreateOnlyDriftForResponse(
	resource *datasafev1beta1.SensitiveDataModel,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", sensitiveDataModelKind)
	}
	current, ok := sensitiveDataModelFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateSensitiveDataModelCreateOnlyDrift(resource.Spec, current)
}

func validateSensitiveDataModelCreateOnlyDrift(
	spec datasafev1beta1.SensitiveDataModelSpec,
	current datasafesdk.SensitiveDataModel,
) error {
	var drift []string
	if !sensitiveDataModelStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if sensitiveDataModelBoolPtrValue(current.IsIncludeAllSchemas) != spec.IsIncludeAllSchemas {
		drift = append(drift, "isIncludeAllSchemas")
	}
	if sensitiveDataModelBoolPtrValue(current.IsIncludeAllSensitiveTypes) != spec.IsIncludeAllSensitiveTypes {
		drift = append(drift, "isIncludeAllSensitiveTypes")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only drift detected for %s; replace the resource or restore the desired spec before update", sensitiveDataModelKind, strings.Join(drift, ", "))
}

func guardSensitiveDataModelExistingBeforeCreate(
	_ context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil || strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
}

func wrapSensitiveDataModelMutationRecorders(hooks *SensitiveDataModelRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Update.Call != nil {
		update := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			response, err := update(ctx, request)
			if err == nil {
				recordSensitiveDataModelMutation(ctx, shared.OSOKAsyncPhaseUpdate, response.OpcWorkRequestId, response.OpcRequestId)
			}
			return response, err
		}
	}
	if hooks.Delete.Call != nil {
		deleteCall := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
			response, err := deleteCall(ctx, request)
			if err == nil {
				recordSensitiveDataModelMutation(ctx, shared.OSOKAsyncPhaseDelete, response.OpcWorkRequestId, response.OpcRequestId)
			}
			return response, err
		}
	}
}

func recordSensitiveDataModelMutation(ctx context.Context, phase shared.OSOKAsyncPhase, workRequestID *string, opcRequestID *string) {
	recorder, _ := ctx.Value(sensitiveDataModelMutationRecorderKey{}).(*sensitiveDataModelMutationRecorder)
	if recorder == nil {
		return
	}
	if id := sensitiveDataModelStringValue(workRequestID); id != "" {
		recorder.phase = phase
		recorder.workRequestID = id
	}
	if requestID := sensitiveDataModelStringValue(opcRequestID); requestID != "" {
		recorder.opcRequestID = requestID
	}
}

func listSensitiveDataModelsAllPages(
	call func(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error),
) func(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error) {
		if call == nil {
			return datasafesdk.ListSensitiveDataModelsResponse{}, fmt.Errorf("%s list operation is not configured", sensitiveDataModelKind)
		}
		return collectSensitiveDataModelPages(ctx, call, request)
	}
}

func collectSensitiveDataModelPages(
	ctx context.Context,
	call func(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error),
	request datasafesdk.ListSensitiveDataModelsRequest,
) (datasafesdk.ListSensitiveDataModelsResponse, error) {
	var combined datasafesdk.ListSensitiveDataModelsResponse
	seenPages := map[string]struct{}{}
	for {
		response, err := call(ctx, request)
		if err != nil {
			return datasafesdk.ListSensitiveDataModelsResponse{}, err
		}
		appendSensitiveDataModelListPage(&combined, response)
		nextPage, ok, err := nextSensitiveDataModelListPage(response, seenPages)
		if err != nil {
			return datasafesdk.ListSensitiveDataModelsResponse{}, err
		}
		if !ok {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = common.String(nextPage)
		combined.OpcNextPage = response.OpcNextPage
	}
}

func appendSensitiveDataModelListPage(
	combined *datasafesdk.ListSensitiveDataModelsResponse,
	response datasafesdk.ListSensitiveDataModelsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func nextSensitiveDataModelListPage(
	response datasafesdk.ListSensitiveDataModelsResponse,
	seenPages map[string]struct{},
) (string, bool, error) {
	nextPage := strings.TrimSpace(sensitiveDataModelStringValue(response.OpcNextPage))
	if nextPage == "" {
		return "", false, nil
	}
	if _, ok := seenPages[nextPage]; ok {
		return "", false, fmt.Errorf("%s list pagination repeated page token %q", sensitiveDataModelKind, nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nextPage, true, nil
}

func projectSensitiveDataModelStatus(resource *datasafev1beta1.SensitiveDataModel, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", sensitiveDataModelKind)
	}
	if current, ok := sensitiveDataModelFromResponse(response); ok {
		projectSensitiveDataModelSDKStatus(resource, current)
		return nil
	}
	if summary, ok := sensitiveDataModelSummaryFromResponse(response); ok {
		projectSensitiveDataModelSummaryStatus(resource, summary)
	}
	return nil
}

func projectSensitiveDataModelSDKStatus(
	resource *datasafev1beta1.SensitiveDataModel,
	current datasafesdk.SensitiveDataModel,
) {
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.SensitiveDataModelStatus{
		OsokStatus:                           osokStatus,
		Id:                                   sensitiveDataModelStringValue(current.Id),
		DisplayName:                          sensitiveDataModelStringValue(current.DisplayName),
		CompartmentId:                        sensitiveDataModelStringValue(current.CompartmentId),
		TargetId:                             sensitiveDataModelStringValue(current.TargetId),
		TimeCreated:                          sensitiveDataModelTimeString(current.TimeCreated),
		TimeUpdated:                          sensitiveDataModelTimeString(current.TimeUpdated),
		LifecycleState:                       string(current.LifecycleState),
		AppSuiteName:                         sensitiveDataModelStringValue(current.AppSuiteName),
		IsSampleDataCollectionEnabled:        sensitiveDataModelBoolPtrValue(current.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: sensitiveDataModelBoolPtrValue(current.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  sensitiveDataModelBoolPtrValue(current.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           sensitiveDataModelBoolPtrValue(current.IsIncludeAllSensitiveTypes),
		Description:                          sensitiveDataModelStringValue(current.Description),
		SchemasForDiscovery:                  cloneSensitiveDataModelStringSlice(current.SchemasForDiscovery),
		TablesForDiscovery:                   sensitiveDataModelTablesForDiscoveryToStatus(current.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         cloneSensitiveDataModelStringSlice(current.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    cloneSensitiveDataModelStringSlice(current.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         maps.Clone(current.FreeformTags),
		DefinedTags:                          sensitiveDataModelTagsFromSDK(current.DefinedTags),
		SystemTags:                           sensitiveDataModelTagsFromSDK(current.SystemTags),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func projectSensitiveDataModelSummaryStatus(
	resource *datasafev1beta1.SensitiveDataModel,
	current datasafesdk.SensitiveDataModelSummary,
) {
	status := &resource.Status
	status.Id = sensitiveDataModelStringValue(current.Id)
	status.DisplayName = sensitiveDataModelStringValue(current.DisplayName)
	status.CompartmentId = sensitiveDataModelStringValue(current.CompartmentId)
	status.TargetId = sensitiveDataModelStringValue(current.TargetId)
	status.TimeCreated = sensitiveDataModelTimeString(current.TimeCreated)
	status.TimeUpdated = sensitiveDataModelTimeString(current.TimeUpdated)
	status.LifecycleState = string(current.LifecycleState)
	status.AppSuiteName = sensitiveDataModelStringValue(current.AppSuiteName)
	status.Description = sensitiveDataModelStringValue(current.Description)
	status.FreeformTags = maps.Clone(current.FreeformTags)
	status.DefinedTags = sensitiveDataModelTagsFromSDK(current.DefinedTags)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
}

func sensitiveDataModelFromResponse(response any) (datasafesdk.SensitiveDataModel, bool) {
	switch current := sensitiveDataModelDereference(response).(type) {
	case datasafesdk.SensitiveDataModel:
		return current, true
	case datasafesdk.CreateSensitiveDataModelResponse:
		return current.SensitiveDataModel, true
	case datasafesdk.GetSensitiveDataModelResponse:
		return current.SensitiveDataModel, true
	case map[string]any:
		return sensitiveDataModelFromStatusMap(current)
	default:
		return datasafesdk.SensitiveDataModel{}, false
	}
}

func sensitiveDataModelSummaryFromResponse(response any) (datasafesdk.SensitiveDataModelSummary, bool) {
	switch current := sensitiveDataModelDereference(response).(type) {
	case datasafesdk.SensitiveDataModelSummary:
		return current, true
	default:
		return datasafesdk.SensitiveDataModelSummary{}, false
	}
}

func sensitiveDataModelDereference(response any) any {
	value := reflect.ValueOf(response)
	if !value.IsValid() || value.Kind() != reflect.Pointer {
		return response
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}

func sensitiveDataModelFromStatusMap(values map[string]any) (datasafesdk.SensitiveDataModel, bool) {
	if len(values) == 0 {
		return datasafesdk.SensitiveDataModel{}, false
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return datasafesdk.SensitiveDataModel{}, false
	}
	var status datasafev1beta1.SensitiveDataModelStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return datasafesdk.SensitiveDataModel{}, false
	}
	return sensitiveDataModelFromStatus(status), true
}

func sensitiveDataModelFromStatus(status datasafev1beta1.SensitiveDataModelStatus) datasafesdk.SensitiveDataModel {
	return datasafesdk.SensitiveDataModel{
		Id:                                   common.String(status.Id),
		DisplayName:                          common.String(status.DisplayName),
		CompartmentId:                        common.String(status.CompartmentId),
		TargetId:                             common.String(status.TargetId),
		LifecycleState:                       datasafesdk.DiscoveryLifecycleStateEnum(status.LifecycleState),
		AppSuiteName:                         common.String(status.AppSuiteName),
		IsSampleDataCollectionEnabled:        common.Bool(status.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: common.Bool(status.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  common.Bool(status.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           common.Bool(status.IsIncludeAllSensitiveTypes),
		Description:                          common.String(status.Description),
		SchemasForDiscovery:                  cloneSensitiveDataModelStringSlice(status.SchemasForDiscovery),
		TablesForDiscovery:                   sensitiveDataModelTablesForDiscoveryFromSpec(status.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         cloneSensitiveDataModelStringSlice(status.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    cloneSensitiveDataModelStringSlice(status.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         maps.Clone(status.FreeformTags),
		DefinedTags:                          sensitiveDataModelDefinedTagsFromStatus(status.DefinedTags),
		SystemTags:                           sensitiveDataModelDefinedTagsFromStatus(status.SystemTags),
	}
}

func handleSensitiveDataModelDeleteError(resource *datasafev1beta1.SensitiveDataModel, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := sensitiveDataModelAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func applySensitiveDataModelDeleteOutcome(
	resource *datasafev1beta1.SensitiveDataModel,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := sensitiveDataModelLifecycleState(response)
	if state == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest &&
		strings.EqualFold(state, string(datasafesdk.DiscoveryLifecycleStateDeleting)) {
		markSensitiveDataModelTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markSensitiveDataModelTerminating(resource *datasafev1beta1.SensitiveDataModel, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	workRequestID := ""
	if current := status.Async.Current; current != nil && current.Phase == shared.OSOKAsyncPhaseDelete {
		workRequestID = strings.TrimSpace(current.WorkRequestID)
	}
	status.UpdatedAt = &now
	status.Message = "OCI resource delete is in progress"
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         status.Message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, loggerutil.OSOKLogger{})
}

func sensitiveDataModelLifecycleState(response any) string {
	if current, ok := sensitiveDataModelFromResponse(response); ok {
		return strings.ToUpper(string(current.LifecycleState))
	}
	if summary, ok := sensitiveDataModelSummaryFromResponse(response); ok {
		return strings.ToUpper(string(summary.LifecycleState))
	}
	return ""
}

func (c sensitiveDataModelRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", sensitiveDataModelKind)
	}
	pendingBefore := currentSensitiveDataModelWorkRequest(resource, shared.OSOKAsyncPhaseUpdate)
	if guarded, handled, err := c.guardSensitiveDataModelPendingUpdate(ctx, resource, pendingBefore); handled || err != nil {
		return guarded, err
	}
	recorder := &sensitiveDataModelMutationRecorder{}
	response, err := c.delegate.CreateOrUpdate(context.WithValue(ctx, sensitiveDataModelMutationRecorderKey{}, recorder), resource, req)
	if err != nil {
		return response, err
	}
	if guarded, handled := sensitiveDataModelStaleUpdateGuard(resource, response, pendingBefore, recorder); handled {
		return guarded, nil
	}
	return response, nil
}

func (c sensitiveDataModelRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.SensitiveDataModel) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", sensitiveDataModelKind)
	}
	pendingBefore := currentSensitiveDataModelWorkRequest(resource, shared.OSOKAsyncPhaseDelete)
	readback, readbackOK, err := c.rejectAuthShapedPreDeleteRead(ctx, resource)
	if err != nil {
		return false, err
	}
	if deleted, handled, err := sensitiveDataModelPendingDeleteGuard(resource, pendingBefore, readback, readbackOK); handled || err != nil {
		return deleted, err
	}
	recorder := &sensitiveDataModelMutationRecorder{}
	deleted, err := c.delegate.Delete(context.WithValue(ctx, sensitiveDataModelMutationRecorderKey{}, recorder), resource)
	return sensitiveDataModelDeleteGuardResult(resource, pendingBefore, recorder, deleted, err)
}

func (c sensitiveDataModelRuntimeClient) guardSensitiveDataModelPendingUpdate(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
	pendingBefore *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, bool, error) {
	if pendingBefore == nil {
		return servicemanager.OSOKResponse{}, false, nil
	}
	response, ok, err := c.readSensitiveDataModelByTrackedID(ctx, resource)
	if err != nil || !ok {
		return servicemanager.OSOKResponse{}, false, err
	}
	current, ok := sensitiveDataModelFromResponse(response)
	if !ok || !strings.EqualFold(string(current.LifecycleState), string(datasafesdk.DiscoveryLifecycleStateActive)) {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if err := validateSensitiveDataModelSpec(resource.Spec); err != nil {
		return servicemanager.OSOKResponse{}, false, err
	}
	if err := validateSensitiveDataModelCreateOnlyDrift(resource.Spec, current); err != nil {
		return servicemanager.OSOKResponse{}, false, err
	}
	if _, updateNeeded := sensitiveDataModelMutableUpdateBody(resource.Spec, current); !updateNeeded {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if err := projectSensitiveDataModelStatus(resource, response); err != nil {
		return servicemanager.OSOKResponse{}, false, err
	}
	return markSensitiveDataModelWorkRequestPending(
		resource,
		pendingBefore.Phase,
		pendingBefore.WorkRequestID,
		"accepted update is not reflected by readback",
	), true, nil
}

func sensitiveDataModelStaleUpdateGuard(
	resource *datasafev1beta1.SensitiveDataModel,
	response servicemanager.OSOKResponse,
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveDataModelMutationRecorder,
) (servicemanager.OSOKResponse, bool) {
	if pendingBefore != nil && !response.ShouldRequeue && sensitiveDataModelDesiredUpdatePending(resource) {
		return markSensitiveDataModelWorkRequestPending(resource, pendingBefore.Phase, pendingBefore.WorkRequestID, "accepted update is not reflected by readback"), true
	}
	if recorder != nil &&
		recorder.phase == shared.OSOKAsyncPhaseUpdate &&
		recorder.workRequestID != "" &&
		!response.ShouldRequeue &&
		sensitiveDataModelDesiredUpdatePending(resource) {
		if recorder.opcRequestID != "" {
			resource.Status.OsokStatus.OpcRequestID = recorder.opcRequestID
		}
		return markSensitiveDataModelWorkRequestPending(resource, recorder.phase, recorder.workRequestID, "accepted update is not reflected by readback"), true
	}
	return servicemanager.OSOKResponse{}, false
}

func sensitiveDataModelDeleteGuardResult(
	resource *datasafev1beta1.SensitiveDataModel,
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveDataModelMutationRecorder,
	deleted bool,
	err error,
) (bool, error) {
	if err != nil {
		return sensitiveDataModelDeleteGuardError(resource, pendingBefore, recorder, deleted, err)
	}
	if shouldMarkSensitiveDataModelDeleteWorkRequest(resource, recorder, deleted) {
		if recorder.opcRequestID != "" {
			resource.Status.OsokStatus.OpcRequestID = recorder.opcRequestID
		}
		markSensitiveDataModelWorkRequestPending(resource, recorder.phase, recorder.workRequestID, "accepted delete is not reflected by readback")
	}
	return deleted, nil
}

func sensitiveDataModelDeleteGuardError(
	resource *datasafev1beta1.SensitiveDataModel,
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveDataModelMutationRecorder,
	deleted bool,
	err error,
) (bool, error) {
	workRequestID := sensitiveDataModelDeleteWorkRequestID(pendingBefore, recorder)
	if workRequestID != "" && sensitiveDataModelDeleteReadbackStillActive(err) {
		markSensitiveDataModelWorkRequestPending(resource, shared.OSOKAsyncPhaseDelete, workRequestID, "accepted delete is not reflected by readback")
		return false, nil
	}
	return deleted, err
}

func shouldMarkSensitiveDataModelDeleteWorkRequest(
	resource *datasafev1beta1.SensitiveDataModel,
	recorder *sensitiveDataModelMutationRecorder,
	deleted bool,
) bool {
	if deleted || recorder == nil || recorder.phase != shared.OSOKAsyncPhaseDelete || recorder.workRequestID == "" {
		return false
	}
	current := currentSensitiveDataModelWorkRequest(resource, recorder.phase)
	return current == nil || strings.TrimSpace(current.WorkRequestID) != recorder.workRequestID
}

func (c sensitiveDataModelRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
) (any, bool, error) {
	if resource == nil {
		return nil, false, nil
	}
	currentID := trackedSensitiveDataModelID(resource)
	if currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c sensitiveDataModelRuntimeClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
	currentID string,
) (any, bool, error) {
	if c.get == nil {
		return nil, false, nil
	}
	response, err := c.get(ctx, datasafesdk.GetSensitiveDataModelRequest{SensitiveDataModelId: common.String(currentID)})
	if ambiguous := sensitiveDataModelAmbiguousDeleteError(resource, err, "pre-delete get"); ambiguous != nil {
		return nil, false, ambiguous
	}
	if err != nil {
		return nil, false, nil
	}
	return response, true, nil
}

func (c sensitiveDataModelRuntimeClient) rejectAuthShapedList(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
) (any, bool, error) {
	if c.list == nil || strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, false, nil
	}
	response, err := c.list(ctx, datasafesdk.ListSensitiveDataModelsRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   sensitiveDataModelOptionalString(resource.Spec.DisplayName),
		TargetId:      common.String(strings.TrimSpace(resource.Spec.TargetId)),
	})
	if ambiguous := sensitiveDataModelAmbiguousDeleteError(resource, err, "pre-delete list"); ambiguous != nil {
		return nil, false, ambiguous
	}
	if err != nil {
		return nil, false, nil
	}
	return response, true, nil
}

func (c sensitiveDataModelRuntimeClient) readSensitiveDataModelByTrackedID(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveDataModel,
) (datasafesdk.GetSensitiveDataModelResponse, bool, error) {
	if c.get == nil {
		return datasafesdk.GetSensitiveDataModelResponse{}, false, nil
	}
	currentID := trackedSensitiveDataModelID(resource)
	if currentID == "" {
		return datasafesdk.GetSensitiveDataModelResponse{}, false, nil
	}
	response, err := c.get(ctx, datasafesdk.GetSensitiveDataModelRequest{SensitiveDataModelId: common.String(currentID)})
	if err != nil {
		return datasafesdk.GetSensitiveDataModelResponse{}, false, err
	}
	return response, true, nil
}

func sensitiveDataModelPendingDeleteGuard(
	resource *datasafev1beta1.SensitiveDataModel,
	pendingBefore *shared.OSOKAsyncOperation,
	response any,
	readbackOK bool,
) (bool, bool, error) {
	if pendingBefore == nil || !readbackOK {
		return false, false, nil
	}
	state := sensitiveDataModelLifecycleState(response)
	if !strings.EqualFold(state, string(datasafesdk.DiscoveryLifecycleStateActive)) {
		return false, false, nil
	}
	if err := projectSensitiveDataModelStatus(resource, response); err != nil {
		return false, false, err
	}
	markSensitiveDataModelWorkRequestPending(
		resource,
		pendingBefore.Phase,
		pendingBefore.WorkRequestID,
		"accepted delete is not reflected by readback",
	)
	return false, true, nil
}

func sensitiveDataModelAmbiguousDeleteError(
	resource *datasafev1beta1.SensitiveDataModel,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return sensitiveDataModelAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", sensitiveDataModelKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func clearTrackedSensitiveDataModelIdentity(resource *datasafev1beta1.SensitiveDataModel) {
	if resource == nil {
		return
	}
	status := resource.Status.OsokStatus
	status.Ocid = ""
	resource.Status = datasafev1beta1.SensitiveDataModelStatus{OsokStatus: status}
}

func trackedSensitiveDataModelID(resource *datasafev1beta1.SensitiveDataModel) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentSensitiveDataModelWorkRequest(
	resource *datasafev1beta1.SensitiveDataModel,
	phase shared.OSOKAsyncPhase,
) *shared.OSOKAsyncOperation {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return nil
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != "" &&
		current.Source != shared.OSOKAsyncSourceWorkRequest &&
		current.Source != shared.OSOKAsyncSourceLifecycle {
		return nil
	}
	if current.Phase != phase || strings.TrimSpace(current.WorkRequestID) == "" {
		return nil
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassSucceeded, shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled:
		return nil
	default:
		return current.DeepCopy()
	}
}

func sensitiveDataModelDesiredUpdatePending(resource *datasafev1beta1.SensitiveDataModel) bool {
	if resource == nil {
		return false
	}
	_, updateNeeded := sensitiveDataModelMutableUpdateBody(resource.Spec, sensitiveDataModelFromStatus(resource.Status))
	return updateNeeded
}

func sensitiveDataModelDeleteWorkRequestID(
	pendingBefore *shared.OSOKAsyncOperation,
	recorder *sensitiveDataModelMutationRecorder,
) string {
	if recorder != nil && recorder.phase == shared.OSOKAsyncPhaseDelete {
		if workRequestID := strings.TrimSpace(recorder.workRequestID); workRequestID != "" {
			return workRequestID
		}
	}
	if pendingBefore != nil {
		return strings.TrimSpace(pendingBefore.WorkRequestID)
	}
	return ""
}

func sensitiveDataModelDeleteReadbackStillActive(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "delete confirmation returned unexpected lifecycle state") &&
		strings.Contains(message, string(datasafesdk.DiscoveryLifecycleStateActive))
}

func markSensitiveDataModelWorkRequestPending(
	resource *datasafev1beta1.SensitiveDataModel,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	message string,
) servicemanager.OSOKResponse {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	current := &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            phase,
		WorkRequestID:    workRequestID,
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawOperationType: sensitiveDataModelWorkRequestOperationType(phase),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          sensitiveDataModelWorkRequestPendingMessage(phase, workRequestID, message),
		UpdatedAt:        &now,
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}
}

func sensitiveDataModelWorkRequestOperationType(phase shared.OSOKAsyncPhase) string {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return string(datasafesdk.WorkRequestOperationTypeCreateSensitiveDataModel)
	case shared.OSOKAsyncPhaseUpdate:
		return string(datasafesdk.WorkRequestOperationTypeUpdateSensitiveDataModel)
	case shared.OSOKAsyncPhaseDelete:
		return string(datasafesdk.WorkRequestOperationTypeDeleteSensitiveDataModel)
	default:
		return ""
	}
}

func sensitiveDataModelWorkRequestPendingMessage(phase shared.OSOKAsyncPhase, workRequestID string, message string) string {
	message = strings.TrimSpace(message)
	if message != "" {
		return fmt.Sprintf("%s; %s work request %s is pending", message, phase, workRequestID)
	}
	return fmt.Sprintf("%s work request %s is pending", phase, workRequestID)
}

func sensitiveDataModelOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func sensitiveDataModelStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sensitiveDataModelStringPtrEqual(value *string, desired string) bool {
	return strings.TrimSpace(sensitiveDataModelStringValue(value)) == strings.TrimSpace(desired)
}

func sensitiveDataModelBoolPtrValue(value *bool) bool {
	return value != nil && *value
}

func sensitiveDataModelTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func cloneSensitiveDataModelStringSlice(source []string) []string {
	if source == nil {
		return nil
	}
	return append([]string(nil), source...)
}

func sensitiveDataModelTablesForDiscoveryFromSpec(
	source []datasafev1beta1.SensitiveDataModelTablesForDiscovery,
) []datasafesdk.TablesForDiscovery {
	if source == nil {
		return nil
	}
	converted := make([]datasafesdk.TablesForDiscovery, 0, len(source))
	for _, table := range source {
		converted = append(converted, datasafesdk.TablesForDiscovery{
			SchemaName: common.String(strings.TrimSpace(table.SchemaName)),
			TableNames: cloneSensitiveDataModelStringSlice(table.TableNames),
		})
	}
	return converted
}

func sensitiveDataModelTablesForDiscoveryToStatus(
	source []datasafesdk.TablesForDiscovery,
) []datasafev1beta1.SensitiveDataModelTablesForDiscovery {
	if source == nil {
		return nil
	}
	converted := make([]datasafev1beta1.SensitiveDataModelTablesForDiscovery, 0, len(source))
	for _, table := range source {
		converted = append(converted, datasafev1beta1.SensitiveDataModelTablesForDiscovery{
			SchemaName: sensitiveDataModelStringValue(table.SchemaName),
			TableNames: cloneSensitiveDataModelStringSlice(table.TableNames),
		})
	}
	return converted
}

func sensitiveDataModelTablesForDiscoveryEqual(left []datasafesdk.TablesForDiscovery, right []datasafesdk.TablesForDiscovery) bool {
	return reflect.DeepEqual(sensitiveDataModelTablesForDiscoveryToStatus(left), sensitiveDataModelTablesForDiscoveryToStatus(right))
}

func sensitiveDataModelDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func sensitiveDataModelDefinedTagsFromStatus(source map[string]shared.MapValue) map[string]map[string]interface{} {
	return sensitiveDataModelDefinedTagsFromSpec(source)
}

func sensitiveDataModelTagsFromSDK(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = fmt.Sprint(value)
		}
		converted[namespace] = children
	}
	return converted
}

func sensitiveDataModelJSONEqual(left any, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

var _ interface{ GetOpcRequestID() string } = sensitiveDataModelAmbiguousNotFoundError{}
