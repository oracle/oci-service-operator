/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package softwaresource

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCompartmentID       = "ocid1.compartment.oc1..software"
	testSoftwareSourceID    = "ocid1.softwaresource.oc1..software"
	testOtherSoftwareSource = "ocid1.softwaresource.oc1..other"
	testVendorSourceID      = "ocid1.softwaresource.oc1..vendor"
	testDisplayName         = "ol8-custom"
	testUpdatedDisplayName  = "ol8-custom-renamed"
)

type fakeSoftwareSourceOCIClient struct {
	createRequests []osmanagementhubsdk.CreateSoftwareSourceRequest
	getRequests    []osmanagementhubsdk.GetSoftwareSourceRequest
	listRequests   []osmanagementhubsdk.ListSoftwareSourcesRequest
	updateRequests []osmanagementhubsdk.UpdateSoftwareSourceRequest
	deleteRequests []osmanagementhubsdk.DeleteSoftwareSourceRequest

	createFunc func(context.Context, osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error)
	getFunc    func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error)
	listFunc   func(context.Context, osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error)
	updateFunc func(context.Context, osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error)
	deleteFunc func(context.Context, osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error)
}

func (f *fakeSoftwareSourceOCIClient) CreateSoftwareSource(
	ctx context.Context,
	request osmanagementhubsdk.CreateSoftwareSourceRequest,
) (osmanagementhubsdk.CreateSoftwareSourceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return osmanagementhubsdk.CreateSoftwareSourceResponse{}, nil
}

func (f *fakeSoftwareSourceOCIClient) GetSoftwareSource(
	ctx context.Context,
	request osmanagementhubsdk.GetSoftwareSourceRequest,
) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return osmanagementhubsdk.GetSoftwareSourceResponse{}, nil
}

func (f *fakeSoftwareSourceOCIClient) ListSoftwareSources(
	ctx context.Context,
	request osmanagementhubsdk.ListSoftwareSourcesRequest,
) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return osmanagementhubsdk.ListSoftwareSourcesResponse{}, nil
}

func (f *fakeSoftwareSourceOCIClient) UpdateSoftwareSource(
	ctx context.Context,
	request osmanagementhubsdk.UpdateSoftwareSourceRequest,
) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return osmanagementhubsdk.UpdateSoftwareSourceResponse{}, nil
}

func (f *fakeSoftwareSourceOCIClient) DeleteSoftwareSource(
	ctx context.Context,
	request osmanagementhubsdk.DeleteSoftwareSourceRequest,
) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return osmanagementhubsdk.DeleteSoftwareSourceResponse{}, nil
}

func TestSoftwareSourceCreateUsesPolymorphicBodyAndRecordsRequestID(t *testing.T) {
	resource := testSoftwareSource()
	fake := &fakeSoftwareSourceOCIClient{}
	configureSoftwareSourceCreateFake(t, fake)

	response, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSoftwareSourceCreated(t, resource, response, fake)
}

func configureSoftwareSourceCreateFake(t *testing.T, fake *fakeSoftwareSourceOCIClient) {
	t.Helper()
	fake.listFunc = func(_ context.Context, request osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
		assertSoftwareSourceCreateListRequest(t, request)
		return osmanagementhubsdk.ListSoftwareSourcesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error) {
		assertSoftwareSourceCreateRequest(t, request)
		return osmanagementhubsdk.CreateSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateCreating),
			OpcRequestId:   common.String("create-opc"),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		assertSoftwareSourceCreateFollowUpRequest(t, request)
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
		}, nil
	}
}

func assertSoftwareSourceCreateListRequest(t *testing.T, request osmanagementhubsdk.ListSoftwareSourcesRequest) {
	t.Helper()
	if got, want := stringValue(request.CompartmentId), testCompartmentID; got != want {
		t.Fatalf("ListSoftwareSources CompartmentId = %q, want %q", got, want)
	}
}

func assertSoftwareSourceCreateRequest(t *testing.T, request osmanagementhubsdk.CreateSoftwareSourceRequest) {
	t.Helper()
	body, ok := request.CreateSoftwareSourceDetails.(osmanagementhubsdk.CreateCustomSoftwareSourceDetails)
	if !ok {
		t.Fatalf("CreateSoftwareSourceDetails = %T, want CreateCustomSoftwareSourceDetails", request.CreateSoftwareSourceDetails)
	}
	assertSoftwareSourceCreateBody(t, body)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("create OpcRetryToken is empty, want deterministic retry token")
	}
}

