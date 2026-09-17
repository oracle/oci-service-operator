/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package baselineablemetric

import (
	"context"
	"fmt"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationBaselineableMetricLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	ocimock.InitializeResource(resource, "mock-baselineablemetric")
	resource.Spec = ocimock.MustJSONFixture[stackmonitoringv1beta1.BaselineableMetricSpec](t, `{
  "column": "CpuUtilization",
  "compartmentId": "\u003cocid:1\u003e",
  "name": "cpu_utilization",
  "namespace": "oci_computeagent",
  "resourceGroup": "instance",
  "resourceType": "compute"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "column": "CpuUtilization",
  "compartmentId": "\u003cocid:1\u003e",
  "id": "\u003cocid:2\u003e",
  "isOutOfBox": false,
  "lifecycleState": "ACTIVE",
  "name": "cpu_utilization-updated",
  "namespace": "oci_computeagent",
  "resourceGroup": "instance",
  "resourceType": "compute"
}`)
	createRequest := ocimock.MustJSONFixture[stackmonitoringsdk.CreateBaselineableMetricDetails](t, `{
  "column": "CpuUtilization",
  "compartmentId": "\u003cocid:1\u003e",
  "name": "cpu_utilization",
  "namespace": "oci_computeagent",
  "resourceGroup": "instance",
  "resourceType": "compute"
}`)
	createdState := ocimock.MustOCIResponseFixture[stackmonitoringsdk.BaselineableMetric](t, `{
  "column": "CpuUtilization",
  "compartmentId": "\u003cocid:1\u003e",
  "id": "\u003cocid:2\u003e",
  "isOutOfBox": false,
  "lifecycleState": "ACTIVE",
  "name": "cpu_utilization",
  "namespace": "oci_computeagent",
  "resourceGroup": "instance",
  "resourceType": "compute"
}`)
	updateRequest := ocimock.MustJSONFixture[stackmonitoringsdk.UpdateBaselineableMetricDetails](t, `{
  "column": "CpuUtilization",
  "compartmentId": "\u003cocid:1\u003e",
  "id": "\u003cocid:2\u003e",
  "isOutOfBox": false,
  "lifecycleState": "ACTIVE",
  "name": "cpu_utilization-updated",
  "namespace": "oci_computeagent",
  "resourceGroup": "instance",
  "resourceType": "compute"
}`)
	updatedState := ocimock.MustOCIResponseFixture[stackmonitoringsdk.BaselineableMetric](t, `{
  "column": "CpuUtilization",
  "compartmentId": "\u003cocid:1\u003e",
  "id": "\u003cocid:2\u003e",
  "isOutOfBox": false,
  "lifecycleState": "ACTIVE",
  "name": "cpu_utilization-updated",
  "namespace": "oci_computeagent",
  "resourceGroup": "instance",
  "resourceType": "compute"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		stackmonitoringsdk.BaselineableMetric,
		stackmonitoringsdk.CreateBaselineableMetricDetails,
		stackmonitoringsdk.UpdateBaselineableMetricDetails,
	]{
		CollectionPath:    "/20210330/baselineableMetrics",
		ItemPath:          "/20210330/baselineableMetrics/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ stackmonitoringsdk.CreateBaselineableMetricDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ stackmonitoringsdk.BaselineableMetric) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210330", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close BaselineableMetric OCI mock: %v", err)
		}
	})
	sdkClient := stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()}
	client := newBaselineableMetricServiceClientWithOCIClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.BaselineableMetric]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.BaselineableMetric) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Column, current.Spec.Column) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Namespace, current.Spec.Namespace) ||
				!reflect.DeepEqual(current.Status.ResourceGroup, current.Spec.ResourceGroup) ||
				!reflect.DeepEqual(current.Status.ResourceType, current.Spec.ResourceType) {
				return fmt.Errorf("created BaselineableMetric status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.BaselineableMetric) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *stackmonitoringv1beta1.BaselineableMetric) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Column, current.Spec.Column) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Id, current.Spec.Id) ||
				!reflect.DeepEqual(current.Status.IsOutOfBox, current.Spec.IsOutOfBox) ||
				!reflect.DeepEqual(current.Status.LifecycleState, current.Spec.LifecycleState) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Namespace, current.Spec.Namespace) ||
				!reflect.DeepEqual(current.Status.ResourceGroup, current.Spec.ResourceGroup) ||
				!reflect.DeepEqual(current.Status.ResourceType, current.Spec.ResourceType) {
				return fmt.Errorf("updated BaselineableMetric status = %+v", current.Status)
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
