/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package customtable

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

const mockCustomTableID = "ocid1.usagecustomtable.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal state-free contract, and vendored OCI SDK.
func TestMockIntegrationCustomTableLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &usageapiv1beta1.CustomTable{ObjectMeta: metav1.ObjectMeta{Name: "mock-custom-table", Namespace: "default", UID: types.UID("mock-custom-table-uid")}, Spec: usageapiv1beta1.CustomTableSpec{
		CompartmentId: "ocid1.tenancy.oc1..mock", SavedReportId: "ocid1.usagesavedquery.oc1..mock",
		SavedCustomTable: usageapiv1beta1.CustomTableSavedCustomTable{DisplayName: "mock-custom-table", RowGroupBy: []string{"service"}, ColumnGroupBy: []string{"region"}, Version: 1},
	}}
	responder, err := newCustomTableMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://usage.mock.invalid", BasePath: "20200107", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &CustomTableServiceManager{}
	hooks := newCustomTableRuntimeHooks(manager, usageapisdk.UsageapiClient{BaseClient: session.BaseClient()})
	client := wrapCustomTableGeneratedClient(hooks, defaultCustomTableServiceClient{ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.CustomTable](buildCustomTableGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*usageapiv1beta1.CustomTable]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *usageapiv1beta1.CustomTable) error {
			if current.Status.Id != mockCustomTableID || current.Status.SavedCustomTable.DisplayName != resource.Spec.SavedCustomTable.DisplayName {
				return fmt.Errorf("created CustomTable status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *usageapiv1beta1.CustomTable) {
			current.Spec.SavedCustomTable.DisplayName = "mock-custom-table-updated"
		},
		ValidateUpdated: func(current *usageapiv1beta1.CustomTable) error {
			if current.Status.SavedCustomTable.DisplayName != current.Spec.SavedCustomTable.DisplayName {
				return fmt.Errorf("updated CustomTable status = %+v", current.Status)
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

func newCustomTableMockResponder(resource *usageapiv1beta1.CustomTable) (*ocimock.CRUDResponder[usageapisdk.CustomTable], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[usageapisdk.CustomTable]{
		CollectionPath: "/20200107/customTables", ItemPath: "/20200107/customTables/" + mockCustomTableID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state usageapisdk.CustomTable) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.CustomTable{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.CustomTable{state}})
		},
		Create: func(request ocimock.Request) (usageapisdk.CustomTable, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero usageapisdk.CustomTable
				return zero, ocimock.Response{}, err
			}
			var details usageapisdk.CreateCustomTableDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.CustomTable{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return usageapisdk.CustomTable{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.SavedReportId == nil || *details.SavedReportId != resource.Spec.SavedReportId || details.SavedCustomTable == nil || details.SavedCustomTable.DisplayName == nil || *details.SavedCustomTable.DisplayName != resource.Spec.SavedCustomTable.DisplayName {
				return usageapisdk.CustomTable{}, ocimock.Response{}, fmt.Errorf("unexpected CreateCustomTable details: %+v", details)
			}
			state := usageapisdk.CustomTable{Id: common.String(mockCustomTableID), SavedReportId: details.SavedReportId, CompartmentId: details.CompartmentId, SavedCustomTable: details.SavedCustomTable}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state usageapisdk.CustomTable) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state usageapisdk.CustomTable) (usageapisdk.CustomTable, ocimock.Response, error) {
			var details usageapisdk.UpdateCustomTableDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.CustomTable{}, ocimock.Response{}, err
			}
			if details.SavedCustomTable == nil || details.SavedCustomTable.DisplayName == nil || *details.SavedCustomTable.DisplayName != "mock-custom-table-updated" {
				return usageapisdk.CustomTable{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateCustomTable details: %+v", details)
			}
			state.SavedCustomTable = details.SavedCustomTable
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ usageapisdk.CustomTable) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
