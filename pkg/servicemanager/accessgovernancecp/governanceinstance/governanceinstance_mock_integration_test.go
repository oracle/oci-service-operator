/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package governanceinstance

import (
	"context"
	"fmt"
	accessgovernancecpsdk "github.com/oracle/oci-go-sdk/v65/accessgovernancecp"
	accessgovernancecpv1beta1 "github.com/oracle/oci-service-operator/api/accessgovernancecp/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationGovernanceInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &accessgovernancecpv1beta1.GovernanceInstance{
		Spec: accessgovernancecpv1beta1.GovernanceInstanceSpec{
			DisplayName:      mockGovernanceInstanceName,
			LicenseType:      string(accessgovernancecpsdk.LicenseTypeNewLicense),
			TenancyNamespace: "synthetic-namespace",
			CompartmentId:    "ocid1.compartment.oc1..mock",
			IdcsAccessToken:  "synthetic-administrator-token",
			Description:      "synthetic create",
			FreeformTags: map[string]string{
				"osok-mock": "create",
			},
		},
	}
	ocimock.InitializeResource(resource, "mock-governanceinstance")
	resource.Spec = ocimock.MustJSONFixture[accessgovernancecpv1beta1.GovernanceInstanceSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "idcsAccessToken": "\u003credacted\u003e",
  "licenseType": "NEW_LICENSE",
  "tenancyNamespace": "synthetic-namespace"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "synthetic update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[accessgovernancecpsdk.CreateGovernanceInstanceDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "idcsAccessToken": "\u003credacted\u003e",
  "licenseType": "NEW_LICENSE",
  "tenancyNamespace": "synthetic-namespace"
}`)
	createdState := ocimock.MustOCIResponseFixture[accessgovernancecpsdk.GovernanceInstance](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "instanceUrl": "https://access-governance.example.test",
  "licenseType": "NEW_LICENSE",
  "lifecycleState": "ACTIVE",
  "tenancyNamespace": "synthetic-namespace",
  "timeCreated": "2026-08-31T12:00:00Z"
}`)
	createdReadStates := []accessgovernancecpsdk.GovernanceInstance{
		ocimock.MustOCIResponseFixture[accessgovernancecpsdk.GovernanceInstance](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic create",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "instanceUrl": "https://access-governance.example.test",
  "licenseType": "NEW_LICENSE",
  "lifecycleState": "ACTIVE",
  "tenancyNamespace": "synthetic-namespace",
  "timeCreated": "2026-08-31T12:00:00Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[accessgovernancecpsdk.UpdateGovernanceInstanceDetails](t, `{
  "description": "synthetic update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[accessgovernancecpsdk.GovernanceInstance](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "instanceUrl": "https://access-governance.example.test",
  "licenseType": "NEW_LICENSE",
  "lifecycleState": "ACTIVE",
  "tenancyNamespace": "synthetic-namespace",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z"
}`)
	updatedReadStates := []accessgovernancecpsdk.GovernanceInstance{
		ocimock.MustOCIResponseFixture[accessgovernancecpsdk.GovernanceInstance](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "instanceUrl": "https://access-governance.example.test",
  "licenseType": "NEW_LICENSE",
  "lifecycleState": "ACTIVE",
  "tenancyNamespace": "synthetic-namespace",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z"
}`),
	}
	deletedReadStates := []accessgovernancecpsdk.GovernanceInstance{
		ocimock.MustOCIResponseFixture[accessgovernancecpsdk.GovernanceInstance](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "instanceUrl": "https://access-governance.example.test",
  "licenseType": "NEW_LICENSE",
  "lifecycleState": "DELETING",
  "tenancyNamespace": "synthetic-namespace",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:05:00Z"
}`),
		ocimock.MustOCIResponseFixture[accessgovernancecpsdk.GovernanceInstance](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "displayName": "osok-mock-synthetic-governance-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "instanceUrl": "https://access-governance.example.test",
  "licenseType": "NEW_LICENSE",
  "lifecycleState": "DELETED",
  "tenancyNamespace": "synthetic-namespace",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:06:00Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		accessgovernancecpsdk.GovernanceInstance,
		accessgovernancecpsdk.CreateGovernanceInstanceDetails,
		accessgovernancecpsdk.UpdateGovernanceInstanceDetails,
	]{
		CollectionPath:    "/20220518/governanceInstances",
		ItemPath:          "/20220518/governanceInstances/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: append(ocimock.LifecycleStates(t, createdState, "CREATING"), append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...)...),
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ accessgovernancecpsdk.CreateGovernanceInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ accessgovernancecpsdk.GovernanceInstance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220518", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close GovernanceInstance OCI mock: %v", err)
		}
	})
	sdkClient := accessgovernancecpsdk.AccessGovernanceCPClient{BaseClient: session.BaseClient()}
	client := newGovernanceInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")},
		sdkClient,
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*accessgovernancecpv1beta1.GovernanceInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *accessgovernancecpv1beta1.GovernanceInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LicenseType, current.Spec.LicenseType) ||
				!reflect.DeepEqual(current.Status.TenancyNamespace, current.Spec.TenancyNamespace) {
				return fmt.Errorf("created GovernanceInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *accessgovernancecpv1beta1.GovernanceInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *accessgovernancecpv1beta1.GovernanceInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated GovernanceInstance status = %+v", current.Status)
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
