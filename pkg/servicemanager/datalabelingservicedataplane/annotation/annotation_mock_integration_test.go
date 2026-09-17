/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package annotation

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/datalabelingservicedataplane"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/datalabelingservicedataplane/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationAnnotationCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Annotation](t, `
{
  "metadata": {"name": "mock-annotation", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "entities": [
    {
      "boundingPolygon": {
        "normalizedVertices": [
          {
            "x": 1,
            "y": 1
          }
        ]
      },
      "entityType": "IMAGEOBJECTSELECTION",
      "labels": [
        {
          "label": "mock-label"
        }
      ]
    }
  ],
  "freeformTags": {
    "mock": "initial"
  },
  "recordId": "<ocid:required>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-annotation")
	resource.Status = apiv1beta1.AnnotationStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateAnnotationDetails](t, `{
  "compartmentId": "<ocid:required>",
  "entities": [
    {
      "boundingPolygon": {
        "normalizedVertices": [
          {
            "x": 1,
            "y": 1
          }
        ]
      },
      "entityType": "IMAGEOBJECTSELECTION",
      "labels": [
        {
          "label": "mock-label"
        }
      ]
    }
  ],
  "freeformTags": {
    "mock": "initial"
  },
  "recordId": "<ocid:required>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateAnnotationDetails](t, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Annotation](t, `{
  "compartmentId": "<ocid:required>",
  "entities": [
    {
      "boundingPolygon": {
        "normalizedVertices": [
          {
            "x": 1,
            "y": 1
          }
        ]
      },
      "entityType": "IMAGEOBJECTSELECTION",
      "labels": [
        {
          "label": "mock-label"
        }
      ]
    }
  ],
  "freeformTags": {
    "mock": "initial"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "recordId": "<ocid:required>",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Annotation](t, `{
  "compartmentId": "<ocid:required>",
  "entities": [
    {
      "boundingPolygon": {
        "normalizedVertices": [
          {
            "x": 1,
            "y": 1
          }
        ]
      },
      "entityType": "IMAGEOBJECTSELECTION",
      "labels": [
        {
          "label": "mock-label"
        }
      ]
    }
  ],
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "recordId": "<ocid:required>",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	updatingState := updatedState
	updatingState.LifecycleState = "UPDATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Annotation, sdksvc.CreateAnnotationDetails, sdksvc.UpdateAnnotationDetails]{
		CollectionPath: "/20211001/annotations", ItemPath: "/20211001/annotations/<ocid:1>",
		CreatePath: "/20211001/annotations", CreateMethod: http.MethodPost,
		UpdatePath: "/20211001/annotations/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20211001/annotations/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateAnnotationDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateAnnotationDetails],
		UpdatedState:      &updatingState,
		UpdatedReadStates: ocimock.StateSequence(updatingState, updatedState),
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateAnnotationDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datalabeling-dp.us-ashburn-1.oci.oraclecloud.com", BasePath: "20211001", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DataLabelingClient{BaseClient: session.BaseClient()}
	manager := &AnnotationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAnnotationRuntimeHooks(manager, sdkClient)
	client := wrapAnnotationGeneratedClient(hooks, defaultAnnotationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Annotation](buildAnnotationGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Annotation]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Annotation) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.FreeformTags["mock"] != "initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Annotation status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Annotation) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Annotation) error {
			if current.Status.FreeformTags["mock"] != "updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Annotation status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.Annotation) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable Annotation update calls = %d, want 1", got)
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
