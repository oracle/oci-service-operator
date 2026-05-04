/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package waaspolicy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeWaasPolicyOCIClient struct {
	changeRequests []waassdk.ChangeWaasPolicyCompartmentRequest
	createRequests []waassdk.CreateWaasPolicyRequest
	getRequests    []waassdk.GetWaasPolicyRequest
	listRequests   []waassdk.ListWaasPoliciesRequest
	updateRequests []waassdk.UpdateWaasPolicyRequest
	deleteRequests []waassdk.DeleteWaasPolicyRequest
	workRequests   []waassdk.GetWorkRequestRequest

	changeFunc func(context.Context, waassdk.ChangeWaasPolicyCompartmentRequest) (waassdk.ChangeWaasPolicyCompartmentResponse, error)
	createFunc func(context.Context, waassdk.CreateWaasPolicyRequest) (waassdk.CreateWaasPolicyResponse, error)
	getFunc    func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error)
	listFunc   func(context.Context, waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error)
	updateFunc func(context.Context, waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error)
	deleteFunc func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error)
	workFunc   func(context.Context, waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error)
}

func (c *fakeWaasPolicyOCIClient) ChangeWaasPolicyCompartment(
	ctx context.Context,
	request waassdk.ChangeWaasPolicyCompartmentRequest,
) (waassdk.ChangeWaasPolicyCompartmentResponse, error) {
	c.changeRequests = append(c.changeRequests, request)
	if c.changeFunc != nil {
		return c.changeFunc(ctx, request)
	}
	return waassdk.ChangeWaasPolicyCompartmentResponse{}, nil
}

func (c *fakeWaasPolicyOCIClient) CreateWaasPolicy(
	ctx context.Context,
	request waassdk.CreateWaasPolicyRequest,
) (waassdk.CreateWaasPolicyResponse, error) {
	c.createRequests = append(c.createRequests, request)
	if c.createFunc != nil {
		return c.createFunc(ctx, request)
	}
	return waassdk.CreateWaasPolicyResponse{}, nil
}

func (c *fakeWaasPolicyOCIClient) GetWaasPolicy(
	ctx context.Context,
	request waassdk.GetWaasPolicyRequest,
) (waassdk.GetWaasPolicyResponse, error) {
	c.getRequests = append(c.getRequests, request)
	if c.getFunc != nil {
		return c.getFunc(ctx, request)
	}
	return waassdk.GetWaasPolicyResponse{}, nil
}

func (c *fakeWaasPolicyOCIClient) ListWaasPolicies(
	ctx context.Context,
	request waassdk.ListWaasPoliciesRequest,
) (waassdk.ListWaasPoliciesResponse, error) {
	c.listRequests = append(c.listRequests, request)
	if c.listFunc != nil {
		return c.listFunc(ctx, request)
	}
	return waassdk.ListWaasPoliciesResponse{}, nil
}

func (c *fakeWaasPolicyOCIClient) UpdateWaasPolicy(
	ctx context.Context,
	request waassdk.UpdateWaasPolicyRequest,
) (waassdk.UpdateWaasPolicyResponse, error) {
	c.updateRequests = append(c.updateRequests, request)
	if c.updateFunc != nil {
		return c.updateFunc(ctx, request)
	}
	return waassdk.UpdateWaasPolicyResponse{}, nil
}

func (c *fakeWaasPolicyOCIClient) DeleteWaasPolicy(
	ctx context.Context,
	request waassdk.DeleteWaasPolicyRequest,
) (waassdk.DeleteWaasPolicyResponse, error) {
	c.deleteRequests = append(c.deleteRequests, request)
	if c.deleteFunc != nil {
		return c.deleteFunc(ctx, request)
	}
	return waassdk.DeleteWaasPolicyResponse{}, nil
}

func (c *fakeWaasPolicyOCIClient) GetWorkRequest(
	ctx context.Context,
	request waassdk.GetWorkRequestRequest,
) (waassdk.GetWorkRequestResponse, error) {
	c.workRequests = append(c.workRequests, request)
	if c.workFunc != nil {
		return c.workFunc(ctx, request)
	}
	return waassdk.GetWorkRequestResponse{}, nil
}

