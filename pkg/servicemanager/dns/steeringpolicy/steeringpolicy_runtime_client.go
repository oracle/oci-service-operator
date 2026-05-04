/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package steeringpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"

	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type steeringPolicyIdentity struct {
	compartmentID string
	displayName   string
}

type steeringPolicyUpdateMask struct {
	displayName          bool
	ttl                  bool
	healthCheckMonitorID bool
	template             bool
	freeformTags         bool
	definedTags          bool
	answers              bool
	rules                bool
}

type steeringPolicyListCall func(context.Context, dnssdk.ListSteeringPoliciesRequest) (dnssdk.ListSteeringPoliciesResponse, error)

func init() {
	registerSteeringPolicyRuntimeHooksMutator(func(_ *SteeringPolicyServiceManager, hooks *SteeringPolicyRuntimeHooks) {
		applySteeringPolicyRuntimeHooks(hooks)
	})
}

func applySteeringPolicyRuntimeHooks(hooks *SteeringPolicyRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newSteeringPolicyRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *dnsv1beta1.SteeringPolicy, _ string) (any, error) {
		return buildSteeringPolicyCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *dnsv1beta1.SteeringPolicy, _ string, currentResponse any) (any, bool, error) {
		return buildSteeringPolicyUpdateBody(resource, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*dnsv1beta1.SteeringPolicy]{
		Resolve: func(resource *dnsv1beta1.SteeringPolicy) (any, error) {
			return resolveSteeringPolicyIdentity(resource)
		},
		LookupExisting: func(ctx context.Context, resource *dnsv1beta1.SteeringPolicy, identity any) (any, error) {
			return lookupExistingSteeringPolicy(ctx, hooks, resource, identity)
		},
	}
	hooks.DeleteHooks.HandleError = handleSteeringPolicyDeleteError
}

func newSteeringPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:       "dns",
		FormalSlug:          "steeringpolicy",
		StatusProjection:    "required",
		SecretSideEffects:   "none",
		FinalizerPolicy:     "retain-until-confirmed-delete",
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"ttl",
				"healthCheckMonitorId",
				"template",
				"freeformTags",
				"definedTags",
				"answers",
				"rules",
			},
			ForceNew:      []string{"compartmentId"},
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
	}
}

func resolveSteeringPolicyIdentity(resource *dnsv1beta1.SteeringPolicy) (steeringPolicyIdentity, error) {
	if resource == nil {
		return steeringPolicyIdentity{}, fmt.Errorf("resolve SteeringPolicy identity: resource is nil")
	}
	identity := steeringPolicyIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
	if identity.compartmentID == "" {
		return steeringPolicyIdentity{}, fmt.Errorf("resolve SteeringPolicy identity: compartmentId is required")
	}
	if identity.displayName == "" {
		return steeringPolicyIdentity{}, fmt.Errorf("resolve SteeringPolicy identity: displayName is required")
	}
	return identity, nil
}

func lookupExistingSteeringPolicy(
	ctx context.Context,
	hooks *SteeringPolicyRuntimeHooks,
	_ *dnsv1beta1.SteeringPolicy,
	identity any,
) (any, error) {
	typedIdentity, err := steeringPolicyIdentityFromValue(identity)
	if err != nil {
		return nil, err
	}
	if hooks == nil || hooks.List.Call == nil {
		return nil, nil
	}

	matches, err := listMatchingSteeringPolicies(ctx, hooks.List.Call, typedIdentity)
	if err != nil {
		return nil, err
	}
	return singleSteeringPolicyMatch(matches, typedIdentity)
}

func steeringPolicyIdentityFromValue(identity any) (steeringPolicyIdentity, error) {
	typedIdentity, ok := identity.(steeringPolicyIdentity)
	if !ok {
		return steeringPolicyIdentity{}, fmt.Errorf("resolve SteeringPolicy identity: expected steeringPolicyIdentity, got %T", identity)
	}
	return typedIdentity, nil
}

