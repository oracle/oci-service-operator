/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package userassessment

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/datasafe"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationUserAssessmentCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.UserAssessment](t, `
{
  "metadata": {"name": "mock-userassessment", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "targetId": "",
  "targetType": "TARGET_DATABASE"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-userassessment")
	resource.Status = apiv1beta1.UserAssessmentStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateUserAssessmentDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "targetId": "",
  "targetType": "TARGET_DATABASE"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateUserAssessmentDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.UserAssessment](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "SUCCEEDED",
  "state": "SUCCEEDED",
  "status": "SUCCEEDED",
  "targetId": "",
  "targetType": "TARGET_DATABASE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.UserAssessment](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "SUCCEEDED",
  "state": "SUCCEEDED",
  "status": "SUCCEEDED",
  "targetId": "",
  "targetType": "TARGET_DATABASE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEUSERASSESSMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "UserAssessment", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEUSERASSESSMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "UserAssessment", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEUSERASSESSMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "UserAssessment", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.UserAssessment, sdksvc.CreateUserAssessmentDetails, sdksvc.UpdateUserAssessmentDetails]{
		CollectionPath: "/20181201/userAssessments", ItemPath: "/20181201/userAssessments/<ocid:2>",
		CreatePath: "/20181201/userAssessments", CreateMethod: http.MethodPost,
		UpdatePath: "/20181201/userAssessments/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20181201/userAssessments/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateUserAssessmentDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateUserAssessmentDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateUserAssessmentDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &UserAssessmentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newUserAssessmentRuntimeHooks(manager, sdkClient)
	client := wrapUserAssessmentGeneratedClient(hooks, defaultUserAssessmentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.UserAssessment](buildUserAssessmentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.UserAssessment]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.UserAssessment) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "SUCCEEDED" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created UserAssessment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.UserAssessment) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.UserAssessment) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated UserAssessment status = %+v", current.Status)
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
