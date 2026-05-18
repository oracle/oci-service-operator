/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vbsinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	vbsinstsdk "github.com/oracle/oci-go-sdk/v65/vbsinst"
	vbsinstv1beta1 "github.com/oracle/oci-service-operator/api/vbsinst/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var vbsInstanceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(vbsinstsdk.OperationStatusAccepted),
		string(vbsinstsdk.OperationStatusInProgress),
		string(vbsinstsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(vbsinstsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(vbsinstsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(vbsinstsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(vbsinstsdk.OperationTypeCreateVbsInstance)},
	UpdateActionTokens:    []string{string(vbsinstsdk.OperationTypeUpdateVbsInstance)},
	DeleteActionTokens:    []string{string(vbsinstsdk.OperationTypeDeleteVbsInstance)},
}

type vbsInstanceOCIClient interface {
	CreateVbsInstance(context.Context, vbsinstsdk.CreateVbsInstanceRequest) (vbsinstsdk.CreateVbsInstanceResponse, error)
	GetVbsInstance(context.Context, vbsinstsdk.GetVbsInstanceRequest) (vbsinstsdk.GetVbsInstanceResponse, error)
	ListVbsInstances(context.Context, vbsinstsdk.ListVbsInstancesRequest) (vbsinstsdk.ListVbsInstancesResponse, error)
	UpdateVbsInstance(context.Context, vbsinstsdk.UpdateVbsInstanceRequest) (vbsinstsdk.UpdateVbsInstanceResponse, error)
	DeleteVbsInstance(context.Context, vbsinstsdk.DeleteVbsInstanceRequest) (vbsinstsdk.DeleteVbsInstanceResponse, error)
	GetWorkRequest(context.Context, vbsinstsdk.GetWorkRequestRequest) (vbsinstsdk.GetWorkRequestResponse, error)
}

type vbsInstanceObserved struct {
	ID                              string
	Name                            string
	DisplayName                     string
	CompartmentID                   string
	IsResourceUsageAgreementGranted *bool
	ResourceCompartmentID           string
	VbsAccessURL                    string
	TimeCreated                     *common.SDKTime
	TimeUpdated                     *common.SDKTime
	LifecycleState                  string
	LifecycleDetails                string
	FreeformTags                    map[string]string
	DefinedTags                     map[string]map[string]interface{}
	SystemTags                      map[string]map[string]interface{}
}

func init() {
	registerVbsInstanceRuntimeHooksMutator(func(manager *VbsInstanceServiceManager, hooks *VbsInstanceRuntimeHooks) {
		client, initErr := newVbsInstanceSDKClient(manager)
		applyVbsInstanceRuntimeHooks(hooks, client, initErr)
	})
}

func newVbsInstanceSDKClient(manager *VbsInstanceServiceManager) (vbsInstanceOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("VbsInstance service manager is nil")
	}

	client, err := vbsinstsdk.NewVbsInstanceClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyVbsInstanceRuntimeHooks(
	hooks *VbsInstanceRuntimeHooks,
	client vbsInstanceOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedVbsInstanceRuntimeSemantics()
	hooks.StatusHooks.ProjectStatus = projectVbsInstanceStatus
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *vbsinstv1beta1.VbsInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildVbsInstanceUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardVbsInstanceExistingBeforeCreate
	hooks.Create.Fields = vbsInstanceCreateFields()
	hooks.Get.Fields = vbsInstanceGetFields()
	hooks.List.Fields = vbsInstanceListFields()
	hooks.Update.Fields = vbsInstanceUpdateFields()
	hooks.Delete.Fields = vbsInstanceDeleteFields()
	hooks.Async.Adapter = vbsInstanceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getVbsInstanceWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveVbsInstanceGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveVbsInstanceGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverVbsInstanceIDFromGeneratedWorkRequest
	hooks.Async.Message = vbsInstanceGeneratedWorkRequestMessage
}

func newVbsInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client vbsInstanceOCIClient,
) VbsInstanceServiceClient {
	return defaultVbsInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*vbsinstv1beta1.VbsInstance](
			newVbsInstanceRuntimeConfig(log, client),
		),
	}
}

func newVbsInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client vbsInstanceOCIClient,
) generatedruntime.Config[*vbsinstv1beta1.VbsInstance] {
	hooks := newVbsInstanceRuntimeHooksWithOCIClient(client)
	applyVbsInstanceRuntimeHooks(&hooks, client, nil)
	return buildVbsInstanceGeneratedRuntimeConfig(&VbsInstanceServiceManager{Log: log}, hooks)
}

