/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package bdsinstance

import (
	"context"
	"testing"
	"time"

	bdssdk "github.com/oracle/oci-go-sdk/v65/bds"
	"github.com/oracle/oci-go-sdk/v65/common"
	bdsv1beta1 "github.com/oracle/oci-service-operator/api/bds/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type bdsInstanceOCIClient interface {
	CreateBdsInstance(context.Context, bdssdk.CreateBdsInstanceRequest) (bdssdk.CreateBdsInstanceResponse, error)
	GetBdsInstance(context.Context, bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error)
	ListBdsInstances(context.Context, bdssdk.ListBdsInstancesRequest) (bdssdk.ListBdsInstancesResponse, error)
	UpdateBdsInstance(context.Context, bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error)
	DeleteBdsInstance(context.Context, bdssdk.DeleteBdsInstanceRequest) (bdssdk.DeleteBdsInstanceResponse, error)
}

type fakeBdsInstanceOCIClient struct {
	createFn func(context.Context, bdssdk.CreateBdsInstanceRequest) (bdssdk.CreateBdsInstanceResponse, error)
	getFn    func(context.Context, bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error)
	listFn   func(context.Context, bdssdk.ListBdsInstancesRequest) (bdssdk.ListBdsInstancesResponse, error)
	updateFn func(context.Context, bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error)
	deleteFn func(context.Context, bdssdk.DeleteBdsInstanceRequest) (bdssdk.DeleteBdsInstanceResponse, error)
}

func (f *fakeBdsInstanceOCIClient) CreateBdsInstance(ctx context.Context, req bdssdk.CreateBdsInstanceRequest) (bdssdk.CreateBdsInstanceResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return bdssdk.CreateBdsInstanceResponse{}, nil
}

func (f *fakeBdsInstanceOCIClient) GetBdsInstance(ctx context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return bdssdk.GetBdsInstanceResponse{}, nil
}

func (f *fakeBdsInstanceOCIClient) ListBdsInstances(ctx context.Context, req bdssdk.ListBdsInstancesRequest) (bdssdk.ListBdsInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return bdssdk.ListBdsInstancesResponse{}, nil
}

func (f *fakeBdsInstanceOCIClient) UpdateBdsInstance(ctx context.Context, req bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return bdssdk.UpdateBdsInstanceResponse{}, nil
}

func (f *fakeBdsInstanceOCIClient) DeleteBdsInstance(ctx context.Context, req bdssdk.DeleteBdsInstanceRequest) (bdssdk.DeleteBdsInstanceResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return bdssdk.DeleteBdsInstanceResponse{}, nil
}

func newBdsInstanceTestManager(client bdsInstanceOCIClient) *BdsInstanceServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewBdsInstanceServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		hooks := newBdsInstanceDefaultRuntimeHooks(bdssdk.BdsClient{})
		hooks.Create.Call = func(ctx context.Context, request bdssdk.CreateBdsInstanceRequest) (bdssdk.CreateBdsInstanceResponse, error) {
			return client.CreateBdsInstance(ctx, request)
		}
		hooks.Get.Call = func(ctx context.Context, request bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
			return client.GetBdsInstance(ctx, request)
		}
		hooks.List.Call = func(ctx context.Context, request bdssdk.ListBdsInstancesRequest) (bdssdk.ListBdsInstancesResponse, error) {
			return client.ListBdsInstances(ctx, request)
		}
		hooks.Update.Call = func(ctx context.Context, request bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error) {
			return client.UpdateBdsInstance(ctx, request)
		}
		hooks.Delete.Call = func(ctx context.Context, request bdssdk.DeleteBdsInstanceRequest) (bdssdk.DeleteBdsInstanceResponse, error) {
			return client.DeleteBdsInstance(ctx, request)
		}
		applyBdsInstanceRuntimeHooks(&hooks)
		manager.WithClient(defaultBdsInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*bdsv1beta1.BdsInstance](buildBdsInstanceGeneratedRuntimeConfig(manager, hooks)),
		})
	}
	return manager
}

