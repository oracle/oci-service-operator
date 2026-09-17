/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sender

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	emailsdk "github.com/oracle/oci-go-sdk/v65/email"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockSenderID = "ocid1.sender.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/email/sender and formal/imports/email/sender.json
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/email
func TestMockIntegrationSenderLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &emailv1beta1.Sender{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-sender", Namespace: "default", UID: types.UID("mock-sender-uid")},
		Spec: emailv1beta1.SenderSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			EmailAddress:  "mock-sender@example.com",
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newSenderMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://ctrl.email.mock.invalid", BasePath: "20170907", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Sender OCI mock: %v", err)
		}
	})
	manager := &SenderServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSenderRuntimeHooks(manager, emailsdk.EmailClient{BaseClient: session.BaseClient()})
	client := wrapSenderGeneratedClient(hooks, defaultSenderServiceClient{ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.Sender](buildSenderGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*emailv1beta1.Sender]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *emailv1beta1.Sender) error {
			if current.Status.Id != mockSenderID || current.Status.EmailAddress != resource.Spec.EmailAddress ||
				current.Status.LifecycleState != string(emailsdk.SenderLifecycleStateActive) {
				return fmt.Errorf("created Sender status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *emailv1beta1.Sender) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *emailv1beta1.Sender) error {
			if current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Sender status = %+v", current.Status)
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

func newSenderMockResponder(resource *emailv1beta1.Sender) (*ocimock.CRUDResponder[emailsdk.Sender], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, deleteRead := false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[emailsdk.Sender]{
		CollectionPath: "/20170907/senders", ItemPath: "/20170907/senders/" + mockSenderID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (emailsdk.Sender, ocimock.Response, error) {
			var details emailsdk.CreateSenderDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return emailsdk.Sender{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return emailsdk.Sender{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.EmailAddress == nil || *details.EmailAddress != resource.Spec.EmailAddress ||
				details.FreeformTags["osok-mock"] != "create" {
				return emailsdk.Sender{}, ocimock.Response{}, fmt.Errorf("unexpected CreateSender details: %+v", details)
			}
			state := emailsdk.Sender{CompartmentId: details.CompartmentId, EmailAddress: details.EmailAddress,
				Id: common.String(mockSenderID), IsSpf: common.Bool(true), LifecycleState: emailsdk.SenderLifecycleStateCreating,
				TimeCreated: &createdAt, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state emailsdk.Sender) (emailsdk.Sender, ocimock.Response, error) {
			switch state.LifecycleState {
			case emailsdk.SenderLifecycleStateCreating:
				if createRead {
					state.LifecycleState = emailsdk.SenderLifecycleStateActive
				} else {
					createRead = true
				}
			case emailsdk.SenderLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = emailsdk.SenderLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state emailsdk.Sender) (emailsdk.Sender, ocimock.Response, error) {
			var details emailsdk.UpdateSenderDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return emailsdk.Sender{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" || details.EmailIpPoolId != nil {
				return emailsdk.Sender{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateSender details: %+v", details)
			}
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state emailsdk.Sender) (emailsdk.Sender, ocimock.Response, error) {
			state.LifecycleState = emailsdk.SenderLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
