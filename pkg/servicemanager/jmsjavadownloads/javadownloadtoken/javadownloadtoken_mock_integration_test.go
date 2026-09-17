/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package javadownloadtoken

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	jmsjavadownloadssdk "github.com/oracle/oci-go-sdk/v65/jmsjavadownloads"
	jmsjavadownloadsv1beta1 "github.com/oracle/oci-service-operator/api/jmsjavadownloads/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationJavaDownloadTokenWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &jmsjavadownloadsv1beta1.JavaDownloadToken{}
	ocimock.InitializeResource(resource, "mock-javadownloadtoken")
	resource.Spec = ocimock.MustJSONFixture[jmsjavadownloadsv1beta1.JavaDownloadTokenSpec](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "token description",
  "displayName": "token",
  "freeformTags": {
    "managed-by": "osok"
  },
  "isDefault": true,
  "javaVersion": "17",
  "licenseType": [
    "NFTC"
  ],
  "timeExpires": "2030-01-01T00:00:00Z"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "token description updated"
}`)

	createRequest := ocimock.MustJSONFixture[jmsjavadownloadssdk.CreateJavaDownloadTokenDetails](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "token description",
  "displayName": "token",
  "freeformTags": {
    "managed-by": "osok"
  },
  "isDefault": true,
  "javaVersion": "17",
  "licenseType": [
    "NFTC"
  ],
  "timeExpires": "2030-01-01T00:00:00Z"
}`)
	updateRequest := ocimock.MustJSONFixture[jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails](t, `{
  "description": "token description updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[jmsjavadownloadssdk.JavaDownloadToken](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": null,
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "token description",
  "displayName": "token",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "<ocid:3>",
  "isDefault": true,
  "javaVersion": "17",
  "lastUpdatedBy": null,
  "licenseType": [
    "NFTC"
  ],
  "lifecycleState": "ACTIVE",
  "systemTags": null,
  "timeCreated": "2029-01-01T00:00:00Z",
  "timeExpires": "2030-01-01T00:00:00Z",
  "timeLastUsed": null,
  "timeUpdated": null,
  "value": null
}`)
	updatedState := ocimock.MustOCIResponseFixture[jmsjavadownloadssdk.JavaDownloadToken](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": null,
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "token description updated",
  "displayName": "token",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "<ocid:3>",
  "isDefault": true,
  "javaVersion": "17",
  "lastUpdatedBy": null,
  "licenseType": [
    "NFTC"
  ],
  "lifecycleState": "ACTIVE",
  "systemTags": null,
  "timeCreated": "2029-01-01T00:00:00Z",
  "timeExpires": "2030-01-01T00:00:00Z",
  "timeLastUsed": null,
  "timeUpdated": null,
  "value": null
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[jmsjavadownloadssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_JAVA_DOWNLOAD_TOKEN",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "JavaDownloadToken",
      "entityUri": null,
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2030-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[jmsjavadownloadssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "wr-update",
  "operationType": "UPDATE_JAVA_DOWNLOAD_TOKEN",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "JavaDownloadToken",
      "entityUri": null,
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2030-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[jmsjavadownloadssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "DELETE_JAVA_DOWNLOAD_TOKEN",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "JavaDownloadToken",
      "entityUri": null,
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2030-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[jmsjavadownloadssdk.JavaDownloadToken, jmsjavadownloadssdk.CreateJavaDownloadTokenDetails, jmsjavadownloadssdk.UpdateJavaDownloadTokenDetails]{
		CollectionPath:     "/20230601/javaDownloadTokens",
		ItemPath:           "/20230601/javaDownloadTokens/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		ValidateCreate: func(request ocimock.Request, _ jmsjavadownloadssdk.CreateJavaDownloadTokenDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20230601/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20230601/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20230601/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://javamanagementservice-download.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := jmsjavadownloadssdk.JavaDownloadClient{BaseClient: session.BaseClient()}
	client := newJavaDownloadTokenServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*jmsjavadownloadsv1beta1.JavaDownloadToken]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *jmsjavadownloadsv1beta1.JavaDownloadToken) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created JavaDownloadToken status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *jmsjavadownloadsv1beta1.JavaDownloadToken) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *jmsjavadownloadsv1beta1.JavaDownloadToken) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated JavaDownloadToken status = %+v", current.Status)
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
