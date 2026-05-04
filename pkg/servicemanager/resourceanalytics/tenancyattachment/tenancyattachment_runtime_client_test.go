/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package tenancyattachment

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeTenancyAttachmentOCIClient struct {
	createTenancyAttachmentFn func(context.Context, resourceanalyticssdk.CreateTenancyAttachmentRequest) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error)
	getTenancyAttachmentFn    func(context.Context, resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error)
	listTenancyAttachmentsFn  func(context.Context, resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error)
	updateTenancyAttachmentFn func(context.Context, resourceanalyticssdk.UpdateTenancyAttachmentRequest) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error)
	deleteTenancyAttachmentFn func(context.Context, resourceanalyticssdk.DeleteTenancyAttachmentRequest) (resourceanalyticssdk.DeleteTenancyAttachmentResponse, error)
}

func (f *fakeTenancyAttachmentOCIClient) CreateTenancyAttachment(
	ctx context.Context,
	req resourceanalyticssdk.CreateTenancyAttachmentRequest,
) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error) {
	if f.createTenancyAttachmentFn != nil {
		return f.createTenancyAttachmentFn(ctx, req)
	}
	return resourceanalyticssdk.CreateTenancyAttachmentResponse{}, nil
}

func (f *fakeTenancyAttachmentOCIClient) GetTenancyAttachment(
	ctx context.Context,
	req resourceanalyticssdk.GetTenancyAttachmentRequest,
) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
	if f.getTenancyAttachmentFn != nil {
		return f.getTenancyAttachmentFn(ctx, req)
	}
	return resourceanalyticssdk.GetTenancyAttachmentResponse{}, nil
}

func (f *fakeTenancyAttachmentOCIClient) ListTenancyAttachments(
	ctx context.Context,
	req resourceanalyticssdk.ListTenancyAttachmentsRequest,
) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error) {
	if f.listTenancyAttachmentsFn != nil {
		return f.listTenancyAttachmentsFn(ctx, req)
	}
	return resourceanalyticssdk.ListTenancyAttachmentsResponse{}, nil
}

func (f *fakeTenancyAttachmentOCIClient) UpdateTenancyAttachment(
	ctx context.Context,
	req resourceanalyticssdk.UpdateTenancyAttachmentRequest,
) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error) {
	if f.updateTenancyAttachmentFn != nil {
		return f.updateTenancyAttachmentFn(ctx, req)
	}
	return resourceanalyticssdk.UpdateTenancyAttachmentResponse{}, nil
}

func (f *fakeTenancyAttachmentOCIClient) DeleteTenancyAttachment(
	ctx context.Context,
	req resourceanalyticssdk.DeleteTenancyAttachmentRequest,
) (resourceanalyticssdk.DeleteTenancyAttachmentResponse, error) {
	if f.deleteTenancyAttachmentFn != nil {
		return f.deleteTenancyAttachmentFn(ctx, req)
	}
	return resourceanalyticssdk.DeleteTenancyAttachmentResponse{}, nil
}

func testTenancyAttachmentClient(fake *fakeTenancyAttachmentOCIClient) TenancyAttachmentServiceClient {
	return newTenancyAttachmentServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func newTenancyAttachmentTestResource() *resourceanalyticsv1beta1.TenancyAttachment {
	return &resourceanalyticsv1beta1.TenancyAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenancy-attachment",
			Namespace: "default",
		},
		Spec: resourceanalyticsv1beta1.TenancyAttachmentSpec{
			ResourceAnalyticsInstanceId: "ocid1.resourceanalyticsinstance.oc1..instance",
			TenancyId:                   "ocid1.tenancy.oc1..customer",
			Description:                 "desired description",
		},
	}
}

