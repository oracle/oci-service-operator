/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package backendset

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const backendSetNetworkLoadBalancerIDAnnotation = "networkloadbalancer.oracle.com/network-load-balancer-id"

var backendSetWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(networkloadbalancersdk.OperationStatusAccepted),
		string(networkloadbalancersdk.OperationStatusInProgress),
		string(networkloadbalancersdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(networkloadbalancersdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(networkloadbalancersdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(networkloadbalancersdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(networkloadbalancersdk.OperationTypeCreateBackendset),
		"CreateBackendSet",
	},
	UpdateActionTokens: []string{
		string(networkloadbalancersdk.OperationTypeUpdateBackendset),
		"UpdateBackendSet",
	},
	DeleteActionTokens: []string{
		string(networkloadbalancersdk.OperationTypeDeleteBackendset),
		"DeleteBackendSet",
	},
}

type backendSetRuntimeOCIClient interface {
	CreateBackendSet(context.Context, networkloadbalancersdk.CreateBackendSetRequest) (networkloadbalancersdk.CreateBackendSetResponse, error)
	GetBackendSet(context.Context, networkloadbalancersdk.GetBackendSetRequest) (networkloadbalancersdk.GetBackendSetResponse, error)
	ListBackendSets(context.Context, networkloadbalancersdk.ListBackendSetsRequest) (networkloadbalancersdk.ListBackendSetsResponse, error)
	UpdateBackendSet(context.Context, networkloadbalancersdk.UpdateBackendSetRequest) (networkloadbalancersdk.UpdateBackendSetResponse, error)
	DeleteBackendSet(context.Context, networkloadbalancersdk.DeleteBackendSetRequest) (networkloadbalancersdk.DeleteBackendSetResponse, error)
	GetWorkRequest(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

type backendSetWorkRequestClient interface {
	GetWorkRequest(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

type backendSetIdentity struct {
	networkLoadBalancerID string
	backendSetName        string
}

type backendSetPendingWorkRequestDeleteClient struct {
	delegate           BackendSetServiceClient
	workRequestClient  backendSetWorkRequestClient
	workRequestInitErr error
	getBackendSet      func(context.Context, networkloadbalancersdk.GetBackendSetRequest) (networkloadbalancersdk.GetBackendSetResponse, error)
	listBackendSets    func(context.Context, networkloadbalancersdk.ListBackendSetsRequest) (networkloadbalancersdk.ListBackendSetsResponse, error)
}

func init() {
	registerBackendSetRuntimeHooksMutator(func(manager *BackendSetServiceManager, hooks *BackendSetRuntimeHooks) {
		workRequestClient, initErr := newBackendSetWorkRequestClient(manager)
		applyBackendSetRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newBackendSetWorkRequestClient(manager *BackendSetServiceManager) (backendSetWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("BackendSet service manager is nil")
	}
	client, err := networkloadbalancersdk.NewNetworkLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyBackendSetRuntimeHooks(
	hooks *BackendSetRuntimeHooks,
	workRequestClient backendSetWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	listCall := hooks.List.Call
	paginatedListCall := func(ctx context.Context, request networkloadbalancersdk.ListBackendSetsRequest) (networkloadbalancersdk.ListBackendSetsResponse, error) {
		return listBackendSetPages(ctx, request, listCall)
	}
	hooks.Semantics = newBackendSetRuntimeSemantics()
	hooks.Async.Adapter = backendSetWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getBackendSetWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveBackendSetWorkRequestAction
	hooks.Async.Message = backendSetWorkRequestMessage
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *networkloadbalancerv1beta1.BackendSet,
		namespace string,
	) (any, error) {
		return buildBackendSetCreateBody(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *networkloadbalancerv1beta1.BackendSet,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildBackendSetUpdateBody(ctx, resource, namespace, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.BackendSet]{
		Resolve: func(resource *networkloadbalancerv1beta1.BackendSet) (any, error) {
			return resolveBackendSetIdentity(resource)
		},
		RecordPath: func(resource *networkloadbalancerv1beta1.BackendSet, identity any) {
			recordBackendSetPathIdentity(resource, identity.(backendSetIdentity))
		},
		RecordTracked: func(resource *networkloadbalancerv1beta1.BackendSet, identity any, _ string) {
			recordBackendSetTrackedIdentity(resource, identity.(backendSetIdentity))
		},
	}
	hooks.DeleteHooks.HandleError = handleBackendSetDeleteError
	hooks.Create.Fields = backendSetCreateFields()
	hooks.Get.Fields = backendSetGetFields()
	hooks.List.Fields = backendSetListFields()
	hooks.List.Call = paginatedListCall
	hooks.Update.Fields = backendSetUpdateFields()
	hooks.Delete.Fields = backendSetDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate BackendSetServiceClient) BackendSetServiceClient {
		return backendSetPendingWorkRequestDeleteClient{
			delegate:           delegate,
			workRequestClient:  workRequestClient,
			workRequestInitErr: initErr,
			getBackendSet:      hooks.Get.Call,
			listBackendSets:    paginatedListCall,
		}
	})
}

func newBackendSetRuntimeHooksWithOCIClient(client backendSetRuntimeOCIClient) BackendSetRuntimeHooks {
	return BackendSetRuntimeHooks{
		Semantics: newBackendSetRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.BackendSet]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[networkloadbalancersdk.CreateBackendSetRequest, networkloadbalancersdk.CreateBackendSetResponse]{
			Fields: backendSetCreateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.CreateBackendSetRequest) (networkloadbalancersdk.CreateBackendSetResponse, error) {
				return client.CreateBackendSet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[networkloadbalancersdk.GetBackendSetRequest, networkloadbalancersdk.GetBackendSetResponse]{
			Fields: backendSetGetFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.GetBackendSetRequest) (networkloadbalancersdk.GetBackendSetResponse, error) {
				return client.GetBackendSet(ctx, request)
			},
		},
		List: runtimeOperationHooks[networkloadbalancersdk.ListBackendSetsRequest, networkloadbalancersdk.ListBackendSetsResponse]{
			Fields: backendSetListFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.ListBackendSetsRequest) (networkloadbalancersdk.ListBackendSetsResponse, error) {
				return client.ListBackendSets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[networkloadbalancersdk.UpdateBackendSetRequest, networkloadbalancersdk.UpdateBackendSetResponse]{
			Fields: backendSetUpdateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.UpdateBackendSetRequest) (networkloadbalancersdk.UpdateBackendSetResponse, error) {
				return client.UpdateBackendSet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[networkloadbalancersdk.DeleteBackendSetRequest, networkloadbalancersdk.DeleteBackendSetResponse]{
			Fields: backendSetDeleteFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.DeleteBackendSetRequest) (networkloadbalancersdk.DeleteBackendSetResponse, error) {
				return client.DeleteBackendSet(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BackendSetServiceClient) BackendSetServiceClient{},
	}
}

func newBackendSetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "networkloadbalancer",
		FormalSlug:    "backendset",
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
			Mutable: []string{
				"policy",
				"healthChecker",
				"isPreserveSource",
				"isFailOpen",
				"isInstantFailoverEnabled",
				"isInstantFailoverTcpResetEnabled",
				"areOperationallyActiveBackendsPreferred",
				"ipVersion",
				"backends",
			},
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

func backendSetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetNetworkLoadBalancerIDField(),
		{
			FieldName:    "CreateBackendSetDetails",
			RequestName:  "CreateBackendSetDetails",
			Contribution: "body",
		},
	}
}

func backendSetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetNetworkLoadBalancerIDField(),
		backendSetNameField(),
	}
}

func backendSetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetNetworkLoadBalancerIDField(),
	}
}

func backendSetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetNetworkLoadBalancerIDField(),
		backendSetNameField(),
		{
			FieldName:    "UpdateBackendSetDetails",
			RequestName:  "UpdateBackendSetDetails",
			Contribution: "body",
		},
	}
}

func backendSetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		backendSetNetworkLoadBalancerIDField(),
		backendSetNameField(),
	}
}

