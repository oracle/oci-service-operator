/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package protecteddatabase

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

type mockProtectedDatabaseCredentialClient struct {
	secrets map[string]map[string][]byte
	creates int
	updates int
	deletes int
}

func newMockProtectedDatabaseCredentialClient() *mockProtectedDatabaseCredentialClient {
	return &mockProtectedDatabaseCredentialClient{secrets: map[string]map[string][]byte{}}
}

func (c *mockProtectedDatabaseCredentialClient) CreateSecret(_ context.Context, name, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
	c.creates++
	c.secrets[namespace+"/"+name] = cloneMockProtectedDatabaseSecret(data)
	return true, nil
}

func (c *mockProtectedDatabaseCredentialClient) UpdateSecret(_ context.Context, name, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
	c.updates++
	c.secrets[namespace+"/"+name] = cloneMockProtectedDatabaseSecret(data)
	return true, nil
}

func (c *mockProtectedDatabaseCredentialClient) GetSecret(_ context.Context, name, namespace string) (map[string][]byte, error) {
	data, ok := c.secrets[namespace+"/"+name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	return cloneMockProtectedDatabaseSecret(data), nil
}

func (c *mockProtectedDatabaseCredentialClient) DeleteSecret(_ context.Context, name, namespace string) (bool, error) {
	c.deletes++
	key := namespace + "/" + name
	if _, ok := c.secrets[key]; !ok {
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	delete(c.secrets, key)
	return true, nil
}

func cloneMockProtectedDatabaseSecret(data map[string][]byte) map[string][]byte {
	cloned := make(map[string][]byte, len(data))
	for key, value := range data {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationProtectedDatabaseWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[recoveryv1beta1.ProtectedDatabase](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "protected-db",
    "namespace": "default"
  },
  "spec": {
    "changeRate": 0.25,
    "compartmentId": "<ocid:1>",
    "compressionRatio": 1.5,
    "databaseId": "<ocid:2>",
    "databaseSize": "M",
    "dbUniqueName": "CDB01",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "displayName": "protected-db",
    "freeformTags": {
      "env": "test"
    },
    "isRedoLogsShipped": true,
    "password": "<redacted>",
    "protectionPolicyId": "<ocid:3>",
    "recoveryServiceSubnets": [
      {
        "recoveryServiceSubnetId": "<ocid:4>"
      }
    ],
    "subscriptionId": "subscription-1"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-protecteddatabase")
	resource.Status = recoveryv1beta1.ProtectedDatabaseStatus{}
	changeSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[recoverysdk.CreateProtectedDatabaseDetails](t, `
{
  "changeRate": 0.25,
  "compartmentId": "<ocid:1>",
  "compressionRatio": 1.5,
  "databaseId": "<ocid:2>",
  "databaseSize": "M",
  "dbUniqueName": "CDB01",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "protected-db",
  "freeformTags": {
    "env": "test"
  },
  "isRedoLogsShipped": true,
  "password": "<redacted>",
  "protectionPolicyId": "<ocid:3>",
  "recoveryServiceSubnets": [
    {
      "recoveryServiceSubnetId": "<ocid:4>"
    }
  ],
  "subscriptionId": "subscription-1"
}
`)
	updateRequest := ocimock.MustJSONFixture[recoverysdk.UpdateProtectedDatabaseDetails](t, `
{
  "displayName": "protected-db-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[recoverysdk.ProtectedDatabase](t, `
{
  "changeRate": 0.25,
  "compartmentId": "<ocid:1>",
  "compressionRatio": 1.5,
  "databaseId": "<ocid:2>",
  "databaseSize": "M",
  "databaseSizeInGBs": null,
  "dbUniqueName": "CDB01",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "protected-db",
  "freeformTags": {
    "env": "test"
  },
  "healthDetails": null,
  "id": "<ocid:6>",
  "isReadOnlyResource": null,
  "isRedoLogsShipped": true,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "metrics": null,
  "policyLockedDateTime": null,
  "protectionPolicyId": "<ocid:3>",
  "recoveryServiceSubnets": [
    {
      "lifecycleState": "ACTIVE",
      "recoveryServiceSubnetId": "<ocid:4>"
    }
  ],
  "subscriptionId": "subscription-1",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "vpcUserName": "vpc-user"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[recoverysdk.ProtectedDatabase](t, `
{
  "changeRate": 0.25,
  "compartmentId": "<ocid:1>",
  "compressionRatio": 1.5,
  "databaseId": "<ocid:2>",
  "databaseSize": "M",
  "databaseSizeInGBs": null,
  "dbUniqueName": "CDB01",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "protected-db-updated",
  "freeformTags": {
    "env": "test"
  },
  "healthDetails": null,
  "id": "<ocid:6>",
  "isReadOnlyResource": null,
  "isRedoLogsShipped": true,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "metrics": null,
  "policyLockedDateTime": null,
  "protectionPolicyId": "<ocid:3>",
  "recoveryServiceSubnets": [
    {
      "lifecycleState": "ACTIVE",
      "recoveryServiceSubnetId": "<ocid:4>"
    }
  ],
  "subscriptionId": "subscription-1",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "vpcUserName": "vpc-user"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[recoverysdk.ProtectedDatabase](t, `
{
  "changeRate": 0.25,
  "compartmentId": "<ocid:1>",
  "compressionRatio": 1.5,
  "databaseId": "<ocid:2>",
  "databaseSize": "M",
  "databaseSizeInGBs": null,
  "dbUniqueName": "CDB01",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "protected-db-updated",
  "freeformTags": {
    "env": "test"
  },
  "healthDetails": null,
  "id": "<ocid:6>",
  "isReadOnlyResource": null,
  "isRedoLogsShipped": true,
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "metrics": null,
  "policyLockedDateTime": null,
  "protectionPolicyId": "<ocid:3>",
  "recoveryServiceSubnets": [
    {
      "lifecycleState": "ACTIVE",
      "recoveryServiceSubnetId": "<ocid:4>"
    }
  ],
  "subscriptionId": "subscription-1",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "vpcUserName": "vpc-user"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "ProtectedDatabase",
      "identifier": "<ocid:6>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "ProtectedDatabase",
      "identifier": "<ocid:6>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "ProtectedDatabase",
      "identifier": "<ocid:6>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[recoverysdk.ProtectedDatabase, recoverysdk.CreateProtectedDatabaseDetails, recoverysdk.UpdateProtectedDatabaseDetails]{
		CollectionPath: "/20210216/protectedDatabases", ItemPath: "/20210216/protectedDatabases/<ocid:6>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ recoverysdk.CreateProtectedDatabaseDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://recovery.mock.invalid", BasePath: "20210216", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := recoverysdk.DatabaseRecoveryClient{BaseClient: session.BaseClient()}
	credentialClient := newMockProtectedDatabaseCredentialClient()
	client := newProtectedDatabaseServiceClientWithOCIClientAndCredentialClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient, credentialClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*recoveryv1beta1.ProtectedDatabase]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *recoveryv1beta1.ProtectedDatabase) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ProtectedDatabase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *recoveryv1beta1.ProtectedDatabase) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "protected-db-updated"
}`)
		},
		ValidateUpdated: func(current *recoveryv1beta1.ProtectedDatabase) error {
			if !(current.Status.DisplayName == "protected-db-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ProtectedDatabase status = %+v", current.Status)
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
	if credentialClient.creates == 0 || credentialClient.deletes == 0 || len(credentialClient.secrets) != 0 {
		t.Fatalf("ProtectedDatabase password-state Secret calls create/update/delete=%d/%d/%d remaining=%d", credentialClient.creates, credentialClient.updates, credentialClient.deletes, len(credentialClient.secrets))
	}

	changeResource := &recoveryv1beta1.ProtectedDatabase{Spec: changeSpec}
	ocimock.InitializeResource(changeResource, "mock-protecteddatabase-change-operations")
	changeResource.Spec.CompartmentId = "<ocid:moved-compartment>"
	changeResource.Spec.SubscriptionId = "subscription-2"
	changeResource.Status.Id = "<ocid:6>"
	changeResource.Status.OsokStatus.Ocid = "<ocid:6>"
	changeResource.Status.CompartmentId = "<ocid:1>"
	changeResource.Status.SubscriptionId = "subscription-1"
	compartmentState := createdState
	compartmentState.CompartmentId = &changeResource.Spec.CompartmentId
	subscriptionState := compartmentState
	subscriptionState.SubscriptionId = &changeResource.Spec.SubscriptionId
	changeStage := 0
	compartmentWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "id": "<ocid:compartment-work-request>",
  "operationType": "MOVE_PROTECTED_DATABASE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "ProtectedDatabase", "identifier": "<ocid:6>"}],
  "status": "SUCCEEDED"
}
`)
	subscriptionWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "id": "<ocid:subscription-work-request>",
  "operationType": "UPDATE_PROTECTED_DATABASE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "ProtectedDatabase", "identifier": "<ocid:6>"}],
  "status": "SUCCEEDED"
}
`)
	changeResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[recoverysdk.ProtectedDatabase]{
		CollectionPath: "/20210216/protectedDatabases", ItemPath: "/20210216/protectedDatabases/<ocid:6>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		ReadTransition: func(_ ocimock.Request, state recoverysdk.ProtectedDatabase) (recoverysdk.ProtectedDatabase, ocimock.Response, error) {
			switch changeStage {
			case 1:
				state = compartmentState
			case 2:
				state = subscriptionState
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "change-compartment", Method: http.MethodPost, Path: "/20210216/protectedDatabases/<ocid:6>/actions/changeCompartment", MinimumCalls: 1,
				Respond: func(request ocimock.Request) (ocimock.Response, error) {
					expected := recoverysdk.ChangeProtectedDatabaseCompartmentDetails{CompartmentId: &changeResource.Spec.CompartmentId}
					if err := ocimock.ValidateJSONRequest(request, expected); err != nil {
						return ocimock.Response{}, err
					}
					changeStage = 1
					return ocimock.Response{StatusCode: http.StatusAccepted, Header: http.Header{
						"Opc-Work-Request-Id": []string{"<ocid:compartment-work-request>"},
						"Opc-Request-Id":      []string{"mock-compartment-request"},
					}}, nil
				},
			},
			{
				Name: "compartment-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:compartment-work-request>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, compartmentWorkRequest)
				},
			},
			{
				Name: "change-subscription", Method: http.MethodPost, Path: "/20210216/protectedDatabases/<ocid:6>/actions/changeSubscription", MinimumCalls: 1,
				Respond: func(request ocimock.Request) (ocimock.Response, error) {
					expected := recoverysdk.ChangeProtectedDatabaseSubscriptionDetails{SubscriptionId: &changeResource.Spec.SubscriptionId}
					if err := ocimock.ValidateJSONRequest(request, expected); err != nil {
						return ocimock.Response{}, err
					}
					changeStage = 2
					return ocimock.Response{StatusCode: http.StatusAccepted, Header: http.Header{
						"Opc-Work-Request-Id": []string{"<ocid:subscription-work-request>"},
						"Opc-Request-Id":      []string{"mock-subscription-request"},
					}}, nil
				},
			},
			{
				Name: "subscription-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:subscription-work-request>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, subscriptionWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	changeSession, err := ocimock.Open(ocimock.Options{Host: "https://recovery.mock.invalid", BasePath: "20210216", Responder: changeResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = changeSession.Close() })
	changeCredentialClient := newMockProtectedDatabaseCredentialClient()
	changeClient := newProtectedDatabaseServiceClientWithOCIClientAndCredentialClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration-change-operations")},
		recoverysdk.DatabaseRecoveryClient{BaseClient: changeSession.BaseClient()},
		changeCredentialClient,
	)
	changeResponse, err := changeClient.CreateOrUpdate(context.Background(), changeResource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if !changeResponse.IsSuccessful || changeResponse.ShouldRequeue || changeStage != 2 ||
		changeResource.Status.CompartmentId != changeResource.Spec.CompartmentId ||
		changeResource.Status.SubscriptionId != changeResource.Spec.SubscriptionId ||
		changeResource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("ProtectedDatabase change-operation response=%+v stage=%d status=%+v", changeResponse, changeStage, changeResource.Status)
	}
	if err := changeSession.Close(); err != nil {
		t.Fatal(err)
	}
}
