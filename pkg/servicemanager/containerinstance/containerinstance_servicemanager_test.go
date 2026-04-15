/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicontainerinstances "github.com/oracle/oci-go-sdk/v65/containerinstances"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	containerinstancemanager "github.com/oracle/oci-service-operator/pkg/servicemanager/containerinstance"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(context.Context, string, string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

type fakeOciClient struct {
	createFn      func(context.Context, ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error)
	getFn         func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error)
	listFn        func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error)
	updateFn      func(context.Context, ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error)
	deleteFn      func(context.Context, ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error)
	createCalled  bool
	updateCalled  bool
	deleteCalled  bool
	createRequest *ocicontainerinstances.CreateContainerInstanceRequest
	updateRequest *ocicontainerinstances.UpdateContainerInstanceRequest
}

func (f *fakeOciClient) CreateContainerInstance(ctx context.Context, req ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
	f.createCalled = true
	f.createRequest = &req
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	id := "ocid1.containerinstance.oc1..new"
	return ocicontainerinstances.CreateContainerInstanceResponse{
		ContainerInstance: ocicontainerinstances.ContainerInstance{
			Id:             common.String(id),
			LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
		},
	}, nil
}

func (f *fakeOciClient) GetContainerInstance(ctx context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	id := "ocid1.containerinstance.oc1..existing"
	if req.ContainerInstanceId != nil {
		id = *req.ContainerInstanceId
	}
	return ocicontainerinstances.GetContainerInstanceResponse{
		ContainerInstance: ocicontainerinstances.ContainerInstance{
			Id:             common.String(id),
			DisplayName:    common.String("test-ci"),
			LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
		},
	}, nil
}

func (f *fakeOciClient) ListContainerInstances(ctx context.Context, req ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return ocicontainerinstances.ListContainerInstancesResponse{
		ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
			Items: []ocicontainerinstances.ContainerInstanceSummary{},
		},
	}, nil
}

func (f *fakeOciClient) UpdateContainerInstance(ctx context.Context, req ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error) {
	f.updateCalled = true
	f.updateRequest = &req
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return ocicontainerinstances.UpdateContainerInstanceResponse{}, nil
}

func (f *fakeOciClient) DeleteContainerInstance(ctx context.Context, req ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error) {
	f.deleteCalled = true
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return ocicontainerinstances.DeleteContainerInstanceResponse{}, nil
}

func newTestManager(ociClient *fakeOciClient) *containerinstancemanager.ContainerInstanceServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := containerinstancemanager.NewContainerInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{},
		nil,
		log,
	)
	containerinstancemanager.ExportSetClientForTest(mgr, ociClient)
	return mgr
}

func makeContainerInstanceSpec(displayName string) *containerinstancesv1beta1.ContainerInstance {
	ci := &containerinstancesv1beta1.ContainerInstance{}
	ci.Name = "test-ci"
	ci.Namespace = "default"
	ci.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	ci.Spec.AvailabilityDomain = "AD-1"
	ci.Spec.Shape = "CI.Standard.E4.Flex"
	ci.Spec.ShapeConfig = containerinstancesv1beta1.ContainerInstanceShapeConfig{
		Ocpus:       1,
		MemoryInGBs: 8,
	}
	ci.Spec.Containers = []containerinstancesv1beta1.ContainerInstanceContainer{
		{ImageUrl: "busybox:latest"},
	}
	ci.Spec.Vnics = []containerinstancesv1beta1.ContainerInstanceVnic{
		{SubnetId: "ocid1.subnet.oc1..xxx"},
	}
	ci.Spec.DisplayName = displayName
	return ci
}

func TestDeleteNoOCID(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})
	ci := makeContainerInstanceSpec("")

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestGetCrdStatusReturnsStatus(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..tracked"

	status, err := mgr.GetCrdStatus(ci)
	assert.NoError(t, err)
	assert.Equal(t, "ocid1.containerinstance.oc1..tracked", string(status.Ocid))
}

