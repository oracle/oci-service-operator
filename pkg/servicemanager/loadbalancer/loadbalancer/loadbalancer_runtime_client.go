/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loadbalancer

import (
	"context"
	"fmt"

	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type loadBalancerRuntimeOCIClient interface {
	CreateLoadBalancer(context.Context, loadbalancersdk.CreateLoadBalancerRequest) (loadbalancersdk.CreateLoadBalancerResponse, error)
	GetLoadBalancer(context.Context, loadbalancersdk.GetLoadBalancerRequest) (loadbalancersdk.GetLoadBalancerResponse, error)
	ListLoadBalancers(context.Context, loadbalancersdk.ListLoadBalancersRequest) (loadbalancersdk.ListLoadBalancersResponse, error)
	UpdateLoadBalancer(context.Context, loadbalancersdk.UpdateLoadBalancerRequest) (loadbalancersdk.UpdateLoadBalancerResponse, error)
	DeleteLoadBalancer(context.Context, loadbalancersdk.DeleteLoadBalancerRequest) (loadbalancersdk.DeleteLoadBalancerResponse, error)
}

func init() {
	newLoadBalancerServiceClient = func(manager *LoadBalancerServiceManager) LoadBalancerServiceClient {
		sdkClient, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
		return newGeneratedLoadBalancerServiceClient(sdkClient, manager.Log, manager.CredentialClient, err)
	}
}

func newGeneratedLoadBalancerServiceClient(
	client loadBalancerRuntimeOCIClient,
	log loggerutil.OSOKLogger,
	credentialClient credhelper.CredentialClient,
	initErr error,
) LoadBalancerServiceClient {
	config := generatedruntime.Config[*loadbalancerv1beta1.LoadBalancer]{
		Kind:             "LoadBalancer",
		SDKName:          "LoadBalancer",
		Log:              log,
		CredentialClient: credentialClient,
		InitError:        newLoadBalancerClientInitError(initErr),
		Semantics:        newReviewedLoadBalancerRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &loadbalancersdk.CreateLoadBalancerRequest{} },
			Fields:     loadBalancerCreateFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateLoadBalancer(ctx, *request.(*loadbalancersdk.CreateLoadBalancerRequest))
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &loadbalancersdk.GetLoadBalancerRequest{} },
			Fields:     loadBalancerGetFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetLoadBalancer(ctx, *request.(*loadbalancersdk.GetLoadBalancerRequest))
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &loadbalancersdk.ListLoadBalancersRequest{} },
			Fields:     loadBalancerListFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListLoadBalancers(ctx, *request.(*loadbalancersdk.ListLoadBalancersRequest))
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &loadbalancersdk.UpdateLoadBalancerRequest{} },
			Fields:     loadBalancerUpdateFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateLoadBalancer(ctx, *request.(*loadbalancersdk.UpdateLoadBalancerRequest))
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &loadbalancersdk.DeleteLoadBalancerRequest{} },
			Fields:     loadBalancerDeleteFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteLoadBalancer(ctx, *request.(*loadbalancersdk.DeleteLoadBalancerRequest))
			},
		},
	}

	return defaultLoadBalancerServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.LoadBalancer](config),
	}
}

func newReviewedLoadBalancerRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "loadbalancer",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
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
			MatchFields:        []string{"displayName", "compartmentId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"displayName",
				"freeformTags",
			},
			ForceNew: []string{
				"backendSets",
				"certificates",
				"compartmentId",
				"hostnames",
				"ipMode",
				"isPrivate",
				"listeners",
				"networkSecurityGroupIds",
				"pathRouteSets",
				"reservedIps",
				"ruleSets",
				"shapeDetails",
				"shapeName",
				"sslCipherSuites",
				"subnetIds",
			},
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

func loadBalancerCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CreateLoadBalancerDetails",
			RequestName:  "CreateLoadBalancerDetails",
			Contribution: "body",
		},
	}
}

func loadBalancerGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		loadBalancerIDField(),
	}
}

func loadBalancerListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		loadBalancerCompartmentIDField(),
		loadBalancerDisplayNameField(),
	}
}

func loadBalancerUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "UpdateLoadBalancerDetails",
			RequestName:  "UpdateLoadBalancerDetails",
			Contribution: "body",
		},
		loadBalancerIDField(),
	}
}

func loadBalancerDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		loadBalancerIDField(),
	}
}

func loadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "LoadBalancerId",
		RequestName:      "loadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.id", "status.status.ocid"},
	}
}

func loadBalancerCompartmentIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "CompartmentId",
		RequestName:  "compartmentId",
		Contribution: "query",
		LookupPaths:  []string{"status.compartmentId", "spec.compartmentId"},
	}
}

func loadBalancerDisplayNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "DisplayName",
		RequestName:  "displayName",
		Contribution: "query",
		LookupPaths:  []string{"status.displayName", "spec.displayName"},
	}
}

func newLoadBalancerClientInitError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("initialize LoadBalancer OCI client: %w", err)
}
