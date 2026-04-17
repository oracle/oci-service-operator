/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package customtable

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

type fakeCustomTableOCIClient struct {
	createCustomTableFn func(context.Context, usageapisdk.CreateCustomTableRequest) (usageapisdk.CreateCustomTableResponse, error)
	getCustomTableFn    func(context.Context, usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error)
	listCustomTablesFn  func(context.Context, usageapisdk.ListCustomTablesRequest) (usageapisdk.ListCustomTablesResponse, error)
	updateCustomTableFn func(context.Context, usageapisdk.UpdateCustomTableRequest) (usageapisdk.UpdateCustomTableResponse, error)
	deleteCustomTableFn func(context.Context, usageapisdk.DeleteCustomTableRequest) (usageapisdk.DeleteCustomTableResponse, error)
}

func (f *fakeCustomTableOCIClient) CreateCustomTable(ctx context.Context, req usageapisdk.CreateCustomTableRequest) (usageapisdk.CreateCustomTableResponse, error) {
	if f.createCustomTableFn != nil {
		return f.createCustomTableFn(ctx, req)
	}
	return usageapisdk.CreateCustomTableResponse{}, nil
}

func (f *fakeCustomTableOCIClient) GetCustomTable(ctx context.Context, req usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error) {
	if f.getCustomTableFn != nil {
		return f.getCustomTableFn(ctx, req)
	}
	return usageapisdk.GetCustomTableResponse{}, nil
}

func (f *fakeCustomTableOCIClient) ListCustomTables(ctx context.Context, req usageapisdk.ListCustomTablesRequest) (usageapisdk.ListCustomTablesResponse, error) {
	if f.listCustomTablesFn != nil {
		return f.listCustomTablesFn(ctx, req)
	}
	return usageapisdk.ListCustomTablesResponse{}, nil
}

func (f *fakeCustomTableOCIClient) UpdateCustomTable(ctx context.Context, req usageapisdk.UpdateCustomTableRequest) (usageapisdk.UpdateCustomTableResponse, error) {
	if f.updateCustomTableFn != nil {
		return f.updateCustomTableFn(ctx, req)
	}
	return usageapisdk.UpdateCustomTableResponse{}, nil
}

func (f *fakeCustomTableOCIClient) DeleteCustomTable(ctx context.Context, req usageapisdk.DeleteCustomTableRequest) (usageapisdk.DeleteCustomTableResponse, error) {
	if f.deleteCustomTableFn != nil {
		return f.deleteCustomTableFn(ctx, req)
	}
	return usageapisdk.DeleteCustomTableResponse{}, nil
}

