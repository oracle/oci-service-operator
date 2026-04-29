/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicecatalog

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	servicecatalogsdk "github.com/oracle/oci-go-sdk/v65/servicecatalog"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testServiceCatalogID     = "ocid1.servicecatalog.oc1..example"
	testServiceCatalogOther  = "ocid1.servicecatalog.oc1..other"
	testServiceCatalogCompID = "ocid1.compartment.oc1..example"
)

type fakeServiceCatalogOCIClient struct {
	createFn func(context.Context, servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error)
	getFn    func(context.Context, servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error)
	listFn   func(context.Context, servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error)
	updateFn func(context.Context, servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error)
	deleteFn func(context.Context, servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error)
}

func (f *fakeServiceCatalogOCIClient) CreateServiceCatalog(
	ctx context.Context,
	request servicecatalogsdk.CreateServiceCatalogRequest,
) (servicecatalogsdk.CreateServiceCatalogResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return servicecatalogsdk.CreateServiceCatalogResponse{}, nil
}

func (f *fakeServiceCatalogOCIClient) GetServiceCatalog(
	ctx context.Context,
	request servicecatalogsdk.GetServiceCatalogRequest,
) (servicecatalogsdk.GetServiceCatalogResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return servicecatalogsdk.GetServiceCatalogResponse{}, nil
}

func (f *fakeServiceCatalogOCIClient) ListServiceCatalogs(
	ctx context.Context,
	request servicecatalogsdk.ListServiceCatalogsRequest,
) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return servicecatalogsdk.ListServiceCatalogsResponse{}, nil
}

func (f *fakeServiceCatalogOCIClient) UpdateServiceCatalog(
	ctx context.Context,
	request servicecatalogsdk.UpdateServiceCatalogRequest,
) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return servicecatalogsdk.UpdateServiceCatalogResponse{}, nil
}

func (f *fakeServiceCatalogOCIClient) DeleteServiceCatalog(
	ctx context.Context,
	request servicecatalogsdk.DeleteServiceCatalogRequest,
) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return servicecatalogsdk.DeleteServiceCatalogResponse{}, nil
}

func testServiceCatalogClient(fake *fakeServiceCatalogOCIClient) ServiceCatalogServiceClient {
	return newServiceCatalogServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeServiceCatalogResource() *servicecatalogv1beta1.ServiceCatalog {
	return &servicecatalogv1beta1.ServiceCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "catalog-alpha",
			Namespace: "default",
			UID:       types.UID("servicecatalog-uid"),
		},
		Spec: servicecatalogv1beta1.ServiceCatalogSpec{
			CompartmentId: testServiceCatalogCompID,
			DisplayName:   "catalog-alpha",
			Status:        string(servicecatalogsdk.ServiceCatalogStatusEnumActive),
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKServiceCatalog(
	id string,
	compartmentID string,
	displayName string,
	status servicecatalogsdk.ServiceCatalogStatusEnumEnum,
	state servicecatalogsdk.ServiceCatalogLifecycleStateEnum,
) servicecatalogsdk.ServiceCatalog {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 30, 0, 0, time.UTC)}
	return servicecatalogsdk.ServiceCatalog{
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		Id:             common.String(id),
		LifecycleState: state,
		TimeCreated:    &created,
		TimeUpdated:    &updated,
		Status:         status,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func makeSDKServiceCatalogSummary(
	id string,
	compartmentID string,
	displayName string,
	status servicecatalogsdk.ServiceCatalogStatusEnumEnum,
	state servicecatalogsdk.ServiceCatalogLifecycleStateEnum,
) servicecatalogsdk.ServiceCatalogSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return servicecatalogsdk.ServiceCatalogSummary{
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		Id:             common.String(id),
		LifecycleState: state,
		TimeCreated:    &created,
		Status:         status,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func TestServiceCatalogRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := reviewedServiceCatalogRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedServiceCatalogRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "best-effort" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want best-effort confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	requireStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "status", "freeformTags", "definedTags"})
	requireStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
}

func TestServiceCatalogServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	var listRequest servicecatalogsdk.ListServiceCatalogsRequest
	var createRequest servicecatalogsdk.CreateServiceCatalogRequest
	var getRequest servicecatalogsdk.GetServiceCatalogRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		listFn: func(_ context.Context, request servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
			listCalls++
			listRequest = request
			return servicecatalogsdk.ListServiceCatalogsResponse{
				ServiceCatalogCollection: servicecatalogsdk.ServiceCatalogCollection{},
				OpcRequestId:             common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error) {
			createCalls++
			createRequest = request
			return servicecatalogsdk.CreateServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			getCalls++
			getRequest = request
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 1 || createCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	assertServiceCatalogCreateRequest(t, resource, listRequest, createRequest)
	requireStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	requireStringPtr(t, "get serviceCatalogId", getRequest.ServiceCatalogId, testServiceCatalogID)
	assertCreatedServiceCatalogStatus(t, resource)
}

func assertServiceCatalogCreateRequest(
	t *testing.T,
	resource *servicecatalogv1beta1.ServiceCatalog,
	listRequest servicecatalogsdk.ListServiceCatalogsRequest,
	createRequest servicecatalogsdk.CreateServiceCatalogRequest,
) {
	t.Helper()
	requireStringPtr(t, "list compartmentId", listRequest.CompartmentId, testServiceCatalogCompID)
	requireStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if listRequest.Status != "" {
		t.Fatalf("list status = %q, want empty so active resources can bind before status update", listRequest.Status)
	}
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, testServiceCatalogCompID)
	requireStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	if createRequest.Status != servicecatalogsdk.ServiceCatalogStatusEnumActive {
		t.Fatalf("create status = %q, want ACTIVE", createRequest.Status)
	}
	if !reflect.DeepEqual(createRequest.FreeformTags, map[string]string{"env": "dev"}) {
		t.Fatalf("create freeformTags = %#v, want env=dev", createRequest.FreeformTags)
	}
	if got := createRequest.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func assertCreatedServiceCatalogStatus(t *testing.T, resource *servicecatalogv1beta1.ServiceCatalog) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testServiceCatalogID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testServiceCatalogID)
	}
	if resource.Status.Id != testServiceCatalogID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testServiceCatalogID)
	}
	if resource.Status.Status != string(servicecatalogsdk.ServiceCatalogStatusEnumActive) {
		t.Fatalf("status.sdkStatus = %q, want ACTIVE", resource.Status.Status)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestServiceCatalogServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	var pages []string
	listCalls := 0
	getCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		listFn: func(_ context.Context, request servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
			listCalls++
			pages = append(pages, stringValue(request.Page))
			if listCalls == 1 {
				return servicecatalogsdk.ListServiceCatalogsResponse{
					ServiceCatalogCollection: servicecatalogsdk.ServiceCatalogCollection{
						Items: []servicecatalogsdk.ServiceCatalogSummary{
							makeSDKServiceCatalogSummary(
								testServiceCatalogOther,
								testServiceCatalogCompID,
								"other-catalog",
								servicecatalogsdk.ServiceCatalogStatusEnumActive,
								servicecatalogsdk.ServiceCatalogLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return servicecatalogsdk.ListServiceCatalogsResponse{
				ServiceCatalogCollection: servicecatalogsdk.ServiceCatalogCollection{
					Items: []servicecatalogsdk.ServiceCatalogSummary{
						makeSDKServiceCatalogSummary(
							testServiceCatalogID,
							testServiceCatalogCompID,
							resource.Spec.DisplayName,
							servicecatalogsdk.ServiceCatalogStatusEnumActive,
							servicecatalogsdk.ServiceCatalogLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			getCalls++
			requireStringPtr(t, "get serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
		createFn: func(context.Context, servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error) {
			t.Fatal("CreateServiceCatalog() called for existing catalog")
			return servicecatalogsdk.CreateServiceCatalogResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 2/1", listCalls, getCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testServiceCatalogID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testServiceCatalogID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestServiceCatalogServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	getCalls := 0
	updateCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			getCalls++
			requireStringPtr(t, "get serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdateServiceCatalogResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts get/update = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestServiceCatalogServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	resource.Spec.DisplayName = "catalog-renamed"
	resource.Spec.Status = string(servicecatalogsdk.ServiceCatalogStatusEnumInactive)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	getCalls := 0
	updateCalls := 0
	var updateRequest servicecatalogsdk.UpdateServiceCatalogRequest

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			getCalls++
			requireStringPtr(t, "get serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: mutableUpdateServiceCatalogReadback(resource, getCalls),
			}, nil
		},
		updateFn: func(_ context.Context, request servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
			updateCalls++
			updateRequest = request
			updated := makeSDKServiceCatalog(
				testServiceCatalogID,
				testServiceCatalogCompID,
				resource.Spec.DisplayName,
				servicecatalogsdk.ServiceCatalogStatusEnumInactive,
				servicecatalogsdk.ServiceCatalogLifecycleStateActive,
			)
			updated.FreeformTags = map[string]string{"env": "prod"}
			updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			return servicecatalogsdk.UpdateServiceCatalogResponse{
				ServiceCatalog: updated,
				OpcRequestId:   common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	assertServiceCatalogMutableUpdateRequest(t, resource, updateRequest)
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func mutableUpdateServiceCatalogReadback(
	resource *servicecatalogv1beta1.ServiceCatalog,
	getCalls int,
) servicecatalogsdk.ServiceCatalog {
	current := makeSDKServiceCatalog(
		testServiceCatalogID,
		testServiceCatalogCompID,
		"catalog-alpha",
		servicecatalogsdk.ServiceCatalogStatusEnumActive,
		servicecatalogsdk.ServiceCatalogLifecycleStateActive,
	)
	if getCalls <= 1 {
		return current
	}

	current.DisplayName = common.String(resource.Spec.DisplayName)
	current.Status = servicecatalogsdk.ServiceCatalogStatusEnumInactive
	current.FreeformTags = map[string]string{"env": "prod"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
	return current
}

func assertServiceCatalogMutableUpdateRequest(
	t *testing.T,
	resource *servicecatalogv1beta1.ServiceCatalog,
	updateRequest servicecatalogsdk.UpdateServiceCatalogRequest,
) {
	t.Helper()
	requireStringPtr(t, "update serviceCatalogId", updateRequest.ServiceCatalogId, testServiceCatalogID)
	requireStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	if updateRequest.Status != servicecatalogsdk.ServiceCatalogStatusEnumInactive {
		t.Fatalf("update status = %q, want INACTIVE", updateRequest.Status)
	}
	if !reflect.DeepEqual(updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
}

func TestServiceCatalogUpdatePreservesCurrentStatusWhenSpecOmitsStatus(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	resource.Spec.DisplayName = "catalog-renamed"
	resource.Spec.Status = ""
	var updateRequest servicecatalogsdk.UpdateServiceCatalogRequest

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					"catalog-alpha",
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, request servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
			updateRequest = request
			return servicecatalogsdk.UpdateServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
	})

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if updateRequest.Status != servicecatalogsdk.ServiceCatalogStatusEnumActive {
		t.Fatalf("update status = %q, want current ACTIVE", updateRequest.Status)
	}
}

func TestServiceCatalogServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	updateCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			requireStringPtr(t, "get serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					"ocid1.compartment.oc1..different",
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
			updateCalls++
			return servicecatalogsdk.UpdateServiceCatalogResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateServiceCatalog() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestServiceCatalogServiceClientRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	getCalls := 0
	deleteCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			getCalls++
			requireStringPtr(t, "get serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.DeleteServiceCatalogResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI still reports ACTIVE")
	}
	if getCalls != 3 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 3/1", getCalls, deleteCalls)
	}
	if resource.Status.LifecycleState != string(servicecatalogsdk.ServiceCatalogLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Terminating)
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", current)
	}
}

func TestServiceCatalogServiceClientMarksDeletedAfterUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	deleteCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(context.Context, servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			return servicecatalogsdk.GetServiceCatalogResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "catalog is gone")
		},
		deleteFn: func(context.Context, servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
			deleteCalls++
			return servicecatalogsdk.DeleteServiceCatalogResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteServiceCatalog() calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestServiceCatalogServiceClientTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	deleteCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(context.Context, servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			return servicecatalogsdk.GetServiceCatalogResponse{
				ServiceCatalog: makeSDKServiceCatalog(
					testServiceCatalogID,
					testServiceCatalogCompID,
					resource.Spec.DisplayName,
					servicecatalogsdk.ServiceCatalogStatusEnumActive,
					servicecatalogsdk.ServiceCatalogLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
			deleteCalls++
			return servicecatalogsdk.DeleteServiceCatalogResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"authorization or existence is ambiguous",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteServiceCatalog() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestServiceCatalogServiceClientRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.Id = testServiceCatalogID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	resource.Finalizers = []string{core.OSOKFinalizerName}
	deleteCalls := 0
	serviceErr := errortest.NewServiceError(
		404,
		errorutil.NotAuthorizedOrNotFound,
		"authorization or existence is ambiguous",
	)
	serviceErr.OpcRequestID = "opc-confirm-pre"

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(_ context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			requireStringPtr(t, "get serviceCatalogId", request.ServiceCatalogId, testServiceCatalogID)
			return servicecatalogsdk.GetServiceCatalogResponse{}, serviceErr
		},
		deleteFn: func(context.Context, servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
			deleteCalls++
			return servicecatalogsdk.DeleteServiceCatalogResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want pre-delete ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteServiceCatalog() calls = %d, want 0 after auth-shaped pre-delete confirm read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous pre-delete confirm-read classification", err)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed after auth-shaped pre-delete confirm read, want retained")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-confirm-pre" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-pre", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestServiceCatalogServiceClientTreatsAuthShapedConfirmReadConservatively(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceCatalogID)
	getCalls := 0

	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		getFn: func(context.Context, servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			getCalls++
			if getCalls == 1 {
				return servicecatalogsdk.GetServiceCatalogResponse{
					ServiceCatalog: makeSDKServiceCatalog(
						testServiceCatalogID,
						testServiceCatalogCompID,
						resource.Spec.DisplayName,
						servicecatalogsdk.ServiceCatalogStatusEnumActive,
						servicecatalogsdk.ServiceCatalogLifecycleStateActive,
					),
				}, nil
			}
			return servicecatalogsdk.GetServiceCatalogResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"authorization or existence is ambiguous",
			)
		},
		deleteFn: func(context.Context, servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
			return servicecatalogsdk.DeleteServiceCatalogResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestServiceCatalogServiceClientRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	client := testServiceCatalogClient(&fakeServiceCatalogOCIClient{
		listFn: func(context.Context, servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
			return servicecatalogsdk.ListServiceCatalogsResponse{}, nil
		},
		createFn: func(context.Context, servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error) {
			return servicecatalogsdk.CreateServiceCatalogResponse{}, errortest.NewServiceError(500, "InternalError", "service unavailable")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Failed)
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

func requireLastCondition(
	t *testing.T,
	resource *servicecatalogv1beta1.ServiceCatalog,
	want shared.OSOKConditionType,
) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil")
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
