/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dashboardservicesdk "github.com/oracle/oci-go-sdk/v65/dashboardservice"
	dashboardservicev1beta1 "github.com/oracle/oci-service-operator/api/dashboardservice/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockDashboardID = "ocid1.consoledashboard.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal polymorphic dashboard contract, and vendored OCI SDK.
func TestMockIntegrationDashboardLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dashboardservicev1beta1.Dashboard{ObjectMeta: metav1.ObjectMeta{Name: "mock-dashboard", Namespace: "default", UID: types.UID("mock-dashboard-uid")}, Spec: dashboardservicev1beta1.DashboardSpec{
		DashboardGroupId: "ocid1.consoledashboardgroup.oc1..mock", SchemaVersion: "V1", DisplayName: "mock-dashboard", Description: "mock create",
		Config: dashboardJSONValue(`{"layout":"grid"}`), Widgets: []shared.JSONValue{dashboardJSONValue(`{"name":"mock-widget"}`)}, FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newDashboardMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dashboard.mock.invalid", BasePath: "20210731", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newDashboardServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, dashboardservicesdk.DashboardClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dashboardservicev1beta1.Dashboard]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dashboardservicev1beta1.Dashboard) error {
			if current.Status.Id != mockDashboardID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(dashboardservicesdk.DashboardLifecycleStateActive) {
				return fmt.Errorf("created Dashboard status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dashboardservicev1beta1.Dashboard) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *dashboardservicev1beta1.Dashboard) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Dashboard status = %+v", current.Status)
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

func newDashboardMockResponder(resource *dashboardservicev1beta1.Dashboard) (*ocimock.CRUDResponder[dashboardservicesdk.V1Dashboard], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dashboardservicesdk.V1Dashboard]{
		CollectionPath: "/20210731/dashboards", ItemPath: "/20210731/dashboards/" + mockDashboardID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (dashboardservicesdk.V1Dashboard, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dashboardservicesdk.V1Dashboard
				return zero, ocimock.Response{}, err
			}
			var envelope struct {
				dashboardservicesdk.CreateV1DashboardDetails
				SchemaVersion string `json:"schemaVersion"`
			}
			if err := ocimock.DecodeJSONRequest(request, &envelope); err != nil {
				return dashboardservicesdk.V1Dashboard{}, ocimock.Response{}, err
			}
			details := envelope.CreateV1DashboardDetails
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dashboardservicesdk.V1Dashboard{}, ocimock.Response{}, err
			}
			if envelope.SchemaVersion != "V1" || details.DashboardGroupId == nil || *details.DashboardGroupId != resource.Spec.DashboardGroupId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || len(details.Widgets) != 1 {
				return dashboardservicesdk.V1Dashboard{}, ocimock.Response{}, fmt.Errorf("unexpected CreateDashboard details: %+v", details)
			}
			state := dashboardservicesdk.V1Dashboard{Id: common.String(mockDashboardID), DashboardGroupId: details.DashboardGroupId, DisplayName: details.DisplayName,
				Description: details.Description, CompartmentId: common.String("ocid1.compartment.oc1..mock"), TimeCreated: &now, TimeUpdated: &now,
				FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{}, Widgets: details.Widgets, Config: details.Config,
				LifecycleState: dashboardservicesdk.DashboardLifecycleStateCreating}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dashboardservicesdk.V1Dashboard) (dashboardservicesdk.V1Dashboard, ocimock.Response, error) {
			switch state.LifecycleState {
			case dashboardservicesdk.DashboardLifecycleStateCreating:
				if createRead {
					state.LifecycleState = dashboardservicesdk.DashboardLifecycleStateActive
				} else {
					createRead = true
				}
			case dashboardservicesdk.DashboardLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = dashboardservicesdk.DashboardLifecycleStateActive
				} else {
					updateRead = true
				}
			case dashboardservicesdk.DashboardLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = dashboardservicesdk.DashboardLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dashboardservicesdk.V1Dashboard) (dashboardservicesdk.V1Dashboard, ocimock.Response, error) {
			var envelope struct {
				dashboardservicesdk.UpdateV1DashboardDetails
				SchemaVersion string `json:"schemaVersion"`
			}
			if err := ocimock.DecodeJSONRequest(request, &envelope); err != nil {
				return dashboardservicesdk.V1Dashboard{}, ocimock.Response{}, err
			}
			details := envelope.UpdateV1DashboardDetails
			if envelope.SchemaVersion != "V1" || details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return dashboardservicesdk.V1Dashboard{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateDashboard details: %+v", details)
			}
			state.Description, state.FreeformTags, state.LifecycleState = details.Description, details.FreeformTags, dashboardservicesdk.DashboardLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state dashboardservicesdk.V1Dashboard) (dashboardservicesdk.V1Dashboard, ocimock.Response, error) {
			state.LifecycleState = dashboardservicesdk.DashboardLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
