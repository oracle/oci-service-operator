/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package librarymaskingformat

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLibraryMaskingFormatCompartmentID = "ocid1.compartment.oc1..librarymaskingformat"
	testLibraryMaskingFormatID            = "ocid1.librarymaskingformat.oc1..resource"
)

type fakeLibraryMaskingFormatOCI struct {
	createRequests []datasafesdk.CreateLibraryMaskingFormatRequest
	getRequests    []datasafesdk.GetLibraryMaskingFormatRequest
	listRequests   []datasafesdk.ListLibraryMaskingFormatsRequest
	updateRequests []datasafesdk.UpdateLibraryMaskingFormatRequest
	deleteRequests []datasafesdk.DeleteLibraryMaskingFormatRequest

	create func(context.Context, datasafesdk.CreateLibraryMaskingFormatRequest) (datasafesdk.CreateLibraryMaskingFormatResponse, error)
	get    func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error)
	list   func(context.Context, datasafesdk.ListLibraryMaskingFormatsRequest) (datasafesdk.ListLibraryMaskingFormatsResponse, error)
	update func(context.Context, datasafesdk.UpdateLibraryMaskingFormatRequest) (datasafesdk.UpdateLibraryMaskingFormatResponse, error)
	delete func(context.Context, datasafesdk.DeleteLibraryMaskingFormatRequest) (datasafesdk.DeleteLibraryMaskingFormatResponse, error)
}

func (f *fakeLibraryMaskingFormatOCI) createLibraryMaskingFormat(
	ctx context.Context,
	request datasafesdk.CreateLibraryMaskingFormatRequest,
) (datasafesdk.CreateLibraryMaskingFormatResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create != nil {
		return f.create(ctx, request)
	}
	return datasafesdk.CreateLibraryMaskingFormatResponse{}, nil
}

func (f *fakeLibraryMaskingFormatOCI) getLibraryMaskingFormat(
	ctx context.Context,
	request datasafesdk.GetLibraryMaskingFormatRequest,
) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get != nil {
		return f.get(ctx, request)
	}
	return datasafesdk.GetLibraryMaskingFormatResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
}

func (f *fakeLibraryMaskingFormatOCI) listLibraryMaskingFormats(
	ctx context.Context,
	request datasafesdk.ListLibraryMaskingFormatsRequest,
) (datasafesdk.ListLibraryMaskingFormatsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list != nil {
		return f.list(ctx, request)
	}
	return datasafesdk.ListLibraryMaskingFormatsResponse{}, nil
}

func (f *fakeLibraryMaskingFormatOCI) updateLibraryMaskingFormat(
	ctx context.Context,
	request datasafesdk.UpdateLibraryMaskingFormatRequest,
) (datasafesdk.UpdateLibraryMaskingFormatResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update != nil {
		return f.update(ctx, request)
	}
	return datasafesdk.UpdateLibraryMaskingFormatResponse{}, nil
}

func (f *fakeLibraryMaskingFormatOCI) deleteLibraryMaskingFormat(
	ctx context.Context,
	request datasafesdk.DeleteLibraryMaskingFormatRequest,
) (datasafesdk.DeleteLibraryMaskingFormatResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete != nil {
		return f.delete(ctx, request)
	}
	return datasafesdk.DeleteLibraryMaskingFormatResponse{}, nil
}

func newTestLibraryMaskingFormatClient(fake *fakeLibraryMaskingFormatOCI) LibraryMaskingFormatServiceClient {
	hooks := newLibraryMaskingFormatDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	hooks.Create.Call = fake.createLibraryMaskingFormat
	hooks.Get.Call = fake.getLibraryMaskingFormat
	hooks.List.Call = fake.listLibraryMaskingFormats
	hooks.Update.Call = fake.updateLibraryMaskingFormat
	hooks.Delete.Call = fake.deleteLibraryMaskingFormat
	applyLibraryMaskingFormatRuntimeHooks(&hooks)

	manager := &LibraryMaskingFormatServiceManager{}
	delegate := defaultLibraryMaskingFormatServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.LibraryMaskingFormat](
			buildLibraryMaskingFormatGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapLibraryMaskingFormatGeneratedClient(hooks, delegate)
}

func TestLibraryMaskingFormatCreateBuildsPolymorphicBodyAndRecordsStatus(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	fake := &fakeLibraryMaskingFormatOCI{}
	configureLibraryMaskingFormatCreateSuccess(t, fake, resource)

	response, err := newTestLibraryMaskingFormatClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create calls = %d, want 1", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testLibraryMaskingFormatID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testLibraryMaskingFormatID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireLibraryMaskingFormatCondition(t, resource, shared.Active)
}

