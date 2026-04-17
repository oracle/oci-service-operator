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
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type backendLookupClient interface {
	GetBackend(context.Context, loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error)
}

type backendRuntimeDelegate interface {
	CreateOrUpdate(context.Context, *loadbalancerv1beta1.Backend, ctrl.Request) (servicemanager.OSOKResponse, error)
	Delete(context.Context, *loadbalancerv1beta1.Backend) (bool, error)
}

type backendRuntimeServiceClient struct {
	delegate backendRuntimeDelegate
	lookup   backendLookupClient
}

type backendIdentity struct {
	loadBalancerID string
	backendSetName string
	backendName    string
}

func init() {
	newBackendServiceClient = func(manager *BackendServiceManager) BackendServiceClient {
		sdkClient, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
		config := generatedruntime.Config[*loadbalancerv1beta1.Backend]{
			Kind:      "Backend",
			SDKName:   "Backend",
			Log:       manager.Log,
			Semantics: newBackendRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &loadbalancersdk.CreateBackendRequest{} },
				Fields: []generatedruntime.RequestField{
					backendLoadBalancerIDField(),
					backendSetNameField(),
					{FieldName: "CreateBackendDetails", Contribution: "body"},
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.CreateBackend(ctx, *request.(*loadbalancersdk.CreateBackendRequest))
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &loadbalancersdk.GetBackendRequest{} },
				Fields: []generatedruntime.RequestField{
					backendLoadBalancerIDField(),
					backendSetNameField(),
					backendNameField(),
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.GetBackend(ctx, *request.(*loadbalancersdk.GetBackendRequest))
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &loadbalancersdk.ListBackendsRequest{} },
				Fields: []generatedruntime.RequestField{
					backendLoadBalancerIDField(),
					backendSetNameField(),
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.ListBackends(ctx, *request.(*loadbalancersdk.ListBackendsRequest))
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &loadbalancersdk.UpdateBackendRequest{} },
				Fields: []generatedruntime.RequestField{
					backendLoadBalancerIDField(),
					backendSetNameField(),
					backendNameField(),
					{FieldName: "UpdateBackendDetails", Contribution: "body"},
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.UpdateBackend(ctx, *request.(*loadbalancersdk.UpdateBackendRequest))
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &loadbalancersdk.DeleteBackendRequest{} },
				Fields: []generatedruntime.RequestField{
					backendLoadBalancerIDField(),
					backendSetNameField(),
					backendNameField(),
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.DeleteBackend(ctx, *request.(*loadbalancersdk.DeleteBackendRequest))
				},
			},
		}
		if err != nil {
			config.InitError = fmt.Errorf("initialize Backend OCI client: %w", err)
		}

		delegate := defaultBackendServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.Backend](config),
		}
		return &backendRuntimeServiceClient{
			delegate: delegate,
			lookup:   sdkClient,
		}
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

func (c *backendRuntimeServiceClient) CreateOrUpdate(ctx context.Context, resource *loadbalancerv1beta1.Backend, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("backend resource is nil")
	}

	identity, err := resolveBackendIdentity(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	existing, err := c.lookupExistingBackend(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	restore := func() {}
	if existing {
		recordBackendPathIdentity(resource, identity)
		restore = seedSyntheticBackendOCID(resource, identity.backendName)
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	restore()
	if err != nil {
		return response, err
	}

	recordBackendPathIdentity(resource, identity)
	return response, nil
}

func (c *backendRuntimeServiceClient) Delete(ctx context.Context, resource *loadbalancerv1beta1.Backend) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("backend resource is nil")
	}

	identity, err := resolveBackendIdentity(resource)
	if err != nil {
		return false, err
	}

	recordBackendPathIdentity(resource, identity)
	restore := seedSyntheticBackendOCID(resource, identity.backendName)
	deleted, err := c.delegate.Delete(ctx, resource)
	restore()
	if err != nil {
		return false, err
	}

	recordBackendPathIdentity(resource, identity)
	return deleted, nil
}

func (c *backendRuntimeServiceClient) lookupExistingBackend(ctx context.Context, identity backendIdentity) (bool, error) {
	if c.lookup == nil {
		return false, nil
	}

	request := loadbalancersdk.GetBackendRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		BackendSetName: common.String(identity.backendSetName),
		BackendName:    common.String(identity.backendName),
	}
	_, err := c.lookup.GetBackend(ctx, request)
	switch {
	case err == nil:
		return true, nil
	case servicemanager.IsNotFoundServiceError(err):
		return false, nil
	default:
		return false, fmt.Errorf("lookup Backend %q: %w", identity.backendName, err)
	}
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