func assertSoftwareSourceCreateBody(t *testing.T, body osmanagementhubsdk.CreateCustomSoftwareSourceDetails) {
	t.Helper()
	if got, want := stringValue(body.CompartmentId), testCompartmentID; got != want {
		t.Fatalf("create CompartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(body.DisplayName), testDisplayName; got != want {
		t.Fatalf("create DisplayName = %q, want %q", got, want)
	}
	if len(body.VendorSoftwareSources) != 1 || stringValue(body.VendorSoftwareSources[0].Id) != testVendorSourceID {
		t.Fatalf("create VendorSoftwareSources = %#v, want source %q", body.VendorSoftwareSources, testVendorSourceID)
	}
	if body.IsAutoResolveDependencies == nil || !*body.IsAutoResolveDependencies {
		t.Fatal("create IsAutoResolveDependencies = nil/false, want true")
	}
}

func assertSoftwareSourceCreateFollowUpRequest(t *testing.T, request osmanagementhubsdk.GetSoftwareSourceRequest) {
	t.Helper()
	if got, want := stringValue(request.SoftwareSourceId), testSoftwareSourceID; got != want {
		t.Fatalf("GetSoftwareSource SoftwareSourceId = %q, want %q", got, want)
	}
}

func assertSoftwareSourceCreated(
	t *testing.T,
	resource *osmanagementhubv1beta1.SoftwareSource,
	response servicemanager.OSOKResponse,
	fake *fakeSoftwareSourceOCIClient,
) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testSoftwareSourceID; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "create-opc"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if got := len(fake.createRequests); got != 1 {
		t.Fatalf("CreateSoftwareSource calls = %d, want 1", got)
	}
}

func TestSoftwareSourceCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := testSoftwareSource()
	fake := &fakeSoftwareSourceOCIClient{}
	fake.listFunc = func(_ context.Context, request osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
		switch len(fake.listRequests) {
		case 1:
			return osmanagementhubsdk.ListSoftwareSourcesResponse{
				SoftwareSourceCollection: osmanagementhubsdk.SoftwareSourceCollection{
					Items: []osmanagementhubsdk.SoftwareSourceSummary{
						customSoftwareSourceSummary(testOtherSoftwareSource, "other", osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			if got, want := stringValue(request.Page), "page-2"; got != want {
				t.Fatalf("second ListSoftwareSources Page = %q, want %q", got, want)
			}
			return osmanagementhubsdk.ListSoftwareSourcesResponse{
				SoftwareSourceCollection: osmanagementhubsdk.SoftwareSourceCollection{
					Items: []osmanagementhubsdk.SoftwareSourceSummary{
						customSoftwareSourceSummary(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected ListSoftwareSources call %d", len(fake.listRequests))
			return osmanagementhubsdk.ListSoftwareSourcesResponse{}, nil
		}
	}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
		}, nil
	}
	fake.createFunc = func(context.Context, osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error) {
		t.Fatal("CreateSoftwareSource() called for existing SoftwareSource")
		return osmanagementhubsdk.CreateSoftwareSourceResponse{}, nil
	}

	if _, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest()); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testSoftwareSourceID; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := len(fake.listRequests); got != 2 {
		t.Fatalf("ListSoftwareSources calls = %d, want 2", got)
	}
	if got := len(fake.createRequests); got != 0 {
		t.Fatalf("CreateSoftwareSource calls = %d, want 0", got)
	}
}

func TestSoftwareSourceNoopReconcileDoesNotUpdate(t *testing.T) {
	resource := testSoftwareSource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
		t.Fatal("UpdateSoftwareSource() called for no-op reconcile")
		return osmanagementhubsdk.UpdateSoftwareSourceResponse{}, nil
	}

	if _, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest()); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateSoftwareSource calls = %d, want 0", got)
	}
}

func TestSoftwareSourceObservedCreatingStateRequeues(t *testing.T) {
	resource := testSoftwareSource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateCreating),
		}, nil
	}

	response, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true for CREATING")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle async projection")
	}
	if got, want := current.Phase, shared.OSOKAsyncPhaseCreate; got != want {
		t.Fatalf("async.current.phase = %q, want %q", got, want)
	}
	if got, want := current.RawStatus, string(osmanagementhubsdk.SoftwareSourceLifecycleStateCreating); got != want {
		t.Fatalf("async.current.rawStatus = %q, want %q", got, want)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateSoftwareSource calls = %d, want 0 while CREATING", got)
	}
}

