/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	osubsubscriptionsdk "github.com/oracle/oci-go-sdk/v65/osubsubscription"
	osubsubscriptionv1beta1 "github.com/oracle/oci-service-operator/api/osubsubscription/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeSubscriptionOCIClient struct {
	listFn       func(context.Context, osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error)
	listRequests []osubsubscriptionsdk.ListSubscriptionsRequest
}

func (f *fakeSubscriptionOCIClient) ListSubscriptions(
	ctx context.Context,
	request osubsubscriptionsdk.ListSubscriptionsRequest,
) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return osubsubscriptionsdk.ListSubscriptionsResponse{}, nil
}

func TestSubscriptionCreateOrUpdateProjectsUniqueListMatch(t *testing.T) {
	t.Parallel()

	fake := &fakeSubscriptionOCIClient{}
	fake.listFn = func(_ context.Context, request osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
		switch len(fake.listRequests) {
		case 1:
			requireStringPtr(t, "compartmentId", request.CompartmentId, "ocid1.compartment.oc1..test")
			requireStringPtr(t, "subscriptionId", request.SubscriptionId, "line-subscription-id")
			requireBoolPtr(t, "isCommitInfoRequired", request.IsCommitInfoRequired, true)
			if request.SortOrder != osubsubscriptionsdk.ListSubscriptionsSortOrderDesc {
				t.Fatalf("sortOrder = %q, want %q", request.SortOrder, osubsubscriptionsdk.ListSubscriptionsSortOrderDesc)
			}
			if request.SortBy != osubsubscriptionsdk.ListSubscriptionsSortByTimecreated {
				t.Fatalf("sortBy = %q, want %q", request.SortBy, osubsubscriptionsdk.ListSubscriptionsSortByTimecreated)
			}
			requireStringPtr(t, "x-one-gateway-subscription-id", request.XOneGatewaySubscriptionId, "gateway-subscription")
			requireStringPtr(t, "x-one-origin-region", request.XOneOriginRegion, "us-phoenix-1")
			if request.Page != nil {
				t.Fatalf("first page request.Page = %q, want nil", *request.Page)
			}
			return osubsubscriptionsdk.ListSubscriptionsResponse{
				Items:       nil,
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			requireStringPtr(t, "second page", request.Page, "page-2")
			requireStringPtr(t, "second page subscriptionId", request.SubscriptionId, "line-subscription-id")
			return osubsubscriptionsdk.ListSubscriptionsResponse{
				Items: []osubsubscriptionsdk.SubscriptionSummary{
					{
						Status:      common.String("ENTITLED"),
						ServiceName: common.String("Oracle Analytics"),
						SubscribedServices: []osubsubscriptionsdk.SubscribedServiceSummary{
							{Id: common.String("line-1"), Status: common.String("ACTIVE")},
						},
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected ListSubscriptions call %d", len(fake.listRequests))
			return osubsubscriptionsdk.ListSubscriptionsResponse{}, nil
		}
	}

	client := newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := &osubsubscriptionv1beta1.Subscription{
		Spec: osubsubscriptionv1beta1.SubscriptionSpec{
			CompartmentId:             "ocid1.compartment.oc1..test",
			SubscriptionId:            "line-subscription-id",
			IsCommitInfoRequired:      true,
			SortOrder:                 "DESC",
			SortBy:                    "TIMECREATED",
			XOneGatewaySubscriptionId: "gateway-subscription",
			XOneOriginRegion:          "us-phoenix-1",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want success without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListSubscriptions() calls = %d, want 2", len(fake.listRequests))
	}
	if got := resource.Status.Status; got != "ENTITLED" {
		t.Fatalf("status.sdkStatus = %q, want ENTITLED", got)
	}
	if got := resource.Status.ServiceName; got != "Oracle Analytics" {
		t.Fatalf("status.serviceName = %q, want Oracle Analytics", got)
	}
	if len(resource.Status.SubscribedServices) != 1 {
		t.Fatalf("status.subscribedServices length = %d, want 1", len(resource.Status.SubscribedServices))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.status.ocid = %q, want empty because no top-level tracked identity is published", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Active)
	}
}

func TestSubscriptionCreateOrUpdateRequeuesWhenSDKStatusBlank(t *testing.T) {
	t.Parallel()

	fake := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
			requireStringPtr(t, "buyerEmail", request.BuyerEmail, "buyer@example.com")
			return osubsubscriptionsdk.ListSubscriptionsResponse{
				Items: []osubsubscriptionsdk.SubscriptionSummary{
					{
						ServiceName: common.String("Oracle Service"),
					},
				},
			}, nil
		},
	}

	client := newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := &osubsubscriptionv1beta1.Subscription{
		Spec: osubsubscriptionv1beta1.SubscriptionSpec{
			CompartmentId: "ocid1.compartment.oc1..test",
			BuyerEmail:    "buyer@example.com",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want success with requeue", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Updating) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Updating)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.status.ocid = %q, want empty", got)
	}
}

func TestSubscriptionCreateOrUpdateRejectsInvalidSelectorCount(t *testing.T) {
	t.Parallel()

	fake := &fakeSubscriptionOCIClient{}
	client := newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := &osubsubscriptionv1beta1.Subscription{
		Spec: osubsubscriptionv1beta1.SubscriptionSpec{
			CompartmentId: "ocid1.compartment.oc1..test",
			PlanNumber:    "plan-1",
			BuyerEmail:    "buyer@example.com",
		},
	}

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "exactly one of spec.planNumber, spec.subscriptionId, or spec.buyerEmail must be set") {
		t.Fatalf("CreateOrUpdate() error = %v, want selector validation error", err)
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListSubscriptions() calls = %d, want 0 when selector validation fails", len(fake.listRequests))
	}
}

func TestSubscriptionCreateOrUpdateFailsOnAmbiguousListMatch(t *testing.T) {
	t.Parallel()

	fake := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
			requireStringPtr(t, "planNumber", request.PlanNumber, "plan-42")
			return osubsubscriptionsdk.ListSubscriptionsResponse{
				Items: []osubsubscriptionsdk.SubscriptionSummary{
					{Status: common.String("ENTITLED")},
					{Status: common.String("ENTITLED")},
				},
			}, nil
		},
	}

	client := newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	resource := &osubsubscriptionv1beta1.Subscription{
		Spec: osubsubscriptionv1beta1.SubscriptionSpec{
			CompartmentId: "ocid1.compartment.oc1..test",
			PlanNumber:    "plan-42",
		},
	}

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match failure", err)
	}
}

