/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitypolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
)

type securityPolicyOCIClient interface {
	CreateSecurityPolicy(context.Context, datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error)
	GetSecurityPolicy(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error)
	ListSecurityPolicies(context.Context, datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error)
	UpdateSecurityPolicy(context.Context, datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error)
	DeleteSecurityPolicy(context.Context, datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error)
}

type securityPolicyListCall func(context.Context, datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error)

// securityPolicyAmbiguousNotFoundError stays opaque so generatedruntime does not
// reclassify the wrapped OCI 404 as safe to delete after this hook rejects it.
type securityPolicyAmbiguousNotFoundError struct {
	operation string
	err       error
}

func (e securityPolicyAmbiguousNotFoundError) Error() string {
	operation := strings.TrimSpace(e.operation)
	if operation == "" {
		operation = "delete"
	}
	return fmt.Sprintf("SecurityPolicy %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", operation, e.err)
}

func (e securityPolicyAmbiguousNotFoundError) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

type securityPolicySpecNotFound struct {
	compartmentID string
	displayName   string
}

func (e securityPolicySpecNotFound) Error() string {
	return fmt.Sprintf("datasafe SecurityPolicy matching spec was not found for compartmentId %q and displayName %q", e.compartmentID, e.displayName)
}

func (e securityPolicySpecNotFound) Is(target error) bool {
	return target != nil && target.Error() == "generated runtime resource not found"
}

func init() {
	registerSecurityPolicyRuntimeHooksMutator(func(_ *SecurityPolicyServiceManager, hooks *SecurityPolicyRuntimeHooks) {
		applySecurityPolicyRuntimeHooks(hooks)
	})
}

func applySecurityPolicyRuntimeHooks(hooks *SecurityPolicyRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = securityPolicyRuntimeSemantics()
	hooks.Create.Fields = securityPolicyCreateFields()
	hooks.Get.Fields = securityPolicyGetFields()
	hooks.List.Fields = securityPolicyListFields()
	hooks.Update.Fields = securityPolicyUpdateFields()
	hooks.Delete.Fields = securityPolicyDeleteFields()
	hooks.BuildUpdateBody = buildSecurityPolicyUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardSecurityPolicyExistingBeforeCreate
	hooks.StatusHooks.ClearProjectedStatus = clearSecurityPolicyProjectedStatus
	hooks.StatusHooks.RestoreStatus = restoreSecurityPolicyProjectedStatus
	hooks.StatusHooks.ProjectStatus = projectSecurityPolicyStatus
	wrapSecurityPolicyListCall(hooks)
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *datasafev1beta1.SecurityPolicy, currentID string) (any, error) {
		return confirmSecurityPolicyDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleSecurityPolicyDeleteError
	hooks.DeleteHooks.ApplyOutcome = applySecurityPolicyDeleteOutcome
}

func newSecurityPolicyServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client securityPolicyOCIClient,
) SecurityPolicyServiceClient {
	hooks := newSecurityPolicyRuntimeHooksWithOCIClient(client)
	applySecurityPolicyRuntimeHooks(&hooks)
	return defaultSecurityPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SecurityPolicy](
			buildSecurityPolicyGeneratedRuntimeConfig(&SecurityPolicyServiceManager{Log: log}, hooks),
		),
	}
}

func newSecurityPolicyRuntimeHooksWithOCIClient(client securityPolicyOCIClient) SecurityPolicyRuntimeHooks {
	hooks := newSecurityPolicyDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	hooks.Create.Call = func(ctx context.Context, request datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error) {
		if client == nil {
			return datasafesdk.CreateSecurityPolicyResponse{}, fmt.Errorf("SecurityPolicy OCI client is not configured")
		}
		return client.CreateSecurityPolicy(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
		if client == nil {
			return datasafesdk.GetSecurityPolicyResponse{}, fmt.Errorf("SecurityPolicy OCI client is not configured")
		}
		return client.GetSecurityPolicy(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
		if client == nil {
			return datasafesdk.ListSecurityPoliciesResponse{}, fmt.Errorf("SecurityPolicy OCI client is not configured")
		}
		return client.ListSecurityPolicies(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
		if client == nil {
			return datasafesdk.UpdateSecurityPolicyResponse{}, fmt.Errorf("SecurityPolicy OCI client is not configured")
		}
		return client.UpdateSecurityPolicy(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error) {
		if client == nil {
			return datasafesdk.DeleteSecurityPolicyResponse{}, fmt.Errorf("SecurityPolicy OCI client is not configured")
		}
		return client.DeleteSecurityPolicy(ctx, request)
	}
	return hooks
}

func securityPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datasafe",
		FormalSlug:    "securitypolicy",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.SecurityPolicyLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.SecurityPolicyLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.SecurityPolicyLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(datasafesdk.SecurityPolicyLifecycleStateCreating),
				string(datasafesdk.SecurityPolicyLifecycleStateUpdating),
				string(datasafesdk.SecurityPolicyLifecycleStateDeleting),
			},
			TerminalStates: []string{string(datasafesdk.SecurityPolicyLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "SecurityPolicy", Action: "CreateSecurityPolicy"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "SecurityPolicy", Action: "UpdateSecurityPolicy"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "SecurityPolicy", Action: "DeleteSecurityPolicy"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "SecurityPolicy", Action: "GetSecurityPolicy"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "SecurityPolicy", Action: "GetSecurityPolicy"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "SecurityPolicy", Action: "GetSecurityPolicy"}},
		},
	}
}

func securityPolicyCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSecurityPolicyDetails", RequestName: "CreateSecurityPolicyDetails", Contribution: "body"},
	}
}

func securityPolicyGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityPolicyId", RequestName: "securityPolicyId", Contribution: "path", PreferResourceID: true},
	}
}

func securityPolicyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "status.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"spec.displayName", "status.displayName", "displayName"},
		},
	}
}

func securityPolicyUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityPolicyId", RequestName: "securityPolicyId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSecurityPolicyDetails", RequestName: "UpdateSecurityPolicyDetails", Contribution: "body"},
	}
}

func securityPolicyDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityPolicyId", RequestName: "securityPolicyId", Contribution: "path", PreferResourceID: true},
	}
}

type securityPolicyObservedMutableState struct {
	displayName  string
	description  string
	freeformTags map[string]string
	definedTags  map[string]map[string]string
}

func buildSecurityPolicyUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.SecurityPolicy,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("build SecurityPolicy update body: resource is nil")
	}

	current := observedSecurityPolicyMutableState(resource, currentResponse)
	body := map[string]any{}
	addSecurityPolicyStringUpdates(body, resource.Spec, current)
	addSecurityPolicyTagUpdates(body, resource.Spec, current)
	if len(body) == 0 {
		return nil, false, nil
	}
	return body, true, nil
}

func addSecurityPolicyStringUpdates(
	body map[string]any,
	spec datasafev1beta1.SecurityPolicySpec,
	current securityPolicyObservedMutableState,
) {
	if securityPolicyShouldUpdateString(spec.DisplayName, current.displayName) {
		body["displayName"] = spec.DisplayName
	}
	if securityPolicyShouldUpdateString(spec.Description, current.description) {
		body["description"] = spec.Description
	}
}

func securityPolicyShouldUpdateString(desired string, observed string) bool {
	return desired != observed
}

func addSecurityPolicyTagUpdates(
	body map[string]any,
	spec datasafev1beta1.SecurityPolicySpec,
	current securityPolicyObservedMutableState,
) {
	if spec.FreeformTags != nil && !securityPolicyStringMapsEqual(spec.FreeformTags, current.freeformTags) {
		body["freeformTags"] = copySecurityPolicyStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags == nil {
		return
	}
	desired := securityPolicyDefinedTagsFromSpec(spec.DefinedTags)
	if !securityPolicyNestedStringMapsEqual(desired, current.definedTags) {
		body["definedTags"] = securityPolicyDefinedTagsUpdateBody(spec.DefinedTags)
	}
}

func observedSecurityPolicyMutableState(
	resource *datasafev1beta1.SecurityPolicy,
	currentResponse any,
) securityPolicyObservedMutableState {
	current := securityPolicyObservedMutableState{}
	if resource != nil {
		current.displayName = resource.Status.DisplayName
		current.description = resource.Status.Description
		current.freeformTags = copySecurityPolicyStringMap(resource.Status.FreeformTags)
		current.definedTags = securityPolicyDefinedTagsFromSpec(resource.Status.DefinedTags)
	}

	switch response := currentResponse.(type) {
	case datasafesdk.GetSecurityPolicyResponse:
		return securityPolicyMutableStateFromSDK(response.SecurityPolicy)
	case *datasafesdk.GetSecurityPolicyResponse:
		if response != nil {
			return securityPolicyMutableStateFromSDK(response.SecurityPolicy)
		}
	case datasafesdk.SecurityPolicy:
		return securityPolicyMutableStateFromSDK(response)
	case *datasafesdk.SecurityPolicy:
		if response != nil {
			return securityPolicyMutableStateFromSDK(*response)
		}
	}
	return current
}

func securityPolicyMutableStateFromSDK(policy datasafesdk.SecurityPolicy) securityPolicyObservedMutableState {
	return securityPolicyObservedMutableState{
		displayName:  securityPolicyStringValue(policy.DisplayName),
		description:  securityPolicyStringValue(policy.Description),
		freeformTags: copySecurityPolicyStringMap(policy.FreeformTags),
		definedTags:  securityPolicyDefinedTagsFromSDK(policy.DefinedTags),
	}
}

func copySecurityPolicyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func securityPolicyDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]string {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]string, len(tags))
	for namespace, values := range tags {
		converted[namespace] = copySecurityPolicyStringMap(values)
	}
	return converted
}

