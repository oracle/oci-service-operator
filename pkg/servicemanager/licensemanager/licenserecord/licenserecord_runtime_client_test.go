/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package licenserecord

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	licensemanagersdk "github.com/oracle/oci-go-sdk/v65/licensemanager"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLicenseRecordID          = "ocid1.licenserecord.oc1..record"
	testProductLicenseID         = "ocid1.productlicense.oc1..parent"
	testLicenseRecordCompartment = "ocid1.compartment.oc1..compartment"
	testLicenseRecordName        = "license-record-sample"
)

type fakeLicenseRecordOCIClient struct {
	createFn func(context.Context, licensemanagersdk.CreateLicenseRecordRequest) (licensemanagersdk.CreateLicenseRecordResponse, error)
	getFn    func(context.Context, licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error)
	listFn   func(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error)
	updateFn func(context.Context, licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error)
	deleteFn func(context.Context, licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error)
}

func (f *fakeLicenseRecordOCIClient) CreateLicenseRecord(
	ctx context.Context,
	req licensemanagersdk.CreateLicenseRecordRequest,
) (licensemanagersdk.CreateLicenseRecordResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return licensemanagersdk.CreateLicenseRecordResponse{}, nil
}

func (f *fakeLicenseRecordOCIClient) GetLicenseRecord(
	ctx context.Context,
	req licensemanagersdk.GetLicenseRecordRequest,
) (licensemanagersdk.GetLicenseRecordResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return licensemanagersdk.GetLicenseRecordResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "LicenseRecord is missing")
}

func (f *fakeLicenseRecordOCIClient) ListLicenseRecords(
	ctx context.Context,
	req licensemanagersdk.ListLicenseRecordsRequest,
) (licensemanagersdk.ListLicenseRecordsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return licensemanagersdk.ListLicenseRecordsResponse{}, nil
}

func (f *fakeLicenseRecordOCIClient) UpdateLicenseRecord(
	ctx context.Context,
	req licensemanagersdk.UpdateLicenseRecordRequest,
) (licensemanagersdk.UpdateLicenseRecordResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return licensemanagersdk.UpdateLicenseRecordResponse{}, nil
}

func (f *fakeLicenseRecordOCIClient) DeleteLicenseRecord(
	ctx context.Context,
	req licensemanagersdk.DeleteLicenseRecordRequest,
) (licensemanagersdk.DeleteLicenseRecordResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return licensemanagersdk.DeleteLicenseRecordResponse{}, nil
}

