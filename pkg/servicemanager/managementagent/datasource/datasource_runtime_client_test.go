/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasource

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestDataSourceCreateOrUpdateRequiresParentAnnotationBeforeOCICalls(t *testing.T) {
	resource := newTestDataSource()
	resource.Annotations = nil
	client := newTestDataSourceRuntimeClient(t, &fakeDataSourceRuntimeOCIClient{t: t})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate returned nil error; want missing annotation error")
	}
	if !strings.Contains(err.Error(), dataSourceManagementAgentIDAnnotation) {
		t.Fatalf("CreateOrUpdate error = %q, want annotation %q", err.Error(), dataSourceManagementAgentIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate response IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status reason = %q, want %q", got, shared.Failed)
	}
}

func TestDataSourceCreateOrUpdateBindsFromSecondListPage(t *testing.T) {
	resource := newTestDataSource()
	var listCalls int
	client := newTestDataSourceRuntimeClient(t, pagedBindingDataSourceClient(t, &listCalls))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful without requeue", response)
	}
	if listCalls != 2 {
		t.Fatalf("ListDataSources calls = %d, want 2", listCalls)
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID("management-agent-1") {
		t.Fatalf("status.status.ocid = %q, want management-agent-1", got)
	}
	if got := resource.Status.Key; got != "datasource-key" {
		t.Fatalf("status.key = %q, want datasource-key", got)
	}
	if got := resource.Status.State; got != string(managementagentsdk.LifecycleStatesActive) {
		t.Fatalf("status.state = %q, want ACTIVE", got)
	}
}

func TestDataSourceCreateUsesPrometheusEmitterWorkRequestAndOpcRequestID(t *testing.T) {
	resource := newTestDataSource()
	var createCalls int
	client := newTestDataSourceRuntimeClient(t, createDataSourceClient(t, &createCalls))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDataSource calls = %d, want 1", createCalls)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful requeue", response)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status async current = nil, want create work request")
	}
	if current.Phase != shared.OSOKAsyncPhaseCreate || current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("async current = %+v, want workrequest create", current)
	}
	if current.WorkRequestID != "wr-create" {
		t.Fatalf("async workRequestID = %q, want wr-create", current.WorkRequestID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status opcRequestId = %q, want opc-create", got)
	}
}

func TestDataSourceNoopReconcileKeepsObservedStateWithoutUpdate(t *testing.T) {
	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://prometheus.example", "oci_metrics"),
			}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			t.Fatal("UpdateDataSource was called for a no-op reconcile")
			return managementagentsdk.UpdateDataSourceResponse{}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful without requeue", response)
	}
	if got := resource.Status.Url; got != "http://prometheus.example" {
		t.Fatalf("status.url = %q, want http://prometheus.example", got)
	}
}

func TestDataSourceMutableUpdateUsesPrometheusEmitterBody(t *testing.T) {
	resource := newTestDataSource()
	resource.Spec.Url = "http://new.example"
	seedTrackedDataSource(resource)
	var updateCalls int
	client := newTestDataSourceRuntimeClient(t, updateDataSourceClient(t, &updateCalls, "http://old.example", "http://new.example"))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDataSource calls = %d, want 1", updateCalls)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful requeue", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status opcRequestId = %q, want opc-update", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseUpdate || current.WorkRequestID != "wr-update" {
		t.Fatalf("async current = %+v, want update work request wr-update", current)
	}
}

func TestDataSourceOmittedOptionalFieldsDoNotTriggerUpdate(t *testing.T) {
	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	client := newTestDataSourceRuntimeClient(t, noUpdateForDefaultedOptionalDataSourceClient(t))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful without requeue", response)
	}
	if got := resource.Status.AllowMetrics; got != "up,down" {
		t.Fatalf("status.allowMetrics = %q, want up,down", got)
	}
	if got := resource.Status.ReadDataLimit; got != 300 {
		t.Fatalf("status.readDataLimit = %d, want 300", got)
	}
}

func TestDataSourceMutableUpdatePreservesOmittedOptionalFields(t *testing.T) {
	resource := newTestDataSource()
	resource.Spec.Url = "http://new.example"
	seedTrackedDataSource(resource)
	var updateCalls int
	client := newTestDataSourceRuntimeClient(t, updateDataSourcePreservingOptionalsClient(t, &updateCalls))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDataSource calls = %d, want 1", updateCalls)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful requeue", response)
	}
}

