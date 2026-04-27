/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pathrouteset

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

const pathRouteSetLoadBalancerIDAnnotation = "loadbalancer.oracle.com/load-balancer-id"

type pathRouteSetRuntimeOCIClient interface {
	CreatePathRouteSet(context.Context, loadbalancersdk.CreatePathRouteSetRequest) (loadbalancersdk.CreatePathRouteSetResponse, error)
	GetPathRouteSet(context.Context, loadbalancersdk.GetPathRouteSetRequest) (loadbalancersdk.GetPathRouteSetResponse, error)
	ListPathRouteSets(context.Context, loadbalancersdk.ListPathRouteSetsRequest) (loadbalancersdk.ListPathRouteSetsResponse, error)
	UpdatePathRouteSet(context.Context, loadbalancersdk.UpdatePathRouteSetRequest) (loadbalancersdk.UpdatePathRouteSetResponse, error)
	DeletePathRouteSet(context.Context, loadbalancersdk.DeletePathRouteSetRequest) (loadbalancersdk.DeletePathRouteSetResponse, error)
}

type pathRouteSetIdentity struct {
	loadBalancerID   string
	pathRouteSetName string
}

func init() {
	registerPathRouteSetRuntimeHooksMutator(func(_ *PathRouteSetServiceManager, hooks *PathRouteSetRuntimeHooks) {
		applyPathRouteSetRuntimeHooks(hooks)
	})
}

func applyPathRouteSetRuntimeHooks(hooks *PathRouteSetRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newPathRouteSetRuntimeSemantics()
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.PathRouteSet,
		_ string,
	) (any, error) {
		return buildPathRouteSetCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loadbalancerv1beta1.PathRouteSet,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildPathRouteSetUpdateBody(resource, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.PathRouteSet]{
		Resolve: func(resource *loadbalancerv1beta1.PathRouteSet) (any, error) {
			return resolvePathRouteSetIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.PathRouteSet, identity any) {
			recordPathRouteSetPathIdentity(resource, identity.(pathRouteSetIdentity))
		},
		RecordTracked: func(resource *loadbalancerv1beta1.PathRouteSet, identity any, _ string) {
			recordPathRouteSetTrackedIdentity(resource, identity.(pathRouteSetIdentity))
		},
		LookupExisting: func(context.Context, *loadbalancerv1beta1.PathRouteSet, any) (any, error) {
			return nil, nil
		},
	}
	hooks.Create.Fields = pathRouteSetCreateFields()
	hooks.Get.Fields = pathRouteSetGetFields()
	hooks.List.Fields = pathRouteSetListFields()
	hooks.Update.Fields = pathRouteSetUpdateFields()
	hooks.Delete.Fields = pathRouteSetDeleteFields()
}

func newPathRouteSetRuntimeHooksWithOCIClient(client pathRouteSetRuntimeOCIClient) PathRouteSetRuntimeHooks {
	return PathRouteSetRuntimeHooks{
		Semantics: newPathRouteSetRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.PathRouteSet]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[loadbalancersdk.CreatePathRouteSetRequest, loadbalancersdk.CreatePathRouteSetResponse]{
			Fields: pathRouteSetCreateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.CreatePathRouteSetRequest) (loadbalancersdk.CreatePathRouteSetResponse, error) {
				return client.CreatePathRouteSet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loadbalancersdk.GetPathRouteSetRequest, loadbalancersdk.GetPathRouteSetResponse]{
			Fields: pathRouteSetGetFields(),
			Call: func(ctx context.Context, request loadbalancersdk.GetPathRouteSetRequest) (loadbalancersdk.GetPathRouteSetResponse, error) {
				return client.GetPathRouteSet(ctx, request)
			},
		},
		List: runtimeOperationHooks[loadbalancersdk.ListPathRouteSetsRequest, loadbalancersdk.ListPathRouteSetsResponse]{
			Fields: pathRouteSetListFields(),
			Call: func(ctx context.Context, request loadbalancersdk.ListPathRouteSetsRequest) (loadbalancersdk.ListPathRouteSetsResponse, error) {
				return client.ListPathRouteSets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdatePathRouteSetRequest, loadbalancersdk.UpdatePathRouteSetResponse]{
			Fields: pathRouteSetUpdateFields(),
			Call: func(ctx context.Context, request loadbalancersdk.UpdatePathRouteSetRequest) (loadbalancersdk.UpdatePathRouteSetResponse, error) {
				return client.UpdatePathRouteSet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeletePathRouteSetRequest, loadbalancersdk.DeletePathRouteSetResponse]{
			Fields: pathRouteSetDeleteFields(),
			Call: func(ctx context.Context, request loadbalancersdk.DeletePathRouteSetRequest) (loadbalancersdk.DeletePathRouteSetResponse, error) {
				return client.DeletePathRouteSet(ctx, request)
			},
		},
		WrapGeneratedClient: []func(PathRouteSetServiceClient) PathRouteSetServiceClient{},
	}
}

func newPathRouteSetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "pathrouteset",
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
			Mutable:       []string{"pathRoutes"},
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

func pathRouteSetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		pathRouteSetLoadBalancerIDField(),
		{
			FieldName:    "CreatePathRouteSetDetails",
			RequestName:  "CreatePathRouteSetDetails",
			Contribution: "body",
		},
	}
}

func pathRouteSetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		pathRouteSetLoadBalancerIDField(),
		pathRouteSetNameField(),
	}
}

func pathRouteSetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		pathRouteSetLoadBalancerIDField(),
	}
}

func pathRouteSetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		pathRouteSetLoadBalancerIDField(),
		pathRouteSetNameField(),
		{
			FieldName:    "UpdatePathRouteSetDetails",
			RequestName:  "UpdatePathRouteSetDetails",
			Contribution: "body",
		},
	}
}

func pathRouteSetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		pathRouteSetLoadBalancerIDField(),
		pathRouteSetNameField(),
	}
}

func pathRouteSetLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "LoadBalancerId",
		RequestName:      "loadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.status.ocid"},
	}
}

func pathRouteSetNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "PathRouteSetName",
		RequestName:  "pathRouteSetName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func buildPathRouteSetCreateBody(resource *loadbalancerv1beta1.PathRouteSet) (loadbalancersdk.CreatePathRouteSetDetails, error) {
	if resource == nil {
		return loadbalancersdk.CreatePathRouteSetDetails{}, fmt.Errorf("pathrouteset resource is nil")
	}
	return loadbalancersdk.CreatePathRouteSetDetails{
		Name:       stringPointer(firstNonEmptyTrim(resource.Spec.Name, resource.Name)),
		PathRoutes: pathRouteSetSDKPathRoutes(resource.Spec.PathRoutes),
	}, nil
}

func buildPathRouteSetUpdateBody(
	resource *loadbalancerv1beta1.PathRouteSet,
	currentResponse any,
) (loadbalancersdk.UpdatePathRouteSetDetails, bool, error) {
	if resource == nil {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, false, fmt.Errorf("pathrouteset resource is nil")
	}

	desired := loadbalancersdk.UpdatePathRouteSetDetails{
		PathRoutes: pathRouteSetSDKPathRoutes(resource.Spec.PathRoutes),
	}
	currentSource, err := pathRouteSetUpdateSource(resource, currentResponse)
	if err != nil {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, false, err
	}
	current, err := pathRouteSetUpdateDetailsFromValue(currentSource)
	if err != nil {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, false, err
	}

	updateNeeded, err := pathRouteSetUpdateNeeded(desired, current)
	if err != nil {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, false, err
	}
	if !updateNeeded {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, false, nil
	}
	return desired, true, nil
}

func pathRouteSetUpdateSource(resource *loadbalancerv1beta1.PathRouteSet, currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case nil:
		if resource == nil {
			return nil, fmt.Errorf("pathrouteset resource is nil")
		}
		return resource.Status, nil
	case loadbalancersdk.PathRouteSet:
		return current, nil
	case *loadbalancersdk.PathRouteSet:
		if current == nil {
			return nil, fmt.Errorf("current PathRouteSet response is nil")
		}
		return *current, nil
	case loadbalancersdk.GetPathRouteSetResponse:
		return current.PathRouteSet, nil
	case *loadbalancersdk.GetPathRouteSetResponse:
		if current == nil {
			return nil, fmt.Errorf("current PathRouteSet response is nil")
		}
		return current.PathRouteSet, nil
	default:
		return currentResponse, nil
	}
}

