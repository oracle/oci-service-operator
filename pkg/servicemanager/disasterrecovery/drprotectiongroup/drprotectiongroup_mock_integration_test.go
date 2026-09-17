/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drprotectiongroup

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	disasterrecoverysdk "github.com/oracle/oci-go-sdk/v65/disasterrecovery"
	disasterrecoveryv1beta1 "github.com/oracle/oci-service-operator/api/disasterrecovery/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationDrProtectionGroupWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &disasterrecoveryv1beta1.DrProtectionGroup{}
	ocimock.InitializeResource(resource, "mock-drprotectiongroup")
	resource.Spec = ocimock.MustJSONFixture[disasterrecoveryv1beta1.DrProtectionGroupSpec](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-dr-protection-group-v1",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>"
  },
  "members": [

  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "definedTags": {
  },
  "displayName": "osok-mock-dr-protection-group-v1-updated",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>"
  },
  "members": [

  ]
}`)

	createRequest := ocimock.MustJSONFixture[disasterrecoverysdk.CreateDrProtectionGroupDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-dr-protection-group-v1",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>"
  },
  "members": [

  ]
}`)
	updateRequest := ocimock.MustJSONFixture[disasterrecoverysdk.UpdateDrProtectionGroupDetails](t, `{
  "definedTags": {
  },
  "displayName": "osok-mock-dr-protection-group-v1-updated",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>"
  },
  "members": [

  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[disasterrecoverysdk.DrProtectionGroup](t, `{
  "authType": "OBO",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T17:43:17.317Z"
    }
  },
  "displayName": "osok-mock-dr-protection-group-v1",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "id": "<ocid:3>",
  "lifeCycleDetails": null,
  "lifecycleState": "ACTIVE",
  "lifecycleSubState": null,
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>",
    "object": null
  },
  "members": null,
  "peerId": null,
  "peerRegion": null,
  "role": "UNCONFIGURED",
  "systemTags": {
  },
  "timeCreated": "2026-09-04T17:43:18.257Z",
  "timeUpdated": "2026-09-04T17:44:01.955Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[disasterrecoverysdk.DrProtectionGroup](t, `{
  "authType": "OBO",
  "compartmentId": "<ocid:1>",
  "definedTags": {
  },
  "displayName": "osok-mock-dr-protection-group-v1-updated",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "id": "<ocid:3>",
  "lifeCycleDetails": null,
  "lifecycleState": "ACTIVE",
  "lifecycleSubState": null,
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>",
    "object": null
  },
  "members": null,
  "peerId": null,
  "peerRegion": null,
  "role": "UNCONFIGURED",
  "systemTags": {
  },
  "timeCreated": "2026-09-04T17:43:18.257Z",
  "timeUpdated": "2026-09-04T17:44:37.119Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[disasterrecoverysdk.DrProtectionGroup](t, `{
  "authType": "OBO",
  "compartmentId": "<ocid:1>",
  "definedTags": {
  },
  "displayName": "osok-mock-dr-protection-group-v1-updated",
  "freeformTags": {
    "managed-by": "osok-mock"
  },
  "id": "<ocid:3>",
  "lifeCycleDetails": null,
  "lifecycleState": "DELETED",
  "lifecycleSubState": null,
  "logLocation": {
    "bucket": "<binding:dr-log-bucket>",
    "namespace": "<binding:objectstorage-namespace>",
    "object": null
  },
  "members": null,
  "peerId": null,
  "peerRegion": null,
  "role": "UNCONFIGURED",
  "systemTags": {
  },
  "timeCreated": "2026-09-04T17:43:18.257Z",
  "timeUpdated": "2026-09-04T17:44:59.874Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[disasterrecoverysdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_DR_PROTECTION_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "drProtectionGroup",
      "entityUri": "/drProtectionGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-04T17:43:18.258Z",
  "timeFinished": "2026-09-04T17:44:01.955Z",
  "timeStarted": "2026-09-04T17:43:59.412Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[disasterrecoverysdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_DR_PROTECTION_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "drProtectionGroup",
      "entityUri": "/drProtectionGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-04T17:44:07.889Z",
  "timeFinished": "2026-09-04T17:44:37.119Z",
  "timeStarted": "2026-09-04T17:44:34.138Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[disasterrecoverysdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_DR_PROTECTION_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "drProtectionGroup",
      "entityUri": "/drProtectionGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-04T17:44:38.581Z",
  "timeFinished": "2026-09-04T17:44:59.864Z",
  "timeStarted": "2026-09-04T17:44:59.212Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[disasterrecoverysdk.DrProtectionGroup, disasterrecoverysdk.CreateDrProtectionGroupDetails, disasterrecoverysdk.UpdateDrProtectionGroupDetails]{
		CollectionPath:    "/20220125/drProtectionGroups",
		ItemPath:          "/20220125/drProtectionGroups/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		DeletedState:      &deletedState, RequireDeleteRead: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ disasterrecoverysdk.CreateDrProtectionGroupDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20220125/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20220125/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20220125/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://disaster-recovery.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220125", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := disasterrecoverysdk.DisasterRecoveryClient{BaseClient: session.BaseClient()}
	hooks := newDrProtectionGroupDefaultRuntimeHooks(sdkClient)
	applyDrProtectionGroupRuntimeHooks(&hooks, sdkClient, nil)
	manager := &DrProtectionGroupServiceManager{Log: log}
	client := wrapDrProtectionGroupGeneratedClient(hooks, defaultDrProtectionGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*disasterrecoveryv1beta1.DrProtectionGroup](buildDrProtectionGroupGeneratedRuntimeConfig(manager, hooks)),
	})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*disasterrecoveryv1beta1.DrProtectionGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *disasterrecoveryv1beta1.DrProtectionGroup) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DrProtectionGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *disasterrecoveryv1beta1.DrProtectionGroup) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *disasterrecoveryv1beta1.DrProtectionGroup) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DrProtectionGroup status = %+v", current.Status)
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