func TestLibraryMaskingFormatCreateBindsFromAllListPages(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	fake := &fakeLibraryMaskingFormatOCI{}
	configureLibraryMaskingFormatPagedBind(t, fake, resource)

	response, err := newTestLibraryMaskingFormatClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list calls = %d, want 2", len(fake.listRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testLibraryMaskingFormatID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testLibraryMaskingFormatID)
	}
	if got := resource.Status.Id; got != testLibraryMaskingFormatID {
		t.Fatalf("status.id = %q, want %q", got, testLibraryMaskingFormatID)
	}
}

func TestLibraryMaskingFormatNoOpReconcileSkipsUpdate(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLibraryMaskingFormatID)
	fake := &fakeLibraryMaskingFormatOCI{}
	fake.get = func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		return datasafesdk.GetLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive),
		}, nil
	}
	fake.update = func(context.Context, datasafesdk.UpdateLibraryMaskingFormatRequest) (datasafesdk.UpdateLibraryMaskingFormatResponse, error) {
		t.Fatal("UpdateLibraryMaskingFormat called for no-op reconcile")
		return datasafesdk.UpdateLibraryMaskingFormatResponse{}, nil
	}

	response, err := newTestLibraryMaskingFormatClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestLibraryMaskingFormatMutableUpdateUsesReviewedBody(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLibraryMaskingFormatID)
	resource.Spec.DisplayName = "updated-format"
	resource.Spec.FreeformTags = map[string]string{}

	fake := &fakeLibraryMaskingFormatOCI{}
	configureLibraryMaskingFormatMutableUpdate(t, fake, resource)

	response, err := newTestLibraryMaskingFormatClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestLibraryMaskingFormatImmutableCompartmentDriftIsRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLibraryMaskingFormatID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	fake := &fakeLibraryMaskingFormatOCI{}
	fake.get = func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		current := libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive)
		current.CompartmentId = common.String(testLibraryMaskingFormatCompartmentID)
		return datasafesdk.GetLibraryMaskingFormatResponse{LibraryMaskingFormat: current}, nil
	}
	fake.update = func(context.Context, datasafesdk.UpdateLibraryMaskingFormatRequest) (datasafesdk.UpdateLibraryMaskingFormatResponse, error) {
		t.Fatal("UpdateLibraryMaskingFormat called after immutable compartment drift")
		return datasafesdk.UpdateLibraryMaskingFormatResponse{}, nil
	}

	response, err := newTestLibraryMaskingFormatClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId replacement detail", err.Error())
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestLibraryMaskingFormatDeleteRetainsFinalizerUntilDeleteIsConfirmed(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLibraryMaskingFormatID)

	deleteCalled := false
	fake := &fakeLibraryMaskingFormatOCI{}
	fake.get = func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		state := datasafesdk.MaskingLifecycleStateActive
		if deleteCalled {
			state = datasafesdk.MaskingLifecycleStateDeleting
		}
		return datasafesdk.GetLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, state),
		}, nil
	}
	fake.delete = func(context.Context, datasafesdk.DeleteLibraryMaskingFormatRequest) (datasafesdk.DeleteLibraryMaskingFormatResponse, error) {
		deleteCalled = true
		return datasafesdk.DeleteLibraryMaskingFormatResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestLibraryMaskingFormatClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI reports DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete calls = %d, want 1", len(fake.deleteRequests))
	}
	requireLibraryMaskingFormatCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status async current = %#v, want delete pending", resource.Status.OsokStatus.Async.Current)
	}
}

