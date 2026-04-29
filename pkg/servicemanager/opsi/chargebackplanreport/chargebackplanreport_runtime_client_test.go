/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package chargebackplanreport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSourceID       = "ocid1.databaseinsight.oc1..source"
	testOtherSourceID  = "ocid1.databaseinsight.oc1..other"
	testReportID       = "ocid1.chargebackplanreport.oc1..report"
	testOtherReportID  = "ocid1.chargebackplanreport.oc1..other"
	testReportName     = "monthly-chargeback"
	testUpdatedName    = "monthly-chargeback-updated"
	testResourceType   = "DATABASE_INSIGHT"
	testCreateWR       = "wr-create"
	testUpdateWR       = "wr-update"
	testDeleteWR       = "wr-delete"
	testTimeStart      = "2026-01-01T00:00:00Z"
	testTimeEnd        = "2026-01-31T00:00:00Z"
	testUpdatedTimeEnd = "2026-02-01T00:00:00Z"
)

type fakeChargebackPlanReportOCIClient struct {
	createFn         func(context.Context, opsisdk.CreateChargebackPlanReportRequest) (opsisdk.CreateChargebackPlanReportResponse, error)
	getFn            func(context.Context, opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error)
	listFn           func(context.Context, opsisdk.ListChargebackPlanReportsRequest) (opsisdk.ListChargebackPlanReportsResponse, error)
	updateFn         func(context.Context, opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error)
	deleteFn         func(context.Context, opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error)
	getWorkRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeChargebackPlanReportOCIClient) CreateChargebackPlanReport(ctx context.Context, request opsisdk.CreateChargebackPlanReportRequest) (opsisdk.CreateChargebackPlanReportResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return opsisdk.CreateChargebackPlanReportResponse{}, nil
}

func (f *fakeChargebackPlanReportOCIClient) GetChargebackPlanReport(ctx context.Context, request opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return opsisdk.GetChargebackPlanReportResponse{}, nil
}

func (f *fakeChargebackPlanReportOCIClient) ListChargebackPlanReports(ctx context.Context, request opsisdk.ListChargebackPlanReportsRequest) (opsisdk.ListChargebackPlanReportsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return opsisdk.ListChargebackPlanReportsResponse{}, nil
}

func (f *fakeChargebackPlanReportOCIClient) UpdateChargebackPlanReport(ctx context.Context, request opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return opsisdk.UpdateChargebackPlanReportResponse{}, nil
}

func (f *fakeChargebackPlanReportOCIClient) DeleteChargebackPlanReport(ctx context.Context, request opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return opsisdk.DeleteChargebackPlanReportResponse{}, nil
}

func (f *fakeChargebackPlanReportOCIClient) GetWorkRequest(ctx context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return opsisdk.GetWorkRequestResponse{}, nil
}

