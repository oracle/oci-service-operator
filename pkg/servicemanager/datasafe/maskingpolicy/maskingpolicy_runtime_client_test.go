/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package maskingpolicy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testMaskingPolicyID            = "ocid1.datasafemaskingpolicy.oc1..policy"
	testMaskingPolicyCompartmentID = "ocid1.compartment.oc1..masking"
	testMaskingPolicyTargetID      = "ocid1.datasafetargetdatabase.oc1..target"
	testMaskingPolicySDMID         = "ocid1.datasafesensitivedatamodel.oc1..model"
	testMaskingPolicyDisplayName   = "customer-masking-policy"
)

type fakeMaskingPolicyOCIClient struct {
	createFn func(context.Context, datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error)
	getFn    func(context.Context, datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error)
	listFn   func(context.Context, datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error)
	updateFn func(context.Context, datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error)
	deleteFn func(context.Context, datasafesdk.DeleteMaskingPolicyRequest) (datasafesdk.DeleteMaskingPolicyResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeMaskingPolicyOCIClient) CreateMaskingPolicy(
	ctx context.Context,
	request datasafesdk.CreateMaskingPolicyRequest,
) (datasafesdk.CreateMaskingPolicyResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateMaskingPolicyResponse{}, nil
}

func (f *fakeMaskingPolicyOCIClient) GetMaskingPolicy(
	ctx context.Context,
	request datasafesdk.GetMaskingPolicyRequest,
) (datasafesdk.GetMaskingPolicyResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetMaskingPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "MaskingPolicy is missing")
}

func (f *fakeMaskingPolicyOCIClient) ListMaskingPolicies(
	ctx context.Context,
	request datasafesdk.ListMaskingPoliciesRequest,
) (datasafesdk.ListMaskingPoliciesResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListMaskingPoliciesResponse{}, nil
}

func (f *fakeMaskingPolicyOCIClient) UpdateMaskingPolicy(
	ctx context.Context,
	request datasafesdk.UpdateMaskingPolicyRequest,
) (datasafesdk.UpdateMaskingPolicyResponse, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return datasafesdk.UpdateMaskingPolicyResponse{}, nil
}

func (f *fakeMaskingPolicyOCIClient) DeleteMaskingPolicy(
	ctx context.Context,
	request datasafesdk.DeleteMaskingPolicyRequest,
) (datasafesdk.DeleteMaskingPolicyResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteMaskingPolicyResponse{}, nil
}

func TestMaskingPolicyRuntimeHooksConfigured(t *testing.T) {
	hooks := newMaskingPolicyDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyMaskingPolicyRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
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

	body, err := hooks.BuildCreateBody(context.Background(), makeMaskingPolicyResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateMaskingPolicyDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateMaskingPolicyDetails", body)
	}
	requireStringPtr(t, "CreateMaskingPolicyDetails.CompartmentId", details.CompartmentId, testMaskingPolicyCompartmentID)
	requireStringPtr(t, "CreateMaskingPolicyDetails.DisplayName", details.DisplayName, testMaskingPolicyDisplayName)
	requireBoolPtr(t, "CreateMaskingPolicyDetails.IsDropTempTablesEnabled", details.IsDropTempTablesEnabled, false)
	requireBoolPtr(t, "CreateMaskingPolicyDetails.IsRedoLoggingEnabled", details.IsRedoLoggingEnabled, true)
	requireBoolPtr(t, "CreateMaskingPolicyDetails.IsRefreshStatsEnabled", details.IsRefreshStatsEnabled, false)
	requireCreateTargetColumnSource(t, details.ColumnSource, testMaskingPolicyTargetID)

	sdmResource := makeMaskingPolicyResource()
	sdmResource.Spec.ColumnSource = datasafev1beta1.MaskingPolicyColumnSource{
		ColumnSource:         string(datasafesdk.ColumnSourceDetailsColumnSourceSensitiveDataModel),
		SensitiveDataModelId: testMaskingPolicySDMID,
	}
	sdmBody, err := hooks.BuildCreateBody(context.Background(), sdmResource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() for sensitive data model source error = %v", err)
	}
	sdmDetails, ok := sdmBody.(datasafesdk.CreateMaskingPolicyDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() for sensitive data model source type = %T, want CreateMaskingPolicyDetails", sdmBody)
	}
	requireCreateSDMColumnSource(t, sdmDetails.ColumnSource, testMaskingPolicySDMID)
}

