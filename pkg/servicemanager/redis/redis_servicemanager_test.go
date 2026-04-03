//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociredis "github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/redis"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	getSecretFn    func(ctx context.Context, name, ns string) (map[string][]byte, error)
	updateSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

type fakeServiceError struct {
	statusCode int
	code       string
	message    string
}

func (e fakeServiceError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.statusCode, e.code, e.message)
}
func (e fakeServiceError) GetHTTPStatusCode() int  { return e.statusCode }
func (e fakeServiceError) GetMessage() string      { return e.message }
func (e fakeServiceError) GetCode() string         { return e.code }
func (e fakeServiceError) GetOpcRequestID() string { return "opc-request-id" }

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func makeActiveRedisCluster(id, displayName string) ociredis.RedisCluster {
	return ociredis.RedisCluster{
		Id:                        common.String(id),
		DisplayName:               common.String(displayName),
		LifecycleState:            ociredis.RedisClusterLifecycleStateActive,
		PrimaryFqdn:               common.String("primary.redis.example.com"),
		PrimaryEndpointIpAddress:  common.String("10.0.0.1"),
		ReplicasFqdn:              common.String("replicas.redis.example.com"),
		ReplicasEndpointIpAddress: common.String("10.0.0.2"),
		NodeCount:                 common.Int(3),
		NodeMemoryInGBs:           common.Float32(16.0),
		SoftwareVersion:           ociredis.RedisClusterSoftwareVersionV705,
		SubnetId:                  common.String("ocid1.subnet.oc1..xxx"),
		CompartmentId:             common.String("ocid1.compartment.oc1..xxx"),
		NodeCollection:            &ociredis.NodeCollection{},
	}
}

// TestGetCredentialMap verifies the secret credential map is built correctly from a RedisCluster.
func TestGetCredentialMap(t *testing.T) {
	cluster := makeActiveRedisCluster("ocid1.redis.xxx", "test-cluster")
	credMap := GetCredentialMapForTest(cluster)

	assert.Equal(t, "primary.redis.example.com", string(credMap["primaryFqdn"]))
	assert.Equal(t, "10.0.0.1", string(credMap["primaryEndpointIpAddress"]))
	assert.Equal(t, "replicas.redis.example.com", string(credMap["replicasFqdn"]))
	assert.Equal(t, "10.0.0.2", string(credMap["replicasEndpointIpAddress"]))
}

// TestGetCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetCredentialMap_NilFields(t *testing.T) {
	cluster := ociredis.RedisCluster{
		Id:             common.String("ocid1.redis.xxx"),
		DisplayName:    common.String("empty-cluster"),
		NodeCollection: &ociredis.NodeCollection{},
	}
	credMap := GetCredentialMapForTest(cluster)
	// nil fields should not appear in the map
	assert.NotContains(t, credMap, "primaryFqdn")
	assert.NotContains(t, credMap, "replicasFqdn")
}

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when OCID is empty")
}

// TestDelete_SecretNotFound verifies Delete ignores missing Redis secrets.
func TestDelete_SecretNotFound(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, apierrors.NewNotFound(corev1.Resource("secret"), "test-cluster")
		},
	}
	ociCl := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error) {
			return ociredis.DeleteRedisClusterResponse{}, nil
		},
		getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{}, &fakeServiceError{statusCode: 404, code: "NotFound", message: "gone"}
		},
	}
	mgr := newMgrWithFakeClient(ociCl, credClient)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = "ocid1.redis.oc1..xxx"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should be skipped when the secret is already missing")
}

// TestDelete_SecretError verifies Delete still fails for non-NotFound secret errors.
func TestDelete_SecretError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return servicemanager.AddManagedSecretData(map[string][]byte{}, "RedisCluster", "test-cluster"), nil
		},
		deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, errors.New("secret delete failed")
		},
	}
	ociCl := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error) {
			return ociredis.DeleteRedisClusterResponse{}, nil
		},
		getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{}, &fakeServiceError{statusCode: 404, code: "NotFound", message: "gone"}
		},
	}
	mgr := newMgrWithFakeClient(ociCl, credClient)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = "ocid1.redis.oc1..xxx"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.Error(t, err)
	assert.False(t, done)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a RedisCluster object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.redis.xxx"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.Equal(t, shared.OCID("ocid1.redis.xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &streamingv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-RedisCluster objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &streamingv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// fakeOciClient implements RedisClusterClientInterface for testing.
type fakeOciClient struct {
	createFn            func(ctx context.Context, req ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error)
	getFn               func(ctx context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error)
	listFn              func(ctx context.Context, req ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error)
	changeCompartmentFn func(ctx context.Context, req ociredis.ChangeRedisClusterCompartmentRequest) (ociredis.ChangeRedisClusterCompartmentResponse, error)
	updateFn            func(ctx context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error)
	deleteFn            func(ctx context.Context, req ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error)

	updateCalled bool
}

