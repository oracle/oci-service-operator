/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drprotectiongroup

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	disasterrecoverysdk "github.com/oracle/oci-go-sdk/v65/disasterrecovery"
	disasterrecoveryv1beta1 "github.com/oracle/oci-service-operator/api/disasterrecovery/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const drProtectionGroupKind = "DrProtectionGroup"

var drProtectionGroupWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(disasterrecoverysdk.OperationStatusAccepted),
		string(disasterrecoverysdk.OperationStatusInProgress),
		string(disasterrecoverysdk.OperationStatusWaiting),
		string(disasterrecoverysdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(disasterrecoverysdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(disasterrecoverysdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(disasterrecoverysdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(disasterrecoverysdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(disasterrecoverysdk.OperationTypeCreateDrProtectionGroup)},
	UpdateActionTokens:    []string{string(disasterrecoverysdk.OperationTypeUpdateDrProtectionGroup)},
	DeleteActionTokens:    []string{string(disasterrecoverysdk.OperationTypeDeleteDrProtectionGroup)},
}

type drProtectionGroupOCIClient interface {
	CreateDrProtectionGroup(context.Context, disasterrecoverysdk.CreateDrProtectionGroupRequest) (disasterrecoverysdk.CreateDrProtectionGroupResponse, error)
	GetDrProtectionGroup(context.Context, disasterrecoverysdk.GetDrProtectionGroupRequest) (disasterrecoverysdk.GetDrProtectionGroupResponse, error)
	ListDrProtectionGroups(context.Context, disasterrecoverysdk.ListDrProtectionGroupsRequest) (disasterrecoverysdk.ListDrProtectionGroupsResponse, error)
	UpdateDrProtectionGroup(context.Context, disasterrecoverysdk.UpdateDrProtectionGroupRequest) (disasterrecoverysdk.UpdateDrProtectionGroupResponse, error)
	DeleteDrProtectionGroup(context.Context, disasterrecoverysdk.DeleteDrProtectionGroupRequest) (disasterrecoverysdk.DeleteDrProtectionGroupResponse, error)
	GetWorkRequest(context.Context, disasterrecoverysdk.GetWorkRequestRequest) (disasterrecoverysdk.GetWorkRequestResponse, error)
}

func init() {
	registerDrProtectionGroupRuntimeHooksMutator(func(manager *DrProtectionGroupServiceManager, hooks *DrProtectionGroupRuntimeHooks) {
		client, initErr := newDrProtectionGroupSDKClient(manager)
		applyDrProtectionGroupRuntimeHooks(hooks, client, initErr)
	})
}

func newDrProtectionGroupSDKClient(manager *DrProtectionGroupServiceManager) (drProtectionGroupOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", drProtectionGroupKind)
	}

	client, err := disasterrecoverysdk.NewDisasterRecoveryClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyDrProtectionGroupRuntimeHooks(
	hooks *DrProtectionGroupRuntimeHooks,
	client drProtectionGroupOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDrProtectionGroupRuntimeSemantics()
	hooks.List.Fields = drProtectionGroupListFields()
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *disasterrecoveryv1beta1.DrProtectionGroup,
		_ string,
	) (any, error) {
		return buildDrProtectionGroupCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *disasterrecoveryv1beta1.DrProtectionGroup,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDrProtectionGroupUpdateBody(resource, currentResponse)
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeDrProtectionGroupDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDrProtectionGroupCreateOnlyDriftForResponse
	hooks.Async.Adapter = drProtectionGroupWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDrProtectionGroupWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveDrProtectionGroupGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveDrProtectionGroupGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverDrProtectionGroupIDFromGeneratedWorkRequest
	hooks.Async.Message = drProtectionGroupGeneratedWorkRequestMessage
}

func reviewedDrProtectionGroupRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newDrProtectionGroupRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "lifecycleState", "role", "lifecycleSubState"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func drProtectionGroupListFields() []generatedruntime.RequestField {
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
			FieldName:    "LifecycleState",
			RequestName:  "lifecycleState",
			Contribution: "query",
			LookupPaths:  []string{"spec.lifecycleState", "lifecycleState"},
		},
		{
			FieldName:    "Role",
			RequestName:  "role",
			Contribution: "query",
			LookupPaths:  []string{"spec.role", "role"},
		},
		{
			FieldName:    "LifecycleSubState",
			RequestName:  "lifecycleSubState",
			Contribution: "query",
			LookupPaths:  []string{"spec.lifecycleSubState", "lifecycleSubState"},
		},
	}
}

func normalizeDrProtectionGroupDesiredState(resource *disasterrecoveryv1beta1.DrProtectionGroup, _ any) {
	if resource == nil {
		return
	}

	resource.Spec.LifecycleState = ""
	resource.Spec.Role = ""
	resource.Spec.LifecycleSubState = ""
}

func validateDrProtectionGroupCreateOnlyDriftForResponse(
	resource *disasterrecoveryv1beta1.DrProtectionGroup,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", drProtectionGroupKind)
	}

	desiredAssociation, ok, err := drProtectionGroupAssociationPayload(resource.Spec.Association)
	if err != nil || !ok {
		return err
	}

	current, err := drProtectionGroupRuntimeBody(currentResponse)
	if err != nil {
		return err
	}

	currentAssociation := map[string]any{
		"role": string(current.Role),
	}
	if peerID := drProtectionGroupStringValue(current.PeerId); peerID != "" {
		currentAssociation["peerId"] = peerID
	}
	if peerRegion := drProtectionGroupStringValue(current.PeerRegion); peerRegion != "" {
		currentAssociation["peerRegion"] = peerRegion
	}

	if drProtectionGroupMapSubsetEqual(desiredAssociation, currentAssociation) {
		return nil
	}
	return fmt.Errorf("%s formal semantics require replacement when association changes", drProtectionGroupKind)
}

