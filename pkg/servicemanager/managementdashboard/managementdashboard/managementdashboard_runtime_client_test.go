/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementdashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	managementdashboardsdk "github.com/oracle/oci-go-sdk/v65/managementdashboard"
	managementdashboardv1beta1 "github.com/oracle/oci-service-operator/api/managementdashboard/v1beta1"
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
	testManagementDashboardID            = "ocid1.managementdashboard.oc1..dashboard"
	testManagementDashboardCompartmentID = "ocid1.compartment.oc1..dashboard"
)

func TestManagementDashboardRuntimeHooksConfigured(t *testing.T) {
	hooks := newManagementDashboardDefaultRuntimeHooks(managementdashboardsdk.DashxApisClient{})
	applyManagementDashboardRuntimeHooks(nil, &hooks)

	assertManagementDashboardRuntimeHooksConfigured(t, hooks)
	assertManagementDashboardCreateBodyPreservesFalse(t, hooks)
}

func assertManagementDashboardRuntimeHooksConfigured(t *testing.T, hooks ManagementDashboardRuntimeHooks) {
	t.Helper()

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want bool-preserving create builder")
	}
	if hooks.DeleteHooks.ConfirmRead == nil {
		t.Fatal("hooks.DeleteHooks.ConfirmRead = nil, want conservative delete confirm read")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
	if hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("hooks.DeleteHooks.ApplyOutcome = nil, want auth-shaped confirm read guard")
	}
}

func assertManagementDashboardCreateBodyPreservesFalse(t *testing.T, hooks ManagementDashboardRuntimeHooks) {
	t.Helper()

	body, err := hooks.BuildCreateBody(context.Background(), testManagementDashboardResource(t), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	values, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want map[string]any", body)
	}
	if got, ok := values["isShowInHome"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() isShowInHome = %#v, present = %t, want explicit false", got, ok)
	}
	if got, ok := values["isFavorite"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() isFavorite = %#v, present = %t, want explicit false", got, ok)
	}
}

func TestManagementDashboardCreateRecordsIdentityAndRequestID(t *testing.T) {
	resource := testManagementDashboardResource(t)
	dashboard := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	fake := &fakeManagementDashboardOCIClient{
		listManagementDashboards: func(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
			return managementdashboardsdk.ListManagementDashboardsResponse{}, nil
		},
		createManagementDashboard: func(_ context.Context, request managementdashboardsdk.CreateManagementDashboardRequest) (managementdashboardsdk.CreateManagementDashboardResponse, error) {
			assertManagementDashboardCreateRequest(t, request)
			return managementdashboardsdk.CreateManagementDashboardResponse{
				ManagementDashboard: dashboard,
				OpcRequestId:        common.String("opc-create"),
				OpcWorkRequestId:    common.String("wr-create"),
			}, nil
		},
		getManagementDashboard: func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			return managementdashboardsdk.GetManagementDashboardResponse{
				ManagementDashboard: dashboard,
				OpcRequestId:        common.String("opc-get"),
			}, nil
		},
	}

	response, err := newTestManagementDashboardClient(fake).CreateOrUpdate(context.Background(), resource, testManagementDashboardRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulManagementDashboardResponse(t, response)
	assertManagementDashboardCallCount(t, "CreateManagementDashboard()", fake.createCalls, 1)
	assertManagementDashboardRecordedID(t, resource, testManagementDashboardID)
	assertManagementDashboardOpcRequestID(t, resource, "opc-create")
	assertManagementDashboardAsyncCurrentNil(t, resource)
}

func assertManagementDashboardCreateRequest(t *testing.T, request managementdashboardsdk.CreateManagementDashboardRequest) {
	t.Helper()

	assertBoolPointerFalse(t, "CreateManagementDashboard() IsShowInHome", request.IsShowInHome)
	assertBoolPointerFalse(t, "CreateManagementDashboard() IsFavorite", request.IsFavorite)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateManagementDashboard() OpcRetryToken is empty, want deterministic retry token")
	}
}

func TestManagementDashboardCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := testManagementDashboardResource(t)
	dashboard := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	var listPages []string
	fake := &fakeManagementDashboardOCIClient{
		listManagementDashboards: func(_ context.Context, request managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
			listPages = append(listPages, managementDashboardStringValue(request.Page))
			if request.Page == nil {
				return managementdashboardsdk.ListManagementDashboardsResponse{
					ManagementDashboardCollection: managementdashboardsdk.ManagementDashboardCollection{
						Items: []managementdashboardsdk.ManagementDashboardSummary{
							sdkManagementDashboardSummaryFromResource(t, resource, "ocid1.managementdashboard.oc1..other", "other-dashboard"),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return managementdashboardsdk.ListManagementDashboardsResponse{
				ManagementDashboardCollection: managementdashboardsdk.ManagementDashboardCollection{
					Items: []managementdashboardsdk.ManagementDashboardSummary{
						sdkManagementDashboardSummaryFromResource(t, resource, testManagementDashboardID, resource.Spec.DisplayName),
					},
				},
			}, nil
		},
		getManagementDashboard: func(_ context.Context, request managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			if got := managementDashboardStringValue(request.ManagementDashboardId); got != testManagementDashboardID {
				t.Fatalf("GetManagementDashboard() id = %q, want %q", got, testManagementDashboardID)
			}
			return managementdashboardsdk.GetManagementDashboardResponse{ManagementDashboard: dashboard}, nil
		},
	}

	response, err := newTestManagementDashboardClient(fake).CreateOrUpdate(context.Background(), resource, testManagementDashboardRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateManagementDashboard() calls = %d, want 0 for bind", fake.createCalls)
	}
	if got := strings.Join(listPages, ","); got != ",page-2" {
		t.Fatalf("ListManagementDashboards() pages = %q, want \",page-2\"", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testManagementDashboardID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testManagementDashboardID)
	}
}

func TestManagementDashboardCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := testManagementDashboardResource(t)
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementDashboardID)
	dashboard := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	fake := &fakeManagementDashboardOCIClient{
		getManagementDashboard: func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			return managementdashboardsdk.GetManagementDashboardResponse{ManagementDashboard: dashboard}, nil
		},
	}

	response, err := newTestManagementDashboardClient(fake).CreateOrUpdate(context.Background(), resource, testManagementDashboardRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateManagementDashboard() calls = %d, want 0", fake.updateCalls)
	}
}

func TestManagementDashboardMutableUpdateUsesUpdatePath(t *testing.T) {
	resource := testManagementDashboardResource(t)
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementDashboardID)
	current := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	current.Description = common.String("old description")
	current.IsFavorite = common.Bool(true)
	updated := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	getResponses := []managementdashboardsdk.GetManagementDashboardResponse{
		{ManagementDashboard: current},
		{ManagementDashboard: updated},
	}
	fake := &fakeManagementDashboardOCIClient{
		getManagementDashboard: managementDashboardGetResponses(t, &getResponses),
		updateManagementDashboard: func(_ context.Context, request managementdashboardsdk.UpdateManagementDashboardRequest) (managementdashboardsdk.UpdateManagementDashboardResponse, error) {
			assertManagementDashboardUpdateRequest(t, request, resource)
			return managementdashboardsdk.UpdateManagementDashboardResponse{
				ManagementDashboard: updated,
				OpcRequestId:        common.String("opc-update"),
			}, nil
		},
	}

	response, err := newTestManagementDashboardClient(fake).CreateOrUpdate(context.Background(), resource, testManagementDashboardRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulManagementDashboardResponse(t, response)
	assertManagementDashboardCallCount(t, "UpdateManagementDashboard()", fake.updateCalls, 1)
	assertManagementDashboardOpcRequestID(t, resource, "opc-update")
}

