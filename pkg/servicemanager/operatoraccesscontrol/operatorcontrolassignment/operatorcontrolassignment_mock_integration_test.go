/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package operatorcontrolassignment

import (
	"context"
	"fmt"
	operatoraccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOperatorControlAssignmentLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newOperatorControlAssignmentResource()
	ocimock.InitializeResource(resource, "mock-operatorcontrolassignment")
	resource.Spec = ocimock.MustJSONFixture[operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec](t, `{
  "comment": "initial assignment",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "dev"
  },
  "isAutoApproveDuringMaintenance": true,
  "isEnforcedAlways": true,
  "isHypervisorLogForwarded": true,
  "isLogForwarded": true,
  "operatorControlId": "\u003cocid:2\u003e",
  "remoteSyslogServerAddress": "192.0.2.10",
  "remoteSyslogServerCACert": "ca-cert",
  "remoteSyslogServerPort": 6514,
  "resourceCompartmentId": "\u003cocid:3\u003e",
  "resourceId": "\u003cocid:4\u003e",
  "resourceName": "target-exacc",
  "resourceType": "EXACC",
  "timeAssignmentFrom": "2026-04-29T10:00:00Z",
  "timeAssignmentTo": "2026-04-30T10:00:00Z"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "isEnforcedAlways": true
}`)
	createRequest := ocimock.MustJSONFixture[operatoraccesscontrolsdk.CreateOperatorControlAssignmentDetails](t, `{
  "comment": "initial assignment",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "dev"
  },
  "isAutoApproveDuringMaintenance": true,
  "isEnforcedAlways": true,
  "isHypervisorLogForwarded": true,
  "isLogForwarded": true,
  "operatorControlId": "\u003cocid:2\u003e",
  "remoteSyslogServerAddress": "192.0.2.10",
  "remoteSyslogServerCACert": "ca-cert",
  "remoteSyslogServerPort": 6514,
  "resourceCompartmentId": "\u003cocid:3\u003e",
  "resourceId": "\u003cocid:4\u003e",
  "resourceName": "target-exacc",
  "resourceType": "EXACC",
  "timeAssignmentFrom": "2026-04-29T10:00:00Z",
  "timeAssignmentTo": "2026-04-30T10:00:00Z"
}`)
	createdState := ocimock.MustOCIResponseFixture[operatoraccesscontrolsdk.OperatorControlAssignment](t, `{
  "comment": "initial assignment",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:5\u003e",
  "isAutoApproveDuringMaintenance": true,
  "isEnforcedAlways": true,
  "isHypervisorLogForwarded": true,
  "isLogForwarded": true,
  "lifecycleState": "APPLIED",
  "operatorControlId": "\u003cocid:2\u003e",
  "remoteSyslogServerAddress": "192.0.2.10",
  "remoteSyslogServerCACert": "ca-cert",
  "remoteSyslogServerPort": 6514,
  "resourceCompartmentId": "\u003cocid:3\u003e",
  "resourceId": "\u003cocid:4\u003e",
  "resourceName": "target-exacc",
  "resourceType": "EXACC",
  "timeAssignmentFrom": "2026-04-29T10:00:00Z",
  "timeAssignmentTo": "2026-04-30T10:00:00Z"
}`)
	updateRequest := ocimock.MustJSONFixture[operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails](t, `{
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "isEnforcedAlways": true
}`)
	updatedState := ocimock.MustOCIResponseFixture[operatoraccesscontrolsdk.OperatorControlAssignment](t, `{
  "comment": "initial assignment",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:5\u003e",
  "isAutoApproveDuringMaintenance": true,
  "isEnforcedAlways": true,
  "isHypervisorLogForwarded": true,
  "isLogForwarded": true,
  "lifecycleState": "APPLIED",
  "operatorControlId": "\u003cocid:2\u003e",
  "remoteSyslogServerAddress": "192.0.2.10",
  "remoteSyslogServerCACert": "ca-cert",
  "remoteSyslogServerPort": 6514,
  "resourceCompartmentId": "\u003cocid:3\u003e",
  "resourceId": "\u003cocid:4\u003e",
  "resourceName": "target-exacc",
  "resourceType": "EXACC",
  "timeAssignmentFrom": "2026-04-29T10:00:00Z",
  "timeAssignmentTo": "2026-04-30T10:00:00Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		operatoraccesscontrolsdk.OperatorControlAssignment,
		operatoraccesscontrolsdk.CreateOperatorControlAssignmentDetails,
		operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails,
	]{
		CollectionPath:    "/20200630/operatorControlAssignments",
		ItemPath:          "/20200630/operatorControlAssignments/<ocid:5>",
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
		ValidateCreate: func(request ocimock.Request, _ operatoraccesscontrolsdk.CreateOperatorControlAssignmentDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ operatoraccesscontrolsdk.OperatorControlAssignment) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OperatorControlAssignment OCI mock: %v", err)
		}
	})
	sdkClient := operatoraccesscontrolsdk.OperatorControlAssignmentClient{BaseClient: session.BaseClient()}
	client := newOperatorControlAssignmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *operatoraccesscontrolv1beta1.OperatorControlAssignment) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "APPLIED" ||
				!reflect.DeepEqual(current.Status.Comment, current.Spec.Comment) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsAutoApproveDuringMaintenance, current.Spec.IsAutoApproveDuringMaintenance) ||
				!reflect.DeepEqual(current.Status.IsEnforcedAlways, current.Spec.IsEnforcedAlways) ||
				!reflect.DeepEqual(current.Status.IsHypervisorLogForwarded, current.Spec.IsHypervisorLogForwarded) ||
				!reflect.DeepEqual(current.Status.IsLogForwarded, current.Spec.IsLogForwarded) ||
				!reflect.DeepEqual(current.Status.OperatorControlId, current.Spec.OperatorControlId) ||
				!reflect.DeepEqual(current.Status.RemoteSyslogServerAddress, current.Spec.RemoteSyslogServerAddress) ||
				!reflect.DeepEqual(current.Status.RemoteSyslogServerCACert, current.Spec.RemoteSyslogServerCACert) ||
				!reflect.DeepEqual(current.Status.RemoteSyslogServerPort, current.Spec.RemoteSyslogServerPort) ||
				!reflect.DeepEqual(current.Status.ResourceCompartmentId, current.Spec.ResourceCompartmentId) ||
				!reflect.DeepEqual(current.Status.ResourceId, current.Spec.ResourceId) ||
				!reflect.DeepEqual(current.Status.ResourceName, current.Spec.ResourceName) ||
				!reflect.DeepEqual(current.Status.ResourceType, current.Spec.ResourceType) ||
				!reflect.DeepEqual(current.Status.TimeAssignmentFrom, current.Spec.TimeAssignmentFrom) ||
				!reflect.DeepEqual(current.Status.TimeAssignmentTo, current.Spec.TimeAssignmentTo) {
				return fmt.Errorf("created OperatorControlAssignment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *operatoraccesscontrolv1beta1.OperatorControlAssignment) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *operatoraccesscontrolv1beta1.OperatorControlAssignment) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "APPLIED" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsEnforcedAlways, current.Spec.IsEnforcedAlways) {
				return fmt.Errorf("updated OperatorControlAssignment status = %+v", current.Status)
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
