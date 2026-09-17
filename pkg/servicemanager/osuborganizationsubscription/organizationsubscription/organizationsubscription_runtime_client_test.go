/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package organizationsubscription

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	osuborganizationsubscriptionsdk "github.com/oracle/oci-go-sdk/v65/osuborganizationsubscription"
	osuborganizationsubscriptionv1beta1 "github.com/oracle/oci-service-operator/api/osuborganizationsubscription/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOrganizationSubscriptionOCIClient struct {
	listFn   func(context.Context, osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error)
	requests []osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest
}

func (f *fakeOrganizationSubscriptionOCIClient) ListOrganizationSubscriptions(
	ctx context.Context,
	req osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest,
) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
	f.requests = append(f.requests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{}, nil
}

func testOrganizationSubscriptionClient(fake *fakeOrganizationSubscriptionOCIClient) OrganizationSubscriptionServiceClient {
	return newOrganizationSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeOrganizationSubscriptionResource() *osuborganizationsubscriptionv1beta1.OrganizationSubscription {
	return &osuborganizationsubscriptionv1beta1.OrganizationSubscription{
		ObjectMeta: metav1ObjectMeta("organizationsubscription-sample", map[string]string{
			OrganizationSubscriptionCompartmentIDAnnotation:  "ocid1.compartment.oc1..example",
			OrganizationSubscriptionSubscriptionIDAnnotation: "sub-active",
		}),
	}
}

func metav1ObjectMeta(name string, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   "default",
		Annotations: annotations,
	}
}

func makeSDKOrganizationSubscriptionSummary(
	id string,
	serviceName string,
	status string,
) osuborganizationsubscriptionsdk.SubscriptionSummary {
	return osuborganizationsubscriptionsdk.SubscriptionSummary{
		Id:          common.String(id),
		ServiceName: common.String(serviceName),
		Status:      common.String(status),
	}
}

func requireOrganizationSubscriptionListRequest(
	t *testing.T,
	req osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest,
	wantCompartmentID string,
	wantSubscriptionID string,
) {
	t.Helper()

	if req.CompartmentId == nil || *req.CompartmentId != wantCompartmentID {
		t.Fatalf("list compartmentId = %v, want %q", req.CompartmentId, wantCompartmentID)
	}
	if req.SubscriptionIds == nil || *req.SubscriptionIds != wantSubscriptionID {
		t.Fatalf("list subscriptionIds = %v, want %q", req.SubscriptionIds, wantSubscriptionID)
	}
	if req.Limit == nil || *req.Limit != organizationSubscriptionListLimit {
		t.Fatalf("list limit = %v, want reviewed limit", req.Limit)
	}
}