func (f *fakeOciClient) CreateRedisCluster(ctx context.Context, req ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return ociredis.CreateRedisClusterResponse{}, nil
}

func (f *fakeOciClient) GetRedisCluster(ctx context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return ociredis.GetRedisClusterResponse{}, nil
}

func (f *fakeOciClient) ListRedisClusters(ctx context.Context, req ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return ociredis.ListRedisClustersResponse{}, nil
}

func (f *fakeOciClient) ChangeRedisClusterCompartment(ctx context.Context, req ociredis.ChangeRedisClusterCompartmentRequest) (ociredis.ChangeRedisClusterCompartmentResponse, error) {
	if f.changeCompartmentFn != nil {
		return f.changeCompartmentFn(ctx, req)
	}
	return ociredis.ChangeRedisClusterCompartmentResponse{}, nil
}

func (f *fakeOciClient) UpdateRedisCluster(ctx context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
	f.updateCalled = true
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return ociredis.UpdateRedisClusterResponse{}, nil
}

func (f *fakeOciClient) DeleteRedisCluster(ctx context.Context, req ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return ociredis.DeleteRedisClusterResponse{}, nil
}

func newMgrWithFakeClient(ociCl *fakeOciClient, credCl *fakeCredentialClient) *RedisClusterServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credCl, nil, log)
	ExportSetClientForTest(mgr, ociCl)
	return mgr
}

// TestGetRedisClusterOcid exercises the display-name lookup paths.
func TestGetRedisClusterOcid(t *testing.T) {
	tests := []struct {
		name       string
		listItems  []ociredis.RedisClusterSummary
		listErr    error
		wantOcid   bool
		wantErrMsg string
	}{
		{
			name: "found_active",
			listItems: []ociredis.RedisClusterSummary{
				{Id: common.String("ocid1.redis.active"), LifecycleState: ociredis.RedisClusterLifecycleStateActive},
			},
			wantOcid: true,
		},
		{
			name: "found_creating",
			listItems: []ociredis.RedisClusterSummary{
				{Id: common.String("ocid1.redis.creating"), LifecycleState: ociredis.RedisClusterLifecycleStateCreating},
			},
			wantOcid: true,
		},
		{
			name: "found_updating",
			listItems: []ociredis.RedisClusterSummary{
				{Id: common.String("ocid1.redis.updating"), LifecycleState: ociredis.RedisClusterLifecycleStateUpdating},
			},
			wantOcid: true,
		},
		{
			name:      "not_found_empty_list",
			listItems: []ociredis.RedisClusterSummary{},
			wantOcid:  false,
		},
		{
			name: "not_found_failed_state",
			listItems: []ociredis.RedisClusterSummary{
				{Id: common.String("ocid1.redis.failed"), LifecycleState: ociredis.RedisClusterLifecycleStateFailed},
			},
			wantOcid: false,
		},
		{
			name:       "api_error",
			listErr:    errors.New("OCI list error"),
			wantErrMsg: "OCI list error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ociCl := &fakeOciClient{
				listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
					if tc.listErr != nil {
						return ociredis.ListRedisClustersResponse{}, tc.listErr
					}
					return ociredis.ListRedisClustersResponse{
						RedisClusterCollection: ociredis.RedisClusterCollection{Items: tc.listItems},
					}, nil
				},
			}
			mgr := newMgrWithFakeClient(ociCl, &fakeCredentialClient{})

			cluster := &ociv1beta1.RedisCluster{}
			cluster.Spec.DisplayName = "test-cluster"
			cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

			ocid, err := mgr.GetRedisClusterOcid(context.Background(), *cluster)
			if tc.wantErrMsg != "" {
				assert.ErrorContains(t, err, tc.wantErrMsg)
				assert.Nil(t, ocid)
			} else {
				assert.NoError(t, err)
				if tc.wantOcid {
					assert.NotNil(t, ocid)
				} else {
					assert.Nil(t, ocid)
				}
			}
		})
	}
}

func TestCreateOrUpdate_StatusOcidUsesUpdatePath(t *testing.T) {
	clusterID := "ocid1.redis.oc1..tracked"
	var updatedID string
	ociCl := &fakeOciClient{
		getFn: func(_ context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: makeActiveRedisCluster(*req.RedisClusterId, "old-redis")}, nil
		},
		updateFn: func(_ context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
			updatedID = *req.RedisClusterId
			return ociredis.UpdateRedisClusterResponse{}, nil
		},
	}
	mgr := newMgrWithFakeClient(ociCl, &fakeCredentialClient{})
	cluster := makeRedisSpec("new-redis")
	cluster.Status.OsokStatus.Ocid = shared.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, clusterID, updatedID)
}

