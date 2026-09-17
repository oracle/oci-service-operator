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

	"github.com/oracle/oci-go-sdk/v65/common"
	datasciencesdk "github.com/oracle/oci-go-sdk/v65/datascience"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockDataScienceProjectID = "ocid1.datascienceproject.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal project contract, and vendored OCI SDK.
func TestMockIntegrationProjectLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &datasciencev1beta1.Project{ObjectMeta: metav1.ObjectMeta{Name: "mock-data-science-project", Namespace: "default", UID: types.UID("mock-data-science-project-uid")}, Spec: datasciencev1beta1.ProjectSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-data-science-project", Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newDataScienceProjectMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datascience.mock.invalid", BasePath: "20190101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newMockDataScienceProjectClient(datasciencesdk.DataScienceClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasciencev1beta1.Project]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasciencev1beta1.Project) error {
			if current.Status.Id != mockDataScienceProjectID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(datasciencesdk.ProjectLifecycleStateActive) {
				return fmt.Errorf("created Data Science Project status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasciencev1beta1.Project) {
			current.Spec.DisplayName = "mock-data-science-project-updated"
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasciencev1beta1.Project) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Data Science Project status = %+v", current.Status)
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

func newDataScienceProjectMockResponder(resource *datasciencev1beta1.Project) (*ocimock.CRUDResponder[datasciencesdk.Project], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[datasciencesdk.Project]{
		CollectionPath: "/20190101/projects", ItemPath: "/20190101/projects/" + mockDataScienceProjectID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (datasciencesdk.Project, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero datasciencesdk.Project
				return zero, ocimock.Response{}, err
			}
			var details datasciencesdk.CreateProjectDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return datasciencesdk.Project{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return datasciencesdk.Project{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return datasciencesdk.Project{}, ocimock.Response{}, fmt.Errorf("unexpected CreateProject details: %+v", details)
			}
			state := datasciencesdk.Project{Id: common.String(mockDataScienceProjectID), TimeCreated: &now, DisplayName: details.DisplayName,
				CompartmentId: details.CompartmentId, CreatedBy: common.String("ocid1.user.oc1..mock"), LifecycleState: datasciencesdk.ProjectLifecycleStateActive,
				Description: details.Description, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state datasciencesdk.Project) (datasciencesdk.Project, ocimock.Response, error) {
			if state.LifecycleState == datasciencesdk.ProjectLifecycleStateDeleting && deleteRead {
				state.LifecycleState = datasciencesdk.ProjectLifecycleStateDeleted
			} else if state.LifecycleState == datasciencesdk.ProjectLifecycleStateDeleting {
				deleteRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state datasciencesdk.Project) (datasciencesdk.Project, ocimock.Response, error) {
			var details datasciencesdk.UpdateProjectDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return datasciencesdk.Project{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-data-science-project-updated" || details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return datasciencesdk.Project{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateProject details: %+v", details)
			}
			state.DisplayName, state.Description, state.FreeformTags = details.DisplayName, details.Description, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state datasciencesdk.Project) (datasciencesdk.Project, ocimock.Response, error) {
			state.LifecycleState = datasciencesdk.ProjectLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
