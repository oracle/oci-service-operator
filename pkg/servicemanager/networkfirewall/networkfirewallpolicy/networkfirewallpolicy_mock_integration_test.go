/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package networkfirewallpolicy

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
func TestMockIntegrationNetworkFirewallPolicyEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.NetworkFirewallPolicy{}
	ocimock.InitializeResource(resource, "mock-networkfirewallpolicy")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.NetworkFirewallPolicySpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK recorded Network Firewall policy",
  "displayName": "osok-mock-network-firewall-policy",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded Network Firewall policy updated",
  "displayName": "osok-mock-network-firewall-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateNetworkFirewallPolicyDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK recorded Network Firewall policy",
  "displayName": "osok-mock-network-firewall-policy",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewallPolicy](t, `{
  "attachedNetworkFirewallCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T16:22:37.514Z"
    }
  },
  "description": "OSOK recorded Network Firewall policy",
  "displayName": "osok-mock-network-firewall-policy",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T16:22:37.691Z",
  "timeUpdated": "2026-09-02T16:22:37.691Z"
}`)
	createdReadStates := []networkfirewallsdk.NetworkFirewallPolicy{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewallPolicy](t, `{
  "attachedNetworkFirewallCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T16:22:37.514Z"
    }
  },
  "description": "OSOK recorded Network Firewall policy",
  "displayName": "osok-mock-network-firewall-policy",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T16:22:37.691Z",
  "timeUpdated": "2026-09-02T16:22:37.691Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateNetworkFirewallPolicyDetails](t, `{
  "description": "OSOK recorded Network Firewall policy updated",
  "displayName": "osok-mock-network-firewall-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewallPolicy](t, `{
  "attachedNetworkFirewallCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T16:22:37.514Z"
    }
  },
  "description": "OSOK recorded Network Firewall policy updated",
  "displayName": "osok-mock-network-firewall-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T16:22:37.691Z",
  "timeUpdated": "2026-09-02T16:22:38.611Z"
}`)
	updatedReadStates := []networkfirewallsdk.NetworkFirewallPolicy{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewallPolicy](t, `{
  "attachedNetworkFirewallCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T16:22:37.514Z"
    }
  },
  "description": "OSOK recorded Network Firewall policy updated",
  "displayName": "osok-mock-network-firewall-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T16:22:37.691Z",
  "timeUpdated": "2026-09-02T16:22:38.611Z"
}`),
	}
	deletedReadStates := []networkfirewallsdk.NetworkFirewallPolicy{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewallPolicy](t, `{
  "attachedNetworkFirewallCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T16:22:37.514Z"
    }
  },
  "description": "OSOK recorded Network Firewall policy updated",
  "displayName": "osok-mock-network-firewall-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETING",
  "systemTags": {},
  "timeCreated": "2026-09-02T16:22:37.691Z",
  "timeUpdated": "2026-09-02T16:22:38.611Z"
}`),
		ocimock.MustOCIResponseFixture[networkfirewallsdk.NetworkFirewallPolicy](t, `{
  "attachedNetworkFirewallCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T16:22:37.514Z"
    }
  },
  "description": "OSOK recorded Network Firewall policy updated",
  "displayName": "osok-mock-network-firewall-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-02T16:22:37.691Z",
  "timeUpdated": "2026-09-02T16:23:06.405Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.NetworkFirewallPolicy,
		networkfirewallsdk.CreateNetworkFirewallPolicyDetails,
		networkfirewallsdk.UpdateNetworkFirewallPolicyDetails,
	]{
		CollectionPath:    "/20230501/networkFirewallPolicies",
		ItemPath:          "/20230501/networkFirewallPolicies/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      202,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateNetworkFirewallPolicyDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.NetworkFirewallPolicy) error {
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
			t.Errorf("close NetworkFirewallPolicy OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &NetworkFirewallPolicyServiceManager{}
	hooks := newNetworkFirewallPolicyRuntimeHooks(manager, sdkClient)
	client := wrapNetworkFirewallPolicyGeneratedClient(hooks, defaultNetworkFirewallPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.NetworkFirewallPolicy](buildNetworkFirewallPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.NetworkFirewallPolicy]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.NetworkFirewallPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created NetworkFirewallPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.NetworkFirewallPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.NetworkFirewallPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated NetworkFirewallPolicy status = %+v", current.Status)
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
