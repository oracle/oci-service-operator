/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package customprotectionrule

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerCustomProtectionRuleRuntimeHooksMutator(func(_ *CustomProtectionRuleServiceManager, hooks *CustomProtectionRuleRuntimeHooks) {
		applyCustomProtectionRuleRuntimeHooks(hooks)
	})
}

func applyCustomProtectionRuleRuntimeHooks(hooks *CustomProtectionRuleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = customProtectionRuleRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *waasv1beta1.CustomProtectionRule,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildCustomProtectionRuleUpdateBody(resource, currentResponse)
	}
	hooks.List.Fields = customProtectionRuleListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listCustomProtectionRulesAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleCustomProtectionRuleDeleteError
	wrapCustomProtectionRuleDeleteConfirmation(hooks)
}

func customProtectionRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "waas",
		FormalSlug:    "customprotectionrule",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(waassdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(waassdk.LifecycleStatesUpdating)},
			ActiveStates:       []string{string(waassdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(waassdk.LifecycleStatesDeleting)},
			TerminalStates: []string{string(waassdk.LifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "description", "displayName", "freeformTags", "template"},
			Mutable:         []string{"definedTags", "description", "displayName", "freeformTags", "template"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
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
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func customProtectionRuleListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

type customProtectionRuleListCall func(context.Context, waassdk.ListCustomProtectionRulesRequest) (waassdk.ListCustomProtectionRulesResponse, error)

func listCustomProtectionRulesAllPages(call customProtectionRuleListCall) customProtectionRuleListCall {
	return func(ctx context.Context, request waassdk.ListCustomProtectionRulesRequest) (waassdk.ListCustomProtectionRulesResponse, error) {
		accumulator := newCustomProtectionRuleListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return waassdk.ListCustomProtectionRulesResponse{}, err
			}
			accumulator.append(response)

			nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
			if nextPage == "" {
				accumulator.response.OpcNextPage = nil
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return waassdk.ListCustomProtectionRulesResponse{}, err
			}
		}
	}
}

type customProtectionRuleListAccumulator struct {
	response  waassdk.ListCustomProtectionRulesResponse
	seenPages map[string]struct{}
}

func newCustomProtectionRuleListAccumulator() customProtectionRuleListAccumulator {
	return customProtectionRuleListAccumulator{
		seenPages: map[string]struct{}{},
	}
}

func (a *customProtectionRuleListAccumulator) append(response waassdk.ListCustomProtectionRulesResponse) {
	a.response.RawResponse = response.RawResponse
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *customProtectionRuleListAccumulator) advance(request *waassdk.ListCustomProtectionRulesRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("CustomProtectionRule list pagination repeated page token %q", nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	a.response.OpcNextPage = common.String(nextPage)
	return nil
}

func buildCustomProtectionRuleUpdateBody(
	resource *waasv1beta1.CustomProtectionRule,
	currentResponse any,
) (waassdk.UpdateCustomProtectionRuleDetails, bool, error) {
	if resource == nil {
		return waassdk.UpdateCustomProtectionRuleDetails{}, false, fmt.Errorf("CustomProtectionRule resource is nil")
	}

	current, err := customProtectionRuleRuntimeBody(currentResponse)
	if err != nil {
		return waassdk.UpdateCustomProtectionRuleDetails{}, false, err
	}

	details := waassdk.UpdateCustomProtectionRuleDetails{}
	updateNeeded := false
	if resource.Spec.DisplayName != "" && stringValue(current.DisplayName) != resource.Spec.DisplayName {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.Template != "" && stringValue(current.Template) != resource.Spec.Template {
		details.Template = common.String(resource.Spec.Template)
		updateNeeded = true
	}
	if desired, ok := customProtectionRuleDesiredDescriptionUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := customProtectionRuleDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := customProtectionRuleDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func customProtectionRuleRuntimeBody(currentResponse any) (waassdk.CustomProtectionRule, error) {
	if current, ok, err := customProtectionRuleDirectRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	if current, ok, err := customProtectionRuleResponseRuntimeBody(currentResponse); ok || err != nil {
		return current, err
	}
	return waassdk.CustomProtectionRule{}, fmt.Errorf("unexpected current CustomProtectionRule response type %T", currentResponse)
}

func customProtectionRuleDirectRuntimeBody(currentResponse any) (waassdk.CustomProtectionRule, bool, error) {
	switch current := currentResponse.(type) {
	case waassdk.CustomProtectionRule:
		return current, true, nil
	case *waassdk.CustomProtectionRule:
		body, err := dereferenceCustomProtectionRuleRuntimeBody(current)
		return body, true, err
	case waassdk.CustomProtectionRuleSummary:
		return customProtectionRuleFromSummary(current), true, nil
	case *waassdk.CustomProtectionRuleSummary:
		summary, err := dereferenceCustomProtectionRuleRuntimeBody(current)
		if err != nil {
			return waassdk.CustomProtectionRule{}, true, err
		}
		return customProtectionRuleFromSummary(summary), true, nil
	default:
		return waassdk.CustomProtectionRule{}, false, nil
	}
}

func customProtectionRuleResponseRuntimeBody(currentResponse any) (waassdk.CustomProtectionRule, bool, error) {
	switch current := currentResponse.(type) {
	case waassdk.CreateCustomProtectionRuleResponse:
		return current.CustomProtectionRule, true, nil
	case *waassdk.CreateCustomProtectionRuleResponse:
		return customProtectionRuleFromCreateResponse(current)
	case waassdk.GetCustomProtectionRuleResponse:
		return current.CustomProtectionRule, true, nil
	case *waassdk.GetCustomProtectionRuleResponse:
		return customProtectionRuleFromGetResponse(current)
	case waassdk.UpdateCustomProtectionRuleResponse:
		return current.CustomProtectionRule, true, nil
	case *waassdk.UpdateCustomProtectionRuleResponse:
		return customProtectionRuleFromUpdateResponse(current)
	default:
		return waassdk.CustomProtectionRule{}, false, nil
	}
}

func customProtectionRuleFromCreateResponse(current *waassdk.CreateCustomProtectionRuleResponse) (waassdk.CustomProtectionRule, bool, error) {
	response, err := dereferenceCustomProtectionRuleRuntimeBody(current)
	if err != nil {
		return waassdk.CustomProtectionRule{}, true, err
	}
	return response.CustomProtectionRule, true, nil
}

func customProtectionRuleFromGetResponse(current *waassdk.GetCustomProtectionRuleResponse) (waassdk.CustomProtectionRule, bool, error) {
	response, err := dereferenceCustomProtectionRuleRuntimeBody(current)
	if err != nil {
		return waassdk.CustomProtectionRule{}, true, err
	}
	return response.CustomProtectionRule, true, nil
}

func customProtectionRuleFromUpdateResponse(current *waassdk.UpdateCustomProtectionRuleResponse) (waassdk.CustomProtectionRule, bool, error) {
	response, err := dereferenceCustomProtectionRuleRuntimeBody(current)
	if err != nil {
		return waassdk.CustomProtectionRule{}, true, err
	}
	return response.CustomProtectionRule, true, nil
}

func dereferenceCustomProtectionRuleRuntimeBody[T any](current *T) (T, error) {
	if current == nil {
		var zero T
		return zero, fmt.Errorf("current CustomProtectionRule response is nil")
	}
	return *current, nil
}

func customProtectionRuleFromSummary(summary waassdk.CustomProtectionRuleSummary) waassdk.CustomProtectionRule {
	return waassdk.CustomProtectionRule{
		Id:                 summary.Id,
		CompartmentId:      summary.CompartmentId,
		DisplayName:        summary.DisplayName,
		ModSecurityRuleIds: summary.ModSecurityRuleIds,
		LifecycleState:     summary.LifecycleState,
		TimeCreated:        summary.TimeCreated,
		FreeformTags:       summary.FreeformTags,
		DefinedTags:        summary.DefinedTags,
	}
}

func customProtectionRuleDesiredDescriptionUpdate(spec string, current *string) (*string, bool) {
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

func customProtectionRuleDesiredFreeformTagsUpdate(
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

func customProtectionRuleDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := customProtectionRuleDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if customProtectionRuleJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func customProtectionRuleDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	definedTags := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		definedTags[namespace] = converted
	}
	return definedTags
}

func customProtectionRuleJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func handleCustomProtectionRuleDeleteError(resource *waasv1beta1.CustomProtectionRule, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("CustomProtectionRule delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapCustomProtectionRuleDeleteConfirmation(hooks *CustomProtectionRuleRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getCustomProtectionRule := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate CustomProtectionRuleServiceClient) CustomProtectionRuleServiceClient {
		return customProtectionRuleDeleteConfirmationClient{
			delegate:                delegate,
			getCustomProtectionRule: getCustomProtectionRule,
		}
	})
}

type customProtectionRuleDeleteConfirmationClient struct {
	delegate                CustomProtectionRuleServiceClient
	getCustomProtectionRule func(context.Context, waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error)
}

func (c customProtectionRuleDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *waasv1beta1.CustomProtectionRule,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c customProtectionRuleDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *waasv1beta1.CustomProtectionRule,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c customProtectionRuleDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *waasv1beta1.CustomProtectionRule,
) error {
	if c.getCustomProtectionRule == nil || resource == nil {
		return nil
	}
	ruleID := trackedCustomProtectionRuleID(resource)
	if ruleID == "" {
		return nil
	}
	_, err := c.getCustomProtectionRule(ctx, waassdk.GetCustomProtectionRuleRequest{
		CustomProtectionRuleId: common.String(ruleID),
	})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("CustomProtectionRule delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedCustomProtectionRuleID(resource *waasv1beta1.CustomProtectionRule) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
