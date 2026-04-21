/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dedicatedaicluster

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDedicatedAiClusterOCIClient struct {
	createFn func(context.Context, generativeaisdk.CreateDedicatedAiClusterRequest) (generativeaisdk.CreateDedicatedAiClusterResponse, error)
	getFn    func(context.Context, generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error)
	listFn   func(context.Context, generativeaisdk.ListDedicatedAiClustersRequest) (generativeaisdk.ListDedicatedAiClustersResponse, error)
	updateFn func(context.Context, generativeaisdk.UpdateDedicatedAiClusterRequest) (generativeaisdk.UpdateDedicatedAiClusterResponse, error)
	deleteFn func(context.Context, generativeaisdk.DeleteDedicatedAiClusterRequest) (generativeaisdk.DeleteDedicatedAiClusterResponse, error)
}

func (f *fakeDedicatedAiClusterOCIClient) CreateDedicatedAiCluster(
	ctx context.Context,
	req generativeaisdk.CreateDedicatedAiClusterRequest,
) (generativeaisdk.CreateDedicatedAiClusterResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return generativeaisdk.CreateDedicatedAiClusterResponse{}, nil
}

func (f *fakeDedicatedAiClusterOCIClient) GetDedicatedAiCluster(
	ctx context.Context,
	req generativeaisdk.GetDedicatedAiClusterRequest,
) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return generativeaisdk.GetDedicatedAiClusterResponse{}, fakeDedicatedAiClusterServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "missing",
	}
}

func (f *fakeDedicatedAiClusterOCIClient) ListDedicatedAiClusters(
	ctx context.Context,
	req generativeaisdk.ListDedicatedAiClustersRequest,
) (generativeaisdk.ListDedicatedAiClustersResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return generativeaisdk.ListDedicatedAiClustersResponse{}, nil
}

func (f *fakeDedicatedAiClusterOCIClient) UpdateDedicatedAiCluster(
	ctx context.Context,
	req generativeaisdk.UpdateDedicatedAiClusterRequest,
) (generativeaisdk.UpdateDedicatedAiClusterResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return generativeaisdk.UpdateDedicatedAiClusterResponse{}, nil
}

func (f *fakeDedicatedAiClusterOCIClient) DeleteDedicatedAiCluster(
	ctx context.Context,
	req generativeaisdk.DeleteDedicatedAiClusterRequest,
) (generativeaisdk.DeleteDedicatedAiClusterResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return generativeaisdk.DeleteDedicatedAiClusterResponse{}, nil
}

type fakeDedicatedAiClusterServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeDedicatedAiClusterServiceError) Error() string          { return f.message }
func (f fakeDedicatedAiClusterServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeDedicatedAiClusterServiceError) GetMessage() string     { return f.message }
func (f fakeDedicatedAiClusterServiceError) GetCode() string        { return f.code }
func (f fakeDedicatedAiClusterServiceError) GetOpcRequestID() string {
	return ""
}

func newTestDedicatedAiClusterDelegate(client *fakeDedicatedAiClusterOCIClient) DedicatedAiClusterServiceClient {
	if client == nil {
		client = &fakeDedicatedAiClusterOCIClient{}
	}

	hooks := newDedicatedAiClusterDefaultRuntimeHooks(generativeaisdk.GenerativeAiClient{})
	hooks.Create.Call = func(ctx context.Context, request generativeaisdk.CreateDedicatedAiClusterRequest) (generativeaisdk.CreateDedicatedAiClusterResponse, error) {
		return client.CreateDedicatedAiCluster(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
		return client.GetDedicatedAiCluster(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request generativeaisdk.ListDedicatedAiClustersRequest) (generativeaisdk.ListDedicatedAiClustersResponse, error) {
		return client.ListDedicatedAiClusters(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request generativeaisdk.UpdateDedicatedAiClusterRequest) (generativeaisdk.UpdateDedicatedAiClusterResponse, error) {
		return client.UpdateDedicatedAiCluster(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request generativeaisdk.DeleteDedicatedAiClusterRequest) (generativeaisdk.DeleteDedicatedAiClusterResponse, error) {
		return client.DeleteDedicatedAiCluster(ctx, request)
	}
	applyDedicatedAiClusterRuntimeHooks(&hooks)
	config := buildDedicatedAiClusterGeneratedRuntimeConfig(&DedicatedAiClusterServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}, hooks)

	delegate := defaultDedicatedAiClusterServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*generativeaiv1beta1.DedicatedAiCluster](config),
	}
	return wrapDedicatedAiClusterGeneratedClient(hooks, delegate)
}

func newTestDedicatedAiClusterClient(client *fakeDedicatedAiClusterOCIClient) DedicatedAiClusterServiceClient {
	return newTestDedicatedAiClusterDelegate(client)
}

func makeSpecDedicatedAiCluster() *generativeaiv1beta1.DedicatedAiCluster {
	return &generativeaiv1beta1.DedicatedAiCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dedicatedaicluster-sample",
			Namespace: "default",
		},
		Spec: generativeaiv1beta1.DedicatedAiClusterSpec{
			Type:          string(generativeaisdk.DedicatedAiClusterTypeHosting),
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
			UnitCount:     1,
			UnitShape:     string(generativeaisdk.DedicatedAiClusterUnitShapeSmallCohere),
			DisplayName:   "osok-dedicatedaicluster-sample",
			Description:   "OSOK DedicatedAiCluster sample",
		},
	}
}

