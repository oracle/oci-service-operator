/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package view

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeViewOCIClient struct {
	createViewFn func(context.Context, dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error)
	getViewFn    func(context.Context, dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error)
	listViewsFn  func(context.Context, dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error)
	updateViewFn func(context.Context, dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error)
	deleteViewFn func(context.Context, dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error)
}

func (f *fakeViewOCIClient) CreateView(ctx context.Context, req dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
	if f.createViewFn != nil {
		return f.createViewFn(ctx, req)
	}
	return dnssdk.CreateViewResponse{}, nil
}

func (f *fakeViewOCIClient) GetView(ctx context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
	if f.getViewFn != nil {
		return f.getViewFn(ctx, req)
	}
	return dnssdk.GetViewResponse{}, nil
}

func (f *fakeViewOCIClient) ListViews(ctx context.Context, req dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error) {
	if f.listViewsFn != nil {
		return f.listViewsFn(ctx, req)
	}
	return dnssdk.ListViewsResponse{}, nil
}

func (f *fakeViewOCIClient) UpdateView(ctx context.Context, req dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error) {
	if f.updateViewFn != nil {
		return f.updateViewFn(ctx, req)
	}
	return dnssdk.UpdateViewResponse{}, nil
}

func (f *fakeViewOCIClient) DeleteView(ctx context.Context, req dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
	if f.deleteViewFn != nil {
		return f.deleteViewFn(ctx, req)
	}
	return dnssdk.DeleteViewResponse{}, nil
}

