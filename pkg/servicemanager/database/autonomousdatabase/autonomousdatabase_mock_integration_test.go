/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package autonomousdatabase

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockAutonomousDatabaseID = "ocid1.autonomousdatabase.oc1..mock"

// Contract evidence: vendored SDK request/response types and the imported Autonomous Database formal contract.
func TestMockIntegrationAutonomousDatabaseLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &databasev1beta1.AutonomousDatabase{Spec: databasev1beta1.AutonomousDatabaseSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DbName: "MOCKADB", DisplayName: "mock-adb",
		CpuCoreCount: 1, DataStorageSizeInTBs: 1, DbWorkload: "OLTP", LicenseModel: "LICENSE_INCLUDED", Source: "NONE",
	}}
	ocimock.InitializeResource(resource, "mock-autonomousdatabase")
	responder, err := newAutonomousDatabaseMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://database.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &AutonomousDatabaseServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAutonomousDatabaseRuntimeHooks(manager, databasesdk.DatabaseClient{BaseClient: session.BaseClient()})
	client := wrapAutonomousDatabaseGeneratedClient(hooks, defaultAutonomousDatabaseServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*databasev1beta1.AutonomousDatabase](buildAutonomousDatabaseGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*databasev1beta1.AutonomousDatabase]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *databasev1beta1.AutonomousDatabase) error {
			if current.Status.Id != mockAutonomousDatabaseID || current.Status.LifecycleState != string(databasesdk.AutonomousDatabaseLifecycleStateAvailable) {
				return fmt.Errorf("created AutonomousDatabase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *databasev1beta1.AutonomousDatabase) {
			current.Spec.FreeformTags = map[string]string{"phase": "updated"}
		},
		ValidateUpdated: func(current *databasev1beta1.AutonomousDatabase) error {
			if current.Status.FreeformTags["phase"] != "updated" {
				return fmt.Errorf("updated AutonomousDatabase status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func newAutonomousDatabaseMockResponder(resource *databasev1beta1.AutonomousDatabase) (*ocimock.CRUDResponder[databasesdk.AutonomousDatabase], error) {
	deleteReads := 0
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[databasesdk.AutonomousDatabase]{
		CollectionPath: "/20160918/autonomousDatabases",
		ItemPath:       "/20160918/autonomousDatabases/" + mockAutonomousDatabaseID,
		ExpectedOperations: []ocimock.Operation{
			ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete,
		},
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (databasesdk.AutonomousDatabase, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero databasesdk.AutonomousDatabase
				return zero, ocimock.Response{}, err
			}
			var details map[string]any
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return databasesdk.AutonomousDatabase{}, ocimock.Response{}, err
			}
			if details["compartmentId"] != resource.Spec.CompartmentId || details["dbName"] != resource.Spec.DbName || details["source"] != resource.Spec.Source {
				return databasesdk.AutonomousDatabase{}, ocimock.Response{}, fmt.Errorf("unexpected AutonomousDatabase create details: %+v", details)
			}
			isDedicated := false
			state := databasesdk.AutonomousDatabase{
				Id: &[]string{mockAutonomousDatabaseID}[0], CompartmentId: &resource.Spec.CompartmentId,
				DbName: &resource.Spec.DbName, DisplayName: &resource.Spec.DisplayName,
				DataStorageSizeInTBs: &resource.Spec.DataStorageSizeInTBs, CpuCoreCount: &resource.Spec.CpuCoreCount,
				IsDedicated: &isDedicated, LifecycleState: databasesdk.AutonomousDatabaseLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state databasesdk.AutonomousDatabase) (databasesdk.AutonomousDatabase, ocimock.Response, error) {
			if state.LifecycleState == databasesdk.AutonomousDatabaseLifecycleStateProvisioning || state.LifecycleState == databasesdk.AutonomousDatabaseLifecycleStateUpdating {
				state.LifecycleState = databasesdk.AutonomousDatabaseLifecycleStateAvailable
			} else if state.LifecycleState == databasesdk.AutonomousDatabaseLifecycleStateTerminating {
				deleteReads++
				if deleteReads > 1 {
					state.LifecycleState = databasesdk.AutonomousDatabaseLifecycleStateUnavailable
				}
			} else if state.LifecycleState == databasesdk.AutonomousDatabaseLifecycleStateUnavailable {
				state.LifecycleState = databasesdk.AutonomousDatabaseLifecycleStateTerminated
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state databasesdk.AutonomousDatabase) (databasesdk.AutonomousDatabase, ocimock.Response, error) {
			var details databasesdk.UpdateAutonomousDatabaseDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.FreeformTags["phase"] != "updated" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected AutonomousDatabase update details: %+v", details)
			}
			if details.DisplayName != nil {
				state.DisplayName = details.DisplayName
			}
			state.FreeformTags, state.LifecycleState = details.FreeformTags, databasesdk.AutonomousDatabaseLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state databasesdk.AutonomousDatabase) (databasesdk.AutonomousDatabase, ocimock.Response, error) {
			state.LifecycleState = databasesdk.AutonomousDatabaseLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
