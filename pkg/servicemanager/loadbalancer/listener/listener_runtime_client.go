/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type listenerRuntimeOCIClient interface {
	CreateListener(context.Context, loadbalancersdk.CreateListenerRequest) (loadbalancersdk.CreateListenerResponse, error)
	GetLoadBalancer(context.Context, loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error)
	UpdateListener(context.Context, loadbalancersdk.UpdateListenerRequest) (loadbalancersdk.UpdateListenerResponse, error)
	DeleteListener(context.Context, loadbalancersdk.DeleteListenerRequest) (loadbalancersdk.DeleteListenerResponse, error)
}

type listenerReadRequest struct {
	LoadBalancerId *string
	ListenerName   *string
}

type listenerListRequest struct {
	LoadBalancerId *string
}

type listenerListResult struct {
	Items []listenerRuntimeView `json:"items"`
}

type listenerIdentity struct {
	loadBalancerID string
	listenerName   string
}

type listenerRuntimeView struct {
	loadbalancersdk.Listener
	Ocid           string `json:"ocid,omitempty"`
	LoadBalancerId string `json:"loadBalancerId,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type listenerNotFoundServiceError struct {
	listenerName string
}

func (e listenerNotFoundServiceError) Error() string {
	return fmt.Sprintf("listener %q was not found on the load balancer", e.listenerName)
}

func (e listenerNotFoundServiceError) GetHTTPStatusCode() int {
	return 404
}

func (e listenerNotFoundServiceError) GetMessage() string {
	return e.Error()
}

func (e listenerNotFoundServiceError) GetCode() string {
	return "NotFound"
}

func (e listenerNotFoundServiceError) GetOpcRequestID() string {
	return ""
}

func init() {
	registerListenerRuntimeHooksMutator(func(manager *ListenerServiceManager, hooks *ListenerRuntimeHooks) {
		applyListenerRuntimeHooks(listenerCredentialClient(manager), hooks)
		applyListenerReadHooks(hooks, providerListenerGetLoadBalancerCall(manager))
	})
}

func newGeneratedListenerServiceClient(
	client listenerRuntimeOCIClient,
	log loggerutil.OSOKLogger,
	credentialClient credhelper.CredentialClient,
	initErr error,
) ListenerServiceClient {
	hooks := newListenerRuntimeHooksWithOCIClient(client)
	applyListenerRuntimeHooks(credentialClient, &hooks)
	applyListenerReadHooks(&hooks, client.GetLoadBalancer)
	config := buildListenerGeneratedRuntimeConfig(&ListenerServiceManager{Log: log}, hooks)
	config.CredentialClient = credentialClient
	config.InitError = initErr

	return defaultListenerServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.Listener](config),
	}
}

func applyListenerRuntimeHooks(credentialClient credhelper.CredentialClient, hooks *ListenerRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = listenerRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *loadbalancerv1beta1.Listener,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildListenerUpdateBody(ctx, resource, credentialClient, namespace, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.Listener]{
		Resolve: func(resource *loadbalancerv1beta1.Listener) (any, error) {
			return resolveListenerIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.Listener, identity any) {
			recordListenerPathIdentity(resource, identity.(listenerIdentity))
		},
		RecordTracked: func(resource *loadbalancerv1beta1.Listener, identity any, resourceID string) {
			recordListenerTrackedIdentity(resource, identity.(listenerIdentity), resourceID)
		},
	}
	hooks.Create.Fields = listenerCreateFields()
	hooks.Update.Fields = listenerUpdateFields()
	hooks.Delete.Fields = listenerDeleteFields()
}

func applyListenerReadHooks(
	hooks *ListenerRuntimeHooks,
	getLoadBalancer func(context.Context, loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error),
) {
	if hooks == nil {
		return
	}

	hooks.Read = generatedruntime.ReadHooks{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &listenerReadRequest{} },
			Fields:     listenerGetFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return getListenerRuntimeView(ctx, listenerReadCallAdapter(getLoadBalancer), request.(*listenerReadRequest))
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &listenerListRequest{} },
			Fields:     listenerListFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return listListenerRuntimeViews(ctx, listenerReadCallAdapter(getLoadBalancer), request.(*listenerListRequest))
			},
		},
	}
}

func newListenerRuntimeHooksWithOCIClient(client listenerRuntimeOCIClient) ListenerRuntimeHooks {
	return ListenerRuntimeHooks{
		Semantics: listenerRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.Listener]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[loadbalancersdk.CreateListenerRequest, loadbalancersdk.CreateListenerResponse]{
			Fields: listenerCreateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.CreateListenerRequest) (loadbalancersdk.CreateListenerResponse, error) {
				return client.CreateListener(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdateListenerRequest, loadbalancersdk.UpdateListenerResponse]{
			Fields: listenerUpdateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.UpdateListenerRequest) (loadbalancersdk.UpdateListenerResponse, error) {
				return client.UpdateListener(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeleteListenerRequest, loadbalancersdk.DeleteListenerResponse]{
			Fields: listenerDeleteFields(),
			Call: func(ctx context.Context, request loadbalancersdk.DeleteListenerRequest) (loadbalancersdk.DeleteListenerResponse, error) {
				return client.DeleteListener(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ListenerServiceClient) ListenerServiceClient{},
	}
}

func listenerCredentialClient(manager *ListenerServiceManager) credhelper.CredentialClient {
	if manager == nil {
		return nil
	}
	return manager.CredentialClient
}

func providerListenerGetLoadBalancerCall(
	manager *ListenerServiceManager,
) func(context.Context, loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error) {
	return func(ctx context.Context, request loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error) {
		if manager == nil || manager.Provider == nil {
			return loadbalancersdk.GetLoadBalancerResponse{}, fmt.Errorf("listener read OCI client is not configured")
		}

		client, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
		if err != nil {
			return loadbalancersdk.GetLoadBalancerResponse{}, fmt.Errorf("initialize Listener OCI client: %w", err)
		}
		return client.GetLoadBalancer(ctx, request)
	}
}

func listenerReadCallAdapter(
	getLoadBalancer func(context.Context, loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error),
) listenerRuntimeOCIClient {
	return listenerReadClient{getLoadBalancer: getLoadBalancer}
}

type listenerReadClient struct {
	getLoadBalancer func(context.Context, loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error)
}

func (c listenerReadClient) CreateListener(context.Context, loadbalancersdk.CreateListenerRequest) (loadbalancersdk.CreateListenerResponse, error) {
	return loadbalancersdk.CreateListenerResponse{}, fmt.Errorf("listener read client does not support CreateListener")
}

func (c listenerReadClient) GetLoadBalancer(ctx context.Context, request loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error) {
	if c.getLoadBalancer == nil {
		return loadbalancersdk.GetLoadBalancerResponse{}, fmt.Errorf("listener read OCI call is not configured")
	}
	return c.getLoadBalancer(ctx, request)
}

func (c listenerReadClient) UpdateListener(context.Context, loadbalancersdk.UpdateListenerRequest) (loadbalancersdk.UpdateListenerResponse, error) {
	return loadbalancersdk.UpdateListenerResponse{}, fmt.Errorf("listener read client does not support UpdateListener")
}

func (c listenerReadClient) DeleteListener(context.Context, loadbalancersdk.DeleteListenerRequest) (loadbalancersdk.DeleteListenerResponse, error) {
	return loadbalancersdk.DeleteListenerResponse{}, fmt.Errorf("listener read client does not support DeleteListener")
}

func listenerRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "listener",
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
			Mutable: []string{
				"connectionConfiguration",
				"defaultBackendSetName",
				"hostnameNames",
				"pathRouteSetName",
				"port",
				"protocol",
				"routingPolicyName",
				"ruleSetNames",
				"sslConfiguration",
			},
			ForceNew:      []string{"loadBalancerId", "name"},
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

func listenerCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerLoadBalancerIDField(),
		{
			FieldName:    "CreateListenerDetails",
			RequestName:  "CreateListenerDetails",
			Contribution: "body",
		},
	}
}

func listenerGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerLoadBalancerIDField(),
		listenerNameField(),
	}
}

func listenerListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerLoadBalancerIDField(),
	}
}

func listenerUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerLoadBalancerIDField(),
		listenerNameField(),
		{
			FieldName:    "UpdateListenerDetails",
			RequestName:  "UpdateListenerDetails",
			Contribution: "body",
		},
	}
}

func listenerDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerLoadBalancerIDField(),
		listenerNameField(),
	}
}

func listenerLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "LoadBalancerId",
		RequestName:  "loadBalancerId",
		Contribution: "path",
		LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
	}
}

func listenerNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "ListenerName",
		RequestName:  "listenerName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func resolveListenerIdentity(resource *loadbalancerv1beta1.Listener) (listenerIdentity, error) {
	identity := listenerIdentity{
		loadBalancerID: firstNonEmptyTrim(resource.Status.LoadBalancerId, resource.Spec.LoadBalancerId),
		listenerName:   firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name, resource.Name),
	}
	if identity.loadBalancerID == "" {
		return listenerIdentity{}, fmt.Errorf("resolve Listener identity: loadBalancerId is empty")
	}
	if identity.listenerName == "" {
		return listenerIdentity{}, fmt.Errorf("resolve Listener identity: listener name is empty")
	}
	return identity, nil
}

func recordListenerPathIdentity(resource *loadbalancerv1beta1.Listener, identity listenerIdentity) {
	if resource == nil {
		return
	}
	resource.Status.LoadBalancerId = identity.loadBalancerID
	resource.Status.Name = identity.listenerName
}

func recordListenerTrackedIdentity(resource *loadbalancerv1beta1.Listener, identity listenerIdentity, resourceID string) {
	recordListenerPathIdentity(resource, identity)
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		resourceID = listenerSyntheticOCID(identity.loadBalancerID, identity.listenerName)
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
}

func buildListenerUpdateBody(
	ctx context.Context,
	resource *loadbalancerv1beta1.Listener,
	credentialClient credhelper.CredentialClient,
	namespace string,
	currentResponse any,
) (loadbalancersdk.UpdateListenerDetails, bool, error) {
	if resource == nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, fmt.Errorf("listener resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credentialClient, namespace)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, err
	}

	currentSource, err := listenerUpdateSource(resource, currentResponse)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, err
	}

	resolvedSpec = pruneListenerResolvedSpec(resolvedSpec)
	resolvedSpec = preserveListenerExplicitFalseSSLConfig(resolvedSpec, resource, currentSource)
	desiredValues, err := listenerJSONMap(resolvedSpec)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, fmt.Errorf("marshal desired Listener update values: %w", err)
	}
	if err := overlayListenerExplicitMutableClears(desiredValues, currentSource); err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, err
	}
	desiredValues, err = listenerJSONMap(desiredValues)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, fmt.Errorf("normalize desired Listener update values: %w", err)
	}
	if err := rejectListenerUnsupportedMutableClears(desiredValues, currentSource); err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, err
	}

	desired, err := listenerUpdateDetailsFromValue(desiredValues)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, fmt.Errorf("build desired Listener update details: %w", err)
	}

	current, err := listenerUpdateDetailsFromValue(currentSource)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, err
	}
	updateNeeded, err := listenerUpdateNeeded(desired, current)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, false, err
	}
	if !updateNeeded {
		return loadbalancersdk.UpdateListenerDetails{}, false, nil
	}

	return desired, true, nil
}

func listenerJSONMap(value any) (map[string]any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, nil
}

func listenerLookupMeaningfulValue(values map[string]any, path string) (any, bool) {
	value, ok := listenerLookupValue(values, path)
	if !ok || !listenerMeaningfulValue(value) {
		return nil, false
	}
	return value, true
}

func listenerLookupValue(values map[string]any, path string) (any, bool) {
	if values == nil {
		return nil, false
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	current := any(values)
	for _, segment := range strings.Split(path, ".") {
		next, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = next[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func listenerMeaningfulValue(value any) bool {
	if value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case []any:
		for _, item := range concrete {
			if listenerMeaningfulValue(item) {
				return true
			}
		}
		return false
	case map[string]any:
		for _, item := range concrete {
			if listenerMeaningfulValue(item) {
				return true
			}
		}
		return false
	case bool:
		return true
	case float64:
		return concrete != 0
	default:
		return true
	}
}

func pruneListenerResolvedSpec(value any) any {
	pruned, ok := pruneListenerResolvedSpecValue(value)
	if !ok {
		return map[string]any{}
	}
	return pruned
}

func pruneListenerResolvedSpecValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		falseChildren := make(map[string]any)
		for key, child := range concrete {
			prunedChild, keepChild := pruneListenerResolvedSpecValue(child)
			if keepChild {
				pruned[key] = prunedChild
				continue
			}
			if boolean, ok := child.(bool); ok && !boolean {
				falseChildren[key] = child
			}
		}
		if len(pruned) == 0 {
			return nil, false
		}
		for key, child := range falseChildren {
			pruned[key] = child
		}
		return pruned, true
	case []any:
		pruned := make([]any, 0, len(concrete))
		for _, child := range concrete {
			prunedChild, keepChild := pruneListenerResolvedSpecValue(child)
			if keepChild {
				pruned = append(pruned, prunedChild)
			}
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case string:
		if strings.TrimSpace(concrete) == "" {
			return nil, false
		}
		return concrete, true
	case float64:
		if concrete == 0 {
			return nil, false
		}
		return concrete, true
	case bool:
		if !concrete {
			return nil, false
		}
		return concrete, true
	case nil:
		return nil, false
	default:
		return concrete, true
	}
}

func getListenerRuntimeView(ctx context.Context, client listenerRuntimeOCIClient, request *listenerReadRequest) (listenerRuntimeView, error) {
	if request == nil {
		return listenerRuntimeView{}, fmt.Errorf("listener read request is nil")
	}

	loadBalancer, err := client.GetLoadBalancer(ctx, loadbalancersdk.GetLoadBalancerRequest{
		LoadBalancerId: request.LoadBalancerId,
	})
	if err != nil {
		return listenerRuntimeView{}, err
	}

	view, ok := listenerRuntimeViewFromLoadBalancer(loadBalancer.LoadBalancer, stringValue(request.ListenerName))
	if !ok {
		return listenerRuntimeView{}, listenerNotFoundServiceError{listenerName: stringValue(request.ListenerName)}
	}
	return view, nil
}

func listListenerRuntimeViews(ctx context.Context, client listenerRuntimeOCIClient, request *listenerListRequest) (listenerListResult, error) {
	if request == nil {
		return listenerListResult{}, fmt.Errorf("listener list request is nil")
	}

	loadBalancer, err := client.GetLoadBalancer(ctx, loadbalancersdk.GetLoadBalancerRequest{
		LoadBalancerId: request.LoadBalancerId,
	})
	if err != nil {
		return listenerListResult{}, err
	}

	return listenerListResult{
		Items: listenerRuntimeViewsFromLoadBalancer(loadBalancer.LoadBalancer),
	}, nil
}

func listenerRuntimeViewFromLoadBalancer(loadBalancer loadbalancersdk.LoadBalancer, listenerName string) (listenerRuntimeView, bool) {
	listenerName = strings.TrimSpace(listenerName)
	if listenerName == "" || loadBalancer.Listeners == nil {
		return listenerRuntimeView{}, false
	}

	listener, ok := loadBalancer.Listeners[listenerName]
	if !ok {
		return listenerRuntimeView{}, false
	}
	if strings.TrimSpace(stringValue(listener.Name)) == "" {
		listener.Name = common.String(listenerName)
	}

	loadBalancerID := stringValue(loadBalancer.Id)
	return listenerRuntimeView{
		Listener:       listener,
		Ocid:           listenerSyntheticOCID(loadBalancerID, listenerName),
		LoadBalancerId: loadBalancerID,
	}, true
}

func listenerRuntimeViewsFromLoadBalancer(loadBalancer loadbalancersdk.LoadBalancer) []listenerRuntimeView {
	if len(loadBalancer.Listeners) == 0 {
		return nil
	}

	keys := make([]string, 0, len(loadBalancer.Listeners))
	for name := range loadBalancer.Listeners {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	items := make([]listenerRuntimeView, 0, len(keys))
	for _, name := range keys {
		view, ok := listenerRuntimeViewFromLoadBalancer(loadBalancer, name)
		if ok {
			items = append(items, view)
		}
	}
	return items
}

func listenerSyntheticOCID(loadBalancerID string, listenerName string) string {
	return "listener/" + strings.TrimSpace(loadBalancerID) + "/" + strings.TrimSpace(listenerName)
}

func listenerUpdateSource(resource *loadbalancerv1beta1.Listener, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("listener resource is nil")
		}
		return resource.Status, nil
	case listenerRuntimeView:
		return current.Listener, nil
	case *listenerRuntimeView:
		if current == nil {
			return nil, fmt.Errorf("current Listener response is nil")
		}
		return current.Listener, nil
	case loadbalancersdk.Listener:
		return current, nil
	case *loadbalancersdk.Listener:
		if current == nil {
			return nil, fmt.Errorf("current Listener response is nil")
		}
		return *current, nil
	default:
		return currentResponse, nil
	}
}

func listenerUpdateDetailsFromValue(value any) (loadbalancersdk.UpdateListenerDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, fmt.Errorf("marshal Listener update details source: %w", err)
	}

	var details loadbalancersdk.UpdateListenerDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, fmt.Errorf("decode Listener update details: %w", err)
	}
	return details, nil
}

func preserveListenerExplicitFalseSSLConfig(pruned any, resource *loadbalancerv1beta1.Listener, currentSource any) any {
	if resource == nil || resource.Spec.SslConfiguration.VerifyPeerCertificate {
		return pruned
	}

	currentValues, err := listenerJSONMap(currentSource)
	if err != nil {
		return pruned
	}
	if _, ok := listenerLookupMeaningfulValue(currentValues, "sslConfiguration"); !ok {
		return pruned
	}

	values, ok := pruned.(map[string]any)
	if !ok || values == nil {
		values = map[string]any{}
	}
	listenerSetValue(values, "sslConfiguration.verifyPeerCertificate", false)
	return values
}

func overlayListenerExplicitMutableClears(values map[string]any, currentSource any) error {
	currentValues, err := listenerJSONMap(currentSource)
	if err != nil {
		return fmt.Errorf("marshal current Listener clear detection source: %w", err)
	}

	for _, path := range []string{"pathRouteSetName", "routingPolicyName"} {
		if _, ok := listenerLookupValue(values, path); ok {
			continue
		}
		if _, ok := listenerLookupMeaningfulValue(currentValues, path); ok {
			listenerSetValue(values, path, "")
		}
	}

	for _, path := range []string{"hostnameNames", "ruleSetNames"} {
		if _, ok := listenerLookupValue(values, path); ok {
			continue
		}
		if _, ok := listenerLookupMeaningfulValue(currentValues, path); ok {
			listenerSetValue(values, path, []string{})
		}
	}

	return nil
}

func rejectListenerUnsupportedMutableClears(desiredValues map[string]any, currentSource any) error {
	currentValues, err := listenerJSONMap(currentSource)
	if err != nil {
		return fmt.Errorf("marshal current Listener nested clear source: %w", err)
	}

	var unsupported []string
	for _, path := range []string{"connectionConfiguration", "sslConfiguration"} {
		currentValue, ok := listenerLookupValue(currentValues, path)
		if !ok || !listenerMeaningfulValue(currentValue) {
			continue
		}

		desiredValue, _ := listenerLookupValue(desiredValues, path)
		unsupported = append(unsupported, listenerOmittedMeaningfulPaths(currentValue, desiredValue, path)...)
	}
	if len(unsupported) == 0 {
		return nil
	}

	sort.Strings(unsupported)
	return fmt.Errorf(
		"listener update does not support clearing mutable fields %s; specify replacement values instead",
		strings.Join(unsupported, ", "),
	)
}

func listenerOmittedMeaningfulPaths(current any, desired any, prefix string) []string {
	switch concrete := current.(type) {
	case map[string]any:
		desiredMap, _ := desired.(map[string]any)
		keys := make([]string, 0, len(concrete))
		for key := range concrete {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var paths []string
		for _, key := range keys {
			childPrefix := key
			if prefix != "" {
				childPrefix = prefix + "." + key
			}
			childDesired, ok := desiredMap[key]
			if !ok {
				if listenerMeaningfulValue(concrete[key]) {
					paths = append(paths, childPrefix)
				}
				continue
			}
			paths = append(paths, listenerOmittedMeaningfulPaths(concrete[key], childDesired, childPrefix)...)
		}
		return paths
	default:
		if listenerMeaningfulValue(current) && !listenerMeaningfulValue(desired) {
			return []string{prefix}
		}
		return nil
	}
}

func listenerUpdateNeeded(desired loadbalancersdk.UpdateListenerDetails, current loadbalancersdk.UpdateListenerDetails) (bool, error) {
	desiredComparable, err := cloneListenerUpdateDetails(desired)
	if err != nil {
		return false, err
	}
	currentComparable, err := cloneListenerUpdateDetails(current)
	if err != nil {
		return false, err
	}

	normalizeListenerOptionalFalseBools(reflect.ValueOf(&desiredComparable))
	normalizeListenerOptionalFalseBools(reflect.ValueOf(&currentComparable))

	desiredPayload, err := json.Marshal(desiredComparable)
	if err != nil {
		return false, fmt.Errorf("marshal desired Listener update details: %w", err)
	}
	currentPayload, err := json.Marshal(currentComparable)
	if err != nil {
		return false, fmt.Errorf("marshal current Listener update details: %w", err)
	}

	return string(desiredPayload) != string(currentPayload), nil
}

func cloneListenerUpdateDetails(details loadbalancersdk.UpdateListenerDetails) (loadbalancersdk.UpdateListenerDetails, error) {
	payload, err := json.Marshal(details)
	if err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, fmt.Errorf("marshal Listener update details clone: %w", err)
	}

	var cloned loadbalancersdk.UpdateListenerDetails
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return loadbalancersdk.UpdateListenerDetails{}, fmt.Errorf("decode Listener update details clone: %w", err)
	}
	return cloned, nil
}

func normalizeListenerOptionalFalseBools(value reflect.Value) {
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
		normalizeListenerOptionalFalseBools(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			normalizeListenerOptionalFalseBools(value.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			normalizeListenerOptionalFalseBools(value.Index(i))
		}
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func listenerSetValue(values map[string]any, path string, value any) {
	if values == nil {
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	current := values
	segments := strings.Split(path, ".")
	for i, segment := range segments {
		if i == len(segments)-1 {
			current[segment] = value
			return
		}

		next, ok := current[segment].(map[string]any)
		if !ok || next == nil {
			next = map[string]any{}
			current[segment] = next
		}
		current = next
	}
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
