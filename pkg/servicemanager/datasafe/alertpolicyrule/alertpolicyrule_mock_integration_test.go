/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package alertpolicyrule

import (
	"context"
	"fmt"
	"testing"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

func TestMockIntegrationAlertPolicyRuleCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := testAlertPolicyRule()
	ocimock.InitializeResource(resource, "mock-alert-policy-rule")
	resource.Annotations[alertPolicyRuleAlertPolicyIDAnnotation] = "<ocid:1>"
	updatedSpec := resource.Spec
	updatedSpec.Expression = "severity == 'CRITICAL'"
	updatedSpec.Description = "updated description"
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateAlertPolicyRuleDetails](t, `{
  "expression":"severity == 'HIGH'","description":"initial","displayName":"high severity"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.AlertPolicyRule](t, `{
  "key":"rule-1","expression":"severity == 'HIGH'","description":"initial",
  "displayName":"high severity","lifecycleState":"ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateAlertPolicyRuleDetails](t, `{
  "expression":"severity == 'CRITICAL'","description":"updated description"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.AlertPolicyRule](t, `{
  "key":"rule-1","expression":"severity == 'CRITICAL'","description":"updated description",
  "displayName":"high severity","lifecycleState":"ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.AlertPolicyRule, datasafesdk.CreateAlertPolicyRuleDetails, datasafesdk.UpdateAlertPolicyRuleDetails]{
		CollectionPath: "/20181201/alertPolicies/<ocid:1>/rules", ItemPath: "/20181201/alertPolicies/<ocid:1>/rules/rule-1",
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 200, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newAlertPolicyRuleServiceClientWithOCIClient(loggerutil.OSOKLogger{}, datasafesdk.DataSafeClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.AlertPolicyRule]{
		Resource: resource, Client: client,
		ValidateCreated: func(current *datasafev1beta1.AlertPolicyRule) error {
			if current.Status.Key != "rule-1" || current.Status.OsokStatus.Ocid == "" || current.Status.Expression != current.Spec.Expression || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created AlertPolicyRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.AlertPolicyRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.AlertPolicyRule) error {
			if current.Status.Expression != current.Spec.Expression || current.Status.Description != current.Spec.Description || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("updated AlertPolicyRule status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*datasafev1beta1.AlertPolicyRule) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable AlertPolicyRule update calls = %d, want 1", got)
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
