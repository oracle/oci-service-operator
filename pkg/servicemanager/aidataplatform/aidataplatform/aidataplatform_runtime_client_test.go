/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package aidataplatform

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	aidataplatformsdk "github.com/oracle/oci-go-sdk/v65/aidataplatform"
	"github.com/oracle/oci-go-sdk/v65/common"
	aidataplatformv1beta1 "github.com/oracle/oci-service-operator/api/aidataplatform/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAiDataPlatformCompartmentID = "ocid1.compartment.oc1..aidp"
	testAiDataPlatformExistingID    = "ocid1.aidataplatform.oc1..existing"
	testAiDataPlatformCreatedID     = "ocid1.aidataplatform.oc1..created"
	testAiDataPlatformDisplayName   = "aidp-sample"
	testAiDataPlatformType          = "DATA_LAKE"
)

func TestAiDataPlatformRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := AiDataPlatformRuntimeHooks{}
	applyAiDataPlatformRuntimeHooks(nil, &hooks, nil, nil)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed AiDataPlatform semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "workrequest" {
		t.Fatalf("Async = %#v, want generated workrequest semantics", hooks.Semantics.Async)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", hooks.Semantics.FinalizerPolicy)
	}
	if hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", hooks.Semantics.DeleteFollowUp.Strategy)
	}
	if hooks.Semantics.List == nil {
		t.Fatal("List semantics = nil, want create-or-bind list matching")
	}
	assertContainsAll(t, hooks.Semantics.List.MatchFields, "compartmentId", "displayName", "id")
	assertContainsAll(t, hooks.Semantics.Mutation.Mutable, "displayName", "aiDataPlatformType", "freeformTags", "definedTags", "systemTags")
	assertContainsAll(t, hooks.Semantics.Mutation.ForceNew, "compartmentId", "defaultWorkspaceName")
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("ParityHooks.ValidateCreateOnlyDrift = nil, want defaultWorkspaceName drift guard")
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("Async.GetWorkRequest = nil, want work-request polling")
	}
}

func TestAiDataPlatformCreateOrUpdateTracksCreateWorkRequestThenCompletesWithoutDuplicateCreate(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	fake := &fakeAiDataPlatformOCIClient{}
	configureAiDataPlatformCreateWorkRequestFake(t, fake)

	client := newTestAiDataPlatformClient(fake)
	assertAiDataPlatformPendingCreate(t, client, resource)
	assertAiDataPlatformCompletedCreate(t, client, fake, resource)
}

func configureAiDataPlatformCreateWorkRequestFake(t *testing.T, fake *fakeAiDataPlatformOCIClient) {
	t.Helper()

	fake.listAiDataPlatforms = func(_ context.Context, request aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
		assertAiDataPlatformListRequest(t, request)
		return aidataplatformsdk.ListAiDataPlatformsResponse{}, nil
	}
	fake.createAiDataPlatform = func(_ context.Context, request aidataplatformsdk.CreateAiDataPlatformRequest) (aidataplatformsdk.CreateAiDataPlatformResponse, error) {
		assertAiDataPlatformCreateRequest(t, request)
		return aidataplatformsdk.CreateAiDataPlatformResponse{
			AiDataPlatform:   aiDataPlatformBody(testAiDataPlatformCreatedID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateCreating),
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("create-opc"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request aidataplatformsdk.GetWorkRequestRequest) (aidataplatformsdk.GetWorkRequestResponse, error) {
		if got, want := stringValue(request.WorkRequestId), "wr-create"; got != want {
			t.Fatalf("GetWorkRequest workRequestId = %q, want %q", got, want)
		}
		status := aidataplatformsdk.OperationStatusInProgress
		if fake.workRequestCalls > 1 {
			status = aidataplatformsdk.OperationStatusSucceeded
		}
		return aidataplatformsdk.GetWorkRequestResponse{
			WorkRequest: aiDataPlatformWorkRequest("wr-create", aidataplatformsdk.OperationTypeCreateDataLake, status, aidataplatformsdk.ActionTypeCreated, testAiDataPlatformCreatedID),
		}, nil
	}
	fake.getAiDataPlatform = func(_ context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		assertAiDataPlatformGetRequest(t, request, testAiDataPlatformCreatedID)
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformCreatedID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
		}, nil
	}
}

func assertAiDataPlatformPendingCreate(
	t *testing.T,
	client AiDataPlatformServiceClient,
	resource *aidataplatformv1beta1.AiDataPlatform,
) {
	t.Helper()

	response, err := client.CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want pending create work request", response)
	}
	if got, want := resource.Status.OsokStatus.Async.Current.WorkRequestID, "wr-create"; got != want {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Provisioning)
}