func TestDataSourceMutableUpdateSendsExplicitOptionalFields(t *testing.T) {
	resource := newTestDataSource()
	resource.Spec.AllowMetrics = "latency,requests"
	resource.Spec.ProxyUrl = "http://new-proxy.example"
	resource.Spec.ConnectionTimeout = 101
	resource.Spec.ReadTimeout = 202
	resource.Spec.ReadDataLimitInKilobytes = 303
	resource.Spec.ScheduleMins = 6
	resource.Spec.ResourceGroup = "new-resource-group"
	resource.Spec.MetricDimensions = []managementagentv1beta1.DataSourceMetricDimension{{
		Name:  "service",
		Value: "api",
	}}
	seedTrackedDataSource(resource)
	var updateCalls int
	client := newTestDataSourceRuntimeClient(t, updateDataSourceExplicitOptionalsClient(t, &updateCalls))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDataSource calls = %d, want 1", updateCalls)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate response = %+v, want successful requeue", response)
	}
}

func TestDataSourceRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestDataSource()
	resource.Spec.Namespace = "new_namespace"
	resource.Status.OsokStatus.Ocid = "management-agent-1"
	resource.Status.Key = "datasource-key"
	resource.Status.Name = "sample-ds"
	resource.Status.Type = string(managementagentsdk.DataSourceTypesPrometheusEmitter)
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://old.example", "old_namespace"),
			}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			t.Fatal("UpdateDataSource was called for create-only namespace drift")
			return managementagentsdk.UpdateDataSourceResponse{}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate returned nil error; want namespace drift error")
	}
	if !strings.Contains(err.Error(), "namespace changed") {
		t.Fatalf("CreateOrUpdate error = %q, want namespace drift", err.Error())
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate response IsSuccessful = true, want false")
	}
}

func TestDataSourceRejectsCreateOnlyNamespaceRemovalBeforeUpdate(t *testing.T) {
	resource := newTestDataSource()
	resource.Spec.Namespace = ""
	seedTrackedDataSource(resource)
	client := newTestDataSourceRuntimeClient(t, namespaceDriftDataSourceClient(t))

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate returned nil error; want namespace drift error")
	}
	if !strings.Contains(err.Error(), "namespace changed") {
		t.Fatalf("CreateOrUpdate error = %q, want namespace drift", err.Error())
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate response IsSuccessful = true, want false")
	}
}

func TestDataSourceDeleteStartsWorkRequestAndKeepsFinalizer(t *testing.T) {
	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	var deleteCalls int
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://prometheus.example", "oci_metrics"),
			}, nil
		},
		deleteFn: func(_ context.Context, request managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			deleteCalls++
			if got := stringValue(request.ManagementAgentId); got != "management-agent-1" {
				t.Fatalf("DeleteDataSource managementAgentId = %q, want management-agent-1", got)
			}
			if got := stringValue(request.DataSourceKey); got != "datasource-key" {
				t.Fatalf("DeleteDataSource key = %q, want datasource-key", got)
			}
			return managementagentsdk.DeleteDataSourceResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDataSource calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status opcRequestId = %q, want opc-delete", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.WorkRequestID != "wr-delete" {
		t.Fatalf("async current = %+v, want delete work request wr-delete", current)
	}
}

func TestDataSourceDeleteReadbackDeletingSkipsDuplicateDelete(t *testing.T) {
	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	var getCalls int
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			getCalls++
			return managementagentsdk.GetDataSourceResponse{
				DataSource: prometheusDataSourceWithState(
					"datasource-key",
					"sample-ds",
					"http://prometheus.example",
					"oci_metrics",
					managementagentsdk.LifecycleStatesDeleting,
				),
			}, nil
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			t.Fatal("DeleteDataSource was called after pre-delete readback returned DELETING")
			return managementagentsdk.DeleteDataSourceResponse{}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained while OCI delete is pending")
	}
	if getCalls != 1 {
		t.Fatalf("GetDataSource calls = %d, want 1", getCalls)
	}
	current := requireDataSourceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncSourceLifecycle)
	if current.WorkRequestID != "" {
		t.Fatalf("status.async.current.workRequestId = %q, want empty", current.WorkRequestID)
	}
	if current.RawStatus != string(managementagentsdk.LifecycleStatesDeleting) ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %+v, want lifecycle DELETING pending projection", current)
	}
}

func TestDataSourceDeleteReleasesAfterMissingParentAnnotationValidation(t *testing.T) {
	resource := newTestDataSource()
	resource.Annotations = nil
	client := newTestDataSourceRuntimeClient(t, &fakeDataSourceRuntimeOCIClient{t: t})

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil {
		t.Fatal("CreateOrUpdate returned nil error; want missing annotation validation error")
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete returned deleted=false, want finalizer release for untracked data source")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", got, shared.Terminating)
	}
}

