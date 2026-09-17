/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package session

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	generativeaiagentruntimesdk "github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime"
	generativeaiagentruntimev1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagentruntime/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationSessionCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &generativeaiagentruntimev1beta1.Session{}
	ocimock.InitializeResource(resource, "mock-session")
	resource.Spec.AgentEndpointId = "<ocid:1>"
	resource.Spec.DisplayName = "osok mock session"
	resource.Spec.Description = "mock create"
	updatedSpec := resource.Spec
	updatedSpec.DisplayName = "osok mock session updated"
	updatedSpec.Description = "mock update"

	createRequest := generativeaiagentruntimesdk.CreateSessionDetails{DisplayName: stringPtr(resource.Spec.DisplayName), Description: stringPtr(resource.Spec.Description)}
	createdState := generativeaiagentruntimesdk.Session{Id: stringPtr("<ocid:2>"), DisplayName: stringPtr(resource.Spec.DisplayName), Description: stringPtr(resource.Spec.Description)}
	updateRequest := generativeaiagentruntimesdk.UpdateSessionDetails{DisplayName: stringPtr(updatedSpec.DisplayName), Description: stringPtr(updatedSpec.Description)}
	updatedState := createdState
	updatedState.DisplayName, updatedState.Description = updateRequest.DisplayName, updateRequest.Description

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		generativeaiagentruntimesdk.Session,
		generativeaiagentruntimesdk.CreateSessionDetails,
		generativeaiagentruntimesdk.UpdateSessionDetails,
	]{
		CollectionPath: "/20240531/agentEndpoints/<ocid:1>/sessions",
		ItemPath:       "/20240531/agentEndpoints/<ocid:1>/sessions/<ocid:2>",
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		UpdatedReadStates:  []generativeaiagentruntimesdk.Session{updatedState},
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       http.StatusCreated,
		DeleteStatus:       http.StatusNoContent,
		NotFoundCode:       "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://agent-runtime.mock.invalid", BasePath: "20240531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newSessionServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		generativeaiagentruntimesdk.GenerativeAiAgentRuntimeClient{BaseClient: session.BaseClient()},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiagentruntimev1beta1.Session]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiagentruntimev1beta1.Session) error {
			if current.Status.Id != "<ocid:2>" || current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description || current.Status.AgentEndpointId != current.Spec.AgentEndpointId {
				return fmt.Errorf("created Session status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiagentruntimev1beta1.Session) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiagentruntimev1beta1.Session) error {
			if current.Status.Id != "<ocid:2>" || current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Session status = %+v", current.Status)
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

func stringPtr(value string) *string { return &value }
