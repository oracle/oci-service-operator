/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package queue

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuesdk "github.com/oracle/oci-go-sdk/v65/queue"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationQueueWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &queuev1beta1.Queue{}
	ocimock.InitializeResource(resource, "mock-queue")
	resource.Spec = ocimock.MustJSONFixture[queuev1beta1.QueueSpec](t, `{
  "capabilities": [

  ],
  "channelConsumptionLimit": 100,
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-async-queue-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "retentionInSeconds": 86400,
  "timeoutInSeconds": 20,
  "visibilityInSeconds": 30
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "capabilities": [

  ],
  "definedTags": {
  },
  "displayName": "osok-mock-async-queue-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "visibilityInSeconds": 45
}`)

	createRequest := ocimock.MustJSONFixture[queuesdk.CreateQueueDetails](t, `{
  "capabilities": [

  ],
  "channelConsumptionLimit": 100,
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-async-queue-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "retentionInSeconds": 86400,
  "timeoutInSeconds": 20,
  "visibilityInSeconds": 30
}`)
	updateRequest := ocimock.MustJSONFixture[queuesdk.UpdateQueueDetails](t, `{
  "capabilities": [

  ],
  "definedTags": {
  },
  "displayName": "osok-mock-async-queue-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "visibilityInSeconds": 45
}`)
	createdState := ocimock.MustOCIResponseFixture[queuesdk.Queue](t, `{
  "capabilities": [

  ],
  "channelConsumptionLimit": 100,
  "compartmentId": "<ocid:1>",
  "customEncryptionKeyId": null,
  "deadLetterQueueDeliveryCount": 0,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:10:02.867Z"
    }
  },
  "displayName": "osok-mock-async-queue-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "messagesEndpoint": "https://cell-1.queue.messaging.us-ashburn-1.oci.oraclecloud.com",
  "retentionInSeconds": 86400,
  "systemTags": {
  },
  "timeCreated": "2026-08-31T21:10:03.031Z",
  "timeUpdated": "2026-08-31T21:11:02.664Z",
  "timeoutInSeconds": 20,
  "visibilityInSeconds": 30
}`)
	updatedState := ocimock.MustOCIResponseFixture[queuesdk.Queue](t, `{
  "capabilities": [

  ],
  "channelConsumptionLimit": 100,
  "compartmentId": "<ocid:1>",
  "customEncryptionKeyId": null,
  "deadLetterQueueDeliveryCount": 0,
  "definedTags": {
  },
  "displayName": "osok-mock-async-queue-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
	  "messagesEndpoint": "https://cell-2.queue.messaging.us-ashburn-1.oci.oraclecloud.com",
  "retentionInSeconds": 86400,
  "systemTags": {
  },
  "timeCreated": "2026-08-31T21:10:03.031Z",
  "timeUpdated": "2026-08-31T21:11:04.543Z",
  "timeoutInSeconds": 20,
  "visibilityInSeconds": 45
}`)
	deletedState := ocimock.MustOCIResponseFixture[queuesdk.Queue](t, `{
  "capabilities": [

  ],
  "channelConsumptionLimit": 100,
  "compartmentId": "<ocid:1>",
  "customEncryptionKeyId": null,
  "deadLetterQueueDeliveryCount": 0,
  "definedTags": {
  },
  "displayName": "osok-mock-async-queue-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "messagesEndpoint": "https://cell-2.queue.messaging.us-ashburn-1.oci.oraclecloud.com",
  "retentionInSeconds": 86400,
  "systemTags": {
  },
  "timeCreated": "2026-08-31T21:10:03.031Z",
  "timeUpdated": "2026-08-31T21:11:45.743Z",
  "timeoutInSeconds": 20,
  "visibilityInSeconds": 45
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[queuesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_QUEUE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "queue",
      "entityUri": "/queues/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:10:03.036Z",
  "timeFinished": "2026-08-31T21:11:02.664Z",
  "timeStarted": "2026-08-31T21:11:02.414Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[queuesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_QUEUE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "queue",
      "entityUri": "/queues/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:11:04.554Z",
  "timeFinished": "2026-08-31T21:11:04.560Z",
  "timeStarted": "2026-08-31T21:11:04.560Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[queuesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_QUEUE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "queue",
      "entityUri": "/queues/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:11:06.404Z",
  "timeFinished": "2026-08-31T21:11:45.736Z",
  "timeStarted": "2026-08-31T21:11:45.458Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[queuesdk.Queue, queuesdk.CreateQueueDetails, queuesdk.UpdateQueueDetails]{
		CollectionPath: "/20210201/queues", ItemPath: "/20210201/queues/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeletedState: &deletedState, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ queuesdk.CreateQueueDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20210201/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20210201/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20210201/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://messaging.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	secretExists := false
	secretRecord := credhelper.SecretRecord{}
	credentials := &fakeQueueCredentialClient{}
	credentials.getSecretRecordFn = func(_ context.Context, name, namespace string) (credhelper.SecretRecord, error) {
		if name != resource.Name || namespace != resource.Namespace {
			return credhelper.SecretRecord{}, fmt.Errorf("Queue endpoint Secret target = %s/%s, want %s/%s", namespace, name, resource.Namespace, resource.Name)
		}
		if !secretExists {
			return credhelper.SecretRecord{}, apierrors.NewNotFound(corev1.Resource("secret"), name)
		}
		return secretRecord, nil
	}
	credentials.createSecretFn = func(_ context.Context, name, namespace string, labels map[string]string, data map[string][]byte) (bool, error) {
		if name != resource.Name || namespace != resource.Namespace ||
			string(data["endpoint"]) != resource.Status.MessagesEndpoint ||
			labels[queueEndpointSecretOwnerUIDLabel] != string(resource.UID) {
			return false, fmt.Errorf("unexpected Queue endpoint Secret create target or data")
		}
		secretRecord = credhelper.SecretRecord{UID: "mock-secret-uid", Labels: cloneQueueSecretLabels(labels), Data: cloneQueueSecretData(data)}
		secretExists = true
		return true, nil
	}
	credentials.updateSecretIfCurrentFn = func(_ context.Context, name, namespace string, current credhelper.SecretRecord, labels map[string]string, data map[string][]byte) (bool, error) {
		if name != resource.Name || namespace != resource.Namespace || current.UID != secretRecord.UID ||
			string(data["endpoint"]) != resource.Status.MessagesEndpoint {
			return false, fmt.Errorf("unexpected Queue endpoint Secret guarded update")
		}
		if labels != nil {
			secretRecord.Labels = cloneQueueSecretLabels(labels)
		}
		secretRecord.Data = cloneQueueSecretData(data)
		return true, nil
	}
	credentials.deleteSecretIfCurrentFn = func(_ context.Context, name, namespace string, current credhelper.SecretRecord) (bool, error) {
		if name != resource.Name || namespace != resource.Namespace || current.UID != secretRecord.UID {
			return false, fmt.Errorf("unexpected Queue endpoint Secret guarded delete")
		}
		secretExists = false
		return true, nil
	}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := queuesdk.QueueAdminClient{BaseClient: session.BaseClient()}
	manager := NewQueueServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), credentials, nil, log, nil)
	hooks := newTestQueueRuntimeHooks(manager, sdkClient)
	appendQueueEndpointSecretWrapper(manager, &hooks)
	client := wrapQueueGeneratedClient(hooks, defaultQueueServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*queuev1beta1.Queue](buildQueueGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*queuev1beta1.Queue]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *queuev1beta1.Queue) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.MessagesEndpoint == "" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Queue status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *queuev1beta1.Queue) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *queuev1beta1.Queue) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.VisibilityInSeconds != current.Spec.VisibilityInSeconds || current.Status.OsokStatus.Async.Current != nil ||
				!secretExists || string(secretRecord.Data["endpoint"]) != current.Status.MessagesEndpoint {
				return fmt.Errorf("updated Queue status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !credentials.createCalled || !credentials.updateCalled || !credentials.deleteCalled || secretExists {
		t.Fatalf("Queue endpoint Secret lifecycle create=%t update=%t delete=%t exists=%t", credentials.createCalled, credentials.updateCalled, credentials.deleteCalled, secretExists)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
