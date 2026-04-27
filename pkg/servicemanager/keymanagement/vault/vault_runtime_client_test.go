/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	keymanagementsdk "github.com/oracle/oci-go-sdk/v65/keymanagement"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeVaultOCIClient struct {
	createFn   func(context.Context, keymanagementsdk.CreateVaultRequest) (keymanagementsdk.CreateVaultResponse, error)
	getFn      func(context.Context, keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error)
	listFn     func(context.Context, keymanagementsdk.ListVaultsRequest) (keymanagementsdk.ListVaultsResponse, error)
	updateFn   func(context.Context, keymanagementsdk.UpdateVaultRequest) (keymanagementsdk.UpdateVaultResponse, error)
	scheduleFn func(context.Context, keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error)
}

func (f *fakeVaultOCIClient) CreateVault(ctx context.Context, req keymanagementsdk.CreateVaultRequest) (keymanagementsdk.CreateVaultResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return keymanagementsdk.CreateVaultResponse{}, nil
}

func (f *fakeVaultOCIClient) GetVault(ctx context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return keymanagementsdk.GetVaultResponse{}, nil
}

func (f *fakeVaultOCIClient) ListVaults(ctx context.Context, req keymanagementsdk.ListVaultsRequest) (keymanagementsdk.ListVaultsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return keymanagementsdk.ListVaultsResponse{}, nil
}

func (f *fakeVaultOCIClient) UpdateVault(ctx context.Context, req keymanagementsdk.UpdateVaultRequest) (keymanagementsdk.UpdateVaultResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return keymanagementsdk.UpdateVaultResponse{}, nil
}

func (f *fakeVaultOCIClient) ScheduleVaultDeletion(ctx context.Context, req keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error) {
	if f.scheduleFn != nil {
		return f.scheduleFn(ctx, req)
	}
	return keymanagementsdk.ScheduleVaultDeletionResponse{}, nil
}

func newVaultTestManager(client vaultOCIClient) *VaultServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewVaultServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		delegate := defaultVaultServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*keymanagementv1beta1.Vault](newVaultRuntimeConfig(log, client)),
		}
		manager.WithClient(&vaultRuntimeClient{
			delegate: delegate,
			sdk:      client,
			log:      log,
		})
	}
	return manager
}

func makeSpecVault() *keymanagementv1beta1.Vault {
	return &keymanagementv1beta1.Vault{
		Spec: keymanagementv1beta1.VaultSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "vault-sample",
			VaultType:     "DEFAULT",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeSDKVault(id string, lifecycleState keymanagementsdk.VaultLifecycleStateEnum, timeOfDeletion *time.Time, displayName string, tags map[string]string) keymanagementsdk.Vault {
	created := common.SDKTime{Time: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}
	var deletion *common.SDKTime
	if timeOfDeletion != nil {
		deletion = &common.SDKTime{Time: timeOfDeletion.UTC()}
	}
	return keymanagementsdk.Vault{
		CompartmentId:      common.String("ocid1.compartment.oc1..example"),
		CryptoEndpoint:     common.String("https://crypto.example"),
		DisplayName:        common.String(displayName),
		Id:                 common.String(id),
		LifecycleState:     lifecycleState,
		ManagementEndpoint: common.String("https://management.example"),
		TimeCreated:        &created,
		VaultType:          keymanagementsdk.VaultVaultTypeDefault,
		WrappingkeyId:      common.String("ocid1.key.oc1..wrap"),
		FreeformTags:       tags,
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {
				"CostCenter": "42",
			},
		},
		TimeOfDeletion: deletion,
	}
}

func makeSDKVaultSummary(id string, lifecycleState keymanagementsdk.VaultSummaryLifecycleStateEnum, displayName string) keymanagementsdk.VaultSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}
	return keymanagementsdk.VaultSummary{
		CompartmentId:      common.String("ocid1.compartment.oc1..example"),
		CryptoEndpoint:     common.String("https://crypto.example"),
		DisplayName:        common.String(displayName),
		Id:                 common.String(id),
		LifecycleState:     lifecycleState,
		ManagementEndpoint: common.String("https://management.example"),
		TimeCreated:        &created,
		VaultType:          keymanagementsdk.VaultSummaryVaultTypeDefault,
		FreeformTags: map[string]string{
			"env": "dev",
		},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {
				"CostCenter": "42",
			},
		},
	}
}

