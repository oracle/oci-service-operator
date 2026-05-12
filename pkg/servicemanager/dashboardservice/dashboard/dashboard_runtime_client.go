/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	dashboardservicesdk "github.com/oracle/oci-go-sdk/v65/dashboardservice"
	dashboardservicev1beta1 "github.com/oracle/oci-service-operator/api/dashboardservice/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const dashboardSchemaVersionV1 = "V1"

type dashboardOCIClient interface {
	CreateDashboard(context.Context, dashboardservicesdk.CreateDashboardRequest) (dashboardservicesdk.CreateDashboardResponse, error)
	GetDashboard(context.Context, dashboardservicesdk.GetDashboardRequest) (dashboardservicesdk.GetDashboardResponse, error)
	ListDashboards(context.Context, dashboardservicesdk.ListDashboardsRequest) (dashboardservicesdk.ListDashboardsResponse, error)
	UpdateDashboard(context.Context, dashboardservicesdk.UpdateDashboardRequest) (dashboardservicesdk.UpdateDashboardResponse, error)
	DeleteDashboard(context.Context, dashboardservicesdk.DeleteDashboardRequest) (dashboardservicesdk.DeleteDashboardResponse, error)
}

type dashboardRuntimeState struct {
	Id               *string                                         `json:"id"`
	DashboardGroupId *string                                         `json:"dashboardGroupId"`
	DisplayName      *string                                         `json:"displayName"`
	Description      *string                                         `json:"description"`
	CompartmentId    *string                                         `json:"compartmentId"`
	TimeCreated      *common.SDKTime                                 `json:"timeCreated"`
	TimeUpdated      *common.SDKTime                                 `json:"timeUpdated"`
	LifecycleState   dashboardservicesdk.DashboardLifecycleStateEnum `json:"lifecycleState"`
	FreeformTags     map[string]string                               `json:"freeformTags"`
	DefinedTags      map[string]map[string]interface{}               `json:"definedTags"`
	SystemTags       map[string]map[string]interface{}               `json:"systemTags"`
	SchemaVersion    string                                          `json:"schemaVersion"`
	Widgets          []shared.JSONValue                              `json:"widgets"`
	Config           shared.JSONValue                                `json:"config"`
}

func init() {
	registerDashboardRuntimeHooksMutator(func(_ *DashboardServiceManager, hooks *DashboardRuntimeHooks) {
		applyDashboardRuntimeHooks(hooks)
	})
}

func applyDashboardRuntimeHooks(hooks *DashboardRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDashboardRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardDashboardExistingBeforeCreate
	hooks.Create.Fields = dashboardCreateFields()
	hooks.Get.Fields = dashboardGetFields()
	hooks.List.Fields = dashboardListFields()
	hooks.Update.Fields = dashboardUpdateFields()
	hooks.Delete.Fields = dashboardDeleteFields()
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *dashboardservicev1beta1.Dashboard,
		_ string,
	) (any, error) {
		return buildDashboardCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *dashboardservicev1beta1.Dashboard,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDashboardUpdateBody(resource, currentResponse)
	}
}

func newDashboardServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client dashboardOCIClient,
) DashboardServiceClient {
	hooks := newDashboardRuntimeHooksWithOCIClient(client)
	applyDashboardRuntimeHooks(&hooks)
	delegate := defaultDashboardServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dashboardservicev1beta1.Dashboard](
			buildDashboardGeneratedRuntimeConfig(&DashboardServiceManager{Log: log}, hooks),
		),
	}
	return wrapDashboardGeneratedClient(hooks, delegate)
}

