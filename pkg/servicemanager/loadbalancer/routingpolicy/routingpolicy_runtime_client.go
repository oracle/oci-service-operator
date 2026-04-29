/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package routingpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const routingPolicyLoadBalancerIDAnnotation = "loadbalancer.oracle.com/load-balancer-id"

var routingPolicyWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		string(loadbalancersdk.WorkRequestLifecycleStateInProgress),
	},
	SucceededStatusTokens: []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded)},
	FailedStatusTokens:    []string{string(loadbalancersdk.WorkRequestLifecycleStateFailed)},
	CreateActionTokens: []string{
		"CreateRoutingPolicy",
		"CREATE_ROUTING_POLICY",
	},
	UpdateActionTokens: []string{
		"UpdateRoutingPolicy",
		"UPDATE_ROUTING_POLICY",
	},
	DeleteActionTokens: []string{
		"DeleteRoutingPolicy",
		"DELETE_ROUTING_POLICY",
	},
}

type routingPolicyRuntimeOCIClient interface {
	CreateRoutingPolicy(context.Context, loadbalancersdk.CreateRoutingPolicyRequest) (loadbalancersdk.CreateRoutingPolicyResponse, error)
	GetRoutingPolicy(context.Context, loadbalancersdk.GetRoutingPolicyRequest) (loadbalancersdk.GetRoutingPolicyResponse, error)
	ListRoutingPolicies(context.Context, loadbalancersdk.ListRoutingPoliciesRequest) (loadbalancersdk.ListRoutingPoliciesResponse, error)
	UpdateRoutingPolicy(context.Context, loadbalancersdk.UpdateRoutingPolicyRequest) (loadbalancersdk.UpdateRoutingPolicyResponse, error)
	DeleteRoutingPolicy(context.Context, loadbalancersdk.DeleteRoutingPolicyRequest) (loadbalancersdk.DeleteRoutingPolicyResponse, error)
	GetWorkRequest(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

type routingPolicyWorkRequestClient interface {
	GetWorkRequest(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

type routingPolicyIdentity struct {
	loadBalancerID    string
	routingPolicyName string
}

type routingPolicyGeneratedWorkRequest struct {
	Id             string
	Status         string
	OperationType  string
	LoadBalancerId string
	Message        string
}

type routingPolicyWorkRequestConvergingClient struct {
	delegate RoutingPolicyServiceClient
}

func init() {
	registerRoutingPolicyRuntimeHooksMutator(func(manager *RoutingPolicyServiceManager, hooks *RoutingPolicyRuntimeHooks) {
		workRequestClient, initErr := newRoutingPolicyWorkRequestClient(manager)
		applyRoutingPolicyRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newRoutingPolicyWorkRequestClient(manager *RoutingPolicyServiceManager) (routingPolicyWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("RoutingPolicy service manager is nil")
	}
	client, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyRoutingPolicyRuntimeHooks(
	hooks *RoutingPolicyRuntimeHooks,
	workRequestClient routingPolicyWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newRoutingPolicyRuntimeSemantics()
	hooks.Async.Adapter = routingPolicyWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getRoutingPolicyWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveRoutingPolicyGeneratedWorkRequestAction
	hooks.Async.RecoverResourceID = recoverRoutingPolicyLoadBalancerIDFromGeneratedWorkRequest
	hooks.Async.Message = routingPolicyGeneratedWorkRequestMessage
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.RoutingPolicy,
		_ string,
	) (any, error) {
		return buildRoutingPolicyCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.RoutingPolicy,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildRoutingPolicyUpdateBody(resource, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.RoutingPolicy]{
		Resolve: func(resource *loadbalancerv1beta1.RoutingPolicy) (any, error) {
			return resolveRoutingPolicyIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.RoutingPolicy, identity any) {
			recordRoutingPolicyPathIdentity(resource, identity.(routingPolicyIdentity))
		},
		RecordTracked: func(resource *loadbalancerv1beta1.RoutingPolicy, identity any, _ string) {
			recordRoutingPolicyTrackedIdentity(resource, identity.(routingPolicyIdentity))
		},
		LookupExisting: func(context.Context, *loadbalancerv1beta1.RoutingPolicy, any) (any, error) {
			return nil, nil
		},
	}
	hooks.Create.Fields = routingPolicyCreateFields()
	hooks.Get.Fields = routingPolicyGetFields()
	hooks.List.Fields = routingPolicyListFields()
	hooks.Update.Fields = routingPolicyUpdateFields()
	hooks.Delete.Fields = routingPolicyDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate RoutingPolicyServiceClient) RoutingPolicyServiceClient {
		return routingPolicyWorkRequestConvergingClient{delegate: delegate}
	})
}

func getRoutingPolicyWorkRequest(
	ctx context.Context,
	client routingPolicyWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize RoutingPolicy OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("RoutingPolicy OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, loadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return routingPolicyGeneratedWorkRequestFromSDK(response.WorkRequest), nil
}

func routingPolicyGeneratedWorkRequestFromSDK(workRequest loadbalancersdk.WorkRequest) routingPolicyGeneratedWorkRequest {
	return routingPolicyGeneratedWorkRequest{
		Id:             stringPointerValue(workRequest.Id),
		Status:         string(workRequest.LifecycleState),
		OperationType:  stringPointerValue(workRequest.Type),
		LoadBalancerId: stringPointerValue(workRequest.LoadBalancerId),
		Message:        stringPointerValue(workRequest.Message),
	}
}

func resolveRoutingPolicyGeneratedWorkRequestAction(workRequest any) (string, error) {
	routingPolicyWorkRequest, err := routingPolicyGeneratedWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return routingPolicyWorkRequest.OperationType, nil
}

func recoverRoutingPolicyLoadBalancerIDFromGeneratedWorkRequest(
	_ *loadbalancerv1beta1.RoutingPolicy,
	workRequest any,
	_ shared.OSOKAsyncPhase,
) (string, error) {
	routingPolicyWorkRequest, err := routingPolicyGeneratedWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return routingPolicyWorkRequest.LoadBalancerId, nil
}

func routingPolicyGeneratedWorkRequestMessage(
	phase shared.OSOKAsyncPhase,
	workRequest any,
) string {
	routingPolicyWorkRequest, err := routingPolicyGeneratedWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	if routingPolicyWorkRequest.Id == "" || routingPolicyWorkRequest.Status == "" {
		return ""
	}
	message := fmt.Sprintf("RoutingPolicy %s work request %s is %s", phase, routingPolicyWorkRequest.Id, routingPolicyWorkRequest.Status)
	if routingPolicyWorkRequest.Message != "" {
		message = message + ": " + routingPolicyWorkRequest.Message
	}
	return message
}

func routingPolicyGeneratedWorkRequestFromAny(workRequest any) (routingPolicyGeneratedWorkRequest, error) {
	switch current := workRequest.(type) {
	case routingPolicyGeneratedWorkRequest:
		return current, nil
	case *routingPolicyGeneratedWorkRequest:
		if current == nil {
			return routingPolicyGeneratedWorkRequest{}, fmt.Errorf("RoutingPolicy work request is nil")
		}
		return *current, nil
	default:
		return routingPolicyGeneratedWorkRequest{}, fmt.Errorf("expected RoutingPolicy work request, got %T", workRequest)
	}
}

func (c routingPolicyWorkRequestConvergingClient) CreateOrUpdate(
	ctx context.Context,
	resource *loadbalancerv1beta1.RoutingPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !routingPolicyNeedsPostWorkRequestObserve(resource, response) {
		return response, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c routingPolicyWorkRequestConvergingClient) Delete(
	ctx context.Context,
	resource *loadbalancerv1beta1.RoutingPolicy,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func routingPolicyNeedsPostWorkRequestObserve(
	resource *loadbalancerv1beta1.RoutingPolicy,
	response servicemanager.OSOKResponse,
) bool {
	if resource == nil || !response.IsSuccessful || !response.ShouldRequeue {
		return false
	}
	status := resource.Status.OsokStatus
	if status.Async.Current != nil {
		return false
	}
	switch shared.OSOKConditionType(status.Reason) {
	case shared.Provisioning, shared.Updating:
		return true
	default:
		return false
	}
}

func newRoutingPolicyRuntimeHooksWithOCIClient(client routingPolicyRuntimeOCIClient) RoutingPolicyRuntimeHooks {
	return RoutingPolicyRuntimeHooks{
		Semantics: newRoutingPolicyRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.RoutingPolicy]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[loadbalancersdk.CreateRoutingPolicyRequest, loadbalancersdk.CreateRoutingPolicyResponse]{
			Fields: routingPolicyCreateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.CreateRoutingPolicyRequest) (loadbalancersdk.CreateRoutingPolicyResponse, error) {
				return client.CreateRoutingPolicy(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loadbalancersdk.GetRoutingPolicyRequest, loadbalancersdk.GetRoutingPolicyResponse]{
			Fields: routingPolicyGetFields(),
			Call: func(ctx context.Context, request loadbalancersdk.GetRoutingPolicyRequest) (loadbalancersdk.GetRoutingPolicyResponse, error) {
				return client.GetRoutingPolicy(ctx, request)
			},
		},
		List: runtimeOperationHooks[loadbalancersdk.ListRoutingPoliciesRequest, loadbalancersdk.ListRoutingPoliciesResponse]{
			Fields: routingPolicyListFields(),
			Call: func(ctx context.Context, request loadbalancersdk.ListRoutingPoliciesRequest) (loadbalancersdk.ListRoutingPoliciesResponse, error) {
				return client.ListRoutingPolicies(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdateRoutingPolicyRequest, loadbalancersdk.UpdateRoutingPolicyResponse]{
			Fields: routingPolicyUpdateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.UpdateRoutingPolicyRequest) (loadbalancersdk.UpdateRoutingPolicyResponse, error) {
				return client.UpdateRoutingPolicy(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeleteRoutingPolicyRequest, loadbalancersdk.DeleteRoutingPolicyResponse]{
			Fields: routingPolicyDeleteFields(),
			Call: func(ctx context.Context, request loadbalancersdk.DeleteRoutingPolicyRequest) (loadbalancersdk.DeleteRoutingPolicyResponse, error) {
				return client.DeleteRoutingPolicy(ctx, request)
			},
		},
		WrapGeneratedClient: []func(RoutingPolicyServiceClient) RoutingPolicyServiceClient{},
	}
}

func newRoutingPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "routingpolicy",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
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
			Mutable:       []string{"conditionLanguageVersion", "rules"},
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

func routingPolicyCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		routingPolicyLoadBalancerIDField(),
		{
			FieldName:    "CreateRoutingPolicyDetails",
			RequestName:  "CreateRoutingPolicyDetails",
			Contribution: "body",
		},
	}
}

func routingPolicyGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		routingPolicyLoadBalancerIDField(),
		routingPolicyNameField(),
	}
}

func routingPolicyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		routingPolicyLoadBalancerIDField(),
	}
}

func routingPolicyUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		routingPolicyLoadBalancerIDField(),
		routingPolicyNameField(),
		{
			FieldName:    "UpdateRoutingPolicyDetails",
			RequestName:  "UpdateRoutingPolicyDetails",
			Contribution: "body",
		},
	}
}

func routingPolicyDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		routingPolicyLoadBalancerIDField(),
		routingPolicyNameField(),
	}
}

func routingPolicyLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "LoadBalancerId",
		RequestName:      "loadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.status.ocid"},
	}
}

func routingPolicyNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "RoutingPolicyName",
		RequestName:  "routingPolicyName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func buildRoutingPolicyCreateBody(resource *loadbalancerv1beta1.RoutingPolicy) (loadbalancersdk.CreateRoutingPolicyDetails, error) {
	if resource == nil {
		return loadbalancersdk.CreateRoutingPolicyDetails{}, fmt.Errorf("routingpolicy resource is nil")
	}

	rules, err := routingPolicySDKRules(resource.Spec.Rules)
	if err != nil {
		return loadbalancersdk.CreateRoutingPolicyDetails{}, err
	}
	return loadbalancersdk.CreateRoutingPolicyDetails{
		Name:                     stringPointer(firstNonEmptyTrim(resource.Spec.Name, resource.Name)),
		ConditionLanguageVersion: loadbalancersdk.CreateRoutingPolicyDetailsConditionLanguageVersionEnum(resource.Spec.ConditionLanguageVersion),
		Rules:                    rules,
	}, nil
}

func buildRoutingPolicyUpdateBody(
	resource *loadbalancerv1beta1.RoutingPolicy,
	currentResponse any,
) (loadbalancersdk.UpdateRoutingPolicyDetails, bool, error) {
	if resource == nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, false, fmt.Errorf("routingpolicy resource is nil")
	}

	rules, err := routingPolicySDKRules(resource.Spec.Rules)
	if err != nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, false, err
	}
	desired := loadbalancersdk.UpdateRoutingPolicyDetails{
		Rules:                    rules,
		ConditionLanguageVersion: loadbalancersdk.UpdateRoutingPolicyDetailsConditionLanguageVersionEnum(resource.Spec.ConditionLanguageVersion),
	}

	currentSource, err := routingPolicyUpdateSource(resource, currentResponse)
	if err != nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, false, err
	}
	current, err := routingPolicyUpdateDetailsFromValue(currentSource)
	if err != nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, false, err
	}

	updateNeeded, err := routingPolicyUpdateNeeded(desired, current)
	if err != nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, false, err
	}
	if !updateNeeded {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, false, nil
	}
	return desired, true, nil
}