func TestDataSourceDeleteReleasesAfterUnsupportedTypeValidation(t *testing.T) {
	resource := newTestDataSource()
	resource.Spec.Type = string(managementagentsdk.DataSourceTypesKubernetesCluster)
	client := newTestDataSourceRuntimeClient(t, &fakeDataSourceRuntimeOCIClient{t: t})

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil {
		t.Fatal("CreateOrUpdate returned nil error; want unsupported type validation error")
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete returned deleted=false, want finalizer release for untracked data source")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", got, shared.Terminating)
	}
}

func TestDataSourceDeleteUsesTrackedKubernetesClusterIdentity(t *testing.T) {
	resource := newTestDataSource()
	resource.Annotations = nil
	resource.Spec.Name = "kube-ds"
	resource.Spec.Type = string(managementagentsdk.DataSourceTypesKubernetesCluster)
	resource.Status.OsokStatus.Ocid = "management-agent-1"
	resource.Status.Key = "kube-key"
	resource.Status.Name = "kube-ds"
	resource.Status.Type = string(managementagentsdk.DataSourceTypesKubernetesCluster)
	var deleteCalls int
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(_ context.Context, request managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			requireStringPtrValue(t, "GetDataSource managementAgentId", request.ManagementAgentId, "management-agent-1")
			requireStringPtrValue(t, "GetDataSource key", request.DataSourceKey, "kube-key")
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activeKubernetesClusterDataSource("kube-key", "kube-ds"),
			}, nil
		},
		deleteFn: func(_ context.Context, request managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			deleteCalls++
			requireStringPtrValue(t, "DeleteDataSource managementAgentId", request.ManagementAgentId, "management-agent-1")
			requireStringPtrValue(t, "DeleteDataSource key", request.DataSourceKey, "kube-key")
			return managementagentsdk.DeleteDataSourceResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained for delete work request")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDataSource calls = %d, want 1", deleteCalls)
	}
	current := requireDataSourceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncSourceWorkRequest)
	if current.WorkRequestID != "wr-delete" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete", current.WorkRequestID)
	}
}

func TestDataSourceDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		requireDataSourceDeleteWaitsForPendingWriteWorkRequest(
			t,
			shared.OSOKAsyncPhaseCreate,
			"wr-create",
			managementagentsdk.OperationTypesCreateDataSource,
		)
	})
	t.Run("update", func(t *testing.T) {
		requireDataSourceDeleteWaitsForPendingWriteWorkRequest(
			t,
			shared.OSOKAsyncPhaseUpdate,
			"wr-update",
			managementagentsdk.OperationTypesUpdateDataSource,
		)
	})
}

func requireDataSourceDeleteWaitsForPendingWriteWorkRequest(
	t *testing.T,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	operation managementagentsdk.OperationTypesEnum,
) {
	t.Helper()

	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		RawStatus:       string(managementagentsdk.OperationStatusAccepted),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	var workRequestCalls int
	client := newTestDataSourceRuntimeClient(t, pendingWriteDeleteDataSourceClient(t, &workRequestCalls, workRequestID, operation))

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained while write work request is pending")
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1", workRequestCalls)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status async current = nil, want pending write work request")
	}
	if current.Phase != phase || current.WorkRequestID != workRequestID {
		t.Fatalf("async current = %+v, want %s work request %s", current, phase, workRequestID)
	}
	if current.RawStatus != string(managementagentsdk.OperationStatusInProgress) ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async current = %+v, want pending IN_PROGRESS projection", current)
	}
}

func pendingWriteDeleteDataSourceClient(
	t *testing.T,
	workRequestCalls *int,
	workRequestID string,
	operation managementagentsdk.OperationTypesEnum,
) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
			(*workRequestCalls)++
			requireStringPtrValue(t, "GetWorkRequest id", request.WorkRequestId, workRequestID)
			percent := float32(50)
			return managementagentsdk.GetWorkRequestResponse{
				WorkRequest: managementagentsdk.WorkRequest{
					OperationType:   operation,
					Status:          managementagentsdk.OperationStatusInProgress,
					Id:              common.String(workRequestID),
					PercentComplete: &percent,
				},
			}, nil
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			t.Fatal("DeleteDataSource was called while a write work request is pending")
			return managementagentsdk.DeleteDataSourceResponse{}, nil
		},
	}
}