func securityPolicyDefinedTagsFromSDK(tags map[string]map[string]interface{}) map[string]map[string]string {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]string, len(tags))
	for namespace, values := range tags {
		namespaceValues := make(map[string]string, len(values))
		for key, value := range values {
			namespaceValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = namespaceValues
	}
	return converted
}

func securityPolicyDefinedTagsUpdateBody(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	body := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		namespaceValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			namespaceValues[key] = value
		}
		body[namespace] = namespaceValues
	}
	return body
}

func securityPolicyStringMapsEqual(left map[string]string, right map[string]string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if rightValue, ok := right[key]; !ok || rightValue != leftValue {
			return false
		}
	}
	return true
}

func securityPolicyNestedStringMapsEqual(left map[string]map[string]string, right map[string]map[string]string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	if len(left) != len(right) {
		return false
	}
	for namespace, leftValues := range left {
		rightValues, ok := right[namespace]
		if !ok || !securityPolicyStringMapsEqual(leftValues, rightValues) {
			return false
		}
	}
	return true
}

func clearSecurityPolicyProjectedStatus(resource *datasafev1beta1.SecurityPolicy) any {
	if resource == nil {
		return nil
	}
	baseline := resource.Status
	resource.Status = datasafev1beta1.SecurityPolicyStatus{OsokStatus: baseline.OsokStatus}
	return baseline
}

func restoreSecurityPolicyProjectedStatus(resource *datasafev1beta1.SecurityPolicy, baseline any) {
	if resource == nil {
		return
	}
	if status, ok := baseline.(datasafev1beta1.SecurityPolicyStatus); ok {
		osokStatus := resource.Status.OsokStatus
		resource.Status = status
		resource.Status.OsokStatus = osokStatus
	}
}

