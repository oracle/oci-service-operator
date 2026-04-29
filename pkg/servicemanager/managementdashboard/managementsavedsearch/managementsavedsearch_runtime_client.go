/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementsavedsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementdashboardsdk "github.com/oracle/oci-go-sdk/v65/managementdashboard"
	managementdashboardv1beta1 "github.com/oracle/oci-service-operator/api/managementdashboard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type managementSavedSearchOCIClient interface {
	CreateManagementSavedSearch(context.Context, managementdashboardsdk.CreateManagementSavedSearchRequest) (managementdashboardsdk.CreateManagementSavedSearchResponse, error)
	GetManagementSavedSearch(context.Context, managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error)
	ListManagementSavedSearches(context.Context, managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error)
	UpdateManagementSavedSearch(context.Context, managementdashboardsdk.UpdateManagementSavedSearchRequest) (managementdashboardsdk.UpdateManagementSavedSearchResponse, error)
	DeleteManagementSavedSearch(context.Context, managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error)
}

type managementSavedSearchRuntimeClient struct {
	delegate ManagementSavedSearchServiceClient
	hooks    ManagementSavedSearchRuntimeHooks
}

var _ ManagementSavedSearchServiceClient = (*managementSavedSearchRuntimeClient)(nil)

func init() {
	registerManagementSavedSearchRuntimeHooksMutator(func(manager *ManagementSavedSearchServiceManager, hooks *ManagementSavedSearchRuntimeHooks) {
		applyManagementSavedSearchRuntimeHooks(manager, hooks)
	})
}

func applyManagementSavedSearchRuntimeHooks(_ *ManagementSavedSearchServiceManager, hooks *ManagementSavedSearchRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newManagementSavedSearchRuntimeSemantics()
	hooks.BuildCreateBody = buildManagementSavedSearchCreateBody
	hooks.BuildUpdateBody = buildManagementSavedSearchUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardManagementSavedSearchExistingBeforeCreate
	hooks.List.Fields = managementSavedSearchListFields()
	hooks.StatusHooks.ProjectStatus = projectManagementSavedSearchStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateManagementSavedSearchCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleManagementSavedSearchDeleteError
	wrapManagementSavedSearchReadAndDeleteCalls(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagementSavedSearchServiceClient) ManagementSavedSearchServiceClient {
		return &managementSavedSearchRuntimeClient{delegate: delegate, hooks: *hooks}
	})
}

func newManagementSavedSearchServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client managementSavedSearchOCIClient,
) ManagementSavedSearchServiceClient {
	hooks := newManagementSavedSearchRuntimeHooksWithOCIClient(client)
	applyManagementSavedSearchRuntimeHooks(&ManagementSavedSearchServiceManager{Log: log}, &hooks)
	manager := &ManagementSavedSearchServiceManager{Log: log}
	delegate := defaultManagementSavedSearchServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*managementdashboardv1beta1.ManagementSavedSearch](
			buildManagementSavedSearchGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapManagementSavedSearchGeneratedClient(hooks, delegate)
}