func testViewClient(fake *fakeViewOCIClient) ViewServiceClient {
	return newViewServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeViewResource() *dnsv1beta1.View {
	return &dnsv1beta1.View{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "view-sample",
			Namespace: "default",
			UID:       types.UID("view-uid"),
		},
		Spec: dnsv1beta1.ViewSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "view-alpha",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKView(
	id string,
	compartmentID string,
	displayName string,
	state dnssdk.ViewLifecycleStateEnum,
) dnssdk.View {
	return dnssdk.View{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		IsProtected:    common.Bool(false),
	}
}

func makeSDKViewSummary(
	id string,
	compartmentID string,
	displayName string,
	state dnssdk.ViewSummaryLifecycleStateEnum,
) dnssdk.ViewSummary {
	return dnssdk.ViewSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		IsProtected:    common.Bool(false),
	}
}

//nolint:gocognit,gocyclo // The request/status assertions keep the create flow visible in one regression.
func TestViewServiceClientCreateOrUpdateCreatesPrivateViewAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	var createRequest dnssdk.CreateViewRequest
	var getRequest dnssdk.GetViewRequest

	client := testViewClient(&fakeViewOCIClient{
		createViewFn: func(_ context.Context, req dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
			createRequest = req
			return dnssdk.CreateViewResponse{
				OpcRequestId: common.String("opc-create-1"),
				View: makeSDKView(
					"ocid1.dnsview.oc1..created",
					"ocid1.compartment.oc1..example",
					"view-alpha",
					dnssdk.ViewLifecycleStateActive,
				),
			}, nil
		},
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			getRequest = req
			return dnssdk.GetViewResponse{
				View: makeSDKView(
					"ocid1.dnsview.oc1..created",
					"ocid1.compartment.oc1..example",
					"view-alpha",
					dnssdk.ViewLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makeViewResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if createRequest.Scope != dnssdk.CreateViewScopePrivate {
		t.Fatalf("create scope = %q, want PRIVATE", createRequest.Scope)
	}
	if createRequest.CompartmentId == nil || *createRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.DisplayName == nil || *createRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %v, want %q", createRequest.DisplayName, resource.Spec.DisplayName)
	}
	if createRequest.OpcRetryToken == nil || *createRequest.OpcRetryToken != string(resource.UID) {
		t.Fatalf("create retry token = %v, want resource UID", createRequest.OpcRetryToken)
	}
	if getRequest.Scope != dnssdk.GetViewScopePrivate {
		t.Fatalf("get scope = %q, want PRIVATE", getRequest.Scope)
	}
	if getRequest.ViewId == nil || *getRequest.ViewId != "ocid1.dnsview.oc1..created" {
		t.Fatalf("get viewId = %v, want created view ID", getRequest.ViewId)
	}
	if resource.Status.Id != "ocid1.dnsview.oc1..created" {
		t.Fatalf("status.id = %q, want created view ID", resource.Status.Id)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.dnsview.oc1..created" {
		t.Fatalf("status.ocid = %q, want created view ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for active create", resource.Status.OsokStatus.Async.Current)
	}
}

//nolint:gocognit,gocyclo // The fake paginated list sequence is clearer when kept with the bind assertions.
func TestViewServiceClientCreateOrUpdateBindsExistingViewFromLaterListPage(t *testing.T) {
	t.Parallel()

	createCalls := 0
	getCalls := 0
	listCalls := 0
	var listPages []string

	client := testViewClient(&fakeViewOCIClient{
		createViewFn: func(_ context.Context, _ dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
			createCalls++
			return dnssdk.CreateViewResponse{}, nil
		},
		listViewsFn: func(_ context.Context, req dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error) {
			listCalls++
			if req.Scope != dnssdk.ListViewsScopePrivate {
				t.Fatalf("list scope = %q, want PRIVATE", req.Scope)
			}
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..example" {
				t.Fatalf("list compartmentId = %v, want spec compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "view-alpha" {
				t.Fatalf("list displayName = %v, want spec displayName", req.DisplayName)
			}
			listPages = append(listPages, stringValue(req.Page))
			if req.Page == nil {
				return dnssdk.ListViewsResponse{
					Items: []dnssdk.ViewSummary{
						makeSDKViewSummary(
							"ocid1.dnsview.oc1..other",
							"ocid1.compartment.oc1..example",
							"view-other",
							dnssdk.ViewSummaryLifecycleStateActive,
						),
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			return dnssdk.ListViewsResponse{
				Items: []dnssdk.ViewSummary{
					makeSDKViewSummary(
						"ocid1.dnsview.oc1..existing",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewSummaryLifecycleStateActive,
					),
				},
			}, nil
		},
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			getCalls++
			if req.Scope != dnssdk.GetViewScopePrivate {
				t.Fatalf("get scope = %q, want PRIVATE", req.Scope)
			}
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("get viewId = %v, want existing view ID", req.ViewId)
			}
			return dnssdk.GetViewResponse{
				View: makeSDKView(
					"ocid1.dnsview.oc1..existing",
					"ocid1.compartment.oc1..example",
					"view-alpha",
					dnssdk.ViewLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makeViewResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalls != 0 {
		t.Fatalf("CreateView() calls = %d, want 0", createCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListViews() calls = %d, want 2", listCalls)
	}
	if got, want := listPages, []string{"", "next-page"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list pages = %#v, want %#v", got, want)
	}
	if getCalls != 1 {
		t.Fatalf("GetView() calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != "ocid1.dnsview.oc1..existing" {
		t.Fatalf("status.id = %q, want existing view ID", resource.Status.Id)
	}
}

func TestViewServiceClientCreateOrUpdateRejectsDuplicateMatchesAcrossListPages(t *testing.T) {
	t.Parallel()

	createCalls := 0

	client := testViewClient(&fakeViewOCIClient{
		createViewFn: func(_ context.Context, _ dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
			createCalls++
			return dnssdk.CreateViewResponse{}, nil
		},
		listViewsFn: func(_ context.Context, req dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error) {
			if req.Page == nil {
				return dnssdk.ListViewsResponse{
					Items: []dnssdk.ViewSummary{
						makeSDKViewSummary(
							"ocid1.dnsview.oc1..first",
							"ocid1.compartment.oc1..example",
							"view-alpha",
							dnssdk.ViewSummaryLifecycleStateActive,
						),
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			return dnssdk.ListViewsResponse{
				Items: []dnssdk.ViewSummary{
					makeSDKViewSummary(
						"ocid1.dnsview.oc1..second",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewSummaryLifecycleStateActive,
					),
				},
			}, nil
		},
	})

	resource := makeViewResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want duplicate match error")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want multiple matching resources", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for duplicate matches")
	}
	if createCalls != 0 {
		t.Fatalf("CreateView() calls = %d, want 0", createCalls)
	}
}

func TestViewServiceClientCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			getCalls++
			if req.Scope != dnssdk.GetViewScopePrivate {
				t.Fatalf("get scope = %q, want PRIVATE", req.Scope)
			}
			return dnssdk.GetViewResponse{
				View: makeSDKView(
					"ocid1.dnsview.oc1..existing",
					"ocid1.compartment.oc1..example",
					"view-alpha",
					dnssdk.ViewLifecycleStateActive,
				),
			}, nil
		},
		updateViewFn: func(_ context.Context, _ dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error) {
			updateCalls++
			return dnssdk.UpdateViewResponse{}, nil
		},
	})

	resource := makeViewResource()
	resource.Status.Id = "ocid1.dnsview.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.dnsview.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if getCalls != 1 {
		t.Fatalf("GetView() calls = %d, want 1", getCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateView() calls = %d, want 0", updateCalls)
	}
	if resource.Status.DisplayName != "view-alpha" {
		t.Fatalf("status.displayName = %q, want view-alpha", resource.Status.DisplayName)
	}
}

//nolint:gocognit,gocyclo // The two-read update sequence is the behavior under test.
func TestViewServiceClientCreateOrUpdateUpdatesMutableDrift(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest dnssdk.UpdateViewRequest

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			getCalls++
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("get viewId = %v, want tracked view ID", req.ViewId)
			}
			switch getCalls {
			case 1:
				stale := makeSDKView(
					"ocid1.dnsview.oc1..existing",
					"ocid1.compartment.oc1..example",
					"view-old",
					dnssdk.ViewLifecycleStateActive,
				)
				stale.FreeformTags = map[string]string{"env": "old"}
				return dnssdk.GetViewResponse{View: stale}, nil
			case 2:
				return dnssdk.GetViewResponse{
					View: makeSDKView(
						"ocid1.dnsview.oc1..existing",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewLifecycleStateActive,
					),
				}, nil
			default:
				t.Fatalf("unexpected GetView() call %d", getCalls)
				return dnssdk.GetViewResponse{}, nil
			}
		},
		updateViewFn: func(_ context.Context, req dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error) {
			updateRequest = req
			return dnssdk.UpdateViewResponse{
				OpcRequestId: common.String("opc-update-1"),
				View: makeSDKView(
					"ocid1.dnsview.oc1..existing",
					"ocid1.compartment.oc1..example",
					"view-alpha",
					dnssdk.ViewLifecycleStateUpdating,
				),
			}, nil
		},
	})

	resource := makeViewResource()
	resource.Status.Id = "ocid1.dnsview.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once follow-up read sees ACTIVE")
	}
	if updateRequest.Scope != dnssdk.UpdateViewScopePrivate {
		t.Fatalf("update scope = %q, want PRIVATE", updateRequest.Scope)
	}
	if updateRequest.ViewId == nil || *updateRequest.ViewId != "ocid1.dnsview.oc1..existing" {
		t.Fatalf("update viewId = %v, want tracked view ID", updateRequest.ViewId)
	}
	if updateRequest.DisplayName == nil || *updateRequest.DisplayName != "view-alpha" {
		t.Fatalf("update displayName = %v, want view-alpha", updateRequest.DisplayName)
	}
	if got := updateRequest.FreeformTags; !reflect.DeepEqual(got, map[string]string{"env": "dev"}) {
		t.Fatalf("update freeformTags = %#v, want desired tags", got)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
}

func TestViewServiceClientCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	updateCalls := 0

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, _ dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			return dnssdk.GetViewResponse{
				View: makeSDKView(
					"ocid1.dnsview.oc1..existing",
					"ocid1.compartment.oc1..observed",
					"view-alpha",
					dnssdk.ViewLifecycleStateActive,
				),
			}, nil
		},
		updateViewFn: func(_ context.Context, _ dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error) {
			updateCalls++
			return dnssdk.UpdateViewResponse{}, nil
		},
	})

	resource := makeViewResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"
	resource.Status.Id = "ocid1.dnsview.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for create-only drift")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateView() calls = %d, want 0", updateCalls)
	}
}

