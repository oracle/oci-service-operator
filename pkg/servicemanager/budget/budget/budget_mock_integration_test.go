/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package budget

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	budgetsdk "github.com/oracle/oci-go-sdk/v65/budget"
	"github.com/oracle/oci-go-sdk/v65/common"
	budgetv1beta1 "github.com/oracle/oci-service-operator/api/budget/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockBudgetID = "ocid1.budget.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/budget/budget.json
//   - repo-authored runtime: formal/controllers/budget/budget/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/budget/budget_budget_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/budget
func TestMockIntegrationBudgetSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := &budgetv1beta1.Budget{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-budget", Namespace: "default", UID: types.UID("mock-budget-uid")},
		Spec: budgetv1beta1.BudgetSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			Amount:        100,
			ResetPeriod:   string(budgetsdk.ResetPeriodMonthly),
			DisplayName:   "mock-budget",
		},
	}
	responder, err := newBudgetMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host:      "https://usage.mock.invalid",
		BasePath:  "20190111",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Budget OCI mock: %v", err)
		}
	})

	sdkClient := budgetsdk.BudgetClient{BaseClient: session.BaseClient()}
	hooks := newBudgetDefaultRuntimeHooks(sdkClient)
	applyBudgetRuntimeHooks(&hooks)
	client := defaultBudgetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*budgetv1beta1.Budget](
		buildBudgetGeneratedRuntimeConfig(&BudgetServiceManager{
			Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		}, hooks),
	)}

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*budgetv1beta1.Budget]{
		Resource: resource,
		Client:   client,
		ValidateCreated: func(current *budgetv1beta1.Budget) error {
			if current.Status.Id != mockBudgetID ||
				current.Status.DisplayName != "mock-budget" ||
				current.Status.LifecycleState != string(budgetsdk.LifecycleStateActive) ||
				current.Status.Amount != 100 ||
				current.Status.Version != 1 {
				return fmt.Errorf("created Budget status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *budgetv1beta1.Budget) {
			current.Spec.DisplayName = "mock-budget-updated"
		},
		ValidateUpdated: func(current *budgetv1beta1.Budget) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.LifecycleState != string(budgetsdk.LifecycleStateActive) ||
				current.Status.Version != 2 {
				return fmt.Errorf("updated Budget status = %+v", current.Status)
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

func TestMockIntegrationBudgetRejectsForceNewCompartmentDrift(t *testing.T) {
	t.Parallel()

	currentSpec := budgetv1beta1.BudgetSpec{
		CompartmentId: "ocid1.compartment.oc1..original",
		Amount:        100,
		ResetPeriod:   string(budgetsdk.ResetPeriodMonthly),
		DisplayName:   "mock-budget",
	}
	state := newMockBudgetState(mockBudgetID, currentSpec, 1, time.Unix(0, 0).UTC())
	responder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[budgetsdk.Budget]{
		CollectionPath:     "/20190111/budgets",
		ItemPath:           "/20190111/budgets/" + mockBudgetID,
		InitialState:       &state,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead},
		Read: func(_ ocimock.Request, current budgetsdk.Budget) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, current)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://usage.mock.invalid", BasePath: "20190111", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	sdkClient := budgetsdk.BudgetClient{BaseClient: session.BaseClient()}
	hooks := newBudgetDefaultRuntimeHooks(sdkClient)
	applyBudgetRuntimeHooks(&hooks)
	client := defaultBudgetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*budgetv1beta1.Budget](
		buildBudgetGeneratedRuntimeConfig(&BudgetServiceManager{}, hooks),
	)}
	resource := &budgetv1beta1.Budget{
		Spec: currentSpec,
		Status: budgetv1beta1.BudgetStatus{
			Id: mockBudgetID,
		},
	}
	resource.Status.OsokStatus.Ocid = mockBudgetID
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..changed"

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new compartmentId rejection", err)
	}
	if counts := responder.OperationCounts(); counts[ocimock.OperationUpdate] != 0 {
		t.Fatalf("update operations = %d, want 0", counts[ocimock.OperationUpdate])
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newBudgetMockResponder(resource *budgetv1beta1.Budget) (*ocimock.CRUDResponder[budgetsdk.Budget], error) {
	createdAt := common.SDKTime{Time: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: time.Date(2024, time.January, 1, 0, 1, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[budgetsdk.Budget]{
		CollectionPath:     "/20190111/budgets",
		ItemPath:           "/20190111/budgets/" + mockBudgetID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(request ocimock.Request, present bool, state budgetsdk.Budget) (ocimock.Response, error) {
			if got := request.URL.Query().Get("compartmentId"); got != resource.Spec.CompartmentId {
				return ocimock.Response{}, fmt.Errorf("list compartmentId = %q", got)
			}
			if got := request.URL.Query().Get("displayName"); got != resource.Spec.DisplayName {
				return ocimock.Response{}, fmt.Errorf("list displayName = %q", got)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []budgetsdk.BudgetSummary{})
			}
			return ocimock.JSONResponse(http.StatusOK, []budgetsdk.BudgetSummary{budgetSummaryFromMockState(state)})
		},
		Create: func(request ocimock.Request) (budgetsdk.Budget, ocimock.Response, error) {
			var details budgetsdk.CreateBudgetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return budgetsdk.Budget{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return budgetsdk.Budget{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.Amount == nil || *details.Amount != resource.Spec.Amount ||
				details.ResetPeriod != budgetsdk.ResetPeriodEnum(resource.Spec.ResetPeriod) {
				return budgetsdk.Budget{}, ocimock.Response{}, fmt.Errorf("create Budget details = %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return budgetsdk.Budget{}, ocimock.Response{}, fmt.Errorf("create Budget opc-retry-token is empty")
			}
			state := newMockBudgetState(mockBudgetID, resource.Spec, 1, createdAt.Time)
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			response.Header = http.Header{"Etag": []string{"mock-etag-1"}, "Opc-Request-Id": []string{"mock-create-request"}}
			return state, response, err
		},
		Read: func(_ ocimock.Request, state budgetsdk.Budget) (ocimock.Response, error) {
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			response.Header = http.Header{"Etag": []string{fmt.Sprintf("mock-etag-%d", *state.Version)}, "Opc-Request-Id": []string{"mock-read-request"}}
			return response, err
		},
		Update: func(request ocimock.Request, state budgetsdk.Budget) (budgetsdk.Budget, ocimock.Response, error) {
			var details budgetsdk.UpdateBudgetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return budgetsdk.Budget{}, ocimock.Response{}, err
			}
			expected := budgetsdk.UpdateBudgetDetails{DisplayName: common.String("mock-budget-updated")}
			if !reflect.DeepEqual(details, expected) {
				return budgetsdk.Budget{}, ocimock.Response{}, fmt.Errorf("update Budget details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			state.TimeUpdated = &updatedAt
			state.Version = common.Int(2)
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			response.Header = http.Header{"Etag": []string{"mock-etag-2"}, "Opc-Request-Id": []string{"mock-update-request"}}
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ budgetsdk.Budget) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete Budget body = %s", request.Body)
			}
			response := ocimock.EmptyResponse(http.StatusNoContent)
			response.Header = http.Header{"Opc-Request-Id": []string{"mock-delete-request"}}
			return response, nil
		},
	})
}

func newMockBudgetState(id string, spec budgetv1beta1.BudgetSpec, version int, timestamp time.Time) budgetsdk.Budget {
	sdkTime := common.SDKTime{Time: timestamp}
	return budgetsdk.Budget{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		Amount:         common.Float32(spec.Amount),
		ResetPeriod:    budgetsdk.ResetPeriodEnum(spec.ResetPeriod),
		LifecycleState: budgetsdk.LifecycleStateActive,
		AlertRuleCount: common.Int(0),
		TimeCreated:    &sdkTime,
		TimeUpdated:    &sdkTime,
		Version:        common.Int(version),
	}
}

func budgetSummaryFromMockState(state budgetsdk.Budget) budgetsdk.BudgetSummary {
	return budgetsdk.BudgetSummary{
		Id:             state.Id,
		CompartmentId:  state.CompartmentId,
		DisplayName:    state.DisplayName,
		Amount:         state.Amount,
		ResetPeriod:    state.ResetPeriod,
		LifecycleState: state.LifecycleState,
		AlertRuleCount: state.AlertRuleCount,
		TimeCreated:    state.TimeCreated,
		TimeUpdated:    state.TimeUpdated,
		Version:        state.Version,
	}
}