func newManagementSavedSearchRuntimeHooksWithOCIClient(client managementSavedSearchOCIClient) ManagementSavedSearchRuntimeHooks {
	return ManagementSavedSearchRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*managementdashboardv1beta1.ManagementSavedSearch]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*managementdashboardv1beta1.ManagementSavedSearch]{},
		StatusHooks:     generatedruntime.StatusHooks[*managementdashboardv1beta1.ManagementSavedSearch]{},
		ParityHooks:     generatedruntime.ParityHooks[*managementdashboardv1beta1.ManagementSavedSearch]{},
		Async:           generatedruntime.AsyncHooks[*managementdashboardv1beta1.ManagementSavedSearch]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*managementdashboardv1beta1.ManagementSavedSearch]{},
		Create: runtimeOperationHooks[managementdashboardsdk.CreateManagementSavedSearchRequest, managementdashboardsdk.CreateManagementSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateManagementSavedSearchDetails", RequestName: "CreateManagementSavedSearchDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request managementdashboardsdk.CreateManagementSavedSearchRequest) (managementdashboardsdk.CreateManagementSavedSearchResponse, error) {
				if client == nil {
					return managementdashboardsdk.CreateManagementSavedSearchResponse{}, fmt.Errorf("ManagementSavedSearch OCI client is nil")
				}
				return client.CreateManagementSavedSearch(ctx, request)
			},
		},
		Get: runtimeOperationHooks[managementdashboardsdk.GetManagementSavedSearchRequest, managementdashboardsdk.GetManagementSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ManagementSavedSearchId", RequestName: "managementSavedSearchId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
				if client == nil {
					return managementdashboardsdk.GetManagementSavedSearchResponse{}, fmt.Errorf("ManagementSavedSearch OCI client is nil")
				}
				return client.GetManagementSavedSearch(ctx, request)
			},
		},
		List: runtimeOperationHooks[managementdashboardsdk.ListManagementSavedSearchesRequest, managementdashboardsdk.ListManagementSavedSearchesResponse]{
			Fields: managementSavedSearchListFields(),
			Call: func(ctx context.Context, request managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
				if client == nil {
					return managementdashboardsdk.ListManagementSavedSearchesResponse{}, fmt.Errorf("ManagementSavedSearch OCI client is nil")
				}
				return client.ListManagementSavedSearches(ctx, request)
			},
		},
		Update: runtimeOperationHooks[managementdashboardsdk.UpdateManagementSavedSearchRequest, managementdashboardsdk.UpdateManagementSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ManagementSavedSearchId", RequestName: "managementSavedSearchId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateManagementSavedSearchDetails", RequestName: "UpdateManagementSavedSearchDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request managementdashboardsdk.UpdateManagementSavedSearchRequest) (managementdashboardsdk.UpdateManagementSavedSearchResponse, error) {
				if client == nil {
					return managementdashboardsdk.UpdateManagementSavedSearchResponse{}, fmt.Errorf("ManagementSavedSearch OCI client is nil")
				}
				return client.UpdateManagementSavedSearch(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[managementdashboardsdk.DeleteManagementSavedSearchRequest, managementdashboardsdk.DeleteManagementSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ManagementSavedSearchId", RequestName: "managementSavedSearchId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error) {
				if client == nil {
					return managementdashboardsdk.DeleteManagementSavedSearchResponse{}, fmt.Errorf("ManagementSavedSearch OCI client is nil")
				}
				return client.DeleteManagementSavedSearch(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ManagementSavedSearchServiceClient) ManagementSavedSearchServiceClient{},
	}
}

func newManagementSavedSearchRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "managementdashboard",
		FormalSlug:        "managementsavedsearch",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(managementdashboardsdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "best-effort",
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "providerId", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"displayName", "providerId", "providerVersion", "providerName", "description",
				"nls", "type", "uiConfig", "dataConfig", "screenImage", "metadataVersion",
				"widgetTemplate", "widgetVM", "parametersConfig", "featuresConfig",
				"drilldownConfig", "freeformTags", "definedTags",
			},
			Mutable: []string{
				"displayName", "providerId", "providerVersion", "providerName", "description",
				"nls", "type", "uiConfig", "dataConfig", "screenImage", "metadataVersion",
				"widgetTemplate", "widgetVM", "parametersConfig", "featuresConfig",
				"drilldownConfig", "freeformTags", "definedTags",
			},
			ForceNew:      []string{"compartmentId", "id", "isOobSavedSearch"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ManagementSavedSearch", Action: "CreateManagementSavedSearch"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ManagementSavedSearch", Action: "UpdateManagementSavedSearch"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ManagementSavedSearch", Action: "DeleteManagementSavedSearch"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func managementSavedSearchListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
	}
}