func makeSDKDedicatedAiCluster(
	id string,
	spec generativeaiv1beta1.DedicatedAiClusterSpec,
	state generativeaisdk.DedicatedAiClusterLifecycleStateEnum,
) generativeaisdk.DedicatedAiCluster {
	cluster := generativeaisdk.DedicatedAiCluster{
		Id:             common.String(id),
		Type:           generativeaisdk.DedicatedAiClusterTypeEnum(spec.Type),
		CompartmentId:  common.String(spec.CompartmentId),
		LifecycleState: state,
		UnitCount:      intPtr(spec.UnitCount),
		UnitShape:      generativeaisdk.DedicatedAiClusterUnitShapeEnum(spec.UnitShape),
		DefinedTags:    map[string]map[string]interface{}{},
		FreeformTags:   map[string]string{},
		SystemTags:     map[string]map[string]interface{}{},
	}
	if spec.DisplayName != "" {
		cluster.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		cluster.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		cluster.FreeformTags = spec.FreeformTags
	}
	return cluster
}

func makeSDKDedicatedAiClusterSummary(
	id string,
	spec generativeaiv1beta1.DedicatedAiClusterSpec,
	state generativeaisdk.DedicatedAiClusterLifecycleStateEnum,
) generativeaisdk.DedicatedAiClusterSummary {
	cluster := generativeaisdk.DedicatedAiClusterSummary{
		Id:             common.String(id),
		Type:           generativeaisdk.DedicatedAiClusterTypeEnum(spec.Type),
		CompartmentId:  common.String(spec.CompartmentId),
		LifecycleState: state,
		UnitCount:      intPtr(spec.UnitCount),
		UnitShape:      generativeaisdk.DedicatedAiClusterUnitShapeEnum(spec.UnitShape),
		DefinedTags:    map[string]map[string]interface{}{},
		FreeformTags:   map[string]string{},
		SystemTags:     map[string]map[string]interface{}{},
	}
	if spec.DisplayName != "" {
		cluster.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		cluster.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		cluster.FreeformTags = spec.FreeformTags
	}
	return cluster
}

func intPtr(v int) *int {
	return &v
}

func TestDedicatedAiClusterCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.dedicatedaicluster.oc1..created"

	resource := makeSpecDedicatedAiCluster()
	resource.Spec.DisplayName = ""

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		createFn: func(_ context.Context, req generativeaisdk.CreateDedicatedAiClusterRequest) (generativeaisdk.CreateDedicatedAiClusterResponse, error) {
			createCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("CreateDedicatedAiClusterRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName != nil {
				t.Fatalf("CreateDedicatedAiClusterRequest.DisplayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			if req.Type != generativeaisdk.DedicatedAiClusterTypeEnum(resource.Spec.Type) {
				t.Fatalf("CreateDedicatedAiClusterRequest.Type = %q, want %q", req.Type, resource.Spec.Type)
			}
			if req.UnitCount == nil || *req.UnitCount != resource.Spec.UnitCount {
				t.Fatalf("CreateDedicatedAiClusterRequest.UnitCount = %v, want %d", req.UnitCount, resource.Spec.UnitCount)
			}
			if req.UnitShape != generativeaisdk.DedicatedAiClusterUnitShapeEnum(resource.Spec.UnitShape) {
				t.Fatalf("CreateDedicatedAiClusterRequest.UnitShape = %q, want %q", req.UnitShape, resource.Spec.UnitShape)
			}
			return generativeaisdk.CreateDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(createdID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateCreating),
			}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			getCalls++
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != createdID {
				t.Fatalf("GetDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, createdID)
			}
			return generativeaisdk.GetDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(createdID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
			}, nil
		},
		listFn: func(context.Context, generativeaisdk.ListDedicatedAiClustersRequest) (generativeaisdk.ListDedicatedAiClustersResponse, error) {
			listCalls++
			return generativeaisdk.ListDedicatedAiClustersResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after ACTIVE follow-up", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDedicatedAiCluster() calls = %d, want 1", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListDedicatedAiClusters() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDedicatedAiCluster() calls = %d, want 1", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestDedicatedAiClusterCreateOrUpdateClearsStaleTrackedIDWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	const (
		staleID = "ocid1.dedicatedaicluster.oc1..stale"
		newID   = "ocid1.dedicatedaicluster.oc1..new"
	)

	resource := makeSpecDedicatedAiCluster()
	resource.Spec.DisplayName = ""
	resource.Status.Id = staleID
	resource.Status.OsokStatus.Ocid = shared.OCID(staleID)

	createCalls := 0
	listCalls := 0
	getIDs := make([]string, 0, 2)

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		createFn: func(_ context.Context, req generativeaisdk.CreateDedicatedAiClusterRequest) (generativeaisdk.CreateDedicatedAiClusterResponse, error) {
			createCalls++
			if req.DisplayName != nil {
				t.Fatalf("CreateDedicatedAiClusterRequest.DisplayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			return generativeaisdk.CreateDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(newID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateCreating),
			}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			gotID := ""
			if req.DedicatedAiClusterId != nil {
				gotID = *req.DedicatedAiClusterId
			}
			getIDs = append(getIDs, gotID)
			switch gotID {
			case staleID:
				return generativeaisdk.GetDedicatedAiClusterResponse{}, fakeDedicatedAiClusterServiceError{
					statusCode: 404,
					code:       "NotFound",
					message:    "missing",
				}
			case newID:
				return generativeaisdk.GetDedicatedAiClusterResponse{
					DedicatedAiCluster: makeSDKDedicatedAiCluster(newID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
				}, nil
			default:
				t.Fatalf("unexpected GetDedicatedAiClusterRequest.DedicatedAiClusterId %q", gotID)
				return generativeaisdk.GetDedicatedAiClusterResponse{}, nil
			}
		},
		listFn: func(context.Context, generativeaisdk.ListDedicatedAiClustersRequest) (generativeaisdk.ListDedicatedAiClustersResponse, error) {
			listCalls++
			return generativeaisdk.ListDedicatedAiClustersResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDedicatedAiCluster() calls = %d, want 1 after clearing stale ID", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListDedicatedAiClusters() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if len(getIDs) != 2 || getIDs[0] != staleID || getIDs[1] != newID {
		t.Fatalf("GetDedicatedAiCluster() ids = %v, want [%q %q]", getIDs, staleID, newID)
	}
	if got := resource.Status.Id; got != newID {
		t.Fatalf("status.id = %q, want %q", got, newID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != newID {
		t.Fatalf("status.status.ocid = %q, want %q", got, newID)
	}
}

func TestDedicatedAiClusterCreateOrUpdateClearsDeletedTrackedIDWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	const (
		staleID = "ocid1.dedicatedaicluster.oc1..deleted"
		newID   = "ocid1.dedicatedaicluster.oc1..new"
	)

	resource := makeSpecDedicatedAiCluster()
	resource.Spec.DisplayName = ""
	resource.Status.Id = staleID
	resource.Status.OsokStatus.Ocid = shared.OCID(staleID)

	createCalls := 0
	listCalls := 0
	getIDs := make([]string, 0, 2)

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		createFn: func(_ context.Context, req generativeaisdk.CreateDedicatedAiClusterRequest) (generativeaisdk.CreateDedicatedAiClusterResponse, error) {
			createCalls++
			if req.DisplayName != nil {
				t.Fatalf("CreateDedicatedAiClusterRequest.DisplayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			return generativeaisdk.CreateDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(newID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateCreating),
			}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			gotID := ""
			if req.DedicatedAiClusterId != nil {
				gotID = *req.DedicatedAiClusterId
			}
			getIDs = append(getIDs, gotID)
			switch gotID {
			case staleID:
				return generativeaisdk.GetDedicatedAiClusterResponse{
					DedicatedAiCluster: makeSDKDedicatedAiCluster(staleID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateDeleted),
				}, nil
			case newID:
				return generativeaisdk.GetDedicatedAiClusterResponse{
					DedicatedAiCluster: makeSDKDedicatedAiCluster(newID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
				}, nil
			default:
				t.Fatalf("unexpected GetDedicatedAiClusterRequest.DedicatedAiClusterId %q", gotID)
				return generativeaisdk.GetDedicatedAiClusterResponse{}, nil
			}
		},
		listFn: func(context.Context, generativeaisdk.ListDedicatedAiClustersRequest) (generativeaisdk.ListDedicatedAiClustersResponse, error) {
			listCalls++
			return generativeaisdk.ListDedicatedAiClustersResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDedicatedAiCluster() calls = %d, want 1 after clearing deleted tracked ID", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListDedicatedAiClusters() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if len(getIDs) != 2 || getIDs[0] != staleID || getIDs[1] != newID {
		t.Fatalf("GetDedicatedAiCluster() ids = %v, want [%q %q]", getIDs, staleID, newID)
	}
	if got := resource.Status.Id; got != newID {
		t.Fatalf("status.id = %q, want %q", got, newID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != newID {
		t.Fatalf("status.status.ocid = %q, want %q", got, newID)
	}
}

func TestDedicatedAiClusterCreateOrUpdateBindsExistingResourceByDisplayName(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.dedicatedaicluster.oc1..existing"

	resource := makeSpecDedicatedAiCluster()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		createFn: func(context.Context, generativeaisdk.CreateDedicatedAiClusterRequest) (generativeaisdk.CreateDedicatedAiClusterResponse, error) {
			createCalled = true
			return generativeaisdk.CreateDedicatedAiClusterResponse{}, nil
		},
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			getCalls++
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != existingID {
				t.Fatalf("GetDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, existingID)
			}
			return generativeaisdk.GetDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(existingID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
			}, nil
		},
		listFn: func(_ context.Context, req generativeaisdk.ListDedicatedAiClustersRequest) (generativeaisdk.ListDedicatedAiClustersResponse, error) {
			listCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("ListDedicatedAiClustersRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("ListDedicatedAiClustersRequest.DisplayName = %v, want %q", req.DisplayName, resource.Spec.DisplayName)
			}
			return generativeaisdk.ListDedicatedAiClustersResponse{
				DedicatedAiClusterCollection: generativeaisdk.DedicatedAiClusterCollection{
					Items: []generativeaisdk.DedicatedAiClusterSummary{
						makeSDKDedicatedAiClusterSummary("ocid1.dedicatedaicluster.oc1..other", generativeaiv1beta1.DedicatedAiClusterSpec{
							Type:          resource.Spec.Type,
							CompartmentId: resource.Spec.CompartmentId,
							UnitCount:     resource.Spec.UnitCount,
							UnitShape:     resource.Spec.UnitShape,
							DisplayName:   "other",
							Description:   "other",
						}, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
						makeSDKDedicatedAiClusterSummary(existingID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
					},
				},
			}, nil
		},
		updateFn: func(context.Context, generativeaisdk.UpdateDedicatedAiClusterRequest) (generativeaisdk.UpdateDedicatedAiClusterResponse, error) {
			updateCalled = true
			return generativeaisdk.UpdateDedicatedAiClusterResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue for ACTIVE bind", response)
	}
	if createCalled {
		t.Fatal("CreateDedicatedAiCluster() called, want displayName bind path")
	}
	if updateCalled {
		t.Fatal("UpdateDedicatedAiCluster() called, want observe-only bind path")
	}
	if listCalls != 1 {
		t.Fatalf("ListDedicatedAiClusters() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDedicatedAiCluster() calls = %d, want 1", getCalls)
	}
	if got := resource.Status.Id; got != existingID {
		t.Fatalf("status.id = %q, want %q", got, existingID)
	}
}

func TestDedicatedAiClusterCreateOrUpdateUpdatesMutableUnitCount(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.dedicatedaicluster.oc1..existing"

	resource := makeSpecDedicatedAiCluster()
	resource.Spec.UnitCount = 2
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	updateCalls := 0

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			getCalls++
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != existingID {
				t.Fatalf("GetDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, existingID)
			}
			currentSpec := resource.Spec
			currentSpec.UnitCount = 1
			if getCalls > 1 {
				currentSpec.UnitCount = resource.Spec.UnitCount
			}
			return generativeaisdk.GetDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(existingID, currentSpec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req generativeaisdk.UpdateDedicatedAiClusterRequest) (generativeaisdk.UpdateDedicatedAiClusterResponse, error) {
			updateCalls++
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != existingID {
				t.Fatalf("UpdateDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, existingID)
			}
			if req.UnitCount == nil || *req.UnitCount != resource.Spec.UnitCount {
				t.Fatalf("UpdateDedicatedAiClusterRequest.UnitCount = %v, want %d", req.UnitCount, resource.Spec.UnitCount)
			}
			if req.DisplayName != nil {
				t.Fatalf("UpdateDedicatedAiClusterRequest.DisplayName = %v, want nil when unchanged", req.DisplayName)
			}
			return generativeaisdk.UpdateDedicatedAiClusterResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after ACTIVE update follow-up", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDedicatedAiCluster() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetDedicatedAiCluster() calls = %d, want 2", getCalls)
	}
	if got := resource.Status.UnitCount; got != resource.Spec.UnitCount {
		t.Fatalf("status.unitCount = %d, want %d", got, resource.Spec.UnitCount)
	}
}

func TestDedicatedAiClusterCreateOrUpdateRejectsForceNewUnitShapeDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.dedicatedaicluster.oc1..existing"

	resource := makeSpecDedicatedAiCluster()
	resource.Spec.UnitShape = string(generativeaisdk.DedicatedAiClusterUnitShapeLargeCohere)
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	updateCalled := false

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != existingID {
				t.Fatalf("GetDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, existingID)
			}
			currentSpec := resource.Spec
			currentSpec.UnitShape = string(generativeaisdk.DedicatedAiClusterUnitShapeSmallCohere)
			return generativeaisdk.GetDedicatedAiClusterResponse{
				DedicatedAiCluster: makeSDKDedicatedAiCluster(existingID, currentSpec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, generativeaisdk.UpdateDedicatedAiClusterRequest) (generativeaisdk.UpdateDedicatedAiClusterResponse, error) {
			updateCalled = true
			return generativeaisdk.UpdateDedicatedAiClusterResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when unitShape changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required unitShape drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateDedicatedAiCluster() called, want force-new drift rejection before update")
	}
}

func TestDedicatedAiClusterDeleteConfirmsTerminalLifecycle(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.dedicatedaicluster.oc1..existing"

	resource := makeSpecDedicatedAiCluster()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	deleteCalls := 0

	client := newTestDedicatedAiClusterClient(&fakeDedicatedAiClusterOCIClient{
		getFn: func(_ context.Context, req generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error) {
			getCalls++
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != existingID {
				t.Fatalf("GetDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, existingID)
			}
			switch getCalls {
			case 1:
				return generativeaisdk.GetDedicatedAiClusterResponse{
					DedicatedAiCluster: makeSDKDedicatedAiCluster(existingID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateActive),
				}, nil
			case 2:
				return generativeaisdk.GetDedicatedAiClusterResponse{
					DedicatedAiCluster: makeSDKDedicatedAiCluster(existingID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateDeleting),
				}, nil
			default:
				return generativeaisdk.GetDedicatedAiClusterResponse{
					DedicatedAiCluster: makeSDKDedicatedAiCluster(existingID, resource.Spec, generativeaisdk.DedicatedAiClusterLifecycleStateDeleted),
				}, nil
			}
		},
		deleteFn: func(_ context.Context, req generativeaisdk.DeleteDedicatedAiClusterRequest) (generativeaisdk.DeleteDedicatedAiClusterResponse, error) {
			deleteCalls++
			if req.DedicatedAiClusterId == nil || *req.DedicatedAiClusterId != existingID {
				t.Fatalf("DeleteDedicatedAiClusterRequest.DedicatedAiClusterId = %v, want %q", req.DedicatedAiClusterId, existingID)
			}
			return generativeaisdk.DeleteDedicatedAiClusterResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call = true, want delete confirmation requeue while DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDedicatedAiCluster() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call = false, want terminal delete confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDedicatedAiCluster() calls after second delete = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetDedicatedAiCluster() calls = %d, want 3", getCalls)
	}
}