func routingPolicyUpdateSource(resource *loadbalancerv1beta1.RoutingPolicy, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("routingpolicy resource is nil")
		}
		return resource.Status, nil
	case loadbalancersdk.RoutingPolicy:
		return current, nil
	case *loadbalancersdk.RoutingPolicy:
		if current == nil {
			return nil, fmt.Errorf("current RoutingPolicy response is nil")
		}
		return *current, nil
	case loadbalancersdk.GetRoutingPolicyResponse:
		return current.RoutingPolicy, nil
	case *loadbalancersdk.GetRoutingPolicyResponse:
		if current == nil {
			return nil, fmt.Errorf("current RoutingPolicy response is nil")
		}
		return current.RoutingPolicy, nil
	default:
		return currentResponse, nil
	}
}

func routingPolicyUpdateDetailsFromValue(value any) (loadbalancersdk.UpdateRoutingPolicyDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, fmt.Errorf("marshal RoutingPolicy update details source: %w", err)
	}

	var details loadbalancersdk.UpdateRoutingPolicyDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return loadbalancersdk.UpdateRoutingPolicyDetails{}, fmt.Errorf("decode RoutingPolicy update details: %w", err)
	}
	return details, nil
}

func routingPolicyUpdateNeeded(desired loadbalancersdk.UpdateRoutingPolicyDetails, current loadbalancersdk.UpdateRoutingPolicyDetails) (bool, error) {
	desiredPayload, err := json.Marshal(desired)
	if err != nil {
		return false, fmt.Errorf("marshal desired RoutingPolicy update details: %w", err)
	}
	currentPayload, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal current RoutingPolicy update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func routingPolicySDKRules(rules []loadbalancerv1beta1.RoutingPolicyRule) ([]loadbalancersdk.RoutingRule, error) {
	if rules == nil {
		return nil, nil
	}

	converted := make([]loadbalancersdk.RoutingRule, 0, len(rules))
	for index, rule := range rules {
		actions, err := routingPolicySDKActions(rule.Actions)
		if err != nil {
			return nil, fmt.Errorf("convert routing policy rule %d actions: %w", index, err)
		}
		converted = append(converted, loadbalancersdk.RoutingRule{
			Name:      stringPointer(rule.Name),
			Condition: stringPointer(rule.Condition),
			Actions:   actions,
		})
	}
	return converted, nil
}

func routingPolicySDKActions(actions []loadbalancerv1beta1.RoutingPolicyRuleAction) ([]loadbalancersdk.Action, error) {
	if actions == nil {
		return nil, nil
	}

	converted := make([]loadbalancersdk.Action, 0, len(actions))
	for index, action := range actions {
		convertedAction, err := routingPolicySDKAction(action)
		if err != nil {
			return nil, fmt.Errorf("convert routing policy action %d: %w", index, err)
		}
		converted = append(converted, convertedAction)
	}
	return converted, nil
}

func routingPolicySDKAction(action loadbalancerv1beta1.RoutingPolicyRuleAction) (loadbalancersdk.Action, error) {
	if raw := strings.TrimSpace(action.JsonData); raw != "" {
		var discriminator struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
			return nil, fmt.Errorf("decode JsonData discriminator: %w", err)
		}
		name := strings.ToUpper(strings.TrimSpace(discriminator.Name))
		if name == "" {
			name = string(loadbalancersdk.ActionNameForwardToBackendset)
		}
		if name != string(loadbalancersdk.ActionNameForwardToBackendset) {
			return nil, fmt.Errorf("unsupported action name %q", discriminator.Name)
		}
		var details loadbalancersdk.ForwardToBackendSet
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode ForwardToBackendSet JsonData: %w", err)
		}
		return details, nil
	}

	name := strings.ToUpper(strings.TrimSpace(action.Name))
	if name == "" && strings.TrimSpace(action.BackendSetName) != "" {
		name = string(loadbalancersdk.ActionNameForwardToBackendset)
	}
	switch name {
	case string(loadbalancersdk.ActionNameForwardToBackendset):
		return loadbalancersdk.ForwardToBackendSet{
			BackendSetName: stringPointer(action.BackendSetName),
		}, nil
	case "":
		return nil, fmt.Errorf("action name is empty")
	default:
		return nil, fmt.Errorf("unsupported action name %q", action.Name)
	}
}

