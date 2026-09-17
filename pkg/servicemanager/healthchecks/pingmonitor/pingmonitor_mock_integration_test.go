/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package pingmonitor

import (
	"context"
	"fmt"
	healthcheckssdk "github.com/oracle/oci-go-sdk/v65/healthchecks"
	healthchecksv1beta1 "github.com/oracle/oci-service-operator/api/healthchecks/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationPingMonitorEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &healthchecksv1beta1.PingMonitor{}
	ocimock.InitializeResource(resource, "mock-pingmonitor")
	resource.Spec = ocimock.MustJSONFixture[healthchecksv1beta1.PingMonitorSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-ping-monitor-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "intervalInSeconds": 60,
  "port": 443,
  "protocol": "TCP",
  "targets": [
    "example.com"
  ],
  "timeoutInSeconds": 10
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-ping-monitor-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "intervalInSeconds": 30
}`)
	createRequest := ocimock.MustJSONFixture[healthcheckssdk.CreatePingMonitorDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-ping-monitor-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "intervalInSeconds": 60,
  "port": 443,
  "protocol": "TCP",
  "targets": [
    "example.com"
  ],
  "timeoutInSeconds": 10
}`)
	createdState := ocimock.MustOCIResponseFixture[healthcheckssdk.PingMonitor](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T17:48:20.639Z"
    }
  },
  "displayName": "osok-mock-ping-monitor-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hash": null,
  "homeRegion": "us-ashburn-1",
  "id": "<ocid:2>",
  "intervalInSeconds": 60,
  "isEnabled": false,
  "lastSplatToken": "<redacted>",
  "lifecycleState": "ACTIVE",
  "network": "ip4",
  "port": 443,
  "protocol": "TCP",
  "resultsUrl": "20180501/pingProbeResults/<ocid:2>",
  "systemTags": {},
  "targets": [
    "example.com"
  ],
  "timeCreated": "2026-09-01T17:48:20.797083Z",
  "timeoutInSeconds": 10,
  "vantagePointNames": [
    "azr-iad1",
    "goo-cbf",
    "aws-pdx"
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[healthcheckssdk.UpdatePingMonitorDetails](t, `{
  "displayName": "osok-mock-ping-monitor-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "intervalInSeconds": 30
}`)
	updatedState := ocimock.MustOCIResponseFixture[healthcheckssdk.PingMonitor](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T17:48:20.639Z"
    }
  },
  "displayName": "osok-mock-ping-monitor-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hash": null,
  "homeRegion": "us-ashburn-1",
  "id": "<ocid:2>",
  "intervalInSeconds": 30,
  "isEnabled": false,
  "lastSplatToken": "<redacted>",
  "lifecycleState": "ACTIVE",
  "network": "ip4",
  "port": 443,
  "protocol": "TCP",
  "resultsUrl": "20180501/pingProbeResults/<ocid:2>",
  "systemTags": {},
  "targets": [
    "example.com"
  ],
  "timeCreated": "2026-09-01T17:48:20.797083Z",
  "timeoutInSeconds": 10,
  "vantagePointNames": [
    "azr-iad1",
    "goo-cbf",
    "aws-pdx"
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		healthcheckssdk.PingMonitor,
		healthcheckssdk.CreatePingMonitorDetails,
		healthcheckssdk.UpdatePingMonitorDetails,
	]{
		CollectionPath:    "/20180501/pingMonitors",
		ItemPath:          "/20180501/pingMonitors/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      200,
		NotFoundCode:      "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ healthcheckssdk.CreatePingMonitorDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ healthcheckssdk.PingMonitor) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20180501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close PingMonitor OCI mock: %v", err)
		}
	})
	sdkClient := healthcheckssdk.HealthChecksClient{BaseClient: session.BaseClient()}
	client := newTestPingMonitorClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*healthchecksv1beta1.PingMonitor]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *healthchecksv1beta1.PingMonitor) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IntervalInSeconds, current.Spec.IntervalInSeconds) ||
				!reflect.DeepEqual(current.Status.Port, current.Spec.Port) ||
				!reflect.DeepEqual(current.Status.Protocol, current.Spec.Protocol) ||
				!reflect.DeepEqual(current.Status.Targets, current.Spec.Targets) ||
				!reflect.DeepEqual(current.Status.TimeoutInSeconds, current.Spec.TimeoutInSeconds) {
				return fmt.Errorf("created PingMonitor status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *healthchecksv1beta1.PingMonitor) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *healthchecksv1beta1.PingMonitor) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IntervalInSeconds, current.Spec.IntervalInSeconds) {
				return fmt.Errorf("updated PingMonitor status = %+v", current.Status)
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
