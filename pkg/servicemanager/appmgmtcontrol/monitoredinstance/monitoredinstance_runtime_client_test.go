/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoredinstance

import (
	"context"
	"testing"
	"time"

	appmgmtcontrolsdk "github.com/oracle/oci-go-sdk/v65/appmgmtcontrol"
	"github.com/oracle/oci-go-sdk/v65/common"
	appmgmtcontrolv1beta1 "github.com/oracle/oci-service-operator/api/appmgmtcontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testMonitoredInstanceID          = "ocid1.instance.oc1..example"
	testOtherMonitoredInstanceID     = "ocid1.instance.oc1..other"
	testMonitoredInstanceCompartment = "ocid1.compartment.oc1..example"
	testMonitoredInstanceName        = "monitored-instance-sample"
)

type fakeMonitoredInstanceOCIClient struct {
	getFn  func(context.Context, appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error)
	listFn func(context.Context, appmgmtcontrolsdk.ListMonitoredInstancesRequest) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error)

	getRequests  []appmgmtcontrolsdk.GetMonitoredInstanceRequest
	listRequests []appmgmtcontrolsdk.ListMonitoredInstancesRequest
}

func (f *fakeMonitoredInstanceOCIClient) GetMonitoredInstance(
	ctx context.Context,
	req appmgmtcontrolsdk.GetMonitoredInstanceRequest,
) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return appmgmtcontrolsdk.GetMonitoredInstanceResponse{}, nil
}

func (f *fakeMonitoredInstanceOCIClient) ListMonitoredInstances(
	ctx context.Context,
	req appmgmtcontrolsdk.ListMonitoredInstancesRequest,
) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, nil
}

func TestMonitoredInstanceRuntimeConfigHasNoCreateOrDeleteOperation(t *testing.T) {
	t.Parallel()

	config := newMonitoredInstanceRuntimeConfig(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, &fakeMonitoredInstanceOCIClient{})
	if config.Create != nil {
		t.Fatal("config.Create != nil, want bind-existing MonitoredInstance runtime without Create")
	}
	if config.Delete != nil {
		t.Fatal("config.Delete != nil, want CR-local unbind without Delete")
	}
	if len(config.List.Fields) != 3 {
		t.Fatalf("List fields = %#v, want focused bind fields plus pagination", config.List.Fields)
	}
	if config.Semantics.FinalizerPolicy != "none" {
		t.Fatalf("FinalizerPolicy = %q, want %q", config.Semantics.FinalizerPolicy, "none")
	}
}

func TestMonitoredInstanceCreateOrUpdateRejectsMissingBindingIdentity(t *testing.T) {
	t.Parallel()

	resource := newMonitoredInstanceResource()
	resource.Spec = appmgmtcontrolv1beta1.MonitoredInstanceSpec{}

	response, err := newTestMonitoredInstanceClient(&fakeMonitoredInstanceOCIClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		requestForMonitoredInstance(resource),
	)
	if err == nil || err.Error() != "MonitoredInstance bind-existing flow requires spec.instanceId or spec.compartmentId plus spec.displayName" {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit bind-existing validation", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestMonitoredInstanceCreateOrUpdateRejectsTrackedInstanceIDDrift(t *testing.T) {
	t.Parallel()

	resource := trackedMonitoredInstanceResource()
	resource.Spec.InstanceId = testOtherMonitoredInstanceID

	response, err := newTestMonitoredInstanceClient(&fakeMonitoredInstanceOCIClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		requestForMonitoredInstance(resource),
	)
	if err == nil || err.Error() != "MonitoredInstance formal semantics require replacement when instanceId changes" {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-only instanceId drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful replacement-only drift failure", response)
	}
}

func TestMonitoredInstanceCreateOrUpdateDirectBindDoesNotFallbackToList(t *testing.T) {
	t.Parallel()

	resource := newMonitoredInstanceResource()
	resource.Spec.InstanceId = testMonitoredInstanceID

	fake := &fakeMonitoredInstanceOCIClient{
		getFn: func(_ context.Context, req appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error) {
			requireStringPtr(t, "GetMonitoredInstanceRequest.MonitoredInstanceId", req.MonitoredInstanceId, testMonitoredInstanceID)
			return appmgmtcontrolsdk.GetMonitoredInstanceResponse{}, stubServiceError{statusCode: 404, code: "NotFound"}
		},
		listFn: func(context.Context, appmgmtcontrolsdk.ListMonitoredInstancesRequest) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error) {
			t.Fatal("ListMonitoredInstances() called, want explicit id bind failure without list fallback")
			return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, nil
		},
	}

	response, err := newTestMonitoredInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForMonitoredInstance(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want explicit id lookup failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful direct-bind failure", response)
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("List calls = %d, want 0 after explicit id miss", len(fake.listRequests))
	}
}