func TestSoftwareSourceMutableUpdateUsesTypedUpdateBody(t *testing.T) {
	resource := testSoftwareSource()
	resource.Spec.DisplayName = testUpdatedDisplayName
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		if len(fake.getRequests) == 1 {
			return osmanagementhubsdk.GetSoftwareSourceResponse{
				SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
			}, nil
		}
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testUpdatedDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
		if got, want := stringValue(request.SoftwareSourceId), testSoftwareSourceID; got != want {
			t.Fatalf("UpdateSoftwareSource SoftwareSourceId = %q, want %q", got, want)
		}
		body, ok := request.UpdateSoftwareSourceDetails.(osmanagementhubsdk.UpdateCustomSoftwareSourceDetails)
		if !ok {
			t.Fatalf("UpdateSoftwareSourceDetails = %T, want UpdateCustomSoftwareSourceDetails", request.UpdateSoftwareSourceDetails)
		}
		if got, want := stringValue(body.DisplayName), testUpdatedDisplayName; got != want {
			t.Fatalf("update DisplayName = %q, want %q", got, want)
		}
		return osmanagementhubsdk.UpdateSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testUpdatedDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateUpdating),
			OpcRequestId:   common.String("update-opc"),
		}, nil
	}

	response, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("UpdateSoftwareSource calls = %d, want 1", got)
	}
	if got, want := resource.Status.DisplayName, testUpdatedDisplayName; got != want {
		t.Fatalf("status.displayName = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "update-opc"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestSoftwareSourceCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := testSoftwareSource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
		t.Fatal("UpdateSoftwareSource() called after create-only drift")
		return osmanagementhubsdk.UpdateSoftwareSourceResponse{}, nil
	}

	_, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId create-only drift", err)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateSoftwareSource calls = %d, want 0", got)
	}
}

func TestSoftwareSourceDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := testSoftwareSource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		if len(fake.getRequests) == 1 {
			return osmanagementhubsdk.GetSoftwareSourceResponse{
				SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
			}, nil
		}
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateDeleting),
		}, nil
	}
	fake.deleteFunc = func(context.Context, osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error) {
		return osmanagementhubsdk.DeleteSoftwareSourceResponse{OpcRequestId: common.String("delete-opc")}, nil
	}

	deleted, err := newSoftwareSourceServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("DeleteSoftwareSource calls = %d, want 1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got, want := resource.Status.LifecycleState, string(osmanagementhubsdk.SoftwareSourceLifecycleStateDeleting); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
}

func TestSoftwareSourceDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := testSoftwareSource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSource(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error) {
		return osmanagementhubsdk.DeleteSoftwareSourceResponse{}, authErr
	}

	deleted, err := newSoftwareSourceServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not-found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got == "" {
		t.Fatal("status.status.opcRequestId is empty, want OCI error request id")
	}
}

func TestSoftwareSourceDeleteRejectsAuthShapedPreDeleteReadBeforeDelete(t *testing.T) {
	resource := testSoftwareSource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		return osmanagementhubsdk.GetSoftwareSourceResponse{}, authErr
	}
	fake.deleteFunc = func(context.Context, osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error) {
		t.Fatal("DeleteSoftwareSource() called after auth-shaped pre-delete read")
		return osmanagementhubsdk.DeleteSoftwareSourceResponse{}, nil
	}

	deleted, err := newSoftwareSourceServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped pre-delete read error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("DeleteSoftwareSource calls = %d, want 0 after auth-shaped pre-delete read", got)
	}
	if got := len(fake.getRequests); got == 0 {
		t.Fatal("GetSoftwareSource calls = 0, want pre-delete readback")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped pre-delete read", resource.Status.OsokStatus.DeletedAt)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestSoftwareSourceCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := testSoftwareSource()
	ociErr := errortest.NewServiceError(500, "InternalError", "temporary failure")
	fake := &fakeSoftwareSourceOCIClient{}
	fake.listFunc = func(context.Context, osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
		return osmanagementhubsdk.ListSoftwareSourcesResponse{}, nil
	}
	fake.createFunc = func(context.Context, osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error) {
		return osmanagementhubsdk.CreateSoftwareSourceResponse{}, ociErr
	}

	_, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got == "" {
		t.Fatal("status.status.opcRequestId is empty, want OCI error request id")
	}
}

