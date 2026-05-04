/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementagentinstallkey

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestManagementAgentInstallKeyServiceClientCreatesAndProjectsStatus(t *testing.T) {
	expiresAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.AllowedKeyInstallCount = 7
	resource.Spec.TimeExpires = expiresAt.Format(time.RFC3339Nano)
	resource.Spec.IsKeyActive = true

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.list = expectManagementAgentInstallKeyListLookup(t, resource)
	fake.create = createManagementAgentInstallKeyHandler(t, resource, expiresAt)
	fake.get = getCreatedManagementAgentInstallKeyHandler(t, resource, expiresAt)

	response, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertCreatedManagementAgentInstallKeyStatus(t, resource)
	if fake.createCalls != 1 {
		t.Fatalf("CreateManagementAgentInstallKey calls = %d, want 1", fake.createCalls)
	}
}

func TestManagementAgentInstallKeyServiceClientNormalizesUnlimitedIgnoredFields(t *testing.T) {
	expiresAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	resource := newUnlimitedManagementAgentInstallKeyResource(expiresAt)

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.list = expectManagementAgentInstallKeyListLookup(t, resource)
	fake.create = func(_ context.Context, request managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error) {
		t.Helper()
		assertManagementAgentInstallKeyCreateDetails(t, request.CreateManagementAgentInstallKeyDetails, resource, expiresAt)
		created := managementAgentInstallKey("ocid1.installkey.oc1..created", resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesCreating)
		created.IsUnlimited = common.Bool(true)
		return managementagentsdk.CreateManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: created,
			OpcRequestId:              common.String("opc-create-1"),
		}, nil
	}
	fake.get = getUnlimitedManagementAgentInstallKeyHandler(t, "ocid1.installkey.oc1..created", resource)

	response, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if fake.createCalls != 1 {
		t.Fatalf("CreateManagementAgentInstallKey calls = %d, want 1", fake.createCalls)
	}

	rehydrated := newUnlimitedManagementAgentInstallKeyResource(expiresAt)
	rehydrated.Status.Id = "ocid1.installkey.oc1..created"
	rehydrated.Status.OsokStatus.Ocid = shared.OCID(rehydrated.Status.Id)
	readback := &fakeManagementAgentInstallKeyOCIClient{t: t}
	readback.get = getUnlimitedManagementAgentInstallKeyHandler(t, rehydrated.Status.Id, rehydrated)

	response, err = newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), readback).CreateOrUpdate(context.Background(), rehydrated, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after unlimited readback error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() after unlimited readback IsSuccessful = false, want true")
	}
	if readback.updateCalls != 0 {
		t.Fatalf("UpdateManagementAgentInstallKey calls after unlimited readback = %d, want 0", readback.updateCalls)
	}
}

func TestManagementAgentInstallKeyServiceClientBindsFromPaginatedList(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.IsKeyActive = true

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.list = paginatedManagementAgentInstallKeyBindHandler(t, fake, resource)
	fake.get = func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		if got, want := stringValue(request.ManagementAgentInstallKeyId), "ocid1.installkey.oc1..bound"; got != want {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, want)
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey("ocid1.installkey.oc1..bound", resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive),
		}, nil
	}

	response, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got, want := resource.Status.Id, "ocid1.installkey.oc1..bound"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateManagementAgentInstallKey calls = %d, want 0", fake.createCalls)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListManagementAgentInstallKeys calls = %d, want 2", fake.listCalls)
	}
}