func newDashboardRuntimeHooksWithOCIClient(client dashboardOCIClient) DashboardRuntimeHooks {
	return DashboardRuntimeHooks{
		Semantics:       newDashboardRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*dashboardservicev1beta1.Dashboard]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*dashboardservicev1beta1.Dashboard]{},
		StatusHooks:     generatedruntime.StatusHooks[*dashboardservicev1beta1.Dashboard]{},
		ParityHooks:     generatedruntime.ParityHooks[*dashboardservicev1beta1.Dashboard]{},
		Async:           generatedruntime.AsyncHooks[*dashboardservicev1beta1.Dashboard]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*dashboardservicev1beta1.Dashboard]{},
		Create: runtimeOperationHooks[dashboardservicesdk.CreateDashboardRequest, dashboardservicesdk.CreateDashboardResponse]{
			Fields: dashboardCreateFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.CreateDashboardRequest) (dashboardservicesdk.CreateDashboardResponse, error) {
				return client.CreateDashboard(ctx, request)
			},
		},
		Get: runtimeOperationHooks[dashboardservicesdk.GetDashboardRequest, dashboardservicesdk.GetDashboardResponse]{
			Fields: dashboardGetFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.GetDashboardRequest) (dashboardservicesdk.GetDashboardResponse, error) {
				return client.GetDashboard(ctx, request)
			},
		},
		List: runtimeOperationHooks[dashboardservicesdk.ListDashboardsRequest, dashboardservicesdk.ListDashboardsResponse]{
			Fields: dashboardListFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.ListDashboardsRequest) (dashboardservicesdk.ListDashboardsResponse, error) {
				return client.ListDashboards(ctx, request)
			},
		},
		Update: runtimeOperationHooks[dashboardservicesdk.UpdateDashboardRequest, dashboardservicesdk.UpdateDashboardResponse]{
			Fields: dashboardUpdateFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.UpdateDashboardRequest) (dashboardservicesdk.UpdateDashboardResponse, error) {
				return client.UpdateDashboard(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[dashboardservicesdk.DeleteDashboardRequest, dashboardservicesdk.DeleteDashboardResponse]{
			Fields: dashboardDeleteFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.DeleteDashboardRequest) (dashboardservicesdk.DeleteDashboardResponse, error) {
				return client.DeleteDashboard(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DashboardServiceClient) DashboardServiceClient{},
	}
}

func reviewedDashboardRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dashboardservice",
		FormalSlug:    "dashboard",
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
			TerminalStates: []string{"DELETED", "NOT_FOUND"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"dashboardGroupId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"config", "definedTags", "description", "displayName", "freeformTags", "widgets"},
			ForceNew:      []string{"dashboardGroupId", "schemaVersion", "systemTags"},
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
		AuxiliaryOperations: nil,
		Unsupported:         nil,
	}
}

func dashboardCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDashboardDetails", RequestName: "CreateDashboardDetails", Contribution: "body"},
	}
}

func dashboardGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DashboardId", RequestName: "dashboardId", Contribution: "path", PreferResourceID: true},
	}
}

func dashboardListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "DashboardGroupId",
			RequestName:  "dashboardGroupId",
			Contribution: "query",
			LookupPaths:  []string{"status.dashboardGroupId", "spec.dashboardGroupId", "dashboardGroupId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
	}
}

func dashboardUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DashboardId", RequestName: "dashboardId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateDashboardDetails", RequestName: "UpdateDashboardDetails", Contribution: "body"},
	}
}

func dashboardDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DashboardId", RequestName: "dashboardId", Contribution: "path", PreferResourceID: true},
	}
}

func guardDashboardExistingBeforeCreate(
	_ context.Context,
	resource *dashboardservicev1beta1.Dashboard,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Dashboard resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DashboardGroupId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Dashboard spec.dashboardGroupId is required")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildDashboardCreateBody(
	resource *dashboardservicev1beta1.Dashboard,
) (dashboardservicesdk.CreateDashboardDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("Dashboard resource is nil")
	}

	schemaVersion, err := dashboardSchemaVersion(resource.Spec.SchemaVersion)
	if err != nil {
		return nil, err
	}

	switch schemaVersion {
	case dashboardSchemaVersionV1:
		return buildDashboardCreateV1Details(resource.Spec)
	default:
		return nil, fmt.Errorf("unsupported Dashboard schemaVersion %q", schemaVersion)
	}
}

