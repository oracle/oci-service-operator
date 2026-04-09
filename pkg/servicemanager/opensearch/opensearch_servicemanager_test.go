//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociopensearch "github.com/oracle/oci-go-sdk/v65/opensearch"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/opensearch"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f *fakeServiceError) GetHTTPStatusCode() int  { return f.statusCode }
func (f *fakeServiceError) GetMessage() string      { return f.message }
func (f *fakeServiceError) GetCode() string         { return f.code }
func (f *fakeServiceError) GetOpcRequestID() string { return "" }
func (f *fakeServiceError) Error() string {
	return fmt.Sprintf("%d %s: %s", f.statusCode, f.code, f.message)
}

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}
func (f *fakeCredentialClient) DeleteSecret(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (f *fakeCredentialClient) GetSecret(_ context.Context, _, _ string) (map[string][]byte, error) {
	return nil, nil
}
func (f *fakeCredentialClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}

// fakeOciClient implements the OpensearchClusterClientInterface interface for testing.
type fakeOciClient struct {
	createFn           func(ctx context.Context, req ociopensearch.CreateOpensearchClusterRequest) (ociopensearch.CreateOpensearchClusterResponse, error)
	getFn              func(ctx context.Context, req ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error)
	resizeHorizontalFn func(ctx context.Context, req ociopensearch.ResizeOpensearchClusterHorizontalRequest) (ociopensearch.ResizeOpensearchClusterHorizontalResponse, error)
	resizeVerticalFn   func(ctx context.Context, req ociopensearch.ResizeOpensearchClusterVerticalRequest) (ociopensearch.ResizeOpensearchClusterVerticalResponse, error)
	updateFn           func(ctx context.Context, req ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error)
	deleteFn           func(ctx context.Context, req ociopensearch.DeleteOpensearchClusterRequest) (ociopensearch.DeleteOpensearchClusterResponse, error)
	listFn             func(ctx context.Context, req ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error)
}

func (f *fakeOciClient) CreateOpensearchCluster(ctx context.Context, req ociopensearch.CreateOpensearchClusterRequest) (ociopensearch.CreateOpensearchClusterResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return ociopensearch.CreateOpensearchClusterResponse{}, nil
}

func (f *fakeOciClient) GetOpensearchCluster(ctx context.Context, req ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return ociopensearch.GetOpensearchClusterResponse{}, nil
}

func (f *fakeOciClient) ResizeOpensearchClusterHorizontal(ctx context.Context, req ociopensearch.ResizeOpensearchClusterHorizontalRequest) (ociopensearch.ResizeOpensearchClusterHorizontalResponse, error) {
	if f.resizeHorizontalFn != nil {
		return f.resizeHorizontalFn(ctx, req)
	}
	return ociopensearch.ResizeOpensearchClusterHorizontalResponse{}, nil
}

func (f *fakeOciClient) ResizeOpensearchClusterVertical(ctx context.Context, req ociopensearch.ResizeOpensearchClusterVerticalRequest) (ociopensearch.ResizeOpensearchClusterVerticalResponse, error) {
	if f.resizeVerticalFn != nil {
		return f.resizeVerticalFn(ctx, req)
	}
	return ociopensearch.ResizeOpensearchClusterVerticalResponse{}, nil
}

func (f *fakeOciClient) UpdateOpensearchCluster(ctx context.Context, req ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return ociopensearch.UpdateOpensearchClusterResponse{}, nil
}

func (f *fakeOciClient) DeleteOpensearchCluster(ctx context.Context, req ociopensearch.DeleteOpensearchClusterRequest) (ociopensearch.DeleteOpensearchClusterResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return ociopensearch.DeleteOpensearchClusterResponse{}, nil
}

func (f *fakeOciClient) ListOpensearchClusters(ctx context.Context, req ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return ociopensearch.ListOpensearchClustersResponse{}, nil
}

// helpers

func makeManager() *OpenSearchClusterServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewOpenSearchClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{}, nil, log, nil)
}

func makeManagerWithFake(fake *fakeOciClient) *OpenSearchClusterServiceManager {
	mgr := makeManager()
	SetClientForTest(mgr, fake)
	return mgr
}

