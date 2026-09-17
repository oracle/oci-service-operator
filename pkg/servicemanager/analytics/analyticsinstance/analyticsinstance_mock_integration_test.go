/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package analyticsinstance

import (
	"context"
	"fmt"
	analyticssdk "github.com/oracle/oci-go-sdk/v65/analytics"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationAnalyticsInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &analyticsv1beta1.AnalyticsInstance{Spec: analyticsv1beta1.AnalyticsInstanceSpec{
		Name: "osok-mock-analytics", CompartmentId: "ocid1.compartment.oc1..mock",
		FeatureSet:  "ENTERPRISE_ANALYTICS",
		Capacity:    analyticsv1beta1.AnalyticsInstanceCapacity{CapacityType: "OLPU_COUNT", CapacityValue: 2},
		LicenseType: "LICENSE_INCLUDED", Description: "OSOK synthetic analytics instance",
	}}
	ocimock.InitializeResource(resource, "mock-analyticsinstance")
	resource.Spec = ocimock.MustJSONFixture[analyticsv1beta1.AnalyticsInstanceSpec](t, `{
  "capacity": {
    "capacityType": "OLPU_COUNT",
    "capacityValue": 2
  },
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK synthetic analytics instance",
  "featureSet": "ENTERPRISE_ANALYTICS",
  "licenseType": "LICENSE_INCLUDED",
  "name": "osok-mock-analytics"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK synthetic analytics instance updated"
}`)
	createRequest := ocimock.MustJSONFixture[analyticssdk.CreateAnalyticsInstanceDetails](t, `{
  "capacity": {
    "capacityType": "OLPU_COUNT",
    "capacityValue": 2
  },
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK synthetic analytics instance",
  "featureSet": "ENTERPRISE_ANALYTICS",
  "licenseType": "LICENSE_INCLUDED",
  "name": "osok-mock-analytics"
}`)
	createdState := ocimock.MustOCIResponseFixture[analyticssdk.AnalyticsInstance](t, `{
  "id": "<ocid:2>",
  "name": "osok-mock-analytics",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "featureSet": "ENTERPRISE_ANALYTICS",
  "capacity": {
    "capacityType": "OLPU_COUNT",
    "capacityValue": 2
  },
  "licenseType": "LICENSE_INCLUDED",
  "description": "OSOK synthetic analytics instance"
}`)
	updateRequest := ocimock.MustJSONFixture[analyticssdk.UpdateAnalyticsInstanceDetails](t, `{
  "description": "OSOK synthetic analytics instance updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[analyticssdk.AnalyticsInstance](t, `{
  "id": "<ocid:2>",
  "name": "osok-mock-analytics",
  "compartmentId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "featureSet": "ENTERPRISE_ANALYTICS",
  "capacity": {
    "capacityType": "OLPU_COUNT",
    "capacityValue": 2
  },
  "licenseType": "LICENSE_INCLUDED",
  "description": "OSOK synthetic analytics instance updated"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		analyticssdk.AnalyticsInstance,
		analyticssdk.CreateAnalyticsInstanceDetails,
		analyticssdk.UpdateAnalyticsInstanceDetails,
	]{
		CollectionPath:     "/20190331/analyticsInstances",
		ItemPath:           "/20190331/analyticsInstances/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:          ocimock.ListShapeArray,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		UpdatedReadStates:  ocimock.LifecycleStateSequence(t, updatedState, "UPDATING"),
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       202,
		UpdateStatus:       200,
		DeleteStatus:       202,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ analyticssdk.CreateAnalyticsInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ analyticssdk.AnalyticsInstance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20190331", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close AnalyticsInstance OCI mock: %v", err)
		}
	})
	sdkClient := analyticssdk.AnalyticsClient{BaseClient: session.BaseClient()}
	manager := &AnalyticsInstanceServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newAnalyticsInstanceRuntimeHooks(manager, sdkClient)
	client := wrapAnalyticsInstanceGeneratedClient(hooks, defaultAnalyticsInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*analyticsv1beta1.AnalyticsInstance](
			buildAnalyticsInstanceGeneratedRuntimeConfig(manager, hooks),
		),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*analyticsv1beta1.AnalyticsInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *analyticsv1beta1.AnalyticsInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Capacity, current.Spec.Capacity) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FeatureSet, current.Spec.FeatureSet) ||
				!reflect.DeepEqual(current.Status.LicenseType, current.Spec.LicenseType) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created AnalyticsInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *analyticsv1beta1.AnalyticsInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *analyticsv1beta1.AnalyticsInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated AnalyticsInstance status = %+v", current.Status)
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
