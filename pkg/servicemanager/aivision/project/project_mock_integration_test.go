/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	aivisionsdk "github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/common"
	aivisionv1beta1 "github.com/oracle/oci-service-operator/api/aivision/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockVisionProjectID = "ocid1.aivisionproject.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal project contract, and vendored OCI SDK.
func TestMockIntegrationProjectLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &aivisionv1beta1.Project{ObjectMeta: metav1.ObjectMeta{Name: "mock-ai-vision-project", Namespace: "default", UID: types.UID("mock-ai-vision-project-uid")}, Spec: aivisionv1beta1.ProjectSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-ai-vision-project", Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newVisionProjectMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://vision.aiservice.mock.invalid", BasePath: "20220125", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := defaultProjectServiceClient{ServiceClient: generatedruntime.NewServiceClient[*aivisionv1beta1.Project](newProjectRuntimeConfig(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, aivisionsdk.AIServiceVisionClient{BaseClient: session.BaseClient()}))}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*aivisionv1beta1.Project]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *aivisionv1beta1.Project) error {
			if current.Status.Id != mockVisionProjectID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(aivisionsdk.ProjectLifecycleStateActive) {
				return fmt.Errorf("created AI Vision Project status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *aivisionv1beta1.Project) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *aivisionv1beta1.Project) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated AI Vision Project status = %+v", current.Status)
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

func newVisionProjectMockResponder(resource *aivisionv1beta1.Project) (*ocimock.CRUDResponder[aivisionsdk.Project], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[aivisionsdk.Project]{
		CollectionPath: "/20220125/projects", ItemPath: "/20220125/projects/" + mockVisionProjectID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (aivisionsdk.Project, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero aivisionsdk.Project
				return zero, ocimock.Response{}, err
			}
			var details aivisionsdk.CreateProjectDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return aivisionsdk.Project{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return aivisionsdk.Project{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return aivisionsdk.Project{}, ocimock.Response{}, fmt.Errorf("unexpected CreateProject details: %+v", details)
			}
			state := aivisionsdk.Project{Id: common.String(mockVisionProjectID), CompartmentId: details.CompartmentId, TimeCreated: &now,
				LifecycleState: aivisionsdk.ProjectLifecycleStateCreating, DisplayName: details.DisplayName, Description: details.Description,
				TimeUpdated: &now, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state aivisionsdk.Project) (aivisionsdk.Project, ocimock.Response, error) {
			switch state.LifecycleState {
			case aivisionsdk.ProjectLifecycleStateCreating:
				if createRead {
					state.LifecycleState = aivisionsdk.ProjectLifecycleStateActive
				} else {
					createRead = true
				}
			case aivisionsdk.ProjectLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = aivisionsdk.ProjectLifecycleStateActive
				} else {
					updateRead = true
				}
			case aivisionsdk.ProjectLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = aivisionsdk.ProjectLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state aivisionsdk.Project) (aivisionsdk.Project, ocimock.Response, error) {
			var details aivisionsdk.UpdateProjectDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return aivisionsdk.Project{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return aivisionsdk.Project{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateProject details: %+v", details)
			}
			state.Description, state.FreeformTags, state.LifecycleState = details.Description, details.FreeformTags, aivisionsdk.ProjectLifecycleStateUpdating
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
		DeleteTransition: func(_ ocimock.Request, state aivisionsdk.Project) (aivisionsdk.Project, ocimock.Response, error) {
			state.LifecycleState = aivisionsdk.ProjectLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
