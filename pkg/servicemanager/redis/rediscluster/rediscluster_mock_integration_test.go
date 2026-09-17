/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package rediscluster

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationRedisClusterWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &redisv1beta1.RedisCluster{}
	ocimock.InitializeResource(resource, "mock-rediscluster")
	resource.Spec = ocimock.MustJSONFixture[redisv1beta1.RedisClusterSpec](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "redis-sample",
  "freeformTags": {
    "env": "dev"
  },
  "nodeCount": 2,
  "nodeMemoryInGBs": 8,
  "softwareVersion": "V7_0_5",
  "subnetId": "<ocid:2>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "redis-updated"
}`)

	createRequest := ocimock.MustJSONFixture[redissdk.CreateRedisClusterDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "redis-sample",
  "freeformTags": {
    "env": "dev"
  },
  "nodeCount": 2,
  "nodeMemoryInGBs": 8,
  "softwareVersion": "V7_0_5",
  "subnetId": "<ocid:2>"
}`)
	updateRequest := ocimock.MustJSONFixture[redissdk.UpdateRedisClusterDetails](t, `{
  "displayName": "redis-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[redissdk.RedisCluster](t, `{
  "id": "<ocid:3>",
  "displayName": "redis-sample",
  "compartmentId": "<ocid:1>",
  "nodeCount": 2,
  "nodeMemoryInGBs": 8,
  "softwareVersion": "V7_0_5",
  "subnetId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "freeformTags": {
    "env": "dev"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>"
    }
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[redissdk.RedisCluster](t, `{
  "id": "<ocid:3>",
  "displayName": "redis-updated",
  "compartmentId": "<ocid:1>",
  "nodeCount": 2,
  "nodeMemoryInGBs": 8,
  "softwareVersion": "V7_0_5",
  "subnetId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "freeformTags": {
    "env": "dev"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>"
    }
  }
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[redissdk.WorkRequest](t, `{
  "id": "<ocid:4>",
  "status": "SUCCEEDED",
  "operationType": "CREATE_REDIS_CLUSTER",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "redis",
      "actionType": "CREATED",
      "identifier": "<ocid:3>",
      "entityUri": "/redisClusters/<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[redissdk.WorkRequest](t, `{
  "id": "<ocid:5>",
  "status": "SUCCEEDED",
  "operationType": "UPDATE_REDIS_CLUSTER",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "redis",
      "actionType": "UPDATED",
      "identifier": "<ocid:3>",
      "entityUri": "/redisClusters/<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[redissdk.WorkRequest](t, `{
  "id": "<ocid:6>",
  "status": "SUCCEEDED",
  "operationType": "DELETE_REDIS_CLUSTER",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "redis",
      "actionType": "DELETED",
      "identifier": "<ocid:3>",
      "entityUri": "/redisClusters/<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[redissdk.RedisCluster, redissdk.CreateRedisClusterDetails, redissdk.UpdateRedisClusterDetails]{
		CollectionPath: "/20220315/redisClusters", ItemPath: "/20220315/redisClusters/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ redissdk.CreateRedisClusterDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20220315/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest),
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20220315/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest),
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20220315/workRequests/<ocid:6>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://redis.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220315", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	_ = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := redissdk.RedisClusterClient{BaseClient: session.BaseClient()}
	client := newRedisTestManager(sdkClient).client
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*redisv1beta1.RedisCluster]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *redisv1beta1.RedisCluster) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created RedisCluster status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *redisv1beta1.RedisCluster) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *redisv1beta1.RedisCluster) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated RedisCluster status = %+v", current.Status)
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