func assertAiDataPlatformCompletedCreate(
	t *testing.T,
	client AiDataPlatformServiceClient,
	fake *fakeAiDataPlatformOCIClient,
	resource *aidataplatformv1beta1.AiDataPlatform,
) {
	t.Helper()

	response, err := client.CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want active completed create", response)
	}
	if fake.createCalls != 1 {
		t.Fatalf("CreateAiDataPlatform calls = %d, want 1", fake.createCalls)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testAiDataPlatformCreatedID; got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
	if got, ok := aiDataPlatformRecordedDefaultWorkspaceNameFingerprint(resource); !ok || got != aiDataPlatformDefaultWorkspaceNameFingerprint("workspace-default") {
		t.Fatalf("defaultWorkspaceName fingerprint = %q, %t, want recorded original spec", got, ok)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestAiDataPlatformCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	fake := &fakeAiDataPlatformOCIClient{}
	fake.listAiDataPlatforms = pagedAiDataPlatformList(t, []aiDataPlatformListPage{
		{
			items: []aidataplatformsdk.AiDataPlatformSummary{
				aiDataPlatformSummary("ocid1.aidataplatform.oc1..other", testAiDataPlatformCompartmentID, "other-aidp"),
			},
			nextPage: "page-2",
		},
		{
			wantPage: "page-2",
			items: []aidataplatformsdk.AiDataPlatformSummary{
				aiDataPlatformSummary(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName),
			},
		},
	})
	fake.getAiDataPlatform = func(_ context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		assertAiDataPlatformGetRequest(t, request, testAiDataPlatformExistingID)
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
		}, nil
	}

	response, err := newTestAiDataPlatformClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind", response)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListAiDataPlatforms calls = %d, want 2", fake.listCalls)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateAiDataPlatform calls = %d, want 0", fake.createCalls)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testAiDataPlatformExistingID; got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestAiDataPlatformMutableUpdateUsesWorkRequestAndReadback(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Spec.DisplayName = "aidp-updated"
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		assertAiDataPlatformGetRequest(t, request, testAiDataPlatformExistingID)
		displayName := testAiDataPlatformDisplayName
		if fake.getCalls > 1 {
			displayName = "aidp-updated"
		}
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, displayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
		}, nil
	}
	fake.updateAiDataPlatform = func(_ context.Context, request aidataplatformsdk.UpdateAiDataPlatformRequest) (aidataplatformsdk.UpdateAiDataPlatformResponse, error) {
		assertAiDataPlatformUpdateRequest(t, request)
		return aidataplatformsdk.UpdateAiDataPlatformResponse{
			OpcWorkRequestId: common.String("wr-update"),
			OpcRequestId:     common.String("update-opc"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request aidataplatformsdk.GetWorkRequestRequest) (aidataplatformsdk.GetWorkRequestResponse, error) {
		if got, want := stringValue(request.WorkRequestId), "wr-update"; got != want {
			t.Fatalf("GetWorkRequest workRequestId = %q, want %q", got, want)
		}
		return aidataplatformsdk.GetWorkRequestResponse{
			WorkRequest: aiDataPlatformWorkRequest("wr-update", aidataplatformsdk.OperationTypeUpdateDataLake, aidataplatformsdk.OperationStatusSucceeded, aidataplatformsdk.ActionTypeUpdated, testAiDataPlatformExistingID),
		}, nil
	}

	response, err := newTestAiDataPlatformClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want completed update", response)
	}
	if fake.updateCalls != 1 {
		t.Fatalf("UpdateAiDataPlatform calls = %d, want 1", fake.updateCalls)
	}
	if got, want := resource.Status.DisplayName, "aidp-updated"; got != want {
		t.Fatalf("status.displayName = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "update-opc"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestAiDataPlatformCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
		}, nil
	}

	response, err := newTestAiDataPlatformClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId replacement rejection", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateAiDataPlatform calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestAiDataPlatformDefaultWorkspaceNameDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	now := metav1.Now()
	resource.Status.OsokStatus.CreatedAt = &now
	recordAiDataPlatformDefaultWorkspaceNameFingerprint(resource)
	resource.Spec.DefaultWorkspaceName = "workspace-renamed"

	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		assertAiDataPlatformGetRequest(t, request, testAiDataPlatformExistingID)
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
		}, nil
	}
	fake.updateAiDataPlatform = func(context.Context, aidataplatformsdk.UpdateAiDataPlatformRequest) (aidataplatformsdk.UpdateAiDataPlatformResponse, error) {
		t.Fatal("UpdateAiDataPlatform should not be called when defaultWorkspaceName drifts")
		return aidataplatformsdk.UpdateAiDataPlatformResponse{}, nil
	}

	response, err := newTestAiDataPlatformClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want defaultWorkspaceName drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when defaultWorkspaceName changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want defaultWorkspaceName replacement rejection", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateAiDataPlatform calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestAiDataPlatformDeleteWaitsForWorkRequestThenConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		if fake.deleteCalls == 0 {
			return aidataplatformsdk.GetAiDataPlatformResponse{
				AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
			}, nil
		}
		return aidataplatformsdk.GetAiDataPlatformResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "aidp deleted")
	}
	fake.deleteAiDataPlatform = func(_ context.Context, request aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error) {
		if got, want := stringValue(request.AiDataPlatformId), testAiDataPlatformExistingID; got != want {
			t.Fatalf("DeleteAiDataPlatform aiDataPlatformId = %q, want %q", got, want)
		}
		return aidataplatformsdk.DeleteAiDataPlatformResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("delete-opc"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request aidataplatformsdk.GetWorkRequestRequest) (aidataplatformsdk.GetWorkRequestResponse, error) {
		if got, want := stringValue(request.WorkRequestId), "wr-delete"; got != want {
			t.Fatalf("GetWorkRequest workRequestId = %q, want %q", got, want)
		}
		status := aidataplatformsdk.OperationStatusInProgress
		if fake.workRequestCalls > 1 {
			status = aidataplatformsdk.OperationStatusSucceeded
		}
		return aidataplatformsdk.GetWorkRequestResponse{
			WorkRequest: aiDataPlatformWorkRequest("wr-delete", aidataplatformsdk.OperationTypeDeleteDataLake, status, aidataplatformsdk.ActionTypeDeleted, testAiDataPlatformExistingID),
		}, nil
	}

	client := newTestAiDataPlatformClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while work request is pending")
	}
	assertTrailingCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want true after unambiguous not-found confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed delete timestamp")
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestAiDataPlatformDeleteAlreadyPendingLifecycleSkipsDelete(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		assertAiDataPlatformGetRequest(t, request, testAiDataPlatformExistingID)
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateDeleting),
		}, nil
	}
	fake.deleteAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error) {
		t.Fatal("DeleteAiDataPlatform should not be called when confirm-delete readback is already DELETING")
		return aidataplatformsdk.DeleteAiDataPlatformResponse{}, nil
	}

	deleted, err := newTestAiDataPlatformClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while lifecycle is DELETING")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteAiDataPlatform calls = %d, want 0", fake.deleteCalls)
	}
	if got, want := resource.Status.LifecycleState, string(aidataplatformsdk.AiDataPlatformLifecycleStateDeleting); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want finalizer retained")
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestAiDataPlatformDeleteConflictConfirmsPendingLifecycle(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		assertAiDataPlatformGetRequest(t, request, testAiDataPlatformExistingID)
		lifecycleState := aidataplatformsdk.AiDataPlatformLifecycleStateActive
		if fake.getCalls > 2 {
			lifecycleState = aidataplatformsdk.AiDataPlatformLifecycleStateDeleting
		}
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, lifecycleState),
		}, nil
	}
	fake.deleteAiDataPlatform = func(_ context.Context, request aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error) {
		if got, want := stringValue(request.AiDataPlatformId), testAiDataPlatformExistingID; got != want {
			t.Fatalf("DeleteAiDataPlatform aiDataPlatformId = %q, want %q", got, want)
		}
		return aidataplatformsdk.DeleteAiDataPlatformResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "delete is still settling")
	}

	deleted, err := newTestAiDataPlatformClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after pending conflict confirmation")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeleteAiDataPlatform calls = %d, want 1", fake.deleteCalls)
	}
	if fake.getCalls != 3 {
		t.Fatalf("GetAiDataPlatform calls = %d, want pre-delete, already-pending, and conflict-confirmation reads", fake.getCalls)
	}
	if got, want := resource.Status.LifecycleState, string(aidataplatformsdk.AiDataPlatformLifecycleStateDeleting); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want finalizer retained")
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestAiDataPlatformDeleteRejectsAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		return aidataplatformsdk.GetAiDataPlatformResponse{
			AiDataPlatform: aiDataPlatformBody(testAiDataPlatformExistingID, testAiDataPlatformCompartmentID, testAiDataPlatformDisplayName, aidataplatformsdk.AiDataPlatformLifecycleStateActive),
		}, nil
	}
	fake.deleteAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error) {
		return aidataplatformsdk.DeleteAiDataPlatformResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	}

	deleted, err := newTestAiDataPlatformClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "keeping the finalizer") {
		t.Fatalf("Delete() error = %q, want conservative finalizer message", err.Error())
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want finalizer retained")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAiDataPlatformDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAiDataPlatformExistingID)
	resource.Status.Id = testAiDataPlatformExistingID
	fake := &fakeAiDataPlatformOCIClient{}
	fake.getAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
		return aidataplatformsdk.GetAiDataPlatformResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	}

	deleted, err := newTestAiDataPlatformClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteAiDataPlatform calls = %d, want 0", fake.deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous auth-shaped message", err.Error())
	}
}