func makeSpecBdsInstance() *bdsv1beta1.BdsInstance {
	return &bdsv1beta1.BdsInstance{
		Spec: bdsv1beta1.BdsInstanceSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "test-bds",
			ClusterVersion:       string(bdssdk.BdsInstanceClusterVersionOdh20),
			ClusterPublicKey:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCy",
			ClusterAdminPassword: "c2VjcmV0",
			IsHighAvailability:   false,
			IsSecure:             false,
			BootstrapScriptUrl:   "",
			KmsKeyId:             "",
			FreeformTags:         map[string]string{"run": "1"},
			DefinedTags:          map[string]shared.MapValue{},
			ClusterProfile:       "",
			NetworkConfig:        bdsv1beta1.BdsInstanceNetworkConfig{},
			Nodes: []bdsv1beta1.BdsInstanceNode{
				{
					NodeType:             string(bdssdk.NodeNodeTypeMaster),
					Shape:                "VM.Standard3.Flex",
					BlockVolumeSizeInGBs: 150,
					SubnetId:             "ocid1.subnet.oc1..example",
					ShapeConfig: bdsv1beta1.BdsInstanceNodeShapeConfig{
						Ocpus:       2,
						MemoryInGBs: 32,
						Nvmes:       1,
					},
				},
			},
		},
	}
}

func makeSDKBdsInstance(id string, lifecycle bdssdk.BdsInstanceLifecycleStateEnum) bdssdk.BdsInstance {
	created := common.SDKTime{Time: time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)}
	return bdssdk.BdsInstance{
		Id:                   common.String(id),
		CompartmentId:        common.String("ocid1.compartment.oc1..example"),
		DisplayName:          common.String("test-bds"),
		LifecycleState:       lifecycle,
		IsHighAvailability:   boolPtr(false),
		IsSecure:             boolPtr(false),
		IsCloudSqlConfigured: boolPtr(false),
		IsKafkaConfigured:    boolPtr(false),
		Nodes: []bdssdk.Node{
			makeSDKBdsNode(150),
		},
		NumberOfNodes:      intPtr(1),
		ClusterVersion:     bdssdk.BdsInstanceClusterVersionOdh20,
		TimeCreated:        &created,
		FreeformTags:       map[string]string{"run": "1"},
		DefinedTags:        map[string]map[string]interface{}{},
		ClusterProfile:     "",
		BootstrapScriptUrl: common.String(""),
		KmsKeyId:           common.String(""),
	}
}

func makeSDKBdsNode(blockVolumeSize int64) bdssdk.Node {
	created := common.SDKTime{Time: time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)}
	return bdssdk.Node{
		InstanceId:         common.String("ocid1.instance.oc1..example"),
		DisplayName:        common.String("master-1"),
		LifecycleState:     bdssdk.NodeLifecycleStateActive,
		NodeType:           bdssdk.NodeNodeTypeMaster,
		Shape:              common.String("VM.Standard3.Flex"),
		SubnetId:           common.String("ocid1.subnet.oc1..example"),
		IpAddress:          common.String("10.0.0.2"),
		SshFingerprint:     common.String("fingerprint"),
		AvailabilityDomain: common.String("AD-1"),
		FaultDomain:        common.String("FAULT-DOMAIN-1"),
		TimeCreated:        &created,
		AttachedBlockVolumes: []bdssdk.VolumeAttachmentDetail{
			{
				VolumeAttachmentId: common.String("ocid1.volumeattachment.oc1..example"),
				VolumeSizeInGBs:    int64Ptr(blockVolumeSize),
			},
		},
		Ocpus:       intPtr(2),
		MemoryInGBs: intPtr(32),
		Nvmes:       intPtr(1),
	}
}

