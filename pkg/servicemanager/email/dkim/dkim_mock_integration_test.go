/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dkim

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

const mockDkimID = "ocid1.dkim.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/email/dkim and formal/imports/email/dkim.json
//   - resource runtime: dkim_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/email
func TestMockIntegrationDkimLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &emailv1beta1.Dkim{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-dkim", Namespace: "default", UID: types.UID("mock-dkim-uid")},
		Spec: emailv1beta1.DkimSpec{EmailDomainId: "ocid1.emaildomain.oc1..mock", Name: "mock-dkim",
			Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"}},
	}
	responder, err := newDkimMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://ctrl.email.mock.invalid", BasePath: "20170907", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Dkim OCI mock: %v", err)
		}
	})
	manager := &DkimServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDkimRuntimeHooks(manager, emailsdk.EmailClient{BaseClient: session.BaseClient()})
	client := wrapDkimGeneratedClient(hooks, defaultDkimServiceClient{ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.Dkim](buildDkimGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*emailv1beta1.Dkim]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *emailv1beta1.Dkim) error {
			if current.Status.Id != mockDkimID || current.Status.Name != resource.Spec.Name ||
				current.Status.LifecycleState != string(emailsdk.DkimLifecycleStateNeedsAttention) || current.Status.TxtRecordValue == "" {
				return fmt.Errorf("created Dkim status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *emailv1beta1.Dkim) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *emailv1beta1.Dkim) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Dkim status = %+v", current.Status)
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

func newDkimMockResponder(resource *emailv1beta1.Dkim) (*ocimock.CRUDResponder[emailsdk.Dkim], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createReadObserved := false
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[emailsdk.Dkim]{
		CollectionPath: "/20170907/dkims", ItemPath: "/20170907/dkims/" + mockDkimID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (emailsdk.Dkim, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero emailsdk.Dkim
				return zero, ocimock.Response{}, err
			}
			var details emailsdk.CreateDkimDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return emailsdk.Dkim{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return emailsdk.Dkim{}, ocimock.Response{}, err
			}
			if details.EmailDomainId == nil || *details.EmailDomainId != resource.Spec.EmailDomainId ||
				details.Name == nil || *details.Name != resource.Spec.Name || details.Description == nil ||
				*details.Description != resource.Spec.Description {
				return emailsdk.Dkim{}, ocimock.Response{}, fmt.Errorf("unexpected CreateDkim details: %+v", details)
			}
			state := emailsdk.Dkim{Name: details.Name, Id: common.String(mockDkimID), EmailDomainId: details.EmailDomainId,
				CompartmentId: common.String("ocid1.compartment.oc1..mock"), LifecycleState: emailsdk.DkimLifecycleStateCreating,
				Description: details.Description, TimeCreated: &now, TimeUpdated: &now,
				DnsSubdomainName: common.String("mock-dkim._domainkey.mock.example.com."),
				CnameRecordValue: common.String("mock-dkim.mock.example.com.dkim.mock.invalid"),
				FreeformTags:     details.FreeformTags, IsImported: common.Bool(false), KeyLength: common.Int(2048)}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state emailsdk.Dkim) (emailsdk.Dkim, ocimock.Response, error) {
			switch state.LifecycleState {
			case emailsdk.DkimLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = emailsdk.DkimLifecycleStateNeedsAttention
					state.LifecycleDetails = common.String("NEED_DNS")
					state.TxtRecordValue = common.String("v=DKIM1;p=mock")
				} else {
					createReadObserved = true
				}
			case emailsdk.DkimLifecycleStateDeleting:
				if deleteReadObserved {
					state.LifecycleState = emailsdk.DkimLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state emailsdk.Dkim) (emailsdk.Dkim, ocimock.Response, error) {
			var details emailsdk.UpdateDkimDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return emailsdk.Dkim{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return emailsdk.Dkim{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateDkim details: %+v", details)
			}
			state.Description = details.Description
			state.FreeformTags = details.FreeformTags
			state.TimeUpdated = &now
			response, err := ocimock.JSONResponse(http.StatusAccepted, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state emailsdk.Dkim) (emailsdk.Dkim, ocimock.Response, error) {
			state.LifecycleState = emailsdk.DkimLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