func listMatchingSteeringPolicies(
	ctx context.Context,
	list steeringPolicyListCall,
	identity steeringPolicyIdentity,
) ([]dnssdk.SteeringPolicySummary, error) {
	var matches []dnssdk.SteeringPolicySummary
	var page *string
	for {
		response, err := list(ctx, dnssdk.ListSteeringPoliciesRequest{
			CompartmentId: stringPointer(identity.compartmentID),
			DisplayName:   stringPointer(identity.displayName),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range response.Items {
			if steeringPolicySummaryMatchesIdentity(item, identity) {
				matches = append(matches, item)
			}
		}
		if steeringPolicyListPageDone(response.OpcNextPage) {
			break
		}
		page = response.OpcNextPage
	}
	return matches, nil
}

func steeringPolicyListPageDone(nextPage *string) bool {
	return nextPage == nil || strings.TrimSpace(*nextPage) == ""
}

func singleSteeringPolicyMatch(matches []dnssdk.SteeringPolicySummary, identity steeringPolicyIdentity) (any, error) {
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("SteeringPolicy list response returned multiple matching resources for compartmentId %q and displayName %q", identity.compartmentID, identity.displayName)
	}
}

func steeringPolicySummaryMatchesIdentity(summary dnssdk.SteeringPolicySummary, identity steeringPolicyIdentity) bool {
	if strings.EqualFold(string(summary.LifecycleState), string(dnssdk.SteeringPolicySummaryLifecycleStateDeleted)) {
		return false
	}
	return stringPointerValue(summary.CompartmentId) == identity.compartmentID &&
		stringPointerValue(summary.DisplayName) == identity.displayName
}

func buildSteeringPolicyCreateBody(resource *dnsv1beta1.SteeringPolicy) (dnssdk.CreateSteeringPolicyDetails, error) {
	if resource == nil {
		return dnssdk.CreateSteeringPolicyDetails{}, fmt.Errorf("SteeringPolicy resource is nil")
	}

	answers := steeringPolicySDKAnswers(resource.Spec.Answers)
	rules, err := steeringPolicySDKRules(resource.Spec.Rules)
	if err != nil {
		return dnssdk.CreateSteeringPolicyDetails{}, err
	}

	return dnssdk.CreateSteeringPolicyDetails{
		CompartmentId:        stringPointer(resource.Spec.CompartmentId),
		DisplayName:          stringPointer(resource.Spec.DisplayName),
		Template:             dnssdk.CreateSteeringPolicyDetailsTemplateEnum(strings.ToUpper(strings.TrimSpace(resource.Spec.Template))),
		Ttl:                  intPointerNonZero(resource.Spec.Ttl),
		HealthCheckMonitorId: stringPointer(resource.Spec.HealthCheckMonitorId),
		FreeformTags:         cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:          steeringPolicyDefinedTags(resource.Spec.DefinedTags),
		Answers:              answers,
		Rules:                rules,
	}, nil
}

func buildSteeringPolicyUpdateBody(
	resource *dnsv1beta1.SteeringPolicy,
	currentResponse any,
) (dnssdk.UpdateSteeringPolicyDetails, bool, error) {
	if resource == nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, false, fmt.Errorf("SteeringPolicy resource is nil")
	}

	mask := steeringPolicyManagedUpdateMask(resource)
	desired, err := steeringPolicyDesiredUpdateDetails(resource)
	if err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, false, err
	}
	desired = applySteeringPolicyUpdateMask(desired, mask)

	currentSource, err := steeringPolicyUpdateSource(resource, currentResponse)
	if err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, false, err
	}
	current, err := steeringPolicyUpdateDetailsFromValue(currentSource)
	if err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, false, err
	}
	current = applySteeringPolicyUpdateMask(current, mask)

	updateNeeded, err := steeringPolicyUpdateNeeded(desired, current)
	if err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, false, err
	}
	if !updateNeeded {
		return dnssdk.UpdateSteeringPolicyDetails{}, false, nil
	}
	return desired, true, nil
}

