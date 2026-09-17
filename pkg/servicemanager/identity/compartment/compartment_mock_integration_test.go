/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compartment

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	identitysdk "github.com/oracle/oci-go-sdk/v65/identity"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockCompartmentID = "ocid1.compartment.oc1..mock-child"

// Contract evidence: the package-owned typed OCI fixtures, formal best-effort delete contract, orphan-delete wrapper, and vendored OCI SDK.
func TestMockIntegrationCompartmentLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &identityv1beta1.Compartment{ObjectMeta: metav1.ObjectMeta{Name: "mock-compartment", Namespace: "default", UID: types.UID("mock-compartment-uid")}, Spec: identityv1beta1.CompartmentSpec{
		CompartmentId: "ocid1.compartment.oc1..mock-parent", Name: "mock-compartment", Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newCompartmentMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://identity.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newMockCompartmentClient(identitysdk.IdentityClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*identityv1beta1.Compartment]{
		Resource: resource, Client: client,
		ValidateCreated: func(current *identityv1beta1.Compartment) error {
			if current.Status.Id != mockCompartmentID || current.Status.Name != resource.Spec.Name || current.Status.LifecycleState != string(identitysdk.CompartmentLifecycleStateActive) {
				return fmt.Errorf("created Compartment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *identityv1beta1.Compartment) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *identityv1beta1.Compartment) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Compartment status = %+v", current.Status)
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

func newCompartmentMockResponder(resource *identityv1beta1.Compartment) (*ocimock.CRUDResponder[identitysdk.Compartment], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[identitysdk.Compartment]{
		CollectionPath: "/20160918/compartments", ItemPath: "/20160918/compartments/" + mockCompartmentID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true,
		List: func(request ocimock.Request, present bool, state identitysdk.Compartment) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("compartmentId") != resource.Spec.CompartmentId || query.Get("name") != resource.Spec.Name {
				return ocimock.Response{}, fmt.Errorf("unexpected ListCompartments query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []identitysdk.Compartment{})
			}
			return ocimock.JSONResponse(http.StatusOK, []identitysdk.Compartment{state})
		},
		Create: func(request ocimock.Request) (identitysdk.Compartment, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero identitysdk.Compartment
				return zero, ocimock.Response{}, err
			}
			var details identitysdk.CreateCompartmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return identitysdk.Compartment{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return identitysdk.Compartment{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.Name == nil || *details.Name != resource.Spec.Name || details.Description == nil || *details.Description != resource.Spec.Description {
				return identitysdk.Compartment{}, ocimock.Response{}, fmt.Errorf("unexpected CreateCompartment details: %+v", details)
			}
			state := identitysdk.Compartment{Id: common.String(mockCompartmentID), CompartmentId: details.CompartmentId, Name: details.Name, Description: details.Description,
				TimeCreated: &now, LifecycleState: identitysdk.CompartmentLifecycleStateCreating, IsAccessible: common.Bool(true), FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state identitysdk.Compartment) (identitysdk.Compartment, ocimock.Response, error) {
			if state.LifecycleState == identitysdk.CompartmentLifecycleStateCreating && createRead {
				state.LifecycleState = identitysdk.CompartmentLifecycleStateActive
			} else if state.LifecycleState == identitysdk.CompartmentLifecycleStateCreating {
				createRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state identitysdk.Compartment) (identitysdk.Compartment, ocimock.Response, error) {
			var details identitysdk.UpdateCompartmentDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return identitysdk.Compartment{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.Name != nil || details.FreeformTags["osok-mock"] != "update" {
				return identitysdk.Compartment{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateCompartment details: %+v", details)
			}
			state.Description, state.FreeformTags = details.Description, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state identitysdk.Compartment) (identitysdk.Compartment, ocimock.Response, error) {
			state.LifecycleState = identitysdk.CompartmentLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusAccepted), nil
		},
		RetainStateAfterDelete: true,
	})
}
