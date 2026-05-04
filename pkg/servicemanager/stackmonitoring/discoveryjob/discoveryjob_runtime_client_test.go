/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package discoveryjob

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDiscoveryJobID            = "ocid1.stackmonitoringdiscoveryjob.oc1..job"
	testDiscoveryJobCompartmentID = "ocid1.compartment.oc1..discovery"
	testDiscoveryJobResourceName  = "database-discovery"
	testDiscoveryJobAgentID       = "ocid1.managementagent.oc1..agent"
)

type fakeDiscoveryJobOCIClient struct {
	createFn func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error)
	getFn    func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error)
	listFn   func(context.Context, stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error)
	deleteFn func(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	deleteCalls int
}

func (f *fakeDiscoveryJobOCIClient) CreateDiscoveryJob(
	ctx context.Context,
	request stackmonitoringsdk.CreateDiscoveryJobRequest,
) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
}

func (f *fakeDiscoveryJobOCIClient) GetDiscoveryJob(
	ctx context.Context,
	request stackmonitoringsdk.GetDiscoveryJobRequest,
) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return stackmonitoringsdk.GetDiscoveryJobResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "DiscoveryJob is missing")
}

func (f *fakeDiscoveryJobOCIClient) ListDiscoveryJobs(
	ctx context.Context,
	request stackmonitoringsdk.ListDiscoveryJobsRequest,
) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return stackmonitoringsdk.ListDiscoveryJobsResponse{}, nil
}

func (f *fakeDiscoveryJobOCIClient) DeleteDiscoveryJob(
	ctx context.Context,
	request stackmonitoringsdk.DeleteDiscoveryJobRequest,
) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, nil
}

func TestDiscoveryJobRuntimeHooksConfigured(t *testing.T) {
	hooks := newDiscoveryJobDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyDiscoveryJobRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
		{name: "Read.Get", ok: hooks.Read.Get != nil},
		{name: "Read.List", ok: hooks.Read.List != nil},
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

	resource := makeDiscoveryJobResource()
	body, err := hooks.BuildCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(stackmonitoringsdk.CreateDiscoveryJobDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateDiscoveryJobDetails", body)
	}
	requireStringPtr(t, "CreateDiscoveryJobDetails.CompartmentId", details.CompartmentId, testDiscoveryJobCompartmentID)
	if details.ShouldPropagateTagsToDiscoveredResources == nil || !*details.ShouldPropagateTagsToDiscoveredResources {
		t.Fatalf("CreateDiscoveryJobDetails.ShouldPropagateTagsToDiscoveredResources = %#v, want true", details.ShouldPropagateTagsToDiscoveredResources)
	}
	requireDiscoveryJobDetails(t, details.DiscoveryDetails, resource.Spec.DiscoveryDetails)
}

func TestDiscoveryJobBuildCreateBodyPreservesFalsePropagationFlag(t *testing.T) {
	hooks := newDiscoveryJobDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyDiscoveryJobRuntimeHooks(&hooks)
	resource := makeDiscoveryJobResource()
	resource.Spec.ShouldPropagateTagsToDiscoveredResources = false

	body, err := hooks.BuildCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(stackmonitoringsdk.CreateDiscoveryJobDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateDiscoveryJobDetails", body)
	}
	if details.ShouldPropagateTagsToDiscoveredResources == nil {
		t.Fatal("CreateDiscoveryJobDetails.ShouldPropagateTagsToDiscoveredResources = nil, want false")
	}
	if *details.ShouldPropagateTagsToDiscoveredResources {
		t.Fatal("CreateDiscoveryJobDetails.ShouldPropagateTagsToDiscoveredResources = true, want false")
	}
}

