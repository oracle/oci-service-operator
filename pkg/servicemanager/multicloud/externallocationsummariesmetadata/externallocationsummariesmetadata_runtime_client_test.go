/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package externallocationsummariesmetadata

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	multicloudsdk "github.com/oracle/oci-go-sdk/v65/multicloud"
	multicloudv1beta1 "github.com/oracle/oci-service-operator/api/multicloud/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testExternalLocationSummariesMetadataCompartmentID  = "ocid1.compartment.oc1..multicloud"
	testExternalLocationSummariesMetadataSubscriptionID = "ocid1.multicloudsubscription.oc1..subscription"
)

func TestExternalLocationSummariesMetadataCreateOrUpdateListsAllPagesAndProjectsActive(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	resource.Annotations[externalLocationSummariesMetadataSubscriptionIDAnnotation] = testExternalLocationSummariesMetadataSubscriptionID
	resource.Annotations[externalLocationSummariesMetadataEntityTypeAnnotation] = "dbsystem"
	resource.Annotations[externalLocationSummariesMetadataLimitAnnotation] = "10"
	resource.Annotations[externalLocationSummariesMetadataSortOrderAnnotation] = "DESC"
	resource.Annotations[externalLocationSummariesMetadataSortByAnnotation] = "timeCreated"

	fake := &fakeExternalLocationSummariesMetadataOCIClient{}
	fake.list = func(_ context.Context, request multicloudsdk.ListExternalLocationSummariesMetadataRequest) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error) {
		if fake.listCalls == 1 {
			assertExternalLocationSummariesMetadataListRequest(t, request, "")
			return multicloudsdk.ListExternalLocationSummariesMetadataResponse{
				ExternalLocationSummariesMetadatumSummaryCollection: multicloudsdk.ExternalLocationSummariesMetadatumSummaryCollection{
					Items: []multicloudsdk.ExternalLocationSummariesMetadatumSummary{
						{OciRegion: common.String("us-ashburn-1")},
					},
				},
				OpcRequestId: common.String("opc-page-1"),
				OpcNextPage:  common.String("page-2"),
			}, nil
		}
		assertExternalLocationSummariesMetadataListRequest(t, request, "page-2")
		return multicloudsdk.ListExternalLocationSummariesMetadataResponse{
			ExternalLocationSummariesMetadatumSummaryCollection: multicloudsdk.ExternalLocationSummariesMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationSummariesMetadatumSummary{
					{OciRegion: common.String("us-phoenix-1")},
				},
			},
			OpcRequestId: common.String("opc-page-2"),
		}, nil
	}

	response, err := newTestExternalLocationSummariesMetadataClient(fake).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observation", response)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListExternalLocationSummariesMetadata calls = %d, want 2", fake.listCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-page-2" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-page-2", got)
	}
	if got := resource.Status.OsokStatus.Message; got != "observed 2 ExternalLocationSummariesMetadata item(s)" {
		t.Fatalf("status.status.message = %q, want observed count", got)
	}
	if len(resource.Status.Items) != 2 || resource.Status.Items[0].OciRegion != "us-ashburn-1" ||
		resource.Status.Items[1].OciRegion != "us-phoenix-1" {
		t.Fatalf("projected ExternalLocationSummariesMetadata items = %+v", resource.Status.Items)
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Active)
}

func TestExternalLocationSummariesMetadataCreateOrUpdateRequiresAnnotationFiltersBeforeOCI(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	delete(resource.Annotations, externalLocationSummariesMetadataCompartmentIDAnnotation)
	fake := &fakeExternalLocationSummariesMetadataOCIClient{}

	response, err := newTestExternalLocationSummariesMetadataClient(fake).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing compartment annotation rejection")
	}
	if !strings.Contains(err.Error(), externalLocationSummariesMetadataCompartmentIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %q, want missing compartment annotation context", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if fake.listCalls != 0 {
		t.Fatalf("ListExternalLocationSummariesMetadata calls = %d, want 0", fake.listCalls)
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Failed)
}

