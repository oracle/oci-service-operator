/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsembridge

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLogAnalyticsEmBridgeNamespace       = "logan-namespace"
	testLogAnalyticsEmBridgeKubeNamespace   = "kube-namespace"
	testLogAnalyticsEmBridgeTenancyID       = "ocid1.tenancy.oc1..logan"
	testLogAnalyticsEmBridgeID              = "ocid1.loganalyticsembridge.oc1..bridge"
	testLogAnalyticsEmBridgeOtherID         = "ocid1.loganalyticsembridge.oc1..other"
	testLogAnalyticsEmBridgeCompartmentID   = "ocid1.compartment.oc1..bridge"
	testLogAnalyticsEmBridgeEntitiesCompID  = "ocid1.compartment.oc1..entities"
	testLogAnalyticsEmBridgeDisplayName     = "em-bridge-sample"
	testLogAnalyticsEmBridgeDescription     = "desired bridge"
	testLogAnalyticsEmBridgeBucketName      = "em-bridge-bucket"
	testLogAnalyticsEmBridgeUpdatedBucket   = "em-bridge-updated-bucket"
	testLogAnalyticsEmBridgeUpdatedDescribe = "updated bridge"
)

type fakeLogAnalyticsEmBridgeOCIClient struct {
	createFunc    func(context.Context, loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error)
	getFunc       func(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error)
	listFunc      func(context.Context, loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error)
	updateFunc    func(context.Context, loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest) (loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse, error)
	deleteFunc    func(context.Context, loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error)
	namespaceFunc func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)

	createRequests    []loganalyticssdk.CreateLogAnalyticsEmBridgeRequest
	getRequests       []loganalyticssdk.GetLogAnalyticsEmBridgeRequest
	listRequests      []loganalyticssdk.ListLogAnalyticsEmBridgesRequest
	updateRequests    []loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest
	deleteRequests    []loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest
	namespaceRequests []loganalyticssdk.ListNamespacesRequest
}

func (f *fakeLogAnalyticsEmBridgeOCIClient) CreateLogAnalyticsEmBridge(
	ctx context.Context,
	req loganalyticssdk.CreateLogAnalyticsEmBridgeRequest,
) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFunc != nil {
		return f.createFunc(ctx, req)
	}
	return loganalyticssdk.CreateLogAnalyticsEmBridgeResponse{}, nil
}

func (f *fakeLogAnalyticsEmBridgeOCIClient) GetLogAnalyticsEmBridge(
	ctx context.Context,
	req loganalyticssdk.GetLogAnalyticsEmBridgeRequest,
) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFunc != nil {
		return f.getFunc(ctx, req)
	}
	return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{}, nil
}

func (f *fakeLogAnalyticsEmBridgeOCIClient) ListLogAnalyticsEmBridges(
	ctx context.Context,
	req loganalyticssdk.ListLogAnalyticsEmBridgesRequest,
) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFunc != nil {
		return f.listFunc(ctx, req)
	}
	return loganalyticssdk.ListLogAnalyticsEmBridgesResponse{}, nil
}

func (f *fakeLogAnalyticsEmBridgeOCIClient) UpdateLogAnalyticsEmBridge(
	ctx context.Context,
	req loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest,
) (loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, req)
	}
	return loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse{}, nil
}

func (f *fakeLogAnalyticsEmBridgeOCIClient) DeleteLogAnalyticsEmBridge(
	ctx context.Context,
	req loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest,
) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, req)
	}
	return loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse{}, nil
}

func (f *fakeLogAnalyticsEmBridgeOCIClient) ListNamespaces(
	ctx context.Context,
	req loganalyticssdk.ListNamespacesRequest,
) (loganalyticssdk.ListNamespacesResponse, error) {
	f.namespaceRequests = append(f.namespaceRequests, req)
	if f.namespaceFunc != nil {
		return f.namespaceFunc(ctx, req)
	}
	return loganalyticssdk.ListNamespacesResponse{
		NamespaceCollection: loganalyticssdk.NamespaceCollection{
			Items: []loganalyticssdk.NamespaceSummary{{
				NamespaceName:  common.String(testLogAnalyticsEmBridgeNamespace),
				CompartmentId:  common.String(testLogAnalyticsEmBridgeTenancyID),
				IsOnboarded:    common.Bool(true),
				LifecycleState: loganalyticssdk.NamespaceSummaryLifecycleStateActive,
			}},
		},
	}, nil
}

func TestLogAnalyticsEmBridgeRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := newLogAnalyticsEmBridgeRuntimeHooksWithOCIClient(&fakeLogAnalyticsEmBridgeOCIClient{})
	applyLogAnalyticsEmBridgeRuntimeHooks(&hooks)

	assertLogAnalyticsEmBridgeCoreSemantics(t, hooks.Semantics)
	assertLogAnalyticsEmBridgeRuntimeHooks(t, hooks)
}

func TestLogAnalyticsEmBridgeNamespaceResolverUsesTenancyNamespaceNotMetadataNamespace(t *testing.T) {
	t.Parallel()

	resource := newLogAnalyticsEmBridgeResource()
	if resource.Namespace == testLogAnalyticsEmBridgeNamespace {
		t.Fatalf("test resource metadata.namespace = %q, want a different Kubernetes namespace", resource.Namespace)
	}
	fake := newLogAnalyticsEmBridgeCreateThenNoOpFake(t, resource)

	requireLogAnalyticsEmBridgeCreateOrUpdateSuccess(t, newTestLogAnalyticsEmBridgeClient(fake), resource, "active create")

	if got := len(fake.namespaceRequests); got != 1 {
		t.Fatalf("ListNamespaces calls = %d, want 1", got)
	}
	requireLogAnalyticsEmBridgeStringPtr(t, "list namespaces compartmentId", fake.namespaceRequests[0].CompartmentId, testLogAnalyticsEmBridgeTenancyID)
}

func assertLogAnalyticsEmBridgeCoreSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed LogAnalyticsEmBridge semantics")
	}
	if got := semantics.FormalService; got != "loganalytics" {
		t.Fatalf("FormalService = %q, want loganalytics", got)
	}
	if got := semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %#v, want lifecycle async semantics", semantics.Async)
	}
	if semantics.List == nil {
		t.Fatal("List semantics = nil, want create-or-bind list matching")
	}
	assertLogAnalyticsEmBridgeContainsAll(t, semantics.List.MatchFields, "compartmentId", "displayName", "id")
	assertLogAnalyticsEmBridgeContainsAll(t, semantics.Mutation.Mutable, "displayName", "description", "bucketName", "freeformTags", "definedTags")
	assertLogAnalyticsEmBridgeContainsAll(t, semantics.Mutation.ForceNew, "compartmentId", "emEntitiesCompartmentId")
}

func assertLogAnalyticsEmBridgeRuntimeHooks(t *testing.T, hooks LogAnalyticsEmBridgeRuntimeHooks) {
	t.Helper()
	if hooks.BuildCreateBody == nil {
		t.Fatal("BuildCreateBody = nil, want typed create request shaping")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("BuildUpdateBody = nil, want typed update request shaping")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want delete confirmation wrapper", len(hooks.WrapGeneratedClient))
	}
}

func TestLogAnalyticsEmBridgeCreateOrUpdateCreatesThenNoOps(t *testing.T) {
	t.Parallel()

	resource := newLogAnalyticsEmBridgeResource()
	fake := newLogAnalyticsEmBridgeCreateThenNoOpFake(t, resource)
	client := newTestLogAnalyticsEmBridgeClient(fake)
	requireLogAnalyticsEmBridgeCreateOrUpdateSuccess(t, client, resource, "active create")
	assertLogAnalyticsEmBridgeCreateResult(t, resource)

	response := requireLogAnalyticsEmBridgeCreateOrUpdateSuccess(t, client, resource, "active no-op")
	assertLogAnalyticsEmBridgeNoOpResult(t, fake, response)
}

func newLogAnalyticsEmBridgeCreateThenNoOpFake(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) *fakeLogAnalyticsEmBridgeOCIClient {
	t.Helper()
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.listFunc = func(_ context.Context, request loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
		assertLogAnalyticsEmBridgeListRequest(t, request)
		return loganalyticssdk.ListLogAnalyticsEmBridgesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error) {
		assertLogAnalyticsEmBridgeCreateRequest(t, resource, request)
		return loganalyticssdk.CreateLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeBucketName, loganalyticssdk.EmBridgeLifecycleStatesActive),
			OpcRequestId:         common.String("opc-create"),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		assertLogAnalyticsEmBridgeGetRequest(t, request, testLogAnalyticsEmBridgeID)
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeBucketName, loganalyticssdk.EmBridgeLifecycleStatesActive),
		}, nil
	}
	return fake
}