func TestDiscoveryJobCreateRecordsIdentityRequestIDLifecycleAndScrubsCredentials(t *testing.T) {
	resource := makeDiscoveryJobResource()
	created := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateCreating, stackmonitoringsdk.DiscoveryJobStatusCreated)
	client := &fakeDiscoveryJobOCIClient{
		createFn: func(_ context.Context, request stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			requireDiscoveryJobCreateRequest(t, request, resource)
			return stackmonitoringsdk.CreateDiscoveryJobResponse{
				DiscoveryJob: created,
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: created}, nil
		},
	}

	response, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true for CREATED DiscoveryJob")
	}
	assertDiscoveryJobCallCount(t, "ListDiscoveryJobs()", client.listCalls, 1)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 1)
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 1)
	assertDiscoveryJobRecordedID(t, resource, testDiscoveryJobID)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-create")
	assertDiscoveryJobCreateOnlyStateRecorded(t, resource)
	assertDiscoveryJobStatusScrubsCredentialMarkers(t, resource)
	if got := resource.Status.DiscoveryDetails.Credentials.Items; len(got) != 0 {
		t.Fatalf("status.discoveryDetails.credentials.items = %#v, want scrubbed", got)
	}
	if got := resource.Status.ResourceName; got != testDiscoveryJobResourceName {
		t.Fatalf("status.resourceName = %q, want %q", got, testDiscoveryJobResourceName)
	}
}

func TestDiscoveryJobCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeDiscoveryJobResource()
	existing := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	var pages []string
	client := &fakeDiscoveryJobOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListDiscoveryJobsRequest.CompartmentId", request.CompartmentId, testDiscoveryJobCompartmentID)
			requireStringPtr(t, "ListDiscoveryJobsRequest.Name", request.Name, testDiscoveryJobResourceName)
			if request.Page == nil {
				return stackmonitoringsdk.ListDiscoveryJobsResponse{
					DiscoveryJobCollection: stackmonitoringsdk.DiscoveryJobCollection{
						Items: []stackmonitoringsdk.DiscoveryJobSummary{
							sdkDiscoveryJobSummary(resource, "ocid1.stackmonitoringdiscoveryjob.oc1..other", "other-resource", stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobSummaryStatusSuccess),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return stackmonitoringsdk.ListDiscoveryJobsResponse{
				DiscoveryJobCollection: stackmonitoringsdk.DiscoveryJobCollection{
					Items: []stackmonitoringsdk.DiscoveryJobSummary{
						sdkDiscoveryJobSummary(resource, testDiscoveryJobID, testDiscoveryJobResourceName, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobSummaryStatusSuccess),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: existing}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite existing list match")
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
		},
	}

	response, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListDiscoveryJobs() pages = %q, want \",page-2\"", got)
	}
	assertDiscoveryJobRecordedID(t, resource, testDiscoveryJobID)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
}

func TestDiscoveryJobCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	current := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: current}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called during no-op reconcile")
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
		},
	}

	response, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 1)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestDiscoveryJobNoUpdateDriftRejectedBeforeMutatingOCI(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	currentResource := makeDiscoveryJobResource()
	currentResource.Spec.DiscoveryDetails.ResourceName = "renamed-outside-osok"
	current := sdkDiscoveryJob(currentResource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: current}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite existing DiscoveryJob drift")
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called during no-update drift handling")
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want no-update drift rejection")
	}
	if !strings.Contains(err.Error(), "discoveryDetails.resourceName") {
		t.Fatalf("CreateOrUpdate() error = %v, want resourceName drift detail", err)
	}
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
}

func TestDiscoveryJobCreateOrUpdateTrackedIdentityDriftRejectedBeforeOCI(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	resource.Status.Id = testDiscoveryJobID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.ResourceName = resource.Spec.DiscoveryDetails.ResourceName
	resource.Status.ResourceType = resource.Spec.DiscoveryDetails.ResourceType
	resource.Status.DiscoveryType = resource.Spec.DiscoveryType
	resource.Status.License = resource.Spec.DiscoveryDetails.License
	resource.Spec.DiscoveryDetails.ResourceName = "renamed-in-cr"
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			t.Fatal("GetDiscoveryJob() called despite tracked identity drift")
			return stackmonitoringsdk.GetDiscoveryJobResponse{}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite tracked identity drift")
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called during create/update drift handling")
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want tracked identity drift rejection")
	}
	if !strings.Contains(err.Error(), "resourceName") {
		t.Fatalf("CreateOrUpdate() error = %v, want resourceName drift detail", err)
	}
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 0)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
	requireLastCondition(t, resource, shared.Failed)
}

