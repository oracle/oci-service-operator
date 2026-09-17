/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package jmsplugin

import (
	"context"
	"fmt"
	jmssdk "github.com/oracle/oci-go-sdk/v65/jms"
	jmsv1beta1 "github.com/oracle/oci-service-operator/api/jms/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationJmsPluginLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeJmsPluginResource()
	ocimock.InitializeResource(resource, "mock-jmsplugin")
	resource.Spec = ocimock.MustJSONFixture[jmsv1beta1.JmsPluginSpec](t, `{
  "agentId": "\u003cocid:1\u003e",
  "agentType": "OCA",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Operations": {
      "team": "platform"
    }
  },
  "freeformTags": {
    "env": "dev"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[jmssdk.CreateJmsPluginDetails](t, `{
  "agentId": "\u003cocid:1\u003e",
  "agentType": "OCA",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Operations": {
      "team": "platform"
    }
  },
  "freeformTags": {
    "env": "dev"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[jmssdk.JmsPlugin](t, `{
  "agentId": "\u003cocid:1\u003e",
  "agentType": "OCA",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Operations": {
      "team": "platform"
    }
  },
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[jmssdk.UpdateJmsPluginDetails](t, `{
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[jmssdk.JmsPlugin](t, `{
  "agentId": "\u003cocid:1\u003e",
  "agentType": "OCA",
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Operations": {
      "team": "platform"
    }
  },
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		jmssdk.JmsPlugin,
		jmssdk.CreateJmsPluginDetails,
		jmssdk.UpdateJmsPluginDetails,
	]{
		CollectionPath:    "/20210610/jmsPlugins",
		ItemPath:          "/20210610/jmsPlugins/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ jmssdk.CreateJmsPluginDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ jmssdk.JmsPlugin) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210610", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close JmsPlugin OCI mock: %v", err)
		}
	})
	sdkClient := jmssdk.JavaManagementServiceClient{BaseClient: session.BaseClient()}
	client := newJmsPluginServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*jmsv1beta1.JmsPlugin]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *jmsv1beta1.JmsPlugin) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AgentId, current.Spec.AgentId) ||
				!reflect.DeepEqual(current.Status.AgentType, current.Spec.AgentType) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created JmsPlugin status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *jmsv1beta1.JmsPlugin) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *jmsv1beta1.JmsPlugin) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated JmsPlugin status = %+v", current.Status)
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
