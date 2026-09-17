/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package networkfirewall

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
func TestMockIntegrationNetworkFirewallLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.NetworkFirewall{Spec: networkfirewallv1beta1.NetworkFirewallSpec{CompartmentId: "ocid1.compartment.oc1..mock", SubnetId: "ocid1.subnet.oc1..mock", NetworkFirewallPolicyId: "ocid1.networkfirewallpolicy.oc1..mock", DisplayName: "osok-mock-network-firewall", Shape: "NETWORK_FIREWALL_SMALL", FreeformTags: map[string]string{"osok-mock": "create"}}}
	ocimock.InitializeResource(resource, "mock-networkfirewall")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.NetworkFirewallSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-network-firewall",
  "freeformTags": {
    "osok-mock": "create"
  },
  "networkFirewallPolicyId": "\u003cocid:2\u003e",
  "shape": "NETWORK_FIREWALL_SMALL",
  "subnetId": "\u003cocid:3\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-network-firewall-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateNetworkFirewallDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-network-firewall",
  "freeformTags": {
    "osok-mock": "create"
  },
  "networkFirewallPolicyId": "\u003cocid:2\u003e",
  "shape": "NETWORK_FIREWALL_SMALL",
  "subnetId": "\u003cocid:3\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewall](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-network-firewall",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:5>",
  "lifecycleState": "ACTIVE",
  "networkFirewallPolicyId": "<ocid:2>",
  "shape": "NETWORK_FIREWALL_SMALL",
  "subnetId": "<ocid:3>"
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateNetworkFirewallDetails](t, `{
  "displayName": "osok-mock-network-firewall-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewall](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-network-firewall-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "lifecycleState": "ACTIVE",
  "networkFirewallPolicyId": "<ocid:2>",
  "shape": "NETWORK_FIREWALL_SMALL",
  "subnetId": "<ocid:3>"
}`)
	deletedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewall](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-network-firewall-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "lifecycleState": "DELETED",
  "networkFirewallPolicyId": "<ocid:2>",
  "shape": "NETWORK_FIREWALL_SMALL",
  "subnetId": "<ocid:3>"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.NetworkFirewall,
		networkfirewallsdk.CreateNetworkFirewallDetails,
		networkfirewallsdk.UpdateNetworkFirewallDetails,
	]{
		CollectionPath:    "/20230501/networkFirewalls",
		ItemPath:          "/20230501/networkFirewalls/<ocid:5>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      202,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateNetworkFirewallDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.NetworkFirewall) error {
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
			t.Errorf("close NetworkFirewall OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.
		NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &NetworkFirewallServiceManager{}
	hooks := newNetworkFirewallRuntimeHooks(manager, sdkClient)
	client := wrapNetworkFirewallGeneratedClient(hooks, defaultNetworkFirewallServiceClient{ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.NetworkFirewall](buildNetworkFirewallGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.NetworkFirewall]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.NetworkFirewall) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.NetworkFirewallPolicyId, current.Spec.NetworkFirewallPolicyId) ||
				!reflect.DeepEqual(current.Status.Shape, current.Spec.Shape) ||
				!reflect.DeepEqual(current.Status.SubnetId, current.Spec.SubnetId) {
				return fmt.Errorf("created NetworkFirewall status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.NetworkFirewall) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.NetworkFirewall) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated NetworkFirewall status = %+v", current.Status)
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