func TestDataSourceDeleteRechecksLifecyclePendingWriteWithoutWorkRequest(t *testing.T) {
	tests := []struct {
		name  string
		phase shared.OSOKAsyncPhase
		state managementagentsdk.LifecycleStatesEnum
	}{
		{
			name:  "create",
			phase: shared.OSOKAsyncPhaseCreate,
			state: managementagentsdk.LifecycleStatesCreating,
		},
		{
			name:  "update",
			phase: shared.OSOKAsyncPhaseUpdate,
			state: managementagentsdk.LifecycleStatesUpdating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireDataSourceDeleteRechecksLifecyclePendingWriteWithoutWorkRequest(t, tt.phase, tt.state)
		})
	}
}

func requireDataSourceDeleteRechecksLifecyclePendingWriteWithoutWorkRequest(
	t *testing.T,
	phase shared.OSOKAsyncPhase,
	state managementagentsdk.LifecycleStatesEnum,
) {
	t.Helper()

	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	seedLifecyclePendingDataSourceWrite(resource, phase, state)
	var getCalls int
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			getCalls++
			return managementagentsdk.GetDataSourceResponse{
				DataSource: prometheusDataSourceWithState(
					"datasource-key",
					"sample-ds",
					"http://prometheus.example",
					"oci_metrics",
					state,
				),
			}, nil
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			t.Fatal("DeleteDataSource was called while lifecycle write was still pending")
			return managementagentsdk.DeleteDataSourceResponse{}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained while lifecycle write is pending")
	}
	if getCalls != 1 {
		t.Fatalf("GetDataSource calls = %d, want 1", getCalls)
	}
	current := requireDataSourceAsyncCurrent(t, resource, phase, shared.OSOKAsyncSourceLifecycle)
	if current.WorkRequestID != "" {
		t.Fatalf("status.async.current.workRequestId = %q, want empty", current.WorkRequestID)
	}
	if current.RawStatus != string(state) || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %+v, want pending %s", current, state)
	}
}

func TestDataSourceDeleteStartsAfterLifecyclePendingWriteCompletesWithoutWorkRequest(t *testing.T) {
	tests := []struct {
		name  string
		phase shared.OSOKAsyncPhase
		state managementagentsdk.LifecycleStatesEnum
	}{
		{
			name:  "create",
			phase: shared.OSOKAsyncPhaseCreate,
			state: managementagentsdk.LifecycleStatesCreating,
		},
		{
			name:  "update",
			phase: shared.OSOKAsyncPhaseUpdate,
			state: managementagentsdk.LifecycleStatesUpdating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireDataSourceDeleteStartsAfterLifecyclePendingWriteCompletesWithoutWorkRequest(t, tt.phase, tt.state)
		})
	}
}

func requireDataSourceDeleteStartsAfterLifecyclePendingWriteCompletesWithoutWorkRequest(
	t *testing.T,
	phase shared.OSOKAsyncPhase,
	state managementagentsdk.LifecycleStatesEnum,
) {
	t.Helper()

	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	seedLifecyclePendingDataSourceWrite(resource, phase, state)
	var getCalledBeforeDelete bool
	var deleteCalls int
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			getCalledBeforeDelete = true
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://prometheus.example", "oci_metrics"),
			}, nil
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			deleteCalls++
			if !getCalledBeforeDelete {
				t.Fatal("DeleteDataSource was called before lifecycle readback")
			}
			return managementagentsdk.DeleteDataSourceResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want delete work request to retain finalizer")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDataSource calls = %d, want 1", deleteCalls)
	}
	current := requireDataSourceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncSourceWorkRequest)
	if current.WorkRequestID != "wr-delete" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete", current.WorkRequestID)
	}
}

func TestDataSourceDeleteLifecyclePendingWriteWithoutWorkRequestKeepsFinalizerOnAuthShapedRead(t *testing.T) {
	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	seedLifecyclePendingDataSourceWrite(resource, shared.OSOKAsyncPhaseCreate, managementagentsdk.LifecycleStatesCreating)
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{}, fakeDataSourceOCIError{
				HTTPStatusCode: 404,
				ErrorCode:      "NotAuthorizedOrNotFound",
				opcRequestID:   "opc-ambiguous",
			}
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			t.Fatal("DeleteDataSource was called after ambiguous lifecycle readback")
			return managementagentsdk.DeleteDataSourceResponse{}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete returned nil error; want ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete error = %q, want ambiguous NotAuthorizedOrNotFound", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-ambiguous" {
		t.Fatalf("status opcRequestId = %q, want opc-ambiguous", got)
	}
	requireDataSourceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncSourceLifecycle)
}

