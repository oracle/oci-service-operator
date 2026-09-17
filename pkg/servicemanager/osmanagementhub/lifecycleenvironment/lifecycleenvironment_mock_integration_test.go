/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package lifecycleenvironment

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockLifecycleEnvironmentID = "ocid1.osmhlifecycleenvironment.oc1..mock"

// Contract evidence: recorded lifecycle-environment CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationLifecycleEnvironmentLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &osmanagementhubv1beta1.LifecycleEnvironment{ObjectMeta: metav1.ObjectMeta{Name: "mock-lifecycle-environment", Namespace: "default", UID: types.UID("mock-lifecycle-environment-uid")}, Spec: osmanagementhubv1beta1.LifecycleEnvironmentSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-lifecycle-environment", Description: "mock create",
		Stages:   []osmanagementhubv1beta1.LifecycleEnvironmentStage{{DisplayName: "Development", Rank: 1}, {DisplayName: "Production", Rank: 2}},
		ArchType: "X86_64", OsFamily: "ORACLE_LINUX_8", VendorName: "ORACLE", Location: "OCI_COMPUTE",
	}}
	responder, err := newLifecycleEnvironmentMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://osmh.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close LifecycleEnvironment OCI mock: %v", err)
		}
	})
	client := newLifecycleEnvironmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, osmanagementhubsdk.LifecycleEnvironmentClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*osmanagementhubv1beta1.LifecycleEnvironment]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *osmanagementhubv1beta1.LifecycleEnvironment) error {
			if current.Status.Id != mockLifecycleEnvironmentID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive) {
				return fmt.Errorf("created LifecycleEnvironment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *osmanagementhubv1beta1.LifecycleEnvironment) {
			current.Spec.DisplayName = "mock-lifecycle-environment-updated"
			current.Spec.Description = "mock update"
		},
		ValidateUpdated: func(current *osmanagementhubv1beta1.LifecycleEnvironment) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated LifecycleEnvironment status = %+v", current.Status)
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

func newLifecycleEnvironmentMockResponder(resource *osmanagementhubv1beta1.LifecycleEnvironment) (*ocimock.CRUDResponder[osmanagementhubsdk.LifecycleEnvironment], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[osmanagementhubsdk.LifecycleEnvironment]{
		CollectionPath: "/20220901/lifecycleEnvironments", ItemPath: "/20220901/lifecycleEnvironments/" + mockLifecycleEnvironmentID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state osmanagementhubsdk.LifecycleEnvironment) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.LifecycleEnvironment{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.LifecycleEnvironment{state}})
		},
		Create: func(request ocimock.Request) (osmanagementhubsdk.LifecycleEnvironment, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero osmanagementhubsdk.LifecycleEnvironment
				return zero, ocimock.Response{}, err
			}
			var details osmanagementhubsdk.CreateLifecycleEnvironmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return osmanagementhubsdk.LifecycleEnvironment{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return osmanagementhubsdk.LifecycleEnvironment{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || len(details.Stages) != 2 || details.ArchType != osmanagementhubsdk.ArchTypeX8664 || details.Location != osmanagementhubsdk.ManagedInstanceLocationOciCompute {
				return osmanagementhubsdk.LifecycleEnvironment{}, ocimock.Response{}, fmt.Errorf("unexpected CreateLifecycleEnvironment details: %+v", details)
			}
			stages := make([]osmanagementhubsdk.LifecycleStage, len(details.Stages))
			for i, stage := range details.Stages {
				stages[i] = osmanagementhubsdk.LifecycleStage{CompartmentId: details.CompartmentId, DisplayName: stage.DisplayName, Rank: stage.Rank, Id: common.String(fmt.Sprintf("ocid1.osmhlifecyclestage.oc1..mock-%d", i+1)), LifecycleEnvironmentId: common.String(mockLifecycleEnvironmentID), OsFamily: details.OsFamily, ArchType: details.ArchType, VendorName: details.VendorName, Location: details.Location, TimeCreated: &now, LifecycleState: osmanagementhubsdk.LifecycleStageLifecycleStateActive}
			}
			state := osmanagementhubsdk.LifecycleEnvironment{Id: common.String(mockLifecycleEnvironmentID), CompartmentId: details.CompartmentId, DisplayName: details.DisplayName,
				Stages: stages, LifecycleState: osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive, OsFamily: details.OsFamily, ArchType: details.ArchType,
				VendorName: details.VendorName, TimeCreated: &now, Description: details.Description, Location: details.Location, TimeModified: &now,
				FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state osmanagementhubsdk.LifecycleEnvironment) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state osmanagementhubsdk.LifecycleEnvironment) (osmanagementhubsdk.LifecycleEnvironment, ocimock.Response, error) {
			var details osmanagementhubsdk.UpdateLifecycleEnvironmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return osmanagementhubsdk.LifecycleEnvironment{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-lifecycle-environment-updated" || details.Description == nil || *details.Description != "mock update" {
				return osmanagementhubsdk.LifecycleEnvironment{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateLifecycleEnvironment details: %+v", details)
			}
			state.DisplayName, state.Description, state.TimeModified = details.DisplayName, details.Description, &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ osmanagementhubsdk.LifecycleEnvironment) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
