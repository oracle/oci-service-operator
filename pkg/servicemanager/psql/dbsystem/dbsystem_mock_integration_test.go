/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dbsystem

import (
	"context"
	"fmt"
	psqlsdk "github.com/oracle/oci-go-sdk/v65/psql"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDbSystemLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := testDbSystemResource()
	ocimock.InitializeResource(resource, "mock-dbsystem")
	resource.Spec = ocimock.MustJSONFixture[psqlv1beta1.DbSystemSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "credentials": {
    "passwordDetails": {
      "password": "\u003credacted\u003e",
      "passwordType": "PLAIN_TEXT"
    },
    "username": "postgres"
  },
  "dbVersion": "14",
  "displayName": "sample-db",
  "networkDetails": {
    "subnetId": "\u003cocid:2\u003e"
  },
  "shape": "VM.Standard.E4.Flex",
  "storageDetails": {
    "iops": 10,
    "isRegionallyDurable": true,
    "systemType": "OCI_OPTIMIZED_STORAGE"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "sample-db-updated"
}`)
	createRequest := ocimock.MustJSONFixture[psqlsdk.CreateDbSystemDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "credentials": {
    "passwordDetails": {
      "password": "\u003credacted\u003e",
      "passwordType": "PLAIN_TEXT"
    },
    "username": "postgres"
  },
  "dbVersion": "14",
  "displayName": "sample-db",
  "networkDetails": {
    "subnetId": "\u003cocid:2\u003e"
  },
  "shape": "VM.Standard.E4.Flex",
  "storageDetails": {
    "iops": 10,
    "isRegionallyDurable": true,
    "systemType": "OCI_OPTIMIZED_STORAGE"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[psqlsdk.DbSystem](t, `{
  "id": "<ocid:3>",
  "displayName": "sample-db",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "systemType": "OCI_OPTIMIZED_STORAGE",
  "dbVersion": "14",
  "shape": "VM.Standard.E4.Flex",
  "instanceOcpuCount": 2,
  "instanceMemorySizeInGBs": 16,
  "instanceCount": 1,
  "storageDetails": {
    "systemType": "OCI_OPTIMIZED_STORAGE",
    "isRegionallyDurable": true,
    "iops": 10
  },
  "networkDetails": {
    "subnetId": "<ocid:2>"
  },
  "adminUsername": "postgres",
  "freeformTags": {},
  "definedTags": {},
  "instances": []
}`)
	createdReadStates := []psqlsdk.DbSystem{
		ocimock.MustOCIResponseFixture[psqlsdk.DbSystem](t, `{
  "id": "<ocid:3>",
  "displayName": "sample-db",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "systemType": "OCI_OPTIMIZED_STORAGE",
  "dbVersion": "14",
  "shape": "VM.Standard.E4.Flex",
  "instanceOcpuCount": 2,
  "instanceMemorySizeInGBs": 16,
  "instanceCount": 1,
  "storageDetails": {
    "systemType": "OCI_OPTIMIZED_STORAGE",
    "isRegionallyDurable": true,
    "iops": 10
  },
  "networkDetails": {
    "subnetId": "<ocid:2>"
  },
  "adminUsername": "postgres",
  "freeformTags": {},
  "definedTags": {},
  "instances": []
}`),
	}
	updateRequest := ocimock.MustJSONFixture[psqlsdk.UpdateDbSystemDetails](t, `{
  "displayName": "sample-db-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[psqlsdk.DbSystem](t, `{
  "id": "<ocid:3>",
  "displayName": "sample-db-updated",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "systemType": "OCI_OPTIMIZED_STORAGE",
  "dbVersion": "14",
  "shape": "VM.Standard.E4.Flex",
  "instanceOcpuCount": 2,
  "instanceMemorySizeInGBs": 16,
  "instanceCount": 1,
  "storageDetails": {
    "systemType": "OCI_OPTIMIZED_STORAGE",
    "isRegionallyDurable": true,
    "iops": 10
  },
  "networkDetails": {
    "subnetId": "<ocid:2>"
  },
  "adminUsername": "postgres",
  "freeformTags": {},
  "definedTags": {},
  "instances": []
}`)
	updatedReadStates := []psqlsdk.DbSystem{
		ocimock.MustOCIResponseFixture[psqlsdk.DbSystem](t, `{
  "id": "<ocid:3>",
  "displayName": "sample-db-updated",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "systemType": "OCI_OPTIMIZED_STORAGE",
  "dbVersion": "14",
  "shape": "VM.Standard.E4.Flex",
  "instanceOcpuCount": 2,
  "instanceMemorySizeInGBs": 16,
  "instanceCount": 1,
  "storageDetails": {
    "systemType": "OCI_OPTIMIZED_STORAGE",
    "isRegionallyDurable": true,
    "iops": 10
  },
  "networkDetails": {
    "subnetId": "<ocid:2>"
  },
  "adminUsername": "postgres",
  "freeformTags": {},
  "definedTags": {},
  "instances": []
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		psqlsdk.DbSystem,
		psqlsdk.CreateDbSystemDetails,
		psqlsdk.UpdateDbSystemDetails,
	]{
		CollectionPath:     "/20220915/dbSystems",
		ItemPath:           "/20220915/dbSystems/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		CreatedReadStates:  append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...),
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       200,
		UpdateStatus:       200,
		DeleteStatus:       202,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ psqlsdk.CreateDbSystemDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ psqlsdk.DbSystem) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220915", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DbSystem OCI mock: %v", err)
		}
	})
	sdkClient := psqlsdk.PostgresqlClient{BaseClient: session.BaseClient()}
	client := manualDbSystemServiceClient{sdk: sdkClient, log: discardDbSystemLogger()}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*psqlv1beta1.DbSystem]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *psqlv1beta1.DbSystem) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				current.Status.CompartmentId != current.Spec.CompartmentId ||
				current.Status.DbVersion != current.Spec.DbVersion ||
				current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.NetworkDetails.SubnetId != current.Spec.NetworkDetails.SubnetId ||
				current.Status.Shape != current.Spec.Shape ||
				current.Status.StorageDetails.Iops != current.Spec.StorageDetails.Iops ||
				current.Status.StorageDetails.SystemType != current.Spec.StorageDetails.SystemType ||
				current.Status.StorageDetails.IsRegionallyDurable == nil ||
				!*current.Status.StorageDetails.IsRegionallyDurable {
				return fmt.Errorf("created DbSystem status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *psqlv1beta1.DbSystem) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *psqlv1beta1.DbSystem) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != current.Spec.DisplayName {
				return fmt.Errorf("updated DbSystem status = %+v", current.Status)
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
