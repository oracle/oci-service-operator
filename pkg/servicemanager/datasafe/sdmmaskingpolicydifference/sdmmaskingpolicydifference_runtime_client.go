/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sdmmaskingpolicydifference

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const (
	sdmMaskingPolicyDifferenceKind                            = "SdmMaskingPolicyDifference"
	sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundMessage = "generated runtime resource not found"
)

type sdmMaskingPolicyDifferenceListCall func(
	context.Context,
	datasafesdk.ListSdmMaskingPolicyDifferencesRequest,
) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error)

type sdmMaskingPolicyDifferenceGetCall func(
	context.Context,
	datasafesdk.GetSdmMaskingPolicyDifferenceRequest,
) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error)

type sdmMaskingPolicyDifferenceAuthShapedNotFound struct {
	operation string
	err       error
}

type sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundError struct {
	reason string
}

func (e sdmMaskingPolicyDifferenceAuthShapedNotFound) Error() string {
	return fmt.Sprintf("datasafe %s delete %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", sdmMaskingPolicyDifferenceKind, e.operation, e.err)
}

func (e sdmMaskingPolicyDifferenceAuthShapedNotFound) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func (e sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundError) Error() string {
	if strings.TrimSpace(e.reason) == "" {
		return sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundMessage
	}
	return fmt.Sprintf("%s: %s", sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundMessage, e.reason)
}

func (e sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundError) Is(target error) bool {
	return target != nil && target.Error() == sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundMessage
}

func init() {
	registerSdmMaskingPolicyDifferenceRuntimeHooksMutator(func(_ *SdmMaskingPolicyDifferenceServiceManager, hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		applySdmMaskingPolicyDifferenceRuntimeHooks(hooks)
	})
}

func applySdmMaskingPolicyDifferenceRuntimeHooks(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = sdmMaskingPolicyDifferenceRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *datasafev1beta1.SdmMaskingPolicyDifference, _ string) (any, error) {
		return buildSdmMaskingPolicyDifferenceCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *datasafev1beta1.SdmMaskingPolicyDifference,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildSdmMaskingPolicyDifferenceUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = sdmMaskingPolicyDifferenceCreateFields()
	hooks.Get.Fields = sdmMaskingPolicyDifferenceGetFields()
	hooks.List.Fields = sdmMaskingPolicyDifferenceListFields()
	hooks.Update.Fields = sdmMaskingPolicyDifferenceUpdateFields()
	hooks.Delete.Fields = sdmMaskingPolicyDifferenceDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listSdmMaskingPolicyDifferencesAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *datasafev1beta1.SdmMaskingPolicyDifference, currentID string) (any, error) {
		return confirmSdmMaskingPolicyDifferenceDeleteRead(ctx, hooks.Get.Call, hooks.List.Call, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleSdmMaskingPolicyDifferenceDeleteError
	hooks.DeleteHooks.ApplyOutcome = applySdmMaskingPolicyDifferenceDeleteOutcome
}

func sdmMaskingPolicyDifferenceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "sdmmaskingpolicydifference",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"maskingPolicyId",
				"differenceType",
				"displayName",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"maskingPolicyId",
				"compartmentId",
				"differenceType",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: sdmMaskingPolicyDifferenceKind, Action: "CreateSdmMaskingPolicyDifference"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: sdmMaskingPolicyDifferenceKind, Action: "UpdateSdmMaskingPolicyDifference"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: sdmMaskingPolicyDifferenceKind, Action: "DeleteSdmMaskingPolicyDifference"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func sdmMaskingPolicyDifferenceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSdmMaskingPolicyDifferenceDetails", RequestName: "CreateSdmMaskingPolicyDifferenceDetails", Contribution: "body"},
	}
}

func sdmMaskingPolicyDifferenceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SdmMaskingPolicyDifferenceId", RequestName: "sdmMaskingPolicyDifferenceId", Contribution: "path", PreferResourceID: true},
	}
}

