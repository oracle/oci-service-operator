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
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
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
	createFn       func(context.Context, ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error)
	getFn          func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error)
	listFn         func(context.Context, ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error)
	getContainerFn func(context.Context, ocicontainerinstances.GetContainerRequest) (ocicontainerinstances.GetContainerResponse, error)
	updateFn       func(context.Context, ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error)
	deleteFn       func(context.Context, ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error)
	createCalled   bool
	updateCalled   bool
	deleteCalled   bool
	createRequest  *ocicontainerinstances.CreateContainerInstanceRequest
	updateRequest  *ocicontainerinstances.UpdateContainerInstanceRequest
	listRequests   []ocicontainerinstances.ListContainerInstancesRequest
	deleteCalls    int
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
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return ocicontainerinstances.ListContainerInstancesResponse{
		ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
			Items: []ocicontainerinstances.ContainerInstanceSummary{},
		},
	}, nil
}

func (f *fakeOciClient) GetContainer(ctx context.Context, req ocicontainerinstances.GetContainerRequest) (ocicontainerinstances.GetContainerResponse, error) {
	if f.getContainerFn != nil {
		return f.getContainerFn(ctx, req)
	}
	id := "ocid1.container.oc1..container"
	if req.ContainerId != nil {
		id = *req.ContainerId
	}
	return ocicontainerinstances.GetContainerResponse{
		Container: ocicontainerinstances.Container{
			Id:                  common.String(id),
			DisplayName:         common.String("container"),
			CompartmentId:       common.String("ocid1.compartment.oc1..xxx"),
			AvailabilityDomain:  common.String("AD-1"),
			ContainerInstanceId: common.String("ocid1.containerinstance.oc1..tracked"),
			ImageUrl:            common.String("busybox:latest"),
			LifecycleState:      ocicontainerinstances.ContainerLifecycleStateActive,
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
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return ocicontainerinstances.DeleteContainerInstanceResponse{}, nil
}

type fakeVnicClient struct {
	getFn func(context.Context, coresdk.GetVnicRequest) (coresdk.GetVnicResponse, error)
}

func (f *fakeVnicClient) GetVnic(ctx context.Context, req coresdk.GetVnicRequest) (coresdk.GetVnicResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	id := "ocid1.vnic.oc1..vnic"
	if req.VnicId != nil {
		id = *req.VnicId
	}
	return coresdk.GetVnicResponse{
		Vnic: coresdk.Vnic{
			Id:                 common.String(id),
			AvailabilityDomain: common.String("AD-1"),
			CompartmentId:      common.String("ocid1.compartment.oc1..xxx"),
			SubnetId:           common.String("ocid1.subnet.oc1..xxx"),
			LifecycleState:     coresdk.VnicLifecycleStateAvailable,
		},
	}, nil
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
	containerinstancemanager.ExportSetVnicClientForTest(mgr, &fakeVnicClient{})
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

func matchingObservedContainerInstance(id string, displayName string) ocicontainerinstances.ContainerInstance {
	return ocicontainerinstances.ContainerInstance{
		Id:                     common.String(id),
		DisplayName:            common.String(displayName),
		CompartmentId:          common.String("ocid1.compartment.oc1..xxx"),
		AvailabilityDomain:     common.String("AD-1"),
		LifecycleState:         ocicontainerinstances.ContainerInstanceLifecycleStateActive,
		Containers:             []ocicontainerinstances.ContainerInstanceContainer{{ContainerId: common.String("ocid1.container.oc1..container")}},
		ContainerCount:         common.Int(1),
		Shape:                  common.String("CI.Standard.E4.Flex"),
		ShapeConfig:            &ocicontainerinstances.ContainerInstanceShapeConfig{Ocpus: common.Float32(1), MemoryInGBs: common.Float32(8)},
		Vnics:                  []ocicontainerinstances.ContainerVnic{{VnicId: common.String("ocid1.vnic.oc1..vnic")}},
		ContainerRestartPolicy: ocicontainerinstances.ContainerInstanceContainerRestartPolicyAlways,
	}
}

func TestContainerInstanceRuntimeSemanticsEncodesHandwrittenLifecycleContract(t *testing.T) {
	semantics := containerinstancemanager.ExportRuntimeSemanticsForTest()

	assert.Equal(t, "containerinstances", semantics.FormalService)
	assert.Equal(t, "containerinstance", semantics.FormalSlug)
	assert.Equal(t, "required", semantics.StatusProjection)
	assert.Equal(t, "none", semantics.SecretSideEffects)
	assert.Equal(t, "retain-until-confirmed-delete", semantics.FinalizerPolicy)
	if assert.NotNil(t, semantics.Async) {
		assert.Equal(t, "lifecycle", semantics.Async.Strategy)
		assert.Equal(t, "handwritten", semantics.Async.Runtime)
		assert.Equal(t, "lifecycle", semantics.Async.FormalClassification)
	}
	assert.Equal(t, []string{"CREATING"}, semantics.Lifecycle.ProvisioningStates)
	assert.Equal(t, []string{"UPDATING"}, semantics.Lifecycle.UpdatingStates)
	assert.Equal(t, []string{"ACTIVE", "INACTIVE"}, semantics.Lifecycle.ActiveStates)
	assert.Equal(t, []string{"DELETING"}, semantics.Delete.PendingStates)
	assert.Equal(t, []string{"DELETED", "NOT_FOUND"}, semantics.Delete.TerminalStates)
	assert.Equal(t, []string{"definedTags", "displayName", "freeformTags"}, semantics.Mutation.Mutable)
	assert.Contains(t, semantics.Mutation.ForceNew, "containers")
	assert.NotEmpty(t, semantics.Unsupported)
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

func TestCreateOrUpdateClassifiesLifecycleStates(t *testing.T) {
	tests := []struct {
		name        string
		state       ocicontainerinstances.ContainerInstanceLifecycleStateEnum
		wantReason  shared.OSOKConditionType
		wantSuccess bool
		wantRequeue bool
		wantAsync   bool
		wantPhase   shared.OSOKAsyncPhase
		wantClass   shared.OSOKAsyncNormalizedClass
	}{
		{
			name:        "creating",
			state:       ocicontainerinstances.ContainerInstanceLifecycleStateCreating,
			wantReason:  shared.Provisioning,
			wantSuccess: true,
			wantRequeue: true,
			wantAsync:   true,
			wantPhase:   shared.OSOKAsyncPhaseCreate,
			wantClass:   shared.OSOKAsyncClassPending,
		},
		{
			name:        "updating",
			state:       ocicontainerinstances.ContainerInstanceLifecycleStateUpdating,
			wantReason:  shared.Updating,
			wantSuccess: true,
			wantRequeue: true,
			wantAsync:   true,
			wantPhase:   shared.OSOKAsyncPhaseUpdate,
			wantClass:   shared.OSOKAsyncClassPending,
		},
		{
			name:        "deleting",
			state:       ocicontainerinstances.ContainerInstanceLifecycleStateDeleting,
			wantReason:  shared.Terminating,
			wantSuccess: true,
			wantRequeue: true,
			wantAsync:   true,
			wantPhase:   shared.OSOKAsyncPhaseDelete,
			wantClass:   shared.OSOKAsyncClassPending,
		},
		{
			name:        "active",
			state:       ocicontainerinstances.ContainerInstanceLifecycleStateActive,
			wantReason:  shared.Active,
			wantSuccess: true,
		},
		{
			name:        "inactive",
			state:       ocicontainerinstances.ContainerInstanceLifecycleStateInactive,
			wantReason:  shared.Active,
			wantSuccess: true,
		},
		{
			name:       "failed",
			state:      ocicontainerinstances.ContainerInstanceLifecycleStateFailed,
			wantReason: shared.Failed,
			wantAsync:  true,
			wantPhase:  shared.OSOKAsyncPhaseCreate,
			wantClass:  shared.OSOKAsyncClassFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existingID := "ocid1.containerinstance.oc1..tracked"
			ociClient := &fakeOciClient{
				getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
					return ocicontainerinstances.GetContainerInstanceResponse{
						ContainerInstance: ocicontainerinstances.ContainerInstance{
							Id:             common.String(existingID),
							DisplayName:    common.String("test-ci"),
							LifecycleState: tt.state,
						},
					}, nil
				},
			}
			mgr := newTestManager(ociClient)
			ci := makeContainerInstanceSpec("test-ci")
			ci.Status.Id = existingID

			resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSuccess, resp.IsSuccessful)
			assert.Equal(t, tt.wantRequeue, resp.ShouldRequeue)
			assert.Equal(t, string(tt.state), ci.Status.LifecycleState)
			assert.Equal(t, string(tt.wantReason), ci.Status.OsokStatus.Reason)
			assertConditionType(t, ci.Status.OsokStatus.Conditions, tt.wantReason)
			if tt.wantAsync {
				requireAsyncCurrent(t, ci, tt.wantPhase, string(tt.state), tt.wantClass)
			} else {
				assert.Nil(t, ci.Status.OsokStatus.Async.Current)
			}
		})
	}
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
				ContainerInstance: matchingObservedContainerInstance(existingID, displayName),
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

func TestCreateOrUpdateTrackedDeletedFallsBackToCreate(t *testing.T) {
	trackedID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(trackedID),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateDeleted,
				},
			}, nil
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

func TestCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(existingID),
					DisplayName:    common.String("old-name"),
					Shape:          common.String("CI.Standard.A1.Flex"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("new-name")
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "shape cannot be updated in place")
}

