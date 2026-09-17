/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sddc

import (
	"context"
	"fmt"
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationSddcLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &ocvpv1beta1.Sddc{Spec: ocvpv1beta1.SddcSpec{
		VmwareSoftwareVersion: "8.0.2",
		CompartmentId:         "ocid1.compartment.oc1..mock",
		HcxMode:               string(ocvpsdk.HcxModesDisabled),
		InitialConfiguration: ocvpv1beta1.SddcInitialConfiguration{
			InitialClusterConfigurations: []ocvpv1beta1.SddcInitialConfigurationInitialClusterConfiguration{
				{
					VsphereType:               string(ocvpsdk.VsphereTypesManagement),
					ComputeAvailabilityDomain: "US-ASHBURN-AD-1",
					EsxiHostsCount:            3,
					DisplayName:               "management",
					InitialCommitment:         string(ocvpsdk.CommitmentHour),
					InitialHostShapeName:      "BM.DenseIO.E5.128",
					InitialHostOcpuCount:      128,
					NetworkConfiguration: ocvpv1beta1.SddcInitialConfigurationInitialClusterConfigurationNetworkConfiguration{
						ProvisioningSubnetId: "ocid1.subnet.oc1..mock",
						VmotionVlanId:        "ocid1.vlan.oc1..vmotion",
						VsanVlanId:           "ocid1.vlan.oc1..vsan",
						NsxVTepVlanId:        "ocid1.vlan.oc1..nsxvtep",
						NsxEdgeVTepVlanId:    "ocid1.vlan.oc1..nsxedgevtep",
						VsphereVlanId:        "ocid1.vlan.oc1..vsphere",
						NsxEdgeUplink1VlanId: "ocid1.vlan.oc1..uplink1",
						NsxEdgeUplink2VlanId: "ocid1.vlan.oc1..uplink2",
					},
				},
			},
		},
		SshAuthorizedKeys: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
		DisplayName:       mockSddcName,
		FreeformTags:      map[string]string{"osok-mock": "create"},
	}}
	ocimock.InitializeResource(resource, "mock-sddc")
	resource.Spec = ocimock.MustJSONFixture[ocvpv1beta1.SddcSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-sddc",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hcxMode": "DISABLED",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "\u003cocid:2\u003e",
          "nsxEdgeUplink2VlanId": "\u003cocid:3\u003e",
          "nsxEdgeVTepVlanId": "\u003cocid:4\u003e",
          "nsxVTepVlanId": "\u003cocid:5\u003e",
          "provisioningSubnetId": "\u003cocid:6\u003e",
          "vmotionVlanId": "\u003cocid:7\u003e",
          "vsanVlanId": "\u003cocid:8\u003e",
          "vsphereVlanId": "\u003cocid:9\u003e"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "vmwareSoftwareVersion": "8.0.2"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-sddc-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[ocvpsdk.CreateSddcDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-sddc",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hcxMode": "DISABLED",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "\u003cocid:2\u003e",
          "nsxEdgeUplink2VlanId": "\u003cocid:3\u003e",
          "nsxEdgeVTepVlanId": "\u003cocid:4\u003e",
          "nsxVTepVlanId": "\u003cocid:5\u003e",
          "provisioningSubnetId": "\u003cocid:6\u003e",
          "vmotionVlanId": "\u003cocid:7\u003e",
          "vsanVlanId": "\u003cocid:8\u003e",
          "vsphereVlanId": "\u003cocid:9\u003e"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "vmwareSoftwareVersion": "8.0.2"
}`)
	createdState := ocimock.MustOCIResponseFixture[ocvpsdk.Sddc](t, `{
  "clustersCount": 1,
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "osok-mock-sddc",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hcxMode": "DISABLED",
  "id": "<ocid:10>",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "<ocid:2>",
          "nsxEdgeUplink2VlanId": "<ocid:3>",
          "nsxEdgeVTepVlanId": "<ocid:4>",
          "nsxVTepVlanId": "<ocid:5>",
          "provisioningSubnetId": "<ocid:6>",
          "vmotionVlanId": "<ocid:7>",
          "vsanVlanId": "<ocid:8>",
          "vsphereVlanId": "<ocid:9>"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "lifecycleState": "ACTIVE",
  "nsxManagerFqdn": "nsx.osok-mock.example.internal",
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:01:00Z",
  "vcenterFqdn": "vcenter.osok-mock.example.internal",
  "vmwareSoftwareVersion": "8.0.2"
}`)
	createdReadStates := []ocvpsdk.Sddc{
		ocimock.MustOCIResponseFixture[ocvpsdk.Sddc](t, `{
  "clustersCount": 1,
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "osok-mock-sddc",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hcxMode": "DISABLED",
  "id": "<ocid:10>",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "<ocid:2>",
          "nsxEdgeUplink2VlanId": "<ocid:3>",
          "nsxEdgeVTepVlanId": "<ocid:4>",
          "nsxVTepVlanId": "<ocid:5>",
          "provisioningSubnetId": "<ocid:6>",
          "vmotionVlanId": "<ocid:7>",
          "vsanVlanId": "<ocid:8>",
          "vsphereVlanId": "<ocid:9>"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "lifecycleState": "ACTIVE",
  "nsxManagerFqdn": "nsx.osok-mock.example.internal",
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:01:00Z",
  "vcenterFqdn": "vcenter.osok-mock.example.internal",
  "vmwareSoftwareVersion": "8.0.2"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[ocvpsdk.UpdateSddcDetails](t, `{
  "displayName": "osok-mock-sddc-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[ocvpsdk.Sddc](t, `{
  "clustersCount": 1,
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "osok-mock-sddc-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hcxMode": "DISABLED",
  "id": "<ocid:10>",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "<ocid:2>",
          "nsxEdgeUplink2VlanId": "<ocid:3>",
          "nsxEdgeVTepVlanId": "<ocid:4>",
          "nsxVTepVlanId": "<ocid:5>",
          "provisioningSubnetId": "<ocid:6>",
          "vmotionVlanId": "<ocid:7>",
          "vsanVlanId": "<ocid:8>",
          "vsphereVlanId": "<ocid:9>"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "lifecycleState": "ACTIVE",
  "nsxManagerFqdn": "nsx.osok-mock.example.internal",
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:02:00Z",
  "vcenterFqdn": "vcenter.osok-mock.example.internal",
  "vmwareSoftwareVersion": "8.0.2"
}`)
	updatedReadStates := []ocvpsdk.Sddc{
		ocimock.MustOCIResponseFixture[ocvpsdk.Sddc](t, `{
  "clustersCount": 1,
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "osok-mock-sddc-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hcxMode": "DISABLED",
  "id": "<ocid:10>",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "<ocid:2>",
          "nsxEdgeUplink2VlanId": "<ocid:3>",
          "nsxEdgeVTepVlanId": "<ocid:4>",
          "nsxVTepVlanId": "<ocid:5>",
          "provisioningSubnetId": "<ocid:6>",
          "vmotionVlanId": "<ocid:7>",
          "vsanVlanId": "<ocid:8>",
          "vsphereVlanId": "<ocid:9>"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "lifecycleState": "ACTIVE",
  "nsxManagerFqdn": "nsx.osok-mock.example.internal",
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:02:00Z",
  "vcenterFqdn": "vcenter.osok-mock.example.internal",
  "vmwareSoftwareVersion": "8.0.2"
}`),
	}
	deletedReadStates := []ocvpsdk.Sddc{
		ocimock.MustOCIResponseFixture[ocvpsdk.Sddc](t, `{
  "clustersCount": 1,
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "osok-mock-sddc-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hcxMode": "DISABLED",
  "id": "<ocid:10>",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "<ocid:2>",
          "nsxEdgeUplink2VlanId": "<ocid:3>",
          "nsxEdgeVTepVlanId": "<ocid:4>",
          "nsxVTepVlanId": "<ocid:5>",
          "provisioningSubnetId": "<ocid:6>",
          "vmotionVlanId": "<ocid:7>",
          "vsanVlanId": "<ocid:8>",
          "vsphereVlanId": "<ocid:9>"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "lifecycleState": "DELETING",
  "nsxManagerFqdn": "nsx.osok-mock.example.internal",
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:03:00Z",
  "vcenterFqdn": "vcenter.osok-mock.example.internal",
  "vmwareSoftwareVersion": "8.0.2"
}`),
		ocimock.MustOCIResponseFixture[ocvpsdk.Sddc](t, `{
  "clustersCount": 1,
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "osok-mock-sddc-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hcxMode": "DISABLED",
  "id": "<ocid:10>",
  "initialConfiguration": {
    "initialClusterConfigurations": [
      {
        "computeAvailabilityDomain": "US-ASHBURN-AD-1",
        "displayName": "management",
        "esxiHostsCount": 3,
        "initialCommitment": "HOUR",
        "initialHostOcpuCount": 128,
        "initialHostShapeName": "BM.DenseIO.E5.128",
        "networkConfiguration": {
          "nsxEdgeUplink1VlanId": "<ocid:2>",
          "nsxEdgeUplink2VlanId": "<ocid:3>",
          "nsxEdgeVTepVlanId": "<ocid:4>",
          "nsxVTepVlanId": "<ocid:5>",
          "provisioningSubnetId": "<ocid:6>",
          "vmotionVlanId": "<ocid:7>",
          "vsanVlanId": "<ocid:8>",
          "vsphereVlanId": "<ocid:9>"
        },
        "vsphereType": "MANAGEMENT"
      }
    ]
  },
  "lifecycleState": "DELETED",
  "nsxManagerFqdn": "nsx.osok-mock.example.internal",
  "sshAuthorizedKeys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockOnlyKey osok-mock",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:04:00Z",
  "vcenterFqdn": "vcenter.osok-mock.example.internal",
  "vmwareSoftwareVersion": "8.0.2"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		ocvpsdk.Sddc,
		ocvpsdk.CreateSddcDetails,
		ocvpsdk.UpdateSddcDetails,
	]{
		CollectionPath:    "/20230701/sddcs",
		ItemPath:          "/20230701/sddcs/<ocid:10>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: append(ocimock.LifecycleStates(t, createdState, "CREATING", "UPDATING"), createdReadStates...),
		UpdatedReadStates: append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      202,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ ocvpsdk.CreateSddcDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ ocvpsdk.Sddc) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20230701", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Sddc OCI mock: %v", err)
		}
	})
	sdkClient := ocvpsdk.SddcClient{BaseClient: session.BaseClient()}
	manager := newMockSddcManager(sdkClient)
	client := manager.client
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*ocvpv1beta1.Sddc]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *ocvpv1beta1.Sddc) error {
			if current.Status.Id != "<ocid:10>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:10>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.HcxMode, current.Spec.HcxMode) ||
				!reflect.DeepEqual(current.Status.VmwareSoftwareVersion, current.Spec.VmwareSoftwareVersion) ||
				current.Status.ClustersCount != 1 ||
				current.Status.VcenterFqdn == "" ||
				current.Status.NsxManagerFqdn == "" {
				return fmt.Errorf("created Sddc status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *ocvpv1beta1.Sddc) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *ocvpv1beta1.Sddc) error {
			if current.Status.Id != "<ocid:10>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:10>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated Sddc status = %+v", current.Status)
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
