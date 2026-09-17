/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dashboardgroup

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockDashboardGroupID = "ocid1.consoledashboardgroup.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal dashboard-group contract, and vendored OCI SDK.
func TestMockIntegrationDashboardGroupLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dashboardservicev1beta1.DashboardGroup{ObjectMeta: metav1.ObjectMeta{Name: "mock-dashboard-group", Namespace: "default", UID: types.UID("mock-dashboard-group-uid")}, Spec: dashboardservicev1beta1.DashboardGroupSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-dashboard-group", Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newDashboardGroupMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dashboard.mock.invalid", BasePath: "20210731", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newDashboardGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, dashboardservicesdk.DashboardGroupClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dashboardservicev1beta1.DashboardGroup]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dashboardservicev1beta1.DashboardGroup) error {
			if current.Status.Id != mockDashboardGroupID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(dashboardservicesdk.DashboardGroupLifecycleStateActive) {
				return fmt.Errorf("created DashboardGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dashboardservicev1beta1.DashboardGroup) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *dashboardservicev1beta1.DashboardGroup) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated DashboardGroup status = %+v", current.Status)
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

func newDashboardGroupMockResponder(resource *dashboardservicev1beta1.DashboardGroup) (*ocimock.CRUDResponder[dashboardservicesdk.DashboardGroup], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dashboardservicesdk.DashboardGroup]{
		CollectionPath: "/20210731/dashboardGroups", ItemPath: "/20210731/dashboardGroups/" + mockDashboardGroupID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (dashboardservicesdk.DashboardGroup, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dashboardservicesdk.DashboardGroup
				return zero, ocimock.Response{}, err
			}
			var details dashboardservicesdk.CreateDashboardGroupDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dashboardservicesdk.DashboardGroup{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dashboardservicesdk.DashboardGroup{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return dashboardservicesdk.DashboardGroup{}, ocimock.Response{}, fmt.Errorf("unexpected CreateDashboardGroup details: %+v", details)
			}
			state := dashboardservicesdk.DashboardGroup{Id: common.String(mockDashboardGroupID), DisplayName: details.DisplayName, Description: details.Description,
				CompartmentId: details.CompartmentId, TimeCreated: &now, TimeUpdated: &now, LifecycleState: dashboardservicesdk.DashboardGroupLifecycleStateCreating,
				FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{}}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dashboardservicesdk.DashboardGroup) (dashboardservicesdk.DashboardGroup, ocimock.Response, error) {
			switch state.LifecycleState {
			case dashboardservicesdk.DashboardGroupLifecycleStateCreating:
				if createRead {
					state.LifecycleState = dashboardservicesdk.DashboardGroupLifecycleStateActive
				} else {
					createRead = true
				}
			case dashboardservicesdk.DashboardGroupLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = dashboardservicesdk.DashboardGroupLifecycleStateActive
				} else {
					updateRead = true
				}
			case dashboardservicesdk.DashboardGroupLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = dashboardservicesdk.DashboardGroupLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dashboardservicesdk.DashboardGroup) (dashboardservicesdk.DashboardGroup, ocimock.Response, error) {
			var details dashboardservicesdk.UpdateDashboardGroupDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dashboardservicesdk.DashboardGroup{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return dashboardservicesdk.DashboardGroup{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateDashboardGroup details: %+v", details)
			}
			state.Description, state.FreeformTags, state.LifecycleState = details.Description, details.FreeformTags, dashboardservicesdk.DashboardGroupLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state dashboardservicesdk.DashboardGroup) (dashboardservicesdk.DashboardGroup, ocimock.Response, error) {
			state.LifecycleState = dashboardservicesdk.DashboardGroupLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