func TestChargebackPlanReportCreateUsesSourceAnnotationsAndPollsWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newChargebackPlanReportResource()
	var createRequest opsisdk.CreateChargebackPlanReportRequest
	var getWorkRequest opsisdk.GetWorkRequestRequest
	var getRequest opsisdk.GetChargebackPlanReportRequest
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		createFn: func(_ context.Context, request opsisdk.CreateChargebackPlanReportRequest) (opsisdk.CreateChargebackPlanReportResponse, error) {
			createRequest = request
			return opsisdk.CreateChargebackPlanReportResponse{
				ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
				OpcWorkRequestId:     common.String(testCreateWR),
				OpcRequestId:         common.String("opc-create"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			getWorkRequest = request
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: newChargebackPlanReportWorkRequest(testCreateWR, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeCreated, testReportID),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
			getRequest = request
			return opsisdk.GetChargebackPlanReportResponse{
				ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, chargebackPlanReportRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	requireStringPtr(t, "CreateChargebackPlanReportRequest.Id", createRequest.Id, testSourceID)
	requireStringPtr(t, "CreateChargebackPlanReportRequest.ResourceType", createRequest.ResourceType, testResourceType)
	if createRequest.ReportName == nil || *createRequest.ReportName != testReportName {
		t.Fatalf("CreateChargebackPlanReportRequest.ReportName = %v, want %q", createRequest.ReportName, testReportName)
	}
	requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", getWorkRequest.WorkRequestId, testCreateWR)
	requireStringPtr(t, "GetChargebackPlanReportRequest.ChargebackPlanReportId", getRequest.ChargebackPlanReportId, testReportID)
	requireChargebackPlanReportStatus(t, resource, testReportID, testSourceID, testResourceType, testReportName)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
}

func TestChargebackPlanReportBindsExistingReportFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := newChargebackPlanReportResource()
	var listRequests []opsisdk.ListChargebackPlanReportsRequest
	createCalled := false
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		listFn: func(_ context.Context, request opsisdk.ListChargebackPlanReportsRequest) (opsisdk.ListChargebackPlanReportsResponse, error) {
			listRequests = append(listRequests, request)
			if request.Page == nil {
				return opsisdk.ListChargebackPlanReportsResponse{
					ChargebackPlanReportCollection: opsisdk.ChargebackPlanReportCollection{
						Items: []opsisdk.ChargebackPlanReportSummary{
							newSDKChargebackPlanReportSummary(testOtherReportID, "other-report", testSourceID, testResourceType, testTimeEnd),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return opsisdk.ListChargebackPlanReportsResponse{
				ChargebackPlanReportCollection: opsisdk.ChargebackPlanReportCollection{
					Items: []opsisdk.ChargebackPlanReportSummary{
						newSDKChargebackPlanReportSummary(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
					},
				},
			}, nil
		},
		createFn: func(context.Context, opsisdk.CreateChargebackPlanReportRequest) (opsisdk.CreateChargebackPlanReportResponse, error) {
			createCalled = true
			return opsisdk.CreateChargebackPlanReportResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, chargebackPlanReportRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalled {
		t.Fatal("CreateChargebackPlanReport() was called, want bind to existing list item")
	}
	if len(listRequests) != 2 {
		t.Fatalf("ListChargebackPlanReports() calls = %d, want 2", len(listRequests))
	}
	if listRequests[0].Page != nil {
		t.Fatalf("first ListChargebackPlanReportsRequest.Page = %q, want nil", *listRequests[0].Page)
	}
	requireStringPtr(t, "second ListChargebackPlanReportsRequest.Page", listRequests[1].Page, "page-2")
	requireStringPtr(t, "ListChargebackPlanReportsRequest.Id", listRequests[0].Id, testSourceID)
	requireStringPtr(t, "ListChargebackPlanReportsRequest.ResourceType", listRequests[0].ResourceType, testResourceType)
	requireChargebackPlanReportStatus(t, resource, testReportID, testSourceID, testResourceType, testReportName)
}

func TestChargebackPlanReportNoOpReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	updateCalled := false
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		getFn: func(_ context.Context, request opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
			requireStringPtr(t, "GetChargebackPlanReportRequest.ChargebackPlanReportId", request.ChargebackPlanReportId, testReportID)
			requireStringPtr(t, "GetChargebackPlanReportRequest.Id", request.Id, testSourceID)
			requireStringPtr(t, "GetChargebackPlanReportRequest.ResourceType", request.ResourceType, testResourceType)
			return opsisdk.GetChargebackPlanReportResponse{
				ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error) {
			updateCalled = true
			return opsisdk.UpdateChargebackPlanReportResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, chargebackPlanReportRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if updateCalled {
		t.Fatal("UpdateChargebackPlanReport() was called for matching readback")
	}
	requireChargebackPlanReportStatus(t, resource, testReportID, testSourceID, testResourceType, testReportName)
}

func TestChargebackPlanReportMutableUpdatePollsWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	resource.Spec.ReportName = testUpdatedName
	resource.Spec.ReportProperties.TimeIntervalEnd = testUpdatedTimeEnd
	var updateRequest opsisdk.UpdateChargebackPlanReportRequest
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		getFn: func(_ context.Context, request opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
			if request.ChargebackPlanReportId == nil || *request.ChargebackPlanReportId != testReportID {
				t.Fatalf("GetChargebackPlanReportRequest.ChargebackPlanReportId = %v, want %q", request.ChargebackPlanReportId, testReportID)
			}
			if updateRequest.ReportName != nil {
				return opsisdk.GetChargebackPlanReportResponse{
					ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testUpdatedName, testSourceID, testResourceType, testUpdatedTimeEnd),
				}, nil
			}
			return opsisdk.GetChargebackPlanReportResponse{
				ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
			}, nil
		},
		updateFn: func(_ context.Context, request opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error) {
			updateRequest = request
			return opsisdk.UpdateChargebackPlanReportResponse{
				OpcWorkRequestId: common.String(testUpdateWR),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testUpdateWR)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: newChargebackPlanReportWorkRequest(testUpdateWR, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeUpdated, testReportID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, chargebackPlanReportRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	requireStringPtr(t, "UpdateChargebackPlanReportRequest.ChargebackPlanReportId", updateRequest.ChargebackPlanReportId, testReportID)
	requireStringPtr(t, "UpdateChargebackPlanReportRequest.Id", updateRequest.Id, testSourceID)
	if updateRequest.ReportName == nil || *updateRequest.ReportName != testUpdatedName {
		t.Fatalf("UpdateChargebackPlanReportRequest.ReportName = %v, want %q", updateRequest.ReportName, testUpdatedName)
	}
	requireChargebackPlanReportStatus(t, resource, testReportID, testSourceID, testResourceType, testUpdatedName)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestChargebackPlanReportRejectsSourceIdentityDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	resource.Annotations[chargebackPlanReportResourceIDAnnotation] = testOtherSourceID
	updateCalled := false
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		getFn: func(context.Context, opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
			return opsisdk.GetChargebackPlanReportResponse{
				ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateChargebackPlanReportRequest) (opsisdk.UpdateChargebackPlanReportResponse, error) {
			updateCalled = true
			return opsisdk.UpdateChargebackPlanReportResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, chargebackPlanReportRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want source identity drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for source identity drift")
	}
	if updateCalled {
		t.Fatal("UpdateChargebackPlanReport() was called after source identity drift")
	}
	if !strings.Contains(err.Error(), "source resourceId is create-only") {
		t.Fatalf("CreateOrUpdate() error = %q, want create-only resourceId drift", err.Error())
	}
}

func TestChargebackPlanReportDeleteRetainsFinalizerUntilConfirmedGone(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	getCalls := 0
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		deleteFn: func(_ context.Context, request opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error) {
			requireStringPtr(t, "DeleteChargebackPlanReportRequest.ChargebackPlanReportId", request.ChargebackPlanReportId, testReportID)
			requireStringPtr(t, "DeleteChargebackPlanReportRequest.Id", request.Id, testSourceID)
			return opsisdk.DeleteChargebackPlanReportResponse{OpcWorkRequestId: common.String(testDeleteWR)}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testDeleteWR)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: newChargebackPlanReportWorkRequest(testDeleteWR, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testReportID),
			}, nil
		},
		getFn: func(context.Context, opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
			getCalls++
			if getCalls <= 2 {
				return opsisdk.GetChargebackPlanReportResponse{
					ChargebackPlanReport: newSDKChargebackPlanReport(testReportID, testReportName, testSourceID, testResourceType, testTimeEnd),
				}, nil
			}
			return opsisdk.GetChargebackPlanReportResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "report deleted")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("first Delete() deleted = true, want finalizer retained while report is still readable")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("first Delete() async current = %#v, want delete phase retained", resource.Status.OsokStatus.Async.Current)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want confirmed deletion")
	}
	if getCalls != 3 {
		t.Fatalf("GetChargebackPlanReport() calls = %d, want 3", getCalls)
	}
}

func TestChargebackPlanReportDeleteKeepsFinalizerForAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		deleteFn: func(context.Context, opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error) {
			return opsisdk.DeleteChargebackPlanReportResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true for ambiguous 404")
	}
	var ambiguous chargebackPlanReportAmbiguousNotFoundError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("Delete() error = %T, want chargebackPlanReportAmbiguousNotFoundError", err)
	}
}

func TestChargebackPlanReportDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	seedChargebackPlanReportCurrentWorkRequest(resource, shared.OSOKAsyncPhaseCreate, testCreateWR)
	deleteCalled := false
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testCreateWR)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: newChargebackPlanReportWorkRequest(testCreateWR, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, testReportID),
			}, nil
		},
		deleteFn: func(context.Context, opsisdk.DeleteChargebackPlanReportRequest) (opsisdk.DeleteChargebackPlanReportResponse, error) {
			deleteCalled = true
			return opsisdk.DeleteChargebackPlanReportResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true while create work request is pending")
	}
	if deleteCalled {
		t.Fatal("DeleteChargebackPlanReport() was called while create work request is pending")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending create work request", current)
	}
}

func TestChargebackPlanReportDeleteKeepsFinalizerForAuthShapedPostWorkRequestRead(t *testing.T) {
	t.Parallel()

	resource := newTrackedChargebackPlanReportResource()
	seedChargebackPlanReportCurrentWorkRequest(resource, shared.OSOKAsyncPhaseDelete, testDeleteWR)
	client := newTestChargebackPlanReportClient(&fakeChargebackPlanReportOCIClient{
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testDeleteWR)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: newChargebackPlanReportWorkRequest(testDeleteWR, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testReportID),
			}, nil
		},
		getFn: func(context.Context, opsisdk.GetChargebackPlanReportRequest) (opsisdk.GetChargebackPlanReportResponse, error) {
			return opsisdk.GetChargebackPlanReportResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous confirm read")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true for ambiguous post-work-request read")
	}
	var ambiguous chargebackPlanReportAmbiguousNotFoundError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("Delete() error = %T, want chargebackPlanReportAmbiguousNotFoundError", err)
	}
}

