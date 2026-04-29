/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsloggroup

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLogAnalyticsNamespace       = "logan"
	testLogAnalyticsLogGroupID      = "ocid1.loganalyticsloggroup.oc1..group"
	testOtherLogAnalyticsLogGroupID = "ocid1.loganalyticsloggroup.oc1..other"
	testLogAnalyticsCompartmentID   = "ocid1.compartment.oc1..logan"
	testLogAnalyticsDisplayName     = "log-group-sample"
)

type fakeLogAnalyticsLogGroupOCIClient struct {
	createFn func(context.Context, loganalyticssdk.CreateLogAnalyticsLogGroupRequest) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error)
	getFn    func(context.Context, loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error)
	listFn   func(context.Context, loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error)
	updateFn func(context.Context, loganalyticssdk.UpdateLogAnalyticsLogGroupRequest) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error)
	deleteFn func(context.Context, loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error)
}

func (f *fakeLogAnalyticsLogGroupOCIClient) CreateLogAnalyticsLogGroup(
	ctx context.Context,
	request loganalyticssdk.CreateLogAnalyticsLogGroupRequest,
) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return loganalyticssdk.CreateLogAnalyticsLogGroupResponse{}, nil
}

func (f *fakeLogAnalyticsLogGroupOCIClient) GetLogAnalyticsLogGroup(
	ctx context.Context,
	request loganalyticssdk.GetLogAnalyticsLogGroupRequest,
) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return loganalyticssdk.GetLogAnalyticsLogGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "log group not found")
}

func (f *fakeLogAnalyticsLogGroupOCIClient) ListLogAnalyticsLogGroups(
	ctx context.Context,
	request loganalyticssdk.ListLogAnalyticsLogGroupsRequest,
) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return loganalyticssdk.ListLogAnalyticsLogGroupsResponse{}, nil
}

func (f *fakeLogAnalyticsLogGroupOCIClient) UpdateLogAnalyticsLogGroup(
	ctx context.Context,
	request loganalyticssdk.UpdateLogAnalyticsLogGroupRequest,
) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return loganalyticssdk.UpdateLogAnalyticsLogGroupResponse{}, nil
}

func (f *fakeLogAnalyticsLogGroupOCIClient) DeleteLogAnalyticsLogGroup(
	ctx context.Context,
	request loganalyticssdk.DeleteLogAnalyticsLogGroupRequest,
) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return loganalyticssdk.DeleteLogAnalyticsLogGroupResponse{}, nil
}

func testLogAnalyticsLogGroupClient(fake *fakeLogAnalyticsLogGroupOCIClient) LogAnalyticsLogGroupServiceClient {
	return newLogAnalyticsLogGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeLogAnalyticsLogGroupResource() *loganalyticsv1beta1.LogAnalyticsLogGroup {
	return &loganalyticsv1beta1.LogAnalyticsLogGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "log-group",
			Namespace: testLogAnalyticsNamespace,
			UID:       types.UID("log-group-uid"),
		},
		Spec: loganalyticsv1beta1.LogAnalyticsLogGroupSpec{
			DisplayName:   testLogAnalyticsDisplayName,
			CompartmentId: testLogAnalyticsCompartmentID,
			Description:   "initial description",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeLogAnalyticsLogGroupRequest(resource *loganalyticsv1beta1.LogAnalyticsLogGroup) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func makeSDKLogAnalyticsLogGroup(id string, spec loganalyticsv1beta1.LogAnalyticsLogGroupSpec) loganalyticssdk.LogAnalyticsLogGroup {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)}
	return loganalyticssdk.LogAnalyticsLogGroup{
		Id:            common.String(id),
		CompartmentId: common.String(spec.CompartmentId),
		DisplayName:   common.String(spec.DisplayName),
		Description:   common.String(spec.Description),
		TimeCreated:   &created,
		TimeUpdated:   &updated,
		FreeformTags:  cloneLogAnalyticsStringMap(spec.FreeformTags),
		DefinedTags:   makeSDKLogAnalyticsDefinedTags(spec.DefinedTags),
	}
}

