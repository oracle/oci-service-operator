/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package table

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	nosqlsdk "github.com/oracle/oci-go-sdk/v65/nosql"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeTableOCIClient struct {
	createTableFn            func(context.Context, nosqlsdk.CreateTableRequest) (nosqlsdk.CreateTableResponse, error)
	getTableFn               func(context.Context, nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error)
	listTablesFn             func(context.Context, nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error)
	updateTableFn            func(context.Context, nosqlsdk.UpdateTableRequest) (nosqlsdk.UpdateTableResponse, error)
	changeTableCompartmentFn func(context.Context, nosqlsdk.ChangeTableCompartmentRequest) (nosqlsdk.ChangeTableCompartmentResponse, error)
	deleteTableFn            func(context.Context, nosqlsdk.DeleteTableRequest) (nosqlsdk.DeleteTableResponse, error)
}

func (f *fakeTableOCIClient) CreateTable(ctx context.Context, req nosqlsdk.CreateTableRequest) (nosqlsdk.CreateTableResponse, error) {
	if f.createTableFn != nil {
		return f.createTableFn(ctx, req)
	}
	return nosqlsdk.CreateTableResponse{}, nil
}

func (f *fakeTableOCIClient) GetTable(ctx context.Context, req nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
	if f.getTableFn != nil {
		return f.getTableFn(ctx, req)
	}
	return nosqlsdk.GetTableResponse{}, nil
}

func (f *fakeTableOCIClient) ListTables(ctx context.Context, req nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error) {
	if f.listTablesFn != nil {
		return f.listTablesFn(ctx, req)
	}
	return nosqlsdk.ListTablesResponse{}, nil
}

func (f *fakeTableOCIClient) UpdateTable(ctx context.Context, req nosqlsdk.UpdateTableRequest) (nosqlsdk.UpdateTableResponse, error) {
	if f.updateTableFn != nil {
		return f.updateTableFn(ctx, req)
	}
	return nosqlsdk.UpdateTableResponse{}, nil
}

func (f *fakeTableOCIClient) ChangeTableCompartment(ctx context.Context, req nosqlsdk.ChangeTableCompartmentRequest) (nosqlsdk.ChangeTableCompartmentResponse, error) {
	if f.changeTableCompartmentFn != nil {
		return f.changeTableCompartmentFn(ctx, req)
	}
	return nosqlsdk.ChangeTableCompartmentResponse{}, nil
}

func (f *fakeTableOCIClient) DeleteTable(ctx context.Context, req nosqlsdk.DeleteTableRequest) (nosqlsdk.DeleteTableResponse, error) {
	if f.deleteTableFn != nil {
		return f.deleteTableFn(ctx, req)
	}
	return nosqlsdk.DeleteTableResponse{}, nil
}

func testTableClient(fake *fakeTableOCIClient) *explicitTableServiceClient {
	return newExplicitTableServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, fake)
}

func makeTableResource() *nosqlv1beta1.Table {
	return &nosqlv1beta1.Table{
		Spec: nosqlv1beta1.TableSpec{
			Name:          "orders",
			CompartmentId: "ocid1.compartment.oc1..example",
			DdlStatement:  "CREATE TABLE orders (id INTEGER, PRIMARY KEY(id))",
		},
	}
}

func makeSDKTable(id string, compartmentID string, state nosqlsdk.TableLifecycleStateEnum) nosqlsdk.Table {
	return nosqlsdk.Table{
		Id:             common.String(id),
		Name:           common.String("orders"),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
		DdlStatement:   common.String("CREATE TABLE orders (id INTEGER, PRIMARY KEY(id))"),
	}
}

func makeSDKSummary(id string, compartmentID string, state nosqlsdk.TableLifecycleStateEnum) nosqlsdk.TableSummary {
	return nosqlsdk.TableSummary{
		Id:             common.String(id),
		Name:           common.String("orders"),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
	}
}

func TestExplicitTableServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest nosqlsdk.CreateTableRequest
	listCount := 0
	client := testTableClient(&fakeTableOCIClient{
		listTablesFn: func(_ context.Context, req nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error) {
			listCount++
			if req.LifecycleState != nosqlsdk.ListTablesLifecycleStateAll {
				t.Fatalf("list lifecycleState = %q, want ALL", req.LifecycleState)
			}
			if listCount == 1 {
				return nosqlsdk.ListTablesResponse{}, nil
			}
			return nosqlsdk.ListTablesResponse{
				TableCollection: nosqlsdk.TableCollection{
					Items: []nosqlsdk.TableSummary{makeSDKSummary("ocid1.table.oc1..created", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateActive)},
				},
			}, nil
		},
		createTableFn: func(_ context.Context, req nosqlsdk.CreateTableRequest) (nosqlsdk.CreateTableResponse, error) {
			createRequest = req
			return nosqlsdk.CreateTableResponse{}, nil
		},
		getTableFn: func(_ context.Context, req nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
			if req.TableNameOrId == nil || *req.TableNameOrId != "orders" {
				t.Fatalf("get TableNameOrId = %v, want orders", req.TableNameOrId)
			}
			return nosqlsdk.GetTableResponse{
				Table: makeSDKTable("ocid1.table.oc1..created", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateActive),
			}, nil
		},
	})

	resource := makeTableResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if createRequest.CreateTableDetails.Name == nil || *createRequest.CreateTableDetails.Name != "orders" {
		t.Fatalf("create name = %v, want orders", createRequest.CreateTableDetails.Name)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.table.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.Id != "ocid1.table.oc1..created" {
		t.Fatalf("status.id = %q, want created OCID", resource.Status.Id)
	}
	if got := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type; got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestExplicitTableServiceClientBindsExistingTableWithoutCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
	updateCalled := false
	client := testTableClient(&fakeTableOCIClient{
		listTablesFn: func(_ context.Context, _ nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error) {
			return nosqlsdk.ListTablesResponse{
				TableCollection: nosqlsdk.TableCollection{
					Items: []nosqlsdk.TableSummary{makeSDKSummary("ocid1.table.oc1..existing", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateActive)},
				},
			}, nil
		},
		getTableFn: func(_ context.Context, req nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
			if req.TableNameOrId == nil || *req.TableNameOrId != "ocid1.table.oc1..existing" {
				t.Fatalf("get TableNameOrId = %v, want existing OCID", req.TableNameOrId)
			}
			return nosqlsdk.GetTableResponse{
				Table: makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateActive),
			}, nil
		},
		createTableFn: func(_ context.Context, _ nosqlsdk.CreateTableRequest) (nosqlsdk.CreateTableResponse, error) {
			createCalled = true
			return nosqlsdk.CreateTableResponse{}, nil
		},
		updateTableFn: func(_ context.Context, _ nosqlsdk.UpdateTableRequest) (nosqlsdk.UpdateTableResponse, error) {
			updateCalled = true
			return nosqlsdk.UpdateTableResponse{}, nil
		},
	})

	resource := makeTableResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalled {
		t.Fatal("CreateTable() should not be called when an exact-name table already exists")
	}
	if updateCalled {
		t.Fatal("UpdateTable() should not be called without mutable drift")
	}
}

func TestExplicitTableServiceClientRejectsForceNewMutation(t *testing.T) {
	t.Parallel()

	client := testTableClient(&fakeTableOCIClient{
		getTableFn: func(_ context.Context, _ nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
			table := makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateActive)
			table.IsAutoReclaimable = common.Bool(false)
			return nosqlsdk.GetTableResponse{Table: table}, nil
		},
	})

	resource := makeTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.table.oc1..existing")
	resource.Spec.IsAutoReclaimable = true

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "replacement when isAutoReclaimable changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new validation error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure on force-new mutation")
	}
}

