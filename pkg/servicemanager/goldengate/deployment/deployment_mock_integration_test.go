/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package deployment

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/goldengate"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/goldengate/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationDeploymentCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Deployment](t, `
{
  "metadata": {"name": "mock-deployment", "namespace": "default"},
  "spec": {
  "availabilityDomain": "mock-availabilitydomain-updated",
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:1>",
  "deploymentBackupId": "<ocid:9>",
  "deploymentType": "OGG",
  "displayName": "mock-displayname-initial",
  "faultDomain": "mock-faultdomain-updated",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "sourceDeploymentId": "<ocid:9>",
  "subnetId": "<ocid:2>",
  "subscriptionId": "<ocid:9>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-deployment")
	resource.Status = apiv1beta1.DeploymentStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDeploymentDetails](t, `{
  "availabilityDomain": "mock-availabilitydomain-updated",
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:1>",
  "deploymentBackupId": "<ocid:9>",
  "deploymentType": "OGG",
  "displayName": "mock-displayname-initial",
  "faultDomain": "mock-faultdomain-updated",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "sourceDeploymentId": "<ocid:9>",
  "subnetId": "<ocid:2>",
  "subscriptionId": "<ocid:9>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateDeploymentDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Deployment](t, `{
  "availabilityDomain": "mock-availabilitydomain-updated",
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:1>",
  "deploymentBackupId": "<ocid:9>",
  "deploymentType": "OGG",
  "displayName": "mock-displayname-initial",
  "faultDomain": "mock-faultdomain-updated",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sourceDeploymentId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:2>",
  "subscriptionId": "<ocid:9>"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Deployment](t, `{
  "availabilityDomain": "mock-availabilitydomain-updated",
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:1>",
  "deploymentBackupId": "<ocid:9>",
  "deploymentType": "OGG",
  "displayName": "mock-displayname-updated",
  "faultDomain": "mock-faultdomain-updated",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sourceDeploymentId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:2>",
  "subscriptionId": "<ocid:9>"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEDEPLOYMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Deployment", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEDEPLOYMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "Deployment", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEDEPLOYMENT",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Deployment", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Deployment, sdksvc.CreateDeploymentDetails, sdksvc.UpdateDeploymentDetails]{
		CollectionPath: "/20200407/deployments", ItemPath: "/20200407/deployments/<ocid:3>",
		CreatePath: "/20200407/deployments", CreateMethod: http.MethodPost,
		UpdatePath: "/20200407/deployments/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200407/deployments/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateDeploymentDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateDeploymentDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateDeploymentDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://goldengate.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200407", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.GoldenGateClient{BaseClient: session.BaseClient()}
	manager := &DeploymentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDeploymentRuntimeHooks(manager, sdkClient)
	client := wrapDeploymentGeneratedClient(hooks, defaultDeploymentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Deployment](buildDeploymentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Deployment]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Deployment) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Deployment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Deployment) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Deployment) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Deployment status = %+v", current.Status)
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
