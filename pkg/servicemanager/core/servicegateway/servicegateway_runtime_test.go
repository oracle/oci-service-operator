/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicegateway

import (
	"context"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	osokcore "github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeServiceGatewayOCIClient struct {
	createFn func(context.Context, coresdk.CreateServiceGatewayRequest) (coresdk.CreateServiceGatewayResponse, error)
	getFn    func(context.Context, coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error)
	updateFn func(context.Context, coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error)
	deleteFn func(context.Context, coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error)
}

func (f *fakeServiceGatewayOCIClient) CreateServiceGateway(ctx context.Context, req coresdk.CreateServiceGatewayRequest) (coresdk.CreateServiceGatewayResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateServiceGatewayResponse{}, nil
}

func (f *fakeServiceGatewayOCIClient) GetServiceGateway(ctx context.Context, req coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetServiceGatewayResponse{}, nil
}

func (f *fakeServiceGatewayOCIClient) UpdateServiceGateway(ctx context.Context, req coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateServiceGatewayResponse{}, nil
}

func (f *fakeServiceGatewayOCIClient) DeleteServiceGateway(ctx context.Context, req coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteServiceGatewayResponse{}, nil
}

type fakeServiceGatewayServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeServiceGatewayServiceError) Error() string          { return f.message }
func (f fakeServiceGatewayServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeServiceGatewayServiceError) GetMessage() string     { return f.message }
func (f fakeServiceGatewayServiceError) GetCode() string        { return f.code }
func (f fakeServiceGatewayServiceError) GetOpcRequestID() string {
	return ""
}

func newServiceGatewayTestManager(client serviceGatewayOCIClient) *ServiceGatewayServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewServiceGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&serviceGatewayRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecServiceGateway() *corev1beta1.ServiceGateway {
	return &corev1beta1.ServiceGateway{
		Spec: corev1beta1.ServiceGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			Services: []corev1beta1.ServiceGatewayService{
				{ServiceId: "ocid1.service.oc1..one"},
				{ServiceId: "ocid1.service.oc1..two"},
			},
			VcnId:        "ocid1.vcn.oc1..example",
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			DisplayName:  "test-service-gateway",
			FreeformTags: map[string]string{"env": "dev"},
			RouteTableId: "ocid1.routetable.oc1..example",
			BlockTraffic: true,
		},
	}
}

