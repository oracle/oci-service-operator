/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package auditprofile

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
func TestMockIntegrationAuditProfileLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeAuditProfileResource()
	ocimock.InitializeResource(resource, "mock-auditprofile")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.AuditProfileSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description",
  "displayName": "audit-profile",
  "freeformTags": {
    "env": "dev"
  },
  "isOverrideGlobalPaidUsage": false,
  "isPaidUsageEnabled": false,
  "offlineMonths": 12,
  "onlineMonths": 6,
  "targetId": "\u003cocid:2\u003e",
  "targetType": "TARGET_DATABASE"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "desired description-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateAuditProfileDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description",
  "displayName": "audit-profile",
  "freeformTags": {
    "env": "dev"
  },
  "isOverrideGlobalPaidUsage": false,
  "isPaidUsageEnabled": false,
  "offlineMonths": 12,
  "onlineMonths": 6,
  "targetId": "\u003cocid:2\u003e",
  "targetType": "TARGET_DATABASE"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.AuditProfile](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description",
  "displayName": "audit-profile",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "isOverrideGlobalPaidUsage": false,
  "isPaidUsageEnabled": false,
  "lifecycleState": "ACTIVE",
  "offlineMonths": 12,
  "onlineMonths": 6,
  "targetId": "\u003cocid:2\u003e",
  "targetType": "TARGET_DATABASE"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateAuditProfileDetails](t, `{
  "description": "desired description-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.AuditProfile](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description-updated",
  "displayName": "audit-profile",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "isOverrideGlobalPaidUsage": false,
  "isPaidUsageEnabled": false,
  "lifecycleState": "ACTIVE",
  "offlineMonths": 12,
  "onlineMonths": 6,
  "targetId": "\u003cocid:2\u003e",
  "targetType": "TARGET_DATABASE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.AuditProfile,
		datasafesdk.CreateAuditProfileDetails,
		datasafesdk.UpdateAuditProfileDetails,
	]{
		CollectionPath:    "/20181201/auditProfiles",
		ItemPath:          "/20181201/auditProfiles/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateAuditProfileDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.AuditProfile) error {
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
			t.Errorf("close AuditProfile OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	client := newAuditProfileServiceClientWithOCIClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.AuditProfile]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.AuditProfile) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsOverrideGlobalPaidUsage, current.Spec.IsOverrideGlobalPaidUsage) ||
				!reflect.DeepEqual(current.Status.IsPaidUsageEnabled, current.Spec.IsPaidUsageEnabled) ||
				!reflect.DeepEqual(current.Status.OfflineMonths, current.Spec.OfflineMonths) ||
				!reflect.DeepEqual(current.Status.OnlineMonths, current.Spec.OnlineMonths) ||
				!reflect.DeepEqual(current.Status.TargetId, current.Spec.TargetId) ||
				!reflect.DeepEqual(current.Status.TargetType, current.Spec.TargetType) {
				return fmt.Errorf("created AuditProfile status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.AuditProfile) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.AuditProfile) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated AuditProfile status = %+v", current.Status)
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
