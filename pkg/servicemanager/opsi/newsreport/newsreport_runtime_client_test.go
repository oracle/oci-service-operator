/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package newsreport

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type erroringNewsReportConfigProvider struct {
	calls int
}

func (p *erroringNewsReportConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	p.calls++
	return nil, errors.New("newsreport provider invalid")
}

func (p *erroringNewsReportConfigProvider) KeyID() (string, error) {
	p.calls++
	return "", errors.New("newsreport provider invalid")
}

func (p *erroringNewsReportConfigProvider) TenancyOCID() (string, error) {
	p.calls++
	return "", errors.New("newsreport provider invalid")
}

func (p *erroringNewsReportConfigProvider) UserOCID() (string, error) {
	p.calls++
	return "", errors.New("newsreport provider invalid")
}

func (p *erroringNewsReportConfigProvider) KeyFingerprint() (string, error) {
	p.calls++
	return "", errors.New("newsreport provider invalid")
}

func (p *erroringNewsReportConfigProvider) Region() (string, error) {
	p.calls++
	return "", errors.New("newsreport provider invalid")
}

func (p *erroringNewsReportConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{}, nil
}

func TestNewsReportRuntimeHooksConfigureReviewedSemantics(t *testing.T) {
	hooks := newNewsReportRuntimeHooksWithOCIClient(&fakeNewsReportOCIClient{})
	applyNewsReportRuntimeHooks(&NewsReportServiceManager{}, &hooks, &fakeNewsReportOCIClient{}, nil)

	assertNewsReportRuntimeSemantics(t, hooks)
	assertNewsReportRuntimeHooks(t, hooks)
	assertNewsReportNoOpUpdateBody(t, hooks)
	assertNewsReportCreateBody(t, hooks)
}

func assertNewsReportRuntimeSemantics(t *testing.T, hooks NewsReportRuntimeHooks) {
	t.Helper()
	if hooks.Semantics == nil {
		t.Fatal("Semantics = nil, want reviewed generatedruntime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got)
	}
	assertNewsReportStringSliceContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "status", "newsFrequency", "contentTypes", "onsTopicId", "areChildCompartmentsIncluded")
	assertNewsReportStringSliceContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId")
}