func buildDrProtectionGroupCreateBody(
	resource *disasterrecoveryv1beta1.DrProtectionGroup,
) (disasterrecoverysdk.CreateDrProtectionGroupDetails, error) {
	if resource == nil {
		return disasterrecoverysdk.CreateDrProtectionGroupDetails{}, fmt.Errorf("%s resource is nil", drProtectionGroupKind)
	}

	payload, err := drProtectionGroupCreatePayload(resource.Spec)
	if err != nil {
		return disasterrecoverysdk.CreateDrProtectionGroupDetails{}, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return disasterrecoverysdk.CreateDrProtectionGroupDetails{}, fmt.Errorf("marshal %s create body: %w", drProtectionGroupKind, err)
	}

	var details disasterrecoverysdk.CreateDrProtectionGroupDetails
	if err := json.Unmarshal(raw, &details); err != nil {
		return disasterrecoverysdk.CreateDrProtectionGroupDetails{}, fmt.Errorf("decode %s create body: %w", drProtectionGroupKind, err)
	}
	return details, nil
}

func buildDrProtectionGroupUpdateBody(
	resource *disasterrecoveryv1beta1.DrProtectionGroup,
	currentResponse any,
) (disasterrecoverysdk.UpdateDrProtectionGroupDetails, bool, error) {
	if resource == nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, fmt.Errorf("%s resource is nil", drProtectionGroupKind)
	}

	desiredPayload, err := drProtectionGroupUpdatePayload(resource.Spec)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, err
	}

	desiredRaw, err := json.Marshal(desiredPayload)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, fmt.Errorf("marshal desired %s update body: %w", drProtectionGroupKind, err)
	}

	var desired disasterrecoverysdk.UpdateDrProtectionGroupDetails
	if err := json.Unmarshal(desiredRaw, &desired); err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, fmt.Errorf("decode desired %s update body: %w", drProtectionGroupKind, err)
	}

	desiredValues, err := drProtectionGroupPrunedJSONMap(desired)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, fmt.Errorf("project desired %s update body: %w", drProtectionGroupKind, err)
	}
	normalizeDrProtectionGroupUpdateMap(desiredValues)

	current, err := currentDrProtectionGroupUpdateDetails(currentResponse)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, err
	}
	currentValues, err := drProtectionGroupPrunedJSONMap(current)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, false, fmt.Errorf("project current %s update body: %w", drProtectionGroupKind, err)
	}
	normalizeDrProtectionGroupUpdateMap(currentValues)

	return desired, !drProtectionGroupMapSubsetEqual(desiredValues, currentValues), nil
}

