/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package hostname

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLoadBalancerID = "ocid1.loadbalancer.oc1..loadbalancer"
	testHostnameName   = "example_hostname_001"
	testHostnameValue  = "app.example.com"
)

func TestHostnameRuntimeSemanticsDocumentResourceLocalException(t *testing.T) {
	semantics := newHostnameRuntimeSemantics()

	if semantics.FormalService != "loadbalancer" || semantics.FormalSlug != "hostname" {
		t.Fatalf("semantics formal target = %s/%s, want loadbalancer/hostname", semantics.FormalService, semantics.FormalSlug)
	}
	if semantics.Async == nil ||
		semantics.Async.Strategy != "workrequest" ||
		semantics.Async.Runtime != "handwritten" ||
		semantics.Async.FormalClassification != "workrequest" {
		t.Fatalf("semantics async = %#v, want handwritten workrequest", semantics.Async)
	}
	if semantics.Async.WorkRequest == nil || !containsTestString(semantics.Async.WorkRequest.Phases, "create") || !containsTestString(semantics.Async.WorkRequest.Phases, "update") || !containsTestString(semantics.Async.WorkRequest.Phases, "delete") {
		t.Fatalf("semantics workrequest = %#v, want create/update/delete", semantics.Async.WorkRequest)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", semantics.FinalizerPolicy)
	}
	if !containsTestString(semantics.Mutation.Mutable, "hostname") {
		t.Fatalf("mutable fields = %#v, want hostname", semantics.Mutation.Mutable)
	}
	for _, field := range []string{"loadBalancerId", "name"} {
		if !containsTestString(semantics.Mutation.ForceNew, field) {
			t.Fatalf("forceNew fields = %#v, want %s", semantics.Mutation.ForceNew, field)
		}
	}
	if len(semantics.Unsupported) != 1 || !strings.Contains(semantics.Unsupported[0].StopCondition, "metadata annotation") {
		t.Fatalf("unsupported semantics = %#v, want annotation exception", semantics.Unsupported)
	}
}