func TestManagementAgentInstallKeyServiceClientBindsWithTrimmedIdentityFields(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.DisplayName = "  install-key  "
	resource.Spec.CompartmentId = "  ocid1.compartment.oc1..source  "
	resource.Spec.IsKeyActive = true

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.list = trimmedManagementAgentInstallKeyBindHandler(t, resource)
	fake.get = func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		if got, want := stringValue(request.ManagementAgentInstallKeyId), "ocid1.installkey.oc1..bound"; got != want {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, want)
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey(
				"ocid1.installkey.oc1..bound",
				"ocid1.compartment.oc1..source",
				"install-key",
				managementagentsdk.LifecycleStatesActive,
			),
		}, nil
	}

	response, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateManagementAgentInstallKey calls = %d, want 0", fake.createCalls)
	}
	if got, want := resource.Spec.DisplayName, "install-key"; got != want {
		t.Fatalf("spec.displayName after normalization = %q, want %q", got, want)
	}
	if got, want := resource.Spec.CompartmentId, "ocid1.compartment.oc1..source"; got != want {
		t.Fatalf("spec.compartmentId after normalization = %q, want %q", got, want)
	}
	if got, want := resource.Status.Id, "ocid1.installkey.oc1..bound"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func TestManagementAgentInstallKeyServiceClientNoopReconcileDoesNotUpdate(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.IsKeyActive = true
	resource.Status.Id = "ocid1.installkey.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.get = func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		if got, want := stringValue(request.ManagementAgentInstallKeyId), resource.Status.Id; got != want {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, want)
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive),
		}, nil
	}

	response, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateManagementAgentInstallKey calls = %d, want 0", fake.updateCalls)
	}
}

func TestManagementAgentInstallKeyServiceClientUpdatesMutableFields(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.DisplayName = "renamed-install-key"
	resource.Spec.IsKeyActive = false
	resource.Status.Id = "ocid1.installkey.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.get = mutableManagementAgentInstallKeyReadHandler(t, fake, resource)
	fake.update = updateManagementAgentInstallKeyHandler(t, resource)

	response, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if fake.updateCalls != 1 {
		t.Fatalf("UpdateManagementAgentInstallKey calls = %d, want 1", fake.updateCalls)
	}
	if got, want := resource.Status.DisplayName, resource.Spec.DisplayName; got != want {
		t.Fatalf("status.displayName = %q, want %q", got, want)
	}
	if got, want := resource.Status.LifecycleState, string(managementagentsdk.LifecycleStatesInactive); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestManagementAgentInstallKeyServiceClientRejectsCreateOnlyDrift(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	resource.Spec.IsKeyActive = true
	resource.Status.Id = "ocid1.installkey.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.get = func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, "ocid1.compartment.oc1..old", resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive),
		}, nil
	}

	_, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateManagementAgentInstallKey calls = %d, want 0", fake.updateCalls)
	}
}

func TestManagementAgentInstallKeyServiceClientRejectsClearedCreateOnlyOptionalFields(t *testing.T) {
	expiresAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		wantField    string
		applyCurrent func(*managementagentsdk.ManagementAgentInstallKey, *managementagentv1beta1.ManagementAgentInstallKey)
	}{
		{
			name:      "allowedKeyInstallCount",
			wantField: "allowedKeyInstallCount",
			applyCurrent: func(current *managementagentsdk.ManagementAgentInstallKey, resource *managementagentv1beta1.ManagementAgentInstallKey) {
				current.AllowedKeyInstallCount = common.Int(7)
				resource.Status.AllowedKeyInstallCount = 7
			},
		},
		{
			name:      "timeExpires",
			wantField: "timeExpires",
			applyCurrent: func(current *managementagentsdk.ManagementAgentInstallKey, resource *managementagentv1beta1.ManagementAgentInstallKey) {
				current.TimeExpires = sdkTimePointer(expiresAt)
				resource.Status.TimeExpires = expiresAt.Format(time.RFC3339Nano)
			},
		},
		{
			name:      "isUnlimited",
			wantField: "isUnlimited",
			applyCurrent: func(current *managementagentsdk.ManagementAgentInstallKey, resource *managementagentv1beta1.ManagementAgentInstallKey) {
				current.IsUnlimited = common.Bool(true)
				resource.Status.IsUnlimited = true
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := newManagementAgentInstallKeyResource()
			resource.Status.Id = "ocid1.installkey.oc1..existing"
			resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

			current := managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive)
			tc.applyCurrent(&current, resource)
			fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
			fake.get = getExistingManagementAgentInstallKeyHandler(t, resource, current)

			assertManagementAgentInstallKeyCreateOnlyDriftRejectedBeforeUpdate(t, resource, fake, tc.wantField)
		})
	}
}

