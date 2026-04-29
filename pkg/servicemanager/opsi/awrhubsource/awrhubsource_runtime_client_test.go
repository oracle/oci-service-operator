/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package awrhubsource

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestAwrHubSourceRuntimeHooksConfigureReviewedSemantics(t *testing.T) {
	hooks := newAwrHubSourceRuntimeHooksWithOCIClient(&fakeAwrHubSourceOCIClient{})
	applyAwrHubSourceRuntimeHooks(&AwrHubSourceServiceManager{}, &hooks, &fakeAwrHubSourceOCIClient{}, nil)

	assertAwrHubSourceReviewedSemantics(t, hooks)
	assertAwrHubSourceRuntimeHooksConfigured(t, hooks)
	assertAwrHubSourceUpdateBodyNoOp(t, hooks)
	assertAwrHubSourceCreateBody(t, hooks)
}

func assertAwrHubSourceReviewedSemantics(t *testing.T, hooks AwrHubSourceRuntimeHooks) {
	t.Helper()
	if hooks.Semantics == nil {
		t.Fatal("Semantics = nil, want reviewed generatedruntime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got)
	}
	assertAwrHubSourceStringSliceContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "type", "freeformTags", "definedTags")
	assertAwrHubSourceStringSliceContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "name", "awrHubId", "compartmentId", "associatedResourceId", "associatedOpsiId")
}

func assertAwrHubSourceRuntimeHooksConfigured(t *testing.T, hooks AwrHubSourceRuntimeHooks) {
	t.Helper()
	if hooks.StatusHooks.ProjectStatus == nil {
		t.Fatal("StatusHooks.ProjectStatus = nil, want sdk status collision-safe projection")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("Async.GetWorkRequest = nil, want work request observation")
	}
}

func assertAwrHubSourceUpdateBodyNoOp(t *testing.T, hooks AwrHubSourceRuntimeHooks) {
	t.Helper()
	body, updateNeeded, err := hooks.BuildUpdateBody(context.Background(), awrHubSourceResource(), "default", opsisdk.GetAwrHubSourceResponse{
		AwrHubSource: awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive),
	})
	if err != nil {
		t.Fatalf("BuildUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("BuildUpdateBody() updateNeeded = true with matching current state; body = %#v", body)
	}
}

func TestAwrHubSourceUpdateBodyTrimsDesiredTypeBeforeComparison(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Spec.Type = "  " + string(opsisdk.AwrHubSourceTypeExternalPdb) + " \t"

	body, updateNeeded, err := buildAwrHubSourceUpdateBody(resource, opsisdk.GetAwrHubSourceResponse{
		AwrHubSource: awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive),
	})
	if err != nil {
		t.Fatalf("buildAwrHubSourceUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("buildAwrHubSourceUpdateBody() updateNeeded = true for whitespace-normalized type; body = %#v", body)
	}
}