func projectSecurityPolicyStatus(resource *datasafev1beta1.SecurityPolicy, response any) error {
	if resource == nil {
		return fmt.Errorf("SecurityPolicy resource is nil")
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current, ok := securityPolicyFromResponse(response)
	if !ok {
		return nil
	}

	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.SecurityPolicyStatus{
		OsokStatus:         osokStatus,
		Id:                 securityPolicyStringValue(current.Id),
		CompartmentId:      securityPolicyStringValue(current.CompartmentId),
		DisplayName:        securityPolicyStringValue(current.DisplayName),
		TimeCreated:        securityPolicyTimeString(current.TimeCreated),
		LifecycleState:     string(current.LifecycleState),
		Description:        securityPolicyStringValue(current.Description),
		SecurityPolicyType: string(current.SecurityPolicyType),
		TimeUpdated:        securityPolicyTimeString(current.TimeUpdated),
		LifecycleDetails:   securityPolicyStringValue(current.LifecycleDetails),
		FreeformTags:       copySecurityPolicyStringMap(current.FreeformTags),
		DefinedTags:        securityPolicyStatusTagsFromSDK(current.DefinedTags),
		SystemTags:         securityPolicyStatusTagsFromSDK(current.SystemTags),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func securityPolicyFromResponse(response any) (datasafesdk.SecurityPolicy, bool) {
	if current, ok := securityPolicyFromModelResponse(response); ok {
		return current, true
	}
	return securityPolicyFromOperationResponse(response)
}

func securityPolicyFromModelResponse(response any) (datasafesdk.SecurityPolicy, bool) {
	switch current := response.(type) {
	case datasafesdk.SecurityPolicy:
		return current, true
	case *datasafesdk.SecurityPolicy:
		if current != nil {
			return *current, true
		}
	case datasafesdk.SecurityPolicySummary:
		return securityPolicyFromSummary(current), true
	case *datasafesdk.SecurityPolicySummary:
		if current != nil {
			return securityPolicyFromSummary(*current), true
		}
	}
	return datasafesdk.SecurityPolicy{}, false
}

func securityPolicyFromOperationResponse(response any) (datasafesdk.SecurityPolicy, bool) {
	switch current := response.(type) {
	case datasafesdk.CreateSecurityPolicyResponse:
		return current.SecurityPolicy, true
	case *datasafesdk.CreateSecurityPolicyResponse:
		if current != nil {
			return current.SecurityPolicy, true
		}
	case datasafesdk.GetSecurityPolicyResponse:
		return current.SecurityPolicy, true
	case *datasafesdk.GetSecurityPolicyResponse:
		if current != nil {
			return current.SecurityPolicy, true
		}
	}
	return datasafesdk.SecurityPolicy{}, false
}

func securityPolicyFromSummary(summary datasafesdk.SecurityPolicySummary) datasafesdk.SecurityPolicy {
	return datasafesdk.SecurityPolicy{
		Id:                 summary.Id,
		CompartmentId:      summary.CompartmentId,
		DisplayName:        summary.DisplayName,
		TimeCreated:        summary.TimeCreated,
		LifecycleState:     summary.LifecycleState,
		Description:        summary.Description,
		SecurityPolicyType: summary.SecurityPolicyType,
		TimeUpdated:        summary.TimeUpdated,
		LifecycleDetails:   summary.LifecycleDetails,
		FreeformTags:       copySecurityPolicyStringMap(summary.FreeformTags),
		DefinedTags:        summary.DefinedTags,
		SystemTags:         summary.SystemTags,
	}
}

func securityPolicyStatusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		namespaceValues := make(shared.MapValue, len(values))
		for key, value := range values {
			namespaceValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = namespaceValues
	}
	return converted
}

func securityPolicyTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func guardSecurityPolicyExistingBeforeCreate(
	_ context.Context,
	resource *datasafev1beta1.SecurityPolicy,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil || strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
}

func wrapSecurityPolicyListCall(hooks *SecurityPolicyRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	delegate := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
		return listSecurityPoliciesAllPages(ctx, request, delegate)
	}
}

func listSecurityPoliciesAllPages(
	ctx context.Context,
	request datasafesdk.ListSecurityPoliciesRequest,
	call securityPolicyListCall,
) (datasafesdk.ListSecurityPoliciesResponse, error) {
	if call == nil {
		return datasafesdk.ListSecurityPoliciesResponse{}, fmt.Errorf("SecurityPolicy list hook is not configured")
	}

	var merged datasafesdk.ListSecurityPoliciesResponse
	seenPages := map[string]bool{}
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if merged.RawResponse == nil {
			merged.RawResponse = response.RawResponse
		}
		if merged.OpcRequestId == nil {
			merged.OpcRequestId = response.OpcRequestId
		}
		if merged.OpcPrevPage == nil {
			merged.OpcPrevPage = response.OpcPrevPage
		}
		merged.Items = append(
			merged.Items,
			response.Items...,
		)

		nextPage := strings.TrimSpace(securityPolicyStringValue(response.OpcNextPage))
		if nextPage == "" {
			merged.OpcNextPage = nil
			return merged, nil
		}
		if seenPages[nextPage] {
			return merged, fmt.Errorf("SecurityPolicy list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = true
		request.Page = common.String(nextPage)
	}
}

func confirmSecurityPolicyDeleteRead(
	ctx context.Context,
	hooks *SecurityPolicyRuntimeHooks,
	resource *datasafev1beta1.SecurityPolicy,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm SecurityPolicy delete: runtime hooks are nil")
	}

	currentID = strings.TrimSpace(currentID)
	if currentID != "" {
		response, err := hooks.Get.Call(ctx, datasafesdk.GetSecurityPolicyRequest{
			SecurityPolicyId: common.String(currentID),
		})
		if handled, handledResponse, handledErr := handleSecurityPolicyConfirmReadError(resource, err, "delete confirmation"); handled || handledErr != nil {
			return handledResponse, handledErr
		}
		return response, nil
	}

	return confirmSecurityPolicyDeleteReadByList(ctx, hooks, resource)
}

func confirmSecurityPolicyDeleteReadByList(
	ctx context.Context,
	hooks *SecurityPolicyRuntimeHooks,
	resource *datasafev1beta1.SecurityPolicy,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("confirm SecurityPolicy delete: resource is nil")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if compartmentID == "" || displayName == "" {
		return nil, fmt.Errorf("confirm SecurityPolicy delete: status OCID is empty and spec compartmentId/displayName are not both set")
	}

	response, err := hooks.List.Call(ctx, datasafesdk.ListSecurityPoliciesRequest{
		CompartmentId: common.String(compartmentID),
		DisplayName:   common.String(displayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			ambiguous := securityPolicyAmbiguousNotFoundError{operation: "delete confirmation list", err: err}
			recordSecurityPolicyDeleteErrorRequestID(resource, ambiguous)
			return nil, ambiguous
		}
		return nil, err
	}
	return selectSecurityPolicyDeleteMatch(response.Items, resource.Spec)
}

func selectSecurityPolicyDeleteMatch(
	items []datasafesdk.SecurityPolicySummary,
	spec datasafev1beta1.SecurityPolicySpec,
) (datasafesdk.SecurityPolicySummary, error) {
	matches := make([]datasafesdk.SecurityPolicySummary, 0, len(items))
	for _, item := range items {
		if securityPolicySummaryMatchesSpec(item, spec) {
			matches = append(matches, item)
		}
	}

	compartmentID := strings.TrimSpace(spec.CompartmentId)
	displayName := strings.TrimSpace(spec.DisplayName)
	switch len(matches) {
	case 0:
		return datasafesdk.SecurityPolicySummary{}, securityPolicySpecNotFound{compartmentID: compartmentID, displayName: displayName}
	case 1:
		return matches[0], nil
	default:
		return datasafesdk.SecurityPolicySummary{}, fmt.Errorf("confirm SecurityPolicy delete: multiple OCI security policies matched compartmentId %q and displayName %q", compartmentID, displayName)
	}
}

func handleSecurityPolicyConfirmReadError(
	resource *datasafev1beta1.SecurityPolicy,
	err error,
	operation string,
) (bool, any, error) {
	if err == nil {
		return false, nil, nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return false, nil, err
	}
	ambiguous := securityPolicyAmbiguousNotFoundError{operation: operation, err: err}
	recordSecurityPolicyDeleteErrorRequestID(resource, ambiguous)
	return true, ambiguous, nil
}

func handleSecurityPolicyDeleteError(resource *datasafev1beta1.SecurityPolicy, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous, ok := securityPolicyAmbiguousError(err); ok {
		recordSecurityPolicyDeleteErrorRequestID(resource, ambiguous)
		return ambiguous
	}
	if securityPolicyIsSpecNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	ambiguous := securityPolicyAmbiguousNotFoundError{operation: "delete", err: err}
	recordSecurityPolicyDeleteErrorRequestID(resource, ambiguous)
	return ambiguous
}

func securityPolicyAmbiguousError(err error) (securityPolicyAmbiguousNotFoundError, bool) {
	switch typed := err.(type) {
	case securityPolicyAmbiguousNotFoundError:
		return typed, true
	case *securityPolicyAmbiguousNotFoundError:
		if typed != nil {
			return *typed, true
		}
	}
	return securityPolicyAmbiguousNotFoundError{}, false
}

func securityPolicyIsSpecNotFound(err error) bool {
	switch typed := err.(type) {
	case securityPolicySpecNotFound:
		return true
	case *securityPolicySpecNotFound:
		return typed != nil
	default:
		return false
	}
}

func applySecurityPolicyDeleteOutcome(
	resource *datasafev1beta1.SecurityPolicy,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	ambiguous, ok := response.(securityPolicyAmbiguousNotFoundError)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	recordSecurityPolicyDeleteErrorRequestID(resource, ambiguous)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
}

func recordSecurityPolicyDeleteErrorRequestID(resource *datasafev1beta1.SecurityPolicy, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func securityPolicySummaryMatchesSpec(
	summary datasafesdk.SecurityPolicySummary,
	spec datasafev1beta1.SecurityPolicySpec,
) bool {
	return strings.TrimSpace(securityPolicyStringValue(summary.CompartmentId)) == strings.TrimSpace(spec.CompartmentId) &&
		strings.TrimSpace(securityPolicyStringValue(summary.DisplayName)) == strings.TrimSpace(spec.DisplayName)
}

func securityPolicyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
