/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package managementagentinstallkey

import (
	"context"
	"fmt"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationManagementAgentInstallKeyEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &managementagentv1beta1.ManagementAgentInstallKey{}
	ocimock.InitializeResource(resource, "mock-managementagentinstallkey")
	resource.Spec = ocimock.MustJSONFixture[managementagentv1beta1.ManagementAgentInstallKeySpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-install-key-v1",
  "isUnlimited": true
}`)
	resource.Spec.IsKeyActive = true
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-install-key-v1-updated"
}`)
	createRequest := ocimock.MustJSONFixture[managementagentsdk.CreateManagementAgentInstallKeyDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-install-key-v1",
  "isUnlimited": true
}`)
	createdState := ocimock.MustOCIResponseFixture[managementagentsdk.ManagementAgentInstallKey](t, `{
  "allowedKeyInstallCount": null,
  "compartmentId": "<ocid:1>",
  "createdByPrincipalId": "<redacted>",
  "currentKeyInstallCount": 0,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T18:17:44.856Z"
    }
  },
  "displayName": "osok-mock-install-key-v1",
  "freeformTags": {},
  "id": "<ocid:2>",
  "isUnlimited": true,
  "key": "Mi4wLHVzLWFzaGJ1cm4tMSxvY2lkMS50ZW5hbmN5Lm9jMS4uYWFhYWFhYWFhdjU3ZmpjNGFibWRlcXZudXd5aDV0bWk0b240Z3VleHNxZDV6b3h2MmU0Nnh1M2NuYmRhLG9jaWQxLm1hbmFnZW1lbnRhZ2VudGluc3RhbGxrZXkub2MxLmlhZC5hbWFhYWFhYXNyM282a2FhM2tjcWtncXFkaXdxa3B4NWFqNXQ2bGJweWVlZzdxYWZiN2lpb3VwNHRubmEsSWZ4ZzVCOEFKSGJiODJNck56M3ZJRnc2ZGItaXdtSFE0T2J5VmJqVg==",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-01T18:17:44.934Z",
  "timeExpires": null,
  "timeUpdated": "2026-09-01T18:17:44.934Z"
}`)
	updateRequest := ocimock.MustJSONFixture[managementagentsdk.UpdateManagementAgentInstallKeyDetails](t, `{
  "displayName": "osok-mock-install-key-v1-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[managementagentsdk.ManagementAgentInstallKey](t, `{
  "allowedKeyInstallCount": null,
  "compartmentId": "<ocid:1>",
  "createdByPrincipalId": "<redacted>",
  "currentKeyInstallCount": 0,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T18:17:44.856Z"
    }
  },
  "displayName": "osok-mock-install-key-v1-updated",
  "freeformTags": {},
  "id": "<ocid:2>",
  "isUnlimited": true,
  "key": "Mi4wLHVzLWFzaGJ1cm4tMSxvY2lkMS50ZW5hbmN5Lm9jMS4uYWFhYWFhYWFhdjU3ZmpjNGFibWRlcXZudXd5aDV0bWk0b240Z3VleHNxZDV6b3h2MmU0Nnh1M2NuYmRhLG9jaWQxLm1hbmFnZW1lbnRhZ2VudGluc3RhbGxrZXkub2MxLmlhZC5hbWFhYWFhYXNyM282a2FhM2tjcWtncXFkaXdxa3B4NWFqNXQ2bGJweWVlZzdxYWZiN2lpb3VwNHRubmEsSWZ4ZzVCOEFKSGJiODJNck56M3ZJRnc2ZGItaXdtSFE0T2J5VmJqVg==",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-01T18:17:44.934Z",
  "timeExpires": null,
  "timeUpdated": "2026-09-01T18:17:45.766Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[managementagentsdk.ManagementAgentInstallKey](t, `{
  "allowedKeyInstallCount": null,
  "compartmentId": "<ocid:1>",
  "createdByPrincipalId": "<redacted>",
  "currentKeyInstallCount": 0,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T18:17:44.856Z"
    }
  },
  "displayName": "osok-mock-install-key-v1-updated",
  "freeformTags": {},
  "id": "<ocid:2>",
  "isUnlimited": true,
  "key": "Mi4wLHVzLWFzaGJ1cm4tMSxvY2lkMS50ZW5hbmN5Lm9jMS4uYWFhYWFhYWFhdjU3ZmpjNGFibWRlcXZudXd5aDV0bWk0b240Z3VleHNxZDV6b3h2MmU0Nnh1M2NuYmRhLG9jaWQxLm1hbmFnZW1lbnRhZ2VudGluc3RhbGxrZXkub2MxLmlhZC5hbWFhYWFhYXNyM282a2FhM2tjcWtncXFkaXdxa3B4NWFqNXQ2bGJweWVlZzdxYWZiN2lpb3VwNHRubmEsSWZ4ZzVCOEFKSGJiODJNck56M3ZJRnc2ZGItaXdtSFE0T2J5VmJqVg==",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-01T18:17:44.934Z",
  "timeExpires": null,
  "timeUpdated": "2026-09-01T18:17:46.723Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		managementagentsdk.ManagementAgentInstallKey,
		managementagentsdk.CreateManagementAgentInstallKeyDetails,
		managementagentsdk.UpdateManagementAgentInstallKeyDetails,
	]{
		CollectionPath:    "/20200202/managementAgentInstallKeys",
		ItemPath:          "/20200202/managementAgentInstallKeys/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ managementagentsdk.CreateManagementAgentInstallKeyDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ managementagentsdk.ManagementAgentInstallKey) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200202", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ManagementAgentInstallKey OCI mock: %v", err)
		}
	})
	sdkClient := managementagentsdk.ManagementAgentClient{BaseClient: session.BaseClient()}
	client := newManagementAgentInstallKeyServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managementagentv1beta1.ManagementAgentInstallKey]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managementagentv1beta1.ManagementAgentInstallKey) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.IsUnlimited, current.Spec.IsUnlimited) {
				return fmt.Errorf("created ManagementAgentInstallKey status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managementagentv1beta1.ManagementAgentInstallKey) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *managementagentv1beta1.ManagementAgentInstallKey) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated ManagementAgentInstallKey status = %+v", current.Status)
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
