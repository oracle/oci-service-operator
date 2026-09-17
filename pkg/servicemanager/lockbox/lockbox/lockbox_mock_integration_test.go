/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package lockbox

import (
	"context"
	"fmt"
	lockboxsdk "github.com/oracle/oci-go-sdk/v65/lockbox"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLockboxLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := baseLockboxResource()
	ocimock.InitializeResource(resource, "mock-lockbox")
	resource.Spec = ocimock.MustJSONFixture[lockboxv1beta1.LockboxSpec](t, `{
  "accessContextAttributes": {
    "items": [
      {
        "defaultValue": "sr-1",
        "description": "Support ticket",
        "name": "ticket",
        "values": [
          "sr-1",
          "sr-2"
        ]
      }
    ]
  },
  "approvalTemplateId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "displayName": "test-lockbox",
  "freeformTags": {
    "env": "test"
  },
  "lockboxPartner": "FAAAS",
  "maxAccessDuration": "PT2H",
  "partnerCompartmentId": "\u003cocid:3\u003e",
  "partnerId": "\u003cocid:4\u003e",
  "resourceId": "\u003cocid:5\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "test-lockbox-updated"
}`)
	createRequest := ocimock.MustJSONFixture[lockboxsdk.CreateLockboxDetails](t, `{
  "accessContextAttributes": {
    "items": [
      {
        "defaultValue": "sr-1",
        "description": "Support ticket",
        "name": "ticket",
        "values": [
          "sr-1",
          "sr-2"
        ]
      }
    ]
  },
  "approvalTemplateId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "displayName": "test-lockbox",
  "freeformTags": {
    "env": "test"
  },
  "lockboxPartner": "FAAAS",
  "maxAccessDuration": "PT2H",
  "partnerCompartmentId": "\u003cocid:3\u003e",
  "partnerId": "\u003cocid:4\u003e",
  "resourceId": "\u003cocid:5\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[lockboxsdk.Lockbox](t, `{
  "accessContextAttributes": {
    "items": [
      {
        "defaultValue": "sr-1",
        "description": "Support ticket",
        "name": "ticket",
        "values": [
          "sr-1",
          "sr-2"
        ]
      }
    ]
  },
  "approvalTemplateId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "displayName": "test-lockbox",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:6\u003e",
  "lifecycleState": "ACTIVE",
  "lockboxPartner": "FAAAS",
  "maxAccessDuration": "PT2H",
  "partnerCompartmentId": "\u003cocid:3\u003e",
  "partnerId": "\u003cocid:4\u003e",
  "resourceId": "\u003cocid:5\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[lockboxsdk.UpdateLockboxDetails](t, `{
  "displayName": "test-lockbox-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[lockboxsdk.Lockbox](t, `{
  "accessContextAttributes": {
    "items": [
      {
        "defaultValue": "sr-1",
        "description": "Support ticket",
        "name": "ticket",
        "values": [
          "sr-1",
          "sr-2"
        ]
      }
    ]
  },
  "approvalTemplateId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "displayName": "test-lockbox-updated",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:6\u003e",
  "lifecycleState": "ACTIVE",
  "lockboxPartner": "FAAAS",
  "maxAccessDuration": "PT2H",
  "partnerCompartmentId": "\u003cocid:3\u003e",
  "partnerId": "\u003cocid:4\u003e",
  "resourceId": "\u003cocid:5\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		lockboxsdk.Lockbox,
		lockboxsdk.CreateLockboxDetails,
		lockboxsdk.UpdateLockboxDetails,
	]{
		CollectionPath:    "/20220126/lockboxes",
		ItemPath:          "/20220126/lockboxes/<ocid:6>",
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
		ValidateCreate: func(request ocimock.Request, _ lockboxsdk.CreateLockboxDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ lockboxsdk.Lockbox) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220126", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Lockbox OCI mock: %v", err)
		}
	})
	sdkClient := lockboxsdk.LockboxClient{BaseClient: session.BaseClient()}
	client := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*lockboxv1beta1.Lockbox]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *lockboxv1beta1.Lockbox) error {
			if current.Status.Id != "<ocid:6>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:6>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AccessContextAttributes, current.Spec.AccessContextAttributes) ||
				!reflect.DeepEqual(current.Status.ApprovalTemplateId, current.Spec.ApprovalTemplateId) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LockboxPartner, current.Spec.LockboxPartner) ||
				!reflect.DeepEqual(current.Status.MaxAccessDuration, current.Spec.MaxAccessDuration) ||
				!reflect.DeepEqual(current.Status.PartnerCompartmentId, current.Spec.PartnerCompartmentId) ||
				!reflect.DeepEqual(current.Status.PartnerId, current.Spec.PartnerId) ||
				!reflect.DeepEqual(current.Status.ResourceId, current.Spec.ResourceId) {
				return fmt.Errorf("created Lockbox status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *lockboxv1beta1.Lockbox) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *lockboxv1beta1.Lockbox) error {
			if current.Status.Id != "<ocid:6>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:6>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated Lockbox status = %+v", current.Status)
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
