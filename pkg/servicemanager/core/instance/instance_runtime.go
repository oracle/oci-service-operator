/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package instance

import (
	"context"
	"fmt"

	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type instanceOCIClient interface {
	LaunchInstance(context.Context, coresdk.LaunchInstanceRequest) (coresdk.LaunchInstanceResponse, error)
	GetInstance(context.Context, coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error)
	ListInstances(context.Context, coresdk.ListInstancesRequest) (coresdk.ListInstancesResponse, error)
	UpdateInstance(context.Context, coresdk.UpdateInstanceRequest) (coresdk.UpdateInstanceResponse, error)
	TerminateInstance(context.Context, coresdk.TerminateInstanceRequest) (coresdk.TerminateInstanceResponse, error)
}

func init() {
	newInstanceServiceClient = func(manager *InstanceServiceManager) InstanceServiceClient {
		sdkClient, err := coresdk.NewComputeClientWithConfigurationProvider(manager.Provider)
		config := newInstanceRuntimeConfig(manager.Log, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize Instance OCI client: %w", err)
		}
		return defaultInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.Instance](config),
		}
	}
}

func newInstanceRuntimeConfig(log loggerutil.OSOKLogger, sdkClient instanceOCIClient) generatedruntime.Config[*corev1beta1.Instance] {
	return generatedruntime.Config[*corev1beta1.Instance]{
		Kind:      "Instance",
		SDKName:   "Instance",
		Log:       log,
		Semantics: newInstanceRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.LaunchInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.LaunchInstance(ctx, *request.(*coresdk.LaunchInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "LaunchInstanceDetails", RequestName: "LaunchInstanceDetails", Contribution: "body", PreferResourceID: false},
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.GetInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.GetInstance(ctx, *request.(*coresdk.GetInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "InstanceId", RequestName: "instanceId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.ListInstancesRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.ListInstances(ctx, *request.(*coresdk.ListInstancesRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
				{FieldName: "AvailabilityDomain", RequestName: "availabilityDomain", Contribution: "query", PreferResourceID: false},
				{FieldName: "CapacityReservationId", RequestName: "capacityReservationId", Contribution: "query", PreferResourceID: false},
				{FieldName: "ComputeClusterId", RequestName: "computeClusterId", Contribution: "query", PreferResourceID: false},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
				{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.UpdateInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.UpdateInstance(ctx, *request.(*coresdk.UpdateInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "InstanceId", RequestName: "instanceId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateInstanceDetails", RequestName: "UpdateInstanceDetails", Contribution: "body", PreferResourceID: false},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.TerminateInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.TerminateInstance(ctx, *request.(*coresdk.TerminateInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "InstanceId", RequestName: "instanceId", Contribution: "path", PreferResourceID: true},
				{FieldName: "PreserveBootVolume", RequestName: "preserveBootVolume", Contribution: "query", PreferResourceID: false},
				{FieldName: "PreserveDataVolumesCreatedAtLaunch", RequestName: "preserveDataVolumesCreatedAtLaunch", Contribution: "query", PreferResourceID: false},
			},
		},
	}
}