func TestBdsInstanceCreateOrUpdate_ReviewedLifecycleProjection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		lifecycleState bdssdk.BdsInstanceLifecycleStateEnum
		wantCondition  shared.OSOKConditionType
		wantRequeue    bool
		wantSuccessful bool
		wantAsyncPhase shared.OSOKAsyncPhase
	}{
		{
			name:           "inactive is steady active",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateInactive,
			wantCondition:  shared.Active,
			wantRequeue:    false,
			wantSuccessful: true,
		},
		{
			name:           "suspended is steady active",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateSuspended,
			wantCondition:  shared.Active,
			wantRequeue:    false,
			wantSuccessful: true,
		},
		{
			name:           "suspending requeues as update",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateSuspending,
			wantCondition:  shared.Updating,
			wantRequeue:    true,
			wantSuccessful: true,
			wantAsyncPhase: shared.OSOKAsyncPhaseUpdate,
		},
		{
			name:           "resuming requeues as update",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateResuming,
			wantCondition:  shared.Updating,
			wantRequeue:    true,
			wantSuccessful: true,
			wantAsyncPhase: shared.OSOKAsyncPhaseUpdate,
		},
		{
			name:           "failed projects failure",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateFailed,
			wantCondition:  shared.Failed,
			wantRequeue:    false,
			wantSuccessful: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ocid := "ocid1.bdsinstance.oc1..lifecycle"
			updateCalls := 0

			manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
				getFn: func(_ context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
					if !assert.NotNil(t, req.BdsInstanceId) {
						return bdssdk.GetBdsInstanceResponse{}, nil
					}
					assert.Equal(t, ocid, *req.BdsInstanceId)
					current := makeSDKBdsInstance(ocid, tc.lifecycleState)
					return bdssdk.GetBdsInstanceResponse{BdsInstance: current}, nil
				},
				updateFn: func(_ context.Context, _ bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error) {
					updateCalls++
					return bdssdk.UpdateBdsInstanceResponse{}, nil
				},
			})

			resource := makeSpecBdsInstance()
			resource.Status.Id = ocid
			resource.Status.OsokStatus.Ocid = shared.OCID(ocid)

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, 0, updateCalls)
			assert.Equal(t, tc.wantSuccessful, response.IsSuccessful)
			assert.Equal(t, tc.wantRequeue, response.ShouldRequeue)
			assert.Equal(t, string(tc.lifecycleState), resource.Status.LifecycleState)
			assert.Equal(t, tc.wantCondition, trailingBdsCondition(resource))

			if tc.wantAsyncPhase == "" {
				assert.Nil(t, resource.Status.OsokStatus.Async.Current)
				return
			}

			if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
				assert.Equal(t, tc.wantAsyncPhase, resource.Status.OsokStatus.Async.Current.Phase)
				assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
			}
		})
	}
}

func TestBdsInstanceCreateOrUpdate_CreateRequestIncludesReviewedSecretInputs(t *testing.T) {
	t.Parallel()

	ocid := "ocid1.bdsinstance.oc1..created"
	createCalls := 0
	listCalls := 0
	var capturedCreate bdssdk.CreateBdsInstanceRequest

	manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
		createFn: func(_ context.Context, req bdssdk.CreateBdsInstanceRequest) (bdssdk.CreateBdsInstanceResponse, error) {
			createCalls++
			capturedCreate = req
			return bdssdk.CreateBdsInstanceResponse{}, nil
		},
		listFn: func(_ context.Context, req bdssdk.ListBdsInstancesRequest) (bdssdk.ListBdsInstancesResponse, error) {
			listCalls++
			if !assert.NotNil(t, req.CompartmentId) || !assert.NotNil(t, req.DisplayName) {
				return bdssdk.ListBdsInstancesResponse{}, nil
			}
			assert.Equal(t, "ocid1.compartment.oc1..example", *req.CompartmentId)
			assert.Equal(t, "test-bds", *req.DisplayName)
			if listCalls == 1 {
				return bdssdk.ListBdsInstancesResponse{}, nil
			}

			created := common.SDKTime{Time: time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)}
			return bdssdk.ListBdsInstancesResponse{
				Items: []bdssdk.BdsInstanceSummary{
					{
						Id:                   common.String(ocid),
						CompartmentId:        common.String("ocid1.compartment.oc1..example"),
						DisplayName:          common.String("test-bds"),
						LifecycleState:       bdssdk.BdsInstanceLifecycleStateActive,
						NumberOfNodes:        intPtr(1),
						IsHighAvailability:   boolPtr(false),
						IsSecure:             boolPtr(false),
						IsCloudSqlConfigured: boolPtr(false),
						IsKafkaConfigured:    boolPtr(false),
						TimeCreated:          &created,
						ClusterVersion:       bdssdk.BdsInstanceClusterVersionOdh20,
					},
				},
			}, nil
		},
	})

	resource := makeSpecBdsInstance()
	resource.Spec.ClusterAdminPassword = ""
	resource.Spec.SecretId = "ocid1.vaultsecret.oc1..example"
	resource.Spec.IsSecretReused = true
	resource.Spec.BdsClusterVersionSummary = bdsv1beta1.BdsInstanceBdsClusterVersionSummary{
		BdsVersion: "3.5.0",
		OdhVersion: "2.0.1",
	}

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 2, listCalls)
	if assert.NotNil(t, capturedCreate.SecretId) {
		assert.Equal(t, "ocid1.vaultsecret.oc1..example", *capturedCreate.SecretId)
	}
	if assert.NotNil(t, capturedCreate.IsSecretReused) {
		assert.True(t, *capturedCreate.IsSecretReused)
	}
	if assert.NotNil(t, capturedCreate.BdsClusterVersionSummary) {
		if assert.NotNil(t, capturedCreate.BdsClusterVersionSummary.BdsVersion) {
			assert.Equal(t, "3.5.0", *capturedCreate.BdsClusterVersionSummary.BdsVersion)
		}
		if assert.NotNil(t, capturedCreate.BdsClusterVersionSummary.OdhVersion) {
			assert.Equal(t, "2.0.1", *capturedCreate.BdsClusterVersionSummary.OdhVersion)
		}
	}
	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
}