func TestManagementAgentInstallKeyServiceClientDeleteKeepsFinalizerWhileDeleting(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Status.Id = "ocid1.installkey.oc1..delete"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.get = func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		if fake.getCalls < 3 {
			return managementagentsdk.GetManagementAgentInstallKeyResponse{
				ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive),
			}, nil
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesDeleting),
		}, nil
	}
	fake.delete = func(_ context.Context, request managementagentsdk.DeleteManagementAgentInstallKeyRequest) (managementagentsdk.DeleteManagementAgentInstallKeyResponse, error) {
		if got, want := stringValue(request.ManagementAgentInstallKeyId), resource.Status.Id; got != want {
			t.Fatalf("DeleteManagementAgentInstallKey id = %q, want %q", got, want)
		}
		return managementagentsdk.DeleteManagementAgentInstallKeyResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI reports DELETING")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeleteManagementAgentInstallKey calls = %d, want 1", fake.deleteCalls)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending operation")
	}
	if got, want := resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete; got != want {
		t.Fatalf("status.status.async.current.phase = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestManagementAgentInstallKeyServiceClientDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Status.Id = "ocid1.installkey.oc1..delete"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	notFound := errortest.NewServiceError(404, errorutil.NotFound, "install key deleted")
	notFound.OpcRequestID = "opc-confirm-notfound"
	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.get = func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		if fake.getCalls < 3 {
			return managementagentsdk.GetManagementAgentInstallKeyResponse{
				ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive),
			}, nil
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{}, notFound
	}
	fake.delete = func(context.Context, managementagentsdk.DeleteManagementAgentInstallKeyRequest) (managementagentsdk.DeleteManagementAgentInstallKeyResponse, error) {
		return managementagentsdk.DeleteManagementAgentInstallKeyResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	fake.list = func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
		return managementagentsdk.ListManagementAgentInstallKeysResponse{}, nil
	}

	deleted, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous confirm-read NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestManagementAgentInstallKeyServiceClientDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()
	resource.Status.Id = "ocid1.installkey.oc1..delete"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-delete-auth"
	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.get = func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		return managementagentsdk.GetManagementAgentInstallKeyResponse{}, serviceErr
	}

	deleted, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found rejection")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteManagementAgentInstallKey calls = %d, want 0", fake.deleteCalls)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete-auth"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestManagementAgentInstallKeyServiceClientRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := newManagementAgentInstallKeyResource()

	serviceErr := errortest.NewServiceError(500, "InternalError", "create failed")
	serviceErr.OpcRequestID = "opc-create-error"
	fake := &fakeManagementAgentInstallKeyOCIClient{t: t}
	fake.list = func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
		return managementagentsdk.ListManagementAgentInstallKeysResponse{}, nil
	}
	fake.create = func(context.Context, managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error) {
		return managementagentsdk.CreateManagementAgentInstallKeyResponse{}, serviceErr
	}

	_, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-error"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func expectManagementAgentInstallKeyListLookup(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
		t.Helper()
		if got, want := stringValue(request.CompartmentId), resource.Spec.CompartmentId; got != want {
			t.Fatalf("ListManagementAgentInstallKeys compartmentId = %q, want %q", got, want)
		}
		if got, want := stringValue(request.DisplayName), resource.Spec.DisplayName; got != want {
			t.Fatalf("ListManagementAgentInstallKeys displayName = %q, want %q", got, want)
		}
		return managementagentsdk.ListManagementAgentInstallKeysResponse{}, nil
	}
}

