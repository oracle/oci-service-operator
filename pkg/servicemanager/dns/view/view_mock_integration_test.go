/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package view

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockViewID = "ocid1.dnsview.oc1..mock"

// Contract evidence: recorded View CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationViewLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dnsv1beta1.View{ObjectMeta: metav1.ObjectMeta{Name: "mock-view", Namespace: "default", UID: types.UID("mock-view-uid")}, Spec: dnsv1beta1.ViewSpec{CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-view", FreeformTags: map[string]string{"osok-mock": "create"}}}
	responder, err := newViewMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dns.mock.invalid", BasePath: "20180115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close View OCI mock: %v", err)
		}
	})
	client := newViewServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, dnssdk.DnsClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dnsv1beta1.View]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dnsv1beta1.View) error {
			if current.Status.Id != mockViewID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(dnssdk.ViewLifecycleStateActive) || current.Status.IsProtected {
				return fmt.Errorf("created View status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dnsv1beta1.View) {
			current.Spec.DisplayName = "mock-view-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *dnsv1beta1.View) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated View status = %+v", current.Status)
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

func newViewMockResponder(resource *dnsv1beta1.View) (*ocimock.CRUDResponder[dnssdk.View], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dnssdk.View]{
		CollectionPath: "/20180115/views", ItemPath: "/20180115/views/" + mockViewID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (dnssdk.View, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dnssdk.View
				return zero, ocimock.Response{}, err
			}
			var details dnssdk.CreateViewDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.View{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dnssdk.View{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return dnssdk.View{}, ocimock.Response{}, fmt.Errorf("unexpected CreateView details: %+v", details)
			}
			state := dnssdk.View{CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, FreeformTags: details.FreeformTags,
				DefinedTags: map[string]map[string]interface{}{}, Id: common.String(mockViewID), Self: common.String("https://dns.mock.invalid/views/" + mockViewID),
				TimeCreated: &now, TimeUpdated: &now, LifecycleState: dnssdk.ViewLifecycleStateActive, IsProtected: common.Bool(false)}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dnssdk.View) (dnssdk.View, ocimock.Response, error) {
			if state.LifecycleState == dnssdk.ViewLifecycleStateDeleting && deleteRead {
				response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
				return state, response, err
			}
			if state.LifecycleState == dnssdk.ViewLifecycleStateDeleting {
				deleteRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dnssdk.View) (dnssdk.View, ocimock.Response, error) {
			var details dnssdk.UpdateViewDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.View{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-view-updated" || details.FreeformTags["osok-mock"] != "update" {
				return dnssdk.View{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateView details: %+v", details)
			}
			state.DisplayName, state.FreeformTags = details.DisplayName, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state dnssdk.View) (dnssdk.View, ocimock.Response, error) {
			state.LifecycleState = dnssdk.ViewLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
