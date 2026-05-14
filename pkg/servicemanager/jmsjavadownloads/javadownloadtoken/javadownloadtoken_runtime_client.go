/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package javadownloadtoken

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmsjavadownloadssdk "github.com/oracle/oci-go-sdk/v65/jmsjavadownloads"
	jmsjavadownloadsv1beta1 "github.com/oracle/oci-service-operator/api/jmsjavadownloads/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const javaDownloadTokenKind = "JavaDownloadToken"

var javaDownloadTokenWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(jmsjavadownloadssdk.OperationStatusAccepted),
		string(jmsjavadownloadssdk.OperationStatusInProgress),
		string(jmsjavadownloadssdk.OperationStatusWaiting),
		string(jmsjavadownloadssdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(jmsjavadownloadssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(jmsjavadownloadssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(jmsjavadownloadssdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(jmsjavadownloadssdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(jmsjavadownloadssdk.OperationTypeCreateJavaDownloadToken)},
	UpdateActionTokens:    []string{string(jmsjavadownloadssdk.OperationTypeUpdateJavaDownloadToken)},
	DeleteActionTokens:    []string{string(jmsjavadownloadssdk.OperationTypeDeleteJavaDownloadToken)},
}

type javaDownloadTokenOCIClient interface {
	CreateJavaDownloadToken(context.Context, jmsjavadownloadssdk.CreateJavaDownloadTokenRequest) (jmsjavadownloadssdk.CreateJavaDownloadTokenResponse, error)
	GetJavaDownloadToken(context.Context, jmsjavadownloadssdk.GetJavaDownloadTokenRequest) (jmsjavadownloadssdk.GetJavaDownloadTokenResponse, error)
	ListJavaDownloadTokens(context.Context, jmsjavadownloadssdk.ListJavaDownloadTokensRequest) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error)
	UpdateJavaDownloadToken(context.Context, jmsjavadownloadssdk.UpdateJavaDownloadTokenRequest) (jmsjavadownloadssdk.UpdateJavaDownloadTokenResponse, error)
	DeleteJavaDownloadToken(context.Context, jmsjavadownloadssdk.DeleteJavaDownloadTokenRequest) (jmsjavadownloadssdk.DeleteJavaDownloadTokenResponse, error)
	GetWorkRequest(context.Context, jmsjavadownloadssdk.GetWorkRequestRequest) (jmsjavadownloadssdk.GetWorkRequestResponse, error)
}

type javaDownloadTokenListCall func(context.Context, jmsjavadownloadssdk.ListJavaDownloadTokensRequest) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error)

func init() {
	registerJavaDownloadTokenRuntimeHooksMutator(func(manager *JavaDownloadTokenServiceManager, hooks *JavaDownloadTokenRuntimeHooks) {
		client, initErr := newJavaDownloadTokenSDKClient(manager)
		applyJavaDownloadTokenRuntimeHooks(hooks, client, initErr)
	})
}

func newJavaDownloadTokenSDKClient(manager *JavaDownloadTokenServiceManager) (javaDownloadTokenOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", javaDownloadTokenKind)
	}
	client, err := jmsjavadownloadssdk.NewJavaDownloadClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyJavaDownloadTokenRuntimeHooks(
	hooks *JavaDownloadTokenRuntimeHooks,
	client javaDownloadTokenOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedJavaDownloadTokenRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *jmsjavadownloadsv1beta1.JavaDownloadToken,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildJavaDownloadTokenUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardJavaDownloadTokenExistingBeforeCreate
	hooks.List.Fields = javaDownloadTokenListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listJavaDownloadTokensAllPages(hooks.List.Call)
	}
	hooks.Async.Adapter = javaDownloadTokenWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getJavaDownloadTokenWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveJavaDownloadTokenGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveJavaDownloadTokenGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverJavaDownloadTokenIDFromGeneratedWorkRequest
	hooks.Async.Message = javaDownloadTokenGeneratedWorkRequestMessage
}

func newJavaDownloadTokenServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client javaDownloadTokenOCIClient,
) JavaDownloadTokenServiceClient {
	manager := &JavaDownloadTokenServiceManager{Log: log}
	hooks := newJavaDownloadTokenRuntimeHooksWithOCIClient(client)
	applyJavaDownloadTokenRuntimeHooks(&hooks, client, nil)
	delegate := defaultJavaDownloadTokenServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*jmsjavadownloadsv1beta1.JavaDownloadToken](
			buildJavaDownloadTokenGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapJavaDownloadTokenGeneratedClient(hooks, delegate)
}

