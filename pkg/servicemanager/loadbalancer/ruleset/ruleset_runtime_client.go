/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ruleset

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const ruleSetLoadBalancerIDAnnotation = "loadbalancer.oracle.com/load-balancer-id"

const (
	ruleSetDefaultRedirectResponseCode      = 302
	ruleSetDefaultControlAccessStatusCode   = 405
	ruleSetDefaultInvalidHeaderCharsAllowed = false
)

type ruleSetRuntimeOCIClient interface {
	CreateRuleSet(context.Context, loadbalancersdk.CreateRuleSetRequest) (loadbalancersdk.CreateRuleSetResponse, error)
	GetRuleSet(context.Context, loadbalancersdk.GetRuleSetRequest) (loadbalancersdk.GetRuleSetResponse, error)
	ListRuleSets(context.Context, loadbalancersdk.ListRuleSetsRequest) (loadbalancersdk.ListRuleSetsResponse, error)
	UpdateRuleSet(context.Context, loadbalancersdk.UpdateRuleSetRequest) (loadbalancersdk.UpdateRuleSetResponse, error)
	DeleteRuleSet(context.Context, loadbalancersdk.DeleteRuleSetRequest) (loadbalancersdk.DeleteRuleSetResponse, error)
}

type ruleSetIdentity struct {
	loadBalancerID string
	ruleSetName    string
}

func init() {
	registerRuleSetRuntimeHooksMutator(func(_ *RuleSetServiceManager, hooks *RuleSetRuntimeHooks) {
		applyRuleSetRuntimeHooks(hooks)
	})
}

func applyRuleSetRuntimeHooks(hooks *RuleSetRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newRuleSetRuntimeSemantics()
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.RuleSet,
		_ string,
	) (any, error) {
		return buildRuleSetCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.RuleSet,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildRuleSetUpdateBody(resource, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.RuleSet]{
		Resolve: func(resource *loadbalancerv1beta1.RuleSet) (any, error) {
			return resolveRuleSetIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.RuleSet, identity any) {
			recordRuleSetPathIdentity(resource, identity.(ruleSetIdentity))
		},
		RecordTracked: func(resource *loadbalancerv1beta1.RuleSet, identity any, _ string) {
			recordRuleSetTrackedIdentity(resource, identity.(ruleSetIdentity))
		},
		LookupExisting: func(context.Context, *loadbalancerv1beta1.RuleSet, any) (any, error) {
			return nil, nil
		},
	}
	hooks.Create.Fields = ruleSetCreateFields()
	hooks.Get.Fields = ruleSetGetFields()
	hooks.List.Fields = ruleSetListFields()
	hooks.Update.Fields = ruleSetUpdateFields()
	hooks.Delete.Fields = ruleSetDeleteFields()
}

func newRuleSetRuntimeHooksWithOCIClient(client ruleSetRuntimeOCIClient) RuleSetRuntimeHooks {
	return RuleSetRuntimeHooks{
		Semantics: newRuleSetRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.RuleSet]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[loadbalancersdk.CreateRuleSetRequest, loadbalancersdk.CreateRuleSetResponse]{
			Fields: ruleSetCreateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.CreateRuleSetRequest) (loadbalancersdk.CreateRuleSetResponse, error) {
				return client.CreateRuleSet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loadbalancersdk.GetRuleSetRequest, loadbalancersdk.GetRuleSetResponse]{
			Fields: ruleSetGetFields(),
			Call: func(ctx context.Context, request loadbalancersdk.GetRuleSetRequest) (loadbalancersdk.GetRuleSetResponse, error) {
				return client.GetRuleSet(ctx, request)
			},
		},
		List: runtimeOperationHooks[loadbalancersdk.ListRuleSetsRequest, loadbalancersdk.ListRuleSetsResponse]{
			Fields: ruleSetListFields(),
			Call: func(ctx context.Context, request loadbalancersdk.ListRuleSetsRequest) (loadbalancersdk.ListRuleSetsResponse, error) {
				return client.ListRuleSets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdateRuleSetRequest, loadbalancersdk.UpdateRuleSetResponse]{
			Fields: ruleSetUpdateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.UpdateRuleSetRequest) (loadbalancersdk.UpdateRuleSetResponse, error) {
				return client.UpdateRuleSet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeleteRuleSetRequest, loadbalancersdk.DeleteRuleSetResponse]{
			Fields: ruleSetDeleteFields(),
			Call: func(ctx context.Context, request loadbalancersdk.DeleteRuleSetRequest) (loadbalancersdk.DeleteRuleSetResponse, error) {
				return client.DeleteRuleSet(ctx, request)
			},
		},
		WrapGeneratedClient: []func(RuleSetServiceClient) RuleSetServiceClient{},
	}
}

func newRuleSetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "ruleset",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{},
			UpdatingStates:     []string{},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"items"},
			ForceNew:      []string{"name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func ruleSetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ruleSetLoadBalancerIDField(),
		{
			FieldName:    "CreateRuleSetDetails",
			RequestName:  "CreateRuleSetDetails",
			Contribution: "body",
		},
	}
}

func ruleSetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ruleSetLoadBalancerIDField(),
		ruleSetNameField(),
	}
}

func ruleSetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ruleSetLoadBalancerIDField(),
	}
}

func ruleSetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ruleSetLoadBalancerIDField(),
		ruleSetNameField(),
		{
			FieldName:    "UpdateRuleSetDetails",
			RequestName:  "UpdateRuleSetDetails",
			Contribution: "body",
		},
	}
}

func ruleSetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ruleSetLoadBalancerIDField(),
		ruleSetNameField(),
	}
}

func ruleSetLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "LoadBalancerId",
		RequestName:      "loadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.status.ocid"},
	}
}

func ruleSetNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "RuleSetName",
		RequestName:  "ruleSetName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func buildRuleSetCreateBody(resource *loadbalancerv1beta1.RuleSet) (loadbalancersdk.CreateRuleSetDetails, error) {
	if resource == nil {
		return loadbalancersdk.CreateRuleSetDetails{}, fmt.Errorf("ruleset resource is nil")
	}

	items, err := ruleSetSDKRules(resource.Spec.Items)
	if err != nil {
		return loadbalancersdk.CreateRuleSetDetails{}, err
	}
	return loadbalancersdk.CreateRuleSetDetails{
		Name:  stringPointer(firstNonEmptyTrim(resource.Spec.Name, resource.Name)),
		Items: items,
	}, nil
}

func buildRuleSetUpdateBody(
	resource *loadbalancerv1beta1.RuleSet,
	currentResponse any,
) (loadbalancersdk.UpdateRuleSetDetails, bool, error) {
	if resource == nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, false, fmt.Errorf("ruleset resource is nil")
	}

	items, err := ruleSetSDKRules(resource.Spec.Items)
	if err != nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, false, err
	}
	desired := loadbalancersdk.UpdateRuleSetDetails{Items: items}

	currentSource, err := ruleSetUpdateSource(resource, currentResponse)
	if err != nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, false, err
	}
	current, err := ruleSetUpdateDetailsFromValue(currentSource)
	if err != nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, false, err
	}

	updateNeeded, err := ruleSetUpdateNeeded(desired, current)
	if err != nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, false, err
	}
	if !updateNeeded {
		return loadbalancersdk.UpdateRuleSetDetails{}, false, nil
	}
	return desired, true, nil
}

func ruleSetUpdateSource(resource *loadbalancerv1beta1.RuleSet, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("ruleset resource is nil")
		}
		return resource.Status, nil
	case loadbalancersdk.RuleSet:
		return current, nil
	case *loadbalancersdk.RuleSet:
		if current == nil {
			return nil, fmt.Errorf("current RuleSet response is nil")
		}
		return *current, nil
	case loadbalancersdk.GetRuleSetResponse:
		return current.RuleSet, nil
	case *loadbalancersdk.GetRuleSetResponse:
		if current == nil {
			return nil, fmt.Errorf("current RuleSet response is nil")
		}
		return current.RuleSet, nil
	default:
		return currentResponse, nil
	}
}

func ruleSetUpdateDetailsFromValue(value any) (loadbalancersdk.UpdateRuleSetDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, fmt.Errorf("marshal RuleSet update details source: %w", err)
	}

	var details loadbalancersdk.UpdateRuleSetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return loadbalancersdk.UpdateRuleSetDetails{}, fmt.Errorf("decode RuleSet update details: %w", err)
	}
	return details, nil
}

