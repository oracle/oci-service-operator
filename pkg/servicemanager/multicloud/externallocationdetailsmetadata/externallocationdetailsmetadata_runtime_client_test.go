package externallocationdetailsmetadata

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	multicloudsdk "github.com/oracle/oci-go-sdk/v65/multicloud"
	multicloudv1beta1 "github.com/oracle/oci-service-operator/api/multicloud/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestExternalLocationDetailsMetadataRequiresListSelectorsBeforeOCI(t *testing.T) {
	t.Parallel()

	lister := &fakeExternalLocationDetailsMetadataLister{}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()
	resource.Annotations = nil
	resource.Status.OsokStatus.OpcRequestID = "stale-opc-request"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseCreate,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing annotation error")
	}
	if !strings.Contains(err.Error(), externalLocationDetailsMetadataSubscriptionIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want %s", err, externalLocationDetailsMetadataSubscriptionIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if len(lister.requests) != 0 {
		t.Fatalf("ListExternalLocationDetailsMetadata calls = %d, want 0", len(lister.requests))
	}
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, err.Error(), "")
}

func TestExternalLocationDetailsMetadataPaginatesAndRecordsObservedIdentity(t *testing.T) {
	t.Parallel()

	selected := externalLocationDetailsMetadataSummary("us-ashburn-1", "iad-ad-1", "iad-lad-1", "cpg-selected")
	selected.ExternalLocation.ServiceName = ""
	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{
						externalLocationDetailsMetadataSummary("us-phoenix-1", "phx-ad-1", "phx-lad-1", "cpg-other"),
					},
				},
				OpcRequestId: common.String("opc-page-1"),
				OpcNextPage:  common.String("page-2"),
			},
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{selected},
				},
				OpcRequestId: common.String("opc-page-2"),
			},
		},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()
	resource.Annotations[externalLocationDetailsMetadataCspRegionAnnotation] = "us-ashburn-1"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful response", response)
	}
	if len(lister.requests) != 2 {
		t.Fatalf("ListExternalLocationDetailsMetadata calls = %d, want 2", len(lister.requests))
	}
	assertExternalLocationDetailsMetadataRequest(t, lister.requests[0], "")
	assertExternalLocationDetailsMetadataRequest(t, lister.requests[1], "page-2")
	assertExternalLocationDetailsMetadataObservedStatus(t, resource.Status.OsokStatus, externalLocationDetailsMetadataTestSelector().identity(selected), "opc-page-2")
	if resource.Status.OciRegion != "us-ashburn-1" || resource.Status.OciPhysicalAd != "iad-ad-1" ||
		resource.Status.ClusterPlacementGroupId != "cpg-selected" {
		t.Fatalf("projected ExternalLocationDetailsMetadata status = %+v", resource.Status)
	}
}

func TestExternalLocationDetailsMetadataNoOpReconcileKeepsRecordedIdentity(t *testing.T) {
	t.Parallel()

	selected := externalLocationDetailsMetadataSummary("us-ashburn-1", "iad-ad-1", "iad-lad-1", "cpg-selected")
	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{selected},
				},
				OpcRequestId: common.String("opc-first"),
			},
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{selected},
				},
				OpcRequestId: common.String("opc-second"),
			},
		},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	recordedIdentity := resource.Status.OsokStatus.Ocid
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if resource.Status.OsokStatus.Ocid != recordedIdentity {
		t.Fatalf("status.status.ocid = %q, want unchanged %q", resource.Status.OsokStatus.Ocid, recordedIdentity)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-second" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-second", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestExternalLocationDetailsMetadataRejectsSelectorDrift(t *testing.T) {
	t.Parallel()

	selected := externalLocationDetailsMetadataSummary("us-ashburn-1", "iad-ad-1", "iad-lad-1", "cpg-selected")
	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{selected},
				},
				OpcRequestId: common.String("opc-list"),
			},
		},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()
	resource.Status.OsokStatus.Ocid = "externallocationdetailsmetadata:old"

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want selector drift error")
	}
	if !strings.Contains(err.Error(), "replacement is required") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required message", err)
	}
	if got := resource.Status.OsokStatus.Ocid; got != "externallocationdetailsmetadata:old" {
		t.Fatalf("status.status.ocid = %q, want existing identity preserved", got)
	}
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, err.Error(), "opc-list")
}