func listBackendSetPages(
	ctx context.Context,
	request networkloadbalancersdk.ListBackendSetsRequest,
	call func(context.Context, networkloadbalancersdk.ListBackendSetsRequest) (networkloadbalancersdk.ListBackendSetsResponse, error),
) (networkloadbalancersdk.ListBackendSetsResponse, error) {
	if call == nil {
		return networkloadbalancersdk.ListBackendSetsResponse{}, fmt.Errorf("BackendSet list operation is not configured")
	}

	seenPages := map[string]struct{}{}
	var merged networkloadbalancersdk.ListBackendSetsResponse
	for {
		pageToken := stringPointerValue(request.Page)
		if _, seen := seenPages[pageToken]; seen {
			return networkloadbalancersdk.ListBackendSetsResponse{}, fmt.Errorf("BackendSet ListBackendSets pagination repeated page token %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return networkloadbalancersdk.ListBackendSetsResponse{}, err
		}
		if len(seenPages) == 1 {
			merged = response
			merged.Items = append([]networkloadbalancersdk.BackendSetSummary(nil), response.Items...)
		} else {
			merged.Items = append(merged.Items, response.Items...)
		}

		nextPage := stringPointerValue(response.OpcNextPage)
		if nextPage == "" {
			merged.OpcNextPage = nil
			return merged, nil
		}
		request.Page = common.String(nextPage)
	}
}

func backendSetNetworkLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "NetworkLoadBalancerId",
		RequestName:      "networkLoadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.status.ocid"},
	}
}

func backendSetNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "BackendSetName",
		RequestName:  "backendSetName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func getBackendSetWorkRequest(
	ctx context.Context,
	client backendSetWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize BackendSet OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("BackendSet OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, networkloadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveBackendSetWorkRequestAction(workRequest any) (string, error) {
	current, err := backendSetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func backendSetWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := backendSetWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	workRequestID := stringPointerValue(current.Id)
	if workRequestID == "" || current.Status == "" {
		return ""
	}
	return fmt.Sprintf("BackendSet %s work request %s is %s", phase, workRequestID, current.Status)
}

func backendSetWorkRequestFromAny(workRequest any) (networkloadbalancersdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case networkloadbalancersdk.WorkRequest:
		return current, nil
	case *networkloadbalancersdk.WorkRequest:
		if current == nil {
			return networkloadbalancersdk.WorkRequest{}, fmt.Errorf("BackendSet work request is nil")
		}
		return *current, nil
	default:
		return networkloadbalancersdk.WorkRequest{}, fmt.Errorf("expected BackendSet work request, got %T", workRequest)
	}
}

func buildBackendSetCreateBody(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.BackendSet,
	namespace string,
) (networkloadbalancersdk.CreateBackendSetDetails, error) {
	if resource == nil {
		return networkloadbalancersdk.CreateBackendSetDetails{}, fmt.Errorf("backendset resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return networkloadbalancersdk.CreateBackendSetDetails{}, err
	}
	resolvedSpec = overlayBackendSetExistingBoolFields(reflect.ValueOf(resource.Spec), resolvedSpec)

	details, err := backendSetCreateDetailsFromValue(resolvedSpec)
	if err != nil {
		return networkloadbalancersdk.CreateBackendSetDetails{}, fmt.Errorf("build desired BackendSet create details: %w", err)
	}
	if details.Name == nil {
		details.Name = stringPointer(firstNonEmptyTrim(resource.Spec.Name, resource.Name))
	}
	stripBackendSetCreateReadOnlyBackendNames(&details)
	return details, nil
}

func buildBackendSetUpdateBody(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.BackendSet,
	namespace string,
	currentResponse any,
) (networkloadbalancersdk.UpdateBackendSetDetails, bool, error) {
	if resource == nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, fmt.Errorf("backendset resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, err
	}
	resolvedSpec = overlayBackendSetExistingBoolFields(reflect.ValueOf(resource.Spec), resolvedSpec)

	desired, err := backendSetUpdateDetailsFromValue(resolvedSpec)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, fmt.Errorf("build desired BackendSet update details: %w", err)
	}
	stripBackendSetUpdateReadOnlyBackendNames(&desired)

	currentSource, err := backendSetUpdateSource(resource, currentResponse)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, err
	}
	current, err := backendSetUpdateDetailsFromValue(currentSource)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, fmt.Errorf("build current BackendSet update details: %w", err)
	}

	updateNeeded, err := backendSetUpdateNeeded(desired, current)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, err
	}
	if !updateNeeded {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, false, nil
	}
	return desired, true, nil
}

