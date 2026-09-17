/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package approvaltemplate

import (
	"context"
	"fmt"
	lockboxsdk "github.com/oracle/oci-go-sdk/v65/lockbox"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationApprovalTemplateEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &lockboxv1beta1.ApprovalTemplate{}
	ocimock.InitializeResource(resource, "mock-approvaltemplate")
	resource.Spec = ocimock.MustJSONFixture[lockboxv1beta1.ApprovalTemplateSpec](t, `{
  "autoApprovalState": "ENABLED",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-approval-template-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-approval-template-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[lockboxsdk.CreateApprovalTemplateDetails](t, `{
  "autoApprovalState": "ENABLED",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-approval-template-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[lockboxsdk.ApprovalTemplate](t, `{
  "approverLevels": null,
  "autoApprovalState": "ENABLED",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:51:03.105Z"
    }
  },
  "displayName": "osok-mock-approval-template-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-03T17:51:03.250Z",
  "timeUpdated": "2026-09-03T17:51:03.250Z"
}`)
	updateRequest := ocimock.MustJSONFixture[lockboxsdk.UpdateApprovalTemplateDetails](t, `{
  "displayName": "osok-mock-approval-template-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[lockboxsdk.ApprovalTemplate](t, `{
  "approverLevels": null,
  "autoApprovalState": "ENABLED",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:51:03.105Z"
    }
  },
  "displayName": "osok-mock-approval-template-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-03T17:51:03.250Z",
  "timeUpdated": "2026-09-03T17:51:04.059Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[lockboxsdk.ApprovalTemplate](t, `{
  "approverLevels": null,
  "autoApprovalState": "ENABLED",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:51:03.105Z"
    }
  },
  "displayName": "osok-mock-approval-template-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-03T17:51:03.250Z",
  "timeUpdated": "2026-09-03T17:51:05.267Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		lockboxsdk.ApprovalTemplate,
		lockboxsdk.CreateApprovalTemplateDetails,
		lockboxsdk.UpdateApprovalTemplateDetails,
	]{
		CollectionPath:    "/20220126/approvalTemplates",
		ItemPath:          "/20220126/approvalTemplates/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ lockboxsdk.CreateApprovalTemplateDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ lockboxsdk.ApprovalTemplate) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220126", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ApprovalTemplate OCI mock: %v", err)
		}
	})
	sdkClient := lockboxsdk.LockboxClient{BaseClient: session.BaseClient()}
	manager := &ApprovalTemplateServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newApprovalTemplateRuntimeHooksWithOCIClient(sdkClient)
	applyApprovalTemplateRuntimeHooks(manager, &hooks)
	client := wrapApprovalTemplateGeneratedClient(hooks, defaultApprovalTemplateServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*lockboxv1beta1.ApprovalTemplate](buildApprovalTemplateGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*lockboxv1beta1.ApprovalTemplate]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *lockboxv1beta1.ApprovalTemplate) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AutoApprovalState, current.Spec.AutoApprovalState) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created ApprovalTemplate status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *lockboxv1beta1.ApprovalTemplate) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *lockboxv1beta1.ApprovalTemplate) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated ApprovalTemplate status = %+v", current.Status)
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
