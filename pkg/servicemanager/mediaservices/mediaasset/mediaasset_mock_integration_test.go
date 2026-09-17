/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package mediaasset

import (
	"context"
	"fmt"
	mediaservicessdk "github.com/oracle/oci-go-sdk/v65/mediaservices"
	mediaservicesv1beta1 "github.com/oracle/oci-service-operator/api/mediaservices/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationMediaAssetLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newMediaAssetTestResource()
	ocimock.InitializeResource(resource, "mock-mediaasset")
	resource.Spec = ocimock.MustJSONFixture[mediaservicesv1beta1.MediaAssetSpec](t, `{
  "bucketName": "media-bucket",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "asset-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "masterMediaAssetId": "\u003cocid:2\u003e",
  "mediaAssetTags": [
    {
      "type": "USER",
      "value": "ingest"
    }
  ],
  "mediaWorkflowJobId": "\u003cocid:3\u003e",
  "metadata": [
    {
      "metadata": "{\"codec\":\"h264\"}"
    }
  ],
  "namespaceName": "object-namespace",
  "objectEtag": "etag-value",
  "objectName": "video.mp4",
  "parentMediaAssetId": "\u003cocid:4\u003e",
  "segmentRangeEndIndex": 10,
  "segmentRangeStartIndex": 1,
  "sourceMediaWorkflowId": "\u003cocid:5\u003e",
  "sourceMediaWorkflowVersion": 7,
  "type": "VIDEO"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "asset-alpha-updated"
}`)
	createRequest := ocimock.MustJSONFixture[mediaservicessdk.CreateMediaAssetDetails](t, `{
  "bucketName": "media-bucket",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "asset-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "masterMediaAssetId": "\u003cocid:2\u003e",
  "mediaAssetTags": [
    {
      "type": "USER",
      "value": "ingest"
    }
  ],
  "mediaWorkflowJobId": "\u003cocid:3\u003e",
  "metadata": [
    {
      "metadata": "{\"codec\":\"h264\"}"
    }
  ],
  "namespaceName": "object-namespace",
  "objectEtag": "etag-value",
  "objectName": "video.mp4",
  "parentMediaAssetId": "\u003cocid:4\u003e",
  "segmentRangeEndIndex": 10,
  "segmentRangeStartIndex": 1,
  "sourceMediaWorkflowId": "\u003cocid:5\u003e",
  "sourceMediaWorkflowVersion": 7,
  "type": "VIDEO"
}`)
	createdState := ocimock.MustOCIResponseFixture[mediaservicessdk.MediaAsset](t, `{
  "bucketName": "media-bucket",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "asset-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:6\u003e",
  "lifecycleState": "ACTIVE",
  "masterMediaAssetId": "\u003cocid:2\u003e",
  "mediaAssetTags": [
    {
      "type": "USER",
      "value": "ingest"
    }
  ],
  "mediaWorkflowJobId": "\u003cocid:3\u003e",
  "metadata": [
    {
      "metadata": "{\"codec\":\"h264\"}"
    }
  ],
  "namespaceName": "object-namespace",
  "objectEtag": "etag-value",
  "objectName": "video.mp4",
  "parentMediaAssetId": "\u003cocid:4\u003e",
  "segmentRangeEndIndex": 10,
  "segmentRangeStartIndex": 1,
  "sourceMediaWorkflowId": "\u003cocid:5\u003e",
  "sourceMediaWorkflowVersion": 7,
  "type": "VIDEO"
}`)
	updateRequest := ocimock.MustJSONFixture[mediaservicessdk.UpdateMediaAssetDetails](t, `{
  "displayName": "asset-alpha-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[mediaservicessdk.MediaAsset](t, `{
  "bucketName": "media-bucket",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "asset-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:6\u003e",
  "lifecycleState": "ACTIVE",
  "masterMediaAssetId": "\u003cocid:2\u003e",
  "mediaAssetTags": [
    {
      "type": "USER",
      "value": "ingest"
    }
  ],
  "mediaWorkflowJobId": "\u003cocid:3\u003e",
  "metadata": [
    {
      "metadata": "{\"codec\":\"h264\"}"
    }
  ],
  "namespaceName": "object-namespace",
  "objectEtag": "etag-value",
  "objectName": "video.mp4",
  "parentMediaAssetId": "\u003cocid:4\u003e",
  "segmentRangeEndIndex": 10,
  "segmentRangeStartIndex": 1,
  "sourceMediaWorkflowId": "\u003cocid:5\u003e",
  "sourceMediaWorkflowVersion": 7,
  "type": "VIDEO"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		mediaservicessdk.MediaAsset,
		mediaservicessdk.CreateMediaAssetDetails,
		mediaservicessdk.UpdateMediaAssetDetails,
	]{
		CollectionPath:     "/20211101/mediaAssets",
		ItemPath:           "/20211101/mediaAssets/<ocid:6>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		UpdatedReadStates:  ocimock.LifecycleStateSequence(t, updatedState, "UPDATING"),
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ mediaservicessdk.CreateMediaAssetDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ mediaservicessdk.MediaAsset) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20211101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MediaAsset OCI mock: %v", err)
		}
	})
	sdkClient := mediaservicessdk.MediaServicesClient{BaseClient: session.BaseClient()}
	client := newMediaAssetServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*mediaservicesv1beta1.MediaAsset]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *mediaservicesv1beta1.MediaAsset) error {
			if current.Status.Id != "<ocid:6>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:6>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.BucketName, current.Spec.BucketName) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.MasterMediaAssetId, current.Spec.MasterMediaAssetId) ||
				!reflect.DeepEqual(current.Status.MediaAssetTags, current.Spec.MediaAssetTags) ||
				!reflect.DeepEqual(current.Status.MediaWorkflowJobId, current.Spec.MediaWorkflowJobId) ||
				!reflect.DeepEqual(current.Status.Metadata, current.Spec.Metadata) ||
				!reflect.DeepEqual(current.Status.NamespaceName, current.Spec.NamespaceName) ||
				!reflect.DeepEqual(current.Status.ObjectEtag, current.Spec.ObjectEtag) ||
				!reflect.DeepEqual(current.Status.ObjectName, current.Spec.ObjectName) ||
				!reflect.DeepEqual(current.Status.ParentMediaAssetId, current.Spec.ParentMediaAssetId) ||
				!reflect.DeepEqual(current.Status.SegmentRangeEndIndex, current.Spec.SegmentRangeEndIndex) ||
				!reflect.DeepEqual(current.Status.SegmentRangeStartIndex, current.Spec.SegmentRangeStartIndex) ||
				!reflect.DeepEqual(current.Status.SourceMediaWorkflowId, current.Spec.SourceMediaWorkflowId) ||
				!reflect.DeepEqual(current.Status.SourceMediaWorkflowVersion, current.Spec.SourceMediaWorkflowVersion) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created MediaAsset status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *mediaservicesv1beta1.MediaAsset) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *mediaservicesv1beta1.MediaAsset) error {
			if current.Status.Id != "<ocid:6>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:6>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated MediaAsset status = %+v", current.Status)
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