func TestSoftwareSourceJsonDataPreservesExplicitFalseInPolymorphicCreateBody(t *testing.T) {
	resource := testSoftwareSource()
	resource.Spec.JsonData = `{
		"softwareSourceType":"PRIVATE",
		"compartmentId":"` + testCompartmentID + `",
		"displayName":"private-yum",
		"url":"https://example.invalid/yum",
		"osFamily":"ORACLE_LINUX_8",
		"archType":"X86_64",
		"isGpgCheckEnabled":false,
		"isSslVerifyEnabled":false
	}`
	resource.Spec.SoftwareSourceType = ""
	resource.Spec.DisplayName = ""
	resource.Spec.VendorSoftwareSources = nil

	body, err := buildSoftwareSourceCreateBody(resource)
	if err != nil {
		t.Fatalf("buildSoftwareSourceCreateBody() error = %v", err)
	}
	privateBody, ok := body.(osmanagementhubsdk.CreatePrivateSoftwareSourceDetails)
	if !ok {
		t.Fatalf("create body = %T, want CreatePrivateSoftwareSourceDetails", body)
	}
	if privateBody.IsGpgCheckEnabled == nil || *privateBody.IsGpgCheckEnabled {
		t.Fatalf("IsGpgCheckEnabled = %#v, want explicit false", privateBody.IsGpgCheckEnabled)
	}
	if privateBody.IsSslVerifyEnabled == nil || *privateBody.IsSslVerifyEnabled {
		t.Fatalf("IsSslVerifyEnabled = %#v, want explicit false", privateBody.IsSslVerifyEnabled)
	}
	encoded, err := json.Marshal(privateBody)
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}
	if strings.Contains(string(encoded), "jsonData") {
		t.Fatalf("create body leaked jsonData helper field: %s", encoded)
	}
}

func TestSoftwareSourceTypedPrivateCreatePreservesFalseBooleans(t *testing.T) {
	resource := testPrivateSoftwareSource()

	body, err := buildSoftwareSourceCreateBody(resource)
	if err != nil {
		t.Fatalf("buildSoftwareSourceCreateBody() error = %v", err)
	}
	privateBody, ok := body.(osmanagementhubsdk.CreatePrivateSoftwareSourceDetails)
	if !ok {
		t.Fatalf("create body = %T, want CreatePrivateSoftwareSourceDetails", body)
	}
	assertBoolPtrFalse(t, "CreatePrivateSoftwareSourceDetails.IsGpgCheckEnabled", privateBody.IsGpgCheckEnabled)
	assertBoolPtrFalse(t, "CreatePrivateSoftwareSourceDetails.IsSslVerifyEnabled", privateBody.IsSslVerifyEnabled)
}

func TestSoftwareSourceTypedCustomUpdatePreservesFalseBooleans(t *testing.T) {
	resource := testSoftwareSource()
	resource.Spec.IsAutoResolveDependencies = false
	resource.Status.OsokStatus.Ocid = shared.OCID(testSoftwareSourceID)
	fake := &fakeSoftwareSourceOCIClient{}
	fake.getFunc = func(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
		if len(fake.getRequests) == 1 {
			return osmanagementhubsdk.GetSoftwareSourceResponse{
				SoftwareSource: customSoftwareSourceWithMutableBools(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive, true, true, true),
			}, nil
		}
		return osmanagementhubsdk.GetSoftwareSourceResponse{
			SoftwareSource: customSoftwareSourceWithMutableBools(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateActive, false, false, false),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
		body, ok := request.UpdateSoftwareSourceDetails.(osmanagementhubsdk.UpdateCustomSoftwareSourceDetails)
		if !ok {
			t.Fatalf("UpdateSoftwareSourceDetails = %T, want UpdateCustomSoftwareSourceDetails", request.UpdateSoftwareSourceDetails)
		}
		assertBoolPtrFalse(t, "UpdateCustomSoftwareSourceDetails.IsAutomaticallyUpdated", body.IsAutomaticallyUpdated)
		assertBoolPtrFalse(t, "UpdateCustomSoftwareSourceDetails.IsAutoResolveDependencies", body.IsAutoResolveDependencies)
		assertBoolPtrFalse(t, "UpdateCustomSoftwareSourceDetails.IsLatestContentOnly", body.IsLatestContentOnly)
		return osmanagementhubsdk.UpdateSoftwareSourceResponse{
			SoftwareSource: customSoftwareSourceWithMutableBools(testSoftwareSourceID, testDisplayName, osmanagementhubsdk.SoftwareSourceLifecycleStateUpdating, false, false, false),
			OpcRequestId:   common.String("update-opc"),
		}, nil
	}

	if _, err := newSoftwareSourceServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, testRequest()); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("UpdateSoftwareSource calls = %d, want 1", got)
	}
}

