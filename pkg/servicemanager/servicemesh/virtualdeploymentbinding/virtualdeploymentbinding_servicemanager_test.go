/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	mocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	"github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	"github.com/oracle/oci-service-operator/test/servicemesh/virtualdeployment"
	"github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

func TestCreateOrUpdate(t *testing.T) {
	vdIdNew := "vd-id-new"
	type fields struct {
		GetVirtualDeployment func(ctx context.Context, id *api.OCID) (*sdk.VirtualDeployment, error)
		GetVirtualService    func(ctx context.Context, id *api.OCID) (*sdk.VirtualService, error)
	}
	tests := []struct {
		name        string
		vdb         *servicemeshapi.VirtualDeploymentBinding
		pod         *corev1.Pod
		podKey      types.NamespacedName
		service     *corev1.Service
		serviceKey  types.NamespacedName
		vd          *servicemeshapi.VirtualDeployment
		vdId        string
		activateVd  bool
		fields      fields
		wantStatus  servicemeshapi.ServiceMeshStatus
		wantErr     error
		response    *servicemanager.OSOKResponse
		serviceMock bool
	}{
		{
			name:       "k8s reconcile successfully",
			vdb:        virtualdeploymentbinding.NewVdbWithVdRef("vdb", "namespace", "vd", "service"),
			vd:         virtualdeployment.NewTestEnvVd("vd", "namespace"),
			activateVd: true,
			pod: k8s.NewPodWithLabels("pod", "namespace", map[string]string{
				commons.ProxyInjectionLabel: commons.Enabled,
				"app":                       "service",
			}),
			podKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "pod",
			},
			service: k8s.NewKubernetesService("service", "namespace"),
			serviceKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "service",
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId:   "vd",
				VirtualDeploymentName: "namespace/vd",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActiveVDB),
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name:       "k8s reconcile successfully when Service Namespace is not specified",
			vdb:        virtualdeploymentbinding.NewVdbWithVdRefWithServiceNamespace("vdb-0", "namespace", "vd-0", "service-0", ""),
			vd:         virtualdeployment.NewTestEnvVd("vd-0", "namespace"),
			activateVd: true,
			pod: k8s.NewPodWithLabels("pod-0", "namespace", map[string]string{
				commons.ProxyInjectionLabel: commons.Enabled,
				"app":                       "service-0",
			}),
			podKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "pod-0",
			},
			service: k8s.NewKubernetesService("service-0", "namespace"),
			serviceKey: types.NamespacedName{
				Name:      "service-0",
				Namespace: "namespace",
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId:   "vd",
				VirtualDeploymentName: "namespace/vd-0",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActiveVDB),
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "k8s fail to resolve virtual development",
			vdb:  virtualdeploymentbinding.NewVdbWithVdRef("vdb-1", "namespace", "vd-1", "service"),
			wantStatus: servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionFalse,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "virtualdeployments.servicemesh.oci.oracle.com \"vd-1\" not found",
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr: errors.New("virtualdeployments.servicemesh.oci.oracle.com \"vd-1\" not found"),
		},
		{
			name: "k8s fail to validate virtual development",
			vdb:  virtualdeploymentbinding.NewVdbWithVdRef("vdb-2", "namespace", "vd-1", "service"),
			vd:   virtualdeployment.NewTestEnvVd("vd-1", "namespace"),
			wantStatus: servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "virtual deployment is not active yet",
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
		{
			name:       "k8s fail to resolve service",
			vdb:        virtualdeploymentbinding.NewVdbWithVdRef("vdb-3", "namespace", "vd-3", "service-1"),
			vd:         virtualdeployment.NewTestEnvVd("vd-3", "namespace"),
			activateVd: true,
			wantStatus: servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "service does not exists",
							ObservedGeneration: 1,
						},
					},
				},
			},
			serviceMock: true,
			wantErr:     nil,
			response:    &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
		{
			name: "cp fail to resolve virtual development",
			vdb:  virtualdeploymentbinding.NewVdbWithVdRef("vdb-4", "namespace", "vd", "service"),
			fields: fields{
				GetVirtualDeployment: func(ctx context.Context, id *api.OCID) (*sdk.VirtualDeployment, error) {
					return nil, meshErrors.NewServiceError(404, "NotFound", "Failed to get VirtualDeployment from ControlPlane", "12-35-89")
				},
			},
			vdId: "vd-id",
			wantStatus: servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionFalse,
							Reason:             "NotFound",
							Message:            "Failed to get VirtualDeployment from ControlPlane (opc-request-id: 12-35-89 )",
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
		{
			name: "cp fail to validate virtual development",
			vdb:  virtualdeploymentbinding.NewVdbWithVdRef("vdb-5", "namespace", "vd-1", "service"),
			fields: fields{
				GetVirtualDeployment: func(ctx context.Context, id *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						LifecycleState: sdk.VirtualDeploymentLifecycleStateCreating,
					}, nil
				},
			},
			vdId: "vd-id",
			wantStatus: servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "virtual deployment is not active yet",
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
		{
			name: "cp fail to resolve service",
			vdb:  virtualdeploymentbinding.NewVdbWithVdRef("vdb-6", "namespace", "vd", "service-1"),
			vdId: "vd-id",
			fields: fields{
				GetVirtualDeployment: func(ctx context.Context, id *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						LifecycleState: sdk.VirtualDeploymentLifecycleStateActive,
					}, nil
				},
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "service does not exists",
							ObservedGeneration: 1,
						},
					},
				},
			},
			serviceMock: true,
			wantErr:     nil,
			response:    &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
		{
			name: "cp reconcile successfully",
			vdb:  virtualdeploymentbinding.NewVdbWithVdRef("vdb-7", "namespace", "vd", "service-7"),
			vdId: "vd-id",
			fields: fields{
				GetVirtualDeployment: func(ctx context.Context, id *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						Id:               &vdIdNew,
						Name:             conversions.String("vd-name"),
						VirtualServiceId: conversions.String("vs-id"),
						LifecycleState:   sdk.VirtualDeploymentLifecycleStateActive,
					}, nil
				},
				GetVirtualService: func(ctx context.Context, id *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						Id:             conversions.String("vs-id"),
						Name:           conversions.String("vs-name"),
						MeshId:         conversions.String("mesh-id"),
						LifecycleState: sdk.VirtualServiceLifecycleStateActive,
					}, nil
				},
			},
			pod: k8s.NewPodWithLabels("pod-7", "namespace", map[string]string{
				commons.ProxyInjectionLabel: commons.Enabled,
				"app":                       "service-1",
			}),
			podKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "pod-7",
			},
			service: k8s.NewKubernetesService("service-7", "namespace"),
			serviceKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "service-7",
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId:   "vd-id-new",
				VirtualDeploymentName: "vd-name",
				VirtualServiceId:      "vs-id",
				VirtualServiceName:    "vs-name",
				MeshId:                "mesh-id",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActiveVDB),
							ObservedGeneration: 1,
						},
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
	}
	f := framework.NewTestEnvClientFramework(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			ctx := context.Background()
			controller := gomock.NewController(t)
			meshClient := mocks.NewMockServiceMeshClient(controller)

			ns := k8s.NewNamespace("namespace", map[string]string{commons.ProxyInjectionLabel: commons.Enabled})
			nsKey := types.NamespacedName{
				Name: "namespace",
			}
			err = k8s.CreateNamespace(ctx, f.K8sClient, nsKey, ns)
			assert.NoError(t, err)

			if len(tt.vdId) > 0 {
				tt.vdb.Spec.VirtualDeployment.Id = api.OCID(tt.vdId)
			}

			err = f.K8sClient.Create(ctx, tt.vdb)
			assert.NoError(t, err)

			if tt.vd != nil {
				err = f.K8sClient.Create(ctx, tt.vd)
				assert.NoError(t, err)
			}

			if tt.activateVd {
				err = f.K8sClient.Status().Update(ctx, tt.vd)
				assert.NoError(t, err)
			}

			if tt.pod != nil {
				err = k8s.CreatePod(ctx, f.K8sClient, tt.podKey, tt.pod)
				assert.NoError(t, err)
			}

			testFramework := framework.NewFakeClientFramework(t)
			if tt.service != nil {
				err = k8s.CreateKubernetesService(ctx, f.K8sClient, tt.serviceKey, tt.service)
				assert.NoError(t, err)
				testFramework.Cache.EXPECT().GetServiceByKey(tt.service.Namespace+"/"+tt.service.Name).Return(tt.service, nil)
				assert.NoError(t, err)
			}

			if tt.serviceMock {
				testFramework.Cache.EXPECT().GetServiceByKey(gomock.Any()).Return(nil, errors.New("service does not exists"))
			}

			if tt.fields.GetVirtualDeployment != nil {
				meshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVirtualDeployment)
			}

			if tt.fields.GetVirtualService != nil {
				meshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVirtualService)
			}

			ref := references.NewDefaultResolver(f.K8sClient, meshClient, f.Log, testFramework.Cache, f.K8sClient)
			m := &VirtualDeploymentBindingServiceManager{
				client:            f.K8sClient,
				clientSet:         f.K8sClientset,
				referenceResolver: ref,
				log:               f.Log,
				meshClient:        meshClient,
			}

			response, err := m.CreateOrUpdate(ctx, tt.vdb, ctrl.Request{})
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
			if tt.response != nil {
				assert.True(t, cmp.Equal(tt.response.ShouldRequeue, response.ShouldRequeue))
			}
			curVdb := &servicemeshapi.VirtualDeploymentBinding{}
			err = f.K8sClient.Get(ctx, types.NamespacedName{Name: tt.vdb.Name, Namespace: "namespace"}, curVdb)
			assert.NoError(t, err)
			opts := equality.IgnoreFakeClientPopulatedFields()
			assert.True(t, cmp.Equal(tt.wantStatus, curVdb.Status, opts), "diff", cmp.Diff(tt.wantStatus, curVdb.Status, opts))
		})
	}
	f.Cleanup()
}

