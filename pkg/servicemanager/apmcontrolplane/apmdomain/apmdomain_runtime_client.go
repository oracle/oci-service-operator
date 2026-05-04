/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apmdomain

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	apmcontrolplanesdk "github.com/oracle/oci-go-sdk/v65/apmcontrolplane"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmcontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/apmcontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var apmDomainWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(apmcontrolplanesdk.OperationStatusAccepted),
		string(apmcontrolplanesdk.OperationStatusInProgress),
		string(apmcontrolplanesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(apmcontrolplanesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(apmcontrolplanesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(apmcontrolplanesdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(apmcontrolplanesdk.OperationTypesCreateApmDomain)},
	UpdateActionTokens:    []string{string(apmcontrolplanesdk.OperationTypesUpdateApmDomain)},
	DeleteActionTokens:    []string{string(apmcontrolplanesdk.OperationTypesDeleteApmDomain)},
}

type apmDomainOCIClient interface {
	CreateApmDomain(context.Context, apmcontrolplanesdk.CreateApmDomainRequest) (apmcontrolplanesdk.CreateApmDomainResponse, error)
	GetApmDomain(context.Context, apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error)
	ListApmDomains(context.Context, apmcontrolplanesdk.ListApmDomainsRequest) (apmcontrolplanesdk.ListApmDomainsResponse, error)
	UpdateApmDomain(context.Context, apmcontrolplanesdk.UpdateApmDomainRequest) (apmcontrolplanesdk.UpdateApmDomainResponse, error)
	DeleteApmDomain(context.Context, apmcontrolplanesdk.DeleteApmDomainRequest) (apmcontrolplanesdk.DeleteApmDomainResponse, error)
	GetWorkRequest(context.Context, apmcontrolplanesdk.GetWorkRequestRequest) (apmcontrolplanesdk.GetWorkRequestResponse, error)
}

func init() {
	registerApmDomainRuntimeHooksMutator(func(manager *ApmDomainServiceManager, hooks *ApmDomainRuntimeHooks) {
		client, initErr := newApmDomainSDKClient(manager)
		applyApmDomainRuntimeHooks(hooks, client, initErr)
	})
}

func newApmDomainSDKClient(manager *ApmDomainServiceManager) (apmDomainOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ApmDomain service manager is nil")
	}

	client, err := apmcontrolplanesdk.NewApmDomainClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyApmDomainRuntimeHooks(
	hooks *ApmDomainRuntimeHooks,
	client apmDomainOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedApmDomainRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *apmcontrolplanev1beta1.ApmDomain,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildApmDomainUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardApmDomainExistingBeforeCreate
	hooks.Create.Fields = apmDomainCreateFields()
	hooks.Get.Fields = apmDomainGetFields()
	hooks.List.Fields = apmDomainListFields()
	hooks.Update.Fields = apmDomainUpdateFields()
	hooks.Delete.Fields = apmDomainDeleteFields()
	hooks.Async.Adapter = apmDomainWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getApmDomainWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveApmDomainGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveApmDomainGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverApmDomainIDFromGeneratedWorkRequest
	hooks.Async.Message = apmDomainGeneratedWorkRequestMessage
}

func newApmDomainServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client apmDomainOCIClient,
) ApmDomainServiceClient {
	return defaultApmDomainServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apmcontrolplanev1beta1.ApmDomain](
			newApmDomainRuntimeConfig(log, client),
		),
	}
}

func newApmDomainRuntimeConfig(
	log loggerutil.OSOKLogger,
	client apmDomainOCIClient,
) generatedruntime.Config[*apmcontrolplanev1beta1.ApmDomain] {
	hooks := newApmDomainRuntimeHooksWithOCIClient(client)
	applyApmDomainRuntimeHooks(&hooks, client, nil)
	return buildApmDomainGeneratedRuntimeConfig(&ApmDomainServiceManager{Log: log}, hooks)
}

