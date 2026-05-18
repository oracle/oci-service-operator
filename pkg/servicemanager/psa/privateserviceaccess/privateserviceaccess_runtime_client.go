/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateserviceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	psasdk "github.com/oracle/oci-go-sdk/v65/psa"
	psav1beta1 "github.com/oracle/oci-service-operator/api/psa/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var privateServiceAccessWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(psasdk.OperationStatusAccepted),
		string(psasdk.OperationStatusInProgress),
		string(psasdk.OperationStatusWaiting),
		string(psasdk.OperationStatusCancelling),
	},
	SucceededStatusTokens: []string{string(psasdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(psasdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(psasdk.OperationStatusCancelled)},
	AttentionStatusTokens: []string{string(psasdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(psasdk.OperationTypeCreatePrivateServiceAccess)},
	UpdateActionTokens:    []string{string(psasdk.OperationTypeUpdatePrivateServiceAccess)},
	DeleteActionTokens:    []string{string(psasdk.OperationTypeDeletePrivateServiceAccess)},
}

type privateServiceAccessOCIClient interface {
	CreatePrivateServiceAccess(context.Context, psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error)
	GetPrivateServiceAccess(context.Context, psasdk.GetPrivateServiceAccessRequest) (psasdk.GetPrivateServiceAccessResponse, error)
	ListPrivateServiceAccesses(context.Context, psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error)
	UpdatePrivateServiceAccess(context.Context, psasdk.UpdatePrivateServiceAccessRequest) (psasdk.UpdatePrivateServiceAccessResponse, error)
	DeletePrivateServiceAccess(context.Context, psasdk.DeletePrivateServiceAccessRequest) (psasdk.DeletePrivateServiceAccessResponse, error)
	GetPsaWorkRequest(context.Context, psasdk.GetPsaWorkRequestRequest) (psasdk.GetPsaWorkRequestResponse, error)
}

func init() {
	registerPrivateServiceAccessRuntimeHooksMutator(func(
		manager *PrivateServiceAccessServiceManager,
		hooks *PrivateServiceAccessRuntimeHooks,
	) {
		client, initErr := newPrivateServiceAccessSDKClient(manager)
		applyPrivateServiceAccessRuntimeHooks(hooks, client, initErr)
	})
}

func newPrivateServiceAccessSDKClient(
	manager *PrivateServiceAccessServiceManager,
) (privateServiceAccessOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("PrivateServiceAccess service manager is nil")
	}
	client, err := psasdk.NewPrivateServiceAccessClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyPrivateServiceAccessRuntimeHooks(
	hooks *PrivateServiceAccessRuntimeHooks,
	client privateServiceAccessOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedPrivateServiceAccessRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *psav1beta1.PrivateServiceAccess,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildPrivateServiceAccessUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardPrivateServiceAccessExistingBeforeCreate
	hooks.Create.Fields = privateServiceAccessCreateFields()
	hooks.Get.Fields = privateServiceAccessGetFields()
	hooks.List.Fields = privateServiceAccessListFields()
	hooks.Update.Fields = privateServiceAccessUpdateFields()
	hooks.Delete.Fields = privateServiceAccessDeleteFields()
	hooks.Async.Adapter = privateServiceAccessWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getPrivateServiceAccessWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolvePrivateServiceAccessGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolvePrivateServiceAccessGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverPrivateServiceAccessIDFromGeneratedWorkRequest
	hooks.Async.Message = privateServiceAccessGeneratedWorkRequestMessage
}

func newPrivateServiceAccessServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client privateServiceAccessOCIClient,
) PrivateServiceAccessServiceClient {
	return defaultPrivateServiceAccessServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*psav1beta1.PrivateServiceAccess](
			newPrivateServiceAccessRuntimeConfig(log, client),
		),
	}
}

func newPrivateServiceAccessRuntimeConfig(
	log loggerutil.OSOKLogger,
	client privateServiceAccessOCIClient,
) generatedruntime.Config[*psav1beta1.PrivateServiceAccess] {
	hooks := newPrivateServiceAccessRuntimeHooksWithOCIClient(client)
	applyPrivateServiceAccessRuntimeHooks(&hooks, client, nil)
	return buildPrivateServiceAccessGeneratedRuntimeConfig(&PrivateServiceAccessServiceManager{Log: log}, hooks)
}

