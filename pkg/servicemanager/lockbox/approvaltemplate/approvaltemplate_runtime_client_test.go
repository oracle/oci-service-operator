/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package approvaltemplate

import (
	"context"
	"reflect"
	"strings"
	"testing"

	lockboxsdk "github.com/oracle/oci-go-sdk/v65/lockbox"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeApprovalTemplateOCIClient struct {
	create func(context.Context, lockboxsdk.CreateApprovalTemplateRequest) (lockboxsdk.CreateApprovalTemplateResponse, error)
	get    func(context.Context, lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error)
	list   func(context.Context, lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error)
	update func(context.Context, lockboxsdk.UpdateApprovalTemplateRequest) (lockboxsdk.UpdateApprovalTemplateResponse, error)
	delete func(context.Context, lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error)

	createRequests []lockboxsdk.CreateApprovalTemplateRequest
	getRequests    []lockboxsdk.GetApprovalTemplateRequest
	listRequests   []lockboxsdk.ListApprovalTemplatesRequest
	updateRequests []lockboxsdk.UpdateApprovalTemplateRequest
	deleteRequests []lockboxsdk.DeleteApprovalTemplateRequest
}

func (f *fakeApprovalTemplateOCIClient) CreateApprovalTemplate(
	ctx context.Context,
	request lockboxsdk.CreateApprovalTemplateRequest,
) (lockboxsdk.CreateApprovalTemplateResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return lockboxsdk.CreateApprovalTemplateResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeApprovalTemplateOCIClient) GetApprovalTemplate(
	ctx context.Context,
	request lockboxsdk.GetApprovalTemplateRequest,
) (lockboxsdk.GetApprovalTemplateResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return lockboxsdk.GetApprovalTemplateResponse{}, nil
	}
	return f.get(ctx, request)
}

func (f *fakeApprovalTemplateOCIClient) ListApprovalTemplates(
	ctx context.Context,
	request lockboxsdk.ListApprovalTemplatesRequest,
) (lockboxsdk.ListApprovalTemplatesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return lockboxsdk.ListApprovalTemplatesResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeApprovalTemplateOCIClient) UpdateApprovalTemplate(
	ctx context.Context,
	request lockboxsdk.UpdateApprovalTemplateRequest,
) (lockboxsdk.UpdateApprovalTemplateResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return lockboxsdk.UpdateApprovalTemplateResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeApprovalTemplateOCIClient) DeleteApprovalTemplate(
	ctx context.Context,
	request lockboxsdk.DeleteApprovalTemplateRequest,
) (lockboxsdk.DeleteApprovalTemplateResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return lockboxsdk.DeleteApprovalTemplateResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestApprovalTemplateCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newTestApprovalTemplate()
	client := &fakeApprovalTemplateOCIClient{}
	client.list = func(_ context.Context, request lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error) {
		assertStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		assertStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
		return lockboxsdk.ListApprovalTemplatesResponse{}, nil
	}
	client.create = func(_ context.Context, request lockboxsdk.CreateApprovalTemplateRequest) (lockboxsdk.CreateApprovalTemplateResponse, error) {
		assertApprovalTemplateCreateRequest(t, request, resource.Spec)
		return lockboxsdk.CreateApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..created", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateCreating),
			OpcRequestId:     stringPointer("opc-create"),
		}, nil
	}
	client.get = func(_ context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		assertStringPtr(t, "get approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..created")
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..created", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateActive),
			OpcRequestId:     stringPointer("opc-get"),
		}, nil
	}

	response, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	assertApprovalTemplateCreatedStatus(t, resource)
	assertApprovalTemplateCallCount(t, "CreateApprovalTemplate()", len(client.createRequests), 1)
}