func TestWaasPolicyRuntimeSemanticsTracksWaasWorkRequests(t *testing.T) {
	semantics := waasPolicyRuntimeSemantics()
	if semantics.Async == nil || semantics.Async.Strategy != "workrequest" || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest semantics", semantics.Async)
	}
	if semantics.Async.WorkRequest == nil || semantics.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest = %#v, want service SDK work request tracking", semantics.Async.WorkRequest)
	}
	assertWaasPolicyStrings(t, "Async.WorkRequest.Phases", semantics.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if semantics.CreateFollowUp.Strategy != "workrequest" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want workrequest", semantics.CreateFollowUp.Strategy)
	}
	if semantics.UpdateFollowUp.Strategy != "workrequest" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want workrequest", semantics.UpdateFollowUp.Strategy)
	}
}

func TestWaasPolicyCreateUsesDeterministicRetryTokenAndProjectsListReadback(t *testing.T) {
	resource := newTestWaasPolicy()
	fake := &fakeWaasPolicyOCIClient{}
	fake.listFunc = func(_ context.Context, request waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error) {
		return waassdk.ListWaasPoliciesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request waassdk.CreateWaasPolicyRequest) (waassdk.CreateWaasPolicyResponse, error) {
		requireWaasPolicyCreateRequest(t, request, resource)
		return waassdk.CreateWaasPolicyResponse{
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
		}, nil
	}
	fake.workFunc = func(_ context.Context, request waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
		if got := waasPolicyStringValue(request.WorkRequestId); got != "wr-create" {
			t.Fatalf("GetWorkRequest id = %q, want wr-create", got)
		}
		if len(fake.workRequests) == 1 {
			return waassdk.GetWorkRequestResponse{
				WorkRequest: sdkWaasPolicyWorkRequest("wr-create", waassdk.WorkRequestOperationTypeCreateWaasPolicy, waassdk.WorkRequestStatusValuesInProgress, nil),
			}, nil
		}
		return waassdk.GetWorkRequestResponse{
			WorkRequest: sdkWaasPolicyWorkRequest("wr-create", waassdk.WorkRequestOperationTypeCreateWaasPolicy, waassdk.WorkRequestStatusValuesSucceeded, []waassdk.WorkRequestResource{
				sdkWaasPolicyWorkRequestResource("ocid1.waaspolicy.oc1..created", waassdk.WorkRequestResourceActionTypeCreated),
			}),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		if got := waasPolicyStringValue(request.WaasPolicyId); got != "ocid1.waaspolicy.oc1..created" {
			t.Fatalf("GetWaasPolicy id = %q, want created OCID", got)
		}
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy("ocid1.waaspolicy.oc1..created", resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
		}, nil
	}

	client := newWaasPolicyServiceClientWithOCIClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name},
	})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireWaasPolicyAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	requireWaasPolicyRequeue(t, response)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() resume error = %v", err)
	}
	requireWaasPolicySuccessNoRequeue(t, response)
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateWaasPolicy calls = %d, want 1", len(fake.createRequests))
	}
	requireWaasPolicyTrackedID(t, resource, "ocid1.waaspolicy.oc1..created")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
}