func TestExplicitTableServiceClientMovesCompartmentBeforeMutableUpdate(t *testing.T) {
	t.Parallel()

	var calls []string
	var updateRequest nosqlsdk.UpdateTableRequest
	getCount := 0

	client := testTableClient(&fakeTableOCIClient{
		getTableFn: func(_ context.Context, _ nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
			getCount++
			calls = append(calls, "get")
			switch getCount {
			case 1:
				table := makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..old", nosqlsdk.TableLifecycleStateActive)
				table.TableLimits = &nosqlsdk.TableLimits{
					MaxReadUnits:    common.Int(10),
					MaxWriteUnits:   common.Int(10),
					MaxStorageInGBs: common.Int(10),
				}
				return nosqlsdk.GetTableResponse{Table: table}, nil
			case 2:
				table := makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..new", nosqlsdk.TableLifecycleStateActive)
				table.TableLimits = &nosqlsdk.TableLimits{
					MaxReadUnits:    common.Int(10),
					MaxWriteUnits:   common.Int(10),
					MaxStorageInGBs: common.Int(10),
				}
				return nosqlsdk.GetTableResponse{Table: table}, nil
			default:
				table := makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..new", nosqlsdk.TableLifecycleStateUpdating)
				table.TableLimits = &nosqlsdk.TableLimits{
					MaxReadUnits:    common.Int(20),
					MaxWriteUnits:   common.Int(10),
					MaxStorageInGBs: common.Int(10),
				}
				return nosqlsdk.GetTableResponse{Table: table}, nil
			}
		},
		changeTableCompartmentFn: func(_ context.Context, req nosqlsdk.ChangeTableCompartmentRequest) (nosqlsdk.ChangeTableCompartmentResponse, error) {
			calls = append(calls, "change")
			if req.ChangeTableCompartmentDetails.ToCompartmentId == nil || *req.ChangeTableCompartmentDetails.ToCompartmentId != "ocid1.compartment.oc1..new" {
				t.Fatalf("change to compartment = %v, want new compartment", req.ChangeTableCompartmentDetails.ToCompartmentId)
			}
			return nosqlsdk.ChangeTableCompartmentResponse{}, nil
		},
		updateTableFn: func(_ context.Context, req nosqlsdk.UpdateTableRequest) (nosqlsdk.UpdateTableResponse, error) {
			calls = append(calls, "update")
			updateRequest = req
			return nosqlsdk.UpdateTableResponse{}, nil
		},
	})

	resource := makeTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.table.oc1..existing")
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	resource.Spec.TableLimits = nosqlv1beta1.TableLimits{
		MaxReadUnits:    20,
		MaxWriteUnits:   10,
		MaxStorageInGBs: 10,
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while the OCI update is still progressing")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle is UPDATING")
	}
	if want := []string{"get", "change", "get", "update", "get"}; len(calls) != len(want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	} else {
		for i := range want {
			if calls[i] != want[i] {
				t.Fatalf("calls[%d] = %q, want %q (%#v)", i, calls[i], want[i], calls)
			}
		}
	}
	if updateRequest.UpdateTableDetails.TableLimits == nil || *updateRequest.UpdateTableDetails.TableLimits.MaxReadUnits != 20 {
		t.Fatalf("update table limits = %#v, want maxReadUnits=20", updateRequest.UpdateTableDetails.TableLimits)
	}
}

func TestExplicitTableServiceClientDeleteConfirmsProgress(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	getCount := 0
	client := testTableClient(&fakeTableOCIClient{
		getTableFn: func(_ context.Context, _ nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
			getCount++
			if getCount == 1 {
				return nosqlsdk.GetTableResponse{
					Table: makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateActive),
				}, nil
			}
			return nosqlsdk.GetTableResponse{
				Table: makeSDKTable("ocid1.table.oc1..existing", "ocid1.compartment.oc1..example", nosqlsdk.TableLifecycleStateDeleting),
			}, nil
		},
		deleteTableFn: func(_ context.Context, req nosqlsdk.DeleteTableRequest) (nosqlsdk.DeleteTableResponse, error) {
			deleteCalled = true
			if req.TableNameOrId == nil || *req.TableNameOrId != "ocid1.table.oc1..existing" {
				t.Fatalf("delete TableNameOrId = %v, want existing OCID", req.TableNameOrId)
			}
			return nosqlsdk.DeleteTableResponse{}, nil
		},
	})

	resource := makeTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.table.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer while OCI reports DELETING")
	}
	if !deleteCalled {
		t.Fatal("DeleteTable() should be called for an existing table")
	}
	if got := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type; got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
}

func TestExplicitTableServiceClientTreatsDeleteNotFoundAsSuccess(t *testing.T) {
	t.Parallel()

	client := testTableClient(&fakeTableOCIClient{
		getTableFn: func(_ context.Context, _ nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
			return nosqlsdk.GetTableResponse{}, errors.New("http status code: 404")
		},
	})

	resource := makeTableResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.table.oc1..missing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should treat missing OCI tables as deleted")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
}
