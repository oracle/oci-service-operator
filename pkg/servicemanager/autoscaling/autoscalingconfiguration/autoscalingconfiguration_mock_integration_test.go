/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package autoscalingconfiguration

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	autoscalingsdk "github.com/oracle/oci-go-sdk/v65/autoscaling"
	"github.com/oracle/oci-go-sdk/v65/common"
	autoscalingv1beta1 "github.com/oracle/oci-service-operator/api/autoscaling/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	mockAutoScalingConfigurationID = "ocid1.autoscalingconfiguration.oc1..mock"
	mockInstancePoolID             = "ocid1.instancepool.oc1..mock"
)

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/autoscaling/autoscalingconfiguration.json
//   - repo-authored runtime: formal/controllers/autoscaling/autoscalingconfiguration/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/autoscaling/autoscaling_auto_scaling_configuration_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/autoscaling
//
// Earlier contract review exposed a stale empty PUT caused by missing formal
// semantics. This dynamic lifecycle proves that a converged read does
// not update while a requested mutable change still exercises the real SDK PUT.
func TestMockIntegrationAutoScalingConfigurationSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := newMockAutoScalingConfigurationResource()
	responder, err := newAutoScalingConfigurationMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host:      "https://autoscaling.mock.invalid",
		BasePath:  "20181001",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close AutoScalingConfiguration OCI mock: %v", err)
		}
	})

	client := newMockAutoScalingConfigurationClient(session)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*autoscalingv1beta1.AutoScalingConfiguration]{
		Resource: resource,
		Client:   client,
		ValidateCreated: func(current *autoscalingv1beta1.AutoScalingConfiguration) error {
			if current.Status.Id != mockAutoScalingConfigurationID ||
				current.Status.DisplayName != "mock-autoscaling" ||
				current.Status.IsEnabled ||
				current.Status.Resource.Id != mockInstancePoolID ||
				len(current.Status.Policies) != 1 ||
				current.Status.Policies[0].PolicyType != "scheduled" ||
				current.Status.Policies[0].IsEnabled == nil ||
				*current.Status.Policies[0].IsEnabled {
				return fmt.Errorf("created AutoScalingConfiguration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *autoscalingv1beta1.AutoScalingConfiguration) {
			current.Spec.DisplayName = "mock-autoscaling-updated"
			current.Spec.CoolDownInSeconds = 300
			current.Spec.IsEnabled = common.Bool(false)
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *autoscalingv1beta1.AutoScalingConfiguration) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.CoolDownInSeconds != current.Spec.CoolDownInSeconds ||
				current.Status.IsEnabled ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated AutoScalingConfiguration status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if counts := responder.OperationCounts(); counts[ocimock.OperationUpdate] != 1 {
		t.Fatalf("update operations = %d, want exactly one requested update", counts[ocimock.OperationUpdate])
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newMockAutoScalingConfigurationResource() *autoscalingv1beta1.AutoScalingConfiguration {
	return &autoscalingv1beta1.AutoScalingConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mock-autoscaling",
			Namespace: "default",
			UID:       types.UID("mock-autoscaling-uid"),
		},
		Spec: autoscalingv1beta1.AutoScalingConfigurationSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			DisplayName:   "mock-autoscaling",
			IsEnabled:     common.Bool(false),
			FreeformTags:  map[string]string{"osok-mock": "create"},
			Resource: autoscalingv1beta1.AutoScalingConfigurationResource{
				Id:   mockInstancePoolID,
				Type: "instancePool",
			},
			Policies: []autoscalingv1beta1.AutoScalingConfigurationPolicy{{
				PolicyType:  "scheduled",
				DisplayName: "nightly-stop",
				IsEnabled:   common.Bool(false),
				Capacity: autoscalingv1beta1.AutoScalingConfigurationPolicyCapacity{
					Min: 1, Max: 1, Initial: 1,
				},
				ExecutionSchedule: autoscalingv1beta1.AutoScalingConfigurationPolicyExecutionSchedule{
					Type:       "cron",
					Expression: "0 0 0 ? * * *",
					Timezone:   "UTC",
				},
			}},
		},
	}
}

func newMockAutoScalingConfigurationClient(
	session *ocimock.Session,
) AutoScalingConfigurationServiceClient {
	sdkClient := autoscalingsdk.AutoScalingClient{BaseClient: session.BaseClient()}
	manager := &AutoScalingConfigurationServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
	}
	hooks := newAutoScalingConfigurationRuntimeHooks(manager, sdkClient)
	return wrapAutoScalingConfigurationGeneratedClient(hooks, defaultAutoScalingConfigurationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*autoscalingv1beta1.AutoScalingConfiguration](
			buildAutoScalingConfigurationGeneratedRuntimeConfig(manager, hooks),
		),
	})
}