func currentDrProtectionGroupUpdateDetails(
	currentResponse any,
) (disasterrecoverysdk.UpdateDrProtectionGroupDetails, error) {
	if currentResponse == nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, nil
	}

	body, err := drProtectionGroupRuntimeBody(currentResponse)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, err
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, fmt.Errorf("marshal current %s response: %w", drProtectionGroupKind, err)
	}

	var details disasterrecoverysdk.UpdateDrProtectionGroupDetails
	if err := json.Unmarshal(raw, &details); err != nil {
		return disasterrecoverysdk.UpdateDrProtectionGroupDetails{}, fmt.Errorf("decode current %s update body: %w", drProtectionGroupKind, err)
	}
	return details, nil
}

func drProtectionGroupCreatePayload(
	spec disasterrecoveryv1beta1.DrProtectionGroupSpec,
) (map[string]any, error) {
	payload := map[string]any{
		"compartmentId": strings.TrimSpace(spec.CompartmentId),
		"displayName":   strings.TrimSpace(spec.DisplayName),
		"logLocation": map[string]any{
			"namespace": strings.TrimSpace(spec.LogLocation.Namespace),
			"bucket":    strings.TrimSpace(spec.LogLocation.Bucket),
		},
	}

	if association, ok, err := drProtectionGroupAssociationPayload(spec.Association); err != nil {
		return nil, err
	} else if ok {
		payload["association"] = association
	}

	if spec.Members != nil {
		members, err := drProtectionGroupMembersPayload(spec.Members)
		if err != nil {
			return nil, err
		}
		payload["members"] = members
	}
	if spec.FreeformTags != nil {
		payload["freeformTags"] = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		payload["definedTags"] = drProtectionGroupDefinedTagsFromSpec(spec.DefinedTags)
	}
	return payload, nil
}

func drProtectionGroupUpdatePayload(
	spec disasterrecoveryv1beta1.DrProtectionGroupSpec,
) (map[string]any, error) {
	payload := map[string]any{
		"logLocation": map[string]any{
			"namespace": strings.TrimSpace(spec.LogLocation.Namespace),
			"bucket":    strings.TrimSpace(spec.LogLocation.Bucket),
		},
		"members":      []any{},
		"freeformTags": map[string]string{},
		"definedTags":  map[string]map[string]interface{}{},
	}

	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" {
		payload["displayName"] = displayName
	}
	if len(spec.Members) != 0 {
		members, err := drProtectionGroupMembersPayload(spec.Members)
		if err != nil {
			return nil, err
		}
		payload["members"] = members
	}
	if spec.FreeformTags != nil {
		payload["freeformTags"] = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		payload["definedTags"] = drProtectionGroupDefinedTagsFromSpec(spec.DefinedTags)
	}
	return payload, nil
}

func drProtectionGroupAssociationPayload(
	association disasterrecoveryv1beta1.DrProtectionGroupAssociation,
) (map[string]any, bool, error) {
	role := strings.TrimSpace(association.Role)
	peerID := strings.TrimSpace(association.PeerId)
	peerRegion := strings.TrimSpace(association.PeerRegion)

	if role == "" && peerID == "" && peerRegion == "" {
		return nil, false, nil
	}
	if role == "" {
		return nil, false, fmt.Errorf("%s association requires role when peer details are set", drProtectionGroupKind)
	}

	payload := map[string]any{"role": role}
	if peerID != "" {
		payload["peerId"] = peerID
	}
	if peerRegion != "" {
		payload["peerRegion"] = peerRegion
	}
	return payload, true, nil
}

func drProtectionGroupMembersPayload(
	members []disasterrecoveryv1beta1.DrProtectionGroupMember,
) ([]any, error) {
	if members == nil {
		return nil, nil
	}
	if len(members) == 0 {
		return []any{}, nil
	}

	out := make([]any, 0, len(members))
	for i, member := range members {
		payload, err := drProtectionGroupMemberPayload(member, fmt.Sprintf("spec.members[%d]", i))
		if err != nil {
			return nil, err
		}
		out = append(out, payload)
	}
	return out, nil
}

