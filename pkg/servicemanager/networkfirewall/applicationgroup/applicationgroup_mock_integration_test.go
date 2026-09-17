/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package applicationgroup

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
func TestMockIntegrationApplicationGroupCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.ApplicationGroup{}
	ocimock.InitializeResource(resource, "mock-applicationgroup")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.ApplicationGroupSpec](t, `{
  "apps": [
    "osok_mock_prereq_icmp"
  ],
  "name": "osok_mock_app_group"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "apps": [
    "osok_mock_prereq_icmp",
    "osok_mock_icmp2"
  ]
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateApplicationGroupDetails](t, `{
  "apps": [
    "osok_mock_prereq_icmp"
  ],
  "name": "osok_mock_app_group"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.ApplicationGroup](t, `{
  "apps": [
    "osok_mock_prereq_icmp"
  ],
  "description": null,
  "name": "osok_mock_app_group",
  "parentResourceId": "<ocid:1>",
  "totalApps": 1
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateApplicationGroupDetails](t, `{
  "apps": [
    "osok_mock_prereq_icmp",
    "osok_mock_icmp2"
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.ApplicationGroup](t, `{
  "apps": [
    "osok_mock_prereq_icmp",
    "osok_mock_icmp2"
  ],
  "description": null,
  "name": "osok_mock_app_group",
  "parentResourceId": "<ocid:1>",
  "totalApps": 2
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.ApplicationGroup,
		networkfirewallsdk.CreateApplicationGroupDetails,
		networkfirewallsdk.UpdateApplicationGroupDetails,
	]{
		CollectionPath:    "/20230501/networkFirewallPolicies/<ocid:1>/applicationGroups",
		ItemPath:          "/20230501/networkFirewallPolicies/<ocid:1>/applicationGroups/osok_mock_app_group",
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
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateApplicationGroupDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.ApplicationGroup) error {
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
			t.Errorf("close ApplicationGroup OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &ApplicationGroupServiceManager{}
	hooks := newApplicationGroupRuntimeHooks(manager, sdkClient)
	client := wrapApplicationGroupGeneratedClient(hooks, defaultApplicationGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.ApplicationGroup](buildApplicationGroupGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.ApplicationGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.ApplicationGroup) error {
			if !reflect.DeepEqual(current.Status.Apps, current.Spec.Apps) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created ApplicationGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.ApplicationGroup) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.ApplicationGroup) error {
			if !reflect.DeepEqual(current.Status.Apps, current.Spec.Apps) {
				return fmt.Errorf("updated ApplicationGroup status = %+v", current.Status)
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
