/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	onesubscriptionsdk "github.com/oracle/oci-go-sdk/v65/onesubscription"
	onesubscriptionv1beta1 "github.com/oracle/oci-service-operator/api/onesubscription/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCompartmentID  = "ocid1.compartment.oc1..exampleuniqueID"
	testSubscriptionID = "example-line-subscription-id"
	testServiceName    = "Oracle Analytics"
	testBuyerEmail     = "buyer@example.com"
	testOpcRequestID   = "opc-request-subscription-1"
)

type fakeSubscriptionOCIClient struct {
	listFn func(context.Context, onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error)

	listRequests []onesubscriptionsdk.ListSubscriptionsRequest
}

func (f *fakeSubscriptionOCIClient) ListSubscriptions(
	ctx context.Context,
	request onesubscriptionsdk.ListSubscriptionsRequest,
) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return onesubscriptionsdk.ListSubscriptionsResponse{}, nil
}

func TestSubscriptionCreateOrUpdateObservesSingleSummaryWithoutTrackedOCID(t *testing.T) {
	t.Parallel()

	resource := subscriptionResource()
	resource.Status.OsokStatus.Ocid = "stale-tracked-id"

	client := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
			requireStringPtr(t, "compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "subscriptionId", request.SubscriptionId, testSubscriptionID)
			if request.PlanNumber != nil || request.BuyerEmail != nil {
				t.Fatalf("unexpected alternate filters in request: %#v", request)
			}
			return onesubscriptionsdk.ListSubscriptionsResponse{
				Items:        []onesubscriptionsdk.SubscriptionSummary{sdkSubscriptionSummary("TERMINATED")},
				OpcRequestId: common.String(testOpcRequestID),
			}, nil
		},
	}

	response, err := newSubscriptionServiceClientWithOCIClient(testSubscriptionLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want stable success without requeue", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.ocid = %q, want empty because onesubscription/Subscription has no stable top-level OCI identity", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != testOpcRequestID {
		t.Fatalf("status.opcRequestId = %q, want %q", got, testOpcRequestID)
	}
	if got := resource.Status.Status; got != "TERMINATED" {
		t.Fatalf("status.sdkStatus = %q, want TERMINATED", got)
	}
	if got := resource.Status.ServiceName; got != testServiceName {
		t.Fatalf("status.serviceName = %q, want %q", got, testServiceName)
	}
	if len(resource.Status.SubscribedServices) != 1 {
		t.Fatalf("status.subscribedServices length = %d, want 1", len(resource.Status.SubscribedServices))
	}
	if got := lastConditionType(t, resource); got != "Active" {
		t.Fatalf("last condition type = %q, want Active", got)
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("ListSubscriptions() calls = %d, want 1", len(client.listRequests))
	}
}

func TestSubscriptionCreateOrUpdateRequeuesClearlyTransitionalStatus(t *testing.T) {
	t.Parallel()

	resource := subscriptionResource()
	client := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
			requireStringPtr(t, "compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "subscriptionId", request.SubscriptionId, testSubscriptionID)
			return onesubscriptionsdk.ListSubscriptionsResponse{
				Items: []onesubscriptionsdk.SubscriptionSummary{sdkSubscriptionSummary("PENDING_ACTIVATION")},
			}, nil
		},
	}

	response, err := newSubscriptionServiceClientWithOCIClient(testSubscriptionLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful provisioning requeue", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != "Provisioning" {
		t.Fatalf("status.reason = %q, want Provisioning", got)
	}
	if got := lastConditionType(t, resource); got != "Provisioning" {
		t.Fatalf("last condition type = %q, want Provisioning", got)
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "PENDING_ACTIVATION") {
		t.Fatalf("status.message = %q, want PENDING_ACTIVATION detail", got)
	}
}

func TestSubscriptionCreateOrUpdateRejectsInvalidQueryContract(t *testing.T) {
	t.Parallel()

	resource := subscriptionResource()
	resource.Spec.SubscriptionId = ""

	client := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
			t.Fatalf("ListSubscriptions(%#v) should not be called when the local query validation fails", request)
			return onesubscriptionsdk.ListSubscriptionsResponse{}, nil
		},
	}

	response, err := newSubscriptionServiceClientWithOCIClient(testSubscriptionLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "exactly one of planNumber, subscriptionId, or buyerEmail") {
		t.Fatalf("CreateOrUpdate() error = %v, want local query validation failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response after invalid query contract", response)
	}
	if len(client.listRequests) != 0 {
		t.Fatalf("ListSubscriptions() calls = %d, want 0", len(client.listRequests))
	}
}