func TestExternalLocationDetailsMetadataRejectsCompartmentSelectorDrift(t *testing.T) {
	t.Parallel()

	runExternalLocationDetailsMetadataUnreturnedSelectorDriftTest(
		t,
		externalLocationDetailsMetadataCompartmentIDAnnotation,
		func(resource *multicloudv1beta1.ExternalLocationDetailsMetadata) {
			resource.Annotations[externalLocationDetailsMetadataCompartmentIDAnnotation] = "ocid1.compartment.oc1..replacement"
		},
		func(t *testing.T, request multicloudsdk.ListExternalLocationDetailsMetadataRequest) {
			t.Helper()
			if got := stringValue(request.CompartmentId); got != "ocid1.compartment.oc1..replacement" {
				t.Fatalf("request.CompartmentId = %q, want replacement compartment", got)
			}
		},
	)
}

func TestExternalLocationDetailsMetadataRejectsLinkedCompartmentSelectorDrift(t *testing.T) {
	t.Parallel()

	runExternalLocationDetailsMetadataUnreturnedSelectorDriftTest(
		t,
		externalLocationDetailsMetadataLinkedCompartmentIDAnnotation,
		func(resource *multicloudv1beta1.ExternalLocationDetailsMetadata) {
			resource.Annotations[externalLocationDetailsMetadataLinkedCompartmentIDAnnotation] = "ocid1.compartment.oc1..linked-replacement"
		},
		func(t *testing.T, request multicloudsdk.ListExternalLocationDetailsMetadataRequest) {
			t.Helper()
			if got := stringValue(request.LinkedCompartmentId); got != "ocid1.compartment.oc1..linked-replacement" {
				t.Fatalf("request.LinkedCompartmentId = %q, want replacement linked compartment", got)
			}
		},
	)
}

func runExternalLocationDetailsMetadataUnreturnedSelectorDriftTest(
	t *testing.T,
	annotation string,
	mutate func(*multicloudv1beta1.ExternalLocationDetailsMetadata),
	assertRequest func(*testing.T, multicloudsdk.ListExternalLocationDetailsMetadataRequest),
) {
	t.Helper()

	selected := externalLocationDetailsMetadataSummary("us-ashburn-1", "iad-ad-1", "iad-lad-1", "cpg-selected")
	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{selected},
				},
				OpcRequestId: common.String("opc-initial"),
			},
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{selected},
				},
				OpcRequestId: common.String("opc-drift"),
			},
		},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()
	resource.Annotations[externalLocationDetailsMetadataLinkedCompartmentIDAnnotation] = "ocid1.compartment.oc1..linked"

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("initial CreateOrUpdate() error = %v", err)
	}
	recordedIdentity := resource.Status.OsokStatus.Ocid
	mutate(resource)

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatalf("CreateOrUpdate() error = nil after changing %s, want selector drift error", annotation)
	}
	if !strings.Contains(err.Error(), "replacement is required") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required message", err)
	}
	if got := resource.Status.OsokStatus.Ocid; got != recordedIdentity {
		t.Fatalf("status.status.ocid = %q, want recorded identity preserved %q", got, recordedIdentity)
	}
	if len(lister.requests) != 2 {
		t.Fatalf("ListExternalLocationDetailsMetadata calls = %d, want 2", len(lister.requests))
	}
	assertRequest(t, lister.requests[1])
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, err.Error(), "opc-drift")
}