func createManagementAgentInstallKeyHandler(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	expiresAt time.Time,
) func(context.Context, managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error) {
		t.Helper()
		assertManagementAgentInstallKeyCreateDetails(t, request.CreateManagementAgentInstallKeyDetails, resource, expiresAt)
		created := managementAgentInstallKey("ocid1.installkey.oc1..created", resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesCreating)
		created = managementAgentInstallKeyWithCreateOnlyFields(created, 7, expiresAt, false)
		return managementagentsdk.CreateManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: created,
			OpcRequestId:              common.String("opc-create-1"),
		}, nil
	}
}

func assertManagementAgentInstallKeyCreateDetails(
	t *testing.T,
	details managementagentsdk.CreateManagementAgentInstallKeyDetails,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	expiresAt time.Time,
) {
	t.Helper()
	assertManagementAgentInstallKeyCreateIdentity(t, details, resource)
	if resource.Spec.IsUnlimited {
		assertUnlimitedManagementAgentInstallKeyCreateDetails(t, details)
		return
	}
	assertLimitedManagementAgentInstallKeyCreateDetails(t, details, expiresAt)
}

func assertManagementAgentInstallKeyCreateIdentity(
	t *testing.T,
	details managementagentsdk.CreateManagementAgentInstallKeyDetails,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) {
	t.Helper()
	if got, want := stringValue(details.DisplayName), resource.Spec.DisplayName; got != want {
		t.Fatalf("CreateManagementAgentInstallKey displayName = %q, want %q", got, want)
	}
	if got, want := stringValue(details.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("CreateManagementAgentInstallKey compartmentId = %q, want %q", got, want)
	}
}

func assertUnlimitedManagementAgentInstallKeyCreateDetails(
	t *testing.T,
	details managementagentsdk.CreateManagementAgentInstallKeyDetails,
) {
	t.Helper()
	if details.AllowedKeyInstallCount != nil {
		t.Fatalf("CreateManagementAgentInstallKey allowedKeyInstallCount = %#v, want nil for unlimited key", details.AllowedKeyInstallCount)
	}
	if details.TimeExpires != nil {
		t.Fatalf("CreateManagementAgentInstallKey timeExpires = %#v, want nil for unlimited key", details.TimeExpires)
	}
	if details.IsUnlimited == nil || !*details.IsUnlimited {
		t.Fatalf("CreateManagementAgentInstallKey isUnlimited = %#v, want true", details.IsUnlimited)
	}
}

func assertLimitedManagementAgentInstallKeyCreateDetails(
	t *testing.T,
	details managementagentsdk.CreateManagementAgentInstallKeyDetails,
	expiresAt time.Time,
) {
	t.Helper()
	if details.AllowedKeyInstallCount == nil || *details.AllowedKeyInstallCount != 7 {
		t.Fatalf("CreateManagementAgentInstallKey allowedKeyInstallCount = %#v, want 7", details.AllowedKeyInstallCount)
	}
	if details.TimeExpires == nil || !details.TimeExpires.Equal(expiresAt) {
		t.Fatalf("CreateManagementAgentInstallKey timeExpires = %#v, want %s", details.TimeExpires, expiresAt)
	}
	if details.IsUnlimited != nil {
		t.Fatalf("CreateManagementAgentInstallKey isUnlimited = %#v, want nil", details.IsUnlimited)
	}
}

func getCreatedManagementAgentInstallKeyHandler(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	expiresAt time.Time,
) func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		t.Helper()
		if got, want := stringValue(request.ManagementAgentInstallKeyId), "ocid1.installkey.oc1..created"; got != want {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, want)
		}
		created := managementAgentInstallKey("ocid1.installkey.oc1..created", resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive)
		created = managementAgentInstallKeyWithCreateOnlyFields(created, 7, expiresAt, false)
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: created,
			OpcRequestId:              common.String("opc-get-1"),
		}, nil
	}
}