func TestWaasPolicyBindsFromPaginatedListBeforeCreate(t *testing.T) {
	resource := newTestWaasPolicy()
	fake := &fakeWaasPolicyOCIClient{}
	fake.listFunc = func(_ context.Context, request waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error) {
		switch waasPolicyStringValue(request.Page) {
		case "":
			return waassdk.ListWaasPoliciesResponse{
				Items: []waassdk.WaasPolicySummary{
					sdkWaasPolicySummary("ocid1.waaspolicy.oc1..other", resource.Spec.CompartmentId, "other.example.com", "other", waassdk.LifecycleStatesActive),
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return waassdk.ListWaasPoliciesResponse{
				Items: []waassdk.WaasPolicySummary{
					sdkWaasPolicySummary("ocid1.waaspolicy.oc1..existing", resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
				},
			}, nil
		default:
			t.Fatalf("ListWaasPolicies page = %q, want empty or page-2", waasPolicyStringValue(request.Page))
			return waassdk.ListWaasPoliciesResponse{}, nil
		}
	}
	fake.getFunc = func(_ context.Context, request waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		if got := waasPolicyStringValue(request.WaasPolicyId); got != "ocid1.waaspolicy.oc1..existing" {
			t.Fatalf("GetWaasPolicy id = %q, want existing OCID", got)
		}
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy("ocid1.waaspolicy.oc1..existing", resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
		}, nil
	}

	response, err := newWaasPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateWaasPolicy calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListWaasPolicies calls = %d, want 2", len(fake.listRequests))
	}
	if got := resource.Status.Id; got != "ocid1.waaspolicy.oc1..existing" {
		t.Fatalf("status.id = %q, want existing OCID", got)
	}
}

func TestWaasPolicyNoopReconcileSkipsUpdate(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Status.Id = "ocid1.waaspolicy.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error) {
		t.Fatal("UpdateWaasPolicy should not be called when mutable state matches")
		return waassdk.UpdateWaasPolicyResponse{}, nil
	}

	response, err := newWaasPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateWaasPolicy calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestWaasPolicyMutableUpdateShapesUpdateBody(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Spec.DisplayName = "new-name"
	resource.Spec.AdditionalDomains = []string{"static.example.com"}
	resource.Status.Id = "ocid1.waaspolicy.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeWaasPolicyOCIClient{}
	getCount := 0
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		getCount++
		displayName := "old-name"
		if getCount > 1 {
			displayName = "new-name"
		}
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.Domain, displayName, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error) {
		if got := waasPolicyStringValue(request.WaasPolicyId); got != resource.Status.Id {
			t.Fatalf("UpdateWaasPolicy id = %q, want %q", got, resource.Status.Id)
		}
		if got := waasPolicyStringValue(request.DisplayName); got != "new-name" {
			t.Fatalf("UpdateWaasPolicy displayName = %q, want new-name", got)
		}
		if got, want := request.AdditionalDomains, []string{"static.example.com"}; len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("UpdateWaasPolicy additionalDomains = %#v, want %#v", got, want)
		}
		return waassdk.UpdateWaasPolicyResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.workFunc = func(_ context.Context, request waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
		if got := waasPolicyStringValue(request.WorkRequestId); got != "wr-update" {
			t.Fatalf("GetWorkRequest id = %q, want wr-update", got)
		}
		return waassdk.GetWorkRequestResponse{
			WorkRequest: sdkWaasPolicyWorkRequest("wr-update", waassdk.WorkRequestOperationTypeUpdateWaasPolicy, waassdk.WorkRequestStatusValuesSucceeded, []waassdk.WorkRequestResource{
				sdkWaasPolicyWorkRequestResource(resource.Status.Id, waassdk.WorkRequestResourceActionTypeUpdated),
			}),
		}, nil
	}

	response, err := newWaasPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireWaasPolicySuccessNoRequeue(t, response)
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateWaasPolicy calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestWaasPolicyCreateBodyIncludesExplicitFalseAddressRateLimiting(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Spec.WafConfig.AddressRateLimiting = waasv1beta1.WaasPolicyWafConfigAddressRateLimiting{IsEnabled: false}

	bodyAny, err := buildWaasPolicyCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildWaasPolicyCreateBody() error = %v", err)
	}
	body := bodyAny.(waassdk.CreateWaasPolicyDetails)
	requireWaasPolicyAddressRateLimitingDisabled(t, body.WafConfig.AddressRateLimiting)
}

func TestWaasPolicyUpdateBodyIncludesExplicitFalseAddressRateLimiting(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Spec.WafConfig.AddressRateLimiting = waasv1beta1.WaasPolicyWafConfigAddressRateLimiting{IsEnabled: false}
	current := sdkWaasPolicy("ocid1.waaspolicy.oc1..existing", resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive)
	current.WafConfig.AddressRateLimiting = &waassdk.AddressRateLimiting{IsEnabled: common.Bool(true)}

	bodyAny, updateNeeded, err := buildWaasPolicyUpdateBody(context.Background(), resource, resource.Namespace, current)
	if err != nil {
		t.Fatalf("buildWaasPolicyUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildWaasPolicyUpdateBody() updateNeeded = false, want true for isEnabled drift")
	}
	body := bodyAny.(waassdk.UpdateWaasPolicyDetails)
	requireWaasPolicyAddressRateLimitingDisabled(t, body.WafConfig.AddressRateLimiting)
}

func TestWaasPolicyImmutableDomainDriftRejectedBeforeUpdate(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Spec.Domain = "new.example.com"
	resource.Status.Id = "ocid1.waaspolicy.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, resource.Spec.CompartmentId, "old.example.com", resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error) {
		t.Fatal("UpdateWaasPolicy should not be called after immutable domain drift")
		return waassdk.UpdateWaasPolicyResponse{}, nil
	}

	_, err := newWaasPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable domain drift")
	}
	if !strings.Contains(err.Error(), "require replacement when domain changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want domain replacement failure", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateWaasPolicy calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestWaasPolicyCompartmentDriftUsesMoveOperation(t *testing.T) {
	resource := newTestWaasPolicy()
	currentCompartmentID := resource.Spec.CompartmentId
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	resource.Status.Id = "ocid1.waaspolicy.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, currentCompartmentID, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.changeFunc = func(_ context.Context, request waassdk.ChangeWaasPolicyCompartmentRequest) (waassdk.ChangeWaasPolicyCompartmentResponse, error) {
		if got := waasPolicyStringValue(request.WaasPolicyId); got != resource.Status.Id {
			t.Fatalf("ChangeWaasPolicyCompartment id = %q, want %q", got, resource.Status.Id)
		}
		if got := waasPolicyStringValue(request.CompartmentId); got != resource.Spec.CompartmentId {
			t.Fatalf("ChangeWaasPolicyCompartment compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
		}
		if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
			t.Fatal("ChangeWaasPolicyCompartment opc retry token is empty")
		}
		return waassdk.ChangeWaasPolicyCompartmentResponse{
			OpcRequestId: common.String("opc-move"),
		}, nil
	}
	fake.updateFunc = func(context.Context, waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error) {
		t.Fatal("UpdateWaasPolicy should not be called for compartment drift")
		return waassdk.UpdateWaasPolicyResponse{}, nil
	}
	fake.workFunc = func(context.Context, waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
		t.Fatal("GetWorkRequest should not be called for WAAS compartment move")
		return waassdk.GetWorkRequestResponse{}, nil
	}

	response, err := newWaasPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireWaasPolicyRequeue(t, response)
	if len(fake.changeRequests) != 1 {
		t.Fatalf("ChangeWaasPolicyCompartment calls = %d, want 1", len(fake.changeRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateWaasPolicy calls = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-move" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-move", got)
	}
	requireWaasPolicyTrackedID(t, resource, "ocid1.waaspolicy.oc1..existing")
	requireWaasPolicyAsyncState(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseUpdate, "", shared.OSOKAsyncClassPending)
}

func TestWaasPolicyDeleteStartsWorkRequestAndRetainsFinalizer(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Status.Id = "ocid1.waaspolicy.oc1..delete"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error) {
		return waassdk.DeleteWaasPolicyResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}
	fake.workFunc = func(_ context.Context, request waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
		if got := waasPolicyStringValue(request.WorkRequestId); got != "wr-delete" {
			t.Fatalf("GetWorkRequest id = %q, want wr-delete", got)
		}
		return waassdk.GetWorkRequestResponse{
			WorkRequest: sdkWaasPolicyWorkRequest("wr-delete", waassdk.WorkRequestOperationTypeDeleteWaasPolicy, waassdk.WorkRequestStatusValuesInProgress, []waassdk.WorkRequestResource{
				sdkWaasPolicyWorkRequestResource(resource.Status.Id, waassdk.WorkRequestResourceActionTypeDeleted),
			}),
		}, nil
	}

	deleted, err := newWaasPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while work request is in progress")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteWaasPolicy calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireWaasPolicyAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
}