func TestExternalLocationDetailsMetadataRejectsAmbiguousListSelection(t *testing.T) {
	t.Parallel()

	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{
				ExternalLocationsMetadatumCollection: multicloudsdk.ExternalLocationsMetadatumCollection{
					Items: []multicloudsdk.ExternalLocationsMetadatumSummary{
						externalLocationDetailsMetadataSummary("us-ashburn-1", "iad-ad-1", "iad-lad-1", "cpg-1"),
						externalLocationDetailsMetadataSummary("us-phoenix-1", "phx-ad-1", "phx-lad-1", "cpg-2"),
					},
				},
				OpcRequestId: common.String("opc-list"),
			},
		},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous selection error")
	}
	if !strings.Contains(err.Error(), "matched 2 items") {
		t.Fatalf("CreateOrUpdate() error = %v, want ambiguous match count", err)
	}
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, err.Error(), "opc-list")
}

func TestExternalLocationDetailsMetadataRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	listErr := errortest.NewServiceError(500, "InternalError", "list failed")
	listErr.OpcRequestID = "opc-list-error"
	lister := &fakeExternalLocationDetailsMetadataLister{err: listErr}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, "list failed", "opc-list-error")
}

func TestExternalLocationDetailsMetadataRecordsLaterPageOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	pageErr := errortest.NewServiceError(500, "InternalError", "page 2 failed")
	pageErr.OpcRequestID = "opc-page-2-error"
	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{
				OpcRequestId: common.String("opc-page-1"),
				OpcNextPage:  common.String("page-2"),
			},
			{},
		},
		errors: []error{nil, pageErr},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want later page OCI error")
	}
	if len(lister.requests) != 2 {
		t.Fatalf("ListExternalLocationDetailsMetadata calls = %d, want 2", len(lister.requests))
	}
	assertExternalLocationDetailsMetadataRequest(t, lister.requests[0], "")
	assertExternalLocationDetailsMetadataRequest(t, lister.requests[1], "page-2")
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, "page 2 failed", "opc-page-2-error")
}

func TestExternalLocationDetailsMetadataRejectsRepeatedPaginationToken(t *testing.T) {
	t.Parallel()

	lister := &fakeExternalLocationDetailsMetadataLister{
		responses: []multicloudsdk.ListExternalLocationDetailsMetadataResponse{
			{OpcRequestId: common.String("opc-page-1"), OpcNextPage: common.String("same-page")},
			{OpcRequestId: common.String("opc-page-2"), OpcNextPage: common.String("same-page")},
		},
	}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want repeated pagination token error")
	}
	if !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("CreateOrUpdate() error = %v, want repeated page token", err)
	}
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, err.Error(), "opc-page-2")
}

func TestExternalLocationDetailsMetadataDeleteConfirmsWithoutOCI(t *testing.T) {
	t.Parallel()

	lister := &fakeExternalLocationDetailsMetadataLister{}
	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, lister)
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()
	resource.Status.OsokStatus.Ocid = "externallocationdetailsmetadata:recorded"
	resource.Status.OsokStatus.OpcRequestID = "stale-opc-request"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for list-only metadata surface")
	}
	if len(lister.requests) != 0 {
		t.Fatalf("ListExternalLocationDetailsMetadata calls = %d, want 0", len(lister.requests))
	}
	assertExternalLocationDetailsMetadataDeletedStatus(t, resource.Status.OsokStatus)
}

