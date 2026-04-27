/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package usagecarbonemissionsquery

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

type fakeUsageCarbonEmissionsQueryOCIClient struct {
	createUsageCarbonEmissionsQueryFn func(context.Context, usageapisdk.CreateUsageCarbonEmissionsQueryRequest) (usageapisdk.CreateUsageCarbonEmissionsQueryResponse, error)
	getUsageCarbonEmissionsQueryFn    func(context.Context, usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error)
	listUsageCarbonEmissionsQueriesFn func(context.Context, usageapisdk.ListUsageCarbonEmissionsQueriesRequest) (usageapisdk.ListUsageCarbonEmissionsQueriesResponse, error)
	updateUsageCarbonEmissionsQueryFn func(context.Context, usageapisdk.UpdateUsageCarbonEmissionsQueryRequest) (usageapisdk.UpdateUsageCarbonEmissionsQueryResponse, error)
	deleteUsageCarbonEmissionsQueryFn func(context.Context, usageapisdk.DeleteUsageCarbonEmissionsQueryRequest) (usageapisdk.DeleteUsageCarbonEmissionsQueryResponse, error)
}

func (f *fakeUsageCarbonEmissionsQueryOCIClient) CreateUsageCarbonEmissionsQuery(ctx context.Context, req usageapisdk.CreateUsageCarbonEmissionsQueryRequest) (usageapisdk.CreateUsageCarbonEmissionsQueryResponse, error) {
	if f.createUsageCarbonEmissionsQueryFn != nil {
		return f.createUsageCarbonEmissionsQueryFn(ctx, req)
	}
	return usageapisdk.CreateUsageCarbonEmissionsQueryResponse{}, nil
}

func (f *fakeUsageCarbonEmissionsQueryOCIClient) GetUsageCarbonEmissionsQuery(ctx context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
	if f.getUsageCarbonEmissionsQueryFn != nil {
		return f.getUsageCarbonEmissionsQueryFn(ctx, req)
	}
	return usageapisdk.GetUsageCarbonEmissionsQueryResponse{}, nil
}

func (f *fakeUsageCarbonEmissionsQueryOCIClient) ListUsageCarbonEmissionsQueries(ctx context.Context, req usageapisdk.ListUsageCarbonEmissionsQueriesRequest) (usageapisdk.ListUsageCarbonEmissionsQueriesResponse, error) {
	if f.listUsageCarbonEmissionsQueriesFn != nil {
		return f.listUsageCarbonEmissionsQueriesFn(ctx, req)
	}
	return usageapisdk.ListUsageCarbonEmissionsQueriesResponse{}, nil
}

func (f *fakeUsageCarbonEmissionsQueryOCIClient) UpdateUsageCarbonEmissionsQuery(ctx context.Context, req usageapisdk.UpdateUsageCarbonEmissionsQueryRequest) (usageapisdk.UpdateUsageCarbonEmissionsQueryResponse, error) {
	if f.updateUsageCarbonEmissionsQueryFn != nil {
		return f.updateUsageCarbonEmissionsQueryFn(ctx, req)
	}
	return usageapisdk.UpdateUsageCarbonEmissionsQueryResponse{}, nil
}

func (f *fakeUsageCarbonEmissionsQueryOCIClient) DeleteUsageCarbonEmissionsQuery(ctx context.Context, req usageapisdk.DeleteUsageCarbonEmissionsQueryRequest) (usageapisdk.DeleteUsageCarbonEmissionsQueryResponse, error) {
	if f.deleteUsageCarbonEmissionsQueryFn != nil {
		return f.deleteUsageCarbonEmissionsQueryFn(ctx, req)
	}
	return usageapisdk.DeleteUsageCarbonEmissionsQueryResponse{}, nil
}

