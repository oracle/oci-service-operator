/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package query

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

const mockQueryID = "ocid1.usagequery.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal state-free contract, and vendored OCI SDK.
func TestMockIntegrationQueryLifecycleCRUD(t *testing.T) {
	t.Parallel()
	const tenancyID = "ocid1.tenancy.oc1..mock"
	resource := &usageapiv1beta1.Query{ObjectMeta: metav1.ObjectMeta{Name: "mock-query", Namespace: "default", UID: types.UID("mock-query-uid")}, Spec: usageapiv1beta1.QuerySpec{CompartmentId: tenancyID, QueryDefinition: usageapiv1beta1.QueryDefinition{
		DisplayName: "mock-query", Version: 1, ReportQuery: usageapiv1beta1.QueryDefinitionReportQuery{TenantId: tenancyID, Granularity: "MONTHLY", QueryType: "COST", GroupBy: []string{"service"}, TimeUsageStarted: "2026-08-01T00:00:00Z", TimeUsageEnded: "2026-09-01T00:00:00Z"}, CostAnalysisUI: usageapiv1beta1.QueryDefinitionCostAnalysisUI{Graph: "BARS"},
	}}}
	responder, err := newQueryMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://usage.mock.invalid", BasePath: "20200107", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &QueryServiceManager{}
	hooks := newQueryRuntimeHooks(manager, usageapisdk.UsageapiClient{BaseClient: session.BaseClient()})
	client := wrapQueryGeneratedClient(hooks, defaultQueryServiceClient{ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.Query](buildQueryGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*usageapiv1beta1.Query]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *usageapiv1beta1.Query) error {
			if current.Status.Id != mockQueryID || current.Status.QueryDefinition.DisplayName != resource.Spec.QueryDefinition.DisplayName {
				return fmt.Errorf("created Query status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *usageapiv1beta1.Query) { current.Spec.QueryDefinition.DisplayName = "mock-query-updated" },
		ValidateUpdated: func(current *usageapiv1beta1.Query) error {
			if current.Status.QueryDefinition.DisplayName != current.Spec.QueryDefinition.DisplayName {
				return fmt.Errorf("updated Query status = %+v", current.Status)
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

func newQueryMockResponder(resource *usageapiv1beta1.Query) (*ocimock.CRUDResponder[usageapisdk.Query], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[usageapisdk.Query]{
		CollectionPath: "/20200107/queries", ItemPath: "/20200107/queries/" + mockQueryID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state usageapisdk.Query) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.Query{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.Query{state}})
		},
		Create: func(request ocimock.Request) (usageapisdk.Query, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero usageapisdk.Query
				return zero, ocimock.Response{}, err
			}
			var details usageapisdk.CreateQueryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.Query{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return usageapisdk.Query{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.QueryDefinition == nil || details.QueryDefinition.DisplayName == nil || *details.QueryDefinition.DisplayName != resource.Spec.QueryDefinition.DisplayName {
				return usageapisdk.Query{}, ocimock.Response{}, fmt.Errorf("unexpected CreateQuery details: %+v", details)
			}
			state := usageapisdk.Query{Id: common.String(mockQueryID), CompartmentId: details.CompartmentId, QueryDefinition: details.QueryDefinition}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state usageapisdk.Query) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state usageapisdk.Query) (usageapisdk.Query, ocimock.Response, error) {
			var details usageapisdk.UpdateQueryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.Query{}, ocimock.Response{}, err
			}
			if details.QueryDefinition == nil || details.QueryDefinition.DisplayName == nil || *details.QueryDefinition.DisplayName != "mock-query-updated" {
				return usageapisdk.Query{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateQuery details: %+v", details)
			}
			state.QueryDefinition = details.QueryDefinition
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ usageapisdk.Query) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
