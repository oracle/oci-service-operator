/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package bdsinstance

import (
	"context"
	"fmt"
	bdssdk "github.com/oracle/oci-go-sdk/v65/bds"
	bdsv1beta1 "github.com/oracle/oci-service-operator/api/bds/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationBdsInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeSpecBdsInstance()
	ocimock.InitializeResource(resource, "mock-bdsinstance")
	resource.Spec = ocimock.MustJSONFixture[bdsv1beta1.BdsInstanceSpec](t, `{
  "clusterAdminPassword": "\u003credacted\u003e",
  "clusterPublicKey": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCy",
  "clusterVersion": "ODH2_0",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "test-bds",
  "freeformTags": {
    "run": "1"
  },
  "isHighAvailability": false,
  "isSecure": false,
  "nodes": [
    {
      "blockVolumeSizeInGBs": 150,
      "nodeType": "MASTER",
      "shape": "VM.Standard3.Flex",
      "shapeConfig": {
        "memoryInGBs": 32,
        "nvmes": 1,
        "ocpus": 2
      },
      "subnetId": "\u003cocid:2\u003e"
    }
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "test-bds-updated"
}`)
	createRequest := ocimock.MustJSONFixture[bdssdk.CreateBdsInstanceDetails](t, `{
  "clusterAdminPassword": "\u003credacted\u003e",
  "clusterPublicKey": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCy",
  "clusterVersion": "ODH2_0",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "test-bds",
  "freeformTags": {
    "run": "1"
  },
  "isHighAvailability": false,
  "isSecure": false,
  "nodes": [
    {
      "blockVolumeSizeInGBs": 150,
      "nodeType": "MASTER",
      "shape": "VM.Standard3.Flex",
      "shapeConfig": {
        "memoryInGBs": 32,
        "nvmes": 1,
        "ocpus": 2
      },
      "subnetId": "\u003cocid:2\u003e"
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[bdssdk.BdsInstance](t, `{
  "id": "<ocid:3>",
  "compartmentId": "<ocid:1>",
  "displayName": "test-bds",
  "lifecycleState": "ACTIVE",
  "isHighAvailability": false,
  "isSecure": false,
  "clusterVersion": "ODH2_0",
  "nodes": [
    {
      "nodeType": "MASTER",
      "shape": "VM.Standard3.Flex",
      "subnetId": "<ocid:2>",
      "attachedBlockVolumes": [
        {
          "volumeSizeInGBs": 150
        }
      ],
      "ocpus": 2,
      "memoryInGBs": 32,
      "nvmes": 1
    }
  ],
  "numberOfNodes": 1,
  "freeformTags": {
    "run": "1"
  }
}`)
	createdReadStates := []bdssdk.BdsInstance{
		ocimock.MustOCIResponseFixture[bdssdk.BdsInstance](t, `{
  "id": "<ocid:3>",
  "compartmentId": "<ocid:1>",
  "displayName": "test-bds",
  "lifecycleState": "ACTIVE",
  "isHighAvailability": false,
  "isSecure": false,
  "clusterVersion": "ODH2_0",
  "nodes": [
    {
      "nodeType": "MASTER",
      "shape": "VM.Standard3.Flex",
      "subnetId": "<ocid:2>",
      "attachedBlockVolumes": [
        {
          "volumeSizeInGBs": 150
        }
      ],
      "ocpus": 2,
      "memoryInGBs": 32,
      "nvmes": 1
    }
  ],
  "numberOfNodes": 1,
  "freeformTags": {
    "run": "1"
  }
}`),
	}
	updateRequest := ocimock.MustJSONFixture[bdssdk.UpdateBdsInstanceDetails](t, `{
  "displayName": "test-bds-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[bdssdk.BdsInstance](t, `{
  "id": "<ocid:3>",
  "compartmentId": "<ocid:1>",
  "displayName": "test-bds-updated",
  "lifecycleState": "ACTIVE",
  "isHighAvailability": false,
  "isSecure": false,
  "clusterVersion": "ODH2_0",
  "nodes": [
    {
      "nodeType": "MASTER",
      "shape": "VM.Standard3.Flex",
      "subnetId": "<ocid:2>",
      "attachedBlockVolumes": [
        {
          "volumeSizeInGBs": 150
        }
      ],
      "ocpus": 2,
      "memoryInGBs": 32,
      "nvmes": 1
    }
  ],
  "numberOfNodes": 1,
  "freeformTags": {
    "run": "1"
  }
}`)
	updatedReadStates := []bdssdk.BdsInstance{
		ocimock.MustOCIResponseFixture[bdssdk.BdsInstance](t, `{
  "id": "<ocid:3>",
  "compartmentId": "<ocid:1>",
  "displayName": "test-bds-updated",
  "lifecycleState": "ACTIVE",
  "isHighAvailability": false,
  "isSecure": false,
  "clusterVersion": "ODH2_0",
  "nodes": [
    {
      "nodeType": "MASTER",
      "shape": "VM.Standard3.Flex",
      "subnetId": "<ocid:2>",
      "attachedBlockVolumes": [
        {
          "volumeSizeInGBs": 150
        }
      ],
      "ocpus": 2,
      "memoryInGBs": 32,
      "nvmes": 1
    }
  ],
  "numberOfNodes": 1,
  "freeformTags": {
    "run": "1"
  }
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		bdssdk.BdsInstance,
		bdssdk.CreateBdsInstanceDetails,
		bdssdk.UpdateBdsInstanceDetails,
	]{
		CollectionPath:     "/20190531/bdsInstances",
		ItemPath:           "/20190531/bdsInstances/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeArray,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		CreatedReadStates:  append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...),
		UpdatedReadStates:  append(ocimock.LifecycleStates(t, updatedState, "RESUMING", "SUSPENDING", "UPDATING"), updatedReadStates...),
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       200,
		UpdateStatus:       200,
		DeleteStatus:       202,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ bdssdk.CreateBdsInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ bdssdk.BdsInstance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20190531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close BdsInstance OCI mock: %v", err)
		}
	})
	sdkClient := bdssdk.BdsClient{BaseClient: session.BaseClient()}
	client := newBdsInstanceTestManager(sdkClient).client
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*bdsv1beta1.BdsInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *bdsv1beta1.BdsInstance) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ClusterVersion, current.Spec.ClusterVersion) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsHighAvailability, current.Spec.IsHighAvailability) ||
				!reflect.DeepEqual(current.Status.IsSecure, current.Spec.IsSecure) ||
				current.Status.NumberOfNodes != 1 {
				return fmt.Errorf("created BdsInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *bdsv1beta1.BdsInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *bdsv1beta1.BdsInstance) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated BdsInstance status = %+v", current.Status)
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