func TestHostnameCreateRequiresLoadBalancerAnnotationBeforeOCICalls(t *testing.T) {
	resource := newHostnameResource()
	resource.Annotations = nil
	client := newHostnameRuntimeClient(&fakeHostnameOCIClient{t: t}, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatalf("CreateOrUpdate error = nil, want missing annotation error")
	}
	if !strings.Contains(err.Error(), hostnameLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate error = %q, want annotation name", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestHostnameCreateBindsStatusAfterReadback(t *testing.T) {
	resource := newHostnameResource()
	fake := &fakeHostnameOCIClient{
		t: t,
		getResults: []hostnameGetResult{
			{err: notFoundErr()},
			{response: getHostnameResponse(testHostnameName, testHostnameValue)},
		},
		listResults: []hostnameListResult{{response: loadbalancersdk.ListHostnamesResponse{}}},
		createResults: []hostnameCreateResult{{
			response: loadbalancersdk.CreateHostnameResponse{
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			},
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.getRequests) != 2 || len(fake.listRequests) != 1 || len(fake.createRequests) != 1 {
		t.Fatalf("call counts get/list/create = %d/%d/%d, want 2/1/1", len(fake.getRequests), len(fake.listRequests), len(fake.createRequests))
	}
	assertCreateRequest(t, fake.createRequests[0], testLoadBalancerID, testHostnameName, testHostnameValue)
	requireHostnameStatus(t, resource, testLoadBalancerID, testHostnameName, testHostnameValue)
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async.current = %#v, want nil", resource.Status.OsokStatus.Async.Current)
	}
}

func TestHostnameBindExistingViaListBeforeCreate(t *testing.T) {
	resource := newHostnameResource()
	fake := &fakeHostnameOCIClient{
		t:          t,
		getResults: []hostnameGetResult{{err: notFoundErr()}},
		listResults: []hostnameListResult{{
			response: loadbalancersdk.ListHostnamesResponse{
				Items: []loadbalancersdk.Hostname{
					sdkHostname("other", "other.example.com"),
					sdkHostname(testHostnameName, testHostnameValue),
				},
			},
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateHostname calls = %d, want 0", len(fake.createRequests))
	}
	requireHostnameStatus(t, resource, testLoadBalancerID, testHostnameName, testHostnameValue)
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestHostnameCreatePendingRequeuesAndAvoidsDuplicateCreate(t *testing.T) {
	resource := newHostnameResource()
	fake := &fakeHostnameOCIClient{
		t: t,
		getResults: []hostnameGetResult{
			{err: notFoundErr()},
			{err: notFoundErr()},
			{err: notFoundErr()},
		},
		listResults: []hostnameListResult{
			{response: loadbalancersdk.ListHostnamesResponse{}},
			{response: loadbalancersdk.ListHostnamesResponse{}},
			{response: loadbalancersdk.ListHostnamesResponse{}},
		},
		createResults: []hostnameCreateResult{{
			response: loadbalancersdk.CreateHostnameResponse{
				OpcWorkRequestId: common.String("wr-create"),
			},
		}},
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-create", loadbalancersdk.WorkRequestLifecycleStateInProgress, "CreateHostname", "create still running"),
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("first CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("first response = %#v, want successful requeue", response)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, hostnameCreatePendingState, "wr-create")
	requireCondition(t, resource, shared.Provisioning, v1.ConditionTrue)

	response, err = client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("second CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("second response = %#v, want successful requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateHostname calls = %d, want 1", len(fake.createRequests))
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, string(loadbalancersdk.WorkRequestLifecycleStateInProgress), "wr-create")
}

func TestHostnameCreateWorkRequestFailedProjectsFailed(t *testing.T) {
	resource := newHostnameResource()
	markTestAsync(resource, shared.OSOKAsyncPhaseCreate, hostnameCreatePendingState, "wr-create")
	fake := &fakeHostnameOCIClient{
		t: t,
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-create", loadbalancersdk.WorkRequestLifecycleStateFailed, "CreateHostname", "create failed"),
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), "work request wr-create finished with status FAILED") {
		t.Fatalf("CreateOrUpdate error = %v, want failed work request error", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want failed without requeue", response)
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.getRequests)+len(fake.listRequests)+len(fake.createRequests) != 0 {
		t.Fatalf("read/list/create calls = %d/%d/%d, want none after failed work request", len(fake.getRequests), len(fake.listRequests), len(fake.createRequests))
	}
	requireAsyncCurrentClass(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, string(loadbalancersdk.WorkRequestLifecycleStateFailed), "wr-create", shared.OSOKAsyncClassFailed)
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestHostnameObserveActiveDoesNotUpdate(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, testHostnameValue)
	fake := &fakeHostnameOCIClient{
		t:          t,
		getResults: []hostnameGetResult{{response: getHostnameResponse(testHostnameName, testHostnameValue)}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateHostname calls = %d, want 0", len(fake.updateRequests))
	}
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestHostnameUpdateMutableHostname(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, "old.example.com")
	resource.Spec.Hostname = "new.example.com"
	fake := &fakeHostnameOCIClient{
		t: t,
		getResults: []hostnameGetResult{
			{response: getHostnameResponse(testHostnameName, "old.example.com")},
			{response: getHostnameResponse(testHostnameName, "new.example.com")},
		},
		updateResults: []hostnameUpdateResult{{
			response: loadbalancersdk.UpdateHostnameResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			},
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateHostname calls = %d, want 1", len(fake.updateRequests))
	}
	assertUpdateRequest(t, fake.updateRequests[0], testLoadBalancerID, testHostnameName, "new.example.com")
	requireHostnameStatus(t, resource, testLoadBalancerID, testHostnameName, "new.example.com")
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestHostnameUpdatePendingLifecycle(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, "old.example.com")
	resource.Spec.Hostname = "new.example.com"
	fake := &fakeHostnameOCIClient{
		t: t,
		getResults: []hostnameGetResult{
			{response: getHostnameResponse(testHostnameName, "old.example.com")},
			{response: getHostnameResponse(testHostnameName, "old.example.com")},
			{response: getHostnameResponse(testHostnameName, "old.example.com")},
		},
		updateResults: []hostnameUpdateResult{{
			response: loadbalancersdk.UpdateHostnameResponse{
				OpcWorkRequestId: common.String("wr-update"),
			},
		}},
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-update", loadbalancersdk.WorkRequestLifecycleStateInProgress, "UpdateHostname", "update still running"),
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("first CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("first response = %#v, want successful requeue", response)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, hostnameUpdatePendingState, "wr-update")
	requireCondition(t, resource, shared.Updating, v1.ConditionTrue)

	response, err = client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("second CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("second response = %#v, want successful requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateHostname calls = %d, want 1", len(fake.updateRequests))
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, string(loadbalancersdk.WorkRequestLifecycleStateInProgress), "wr-update")
}

func TestHostnameUpdateWorkRequestFailedProjectsFailed(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, "old.example.com")
	resource.Spec.Hostname = "new.example.com"
	markTestAsync(resource, shared.OSOKAsyncPhaseUpdate, hostnameUpdatePendingState, "wr-update")
	fake := &fakeHostnameOCIClient{
		t: t,
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-update", loadbalancersdk.WorkRequestLifecycleStateFailed, "UpdateHostname", "update failed"),
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), "work request wr-update finished with status FAILED") {
		t.Fatalf("CreateOrUpdate error = %v, want failed work request error", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want failed without requeue", response)
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.getRequests)+len(fake.listRequests)+len(fake.updateRequests) != 0 {
		t.Fatalf("read/list/update calls = %d/%d/%d, want none after failed work request", len(fake.getRequests), len(fake.listRequests), len(fake.updateRequests))
	}
	requireAsyncCurrentClass(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseUpdate, string(loadbalancersdk.WorkRequestLifecycleStateFailed), "wr-update", shared.OSOKAsyncClassFailed)
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestHostnameImmutableDriftRejectedBeforeOCIMutation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*loadbalancerv1beta1.Hostname)
		wantErr string
	}{
		{
			name: "name",
			mutate: func(resource *loadbalancerv1beta1.Hostname) {
				resource.Spec.Name = "replacement"
			},
			wantErr: "require replacement when name changes",
		},
		{
			name: "loadBalancerId",
			mutate: func(resource *loadbalancerv1beta1.Hostname) {
				resource.Annotations[hostnameLoadBalancerIDAnnotation] = "ocid1.loadbalancer.oc1..replacement"
			},
			wantErr: "require replacement when loadBalancerId changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := newHostnameResource()
			trackHostname(resource, testHostnameValue)
			tt.mutate(resource)
			fake := &fakeHostnameOCIClient{t: t}
			client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

			response, err := client.CreateOrUpdate(context.Background(), resource, testRequest())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("CreateOrUpdate error = %v, want %q", err, tt.wantErr)
			}
			if response.IsSuccessful {
				t.Fatalf("response.IsSuccessful = true, want false")
			}
			if len(fake.getRequests)+len(fake.listRequests)+len(fake.updateRequests)+len(fake.createRequests)+len(fake.deleteRequests) != 0 {
				t.Fatalf("OCI calls were issued before immutable drift rejection")
			}
			requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
		})
	}
}

func TestHostnameDeleteMarksTerminatingAndRetainsFinalizer(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, testHostnameValue)
	fake := &fakeHostnameOCIClient{
		t:             t,
		deleteResults: []hostnameDeleteResult{{response: loadbalancersdk.DeleteHostnameResponse{OpcWorkRequestId: common.String("wr-delete")}}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if deleted {
		t.Fatalf("deleted = true, want false until readback confirms delete")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteHostname calls = %d, want 1", len(fake.deleteRequests))
	}
	assertDeleteRequest(t, fake.deleteRequests[0], testLoadBalancerID, testHostnameName)
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, hostnameDeletePendingState, "wr-delete")
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func TestHostnamePendingDeleteReadSuccessDoesNotRepeatDelete(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, testHostnameValue)
	markTestAsync(resource, shared.OSOKAsyncPhaseDelete, hostnameDeletePendingState, "wr-delete")
	fake := &fakeHostnameOCIClient{
		t: t,
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-delete", loadbalancersdk.WorkRequestLifecycleStateInProgress, "DeleteHostname", "delete still running"),
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if deleted {
		t.Fatalf("deleted = true, want false while readback still finds hostname")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteHostname calls = %d, want 0", len(fake.deleteRequests))
	}
	if len(fake.getRequests)+len(fake.listRequests) != 0 {
		t.Fatalf("read/list calls = %d/%d, want none while work request is still running", len(fake.getRequests), len(fake.listRequests))
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, string(loadbalancersdk.WorkRequestLifecycleStateInProgress), "wr-delete")
}

func TestHostnamePendingDeleteReadNotFoundReleasesFinalizer(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, testHostnameValue)
	markTestAsync(resource, shared.OSOKAsyncPhaseDelete, hostnameDeletePendingState, "wr-delete")
	fake := &fakeHostnameOCIClient{
		t: t,
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-delete", loadbalancersdk.WorkRequestLifecycleStateSucceeded, "DeleteHostname", "delete succeeded"),
		}},
		getResults:  []hostnameGetResult{{err: notFoundErr()}},
		listResults: []hostnameListResult{{response: loadbalancersdk.ListHostnamesResponse{}}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if !deleted {
		t.Fatalf("deleted = false, want true after readback confirms not found")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteHostname calls = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatalf("status.deletedAt = nil, want timestamp")
	}
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func TestHostnameDeleteWorkRequestFailedProjectsFailed(t *testing.T) {
	resource := newHostnameResource()
	trackHostname(resource, testHostnameValue)
	markTestAsync(resource, shared.OSOKAsyncPhaseDelete, hostnameDeletePendingState, "wr-delete")
	fake := &fakeHostnameOCIClient{
		t: t,
		workRequestResults: []hostnameWorkRequestResult{{
			response: workRequestResponse("wr-delete", loadbalancersdk.WorkRequestLifecycleStateFailed, "DeleteHostname", "delete failed"),
		}},
	}
	client := newHostnameRuntimeClient(fake, nilLogger()).(*hostnameRuntimeClient)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "work request wr-delete finished with status FAILED") {
		t.Fatalf("Delete error = %v, want failed work request error", err)
	}
	if deleted {
		t.Fatalf("deleted = true, want false after failed work request")
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.getRequests)+len(fake.listRequests)+len(fake.deleteRequests) != 0 {
		t.Fatalf("read/list/delete calls = %d/%d/%d, want none after failed work request", len(fake.getRequests), len(fake.listRequests), len(fake.deleteRequests))
	}
	requireAsyncCurrentClass(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseDelete, string(loadbalancersdk.WorkRequestLifecycleStateFailed), "wr-delete", shared.OSOKAsyncClassFailed)
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

type hostnameGetResult struct {
	response loadbalancersdk.GetHostnameResponse
	err      error
}

type hostnameListResult struct {
	response loadbalancersdk.ListHostnamesResponse
	err      error
}

type hostnameCreateResult struct {
	response loadbalancersdk.CreateHostnameResponse
	err      error
}

type hostnameUpdateResult struct {
	response loadbalancersdk.UpdateHostnameResponse
	err      error
}

type hostnameDeleteResult struct {
	response loadbalancersdk.DeleteHostnameResponse
	err      error
}

type hostnameWorkRequestResult struct {
	response loadbalancersdk.GetWorkRequestResponse
	err      error
}

type fakeHostnameOCIClient struct {
	t *testing.T

	getResults         []hostnameGetResult
	listResults        []hostnameListResult
	createResults      []hostnameCreateResult
	updateResults      []hostnameUpdateResult
	deleteResults      []hostnameDeleteResult
	workRequestResults []hostnameWorkRequestResult

	getRequests         []loadbalancersdk.GetHostnameRequest
	listRequests        []loadbalancersdk.ListHostnamesRequest
	createRequests      []loadbalancersdk.CreateHostnameRequest
	updateRequests      []loadbalancersdk.UpdateHostnameRequest
	deleteRequests      []loadbalancersdk.DeleteHostnameRequest
	workRequestRequests []loadbalancersdk.GetWorkRequestRequest
}

func (f *fakeHostnameOCIClient) CreateHostname(_ context.Context, request loadbalancersdk.CreateHostnameRequest) (loadbalancersdk.CreateHostnameResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if len(f.createResults) == 0 {
		f.t.Fatalf("unexpected CreateHostname call")
		return loadbalancersdk.CreateHostnameResponse{}, nil
	}
	result := f.createResults[0]
	f.createResults = f.createResults[1:]
	return result.response, result.err
}

func (f *fakeHostnameOCIClient) GetHostname(_ context.Context, request loadbalancersdk.GetHostnameRequest) (loadbalancersdk.GetHostnameResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		f.t.Fatalf("unexpected GetHostname call")
		return loadbalancersdk.GetHostnameResponse{}, nil
	}
	result := f.getResults[0]
	f.getResults = f.getResults[1:]
	return result.response, result.err
}

func (f *fakeHostnameOCIClient) ListHostnames(_ context.Context, request loadbalancersdk.ListHostnamesRequest) (loadbalancersdk.ListHostnamesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		f.t.Fatalf("unexpected ListHostnames call")
		return loadbalancersdk.ListHostnamesResponse{}, nil
	}
	result := f.listResults[0]
	f.listResults = f.listResults[1:]
	return result.response, result.err
}

func (f *fakeHostnameOCIClient) UpdateHostname(_ context.Context, request loadbalancersdk.UpdateHostnameRequest) (loadbalancersdk.UpdateHostnameResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if len(f.updateResults) == 0 {
		f.t.Fatalf("unexpected UpdateHostname call")
		return loadbalancersdk.UpdateHostnameResponse{}, nil
	}
	result := f.updateResults[0]
	f.updateResults = f.updateResults[1:]
	return result.response, result.err
}

func (f *fakeHostnameOCIClient) DeleteHostname(_ context.Context, request loadbalancersdk.DeleteHostnameRequest) (loadbalancersdk.DeleteHostnameResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		f.t.Fatalf("unexpected DeleteHostname call")
		return loadbalancersdk.DeleteHostnameResponse{}, nil
	}
	result := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return result.response, result.err
}

func (f *fakeHostnameOCIClient) GetWorkRequest(_ context.Context, request loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if len(f.workRequestResults) == 0 {
		f.t.Fatalf("unexpected GetWorkRequest call")
		return loadbalancersdk.GetWorkRequestResponse{}, nil
	}
	result := f.workRequestResults[0]
	f.workRequestResults = f.workRequestResults[1:]
	return result.response, result.err
}

func newHostnameResource() *loadbalancerv1beta1.Hostname {
	return &loadbalancerv1beta1.Hostname{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hostname-cr",
			Namespace: "default",
			Annotations: map[string]string{
				hostnameLoadBalancerIDAnnotation: testLoadBalancerID,
			},
		},
		Spec: loadbalancerv1beta1.HostnameSpec{
			Name:     testHostnameName,
			Hostname: testHostnameValue,
		},
	}
}

func trackHostname(resource *loadbalancerv1beta1.Hostname, hostnameValue string) {
	resource.Status.OsokStatus.Ocid = shared.OCID(testLoadBalancerID)
	resource.Status.Name = testHostnameName
	resource.Status.Hostname = hostnameValue
}

func markTestAsync(resource *loadbalancerv1beta1.Hostname, phase shared.OSOKAsyncPhase, rawState string, workRequestID string) {
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		RawStatus:       rawState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
}

func getHostnameResponse(name string, hostnameValue string) loadbalancersdk.GetHostnameResponse {
	return loadbalancersdk.GetHostnameResponse{
		Hostname:     sdkHostname(name, hostnameValue),
		OpcRequestId: common.String("opc-get"),
	}
}

func sdkHostname(name string, hostnameValue string) loadbalancersdk.Hostname {
	return loadbalancersdk.Hostname{
		Name:     common.String(name),
		Hostname: common.String(hostnameValue),
	}
}

func workRequestResponse(id string, state loadbalancersdk.WorkRequestLifecycleStateEnum, operationType string, message string) loadbalancersdk.GetWorkRequestResponse {
	return loadbalancersdk.GetWorkRequestResponse{
		WorkRequest: loadbalancersdk.WorkRequest{
			Id:             common.String(id),
			LifecycleState: state,
			Type:           common.String(operationType),
			Message:        common.String(message),
		},
		OpcRequestId: common.String("opc-workrequest"),
	}
}

func notFoundErr() error {
	return errortest.NewServiceError(404, errorutil.NotFound, "hostname not found")
}

func requireHostnameStatus(t *testing.T, resource *loadbalancerv1beta1.Hostname, wantLoadBalancerID string, wantName string, wantHostname string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != wantLoadBalancerID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantName {
		t.Fatalf("status.name = %q, want %q", got, wantName)
	}
	if got := resource.Status.Hostname; got != wantHostname {
		t.Fatalf("status.hostname = %q, want %q", got, wantHostname)
	}
}

func requireCondition(t *testing.T, resource *loadbalancerv1beta1.Hostname, condition shared.OSOKConditionType, status v1.ConditionStatus) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = nil, want %s", condition)
	}
	got := conditions[len(conditions)-1]
	if got.Type != condition || got.Status != status {
		t.Fatalf("last condition = %s/%s, want %s/%s", got.Type, got.Status, condition, status)
	}
}

func requireAsyncCurrent(t *testing.T, resource *loadbalancerv1beta1.Hostname, phase shared.OSOKAsyncPhase, rawStatus string, workRequestID string) {
	t.Helper()
	requireAsyncCurrentClass(t, resource, "", phase, rawStatus, workRequestID, shared.OSOKAsyncClassPending)
}

func requireAsyncCurrentClass(
	t *testing.T,
	resource *loadbalancerv1beta1.Hostname,
	source shared.OSOKAsyncSource,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("async.current = nil, want phase %s", phase)
	}
	if source != "" && current.Source != source {
		t.Fatalf("async.current.source = %q, want %q", current.Source, source)
	}
	if current.Phase != phase {
		t.Fatalf("async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("async.current.workRequestID = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func assertCreateRequest(t *testing.T, request loadbalancersdk.CreateHostnameRequest, wantLoadBalancerID string, wantName string, wantHostname string) {
	t.Helper()
	if got := stringValue(request.LoadBalancerId); got != wantLoadBalancerID {
		t.Fatalf("CreateHostname.LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(request.CreateHostnameDetails.Name); got != wantName {
		t.Fatalf("CreateHostname.Name = %q, want %q", got, wantName)
	}
	if got := stringValue(request.CreateHostnameDetails.Hostname); got != wantHostname {
		t.Fatalf("CreateHostname.Hostname = %q, want %q", got, wantHostname)
	}
}

func assertUpdateRequest(t *testing.T, request loadbalancersdk.UpdateHostnameRequest, wantLoadBalancerID string, wantName string, wantHostname string) {
	t.Helper()
	if got := stringValue(request.LoadBalancerId); got != wantLoadBalancerID {
		t.Fatalf("UpdateHostname.LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(request.Name); got != wantName {
		t.Fatalf("UpdateHostname.Name = %q, want %q", got, wantName)
	}
	if got := stringValue(request.UpdateHostnameDetails.Hostname); got != wantHostname {
		t.Fatalf("UpdateHostname.Hostname = %q, want %q", got, wantHostname)
	}
}

func assertDeleteRequest(t *testing.T, request loadbalancersdk.DeleteHostnameRequest, wantLoadBalancerID string, wantName string) {
	t.Helper()
	if got := stringValue(request.LoadBalancerId); got != wantLoadBalancerID {
		t.Fatalf("DeleteHostname.LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(request.Name); got != wantName {
		t.Fatalf("DeleteHostname.Name = %q, want %q", got, wantName)
	}
}

func containsTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func testRequest() ctrl.Request {
	return ctrl.Request{}
}

func nilLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