func requireOrganizationSubscriptionActiveStatus(
	t *testing.T,
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
	wantSubscriptionID string,
	wantRequestID string,
) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != wantSubscriptionID {
		t.Fatalf("status.status.ocid = %q, want subscription id", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want list request id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want Active", got)
	}
	if resource.Status.Id != wantSubscriptionID || resource.Status.ServiceName != "Compute" || resource.Status.Status != "ACTIVE" {
		t.Fatalf("projected OrganizationSubscription status = %+v", resource.Status)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 ||
		resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Active {
		t.Fatalf("status conditions = %#v, want trailing Active condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestOrganizationSubscriptionRuntimeSemanticsDocumentReadOnlySDKSurface(t *testing.T) {
	t.Parallel()

	semantics := organizationSubscriptionRuntimeSemantics()
	if semantics == nil {
		t.Fatal("organizationSubscriptionRuntimeSemantics() = nil")
	}
	if semantics.Async == nil || semantics.Async.Strategy != "none" {
		t.Fatalf("async semantics = %#v, want no async for list-only SDK surface", semantics.Async)
	}
	if got := semantics.FinalizerPolicy; got != "unbind-only-read-only-resource" {
		t.Fatalf("finalizer policy = %q, want unbind-only read-only policy", got)
	}
	if got := semantics.Delete.Policy; got != "best-effort" {
		t.Fatalf("delete policy = %q, want best-effort unbind", got)
	}
	if len(semantics.Unsupported) != 2 {
		t.Fatalf("unsupported semantics = %#v, want SDK and CRD-shape stop conditions", semantics.Unsupported)
	}
	if !strings.Contains(semantics.Unsupported[0].StopCondition, "ListOrganizationSubscriptions only") {
		t.Fatalf("sdk unsupported stop condition = %q, want list-only SDK explanation", semantics.Unsupported[0].StopCondition)
	}
	if !strings.Contains(semantics.Unsupported[1].StopCondition, "metadata annotations") {
		t.Fatalf("crd unsupported stop condition = %q, want annotation identity explanation", semantics.Unsupported[1].StopCondition)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateBindsListedSubscriptionAndProjectsStatus(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(_ context.Context, req osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			requireOrganizationSubscriptionListRequest(t, req, "ocid1.compartment.oc1..example", "sub-active")
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
				Items: []osuborganizationsubscriptionsdk.SubscriptionSummary{
					makeSDKOrganizationSubscriptionSummary("sub-active", "Compute", "ACTIVE"),
				},
				OpcRequestId: common.String("opc-list-1"),
			}, nil
		},
	}

	resource := makeOrganizationSubscriptionResource()
	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for a listed read-only subscription")
	}
	requireOrganizationSubscriptionActiveStatus(t, resource, "sub-active", "opc-list-1")
	if len(fake.requests) != 1 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 1", len(fake.requests))
	}
}

func TestOrganizationSubscriptionCreateOrUpdateIsNoopWhenTrackedSubscriptionStillExists(t *testing.T) {
	t.Parallel()

	listCalls := 0
	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(context.Context, osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			listCalls++
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
				Items: []osuborganizationsubscriptionsdk.SubscriptionSummary{
					makeSDKOrganizationSubscriptionSummary("sub-active", "Compute", "ACTIVE"),
				},
				OpcRequestId: common.String("opc-list-noop"),
			}, nil
		},
	}

	resource := makeOrganizationSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("sub-active")
	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op without requeue", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 1", listCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-list-noop" {
		t.Fatalf("status.status.opcRequestId = %q, want no-op list request id", got)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateUsesTrackedSubscriptionWhenAnnotationIsRemoved(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(_ context.Context, req osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			requireOrganizationSubscriptionListRequest(t, req, "ocid1.compartment.oc1..example", "sub-active")
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
				Items: []osuborganizationsubscriptionsdk.SubscriptionSummary{
					makeSDKOrganizationSubscriptionSummary("sub-active", "Compute", "ACTIVE"),
				},
				OpcRequestId: common.String("opc-list-tracked"),
			}, nil
		},
	}

	resource := makeOrganizationSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("sub-active")
	delete(resource.Annotations, OrganizationSubscriptionSubscriptionIDAnnotation)

	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful tracked-identity readback", response)
	}
	requireOrganizationSubscriptionActiveStatus(t, resource, "sub-active", "opc-list-tracked")
}

func TestOrganizationSubscriptionCreateOrUpdateBindsFromLaterListPage(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(_ context.Context, req osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			switch organizationSubscriptionString(req.Page) {
			case "":
				return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
					Items: []osuborganizationsubscriptionsdk.SubscriptionSummary{
						makeSDKOrganizationSubscriptionSummary("sub-other", "Storage", "ACTIVE"),
					},
					OpcNextPage:  common.String("page-2"),
					OpcRequestId: common.String("opc-list-page-1"),
				}, nil
			case "page-2":
				return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
					Items: []osuborganizationsubscriptionsdk.SubscriptionSummary{
						makeSDKOrganizationSubscriptionSummary("sub-active", "Compute", "ACTIVE"),
					},
					OpcRequestId: common.String("opc-list-page-2"),
				}, nil
			default:
				t.Fatalf("list page = %q, want empty or page-2", organizationSubscriptionString(req.Page))
				return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{}, nil
			}
		},
	}

	resource := makeOrganizationSubscriptionResource()
	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "sub-active" {
		t.Fatalf("status.status.ocid = %q, want later-page subscription", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-list-page-2" {
		t.Fatalf("status.status.opcRequestId = %q, want matched page request id", got)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 2", len(fake.requests))
	}
}