func newExistingTenancyAttachmentTestResource(id string) *resourceanalyticsv1beta1.TenancyAttachment {
	resource := newTenancyAttachmentTestResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.ResourceAnalyticsInstanceId = resource.Spec.ResourceAnalyticsInstanceId
	resource.Status.TenancyId = resource.Spec.TenancyId
	resource.Status.Description = resource.Spec.Description
	resource.Status.LifecycleState = string(resourceanalyticssdk.TenancyAttachmentLifecycleStateActive)
	return resource
}

func newSDKTenancyAttachment(
	id string,
	resourceAnalyticsInstanceID string,
	tenancyID string,
	description string,
	state resourceanalyticssdk.TenancyAttachmentLifecycleStateEnum,
) resourceanalyticssdk.TenancyAttachment {
	attachment := resourceanalyticssdk.TenancyAttachment{
		Id:                          common.String(id),
		ResourceAnalyticsInstanceId: common.String(resourceAnalyticsInstanceID),
		TenancyId:                   common.String(tenancyID),
		IsReportingTenancy:          common.Bool(false),
		LifecycleState:              state,
	}
	if description != "" {
		attachment.Description = common.String(description)
	}
	return attachment
}

func newSDKTenancyAttachmentSummary(
	id string,
	resourceAnalyticsInstanceID string,
	tenancyID string,
	description string,
	state resourceanalyticssdk.TenancyAttachmentLifecycleStateEnum,
) resourceanalyticssdk.TenancyAttachmentSummary {
	summary := resourceanalyticssdk.TenancyAttachmentSummary{
		Id:                          common.String(id),
		ResourceAnalyticsInstanceId: common.String(resourceAnalyticsInstanceID),
		TenancyId:                   common.String(tenancyID),
		IsReportingTenancy:          common.Bool(false),
		LifecycleState:              state,
	}
	if description != "" {
		summary.Description = common.String(description)
	}
	return summary
}

type tenancyAttachmentCreateRefreshRecorder struct {
	createRequest resourceanalyticssdk.CreateTenancyAttachmentRequest
	listCalls     int
	getCalls      int
}

func TestTenancyAttachmentCreateOrUpdateCreatesAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.tenancyattachment.oc1..created"

	resource := newTenancyAttachmentTestResource()
	recorder := &tenancyAttachmentCreateRefreshRecorder{}
	client := newTenancyAttachmentCreateRefreshClient(t, createdID, recorder)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireTenancyAttachmentCreateRefreshResult(t, response, err, resource, recorder, createdID)
}

func newTenancyAttachmentCreateRefreshClient(
	t *testing.T,
	createdID string,
	recorder *tenancyAttachmentCreateRefreshRecorder,
) TenancyAttachmentServiceClient {
	t.Helper()

	return testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		listTenancyAttachmentsFn: func(_ context.Context, req resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error) {
			recorder.listCalls++
			requireStringPtr(t, req.ResourceAnalyticsInstanceId, "ocid1.resourceanalyticsinstance.oc1..instance", "list resourceAnalyticsInstanceId")
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil before create", req.Id)
			}
			return resourceanalyticssdk.ListTenancyAttachmentsResponse{}, nil
		},
		createTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.CreateTenancyAttachmentRequest) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error) {
			recorder.createRequest = req
			return resourceanalyticssdk.CreateTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					createdID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"ocid1.tenancy.oc1..customer",
					"desired description",
					resourceanalyticssdk.TenancyAttachmentLifecycleStateCreating,
				),
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		},
		getTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			recorder.getCalls++
			requireStringPtr(t, req.TenancyAttachmentId, createdID, "get tenancyAttachmentId")
			return resourceanalyticssdk.GetTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					createdID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"ocid1.tenancy.oc1..customer",
					"desired description",
					resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
				),
			}, nil
		},
	})
}

