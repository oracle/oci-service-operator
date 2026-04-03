/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compartment

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	identitysdk "github.com/oracle/oci-go-sdk/v65/identity"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestCompartmentOrphanDeleteReturnsTrueWhenAlreadyDeleting(t *testing.T) {
	t.Parallel()

	resource := &identityv1beta1.Compartment{}
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.compartment.oc1..alreadydeleting")

	deleteCalled := false
	client := compartmentOrphanDeleteClient{
		delegate: noopCompartmentServiceClient{},
		deleteCompartment: func(context.Context, shared.OCID) error {
			deleteCalled = true
			return nil
		},
		loadCompartment: func(context.Context, shared.OCID) (*identitysdk.Compartment, error) {
			return &identitysdk.Compartment{LifecycleState: identitysdk.CompartmentLifecycleStateDeleting}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatalf("Delete returned deleted=false, want true")
	}
	if deleteCalled {
		t.Fatalf("DeleteCompartment was called, want short-circuit on live DELETING state")
	}
}

func TestCompartmentOrphanDeleteReturnsTrueAfterAcceptedDeleteRequest(t *testing.T) {
	t.Parallel()

	resource := &identityv1beta1.Compartment{}
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.compartment.oc1..accepted")

	deleteCalled := false
	client := compartmentOrphanDeleteClient{
		delegate: noopCompartmentServiceClient{},
		deleteCompartment: func(context.Context, shared.OCID) error {
			deleteCalled = true
			return nil
		},
		loadCompartment: func(context.Context, shared.OCID) (*identitysdk.Compartment, error) {
			return &identitysdk.Compartment{LifecycleState: identitysdk.CompartmentLifecycleStateActive}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatalf("Delete returned deleted=false, want true")
	}
	if !deleteCalled {
		t.Fatalf("DeleteCompartment was not called, want accepted delete request")
	}
}

func TestCompartmentOrphanDeleteTreatsConflictWithDeletingResourceAsSuccess(t *testing.T) {
	t.Parallel()

	resource := &identityv1beta1.Compartment{}
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.compartment.oc1..conflictdeleting")

	client := compartmentOrphanDeleteClient{
		delegate: noopCompartmentServiceClient{},
		deleteCompartment: func(context.Context, shared.OCID) error {
			return stubServiceError{statusCode: 409, code: "Conflict"}
		},
		loadCompartment: func(context.Context, shared.OCID) (*identitysdk.Compartment, error) {
			return &identitysdk.Compartment{LifecycleState: identitysdk.CompartmentLifecycleStateDeleting}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatalf("Delete returned deleted=false, want true")
	}
}

func TestCompartmentOrphanDeleteRetriesOnConflictWhenDeleteNotYetAccepted(t *testing.T) {
	t.Parallel()

	resource := &identityv1beta1.Compartment{}
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.compartment.oc1..conflictactive")

	client := compartmentOrphanDeleteClient{
		delegate: noopCompartmentServiceClient{},
		deleteCompartment: func(context.Context, shared.OCID) error {
			return stubServiceError{statusCode: 409, code: "Conflict"}
		},
		loadCompartment: func(context.Context, shared.OCID) (*identitysdk.Compartment, error) {
			return &identitysdk.Compartment{LifecycleState: identitysdk.CompartmentLifecycleStateActive}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatalf("Delete returned deleted=true, want false while delete acceptance is unconfirmed")
	}
}

func TestCompartmentOrphanDeleteDelegatesWhenNoTrackedIDExists(t *testing.T) {
	t.Parallel()

	resource := &identityv1beta1.Compartment{}
	delegateCalled := false
	client := compartmentOrphanDeleteClient{
		delegate: noopCompartmentServiceClient{
			deleteFunc: func(context.Context, *identityv1beta1.Compartment) (bool, error) {
				delegateCalled = true
				return true, nil
			},
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatalf("Delete returned deleted=false, want true")
	}
	if !delegateCalled {
		t.Fatalf("delegate Delete was not called")
	}
}

type noopCompartmentServiceClient struct {
	deleteFunc func(context.Context, *identityv1beta1.Compartment) (bool, error)
}

func (c noopCompartmentServiceClient) CreateOrUpdate(context.Context, *identityv1beta1.Compartment, ctrl.Request) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func (c noopCompartmentServiceClient) Delete(ctx context.Context, resource *identityv1beta1.Compartment) (bool, error) {
	if c.deleteFunc != nil {
		return c.deleteFunc(ctx, resource)
	}
	return false, nil
}

type stubServiceError struct {
	statusCode int
	code       string
}

func (e stubServiceError) Error() string {
	return e.code
}

func (e stubServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e stubServiceError) GetMessage() string {
	return e.code
}

func (e stubServiceError) GetCode() string {
	return e.code
}

func (e stubServiceError) GetOpcRequestID() string {
	return ""
}

func (e stubServiceError) GetCause() error {
	return nil
}

func (e stubServiceError) GetSuggestions() []string {
	return nil
}

func (e stubServiceError) GetOperationName() string {
	return ""
}

func (e stubServiceError) GetTimestamp() string {
	return ""
}

func TestCompartmentDeleteIsNotFound(t *testing.T) {
	t.Parallel()

	if !compartmentDeleteIsNotFound(stubServiceError{statusCode: 404, code: "NotAuthorizedOrNotFound"}) {
		t.Fatalf("compartmentDeleteIsNotFound returned false, want true")
	}
	if compartmentDeleteIsNotFound(errors.New("boom")) {
		t.Fatalf("compartmentDeleteIsNotFound returned true for generic error")
	}
}

var _ common.ServiceError = stubServiceError{}
