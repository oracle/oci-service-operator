/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedinstance

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	wlmssdk "github.com/oracle/oci-go-sdk/v65/wlms"
	wlmsv1beta1 "github.com/oracle/oci-service-operator/api/wlms/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testManagedInstanceID          = "ocid1.managedinstance.oc1..example"
	testOtherManagedInstanceID     = "ocid1.managedinstance.oc1..other"
	testManagedInstanceCompartment = "ocid1.compartment.oc1..example"
	testManagedInstanceName        = "managedinstance-sample"
	testManagedInstanceHostName    = "managedinstance.example.internal"
)

type fakeManagedInstanceOCIClient struct {
	getFn    func(context.Context, wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error)
	listFn   func(context.Context, wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error)
	updateFn func(context.Context, wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error)

	getRequests    []wlmssdk.GetManagedInstanceRequest
	listRequests   []wlmssdk.ListManagedInstancesRequest
	updateRequests []wlmssdk.UpdateManagedInstanceRequest
}

func (f *fakeManagedInstanceOCIClient) GetManagedInstance(
	ctx context.Context,
	req wlmssdk.GetManagedInstanceRequest,
) (wlmssdk.GetManagedInstanceResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return wlmssdk.GetManagedInstanceResponse{}, nil
}

func (f *fakeManagedInstanceOCIClient) ListManagedInstances(
	ctx context.Context,
	req wlmssdk.ListManagedInstancesRequest,
) (wlmssdk.ListManagedInstancesResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return wlmssdk.ListManagedInstancesResponse{}, nil
}

func (f *fakeManagedInstanceOCIClient) UpdateManagedInstance(
	ctx context.Context,
	req wlmssdk.UpdateManagedInstanceRequest,
) (wlmssdk.UpdateManagedInstanceResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return wlmssdk.UpdateManagedInstanceResponse{}, nil
}

func TestManagedInstanceRuntimeConfigHasNoCreateOrDeleteOperation(t *testing.T) {
	t.Parallel()

	config := newManagedInstanceRuntimeConfig(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, &fakeManagedInstanceOCIClient{})
	if config.Create != nil {
		t.Fatal("config.Create != nil, want bind-existing ManagedInstance runtime without Create")
	}
	if config.Delete != nil {
		t.Fatal("config.Delete != nil, want CR-local unbind without Delete")
	}
	if len(config.List.Fields) != 5 {
		t.Fatalf("List fields = %#v, want focused bind fields plus pagination", config.List.Fields)
	}
	if config.Semantics.FinalizerPolicy != "none" {
		t.Fatalf("FinalizerPolicy = %q, want %q", config.Semantics.FinalizerPolicy, "none")
	}
}