func TestResolveVdK8s(t *testing.T) {
	tests := []struct {
		name    string
		vdb     *servicemeshapi.VirtualDeploymentBinding
		vd      *servicemeshapi.VirtualDeployment
		wantErr error
	}{
		{
			name: "virtualDeployment without namespace",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{
					VirtualDeployment: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name: "vd",
						},
					},
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vd",
					Namespace: "namespace",
				},
			},
			wantErr: nil,
		},
		{
			name: "virtualDeployment with namespace",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{
					VirtualDeployment: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name:      "vd",
							Namespace: "namespace",
						},
					},
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vd",
					Namespace: "namespace",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			ref := references.NewDefaultResolver(f.K8sClient, nil, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("testFramework").WithName("ServiceMesh")}, nil, f.K8sClient)
			h := &VirtualDeploymentBindingServiceManager{
				meshClient:        f.MeshClient,
				referenceResolver: ref,
			}
			err := f.K8sClient.Create(ctx, tt.vd)
			assert.NoError(t, err)
			err = f.K8sClient.Create(ctx, tt.vdb)
			assert.NoError(t, err)
			vd, err := h.resolveVirtualDeploymentK8s(ctx, tt.vdb)
			assert.NoError(t, err)
			opts := equality.IgnoreFakeClientPopulatedFields()
			assert.True(t, cmp.Equal(tt.vd, vd, opts))
		})
	}

}