func steeringPolicyDesiredUpdateDetails(resource *dnsv1beta1.SteeringPolicy) (dnssdk.UpdateSteeringPolicyDetails, error) {
	rules, err := steeringPolicySDKRules(resource.Spec.Rules)
	if err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, err
	}
	return dnssdk.UpdateSteeringPolicyDetails{
		DisplayName:          stringPointer(resource.Spec.DisplayName),
		Ttl:                  intPointerNonZero(resource.Spec.Ttl),
		HealthCheckMonitorId: stringPointer(resource.Spec.HealthCheckMonitorId),
		Template:             dnssdk.UpdateSteeringPolicyDetailsTemplateEnum(strings.ToUpper(strings.TrimSpace(resource.Spec.Template))),
		FreeformTags:         cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:          steeringPolicyDefinedTags(resource.Spec.DefinedTags),
		Answers:              steeringPolicySDKAnswers(resource.Spec.Answers),
		Rules:                rules,
	}, nil
}

func steeringPolicyManagedUpdateMask(resource *dnsv1beta1.SteeringPolicy) steeringPolicyUpdateMask {
	return steeringPolicyUpdateMask{
		displayName:          strings.TrimSpace(resource.Spec.DisplayName) != "",
		ttl:                  resource.Spec.Ttl != 0,
		healthCheckMonitorID: strings.TrimSpace(resource.Spec.HealthCheckMonitorId) != "",
		template:             strings.TrimSpace(resource.Spec.Template) != "",
		freeformTags:         resource.Spec.FreeformTags != nil,
		definedTags:          resource.Spec.DefinedTags != nil,
		answers:              resource.Spec.Answers != nil,
		rules:                resource.Spec.Rules != nil,
	}
}

func applySteeringPolicyUpdateMask(details dnssdk.UpdateSteeringPolicyDetails, mask steeringPolicyUpdateMask) dnssdk.UpdateSteeringPolicyDetails {
	if !mask.displayName {
		details.DisplayName = nil
	}
	if !mask.ttl {
		details.Ttl = nil
	}
	if !mask.healthCheckMonitorID {
		details.HealthCheckMonitorId = nil
	}
	if !mask.template {
		details.Template = ""
	}
	if !mask.freeformTags {
		details.FreeformTags = nil
	}
	if !mask.definedTags {
		details.DefinedTags = nil
	}
	if !mask.answers {
		details.Answers = nil
	}
	if !mask.rules {
		details.Rules = nil
	}
	return details
}

func steeringPolicyUpdateSource(resource *dnsv1beta1.SteeringPolicy, currentResponse any) (any, error) {
	if currentResponse == nil {
		return steeringPolicyStatusUpdateSource(resource)
	}
	if source, ok, err := steeringPolicyFromReadResponse(currentResponse); err != nil {
		return nil, err
	} else if ok {
		return source, nil
	}
	if source, ok, err := steeringPolicyFromWriteResponse(currentResponse); err != nil {
		return nil, err
	} else if ok {
		return source, nil
	}
	return steeringPolicyFromSummary(currentResponse)
}

func steeringPolicyStatusUpdateSource(resource *dnsv1beta1.SteeringPolicy) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("SteeringPolicy resource is nil")
	}
	return resource.Status, nil
}

func steeringPolicyFromReadResponse(currentResponse any) (any, bool, error) {
	switch current := currentResponse.(type) {
	case dnssdk.SteeringPolicy:
		return current, true, nil
	case *dnssdk.SteeringPolicy:
		if current == nil {
			return nil, true, fmt.Errorf("current SteeringPolicy response is nil")
		}
		return *current, true, nil
	case dnssdk.GetSteeringPolicyResponse:
		return current.SteeringPolicy, true, nil
	case *dnssdk.GetSteeringPolicyResponse:
		if current == nil {
			return nil, true, fmt.Errorf("current SteeringPolicy get response is nil")
		}
		return current.SteeringPolicy, true, nil
	default:
		return nil, false, nil
	}
}

