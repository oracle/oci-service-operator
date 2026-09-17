/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package logsavedsearch

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockLogSavedSearchID = "ocid1.logsavedsearch.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, reviewed formal contract, resource-local semantics, and vendored OCI SDK.
func TestMockIntegrationLogSavedSearchLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &loggingv1beta1.LogSavedSearch{ObjectMeta: metav1.ObjectMeta{Name: "mock-log-saved-search", Namespace: "default", UID: types.UID("mock-log-saved-search-uid")}, Spec: loggingv1beta1.LogSavedSearchSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", Name: "mock-log-saved-search", Query: `search "mock-token" | sort by datetime desc`,
		Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newLogSavedSearchMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://logging.mock.invalid", BasePath: "20200531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newTestLogSavedSearchClient(loggingsdk.LoggingManagementClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loggingv1beta1.LogSavedSearch]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loggingv1beta1.LogSavedSearch) error {
			if current.Status.Id != mockLogSavedSearchID || current.Status.Name != resource.Spec.Name || current.Status.LifecycleState != string(loggingsdk.LogSavedSearchLifecycleStateActive) {
				return fmt.Errorf("created LogSavedSearch status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loggingv1beta1.LogSavedSearch) {
			current.Spec.Name = "mock-log-saved-search-updated"
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *loggingv1beta1.LogSavedSearch) error {
			if current.Status.Name != current.Spec.Name || current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated LogSavedSearch status = %+v", current.Status)
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

func newLogSavedSearchMockResponder(resource *loggingv1beta1.LogSavedSearch) (*ocimock.CRUDResponder[loggingsdk.LogSavedSearch], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead := false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[loggingsdk.LogSavedSearch]{
		CollectionPath: "/20200531/logSavedSearches", ItemPath: "/20200531/logSavedSearches/" + mockLogSavedSearchID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		Create: func(request ocimock.Request) (loggingsdk.LogSavedSearch, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero loggingsdk.LogSavedSearch
				return zero, ocimock.Response{}, err
			}
			var details loggingsdk.CreateLogSavedSearchDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return loggingsdk.LogSavedSearch{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return loggingsdk.LogSavedSearch{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.Name == nil || *details.Name != resource.Spec.Name || details.Query == nil || *details.Query != resource.Spec.Query {
				return loggingsdk.LogSavedSearch{}, ocimock.Response{}, fmt.Errorf("unexpected CreateLogSavedSearch details: %+v", details)
			}
			state := loggingsdk.LogSavedSearch{Id: common.String(mockLogSavedSearchID), CompartmentId: details.CompartmentId, Name: details.Name, Query: details.Query,
				TimeCreated: &now, TimeLastModified: &now, Description: details.Description, FreeformTags: details.FreeformTags, LifecycleState: loggingsdk.LogSavedSearchLifecycleStateCreating}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state loggingsdk.LogSavedSearch) (loggingsdk.LogSavedSearch, ocimock.Response, error) {
			switch state.LifecycleState {
			case loggingsdk.LogSavedSearchLifecycleStateCreating:
				if createRead {
					state.LifecycleState = loggingsdk.LogSavedSearchLifecycleStateActive
				} else {
					createRead = true
				}
			case loggingsdk.LogSavedSearchLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = loggingsdk.LogSavedSearchLifecycleStateActive
				} else {
					updateRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state loggingsdk.LogSavedSearch) (loggingsdk.LogSavedSearch, ocimock.Response, error) {
			var details loggingsdk.UpdateLogSavedSearchDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return loggingsdk.LogSavedSearch{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != "mock-log-saved-search-updated" || details.Description == nil || *details.Description != "mock update" || details.Query != nil || details.FreeformTags["osok-mock"] != "update" {
				return loggingsdk.LogSavedSearch{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateLogSavedSearch details: %+v", details)
			}
			state.Name, state.Description, state.FreeformTags, state.LifecycleState = details.Name, details.Description, details.FreeformTags, loggingsdk.LogSavedSearchLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ loggingsdk.LogSavedSearch) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