func assertNewsReportRuntimeHooks(t *testing.T, hooks NewsReportRuntimeHooks) {
	t.Helper()
	if hooks.StatusHooks.ProjectStatus == nil {
		t.Fatal("StatusHooks.ProjectStatus = nil, want sdkStatus collision-safe projection")
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

func assertNewsReportNoOpUpdateBody(t *testing.T, hooks NewsReportRuntimeHooks) {
	t.Helper()
	body, updateNeeded, err := hooks.BuildUpdateBody(context.Background(), newsReportResource(), "default", opsisdk.GetNewsReportResponse{
		NewsReport: newsReportSDK("news-1", opsisdk.LifecycleStateActive),
	})
	if err != nil {
		t.Fatalf("BuildUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("BuildUpdateBody() updateNeeded = true with matching current state; body = %#v", body)
	}
}

func assertNewsReportCreateBody(t *testing.T, hooks NewsReportRuntimeHooks) {
	t.Helper()
	createBody, err := hooks.BuildCreateBody(context.Background(), newsReportResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	createDetails, ok := createBody.(opsisdk.CreateNewsReportDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want opsi.CreateNewsReportDetails", createBody)
	}
	if got := newsReportStringValue(createDetails.Name); got != "weekly-news" {
		t.Fatalf("Create body name = %q, want weekly-news", got)
	}
	if createDetails.AreChildCompartmentsIncluded == nil || *createDetails.AreChildCompartmentsIncluded {
		t.Fatalf("Create body areChildCompartmentsIncluded = %v, want explicit false", createDetails.AreChildCompartmentsIncluded)
	}
}

func TestNewsReportCreateOrUpdateCreatesAndTracksWorkRequest(t *testing.T) {
	resource := newsReportResource()
	fake := &fakeNewsReportOCIClient{}
	fake.list = func(_ context.Context, _ opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error) {
		return opsisdk.ListNewsReportsResponse{}, nil
	}
	fake.create = func(_ context.Context, request opsisdk.CreateNewsReportRequest) (opsisdk.CreateNewsReportResponse, error) {
		assertNewsReportCreateRequest(t, request, resource)
		return opsisdk.CreateNewsReportResponse{
			NewsReport:       newsReportSDK("news-1", opsisdk.LifecycleStateCreating),
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
			ContentLocation:  common.String("/newsReports/news-1"),
			Location:         common.String("/newsReports/news-1"),
			RawResponse:      nil,
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		if got := newsReportStringValue(request.WorkRequestId); got != "wr-create" {
			t.Fatalf("GetWorkRequest id = %q, want wr-create", got)
		}
		return opsisdk.GetWorkRequestResponse{WorkRequest: newsReportWorkRequest("wr-create", opsisdk.OperationTypeCreateNewsReport, opsisdk.OperationStatusInProgress, "news-1")}, nil
	}

	response, err := newNewsReportServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue for pending work request", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("Create calls = %d, want 1", len(fake.createRequests))
	}
	assertNewsReportAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, opsisdk.ResourceStatusEnabled)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", got)
	}
}

func assertNewsReportCreateRequest(
	t *testing.T,
	request opsisdk.CreateNewsReportRequest,
	resource *opsiv1beta1.NewsReport,
) {
	t.Helper()
	if got := newsReportStringValue(request.Name); got != resource.Spec.Name {
		t.Fatalf("Create request name = %q, want %q", got, resource.Spec.Name)
	}
	if got := request.NewsFrequency; got != opsisdk.NewsFrequencyWeekly {
		t.Fatalf("Create request newsFrequency = %q, want %q", got, opsisdk.NewsFrequencyWeekly)
	}
	if request.AreChildCompartmentsIncluded == nil || *request.AreChildCompartmentsIncluded {
		t.Fatalf("Create request areChildCompartmentsIncluded = %v, want explicit false", request.AreChildCompartmentsIncluded)
	}
}

func TestNewsReportCreateOrUpdateBindsFromPaginatedListWithoutCreate(t *testing.T) {
	resource := newsReportResource()
	fake := &fakeNewsReportOCIClient{}
	fake.list = func(_ context.Context, request opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error) {
		switch page := newsReportStringValue(request.Page); page {
		case "":
			return opsisdk.ListNewsReportsResponse{
				NewsReportCollection: opsisdk.NewsReportCollection{
					Items: []opsisdk.NewsReportSummary{newsReportSummary("other", "other-news", opsisdk.LifecycleStateActive)},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return opsisdk.ListNewsReportsResponse{
				NewsReportCollection: opsisdk.NewsReportCollection{
					Items: []opsisdk.NewsReportSummary{newsReportSummary("news-1", resource.Spec.Name, opsisdk.LifecycleStateActive)},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", page)
			return opsisdk.ListNewsReportsResponse{}, nil
		}
	}
	fake.get = func(_ context.Context, request opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		if got := newsReportStringValue(request.NewsReportId); got != "news-1" {
			t.Fatalf("Get id = %q, want news-1", got)
		}
		return opsisdk.GetNewsReportResponse{NewsReport: newsReportSDK("news-1", opsisdk.LifecycleStateActive)}, nil
	}

	response, err := newNewsReportServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID("news-1") {
		t.Fatalf("status.ocid = %q, want news-1", got)
	}
}

func TestNewsReportCreateOrUpdateNoOpDoesNotUpdate(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	fake := &fakeNewsReportOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		return opsisdk.GetNewsReportResponse{NewsReport: newsReportSDK("news-1", opsisdk.LifecycleStateActive)}, nil
	}

	response, err := newNewsReportServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active no-op without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, opsisdk.ResourceStatusEnabled)
	}
}

func TestNewsReportCreateOrUpdateMutableUpdateUsesUpdatePath(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	resource.Spec.Status = string(opsisdk.ResourceStatusDisabled)
	resource.Spec.Description = "updated description"
	resource.Spec.AreChildCompartmentsIncluded = true
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	getResponses := []opsisdk.NewsReport{
		newsReportWithMutableState(opsisdk.ResourceStatusEnabled, "weekly report", false, map[string]string{"env": "dev"}),
		newsReportWithMutableState(opsisdk.ResourceStatusDisabled, "updated description", true, map[string]string{"env": "prod"}),
	}
	fake := &fakeNewsReportOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		current := getResponses[newsReportMin(len(fake.getRequests)-1, len(getResponses)-1)]
		return opsisdk.GetNewsReportResponse{NewsReport: current}, nil
	}
	fake.update = func(_ context.Context, request opsisdk.UpdateNewsReportRequest) (opsisdk.UpdateNewsReportResponse, error) {
		assertNewsReportMutableUpdateRequest(t, request)
		return opsisdk.UpdateNewsReportResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		workRequest := newsReportWorkRequest("wr-update", opsisdk.OperationTypeUpdateNewsReport, opsisdk.OperationStatusSucceeded, "news-1")
		return opsisdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}

	response, err := newNewsReportServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want completed update without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusDisabled) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, opsisdk.ResourceStatusDisabled)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
}

