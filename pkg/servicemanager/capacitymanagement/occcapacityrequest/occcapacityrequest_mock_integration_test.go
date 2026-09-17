/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package occcapacityrequest

import (
	"context"
	"fmt"
	capacitymanagementsdk "github.com/oracle/oci-go-sdk/v65/capacitymanagement"
	capacitymanagementv1beta1 "github.com/oracle/oci-service-operator/api/capacitymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOccCapacityRequestLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newOccCapacityRequestTestResource()
	ocimock.InitializeResource(resource, "mock-occcapacityrequest")
	resource.Spec = ocimock.MustJSONFixture[capacitymanagementv1beta1.OccCapacityRequestSpec](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "\u003cocid:1\u003e",
  "dateExpectedCapacityHandover": "2026-06-01T00:00:00Z",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial request",
  "details": [
    {
      "demandQuantity": 2,
      "resourceName": "BM.Standard.E5.192",
      "resourceType": "SERVER",
      "workloadType": "GENERIC"
    }
  ],
  "displayName": "capacity-request",
  "freeformTags": {
    "env": "dev"
  },
  "namespace": "COMPUTE",
  "occAvailabilityCatalogId": "\u003cocid:2\u003e",
  "region": "us-phoenix-1",
  "requestState": "SUBMITTED",
  "requestType": "NEW"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "84"
    }
  },
  "displayName": "capacity-request-updated",
  "freeformTags": {
    "env": "prod"
  },
  "requestState": "CANCELLED"
}`)
	createRequest := ocimock.MustJSONFixture[capacitymanagementsdk.CreateOccCapacityRequestDetails](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "\u003cocid:1\u003e",
  "dateExpectedCapacityHandover": "2026-06-01T00:00:00Z",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial request",
  "details": [
    {
      "demandQuantity": 2,
      "resourceName": "BM.Standard.E5.192",
      "resourceType": "SERVER",
      "workloadType": "GENERIC"
    }
  ],
  "displayName": "capacity-request",
  "freeformTags": {
    "env": "dev"
  },
  "namespace": "COMPUTE",
  "occAvailabilityCatalogId": "\u003cocid:2\u003e",
  "region": "us-phoenix-1",
  "requestState": "SUBMITTED",
  "requestType": "NEW"
}`)
	createdState := ocimock.MustOCIResponseFixture[capacitymanagementsdk.OccCapacityRequest](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "\u003cocid:1\u003e",
  "dateExpectedCapacityHandover": "2026-06-01T00:00:00Z",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial request",
  "details": [
    {
      "demandQuantity": 2,
      "resourceName": "BM.Standard.E5.192",
      "resourceType": "SERVER",
      "workloadType": "GENERIC"
    }
  ],
  "displayName": "capacity-request",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "namespace": "COMPUTE",
  "occAvailabilityCatalogId": "\u003cocid:2\u003e",
  "region": "us-phoenix-1",
  "requestState": "SUBMITTED",
  "requestType": "NEW"
}`)
	updateRequest := ocimock.MustJSONFixture[capacitymanagementsdk.UpdateOccCapacityRequestDetails](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "84"
    }
  },
  "displayName": "capacity-request-updated",
  "freeformTags": {
    "env": "prod"
  },
  "requestState": "CANCELLED"
}`)
	updatedState := createdState
	updatedState.DefinedTags = updateRequest.DefinedTags
	updatedState.DisplayName = updateRequest.DisplayName
	updatedState.FreeformTags = updateRequest.FreeformTags
	updatedState.RequestState = capacitymanagementsdk.OccCapacityRequestRequestStateCancelled
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		capacitymanagementsdk.OccCapacityRequest,
		capacitymanagementsdk.CreateOccCapacityRequestDetails,
		capacitymanagementsdk.UpdateOccCapacityRequestDetails,
	]{
		CollectionPath:     "/20231107/occCapacityRequests",
		ItemPath:           "/20231107/occCapacityRequests/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		UpdatedReadStates:  append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), []capacitymanagementsdk.OccCapacityRequest{updatedState}...),
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ capacitymanagementsdk.CreateOccCapacityRequestDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ capacitymanagementsdk.OccCapacityRequest) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20231107", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OccCapacityRequest OCI mock: %v", err)
		}
	})
	sdkClient := capacitymanagementsdk.CapacityManagementClient{BaseClient: session.BaseClient()}
	client := newOccCapacityRequestServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*capacitymanagementv1beta1.OccCapacityRequest]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *capacitymanagementv1beta1.OccCapacityRequest) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AvailabilityDomain, current.Spec.AvailabilityDomain) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DateExpectedCapacityHandover, current.Spec.DateExpectedCapacityHandover) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Details, current.Spec.Details) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Namespace, current.Spec.Namespace) ||
				!reflect.DeepEqual(current.Status.OccAvailabilityCatalogId, current.Spec.OccAvailabilityCatalogId) ||
				!reflect.DeepEqual(current.Status.Region, current.Spec.Region) ||
				!reflect.DeepEqual(current.Status.RequestState, current.Spec.RequestState) ||
				!reflect.DeepEqual(current.Status.RequestType, current.Spec.RequestType) {
				return fmt.Errorf("created OccCapacityRequest status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *capacitymanagementv1beta1.OccCapacityRequest) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *capacitymanagementv1beta1.OccCapacityRequest) error {
			if current.Status.Id != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.RequestState, current.Spec.RequestState) {
				return fmt.Errorf("updated OccCapacityRequest status = %+v", current.Status)
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