func TestUpdateRedisCluster_SendsCompartmentMove(t *testing.T) {
	clusterID := "ocid1.redis.oc1..move"
	var moved ociredis.ChangeRedisClusterCompartmentRequest
	ociCl := &fakeOciClient{
		getFn: func(_ context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			cluster := makeActiveRedisCluster(*req.RedisClusterId, "redis")
			cluster.CompartmentId = common.String("ocid1.compartment.oc1..old")
			return ociredis.GetRedisClusterResponse{RedisCluster: cluster}, nil
		},
		changeCompartmentFn: func(_ context.Context, req ociredis.ChangeRedisClusterCompartmentRequest) (ociredis.ChangeRedisClusterCompartmentResponse, error) {
			moved = req
			return ociredis.ChangeRedisClusterCompartmentResponse{}, nil
		},
	}
	mgr := newMgrWithFakeClient(ociCl, &fakeCredentialClient{})
	cluster := makeRedisSpec("redis")
	cluster.Status.OsokStatus.Ocid = shared.OCID(clusterID)
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	err := mgr.UpdateRedisCluster(context.Background(), cluster)
	assert.NoError(t, err)
	assert.Equal(t, clusterID, *moved.RedisClusterId)
	assert.Equal(t, string(cluster.Spec.CompartmentId), *moved.CompartmentId)
}

// TestCreateOrUpdate_CreateNew covers the no-OCID create path.
func TestCreateOrUpdate_CreateNew(t *testing.T) {
	activeCluster := makeActiveRedisCluster("ocid1.redis.new", "new-cluster")

	tests := []struct {
		name        string
		createErr   error
		getErr      error
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "success",
			wantSuccess: true,
		},
		{
			name:      "create_oci_error",
			createErr: errors.New("OCI create failed"),
			wantErr:   true,
		},
		{
			name:    "get_after_create_error",
			getErr:  errors.New("OCI get failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credCl := &fakeCredentialClient{}
			ociCl := &fakeOciClient{
				listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
					return ociredis.ListRedisClustersResponse{
						RedisClusterCollection: ociredis.RedisClusterCollection{Items: nil},
					}, nil
				},
				createFn: func(_ context.Context, _ ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error) {
					if tc.createErr != nil {
						return ociredis.CreateRedisClusterResponse{}, tc.createErr
					}
					return ociredis.CreateRedisClusterResponse{
						RedisCluster: ociredis.RedisCluster{Id: common.String("ocid1.redis.new")},
					}, nil
				},
				getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
					if tc.getErr != nil {
						return ociredis.GetRedisClusterResponse{}, tc.getErr
					}
					return ociredis.GetRedisClusterResponse{RedisCluster: activeCluster}, nil
				},
			}

			mgr := newMgrWithFakeClient(ociCl, credCl)
			cluster := &ociv1beta1.RedisCluster{}
			cluster.Name = "new-cluster"
			cluster.Namespace = "default"
			cluster.Spec.DisplayName = "new-cluster"
			cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

			resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
			if tc.wantErr {
				assert.Error(t, err)
				assert.False(t, resp.IsSuccessful)
			} else {
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.True(t, credCl.createCalled, "CreateSecret should be called on successful create")
			}
		})
	}
}

// TestCreateOrUpdate_Update covers the existing-OCID update path.
func TestCreateOrUpdate_Update(t *testing.T) {
	existingCluster := ociredis.RedisCluster{
		Id:              common.String("ocid1.redis.existing"),
		DisplayName:     common.String("old-name"),
		NodeCount:       common.Int(3),
		NodeMemoryInGBs: common.Float32(16.0),
		LifecycleState:  ociredis.RedisClusterLifecycleStateActive,
	}

	tests := []struct {
		name        string
		specName    string
		specNodes   int
		wantUpdate  bool
		updateErr   error
		wantErr     bool
		wantSuccess bool
	}{
		{
			name:        "display_name_changed",
			specName:    "new-name",
			specNodes:   3,
			wantUpdate:  true,
			wantSuccess: true,
		},
		{
			name:        "no_changes_no_op",
			specName:    "old-name",
			specNodes:   3,
			wantUpdate:  false,
			wantSuccess: true,
		},
		{
			name:       "update_oci_error",
			specName:   "new-name",
			specNodes:  3,
			wantUpdate: true,
			updateErr:  errors.New("OCI update failed"),
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credCl := &fakeCredentialClient{}
			ociCl := &fakeOciClient{
				getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
					return ociredis.GetRedisClusterResponse{RedisCluster: existingCluster}, nil
				},
				updateFn: func(_ context.Context, _ ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
					if tc.updateErr != nil {
						return ociredis.UpdateRedisClusterResponse{}, tc.updateErr
					}
					return ociredis.UpdateRedisClusterResponse{}, nil
				},
			}

			mgr := newMgrWithFakeClient(ociCl, credCl)
			cluster := &ociv1beta1.RedisCluster{}
			cluster.Name = "test-cluster"
			cluster.Namespace = "default"
			cluster.Spec.DisplayName = tc.specName
			cluster.Spec.NodeCount = tc.specNodes
			cluster.Spec.NodeMemoryInGBs = 16.0
			// Set RedisClusterId to trigger the update (bind) path, not create.
			cluster.Spec.RedisClusterId = "ocid1.redis.existing"
			cluster.Status.OsokStatus.Ocid = "ocid1.redis.existing"

			resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
			assert.Equal(t, tc.wantUpdate, ociCl.updateCalled, "update call mismatch")
			if tc.wantErr {
				assert.Error(t, err)
				assert.False(t, resp.IsSuccessful)
			} else {
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
			}
		})
	}
}

