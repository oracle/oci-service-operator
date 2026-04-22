/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cluster

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestClusterCreateOrUpdateRejectsCreateWithoutDisplayNameWhenUntracked(t *testing.T) {
	t.Parallel()

	resource := newClusterTestResource()
	resource.Spec.DisplayName = ""
	createCalled := false
	listCalled := false

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Cluster]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateClusterRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when displayName is missing")
				return nil, nil
			},
			Fields: clusterCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListClustersRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				listCalled = true
				t.Fatal("List() should not be called when displayName is missing")
				return nil, nil
			},
			Fields: clusterListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want displayName requirement failure")
	}
	if err.Error() != "Cluster spec.displayName is required when no OCI identifier is recorded" {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit displayName requirement", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when displayName is missing")
	}
	if createCalled {
		t.Fatal("Create() should not be called when displayName is missing")
	}
	if listCalled {
		t.Fatal("List() should not be called when displayName is missing")
	}
	requireClusterCondition(t, resource, shared.Failed)
}

func TestClusterCreateOrUpdateCreatesAndRequeuesFromLifecycleFollowUp(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.cluster.oc1..created"

	resource := newClusterTestResource()
	calls := make([]string, 0, 3)
	var createRequest ocvpsdk.CreateClusterRequest
	var listRequest ocvpsdk.ListClustersRequest
	listCalls := 0

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Cluster]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				calls = append(calls, "create")
				createRequest = *request.(*ocvpsdk.CreateClusterRequest)
				return ocvpsdk.CreateClusterResponse{
					OpcWorkRequestId: common.String("ocid1.workrequest.oc1..create"),
				}, nil
			},
			Fields: clusterCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListClustersRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				calls = append(calls, "list")
				listRequest = *request.(*ocvpsdk.ListClustersRequest)
				listCalls++
				if listCalls == 1 {
					return ocvpsdk.ListClustersResponse{
						ClusterCollection: ocvpsdk.ClusterCollection{
							Items: nil,
						},
					}, nil
				}
				return ocvpsdk.ListClustersResponse{
					ClusterCollection: ocvpsdk.ClusterCollection{
						Items: []ocvpsdk.ClusterSummary{
							observedClusterSummaryFromSpec(createdID, resource.Spec, "CREATING"),
						},
					},
				}, nil
			},
			Fields: clusterListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when create follow-up observes CREATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle remains CREATING")
	}
	if len(calls) != 3 || calls[0] != "list" || calls[1] != "create" || calls[2] != "list" {
		t.Fatalf("call order = %v, want [list create list]", calls)
	}
	requireClusterStringPointer(t, "create request displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireClusterStringPointer(t, "create request sddcId", createRequest.SddcId, resource.Spec.SddcId)
	requireClusterStringPointer(t, "create request computeAvailabilityDomain", createRequest.ComputeAvailabilityDomain, resource.Spec.ComputeAvailabilityDomain)
	requireClusterIntPointer(t, "create request esxiHostsCount", createRequest.EsxiHostsCount, resource.Spec.EsxiHostsCount)
	if createRequest.NetworkConfiguration == nil {
		t.Fatal("create request networkConfiguration = nil, want required network configuration")
	}
	requireClusterStringPointer(t, "list request displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireClusterStringPointer(t, "list request sddcId", listRequest.SddcId, resource.Spec.SddcId)
	requireClusterCondition(t, resource, shared.Provisioning)
	requireClusterOCID(t, resource, createdID)
	if resource.Status.LifecycleState != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "CREATING")
	}
}

func TestClusterCreateOrUpdateReusesMatchingListEntryWhenDisplayNamePresent(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.cluster.oc1..existing"

	resource := newClusterTestResource()
	createCalled := false
	var listRequest ocvpsdk.ListClustersRequest

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Cluster]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateClusterRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable cluster")
				return nil, nil
			},
			Fields: clusterCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListClustersRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*ocvpsdk.ListClustersRequest)
				return ocvpsdk.ListClustersResponse{
					ClusterCollection: ocvpsdk.ClusterCollection{
						Items: []ocvpsdk.ClusterSummary{
							observedClusterSummaryFromSpec(existingID, resource.Spec, "ACTIVE"),
						},
					},
				}, nil
			},
			Fields: clusterListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list lookup reuses an existing cluster")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE list reuse")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable cluster")
	}
	requireClusterStringPointer(t, "list request displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireClusterStringPointer(t, "list request sddcId", listRequest.SddcId, resource.Spec.SddcId)
	requireClusterOCID(t, resource, existingID)
	if resource.Status.Id != existingID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, existingID)
	}
	requireClusterCondition(t, resource, shared.Active)
}

func TestClusterCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.cluster.oc1..existing"

	resource := newExistingClusterTestResource(existingID)
	resource.Spec.DisplayName = "cluster-renamed"

	getCalls := 0
	var updateRequest ocvpsdk.UpdateClusterRequest

	manager := newClusterRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Cluster]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireClusterStringPointer(t, "get request clusterId", request.(*ocvpsdk.GetClusterRequest).ClusterId, existingID)
				live := observedClusterFromSpec(existingID, newClusterTestResource().Spec, "ACTIVE")
				if getCalls > 1 {
					live.DisplayName = common.String(resource.Spec.DisplayName)
				}
				return ocvpsdk.GetClusterResponse{Cluster: live}, nil
			},
			Fields: clusterGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*ocvpsdk.UpdateClusterRequest)
				return ocvpsdk.UpdateClusterResponse{
					Cluster: observedClusterFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: clusterUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when a mutable Cluster field changes")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when the update follow-up sees ACTIVE state")
	}
	if getCalls != 2 {
		t.Fatalf("get calls = %d, want 2", getCalls)
	}
	requireClusterStringPointer(t, "update request clusterId", updateRequest.ClusterId, existingID)
	requireClusterStringPointer(t, "update request displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireClusterOCID(t, resource, existingID)
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
	requireClusterCondition(t, resource, shared.Active)
}

func TestClusterCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.cluster.oc1..existing"

	resource := newExistingClusterTestResource(existingID)
	resource.Spec.SddcId = "ocid1.sddc.oc1..desired"

	updateCalled := false
	manager := newClusterRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Cluster]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetClusterRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				requireClusterStringPointer(t, "get request clusterId", request.(*ocvpsdk.GetClusterRequest).ClusterId, existingID)
				live := observedClusterFromSpec(existingID, newClusterTestResource().Spec, "ACTIVE")
				live.SddcId = common.String("ocid1.sddc.oc1..live")
				return ocvpsdk.GetClusterResponse{Cluster: live}, nil
			},
			Fields: clusterGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateClusterRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalled = true
				t.Fatal("Update() should not be called when create-only Cluster fields drift")
				return nil, nil
			},
			Fields: clusterUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "Cluster formal semantics require replacement when sddcId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit sddcId replacement failure", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when create-only Cluster fields drift")
	}
	if updateCalled {
		t.Fatal("Update() should not be called when create-only Cluster fields drift")
	}
	requireClusterCondition(t, resource, shared.Failed)
}

func TestClusterDeleteConfirmsDeleteOutcomes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		confirmedState   string
		wantDeleted      bool
		wantMessage      string
		wantDeletedStamp bool
	}{
		{
			name:             "deleting remains pending",
			confirmedState:   "DELETING",
			wantDeleted:      false,
			wantMessage:      "OCI resource delete is in progress",
			wantDeletedStamp: false,
		},
		{
			name:             "deleted confirms removal",
			confirmedState:   "DELETED",
			wantDeleted:      true,
			wantMessage:      "OCI resource deleted",
			wantDeletedStamp: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.cluster.oc1..existing"

			resource := newExistingClusterTestResource(existingID)
			getCalls := 0
			var deleteRequest ocvpsdk.DeleteClusterRequest

			manager := newClusterRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Cluster]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &ocvpsdk.GetClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getCalls++
						requireClusterStringPointer(t, "get request clusterId", request.(*ocvpsdk.GetClusterRequest).ClusterId, existingID)
						state := "ACTIVE"
						if getCalls > 1 {
							state = tc.confirmedState
						}
						return ocvpsdk.GetClusterResponse{
							Cluster: observedClusterFromSpec(existingID, resource.Spec, state),
						}, nil
					},
					Fields: clusterGetFields(),
				},
				Delete: &generatedruntime.Operation{
					NewRequest: func() any { return &ocvpsdk.DeleteClusterRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						deleteRequest = *request.(*ocvpsdk.DeleteClusterRequest)
						return ocvpsdk.DeleteClusterResponse{
							OpcWorkRequestId: common.String("ocid1.workrequest.oc1..delete"),
						}, nil
					},
					Fields: clusterDeleteFields(),
				},
			})

			deleted, err := manager.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted != tc.wantDeleted {
				t.Fatalf("Delete() deleted = %t, want %t", deleted, tc.wantDeleted)
			}
			if getCalls != 2 {
				t.Fatalf("get calls = %d, want 2", getCalls)
			}
			requireClusterStringPointer(t, "delete request clusterId", deleteRequest.ClusterId, existingID)
			requireClusterCondition(t, resource, shared.Terminating)
			if resource.Status.LifecycleState != tc.confirmedState {
				t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.confirmedState)
			}
			if resource.Status.OsokStatus.Message != tc.wantMessage {
				t.Fatalf("status.message = %q, want %q", resource.Status.OsokStatus.Message, tc.wantMessage)
			}
			if tc.wantDeletedStamp {
				if resource.Status.OsokStatus.DeletedAt == nil {
					t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
				}
			} else if resource.Status.OsokStatus.DeletedAt != nil {
				t.Fatalf("status.deletedAt = %v, want nil while delete is still pending", resource.Status.OsokStatus.DeletedAt)
			}
		})
	}
}

