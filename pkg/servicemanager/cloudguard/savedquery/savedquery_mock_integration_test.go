/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package savedquery

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockSavedQueryID = "ocid1.savedquery.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/cloudguard/savedquery.json
//   - repo-authored runtime: formal/controllers/cloudguard/savedquery/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/cloud_guard/cloud_guard_saved_query_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/cloudguard
func TestMockIntegrationSavedQueryLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &cloudguardv1beta1.SavedQuery{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-saved-query", Namespace: "default", UID: types.UID("mock-saved-query-uid")},
		Spec: cloudguardv1beta1.SavedQuerySpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			DisplayName:   "mock-saved-query",
			Query:         "select name, pid from processes",
			Description:   "mock Cloud Guard saved query",
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newSavedQueryMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard.mock.invalid", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SavedQuery OCI mock: %v", err)
		}
	})

	sdkClient := cloudguardsdk.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &SavedQueryServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSavedQueryRuntimeHooks(manager, sdkClient)
	client := wrapSavedQueryGeneratedClient(hooks, defaultSavedQueryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.SavedQuery](buildSavedQueryGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.SavedQuery]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.SavedQuery) error {
			if current.Status.Id != mockSavedQueryID ||
				current.Status.DisplayName != "mock-saved-query" ||
				current.Status.LifecycleState != string(cloudguardsdk.LifecycleStateActive) ||
				current.Status.Query != resource.Spec.Query {
				return fmt.Errorf("created SavedQuery status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.SavedQuery) {
			current.Spec.Description = "mock Cloud Guard saved query updated"
		},
		ValidateUpdated: func(current *cloudguardv1beta1.SavedQuery) error {
			if current.Status.Description != current.Spec.Description ||
				current.Status.LifecycleState != string(cloudguardsdk.LifecycleStateActive) {
				return fmt.Errorf("updated SavedQuery status = %+v", current.Status)
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

func newSavedQueryMockResponder(resource *cloudguardv1beta1.SavedQuery) (*ocimock.CRUDResponder[cloudguardsdk.SavedQuery], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 4, 17, 0, 0, 0, time.UTC)}
	updatedAt := common.SDKTime{Time: createdAt.Time.Add(time.Minute)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[cloudguardsdk.SavedQuery]{
		CollectionPath:         "/20200131/savedQueries",
		ItemPath:               "/20200131/savedQueries/" + mockSavedQueryID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (cloudguardsdk.SavedQuery, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero cloudguardsdk.SavedQuery
				return zero, ocimock.Response{}, err
			}
			var details cloudguardsdk.CreateSavedQueryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudguardsdk.SavedQuery{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return cloudguardsdk.SavedQuery{}, ocimock.Response{}, err
			}
			expected := cloudguardsdk.CreateSavedQueryDetails{
				CompartmentId: common.String(resource.Spec.CompartmentId),
				DisplayName:   common.String(resource.Spec.DisplayName),
				Query:         common.String(resource.Spec.Query),
				Description:   common.String(resource.Spec.Description),
				FreeformTags:  map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return cloudguardsdk.SavedQuery{}, ocimock.Response{}, fmt.Errorf("create SavedQuery details = %+v, want %+v", details, expected)
			}
			state := cloudguardsdk.SavedQuery{
				Id:             common.String(mockSavedQueryID),
				CompartmentId:  details.CompartmentId,
				DisplayName:    details.DisplayName,
				Query:          details.Query,
				Description:    details.Description,
				LifecycleState: cloudguardsdk.LifecycleStateCreating,
				TimeCreated:    &createdAt,
				TimeUpdated:    &createdAt,
				FreeformTags:   details.FreeformTags,
				DefinedTags:    details.DefinedTags,
				SystemTags:     map[string]map[string]interface{}{},
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state cloudguardsdk.SavedQuery) (cloudguardsdk.SavedQuery, ocimock.Response, error) {
			switch state.LifecycleState {
			case cloudguardsdk.LifecycleStateCreating:
				state.LifecycleState = cloudguardsdk.LifecycleStateActive
			case cloudguardsdk.LifecycleStateDeleting:
				state.LifecycleState = cloudguardsdk.LifecycleStateDeleted
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state cloudguardsdk.SavedQuery) (cloudguardsdk.SavedQuery, ocimock.Response, error) {
			var details cloudguardsdk.UpdateSavedQueryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudguardsdk.SavedQuery{}, ocimock.Response{}, err
			}
			expected := cloudguardsdk.UpdateSavedQueryDetails{
				DisplayName:  common.String("mock-saved-query"),
				Query:        common.String("select name, pid from processes"),
				Description:  common.String("mock Cloud Guard saved query updated"),
				FreeformTags: map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return cloudguardsdk.SavedQuery{}, ocimock.Response{}, fmt.Errorf("update SavedQuery details = %+v, want %+v", details, expected)
			}
			state.Description = details.Description
			state.TimeUpdated = &updatedAt
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(request ocimock.Request, state cloudguardsdk.SavedQuery) (cloudguardsdk.SavedQuery, ocimock.Response, error) {
			if len(request.Body) != 0 {
				return cloudguardsdk.SavedQuery{}, ocimock.Response{}, fmt.Errorf("delete SavedQuery body = %s", request.Body)
			}
			state.LifecycleState = cloudguardsdk.LifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