func TestLibraryMaskingFormatDeleteKeepsFinalizerOnAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLibraryMaskingFormatID)

	fake := &fakeLibraryMaskingFormatOCI{}
	fake.get = func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		return datasafesdk.GetLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive),
		}, nil
	}
	fake.delete = func(context.Context, datasafesdk.DeleteLibraryMaskingFormatRequest) (datasafesdk.DeleteLibraryMaskingFormatResponse, error) {
		return datasafesdk.DeleteLibraryMaskingFormatResponse{}, errortest.NewServiceError(
			404,
			errorutil.NotAuthorizedOrNotFound,
			"not authorized or not found",
		)
	}

	deleted, err := newTestLibraryMaskingFormatClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not-found")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous NotAuthorizedOrNotFound detail", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestLibraryMaskingFormatDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLibraryMaskingFormatID)

	fake := &fakeLibraryMaskingFormatOCI{}
	fake.get = func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		return datasafesdk.GetLibraryMaskingFormatResponse{}, errortest.NewServiceError(
			404,
			errorutil.NotAuthorizedOrNotFound,
			"not authorized or not found",
		)
	}
	fake.delete = func(context.Context, datasafesdk.DeleteLibraryMaskingFormatRequest) (datasafesdk.DeleteLibraryMaskingFormatResponse, error) {
		t.Fatal("DeleteLibraryMaskingFormat called after ambiguous pre-delete read")
		return datasafesdk.DeleteLibraryMaskingFormatResponse{}, nil
	}

	deleted, err := newTestLibraryMaskingFormatClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-delete read")
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get calls = %d, want 1", len(fake.getRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete calls = %d, want 0", len(fake.deleteRequests))
	}
	if !strings.Contains(err.Error(), "refusing to call delete") {
		t.Fatalf("Delete() error = %q, want refusing to call delete detail", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestLibraryMaskingFormatRejectsUnsupportedFormatEntryType(t *testing.T) {
	t.Parallel()

	resource := newLibraryMaskingFormatResource()
	resource.Spec.FormatEntries = []datasafev1beta1.LibraryMaskingFormatFormatEntry{{
		Type:        "UNSUPPORTED",
		FixedString: "MASKED",
	}}

	_, err := buildLibraryMaskingFormatCreateBody(resource)
	if err == nil {
		t.Fatal("buildLibraryMaskingFormatCreateBody() error = nil, want unsupported type error")
	}
	if !strings.Contains(err.Error(), `unsupported format entry type "UNSUPPORTED"`) {
		t.Fatalf("buildLibraryMaskingFormatCreateBody() error = %q, want unsupported type detail", err.Error())
	}
}

func configureLibraryMaskingFormatCreateSuccess(
	t *testing.T,
	fake *fakeLibraryMaskingFormatOCI,
	resource *datasafev1beta1.LibraryMaskingFormat,
) {
	t.Helper()
	fake.list = func(context.Context, datasafesdk.ListLibraryMaskingFormatsRequest) (datasafesdk.ListLibraryMaskingFormatsResponse, error) {
		return datasafesdk.ListLibraryMaskingFormatsResponse{}, nil
	}
	fake.create = func(_ context.Context, request datasafesdk.CreateLibraryMaskingFormatRequest) (datasafesdk.CreateLibraryMaskingFormatResponse, error) {
		requireCreateLibraryMaskingFormatRequest(t, request, resource)
		return datasafesdk.CreateLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateCreating),
			OpcRequestId:         common.String("opc-create"),
			OpcWorkRequestId:     common.String("wr-create"),
		}, nil
	}
	fake.get = func(_ context.Context, request datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		requireGetLibraryMaskingFormatRequest(t, request)
		return datasafesdk.GetLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive),
			OpcRequestId:         common.String("opc-get"),
		}, nil
	}
}

func requireCreateLibraryMaskingFormatRequest(
	t *testing.T,
	request datasafesdk.CreateLibraryMaskingFormatRequest,
	resource *datasafev1beta1.LibraryMaskingFormat,
) {
	t.Helper()
	if request.OpcRetryToken == nil || *request.OpcRetryToken == "" {
		t.Fatal("CreateLibraryMaskingFormatRequest.OpcRetryToken is empty, want deterministic retry token")
	}
	if got := stringPointerValue(request.CompartmentId); got != testLibraryMaskingFormatCompartmentID {
		t.Fatalf("create compartmentId = %q, want %q", got, testLibraryMaskingFormatCompartmentID)
	}
	if got := stringPointerValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	requireLibraryMaskingFormatCreateEntry(t, request)
}

func requireLibraryMaskingFormatCreateEntry(t *testing.T, request datasafesdk.CreateLibraryMaskingFormatRequest) {
	t.Helper()
	if len(request.FormatEntries) != 1 {
		t.Fatalf("create formatEntries length = %d, want 1", len(request.FormatEntries))
	}
	entry, ok := request.FormatEntries[0].(datasafesdk.FixedStringFormatEntry)
	if !ok {
		t.Fatalf("create formatEntries[0] type = %T, want datasafe.FixedStringFormatEntry", request.FormatEntries[0])
	}
	if got := stringPointerValue(entry.FixedString); got != "MASKED" {
		t.Fatalf("create fixedString = %q, want MASKED", got)
	}
}