//nolint:gocognit,gocyclo // The staged OCI lifecycle responses are easiest to review as one delete scenario.
func TestViewServiceClientDeleteRetainsFinalizerUntilLifecycleDeleteConfirmed(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			getCalls++
			if req.Scope != dnssdk.GetViewScopePrivate {
				t.Fatalf("get scope = %q, want PRIVATE", req.Scope)
			}
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("get viewId = %v, want tracked view ID", req.ViewId)
			}
			switch getCalls {
			case 1, 2:
				return dnssdk.GetViewResponse{
					View: makeSDKView(
						"ocid1.dnsview.oc1..existing",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewLifecycleStateActive,
					),
				}, nil
			case 3, 4:
				return dnssdk.GetViewResponse{
					View: makeSDKView(
						"ocid1.dnsview.oc1..existing",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewLifecycleStateDeleting,
					),
				}, nil
			case 5:
				return dnssdk.GetViewResponse{
					View: makeSDKView(
						"ocid1.dnsview.oc1..existing",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewLifecycleStateDeleted,
					),
				}, nil
			default:
				t.Fatalf("unexpected GetView() call %d", getCalls)
				return dnssdk.GetViewResponse{}, nil
			}
		},
		deleteViewFn: func(_ context.Context, req dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
			deleteCalls++
			if req.Scope != dnssdk.DeleteViewScopePrivate {
				t.Fatalf("delete scope = %q, want PRIVATE", req.Scope)
			}
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("delete viewId = %v, want tracked view ID", req.ViewId)
			}
			return dnssdk.DeleteViewResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	resource := makeViewResource()
	resource.Status.Id = "ocid1.dnsview.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.dnsview.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call should keep the finalizer while OCI reports DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteView() calls after first delete = %d, want 1", deleteCalls)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want delete tracker")
	}
	if resource.Status.OsokStatus.Async.Current.WorkRequestID != "wr-delete-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete-1", resource.Status.OsokStatus.Async.Current.WorkRequestID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call should release the finalizer after OCI reports DELETED")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteView() calls after second delete = %d, want no reissue", deleteCalls)
	}
	if resource.Status.LifecycleState != "DELETED" {
		t.Fatalf("status.lifecycleState after second delete = %q, want DELETED", resource.Status.LifecycleState)
	}
}

func TestViewServiceClientDeleteRejectsAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-delete-error-1"

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, _ dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			return dnssdk.GetViewResponse{
				View: makeSDKView(
					"ocid1.dnsview.oc1..existing",
					"ocid1.compartment.oc1..example",
					"view-alpha",
					dnssdk.ViewLifecycleStateActive,
				),
			}, nil
		},
		deleteViewFn: func(_ context.Context, _ dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
			return dnssdk.DeleteViewResponse{}, serviceErr
		},
	})

	resource := makeViewResource()
	resource.Status.Id = "ocid1.dnsview.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.dnsview.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestViewServiceClientDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			if req.Scope != dnssdk.GetViewScopePrivate {
				t.Fatalf("get scope = %q, want PRIVATE", req.Scope)
			}
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("get viewId = %v, want tracked view ID", req.ViewId)
			}
			return dnssdk.GetViewResponse{}, serviceErr
		},
		deleteViewFn: func(_ context.Context, _ dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
			deleteCalls++
			return dnssdk.DeleteViewResponse{}, nil
		},
	})

	resource := makeViewResource()
	resource.Status.Id = "ocid1.dnsview.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.dnsview.oc1..existing"
	resource.Status.LifecycleState = "DELETING"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want pre-delete auth-shaped GetView 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteView() calls = %d, want 0 after auth-shaped confirm read", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped confirm read", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-confirm-pre-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-confirm-pre-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

//nolint:gocognit,gocyclo // The post-delete read sequence keeps the delete/fatal-path assertions together.
func TestViewServiceClientDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-post-error-1"

	client := testViewClient(&fakeViewOCIClient{
		getViewFn: func(_ context.Context, req dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			getCalls++
			if req.Scope != dnssdk.GetViewScopePrivate {
				t.Fatalf("get scope = %q, want PRIVATE", req.Scope)
			}
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("get viewId = %v, want tracked view ID", req.ViewId)
			}
			switch getCalls {
			case 1, 2:
				return dnssdk.GetViewResponse{
					View: makeSDKView(
						"ocid1.dnsview.oc1..existing",
						"ocid1.compartment.oc1..example",
						"view-alpha",
						dnssdk.ViewLifecycleStateActive,
					),
				}, nil
			case 3:
				return dnssdk.GetViewResponse{}, serviceErr
			default:
				t.Fatalf("unexpected GetView() call %d", getCalls)
				return dnssdk.GetViewResponse{}, nil
			}
		},
		deleteViewFn: func(_ context.Context, req dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
			deleteCalls++
			if req.Scope != dnssdk.DeleteViewScopePrivate {
				t.Fatalf("delete scope = %q, want PRIVATE", req.Scope)
			}
			if req.ViewId == nil || *req.ViewId != "ocid1.dnsview.oc1..existing" {
				t.Fatalf("delete viewId = %v, want tracked view ID", req.ViewId)
			}
			return dnssdk.DeleteViewResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeViewResource()
	resource.Status.Id = "ocid1.dnsview.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.dnsview.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want post-delete auth-shaped GetView 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete confirm read")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteView() calls = %d, want 1 before post-delete confirm read", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped post-delete confirm read", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-confirm-post-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-confirm-post-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestViewServiceClientCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, "InternalError", "create failed")
	serviceErr.OpcRequestID = "opc-error-1"

	client := testViewClient(&fakeViewOCIClient{
		createViewFn: func(_ context.Context, _ dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
			return dnssdk.CreateViewResponse{}, serviceErr
		},
	})

	resource := makeViewResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for OCI error")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-error-1", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