func TestCreateOrUpdateRejectsContainerImageDriftWithoutMutableUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: matchingObservedContainerInstance(existingID, "test-ci"),
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.Containers[0].ImageUrl = "nginx:latest"
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "containers[0].imageUrl cannot be updated in place")
}

func TestCreateOrUpdateRejectsContainerImageDriftBeforeMutableUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: matchingObservedContainerInstance(existingID, "old-name"),
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("new-name")
	ci.Spec.Containers[0].ImageUrl = "nginx:latest"
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "containers[0].imageUrl cannot be updated in place")
}

func TestCreateOrUpdateRejectsVnicDriftBeforeMutableUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: matchingObservedContainerInstance(existingID, "old-name"),
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	containerinstancemanager.ExportSetVnicClientForTest(mgr, &fakeVnicClient{
		getFn: func(context.Context, coresdk.GetVnicRequest) (coresdk.GetVnicResponse, error) {
			return coresdk.GetVnicResponse{
				Vnic: coresdk.Vnic{
					Id:                 common.String("ocid1.vnic.oc1..vnic"),
					AvailabilityDomain: common.String("AD-1"),
					CompartmentId:      common.String("ocid1.compartment.oc1..xxx"),
					SubnetId:           common.String("ocid1.subnet.oc1..different"),
					LifecycleState:     coresdk.VnicLifecycleStateAvailable,
				},
			}, nil
		},
	})
	ci := makeContainerInstanceSpec("new-name")
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "vnics[0].subnetId cannot be updated in place")
}