func TestWaasPolicyDeletePreReadDeletedSkipsDelete(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Status.Id = "ocid1.waaspolicy.oc1..deleted"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, waassdk.LifecycleStatesDeleted),
		}, nil
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error) {
		t.Fatal("DeleteWaasPolicy should not be called after DELETED pre-read")
		return waassdk.DeleteWaasPolicyResponse{}, nil
	}

	deleted, err := newWaasPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED pre-read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteWaasPolicy calls = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestWaasPolicyDeleteWorkRequestSucceededStillWaitsForDeletedReadback(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Status.Id = "ocid1.waaspolicy.oc1..delete"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	getCount := 0
	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		getCount++
		state := waassdk.LifecycleStatesActive
		if getCount >= 3 {
			state = waassdk.LifecycleStatesDeleted
		}
		return waassdk.GetWaasPolicyResponse{
			WaasPolicy: sdkWaasPolicy(resource.Status.Id, resource.Spec.CompartmentId, resource.Spec.Domain, resource.Spec.DisplayName, state),
		}, nil
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error) {
		return waassdk.DeleteWaasPolicyResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}
	fake.workFunc = func(_ context.Context, request waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
		if got := waasPolicyStringValue(request.WorkRequestId); got != "wr-delete" {
			t.Fatalf("GetWorkRequest id = %q, want wr-delete", got)
		}
		return waassdk.GetWorkRequestResponse{
			WorkRequest: sdkWaasPolicyWorkRequest("wr-delete", waassdk.WorkRequestOperationTypeDeleteWaasPolicy, waassdk.WorkRequestStatusValuesSucceeded, []waassdk.WorkRequestResource{
				sdkWaasPolicyWorkRequestResource(resource.Status.Id, waassdk.WorkRequestResourceActionTypeDeleted),
			}),
		}, nil
	}

	client := newWaasPolicyServiceClientWithOCIClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call deleted = true, want false while readback is still ACTIVE")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteWaasPolicy calls = %d, want 1", len(fake.deleteRequests))
	}
	requireWaasPolicyAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call deleted = false, want true after DELETED readback")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteWaasPolicy calls = %d, want no duplicate delete", len(fake.deleteRequests))
	}
}