func makeSDKServiceGateway(id, displayName string, state coresdk.ServiceGatewayLifecycleStateEnum) coresdk.ServiceGateway {
	return coresdk.ServiceGateway{
		Id:             common.String(id),
		BlockTraffic:   common.Bool(true),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		LifecycleState: state,
		Services: []coresdk.ServiceIdResponseDetails{
			{ServiceId: common.String("ocid1.service.oc1..one"), ServiceName: common.String("svc-one")},
			{ServiceId: common.String("ocid1.service.oc1..two"), ServiceName: common.String("svc-two")},
		},
		VcnId:        common.String("ocid1.vcn.oc1..example"),
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		DisplayName:  common.String(displayName),
		FreeformTags: map[string]string{"env": "dev"},
		RouteTableId: common.String("ocid1.routetable.oc1..example"),
		TimeCreated:  &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateServiceGatewayRequest
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateServiceGatewayRequest) (coresdk.CreateServiceGatewayResponse, error) {
			captured = req
			return coresdk.CreateServiceGatewayResponse{
				ServiceGateway: makeSDKServiceGateway("ocid1.servicegateway.oc1..create", "test-service-gateway", coresdk.ServiceGatewayLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecServiceGateway()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("test-service-gateway"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, common.String("ocid1.routetable.oc1..example"), captured.RouteTableId)
	assert.Len(t, captured.Services, 2)
	assert.Equal(t, "ocid1.service.oc1..one", *captured.Services[0].ServiceId)
	assert.Equal(t, "ocid1.service.oc1..two", *captured.Services[1].ServiceId)
	assert.Equal(t, "ocid1.servicegateway.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.servicegateway.oc1..create", resource.Status.Id)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.True(t, resource.Status.BlockTraffic)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
	assert.Equal(t, []corev1beta1.ServiceGatewayService{
		{ServiceId: "ocid1.service.oc1..one"},
		{ServiceId: "ocid1.service.oc1..two"},
	}, resource.Status.Services)
}

func TestCreateOrUpdate_ObserveByStatusOCID_NoOpWhenStateMatchesIncludingReorderedServices(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		getFn: func(_ context.Context, req coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.servicegateway.oc1..existing", *req.ServiceGatewayId)
			current := makeSDKServiceGateway("ocid1.servicegateway.oc1..existing", "test-service-gateway", coresdk.ServiceGatewayLifecycleStateAvailable)
			current.Services = []coresdk.ServiceIdResponseDetails{
				{ServiceId: common.String("ocid1.service.oc1..two"), ServiceName: common.String("svc-two")},
				{ServiceId: common.String("ocid1.service.oc1..one"), ServiceName: common.String("svc-one")},
			}
			return coresdk.GetServiceGatewayResponse{ServiceGateway: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error) {
			updateCalls++
			return coresdk.UpdateServiceGatewayResponse{}, nil
		},
	})

	resource := makeSpecServiceGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, []corev1beta1.ServiceGatewayService{
		{ServiceId: "ocid1.service.oc1..two"},
		{ServiceId: "ocid1.service.oc1..one"},
	}, resource.Status.Services)
}

func TestCreateOrUpdate_ClearsStaleOptionalStatusFieldsOnProjection(t *testing.T) {
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			current := makeSDKServiceGateway("ocid1.servicegateway.oc1..existing", "test-service-gateway", coresdk.ServiceGatewayLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.RouteTableId = nil
			current.TimeCreated = nil
			current.Services = nil
			current.BlockTraffic = nil
			return coresdk.GetServiceGatewayResponse{ServiceGateway: current}, nil
		},
	})

	resource := makeSpecServiceGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Spec.RouteTableId = ""
	resource.Spec.Services = nil
	resource.Spec.BlockTraffic = false
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.RouteTableId = "ocid1.routetable.oc1..stale"
	resource.Status.TimeCreated = "2026-04-01T00:00:00Z"
	resource.Status.Services = []corev1beta1.ServiceGatewayService{{ServiceId: "ocid1.service.oc1..stale"}}
	resource.Status.BlockTraffic = true

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Equal(t, "", resource.Status.RouteTableId)
	assert.Equal(t, "", resource.Status.TimeCreated)
	assert.Empty(t, resource.Status.Services)
	assert.False(t, resource.Status.BlockTraffic)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdateForScalarAndServices(t *testing.T) {
	var captured coresdk.UpdateServiceGatewayRequest
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			current := makeSDKServiceGateway("ocid1.servicegateway.oc1..existing", "old-name", coresdk.ServiceGatewayLifecycleStateAvailable)
			current.BlockTraffic = common.Bool(false)
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
			current.FreeformTags = map[string]string{"env": "stage"}
			current.RouteTableId = common.String("ocid1.routetable.oc1..old")
			current.Services = []coresdk.ServiceIdResponseDetails{
				{ServiceId: common.String("ocid1.service.oc1..old"), ServiceName: common.String("svc-old")},
			}
			return coresdk.GetServiceGatewayResponse{ServiceGateway: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error) {
			captured = req
			updated := makeSDKServiceGateway("ocid1.servicegateway.oc1..existing", "new-name", coresdk.ServiceGatewayLifecycleStateAvailable)
			return coresdk.UpdateServiceGatewayResponse{ServiceGateway: updated}, nil
		},
	})

	resource := makeSpecServiceGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.servicegateway.oc1..existing", *captured.ServiceGatewayId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, true, *captured.BlockTraffic)
	assert.Equal(t, "ocid1.routetable.oc1..example", *captured.RouteTableId)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Len(t, captured.Services, 2)
	assert.Equal(t, "ocid1.service.oc1..one", *captured.Services[0].ServiceId)
	assert.Equal(t, "ocid1.service.oc1..two", *captured.Services[1].ServiceId)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
	assert.Equal(t, []corev1beta1.ServiceGatewayService{
		{ServiceId: "ocid1.service.oc1..one"},
		{ServiceId: "ocid1.service.oc1..two"},
	}, resource.Status.Services)
}