func testSoftwareSource() *osmanagementhubv1beta1.SoftwareSource {
	return &osmanagementhubv1beta1.SoftwareSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "software-source",
			Namespace: "default",
			UID:       types.UID("software-source-uid"),
		},
		Spec: osmanagementhubv1beta1.SoftwareSourceSpec{
			CompartmentId:      testCompartmentID,
			DisplayName:        testDisplayName,
			Description:        "custom source",
			SoftwareSourceType: string(osmanagementhubsdk.SoftwareSourceTypeCustom),
			VendorSoftwareSources: []osmanagementhubv1beta1.SoftwareSourceVendorSoftwareSource{
				{Id: testVendorSourceID, DisplayName: "vendor"},
			},
			IsAutoResolveDependencies: true,
		},
	}
}

func testPrivateSoftwareSource() *osmanagementhubv1beta1.SoftwareSource {
	resource := testSoftwareSource()
	resource.Spec.SoftwareSourceType = string(osmanagementhubsdk.SoftwareSourceTypePrivate)
	resource.Spec.VendorSoftwareSources = nil
	resource.Spec.Url = "https://example.invalid/yum"
	resource.Spec.OsFamily = string(osmanagementhubsdk.OsFamilyOracleLinux8)
	resource.Spec.ArchType = string(osmanagementhubsdk.ArchTypeX8664)
	resource.Spec.IsGpgCheckEnabled = false
	resource.Spec.IsSslVerifyEnabled = false
	return resource
}

func testRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "software-source"}}
}

func customSoftwareSource(
	id string,
	displayName string,
	state osmanagementhubsdk.SoftwareSourceLifecycleStateEnum,
) osmanagementhubsdk.CustomSoftwareSource {
	return osmanagementhubsdk.CustomSoftwareSource{
		Id:                        common.String(id),
		CompartmentId:             common.String(testCompartmentID),
		DisplayName:               common.String(displayName),
		RepoId:                    common.String("repo"),
		Url:                       common.String("custom/repo"),
		VendorSoftwareSources:     []osmanagementhubsdk.Id{{Id: common.String(testVendorSourceID), DisplayName: common.String("vendor")}},
		Description:               common.String("custom source"),
		IsAutoResolveDependencies: common.Bool(true),
		Availability:              osmanagementhubsdk.AvailabilityAvailable,
		AvailabilityAtOci:         osmanagementhubsdk.AvailabilityAvailable,
		OsFamily:                  osmanagementhubsdk.OsFamilyOracleLinux8,
		ArchType:                  osmanagementhubsdk.ArchTypeX8664,
		LifecycleState:            state,
	}
}

func customSoftwareSourceWithMutableBools(
	id string,
	displayName string,
	state osmanagementhubsdk.SoftwareSourceLifecycleStateEnum,
	automaticallyUpdated bool,
	autoResolveDependencies bool,
	latestContentOnly bool,
) osmanagementhubsdk.CustomSoftwareSource {
	source := customSoftwareSource(id, displayName, state)
	source.IsAutomaticallyUpdated = common.Bool(automaticallyUpdated)
	source.IsAutoResolveDependencies = common.Bool(autoResolveDependencies)
	source.IsLatestContentOnly = common.Bool(latestContentOnly)
	return source
}

func customSoftwareSourceSummary(
	id string,
	displayName string,
	state osmanagementhubsdk.SoftwareSourceLifecycleStateEnum,
) osmanagementhubsdk.CustomSoftwareSourceSummary {
	return osmanagementhubsdk.CustomSoftwareSourceSummary{
		Id:                    common.String(id),
		CompartmentId:         common.String(testCompartmentID),
		DisplayName:           common.String(displayName),
		RepoId:                common.String("repo"),
		Url:                   common.String("custom/repo"),
		VendorSoftwareSources: []osmanagementhubsdk.Id{{Id: common.String(testVendorSourceID), DisplayName: common.String("vendor")}},
		Availability:          osmanagementhubsdk.AvailabilityAvailable,
		AvailabilityAtOci:     osmanagementhubsdk.AvailabilityAvailable,
		OsFamily:              osmanagementhubsdk.OsFamilyOracleLinux8,
		ArchType:              osmanagementhubsdk.ArchTypeX8664,
		LifecycleState:        state,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func assertBoolPtrFalse(t *testing.T, label string, value *bool) {
	t.Helper()
	if value == nil || *value {
		t.Fatalf("%s = %#v, want explicit false", label, value)
	}
}