func backendSetCreateDetailsFromValue(value any) (networkloadbalancersdk.CreateBackendSetDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return networkloadbalancersdk.CreateBackendSetDetails{}, fmt.Errorf("marshal BackendSet create details source: %w", err)
	}

	var details networkloadbalancersdk.CreateBackendSetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return networkloadbalancersdk.CreateBackendSetDetails{}, fmt.Errorf("decode BackendSet create details: %w", err)
	}
	return details, nil
}

func backendSetUpdateDetailsFromValue(value any) (networkloadbalancersdk.UpdateBackendSetDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("marshal BackendSet update details source: %w", err)
	}

	var details networkloadbalancersdk.UpdateBackendSetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("decode BackendSet update details: %w", err)
	}
	return details, nil
}

//nolint:gocyclo // The generated SDK exposes several equivalent readback wrapper shapes for BackendSet.
func backendSetUpdateSource(resource *networkloadbalancerv1beta1.BackendSet, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("backendset resource is nil")
		}
		return resource.Status, nil
	case networkloadbalancersdk.BackendSet:
		return current, nil
	case *networkloadbalancersdk.BackendSet:
		if current == nil {
			return nil, fmt.Errorf("current BackendSet response is nil")
		}
		return *current, nil
	case networkloadbalancersdk.BackendSetSummary:
		return current, nil
	case *networkloadbalancersdk.BackendSetSummary:
		if current == nil {
			return nil, fmt.Errorf("current BackendSet response is nil")
		}
		return *current, nil
	case networkloadbalancersdk.GetBackendSetResponse:
		return current.BackendSet, nil
	case *networkloadbalancersdk.GetBackendSetResponse:
		if current == nil {
			return nil, fmt.Errorf("current BackendSet response is nil")
		}
		return current.BackendSet, nil
	default:
		return currentResponse, nil
	}
}

