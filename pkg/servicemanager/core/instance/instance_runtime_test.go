/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package instance

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeInstanceOCIClient struct {
	launchFn    func(context.Context, coresdk.LaunchInstanceRequest) (coresdk.LaunchInstanceResponse, error)
	getFn       func(context.Context, coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error)
	listFn      func(context.Context, coresdk.ListInstancesRequest) (coresdk.ListInstancesResponse, error)
	updateFn    func(context.Context, coresdk.UpdateInstanceRequest) (coresdk.UpdateInstanceResponse, error)
	terminateFn func(context.Context, coresdk.TerminateInstanceRequest) (coresdk.TerminateInstanceResponse, error)
}

func (f *fakeInstanceOCIClient) LaunchInstance(ctx context.Context, req coresdk.LaunchInstanceRequest) (coresdk.LaunchInstanceResponse, error) {
	if f.launchFn != nil {
		return f.launchFn(ctx, req)
	}
	return coresdk.LaunchInstanceResponse{}, nil
}

func (f *fakeInstanceOCIClient) GetInstance(ctx context.Context, req coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetInstanceResponse{}, nil
}

func (f *fakeInstanceOCIClient) ListInstances(ctx context.Context, req coresdk.ListInstancesRequest) (coresdk.ListInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return coresdk.ListInstancesResponse{}, nil
}

func (f *fakeInstanceOCIClient) UpdateInstance(ctx context.Context, req coresdk.UpdateInstanceRequest) (coresdk.UpdateInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateInstanceResponse{}, nil
}

func (f *fakeInstanceOCIClient) TerminateInstance(ctx context.Context, req coresdk.TerminateInstanceRequest) (coresdk.TerminateInstanceResponse, error) {
	if f.terminateFn != nil {
		return f.terminateFn(ctx, req)
	}
	return coresdk.TerminateInstanceResponse{}, nil
}

func newInstanceTestManager(client instanceOCIClient) *InstanceServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewInstanceServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(defaultInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.Instance](newInstanceRuntimeConfig(log, client)),
		})
	}
	return manager
}

func makeSpecInstance() *corev1beta1.Instance {
	return &corev1beta1.Instance{
		Spec: corev1beta1.InstanceSpec{
			AvailabilityDomain: "AD-1",
			CompartmentId:      "ocid1.compartment.oc1..example",
			DisplayName:        "test-instance",
			FreeformTags: map[string]string{
				"run":      "1",
				"scenario": "create",
			},
			Shape: "VM.Standard.E4.Flex",
			ShapeConfig: corev1beta1.InstanceShapeConfig{
				Ocpus:       1,
				MemoryInGBs: 16,
			},
			SourceDetails: corev1beta1.InstanceSourceDetails{
				SourceType: "image",
				ImageId:    "ocid1.image.oc1..example",
			},
			SubnetId: "ocid1.subnet.oc1..example",
		},
	}
}

func makeSDKInstance(id string, lifecycleState coresdk.InstanceLifecycleStateEnum, tags map[string]string) coresdk.Instance {
	created := common.SDKTime{Time: time.Date(2026, 4, 6, 4, 11, 1, 0, time.UTC)}
	return coresdk.Instance{
		Id:                 common.String(id),
		AvailabilityDomain: common.String("AD-1"),
		CompartmentId:      common.String("ocid1.compartment.oc1..example"),
		DisplayName:        common.String("test-instance"),
		FreeformTags:       tags,
		LifecycleState:     lifecycleState,
		Shape:              common.String("VM.Standard.E4.Flex"),
		ShapeConfig: &coresdk.InstanceShapeConfig{
			Ocpus:       common.Float32(1),
			MemoryInGBs: common.Float32(16),
			Vcpus:       common.Int(2),
		},
		SourceDetails: coresdk.InstanceSourceViaImageDetails{
			ImageId: common.String("ocid1.image.oc1..example"),
		},
		TimeCreated: &created,
	}
}