func configureLibraryMaskingFormatPagedBind(
	t *testing.T,
	fake *fakeLibraryMaskingFormatOCI,
	resource *datasafev1beta1.LibraryMaskingFormat,
) {
	t.Helper()
	fake.list = func(_ context.Context, request datasafesdk.ListLibraryMaskingFormatsRequest) (datasafesdk.ListLibraryMaskingFormatsResponse, error) {
		return libraryMaskingFormatPagedListResponse(t, resource, request), nil
	}
	fake.get = func(_ context.Context, request datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		requireGetLibraryMaskingFormatRequest(t, request)
		return datasafesdk.GetLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive),
		}, nil
	}
	fake.create = func(context.Context, datasafesdk.CreateLibraryMaskingFormatRequest) (datasafesdk.CreateLibraryMaskingFormatResponse, error) {
		t.Fatal("CreateLibraryMaskingFormat called, want bind to existing list item")
		return datasafesdk.CreateLibraryMaskingFormatResponse{}, nil
	}
}

func libraryMaskingFormatPagedListResponse(
	t *testing.T,
	resource *datasafev1beta1.LibraryMaskingFormat,
	request datasafesdk.ListLibraryMaskingFormatsRequest,
) datasafesdk.ListLibraryMaskingFormatsResponse {
	t.Helper()
	requireLibraryMaskingFormatListRequest(t, request, resource)
	if request.Page == nil {
		return datasafesdk.ListLibraryMaskingFormatsResponse{
			OpcNextPage: common.String("page-2"),
			LibraryMaskingFormatCollection: datasafesdk.LibraryMaskingFormatCollection{
				Items: []datasafesdk.LibraryMaskingFormatSummary{},
			},
		}
	}
	requireSecondLibraryMaskingFormatListPage(t, request)
	return datasafesdk.ListLibraryMaskingFormatsResponse{
		LibraryMaskingFormatCollection: datasafesdk.LibraryMaskingFormatCollection{
			Items: []datasafesdk.LibraryMaskingFormatSummary{
				libraryMaskingFormatSummary(resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive),
			},
		},
	}
}

func requireLibraryMaskingFormatListRequest(
	t *testing.T,
	request datasafesdk.ListLibraryMaskingFormatsRequest,
	resource *datasafev1beta1.LibraryMaskingFormat,
) {
	t.Helper()
	if request.LibraryMaskingFormatSource != datasafesdk.ListLibraryMaskingFormatsLibraryMaskingFormatSourceUser {
		t.Fatalf("list source = %q, want USER", request.LibraryMaskingFormatSource)
	}
	if got := stringPointerValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("list displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
}

func requireSecondLibraryMaskingFormatListPage(t *testing.T, request datasafesdk.ListLibraryMaskingFormatsRequest) {
	t.Helper()
	if got := *request.Page; got != "page-2" {
		t.Fatalf("list second page token = %q, want page-2", got)
	}
}

func configureLibraryMaskingFormatMutableUpdate(
	t *testing.T,
	fake *fakeLibraryMaskingFormatOCI,
	resource *datasafev1beta1.LibraryMaskingFormat,
) {
	t.Helper()
	getCalls := 0
	fake.get = func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error) {
		getCalls++
		return datasafesdk.GetLibraryMaskingFormatResponse{
			LibraryMaskingFormat: libraryMaskingFormatBodyForMutableUpdate(t, resource, getCalls),
		}, nil
	}
	fake.update = func(_ context.Context, request datasafesdk.UpdateLibraryMaskingFormatRequest) (datasafesdk.UpdateLibraryMaskingFormatResponse, error) {
		requireLibraryMaskingFormatUpdateRequest(t, request, resource)
		return datasafesdk.UpdateLibraryMaskingFormatResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
}

func libraryMaskingFormatBodyForMutableUpdate(
	t *testing.T,
	resource *datasafev1beta1.LibraryMaskingFormat,
	getCalls int,
) datasafesdk.LibraryMaskingFormat {
	t.Helper()
	current := libraryMaskingFormatBody(t, resource, testLibraryMaskingFormatID, datasafesdk.MaskingLifecycleStateActive)
	if getCalls == 1 {
		current.DisplayName = common.String("old-format")
		current.FreeformTags = map[string]string{"owner": "old"}
		return current
	}
	current.DisplayName = common.String(resource.Spec.DisplayName)
	current.FreeformTags = map[string]string{}
	return current
}

func requireLibraryMaskingFormatUpdateRequest(
	t *testing.T,
	request datasafesdk.UpdateLibraryMaskingFormatRequest,
	resource *datasafev1beta1.LibraryMaskingFormat,
) {
	t.Helper()
	if got := stringPointerValue(request.LibraryMaskingFormatId); got != testLibraryMaskingFormatID {
		t.Fatalf("update libraryMaskingFormatId = %q, want %q", got, testLibraryMaskingFormatID)
	}
	if got := stringPointerValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("update displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if request.FreeformTags == nil || len(request.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map", request.FreeformTags)
	}
}