func TestBdsInstanceCreateOrUpdate_UpdatesOnlyMutableFields(t *testing.T) {
	t.Parallel()

	ocid := "ocid1.bdsinstance.oc1..update"
	getCalls := 0
	updateCalls := 0
	var capturedUpdate bdssdk.UpdateBdsInstanceRequest

	current := makeSDKBdsInstance(ocid, bdssdk.BdsInstanceLifecycleStateActive)
	current.DisplayName = common.String("old-bds")
	current.BootstrapScriptUrl = common.String("https://example.com/bootstrap.sh")
	current.FreeformTags = map[string]string{"run": "1", "stale": "true"}
	current.DefinedTags = map[string]map[string]interface{}{
		"team": {"env": "dev"},
	}
	current.KmsKeyId = common.String("ocid1.key.oc1..old")

	refreshed := current
	refreshed.DisplayName = common.String("new-bds")
	refreshed.BootstrapScriptUrl = common.String("")
	refreshed.FreeformTags = map[string]string{}
	refreshed.DefinedTags = map[string]map[string]interface{}{}
	refreshed.KmsKeyId = common.String("")
	refreshed.LifecycleState = bdssdk.BdsInstanceLifecycleStateUpdating

	manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
		getFn: func(_ context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
			getCalls++
			if !assert.NotNil(t, req.BdsInstanceId) {
				return bdssdk.GetBdsInstanceResponse{}, nil
			}
			assert.Equal(t, ocid, *req.BdsInstanceId)
			if getCalls == 1 {
				return bdssdk.GetBdsInstanceResponse{BdsInstance: current}, nil
			}
			return bdssdk.GetBdsInstanceResponse{BdsInstance: refreshed}, nil
		},
		updateFn: func(_ context.Context, req bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error) {
			updateCalls++
			capturedUpdate = req
			return bdssdk.UpdateBdsInstanceResponse{
				OpcWorkRequestId: common.String("wr-bds-update"),
			}, nil
		},
	})

	resource := makeSpecBdsInstance()
	resource.Status.Id = ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(ocid)
	resource.Spec.DisplayName = "new-bds"
	resource.Spec.BootstrapScriptUrl = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Spec.KmsKeyId = ""

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, 1, updateCalls)
	assert.Equal(t, 2, getCalls)
	if assert.NotNil(t, capturedUpdate.BdsInstanceId) {
		assert.Equal(t, ocid, *capturedUpdate.BdsInstanceId)
	}
	if assert.NotNil(t, capturedUpdate.DisplayName) {
		assert.Equal(t, "new-bds", *capturedUpdate.DisplayName)
	}
	if assert.NotNil(t, capturedUpdate.BootstrapScriptUrl) {
		assert.Equal(t, "", *capturedUpdate.BootstrapScriptUrl)
	}
	if assert.NotNil(t, capturedUpdate.KmsKeyId) {
		assert.Equal(t, "", *capturedUpdate.KmsKeyId)
	}
	assert.Nil(t, capturedUpdate.SecretId)
	assert.Nil(t, capturedUpdate.IsSecretReused)
	assert.Empty(t, capturedUpdate.FreeformTags)
	assert.Empty(t, capturedUpdate.DefinedTags)

	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	assert.Equal(t, shared.Updating, trailingBdsCondition(resource))
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncPhaseUpdate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
		assert.Equal(t, "wr-bds-update", resource.Status.OsokStatus.Async.Current.WorkRequestID)
	}
}

