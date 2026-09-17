/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package oceinstance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	ocesdk "github.com/oracle/oci-go-sdk/v65/oce"
	ocev1beta1 "github.com/oracle/oci-service-operator/api/oce/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationOceInstanceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &ocev1beta1.OceInstance{}
	ocimock.InitializeResource(resource, "mock-oceinstance")
	resource.Spec = ocimock.MustJSONFixture[ocev1beta1.OceInstanceSpec](t, `{
  "addOnFeatures": [
    "CONTENT_CAPTURE"
  ],
  "adminEmail": "admin@example.com",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime test instance",
  "drRegion": "us-phoenix-1",
  "freeformTags": {
    "environment": "dev"
  },
  "idcsAccessToken": "<redacted>",
  "instanceAccessType": "PRIVATE",
  "instanceLicenseType": "NEW",
  "instanceUsageType": "PRIMARY",
  "name": "oce-runtime-test",
  "objectStorageNamespace": "oce-namespace",
  "tenancyId": "<ocid:2>",
  "tenancyName": "oce-tenancy",
  "upgradeSchedule": "UPGRADE_IMMEDIATELY",
  "wafPrimaryDomain": "oce.example.com"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "runtime test instance updated"
}`)

	createRequest := ocimock.MustJSONFixture[ocesdk.CreateOceInstanceDetails](t, `{
  "addOnFeatures": [
    "CONTENT_CAPTURE"
  ],
  "adminEmail": "admin@example.com",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime test instance",
  "drRegion": "us-phoenix-1",
  "freeformTags": {
    "environment": "dev"
  },
  "idcsAccessToken": "<redacted>",
  "instanceAccessType": "PRIVATE",
  "instanceLicenseType": "NEW",
  "instanceUsageType": "PRIMARY",
  "name": "oce-runtime-test",
  "objectStorageNamespace": "oce-namespace",
  "tenancyId": "<ocid:2>",
  "tenancyName": "oce-tenancy",
  "upgradeSchedule": "UPGRADE_IMMEDIATELY",
  "wafPrimaryDomain": "oce.example.com"
}`)
	updateRequest := ocimock.MustJSONFixture[ocesdk.UpdateOceInstanceDetails](t, `{
  "description": "runtime test instance updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[ocesdk.OceInstance](t, `{
  "id": "<ocid:4>",
  "guid": "guid-1",
  "compartmentId": "<ocid:1>",
  "name": "oce-runtime-test",
  "tenancyId": "<ocid:2>",
  "tenancyName": "oce-tenancy",
  "objectStorageNamespace": "oce-namespace",
  "adminEmail": "admin@example.com",
  "description": "runtime test instance",
  "instanceUsageType": "PRIMARY",
  "addOnFeatures": [
    "CONTENT_CAPTURE"
  ],
  "upgradeSchedule": "UPGRADE_IMMEDIATELY",
  "wafPrimaryDomain": "oce.example.com",
  "instanceAccessType": "PRIVATE",
  "instanceLicenseType": "NEW",
  "lifecycleState": "ACTIVE",
  "drRegion": "us-phoenix-1",
  "freeformTags": {
    "environment": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[ocesdk.OceInstance](t, `{
  "id": "<ocid:4>",
  "guid": "guid-1",
  "compartmentId": "<ocid:1>",
  "name": "oce-runtime-test",
  "tenancyId": "<ocid:2>",
  "tenancyName": "oce-tenancy",
  "objectStorageNamespace": "oce-namespace",
  "adminEmail": "admin@example.com",
  "description": "runtime test instance updated",
  "instanceUsageType": "PRIMARY",
  "addOnFeatures": [
    "CONTENT_CAPTURE"
  ],
  "upgradeSchedule": "UPGRADE_IMMEDIATELY",
  "wafPrimaryDomain": "oce.example.com",
  "instanceAccessType": "PRIVATE",
  "instanceLicenseType": "NEW",
  "lifecycleState": "ACTIVE",
  "drRegion": "us-phoenix-1",
  "freeformTags": {
    "environment": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  }
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[ocesdk.WorkRequest](t, `{
  "id": "<ocid:3>",
  "operationType": "CREATE_OCE_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "OceInstance",
      "actionType": "CREATED",
      "identifier": "<ocid:4>",
      "entityUri": "/oceInstances/<ocid:4>"
    }
  ],
  "percentComplete": 100
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[ocesdk.WorkRequest](t, `{
  "id": "<ocid:5>",
  "operationType": "UPDATE_OCE_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "OceInstance",
      "actionType": "UPDATED",
      "identifier": "<ocid:4>",
      "entityUri": "/oceInstances/<ocid:4>"
    }
  ],
  "percentComplete": 100
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[ocesdk.WorkRequest](t, `{
  "id": "<ocid:6>",
  "operationType": "DELETE_OCE_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "OceInstance",
      "actionType": "DELETED",
      "identifier": "<ocid:4>",
      "entityUri": "/oceInstances/<ocid:4>"
    }
  ],
  "percentComplete": 100
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[ocesdk.OceInstance, ocesdk.CreateOceInstanceDetails, ocesdk.UpdateOceInstanceDetails]{
		CollectionPath: "/20190912/oceInstances", ItemPath: "/20190912/oceInstances/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeArray, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ ocesdk.CreateOceInstanceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20190912/workRequests/<ocid:3>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest),
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20190912/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest),
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20190912/workRequests/<ocid:6>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cp.oce.us-ashburn-1.ocp.oraclecloud.com", BasePath: "20190912", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := ocesdk.OceInstanceClient{BaseClient: session.BaseClient()}
	client := newOceInstanceServiceClientWithOCIClient(log, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*ocev1beta1.OceInstance]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *ocev1beta1.OceInstance) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Name != resource.Spec.Name || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OceInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *ocev1beta1.OceInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *ocev1beta1.OceInstance) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OceInstance status = %+v", current.Status)
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
