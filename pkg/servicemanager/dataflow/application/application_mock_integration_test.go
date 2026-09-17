/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package application

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dataflowsdk "github.com/oracle/oci-go-sdk/v65/dataflow"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockApplicationID = "ocid1.dataflowapplication.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal immediate-response contract, resource-local runtime, and vendored OCI SDK.
func TestMockIntegrationApplicationLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataflowv1beta1.Application{ObjectMeta: metav1.ObjectMeta{Name: "mock-data-flow-application", Namespace: "default", UID: types.UID("mock-data-flow-application-uid")}, Spec: dataflowv1beta1.ApplicationSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-data-flow-application",
		DriverShape: "VM.Standard.E4.Flex", ExecutorShape: "VM.Standard.E4.Flex",
		DriverShapeConfig: dataflowv1beta1.ApplicationDriverShapeConfig{Ocpus: 1, MemoryInGBs: 16}, ExecutorShapeConfig: dataflowv1beta1.ApplicationExecutorShapeConfig{Ocpus: 1, MemoryInGBs: 16},
		Language: "PYTHON", NumExecutors: 1, SparkVersion: "3.5.0", FileUri: "oci://mock-bucket@mocknamespace/mock.py", LogsBucketUri: "oci://mock-logs@mocknamespace/",
		Type: "BATCH", Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newApplicationMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dataflow.mock.invalid", BasePath: "20200129", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newApplicationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, dataflowsdk.DataFlowClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataflowv1beta1.Application]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataflowv1beta1.Application) error {
			if current.Status.Id != mockApplicationID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(dataflowsdk.ApplicationLifecycleStateActive) {
				return fmt.Errorf("created Application status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataflowv1beta1.Application) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *dataflowv1beta1.Application) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Application status = %+v", current.Status)
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

func newApplicationMockResponder(resource *dataflowv1beta1.Application) (*ocimock.CRUDResponder[dataflowsdk.Application], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dataflowsdk.Application]{
		CollectionPath: "/20200129/applications", ItemPath: "/20200129/applications/" + mockApplicationID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (dataflowsdk.Application, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dataflowsdk.Application
				return zero, ocimock.Response{}, err
			}
			var details dataflowsdk.CreateApplicationDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dataflowsdk.Application{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dataflowsdk.Application{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.Language != dataflowsdk.ApplicationLanguagePython || details.FileUri == nil || details.DriverShapeConfig == nil || details.ExecutorShapeConfig == nil {
				return dataflowsdk.Application{}, ocimock.Response{}, fmt.Errorf("unexpected CreateApplication details: %+v", details)
			}
			state := dataflowsdk.Application{CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, DriverShape: details.DriverShape, ExecutorShape: details.ExecutorShape,
				FileUri: details.FileUri, Id: common.String(mockApplicationID), Language: details.Language, LifecycleState: dataflowsdk.ApplicationLifecycleStateActive,
				NumExecutors: details.NumExecutors, OwnerPrincipalId: common.String("ocid1.user.oc1..mock"), SparkVersion: details.SparkVersion, TimeCreated: &now, TimeUpdated: &now,
				Description: details.Description, DriverShapeConfig: details.DriverShapeConfig, ExecutorShapeConfig: details.ExecutorShapeConfig,
				FreeformTags: details.FreeformTags, LogsBucketUri: details.LogsBucketUri, Type: details.Type}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dataflowsdk.Application) (dataflowsdk.Application, ocimock.Response, error) {
			if state.LifecycleState == dataflowsdk.ApplicationLifecycleStateDeleting {
				if deleteRead {
					state.LifecycleState = dataflowsdk.ApplicationLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dataflowsdk.Application) (dataflowsdk.Application, ocimock.Response, error) {
			var details dataflowsdk.UpdateApplicationDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dataflowsdk.Application{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return dataflowsdk.Application{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateApplication details: %+v", details)
			}
			state.Description, state.FreeformTags, state.TimeUpdated = details.Description, details.FreeformTags, &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state dataflowsdk.Application) (dataflowsdk.Application, ocimock.Response, error) {
			state.LifecycleState = dataflowsdk.ApplicationLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
