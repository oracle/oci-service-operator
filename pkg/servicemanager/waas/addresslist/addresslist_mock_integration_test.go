/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package addresslist

import (
	"context"
	"fmt"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationAddressListEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &waasv1beta1.AddressList{}
	ocimock.InitializeResource(resource, "mock-addresslist")
	resource.Spec = ocimock.MustJSONFixture[waasv1beta1.AddressListSpec](t, `{
  "addresses": [
    "192.0.2.10",
    "198.51.100.0/24"
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waas-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "addresses": [
    "192.0.2.20"
  ],
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[waassdk.CreateAddressListDetails](t, `{
  "addresses": [
    "192.0.2.10",
    "198.51.100.0/24"
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waas-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[waassdk.AddressList](t, `{
  "addressCount": 257,
  "addresses": [
    "192.0.2.10",
    "198.51.100.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:33.131Z"
    }
  },
  "displayName": "osok-mock-waas-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-01T21:43:33.597Z"
}`)
	updateRequest := ocimock.MustJSONFixture[waassdk.UpdateAddressListDetails](t, `{
  "addresses": [
    "192.0.2.20"
  ],
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[waassdk.AddressList](t, `{
  "addressCount": 1,
  "addresses": [
    "192.0.2.20"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:33.131Z"
    }
  },
  "displayName": "osok-mock-waas-address-list-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-01T21:43:33.597Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[waassdk.AddressList](t, `{
  "addressCount": 1,
  "addresses": [
    "192.0.2.20"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:33.131Z"
    }
  },
  "displayName": "osok-mock-waas-address-list-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETED",
  "timeCreated": "2026-09-01T21:43:33.597Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		waassdk.AddressList,
		waassdk.CreateAddressListDetails,
		waassdk.UpdateAddressListDetails,
	]{
		CollectionPath:    "/20181116/addressLists",
		ItemPath:          "/20181116/addressLists/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ waassdk.CreateAddressListDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ waassdk.AddressList) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181116", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close AddressList OCI mock: %v", err)
		}
	})
	sdkClient := waassdk.WaasClient{BaseClient: session.BaseClient()}
	hooks := newAddressListDefaultRuntimeHooks(sdkClient)
	applyAddressListRuntimeHooks(&hooks, sdkClient, nil)
	manager := &AddressListServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	client := wrapAddressListGeneratedClient(hooks, defaultAddressListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.AddressList](buildAddressListGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waasv1beta1.AddressList]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waasv1beta1.AddressList) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Addresses, current.Spec.Addresses) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created AddressList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *waasv1beta1.AddressList) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *waasv1beta1.AddressList) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Addresses, current.Spec.Addresses) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated AddressList status = %+v", current.Status)
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
