/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package zprpolicy

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	zprsdk "github.com/oracle/oci-go-sdk/v65/zpr"
	zprv1beta1 "github.com/oracle/oci-service-operator/api/zpr/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

var zprPolicyWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(zprsdk.WorkRequestStatusAccepted),
		string(zprsdk.WorkRequestStatusInProgress),
		string(zprsdk.WorkRequestStatusWaiting),
		string(zprsdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(zprsdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(zprsdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(zprsdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(zprsdk.WorkRequestStatusNeedsAttention)},
	CreateActionTokens:    []string{string(zprsdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(zprsdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(zprsdk.ActionTypeDeleted)},
}

type zprPolicyOCIClient interface {
	CreateZprPolicy(context.Context, zprsdk.CreateZprPolicyRequest) (zprsdk.CreateZprPolicyResponse, error)
	GetZprPolicy(context.Context, zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error)
	ListZprPolicies(context.Context, zprsdk.ListZprPoliciesRequest) (zprsdk.ListZprPoliciesResponse, error)
	UpdateZprPolicy(context.Context, zprsdk.UpdateZprPolicyRequest) (zprsdk.UpdateZprPolicyResponse, error)
	DeleteZprPolicy(context.Context, zprsdk.DeleteZprPolicyRequest) (zprsdk.DeleteZprPolicyResponse, error)
	GetZprPolicyWorkRequest(context.Context, zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error)
}

type zprPolicyWorkRequestClient interface {
	GetZprPolicyWorkRequest(context.Context, zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error)
}

func init() {
	registerZprPolicyRuntimeHooksMutator(func(manager *ZprPolicyServiceManager, hooks *ZprPolicyRuntimeHooks) {
		workRequestClient, initErr := newZprPolicyWorkRequestClient(manager)
		applyZprPolicyRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newZprPolicyWorkRequestClient(manager *ZprPolicyServiceManager) (zprPolicyWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ZprPolicy service manager is nil")
	}
	client, err := zprsdk.NewZprClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyZprPolicyRuntimeHooks(
	hooks *ZprPolicyRuntimeHooks,
	workRequestClient zprPolicyWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.BuildCreateBody = func(_ context.Context, resource *zprv1beta1.ZprPolicy, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("ZprPolicy resource is nil")
		}
		return buildCreateZprPolicyDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *zprv1beta1.ZprPolicy,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildZprPolicyUpdateBody(resource, currentResponse)
	}
	hooks.StatusHooks.ProjectStatus = projectZprPolicyStatus
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedZprPolicyIdentity
	hooks.Async.Adapter = zprPolicyWorkRequestAsyncAdapter
	hooks.Async.ResolveAction = resolveZprPolicyGeneratedWorkRequestAction
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getZprPolicyWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolvePhase = resolveZprPolicyGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverZprPolicyIDFromGeneratedWorkRequest
	hooks.Async.Message = zprPolicyGeneratedWorkRequestMessage
}

func buildCreateZprPolicyDetails(spec zprv1beta1.ZprPolicySpec) zprsdk.CreateZprPolicyDetails {
	createDetails := zprsdk.CreateZprPolicyDetails{
		CompartmentId: common.String(spec.CompartmentId),
		Description:   common.String(spec.Description),
		Name:          common.String(spec.Name),
		Statements:    append([]string(nil), spec.Statements...),
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return createDetails
}

func buildZprPolicyUpdateBody(
	resource *zprv1beta1.ZprPolicy,
	currentResponse any,
) (zprsdk.UpdateZprPolicyDetails, bool, error) {
	if resource == nil {
		return zprsdk.UpdateZprPolicyDetails{}, false, fmt.Errorf("ZprPolicy resource is nil")
	}

	current, ok := zprPolicyFromResponse(currentResponse)
	if !ok {
		return zprsdk.UpdateZprPolicyDetails{}, false, fmt.Errorf("current ZprPolicy response does not expose a ZprPolicy body")
	}

	updateDetails := zprsdk.UpdateZprPolicyDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.Description, resource.Spec.Description) {
		updateDetails.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}

	desiredStatements := desiredZprPolicyStatementsForUpdate(resource.Spec.Statements, current.Statements)
	if !slices.Equal(current.Statements, desiredStatements) {
		updateDetails.Statements = desiredStatements
		updateNeeded = true
	}

	desiredFreeformTags := desiredZprPolicyFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredZprPolicyDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return zprsdk.UpdateZprPolicyDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func projectZprPolicyStatus(resource *zprv1beta1.ZprPolicy, response any) error {
	if resource == nil {
		return fmt.Errorf("ZprPolicy resource is nil")
	}

	current, ok := zprPolicyFromResponse(response)
	if !ok {
		return nil
	}

	resource.Status = zprv1beta1.ZprPolicyStatus{
		OsokStatus:       resource.Status.OsokStatus,
		Id:               stringValue(current.Id),
		Name:             stringValue(current.Name),
		Description:      stringValue(current.Description),
		CompartmentId:    stringValue(current.CompartmentId),
		Statements:       append([]string(nil), current.Statements...),
		LifecycleState:   string(current.LifecycleState),
		TimeCreated:      sdkTimeString(current.TimeCreated),
		FreeformTags:     cloneStringMap(current.FreeformTags),
		DefinedTags:      convertOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:       convertOCIToStatusDefinedTags(current.SystemTags),
		LifecycleDetails: stringValue(current.LifecycleDetails),
		TimeUpdated:      sdkTimeString(current.TimeUpdated),
	}
	return nil
}

func zprPolicyFromResponse(response any) (zprsdk.ZprPolicy, bool) {
	switch current := response.(type) {
	case zprsdk.CreateZprPolicyResponse:
		return current.ZprPolicy, true
	case *zprsdk.CreateZprPolicyResponse:
		if current == nil {
			return zprsdk.ZprPolicy{}, false
		}
		return current.ZprPolicy, true
	case zprsdk.GetZprPolicyResponse:
		return current.ZprPolicy, true
	case *zprsdk.GetZprPolicyResponse:
		if current == nil {
			return zprsdk.ZprPolicy{}, false
		}
		return current.ZprPolicy, true
	case zprsdk.ZprPolicy:
		return current, true
	case *zprsdk.ZprPolicy:
		if current == nil {
			return zprsdk.ZprPolicy{}, false
		}
		return *current, true
	case zprsdk.ZprPolicySummary:
		return zprPolicyFromSummary(current), true
	case *zprsdk.ZprPolicySummary:
		if current == nil {
			return zprsdk.ZprPolicy{}, false
		}
		return zprPolicyFromSummary(*current), true
	default:
		return zprsdk.ZprPolicy{}, false
	}
}

func zprPolicyFromSummary(summary zprsdk.ZprPolicySummary) zprsdk.ZprPolicy {
	return zprsdk.ZprPolicy{
		Id:               summary.Id,
		Name:             summary.Name,
		Description:      summary.Description,
		CompartmentId:    summary.CompartmentId,
		Statements:       append([]string(nil), summary.Statements...),
		LifecycleState:   summary.LifecycleState,
		TimeCreated:      summary.TimeCreated,
		FreeformTags:     cloneStringMap(summary.FreeformTags),
		DefinedTags:      cloneOCIDefinedTags(summary.DefinedTags),
		SystemTags:       cloneOCIDefinedTags(summary.SystemTags),
		LifecycleDetails: summary.LifecycleDetails,
		TimeUpdated:      summary.TimeUpdated,
	}
}

func clearTrackedZprPolicyIdentity(resource *zprv1beta1.ZprPolicy) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func getZprPolicyWorkRequest(
	ctx context.Context,
	client zprPolicyWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize ZprPolicy OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("ZprPolicy OCI client is not configured")
	}

	response, err := client.GetZprPolicyWorkRequest(ctx, zprsdk.GetZprPolicyWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveZprPolicyGeneratedWorkRequestAction(workRequest any) (string, error) {
	zprWorkRequest, err := zprPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveZprPolicyWorkRequestAction(zprWorkRequest)
}

func resolveZprPolicyGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	zprWorkRequest, err := zprPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := zprPolicyWorkRequestPhaseFromOperationType(zprWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverZprPolicyIDFromGeneratedWorkRequest(
	_ *zprv1beta1.ZprPolicy,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	zprWorkRequest, err := zprPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveZprPolicyIDFromWorkRequest(zprWorkRequest, zprPolicyWorkRequestActionForPhase(phase))
}

func zprPolicyGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	zprWorkRequest, err := zprPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ZprPolicy %s work request %s is %s", phase, stringValue(zprWorkRequest.Id), zprWorkRequest.Status)
}

func zprPolicyWorkRequestFromAny(workRequest any) (zprsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case zprsdk.WorkRequest:
		return current, nil
	case *zprsdk.WorkRequest:
		if current == nil {
			return zprsdk.WorkRequest{}, fmt.Errorf("ZprPolicy work request is nil")
		}
		return *current, nil
	default:
		return zprsdk.WorkRequest{}, fmt.Errorf("unexpected ZprPolicy work request type %T", workRequest)
	}
}

func zprPolicyWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) zprsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return zprsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return zprsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return zprsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveZprPolicyIDFromWorkRequest(workRequest zprsdk.WorkRequest, action zprsdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveZprPolicyIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveZprPolicyIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("ZprPolicy work request %s does not expose a ZprPolicy identifier", stringValue(workRequest.Id))
}

func resolveZprPolicyIDFromResources(
	resources []zprsdk.WorkRequestResource,
	action zprsdk.ActionTypeEnum,
	preferPolicyOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferPolicyOnly && !isZprPolicyWorkRequestResource(resource) {
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

func resolveZprPolicyWorkRequestAction(workRequest zprsdk.WorkRequest) (string, error) {
	var action string

	for _, resource := range workRequest.Resources {
		if !isZprPolicyWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf(
				"ZprPolicy work request %s exposes conflicting ZprPolicy action types %q and %q",
				stringValue(workRequest.Id),
				action,
				candidate,
			)
		}
	}

	return action, nil
}

func zprPolicyWorkRequestPhaseFromOperationType(operationType zprsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case zprsdk.OperationTypeCreateZprPolicy:
		return shared.OSOKAsyncPhaseCreate, true
	case zprsdk.OperationTypeUpdateZprPolicy:
		return shared.OSOKAsyncPhaseUpdate, true
	case zprsdk.OperationTypeDeleteZprPolicy:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func isZprPolicyWorkRequestResource(resource zprsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "zprpolicy", "zpr_policy", "zprpolicies", "zpr_policies", "policy", "policies":
		return true
	}
	if strings.Contains(entityType, "zprpolicy") || strings.Contains(entityType, "zpr_policy") {
		return true
	}

	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/zprpolicies/")
}

func desiredZprPolicyStatementsForUpdate(spec []string, current []string) []string {
	if spec != nil {
		return append([]string(nil), spec...)
	}
	if current != nil {
		return []string{}
	}
	return nil
}

func desiredZprPolicyFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredZprPolicyDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		converted := util.ConvertToOciDefinedTags(&spec)
		if converted == nil {
			return nil
		}
		return *converted
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func cloneOCIDefinedTags(values map[string]map[string]interface{}) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(values))
	for namespace, entries := range values {
		namespaceClone := make(map[string]interface{}, len(entries))
		for key, value := range entries {
			namespaceClone[key] = value
		}
		clone[namespace] = namespaceClone
	}
	return clone
}

func convertOCIToStatusDefinedTags(values map[string]map[string]interface{}) map[string]shared.MapValue {
	if values == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(values))
	for namespace, entries := range values {
		convertedEntries := make(shared.MapValue, len(entries))
		for key, value := range entries {
			convertedEntries[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedEntries
	}
	return converted
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}

func stringPtrEqual(current *string, desired string) bool {
	if current == nil {
		return desired == ""
	}
	return *current == desired
}
