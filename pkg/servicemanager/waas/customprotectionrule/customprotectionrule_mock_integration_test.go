/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package customprotectionrule

import (
	"context"
	"fmt"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationCustomProtectionRuleEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &waasv1beta1.CustomProtectionRule{}
	ocimock.InitializeResource(resource, "mock-customprotectionrule")
	resource.Spec = ocimock.MustJSONFixture[waasv1beta1.CustomProtectionRuleSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "blocks example input",
  "displayName": "osok-mock-custom-protection-rule-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "template": "SecRule REQUEST_HEADERS \"example\" \"id: {{id_1}}, ctl:ruleEngine={{mode}}\""
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[waassdk.CreateCustomProtectionRuleDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "blocks example input",
  "displayName": "osok-mock-custom-protection-rule-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "template": "SecRule REQUEST_HEADERS \"example\" \"id: {{id_1}}, ctl:ruleEngine={{mode}}\""
}`)
	createdState := ocimock.MustOCIResponseFixture[waassdk.CustomProtectionRule](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:13:30.912Z"
    }
  },
  "description": "blocks example input",
  "displayName": "osok-mock-custom-protection-rule-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "modSecurityRuleIds": [
    "776987"
  ],
  "template": "SecRule REQUEST_HEADERS \"example\" \"id: {{id_1}}, ctl:ruleEngine={{mode}}\"",
  "timeCreated": "2026-09-01T23:13:31.078Z"
}`)
	updateRequest := ocimock.MustJSONFixture[waassdk.UpdateCustomProtectionRuleDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[waassdk.CustomProtectionRule](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:13:30.912Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-custom-protection-rule-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "modSecurityRuleIds": [
    "776987"
  ],
  "template": "SecRule REQUEST_HEADERS \"example\" \"id: {{id_1}}, ctl:ruleEngine={{mode}}\"",
  "timeCreated": "2026-09-01T23:13:31.078Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[waassdk.CustomProtectionRule](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:13:30.912Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-custom-protection-rule-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETED",
  "modSecurityRuleIds": [
    "776987"
  ],
  "template": "SecRule REQUEST_HEADERS \"example\" \"id: {{id_1}}, ctl:ruleEngine={{mode}}\"",
  "timeCreated": "2026-09-01T23:13:31.078Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		waassdk.CustomProtectionRule,
		waassdk.CreateCustomProtectionRuleDetails,
		waassdk.UpdateCustomProtectionRuleDetails,
	]{
		CollectionPath:    "/20181116/customProtectionRules",
		ItemPath:          "/20181116/customProtectionRules/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ waassdk.CreateCustomProtectionRuleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ waassdk.CustomProtectionRule) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181116", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close CustomProtectionRule OCI mock: %v", err)
		}
	})
	sdkClient := waassdk.WaasClient{BaseClient: session.BaseClient()}
	hooks := newCustomProtectionRuleDefaultRuntimeHooks(sdkClient)
	applyCustomProtectionRuleRuntimeHooks(&hooks)
	manager := &CustomProtectionRuleServiceManager{Log: loggerutil.OSOKLogger{}}
	client := wrapCustomProtectionRuleGeneratedClient(hooks, defaultCustomProtectionRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.CustomProtectionRule](buildCustomProtectionRuleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waasv1beta1.CustomProtectionRule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waasv1beta1.CustomProtectionRule) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Template, current.Spec.Template) {
				return fmt.Errorf("created CustomProtectionRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *waasv1beta1.CustomProtectionRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *waasv1beta1.CustomProtectionRule) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated CustomProtectionRule status = %+v", current.Status)
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