func steeringPolicyFromWriteResponse(currentResponse any) (any, bool, error) {
	switch current := currentResponse.(type) {
	case dnssdk.CreateSteeringPolicyResponse:
		return current.SteeringPolicy, true, nil
	case *dnssdk.CreateSteeringPolicyResponse:
		if current == nil {
			return nil, true, fmt.Errorf("current SteeringPolicy create response is nil")
		}
		return current.SteeringPolicy, true, nil
	case dnssdk.UpdateSteeringPolicyResponse:
		return current.SteeringPolicy, true, nil
	case *dnssdk.UpdateSteeringPolicyResponse:
		if current == nil {
			return nil, true, fmt.Errorf("current SteeringPolicy update response is nil")
		}
		return current.SteeringPolicy, true, nil
	default:
		return nil, false, nil
	}
}

func steeringPolicyFromSummary(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case dnssdk.SteeringPolicySummary:
		return current, nil
	case *dnssdk.SteeringPolicySummary:
		if current == nil {
			return nil, fmt.Errorf("current SteeringPolicy summary is nil")
		}
		return *current, nil
	default:
		return currentResponse, nil
	}
}

func steeringPolicyUpdateDetailsFromValue(value any) (dnssdk.UpdateSteeringPolicyDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, fmt.Errorf("marshal SteeringPolicy update details source: %w", err)
	}
	var details dnssdk.UpdateSteeringPolicyDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return dnssdk.UpdateSteeringPolicyDetails{}, fmt.Errorf("decode SteeringPolicy update details: %w", err)
	}
	return details, nil
}