func newPrivateServiceAccessRuntimeHooksWithOCIClient(
	client privateServiceAccessOCIClient,
) PrivateServiceAccessRuntimeHooks {
	return PrivateServiceAccessRuntimeHooks{
		Semantics: newPrivateServiceAccessRuntimeSemantics(),
		Create: runtimeOperationHooks[psasdk.CreatePrivateServiceAccessRequest, psasdk.CreatePrivateServiceAccessResponse]{
			Fields: privateServiceAccessCreateFields(),
			Call: func(ctx context.Context, request psasdk.CreatePrivateServiceAccessRequest) (psasdk.CreatePrivateServiceAccessResponse, error) {
				return client.CreatePrivateServiceAccess(ctx, request)
			},
		},
		Get: runtimeOperationHooks[psasdk.GetPrivateServiceAccessRequest, psasdk.GetPrivateServiceAccessResponse]{
			Fields: privateServiceAccessGetFields(),
			Call: func(ctx context.Context, request psasdk.GetPrivateServiceAccessRequest) (psasdk.GetPrivateServiceAccessResponse, error) {
				return client.GetPrivateServiceAccess(ctx, request)
			},
		},
		List: runtimeOperationHooks[psasdk.ListPrivateServiceAccessesRequest, psasdk.ListPrivateServiceAccessesResponse]{
			Fields: privateServiceAccessListFields(),
			Call: func(ctx context.Context, request psasdk.ListPrivateServiceAccessesRequest) (psasdk.ListPrivateServiceAccessesResponse, error) {
				return client.ListPrivateServiceAccesses(ctx, request)
			},
		},
		Update: runtimeOperationHooks[psasdk.UpdatePrivateServiceAccessRequest, psasdk.UpdatePrivateServiceAccessResponse]{
			Fields: privateServiceAccessUpdateFields(),
			Call: func(ctx context.Context, request psasdk.UpdatePrivateServiceAccessRequest) (psasdk.UpdatePrivateServiceAccessResponse, error) {
				return client.UpdatePrivateServiceAccess(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[psasdk.DeletePrivateServiceAccessRequest, psasdk.DeletePrivateServiceAccessResponse]{
			Fields: privateServiceAccessDeleteFields(),
			Call: func(ctx context.Context, request psasdk.DeletePrivateServiceAccessRequest) (psasdk.DeletePrivateServiceAccessResponse, error) {
				return client.DeletePrivateServiceAccess(ctx, request)
			},
		},
	}
}

func reviewedPrivateServiceAccessRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newPrivateServiceAccessRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "serviceId", "subnetId", "ipv4Ip"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func privateServiceAccessCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreatePrivateServiceAccessDetails", RequestName: "CreatePrivateServiceAccessDetails", Contribution: "body"},
	}
}

func privateServiceAccessGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "PrivateServiceAccessId", RequestName: "privateServiceAccessId", Contribution: "path", PreferResourceID: true},
	}
}

func privateServiceAccessListFields() []generatedruntime.RequestField {
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
			FieldName:    "ServiceId",
			RequestName:  "serviceId",
			Contribution: "query",
			LookupPaths:  []string{"status.serviceId", "spec.serviceId", "serviceId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func privateServiceAccessUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "PrivateServiceAccessId", RequestName: "privateServiceAccessId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdatePrivateServiceAccessDetails", RequestName: "UpdatePrivateServiceAccessDetails", Contribution: "body"},
	}
}

func privateServiceAccessDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "PrivateServiceAccessId", RequestName: "privateServiceAccessId", Contribution: "path", PreferResourceID: true},
	}
}