func TestDiscoveryJobWriteOnlyCreateFieldDriftRejectedBeforeMutatingOCI(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	resource.Status.Id = testDiscoveryJobID
	recordDiscoveryJobCreateOnlyState(resource)
	resource.Spec.ShouldPropagateTagsToDiscoveredResources = false
	currentResource := makeDiscoveryJobResource()
	current := sdkDiscoveryJob(currentResource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: current}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite write-only create field drift")
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called during write-only create field drift handling")
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want write-only create field drift rejection")
	}
	if !strings.Contains(err.Error(), "shouldPropagateTagsToDiscoveredResources") {
		t.Fatalf("CreateOrUpdate() error = %v, want shouldPropagateTagsToDiscoveredResources drift detail", err)
	}
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
	if _, ok := discoveryJobRecordedCreateOnlyState(resource); !ok {
		t.Fatalf("create-only state missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	assertDiscoveryJobStatusScrubsCredentialMarkers(t, resource)
}

func TestDiscoveryJobCredentialDriftRejectedWhenReadbackOmitsCredentials(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Generation = 1
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	resource.Status.Id = testDiscoveryJobID
	recordDiscoveryJobCreateOnlyState(resource)
	resource.Generation = 2
	resource.Spec.DiscoveryDetails.Credentials.Items[0].Properties.PropertiesMap["password"] = "rotated"
	currentResource := makeDiscoveryJobResource()
	current := sdkDiscoveryJob(currentResource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	current.DiscoveryDetails.Credentials = nil
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: current}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite credential create-only drift")
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, nil
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called during credential create-only drift handling")
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want credential drift rejection")
	}
	if !strings.Contains(err.Error(), "discoveryDetails.credentials") {
		t.Fatalf("CreateOrUpdate() error = %v, want credential drift detail", err)
	}
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 1)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
	if got := resource.Status.DiscoveryDetails.Credentials.Items; len(got) != 0 {
		t.Fatalf("status.discoveryDetails.credentials.items = %#v, want scrubbed", got)
	}
	assertDiscoveryJobStatusScrubsCredentialMarkers(t, resource)
}

func TestDiscoveryJobDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	active := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	getResponses := []stackmonitoringsdk.GetDiscoveryJobResponse{
		{DiscoveryJob: active},
		{DiscoveryJob: active},
		{DiscoveryJob: active},
	}
	client := &fakeDiscoveryJobOCIClient{
		getFn: getDiscoveryJobResponses(t, &getResponses),
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			requireStringPtr(t, "DeleteDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 1)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-delete")
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDiscoveryJobDeleteByTrackedOCIDIgnoresCreateOnlyIdentityDrift(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	resource.Status.Id = testDiscoveryJobID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.ResourceName = resource.Spec.DiscoveryDetails.ResourceName
	resource.Status.ResourceType = resource.Spec.DiscoveryDetails.ResourceType
	resource.Status.DiscoveryType = resource.Spec.DiscoveryType
	resource.Status.License = resource.Spec.DiscoveryDetails.License
	resource.Spec.DiscoveryDetails.ResourceName = "renamed-in-cr"
	active := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	deletedJob := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateDeleted, stackmonitoringsdk.DiscoveryJobStatusDeleted)
	getResponses := []stackmonitoringsdk.GetDiscoveryJobResponse{
		{DiscoveryJob: active},
		{DiscoveryJob: active},
		{DiscoveryJob: deletedJob},
	}
	client := &fakeDiscoveryJobOCIClient{
		getFn: getDiscoveryJobResponses(t, &getResponses),
		listFn: func(context.Context, stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
			t.Fatal("ListDiscoveryJobs() called despite tracked OCID")
			return stackmonitoringsdk.ListDiscoveryJobsResponse{}, nil
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			requireStringPtr(t, "DeleteDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after deleted readback")
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 1)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-delete")
}

func TestDiscoveryJobDeleteWithoutTrackedIDUsesPaginatedListBeforeDelete(t *testing.T) {
	resource := makeDiscoveryJobResource()
	active := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobStatusSuccess)
	deletedJob := sdkDiscoveryJob(resource, testDiscoveryJobID, stackmonitoringsdk.LifecycleStateDeleted, stackmonitoringsdk.DiscoveryJobStatusDeleted)
	var pages []string
	getCalls := 0
	client := &fakeDiscoveryJobOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListDiscoveryJobsRequest.CompartmentId", request.CompartmentId, testDiscoveryJobCompartmentID)
			requireStringPtr(t, "ListDiscoveryJobsRequest.Name", request.Name, testDiscoveryJobResourceName)
			if request.Page == nil {
				return stackmonitoringsdk.ListDiscoveryJobsResponse{
					DiscoveryJobCollection: stackmonitoringsdk.DiscoveryJobCollection{
						Items: []stackmonitoringsdk.DiscoveryJobSummary{
							sdkDiscoveryJobSummary(resource, "ocid1.stackmonitoringdiscoveryjob.oc1..other", "other-resource", stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobSummaryStatusSuccess),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return stackmonitoringsdk.ListDiscoveryJobsResponse{
				DiscoveryJobCollection: stackmonitoringsdk.DiscoveryJobCollection{
					Items: []stackmonitoringsdk.DiscoveryJobSummary{
						sdkDiscoveryJobSummary(resource, testDiscoveryJobID, testDiscoveryJobResourceName, stackmonitoringsdk.LifecycleStateActive, stackmonitoringsdk.DiscoveryJobSummaryStatusSuccess),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			getCalls++
			if getCalls < 2 {
				return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: active}, nil
			}
			return stackmonitoringsdk.GetDiscoveryJobResponse{DiscoveryJob: deletedJob}, nil
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			requireStringPtr(t, "DeleteDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after deleted readback")
	}
	if got := strings.Join(pages, ","); got != ",page-2,,page-2" {
		t.Fatalf("ListDiscoveryJobs() pages = %q, want two full paginated list scans", got)
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 1)
	assertDiscoveryJobRecordedID(t, resource, testDiscoveryJobID)
}

func TestDiscoveryJobDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-auth"
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
			return stackmonitoringsdk.GetDiscoveryJobResponse{}, authErr
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called after ambiguous pre-delete read")
			return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-auth")
}

func TestDiscoveryJobCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := makeDiscoveryJobResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	client := &fakeDiscoveryJobOCIClient{
		createFn: func(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
			return stackmonitoringsdk.CreateDiscoveryJobResponse{}, createErr
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	assertDiscoveryJobOpcRequestID(t, resource, "opc-create-error")
	requireLastCondition(t, resource, shared.Failed)
}

func newTestDiscoveryJobClient(client discoveryJobOCIClient) DiscoveryJobServiceClient {
	return newDiscoveryJobServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDiscoveryJobResource() *stackmonitoringv1beta1.DiscoveryJob {
	return &stackmonitoringv1beta1.DiscoveryJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "discovery-job",
			Namespace: "default",
		},
		Spec: stackmonitoringv1beta1.DiscoveryJobSpec{
			CompartmentId: testDiscoveryJobCompartmentID,
			DiscoveryDetails: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails{
				AgentId:      testDiscoveryJobAgentID,
				ResourceType: string(stackmonitoringsdk.DiscoveryDetailsResourceTypeOracleDatabase),
				ResourceName: testDiscoveryJobResourceName,
				Properties: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsProperties{
					PropertiesMap: map[string]string{"host": "db.example.com"},
				},
				License: string(stackmonitoringsdk.LicenseTypeEnterpriseEdition),
				Credentials: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials{
					Items: []stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentialsItem{{
						CredentialName: "db-credential",
						CredentialType: "BASIC",
						Properties: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentialsItemProperties{
							PropertiesMap: map[string]string{"username": "admin", "password": "secret"},
						},
					}},
				},
				Tags: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsTags{
					PropertiesMap: map[string]string{"env": "test"},
				},
			},
			DiscoveryType:                            string(stackmonitoringsdk.CreateDiscoveryJobDetailsDiscoveryTypeAdd),
			DiscoveryClient:                          "osok",
			ShouldPropagateTagsToDiscoveredResources: true,
			FreeformTags:                             map[string]string{"owner": "runtime"},
			DefinedTags:                              map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func discoveryJobRequest(resource *stackmonitoringv1beta1.DiscoveryJob) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func sdkDiscoveryJob(
	resource *stackmonitoringv1beta1.DiscoveryJob,
	id string,
	lifecycleState stackmonitoringsdk.LifecycleStateEnum,
	status stackmonitoringsdk.DiscoveryJobStatusEnum,
) stackmonitoringsdk.DiscoveryJob {
	return stackmonitoringsdk.DiscoveryJob{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DiscoveryType:    stackmonitoringsdk.DiscoveryJobDiscoveryTypeEnum(resource.Spec.DiscoveryType),
		Status:           status,
		StatusMessage:    common.String("discovery status"),
		TenantId:         common.String("ocid1.tenancy.oc1..tenant"),
		UserId:           common.String("ocid1.user.oc1..user"),
		DiscoveryClient:  common.String(resource.Spec.DiscoveryClient),
		DiscoveryDetails: sdkDiscoveryDetails(resource.Spec.DiscoveryDetails),
		LifecycleState:   lifecycleState,
		FreeformTags:     discoveryJobStringMap(resource.Spec.FreeformTags),
		DefinedTags:      discoveryJobDefinedTags(resource.Spec.DefinedTags),
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
	}
}

func sdkDiscoveryJobSummary(
	resource *stackmonitoringv1beta1.DiscoveryJob,
	id string,
	resourceName string,
	lifecycleState stackmonitoringsdk.LifecycleStateEnum,
	status stackmonitoringsdk.DiscoveryJobSummaryStatusEnum,
) stackmonitoringsdk.DiscoveryJobSummary {
	return stackmonitoringsdk.DiscoveryJobSummary{
		Id:             common.String(id),
		ResourceType:   stackmonitoringsdk.DiscoveryJobSummaryResourceTypeEnum(resource.Spec.DiscoveryDetails.ResourceType),
		ResourceName:   common.String(resourceName),
		License:        stackmonitoringsdk.LicenseTypeEnum(resource.Spec.DiscoveryDetails.License),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DiscoveryType:  stackmonitoringsdk.DiscoveryJobSummaryDiscoveryTypeEnum(resource.Spec.DiscoveryType),
		Status:         status,
		StatusMessage:  common.String("summary status"),
		TenantId:       common.String("ocid1.tenancy.oc1..tenant"),
		UserId:         common.String("ocid1.user.oc1..user"),
		LifecycleState: lifecycleState,
		FreeformTags:   discoveryJobStringMap(resource.Spec.FreeformTags),
		DefinedTags:    discoveryJobDefinedTags(resource.Spec.DefinedTags),
	}
}

func sdkDiscoveryDetails(spec stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails) *stackmonitoringsdk.DiscoveryDetails {
	details := discoveryJobDetailsFromSpec(spec)
	return details
}

func getDiscoveryJobResponses(
	t *testing.T,
	responses *[]stackmonitoringsdk.GetDiscoveryJobResponse,
) func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
	t.Helper()
	return func(_ context.Context, request stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
		requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
		if len(*responses) == 0 {
			return stackmonitoringsdk.GetDiscoveryJobResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "DiscoveryJob is gone")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func requireDiscoveryJobCreateRequest(
	t *testing.T,
	request stackmonitoringsdk.CreateDiscoveryJobRequest,
	resource *stackmonitoringv1beta1.DiscoveryJob,
) {
	t.Helper()
	requireStringPtr(t, "CreateDiscoveryJobDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateDiscoveryJobRequest.OpcRetryToken is empty")
	}
	if request.ShouldPropagateTagsToDiscoveredResources == nil {
		t.Fatalf("CreateDiscoveryJobDetails.ShouldPropagateTagsToDiscoveredResources = nil, want %t", resource.Spec.ShouldPropagateTagsToDiscoveredResources)
	}
	if *request.ShouldPropagateTagsToDiscoveredResources != resource.Spec.ShouldPropagateTagsToDiscoveredResources {
		t.Fatalf(
			"CreateDiscoveryJobDetails.ShouldPropagateTagsToDiscoveredResources = %t, want %t",
			*request.ShouldPropagateTagsToDiscoveredResources,
			resource.Spec.ShouldPropagateTagsToDiscoveredResources,
		)
	}
	requireDiscoveryJobDetails(t, request.DiscoveryDetails, resource.Spec.DiscoveryDetails)
}

func requireDiscoveryJobDetails(
	t *testing.T,
	got *stackmonitoringsdk.DiscoveryDetails,
	want stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails,
) {
	t.Helper()
	if got == nil {
		t.Fatal("DiscoveryDetails = nil")
	}
	requireStringPtr(t, "DiscoveryDetails.AgentId", got.AgentId, want.AgentId)
	if string(got.ResourceType) != want.ResourceType {
		t.Fatalf("DiscoveryDetails.ResourceType = %q, want %q", got.ResourceType, want.ResourceType)
	}
	requireStringPtr(t, "DiscoveryDetails.ResourceName", got.ResourceName, want.ResourceName)
	if got.Properties == nil || got.Properties.PropertiesMap["host"] != "db.example.com" {
		t.Fatalf("DiscoveryDetails.Properties = %#v, want host property", got.Properties)
	}
	if string(got.License) != want.License {
		t.Fatalf("DiscoveryDetails.License = %q, want %q", got.License, want.License)
	}
	if got.Credentials == nil || len(got.Credentials.Items) != 1 {
		t.Fatalf("DiscoveryDetails.Credentials = %#v, want one credential", got.Credentials)
	}
	credential := got.Credentials.Items[0]
	requireStringPtr(t, "CredentialDetails.CredentialName", credential.CredentialName, "db-credential")
	requireStringPtr(t, "CredentialDetails.CredentialType", credential.CredentialType, "BASIC")
	if credential.Properties == nil || credential.Properties.PropertiesMap["password"] != "secret" {
		t.Fatalf("CredentialDetails.Properties = %#v, want password property in OCI request only", credential.Properties)
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertDiscoveryJobRecordedID(t *testing.T, resource *stackmonitoringv1beta1.DiscoveryJob, want string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertDiscoveryJobOpcRequestID(t *testing.T, resource *stackmonitoringv1beta1.DiscoveryJob, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertDiscoveryJobCreateOnlyStateRecorded(t *testing.T, resource *stackmonitoringv1beta1.DiscoveryJob) {
	t.Helper()
	state, ok := discoveryJobRecordedCreateOnlyState(resource)
	if !ok {
		t.Fatalf("create-only state missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	want := discoveryJobCreateOnlyStateFromResource(resource)
	if state != want {
		t.Fatalf("create-only state = %#v, want %#v", state, want)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, discoveryJobCreateOnlyPropagationField+"=true") {
		t.Fatalf("status.status.message = %q, want human-readable create-only state", resource.Status.OsokStatus.Message)
	}
	if discoveryJobHasCredentials(resource.Spec.DiscoveryDetails.Credentials) &&
		!strings.Contains(resource.Status.OsokStatus.Message, discoveryJobCreateOnlyCredentialsField+"=") {
		t.Fatalf("status.status.message = %q, want non-sensitive credential generation marker", resource.Status.OsokStatus.Message)
	}
}

func assertDiscoveryJobStatusScrubsCredentialMarkers(t *testing.T, resource *stackmonitoringv1beta1.DiscoveryJob) {
	t.Helper()
	encoded, err := json.Marshal(resource.Status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	status := string(encoded)
	for _, forbidden := range []string{
		"secret",
		"password",
		"admin",
		"rotated",
		"db-credential",
		"BASIC",
		discoveryJobLegacyCreateOnlyStatusMarker,
	} {
		if strings.Contains(status, forbidden) {
			t.Fatalf("status contains credential-derived marker %q: %s", forbidden, status)
		}
	}
}

func assertDiscoveryJobCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireLastCondition(
	t *testing.T,
	resource *stackmonitoringv1beta1.DiscoveryJob,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = nil, want %s", want)
	}
	last := conditions[len(conditions)-1]
	if last.Type != want {
		t.Fatalf("last condition type = %s, want %s", last.Type, want)
	}
	if want == shared.Failed && last.Status != corev1.ConditionFalse {
		t.Fatalf("last condition status = %s, want False for Failed", last.Status)
	}
}