func TestManagedInstanceCreateOrUpdateRejectsMissingBindingIdentity(t *testing.T) {
	t.Parallel()

	resource := newManagedInstanceResource()
	resource.Spec = wlmsv1beta1.ManagedInstanceSpec{}

	response, err := newTestManagedInstanceClient(&fakeManagedInstanceOCIClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		requestForManagedInstance(resource),
	)
	if err == nil || err.Error() != "ManagedInstance bind-existing flow requires spec.id or spec.compartmentId plus spec.displayName" {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit bind-existing validation", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestManagedInstanceCreateOrUpdateDirectBindDoesNotFallbackToList(t *testing.T) {
	t.Parallel()

	resource := newManagedInstanceResource()
	resource.Spec.Id = testManagedInstanceID

	fake := &fakeManagedInstanceOCIClient{
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			requireStringPtr(t, "GetManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			return wlmssdk.GetManagedInstanceResponse{}, stubServiceError{statusCode: 404, code: "NotFound"}
		},
		listFn: func(context.Context, wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error) {
			t.Fatal("ListManagedInstances() called, want explicit id bind failure without list fallback")
			return wlmssdk.ListManagedInstancesResponse{}, nil
		},
	}

	response, err := newTestManagedInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
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

func TestManagedInstanceCreateOrUpdateBindsExistingManagedInstanceByPagedList(t *testing.T) {
	t.Parallel()

	resource := newManagedInstanceResource()
	resource.Spec.Id = ""

	fake := &fakeManagedInstanceOCIClient{
		listFn: func(_ context.Context, req wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error) {
			requireStringPtr(t, "ListManagedInstancesRequest.CompartmentId", req.CompartmentId, testManagedInstanceCompartment)
			requireStringPtr(t, "ListManagedInstancesRequest.DisplayName", req.DisplayName, testManagedInstanceName)
			if got := string(req.PluginStatus); got != resource.Spec.PluginStatus {
				t.Fatalf("ListManagedInstancesRequest.PluginStatus = %q, want %q", got, resource.Spec.PluginStatus)
			}
			if req.Limit != nil {
				t.Fatalf("ListManagedInstancesRequest.Limit = %#v, want nil", req.Limit)
			}
			if req.SortOrder != "" {
				t.Fatalf("ListManagedInstancesRequest.SortOrder = %q, want empty", req.SortOrder)
			}
			if req.SortBy != "" {
				t.Fatalf("ListManagedInstancesRequest.SortBy = %q, want empty", req.SortBy)
			}
			switch page := stringPtrValue(req.Page); page {
			case "":
				return wlmssdk.ListManagedInstancesResponse{
					ManagedInstanceCollection: wlmssdk.ManagedInstanceCollection{
						Items: []wlmssdk.ManagedInstanceSummary{
							managedInstanceSummary(testOtherManagedInstanceID, "other", string(wlmssdk.ListManagedInstancesPluginStatusInactive)),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return wlmssdk.ListManagedInstancesResponse{
					ManagedInstanceCollection: wlmssdk.ManagedInstanceCollection{
						Items: []wlmssdk.ManagedInstanceSummary{
							managedInstanceSummary(testManagedInstanceID, testManagedInstanceName, resource.Spec.PluginStatus),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", page)
				return wlmssdk.ListManagedInstancesResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			requireStringPtr(t, "GetManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			return wlmssdk.GetManagedInstanceResponse{
				ManagedInstance: activeManagedInstanceWithConfigSDK(
					testManagedInstanceID,
					6,
					[]string{"/u01/oracle/user_projects/domains"},
				),
			}, nil
		},
	}

	response, err := newTestManagedInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("List calls = %d, want 2 paged lookups", len(fake.listRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("Get calls = %d, want 1 live read after list bind", len(fake.getRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want no update on matching bind", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testManagedInstanceID {
		t.Fatalf("status.ocid = %q, want %q", got, testManagedInstanceID)
	}
	if got := resource.Status.HostName; got != testManagedInstanceHostName {
		t.Fatalf("status.hostName = %q, want %q", got, testManagedInstanceHostName)
	}
}

func TestManagedInstanceCreateOrUpdateClearsStaleTrackedIDBeforeListRebindUpdate(t *testing.T) {
	t.Parallel()

	resource := trackedManagedInstanceResource()
	resource.Spec.Id = ""
	resource.Spec.Configuration = map[string]shared.JSONValue{
		"discoveryInterval": jsonValue("0"),
		"domainSearchPaths": jsonValue("[]"),
	}
	resource.Status.Id = testOtherManagedInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOtherManagedInstanceID)

	updateApplied := false
	fake := &fakeManagedInstanceOCIClient{
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			switch managedInstanceID := stringPtrValue(req.ManagedInstanceId); managedInstanceID {
			case testOtherManagedInstanceID:
				return wlmssdk.GetManagedInstanceResponse{}, stubServiceError{statusCode: 404, code: "NotFound"}
			case testManagedInstanceID:
				if updateApplied {
					return wlmssdk.GetManagedInstanceResponse{
						ManagedInstance: activeManagedInstanceWithConfigSDK(testManagedInstanceID, 0, []string{}),
					}, nil
				}
				return wlmssdk.GetManagedInstanceResponse{
					ManagedInstance: activeManagedInstanceWithConfigSDK(
						testManagedInstanceID,
						6,
						[]string{"/u01/oracle/old"},
					),
				}, nil
			default:
				t.Fatalf("unexpected GetManagedInstanceRequest.ManagedInstanceId %q", managedInstanceID)
				return wlmssdk.GetManagedInstanceResponse{}, nil
			}
		},
		listFn: func(_ context.Context, req wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error) {
			requireStringPtr(t, "ListManagedInstancesRequest.CompartmentId", req.CompartmentId, testManagedInstanceCompartment)
			requireStringPtr(t, "ListManagedInstancesRequest.DisplayName", req.DisplayName, testManagedInstanceName)
			if req.Id != nil {
				t.Fatalf("ListManagedInstancesRequest.Id = %#v, want nil after stale tracked identity clear", req.Id)
			}
			if got := string(req.PluginStatus); got != resource.Spec.PluginStatus {
				t.Fatalf("ListManagedInstancesRequest.PluginStatus = %q, want %q", got, resource.Spec.PluginStatus)
			}
			return wlmssdk.ListManagedInstancesResponse{
				ManagedInstanceCollection: wlmssdk.ManagedInstanceCollection{
					Items: []wlmssdk.ManagedInstanceSummary{
						managedInstanceSummary(testManagedInstanceID, testManagedInstanceName, resource.Spec.PluginStatus),
					},
				},
			}, nil
		},
		updateFn: func(_ context.Context, req wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error) {
			requireStringPtr(t, "UpdateManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			if req.Configuration == nil {
				t.Fatal("UpdateManagedInstanceRequest.Configuration = nil, want explicit configuration update after stale tracked-ID rebind")
			}
			if req.Configuration.DiscoveryInterval == nil || *req.Configuration.DiscoveryInterval != 0 {
				t.Fatalf("Configuration.DiscoveryInterval = %#v, want 0", req.Configuration.DiscoveryInterval)
			}
			if req.Configuration.DomainSearchPaths == nil || len(req.Configuration.DomainSearchPaths) != 0 {
				t.Fatalf("Configuration.DomainSearchPaths = %#v, want explicit empty slice", req.Configuration.DomainSearchPaths)
			}
			updateApplied = true
			return wlmssdk.UpdateManagedInstanceResponse{
				ManagedInstance: activeManagedInstanceWithConfigSDK(testManagedInstanceID, 0, []string{}),
				OpcRequestId:    common.String("opc-update"),
			}, nil
		},
	}

	response, err := newTestManagedInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want bounded confirmation requeue after stale tracked-ID rebind update", response)
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("Get calls = %d, want stale tracked-ID preflight plus live Get on rebound resource", len(fake.getRequests))
	}
	requireStringPtr(t, "first GetManagedInstanceRequest.ManagedInstanceId", fake.getRequests[0].ManagedInstanceId, testOtherManagedInstanceID)
	requireStringPtr(t, "second GetManagedInstanceRequest.ManagedInstanceId", fake.getRequests[1].ManagedInstanceId, testManagedInstanceID)
	if len(fake.listRequests) != 1 {
		t.Fatalf("List calls = %d, want 1 rebind lookup after stale tracked-ID clear", len(fake.listRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls = %d, want 1 update after rebound GetManagedInstance", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testManagedInstanceID {
		t.Fatalf("status.ocid = %q, want rebound id %q", got, testManagedInstanceID)
	}
	if got := resource.Status.Id; got != testManagedInstanceID {
		t.Fatalf("status.id = %q, want rebound id %q", got, testManagedInstanceID)
	}
	if resource.Status.Configuration.DiscoveryInterval != 0 {
		t.Fatalf("status.configuration.discoveryInterval = %d, want 0 after rebound update", resource.Status.Configuration.DiscoveryInterval)
	}
	if len(resource.Status.Configuration.DomainSearchPaths) != 0 {
		t.Fatalf("status.configuration.domainSearchPaths = %#v, want empty after rebound update", resource.Status.Configuration.DomainSearchPaths)
	}
}

func TestManagedInstanceCreateOrUpdateRejectsForceNewIDDraftOnTrackedResource(t *testing.T) {
	t.Parallel()

	resource := trackedManagedInstanceResource()
	resource.Spec.Id = testOtherManagedInstanceID

	fake := &fakeManagedInstanceOCIClient{
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			requireStringPtr(t, "GetManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			return wlmssdk.GetManagedInstanceResponse{
				ManagedInstance: activeManagedInstanceWithConfigSDK(
					testManagedInstanceID,
					6,
					[]string{"/u01/oracle/user_projects/domains"},
				),
			}, nil
		},
		updateFn: func(context.Context, wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error) {
			t.Fatal("UpdateManagedInstance should not be called after force-new id drift")
			return wlmssdk.UpdateManagedInstanceResponse{}, nil
		},
	}

	response, err := newTestManagedInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err == nil || !strings.Contains(err.Error(), "require replacement when id changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new id drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("Get calls = %d, want 1 on the currently tracked resource", len(fake.getRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0 after force-new id drift", len(fake.updateRequests))
	}
}

func TestManagedInstanceCreateOrUpdateRebindsStaleTrackedIDWithoutConfigurationUpdate(t *testing.T) {
	t.Parallel()

	resource := trackedManagedInstanceResource()
	resource.Spec.Id = ""
	resource.Spec.Configuration = nil
	resource.Status.Id = testOtherManagedInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOtherManagedInstanceID)

	fake := &fakeManagedInstanceOCIClient{
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			requireStringPtr(t, "GetManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testOtherManagedInstanceID)
			return wlmssdk.GetManagedInstanceResponse{}, stubServiceError{statusCode: 404, code: "NotFound"}
		},
		listFn: func(_ context.Context, req wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error) {
			requireStringPtr(t, "ListManagedInstancesRequest.CompartmentId", req.CompartmentId, testManagedInstanceCompartment)
			requireStringPtr(t, "ListManagedInstancesRequest.DisplayName", req.DisplayName, testManagedInstanceName)
			if req.Id != nil {
				t.Fatalf("ListManagedInstancesRequest.Id = %#v, want nil after stale tracked identity clear", req.Id)
			}
			if got := string(req.PluginStatus); got != resource.Spec.PluginStatus {
				t.Fatalf("ListManagedInstancesRequest.PluginStatus = %q, want %q", got, resource.Spec.PluginStatus)
			}
			return wlmssdk.ListManagedInstancesResponse{
				ManagedInstanceCollection: wlmssdk.ManagedInstanceCollection{
					Items: []wlmssdk.ManagedInstanceSummary{
						managedInstanceSummary(testManagedInstanceID, testManagedInstanceName, resource.Spec.PluginStatus),
					},
				},
			}, nil
		},
		updateFn: func(context.Context, wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error) {
			t.Fatal("UpdateManagedInstance should not be called when no configuration update is requested")
			return wlmssdk.UpdateManagedInstanceResponse{}, nil
		},
	}

	response, err := newTestManagedInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want successful stale tracked-ID rebind without configuration drift", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful observe after stale tracked-ID rebind", response)
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("Get calls = %d, want 1 stale tracked-ID probe", len(fake.getRequests))
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("List calls = %d, want 1 rebind lookup after stale tracked-ID clear", len(fake.listRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0 when no configuration update is requested", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testManagedInstanceID {
		t.Fatalf("status.ocid = %q, want rebound id %q", got, testManagedInstanceID)
	}
	if got := resource.Status.Id; got != testManagedInstanceID {
		t.Fatalf("status.id = %q, want rebound id %q", got, testManagedInstanceID)
	}
	if got := resource.Status.HostName; got != testManagedInstanceHostName {
		t.Fatalf("status.hostName = %q, want %q", got, testManagedInstanceHostName)
	}
}

func TestManagedInstanceCreateOrUpdateUsesBoundedConfirmationPassForUpdate(t *testing.T) {
	t.Parallel()

	resource := trackedManagedInstanceResource()
	resource.Spec.Configuration = map[string]shared.JSONValue{
		"discoveryInterval": jsonValue("0"),
		"domainSearchPaths": jsonValue("[]"),
	}

	updateApplied := false
	fake := &fakeManagedInstanceOCIClient{
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			requireStringPtr(t, "GetManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			if updateApplied {
				return wlmssdk.GetManagedInstanceResponse{
					ManagedInstance: activeManagedInstanceWithConfigSDK(testManagedInstanceID, 0, []string{}),
				}, nil
			}
			return wlmssdk.GetManagedInstanceResponse{
				ManagedInstance: activeManagedInstanceWithConfigSDK(
					testManagedInstanceID,
					6,
					[]string{"/u01/oracle/old"},
				),
			}, nil
		},
		updateFn: func(_ context.Context, req wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error) {
			requireStringPtr(t, "UpdateManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			if req.Configuration == nil {
				t.Fatal("UpdateManagedInstanceRequest.Configuration = nil, want explicit configuration update")
			}
			if req.Configuration.DiscoveryInterval == nil || *req.Configuration.DiscoveryInterval != 0 {
				t.Fatalf("Configuration.DiscoveryInterval = %#v, want 0", req.Configuration.DiscoveryInterval)
			}
			if req.Configuration.DomainSearchPaths == nil || len(req.Configuration.DomainSearchPaths) != 0 {
				t.Fatalf("Configuration.DomainSearchPaths = %#v, want explicit empty slice", req.Configuration.DomainSearchPaths)
			}
			updateApplied = true
			return wlmssdk.UpdateManagedInstanceResponse{
				ManagedInstance: activeManagedInstanceWithConfigSDK(testManagedInstanceID, 0, []string{}),
				OpcRequestId:    common.String("opc-update"),
			}, nil
		},
	}

	client := newTestManagedInstanceClient(fake)

	firstResponse, err := client.CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err != nil {
		t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	if !firstResponse.IsSuccessful || !firstResponse.ShouldRequeue {
		t.Fatalf("first CreateOrUpdate() response = %#v, want bounded confirmation requeue", firstResponse)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Updating) {
		t.Fatalf("status.reason after first pass = %q, want %q", resource.Status.OsokStatus.Reason, shared.Updating)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls after first pass = %d, want 1", len(fake.updateRequests))
	}
	if resource.Status.Configuration.DiscoveryInterval != 0 {
		t.Fatalf("status.configuration.discoveryInterval = %d, want 0 after update", resource.Status.Configuration.DiscoveryInterval)
	}
	if len(resource.Status.Configuration.DomainSearchPaths) != 0 {
		t.Fatalf("status.configuration.domainSearchPaths = %#v, want empty after update", resource.Status.Configuration.DomainSearchPaths)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update")
	}

	secondResponse, err := client.CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !secondResponse.IsSuccessful || secondResponse.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want settled active state", secondResponse)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.reason after confirmation = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("Get calls = %d, want 2", len(fake.getRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls after confirmation = %d, want 1", len(fake.updateRequests))
	}
}

func TestManagedInstanceCreateOrUpdateSkipsUpdateWhenPartialConfigurationAlreadyMatches(t *testing.T) {
	t.Parallel()

	resource := trackedManagedInstanceResource()
	resource.Spec.Configuration = map[string]shared.JSONValue{
		"discoveryInterval": jsonValue("6"),
	}

	fake := &fakeManagedInstanceOCIClient{
		getFn: func(_ context.Context, req wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
			requireStringPtr(t, "GetManagedInstanceRequest.ManagedInstanceId", req.ManagedInstanceId, testManagedInstanceID)
			return wlmssdk.GetManagedInstanceResponse{
				ManagedInstance: activeManagedInstanceWithConfigSDK(
					testManagedInstanceID,
					6,
					[]string{"/u01/oracle/old"},
				),
			}, nil
		},
		updateFn: func(context.Context, wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error) {
			t.Fatal("UpdateManagedInstance should not be called when the requested partial configuration already matches OCI")
			return wlmssdk.UpdateManagedInstanceResponse{}, nil
		},
	}

	response, err := newTestManagedInstanceClient(fake).CreateOrUpdate(context.Background(), resource, requestForManagedInstance(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op observe", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestManagedInstanceDeleteIsCRLocalUnbind(t *testing.T) {
	t.Parallel()

	resource := trackedManagedInstanceResource()
	fake := &fakeManagedInstanceOCIClient{}

	deleted, err := newTestManagedInstanceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true for CR-local unbind")
	}
	if len(fake.getRequests) != 0 || len(fake.listRequests) != 0 || len(fake.updateRequests) != 0 {
		t.Fatalf("unexpected OCI calls during delete: get=%d list=%d update=%d", len(fake.getRequests), len(fake.listRequests), len(fake.updateRequests))
	}
}

func newTestManagedInstanceClient(fake *fakeManagedInstanceOCIClient) ManagedInstanceServiceClient {
	return newManagedInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		fake,
	)
}

func newManagedInstanceResource() *wlmsv1beta1.ManagedInstance {
	return &wlmsv1beta1.ManagedInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testManagedInstanceName,
			Namespace: "default",
		},
		Spec: wlmsv1beta1.ManagedInstanceSpec{
			CompartmentId: testManagedInstanceCompartment,
			DisplayName:   testManagedInstanceName,
			PluginStatus:  string(wlmssdk.ListManagedInstancesPluginStatusActive),
		},
	}
}

func trackedManagedInstanceResource() *wlmsv1beta1.ManagedInstance {
	resource := newManagedInstanceResource()
	resource.Status.Id = testManagedInstanceID
	resource.Status.CompartmentId = testManagedInstanceCompartment
	resource.Status.DisplayName = testManagedInstanceName
	resource.Status.PluginStatus = resource.Spec.PluginStatus
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceID)
	return resource
}

func requestForManagedInstance(resource *wlmsv1beta1.ManagedInstance) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      resource.Name,
			Namespace: resource.Namespace,
		},
	}
}

func activeManagedInstanceWithConfigSDK(
	id string,
	discoveryInterval int,
	domainSearchPaths []string,
) wlmssdk.ManagedInstance {
	return wlmssdk.ManagedInstance{
		Id:            common.String(id),
		DisplayName:   common.String(testManagedInstanceName),
		CompartmentId: common.String(testManagedInstanceCompartment),
		HostName:      common.String(testManagedInstanceHostName),
		ServerCount:   common.Int(1),
		PluginStatus:  common.String(string(wlmssdk.ListManagedInstancesPluginStatusActive)),
		Configuration: &wlmssdk.ManagedInstanceConfiguration{
			DiscoveryInterval: common.Int(discoveryInterval),
			DomainSearchPaths: append([]string(nil), domainSearchPaths...),
		},
	}
}

func managedInstanceSummary(id string, displayName string, pluginStatus string) wlmssdk.ManagedInstanceSummary {
	return wlmssdk.ManagedInstanceSummary{
		Id:            common.String(id),
		DisplayName:   common.String(displayName),
		CompartmentId: common.String(testManagedInstanceCompartment),
		HostName:      common.String(testManagedInstanceHostName),
		ServerCount:   common.Int(1),
		PluginStatus:  common.String(pluginStatus),
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

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
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