func guardManagementSavedSearchExistingBeforeCreate(
	_ context.Context,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ManagementSavedSearch resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildManagementSavedSearchCreateBody(
	_ context.Context,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ManagementSavedSearch resource is nil")
	}

	spec := resource.Spec
	nls, err := requiredJSONInterfacePointer("nls", spec.Nls)
	if err != nil {
		return nil, err
	}
	uiConfig, err := requiredJSONInterfacePointer("uiConfig", spec.UiConfig)
	if err != nil {
		return nil, err
	}
	dataConfig, err := jsonInterfaceSlice("dataConfig", spec.DataConfig)
	if err != nil {
		return nil, err
	}
	featuresConfig, err := optionalJSONInterfacePointer("featuresConfig", spec.FeaturesConfig)
	if err != nil {
		return nil, err
	}
	parametersConfig, err := jsonInterfaceSlice("parametersConfig", spec.ParametersConfig)
	if err != nil {
		return nil, err
	}
	drilldownConfig, err := jsonInterfaceSlice("drilldownConfig", spec.DrilldownConfig)
	if err != nil {
		return nil, err
	}

	details := managementdashboardsdk.CreateManagementSavedSearchDetails{
		DisplayName:      stringPointer(strings.TrimSpace(spec.DisplayName)),
		ProviderId:       stringPointer(strings.TrimSpace(spec.ProviderId)),
		ProviderVersion:  stringPointer(strings.TrimSpace(spec.ProviderVersion)),
		ProviderName:     stringPointer(strings.TrimSpace(spec.ProviderName)),
		CompartmentId:    stringPointer(strings.TrimSpace(spec.CompartmentId)),
		IsOobSavedSearch: boolPointer(spec.IsOobSavedSearch),
		Description:      stringPointer(strings.TrimSpace(spec.Description)),
		Nls:              nls,
		Type:             managementdashboardsdk.SavedSearchTypesEnum(strings.TrimSpace(spec.Type)),
		UiConfig:         uiConfig,
		DataConfig:       dataConfig,
		ScreenImage:      stringPointer(strings.TrimSpace(spec.ScreenImage)),
		MetadataVersion:  stringPointer(strings.TrimSpace(spec.MetadataVersion)),
		WidgetTemplate:   stringPointer(strings.TrimSpace(spec.WidgetTemplate)),
		WidgetVM:         stringPointer(strings.TrimSpace(spec.WidgetVM)),
		ParametersConfig: parametersConfig,
		FeaturesConfig:   featuresConfig,
		DrilldownConfig:  drilldownConfig,
		FreeformTags:     cloneStringMap(spec.FreeformTags),
		DefinedTags:      definedTagsFromSpec(spec.DefinedTags),
	}
	if id := strings.TrimSpace(spec.Id); id != "" {
		details.Id = stringPointer(id)
	}
	return details, nil
}

func buildManagementSavedSearchUpdateBody(
	_ context.Context,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return managementdashboardsdk.UpdateManagementSavedSearchDetails{}, false, fmt.Errorf("ManagementSavedSearch resource is nil")
	}
	current, ok := managementSavedSearchFromResponse(currentResponse)
	if !ok {
		return managementdashboardsdk.UpdateManagementSavedSearchDetails{}, false, fmt.Errorf("current ManagementSavedSearch response does not expose a ManagementSavedSearch body")
	}
	if err := validateManagementSavedSearchCreateOnlyDrift(resource.Spec, current); err != nil {
		return managementdashboardsdk.UpdateManagementSavedSearchDetails{}, false, err
	}

	details := managementdashboardsdk.UpdateManagementSavedSearchDetails{}
	updateNeeded := applyManagementSavedSearchScalarUpdates(resource.Spec, current, &details)
	jsonUpdated, err := applyManagementSavedSearchJSONUpdates(resource.Spec, current, &details)
	if err != nil {
		return managementdashboardsdk.UpdateManagementSavedSearchDetails{}, false, err
	}
	updateNeeded = applyManagementSavedSearchTagUpdates(resource.Spec, current, &details) || jsonUpdated || updateNeeded
	if updateNeeded && managementSavedSearchIsOOB(resource.Spec, current) {
		return managementdashboardsdk.UpdateManagementSavedSearchDetails{}, false, fmt.Errorf("ManagementSavedSearch isOobSavedSearch resources cannot be modified")
	}
	return details, updateNeeded, nil
}