func TestExternalLocationDetailsMetadataPreservesGeneratedOCIInitErrorWhenWrapped(t *testing.T) {
	t.Parallel()

	initErr := errors.New("initialize ExternalLocationDetailsMetadata OCI client: provider invalid")
	lister := &fakeExternalLocationDetailsMetadataLister{}
	client := newExternalLocationDetailsMetadataRuntimeTestClientWithDelegate(t, lister, defaultExternalLocationDetailsMetadataServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*multicloudv1beta1.ExternalLocationDetailsMetadata](
			generatedruntime.Config[*multicloudv1beta1.ExternalLocationDetailsMetadata]{
				Kind:      "ExternalLocationDetailsMetadata",
				InitError: initErr,
			},
		),
	})
	resource := newExternalLocationDetailsMetadataRuntimeTestResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !errors.Is(err, initErr) {
		t.Fatalf("CreateOrUpdate() error = %v, want generated init error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	assertExternalLocationDetailsMetadataFailedStatus(t, resource.Status.OsokStatus, initErr.Error(), "")

	deleted, err := client.Delete(context.Background(), resource)
	if !errors.Is(err, initErr) {
		t.Fatalf("Delete() error = %v, want generated init error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(lister.requests) != 0 {
		t.Fatalf("ListExternalLocationDetailsMetadata calls = %d, want 0", len(lister.requests))
	}
}

func TestExternalLocationDetailsMetadataRuntimeRejectsNilResource(t *testing.T) {
	t.Parallel()

	client := newExternalLocationDetailsMetadataRuntimeTestClient(t, &fakeExternalLocationDetailsMetadataLister{})
	if _, err := client.CreateOrUpdate(context.Background(), nil, ctrl.Request{}); err == nil {
		t.Fatal("CreateOrUpdate(nil) error = nil, want error")
	}
	if deleted, err := client.Delete(context.Background(), nil); err == nil || deleted {
		t.Fatalf("Delete(nil) = (%v, %v), want false with error", deleted, err)
	}
}

func newExternalLocationDetailsMetadataRuntimeTestClient(
	t *testing.T,
	lister *fakeExternalLocationDetailsMetadataLister,
) ExternalLocationDetailsMetadataServiceClient {
	t.Helper()

	return newExternalLocationDetailsMetadataRuntimeTestClientWithDelegate(t, lister, recordingExternalLocationDetailsMetadataDelegate{})
}

func newExternalLocationDetailsMetadataRuntimeTestClientWithDelegate(
	t *testing.T,
	lister *fakeExternalLocationDetailsMetadataLister,
	delegate ExternalLocationDetailsMetadataServiceClient,
) ExternalLocationDetailsMetadataServiceClient {
	t.Helper()

	hooks := newExternalLocationDetailsMetadataDefaultRuntimeHooks(multicloudsdk.MetadataClient{})
	hooks.List.Call = lister.list
	applyExternalLocationDetailsMetadataRuntimeHooks(nil, &hooks)
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}
	return hooks.WrapGeneratedClient[0](delegate)
}

func newExternalLocationDetailsMetadataRuntimeTestResource() *multicloudv1beta1.ExternalLocationDetailsMetadata {
	return &multicloudv1beta1.ExternalLocationDetailsMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-location-details-metadata",
			Namespace: "default",
			Annotations: map[string]string{
				externalLocationDetailsMetadataSubscriptionIDAnnotation:          "ocid1.multicloudsubscription.oc1..subscription",
				externalLocationDetailsMetadataSubscriptionServiceNameAnnotation: string(multicloudsdk.ListExternalLocationDetailsMetadataSubscriptionServiceNameOracledbatazure),
				externalLocationDetailsMetadataEntityTypeAnnotation:              string(multicloudsdk.ListExternalLocationDetailsMetadataEntityTypeDbsystem),
				externalLocationDetailsMetadataCompartmentIDAnnotation:           "ocid1.compartment.oc1..base",
			},
		},
	}
}

func externalLocationDetailsMetadataTestSelector() externalLocationDetailsMetadataSelector {
	return externalLocationDetailsMetadataSelector{
		subscriptionID:          "ocid1.multicloudsubscription.oc1..subscription",
		subscriptionServiceName: string(multicloudsdk.ListExternalLocationDetailsMetadataSubscriptionServiceNameOracledbatazure),
		entityType:              string(multicloudsdk.ListExternalLocationDetailsMetadataEntityTypeDbsystem),
		compartmentID:           "ocid1.compartment.oc1..base",
		cspRegion:               "us-ashburn-1",
	}
}