func assertAwrHubSourceCreateBody(t *testing.T, hooks AwrHubSourceRuntimeHooks) {
	t.Helper()
	createBody, err := hooks.BuildCreateBody(context.Background(), awrHubSourceResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	createDetails, ok := createBody.(opsisdk.CreateAwrHubSourceDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want opsi.CreateAwrHubSourceDetails", createBody)
	}
	if got := awrHubSourceStringValue(createDetails.Name); got != "source-a" {
		t.Fatalf("Create body name = %q, want source-a", got)
	}
}

func TestAwrHubSourceCreateOrUpdateCreatesAndTracksWorkRequest(t *testing.T) {
	resource := awrHubSourceResource()
	fake := &fakeAwrHubSourceOCIClient{}
	fake.list = func(_ context.Context, _ opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error) {
		return opsisdk.ListAwrHubSourcesResponse{}, nil
	}
	fake.create = func(_ context.Context, request opsisdk.CreateAwrHubSourceRequest) (opsisdk.CreateAwrHubSourceResponse, error) {
		if got := awrHubSourceStringValue(request.Name); got != resource.Spec.Name {
			t.Fatalf("Create request name = %q, want %q", got, resource.Spec.Name)
		}
		if got := request.Type; got != opsisdk.AwrHubSourceTypeExternalPdb {
			t.Fatalf("Create request type = %q, want %q", got, opsisdk.AwrHubSourceTypeExternalPdb)
		}
		return opsisdk.CreateAwrHubSourceResponse{
			AwrHubSource:     awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateCreating),
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		if got := awrHubSourceStringValue(request.WorkRequestId); got != "wr-create" {
			t.Fatalf("GetWorkRequest id = %q, want wr-create", got)
		}
		return opsisdk.GetWorkRequestResponse{WorkRequest: awrHubSourceWorkRequest("wr-create", opsisdk.OperationTypeCreateAwrhubSource, opsisdk.OperationStatusInProgress, "awr-src-1")}, nil
	}

	response, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue for pending work request", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("Create calls = %d, want 1", len(fake.createRequests))
	}
	assertAwrHubSourceAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
	if got := resource.Status.Status; got != string(opsisdk.AwrHubSourceStatusAccepting) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, opsisdk.AwrHubSourceStatusAccepting)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", got)
	}
}

func TestAwrHubSourceCreateOrUpdateBindsFromPaginatedListWithoutCreate(t *testing.T) {
	resource := awrHubSourceResource()
	fake := &fakeAwrHubSourceOCIClient{}
	fake.list = func(_ context.Context, request opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error) {
		switch page := awrHubSourceStringValue(request.Page); page {
		case "":
			return opsisdk.ListAwrHubSourcesResponse{
				AwrHubSourceSummaryCollection: opsisdk.AwrHubSourceSummaryCollection{
					Items: []opsisdk.AwrHubSourceSummary{awrHubSourceSummary("other", "other-name", opsisdk.AwrHubSourceLifecycleStateActive)},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return opsisdk.ListAwrHubSourcesResponse{
				AwrHubSourceSummaryCollection: opsisdk.AwrHubSourceSummaryCollection{
					Items: []opsisdk.AwrHubSourceSummary{awrHubSourceSummary("awr-src-1", resource.Spec.Name, opsisdk.AwrHubSourceLifecycleStateActive)},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", page)
			return opsisdk.ListAwrHubSourcesResponse{}, nil
		}
	}
	fake.get = func(_ context.Context, request opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		if got := awrHubSourceStringValue(request.AwrHubSourceId); got != "awr-src-1" {
			t.Fatalf("Get id = %q, want awr-src-1", got)
		}
		return opsisdk.GetAwrHubSourceResponse{AwrHubSource: awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive)}, nil
	}

	response, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("Create calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("List calls = %d, want 2 for pagination", len(fake.listRequests))
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID("awr-src-1") {
		t.Fatalf("status.ocid = %q, want awr-src-1", got)
	}
}

func TestAwrHubSourceCreateOrUpdateNoOpDoesNotUpdate(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		return opsisdk.GetAwrHubSourceResponse{AwrHubSource: awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive)}, nil
	}

	response, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active no-op without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestAwrHubSourceCreateOrUpdateMutableUpdateUsesUpdatePath(t *testing.T) {
	resource := mutableUpdateAwrHubSourceResource()
	fake := mutableUpdateAwrHubSourceFake(t)

	response, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertAwrHubSourceMutableUpdateResult(t, resource, response, fake)
}

func mutableUpdateAwrHubSourceResource() *opsiv1beta1.AwrHubSource {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	resource.Spec.Type = string(opsisdk.AwrHubSourceTypeExternalNoncdb)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	return resource
}

func mutableUpdateAwrHubSourceFake(t *testing.T) *fakeAwrHubSourceOCIClient {
	t.Helper()
	getResponses := []opsisdk.AwrHubSource{
		awrHubSourceWithMutableState(opsisdk.AwrHubSourceTypeExternalPdb, map[string]string{"env": "dev"}),
		awrHubSourceWithMutableState(opsisdk.AwrHubSourceTypeExternalNoncdb, map[string]string{"env": "prod"}),
	}
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		current := getResponses[awrHubSourceMin(len(fake.getRequests)-1, len(getResponses)-1)]
		return opsisdk.GetAwrHubSourceResponse{AwrHubSource: current}, nil
	}
	fake.update = func(_ context.Context, request opsisdk.UpdateAwrHubSourceRequest) (opsisdk.UpdateAwrHubSourceResponse, error) {
		assertAwrHubSourceMutableUpdateRequest(t, request)
		return opsisdk.UpdateAwrHubSourceResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		workRequest := awrHubSourceWorkRequest("wr-update", opsisdk.OperationTypeUpdateAwrhubSource, opsisdk.OperationStatusSucceeded, "awr-src-1")
		return opsisdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}
	return fake
}

func awrHubSourceWithMutableState(sourceType opsisdk.AwrHubSourceTypeEnum, freeformTags map[string]string) opsisdk.AwrHubSource {
	current := awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive)
	current.Type = sourceType
	current.FreeformTags = freeformTags
	return current
}

func assertAwrHubSourceMutableUpdateRequest(t *testing.T, request opsisdk.UpdateAwrHubSourceRequest) {
	t.Helper()
	if got := awrHubSourceStringValue(request.AwrHubSourceId); got != "awr-src-1" {
		t.Fatalf("Update id = %q, want awr-src-1", got)
	}
	if got := request.Type; got != opsisdk.AwrHubSourceTypeExternalNoncdb {
		t.Fatalf("Update type = %q, want %q", got, opsisdk.AwrHubSourceTypeExternalNoncdb)
	}
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("Update freeformTags[env] = %q, want prod", got)
	}
}