func managementDashboardGetResponses(
	t *testing.T,
	responses *[]managementdashboardsdk.GetManagementDashboardResponse,
) func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
	t.Helper()

	return func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
		if len(*responses) == 0 {
			t.Fatal("GetManagementDashboard() called more times than expected")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func assertManagementDashboardUpdateRequest(
	t *testing.T,
	request managementdashboardsdk.UpdateManagementDashboardRequest,
	resource *managementdashboardv1beta1.ManagementDashboard,
) {
	t.Helper()

	if got := managementDashboardStringValue(request.ManagementDashboardId); got != testManagementDashboardID {
		t.Fatalf("UpdateManagementDashboard() id = %q, want %q", got, testManagementDashboardID)
	}
	if got := managementDashboardStringValue(request.Description); got != resource.Spec.Description {
		t.Fatalf("UpdateManagementDashboard() description = %q, want %q", got, resource.Spec.Description)
	}
	assertBoolPointerFalse(t, "UpdateManagementDashboard() isFavorite", request.IsFavorite)
	if request.CompartmentId != nil {
		t.Fatalf("UpdateManagementDashboard() compartmentId = %q, want nil because compartment moves are not handled by update", *request.CompartmentId)
	}
}

func assertSuccessfulManagementDashboardResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
}

func assertManagementDashboardCallCount(t *testing.T, operation string, got, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", operation, got, want)
	}
}

func assertManagementDashboardRecordedID(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementDashboard,
	want string,
) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertManagementDashboardOpcRequestID(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementDashboard,
	want string,
) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertManagementDashboardAsyncCurrentNil(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementDashboard,
) {
	t.Helper()

	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after ACTIVE follow-up", resource.Status.OsokStatus.Async.Current)
	}
}

func assertBoolPointerFalse(t *testing.T, label string, value *bool) {
	t.Helper()

	if value == nil || *value {
		t.Fatalf("%s = %#v, want explicit false", label, value)
	}
}

func TestManagementDashboardRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := testManagementDashboardResource(t)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementDashboardID)
	current := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	current.CompartmentId = common.String(testManagementDashboardCompartmentID)
	fake := &fakeManagementDashboardOCIClient{
		getManagementDashboard: func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			return managementdashboardsdk.GetManagementDashboardResponse{ManagementDashboard: current}, nil
		},
	}

	response, err := newTestManagementDashboardClient(fake).CreateOrUpdate(context.Background(), resource, testManagementDashboardRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId context", err.Error())
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateManagementDashboard() calls = %d, want 0", fake.updateCalls)
	}
}

func TestManagementDashboardDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := testManagementDashboardResource(t)
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementDashboardID)
	active := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	getCalls := 0
	fake := &fakeManagementDashboardOCIClient{
		getManagementDashboard: func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			getCalls++
			if getCalls == 1 {
				return managementdashboardsdk.GetManagementDashboardResponse{ManagementDashboard: active}, nil
			}
			return managementdashboardsdk.GetManagementDashboardResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "dashboard not found")
		},
		deleteManagementDashboard: func(context.Context, managementdashboardsdk.DeleteManagementDashboardRequest) (managementdashboardsdk.DeleteManagementDashboardResponse, error) {
			return managementdashboardsdk.DeleteManagementDashboardResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestManagementDashboardClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found confirmation")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeleteManagementDashboard() calls = %d, want 1", fake.deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestManagementDashboardDeleteWithoutRecordedIDMarksDeletedWhenIdentityMissing(t *testing.T) {
	resource := testManagementDashboardResource(t)
	fake := &fakeManagementDashboardOCIClient{
		listManagementDashboards: func(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
			return managementdashboardsdk.ListManagementDashboardsResponse{}, nil
		},
	}

	deleted, err := newTestManagementDashboardClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when OCI identity is absent")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteManagementDashboard() calls = %d, want 0", fake.deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestManagementDashboardDeleteWithoutRecordedIDResolvesExistingIdentity(t *testing.T) {
	resource := testManagementDashboardResource(t)
	dashboard := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	getCalls := 0
	fake := &fakeManagementDashboardOCIClient{
		listManagementDashboards: func(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
			return managementdashboardsdk.ListManagementDashboardsResponse{
				ManagementDashboardCollection: managementdashboardsdk.ManagementDashboardCollection{
					Items: []managementdashboardsdk.ManagementDashboardSummary{
						sdkManagementDashboardSummaryFromResource(t, resource, testManagementDashboardID, resource.Spec.DisplayName),
					},
				},
			}, nil
		},
		getManagementDashboard: func(_ context.Context, request managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			if got := managementDashboardStringValue(request.ManagementDashboardId); got != testManagementDashboardID {
				t.Fatalf("GetManagementDashboard() id = %q, want %q", got, testManagementDashboardID)
			}
			getCalls++
			if getCalls == 1 {
				return managementdashboardsdk.GetManagementDashboardResponse{ManagementDashboard: dashboard}, nil
			}
			return managementdashboardsdk.GetManagementDashboardResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "dashboard not found")
		},
		deleteManagementDashboard: func(_ context.Context, request managementdashboardsdk.DeleteManagementDashboardRequest) (managementdashboardsdk.DeleteManagementDashboardResponse, error) {
			if got := managementDashboardStringValue(request.ManagementDashboardId); got != testManagementDashboardID {
				t.Fatalf("DeleteManagementDashboard() id = %q, want %q", got, testManagementDashboardID)
			}
			return managementdashboardsdk.DeleteManagementDashboardResponse{}, nil
		},
	}

	deleted, err := newTestManagementDashboardClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after resolved identity delete confirmation")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testManagementDashboardID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testManagementDashboardID)
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeleteManagementDashboard() calls = %d, want 1", fake.deleteCalls)
	}
}

func TestManagementDashboardDeleteStopsBeforeDeleteOnAuthShapedConfirmRead(t *testing.T) {
	resource := testManagementDashboardResource(t)
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementDashboardID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeManagementDashboardOCIClient{
		getManagementDashboard: func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			return managementdashboardsdk.GetManagementDashboardResponse{}, authErr
		},
	}

	deleted, err := newTestManagementDashboardClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteManagementDashboard() calls = %d, want 0", fake.deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != authErr.GetOpcRequestID() {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, authErr.GetOpcRequestID())
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound context", err.Error())
	}
}

func TestManagementDashboardDeleteTreatsAuthShapedDeleteErrorAsAmbiguous(t *testing.T) {
	resource := testManagementDashboardResource(t)
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementDashboardID)
	active := sdkManagementDashboardFromResource(t, resource, testManagementDashboardID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeManagementDashboardOCIClient{
		getManagementDashboard: func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
			return managementdashboardsdk.GetManagementDashboardResponse{ManagementDashboard: active}, nil
		},
		deleteManagementDashboard: func(context.Context, managementdashboardsdk.DeleteManagementDashboardRequest) (managementdashboardsdk.DeleteManagementDashboardResponse, error) {
			return managementdashboardsdk.DeleteManagementDashboardResponse{}, authErr
		},
	}

	deleted, err := newTestManagementDashboardClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped delete rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeleteManagementDashboard() calls = %d, want 1", fake.deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != authErr.GetOpcRequestID() {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, authErr.GetOpcRequestID())
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound context", err.Error())
	}
}

func TestManagementDashboardCreateRecordsOCIErrorRequestID(t *testing.T) {
	resource := testManagementDashboardResource(t)
	createErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	fake := &fakeManagementDashboardOCIClient{
		listManagementDashboards: func(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
			return managementdashboardsdk.ListManagementDashboardsResponse{}, nil
		},
		createManagementDashboard: func(context.Context, managementdashboardsdk.CreateManagementDashboardRequest) (managementdashboardsdk.CreateManagementDashboardResponse, error) {
			return managementdashboardsdk.CreateManagementDashboardResponse{}, createErr
		},
	}

	response, err := newTestManagementDashboardClient(fake).CreateOrUpdate(context.Background(), resource, testManagementDashboardRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != createErr.GetOpcRequestID() {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, createErr.GetOpcRequestID())
	}
}