func requireLogAnalyticsEmBridgeCreateOrUpdateSuccess(
	t *testing.T,
	client LogAnalyticsEmBridgeServiceClient,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	want string,
) servicemanager.OSOKResponse {
	t.Helper()
	response, err := client.CreateOrUpdate(context.Background(), resource, logAnalyticsEmBridgeReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want %s", response, want)
	}
	return response
}

func assertLogAnalyticsEmBridgeCreateResult(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) {
	t.Helper()
	if got, want := resource.Status.Id, testLogAnalyticsEmBridgeID; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testLogAnalyticsEmBridgeID; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	assertLogAnalyticsEmBridgeTrailingCondition(t, resource, shared.Active)
}

func assertLogAnalyticsEmBridgeNoOpResult(
	t *testing.T,
	fake *fakeLogAnalyticsEmBridgeOCIClient,
	response servicemanager.OSOKResponse,
) {
	t.Helper()
	if got := len(fake.createRequests); got != 1 {
		t.Fatalf("CreateLogAnalyticsEmBridge calls = %d, want 1", got)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateLogAnalyticsEmBridge calls = %d, want 0", got)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active no-op", response)
	}
}

func TestLogAnalyticsEmBridgeCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newLogAnalyticsEmBridgeResource()
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.listFunc = pagedLogAnalyticsEmBridgeList(t, []logAnalyticsEmBridgeListPage{
		{
			items: []loganalyticssdk.LogAnalyticsEmBridgeSummary{
				newSDKLogAnalyticsEmBridgeSummary(testLogAnalyticsEmBridgeOtherID, "other-bridge", loganalyticssdk.EmBridgeLifecycleStatesActive),
			},
			nextPage: "page-2",
		},
		{
			wantPage: "page-2",
			items: []loganalyticssdk.LogAnalyticsEmBridgeSummary{
				newSDKLogAnalyticsEmBridgeSummary(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeDisplayName, loganalyticssdk.EmBridgeLifecycleStatesActive),
			},
		},
	})
	fake.getFunc = func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		assertLogAnalyticsEmBridgeGetRequest(t, request, testLogAnalyticsEmBridgeID)
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeBucketName, loganalyticssdk.EmBridgeLifecycleStatesActive),
		}, nil
	}
	fake.createFunc = func(context.Context, loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error) {
		t.Fatal("CreateLogAnalyticsEmBridge should not be called when list resolves an existing bridge")
		return loganalyticssdk.CreateLogAnalyticsEmBridgeResponse{}, nil
	}

	response, err := newTestLogAnalyticsEmBridgeClient(fake).CreateOrUpdate(context.Background(), resource, logAnalyticsEmBridgeReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind", response)
	}
	if got := len(fake.listRequests); got != 2 {
		t.Fatalf("ListLogAnalyticsEmBridges calls = %d, want 2", got)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testLogAnalyticsEmBridgeID; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	assertLogAnalyticsEmBridgeTrailingCondition(t, resource, shared.Active)
}

func TestLogAnalyticsEmBridgeMutableUpdateRefreshesObservedState(t *testing.T) {
	t.Parallel()

	resource := newLogAnalyticsEmBridgeResource()
	resource.Spec.Description = testLogAnalyticsEmBridgeUpdatedDescribe
	resource.Spec.BucketName = testLogAnalyticsEmBridgeUpdatedBucket
	resource.Status.Id = testLogAnalyticsEmBridgeID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsEmBridgeID)
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.getFunc = func(_ context.Context, request loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		assertLogAnalyticsEmBridgeGetRequest(t, request, testLogAnalyticsEmBridgeID)
		if len(fake.getRequests) == 1 {
			current := newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, "old-bucket", loganalyticssdk.EmBridgeLifecycleStatesActive)
			current.FreeformTags = nil
			current.DefinedTags = nil
			return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
				LogAnalyticsEmBridge: current,
			}, nil
		}
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeUpdatedBucket, loganalyticssdk.EmBridgeLifecycleStatesActive),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest) (loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse, error) {
		assertLogAnalyticsEmBridgeUpdateRequest(t, resource, request)
		return loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeUpdatedBucket, loganalyticssdk.EmBridgeLifecycleStatesActive),
			OpcRequestId:         common.String("opc-update"),
		}, nil
	}

	response, err := newTestLogAnalyticsEmBridgeClient(fake).CreateOrUpdate(context.Background(), resource, logAnalyticsEmBridgeReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active update", response)
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("UpdateLogAnalyticsEmBridge calls = %d, want 1", got)
	}
	if got, want := resource.Status.BucketName, testLogAnalyticsEmBridgeUpdatedBucket; got != want {
		t.Fatalf("status.bucketName = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	assertLogAnalyticsEmBridgeTrailingCondition(t, resource, shared.Active)
}

func TestLogAnalyticsEmBridgeCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newLogAnalyticsEmBridgeResource()
	resource.Spec.EmEntitiesCompartmentId = "ocid1.compartment.oc1..different"
	resource.Status.Id = testLogAnalyticsEmBridgeID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsEmBridgeID)
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.getFunc = func(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeBucketName, loganalyticssdk.EmBridgeLifecycleStatesActive),
		}, nil
	}

	response, err := newTestLogAnalyticsEmBridgeClient(fake).CreateOrUpdate(context.Background(), resource, logAnalyticsEmBridgeReconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "emEntitiesCompartmentId") {
		t.Fatalf("CreateOrUpdate() error = %q, want emEntitiesCompartmentId drift", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateLogAnalyticsEmBridge calls = %d, want 0", got)
	}
	assertLogAnalyticsEmBridgeTrailingCondition(t, resource, shared.Failed)
}