func TestDataSourceDeleteWorkRequestSucceededKeepsFinalizerOnAuthShapedConfirmRead(t *testing.T) {
	resource := newTestDataSource()
	seedTrackedDataSource(resource)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		RawStatus:       string(managementagentsdk.OperationStatusAccepted),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	fake := &fakeDataSourceRuntimeOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
			if got := stringValue(request.WorkRequestId); got != "wr-delete" {
				t.Fatalf("GetWorkRequest id = %q, want wr-delete", got)
			}
			percent := float32(100)
			return managementagentsdk.GetWorkRequestResponse{
				WorkRequest: managementagentsdk.WorkRequest{
					OperationType:   managementagentsdk.OperationTypesDeleteDataSource,
					Status:          managementagentsdk.OperationStatusSucceeded,
					Id:              common.String("wr-delete"),
					PercentComplete: &percent,
				},
			}, nil
		},
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{}, fakeDataSourceOCIError{
				HTTPStatusCode: 404,
				ErrorCode:      "NotAuthorizedOrNotFound",
				opcRequestID:   "opc-ambiguous",
			}
		},
		deleteFn: func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
			t.Fatal("DeleteDataSource was called after successful delete work request required confirmation")
			return managementagentsdk.DeleteDataSourceResponse{}, nil
		},
	}
	client := newTestDataSourceRuntimeClient(t, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete returned nil error; want ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete error = %q, want ambiguous NotAuthorizedOrNotFound", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-ambiguous" {
		t.Fatalf("status opcRequestId = %q, want opc-ambiguous", got)
	}
}

func newTestDataSourceRuntimeClient(t *testing.T, fake *fakeDataSourceRuntimeOCIClient) *dataSourceRuntimeClient {
	t.Helper()
	if fake == nil {
		fake = &fakeDataSourceRuntimeOCIClient{t: t}
	}
	return &dataSourceRuntimeClient{client: fake}
}

func pagedBindingDataSourceClient(t *testing.T, listCalls *int) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		listFn: func(_ context.Context, request managementagentsdk.ListDataSourcesRequest) (managementagentsdk.ListDataSourcesResponse, error) {
			return pagedBindingListResponse(t, listCalls, request)
		},
		getFn: func(_ context.Context, request managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			requireStringPtrValue(t, "GetDataSource key", request.DataSourceKey, "datasource-key")
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://prometheus.example", "oci_metrics"),
			}, nil
		},
	}
}

func pagedBindingListResponse(
	t *testing.T,
	listCalls *int,
	request managementagentsdk.ListDataSourcesRequest,
) (managementagentsdk.ListDataSourcesResponse, error) {
	t.Helper()
	(*listCalls)++
	requireStringPtrValue(t, "ListDataSources managementAgentId", request.ManagementAgentId, "management-agent-1")
	if len(request.Name) != 1 || request.Name[0] != "sample-ds" {
		t.Fatalf("ListDataSources name filter = %#v, want [sample-ds]", request.Name)
	}
	switch *listCalls {
	case 1:
		if request.Page != nil {
			t.Fatalf("first ListDataSources page = %q, want nil", stringValue(request.Page))
		}
		return managementagentsdk.ListDataSourcesResponse{
			Items:       []managementagentsdk.DataSourceSummary{prometheusSummary("other-key", "other")},
			OpcNextPage: common.String("page-2"),
		}, nil
	case 2:
		requireStringPtrValue(t, "second ListDataSources page", request.Page, "page-2")
		return managementagentsdk.ListDataSourcesResponse{
			Items: []managementagentsdk.DataSourceSummary{prometheusSummary("datasource-key", "sample-ds")},
		}, nil
	default:
		t.Fatalf("unexpected ListDataSources call %d", *listCalls)
		return managementagentsdk.ListDataSourcesResponse{}, nil
	}
}

func createDataSourceClient(t *testing.T, createCalls *int) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		listFn: func(context.Context, managementagentsdk.ListDataSourcesRequest) (managementagentsdk.ListDataSourcesResponse, error) {
			return managementagentsdk.ListDataSourcesResponse{}, nil
		},
		createFn: func(_ context.Context, request managementagentsdk.CreateDataSourceRequest) (managementagentsdk.CreateDataSourceResponse, error) {
			(*createCalls)++
			requireCreatePrometheusRequest(t, request)
			return managementagentsdk.CreateDataSourceResponse{
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
	}
}