func TestAiDataPlatformCreateErrorCapturesOpcRequestID(t *testing.T) {
	t.Parallel()

	resource := newAiDataPlatformResource()
	fake := &fakeAiDataPlatformOCIClient{}
	fake.listAiDataPlatforms = func(_ context.Context, _ aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
		return aidataplatformsdk.ListAiDataPlatformsResponse{}, nil
	}
	fake.createAiDataPlatform = func(_ context.Context, _ aidataplatformsdk.CreateAiDataPlatformRequest) (aidataplatformsdk.CreateAiDataPlatformResponse, error) {
		return aidataplatformsdk.CreateAiDataPlatformResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "create is still settling")
	}

	response, err := newTestAiDataPlatformClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

type fakeAiDataPlatformOCIClient struct {
	createAiDataPlatform func(context.Context, aidataplatformsdk.CreateAiDataPlatformRequest) (aidataplatformsdk.CreateAiDataPlatformResponse, error)
	getAiDataPlatform    func(context.Context, aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error)
	listAiDataPlatforms  func(context.Context, aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error)
	updateAiDataPlatform func(context.Context, aidataplatformsdk.UpdateAiDataPlatformRequest) (aidataplatformsdk.UpdateAiDataPlatformResponse, error)
	deleteAiDataPlatform func(context.Context, aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error)
	getWorkRequest       func(context.Context, aidataplatformsdk.GetWorkRequestRequest) (aidataplatformsdk.GetWorkRequestResponse, error)

	createCalls      int
	getCalls         int
	listCalls        int
	updateCalls      int
	deleteCalls      int
	workRequestCalls int
}

