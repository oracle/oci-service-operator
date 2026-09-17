/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package catalog

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datacatalogsdk "github.com/oracle/oci-go-sdk/v65/datacatalog"
	datacatalogv1beta1 "github.com/oracle/oci-service-operator/api/datacatalog/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockCatalogID = "ocid1.datacatalog.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/datacatalog/catalog and formal/imports/datacatalog/catalog.json
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/datacatalog
func TestMockIntegrationCatalogLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &datacatalogv1beta1.Catalog{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-catalog", Namespace: "default", UID: types.UID("mock-catalog-uid")},
		Spec: datacatalogv1beta1.CatalogSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			DisplayName:   "mock-catalog",
		},
	}
	responder, err := newCatalogMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host:      "https://datacatalog.mock.invalid",
		BasePath:  "20190325",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Catalog OCI mock: %v", err)
		}
	})

	sdkClient := datacatalogsdk.DataCatalogClient{BaseClient: session.BaseClient()}
	manager := &CatalogServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newCatalogRuntimeHooks(manager, sdkClient)
	client := wrapCatalogGeneratedClient(hooks, defaultCatalogServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datacatalogv1beta1.Catalog](buildCatalogGeneratedRuntimeConfig(manager, hooks)),
	})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datacatalogv1beta1.Catalog]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datacatalogv1beta1.Catalog) error {
			if current.Status.Id != mockCatalogID ||
				current.Status.DisplayName != "mock-catalog" ||
				current.Status.LifecycleState != string(datacatalogsdk.LifecycleStateActive) {
				return fmt.Errorf("created Catalog status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datacatalogv1beta1.Catalog) {
			current.Spec.DisplayName = "mock-catalog-updated"
		},
		ValidateUpdated: func(current *datacatalogv1beta1.Catalog) error {
			if current.Status.Id != mockCatalogID ||
				current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.LifecycleState != string(datacatalogsdk.LifecycleStateActive) {
				return fmt.Errorf("updated Catalog status = %+v", current.Status)
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

func newCatalogMockResponder(resource *datacatalogv1beta1.Catalog) (*ocimock.CRUDResponder[datacatalogsdk.Catalog], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[datacatalogsdk.Catalog]{
		CollectionPath:     "/20190325/catalogs",
		ItemPath:           "/20190325/catalogs/" + mockCatalogID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(request ocimock.Request, present bool, state datacatalogsdk.Catalog) (ocimock.Response, error) {
			if got := request.URL.Query().Get("compartmentId"); got != resource.Spec.CompartmentId {
				return ocimock.Response{}, fmt.Errorf("ListCatalogs compartmentId = %q", got)
			}
			if got := request.URL.Query().Get("displayName"); got != resource.Spec.DisplayName {
				return ocimock.Response{}, fmt.Errorf("ListCatalogs displayName = %q", got)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []datacatalogsdk.CatalogSummary{})
			}
			return ocimock.JSONResponse(http.StatusOK, []datacatalogsdk.CatalogSummary{catalogSummaryFromMockState(state)})
		},
		Create: func(request ocimock.Request) (datacatalogsdk.Catalog, ocimock.Response, error) {
			var details datacatalogsdk.CreateCatalogDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return datacatalogsdk.Catalog{}, ocimock.Response{}, err
			}
			expected := datacatalogsdk.CreateCatalogDetails{
				CompartmentId: common.String(resource.Spec.CompartmentId),
				DisplayName:   common.String(resource.Spec.DisplayName),
			}
			if !reflect.DeepEqual(details, expected) {
				return datacatalogsdk.Catalog{}, ocimock.Response{}, fmt.Errorf("CreateCatalog details = %+v, want %+v", details, expected)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return datacatalogsdk.Catalog{}, ocimock.Response{}, fmt.Errorf("CreateCatalog opc-retry-token is empty")
			}
			state := datacatalogsdk.Catalog{
				Id:             common.String(mockCatalogID),
				CompartmentId:  details.CompartmentId,
				DisplayName:    details.DisplayName,
				LifecycleState: datacatalogsdk.LifecycleStateActive,
			}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state datacatalogsdk.Catalog) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state datacatalogsdk.Catalog) (datacatalogsdk.Catalog, ocimock.Response, error) {
			var details datacatalogsdk.UpdateCatalogDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return datacatalogsdk.Catalog{}, ocimock.Response{}, err
			}
			expected := datacatalogsdk.UpdateCatalogDetails{DisplayName: common.String("mock-catalog-updated")}
			if !reflect.DeepEqual(details, expected) {
				return datacatalogsdk.Catalog{}, ocimock.Response{}, fmt.Errorf("UpdateCatalog details = %+v, want %+v", details, expected)
			}
			state.DisplayName = details.DisplayName
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ datacatalogsdk.Catalog) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("DeleteCatalog body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

func catalogSummaryFromMockState(state datacatalogsdk.Catalog) datacatalogsdk.CatalogSummary {
	return datacatalogsdk.CatalogSummary{
		Id:             state.Id,
		CompartmentId:  state.CompartmentId,
		DisplayName:    state.DisplayName,
		LifecycleState: state.LifecycleState,
	}
}