func requireCreatePrometheusRequest(t *testing.T, request managementagentsdk.CreateDataSourceRequest) {
	t.Helper()
	requireStringPtrValue(t, "CreateDataSource managementAgentId", request.ManagementAgentId, "management-agent-1")
	requireStringPtrValue(t, "CreateDataSource retry token", request.OpcRetryToken, "uid-123")
	details, ok := request.CreateDataSourceDetails.(managementagentsdk.CreatePrometheusEmitterDataSourceDetails)
	if !ok {
		t.Fatalf("CreateDataSource details type = %T, want CreatePrometheusEmitterDataSourceDetails", request.CreateDataSourceDetails)
	}
	requireStringPtrValue(t, "CreateDataSource details.name", details.Name, "sample-ds")
	requireStringPtrValue(t, "CreateDataSource details.namespace", details.Namespace, "oci_metrics")
}

func updateDataSourceClient(
	t *testing.T,
	updateCalls *int,
	currentURL string,
	wantURL string,
) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(_ context.Context, request managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			requireStringPtrValue(t, "GetDataSource key", request.DataSourceKey, "datasource-key")
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", currentURL, "oci_metrics"),
			}, nil
		},
		updateFn: func(_ context.Context, request managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			(*updateCalls)++
			requireUpdatePrometheusRequest(t, request, wantURL)
			return managementagentsdk.UpdateDataSourceResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
	}
}

func requireUpdatePrometheusRequest(
	t *testing.T,
	request managementagentsdk.UpdateDataSourceRequest,
	wantURL string,
) managementagentsdk.UpdatePrometheusEmitterDataSourceDetails {
	t.Helper()
	requireStringPtrValue(t, "UpdateDataSource managementAgentId", request.ManagementAgentId, "management-agent-1")
	requireStringPtrValue(t, "UpdateDataSource key", request.DataSourceKey, "datasource-key")
	details, ok := request.UpdateDataSourceDetails.(managementagentsdk.UpdatePrometheusEmitterDataSourceDetails)
	if !ok {
		t.Fatalf("UpdateDataSource details type = %T, want UpdatePrometheusEmitterDataSourceDetails", request.UpdateDataSourceDetails)
	}
	requireStringPtrValue(t, "UpdateDataSource details.url", details.Url, wantURL)
	return details
}

func noUpdateForDefaultedOptionalDataSourceClient(t *testing.T) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSourceWithOptionals(),
			}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			t.Fatal("UpdateDataSource was called for omitted optional fields with server defaults")
			return managementagentsdk.UpdateDataSourceResponse{}, nil
		},
	}
}

func updateDataSourcePreservingOptionalsClient(t *testing.T, updateCalls *int) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSourceWithOptionals(),
			}, nil
		},
		updateFn: func(_ context.Context, request managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			(*updateCalls)++
			details := requireUpdatePrometheusRequest(t, request, "http://new.example")
			requirePrometheusUpdateOptionals(t, details, dataSourceUpdateOptionals{
				allowMetrics:             "up,down",
				proxyURL:                 "http://proxy.example",
				connectionTimeout:        100,
				readTimeout:              200,
				readDataLimitInKilobytes: 300,
				scheduleMins:             5,
				resourceGroup:            "resource-group",
				metricDimensions: []managementagentv1beta1.DataSourceMetricDimension{{
					Name:  "app",
					Value: "api",
				}},
			})
			return managementagentsdk.UpdateDataSourceResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
	}
}

func updateDataSourceExplicitOptionalsClient(t *testing.T, updateCalls *int) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://prometheus.example", "oci_metrics"),
			}, nil
		},
		updateFn: func(_ context.Context, request managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			(*updateCalls)++
			details := requireUpdatePrometheusRequest(t, request, "http://prometheus.example")
			requirePrometheusUpdateOptionals(t, details, dataSourceUpdateOptionals{
				allowMetrics:             "latency,requests",
				proxyURL:                 "http://new-proxy.example",
				connectionTimeout:        101,
				readTimeout:              202,
				readDataLimitInKilobytes: 303,
				scheduleMins:             6,
				resourceGroup:            "new-resource-group",
				metricDimensions: []managementagentv1beta1.DataSourceMetricDimension{{
					Name:  "service",
					Value: "api",
				}},
			})
			return managementagentsdk.UpdateDataSourceResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
	}
}

type dataSourceUpdateOptionals struct {
	allowMetrics             string
	proxyURL                 string
	connectionTimeout        int
	readTimeout              int
	readDataLimitInKilobytes int
	scheduleMins             int
	resourceGroup            string
	metricDimensions         []managementagentv1beta1.DataSourceMetricDimension
}

