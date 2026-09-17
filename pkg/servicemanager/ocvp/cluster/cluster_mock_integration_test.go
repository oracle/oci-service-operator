/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package cluster

import (
	"context"
	"fmt"
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationClusterLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newClusterTestResource()
	ocimock.InitializeResource(resource, "mock-cluster")
	resource.Spec = ocimock.MustJSONFixture[ocvpv1beta1.ClusterSpec](t, `{
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "displayName": "cluster-sample",
  "esxiHostsCount": 3,
  "freeformTags": {
    "env": "dev"
  },
  "isShieldedInstanceEnabled": true,
  "networkConfiguration": {
    "nsxEdgeVTepVlanId": "\u003cocid:1\u003e",
    "nsxVTepVlanId": "\u003cocid:2\u003e",
    "provisioningSubnetId": "\u003cocid:3\u003e",
    "vmotionVlanId": "\u003cocid:4\u003e",
    "vsanVlanId": "\u003cocid:5\u003e"
  },
  "sddcId": "\u003cocid:6\u003e"
}`)
	updatedSpec := resource.Spec
	updatedSpec.DisplayName = "cluster-sample-updated"
	createRequest := ocimock.MustJSONFixture[ocvpsdk.CreateClusterDetails](t, `{
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "displayName": "cluster-sample",
  "esxiHostsCount": 3,
  "freeformTags": {
    "env": "dev"
  },
  "isShieldedInstanceEnabled": true,
  "networkConfiguration": {
    "nsxEdgeVTepVlanId": "\u003cocid:1\u003e",
    "nsxVTepVlanId": "\u003cocid:2\u003e",
    "provisioningSubnetId": "\u003cocid:3\u003e",
    "vmotionVlanId": "\u003cocid:4\u003e",
    "vsanVlanId": "\u003cocid:5\u003e"
  },
  "sddcId": "\u003cocid:6\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[ocvpsdk.Cluster](t, `{
  "compartmentId": "\u003cocid:7\u003e",
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "displayName": "cluster-sample",
  "esxiHostsCount": 3,
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:8\u003e",
  "isShieldedInstanceEnabled": true,
  "lifecycleState": "ACTIVE",
  "networkConfiguration": {
    "nsxEdgeVTepVlanId": "\u003cocid:1\u003e",
    "nsxVTepVlanId": "\u003cocid:2\u003e",
    "provisioningSubnetId": "\u003cocid:3\u003e",
    "vmotionVlanId": "\u003cocid:4\u003e",
    "vsanVlanId": "\u003cocid:5\u003e"
  },
  "sddcId": "\u003cocid:6\u003e"
}`)
	createdReadStates := []ocvpsdk.Cluster{
		ocimock.MustOCIResponseFixture[ocvpsdk.Cluster](t, `{
  "compartmentId": "\u003cocid:7\u003e",
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "displayName": "cluster-sample",
  "esxiHostsCount": 3,
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:8\u003e",
  "isShieldedInstanceEnabled": true,
  "lifecycleState": "ACTIVE",
  "networkConfiguration": {
    "nsxEdgeVTepVlanId": "\u003cocid:1\u003e",
    "nsxVTepVlanId": "\u003cocid:2\u003e",
    "provisioningSubnetId": "\u003cocid:3\u003e",
    "vmotionVlanId": "\u003cocid:4\u003e",
    "vsanVlanId": "\u003cocid:5\u003e"
  },
  "sddcId": "\u003cocid:6\u003e"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[ocvpsdk.UpdateClusterDetails](t, `{
  "displayName": "cluster-sample-updated"
}`)
	updatedState := createdState
	updatedState.DisplayName = updateRequest.DisplayName
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		ocvpsdk.Cluster,
		ocvpsdk.CreateClusterDetails,
		ocvpsdk.UpdateClusterDetails,
	]{
		CollectionPath:     "/20230701/clusters",
		ItemPath:           "/20230701/clusters/<ocid:8>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		CreatedReadStates:  append(ocimock.LifecycleStates(t, createdState, "CREATING", "UPDATING"), createdReadStates...),
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		UpdatedReadStates:  append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), []ocvpsdk.Cluster{updatedState}...),
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ ocvpsdk.CreateClusterDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ ocvpsdk.Cluster) error {
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
			t.Errorf("close Cluster OCI mock: %v", err)
		}
	})
	sdkClient := ocvpsdk.ClusterClient{BaseClient: session.BaseClient()}
	manager := &ClusterServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newClusterRuntimeHooks(manager, sdkClient)
	client := wrapClusterGeneratedClient(hooks, defaultClusterServiceClient{ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.Cluster](buildClusterGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*ocvpv1beta1.Cluster]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *ocvpv1beta1.Cluster) error {
			if current.Status.Id != "<ocid:8>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:8>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ComputeAvailabilityDomain, current.Spec.ComputeAvailabilityDomain) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.EsxiHostsCount, current.Spec.EsxiHostsCount) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsShieldedInstanceEnabled, current.Spec.IsShieldedInstanceEnabled) ||
				!reflect.DeepEqual(current.Status.SddcId, current.Spec.SddcId) {
				return fmt.Errorf("created Cluster status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *ocvpv1beta1.Cluster) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *ocvpv1beta1.Cluster) error {
			if current.Status.Id != "<ocid:8>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.SddcId, current.Spec.SddcId) {
				return fmt.Errorf("updated Cluster status = %+v", current.Status)
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