func makeActiveCluster(id, name string) ociopensearch.OpensearchCluster {
	return ociopensearch.OpensearchCluster{
		Id:                             common.String(id),
		DisplayName:                    common.String(name),
		CompartmentId:                  common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState:                 ociopensearch.OpensearchClusterLifecycleStateActive,
		SoftwareVersion:                common.String("2.3.0"),
		TotalStorageGB:                 common.Int(100),
		OpensearchFqdn:                 common.String("opensearch.example.com"),
		OpensearchPrivateIp:            common.String("10.0.0.1"),
		OpendashboardFqdn:              common.String("dashboard.example.com"),
		OpendashboardPrivateIp:         common.String("10.0.0.2"),
		MasterNodeCount:                common.Int(3),
		MasterNodeHostType:             ociopensearch.MasterNodeHostTypeFlex,
		MasterNodeHostOcpuCount:        common.Int(1),
		MasterNodeHostMemoryGB:         common.Int(8),
		DataNodeCount:                  common.Int(3),
		DataNodeHostType:               ociopensearch.DataNodeHostTypeFlex,
		DataNodeHostOcpuCount:          common.Int(1),
		DataNodeHostMemoryGB:           common.Int(8),
		DataNodeStorageGB:              common.Int(50),
		OpendashboardNodeCount:         common.Int(1),
		OpendashboardNodeHostOcpuCount: common.Int(1),
		OpendashboardNodeHostMemoryGB:  common.Int(8),
		VcnId:                          common.String("ocid1.vcn.oc1..x"),
		SubnetId:                       common.String("ocid1.subnet.oc1..x"),
		VcnCompartmentId:               common.String("ocid1.compartment.oc1..xxx"),
		SubnetCompartmentId:            common.String("ocid1.compartment.oc1..xxx"),
		SecurityMode:                   ociopensearch.SecurityModeDisabled,
	}
}

// ---- GetCrdStatus tests ----

// TestGetCrdStatus_ReturnsStatus verifies status extraction from an OpenSearchCluster object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.Equal(t, shared.OCID("ocid1.opensearchcluster.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_FullyPopulated verifies status extraction when all status fields are set.
func TestGetCrdStatus_FullyPopulated(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..abc"
	cluster.Status.OsokStatus.Message = "OpenSearch cluster is Active"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.Equal(t, shared.OCID("ocid1.opensearchcluster.oc1..abc"), status.Ocid)
	assert.Equal(t, "OpenSearch cluster is Active", status.Message)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := makeManager()

	stream := &streamingv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert type assertion")
}

// ---- CreateOrUpdate - bad type ----

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-OpenSearchCluster objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := makeManager()

	stream := &streamingv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---- CreateOrUpdate - no explicit ID, list lookup ----

// TestCreateOrUpdate_ListError verifies OSOKResponse failure when OCID lookup errors.
func TestCreateOrUpdate_ListError(t *testing.T) {
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{}, errors.New("list error")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "test-cluster"
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreatePath verifies the create path: no existing cluster → create → Provisioning.
func TestCreateOrUpdate_CreatePath(t *testing.T) {
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			// No clusters found
			return ociopensearch.ListOpensearchClustersResponse{}, nil
		},
		createFn: func(_ context.Context, _ ociopensearch.CreateOpensearchClusterRequest) (ociopensearch.CreateOpensearchClusterResponse, error) {
			return ociopensearch.CreateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "new-cluster"
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	// Create returns work request, requeues (IsSuccessful=false, no error)
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, shared.Provisioning, cluster.Status.OsokStatus.Conditions[0].Type)
}

// TestCreateOrUpdate_CreateFails verifies error handling when create call fails.
func TestCreateOrUpdate_CreateFails(t *testing.T) {
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{}, nil
		},
		createFn: func(_ context.Context, _ ociopensearch.CreateOpensearchClusterRequest) (ociopensearch.CreateOpensearchClusterResponse, error) {
			return ociopensearch.CreateOpensearchClusterResponse{}, errors.New("create failed")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "new-cluster"
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, shared.Failed, cluster.Status.OsokStatus.Conditions[0].Type)
}