func TestWaasPolicyDeleteResumesPendingWorkRequestBeforeAuthGuard(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Status.Id = "ocid1.waaspolicy.oc1..pending-delete"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")

	getCalls := 0
	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		getCalls++
		return waassdk.GetWaasPolicyResponse{}, authErr
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error) {
		t.Fatal("DeleteWaasPolicy should not be called while delete work request is pending")
		return waassdk.DeleteWaasPolicyResponse{}, nil
	}
	fake.workFunc = func(_ context.Context, request waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error) {
		if got := waasPolicyStringValue(request.WorkRequestId); got != "wr-delete" {
			t.Fatalf("GetWorkRequest id = %q, want wr-delete", got)
		}
		return waassdk.GetWorkRequestResponse{
			WorkRequest: sdkWaasPolicyWorkRequest("wr-delete", waassdk.WorkRequestOperationTypeDeleteWaasPolicy, waassdk.WorkRequestStatusValuesInProgress, []waassdk.WorkRequestResource{
				sdkWaasPolicyWorkRequestResource(resource.Status.Id, waassdk.WorkRequestResourceActionTypeDeleted),
			}),
		}, nil
	}

	deleted, err := newWaasPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while work request is pending")
	}
	if getCalls != 0 {
		t.Fatalf("GetWaasPolicy calls = %d, want 0 before resuming pending delete work request", getCalls)
	}
	if len(fake.workRequests) != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1", len(fake.workRequests))
	}
	requireWaasPolicyAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
}

func TestWaasPolicyDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Status.Id = "ocid1.waaspolicy.oc1..ambiguous"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")

	fake := &fakeWaasPolicyOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		return waassdk.GetWaasPolicyResponse{}, authErr
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error) {
		t.Fatal("DeleteWaasPolicy should not be called after auth-shaped pre-delete read")
		return waassdk.DeleteWaasPolicyResponse{}, nil
	}

	deleted, err := newWaasPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteWaasPolicy calls = %d, want 0", len(fake.deleteRequests))
	}
}

func TestWaasPolicyCreateBodyDecodesJSONDataHelpers(t *testing.T) {
	resource := newTestWaasPolicy()
	resource.Spec.PolicyConfig = waasv1beta1.WaasPolicyPolicyConfig{
		LoadBalancingMethod: waasv1beta1.WaasPolicyPolicyConfigLoadBalancingMethod{
			JsonData: `{"method":"STICKY_COOKIE","name":"lb","domain":"example.com","expirationTimeInSeconds":60}`,
		},
	}
	resource.Spec.WafConfig = waasv1beta1.WaasPolicyWafConfig{
		AccessRules: []waasv1beta1.WaasPolicyWafConfigAccessRule{
			{
				Name:     "headers",
				Action:   "ALLOW",
				Criteria: []waasv1beta1.WaasPolicyWafConfigAccessRuleCriteria{{Condition: "URL_STARTS_WITH", Value: "/"}},
				ResponseHeaderManipulation: []waasv1beta1.WaasPolicyWafConfigAccessRuleResponseHeaderManipulation{
					{JsonData: `{"action":"ADD_HTTP_RESPONSE_HEADER","header":"X-Test","value":"enabled"}`},
				},
			},
		},
	}

	bodyAny, err := buildWaasPolicyCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildWaasPolicyCreateBody() error = %v", err)
	}
	body := bodyAny.(waassdk.CreateWaasPolicyDetails)
	if _, ok := body.PolicyConfig.LoadBalancingMethod.(waassdk.StickyCookieLoadBalancingMethod); !ok {
		t.Fatalf("loadBalancingMethod type = %T, want StickyCookieLoadBalancingMethod", body.PolicyConfig.LoadBalancingMethod)
	}
	if len(body.WafConfig.AccessRules) != 1 || len(body.WafConfig.AccessRules[0].ResponseHeaderManipulation) != 1 {
		t.Fatalf("responseHeaderManipulation = %#v, want one decoded action", body.WafConfig.AccessRules)
	}
	if _, ok := body.WafConfig.AccessRules[0].ResponseHeaderManipulation[0].(waassdk.AddHttpResponseHeaderAction); !ok {
		t.Fatalf("responseHeaderManipulation[0] type = %T, want AddHttpResponseHeaderAction", body.WafConfig.AccessRules[0].ResponseHeaderManipulation[0])
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}
	if strings.Contains(string(encoded), "jsonData") {
		t.Fatalf("create body leaked jsonData helper field: %s", encoded)
	}
}

func assertWaasPolicyStrings(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", field, got, want)
		}
	}
}

func requireWaasPolicyCreateRequest(
	t *testing.T,
	request waassdk.CreateWaasPolicyRequest,
	resource *waasv1beta1.WaasPolicy,
) {
	t.Helper()
	if request.CompartmentId == nil || *request.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("CreateWaasPolicy compartmentId = %#v, want %q", request.CompartmentId, resource.Spec.CompartmentId)
	}
	if request.Domain == nil || *request.Domain != resource.Spec.Domain {
		t.Fatalf("CreateWaasPolicy domain = %#v, want %q", request.Domain, resource.Spec.Domain)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateWaasPolicy opc retry token is empty")
	}
}

