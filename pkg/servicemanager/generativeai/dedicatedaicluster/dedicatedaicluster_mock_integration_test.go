/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dedicatedaicluster

import (
	"context"
	"fmt"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDedicatedAiClusterLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &generativeaiv1beta1.DedicatedAiCluster{
		Spec: generativeaiv1beta1.DedicatedAiClusterSpec{
			Type:          string(generativeaisdk.DedicatedAiClusterTypeHosting),
			CompartmentId: "ocid1.compartment.oc1..mock",
			UnitCount:     1,
			UnitShape: string(
				generativeaisdk.DedicatedAiClusterUnitShapeSmallCohere,
			),
			DisplayName: mockDedicatedAiClusterName,
			Description: "synthetic create",
			FreeformTags: map[string]string{
				"osok-mock": "create",
			},
		},
	}
	ocimock.InitializeResource(resource, "mock-dedicatedaicluster")
	resource.Spec = ocimock.MustJSONFixture[generativeaiv1beta1.DedicatedAiClusterSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "type": "HOSTING",
  "unitCount": 1,
  "unitShape": "SMALL_COHERE"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "synthetic update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "unitCount": 2
}`)
	createRequest := ocimock.MustJSONFixture[generativeaisdk.CreateDedicatedAiClusterDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "type": "HOSTING",
  "unitCount": 1,
  "unitShape": "SMALL_COHERE"
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaisdk.DedicatedAiCluster](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-08-31T12:00:00Z",
  "type": "HOSTING",
  "unitCount": 1,
  "unitShape": "SMALL_COHERE"
}`)
	createdReadStates := []generativeaisdk.DedicatedAiCluster{
		ocimock.MustOCIResponseFixture[generativeaisdk.DedicatedAiCluster](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-08-31T12:00:00Z",
  "type": "HOSTING",
  "unitCount": 1,
  "unitShape": "SMALL_COHERE"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[generativeaisdk.UpdateDedicatedAiClusterDetails](t, `{
  "description": "synthetic update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "unitCount": 2
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaisdk.DedicatedAiCluster](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z",
  "type": "HOSTING",
  "unitCount": 2,
  "unitShape": "SMALL_COHERE"
}`)
	updatedReadStates := []generativeaisdk.DedicatedAiCluster{
		ocimock.MustOCIResponseFixture[generativeaisdk.DedicatedAiCluster](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z",
  "type": "HOSTING",
  "unitCount": 2,
  "unitShape": "SMALL_COHERE"
}`),
	}
	deletedReadStates := []generativeaisdk.DedicatedAiCluster{
		ocimock.MustOCIResponseFixture[generativeaisdk.DedicatedAiCluster](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETING",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z",
  "type": "HOSTING",
  "unitCount": 2,
  "unitShape": "SMALL_COHERE"
}`),
		ocimock.MustOCIResponseFixture[generativeaisdk.DedicatedAiCluster](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-ai-cluster-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETED",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z",
  "type": "HOSTING",
  "unitCount": 2,
  "unitShape": "SMALL_COHERE"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		generativeaisdk.DedicatedAiCluster,
		generativeaisdk.CreateDedicatedAiClusterDetails,
		generativeaisdk.UpdateDedicatedAiClusterDetails,
	]{
		CollectionPath:    "/20231130/dedicatedAiClusters",
		ItemPath:          "/20231130/dedicatedAiClusters/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...),
		UpdatedReadStates: append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ generativeaisdk.CreateDedicatedAiClusterDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ generativeaisdk.DedicatedAiCluster) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20231130", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DedicatedAiCluster OCI mock: %v", err)
		}
	})
	sdkClient := generativeaisdk.GenerativeAiClient{BaseClient: session.BaseClient()}
	client := newMockDedicatedAiClusterClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiv1beta1.DedicatedAiCluster]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiv1beta1.DedicatedAiCluster) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) ||
				!reflect.DeepEqual(current.Status.UnitCount, current.Spec.UnitCount) ||
				!reflect.DeepEqual(current.Status.UnitShape, current.Spec.UnitShape) {
				return fmt.Errorf("created DedicatedAiCluster status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiv1beta1.DedicatedAiCluster) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiv1beta1.DedicatedAiCluster) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.UnitCount, current.Spec.UnitCount) {
				return fmt.Errorf("updated DedicatedAiCluster status = %+v", current.Status)
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