func applyManagementSavedSearchScalarUpdates(
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	current managementdashboardsdk.ManagementSavedSearch,
	details *managementdashboardsdk.UpdateManagementSavedSearchDetails,
) bool {
	stringFields := []struct {
		target  **string
		desired string
		current *string
	}{
		{target: &details.DisplayName, desired: spec.DisplayName, current: current.DisplayName},
		{target: &details.ProviderId, desired: spec.ProviderId, current: current.ProviderId},
		{target: &details.ProviderVersion, desired: spec.ProviderVersion, current: current.ProviderVersion},
		{target: &details.ProviderName, desired: spec.ProviderName, current: current.ProviderName},
		{target: &details.Description, desired: spec.Description, current: current.Description},
		{target: &details.ScreenImage, desired: spec.ScreenImage, current: current.ScreenImage},
		{target: &details.MetadataVersion, desired: spec.MetadataVersion, current: current.MetadataVersion},
		{target: &details.WidgetTemplate, desired: spec.WidgetTemplate, current: current.WidgetTemplate},
		{target: &details.WidgetVM, desired: spec.WidgetVM, current: current.WidgetVM},
	}

	updateNeeded := false
	for _, field := range stringFields {
		updateNeeded = setStringUpdate(field.target, field.desired, field.current) || updateNeeded
	}
	return setSavedSearchTypeUpdate(spec.Type, current.Type, details) || updateNeeded
}

func applyManagementSavedSearchJSONUpdates(
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	current managementdashboardsdk.ManagementSavedSearch,
	details *managementdashboardsdk.UpdateManagementSavedSearchDetails,
) (bool, error) {
	jsonFields := []struct {
		name    string
		desired shared.JSONValue
		current *interface{}
		target  **interface{}
	}{
		{name: "nls", desired: spec.Nls, current: current.Nls, target: &details.Nls},
		{name: "uiConfig", desired: spec.UiConfig, current: current.UiConfig, target: &details.UiConfig},
		{name: "featuresConfig", desired: spec.FeaturesConfig, current: current.FeaturesConfig, target: &details.FeaturesConfig},
	}
	sliceFields := []struct {
		name    string
		desired []shared.JSONValue
		current []interface{}
		target  *[]interface{}
	}{
		{name: "dataConfig", desired: spec.DataConfig, current: current.DataConfig, target: &details.DataConfig},
		{name: "parametersConfig", desired: spec.ParametersConfig, current: current.ParametersConfig, target: &details.ParametersConfig},
		{name: "drilldownConfig", desired: spec.DrilldownConfig, current: current.DrilldownConfig, target: &details.DrilldownConfig},
	}

	updateNeeded := false
	for _, field := range jsonFields {
		updated, err := setJSONPointerUpdate(field.name, field.desired, field.current, field.target)
		if err != nil {
			return false, err
		}
		updateNeeded = updated || updateNeeded
	}
	for _, field := range sliceFields {
		updated, err := setJSONSliceUpdate(field.name, field.desired, field.current, field.target)
		if err != nil {
			return false, err
		}
		updateNeeded = updated || updateNeeded
	}
	return updateNeeded, nil
}

func applyManagementSavedSearchTagUpdates(
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	current managementdashboardsdk.ManagementSavedSearch,
	details *managementdashboardsdk.UpdateManagementSavedSearchDetails,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := definedTagsFromSpec(spec.DefinedTags)
		if !reflect.DeepEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	return updateNeeded
}

func managementSavedSearchIsOOB(
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	current managementdashboardsdk.ManagementSavedSearch,
) bool {
	return spec.IsOobSavedSearch || boolPointerValue(current.IsOobSavedSearch)
}

func validateManagementSavedSearchCreateOnlyDriftForResponse(
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("ManagementSavedSearch resource is nil")
	}
	current, ok := managementSavedSearchFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateManagementSavedSearchCreateOnlyDrift(resource.Spec, current)
}

