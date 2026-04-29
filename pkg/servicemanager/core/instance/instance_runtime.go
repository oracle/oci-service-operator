/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package instance

import (
	"context"

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
	registerInstanceRuntimeHooksMutator(func(_ *InstanceServiceManager, hooks *InstanceRuntimeHooks) {
		applyInstanceRuntimeHooks(hooks)
	})
}

func applyInstanceRuntimeHooks(_ *InstanceRuntimeHooks) {
}

func newInstanceRuntimeConfig(log loggerutil.OSOKLogger, sdkClient instanceOCIClient) generatedruntime.Config[*corev1beta1.Instance] {
	hooks := newInstanceRuntimeHooksWithOCIClient(sdkClient)
	applyInstanceRuntimeHooks(&hooks)
	return buildInstanceGeneratedRuntimeConfig(&InstanceServiceManager{Log: log}, hooks)
}

func newInstanceRuntimeHooksWithOCIClient(client instanceOCIClient) InstanceRuntimeHooks {
	return InstanceRuntimeHooks{
		Semantics: newInstanceRuntimeSemantics(),
		Create: runtimeOperationHooks[coresdk.LaunchInstanceRequest, coresdk.LaunchInstanceResponse]{
			Fields: instanceCreateFields(),
			Call: func(ctx context.Context, request coresdk.LaunchInstanceRequest) (coresdk.LaunchInstanceResponse, error) {
				return client.LaunchInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[coresdk.GetInstanceRequest, coresdk.GetInstanceResponse]{
			Fields: instanceGetFields(),
			Call: func(ctx context.Context, request coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error) {
				return client.GetInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[coresdk.ListInstancesRequest, coresdk.ListInstancesResponse]{
			Fields: instanceListFields(),
			Call: func(ctx context.Context, request coresdk.ListInstancesRequest) (coresdk.ListInstancesResponse, error) {
				return client.ListInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[coresdk.UpdateInstanceRequest, coresdk.UpdateInstanceResponse]{
			Fields: instanceUpdateFields(),
			Call: func(ctx context.Context, request coresdk.UpdateInstanceRequest) (coresdk.UpdateInstanceResponse, error) {
				return client.UpdateInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[coresdk.TerminateInstanceRequest, coresdk.TerminateInstanceResponse]{
			Fields: instanceDeleteFields(),
			Call: func(ctx context.Context, request coresdk.TerminateInstanceRequest) (coresdk.TerminateInstanceResponse, error) {
				return client.TerminateInstance(ctx, request)
			},
		},
	}
}

func instanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "LaunchInstanceDetails", RequestName: "LaunchInstanceDetails", Contribution: "body"},
	}
}

func instanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "InstanceId", RequestName: "instanceId", Contribution: "path", PreferResourceID: true},
	}
}

func instanceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "AvailabilityDomain", RequestName: "availabilityDomain", Contribution: "query"},
		{FieldName: "CapacityReservationId", RequestName: "capacityReservationId", Contribution: "query"},
		{FieldName: "ComputeClusterId", RequestName: "computeClusterId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func instanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "InstanceId", RequestName: "instanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateInstanceDetails", RequestName: "UpdateInstanceDetails", Contribution: "body"},
	}
}

func instanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "InstanceId", RequestName: "instanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "PreserveBootVolume", RequestName: "preserveBootVolume", Contribution: "query"},
		{FieldName: "PreserveDataVolumesCreatedAtLaunch", RequestName: "preserveDataVolumesCreatedAtLaunch", Contribution: "query"},
	}
}
