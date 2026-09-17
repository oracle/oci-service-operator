/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package unifiedauditpolicy

import (
	"context"
	"encoding/json"
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
func TestMockIntegrationUnifiedAuditPolicyLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &datasafev1beta1.UnifiedAuditPolicy{}
	if err := json.Unmarshal([]byte(`{"metadata":{"name":"osok-mock-unifiedauditpolicy"},"spec":{"securityPolicyId":"ocid1.securitypolicy.oc1..synthetic","unifiedAuditPolicyDefinitionId":"ocid1.unifiedauditpolicydefinition.oc1..synthetic","compartmentId":"ocid1.compartment.oc1..synthetic","status":"DISABLED","conditions":[{"entitySelection":"ALL_USERS","operationStatus":"ALL"}],"displayName":"osok-mock-unified-audit-policy"}}`), resource); err != nil {
		t.Fatal(err)
	}
	ocimock.InitializeResource(resource, "mock-unifiedauditpolicy")
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "conditions": [
    {
      "JsonData": "eyJhdHRyaWJ1dGVTZXRJZCI6bnVsbCwiZW50aXR5U2VsZWN0aW9uIjoiQUxMX1VTRVJTIiwiZW50aXR5VHlwZSI6bnVsbCwianNvbkRhdGEiOm51bGwsIm9wZXJhdGlvblN0YXR1cyI6IkFMTCIsInJvbGVOYW1lcyI6bnVsbCwidXNlck5hbWVzIjpudWxsfQ==",
      "entitySelection": "ALL_USERS",
      "entityType": "",
      "operationStatus": "ALL"
    }
  ],
  "description": "mock-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateUnifiedAuditPolicyDetails](t, `{
  "compartmentId": "ocid1.compartment.oc1..synthetic",
  "conditions": [
    {
      "JsonData": "eyJhdHRyaWJ1dGVTZXRJZCI6bnVsbCwiZW50aXR5U2VsZWN0aW9uIjoiQUxMX1VTRVJTIiwiZW50aXR5VHlwZSI6bnVsbCwianNvbkRhdGEiOm51bGwsIm9wZXJhdGlvblN0YXR1cyI6IkFMTCIsInJvbGVOYW1lcyI6bnVsbCwidXNlck5hbWVzIjpudWxsfQ==",
      "entitySelection": "ALL_USERS",
      "entityType": "",
      "operationStatus": "ALL"
    }
  ],
  "displayName": "osok-mock-unified-audit-policy",
  "securityPolicyId": "ocid1.securitypolicy.oc1..synthetic",
  "status": "DISABLED",
  "unifiedAuditPolicyDefinitionId": "ocid1.unifiedauditpolicydefinition.oc1..synthetic"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.UnifiedAuditPolicy](t, `{
  "compartmentId": "ocid1.compartment.oc1..synthetic",
  "conditions": [
    {
      "entitySelection": "ALL_USERS",
      "operationStatus": "ALL"
    }
  ],
  "displayName": "osok-mock-unified-audit-policy",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "securityPolicyId": "ocid1.securitypolicy.oc1..synthetic",
  "status": "DISABLED",
  "unifiedAuditPolicyDefinitionId": "ocid1.unifiedauditpolicydefinition.oc1..synthetic"
}`)
	createdReadStates := []datasafesdk.UnifiedAuditPolicy{
		ocimock.MustOCIResponseFixture[datasafesdk.UnifiedAuditPolicy](t, `{
  "compartmentId": "ocid1.compartment.oc1..synthetic",
  "conditions": [
    {
      "entitySelection": "ALL_USERS",
      "operationStatus": "ALL"
    }
  ],
  "displayName": "osok-mock-unified-audit-policy",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "securityPolicyId": "ocid1.securitypolicy.oc1..synthetic",
  "status": "DISABLED",
  "unifiedAuditPolicyDefinitionId": "ocid1.unifiedauditpolicydefinition.oc1..synthetic"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateUnifiedAuditPolicyDetails](t, `{
  "conditions": [
    {
      "JsonData": "eyJhdHRyaWJ1dGVTZXRJZCI6bnVsbCwiZW50aXR5U2VsZWN0aW9uIjoiQUxMX1VTRVJTIiwiZW50aXR5VHlwZSI6bnVsbCwianNvbkRhdGEiOm51bGwsIm9wZXJhdGlvblN0YXR1cyI6IkFMTCIsInJvbGVOYW1lcyI6bnVsbCwidXNlck5hbWVzIjpudWxsfQ==",
      "entitySelection": "ALL_USERS",
      "entityType": "",
      "operationStatus": "ALL"
    }
  ],
  "description": "mock-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.UnifiedAuditPolicy](t, `{
  "compartmentId": "ocid1.compartment.oc1..synthetic",
  "conditions": [
    {
      "JsonData": "eyJhdHRyaWJ1dGVTZXRJZCI6bnVsbCwiZW50aXR5U2VsZWN0aW9uIjoiQUxMX1VTRVJTIiwiZW50aXR5VHlwZSI6bnVsbCwianNvbkRhdGEiOm51bGwsIm9wZXJhdGlvblN0YXR1cyI6IkFMTCIsInJvbGVOYW1lcyI6bnVsbCwidXNlck5hbWVzIjpudWxsfQ==",
      "entitySelection": "ALL_USERS",
      "entityType": "",
      "operationStatus": "ALL"
    }
  ],
  "description": "mock-updated",
  "displayName": "osok-mock-unified-audit-policy",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "securityPolicyId": "ocid1.securitypolicy.oc1..synthetic",
  "status": "DISABLED",
  "unifiedAuditPolicyDefinitionId": "ocid1.unifiedauditpolicydefinition.oc1..synthetic"
}`)
	updatedReadStates := []datasafesdk.UnifiedAuditPolicy{
		ocimock.MustOCIResponseFixture[datasafesdk.UnifiedAuditPolicy](t, `{
  "compartmentId": "ocid1.compartment.oc1..synthetic",
  "conditions": [
    {
      "JsonData": "eyJhdHRyaWJ1dGVTZXRJZCI6bnVsbCwiZW50aXR5U2VsZWN0aW9uIjoiQUxMX1VTRVJTIiwiZW50aXR5VHlwZSI6bnVsbCwianNvbkRhdGEiOm51bGwsIm9wZXJhdGlvblN0YXR1cyI6IkFMTCIsInJvbGVOYW1lcyI6bnVsbCwidXNlck5hbWVzIjpudWxsfQ==",
      "entitySelection": "ALL_USERS",
      "entityType": "",
      "operationStatus": "ALL"
    }
  ],
  "description": "mock-updated",
  "displayName": "osok-mock-unified-audit-policy",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "securityPolicyId": "ocid1.securitypolicy.oc1..synthetic",
  "status": "DISABLED",
  "unifiedAuditPolicyDefinitionId": "ocid1.unifiedauditpolicydefinition.oc1..synthetic"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.UnifiedAuditPolicy,
		datasafesdk.CreateUnifiedAuditPolicyDetails,
		datasafesdk.UpdateUnifiedAuditPolicyDetails,
	]{
		CollectionPath: "/20181201/unifiedAuditPolicies",
		ItemPath:       "/20181201/unifiedAuditPolicies/<ocid:4>",
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:  &createRequest,
		CompareCreate: func(actual, expected datasafesdk.CreateUnifiedAuditPolicyDetails) error {
			if len(actual.Conditions) != len(expected.Conditions) {
				return fmt.Errorf("create policy conditions = %d, want %d", len(actual.Conditions), len(expected.Conditions))
			}
			for index := range actual.Conditions {
				if actual.Conditions[index].GetEntitySelection() != expected.Conditions[index].GetEntitySelection() ||
					actual.Conditions[index].GetOperationStatus() != expected.Conditions[index].GetOperationStatus() {
					return fmt.Errorf("create policy condition %d differs", index)
				}
			}
			actual.Conditions = nil
			expected.Conditions = nil
			return ocimock.CompareJSONValues(actual, expected)
		},
		CreatedState:  &createdState,
		ListShape:     ocimock.ListShapeItems,
		UpdateRequest: &updateRequest,
		CompareUpdate: func(actual, expected datasafesdk.UpdateUnifiedAuditPolicyDetails) error {
			if len(actual.Conditions) != len(expected.Conditions) {
				return fmt.Errorf("update policy conditions = %d, want %d", len(actual.Conditions), len(expected.Conditions))
			}
			for index := range actual.Conditions {
				if actual.Conditions[index].GetEntitySelection() != expected.Conditions[index].GetEntitySelection() ||
					actual.Conditions[index].GetOperationStatus() != expected.Conditions[index].GetOperationStatus() {
					return fmt.Errorf("update policy condition %d differs", index)
				}
			}
			actual.Conditions = nil
			expected.Conditions = nil
			return ocimock.CompareJSONValues(actual, expected)
		},
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateUnifiedAuditPolicyDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.UnifiedAuditPolicy) error {
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
			t.Errorf("close UnifiedAuditPolicy OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &UnifiedAuditPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newUnifiedAuditPolicyRuntimeHooks(manager, sdkClient)
	client := wrapUnifiedAuditPolicyGeneratedClient(hooks, defaultUnifiedAuditPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.UnifiedAuditPolicy](buildUnifiedAuditPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.UnifiedAuditPolicy]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.UnifiedAuditPolicy) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.SecurityPolicyId, current.Spec.SecurityPolicyId) ||
				!reflect.DeepEqual(current.Status.Status, current.Spec.Status) ||
				!reflect.DeepEqual(current.Status.UnifiedAuditPolicyDefinitionId, current.Spec.UnifiedAuditPolicyDefinitionId) ||
				len(current.Status.Conditions) != 1 ||
				current.Status.Conditions[0].EntitySelection != "ALL_USERS" ||
				current.Status.Conditions[0].OperationStatus != "ALL" {
				return fmt.Errorf("created UnifiedAuditPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.UnifiedAuditPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.UnifiedAuditPolicy) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				len(current.Status.Conditions) != 1 ||
				current.Status.Conditions[0].EntitySelection != "ALL_USERS" ||
				current.Status.Conditions[0].OperationStatus != "ALL" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated UnifiedAuditPolicy status = %+v", current.Status)
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
