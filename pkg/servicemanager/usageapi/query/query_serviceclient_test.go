/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package query

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeQueryOCIClient struct {
	createQueryFn func(context.Context, usageapisdk.CreateQueryRequest) (usageapisdk.CreateQueryResponse, error)
	getQueryFn    func(context.Context, usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error)
	listQueriesFn func(context.Context, usageapisdk.ListQueriesRequest) (usageapisdk.ListQueriesResponse, error)
	updateQueryFn func(context.Context, usageapisdk.UpdateQueryRequest) (usageapisdk.UpdateQueryResponse, error)
	deleteQueryFn func(context.Context, usageapisdk.DeleteQueryRequest) (usageapisdk.DeleteQueryResponse, error)
}

func (f *fakeQueryOCIClient) CreateQuery(ctx context.Context, req usageapisdk.CreateQueryRequest) (usageapisdk.CreateQueryResponse, error) {
	if f.createQueryFn != nil {
		return f.createQueryFn(ctx, req)
	}
	return usageapisdk.CreateQueryResponse{}, nil
}

func (f *fakeQueryOCIClient) GetQuery(ctx context.Context, req usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error) {
	if f.getQueryFn != nil {
		return f.getQueryFn(ctx, req)
	}
	return usageapisdk.GetQueryResponse{}, nil
}

func (f *fakeQueryOCIClient) ListQueries(ctx context.Context, req usageapisdk.ListQueriesRequest) (usageapisdk.ListQueriesResponse, error) {
	if f.listQueriesFn != nil {
		return f.listQueriesFn(ctx, req)
	}
	return usageapisdk.ListQueriesResponse{}, nil
}

func (f *fakeQueryOCIClient) UpdateQuery(ctx context.Context, req usageapisdk.UpdateQueryRequest) (usageapisdk.UpdateQueryResponse, error) {
	if f.updateQueryFn != nil {
		return f.updateQueryFn(ctx, req)
	}
	return usageapisdk.UpdateQueryResponse{}, nil
}

func (f *fakeQueryOCIClient) DeleteQuery(ctx context.Context, req usageapisdk.DeleteQueryRequest) (usageapisdk.DeleteQueryResponse, error) {
	if f.deleteQueryFn != nil {
		return f.deleteQueryFn(ctx, req)
	}
	return usageapisdk.DeleteQueryResponse{}, nil
}

