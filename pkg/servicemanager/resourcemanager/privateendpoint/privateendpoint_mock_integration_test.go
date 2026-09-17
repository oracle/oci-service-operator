/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package privateendpoint

import (
	"context"
	"fmt"
	resourcemanagersdk "github.com/oracle/oci-go-sdk/v65/resourcemanager"
	resourcemanagerv1beta1 "github.com/oracle/oci-service-operator/api/resourcemanager/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationPrivateEndpointEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &resourcemanagerv1beta1.PrivateEndpoint{}
	ocimock.InitializeResource(resource, "mock-privateendpoint")
	resource.Spec = ocimock.MustJSONFixture[resourcemanagerv1beta1.PrivateEndpointSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5",
  "freeformTags": {
    "osok-mock": "create"
  },
  "subnetId": "\u003cbinding:subnet-id\u003e",
  "vcnId": "\u003cbinding:vcn-id\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[resourcemanagersdk.CreatePrivateEndpointDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5",
  "freeformTags": {
    "osok-mock": "create"
  },
  "subnetId": "\u003cbinding:subnet-id\u003e",
  "vcnId": "\u003cbinding:vcn-id\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[resourcemanagersdk.PrivateEndpoint](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T18:17:30.774Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5",
  "dnsZones": [],
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isUsedWithConfigurationSourceProvider": false,
  "lifecycleState": "ACTIVE",
  "nsgIdList": [],
  "securityAttributes": {},
  "sourceIps": [
    "10.231.1.3"
  ],
  "subnetId": "<binding:subnet-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T18:17:30.942Z",
  "vcnId": "<binding:vcn-id>"
}`)
	createdReadStates := []resourcemanagersdk.PrivateEndpoint{
		ocimock.MustOCIResponseFixture[resourcemanagersdk.PrivateEndpoint](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T18:17:30.774Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5",
  "dnsZones": [],
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isUsedWithConfigurationSourceProvider": false,
  "lifecycleState": "ACTIVE",
  "nsgIdList": [],
  "securityAttributes": {},
  "sourceIps": [
    "10.231.1.3"
  ],
  "subnetId": "<binding:subnet-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T18:17:30.942Z",
  "vcnId": "<binding:vcn-id>"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[resourcemanagersdk.UpdatePrivateEndpointDetails](t, `{
  "description": "recorded update",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[resourcemanagersdk.PrivateEndpoint](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T18:17:30.774Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5-updated",
  "dnsZones": [],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isUsedWithConfigurationSourceProvider": false,
  "lifecycleState": "ACTIVE",
  "nsgIdList": [],
  "securityAttributes": {},
  "sourceIps": [
    "10.231.1.41"
  ],
  "subnetId": "<binding:subnet-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T18:17:30.942Z",
  "vcnId": "<binding:vcn-id>"
}`)
	updatedReadStates := []resourcemanagersdk.PrivateEndpoint{
		ocimock.MustOCIResponseFixture[resourcemanagersdk.PrivateEndpoint](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T18:17:30.774Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5-updated",
  "dnsZones": [],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isUsedWithConfigurationSourceProvider": false,
  "lifecycleState": "ACTIVE",
  "nsgIdList": [],
  "securityAttributes": {},
  "sourceIps": [
    "10.231.1.41"
  ],
  "subnetId": "<binding:subnet-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T18:17:30.942Z",
  "vcnId": "<binding:vcn-id>"
}`),
	}
	deletedReadStates := []resourcemanagersdk.PrivateEndpoint{
		ocimock.MustOCIResponseFixture[resourcemanagersdk.PrivateEndpoint](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T18:17:30.774Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-resource-manager-private-endpoint-v5-updated",
  "dnsZones": [],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isUsedWithConfigurationSourceProvider": false,
  "lifecycleState": "DELETING",
  "nsgIdList": [],
  "securityAttributes": {},
  "sourceIps": [
    "10.231.1.41"
  ],
  "subnetId": "<binding:subnet-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T18:17:30.942Z",
  "vcnId": "<binding:vcn-id>"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		resourcemanagersdk.PrivateEndpoint,
		resourcemanagersdk.CreatePrivateEndpointDetails,
		resourcemanagersdk.UpdatePrivateEndpointDetails,
	]{
		CollectionPath:     "/20180917/privateEndpoints",
		ItemPath:           "/20180917/privateEndpoints/<ocid:3>",
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
		CreateStatus:       200,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ resourcemanagersdk.CreatePrivateEndpointDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ resourcemanagersdk.PrivateEndpoint) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20180917", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close PrivateEndpoint OCI mock: %v", err)
		}
	})
	sdkClient := resourcemanagersdk.ResourceManagerClient{BaseClient: session.BaseClient()}
	manager := &PrivateEndpointServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newPrivateEndpointRuntimeHooks(manager, sdkClient)
	client := wrapPrivateEndpointGeneratedClient(hooks, defaultPrivateEndpointServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourcemanagerv1beta1.PrivateEndpoint](buildPrivateEndpointGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*resourcemanagerv1beta1.PrivateEndpoint]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *resourcemanagerv1beta1.PrivateEndpoint) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.SubnetId, current.Spec.SubnetId) ||
				!reflect.DeepEqual(current.Status.VcnId, current.Spec.VcnId) {
				return fmt.Errorf("created PrivateEndpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *resourcemanagerv1beta1.PrivateEndpoint) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *resourcemanagerv1beta1.PrivateEndpoint) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated PrivateEndpoint status = %+v", current.Status)
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
