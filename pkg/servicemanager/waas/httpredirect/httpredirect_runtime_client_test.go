/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package httpredirect

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testHttpRedirectID          = "ocid1.httpredirect.oc1..redirect"
	testHttpRedirectCompartment = "ocid1.compartment.oc1..compartment"
	testHttpRedirectName        = "redirect-sample"
	testHttpRedirectDomain      = "source.example.com"
)

type fakeHttpRedirectOCIClient struct {
	createFn      func(context.Context, waassdk.CreateHttpRedirectRequest) (waassdk.CreateHttpRedirectResponse, error)
	getFn         func(context.Context, waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error)
	listFn        func(context.Context, waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error)
	updateFn      func(context.Context, waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error)
	deleteFn      func(context.Context, waassdk.DeleteHttpRedirectRequest) (waassdk.DeleteHttpRedirectResponse, error)
	workRequestFn func(context.Context, waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error)
}

func (f *fakeHttpRedirectOCIClient) CreateHttpRedirect(ctx context.Context, req waassdk.CreateHttpRedirectRequest) (waassdk.CreateHttpRedirectResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return waassdk.CreateHttpRedirectResponse{}, nil
}

func (f *fakeHttpRedirectOCIClient) GetHttpRedirect(ctx context.Context, req waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return waassdk.GetHttpRedirectResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "HttpRedirect is missing")
}

func (f *fakeHttpRedirectOCIClient) ListHttpRedirects(ctx context.Context, req waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return waassdk.ListHttpRedirectsResponse{}, nil
}

func (f *fakeHttpRedirectOCIClient) UpdateHttpRedirect(ctx context.Context, req waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return waassdk.UpdateHttpRedirectResponse{}, nil
}

func (f *fakeHttpRedirectOCIClient) DeleteHttpRedirect(ctx context.Context, req waassdk.DeleteHttpRedirectRequest) (waassdk.DeleteHttpRedirectResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return waassdk.DeleteHttpRedirectResponse{}, nil
}

func (f *fakeHttpRedirectOCIClient) GetWorkRequest(ctx context.Context, req waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return waassdk.GetWorkRequestResponse{}, nil
}

func newTestHttpRedirectClient(client httpRedirectOCIClient) HttpRedirectServiceClient {
	return newHttpRedirectServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeHttpRedirectResource() *waasv1beta1.HttpRedirect {
	return &waasv1beta1.HttpRedirect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testHttpRedirectName,
			Namespace: "default",
		},
		Spec: waasv1beta1.HttpRedirectSpec{
			CompartmentId: testHttpRedirectCompartment,
			Domain:        testHttpRedirectDomain,
			Target: waasv1beta1.HttpRedirectTarget{
				Protocol: "https",
				Host:     "target.example.com",
				Path:     "/target",
				Query:    "?from=osok",
				Port:     8443,
			},
			DisplayName:  testHttpRedirectName,
			ResponseCode: 302,
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedHttpRedirectResource() *waasv1beta1.HttpRedirect {
	resource := makeHttpRedirectResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testHttpRedirectID)
	resource.Status.Id = testHttpRedirectID
	resource.Status.CompartmentId = testHttpRedirectCompartment
	resource.Status.Domain = testHttpRedirectDomain
	resource.Status.DisplayName = testHttpRedirectName
	resource.Status.LifecycleState = string(waassdk.LifecycleStatesActive)
	return resource
}

func makeHttpRedirectRequest(resource *waasv1beta1.HttpRedirect) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKHttpRedirect(
	t *testing.T,
	id string,
	spec waasv1beta1.HttpRedirectSpec,
	state waassdk.LifecycleStatesEnum,
) waassdk.HttpRedirect {
	t.Helper()
	target, err := httpRedirectTargetFromSpec(spec.Target)
	if err != nil {
		t.Fatalf("httpRedirectTargetFromSpec() error = %v", err)
	}
	return waassdk.HttpRedirect{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		Domain:         common.String(spec.Domain),
		Target:         target,
		ResponseCode:   common.Int(spec.ResponseCode),
		LifecycleState: state,
		FreeformTags:   cloneHttpRedirectStringMap(spec.FreeformTags),
		DefinedTags:    httpRedirectDefinedTags(spec.DefinedTags),
	}
}

func makeSDKHttpRedirectSummary(
	t *testing.T,
	id string,
	spec waasv1beta1.HttpRedirectSpec,
	state waassdk.LifecycleStatesEnum,
) waassdk.HttpRedirectSummary {
	t.Helper()
	target, err := httpRedirectTargetFromSpec(spec.Target)
	if err != nil {
		t.Fatalf("httpRedirectTargetFromSpec() error = %v", err)
	}
	return waassdk.HttpRedirectSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		Domain:         common.String(spec.Domain),
		Target:         target,
		ResponseCode:   common.Int(spec.ResponseCode),
		LifecycleState: state,
		FreeformTags:   cloneHttpRedirectStringMap(spec.FreeformTags),
		DefinedTags:    httpRedirectDefinedTags(spec.DefinedTags),
	}
}