func makeSDKLogAnalyticsLogGroupSummary(id string, spec loganalyticsv1beta1.LogAnalyticsLogGroupSpec) loganalyticssdk.LogAnalyticsLogGroupSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return loganalyticssdk.LogAnalyticsLogGroupSummary{
		Id:            common.String(id),
		CompartmentId: common.String(spec.CompartmentId),
		DisplayName:   common.String(spec.DisplayName),
		Description:   common.String(spec.Description),
		TimeCreated:   &created,
		FreeformTags:  cloneLogAnalyticsStringMap(spec.FreeformTags),
		DefinedTags:   makeSDKLogAnalyticsDefinedTags(spec.DefinedTags),
	}
}

func makeSDKLogAnalyticsDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func cloneLogAnalyticsStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func TestLogAnalyticsLogGroupRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newLogAnalyticsLogGroupRuntimeSemantics()
	if got == nil {
		t.Fatal("newLogAnalyticsLogGroupRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireLogAnalyticsStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	requireLogAnalyticsStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"})
	requireLogAnalyticsStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
}

func TestLogAnalyticsLogGroupServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	var createRequest loganalyticssdk.CreateLogAnalyticsLogGroupRequest
	var getRequest loganalyticssdk.GetLogAnalyticsLogGroupRequest
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		createFn: func(_ context.Context, request loganalyticssdk.CreateLogAnalyticsLogGroupRequest) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error) {
			createRequest = request
			return loganalyticssdk.CreateLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
				OpcRequestId:         common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getRequest = request
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLogAnalyticsLogGroupRequest(resource))
	requireLogAnalyticsSuccessfulResponse(t, response, err)
	requireLogAnalyticsCreateRequest(t, createRequest, resource)
	requireLogAnalyticsGetRequest(t, getRequest, testLogAnalyticsLogGroupID)
	if resource.Status.Id != testLogAnalyticsLogGroupID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testLogAnalyticsLogGroupID)
	}
	if string(resource.Status.OsokStatus.Ocid) != testLogAnalyticsLogGroupID {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testLogAnalyticsLogGroupID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestLogAnalyticsLogGroupServiceClientBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0
	var pages []string
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		listFn: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error) {
			listCalls++
			requireLogAnalyticsListRequest(t, request, resource)
			pages = append(pages, logAnalyticsStringPtrValue(request.Page))
			if listCalls == 1 {
				otherSpec := resource.Spec
				otherSpec.DisplayName = "other-log-group"
				return loganalyticssdk.ListLogAnalyticsLogGroupsResponse{
					LogAnalyticsLogGroupSummaryCollection: loganalyticssdk.LogAnalyticsLogGroupSummaryCollection{
						Items: []loganalyticssdk.LogAnalyticsLogGroupSummary{
							makeSDKLogAnalyticsLogGroupSummary(testOtherLogAnalyticsLogGroupID, otherSpec),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return loganalyticssdk.ListLogAnalyticsLogGroupsResponse{
				LogAnalyticsLogGroupSummaryCollection: loganalyticssdk.LogAnalyticsLogGroupSummaryCollection{
					Items: []loganalyticssdk.LogAnalyticsLogGroupSummary{
						makeSDKLogAnalyticsLogGroupSummary(testLogAnalyticsLogGroupID, resource.Spec),
					},
				},
			}, nil
		},
		createFn: func(context.Context, loganalyticssdk.CreateLogAnalyticsLogGroupRequest) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error) {
			createCalls++
			return loganalyticssdk.CreateLogAnalyticsLogGroupResponse{}, nil
		},
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			requireLogAnalyticsGetRequest(t, request, testLogAnalyticsLogGroupID)
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLogAnalyticsLogGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if createCalls != 0 {
		t.Fatalf("CreateLogAnalyticsLogGroup() calls = %d, want 0", createCalls)
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("list/get calls = %d/%d, want 2/1", listCalls, getCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	if string(resource.Status.OsokStatus.Ocid) != testLogAnalyticsLogGroupID {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testLogAnalyticsLogGroupID)
	}
	requireLogAnalyticsLastCondition(t, resource, shared.Active)
}

func TestLogAnalyticsLogGroupServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	getCalls := 0
	updateCalls := 0
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			requireLogAnalyticsGetRequest(t, request, testLogAnalyticsLogGroupID)
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
			}, nil
		},
		updateFn: func(context.Context, loganalyticssdk.UpdateLogAnalyticsLogGroupRequest) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error) {
			updateCalls++
			return loganalyticssdk.UpdateLogAnalyticsLogGroupResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLogAnalyticsLogGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("get/update calls = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireLogAnalyticsLastCondition(t, resource, shared.Active)
}

func TestLogAnalyticsLogGroupServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	resource.Spec.DisplayName = "updated-log-group"
	resource.Spec.Description = ""
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	getCalls := 0
	var updateRequest loganalyticssdk.UpdateLogAnalyticsLogGroupRequest
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			requireLogAnalyticsGetRequest(t, request, testLogAnalyticsLogGroupID)
			currentSpec := makeLogAnalyticsLogGroupResource().Spec
			if getCalls > 1 {
				currentSpec = resource.Spec
			}
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, currentSpec),
			}, nil
		},
		updateFn: func(_ context.Context, request loganalyticssdk.UpdateLogAnalyticsLogGroupRequest) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error) {
			updateRequest = request
			return loganalyticssdk.UpdateLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
				OpcRequestId:         common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLogAnalyticsLogGroupRequest(resource))
	requireLogAnalyticsSuccessfulResponse(t, response, err)
	requireLogAnalyticsUpdateRequest(t, updateRequest, resource)
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
	if resource.Status.Description != "" {
		t.Fatalf("status.description = %q, want cleared description", resource.Status.Description)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestLogAnalyticsLogGroupServiceClientRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	updateCalls := 0
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			requireLogAnalyticsGetRequest(t, request, testLogAnalyticsLogGroupID)
			currentSpec := makeLogAnalyticsLogGroupResource().Spec
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, currentSpec),
			}, nil
		},
		updateFn: func(context.Context, loganalyticssdk.UpdateLogAnalyticsLogGroupRequest) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error) {
			updateCalls++
			return loganalyticssdk.UpdateLogAnalyticsLogGroupResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLogAnalyticsLogGroupRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateLogAnalyticsLogGroup() calls = %d, want 0", updateCalls)
	}
	requireLogAnalyticsLastCondition(t, resource, shared.Failed)
}

