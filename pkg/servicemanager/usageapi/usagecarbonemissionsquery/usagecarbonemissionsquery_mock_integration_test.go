/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package usagecarbonemissionsquery

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockCarbonQueryID = "ocid1.usagecarbonemissionsquery.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, reviewed formal state-free contract, and vendored OCI SDK.
func TestMockIntegrationUsageCarbonEmissionsQueryLifecycleCRUD(t *testing.T) {
	t.Parallel()
	const tenancyID = "ocid1.tenancy.oc1..mock"
	resource := &usageapiv1beta1.UsageCarbonEmissionsQuery{ObjectMeta: metav1.ObjectMeta{Name: "mock-carbon-query", Namespace: "default", UID: types.UID("mock-carbon-query-uid")}, Spec: usageapiv1beta1.UsageCarbonEmissionsQuerySpec{
		CompartmentId: tenancyID, QueryDefinition: usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinition{DisplayName: "mock-carbon-query", Version: 1,
			ReportQuery:    usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionReportQuery{TenantId: tenancyID, TimeUsageStarted: "2026-08-01T00:00:00Z", TimeUsageEnded: "2026-09-01T00:00:00Z", EmissionCalculationMethod: "SPEND_BASED", EmissionType: "LOCATION_BASED", Granularity: "MONTHLY", GroupBy: []string{"service"}},
			CostAnalysisUI: usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionCostAnalysisUI{Graph: "BARS"}},
	}}
	responder, err := newCarbonQueryMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://usage.mock.invalid", BasePath: "20200107", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &UsageCarbonEmissionsQueryServiceManager{}
	hooks := newUsageCarbonEmissionsQueryRuntimeHooks(manager, usageapisdk.UsageapiClient{BaseClient: session.BaseClient()})
	client := wrapUsageCarbonEmissionsQueryGeneratedClient(hooks, defaultUsageCarbonEmissionsQueryServiceClient{ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.UsageCarbonEmissionsQuery](buildUsageCarbonEmissionsQueryGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*usageapiv1beta1.UsageCarbonEmissionsQuery]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *usageapiv1beta1.UsageCarbonEmissionsQuery) error {
			if current.Status.Id != mockCarbonQueryID || current.Status.QueryDefinition.DisplayName != resource.Spec.QueryDefinition.DisplayName {
				return fmt.Errorf("created UsageCarbonEmissionsQuery status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *usageapiv1beta1.UsageCarbonEmissionsQuery) {
			current.Spec.QueryDefinition.DisplayName = "mock-carbon-query-updated"
		},
		ValidateUpdated: func(current *usageapiv1beta1.UsageCarbonEmissionsQuery) error {
			if current.Status.QueryDefinition.DisplayName != current.Spec.QueryDefinition.DisplayName {
				return fmt.Errorf("updated UsageCarbonEmissionsQuery status = %+v", current.Status)
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

func newCarbonQueryMockResponder(resource *usageapiv1beta1.UsageCarbonEmissionsQuery) (*ocimock.CRUDResponder[usageapisdk.UsageCarbonEmissionsQuery], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[usageapisdk.UsageCarbonEmissionsQuery]{
		CollectionPath: "/20200107/usageCarbonEmissionsQueries", ItemPath: "/20200107/usageCarbonEmissionsQueries/" + mockCarbonQueryID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state usageapisdk.UsageCarbonEmissionsQuery) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.UsageCarbonEmissionsQuery{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.UsageCarbonEmissionsQuery{state}})
		},
		Create: func(request ocimock.Request) (usageapisdk.UsageCarbonEmissionsQuery, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero usageapisdk.UsageCarbonEmissionsQuery
				return zero, ocimock.Response{}, err
			}
			var details usageapisdk.CreateUsageCarbonEmissionsQueryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.UsageCarbonEmissionsQuery{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return usageapisdk.UsageCarbonEmissionsQuery{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.QueryDefinition == nil || details.QueryDefinition.DisplayName == nil || *details.QueryDefinition.DisplayName != resource.Spec.QueryDefinition.DisplayName {
				return usageapisdk.UsageCarbonEmissionsQuery{}, ocimock.Response{}, fmt.Errorf("unexpected CreateUsageCarbonEmissionsQuery details: %+v", details)
			}
			state := usageapisdk.UsageCarbonEmissionsQuery{Id: common.String(mockCarbonQueryID), CompartmentId: details.CompartmentId, QueryDefinition: details.QueryDefinition}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state usageapisdk.UsageCarbonEmissionsQuery) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state usageapisdk.UsageCarbonEmissionsQuery) (usageapisdk.UsageCarbonEmissionsQuery, ocimock.Response, error) {
			var details usageapisdk.UpdateUsageCarbonEmissionsQueryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.UsageCarbonEmissionsQuery{}, ocimock.Response{}, err
			}
			if details.QueryDefinition == nil || details.QueryDefinition.DisplayName == nil || *details.QueryDefinition.DisplayName != "mock-carbon-query-updated" {
				return usageapisdk.UsageCarbonEmissionsQuery{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateUsageCarbonEmissionsQuery details: %+v", details)
			}
			state.QueryDefinition = details.QueryDefinition
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ usageapisdk.UsageCarbonEmissionsQuery) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
