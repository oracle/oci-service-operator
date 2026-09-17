/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package esxihost

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
func TestMockIntegrationEsxiHostLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newEsxiHostTestResource()
	ocimock.InitializeResource(resource, "mock-esxihost")
	resource.Spec = ocimock.MustJSONFixture[ocvpv1beta1.EsxiHostSpec](t, `{
  "billingDonorHostId": "\u003cocid:1\u003e",
  "capacityReservationId": "\u003cocid:2\u003e",
  "clusterId": "\u003cocid:3\u003e",
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "currentCommitment": "MONTH",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "esxihost-sample",
  "esxiSoftwareVersion": "7.0.0",
  "freeformTags": {
    "env": "dev"
  },
  "hostOcpuCount": 32,
  "hostShapeName": "BM.DenseIO2.52",
  "nextCommitment": "MONTH"
}`)
	updatedSpec := resource.Spec
	updatedSpec.DisplayName = "esxihost-sample-updated"
	updatedSpec.NextCommitment = string(ocvpsdk.CommitmentOneYear)
	createRequest := ocimock.MustJSONFixture[ocvpsdk.CreateEsxiHostDetails](t, `{
  "billingDonorHostId": "\u003cocid:1\u003e",
  "capacityReservationId": "\u003cocid:2\u003e",
  "clusterId": "\u003cocid:3\u003e",
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "currentCommitment": "MONTH",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "esxihost-sample",
  "esxiSoftwareVersion": "7.0.0",
  "freeformTags": {
    "env": "dev"
  },
  "hostOcpuCount": 32,
  "hostShapeName": "BM.DenseIO2.52",
  "nextCommitment": "MONTH"
}`)
	createdState := ocimock.MustOCIResponseFixture[ocvpsdk.EsxiHost](t, `{
  "billingDonorHostId": "\u003cocid:1\u003e",
  "capacityReservationId": "\u003cocid:2\u003e",
  "clusterId": "\u003cocid:3\u003e",
  "compartmentId": "\u003cocid:4\u003e",
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "currentCommitment": "MONTH",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "esxihost-sample",
  "esxiSoftwareVersion": "7.0.0",
  "freeformTags": {
    "env": "dev"
  },
  "hostOcpuCount": 32,
  "hostShapeName": "BM.DenseIO2.52",
  "id": "\u003cocid:5\u003e",
  "lifecycleState": "ACTIVE",
  "nextCommitment": "MONTH"
}`)
	createdReadStates := []ocvpsdk.EsxiHost{
		ocimock.MustOCIResponseFixture[ocvpsdk.EsxiHost](t, `{
  "billingDonorHostId": "\u003cocid:1\u003e",
  "capacityReservationId": "\u003cocid:2\u003e",
  "clusterId": "\u003cocid:3\u003e",
  "compartmentId": "\u003cocid:4\u003e",
  "computeAvailabilityDomain": "Uocm:PHX-AD-1",
  "currentCommitment": "MONTH",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "esxihost-sample",
  "esxiSoftwareVersion": "7.0.0",
  "freeformTags": {
    "env": "dev"
  },
  "hostOcpuCount": 32,
  "hostShapeName": "BM.DenseIO2.52",
  "id": "\u003cocid:5\u003e",
  "lifecycleState": "ACTIVE",
  "nextCommitment": "MONTH"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[ocvpsdk.UpdateEsxiHostDetails](t, `{
  "displayName": "esxihost-sample-updated",
  "nextCommitment": "ONE_YEAR"
}`)
	updatedState := createdState
	updatedState.DisplayName = updateRequest.DisplayName
	updatedState.NextCommitment = updateRequest.NextCommitment
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		ocvpsdk.EsxiHost,
		ocvpsdk.CreateEsxiHostDetails,
		ocvpsdk.UpdateEsxiHostDetails,
	]{
		CollectionPath:     "/20230701/esxiHosts",
		ItemPath:           "/20230701/esxiHosts/<ocid:5>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		CreatedReadStates:  createdReadStates,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		UpdatedReadStates:  []ocvpsdk.EsxiHost{updatedState},
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ ocvpsdk.CreateEsxiHostDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ ocvpsdk.EsxiHost) error {
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
			t.Errorf("close EsxiHost OCI mock: %v", err)
		}
	})
	sdkClient := ocvpsdk.EsxiHostClient{BaseClient: session.BaseClient()}
	manager := &EsxiHostServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newEsxiHostRuntimeHooks(manager, sdkClient)
	client := wrapEsxiHostGeneratedClient(hooks, defaultEsxiHostServiceClient{ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.EsxiHost](buildEsxiHostGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*ocvpv1beta1.EsxiHost]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *ocvpv1beta1.EsxiHost) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.BillingDonorHostId, current.Spec.BillingDonorHostId) ||
				!reflect.DeepEqual(current.Status.ClusterId, current.Spec.ClusterId) ||
				!reflect.DeepEqual(current.Status.ComputeAvailabilityDomain, current.Spec.ComputeAvailabilityDomain) ||
				!reflect.DeepEqual(current.Status.CurrentCommitment, current.Spec.CurrentCommitment) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.HostOcpuCount, current.Spec.HostOcpuCount) ||
				!reflect.DeepEqual(current.Status.HostShapeName, current.Spec.HostShapeName) ||
				!reflect.DeepEqual(current.Status.NextCommitment, current.Spec.NextCommitment) {
				return fmt.Errorf("created EsxiHost status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *ocvpv1beta1.EsxiHost) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *ocvpv1beta1.EsxiHost) error {
			if current.Status.Id != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.NextCommitment, current.Spec.NextCommitment) ||
				!reflect.DeepEqual(current.Status.ClusterId, current.Spec.ClusterId) {
				return fmt.Errorf("updated EsxiHost status = %+v", current.Status)
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