func assertNewsReportMutableUpdateRequest(t *testing.T, request opsisdk.UpdateNewsReportRequest) {
	t.Helper()
	if got := newsReportStringValue(request.NewsReportId); got != "news-1" {
		t.Fatalf("Update id = %q, want news-1", got)
	}
	if got := request.Status; got != opsisdk.ResourceStatusDisabled {
		t.Fatalf("Update status = %q, want %q", got, opsisdk.ResourceStatusDisabled)
	}
	if got := newsReportStringValue(request.Description); got != "updated description" {
		t.Fatalf("Update description = %q, want updated description", got)
	}
	if request.AreChildCompartmentsIncluded == nil || !*request.AreChildCompartmentsIncluded {
		t.Fatalf("Update areChildCompartmentsIncluded = %v, want true", request.AreChildCompartmentsIncluded)
	}
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("Update freeformTags[env] = %q, want prod", got)
	}
}

func TestNewsReportCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	resource.Spec.CompartmentId = "compartment-new"
	resource.Spec.Description = "updated description"
	fake := &fakeNewsReportOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		return opsisdk.GetNewsReportResponse{NewsReport: newsReportSDK("news-1", opsisdk.LifecycleStateActive)}, nil
	}

	_, err := newNewsReportServiceClientWithOCIClient(logger(), fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId create-only drift", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestNewsReportDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeNewsReportOCIClient{}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{WorkRequest: newsReportWorkRequest("wr-create", opsisdk.OperationTypeCreateNewsReport, opsisdk.OperationStatusInProgress, "news-1")}, nil
	}

	deleted, err := newNewsReportServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while create work request is pending")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("Delete OCI calls = %d, want 0", len(fake.deleteRequests))
	}
	assertNewsReportAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
}

func TestNewsReportDeletePreservesGeneratedOCIInitErrorBeforePreflight(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	provider := &erroringNewsReportConfigProvider{}
	manager := &NewsReportServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
	client := newNewsReportServiceClient(manager)
	callsAfterInit := provider.calls

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want OCI client initialization error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "initialize NewsReport OCI client") {
		t.Fatalf("Delete() error = %v, want NewsReport OCI client initialization failure", err)
	}
	if !strings.Contains(err.Error(), "newsreport provider invalid") {
		t.Fatalf("Delete() error = %v, want provider failure detail", err)
	}
	if provider.calls != callsAfterInit {
		t.Fatalf("provider calls after Delete() = %d, want %d; delete preflight should not run before InitError", provider.calls, callsAfterInit)
	}
}

func TestNewsReportDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	fake := &fakeNewsReportOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		return opsisdk.GetNewsReportResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newNewsReportServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
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

func TestNewsReportDeleteStartsWorkRequestAndRetainsFinalizer(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	fake := &fakeNewsReportOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		return opsisdk.GetNewsReportResponse{NewsReport: newsReportSDK("news-1", opsisdk.LifecycleStateActive)}, nil
	}
	fake.delete = func(_ context.Context, _ opsisdk.DeleteNewsReportRequest) (opsisdk.DeleteNewsReportResponse, error) {
		return opsisdk.DeleteNewsReportResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}

	deleted, err := newNewsReportServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for delete work request")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("Delete OCI calls = %d, want 1", len(fake.deleteRequests))
	}
	assertNewsReportAsync(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
}