func externalLocationDetailsMetadataSummary(cspRegion string, physicalAD string, logicalAD string, cpgID string) multicloudsdk.ExternalLocationsMetadatumSummary {
	return multicloudsdk.ExternalLocationsMetadatumSummary{
		FreeformTags:            map[string]string{},
		DefinedTags:             map[string]map[string]interface{}{},
		ExternalLocation:        externalLocationDetailsMetadataLocation(cspRegion, physicalAD, logicalAD),
		OciPhysicalAd:           common.String(physicalAD),
		OciRegion:               common.String("us-ashburn-1"),
		OciLogicalAd:            common.String(logicalAD),
		CpgId:                   common.String(cpgID + "-legacy"),
		ClusterPlacementGroupId: common.String(cpgID),
		PartnerCloudName:        common.String("azure"),
		PartnerCloudAccountName: common.String("account"),
		PartnerCloudAccountUrl:  common.String("https://example.invalid/account"),
		SystemTags:              map[string]map[string]interface{}{},
	}
}

func externalLocationDetailsMetadataLocation(cspRegion string, physicalAD string, logicalAD string) *multicloudsdk.ExternalLocationDetail {
	return &multicloudsdk.ExternalLocationDetail{
		CspRegion:                common.String(cspRegion),
		CspRegionDisplayName:     common.String(cspRegion),
		CspPhysicalAz:            common.String(physicalAD),
		CspPhysicalAzDisplayName: common.String(physicalAD),
		CspLogicalAz:             common.String(logicalAD),
		CspLogicalAzDisplayName:  common.String(logicalAD),
		ServiceName:              multicloudsdk.SubscriptionTypeOracledbatazure,
	}
}

func assertExternalLocationDetailsMetadataRequest(
	t *testing.T,
	request multicloudsdk.ListExternalLocationDetailsMetadataRequest,
	wantPage string,
) {
	t.Helper()

	if got := stringValue(request.SubscriptionId); got != "ocid1.multicloudsubscription.oc1..subscription" {
		t.Fatalf("request.SubscriptionId = %q, want subscription OCID", got)
	}
	if request.SubscriptionServiceName != multicloudsdk.ListExternalLocationDetailsMetadataSubscriptionServiceNameOracledbatazure {
		t.Fatalf("request.SubscriptionServiceName = %q, want ORACLEDBATAZURE", request.SubscriptionServiceName)
	}
	if request.EntityType != multicloudsdk.ListExternalLocationDetailsMetadataEntityTypeDbsystem {
		t.Fatalf("request.EntityType = %q, want dbsystem", request.EntityType)
	}
	if got := stringValue(request.CompartmentId); got != "ocid1.compartment.oc1..base" {
		t.Fatalf("request.CompartmentId = %q, want base compartment", got)
	}
	if got := stringValue(request.Page); got != wantPage {
		t.Fatalf("request.Page = %q, want %q", got, wantPage)
	}
}

func lastExternalLocationDetailsMetadataCondition(t *testing.T, status shared.OSOKStatus) shared.OSOKCondition {
	t.Helper()
	if len(status.Conditions) == 0 {
		t.Fatal("status.status.conditions is empty")
	}
	return status.Conditions[len(status.Conditions)-1]
}

func assertExternalLocationDetailsMetadataObservedStatus(t *testing.T, status shared.OSOKStatus, wantIdentity string, wantRequestID string) {
	t.Helper()

	if got := string(status.Ocid); got != wantIdentity {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantIdentity)
	}
	if status.Reason != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want Active", status.Reason)
	}
	if status.Message != externalLocationDetailsMetadataObservedMessage {
		t.Fatalf("status.status.message = %q, want observed message", status.Message)
	}
	if status.CreatedAt == nil || status.UpdatedAt == nil {
		t.Fatalf("createdAt=%v updatedAt=%v, want observation timestamps", status.CreatedAt, status.UpdatedAt)
	}
	if status.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want cleared async state", status.Async.Current)
	}
	if status.OpcRequestID != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", status.OpcRequestID, wantRequestID)
	}
	condition := lastExternalLocationDetailsMetadataCondition(t, status)
	if condition.Type != shared.Active || condition.Status != v1.ConditionTrue {
		t.Fatalf("last condition = %#v, want Active=True", condition)
	}
}

