/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package decryptionrule

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
func TestMockIntegrationDecryptionRuleCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.DecryptionRule{}
	ocimock.InitializeResource(resource, "mock-decryptionrule")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.DecryptionRuleSpec](t, `{
  "action": "NO_DECRYPT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_decryption_rule"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "action": "NO_DECRYPT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  }
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateDecryptionRuleDetails](t, `{
  "action": "NO_DECRYPT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_decryption_rule"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.DecryptionRule](t, `{
  "action": "NO_DECRYPT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "decryptionProfile": null,
  "description": null,
  "name": "osok_mock_decryption_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "secret": null,
  "secrets": null
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateDecryptionRuleDetails](t, `{
  "action": "NO_DECRYPT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.DecryptionRule](t, `{
  "action": "NO_DECRYPT",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses",
      "osok_mock_addresses2"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "decryptionProfile": null,
  "description": null,
  "name": "osok_mock_decryption_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  },
  "secret": null,
  "secrets": null
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.DecryptionRule,
		networkfirewallsdk.CreateDecryptionRuleDetails,
		networkfirewallsdk.UpdateDecryptionRuleDetails,
	]{
		CollectionPath:    "/20230501/networkFirewallPolicies/<ocid:1>/decryptionRules",
		ItemPath:          "/20230501/networkFirewallPolicies/<ocid:1>/decryptionRules/osok_mock_decryption_rule",
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
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateDecryptionRuleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.DecryptionRule) error {
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
			t.Errorf("close DecryptionRule OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &DecryptionRuleServiceManager{}
	hooks := newDecryptionRuleRuntimeHooks(manager, sdkClient)
	client := wrapDecryptionRuleGeneratedClient(hooks, defaultDecryptionRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.DecryptionRule](buildDecryptionRuleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.DecryptionRule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.DecryptionRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created DecryptionRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.DecryptionRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.DecryptionRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) {
				return fmt.Errorf("updated DecryptionRule status = %+v", current.Status)
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