func requireTenancyAttachmentCreateRefreshResult(
	t *testing.T,
	response servicemanager.OSOKResponse,
	err error,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
	recorder *tenancyAttachmentCreateRefreshRecorder,
	createdID string,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE readback")
	}
	if recorder.listCalls != 1 {
		t.Fatalf("ListTenancyAttachments() calls = %d, want 1", recorder.listCalls)
	}
	if recorder.getCalls != 1 {
		t.Fatalf("GetTenancyAttachment() calls = %d, want 1", recorder.getCalls)
	}
	requireStringPtr(t, recorder.createRequest.ResourceAnalyticsInstanceId, resource.Spec.ResourceAnalyticsInstanceId, "create resourceAnalyticsInstanceId")
	requireStringPtr(t, recorder.createRequest.TenancyId, resource.Spec.TenancyId, "create tenancyId")
	requireStringPtr(t, recorder.createRequest.Description, resource.Spec.Description, "create description")
	if recorder.createRequest.OpcRetryToken == nil || strings.TrimSpace(*recorder.createRequest.OpcRetryToken) == "" {
		t.Fatal("create opcRetryToken is empty, want deterministic retry token")
	}
	requireTenancyAttachmentStatus(t, resource, createdID, resource.Spec.Description, "ACTIVE")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after ACTIVE readback", resource.Status.OsokStatus.Async.Current)
	}
}

func TestTenancyAttachmentCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.tenancyattachment.oc1..existing"

	createCalls := 0
	getCalls := 0
	var pages []string

	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		listTenancyAttachmentsFn: func(_ context.Context, req resourceanalyticssdk.ListTenancyAttachmentsRequest) (resourceanalyticssdk.ListTenancyAttachmentsResponse, error) {
			pages = append(pages, stringValue(req.Page))
			requireStringPtr(t, req.ResourceAnalyticsInstanceId, "ocid1.resourceanalyticsinstance.oc1..instance", "list resourceAnalyticsInstanceId")
			switch stringValue(req.Page) {
			case "":
				return resourceanalyticssdk.ListTenancyAttachmentsResponse{
					TenancyAttachmentCollection: resourceanalyticssdk.TenancyAttachmentCollection{
						Items: []resourceanalyticssdk.TenancyAttachmentSummary{
							newSDKTenancyAttachmentSummary(
								"ocid1.tenancyattachment.oc1..other",
								"ocid1.resourceanalyticsinstance.oc1..instance",
								"ocid1.tenancy.oc1..other",
								"other",
								resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return resourceanalyticssdk.ListTenancyAttachmentsResponse{
					TenancyAttachmentCollection: resourceanalyticssdk.TenancyAttachmentCollection{
						Items: []resourceanalyticssdk.TenancyAttachmentSummary{
							newSDKTenancyAttachmentSummary(
								existingID,
								"ocid1.resourceanalyticsinstance.oc1..instance",
								"ocid1.tenancy.oc1..customer",
								"desired description",
								resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
							),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page = %q", stringValue(req.Page))
				return resourceanalyticssdk.ListTenancyAttachmentsResponse{}, nil
			}
		},
		createTenancyAttachmentFn: func(context.Context, resourceanalyticssdk.CreateTenancyAttachmentRequest) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error) {
			createCalls++
			return resourceanalyticssdk.CreateTenancyAttachmentResponse{}, nil
		},
		getTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, req.TenancyAttachmentId, existingID, "get tenancyAttachmentId")
			return resourceanalyticssdk.GetTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"ocid1.tenancy.oc1..customer",
					"desired description",
					resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := newTenancyAttachmentTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalls != 0 {
		t.Fatalf("CreateTenancyAttachment() calls = %d, want 0", createCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetTenancyAttachment() calls = %d, want 1 after bind", getCalls)
	}
	if got, want := strings.Join(pages, ","), ",page-2"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	requireTenancyAttachmentStatus(t, resource, existingID, "desired description", "ACTIVE")
}

func TestTenancyAttachmentCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.tenancyattachment.oc1..existing"

	updateCalls := 0
	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		getTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			requireStringPtr(t, req.TenancyAttachmentId, existingID, "get tenancyAttachmentId")
			return resourceanalyticssdk.GetTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"ocid1.tenancy.oc1..customer",
					"desired description",
					resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
				),
			}, nil
		},
		updateTenancyAttachmentFn: func(context.Context, resourceanalyticssdk.UpdateTenancyAttachmentRequest) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error) {
			updateCalls++
			return resourceanalyticssdk.UpdateTenancyAttachmentResponse{}, nil
		},
	})

	resource := newExistingTenancyAttachmentTestResource(existingID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateTenancyAttachment() calls = %d, want 0", updateCalls)
	}
	requireTenancyAttachmentStatus(t, resource, existingID, "desired description", "ACTIVE")
}

