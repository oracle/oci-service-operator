/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package networkaddresslist

import (
	"context"
	"fmt"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationNetworkAddressListEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &wafv1beta1.NetworkAddressList{}
	ocimock.InitializeResource(resource, "mock-networkaddresslist")
	resource.Spec = ocimock.MustJSONFixture[wafv1beta1.NetworkAddressListSpec](t, `{
  "addresses": [
    "192.0.2.0/24"
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "type": "ADDRESSES"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "addresses": [
    "198.51.100.0/24"
  ],
  "freeformTags": {
    "osok-mock": "update"
  },
  "type": "ADDRESSES"
}`)
	createRequest := ocimock.MustJSONFixture[wafsdk.CreateNetworkAddressListAddressesDetails](t, `{
  "addresses": [
    "192.0.2.0/24"
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[wafsdk.NetworkAddressListAddresses](t, `{
  "addresses": [
    "192.0.2.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:55.607Z"
    }
  },
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T21:43:55.695Z",
  "timeUpdated": "2026-09-01T21:44:15.169Z",
  "type": "ADDRESSES"
}`)
	createdReadStates := []wafsdk.NetworkAddressListAddresses{
		ocimock.MustOCIResponseFixture[wafsdk.NetworkAddressListAddresses](t, `{
  "addresses": [
    "192.0.2.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:55.607Z"
    }
  },
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T21:43:55.695Z",
  "timeUpdated": "2026-09-01T21:44:15.169Z",
  "type": "ADDRESSES"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[wafsdk.UpdateNetworkAddressListAddressesDetails](t, `{
  "addresses": [
    "198.51.100.0/24"
  ],
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[wafsdk.NetworkAddressListAddresses](t, `{
  "addresses": [
    "198.51.100.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:55.607Z"
    }
  },
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T21:43:55.695Z",
  "timeUpdated": "2026-09-01T21:44:59.069Z",
  "type": "ADDRESSES"
}`)
	updatedReadStates := []wafsdk.NetworkAddressListAddresses{
		ocimock.MustOCIResponseFixture[wafsdk.NetworkAddressListAddresses](t, `{
  "addresses": [
    "198.51.100.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:55.607Z"
    }
  },
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T21:43:55.695Z",
  "timeUpdated": "2026-09-01T21:44:59.069Z",
  "type": "ADDRESSES"
}`),
	}
	deletedReadStates := []wafsdk.NetworkAddressListAddresses{
		ocimock.MustOCIResponseFixture[wafsdk.NetworkAddressListAddresses](t, `{
  "addresses": [
    "198.51.100.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:55.607Z"
    }
  },
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETING",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T21:43:55.695Z",
  "timeUpdated": "2026-09-01T21:45:02.177Z",
  "type": "ADDRESSES"
}`),
		ocimock.MustOCIResponseFixture[wafsdk.NetworkAddressListAddresses](t, `{
  "addresses": [
    "198.51.100.0/24"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:43:55.607Z"
    }
  },
  "displayName": "osok-mock-waf-address-list-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T21:43:55.695Z",
  "timeUpdated": "2026-09-01T21:45:17.214Z",
  "type": "ADDRESSES"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		wafsdk.NetworkAddressListAddresses,
		wafsdk.CreateNetworkAddressListAddressesDetails,
		wafsdk.UpdateNetworkAddressListAddressesDetails,
	]{
		CollectionPath:    "/20210930/networkAddressLists",
		ItemPath:          "/20210930/networkAddressLists/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:      &createdState,
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
		ValidateCreateRaw: func(request ocimock.Request) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "ADDRESSES", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "ADDRESSES", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ wafsdk.NetworkAddressListAddresses) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210930", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close NetworkAddressList OCI mock: %v", err)
		}
	})
	sdkClient := wafsdk.WafClient{BaseClient: session.BaseClient()}
	hooks := newNetworkAddressListDefaultRuntimeHooks(sdkClient)
	applyNetworkAddressListRuntimeHooks(&hooks)
	manager := &NetworkAddressListServiceManager{Log: loggerutil.OSOKLogger{}}
	client := wrapNetworkAddressListGeneratedClient(hooks, defaultNetworkAddressListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*wafv1beta1.NetworkAddressList](buildNetworkAddressListGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*wafv1beta1.NetworkAddressList]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *wafv1beta1.NetworkAddressList) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Addresses, current.Spec.Addresses) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created NetworkAddressList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *wafv1beta1.NetworkAddressList) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *wafv1beta1.NetworkAddressList) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Addresses, current.Spec.Addresses) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("updated NetworkAddressList status = %+v", current.Status)
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