func pathRouteSetUpdateDetailsFromValue(value any) (loadbalancersdk.UpdatePathRouteSetDetails, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, fmt.Errorf("marshal PathRouteSet update details source: %w", err)
	}

	var details loadbalancersdk.UpdatePathRouteSetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return loadbalancersdk.UpdatePathRouteSetDetails{}, fmt.Errorf("decode PathRouteSet update details: %w", err)
	}
	return details, nil
}

func pathRouteSetUpdateNeeded(desired loadbalancersdk.UpdatePathRouteSetDetails, current loadbalancersdk.UpdatePathRouteSetDetails) (bool, error) {
	desiredPayload, err := json.Marshal(desired)
	if err != nil {
		return false, fmt.Errorf("marshal desired PathRouteSet update details: %w", err)
	}
	currentPayload, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal current PathRouteSet update details: %w", err)
	}
	return string(desiredPayload) != string(currentPayload), nil
}

func pathRouteSetSDKPathRoutes(routes []loadbalancerv1beta1.PathRouteSetPathRoute) []loadbalancersdk.PathRoute {
	if routes == nil {
		return nil
	}

	converted := make([]loadbalancersdk.PathRoute, 0, len(routes))
	for _, route := range routes {
		converted = append(converted, loadbalancersdk.PathRoute{
			Path: stringPointer(route.Path),
			PathMatchType: &loadbalancersdk.PathMatchType{
				MatchType: loadbalancersdk.PathMatchTypeMatchTypeEnum(route.PathMatchType.MatchType),
			},
			BackendSetName: stringPointer(route.BackendSetName),
		})
	}
	return converted
}

func resolvePathRouteSetIdentity(resource *loadbalancerv1beta1.PathRouteSet) (pathRouteSetIdentity, error) {
	if resource == nil {
		return pathRouteSetIdentity{}, fmt.Errorf("resolve PathRouteSet identity: resource is nil")
	}

	statusLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationLoadBalancerID := strings.TrimSpace(resource.Annotations[pathRouteSetLoadBalancerIDAnnotation])
	if statusLoadBalancerID != "" && annotationLoadBalancerID != "" && statusLoadBalancerID != annotationLoadBalancerID {
		return pathRouteSetIdentity{}, fmt.Errorf(
			"resolve PathRouteSet identity: %s changed from recorded loadBalancerId %q to %q",
			pathRouteSetLoadBalancerIDAnnotation,
			statusLoadBalancerID,
			annotationLoadBalancerID,
		)
	}

	identity := pathRouteSetIdentity{
		loadBalancerID:   firstNonEmptyTrim(statusLoadBalancerID, annotationLoadBalancerID),
		pathRouteSetName: firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name, resource.Name),
	}
	if identity.loadBalancerID == "" {
		return pathRouteSetIdentity{}, fmt.Errorf("resolve PathRouteSet identity: %s annotation is required", pathRouteSetLoadBalancerIDAnnotation)
	}
	if identity.pathRouteSetName == "" {
		return pathRouteSetIdentity{}, fmt.Errorf("resolve PathRouteSet identity: path route set name is empty")
	}
	return identity, nil
}

func recordPathRouteSetPathIdentity(resource *loadbalancerv1beta1.PathRouteSet, identity pathRouteSetIdentity) {
	if resource == nil {
		return
	}
	resource.Status.Name = identity.pathRouteSetName
	// PathRouteSet has no child OCID in the Load Balancer API, so the runtime records
	// the parent loadBalancerId as the stable path identity used for Get/Update/Delete.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.loadBalancerID)
}

func recordPathRouteSetTrackedIdentity(resource *loadbalancerv1beta1.PathRouteSet, identity pathRouteSetIdentity) {
	recordPathRouteSetPathIdentity(resource, identity)
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