func assertAwrHubSourceMutableUpdateResult(
	t *testing.T,
	resource *opsiv1beta1.AwrHubSource,
	response servicemanager.OSOKResponse,
	fake *fakeAwrHubSourceOCIClient,
) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want completed update without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.Type; got != string(opsisdk.AwrHubSourceTypeExternalNoncdb) {
		t.Fatalf("status.type = %q, want %q", got, opsisdk.AwrHubSourceTypeExternalNoncdb)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
}

func awrHubSourceMin(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestAwrHubSourceCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	resource.Spec.AssociatedResourceId = "db-new"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		current := awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive)
		current.AssociatedResourceId = common.String("db-old")
		current.FreeformTags = map[string]string{"env": "dev"}
		return opsisdk.GetAwrHubSourceResponse{AwrHubSource: current}, nil
	}

	_, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "associatedResourceId") {
		t.Fatalf("CreateOrUpdate() error = %v, want associatedResourceId create-only drift", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestAwrHubSourceOmittedCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	resource.Spec.AssociatedResourceId = ""
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		current := awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive)
		current.AssociatedResourceId = common.String("db-1")
		current.FreeformTags = map[string]string{"env": "dev"}
		return opsisdk.GetAwrHubSourceResponse{AwrHubSource: current}, nil
	}

	_, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "associatedResourceId") {
		t.Fatalf("CreateOrUpdate() error = %v, want omitted associatedResourceId create-only drift", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestAwrHubSourceDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeAwrHubSourceOCIClient{}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{WorkRequest: awrHubSourceWorkRequest("wr-create", opsisdk.OperationTypeCreateAwrhubSource, opsisdk.OperationStatusInProgress, "awr-src-1")}, nil
	}

	deleted, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while create work request is pending")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("Delete OCI calls = %d, want 0", len(fake.deleteRequests))
	}
	assertAwrHubSourceAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
}