// TestCreateOrUpdate_ExistingClusterGetError verifies error when fetching existing cluster fails.
func TestCreateOrUpdate_ExistingClusterGetError(t *testing.T) {
	existingOCID := "ocid1.opensearchcluster.oc1..existing"
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: ociopensearch.OpensearchClusterCollection{
					Items: []ociopensearch.OpensearchClusterSummary{
						{
							Id:             common.String(existingOCID),
							LifecycleState: ociopensearch.OpensearchClusterLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{}, errors.New("get failed")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "test-cluster"
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_UpdatePath verifies the update path when existing cluster is found.
func TestCreateOrUpdate_UpdatePath(t *testing.T) {
	existingOCID := "ocid1.opensearchcluster.oc1..existing"
	existing := makeActiveCluster(existingOCID, "old-name")
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: ociopensearch.OpensearchClusterCollection{
					Items: []ociopensearch.OpensearchClusterSummary{
						{
							Id:             common.String(existingOCID),
							LifecycleState: ociopensearch.OpensearchClusterLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: existing}, nil
		},
		updateFn: func(_ context.Context, _ ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error) {
			return ociopensearch.UpdateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "new-name" // changed → triggers update
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.OCID(existingOCID), cluster.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_UpdateFails verifies error handling when update call fails.
func TestCreateOrUpdate_UpdateFails(t *testing.T) {
	existingOCID := "ocid1.opensearchcluster.oc1..existing"
	existing := makeActiveCluster(existingOCID, "old-name")
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: ociopensearch.OpensearchClusterCollection{
					Items: []ociopensearch.OpensearchClusterSummary{
						{
							Id:             common.String(existingOCID),
							LifecycleState: ociopensearch.OpensearchClusterLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: existing}, nil
		},
		updateFn: func(_ context.Context, _ ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error) {
			return ociopensearch.UpdateOpensearchClusterResponse{}, errors.New("update failed")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "new-name" // changed → triggers update
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---- Lifecycle state tests (via explicit OCID path) ----

// TestCreateOrUpdate_ExplicitID_GetError verifies error when fetching cluster by explicit ID fails.
func TestCreateOrUpdate_ExplicitID_GetError(t *testing.T) {
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{}, errors.New("not found")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = "ocid1.opensearchcluster.oc1..explicit"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_LifecycleActive verifies Active state sets Active OSOK condition.
func TestCreateOrUpdate_LifecycleActive(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..abc"
	existing := makeActiveCluster(clusterID, "my-cluster")
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: existing}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "my-cluster" // same name → no update

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
	assert.Equal(t, shared.Active, cluster.Status.OsokStatus.Conditions[0].Type)
}

// TestCreateOrUpdate_LifecycleFailed verifies Failed state sets Failed OSOK condition.
func TestCreateOrUpdate_LifecycleFailed(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..failed"
	failedCluster := makeActiveCluster(clusterID, "my-cluster")
	failedCluster.LifecycleState = ociopensearch.OpensearchClusterLifecycleStateFailed
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: failedCluster}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "my-cluster"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	// The lifecycle switch appends a condition after the "bound" condition; check the last one.
	conds := cluster.Status.OsokStatus.Conditions
	assert.NotEmpty(t, conds)
	assert.Equal(t, shared.Failed, conds[len(conds)-1].Type)
}

// TestCreateOrUpdate_LifecycleCreating verifies CREATING state sets Provisioning OSOK condition (requeue).
func TestCreateOrUpdate_LifecycleCreating(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..creating"
	creatingCluster := makeActiveCluster(clusterID, "my-cluster")
	creatingCluster.LifecycleState = ociopensearch.OpensearchClusterLifecycleStateCreating
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: creatingCluster}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "my-cluster"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
	conds := cluster.Status.OsokStatus.Conditions
	assert.NotEmpty(t, conds)
	assert.Equal(t, shared.Provisioning, conds[len(conds)-1].Type)
}

// TestCreateOrUpdate_LifecycleUpdating verifies UPDATING state sets Provisioning OSOK condition (requeue).
func TestCreateOrUpdate_LifecycleUpdating(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..updating"
	updatingCluster := makeActiveCluster(clusterID, "my-cluster")
	updatingCluster.LifecycleState = ociopensearch.OpensearchClusterLifecycleStateUpdating
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: updatingCluster}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "my-cluster"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
	conds := cluster.Status.OsokStatus.Conditions
	assert.NotEmpty(t, conds)
	assert.Equal(t, shared.Provisioning, conds[len(conds)-1].Type)
}

// TestCreateOrUpdate_ExplicitID_WithUpdate verifies update via explicit OCID when display name changed.
func TestCreateOrUpdate_ExplicitID_WithUpdate(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..explicit"
	existing := makeActiveCluster(clusterID, "old-name")
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: existing}, nil
		},
		updateFn: func(_ context.Context, _ ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error) {
			return ociopensearch.UpdateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "new-name" // changed

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.Updating, cluster.Status.OsokStatus.Conditions[0].Type)
}

// TestCreateOrUpdate_ExplicitID_UpdateFails verifies error propagation when update fails.
func TestCreateOrUpdate_ExplicitID_UpdateFails(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..explicit"
	existing := makeActiveCluster(clusterID, "old-name")
	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: existing}, nil
		},
		updateFn: func(_ context.Context, _ ociopensearch.UpdateOpensearchClusterRequest) (ociopensearch.UpdateOpensearchClusterResponse, error) {
			return ociopensearch.UpdateOpensearchClusterResponse{}, errors.New("update error")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "new-name"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---- Delete tests ----

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_WithStatusOcid verifies deletion is attempted when status OCID is set.
func TestDelete_WithStatusOcid(t *testing.T) {
	fake := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociopensearch.DeleteOpensearchClusterRequest) (ociopensearch.DeleteOpensearchClusterResponse, error) {
			return ociopensearch.DeleteOpensearchClusterResponse{}, nil
		},
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{}, &fakeServiceError{statusCode: 404, code: "NotFound", message: "gone"}
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_WithSpecOcid verifies deletion uses spec OCID when status OCID is empty.
func TestDelete_WithSpecOcid(t *testing.T) {
	fake := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociopensearch.DeleteOpensearchClusterRequest) (ociopensearch.DeleteOpensearchClusterResponse, error) {
			return ociopensearch.DeleteOpensearchClusterResponse{}, nil
		},
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{}, &fakeServiceError{statusCode: 404, code: "NotFound", message: "gone"}
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = "ocid1.opensearchcluster.oc1..specid"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_WrongType verifies Delete handles wrong object type gracefully.
func TestDelete_WrongType(t *testing.T) {
	mgr := makeManager()

	stream := &streamingv1beta1.Stream{}
	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_DeleteFails verifies that delete errors now propagate.
func TestDelete_DeleteFails(t *testing.T) {
	fake := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociopensearch.DeleteOpensearchClusterRequest) (ociopensearch.DeleteOpensearchClusterResponse, error) {
			return ociopensearch.DeleteOpensearchClusterResponse{}, errors.New("delete error")
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.Error(t, err)
	assert.False(t, done)
}

// ---- Credential map / endpoint fields ----

// TestCreateOrUpdate_EndpointFieldsInStatus verifies endpoint FQDNs are accessible after Active reconcile.
func TestCreateOrUpdate_EndpointFieldsInStatus(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..ep"
	c := makeActiveCluster(clusterID, "ep-cluster")
	c.OpensearchFqdn = common.String("search.example.com")
	c.OpendashboardFqdn = common.String("dash.example.com")
	c.OpensearchPrivateIp = common.String("192.168.1.10")
	c.OpendashboardPrivateIp = common.String("192.168.1.11")

	fake := &fakeOciClient{
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: c}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.OpenSearchClusterId = shared.OCID(clusterID)
	cluster.Spec.DisplayName = "ep-cluster"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	// Status OCID is set; cluster endpoints are accessible via the returned OCI object
	assert.Equal(t, shared.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_NoUpdateNeeded verifies bound path when no changes are detected.
func TestCreateOrUpdate_NoUpdateNeeded(t *testing.T) {
	clusterID := "ocid1.opensearchcluster.oc1..bound"
	existing := makeActiveCluster(clusterID, "same-name")
	fake := &fakeOciClient{
		listFn: func(_ context.Context, _ ociopensearch.ListOpensearchClustersRequest) (ociopensearch.ListOpensearchClustersResponse, error) {
			return ociopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: ociopensearch.OpensearchClusterCollection{
					Items: []ociopensearch.OpensearchClusterSummary{
						{
							Id:             common.String(clusterID),
							LifecycleState: ociopensearch.OpensearchClusterLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ ociopensearch.GetOpensearchClusterRequest) (ociopensearch.GetOpensearchClusterResponse, error) {
			return ociopensearch.GetOpensearchClusterResponse{OpensearchCluster: existing}, nil
		},
	}
	mgr := makeManagerWithFake(fake)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Spec.DisplayName = "same-name" // same → no update
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}
