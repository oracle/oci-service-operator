/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vbsinstance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	vbsinstsdk "github.com/oracle/oci-go-sdk/v65/vbsinst"
	vbsinstv1beta1 "github.com/oracle/oci-service-operator/api/vbsinst/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationVbsInstanceWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &vbsinstv1beta1.VbsInstance{}
	ocimock.InitializeResource(resource, "mock-vbsinstance")
	resource.Spec = ocimock.MustJSONFixture[vbsinstv1beta1.VbsInstanceSpec](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "vbsinstance-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "isResourceUsageAgreementGranted": true,
  "name": "vbsinstance-sample",
  "resourceCompartmentId": "<ocid:2>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "vbsinstance-sample-updated"
}`)

	createRequest := ocimock.MustJSONFixture[vbsinstsdk.CreateVbsInstanceDetails](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "vbsinstance-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "isResourceUsageAgreementGranted": true,
  "name": "vbsinstance-sample",
  "resourceCompartmentId": "<ocid:2>"
}`)
	updateRequest := ocimock.MustJSONFixture[vbsinstsdk.UpdateVbsInstanceDetails](t, `{
  "displayName": "vbsinstance-sample-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[vbsinstsdk.VbsInstance](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "vbsinstance-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:4>",
  "isResourceUsageAgreementGranted": true,
  "lifecycleState": "ACTIVE",
  "lifecyleDetails": "state ACTIVE",
  "name": "vbsinstance-sample",
  "resourceCompartmentId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z",
  "vbsAccessUrl": "https://vbs.example"
}`)
	updatedState := ocimock.MustOCIResponseFixture[vbsinstsdk.VbsInstance](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "vbsinstance-sample-updated",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:4>",
  "isResourceUsageAgreementGranted": true,
  "lifecycleState": "ACTIVE",
  "lifecyleDetails": "state ACTIVE",
  "name": "vbsinstance-sample",
  "resourceCompartmentId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z",
  "vbsAccessUrl": "https://vbs.example"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[vbsinstsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_VBS_INSTANCE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "VbsInstance",
      "entityUri": null,
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[vbsinstsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "wr-update",
  "operationType": "UPDATE_VBS_INSTANCE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "VbsInstance",
      "entityUri": null,
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[vbsinstsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_VBS_INSTANCE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "VbsInstance",
      "entityUri": null,
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[vbsinstsdk.VbsInstance, vbsinstsdk.CreateVbsInstanceDetails, vbsinstsdk.UpdateVbsInstanceDetails]{
		CollectionPath:     "/20180828/vbsInstances",
		ItemPath:           "/20180828/vbsInstances/<ocid:4>",
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
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ vbsinstsdk.CreateVbsInstanceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20180828/workRequests/<ocid:3>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20180828/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20180828/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://vbstudio.us-ashburn-1.ocp.oraclecloud.com", BasePath: "20180828", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := vbsinstsdk.VbsInstanceClient{BaseClient: session.BaseClient()}
	client := newVbsInstanceServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*vbsinstv1beta1.VbsInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *vbsinstv1beta1.VbsInstance) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created VbsInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *vbsinstv1beta1.VbsInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *vbsinstv1beta1.VbsInstance) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated VbsInstance status = %+v", current.Status)
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
