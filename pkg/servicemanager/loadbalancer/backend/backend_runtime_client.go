/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type backendRuntimeOCIClient interface {
	CreateBackend(context.Context, loadbalancersdk.CreateBackendRequest) (loadbalancersdk.CreateBackendResponse, error)
	GetBackend(context.Context, loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error)
	ListBackends(context.Context, loadbalancersdk.ListBackendsRequest) (loadbalancersdk.ListBackendsResponse, error)
	UpdateBackend(context.Context, loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error)
	DeleteBackend(context.Context, loadbalancersdk.DeleteBackendRequest) (loadbalancersdk.DeleteBackendResponse, error)
}

type backendIdentity struct {
	loadBalancerID string
	backendSetName string
	backendName    string
}

func init() {
	registerBackendRuntimeHooksMutator(func(_ *BackendServiceManager, hooks *BackendRuntimeHooks) {
		applyBackendRuntimeHooks(hooks)
	})
}

func applyBackendRuntimeHooks(hooks *BackendRuntimeHooks) {
	if hooks == nil {
		return
	}

	getCall := hooks.Get.Call
	hooks.Semantics = newBackendRuntimeSemantics()
	hooks.Identity = generatedruntime.IdentityHooks[*loadbalancerv1beta1.Backend]{
		Resolve: func(resource *loadbalancerv1beta1.Backend) (any, error) {
			return resolveBackendIdentity(resource)
		},
		RecordPath: func(resource *loadbalancerv1beta1.Backend, identity any) {
			recordBackendPathIdentity(resource, identity.(backendIdentity))
		},
		LookupExisting: func(ctx context.Context, _ *loadbalancerv1beta1.Backend, identity any) (any, error) {
			return lookupExistingBackend(ctx, getCall, identity.(backendIdentity))
		},
		SeedSyntheticTrackedID: func(resource *loadbalancerv1beta1.Backend, identity any) func() {
			return seedSyntheticBackendOCID(resource, identity.(backendIdentity).backendName)
		},
	}
	hooks.Create.Fields = []generatedruntime.RequestField{
		backendLoadBalancerIDField(),
		backendSetNameField(),
		{FieldName: "CreateBackendDetails", Contribution: "body"},
	}
	hooks.Get.Fields = []generatedruntime.RequestField{
		backendLoadBalancerIDField(),
		backendSetNameField(),
		backendNameField(),
	}
	hooks.List.Fields = []generatedruntime.RequestField{
		backendLoadBalancerIDField(),
		backendSetNameField(),
	}
	hooks.Update.Fields = []generatedruntime.RequestField{
		backendLoadBalancerIDField(),
		backendSetNameField(),
		backendNameField(),
		{FieldName: "UpdateBackendDetails", Contribution: "body"},
	}
	hooks.Delete.Fields = []generatedruntime.RequestField{
		backendLoadBalancerIDField(),
		backendSetNameField(),
		backendNameField(),
	}
}

