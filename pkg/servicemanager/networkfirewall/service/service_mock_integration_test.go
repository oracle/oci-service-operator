/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package service

import (
	"context"
	"fmt"
	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationServiceCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.Service{}
	ocimock.InitializeResource(resource, "mock-service")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.ServiceSpec](t, `{
  "description": "OSOK recorded service",
  "name": "osok_mock_service",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ],
  "type": "TCP_SERVICE"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded Service updated",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ],
  "type": "TCP_SERVICE"
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateTcpServiceDetails](t, `{
  "description": "OSOK recorded service",
  "name": "osok_mock_service",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.TcpService](t, `{
  "description": "OSOK recorded service",
  "name": "osok_mock_service",
  "parentResourceId": "<ocid:1>",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ],
  "type": "TCP_SERVICE"
}`)
	createdReadStates := []networkfirewallsdk.TcpService{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.TcpService](t, `{
  "description": "OSOK recorded service",
  "name": "osok_mock_service",
  "parentResourceId": "<ocid:1>",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ],
  "type": "TCP_SERVICE"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateTcpServiceDetails](t, `{
  "description": "OSOK recorded Service updated",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.TcpService](t, `{
  "description": "OSOK recorded Service updated",
  "name": "osok_mock_service",
  "parentResourceId": "<ocid:1>",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ],
  "type": "TCP_SERVICE"
}`)
	updatedReadStates := []networkfirewallsdk.TcpService{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.TcpService](t, `{
  "description": "OSOK recorded Service updated",
  "name": "osok_mock_service",
  "parentResourceId": "<ocid:1>",
  "portRanges": [
    {
      "maximumPort": 8080,
      "minimumPort": 8080
    }
  ],
  "type": "TCP_SERVICE"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.TcpService,
		networkfirewallsdk.CreateTcpServiceDetails,
		networkfirewallsdk.UpdateTcpServiceDetails,
	]{
		CollectionPath:     "/20230501/networkFirewallPolicies/<ocid:1>/services",
		ItemPath:           "/20230501/networkFirewallPolicies/<ocid:1>/services/osok_mock_service",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:       &createdState,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "TCP_SERVICE", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "TCP_SERVICE", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.TcpService) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20230501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Service OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &ServiceServiceManager{}
	hooks := newServiceRuntimeHooks(manager, sdkClient)
	client := wrapServiceGeneratedClient(hooks, defaultServiceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.Service](buildServiceGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.Service]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.Service) error {
			if !reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.PortRanges, current.Spec.PortRanges) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created Service status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.Service) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.Service) error {
			if !reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.PortRanges, current.Spec.PortRanges) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("updated Service status = %+v", current.Status)
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
