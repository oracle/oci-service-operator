/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dataset

import (
	"context"
	"fmt"
	datalabelingservicesdk "github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	datalabelingservicev1beta1 "github.com/oracle/oci-service-operator/api/datalabelingservice/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDatasetLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newDatasetTestResource()
	ocimock.InitializeResource(resource, "mock-dataset")
	resource.Spec = ocimock.MustJSONFixture[datalabelingservicev1beta1.DatasetSpec](t, `{
  "annotationFormat": "SINGLE_LABEL",
  "compartmentId": "\u003cocid:1\u003e",
  "datasetFormatDetails": {
    "formatType": "TEXT",
    "textFileTypeMetadata": {
      "columnDelimiter": ",",
      "columnIndex": 2,
      "columnName": "text",
      "escapeCharacter": "\\",
      "formatType": "DELIMITED",
      "lineDelimiter": "\n"
    }
  },
  "datasetSourceDetails": {
    "bucket": "dataset-bucket",
    "namespace": "datasetns",
    "prefix": "records/",
    "sourceType": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "dataset description",
  "displayName": "dataset-alpha",
  "freeformTags": {
    "env": "test"
  },
  "initialImportDatasetConfiguration": {
    "importFormat": {
      "name": "JSONL_CONSOLIDATED"
    },
    "importMetadataPath": {
      "bucket": "dataset-bucket",
      "namespace": "datasetns",
      "path": "imports/preannotated.jsonl",
      "sourceType": "OBJECT_STORAGE"
    }
  },
  "initialRecordGenerationConfiguration": {
    "limit": 25
  },
  "labelSet": {
    "items": [
      {
        "name": "cat"
      }
    ]
  },
  "labelingInstructions": "label carefully"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "dataset description-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datalabelingservicesdk.CreateDatasetDetails](t, `{
  "annotationFormat": "SINGLE_LABEL",
  "compartmentId": "\u003cocid:1\u003e",
  "datasetFormatDetails": {
    "formatType": "TEXT",
    "textFileTypeMetadata": {
      "columnDelimiter": ",",
      "columnIndex": 2,
      "columnName": "text",
      "escapeCharacter": "\\",
      "formatType": "DELIMITED",
      "lineDelimiter": "\n"
    }
  },
  "datasetSourceDetails": {
    "bucket": "dataset-bucket",
    "namespace": "datasetns",
    "prefix": "records/",
    "sourceType": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "dataset description",
  "displayName": "dataset-alpha",
  "freeformTags": {
    "env": "test"
  },
  "initialImportDatasetConfiguration": {
    "importFormat": {
      "name": "JSONL_CONSOLIDATED"
    },
    "importMetadataPath": {
      "bucket": "dataset-bucket",
      "namespace": "datasetns",
      "path": "imports/preannotated.jsonl",
      "sourceType": "OBJECT_STORAGE"
    }
  },
  "initialRecordGenerationConfiguration": {
    "limit": 25
  },
  "labelSet": {
    "items": [
      {
        "name": "cat"
      }
    ]
  },
  "labelingInstructions": "label carefully"
}`)
	createdState := ocimock.MustOCIResponseFixture[datalabelingservicesdk.Dataset](t, `{
  "annotationFormat": "SINGLE_LABEL",
  "compartmentId": "\u003cocid:1\u003e",
  "datasetFormatDetails": {
    "formatType": "TEXT",
    "textFileTypeMetadata": {
      "columnDelimiter": ",",
      "columnIndex": 2,
      "columnName": "text",
      "escapeCharacter": "\\",
      "formatType": "DELIMITED",
      "lineDelimiter": "\n"
    }
  },
  "datasetSourceDetails": {
    "bucket": "dataset-bucket",
    "namespace": "datasetns",
    "prefix": "records/",
    "sourceType": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "dataset description",
  "displayName": "dataset-alpha",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "initialImportDatasetConfiguration": {
    "importFormat": {
      "name": "JSONL_CONSOLIDATED"
    },
    "importMetadataPath": {
      "bucket": "dataset-bucket",
      "namespace": "datasetns",
      "path": "imports/preannotated.jsonl",
      "sourceType": "OBJECT_STORAGE"
    }
  },
  "initialRecordGenerationConfiguration": {
    "limit": 25
  },
  "labelSet": {
    "items": [
      {
        "name": "cat"
      }
    ]
  },
  "labelingInstructions": "label carefully",
  "lifecycleState": "ACTIVE"
}`)
	createdReadStates := []datalabelingservicesdk.Dataset{
		ocimock.MustOCIResponseFixture[datalabelingservicesdk.Dataset](t, `{
  "annotationFormat": "SINGLE_LABEL",
  "compartmentId": "\u003cocid:1\u003e",
  "datasetFormatDetails": {
    "formatType": "TEXT",
    "textFileTypeMetadata": {
      "columnDelimiter": ",",
      "columnIndex": 2,
      "columnName": "text",
      "escapeCharacter": "\\",
      "formatType": "DELIMITED",
      "lineDelimiter": "\n"
    }
  },
  "datasetSourceDetails": {
    "bucket": "dataset-bucket",
    "namespace": "datasetns",
    "prefix": "records/",
    "sourceType": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "dataset description",
  "displayName": "dataset-alpha",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "initialImportDatasetConfiguration": {
    "importFormat": {
      "name": "JSONL_CONSOLIDATED"
    },
    "importMetadataPath": {
      "bucket": "dataset-bucket",
      "namespace": "datasetns",
      "path": "imports/preannotated.jsonl",
      "sourceType": "OBJECT_STORAGE"
    }
  },
  "initialRecordGenerationConfiguration": {
    "limit": 25
  },
  "labelSet": {
    "items": [
      {
        "name": "cat"
      }
    ]
  },
  "labelingInstructions": "label carefully",
  "lifecycleState": "ACTIVE"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[datalabelingservicesdk.UpdateDatasetDetails](t, `{
  "description": "dataset description-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datalabelingservicesdk.Dataset](t, `{
  "annotationFormat": "SINGLE_LABEL",
  "compartmentId": "\u003cocid:1\u003e",
  "datasetFormatDetails": {
    "formatType": "TEXT",
    "textFileTypeMetadata": {
      "columnDelimiter": ",",
      "columnIndex": 2,
      "columnName": "text",
      "escapeCharacter": "\\",
      "formatType": "DELIMITED",
      "lineDelimiter": "\n"
    }
  },
  "datasetSourceDetails": {
    "bucket": "dataset-bucket",
    "namespace": "datasetns",
    "prefix": "records/",
    "sourceType": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "dataset description-updated",
  "displayName": "dataset-alpha",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "initialImportDatasetConfiguration": {
    "importFormat": {
      "name": "JSONL_CONSOLIDATED"
    },
    "importMetadataPath": {
      "bucket": "dataset-bucket",
      "namespace": "datasetns",
      "path": "imports/preannotated.jsonl",
      "sourceType": "OBJECT_STORAGE"
    }
  },
  "initialRecordGenerationConfiguration": {
    "limit": 25
  },
  "labelSet": {
    "items": [
      {
        "name": "cat"
      }
    ]
  },
  "labelingInstructions": "label carefully",
  "lifecycleState": "ACTIVE"
}`)
	updatedReadStates := []datalabelingservicesdk.Dataset{
		ocimock.MustOCIResponseFixture[datalabelingservicesdk.Dataset](t, `{
  "annotationFormat": "SINGLE_LABEL",
  "compartmentId": "\u003cocid:1\u003e",
  "datasetFormatDetails": {
    "formatType": "TEXT",
    "textFileTypeMetadata": {
      "columnDelimiter": ",",
      "columnIndex": 2,
      "columnName": "text",
      "escapeCharacter": "\\",
      "formatType": "DELIMITED",
      "lineDelimiter": "\n"
    }
  },
  "datasetSourceDetails": {
    "bucket": "dataset-bucket",
    "namespace": "datasetns",
    "prefix": "records/",
    "sourceType": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "dataset description-updated",
  "displayName": "dataset-alpha",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "initialImportDatasetConfiguration": {
    "importFormat": {
      "name": "JSONL_CONSOLIDATED"
    },
    "importMetadataPath": {
      "bucket": "dataset-bucket",
      "namespace": "datasetns",
      "path": "imports/preannotated.jsonl",
      "sourceType": "OBJECT_STORAGE"
    }
  },
  "initialRecordGenerationConfiguration": {
    "limit": 25
  },
  "labelSet": {
    "items": [
      {
        "name": "cat"
      }
    ]
  },
  "labelingInstructions": "label carefully",
  "lifecycleState": "ACTIVE"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datalabelingservicesdk.Dataset,
		datalabelingservicesdk.CreateDatasetDetails,
		datalabelingservicesdk.UpdateDatasetDetails,
	]{
		CollectionPath:     "/20211001/datasets",
		ItemPath:           "/20211001/datasets/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		CreatedReadStates:  append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...),
		UpdatedReadStates:  append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datalabelingservicesdk.CreateDatasetDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datalabelingservicesdk.Dataset) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20211001", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Dataset OCI mock: %v", err)
		}
	})
	sdkClient := datalabelingservicesdk.DataLabelingManagementClient{BaseClient: session.BaseClient()}
	client := newDatasetServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datalabelingservicev1beta1.Dataset]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datalabelingservicev1beta1.Dataset) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				current.Status.AnnotationFormat != current.Spec.AnnotationFormat ||
				current.Status.CompartmentId != current.Spec.CompartmentId ||
				current.Status.Description != current.Spec.Description ||
				current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["env"] != "test" ||
				current.Status.LabelingInstructions != current.Spec.LabelingInstructions ||
				current.Status.DatasetSourceDetails.SourceType != "OBJECT_STORAGE" ||
				current.Status.DatasetSourceDetails.Bucket != "dataset-bucket" ||
				current.Status.DatasetFormatDetails.FormatType != "TEXT" ||
				len(current.Status.LabelSet.Items) != 1 ||
				current.Status.LabelSet.Items[0].Name != "cat" {
				return fmt.Errorf("created Dataset status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datalabelingservicev1beta1.Dataset) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datalabelingservicev1beta1.Dataset) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Dataset status = %+v", current.Status)
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
