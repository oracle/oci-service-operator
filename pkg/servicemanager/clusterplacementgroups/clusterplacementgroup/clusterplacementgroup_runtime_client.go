/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package clusterplacementgroup

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	clusterplacementgroupssdk "github.com/oracle/oci-go-sdk/v65/clusterplacementgroups"
	"github.com/oracle/oci-go-sdk/v65/common"
	clusterplacementgroupsv1beta1 "github.com/oracle/oci-service-operator/api/clusterplacementgroups/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var clusterPlacementGroupWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(clusterplacementgroupssdk.OperationStatusAccepted),
		string(clusterplacementgroupssdk.OperationStatusInProgress),
		string(clusterplacementgroupssdk.OperationStatusWaiting),
		string(clusterplacementgroupssdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(clusterplacementgroupssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(clusterplacementgroupssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(clusterplacementgroupssdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(clusterplacementgroupssdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(clusterplacementgroupssdk.OperationTypeCreateClusterPlacementGroup)},
	UpdateActionTokens:    []string{string(clusterplacementgroupssdk.OperationTypeUpdateClusterPlacementGroup)},
	DeleteActionTokens:    []string{string(clusterplacementgroupssdk.OperationTypeDeleteClusterPlacementGroup)},
}

type clusterPlacementGroupOCIClient interface {
	CreateClusterPlacementGroup(context.Context, clusterplacementgroupssdk.CreateClusterPlacementGroupRequest) (clusterplacementgroupssdk.CreateClusterPlacementGroupResponse, error)
	GetClusterPlacementGroup(context.Context, clusterplacementgroupssdk.GetClusterPlacementGroupRequest) (clusterplacementgroupssdk.GetClusterPlacementGroupResponse, error)
	ListClusterPlacementGroups(context.Context, clusterplacementgroupssdk.ListClusterPlacementGroupsRequest) (clusterplacementgroupssdk.ListClusterPlacementGroupsResponse, error)
	UpdateClusterPlacementGroup(context.Context, clusterplacementgroupssdk.UpdateClusterPlacementGroupRequest) (clusterplacementgroupssdk.UpdateClusterPlacementGroupResponse, error)
	DeleteClusterPlacementGroup(context.Context, clusterplacementgroupssdk.DeleteClusterPlacementGroupRequest) (clusterplacementgroupssdk.DeleteClusterPlacementGroupResponse, error)
	GetWorkRequest(context.Context, clusterplacementgroupssdk.GetWorkRequestRequest) (clusterplacementgroupssdk.GetWorkRequestResponse, error)
}

func init() {
	registerClusterPlacementGroupRuntimeHooksMutator(func(manager *ClusterPlacementGroupServiceManager, hooks *ClusterPlacementGroupRuntimeHooks) {
		client, initErr := newClusterPlacementGroupSDKClient(manager)
		applyClusterPlacementGroupRuntimeHooks(hooks, client, initErr)
	})
}

func newClusterPlacementGroupSDKClient(manager *ClusterPlacementGroupServiceManager) (clusterPlacementGroupOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ClusterPlacementGroup service manager is nil")
	}
	client, err := clusterplacementgroupssdk.NewClusterPlacementGroupsCPClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyClusterPlacementGroupRuntimeHooks(
	hooks *ClusterPlacementGroupRuntimeHooks,
	client clusterPlacementGroupOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedClusterPlacementGroupRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildClusterPlacementGroupUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardClusterPlacementGroupExistingBeforeCreate
	hooks.Create.Fields = clusterPlacementGroupCreateFields()
	hooks.Get.Fields = clusterPlacementGroupGetFields()
	hooks.List.Fields = clusterPlacementGroupListFields()
	hooks.Update.Fields = clusterPlacementGroupUpdateFields()
	hooks.Delete.Fields = clusterPlacementGroupDeleteFields()
	hooks.Async.Adapter = clusterPlacementGroupWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getClusterPlacementGroupWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveClusterPlacementGroupGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveClusterPlacementGroupGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverClusterPlacementGroupIDFromGeneratedWorkRequest
	hooks.Async.Message = clusterPlacementGroupGeneratedWorkRequestMessage
}

func newClusterPlacementGroupServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client clusterPlacementGroupOCIClient,
) ClusterPlacementGroupServiceClient {
	return defaultClusterPlacementGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*clusterplacementgroupsv1beta1.ClusterPlacementGroup](
			newClusterPlacementGroupRuntimeConfig(log, client),
		),
	}
}