func TestBdsInstanceCreateOrUpdate_ReviewedSecretDriftRequiresReplacement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		prepareSpec func(*bdsv1beta1.BdsInstance)
		prepareLive func(*bdssdk.BdsInstance)
		wantErr     string
	}{
		{
			name: "secret id drift requires replacement",
			prepareSpec: func(resource *bdsv1beta1.BdsInstance) {
				resource.Spec.ClusterAdminPassword = ""
				resource.Spec.SecretId = "ocid1.vaultsecret.oc1..desired"
			},
			prepareLive: func(current *bdssdk.BdsInstance) {
				current.SecretId = common.String("ocid1.vaultsecret.oc1..current")
			},
			wantErr: "secretId",
		},
		{
			name: "is secret reused drift requires replacement",
			prepareSpec: func(resource *bdsv1beta1.BdsInstance) {
				resource.Spec.IsSecretReused = true
			},
			prepareLive: func(current *bdssdk.BdsInstance) {
				current.IsSecretReused = boolPtr(false)
			},
			wantErr: "isSecretReused",
		},
		{
			name: "cluster version summary drift requires replacement",
			prepareSpec: func(resource *bdsv1beta1.BdsInstance) {
				resource.Spec.BdsClusterVersionSummary = bdsv1beta1.BdsInstanceBdsClusterVersionSummary{
					BdsVersion: "3.5.0",
					OdhVersion: "2.1.0",
				}
			},
			prepareLive: func(current *bdssdk.BdsInstance) {
				current.BdsClusterVersionSummary = &bdssdk.BdsClusterVersionSummary{
					BdsVersion: common.String("3.4.0"),
					OdhVersion: common.String("2.0.0"),
				}
			},
			wantErr: "bdsClusterVersionSummary",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ocid := "ocid1.bdsinstance.oc1..reviewed-drift"
			updateCalls := 0

			manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
				getFn: func(_ context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
					if !assert.NotNil(t, req.BdsInstanceId) {
						return bdssdk.GetBdsInstanceResponse{}, nil
					}
					assert.Equal(t, ocid, *req.BdsInstanceId)
					current := makeSDKBdsInstance(ocid, bdssdk.BdsInstanceLifecycleStateActive)
					tc.prepareLive(&current)
					return bdssdk.GetBdsInstanceResponse{BdsInstance: current}, nil
				},
				updateFn: func(_ context.Context, _ bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error) {
					updateCalls++
					return bdssdk.UpdateBdsInstanceResponse{}, nil
				},
			})

			resource := makeSpecBdsInstance()
			resource.Status.Id = ocid
			resource.Status.OsokStatus.Ocid = shared.OCID(ocid)
			tc.prepareSpec(resource)

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			assert.Error(t, err)
			assert.False(t, response.IsSuccessful)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Equal(t, 0, updateCalls)
			assert.Equal(t, shared.Failed, trailingBdsCondition(resource))
		})
	}
}

func TestBdsInstanceCreateOrUpdate_NodeDriftRequiresReplacement(t *testing.T) {
	t.Parallel()

	ocid := "ocid1.bdsinstance.oc1..nodes"
	updateCalls := 0

	manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
		getFn: func(_ context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
			if !assert.NotNil(t, req.BdsInstanceId) {
				return bdssdk.GetBdsInstanceResponse{}, nil
			}
			assert.Equal(t, ocid, *req.BdsInstanceId)
			current := makeSDKBdsInstance(ocid, bdssdk.BdsInstanceLifecycleStateActive)
			current.Nodes = []bdssdk.Node{makeSDKBdsNode(200)}
			return bdssdk.GetBdsInstanceResponse{BdsInstance: current}, nil
		},
		updateFn: func(_ context.Context, _ bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error) {
			updateCalls++
			return bdssdk.UpdateBdsInstanceResponse{}, nil
		},
	})

	resource := makeSpecBdsInstance()
	resource.Status.Id = ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(ocid)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, response.IsSuccessful)
	assert.Contains(t, err.Error(), "nodes")
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, shared.Failed, trailingBdsCondition(resource))
}