func TestVaultRuntimeCreateOrUpdate_CreatesVault(t *testing.T) {
	vaultID := "ocid1.vault.oc1..created"
	createCalls := 0
	getCalls := 0

	manager := newVaultTestManager(&fakeVaultOCIClient{
		createFn: func(_ context.Context, req keymanagementsdk.CreateVaultRequest) (keymanagementsdk.CreateVaultResponse, error) {
			createCalls++
			assert.Equal(t, "ocid1.compartment.oc1..example", *req.CompartmentId)
			assert.Equal(t, "vault-sample", *req.DisplayName)
			assert.Equal(t, keymanagementsdk.CreateVaultDetailsVaultTypeDefault, req.VaultType)
			return keymanagementsdk.CreateVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateCreating, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			getCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecVault()

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, vaultID, resource.Status.Id)
	assert.Equal(t, vaultID, string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
	assert.Zero(t, resource.Status.RequestedDeletionScheduleDays)
}

func TestVaultRuntimeCreateOrUpdate_BindsExistingVaultFromList(t *testing.T) {
	vaultID := "ocid1.vault.oc1..existing"
	listCalls := 0
	getCalls := 0

	manager := newVaultTestManager(&fakeVaultOCIClient{
		createFn: func(_ context.Context, _ keymanagementsdk.CreateVaultRequest) (keymanagementsdk.CreateVaultResponse, error) {
			t.Fatal("CreateVault() should not be called when list lookup finds a reusable Vault")
			return keymanagementsdk.CreateVaultResponse{}, nil
		},
		listFn: func(_ context.Context, req keymanagementsdk.ListVaultsRequest) (keymanagementsdk.ListVaultsResponse, error) {
			listCalls++
			assert.Equal(t, "ocid1.compartment.oc1..example", *req.CompartmentId)
			return keymanagementsdk.ListVaultsResponse{
				Items: []keymanagementsdk.VaultSummary{
					makeSDKVaultSummary(vaultID, keymanagementsdk.VaultSummaryLifecycleStateActive, "vault-sample"),
				},
			}, nil
		},
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			getCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecVault()

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, 1, listCalls)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, vaultID, resource.Status.Id)
	assert.Equal(t, vaultID, string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
}

func TestVaultRuntimeCreateOrUpdate_UpdatesMutableFieldsInPlace(t *testing.T) {
	vaultID := "ocid1.vault.oc1..existing"
	getCalls := 0
	updateCalls := 0
	var captured keymanagementsdk.UpdateVaultRequest

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			getCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			if getCalls == 1 {
				return keymanagementsdk.GetVaultResponse{
					Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-old", map[string]string{"env": "dev"}),
				}, nil
			}
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-new", map[string]string{"env": "prod"}),
			}, nil
		},
		updateFn: func(_ context.Context, req keymanagementsdk.UpdateVaultRequest) (keymanagementsdk.UpdateVaultResponse, error) {
			updateCalls++
			captured = req
			return keymanagementsdk.UpdateVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateUpdating, nil, "vault-new", map[string]string{"env": "prod"}),
			}, nil
		},
	})

	resource := makeSpecVault()
	resource.Spec.DisplayName = "vault-new"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Status.Id = vaultID
	resource.Status.DisplayName = "vault-old"
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.FreeformTags = map[string]string{"env": "dev"}
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 1, updateCalls)
	if !assert.NotNil(t, captured.VaultId) {
		return
	}
	assert.Equal(t, vaultID, *captured.VaultId)
	assert.Equal(t, "vault-new", *captured.UpdateVaultDetails.DisplayName)
	assert.Equal(t, map[string]string{"env": "prod"}, captured.UpdateVaultDetails.FreeformTags)
	assert.Equal(t, "vault-new", resource.Status.DisplayName)
	assert.Equal(t, map[string]string{"env": "prod"}, resource.Status.FreeformTags)
}