func backendSetUpdateNeeded(
	desired networkloadbalancersdk.UpdateBackendSetDetails,
	current networkloadbalancersdk.UpdateBackendSetDetails,
) (bool, error) {
	desiredComparable, err := cloneBackendSetUpdateDetails(desired)
	if err != nil {
		return false, err
	}
	currentComparable, err := cloneBackendSetUpdateDetails(current)
	if err != nil {
		return false, err
	}
	normalizeBackendSetOptionalFalseBools(reflect.ValueOf(&desiredComparable))
	normalizeBackendSetOptionalFalseBools(reflect.ValueOf(&currentComparable))
	stripBackendSetUpdateReadOnlyBackendNames(&desiredComparable)
	stripBackendSetUpdateReadOnlyBackendNames(&currentComparable)
	normalizeBackendSetDefaultedReadback(&desiredComparable, &currentComparable)

	desiredPayload, err := json.Marshal(desiredComparable)
	if err != nil {
		return false, fmt.Errorf("marshal desired BackendSet update details: %w", err)
	}
	currentPayload, err := json.Marshal(currentComparable)
	if err != nil {
		return false, fmt.Errorf("marshal current BackendSet update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func stripBackendSetCreateReadOnlyBackendNames(details *networkloadbalancersdk.CreateBackendSetDetails) {
	if details == nil {
		return
	}
	for i := range details.Backends {
		details.Backends[i].Name = nil
	}
}

func stripBackendSetUpdateReadOnlyBackendNames(details *networkloadbalancersdk.UpdateBackendSetDetails) {
	if details == nil {
		return
	}
	for i := range details.Backends {
		details.Backends[i].Name = nil
	}
}

func normalizeBackendSetDefaultedReadback(
	desired *networkloadbalancersdk.UpdateBackendSetDetails,
	current *networkloadbalancersdk.UpdateBackendSetDetails,
) {
	if desired == nil || current == nil {
		return
	}

	normalizeBackendSetDefaultBool(desired.IsPreserveSource, &current.IsPreserveSource, true)
	normalizeBackendSetDefaultBool(desired.IsInstantFailoverTcpResetEnabled, &current.IsInstantFailoverTcpResetEnabled, true)
	if desired.IpVersion == "" && current.IpVersion == networkloadbalancersdk.IpVersionIpv4 {
		current.IpVersion = ""
	}
	normalizeBackendSetHealthCheckerDefaults(desired.HealthChecker, current.HealthChecker)
	normalizeBackendSetBackendDefaults(desired.Backends, current.Backends)
}

func normalizeBackendSetDefaultBool(desired *bool, current **bool, defaultValue bool) {
	if desired != nil || current == nil || *current == nil || **current != defaultValue {
		return
	}
	*current = nil
}

func normalizeBackendSetHealthCheckerDefaults(desired, current *networkloadbalancersdk.HealthCheckerDetails) {
	if desired == nil || current == nil {
		return
	}
	normalizeBackendSetDefaultInt(desired.Retries, &current.Retries, 3)
	normalizeBackendSetDefaultInt(desired.TimeoutInMillis, &current.TimeoutInMillis, 3000)
	normalizeBackendSetDefaultInt(desired.IntervalInMillis, &current.IntervalInMillis, 10000)
}

func normalizeBackendSetBackendDefaults(desired, current []networkloadbalancersdk.BackendDetails) {
	if len(desired) == 0 || len(current) == 0 {
		return
	}
	desiredByIdentity := make(map[string]networkloadbalancersdk.BackendDetails, len(desired))
	for _, backend := range desired {
		if identity := backendSetBackendDefaultIdentity(backend); identity != "" {
			desiredByIdentity[identity] = backend
		}
	}
	for i := range current {
		identity := backendSetBackendDefaultIdentity(current[i])
		desiredBackend, ok := desiredByIdentity[identity]
		if !ok {
			continue
		}
		normalizeBackendSetDefaultInt(desiredBackend.Weight, &current[i].Weight, 1)
	}
}

func backendSetBackendDefaultIdentity(backend networkloadbalancersdk.BackendDetails) string {
	return strings.Join([]string{
		stringPointerValue(backend.IpAddress),
		stringPointerValue(backend.TargetId),
		fmt.Sprintf("%d", intPointerValue(backend.Port)),
	}, "\x00")
}

func normalizeBackendSetDefaultInt(desired *int, current **int, defaultValue int) {
	if desired != nil || current == nil || *current == nil || **current != defaultValue {
		return
	}
	*current = nil
}

func cloneBackendSetUpdateDetails(details networkloadbalancersdk.UpdateBackendSetDetails) (networkloadbalancersdk.UpdateBackendSetDetails, error) {
	payload, err := json.Marshal(details)
	if err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("marshal BackendSet update details clone: %w", err)
	}

	var cloned networkloadbalancersdk.UpdateBackendSetDetails
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return networkloadbalancersdk.UpdateBackendSetDetails{}, fmt.Errorf("decode BackendSet update details clone: %w", err)
	}
	return cloned, nil
}