func TestLogAnalyticsEmBridgeDeleteWaitsForConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := newTrackedLogAnalyticsEmBridgeResource()
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.getFunc = func(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		if len(fake.getRequests) < 3 {
			return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
				LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeBucketName, loganalyticssdk.EmBridgeLifecycleStatesActive),
			}, nil
		}
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "bridge deleted")
	}
	fake.deleteFunc = func(_ context.Context, request loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error) {
		assertLogAnalyticsEmBridgeDeleteRequest(t, request)
		return loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestLogAnalyticsEmBridgeClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed delete")
	}
	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("DeleteLogAnalyticsEmBridge calls = %d, want 1", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want confirmed delete timestamp")
	}
	assertLogAnalyticsEmBridgeTrailingCondition(t, resource, shared.Terminating)
}

func TestLogAnalyticsEmBridgeDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newTrackedLogAnalyticsEmBridgeResource()
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.getFunc = func(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous pre-delete read")
	}

	deleted, err := newTestLogAnalyticsEmBridgeClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete read to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("DeleteLogAnalyticsEmBridge calls = %d, want 0", got)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 detail", err.Error())
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
}

func TestLogAnalyticsEmBridgeDeleteTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := newTrackedLogAnalyticsEmBridgeResource()
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.getFunc = func(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
		return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{
			LogAnalyticsEmBridge: newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, testLogAnalyticsEmBridgeBucketName, loganalyticssdk.EmBridgeLifecycleStatesActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error) {
		return loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete")
	}

	deleted, err := newTestLogAnalyticsEmBridgeClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped delete not-found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 detail", err.Error())
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLogAnalyticsEmBridgeCreateErrorCapturesOpcRequestID(t *testing.T) {
	t.Parallel()

	resource := newLogAnalyticsEmBridgeResource()
	fake := &fakeLogAnalyticsEmBridgeOCIClient{}
	fake.listFunc = func(context.Context, loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
		return loganalyticssdk.ListLogAnalyticsEmBridgesResponse{}, nil
	}
	fake.createFunc = func(context.Context, loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error) {
		return loganalyticssdk.CreateLogAnalyticsEmBridgeResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "create is still settling")
	}

	response, err := newTestLogAnalyticsEmBridgeClient(fake).CreateOrUpdate(context.Background(), resource, logAnalyticsEmBridgeReconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	assertLogAnalyticsEmBridgeTrailingCondition(t, resource, shared.Failed)
}

type logAnalyticsEmBridgeListPage struct {
	wantPage string
	nextPage string
	items    []loganalyticssdk.LogAnalyticsEmBridgeSummary
}

func pagedLogAnalyticsEmBridgeList(
	t *testing.T,
	pages []logAnalyticsEmBridgeListPage,
) func(context.Context, loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
	t.Helper()
	index := 0
	return func(_ context.Context, request loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
		if index >= len(pages) {
			t.Fatalf("ListLogAnalyticsEmBridges called %d times, want %d", index+1, len(pages))
		}
		page := pages[index]
		index++
		if got := logAnalyticsEmBridgeStringValue(request.Page); got != page.wantPage {
			t.Fatalf("ListLogAnalyticsEmBridges page = %q, want %q", got, page.wantPage)
		}
		assertLogAnalyticsEmBridgeListRequest(t, request)
		response := loganalyticssdk.ListLogAnalyticsEmBridgesResponse{
			LogAnalyticsEmBridgeCollection: loganalyticssdk.LogAnalyticsEmBridgeCollection{Items: page.items},
		}
		if page.nextPage != "" {
			response.OpcNextPage = common.String(page.nextPage)
		}
		return response, nil
	}
}

func newTestLogAnalyticsEmBridgeClient(fake *fakeLogAnalyticsEmBridgeOCIClient) LogAnalyticsEmBridgeServiceClient {
	provider := common.NewRawConfigurationProvider(
		testLogAnalyticsEmBridgeTenancyID,
		"ocid1.user.oc1..logan",
		"us-ashburn-1",
		"00:00:00",
		"unused",
		nil,
	)
	return newLogAnalyticsEmBridgeServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, provider, fake)
}

