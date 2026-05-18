/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package auditprofile

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAuditProfileID            = "ocid1.auditprofile.oc1..example"
	testAuditProfileCompartmentID = "ocid1.compartment.oc1..example"
	testAuditProfileTargetID      = "ocid1.datasafetarget.oc1..example"
)

type fakeAuditProfileOCIClient struct {
	createFn func(context.Context, datasafesdk.CreateAuditProfileRequest) (datasafesdk.CreateAuditProfileResponse, error)
	getFn    func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error)
	listFn   func(context.Context, datasafesdk.ListAuditProfilesRequest) (datasafesdk.ListAuditProfilesResponse, error)
	updateFn func(context.Context, datasafesdk.UpdateAuditProfileRequest) (datasafesdk.UpdateAuditProfileResponse, error)
	deleteFn func(context.Context, datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error)
}

func (f *fakeAuditProfileOCIClient) CreateAuditProfile(
	ctx context.Context,
	req datasafesdk.CreateAuditProfileRequest,
) (datasafesdk.CreateAuditProfileResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return datasafesdk.CreateAuditProfileResponse{}, nil
}

func (f *fakeAuditProfileOCIClient) GetAuditProfile(
	ctx context.Context,
	req datasafesdk.GetAuditProfileRequest,
) (datasafesdk.GetAuditProfileResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return datasafesdk.GetAuditProfileResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "")
}

func (f *fakeAuditProfileOCIClient) ListAuditProfiles(
	ctx context.Context,
	req datasafesdk.ListAuditProfilesRequest,
) (datasafesdk.ListAuditProfilesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return datasafesdk.ListAuditProfilesResponse{}, nil
}

func (f *fakeAuditProfileOCIClient) UpdateAuditProfile(
	ctx context.Context,
	req datasafesdk.UpdateAuditProfileRequest,
) (datasafesdk.UpdateAuditProfileResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return datasafesdk.UpdateAuditProfileResponse{}, nil
}

func (f *fakeAuditProfileOCIClient) DeleteAuditProfile(
	ctx context.Context,
	req datasafesdk.DeleteAuditProfileRequest,
) (datasafesdk.DeleteAuditProfileResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return datasafesdk.DeleteAuditProfileResponse{}, nil
}

func testAuditProfileClient(fake *fakeAuditProfileOCIClient) AuditProfileServiceClient {
	return newAuditProfileServiceClientWithOCIClient(fake)
}