func TestCreateOrUpdateRejectsVolumeDriftWithoutMutableUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			instance := matchingObservedContainerInstance(existingID, "test-ci")
			instance.Volumes = []ocicontainerinstances.ContainerVolume{
				ocicontainerinstances.ContainerEmptyDirVolume{
					Name:         common.String("cache"),
					BackingStore: ocicontainerinstances.ContainerEmptyDirVolumeBackingStoreMemory,
				},
			}
			instance.VolumeCount = common.Int(1)
			return ocicontainerinstances.GetContainerInstanceResponse{ContainerInstance: instance}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.Volumes = []containerinstancesv1beta1.ContainerInstanceVolume{
		{Name: "cache", BackingStore: string(ocicontainerinstances.ContainerEmptyDirVolumeBackingStoreEphemeralStorage)},
	}
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "volumes[0] cannot be updated in place")
}

func TestCreateOrUpdateRejectsVaultImagePullSecretDriftWithoutMutableUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			instance := matchingObservedContainerInstance(existingID, "test-ci")
			instance.ImagePullSecrets = []ocicontainerinstances.ImagePullSecret{
				ocicontainerinstances.VaultImagePullSecret{
					RegistryEndpoint: common.String("registry.example.com"),
					SecretId:         common.String("ocid1.vaultsecret.oc1..old"),
				},
			}
			return ocicontainerinstances.GetContainerInstanceResponse{ContainerInstance: instance}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.ImagePullSecrets = []containerinstancesv1beta1.ContainerInstanceImagePullSecret{
		{
			RegistryEndpoint: "registry.example.com",
			SecretId:         "ocid1.vaultsecret.oc1..new",
		},
	}
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "imagePullSecrets[0].secretId cannot be updated in place")
}