func newTestLicenseRecordClient(client licenseRecordOCIClient) LicenseRecordServiceClient {
	return newLicenseRecordServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeLicenseRecordResource() *licensemanagerv1beta1.LicenseRecord {
	return &licensemanagerv1beta1.LicenseRecord{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testLicenseRecordName,
			Namespace: "default",
			Annotations: map[string]string{
				LicenseRecordProductLicenseIDAnnotation: testProductLicenseID,
			},
		},
		Spec: licensemanagerv1beta1.LicenseRecordSpec{
			DisplayName:    testLicenseRecordName,
			IsPerpetual:    false,
			IsUnlimited:    true,
			ExpirationDate: "2027-01-02",
			SupportEndDate: "2027-06-03T04:05:06Z",
			LicenseCount:   3,
			ProductId:      "license-product",
			FreeformTags:   map[string]string{"env": "test"},
			DefinedTags:    map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedLicenseRecordResource() *licensemanagerv1beta1.LicenseRecord {
	resource := makeLicenseRecordResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLicenseRecordID)
	resource.Status.Id = testLicenseRecordID
	resource.Status.ProductLicenseId = testProductLicenseID
	resource.Status.CompartmentId = testLicenseRecordCompartment
	resource.Status.DisplayName = testLicenseRecordName
	resource.Status.LifecycleState = string(licensemanagersdk.LifeCycleStateActive)
	return resource
}

func makeLicenseRecordRequest(resource *licensemanagerv1beta1.LicenseRecord) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKLicenseRecord(
	id string,
	spec licensemanagerv1beta1.LicenseRecordSpec,
	state licensemanagersdk.LifeCycleStateEnum,
) licensemanagersdk.LicenseRecord {
	expirationDate, _ := licenseRecordOptionalSDKTime("expirationDate", spec.ExpirationDate)
	supportEndDate, _ := licenseRecordOptionalSDKTime("supportEndDate", spec.SupportEndDate)
	return licensemanagersdk.LicenseRecord{
		Id:               common.String(id),
		DisplayName:      common.String(spec.DisplayName),
		IsUnlimited:      common.Bool(spec.IsUnlimited),
		IsPerpetual:      common.Bool(spec.IsPerpetual),
		LifecycleState:   state,
		ProductLicenseId: common.String(testProductLicenseID),
		CompartmentId:    common.String(testLicenseRecordCompartment),
		ProductId:        common.String(spec.ProductId),
		LicenseCount:     common.Int(spec.LicenseCount),
		ExpirationDate:   expirationDate,
		SupportEndDate:   supportEndDate,
		FreeformTags:     cloneLicenseRecordStringMap(spec.FreeformTags),
		DefinedTags:      licenseRecordDefinedTags(spec.DefinedTags),
	}
}

func makeSDKLicenseRecordSummary(
	id string,
	spec licensemanagerv1beta1.LicenseRecordSpec,
	state licensemanagersdk.LifeCycleStateEnum,
) licensemanagersdk.LicenseRecordSummary {
	record := makeSDKLicenseRecord(id, spec, state)
	return licensemanagersdk.LicenseRecordSummary{
		Id:               record.Id,
		DisplayName:      record.DisplayName,
		IsUnlimited:      record.IsUnlimited,
		IsPerpetual:      record.IsPerpetual,
		ProductLicenseId: record.ProductLicenseId,
		CompartmentId:    record.CompartmentId,
		ProductId:        record.ProductId,
		LicenseCount:     record.LicenseCount,
		ExpirationDate:   record.ExpirationDate,
		SupportEndDate:   record.SupportEndDate,
		LifecycleState:   state,
		FreeformTags:     record.FreeformTags,
		DefinedTags:      record.DefinedTags,
	}
}

type licenseRecordPagedBindFixture struct {
	resource  *licensemanagerv1beta1.LicenseRecord
	listCalls int
}

func (f *licenseRecordPagedBindFixture) listLicenseRecords(
	t *testing.T,
) func(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
	t.Helper()
	return func(_ context.Context, req licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
		f.listCalls++
		requireStringPtr(t, "ListLicenseRecordsRequest.ProductLicenseId", req.ProductLicenseId, testProductLicenseID)
		if f.listCalls == 1 {
			return f.firstLicenseRecordListPage(t, req), nil
		}
		requireStringPtr(t, "second ListLicenseRecordsRequest.Page", req.Page, "page-2")
		return f.secondLicenseRecordListPage(), nil
	}
}

func (f *licenseRecordPagedBindFixture) firstLicenseRecordListPage(
	t *testing.T,
	req licensemanagersdk.ListLicenseRecordsRequest,
) licensemanagersdk.ListLicenseRecordsResponse {
	t.Helper()
	if req.Page != nil {
		t.Fatalf("first ListLicenseRecordsRequest.Page = %q, want nil", *req.Page)
	}
	otherSpec := f.resource.Spec
	otherSpec.DisplayName = "other-record"
	return licensemanagersdk.ListLicenseRecordsResponse{
		LicenseRecordCollection: licensemanagersdk.LicenseRecordCollection{
			Items: []licensemanagersdk.LicenseRecordSummary{
				makeSDKLicenseRecordSummary("ocid1.licenserecord.oc1..other", otherSpec, licensemanagersdk.LifeCycleStateActive),
			},
		},
		OpcNextPage: common.String("page-2"),
	}
}

func (f *licenseRecordPagedBindFixture) secondLicenseRecordListPage() licensemanagersdk.ListLicenseRecordsResponse {
	return licensemanagersdk.ListLicenseRecordsResponse{
		LicenseRecordCollection: licensemanagersdk.LicenseRecordCollection{
			Items: []licensemanagersdk.LicenseRecordSummary{
				makeSDKLicenseRecordSummary(testLicenseRecordID, f.resource.Spec, licensemanagersdk.LifeCycleStateActive),
			},
		},
	}
}

func TestLicenseRecordCreateOrUpdateBindsExistingRecordByPagedProductLicenseList(t *testing.T) {
	t.Parallel()

	fixture := &licenseRecordPagedBindFixture{resource: makeLicenseRecordResource()}
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		listFn: fixture.listLicenseRecords(t),
		getFn: func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			getCalls++
			requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			return licensemanagersdk.GetLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, fixture.resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		},
		createFn: func(context.Context, licensemanagersdk.CreateLicenseRecordRequest) (licensemanagersdk.CreateLicenseRecordResponse, error) {
			createCalled = true
			return licensemanagersdk.CreateLicenseRecordResponse{}, nil
		},
		updateFn: func(context.Context, licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error) {
			updateCalled = true
			return licensemanagersdk.UpdateLicenseRecordResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), fixture.resource, makeLicenseRecordRequest(fixture.resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateLicenseRecord() called for existing record")
	}
	if updateCalled {
		t.Fatal("UpdateLicenseRecord() called for matching record")
	}
	if fixture.listCalls != 2 {
		t.Fatalf("ListLicenseRecords() calls = %d, want 2 paginated calls", fixture.listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetLicenseRecord() calls = %d, want 1", getCalls)
	}
	if got := string(fixture.resource.Status.OsokStatus.Ocid); got != testLicenseRecordID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testLicenseRecordID)
	}
	if got := fixture.resource.Status.ProductLicenseId; got != testProductLicenseID {
		t.Fatalf("status.productLicenseId = %q, want %q", got, testProductLicenseID)
	}
	requireLastCondition(t, fixture.resource, shared.Active)
}

