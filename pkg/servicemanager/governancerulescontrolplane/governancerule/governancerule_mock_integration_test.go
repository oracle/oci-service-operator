/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package governancerule

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	governancerulescontrolplanesdk "github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type mockGovernanceRuleOCIClient struct {
	governancerulescontrolplanesdk.GovernanceRuleClient
	governancerulescontrolplanesdk.WorkRequestClient
}

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationGovernanceRuleWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[governancerulescontrolplanev1beta1.GovernanceRule](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "rule-alpha",
    "namespace": "default",
    "uid": "governance-rule-uid"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "creationOption": "TEMPLATE",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "governance quota rule",
    "displayName": "rule-alpha",
    "freeformTags": {
      "env": "dev"
    },
    "template": {
      "description": "quota template",
      "displayName": "quota-alpha",
      "statements": [
        "set compute-core quota standard-e4-core-count to 10 in tenancy"
      ],
      "type": "QUOTA"
    },
    "type": "QUOTA"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-governancerule")
	resource.Status = governancerulescontrolplanev1beta1.GovernanceRuleStatus{}
	createRequest := ocimock.MustJSONFixture[governancerulescontrolplanesdk.CreateGovernanceRuleDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "creationOption": "TEMPLATE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "governance quota rule",
  "displayName": "rule-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "template": {
    "description": "quota template",
    "displayName": "quota-alpha",
    "statements": [
      "set compute-core quota standard-e4-core-count to 10 in tenancy"
    ],
    "type": "QUOTA"
  },
  "type": "QUOTA"
}
`)
	updateRequest := ocimock.MustJSONFixture[governancerulescontrolplanesdk.UpdateGovernanceRuleDetails](t, `
{
  "description": "updated governance rule"
}
`)
	createdState := ocimock.MustOCIResponseFixture[governancerulescontrolplanesdk.GovernanceRule](t, `
{
  "compartmentId": "<ocid:1>",
  "creationOption": "TEMPLATE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "governance quota rule",
  "displayName": "rule-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "relatedResourceId": null,
  "systemTags": null,
  "template": {
    "description": "quota template",
    "displayName": "quota-alpha",
    "statements": [
      "set compute-core quota standard-e4-core-count to 10 in tenancy"
    ],
    "type": "QUOTA"
  },
  "timeCreated": "2026-04-29T12:00:00Z",
  "timeUpdated": "2026-04-29T12:00:00Z",
  "type": "QUOTA"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[governancerulescontrolplanesdk.GovernanceRule](t, `
{
  "compartmentId": "<ocid:1>",
  "creationOption": "TEMPLATE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated governance rule",
  "displayName": "rule-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "relatedResourceId": null,
  "systemTags": null,
  "template": {
    "description": "quota template",
    "displayName": "quota-alpha",
    "statements": [
      "set compute-core quota standard-e4-core-count to 10 in tenancy"
    ],
    "type": "QUOTA"
  },
  "timeCreated": "2026-04-29T12:00:00Z",
  "timeUpdated": "2026-04-29T12:00:00Z",
  "type": "QUOTA"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[governancerulescontrolplanesdk.GovernanceRule](t, `
{
  "compartmentId": "<ocid:1>",
  "creationOption": "TEMPLATE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated governance rule",
  "displayName": "rule-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:3>",
  "lifecycleState": "DELETED",
  "relatedResourceId": null,
  "systemTags": null,
  "template": {
    "description": "quota template",
    "displayName": "quota-alpha",
    "statements": [
      "set compute-core quota standard-e4-core-count to 10 in tenancy"
    ],
    "type": "QUOTA"
  },
  "timeCreated": "2026-04-29T12:00:00Z",
  "timeUpdated": "2026-04-29T12:00:00Z",
  "type": "QUOTA"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[governancerulescontrolplanesdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_GOVERNANCE_RULE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "GovernanceRule",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[governancerulescontrolplanesdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_GOVERNANCE_RULE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "GovernanceRule",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[governancerulescontrolplanesdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_GOVERNANCE_RULE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "GovernanceRule",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[governancerulescontrolplanesdk.GovernanceRule, governancerulescontrolplanesdk.CreateGovernanceRuleDetails, governancerulescontrolplanesdk.UpdateGovernanceRuleDetails]{
		CollectionPath: "/20220504/governanceRules", ItemPath: "/20220504/governanceRules/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ governancerulescontrolplanesdk.CreateGovernanceRuleDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220504/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20220504/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220504/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://governancerulescontrolplane.mock.invalid", BasePath: "20220504", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := governancerulescontrolplanesdk.GovernanceRuleClient{BaseClient: session.BaseClient()}
	sdkBundle := mockGovernanceRuleOCIClient{GovernanceRuleClient: sdkClient, WorkRequestClient: governancerulescontrolplanesdk.WorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &GovernanceRuleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newGovernanceRuleRuntimeHooksWithOCIClient(sdkBundle)
	applyGovernanceRuleRuntimeHooks(&hooks, sdkBundle, nil)
	client := wrapGovernanceRuleGeneratedClient(hooks, defaultGovernanceRuleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*governancerulescontrolplanev1beta1.GovernanceRule](buildGovernanceRuleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*governancerulescontrolplanev1beta1.GovernanceRule]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *governancerulescontrolplanev1beta1.GovernanceRule) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created GovernanceRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *governancerulescontrolplanev1beta1.GovernanceRule) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated governance rule"
}`)
		},
		ValidateUpdated: func(current *governancerulescontrolplanev1beta1.GovernanceRule) error {
			if !(current.Status.Description == "updated governance rule") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated GovernanceRule status = %+v", current.Status)
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