type fakeManagementDashboardOCIClient struct {
	createManagementDashboard func(context.Context, managementdashboardsdk.CreateManagementDashboardRequest) (managementdashboardsdk.CreateManagementDashboardResponse, error)
	getManagementDashboard    func(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error)
	listManagementDashboards  func(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error)
	updateManagementDashboard func(context.Context, managementdashboardsdk.UpdateManagementDashboardRequest) (managementdashboardsdk.UpdateManagementDashboardResponse, error)
	deleteManagementDashboard func(context.Context, managementdashboardsdk.DeleteManagementDashboardRequest) (managementdashboardsdk.DeleteManagementDashboardResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeManagementDashboardOCIClient) CreateManagementDashboard(
	ctx context.Context,
	request managementdashboardsdk.CreateManagementDashboardRequest,
) (managementdashboardsdk.CreateManagementDashboardResponse, error) {
	f.createCalls++
	if f.createManagementDashboard == nil {
		return managementdashboardsdk.CreateManagementDashboardResponse{}, fmt.Errorf("unexpected CreateManagementDashboard call")
	}
	return f.createManagementDashboard(ctx, request)
}

func (f *fakeManagementDashboardOCIClient) GetManagementDashboard(
	ctx context.Context,
	request managementdashboardsdk.GetManagementDashboardRequest,
) (managementdashboardsdk.GetManagementDashboardResponse, error) {
	f.getCalls++
	if f.getManagementDashboard == nil {
		return managementdashboardsdk.GetManagementDashboardResponse{}, fmt.Errorf("unexpected GetManagementDashboard call")
	}
	return f.getManagementDashboard(ctx, request)
}

func (f *fakeManagementDashboardOCIClient) ListManagementDashboards(
	ctx context.Context,
	request managementdashboardsdk.ListManagementDashboardsRequest,
) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
	f.listCalls++
	if f.listManagementDashboards == nil {
		return managementdashboardsdk.ListManagementDashboardsResponse{}, fmt.Errorf("unexpected ListManagementDashboards call")
	}
	return f.listManagementDashboards(ctx, request)
}

func (f *fakeManagementDashboardOCIClient) UpdateManagementDashboard(
	ctx context.Context,
	request managementdashboardsdk.UpdateManagementDashboardRequest,
) (managementdashboardsdk.UpdateManagementDashboardResponse, error) {
	f.updateCalls++
	if f.updateManagementDashboard == nil {
		return managementdashboardsdk.UpdateManagementDashboardResponse{}, fmt.Errorf("unexpected UpdateManagementDashboard call")
	}
	return f.updateManagementDashboard(ctx, request)
}

func (f *fakeManagementDashboardOCIClient) DeleteManagementDashboard(
	ctx context.Context,
	request managementdashboardsdk.DeleteManagementDashboardRequest,
) (managementdashboardsdk.DeleteManagementDashboardResponse, error) {
	f.deleteCalls++
	if f.deleteManagementDashboard == nil {
		return managementdashboardsdk.DeleteManagementDashboardResponse{}, fmt.Errorf("unexpected DeleteManagementDashboard call")
	}
	return f.deleteManagementDashboard(ctx, request)
}

func newTestManagementDashboardClient(client *fakeManagementDashboardOCIClient) ManagementDashboardServiceClient {
	return newManagementDashboardServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
	)
}