func newClusterPlacementGroupRuntimeConfig(
	log loggerutil.OSOKLogger,
	client clusterPlacementGroupOCIClient,
) generatedruntime.Config[*clusterplacementgroupsv1beta1.ClusterPlacementGroup] {
	hooks := newClusterPlacementGroupRuntimeHooksWithOCIClient(client)
	applyClusterPlacementGroupRuntimeHooks(&hooks, client, nil)
	return buildClusterPlacementGroupGeneratedRuntimeConfig(&ClusterPlacementGroupServiceManager{Log: log}, hooks)
}

func newClusterPlacementGroupRuntimeHooksWithOCIClient(client clusterPlacementGroupOCIClient) ClusterPlacementGroupRuntimeHooks {
	return ClusterPlacementGroupRuntimeHooks{
		Semantics: newClusterPlacementGroupRuntimeSemantics(),
		Create: runtimeOperationHooks[clusterplacementgroupssdk.CreateClusterPlacementGroupRequest, clusterplacementgroupssdk.CreateClusterPlacementGroupResponse]{
			Fields: clusterPlacementGroupCreateFields(),
			Call: func(ctx context.Context, request clusterplacementgroupssdk.CreateClusterPlacementGroupRequest) (clusterplacementgroupssdk.CreateClusterPlacementGroupResponse, error) {
				return client.CreateClusterPlacementGroup(ctx, request)
			},
		},
		Get: runtimeOperationHooks[clusterplacementgroupssdk.GetClusterPlacementGroupRequest, clusterplacementgroupssdk.GetClusterPlacementGroupResponse]{
			Fields: clusterPlacementGroupGetFields(),
			Call: func(ctx context.Context, request clusterplacementgroupssdk.GetClusterPlacementGroupRequest) (clusterplacementgroupssdk.GetClusterPlacementGroupResponse, error) {
				return client.GetClusterPlacementGroup(ctx, request)
			},
		},
		List: runtimeOperationHooks[clusterplacementgroupssdk.ListClusterPlacementGroupsRequest, clusterplacementgroupssdk.ListClusterPlacementGroupsResponse]{
			Fields: clusterPlacementGroupListFields(),
			Call: func(ctx context.Context, request clusterplacementgroupssdk.ListClusterPlacementGroupsRequest) (clusterplacementgroupssdk.ListClusterPlacementGroupsResponse, error) {
				return client.ListClusterPlacementGroups(ctx, request)
			},
		},
		Update: runtimeOperationHooks[clusterplacementgroupssdk.UpdateClusterPlacementGroupRequest, clusterplacementgroupssdk.UpdateClusterPlacementGroupResponse]{
			Fields: clusterPlacementGroupUpdateFields(),
			Call: func(ctx context.Context, request clusterplacementgroupssdk.UpdateClusterPlacementGroupRequest) (clusterplacementgroupssdk.UpdateClusterPlacementGroupResponse, error) {
				return client.UpdateClusterPlacementGroup(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[clusterplacementgroupssdk.DeleteClusterPlacementGroupRequest, clusterplacementgroupssdk.DeleteClusterPlacementGroupResponse]{
			Fields: clusterPlacementGroupDeleteFields(),
			Call: func(ctx context.Context, request clusterplacementgroupssdk.DeleteClusterPlacementGroupRequest) (clusterplacementgroupssdk.DeleteClusterPlacementGroupResponse, error) {
				return client.DeleteClusterPlacementGroup(ctx, request)
			},
		},
	}
}

func reviewedClusterPlacementGroupRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newClusterPlacementGroupRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields: []string{
			"availabilityDomain",
			"clusterPlacementGroupType",
			"compartmentId",
			"name",
		},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func clusterPlacementGroupCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateClusterPlacementGroupDetails", RequestName: "CreateClusterPlacementGroupDetails", Contribution: "body"},
	}
}

func clusterPlacementGroupGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterPlacementGroupId", RequestName: "clusterPlacementGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func clusterPlacementGroupListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths:  []string{"status.name", "spec.name", "name"},
		},
		{
			FieldName:    "Ad",
			RequestName:  "ad",
			Contribution: "query",
			LookupPaths:  []string{"status.availabilityDomain", "spec.availabilityDomain", "availabilityDomain"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func clusterPlacementGroupUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterPlacementGroupId", RequestName: "clusterPlacementGroupId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateClusterPlacementGroupDetails", RequestName: "UpdateClusterPlacementGroupDetails", Contribution: "body"},
	}
}

func clusterPlacementGroupDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterPlacementGroupId", RequestName: "clusterPlacementGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func guardClusterPlacementGroupExistingBeforeCreate(
	_ context.Context,
	resource *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ClusterPlacementGroup resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.Name) == "" ||
		strings.TrimSpace(resource.Spec.AvailabilityDomain) == "" ||
		strings.TrimSpace(resource.Spec.ClusterPlacementGroupType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildClusterPlacementGroupUpdateBody(
	resource *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
	currentResponse any,
) (clusterplacementgroupssdk.UpdateClusterPlacementGroupDetails, bool, error) {
	if resource == nil {
		return clusterplacementgroupssdk.UpdateClusterPlacementGroupDetails{}, false, fmt.Errorf("ClusterPlacementGroup resource is nil")
	}

	current, err := clusterPlacementGroupFromResponse(currentResponse)
	if err != nil {
		return clusterplacementgroupssdk.UpdateClusterPlacementGroupDetails{}, false, err
	}

	details := clusterplacementgroupssdk.UpdateClusterPlacementGroupDetails{}
	updateNeeded := false

	if desired, ok := clusterPlacementGroupDesiredDescriptionUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := clusterPlacementGroupDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := clusterPlacementGroupDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func clusterPlacementGroupFromResponse(currentResponse any) (clusterplacementgroupssdk.ClusterPlacementGroup, error) {
	switch current := currentResponse.(type) {
	case clusterplacementgroupssdk.ClusterPlacementGroup:
		return current, nil
	case *clusterplacementgroupssdk.ClusterPlacementGroup:
		if current == nil {
			return clusterplacementgroupssdk.ClusterPlacementGroup{}, fmt.Errorf("current ClusterPlacementGroup response is nil")
		}
		return *current, nil
	case clusterplacementgroupssdk.ClusterPlacementGroupSummary:
		return clusterPlacementGroupFromSummary(current), nil
	case *clusterplacementgroupssdk.ClusterPlacementGroupSummary:
		if current == nil {
			return clusterplacementgroupssdk.ClusterPlacementGroup{}, fmt.Errorf("current ClusterPlacementGroup response is nil")
		}
		return clusterPlacementGroupFromSummary(*current), nil
	case clusterplacementgroupssdk.CreateClusterPlacementGroupResponse:
		return current.ClusterPlacementGroup, nil
	case *clusterplacementgroupssdk.CreateClusterPlacementGroupResponse:
		if current == nil {
			return clusterplacementgroupssdk.ClusterPlacementGroup{}, fmt.Errorf("current ClusterPlacementGroup response is nil")
		}
		return current.ClusterPlacementGroup, nil
	case clusterplacementgroupssdk.GetClusterPlacementGroupResponse:
		return current.ClusterPlacementGroup, nil
	case *clusterplacementgroupssdk.GetClusterPlacementGroupResponse:
		if current == nil {
			return clusterplacementgroupssdk.ClusterPlacementGroup{}, fmt.Errorf("current ClusterPlacementGroup response is nil")
		}
		return current.ClusterPlacementGroup, nil
	default:
		return clusterplacementgroupssdk.ClusterPlacementGroup{}, fmt.Errorf("unexpected current ClusterPlacementGroup response type %T", currentResponse)
	}
}

func clusterPlacementGroupFromSummary(
	summary clusterplacementgroupssdk.ClusterPlacementGroupSummary,
) clusterplacementgroupssdk.ClusterPlacementGroup {
	return clusterplacementgroupssdk.ClusterPlacementGroup{
		Id:                        summary.Id,
		Name:                      summary.Name,
		CompartmentId:             summary.CompartmentId,
		AvailabilityDomain:        summary.AvailabilityDomain,
		ClusterPlacementGroupType: summary.ClusterPlacementGroupType,
		TimeCreated:               summary.TimeCreated,
		LifecycleState:            summary.LifecycleState,
		FreeformTags:              summary.FreeformTags,
		DefinedTags:               summary.DefinedTags,
		TimeUpdated:               summary.TimeUpdated,
		LifecycleDetails:          summary.LifecycleDetails,
		SystemTags:                summary.SystemTags,
	}
}

func clusterPlacementGroupDesiredDescriptionUpdate(spec string, current *string) (*string, bool) {
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

func clusterPlacementGroupDesiredFreeformTagsUpdate(
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

func clusterPlacementGroupDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := clusterPlacementGroupDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if clusterPlacementGroupJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func clusterPlacementGroupDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func clusterPlacementGroupJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getClusterPlacementGroupWorkRequest(
	ctx context.Context,
	client clusterPlacementGroupOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize ClusterPlacementGroup OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("ClusterPlacementGroup OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, clusterplacementgroupssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveClusterPlacementGroupGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := clusterPlacementGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveClusterPlacementGroupGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := clusterPlacementGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := clusterPlacementGroupWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverClusterPlacementGroupIDFromGeneratedWorkRequest(
	_ *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := clusterPlacementGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := clusterPlacementGroupWorkRequestActionForPhase(phase)
	if id, ok := resolveClusterPlacementGroupIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveClusterPlacementGroupIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("ClusterPlacementGroup work request %s does not expose a cluster placement group identifier", stringValue(current.Id))
}

func clusterPlacementGroupGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := clusterPlacementGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ClusterPlacementGroup %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func clusterPlacementGroupWorkRequestFromAny(workRequest any) (clusterplacementgroupssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case clusterplacementgroupssdk.WorkRequest:
		return current, nil
	case *clusterplacementgroupssdk.WorkRequest:
		if current == nil {
			return clusterplacementgroupssdk.WorkRequest{}, fmt.Errorf("ClusterPlacementGroup work request is nil")
		}
		return *current, nil
	default:
		return clusterplacementgroupssdk.WorkRequest{}, fmt.Errorf("unexpected ClusterPlacementGroup work request type %T", workRequest)
	}
}

func clusterPlacementGroupWorkRequestPhaseFromOperationType(
	operationType clusterplacementgroupssdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case clusterplacementgroupssdk.OperationTypeCreateClusterPlacementGroup:
		return shared.OSOKAsyncPhaseCreate, true
	case clusterplacementgroupssdk.OperationTypeUpdateClusterPlacementGroup:
		return shared.OSOKAsyncPhaseUpdate, true
	case clusterplacementgroupssdk.OperationTypeDeleteClusterPlacementGroup:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func clusterPlacementGroupWorkRequestActionForPhase(
	phase shared.OSOKAsyncPhase,
) clusterplacementgroupssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return clusterplacementgroupssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return clusterplacementgroupssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return clusterplacementgroupssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveClusterPlacementGroupIDFromResources(
	resources []clusterplacementgroupssdk.WorkRequestResource,
	action clusterplacementgroupssdk.ActionTypeEnum,
	preferClusterPlacementGroupOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferClusterPlacementGroupOnly && !isClusterPlacementGroupWorkRequestResource(resource) {
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

func isClusterPlacementGroupWorkRequestResource(resource clusterplacementgroupssdk.WorkRequestResource) bool {
	entityType := normalizeClusterPlacementGroupWorkRequestToken(stringValue(resource.EntityType))
	if strings.Contains(entityType, "clusterplacementgroup") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/clusterplacementgroups/")
}

func normalizeClusterPlacementGroupWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
