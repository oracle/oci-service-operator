/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappfirewall

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCompartmentID          = "ocid1.compartment.oc1..webappfirewall"
	testWebAppFirewallID       = "ocid1.webappfirewall.oc1..resource"
	testWebAppFirewallPolicyID = "ocid1.webappfirewallpolicy.oc1..policy"
	testLoadBalancerID         = "ocid1.loadbalancer.oc1..backend"
)

type fakeWebAppFirewallOCIClient struct {
	createFn func(context.Context, wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error)
	getFn    func(context.Context, wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error)
	listFn   func(context.Context, wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error)
	updateFn func(context.Context, wafsdk.UpdateWebAppFirewallRequest) (wafsdk.UpdateWebAppFirewallResponse, error)
	deleteFn func(context.Context, wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error)
}

func (f *fakeWebAppFirewallOCIClient) CreateWebAppFirewall(
	ctx context.Context,
	request wafsdk.CreateWebAppFirewallRequest,
) (wafsdk.CreateWebAppFirewallResponse, error) {
	if f.createFn == nil {
		return wafsdk.CreateWebAppFirewallResponse{}, unexpectedWebAppFirewallCall("CreateWebAppFirewall")
	}
	return f.createFn(ctx, request)
}

func (f *fakeWebAppFirewallOCIClient) GetWebAppFirewall(
	ctx context.Context,
	request wafsdk.GetWebAppFirewallRequest,
) (wafsdk.GetWebAppFirewallResponse, error) {
	if f.getFn == nil {
		return wafsdk.GetWebAppFirewallResponse{}, unexpectedWebAppFirewallCall("GetWebAppFirewall")
	}
	return f.getFn(ctx, request)
}

func (f *fakeWebAppFirewallOCIClient) ListWebAppFirewalls(
	ctx context.Context,
	request wafsdk.ListWebAppFirewallsRequest,
) (wafsdk.ListWebAppFirewallsResponse, error) {
	if f.listFn == nil {
		return wafsdk.ListWebAppFirewallsResponse{}, unexpectedWebAppFirewallCall("ListWebAppFirewalls")
	}
	return f.listFn(ctx, request)
}

func (f *fakeWebAppFirewallOCIClient) UpdateWebAppFirewall(
	ctx context.Context,
	request wafsdk.UpdateWebAppFirewallRequest,
) (wafsdk.UpdateWebAppFirewallResponse, error) {
	if f.updateFn == nil {
		return wafsdk.UpdateWebAppFirewallResponse{}, unexpectedWebAppFirewallCall("UpdateWebAppFirewall")
	}
	return f.updateFn(ctx, request)
}

func (f *fakeWebAppFirewallOCIClient) DeleteWebAppFirewall(
	ctx context.Context,
	request wafsdk.DeleteWebAppFirewallRequest,
) (wafsdk.DeleteWebAppFirewallResponse, error) {
	if f.deleteFn == nil {
		return wafsdk.DeleteWebAppFirewallResponse{}, unexpectedWebAppFirewallCall("DeleteWebAppFirewall")
	}
	return f.deleteFn(ctx, request)
}

func unexpectedWebAppFirewallCall(name string) error {
	return errortest.NewServiceError(500, "UnexpectedCall", name)
}

type webAppFirewallCreateCalls struct {
	create int
	list   int
	get    int
}

func TestWebAppFirewallCreateOrUpdateCreatesLoadBalancerFirewall(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Spec.BackendType = ""
	calls := &webAppFirewallCreateCalls{}
	client := newCreateLoadBalancerFirewallTestClient(t, resource, calls)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppFirewallRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	requireWebAppFirewallCreateCalls(t, calls, webAppFirewallCreateCalls{create: 1, list: 1, get: 1})
	requireWebAppFirewallStatus(t, resource, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func newCreateLoadBalancerFirewallTestClient(
	t *testing.T,
	resource *wafv1beta1.WebAppFirewall,
	calls *webAppFirewallCreateCalls,
) WebAppFirewallServiceClient {
	t.Helper()
	return newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		listFn: func(_ context.Context, request wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error) {
			calls.list++
			requireStringPtr(t, "ListWebAppFirewallsRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			return wafsdk.ListWebAppFirewallsResponse{}, nil
		},
		createFn: func(_ context.Context, request wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error) {
			calls.create++
			requireCreateLoadBalancerDetails(t, request, resource)
			return wafsdk.CreateWebAppFirewallResponse{
				WebAppFirewall:   sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateCreating),
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			calls.get++
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive),
			}, nil
		},
	})
}

