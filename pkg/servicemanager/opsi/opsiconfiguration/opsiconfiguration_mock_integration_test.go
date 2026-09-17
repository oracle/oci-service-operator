/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package opsiconfiguration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationOpsiConfigurationWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.OpsiConfiguration](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "opsiconfiguration-sample",
    "namespace": "default",
    "uid": "uid-opsiconfiguration"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "initial description",
    "displayName": "opsi-configuration",
    "freeformTags": {
      "env": "test"
    },
    "opsiConfigType": "UX_CONFIGURATION",
    "systemTags": {
      "orcl-cloud": {
        "free-tier-retained": "true"
      }
    }
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-opsiconfiguration")
	resource.Status = opsiv1beta1.OpsiConfigurationStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateOpsiUxConfigurationDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "displayName": "opsi-configuration",
  "freeformTags": {
    "env": "test"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateOpsiUxConfigurationDetails](t, `
{
  "description": "updated opsi configuration"
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.OpsiUxConfiguration](t, `
{
  "compartmentId": "<ocid:1>",
  "configItems": [
    {
      "applicableContexts": null,
      "configItemType": "BASIC",
      "defaultValue": null,
      "metadata": null,
      "name": "ui.theme",
      "value": "dark"
    }
  ],
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "displayName": "opsi-configuration",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "opsiConfigType": "UX_CONFIGURATION",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.OpsiUxConfiguration](t, `
{
  "compartmentId": "<ocid:1>",
  "configItems": [
    {
      "applicableContexts": null,
      "configItemType": "BASIC",
      "defaultValue": null,
      "metadata": null,
      "name": "ui.theme",
      "value": "dark"
    }
  ],
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated opsi configuration",
  "displayName": "opsi-configuration",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "opsiConfigType": "UX_CONFIGURATION",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.OpsiUxConfiguration](t, `
{
  "compartmentId": "<ocid:1>",
  "configItems": [
    {
      "applicableContexts": null,
      "configItemType": "BASIC",
      "defaultValue": null,
      "metadata": null,
      "name": "ui.theme",
      "value": "dark"
    }
  ],
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated opsi configuration",
  "displayName": "opsi-configuration",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "DELETED",
  "opsiConfigType": "UX_CONFIGURATION",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "OpsiConfiguration",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "OpsiConfiguration",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "OpsiConfiguration",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.OpsiUxConfiguration, opsisdk.CreateOpsiUxConfigurationDetails, opsisdk.UpdateOpsiUxConfigurationDetails]{
		CollectionPath: "/20200630/opsiConfigurations", ItemPath: "/20200630/opsiConfigurations/<ocid:3>",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "opsiConfigType", "UX_CONFIGURATION", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "opsiConfigType", "UX_CONFIGURATION", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://opsi.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()}
	hooks := newOpsiConfigurationRuntimeHooksWithOCIClient(sdkClient)
	applyOpsiConfigurationRuntimeHooks(&hooks, sdkClient, nil)
	client := defaultOpsiConfigurationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.OpsiConfiguration](buildOpsiConfigurationGeneratedRuntimeConfig(&OpsiConfigurationServiceManager{}, hooks))}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.OpsiConfiguration]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.OpsiConfiguration) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.OpsiConfigType != resource.Spec.OpsiConfigType || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OpsiConfiguration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.OpsiConfiguration) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated opsi configuration"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.OpsiConfiguration) error {
			if !(current.Status.Description == "updated opsi configuration") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OpsiConfiguration status = %+v", current.Status)
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
