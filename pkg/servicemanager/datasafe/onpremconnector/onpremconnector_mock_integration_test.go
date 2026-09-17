/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package onpremconnector

import (
	"context"
	"fmt"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOnPremConnectorLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.OnPremConnector{}
	ocimock.InitializeResource(resource, "mock-onpremconnector")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.OnPremConnectorSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-onprem-connector"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "mock-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateOnPremConnectorDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-onprem-connector"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.OnPremConnector](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-onprem-connector",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateOnPremConnectorDetails](t, `{
  "description": "mock-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.OnPremConnector](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "mock-updated",
  "displayName": "osok-mock-onprem-connector",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.OnPremConnector,
		datasafesdk.CreateOnPremConnectorDetails,
		datasafesdk.UpdateOnPremConnectorDetails,
	]{
		CollectionPath:    "/20181201/onPremConnectors",
		ItemPath:          "/20181201/onPremConnectors/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateOnPremConnectorDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.OnPremConnector) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OnPremConnector OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &OnPremConnectorServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newOnPremConnectorRuntimeHooks(manager, sdkClient)
	client := wrapOnPremConnectorGeneratedClient(hooks, defaultOnPremConnectorServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.OnPremConnector](buildOnPremConnectorGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.OnPremConnector]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.OnPremConnector) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("created OnPremConnector status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.OnPremConnector) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.OnPremConnector) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated OnPremConnector status = %+v", current.Status)
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