func TestInstanceDelete_TerminatedCompletesDeletion(t *testing.T) {
	ocid := "ocid1.instance.oc1..delete"
	terminateCalls := 0
	getCalls := 0

	manager := newInstanceTestManager(&fakeInstanceOCIClient{
		terminateFn: func(_ context.Context, req coresdk.TerminateInstanceRequest) (coresdk.TerminateInstanceResponse, error) {
			terminateCalls++
			if !assert.NotNil(t, req.InstanceId) {
				return coresdk.TerminateInstanceResponse{}, nil
			}
			assert.Equal(t, ocid, *req.InstanceId)
			return coresdk.TerminateInstanceResponse{}, nil
		},
		getFn: func(_ context.Context, req coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error) {
			getCalls++
			if !assert.NotNil(t, req.InstanceId) {
				return coresdk.GetInstanceResponse{}, nil
			}
			assert.Equal(t, ocid, *req.InstanceId)
			return coresdk.GetInstanceResponse{
				Instance: makeSDKInstance(ocid, coresdk.InstanceLifecycleStateTerminated, map[string]string{
					"run":      "1",
					"scenario": "create",
				}),
			}, nil
		},
	})

	resource := makeSpecInstance()
	resource.Status.Id = ocid
	resource.Status.LifecycleState = string(coresdk.InstanceLifecycleStateRunning)
	resource.Status.OsokStatus.Ocid = shared.OCID(ocid)

	deleted, err := manager.Delete(context.Background(), resource)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, deleted)
	assert.Equal(t, 0, terminateCalls)
	assert.Equal(t, 1, getCalls)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, string(coresdk.InstanceLifecycleStateTerminated), resource.Status.LifecycleState)
}

func TestInstanceCreateOrUpdate_FreeformTagChangeUsesInPlaceUpdate(t *testing.T) {
	ocid := "ocid1.instance.oc1..update"
	launchCalls := 0
	updateCalls := 0
	getCalls := 0
	var capturedUpdate coresdk.UpdateInstanceRequest

	manager := newInstanceTestManager(&fakeInstanceOCIClient{
		launchFn: func(_ context.Context, _ coresdk.LaunchInstanceRequest) (coresdk.LaunchInstanceResponse, error) {
			launchCalls++
			return coresdk.LaunchInstanceResponse{}, nil
		},
		getFn: func(_ context.Context, req coresdk.GetInstanceRequest) (coresdk.GetInstanceResponse, error) {
			getCalls++
			if !assert.NotNil(t, req.InstanceId) {
				return coresdk.GetInstanceResponse{}, nil
			}
			assert.Equal(t, ocid, *req.InstanceId)
			tags := map[string]string{
				"run":      "1",
				"scenario": "create",
			}
			if getCalls > 1 {
				tags = map[string]string{
					"revision": "2",
					"run":      "1",
					"scenario": "update",
				}
			}
			return coresdk.GetInstanceResponse{
				Instance: makeSDKInstance(ocid, coresdk.InstanceLifecycleStateRunning, tags),
			}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateInstanceRequest) (coresdk.UpdateInstanceResponse, error) {
			updateCalls++
			capturedUpdate = req
			return coresdk.UpdateInstanceResponse{
				Instance: makeSDKInstance(ocid, coresdk.InstanceLifecycleStateRunning, map[string]string{
					"revision": "2",
					"run":      "1",
					"scenario": "update",
				}),
			}, nil
		},
	})

	resource := makeSpecInstance()
	resource.Spec.FreeformTags = map[string]string{
		"revision": "2",
		"run":      "1",
		"scenario": "update",
	}
	resource.Status.Id = ocid
	resource.Status.LifecycleState = string(coresdk.InstanceLifecycleStateRunning)
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.AvailabilityDomain = resource.Spec.AvailabilityDomain
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.Shape = resource.Spec.Shape
	resource.Status.ShapeConfig = resource.Spec.ShapeConfig
	resource.Status.SourceDetails = resource.Spec.SourceDetails
	resource.Status.FreeformTags = map[string]string{
		"run":      "1",
		"scenario": "create",
	}
	resource.Status.TimeCreated = "2026-04-06T04:11:01.163Z"
	resource.Status.OsokStatus.Ocid = shared.OCID(ocid)

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 0, launchCalls)
	assert.Equal(t, 1, updateCalls)
	assert.GreaterOrEqual(t, getCalls, 2)
	if !assert.NotNil(t, capturedUpdate.InstanceId) {
		return
	}
	assert.Equal(t, ocid, *capturedUpdate.InstanceId)
	assert.Equal(t, resource.Spec.FreeformTags, capturedUpdate.UpdateInstanceDetails.FreeformTags)
	assert.Nil(t, capturedUpdate.UpdateInstanceDetails.DisplayName)
	assert.Nil(t, capturedUpdate.UpdateInstanceDetails.DefinedTags)
	assert.Equal(t, resource.Spec.FreeformTags, resource.Status.FreeformTags)
	assert.Equal(t, string(shared.Active), resource.Status.OsokStatus.Reason)
}