func TestApprovalTemplateCreateOrUpdateBindsExistingFromSecondListPage(t *testing.T) {
	resource := newTestApprovalTemplate()
	client := &fakeApprovalTemplateOCIClient{}
	client.list = func(_ context.Context, request lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error) {
		if request.Page == nil {
			return lockboxsdk.ListApprovalTemplatesResponse{
				ApprovalTemplateCollection: lockboxsdk.ApprovalTemplateCollection{
					Items: []lockboxsdk.ApprovalTemplateSummary{
						approvalTemplateSummary("ocid1.approvaltemplate.oc1..other", resource.Spec.CompartmentId, "other", lockboxsdk.ApprovalTemplateLifecycleStateActive),
					},
				},
				OpcNextPage: stringPointer("page-2"),
			}, nil
		}
		assertStringPtr(t, "second page", request.Page, "page-2")
		return lockboxsdk.ListApprovalTemplatesResponse{
			ApprovalTemplateCollection: lockboxsdk.ApprovalTemplateCollection{
				Items: []lockboxsdk.ApprovalTemplateSummary{
					approvalTemplateSummary("ocid1.approvaltemplate.oc1..existing", resource.Spec.CompartmentId, resource.Spec.DisplayName, lockboxsdk.ApprovalTemplateLifecycleStateActive),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		assertStringPtr(t, "get approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..existing")
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateActive),
		}, nil
	}

	response, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful bind without requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateApprovalTemplate() calls = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListApprovalTemplates() calls = %d, want 2 pages", len(client.listRequests))
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.approvaltemplate.oc1..existing"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestApprovalTemplateCreateOrUpdateNoopsWithoutMutableDrift(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	client := &fakeApprovalTemplateOCIClient{}
	client.get = func(_ context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		assertStringPtr(t, "get approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..existing")
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateActive),
		}, nil
	}

	response, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateApprovalTemplate() calls = %d, want 0", len(client.updateRequests))
	}
}

func TestApprovalTemplateCreateOrUpdateSendsMutableUpdate(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	resource.Spec.DisplayName = "template-new"
	resource.Spec.AutoApprovalState = string(lockboxsdk.LockboxAutoApprovalStateEnabled)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "platform"}}
	resource.Spec.ApproverLevels.Level2 = lockboxv1beta1.ApprovalTemplateApproverLevelsLevel2{
		ApproverType: string(lockboxsdk.ApproverTypeUser),
		ApproverId:   "ocid1.user.oc1..approver2",
	}

	currentSpec := resource.Spec
	currentSpec.DisplayName = "template-old"
	currentSpec.AutoApprovalState = string(lockboxsdk.LockboxAutoApprovalStateDisabled)
	currentSpec.FreeformTags = map[string]string{"env": "dev"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "old"}}
	currentSpec.ApproverLevels.Level2 = lockboxv1beta1.ApprovalTemplateApproverLevelsLevel2{}

	client := &fakeApprovalTemplateOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		assertStringPtr(t, "get approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..existing")
		getCalls++
		if getCalls == 2 {
			return lockboxsdk.GetApprovalTemplateResponse{
				ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateUpdating),
			}, nil
		}
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", currentSpec, lockboxsdk.ApprovalTemplateLifecycleStateActive),
		}, nil
	}
	client.update = func(_ context.Context, request lockboxsdk.UpdateApprovalTemplateRequest) (lockboxsdk.UpdateApprovalTemplateResponse, error) {
		assertApprovalTemplateMutableUpdateRequest(t, request, resource.Spec)
		return lockboxsdk.UpdateApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateUpdating),
			OpcRequestId:     stringPointer("opc-update"),
		}, nil
	}

	response, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful updating requeue", response)
	}
	assertApprovalTemplateUpdatedStatus(t, resource)
	assertApprovalTemplateCallCount(t, "UpdateApprovalTemplate()", len(client.updateRequests), 1)
}

func TestApprovalTemplateCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	currentSpec := resource.Spec
	currentSpec.CompartmentId = "ocid1.compartment.oc1..old"
	client := &fakeApprovalTemplateOCIClient{}
	client.get = func(_ context.Context, _ lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", currentSpec, lockboxsdk.ApprovalTemplateLifecycleStateActive),
		}, nil
	}

	_, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only drift rejection", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateApprovalTemplate() calls = %d, want 0 after drift rejection", len(client.updateRequests))
	}
}

func TestApprovalTemplateCreateOrUpdateRequeuesWhileCreating(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..creating")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..creating"
	client := &fakeApprovalTemplateOCIClient{}
	client.get = func(_ context.Context, _ lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..creating", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateCreating),
		}, nil
	}

	response, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want creating requeue", response)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle create tracker")
	}
	if got, want := resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseCreate; got != want {
		t.Fatalf("status.status.async.current.phase = %q, want %q", got, want)
	}
}

func TestApprovalTemplateDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	client := &fakeApprovalTemplateOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		assertStringPtr(t, "get approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..existing")
		getCalls++
		state := lockboxsdk.ApprovalTemplateLifecycleStateActive
		if getCalls == 3 {
			state = lockboxsdk.ApprovalTemplateLifecycleStateDeleting
		}
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, state),
		}, nil
	}
	client.delete = func(_ context.Context, request lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error) {
		assertStringPtr(t, "delete approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..existing")
		return lockboxsdk.DeleteApprovalTemplateResponse{OpcRequestId: stringPointer("opc-delete")}, nil
	}

	deleted, err := newTestApprovalTemplateClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending delete confirmation")
	}
	if got, want := resource.Status.LifecycleState, string(lockboxsdk.ApprovalTemplateLifecycleStateDeleting); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want delete tracker")
	}
	if got, want := resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete; got != want {
		t.Fatalf("status.status.async.current.phase = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestApprovalTemplateDeleteConfirmsDeletedLifecycle(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	client := &fakeApprovalTemplateOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, _ lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		getCalls++
		state := lockboxsdk.ApprovalTemplateLifecycleStateActive
		if getCalls == 3 {
			state = lockboxsdk.ApprovalTemplateLifecycleStateDeleted
		}
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, state),
		}, nil
	}
	client.delete = func(_ context.Context, _ lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error) {
		return lockboxsdk.DeleteApprovalTemplateResponse{OpcRequestId: stringPointer("opc-delete")}, nil
	}

	deleted, err := newTestApprovalTemplateClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete completion timestamp")
	}
}

func TestApprovalTemplateDeleteRejectsAuthShapedPreRead(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	client := &fakeApprovalTemplateOCIClient{}
	client.get = func(_ context.Context, _ lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		return lockboxsdk.GetApprovalTemplateResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newTestApprovalTemplateClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped pre-read failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-read")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteApprovalTemplate() calls = %d, want 0 after auth-shaped pre-read", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped pre-read", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestApprovalTemplateDeleteRejectsAuthShapedDeleteCall(t *testing.T) {
	resource := newTestApprovalTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.approvaltemplate.oc1..existing")
	resource.Status.Id = "ocid1.approvaltemplate.oc1..existing"
	client := &fakeApprovalTemplateOCIClient{}
	client.get = func(_ context.Context, _ lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
		return lockboxsdk.GetApprovalTemplateResponse{
			ApprovalTemplate: approvalTemplateFromSpec("ocid1.approvaltemplate.oc1..existing", resource.Spec, lockboxsdk.ApprovalTemplateLifecycleStateActive),
		}, nil
	}
	client.delete = func(_ context.Context, _ lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error) {
		return lockboxsdk.DeleteApprovalTemplateResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newTestApprovalTemplateClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped delete failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete 404")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped delete 404", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestApprovalTemplateCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := newTestApprovalTemplate()
	client := &fakeApprovalTemplateOCIClient{}
	client.list = func(_ context.Context, _ lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error) {
		return lockboxsdk.ListApprovalTemplatesResponse{}, nil
	}
	client.create = func(_ context.Context, _ lockboxsdk.CreateApprovalTemplateRequest) (lockboxsdk.CreateApprovalTemplateResponse, error) {
		return lockboxsdk.CreateApprovalTemplateResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
	}

	response, err := newTestApprovalTemplateClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "create failed") {
		t.Fatalf("status.status.message = %q, want OCI error message", resource.Status.OsokStatus.Message)
	}
}

func assertApprovalTemplateCreateRequest(
	t *testing.T,
	request lockboxsdk.CreateApprovalTemplateRequest,
	spec lockboxv1beta1.ApprovalTemplateSpec,
) {
	t.Helper()
	assertStringPtr(t, "create compartmentId", request.CompartmentId, spec.CompartmentId)
	assertStringPtr(t, "create displayName", request.DisplayName, spec.DisplayName)
	levels := requireApprovalTemplateApproverLevels(t, "create approverLevels", request.ApproverLevels)
	assertApprovalTemplateApproverID(t, "create approverLevels.level1", levels.Level1, "ocid1.group.oc1..approver")
}

func assertApprovalTemplateMutableUpdateRequest(
	t *testing.T,
	request lockboxsdk.UpdateApprovalTemplateRequest,
	spec lockboxv1beta1.ApprovalTemplateSpec,
) {
	t.Helper()
	assertStringPtr(t, "update approvalTemplateId", request.ApprovalTemplateId, "ocid1.approvaltemplate.oc1..existing")
	assertStringPtr(t, "update displayName", request.DisplayName, "template-new")
	if request.AutoApprovalState != lockboxsdk.LockboxAutoApprovalStateEnabled {
		t.Fatalf("update autoApprovalState = %q, want ENABLED", request.AutoApprovalState)
	}
	levels := requireApprovalTemplateApproverLevels(t, "update approverLevels", request.ApproverLevels)
	assertApprovalTemplateApproverID(t, "update approverLevels.level2", levels.Level2, "ocid1.user.oc1..approver2")
	if !reflect.DeepEqual(request.FreeformTags, spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", request.FreeformTags, spec.FreeformTags)
	}
	wantDefinedTags := map[string]map[string]interface{}{"ops": {"owner": "platform"}}
	if !reflect.DeepEqual(request.DefinedTags, wantDefinedTags) {
		t.Fatalf("update definedTags = %#v, want %#v", request.DefinedTags, wantDefinedTags)
	}
}

func requireApprovalTemplateApproverLevels(
	t *testing.T,
	name string,
	got *lockboxsdk.ApproverLevels,
) *lockboxsdk.ApproverLevels {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want desired approver levels", name)
	}
	return got
}

