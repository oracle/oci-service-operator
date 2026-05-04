/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lockbox

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	lockboxsdk "github.com/oracle/oci-go-sdk/v65/lockbox"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestLockboxCreateOrUpdateCreatesWithStableIdentity(t *testing.T) {
	resource := baseLockboxResource()
	fake := &fakeLockboxOCIClient{}
	fake.list = func(_ context.Context, request lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error) {
		if got, want := stringValue(request.ResourceId), resource.Spec.ResourceId; got != want {
			t.Fatalf("ListLockboxes() resourceId = %q, want %q", got, want)
		}
		return lockboxsdk.ListLockboxesResponse{}, nil
	}
	fake.create = func(_ context.Context, request lockboxsdk.CreateLockboxRequest) (lockboxsdk.CreateLockboxResponse, error) {
		assertLockboxCreateRequest(t, request, resource)
		return lockboxsdk.CreateLockboxResponse{
			Lockbox:      sdkLockbox("ocid1.lockbox.oc1..created", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
			OpcRequestId: common.String("opc-create-1"),
		}, nil
	}
	fake.get = func(_ context.Context, request lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
		if got, want := stringValue(request.LockboxId), "ocid1.lockbox.oc1..created"; got != want {
			t.Fatalf("GetLockbox() lockboxId = %q, want %q", got, want)
		}
		return lockboxsdk.GetLockboxResponse{
			Lockbox:      sdkLockbox("ocid1.lockbox.oc1..created", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
			OpcRequestId: common.String("opc-get-1"),
		}, nil
	}

	response, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "lockbox"}})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateLockbox() calls = %d, want 1", len(fake.createRequests))
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.lockbox.oc1..created"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.Id, "ocid1.lockbox.oc1..created"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLockboxCreateOrUpdateBindsFromPaginatedListWithoutDuplicateCreate(t *testing.T) {
	resource := baseLockboxResource()
	fake := &fakeLockboxOCIClient{}
	fake.list = func(_ context.Context, request lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error) {
		switch page := stringValue(request.Page); page {
		case "":
			otherSpec := resource.Spec
			otherSpec.ResourceId = "ocid1.instance.oc1..other"
			return lockboxsdk.ListLockboxesResponse{
				LockboxCollection: lockboxsdk.LockboxCollection{
					Items: []lockboxsdk.LockboxSummary{
						sdkLockboxSummary("ocid1.lockbox.oc1..other", otherSpec, lockboxsdk.LockboxLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return lockboxsdk.ListLockboxesResponse{
				LockboxCollection: lockboxsdk.LockboxCollection{
					Items: []lockboxsdk.LockboxSummary{
						sdkLockboxSummary("ocid1.lockbox.oc1..bound", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("ListLockboxes() page = %q, want first page or page-2", page)
			return lockboxsdk.ListLockboxesResponse{}, nil
		}
	}
	fake.get = func(_ context.Context, request lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
		if got, want := stringValue(request.LockboxId), "ocid1.lockbox.oc1..bound"; got != want {
			t.Fatalf("GetLockbox() lockboxId = %q, want %q", got, want)
		}
		return lockboxsdk.GetLockboxResponse{
			Lockbox: sdkLockbox("ocid1.lockbox.oc1..bound", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
		}, nil
	}

	response, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateLockbox() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateLockbox() calls = %d, want 0", len(fake.updateRequests))
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.lockbox.oc1..bound"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestLockboxCreateOrUpdateBindsAndUpdatesWhenDisplayNameDiffers(t *testing.T) {
	resource := baseLockboxResource()
	currentSpec := resource.Spec
	currentSpec.DisplayName = "existing-lockbox"
	fake := &fakeLockboxOCIClient{}
	fake.list = func(_ context.Context, request lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error) {
		assertLockboxBindListRequest(t, request, resource)
		return lockboxsdk.ListLockboxesResponse{
			LockboxCollection: lockboxsdk.LockboxCollection{
				Items: []lockboxsdk.LockboxSummary{
					sdkLockboxSummary("ocid1.lockbox.oc1..renamed", currentSpec, lockboxsdk.LockboxLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.get = lockboxGetSequence(
		t,
		"ocid1.lockbox.oc1..renamed",
		currentSpec,
		resource.Spec,
	)
	fake.update = func(_ context.Context, request lockboxsdk.UpdateLockboxRequest) (lockboxsdk.UpdateLockboxResponse, error) {
		assertLockboxDisplayNameUpdateRequest(t, request, "ocid1.lockbox.oc1..renamed", resource.Spec.DisplayName)
		return lockboxsdk.UpdateLockboxResponse{
			Lockbox:      sdkLockbox("ocid1.lockbox.oc1..renamed", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
			OpcRequestId: common.String("opc-update-rename-1"),
		}, nil
	}

	response, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateLockbox() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateLockbox() calls = %d, want 1", len(fake.updateRequests))
	}
	if got, want := resource.Status.DisplayName, resource.Spec.DisplayName; got != want {
		t.Fatalf("status.displayName = %q, want %q", got, want)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.lockbox.oc1..renamed"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestLockboxCreateOrUpdateUpdatesMutableFieldsOnly(t *testing.T) {
	resource := baseLockboxResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.lockbox.oc1..existing")
	resource.Spec.DisplayName = "updated-lockbox"
	resource.Spec.ApprovalTemplateId = "ocid1.approvaltemplate.oc1..new"
	fake := &fakeLockboxOCIClient{}
	currentSpec := resource.Spec
	currentSpec.DisplayName = "old-lockbox"
	currentSpec.ApprovalTemplateId = "ocid1.approvaltemplate.oc1..old"
	fake.get = lockboxGetSequence(t, "ocid1.lockbox.oc1..existing", currentSpec, resource.Spec)
	fake.update = func(_ context.Context, request lockboxsdk.UpdateLockboxRequest) (lockboxsdk.UpdateLockboxResponse, error) {
		assertLockboxMutableUpdateRequest(t, request, resource)
		return lockboxsdk.UpdateLockboxResponse{
			Lockbox:      sdkLockbox("ocid1.lockbox.oc1..existing", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
			OpcRequestId: common.String("opc-update-1"),
		}, nil
	}

	response, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateLockbox() calls = %d, want 1", len(fake.updateRequests))
	}
	if got, want := resource.Status.DisplayName, resource.Spec.DisplayName; got != want {
		t.Fatalf("status.displayName = %q, want %q", got, want)
	}
}

func TestLockboxCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := baseLockboxResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.lockbox.oc1..existing")
	resource.Spec.ResourceId = "ocid1.instance.oc1..new"
	fake := &fakeLockboxOCIClient{}
	fake.get = func(_ context.Context, _ lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
		currentSpec := resource.Spec
		currentSpec.ResourceId = "ocid1.instance.oc1..old"
		return lockboxsdk.GetLockboxResponse{
			Lockbox: sdkLockbox("ocid1.lockbox.oc1..existing", currentSpec, lockboxsdk.LockboxLifecycleStateActive),
		}, nil
	}

	response, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when resourceId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only resourceId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateLockbox() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestLockboxDeleteRetainsFinalizerUntilReadbackConfirmsDeletion(t *testing.T) {
	tests := []struct {
		name        string
		confirmErr  error
		confirm     lockboxsdk.LockboxLifecycleStateEnum
		wantDeleted bool
	}{
		{
			name:        "pending delete",
			confirm:     lockboxsdk.LockboxLifecycleStateDeleting,
			wantDeleted: false,
		},
		{
			name:        "not found after delete",
			confirmErr:  errortest.NewServiceError(404, errorutil.NotFound, "lockbox deleted"),
			wantDeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLockboxDeleteRetentionCase(t, tt.confirmErr, tt.confirm, tt.wantDeleted)
		})
	}
}

func TestLockboxDeleteRejectsAuthShapedNotFound(t *testing.T) {
	tests := []struct {
		name       string
		getError   error
		deleteErr  error
		wantDelete bool
	}{
		{
			name:       "pre-delete read is ambiguous",
			getError:   errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous read"),
			wantDelete: false,
		},
		{
			name:       "delete response is ambiguous",
			deleteErr:  errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete"),
			wantDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLockboxDeleteAmbiguousNotFoundCase(t, tt.getError, tt.deleteErr, tt.wantDelete)
		})
	}
}

func assertLockboxCreateRequest(
	t *testing.T,
	request lockboxsdk.CreateLockboxRequest,
	resource *lockboxv1beta1.Lockbox,
) {
	t.Helper()
	if request.OpcRetryToken == nil || *request.OpcRetryToken == "" {
		t.Fatal("CreateLockbox() opc retry token is empty")
	}
	if got, want := stringValue(request.ResourceId), resource.Spec.ResourceId; got != want {
		t.Fatalf("CreateLockbox() resourceId = %q, want %q", got, want)
	}
	if got := request.AccessContextAttributes; got == nil || len(got.Items) != 1 || stringValue(got.Items[0].Name) != "ticket" {
		t.Fatalf("CreateLockbox() accessContextAttributes = %#v, want ticket attribute", got)
	}
}

func assertLockboxBindListRequest(
	t *testing.T,
	request lockboxsdk.ListLockboxesRequest,
	resource *lockboxv1beta1.Lockbox,
) {
	t.Helper()
	if request.DisplayName != nil {
		t.Fatalf("ListLockboxes() displayName = %q, want omitted for mutable displayName", stringValue(request.DisplayName))
	}
	if got, want := stringValue(request.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("ListLockboxes() compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.ResourceId), resource.Spec.ResourceId; got != want {
		t.Fatalf("ListLockboxes() resourceId = %q, want %q", got, want)
	}
}

func lockboxGetSequence(
	t *testing.T,
	lockboxID string,
	before lockboxv1beta1.LockboxSpec,
	after lockboxv1beta1.LockboxSpec,
) func(context.Context, lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
	t.Helper()
	getCalls := 0
	return func(_ context.Context, request lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
		if got := stringValue(request.LockboxId); got != lockboxID {
			t.Fatalf("GetLockbox() lockboxId = %q, want %q", got, lockboxID)
		}
		getCalls++
		currentSpec := after
		if getCalls == 1 {
			currentSpec = before
		}
		return lockboxsdk.GetLockboxResponse{
			Lockbox: sdkLockbox(lockboxID, currentSpec, lockboxsdk.LockboxLifecycleStateActive),
		}, nil
	}
}

func assertLockboxDisplayNameUpdateRequest(
	t *testing.T,
	request lockboxsdk.UpdateLockboxRequest,
	lockboxID string,
	displayName string,
) {
	t.Helper()
	if got := stringValue(request.LockboxId); got != lockboxID {
		t.Fatalf("UpdateLockbox() lockboxId = %q, want %q", got, lockboxID)
	}
	if got := stringValue(request.DisplayName); got != displayName {
		t.Fatalf("UpdateLockbox() displayName = %q, want %q", got, displayName)
	}
}

func assertLockboxMutableUpdateRequest(
	t *testing.T,
	request lockboxsdk.UpdateLockboxRequest,
	resource *lockboxv1beta1.Lockbox,
) {
	t.Helper()
	assertLockboxDisplayNameUpdateRequest(
		t,
		request,
		"ocid1.lockbox.oc1..existing",
		resource.Spec.DisplayName,
	)
	if got, want := stringValue(request.ApprovalTemplateId), resource.Spec.ApprovalTemplateId; got != want {
		t.Fatalf("UpdateLockbox() approvalTemplateId = %q, want %q", got, want)
	}
	if request.MaxAccessDuration != nil {
		t.Fatalf("UpdateLockbox() maxAccessDuration = %q, want omitted unchanged optional", stringValue(request.MaxAccessDuration))
	}
}

func runLockboxDeleteRetentionCase(
	t *testing.T,
	confirmErr error,
	confirm lockboxsdk.LockboxLifecycleStateEnum,
	wantDeleted bool,
) {
	t.Helper()
	resource := baseLockboxResource()
	resource.Status.Id = "ocid1.lockbox.oc1..delete"
	fake := &fakeLockboxOCIClient{
		get:    deleteRetentionGet(t, resource, confirmErr, confirm),
		delete: successfulLockboxDelete,
	}

	deleted, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted != wantDeleted {
		t.Fatalf("Delete() = %t, want %t", deleted, wantDeleted)
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteLockbox() calls = %d, want 1", len(fake.deleteRequests))
	}
	assertLockboxDeletedAt(t, resource, wantDeleted)
}

func deleteRetentionGet(
	t *testing.T,
	resource *lockboxv1beta1.Lockbox,
	confirmErr error,
	confirm lockboxsdk.LockboxLifecycleStateEnum,
) func(context.Context, lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
	t.Helper()
	getCalls := 0
	return func(_ context.Context, _ lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
		getCalls++
		if getCalls <= 2 {
			return lockboxsdk.GetLockboxResponse{
				Lockbox: sdkLockbox("ocid1.lockbox.oc1..delete", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
			}, nil
		}
		if confirmErr != nil {
			return lockboxsdk.GetLockboxResponse{}, confirmErr
		}
		return lockboxsdk.GetLockboxResponse{
			Lockbox: sdkLockbox("ocid1.lockbox.oc1..delete", resource.Spec, confirm),
		}, nil
	}
}

func successfulLockboxDelete(
	context.Context,
	lockboxsdk.DeleteLockboxRequest,
) (lockboxsdk.DeleteLockboxResponse, error) {
	return lockboxsdk.DeleteLockboxResponse{OpcRequestId: common.String("opc-delete-1")}, nil
}

func assertLockboxDeletedAt(t *testing.T, resource *lockboxv1beta1.Lockbox, wantDeleted bool) {
	t.Helper()
	if !wantDeleted && resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set while delete is still pending")
	}
	if wantDeleted && resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt was not set after delete confirmation")
	}
}

func runLockboxDeleteAmbiguousNotFoundCase(t *testing.T, getError error, deleteErr error, wantDelete bool) {
	t.Helper()
	resource := baseLockboxResource()
	resource.Status.Id = "ocid1.lockbox.oc1..ambiguous"
	fake := &fakeLockboxOCIClient{
		get:    ambiguousNotFoundGet(t, resource, getError),
		delete: ambiguousNotFoundDelete(deleteErr),
	}

	deleted, err := newLockboxServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake).
		Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found rejection", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained for ambiguous not-found")
	}
	if gotDelete := len(fake.deleteRequests) != 0; gotDelete != wantDelete {
		t.Fatalf("DeleteLockbox() called = %t, want %t", gotDelete, wantDelete)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func ambiguousNotFoundGet(
	t *testing.T,
	resource *lockboxv1beta1.Lockbox,
	getError error,
) func(context.Context, lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
	t.Helper()
	getCalls := 0
	return func(_ context.Context, _ lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
		getCalls++
		if getCalls == 1 && getError != nil {
			return lockboxsdk.GetLockboxResponse{}, getError
		}
		return lockboxsdk.GetLockboxResponse{
			Lockbox: sdkLockbox("ocid1.lockbox.oc1..ambiguous", resource.Spec, lockboxsdk.LockboxLifecycleStateActive),
		}, nil
	}
}

func ambiguousNotFoundDelete(errorToReturn error) func(context.Context, lockboxsdk.DeleteLockboxRequest) (lockboxsdk.DeleteLockboxResponse, error) {
	return func(_ context.Context, _ lockboxsdk.DeleteLockboxRequest) (lockboxsdk.DeleteLockboxResponse, error) {
		if errorToReturn != nil {
			return lockboxsdk.DeleteLockboxResponse{}, errorToReturn
		}
		return lockboxsdk.DeleteLockboxResponse{}, nil
	}
}

type fakeLockboxOCIClient struct {
	create func(context.Context, lockboxsdk.CreateLockboxRequest) (lockboxsdk.CreateLockboxResponse, error)
	get    func(context.Context, lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error)
	list   func(context.Context, lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error)
	update func(context.Context, lockboxsdk.UpdateLockboxRequest) (lockboxsdk.UpdateLockboxResponse, error)
	delete func(context.Context, lockboxsdk.DeleteLockboxRequest) (lockboxsdk.DeleteLockboxResponse, error)

	createRequests []lockboxsdk.CreateLockboxRequest
	getRequests    []lockboxsdk.GetLockboxRequest
	listRequests   []lockboxsdk.ListLockboxesRequest
	updateRequests []lockboxsdk.UpdateLockboxRequest
	deleteRequests []lockboxsdk.DeleteLockboxRequest
}

func (c *fakeLockboxOCIClient) CreateLockbox(
	ctx context.Context,
	request lockboxsdk.CreateLockboxRequest,
) (lockboxsdk.CreateLockboxResponse, error) {
	c.createRequests = append(c.createRequests, request)
	if c.create == nil {
		return lockboxsdk.CreateLockboxResponse{}, nil
	}
	return c.create(ctx, request)
}

func (c *fakeLockboxOCIClient) GetLockbox(
	ctx context.Context,
	request lockboxsdk.GetLockboxRequest,
) (lockboxsdk.GetLockboxResponse, error) {
	c.getRequests = append(c.getRequests, request)
	if c.get == nil {
		return lockboxsdk.GetLockboxResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "lockbox not found")
	}
	return c.get(ctx, request)
}

func (c *fakeLockboxOCIClient) ListLockboxes(
	ctx context.Context,
	request lockboxsdk.ListLockboxesRequest,
) (lockboxsdk.ListLockboxesResponse, error) {
	c.listRequests = append(c.listRequests, request)
	if c.list == nil {
		return lockboxsdk.ListLockboxesResponse{}, nil
	}
	return c.list(ctx, request)
}

func (c *fakeLockboxOCIClient) UpdateLockbox(
	ctx context.Context,
	request lockboxsdk.UpdateLockboxRequest,
) (lockboxsdk.UpdateLockboxResponse, error) {
	c.updateRequests = append(c.updateRequests, request)
	if c.update == nil {
		return lockboxsdk.UpdateLockboxResponse{}, nil
	}
	return c.update(ctx, request)
}

func (c *fakeLockboxOCIClient) DeleteLockbox(
	ctx context.Context,
	request lockboxsdk.DeleteLockboxRequest,
) (lockboxsdk.DeleteLockboxResponse, error) {
	c.deleteRequests = append(c.deleteRequests, request)
	if c.delete == nil {
		return lockboxsdk.DeleteLockboxResponse{}, nil
	}
	return c.delete(ctx, request)
}

func baseLockboxResource() *lockboxv1beta1.Lockbox {
	return &lockboxv1beta1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "lockbox",
		},
		Spec: lockboxv1beta1.LockboxSpec{
			ResourceId:    "ocid1.instance.oc1..target",
			CompartmentId: "ocid1.compartment.oc1..target",
			AccessContextAttributes: lockboxv1beta1.LockboxAccessContextAttributes{
				Items: []lockboxv1beta1.LockboxAccessContextAttributesItem{
					{
						Name:         "ticket",
						Description:  "Support ticket",
						DefaultValue: "sr-1",
						Values:       []string{"sr-1", "sr-2"},
					},
				},
			},
			DisplayName:          "test-lockbox",
			LockboxPartner:       string(lockboxsdk.LockboxPartnerFaaas),
			PartnerId:            "ocid1.partner.oc1..target",
			PartnerCompartmentId: "ocid1.compartment.oc1..partner",
			ApprovalTemplateId:   "ocid1.approvaltemplate.oc1..old",
			MaxAccessDuration:    "PT2H",
			FreeformTags:         map[string]string{"env": "test"},
			DefinedTags:          map[string]shared.MapValue{"ns": {"key": "value"}},
		},
	}
}

func sdkLockbox(
	id string,
	spec lockboxv1beta1.LockboxSpec,
	state lockboxsdk.LockboxLifecycleStateEnum,
) lockboxsdk.Lockbox {
	return lockboxsdk.Lockbox{
		Id:                      common.String(id),
		DisplayName:             common.String(spec.DisplayName),
		CompartmentId:           common.String(spec.CompartmentId),
		ResourceId:              common.String(spec.ResourceId),
		LifecycleState:          state,
		FreeformTags:            mapsClone(spec.FreeformTags),
		DefinedTags:             sdkDefinedTags(spec.DefinedTags),
		PartnerId:               common.String(spec.PartnerId),
		PartnerCompartmentId:    common.String(spec.PartnerCompartmentId),
		LockboxPartner:          lockboxsdk.LockboxPartnerEnum(spec.LockboxPartner),
		AccessContextAttributes: lockboxAccessContextAttributesFromSpec(spec.AccessContextAttributes),
		ApprovalTemplateId:      common.String(spec.ApprovalTemplateId),
		MaxAccessDuration:       common.String(spec.MaxAccessDuration),
	}
}

func sdkLockboxSummary(
	id string,
	spec lockboxv1beta1.LockboxSpec,
	state lockboxsdk.LockboxLifecycleStateEnum,
) lockboxsdk.LockboxSummary {
	return lockboxsdk.LockboxSummary{
		Id:                   common.String(id),
		DisplayName:          common.String(spec.DisplayName),
		CompartmentId:        common.String(spec.CompartmentId),
		ResourceId:           common.String(spec.ResourceId),
		LifecycleState:       state,
		FreeformTags:         mapsClone(spec.FreeformTags),
		DefinedTags:          sdkDefinedTags(spec.DefinedTags),
		PartnerId:            common.String(spec.PartnerId),
		PartnerCompartmentId: common.String(spec.PartnerCompartmentId),
		LockboxPartner:       lockboxsdk.LockboxPartnerEnum(spec.LockboxPartner),
		ApprovalTemplateId:   common.String(spec.ApprovalTemplateId),
		MaxAccessDuration:    common.String(spec.MaxAccessDuration),
	}
}

func sdkDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func mapsClone[K comparable, V any](input map[K]V) map[K]V {
	if input == nil {
		return nil
	}
	output := make(map[K]V, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