func assertExternalLocationDetailsMetadataFailedStatus(t *testing.T, status shared.OSOKStatus, wantMessage string, wantRequestID string) {
	t.Helper()

	if status.Reason != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", status.Reason)
	}
	if status.Message != wantMessage {
		t.Fatalf("status.status.message = %q, want %q", status.Message, wantMessage)
	}
	if status.UpdatedAt == nil {
		t.Fatal("status.status.updatedAt = nil, want failed status timestamp")
	}
	condition := lastExternalLocationDetailsMetadataCondition(t, status)
	if condition.Type != shared.Failed || condition.Status != v1.ConditionFalse {
		t.Fatalf("last condition = %#v, want Failed=False", condition)
	}
	if status.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want cleared async state", status.Async.Current)
	}
	if status.OpcRequestID != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", status.OpcRequestID, wantRequestID)
	}
}

func assertExternalLocationDetailsMetadataDeletedStatus(t *testing.T, status shared.OSOKStatus) {
	t.Helper()

	if status.Ocid != "" {
		t.Fatalf("status.status.ocid = %q, want cleared list-only identity", status.Ocid)
	}
	if status.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want cleared async state", status.Async.Current)
	}
	if status.OpcRequestID != "" {
		t.Fatalf("status.status.opcRequestId = %q, want cleared request id without OCI delete call", status.OpcRequestID)
	}
	if status.DeletedAt == nil || status.UpdatedAt == nil {
		t.Fatalf("deletedAt=%v updatedAt=%v, want delete timestamps", status.DeletedAt, status.UpdatedAt)
	}
	if status.Message != externalLocationDetailsMetadataDeletedMessage {
		t.Fatalf("status.status.message = %q, want list-only delete message", status.Message)
	}
	condition := lastExternalLocationDetailsMetadataCondition(t, status)
	if condition.Type != shared.Terminating || condition.Status != v1.ConditionTrue {
		t.Fatalf("last condition = %#v, want Terminating=True", condition)
	}
}

type fakeExternalLocationDetailsMetadataLister struct {
	responses []multicloudsdk.ListExternalLocationDetailsMetadataResponse
	errors    []error
	err       error
	requests  []multicloudsdk.ListExternalLocationDetailsMetadataRequest
}

func (l *fakeExternalLocationDetailsMetadataLister) list(
	_ context.Context,
	request multicloudsdk.ListExternalLocationDetailsMetadataRequest,
) (multicloudsdk.ListExternalLocationDetailsMetadataResponse, error) {
	l.requests = append(l.requests, request)
	if l.err != nil {
		return multicloudsdk.ListExternalLocationDetailsMetadataResponse{}, l.err
	}
	index := len(l.requests) - 1
	if index < len(l.errors) && l.errors[index] != nil {
		if index < len(l.responses) {
			return l.responses[index], l.errors[index]
		}
		return multicloudsdk.ListExternalLocationDetailsMetadataResponse{}, l.errors[index]
	}
	if index >= len(l.responses) {
		return multicloudsdk.ListExternalLocationDetailsMetadataResponse{}, nil
	}
	return l.responses[index], nil
}

type recordingExternalLocationDetailsMetadataDelegate struct{}

func (recordingExternalLocationDetailsMetadataDelegate) CreateOrUpdate(
	context.Context,
	*multicloudv1beta1.ExternalLocationDetailsMetadata,
	ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func (recordingExternalLocationDetailsMetadataDelegate) Delete(
	context.Context,
	*multicloudv1beta1.ExternalLocationDetailsMetadata,
) (bool, error) {
	return true, nil
}
