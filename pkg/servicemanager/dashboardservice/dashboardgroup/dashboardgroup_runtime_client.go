/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dashboardgroup

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

type dashboardGroupOCIClient interface {
	CreateDashboardGroup(context.Context, dashboardservicesdk.CreateDashboardGroupRequest) (dashboardservicesdk.CreateDashboardGroupResponse, error)
	GetDashboardGroup(context.Context, dashboardservicesdk.GetDashboardGroupRequest) (dashboardservicesdk.GetDashboardGroupResponse, error)
	ListDashboardGroups(context.Context, dashboardservicesdk.ListDashboardGroupsRequest) (dashboardservicesdk.ListDashboardGroupsResponse, error)
	UpdateDashboardGroup(context.Context, dashboardservicesdk.UpdateDashboardGroupRequest) (dashboardservicesdk.UpdateDashboardGroupResponse, error)
	DeleteDashboardGroup(context.Context, dashboardservicesdk.DeleteDashboardGroupRequest) (dashboardservicesdk.DeleteDashboardGroupResponse, error)
}

func init() {
	registerDashboardGroupRuntimeHooksMutator(func(_ *DashboardGroupServiceManager, hooks *DashboardGroupRuntimeHooks) {
		applyDashboardGroupRuntimeHooks(hooks)
	})
}

func applyDashboardGroupRuntimeHooks(hooks *DashboardGroupRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDashboardGroupRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardDashboardGroupExistingBeforeCreate
	hooks.Create.Fields = dashboardGroupCreateFields()
	hooks.Get.Fields = dashboardGroupGetFields()
	hooks.List.Fields = dashboardGroupListFields()
	hooks.Update.Fields = dashboardGroupUpdateFields()
	hooks.Delete.Fields = dashboardGroupDeleteFields()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *dashboardservicev1beta1.DashboardGroup,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDashboardGroupUpdateBody(resource, currentResponse)
	}
}

func newDashboardGroupServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client dashboardGroupOCIClient,
) DashboardGroupServiceClient {
	hooks := newDashboardGroupRuntimeHooksWithOCIClient(client)
	applyDashboardGroupRuntimeHooks(&hooks)
	delegate := defaultDashboardGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dashboardservicev1beta1.DashboardGroup](
			buildDashboardGroupGeneratedRuntimeConfig(&DashboardGroupServiceManager{Log: log}, hooks),
		),
	}
	return wrapDashboardGroupGeneratedClient(hooks, delegate)
}

func newDashboardGroupRuntimeHooksWithOCIClient(client dashboardGroupOCIClient) DashboardGroupRuntimeHooks {
	return DashboardGroupRuntimeHooks{
		Semantics:       newDashboardGroupRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*dashboardservicev1beta1.DashboardGroup]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*dashboardservicev1beta1.DashboardGroup]{},
		StatusHooks:     generatedruntime.StatusHooks[*dashboardservicev1beta1.DashboardGroup]{},
		ParityHooks:     generatedruntime.ParityHooks[*dashboardservicev1beta1.DashboardGroup]{},
		Async:           generatedruntime.AsyncHooks[*dashboardservicev1beta1.DashboardGroup]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*dashboardservicev1beta1.DashboardGroup]{},
		Create: runtimeOperationHooks[dashboardservicesdk.CreateDashboardGroupRequest, dashboardservicesdk.CreateDashboardGroupResponse]{
			Fields: dashboardGroupCreateFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.CreateDashboardGroupRequest) (dashboardservicesdk.CreateDashboardGroupResponse, error) {
				return client.CreateDashboardGroup(ctx, request)
			},
		},
		Get: runtimeOperationHooks[dashboardservicesdk.GetDashboardGroupRequest, dashboardservicesdk.GetDashboardGroupResponse]{
			Fields: dashboardGroupGetFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.GetDashboardGroupRequest) (dashboardservicesdk.GetDashboardGroupResponse, error) {
				return client.GetDashboardGroup(ctx, request)
			},
		},
		List: runtimeOperationHooks[dashboardservicesdk.ListDashboardGroupsRequest, dashboardservicesdk.ListDashboardGroupsResponse]{
			Fields: dashboardGroupListFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.ListDashboardGroupsRequest) (dashboardservicesdk.ListDashboardGroupsResponse, error) {
				return client.ListDashboardGroups(ctx, request)
			},
		},
		Update: runtimeOperationHooks[dashboardservicesdk.UpdateDashboardGroupRequest, dashboardservicesdk.UpdateDashboardGroupResponse]{
			Fields: dashboardGroupUpdateFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.UpdateDashboardGroupRequest) (dashboardservicesdk.UpdateDashboardGroupResponse, error) {
				return client.UpdateDashboardGroup(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[dashboardservicesdk.DeleteDashboardGroupRequest, dashboardservicesdk.DeleteDashboardGroupResponse]{
			Fields: dashboardGroupDeleteFields(),
			Call: func(ctx context.Context, request dashboardservicesdk.DeleteDashboardGroupRequest) (dashboardservicesdk.DeleteDashboardGroupResponse, error) {
				return client.DeleteDashboardGroup(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DashboardGroupServiceClient) DashboardGroupServiceClient{},
	}
}

func reviewedDashboardGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dashboardservice",
		FormalSlug:    "dashboardgroup",
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
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"definedTags", "description", "displayName", "freeformTags"},
			ForceNew:      []string{"compartmentId", "systemTags"},
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

func dashboardGroupCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDashboardGroupDetails", RequestName: "CreateDashboardGroupDetails", Contribution: "body"},
	}
}

func dashboardGroupGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DashboardGroupId", RequestName: "dashboardGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func dashboardGroupListFields() []generatedruntime.RequestField {
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
	}
}

func dashboardGroupUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DashboardGroupId", RequestName: "dashboardGroupId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateDashboardGroupDetails", RequestName: "UpdateDashboardGroupDetails", Contribution: "body"},
	}
}

func dashboardGroupDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DashboardGroupId", RequestName: "dashboardGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func guardDashboardGroupExistingBeforeCreate(
	_ context.Context,
	resource *dashboardservicev1beta1.DashboardGroup,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("DashboardGroup resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("DashboardGroup spec.compartmentId is required")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildDashboardGroupUpdateBody(
	resource *dashboardservicev1beta1.DashboardGroup,
	currentResponse any,
) (dashboardservicesdk.UpdateDashboardGroupDetails, bool, error) {
	if resource == nil {
		return dashboardservicesdk.UpdateDashboardGroupDetails{}, false, fmt.Errorf("DashboardGroup resource is nil")
	}

	current, err := dashboardGroupFromResponse(currentResponse)
	if err != nil {
		return dashboardservicesdk.UpdateDashboardGroupDetails{}, false, err
	}

	details := dashboardservicesdk.UpdateDashboardGroupDetails{}
	updateNeeded := false

	if desired, ok := dashboardGroupDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := dashboardGroupDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := dashboardGroupDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := dashboardGroupDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func dashboardGroupFromResponse(currentResponse any) (dashboardservicesdk.DashboardGroup, error) {
	switch current := currentResponse.(type) {
	case dashboardservicesdk.DashboardGroup:
		return current, nil
	case *dashboardservicesdk.DashboardGroup:
		if current == nil {
			return dashboardservicesdk.DashboardGroup{}, fmt.Errorf("current DashboardGroup response is nil")
		}
		return *current, nil
	case dashboardservicesdk.DashboardGroupSummary:
		return dashboardservicesdk.DashboardGroup{
			Id:             current.Id,
			DisplayName:    current.DisplayName,
			Description:    current.Description,
			CompartmentId:  current.CompartmentId,
			TimeCreated:    current.TimeCreated,
			TimeUpdated:    current.TimeUpdated,
			LifecycleState: current.LifecycleState,
			FreeformTags:   current.FreeformTags,
			DefinedTags:    current.DefinedTags,
			SystemTags:     current.SystemTags,
		}, nil
	case *dashboardservicesdk.DashboardGroupSummary:
		if current == nil {
			return dashboardservicesdk.DashboardGroup{}, fmt.Errorf("current DashboardGroup response is nil")
		}
		return dashboardGroupFromResponse(*current)
	case dashboardservicesdk.CreateDashboardGroupResponse:
		return current.DashboardGroup, nil
	case *dashboardservicesdk.CreateDashboardGroupResponse:
		if current == nil {
			return dashboardservicesdk.DashboardGroup{}, fmt.Errorf("current DashboardGroup response is nil")
		}
		return current.DashboardGroup, nil
	case dashboardservicesdk.GetDashboardGroupResponse:
		return current.DashboardGroup, nil
	case *dashboardservicesdk.GetDashboardGroupResponse:
		if current == nil {
			return dashboardservicesdk.DashboardGroup{}, fmt.Errorf("current DashboardGroup response is nil")
		}
		return current.DashboardGroup, nil
	case dashboardservicesdk.UpdateDashboardGroupResponse:
		return current.DashboardGroup, nil
	case *dashboardservicesdk.UpdateDashboardGroupResponse:
		if current == nil {
			return dashboardservicesdk.DashboardGroup{}, fmt.Errorf("current DashboardGroup response is nil")
		}
		return current.DashboardGroup, nil
	default:
		return dashboardservicesdk.DashboardGroup{}, fmt.Errorf("unexpected current DashboardGroup response type %T", currentResponse)
	}
}

func dashboardGroupDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func dashboardGroupDesiredFreeformTagsUpdate(
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

func dashboardGroupDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := dashboardGroupDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if dashboardGroupJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func dashboardGroupDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func dashboardGroupJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