func newTestChargebackPlanReportClient(fake *fakeChargebackPlanReportOCIClient) ChargebackPlanReportServiceClient {
	hooks := newChargebackPlanReportDefaultRuntimeHooks(opsisdk.OperationsInsightsClient{})
	configureChargebackPlanReportRuntimeHooks(&hooks, fake, nil, loggerutil.OSOKLogger{})
	config := buildChargebackPlanReportGeneratedRuntimeConfig(&ChargebackPlanReportServiceManager{}, hooks)
	delegate := defaultChargebackPlanReportServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.ChargebackPlanReport](config),
	}
	return wrapChargebackPlanReportGeneratedClient(hooks, delegate)
}

func chargebackPlanReportRequest(resource *opsiv1beta1.ChargebackPlanReport) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func newChargebackPlanReportResource() *opsiv1beta1.ChargebackPlanReport {
	return &opsiv1beta1.ChargebackPlanReport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-report",
			Namespace: "default",
			Annotations: map[string]string{
				chargebackPlanReportResourceIDAnnotation:   testSourceID,
				chargebackPlanReportResourceTypeAnnotation: testResourceType,
			},
		},
		Spec: opsiv1beta1.ChargebackPlanReportSpec{
			ReportName:       testReportName,
			ReportProperties: newChargebackPlanReportSpecProperties(testTimeEnd),
		},
	}
}

