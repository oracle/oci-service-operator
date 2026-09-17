/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package loganalyticsentitytype

import (
	"context"
	"fmt"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLogAnalyticsEntityTypeCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loganalyticsv1beta1.LogAnalyticsEntityType{}
	ocimock.InitializeResource(resource, "mock-loganalyticsentitytype")
	resource.Spec = ocimock.MustJSONFixture[loganalyticsv1beta1.LogAnalyticsEntityTypeSpec](t, `{
  "category": "osok-mock",
  "name": "\u003cbinding:entity-type-name\u003e",
  "properties": [
    {
      "description": "recorded hostname",
      "name": "host"
    }
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "category": "osok-mock-updated"
}`)
	createRequest := ocimock.MustJSONFixture[loganalyticssdk.CreateLogAnalyticsEntityTypeDetails](t, `{
  "category": "osok-mock",
  "name": "\u003cbinding:entity-type-name\u003e",
  "properties": [
    {
      "description": "recorded hostname",
      "name": "host"
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntityType](t, `{
  "category": "osok-mock",
  "cloudType": "NON_CLOUD",
  "internalName": "custom_osokmockentitytype1788326489",
  "lifecycleState": "ACTIVE",
  "managementAgentEligibilityStatus": "ELIGIBLE",
  "name": "<binding:entity-type-name>",
  "properties": [
    {
      "description": "recorded hostname",
      "name": "host"
    }
  ],
  "timeCreated": "2026-09-02T05:21:24.538Z",
  "timeUpdated": "2026-09-02T05:21:24.538Z"
}`)
	updateRequest := ocimock.MustJSONFixture[loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails](t, `{
  "category": "osok-mock-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntityType](t, `{
  "category": "osok-mock-updated",
  "cloudType": "NON_CLOUD",
  "internalName": "custom_osokmockentitytype1788326489",
  "lifecycleState": "ACTIVE",
  "managementAgentEligibilityStatus": "ELIGIBLE",
  "name": "<binding:entity-type-name>",
  "properties": [
    {
      "description": "recorded hostname",
      "name": "host"
    }
  ],
  "timeCreated": "2026-09-02T05:21:24.538Z",
  "timeUpdated": "2026-09-02T05:21:24.993Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntityType](t, `{
  "category": "osok-mock-updated",
  "cloudType": "NON_CLOUD",
  "internalName": "custom_osokmockentitytype1788326489",
  "lifecycleState": "DELETED",
  "managementAgentEligibilityStatus": "ELIGIBLE",
  "name": "<binding:entity-type-name>",
  "properties": [
    {
      "description": "recorded hostname",
      "name": "host"
    }
  ],
  "timeCreated": "2026-09-02T05:21:24.538Z",
  "timeUpdated": "2026-09-02T05:21:25.310Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loganalyticssdk.LogAnalyticsEntityType,
		loganalyticssdk.CreateLogAnalyticsEntityTypeDetails,
		loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails,
	]{
		CollectionPath:    "/20200601/namespaces/<binding:loganalytics-namespace>/logAnalyticsEntityTypes",
		ItemPath:          "/20200601/namespaces/<binding:loganalytics-namespace>/logAnalyticsEntityTypes/<binding:entity-type-name>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ loganalyticssdk.CreateLogAnalyticsEntityTypeDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loganalyticssdk.LogAnalyticsEntityType) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close LogAnalyticsEntityType OCI mock: %v", err)
		}
	})
	sdkClient := loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()}
	client := newLogAnalyticsEntityTypeServiceClientWithOCIClientAndNamespaceGetter(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient, mockEntityTypeNamespaceGetter{namespace: "<binding:loganalytics-namespace>"})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.LogAnalyticsEntityType]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loganalyticsv1beta1.LogAnalyticsEntityType) error {
			if current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Category, current.Spec.Category) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Properties, current.Spec.Properties) {
				return fmt.Errorf("created LogAnalyticsEntityType status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.LogAnalyticsEntityType) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.LogAnalyticsEntityType) error {
			if current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Category, current.Spec.Category) {
				return fmt.Errorf("updated LogAnalyticsEntityType status = %+v", current.Status)
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