func newVbsInstanceRuntimeHooksWithOCIClient(client vbsInstanceOCIClient) VbsInstanceRuntimeHooks {
	return VbsInstanceRuntimeHooks{
		Semantics: reviewedVbsInstanceRuntimeSemantics(),
		Create: runtimeOperationHooks[vbsinstsdk.CreateVbsInstanceRequest, vbsinstsdk.CreateVbsInstanceResponse]{
			Fields: vbsInstanceCreateFields(),
			Call: func(ctx context.Context, request vbsinstsdk.CreateVbsInstanceRequest) (vbsinstsdk.CreateVbsInstanceResponse, error) {
				return client.CreateVbsInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[vbsinstsdk.GetVbsInstanceRequest, vbsinstsdk.GetVbsInstanceResponse]{
			Fields: vbsInstanceGetFields(),
			Call: func(ctx context.Context, request vbsinstsdk.GetVbsInstanceRequest) (vbsinstsdk.GetVbsInstanceResponse, error) {
				return client.GetVbsInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[vbsinstsdk.ListVbsInstancesRequest, vbsinstsdk.ListVbsInstancesResponse]{
			Fields: vbsInstanceListFields(),
			Call: func(ctx context.Context, request vbsinstsdk.ListVbsInstancesRequest) (vbsinstsdk.ListVbsInstancesResponse, error) {
				return client.ListVbsInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[vbsinstsdk.UpdateVbsInstanceRequest, vbsinstsdk.UpdateVbsInstanceResponse]{
			Fields: vbsInstanceUpdateFields(),
			Call: func(ctx context.Context, request vbsinstsdk.UpdateVbsInstanceRequest) (vbsinstsdk.UpdateVbsInstanceResponse, error) {
				return client.UpdateVbsInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[vbsinstsdk.DeleteVbsInstanceRequest, vbsinstsdk.DeleteVbsInstanceResponse]{
			Fields: vbsInstanceDeleteFields(),
			Call: func(ctx context.Context, request vbsinstsdk.DeleteVbsInstanceRequest) (vbsinstsdk.DeleteVbsInstanceResponse, error) {
				return client.DeleteVbsInstance(ctx, request)
			},
		},
	}
}

func reviewedVbsInstanceRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newVbsInstanceRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "name", "id"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func vbsInstanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateVbsInstanceDetails", RequestName: "CreateVbsInstanceDetails", Contribution: "body"},
	}
}

func vbsInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "VbsInstanceId", RequestName: "vbsInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func vbsInstanceListFields() []generatedruntime.RequestField {
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
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func vbsInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "VbsInstanceId", RequestName: "vbsInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateVbsInstanceDetails", RequestName: "UpdateVbsInstanceDetails", Contribution: "body"},
	}
}

func vbsInstanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "VbsInstanceId", RequestName: "vbsInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func guardVbsInstanceExistingBeforeCreate(
	_ context.Context,
	resource *vbsinstv1beta1.VbsInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("VbsInstance resource is nil")
	}

	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.Name) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildVbsInstanceUpdateBody(
	resource *vbsinstv1beta1.VbsInstance,
	currentResponse any,
) (vbsinstsdk.UpdateVbsInstanceDetails, bool, error) {
	if resource == nil {
		return vbsinstsdk.UpdateVbsInstanceDetails{}, false, fmt.Errorf("VbsInstance resource is nil")
	}

	current, err := vbsInstanceObservedFromResponse(currentResponse)
	if err != nil {
		return vbsinstsdk.UpdateVbsInstanceDetails{}, false, err
	}

	details := vbsinstsdk.UpdateVbsInstanceDetails{}
	updateNeeded := false

	if desired, ok := vbsInstanceDesiredDisplayNameUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := vbsInstanceDesiredAgreementUpdate(
		resource.Spec.IsResourceUsageAgreementGranted,
		current.IsResourceUsageAgreementGranted,
	); ok {
		details.IsResourceUsageAgreementGranted = desired
		updateNeeded = true
	}
	if desired, ok := vbsInstanceDesiredResourceCompartmentIDUpdate(
		resource.Spec.ResourceCompartmentId,
		current.ResourceCompartmentID,
	); ok {
		details.ResourceCompartmentId = desired
		updateNeeded = true
	}
	if desired, ok := vbsInstanceDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := vbsInstanceDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func projectVbsInstanceStatus(resource *vbsinstv1beta1.VbsInstance, response any) error {
	if resource == nil || response == nil {
		return nil
	}

	switch current := response.(type) {
	case vbsinstsdk.CreateVbsInstanceResponse, *vbsinstsdk.CreateVbsInstanceResponse,
		vbsinstsdk.UpdateVbsInstanceResponse, *vbsinstsdk.UpdateVbsInstanceResponse,
		vbsinstsdk.DeleteVbsInstanceResponse, *vbsinstsdk.DeleteVbsInstanceResponse:
		return nil
	case nil:
		return nil
	case vbsinstsdk.VbsInstance, *vbsinstsdk.VbsInstance,
		vbsinstsdk.VbsInstanceSummary, *vbsinstsdk.VbsInstanceSummary,
		vbsinstsdk.GetVbsInstanceResponse, *vbsinstsdk.GetVbsInstanceResponse:
	default:
		return fmt.Errorf("unexpected VbsInstance response type %T", current)
	}

	observed, err := vbsInstanceObservedFromResponse(response)
	if err != nil {
		return err
	}

	resource.Status.Id = observed.ID
	resource.Status.Name = observed.Name
	resource.Status.DisplayName = observed.DisplayName
	resource.Status.CompartmentId = observed.CompartmentID
	resource.Status.IsResourceUsageAgreementGranted = observed.IsResourceUsageAgreementGranted != nil && *observed.IsResourceUsageAgreementGranted
	resource.Status.ResourceCompartmentId = observed.ResourceCompartmentID
	resource.Status.VbsAccessUrl = observed.VbsAccessURL
	if observed.TimeCreated != nil {
		resource.Status.TimeCreated = observed.TimeCreated.Format(timeLayout)
	} else {
		resource.Status.TimeCreated = ""
	}
	if observed.TimeUpdated != nil {
		resource.Status.TimeUpdated = observed.TimeUpdated.Format(timeLayout)
	} else {
		resource.Status.TimeUpdated = ""
	}
	resource.Status.LifecycleState = observed.LifecycleState
	resource.Status.LifecycleDetails = observed.LifecycleDetails
	resource.Status.FreeformTags = maps.Clone(observed.FreeformTags)
	resource.Status.DefinedTags = vbsInstanceStatusDefinedTags(observed.DefinedTags)
	resource.Status.SystemTags = vbsInstanceStatusDefinedTags(observed.SystemTags)
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func vbsInstanceObservedFromResponse(currentResponse any) (vbsInstanceObserved, error) {
	switch current := currentResponse.(type) {
	case vbsinstsdk.VbsInstance:
		return observedVbsInstance(current, stringValue(current.LifecyleDetails)), nil
	case *vbsinstsdk.VbsInstance:
		if current == nil {
			return vbsInstanceObserved{}, fmt.Errorf("current VbsInstance response is nil")
		}
		return observedVbsInstance(*current, stringValue(current.LifecyleDetails)), nil
	case vbsinstsdk.VbsInstanceSummary:
		return vbsInstanceObserved{
			ID:                              stringValue(current.Id),
			Name:                            stringValue(current.Name),
			DisplayName:                     stringValue(current.DisplayName),
			CompartmentID:                   stringValue(current.CompartmentId),
			IsResourceUsageAgreementGranted: current.IsResourceUsageAgreementGranted,
			TimeCreated:                     current.TimeCreated,
			TimeUpdated:                     current.TimeUpdated,
			LifecycleState:                  string(current.LifecycleState),
			LifecycleDetails:                stringValue(current.LifecycleDetails),
			FreeformTags:                    maps.Clone(current.FreeformTags),
			DefinedTags:                     cloneSDKDefinedTags(current.DefinedTags),
			SystemTags:                      cloneSDKDefinedTags(current.SystemTags),
		}, nil
	case *vbsinstsdk.VbsInstanceSummary:
		if current == nil {
			return vbsInstanceObserved{}, fmt.Errorf("current VbsInstance response is nil")
		}
		return vbsInstanceObservedFromResponse(*current)
	case vbsinstsdk.GetVbsInstanceResponse:
		return vbsInstanceObservedFromResponse(current.VbsInstance)
	case *vbsinstsdk.GetVbsInstanceResponse:
		if current == nil {
			return vbsInstanceObserved{}, fmt.Errorf("current VbsInstance response is nil")
		}
		return vbsInstanceObservedFromResponse(current.VbsInstance)
	default:
		return vbsInstanceObserved{}, fmt.Errorf("unexpected VbsInstance response type %T", currentResponse)
	}
}

func observedVbsInstance(current vbsinstsdk.VbsInstance, lifecycleDetails string) vbsInstanceObserved {
	return vbsInstanceObserved{
		ID:                              stringValue(current.Id),
		Name:                            stringValue(current.Name),
		DisplayName:                     stringValue(current.DisplayName),
		CompartmentID:                   stringValue(current.CompartmentId),
		IsResourceUsageAgreementGranted: current.IsResourceUsageAgreementGranted,
		ResourceCompartmentID:           stringValue(current.ResourceCompartmentId),
		VbsAccessURL:                    stringValue(current.VbsAccessUrl),
		TimeCreated:                     current.TimeCreated,
		TimeUpdated:                     current.TimeUpdated,
		LifecycleState:                  string(current.LifecycleState),
		LifecycleDetails:                lifecycleDetails,
		FreeformTags:                    maps.Clone(current.FreeformTags),
		DefinedTags:                     cloneSDKDefinedTags(current.DefinedTags),
		SystemTags:                      cloneSDKDefinedTags(current.SystemTags),
	}
}

func vbsInstanceDesiredDisplayNameUpdate(spec string, current string) (*string, bool) {
	if spec == current {
		return nil, false
	}
	if spec == "" && current == "" {
		return nil, false
	}
	return common.String(spec), true
}

func vbsInstanceDesiredAgreementUpdate(spec bool, current *bool) (*bool, bool) {
	currentValue := false
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if !spec && current == nil {
		return nil, false
	}
	return common.Bool(spec), true
}

func vbsInstanceDesiredResourceCompartmentIDUpdate(spec string, current string) (*string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == current {
		return nil, false
	}
	return common.String(spec), true
}

func vbsInstanceDesiredFreeformTagsUpdate(
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

func vbsInstanceDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := vbsInstanceDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if vbsInstanceJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func vbsInstanceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func cloneSDKDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}

	cloned := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		cloned[namespace] = maps.Clone(values)
	}
	return cloned
}

func vbsInstanceStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}

	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		nested := make(shared.MapValue, len(values))
		for key, value := range values {
			nested[key] = fmt.Sprint(value)
		}
		converted[namespace] = nested
	}
	return converted
}

func vbsInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getVbsInstanceWorkRequest(
	ctx context.Context,
	client vbsInstanceOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize VbsInstance OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("VbsInstance OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, vbsinstsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveVbsInstanceGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := vbsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveVbsInstanceGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := vbsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := vbsInstanceWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverVbsInstanceIDFromGeneratedWorkRequest(
	_ *vbsinstv1beta1.VbsInstance,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := vbsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := vbsInstanceWorkRequestActionForPhase(phase)
	if id, ok := resolveVbsInstanceIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveVbsInstanceIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("VbsInstance work request %s does not expose a VbsInstance identifier", stringValue(current.Id))
}

func vbsInstanceGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := vbsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("VbsInstance %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func vbsInstanceWorkRequestFromAny(workRequest any) (vbsinstsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case vbsinstsdk.WorkRequest:
		return current, nil
	case *vbsinstsdk.WorkRequest:
		if current == nil {
			return vbsinstsdk.WorkRequest{}, fmt.Errorf("VbsInstance work request is nil")
		}
		return *current, nil
	default:
		return vbsinstsdk.WorkRequest{}, fmt.Errorf("unexpected VbsInstance work request type %T", workRequest)
	}
}

func vbsInstanceWorkRequestPhaseFromOperationType(operationType vbsinstsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case vbsinstsdk.OperationTypeCreateVbsInstance:
		return shared.OSOKAsyncPhaseCreate, true
	case vbsinstsdk.OperationTypeUpdateVbsInstance:
		return shared.OSOKAsyncPhaseUpdate, true
	case vbsinstsdk.OperationTypeDeleteVbsInstance:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func vbsInstanceWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) vbsinstsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return vbsinstsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return vbsinstsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return vbsinstsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveVbsInstanceIDFromResources(
	resources []vbsinstsdk.WorkRequestResource,
	action vbsinstsdk.ActionTypeEnum,
	preferVbsInstanceOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferVbsInstanceOnly && !isVbsInstanceWorkRequestResource(resource) {
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

func isVbsInstanceWorkRequestResource(resource vbsinstsdk.WorkRequestResource) bool {
	return strings.Contains(normalizeVbsInstanceWorkRequestToken(stringValue(resource.EntityType)), "vbsinstance")
}

func normalizeVbsInstanceWorkRequestToken(value string) string {
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

const timeLayout = "2006-01-02T15:04:05.999999999Z07:00"