func newTrackedChargebackPlanReportResource() *opsiv1beta1.ChargebackPlanReport {
	resource := newChargebackPlanReportResource()
	resource.Status.ReportId = testReportID
	resource.Status.ResourceId = testSourceID
	resource.Status.ResourceType = testResourceType
	resource.Status.ReportName = testReportName
	resource.Status.ReportProperties = newChargebackPlanReportSpecProperties(testTimeEnd)
	resource.Status.OsokStatus.Ocid = shared.OCID(testReportID)
	return resource
}

func seedChargebackPlanReportCurrentWorkRequest(resource *opsiv1beta1.ChargebackPlanReport, phase shared.OSOKAsyncPhase, workRequestID string) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func newChargebackPlanReportSpecProperties(timeEnd string) opsiv1beta1.ChargebackPlanReportReportProperties {
	groupBy := map[string]any{"dimension": "databaseName"}
	payload, _ := json.Marshal(groupBy)
	return opsiv1beta1.ChargebackPlanReportReportProperties{
		AnalysisTimeInterval: "P30D",
		TimeIntervalStart:    testTimeStart,
		TimeIntervalEnd:      timeEnd,
		GroupBy:              shared.JSONValue{Raw: payload},
	}
}

func newSDKChargebackPlanReport(reportID, reportName, resourceID, resourceType, timeEnd string) opsisdk.ChargebackPlanReport {
	return opsisdk.ChargebackPlanReport{
		ReportId:         common.String(reportID),
		ReportName:       common.String(reportName),
		ResourceId:       common.String(resourceID),
		ResourceType:     opsisdk.ChargebackPlanReportResourceTypeEnum(resourceType),
		TimeCreated:      sdkTime(testTimeStart),
		TimeUpdated:      sdkTime(timeEnd),
		ReportProperties: sdkReportProperties(timeEnd),
	}
}