func TestOrganizationSubscriptionCreateOrUpdateSendsOriginRegionHeader(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(_ context.Context, req osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			if req.XOneOriginRegion == nil || *req.XOneOriginRegion != "us-phoenix-1" {
				t.Fatalf("list x-one-origin-region = %v, want us-phoenix-1", req.XOneOriginRegion)
			}
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
				Items: []osuborganizationsubscriptionsdk.SubscriptionSummary{
					makeSDKOrganizationSubscriptionSummary("sub-active", "Compute", "ACTIVE"),
				},
				OpcRequestId: common.String("opc-list-origin"),
			}, nil
		},
	}

	resource := makeOrganizationSubscriptionResource()
	resource.Annotations[OrganizationSubscriptionOriginRegionAnnotation] = "us-phoenix-1"

	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	requireOrganizationSubscriptionActiveStatus(t, resource, "sub-active", "opc-list-origin")
}

func TestOrganizationSubscriptionCreateOrUpdateRejectsMissingLookupAnnotationsBeforeOCI(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{}
	resource := &osuborganizationsubscriptionv1beta1.OrganizationSubscription{
		ObjectMeta: metav1ObjectMeta("organizationsubscription-sample", nil),
	}

	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing annotation error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if len(fake.requests) != 0 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 0 before OCI", len(fake.requests))
	}
	if !strings.Contains(err.Error(), OrganizationSubscriptionCompartmentIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %q, want compartment annotation message", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", got)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateRejectsMultipleSubscriptionIDsBeforeOCI(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{}
	resource := makeOrganizationSubscriptionResource()
	delete(resource.Annotations, OrganizationSubscriptionSubscriptionIDAnnotation)
	resource.Annotations[OrganizationSubscriptionSubscriptionIDsAnnotation] = "sub-one, sub-two"

	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want multiple subscription IDs rejected")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if len(fake.requests) != 0 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 0 before OCI", len(fake.requests))
	}
	if !strings.Contains(err.Error(), "exactly one subscription id") {
		t.Fatalf("CreateOrUpdate() error = %q, want exact single-ID message", err)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateRejectsTrackedIdentityDriftBeforeOCI(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{}
	resource := makeOrganizationSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("sub-original")
	resource.Annotations[OrganizationSubscriptionSubscriptionIDAnnotation] = "sub-replacement"

	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want tracked identity drift rejected")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if len(fake.requests) != 0 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 0 before OCI", len(fake.requests))
	}
	if !strings.Contains(err.Error(), "create a replacement resource") {
		t.Fatalf("CreateOrUpdate() error = %q, want replacement guidance", err)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateMarksFailureWhenListedSubscriptionMissing(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(context.Context, osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
				Items:        []osuborganizationsubscriptionsdk.SubscriptionSummary{},
				OpcRequestId: common.String("opc-list-empty"),
			}, nil
		},
	}

	resource := makeOrganizationSubscriptionResource()
	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want not-found failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("CreateOrUpdate() error = %q, want not-found message", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", got)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(context.Context, osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "auth-shaped read failure")
		},
	}

	resource := makeOrganizationSubscriptionResource()
	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want surfaced OCI request id", got)
	}
}

func TestOrganizationSubscriptionDeleteOnlyRemovesKubernetesBinding(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{}
	resource := makeOrganizationSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("sub-active")

	deleted, err := testOrganizationSubscriptionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release for read-only SDK surface")
	}
	if len(fake.requests) != 0 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 0 during read-only delete", len(fake.requests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.status.reason = %q, want Terminating", got)
	}
}

func TestOrganizationSubscriptionCreateOrUpdateRejectsRepeatedPaginationToken(t *testing.T) {
	t.Parallel()

	fake := &fakeOrganizationSubscriptionOCIClient{
		listFn: func(context.Context, osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error) {
			return osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse{
				Items:        []osuborganizationsubscriptionsdk.SubscriptionSummary{},
				OpcNextPage:  common.String("page-loop"),
				OpcRequestId: common.String("opc-list-loop"),
			}, nil
		},
	}

	resource := makeOrganizationSubscriptionResource()
	response, err := testOrganizationSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want pagination loop error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("CreateOrUpdate() error = %q, want pagination loop message", err)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("ListOrganizationSubscriptions calls = %d, want 2 before repeated token detection", len(fake.requests))
	}
}
