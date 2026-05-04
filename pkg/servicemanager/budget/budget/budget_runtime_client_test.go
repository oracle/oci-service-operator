/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package budget

import (
	"context"
	"strings"
	"testing"
	"time"

	budgetsdk "github.com/oracle/oci-go-sdk/v65/budget"
	"github.com/oracle/oci-go-sdk/v65/common"
	budgetv1beta1 "github.com/oracle/oci-service-operator/api/budget/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeBudgetOCIClient struct {
	createFn func(context.Context, budgetsdk.CreateBudgetRequest) (budgetsdk.CreateBudgetResponse, error)
	getFn    func(context.Context, budgetsdk.GetBudgetRequest) (budgetsdk.GetBudgetResponse, error)
	listFn   func(context.Context, budgetsdk.ListBudgetsRequest) (budgetsdk.ListBudgetsResponse, error)
	updateFn func(context.Context, budgetsdk.UpdateBudgetRequest) (budgetsdk.UpdateBudgetResponse, error)
	deleteFn func(context.Context, budgetsdk.DeleteBudgetRequest) (budgetsdk.DeleteBudgetResponse, error)
}

func (f *fakeBudgetOCIClient) CreateBudget(
	ctx context.Context,
	req budgetsdk.CreateBudgetRequest,
) (budgetsdk.CreateBudgetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return budgetsdk.CreateBudgetResponse{}, nil
}

func (f *fakeBudgetOCIClient) GetBudget(
	ctx context.Context,
	req budgetsdk.GetBudgetRequest,
) (budgetsdk.GetBudgetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return budgetsdk.GetBudgetResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeBudgetOCIClient) ListBudgets(
	ctx context.Context,
	req budgetsdk.ListBudgetsRequest,
) (budgetsdk.ListBudgetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return budgetsdk.ListBudgetsResponse{}, nil
}

func (f *fakeBudgetOCIClient) UpdateBudget(
	ctx context.Context,
	req budgetsdk.UpdateBudgetRequest,
) (budgetsdk.UpdateBudgetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return budgetsdk.UpdateBudgetResponse{}, nil
}

func (f *fakeBudgetOCIClient) DeleteBudget(
	ctx context.Context,
	req budgetsdk.DeleteBudgetRequest,
) (budgetsdk.DeleteBudgetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return budgetsdk.DeleteBudgetResponse{}, nil
}

func newTestBudgetClient(client *fakeBudgetOCIClient) BudgetServiceClient {
	if client == nil {
		client = &fakeBudgetOCIClient{}
	}

	hooks := newBudgetDefaultRuntimeHooks(budgetsdk.BudgetClient{})
	hooks.Create.Call = func(ctx context.Context, req budgetsdk.CreateBudgetRequest) (budgetsdk.CreateBudgetResponse, error) {
		return client.CreateBudget(ctx, req)
	}
	hooks.Get.Call = func(ctx context.Context, req budgetsdk.GetBudgetRequest) (budgetsdk.GetBudgetResponse, error) {
		return client.GetBudget(ctx, req)
	}
	hooks.List.Call = func(ctx context.Context, req budgetsdk.ListBudgetsRequest) (budgetsdk.ListBudgetsResponse, error) {
		return client.ListBudgets(ctx, req)
	}
	hooks.Update.Call = func(ctx context.Context, req budgetsdk.UpdateBudgetRequest) (budgetsdk.UpdateBudgetResponse, error) {
		return client.UpdateBudget(ctx, req)
	}
	hooks.Delete.Call = func(ctx context.Context, req budgetsdk.DeleteBudgetRequest) (budgetsdk.DeleteBudgetResponse, error) {
		return client.DeleteBudget(ctx, req)
	}
	applyBudgetRuntimeHooks(&hooks)

	config := buildBudgetGeneratedRuntimeConfig(&BudgetServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}, hooks)

	delegate := defaultBudgetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*budgetv1beta1.Budget](config),
	}
	return wrapBudgetGeneratedClient(hooks, delegate)
}

func makeSpecBudget() *budgetv1beta1.Budget {
	return &budgetv1beta1.Budget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "budget-sample",
			Namespace: "default",
		},
		Spec: budgetv1beta1.BudgetSpec{
			CompartmentId: "ocid1.compartment.oc1..budgetexample",
			Amount:        100,
			ResetPeriod:   "MONTHLY",
			DisplayName:   "budget-sample",
		},
	}
}

