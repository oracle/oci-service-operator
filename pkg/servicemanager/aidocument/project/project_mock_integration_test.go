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

	aidocumentsdk "github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/common"
	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockAIProjectID = "ocid1.aidocumentproject.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal project contract, and vendored OCI SDK.
func TestMockIntegrationProjectLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &aidocumentv1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-ai-document-project", Namespace: "default", UID: types.UID("mock-ai-document-project-uid")},
		Spec: aidocumentv1beta1.ProjectSpec{CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-ai-document-project",
			Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"}},
	}
	responder, err := newAIProjectMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://document.aiservice.mock.invalid", BasePath: "20221109", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newProjectServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, aidocumentsdk.AIServiceDocumentClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*aidocumentv1beta1.Project]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *aidocumentv1beta1.Project) error {
			if current.Status.Id != mockAIProjectID || current.Status.DisplayName != resource.Spec.DisplayName ||
				current.Status.LifecycleState != string(aidocumentsdk.ProjectLifecycleStateActive) {
				return fmt.Errorf("created AI Document Project status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *aidocumentv1beta1.Project) { current.Spec.Description = "mock update" },
		ValidateUpdated: func(current *aidocumentv1beta1.Project) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated AI Document Project status = %+v", current.Status)
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

func newAIProjectMockResponder(resource *aidocumentv1beta1.Project) (*ocimock.CRUDResponder[aidocumentsdk.Project], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[aidocumentsdk.Project]{
		CollectionPath: "/20221109/projects", ItemPath: "/20221109/projects/" + mockAIProjectID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (aidocumentsdk.Project, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero aidocumentsdk.Project
				return zero, ocimock.Response{}, err
			}
			var details aidocumentsdk.CreateProjectDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return aidocumentsdk.Project{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return aidocumentsdk.Project{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return aidocumentsdk.Project{}, ocimock.Response{}, fmt.Errorf("unexpected CreateProject details: %+v", details)
			}
			state := aidocumentsdk.Project{Id: common.String(mockAIProjectID), CompartmentId: details.CompartmentId, TimeCreated: &now,
				LifecycleState: aidocumentsdk.ProjectLifecycleStateCreating, DisplayName: details.DisplayName, Description: details.Description,
				TimeUpdated: &now, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state aidocumentsdk.Project) (aidocumentsdk.Project, ocimock.Response, error) {
			switch state.LifecycleState {
			case aidocumentsdk.ProjectLifecycleStateCreating:
				if createRead {
					state.LifecycleState = aidocumentsdk.ProjectLifecycleStateActive
				} else {
					createRead = true
				}
			case aidocumentsdk.ProjectLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = aidocumentsdk.ProjectLifecycleStateActive
				} else {
					updateRead = true
				}
			case aidocumentsdk.ProjectLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = aidocumentsdk.ProjectLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state aidocumentsdk.Project) (aidocumentsdk.Project, ocimock.Response, error) {
			var details aidocumentsdk.UpdateProjectDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return aidocumentsdk.Project{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags != nil {
				return aidocumentsdk.Project{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateProject details: %+v", details)
			}
			state.Description, state.LifecycleState = details.Description, aidocumentsdk.ProjectLifecycleStateUpdating
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
		DeleteTransition: func(_ ocimock.Request, state aidocumentsdk.Project) (aidocumentsdk.Project, ocimock.Response, error) {
			state.LifecycleState = aidocumentsdk.ProjectLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