func testQueryClient(fake *fakeQueryOCIClient) QueryServiceClient {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	delegate := defaultQueryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.Query](generatedruntime.Config[*usageapiv1beta1.Query]{
			Kind:      "Query",
			SDKName:   "Query",
			Log:       log,
			Semantics: newQueryRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.CreateQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.CreateQuery(ctx, *request.(*usageapisdk.CreateQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateQueryDetails", RequestName: "CreateQueryDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.GetQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.GetQuery(ctx, *request.(*usageapisdk.GetQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "QueryId", RequestName: "queryId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.ListQueriesRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.ListQueries(ctx, *request.(*usageapisdk.ListQueriesRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
					{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.UpdateQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.UpdateQuery(ctx, *request.(*usageapisdk.UpdateQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "QueryId", RequestName: "queryId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateQueryDetails", RequestName: "UpdateQueryDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.DeleteQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.DeleteQuery(ctx, *request.(*usageapisdk.DeleteQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "QueryId", RequestName: "queryId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
	return &synchronousQueryServiceClient{
		delegate: delegate,
		log:      log,
	}
}

func makeQueryResource() *usageapiv1beta1.Query {
	return &usageapiv1beta1.Query{
		Spec: usageapiv1beta1.QuerySpec{
			CompartmentId: "ocid1.compartment.oc1..queryexample",
			QueryDefinition: usageapiv1beta1.QueryDefinition{
				DisplayName: "query-sample",
				ReportQuery: usageapiv1beta1.QueryDefinitionReportQuery{
					TenantId:      "ocid1.tenancy.oc1..queryexample",
					Granularity:   "MONTHLY",
					QueryType:     "COST",
					GroupBy:       []string{"service"},
					DateRangeName: "LAST_THREE_MONTHS",
				},
				CostAnalysisUI: usageapiv1beta1.QueryDefinitionCostAnalysisUI{
					Graph: "BARS",
				},
				Version: 1,
			},
		},
	}
}

func makeSDKQueryDefinition(displayName string, graph usageapisdk.CostAnalysisUiGraphEnum) *usageapisdk.QueryDefinition {
	version := float32(1)
	return &usageapisdk.QueryDefinition{
		DisplayName: common.String(displayName),
		ReportQuery: &usageapisdk.ReportQuery{
			TenantId:      common.String("ocid1.tenancy.oc1..queryexample"),
			Granularity:   usageapisdk.ReportQueryGranularityMonthly,
			QueryType:     usageapisdk.ReportQueryQueryTypeCost,
			GroupBy:       []string{"service"},
			DateRangeName: usageapisdk.ReportQueryDateRangeNameLastThreeMonths,
		},
		CostAnalysisUI: &usageapisdk.CostAnalysisUi{
			Graph: graph,
		},
		Version: &version,
	}
}

func makeSDKQuery(id string, displayName string, graph usageapisdk.CostAnalysisUiGraphEnum) usageapisdk.Query {
	return usageapisdk.Query{
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..queryexample"),
		QueryDefinition: makeSDKQueryDefinition(displayName, graph),
	}
}

func makeSDKQuerySummary(id string, displayName string, graph usageapisdk.CostAnalysisUiGraphEnum) usageapisdk.QuerySummary {
	return usageapisdk.QuerySummary{
		Id:              common.String(id),
		QueryDefinition: makeSDKQueryDefinition(displayName, graph),
	}
}

func TestQueryServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest usageapisdk.CreateQueryRequest
	getCalls := 0
	client := testQueryClient(&fakeQueryOCIClient{
		listQueriesFn: func(_ context.Context, req usageapisdk.ListQueriesRequest) (usageapisdk.ListQueriesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..queryexample" {
				t.Fatalf("list compartmentId = %v, want query compartment", req.CompartmentId)
			}
			return usageapisdk.ListQueriesResponse{}, nil
		},
		createQueryFn: func(_ context.Context, req usageapisdk.CreateQueryRequest) (usageapisdk.CreateQueryResponse, error) {
			createRequest = req
			return usageapisdk.CreateQueryResponse{
				Query:        makeSDKQuery("ocid1.query.oc1..created", "query-sample", usageapisdk.CostAnalysisUiGraphBars),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getQueryFn: func(_ context.Context, req usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error) {
			getCalls++
			if req.QueryId == nil || *req.QueryId != "ocid1.query.oc1..created" {
				t.Fatalf("get queryId = %v, want created query OCID", req.QueryId)
			}
			return usageapisdk.GetQueryResponse{
				Query: makeSDKQuery("ocid1.query.oc1..created", "query-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
	})

	resource := makeQueryResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after read-after-write GetQuery confirmation")
	}
	if createRequest.CreateQueryDetails.CompartmentId == nil || *createRequest.CreateQueryDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateQueryDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateQueryDetails.QueryDefinition == nil || createRequest.CreateQueryDetails.QueryDefinition.DisplayName == nil || *createRequest.CreateQueryDetails.QueryDefinition.DisplayName != "query-sample" {
		t.Fatalf("create queryDefinition.displayName = %v, want %q", createRequest.CreateQueryDetails.QueryDefinition, "query-sample")
	}
	if getCalls != 1 {
		t.Fatalf("GetQuery() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.query.oc1..created" {
		t.Fatalf("status.ocid = %q, want created query OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != "ocid1.query.oc1..created" {
		t.Fatalf("status.id = %q, want created query OCID", got)
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.QueryDefinition.DisplayName; got != resource.Spec.QueryDefinition.DisplayName {
		t.Fatalf("status.queryDefinition.displayName = %q, want %q", got, resource.Spec.QueryDefinition.DisplayName)
	}
}

func TestQueryServiceClientBindsExistingQueryWithoutCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
	getCalls := 0
	client := testQueryClient(&fakeQueryOCIClient{
		listQueriesFn: func(_ context.Context, req usageapisdk.ListQueriesRequest) (usageapisdk.ListQueriesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..queryexample" {
				t.Fatalf("list compartmentId = %v, want query compartment", req.CompartmentId)
			}
			return usageapisdk.ListQueriesResponse{
				QueryCollection: usageapisdk.QueryCollection{
					Items: []usageapisdk.QuerySummary{
						makeSDKQuerySummary("ocid1.query.oc1..mismatch", "query-sample", usageapisdk.CostAnalysisUiGraphLines),
						makeSDKQuerySummary("ocid1.query.oc1..existing", "query-sample", usageapisdk.CostAnalysisUiGraphBars),
					},
				},
			}, nil
		},
		getQueryFn: func(_ context.Context, req usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error) {
			getCalls++
			if req.QueryId == nil || *req.QueryId != "ocid1.query.oc1..existing" {
				t.Fatalf("get queryId = %v, want existing query OCID", req.QueryId)
			}
			return usageapisdk.GetQueryResponse{
				Query: makeSDKQuery("ocid1.query.oc1..existing", "query-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		createQueryFn: func(_ context.Context, _ usageapisdk.CreateQueryRequest) (usageapisdk.CreateQueryResponse, error) {
			createCalled = true
			return usageapisdk.CreateQueryResponse{}, nil
		},
	})

	resource := makeQueryResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when list lookup reuses an existing query")
	}
	if createCalled {
		t.Fatal("CreateQuery() should not be called when ListQueries finds a matching query")
	}
	if getCalls != 1 {
		t.Fatalf("GetQuery() calls = %d, want 1 read of the bound query", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.query.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing query OCID", got)
	}
}

func TestQueryServiceClientUpdatesQueryDefinition(t *testing.T) {
	t.Parallel()

	var updateRequest usageapisdk.UpdateQueryRequest
	getCalls := 0
	updateCalls := 0
	client := testQueryClient(&fakeQueryOCIClient{
		getQueryFn: func(_ context.Context, req usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error) {
			getCalls++
			if req.QueryId == nil || *req.QueryId != "ocid1.query.oc1..existing" {
				t.Fatalf("get queryId = %v, want existing query OCID", req.QueryId)
			}
			graph := usageapisdk.CostAnalysisUiGraphBars
			displayName := "query-sample"
			if getCalls > 1 {
				graph = usageapisdk.CostAnalysisUiGraphLines
				displayName = "query-updated"
			}
			return usageapisdk.GetQueryResponse{
				Query: makeSDKQuery("ocid1.query.oc1..existing", displayName, graph),
			}, nil
		},
		updateQueryFn: func(_ context.Context, req usageapisdk.UpdateQueryRequest) (usageapisdk.UpdateQueryResponse, error) {
			updateCalls++
			updateRequest = req
			return usageapisdk.UpdateQueryResponse{
				Query:        makeSDKQuery("ocid1.query.oc1..existing", "query-updated", usageapisdk.CostAnalysisUiGraphLines),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.query.oc1..existing")
	resource.Spec.QueryDefinition.DisplayName = "query-updated"
	resource.Spec.QueryDefinition.CostAnalysisUI.Graph = "LINES"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating queryDefinition")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after update follow-up GetQuery confirmation")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateQuery() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetQuery() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.QueryId == nil || *updateRequest.QueryId != "ocid1.query.oc1..existing" {
		t.Fatalf("update queryId = %v, want existing query OCID", updateRequest.QueryId)
	}
	if updateRequest.UpdateQueryDetails.QueryDefinition == nil || updateRequest.UpdateQueryDetails.QueryDefinition.DisplayName == nil || *updateRequest.UpdateQueryDetails.QueryDefinition.DisplayName != "query-updated" {
		t.Fatalf("update queryDefinition.displayName = %v, want %q", updateRequest.UpdateQueryDetails.QueryDefinition, "query-updated")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
	if got := resource.Status.QueryDefinition.DisplayName; got != "query-updated" {
		t.Fatalf("status.queryDefinition.displayName = %q, want %q", got, "query-updated")
	}
	if got := resource.Status.QueryDefinition.CostAnalysisUI.Graph; got != "LINES" {
		t.Fatalf("status.queryDefinition.costAnalysisUI.graph = %q, want %q", got, "LINES")
	}
}

func TestQueryServiceClientRejectsReplacementOnlyCompartmentDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testQueryClient(&fakeQueryOCIClient{
		getQueryFn: func(_ context.Context, _ usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error) {
			return usageapisdk.GetQueryResponse{
				Query: makeSDKQuery("ocid1.query.oc1..existing", "query-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		updateQueryFn: func(_ context.Context, _ usageapisdk.UpdateQueryRequest) (usageapisdk.UpdateQueryResponse, error) {
			updateCalled = true
			return usageapisdk.UpdateQueryResponse{}, nil
		},
	})

	resource := makeQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.query.oc1..existing")
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateQuery() should not be called when replacement-only drift is detected")
	}
}

func TestQueryServiceClientDeleteConfirmsNotFound(t *testing.T) {
	t.Parallel()

	var deleteRequest usageapisdk.DeleteQueryRequest
	getCalls := 0
	client := testQueryClient(&fakeQueryOCIClient{
		getQueryFn: func(_ context.Context, req usageapisdk.GetQueryRequest) (usageapisdk.GetQueryResponse, error) {
			getCalls++
			if req.QueryId == nil || *req.QueryId != "ocid1.query.oc1..existing" {
				t.Fatalf("get queryId = %v, want existing query OCID", req.QueryId)
			}
			if getCalls > 1 {
				return usageapisdk.GetQueryResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "query not found")
			}
			return usageapisdk.GetQueryResponse{
				Query: makeSDKQuery("ocid1.query.oc1..existing", "query-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		deleteQueryFn: func(_ context.Context, req usageapisdk.DeleteQueryRequest) (usageapisdk.DeleteQueryResponse, error) {
			deleteRequest = req
			return usageapisdk.DeleteQueryResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.query.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetQuery confirms not found")
	}
	if getCalls != 2 {
		t.Fatalf("GetQuery() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.QueryId == nil || *deleteRequest.QueryId != "ocid1.query.oc1..existing" {
		t.Fatalf("delete queryId = %v, want existing query OCID", deleteRequest.QueryId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
}