func requireGetLibraryMaskingFormatRequest(t *testing.T, request datasafesdk.GetLibraryMaskingFormatRequest) {
	t.Helper()
	if got := stringPointerValue(request.LibraryMaskingFormatId); got != testLibraryMaskingFormatID {
		t.Fatalf("get libraryMaskingFormatId = %q, want %q", got, testLibraryMaskingFormatID)
	}
}

func newLibraryMaskingFormatResource() *datasafev1beta1.LibraryMaskingFormat {
	return &datasafev1beta1.LibraryMaskingFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "library-format",
			Namespace: "default",
		},
		Spec: datasafev1beta1.LibraryMaskingFormatSpec{
			CompartmentId: testLibraryMaskingFormatCompartmentID,
			DisplayName:   "library-format",
			Description:   "format description",
			FormatEntries: []datasafev1beta1.LibraryMaskingFormatFormatEntry{{
				Type:        string(datasafesdk.FormatEntryTypeFixedString),
				FixedString: "MASKED",
				Description: "fixed string",
			}},
			SensitiveTypeIds: []string{"ocid1.sensitivetype.oc1..example"},
			FreeformTags:     map[string]string{"owner": "datasafe"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func libraryMaskingFormatBody(
	t *testing.T,
	resource *datasafev1beta1.LibraryMaskingFormat,
	id string,
	state datasafesdk.MaskingLifecycleStateEnum,
) datasafesdk.LibraryMaskingFormat {
	t.Helper()
	entries := mustLibraryMaskingFormatEntriesForTest(t, resource.Spec.FormatEntries)
	return datasafesdk.LibraryMaskingFormat{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      common.String(resource.Spec.DisplayName),
		LifecycleState:   state,
		Source:           datasafesdk.LibraryMaskingFormatSourceUser,
		Description:      common.String(resource.Spec.Description),
		SensitiveTypeIds: cloneLibraryMaskingFormatStringSlice(resource.Spec.SensitiveTypeIds),
		FormatEntries:    entries,
		FreeformTags:     cloneLibraryMaskingFormatStringMap(resource.Spec.FreeformTags),
		DefinedTags:      libraryMaskingFormatDefinedTags(resource.Spec.DefinedTags),
	}
}

func libraryMaskingFormatSummary(
	resource *datasafev1beta1.LibraryMaskingFormat,
	id string,
	state datasafesdk.MaskingLifecycleStateEnum,
) datasafesdk.LibraryMaskingFormatSummary {
	return datasafesdk.LibraryMaskingFormatSummary{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      common.String(resource.Spec.DisplayName),
		LifecycleState:   state,
		Source:           datasafesdk.LibraryMaskingFormatSourceUser,
		Description:      common.String(resource.Spec.Description),
		SensitiveTypeIds: cloneLibraryMaskingFormatStringSlice(resource.Spec.SensitiveTypeIds),
		FreeformTags:     cloneLibraryMaskingFormatStringMap(resource.Spec.FreeformTags),
		DefinedTags:      libraryMaskingFormatDefinedTags(resource.Spec.DefinedTags),
	}
}

func mustLibraryMaskingFormatEntriesForTest(
	t *testing.T,
	entries []datasafev1beta1.LibraryMaskingFormatFormatEntry,
) []datasafesdk.FormatEntry {
	t.Helper()
	converted, err := libraryMaskingFormatEntriesForOCI(entries)
	if err != nil {
		t.Fatalf("libraryMaskingFormatEntriesForOCI() error = %v", err)
	}
	return converted
}

func requireLibraryMaskingFormatCondition(
	t *testing.T,
	resource *datasafev1beta1.LibraryMaskingFormat,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status conditions are empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition type = %s, want %s", got, want)
	}
}
