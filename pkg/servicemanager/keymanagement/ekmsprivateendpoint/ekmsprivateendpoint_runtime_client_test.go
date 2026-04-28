/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ekmsprivateendpoint

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	keymanagementsdk "github.com/oracle/oci-go-sdk/v65/keymanagement"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeEkmsPrivateEndpointOCIClient struct {
	createFn func(context.Context, keymanagementsdk.CreateEkmsPrivateEndpointRequest) (keymanagementsdk.CreateEkmsPrivateEndpointResponse, error)
	getFn    func(context.Context, keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error)
	listFn   func(context.Context, keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error)
	updateFn func(context.Context, keymanagementsdk.UpdateEkmsPrivateEndpointRequest) (keymanagementsdk.UpdateEkmsPrivateEndpointResponse, error)
	deleteFn func(context.Context, keymanagementsdk.DeleteEkmsPrivateEndpointRequest) (keymanagementsdk.DeleteEkmsPrivateEndpointResponse, error)
}

func (f *fakeEkmsPrivateEndpointOCIClient) CreateEkmsPrivateEndpoint(ctx context.Context, req keymanagementsdk.CreateEkmsPrivateEndpointRequest) (keymanagementsdk.CreateEkmsPrivateEndpointResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return keymanagementsdk.CreateEkmsPrivateEndpointResponse{}, nil
}

func (f *fakeEkmsPrivateEndpointOCIClient) GetEkmsPrivateEndpoint(ctx context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return keymanagementsdk.GetEkmsPrivateEndpointResponse{}, nil
}

func (f *fakeEkmsPrivateEndpointOCIClient) ListEkmsPrivateEndpoints(ctx context.Context, req keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return keymanagementsdk.ListEkmsPrivateEndpointsResponse{}, nil
}

func (f *fakeEkmsPrivateEndpointOCIClient) UpdateEkmsPrivateEndpoint(ctx context.Context, req keymanagementsdk.UpdateEkmsPrivateEndpointRequest) (keymanagementsdk.UpdateEkmsPrivateEndpointResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return keymanagementsdk.UpdateEkmsPrivateEndpointResponse{}, nil
}

func (f *fakeEkmsPrivateEndpointOCIClient) DeleteEkmsPrivateEndpoint(ctx context.Context, req keymanagementsdk.DeleteEkmsPrivateEndpointRequest) (keymanagementsdk.DeleteEkmsPrivateEndpointResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return keymanagementsdk.DeleteEkmsPrivateEndpointResponse{}, nil
}

func newEkmsPrivateEndpointTestManager(client *fakeEkmsPrivateEndpointOCIClient) *EkmsPrivateEndpointServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := &EkmsPrivateEndpointServiceManager{Log: log}
	if client != nil {
		delegate := defaultEkmsPrivateEndpointServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*keymanagementv1beta1.EkmsPrivateEndpoint](newEkmsPrivateEndpointTestRuntimeConfig(log, client)),
		}
		manager.WithClient(&ekmsPrivateEndpointRuntimeClient{
			delegate: delegate,
			sdk:      client,
			log:      log,
		})
	}
	return manager
}