func testManagementDashboardRequest(resource *managementdashboardv1beta1.ManagementDashboard) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func testManagementDashboardResource(t *testing.T) *managementdashboardv1beta1.ManagementDashboard {
	t.Helper()

	return &managementdashboardv1beta1.ManagementDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dashboard",
			Namespace: "default",
			UID:       types.UID("dashboard-uid"),
		},
		Spec: managementdashboardv1beta1.ManagementDashboardSpec{
			ProviderId:        "log-analytics",
			ProviderName:      "Logging Analytics",
			ProviderVersion:   "3.0.0",
			Tiles:             []managementdashboardv1beta1.ManagementDashboardTile{testManagementDashboardTile(t)},
			DisplayName:       "dashboard",
			Description:       "dashboard description",
			CompartmentId:     testManagementDashboardCompartmentID,
			IsOobDashboard:    false,
			IsShowInHome:      false,
			MetadataVersion:   "2.0",
			IsShowDescription: false,
			ScreenImage:       "screen-image",
			Nls:               testJSONValue(`{"title":"Dashboard"}`),
			UiConfig:          testJSONValue(`{"layout":"grid"}`),
			DataConfig:        []shared.JSONValue{testJSONValue(`{"source":"logs"}`)},
			Type:              "NORMAL",
			IsFavorite:        false,
			ParametersConfig:  []shared.JSONValue{testJSONValue(`{"name":"compartment"}`)},
			FeaturesConfig:    testJSONValue(`{"refresh":true}`),
			DrilldownConfig:   []shared.JSONValue{testJSONValue(`{"destination":"details"}`)},
			FreeformTags:      map[string]string{"owner": "osok"},
			DefinedTags: map[string]shared.MapValue{
				"operations": {"costCenter": "test"},
			},
		},
	}
}

func testManagementDashboardTile(t *testing.T) managementdashboardv1beta1.ManagementDashboardTile {
	t.Helper()

	return managementdashboardv1beta1.ManagementDashboardTile{
		DisplayName:     "tile",
		SavedSearchId:   "saved-search",
		Row:             1,
		Column:          2,
		Height:          3,
		Width:           4,
		Nls:             testJSONValue(`{"title":"Tile"}`),
		UiConfig:        testJSONValue(`{"chart":"line"}`),
		DataConfig:      []shared.JSONValue{testJSONValue(`{"query":"search"}`)},
		State:           string(managementdashboardsdk.ManagementDashboardTileDetailsStateDefault),
		DrilldownConfig: testJSONValue(`{"enabled":true}`),
		ParametersMap:   testJSONValue(`{"param":"value"}`),
		Description:     "tile description",
	}
}

func sdkManagementDashboardFromResource(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementDashboard,
	id string,
) managementdashboardsdk.ManagementDashboard {
	t.Helper()

	spec := resource.Spec
	return managementdashboardsdk.ManagementDashboard{
		DashboardId:       common.String(id),
		Id:                common.String(id),
		ProviderId:        common.String(spec.ProviderId),
		ProviderName:      common.String(spec.ProviderName),
		ProviderVersion:   common.String(spec.ProviderVersion),
		Tiles:             sdkManagementDashboardTiles(t, spec.Tiles),
		DisplayName:       common.String(spec.DisplayName),
		Description:       common.String(spec.Description),
		CompartmentId:     common.String(spec.CompartmentId),
		IsOobDashboard:    common.Bool(spec.IsOobDashboard),
		IsShowInHome:      common.Bool(spec.IsShowInHome),
		CreatedBy:         common.String("creator"),
		UpdatedBy:         common.String("updater"),
		MetadataVersion:   common.String(spec.MetadataVersion),
		IsShowDescription: common.Bool(spec.IsShowDescription),
		ScreenImage:       common.String(spec.ScreenImage),
		Nls:               sdkInterfacePointer(t, spec.Nls),
		UiConfig:          sdkInterfacePointer(t, spec.UiConfig),
		DataConfig:        sdkInterfaceSlice(t, spec.DataConfig),
		Type:              common.String(spec.Type),
		IsFavorite:        common.Bool(spec.IsFavorite),
		SavedSearches:     []managementdashboardsdk.ManagementSavedSearch{},
		LifecycleState:    managementdashboardsdk.LifecycleStatesActive,
		ParametersConfig:  sdkInterfaceSlice(t, spec.ParametersConfig),
		DrilldownConfig:   sdkInterfaceSlice(t, spec.DrilldownConfig),
		FeaturesConfig:    sdkInterfacePointer(t, spec.FeaturesConfig),
		FreeformTags:      spec.FreeformTags,
		DefinedTags:       sdkDefinedTags(spec.DefinedTags),
	}
}

