/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package addresslist

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
func TestMockIntegrationAddressListCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.AddressList{}
	ocimock.InitializeResource(resource, "mock-addresslist")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.AddressListSpec](t, `{
  "addresses": [
    "10.20.0.0/24"
  ],
  "description": "OSOK recorded address list",
  "name": "osok_mock_address_list",
  "type": "IP"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "addresses": [
    "10.20.0.0/24",
    "10.21.0.0/24"
  ],
  "description": "OSOK recorded address list updated",
  "type": "IP"
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateAddressListDetails](t, `{
  "addresses": [
    "10.20.0.0/24"
  ],
  "description": "OSOK recorded address list",
  "name": "osok_mock_address_list",
  "type": "IP"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.AddressList](t, `{
  "addresses": [
    "10.20.0.0/24"
  ],
  "description": "OSOK recorded address list",
  "name": "osok_mock_address_list",
  "parentResourceId": "<ocid:1>",
  "totalAddresses": 1,
  "type": "IP"
}`)
	createdReadStates := []networkfirewallsdk.AddressList{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.AddressList](t, `{
  "addresses": [
    "10.20.0.0/24"
  ],
  "description": "OSOK recorded address list",
  "name": "osok_mock_address_list",
  "parentResourceId": "<ocid:1>",
  "totalAddresses": 1,
  "type": "IP"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateIpAddressListDetails](t, `{
  "addresses": [
    "10.20.0.0/24",
    "10.21.0.0/24"
  ],
  "description": "OSOK recorded address list updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.AddressList](t, `{
  "addresses": [
    "10.20.0.0/24",
    "10.21.0.0/24"
  ],
  "description": "OSOK recorded address list updated",
  "name": "osok_mock_address_list",
  "parentResourceId": "<ocid:1>",
  "totalAddresses": 2,
  "type": "IP"
}`)
	updatedReadStates := []networkfirewallsdk.AddressList{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.AddressList](t, `{
  "addresses": [
    "10.20.0.0/24",
    "10.21.0.0/24"
  ],
  "description": "OSOK recorded address list updated",
  "name": "osok_mock_address_list",
  "parentResourceId": "<ocid:1>",
  "totalAddresses": 2,
  "type": "IP"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.AddressList,
		networkfirewallsdk.CreateAddressListDetails,
		networkfirewallsdk.UpdateIpAddressListDetails,
	]{
		CollectionPath:     "/20230501/networkFirewallPolicies/<ocid:1>/addressLists",
		ItemPath:           "/20230501/networkFirewallPolicies/<ocid:1>/addressLists/osok_mock_address_list",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
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
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateAddressListDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "IP", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.AddressList) error {
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
			t.Errorf("close AddressList OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &AddressListServiceManager{}
	hooks := newAddressListRuntimeHooks(manager, sdkClient)
	client := wrapAddressListGeneratedClient(hooks, defaultAddressListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.AddressList](buildAddressListGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.AddressList]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.AddressList) error {
			if !reflect.DeepEqual(current.Status.Addresses, current.Spec.Addresses) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created AddressList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.AddressList) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.AddressList) error {
			if !reflect.DeepEqual(current.Status.Addresses, current.Spec.Addresses) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("updated AddressList status = %+v", current.Status)
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
