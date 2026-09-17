/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securityrule

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
func TestMockIntegrationSecurityRuleCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.SecurityRule{}
	ocimock.InitializeResource(resource, "mock-securityrule")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.SecurityRuleSpec](t, `{
  "action": "ALLOW",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_security_rule"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "action": "DROP",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  }
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateSecurityRuleDetails](t, `{
  "action": "ALLOW",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  },
  "name": "osok_mock_security_rule"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.SecurityRule](t, `{
  "action": "ALLOW",
  "condition": {
    "application": null,
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "service": null,
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ],
    "url": null
  },
  "description": null,
  "inspection": null,
  "name": "osok_mock_security_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  }
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateSecurityRuleDetails](t, `{
  "action": "DROP",
  "condition": {
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ]
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.SecurityRule](t, `{
  "action": "DROP",
  "condition": {
    "application": null,
    "destinationAddress": [
      "osok_mock_prereq_addresses"
    ],
    "service": null,
    "sourceAddress": [
      "osok_mock_prereq_addresses"
    ],
    "url": null
  },
  "description": null,
  "inspection": null,
  "name": "osok_mock_security_rule",
  "parentResourceId": "<ocid:1>",
  "position": {
    "afterRule": null,
    "beforeRule": null
  }
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.SecurityRule,
		networkfirewallsdk.CreateSecurityRuleDetails,
		networkfirewallsdk.UpdateSecurityRuleDetails,
	]{
		CollectionPath:    "/20230501/networkFirewallPolicies/<ocid:1>/securityRules",
		ItemPath:          "/20230501/networkFirewallPolicies/<ocid:1>/securityRules/osok_mock_security_rule",
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
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateSecurityRuleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.SecurityRule) error {
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
			t.Errorf("close SecurityRule OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &SecurityRuleServiceManager{}
	hooks := newSecurityRuleRuntimeHooks(manager, sdkClient)
	client := wrapSecurityRuleGeneratedClient(hooks, defaultSecurityRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.SecurityRule](buildSecurityRuleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.SecurityRule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.SecurityRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created SecurityRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.SecurityRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.SecurityRule) error {
			if !reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.Condition, current.Spec.Condition) {
				return fmt.Errorf("updated SecurityRule status = %+v", current.Status)
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