func TestVaultRuntimeCreateOrUpdate_RejectsForceNewCompartmentDrift(t *testing.T) {
	vaultID := "ocid1.vault.oc1..existing"
	updateCalls := 0

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		updateFn: func(_ context.Context, _ keymanagementsdk.UpdateVaultRequest) (keymanagementsdk.UpdateVaultResponse, error) {
			updateCalls++
			t.Fatal("UpdateVault() should not be called when force-new compartment drift is detected")
			return keymanagementsdk.UpdateVaultResponse{}, nil
		},
	})

	resource := makeSpecVault()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
	resource.Status.Id = vaultID
	resource.Status.CompartmentId = "ocid1.compartment.oc1..example"
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "require replacement when compartmentId changes")
	assert.False(t, response.IsSuccessful)
	assert.Equal(t, 0, updateCalls)
}

func TestVaultRuntimeCreateOrUpdate_MapsLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		state         keymanagementsdk.VaultLifecycleStateEnum
		wantReason    shared.OSOKConditionType
		wantSuccess   bool
		wantRequeue   bool
		wantCondition v1.ConditionStatus
	}{
		{
			name:          "creating",
			state:         keymanagementsdk.VaultLifecycleStateCreating,
			wantReason:    shared.Provisioning,
			wantSuccess:   true,
			wantRequeue:   true,
			wantCondition: v1.ConditionTrue,
		},
		{
			name:          "updating",
			state:         keymanagementsdk.VaultLifecycleStateUpdating,
			wantReason:    shared.Updating,
			wantSuccess:   true,
			wantRequeue:   true,
			wantCondition: v1.ConditionTrue,
		},
		{
			name:          "active",
			state:         keymanagementsdk.VaultLifecycleStateActive,
			wantReason:    shared.Active,
			wantSuccess:   true,
			wantRequeue:   false,
			wantCondition: v1.ConditionTrue,
		},
		{
			name:          "unmodeled",
			state:         keymanagementsdk.VaultLifecycleStateEnum("FAILED"),
			wantReason:    shared.Failed,
			wantSuccess:   false,
			wantRequeue:   false,
			wantCondition: v1.ConditionFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultID := "ocid1.vault.oc1.." + tt.name
			updateCalls := 0
			manager := newVaultTestManager(&fakeVaultOCIClient{
				getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
					assert.Equal(t, vaultID, *req.VaultId)
					return keymanagementsdk.GetVaultResponse{
						Vault: makeSDKVault(vaultID, tt.state, nil, "vault-sample", map[string]string{"env": "dev"}),
					}, nil
				},
				updateFn: func(_ context.Context, _ keymanagementsdk.UpdateVaultRequest) (keymanagementsdk.UpdateVaultResponse, error) {
					updateCalls++
					t.Fatal("UpdateVault() should not be called while lifecycle observation owns reconciliation")
					return keymanagementsdk.UpdateVaultResponse{}, nil
				},
			})

			resource := makeSpecVault()
			resource.Status.Id = vaultID
			resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.wantSuccess, response.IsSuccessful)
			assert.Equal(t, tt.wantRequeue, response.ShouldRequeue)
			assert.Equal(t, 0, updateCalls)
			assert.Equal(t, string(tt.state), resource.Status.LifecycleState)
			assert.Equal(t, string(tt.wantReason), resource.Status.OsokStatus.Reason)
			condition := findVaultCondition(resource, tt.wantReason)
			if assert.NotNil(t, condition) {
				assert.Equal(t, tt.wantCondition, condition.Status)
			}
		})
	}
}

