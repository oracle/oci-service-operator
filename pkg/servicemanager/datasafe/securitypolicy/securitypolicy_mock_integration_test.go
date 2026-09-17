/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securitypolicy

import (
	"context"
	"fmt"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationSecurityPolicyEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.SecurityPolicy{}
	ocimock.InitializeResource(resource, "mock-securitypolicy")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.SecurityPolicySpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSecurityPolicyDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:19:24.520Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy is active",
  "lifecycleState": "ACTIVE",
  "securityPolicyType": "DATASAFE_MANAGED",
  "timeCreated": "2026-09-03T17:19:24.580Z",
  "timeUpdated": "2026-09-03T17:19:33.053Z"
}`)
	createdReadStates := []datasafesdk.SecurityPolicy{
		ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:19:24.520Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy is active",
  "lifecycleState": "ACTIVE",
  "securityPolicyType": "DATASAFE_MANAGED",
  "timeCreated": "2026-09-03T17:19:24.580Z",
  "timeUpdated": "2026-09-03T17:19:33.053Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSecurityPolicyDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:19:24.520Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy is active",
  "lifecycleState": "ACTIVE",
  "securityPolicyType": "DATASAFE_MANAGED",
  "timeCreated": "2026-09-03T17:19:24.580Z",
  "timeUpdated": "2026-09-03T17:19:41.793Z"
}`)
	updatedReadStates := []datasafesdk.SecurityPolicy{
		ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:19:24.520Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy is active",
  "lifecycleState": "ACTIVE",
  "securityPolicyType": "DATASAFE_MANAGED",
  "timeCreated": "2026-09-03T17:19:24.580Z",
  "timeUpdated": "2026-09-03T17:19:41.793Z"
}`),
	}
	deletedReadStates := []datasafesdk.SecurityPolicy{
		ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:19:24.520Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Deleting the security policy resource",
  "lifecycleState": "DELETING",
  "securityPolicyType": "DATASAFE_MANAGED",
  "timeCreated": "2026-09-03T17:19:24.580Z",
  "timeUpdated": "2026-09-03T17:19:43.530Z"
}`),
		ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:19:24.520Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-security-policy-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy is deleted",
  "lifecycleState": "DELETED",
  "securityPolicyType": "DATASAFE_MANAGED",
  "timeCreated": "2026-09-03T17:19:24.580Z",
  "timeUpdated": "2026-09-03T17:19:47.288Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.SecurityPolicy,
		datasafesdk.CreateSecurityPolicyDetails,
		datasafesdk.UpdateSecurityPolicyDetails,
	]{
		CollectionPath:    "/20181201/securityPolicies",
		ItemPath:          "/20181201/securityPolicies/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      202,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSecurityPolicyDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.SecurityPolicy) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SecurityPolicy OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &SecurityPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSecurityPolicyRuntimeHooksWithOCIClient(sdkClient)
	applySecurityPolicyRuntimeHooks(&hooks)
	client := wrapSecurityPolicyGeneratedClient(hooks, defaultSecurityPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SecurityPolicy](buildSecurityPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SecurityPolicy]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SecurityPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created SecurityPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SecurityPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.SecurityPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated SecurityPolicy status = %+v", current.Status)
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