func newEkmsPrivateEndpointTestRuntimeConfig(log loggerutil.OSOKLogger, client *fakeEkmsPrivateEndpointOCIClient) generatedruntime.Config[*keymanagementv1beta1.EkmsPrivateEndpoint] {
	return generatedruntime.Config[*keymanagementv1beta1.EkmsPrivateEndpoint]{
		Kind:      "EkmsPrivateEndpoint",
		SDKName:   "EkmsPrivateEndpoint",
		Log:       log,
		Semantics: newEkmsPrivateEndpointRuntimeSemantics(),
		BuildCreateBody: func(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, namespace string) (any, error) {
			return buildEkmsPrivateEndpointCreateDetails(ctx, resource, namespace)
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.CreateEkmsPrivateEndpointRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateEkmsPrivateEndpoint(ctx, *request.(*keymanagementsdk.CreateEkmsPrivateEndpointRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateEkmsPrivateEndpointDetails", RequestName: "CreateEkmsPrivateEndpointDetails", Contribution: "body", PreferResourceID: false},
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.GetEkmsPrivateEndpointRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetEkmsPrivateEndpoint(ctx, *request.(*keymanagementsdk.GetEkmsPrivateEndpointRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "EkmsPrivateEndpointId", RequestName: "ekmsPrivateEndpointId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.ListEkmsPrivateEndpointsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return listEkmsPrivateEndpointsAllPages(ctx, *request.(*keymanagementsdk.ListEkmsPrivateEndpointsRequest), client.ListEkmsPrivateEndpoints)
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
				{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.UpdateEkmsPrivateEndpointRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateEkmsPrivateEndpoint(ctx, *request.(*keymanagementsdk.UpdateEkmsPrivateEndpointRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "EkmsPrivateEndpointId", RequestName: "ekmsPrivateEndpointId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateEkmsPrivateEndpointDetails", RequestName: "UpdateEkmsPrivateEndpointDetails", Contribution: "body", PreferResourceID: false},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.DeleteEkmsPrivateEndpointRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteEkmsPrivateEndpoint(ctx, *request.(*keymanagementsdk.DeleteEkmsPrivateEndpointRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "EkmsPrivateEndpointId", RequestName: "ekmsPrivateEndpointId", Contribution: "path", PreferResourceID: true},
			},
		},
	}
}

func makeSpecEkmsPrivateEndpoint() *keymanagementv1beta1.EkmsPrivateEndpoint {
	return &keymanagementv1beta1.EkmsPrivateEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ekms-private-endpoint-sample",
			Namespace: "default",
			UID:       types.UID("ekms-private-endpoint-uid"),
		},
		Spec: keymanagementv1beta1.EkmsPrivateEndpointSpec{
			SubnetId:             "ocid1.subnet.oc1..example",
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "ekms-private-endpoint-sample",
			ExternalKeyManagerIp: "10.0.0.10",
			CaBundle:             "-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			Port: 5696,
		},
	}
}

func makeSDKEkmsPrivateEndpoint(id string, lifecycleState keymanagementsdk.EkmsPrivateEndpointLifecycleStateEnum, displayName string, tags map[string]string) keymanagementsdk.EkmsPrivateEndpoint {
	created := common.SDKTime{Time: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 8, 12, 30, 0, 0, time.UTC)}
	return keymanagementsdk.EkmsPrivateEndpoint{
		Id:                   common.String(id),
		CompartmentId:        common.String("ocid1.compartment.oc1..example"),
		SubnetId:             common.String("ocid1.subnet.oc1..example"),
		DisplayName:          common.String(displayName),
		TimeCreated:          &created,
		LifecycleState:       lifecycleState,
		ExternalKeyManagerIp: common.String("10.0.0.10"),
		TimeUpdated:          &updated,
		FreeformTags:         tags,
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {
				"CostCenter": "42",
			},
		},
		LifecycleDetails:  common.String("ready"),
		Port:              common.Int(5696),
		CaBundle:          common.String("-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----"),
		PrivateEndpointIp: common.String("10.0.0.5"),
	}
}

func makeSDKEkmsPrivateEndpointSummary(id string, lifecycleState keymanagementsdk.EkmsPrivateEndpointSummaryLifecycleStateEnum, displayName string) keymanagementsdk.EkmsPrivateEndpointSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)}
	return keymanagementsdk.EkmsPrivateEndpointSummary{
		Id:             common.String(id),
		SubnetId:       common.String("ocid1.subnet.oc1..example"),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		TimeCreated:    &created,
		DisplayName:    common.String(displayName),
		LifecycleState: lifecycleState,
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

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_CreatesEndpoint(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..created"
	createCalls := 0
	getCalls := 0
	listCalls := 0

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		listFn: func(_ context.Context, req keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error) {
			listCalls++
			assert.Equal(t, "ocid1.compartment.oc1..example", stringValue(req.CompartmentId))
			return keymanagementsdk.ListEkmsPrivateEndpointsResponse{}, nil
		},
		createFn: func(_ context.Context, req keymanagementsdk.CreateEkmsPrivateEndpointRequest) (keymanagementsdk.CreateEkmsPrivateEndpointResponse, error) {
			createCalls++
			assert.Equal(t, "ocid1.subnet.oc1..example", stringValue(req.SubnetId))
			assert.Equal(t, "ocid1.compartment.oc1..example", stringValue(req.CompartmentId))
			assert.Equal(t, "ekms-private-endpoint-sample", stringValue(req.DisplayName))
			assert.Equal(t, "10.0.0.10", stringValue(req.ExternalKeyManagerIp))
			assert.Equal(t, 5696, intValue(req.Port))
			assert.Equal(t, "ekms-private-endpoint-uid", stringValue(req.OpcRetryToken))
			return keymanagementsdk.CreateEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateCreating, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
				OpcRequestId:        common.String("create-request"),
				OpcWorkRequestId:    common.String("create-work-request"),
			}, nil
		},
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			getCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, 1, listCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, endpointID, resource.Status.Id)
	assert.Equal(t, endpointID, string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "create-request", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
	assert.Equal(t, "10.0.0.5", resource.Status.PrivateEndpointIp)
}

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_BindsExistingEndpointFromPaginatedList(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..existing"
	var pages []string
	getCalls := 0

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		createFn: func(_ context.Context, _ keymanagementsdk.CreateEkmsPrivateEndpointRequest) (keymanagementsdk.CreateEkmsPrivateEndpointResponse, error) {
			t.Fatal("CreateEkmsPrivateEndpoint() should not be called when list lookup finds a reusable endpoint")
			return keymanagementsdk.CreateEkmsPrivateEndpointResponse{}, nil
		},
		listFn: func(_ context.Context, req keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error) {
			pages = append(pages, stringValue(req.Page))
			if req.Page == nil {
				return keymanagementsdk.ListEkmsPrivateEndpointsResponse{
					Items: []keymanagementsdk.EkmsPrivateEndpointSummary{
						makeSDKEkmsPrivateEndpointSummary("ocid1.ekmsprivateendpoint.oc1..other", keymanagementsdk.EkmsPrivateEndpointSummaryLifecycleStateActive, "other"),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			assert.Equal(t, "page-2", stringValue(req.Page))
			return keymanagementsdk.ListEkmsPrivateEndpointsResponse{
				Items: []keymanagementsdk.EkmsPrivateEndpointSummary{
					makeSDKEkmsPrivateEndpointSummary(endpointID, keymanagementsdk.EkmsPrivateEndpointSummaryLifecycleStateActive, "ekms-private-endpoint-sample"),
				},
			}, nil
		},
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			getCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
			}, nil
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, []string{"", "page-2"}, pages)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, endpointID, resource.Status.Id)
	assert.Equal(t, endpointID, string(resource.Status.OsokStatus.Ocid))
}

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_NoOpReconcileDoesNotUpdate(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..noop"
	getCalls := 0
	updateCalls := 0

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			getCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		updateFn: func(_ context.Context, _ keymanagementsdk.UpdateEkmsPrivateEndpointRequest) (keymanagementsdk.UpdateEkmsPrivateEndpointResponse, error) {
			updateCalls++
			t.Fatal("UpdateEkmsPrivateEndpoint() should not be called without mutable drift")
			return keymanagementsdk.UpdateEkmsPrivateEndpointResponse{}, nil
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	resource.Status.Id = endpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
}

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_UpdatesMutableFieldsInPlace(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..update"
	getCalls := 0
	updateCalls := 0
	var captured keymanagementsdk.UpdateEkmsPrivateEndpointRequest

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			getCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			if getCalls == 1 {
				return keymanagementsdk.GetEkmsPrivateEndpointResponse{
					EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "endpoint-old", map[string]string{"env": "dev"}),
				}, nil
			}
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "endpoint-new", map[string]string{"env": "prod"}),
			}, nil
		},
		updateFn: func(_ context.Context, req keymanagementsdk.UpdateEkmsPrivateEndpointRequest) (keymanagementsdk.UpdateEkmsPrivateEndpointResponse, error) {
			updateCalls++
			captured = req
			return keymanagementsdk.UpdateEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "endpoint-new", map[string]string{"env": "prod"}),
				OpcRequestId:        common.String("update-request"),
			}, nil
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	resource.Spec.DisplayName = "endpoint-new"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Status.Id = endpointID
	resource.Status.DisplayName = "endpoint-old"
	resource.Status.FreeformTags = map[string]string{"env": "dev"}
	resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 1, updateCalls)
	assert.Equal(t, endpointID, stringValue(captured.EkmsPrivateEndpointId))
	assert.Equal(t, "endpoint-new", stringValue(captured.DisplayName))
	assert.Equal(t, map[string]string{"env": "prod"}, captured.FreeformTags)
	assert.Nil(t, captured.DefinedTags)
	assert.Equal(t, "endpoint-new", resource.Status.DisplayName)
	assert.Equal(t, map[string]string{"env": "prod"}, resource.Status.FreeformTags)
	assert.Equal(t, "update-request", resource.Status.OsokStatus.OpcRequestID)
}

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_RejectsForceNewDrift(t *testing.T) {
	tests := []struct {
		name       string
		mutateSpec func(*keymanagementv1beta1.EkmsPrivateEndpoint)
		wantPath   string
	}{
		{
			name: "subnet",
			mutateSpec: func(resource *keymanagementv1beta1.EkmsPrivateEndpoint) {
				resource.Spec.SubnetId = "ocid1.subnet.oc1..different"
			},
			wantPath: "subnetId",
		},
		{
			name: "external key manager ip",
			mutateSpec: func(resource *keymanagementv1beta1.EkmsPrivateEndpoint) {
				resource.Spec.ExternalKeyManagerIp = "10.0.0.20"
			},
			wantPath: "externalKeyManagerIp",
		},
		{
			name: "ca bundle",
			mutateSpec: func(resource *keymanagementv1beta1.EkmsPrivateEndpoint) {
				resource.Spec.CaBundle = "-----BEGIN CERTIFICATE-----\ndifferent\n-----END CERTIFICATE-----"
			},
			wantPath: "caBundle",
		},
		{
			name: "port",
			mutateSpec: func(resource *keymanagementv1beta1.EkmsPrivateEndpoint) {
				resource.Spec.Port = 9443
			},
			wantPath: "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointID := "ocid1.ekmsprivateendpoint.oc1.." + tt.name
			updateCalls := 0
			manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
				getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
					assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
					return keymanagementsdk.GetEkmsPrivateEndpointResponse{
						EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
					}, nil
				},
				updateFn: func(_ context.Context, _ keymanagementsdk.UpdateEkmsPrivateEndpointRequest) (keymanagementsdk.UpdateEkmsPrivateEndpointResponse, error) {
					updateCalls++
					t.Fatal("UpdateEkmsPrivateEndpoint() should not be called when force-new drift is detected")
					return keymanagementsdk.UpdateEkmsPrivateEndpointResponse{}, nil
				},
			})

			resource := makeSpecEkmsPrivateEndpoint()
			tt.mutateSpec(resource)
			resource.Status.Id = endpointID
			resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "require replacement when "+tt.wantPath+" changes")
			assert.False(t, response.IsSuccessful)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_MapsLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		state         keymanagementsdk.EkmsPrivateEndpointLifecycleStateEnum
		wantReason    shared.OSOKConditionType
		wantSuccess   bool
		wantRequeue   bool
		wantCondition v1.ConditionStatus
	}{
		{
			name:          "creating",
			state:         keymanagementsdk.EkmsPrivateEndpointLifecycleStateCreating,
			wantReason:    shared.Provisioning,
			wantSuccess:   true,
			wantRequeue:   true,
			wantCondition: v1.ConditionTrue,
		},
		{
			name:          "failed",
			state:         keymanagementsdk.EkmsPrivateEndpointLifecycleStateFailed,
			wantReason:    shared.Failed,
			wantSuccess:   false,
			wantRequeue:   false,
			wantCondition: v1.ConditionFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointID := "ocid1.ekmsprivateendpoint.oc1.." + tt.name
			updateCalls := 0
			manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
				getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
					assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
					return keymanagementsdk.GetEkmsPrivateEndpointResponse{
						EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, tt.state, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
					}, nil
				},
				updateFn: func(_ context.Context, _ keymanagementsdk.UpdateEkmsPrivateEndpointRequest) (keymanagementsdk.UpdateEkmsPrivateEndpointResponse, error) {
					updateCalls++
					t.Fatal("UpdateEkmsPrivateEndpoint() should not be called while lifecycle observation owns reconciliation")
					return keymanagementsdk.UpdateEkmsPrivateEndpointResponse{}, nil
				},
			})

			resource := makeSpecEkmsPrivateEndpoint()
			resource.Status.Id = endpointID
			resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.wantSuccess, response.IsSuccessful)
			assert.Equal(t, tt.wantRequeue, response.ShouldRequeue)
			assert.Equal(t, 0, updateCalls)
			assert.Equal(t, string(tt.state), resource.Status.LifecycleState)
			assert.Equal(t, string(tt.wantReason), resource.Status.OsokStatus.Reason)
			condition := findEkmsPrivateEndpointCondition(resource, tt.wantReason)
			if assert.NotNil(t, condition) {
				assert.Equal(t, tt.wantCondition, condition.Status)
			}
		})
	}
}