func makeHttpRedirectWorkRequest(
	id string,
	operation waassdk.WorkRequestOperationTypesEnum,
	status waassdk.WorkRequestStatusValuesEnum,
	resourceID string,
) waassdk.WorkRequest {
	workRequest := waassdk.WorkRequest{
		Id:            common.String(id),
		OperationType: operation,
		Status:        status,
	}
	if resourceID != "" {
		workRequest.Resources = []waassdk.WorkRequestResource{
			{
				EntityType: common.String("HttpRedirect"),
				ActionType: httpRedirectActionForOperation(operation),
				Identifier: common.String(resourceID),
			},
		}
	}
	return workRequest
}

func seedHttpRedirectDeleteWorkRequest(resource *waasv1beta1.HttpRedirect) {
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		RawStatus:       string(waassdk.WorkRequestStatusValuesInProgress),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         httpRedirectDeletePendingMessage,
		UpdatedAt:       &now,
	}
}

func httpRedirectActionForOperation(operation waassdk.WorkRequestOperationTypesEnum) waassdk.WorkRequestResourceActionTypeEnum {
	switch operation {
	case waassdk.WorkRequestOperationTypesCreateHttpRedirect:
		return waassdk.WorkRequestResourceActionTypeCreated
	case waassdk.WorkRequestOperationTypesUpdateHttpRedirect:
		return waassdk.WorkRequestResourceActionTypeUpdated
	case waassdk.WorkRequestOperationTypesDeleteHttpRedirect:
		return waassdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

type pagedHttpRedirectBindScenario struct {
	t            *testing.T
	resource     *waasv1beta1.HttpRedirect
	createCalled bool
	updateCalled bool
	listCalls    int
	getCalls     int
}

func newPagedHttpRedirectBindScenario(
	t *testing.T,
	resource *waasv1beta1.HttpRedirect,
) *pagedHttpRedirectBindScenario {
	t.Helper()
	return &pagedHttpRedirectBindScenario{
		t:        t,
		resource: resource,
	}
}

func (s *pagedHttpRedirectBindScenario) client() HttpRedirectServiceClient {
	s.t.Helper()
	return newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		listFn:   s.listHttpRedirects,
		getFn:    s.getHttpRedirect,
		createFn: s.createHttpRedirect,
		updateFn: s.updateHttpRedirect,
	})
}

func (s *pagedHttpRedirectBindScenario) listHttpRedirects(
	_ context.Context,
	req waassdk.ListHttpRedirectsRequest,
) (waassdk.ListHttpRedirectsResponse, error) {
	s.t.Helper()
	s.listCalls++
	requireStringPtr(s.t, "ListHttpRedirectsRequest.CompartmentId", req.CompartmentId, s.resource.Spec.CompartmentId)
	if len(req.DisplayName) != 0 {
		s.t.Fatalf("ListHttpRedirectsRequest.DisplayName = %#v, want no slice filter", req.DisplayName)
	}
	if s.listCalls == 1 {
		return s.firstListHttpRedirectsPage(req), nil
	}
	return s.secondListHttpRedirectsPage(req), nil
}

func (s *pagedHttpRedirectBindScenario) firstListHttpRedirectsPage(
	req waassdk.ListHttpRedirectsRequest,
) waassdk.ListHttpRedirectsResponse {
	s.t.Helper()
	if req.Page != nil {
		s.t.Fatalf("first ListHttpRedirectsRequest.Page = %q, want nil", *req.Page)
	}
	otherSpec := s.resource.Spec
	otherSpec.Domain = "other.example.com"
	return waassdk.ListHttpRedirectsResponse{
		Items: []waassdk.HttpRedirectSummary{
			makeSDKHttpRedirectSummary(s.t, "ocid1.httpredirect.oc1..other", otherSpec, waassdk.LifecycleStatesActive),
		},
		OpcNextPage: common.String("page-2"),
	}
}

