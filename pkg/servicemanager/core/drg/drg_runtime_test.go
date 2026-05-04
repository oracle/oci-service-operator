/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drg

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDrgOCIClient struct {
	createFn func(context.Context, coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error)
	getFn    func(context.Context, coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error)
	listFn   func(context.Context, coresdk.ListDrgsRequest) (coresdk.ListDrgsResponse, error)
	updateFn func(context.Context, coresdk.UpdateDrgRequest) (coresdk.UpdateDrgResponse, error)
	deleteFn func(context.Context, coresdk.DeleteDrgRequest) (coresdk.DeleteDrgResponse, error)
}

func (f *fakeDrgOCIClient) CreateDrg(ctx context.Context, req coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateDrgResponse{}, nil
}

func (f *fakeDrgOCIClient) GetDrg(ctx context.Context, req coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetDrgResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
}

func (f *fakeDrgOCIClient) ListDrgs(ctx context.Context, req coresdk.ListDrgsRequest) (coresdk.ListDrgsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return coresdk.ListDrgsResponse{}, nil
}

func (f *fakeDrgOCIClient) UpdateDrg(ctx context.Context, req coresdk.UpdateDrgRequest) (coresdk.UpdateDrgResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateDrgResponse{}, nil
}

func (f *fakeDrgOCIClient) DeleteDrg(ctx context.Context, req coresdk.DeleteDrgRequest) (coresdk.DeleteDrgResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteDrgResponse{}, nil
}

func newTestDrgRuntimeHooks(manager *DrgServiceManager, client *fakeDrgOCIClient) DrgRuntimeHooks {
	if client == nil {
		client = &fakeDrgOCIClient{}
	}

	hooks := DrgRuntimeHooks{
		Create: runtimeOperationHooks[coresdk.CreateDrgRequest, coresdk.CreateDrgResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateDrgDetails", RequestName: "CreateDrgDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error) {
				return client.CreateDrg(ctx, request)
			},
		},
		Get: runtimeOperationHooks[coresdk.GetDrgRequest, coresdk.GetDrgResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DrgId", RequestName: "drgId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
				return client.GetDrg(ctx, request)
			},
		},
		List: runtimeOperationHooks[coresdk.ListDrgsRequest, coresdk.ListDrgsResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
			},
			Call: func(ctx context.Context, request coresdk.ListDrgsRequest) (coresdk.ListDrgsResponse, error) {
				return client.ListDrgs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[coresdk.UpdateDrgRequest, coresdk.UpdateDrgResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "DrgId", RequestName: "drgId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateDrgDetails", RequestName: "UpdateDrgDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request coresdk.UpdateDrgRequest) (coresdk.UpdateDrgResponse, error) {
				return client.UpdateDrg(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[coresdk.DeleteDrgRequest, coresdk.DeleteDrgResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DrgId", RequestName: "drgId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request coresdk.DeleteDrgRequest) (coresdk.DeleteDrgResponse, error) {
				return client.DeleteDrg(ctx, request)
			},
		},
	}
	applyDrgRuntimeHooks(manager, &hooks)
	appendDrgCreateFallbackRuntimeWrapper(manager, &hooks)
	return hooks
}

func newTestGeneratedDrgDelegate(manager *DrgServiceManager, client *fakeDrgOCIClient) DrgServiceClient {
	hooks := newTestDrgRuntimeHooks(manager, client)
	delegate := defaultDrgServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.Drg](buildDrgGeneratedRuntimeConfig(manager, hooks)),
	}
	return wrapDrgGeneratedClient(hooks, delegate)
}

func newTestDrgManager(client *fakeDrgOCIClient) *DrgServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewDrgServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(newTestGeneratedDrgDelegate(manager, client))
	}
	return manager
}

func makeSpecDrg() *corev1beta1.Drg {
	return &corev1beta1.Drg{
		Spec: corev1beta1.DrgSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			DisplayName:   "test-drg",
			FreeformTags:  map[string]string{"env": "dev"},
			DefaultDrgRouteTables: corev1beta1.DrgDefaultDrgRouteTables{
				Vcn:                     "ocid1.drgroutetable.oc1..vcn",
				IpsecTunnel:             "ocid1.drgroutetable.oc1..ipsec",
				VirtualCircuit:          "ocid1.drgroutetable.oc1..vc",
				RemotePeeringConnection: "ocid1.drgroutetable.oc1..rpc",
			},
		},
	}
}