func newAutoScalingConfigurationMockResponder(
	resource *autoscalingv1beta1.AutoScalingConfiguration,
) (*ocimock.CRUDResponder[autoscalingsdk.AutoScalingConfiguration], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[autoscalingsdk.AutoScalingConfiguration]{
		CollectionPath:     "/20181001/autoScalingConfigurations",
		ItemPath:           "/20181001/autoScalingConfigurations/" + mockAutoScalingConfigurationID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(request ocimock.Request, present bool, state autoscalingsdk.AutoScalingConfiguration) (ocimock.Response, error) {
			if got := request.URL.Query().Get("compartmentId"); got != resource.Spec.CompartmentId {
				return ocimock.Response{}, fmt.Errorf("list compartmentId = %q", got)
			}
			if got := request.URL.Query().Get("displayName"); got != resource.Spec.DisplayName {
				return ocimock.Response{}, fmt.Errorf("list displayName = %q", got)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []autoscalingsdk.AutoScalingConfigurationSummary{})
			}
			return ocimock.JSONResponse(http.StatusOK, []autoscalingsdk.AutoScalingConfigurationSummary{
				autoScalingConfigurationSummaryFromMockState(state),
			})
		},
		Create: func(request ocimock.Request) (autoscalingsdk.AutoScalingConfiguration, ocimock.Response, error) {
			var details autoscalingsdk.CreateAutoScalingConfigurationDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return autoscalingsdk.AutoScalingConfiguration{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return autoscalingsdk.AutoScalingConfiguration{}, ocimock.Response{}, err
			}
			expected := mockCreateAutoScalingConfigurationDetails(resource)
			if !reflect.DeepEqual(details, expected) {
				return autoscalingsdk.AutoScalingConfiguration{}, ocimock.Response{}, fmt.Errorf("create AutoScalingConfiguration details = %+v, want %+v", details, expected)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return autoscalingsdk.AutoScalingConfiguration{}, ocimock.Response{}, fmt.Errorf("create AutoScalingConfiguration opc-retry-token is empty")
			}
			state := autoscalingsdk.AutoScalingConfiguration{
				Id:                common.String(mockAutoScalingConfigurationID),
				CompartmentId:     details.CompartmentId,
				Resource:          details.Resource,
				Policies:          mockObservedAutoScalingPolicies(details.Policies, createdAt),
				TimeCreated:       &createdAt,
				DisplayName:       details.DisplayName,
				FreeformTags:      details.FreeformTags,
				CoolDownInSeconds: details.CoolDownInSeconds,
				IsEnabled:         details.IsEnabled,
				MaxResourceCount:  common.Int(10),
				MinResourceCount:  common.Int(1),
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state autoscalingsdk.AutoScalingConfiguration) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state autoscalingsdk.AutoScalingConfiguration) (autoscalingsdk.AutoScalingConfiguration, ocimock.Response, error) {
			var details autoscalingsdk.UpdateAutoScalingConfigurationDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return autoscalingsdk.AutoScalingConfiguration{}, ocimock.Response{}, err
			}
			expected := autoscalingsdk.UpdateAutoScalingConfigurationDetails{
				DisplayName:       common.String("mock-autoscaling-updated"),
				FreeformTags:      map[string]string{"osok-mock": "update"},
				IsEnabled:         common.Bool(false),
				CoolDownInSeconds: common.Int(300),
			}
			if !reflect.DeepEqual(details, expected) {
				return autoscalingsdk.AutoScalingConfiguration{}, ocimock.Response{}, fmt.Errorf("update AutoScalingConfiguration details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.IsEnabled = details.IsEnabled
			state.CoolDownInSeconds = details.CoolDownInSeconds
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ autoscalingsdk.AutoScalingConfiguration) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete AutoScalingConfiguration body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

func mockCreateAutoScalingConfigurationDetails(
	resource *autoscalingv1beta1.AutoScalingConfiguration,
) autoscalingsdk.CreateAutoScalingConfigurationDetails {
	return autoscalingsdk.CreateAutoScalingConfigurationDetails{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		DisplayName:   common.String(resource.Spec.DisplayName),
		IsEnabled:     common.Bool(false),
		FreeformTags:  map[string]string{"osok-mock": "create"},
		Resource: autoscalingsdk.InstancePoolResource{
			Id: common.String(mockInstancePoolID),
		},
		Policies: []autoscalingsdk.CreateAutoScalingPolicyDetails{
			autoscalingsdk.CreateScheduledPolicyDetails{
				Capacity: &autoscalingsdk.Capacity{
					Min: common.Int(1), Max: common.Int(1), Initial: common.Int(1),
				},
				ExecutionSchedule: autoscalingsdk.CronExecutionSchedule{
					Expression: common.String("0 0 0 ? * * *"),
					Timezone:   autoscalingsdk.ExecutionScheduleTimezoneUtc,
				},
				DisplayName: common.String("nightly-stop"),
				IsEnabled:   common.Bool(false),
			},
		},
	}
}

func mockObservedAutoScalingPolicies(
	policies []autoscalingsdk.CreateAutoScalingPolicyDetails,
	createdAt common.SDKTime,
) []autoscalingsdk.AutoScalingPolicy {
	observed := make([]autoscalingsdk.AutoScalingPolicy, 0, len(policies))
	for index, policy := range policies {
		scheduled := policy.(autoscalingsdk.CreateScheduledPolicyDetails)
		observed = append(observed, autoscalingsdk.ScheduledPolicy{
			Id:                common.String(fmt.Sprintf("ocid1.autoscalingpolicy.oc1..mock%d", index+1)),
			TimeCreated:       &createdAt,
			ExecutionSchedule: scheduled.ExecutionSchedule,
			Capacity:          scheduled.Capacity,
			DisplayName:       scheduled.DisplayName,
			IsEnabled:         scheduled.IsEnabled,
			ResourceAction:    scheduled.ResourceAction,
		})
	}
	return observed
}

func autoScalingConfigurationSummaryFromMockState(
	state autoscalingsdk.AutoScalingConfiguration,
) autoscalingsdk.AutoScalingConfigurationSummary {
	return autoscalingsdk.AutoScalingConfigurationSummary{
		Id:                state.Id,
		CompartmentId:     state.CompartmentId,
		Resource:          state.Resource,
		TimeCreated:       state.TimeCreated,
		DisplayName:       state.DisplayName,
		CoolDownInSeconds: state.CoolDownInSeconds,
		IsEnabled:         state.IsEnabled,
		DefinedTags:       state.DefinedTags,
		FreeformTags:      state.FreeformTags,
	}
}