func TestLicenseRecordCreateRecordsTypedPayloadRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeLicenseRecordResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		listFn: func(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
			listCalls++
			return licensemanagersdk.ListLicenseRecordsResponse{}, nil
		},
		createFn: func(_ context.Context, req licensemanagersdk.CreateLicenseRecordRequest) (licensemanagersdk.CreateLicenseRecordResponse, error) {
			createCalls++
			requireLicenseRecordCreateRequest(t, req, resource)
			return licensemanagersdk.CreateLicenseRecordResponse{
				LicenseRecord:    makeSDKLicenseRecord(testLicenseRecordID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			getCalls++
			requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			return licensemanagersdk.GetLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLicenseRecordRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 || createCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testLicenseRecordID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testLicenseRecordID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after active readback", resource.Status.OsokStatus.Async.Current)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestLicenseRecordCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedLicenseRecordResource()
	updateCalled := false

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		getFn: func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			return licensemanagersdk.GetLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error) {
			updateCalled = true
			return licensemanagersdk.UpdateLicenseRecordResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLicenseRecordRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateLicenseRecord() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestLicenseRecordCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedLicenseRecordResource()
	currentSpec := resource.Spec
	resource.Spec.DisplayName = "updated-record"
	resource.Spec.IsPerpetual = true
	resource.Spec.IsUnlimited = false
	resource.Spec.ExpirationDate = "2028-01-02"
	resource.Spec.SupportEndDate = "2028-06-03T04:05:06Z"
	resource.Spec.LicenseCount = 7
	resource.Spec.ProductId = "updated-product"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	getCalls := 0
	updateCalls := 0

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		getFn: func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			getCalls++
			requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			if getCalls == 1 {
				return licensemanagersdk.GetLicenseRecordResponse{
					LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, currentSpec, licensemanagersdk.LifeCycleStateActive),
				}, nil
			}
			return licensemanagersdk.GetLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			requireStringPtr(t, "UpdateLicenseRecordDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireBoolPtr(t, "UpdateLicenseRecordDetails.IsPerpetual", req.IsPerpetual, true)
			requireBoolPtr(t, "UpdateLicenseRecordDetails.IsUnlimited", req.IsUnlimited, false)
			requireSDKTimeDate(t, "UpdateLicenseRecordDetails.ExpirationDate", req.ExpirationDate, "2028-01-02")
			requireSDKTimeDate(t, "UpdateLicenseRecordDetails.SupportEndDate", req.SupportEndDate, "2028-06-03")
			requireIntPtr(t, "UpdateLicenseRecordDetails.LicenseCount", req.LicenseCount, 7)
			requireStringPtr(t, "UpdateLicenseRecordDetails.ProductId", req.ProductId, resource.Spec.ProductId)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateLicenseRecordDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateLicenseRecordDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return licensemanagersdk.UpdateLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
				OpcRequestId:  common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLicenseRecordRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 || getCalls != 2 {
		t.Fatalf("call counts update/get = %d/%d, want 1/2", updateCalls, getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestLicenseRecordCreateOrUpdateRejectsParentAnnotationDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedLicenseRecordResource()
	resource.Annotations[LicenseRecordProductLicenseIDAnnotation] = "ocid1.productlicense.oc1..different"
	getCalled := false
	updateCalled := false

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		getFn: func(context.Context, licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			getCalled = true
			return licensemanagersdk.GetLicenseRecordResponse{}, nil
		},
		updateFn: func(context.Context, licensemanagersdk.UpdateLicenseRecordRequest) (licensemanagersdk.UpdateLicenseRecordResponse, error) {
			updateCalled = true
			return licensemanagersdk.UpdateLicenseRecordResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLicenseRecordRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want parent drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if getCalled {
		t.Fatal("GetLicenseRecord() called after parent annotation drift")
	}
	if updateCalled {
		t.Fatal("UpdateLicenseRecord() called after parent annotation drift")
	}
	if !strings.Contains(err.Error(), "parent product license annotation") {
		t.Fatalf("CreateOrUpdate() error = %v, want parent annotation drift", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

type licenseRecordDeleteFixture struct {
	resource    *licensemanagerv1beta1.LicenseRecord
	client      LicenseRecordServiceClient
	getCalls    int
	deleteCalls int
}

func TestLicenseRecordDeleteRetainsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	fixture := newLicenseRecordDeleteFixture(t)
	fixture.requireFirstDeletePending(t)
	fixture.requireSecondDeleteConfirmed(t)
}

func newLicenseRecordDeleteFixture(t *testing.T) *licenseRecordDeleteFixture {
	t.Helper()
	fixture := &licenseRecordDeleteFixture{resource: makeTrackedLicenseRecordResource()}
	fixture.client = newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		getFn:    fixture.getLicenseRecordDuringDelete(t),
		deleteFn: fixture.deleteLicenseRecord(t),
	})
	return fixture
}

func (f *licenseRecordDeleteFixture) getLicenseRecordDuringDelete(
	t *testing.T,
) func(context.Context, licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
	t.Helper()
	return func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
		f.getCalls++
		requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
		if f.deleteCalls == 0 || f.getCalls == 3 {
			return licensemanagersdk.GetLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, f.resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		}
		return licensemanagersdk.GetLicenseRecordResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "LicenseRecord is gone")
	}
}

func (f *licenseRecordDeleteFixture) deleteLicenseRecord(
	t *testing.T,
) func(context.Context, licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error) {
	t.Helper()
	return func(_ context.Context, req licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error) {
		f.deleteCalls++
		requireStringPtr(t, "DeleteLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
		return licensemanagersdk.DeleteLicenseRecordResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("opc-delete"),
		}, nil
	}
}

func (f *licenseRecordDeleteFixture) requireFirstDeletePending(t *testing.T) {
	t.Helper()
	deleted, err := f.client.Delete(context.Background(), f.resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains active")
	}
	if f.deleteCalls != 1 {
		t.Fatalf("DeleteLicenseRecord() calls after first delete = %d, want 1", f.deleteCalls)
	}
	if got := f.resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireLifecycleDeleteStatus(t, f.resource, licensemanagersdk.LifeCycleStateActive)
	requireLastCondition(t, f.resource, shared.Terminating)
}

func (f *licenseRecordDeleteFixture) requireSecondDeleteConfirmed(t *testing.T) {
	t.Helper()
	deleted, err := f.client.Delete(context.Background(), f.resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after NotFound confirmation")
	}
	if f.deleteCalls != 1 {
		t.Fatalf("DeleteLicenseRecord() calls after confirmed delete = %d, want still 1", f.deleteCalls)
	}
	if f.resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, f.resource, shared.Terminating)
}

func TestLicenseRecordDeleteTreatsAuthShapedPreDeleteReadConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedLicenseRecordResource()
	deleteCalled := false

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		getFn: func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			return licensemanagersdk.GetLicenseRecordResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error) {
			deleteCalled = true
			return licensemanagersdk.DeleteLicenseRecordResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if deleteCalled {
		t.Fatal("DeleteLicenseRecord() called after auth-shaped pre-delete read")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestLicenseRecordDeleteTreatsAuthShapedDeleteErrorConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedLicenseRecordResource()

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		getFn: func(_ context.Context, req licensemanagersdk.GetLicenseRecordRequest) (licensemanagersdk.GetLicenseRecordResponse, error) {
			requireStringPtr(t, "GetLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			return licensemanagersdk.GetLicenseRecordResponse{
				LicenseRecord: makeSDKLicenseRecord(testLicenseRecordID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req licensemanagersdk.DeleteLicenseRecordRequest) (licensemanagersdk.DeleteLicenseRecordResponse, error) {
			requireStringPtr(t, "DeleteLicenseRecordRequest.LicenseRecordId", req.LicenseRecordId, testLicenseRecordID)
			return licensemanagersdk.DeleteLicenseRecordResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous delete error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete error")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestLicenseRecordCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeLicenseRecordResource()

	client := newTestLicenseRecordClient(&fakeLicenseRecordOCIClient{
		listFn: func(context.Context, licensemanagersdk.ListLicenseRecordsRequest) (licensemanagersdk.ListLicenseRecordsResponse, error) {
			return licensemanagersdk.ListLicenseRecordsResponse{}, nil
		},
		createFn: func(context.Context, licensemanagersdk.CreateLicenseRecordRequest) (licensemanagersdk.CreateLicenseRecordResponse, error) {
			return licensemanagersdk.CreateLicenseRecordResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLicenseRecordRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireLicenseRecordCreateRequest(
	t *testing.T,
	req licensemanagersdk.CreateLicenseRecordRequest,
	resource *licensemanagerv1beta1.LicenseRecord,
) {
	t.Helper()
	requireStringPtr(t, "CreateLicenseRecordRequest.ProductLicenseId", req.ProductLicenseId, testProductLicenseID)
	requireStringPtr(t, "CreateLicenseRecordDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireBoolPtr(t, "CreateLicenseRecordDetails.IsPerpetual", req.IsPerpetual, false)
	requireBoolPtr(t, "CreateLicenseRecordDetails.IsUnlimited", req.IsUnlimited, true)
	requireSDKTimeDate(t, "CreateLicenseRecordDetails.ExpirationDate", req.ExpirationDate, "2027-01-02")
	requireSDKTimeDate(t, "CreateLicenseRecordDetails.SupportEndDate", req.SupportEndDate, "2027-06-03")
	requireIntPtr(t, "CreateLicenseRecordDetails.LicenseCount", req.LicenseCount, 3)
	requireStringPtr(t, "CreateLicenseRecordDetails.ProductId", req.ProductId, resource.Spec.ProductId)
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateLicenseRecordDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateLicenseRecordDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateLicenseRecordRequest.OpcRetryToken is empty, want deterministic retry token")
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

func requireBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func requireIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", name, *got, want)
	}
}

func requireSDKTimeDate(t *testing.T, name string, got *common.SDKTime, wantDate string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want date %q", name, wantDate)
	}
	if gotDate := got.Format("2006-01-02"); gotDate != wantDate {
		t.Fatalf("%s date = %q, want %q", name, gotDate, wantDate)
	}
}

func requireLifecycleDeleteStatus(
	t *testing.T,
	resource *licensemanagerv1beta1.LicenseRecord,
	wantRaw licensemanagersdk.LifeCycleStateEnum,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle delete tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.RawStatus != string(wantRaw) ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want lifecycle/delete/%s/pending", current, wantRaw)
	}
}

func requireLastCondition(t *testing.T, resource *licensemanagerv1beta1.LicenseRecord, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
