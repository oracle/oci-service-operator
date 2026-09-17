/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package emaildomain

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

const mockEmailDomainID = "ocid1.emaildomain.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/email/emaildomain and formal/imports/email/emaildomain.json
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/email
func TestMockIntegrationEmailDomainLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &emailv1beta1.EmailDomain{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-email-domain", Namespace: "default", UID: types.UID("mock-email-domain-uid")},
		Spec: emailv1beta1.EmailDomainSpec{CompartmentId: "ocid1.compartment.oc1..mock", Name: "mock.example.com",
			Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"}},
	}
	responder, err := newEmailDomainMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://ctrl.email.mock.invalid", BasePath: "20170907", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close EmailDomain OCI mock: %v", err)
		}
	})
	manager := &EmailDomainServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newEmailDomainRuntimeHooks(manager, emailsdk.EmailClient{BaseClient: session.BaseClient()})
	client := wrapEmailDomainGeneratedClient(hooks, defaultEmailDomainServiceClient{ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.EmailDomain](buildEmailDomainGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*emailv1beta1.EmailDomain]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *emailv1beta1.EmailDomain) error {
			if current.Status.Id != mockEmailDomainID || current.Status.Name != resource.Spec.Name ||
				current.Status.LifecycleState != string(emailsdk.EmailDomainLifecycleStateActive) {
				return fmt.Errorf("created EmailDomain status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *emailv1beta1.EmailDomain) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *emailv1beta1.EmailDomain) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated EmailDomain status = %+v", current.Status)
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

func newEmailDomainMockResponder(resource *emailv1beta1.EmailDomain) (*ocimock.CRUDResponder[emailsdk.EmailDomain], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createReadObserved := false
	updateReadObserved := false
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[emailsdk.EmailDomain]{
		CollectionPath: "/20170907/emailDomains", ItemPath: "/20170907/emailDomains/" + mockEmailDomainID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (emailsdk.EmailDomain, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero emailsdk.EmailDomain
				return zero, ocimock.Response{}, err
			}
			var details emailsdk.CreateEmailDomainDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return emailsdk.EmailDomain{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return emailsdk.EmailDomain{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.Name == nil || *details.Name != resource.Spec.Name || details.Description == nil ||
				*details.Description != resource.Spec.Description {
				return emailsdk.EmailDomain{}, ocimock.Response{}, fmt.Errorf("unexpected CreateEmailDomain details: %+v", details)
			}
			state := emailsdk.EmailDomain{Name: details.Name, Id: common.String(mockEmailDomainID), CompartmentId: details.CompartmentId,
				LifecycleState: emailsdk.EmailDomainLifecycleStateCreating, Description: details.Description,
				TimeCreated: &createdAt, FreeformTags: details.FreeformTags, IsSpf: common.Bool(false)}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state emailsdk.EmailDomain) (emailsdk.EmailDomain, ocimock.Response, error) {
			switch state.LifecycleState {
			case emailsdk.EmailDomainLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = emailsdk.EmailDomainLifecycleStateActive
				} else {
					createReadObserved = true
				}
			case emailsdk.EmailDomainLifecycleStateUpdating:
				if updateReadObserved {
					state.LifecycleState = emailsdk.EmailDomainLifecycleStateActive
				} else {
					updateReadObserved = true
				}
			case emailsdk.EmailDomainLifecycleStateDeleting:
				if deleteReadObserved {
					state.LifecycleState = emailsdk.EmailDomainLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state emailsdk.EmailDomain) (emailsdk.EmailDomain, ocimock.Response, error) {
			var details emailsdk.UpdateEmailDomainDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return emailsdk.EmailDomain{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return emailsdk.EmailDomain{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateEmailDomain details: %+v", details)
			}
			state.Description = details.Description
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = emailsdk.EmailDomainLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusAccepted, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state emailsdk.EmailDomain) (emailsdk.EmailDomain, ocimock.Response, error) {
			state.LifecycleState = emailsdk.EmailDomainLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