func makeSDKBudget(
	id string,
	spec budgetv1beta1.BudgetSpec,
	state budgetsdk.LifecycleStateEnum,
) budgetsdk.Budget {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	amount := spec.Amount
	alertRuleCount := 0
	version := 1

	budget := budgetsdk.Budget{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Amount:         &amount,
		ResetPeriod:    budgetsdk.ResetPeriodEnum(spec.ResetPeriod),
		LifecycleState: state,
		AlertRuleCount: &alertRuleCount,
		TimeCreated:    &now,
		TimeUpdated:    &now,
		Version:        &version,
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		budget.DisplayName = common.String(spec.DisplayName)
	}
	return budget
}

func makeSDKBudgetSummary(
	id string,
	spec budgetv1beta1.BudgetSpec,
	state budgetsdk.LifecycleStateEnum,
) budgetsdk.BudgetSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	amount := spec.Amount
	alertRuleCount := 0
	version := 1

	summary := budgetsdk.BudgetSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Amount:         &amount,
		ResetPeriod:    budgetsdk.ResetPeriodEnum(spec.ResetPeriod),
		LifecycleState: state,
		AlertRuleCount: &alertRuleCount,
		TimeCreated:    &now,
		TimeUpdated:    &now,
		Version:        &version,
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		summary.DisplayName = common.String(spec.DisplayName)
	}
	return summary
}

func TestBudgetCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.budget.oc1..created"

	resource := makeSpecBudget()
	resource.Spec.DisplayName = ""

	createCalls := 0
	listCalls := 0

	client := newTestBudgetClient(&fakeBudgetOCIClient{
		createFn: func(_ context.Context, req budgetsdk.CreateBudgetRequest) (budgetsdk.CreateBudgetResponse, error) {
			createCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("CreateBudgetRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName != nil {
				t.Fatalf("CreateBudgetRequest.DisplayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			return budgetsdk.CreateBudgetResponse{
				Budget: makeSDKBudget(createdID, resource.Spec, budgetsdk.LifecycleStateActive),
			}, nil
		},
		getFn: func(_ context.Context, req budgetsdk.GetBudgetRequest) (budgetsdk.GetBudgetResponse, error) {
			if req.BudgetId == nil || *req.BudgetId != createdID {
				t.Fatalf("GetBudgetRequest.BudgetId = %v, want %q", req.BudgetId, createdID)
			}
			return budgetsdk.GetBudgetResponse{
				Budget: makeSDKBudget(createdID, resource.Spec, budgetsdk.LifecycleStateActive),
			}, nil
		},
		listFn: func(_ context.Context, _ budgetsdk.ListBudgetsRequest) (budgetsdk.ListBudgetsResponse, error) {
			listCalls++
			return budgetsdk.ListBudgetsResponse{}, nil
		},
	})

	resp, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !resp.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful create", resp)
	}
	if createCalls != 1 {
		t.Fatalf("CreateBudget() calls = %d, want 1", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListBudgets() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.status.ocid = %q, want %q", got, createdID)
	}
}

func TestBudgetCreateOrUpdateRejectsAmbiguousDisplayNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeSpecBudget()
	createCalls := 0

	client := newTestBudgetClient(&fakeBudgetOCIClient{
		listFn: func(_ context.Context, req budgetsdk.ListBudgetsRequest) (budgetsdk.ListBudgetsResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("ListBudgetsRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("ListBudgetsRequest.DisplayName = %v, want %q", req.DisplayName, resource.Spec.DisplayName)
			}
			return budgetsdk.ListBudgetsResponse{
				Items: []budgetsdk.BudgetSummary{
					makeSDKBudgetSummary("ocid1.budget.oc1..first", resource.Spec, budgetsdk.LifecycleStateActive),
					makeSDKBudgetSummary("ocid1.budget.oc1..second", resource.Spec, budgetsdk.LifecycleStateInactive),
				},
			}, nil
		},
		createFn: func(_ context.Context, _ budgetsdk.CreateBudgetRequest) (budgetsdk.CreateBudgetResponse, error) {
			createCalls++
			return budgetsdk.CreateBudgetResponse{}, nil
		},
	})

	resp, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous pre-create reuse failure")
	}
	if resp.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful ambiguous match result", resp)
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate displayName match failure", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreateBudget() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
	if resource.Status.Id != "" || string(resource.Status.OsokStatus.Ocid) != "" {
		t.Fatalf("status should stay empty after ambiguous reuse failure, got id=%q ocid=%q", resource.Status.Id, resource.Status.OsokStatus.Ocid)
	}
}
