/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package ruleset

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
func TestMockIntegrationRuleSetCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loadbalancerv1beta1.RuleSet{}
	ocimock.InitializeResource(resource, "mock-ruleset")
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.RuleSetSpec](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "header": "x-osok-mock",
      "value": "created"
    }
  ],
  "name": "osok_mock_rule_set"
}`)
	resource.Spec.LoadBalancerId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "header": "x-osok-mock",
      "value": "updated"
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreateRuleSetDetails](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "header": "x-osok-mock",
      "value": "created"
    }
  ],
  "name": "osok_mock_rule_set"
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.RuleSet](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "conditions": null,
      "header": "x-osok-mock",
      "value": "created"
    }
  ],
  "name": "osok_mock_rule_set"
}`)
	createdReadStates := []loadbalancersdk.RuleSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.RuleSet](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "conditions": null,
      "header": "x-osok-mock",
      "value": "created"
    }
  ],
  "name": "osok_mock_rule_set"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdateRuleSetDetails](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "header": "x-osok-mock",
      "value": "updated"
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.RuleSet](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "conditions": null,
      "header": "x-osok-mock",
      "value": "updated"
    }
  ],
  "name": "osok_mock_rule_set"
}`)
	updatedReadStates := []loadbalancersdk.RuleSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.RuleSet](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "conditions": null,
      "header": "x-osok-mock",
      "value": "updated"
    }
  ],
  "name": "osok_mock_rule_set"
}`),
	}
	deletedReadStates := []loadbalancersdk.RuleSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.RuleSet](t, `{
  "items": [
    {
      "action": "ADD_HTTP_REQUEST_HEADER",
      "conditions": null,
      "header": "x-osok-mock",
      "value": "updated"
    }
  ],
  "name": "osok_mock_rule_set"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loadbalancersdk.RuleSet,
		loadbalancersdk.CreateRuleSetDetails,
		loadbalancersdk.UpdateRuleSetDetails,
	]{
		CollectionPath:     "/20170115/loadBalancers/<ocid:1>/ruleSets",
		ItemPath:           "/20170115/loadBalancers/<ocid:1>/ruleSets/osok_mock_rule_set",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
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
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreateRuleSetDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loadbalancersdk.RuleSet) error {
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
			t.Errorf("close RuleSet OCI mock: %v", err)
		}
	})
	sdkClient := loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()}
	hooks := newRuleSetRuntimeHooksWithOCIClient(sdkClient)
	applyRuleSetRuntimeHooks(&hooks)
	manager := &RuleSetServiceManager{}
	client := wrapRuleSetGeneratedClient(hooks, defaultRuleSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.RuleSet](buildRuleSetGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.RuleSet]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loadbalancerv1beta1.RuleSet) error {
			if !reflect.DeepEqual(current.Status.Items, current.Spec.Items) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created RuleSet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.RuleSet) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.RuleSet) error {
			if !reflect.DeepEqual(current.Status.Items, current.Spec.Items) {
				return fmt.Errorf("updated RuleSet status = %+v", current.Status)
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
