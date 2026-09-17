/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package environment

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	cloudbridgesdk "github.com/oracle/oci-go-sdk/v65/cloudbridge"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudbridgev1beta1 "github.com/oracle/oci-service-operator/api/cloudbridge/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockEnvironmentID = "ocid1.cloudbridgeenvironment.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal environment contract, and vendored OCI SDK.
func TestMockIntegrationEnvironmentLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &cloudbridgev1beta1.Environment{ObjectMeta: metav1.ObjectMeta{Name: "mock-environment", Namespace: "default", UID: types.UID("mock-environment-uid")}, Spec: cloudbridgev1beta1.EnvironmentSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-environment", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newEnvironmentMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudbridge.mock.invalid", BasePath: "20220509", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &EnvironmentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newEnvironmentRuntimeHooks(manager, cloudbridgesdk.OcbAgentSvcClient{BaseClient: session.BaseClient()})
	client := wrapEnvironmentGeneratedClient(hooks, defaultEnvironmentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*cloudbridgev1beta1.Environment](buildEnvironmentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudbridgev1beta1.Environment]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudbridgev1beta1.Environment) error {
			if current.Status.Id != mockEnvironmentID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(cloudbridgesdk.EnvironmentLifecycleStateActive) {
				return fmt.Errorf("created Environment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudbridgev1beta1.Environment) {
			current.Spec.DisplayName = "mock-environment-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *cloudbridgev1beta1.Environment) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Environment status = %+v", current.Status)
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

func newEnvironmentMockResponder(resource *cloudbridgev1beta1.Environment) (*ocimock.CRUDResponder[cloudbridgesdk.Environment], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[cloudbridgesdk.Environment]{
		CollectionPath: "/20220509/environments", ItemPath: "/20220509/environments/" + mockEnvironmentID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (cloudbridgesdk.Environment, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero cloudbridgesdk.Environment
				return zero, ocimock.Response{}, err
			}
			var details cloudbridgesdk.CreateEnvironmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudbridgesdk.Environment{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return cloudbridgesdk.Environment{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return cloudbridgesdk.Environment{}, ocimock.Response{}, fmt.Errorf("unexpected CreateEnvironment details: %+v", details)
			}
			state := cloudbridgesdk.Environment{Id: common.String(mockEnvironmentID), DisplayName: details.DisplayName, CompartmentId: details.CompartmentId,
				TimeCreated: &now, TimeUpdated: &now, LifecycleState: cloudbridgesdk.EnvironmentLifecycleStateCreating, FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{}}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state cloudbridgesdk.Environment) (cloudbridgesdk.Environment, ocimock.Response, error) {
			switch state.LifecycleState {
			case cloudbridgesdk.EnvironmentLifecycleStateCreating:
				if createRead {
					state.LifecycleState = cloudbridgesdk.EnvironmentLifecycleStateActive
				} else {
					createRead = true
				}
			case cloudbridgesdk.EnvironmentLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = cloudbridgesdk.EnvironmentLifecycleStateActive
				} else {
					updateRead = true
				}
			case cloudbridgesdk.EnvironmentLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = cloudbridgesdk.EnvironmentLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state cloudbridgesdk.Environment) (cloudbridgesdk.Environment, ocimock.Response, error) {
			var details cloudbridgesdk.UpdateEnvironmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudbridgesdk.Environment{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-environment-updated" || details.FreeformTags["osok-mock"] != "update" {
				return cloudbridgesdk.Environment{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateEnvironment details: %+v", details)
			}
			state.DisplayName, state.FreeformTags, state.LifecycleState = details.DisplayName, details.FreeformTags, cloudbridgesdk.EnvironmentLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state cloudbridgesdk.Environment) (cloudbridgesdk.Environment, ocimock.Response, error) {
			state.LifecycleState = cloudbridgesdk.EnvironmentLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