func testUsageCarbonEmissionsQueryClient(fake *fakeUsageCarbonEmissionsQueryOCIClient) UsageCarbonEmissionsQueryServiceClient {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	delegate := defaultUsageCarbonEmissionsQueryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.UsageCarbonEmissionsQuery](generatedruntime.Config[*usageapiv1beta1.UsageCarbonEmissionsQuery]{
			Kind:            "UsageCarbonEmissionsQuery",
			SDKName:         "UsageCarbonEmissionsQuery",
			Log:             log,
			Semantics:       newUsageCarbonEmissionsQueryRuntimeSemantics(),
			BuildCreateBody: buildUsageCarbonEmissionsQueryCreateBody,
			BuildUpdateBody: buildUsageCarbonEmissionsQueryUpdateBody,
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.CreateUsageCarbonEmissionsQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.CreateUsageCarbonEmissionsQuery(ctx, *request.(*usageapisdk.CreateUsageCarbonEmissionsQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateUsageCarbonEmissionsQueryDetails", RequestName: "CreateUsageCarbonEmissionsQueryDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.GetUsageCarbonEmissionsQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.GetUsageCarbonEmissionsQuery(ctx, *request.(*usageapisdk.GetUsageCarbonEmissionsQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "UsageCarbonEmissionsQueryId", RequestName: "usageCarbonEmissionsQueryId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.ListUsageCarbonEmissionsQueriesRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.ListUsageCarbonEmissionsQueries(ctx, *request.(*usageapisdk.ListUsageCarbonEmissionsQueriesRequest))
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
				NewRequest: func() any { return &usageapisdk.UpdateUsageCarbonEmissionsQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.UpdateUsageCarbonEmissionsQuery(ctx, *request.(*usageapisdk.UpdateUsageCarbonEmissionsQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "UsageCarbonEmissionsQueryId", RequestName: "usageCarbonEmissionsQueryId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateUsageCarbonEmissionsQueryDetails", RequestName: "UpdateUsageCarbonEmissionsQueryDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.DeleteUsageCarbonEmissionsQueryRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.DeleteUsageCarbonEmissionsQuery(ctx, *request.(*usageapisdk.DeleteUsageCarbonEmissionsQueryRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "UsageCarbonEmissionsQueryId", RequestName: "usageCarbonEmissionsQueryId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
	return &synchronousUsageCarbonEmissionsQueryServiceClient{
		delegate: delegate,
		log:      log,
	}
}

func makeUsageCarbonEmissionsQueryResource() *usageapiv1beta1.UsageCarbonEmissionsQuery {
	return &usageapiv1beta1.UsageCarbonEmissionsQuery{
		Spec: usageapiv1beta1.UsageCarbonEmissionsQuerySpec{
			CompartmentId: "ocid1.compartment.oc1..carbonexample",
			QueryDefinition: usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinition{
				DisplayName: "carbon-sample",
				ReportQuery: usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionReportQuery{
					TenantId:      "ocid1.tenancy.oc1..carbonexample",
					GroupBy:       []string{"service"},
					DateRangeName: "LAST_THREE_MONTHS",
				},
				CostAnalysisUI: usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionCostAnalysisUI{
					Graph: "BARS",
				},
				Version: 1,
			},
		},
	}
}

func makeSDKUsageCarbonEmissionsQueryDefinition(displayName string, graph usageapisdk.CostAnalysisUiGraphEnum) *usageapisdk.UsageCarbonEmissionsQueryDefinition {
	return &usageapisdk.UsageCarbonEmissionsQueryDefinition{
		DisplayName: common.String(displayName),
		ReportQuery: &usageapisdk.UsageCarbonEmissionsReportQuery{
			TenantId:      common.String("ocid1.tenancy.oc1..carbonexample"),
			GroupBy:       []string{"service"},
			DateRangeName: usageapisdk.UsageCarbonEmissionsReportQueryDateRangeNameLastThreeMonths,
		},
		CostAnalysisUI: &usageapisdk.CostAnalysisUi{
			Graph: graph,
		},
		Version: common.Int(1),
	}
}

func makeSDKUsageCarbonEmissionsQuery(id string, displayName string, graph usageapisdk.CostAnalysisUiGraphEnum) usageapisdk.UsageCarbonEmissionsQuery {
	return usageapisdk.UsageCarbonEmissionsQuery{
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..carbonexample"),
		QueryDefinition: makeSDKUsageCarbonEmissionsQueryDefinition(displayName, graph),
	}
}

func makeSDKUsageCarbonEmissionsQuerySummary(id string, displayName string, graph usageapisdk.CostAnalysisUiGraphEnum) usageapisdk.UsageCarbonEmissionsQuerySummary {
	return usageapisdk.UsageCarbonEmissionsQuerySummary{
		Id:              common.String(id),
		QueryDefinition: makeSDKUsageCarbonEmissionsQueryDefinition(displayName, graph),
	}
}

func TestUsageCarbonEmissionsQueryServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest usageapisdk.CreateUsageCarbonEmissionsQueryRequest
	getCalls := 0
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		listUsageCarbonEmissionsQueriesFn: func(_ context.Context, req usageapisdk.ListUsageCarbonEmissionsQueriesRequest) (usageapisdk.ListUsageCarbonEmissionsQueriesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..carbonexample" {
				t.Fatalf("list compartmentId = %v, want carbon emissions query compartment", req.CompartmentId)
			}
			return usageapisdk.ListUsageCarbonEmissionsQueriesResponse{}, nil
		},
		createUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.CreateUsageCarbonEmissionsQueryRequest) (usageapisdk.CreateUsageCarbonEmissionsQueryResponse, error) {
			createRequest = req
			return usageapisdk.CreateUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..created", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
				OpcRequestId:              common.String("opc-create-1"),
			}, nil
		},
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			getCalls++
			if req.UsageCarbonEmissionsQueryId == nil || *req.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..created" {
				t.Fatalf("get usageCarbonEmissionsQueryId = %v, want created query OCID", req.UsageCarbonEmissionsQueryId)
			}
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..created", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after read-after-write GetUsageCarbonEmissionsQuery confirmation")
	}
	if createRequest.CreateUsageCarbonEmissionsQueryDetails.CompartmentId == nil || *createRequest.CreateUsageCarbonEmissionsQueryDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateUsageCarbonEmissionsQueryDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateUsageCarbonEmissionsQueryDetails.QueryDefinition == nil || createRequest.CreateUsageCarbonEmissionsQueryDetails.QueryDefinition.DisplayName == nil || *createRequest.CreateUsageCarbonEmissionsQueryDetails.QueryDefinition.DisplayName != "carbon-sample" {
		t.Fatalf("create queryDefinition.displayName = %v, want %q", createRequest.CreateUsageCarbonEmissionsQueryDetails.QueryDefinition, "carbon-sample")
	}
	if createRequest.CreateUsageCarbonEmissionsQueryDetails.QueryDefinition.ReportQuery.Filter != nil {
		t.Fatal("create queryDefinition.reportQuery.filter should be omitted when no filter fields are set")
	}
	if createRequest.CreateUsageCarbonEmissionsQueryDetails.QueryDefinition.ReportQuery.IsAggregateByTime != nil {
		t.Fatal("create queryDefinition.reportQuery.isAggregateByTime should be omitted when false and unset in OCI")
	}
	if getCalls != 1 {
		t.Fatalf("GetUsageCarbonEmissionsQuery() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.usagecarbonquery.oc1..created" {
		t.Fatalf("status.ocid = %q, want created query OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != "ocid1.usagecarbonquery.oc1..created" {
		t.Fatalf("status.id = %q, want created query OCID", got)
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.QueryDefinition.DisplayName; got != resource.Spec.QueryDefinition.DisplayName {
		t.Fatalf("status.queryDefinition.displayName = %q, want %q", got, resource.Spec.QueryDefinition.DisplayName)
	}
}

func TestUsageCarbonEmissionsQueryServiceClientBindsExistingQueryWithoutCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
	getCalls := 0
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		listUsageCarbonEmissionsQueriesFn: func(_ context.Context, req usageapisdk.ListUsageCarbonEmissionsQueriesRequest) (usageapisdk.ListUsageCarbonEmissionsQueriesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..carbonexample" {
				t.Fatalf("list compartmentId = %v, want carbon emissions query compartment", req.CompartmentId)
			}
			return usageapisdk.ListUsageCarbonEmissionsQueriesResponse{
				UsageCarbonEmissionsQueryCollection: usageapisdk.UsageCarbonEmissionsQueryCollection{
					Items: []usageapisdk.UsageCarbonEmissionsQuerySummary{
						makeSDKUsageCarbonEmissionsQuerySummary("ocid1.usagecarbonquery.oc1..mismatch", "carbon-sample", usageapisdk.CostAnalysisUiGraphLines),
						makeSDKUsageCarbonEmissionsQuerySummary("ocid1.usagecarbonquery.oc1..existing", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
					},
				},
			}, nil
		},
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			getCalls++
			if req.UsageCarbonEmissionsQueryId == nil || *req.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
				t.Fatalf("get usageCarbonEmissionsQueryId = %v, want existing query OCID", req.UsageCarbonEmissionsQueryId)
			}
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		createUsageCarbonEmissionsQueryFn: func(_ context.Context, _ usageapisdk.CreateUsageCarbonEmissionsQueryRequest) (usageapisdk.CreateUsageCarbonEmissionsQueryResponse, error) {
			createCalled = true
			return usageapisdk.CreateUsageCarbonEmissionsQueryResponse{}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when list lookup reuses an existing query")
	}
	if createCalled {
		t.Fatal("CreateUsageCarbonEmissionsQuery() should not be called when ListUsageCarbonEmissionsQueries finds a matching query")
	}
	if getCalls != 1 {
		t.Fatalf("GetUsageCarbonEmissionsQuery() calls = %d, want 1 read of the bound query", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.usagecarbonquery.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing query OCID", got)
	}
}

func TestUsageCarbonEmissionsQueryServiceClientObservesWithoutUpdateWhenNoDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			if req.UsageCarbonEmissionsQueryId == nil || *req.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
				t.Fatalf("get usageCarbonEmissionsQueryId = %v, want existing query OCID", req.UsageCarbonEmissionsQueryId)
			}
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		updateUsageCarbonEmissionsQueryFn: func(_ context.Context, _ usageapisdk.UpdateUsageCarbonEmissionsQueryRequest) (usageapisdk.UpdateUsageCarbonEmissionsQueryResponse, error) {
			updateCalled = true
			return usageapisdk.UpdateUsageCarbonEmissionsQueryResponse{}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.usagecarbonquery.oc1..existing")

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when observed state matches spec")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when observed state is current")
	}
	if updateCalled {
		t.Fatal("UpdateUsageCarbonEmissionsQuery() should not be called when queryDefinition has no drift")
	}
}

func TestUsageCarbonEmissionsQueryServiceClientUpdatesQueryDefinition(t *testing.T) {
	t.Parallel()

	var updateRequest usageapisdk.UpdateUsageCarbonEmissionsQueryRequest
	getCalls := 0
	updateCalls := 0
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			getCalls++
			if req.UsageCarbonEmissionsQueryId == nil || *req.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
				t.Fatalf("get usageCarbonEmissionsQueryId = %v, want existing query OCID", req.UsageCarbonEmissionsQueryId)
			}
			graph := usageapisdk.CostAnalysisUiGraphBars
			displayName := "carbon-sample"
			if getCalls > 1 {
				graph = usageapisdk.CostAnalysisUiGraphLines
				displayName = "carbon-updated"
			}
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", displayName, graph),
			}, nil
		},
		updateUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.UpdateUsageCarbonEmissionsQueryRequest) (usageapisdk.UpdateUsageCarbonEmissionsQueryResponse, error) {
			updateCalls++
			updateRequest = req
			return usageapisdk.UpdateUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", "carbon-updated", usageapisdk.CostAnalysisUiGraphLines),
				OpcRequestId:              common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.usagecarbonquery.oc1..existing")
	resource.Spec.QueryDefinition.DisplayName = "carbon-updated"
	resource.Spec.QueryDefinition.CostAnalysisUI.Graph = "LINES"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating queryDefinition")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after update follow-up GetUsageCarbonEmissionsQuery confirmation")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateUsageCarbonEmissionsQuery() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetUsageCarbonEmissionsQuery() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.UsageCarbonEmissionsQueryId == nil || *updateRequest.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
		t.Fatalf("update usageCarbonEmissionsQueryId = %v, want existing query OCID", updateRequest.UsageCarbonEmissionsQueryId)
	}
	if updateRequest.UpdateUsageCarbonEmissionsQueryDetails.QueryDefinition == nil || updateRequest.UpdateUsageCarbonEmissionsQueryDetails.QueryDefinition.DisplayName == nil || *updateRequest.UpdateUsageCarbonEmissionsQueryDetails.QueryDefinition.DisplayName != "carbon-updated" {
		t.Fatalf("update queryDefinition.displayName = %v, want %q", updateRequest.UpdateUsageCarbonEmissionsQueryDetails.QueryDefinition, "carbon-updated")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
	if got := resource.Status.QueryDefinition.DisplayName; got != "carbon-updated" {
		t.Fatalf("status.queryDefinition.displayName = %q, want %q", got, "carbon-updated")
	}
	if got := resource.Status.QueryDefinition.CostAnalysisUI.Graph; got != "LINES" {
		t.Fatalf("status.queryDefinition.costAnalysisUI.graph = %q, want %q", got, "LINES")
	}
}