func requirePrometheusUpdateOptionals(
	t *testing.T,
	details managementagentsdk.UpdatePrometheusEmitterDataSourceDetails,
	want dataSourceUpdateOptionals,
) {
	t.Helper()
	requireStringPtrValue(t, "UpdateDataSource details.allowMetrics", details.AllowMetrics, want.allowMetrics)
	requireStringPtrValue(t, "UpdateDataSource details.proxyUrl", details.ProxyUrl, want.proxyURL)
	requireStringPtrValue(t, "UpdateDataSource details.resourceGroup", details.ResourceGroup, want.resourceGroup)
	requireIntPtrValue(t, "UpdateDataSource details.connectionTimeout", details.ConnectionTimeout, want.connectionTimeout)
	requireIntPtrValue(t, "UpdateDataSource details.readTimeout", details.ReadTimeout, want.readTimeout)
	requireIntPtrValue(t, "UpdateDataSource details.readDataLimitInKilobytes", details.ReadDataLimitInKilobytes, want.readDataLimitInKilobytes)
	requireIntPtrValue(t, "UpdateDataSource details.scheduleMins", details.ScheduleMins, want.scheduleMins)
	requireSDKMetricDimensions(t, "UpdateDataSource details.metricDimensions", details.MetricDimensions, want.metricDimensions)
}

func namespaceDriftDataSourceClient(t *testing.T) *fakeDataSourceRuntimeOCIClient {
	t.Helper()
	return &fakeDataSourceRuntimeOCIClient{
		t: t,
		getFn: func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
			return managementagentsdk.GetDataSourceResponse{
				DataSource: activePrometheusDataSource("datasource-key", "sample-ds", "http://old.example", "old_namespace"),
			}, nil
		},
		updateFn: func(context.Context, managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
			t.Fatal("UpdateDataSource was called for create-only namespace removal")
			return managementagentsdk.UpdateDataSourceResponse{}, nil
		},
	}
}

func newTestDataSource() *managementagentv1beta1.DataSource {
	return &managementagentv1beta1.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample",
			Namespace: "default",
			UID:       types.UID("uid-123"),
			Annotations: map[string]string{
				dataSourceManagementAgentIDAnnotation: "management-agent-1",
			},
		},
		Spec: managementagentv1beta1.DataSourceSpec{
			Name:          "sample-ds",
			CompartmentId: "compartment-1",
			Type:          string(managementagentsdk.DataSourceTypesPrometheusEmitter),
			Url:           "http://prometheus.example",
			Namespace:     "oci_metrics",
		},
	}
}

func seedTrackedDataSource(resource *managementagentv1beta1.DataSource) {
	resource.Status.OsokStatus.Ocid = "management-agent-1"
	resource.Status.Key = "datasource-key"
	resource.Status.Name = "sample-ds"
	resource.Status.Type = string(managementagentsdk.DataSourceTypesPrometheusEmitter)
}

func seedLifecyclePendingDataSourceWrite(
	resource *managementagentv1beta1.DataSource,
	phase shared.OSOKAsyncPhase,
	state managementagentsdk.LifecycleStatesEnum,
) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       string(state),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
}

func requireDataSourceAsyncCurrent(
	t *testing.T,
	resource *managementagentv1beta1.DataSource,
	phase shared.OSOKAsyncPhase,
	source shared.OSOKAsyncSource,
) *shared.OSOKAsyncOperation {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want async operation")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.Source != source {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, source)
	}
	return current
}

func activePrometheusDataSource(key string, name string, url string, namespace string) managementagentsdk.PrometheusEmitterDataSource {
	return prometheusDataSourceWithState(key, name, url, namespace, managementagentsdk.LifecycleStatesActive)
}

func activeKubernetesClusterDataSource(key string, name string) managementagentsdk.KubernetesClusterDataSource {
	return managementagentsdk.KubernetesClusterDataSource{
		Key:           common.String(key),
		Name:          common.String(name),
		CompartmentId: common.String("compartment-1"),
		Namespace:     common.String("kube-system"),
		State:         managementagentsdk.LifecycleStatesActive,
	}
}

func prometheusDataSourceWithState(
	key string,
	name string,
	url string,
	namespace string,
	state managementagentsdk.LifecycleStatesEnum,
) managementagentsdk.PrometheusEmitterDataSource {
	return managementagentsdk.PrometheusEmitterDataSource{
		Key:           common.String(key),
		Name:          common.String(name),
		CompartmentId: common.String("compartment-1"),
		State:         state,
		Url:           common.String(url),
		Namespace:     common.String(namespace),
	}
}