func newApmDomainRuntimeHooksWithOCIClient(client apmDomainOCIClient) ApmDomainRuntimeHooks {
	return ApmDomainRuntimeHooks{
		Semantics: newApmDomainRuntimeSemantics(),
		Create: runtimeOperationHooks[apmcontrolplanesdk.CreateApmDomainRequest, apmcontrolplanesdk.CreateApmDomainResponse]{
			Fields: apmDomainCreateFields(),
			Call: func(ctx context.Context, request apmcontrolplanesdk.CreateApmDomainRequest) (apmcontrolplanesdk.CreateApmDomainResponse, error) {
				return client.CreateApmDomain(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apmcontrolplanesdk.GetApmDomainRequest, apmcontrolplanesdk.GetApmDomainResponse]{
			Fields: apmDomainGetFields(),
			Call: func(ctx context.Context, request apmcontrolplanesdk.GetApmDomainRequest) (apmcontrolplanesdk.GetApmDomainResponse, error) {
				return client.GetApmDomain(ctx, request)
			},
		},
		List: runtimeOperationHooks[apmcontrolplanesdk.ListApmDomainsRequest, apmcontrolplanesdk.ListApmDomainsResponse]{
			Fields: apmDomainListFields(),
			Call: func(ctx context.Context, request apmcontrolplanesdk.ListApmDomainsRequest) (apmcontrolplanesdk.ListApmDomainsResponse, error) {
				return client.ListApmDomains(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apmcontrolplanesdk.UpdateApmDomainRequest, apmcontrolplanesdk.UpdateApmDomainResponse]{
			Fields: apmDomainUpdateFields(),
			Call: func(ctx context.Context, request apmcontrolplanesdk.UpdateApmDomainRequest) (apmcontrolplanesdk.UpdateApmDomainResponse, error) {
				return client.UpdateApmDomain(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apmcontrolplanesdk.DeleteApmDomainRequest, apmcontrolplanesdk.DeleteApmDomainResponse]{
			Fields: apmDomainDeleteFields(),
			Call: func(ctx context.Context, request apmcontrolplanesdk.DeleteApmDomainRequest) (apmcontrolplanesdk.DeleteApmDomainResponse, error) {
				return client.DeleteApmDomain(ctx, request)
			},
		},
	}
}

func reviewedApmDomainRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newApmDomainRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func apmDomainCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateApmDomainDetails", RequestName: "CreateApmDomainDetails", Contribution: "body"},
	}
}

func apmDomainGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ApmDomainId", RequestName: "apmDomainId", Contribution: "path", PreferResourceID: true},
	}
}

func apmDomainListFields() []generatedruntime.RequestField {
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

func apmDomainUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ApmDomainId", RequestName: "apmDomainId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateApmDomainDetails", RequestName: "UpdateApmDomainDetails", Contribution: "body"},
	}
}

func apmDomainDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ApmDomainId", RequestName: "apmDomainId", Contribution: "path", PreferResourceID: true},
	}
}

