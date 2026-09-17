/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package auditarchiveretrieval

import (
	"context"
	"fmt"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationAuditArchiveRetrievalLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	ocimock.InitializeResource(resource, "mock-auditarchiveretrieval")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.AuditArchiveRetrievalSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "retrieve archived audit events",
  "displayName": "audit-retrieval",
  "endDate": "2026-01-31T00:00:00Z",
  "freeformTags": {
    "team": "security"
  },
  "startDate": "2026-01-01T00:00:00Z",
  "targetId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "retrieve archived audit events-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateAuditArchiveRetrievalDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "retrieve archived audit events",
  "displayName": "audit-retrieval",
  "endDate": "2026-01-31T00:00:00Z",
  "freeformTags": {
    "team": "security"
  },
  "startDate": "2026-01-01T00:00:00Z",
  "targetId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.AuditArchiveRetrieval](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "retrieve archived audit events",
  "displayName": "audit-retrieval",
  "endDate": "2026-01-31T00:00:00Z",
  "freeformTags": {
    "team": "security"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "startDate": "2026-01-01T00:00:00Z",
  "targetId": "\u003cocid:2\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateAuditArchiveRetrievalDetails](t, `{
  "description": "retrieve archived audit events-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.AuditArchiveRetrieval](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "retrieve archived audit events-updated",
  "displayName": "audit-retrieval",
  "endDate": "2026-01-31T00:00:00Z",
  "freeformTags": {
    "team": "security"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "startDate": "2026-01-01T00:00:00Z",
  "targetId": "\u003cocid:2\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.AuditArchiveRetrieval,
		datasafesdk.CreateAuditArchiveRetrievalDetails,
		datasafesdk.UpdateAuditArchiveRetrievalDetails,
	]{
		CollectionPath:    "/20181201/auditArchiveRetrievals",
		ItemPath:          "/20181201/auditArchiveRetrievals/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateAuditArchiveRetrievalDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.AuditArchiveRetrieval) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close AuditArchiveRetrieval OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	hooks := newAuditArchiveRetrievalDefaultRuntimeHooks(sdkClient)
	applyAuditArchiveRetrievalRuntimeHooks(&hooks)
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.AuditArchiveRetrieval]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.AuditArchiveRetrieval) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.EndDate, current.Spec.EndDate) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.StartDate, current.Spec.StartDate) ||
				!reflect.DeepEqual(current.Status.TargetId, current.Spec.TargetId) {
				return fmt.Errorf("created AuditArchiveRetrieval status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.AuditArchiveRetrieval) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.AuditArchiveRetrieval) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated AuditArchiveRetrieval status = %+v", current.Status)
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