func TestSubscriptionDeleteIsKubernetesLocalOnly(t *testing.T) {
	t.Parallel()

	client := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
			t.Fatalf("ListSubscriptions(%#v) should not be called during Kubernetes-local delete", request)
			return osubsubscriptionsdk.ListSubscriptionsResponse{}, nil
		},
	}

	resource := &osubsubscriptionv1beta1.Subscription{
		Spec: osubsubscriptionv1beta1.SubscriptionSpec{
			CompartmentId:  "ocid1.compartment.oc1..test",
			SubscriptionId: "line-subscription-id",
		},
	}
	resource.Status.ServiceName = "Oracle Analytics"

	deleted, err := newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for local finalizer cleanup")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want timestamp after local delete")
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "released from Kubernetes control") {
		t.Fatalf("status.message = %q, want local delete note", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.status.ocid = %q, want empty for local delete", got)
	}
	if got := lastConditionType(t, resource); got != string(shared.Terminating) {
		t.Fatalf("last condition type = %q, want %q", got, shared.Terminating)
	}
}

func requireStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if got := strings.TrimSpace(*value); got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}

func requireBoolPtr(t *testing.T, label string, value *bool, want bool) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %t", label, want)
	}
	if *value != want {
		t.Fatalf("%s = %t, want %t", label, *value, want)
	}
}

func lastConditionType(t *testing.T, resource *osubsubscriptionv1beta1.Subscription) string {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want at least one condition")
	}
	return string(conditions[len(conditions)-1].Type)
}
