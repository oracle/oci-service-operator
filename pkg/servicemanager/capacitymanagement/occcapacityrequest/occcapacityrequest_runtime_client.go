/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package occcapacityrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	capacitymanagementsdk "github.com/oracle/oci-go-sdk/v65/capacitymanagement"
	capacitymanagementv1beta1 "github.com/oracle/oci-service-operator/api/capacitymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type occCapacityRequestOCIClient interface {
	CreateOccCapacityRequest(context.Context, capacitymanagementsdk.CreateOccCapacityRequestRequest) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error)
	GetOccCapacityRequest(context.Context, capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error)
	ListOccCapacityRequests(context.Context, capacitymanagementsdk.ListOccCapacityRequestsRequest) (capacitymanagementsdk.ListOccCapacityRequestsResponse, error)
	UpdateOccCapacityRequest(context.Context, capacitymanagementsdk.UpdateOccCapacityRequestRequest) (capacitymanagementsdk.UpdateOccCapacityRequestResponse, error)
	DeleteOccCapacityRequest(context.Context, capacitymanagementsdk.DeleteOccCapacityRequestRequest) (capacitymanagementsdk.DeleteOccCapacityRequestResponse, error)
}

func init() {
	registerOccCapacityRequestRuntimeHooksMutator(func(_ *OccCapacityRequestServiceManager, hooks *OccCapacityRequestRuntimeHooks) {
		applyOccCapacityRequestRuntimeHooks(hooks)
	})
}

func applyOccCapacityRequestRuntimeHooks(hooks *OccCapacityRequestRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedOccCapacityRequestRuntimeSemantics()
	hooks.ParityHooks.NormalizeDesiredState = normalizeOccCapacityRequestDesiredState
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *capacitymanagementv1beta1.OccCapacityRequest,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildOccCapacityRequestUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardOccCapacityRequestExistingBeforeCreate
	hooks.Read.List = disabledOccCapacityRequestListReadOperation()
}

func newOccCapacityRequestServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client occCapacityRequestOCIClient,
) OccCapacityRequestServiceClient {
	return defaultOccCapacityRequestServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*capacitymanagementv1beta1.OccCapacityRequest](
			newOccCapacityRequestRuntimeConfig(log, client),
		),
	}
}

func newOccCapacityRequestRuntimeConfig(
	log loggerutil.OSOKLogger,
	client occCapacityRequestOCIClient,
) generatedruntime.Config[*capacitymanagementv1beta1.OccCapacityRequest] {
	hooks := newOccCapacityRequestRuntimeHooksWithOCIClient(client)
	applyOccCapacityRequestRuntimeHooks(&hooks)
	return buildOccCapacityRequestGeneratedRuntimeConfig(&OccCapacityRequestServiceManager{Log: log}, hooks)
}

func newOccCapacityRequestRuntimeHooksWithOCIClient(
	client occCapacityRequestOCIClient,
) OccCapacityRequestRuntimeHooks {
	return OccCapacityRequestRuntimeHooks{
		Semantics: reviewedOccCapacityRequestRuntimeSemantics(),
		Create: runtimeOperationHooks[capacitymanagementsdk.CreateOccCapacityRequestRequest, capacitymanagementsdk.CreateOccCapacityRequestResponse]{
			Fields: occCapacityRequestCreateFields(),
			Call: func(ctx context.Context, request capacitymanagementsdk.CreateOccCapacityRequestRequest) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error) {
				return client.CreateOccCapacityRequest(ctx, request)
			},
		},
		Get: runtimeOperationHooks[capacitymanagementsdk.GetOccCapacityRequestRequest, capacitymanagementsdk.GetOccCapacityRequestResponse]{
			Fields: occCapacityRequestGetFields(),
			Call: func(ctx context.Context, request capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
				return client.GetOccCapacityRequest(ctx, request)
			},
		},
		List: runtimeOperationHooks[capacitymanagementsdk.ListOccCapacityRequestsRequest, capacitymanagementsdk.ListOccCapacityRequestsResponse]{
			Fields: occCapacityRequestListFields(),
			Call: func(ctx context.Context, request capacitymanagementsdk.ListOccCapacityRequestsRequest) (capacitymanagementsdk.ListOccCapacityRequestsResponse, error) {
				return client.ListOccCapacityRequests(ctx, request)
			},
		},
		Update: runtimeOperationHooks[capacitymanagementsdk.UpdateOccCapacityRequestRequest, capacitymanagementsdk.UpdateOccCapacityRequestResponse]{
			Fields: occCapacityRequestUpdateFields(),
			Call: func(ctx context.Context, request capacitymanagementsdk.UpdateOccCapacityRequestRequest) (capacitymanagementsdk.UpdateOccCapacityRequestResponse, error) {
				return client.UpdateOccCapacityRequest(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[capacitymanagementsdk.DeleteOccCapacityRequestRequest, capacitymanagementsdk.DeleteOccCapacityRequestResponse]{
			Fields: occCapacityRequestDeleteFields(),
			Call: func(ctx context.Context, request capacitymanagementsdk.DeleteOccCapacityRequestRequest) (capacitymanagementsdk.DeleteOccCapacityRequestResponse, error) {
				return client.DeleteOccCapacityRequest(ctx, request)
			},
		},
	}
}

func reviewedOccCapacityRequestRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newOccCapacityRequestRuntimeSemantics()
	semantics.List = nil
	return semantics
}

func occCapacityRequestCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateOccCapacityRequestDetails", RequestName: "CreateOccCapacityRequestDetails", Contribution: "body"},
	}
}

func occCapacityRequestGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OccCapacityRequestId", RequestName: "occCapacityRequestId", Contribution: "path", PreferResourceID: true},
	}
}

func occCapacityRequestListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "OccAvailabilityCatalogId", RequestName: "occAvailabilityCatalogId", Contribution: "query"},
		{FieldName: "Namespace", RequestName: "namespace", Contribution: "query"},
		{FieldName: "RequestType", RequestName: "requestType", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query"},
	}
}

func occCapacityRequestUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OccCapacityRequestId", RequestName: "occCapacityRequestId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateOccCapacityRequestDetails", RequestName: "UpdateOccCapacityRequestDetails", Contribution: "body"},
	}
}

func occCapacityRequestDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OccCapacityRequestId", RequestName: "occCapacityRequestId", Contribution: "path", PreferResourceID: true},
	}
}

func guardOccCapacityRequestExistingBeforeCreate(
	_ context.Context,
	resource *capacitymanagementv1beta1.OccCapacityRequest,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("OccCapacityRequest resource is nil")
	}
	return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
}

func disabledOccCapacityRequestListReadOperation() *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &capacitymanagementsdk.ListOccCapacityRequestsRequest{} },
		Call: func(context.Context, any) (any, error) {
			return capacitymanagementsdk.ListOccCapacityRequestsResponse{
				OccCapacityRequestCollection: capacitymanagementsdk.OccCapacityRequestCollection{
					Items: []capacitymanagementsdk.OccCapacityRequestSummary{},
				},
			}, nil
		},
	}
}

func normalizeOccCapacityRequestDesiredState(
	resource *capacitymanagementv1beta1.OccCapacityRequest,
	currentResponse any,
) {
	if resource == nil || currentResponse == nil {
		return
	}

	// OCI owns lifecycleDetails after create; do not treat it as steady-state drift.
	resource.Spec.LifecycleDetails = ""

	if !occCapacityRequestRequestStateUpdatable(resource.Spec.RequestState) {
		resource.Spec.RequestState = ""
		return
	}

	current, err := occCapacityRequestFromResponse(currentResponse)
	if err != nil {
		return
	}
	if !occCapacityRequestRequestStateMutable(current.RequestState) {
		resource.Spec.RequestState = ""
	}
}

func occCapacityRequestRequestStateUpdatable(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case string(capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateSubmitted),
		string(capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateCancelled):
		return true
	default:
		return false
	}
}

func occCapacityRequestRequestStateMutable(state capacitymanagementsdk.OccCapacityRequestRequestStateEnum) bool {
	switch state {
	case capacitymanagementsdk.OccCapacityRequestRequestStateCreated,
		capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted,
		capacitymanagementsdk.OccCapacityRequestRequestStateCancelled:
		return true
	default:
		return false
	}
}