func requireCreateLoadBalancerDetails(
	t *testing.T,
	request wafsdk.CreateWebAppFirewallRequest,
	resource *wafv1beta1.WebAppFirewall,
) {
	t.Helper()
	details, ok := request.CreateWebAppFirewallDetails.(wafsdk.CreateWebAppFirewallLoadBalancerDetails)
	if !ok {
		t.Fatalf("CreateWebAppFirewallDetails type = %T, want CreateWebAppFirewallLoadBalancerDetails", request.CreateWebAppFirewallDetails)
	}
	requireStringPtr(t, "CreateWebAppFirewallDetails.CompartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateWebAppFirewallDetails.WebAppFirewallPolicyId", details.WebAppFirewallPolicyId, resource.Spec.WebAppFirewallPolicyId)
	requireStringPtr(t, "CreateWebAppFirewallDetails.LoadBalancerId", details.LoadBalancerId, resource.Spec.LoadBalancerId)
	requireStringPtr(t, "CreateWebAppFirewallDetails.DisplayName", details.DisplayName, resource.Spec.DisplayName)
	if got := details.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateWebAppFirewallDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateWebAppFirewallDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateWebAppFirewallRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireWebAppFirewallCreateCalls(t *testing.T, got *webAppFirewallCreateCalls, want webAppFirewallCreateCalls) {
	t.Helper()
	if got.create != want.create {
		t.Fatalf("CreateWebAppFirewall() calls = %d, want %d", got.create, want.create)
	}
	if got.list != want.list {
		t.Fatalf("ListWebAppFirewalls() calls = %d, want %d", got.list, want.list)
	}
	if got.get != want.get {
		t.Fatalf("GetWebAppFirewall() calls = %d, want %d", got.get, want.get)
	}
}

func TestWebAppFirewallCreateOrUpdateBindsExistingAcrossPaginatedList(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	var pages []string
	createCalled := false
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		listFn: func(_ context.Context, request wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error) {
			requireStringPtr(t, "ListWebAppFirewallsRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			pages = append(pages, stringValue(request.Page))
			if request.Page == nil {
				return wafsdk.ListWebAppFirewallsResponse{
					WebAppFirewallCollection: wafsdk.WebAppFirewallCollection{
						Items: []wafsdk.WebAppFirewallSummary{
							sdkWebAppFirewallSummary(resource.Spec, "ocid1.webappfirewall.oc1..other", wafsdk.WebAppFirewallLifecycleStateActive, "ocid1.loadbalancer.oc1..other"),
						},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			return wafsdk.ListWebAppFirewallsResponse{
				WebAppFirewallCollection: wafsdk.WebAppFirewallCollection{
					Items: []wafsdk.WebAppFirewallSummary{
						sdkWebAppFirewallSummary(resource.Spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive, resource.Spec.LoadBalancerId),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error) {
			createCalled = true
			return wafsdk.CreateWebAppFirewallResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppFirewallRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateWebAppFirewall() was called, want bind to existing OCI resource")
	}
	if got, want := strings.Join(pages, ","), ",next-page"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	requireWebAppFirewallStatus(t, resource, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive)
}

func TestWebAppFirewallCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	updateCalled := false
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, wafsdk.UpdateWebAppFirewallRequest) (wafsdk.UpdateWebAppFirewallResponse, error) {
			updateCalled = true
			return wafsdk.UpdateWebAppFirewallResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppFirewallRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateWebAppFirewall() was called, want no-op reconcile")
	}
	requireWebAppFirewallStatus(t, resource, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive)
	requireLastCondition(t, resource, shared.Active)
}

func TestWebAppFirewallCreateOrUpdateShapesMutableUpdate(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	observedSpec := resource.Spec
	observedSpec.DisplayName = "old-display"
	observedSpec.WebAppFirewallPolicyId = "ocid1.webappfirewallpolicy.oc1..old"
	observedSpec.FreeformTags = map[string]string{"env": "old"}
	observedSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "7"}}
	observedSpec.SystemTags = map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "false"}}
	getCalls := 0
	updateCalls := 0
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			spec := observedSpec
			if getCalls > 1 {
				spec = resource.Spec
			}
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request wafsdk.UpdateWebAppFirewallRequest) (wafsdk.UpdateWebAppFirewallResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			requireStringPtr(t, "UpdateWebAppFirewallDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "UpdateWebAppFirewallDetails.WebAppFirewallPolicyId", request.WebAppFirewallPolicyId, resource.Spec.WebAppFirewallPolicyId)
			if got := request.FreeformTags["env"]; got != "test" {
				t.Fatalf("UpdateWebAppFirewallDetails.FreeformTags[env] = %q, want test", got)
			}
			if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
				t.Fatalf("UpdateWebAppFirewallDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
			}
			if got := request.SystemTags["orcl-cloud"]["free-tier-retained"]; got != "true" {
				t.Fatalf("UpdateWebAppFirewallDetails.SystemTags[orcl-cloud][free-tier-retained] = %v, want true", got)
			}
			return wafsdk.UpdateWebAppFirewallResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppFirewallRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateWebAppFirewall() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetWebAppFirewall() calls = %d, want pre-update read plus read-after-update", getCalls)
	}
	requireWebAppFirewallStatus(t, resource, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestWebAppFirewallCreateOrUpdateRejectsCreateOnlyLoadBalancerDrift(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	observedSpec := resource.Spec
	observedSpec.LoadBalancerId = "ocid1.loadbalancer.oc1..old"
	updateCalled := false
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(observedSpec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, wafsdk.UpdateWebAppFirewallRequest) (wafsdk.UpdateWebAppFirewallResponse, error) {
			updateCalled = true
			return wafsdk.UpdateWebAppFirewallResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppFirewallRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "loadBalancerId") {
		t.Fatalf("CreateOrUpdate() error = %v, want loadBalancerId drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateWebAppFirewall() was called after create-only drift")
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestWebAppFirewallDeleteKeepsFinalizerWhileDeleting(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	getCalls := 0
	deleteCalls := 0
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			state := wafsdk.WebAppFirewallLifecycleStateActive
			if getCalls > 2 {
				state = wafsdk.WebAppFirewallLifecycleStateDeleting
			}
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, state),
			}, nil
		},
		deleteFn: func(_ context.Context, request wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.DeleteWebAppFirewallResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteWebAppFirewall() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetWebAppFirewall() calls = %d, want guard read, pre-delete read, and post-delete read", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay nil while delete is pending")
	}
	if resource.Status.LifecycleState != string(wafsdk.WebAppFirewallLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestWebAppFirewallDeleteReleasesFinalizerWhenDeleted(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	getCalls := 0
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			state := wafsdk.WebAppFirewallLifecycleStateActive
			if getCalls > 2 {
				state = wafsdk.WebAppFirewallLifecycleStateDeleted
			}
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, state),
			}, nil
		},
		deleteFn: func(_ context.Context, request wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error) {
			requireStringPtr(t, "DeleteWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.DeleteWebAppFirewallResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true once lifecycle is DELETED")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if resource.Status.LifecycleState != string(wafsdk.WebAppFirewallLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", resource.Status.LifecycleState)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestWebAppFirewallDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	deleteCalled := false
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.GetWebAppFirewallResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error) {
			deleteCalled = true
			return wafsdk.DeleteWebAppFirewallResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous auth-shaped read")
	}
	if deleteCalled {
		t.Fatal("DeleteWebAppFirewall() was called after auth-shaped pre-delete read")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay nil for ambiguous auth-shaped read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestWebAppFirewallDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppFirewallID)
	getCalls := 0
	deleteCalls := 0
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		getFn: func(_ context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			if getCalls > 2 {
				return wafsdk.GetWebAppFirewallResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
			}
			return wafsdk.GetWebAppFirewallResponse{
				WebAppFirewall: sdkWebAppFirewall(resource.Spec, testWebAppFirewallID, wafsdk.WebAppFirewallLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteWebAppFirewallRequest.WebAppFirewallId", request.WebAppFirewallId, testWebAppFirewallID)
			return wafsdk.DeleteWebAppFirewallResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirm-read error")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want NotAuthorizedOrNotFound context", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous post-delete confirm read")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteWebAppFirewall() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetWebAppFirewall() calls = %d, want guard read, pre-delete read, and post-delete confirm read", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay nil for ambiguous post-delete confirm read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestWebAppFirewallCreateOrUpdateRecordsCreateServiceErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := testWebAppFirewallResource()
	client := newWebAppFirewallServiceClientWithOCIClient(loggerForWebAppFirewallTest(), &fakeWebAppFirewallOCIClient{
		listFn: func(context.Context, wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error) {
			return wafsdk.ListWebAppFirewallsResponse{}, nil
		},
		createFn: func(context.Context, wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error) {
			return wafsdk.CreateWebAppFirewallResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppFirewallRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func testWebAppFirewallResource() *wafv1beta1.WebAppFirewall {
	return &wafv1beta1.WebAppFirewall{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-waf",
		},
		Spec: wafv1beta1.WebAppFirewallSpec{
			DisplayName:            "test-waf",
			FreeformTags:           map[string]string{"env": "test"},
			DefinedTags:            map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			SystemTags:             map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "true"}},
			CompartmentId:          testCompartmentID,
			WebAppFirewallPolicyId: testWebAppFirewallPolicyID,
			BackendType:            webAppFirewallBackendTypeLoadBalancer,
			LoadBalancerId:         testLoadBalancerID,
		},
	}
}

func makeWebAppFirewallRequest(resource *wafv1beta1.WebAppFirewall) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func sdkWebAppFirewall(
	spec wafv1beta1.WebAppFirewallSpec,
	id string,
	state wafsdk.WebAppFirewallLifecycleStateEnum,
) wafsdk.WebAppFirewallLoadBalancer {
	return wafsdk.WebAppFirewallLoadBalancer{
		Id:                     common.String(id),
		DisplayName:            common.String(spec.DisplayName),
		CompartmentId:          common.String(spec.CompartmentId),
		WebAppFirewallPolicyId: common.String(spec.WebAppFirewallPolicyId),
		LifecycleState:         state,
		FreeformTags:           cloneWebAppFirewallFreeformTags(spec.FreeformTags),
		DefinedTags:            webAppFirewallDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:             webAppFirewallDefinedTagsFromSpec(spec.SystemTags),
		LoadBalancerId:         common.String(spec.LoadBalancerId),
	}
}

func sdkWebAppFirewallSummary(
	spec wafv1beta1.WebAppFirewallSpec,
	id string,
	state wafsdk.WebAppFirewallLifecycleStateEnum,
	loadBalancerID string,
) wafsdk.WebAppFirewallLoadBalancerSummary {
	spec.LoadBalancerId = loadBalancerID
	return wafsdk.WebAppFirewallLoadBalancerSummary{
		Id:                     common.String(id),
		DisplayName:            common.String(spec.DisplayName),
		CompartmentId:          common.String(spec.CompartmentId),
		WebAppFirewallPolicyId: common.String(spec.WebAppFirewallPolicyId),
		LifecycleState:         state,
		FreeformTags:           cloneWebAppFirewallFreeformTags(spec.FreeformTags),
		DefinedTags:            webAppFirewallDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:             webAppFirewallDefinedTagsFromSpec(spec.SystemTags),
		LoadBalancerId:         common.String(spec.LoadBalancerId),
	}
}

func requireWebAppFirewallStatus(
	t *testing.T,
	resource *wafv1beta1.WebAppFirewall,
	wantID string,
	wantState wafsdk.WebAppFirewallLifecycleStateEnum,
) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if resource.Status.Id != wantID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, wantID)
	}
	if resource.Status.LifecycleState != string(wantState) {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, wantState)
	}
	if resource.Status.BackendType != webAppFirewallBackendTypeLoadBalancer {
		t.Fatalf("status.backendType = %q, want LOAD_BALANCER", resource.Status.BackendType)
	}
	if resource.Status.LoadBalancerId != testLoadBalancerID {
		t.Fatalf("status.loadBalancerId = %q, want %q", resource.Status.LoadBalancerId, testLoadBalancerID)
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

func requireLastCondition(t *testing.T, resource *wafv1beta1.WebAppFirewall, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last status condition = %s, want %s", got, want)
	}
}

func loggerForWebAppFirewallTest() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