func newSDKChargebackPlanReportSummary(reportID, reportName, resourceID, resourceType, timeEnd string) opsisdk.ChargebackPlanReportSummary {
	return opsisdk.ChargebackPlanReportSummary{
		ReportId:         common.String(reportID),
		ReportName:       common.String(reportName),
		ResourceId:       common.String(resourceID),
		ResourceType:     opsisdk.ChargebackPlanReportSummaryResourceTypeEnum(resourceType),
		TimeCreated:      sdkTime(testTimeStart),
		TimeUpdated:      sdkTime(timeEnd),
		ReportProperties: sdkReportProperties(timeEnd),
	}
}

func sdkReportProperties(timeEnd string) *opsisdk.ReportPropertyDetails {
	groupBy := any(map[string]any{"dimension": "databaseName"})
	return &opsisdk.ReportPropertyDetails{
		AnalysisTimeInterval: common.String("P30D"),
		TimeIntervalStart:    sdkTime(testTimeStart),
		TimeIntervalEnd:      sdkTime(timeEnd),
		GroupBy:              &groupBy,
	}
}

func sdkTime(value string) *common.SDKTime {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &common.SDKTime{Time: parsed}
}

func newChargebackPlanReportWorkRequest(
	id string,
	status opsisdk.OperationStatusEnum,
	action opsisdk.ActionTypeEnum,
	reportID string,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		Id:              common.String(id),
		Status:          status,
		OperationType:   opsisdk.OperationTypeCreateChargeBack,
		PercentComplete: common.Float32(100),
		Resources: []opsisdk.WorkRequestResource{
			{
				EntityType: common.String(chargebackPlanReportWorkRequestEntityType),
				ActionType: action,
				Identifier: common.String(reportID),
			},
		},
	}
}

func requireChargebackPlanReportStatus(
	t *testing.T,
	resource *opsiv1beta1.ChargebackPlanReport,
	reportID string,
	resourceID string,
	resourceType string,
	reportName string,
) {
	t.Helper()
	if got := resource.Status.ReportId; got != reportID {
		t.Fatalf("status.reportId = %q, want %q", got, reportID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != reportID {
		t.Fatalf("status.status.ocid = %q, want %q", got, reportID)
	}
	if got := resource.Status.ResourceId; got != resourceID {
		t.Fatalf("status.resourceId = %q, want %q", got, resourceID)
	}
	if got := resource.Status.ResourceType; got != resourceType {
		t.Fatalf("status.resourceType = %q, want %q", got, resourceType)
	}
	if got := resource.Status.ReportName; got != reportName {
		t.Fatalf("status.reportName = %q, want %q", got, reportName)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}