func buildOccCapacityRequestUpdateBody(
	resource *capacitymanagementv1beta1.OccCapacityRequest,
	currentResponse any,
) (capacitymanagementsdk.UpdateOccCapacityRequestDetails, bool, error) {
	if resource == nil {
		return capacitymanagementsdk.UpdateOccCapacityRequestDetails{}, false, fmt.Errorf("OccCapacityRequest resource is nil")
	}

	current, err := occCapacityRequestFromResponse(currentResponse)
	if err != nil {
		return capacitymanagementsdk.UpdateOccCapacityRequestDetails{}, false, err
	}

	details := capacitymanagementsdk.UpdateOccCapacityRequestDetails{}
	updateNeeded := false

	if desired, ok := occCapacityRequestDesiredDisplayNameUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok, err := occCapacityRequestDesiredRequestStateUpdate(resource.Spec.RequestState, current.RequestState); err != nil {
		return capacitymanagementsdk.UpdateOccCapacityRequestDetails{}, false, err
	} else if ok {
		details.RequestState = desired
		updateNeeded = true
	}
	if desired, ok := occCapacityRequestDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := occCapacityRequestDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func occCapacityRequestFromResponse(currentResponse any) (capacitymanagementsdk.OccCapacityRequest, error) {
	switch current := currentResponse.(type) {
	case capacitymanagementsdk.OccCapacityRequest:
		return current, nil
	case *capacitymanagementsdk.OccCapacityRequest:
		if current == nil {
			return capacitymanagementsdk.OccCapacityRequest{}, fmt.Errorf("current OccCapacityRequest response is nil")
		}
		return *current, nil
	case capacitymanagementsdk.OccCapacityRequestSummary:
		return capacitymanagementsdk.OccCapacityRequest{
			Id:                           current.Id,
			CompartmentId:                current.CompartmentId,
			OccAvailabilityCatalogId:     current.OccAvailabilityCatalogId,
			DisplayName:                  current.DisplayName,
			Namespace:                    current.Namespace,
			OccCustomerGroupId:           current.OccCustomerGroupId,
			Region:                       current.Region,
			AvailabilityDomain:           current.AvailabilityDomain,
			DateExpectedCapacityHandover: current.DateExpectedCapacityHandover,
			RequestState:                 current.RequestState,
			TimeCreated:                  current.TimeCreated,
			TimeUpdated:                  current.TimeUpdated,
			LifecycleState:               current.LifecycleState,
			Description:                  current.Description,
			RequestType:                  current.RequestType,
			LifecycleDetails:             current.LifecycleDetails,
			FreeformTags:                 current.FreeformTags,
			DefinedTags:                  current.DefinedTags,
			SystemTags:                   current.SystemTags,
		}, nil
	case *capacitymanagementsdk.OccCapacityRequestSummary:
		if current == nil {
			return capacitymanagementsdk.OccCapacityRequest{}, fmt.Errorf("current OccCapacityRequest response is nil")
		}
		return occCapacityRequestFromResponse(*current)
	case capacitymanagementsdk.CreateOccCapacityRequestResponse:
		return current.OccCapacityRequest, nil
	case *capacitymanagementsdk.CreateOccCapacityRequestResponse:
		if current == nil {
			return capacitymanagementsdk.OccCapacityRequest{}, fmt.Errorf("current OccCapacityRequest response is nil")
		}
		return current.OccCapacityRequest, nil
	case capacitymanagementsdk.GetOccCapacityRequestResponse:
		return current.OccCapacityRequest, nil
	case *capacitymanagementsdk.GetOccCapacityRequestResponse:
		if current == nil {
			return capacitymanagementsdk.OccCapacityRequest{}, fmt.Errorf("current OccCapacityRequest response is nil")
		}
		return current.OccCapacityRequest, nil
	case capacitymanagementsdk.UpdateOccCapacityRequestResponse:
		return current.OccCapacityRequest, nil
	case *capacitymanagementsdk.UpdateOccCapacityRequestResponse:
		if current == nil {
			return capacitymanagementsdk.OccCapacityRequest{}, fmt.Errorf("current OccCapacityRequest response is nil")
		}
		return current.OccCapacityRequest, nil
	default:
		return capacitymanagementsdk.OccCapacityRequest{}, fmt.Errorf("unexpected current OccCapacityRequest response type %T", currentResponse)
	}
}

func occCapacityRequestDesiredDisplayNameUpdate(spec string, current *string) (*string, bool) {
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
	return &spec, true
}

func occCapacityRequestDesiredRequestStateUpdate(
	spec string,
	current capacitymanagementsdk.OccCapacityRequestRequestStateEnum,
) (capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateEnum, bool, error) {
	normalized := strings.ToUpper(strings.TrimSpace(spec))
	if normalized == "" || normalized == string(current) {
		return "", false, nil
	}

	switch normalized {
	case string(capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateSubmitted):
		return capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateSubmitted, true, nil
	case string(capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateCancelled):
		return capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateCancelled, true, nil
	default:
		return "", false, fmt.Errorf("OccCapacityRequest requestState %q is not supported for in-place update", spec)
	}
}

func occCapacityRequestDesiredFreeformTagsUpdate(
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

func occCapacityRequestDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := occCapacityRequestDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if occCapacityRequestJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func occCapacityRequestDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func occCapacityRequestJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
