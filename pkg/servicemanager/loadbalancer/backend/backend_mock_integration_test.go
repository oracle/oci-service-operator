/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package backend

import (
	"context"
	"fmt"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationBackendCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loadbalancerv1beta1.Backend{}
	ocimock.InitializeResource(resource, "mock-backend")
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.BackendSpec](t, `{
  "ipAddress": "10.0.20.201",
  "port": 8081,
  "weight": 1
}`)
	resource.Spec.LoadBalancerId = "<ocid:1>"
	resource.Spec.BackendSetName = "osok_mock_backend_set"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "backup": false,
  "drain": false,
  "offline": false,
  "weight": 2
}`)
	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreateBackendDetails](t, `{
  "ipAddress": "10.0.20.201",
  "port": 8081,
  "weight": 1
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.Backend](t, `{
  "backup": false,
  "drain": false,
  "ipAddress": "10.0.20.201",
  "maxConnections": null,
  "name": "10.0.20.201:8081",
  "offline": false,
  "port": 8081,
  "weight": 1
}`)
	createdReadStates := []loadbalancersdk.Backend{
		ocimock.MustOCIResponseFixture[loadbalancersdk.Backend](t, `{
  "backup": false,
  "drain": false,
  "ipAddress": "10.0.20.201",
  "maxConnections": null,
  "name": "10.0.20.201:8081",
  "offline": false,
  "port": 8081,
  "weight": 1
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdateBackendDetails](t, `{
  "backup": false,
  "drain": false,
  "offline": false,
  "weight": 2
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.Backend](t, `{
  "backup": false,
  "drain": false,
  "ipAddress": "10.0.20.201",
  "maxConnections": null,
  "name": "10.0.20.201:8081",
  "offline": false,
  "port": 8081,
  "weight": 2
}`)
	updatedReadStates := []loadbalancersdk.Backend{
		ocimock.MustOCIResponseFixture[loadbalancersdk.Backend](t, `{
  "backup": false,
  "drain": false,
  "ipAddress": "10.0.20.201",
  "maxConnections": null,
  "name": "10.0.20.201:8081",
  "offline": false,
  "port": 8081,
  "weight": 2
}`),
	}
	deletedReadStates := []loadbalancersdk.Backend{
		ocimock.MustOCIResponseFixture[loadbalancersdk.Backend](t, `{
  "backup": false,
  "drain": false,
  "ipAddress": "10.0.20.201",
  "maxConnections": null,
  "name": "10.0.20.201:8081",
  "offline": false,
  "port": 8081,
  "weight": 2
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loadbalancersdk.Backend,
		loadbalancersdk.CreateBackendDetails,
		loadbalancersdk.UpdateBackendDetails,
	]{
		CollectionPath:     "/20170115/loadBalancers/<ocid:1>/backendSets/osok_mock_backend_set/backends",
		ItemPath:           "/20170115/loadBalancers/<ocid:1>/backendSets/osok_mock_backend_set/backends/10.0.20.201:8081",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeArray,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeletedReadStates:  deletedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       204,
		UpdateStatus:       204,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreateBackendDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loadbalancersdk.Backend) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20170115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Backend OCI mock: %v", err)
		}
	})
	sdkClient := loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()}
	hooks := newBackendRuntimeHooksWithOCIClient(sdkClient)
	applyBackendRuntimeHooks(&hooks)
	manager := &BackendServiceManager{}
	client := wrapBackendGeneratedClient(hooks, defaultBackendServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.Backend](buildBackendGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.Backend]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loadbalancerv1beta1.Backend) error {
			if !reflect.DeepEqual(current.Status.IpAddress, current.Spec.IpAddress) ||
				!reflect.DeepEqual(current.Status.Port, current.Spec.Port) ||
				!reflect.DeepEqual(current.Status.Weight, current.Spec.Weight) {
				return fmt.Errorf("created Backend status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.Backend) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.Backend) error {
			if !reflect.DeepEqual(current.Status.Backup, current.Spec.Backup) ||
				!reflect.DeepEqual(current.Status.Drain, current.Spec.Drain) ||
				!reflect.DeepEqual(current.Status.Offline, current.Spec.Offline) ||
				!reflect.DeepEqual(current.Status.Weight, current.Spec.Weight) {
				return fmt.Errorf("updated Backend status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