func activePrometheusDataSourceWithOptionals() managementagentsdk.PrometheusEmitterDataSource {
	current := activePrometheusDataSource("datasource-key", "sample-ds", "http://prometheus.example", "oci_metrics")
	current.AllowMetrics = common.String("up,down")
	current.ProxyUrl = common.String("http://proxy.example")
	current.ConnectionTimeout = common.Int(100)
	current.ReadTimeout = common.Int(200)
	current.ReadDataLimit = common.Int(300)
	current.ScheduleMins = common.Int(5)
	current.ResourceGroup = common.String("resource-group")
	current.MetricDimensions = []managementagentsdk.MetricDimension{{
		Name:  common.String("app"),
		Value: common.String("api"),
	}}
	return current
}

func prometheusSummary(key string, name string) managementagentsdk.PrometheusEmitterDataSourceSummary {
	return managementagentsdk.PrometheusEmitterDataSourceSummary{
		Key:  common.String(key),
		Name: common.String(name),
	}
}

func requireStringPtrValue(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if stringValue(got) != want {
		t.Fatalf("%s = %q, want %q", field, stringValue(got), want)
	}
}

func requireIntPtrValue(t *testing.T, field string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", field, *got, want)
	}
}

func requireSDKMetricDimensions(
	t *testing.T,
	field string,
	got []managementagentsdk.MetricDimension,
	want []managementagentv1beta1.DataSourceMetricDimension,
) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", field, len(got), len(want))
	}
	for i := range want {
		if stringValue(got[i].Name) != want[i].Name || stringValue(got[i].Value) != want[i].Value {
			t.Fatalf("%s[%d] = {%q, %q}, want {%q, %q}",
				field,
				i,
				stringValue(got[i].Name),
				stringValue(got[i].Value),
				want[i].Name,
				want[i].Value,
			)
		}
	}
}

type fakeDataSourceOCIError struct {
	HTTPStatusCode int
	ErrorCode      string
	opcRequestID   string
}

func (e fakeDataSourceOCIError) Error() string {
	return fmt.Sprintf("%d %s", e.HTTPStatusCode, e.ErrorCode)
}

func (e fakeDataSourceOCIError) GetOpcRequestID() string {
	return e.opcRequestID
}

type fakeDataSourceRuntimeOCIClient struct {
	t *testing.T

	createFn         func(context.Context, managementagentsdk.CreateDataSourceRequest) (managementagentsdk.CreateDataSourceResponse, error)
	getFn            func(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error)
	listFn           func(context.Context, managementagentsdk.ListDataSourcesRequest) (managementagentsdk.ListDataSourcesResponse, error)
	updateFn         func(context.Context, managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error)
	deleteFn         func(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error)
	getWorkRequestFn func(context.Context, managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error)
}

func (f *fakeDataSourceRuntimeOCIClient) CreateDataSource(ctx context.Context, request managementagentsdk.CreateDataSourceRequest) (managementagentsdk.CreateDataSourceResponse, error) {
	if f.createFn == nil {
		f.t.Fatal("unexpected CreateDataSource call")
		return managementagentsdk.CreateDataSourceResponse{}, nil
	}
	return f.createFn(ctx, request)
}

func (f *fakeDataSourceRuntimeOCIClient) GetDataSource(ctx context.Context, request managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error) {
	if f.getFn == nil {
		f.t.Fatal("unexpected GetDataSource call")
		return managementagentsdk.GetDataSourceResponse{}, nil
	}
	return f.getFn(ctx, request)
}

func (f *fakeDataSourceRuntimeOCIClient) ListDataSources(ctx context.Context, request managementagentsdk.ListDataSourcesRequest) (managementagentsdk.ListDataSourcesResponse, error) {
	if f.listFn == nil {
		f.t.Fatal("unexpected ListDataSources call")
		return managementagentsdk.ListDataSourcesResponse{}, nil
	}
	return f.listFn(ctx, request)
}

func (f *fakeDataSourceRuntimeOCIClient) UpdateDataSource(ctx context.Context, request managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error) {
	if f.updateFn == nil {
		f.t.Fatal("unexpected UpdateDataSource call")
		return managementagentsdk.UpdateDataSourceResponse{}, nil
	}
	return f.updateFn(ctx, request)
}

func (f *fakeDataSourceRuntimeOCIClient) DeleteDataSource(ctx context.Context, request managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error) {
	if f.deleteFn == nil {
		f.t.Fatal("unexpected DeleteDataSource call")
		return managementagentsdk.DeleteDataSourceResponse{}, nil
	}
	return f.deleteFn(ctx, request)
}

func (f *fakeDataSourceRuntimeOCIClient) GetWorkRequest(ctx context.Context, request managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn == nil {
		f.t.Fatal("unexpected GetWorkRequest call")
		return managementagentsdk.GetWorkRequestResponse{}, nil
	}
	return f.getWorkRequestFn(ctx, request)
}
