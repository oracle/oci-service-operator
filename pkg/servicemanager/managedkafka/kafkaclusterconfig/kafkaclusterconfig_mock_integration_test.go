/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package kafkaclusterconfig

import (
	"context"
	"fmt"
	managedkafkasdk "github.com/oracle/oci-go-sdk/v65/managedkafka"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationKafkaClusterConfigLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeKafkaClusterConfigResource()
	ocimock.InitializeResource(resource, "mock-kafkaclusterconfig")
	resource.Spec = ocimock.MustJSONFixture[managedkafkav1beta1.KafkaClusterConfigSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-config-sample",
  "freeformTags": {
    "env": "dev"
  },
  "latestConfig": {
    "properties": {
      "auto.create.topics.enable": "false"
    }
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "kafka-config-sample-updated"
}`)
	createRequest := ocimock.MustJSONFixture[managedkafkasdk.CreateKafkaClusterConfigDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-config-sample",
  "freeformTags": {
    "env": "dev"
  },
  "latestConfig": {
    "properties": {
      "auto.create.topics.enable": "false"
    }
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[managedkafkasdk.KafkaClusterConfig](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-config-sample",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "latestConfig": {
    "properties": {
      "auto.create.topics.enable": "false"
    }
  },
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[managedkafkasdk.UpdateKafkaClusterConfigDetails](t, `{
  "displayName": "kafka-config-sample-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[managedkafkasdk.KafkaClusterConfig](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "kafka-config-sample-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "latestConfig": {
    "properties": {
      "auto.create.topics.enable": "false"
    }
  },
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		managedkafkasdk.KafkaClusterConfig,
		managedkafkasdk.CreateKafkaClusterConfigDetails,
		managedkafkasdk.UpdateKafkaClusterConfigDetails,
	]{
		CollectionPath:    "/20240901/kafkaClusterConfigs",
		ItemPath:          "/20240901/kafkaClusterConfigs/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ managedkafkasdk.CreateKafkaClusterConfigDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ managedkafkasdk.KafkaClusterConfig) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20240901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close KafkaClusterConfig OCI mock: %v", err)
		}
	})
	sdkClient := managedkafkasdk.KafkaClusterClient{BaseClient: session.BaseClient()}
	client := newKafkaClusterConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managedkafkav1beta1.KafkaClusterConfig]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managedkafkav1beta1.KafkaClusterConfig) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LatestConfig, current.Spec.LatestConfig) {
				return fmt.Errorf("created KafkaClusterConfig status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managedkafkav1beta1.KafkaClusterConfig) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *managedkafkav1beta1.KafkaClusterConfig) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated KafkaClusterConfig status = %+v", current.Status)
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
