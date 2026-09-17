/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package limitsincreaserequest

import (
	"context"
	"fmt"
	limitsincreasesdk "github.com/oracle/oci-go-sdk/v65/limitsincrease"
	limitsincreasev1beta1 "github.com/oracle/oci-service-operator/api/limitsincrease/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLimitsIncreaseRequestLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeLimitsIncreaseRequestResource()
	ocimock.InitializeResource(resource, "mock-limitsincreaserequest")
	resource.Spec = ocimock.MustJSONFixture[limitsincreasev1beta1.LimitsIncreaseRequestSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "limits-request",
  "freeformTags": {
    "managed-by": "osok"
  },
  "justification": "need more capacity",
  "limitsIncreaseItemRequests": [
    {
      "limitName": "volume-count",
      "region": "us-ashburn-1",
      "serviceName": "blockstorage",
      "value": 10
    },
    {
      "limitName": "vm-count",
      "questionnaireResponse": [
        {
          "id": "question-1",
          "questionResponse": "first"
        },
        {
          "id": "question-2",
          "questionResponse": "second"
        }
      ],
      "region": "us-ashburn-1",
      "scope": "AD-1",
      "serviceName": "compute",
      "value": 20
    }
  ],
  "subscriptionId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "managed-by": "osok",
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[limitsincreasesdk.CreateLimitsIncreaseRequestDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "limits-request",
  "freeformTags": {
    "managed-by": "osok"
  },
  "justification": "need more capacity",
  "limitsIncreaseItemRequests": [
    {
      "limitName": "volume-count",
      "region": "us-ashburn-1",
      "serviceName": "blockstorage",
      "value": 10
    },
    {
      "limitName": "vm-count",
      "questionnaireResponse": [
        {
          "id": "question-1",
          "questionResponse": "first"
        },
        {
          "id": "question-2",
          "questionResponse": "second"
        }
      ],
      "region": "us-ashburn-1",
      "scope": "AD-1",
      "serviceName": "compute",
      "value": 20
    }
  ],
  "subscriptionId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[limitsincreasesdk.LimitsIncreaseRequest](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "limits-request",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "\u003cocid:3\u003e",
  "justification": "need more capacity",
  "lifecycleState": "SUCCEEDED",
  "limitsIncreaseItemRequests": [
    {
      "limitName": "vm-count",
      "questionnaireResponse": [
        {
          "id": "question-2",
          "questionResponse": "second"
        },
        {
          "id": "question-1",
          "questionResponse": "first"
        }
      ],
      "region": "us-ashburn-1",
      "scope": "AD-1",
      "serviceName": "compute",
      "value": 20
    },
    {
      "limitName": "volume-count",
      "region": "us-ashburn-1",
      "serviceName": "blockstorage",
      "value": 10
    }
  ],
  "subscriptionId": "\u003cocid:2\u003e"
}`)
	createdReadStates := []limitsincreasesdk.LimitsIncreaseRequest{
		ocimock.MustOCIResponseFixture[limitsincreasesdk.LimitsIncreaseRequest](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "limits-request",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "\u003cocid:3\u003e",
  "justification": "need more capacity",
  "lifecycleState": "SUCCEEDED",
  "limitsIncreaseItemRequests": [
    {
      "limitName": "vm-count",
      "questionnaireResponse": [
        {
          "id": "question-2",
          "questionResponse": "second"
        },
        {
          "id": "question-1",
          "questionResponse": "first"
        }
      ],
      "region": "us-ashburn-1",
      "scope": "AD-1",
      "serviceName": "compute",
      "value": 20
    },
    {
      "limitName": "volume-count",
      "region": "us-ashburn-1",
      "serviceName": "blockstorage",
      "value": 10
    }
  ],
  "subscriptionId": "\u003cocid:2\u003e"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[limitsincreasesdk.UpdateLimitsIncreaseRequestDetails](t, `{
  "freeformTags": {
    "managed-by": "osok",
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[limitsincreasesdk.LimitsIncreaseRequest](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "limits-request",
  "freeformTags": {
    "managed-by": "osok",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:3\u003e",
  "justification": "need more capacity",
  "lifecycleState": "SUCCEEDED",
  "limitsIncreaseItemRequests": [
    {
      "limitName": "vm-count",
      "questionnaireResponse": [
        {
          "id": "question-2",
          "questionResponse": "second"
        },
        {
          "id": "question-1",
          "questionResponse": "first"
        }
      ],
      "region": "us-ashburn-1",
      "scope": "AD-1",
      "serviceName": "compute",
      "value": 20
    },
    {
      "limitName": "volume-count",
      "region": "us-ashburn-1",
      "serviceName": "blockstorage",
      "value": 10
    }
  ],
  "subscriptionId": "\u003cocid:2\u003e"
}`)
	updatedReadStates := []limitsincreasesdk.LimitsIncreaseRequest{
		ocimock.MustOCIResponseFixture[limitsincreasesdk.LimitsIncreaseRequest](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "limits-request",
  "freeformTags": {
    "managed-by": "osok",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:3\u003e",
  "justification": "need more capacity",
  "lifecycleState": "SUCCEEDED",
  "limitsIncreaseItemRequests": [
    {
      "limitName": "vm-count",
      "questionnaireResponse": [
        {
          "id": "question-2",
          "questionResponse": "second"
        },
        {
          "id": "question-1",
          "questionResponse": "first"
        }
      ],
      "region": "us-ashburn-1",
      "scope": "AD-1",
      "serviceName": "compute",
      "value": 20
    },
    {
      "limitName": "volume-count",
      "region": "us-ashburn-1",
      "serviceName": "blockstorage",
      "value": 10
    }
  ],
  "subscriptionId": "\u003cocid:2\u003e"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		limitsincreasesdk.LimitsIncreaseRequest,
		limitsincreasesdk.CreateLimitsIncreaseRequestDetails,
		limitsincreasesdk.UpdateLimitsIncreaseRequestDetails,
	]{
		CollectionPath:     "/20251101/limitsIncreaseRequests",
		ItemPath:           "/20251101/limitsIncreaseRequests/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ limitsincreasesdk.CreateLimitsIncreaseRequestDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ limitsincreasesdk.LimitsIncreaseRequest) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20251101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close LimitsIncreaseRequest OCI mock: %v", err)
		}
	})
	sdkClient := limitsincreasesdk.LimitsIncreaseClient{BaseClient: session.BaseClient()}
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{createFn: sdkClient.CreateLimitsIncreaseRequest, getFn: sdkClient.GetLimitsIncreaseRequest, listFn: sdkClient.ListLimitsIncreaseRequests, updateFn: sdkClient.UpdateLimitsIncreaseRequest, deleteFn: sdkClient.DeleteLimitsIncreaseRequest})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*limitsincreasev1beta1.LimitsIncreaseRequest]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *limitsincreasev1beta1.LimitsIncreaseRequest) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "SUCCEEDED" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Justification, current.Spec.Justification) ||
				!reflect.DeepEqual(current.Status.SubscriptionId, current.Spec.SubscriptionId) ||
				len(current.Status.LimitsIncreaseItemRequests) != 2 ||
				current.Status.LimitsIncreaseItemRequests[0].LimitName != "volume-count" ||
				current.Status.LimitsIncreaseItemRequests[0].Value != 10 ||
				current.Status.LimitsIncreaseItemRequests[1].LimitName != "vm-count" ||
				current.Status.LimitsIncreaseItemRequests[1].Value != 20 {
				return fmt.Errorf("created LimitsIncreaseRequest status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *limitsincreasev1beta1.LimitsIncreaseRequest) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *limitsincreasev1beta1.LimitsIncreaseRequest) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "SUCCEEDED" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated LimitsIncreaseRequest status = %+v", current.Status)
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