func guardPrivateServiceAccessExistingBeforeCreate(
	_ context.Context,
	resource *psav1beta1.PrivateServiceAccess,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("PrivateServiceAccess resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.SubnetId) == "" ||
		strings.TrimSpace(resource.Spec.ServiceId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildPrivateServiceAccessUpdateBody(
	resource *psav1beta1.PrivateServiceAccess,
	currentResponse any,
) (psasdk.UpdatePrivateServiceAccessDetails, bool, error) {
	if resource == nil {
		return psasdk.UpdatePrivateServiceAccessDetails{}, false, fmt.Errorf("PrivateServiceAccess resource is nil")
	}

	current, err := privateServiceAccessFromResponse(currentResponse)
	if err != nil {
		return psasdk.UpdatePrivateServiceAccessDetails{}, false, err
	}

	details := psasdk.UpdatePrivateServiceAccessDetails{}
	updateNeeded := false

	if desired, ok := privateServiceAccessDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := privateServiceAccessDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := privateServiceAccessDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := privateServiceAccessDesiredNestedMapUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if desired, ok := privateServiceAccessDesiredNestedMapUpdate(resource.Spec.SecurityAttributes, current.SecurityAttributes); ok {
		details.SecurityAttributes = desired
		updateNeeded = true
	}
	if desired, ok := privateServiceAccessDesiredStringSliceUpdate(resource.Spec.NsgIds, current.NsgIds); ok {
		details.NsgIds = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func privateServiceAccessFromResponse(currentResponse any) (psasdk.PrivateServiceAccess, error) {
	switch current := currentResponse.(type) {
	case psasdk.PrivateServiceAccess:
		return current, nil
	case *psasdk.PrivateServiceAccess:
		if current == nil {
			return psasdk.PrivateServiceAccess{}, fmt.Errorf("current PrivateServiceAccess response is nil")
		}
		return *current, nil
	case psasdk.PrivateServiceAccessSummary:
		return privateServiceAccessFromSummary(current), nil
	case *psasdk.PrivateServiceAccessSummary:
		if current == nil {
			return psasdk.PrivateServiceAccess{}, fmt.Errorf("current PrivateServiceAccess response is nil")
		}
		return privateServiceAccessFromSummary(*current), nil
	case psasdk.CreatePrivateServiceAccessResponse:
		return current.PrivateServiceAccess, nil
	case *psasdk.CreatePrivateServiceAccessResponse:
		if current == nil {
			return psasdk.PrivateServiceAccess{}, fmt.Errorf("current PrivateServiceAccess response is nil")
		}
		return current.PrivateServiceAccess, nil
	case psasdk.GetPrivateServiceAccessResponse:
		return current.PrivateServiceAccess, nil
	case *psasdk.GetPrivateServiceAccessResponse:
		if current == nil {
			return psasdk.PrivateServiceAccess{}, fmt.Errorf("current PrivateServiceAccess response is nil")
		}
		return current.PrivateServiceAccess, nil
	default:
		return psasdk.PrivateServiceAccess{}, fmt.Errorf("unexpected current PrivateServiceAccess response type %T", currentResponse)
	}
}

func privateServiceAccessFromSummary(
	summary psasdk.PrivateServiceAccessSummary,
) psasdk.PrivateServiceAccess {
	return psasdk.PrivateServiceAccess{
		CompartmentId:      summary.CompartmentId,
		DisplayName:        summary.DisplayName,
		Id:                 summary.Id,
		VcnId:              summary.VcnId,
		SubnetId:           summary.SubnetId,
		VnicId:             summary.VnicId,
		LifecycleState:     summary.LifecycleState,
		ServiceId:          summary.ServiceId,
		Fqdns:              append([]string(nil), summary.Fqdns...),
		DefinedTags:        summary.DefinedTags,
		FreeformTags:       summary.FreeformTags,
		SystemTags:         summary.SystemTags,
		SecurityAttributes: summary.SecurityAttributes,
		Description:        summary.Description,
		TimeCreated:        summary.TimeCreated,
		TimeUpdated:        summary.TimeUpdated,
		NsgIds:             append([]string(nil), summary.NsgIds...),
		Ipv4Ip:             summary.Ipv4Ip,
	}
}

func privateServiceAccessDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func privateServiceAccessDesiredStringSliceUpdate(spec []string, current []string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if slices.Equal(spec, current) {
		return nil, false
	}
	return append([]string(nil), spec...), true
}

func privateServiceAccessDesiredFreeformTagsUpdate(
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

func privateServiceAccessDesiredNestedMapUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := privateServiceAccessNestedMapFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if privateServiceAccessJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func privateServiceAccessNestedMapFromSpec(
	spec map[string]shared.MapValue,
) map[string]map[string]interface{} {
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

func privateServiceAccessJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getPrivateServiceAccessWorkRequest(
	ctx context.Context,
	client privateServiceAccessOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize PrivateServiceAccess OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("PrivateServiceAccess OCI client is not configured")
	}

	response, err := client.GetPsaWorkRequest(ctx, psasdk.GetPsaWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolvePrivateServiceAccessGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := privateServiceAccessWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolvePrivateServiceAccessGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := privateServiceAccessWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := privateServiceAccessWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverPrivateServiceAccessIDFromGeneratedWorkRequest(
	_ *psav1beta1.PrivateServiceAccess,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := privateServiceAccessWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := privateServiceAccessWorkRequestActionForPhase(phase)
	if id, ok := resolvePrivateServiceAccessIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolvePrivateServiceAccessIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("PrivateServiceAccess work request %s does not expose a private service access identifier", stringValue(current.Id))
}

func privateServiceAccessGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := privateServiceAccessWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("PrivateServiceAccess %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func privateServiceAccessWorkRequestFromAny(workRequest any) (psasdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case psasdk.WorkRequest:
		return current, nil
	case *psasdk.WorkRequest:
		if current == nil {
			return psasdk.WorkRequest{}, fmt.Errorf("PrivateServiceAccess work request is nil")
		}
		return *current, nil
	default:
		return psasdk.WorkRequest{}, fmt.Errorf("unexpected PrivateServiceAccess work request type %T", workRequest)
	}
}

func privateServiceAccessWorkRequestPhaseFromOperationType(
	operationType psasdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case psasdk.OperationTypeCreatePrivateServiceAccess:
		return shared.OSOKAsyncPhaseCreate, true
	case psasdk.OperationTypeUpdatePrivateServiceAccess:
		return shared.OSOKAsyncPhaseUpdate, true
	case psasdk.OperationTypeDeletePrivateServiceAccess:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func privateServiceAccessWorkRequestActionForPhase(
	phase shared.OSOKAsyncPhase,
) psasdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return psasdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return psasdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return psasdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolvePrivateServiceAccessIDFromResources(
	resources []psasdk.WorkRequestResource,
	action psasdk.ActionTypeEnum,
	preferPrivateServiceAccessOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferPrivateServiceAccessOnly && !isPrivateServiceAccessWorkRequestResource(resource) {
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

func isPrivateServiceAccessWorkRequestResource(resource psasdk.WorkRequestResource) bool {
	entityType := normalizePrivateServiceAccessWorkRequestToken(stringValue(resource.EntityType))
	if strings.Contains(entityType, "privateserviceaccess") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/privateserviceaccess/")
}

func normalizePrivateServiceAccessWorkRequestToken(value string) string {
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