func assertCreatedManagementAgentInstallKeyStatus(t *testing.T, resource *managementagentv1beta1.ManagementAgentInstallKey) {
	t.Helper()
	if got, want := resource.Status.Id, "ocid1.installkey.oc1..created"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.installkey.oc1..created"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.Key, "install-key-value"; got != want {
		t.Fatalf("status.key = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func paginatedManagementAgentInstallKeyBindHandler(
	t *testing.T,
	fake *fakeManagementAgentInstallKeyOCIClient,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
		t.Helper()
		if fake.listCalls == 1 {
			return firstManagementAgentInstallKeyBindListPage(t, request, resource)
		}
		if fake.listCalls == 2 {
			return secondManagementAgentInstallKeyBindListPage(t, request, resource)
		}
		t.Fatalf("unexpected ListManagementAgentInstallKeys call %d", fake.listCalls)
		return managementagentsdk.ListManagementAgentInstallKeysResponse{}, nil
	}
}

func firstManagementAgentInstallKeyBindListPage(
	t *testing.T,
	request managementagentsdk.ListManagementAgentInstallKeysRequest,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	t.Helper()
	if request.Page != nil {
		t.Fatalf("first ListManagementAgentInstallKeys page = %q, want nil", *request.Page)
	}
	return managementagentsdk.ListManagementAgentInstallKeysResponse{
		Items: []managementagentsdk.ManagementAgentInstallKeySummary{
			managementAgentInstallKeySummary("ocid1.installkey.oc1..other", resource.Spec.CompartmentId, "other-key", managementagentsdk.LifecycleStatesActive),
		},
		OpcNextPage: common.String("page-2"),
	}, nil
}

func secondManagementAgentInstallKeyBindListPage(
	t *testing.T,
	request managementagentsdk.ListManagementAgentInstallKeysRequest,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	t.Helper()
	if got, want := stringValue(request.Page), "page-2"; got != want {
		t.Fatalf("second ListManagementAgentInstallKeys page = %q, want %q", got, want)
	}
	return managementagentsdk.ListManagementAgentInstallKeysResponse{
		Items: []managementagentsdk.ManagementAgentInstallKeySummary{
			managementAgentInstallKeySummary("ocid1.installkey.oc1..bound", resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive),
		},
	}, nil
}

func trimmedManagementAgentInstallKeyBindHandler(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
		t.Helper()
		if got, want := stringValue(request.CompartmentId), strings.TrimSpace(resource.Spec.CompartmentId); got != want {
			t.Fatalf("ListManagementAgentInstallKeys compartmentId = %q, want %q", got, want)
		}
		if got, want := stringValue(request.DisplayName), strings.TrimSpace(resource.Spec.DisplayName); got != want {
			t.Fatalf("ListManagementAgentInstallKeys displayName = %q, want %q", got, want)
		}
		return managementagentsdk.ListManagementAgentInstallKeysResponse{
			Items: []managementagentsdk.ManagementAgentInstallKeySummary{
				managementAgentInstallKeySummary(
					"ocid1.installkey.oc1..bound",
					"ocid1.compartment.oc1..source",
					"install-key",
					managementagentsdk.LifecycleStatesActive,
				),
			},
		}, nil
	}
}

func mutableManagementAgentInstallKeyReadHandler(
	t *testing.T,
	fake *fakeManagementAgentInstallKeyOCIClient,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		t.Helper()
		if got, want := stringValue(request.ManagementAgentInstallKeyId), resource.Status.Id; got != want {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, want)
		}
		if fake.getCalls == 1 {
			return managementagentsdk.GetManagementAgentInstallKeyResponse{
				ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, "old-install-key", managementagentsdk.LifecycleStatesActive),
			}, nil
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesInactive),
		}, nil
	}
}