func newClusterRuntimeTestManager(cfg generatedruntime.Config[*ocvpv1beta1.Cluster]) *ClusterServiceManager {
	manager := &ClusterServiceManager{}
	hooks := ClusterRuntimeHooks{
		Semantics:       newClusterRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*ocvpv1beta1.Cluster]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*ocvpv1beta1.Cluster]{},
		StatusHooks:     generatedruntime.StatusHooks[*ocvpv1beta1.Cluster]{},
		ParityHooks:     generatedruntime.ParityHooks[*ocvpv1beta1.Cluster]{},
		Create: clusterRuntimeTestOperation[ocvpsdk.CreateClusterRequest, ocvpsdk.CreateClusterResponse](
			clusterCreateFields(),
			cfg.Create,
		),
		Get: clusterRuntimeTestOperation[ocvpsdk.GetClusterRequest, ocvpsdk.GetClusterResponse](
			clusterGetFields(),
			cfg.Get,
		),
		List: clusterRuntimeTestOperation[ocvpsdk.ListClustersRequest, ocvpsdk.ListClustersResponse](
			clusterListFields(),
			cfg.List,
		),
		Update: clusterRuntimeTestOperation[ocvpsdk.UpdateClusterRequest, ocvpsdk.UpdateClusterResponse](
			clusterUpdateFields(),
			cfg.Update,
		),
		Delete: clusterRuntimeTestOperation[ocvpsdk.DeleteClusterRequest, ocvpsdk.DeleteClusterResponse](
			clusterDeleteFields(),
			cfg.Delete,
		),
		WrapGeneratedClient: []func(ClusterServiceClient) ClusterServiceClient{},
	}
	if cfg.Semantics != nil {
		hooks.Semantics = cfg.Semantics
	}
	applyClusterRuntimeHooks(&hooks)
	config := buildClusterGeneratedRuntimeConfig(manager, hooks)
	config.InitError = cfg.InitError
	if cfg.Create == nil {
		config.Create = nil
	}
	if cfg.Get == nil {
		config.Get = nil
		config.Read.Get = nil
	}
	if cfg.List == nil {
		config.List = nil
		config.Read.List = nil
	}
	if cfg.Update == nil {
		config.Update = nil
	}
	if cfg.Delete == nil {
		config.Delete = nil
	}
	delegate := defaultClusterServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.Cluster](config),
	}
	manager.client = wrapClusterGeneratedClient(hooks, delegate)
	return manager
}

func clusterRuntimeTestOperation[Req any, Resp any](
	defaultFields []generatedruntime.RequestField,
	override *generatedruntime.Operation,
) runtimeOperationHooks[Req, Resp] {
	op := runtimeOperationHooks[Req, Resp]{
		Fields: append([]generatedruntime.RequestField(nil), defaultFields...),
		Call: func(context.Context, Req) (Resp, error) {
			var zero Resp
			return zero, nil
		},
	}
	if override == nil {
		return op
	}
	if len(override.Fields) != 0 {
		op.Fields = append([]generatedruntime.RequestField(nil), override.Fields...)
	}
	op.Call = func(ctx context.Context, request Req) (Resp, error) {
		var zero Resp
		if override.Call == nil {
			return zero, nil
		}
		response, err := override.Call(ctx, &request)
		if err != nil {
			return zero, err
		}
		if response == nil {
			return zero, nil
		}
		typed, ok := response.(Resp)
		if !ok {
			return zero, fmt.Errorf("cluster test operation returned %T, want %T", response, zero)
		}
		return typed, nil
	}
	return op
}