func TestExternalLocationSummariesMetadataCreateOrUpdateRejectsUnsupportedAnnotationEnums(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	resource.Annotations[externalLocationSummariesMetadataSubscriptionServiceNameAnnotation] = "ORACLEDBATMOON"
	fake := &fakeExternalLocationSummariesMetadataOCIClient{}

	_, err := newTestExternalLocationSummariesMetadataClient(fake).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported subscription service name rejection")
	}
	if !strings.Contains(err.Error(), "unsupported value") {
		t.Fatalf("CreateOrUpdate() error = %q, want unsupported value context", err.Error())
	}
	if fake.listCalls != 0 {
		t.Fatalf("ListExternalLocationSummariesMetadata calls = %d, want 0", fake.listCalls)
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Failed)
}

func TestExternalLocationSummariesMetadataCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	fake := &fakeExternalLocationSummariesMetadataOCIClient{}
	fake.list = func(context.Context, multicloudsdk.ListExternalLocationSummariesMetadataRequest) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error) {
		return multicloudsdk.ListExternalLocationSummariesMetadataResponse{}, fakeExternalLocationSummariesMetadataServiceError{
			message:      "list failed",
			opcRequestID: "opc-list-error",
		}
	}

	_, err := newTestExternalLocationSummariesMetadataClient(fake).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-list-error" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-list-error", got)
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Failed)
}

func TestExternalLocationSummariesMetadataCreateOrUpdateRejectsRepeatedListPage(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	fake := &fakeExternalLocationSummariesMetadataOCIClient{}
	fake.list = func(_ context.Context, request multicloudsdk.ListExternalLocationSummariesMetadataRequest) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error) {
		if fake.listCalls == 1 {
			return multicloudsdk.ListExternalLocationSummariesMetadataResponse{OpcNextPage: common.String("page-2")}, nil
		}
		if got := stringPtrValue(request.Page); got != "page-2" {
			t.Fatalf("ListExternalLocationSummariesMetadata page = %q, want page-2", got)
		}
		return multicloudsdk.ListExternalLocationSummariesMetadataResponse{OpcNextPage: common.String("page-2")}, nil
	}

	_, err := newTestExternalLocationSummariesMetadataClient(fake).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want repeated page token rejection")
	}
	if !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("CreateOrUpdate() error = %q, want repeated page token context", err.Error())
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Failed)
}

func TestExternalLocationSummariesMetadataCreateOrUpdateIsReadOnlyNoOpReconcile(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	fake := &fakeExternalLocationSummariesMetadataOCIClient{}
	fake.list = func(context.Context, multicloudsdk.ListExternalLocationSummariesMetadataRequest) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error) {
		return multicloudsdk.ListExternalLocationSummariesMetadataResponse{
			ExternalLocationSummariesMetadatumSummaryCollection: multicloudsdk.ExternalLocationSummariesMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationSummariesMetadatumSummary{
					{OciRegion: common.String("us-ashburn-1")},
				},
			},
			OpcRequestId: common.String("opc-list"),
		}, nil
	}
	client := newTestExternalLocationSummariesMetadataClient(fake)

	for i := 0; i < 2; i++ {
		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err != nil {
			t.Fatalf("CreateOrUpdate() error = %v", err)
		}
		if !response.IsSuccessful || response.ShouldRequeue {
			t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observation", response)
		}
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListExternalLocationSummariesMetadata calls = %d, want one read per reconcile", fake.listCalls)
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Active)
}