func testCustomTableClient(fake *fakeCustomTableOCIClient) CustomTableServiceClient {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	delegate := defaultCustomTableServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.CustomTable](generatedruntime.Config[*usageapiv1beta1.CustomTable]{
			Kind:      "CustomTable",
			SDKName:   "CustomTable",
			Log:       log,
			Semantics: newCustomTableRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.CreateCustomTableRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.CreateCustomTable(ctx, *request.(*usageapisdk.CreateCustomTableRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateCustomTableDetails", RequestName: "CreateCustomTableDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.GetCustomTableRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.GetCustomTable(ctx, *request.(*usageapisdk.GetCustomTableRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CustomTableId", RequestName: "customTableId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.ListCustomTablesRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.ListCustomTables(ctx, *request.(*usageapisdk.ListCustomTablesRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
					{FieldName: "SavedReportId", RequestName: "savedReportId", Contribution: "query", PreferResourceID: false},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
					{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.UpdateCustomTableRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.UpdateCustomTable(ctx, *request.(*usageapisdk.UpdateCustomTableRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CustomTableId", RequestName: "customTableId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateCustomTableDetails", RequestName: "UpdateCustomTableDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &usageapisdk.DeleteCustomTableRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.DeleteCustomTable(ctx, *request.(*usageapisdk.DeleteCustomTableRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CustomTableId", RequestName: "customTableId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
	return &synchronousCustomTableServiceClient{
		delegate: delegate,
		log:      log,
	}
}

func makeCustomTableResource() *usageapiv1beta1.CustomTable {
	return &usageapiv1beta1.CustomTable{
		Spec: usageapiv1beta1.CustomTableSpec{
			CompartmentId: "ocid1.compartment.oc1..customtableexample",
			SavedReportId: "ocid1.savedreport.oc1..customtableexample",
			SavedCustomTable: usageapiv1beta1.CustomTableSavedCustomTable{
				DisplayName:   "customtable-sample",
				RowGroupBy:    []string{"compartmentId"},
				ColumnGroupBy: []string{"service"},
			},
		},
	}
}

func makeSDKSavedCustomTable(displayName string, rowGroupBy []string, columnGroupBy []string) *usageapisdk.SavedCustomTable {
	return &usageapisdk.SavedCustomTable{
		DisplayName:   common.String(displayName),
		RowGroupBy:    append([]string(nil), rowGroupBy...),
		ColumnGroupBy: append([]string(nil), columnGroupBy...),
	}
}

func makeSDKCustomTable(id string, displayName string, rowGroupBy []string, columnGroupBy []string) usageapisdk.CustomTable {
	return usageapisdk.CustomTable{
		Id:               common.String(id),
		CompartmentId:    common.String("ocid1.compartment.oc1..customtableexample"),
		SavedReportId:    common.String("ocid1.savedreport.oc1..customtableexample"),
		SavedCustomTable: makeSDKSavedCustomTable(displayName, rowGroupBy, columnGroupBy),
	}
}

func makeSDKCustomTableSummary(id string, displayName string, rowGroupBy []string, columnGroupBy []string) usageapisdk.CustomTableSummary {
	return usageapisdk.CustomTableSummary{
		Id:               common.String(id),
		SavedCustomTable: makeSDKSavedCustomTable(displayName, rowGroupBy, columnGroupBy),
	}
}

func TestCustomTableServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest usageapisdk.CreateCustomTableRequest
	getCalls := 0
	client := testCustomTableClient(&fakeCustomTableOCIClient{
		listCustomTablesFn: func(_ context.Context, req usageapisdk.ListCustomTablesRequest) (usageapisdk.ListCustomTablesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..customtableexample" {
				t.Fatalf("list compartmentId = %v, want custom table compartment", req.CompartmentId)
			}
			if req.SavedReportId == nil || *req.SavedReportId != "ocid1.savedreport.oc1..customtableexample" {
				t.Fatalf("list savedReportId = %v, want custom table saved report", req.SavedReportId)
			}
			return usageapisdk.ListCustomTablesResponse{}, nil
		},
		createCustomTableFn: func(_ context.Context, req usageapisdk.CreateCustomTableRequest) (usageapisdk.CreateCustomTableResponse, error) {
			createRequest = req
			return usageapisdk.CreateCustomTableResponse{
				CustomTable:  makeSDKCustomTable("ocid1.customtable.oc1..created", "customtable-sample", []string{"compartmentId"}, []string{"service"}),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getCustomTableFn: func(_ context.Context, req usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error) {
			getCalls++
			if req.CustomTableId == nil || *req.CustomTableId != "ocid1.customtable.oc1..created" {
				t.Fatalf("get customTableId = %v, want created custom table OCID", req.CustomTableId)
			}
			return usageapisdk.GetCustomTableResponse{
				CustomTable: makeSDKCustomTable("ocid1.customtable.oc1..created", "customtable-sample", []string{"compartmentId"}, []string{"service"}),
			}, nil
		},
	})

	resource := makeCustomTableResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after read-after-write GetCustomTable confirmation")
	}
	if createRequest.CreateCustomTableDetails.CompartmentId == nil || *createRequest.CreateCustomTableDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateCustomTableDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateCustomTableDetails.SavedReportId == nil || *createRequest.CreateCustomTableDetails.SavedReportId != resource.Spec.SavedReportId {
		t.Fatalf("create savedReportId = %v, want %q", createRequest.CreateCustomTableDetails.SavedReportId, resource.Spec.SavedReportId)
	}
	if createRequest.CreateCustomTableDetails.SavedCustomTable == nil || createRequest.CreateCustomTableDetails.SavedCustomTable.DisplayName == nil || *createRequest.CreateCustomTableDetails.SavedCustomTable.DisplayName != "customtable-sample" {
		t.Fatalf("create savedCustomTable.displayName = %v, want %q", createRequest.CreateCustomTableDetails.SavedCustomTable, "customtable-sample")
	}
	if getCalls != 1 {
		t.Fatalf("GetCustomTable() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.customtable.oc1..created" {
		t.Fatalf("status.ocid = %q, want created custom table OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != "ocid1.customtable.oc1..created" {
		t.Fatalf("status.id = %q, want created custom table OCID", got)
	}
	if got := resource.Status.SavedReportId; got != resource.Spec.SavedReportId {
		t.Fatalf("status.savedReportId = %q, want %q", got, resource.Spec.SavedReportId)
	}
}

func TestCustomTableServiceClientBindsExistingCustomTableWithoutCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
	getCalls := 0
	client := testCustomTableClient(&fakeCustomTableOCIClient{
		listCustomTablesFn: func(_ context.Context, req usageapisdk.ListCustomTablesRequest) (usageapisdk.ListCustomTablesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..customtableexample" {
				t.Fatalf("list compartmentId = %v, want custom table compartment", req.CompartmentId)
			}
			if req.SavedReportId == nil || *req.SavedReportId != "ocid1.savedreport.oc1..customtableexample" {
				t.Fatalf("list savedReportId = %v, want custom table saved report", req.SavedReportId)
			}
			return usageapisdk.ListCustomTablesResponse{
				CustomTableCollection: usageapisdk.CustomTableCollection{
					Items: []usageapisdk.CustomTableSummary{
						makeSDKCustomTableSummary("ocid1.customtable.oc1..mismatch", "customtable-sample", []string{"tenantId"}, []string{"service"}),
						makeSDKCustomTableSummary("ocid1.customtable.oc1..existing", "customtable-sample", []string{"compartmentId"}, []string{"service"}),
					},
				},
			}, nil
		},
		getCustomTableFn: func(_ context.Context, req usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error) {
			getCalls++
			if req.CustomTableId == nil || *req.CustomTableId != "ocid1.customtable.oc1..existing" {
				t.Fatalf("get customTableId = %v, want existing custom table OCID", req.CustomTableId)
			}
			return usageapisdk.GetCustomTableResponse{
				CustomTable: makeSDKCustomTable("ocid1.customtable.oc1..existing", "customtable-sample", []string{"compartmentId"}, []string{"service"}),
			}, nil
		},
		createCustomTableFn: func(_ context.Context, _ usageapisdk.CreateCustomTableRequest) (usageapisdk.CreateCustomTableResponse, error) {
			createCalled = true
			return usageapisdk.CreateCustomTableResponse{}, nil
		},
	})

	resource := makeCustomTableResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when list lookup reuses an existing custom table")
	}
	if createCalled {
		t.Fatal("CreateCustomTable() should not be called when ListCustomTables finds a matching custom table")
	}
	if getCalls != 2 {
		t.Fatalf("GetCustomTable() calls = %d, want 2 reads of the bound custom table", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.customtable.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing custom table OCID", got)
	}
}

func TestCustomTableServiceClientUpdatesSavedCustomTable(t *testing.T) {
	t.Parallel()

	var updateRequest usageapisdk.UpdateCustomTableRequest
	getCalls := 0
	updateCalls := 0
	client := testCustomTableClient(&fakeCustomTableOCIClient{
		getCustomTableFn: func(_ context.Context, req usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error) {
			getCalls++
			if req.CustomTableId == nil || *req.CustomTableId != "ocid1.customtable.oc1..existing" {
				t.Fatalf("get customTableId = %v, want existing custom table OCID", req.CustomTableId)
			}
			displayName := "legacy-customtable"
			if getCalls > 1 {
				displayName = "customtable-updated"
			}
			return usageapisdk.GetCustomTableResponse{
				CustomTable: makeSDKCustomTable("ocid1.customtable.oc1..existing", displayName, []string{"compartmentId"}, []string{"service"}),
			}, nil
		},
		updateCustomTableFn: func(_ context.Context, req usageapisdk.UpdateCustomTableRequest) (usageapisdk.UpdateCustomTableResponse, error) {
			updateCalls++
			updateRequest = req
			return usageapisdk.UpdateCustomTableResponse{
				CustomTable:  makeSDKCustomTable("ocid1.customtable.oc1..existing", "customtable-updated", []string{"compartmentId"}, []string{"service"}),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeCustomTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.customtable.oc1..existing")
	resource.Spec.SavedCustomTable.DisplayName = "customtable-updated"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating savedCustomTable")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after update follow-up GetCustomTable confirmation")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateCustomTable() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetCustomTable() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.CustomTableId == nil || *updateRequest.CustomTableId != "ocid1.customtable.oc1..existing" {
		t.Fatalf("update customTableId = %v, want existing custom table OCID", updateRequest.CustomTableId)
	}
	if updateRequest.UpdateCustomTableDetails.SavedCustomTable == nil || updateRequest.UpdateCustomTableDetails.SavedCustomTable.DisplayName == nil || *updateRequest.UpdateCustomTableDetails.SavedCustomTable.DisplayName != "customtable-updated" {
		t.Fatalf("update savedCustomTable.displayName = %v, want %q", updateRequest.UpdateCustomTableDetails.SavedCustomTable, "customtable-updated")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
	if got := resource.Status.SavedCustomTable.DisplayName; got != "customtable-updated" {
		t.Fatalf("status.savedCustomTable.displayName = %q, want %q", got, "customtable-updated")
	}
}

func TestCustomTableServiceClientRejectsReplacementOnlySavedReportDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testCustomTableClient(&fakeCustomTableOCIClient{
		getCustomTableFn: func(_ context.Context, _ usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error) {
			return usageapisdk.GetCustomTableResponse{
				CustomTable: makeSDKCustomTable("ocid1.customtable.oc1..existing", "customtable-sample", []string{"compartmentId"}, []string{"service"}),
			}, nil
		},
		updateCustomTableFn: func(_ context.Context, _ usageapisdk.UpdateCustomTableRequest) (usageapisdk.UpdateCustomTableResponse, error) {
			updateCalled = true
			return usageapisdk.UpdateCustomTableResponse{}, nil
		},
	})

	resource := makeCustomTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.customtable.oc1..existing")
	resource.Spec.SavedReportId = "ocid1.savedreport.oc1..replacement"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when savedReportId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateCustomTable() should not be called when replacement-only drift is detected")
	}
}

func TestCustomTableServiceClientDeleteConfirmsNotFound(t *testing.T) {
	t.Parallel()

	var deleteRequest usageapisdk.DeleteCustomTableRequest
	getCalls := 0
	client := testCustomTableClient(&fakeCustomTableOCIClient{
		getCustomTableFn: func(_ context.Context, req usageapisdk.GetCustomTableRequest) (usageapisdk.GetCustomTableResponse, error) {
			getCalls++
			if req.CustomTableId == nil || *req.CustomTableId != "ocid1.customtable.oc1..existing" {
				t.Fatalf("get customTableId = %v, want existing custom table OCID", req.CustomTableId)
			}
			if getCalls > 1 {
				return usageapisdk.GetCustomTableResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "custom table not found")
			}
			return usageapisdk.GetCustomTableResponse{
				CustomTable: makeSDKCustomTable("ocid1.customtable.oc1..existing", "customtable-sample", []string{"compartmentId"}, []string{"service"}),
			}, nil
		},
		deleteCustomTableFn: func(_ context.Context, req usageapisdk.DeleteCustomTableRequest) (usageapisdk.DeleteCustomTableResponse, error) {
			deleteRequest = req
			return usageapisdk.DeleteCustomTableResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeCustomTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.customtable.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetCustomTable confirms not found")
	}
	if getCalls != 2 {
		t.Fatalf("GetCustomTable() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.CustomTableId == nil || *deleteRequest.CustomTableId != "ocid1.customtable.oc1..existing" {
		t.Fatalf("delete customTableId = %v, want existing custom table OCID", deleteRequest.CustomTableId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
}
