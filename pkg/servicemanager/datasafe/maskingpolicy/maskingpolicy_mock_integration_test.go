/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package maskingpolicy

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
func TestMockIntegrationMaskingPolicyLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.MaskingPolicy{}
	ocimock.InitializeResource(resource, "mock-maskingpolicy")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.MaskingPolicySpec](t, `{
  "columnSource": {
    "columnSource": "SENSITIVE_DATA_MODEL",
    "sensitiveDataModelId": "\u003cocid:1\u003e"
  },
  "compartmentId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-masking-policy",
  "isDropTempTablesEnabled": false,
  "isRedoLoggingEnabled": false,
  "isRefreshStatsEnabled": false
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "columnSource": {
    "columnSource": "SENSITIVE_DATA_MODEL",
    "sensitiveDataModelId": "\u003cocid:1\u003e"
  },
  "description": "mock-updated",
  "displayName": "osok-mock-masking-policy",
  "isDropTempTablesEnabled": false,
  "isRedoLoggingEnabled": false,
  "isRefreshStatsEnabled": false
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateMaskingPolicyDetails](t, `{
  "columnSource": {
    "columnSource": "SENSITIVE_DATA_MODEL",
    "sensitiveDataModelId": "\u003cocid:1\u003e"
  },
  "compartmentId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-masking-policy",
  "isDropTempTablesEnabled": false,
  "isRedoLoggingEnabled": false,
  "isRefreshStatsEnabled": false
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.MaskingPolicy](t, `{
  "columnSource": {
    "columnSource": "SENSITIVE_DATA_MODEL",
    "sensitiveDataModelId": "\u003cocid:1\u003e"
  },
  "compartmentId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-masking-policy",
  "id": "\u003cocid:3\u003e",
  "isDropTempTablesEnabled": false,
  "isRedoLoggingEnabled": false,
  "isRefreshStatsEnabled": false,
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateMaskingPolicyDetails](t, `{
  "columnSource": {
    "columnSource": "SENSITIVE_DATA_MODEL",
    "sensitiveDataModelId": "\u003cocid:1\u003e"
  },
  "description": "mock-updated",
  "displayName": "osok-mock-masking-policy",
  "isDropTempTablesEnabled": false,
  "isRedoLoggingEnabled": false,
  "isRefreshStatsEnabled": false
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.MaskingPolicy](t, `{
  "columnSource": {
    "columnSource": "SENSITIVE_DATA_MODEL",
    "sensitiveDataModelId": "\u003cocid:1\u003e"
  },
  "compartmentId": "\u003cocid:2\u003e",
  "description": "mock-updated",
  "displayName": "osok-mock-masking-policy",
  "id": "\u003cocid:3\u003e",
  "isDropTempTablesEnabled": false,
  "isRedoLoggingEnabled": false,
  "isRefreshStatsEnabled": false,
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.MaskingPolicy,
		datasafesdk.CreateMaskingPolicyDetails,
		datasafesdk.UpdateMaskingPolicyDetails,
	]{
		CollectionPath:    "/20181201/maskingPolicies",
		ItemPath:          "/20181201/maskingPolicies/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateMaskingPolicyDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.MaskingPolicy) error {
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
			t.Errorf("close MaskingPolicy OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &MaskingPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newMaskingPolicyRuntimeHooks(manager, sdkClient)
	client := wrapMaskingPolicyGeneratedClient(hooks, defaultMaskingPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.MaskingPolicy](buildMaskingPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.MaskingPolicy]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.MaskingPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ColumnSource, current.Spec.ColumnSource) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.IsDropTempTablesEnabled, current.Spec.IsDropTempTablesEnabled) ||
				!reflect.DeepEqual(current.Status.IsRedoLoggingEnabled, current.Spec.IsRedoLoggingEnabled) ||
				!reflect.DeepEqual(current.Status.IsRefreshStatsEnabled, current.Spec.IsRefreshStatsEnabled) {
				return fmt.Errorf("created MaskingPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.MaskingPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.MaskingPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ColumnSource, current.Spec.ColumnSource) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.IsDropTempTablesEnabled, current.Spec.IsDropTempTablesEnabled) ||
				!reflect.DeepEqual(current.Status.IsRedoLoggingEnabled, current.Spec.IsRedoLoggingEnabled) ||
				!reflect.DeepEqual(current.Status.IsRefreshStatsEnabled, current.Spec.IsRefreshStatsEnabled) {
				return fmt.Errorf("updated MaskingPolicy status = %+v", current.Status)
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
