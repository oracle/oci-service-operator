/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package vbinstance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	visualbuildersdk "github.com/oracle/oci-go-sdk/v65/visualbuilder"
	visualbuilderv1beta1 "github.com/oracle/oci-service-operator/api/visualbuilder/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationVbInstanceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &visualbuilderv1beta1.VbInstance{}
	ocimock.InitializeResource(resource, "mock-vbinstance")
	resource.Spec = ocimock.MustJSONFixture[visualbuilderv1beta1.VbInstanceSpec](t, `{
  "alternateCustomEndpoints": [

  ],
  "compartmentId": "<ocid:1>",
  "displayName": "vb-instance-minimal",
  "nodeCount": 2
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "alternateCustomEndpoints": [

  ],
  "displayName": "vb-instance-updated",
  "nodeCount": 2
}`)

	createRequest := ocimock.MustJSONFixture[visualbuildersdk.CreateVbInstanceDetails](t, `{
  "alternateCustomEndpoints": [

  ],
  "compartmentId": "<ocid:1>",
  "displayName": "vb-instance-minimal",
  "nodeCount": 2
}`)
	updateRequest := ocimock.MustJSONFixture[visualbuildersdk.UpdateVbInstanceDetails](t, `{
  "alternateCustomEndpoints": [

  ],
  "displayName": "vb-instance-updated",
  "nodeCount": 2
}`)
	createdState := ocimock.MustOCIResponseFixture[visualbuildersdk.VbInstance](t, `{
  "id": "<ocid:3>",
  "displayName": "vb-instance-minimal",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "nodeCount": 2,
  "freeformTags": {
  },
  "definedTags": {
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[visualbuildersdk.VbInstance](t, `{
  "id": "<ocid:3>",
  "displayName": "vb-instance-updated",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "nodeCount": 2,
  "freeformTags": {
  },
  "definedTags": {
  }
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[visualbuildersdk.WorkRequest](t, `{
  "id": "<ocid:2>",
  "operationType": "CREATE_VB_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "VbInstance",
      "actionType": "CREATED",
      "identifier": "<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[visualbuildersdk.WorkRequest](t, `{
  "id": "<ocid:4>",
  "operationType": "UPDATE_VB_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "VbInstance",
      "actionType": "UPDATED",
      "identifier": "<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[visualbuildersdk.WorkRequest](t, `{
  "id": "<ocid:5>",
  "operationType": "DELETE_VB_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "VbInstance",
      "actionType": "DELETED",
      "identifier": "<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[visualbuildersdk.VbInstance, visualbuildersdk.CreateVbInstanceDetails, visualbuildersdk.UpdateVbInstanceDetails]{
		CollectionPath: "/20210601/vbInstances", ItemPath: "/20210601/vbInstances/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ visualbuildersdk.CreateVbInstanceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20210601/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest),
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20210601/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest),
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20210601/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://visualbuilder.us-ashburn-1.ocp.oraclecloud.com", BasePath: "20210601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	_ = log
	client := newVbInstanceServiceClientWithOCIClient(visualbuildersdk.VbInstanceClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*visualbuilderv1beta1.VbInstance]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *visualbuilderv1beta1.VbInstance) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created VbInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *visualbuilderv1beta1.VbInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *visualbuilderv1beta1.VbInstance) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated VbInstance status = %+v", current.Status)
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