func drProtectionGroupMemberPayload(
	member disasterrecoveryv1beta1.DrProtectionGroupMember,
	fieldPath string,
) (map[string]any, error) {
	payload := map[string]any{}
	if raw := strings.TrimSpace(member.JsonData); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return nil, fmt.Errorf("decode %s.jsonData: %w", fieldPath, err)
		}
	}

	typed, err := drProtectionGroupMeaningfulJSONMap(member)
	if err != nil {
		return nil, fmt.Errorf("project %s typed fields: %w", fieldPath, err)
	}
	delete(typed, "jsonData")

	memberID := strings.TrimSpace(member.MemberId)
	if memberID == "" {
		if rawMemberID, _ := payload["memberId"].(string); strings.TrimSpace(rawMemberID) != "" {
			memberID = strings.TrimSpace(rawMemberID)
		}
	}
	if memberID == "" {
		return nil, fmt.Errorf("%s.memberId is required", fieldPath)
	}

	memberType := strings.TrimSpace(member.MemberType)
	if memberType == "" {
		if rawMemberType, _ := payload["memberType"].(string); strings.TrimSpace(rawMemberType) != "" {
			memberType = strings.TrimSpace(rawMemberType)
		}
	}
	if memberType == "" {
		return nil, fmt.Errorf("%s.memberType is required", fieldPath)
	}

	if rawMemberID, ok := payload["memberId"].(string); ok && strings.TrimSpace(rawMemberID) != "" && strings.TrimSpace(rawMemberID) != memberID {
		return nil, fmt.Errorf("%s.jsonData.memberId conflicts with %s.memberId", fieldPath, fieldPath)
	}
	if rawMemberType, ok := payload["memberType"].(string); ok && strings.TrimSpace(rawMemberType) != "" && !strings.EqualFold(strings.TrimSpace(rawMemberType), memberType) {
		return nil, fmt.Errorf("%s.jsonData.memberType conflicts with %s.memberType", fieldPath, fieldPath)
	}

	for key, value := range typed {
		payload[key] = value
	}
	payload["memberId"] = memberID
	payload["memberType"] = memberType
	return payload, nil
}

func drProtectionGroupRuntimeBody(currentResponse any) (disasterrecoverysdk.DrProtectionGroup, error) {
	switch current := currentResponse.(type) {
	case disasterrecoverysdk.DrProtectionGroup:
		return current, nil
	case *disasterrecoverysdk.DrProtectionGroup:
		if current == nil {
			return disasterrecoverysdk.DrProtectionGroup{}, fmt.Errorf("current %s response is nil", drProtectionGroupKind)
		}
		return *current, nil
	case disasterrecoverysdk.DrProtectionGroupSummary:
		return drProtectionGroupFromSummary(current), nil
	case *disasterrecoverysdk.DrProtectionGroupSummary:
		if current == nil {
			return disasterrecoverysdk.DrProtectionGroup{}, fmt.Errorf("current %s response is nil", drProtectionGroupKind)
		}
		return drProtectionGroupFromSummary(*current), nil
	case disasterrecoverysdk.CreateDrProtectionGroupResponse:
		return current.DrProtectionGroup, nil
	case *disasterrecoverysdk.CreateDrProtectionGroupResponse:
		if current == nil {
			return disasterrecoverysdk.DrProtectionGroup{}, fmt.Errorf("current %s response is nil", drProtectionGroupKind)
		}
		return current.DrProtectionGroup, nil
	case disasterrecoverysdk.GetDrProtectionGroupResponse:
		return current.DrProtectionGroup, nil
	case *disasterrecoverysdk.GetDrProtectionGroupResponse:
		if current == nil {
			return disasterrecoverysdk.DrProtectionGroup{}, fmt.Errorf("current %s response is nil", drProtectionGroupKind)
		}
		return current.DrProtectionGroup, nil
	default:
		return disasterrecoverysdk.DrProtectionGroup{}, fmt.Errorf("unexpected %s response type %T", drProtectionGroupKind, currentResponse)
	}
}