func sdmMaskingPolicyDifferenceListFields() []generatedruntime.RequestField {
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
			FieldName:    "MaskingPolicyId",
			RequestName:  "maskingPolicyId",
			Contribution: "query",
			LookupPaths:  []string{"status.maskingPolicyId", "spec.maskingPolicyId", "maskingPolicyId"},
		},
		{
			FieldName:    "SensitiveDataModelId",
			RequestName:  "sensitiveDataModelId",
			Contribution: "query",
			LookupPaths:  []string{"status.sensitiveDataModelId", "sensitiveDataModelId"},
		},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func sdmMaskingPolicyDifferenceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SdmMaskingPolicyDifferenceId", RequestName: "sdmMaskingPolicyDifferenceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSdmMaskingPolicyDifferenceDetails", RequestName: "UpdateSdmMaskingPolicyDifferenceDetails", Contribution: "body"},
	}
}

func sdmMaskingPolicyDifferenceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SdmMaskingPolicyDifferenceId", RequestName: "sdmMaskingPolicyDifferenceId", Contribution: "path", PreferResourceID: true},
	}
}

func buildSdmMaskingPolicyDifferenceCreateBody(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
) (datasafesdk.CreateSdmMaskingPolicyDifferenceDetails, error) {
	if resource == nil {
		return datasafesdk.CreateSdmMaskingPolicyDifferenceDetails{}, fmt.Errorf("%s resource is nil", sdmMaskingPolicyDifferenceKind)
	}

	details := datasafesdk.CreateSdmMaskingPolicyDifferenceDetails{
		MaskingPolicyId: common.String(strings.TrimSpace(resource.Spec.MaskingPolicyId)),
		CompartmentId:   common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
	}
	if resource.Spec.DifferenceType != "" {
		details.DifferenceType = datasafesdk.SdmMaskingPolicyDifferenceDifferenceTypeEnum(strings.TrimSpace(resource.Spec.DifferenceType))
	}
	if resource.Spec.DisplayName != "" {
		details.DisplayName = common.String(resource.Spec.DisplayName)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneSdmMaskingPolicyDifferenceStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = sdmMaskingPolicyDifferenceDefinedTags(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildSdmMaskingPolicyDifferenceUpdateBody(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	currentResponse any,
) (datasafesdk.UpdateSdmMaskingPolicyDifferenceDetails, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateSdmMaskingPolicyDifferenceDetails{}, false, fmt.Errorf("%s resource is nil", sdmMaskingPolicyDifferenceKind)
	}
	current, ok := sdmMaskingPolicyDifferenceFromResponse(currentResponse)
	if !ok {
		return datasafesdk.UpdateSdmMaskingPolicyDifferenceDetails{}, false, fmt.Errorf("current %s response does not expose a %s body", sdmMaskingPolicyDifferenceKind, sdmMaskingPolicyDifferenceKind)
	}

	details := datasafesdk.UpdateSdmMaskingPolicyDifferenceDetails{}
	updateNeeded := false
	if resource.Spec.DisplayName != "" && stringPointerValue(current.DisplayName) != resource.Spec.DisplayName {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil {
		tags := cloneSdmMaskingPolicyDifferenceStringMap(resource.Spec.FreeformTags)
		if !reflect.DeepEqual(tags, current.FreeformTags) {
			details.FreeformTags = tags
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		tags := sdmMaskingPolicyDifferenceDefinedTags(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(tags, current.DefinedTags) {
			details.DefinedTags = tags
			updateNeeded = true
		}
	}
	return details, updateNeeded, nil
}

func sdmMaskingPolicyDifferenceFromResponse(response any) (datasafesdk.SdmMaskingPolicyDifference, bool) {
	if resource, ok := sdmMaskingPolicyDifferenceFromDirectResponse(response); ok {
		return resource, true
	}
	if resource, ok := sdmMaskingPolicyDifferenceFromWriteResponse(response); ok {
		return resource, true
	}
	return sdmMaskingPolicyDifferenceFromSummaryResponse(response)
}

func sdmMaskingPolicyDifferenceFromDirectResponse(response any) (datasafesdk.SdmMaskingPolicyDifference, bool) {
	switch typed := response.(type) {
	case datasafesdk.SdmMaskingPolicyDifference:
		return typed, true
	case *datasafesdk.SdmMaskingPolicyDifference:
		if typed != nil {
			return *typed, true
		}
	}
	return datasafesdk.SdmMaskingPolicyDifference{}, false
}

func sdmMaskingPolicyDifferenceFromWriteResponse(response any) (datasafesdk.SdmMaskingPolicyDifference, bool) {
	switch typed := response.(type) {
	case datasafesdk.GetSdmMaskingPolicyDifferenceResponse:
		return typed.SdmMaskingPolicyDifference, true
	case *datasafesdk.GetSdmMaskingPolicyDifferenceResponse:
		if typed != nil {
			return typed.SdmMaskingPolicyDifference, true
		}
	case datasafesdk.CreateSdmMaskingPolicyDifferenceResponse:
		return typed.SdmMaskingPolicyDifference, true
	case *datasafesdk.CreateSdmMaskingPolicyDifferenceResponse:
		if typed != nil {
			return typed.SdmMaskingPolicyDifference, true
		}
	}
	return datasafesdk.SdmMaskingPolicyDifference{}, false
}

func sdmMaskingPolicyDifferenceFromSummaryResponse(response any) (datasafesdk.SdmMaskingPolicyDifference, bool) {
	switch typed := response.(type) {
	case datasafesdk.SdmMaskingPolicyDifferenceSummary:
		return sdmMaskingPolicyDifferenceFromSummary(typed), true
	case *datasafesdk.SdmMaskingPolicyDifferenceSummary:
		if typed != nil {
			return sdmMaskingPolicyDifferenceFromSummary(*typed), true
		}
	}
	return datasafesdk.SdmMaskingPolicyDifference{}, false
}

func sdmMaskingPolicyDifferenceFromSummary(summary datasafesdk.SdmMaskingPolicyDifferenceSummary) datasafesdk.SdmMaskingPolicyDifference {
	return datasafesdk.SdmMaskingPolicyDifference{
		Id:                   summary.Id,
		CompartmentId:        summary.CompartmentId,
		DifferenceType:       summary.DifferenceType,
		DisplayName:          summary.DisplayName,
		TimeCreated:          summary.TimeCreated,
		TimeCreationStarted:  summary.TimeCreationStarted,
		LifecycleState:       summary.LifecycleState,
		SensitiveDataModelId: summary.SensitiveDataModelId,
		MaskingPolicyId:      summary.MaskingPolicyId,
		FreeformTags:         summary.FreeformTags,
		DefinedTags:          summary.DefinedTags,
	}
}

func listSdmMaskingPolicyDifferencesAllPages(
	next sdmMaskingPolicyDifferenceListCall,
) sdmMaskingPolicyDifferenceListCall {
	return func(ctx context.Context, request datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
		var combined datasafesdk.ListSdmMaskingPolicyDifferencesResponse
		for {
			response, err := next(ctx, request)
			if err != nil {
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.RawResponse = response.RawResponse
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			request.Page = response.OpcNextPage
		}
	}
}

func confirmSdmMaskingPolicyDifferenceDeleteRead(
	ctx context.Context,
	get sdmMaskingPolicyDifferenceGetCall,
	list sdmMaskingPolicyDifferenceListCall,
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	currentID string,
) (any, error) {
	currentID = strings.TrimSpace(currentID)
	if currentID == "" {
		return confirmSdmMaskingPolicyDifferenceDeleteReadByList(ctx, list, resource)
	}
	return confirmSdmMaskingPolicyDifferenceDeleteReadByID(ctx, get, currentID)
}

func confirmSdmMaskingPolicyDifferenceDeleteReadByID(
	ctx context.Context,
	get sdmMaskingPolicyDifferenceGetCall,
	currentID string,
) (any, error) {
	if get == nil {
		return nil, fmt.Errorf("confirm %s delete: get hook is not configured", sdmMaskingPolicyDifferenceKind)
	}

	response, err := get(ctx, datasafesdk.GetSdmMaskingPolicyDifferenceRequest{
		SdmMaskingPolicyDifferenceId: common.String(currentID),
	})
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return sdmMaskingPolicyDifferenceAuthShapedNotFound{operation: "confirmation", err: err}, nil
	}
	return nil, err
}

func confirmSdmMaskingPolicyDifferenceDeleteReadByList(
	ctx context.Context,
	list sdmMaskingPolicyDifferenceListCall,
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
) (any, error) {
	if list == nil {
		return nil, fmt.Errorf("confirm %s delete: list hook is not configured", sdmMaskingPolicyDifferenceKind)
	}
	request, err := sdmMaskingPolicyDifferenceDeleteListRequest(resource)
	if err != nil {
		return nil, err
	}
	response, err := list(ctx, request)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			ambiguous := sdmMaskingPolicyDifferenceAuthShapedNotFound{operation: "list confirmation", err: err}
			recordSdmMaskingPolicyDifferenceDeleteRequestID(resource, ambiguous)
			return nil, ambiguous
		}
		return nil, err
	}
	match, err := selectSdmMaskingPolicyDifferenceDeleteListMatch(resource, response.Items)
	if err != nil {
		return nil, err
	}
	recordSdmMaskingPolicyDifferenceRecoveredID(resource, stringPointerValue(match.Id))
	return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
		SdmMaskingPolicyDifference: sdmMaskingPolicyDifferenceFromSummary(match),
		OpcRequestId:               response.OpcRequestId,
	}, nil
}

func sdmMaskingPolicyDifferenceDeleteListRequest(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
) (datasafesdk.ListSdmMaskingPolicyDifferencesRequest, error) {
	compartmentID := currentSdmMaskingPolicyDifferenceCompartmentID(resource)
	if compartmentID == "" {
		return datasafesdk.ListSdmMaskingPolicyDifferencesRequest{}, fmt.Errorf("confirm %s delete: compartmentId is empty", sdmMaskingPolicyDifferenceKind)
	}
	request := datasafesdk.ListSdmMaskingPolicyDifferencesRequest{
		CompartmentId: common.String(compartmentID),
	}
	if displayName := currentSdmMaskingPolicyDifferenceDisplayName(resource); displayName != "" {
		request.DisplayName = common.String(displayName)
	}
	if maskingPolicyID := currentSdmMaskingPolicyDifferenceMaskingPolicyID(resource); maskingPolicyID != "" {
		request.MaskingPolicyId = common.String(maskingPolicyID)
	}
	if sensitiveDataModelID := currentSdmMaskingPolicyDifferenceSensitiveDataModelID(resource); sensitiveDataModelID != "" {
		request.SensitiveDataModelId = common.String(sensitiveDataModelID)
	}
	return request, nil
}

func selectSdmMaskingPolicyDifferenceDeleteListMatch(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	items []datasafesdk.SdmMaskingPolicyDifferenceSummary,
) (datasafesdk.SdmMaskingPolicyDifferenceSummary, error) {
	var matches []datasafesdk.SdmMaskingPolicyDifferenceSummary
	for _, item := range items {
		if sdmMaskingPolicyDifferenceSummaryMatchesResource(resource, item) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return datasafesdk.SdmMaskingPolicyDifferenceSummary{}, sdmMaskingPolicyDifferenceGeneratedRuntimeNotFoundError{
			reason: "delete confirmation list found no matching resource",
		}
	case 1:
		if stringPointerValue(matches[0].Id) == "" {
			return datasafesdk.SdmMaskingPolicyDifferenceSummary{}, fmt.Errorf("%s delete confirmation list match does not include an OCID", sdmMaskingPolicyDifferenceKind)
		}
		return matches[0], nil
	default:
		return datasafesdk.SdmMaskingPolicyDifferenceSummary{}, fmt.Errorf("%s delete confirmation list found multiple matching resources", sdmMaskingPolicyDifferenceKind)
	}
}

func sdmMaskingPolicyDifferenceSummaryMatchesResource(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	item datasafesdk.SdmMaskingPolicyDifferenceSummary,
) bool {
	if resource == nil {
		return false
	}
	for _, constraint := range sdmMaskingPolicyDifferenceSummaryMatchConstraints(resource, item) {
		if !constraint.matches() {
			return false
		}
	}
	return true
}

type sdmMaskingPolicyDifferenceSummaryMatchConstraint struct {
	expected string
	actual   string
}

func (constraint sdmMaskingPolicyDifferenceSummaryMatchConstraint) matches() bool {
	return constraint.expected == "" || constraint.expected == constraint.actual
}

func sdmMaskingPolicyDifferenceSummaryMatchConstraints(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	item datasafesdk.SdmMaskingPolicyDifferenceSummary,
) []sdmMaskingPolicyDifferenceSummaryMatchConstraint {
	return []sdmMaskingPolicyDifferenceSummaryMatchConstraint{
		{
			expected: currentSdmMaskingPolicyDifferenceCompartmentID(resource),
			actual:   stringPointerValue(item.CompartmentId),
		},
		{
			expected: currentSdmMaskingPolicyDifferenceMaskingPolicyID(resource),
			actual:   stringPointerValue(item.MaskingPolicyId),
		},
		{
			expected: currentSdmMaskingPolicyDifferenceDifferenceType(resource),
			actual:   string(item.DifferenceType),
		},
		{
			expected: currentSdmMaskingPolicyDifferenceDisplayName(resource),
			actual:   stringPointerValue(item.DisplayName),
		},
		{
			expected: currentSdmMaskingPolicyDifferenceSensitiveDataModelID(resource),
			actual:   stringPointerValue(item.SensitiveDataModelId),
		},
	}
}

func handleSdmMaskingPolicyDifferenceDeleteError(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	err error,
) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return sdmMaskingPolicyDifferenceAuthShapedNotFound{operation: "request", err: err}
}

func applySdmMaskingPolicyDifferenceDeleteOutcome(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case sdmMaskingPolicyDifferenceAuthShapedNotFound:
		recordSdmMaskingPolicyDifferenceDeleteRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *sdmMaskingPolicyDifferenceAuthShapedNotFound:
		if typed != nil {
			recordSdmMaskingPolicyDifferenceDeleteRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func recordSdmMaskingPolicyDifferenceDeleteRequestID(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	err sdmMaskingPolicyDifferenceAuthShapedNotFound,
) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func recordSdmMaskingPolicyDifferenceRecoveredID(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	resourceID string,
) {
	if resource == nil || strings.TrimSpace(resourceID) == "" {
		return
	}
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
}

func cloneSdmMaskingPolicyDifferenceStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func sdmMaskingPolicyDifferenceDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func currentSdmMaskingPolicyDifferenceCompartmentID(resource *datasafev1beta1.SdmMaskingPolicyDifference) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptySdmMaskingPolicyDifferenceString(resource.Status.CompartmentId, resource.Spec.CompartmentId)
}

func currentSdmMaskingPolicyDifferenceMaskingPolicyID(resource *datasafev1beta1.SdmMaskingPolicyDifference) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptySdmMaskingPolicyDifferenceString(resource.Status.MaskingPolicyId, resource.Spec.MaskingPolicyId)
}

func currentSdmMaskingPolicyDifferenceDifferenceType(resource *datasafev1beta1.SdmMaskingPolicyDifference) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptySdmMaskingPolicyDifferenceString(resource.Status.DifferenceType, resource.Spec.DifferenceType)
}

func currentSdmMaskingPolicyDifferenceDisplayName(resource *datasafev1beta1.SdmMaskingPolicyDifference) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptySdmMaskingPolicyDifferenceString(resource.Status.DisplayName, resource.Spec.DisplayName)
}

func currentSdmMaskingPolicyDifferenceSensitiveDataModelID(resource *datasafev1beta1.SdmMaskingPolicyDifference) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Status.SensitiveDataModelId)
}

func firstNonEmptySdmMaskingPolicyDifferenceString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