func makeSDKDrg(id, displayName string, state coresdk.DrgLifecycleStateEnum) coresdk.Drg {
	return coresdk.Drg{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		LifecycleState: state,
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		DisplayName:    common.String(displayName),
		FreeformTags:   map[string]string{"env": "dev"},
		TimeCreated:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		DefaultDrgRouteTables: &coresdk.DefaultDrgRouteTables{
			Vcn:                     common.String("ocid1.drgroutetable.oc1..vcn"),
			IpsecTunnel:             common.String("ocid1.drgroutetable.oc1..ipsec"),
			VirtualCircuit:          common.String("ocid1.drgroutetable.oc1..vc"),
			RemotePeeringConnection: common.String("ocid1.drgroutetable.oc1..rpc"),
		},
		DefaultExportDrgRouteDistributionId: common.String("ocid1.drgroutedistribution.oc1..export"),
	}
}

func TestDrgCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateDrgRequest
	manager := newTestDrgManager(&fakeDrgOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error) {
			captured = req
			return coresdk.CreateDrgResponse{
				Drg: makeSDKDrg("ocid1.drg.oc1..create", "test-drg", coresdk.DrgLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecDrg()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("test-drg"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, "ocid1.drg.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-drg", resource.Status.DisplayName)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
	assert.Equal(t, "ocid1.drgroutetable.oc1..vcn", resource.Status.DefaultDrgRouteTables.Vcn)
	assert.Equal(t, "ocid1.drgroutedistribution.oc1..export", resource.Status.DefaultExportDrgRouteDistributionId)
}

func TestDrgCreateOrUpdate_BindsExistingByDisplayNameAndCompartment(t *testing.T) {
	createCalls := 0
	getCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		listFn: func(_ context.Context, req coresdk.ListDrgsRequest) (coresdk.ListDrgsResponse, error) {
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.ListDrgsResponse{
				Items: []coresdk.Drg{makeSDKDrg("ocid1.drg.oc1..existing", "test-drg", coresdk.DrgLifecycleStateAvailable)},
			}, nil
		},
		getFn: func(_ context.Context, req coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.drg.oc1..existing", *req.DrgId)
			return coresdk.GetDrgResponse{
				Drg: makeSDKDrg("ocid1.drg.oc1..existing", "test-drg", coresdk.DrgLifecycleStateAvailable),
			}, nil
		},
		createFn: func(_ context.Context, _ coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error) {
			createCalls++
			return coresdk.CreateDrgResponse{}, nil
		},
	})

	resource := makeSpecDrg()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, createCalls)
	assert.Equal(t, "ocid1.drg.oc1..existing", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestDrgCreateOrUpdate_DoesNotBindWithoutDisplayName(t *testing.T) {
	listCalls := 0
	createCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		listFn: func(_ context.Context, _ coresdk.ListDrgsRequest) (coresdk.ListDrgsResponse, error) {
			listCalls++
			return coresdk.ListDrgsResponse{
				Items: []coresdk.Drg{makeSDKDrg("ocid1.drg.oc1..wrong", "some-drg", coresdk.DrgLifecycleStateAvailable)},
			}, nil
		},
		createFn: func(_ context.Context, _ coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error) {
			createCalls++
			return coresdk.CreateDrgResponse{
				Drg: makeSDKDrg("ocid1.drg.oc1..create", "", coresdk.DrgLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecDrg()
	resource.Spec.DisplayName = ""
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 0, listCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.drg.oc1..create", string(resource.Status.OsokStatus.Ocid))
}

func TestDrgCreateOrUpdate_RejectsAmbiguousDisplayNameReuse(t *testing.T) {
	createCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		listFn: func(_ context.Context, req coresdk.ListDrgsRequest) (coresdk.ListDrgsResponse, error) {
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.ListDrgsResponse{
				Items: []coresdk.Drg{
					makeSDKDrg("ocid1.drg.oc1..first", "test-drg", coresdk.DrgLifecycleStateAvailable),
					makeSDKDrg("ocid1.drg.oc1..second", "test-drg", coresdk.DrgLifecycleStateProvisioning),
				},
			}, nil
		},
		createFn: func(_ context.Context, _ coresdk.CreateDrgRequest) (coresdk.CreateDrgResponse, error) {
			createCalls++
			return coresdk.CreateDrgResponse{}, nil
		},
	})

	resource := makeSpecDrg()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "multiple matching resources")
	assert.Equal(t, 0, createCalls)
	assert.Empty(t, resource.Status.Id)
	assert.Empty(t, string(resource.Status.OsokStatus.Ocid))
}

func TestProjectDrgStatus_ClearsStaleOptionalFields(t *testing.T) {
	resource := makeSpecDrg()
	resource.Status = corev1beta1.DrgStatus{
		OsokStatus: shared.OSOKStatus{Reason: "kept"},
		Id:         "ocid1.drg.oc1..stale",
		DefinedTags: map[string]shared.MapValue{
			"Operations": {"CostCenter": "42"},
		},
		DisplayName:  "stale-name",
		FreeformTags: map[string]string{"env": "stale"},
		TimeCreated:  "2026-04-01T00:00:00Z",
		DefaultDrgRouteTables: corev1beta1.DrgDefaultDrgRouteTables{
			Vcn:                     "ocid1.drgroutetable.oc1..stale-vcn",
			IpsecTunnel:             "ocid1.drgroutetable.oc1..stale-ipsec",
			VirtualCircuit:          "ocid1.drgroutetable.oc1..stale-vc",
			RemotePeeringConnection: "ocid1.drgroutetable.oc1..stale-rpc",
		},
		DefaultExportDrgRouteDistributionId: "ocid1.drgroutedistribution.oc1..stale",
	}

	err := projectDrgStatus(resource, coresdk.GetDrgResponse{
		Drg: coresdk.Drg{
			Id:             common.String("ocid1.drg.oc1..current"),
			CompartmentId:  common.String("ocid1.compartment.oc1..example"),
			LifecycleState: coresdk.DrgLifecycleStateAvailable,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "kept", resource.Status.OsokStatus.Reason)
	assert.Equal(t, "ocid1.drg.oc1..current", resource.Status.Id)
	assert.Equal(t, "ocid1.compartment.oc1..example", resource.Status.CompartmentId)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Empty(t, resource.Status.DisplayName)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Empty(t, resource.Status.TimeCreated)
	assert.Empty(t, resource.Status.DefaultDrgRouteTables)
	assert.Empty(t, resource.Status.DefaultExportDrgRouteDistributionId)
}

func TestDrgCreateOrUpdate_MutableDriftTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateDrgRequest
	manager := newTestDrgManager(&fakeDrgOCIClient{
		getFn: func(_ context.Context, req coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
			assert.Equal(t, "ocid1.drg.oc1..existing", *req.DrgId)
			current := makeSDKDrg("ocid1.drg.oc1..existing", "old-drg", coresdk.DrgLifecycleStateAvailable)
			current.FreeformTags = map[string]string{"env": "old"}
			current.DefaultDrgRouteTables.Vcn = common.String("ocid1.drgroutetable.oc1..old")
			return coresdk.GetDrgResponse{Drg: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateDrgRequest) (coresdk.UpdateDrgResponse, error) {
			captured = req
			return coresdk.UpdateDrgResponse{
				Drg: makeSDKDrg("ocid1.drg.oc1..existing", "test-drg", coresdk.DrgLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecDrg()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.drg.oc1..existing")
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, common.String("ocid1.drg.oc1..existing"), captured.DrgId)
	assert.Equal(t, common.String("test-drg"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, common.String("ocid1.drgroutetable.oc1..vcn"), captured.DefaultDrgRouteTables.Vcn)
	assert.Equal(t, "test-drg", resource.Status.DisplayName)
}

func TestDrgCreateOrUpdate_RejectsCreateOnlyCompartmentDrift(t *testing.T) {
	updateCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
			current := makeSDKDrg("ocid1.drg.oc1..existing", "test-drg", coresdk.DrgLifecycleStateAvailable)
			current.CompartmentId = common.String("ocid1.compartment.oc1..other")
			return coresdk.GetDrgResponse{Drg: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateDrgRequest) (coresdk.UpdateDrgResponse, error) {
			updateCalls++
			return coresdk.UpdateDrgResponse{}, nil
		},
	})

	resource := makeSpecDrg()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.drg.oc1..existing")
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "compartmentId")
	assert.Equal(t, 0, updateCalls)
}

func TestDrgCreateOrUpdate_RequeuesRetryableLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name   string
		state  coresdk.DrgLifecycleStateEnum
		reason shared.OSOKConditionType
	}{
		{name: "provisioning", state: coresdk.DrgLifecycleStateProvisioning, reason: shared.Provisioning},
		{name: "updating", state: coresdk.DrgLifecycleStateEnum("UPDATING"), reason: shared.Updating},
		{name: "terminating", state: coresdk.DrgLifecycleStateTerminating, reason: shared.Terminating},
	} {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newTestDrgManager(&fakeDrgOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
					return coresdk.GetDrgResponse{
						Drg: makeSDKDrg("ocid1.drg.oc1..existing", "test-drg", tt.state),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateDrgRequest) (coresdk.UpdateDrgResponse, error) {
					updateCalls++
					return coresdk.UpdateDrgResponse{}, nil
				},
			})

			resource := makeSpecDrg()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.drg.oc1..existing")
			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, drgRequeueDuration, resp.RequeueDuration)
			assert.Equal(t, 0, updateCalls)
			assert.Equal(t, string(tt.reason), resource.Status.OsokStatus.Reason)
		})
	}
}