// TestCreateOrUpdate_SecretWrite verifies secret handling on successful create.
func TestCreateOrUpdate_SecretWrite(t *testing.T) {
	activeCluster := makeActiveRedisCluster("ocid1.redis.new", "test-cluster")

	tests := []struct {
		name        string
		secretErr   error
		wantSuccess bool
	}{
		{
			name:        "secret_created_successfully",
			wantSuccess: true,
		},
		{
			name:        "secret_already_exists",
			secretErr:   apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, "test-cluster"),
			wantSuccess: true,
		},
		{
			name:        "secret_write_error",
			secretErr:   errors.New("secret write failed"),
			wantSuccess: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credCl := &fakeCredentialClient{
				createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
					return tc.secretErr == nil, tc.secretErr
				},
				getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
					return servicemanager.AddManagedSecretData(GetCredentialMapForTest(activeCluster), "RedisCluster", "test-cluster"), nil
				},
			}
			ociCl := &fakeOciClient{
				listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
					return ociredis.ListRedisClustersResponse{
						RedisClusterCollection: ociredis.RedisClusterCollection{Items: nil},
					}, nil
				},
				createFn: func(_ context.Context, _ ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error) {
					return ociredis.CreateRedisClusterResponse{
						RedisCluster: ociredis.RedisCluster{Id: common.String("ocid1.redis.new")},
					}, nil
				},
				getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
					return ociredis.GetRedisClusterResponse{RedisCluster: activeCluster}, nil
				},
			}

			mgr := newMgrWithFakeClient(ociCl, credCl)
			cluster := &ociv1beta1.RedisCluster{}
			cluster.Name = "test-cluster"
			cluster.Namespace = "default"
			cluster.Spec.DisplayName = "test-cluster"
			cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

			resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
			assert.Equal(t, tc.wantSuccess, resp.IsSuccessful)
			if tc.wantSuccess {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetRetryPolicy verifies the retry policy behavior.
func TestGetRetryPolicy(t *testing.T) {
	mgr := newMgrWithFakeClient(&fakeOciClient{}, &fakeCredentialClient{})
	policy := ExportGetRetryPolicyForTest(mgr, 30)

	t.Run("max_attempts_30", func(t *testing.T) {
		assert.Equal(t, uint(30), policy.MaximumNumberAttempts)
	})

	t.Run("should_retry_creating_state", func(t *testing.T) {
		resp := ociredis.GetRedisClusterResponse{}
		resp.LifecycleState = ociredis.RedisClusterLifecycleStateCreating
		opResp := common.OCIOperationResponse{Response: resp}
		assert.True(t, policy.ShouldRetryOperation(opResp), "should retry when CREATING")
	})

	t.Run("should_not_retry_active_state", func(t *testing.T) {
		resp := ociredis.GetRedisClusterResponse{}
		resp.LifecycleState = ociredis.RedisClusterLifecycleStateActive
		opResp := common.OCIOperationResponse{Response: resp}
		assert.False(t, policy.ShouldRetryOperation(opResp), "should not retry when ACTIVE")
	})

	t.Run("should_retry_non_redis_response", func(t *testing.T) {
		opResp := common.OCIOperationResponse{Response: nil}
		assert.True(t, policy.ShouldRetryOperation(opResp), "should retry for unknown response type")
	})

	t.Run("next_duration_is_one_minute", func(t *testing.T) {
		opResp := common.OCIOperationResponse{}
		assert.Equal(t, time.Minute, policy.NextDuration(opResp))
	})
}
