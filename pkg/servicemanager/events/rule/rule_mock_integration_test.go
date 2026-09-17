/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package rule

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	eventssdk "github.com/oracle/oci-go-sdk/v65/events"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

const mockRuleID = "ocid1.rule.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/events/rule and formal/imports/events/rule.json
//   - resource runtime: rule_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/events
func TestMockIntegrationRuleExplicitCRUD(t *testing.T) {
	t.Parallel()

	resource := newTestRule()
	ocimock.InitializeResource(resource, "mock-rule")
	updated := resource.DeepCopy()
	updated.Spec.DisplayName = "mock-rule-updated"
	updated.Spec.Description = "updated rule description"
	updated.Spec.FreeformTags = map[string]string{"env": "updated"}

	createRequest := ocimock.MustJSONFixture[eventssdk.CreateRuleDetails](t, `{
		"displayName":"test-rule",
		"isEnabled":true,
		"condition":"{\"eventType\":\"com.oraclecloud.objectstorage.createbucket\"}",
		"compartmentId":"ocid1.compartment.oc1..rule",
		"actions":{"actions":[{"actionType":"ONS","isEnabled":true,"description":"notify","topicId":"ocid1.onstopic.oc1..topic"}]},
		"description":"rule description",
		"freeformTags":{"env":"test"},
		"definedTags":{"Operations":{"CostCenter":"42"}}
	}`)
	createdState := sdkRuleFromResource(resource, mockRuleID, eventssdk.RuleLifecycleStateInactive)
	updateRequest := ocimock.MustJSONFixture[eventssdk.UpdateRuleDetails](t, `{
		"displayName":"mock-rule-updated",
		"description":"updated rule description",
		"freeformTags":{"env":"updated"}
	}`)
	updatedState := sdkRuleFromResource(updated, mockRuleID, eventssdk.RuleLifecycleStateInactive)
	deletedState := updatedState
	deletedState.Actions = &eventssdk.ActionList{Actions: []eventssdk.Action{}}
	deletedState.LifecycleState = eventssdk.RuleLifecycleStateDeleted

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		eventssdk.Rule,
		eventssdk.CreateRuleDetails,
		eventssdk.UpdateRuleDetails,
	]{
		CollectionPath: "/20181201/rules",
		ItemPath:       "/20181201/rules/" + mockRuleID,
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		DeleteStatus:      http.StatusNoContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://events.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Rule OCI mock: %v", err)
		}
	})

	client := newMockRuleClient(eventssdk.EventsClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*eventsv1beta1.Rule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *eventsv1beta1.Rule) error {
			if current.Status.Id != mockRuleID ||
				current.Status.DisplayName != resource.Spec.DisplayName ||
				current.Status.Description != resource.Spec.Description ||
				current.Status.LifecycleState != string(eventssdk.RuleLifecycleStateInactive) {
				return fmt.Errorf("created Rule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *eventsv1beta1.Rule) {
			current.Spec = updated.Spec
		},
		ValidateUpdated: func(current *eventsv1beta1.Rule) error {
			if current.Status.Id != mockRuleID ||
				current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.Description != current.Spec.Description ||
				current.Status.FreeformTags["env"] != "updated" ||
				current.Status.LifecycleState != string(eventssdk.RuleLifecycleStateInactive) {
				return fmt.Errorf("updated Rule status = %+v", current.Status)
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