func TestUsageCarbonEmissionsQueryServiceClientRejectsReplacementOnlyCompartmentDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, _ usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		updateUsageCarbonEmissionsQueryFn: func(_ context.Context, _ usageapisdk.UpdateUsageCarbonEmissionsQueryRequest) (usageapisdk.UpdateUsageCarbonEmissionsQueryResponse, error) {
			updateCalled = true
			return usageapisdk.UpdateUsageCarbonEmissionsQueryResponse{}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.usagecarbonquery.oc1..existing")
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateUsageCarbonEmissionsQuery() should not be called when replacement-only drift is detected")
	}
}

func TestUsageCarbonEmissionsQueryServiceClientDeleteRetainsFinalizerUntilReadbackConfirms(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	getCalls := 0
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			getCalls++
			if req.UsageCarbonEmissionsQueryId == nil || *req.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
				t.Fatalf("get usageCarbonEmissionsQueryId = %v, want existing query OCID", req.UsageCarbonEmissionsQueryId)
			}
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		deleteUsageCarbonEmissionsQueryFn: func(_ context.Context, _ usageapisdk.DeleteUsageCarbonEmissionsQueryRequest) (usageapisdk.DeleteUsageCarbonEmissionsQueryResponse, error) {
			deleteCalls++
			return usageapisdk.DeleteUsageCarbonEmissionsQueryResponse{}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.usagecarbonquery.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer while GetUsageCarbonEmissionsQuery still returns the resource")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteUsageCarbonEmissionsQuery() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetUsageCarbonEmissionsQuery() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil until delete is confirmed")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
}