func TestTenancyAttachmentCreateOrUpdateUpdatesMutableDescription(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.tenancyattachment.oc1..existing"

	getCalls := 0
	var updateRequest resourceanalyticssdk.UpdateTenancyAttachmentRequest

	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		getTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, req.TenancyAttachmentId, existingID, "get tenancyAttachmentId")
			description := "old description"
			if getCalls > 1 {
				description = "updated description"
			}
			return resourceanalyticssdk.GetTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"ocid1.tenancy.oc1..customer",
					description,
					resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
				),
			}, nil
		},
		updateTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.UpdateTenancyAttachmentRequest) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error) {
			updateRequest = req
			return resourceanalyticssdk.UpdateTenancyAttachmentResponse{
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := newExistingTenancyAttachmentTestResource(existingID)
	resource.Spec.Description = "updated description"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if getCalls != 2 {
		t.Fatalf("GetTenancyAttachment() calls = %d, want 2 for update readback", getCalls)
	}
	requireStringPtr(t, updateRequest.TenancyAttachmentId, existingID, "update tenancyAttachmentId")
	requireStringPtr(t, updateRequest.Description, "updated description", "update description")
	requireTenancyAttachmentStatus(t, resource, existingID, "updated description", "ACTIVE")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestTenancyAttachmentRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.tenancyattachment.oc1..existing"

	updateCalls := 0
	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		getTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			requireStringPtr(t, req.TenancyAttachmentId, existingID, "get tenancyAttachmentId")
			return resourceanalyticssdk.GetTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..original",
					"ocid1.tenancy.oc1..customer",
					"desired description",
					resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
				),
			}, nil
		},
		updateTenancyAttachmentFn: func(context.Context, resourceanalyticssdk.UpdateTenancyAttachmentRequest) (resourceanalyticssdk.UpdateTenancyAttachmentResponse, error) {
			updateCalls++
			return resourceanalyticssdk.UpdateTenancyAttachmentResponse{}, nil
		},
	})

	resource := newExistingTenancyAttachmentTestResource(existingID)
	resource.Spec.ResourceAnalyticsInstanceId = "ocid1.resourceanalyticsinstance.oc1..changed"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for create-only drift")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateTenancyAttachment() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "resourceAnalyticsInstanceId") {
		t.Fatalf("CreateOrUpdate() error = %v, want resourceAnalyticsInstanceId drift context", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestTenancyAttachmentDeleteRetainsFinalizerWhileLifecycleDeleting(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.tenancyattachment.oc1..existing"

	getCalls := 0
	var deleteRequest resourceanalyticssdk.DeleteTenancyAttachmentRequest
	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		getTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, req.TenancyAttachmentId, existingID, "get tenancyAttachmentId")
			state := resourceanalyticssdk.TenancyAttachmentLifecycleStateActive
			if getCalls == 3 {
				state = resourceanalyticssdk.TenancyAttachmentLifecycleStateDeleting
			}
			return resourceanalyticssdk.GetTenancyAttachmentResponse{
				TenancyAttachment: newSDKTenancyAttachment(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"ocid1.tenancy.oc1..customer",
					"desired description",
					state,
				),
			}, nil
		},
		deleteTenancyAttachmentFn: func(_ context.Context, req resourceanalyticssdk.DeleteTenancyAttachmentRequest) (resourceanalyticssdk.DeleteTenancyAttachmentResponse, error) {
			deleteRequest = req
			return resourceanalyticssdk.DeleteTenancyAttachmentResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	resource := newExistingTenancyAttachmentTestResource(existingID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI lifecycle is DELETING")
	}
	if getCalls != 3 {
		t.Fatalf("GetTenancyAttachment() calls = %d, want 3", getCalls)
	}
	requireStringPtr(t, deleteRequest.TenancyAttachmentId, existingID, "delete tenancyAttachmentId")
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete phase", current)
	}
}

func TestTenancyAttachmentDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.tenancyattachment.oc1..existing"

	deleteCalls := 0
	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		getTenancyAttachmentFn: func(context.Context, resourceanalyticssdk.GetTenancyAttachmentRequest) (resourceanalyticssdk.GetTenancyAttachmentResponse, error) {
			return resourceanalyticssdk.GetTenancyAttachmentResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
		deleteTenancyAttachmentFn: func(context.Context, resourceanalyticssdk.DeleteTenancyAttachmentRequest) (resourceanalyticssdk.DeleteTenancyAttachmentResponse, error) {
			deleteCalls++
			return resourceanalyticssdk.DeleteTenancyAttachmentResponse{}, nil
		},
	})

	resource := newExistingTenancyAttachmentTestResource(existingID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteTenancyAttachment() calls = %d, want 0", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 context", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestTenancyAttachmentCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := testTenancyAttachmentClient(&fakeTenancyAttachmentOCIClient{
		createTenancyAttachmentFn: func(context.Context, resourceanalyticssdk.CreateTenancyAttachmentRequest) (resourceanalyticssdk.CreateTenancyAttachmentResponse, error) {
			return resourceanalyticssdk.CreateTenancyAttachmentResponse{}, errortest.NewServiceError(
				500,
				"InternalError",
				"create failed",
			)
		},
	})

	resource := newTenancyAttachmentTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestApplyTenancyAttachmentRuntimeHooksInstallsReviewedBehavior(t *testing.T) {
	t.Parallel()

	hooks := newTenancyAttachmentDefaultRuntimeHooks(resourceanalyticssdk.TenancyAttachmentClient{})
	applyTenancyAttachmentRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("hooks.Semantics.FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handler")
	}

	resource := newTenancyAttachmentTestResource()
	resource.Spec.Description = "updated"
	body, updateNeeded, err := hooks.BuildUpdateBody(
		context.Background(),
		resource,
		resource.Namespace,
		newSDKTenancyAttachment(
			"ocid1.tenancyattachment.oc1..existing",
			resource.Spec.ResourceAnalyticsInstanceId,
			resource.Spec.TenancyId,
			"old",
			resourceanalyticssdk.TenancyAttachmentLifecycleStateActive,
		),
	)
	if err != nil {
		t.Fatalf("hooks.BuildUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("hooks.BuildUpdateBody() updateNeeded = false, want true for description drift")
	}
	details, ok := body.(resourceanalyticssdk.UpdateTenancyAttachmentDetails)
	if !ok {
		t.Fatalf("hooks.BuildUpdateBody() body type = %T, want resourceanalytics.UpdateTenancyAttachmentDetails", body)
	}
	requireStringPtr(t, details.Description, "updated", "update body description")
}

func requireTenancyAttachmentStatus(
	t *testing.T,
	resource *resourceanalyticsv1beta1.TenancyAttachment,
	wantID string,
	wantDescription string,
	wantLifecycleState string,
) {
	t.Helper()

	if got := resource.Status.Id; got != wantID {
		t.Fatalf("status.id = %q, want %q", got, wantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if got := resource.Status.Description; got != wantDescription {
		t.Fatalf("status.description = %q, want %q", got, wantDescription)
	}
	if got := resource.Status.LifecycleState; got != wantLifecycleState {
		t.Fatalf("status.lifecycleState = %q, want %q", got, wantLifecycleState)
	}
}

func requireStringPtr(t *testing.T, got *string, want string, name string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