func requireWaasPolicyAddressRateLimitingDisabled(t *testing.T, addressRateLimiting *waassdk.AddressRateLimiting) {
	t.Helper()
	if addressRateLimiting == nil {
		t.Fatal("wafConfig.addressRateLimiting = nil, want explicit disabled settings")
	}
	if addressRateLimiting.IsEnabled == nil {
		t.Fatal("wafConfig.addressRateLimiting.isEnabled = nil, want explicit false")
	}
	if *addressRateLimiting.IsEnabled {
		t.Fatal("wafConfig.addressRateLimiting.isEnabled = true, want false")
	}
}

func requireWaasPolicySuccessNoRequeue(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("response should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("response should not requeue")
	}
}

func requireWaasPolicyRequeue(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("response should report success")
	}
	if !response.ShouldRequeue {
		t.Fatal("response should requeue")
	}
}

func requireWaasPolicyTrackedID(t *testing.T, resource *waasv1beta1.WaasPolicy, want string) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func requireWaasPolicyAsync(
	t *testing.T,
	resource *waasv1beta1.WaasPolicy,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	requireWaasPolicyAsyncState(t, resource, shared.OSOKAsyncSourceWorkRequest, wantPhase, wantWorkRequestID, wantClass)
}

func requireWaasPolicyAsyncState(
	t *testing.T,
	resource *waasv1beta1.WaasPolicy,
	wantSource shared.OSOKAsyncSource,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil")
	}
	if current.Source != wantSource {
		t.Fatalf("async.source = %q, want %q", current.Source, wantSource)
	}
	if current.Phase != wantPhase || current.WorkRequestID != wantWorkRequestID || current.NormalizedClass != wantClass {
		t.Fatalf("async.current = %#v, want phase=%q workRequestID=%q class=%q", current, wantPhase, wantWorkRequestID, wantClass)
	}
}

func newTestWaasPolicy() *waasv1beta1.WaasPolicy {
	return &waasv1beta1.WaasPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy",
			Namespace: "default",
			UID:       types.UID("uid-policy"),
		},
		Spec: waasv1beta1.WaasPolicySpec{
			CompartmentId: "ocid1.compartment.oc1..test",
			Domain:        "app.example.com",
			DisplayName:   "policy",
			Origins: map[string]waasv1beta1.WaasPolicyOrigins{
				"primary": {Uri: "app-origin.example.com"},
			},
			WafConfig: waasv1beta1.WaasPolicyWafConfig{Origin: "primary"},
			FreeformTags: map[string]string{
				"env": "test",
			},
		},
	}
}

func sdkWaasPolicy(
	id string,
	compartmentID string,
	domain string,
	displayName string,
	state waassdk.LifecycleStatesEnum,
) waassdk.WaasPolicy {
	return waassdk.WaasPolicy{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		Domain:         common.String(domain),
		DisplayName:    common.String(displayName),
		Origins:        map[string]waassdk.Origin{"primary": {Uri: common.String("app-origin.example.com")}},
		WafConfig:      &waassdk.WafConfig{Origin: common.String("primary")},
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "test"},
	}
}

func sdkWaasPolicySummary(
	id string,
	compartmentID string,
	domain string,
	displayName string,
	state waassdk.LifecycleStatesEnum,
) waassdk.WaasPolicySummary {
	return waassdk.WaasPolicySummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		Domain:         common.String(domain),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "test"},
	}
}

func sdkWaasPolicyWorkRequest(
	id string,
	operation waassdk.WorkRequestOperationTypesEnum,
	status waassdk.WorkRequestStatusValuesEnum,
	resources []waassdk.WorkRequestResource,
) waassdk.WorkRequest {
	return waassdk.WorkRequest{
		Id:            common.String(id),
		OperationType: operation,
		Status:        status,
		CompartmentId: common.String("ocid1.compartment.oc1..test"),
		Resources:     resources,
	}
}

func sdkWaasPolicyWorkRequestResource(id string, action waassdk.WorkRequestResourceActionTypeEnum) waassdk.WorkRequestResource {
	return waassdk.WorkRequestResource{
		ActionType: action,
		EntityType: common.String("WaasPolicy"),
		Identifier: common.String(id),
	}
}
