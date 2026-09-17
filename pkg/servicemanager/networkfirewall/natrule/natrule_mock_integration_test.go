/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package natrule

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
func TestMockIntegrationNatRuleCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.NatRule{}
	ocimock.InitializeResource(resource, "mock-natrule")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.NatRuleSpec](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_nat_rule",
  "type": "NATV4"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "type": "NATV4"
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateNatV4RuleDetails](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_nat_rule"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NatV4NatRule](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_nat_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "type": "NATV4"
}`)
	createdReadStates := []networkfirewallsdk.NatV4NatRule{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.NatV4NatRule](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_nat_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "type": "NATV4"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateNatV4RuleDetails](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.NatV4NatRule](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_nat_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "type": "NATV4"
}`)
	updatedReadStates := []networkfirewallsdk.NatV4NatRule{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.NatV4NatRule](t, `{
  "action": "DIPP_SRC_NAT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "service": "osok_mock_prereq_tcp",
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "description": null,
  "name": "osok_mock_nat_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "priorityOrder": 1,
  "type": "NATV4"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.NatV4NatRule,
		networkfirewallsdk.CreateNatV4RuleDetails,
		networkfirewallsdk.UpdateNatV4RuleDetails,
	]{
		CollectionPath:     "/20230501/networkFirewallPolicies/<ocid:1>/natRules",
		ItemPath:           "/20230501/networkFirewallPolicies/<ocid:1>/natRules/osok_mock_nat_rule",
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
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "NATV4", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "NATV4", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.NatV4NatRule) error {
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
			t.Errorf("close NatRule OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &NatRuleServiceManager{}
	hooks := newNatRuleRuntimeHooks(manager, sdkClient)
	client := wrapNatRuleGeneratedClient(hooks, defaultNatRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.NatRule](buildNatRuleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.NatRule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.NatRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created NatRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.NatRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.NatRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("updated NatRule status = %+v", current.Status)
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