func TestMonitoredInstanceCreateOrUpdateBindsExistingMonitoredInstanceByPagedList(t *testing.T) {
	t.Parallel()

	resource := newMonitoredInstanceResource()
	fake := &fakeMonitoredInstanceOCIClient{
		listFn: func(_ context.Context, req appmgmtcontrolsdk.ListMonitoredInstancesRequest) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error) {
			requireStringPtr(t, "ListMonitoredInstancesRequest.CompartmentId", req.CompartmentId, testMonitoredInstanceCompartment)
			requireStringPtr(t, "ListMonitoredInstancesRequest.DisplayName", req.DisplayName, testMonitoredInstanceName)
			switch page := stringPtrValue(req.Page); page {
			case "":
				return appmgmtcontrolsdk.ListMonitoredInstancesResponse{
					MonitoredInstanceCollection: appmgmtcontrolsdk.MonitoredInstanceCollection{
						Items: []appmgmtcontrolsdk.MonitoredInstanceSummary{
							sdkMonitoredInstanceSummary(testOtherMonitoredInstanceID, "other-name", appmgmtcontrolsdk.MonitoredInstanceLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return appmgmtcontrolsdk.ListMonitoredInstancesResponse{
					MonitoredInstanceCollection: appmgmtcontrolsdk.MonitoredInstanceCollection{
						Items: []appmgmtcontrolsdk.MonitoredInstanceSummary{
							sdkMonitoredInstanceSummary(testMonitoredInstanceID, testMonitoredInstanceName, appmgmtcontrolsdk.MonitoredInstanceLifecycleStateActive),
						},
					},
				}, nil
			default:
				t.Fatalf("ListMonitoredInstancesRequest.Page = %q, want empty or page-2", page)
				return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error) {
			requireStringPtr(t, "GetMonitoredInstanceRequest.MonitoredInstanceId", req.MonitoredInstanceId, testMonitoredInstanceID)
			return appmgmtcontrolsdk.GetMonitoredInstanceResponse{
				MonitoredInstance: sdkMonitoredInstance(testMonitoredInstanceID, testMonitoredInstanceName, appmgmtcontrolsdk.MonitoredInstanceLifecycleStateActive),
			}, nil
		},
	}

	response, err := newTestMonitoredInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForMonitoredInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady observe", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListMonitoredInstances() calls = %d, want 2 paged list calls", len(fake.listRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetMonitoredInstance() calls = %d, want 1 follow-up get after list bind", len(fake.getRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testMonitoredInstanceID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testMonitoredInstanceID)
	}
	if got := resource.Status.InstanceId; got != testMonitoredInstanceID {
		t.Fatalf("status.instanceId = %q, want %q", got, testMonitoredInstanceID)
	}
	if got := resource.Status.TimeCreated; got != "2026-05-12T10:00:00Z" {
		t.Fatalf("status.timeCreated = %q, want full Get-backed timestamp", got)
	}
	if got := resource.Status.LifecycleDetails; got != "Observed by AppMgmt Control" {
		t.Fatalf("status.lifecycleDetails = %q, want full Get-backed lifecycle details", got)
	}
}

func TestMonitoredInstanceCreateOrUpdateTreatsInactiveAsSteadyObservation(t *testing.T) {
	t.Parallel()

	resource := trackedMonitoredInstanceResource()
	fake := &fakeMonitoredInstanceOCIClient{
		getFn: func(_ context.Context, req appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error) {
			requireStringPtr(t, "GetMonitoredInstanceRequest.MonitoredInstanceId", req.MonitoredInstanceId, testMonitoredInstanceID)
			return appmgmtcontrolsdk.GetMonitoredInstanceResponse{
				MonitoredInstance: sdkMonitoredInstance(testMonitoredInstanceID, testMonitoredInstanceName, appmgmtcontrolsdk.MonitoredInstanceLifecycleStateInactive),
			}, nil
		},
	}

	response, err := newTestMonitoredInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForMonitoredInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful inactive observe", response)
	}
	if got := resource.Status.LifecycleState; got != string(appmgmtcontrolsdk.MonitoredInstanceLifecycleStateInactive) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, appmgmtcontrolsdk.MonitoredInstanceLifecycleStateInactive)
	}
}

func TestMonitoredInstanceCreateOrUpdateRequeuesProvisioningLifecycle(t *testing.T) {
	t.Parallel()

	resource := trackedMonitoredInstanceResource()
	fake := &fakeMonitoredInstanceOCIClient{
		getFn: func(_ context.Context, req appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error) {
			requireStringPtr(t, "GetMonitoredInstanceRequest.MonitoredInstanceId", req.MonitoredInstanceId, testMonitoredInstanceID)
			return appmgmtcontrolsdk.GetMonitoredInstanceResponse{
				MonitoredInstance: sdkMonitoredInstance(testMonitoredInstanceID, testMonitoredInstanceName, appmgmtcontrolsdk.MonitoredInstanceLifecycleStateCreating),
			}, nil
		},
	}

	response, err := newTestMonitoredInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForMonitoredInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue during CREATING", response)
	}
}