func makeAuditProfileResource() *datasafev1beta1.AuditProfile {
	return &datasafev1beta1.AuditProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "audit-profile",
			Namespace: "default",
			UID:       types.UID("audit-profile-uid"),
		},
		Spec: datasafev1beta1.AuditProfileSpec{
			CompartmentId:             testAuditProfileCompartmentID,
			TargetId:                  testAuditProfileTargetID,
			TargetType:                string(datasafesdk.AuditProfileTargetTypeTargetDatabase),
			DisplayName:               "audit-profile",
			Description:               "desired description",
			IsPaidUsageEnabled:        false,
			OnlineMonths:              6,
			OfflineMonths:             12,
			IsOverrideGlobalPaidUsage: false,
			FreeformTags:              map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKAuditProfile(
	id string,
	spec datasafev1beta1.AuditProfileSpec,
	state datasafesdk.AuditProfileLifecycleStateEnum,
) datasafesdk.AuditProfile {
	return datasafesdk.AuditProfile{
		Id:                               common.String(id),
		CompartmentId:                    common.String(spec.CompartmentId),
		DisplayName:                      common.String(spec.DisplayName),
		LifecycleState:                   state,
		TargetId:                         common.String(spec.TargetId),
		TargetType:                       datasafesdk.AuditProfileTargetTypeEnum(spec.TargetType),
		Description:                      common.String(spec.Description),
		IsPaidUsageEnabled:               common.Bool(spec.IsPaidUsageEnabled),
		OnlineMonths:                     common.Int(spec.OnlineMonths),
		OfflineMonths:                    common.Int(spec.OfflineMonths),
		IsOverrideGlobalRetentionSetting: common.Bool(false),
		IsOverrideGlobalPaidUsage:        common.Bool(spec.IsOverrideGlobalPaidUsage),
		FreeformTags:                     spec.FreeformTags,
		DefinedTags:                      auditProfileDefinedTags(spec.DefinedTags),
	}
}

func makeSDKAuditProfileSummary(
	id string,
	spec datasafev1beta1.AuditProfileSpec,
	state datasafesdk.AuditProfileLifecycleStateEnum,
) datasafesdk.AuditProfileSummary {
	return datasafesdk.AuditProfileSummary{
		Id:                               common.String(id),
		CompartmentId:                    common.String(spec.CompartmentId),
		DisplayName:                      common.String(spec.DisplayName),
		TargetId:                         common.String(spec.TargetId),
		TargetType:                       datasafesdk.AuditProfileTargetTypeEnum(spec.TargetType),
		LifecycleState:                   state,
		IsPaidUsageEnabled:               common.Bool(spec.IsPaidUsageEnabled),
		OnlineMonths:                     common.Int(spec.OnlineMonths),
		OfflineMonths:                    common.Int(spec.OfflineMonths),
		IsOverrideGlobalRetentionSetting: common.Bool(false),
		IsOverrideGlobalPaidUsage:        common.Bool(spec.IsOverrideGlobalPaidUsage),
		FreeformTags:                     spec.FreeformTags,
		DefinedTags:                      auditProfileDefinedTags(spec.DefinedTags),
	}
}

func auditProfileDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = map[string]interface{}{}
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func makeAuditProfileRequest(resource *datasafev1beta1.AuditProfile) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func requireAuditProfileAsyncCurrent(
	t *testing.T,
	resource *datasafev1beta1.AuditProfile,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want active async tracker")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func TestAuditProfileRuntimeHooksUseReviewedSemantics(t *testing.T) {
	hooks := newAuditProfileRuntimeHooks(&AuditProfileServiceManager{}, datasafesdk.DataSafeClient{})

	requireAuditProfileHookFields(t, hooks)
	requireAuditProfileHookConfiguration(t, hooks)
	requireAuditProfileBoolCreateBody(t, hooks)
}

func requireAuditProfileHookFields(t *testing.T, hooks AuditProfileRuntimeHooks) {
	t.Helper()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{name: "create", got: hooks.Create.Fields, want: auditProfileCreateFields()},
		{name: "get", got: hooks.Get.Fields, want: auditProfileGetFields()},
		{name: "list", got: hooks.List.Fields, want: auditProfileListFields()},
		{name: "update", got: hooks.Update.Fields, want: auditProfileUpdateFields()},
		{name: "delete", got: hooks.Delete.Fields, want: auditProfileDeleteFields()},
	}

	for _, tt := range tests {
		if !reflect.DeepEqual(tt.got, tt.want) {
			t.Fatalf("%s fields = %#v, want %#v", tt.name, tt.got, tt.want)
		}
	}
}

func requireAuditProfileHookConfiguration(t *testing.T, hooks AuditProfileRuntimeHooks) {
	t.Helper()

	if hooks.Semantics == nil {
		t.Fatal("semantics = nil, want reviewed semantics")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("semantics.auxiliaryOperations = %#v, want reviewed omission of auxiliary operations", hooks.Semantics.AuxiliaryOperations)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("build create body = nil, want bool-preserving create builder")
	}
	if hooks.DeleteHooks.ConfirmRead == nil {
		t.Fatal("delete confirm read hook = nil, want conservative NotAuthorizedOrNotFound handling")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("delete error hook = nil, want conservative NotAuthorizedOrNotFound handling")
	}
	if hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("delete outcome hook = nil, want conservative NotAuthorizedOrNotFound handling")
	}
}

func requireAuditProfileBoolCreateBody(t *testing.T, hooks AuditProfileRuntimeHooks) {
	t.Helper()

	body, err := hooks.BuildCreateBody(context.Background(), makeAuditProfileResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	values, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want map[string]any", body)
	}
	if got, ok := values["isPaidUsageEnabled"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() isPaidUsageEnabled = %#v, present = %t, want explicit false", got, ok)
	}
	if got, ok := values["isOverrideGlobalPaidUsage"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() isOverrideGlobalPaidUsage = %#v, present = %t, want explicit false", got, ok)
	}
}

func TestAuditProfileCreateOrUpdateCreatesAndTracksLifecycle(t *testing.T) {
	resource := makeAuditProfileResource()
	created := makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateCreating)
	var createRequest datasafesdk.CreateAuditProfileRequest
	var getRequest datasafesdk.GetAuditProfileRequest

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		createFn: func(_ context.Context, req datasafesdk.CreateAuditProfileRequest) (datasafesdk.CreateAuditProfileResponse, error) {
			createRequest = req
			return datasafesdk.CreateAuditProfileResponse{
				AuditProfile:     created,
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			getRequest = req
			return datasafesdk.GetAuditProfileResponse{AuditProfile: created}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAuditProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireAuditProfileCreateResponse(t, response)
	requireAuditProfileCreateRequest(t, createRequest, resource.Spec)
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("create opc retry token is empty")
	}
	requireAuditProfileGetRequest(t, getRequest, testAuditProfileID)
	requireAuditProfileCreatedStatus(t, resource)
}

func requireAuditProfileCreateResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create is in progress")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the audit profile is CREATING")
	}
}

func requireAuditProfileCreateRequest(
	t *testing.T,
	request datasafesdk.CreateAuditProfileRequest,
	spec datasafev1beta1.AuditProfileSpec,
) {
	t.Helper()

	if request.CompartmentId == nil || *request.CompartmentId != spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", request.CompartmentId, spec.CompartmentId)
	}
	if request.IsPaidUsageEnabled == nil || *request.IsPaidUsageEnabled != false {
		t.Fatalf("create isPaidUsageEnabled = %v, want explicit false", request.IsPaidUsageEnabled)
	}
}

func requireAuditProfileGetRequest(t *testing.T, request datasafesdk.GetAuditProfileRequest, wantID string) {
	t.Helper()

	if request.AuditProfileId == nil || *request.AuditProfileId != wantID {
		t.Fatalf("get auditProfileId = %v, want %q", request.AuditProfileId, wantID)
	}
}

func requireAuditProfileCreatedStatus(t *testing.T, resource *datasafev1beta1.AuditProfile) {
	t.Helper()

	if resource.Status.Id != testAuditProfileID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testAuditProfileID)
	}
	if string(resource.Status.OsokStatus.Ocid) != testAuditProfileID {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testAuditProfileID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
	requireAuditProfileAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "CREATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want CREATING", resource.Status.OsokStatus.Async.Current.RawStatus)
	}
}

func TestAuditProfileCreateOrUpdateBindsExistingFromLaterTargetGroupListPage(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Spec.TargetType = string(datasafesdk.AuditProfileTargetTypeTargetDatabaseGroup)
	existing := makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateActive)
	var pages []string
	createCalls := 0

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		listFn: auditProfileTargetGroupListPages(t, resource, &pages),
		getFn: func(_ context.Context, req datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			requireAuditProfileGetRequest(t, req, testAuditProfileID)
			return datasafesdk.GetAuditProfileResponse{AuditProfile: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateAuditProfileRequest) (datasafesdk.CreateAuditProfileResponse, error) {
			createCalls++
			return datasafesdk.CreateAuditProfileResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAuditProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if createCalls != 0 {
		t.Fatalf("CreateAuditProfile() calls = %d, want 0 for bind", createCalls)
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("list pages = %q, want \",page-2\"", got)
	}
	if resource.Status.Id != testAuditProfileID {
		t.Fatalf("status.id = %q, want resolved ID", resource.Status.Id)
	}
}

func auditProfileTargetGroupListPages(
	t *testing.T,
	resource *datasafev1beta1.AuditProfile,
	pages *[]string,
) auditProfileListCall {
	t.Helper()

	return func(_ context.Context, req datasafesdk.ListAuditProfilesRequest) (datasafesdk.ListAuditProfilesResponse, error) {
		*pages = append(*pages, stringValue(req.Page))
		requireAuditProfileTargetGroupListRequest(t, req, resource.Spec.TargetId)
		if req.Page == nil {
			otherSpec := resource.Spec
			otherSpec.TargetId = "ocid1.datasafetarget.oc1..other"
			item := makeSDKAuditProfileSummary("ocid1.auditprofile.oc1..other", otherSpec, datasafesdk.AuditProfileLifecycleStateActive)
			return auditProfileListResponse([]datasafesdk.AuditProfileSummary{item}, "page-2"), nil
		}
		item := makeSDKAuditProfileSummary(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateActive)
		return auditProfileListResponse([]datasafesdk.AuditProfileSummary{item}, ""), nil
	}
}

func requireAuditProfileTargetGroupListRequest(t *testing.T, req datasafesdk.ListAuditProfilesRequest, wantTargetID string) {
	t.Helper()

	if req.TargetId != nil {
		t.Fatalf("list targetId = %v, want nil for TARGET_DATABASE_GROUP", req.TargetId)
	}
	if req.TargetDatabaseGroupId == nil || *req.TargetDatabaseGroupId != wantTargetID {
		t.Fatalf("list targetDatabaseGroupId = %v, want %q", req.TargetDatabaseGroupId, wantTargetID)
	}
}

func auditProfileListResponse(items []datasafesdk.AuditProfileSummary, nextPage string) datasafesdk.ListAuditProfilesResponse {
	response := datasafesdk.ListAuditProfilesResponse{
		AuditProfileCollection: datasafesdk.AuditProfileCollection{Items: items},
	}
	if nextPage != "" {
		response.OpcNextPage = common.String(nextPage)
	}
	return response
}

func TestAuditProfileCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	updateCalls := 0

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(_ context.Context, req datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			if req.AuditProfileId == nil || *req.AuditProfileId != testAuditProfileID {
				t.Fatalf("get auditProfileId = %v, want tracked ID", req.AuditProfileId)
			}
			return datasafesdk.GetAuditProfileResponse{
				AuditProfile: makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateAuditProfileRequest) (datasafesdk.UpdateAuditProfileResponse, error) {
			updateCalls++
			return datasafesdk.UpdateAuditProfileResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAuditProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateAuditProfile() calls = %d, want 0", updateCalls)
	}
}

func TestAuditProfileMutableUpdatePreservesExplicitFalse(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	currentSpec := resource.Spec
	currentSpec.Description = "old description"
	currentSpec.IsPaidUsageEnabled = true
	updated := makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateUpdating)
	getCalls := 0
	var updateRequest datasafesdk.UpdateAuditProfileRequest

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(_ context.Context, req datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			getCalls++
			if req.AuditProfileId == nil || *req.AuditProfileId != testAuditProfileID {
				t.Fatalf("get auditProfileId = %v, want tracked ID", req.AuditProfileId)
			}
			if getCalls == 1 {
				return datasafesdk.GetAuditProfileResponse{
					AuditProfile: makeSDKAuditProfile(testAuditProfileID, currentSpec, datasafesdk.AuditProfileLifecycleStateActive),
				}, nil
			}
			return datasafesdk.GetAuditProfileResponse{AuditProfile: updated}, nil
		},
		updateFn: func(_ context.Context, req datasafesdk.UpdateAuditProfileRequest) (datasafesdk.UpdateAuditProfileResponse, error) {
			updateRequest = req
			return datasafesdk.UpdateAuditProfileResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAuditProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while update is in progress")
	}
	requireAuditProfileUpdateRequest(t, updateRequest, resource.Spec)
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
	requireAuditProfileAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1")
}