func TestNewsReportSucceededDeleteWorkRequestKeepsAuthShapedReadFatal(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeNewsReportOCIClient{}
	fake.getWorkRequest = func(_ context.Context, _ opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{WorkRequest: newsReportWorkRequest("wr-delete", opsisdk.OperationTypeDeleteNewsReport, opsisdk.OperationStatusSucceeded, "news-1")}, nil
	}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		return opsisdk.GetNewsReportResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newNewsReportServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
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

func TestNewsReportDeleteReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	resource := newsReportResource()
	resource.Status.OsokStatus.Ocid = "news-1"
	fake := &fakeNewsReportOCIClient{}
	fake.get = func(_ context.Context, _ opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
		return opsisdk.GetNewsReportResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}

	deleted, err := newNewsReportServiceClientWithOCIClient(logger(), fake).Delete(context.Background(), resource)
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

type fakeNewsReportOCIClient struct {
	createRequests         []opsisdk.CreateNewsReportRequest
	getRequests            []opsisdk.GetNewsReportRequest
	listRequests           []opsisdk.ListNewsReportsRequest
	updateRequests         []opsisdk.UpdateNewsReportRequest
	deleteRequests         []opsisdk.DeleteNewsReportRequest
	getWorkRequestRequests []opsisdk.GetWorkRequestRequest

	create         func(context.Context, opsisdk.CreateNewsReportRequest) (opsisdk.CreateNewsReportResponse, error)
	get            func(context.Context, opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error)
	list           func(context.Context, opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error)
	update         func(context.Context, opsisdk.UpdateNewsReportRequest) (opsisdk.UpdateNewsReportResponse, error)
	delete         func(context.Context, opsisdk.DeleteNewsReportRequest) (opsisdk.DeleteNewsReportResponse, error)
	getWorkRequest func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeNewsReportOCIClient) CreateNewsReport(
	ctx context.Context,
	request opsisdk.CreateNewsReportRequest,
) (opsisdk.CreateNewsReportResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create != nil {
		return f.create(ctx, request)
	}
	return opsisdk.CreateNewsReportResponse{}, nil
}

func (f *fakeNewsReportOCIClient) GetNewsReport(
	ctx context.Context,
	request opsisdk.GetNewsReportRequest,
) (opsisdk.GetNewsReportResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get != nil {
		return f.get(ctx, request)
	}
	return opsisdk.GetNewsReportResponse{}, nil
}

func (f *fakeNewsReportOCIClient) ListNewsReports(
	ctx context.Context,
	request opsisdk.ListNewsReportsRequest,
) (opsisdk.ListNewsReportsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list != nil {
		return f.list(ctx, request)
	}
	return opsisdk.ListNewsReportsResponse{}, nil
}

func (f *fakeNewsReportOCIClient) UpdateNewsReport(
	ctx context.Context,
	request opsisdk.UpdateNewsReportRequest,
) (opsisdk.UpdateNewsReportResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update != nil {
		return f.update(ctx, request)
	}
	return opsisdk.UpdateNewsReportResponse{}, nil
}

func (f *fakeNewsReportOCIClient) DeleteNewsReport(
	ctx context.Context,
	request opsisdk.DeleteNewsReportRequest,
) (opsisdk.DeleteNewsReportResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete != nil {
		return f.delete(ctx, request)
	}
	return opsisdk.DeleteNewsReportResponse{}, nil
}

func (f *fakeNewsReportOCIClient) GetWorkRequest(
	ctx context.Context,
	request opsisdk.GetWorkRequestRequest,
) (opsisdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequest != nil {
		return f.getWorkRequest(ctx, request)
	}
	return opsisdk.GetWorkRequestResponse{}, nil
}

func newsReportResource() *opsiv1beta1.NewsReport {
	return &opsiv1beta1.NewsReport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "weekly-news",
			Namespace: "default",
			UID:       types.UID("newsreport-uid"),
		},
		Spec: opsiv1beta1.NewsReportSpec{
			Name:          "weekly-news",
			NewsFrequency: string(opsisdk.NewsFrequencyWeekly),
			Description:   "weekly report",
			OnsTopicId:    "ons-topic-1",
			CompartmentId: "compartment-1",
			ContentTypes: opsiv1beta1.NewsReportContentTypes{
				CapacityPlanningResources: []string{string(opsisdk.NewsContentTypesResourceDatabase)},
				ActionableInsightsResources: []string{
					string(opsisdk.ActionableInsightsContentTypesResourceNewHighs),
				},
			},
			Locale:                       string(opsisdk.NewsLocaleEn),
			FreeformTags:                 map[string]string{"env": "dev"},
			DefinedTags:                  map[string]shared.MapValue{"ns": {"key": "value"}},
			Status:                       string(opsisdk.ResourceStatusEnabled),
			DayOfWeek:                    string(opsisdk.DayOfWeekMonday),
			AreChildCompartmentsIncluded: false,
			TagFilters:                   []string{"department=finance"},
			MatchRule:                    string(opsisdk.MatchRuleMatchAny),
		},
	}
}