func resolveBackendSetIdentity(resource *networkloadbalancerv1beta1.BackendSet) (backendSetIdentity, error) {
	if resource == nil {
		return backendSetIdentity{}, fmt.Errorf("resolve BackendSet identity: resource is nil")
	}

	statusNetworkLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationNetworkLoadBalancerID := strings.TrimSpace(resource.Annotations[backendSetNetworkLoadBalancerIDAnnotation])
	if statusNetworkLoadBalancerID != "" && annotationNetworkLoadBalancerID != "" && statusNetworkLoadBalancerID != annotationNetworkLoadBalancerID {
		return backendSetIdentity{}, fmt.Errorf(
			"resolve BackendSet identity: %s changed from recorded networkLoadBalancerId %q to %q",
			backendSetNetworkLoadBalancerIDAnnotation,
			statusNetworkLoadBalancerID,
			annotationNetworkLoadBalancerID,
		)
	}

	statusBackendSetName := strings.TrimSpace(resource.Status.Name)
	specBackendSetName := strings.TrimSpace(resource.Spec.Name)
	if statusBackendSetName != "" && specBackendSetName != "" && statusBackendSetName != specBackendSetName {
		return backendSetIdentity{}, fmt.Errorf("BackendSet formal semantics require replacement when name changes")
	}

	identity := backendSetIdentity{
		networkLoadBalancerID: firstNonEmptyTrim(statusNetworkLoadBalancerID, annotationNetworkLoadBalancerID),
		backendSetName:        firstNonEmptyTrim(statusBackendSetName, specBackendSetName, resource.Name),
	}
	if identity.networkLoadBalancerID == "" {
		return backendSetIdentity{}, fmt.Errorf("resolve BackendSet identity: %s annotation is required", backendSetNetworkLoadBalancerIDAnnotation)
	}
	if identity.backendSetName == "" {
		return backendSetIdentity{}, fmt.Errorf("resolve BackendSet identity: backend set name is empty")
	}
	return identity, nil
}

func recordBackendSetPathIdentity(resource *networkloadbalancerv1beta1.BackendSet, identity backendSetIdentity) {
	if resource == nil {
		return
	}
	resource.Status.Name = identity.backendSetName
	// BackendSet has no child OCID in the Network Load Balancer API, so the runtime records
	// the parent networkLoadBalancerId as the stable path identity used for Get/Update/Delete.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.networkLoadBalancerID)
}

func recordBackendSetTrackedIdentity(resource *networkloadbalancerv1beta1.BackendSet, identity backendSetIdentity) {
	recordBackendSetPathIdentity(resource, identity)
}

func handleBackendSetDeleteError(resource *networkloadbalancerv1beta1.BackendSet, err error) error {
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("BackendSet delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func (c backendSetPendingWorkRequestDeleteClient) CreateOrUpdate(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.BackendSet,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !backendSetNeedsPostWorkRequestObserve(resource, response) {
		return response, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c backendSetPendingWorkRequestDeleteClient) Delete(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.BackendSet,
) (bool, error) {
	if backendSetHasPendingWriteWorkRequest(resource) {
		response, err := c.delegate.CreateOrUpdate(ctx, resource, ctrl.Request{})
		if err != nil {
			return false, err
		}
		if backendSetHasPendingWriteWorkRequest(resource) || response.ShouldRequeue {
			return false, nil
		}
	}
	if err := c.rejectAmbiguousSucceededDeleteWorkRequestConfirmation(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c backendSetPendingWorkRequestDeleteClient) rejectAmbiguousSucceededDeleteWorkRequestConfirmation(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.BackendSet,
) error {
	workRequestID := backendSetPendingDeleteWorkRequestID(resource)
	if workRequestID == "" {
		return nil
	}

	workRequest, err := getBackendSetWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, workRequestID)
	if err != nil {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return err
	}
	currentAsync, err := buildBackendSetWorkRequestOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return err
	}
	if currentAsync.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return nil
	}
	return c.rejectAmbiguousDeleteConfirmation(ctx, resource)
}

func (c backendSetPendingWorkRequestDeleteClient) rejectAmbiguousDeleteConfirmation(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.BackendSet,
) error {
	identity, err := resolveBackendSetIdentity(resource)
	if err != nil {
		return err
	}

	if c.getBackendSet != nil {
		_, err := c.getBackendSet(ctx, networkloadbalancersdk.GetBackendSetRequest{
			NetworkLoadBalancerId: common.String(identity.networkLoadBalancerID),
			BackendSetName:        common.String(identity.backendSetName),
		})
		return backendSetAmbiguousDeleteConfirmationError(resource, err)
	}

	if c.listBackendSets == nil {
		return nil
	}
	_, err = c.listBackendSets(ctx, networkloadbalancersdk.ListBackendSetsRequest{
		NetworkLoadBalancerId: common.String(identity.networkLoadBalancerID),
	})
	return backendSetAmbiguousDeleteConfirmationError(resource, err)
}