func (f *fakeAiDataPlatformOCIClient) CreateAiDataPlatform(
	ctx context.Context,
	request aidataplatformsdk.CreateAiDataPlatformRequest,
) (aidataplatformsdk.CreateAiDataPlatformResponse, error) {
	f.createCalls++
	if f.createAiDataPlatform == nil {
		return aidataplatformsdk.CreateAiDataPlatformResponse{}, nil
	}
	return f.createAiDataPlatform(ctx, request)
}

func (f *fakeAiDataPlatformOCIClient) GetAiDataPlatform(
	ctx context.Context,
	request aidataplatformsdk.GetAiDataPlatformRequest,
) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
	f.getCalls++
	if f.getAiDataPlatform == nil {
		return aidataplatformsdk.GetAiDataPlatformResponse{}, nil
	}
	return f.getAiDataPlatform(ctx, request)
}

func (f *fakeAiDataPlatformOCIClient) ListAiDataPlatforms(
	ctx context.Context,
	request aidataplatformsdk.ListAiDataPlatformsRequest,
) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
	f.listCalls++
	if f.listAiDataPlatforms == nil {
		return aidataplatformsdk.ListAiDataPlatformsResponse{}, nil
	}
	return f.listAiDataPlatforms(ctx, request)
}

func (f *fakeAiDataPlatformOCIClient) UpdateAiDataPlatform(
	ctx context.Context,
	request aidataplatformsdk.UpdateAiDataPlatformRequest,
) (aidataplatformsdk.UpdateAiDataPlatformResponse, error) {
	f.updateCalls++
	if f.updateAiDataPlatform == nil {
		return aidataplatformsdk.UpdateAiDataPlatformResponse{}, nil
	}
	return f.updateAiDataPlatform(ctx, request)
}