func sdkManagementDashboardSummaryFromResource(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementDashboard,
	id string,
	displayName string,
) managementdashboardsdk.ManagementDashboardSummary {
	t.Helper()

	spec := resource.Spec
	return managementdashboardsdk.ManagementDashboardSummary{
		DashboardId:     common.String(id),
		Id:              common.String(id),
		DisplayName:     common.String(displayName),
		Description:     common.String(spec.Description),
		CompartmentId:   common.String(spec.CompartmentId),
		ProviderId:      common.String(spec.ProviderId),
		ProviderName:    common.String(spec.ProviderName),
		ProviderVersion: common.String(spec.ProviderVersion),
		IsOobDashboard:  common.Bool(spec.IsOobDashboard),
		MetadataVersion: common.String(spec.MetadataVersion),
		ScreenImage:     common.String(spec.ScreenImage),
		Nls:             sdkInterfacePointer(t, spec.Nls),
		Type:            common.String(spec.Type),
		LifecycleState:  managementdashboardsdk.LifecycleStatesActive,
		FreeformTags:    spec.FreeformTags,
		DefinedTags:     sdkDefinedTags(spec.DefinedTags),
	}
}

func sdkManagementDashboardTiles(
	t *testing.T,
	tiles []managementdashboardv1beta1.ManagementDashboardTile,
) []managementdashboardsdk.ManagementDashboardTileDetails {
	t.Helper()

	converted := make([]managementdashboardsdk.ManagementDashboardTileDetails, 0, len(tiles))
	for _, tile := range tiles {
		converted = append(converted, managementdashboardsdk.ManagementDashboardTileDetails{
			DisplayName:     common.String(tile.DisplayName),
			SavedSearchId:   common.String(tile.SavedSearchId),
			Row:             common.Int(tile.Row),
			Column:          common.Int(tile.Column),
			Height:          common.Int(tile.Height),
			Width:           common.Int(tile.Width),
			Nls:             sdkInterfacePointer(t, tile.Nls),
			UiConfig:        sdkInterfacePointer(t, tile.UiConfig),
			DataConfig:      sdkInterfaceSlice(t, tile.DataConfig),
			State:           managementdashboardsdk.ManagementDashboardTileDetailsStateEnum(tile.State),
			DrilldownConfig: sdkInterfacePointer(t, tile.DrilldownConfig),
			ParametersMap:   sdkInterfacePointer(t, tile.ParametersMap),
			Description:     common.String(tile.Description),
		})
	}
	return converted
}

func testJSONValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func sdkInterfacePointer(t *testing.T, value shared.JSONValue) *interface{} {
	t.Helper()

	decoded := sdkInterfaceValue(t, value)
	return &decoded
}

func sdkInterfaceSlice(t *testing.T, values []shared.JSONValue) []interface{} {
	t.Helper()

	converted := make([]interface{}, 0, len(values))
	for _, value := range values {
		converted = append(converted, sdkInterfaceValue(t, value))
	}
	return converted
}

func sdkInterfaceValue(t *testing.T, value shared.JSONValue) interface{} {
	t.Helper()

	var decoded interface{}
	if err := json.Unmarshal(value.Raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", string(value.Raw), err)
	}
	return decoded
}

func sdkDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		convertedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			convertedValues[key] = value
		}
		converted[namespace] = convertedValues
	}
	return converted
}