func buildDashboardCreateV1Details(
	spec dashboardservicev1beta1.DashboardSpec,
) (dashboardservicesdk.CreateV1DashboardDetails, error) {
	if strings.TrimSpace(spec.DashboardGroupId) == "" {
		return dashboardservicesdk.CreateV1DashboardDetails{}, fmt.Errorf("Dashboard spec.dashboardGroupId is required")
	}

	widgets, ok, err := dashboardWidgetsFromSpec(spec.Widgets)
	if err != nil {
		return dashboardservicesdk.CreateV1DashboardDetails{}, err
	}
	if !ok {
		return dashboardservicesdk.CreateV1DashboardDetails{}, fmt.Errorf("Dashboard spec.widgets is required for schemaVersion %q", dashboardSchemaVersionV1)
	}

	details := dashboardservicesdk.CreateV1DashboardDetails{
		DashboardGroupId: common.String(spec.DashboardGroupId),
		Widgets:          widgets,
	}
	if spec.DisplayName != "" {
		details.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		details.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = dashboardDefinedTagsFromSpec(spec.DefinedTags)
	}
	if config, ok, err := dashboardConfigFromSpec(spec.Config); err != nil {
		return dashboardservicesdk.CreateV1DashboardDetails{}, err
	} else if ok {
		details.Config = config
	}

	return details, nil
}

func buildDashboardUpdateBody(
	resource *dashboardservicev1beta1.Dashboard,
	currentResponse any,
) (dashboardservicesdk.UpdateDashboardDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("Dashboard resource is nil")
	}

	current, err := dashboardRuntimeStateFromResponse(currentResponse)
	if err != nil {
		return nil, false, err
	}

	schemaVersion, err := dashboardSchemaVersion(dashboardEffectiveSchemaVersion(resource.Spec.SchemaVersion, current.SchemaVersion))
	if err != nil {
		return nil, false, err
	}

	switch schemaVersion {
	case dashboardSchemaVersionV1:
		details, updateNeeded, err := buildDashboardUpdateV1Details(resource.Spec, current)
		if err != nil {
			return nil, false, err
		}
		return details, updateNeeded, nil
	default:
		return nil, false, fmt.Errorf("unsupported Dashboard schemaVersion %q", schemaVersion)
	}
}