func (f *fakeAiDataPlatformOCIClient) DeleteAiDataPlatform(
	ctx context.Context,
	request aidataplatformsdk.DeleteAiDataPlatformRequest,
) (aidataplatformsdk.DeleteAiDataPlatformResponse, error) {
	f.deleteCalls++
	if f.deleteAiDataPlatform == nil {
		return aidataplatformsdk.DeleteAiDataPlatformResponse{}, nil
	}
	return f.deleteAiDataPlatform(ctx, request)
}

func (f *fakeAiDataPlatformOCIClient) GetWorkRequest(
	ctx context.Context,
	request aidataplatformsdk.GetWorkRequestRequest,
) (aidataplatformsdk.GetWorkRequestResponse, error) {
	f.workRequestCalls++
	if f.getWorkRequest == nil {
		return aidataplatformsdk.GetWorkRequestResponse{}, nil
	}
	return f.getWorkRequest(ctx, request)
}

type aiDataPlatformListPage struct {
	wantPage string
	nextPage string
	items    []aidataplatformsdk.AiDataPlatformSummary
}

func pagedAiDataPlatformList(
	t *testing.T,
	pages []aiDataPlatformListPage,
) func(context.Context, aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
	t.Helper()
	remaining := append([]aiDataPlatformListPage(nil), pages...)
	return func(_ context.Context, request aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
		t.Helper()
		assertAiDataPlatformListRequest(t, request)
		if len(remaining) == 0 {
			t.Fatalf("unexpected ListAiDataPlatforms page request %q", stringValue(request.Page))
		}
		page := remaining[0]
		remaining = remaining[1:]
		if got := stringValue(request.Page); got != page.wantPage {
			t.Fatalf("ListAiDataPlatforms page = %q, want %q", got, page.wantPage)
		}
		response := aidataplatformsdk.ListAiDataPlatformsResponse{
			AiDataPlatformCollection: aidataplatformsdk.AiDataPlatformCollection{Items: page.items},
		}
		if page.nextPage != "" {
			response.OpcNextPage = common.String(page.nextPage)
		}
		return response, nil
	}
}

func newTestAiDataPlatformClient(client aiDataPlatformOCIClient) AiDataPlatformServiceClient {
	return newAiDataPlatformServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
	)
}

func newAiDataPlatformResource() *aidataplatformv1beta1.AiDataPlatform {
	return &aidataplatformv1beta1.AiDataPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAiDataPlatformDisplayName,
			Namespace: "default",
			UID:       k8stypes.UID("aidp-uid"),
		},
		Spec: aidataplatformv1beta1.AiDataPlatformSpec{
			CompartmentId:        testAiDataPlatformCompartmentID,
			DisplayName:          testAiDataPlatformDisplayName,
			AiDataPlatformType:   testAiDataPlatformType,
			DefaultWorkspaceName: "workspace-default",
			FreeformTags:         map[string]string{"managed-by": "osok"},
			DefinedTags:          map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			SystemTags:           map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "true"}},
		},
	}
}