func TestSubscriptionCreateOrUpdateAggregatesPagesAndRejectsMultipleMatches(t *testing.T) {
	t.Parallel()

	resource := subscriptionResource()
	client := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
			switch page := stringPtrValue(request.Page); page {
			case "":
				return onesubscriptionsdk.ListSubscriptionsResponse{
					Items:       []onesubscriptionsdk.SubscriptionSummary{sdkSubscriptionSummary("ACTIVE")},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return onesubscriptionsdk.ListSubscriptionsResponse{
					Items: []onesubscriptionsdk.SubscriptionSummary{sdkSubscriptionSummary("ACTIVE")},
				}, nil
			default:
				t.Fatalf("unexpected page token %q", page)
				return onesubscriptionsdk.ListSubscriptionsResponse{}, nil
			}
		},
	}

	response, err := newSubscriptionServiceClientWithOCIClient(testSubscriptionLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "returned 2 matches") {
		t.Fatalf("CreateOrUpdate() error = %v, want multiple-match failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response after ambiguous query result", response)
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListSubscriptions() calls = %d, want 2 paginated calls", len(client.listRequests))
	}
}

func TestSubscriptionDeleteIsKubernetesLocalOnly(t *testing.T) {
	t.Parallel()

	resource := subscriptionResource()
	resource.Status.OsokStatus.Ocid = "stale-tracked-id"
	resource.Status.ServiceName = testServiceName

	deleted, err := newSubscriptionServiceClientWithOCIClient(testSubscriptionLogger(), &fakeSubscriptionOCIClient{}).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for local finalizer cleanup")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.ocid = %q, want cleared local-only delete state", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want timestamp after local delete")
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "released from Kubernetes control") {
		t.Fatalf("status.message = %q, want local delete note", got)
	}
	if got := lastConditionType(t, resource); got != "Terminating" {
		t.Fatalf("last condition type = %q, want Terminating", got)
	}
}

func subscriptionResource() *onesubscriptionv1beta1.Subscription {
	return &onesubscriptionv1beta1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onesubscription-subscription-sample",
			Namespace: "default",
		},
		Spec: onesubscriptionv1beta1.SubscriptionSpec{
			CompartmentId:  testCompartmentID,
			SubscriptionId: testSubscriptionID,
		},
	}
}

func sdkSubscriptionSummary(rawStatus string) onesubscriptionsdk.SubscriptionSummary {
	return onesubscriptionsdk.SubscriptionSummary{
		Status:      common.String(rawStatus),
		ServiceName: common.String(testServiceName),
		TimeStart:   sdkTime("2026-01-02T03:04:05Z"),
		TimeEnd:     sdkTime("2027-01-02T03:04:05Z"),
		Currency: &onesubscriptionsdk.SubscriptionCurrency{
			IsoCode:      common.String("USD"),
			Name:         common.String("US Dollar"),
			StdPrecision: common.Int64(2),
		},
		HoldReason: common.String("none"),
		SubscribedServices: []onesubscriptionsdk.SubscriptionSubscribedService{
			{
				Id:       common.String("subscribed-service-1"),
				Status:   common.String(rawStatus),
				Quantity: common.String("1"),
				Product: &onesubscriptionsdk.SubscriptionProduct{
					Name:       common.String("Analytics Cloud"),
					PartNumber: common.String("B12345"),
				},
			},
		},
	}
}

func sdkTime(raw string) *common.SDKTime {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		panic(err)
	}
	value := common.SDKTime{Time: parsed}
	return &value
}

func testSubscriptionLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("onesubscription-subscription-runtime-test")}
}

func requireStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()
	if value == nil || *value != want {
		t.Fatalf("%s = %v, want %q", label, value, want)
	}
}

func lastConditionType(t *testing.T, resource *onesubscriptionv1beta1.Subscription) string {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want at least one condition")
	}
	return string(conditions[len(conditions)-1].Type)
}
