/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package baselineablemetric

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testBaselineableMetricID = "ocid1.baselineablemetric.oc1..example"
	testCompartmentID        = "ocid1.compartment.oc1..example"
)

type fakeBaselineableMetricOCIClient struct {
	createFn func(context.Context, stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error)
	getFn    func(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error)
	listFn   func(context.Context, stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error)
	updateFn func(context.Context, stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error)
	deleteFn func(context.Context, stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error)

	createRequests []stackmonitoringsdk.CreateBaselineableMetricRequest
	getRequests    []stackmonitoringsdk.GetBaselineableMetricRequest
	listRequests   []stackmonitoringsdk.ListBaselineableMetricsRequest
	updateRequests []stackmonitoringsdk.UpdateBaselineableMetricRequest
	deleteRequests []stackmonitoringsdk.DeleteBaselineableMetricRequest
}

func (f *fakeBaselineableMetricOCIClient) CreateBaselineableMetric(
	ctx context.Context,
	req stackmonitoringsdk.CreateBaselineableMetricRequest,
) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return stackmonitoringsdk.CreateBaselineableMetricResponse{}, nil
}

func (f *fakeBaselineableMetricOCIClient) GetBaselineableMetric(
	ctx context.Context,
	req stackmonitoringsdk.GetBaselineableMetricRequest,
) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return stackmonitoringsdk.GetBaselineableMetricResponse{}, nil
}

func (f *fakeBaselineableMetricOCIClient) ListBaselineableMetrics(
	ctx context.Context,
	req stackmonitoringsdk.ListBaselineableMetricsRequest,
) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return stackmonitoringsdk.ListBaselineableMetricsResponse{}, nil
}

func (f *fakeBaselineableMetricOCIClient) UpdateBaselineableMetric(
	ctx context.Context,
	req stackmonitoringsdk.UpdateBaselineableMetricRequest,
) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return stackmonitoringsdk.UpdateBaselineableMetricResponse{}, nil
}

func (f *fakeBaselineableMetricOCIClient) DeleteBaselineableMetric(
	ctx context.Context,
	req stackmonitoringsdk.DeleteBaselineableMetricRequest,
) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return stackmonitoringsdk.DeleteBaselineableMetricResponse{}, nil
}

func TestBaselineableMetricRuntimeHooksConfigureReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newBaselineableMetricRuntimeHooksWithOCIClient(&fakeBaselineableMetricOCIClient{})
	applyBaselineableMetricRuntimeHooks(&hooks, &fakeBaselineableMetricOCIClient{}, nil)

	if hooks.Semantics == nil {
		t.Fatal("Semantics = nil, want reviewed generatedruntime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "lifecycle" {
		t.Fatalf("Async.Strategy = %q, want lifecycle", got)
	}
	if hooks.Semantics.Delete.Policy != "required" || hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", hooks.Semantics.Delete, hooks.Semantics.DeleteFollowUp)
	}
	assertBaselineableMetricStringSliceContains(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "column", "namespace", "freeformTags", "definedTags")
	assertBaselineableMetricStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "id", "lifecycleState", "tenancyId")
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("ParityHooks.NormalizeDesiredState = nil, want spec id normalization from OCI readback")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("Identity.GuardExistingBeforeCreate = nil, want no-id create guard")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
}

func TestBaselineableMetricServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Spec.Id = ""
	fake := &fakeBaselineableMetricOCIClient{}
	fake.listFn = func(_ context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
		t.Fatalf("ListBaselineableMetrics(%#v) should not be called when spec.id is empty", request)
		return stackmonitoringsdk.ListBaselineableMetricsResponse{}, nil
	}
	fake.createFn = func(_ context.Context, request stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
		requireStringPtr(t, "create compartmentId", request.CompartmentId, testCompartmentID)
		requireStringPtr(t, "create column", request.Column, resource.Spec.Column)
		requireStringPtr(t, "create namespace", request.Namespace, resource.Spec.Namespace)
		requireStringPtr(t, "create name", request.Name, resource.Spec.Name)
		return stackmonitoringsdk.CreateBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
			OpcRequestId:       common.String("opc-create"),
		}, nil
	}
	fake.getFn = func(_ context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		requireStringPtr(t, "get baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
		}, nil
	}

	response, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success without requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("Create calls = %d, want 1", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testBaselineableMetricID {
		t.Fatalf("status.ocid = %q, want %q", got, testBaselineableMetricID)
	}
	if got := resource.Status.Id; got != testBaselineableMetricID {
		t.Fatalf("status.id = %q, want %q", got, testBaselineableMetricID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", got)
	}
}

func TestBaselineableMetricServiceClientRejectsOutOfBoxCreateBeforeOCI(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Spec.Id = ""
	resource.Spec.IsOutOfBox = true
	fake := &fakeBaselineableMetricOCIClient{}
	fake.listFn = func(_ context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
		t.Fatalf("ListBaselineableMetrics(%#v) should not be called when spec.id is empty", request)
		return stackmonitoringsdk.ListBaselineableMetricsResponse{}, nil
	}
	fake.createFn = func(context.Context, stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
		t.Fatal("CreateBaselineableMetric should not be called when spec.isOutOfBox cannot be honored")
		return stackmonitoringsdk.CreateBaselineableMetricResponse{}, nil
	}

	response, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "spec.isOutOfBox=true") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported isOutOfBox create rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful create rejection", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("Create calls = %d, want 0", len(fake.createRequests))
	}
}

