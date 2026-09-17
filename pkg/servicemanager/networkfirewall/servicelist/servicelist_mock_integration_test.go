/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package servicelist

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
func TestMockIntegrationServiceListCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.ServiceList{}
	ocimock.InitializeResource(resource, "mock-servicelist")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.ServiceListSpec](t, `{
  "name": "osok_mock_service_list",
  "services": [
    "osok_mock_prereq_tcp"
  ]
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "services": [
    "osok_mock_prereq_tcp",
    "osok_mock_udp2"
  ]
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateServiceListDetails](t, `{
  "name": "osok_mock_service_list",
  "services": [
    "osok_mock_prereq_tcp"
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.ServiceList](t, `{
  "description": null,
  "name": "osok_mock_service_list",
  "parentResourceId": "<ocid:1>",
  "services": [
    "osok_mock_prereq_tcp"
  ],
  "totalServices": 1
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateServiceListDetails](t, `{
  "services": [
    "osok_mock_prereq_tcp",
    "osok_mock_udp2"
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.ServiceList](t, `{
  "description": null,
  "name": "osok_mock_service_list",
  "parentResourceId": "<ocid:1>",
  "services": [
    "osok_mock_prereq_tcp",
    "osok_mock_udp2"
  ],
  "totalServices": 2
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.ServiceList,
		networkfirewallsdk.CreateServiceListDetails,
		networkfirewallsdk.UpdateServiceListDetails,
	]{
		CollectionPath:    "/20230501/networkFirewallPolicies/<ocid:1>/serviceLists",
		ItemPath:          "/20230501/networkFirewallPolicies/<ocid:1>/serviceLists/osok_mock_service_list",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateServiceListDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.ServiceList) error {
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
			t.Errorf("close ServiceList OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &ServiceListServiceManager{}
	hooks := newServiceListRuntimeHooks(manager, sdkClient)
	client := wrapServiceListGeneratedClient(hooks, defaultServiceListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.ServiceList](buildServiceListGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.ServiceList]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.ServiceList) error {
			if !reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Services, current.Spec.Services) {
				return fmt.Errorf("created ServiceList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.ServiceList) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.ServiceList) error {
			if !reflect.DeepEqual(current.Status.Services, current.Spec.Services) {
				return fmt.Errorf("updated ServiceList status = %+v", current.Status)
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