func TestVaultRuntimeCreateOrUpdate_SchedulesDeletion(t *testing.T) {
	vaultID := "ocid1.vault.oc1..scheduled"
	scheduleCalls := 0
	var captured keymanagementsdk.ScheduleVaultDeletionRequest
	deletionTime := time.Now().UTC().AddDate(0, 0, 7)

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		scheduleFn: func(_ context.Context, req keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error) {
			scheduleCalls++
			captured = req
			return keymanagementsdk.ScheduleVaultDeletionResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStatePendingDeletion, &deletionTime, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecVault()
	resource.Spec.DeletionScheduleDays = 7
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	assert.Equal(t, 1, scheduleCalls)
	assert.Equal(t, int32(7), resource.Status.RequestedDeletionScheduleDays)
	assert.Equal(t, "PENDING_DELETION", resource.Status.LifecycleState)
	assert.NotEmpty(t, resource.Status.TimeOfDeletion)
	if assert.NotNil(t, captured.ScheduleVaultDeletionDetails.TimeOfDeletion) {
		duration := captured.ScheduleVaultDeletionDetails.TimeOfDeletion.Time.Sub(time.Now().UTC())
		assert.Greater(t, duration, (7*24*time.Hour)-(2*time.Minute))
		assert.Less(t, duration, (7*24*time.Hour)+(2*time.Minute))
	}
}

func TestVaultRuntimeCreateOrUpdate_DoesNotCancelScheduledDeletion(t *testing.T) {
	vaultID := "ocid1.vault.oc1..pending"
	deletionTime := time.Now().UTC().AddDate(0, 0, 7)

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStatePendingDeletion, &deletionTime, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecVault()
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "PENDING_DELETION"
	resource.Status.TimeOfDeletion = deletionTime.Format(time.RFC3339)
	resource.Status.RequestedDeletionScheduleDays = 7
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	assert.Equal(t, int32(7), resource.Status.RequestedDeletionScheduleDays)
	assert.Equal(t, "PENDING_DELETION", resource.Status.LifecycleState)
	recordedDeletion, err := time.Parse(time.RFC3339Nano, resource.Status.TimeOfDeletion)
	if assert.NoError(t, err) {
		assert.WithinDuration(t, deletionTime.UTC(), recordedDeletion, time.Second)
	}
}

func TestVaultRuntimeDelete_SchedulesDeletionAndReturnsSuccess(t *testing.T) {
	vaultID := "ocid1.vault.oc1..delete"
	scheduleCalls := 0
	deletionTime := time.Now().UTC().AddDate(0, 0, 14)

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		scheduleFn: func(_ context.Context, req keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error) {
			scheduleCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.ScheduleVaultDeletionResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStatePendingDeletion, &deletionTime, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecVault()
	resource.Spec.DeletionScheduleDays = 14
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, deleted)
	assert.Equal(t, 1, scheduleCalls)
	assert.Equal(t, int32(14), resource.Status.RequestedDeletionScheduleDays)
	assert.Equal(t, "PENDING_DELETION", resource.Status.LifecycleState)
	assert.NotNil(t, resource.Status.OsokStatus.UpdatedAt)
}

func TestVaultRuntimeDelete_PendingDeletionReturnsSuccess(t *testing.T) {
	vaultID := "ocid1.vault.oc1..already-pending"
	deletionTime := time.Now().UTC().AddDate(0, 0, 10)

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStatePendingDeletion, &deletionTime, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		scheduleFn: func(_ context.Context, req keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error) {
			t.Fatalf("ScheduleVaultDeletion() should not be called once OCI already reports pending deletion for %s", *req.VaultId)
			return keymanagementsdk.ScheduleVaultDeletionResponse{}, nil
		},
	})

	resource := makeSpecVault()
	resource.Spec.DeletionScheduleDays = 10
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "PENDING_DELETION"
	resource.Status.TimeOfDeletion = deletionTime.Format(time.RFC3339)
	resource.Status.RequestedDeletionScheduleDays = 10
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, deleted)
	assert.Equal(t, "PENDING_DELETION", resource.Status.LifecycleState)
	assert.Equal(t, int32(10), resource.Status.RequestedDeletionScheduleDays)
}