func TestLogAnalyticsLogGroupServiceClientDeleteKeepsFinalizerUntilReadbackIsGone(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.Id = testLogAnalyticsLogGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	getCalls := 0
	deleteCalls := 0
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			requireLogAnalyticsGetRequest(t, request, testLogAnalyticsLogGroupID)
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
				LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
			}, nil
		},
		deleteFn: func(_ context.Context, request loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error) {
			deleteCalls++
			requireLogAnalyticsDeleteRequest(t, request, testLogAnalyticsLogGroupID)
			return loganalyticssdk.DeleteLogAnalyticsLogGroupResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still finds log group")
	}
	if getCalls != 3 || deleteCalls != 1 {
		t.Fatalf("get/delete calls = %d/%d, want 3/1", getCalls, deleteCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestLogAnalyticsLogGroupServiceClientDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.Id = testLogAnalyticsLogGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	getCalls := 0
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			if getCalls <= 2 {
				return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
					LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
				}, nil
			}
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(context.Context, loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error) {
			return loganalyticssdk.DeleteLogAnalyticsLogGroupResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed deletion")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestLogAnalyticsLogGroupServiceClientDeleteTreatsAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.Id = testLogAnalyticsLogGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	getCalls := 0
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			if getCalls <= 2 {
				return loganalyticssdk.GetLogAnalyticsLogGroupResponse{
					LogAnalyticsLogGroup: makeSDKLogAnalyticsLogGroup(testLogAnalyticsLogGroupID, resource.Spec),
				}, nil
			}
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error) {
			return loganalyticssdk.DeleteLogAnalyticsLogGroupResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want surfaced auth error request ID", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestLogAnalyticsLogGroupServiceClientDeleteFailsFastOnPreDeleteAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	resource.Status.Id = testLogAnalyticsLogGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testLogAnalyticsLogGroupID)
	getCalls := 0
	deleteCalls := 0
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
			getCalls++
			return loganalyticssdk.GetLogAnalyticsLogGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error) {
			deleteCalls++
			return loganalyticssdk.DeleteLogAnalyticsLogGroupResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want pre-delete auth-shaped 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if getCalls != 1 || deleteCalls != 0 {
		t.Fatalf("get/delete calls = %d/%d, want 1/0", getCalls, deleteCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want surfaced auth error request ID", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestLogAnalyticsLogGroupServiceClientCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsLogGroupResource()
	client := testLogAnalyticsLogGroupClient(&fakeLogAnalyticsLogGroupOCIClient{
		createFn: func(context.Context, loganalyticssdk.CreateLogAnalyticsLogGroupRequest) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error) {
			return loganalyticssdk.CreateLogAnalyticsLogGroupResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeLogAnalyticsLogGroupRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want error request ID", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLogAnalyticsLastCondition(t, resource, shared.Failed)
}

func requireLogAnalyticsCreateRequest(
	t *testing.T,
	request loganalyticssdk.CreateLogAnalyticsLogGroupRequest,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
) {
	t.Helper()
	requireLogAnalyticsStringPtr(t, "create namespaceName", request.NamespaceName, testLogAnalyticsNamespace)
	if request.DisplayName == nil || *request.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %v, want %q", request.DisplayName, resource.Spec.DisplayName)
	}
	if request.CompartmentId == nil || *request.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", request.CompartmentId, resource.Spec.CompartmentId)
	}
	if request.OpcRetryToken == nil || *request.OpcRetryToken == "" {
		t.Fatal("create opcRetryToken is empty, want deterministic retry token")
	}
}

func requireLogAnalyticsListRequest(
	t *testing.T,
	request loganalyticssdk.ListLogAnalyticsLogGroupsRequest,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
) {
	t.Helper()
	requireLogAnalyticsStringPtr(t, "list namespaceName", request.NamespaceName, testLogAnalyticsNamespace)
	requireLogAnalyticsStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireLogAnalyticsStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
}

func requireLogAnalyticsGetRequest(t *testing.T, request loganalyticssdk.GetLogAnalyticsLogGroupRequest, id string) {
	t.Helper()
	requireLogAnalyticsStringPtr(t, "get namespaceName", request.NamespaceName, testLogAnalyticsNamespace)
	requireLogAnalyticsStringPtr(t, "get logAnalyticsLogGroupId", request.LogAnalyticsLogGroupId, id)
}

func requireLogAnalyticsUpdateRequest(
	t *testing.T,
	request loganalyticssdk.UpdateLogAnalyticsLogGroupRequest,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
) {
	t.Helper()
	requireLogAnalyticsStringPtr(t, "update namespaceName", request.NamespaceName, testLogAnalyticsNamespace)
	requireLogAnalyticsStringPtr(t, "update logAnalyticsLogGroupId", request.LogAnalyticsLogGroupId, testLogAnalyticsLogGroupID)
	requireLogAnalyticsStringPtr(t, "update displayName", request.DisplayName, resource.Spec.DisplayName)
	if request.Description == nil || *request.Description != "" {
		t.Fatalf("update description = %v, want explicit empty string", request.Description)
	}
	if !reflect.DeepEqual(request.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", request.FreeformTags, resource.Spec.FreeformTags)
	}
	if !reflect.DeepEqual(request.DefinedTags, makeSDKLogAnalyticsDefinedTags(resource.Spec.DefinedTags)) {
		t.Fatalf("update definedTags = %#v, want desired tags", request.DefinedTags)
	}
}

func requireLogAnalyticsDeleteRequest(t *testing.T, request loganalyticssdk.DeleteLogAnalyticsLogGroupRequest, id string) {
	t.Helper()
	requireLogAnalyticsStringPtr(t, "delete namespaceName", request.NamespaceName, testLogAnalyticsNamespace)
	requireLogAnalyticsStringPtr(t, "delete logAnalyticsLogGroupId", request.LogAnalyticsLogGroupId, id)
}

func requireLogAnalyticsStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func logAnalyticsStringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func requireLogAnalyticsSuccessfulResponse(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful", response)
	}
}

func requireLogAnalyticsLastCondition(
	t *testing.T,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireLogAnalyticsStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}
