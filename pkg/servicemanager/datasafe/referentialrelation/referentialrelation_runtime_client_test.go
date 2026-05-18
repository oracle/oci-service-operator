/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package referentialrelation

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testReferentialRelationKey                  = "1001"
	testReferentialRelationOtherKey             = "2002"
	testReferentialRelationSensitiveDataModelID = "ocid1.datasafesensitivedatamodel.oc1..model"
)

type fakeReferentialRelationOCIClient struct {
	createFn func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error)
	getFn    func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error)
	listFn   func(context.Context, datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error)
	deleteFn func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	deleteCalls int
}

func (f *fakeReferentialRelationOCIClient) CreateReferentialRelation(
	ctx context.Context,
	request datasafesdk.CreateReferentialRelationRequest,
) (datasafesdk.CreateReferentialRelationResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateReferentialRelationResponse{}, nil
}

func (f *fakeReferentialRelationOCIClient) GetReferentialRelation(
	ctx context.Context,
	request datasafesdk.GetReferentialRelationRequest,
) (datasafesdk.GetReferentialRelationResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetReferentialRelationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "ReferentialRelation is missing")
}

func (f *fakeReferentialRelationOCIClient) ListReferentialRelations(
	ctx context.Context,
	request datasafesdk.ListReferentialRelationsRequest,
) (datasafesdk.ListReferentialRelationsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListReferentialRelationsResponse{}, nil
}

func (f *fakeReferentialRelationOCIClient) DeleteReferentialRelation(
	ctx context.Context,
	request datasafesdk.DeleteReferentialRelationRequest,
) (datasafesdk.DeleteReferentialRelationResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteReferentialRelationResponse{}, nil
}