func guardApmDomainExistingBeforeCreate(
	_ context.Context,
	resource *apmcontrolplanev1beta1.ApmDomain,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ApmDomain resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildApmDomainUpdateBody(
	resource *apmcontrolplanev1beta1.ApmDomain,
	currentResponse any,
) (apmcontrolplanesdk.UpdateApmDomainDetails, bool, error) {
	if resource == nil {
		return apmcontrolplanesdk.UpdateApmDomainDetails{}, false, fmt.Errorf("ApmDomain resource is nil")
	}

	current, err := apmDomainFromResponse(currentResponse)
	if err != nil {
		return apmcontrolplanesdk.UpdateApmDomainDetails{}, false, err
	}

	details := apmcontrolplanesdk.UpdateApmDomainDetails{}
	updateNeeded := false

	if desired, ok := apmDomainDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := apmDomainDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := apmDomainDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := apmDomainDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func apmDomainFromResponse(currentResponse any) (apmcontrolplanesdk.ApmDomain, error) {
	switch current := currentResponse.(type) {
	case apmcontrolplanesdk.ApmDomain:
		return current, nil
	case *apmcontrolplanesdk.ApmDomain:
		if current == nil {
			return apmcontrolplanesdk.ApmDomain{}, fmt.Errorf("current ApmDomain response is nil")
		}
		return *current, nil
	case apmcontrolplanesdk.ApmDomainSummary:
		return apmcontrolplanesdk.ApmDomain{
			Id:             current.Id,
			DisplayName:    current.DisplayName,
			CompartmentId:  current.CompartmentId,
			Description:    current.Description,
			LifecycleState: current.LifecycleState,
			IsFreeTier:     current.IsFreeTier,
			TimeCreated:    current.TimeCreated,
			TimeUpdated:    current.TimeUpdated,
			FreeformTags:   current.FreeformTags,
			DefinedTags:    current.DefinedTags,
		}, nil
	case *apmcontrolplanesdk.ApmDomainSummary:
		if current == nil {
			return apmcontrolplanesdk.ApmDomain{}, fmt.Errorf("current ApmDomain response is nil")
		}
		return apmDomainFromResponse(*current)
	case apmcontrolplanesdk.GetApmDomainResponse:
		return current.ApmDomain, nil
	case *apmcontrolplanesdk.GetApmDomainResponse:
		if current == nil {
			return apmcontrolplanesdk.ApmDomain{}, fmt.Errorf("current ApmDomain response is nil")
		}
		return current.ApmDomain, nil
	default:
		return apmcontrolplanesdk.ApmDomain{}, fmt.Errorf("unexpected current ApmDomain response type %T", currentResponse)
	}
}

func apmDomainDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func apmDomainDesiredFreeformTagsUpdate(
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

func apmDomainDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := apmDomainDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if apmDomainJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func apmDomainDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func apmDomainJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getApmDomainWorkRequest(
	ctx context.Context,
	client apmDomainOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize ApmDomain OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("ApmDomain OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, apmcontrolplanesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveApmDomainGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := apmDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveApmDomainGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := apmDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := apmDomainWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverApmDomainIDFromGeneratedWorkRequest(
	_ *apmcontrolplanev1beta1.ApmDomain,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := apmDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := apmDomainWorkRequestActionForPhase(phase)
	if id, ok := resolveApmDomainIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveApmDomainIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("ApmDomain work request %s does not expose an ApmDomain identifier", stringValue(current.Id))
}

func apmDomainGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := apmDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ApmDomain %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func apmDomainWorkRequestFromAny(workRequest any) (apmcontrolplanesdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case apmcontrolplanesdk.WorkRequest:
		return current, nil
	case *apmcontrolplanesdk.WorkRequest:
		if current == nil {
			return apmcontrolplanesdk.WorkRequest{}, fmt.Errorf("ApmDomain work request is nil")
		}
		return *current, nil
	default:
		return apmcontrolplanesdk.WorkRequest{}, fmt.Errorf("unexpected ApmDomain work request type %T", workRequest)
	}
}

func apmDomainWorkRequestPhaseFromOperationType(operationType apmcontrolplanesdk.OperationTypesEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case apmcontrolplanesdk.OperationTypesCreateApmDomain:
		return shared.OSOKAsyncPhaseCreate, true
	case apmcontrolplanesdk.OperationTypesUpdateApmDomain:
		return shared.OSOKAsyncPhaseUpdate, true
	case apmcontrolplanesdk.OperationTypesDeleteApmDomain:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func apmDomainWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) apmcontrolplanesdk.ActionTypesEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return apmcontrolplanesdk.ActionTypesCreated
	case shared.OSOKAsyncPhaseUpdate:
		return apmcontrolplanesdk.ActionTypesUpdated
	case shared.OSOKAsyncPhaseDelete:
		return apmcontrolplanesdk.ActionTypesDeleted
	default:
		return ""
	}
}

func resolveApmDomainIDFromResources(
	resources []apmcontrolplanesdk.WorkRequestResource,
	action apmcontrolplanesdk.ActionTypesEnum,
	preferApmDomainOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferApmDomainOnly && !isApmDomainWorkRequestResource(resource) {
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

func isApmDomainWorkRequestResource(resource apmcontrolplanesdk.WorkRequestResource) bool {
	return normalizeApmDomainWorkRequestToken(stringValue(resource.EntityType)) == "apmdomain"
}

func normalizeApmDomainWorkRequestToken(value string) string {
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
	return strings.TrimSpace(*value)
}
