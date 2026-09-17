/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package config

import (
	"context"
	"fmt"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationConfigLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &stackmonitoringv1beta1.Config{}
	ocimock.InitializeResource(resource, "mock-config")
	resource.Spec = ocimock.MustJSONFixture[stackmonitoringv1beta1.ConfigSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "configType": "AUTO_PROMOTE",
  "displayName": "osok-mock-stack-config",
  "isEnabled": true,
  "resourceType": "HOST"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "configType": "AUTO_PROMOTE",
  "displayName": "osok-mock-stack-config-updated",
  "isEnabled": true
}`)
	createRequest := ocimock.MustJSONFixture[stackmonitoringsdk.CreateAutoPromoteConfigDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-stack-config",
  "isEnabled": true,
  "resourceType": "HOST"
}`)
	createdState := ocimock.MustOCIResponseFixture[stackmonitoringsdk.AutoPromoteConfigDetails](t, `{
  "additionalConfigurations": {},
  "compartmentId": "\u003cocid:1\u003e",
  "configType": "AUTO_PROMOTE",
  "displayName": "osok-mock-stack-config",
  "id": "\u003cocid:2\u003e",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "resourceType": "HOST"
}`)
	createdReadStates := []stackmonitoringsdk.AutoPromoteConfigDetails{
		ocimock.MustOCIResponseFixture[stackmonitoringsdk.AutoPromoteConfigDetails](t, `{
  "additionalConfigurations": {},
  "compartmentId": "\u003cocid:1\u003e",
  "configType": "AUTO_PROMOTE",
  "displayName": "osok-mock-stack-config",
  "id": "\u003cocid:2\u003e",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "resourceType": "HOST"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[stackmonitoringsdk.UpdateAutoPromoteConfigDetails](t, `{
  "displayName": "osok-mock-stack-config-updated",
  "isEnabled": true
}`)
	updatedState := ocimock.MustOCIResponseFixture[stackmonitoringsdk.AutoPromoteConfigDetails](t, `{
  "additionalConfigurations": {},
  "compartmentId": "\u003cocid:1\u003e",
  "configType": "AUTO_PROMOTE",
  "displayName": "osok-mock-stack-config-updated",
  "id": "\u003cocid:2\u003e",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "resourceType": "HOST"
}`)
	updatedReadStates := []stackmonitoringsdk.AutoPromoteConfigDetails{
		ocimock.MustOCIResponseFixture[stackmonitoringsdk.AutoPromoteConfigDetails](t, `{
  "additionalConfigurations": {},
  "compartmentId": "\u003cocid:1\u003e",
  "configType": "AUTO_PROMOTE",
  "displayName": "osok-mock-stack-config-updated",
  "id": "\u003cocid:2\u003e",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "resourceType": "HOST"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		stackmonitoringsdk.AutoPromoteConfigDetails,
		stackmonitoringsdk.CreateAutoPromoteConfigDetails,
		stackmonitoringsdk.UpdateAutoPromoteConfigDetails,
	]{
		CollectionPath:     "/20210330/configs",
		ItemPath:           "/20210330/configs/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "configType", "AUTO_PROMOTE", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "configType", "AUTO_PROMOTE", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ stackmonitoringsdk.AutoPromoteConfigDetails) error {
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
			t.Errorf("close Config OCI mock: %v", err)
		}
	})
	sdkClient := stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()}
	manager := &ConfigServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newConfigRuntimeHooks(manager, sdkClient)
	client := wrapConfigGeneratedClient(hooks, defaultConfigServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.Config](buildConfigGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.Config]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.Config) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.ConfigType, current.Spec.ConfigType) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.IsEnabled, current.Spec.IsEnabled) ||
				!reflect.DeepEqual(current.Status.ResourceType, current.Spec.ResourceType) {
				return fmt.Errorf("created Config status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.Config) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *stackmonitoringv1beta1.Config) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ConfigType, current.Spec.ConfigType) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.IsEnabled, current.Spec.IsEnabled) {
				return fmt.Errorf("updated Config status = %+v", current.Status)
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