func TestCreateOrUpdateRejectsBasicImagePullSecretBeforeMutableUpdate(t *testing.T) {
	existingID := "ocid1.containerinstance.oc1..tracked"
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			instance := matchingObservedContainerInstance(existingID, "old-name")
			instance.ImagePullSecrets = []ocicontainerinstances.ImagePullSecret{
				ocicontainerinstances.BasicImagePullSecret{
					RegistryEndpoint: common.String("registry.example.com"),
				},
			}
			return ocicontainerinstances.GetContainerInstanceResponse{ContainerInstance: instance}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("new-name")
	ci.Spec.ImagePullSecrets = []containerinstancesv1beta1.ContainerInstanceImagePullSecret{
		{
			RegistryEndpoint: "registry.example.com",
			Username:         "new-user",
			Password:         "new-password",
		},
	}
	ci.Status.Id = existingID

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, ociClient.updateCalled)
	assert.Contains(t, err.Error(), "imagePullSecrets[0] BASIC credentials cannot be verified before updating mutable fields")
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
	requireAsyncCurrent(t, ci, shared.OSOKAsyncPhaseDelete, string(ocicontainerinstances.ContainerInstanceLifecycleStateDeleting), shared.OSOKAsyncClassPending)
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
	assert.NotNil(t, ci.Status.OsokStatus.DeletedAt)
	requireAsyncCurrent(t, ci, shared.OSOKAsyncPhaseDelete, string(ocicontainerinstances.ContainerInstanceLifecycleStateDeleted), shared.OSOKAsyncClassSucceeded)
}

func TestDeleteKeepsFinalizerUntilDeleteConfirmed(t *testing.T) {
	getCalls := 0
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			getCalls++
			state := ocicontainerinstances.ContainerInstanceLifecycleStateActive
			if getCalls > 1 {
				state = ocicontainerinstances.ContainerInstanceLifecycleStateDeleted
			}
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..del"),
					LifecycleState: state,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.False(t, done)
	assert.Equal(t, 1, ociClient.deleteCalls)
	assert.Equal(t, string(ocicontainerinstances.ContainerInstanceLifecycleStateDeleting), ci.Status.LifecycleState)

	done, err = mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, 1, ociClient.deleteCalls)
	assert.NotNil(t, ci.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(ocicontainerinstances.ContainerInstanceLifecycleStateDeleted), ci.Status.LifecycleState)
}

func TestDeletePendingProjectsTerminatingLifecycle(t *testing.T) {
	ociClient := &fakeOciClient{
		getFn: func(context.Context, ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..del"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateDeleting,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.False(t, done)
	assert.False(t, ociClient.deleteCalled)
	requireAsyncCurrent(t, ci, shared.OSOKAsyncPhaseDelete, string(ocicontainerinstances.ContainerInstanceLifecycleStateDeleting), shared.OSOKAsyncClassPending)
	assert.Equal(t, string(shared.Terminating), ci.Status.OsokStatus.Reason)
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
	if assert.NotEmpty(t, ociClient.listRequests) {
		assert.Equal(t, ocicontainerinstances.ContainerInstanceLifecycleStateActive, ociClient.listRequests[0].LifecycleState)
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

func TestGetContainerInstanceOcidPaginatesReusableLifecycleState(t *testing.T) {
	foundID := "ocid1.containerinstance.oc1..active"
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, req ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			if req.Page == nil {
				return ocicontainerinstances.ListContainerInstancesResponse{
					ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
						Items: []ocicontainerinstances.ContainerInstanceSummary{},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
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
	if assert.Len(t, ociClient.listRequests, 2) {
		assert.Nil(t, ociClient.listRequests[0].Page)
		assert.Equal(t, "page-2", *ociClient.listRequests[1].Page)
		assert.Equal(t, ocicontainerinstances.ContainerInstanceLifecycleStateActive, ociClient.listRequests[0].LifecycleState)
	}
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

func assertConditionType(t *testing.T, conditions []shared.OSOKCondition, conditionType shared.OSOKConditionType) {
	t.Helper()
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return
		}
	}
	t.Fatalf("condition %q not found in %#v", conditionType, conditions)
}

func requireAsyncCurrent(
	t *testing.T,
	ci *containerinstancesv1beta1.ContainerInstance,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := ci.Status.OsokStatus.Async.Current
	if assert.NotNil(t, current) {
		assert.Equal(t, shared.OSOKAsyncSourceLifecycle, current.Source)
		assert.Equal(t, phase, current.Phase)
		assert.Equal(t, rawStatus, current.RawStatus)
		assert.Equal(t, class, current.NormalizedClass)
	}
}
