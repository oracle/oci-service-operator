/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package distributeddatabaseprivateendpoint

import (
	"context"
	"fmt"
	distributeddatabasesdk "github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	distributeddatabasev1beta1 "github.com/oracle/oci-service-operator/api/distributeddatabase/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDistributedDatabasePrivateEndpointLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newTestDistributedDatabasePrivateEndpointResource()
	ocimock.InitializeResource(resource, "mock-distributeddatabaseprivateendpoint")
	resource.Spec = ocimock.MustJSONFixture[distributeddatabasev1beta1.DistributedDatabasePrivateEndpointSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "private endpoint for distributed database",
  "displayName": "ddb-private-endpoint",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "nsgIds": [
    "\u003cocid:2\u003e",
    "\u003cocid:3\u003e"
  ],
  "subnetId": "\u003cocid:4\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "private endpoint for distributed database-updated"
}`)
	createRequest := ocimock.MustJSONFixture[distributeddatabasesdk.CreateDistributedDatabasePrivateEndpointDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "private endpoint for distributed database",
  "displayName": "ddb-private-endpoint",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "nsgIds": [
    "\u003cocid:2\u003e",
    "\u003cocid:3\u003e"
  ],
  "subnetId": "\u003cocid:4\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[distributeddatabasesdk.DistributedDatabasePrivateEndpoint](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "private endpoint for distributed database",
  "displayName": "ddb-private-endpoint",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "\u003cocid:5\u003e",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "\u003cocid:2\u003e",
    "\u003cocid:3\u003e"
  ],
  "subnetId": "\u003cocid:4\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails](t, `{
  "description": "private endpoint for distributed database-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[distributeddatabasesdk.DistributedDatabasePrivateEndpoint](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "private endpoint for distributed database-updated",
  "displayName": "ddb-private-endpoint",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "\u003cocid:5\u003e",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "\u003cocid:2\u003e",
    "\u003cocid:3\u003e"
  ],
  "subnetId": "\u003cocid:4\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		distributeddatabasesdk.DistributedDatabasePrivateEndpoint,
		distributeddatabasesdk.CreateDistributedDatabasePrivateEndpointDetails,
		distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointDetails,
	]{
		CollectionPath:     "/20250101/distributedDatabasePrivateEndpoints",
		ItemPath:           "/20250101/distributedDatabasePrivateEndpoints/<ocid:5>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		UpdatedReadStates:  ocimock.LifecycleStateSequence(t, updatedState, "UPDATING"),
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ distributeddatabasesdk.CreateDistributedDatabasePrivateEndpointDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ distributeddatabasesdk.DistributedDatabasePrivateEndpoint) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20250101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DistributedDatabasePrivateEndpoint OCI mock: %v", err)
		}
	})
	sdkClient := distributeddatabasesdk.DistributedDbPrivateEndpointServiceClient{BaseClient: session.BaseClient()}
	manager := &DistributedDatabasePrivateEndpointServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newDistributedDatabasePrivateEndpointDefaultRuntimeHooks(sdkClient)
	applyDistributedDatabasePrivateEndpointRuntimeHooks(&hooks)
	client := wrapDistributedDatabasePrivateEndpointGeneratedClient(hooks, defaultDistributedDatabasePrivateEndpointServiceClient{ServiceClient: generatedruntime.NewServiceClient[*distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint](buildDistributedDatabasePrivateEndpointGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.NsgIds, current.Spec.NsgIds) ||
				!reflect.DeepEqual(current.Status.SubnetId, current.Spec.SubnetId) {
				return fmt.Errorf("created DistributedDatabasePrivateEndpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated DistributedDatabasePrivateEndpoint status = %+v", current.Status)
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
