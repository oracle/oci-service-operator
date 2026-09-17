/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package domaingovernance

import (
	"context"
	"fmt"
	tenantmanagercontrolplanesdk "github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane"
	tenantmanagercontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/tenantmanagercontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDomainGovernanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newDomainGovernanceResource()
	ocimock.InitializeResource(resource, "mock-domaingovernance")
	resource.Spec = ocimock.MustJSONFixture[tenantmanagercontrolplanev1beta1.DomainGovernanceSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "domainId": "\u003cocid:2\u003e",
  "freeformTags": {
    "env": "test"
  },
  "onsSubscriptionId": "\u003cocid:3\u003e",
  "onsTopicId": "\u003cocid:4\u003e",
  "subscriptionEmail": "admin@example.com"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[tenantmanagercontrolplanesdk.CreateDomainGovernanceDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "domainId": "\u003cocid:2\u003e",
  "freeformTags": {
    "env": "test"
  },
  "onsSubscriptionId": "\u003cocid:3\u003e",
  "onsTopicId": "\u003cocid:4\u003e",
  "subscriptionEmail": "admin@example.com"
}`)
	createdState := ocimock.MustOCIResponseFixture[tenantmanagercontrolplanesdk.DomainGovernance](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "domainId": "\u003cocid:2\u003e",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:5\u003e",
  "lifecycleState": "ACTIVE",
  "onsSubscriptionId": "\u003cocid:3\u003e",
  "onsTopicId": "\u003cocid:4\u003e",
  "subscriptionEmail": "admin@example.com"
}`)
	updateRequest := ocimock.MustJSONFixture[tenantmanagercontrolplanesdk.UpdateDomainGovernanceDetails](t, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[tenantmanagercontrolplanesdk.DomainGovernance](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "domainId": "\u003cocid:2\u003e",
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:5\u003e",
  "lifecycleState": "ACTIVE",
  "onsSubscriptionId": "\u003cocid:3\u003e",
  "onsTopicId": "\u003cocid:4\u003e",
  "subscriptionEmail": "admin@example.com"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		tenantmanagercontrolplanesdk.DomainGovernance,
		tenantmanagercontrolplanesdk.CreateDomainGovernanceDetails,
		tenantmanagercontrolplanesdk.UpdateDomainGovernanceDetails,
	]{
		CollectionPath:    "/20230401/domainGovernances",
		ItemPath:          "/20230401/domainGovernances/<ocid:5>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		CreatedReadStates: ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		UpdatedReadStates: ocimock.LifecycleStateSequence(t, updatedState, "UPDATING"),
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ tenantmanagercontrolplanesdk.CreateDomainGovernanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ tenantmanagercontrolplanesdk.DomainGovernance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20230401", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DomainGovernance OCI mock: %v", err)
		}
	})
	sdkClient := tenantmanagercontrolplanesdk.DomainGovernanceClient{BaseClient: session.BaseClient()}
	client := newDomainGovernanceServiceClientWithClients(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*tenantmanagercontrolplanev1beta1.DomainGovernance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *tenantmanagercontrolplanev1beta1.DomainGovernance) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DomainId, current.Spec.DomainId) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.OnsSubscriptionId, current.Spec.OnsSubscriptionId) ||
				!reflect.DeepEqual(current.Status.OnsTopicId, current.Spec.OnsTopicId) ||
				!reflect.DeepEqual(current.Status.SubscriptionEmail, current.Spec.SubscriptionEmail) {
				return fmt.Errorf("created DomainGovernance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *tenantmanagercontrolplanev1beta1.DomainGovernance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *tenantmanagercontrolplanev1beta1.DomainGovernance) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated DomainGovernance status = %+v", current.Status)
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