func TestReferentialRelationRuntimeHooksConfigured(t *testing.T) {
	hooks := newReferentialRelationDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyReferentialRelationRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
		{name: "Identity.RecordTracked", ok: hooks.Identity.RecordTracked != nil},
		{name: "Read.Get", ok: hooks.Read.Get != nil},
		{name: "Read.List", ok: hooks.Read.List != nil},
		{name: "ParityHooks.ValidateCreateOnlyDrift", ok: hooks.ParityHooks.ValidateCreateOnlyDrift != nil},
		{name: "DeleteHooks.ConfirmRead", ok: hooks.DeleteHooks.ConfirmRead != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "DeleteHooks.ApplyOutcome", ok: hooks.DeleteHooks.ApplyOutcome != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "StatusHooks.MarkTerminating", ok: hooks.StatusHooks.MarkTerminating != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	body, err := hooks.BuildCreateBody(context.Background(), makeReferentialRelationResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateReferentialRelationDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateReferentialRelationDetails", body)
	}
	if details.RelationType != datasafesdk.CreateReferentialRelationDetailsRelationTypeAppDefined {
		t.Fatalf("CreateReferentialRelationDetails.RelationType = %q, want APP_DEFINED", details.RelationType)
	}
	requireBoolPtr(t, "CreateReferentialRelationDetails.IsSensitive", details.IsSensitive, false)
	requireSDKColumnsInfo(t, "CreateReferentialRelationDetails.Parent", details.Parent, makeReferentialRelationResource().Spec.Parent)
	requireSDKColumnsInfo(t, "CreateReferentialRelationDetails.Child", details.Child, parentToChild(makeReferentialRelationResource().Spec.Child))
}

func TestReferentialRelationCreateRecordsKeyParentRequestIDAndLifecycle(t *testing.T) {
	resource := makeReferentialRelationResource()
	created := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateCreating)
	listCalls := 0
	client := &fakeReferentialRelationOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListReferentialRelationsRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			requireBoolPtr(t, "ListReferentialRelationsRequest.IsSensitive", request.IsSensitive, false)
			if listCalls == 3 {
				return datasafesdk.ListReferentialRelationsResponse{
					ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
						Items: []datasafesdk.ReferentialRelationSummary{
							sdkReferentialRelationSummary(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateCreating),
						},
					},
				}, nil
			}
			return datasafesdk.ListReferentialRelationsResponse{}, nil
		},
		createFn: func(_ context.Context, request datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			requireReferentialRelationCreateRequest(t, request, resource)
			return datasafesdk.CreateReferentialRelationResponse{
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: created}, nil
		},
	}

	response, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while CREATING")
	}
	assertCallCount(t, "ListReferentialRelations()", client.listCalls, 3)
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 1)
	assertReferentialRelationTracked(t, resource, testReferentialRelationKey)
	if got := resource.Status.SensitiveDataModelId; got != testReferentialRelationSensitiveDataModelID {
		t.Fatalf("status.sensitiveDataModelId = %q, want %q", got, testReferentialRelationSensitiveDataModelID)
	}
	assertReferentialRelationOpcRequestID(t, resource, "opc-create")
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %#v, want create lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestReferentialRelationCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeReferentialRelationResource()
	existing := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	var pages []string
	client := &fakeReferentialRelationOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListReferentialRelationsRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			if request.Page == nil {
				return datasafesdk.ListReferentialRelationsResponse{
					ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
						Items: []datasafesdk.ReferentialRelationSummary{
							sdkReferentialRelationSummary(withChildObject(resource, "UNRELATED"), testReferentialRelationOtherKey, datasafesdk.ReferentialRelationLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListReferentialRelationsResponse{
				ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
					Items: []datasafesdk.ReferentialRelationSummary{
						sdkReferentialRelationSummary(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			requireGetRequest(t, request, testReferentialRelationKey)
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			t.Fatal("CreateReferentialRelation() called despite existing list match")
			return datasafesdk.CreateReferentialRelationResponse{}, nil
		},
	}

	response, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListReferentialRelations() pages = %q, want \",page-2\"", got)
	}
	assertReferentialRelationTracked(t, resource, testReferentialRelationKey)
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 0)
}

func TestReferentialRelationCreateOrUpdateBindsExistingWithExplicitEmptySensitiveTypes(t *testing.T) {
	resource := makeReferentialRelationResource()
	resource.Spec.Parent.SensitiveTypeIds = []string{}
	resource.Spec.Child.SensitiveTypeIds = []string{}
	existing := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	client := &fakeReferentialRelationOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
			requireStringPtr(t, "ListReferentialRelationsRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			return datasafesdk.ListReferentialRelationsResponse{
				ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
					Items: []datasafesdk.ReferentialRelationSummary{
						sdkReferentialRelationSummary(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			requireGetRequest(t, request, testReferentialRelationKey)
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			t.Fatal("CreateReferentialRelation() called despite existing list match with nil sensitive type readback")
			return datasafesdk.CreateReferentialRelationResponse{}, nil
		},
	}

	response, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertReferentialRelationTracked(t, resource, testReferentialRelationKey)
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 0)
}

func TestReferentialRelationCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeReferentialRelationResource()
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	current := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	client := &fakeReferentialRelationOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			requireGetRequest(t, request, testReferentialRelationKey)
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			t.Fatal("CreateReferentialRelation() called during no-op reconcile")
			return datasafesdk.CreateReferentialRelationResponse{}, nil
		},
	}

	response, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertCallCount(t, "GetReferentialRelation()", client.getCalls, 1)
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestReferentialRelationCreateOnlyDriftAllowsExplicitEmptySensitiveTypes(t *testing.T) {
	resource := makeReferentialRelationResource()
	resource.Spec.Parent.SensitiveTypeIds = []string{}
	resource.Spec.Child.SensitiveTypeIds = []string{}
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	current := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	client := &fakeReferentialRelationOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			requireGetRequest(t, request, testReferentialRelationKey)
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			t.Fatal("CreateReferentialRelation() called during no-op reconcile with nil sensitive type readback")
			return datasafesdk.CreateReferentialRelationResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			t.Fatal("DeleteReferentialRelation() called during create-only drift check")
			return datasafesdk.DeleteReferentialRelationResponse{}, nil
		},
	}

	response, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 0)
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestReferentialRelationNoUpdateDriftRejectedBeforeMutatingOCI(t *testing.T) {
	resource := makeReferentialRelationResource()
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	currentResource := makeReferentialRelationResource()
	current := sdkReferentialRelation(currentResource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	resource.Spec.Child.ObjectName = "ORDERS_RENAMED"
	client := &fakeReferentialRelationOCIClient{
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			t.Fatal("CreateReferentialRelation() called despite create-only drift")
			return datasafesdk.CreateReferentialRelationResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			t.Fatal("DeleteReferentialRelation() called during create/update drift handling")
			return datasafesdk.DeleteReferentialRelationResponse{}, nil
		},
	}

	_, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "child") {
		t.Fatalf("CreateOrUpdate() error = %v, want child drift detail", err)
	}
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 0)
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 0)
}