func updateManagementAgentInstallKeyHandler(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.UpdateManagementAgentInstallKeyRequest) (managementagentsdk.UpdateManagementAgentInstallKeyResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.UpdateManagementAgentInstallKeyRequest) (managementagentsdk.UpdateManagementAgentInstallKeyResponse, error) {
		t.Helper()
		assertManagementAgentInstallKeyUpdateRequest(t, request, resource)
		return managementagentsdk.UpdateManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: managementAgentInstallKey(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesInactive),
			OpcRequestId:              common.String("opc-update-1"),
		}, nil
	}
}

func assertManagementAgentInstallKeyUpdateRequest(
	t *testing.T,
	request managementagentsdk.UpdateManagementAgentInstallKeyRequest,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) {
	t.Helper()
	if got, want := stringValue(request.ManagementAgentInstallKeyId), resource.Status.Id; got != want {
		t.Fatalf("UpdateManagementAgentInstallKey id = %q, want %q", got, want)
	}
	details := request.UpdateManagementAgentInstallKeyDetails
	if got, want := stringValue(details.DisplayName), resource.Spec.DisplayName; got != want {
		t.Fatalf("UpdateManagementAgentInstallKey displayName = %q, want %q", got, want)
	}
	if details.IsKeyActive == nil || *details.IsKeyActive {
		t.Fatalf("UpdateManagementAgentInstallKey isKeyActive = %#v, want explicit false", details.IsKeyActive)
	}
}

func getExistingManagementAgentInstallKeyHandler(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	current managementagentsdk.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		t.Helper()
		if got, want := stringValue(request.ManagementAgentInstallKeyId), resource.Status.Id; got != want {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, want)
		}
		return managementagentsdk.GetManagementAgentInstallKeyResponse{ManagementAgentInstallKey: current}, nil
	}
}

func assertManagementAgentInstallKeyCreateOnlyDriftRejectedBeforeUpdate(
	t *testing.T,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	fake *fakeManagementAgentInstallKeyOCIClient,
	wantField string,
) {
	t.Helper()
	_, err := newManagementAgentInstallKeyServiceClientWithOCIClient(testLog(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), wantField) {
		t.Fatalf("CreateOrUpdate() error = %v, want %s drift", err, wantField)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateManagementAgentInstallKey calls = %d, want 0", fake.updateCalls)
	}
}

type fakeManagementAgentInstallKeyOCIClient struct {
	t *testing.T

	create func(context.Context, managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error)
	get    func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error)
	list   func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error)
	update func(context.Context, managementagentsdk.UpdateManagementAgentInstallKeyRequest) (managementagentsdk.UpdateManagementAgentInstallKeyResponse, error)
	delete func(context.Context, managementagentsdk.DeleteManagementAgentInstallKeyRequest) (managementagentsdk.DeleteManagementAgentInstallKeyResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeManagementAgentInstallKeyOCIClient) CreateManagementAgentInstallKey(
	ctx context.Context,
	request managementagentsdk.CreateManagementAgentInstallKeyRequest,
) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error) {
	f.createCalls++
	if f.create == nil {
		f.t.Fatalf("unexpected CreateManagementAgentInstallKey call %d", f.createCalls)
	}
	return f.create(ctx, request)
}

func (f *fakeManagementAgentInstallKeyOCIClient) GetManagementAgentInstallKey(
	ctx context.Context,
	request managementagentsdk.GetManagementAgentInstallKeyRequest,
) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
	f.getCalls++
	if f.get == nil {
		f.t.Fatalf("unexpected GetManagementAgentInstallKey call %d", f.getCalls)
	}
	return f.get(ctx, request)
}

func (f *fakeManagementAgentInstallKeyOCIClient) ListManagementAgentInstallKeys(
	ctx context.Context,
	request managementagentsdk.ListManagementAgentInstallKeysRequest,
) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	f.listCalls++
	if f.list == nil {
		f.t.Fatalf("unexpected ListManagementAgentInstallKeys call %d", f.listCalls)
	}
	return f.list(ctx, request)
}