func newBackendRuntimeHooksWithOCIClient(client backendRuntimeOCIClient) BackendRuntimeHooks {
	return BackendRuntimeHooks{
		Semantics: newBackendRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*loadbalancerv1beta1.Backend]{},
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[loadbalancersdk.CreateBackendRequest, loadbalancersdk.CreateBackendResponse]{
			Fields: []generatedruntime.RequestField{
				backendLoadBalancerIDField(),
				backendSetNameField(),
				{FieldName: "CreateBackendDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request loadbalancersdk.CreateBackendRequest) (loadbalancersdk.CreateBackendResponse, error) {
				return client.CreateBackend(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loadbalancersdk.GetBackendRequest, loadbalancersdk.GetBackendResponse]{
			Fields: []generatedruntime.RequestField{
				backendLoadBalancerIDField(),
				backendSetNameField(),
				backendNameField(),
			},
			Call: func(ctx context.Context, request loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
				return client.GetBackend(ctx, request)
			},
		},
		List: runtimeOperationHooks[loadbalancersdk.ListBackendsRequest, loadbalancersdk.ListBackendsResponse]{
			Fields: []generatedruntime.RequestField{
				backendLoadBalancerIDField(),
				backendSetNameField(),
			},
			Call: func(ctx context.Context, request loadbalancersdk.ListBackendsRequest) (loadbalancersdk.ListBackendsResponse, error) {
				return client.ListBackends(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loadbalancersdk.UpdateBackendRequest, loadbalancersdk.UpdateBackendResponse]{
			Fields: []generatedruntime.RequestField{
				backendLoadBalancerIDField(),
				backendSetNameField(),
				backendNameField(),
				{FieldName: "UpdateBackendDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error) {
				return client.UpdateBackend(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loadbalancersdk.DeleteBackendRequest, loadbalancersdk.DeleteBackendResponse]{
			Fields: []generatedruntime.RequestField{
				backendLoadBalancerIDField(),
				backendSetNameField(),
				backendNameField(),
			},
			Call: func(ctx context.Context, request loadbalancersdk.DeleteBackendRequest) (loadbalancersdk.DeleteBackendResponse, error) {
				return client.DeleteBackend(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BackendServiceClient) BackendServiceClient{},
	}
}

func backendLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "LoadBalancerId",
		RequestName:  "loadBalancerId",
		Contribution: "path",
		LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
	}
}

func backendSetNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "BackendSetName",
		RequestName:  "backendSetName",
		Contribution: "path",
		LookupPaths:  []string{"status.backendSetName", "spec.backendSetName"},
	}
}

func backendNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "BackendName",
		RequestName:      "backendName",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.name"},
	}
}

func lookupExistingBackend(
	ctx context.Context,
	getCall func(context.Context, loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error),
	identity backendIdentity,
) (any, error) {
	if getCall == nil {
		return nil, nil
	}

	return getCall(ctx, loadbalancersdk.GetBackendRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		BackendSetName: common.String(identity.backendSetName),
		BackendName:    common.String(identity.backendName),
	})
}

func resolveBackendIdentity(resource *loadbalancerv1beta1.Backend) (backendIdentity, error) {
	identity := backendIdentity{
		loadBalancerID: firstNonEmptyTrim(resource.Status.LoadBalancerId, resource.Spec.LoadBalancerId),
		backendSetName: firstNonEmptyTrim(resource.Status.BackendSetName, resource.Spec.BackendSetName),
		backendName:    currentBackendName(resource),
	}
	if identity.loadBalancerID == "" {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: loadBalancerId is empty")
	}
	if identity.backendSetName == "" {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: backendSetName is empty")
	}
	if identity.backendName == "" {
		return backendIdentity{}, fmt.Errorf("resolve Backend identity: backend name is empty")
	}
	return identity, nil
}

func currentBackendName(resource *loadbalancerv1beta1.Backend) string {
	if resource == nil {
		return ""
	}
	if name := strings.TrimSpace(resource.Status.Name); name != "" {
		return name
	}

	ipAddress := strings.TrimSpace(resource.Spec.IpAddress)
	if ipAddress == "" || resource.Spec.Port == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", ipAddress, resource.Spec.Port)
}

func recordBackendPathIdentity(resource *loadbalancerv1beta1.Backend, identity backendIdentity) {
	if resource == nil {
		return
	}
	resource.Status.LoadBalancerId = identity.loadBalancerID
	resource.Status.BackendSetName = identity.backendSetName
}

func seedSyntheticBackendOCID(resource *loadbalancerv1beta1.Backend, backendName string) func() {
	if resource == nil || strings.TrimSpace(backendName) == "" {
		return func() {}
	}

	previous := resource.Status.OsokStatus.Ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(backendName)
	return func() {
		resource.Status.OsokStatus.Ocid = previous
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
