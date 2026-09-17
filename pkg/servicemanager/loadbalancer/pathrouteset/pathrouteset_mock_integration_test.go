/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package pathrouteset

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
func TestMockIntegrationPathRouteSetCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loadbalancerv1beta1.PathRouteSet{}
	ocimock.InitializeResource(resource, "mock-pathrouteset")
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.PathRouteSetSpec](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/images",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`)
	resource.Spec.LoadBalancerId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/assets",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreatePathRouteSetDetails](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/images",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.PathRouteSet](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/images",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`)
	createdReadStates := []loadbalancersdk.PathRouteSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.PathRouteSet](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/images",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdatePathRouteSetDetails](t, `{
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/assets",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.PathRouteSet](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/assets",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`)
	updatedReadStates := []loadbalancersdk.PathRouteSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.PathRouteSet](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/assets",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`),
	}
	deletedReadStates := []loadbalancersdk.PathRouteSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.PathRouteSet](t, `{
  "name": "osok_mock_path_routes",
  "pathRoutes": [
    {
      "backendSetName": "osok_mock_backend_set",
      "path": "/assets",
      "pathMatchType": {
        "matchType": "PREFIX_MATCH"
      }
    }
  ]
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loadbalancersdk.PathRouteSet,
		loadbalancersdk.CreatePathRouteSetDetails,
		loadbalancersdk.UpdatePathRouteSetDetails,
	]{
		CollectionPath:     "/20170115/loadBalancers/<ocid:1>/pathRouteSets",
		ItemPath:           "/20170115/loadBalancers/<ocid:1>/pathRouteSets/osok_mock_path_routes",
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
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreatePathRouteSetDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loadbalancersdk.PathRouteSet) error {
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
			t.Errorf("close PathRouteSet OCI mock: %v", err)
		}
	})
	sdkClient := loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()}
	hooks := newPathRouteSetRuntimeHooksWithOCIClient(sdkClient)
	applyPathRouteSetRuntimeHooks(&hooks)
	manager := &PathRouteSetServiceManager{}
	client := wrapPathRouteSetGeneratedClient(hooks, defaultPathRouteSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.PathRouteSet](buildPathRouteSetGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.PathRouteSet]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loadbalancerv1beta1.PathRouteSet) error {
			if !reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.PathRoutes, current.Spec.PathRoutes) {
				return fmt.Errorf("created PathRouteSet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.PathRouteSet) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.PathRouteSet) error {
			if !reflect.DeepEqual(current.Status.PathRoutes, current.Spec.PathRoutes) {
				return fmt.Errorf("updated PathRouteSet status = %+v", current.Status)
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