func reconcileRequest(resource *aidataplatformv1beta1.AiDataPlatform) ctrl.Request {
	return ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func aiDataPlatformBody(
	id string,
	compartmentID string,
	displayName string,
	state aidataplatformsdk.AiDataPlatformLifecycleStateEnum,
) aidataplatformsdk.AiDataPlatform {
	return aidataplatformsdk.AiDataPlatform{
		Id:                 common.String(id),
		CompartmentId:      common.String(compartmentID),
		DisplayName:        common.String(displayName),
		AiDataPlatformType: common.String(testAiDataPlatformType),
		LifecycleState:     state,
		FreeformTags:       map[string]string{"managed-by": "osok"},
		DefinedTags:        map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:         map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func aiDataPlatformSummary(id string, compartmentID string, displayName string) aidataplatformsdk.AiDataPlatformSummary {
	return aidataplatformsdk.AiDataPlatformSummary{
		Id:                 common.String(id),
		CompartmentId:      common.String(compartmentID),
		DisplayName:        common.String(displayName),
		AiDataPlatformType: common.String(testAiDataPlatformType),
		LifecycleState:     aidataplatformsdk.AiDataPlatformLifecycleStateActive,
		FreeformTags:       map[string]string{"managed-by": "osok"},
		DefinedTags:        map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:         map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func aiDataPlatformWorkRequest(
	id string,
	operationType aidataplatformsdk.OperationTypeEnum,
	status aidataplatformsdk.OperationStatusEnum,
	action aidataplatformsdk.ActionTypeEnum,
	resourceID string,
) aidataplatformsdk.WorkRequest {
	return aidataplatformsdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		CompartmentId:   common.String(testAiDataPlatformCompartmentID),
		PercentComplete: common.Float32(50),
		Resources: []aidataplatformsdk.WorkRequestResource{
			{
				EntityType: common.String("data_lake"),
				ActionType: action,
				Identifier: common.String(resourceID),
				EntityUri:  common.String("/data-lakes/" + resourceID),
			},
		},
	}
}

func assertAiDataPlatformListRequest(t *testing.T, request aidataplatformsdk.ListAiDataPlatformsRequest) {
	t.Helper()
	if got, want := stringValue(request.CompartmentId), testAiDataPlatformCompartmentID; got != want {
		t.Fatalf("ListAiDataPlatforms compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.DisplayName), testAiDataPlatformDisplayName; got != want {
		t.Fatalf("ListAiDataPlatforms displayName = %q, want %q", got, want)
	}
}

func assertAiDataPlatformCreateRequest(t *testing.T, request aidataplatformsdk.CreateAiDataPlatformRequest) {
	t.Helper()
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateAiDataPlatform OpcRetryToken is empty")
	}
	if got, want := stringValue(request.CompartmentId), testAiDataPlatformCompartmentID; got != want {
		t.Fatalf("CreateAiDataPlatform compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.DefaultWorkspaceName), "workspace-default"; got != want {
		t.Fatalf("CreateAiDataPlatform defaultWorkspaceName = %q, want %q", got, want)
	}
}

func assertAiDataPlatformGetRequest(t *testing.T, request aidataplatformsdk.GetAiDataPlatformRequest, want string) {
	t.Helper()
	if got := stringValue(request.AiDataPlatformId); got != want {
		t.Fatalf("GetAiDataPlatform aiDataPlatformId = %q, want %q", got, want)
	}
}

func assertAiDataPlatformUpdateRequest(t *testing.T, request aidataplatformsdk.UpdateAiDataPlatformRequest) {
	t.Helper()
	if got, want := stringValue(request.AiDataPlatformId), testAiDataPlatformExistingID; got != want {
		t.Fatalf("UpdateAiDataPlatform aiDataPlatformId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.DisplayName), "aidp-updated"; got != want {
		t.Fatalf("UpdateAiDataPlatform displayName = %q, want %q", got, want)
	}
	if request.FreeformTags != nil || request.DefinedTags != nil || request.SystemTags != nil {
		t.Fatalf("UpdateAiDataPlatform tags = %#v/%#v/%#v, want omitted unchanged tags", request.FreeformTags, request.DefinedTags, request.SystemTags)
	}
}

func assertTrailingCondition(t *testing.T, resource *aidataplatformv1beta1.AiDataPlatform, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %s, want %s", got, want)
	}
}

func assertContainsAll(t *testing.T, got []string, want ...string) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("slice %#v missing %q", got, item)
		}
	}
}