func TestGetCrdStatusWrongType(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})

	_, err := mgr.GetCrdStatus(&corev1.ConfigMap{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestCreateOrUpdateBadType(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})

	resp, err := mgr.CreateOrUpdate(context.Background(), &corev1.ConfigMap{}, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdateCreatePath(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
		createFn: func(context.Context, ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{
				OpcRequestId: common.String("opc-create-1"),
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..created"),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, ociClient.createCalled)
	assert.Equal(t, "opc-create-1", ci.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, "ocid1.containerinstance.oc1..created", string(ci.Status.OsokStatus.Ocid))
}

func TestCreateOrUpdateListError(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{}, errors.New("list failed")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdateCreateError(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
		createFn: func(context.Context, ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{}, errors.New("create failed")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdateExistingInstanceFound(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..existing"
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{
						{
							Id:             common.String(existingID),
							LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(existingID),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, ociClient.createCalled)
	assert.False(t, ociClient.updateCalled)
	assert.Equal(t, existingID, string(ci.Status.OsokStatus.Ocid))
}

func TestCreateOrUpdateFailedLifecycleState(t *testing.T) {
	ociClient := &fakeOciClient{
		createFn: func(context.Context, ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..failed"),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateFailed,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, string(ocicontainerinstances.ContainerInstanceLifecycleStateFailed), ci.Status.LifecycleState)
}

func TestCreateOrUpdateTrackedInstanceUpdatesMutableFields(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	getCalls := 0
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			getCalls++
			displayName := "old-name"
			if getCalls > 1 {
				displayName = "new-name"
			}
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(existingID),
					DisplayName:    common.String(displayName),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
		updateFn: func(context.Context, ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error) {
			return ocicontainerinstances.UpdateContainerInstanceResponse{
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("new-name")
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, ociClient.updateCalled)
	assert.Equal(t, "opc-update-1", ci.Status.OsokStatus.OpcRequestID)
	if assert.NotNil(t, ociClient.updateRequest) {
		assert.Equal(t, existingID, *ociClient.updateRequest.ContainerInstanceId)
		assert.Equal(t, "new-name", *ociClient.updateRequest.UpdateContainerInstanceDetails.DisplayName)
	}
	assert.Equal(t, "new-name", ci.Status.DisplayName)
}

func TestCreateOrUpdateListServiceErrorCapturesOpcRequestID(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{}, errortest.NewServiceError(409, "IncorrectState", "list conflict")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, "opc-request-id", ci.Status.OsokStatus.OpcRequestID)
}

func TestCreateOrUpdateTrackedNotFoundFallsBackToCreate(t *testing.T) {
	trackedID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "tracked container instance missing")
		},
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
		createFn: func(context.Context, ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..created"),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Status.Id = trackedID
	ci.Status.OsokStatus.Ocid = shared.OCID(trackedID)

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, ociClient.createCalled)
	assert.Equal(t, "ocid1.containerinstance.oc1..created", ci.Status.Id)
	assert.Equal(t, "ocid1.containerinstance.oc1..created", string(ci.Status.OsokStatus.Ocid))
}

func TestDeleteWithOCID(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.False(t, done)
	assert.True(t, ociClient.deleteCalled)
}

func TestDeleteAlreadyDeleted(t *testing.T) {
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..del"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateDeleted,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, ociClient.deleteCalled)
}

func TestDeleteTreatsDeleteNotFoundAsDeleted(t *testing.T) {
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..del"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
		deleteFn: func(context.Context, ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error) {
			return ocicontainerinstances.DeleteContainerInstanceResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "container instance already gone")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, ociClient.deleteCalled)
	assert.Equal(t, "opc-request-id", ci.Status.OsokStatus.OpcRequestID)
}

func TestDeleteError(t *testing.T) {
	ociClient := &fakeOciClient{
		deleteFn: func(context.Context, ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error) {
			return ocicontainerinstances.DeleteContainerInstanceResponse{}, errors.New("delete failed")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestGetContainerInstanceOcidNoDisplayName(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})
	ci := makeContainerInstanceSpec("")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

func TestGetContainerInstanceOcidNotFound(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("missing")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

func TestGetContainerInstanceOcidFoundActive(t *testing.T) {
	foundID := "ocid1.containerinstance.oc1..active"
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{
						{
							Id:             common.String(foundID),
							LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
						},
					},
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	if assert.NotNil(t, ocid) {
		assert.Equal(t, foundID, string(*ocid))
	}
}

func TestGetContainerInstanceOcidFoundCreating(t *testing.T) {
	foundID := "ocid1.containerinstance.oc1..creating"
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{
						{
							Id:             common.String(foundID),
							LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateCreating,
						},
					},
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	if assert.NotNil(t, ocid) {
		assert.Equal(t, foundID, string(*ocid))
	}
}

func TestGetContainerInstanceOcidListError(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{}, errors.New("network error")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.Error(t, err)
	assert.Nil(t, ocid)
}

func TestCreateContainerInstanceWithVolumeMounts(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.Containers[0].VolumeMounts = []containerinstancesv1beta1.ContainerInstanceContainerVolumeMount{
		{
			MountPath:  "/data",
			VolumeName: "my-volume",
			SubPath:    "data",
			IsReadOnly: true,
		},
	}

	_, err := mgr.CreateContainerInstance(context.Background(), *ci)
	assert.NoError(t, err)
	if assert.NotNil(t, ociClient.createRequest) {
		assert.Len(t, ociClient.createRequest.CreateContainerInstanceDetails.Containers, 1)
		assert.Len(t, ociClient.createRequest.CreateContainerInstanceDetails.Containers[0].VolumeMounts, 1)
		vm := ociClient.createRequest.CreateContainerInstanceDetails.Containers[0].VolumeMounts[0]
		assert.Equal(t, "/data", *vm.MountPath)
		assert.Equal(t, "my-volume", *vm.VolumeName)
		assert.Equal(t, "data", *vm.SubPath)
		assert.Equal(t, true, *vm.IsReadOnly)
	}
}

func TestCreateContainerInstanceWithImagePullSecrets(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.ImagePullSecrets = []containerinstancesv1beta1.ContainerInstanceImagePullSecret{
		{
			RegistryEndpoint: "registry.example.com",
			Username:         "myuser",
			Password:         "mypassword",
		},
	}

	_, err := mgr.CreateContainerInstance(context.Background(), *ci)
	assert.NoError(t, err)
	if assert.NotNil(t, ociClient.createRequest) {
		assert.Len(t, ociClient.createRequest.CreateContainerInstanceDetails.ImagePullSecrets, 1)
		secret, ok := ociClient.createRequest.CreateContainerInstanceDetails.ImagePullSecrets[0].(ocicontainerinstances.CreateBasicImagePullSecretDetails)
		if assert.True(t, ok) {
			assert.Equal(t, "registry.example.com", *secret.RegistryEndpoint)
			assert.Equal(t, "myuser", *secret.Username)
			assert.Equal(t, "mypassword", *secret.Password)
		}
	}
}