func TestAwrHubSourceDeleteWaitsWhenSucceededWriteReadbackMisses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		phase     shared.OSOKAsyncPhase
		operation opsisdk.OperationTypeEnum
	}{
		{
			name:      "create",
			phase:     shared.OSOKAsyncPhaseCreate,
			operation: opsisdk.OperationTypeCreateAwrhubSource,
		},
		{
			name:      "update",
			phase:     shared.OSOKAsyncPhaseUpdate,
			operation: opsisdk.OperationTypeUpdateAwrhubSource,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource, fake, workRequestID := awrHubSourceSucceededWriteReadbackMissFixture(t, tc.name, tc.phase, tc.operation)

			deleted, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
			assertAwrHubSourceDeleteWaitedForWriteReadback(t, resource, fake, deleted, err, tc.phase, workRequestID)
		})
	}
}

func awrHubSourceSucceededWriteReadbackMissFixture(
	t *testing.T,
	name string,
	phase shared.OSOKAsyncPhase,
	operation opsisdk.OperationTypeEnum,
) (*opsiv1beta1.AwrHubSource, *fakeAwrHubSourceOCIClient, string) {
	t.Helper()
	resource := awrHubSourceResource()
	workRequestID := "wr-" + name
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeAwrHubSourceOCIClient{}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: awrHubSourceWorkRequest(workRequestID, operation, opsisdk.OperationStatusSucceeded, "awr-src-1"),
		}, nil
	}
	fake.get = func(_ context.Context, request opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		if got := awrHubSourceStringValue(request.AwrHubSourceId); got != "awr-src-1" {
			t.Fatalf("Get id = %q, want awr-src-1", got)
		}
		return opsisdk.GetAwrHubSourceResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not yet readable")
	}
	return resource, fake, workRequestID
}

func assertAwrHubSourceDeleteWaitedForWriteReadback(
	t *testing.T,
	resource *opsiv1beta1.AwrHubSource,
	fake *fakeAwrHubSourceOCIClient,
	deleted bool,
	err error,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil while write readback is not yet visible", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while write readback is not visible")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("Delete OCI calls = %d, want 0 before write readback is visible", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want unset while write readback is not visible")
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID("awr-src-1") {
		t.Fatalf("status.ocid = %q, want awr-src-1 recovered from work request", got)
	}
	assertAwrHubSourceAsync(t, resource, phase, shared.OSOKAsyncClassPending, workRequestID)
	if got := resource.Status.OsokStatus.Async.Current.Message; !strings.Contains(got, "waiting") {
		t.Fatalf("async message = %q, want waiting detail", got)
	}
}

func TestAwrHubSourceDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		return opsisdk.GetAwrHubSourceResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("Delete OCI calls = %d, want 0", len(fake.deleteRequests))
	}
}

func TestAwrHubSourceDeleteStartsWorkRequestAndRetainsFinalizer(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		return opsisdk.GetAwrHubSourceResponse{AwrHubSource: awrHubSourceSDK("awr-src-1", opsisdk.AwrHubSourceLifecycleStateActive)}, nil
	}
	fake.delete = func(_ context.Context, _ opsisdk.DeleteAwrHubSourceRequest) (opsisdk.DeleteAwrHubSourceResponse, error) {
		return opsisdk.DeleteAwrHubSourceResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}

	deleted, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for delete work request")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("Delete OCI calls = %d, want 1", len(fake.deleteRequests))
	}
	assertAwrHubSourceAsync(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
}

func TestAwrHubSourceSucceededDeleteWorkRequestKeepsAuthShapedReadFatal(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeAwrHubSourceOCIClient{}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{WorkRequest: awrHubSourceWorkRequest("wr-delete", opsisdk.OperationTypeDeleteAwrhubSource, opsisdk.OperationStatusSucceeded, "awr-src-1")}, nil
	}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		return opsisdk.GetAwrHubSourceResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped confirm read to stay fatal", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want unset")
	}
}

func TestAwrHubSourceDeleteReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	resource := awrHubSourceResource()
	resource.Status.OsokStatus.Ocid = "awr-src-1"
	fake := &fakeAwrHubSourceOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
		return opsisdk.GetAwrHubSourceResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}

	deleted, err := newAwrHubSourceServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release for unambiguous NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

type fakeAwrHubSourceOCIClient struct {
	createRequests         []opsisdk.CreateAwrHubSourceRequest
	getRequests            []opsisdk.GetAwrHubSourceRequest
	listRequests           []opsisdk.ListAwrHubSourcesRequest
	updateRequests         []opsisdk.UpdateAwrHubSourceRequest
	deleteRequests         []opsisdk.DeleteAwrHubSourceRequest
	getWorkRequestRequests []opsisdk.GetWorkRequestRequest

	create         func(context.Context, opsisdk.CreateAwrHubSourceRequest) (opsisdk.CreateAwrHubSourceResponse, error)
	get            func(context.Context, opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error)
	list           func(context.Context, opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error)
	update         func(context.Context, opsisdk.UpdateAwrHubSourceRequest) (opsisdk.UpdateAwrHubSourceResponse, error)
	delete         func(context.Context, opsisdk.DeleteAwrHubSourceRequest) (opsisdk.DeleteAwrHubSourceResponse, error)
	getWorkRequest func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeAwrHubSourceOCIClient) CreateAwrHubSource(
	ctx context.Context,
	request opsisdk.CreateAwrHubSourceRequest,
) (opsisdk.CreateAwrHubSourceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create != nil {
		return f.create(ctx, request)
	}
	return opsisdk.CreateAwrHubSourceResponse{}, nil
}

func (f *fakeAwrHubSourceOCIClient) GetAwrHubSource(
	ctx context.Context,
	request opsisdk.GetAwrHubSourceRequest,
) (opsisdk.GetAwrHubSourceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get != nil {
		return f.get(ctx, request)
	}
	return opsisdk.GetAwrHubSourceResponse{}, nil
}

func (f *fakeAwrHubSourceOCIClient) ListAwrHubSources(
	ctx context.Context,
	request opsisdk.ListAwrHubSourcesRequest,
) (opsisdk.ListAwrHubSourcesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list != nil {
		return f.list(ctx, request)
	}
	return opsisdk.ListAwrHubSourcesResponse{}, nil
}

func (f *fakeAwrHubSourceOCIClient) UpdateAwrHubSource(
	ctx context.Context,
	request opsisdk.UpdateAwrHubSourceRequest,
) (opsisdk.UpdateAwrHubSourceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update != nil {
		return f.update(ctx, request)
	}
	return opsisdk.UpdateAwrHubSourceResponse{}, nil
}

func (f *fakeAwrHubSourceOCIClient) DeleteAwrHubSource(
	ctx context.Context,
	request opsisdk.DeleteAwrHubSourceRequest,
) (opsisdk.DeleteAwrHubSourceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete != nil {
		return f.delete(ctx, request)
	}
	return opsisdk.DeleteAwrHubSourceResponse{}, nil
}

func (f *fakeAwrHubSourceOCIClient) GetWorkRequest(
	ctx context.Context,
	request opsisdk.GetWorkRequestRequest,
) (opsisdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequest != nil {
		return f.getWorkRequest(ctx, request)
	}
	return opsisdk.GetWorkRequestResponse{}, nil
}

func awrHubSourceResource() *opsiv1beta1.AwrHubSource {
	return &opsiv1beta1.AwrHubSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "source-a",
			Namespace: "default",
			UID:       types.UID("awrhubsource-uid"),
		},
		Spec: opsiv1beta1.AwrHubSourceSpec{
			Name:                 "source-a",
			AwrHubId:             "awr-hub-1",
			CompartmentId:        "compartment-1",
			Type:                 string(opsisdk.AwrHubSourceTypeExternalPdb),
			AssociatedResourceId: "db-1",
			FreeformTags:         map[string]string{"env": "dev"},
			DefinedTags:          map[string]shared.MapValue{"ns": {"key": "value"}},
		},
	}
}

