/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package opensearchcluster

import (
	"context"
	"fmt"
	opensearchsdk "github.com/oracle/oci-go-sdk/v65/opensearch"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOpensearchClusterLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &opensearchv1beta1.OpensearchCluster{Spec: opensearchv1beta1.OpensearchClusterSpec{
		DisplayName:                    mockOpensearchClusterName,
		CompartmentId:                  "ocid1.compartment.oc1..mock",
		SoftwareVersion:                "2.11.0",
		MasterNodeCount:                3,
		MasterNodeHostType:             string(opensearchsdk.MasterNodeHostTypeFlex),
		MasterNodeHostOcpuCount:        1,
		MasterNodeHostMemoryGB:         16,
		DataNodeCount:                  3,
		DataNodeHostType:               string(opensearchsdk.DataNodeHostTypeFlex),
		DataNodeHostOcpuCount:          2,
		DataNodeHostMemoryGB:           32,
		DataNodeStorageGB:              50,
		OpendashboardNodeCount:         1,
		OpendashboardNodeHostOcpuCount: 1,
		OpendashboardNodeHostMemoryGB:  8,
		VcnId:                          "ocid1.vcn.oc1..mock",
		SubnetId:                       "ocid1.subnet.oc1..mock",
		VcnCompartmentId:               "ocid1.compartment.oc1..mock",
		SubnetCompartmentId:            "ocid1.compartment.oc1..mock",
		SecurityMode:                   string(opensearchsdk.SecurityModeDisabled),
		FreeformTags:                   map[string]string{"osok-mock": "synthetic"},
	}}
	ocimock.InitializeResource(resource, "mock-opensearchcluster")
	resource.Spec = ocimock.MustJSONFixture[opensearchv1beta1.OpensearchClusterSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dataNodeCount": 3,
  "dataNodeHostMemoryGB": 32,
  "dataNodeHostOcpuCount": 2,
  "dataNodeHostType": "FLEX",
  "dataNodeStorageGB": 50,
  "displayName": "osok-mock-opensearch-v1",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "masterNodeCount": 3,
  "masterNodeHostMemoryGB": 16,
  "masterNodeHostOcpuCount": 1,
  "masterNodeHostType": "FLEX",
  "opendashboardNodeCount": 1,
  "opendashboardNodeHostMemoryGB": 8,
  "opendashboardNodeHostOcpuCount": 1,
  "securityMode": "DISABLED",
  "softwareVersion": "2.11.0",
  "subnetCompartmentId": "\u003cocid:1\u003e",
  "subnetId": "\u003cocid:2\u003e",
  "vcnCompartmentId": "\u003cocid:1\u003e",
  "vcnId": "\u003cocid:3\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-opensearch-v1-updated"
}`)
	createRequest := ocimock.MustJSONFixture[opensearchsdk.CreateOpensearchClusterDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dataNodeCount": 3,
  "dataNodeHostMemoryGB": 32,
  "dataNodeHostOcpuCount": 2,
  "dataNodeHostType": "FLEX",
  "dataNodeStorageGB": 50,
  "displayName": "osok-mock-opensearch-v1",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "masterNodeCount": 3,
  "masterNodeHostMemoryGB": 16,
  "masterNodeHostOcpuCount": 1,
  "masterNodeHostType": "FLEX",
  "opendashboardNodeCount": 1,
  "opendashboardNodeHostMemoryGB": 8,
  "opendashboardNodeHostOcpuCount": 1,
  "securityMode": "DISABLED",
  "softwareVersion": "2.11.0",
  "subnetCompartmentId": "\u003cocid:1\u003e",
  "subnetId": "\u003cocid:2\u003e",
  "vcnCompartmentId": "\u003cocid:1\u003e",
  "vcnId": "\u003cocid:3\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[opensearchsdk.OpensearchCluster](t, `{
  "compartmentId": "<ocid:1>",
  "dataNodeCount": 3,
  "dataNodeHostMemoryGB": 32,
  "dataNodeHostOcpuCount": 2,
  "dataNodeHostType": "FLEX",
  "dataNodeStorageGB": 50,
  "displayName": "osok-mock-opensearch-v1",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "masterNodeCount": 3,
  "masterNodeHostMemoryGB": 16,
  "masterNodeHostOcpuCount": 1,
  "masterNodeHostType": "FLEX",
  "opendashboardNodeCount": 1,
  "opendashboardNodeHostMemoryGB": 8,
  "opendashboardNodeHostOcpuCount": 1,
  "securityMode": "DISABLED",
  "softwareVersion": "2.11.0",
  "subnetCompartmentId": "<ocid:1>",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:01:00Z",
  "vcnCompartmentId": "<ocid:1>",
  "vcnId": "<ocid:3>"
}`)
	createdReadStates := []opensearchsdk.OpensearchCluster{
		ocimock.MustOCIResponseFixture[opensearchsdk.OpensearchCluster](t, `{
  "compartmentId": "<ocid:1>",
  "dataNodeCount": 3,
  "dataNodeHostMemoryGB": 32,
  "dataNodeHostOcpuCount": 2,
  "dataNodeHostType": "FLEX",
  "dataNodeStorageGB": 50,
  "displayName": "osok-mock-opensearch-v1",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "masterNodeCount": 3,
  "masterNodeHostMemoryGB": 16,
  "masterNodeHostOcpuCount": 1,
  "masterNodeHostType": "FLEX",
  "opendashboardNodeCount": 1,
  "opendashboardNodeHostMemoryGB": 8,
  "opendashboardNodeHostOcpuCount": 1,
  "securityMode": "DISABLED",
  "softwareVersion": "2.11.0",
  "subnetCompartmentId": "<ocid:1>",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:01:00Z",
  "vcnCompartmentId": "<ocid:1>",
  "vcnId": "<ocid:3>"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[opensearchsdk.UpdateOpensearchClusterDetails](t, `{
  "displayName": "osok-mock-opensearch-v1-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[opensearchsdk.OpensearchCluster](t, `{
  "compartmentId": "<ocid:1>",
  "dataNodeCount": 3,
  "dataNodeHostMemoryGB": 32,
  "dataNodeHostOcpuCount": 2,
  "dataNodeHostType": "FLEX",
  "dataNodeStorageGB": 50,
  "displayName": "osok-mock-opensearch-v1-updated",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "masterNodeCount": 3,
  "masterNodeHostMemoryGB": 16,
  "masterNodeHostOcpuCount": 1,
  "masterNodeHostType": "FLEX",
  "opendashboardNodeCount": 1,
  "opendashboardNodeHostMemoryGB": 8,
  "opendashboardNodeHostOcpuCount": 1,
  "securityMode": "DISABLED",
  "softwareVersion": "2.11.0",
  "subnetCompartmentId": "<ocid:1>",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:02:00Z",
  "vcnCompartmentId": "<ocid:1>",
  "vcnId": "<ocid:3>"
}`)
	updatedReadStates := []opensearchsdk.OpensearchCluster{
		ocimock.MustOCIResponseFixture[opensearchsdk.OpensearchCluster](t, `{
  "compartmentId": "<ocid:1>",
  "dataNodeCount": 3,
  "dataNodeHostMemoryGB": 32,
  "dataNodeHostOcpuCount": 2,
  "dataNodeHostType": "FLEX",
  "dataNodeStorageGB": 50,
  "displayName": "osok-mock-opensearch-v1-updated",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "masterNodeCount": 3,
  "masterNodeHostMemoryGB": 16,
  "masterNodeHostOcpuCount": 1,
  "masterNodeHostType": "FLEX",
  "opendashboardNodeCount": 1,
  "opendashboardNodeHostMemoryGB": 8,
  "opendashboardNodeHostOcpuCount": 1,
  "securityMode": "DISABLED",
  "softwareVersion": "2.11.0",
  "subnetCompartmentId": "<ocid:1>",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:02:00Z",
  "vcnCompartmentId": "<ocid:1>",
  "vcnId": "<ocid:3>"
}`),
	}
	deletedReadStates := []opensearchsdk.OpensearchCluster{
		ocimock.MustOCIResponseFixture[opensearchsdk.OpensearchCluster](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-opensearch-v1-updated",
  "id": "<ocid:4>",
  "lifecycleState": "DELETING"
}`),
		ocimock.MustOCIResponseFixture[opensearchsdk.OpensearchCluster](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-opensearch-v1-updated",
  "id": "<ocid:4>",
  "lifecycleState": "DELETED"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		opensearchsdk.OpensearchCluster,
		opensearchsdk.CreateOpensearchClusterDetails,
		opensearchsdk.UpdateOpensearchClusterDetails,
	]{
		CollectionPath:    "/20180828/opensearchClusters",
		ItemPath:          "/20180828/opensearchClusters/<ocid:4>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: append(ocimock.LifecycleStates(t, createdState, "PROVISIONING"), createdReadStates...),
		UpdatedReadStates: append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      202,
		UpdateStatus:      202,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ opensearchsdk.CreateOpensearchClusterDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ opensearchsdk.OpensearchCluster) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20180828", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OpensearchCluster OCI mock: %v", err)
		}
	})
	sdkClient := opensearchsdk.OpensearchClusterClient{BaseClient: session.BaseClient()}
	client := newMockOpensearchClusterClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opensearchv1beta1.OpensearchCluster]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opensearchv1beta1.OpensearchCluster) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.SecurityMode, current.Spec.SecurityMode) ||
				!reflect.DeepEqual(current.Status.SoftwareVersion, current.Spec.SoftwareVersion) {
				return fmt.Errorf("created OpensearchCluster status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opensearchv1beta1.OpensearchCluster) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *opensearchv1beta1.OpensearchCluster) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated OpensearchCluster status = %+v", current.Status)
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