func TestExternalLocationSummariesMetadataDeleteIsConfirmedNoOp(t *testing.T) {
	t.Parallel()

	resource := newExternalLocationSummariesMetadataResource()
	fake := &fakeExternalLocationSummariesMetadataOCIClient{}

	deleted, err := newTestExternalLocationSummariesMetadataClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for read-only metadata no-op delete")
	}
	if fake.listCalls != 0 {
		t.Fatalf("ListExternalLocationSummariesMetadata calls = %d, want 0", fake.listCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	assertExternalLocationSummariesMetadataCondition(t, resource, shared.Terminating)
}

type fakeExternalLocationSummariesMetadataOCIClient struct {
	list      externalLocationSummariesMetadataListCall
	listCalls int
	requests  []multicloudsdk.ListExternalLocationSummariesMetadataRequest
}

func (f *fakeExternalLocationSummariesMetadataOCIClient) ListExternalLocationSummariesMetadata(
	ctx context.Context,
	request multicloudsdk.ListExternalLocationSummariesMetadataRequest,
) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error) {
	f.listCalls++
	f.requests = append(f.requests, request)
	if f.list != nil {
		return f.list(ctx, request)
	}
	return multicloudsdk.ListExternalLocationSummariesMetadataResponse{}, nil
}

type fakeExternalLocationSummariesMetadataServiceError struct {
	message      string
	opcRequestID string
}

func (e fakeExternalLocationSummariesMetadataServiceError) Error() string {
	return e.message
}

func (e fakeExternalLocationSummariesMetadataServiceError) GetOpcRequestID() string {
	return e.opcRequestID
}

func newTestExternalLocationSummariesMetadataClient(
	fake *fakeExternalLocationSummariesMetadataOCIClient,
) ExternalLocationSummariesMetadataServiceClient {
	return newExternalLocationSummariesMetadataServiceClientWithListCall(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake.ListExternalLocationSummariesMetadata,
		nil,
	)
}

func newExternalLocationSummariesMetadataResource() *multicloudv1beta1.ExternalLocationSummariesMetadata {
	return &multicloudv1beta1.ExternalLocationSummariesMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-location-summaries-metadata",
			Namespace: "default",
			Annotations: map[string]string{
				externalLocationSummariesMetadataSubscriptionServiceNameAnnotation: "ORACLEDBATAZURE",
				externalLocationSummariesMetadataCompartmentIDAnnotation:           testExternalLocationSummariesMetadataCompartmentID,
			},
		},
	}
}

func assertExternalLocationSummariesMetadataListRequest(
	t *testing.T,
	request multicloudsdk.ListExternalLocationSummariesMetadataRequest,
	wantPage string,
) {
	t.Helper()

	if got := request.SubscriptionServiceName; got != multicloudsdk.ListExternalLocationSummariesMetadataSubscriptionServiceNameOracledbatazure {
		t.Fatalf("SubscriptionServiceName = %q, want ORACLEDBATAZURE", got)
	}
	if got := stringPtrValue(request.CompartmentId); got != testExternalLocationSummariesMetadataCompartmentID {
		t.Fatalf("CompartmentId = %q, want %q", got, testExternalLocationSummariesMetadataCompartmentID)
	}
	if got := stringPtrValue(request.SubscriptionId); got != testExternalLocationSummariesMetadataSubscriptionID {
		t.Fatalf("SubscriptionId = %q, want %q", got, testExternalLocationSummariesMetadataSubscriptionID)
	}
	if got := request.EntityType; got != multicloudsdk.ListExternalLocationSummariesMetadataEntityTypeDbsystem {
		t.Fatalf("EntityType = %q, want dbsystem", got)
	}
	if request.Limit == nil || *request.Limit != 10 {
		t.Fatalf("Limit = %#v, want 10", request.Limit)
	}
	if got := request.SortOrder; got != multicloudsdk.ListExternalLocationSummariesMetadataSortOrderDesc {
		t.Fatalf("SortOrder = %q, want DESC", got)
	}
	if got := request.SortBy; got != multicloudsdk.ListExternalLocationSummariesMetadataSortByTimecreated {
		t.Fatalf("SortBy = %q, want timeCreated", got)
	}
	if got := stringPtrValue(request.Page); got != wantPage {
		t.Fatalf("Page = %q, want %q", got, wantPage)
	}
}

func assertExternalLocationSummariesMetadataCondition(
	t *testing.T,
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
	want shared.OSOKConditionType,
) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s condition", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q", got, want)
	}
}
