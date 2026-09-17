/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package kafkacluster

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	managedkafkasdk "github.com/oracle/oci-go-sdk/v65/managedkafka"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationKafkaClusterWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[managedkafkav1beta1.KafkaCluster](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "kafka-cluster-sample",
    "namespace": "default",
    "uid": "uid-kafka-cluster-sample"
  },
  "spec": {
    "accessSubnets": [
      {
        "subnets": [
          "<ocid:1>",
          "<ocid:2>"
        ]
      }
    ],
    "brokerShape": {
      "nodeCount": 1,
      "nodeShape": "VM.Standard.E4.Flex",
      "ocpuCount": 1
    },
    "clusterConfigId": "<ocid:3>",
    "clusterConfigVersion": 1,
    "clusterType": "DEVELOPMENT",
    "compartmentId": "<ocid:4>",
    "coordinationType": "KRAFT",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "displayName": "kafka-cluster-sample",
    "freeformTags": {
      "owner": "osok"
    },
    "kafkaVersion": "3.8.0"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-kafkacluster")
	resource.Status = managedkafkav1beta1.KafkaClusterStatus{}
	createRequest := ocimock.MustJSONFixture[managedkafkasdk.CreateKafkaClusterDetails](t, `
{
  "accessSubnets": [
    {
      "subnets": [
        "<ocid:1>",
        "<ocid:2>"
      ]
    }
  ],
  "brokerShape": {
    "nodeCount": 1,
    "nodeShape": "VM.Standard.E4.Flex",
    "ocpuCount": 1
  },
  "clusterConfigId": "<ocid:3>",
  "clusterConfigVersion": 1,
  "clusterType": "DEVELOPMENT",
  "compartmentId": "<ocid:4>",
  "coordinationType": "KRAFT",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-cluster-sample",
  "freeformTags": {
    "owner": "osok"
  },
  "kafkaVersion": "3.8.0"
}
`)
	updateRequest := ocimock.MustJSONFixture[managedkafkasdk.UpdateKafkaClusterDetails](t, `
{
  "displayName": "kafka-cluster-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[managedkafkasdk.KafkaCluster](t, `
{
  "accessSubnets": [
    {
      "subnets": [
        "<ocid:1>",
        "<ocid:2>"
      ]
    }
  ],
  "brokerShape": {
    "nodeCount": 1,
    "nodeShape": "VM.Standard.E4.Flex",
    "ocpuCount": 1
  },
  "clusterConfigId": "<ocid:3>",
  "clusterConfigVersion": 1,
  "clusterType": "DEVELOPMENT",
  "compartmentId": "<ocid:4>",
  "coordinationType": "KRAFT",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-cluster-sample",
  "freeformTags": {
    "owner": "osok"
  },
  "id": "<ocid:5>",
  "kafkaVersion": "3.8.0",
  "lifecycleState": "ACTIVE"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[managedkafkasdk.KafkaCluster](t, `
{
  "accessSubnets": [
    {
      "subnets": [
        "<ocid:1>",
        "<ocid:2>"
      ]
    }
  ],
  "brokerShape": {
    "nodeCount": 1,
    "nodeShape": "VM.Standard.E4.Flex",
    "ocpuCount": 1
  },
  "clusterConfigId": "<ocid:3>",
  "clusterConfigVersion": 1,
  "clusterType": "DEVELOPMENT",
  "compartmentId": "<ocid:4>",
  "coordinationType": "KRAFT",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-cluster-updated",
  "freeformTags": {
    "owner": "osok"
  },
  "id": "<ocid:5>",
  "kafkaVersion": "3.8.0",
  "lifecycleState": "ACTIVE"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[managedkafkasdk.KafkaCluster](t, `
{
  "accessSubnets": [
    {
      "subnets": [
        "<ocid:1>",
        "<ocid:2>"
      ]
    }
  ],
  "brokerShape": {
    "nodeCount": 1,
    "nodeShape": "VM.Standard.E4.Flex",
    "ocpuCount": 1
  },
  "clusterConfigId": "<ocid:3>",
  "clusterConfigVersion": 1,
  "clusterType": "DEVELOPMENT",
  "compartmentId": "<ocid:4>",
  "coordinationType": "KRAFT",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-cluster-updated",
  "freeformTags": {
    "owner": "osok"
  },
  "id": "<ocid:5>",
  "kafkaVersion": "3.8.0",
  "lifecycleState": "DELETED"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[managedkafkasdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "KafkaCluster",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[managedkafkasdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "KafkaCluster",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[managedkafkasdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "KafkaCluster",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[managedkafkasdk.KafkaCluster, managedkafkasdk.CreateKafkaClusterDetails, managedkafkasdk.UpdateKafkaClusterDetails]{
		CollectionPath: "/20240901/kafkaClusters", ItemPath: "/20240901/kafkaClusters/<ocid:5>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ managedkafkasdk.CreateKafkaClusterDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20240901/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20240901/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20240901/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://managedkafka.mock.invalid", BasePath: "20240901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := managedkafkasdk.KafkaClusterClient{BaseClient: session.BaseClient()}
	hooks := newKafkaClusterDefaultRuntimeHooks(sdkClient)
	applyKafkaClusterRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapKafkaClusterGeneratedClient(hooks, defaultKafkaClusterServiceClient{ServiceClient: generatedruntime.NewServiceClient[*managedkafkav1beta1.KafkaCluster](buildKafkaClusterGeneratedRuntimeConfig(&KafkaClusterServiceManager{}, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managedkafkav1beta1.KafkaCluster]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managedkafkav1beta1.KafkaCluster) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created KafkaCluster status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managedkafkav1beta1.KafkaCluster) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "kafka-cluster-updated"
}`)
		},
		ValidateUpdated: func(current *managedkafkav1beta1.KafkaCluster) error {
			if !(current.Status.DisplayName == "kafka-cluster-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated KafkaCluster status = %+v", current.Status)
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