func TestBdsInstanceCreateOrUpdate_ProjectsRefreshedStatusFields(t *testing.T) {
	t.Parallel()

	ocid := "ocid1.bdsinstance.oc1..status"
	earliest := common.SDKTime{Time: time.Date(2026, 4, 9, 15, 30, 0, 0, time.UTC)}
	current := makeSDKBdsInstance(ocid, bdssdk.BdsInstanceLifecycleStateActive)
	current.SecretId = common.String("ocid1.vaultsecret.oc1..status")
	current.IsSecretReused = boolPtr(true)
	current.BdsClusterVersionSummary = &bdssdk.BdsClusterVersionSummary{
		BdsVersion: common.String("3.5.0"),
		OdhVersion: common.String("2.0.1"),
	}
	current.TimeEarliestCertificateExpiration = &earliest

	manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
		getFn: func(_ context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
			if !assert.NotNil(t, req.BdsInstanceId) {
				return bdssdk.GetBdsInstanceResponse{}, nil
			}
			assert.Equal(t, ocid, *req.BdsInstanceId)
			return bdssdk.GetBdsInstanceResponse{BdsInstance: current}, nil
		},
	})

	resource := makeSpecBdsInstance()
	resource.Status.Id = ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(ocid)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, "ocid1.vaultsecret.oc1..status", resource.Status.SecretId)
	assert.True(t, resource.Status.IsSecretReused)
	assert.Equal(t, "3.5.0", resource.Status.BdsClusterVersionSummary.BdsVersion)
	assert.Equal(t, "2.0.1", resource.Status.BdsClusterVersionSummary.OdhVersion)
	assert.Equal(t, earliest.Time.Format(time.RFC3339), resource.Status.TimeEarliestCertificateExpiration)
}

func TestBdsInstanceDelete_ConfirmedDeleteStates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		lifecycleState bdssdk.BdsInstanceLifecycleStateEnum
		wantDeleted    bool
		wantCondition  shared.OSOKConditionType
		wantAsync      bool
	}{
		{
			name:           "deleting requeues finalizer",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateDeleting,
			wantDeleted:    false,
			wantCondition:  shared.Terminating,
			wantAsync:      true,
		},
		{
			name:           "deleted completes finalizer",
			lifecycleState: bdssdk.BdsInstanceLifecycleStateDeleted,
			wantDeleted:    true,
			wantCondition:  shared.Terminating,
			wantAsync:      false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ocid := "ocid1.bdsinstance.oc1..delete"
			deleteCalls := 0

			manager := newBdsInstanceTestManager(&fakeBdsInstanceOCIClient{
				getFn: func(_ context.Context, req bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error) {
					if !assert.NotNil(t, req.BdsInstanceId) {
						return bdssdk.GetBdsInstanceResponse{}, nil
					}
					assert.Equal(t, ocid, *req.BdsInstanceId)
					current := makeSDKBdsInstance(ocid, tc.lifecycleState)
					return bdssdk.GetBdsInstanceResponse{BdsInstance: current}, nil
				},
				deleteFn: func(_ context.Context, _ bdssdk.DeleteBdsInstanceRequest) (bdssdk.DeleteBdsInstanceResponse, error) {
					deleteCalls++
					return bdssdk.DeleteBdsInstanceResponse{}, nil
				},
			})

			resource := makeSpecBdsInstance()
			resource.Status.Id = ocid
			resource.Status.OsokStatus.Ocid = shared.OCID(ocid)

			deleted, err := manager.Delete(context.Background(), resource)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tc.wantDeleted, deleted)
			assert.Equal(t, 0, deleteCalls)
			assert.Equal(t, tc.wantCondition, trailingBdsCondition(resource))

			if tc.wantAsync {
				if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
					assert.Equal(t, shared.OSOKAsyncPhaseDelete, resource.Status.OsokStatus.Async.Current.Phase)
					assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
				}
				return
			}

			assert.Nil(t, resource.Status.OsokStatus.Async.Current)
			assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
		})
	}
}

func trailingBdsCondition(resource *bdsv1beta1.BdsInstance) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