func TestUsageCarbonEmissionsQueryServiceClientDeleteConfirmsNotFound(t *testing.T) {
	t.Parallel()

	var deleteRequest usageapisdk.DeleteUsageCarbonEmissionsQueryRequest
	getCalls := 0
	client := testUsageCarbonEmissionsQueryClient(&fakeUsageCarbonEmissionsQueryOCIClient{
		getUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.GetUsageCarbonEmissionsQueryRequest) (usageapisdk.GetUsageCarbonEmissionsQueryResponse, error) {
			getCalls++
			if req.UsageCarbonEmissionsQueryId == nil || *req.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
				t.Fatalf("get usageCarbonEmissionsQueryId = %v, want existing query OCID", req.UsageCarbonEmissionsQueryId)
			}
			if getCalls > 1 {
				return usageapisdk.GetUsageCarbonEmissionsQueryResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "usage carbon emissions query not found")
			}
			return usageapisdk.GetUsageCarbonEmissionsQueryResponse{
				UsageCarbonEmissionsQuery: makeSDKUsageCarbonEmissionsQuery("ocid1.usagecarbonquery.oc1..existing", "carbon-sample", usageapisdk.CostAnalysisUiGraphBars),
			}, nil
		},
		deleteUsageCarbonEmissionsQueryFn: func(_ context.Context, req usageapisdk.DeleteUsageCarbonEmissionsQueryRequest) (usageapisdk.DeleteUsageCarbonEmissionsQueryResponse, error) {
			deleteRequest = req
			return usageapisdk.DeleteUsageCarbonEmissionsQueryResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeUsageCarbonEmissionsQueryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.usagecarbonquery.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetUsageCarbonEmissionsQuery confirms not found")
	}
	if getCalls != 2 {
		t.Fatalf("GetUsageCarbonEmissionsQuery() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.UsageCarbonEmissionsQueryId == nil || *deleteRequest.UsageCarbonEmissionsQueryId != "ocid1.usagecarbonquery.oc1..existing" {
		t.Fatalf("delete usageCarbonEmissionsQueryId = %v, want existing query OCID", deleteRequest.UsageCarbonEmissionsQueryId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
}