func TestBaselineableMetricServiceClientCreateGeneratedIDThenNoOps(t *testing.T) {
	t.Parallel()

	const generatedID = "ocid1.baselineablemetric.oc1..generated"
	resource := baselineableMetricResource()
	resource.Spec.Id = ""
	fake := &fakeBaselineableMetricOCIClient{}
	fake.listFn = func(_ context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
		t.Fatalf("ListBaselineableMetrics(%#v) should not be called when creating without spec.id", request)
		return stackmonitoringsdk.ListBaselineableMetricsResponse{}, nil
	}
	fake.createFn = func(context.Context, stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
		return stackmonitoringsdk.CreateBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(generatedID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
			OpcRequestId:       common.String("opc-create"),
		}, nil
	}
	fake.getFn = func(_ context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		requireStringPtr(t, "get baselineableMetricId", request.BaselineableMetricId, generatedID)
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(generatedID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
		}, nil
	}

	client := newBaselineableMetricServiceClientWithOCIClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("initial CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("initial CreateOrUpdate() response = %#v, want active create without requeue", response)
	}
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want no-op success", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("Create calls = %d, want 1", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0 after generated id readback", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != generatedID {
		t.Fatalf("status.ocid = %q, want %q", got, generatedID)
	}
}

func TestBaselineableMetricServiceClientBindsFromPaginatedListWithoutCreate(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	fake := &fakeBaselineableMetricOCIClient{}
	fake.listFn = func(_ context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
		switch page := stringPtrValue(request.Page); page {
		case "":
			return stackmonitoringsdk.ListBaselineableMetricsResponse{
				BaselineableMetricSummaryCollection: stackmonitoringsdk.BaselineableMetricSummaryCollection{
					Items: []stackmonitoringsdk.BaselineableMetricSummary{
						baselineableMetricSummary("ocid1.baselineablemetric.oc1..other", stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return stackmonitoringsdk.ListBaselineableMetricsResponse{
				BaselineableMetricSummaryCollection: stackmonitoringsdk.BaselineableMetricSummaryCollection{
					Items: []stackmonitoringsdk.BaselineableMetricSummary{
						baselineableMetricSummary(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", page)
			return stackmonitoringsdk.ListBaselineableMetricsResponse{}, nil
		}
	}
	fake.getFn = func(_ context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		requireStringPtr(t, "get baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
		}, nil
	}

	response, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("Create calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("List calls = %d, want 2 for pagination", len(fake.listRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testBaselineableMetricID {
		t.Fatalf("status.ocid = %q, want %q", got, testBaselineableMetricID)
	}
}

func TestBaselineableMetricServiceClientRejectsExplicitIDBindMissBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	fake := &fakeBaselineableMetricOCIClient{}
	fake.listFn = func(_ context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
		requireStringPtr(t, "list baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		return stackmonitoringsdk.ListBaselineableMetricsResponse{
			BaselineableMetricSummaryCollection: stackmonitoringsdk.BaselineableMetricSummaryCollection{
				Items: []stackmonitoringsdk.BaselineableMetricSummary{
					baselineableMetricSummary("ocid1.baselineablemetric.oc1..other", stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
				},
			},
		}, nil
	}
	fake.getFn = func(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		t.Fatal("GetBaselineableMetric should not be called when explicit id bind lookup misses")
		return stackmonitoringsdk.GetBaselineableMetricResponse{}, nil
	}
	fake.createFn = func(context.Context, stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
		t.Fatal("CreateBaselineableMetric should not be called when spec.id bind lookup misses")
		return stackmonitoringsdk.CreateBaselineableMetricResponse{}, nil
	}

	response, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "CreateBaselineableMetricDetails does not accept id") {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit id bind-miss rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful bind-miss rejection", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("Create calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("List calls = %d, want 1", len(fake.listRequests))
	}
}

func TestBaselineableMetricServiceClientNoOpDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(_ context.Context, _ stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDKWithTags(testBaselineableMetricID, map[string]string{"managed": "elsewhere"}),
		}, nil
	}

	response, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active no-op without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestBaselineableMetricServiceClientMutableUpdateUsesUpdatePath(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	resource.Spec.Column = "CpuUsage"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	fake := &fakeBaselineableMetricOCIClient{}
	updateApplied := false
	fake.getFn = func(_ context.Context, _ stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		if updateApplied {
			metric := baselineableMetricSDKWithTags(testBaselineableMetricID, map[string]string{"env": "prod"})
			metric.Column = common.String("CpuUsage")
			return stackmonitoringsdk.GetBaselineableMetricResponse{
				BaselineableMetric: metric,
			}, nil
		}
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDKWithTags(testBaselineableMetricID, map[string]string{"env": "dev"}),
		}, nil
	}
	fake.updateFn = func(_ context.Context, request stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error) {
		requireStringPtr(t, "update baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		if got := request.FreeformTags["env"]; got != "prod" {
			t.Fatalf("update freeformTags[env] = %q, want prod", got)
		}
		requireStringPtr(t, "update body id", request.Id, testBaselineableMetricID)
		requireStringPtr(t, "update column", request.Column, "CpuUsage")
		updateApplied = true
		metric := baselineableMetricSDKWithTags(testBaselineableMetricID, map[string]string{"env": "prod"})
		metric.Column = common.String("CpuUsage")
		return stackmonitoringsdk.UpdateBaselineableMetricResponse{
			BaselineableMetric: metric,
			OpcRequestId:       common.String("opc-update"),
		}, nil
	}

	response, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active update without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.FreeformTags["env"]; got != "prod" {
		t.Fatalf("status.freeformTags[env] = %q, want prod", got)
	}
	if got := resource.Status.Column; got != "CpuUsage" {
		t.Fatalf("status.column = %q, want CpuUsage", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
}

func TestBaselineableMetricServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	resource.Spec.LifecycleState = string(stackmonitoringsdk.BaselineableMetricLifeCycleStatesDeleted)
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(_ context.Context, _ stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
		}, nil
	}
	fake.updateFn = func(context.Context, stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error) {
		t.Fatal("UpdateBaselineableMetric should not be called after create-only drift")
		return stackmonitoringsdk.UpdateBaselineableMetricResponse{}, nil
	}

	_, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "lifecycleState") {
		t.Fatalf("CreateOrUpdate() error = %v, want lifecycleState drift rejection", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestBaselineableMetricServiceClientRejectsExplicitIDDriftWithoutMutatingSpec(t *testing.T) {
	t.Parallel()

	const changedID = "ocid1.baselineablemetric.oc1..changed"
	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	resource.Spec.Id = changedID
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(_ context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		requireStringPtr(t, "get baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
		}, nil
	}
	fake.updateFn = func(context.Context, stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error) {
		t.Fatal("UpdateBaselineableMetric should not be called after explicit id drift")
		return stackmonitoringsdk.UpdateBaselineableMetricResponse{}, nil
	}

	_, err := newBaselineableMetricServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("CreateOrUpdate() error = %v, want id drift rejection", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Spec.Id; got != changedID {
		t.Fatalf("spec.id = %q, want preserved explicit value %q", got, changedID)
	}
}

func TestBaselineableMetricServiceClientDeleteWaitsForConfirmedDeletedReadback(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	getCall := 0
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(_ context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		getCall++
		requireStringPtr(t, "get baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		state := stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive
		if getCall == 3 {
			state = stackmonitoringsdk.BaselineableMetricLifeCycleStatesDeleted
		}
		return stackmonitoringsdk.GetBaselineableMetricResponse{BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, state)}, nil
	}
	fake.deleteFn = func(_ context.Context, request stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
		requireStringPtr(t, "delete baselineableMetricId", request.BaselineableMetricId, testBaselineableMetricID)
		return stackmonitoringsdk.DeleteBaselineableMetricResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newBaselineableMetricServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("Delete calls = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete marker after confirmed delete")
	}
}

func TestBaselineableMetricServiceClientDeleteKeepsFinalizerWhenReadbackIsNotTerminal(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		return stackmonitoringsdk.GetBaselineableMetricResponse{
			BaselineableMetric: baselineableMetricSDK(testBaselineableMetricID, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive),
		}, nil
	}
	fake.deleteFn = func(context.Context, stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
		return stackmonitoringsdk.DeleteBaselineableMetricResponse{}, nil
	}

	deleted, err := newBaselineableMetricServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "unexpected lifecycle state") {
		t.Fatalf("Delete() error = %v, want unexpected lifecycle state while active", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until delete is confirmed")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set before terminal delete confirmation")
	}
}

func TestBaselineableMetricServiceClientDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBaselineableMetricID)
	resource.Status.Id = testBaselineableMetricID
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		return stackmonitoringsdk.GetBaselineableMetricResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	fake.deleteFn = func(context.Context, stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
		t.Fatal("DeleteBaselineableMetric should not be called after ambiguous confirm read")
		return stackmonitoringsdk.DeleteBaselineableMetricResponse{}, nil
	}

	deleted, err := newBaselineableMetricServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 blocker", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous 404")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("Delete calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestBaselineableMetricServiceClientDeleteWithoutTrackedIdentityDoesNotDeleteSpecID(t *testing.T) {
	t.Parallel()

	resource := baselineableMetricResource()
	fake := &fakeBaselineableMetricOCIClient{}
	fake.getFn = func(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		t.Fatal("GetBaselineableMetric should not be called before OSOK records an OCI identity")
		return stackmonitoringsdk.GetBaselineableMetricResponse{}, nil
	}
	fake.deleteFn = func(context.Context, stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
		t.Fatal("DeleteBaselineableMetric should not be called before OSOK records an OCI identity")
		return stackmonitoringsdk.DeleteBaselineableMetricResponse{}, nil
	}

	deleted, err := newBaselineableMetricServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for untracked CR cleanup")
	}
}

func baselineableMetricResource() *stackmonitoringv1beta1.BaselineableMetric {
	return &stackmonitoringv1beta1.BaselineableMetric{
		Spec: stackmonitoringv1beta1.BaselineableMetricSpec{
			CompartmentId: testCompartmentID,
			Column:        "CpuUtilization",
			Namespace:     "oci_computeagent",
			Name:          "cpu_utilization",
			ResourceGroup: "instance",
			ResourceType:  "compute",
			Id:            testBaselineableMetricID,
			IsOutOfBox:    false,
		},
	}
}

func baselineableMetricSDK(
	id string,
	state stackmonitoringsdk.BaselineableMetricLifeCycleStatesEnum,
) stackmonitoringsdk.BaselineableMetric {
	return stackmonitoringsdk.BaselineableMetric{
		Id:             common.String(id),
		Name:           common.String("cpu_utilization"),
		Column:         common.String("CpuUtilization"),
		Namespace:      common.String("oci_computeagent"),
		ResourceGroup:  common.String("instance"),
		IsOutOfBox:     common.Bool(false),
		LifecycleState: state,
		TenancyId:      common.String("ocid1.tenancy.oc1..example"),
		CompartmentId:  common.String(testCompartmentID),
		ResourceType:   common.String("compute"),
	}
}

func baselineableMetricSDKWithTags(id string, tags map[string]string) stackmonitoringsdk.BaselineableMetric {
	metric := baselineableMetricSDK(id, stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive)
	metric.FreeformTags = tags
	return metric
}

func baselineableMetricSummary(
	id string,
	state stackmonitoringsdk.BaselineableMetricLifeCycleStatesEnum,
) stackmonitoringsdk.BaselineableMetricSummary {
	metric := baselineableMetricSDK(id, state)
	return stackmonitoringsdk.BaselineableMetricSummary{
		Id:             metric.Id,
		Name:           metric.Name,
		Column:         metric.Column,
		Namespace:      metric.Namespace,
		ResourceGroup:  metric.ResourceGroup,
		IsOutOfBox:     metric.IsOutOfBox,
		LifecycleState: metric.LifecycleState,
		TenancyId:      metric.TenancyId,
		CompartmentId:  metric.CompartmentId,
		ResourceType:   metric.ResourceType,
		FreeformTags:   metric.FreeformTags,
		DefinedTags:    metric.DefinedTags,
		SystemTags:     metric.SystemTags,
	}
}

func requireStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if got := *value; got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertBaselineableMetricStringSliceContains(t *testing.T, label string, got []string, wants ...string) {
	t.Helper()
	values := make(map[string]bool, len(got))
	for _, value := range got {
		values[value] = true
	}
	for _, want := range wants {
		if !values[want] {
			t.Fatalf("%s = %#v, want to contain %q", label, got, want)
		}
	}
}
