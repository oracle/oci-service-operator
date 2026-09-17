/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package urllist

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
func TestMockIntegrationUrlListCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.UrlList{}
	ocimock.InitializeResource(resource, "mock-urllist")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.UrlListSpec](t, `{
  "description": "OSOK recorded URL list",
  "name": "osok_mock_url_list",
  "urls": [
    {
      "pattern": "*.example.com/*",
      "type": "SIMPLE"
    }
  ]
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded UrlList updated",
  "urls": [
    {
      "pattern": "*.example.com/*",
      "type": "SIMPLE"
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateUrlListDetails](t, `{
  "description": "OSOK recorded URL list",
  "name": "osok_mock_url_list",
  "urls": [
    {
      "pattern": "*.example.com/*",
      "type": "SIMPLE"
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.UrlList](t, `{
  "description": "OSOK recorded URL list",
  "name": "osok_mock_url_list",
  "parentResourceId": "<ocid:1>",
  "totalUrls": 1,
  "urls": [
    {
      "pattern": "*.example.com/*",
      "type": "SIMPLE"
    }
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateUrlListDetails](t, `{
  "description": "OSOK recorded UrlList updated",
  "urls": [
    {
      "pattern": "*.example.com/*",
      "type": "SIMPLE"
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.UrlList](t, `{
  "description": "OSOK recorded UrlList updated",
  "name": "osok_mock_url_list",
  "parentResourceId": "<ocid:1>",
  "totalUrls": 1,
  "urls": [
    {
      "pattern": "*.example.com/*",
      "type": "SIMPLE"
    }
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.UrlList,
		networkfirewallsdk.CreateUrlListDetails,
		networkfirewallsdk.UpdateUrlListDetails,
	]{
		CollectionPath:    "/20230501/networkFirewallPolicies/<ocid:1>/urlLists",
		ItemPath:          "/20230501/networkFirewallPolicies/<ocid:1>/urlLists/osok_mock_url_list",
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
		ValidateCreate: func(request ocimock.Request, _ networkfirewallsdk.CreateUrlListDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.UrlList) error {
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
			t.Errorf("close UrlList OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &UrlListServiceManager{}
	hooks := newUrlListRuntimeHooks(manager, sdkClient)
	client := wrapUrlListGeneratedClient(hooks, defaultUrlListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.UrlList](buildUrlListGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.UrlList]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.UrlList) error {
			if !reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Urls, current.Spec.Urls) {
				return fmt.Errorf("created UrlList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.UrlList) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.UrlList) error {
			if !reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Urls, current.Spec.Urls) {
				return fmt.Errorf("updated UrlList status = %+v", current.Status)
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