func TestVaultRuntimeDelete_CancellingDeletionKeepsFinalizer(t *testing.T) {
	vaultID := "ocid1.vault.oc1..cancel-delete"

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateCancellingDeletion, nil, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		scheduleFn: func(_ context.Context, req keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error) {
			t.Fatalf("ScheduleVaultDeletion() should not be called while OCI is cancelling deletion for %s", *req.VaultId)
			return keymanagementsdk.ScheduleVaultDeletionResponse{}, nil
		},
	})

	resource := makeSpecVault()
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "CANCELLING_DELETION"
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.False(t, deleted)
	assert.Equal(t, "CANCELLING_DELETION", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestVaultRuntimeDelete_DeletedLifecycleClearsScheduleAndReleasesFinalizer(t *testing.T) {
	vaultID := "ocid1.vault.oc1..deleted"
	deletionTime := time.Now().UTC().AddDate(0, 0, -1)

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateDeleted, &deletionTime, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecVault()
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "PENDING_DELETION"
	resource.Status.TimeOfDeletion = time.Now().UTC().AddDate(0, 0, 14).Format(time.RFC3339)
	resource.Status.RequestedDeletionScheduleDays = 14
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, deleted)
	assert.Equal(t, "DELETED", resource.Status.LifecycleState)
	assert.Zero(t, resource.Status.RequestedDeletionScheduleDays)
	assert.Empty(t, resource.Status.TimeOfDeletion)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestVaultRuntimeDelete_TreatsAuthShaped404AsDeleted(t *testing.T) {
	vaultID := "ocid1.vault.oc1..missing"
	getCalls := 0
	listCalls := 0

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			getCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.GetVaultResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "vault missing")
		},
		listFn: func(_ context.Context, req keymanagementsdk.ListVaultsRequest) (keymanagementsdk.ListVaultsResponse, error) {
			listCalls++
			assert.Equal(t, "ocid1.compartment.oc1..example", *req.CompartmentId)
			return keymanagementsdk.ListVaultsResponse{}, nil
		},
	})

	resource := makeSpecVault()
	resource.Status.Id = vaultID
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)
	resource.Status.RequestedDeletionScheduleDays = 14
	resource.Status.TimeOfDeletion = time.Now().UTC().AddDate(0, 0, 14).Format(time.RFC3339)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, deleted)
	assert.Equal(t, 1, getCalls)
	assert.Zero(t, listCalls)
	assert.Zero(t, resource.Status.RequestedDeletionScheduleDays)
	assert.Empty(t, resource.Status.TimeOfDeletion)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestVaultRuntimeDelete_ConflictWithPendingDeletionReturnsSuccess(t *testing.T) {
	vaultID := "ocid1.vault.oc1..conflict"
	deletionTime := time.Now().UTC().AddDate(0, 0, 21)
	getCalls := 0
	scheduleCalls := 0

	manager := newVaultTestManager(&fakeVaultOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error) {
			getCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			if getCalls == 1 {
				return keymanagementsdk.GetVaultResponse{
					Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStateActive, nil, "vault-sample", map[string]string{"env": "dev"}),
				}, nil
			}
			return keymanagementsdk.GetVaultResponse{
				Vault: makeSDKVault(vaultID, keymanagementsdk.VaultLifecycleStatePendingDeletion, &deletionTime, "vault-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		scheduleFn: func(_ context.Context, req keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error) {
			scheduleCalls++
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagementsdk.ScheduleVaultDeletionResponse{}, errors.New("http status code: 409")
		},
	})

	resource := makeSpecVault()
	resource.Spec.DeletionScheduleDays = 21
	resource.Status.Id = vaultID
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.OsokStatus.Ocid = shared.OCID(vaultID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, deleted)
	assert.Equal(t, 1, scheduleCalls)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, "PENDING_DELETION", resource.Status.LifecycleState)
	recordedDeletion, err := time.Parse(time.RFC3339Nano, resource.Status.TimeOfDeletion)
	if assert.NoError(t, err) {
		assert.WithinDuration(t, deletionTime.UTC(), recordedDeletion, time.Second)
	}
}

func findVaultCondition(resource *keymanagementv1beta1.Vault, conditionType shared.OSOKConditionType) *shared.OSOKCondition {
	for i := range resource.Status.OsokStatus.Conditions {
		if resource.Status.OsokStatus.Conditions[i].Type == conditionType {
			return &resource.Status.OsokStatus.Conditions[i]
		}
	}
	return nil
}