func TestMaskingPolicyCreateRecordsIdentityRequestIDAndLifecycle(t *testing.T) {
	resource := makeMaskingPolicyResource()
	created := sdkMaskingPolicy(resource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateCreating)
	client := &fakeMaskingPolicyOCIClient{
		createFn: func(_ context.Context, request datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error) {
			requireMaskingPolicyCreateRequest(t, request, resource)
			return datasafesdk.CreateMaskingPolicyResponse{
				MaskingPolicy: created,
				OpcRequestId:  common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
			requireStringPtr(t, "GetMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
			return datasafesdk.GetMaskingPolicyResponse{MaskingPolicy: created}, nil
		},
	}

	response, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while CREATING")
	}
	assertMaskingPolicyCallCount(t, "ListMaskingPolicies()", client.listCalls, 1)
	assertMaskingPolicyCallCount(t, "CreateMaskingPolicy()", client.createCalls, 1)
	assertMaskingPolicyCallCount(t, "GetMaskingPolicy()", client.getCalls, 1)
	assertMaskingPolicyRecordedID(t, resource, testMaskingPolicyID)
	assertMaskingPolicyOpcRequestID(t, resource, "opc-create")
	if got := resource.Status.ColumnSource.TargetId; got != testMaskingPolicyTargetID {
		t.Fatalf("status.columnSource.targetId = %q, want %q", got, testMaskingPolicyTargetID)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %#v, want create lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMaskingPolicyCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeMaskingPolicyResource()
	existing := sdkMaskingPolicy(resource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	var pages []string
	client := &fakeMaskingPolicyOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListMaskingPoliciesRequest.CompartmentId", request.CompartmentId, testMaskingPolicyCompartmentID)
			requireStringPtr(t, "ListMaskingPoliciesRequest.DisplayName", request.DisplayName, testMaskingPolicyDisplayName)
			requireStringPtr(t, "ListMaskingPoliciesRequest.TargetId", request.TargetId, testMaskingPolicyTargetID)
			if request.Page == nil {
				return datasafesdk.ListMaskingPoliciesResponse{
					MaskingPolicyCollection: datasafesdk.MaskingPolicyCollection{
						Items: []datasafesdk.MaskingPolicySummary{
							sdkMaskingPolicySummary(resource, "ocid1.datasafemaskingpolicy.oc1..other", "other-policy", datasafesdk.MaskingLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListMaskingPoliciesResponse{
				MaskingPolicyCollection: datasafesdk.MaskingPolicyCollection{
					Items: []datasafesdk.MaskingPolicySummary{
						sdkMaskingPolicySummary(resource, testMaskingPolicyID, testMaskingPolicyDisplayName, datasafesdk.MaskingLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
			requireStringPtr(t, "GetMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
			return datasafesdk.GetMaskingPolicyResponse{MaskingPolicy: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error) {
			t.Fatal("CreateMaskingPolicy() called despite existing list match")
			return datasafesdk.CreateMaskingPolicyResponse{}, nil
		},
	}

	response, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListMaskingPolicies() pages = %q, want \",page-2\"", got)
	}
	assertMaskingPolicyRecordedID(t, resource, testMaskingPolicyID)
	assertMaskingPolicyCallCount(t, "CreateMaskingPolicy()", client.createCalls, 0)
}

func TestMaskingPolicyCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeMaskingPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMaskingPolicyID)
	current := sdkMaskingPolicy(resource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	client := &fakeMaskingPolicyOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
			requireStringPtr(t, "GetMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
			return datasafesdk.GetMaskingPolicyResponse{MaskingPolicy: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error) {
			t.Fatal("UpdateMaskingPolicy() called during no-op reconcile")
			return datasafesdk.UpdateMaskingPolicyResponse{}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error) {
			t.Fatal("CreateMaskingPolicy() called during no-op reconcile")
			return datasafesdk.CreateMaskingPolicyResponse{}, nil
		},
	}

	response, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMaskingPolicyCallCount(t, "GetMaskingPolicy()", client.getCalls, 1)
	assertMaskingPolicyCallCount(t, "UpdateMaskingPolicy()", client.updateCalls, 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestMaskingPolicyLowercaseRecompileIsCanonicalizedBeforeDriftComparison(t *testing.T) {
	resource := makeMaskingPolicyResource()
	resource.Spec.Recompile = "parallel"
	resource.Status.OsokStatus.Ocid = shared.OCID(testMaskingPolicyID)
	current := sdkMaskingPolicy(resource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	current.Recompile = datasafesdk.MaskingPolicyRecompileParallel
	client := &fakeMaskingPolicyOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
			requireStringPtr(t, "GetMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
			return datasafesdk.GetMaskingPolicyResponse{MaskingPolicy: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error) {
			t.Fatal("UpdateMaskingPolicy() called when lowercase recompile matches canonical readback")
			return datasafesdk.UpdateMaskingPolicyResponse{}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error) {
			t.Fatal("CreateMaskingPolicy() called during no-op reconcile")
			return datasafesdk.CreateMaskingPolicyResponse{}, nil
		},
	}

	createBody, err := buildMaskingPolicyCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildMaskingPolicyCreateBody() error = %v", err)
	}
	createDetails, ok := createBody.(datasafesdk.CreateMaskingPolicyDetails)
	if !ok {
		t.Fatalf("buildMaskingPolicyCreateBody() type = %T, want CreateMaskingPolicyDetails", createBody)
	}
	if createDetails.Recompile != datasafesdk.MaskingPolicyRecompileParallel {
		t.Fatalf("CreateMaskingPolicyDetails.Recompile = %q, want %q", createDetails.Recompile, datasafesdk.MaskingPolicyRecompileParallel)
	}
	updateBody, needsUpdate, err := buildMaskingPolicyUpdateBody(context.Background(), resource, "default", datasafesdk.GetMaskingPolicyResponse{MaskingPolicy: current})
	if err != nil {
		t.Fatalf("buildMaskingPolicyUpdateBody() error = %v", err)
	}
	updateDetails, ok := updateBody.(datasafesdk.UpdateMaskingPolicyDetails)
	if !ok {
		t.Fatalf("buildMaskingPolicyUpdateBody() type = %T, want UpdateMaskingPolicyDetails", updateBody)
	}
	if updateDetails.Recompile != datasafesdk.MaskingPolicyRecompileParallel {
		t.Fatalf("UpdateMaskingPolicyDetails.Recompile = %q, want %q", updateDetails.Recompile, datasafesdk.MaskingPolicyRecompileParallel)
	}
	if needsUpdate {
		t.Fatal("buildMaskingPolicyUpdateBody() needsUpdate = true, want false")
	}

	response, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMaskingPolicyCallCount(t, "GetMaskingPolicy()", client.getCalls, 1)
	assertMaskingPolicyCallCount(t, "UpdateMaskingPolicy()", client.updateCalls, 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestMaskingPolicyMutableUpdateRefreshesObservedState(t *testing.T) {
	resource := makeMaskingPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMaskingPolicyID)
	currentResource := makeMaskingPolicyResource()
	currentResource.Spec.DisplayName = "old-policy-name"
	current := sdkMaskingPolicy(currentResource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	updated := sdkMaskingPolicy(resource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	getResponses := []datasafesdk.GetMaskingPolicyResponse{
		{MaskingPolicy: current},
		{MaskingPolicy: updated},
	}
	client := &fakeMaskingPolicyOCIClient{
		getFn: getMaskingPolicyResponses(t, &getResponses),
		updateFn: func(_ context.Context, request datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error) {
			requireStringPtr(t, "UpdateMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
			requireStringPtr(t, "UpdateMaskingPolicyDetails.DisplayName", request.DisplayName, testMaskingPolicyDisplayName)
			if request.Recompile != datasafesdk.MaskingPolicyRecompileParallel {
				t.Fatalf("UpdateMaskingPolicyDetails.Recompile = %q, want %q", request.Recompile, datasafesdk.MaskingPolicyRecompileParallel)
			}
			requireUpdateTargetColumnSource(t, request.ColumnSource, testMaskingPolicyTargetID)
			return datasafesdk.UpdateMaskingPolicyResponse{OpcRequestId: common.String("opc-update")}, nil
		},
	}

	response, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMaskingPolicyCallCount(t, "UpdateMaskingPolicy()", client.updateCalls, 1)
	assertMaskingPolicyCallCount(t, "GetMaskingPolicy()", client.getCalls, 2)
	assertMaskingPolicyOpcRequestID(t, resource, "opc-update")
	if got := resource.Status.DisplayName; got != testMaskingPolicyDisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, testMaskingPolicyDisplayName)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestMaskingPolicyImmutableCompartmentDriftRejectedBeforeUpdate(t *testing.T) {
	resource := makeMaskingPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMaskingPolicyID)
	currentResource := makeMaskingPolicyResource()
	current := sdkMaskingPolicy(currentResource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..renamed"
	client := &fakeMaskingPolicyOCIClient{
		getFn: func(context.Context, datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
			return datasafesdk.GetMaskingPolicyResponse{MaskingPolicy: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error) {
			t.Fatal("UpdateMaskingPolicy() called despite immutable compartment drift")
			return datasafesdk.UpdateMaskingPolicyResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteMaskingPolicyRequest) (datasafesdk.DeleteMaskingPolicyResponse, error) {
			t.Fatal("DeleteMaskingPolicy() called during create/update drift handling")
			return datasafesdk.DeleteMaskingPolicyResponse{}, nil
		},
	}

	_, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift detail", err)
	}
	assertMaskingPolicyCallCount(t, "UpdateMaskingPolicy()", client.updateCalls, 0)
	assertMaskingPolicyCallCount(t, "DeleteMaskingPolicy()", client.deleteCalls, 0)
	requireLastCondition(t, resource, shared.Failed)
}

func TestMaskingPolicyDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := makeMaskingPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMaskingPolicyID)
	active := sdkMaskingPolicy(resource, testMaskingPolicyID, datasafesdk.MaskingLifecycleStateActive)
	getResponses := []datasafesdk.GetMaskingPolicyResponse{
		{MaskingPolicy: active},
		{MaskingPolicy: active},
		{MaskingPolicy: active},
	}
	client := &fakeMaskingPolicyOCIClient{
		getFn: getMaskingPolicyResponses(t, &getResponses),
		deleteFn: func(_ context.Context, request datasafesdk.DeleteMaskingPolicyRequest) (datasafesdk.DeleteMaskingPolicyResponse, error) {
			requireStringPtr(t, "DeleteMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
			return datasafesdk.DeleteMaskingPolicyResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestMaskingPolicyClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertMaskingPolicyCallCount(t, "DeleteMaskingPolicy()", client.deleteCalls, 1)
	assertMaskingPolicyOpcRequestID(t, resource, "opc-delete")
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMaskingPolicyDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := makeMaskingPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMaskingPolicyID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-auth"
	client := &fakeMaskingPolicyOCIClient{
		getFn: func(context.Context, datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
			return datasafesdk.GetMaskingPolicyResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteMaskingPolicyRequest) (datasafesdk.DeleteMaskingPolicyResponse, error) {
			t.Fatal("DeleteMaskingPolicy() called after ambiguous pre-delete read")
			return datasafesdk.DeleteMaskingPolicyResponse{}, nil
		},
	}

	deleted, err := newTestMaskingPolicyClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	assertMaskingPolicyCallCount(t, "DeleteMaskingPolicy()", client.deleteCalls, 0)
	assertMaskingPolicyOpcRequestID(t, resource, "opc-auth")
}

func TestMaskingPolicyCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := makeMaskingPolicyResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	client := &fakeMaskingPolicyOCIClient{
		createFn: func(context.Context, datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error) {
			return datasafesdk.CreateMaskingPolicyResponse{}, createErr
		},
	}

	_, err := newTestMaskingPolicyClient(client).CreateOrUpdate(context.Background(), resource, maskingPolicyRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	assertMaskingPolicyOpcRequestID(t, resource, "opc-create-error")
	requireLastCondition(t, resource, shared.Failed)
}

func newTestMaskingPolicyClient(client maskingPolicyOCIClient) MaskingPolicyServiceClient {
	return newMaskingPolicyServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeMaskingPolicyResource() *datasafev1beta1.MaskingPolicy {
	return &datasafev1beta1.MaskingPolicy{
		ObjectMeta: v1.ObjectMeta{
			Name:      "masking-policy",
			Namespace: "default",
		},
		Spec: datasafev1beta1.MaskingPolicySpec{
			CompartmentId: testMaskingPolicyCompartmentID,
			ColumnSource: datasafev1beta1.MaskingPolicyColumnSource{
				ColumnSource: string(datasafesdk.ColumnSourceDetailsColumnSourceTarget),
				TargetId:     testMaskingPolicyTargetID,
			},
			DisplayName:             testMaskingPolicyDisplayName,
			Description:             "mask customer data",
			IsDropTempTablesEnabled: false,
			IsRedoLoggingEnabled:    true,
			IsRefreshStatsEnabled:   false,
			ParallelDegree:          "2",
			Recompile:               string(datasafesdk.MaskingPolicyRecompileParallel),
			PreMaskingScript:        "begin null; end;",
			PostMaskingScript:       "begin null; end;",
			FreeformTags:            map[string]string{"owner": "runtime"},
			DefinedTags:             map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func maskingPolicyRequest(resource *datasafev1beta1.MaskingPolicy) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func sdkMaskingPolicy(
	resource *datasafev1beta1.MaskingPolicy,
	id string,
	lifecycleState datasafesdk.MaskingLifecycleStateEnum,
) datasafesdk.MaskingPolicy {
	created := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 5, 0, 0, time.UTC)}
	return datasafesdk.MaskingPolicy{
		Id:                           common.String(id),
		CompartmentId:                common.String(resource.Spec.CompartmentId),
		DisplayName:                  common.String(resource.Spec.DisplayName),
		TimeCreated:                  &created,
		LifecycleState:               lifecycleState,
		TimeUpdated:                  &updated,
		IsDropTempTablesEnabled:      common.Bool(resource.Spec.IsDropTempTablesEnabled),
		IsRedoLoggingEnabled:         common.Bool(resource.Spec.IsRedoLoggingEnabled),
		IsRefreshStatsEnabled:        common.Bool(resource.Spec.IsRefreshStatsEnabled),
		ParallelDegree:               common.String(resource.Spec.ParallelDegree),
		Recompile:                    datasafesdk.MaskingPolicyRecompileEnum(maskingPolicyCanonicalRecompile(resource.Spec.Recompile)),
		Description:                  common.String(resource.Spec.Description),
		PreMaskingScript:             common.String(resource.Spec.PreMaskingScript),
		PostMaskingScript:            common.String(resource.Spec.PostMaskingScript),
		ColumnSource:                 sdkMaskingPolicyColumnSource(resource.Spec.ColumnSource),
		AreTargetCredentialsRequired: common.Bool(true),
		FreeformTags:                 maskingPolicyStringMap(resource.Spec.FreeformTags),
		DefinedTags:                  maskingPolicyDefinedTags(resource.Spec.DefinedTags),
	}
}

func sdkMaskingPolicySummary(
	resource *datasafev1beta1.MaskingPolicy,
	id string,
	displayName string,
	lifecycleState datasafesdk.MaskingLifecycleStateEnum,
) datasafesdk.MaskingPolicySummary {
	created := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 5, 0, 0, time.UTC)}
	return datasafesdk.MaskingPolicySummary{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(displayName),
		TimeCreated:    &created,
		TimeUpdated:    &updated,
		LifecycleState: lifecycleState,
		Description:    common.String(resource.Spec.Description),
		ColumnSource:   sdkMaskingPolicyColumnSource(resource.Spec.ColumnSource),
		FreeformTags:   maskingPolicyStringMap(resource.Spec.FreeformTags),
		DefinedTags:    maskingPolicyDefinedTags(resource.Spec.DefinedTags),
	}
}

func sdkMaskingPolicyColumnSource(source datasafev1beta1.MaskingPolicyColumnSource) datasafesdk.ColumnSourceDetails {
	normalized := maskingPolicyAPIColumnSource(source)
	if normalized.ColumnSource == string(datasafesdk.ColumnSourceDetailsColumnSourceSensitiveDataModel) {
		return datasafesdk.ColumnSourceFromSdmDetails{SensitiveDataModelId: common.String(normalized.SensitiveDataModelId)}
	}
	return datasafesdk.ColumnSourceFromTargetDetails{TargetId: common.String(normalized.TargetId)}
}

func getMaskingPolicyResponses(
	t *testing.T,
	responses *[]datasafesdk.GetMaskingPolicyResponse,
) func(context.Context, datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
		requireStringPtr(t, "GetMaskingPolicyRequest.MaskingPolicyId", request.MaskingPolicyId, testMaskingPolicyID)
		if len(*responses) == 0 {
			return datasafesdk.GetMaskingPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "MaskingPolicy is gone")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func requireMaskingPolicyCreateRequest(
	t *testing.T,
	request datasafesdk.CreateMaskingPolicyRequest,
	resource *datasafev1beta1.MaskingPolicy,
) {
	t.Helper()
	requireStringPtr(t, "CreateMaskingPolicyDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateMaskingPolicyDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateMaskingPolicyDetails.Description", request.Description, resource.Spec.Description)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateMaskingPolicyRequest.OpcRetryToken is empty")
	}
	requireBoolPtr(t, "CreateMaskingPolicyDetails.IsDropTempTablesEnabled", request.IsDropTempTablesEnabled, resource.Spec.IsDropTempTablesEnabled)
	requireBoolPtr(t, "CreateMaskingPolicyDetails.IsRedoLoggingEnabled", request.IsRedoLoggingEnabled, resource.Spec.IsRedoLoggingEnabled)
	requireBoolPtr(t, "CreateMaskingPolicyDetails.IsRefreshStatsEnabled", request.IsRefreshStatsEnabled, resource.Spec.IsRefreshStatsEnabled)
	if request.Recompile != datasafesdk.MaskingPolicyRecompileEnum(maskingPolicyCanonicalRecompile(resource.Spec.Recompile)) {
		t.Fatalf("CreateMaskingPolicyDetails.Recompile = %q, want %q", request.Recompile, maskingPolicyCanonicalRecompile(resource.Spec.Recompile))
	}
	requireCreateTargetColumnSource(t, request.ColumnSource, testMaskingPolicyTargetID)
}

func requireCreateTargetColumnSource(t *testing.T, got datasafesdk.CreateColumnSourceDetails, wantTargetID string) {
	t.Helper()
	source, ok := got.(datasafesdk.CreateColumnSourceFromTargetDetails)
	if !ok {
		t.Fatalf("ColumnSource = %T, want CreateColumnSourceFromTargetDetails", got)
	}
	requireStringPtr(t, "ColumnSource.TargetId", source.TargetId, wantTargetID)
}

func requireCreateSDMColumnSource(t *testing.T, got datasafesdk.CreateColumnSourceDetails, wantSensitiveDataModelID string) {
	t.Helper()
	source, ok := got.(datasafesdk.CreateColumnSourceFromSdmDetails)
	if !ok {
		t.Fatalf("ColumnSource = %T, want CreateColumnSourceFromSdmDetails", got)
	}
	requireStringPtr(t, "ColumnSource.SensitiveDataModelId", source.SensitiveDataModelId, wantSensitiveDataModelID)
}

func requireUpdateTargetColumnSource(t *testing.T, got datasafesdk.UpdateColumnSourceDetails, wantTargetID string) {
	t.Helper()
	source, ok := got.(datasafesdk.UpdateColumnSourceTargetDetails)
	if !ok {
		t.Fatalf("ColumnSource = %T, want UpdateColumnSourceTargetDetails", got)
	}
	requireStringPtr(t, "ColumnSource.TargetId", source.TargetId, wantTargetID)
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertMaskingPolicyRecordedID(t *testing.T, resource *datasafev1beta1.MaskingPolicy, want string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertMaskingPolicyOpcRequestID(t *testing.T, resource *datasafev1beta1.MaskingPolicy, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertMaskingPolicyCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireLastCondition(t *testing.T, resource *datasafev1beta1.MaskingPolicy, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions empty, want last condition %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