func (f *fakeManagementAgentInstallKeyOCIClient) UpdateManagementAgentInstallKey(
	ctx context.Context,
	request managementagentsdk.UpdateManagementAgentInstallKeyRequest,
) (managementagentsdk.UpdateManagementAgentInstallKeyResponse, error) {
	f.updateCalls++
	if f.update == nil {
		f.t.Fatalf("unexpected UpdateManagementAgentInstallKey call %d", f.updateCalls)
	}
	return f.update(ctx, request)
}

func (f *fakeManagementAgentInstallKeyOCIClient) DeleteManagementAgentInstallKey(
	ctx context.Context,
	request managementagentsdk.DeleteManagementAgentInstallKeyRequest,
) (managementagentsdk.DeleteManagementAgentInstallKeyResponse, error) {
	f.deleteCalls++
	if f.delete == nil {
		f.t.Fatalf("unexpected DeleteManagementAgentInstallKey call %d", f.deleteCalls)
	}
	return f.delete(ctx, request)
}

func newManagementAgentInstallKeyResource() *managementagentv1beta1.ManagementAgentInstallKey {
	return &managementagentv1beta1.ManagementAgentInstallKey{
		Spec: managementagentv1beta1.ManagementAgentInstallKeySpec{
			DisplayName:   "install-key",
			CompartmentId: "ocid1.compartment.oc1..source",
		},
	}
}

func newUnlimitedManagementAgentInstallKeyResource(expiresAt time.Time) *managementagentv1beta1.ManagementAgentInstallKey {
	resource := newManagementAgentInstallKeyResource()
	resource.Spec.AllowedKeyInstallCount = 7
	resource.Spec.TimeExpires = expiresAt.Format(time.RFC3339Nano)
	resource.Spec.IsUnlimited = true
	resource.Spec.IsKeyActive = true
	return resource
}

func getUnlimitedManagementAgentInstallKeyHandler(
	t *testing.T,
	id string,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
	t.Helper()
	return func(_ context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
		t.Helper()
		if got := stringValue(request.ManagementAgentInstallKeyId); got != id {
			t.Fatalf("GetManagementAgentInstallKey id = %q, want %q", got, id)
		}
		key := managementAgentInstallKey(id, resource.Spec.CompartmentId, resource.Spec.DisplayName, managementagentsdk.LifecycleStatesActive)
		key.IsUnlimited = common.Bool(true)
		return managementagentsdk.GetManagementAgentInstallKeyResponse{
			ManagementAgentInstallKey: key,
			OpcRequestId:              common.String("opc-get-1"),
		}, nil
	}
}

func managementAgentInstallKey(
	id string,
	compartmentID string,
	displayName string,
	lifecycleState managementagentsdk.LifecycleStatesEnum,
) managementagentsdk.ManagementAgentInstallKey {
	return managementagentsdk.ManagementAgentInstallKey{
		Id:                     common.String(id),
		CompartmentId:          common.String(compartmentID),
		DisplayName:            common.String(displayName),
		Key:                    common.String("install-key-value"),
		CurrentKeyInstallCount: common.Int(1),
		LifecycleState:         lifecycleState,
	}
}

func managementAgentInstallKeyWithCreateOnlyFields(
	key managementagentsdk.ManagementAgentInstallKey,
	allowedKeyInstallCount int,
	expiresAt time.Time,
	isUnlimited bool,
) managementagentsdk.ManagementAgentInstallKey {
	key.AllowedKeyInstallCount = common.Int(allowedKeyInstallCount)
	key.TimeExpires = sdkTimePointer(expiresAt)
	key.IsUnlimited = common.Bool(isUnlimited)
	return key
}

func sdkTimePointer(value time.Time) *common.SDKTime {
	return &common.SDKTime{Time: value}
}

func managementAgentInstallKeySummary(
	id string,
	compartmentID string,
	displayName string,
	lifecycleState managementagentsdk.LifecycleStatesEnum,
) managementagentsdk.ManagementAgentInstallKeySummary {
	return managementagentsdk.ManagementAgentInstallKeySummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: lifecycleState,
	}
}

func testLog() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
