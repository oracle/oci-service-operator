/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fusionenvironment

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/fusionapps"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/fusionapps/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationFusionEnvironmentCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.FusionEnvironment](t, `
{
  "metadata": {"name": "mock-fusionenvironment", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "createFusionEnvironmentAdminUserDetails": {
    "emailAddress": "mock-emailaddress",
    "firstName": "mock-firstname",
    "lastName": "mock-lastname",
    "username": "mock-username"
  },
  "displayName": "mock-displayname-initial",
  "dnsPrefix": "mock-dnsprefix-updated",
  "fusionEnvironmentFamilyId": "<ocid:required>",
  "fusionEnvironmentType": "PRODUCTION"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-fusionenvironment")
	resource.Status = apiv1beta1.FusionEnvironmentStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateFusionEnvironmentDetails](t, `{
  "compartmentId": "<ocid:required>",
  "createFusionEnvironmentAdminUserDetails": {
    "emailAddress": "mock-emailaddress",
    "firstName": "mock-firstname",
    "lastName": "mock-lastname",
    "username": "mock-username"
  },
  "displayName": "mock-displayname-initial",
  "dnsPrefix": "mock-dnsprefix-updated",
  "fusionEnvironmentFamilyId": "<ocid:required>",
  "fusionEnvironmentType": "PRODUCTION"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateFusionEnvironmentDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.FusionEnvironment](t, `{
  "compartmentId": "<ocid:required>",
  "createFusionEnvironmentAdminUserDetails": {
    "emailAddress": "mock-emailaddress",
    "firstName": "mock-firstname",
    "lastName": "mock-lastname",
    "username": "mock-username"
  },
  "displayName": "mock-displayname-initial",
  "dnsPrefix": "mock-dnsprefix-updated",
  "fusionEnvironmentFamilyId": "<ocid:required>",
  "fusionEnvironmentType": "PRODUCTION",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.FusionEnvironment](t, `{
  "compartmentId": "<ocid:required>",
  "createFusionEnvironmentAdminUserDetails": {
    "emailAddress": "mock-emailaddress",
    "firstName": "mock-firstname",
    "lastName": "mock-lastname",
    "username": "mock-username"
  },
  "displayName": "mock-displayname-updated",
  "dnsPrefix": "mock-dnsprefix-updated",
  "fusionEnvironmentFamilyId": "<ocid:required>",
  "fusionEnvironmentType": "PRODUCTION",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEFUSIONENVIRONMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "FusionEnvironment", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEFUSIONENVIRONMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "FusionEnvironment", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEFUSIONENVIRONMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "FusionEnvironment", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.FusionEnvironment, sdksvc.CreateFusionEnvironmentDetails, sdksvc.UpdateFusionEnvironmentDetails]{
		CollectionPath: "/20211201/fusionEnvironments", ItemPath: "/20211201/fusionEnvironments/<ocid:1>",
		CreatePath: "/20211201/fusionEnvironments", CreateMethod: http.MethodPost,
		UpdatePath: "/20211201/fusionEnvironments/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20211201/fusionEnvironments/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateFusionEnvironmentDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateFusionEnvironmentDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateFusionEnvironmentDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20211201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20211201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20211201/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://fusionapps.us-ashburn-1.oci.oraclecloud.com", BasePath: "20211201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.FusionApplicationsClient{BaseClient: session.BaseClient()}
	manager := &FusionEnvironmentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFusionEnvironmentRuntimeHooks(manager, sdkClient)
	client := wrapFusionEnvironmentGeneratedClient(hooks, defaultFusionEnvironmentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.FusionEnvironment](buildFusionEnvironmentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.FusionEnvironment]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.FusionEnvironment) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created FusionEnvironment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.FusionEnvironment) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.FusionEnvironment) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated FusionEnvironment status = %+v", current.Status)
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