func TestValidateVdK8s(t *testing.T) {
	tests := []struct {
		name    string
		vd      *servicemeshapi.VirtualDeployment
		wantErr error
	}{
		{
			name: "virtualDeployment active",
			vd: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "virtualDeployment inactive",
			vd: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionFalse,
							},
						},
					},
				},
			},
			wantErr: errors.New("virtual deployment is not active yet"),
		},
		{
			name: "virtualDeployment active without virtual Deployment Id",
			vd: &servicemeshapi.VirtualDeployment{
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: errors.New("virtualDeployment active, and virtualDeploymentId is not set"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &VirtualDeploymentBindingServiceManager{}
			err := h.validateVirtualDeploymentK8s(tt.vd)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVdCp(t *testing.T) {
	tests := []struct {
		name    string
		vd      *sdk.VirtualDeployment
		wantErr error
	}{
		{
			name: "virtualDeployment active",
			vd: &sdk.VirtualDeployment{
				LifecycleState: sdk.VirtualDeploymentLifecycleStateActive,
			},
			wantErr: nil,
		},
		{
			name: "virtualDeployment inactive",
			vd: &sdk.VirtualDeployment{
				LifecycleState: sdk.VirtualDeploymentLifecycleStateCreating,
			},
			wantErr: errors.New("virtual deployment is not active yet"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &VirtualDeploymentBindingServiceManager{}
			err := h.validateVirtualDeploymentCp(tt.vd)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	tests := []struct {
		name    string
		service *corev1.Service
		wantErr error
	}{
		{
			name:    "virtualDeployment active",
			service: k8s.NewKubernetesService("service", "namespace"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &VirtualDeploymentBindingServiceManager{}
			err := h.validateService(tt.service)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvictPodsForVDB(t *testing.T) {
	tests := []struct {
		name          string
		vdb           *servicemeshapi.VirtualDeploymentBinding
		ns            *corev1.Namespace
		nsKey         types.NamespacedName
		pod           *corev1.Pod
		podKey        types.NamespacedName
		service       *corev1.Service
		serviceKey    types.NamespacedName
		testEnvClient bool
		wantErr       error
		response      *servicemanager.OSOKResponse
	}{
		{
			name: "namespace not exists",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{},
			},
		},
		{
			name: "no pod for VDB",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{},
			},
			ns: k8s.NewNamespace("namespace", map[string]string{}),
			nsKey: types.NamespacedName{
				Name: "namespace",
			},
		},
		{
			name: "pod upgrade enabled but fail to evict pod",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{
					Target: servicemeshapi.Target{
						Service: servicemeshapi.Service{
							ServiceRef: servicemeshapi.ResourceRef{
								Name: "service",
							},
						},
					},
				},
			},
			ns: k8s.NewNamespace("namespace", map[string]string{commons.ProxyInjectionLabel: commons.Enabled}),
			nsKey: types.NamespacedName{
				Name: "namespace",
			},
			pod: k8s.NewPodWithLabels("pod", "namespace", map[string]string{
				commons.ProxyInjectionLabel: commons.Enabled,
				"app":                       "service",
			}),
			podKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "pod",
			},
			service: k8s.NewKubernetesService("service", "namespace"),
			serviceKey: types.NamespacedName{
				Namespace: "namespace",
				Name:      "service",
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: time.Minute},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)

			m := &VirtualDeploymentBindingServiceManager{
				client:            f.K8sClient,
				clientSet:         f.K8sClientset,
				referenceResolver: f.Resolver,
				log:               f.Log}

			if tt.ns != nil {
				err := k8s.CreateNamespace(ctx, f.K8sClient, tt.nsKey, tt.ns)
				assert.NoError(t, err)
			}

			if tt.pod != nil {
				err := k8s.CreatePod(ctx, f.K8sClient, tt.podKey, tt.pod)
				assert.NoError(t, err)
			}

			if tt.service != nil {
				err := k8s.CreateKubernetesService(ctx, f.K8sClient, tt.serviceKey, tt.service)
				assert.NoError(t, err)
			}

			response, err := m.evictPodsForVirtualDeploymentBinding(ctx, tt.vdb, tt.service)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				if tt.response != nil {
					assert.True(t, cmp.Equal(tt.response.ShouldRequeue, response.ShouldRequeue))
					assert.True(t, cmp.Equal(tt.response.IsSuccessful, response.IsSuccessful))
					assert.True(t, cmp.Equal(tt.response.RequeueDuration, response.RequeueDuration))
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestUpdateCRCp(t *testing.T) {
	vdId := "vd-id"
	type fields struct {
		GetResourceRefById func(ctx context.Context, virtualServiceId *api.OCID) (*commons.ResourceRef, error)
	}
	tests := []struct {
		name       string
		vdb        *servicemeshapi.VirtualDeploymentBinding
		vd         *sdk.VirtualDeployment
		fields     fields
		wantStatus servicemeshapi.ServiceMeshStatus
		wantErr    error
	}{
		{
			name: "update virtual deployment id",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			vd: &sdk.VirtualDeployment{
				Id:               &vdId,
				Name:             conversions.String("vd-name"),
				VirtualServiceId: conversions.String("vs-id"),
				LifecycleState:   sdk.VirtualDeploymentLifecycleStateCreating,
			},
			fields: fields{
				GetResourceRefById: func(ctx context.Context, virtualServiceId *api.OCID) (*commons.ResourceRef, error) {
					return &commons.ResourceRef{
						Id:     "vs-id",
						Name:   "vs-name",
						MeshId: "mesh-id",
					}, nil
				},
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId:   "vd-id",
				VirtualDeploymentName: "vd-name",
				VirtualServiceId:      "vs-id",
				VirtualServiceName:    "vs-name",
				MeshId:                "mesh-id",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionUnknown,
							Reason:  string(commons.DependenciesNotResolved),
							Message: string(commons.ResourceCreatingVDB),
						},
					},
				},
			},
			wantErr: errors.New(commons.UnknownStatus),
		},
		{
			name: "update virtual deployment condition true",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  "",
								Message: "",
							},
						},
					},
				},
			},
			vd: &sdk.VirtualDeployment{
				Id:             &vdId,
				LifecycleState: sdk.VirtualDeploymentLifecycleStateActive,
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId: "vd-id",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionTrue,
							Reason:  string(commons.Successful),
							Message: string(commons.ResourceActiveVDB),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "update virtual deployment condition false",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  "",
								Message: "",
							},
						},
					},
				},
			},
			vd: &sdk.VirtualDeployment{
				Id:             &vdId,
				LifecycleState: sdk.VirtualDeploymentLifecycleStateDeleted,
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId: "vd-id",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionFalse,
							Reason:  string(commons.DependenciesNotResolved),
							Message: string(commons.ResourceDeletedVDB),
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)

			m := &VirtualDeploymentBindingServiceManager{
				client:            f.K8sClient,
				clientSet:         f.K8sClientset,
				referenceResolver: f.Resolver,
				log:               f.Log}

			if tt.fields.GetResourceRefById != nil {
				f.Resolver.EXPECT().ResolveVirtualServiceRefById(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetResourceRefById).AnyTimes()
			}

			err := f.K8sClient.Create(ctx, tt.vdb)
			assert.NoError(t, err)
			err = m.updateCRCp(ctx, tt.vdb, tt.vd)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			curVdb := &servicemeshapi.VirtualDeploymentBinding{}
			err = f.K8sClient.Get(ctx, types.NamespacedName{Name: "vdb", Namespace: "namespace"}, curVdb)
			assert.NoError(t, err)
			opts := equality.IgnoreFakeClientPopulatedFields()
			assert.True(t, cmp.Equal(tt.wantStatus, curVdb.Status, opts), "diff", cmp.Diff(tt.wantStatus, curVdb.Status, opts))
		})
	}
}

