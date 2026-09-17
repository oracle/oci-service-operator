/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package tsigkey

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockTsigKeyID = "ocid1.dnstsigkey.oc1..mock"

// Contract evidence: recorded TSIG CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationTsigKeyLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dnsv1beta1.TsigKey{ObjectMeta: metav1.ObjectMeta{Name: "mock-tsig-key", Namespace: "default", UID: types.UID("mock-tsig-key-uid")}, Spec: dnsv1beta1.TsigKeySpec{
		Algorithm: "hmac-sha256", Name: "mock-tsig-key", CompartmentId: "ocid1.compartment.oc1..mock", Secret: "bW9jay10c2lnLWtleQ==", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newTsigKeyMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dns.mock.invalid", BasePath: "20180115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close TsigKey OCI mock: %v", err)
		}
	})
	client := newTsigKeyServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, dnssdk.DnsClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dnsv1beta1.TsigKey]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dnsv1beta1.TsigKey) error {
			if current.Status.Id != mockTsigKeyID || current.Status.Name != resource.Spec.Name || current.Status.LifecycleState != string(dnssdk.TsigKeyLifecycleStateActive) {
				return fmt.Errorf("created TsigKey status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dnsv1beta1.TsigKey) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *dnsv1beta1.TsigKey) error {
			if current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated TsigKey status = %+v", current.Status)
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

func newTsigKeyMockResponder(resource *dnsv1beta1.TsigKey) (*ocimock.CRUDResponder[dnssdk.TsigKey], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dnssdk.TsigKey]{
		CollectionPath: "/20180115/tsigKeys", ItemPath: "/20180115/tsigKeys/" + mockTsigKeyID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		List: func(_ ocimock.Request, present bool, state dnssdk.TsigKey) (ocimock.Response, error) {
			if !present || (state.LifecycleState == dnssdk.TsigKeyLifecycleStateDeleting && deleteRead) {
				return ocimock.JSONResponse(http.StatusOK, []dnssdk.TsigKey{})
			}
			return ocimock.JSONResponse(http.StatusOK, []dnssdk.TsigKey{state})
		},
		Create: func(request ocimock.Request) (dnssdk.TsigKey, ocimock.Response, error) {
			var details dnssdk.CreateTsigKeyDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.TsigKey{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dnssdk.TsigKey{}, ocimock.Response{}, err
			}
			if details.Algorithm == nil || *details.Algorithm != resource.Spec.Algorithm || details.Name == nil || *details.Name != resource.Spec.Name || details.Secret == nil || *details.Secret != resource.Spec.Secret {
				return dnssdk.TsigKey{}, ocimock.Response{}, fmt.Errorf("unexpected CreateTsigKey details: %+v", details)
			}
			state := dnssdk.TsigKey{Algorithm: details.Algorithm, Name: details.Name, CompartmentId: details.CompartmentId, Secret: details.Secret,
				FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{}, Id: common.String(mockTsigKeyID), Self: common.String("https://dns.mock.invalid/tsigKeys/" + mockTsigKeyID),
				TimeCreated: &now, TimeUpdated: &now, LifecycleState: dnssdk.TsigKeyLifecycleStateCreating}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dnssdk.TsigKey) (dnssdk.TsigKey, ocimock.Response, error) {
			switch state.LifecycleState {
			case dnssdk.TsigKeyLifecycleStateCreating:
				if createRead {
					state.LifecycleState = dnssdk.TsigKeyLifecycleStateActive
				} else {
					createRead = true
				}
			case dnssdk.TsigKeyLifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = dnssdk.TsigKeyLifecycleStateActive
				} else {
					updateRead = true
				}
			case dnssdk.TsigKeyLifecycleStateDeleting:
				if deleteRead {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
					return state, response, err
				}
				deleteRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dnssdk.TsigKey) (dnssdk.TsigKey, ocimock.Response, error) {
			var details dnssdk.UpdateTsigKeyDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.TsigKey{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" {
				return dnssdk.TsigKey{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateTsigKey details: %+v", details)
			}
			state.FreeformTags, state.LifecycleState = details.FreeformTags, dnssdk.TsigKeyLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state dnssdk.TsigKey) (dnssdk.TsigKey, ocimock.Response, error) {
			state.LifecycleState = dnssdk.TsigKeyLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
	})
}