func newLogAnalyticsEmBridgeResource() *loganalyticsv1beta1.LogAnalyticsEmBridge {
	return &loganalyticsv1beta1.LogAnalyticsEmBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "em-bridge",
			Namespace: testLogAnalyticsEmBridgeKubeNamespace,
			UID:       k8stypes.UID("loganalytics-em-bridge-uid"),
		},
		Spec: loganalyticsv1beta1.LogAnalyticsEmBridgeSpec{
			DisplayName:             testLogAnalyticsEmBridgeDisplayName,
			CompartmentId:           testLogAnalyticsEmBridgeCompartmentID,
			EmEntitiesCompartmentId: testLogAnalyticsEmBridgeEntitiesCompID,
			BucketName:              testLogAnalyticsEmBridgeBucketName,
			Description:             testLogAnalyticsEmBridgeDescription,
			FreeformTags:            map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func newTrackedLogAnalyticsEmBridgeResource() *loganalyticsv1beta1.LogAnalyticsEmBridge {
	resource := newLogAnalyticsEmBridgeResource()
	resource.Status.Id = testLogAnalyticsEmBridgeID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsEmBridgeID)
	return resource
}

func newSDKLogAnalyticsEmBridge(
	id string,
	bucketName string,
	state loganalyticssdk.EmBridgeLifecycleStatesEnum,
) loganalyticssdk.LogAnalyticsEmBridge {
	return loganalyticssdk.LogAnalyticsEmBridge{
		Id:                         common.String(id),
		DisplayName:                common.String(testLogAnalyticsEmBridgeDisplayName),
		CompartmentId:              common.String(testLogAnalyticsEmBridgeCompartmentID),
		EmEntitiesCompartmentId:    common.String(testLogAnalyticsEmBridgeEntitiesCompID),
		BucketName:                 common.String(bucketName),
		LifecycleState:             state,
		LastImportProcessingStatus: loganalyticssdk.EmBridgeLatestImportProcessingStatusSuccess,
		Description:                common.String(testLogAnalyticsEmBridgeDescription),
		FreeformTags:               map[string]string{"env": "dev"},
		DefinedTags:                map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func newSDKLogAnalyticsEmBridgeSummary(
	id string,
	displayName string,
	state loganalyticssdk.EmBridgeLifecycleStatesEnum,
) loganalyticssdk.LogAnalyticsEmBridgeSummary {
	return loganalyticssdk.LogAnalyticsEmBridgeSummary{
		Id:                         common.String(id),
		DisplayName:                common.String(displayName),
		CompartmentId:              common.String(testLogAnalyticsEmBridgeCompartmentID),
		EmEntitiesCompartmentId:    common.String(testLogAnalyticsEmBridgeEntitiesCompID),
		BucketName:                 common.String(testLogAnalyticsEmBridgeBucketName),
		LifecycleState:             state,
		LastImportProcessingStatus: loganalyticssdk.EmBridgeLatestImportProcessingStatusSuccess,
		Description:                common.String(testLogAnalyticsEmBridgeDescription),
		FreeformTags:               map[string]string{"env": "dev"},
		DefinedTags:                map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func logAnalyticsEmBridgeReconcileRequest(resource *loganalyticsv1beta1.LogAnalyticsEmBridge) ctrl.Request {
	return ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func assertLogAnalyticsEmBridgeCreateRequest(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	request loganalyticssdk.CreateLogAnalyticsEmBridgeRequest,
) {
	t.Helper()
	requireLogAnalyticsEmBridgeStringPtr(t, "create namespaceName", request.NamespaceName, testLogAnalyticsEmBridgeNamespace)
	requireLogAnalyticsEmBridgeStringPtr(t, "create opcRetryToken", request.OpcRetryToken, string(resource.UID))
	requireLogAnalyticsEmBridgeStringPtr(t, "create displayName", request.DisplayName, testLogAnalyticsEmBridgeDisplayName)
	requireLogAnalyticsEmBridgeStringPtr(t, "create compartmentId", request.CompartmentId, testLogAnalyticsEmBridgeCompartmentID)
	requireLogAnalyticsEmBridgeStringPtr(t, "create emEntitiesCompartmentId", request.EmEntitiesCompartmentId, testLogAnalyticsEmBridgeEntitiesCompID)
	requireLogAnalyticsEmBridgeStringPtr(t, "create bucketName", request.BucketName, testLogAnalyticsEmBridgeBucketName)
	requireLogAnalyticsEmBridgeStringPtr(t, "create description", request.Description, testLogAnalyticsEmBridgeDescription)
	if got, want := request.FreeformTags["env"], "dev"; got != want {
		t.Fatalf("create freeformTags[env] = %q, want %q", got, want)
	}
	if got, want := request.DefinedTags["Operations"]["CostCenter"], "42"; got != want {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want %q", got, want)
	}
}

func assertLogAnalyticsEmBridgeUpdateRequest(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	request loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest,
) {
	t.Helper()
	requireLogAnalyticsEmBridgeStringPtr(t, "update namespaceName", request.NamespaceName, testLogAnalyticsEmBridgeNamespace)
	requireLogAnalyticsEmBridgeStringPtr(t, "update id", request.LogAnalyticsEmBridgeId, testLogAnalyticsEmBridgeID)
	if request.DisplayName != nil {
		t.Fatalf("update displayName = %q, want nil when unchanged", *request.DisplayName)
	}
	requireLogAnalyticsEmBridgeStringPtr(t, "update description", request.Description, resource.Spec.Description)
	requireLogAnalyticsEmBridgeStringPtr(t, "update bucketName", request.BucketName, resource.Spec.BucketName)
	if got, want := request.FreeformTags["env"], "dev"; got != want {
		t.Fatalf("update freeformTags[env] = %q, want %q", got, want)
	}
	if got, want := request.DefinedTags["Operations"]["CostCenter"], "42"; got != want {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want %q", got, want)
	}
}

func assertLogAnalyticsEmBridgeGetRequest(
	t *testing.T,
	request loganalyticssdk.GetLogAnalyticsEmBridgeRequest,
	wantID string,
) {
	t.Helper()
	requireLogAnalyticsEmBridgeStringPtr(t, "get namespaceName", request.NamespaceName, testLogAnalyticsEmBridgeNamespace)
	requireLogAnalyticsEmBridgeStringPtr(t, "get id", request.LogAnalyticsEmBridgeId, wantID)
}

func assertLogAnalyticsEmBridgeListRequest(
	t *testing.T,
	request loganalyticssdk.ListLogAnalyticsEmBridgesRequest,
) {
	t.Helper()
	requireLogAnalyticsEmBridgeStringPtr(t, "list namespaceName", request.NamespaceName, testLogAnalyticsEmBridgeNamespace)
	requireLogAnalyticsEmBridgeStringPtr(t, "list compartmentId", request.CompartmentId, testLogAnalyticsEmBridgeCompartmentID)
	requireLogAnalyticsEmBridgeStringPtr(t, "list displayName", request.DisplayName, testLogAnalyticsEmBridgeDisplayName)
}

func assertLogAnalyticsEmBridgeDeleteRequest(
	t *testing.T,
	request loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest,
) {
	t.Helper()
	requireLogAnalyticsEmBridgeStringPtr(t, "delete namespaceName", request.NamespaceName, testLogAnalyticsEmBridgeNamespace)
	requireLogAnalyticsEmBridgeStringPtr(t, "delete id", request.LogAnalyticsEmBridgeId, testLogAnalyticsEmBridgeID)
}

func requireLogAnalyticsEmBridgeStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertLogAnalyticsEmBridgeTrailingCondition(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = nil, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %s, want %s", got, want)
	}
}

func assertLogAnalyticsEmBridgeContainsAll(t *testing.T, got []string, want ...string) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, value := range got {
		seen[value] = struct{}{}
	}
	for _, value := range want {
		if _, ok := seen[value]; !ok {
			t.Fatalf("values %v missing %q", got, value)
		}
	}
}