func steeringPolicyUpdateNeeded(desired dnssdk.UpdateSteeringPolicyDetails, current dnssdk.UpdateSteeringPolicyDetails) (bool, error) {
	desired = steeringPolicyComparableUpdateDetails(desired)
	current = steeringPolicyComparableUpdateDetails(current)

	desiredPayload, err := json.Marshal(desired)
	if err != nil {
		return false, fmt.Errorf("marshal desired SteeringPolicy update details: %w", err)
	}
	currentPayload, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal current SteeringPolicy update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func steeringPolicyComparableUpdateDetails(details dnssdk.UpdateSteeringPolicyDetails) dnssdk.UpdateSteeringPolicyDetails {
	details.Answers = steeringPolicyComparableAnswers(details.Answers)
	return details
}

func steeringPolicyComparableAnswers(answers []dnssdk.SteeringPolicyAnswer) []dnssdk.SteeringPolicyAnswer {
	if answers == nil {
		return nil
	}
	comparable := append([]dnssdk.SteeringPolicyAnswer(nil), answers...)
	for index := range comparable {
		rtype := strings.ToUpper(strings.TrimSpace(stringPointerValue(comparable[index].Rtype)))
		if comparable[index].Rtype != nil {
			comparable[index].Rtype = stringPointer(rtype)
		}
		if comparable[index].Rdata != nil {
			rdata := steeringPolicyComparableRdata(rtype, stringPointerValue(comparable[index].Rdata))
			comparable[index].Rdata = stringPointer(rdata)
		}
	}
	return comparable
}

func steeringPolicyComparableRdata(rtype string, rdata string) string {
	normalized := strings.TrimSpace(rdata)
	switch strings.ToUpper(strings.TrimSpace(rtype)) {
	case "A", "AAAA":
		if addr, err := netip.ParseAddr(normalized); err == nil {
			return addr.String()
		}
	case "CNAME":
		return strings.ToLower(strings.TrimRight(normalized, "."))
	}
	return normalized
}

func steeringPolicySDKAnswers(answers []dnsv1beta1.SteeringPolicyAnswer) []dnssdk.SteeringPolicyAnswer {
	if answers == nil {
		return nil
	}
	converted := make([]dnssdk.SteeringPolicyAnswer, 0, len(answers))
	for _, answer := range answers {
		converted = append(converted, dnssdk.SteeringPolicyAnswer{
			Name:       stringPointer(answer.Name),
			Rtype:      stringPointer(answer.Rtype),
			Rdata:      stringPointer(answer.Rdata),
			Pool:       stringPointer(answer.Pool),
			IsDisabled: boolPointer(answer.IsDisabled),
		})
	}
	return converted
}

func steeringPolicySDKRules(rules []dnsv1beta1.SteeringPolicyRule) ([]dnssdk.SteeringPolicyRule, error) {
	if rules == nil {
		return nil, nil
	}
	converted := make([]dnssdk.SteeringPolicyRule, 0, len(rules))
	for index, rule := range rules {
		convertedRule, err := steeringPolicySDKRule(rule)
		if err != nil {
			return nil, fmt.Errorf("convert SteeringPolicy rule %d: %w", index, err)
		}
		converted = append(converted, convertedRule)
	}
	return converted, nil
}

func steeringPolicySDKRule(rule dnsv1beta1.SteeringPolicyRule) (dnssdk.SteeringPolicyRule, error) {
	if strings.TrimSpace(rule.JsonData) != "" {
		return steeringPolicySDKRuleFromJSONData(rule)
	}

	switch strings.ToUpper(strings.TrimSpace(rule.RuleType)) {
	case string(dnssdk.SteeringPolicyRuleRuleTypeFilter):
		return dnssdk.SteeringPolicyFilterRule{
			Description:       stringPointer(rule.Description),
			Cases:             steeringPolicyFilterRuleCases(rule.Cases),
			DefaultAnswerData: steeringPolicyFilterAnswerData(rule.DefaultAnswerData),
		}, nil
	case string(dnssdk.SteeringPolicyRuleRuleTypeHealth):
		return dnssdk.SteeringPolicyHealthRule{
			Description: stringPointer(rule.Description),
			Cases:       steeringPolicyHealthRuleCases(rule.Cases),
		}, nil
	case string(dnssdk.SteeringPolicyRuleRuleTypeLimit):
		if len(rule.Cases) > 0 {
			return nil, fmt.Errorf("LIMIT rule cases require jsonData because the SteeringPolicy CRD does not expose case count")
		}
		return dnssdk.SteeringPolicyLimitRule{
			Description:  stringPointer(rule.Description),
			DefaultCount: intPointerNonZero(rule.DefaultCount),
		}, nil
	case string(dnssdk.SteeringPolicyRuleRuleTypePriority), string(dnssdk.SteeringPolicyRuleRuleTypeWeighted):
		return nil, fmt.Errorf("%s rule requires jsonData because the SteeringPolicy CRD does not expose answer value", strings.ToUpper(strings.TrimSpace(rule.RuleType)))
	case "":
		return nil, fmt.Errorf("ruleType is required")
	default:
		return nil, fmt.Errorf("unsupported ruleType %q", rule.RuleType)
	}
}

func steeringPolicySDKRuleFromJSONData(rule dnsv1beta1.SteeringPolicyRule) (dnssdk.SteeringPolicyRule, error) {
	raw := []byte(strings.TrimSpace(rule.JsonData))
	var discriminator struct {
		RuleType string `json:"ruleType"`
	}
	if err := json.Unmarshal(raw, &discriminator); err != nil {
		return nil, fmt.Errorf("decode jsonData discriminator: %w", err)
	}
	ruleType := strings.ToUpper(firstNonEmptyTrim(discriminator.RuleType, rule.RuleType))
	if ruleType == "" {
		return nil, fmt.Errorf("jsonData ruleType is required")
	}
	decoder, ok := steeringPolicyJSONRuleDecoders[ruleType]
	if !ok {
		return nil, fmt.Errorf("unsupported jsonData ruleType %q", ruleType)
	}
	return decoder(raw)
}

type steeringPolicyJSONRuleDecoder func([]byte) (dnssdk.SteeringPolicyRule, error)

var steeringPolicyJSONRuleDecoders = map[string]steeringPolicyJSONRuleDecoder{
	string(dnssdk.SteeringPolicyRuleRuleTypeFilter):   decodeSteeringPolicyFilterJSONRule,
	string(dnssdk.SteeringPolicyRuleRuleTypeHealth):   decodeSteeringPolicyHealthJSONRule,
	string(dnssdk.SteeringPolicyRuleRuleTypeLimit):    decodeSteeringPolicyLimitJSONRule,
	string(dnssdk.SteeringPolicyRuleRuleTypePriority): decodeSteeringPolicyPriorityJSONRule,
	string(dnssdk.SteeringPolicyRuleRuleTypeWeighted): decodeSteeringPolicyWeightedJSONRule,
}

func decodeSteeringPolicyFilterJSONRule(raw []byte) (dnssdk.SteeringPolicyRule, error) {
	var details dnssdk.SteeringPolicyFilterRule
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("decode FILTER jsonData: %w", err)
	}
	return details, nil
}