func TestCreateOrUpdate_RejectsUnsupportedCreateOnlyDrift(t *testing.T) {
	tests := []struct {
		name        string
		mutateSpec  func(*corev1beta1.ServiceGateway)
		expectField string
	}{
		{
			name: "compartmentId",
			mutateSpec: func(resource *corev1beta1.ServiceGateway) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			expectField: "compartmentId",
		},
		{
			name: "vcnId",
			mutateSpec: func(resource *corev1beta1.ServiceGateway) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..different"
			},
			expectField: "vcnId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
					return coresdk.GetServiceGatewayResponse{
						ServiceGateway: makeSDKServiceGateway("ocid1.servicegateway.oc1..existing", "test-service-gateway", coresdk.ServiceGatewayLifecycleStateAvailable),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error) {
					updateCalls++
					return coresdk.UpdateServiceGatewayResponse{}, nil
				},
			})

			resource := makeSpecServiceGateway()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..existing")
			tt.mutateSpec(resource)

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.Error(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.Contains(t, err.Error(), tt.expectField)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_RecreatesOnNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			getCalls++
			return coresdk.GetServiceGatewayResponse{}, fakeServiceGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
		createFn: func(_ context.Context, req coresdk.CreateServiceGatewayRequest) (coresdk.CreateServiceGatewayResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			assert.Len(t, req.Services, 2)
			return coresdk.CreateServiceGatewayResponse{
				ServiceGateway: makeSDKServiceGateway("ocid1.servicegateway.oc1..recreated", "test-service-gateway", coresdk.ServiceGatewayLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecServiceGateway()
	resource.Status.Id = "ocid1.servicegateway.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, shared.OCID("ocid1.servicegateway.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Equal(t, "ocid1.servicegateway.oc1..recreated", resource.Status.Id)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error) {
			assert.Equal(t, "ocid1.servicegateway.oc1..delete", *req.ServiceGatewayId)
			return coresdk.DeleteServiceGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			return coresdk.GetServiceGatewayResponse{}, fakeServiceGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecServiceGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_KeepsFinalizerWhileObservedTerminating(t *testing.T) {
	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error) {
			return coresdk.DeleteServiceGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			return coresdk.GetServiceGatewayResponse{
				ServiceGateway: makeSDKServiceGateway("ocid1.servicegateway.oc1..delete", "test-service-gateway", coresdk.ServiceGatewayLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecServiceGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.servicegateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "TERMINATING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.ServiceGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "ServiceGateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-service-gateway",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.ServiceGatewayStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.servicegateway.oc1..delete"),
			},
		},
	}

	manager := newServiceGatewayTestManager(&fakeServiceGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error) {
			assert.Equal(t, "ocid1.servicegateway.oc1..delete", *req.ServiceGatewayId)
			return coresdk.DeleteServiceGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error) {
			return coresdk.GetServiceGatewayResponse{}, fakeServiceGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemoryServiceGatewayClient(scheme, resource)
	recorder := record.NewFakeRecorder(10)
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	reconciler := &osokcore.BaseReconciler{
		Client:             kubeClient,
		OSOKServiceManager: manager,
		Log:                log,
		Metrics:            &metrics.Metrics{Name: "oci", ServiceName: "core", Logger: log},
		Recorder:           recorder,
		Scheme:             scheme,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: ctrlclient.ObjectKey{Name: "test-service-gateway", Namespace: "default"},
	}, &corev1beta1.ServiceGateway{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredServiceGateway(), osokcore.OSOKFinalizerName))

	events := drainServiceGatewayEvents(recorder)
	assertServiceGatewayEventContains(t, events, "Removed finalizer")
	assertNoServiceGatewayEventContains(t, events, "Failed to delete resource")
}

func drainServiceGatewayEvents(recorder *record.FakeRecorder) []string {
	events := make([]string, 0, len(recorder.Events))
	for {
		select {
		case event := <-recorder.Events:
			events = append(events, event)
		default:
			return events
		}
	}
}

func assertServiceGatewayEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoServiceGatewayEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memoryServiceGatewayClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemoryServiceGatewayClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memoryServiceGatewayClient {
	return &memoryServiceGatewayClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memoryServiceGatewayClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "servicegateways"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memoryServiceGatewayClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memoryServiceGatewayClient) StoredServiceGateway() *corev1beta1.ServiceGateway {
	return c.stored.DeepCopyObject().(*corev1beta1.ServiceGateway)
}