func TestMonitoredInstanceDeleteIsCRLocalUnbind(t *testing.T) {
	t.Parallel()

	resource := trackedMonitoredInstanceResource()
	fake := &fakeMonitoredInstanceOCIClient{}

	deleted, err := newTestMonitoredInstanceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true for CR-local unbind")
	}
	if len(fake.getRequests) != 0 || len(fake.listRequests) != 0 {
		t.Fatalf("unexpected OCI calls during delete: get=%d list=%d", len(fake.getRequests), len(fake.listRequests))
	}
}

func newTestMonitoredInstanceClient(fake *fakeMonitoredInstanceOCIClient) MonitoredInstanceServiceClient {
	return newMonitoredInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		fake,
	)
}

func newMonitoredInstanceResource() *appmgmtcontrolv1beta1.MonitoredInstance {
	return &appmgmtcontrolv1beta1.MonitoredInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testMonitoredInstanceName,
			Namespace: "default",
		},
		Spec: appmgmtcontrolv1beta1.MonitoredInstanceSpec{
			CompartmentId: testMonitoredInstanceCompartment,
			DisplayName:   testMonitoredInstanceName,
		},
	}
}

func trackedMonitoredInstanceResource() *appmgmtcontrolv1beta1.MonitoredInstance {
	resource := newMonitoredInstanceResource()
	resource.Status.InstanceId = testMonitoredInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoredInstanceID)
	return resource
}

func requestForMonitoredInstance(resource *appmgmtcontrolv1beta1.MonitoredInstance) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      resource.Name,
			Namespace: resource.Namespace,
		},
	}
}

func sdkMonitoredInstance(
	id string,
	displayName string,
	lifecycle appmgmtcontrolsdk.MonitoredInstanceLifecycleStateEnum,
) appmgmtcontrolsdk.MonitoredInstance {
	created := common.SDKTime{Time: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 5, 12, 11, 0, 0, 0, time.UTC)}
	return appmgmtcontrolsdk.MonitoredInstance{
		InstanceId:        common.String(id),
		CompartmentId:     common.String(testMonitoredInstanceCompartment),
		DisplayName:       common.String(displayName),
		ManagementAgentId: common.String("ocid1.managementagent.oc1..example"),
		TimeCreated:       &created,
		TimeUpdated:       &updated,
		MonitoringState:   appmgmtcontrolsdk.MonitoredInstanceMonitoringStateEnabled,
		LifecycleState:    lifecycle,
		LifecycleDetails:  common.String("Observed by AppMgmt Control"),
	}
}

func sdkMonitoredInstanceSummary(
	id string,
	displayName string,
	lifecycle appmgmtcontrolsdk.MonitoredInstanceLifecycleStateEnum,
) appmgmtcontrolsdk.MonitoredInstanceSummary {
	return appmgmtcontrolsdk.MonitoredInstanceSummary{
		InstanceId:        common.String(id),
		CompartmentId:     common.String(testMonitoredInstanceCompartment),
		DisplayName:       common.String(displayName),
		ManagementAgentId: common.String("ocid1.managementagent.oc1..example"),
		MonitoringState:   appmgmtcontrolsdk.MonitoredInstanceMonitoringStateEnabled,
		LifecycleState:    lifecycle,
	}
}

func requireStringPtr(t *testing.T, field string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if got := *value; got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

type stubServiceError struct {
	statusCode int
	code       string
}

func (e stubServiceError) Error() string {
	return e.code
}

func (e stubServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e stubServiceError) GetMessage() string {
	return e.code
}

func (e stubServiceError) GetCode() string {
	return e.code
}

func (e stubServiceError) GetOpcRequestID() string {
	return ""
}

func (e stubServiceError) GetCause() error {
	return nil
}

func (e stubServiceError) GetSuggestions() []string {
	return nil
}

func (e stubServiceError) GetOperationName() string {
	return ""
}

func (e stubServiceError) GetTimestamp() string {
	return ""
}

var _ common.ServiceError = stubServiceError{}