func TestUpdateCRK8s(t *testing.T) {
	type fields struct {
		GetResourceRefById func(ctx context.Context, virtualServiceId *api.OCID) (*commons.ResourceRef, error)
	}
	tests := []struct {
		name       string
		vdb        *servicemeshapi.VirtualDeploymentBinding
		vd         *servicemeshapi.VirtualDeployment
		fields     fields
		wantStatus servicemeshapi.ServiceMeshStatus
		wantErr    error
	}{
		{
			name: "update virtual deployment id",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vd-name",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
					VirtualServiceId:    "vs-id",
					VirtualServiceName:  "vs-name",
					MeshId:              "mesh-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  "",
								Message: "",
							},
						},
					},
				},
			},
			fields: fields{
				GetResourceRefById: func(ctx context.Context, virtualServiceId *api.OCID) (*commons.ResourceRef, error) {
					return &commons.ResourceRef{
						Id:     "vs-id",
						Name:   "vs-name",
						MeshId: "mesh-id",
					}, nil
				},
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId:   "vd-id",
				VirtualDeploymentName: "namespace/vd-name",
				VirtualServiceId:      "vs-id",
				VirtualServiceName:    "vs-name",
				MeshId:                "mesh-id",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionTrue,
							Reason:  string(commons.Successful),
							Message: string(commons.ResourceActiveVDB),
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "update virtual deployment condition true",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vdb",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  "",
								Message: "",
							},
						},
					},
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vd",
					Namespace: "namespace",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  "",
								Message: "",
							},
						},
					},
				},
			},
			wantStatus: servicemeshapi.ServiceMeshStatus{
				VirtualDeploymentId: "vd-id",
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionTrue,
							Reason:  string(commons.Successful),
							Message: string(commons.ResourceActiveVDB),
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)

			m := &VirtualDeploymentBindingServiceManager{
				client:            f.K8sClient,
				clientSet:         f.K8sClientset,
				referenceResolver: f.Resolver,
				log:               f.Log}

			if tt.fields.GetResourceRefById != nil {
				f.Resolver.EXPECT().ResolveVirtualServiceRefById(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetResourceRefById).AnyTimes()
			}

			err := f.K8sClient.Create(ctx, tt.vdb)
			assert.NoError(t, err)
			err = m.updateCRK8s(ctx, tt.vdb, tt.vd)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			curVdb := &servicemeshapi.VirtualDeploymentBinding{}
			err = f.K8sClient.Get(ctx, types.NamespacedName{Name: "vdb", Namespace: "namespace"}, curVdb)
			assert.NoError(t, err)
			opts := equality.IgnoreFakeClientPopulatedFields()
			assert.True(t, cmp.Equal(tt.wantStatus, curVdb.Status, opts), "diff", cmp.Diff(tt.wantStatus, curVdb.Status, opts))
		})
	}
}