func awrHubSourceSDK(id string, state opsisdk.AwrHubSourceLifecycleStateEnum) opsisdk.AwrHubSource {
	return opsisdk.AwrHubSource{
		Name:                   common.String("source-a"),
		AwrHubId:               common.String("awr-hub-1"),
		CompartmentId:          common.String("compartment-1"),
		Type:                   opsisdk.AwrHubSourceTypeExternalPdb,
		Id:                     common.String(id),
		AwrHubOpsiSourceId:     common.String("short-" + id),
		SourceMailBoxUrl:       common.String("https://example.invalid/mailbox"),
		LifecycleState:         state,
		Status:                 opsisdk.AwrHubSourceStatusAccepting,
		AssociatedResourceId:   common.String("db-1"),
		FreeformTags:           map[string]string{"env": "dev"},
		DefinedTags:            map[string]map[string]interface{}{"ns": {"key": "value"}},
		IsRegisteredWithAwrHub: common.Bool(true),
	}
}

func awrHubSourceSummary(id string, name string, state opsisdk.AwrHubSourceLifecycleStateEnum) opsisdk.AwrHubSourceSummary {
	current := awrHubSourceSDK(id, state)
	current.Name = common.String(name)
	return opsisdk.AwrHubSourceSummary{
		Name:                   current.Name,
		AwrHubId:               current.AwrHubId,
		CompartmentId:          current.CompartmentId,
		Type:                   current.Type,
		Id:                     current.Id,
		AwrHubOpsiSourceId:     current.AwrHubOpsiSourceId,
		SourceMailBoxUrl:       current.SourceMailBoxUrl,
		LifecycleState:         current.LifecycleState,
		Status:                 current.Status,
		AssociatedResourceId:   current.AssociatedResourceId,
		FreeformTags:           current.FreeformTags,
		DefinedTags:            current.DefinedTags,
		IsRegisteredWithAwrHub: current.IsRegisteredWithAwrHub,
	}
}

func awrHubSourceWorkRequest(
	id string,
	operation opsisdk.OperationTypeEnum,
	status opsisdk.OperationStatusEnum,
	resourceID string,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operation,
		Status:          status,
		CompartmentId:   common.String("compartment-1"),
		PercentComplete: common.Float32(50),
		Resources: []opsisdk.WorkRequestResource{{
			EntityType: common.String("AwrHubSource"),
			ActionType: awrHubSourceActionForOperation(operation, status),
			Identifier: common.String(resourceID),
			EntityUri:  common.String("/awrHubSources/" + resourceID),
		}},
	}
}

func awrHubSourceActionForOperation(operation opsisdk.OperationTypeEnum, status opsisdk.OperationStatusEnum) opsisdk.ActionTypeEnum {
	if status == opsisdk.OperationStatusInProgress || status == opsisdk.OperationStatusAccepted || status == opsisdk.OperationStatusWaiting {
		return opsisdk.ActionTypeInProgress
	}
	switch operation {
	case opsisdk.OperationTypeCreateAwrhubSource:
		return opsisdk.ActionTypeCreated
	case opsisdk.OperationTypeUpdateAwrhubSource:
		return opsisdk.ActionTypeUpdated
	case opsisdk.OperationTypeDeleteAwrhubSource:
		return opsisdk.ActionTypeDeleted
	default:
		return opsisdk.ActionTypeRelated
	}
}

func assertAwrHubSourceAsync(
	t *testing.T,
	resource *opsiv1beta1.AwrHubSource,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Phase != phase {
		t.Fatalf("async.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != class {
		t.Fatalf("async.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("async.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func assertAwrHubSourceStringSliceContainsAll(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	for _, value := range want {
		found := false
		for _, candidate := range got {
			if candidate == value {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s = %#v, missing %q", name, got, value)
		}
	}
}

func logger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
