/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package chargebackplan

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockChargebackPlanID = "ocid1.chargebackplan.oc1..mock"

// Contract evidence: vendored SDK request/response types and the handwritten chargeback-plan runtime contract.
func TestMockIntegrationChargebackPlanLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := baseChargebackPlan()
	ocimock.InitializeResource(resource, "mock-chargebackplan")
	responder, err := newChargebackPlanMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://operationsinsights.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &ChargebackPlanServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newChargebackPlanRuntimeHooks(manager, opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()})
	client := wrapChargebackPlanGeneratedClient(hooks, defaultChargebackPlanServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.ChargebackPlan](buildChargebackPlanGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.ChargebackPlan]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.ChargebackPlan) error {
			if current.Status.Id != mockChargebackPlanID || current.Status.LifecycleState != string(opsisdk.LifecycleStateActive) {
				return fmt.Errorf("created ChargebackPlan status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.ChargebackPlan) {
			current.Spec.PlanName = "mock-chargeback-plan-updated"
			current.Spec.PlanDescription = "updated"
		},
		ValidateUpdated: func(current *opsiv1beta1.ChargebackPlan) error {
			if current.Status.PlanName != current.Spec.PlanName || current.Status.PlanDescription != current.Spec.PlanDescription {
				return fmt.Errorf("updated ChargebackPlan status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func newChargebackPlanMockResponder(resource *opsiv1beta1.ChargebackPlan) (*ocimock.CRUDResponder[opsisdk.ChargebackPlan], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[opsisdk.ChargebackPlan]{
		CollectionPath: "/20200630/chargebackPlans",
		ItemPath:       "/20200630/chargebackPlans/" + mockChargebackPlanID,
		ExpectedOperations: []ocimock.Operation{
			ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete,
		},
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state opsisdk.ChargebackPlan) (ocimock.Response, error) {
			items := []opsisdk.ChargebackPlanSummary{}
			if present {
				items = append(items, chargebackPlanSummary(mockChargebackPlanID, *state.PlanName, *state.PlanType, state.LifecycleState))
			}
			return ocimock.JSONResponse(http.StatusOK, opsisdk.ChargebackPlanCollection{Items: items})
		},
		Create: func(request ocimock.Request) (opsisdk.ChargebackPlan, ocimock.Response, error) {
			if err := ocimock.ValidateRetryTokenValue(request, chargebackPlanRetryToken(resource, resource.Namespace)); err != nil {
				var zero opsisdk.ChargebackPlan
				return zero, ocimock.Response{}, err
			}
			var details map[string]any
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return opsisdk.ChargebackPlan{}, ocimock.Response{}, err
			}
			planName, nameOK := details["planName"].(string)
			planType, typeOK := details["planType"].(string)
			if !nameOK || planName == "" || !typeOK || planType == "" || details["entitySource"] != "CHARGEBACK_EXADATA" {
				return opsisdk.ChargebackPlan{}, ocimock.Response{}, fmt.Errorf("unexpected ChargebackPlan create details: %+v", details)
			}
			state := chargebackPlanSDK(mockChargebackPlanID, planName, planType, opsisdk.LifecycleStateCreating)
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state opsisdk.ChargebackPlan) (opsisdk.ChargebackPlan, ocimock.Response, error) {
			if state.LifecycleState == opsisdk.LifecycleStateCreating || state.LifecycleState == opsisdk.LifecycleStateUpdating {
				state.LifecycleState = opsisdk.LifecycleStateActive
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state opsisdk.ChargebackPlan) (opsisdk.ChargebackPlan, ocimock.Response, error) {
			var details opsisdk.UpdateChargebackPlanDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.PlanName == nil || *details.PlanName != "mock-chargeback-plan-updated" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected ChargebackPlan update details: %+v", details)
			}
			state.PlanName, state.PlanDescription, state.LifecycleState = details.PlanName, details.PlanDescription, opsisdk.LifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ opsisdk.ChargebackPlan) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(_ ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotFound", "message": "resource deleted"})
		},
	})
}
