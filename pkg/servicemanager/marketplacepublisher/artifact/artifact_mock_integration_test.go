/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package artifact

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationArtifactWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[marketplacepublisherv1beta1.Artifact](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "artifact",
    "namespace": "default",
    "uid": "artifact-uid"
  },
  "spec": {
    "artifactType": "CONTAINER_IMAGE",
    "compartmentId": "<ocid:1>",
    "containerImage": {
      "sourceRegistryId": "<ocid:2>",
      "sourceRegistryUrl": "iad.ocir.io/example/image:1.0.0"
    },
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "displayName": "artifact",
    "freeformTags": {
      "env": "dev"
    }
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-artifact")
	resource.Status = marketplacepublisherv1beta1.ArtifactStatus{}
	createRequest := ocimock.MustJSONFixture[marketplacepublishersdk.CreateContainerImageArtifactDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "containerImage": {
    "sourceRegistryId": "<ocid:2>",
    "sourceRegistryUrl": "iad.ocir.io/example/image:1.0.0"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "artifact",
  "freeformTags": {
    "env": "dev"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[marketplacepublishersdk.UpdateContainerImageArtifactDetails](t, `
{
  "containerImage": {
    "sourceRegistryId": "<ocid:2>",
    "sourceRegistryUrl": "iad.ocir.io/example/image:1.0.0"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "artifact-updated",
  "freeformTags": {
    "env": "dev"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.ContainerImageArtifact](t, `
{
  "artifactType": "CONTAINER_IMAGE",
  "compartmentId": "<ocid:1>",
  "containerImage": {
    "publicationError": null,
    "publicationStatus": "PUBLICATION_COMPLETED",
    "sourceRegistryId": "<ocid:2>",
    "sourceRegistryUrl": "iad.ocir.io/example/image:1.0.0",
    "validationError": null,
    "validationStatus": "VALIDATION_COMPLETED"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "artifact",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "publisherId": "<ocid:5>",
  "status": "AVAILABLE",
  "statusNotes": null,
  "systemTags": null,
  "timeCreated": "2026-04-29T12:00:00Z",
  "timeUpdated": "2026-04-29T12:00:00Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.ContainerImageArtifact](t, `
{
  "artifactType": "CONTAINER_IMAGE",
  "compartmentId": "<ocid:1>",
  "containerImage": {
    "publicationError": null,
    "publicationStatus": "PUBLICATION_COMPLETED",
    "sourceRegistryId": "<ocid:2>",
    "sourceRegistryUrl": "iad.ocir.io/example/image:1.0.0",
    "validationError": null,
    "validationStatus": "VALIDATION_COMPLETED"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "artifact-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "publisherId": "<ocid:5>",
  "status": "AVAILABLE",
  "statusNotes": null,
  "systemTags": null,
  "timeCreated": "2026-04-29T12:00:00Z",
  "timeUpdated": "2026-04-29T12:00:00Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.ContainerImageArtifact](t, `
{
  "artifactType": "CONTAINER_IMAGE",
  "compartmentId": "<ocid:1>",
  "containerImage": {
    "publicationError": null,
    "publicationStatus": "PUBLICATION_COMPLETED",
    "sourceRegistryId": "<ocid:2>",
    "sourceRegistryUrl": "iad.ocir.io/example/image:1.0.0",
    "validationError": null,
    "validationStatus": "VALIDATION_COMPLETED"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "artifact-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:4>",
  "lifecycleState": "DELETED",
  "publisherId": "<ocid:5>",
  "status": "AVAILABLE",
  "statusNotes": null,
  "systemTags": null,
  "timeCreated": "2026-04-29T12:00:00Z",
  "timeUpdated": "2026-04-29T12:00:00Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[marketplacepublishersdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_ARTIFACT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "Artifact",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[marketplacepublishersdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_ARTIFACT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "Artifact",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[marketplacepublishersdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_ARTIFACT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "Artifact",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[marketplacepublishersdk.ContainerImageArtifact, marketplacepublishersdk.CreateContainerImageArtifactDetails, marketplacepublishersdk.UpdateContainerImageArtifactDetails]{
		CollectionPath: "/20241201/artifacts", ItemPath: "/20241201/artifacts/<ocid:4>",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "artifactType", "CONTAINER_IMAGE", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "artifactType", "CONTAINER_IMAGE", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20241201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20241201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20241201/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://marketplacepublisher.mock.invalid", BasePath: "20241201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	client := newArtifactServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.Artifact]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.Artifact) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.ArtifactType != resource.Spec.ArtifactType ||
				current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Artifact status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.Artifact) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "artifact-updated"
}`)
		},
		ValidateUpdated: func(current *marketplacepublisherv1beta1.Artifact) error {
			if !(current.Status.DisplayName == "artifact-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Artifact status = %+v", current.Status)
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