func TestReferentialRelationParentAnnotationDriftRejectedBeforeOCI(t *testing.T) {
	resource := makeReferentialRelationResource()
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	resource.Annotations[referentialRelationSensitiveDataModelIDAnnotation] = "ocid1.datasafesensitivedatamodel.oc1..other"
	client := &fakeReferentialRelationOCIClient{
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			t.Fatal("GetReferentialRelation() called despite parent identity drift")
			return datasafesdk.GetReferentialRelationResponse{}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			t.Fatal("CreateReferentialRelation() called despite parent identity drift")
			return datasafesdk.CreateReferentialRelationResponse{}, nil
		},
	}

	_, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want parent identity drift rejection")
	}
	if !strings.Contains(err.Error(), "sensitiveDataModelId") {
		t.Fatalf("CreateOrUpdate() error = %v, want sensitiveDataModelId drift detail", err)
	}
	assertCallCount(t, "GetReferentialRelation()", client.getCalls, 0)
	assertCallCount(t, "CreateReferentialRelation()", client.createCalls, 0)
	requireLastCondition(t, resource, shared.Failed)
}

func TestReferentialRelationDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := makeReferentialRelationResource()
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	active := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	getResponses := []datasafesdk.GetReferentialRelationResponse{
		{ReferentialRelation: active},
		{ReferentialRelation: active},
	}
	client := &fakeReferentialRelationOCIClient{
		getFn: getReferentialRelationResponses(t, &getResponses),
		deleteFn: func(_ context.Context, request datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			requireStringPtr(t, "DeleteReferentialRelationRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			requireStringPtr(t, "DeleteReferentialRelationRequest.ReferentialRelationKey", request.ReferentialRelationKey, testReferentialRelationKey)
			return datasafesdk.DeleteReferentialRelationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestReferentialRelationClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertCallCount(t, "GetReferentialRelation()", client.getCalls, 2)
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 1)
	assertReferentialRelationOpcRequestID(t, resource, "opc-delete")
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestReferentialRelationDeleteResolvesMissingTrackedKeyFromLaterListPage(t *testing.T) {
	resource := makeReferentialRelationResource()
	active := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	var pages []string
	client := &fakeReferentialRelationOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListReferentialRelationsRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			requireBoolPtr(t, "ListReferentialRelationsRequest.IsSensitive", request.IsSensitive, false)
			if request.Page == nil {
				return datasafesdk.ListReferentialRelationsResponse{
					ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
						Items: []datasafesdk.ReferentialRelationSummary{
							sdkReferentialRelationSummary(withChildObject(resource, "UNRELATED"), testReferentialRelationOtherKey, datasafesdk.ReferentialRelationLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListReferentialRelationsResponse{
				ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
					Items: []datasafesdk.ReferentialRelationSummary{
						sdkReferentialRelationSummary(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			requireGetRequest(t, request, testReferentialRelationKey)
			return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: active}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			requireStringPtr(t, "DeleteReferentialRelationRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			requireStringPtr(t, "DeleteReferentialRelationRequest.ReferentialRelationKey", request.ReferentialRelationKey, testReferentialRelationKey)
			return datasafesdk.DeleteReferentialRelationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestReferentialRelationClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback is still active")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListReferentialRelations() pages = %q, want \",page-2\"", got)
	}
	assertCallCount(t, "ListReferentialRelations()", client.listCalls, 2)
	assertCallCount(t, "GetReferentialRelation()", client.getCalls, 2)
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 1)
	assertReferentialRelationTracked(t, resource, testReferentialRelationKey)
	assertReferentialRelationOpcRequestID(t, resource, "opc-delete")
	requireLastCondition(t, resource, shared.Terminating)
}

func TestReferentialRelationDeleteConfirmsAbsentWhenNoTrackedKeyAndListHasNoMatch(t *testing.T) {
	resource := makeReferentialRelationResource()
	var pages []string
	client := &fakeReferentialRelationOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListReferentialRelationsRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
			if request.Page == nil {
				return datasafesdk.ListReferentialRelationsResponse{
					ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
						Items: []datasafesdk.ReferentialRelationSummary{
							sdkReferentialRelationSummary(withChildObject(resource, "UNRELATED"), testReferentialRelationOtherKey, datasafesdk.ReferentialRelationLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListReferentialRelationsResponse{
				ReferentialRelationCollection: datasafesdk.ReferentialRelationCollection{
					Items: []datasafesdk.ReferentialRelationSummary{
						sdkReferentialRelationSummary(withChildObject(resource, "STILL_UNRELATED"), testReferentialRelationOtherKey, datasafesdk.ReferentialRelationLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			t.Fatal("GetReferentialRelation() called despite no list match")
			return datasafesdk.GetReferentialRelationResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			t.Fatal("DeleteReferentialRelation() called despite no list match")
			return datasafesdk.DeleteReferentialRelationResponse{}, nil
		},
	}

	deleted, err := newTestReferentialRelationClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after full list confirms absence")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListReferentialRelations() pages = %q, want \",page-2\"", got)
	}
	assertCallCount(t, "ListReferentialRelations()", client.listCalls, 2)
	assertCallCount(t, "GetReferentialRelation()", client.getCalls, 0)
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestReferentialRelationDeleteRejectsAuthShapedListWhenTrackedKeyMissing(t *testing.T) {
	resource := makeReferentialRelationResource()
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-list-auth"
	client := &fakeReferentialRelationOCIClient{
		listFn: func(context.Context, datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
			return datasafesdk.ListReferentialRelationsResponse{}, authErr
		},
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			t.Fatal("GetReferentialRelation() called after ambiguous list read")
			return datasafesdk.GetReferentialRelationResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			t.Fatal("DeleteReferentialRelation() called after ambiguous list read")
			return datasafesdk.DeleteReferentialRelationResponse{}, nil
		},
	}

	deleted, err := newTestReferentialRelationClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous list read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	assertCallCount(t, "ListReferentialRelations()", client.listCalls, 1)
	assertCallCount(t, "GetReferentialRelation()", client.getCalls, 0)
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 0)
	assertReferentialRelationOpcRequestID(t, resource, "opc-list-auth")
}

func TestReferentialRelationDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := makeReferentialRelationResource()
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-auth"
	client := &fakeReferentialRelationOCIClient{
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			return datasafesdk.GetReferentialRelationResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			t.Fatal("DeleteReferentialRelation() called after ambiguous pre-delete read")
			return datasafesdk.DeleteReferentialRelationResponse{}, nil
		},
	}

	deleted, err := newTestReferentialRelationClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 0)
	assertReferentialRelationOpcRequestID(t, resource, "opc-auth")
}

func TestReferentialRelationDeleteConfirmsUnambiguousNotFoundAfterDelete(t *testing.T) {
	resource := makeReferentialRelationResource()
	seedReferentialRelationStatus(resource, testReferentialRelationKey)
	active := sdkReferentialRelation(resource, testReferentialRelationKey, datasafesdk.ReferentialRelationLifecycleStateActive)
	getCalls := 0
	client := &fakeReferentialRelationOCIClient{
		getFn: func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetReferentialRelationResponse{ReferentialRelation: active}, nil
			}
			return datasafesdk.GetReferentialRelationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
			return datasafesdk.DeleteReferentialRelationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestReferentialRelationClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after unambiguous not found")
	}
	assertCallCount(t, "DeleteReferentialRelation()", client.deleteCalls, 1)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	assertReferentialRelationOpcRequestID(t, resource, "opc-request-id")
}

func TestReferentialRelationCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := makeReferentialRelationResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	client := &fakeReferentialRelationOCIClient{
		createFn: func(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
			return datasafesdk.CreateReferentialRelationResponse{}, createErr
		},
	}

	_, err := newTestReferentialRelationClient(client).CreateOrUpdate(context.Background(), resource, referentialRelationRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	assertReferentialRelationOpcRequestID(t, resource, "opc-create-error")
	requireLastCondition(t, resource, shared.Failed)
}

func newTestReferentialRelationClient(client referentialRelationOCIClient) ReferentialRelationServiceClient {
	return newReferentialRelationServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeReferentialRelationResource() *datasafev1beta1.ReferentialRelation {
	return &datasafev1beta1.ReferentialRelation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "referential-relation",
			Namespace: "default",
			UID:       types.UID("referential-relation-uid"),
			Annotations: map[string]string{
				referentialRelationSensitiveDataModelIDAnnotation: testReferentialRelationSensitiveDataModelID,
			},
		},
		Spec: datasafev1beta1.ReferentialRelationSpec{
			RelationType: string(datasafesdk.CreateReferentialRelationDetailsRelationTypeAppDefined),
			Parent: datasafev1beta1.ReferentialRelationParent{
				SchemaName:       "APP",
				ObjectType:       string(datasafesdk.ColumnsInfoObjectTypeTable),
				ObjectName:       "CUSTOMERS",
				AppName:          "CRM",
				ColumnGroup:      []string{"CUSTOMER_ID"},
				SensitiveTypeIds: []string{"ocid1.datasafesensitivetype.oc1..parent"},
			},
			Child: datasafev1beta1.ReferentialRelationChild{
				SchemaName:       "APP",
				ObjectType:       string(datasafesdk.ColumnsInfoObjectTypeTable),
				ObjectName:       "ORDERS",
				AppName:          "CRM",
				ColumnGroup:      []string{"CUSTOMER_ID"},
				SensitiveTypeIds: []string{"ocid1.datasafesensitivetype.oc1..child"},
			},
			IsSensitive: false,
		},
	}
}

func referentialRelationRequest(resource *datasafev1beta1.ReferentialRelation) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func seedReferentialRelationStatus(resource *datasafev1beta1.ReferentialRelation, key string) {
	resource.Status.Key = key
	resource.Status.SensitiveDataModelId = testReferentialRelationSensitiveDataModelID
	resource.Status.OsokStatus.Ocid = shared.OCID(key)
}

func sdkReferentialRelation(
	resource *datasafev1beta1.ReferentialRelation,
	key string,
	state datasafesdk.ReferentialRelationLifecycleStateEnum,
) datasafesdk.ReferentialRelation {
	return datasafesdk.ReferentialRelation{
		Key:                  common.String(key),
		LifecycleState:       state,
		SensitiveDataModelId: common.String(testReferentialRelationSensitiveDataModelID),
		RelationType:         datasafesdk.ReferentialRelationRelationTypeEnum(resource.Spec.RelationType),
		Parent:               sdkColumnsInfoFromParent(resource.Spec.Parent),
		Child:                sdkColumnsInfoFromChild(resource.Spec.Child),
		IsSensitive:          common.Bool(resource.Spec.IsSensitive),
	}
}

func sdkReferentialRelationSummary(
	resource *datasafev1beta1.ReferentialRelation,
	key string,
	state datasafesdk.ReferentialRelationLifecycleStateEnum,
) datasafesdk.ReferentialRelationSummary {
	return datasafesdk.ReferentialRelationSummary{
		Key:                  common.String(key),
		LifecycleState:       state,
		SensitiveDataModelId: common.String(testReferentialRelationSensitiveDataModelID),
		RelationType:         datasafesdk.ReferentialRelationSummaryRelationTypeEnum(resource.Spec.RelationType),
		Parent:               sdkColumnsInfoFromParent(resource.Spec.Parent),
		Child:                sdkColumnsInfoFromChild(resource.Spec.Child),
		IsSensitive:          common.Bool(resource.Spec.IsSensitive),
	}
}

func sdkColumnsInfoFromParent(input datasafev1beta1.ReferentialRelationParent) *datasafesdk.ColumnsInfo {
	return &datasafesdk.ColumnsInfo{
		SchemaName:       common.String(input.SchemaName),
		ObjectType:       datasafesdk.ColumnsInfoObjectTypeEnum(input.ObjectType),
		ObjectName:       common.String(input.ObjectName),
		AppName:          common.String(input.AppName),
		ColumnGroup:      referentialRelationStringSlice(input.ColumnGroup),
		SensitiveTypeIds: referentialRelationStringSlice(input.SensitiveTypeIds),
	}
}

func sdkColumnsInfoFromChild(input datasafev1beta1.ReferentialRelationChild) *datasafesdk.ColumnsInfo {
	return &datasafesdk.ColumnsInfo{
		SchemaName:       common.String(input.SchemaName),
		ObjectType:       datasafesdk.ColumnsInfoObjectTypeEnum(input.ObjectType),
		ObjectName:       common.String(input.ObjectName),
		AppName:          common.String(input.AppName),
		ColumnGroup:      referentialRelationStringSlice(input.ColumnGroup),
		SensitiveTypeIds: referentialRelationStringSlice(input.SensitiveTypeIds),
	}
}

func withChildObject(resource *datasafev1beta1.ReferentialRelation, objectName string) *datasafev1beta1.ReferentialRelation {
	cloned := resource.DeepCopy()
	cloned.Spec.Child.ObjectName = objectName
	return cloned
}

func getReferentialRelationResponses(
	t *testing.T,
	responses *[]datasafesdk.GetReferentialRelationResponse,
) func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
		requireGetRequest(t, request, testReferentialRelationKey)
		if len(*responses) == 0 {
			t.Fatal("GetReferentialRelation() called more times than expected")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func requireReferentialRelationCreateRequest(
	t *testing.T,
	request datasafesdk.CreateReferentialRelationRequest,
	resource *datasafev1beta1.ReferentialRelation,
) {
	t.Helper()
	requireStringPtr(t, "CreateReferentialRelationRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateReferentialRelationRequest.OpcRetryToken is empty")
	}
	if request.RelationType != datasafesdk.CreateReferentialRelationDetailsRelationTypeEnum(resource.Spec.RelationType) {
		t.Fatalf("CreateReferentialRelationRequest.RelationType = %q, want %q", request.RelationType, resource.Spec.RelationType)
	}
	requireSDKColumnsInfo(t, "CreateReferentialRelationRequest.Parent", request.Parent, resource.Spec.Parent)
	requireSDKColumnsInfo(t, "CreateReferentialRelationRequest.Child", request.Child, parentToChild(resource.Spec.Child))
	requireBoolPtr(t, "CreateReferentialRelationRequest.IsSensitive", request.IsSensitive, resource.Spec.IsSensitive)
}

func requireGetRequest(t *testing.T, request datasafesdk.GetReferentialRelationRequest, key string) {
	t.Helper()
	requireStringPtr(t, "GetReferentialRelationRequest.SensitiveDataModelId", request.SensitiveDataModelId, testReferentialRelationSensitiveDataModelID)
	requireStringPtr(t, "GetReferentialRelationRequest.ReferentialRelationKey", request.ReferentialRelationKey, key)
}

func requireSDKColumnsInfo(
	t *testing.T,
	name string,
	got *datasafesdk.ColumnsInfo,
	want datasafev1beta1.ReferentialRelationParent,
) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil", name)
	}
	requireStringPtr(t, name+".SchemaName", got.SchemaName, want.SchemaName)
	if string(got.ObjectType) != want.ObjectType {
		t.Fatalf("%s.ObjectType = %q, want %q", name, got.ObjectType, want.ObjectType)
	}
	requireStringPtr(t, name+".ObjectName", got.ObjectName, want.ObjectName)
	requireStringPtr(t, name+".AppName", got.AppName, want.AppName)
	if strings.Join(got.ColumnGroup, ",") != strings.Join(want.ColumnGroup, ",") {
		t.Fatalf("%s.ColumnGroup = %#v, want %#v", name, got.ColumnGroup, want.ColumnGroup)
	}
	if strings.Join(got.SensitiveTypeIds, ",") != strings.Join(want.SensitiveTypeIds, ",") {
		t.Fatalf("%s.SensitiveTypeIds = %#v, want %#v", name, got.SensitiveTypeIds, want.SensitiveTypeIds)
	}
}

func parentToChild(input datasafev1beta1.ReferentialRelationChild) datasafev1beta1.ReferentialRelationParent {
	return datasafev1beta1.ReferentialRelationParent(input)
}

func assertReferentialRelationTracked(t *testing.T, resource *datasafev1beta1.ReferentialRelation, key string) {
	t.Helper()
	if got := resource.Status.Key; got != key {
		t.Fatalf("status.key = %q, want %q", got, key)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != key {
		t.Fatalf("status.status.ocid = %q, want %q", got, key)
	}
}

func assertReferentialRelationOpcRequestID(t *testing.T, resource *datasafev1beta1.ReferentialRelation, requestID string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != requestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, requestID)
	}
}

func assertCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
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

func requireLastCondition(
	t *testing.T,
	resource *datasafev1beta1.ReferentialRelation,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = nil, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
