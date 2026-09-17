/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package odainstanceattachment

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationOdaInstanceAttachmentWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &odav1beta1.OdaInstanceAttachment{}
	ocimock.InitializeResource(resource, "mock-odainstanceattachment")
	resource.SetAnnotations(map[string]string{odaInstanceAttachmentOdaInstanceIDAnnotation: "<ocid:1>"})
	resource.Spec = ocimock.MustJSONFixture[odav1beta1.OdaInstanceAttachmentSpec](t, `{
  "attachToId": "<ocid:2>",
  "attachmentMetadata": "metadata-v1",
  "attachmentType": "FUSION",
  "owner": {
    "ownerServiceName": "owner-service-v1",
    "ownerServiceTenancy": "<ocid:3>"
  },
  "restrictedOperations": [
    "DELETE"
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[odasdk.CreateOdaInstanceAttachmentDetails](t, `{
  "attachToId": "<ocid:2>",
  "attachmentMetadata": "metadata-v1",
  "attachmentType": "FUSION",
  "owner": {
    "ownerServiceName": "owner-service-v1",
    "ownerServiceTenancy": "<ocid:3>"
  },
  "restrictedOperations": [
    "DELETE"
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[odasdk.UpdateOdaInstanceAttachmentDetails](t, `{
	"attachmentMetadata": "metadata-v1",
	"freeformTags": {
		"osok-mock": "update"
	},
	"owner": {
		"ownerServiceName": "owner-service-v1",
		"ownerServiceTenancy": "<ocid:3>"
	},
	"restrictedOperations": [
		"DELETE"
	]
}`)
	createdState := ocimock.MustOCIResponseFixture[odasdk.OdaInstanceAttachment](t, `{
  "attachToId": "<ocid:2>",
  "attachmentMetadata": "metadata-v1",
  "attachmentType": "FUSION",
  "definedTags": null,
  "freeformTags": null,
  "id": "<ocid:5>",
  "instanceId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "owner": {
    "ownerServiceName": "owner-service-v1",
    "ownerServiceTenancy": "<ocid:3>"
  },
  "restrictedOperations": [
    "DELETE"
  ],
  "timeCreated": null,
  "timeLastUpdate": null
}`)
	updatedState := ocimock.MustOCIResponseFixture[odasdk.OdaInstanceAttachment](t, `{
  "attachToId": "<ocid:2>",
  "attachmentMetadata": "metadata-v1",
  "attachmentType": "FUSION",
  "definedTags": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "instanceId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "owner": {
    "ownerServiceName": "owner-service-v1",
    "ownerServiceTenancy": "<ocid:3>"
  },
  "restrictedOperations": [
    "DELETE"
  ],
  "timeCreated": null,
  "timeLastUpdate": null
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[odasdk.WorkRequest](t, `{
  "compartmentId": null,
  "id": "<ocid:4>",
  "odaInstanceId": "<ocid:1>",
  "percentComplete": 50,
  "requestAction": "CREATE_ODA_INSTANCE_ATTACHMENT",
  "resourceId": "<ocid:5>",
  "resources": [
    {
      "resourceAction": "CREATE_ODA_INSTANCE_ATTACHMENT",
      "resourceId": "<ocid:5>",
      "resourceType": null,
      "resourceUri": null,
      "status": "",
      "statusMessage": null
    }
  ],
  "status": "SUCCEEDED",
  "statusMessage": null,
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[odasdk.WorkRequest](t, `{
  "compartmentId": null,
  "id": "wr-update",
  "odaInstanceId": "<ocid:1>",
  "percentComplete": 50,
  "requestAction": "UPDATE_ODA_INSTANCE_ATTACHMENT",
  "resourceId": "<ocid:5>",
  "resources": [
    {
      "resourceAction": "UPDATE_ODA_INSTANCE_ATTACHMENT",
      "resourceId": "<ocid:5>",
      "resourceType": null,
      "resourceUri": null,
      "status": "",
      "statusMessage": null
    }
  ],
  "status": "SUCCEEDED",
  "statusMessage": null,
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[odasdk.WorkRequest](t, `{
  "compartmentId": null,
  "id": "<ocid:6>",
  "odaInstanceId": "<ocid:1>",
  "percentComplete": 50,
  "requestAction": "DELETE_ODA_INSTANCE_ATTACHMENT",
  "resourceId": "<ocid:5>",
  "resources": [
    {
      "resourceAction": "DELETE_ODA_INSTANCE_ATTACHMENT",
      "resourceId": "<ocid:5>",
      "resourceType": null,
      "resourceUri": null,
      "status": "",
      "statusMessage": null
    }
  ],
  "status": "SUCCEEDED",
  "statusMessage": null,
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[odasdk.OdaInstanceAttachment, odasdk.CreateOdaInstanceAttachmentDetails, odasdk.UpdateOdaInstanceAttachmentDetails]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/attachments", ItemPath: "/20190506/odaInstances/<ocid:1>/attachments/<ocid:5>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ odasdk.CreateOdaInstanceAttachmentDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20190506/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20190506/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20190506/workRequests/<ocid:6>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://digitalassistant-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := odasdk.OdaClient{BaseClient: session.BaseClient()}
	client := newOdaInstanceAttachmentServiceClientWithOCIClient(log, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.OdaInstanceAttachment]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.OdaInstanceAttachment) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.AttachToId != resource.Spec.AttachToId || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OdaInstanceAttachment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.OdaInstanceAttachment) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.OdaInstanceAttachment) error {
			if current.Status.FreeformTags["osok-mock"] != "update" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OdaInstanceAttachment status = %+v", current.Status)
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