func resolveRoutingPolicyIdentity(resource *loadbalancerv1beta1.RoutingPolicy) (routingPolicyIdentity, error) {
	if resource == nil {
		return routingPolicyIdentity{}, fmt.Errorf("resolve RoutingPolicy identity: resource is nil")
	}

	statusLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationLoadBalancerID := strings.TrimSpace(resource.Annotations[routingPolicyLoadBalancerIDAnnotation])
	if statusLoadBalancerID != "" && annotationLoadBalancerID != "" && statusLoadBalancerID != annotationLoadBalancerID {
		return routingPolicyIdentity{}, fmt.Errorf(
			"resolve RoutingPolicy identity: %s changed from recorded loadBalancerId %q to %q",
			routingPolicyLoadBalancerIDAnnotation,
			statusLoadBalancerID,
			annotationLoadBalancerID,
		)
	}

	identity := routingPolicyIdentity{
		loadBalancerID:    firstNonEmptyTrim(statusLoadBalancerID, annotationLoadBalancerID),
		routingPolicyName: firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name, resource.Name),
	}
	if identity.loadBalancerID == "" {
		return routingPolicyIdentity{}, fmt.Errorf("resolve RoutingPolicy identity: %s annotation is required", routingPolicyLoadBalancerIDAnnotation)
	}
	if identity.routingPolicyName == "" {
		return routingPolicyIdentity{}, fmt.Errorf("resolve RoutingPolicy identity: routing policy name is empty")
	}
	return identity, nil
}

func recordRoutingPolicyPathIdentity(resource *loadbalancerv1beta1.RoutingPolicy, identity routingPolicyIdentity) {
	if resource == nil {
		return
	}
	resource.Status.Name = identity.routingPolicyName
	// RoutingPolicy has no child OCID in the Load Balancer API, so the runtime records
	// the parent loadBalancerId as the stable path identity used for Get/Update/Delete.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.loadBalancerID)
}

func recordRoutingPolicyTrackedIdentity(resource *loadbalancerv1beta1.RoutingPolicy, identity routingPolicyIdentity) {
	recordRoutingPolicyPathIdentity(resource, identity)
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

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
