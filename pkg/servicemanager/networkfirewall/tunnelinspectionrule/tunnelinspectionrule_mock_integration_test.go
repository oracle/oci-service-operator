/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package tunnelinspectionrule

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
func TestMockIntegrationTunnelInspectionRuleCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.TunnelInspectionRule{}
	ocimock.InitializeResource(resource, "mock-tunnelinspectionrule")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.TunnelInspectionRuleSpec](t, `{
  "action": "INSPECT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_tunnel_rule",
  "profile": {
    "mustReturnTrafficToSource": true
  },
  "protocol": "VXLAN"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "action": "INSPECT_AND_CAPTURE_LOG",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "profile": {
    "mustReturnTrafficToSource": true
  },
  "protocol": "VXLAN"
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateVxlanInspectionRuleDetails](t, `{
  "action": "INSPECT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_tunnel_rule",
  "profile": {
    "mustReturnTrafficToSource": true
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.VxlanInspectionRule](t, `{
  "action": "INSPECT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_tunnel_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "profile": {
    "mustReturnTrafficToSource": true
  },
  "protocol": "VXLAN"
}`)
	createdReadStates := []networkfirewallsdk.VxlanInspectionRule{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.VxlanInspectionRule](t, `{
  "action": "INSPECT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_tunnel_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "profile": {
    "mustReturnTrafficToSource": true
  },
  "protocol": "VXLAN"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateVxlanInspectionRuleDetails](t, `{
  "action": "INSPECT_AND_CAPTURE_LOG",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "profile": {
    "mustReturnTrafficToSource": true
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.VxlanInspectionRule](t, `{
  "action": "INSPECT_AND_CAPTURE_LOG",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_tunnel_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "profile": {
    "mustReturnTrafficToSource": true
  },
  "protocol": "VXLAN"
}`)
	updatedReadStates := []networkfirewallsdk.VxlanInspectionRule{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.VxlanInspectionRule](t, `{
  "action": "INSPECT_AND_CAPTURE_LOG",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_tunnel_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "profile": {
    "mustReturnTrafficToSource": true
  },
  "protocol": "VXLAN"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.VxlanInspectionRule,
		networkfirewallsdk.CreateVxlanInspectionRuleDetails,
		networkfirewallsdk.UpdateVxlanInspectionRuleDetails,
	]{
		CollectionPath:     "/20230501/networkFirewallPolicies/<ocid:1>/tunnelInspectionRules",
		ItemPath:           "/20230501/networkFirewallPolicies/<ocid:1>/tunnelInspectionRules/osok_mock_tunnel_rule",
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
			return ocimock.ValidateDiscriminatedJSONRequest(request, "protocol", "VXLAN", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "protocol", "VXLAN", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.VxlanInspectionRule) error {
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
			t.Errorf("close TunnelInspectionRule OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &TunnelInspectionRuleServiceManager{}
	hooks := newTunnelInspectionRuleRuntimeHooks(manager, sdkClient)
	client := wrapTunnelInspectionRuleGeneratedClient(hooks, defaultTunnelInspectionRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.TunnelInspectionRule](buildTunnelInspectionRuleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.TunnelInspectionRule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.TunnelInspectionRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Profile, current.Spec.Profile) ||
				!reflect.DeepEqual(current.Status.Protocol, current.Spec.Protocol) {
				return fmt.Errorf("created TunnelInspectionRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.TunnelInspectionRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.TunnelInspectionRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) ||
				!reflect.DeepEqual(current.Status.Profile, current.Spec.Profile) ||
				!reflect.DeepEqual(current.Status.Protocol, current.Spec.Protocol) {
				return fmt.Errorf("updated TunnelInspectionRule status = %+v", current.Status)
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