func drProtectionGroupFromSummary(summary disasterrecoverysdk.DrProtectionGroupSummary) disasterrecoverysdk.DrProtectionGroup {
	return disasterrecoverysdk.DrProtectionGroup{
		Id:                summary.Id,
		CompartmentId:     summary.CompartmentId,
		DisplayName:       summary.DisplayName,
		Role:              summary.Role,
		TimeCreated:       summary.TimeCreated,
		TimeUpdated:       summary.TimeUpdated,
		LifecycleState:    summary.LifecycleState,
		PeerId:            summary.PeerId,
		PeerRegion:        summary.PeerRegion,
		LifeCycleDetails:  summary.LifeCycleDetails,
		LifecycleSubState: summary.LifecycleSubState,
		FreeformTags:      summary.FreeformTags,
		DefinedTags:       summary.DefinedTags,
		SystemTags:        summary.SystemTags,
	}
}

func getDrProtectionGroupWorkRequest(
	ctx context.Context,
	client drProtectionGroupOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", drProtectionGroupKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", drProtectionGroupKind)
	}

	response, err := client.GetWorkRequest(ctx, disasterrecoverysdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveDrProtectionGroupGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := drProtectionGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveDrProtectionGroupGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := drProtectionGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := drProtectionGroupWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverDrProtectionGroupIDFromGeneratedWorkRequest(
	_ *disasterrecoveryv1beta1.DrProtectionGroup,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := drProtectionGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := drProtectionGroupWorkRequestActionForPhase(phase)
	if id, ok := resolveDrProtectionGroupIDFromWorkRequestResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveDrProtectionGroupIDFromWorkRequestResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", drProtectionGroupKind, phase, drProtectionGroupStringValue(current.Id), drProtectionGroupKind)
}

func drProtectionGroupGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := drProtectionGroupWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", drProtectionGroupKind, phase, drProtectionGroupStringValue(current.Id), current.Status)
}

func drProtectionGroupWorkRequestFromAny(workRequest any) (disasterrecoverysdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case disasterrecoverysdk.WorkRequest:
		return current, nil
	case *disasterrecoverysdk.WorkRequest:
		if current == nil {
			return disasterrecoverysdk.WorkRequest{}, fmt.Errorf("%s work request is nil", drProtectionGroupKind)
		}
		return *current, nil
	default:
		return disasterrecoverysdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", drProtectionGroupKind, workRequest)
	}
}

func drProtectionGroupWorkRequestPhaseFromOperationType(
	operationType disasterrecoverysdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case disasterrecoverysdk.OperationTypeCreateDrProtectionGroup:
		return shared.OSOKAsyncPhaseCreate, true
	case disasterrecoverysdk.OperationTypeUpdateDrProtectionGroup:
		return shared.OSOKAsyncPhaseUpdate, true
	case disasterrecoverysdk.OperationTypeDeleteDrProtectionGroup:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func drProtectionGroupWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) disasterrecoverysdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return disasterrecoverysdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return disasterrecoverysdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return disasterrecoverysdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveDrProtectionGroupIDFromWorkRequestResources(
	resources []disasterrecoverysdk.WorkRequestResource,
	action disasterrecoverysdk.ActionTypeEnum,
	preferDrProtectionGroupOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferDrProtectionGroupOnly && !isDrProtectionGroupWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(drProtectionGroupStringValue(resource.Identifier))
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

func isDrProtectionGroupWorkRequestResource(resource disasterrecoverysdk.WorkRequestResource) bool {
	entityToken := normalizeDrProtectionGroupWorkRequestToken(drProtectionGroupStringValue(resource.EntityType))
	if entityToken == "drprotectiongroup" || strings.Contains(entityToken, "drprotectiongroup") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(drProtectionGroupStringValue(resource.Identifier))), "ocid1.drprotectiongroup")
}

func normalizeDrProtectionGroupWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func drProtectionGroupDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	var out map[string]map[string]interface{}
	if err := drProtectionGroupJSONConvert(spec, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return map[string]map[string]interface{}{}
	}
	return out
}

func drProtectionGroupMeaningfulJSONMap(value any) (map[string]any, error) {
	values, err := drProtectionGroupJSONMap(value)
	if err != nil {
		return nil, err
	}
	pruned, ok := drProtectionGroupPruneMeaningfulJSONValue(values)
	if !ok {
		return map[string]any{}, nil
	}
	prunedMap, ok := pruned.(map[string]any)
	if !ok || prunedMap == nil {
		return map[string]any{}, nil
	}
	return prunedMap, nil
}

func drProtectionGroupPruneMeaningfulJSONValue(value any) (any, bool) {
	switch current := value.(type) {
	case nil:
		return nil, false
	case string:
		return strings.TrimSpace(current), strings.TrimSpace(current) != ""
	case bool:
		return current, current
	case float64:
		return current, current != 0
	case map[string]any:
		pruned := make(map[string]any, len(current))
		for key, child := range current {
			prunedChild, ok := drProtectionGroupPruneMeaningfulJSONValue(child)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case []any:
		pruned := make([]any, 0, len(current))
		for _, child := range current {
			prunedChild, ok := drProtectionGroupPruneMeaningfulJSONValue(child)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	default:
		return value, true
	}
}

func drProtectionGroupJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func drProtectionGroupPrunedJSONMap(value any) (map[string]any, error) {
	values, err := drProtectionGroupJSONMap(value)
	if err != nil {
		return nil, err
	}
	pruned, ok := drProtectionGroupPruneJSONValue(values)
	if !ok {
		return map[string]any{}, nil
	}
	prunedMap, ok := pruned.(map[string]any)
	if !ok || prunedMap == nil {
		return map[string]any{}, nil
	}
	return prunedMap, nil
}

func drProtectionGroupPruneJSONValue(value any) (any, bool) {
	switch current := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		if len(current) == 0 {
			return map[string]any{}, true
		}
		pruned := make(map[string]any, len(current))
		for key, child := range current {
			prunedChild, ok := drProtectionGroupPruneJSONValue(child)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case []any:
		if len(current) == 0 {
			return []any{}, true
		}
		pruned := make([]any, 0, len(current))
		for _, child := range current {
			prunedChild, ok := drProtectionGroupPruneJSONValue(child)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
		}
		if len(pruned) == 0 {
			return []any{}, true
		}
		return pruned, true
	default:
		return value, true
	}
}

func normalizeDrProtectionGroupUpdateMap(values map[string]any) {
	if values == nil {
		return
	}

	if _, ok := values["members"]; !ok || values["members"] == nil {
		values["members"] = []any{}
	}
	if _, ok := values["freeformTags"]; !ok || values["freeformTags"] == nil {
		values["freeformTags"] = map[string]any{}
	}
	if _, ok := values["definedTags"]; !ok || values["definedTags"] == nil {
		values["definedTags"] = map[string]any{}
	}
}

func drProtectionGroupJSONConvert(source any, destination any) error {
	payload, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, destination)
}

func drProtectionGroupMapSubsetEqual(want map[string]any, got map[string]any) bool {
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			return false
		}
		if !drProtectionGroupJSONValueEqual(wantValue, gotValue) {
			return false
		}
	}
	return true
}

func drProtectionGroupJSONValueEqual(left any, right any) bool {
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	switch {
	case leftIsMap && rightIsMap:
		return drProtectionGroupMapSubsetEqual(leftMap, rightMap)
	case leftIsMap || rightIsMap:
		return false
	}

	leftSlice, leftIsSlice := left.([]any)
	rightSlice, rightIsSlice := right.([]any)
	switch {
	case leftIsSlice && rightIsSlice:
		if len(leftSlice) != len(rightSlice) {
			return false
		}
		for i := range leftSlice {
			if !drProtectionGroupJSONValueEqual(leftSlice[i], rightSlice[i]) {
				return false
			}
		}
		return true
	case leftIsSlice || rightIsSlice:
		return false
	}

	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func drProtectionGroupStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