func decodeSteeringPolicyHealthJSONRule(raw []byte) (dnssdk.SteeringPolicyRule, error) {
	var details dnssdk.SteeringPolicyHealthRule
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("decode HEALTH jsonData: %w", err)
	}
	return details, nil
}

func decodeSteeringPolicyLimitJSONRule(raw []byte) (dnssdk.SteeringPolicyRule, error) {
	var details dnssdk.SteeringPolicyLimitRule
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("decode LIMIT jsonData: %w", err)
	}
	return details, nil
}

func decodeSteeringPolicyPriorityJSONRule(raw []byte) (dnssdk.SteeringPolicyRule, error) {
	var details dnssdk.SteeringPolicyPriorityRule
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("decode PRIORITY jsonData: %w", err)
	}
	return details, nil
}

func decodeSteeringPolicyWeightedJSONRule(raw []byte) (dnssdk.SteeringPolicyRule, error) {
	var details dnssdk.SteeringPolicyWeightedRule
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("decode WEIGHTED jsonData: %w", err)
	}
	return details, nil
}

func steeringPolicyFilterRuleCases(cases []dnsv1beta1.SteeringPolicyRuleCase) []dnssdk.SteeringPolicyFilterRuleCase {
	if cases == nil {
		return nil
	}
	converted := make([]dnssdk.SteeringPolicyFilterRuleCase, 0, len(cases))
	for _, item := range cases {
		converted = append(converted, dnssdk.SteeringPolicyFilterRuleCase{
			CaseCondition: stringPointer(item.CaseCondition),
			AnswerData:    steeringPolicyFilterCaseAnswerData(item.AnswerData),
		})
	}
	return converted
}

func steeringPolicyHealthRuleCases(cases []dnsv1beta1.SteeringPolicyRuleCase) []dnssdk.SteeringPolicyHealthRuleCase {
	if cases == nil {
		return nil
	}
	converted := make([]dnssdk.SteeringPolicyHealthRuleCase, 0, len(cases))
	for _, item := range cases {
		converted = append(converted, dnssdk.SteeringPolicyHealthRuleCase{
			CaseCondition: stringPointer(item.CaseCondition),
		})
	}
	return converted
}

func steeringPolicyFilterCaseAnswerData(items []dnsv1beta1.SteeringPolicyRuleCaseAnswerData) []dnssdk.SteeringPolicyFilterAnswerData {
	if items == nil {
		return nil
	}
	converted := make([]dnssdk.SteeringPolicyFilterAnswerData, 0, len(items))
	for _, item := range items {
		converted = append(converted, dnssdk.SteeringPolicyFilterAnswerData{
			AnswerCondition: stringPointer(item.AnswerCondition),
			ShouldKeep:      boolPointer(item.ShouldKeep),
		})
	}
	return converted
}

func steeringPolicyFilterAnswerData(items []dnsv1beta1.SteeringPolicyRuleDefaultAnswerData) []dnssdk.SteeringPolicyFilterAnswerData {
	if items == nil {
		return nil
	}
	converted := make([]dnssdk.SteeringPolicyFilterAnswerData, 0, len(items))
	for _, item := range items {
		converted = append(converted, dnssdk.SteeringPolicyFilterAnswerData{
			AnswerCondition: stringPointer(item.AnswerCondition),
			ShouldKeep:      boolPointer(item.ShouldKeep),
		})
	}
	return converted
}

func handleSteeringPolicyDeleteError(resource *dnsv1beta1.SteeringPolicy, err error) error {
	if err == nil {
		return nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return fmt.Errorf("SteeringPolicy delete returned authorization-shaped not found; refusing to confirm deletion: %s", err)
	}
	return err
}

func steeringPolicyDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		copied := make(map[string]interface{}, len(values))
		for key, value := range values {
			copied[key] = value
		}
		converted[namespace] = copied
	}
	return converted
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func intPointerNonZero(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