func requireAuditProfileUpdateRequest(
	t *testing.T,
	request datasafesdk.UpdateAuditProfileRequest,
	spec datasafev1beta1.AuditProfileSpec,
) {
	t.Helper()

	if request.AuditProfileId == nil || *request.AuditProfileId != testAuditProfileID {
		t.Fatalf("update auditProfileId = %v, want tracked ID", request.AuditProfileId)
	}
	if request.Description == nil || *request.Description != spec.Description {
		t.Fatalf("update description = %v, want %q", request.Description, spec.Description)
	}
	if request.IsPaidUsageEnabled == nil || *request.IsPaidUsageEnabled != false {
		t.Fatalf("update isPaidUsageEnabled = %v, want explicit false", request.IsPaidUsageEnabled)
	}
}

func TestAuditProfileCreateOrUpdateRejectsCreateOnlyTargetDrift(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	observedSpec := resource.Spec
	observedSpec.TargetId = "ocid1.datasafetarget.oc1..observed"
	updateCalls := 0

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			return datasafesdk.GetAuditProfileResponse{
				AuditProfile: makeSDKAuditProfile(testAuditProfileID, observedSpec, datasafesdk.AuditProfileLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateAuditProfileRequest) (datasafesdk.UpdateAuditProfileResponse, error) {
			updateCalls++
			return datasafesdk.UpdateAuditProfileResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAuditProfileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want replacement validation error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when target drift requires replacement")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateAuditProfile() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "AuditProfile formal semantics require replacement when targetId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want targetId replacement message", err)
	}
}

func TestAuditProfileDeleteRetainsFinalizerUntilLifecycleDeleted(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	getCalls := 0
	deleteCalls := 0

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn:    auditProfileDeleteLifecycleGet(t, resource, &getCalls),
		deleteFn: auditProfileTrackedDelete(t, &deleteCalls),
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep finalizer while lifecycle is DELETING")
	}
	if getCalls != 2 {
		t.Fatalf("GetAuditProfile() calls = %d, want 2", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAuditProfile() calls = %d, want 1", deleteCalls)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.LifecycleState != string(datasafesdk.AuditProfileLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	requireAuditProfileAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")
}

func auditProfileDeleteLifecycleGet(
	t *testing.T,
	resource *datasafev1beta1.AuditProfile,
	calls *int,
) auditProfileGetCall {
	t.Helper()

	return func(_ context.Context, req datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
		*calls++
		requireAuditProfileGetRequest(t, req, testAuditProfileID)
		state := datasafesdk.AuditProfileLifecycleStateActive
		if *calls > 1 {
			state = datasafesdk.AuditProfileLifecycleStateDeleting
		}
		return datasafesdk.GetAuditProfileResponse{
			AuditProfile: makeSDKAuditProfile(testAuditProfileID, resource.Spec, state),
		}, nil
	}
}

func auditProfileTrackedDelete(t *testing.T, calls *int) func(context.Context, datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error) {
	t.Helper()

	return func(_ context.Context, req datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error) {
		*calls++
		if req.AuditProfileId == nil || *req.AuditProfileId != testAuditProfileID {
			t.Fatalf("delete auditProfileId = %v, want tracked ID", req.AuditProfileId)
		}
		return datasafesdk.DeleteAuditProfileResponse{
			OpcRequestId:     common.String("opc-delete-1"),
			OpcWorkRequestId: common.String("wr-delete-1"),
		}, nil
	}
}

func TestAuditProfileDeleteConfirmsDeletedLifecycle(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			return datasafesdk.GetAuditProfileResponse{
				AuditProfile: makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateDeleted),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should remove finalizer when lifecycle is DELETED")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestAuditProfileDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	ambiguous := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "")
	getCalls := 0
	deleteCalls := 0

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			getCalls++
			return datasafesdk.GetAuditProfileResponse{}, ambiguous
		},
		deleteFn: func(context.Context, datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error) {
			deleteCalls++
			return datasafesdk.DeleteAuditProfileResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	requireAuditProfileAuthShapedDeleteRetained(t, resource, deleted, err)
	if getCalls != 1 {
		t.Fatalf("GetAuditProfile() calls = %d, want 1", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteAuditProfile() calls = %d, want 0 after ambiguous pre-delete read", deleteCalls)
	}
}

func TestAuditProfileDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	ambiguous := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "")
	getCalls := 0
	deleteCalls := 0

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAuditProfileResponse{
					AuditProfile: makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateActive),
				}, nil
			}
			return datasafesdk.GetAuditProfileResponse{}, ambiguous
		},
		deleteFn: func(context.Context, datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error) {
			deleteCalls++
			return datasafesdk.DeleteAuditProfileResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	requireAuditProfileAuthShapedDeleteRetained(t, resource, deleted, err)
	if getCalls != 2 {
		t.Fatalf("GetAuditProfile() calls = %d, want 2", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAuditProfile() calls = %d, want 1", deleteCalls)
	}
}

func TestAuditProfileDeleteTreatsNotAuthorizedOrNotFoundConservatively(t *testing.T) {
	resource := makeAuditProfileResource()
	resource.Status.Id = testAuditProfileID
	ambiguous := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "")

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		getFn: func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
			return datasafesdk.GetAuditProfileResponse{
				AuditProfile: makeSDKAuditProfile(testAuditProfileID, resource.Spec, datasafesdk.AuditProfileLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error) {
			return datasafesdk.DeleteAuditProfileResponse{}, ambiguous
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped not found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer-retaining status")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want auth-shaped request id", resource.Status.OsokStatus.OpcRequestID)
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want auth-shaped classification", err)
	}
}

func requireAuditProfileAuthShapedDeleteRetained(
	t *testing.T,
	resource *datasafev1beta1.AuditProfile,
	deleted bool,
	err error,
) {
	t.Helper()

	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped not found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer-retaining status")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want auth-shaped request id", resource.Status.OsokStatus.OpcRequestID)
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want auth-shaped classification", err)
	}
}

func TestAuditProfileCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := makeAuditProfileResource()
	creationErr := errortest.NewServiceError(500, "InternalError", "")

	client := testAuditProfileClient(&fakeAuditProfileOCIClient{
		createFn: func(context.Context, datasafesdk.CreateAuditProfileRequest) (datasafesdk.CreateAuditProfileResponse, error) {
			return datasafesdk.CreateAuditProfileResponse{}, creationErr
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAuditProfileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want create error request id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
