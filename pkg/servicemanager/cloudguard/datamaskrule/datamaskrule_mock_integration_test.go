/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package datamaskrule

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudguard"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationDataMaskRuleCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.DataMaskRule](t, `
{
  "metadata": {"name": "mock-datamaskrule", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "dataMaskCategories": [
    "PII"
  ],
  "description": "mock-description-updated",
  "displayName": "mock-displayname-initial",
  "iamGroupId": "<ocid:2>",
  "targetSelected": {
    "kind": "ALL"
  }
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-datamaskrule")
	resource.Status = apiv1beta1.DataMaskRuleStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDataMaskRuleDetails](t, `{
  "compartmentId": "<ocid:1>",
  "dataMaskCategories": [
    "PII"
  ],
  "description": "mock-description-updated",
  "displayName": "mock-displayname-initial",
  "iamGroupId": "<ocid:2>",
  "targetSelected": {
    "kind": "ALL"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateDataMaskRuleDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.DataMaskRule](t, `{
  "compartmentId": "<ocid:1>",
  "dataMaskCategories": [
    "PII"
  ],
  "description": "mock-description-updated",
  "displayName": "mock-displayname-initial",
  "iamGroupId": "<ocid:2>",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "targetSelected": {
    "kind": "ALL"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.DataMaskRule](t, `{
  "compartmentId": "<ocid:1>",
  "dataMaskCategories": [
    "PII"
  ],
  "description": "mock-description-updated",
  "displayName": "mock-displayname-updated",
  "iamGroupId": "<ocid:2>",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "targetSelected": {
    "kind": "ALL"
  }
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	updatingState := updatedState
	updatingState.LifecycleState = "UPDATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.DataMaskRule, sdksvc.CreateDataMaskRuleDetails, sdksvc.UpdateDataMaskRuleDetails]{
		CollectionPath: "/20200131/dataMaskRules", ItemPath: "/20200131/dataMaskRules/<ocid:3>",
		CreatePath: "/20200131/dataMaskRules", CreateMethod: http.MethodPost,
		UpdatePath: "/20200131/dataMaskRules/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200131/dataMaskRules/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateDataMaskRuleDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateDataMaskRuleDetails],
		UpdatedState:      &updatingState,
		UpdatedReadStates: ocimock.StateSequence(updatingState, updatedState),
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateDataMaskRuleDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard-cp-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &DataMaskRuleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDataMaskRuleRuntimeHooks(manager, sdkClient)
	client := wrapDataMaskRuleGeneratedClient(hooks, defaultDataMaskRuleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.DataMaskRule](buildDataMaskRuleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.DataMaskRule]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.DataMaskRule) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DataMaskRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.DataMaskRule) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.DataMaskRule) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DataMaskRule status = %+v", current.Status)
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