func validateManagementSavedSearchCreateOnlyDrift(
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	current managementdashboardsdk.ManagementSavedSearch,
) error {
	var drift []string
	if desired := strings.TrimSpace(spec.CompartmentId); desired != "" && desired != stringPointerValue(current.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if desired := strings.TrimSpace(spec.Id); desired != "" && desired != stringPointerValue(current.Id) {
		drift = append(drift, "id")
	}
	if current.IsOobSavedSearch != nil && spec.IsOobSavedSearch != boolPointerValue(current.IsOobSavedSearch) {
		drift = append(drift, "isOobSavedSearch")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("ManagementSavedSearch create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func (c *managementSavedSearchRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ManagementSavedSearch runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *managementSavedSearchRuntimeClient) Delete(
	ctx context.Context,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("ManagementSavedSearch runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *managementSavedSearchRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
) error {
	currentID := currentManagementSavedSearchID(resource)
	if currentID == "" || c.hooks.Get.Call == nil {
		return nil
	}
	_, err := c.hooks.Get.Call(ctx, managementdashboardsdk.GetManagementSavedSearchRequest{ManagementSavedSearchId: stringPointer(currentID)})
	if err == nil || !isManagementSavedSearchAmbiguousNotFound(err) {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("ManagementSavedSearch delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
}

func currentManagementSavedSearchID(resource *managementdashboardv1beta1.ManagementSavedSearch) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func wrapManagementSavedSearchReadAndDeleteCalls(hooks *ManagementSavedSearchRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeManagementSavedSearchNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
			return listManagementSavedSearchPages(ctx, call, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeManagementSavedSearchNotFoundError(err, "delete")
		}
	}
}

func listManagementSavedSearchPages(
	ctx context.Context,
	call func(context.Context, managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error),
	request managementdashboardsdk.ListManagementSavedSearchesRequest,
) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
	var combined managementdashboardsdk.ListManagementSavedSearchesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return managementdashboardsdk.ListManagementSavedSearchesResponse{}, conservativeManagementSavedSearchNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleManagementSavedSearchDeleteError(resource *managementdashboardv1beta1.ManagementSavedSearch, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeManagementSavedSearchNotFoundError(err, "delete")
}

type managementSavedSearchAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e managementSavedSearchAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e managementSavedSearchAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func conservativeManagementSavedSearchNotFoundError(err error, operation string) error {
	if err == nil || isManagementSavedSearchAmbiguousNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return managementSavedSearchAmbiguousNotFoundError{
		message:      fmt.Sprintf("ManagementSavedSearch %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isManagementSavedSearchAmbiguousNotFound(err error) bool {
	var ambiguous managementSavedSearchAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func projectManagementSavedSearchStatus(
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("ManagementSavedSearch resource is nil")
	}
	current, ok := managementSavedSearchFromResponse(response)
	if !ok {
		return nil
	}

	osokStatus := resource.Status.OsokStatus
	resource.Status = managementdashboardv1beta1.ManagementSavedSearchStatus{
		OsokStatus:       osokStatus,
		Id:               stringPointerValue(current.Id),
		DisplayName:      stringPointerValue(current.DisplayName),
		ProviderId:       stringPointerValue(current.ProviderId),
		ProviderVersion:  stringPointerValue(current.ProviderVersion),
		ProviderName:     stringPointerValue(current.ProviderName),
		CompartmentId:    stringPointerValue(current.CompartmentId),
		IsOobSavedSearch: boolPointerValue(current.IsOobSavedSearch),
		Description:      stringPointerValue(current.Description),
		Nls:              jsonValueFromInterfacePointer(current.Nls),
		Type:             string(current.Type),
		UiConfig:         jsonValueFromInterfacePointer(current.UiConfig),
		DataConfig:       jsonValueSliceFromInterfaces(current.DataConfig),
		CreatedBy:        stringPointerValue(current.CreatedBy),
		UpdatedBy:        stringPointerValue(current.UpdatedBy),
		TimeCreated:      sdkTimeString(current.TimeCreated),
		TimeUpdated:      sdkTimeString(current.TimeUpdated),
		ScreenImage:      stringPointerValue(current.ScreenImage),
		MetadataVersion:  stringPointerValue(current.MetadataVersion),
		WidgetTemplate:   stringPointerValue(current.WidgetTemplate),
		WidgetVM:         stringPointerValue(current.WidgetVM),
		LifecycleState:   string(current.LifecycleState),
		ParametersConfig: jsonValueSliceFromInterfaces(current.ParametersConfig),
		FeaturesConfig:   jsonValueFromInterfacePointer(current.FeaturesConfig),
		DrilldownConfig:  jsonValueSliceFromInterfaces(current.DrilldownConfig),
		FreeformTags:     cloneStringMap(current.FreeformTags),
		DefinedTags:      statusTagsFromSDK(current.DefinedTags),
		SystemTags:       statusTagsFromSDK(current.SystemTags),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func managementSavedSearchFromResponse(response any) (managementdashboardsdk.ManagementSavedSearch, bool) {
	if current, ok := managementSavedSearchFromDirectResponse(response); ok {
		return current, true
	}
	return managementSavedSearchFromOperationResponse(response)
}

func managementSavedSearchFromDirectResponse(response any) (managementdashboardsdk.ManagementSavedSearch, bool) {
	switch current := response.(type) {
	case managementdashboardsdk.ManagementSavedSearch:
		return current, true
	case *managementdashboardsdk.ManagementSavedSearch:
		if current == nil {
			return managementdashboardsdk.ManagementSavedSearch{}, false
		}
		return *current, true
	case managementdashboardsdk.ManagementSavedSearchSummary:
		return managementSavedSearchFromSummary(current), true
	case *managementdashboardsdk.ManagementSavedSearchSummary:
		if current == nil {
			return managementdashboardsdk.ManagementSavedSearch{}, false
		}
		return managementSavedSearchFromSummary(*current), true
	default:
		return managementdashboardsdk.ManagementSavedSearch{}, false
	}
}

func managementSavedSearchFromOperationResponse(response any) (managementdashboardsdk.ManagementSavedSearch, bool) {
	switch current := response.(type) {
	case managementdashboardsdk.CreateManagementSavedSearchResponse:
		return current.ManagementSavedSearch, true
	case *managementdashboardsdk.CreateManagementSavedSearchResponse:
		if current == nil {
			return managementdashboardsdk.ManagementSavedSearch{}, false
		}
		return current.ManagementSavedSearch, true
	case managementdashboardsdk.GetManagementSavedSearchResponse:
		return current.ManagementSavedSearch, true
	case *managementdashboardsdk.GetManagementSavedSearchResponse:
		if current == nil {
			return managementdashboardsdk.ManagementSavedSearch{}, false
		}
		return current.ManagementSavedSearch, true
	case managementdashboardsdk.UpdateManagementSavedSearchResponse:
		return current.ManagementSavedSearch, true
	case *managementdashboardsdk.UpdateManagementSavedSearchResponse:
		if current == nil {
			return managementdashboardsdk.ManagementSavedSearch{}, false
		}
		return current.ManagementSavedSearch, true
	default:
		return managementdashboardsdk.ManagementSavedSearch{}, false
	}
}

func managementSavedSearchFromSummary(summary managementdashboardsdk.ManagementSavedSearchSummary) managementdashboardsdk.ManagementSavedSearch {
	return managementdashboardsdk.ManagementSavedSearch{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		ProviderId:       summary.ProviderId,
		ProviderVersion:  summary.ProviderVersion,
		ProviderName:     summary.ProviderName,
		CompartmentId:    summary.CompartmentId,
		IsOobSavedSearch: summary.IsOobSavedSearch,
		Description:      summary.Description,
		Nls:              summary.Nls,
		Type:             summary.Type,
		UiConfig:         summary.UiConfig,
		DataConfig:       summary.DataConfig,
		CreatedBy:        summary.CreatedBy,
		UpdatedBy:        summary.UpdatedBy,
		TimeCreated:      summary.TimeCreated,
		TimeUpdated:      summary.TimeUpdated,
		ScreenImage:      summary.ScreenImage,
		MetadataVersion:  summary.MetadataVersion,
		WidgetTemplate:   summary.WidgetTemplate,
		WidgetVM:         summary.WidgetVM,
		LifecycleState:   summary.LifecycleState,
		ParametersConfig: summary.ParametersConfig,
		FeaturesConfig:   summary.FeaturesConfig,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		SystemTags:       summary.SystemTags,
	}
}

func setStringUpdate(target **string, desired string, current *string) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == stringPointerValue(current) {
		return false
	}
	*target = stringPointer(desired)
	return true
}

func setSavedSearchTypeUpdate(
	desired string,
	current managementdashboardsdk.SavedSearchTypesEnum,
	details *managementdashboardsdk.UpdateManagementSavedSearchDetails,
) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == string(current) {
		return false
	}
	details.Type = managementdashboardsdk.SavedSearchTypesEnum(desired)
	return true
}

func setJSONPointerUpdate(
	name string,
	desired shared.JSONValue,
	current *interface{},
	target **interface{},
) (bool, error) {
	if len(desired.Raw) == 0 {
		return false, nil
	}
	decoded, err := jsonInterface(name, desired)
	if err != nil {
		return false, err
	}
	if jsonEquivalent(decoded, interfacePointerValue(current)) {
		return false, nil
	}
	*target = &decoded
	return true, nil
}

func setJSONSliceUpdate(
	name string,
	desired []shared.JSONValue,
	current []interface{},
	target *[]interface{},
) (bool, error) {
	if len(desired) == 0 {
		return false, nil
	}
	decoded, err := jsonInterfaceSlice(name, desired)
	if err != nil {
		return false, err
	}
	if jsonEquivalent(decoded, current) {
		return false, nil
	}
	*target = decoded
	return true, nil
}

func requiredJSONInterfacePointer(name string, value shared.JSONValue) (*interface{}, error) {
	decoded, err := jsonInterface(name, value)
	if err != nil {
		return nil, err
	}
	return &decoded, nil
}

func optionalJSONInterfacePointer(name string, value shared.JSONValue) (*interface{}, error) {
	if len(value.Raw) == 0 {
		return nil, nil
	}
	decoded, err := jsonInterface(name, value)
	if err != nil {
		return nil, err
	}
	if decoded == nil {
		return nil, nil
	}
	return &decoded, nil
}

func jsonInterface(name string, value shared.JSONValue) (interface{}, error) {
	if len(value.Raw) == 0 {
		return nil, nil
	}
	var decoded interface{}
	if err := json.Unmarshal(value.Raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode ManagementSavedSearch %s JSON: %w", name, err)
	}
	return decoded, nil
}

func jsonInterfaceSlice(name string, values []shared.JSONValue) ([]interface{}, error) {
	if len(values) == 0 {
		return nil, nil
	}
	decoded := make([]interface{}, 0, len(values))
	for index, value := range values {
		item, err := jsonInterface(fmt.Sprintf("%s[%d]", name, index), value)
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, item)
	}
	return decoded, nil
}

func jsonValueFromInterfacePointer(value *interface{}) shared.JSONValue {
	return jsonValueFromInterface(interfacePointerValue(value))
}

func jsonValueSliceFromInterfaces(values []interface{}) []shared.JSONValue {
	if len(values) == 0 {
		return nil
	}
	converted := make([]shared.JSONValue, 0, len(values))
	for _, value := range values {
		converted = append(converted, jsonValueFromInterface(value))
	}
	return converted
}

func jsonValueFromInterface(value interface{}) shared.JSONValue {
	if value == nil {
		return shared.JSONValue{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return shared.JSONValue{}
	}
	return shared.JSONValue{Raw: payload}
}

func jsonEquivalent(left interface{}, right interface{}) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func interfacePointerValue(value *interface{}) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func definedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func statusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(tags) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			switch typed := value.(type) {
			case string:
				children[key] = typed
			default:
				children[key] = fmt.Sprint(typed)
			}
		}
		converted[namespace] = children
	}
	return converted
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	return maps.Clone(source)
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func stringPointer(value string) *string {
	return &value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolPointer(value bool) *bool {
	return &value
}

func boolPointerValue(value *bool) bool {
	return value != nil && *value
}

var _ error = managementSavedSearchAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = managementSavedSearchAmbiguousNotFoundError{}