func newsReportSDK(id string, state opsisdk.LifecycleStateEnum) opsisdk.NewsReport {
	return opsisdk.NewsReport{
		NewsFrequency: opsisdk.NewsFrequencyWeekly,
		ContentTypes: &opsisdk.NewsContentTypes{
			CapacityPlanningResources: []opsisdk.NewsContentTypesResourceEnum{opsisdk.NewsContentTypesResourceDatabase},
			ActionableInsightsResources: []opsisdk.ActionableInsightsContentTypesResourceEnum{
				opsisdk.ActionableInsightsContentTypesResourceNewHighs,
			},
		},
		Id:                           common.String(id),
		CompartmentId:                common.String("compartment-1"),
		OnsTopicId:                   common.String("ons-topic-1"),
		Locale:                       opsisdk.NewsLocaleEn,
		Description:                  common.String("weekly report"),
		Name:                         common.String("weekly-news"),
		FreeformTags:                 map[string]string{"env": "dev"},
		DefinedTags:                  map[string]map[string]interface{}{"ns": {"key": "value"}},
		Status:                       opsisdk.ResourceStatusEnabled,
		LifecycleState:               state,
		DayOfWeek:                    opsisdk.DayOfWeekMonday,
		AreChildCompartmentsIncluded: common.Bool(false),
		TagFilters:                   []string{"department=finance"},
		MatchRule:                    opsisdk.MatchRuleMatchAny,
	}
}

func newsReportWithMutableState(
	status opsisdk.ResourceStatusEnum,
	description string,
	childCompartments bool,
	freeformTags map[string]string,
) opsisdk.NewsReport {
	current := newsReportSDK("news-1", opsisdk.LifecycleStateActive)
	current.Status = status
	current.Description = common.String(description)
	current.AreChildCompartmentsIncluded = common.Bool(childCompartments)
	current.FreeformTags = freeformTags
	return current
}

func newsReportSummary(id string, name string, state opsisdk.LifecycleStateEnum) opsisdk.NewsReportSummary {
	current := newsReportSDK(id, state)
	current.Name = common.String(name)
	return opsisdk.NewsReportSummary{
		NewsFrequency:                current.NewsFrequency,
		ContentTypes:                 current.ContentTypes,
		Id:                           current.Id,
		CompartmentId:                current.CompartmentId,
		Locale:                       current.Locale,
		Description:                  current.Description,
		Name:                         current.Name,
		OnsTopicId:                   current.OnsTopicId,
		FreeformTags:                 current.FreeformTags,
		DefinedTags:                  current.DefinedTags,
		Status:                       current.Status,
		LifecycleState:               current.LifecycleState,
		DayOfWeek:                    current.DayOfWeek,
		AreChildCompartmentsIncluded: current.AreChildCompartmentsIncluded,
		TagFilters:                   current.TagFilters,
		MatchRule:                    current.MatchRule,
	}
}

func newsReportWorkRequest(
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
			EntityType: common.String("NewsReport"),
			ActionType: newsReportActionForOperation(operation, status),
			Identifier: common.String(resourceID),
			EntityUri:  common.String("/newsReports/" + resourceID),
		}},
	}
}

func newsReportActionForOperation(operation opsisdk.OperationTypeEnum, status opsisdk.OperationStatusEnum) opsisdk.ActionTypeEnum {
	if status == opsisdk.OperationStatusInProgress || status == opsisdk.OperationStatusAccepted || status == opsisdk.OperationStatusWaiting {
		return opsisdk.ActionTypeInProgress
	}
	switch operation {
	case opsisdk.OperationTypeCreateNewsReport:
		return opsisdk.ActionTypeCreated
	case opsisdk.OperationTypeUpdateNewsReport, opsisdk.OperationTypeEnableNewsReport, opsisdk.OperationTypeDisableNewsReport:
		return opsisdk.ActionTypeUpdated
	case opsisdk.OperationTypeDeleteNewsReport:
		return opsisdk.ActionTypeDeleted
	default:
		return opsisdk.ActionTypeRelated
	}
}

func assertNewsReportAsync(
	t *testing.T,
	resource *opsiv1beta1.NewsReport,
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

func assertNewsReportStringSliceContainsAll(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	for _, value := range want {
		if !slicesContains(got, value) {
			t.Fatalf("%s = %#v, missing %q", name, got, value)
		}
	}
}

func slicesContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func newsReportMin(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func logger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
