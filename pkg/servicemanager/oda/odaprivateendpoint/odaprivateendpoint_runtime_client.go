/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odaprivateendpoint

import (
	"context"
	"strings"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type odaPrivateEndpointOCIClient interface {
	CreateOdaPrivateEndpoint(context.Context, odasdk.CreateOdaPrivateEndpointRequest) (odasdk.CreateOdaPrivateEndpointResponse, error)
	GetOdaPrivateEndpoint(context.Context, odasdk.GetOdaPrivateEndpointRequest) (odasdk.GetOdaPrivateEndpointResponse, error)
	ListOdaPrivateEndpoints(context.Context, odasdk.ListOdaPrivateEndpointsRequest) (odasdk.ListOdaPrivateEndpointsResponse, error)
	UpdateOdaPrivateEndpoint(context.Context, odasdk.UpdateOdaPrivateEndpointRequest) (odasdk.UpdateOdaPrivateEndpointResponse, error)
	DeleteOdaPrivateEndpoint(context.Context, odasdk.DeleteOdaPrivateEndpointRequest) (odasdk.DeleteOdaPrivateEndpointResponse, error)
}

func init() {
	registerOdaPrivateEndpointRuntimeHooksMutator(func(_ *OdaPrivateEndpointServiceManager, hooks *OdaPrivateEndpointRuntimeHooks) {
		applyOdaPrivateEndpointRuntimeHooks(hooks)
	})
}

func applyOdaPrivateEndpointRuntimeHooks(hooks *OdaPrivateEndpointRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOdaPrivateEndpointRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardOdaPrivateEndpointExistingBeforeCreate
}

func newOdaPrivateEndpointRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "odaprivateendpoint",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.OdaPrivateEndpointLifecycleStateCreating)},
			UpdatingStates:     []string{string(odasdk.OdaPrivateEndpointLifecycleStateUpdating)},
			ActiveStates:       []string{string(odasdk.OdaPrivateEndpointLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.OdaPrivateEndpointLifecycleStateDeleting)},
			TerminalStates: []string{string(odasdk.OdaPrivateEndpointLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"displayName", "compartmentId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"nsgIds",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId", "subnetId"},
			ConflictsWith: map[string][]string{},
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

func guardOdaPrivateEndpointExistingBeforeCreate(_ context.Context, resource *odav1beta1.OdaPrivateEndpoint) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func newOdaPrivateEndpointServiceClientWithOCIClient(client odaPrivateEndpointOCIClient) OdaPrivateEndpointServiceClient {
	hooks := newOdaPrivateEndpointRuntimeHooksWithOCIClient(client)
	applyOdaPrivateEndpointRuntimeHooks(&hooks)
	delegate := defaultOdaPrivateEndpointServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.OdaPrivateEndpoint](
			buildOdaPrivateEndpointGeneratedRuntimeConfig(&OdaPrivateEndpointServiceManager{}, hooks),
		),
	}
	return wrapOdaPrivateEndpointGeneratedClient(hooks, delegate)
}

func newOdaPrivateEndpointRuntimeHooksWithOCIClient(client odaPrivateEndpointOCIClient) OdaPrivateEndpointRuntimeHooks {
	hooks := newOdaPrivateEndpointDefaultRuntimeHooks(odasdk.ManagementClient{})
	if client == nil {
		return hooks
	}

	hooks.Create.Call = func(ctx context.Context, request odasdk.CreateOdaPrivateEndpointRequest) (odasdk.CreateOdaPrivateEndpointResponse, error) {
		return client.CreateOdaPrivateEndpoint(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request odasdk.GetOdaPrivateEndpointRequest) (odasdk.GetOdaPrivateEndpointResponse, error) {
		return client.GetOdaPrivateEndpoint(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request odasdk.ListOdaPrivateEndpointsRequest) (odasdk.ListOdaPrivateEndpointsResponse, error) {
		return client.ListOdaPrivateEndpoints(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request odasdk.UpdateOdaPrivateEndpointRequest) (odasdk.UpdateOdaPrivateEndpointResponse, error) {
		return client.UpdateOdaPrivateEndpoint(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request odasdk.DeleteOdaPrivateEndpointRequest) (odasdk.DeleteOdaPrivateEndpointResponse, error) {
		return client.DeleteOdaPrivateEndpoint(ctx, request)
	}
	return hooks
}
