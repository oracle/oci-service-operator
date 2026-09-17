/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sqlcollection

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
func TestMockIntegrationSqlCollectionLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.SqlCollection{}
	ocimock.InitializeResource(resource, "mock-sqlcollection")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.SqlCollectionSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dbUserName": "APPUSER",
  "displayName": "osok-mock-sql-collection",
  "sqlLevel": "USER_ISSUED_SQL",
  "status": "DISABLED",
  "targetId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "mock-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSqlCollectionDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dbUserName": "APPUSER",
  "displayName": "osok-mock-sql-collection",
  "sqlLevel": "USER_ISSUED_SQL",
  "status": "DISABLED",
  "targetId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SqlCollection](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dbUserName": "APPUSER",
  "displayName": "osok-mock-sql-collection",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "INACTIVE",
  "sqlLevel": "USER_ISSUED_SQL",
  "status": "DISABLED",
  "targetId": "\u003cocid:2\u003e"
}`)
	createdReadStates := []datasafesdk.SqlCollection{
		ocimock.MustOCIResponseFixture[datasafesdk.SqlCollection](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dbUserName": "APPUSER",
  "displayName": "osok-mock-sql-collection",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "INACTIVE",
  "sqlLevel": "USER_ISSUED_SQL",
  "status": "DISABLED",
  "targetId": "\u003cocid:2\u003e"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSqlCollectionDetails](t, `{
  "description": "mock-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SqlCollection](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dbUserName": "APPUSER",
  "description": "mock-updated",
  "displayName": "osok-mock-sql-collection",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "INACTIVE",
  "sqlLevel": "USER_ISSUED_SQL",
  "status": "DISABLED",
  "targetId": "\u003cocid:2\u003e"
}`)
	updatedReadStates := []datasafesdk.SqlCollection{
		ocimock.MustOCIResponseFixture[datasafesdk.SqlCollection](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dbUserName": "APPUSER",
  "description": "mock-updated",
  "displayName": "osok-mock-sql-collection",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "INACTIVE",
  "sqlLevel": "USER_ISSUED_SQL",
  "status": "DISABLED",
  "targetId": "\u003cocid:2\u003e"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.SqlCollection,
		datasafesdk.CreateSqlCollectionDetails,
		datasafesdk.UpdateSqlCollectionDetails,
	]{
		CollectionPath:     "/20181201/sqlCollections",
		ItemPath:           "/20181201/sqlCollections/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
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
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSqlCollectionDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.SqlCollection) error {
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
			t.Errorf("close SqlCollection OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &SqlCollectionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newSqlCollectionRuntimeHooks(manager, sdkClient)
	client := wrapSqlCollectionGeneratedClient(hooks, defaultSqlCollectionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SqlCollection](buildSqlCollectionGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SqlCollection]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SqlCollection) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "INACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DbUserName, current.Spec.DbUserName) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.SqlLevel, current.Spec.SqlLevel) ||
				!reflect.DeepEqual(current.Status.Status, current.Spec.Status) ||
				!reflect.DeepEqual(current.Status.TargetId, current.Spec.TargetId) {
				return fmt.Errorf("created SqlCollection status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SqlCollection) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.SqlCollection) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "INACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated SqlCollection status = %+v", current.Status)
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