func (s *pagedHttpRedirectBindScenario) secondListHttpRedirectsPage(
	req waassdk.ListHttpRedirectsRequest,
) waassdk.ListHttpRedirectsResponse {
	s.t.Helper()
	requireStringPtr(s.t, "second ListHttpRedirectsRequest.Page", req.Page, "page-2")
	return waassdk.ListHttpRedirectsResponse{
		Items: []waassdk.HttpRedirectSummary{
			makeSDKHttpRedirectSummary(s.t, testHttpRedirectID, s.resource.Spec, waassdk.LifecycleStatesActive),
		},
	}
}

func (s *pagedHttpRedirectBindScenario) getHttpRedirect(
	_ context.Context,
	req waassdk.GetHttpRedirectRequest,
) (waassdk.GetHttpRedirectResponse, error) {
	s.t.Helper()
	s.getCalls++
	requireStringPtr(s.t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
	return waassdk.GetHttpRedirectResponse{
		HttpRedirect: makeSDKHttpRedirect(s.t, testHttpRedirectID, s.resource.Spec, waassdk.LifecycleStatesActive),
	}, nil
}

func (s *pagedHttpRedirectBindScenario) createHttpRedirect(
	context.Context,
	waassdk.CreateHttpRedirectRequest,
) (waassdk.CreateHttpRedirectResponse, error) {
	s.createCalled = true
	return waassdk.CreateHttpRedirectResponse{}, nil
}

func (s *pagedHttpRedirectBindScenario) updateHttpRedirect(
	context.Context,
	waassdk.UpdateHttpRedirectRequest,
) (waassdk.UpdateHttpRedirectResponse, error) {
	s.updateCalled = true
	return waassdk.UpdateHttpRedirectResponse{}, nil
}

func (s *pagedHttpRedirectBindScenario) requireBoundResource(
	resource *waasv1beta1.HttpRedirect,
	response servicemanager.OSOKResponse,
) {
	s.t.Helper()
	if !response.IsSuccessful {
		s.t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if s.createCalled {
		s.t.Fatal("CreateHttpRedirect() called for existing redirect")
	}
	if s.updateCalled {
		s.t.Fatal("UpdateHttpRedirect() called for matching redirect")
	}
	if s.listCalls != 2 {
		s.t.Fatalf("ListHttpRedirects() calls = %d, want 2 paginated calls", s.listCalls)
	}
	if s.getCalls != 1 {
		s.t.Fatalf("GetHttpRedirect() calls = %d, want 1", s.getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testHttpRedirectID {
		s.t.Fatalf("status.status.ocid = %q, want %q", got, testHttpRedirectID)
	}
	requireLastCondition(s.t, resource, shared.Active)
}

func TestHttpRedirectCreateOrUpdateBindsExistingRedirectByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeHttpRedirectResource()
	scenario := newPagedHttpRedirectBindScenario(t, resource)

	response, err := scenario.client().CreateOrUpdate(context.Background(), resource, makeHttpRedirectRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	scenario.requireBoundResource(resource, response)
}

func TestHttpRedirectCreateRecordsPayloadWorkRequestRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeHttpRedirectResource()
	listCalls := 0
	createCalls := 0
	workRequestCalls := 0
	getCalls := 0

	client := newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		listFn: func(context.Context, waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error) {
			listCalls++
			return waassdk.ListHttpRedirectsResponse{}, nil
		},
		createFn: func(_ context.Context, req waassdk.CreateHttpRedirectRequest) (waassdk.CreateHttpRedirectResponse, error) {
			createCalls++
			requireHttpRedirectCreateRequest(t, req, resource)
			return waassdk.CreateHttpRedirectResponse{
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-create")
			return waassdk.GetWorkRequestResponse{
				WorkRequest: makeHttpRedirectWorkRequest("wr-create", waassdk.WorkRequestOperationTypesCreateHttpRedirect, waassdk.WorkRequestStatusValuesSucceeded, testHttpRedirectID),
			}, nil
		},
		getFn: func(_ context.Context, req waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
			getCalls++
			requireStringPtr(t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
			return waassdk.GetHttpRedirectResponse{
				HttpRedirect: makeSDKHttpRedirect(t, testHttpRedirectID, resource.Spec, waassdk.LifecycleStatesActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeHttpRedirectRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListHttpRedirects() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 || workRequestCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts create/workRequest/get = %d/%d/%d, want 1/1/1", createCalls, workRequestCalls, getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testHttpRedirectID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testHttpRedirectID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after succeeded work request", resource.Status.OsokStatus.Async.Current)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestHttpRedirectCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedHttpRedirectResource()
	updateCalled := false

	client := newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		getFn: func(_ context.Context, req waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
			requireStringPtr(t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
			return waassdk.GetHttpRedirectResponse{
				HttpRedirect: makeSDKHttpRedirect(t, testHttpRedirectID, resource.Spec, waassdk.LifecycleStatesActive),
			}, nil
		},
		updateFn: func(context.Context, waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error) {
			updateCalled = true
			return waassdk.UpdateHttpRedirectResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeHttpRedirectRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateHttpRedirect() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestHttpRedirectCreateOrUpdateUpdatesMutableFieldsThroughWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedHttpRedirectResource()
	resource.Spec.DisplayName = "updated-redirect"
	resource.Spec.Target.Host = "updated.example.com"
	resource.Spec.Target.Path = "/updated"
	resource.Spec.ResponseCode = 301
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := resource.Spec
	currentSpec.DisplayName = testHttpRedirectName
	currentSpec.Target.Host = "target.example.com"
	currentSpec.Target.Path = "/target"
	currentSpec.ResponseCode = 302
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0

	client := newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		getFn: func(_ context.Context, req waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
			getCalls++
			requireStringPtr(t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
			if getCalls == 1 {
				return waassdk.GetHttpRedirectResponse{
					HttpRedirect: makeSDKHttpRedirect(t, testHttpRedirectID, currentSpec, waassdk.LifecycleStatesActive),
				}, nil
			}
			return waassdk.GetHttpRedirectResponse{
				HttpRedirect: makeSDKHttpRedirect(t, testHttpRedirectID, resource.Spec, waassdk.LifecycleStatesActive),
			}, nil
		},
		updateFn: func(_ context.Context, req waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
			requireStringPtr(t, "UpdateHttpRedirectDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireSDKTarget(t, "UpdateHttpRedirectDetails.Target", req.Target, resource.Spec.Target)
			requireIntPtr(t, "UpdateHttpRedirectDetails.ResponseCode", req.ResponseCode, resource.Spec.ResponseCode)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateHttpRedirectDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateHttpRedirectDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return waassdk.UpdateHttpRedirectResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update")
			return waassdk.GetWorkRequestResponse{
				WorkRequest: makeHttpRedirectWorkRequest("wr-update", waassdk.WorkRequestOperationTypesUpdateHttpRedirect, waassdk.WorkRequestStatusValuesSucceeded, testHttpRedirectID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeHttpRedirectRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 || workRequestCalls != 1 || getCalls != 2 {
		t.Fatalf("call counts update/workRequest/get = %d/%d/%d, want 1/1/2", updateCalls, workRequestCalls, getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestHttpRedirectCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mutateSpec func(*waasv1beta1.HttpRedirect)
		wantError  string
	}{
		{
			name: "changed compartmentId",
			mutateSpec: func(resource *waasv1beta1.HttpRedirect) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			wantError: "compartmentId changes",
		},
		{
			name: "omitted domain",
			mutateSpec: func(resource *waasv1beta1.HttpRedirect) {
				resource.Spec.Domain = ""
			},
			wantError: "domain changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedHttpRedirectResource()
			currentSpec := resource.Spec
			resource.Spec.DisplayName = "updated-redirect"
			tt.mutateSpec(resource)
			updateCalled := false

			client := newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
				getFn: func(_ context.Context, req waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
					requireStringPtr(t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
					return waassdk.GetHttpRedirectResponse{
						HttpRedirect: makeSDKHttpRedirect(t, testHttpRedirectID, currentSpec, waassdk.LifecycleStatesActive),
					}, nil
				},
				updateFn: func(context.Context, waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error) {
					updateCalled = true
					return waassdk.UpdateHttpRedirectResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, makeHttpRedirectRequest(resource))
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() successful = true, want false")
			}
			if updateCalled {
				t.Fatal("UpdateHttpRedirect() called after create-only drift")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tt.wantError)
			}
			requireLastCondition(t, resource, shared.Failed)
		})
	}
}

type deleteHttpRedirectWorkRequestScenario struct {
	t                *testing.T
	resource         *waasv1beta1.HttpRedirect
	getCalls         int
	deleteCalls      int
	workRequestCalls int
}

func newDeleteHttpRedirectWorkRequestScenario(
	t *testing.T,
	resource *waasv1beta1.HttpRedirect,
) *deleteHttpRedirectWorkRequestScenario {
	t.Helper()
	return &deleteHttpRedirectWorkRequestScenario{
		t:        t,
		resource: resource,
	}
}

func (s *deleteHttpRedirectWorkRequestScenario) client() HttpRedirectServiceClient {
	s.t.Helper()
	return newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		getFn:         s.getHttpRedirect,
		deleteFn:      s.deleteHttpRedirect,
		workRequestFn: s.getWorkRequest,
	})
}

func (s *deleteHttpRedirectWorkRequestScenario) getHttpRedirect(
	_ context.Context,
	req waassdk.GetHttpRedirectRequest,
) (waassdk.GetHttpRedirectResponse, error) {
	s.t.Helper()
	s.getCalls++
	requireStringPtr(s.t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
	if s.getCalls <= 2 {
		return waassdk.GetHttpRedirectResponse{
			HttpRedirect: makeSDKHttpRedirect(s.t, testHttpRedirectID, s.resource.Spec, waassdk.LifecycleStatesActive),
		}, nil
	}
	return waassdk.GetHttpRedirectResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "HttpRedirect is gone")
}

func (s *deleteHttpRedirectWorkRequestScenario) deleteHttpRedirect(
	_ context.Context,
	req waassdk.DeleteHttpRedirectRequest,
) (waassdk.DeleteHttpRedirectResponse, error) {
	s.t.Helper()
	s.deleteCalls++
	requireStringPtr(s.t, "DeleteHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
	return waassdk.DeleteHttpRedirectResponse{
		OpcWorkRequestId: common.String("wr-delete"),
		OpcRequestId:     common.String("opc-delete"),
	}, nil
}

func (s *deleteHttpRedirectWorkRequestScenario) getWorkRequest(
	_ context.Context,
	req waassdk.GetWorkRequestRequest,
) (waassdk.GetWorkRequestResponse, error) {
	s.t.Helper()
	s.workRequestCalls++
	requireStringPtr(s.t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
	return waassdk.GetWorkRequestResponse{
		WorkRequest: makeHttpRedirectWorkRequest(
			"wr-delete",
			waassdk.WorkRequestOperationTypesDeleteHttpRedirect,
			s.workRequestStatus(),
			testHttpRedirectID,
		),
	}, nil
}

func (s *deleteHttpRedirectWorkRequestScenario) workRequestStatus() waassdk.WorkRequestStatusValuesEnum {
	if s.workRequestCalls > 1 {
		return waassdk.WorkRequestStatusValuesSucceeded
	}
	return waassdk.WorkRequestStatusValuesInProgress
}

func (s *deleteHttpRedirectWorkRequestScenario) requireFirstDeletePending(
	resource *waasv1beta1.HttpRedirect,
	deleted bool,
	err error,
) {
	s.t.Helper()
	if err != nil {
		s.t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		s.t.Fatal("Delete() first deleted = true, want false while work request is pending")
	}
	if s.deleteCalls != 1 {
		s.t.Fatalf("DeleteHttpRedirect() calls after first delete = %d, want 1", s.deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		s.t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireWorkRequestStatus(s.t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireLastCondition(s.t, resource, shared.Terminating)
}

func (s *deleteHttpRedirectWorkRequestScenario) requireSecondDeleteConfirmed(
	resource *waasv1beta1.HttpRedirect,
	deleted bool,
	err error,
) {
	s.t.Helper()
	if err != nil {
		s.t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		s.t.Fatal("Delete() second deleted = false, want true after work request and NotFound confirmation")
	}
	if s.deleteCalls != 1 {
		s.t.Fatalf("DeleteHttpRedirect() calls after confirmed delete = %d, want still 1", s.deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		s.t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(s.t, resource, shared.Terminating)
}

func TestHttpRedirectDeleteTracksWorkRequestUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedHttpRedirectResource()
	scenario := newDeleteHttpRedirectWorkRequestScenario(t, resource)
	client := scenario.client()

	deleted, err := client.Delete(context.Background(), resource)
	scenario.requireFirstDeletePending(resource, deleted, err)

	deleted, err = client.Delete(context.Background(), resource)
	scenario.requireSecondDeleteConfirmed(resource, deleted, err)
}

type succeededDeleteReadbackScenario struct {
	t                *testing.T
	resource         *waasv1beta1.HttpRedirect
	getCalls         int
	deleteCalls      int
	workRequestCalls int
}

func newSucceededDeleteReadbackScenario(
	t *testing.T,
	resource *waasv1beta1.HttpRedirect,
) *succeededDeleteReadbackScenario {
	t.Helper()
	return &succeededDeleteReadbackScenario{
		t:        t,
		resource: resource,
	}
}

func (s *succeededDeleteReadbackScenario) client() HttpRedirectServiceClient {
	s.t.Helper()
	return newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		getFn:         s.getHttpRedirect,
		deleteFn:      s.deleteHttpRedirect,
		workRequestFn: s.getWorkRequest,
	})
}

func (s *succeededDeleteReadbackScenario) getHttpRedirect(
	_ context.Context,
	req waassdk.GetHttpRedirectRequest,
) (waassdk.GetHttpRedirectResponse, error) {
	s.t.Helper()
	s.getCalls++
	requireStringPtr(s.t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
	if s.getCalls == 1 {
		return waassdk.GetHttpRedirectResponse{
			HttpRedirect: makeSDKHttpRedirect(s.t, testHttpRedirectID, s.resource.Spec, waassdk.LifecycleStatesActive),
		}, nil
	}
	return waassdk.GetHttpRedirectResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "HttpRedirect is gone")
}

func (s *succeededDeleteReadbackScenario) deleteHttpRedirect(
	context.Context,
	waassdk.DeleteHttpRedirectRequest,
) (waassdk.DeleteHttpRedirectResponse, error) {
	s.deleteCalls++
	return waassdk.DeleteHttpRedirectResponse{}, nil
}

func (s *succeededDeleteReadbackScenario) getWorkRequest(
	_ context.Context,
	req waassdk.GetWorkRequestRequest,
) (waassdk.GetWorkRequestResponse, error) {
	s.t.Helper()
	s.workRequestCalls++
	requireStringPtr(s.t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
	return waassdk.GetWorkRequestResponse{
		WorkRequest: makeHttpRedirectWorkRequest(
			"wr-delete",
			waassdk.WorkRequestOperationTypesDeleteHttpRedirect,
			waassdk.WorkRequestStatusValuesSucceeded,
			testHttpRedirectID,
		),
	}, nil
}

func (s *succeededDeleteReadbackScenario) requireLiveReadbackWaited(
	resource *waasv1beta1.HttpRedirect,
	deleted bool,
	err error,
) {
	s.t.Helper()
	if err != nil {
		s.t.Fatalf("Delete() live readback error = %v, want nil", err)
	}
	if deleted {
		s.t.Fatal("Delete() live readback deleted = true, want false")
	}
	if s.deleteCalls != 0 {
		s.t.Fatalf("DeleteHttpRedirect() calls = %d, want 0 while delete work request is tracked", s.deleteCalls)
	}
	if s.getCalls != 1 || s.workRequestCalls != 1 {
		s.t.Fatalf("call counts get/workRequest = %d/%d, want 1/1", s.getCalls, s.workRequestCalls)
	}
	requireWorkRequestStatus(s.t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireLastCondition(s.t, resource, shared.Terminating)
}

func (s *succeededDeleteReadbackScenario) requireNotFoundConfirmed(
	resource *waasv1beta1.HttpRedirect,
	deleted bool,
	err error,
) {
	s.t.Helper()
	if err != nil {
		s.t.Fatalf("Delete() NotFound confirmation error = %v", err)
	}
	if !deleted {
		s.t.Fatal("Delete() NotFound confirmation deleted = false, want true")
	}
	if s.deleteCalls != 0 {
		s.t.Fatalf("DeleteHttpRedirect() calls after NotFound = %d, want 0", s.deleteCalls)
	}
	if s.getCalls != 2 || s.workRequestCalls != 2 {
		s.t.Fatalf("final call counts get/workRequest = %d/%d, want 2/2", s.getCalls, s.workRequestCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		s.t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		s.t.Fatalf("status.status.async.current = %#v, want nil after NotFound confirmation", resource.Status.OsokStatus.Async.Current)
	}
	requireLastCondition(s.t, resource, shared.Terminating)
}

func TestHttpRedirectDeleteWaitsWhenSucceededWorkRequestReadbackStillLive(t *testing.T) {
	t.Parallel()

	resource := makeTrackedHttpRedirectResource()
	seedHttpRedirectDeleteWorkRequest(resource)
	scenario := newSucceededDeleteReadbackScenario(t, resource)
	client := scenario.client()

	deleted, err := client.Delete(context.Background(), resource)
	scenario.requireLiveReadbackWaited(resource, deleted, err)

	deleted, err = client.Delete(context.Background(), resource)
	scenario.requireNotFoundConfirmed(resource, deleted, err)
}

func TestHttpRedirectDeleteRejectsAuthShapedPreDeleteReadBeforeOCIRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedHttpRedirectResource()
	deleteCalled := false
	client := newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		getFn: func(_ context.Context, req waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
			requireStringPtr(t, "GetHttpRedirectRequest.HttpRedirectId", req.HttpRedirectId, testHttpRedirectID)
			return waassdk.GetHttpRedirectResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, waassdk.DeleteHttpRedirectRequest) (waassdk.DeleteHttpRedirectResponse, error) {
			deleteCalled = true
			return waassdk.DeleteHttpRedirectResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous readback rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous readback")
	}
	if deleteCalled {
		t.Fatal("DeleteHttpRedirect() called after auth-shaped pre-delete read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestHttpRedirectCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeHttpRedirectResource()
	client := newTestHttpRedirectClient(&fakeHttpRedirectOCIClient{
		listFn: func(context.Context, waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error) {
			return waassdk.ListHttpRedirectsResponse{}, nil
		},
		createFn: func(context.Context, waassdk.CreateHttpRedirectRequest) (waassdk.CreateHttpRedirectResponse, error) {
			return waassdk.CreateHttpRedirectResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeHttpRedirectRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireHttpRedirectCreateRequest(
	t *testing.T,
	req waassdk.CreateHttpRedirectRequest,
	resource *waasv1beta1.HttpRedirect,
) {
	t.Helper()
	requireStringPtr(t, "CreateHttpRedirectDetails.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateHttpRedirectDetails.Domain", req.Domain, resource.Spec.Domain)
	requireSDKTarget(t, "CreateHttpRedirectDetails.Target", req.Target, resource.Spec.Target)
	requireStringPtr(t, "CreateHttpRedirectDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireIntPtr(t, "CreateHttpRedirectDetails.ResponseCode", req.ResponseCode, resource.Spec.ResponseCode)
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateHttpRedirectDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateHttpRedirectDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateHttpRedirectRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireSDKTarget(
	t *testing.T,
	field string,
	target *waassdk.HttpRedirectTarget,
	spec waasv1beta1.HttpRedirectTarget,
) {
	t.Helper()
	if target == nil {
		t.Fatalf("%s = nil, want target", field)
	}
	if got, want := target.Protocol, waassdk.HttpRedirectTargetProtocolEnum(strings.ToUpper(spec.Protocol)); got != want {
		t.Fatalf("%s.Protocol = %q, want %q", field, got, want)
	}
	requireStringPtr(t, field+".Host", target.Host, spec.Host)
	requireStringPtr(t, field+".Path", target.Path, spec.Path)
	requireStringPtr(t, field+".Query", target.Query, spec.Query)
	if spec.Port == 0 {
		if target.Port != nil {
			t.Fatalf("%s.Port = %d, want nil", field, *target.Port)
		}
		return
	}
	requireIntPtr(t, field+".Port", target.Port, spec.Port)
}

func requireStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}

func requireIntPtr(t *testing.T, field string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", field, *got, want)
	}
}

func requireLastCondition(t *testing.T, resource *waasv1beta1.HttpRedirect, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = empty, want last condition %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func requireWorkRequestStatus(
	t *testing.T,
	resource *waasv1beta1.HttpRedirect,
	wantPhase shared.OSOKAsyncPhase,
	wantID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %q, want workrequest", current.Source)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantID)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}
