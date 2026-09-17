/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package application

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
func TestMockIntegrationApplicationCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.Application{}
	ocimock.InitializeResource(resource, "mock-application")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.ApplicationSpec](t, `{
  "icmpType": 8,
  "name": "osok_mock_application",
  "type": "ICMP"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "icmpCode": 1,
  "icmpType": 3,
  "type": "ICMP"
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateIcmpApplicationDetails](t, `{
  "icmpType": 8,
  "name": "osok_mock_application"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.IcmpApplication](t, `{
  "description": null,
  "icmpCode": null,
  "icmpType": 8,
  "name": "osok_mock_application",
  "parentResourceId": "<ocid:1>",
  "type": "ICMP"
}`)
	createdReadStates := []networkfirewallsdk.IcmpApplication{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.IcmpApplication](t, `{
  "description": null,
  "icmpCode": null,
  "icmpType": 8,
  "name": "osok_mock_application",
  "parentResourceId": "<ocid:1>",
  "type": "ICMP"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateIcmpApplicationDetails](t, `{
  "icmpCode": 1,
  "icmpType": 3
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.IcmpApplication](t, `{
  "description": null,
  "icmpCode": 1,
  "icmpType": 3,
  "name": "osok_mock_application",
  "parentResourceId": "<ocid:1>",
  "type": "ICMP"
}`)
	updatedReadStates := []networkfirewallsdk.IcmpApplication{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.IcmpApplication](t, `{
  "description": null,
  "icmpCode": 1,
  "icmpType": 3,
  "name": "osok_mock_application",
  "parentResourceId": "<ocid:1>",
  "type": "ICMP"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.IcmpApplication,
		networkfirewallsdk.CreateIcmpApplicationDetails,
		networkfirewallsdk.UpdateIcmpApplicationDetails,
	]{
		CollectionPath:     "/20230501/networkFirewallPolicies/<ocid:1>/applications",
		ItemPath:           "/20230501/networkFirewallPolicies/<ocid:1>/applications/osok_mock_application",
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
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "ICMP", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "ICMP", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.IcmpApplication) error {
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
			t.Errorf("close Application OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &ApplicationServiceManager{}
	hooks := newApplicationRuntimeHooks(manager, sdkClient)
	client := wrapApplicationGeneratedClient(hooks, defaultApplicationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.Application](buildApplicationGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.Application]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.Application) error {
			if !reflect.DeepEqual(current.Status.IcmpType, current.Spec.IcmpType) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created Application status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.Application) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.Application) error {
			if !reflect.DeepEqual(current.Status.IcmpCode, current.Spec.IcmpCode) ||
				!reflect.DeepEqual(current.Status.IcmpType, current.Spec.IcmpType) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("updated Application status = %+v", current.Status)
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