func assertApprovalTemplateApproverID(t *testing.T, name string, got *lockboxsdk.ApproverInfo, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want approver %q", name, want)
	}
	assertStringPtr(t, name+".approverId", got.ApproverId, want)
}

func assertApprovalTemplateCreatedStatus(t *testing.T, resource *lockboxv1beta1.ApprovalTemplate) {
	t.Helper()
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.approvaltemplate.oc1..created"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.Id, "ocid1.approvaltemplate.oc1..created"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := resource.Status.LifecycleState, string(lockboxsdk.ApprovalTemplateLifecycleStateActive); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertApprovalTemplateUpdatedStatus(t *testing.T, resource *lockboxv1beta1.ApprovalTemplate) {
	t.Helper()
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	assertApprovalTemplateAsyncPhase(t, resource, shared.OSOKAsyncPhaseUpdate)
}

func assertApprovalTemplateAsyncPhase(
	t *testing.T,
	resource *lockboxv1beta1.ApprovalTemplate,
	want shared.OSOKAsyncPhase,
) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != want {
		t.Fatalf("status.status.async.current.phase = %q, want %q", got, want)
	}
}

func assertApprovalTemplateCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func newTestApprovalTemplateClient(client *fakeApprovalTemplateOCIClient) ApprovalTemplateServiceClient {
	return newApprovalTemplateServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func newTestApprovalTemplate() *lockboxv1beta1.ApprovalTemplate {
	return &lockboxv1beta1.ApprovalTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "approval-template",
			Namespace: "default",
			UID:       types.UID("approval-template-uid"),
		},
		Spec: lockboxv1beta1.ApprovalTemplateSpec{
			CompartmentId:     "ocid1.compartment.oc1..example",
			DisplayName:       "template",
			AutoApprovalState: string(lockboxsdk.LockboxAutoApprovalStateDisabled),
			ApproverLevels: lockboxv1beta1.ApprovalTemplateApproverLevels{
				Level1: lockboxv1beta1.ApprovalTemplateApproverLevelsLevel1{
					ApproverType: string(lockboxsdk.ApproverTypeGroup),
					ApproverId:   "ocid1.group.oc1..approver",
					DomainId:     "ocid1.domain.oc1..example",
				},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags:  map[string]shared.MapValue{"ops": {"owner": "old"}},
		},
	}
}

func testRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "approval-template"}}
}

func approvalTemplateFromSpec(
	id string,
	spec lockboxv1beta1.ApprovalTemplateSpec,
	state lockboxsdk.ApprovalTemplateLifecycleStateEnum,
) lockboxsdk.ApprovalTemplate {
	approverLevels, _, err := approvalTemplateApproverLevelsFromSpec(spec.ApproverLevels)
	if err != nil {
		panic(err)
	}
	return lockboxsdk.ApprovalTemplate{
		Id:                stringPointer(id),
		DisplayName:       stringPointer(spec.DisplayName),
		CompartmentId:     stringPointer(spec.CompartmentId),
		LifecycleState:    state,
		ApproverLevels:    approverLevels,
		AutoApprovalState: lockboxsdk.LockboxAutoApprovalStateEnum(spec.AutoApprovalState),
		FreeformTags:      spec.FreeformTags,
		DefinedTags:       map[string]map[string]interface{}{"ops": {"owner": spec.DefinedTags["ops"]["owner"]}},
	}
}

func approvalTemplateSummary(
	id string,
	compartmentID string,
	displayName string,
	state lockboxsdk.ApprovalTemplateLifecycleStateEnum,
) lockboxsdk.ApprovalTemplateSummary {
	return lockboxsdk.ApprovalTemplateSummary{
		Id:             stringPointer(id),
		CompartmentId:  stringPointer(compartmentID),
		DisplayName:    stringPointer(displayName),
		LifecycleState: state,
	}
}

func assertStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}
