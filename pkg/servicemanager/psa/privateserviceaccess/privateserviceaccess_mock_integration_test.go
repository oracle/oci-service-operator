/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateserviceaccess

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	psasdk "github.com/oracle/oci-go-sdk/v65/psa"
	psav1beta1 "github.com/oracle/oci-service-operator/api/psa/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationPrivateServiceAccessWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &psav1beta1.PrivateServiceAccess{}
	ocimock.InitializeResource(resource, "mock-privateserviceaccess")
	resource.Spec = ocimock.MustJSONFixture[psav1beta1.PrivateServiceAccessSpec](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "private service access",
  "displayName": "privateserviceaccess-sample",
  "freeformTags": {
    "env": "dev"
  },
  "ipv4Ip": "10.0.0.12",
  "nsgIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "securityAttributes": {
    "Oracle-DataSecurity-ZPR": {
      "mode": "audit"
    }
  },
  "serviceId": "<ocid:4>",
  "subnetId": "<ocid:5>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "private service access updated"
}`)

	createRequest := ocimock.MustJSONFixture[psasdk.CreatePrivateServiceAccessDetails](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "private service access",
  "displayName": "privateserviceaccess-sample",
  "freeformTags": {
    "env": "dev"
  },
  "ipv4Ip": "10.0.0.12",
  "nsgIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "securityAttributes": {
    "Oracle-DataSecurity-ZPR": {
      "mode": "audit"
    }
  },
  "serviceId": "<ocid:4>",
  "subnetId": "<ocid:5>"
}`)
	updateRequest := ocimock.MustJSONFixture[psasdk.UpdatePrivateServiceAccessDetails](t, `{
  "description": "private service access updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[psasdk.PrivateServiceAccess](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "private service access",
  "displayName": "privateserviceaccess-sample",
  "fqdns": [
    "service.example.oraclecloud.com"
  ],
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:7>",
  "ipv4Ip": "10.0.0.12",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "securityAttributes": {
    "Oracle-DataSecurity-ZPR": {
      "mode": "audit"
    }
  },
  "serviceId": "<ocid:4>",
  "subnetId": "<ocid:5>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null,
  "vcnId": "<ocid:8>",
  "vnicId": "<ocid:9>"
}`)
	updatedState := ocimock.MustOCIResponseFixture[psasdk.PrivateServiceAccess](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "costCenter": "42"
    }
  },
  "description": "private service access updated",
  "displayName": "privateserviceaccess-sample",
  "fqdns": [
    "service.example.oraclecloud.com"
  ],
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:7>",
  "ipv4Ip": "10.0.0.12",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "securityAttributes": {
    "Oracle-DataSecurity-ZPR": {
      "mode": "audit"
    }
  },
  "serviceId": "<ocid:4>",
  "subnetId": "<ocid:5>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null,
  "vcnId": "<ocid:8>",
  "vnicId": "<ocid:9>"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[psasdk.WorkRequest](t, `{
  "compartmentId": null,
  "id": "<ocid:6>",
  "operationType": "CREATE_PRIVATE_SERVICE_ACCESS",
  "percentComplete": null,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "PrivateServiceAccess",
      "entityUri": "/20240301/privateServiceAccess/<ocid:7>",
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[psasdk.WorkRequest](t, `{
  "compartmentId": null,
  "id": "wr-update",
  "operationType": "UPDATE_PRIVATE_SERVICE_ACCESS",
  "percentComplete": null,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "PrivateServiceAccess",
      "entityUri": "/20240301/privateServiceAccess/<ocid:7>",
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[psasdk.WorkRequest](t, `{
  "compartmentId": null,
  "id": "<ocid:10>",
  "operationType": "DELETE_PRIVATE_SERVICE_ACCESS",
  "percentComplete": null,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "PrivateServiceAccess",
      "entityUri": "/20240301/privateServiceAccess/<ocid:7>",
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[psasdk.PrivateServiceAccess, psasdk.CreatePrivateServiceAccessDetails, psasdk.UpdatePrivateServiceAccessDetails]{
		CollectionPath:     "/20240301/privateServiceAccess",
		ItemPath:           "/20240301/privateServiceAccess/<ocid:7>",
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
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:10>"}},
		ValidateCreate: func(request ocimock.Request, _ psasdk.CreatePrivateServiceAccessDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20240301/psaWorkRequests/<ocid:6>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20240301/psaWorkRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20240301/psaWorkRequests/<ocid:10>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://psasvc.us-ashburn-1.oci.oraclecloud.com", BasePath: "20240301", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := psasdk.PrivateServiceAccessClient{BaseClient: session.BaseClient()}
	client := newPrivateServiceAccessServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*psav1beta1.PrivateServiceAccess]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *psav1beta1.PrivateServiceAccess) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created PrivateServiceAccess status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *psav1beta1.PrivateServiceAccess) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *psav1beta1.PrivateServiceAccess) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated PrivateServiceAccess status = %+v", current.Status)
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