func buildDashboardUpdateV1Details(
	spec dashboardservicev1beta1.DashboardSpec,
	current dashboardRuntimeState,
) (dashboardservicesdk.UpdateV1DashboardDetails, bool, error) {
	details := dashboardservicesdk.UpdateV1DashboardDetails{}
	updateNeeded := false

	if desired, ok := dashboardDesiredStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := dashboardDesiredStringUpdate(spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := dashboardDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := dashboardDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if desired, ok, err := dashboardDesiredConfigUpdate(spec.Config, current.Config); err != nil {
		return dashboardservicesdk.UpdateV1DashboardDetails{}, false, err
	} else if ok {
		details.Config = desired
		updateNeeded = true
	}
	if desired, ok, err := dashboardDesiredWidgetsUpdate(spec.Widgets, current.Widgets); err != nil {
		return dashboardservicesdk.UpdateV1DashboardDetails{}, false, err
	} else if ok {
		details.Widgets = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func dashboardRuntimeStateFromResponse(currentResponse any) (dashboardRuntimeState, error) {
	payload, err := dashboardRuntimePayload(currentResponse)
	if err != nil {
		return dashboardRuntimeState{}, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return dashboardRuntimeState{}, fmt.Errorf("marshal current Dashboard response body: %w", err)
	}

	var state dashboardRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		return dashboardRuntimeState{}, fmt.Errorf("decode current Dashboard response body: %w", err)
	}
	return state, nil
}

func dashboardRuntimePayload(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case dashboardservicesdk.CreateDashboardResponse:
		if current.Dashboard == nil {
			return nil, fmt.Errorf("current Dashboard create response body is nil")
		}
		return current.Dashboard, nil
	case *dashboardservicesdk.CreateDashboardResponse:
		if current == nil {
			return nil, fmt.Errorf("current Dashboard create response is nil")
		}
		return dashboardRuntimePayload(*current)
	case dashboardservicesdk.GetDashboardResponse:
		if current.Dashboard == nil {
			return nil, fmt.Errorf("current Dashboard get response body is nil")
		}
		return current.Dashboard, nil
	case *dashboardservicesdk.GetDashboardResponse:
		if current == nil {
			return nil, fmt.Errorf("current Dashboard get response is nil")
		}
		return dashboardRuntimePayload(*current)
	case dashboardservicesdk.UpdateDashboardResponse:
		if current.Dashboard == nil {
			return nil, fmt.Errorf("current Dashboard update response body is nil")
		}
		return current.Dashboard, nil
	case *dashboardservicesdk.UpdateDashboardResponse:
		if current == nil {
			return nil, fmt.Errorf("current Dashboard update response is nil")
		}
		return dashboardRuntimePayload(*current)
	case dashboardservicesdk.DashboardSummary:
		return current, nil
	case *dashboardservicesdk.DashboardSummary:
		if current == nil {
			return nil, fmt.Errorf("current Dashboard summary is nil")
		}
		return *current, nil
	case dashboardservicesdk.Dashboard:
		if current == nil {
			return nil, fmt.Errorf("current Dashboard body is nil")
		}
		return current, nil
	default:
		return nil, fmt.Errorf("unexpected current Dashboard response type %T", currentResponse)
	}
}

func dashboardSchemaVersion(raw string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", dashboardSchemaVersionV1:
		return dashboardSchemaVersionV1, nil
	default:
		return "", fmt.Errorf("Dashboard spec.schemaVersion %q is not supported; only %q is currently published", raw, dashboardSchemaVersionV1)
	}
}

func dashboardEffectiveSchemaVersion(spec string, current string) string {
	if strings.TrimSpace(spec) != "" {
		return spec
	}
	return current
}

// Empty string and omission are indistinguishable for Dashboard string spec
// fields, so only non-empty desired values trigger in-place updates.
func dashboardDesiredStringUpdate(spec string, current *string) (*string, bool) {
	if spec == "" {
		return nil, false
	}

	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	return common.String(spec), true
}

func dashboardDesiredFreeformTagsUpdate(
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

func dashboardDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := dashboardDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if dashboardJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func dashboardDesiredConfigUpdate(
	spec shared.JSONValue,
	current shared.JSONValue,
) (*interface{}, bool, error) {
	if spec.Raw == nil {
		return nil, false, nil
	}
	if dashboardJSONEqual(spec, current) {
		return nil, false, nil
	}
	desired, ok, err := dashboardConfigFromSpec(spec)
	return desired, ok, err
}

func dashboardDesiredWidgetsUpdate(
	spec []shared.JSONValue,
	current []shared.JSONValue,
) ([]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if dashboardJSONEqual(spec, current) {
		return nil, false, nil
	}
	return dashboardWidgetsFromSpec(spec)
}

func dashboardConfigFromSpec(spec shared.JSONValue) (*interface{}, bool, error) {
	if spec.Raw == nil {
		return nil, false, nil
	}

	var decoded interface{}
	if err := json.Unmarshal(spec.Raw, &decoded); err != nil {
		return nil, false, fmt.Errorf("decode Dashboard config JSON: %w", err)
	}
	return &decoded, true, nil
}

func dashboardWidgetsFromSpec(spec []shared.JSONValue) ([]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, false, fmt.Errorf("marshal Dashboard widgets JSON: %w", err)
	}

	var decoded []interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false, fmt.Errorf("decode Dashboard widgets JSON: %w", err)
	}
	return decoded, true, nil
}

func dashboardDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func dashboardJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