func TestDrgDelete_ConfirmsDeletionOnNotFoundAfterDelete(t *testing.T) {
	getCalls := 0
	deleteCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
			getCalls++
			if getCalls == 1 {
				return coresdk.GetDrgResponse{
					Drg: makeSDKDrg("ocid1.drg.oc1..delete", "test-drg", coresdk.DrgLifecycleStateAvailable),
				}, nil
			}
			return coresdk.GetDrgResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
		},
		deleteFn: func(_ context.Context, req coresdk.DeleteDrgRequest) (coresdk.DeleteDrgResponse, error) {
			deleteCalls++
			assert.Equal(t, "ocid1.drg.oc1..delete", *req.DrgId)
			return coresdk.DeleteDrgResponse{}, nil
		},
	})

	resource := makeSpecDrg()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.drg.oc1..delete")
	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 1, deleteCalls)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestDrgDelete_ReleasesFinalizerWhenAlreadyTerminated(t *testing.T) {
	deleteCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
			return coresdk.GetDrgResponse{
				Drg: makeSDKDrg("ocid1.drg.oc1..delete", "test-drg", coresdk.DrgLifecycleStateTerminated),
			}, nil
		},
		deleteFn: func(_ context.Context, _ coresdk.DeleteDrgRequest) (coresdk.DeleteDrgResponse, error) {
			deleteCalls++
			return coresdk.DeleteDrgResponse{}, nil
		},
	})

	resource := makeSpecDrg()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.drg.oc1..delete")
	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, 0, deleteCalls)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, "TERMINATED", resource.Status.LifecycleState)
}

func TestDrgDelete_KeepsFinalizerWhileTerminating(t *testing.T) {
	deleteCalls := 0
	manager := newTestDrgManager(&fakeDrgOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetDrgRequest) (coresdk.GetDrgResponse, error) {
			return coresdk.GetDrgResponse{
				Drg: makeSDKDrg("ocid1.drg.oc1..delete", "test-drg", coresdk.DrgLifecycleStateTerminating),
			}, nil
		},
		deleteFn: func(_ context.Context, _ coresdk.DeleteDrgRequest) (coresdk.DeleteDrgResponse, error) {
			deleteCalls++
			return coresdk.DeleteDrgResponse{}, nil
		},
	})

	resource := makeSpecDrg()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.drg.oc1..delete")
	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, 0, deleteCalls)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "TERMINATING", resource.Status.LifecycleState)
}