func ruleSetUpdateNeeded(desired loadbalancersdk.UpdateRuleSetDetails, current loadbalancersdk.UpdateRuleSetDetails) (bool, error) {
	desired = normalizeRuleSetUpdateDetails(desired)
	current = normalizeRuleSetUpdateDetails(current)

	desiredPayload, err := json.Marshal(desired)
	if err != nil {
		return false, fmt.Errorf("marshal desired RuleSet update details: %w", err)
	}
	currentPayload, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal current RuleSet update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func normalizeRuleSetUpdateDetails(details loadbalancersdk.UpdateRuleSetDetails) loadbalancersdk.UpdateRuleSetDetails {
	return loadbalancersdk.UpdateRuleSetDetails{
		Items: normalizeRuleSetRules(details.Items),
	}
}

func normalizeRuleSetRules(rules []loadbalancersdk.Rule) []loadbalancersdk.Rule {
	if rules == nil {
		return nil
	}

	normalized := make([]loadbalancersdk.Rule, len(rules))
	for i, rule := range rules {
		normalized[i] = normalizeRuleSetRule(rule)
	}
	return normalized
}

func normalizeRuleSetRule(rule loadbalancersdk.Rule) loadbalancersdk.Rule {
	switch typed := rule.(type) {
	case loadbalancersdk.RedirectRule:
		typed.Conditions = normalizeRuleSetConditions(typed.Conditions)
		typed.ResponseCode = intPointerWithDefault(typed.ResponseCode, ruleSetDefaultRedirectResponseCode)
		return typed
	case *loadbalancersdk.RedirectRule:
		if typed == nil {
			return typed
		}
		copied := *typed
		copied.Conditions = normalizeRuleSetConditions(copied.Conditions)
		copied.ResponseCode = intPointerWithDefault(copied.ResponseCode, ruleSetDefaultRedirectResponseCode)
		return copied
	case loadbalancersdk.ControlAccessUsingHttpMethodsRule:
		typed.StatusCode = intPointerWithDefault(typed.StatusCode, ruleSetDefaultControlAccessStatusCode)
		return typed
	case *loadbalancersdk.ControlAccessUsingHttpMethodsRule:
		if typed == nil {
			return typed
		}
		copied := *typed
		copied.StatusCode = intPointerWithDefault(copied.StatusCode, ruleSetDefaultControlAccessStatusCode)
		return copied
	case loadbalancersdk.HttpHeaderRule:
		typed.AreInvalidCharactersAllowed = boolPointerWithDefault(
			typed.AreInvalidCharactersAllowed,
			ruleSetDefaultInvalidHeaderCharsAllowed,
		)
		return typed
	case *loadbalancersdk.HttpHeaderRule:
		if typed == nil {
			return typed
		}
		copied := *typed
		copied.AreInvalidCharactersAllowed = boolPointerWithDefault(
			copied.AreInvalidCharactersAllowed,
			ruleSetDefaultInvalidHeaderCharsAllowed,
		)
		return copied
	case loadbalancersdk.AllowRule:
		typed.Conditions = normalizeRuleSetConditions(typed.Conditions)
		return typed
	case *loadbalancersdk.AllowRule:
		if typed == nil {
			return typed
		}
		copied := *typed
		copied.Conditions = normalizeRuleSetConditions(copied.Conditions)
		return copied
	default:
		return rule
	}
}

func normalizeRuleSetConditions(conditions []loadbalancersdk.RuleCondition) []loadbalancersdk.RuleCondition {
	if conditions == nil {
		return nil
	}

	normalized := make([]loadbalancersdk.RuleCondition, len(conditions))
	for i, condition := range conditions {
		normalized[i] = normalizeRuleSetCondition(condition)
	}
	return normalized
}

func normalizeRuleSetCondition(condition loadbalancersdk.RuleCondition) loadbalancersdk.RuleCondition {
	switch typed := condition.(type) {
	case *loadbalancersdk.SourceIpAddressCondition:
		if typed == nil {
			return typed
		}
		return *typed
	case *loadbalancersdk.SourceVcnIdCondition:
		if typed == nil {
			return typed
		}
		return *typed
	case *loadbalancersdk.SourceVcnIpAddressCondition:
		if typed == nil {
			return typed
		}
		return *typed
	case *loadbalancersdk.PathMatchCondition:
		if typed == nil {
			return typed
		}
		return *typed
	default:
		return condition
	}
}

func ruleSetSDKRules(items []loadbalancerv1beta1.RuleSetItem) ([]loadbalancersdk.Rule, error) {
	if items == nil {
		return nil, nil
	}

	converted := make([]loadbalancersdk.Rule, 0, len(items))
	for i, item := range items {
		rule, err := ruleSetSDKRule(item)
		if err != nil {
			return nil, fmt.Errorf("items[%d]: %w", i, err)
		}
		converted = append(converted, rule)
	}
	return converted, nil
}

func ruleSetSDKRule(item loadbalancerv1beta1.RuleSetItem) (loadbalancersdk.Rule, error) {
	switch strings.ToUpper(strings.TrimSpace(item.Action)) {
	case string(loadbalancersdk.RuleActionAddHttpRequestHeader):
		return loadbalancersdk.AddHttpRequestHeaderRule{
			Header: stringPointer(item.Header),
			Value:  stringPointer(item.Value),
		}, nil
	case string(loadbalancersdk.RuleActionExtendHttpRequestHeaderValue):
		return loadbalancersdk.ExtendHttpRequestHeaderValueRule{
			Header: stringPointer(item.Header),
			Prefix: stringPointer(item.Prefix),
			Suffix: stringPointer(item.Suffix),
		}, nil
	case string(loadbalancersdk.RuleActionRemoveHttpRequestHeader):
		return loadbalancersdk.RemoveHttpRequestHeaderRule{
			Header: stringPointer(item.Header),
		}, nil
	case string(loadbalancersdk.RuleActionAddHttpResponseHeader):
		return loadbalancersdk.AddHttpResponseHeaderRule{
			Header: stringPointer(item.Header),
			Value:  stringPointer(item.Value),
		}, nil
	case string(loadbalancersdk.RuleActionExtendHttpResponseHeaderValue):
		return loadbalancersdk.ExtendHttpResponseHeaderValueRule{
			Header: stringPointer(item.Header),
			Prefix: stringPointer(item.Prefix),
			Suffix: stringPointer(item.Suffix),
		}, nil
	case string(loadbalancersdk.RuleActionRemoveHttpResponseHeader):
		return loadbalancersdk.RemoveHttpResponseHeaderRule{
			Header: stringPointer(item.Header),
		}, nil
	case string(loadbalancersdk.RuleActionAllow):
		conditions, err := ruleSetSDKRuleConditions(item.Conditions)
		if err != nil {
			return nil, err
		}
		return loadbalancersdk.AllowRule{
			Conditions:  conditions,
			Description: stringPointer(item.Description),
		}, nil
	case string(loadbalancersdk.RuleActionControlAccessUsingHttpMethods):
		return loadbalancersdk.ControlAccessUsingHttpMethodsRule{
			AllowedMethods: copyStringSlice(item.AllowedMethods),
			StatusCode:     intPointerNonZero(item.StatusCode),
		}, nil
	case string(loadbalancersdk.RuleActionRedirect):
		conditions, err := ruleSetSDKRuleConditions(item.Conditions)
		if err != nil {
			return nil, err
		}
		return loadbalancersdk.RedirectRule{
			Conditions:   conditions,
			ResponseCode: intPointerNonZero(item.ResponseCode),
			RedirectUri:  ruleSetSDKRedirectURI(item.RedirectUri),
		}, nil
	case string(loadbalancersdk.RuleActionHttpHeader):
		return loadbalancersdk.HttpHeaderRule{
			AreInvalidCharactersAllowed: boolPointerTrue(item.AreInvalidCharactersAllowed),
			HttpLargeHeaderSizeInKB:     intPointerNonZero(item.HttpLargeHeaderSizeInKB),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported RuleSet item action %q", item.Action)
	}
}

func ruleSetSDKRuleConditions(items []loadbalancerv1beta1.RuleSetItemCondition) ([]loadbalancersdk.RuleCondition, error) {
	if items == nil {
		return nil, nil
	}

	converted := make([]loadbalancersdk.RuleCondition, 0, len(items))
	for i, item := range items {
		condition, err := ruleSetSDKRuleCondition(item)
		if err != nil {
			return nil, fmt.Errorf("conditions[%d]: %w", i, err)
		}
		converted = append(converted, condition)
	}
	return converted, nil
}

func ruleSetSDKRuleCondition(item loadbalancerv1beta1.RuleSetItemCondition) (loadbalancersdk.RuleCondition, error) {
	switch strings.ToUpper(strings.TrimSpace(item.AttributeName)) {
	case string(loadbalancersdk.RuleConditionAttributeNameSourceIpAddress):
		return loadbalancersdk.SourceIpAddressCondition{
			AttributeValue: stringPointer(item.AttributeValue),
		}, nil
	case string(loadbalancersdk.RuleConditionAttributeNameSourceVcnId):
		return loadbalancersdk.SourceVcnIdCondition{
			AttributeValue: stringPointer(item.AttributeValue),
		}, nil
	case string(loadbalancersdk.RuleConditionAttributeNameSourceVcnIpAddress):
		return loadbalancersdk.SourceVcnIpAddressCondition{
			AttributeValue: stringPointer(item.AttributeValue),
		}, nil
	case string(loadbalancersdk.RuleConditionAttributeNamePath):
		return loadbalancersdk.PathMatchCondition{
			AttributeValue: stringPointer(item.AttributeValue),
			Operator:       loadbalancersdk.PathMatchConditionOperatorEnum(item.Operator),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported RuleSet condition attributeName %q", item.AttributeName)
	}
}

func ruleSetSDKRedirectURI(item loadbalancerv1beta1.RuleSetItemRedirectUri) *loadbalancersdk.RedirectUri {
	if item.Protocol == "" && item.Host == "" && item.Port == 0 && item.Path == "" && item.Query == "" {
		return nil
	}
	return &loadbalancersdk.RedirectUri{
		Protocol: stringPointer(item.Protocol),
		Host:     stringPointer(item.Host),
		Port:     intPointerNonZero(item.Port),
		Path:     stringPointer(item.Path),
		Query:    stringPointer(item.Query),
	}
}

func resolveRuleSetIdentity(resource *loadbalancerv1beta1.RuleSet) (ruleSetIdentity, error) {
	if resource == nil {
		return ruleSetIdentity{}, fmt.Errorf("resolve RuleSet identity: resource is nil")
	}

	statusLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationLoadBalancerID := strings.TrimSpace(resource.Annotations[ruleSetLoadBalancerIDAnnotation])
	if statusLoadBalancerID != "" && annotationLoadBalancerID != "" && statusLoadBalancerID != annotationLoadBalancerID {
		return ruleSetIdentity{}, fmt.Errorf(
			"resolve RuleSet identity: %s changed from recorded loadBalancerId %q to %q",
			ruleSetLoadBalancerIDAnnotation,
			statusLoadBalancerID,
			annotationLoadBalancerID,
		)
	}

	identity := ruleSetIdentity{
		loadBalancerID: firstNonEmptyTrim(statusLoadBalancerID, annotationLoadBalancerID),
		ruleSetName:    firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name, resource.Name),
	}
	if identity.loadBalancerID == "" {
		return ruleSetIdentity{}, fmt.Errorf("resolve RuleSet identity: %s annotation is required", ruleSetLoadBalancerIDAnnotation)
	}
	if identity.ruleSetName == "" {
		return ruleSetIdentity{}, fmt.Errorf("resolve RuleSet identity: rule set name is empty")
	}
	return identity, nil
}

func recordRuleSetPathIdentity(resource *loadbalancerv1beta1.RuleSet, identity ruleSetIdentity) {
	if resource == nil {
		return
	}
	resource.Status.Name = identity.ruleSetName
	// RuleSet has no child OCID in the Load Balancer API; the parent loadBalancerId
	// is the stable path identity used for Get, Update, and Delete.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.loadBalancerID)
}

func recordRuleSetTrackedIdentity(resource *loadbalancerv1beta1.RuleSet, identity ruleSetIdentity) {
	recordRuleSetPathIdentity(resource, identity)
}

func copyStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func intPointerNonZero(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func intPointerWithDefault(value *int, defaultValue int) *int {
	if value != nil {
		return value
	}
	return &defaultValue
}

func boolPointerTrue(value bool) *bool {
	if !value {
		return nil
	}
	return &value
}

func boolPointerWithDefault(value *bool, defaultValue bool) *bool {
	if value != nil {
		return value
	}
	return &defaultValue
}