func TestEkmsPrivateEndpointRuntimeCreateOrUpdate_RecordsCreateErrorRequestID(t *testing.T) {
	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		listFn: func(_ context.Context, _ keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error) {
			return keymanagementsdk.ListEkmsPrivateEndpointsResponse{}, nil
		},
		createFn: func(_ context.Context, _ keymanagementsdk.CreateEkmsPrivateEndpointRequest) (keymanagementsdk.CreateEkmsPrivateEndpointResponse, error) {
			return keymanagementsdk.CreateEkmsPrivateEndpointResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, response.IsSuccessful)
	assert.Equal(t, "opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
}

func TestEkmsPrivateEndpointRuntimeDelete_RetainsFinalizerUntilDeletingIsConfirmed(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..delete"
	getCalls := 0
	deleteCalls := 0

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			getCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			if getCalls == 1 {
				return keymanagementsdk.GetEkmsPrivateEndpointResponse{
					EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
				}, nil
			}
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateDeleting, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		deleteFn: func(_ context.Context, req keymanagementsdk.DeleteEkmsPrivateEndpointRequest) (keymanagementsdk.DeleteEkmsPrivateEndpointResponse, error) {
			deleteCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.DeleteEkmsPrivateEndpointResponse{
				OpcRequestId:     common.String("delete-request"),
				OpcWorkRequestId: common.String("delete-work-request"),
			}, nil
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	resource.Status.Id = endpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.False(t, deleted)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 1, deleteCalls)
	assert.Equal(t, "DELETING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "delete-request", resource.Status.OsokStatus.OpcRequestID)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	if current := resource.Status.OsokStatus.Async.Current; assert.NotNil(t, current) {
		assert.Equal(t, shared.OSOKAsyncSourceLifecycle, current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseDelete, current.Phase)
		assert.Equal(t, "delete-work-request", current.WorkRequestID)
		assert.Equal(t, shared.OSOKAsyncClassPending, current.NormalizedClass)
	}
}

func TestEkmsPrivateEndpointRuntimeDelete_ReleasesFinalizerWhenDeletedLifecycleObserved(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..deleted"
	deleteCalls := 0

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateDeleted, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		deleteFn: func(_ context.Context, _ keymanagementsdk.DeleteEkmsPrivateEndpointRequest) (keymanagementsdk.DeleteEkmsPrivateEndpointResponse, error) {
			deleteCalls++
			t.Fatal("DeleteEkmsPrivateEndpoint() should not be called once OCI already reports DELETED")
			return keymanagementsdk.DeleteEkmsPrivateEndpointResponse{}, nil
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	resource.Status.Id = endpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, deleted)
	assert.Equal(t, 0, deleteCalls)
	assert.Equal(t, "DELETED", resource.Status.LifecycleState)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestEkmsPrivateEndpointRuntimeDelete_TreatsAuthShaped404Conservatively(t *testing.T) {
	endpointID := "ocid1.ekmsprivateendpoint.oc1..auth-shaped"
	deleteCalls := 0

	manager := newEkmsPrivateEndpointTestManager(&fakeEkmsPrivateEndpointOCIClient{
		getFn: func(_ context.Context, req keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error) {
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.GetEkmsPrivateEndpointResponse{
				EkmsPrivateEndpoint: makeSDKEkmsPrivateEndpoint(endpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive, "ekms-private-endpoint-sample", map[string]string{"env": "dev"}),
			}, nil
		},
		deleteFn: func(_ context.Context, req keymanagementsdk.DeleteEkmsPrivateEndpointRequest) (keymanagementsdk.DeleteEkmsPrivateEndpointResponse, error) {
			deleteCalls++
			assert.Equal(t, endpointID, stringValue(req.EkmsPrivateEndpointId))
			return keymanagementsdk.DeleteEkmsPrivateEndpointResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete")
		},
	})

	resource := makeSpecEkmsPrivateEndpoint()
	resource.Status.Id = endpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(endpointID)

	deleted, err := manager.Delete(context.Background(), resource)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authorization-shaped not-found")
	assert.False(t, deleted)
	assert.Equal(t, 1, deleteCalls)
	assert.Equal(t, "opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
}

func findEkmsPrivateEndpointCondition(resource *keymanagementv1beta1.EkmsPrivateEndpoint, conditionType shared.OSOKConditionType) *shared.OSOKCondition {
	for i := range resource.Status.OsokStatus.Conditions {
		if resource.Status.OsokStatus.Conditions[i].Type == conditionType {
			return &resource.Status.OsokStatus.Conditions[i]
		}
	}
	return nil
}
