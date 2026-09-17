/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package librarymaskingformat

import (
	"context"
	"fmt"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLibraryMaskingFormatEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.LibraryMaskingFormat{}
	ocimock.InitializeResource(resource, "mock-librarymaskingformat")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.LibraryMaskingFormatSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-library-masking-format-v2",
  "formatEntries": [
    {
      "description": "fixed mock value",
      "fixedString": "MASKED",
      "type": "FIXED_STRING"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  },
  "sensitiveTypeIds": []
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateLibraryMaskingFormatDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-library-masking-format-v2",
  "formatEntries": [
    {
      "description": "fixed mock value",
      "fixedString": "MASKED",
      "type": "FIXED_STRING"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  },
  "sensitiveTypeIds": []
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.LibraryMaskingFormat](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:28:39.013Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-library-masking-format-v2",
  "formatEntries": [
    {
      "description": "fixed mock value",
      "fixedString": "MASKED",
      "type": "FIXED_STRING"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sensitiveTypeIds": [],
  "source": "USER",
  "timeCreated": "2026-09-03T17:28:39.078Z",
  "timeUpdated": "2026-09-03T17:28:42.792Z"
}`)
	createdReadStates := []datasafesdk.LibraryMaskingFormat{
		ocimock.MustOCIResponseFixture[datasafesdk.LibraryMaskingFormat](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:28:39.013Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-library-masking-format-v2",
  "formatEntries": [
    {
      "description": "fixed mock value",
      "fixedString": "MASKED",
      "type": "FIXED_STRING"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sensitiveTypeIds": [],
  "source": "USER",
  "timeCreated": "2026-09-03T17:28:39.078Z",
  "timeUpdated": "2026-09-03T17:28:42.792Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateLibraryMaskingFormatDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.LibraryMaskingFormat](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:28:39.013Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-library-masking-format-v2",
  "formatEntries": [
    {
      "description": "fixed mock value",
      "fixedString": "MASKED",
      "type": "FIXED_STRING"
    }
  ],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sensitiveTypeIds": [],
  "source": "USER",
  "timeCreated": "2026-09-03T17:28:39.078Z",
  "timeUpdated": "2026-09-03T17:28:51.286Z"
}`)
	updatedReadStates := []datasafesdk.LibraryMaskingFormat{
		ocimock.MustOCIResponseFixture[datasafesdk.LibraryMaskingFormat](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:28:39.013Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-library-masking-format-v2",
  "formatEntries": [
    {
      "description": "fixed mock value",
      "fixedString": "MASKED",
      "type": "FIXED_STRING"
    }
  ],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sensitiveTypeIds": [],
  "source": "USER",
  "timeCreated": "2026-09-03T17:28:39.078Z",
  "timeUpdated": "2026-09-03T17:28:51.286Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.LibraryMaskingFormat,
		datasafesdk.CreateLibraryMaskingFormatDetails,
		datasafesdk.UpdateLibraryMaskingFormatDetails,
	]{
		CollectionPath:     "/20181201/libraryMaskingFormats",
		ItemPath:           "/20181201/libraryMaskingFormats/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       202,
		DeleteStatus:       204,
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateLibraryMaskingFormatDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.LibraryMaskingFormat) error {
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
			t.Errorf("close LibraryMaskingFormat OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &LibraryMaskingFormatServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newLibraryMaskingFormatDefaultRuntimeHooks(sdkClient)
	applyLibraryMaskingFormatRuntimeHooks(&hooks)
	client := wrapLibraryMaskingFormatGeneratedClient(hooks, defaultLibraryMaskingFormatServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.LibraryMaskingFormat](buildLibraryMaskingFormatGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.LibraryMaskingFormat]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.LibraryMaskingFormat) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FormatEntries, current.Spec.FormatEntries) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.SensitiveTypeIds, current.Spec.SensitiveTypeIds) {
				return fmt.Errorf("created LibraryMaskingFormat status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.LibraryMaskingFormat) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.LibraryMaskingFormat) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				len(current.Status.FormatEntries) != 1 ||
				current.Status.FormatEntries[0].FixedString != "MASKED" ||
				len(current.Status.SensitiveTypeIds) != 0 {
				return fmt.Errorf("updated LibraryMaskingFormat status = %+v", current.Status)
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