func newClusterTestResource() *ocvpv1beta1.Cluster {
	return &ocvpv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-sample",
			Namespace: "default",
		},
		Spec: ocvpv1beta1.ClusterSpec{
			DisplayName:               "cluster-sample",
			SddcId:                    "ocid1.sddc.oc1..example",
			ComputeAvailabilityDomain: "Uocm:PHX-AD-1",
			EsxiHostsCount:            3,
			IsShieldedInstanceEnabled: true,
			NetworkConfiguration: ocvpv1beta1.ClusterNetworkConfiguration{
				ProvisioningSubnetId: "ocid1.subnet.oc1..example",
				VmotionVlanId:        "ocid1.vlan.oc1..vmotion",
				VsanVlanId:           "ocid1.vlan.oc1..vsan",
				NsxVTepVlanId:        "ocid1.vlan.oc1..nsx-vtep",
				NsxEdgeVTepVlanId:    "ocid1.vlan.oc1..nsx-edge-vtep",
			},
			FreeformTags: map[string]string{
				"env": "dev",
			},
		},
	}
}

func newExistingClusterTestResource(existingID string) *ocvpv1beta1.Cluster {
	resource := newClusterTestResource()
	resource.Status = ocvpv1beta1.ClusterStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedClusterFromSpec(id string, spec ocvpv1beta1.ClusterSpec, lifecycleState string) ocvpsdk.Cluster {
	return ocvpsdk.Cluster{
		Id:                        common.String(id),
		DisplayName:               common.String(spec.DisplayName),
		ComputeAvailabilityDomain: common.String(spec.ComputeAvailabilityDomain),
		CompartmentId:             common.String("ocid1.compartment.oc1..example"),
		SddcId:                    common.String(spec.SddcId),
		EsxiHostsCount:            common.Int(spec.EsxiHostsCount),
		InitialHostShapeName:      common.String("BM.DenseIO2.52"),
		VmwareSoftwareVersion:     common.String("vmware-version"),
		FreeformTags:              map[string]string{"env": "dev"},
		DefinedTags:               map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		IsShieldedInstanceEnabled: common.Bool(spec.IsShieldedInstanceEnabled),
		LifecycleState:            ocvpsdk.LifecycleStatesEnum(lifecycleState),
	}
}

func observedClusterSummaryFromSpec(id string, spec ocvpv1beta1.ClusterSpec, lifecycleState string) ocvpsdk.ClusterSummary {
	return ocvpsdk.ClusterSummary{
		Id:                        common.String(id),
		DisplayName:               common.String(spec.DisplayName),
		ComputeAvailabilityDomain: common.String(spec.ComputeAvailabilityDomain),
		CompartmentId:             common.String("ocid1.compartment.oc1..example"),
		SddcId:                    common.String(spec.SddcId),
		EsxiHostsCount:            common.Int(spec.EsxiHostsCount),
		InitialHostShapeName:      common.String("BM.DenseIO2.52"),
		VmwareSoftwareVersion:     common.String("vmware-version"),
		FreeformTags:              map[string]string{"env": "dev"},
		DefinedTags:               map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		IsShieldedInstanceEnabled: common.Bool(spec.IsShieldedInstanceEnabled),
		LifecycleState:            ocvpsdk.LifecycleStatesEnum(lifecycleState),
	}
}

func clusterCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateClusterDetails", RequestName: "CreateClusterDetails", Contribution: "body"},
	}
}

func clusterGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
	}
}

func clusterListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SddcId", RequestName: "sddcId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
	}
}

func clusterUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateClusterDetails", RequestName: "UpdateClusterDetails", Contribution: "body"},
	}
}

func clusterDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "path", PreferResourceID: true},
	}
}

func requireClusterCondition(t *testing.T, resource *ocvpv1beta1.Cluster, want shared.OSOKConditionType) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != want {
		t.Fatalf("status.conditions = %#v, want trailing %s condition", conditions, want)
	}
}

func requireClusterOCID(t *testing.T, resource *ocvpv1beta1.Cluster, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
}

func requireClusterStringPointer(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func requireClusterIntPointer(t *testing.T, label string, got *int, want int) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", label, got, want)
	}
}