func newJavaDownloadTokenRuntimeHooksWithOCIClient(client javaDownloadTokenOCIClient) JavaDownloadTokenRuntimeHooks {
	hooks := newJavaDownloadTokenDefaultRuntimeHooks(jmsjavadownloadssdk.JavaDownloadClient{})
	hooks.Create.Call = func(ctx context.Context, request jmsjavadownloadssdk.CreateJavaDownloadTokenRequest) (jmsjavadownloadssdk.CreateJavaDownloadTokenResponse, error) {
		return client.CreateJavaDownloadToken(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request jmsjavadownloadssdk.GetJavaDownloadTokenRequest) (jmsjavadownloadssdk.GetJavaDownloadTokenResponse, error) {
		return client.GetJavaDownloadToken(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request jmsjavadownloadssdk.ListJavaDownloadTokensRequest) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error) {
		return client.ListJavaDownloadTokens(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request jmsjavadownloadssdk.UpdateJavaDownloadTokenRequest) (jmsjavadownloadssdk.UpdateJavaDownloadTokenResponse, error) {
		return client.UpdateJavaDownloadToken(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request jmsjavadownloadssdk.DeleteJavaDownloadTokenRequest) (jmsjavadownloadssdk.DeleteJavaDownloadTokenResponse, error) {
		return client.DeleteJavaDownloadToken(ctx, request)
	}
	return hooks
}

func reviewedJavaDownloadTokenRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newJavaDownloadTokenRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "id"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func javaDownloadTokenListFields() []generatedruntime.RequestField {
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
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func guardJavaDownloadTokenExistingBeforeCreate(
	_ context.Context,
	resource *jmsjavadownloadsv1beta1.JavaDownloadToken,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", javaDownloadTokenKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildJavaDownloadTokenUpdateBody(
	resource *jmsjavadownloadsv1beta1.JavaDownloadToken,
	currentResponse any,
) (jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails, bool, error) {
	if resource == nil {
		return jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails{}, false, fmt.Errorf("%s resource is nil", javaDownloadTokenKind)
	}

	desired, err := convertJavaDownloadTokenThroughJSON[jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails](resource.Spec)
	if err != nil {
		return jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails{}, false, err
	}
	current, err := javaDownloadTokenFromResponse(currentResponse)
	if err != nil {
		return jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails{}, false, err
	}

	details := jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails{}
	updateNeeded := false

	if desired.DisplayName != nil && stringValue(desired.DisplayName) != stringValue(current.DisplayName) {
		details.DisplayName = desired.DisplayName
		updateNeeded = true
	}
	if desired.Description != nil && stringValue(desired.Description) != stringValue(current.Description) {
		details.Description = desired.Description
		updateNeeded = true
	}
	if !boolPtrsEqual(common.Bool(resource.Spec.IsDefault), current.IsDefault) {
		details.IsDefault = common.Bool(resource.Spec.IsDefault)
		updateNeeded = true
	}
	if desired.TimeExpires != nil && !sdkTimePtrsEqual(desired.TimeExpires, current.TimeExpires) {
		details.TimeExpires = desired.TimeExpires
		updateNeeded = true
	}
	if len(desired.LicenseType) != 0 && !javaDownloadTokenJSONEqual(desired.LicenseType, current.LicenseType) {
		details.LicenseType = desired.LicenseType
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil {
		desiredFreeformTags := make(map[string]string, len(resource.Spec.FreeformTags))
		for key, value := range resource.Spec.FreeformTags {
			desiredFreeformTags[key] = value
		}
		if !javaDownloadTokenJSONEqual(desiredFreeformTags, current.FreeformTags) {
			details.FreeformTags = desiredFreeformTags
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := make(map[string]map[string]interface{}, len(resource.Spec.DefinedTags))
		for namespace, values := range resource.Spec.DefinedTags {
			converted := make(map[string]interface{}, len(values))
			for key, value := range values {
				converted[key] = value
			}
			desiredDefinedTags[namespace] = converted
		}
		if !javaDownloadTokenJSONEqual(desiredDefinedTags, current.DefinedTags) {
			details.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}

	return details, updateNeeded, nil
}

func listJavaDownloadTokensAllPages(call javaDownloadTokenListCall) javaDownloadTokenListCall {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request jmsjavadownloadssdk.ListJavaDownloadTokensRequest) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error) {
		var combined jmsjavadownloadssdk.ListJavaDownloadTokensResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return jmsjavadownloadssdk.ListJavaDownloadTokensResponse{}, err
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
}

func getJavaDownloadTokenWorkRequest(
	ctx context.Context,
	client javaDownloadTokenOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", javaDownloadTokenKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", javaDownloadTokenKind)
	}

	response, err := client.GetWorkRequest(ctx, jmsjavadownloadssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveJavaDownloadTokenGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := javaDownloadTokenWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveJavaDownloadTokenGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := javaDownloadTokenWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case jmsjavadownloadssdk.OperationTypeCreateJavaDownloadToken:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case jmsjavadownloadssdk.OperationTypeUpdateJavaDownloadToken:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case jmsjavadownloadssdk.OperationTypeDeleteJavaDownloadToken:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverJavaDownloadTokenIDFromGeneratedWorkRequest(
	_ *jmsjavadownloadsv1beta1.JavaDownloadToken,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := javaDownloadTokenWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := javaDownloadTokenWorkRequestActionForPhase(phase)
	if id, ok := resolveJavaDownloadTokenIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveJavaDownloadTokenIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a Java download token identifier", javaDownloadTokenKind, stringValue(current.Id))
}

func javaDownloadTokenGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := javaDownloadTokenWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", javaDownloadTokenKind, phase, stringValue(current.Id), current.Status)
}

func javaDownloadTokenFromResponse(currentResponse any) (jmsjavadownloadssdk.JavaDownloadToken, error) {
	switch current := currentResponse.(type) {
	case jmsjavadownloadssdk.JavaDownloadToken:
		return current, nil
	case *jmsjavadownloadssdk.JavaDownloadToken:
		if current == nil {
			return jmsjavadownloadssdk.JavaDownloadToken{}, fmt.Errorf("current %s response is nil", javaDownloadTokenKind)
		}
		return *current, nil
	case jmsjavadownloadssdk.JavaDownloadTokenSummary:
		return jmsjavadownloadssdk.JavaDownloadToken{
			Id:               current.Id,
			DisplayName:      current.DisplayName,
			CompartmentId:    current.CompartmentId,
			CreatedBy:        current.CreatedBy,
			Description:      current.Description,
			TimeCreated:      current.TimeCreated,
			TimeExpires:      current.TimeExpires,
			JavaVersion:      current.JavaVersion,
			LifecycleState:   current.LifecycleState,
			LastUpdatedBy:    current.LastUpdatedBy,
			TimeUpdated:      current.TimeUpdated,
			TimeLastUsed:     current.TimeLastUsed,
			LicenseType:      current.LicenseType,
			IsDefault:        current.IsDefault,
			LifecycleDetails: current.LifecycleDetails,
			FreeformTags:     current.FreeformTags,
			DefinedTags:      current.DefinedTags,
			SystemTags:       current.SystemTags,
		}, nil
	case *jmsjavadownloadssdk.JavaDownloadTokenSummary:
		if current == nil {
			return jmsjavadownloadssdk.JavaDownloadToken{}, fmt.Errorf("current %s response is nil", javaDownloadTokenKind)
		}
		return javaDownloadTokenFromResponse(*current)
	case jmsjavadownloadssdk.GetJavaDownloadTokenResponse:
		return current.JavaDownloadToken, nil
	case *jmsjavadownloadssdk.GetJavaDownloadTokenResponse:
		if current == nil {
			return jmsjavadownloadssdk.JavaDownloadToken{}, fmt.Errorf("current %s response is nil", javaDownloadTokenKind)
		}
		return current.JavaDownloadToken, nil
	case jmsjavadownloadssdk.CreateJavaDownloadTokenResponse:
		return current.JavaDownloadToken, nil
	case *jmsjavadownloadssdk.CreateJavaDownloadTokenResponse:
		if current == nil {
			return jmsjavadownloadssdk.JavaDownloadToken{}, fmt.Errorf("current %s response is nil", javaDownloadTokenKind)
		}
		return current.JavaDownloadToken, nil
	default:
		return jmsjavadownloadssdk.JavaDownloadToken{}, fmt.Errorf("unexpected current %s response type %T", javaDownloadTokenKind, currentResponse)
	}
}

func javaDownloadTokenWorkRequestFromAny(workRequest any) (jmsjavadownloadssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case jmsjavadownloadssdk.WorkRequest:
		return current, nil
	case *jmsjavadownloadssdk.WorkRequest:
		if current == nil {
			return jmsjavadownloadssdk.WorkRequest{}, fmt.Errorf("%s work request is nil", javaDownloadTokenKind)
		}
		return *current, nil
	default:
		return jmsjavadownloadssdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", javaDownloadTokenKind, workRequest)
	}
}

func javaDownloadTokenWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) jmsjavadownloadssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return jmsjavadownloadssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return jmsjavadownloadssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return jmsjavadownloadssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveJavaDownloadTokenIDFromResources(
	resources []jmsjavadownloadssdk.WorkRequestResource,
	action jmsjavadownloadssdk.ActionTypeEnum,
	preferJavaDownloadTokenOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferJavaDownloadTokenOnly && !isJavaDownloadTokenWorkRequestResource(resource) {
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

func isJavaDownloadTokenWorkRequestResource(resource jmsjavadownloadssdk.WorkRequestResource) bool {
	return normalizeJavaDownloadTokenWorkRequestToken(stringValue(resource.EntityType)) == "javadownloadtoken"
}

func normalizeJavaDownloadTokenWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func convertJavaDownloadTokenThroughJSON[T any](source any) (T, error) {
	var out T
	payload, err := json.Marshal(source)
	if err != nil {
		return out, fmt.Errorf("marshal %s payload: %w", javaDownloadTokenKind, err)
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, fmt.Errorf("unmarshal %s payload: %w", javaDownloadTokenKind, err)
	}
	return out, nil
}

func javaDownloadTokenJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func boolPtrsEqual(left *bool, right *bool) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func sdkTimePtrsEqual(left *common.SDKTime, right *common.SDKTime) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Time.Equal(right.Time)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
