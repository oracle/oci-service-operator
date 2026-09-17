/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package autoscalingpolicy

import (
	"context"
	"fmt"
	"testing"

	autoscalingsdk "github.com/oracle/oci-go-sdk/v65/autoscaling"
	autoscalingv1beta1 "github.com/oracle/oci-service-operator/api/autoscaling/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationAutoScalingPolicyCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &autoscalingv1beta1.AutoScalingPolicy{}
	ocimock.InitializeResource(resource, "mock-autoscalingpolicy")
	resource.Spec = ocimock.MustJSONFixture[autoscalingv1beta1.AutoScalingPolicySpec](t, `{
  "autoScalingConfigurationId": "<ocid:1>",
  "policyType": "scheduled",
  "capacity": {
    "min": 1,
    "max": 3,
    "initial": 1
  },
  "displayName": "policy create",
  "isEnabled": false,
  "executionSchedule": {
    "type": "cron",
    "expression": "0 0 0 ? * * *",
    "timezone": "UTC"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"displayName":"policy updated"}`)
	createRequest := ocimock.MustJSONFixture[autoscalingsdk.CreateScheduledPolicyDetails](t, `{
  "capacity": {
    "min": 1,
    "max": 3,
    "initial": 1
  },
  "displayName": "policy create",
  "isEnabled": false,
  "executionSchedule": {
    "type": "cron",
    "expression": "0 0 0 ? * * *",
    "timezone": "UTC"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[autoscalingsdk.UpdateScheduledPolicyDetails](t, `{
  "displayName": "policy updated",
  "executionSchedule": {
    "type": "cron",
    "expression": "0 0 0 ? * * *",
    "timezone": "UTC"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[autoscalingsdk.ScheduledPolicy](t, `{
  "policyType": "scheduled",
  "capacity": {
    "min": 1,
    "max": 3,
    "initial": 1
  },
  "displayName": "policy create",
  "isEnabled": false,
  "executionSchedule": {
    "type": "cron",
    "expression": "0 0 0 ? * * *",
    "timezone": "UTC"
  },
  "id": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[autoscalingsdk.ScheduledPolicy](t, `{
  "policyType": "scheduled",
  "capacity": {
    "min": 1,
    "max": 3,
    "initial": 1
  },
  "displayName": "policy updated",
  "isEnabled": false,
  "executionSchedule": {
    "type": "cron",
    "expression": "0 0 0 ? * * *",
    "timezone": "UTC"
  },
  "id": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[autoscalingsdk.ScheduledPolicy, autoscalingsdk.CreateScheduledPolicyDetails, autoscalingsdk.UpdateScheduledPolicyDetails]{
		CollectionPath: "/20181001/autoScalingConfigurations/<ocid:1>/policies", ItemPath: "/20181001/autoScalingConfigurations/<ocid:1>/policies/resource-key",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeArray,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "policyType", "scheduled", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "policyType", "scheduled", updateRequest)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://autoscaling.mock.invalid", BasePath: "20181001", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := autoscalingsdk.AutoScalingClient{BaseClient: session.BaseClient()}
	manager := &AutoScalingPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAutoScalingPolicyRuntimeHooks(manager, sdkClient)
	client := wrapAutoScalingPolicyGeneratedClient(hooks, defaultAutoScalingPolicyServiceClient{ServiceClient: generatedruntime.NewServiceClient[*autoscalingv1beta1.AutoScalingPolicy](buildAutoScalingPolicyGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*autoscalingv1beta1.AutoScalingPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *autoscalingv1beta1.AutoScalingPolicy) error {
			if current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created AutoScalingPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *autoscalingv1beta1.AutoScalingPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *autoscalingv1beta1.AutoScalingPolicy) error {
			if current.Status.DisplayName != current.Spec.DisplayName {
				return fmt.Errorf("updated AutoScalingPolicy status = %+v", current.Status)
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
