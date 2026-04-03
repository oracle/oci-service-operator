//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociopensearch "github.com/oracle/oci-go-sdk/v65/opensearch"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type OpenSearchCluster = opensearchv1beta1.OpensearchCluster

func makePendingOpenSearchCluster(id, name string, state ociopensearch.OpensearchClusterLifecycleStateEnum) ociopensearch.OpensearchCluster {
	cluster := makeActiveCluster(id, name)
	cluster.LifecycleState = state
	return cluster
}

func makeOpenSearchSpec(name string) *OpenSearchCluster {
	cluster := &OpenSearchCluster{}
	cluster.Name = name
	cluster.Namespace = "default"
	cluster.Spec.DisplayName = name
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	cluster.Spec.SoftwareVersion = "2.3.0"
	cluster.Spec.MasterNodeCount = 3
	cluster.Spec.MasterNodeHostType = "FLEX"
	cluster.Spec.MasterNodeHostOcpuCount = 1
	cluster.Spec.MasterNodeHostMemoryGB = 8
	cluster.Spec.DataNodeCount = 3
	cluster.Spec.DataNodeHostType = "FLEX"
	cluster.Spec.DataNodeHostOcpuCount = 1
	cluster.Spec.DataNodeHostMemoryGB = 8
	cluster.Spec.DataNodeStorageGB = 50
	cluster.Spec.OpendashboardNodeCount = 1
	cluster.Spec.OpendashboardNodeHostOcpuCount = 1
	cluster.Spec.OpendashboardNodeHostMemoryGB = 8
	cluster.Spec.VcnId = "ocid1.vcn.oc1..x"
	cluster.Spec.SubnetId = "ocid1.subnet.oc1..x"
	cluster.Spec.VcnCompartmentId = "ocid1.compartment.oc1..xxx"
	cluster.Spec.SubnetCompartmentId = "ocid1.compartment.oc1..xxx"
	return cluster
}

func TestPropertyOpenSearchCreatePathRequestsRequeue(t *testing.T) {
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{}, nil
		},
		createFn: func(_ context.Context, _ ociopensearch.CreateOpensearchClusterRequest) (ociopensearch.CreateOpensearchClusterResponse, error) {
			return ociopensearch.CreateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithFake(fake)
	cluster := makeOpenSearchSpec("new-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
}

func TestPropertyOpenSearchPendingStatesRequestRequeue(t *testing.T) {
	for _, state := range []ociopensearch.OpensearchClusterLifecycleStateEnum{
		ociopensearch.OpensearchClusterLifecycleStateCreating,
		ociopensearch.OpensearchClusterLifecycleStateUpdating,
	} {
		t.Run(string(state), func(t *testing.T) {
			fake := &fakeOciClient{
				listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
					return ociopensearch.ListOpensearchClustersResponse{
						OpensearchClusterCollection: ociopensearch.OpensearchClusterCollection{
							Items: []ociopensearch.OpensearchClusterSummary{{Id: common.String("ocid1.opensearchcluster.oc1..pending"), DisplayName: common.String("pending-cluster"), LifecycleState: state}},
						},
					}, nil
				},
				getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
					return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: makePendingOpenSearchCluster("ocid1.opensearchcluster.oc1..pending", "pending-cluster", state)}, nil
				},
			}
			mgr := makeManagerWithFake(fake)
			cluster := makeOpenSearchSpec("pending-cluster")

			resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
		})
	}
}

func TestPropertyOpenSearchBindByIDUsesSpecIDWhenStatusIsEmpty(t *testing.T) {
	var updatedID string
	fake := &fakeOciClient{
		getFn: func(_ context.Context, req ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: makeActiveCluster(*req.OpensearchClusterId, "old-bound-cluster")}, nil
		},
		updateFn: func(_ context.Context, req ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error) {
			updatedID = *req.OpensearchClusterId
			return ociopensearch.UpdateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithFake(fake)
	cluster := makeOpenSearchSpec("new-bound-cluster")
	cluster.Spec.OpenSearchClusterId = "ocid1.opensearchcluster.oc1..bind"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, string(cluster.Spec.OpenSearchClusterId), updatedID)
}

func TestPropertyOpenSearchDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	fake := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociopensearch.DeleteOpensearchClusterRequest) (ociopensearch.DeleteOpensearchClusterResponse, error) {
			return ociopensearch.DeleteOpensearchClusterResponse{}, nil
		},
		getFn: func(_ context.Context, req ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: makeActiveCluster(*req.OpensearchClusterId, "still-there")}, nil
		},
	}
	mgr := makeManagerWithFake(fake)
	cluster := makeOpenSearchSpec("still-there")
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..delete"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.False(t, done)
}