func backendSetAmbiguousDeleteConfirmationError(resource *networkloadbalancerv1beta1.BackendSet, err error) error {
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("BackendSet delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func backendSetPendingDeleteWorkRequestID(resource *networkloadbalancerv1beta1.BackendSet) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func backendSetHasPendingWriteWorkRequest(resource *networkloadbalancerv1beta1.BackendSet) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func buildBackendSetWorkRequestOperation(
	resource *networkloadbalancerv1beta1.BackendSet,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := backendSetWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}

	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}
	operation, err := servicemanager.BuildWorkRequestAsyncOperation(status, backendSetWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        string(current.OperationType),
		RawOperationType: string(current.OperationType),
		WorkRequestID:    stringPointerValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := backendSetWorkRequestMessage(operation.Phase, current); message != "" {
		operation.Message = message
	}
	return operation, nil
}

func backendSetNeedsPostWorkRequestObserve(
	resource *networkloadbalancerv1beta1.BackendSet,
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

func overlayBackendSetExistingBoolFields(value reflect.Value, decoded any) any {
	overlaid, _ := overlayBackendSetExistingBoolFieldsValue(value, decoded)
	return overlaid
}

//nolint:gocognit,gocyclo // Recursive bool preservation follows the nested generated SDK/API shape.
func overlayBackendSetExistingBoolFieldsValue(value reflect.Value, decoded any) (any, bool) {
	value, ok := indirectBackendSetValue(value)
	if !ok {
		return decoded, decoded != nil
	}

	switch value.Kind() {
	case reflect.Struct:
		decodedMap, ok := decoded.(map[string]any)
		if !ok || decodedMap == nil {
			return decoded, decoded != nil
		}
		hasAny := len(decodedMap) > 0
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			fieldType := valueType.Field(i)
			if !fieldType.IsExported() {
				continue
			}

			jsonName := backendSetJSONFieldName(fieldType)
			if jsonName == "" {
				continue
			}

			childDecoded, exists := decodedMap[jsonName]
			if !exists {
				continue
			}

			fieldValue := value.Field(i)
			indirectField, ok := indirectBackendSetValue(fieldValue)
			if !ok {
				continue
			}

			switch indirectField.Kind() {
			case reflect.Bool:
				decodedMap[jsonName] = indirectField.Bool()
				hasAny = true
			case reflect.Struct, reflect.Slice, reflect.Array:
				child, childHasAny := overlayBackendSetExistingBoolFieldsValue(fieldValue, childDecoded)
				if childHasAny {
					decodedMap[jsonName] = child
					hasAny = true
				}
			}
		}
		return decodedMap, hasAny
	case reflect.Slice, reflect.Array:
		decodedSlice, ok := decoded.([]any)
		if !ok {
			return decoded, decoded != nil
		}
		hasAny := len(decodedSlice) > 0
		limit := value.Len()
		if len(decodedSlice) < limit {
			limit = len(decodedSlice)
		}
		for i := 0; i < limit; i++ {
			child, childHasAny := overlayBackendSetExistingBoolFieldsValue(value.Index(i), decodedSlice[i])
			if childHasAny {
				decodedSlice[i] = child
				hasAny = true
			}
		}
		return decodedSlice, hasAny
	default:
		return decoded, decoded != nil
	}
}

//nolint:gocognit,gocyclo // Recursive normalization is needed for nested optional SDK bool pointers.
func normalizeBackendSetOptionalFalseBools(value reflect.Value) {
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return
		}
		if value.Elem().Kind() == reflect.Bool {
			if !value.Elem().Bool() && value.CanSet() {
				value.Set(reflect.Zero(value.Type()))
			}
			return
		}
		normalizeBackendSetOptionalFalseBools(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			normalizeBackendSetOptionalFalseBools(value.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			normalizeBackendSetOptionalFalseBools(value.Index(i))
		}
	}
}

func indirectBackendSetValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func backendSetJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
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

func intPointerValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
