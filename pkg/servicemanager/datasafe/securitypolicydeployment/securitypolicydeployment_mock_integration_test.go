/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securitypolicydeployment

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
func TestMockIntegrationSecurityPolicyDeploymentCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.SecurityPolicyDeployment](t, `
{
  "metadata": {"name": "mock-securitypolicydeployment", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "securityPolicyId": "<ocid:2>",
  "targetId": "<ocid:3>",
  "targetType": "TARGET_DATABASE"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-securitypolicydeployment")
	resource.Status = apiv1beta1.SecurityPolicyDeploymentStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateSecurityPolicyDeploymentDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "securityPolicyId": "<ocid:2>",
  "targetId": "<ocid:3>",
  "targetType": "TARGET_DATABASE"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateSecurityPolicyDeploymentDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.SecurityPolicyDeployment](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:4>",
  "key": "<ocid:4>",
  "lifecycleState": "DEPLOYED",
  "securityPolicyId": "<ocid:2>",
  "state": "DEPLOYED",
  "status": "DEPLOYED",
  "targetId": "<ocid:3>",
  "targetType": "TARGET_DATABASE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.SecurityPolicyDeployment](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:4>",
  "key": "<ocid:4>",
  "lifecycleState": "DEPLOYED",
  "securityPolicyId": "<ocid:2>",
  "state": "DEPLOYED",
  "status": "DEPLOYED",
  "targetId": "<ocid:3>",
  "targetType": "TARGET_DATABASE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATESECURITYPOLICYDEPLOYMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "SecurityPolicyDeployment", "identifier": "<ocid:4>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATESECURITYPOLICYDEPLOYMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "SecurityPolicyDeployment", "identifier": "<ocid:4>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETESECURITYPOLICYDEPLOYMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "SecurityPolicyDeployment", "identifier": "<ocid:4>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.SecurityPolicyDeployment, sdksvc.CreateSecurityPolicyDeploymentDetails, sdksvc.UpdateSecurityPolicyDeploymentDetails]{
		CollectionPath: "/20181201/securityPolicyDeployments", ItemPath: "/20181201/securityPolicyDeployments/<ocid:4>",
		CreatePath: "/20181201/securityPolicyDeployments", CreateMethod: http.MethodPost,
		UpdatePath: "/20181201/securityPolicyDeployments/<ocid:4>", UpdateMethod: http.MethodPut,
		DeletePath: "/20181201/securityPolicyDeployments/<ocid:4>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateSecurityPolicyDeploymentDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateSecurityPolicyDeploymentDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateSecurityPolicyDeploymentDetails) error {
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
	manager := &SecurityPolicyDeploymentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSecurityPolicyDeploymentRuntimeHooks(manager, sdkClient)
	client := wrapSecurityPolicyDeploymentGeneratedClient(hooks, defaultSecurityPolicyDeploymentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.SecurityPolicyDeployment](buildSecurityPolicyDeploymentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.SecurityPolicyDeployment]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.SecurityPolicyDeployment) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "DEPLOYED" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created SecurityPolicyDeployment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.SecurityPolicyDeployment) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.SecurityPolicyDeployment) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated SecurityPolicyDeployment status = %+v", current.Status)
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
