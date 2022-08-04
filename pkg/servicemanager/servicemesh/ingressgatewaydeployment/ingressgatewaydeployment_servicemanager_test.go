/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewaydeployment

import (
	"context"
	"k8s.io/apimachinery/pkg/api/resource"
	"sort"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
)

var globalConfigMap = NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName,
	map[string]string{
		commons.ProxyLabelInMeshConfigMap:  "iad.ocir.io/idotidhmwg5o/awille/envoy:1.17.0.1",
		commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443",
	})

const (
	igdKind       = "IngressGatewayDeployment"
	igdAPIVersion = "oci.oracle.com/servicemeshapi"
	igdName       = "product-igd"
)

var (
	ctx               = context.Background()
	k8sClient         client.Client
	k8sClientSet      kubernetes.Interface
	igdServiceManager *IngressGatewayDeploymentServiceManager
	resolver          *meshMocks.MockResolver
	mockCaches        *meshMocks.MockCacheMapClient
	testNamespace     *corev1.Namespace
)

func BeforeSuite(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockCaches = meshMocks.NewMockCacheMapClient(mockCtrl)
	k8sSchema := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(k8sSchema)
	_ = servicemeshapi.AddToScheme(k8sSchema)
	k8sClient = testclient.NewClientBuilder().WithScheme(k8sSchema).Build()
	k8sClientSet = fake.NewSimpleClientset()
	controller := gomock.NewController(t)
	resolver = meshMocks.NewMockResolver(controller)
	igdServiceManager = &IngressGatewayDeploymentServiceManager{
		client:            k8sClient,
		log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGD")},
		clientSet:         k8sClientSet,
		referenceResolver: resolver,
		Caches:            mockCaches,
		namespace:         commons.OsokNamespace,
	}
}

func BeforeTest(t *testing.T, suffix string) {
	testNamespace = NewNamespace("product"+suffix, map[string]string{})
	err := CreateNamespace(ctx, k8sClient, types.NamespacedName{Name: testNamespace.Name}, testNamespace)
	if err != nil {
		t.Fatal("Failed to create product namespace", err)
	}

}
func AfterTest(t *testing.T) {
	err := DeleteNamespace(context.Background(), k8sClient, testNamespace)
	if err != nil {
		t.Fatal("Failed to delete product namespace", err)
	}
}
func TestReconcile(t *testing.T) {

	type args struct {
		hasService                              bool
		serviceAnnotations                      map[string]string
		serviceLabels                           map[string]string
		servicePortCount                        int
		differentProtocols                      bool
		totalPortCount                          int
		useConfigMap                            bool
		times                                   int
		updatePort                              bool
		updateScalar                            bool
		secrets                                 []string
		updateSecrets                           []string
		ResolveIngressGatewayIdAndNameAndMeshId func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
	}
	type expected struct {
		hasService          bool
		hasDeployment       bool
		hasPodAutoscalar    bool
		hasError            bool
		deploymentPortCount int
		servicePortCount    int
		wantErr             error
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Create Deployment, HPA and Service",
			args: args{
				hasService: true,
				serviceAnnotations: map[string]string{
					"oci.oraclecloud.com/load-balancer-type":                "lb",
					"service.beta.kubernetes.io/oci-load-balancer-internal": "true",
				},
				serviceLabels: map[string]string{
					"some-key": "some-value",
				},
				servicePortCount:   3,
				times:              3,
				differentProtocols: false,
				totalPortCount:     3,
				useConfigMap:       true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
				servicePortCount:    3,
				wantErr:             nil,
			},
		},
		{
			name: "Different Protocol",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				differentProtocols: true,
				totalPortCount:     3,
				times:              4,
				useConfigMap:       true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          false,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            true,
				deploymentPortCount: 4,
			},
		},

		{
			name: "No Service",
			args: args{
				hasService:         false,
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     3,
				times:              1,
				useConfigMap:       true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          false,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
			},
		},

		{
			name: "Some ports not exposed in service",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     5,
				times:              1,
				useConfigMap:       true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 6,
				servicePortCount:    3,
			},
		},
		{
			name: "Some ports not exposed in service",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     5,
				times:              1,
				useConfigMap:       true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 6,
				servicePortCount:    3,
			},
		},
		{
			name: "Update Port in Service and Deployment",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				times:              3,
				differentProtocols: false,
				totalPortCount:     3,
				useConfigMap:       true,
				updatePort:         true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
				servicePortCount:    3,
				wantErr:             nil,
			},
		},
		{
			name: "Update HPA",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				times:              3,
				differentProtocols: false,
				totalPortCount:     3,
				useConfigMap:       true,
				updateScalar:       true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
				servicePortCount:    3,
				wantErr:             nil,
			},
		},
		{
			name: "Update HPA and Port",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				times:              3,
				differentProtocols: false,
				totalPortCount:     3,
				useConfigMap:       true,
				updateScalar:       true,
				updatePort:         true,
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
				servicePortCount:    3,
				wantErr:             nil,
			},
		},
		{
			name: "Create Deployment with kube secrets",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				times:              3,
				differentProtocols: false,
				totalPortCount:     3,
				useConfigMap:       true,
				secrets:            []string{"bookinfo-tls-secret", "bookinfo-cabundle-secret"},
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
				servicePortCount:    3,
				wantErr:             nil,
			},
		},
		{
			name: "Update Deployment with kube secrets",
			args: args{
				hasService:         true,
				servicePortCount:   3,
				times:              3,
				differentProtocols: false,
				totalPortCount:     3,
				useConfigMap:       true,
				updateSecrets:      []string{"bookinfo-tls-secret", "bookinfo-cabundle-secret"},
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, igRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ResourceRef := commons.ResourceRef{
						Id:     api.OCID("ingress-id"),
						Name:   servicemeshapi.Name("ingress-name"),
						MeshId: api.OCID("my-mesh-id"),
					}
					return &ResourceRef, nil
				},
			},
			want: expected{
				hasService:          true,
				hasDeployment:       true,
				hasPodAutoscalar:    true,
				hasError:            false,
				deploymentPortCount: 4,
				servicePortCount:    3,
				wantErr:             nil,
			},
		},
	}
	BeforeSuite(t)
	for i, tt := range tests {
		BeforeTest(t, "reconcile"+strconv.Itoa(i))
		if tt.args.times == 0 {
			tt.args.times = 1
		}
		if tt.args.useConfigMap {
			mockCaches.EXPECT().GetConfigMapByKey(commons.GlobalConfigMap).Return(globalConfigMap, nil).Times(tt.args.times)
		}
		resolver.EXPECT().ResolveIngressGatewayIdAndNameAndMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.ResolveIngressGatewayIdAndNameAndMeshId).AnyTimes()

		igd := servicemeshapi.IngressGatewayDeployment{}
		igd.Kind = igdKind
		igd.APIVersion = igdAPIVersion
		igd.Name = igdName
		igd.Namespace = testNamespace.Name

		igd.Spec.IngressGateway = servicemeshapi.RefOrId{
			ResourceRef: &servicemeshapi.ResourceRef{
				Namespace: "my-namespace",
				Name:      "my-mesh",
			},
		}

		if tt.args.hasService {
			igd.Spec.Service = &servicemeshapi.IngressGatewayService{
				Type:        corev1.ServiceTypeLoadBalancer,
				Annotations: tt.args.serviceAnnotations,
				Labels:      tt.args.serviceLabels,
			}
		}

		for _, secret := range tt.args.secrets {
			igd.Spec.Secrets = append(igd.Spec.Secrets, servicemeshapi.SecretReference{
				SecretName: secret,
			})
		}

		ports := make([]servicemeshapi.GatewayListener, 0)
		for i := 0; i < tt.args.totalPortCount; i++ {
			pNumber := int32(i)
			ports = append(ports, servicemeshapi.GatewayListener{
				Protocol: corev1.ProtocolTCP,
				Port:     &pNumber,
			})
		}

		for i := 0; i < tt.args.servicePortCount; i++ {
			sport := int32(i)
			ports[i].ServicePort = &sport
		}
		if tt.args.differentProtocols {
			ports[0].Protocol = corev1.ProtocolUDP
		}
		igd.Spec.Ports = ports
		igd.Spec.Deployment.Autoscaling = &servicemeshapi.Autoscaling{MinPods: 2, MaxPods: 3}

		k8sClient.Create(ctx, &igd)
		for i := 0; i < tt.args.times; i++ {
			_, err := igdServiceManager.CreateOrUpdate(ctx, &igd, ctrl.Request{})
			assert.Equal(t, err != nil, tt.want.hasError)

		}

		service := corev1.Service{}
		deployment := appsv1.Deployment{}
		hpa := autoscalingv1.HorizontalPodAutoscaler{}
		serr := k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeService)}, &service)
		assert.Equal(t, tt.want.hasService, serr == nil)
		if tt.want.hasService {
			assert.Equal(t, len(service.Spec.Ports), tt.want.servicePortCount)
			assert.Equal(t, igd.Name+string(commons.NativeService), service.Name)
			assert.Equal(t, igd.Namespace, service.Namespace)
			assert.Equal(t, igd.Spec.Service.Labels, service.Labels)
			assert.Equal(t, igd.Spec.Service.Annotations, service.Annotations)
		}

		derr := k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeDeployment)}, &deployment)
		assert.Equal(t, tt.want.hasDeployment, derr == nil)
		if tt.want.hasDeployment {
			assert.Equal(t, len(deployment.Spec.Template.Spec.Containers[0].Ports), tt.want.deploymentPortCount)
			assert.Equal(t, igd.Name+string(commons.NativeDeployment), deployment.Name)
			assert.Equal(t, igd.Namespace, deployment.Namespace)
		}

		validateSecrets(t, tt.args.secrets, &deployment.Spec)

		herr := k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeHorizontalPodAutoScalar)}, &hpa)
		assert.Equal(t, tt.want.hasPodAutoscalar, herr == nil)
		if tt.want.hasPodAutoscalar {
			assert.Equal(t, igd.Spec.Deployment.Autoscaling.MinPods, *hpa.Spec.MinReplicas)
			assert.Equal(t, igd.Spec.Deployment.Autoscaling.MaxPods, hpa.Spec.MaxReplicas)
			assert.Equal(t, igd.Name+string(commons.NativeHorizontalPodAutoScalar), hpa.Name)
			assert.Equal(t, igd.Namespace, hpa.Namespace)

		}

		if tt.args.updatePort {
			if tt.args.useConfigMap {
				mockCaches.EXPECT().GetConfigMapByKey(commons.GlobalConfigMap).Return(globalConfigMap, nil)
			}
			resolver.EXPECT().ResolveIngressGatewayIdAndNameAndMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.ResolveIngressGatewayIdAndNameAndMeshId).AnyTimes()

			for i := 0; i < len(ports); i++ {
				newPort := int32(1000) + int32(i)
				ports[i].Port = &newPort
			}
			igd.Spec.Ports = ports
			_, err := igdServiceManager.CreateOrUpdate(ctx, &igd, ctrl.Request{})
			assert.Nil(t, err)
			newService := corev1.Service{}

			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeService)}, &newService)
			assert.Nil(t, err)
			servicePorts := newService.Spec.Ports

			portsSlice := []int{}
			for i := 0; i < len(servicePorts); i++ {
				portsSlice = append(portsSlice, int(servicePorts[i].TargetPort.IntVal))
			}
			sort.Ints(portsSlice)
			for i := 0; i < len(portsSlice); i++ {
				assert.Equal(t, 1000+i, portsSlice[i])
			}

			newDeployment := appsv1.Deployment{}

			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeDeployment)}, &newDeployment)
			assert.Nil(t, err)
			deploymentPorts := newDeployment.Spec.Template.Spec.Containers[0].Ports

			deploymentPortsSlice := []int{}
			for i := 0; i < len(deploymentPorts); i++ {
				deploymentPortsSlice = append(deploymentPortsSlice, int(deploymentPorts[i].ContainerPort))
			}
			sort.Ints(deploymentPortsSlice)
			for i := 0; i < len(deploymentPortsSlice)-1; i++ {
				assert.Equal(t, 1000+i, deploymentPortsSlice[i])
			}
			assert.Equal(t, 15006, deploymentPortsSlice[len(deploymentPortsSlice)-1])
		}

		if tt.args.updateScalar {
			if tt.args.useConfigMap {
				mockCaches.EXPECT().GetConfigMapByKey(commons.GlobalConfigMap).Return(globalConfigMap, nil)
			}
			resolver.EXPECT().ResolveIngressGatewayIdAndNameAndMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.ResolveIngressGatewayIdAndNameAndMeshId).AnyTimes()
			igd.Spec.Deployment.Autoscaling.MaxPods = 100
			igd.Spec.Deployment.Autoscaling.MinPods = 100

			_, err := igdServiceManager.CreateOrUpdate(ctx, &igd, ctrl.Request{})
			assert.Nil(t, err)
			newHpa := autoscalingv1.HorizontalPodAutoscaler{}

			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeHorizontalPodAutoScalar)}, &newHpa)
			assert.Nil(t, err)
			assert.Equal(t, *newHpa.Spec.MinReplicas, int32(100))
			assert.Equal(t, newHpa.Spec.MaxReplicas, int32(100))

		}

		if len(tt.args.updateSecrets) != 0 {
			if tt.args.useConfigMap {
				mockCaches.EXPECT().GetConfigMapByKey(commons.GlobalConfigMap).Return(globalConfigMap, nil)
			}
			resolver.EXPECT().ResolveIngressGatewayIdAndNameAndMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.args.ResolveIngressGatewayIdAndNameAndMeshId).AnyTimes()
			igd.Spec.Secrets = []servicemeshapi.SecretReference{}
			for _, secret := range tt.args.updateSecrets {
				igd.Spec.Secrets = append(igd.Spec.Secrets, servicemeshapi.SecretReference{SecretName: secret})
			}

			_, err := igdServiceManager.CreateOrUpdate(ctx, &igd, ctrl.Request{})
			assert.Nil(t, err)

			newDeployment := appsv1.Deployment{}

			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: igd.Namespace, Name: igd.Name + string(commons.NativeDeployment)}, &newDeployment)
			assert.Nil(t, err)
			validateSecrets(t, tt.args.updateSecrets, &newDeployment.Spec)
		}
	}

	AfterTest(t)
}

func validateSecrets(t *testing.T, secrets []string, deployment *appsv1.DeploymentSpec) {
	for _, secret := range secrets {
		volumePresent := false
		for _, volume := range deployment.Template.Spec.Volumes {
			if volume.Name == secret && volume.Secret.SecretName == secret {
				volumePresent = true
				break
			}
		}
		assert.Truef(t, volumePresent, "volume missing for secret %s", secret)

		volumeMountPresent := false
		for _, volumeMount := range deployment.Template.Spec.Containers[0].VolumeMounts {
			if volumeMount.Name == secret {
				assert.Equal(t, true, volumeMount.ReadOnly)
				assert.Equal(t, "/etc/oci/secrets/"+secret, volumeMount.MountPath)
				volumeMountPresent = true
				break
			}
		}
		assert.Truef(t, volumeMountPresent, "volumeMount missing for secret %s", secret)
	}
}

func TestCreateService(t *testing.T) {
	type args struct {
		hasIGService       bool
		servicePortCount   int
		differentProtocols bool
		totalPortCount     int
		serviceType        corev1.ServiceType
	}
	type expected struct {
		hasService       bool
		hasError         bool
		servicePortCount int
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Create Service",
			args: args{
				hasIGService:       true,
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     3,
				serviceType:        corev1.ServiceTypeLoadBalancer,
			},
			want: expected{
				hasService:       true,
				hasError:         false,
				servicePortCount: 3,
			},
		},

		{
			name: "Different Protocols",
			args: args{
				hasIGService:       true,
				servicePortCount:   3,
				differentProtocols: true,
				totalPortCount:     3,
				serviceType:        corev1.ServiceTypeLoadBalancer,
			},
			want: expected{
				hasService: false,
				hasError:   true,
			},
		},

		{
			name: "No Service in IGD",
			args: args{
				hasIGService:       false,
				servicePortCount:   1,
				differentProtocols: true,
				totalPortCount:     1,
				serviceType:        corev1.ServiceTypeLoadBalancer,
			},
			want: expected{
				hasService: false,
				hasError:   false,
			},
		},
	}
	BeforeSuite(t)
	for i, tt := range tests {
		BeforeTest(t, "service"+strconv.Itoa(i))

		igd := servicemeshapi.IngressGatewayDeployment{}
		igd.Kind = igdKind
		igd.APIVersion = igdAPIVersion
		igd.Name = igdName
		igd.Namespace = testNamespace.Name

		igd.Spec.IngressGateway = servicemeshapi.RefOrId{
			ResourceRef: &servicemeshapi.ResourceRef{
				Namespace: "my-namespace",
				Name:      "my-mesh",
			},
		}

		if tt.args.hasIGService {
			igd.Spec.Service = &servicemeshapi.IngressGatewayService{
				Type: tt.args.serviceType,
			}

		}
		ports := make([]servicemeshapi.GatewayListener, 0)
		for i := 0; i < tt.args.totalPortCount; i++ {
			pNUmber := int32(i)
			ports = append(ports, servicemeshapi.GatewayListener{
				Protocol: corev1.ProtocolTCP,
				Port:     &pNUmber,
			})
		}

		for i := 0; i < tt.args.servicePortCount; i++ {
			sport := int32(i)
			ports[i].ServicePort = &sport
		}
		if tt.args.differentProtocols {
			ports[0].Protocol = corev1.ProtocolUDP
		}
		igd.Spec.Ports = ports

		var service []corev1.ServicePort
		var err error
		m := &igdServiceManager
		service, err = (*m).getServicePorts(&igd)
		assert.Equal(t, err != nil, tt.want.hasError)
		assert.Equal(t, service != nil, tt.want.hasService)

		if service == nil {
			continue
		}
		assert.Equal(t, len(service), tt.want.servicePortCount)
		AfterTest(t)
	}
}

func TestCreateDeployment(t *testing.T) {
	type args struct {
		servicePortCount   int
		differentProtocols bool
		totalPortCount     int
		secrets            []string
		annotations        map[string]string
		resources          *corev1.ResourceRequirements
	}
	type expected struct {
		portCount   int
		loggerLevel commons.ProxyLogLevelType
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Create deployment with same port count",
			args: args{
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     5,
			},
			want: expected{
				portCount:   6,
				loggerLevel: commons.DefaultProxyLogLevel,
			},
		},
		{
			name: "UnsupportedServiceType with different protocols",
			args: args{
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     3,
			},
			want: expected{
				portCount:   4,
				loggerLevel: commons.DefaultProxyLogLevel,
			},
		},
		{
			name: "Create deployment with secrets",
			args: args{
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     3,
				secrets:            []string{"bookinfo-tls-secret", "bookinfo-cabundle-secret"},
			},
			want: expected{
				portCount:   4,
				loggerLevel: commons.DefaultProxyLogLevel,
			},
		},
		{
			name: "Create deployment with custom requests",
			args: args{
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     3,
				secrets:            []string{"bookinfo-tls-secret", "bookinfo-cabundle-secret"},
				resources: &corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    *resource.NewMilliQuantity(500, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(1000, resource.DecimalSI),
					},
				},
			},
			want: expected{
				portCount:   4,
				loggerLevel: commons.DefaultProxyLogLevel,
			},
		},
		{
			name: "Create deployment with annotations supplied",
			args: args{
				servicePortCount:   3,
				differentProtocols: false,
				totalPortCount:     5,
				annotations: map[string]string{
					commons.ProxyLogLevelAnnotation: string(commons.ProxyLogLevelInfo),
				},
			},
			want: expected{
				portCount:   6,
				loggerLevel: commons.ProxyLogLevelInfo,
			},
		},
	}
	BeforeSuite(t)
	for i, tt := range tests {
		BeforeTest(t, "deployment"+strconv.Itoa(i))

		igd := servicemeshapi.IngressGatewayDeployment{}
		igd.Kind = igdKind
		igd.APIVersion = igdAPIVersion
		igd.Name = igdName
		igd.Namespace = testNamespace.Name
		igd.Annotations = tt.args.annotations

		igd.Spec.IngressGateway = servicemeshapi.RefOrId{
			ResourceRef: &servicemeshapi.ResourceRef{
				Namespace: "my-namespace",
				Name:      "my-ingressgateway",
			},
		}

		ports := make([]servicemeshapi.GatewayListener, 0)
		for i := 0; i < tt.args.totalPortCount; i++ {
			pNumber := int32(i)
			ports = append(ports, servicemeshapi.GatewayListener{
				Protocol: corev1.ProtocolTCP,
				Port:     &pNumber,
			})
		}

		for i := 0; i < tt.args.servicePortCount; i++ {
			sport := int32(i)
			ports[i].ServicePort = &sport
		}
		if tt.args.differentProtocols {
			ports[0].Protocol = corev1.ProtocolUDP
		}
		igd.Spec.Ports = ports
		igd.Spec.Deployment.Autoscaling = &servicemeshapi.Autoscaling{MinPods: 2, MaxPods: 3}
		if tt.args.resources != nil {
			igd.Spec.Deployment.Autoscaling.Resources = tt.args.resources
		}
		for _, secret := range tt.args.secrets {
			igd.Spec.Secrets = append(igd.Spec.Secrets, servicemeshapi.SecretReference{SecretName: secret})
		}

		mockCaches.EXPECT().GetConfigMapByKey(commons.GlobalConfigMap).Return(globalConfigMap, nil)

		var deployment *appsv1.DeploymentSpec
		m := &igdServiceManager
		ResourceRef := commons.ResourceRef{
			Id:     api.OCID("ig1"),
			Name:   servicemeshapi.Name("ig-name"),
			MeshId: api.OCID("m1"),
		}
		deployment, _ = (*m).createDeploymentSpec(&igd, ResourceRef)
		assert.Equal(t, len(deployment.Template.Spec.Containers[0].Ports), tt.want.portCount)
		container0 := deployment.Template.Spec.Containers[0]
		envVars := container0.Env
		envMap := make(map[string]corev1.EnvVar)
		resources := container0.Resources
		for i := 0; i < len(envVars); i++ {
			envMap[envVars[i].Name] = envVars[i]
		}
		assert.Equal(t, len(envMap), 8)
		assert.Equal(t, "ig1", envMap[string(commons.DeploymentId)].Value)
		assert.Equal(t, string(tt.want.loggerLevel), envMap[string(commons.ProxyLogLevel)].Value)
		assert.Equal(t, "status.podIP", envMap[string(commons.IPAddress)].ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, "status.podIP", envMap[string(commons.PodIp)].ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, "metadata.uid", envMap[string(commons.PodUId)].ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, "metadata.name", envMap[string(commons.PodName)].ValueFrom.FieldRef.FieldPath)
		assert.Equal(t, "metadata.namespace", envMap[string(commons.PodNamespace)].ValueFrom.FieldRef.FieldPath)
		if tt.args.resources == nil {
			assert.True(t, resources.Requests.Cpu().Equal(resource.MustParse(string(commons.IngressCPURequestSize))))
			assert.True(t, resources.Requests.Memory().Equal(resource.MustParse(string(commons.IngressMemoryRequestSize))))
			assert.True(t, resources.Limits.Cpu().Equal(resource.MustParse(string(commons.IngressCPULimitSize))))
			assert.True(t, resources.Limits.Memory().Equal(resource.MustParse(string(commons.IngressMemoryLimitSize))))
		} else {
			assert.Equal(t, tt.args.resources, &resources)
		}

		validateSecrets(t, tt.args.secrets, deployment)
		AfterTest(t)
	}
}

func TestCreateHPA(t *testing.T) {
	type args struct {
		minReplicas int
		maxReplicas int
	}
	type expected struct {
		hasError bool
		hasHPA   bool
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Create HPA",
			args: args{
				minReplicas: 3,
				maxReplicas: 5,
			},
			want: expected{
				hasHPA: true,
			},
		},
		{
			name: "Min > Max replicas",
			args: args{
				minReplicas: 3,
				maxReplicas: 2,
			},
			want: expected{
				hasError: true,
				hasHPA:   false,
			},
		},

		{
			name: "Min == Max replicas",
			args: args{
				minReplicas: 3,
				maxReplicas: 3,
			},
			want: expected{
				hasError: false,
				hasHPA:   true,
			},
		},
	}
	BeforeSuite(t)
	for i, tt := range tests {
		BeforeTest(t, "hpa"+strconv.Itoa(i))

		igd := servicemeshapi.IngressGatewayDeployment{}
		igd.Kind = igdKind
		igd.APIVersion = igdAPIVersion
		igd.Name = igdName
		igd.Namespace = testNamespace.Name

		igd.Spec.Deployment.Autoscaling = &servicemeshapi.Autoscaling{
			MinPods: int32(tt.args.minReplicas),
			MaxPods: int32(tt.args.maxReplicas),
		}

		m := &igdServiceManager
		var autoscaler *autoscalingv1.HorizontalPodAutoscalerSpec
		var err error
		autoscaler, err = (*m).createPodAutoScalerSpec(&igd, "test")
		assert.Equal(t, tt.want.hasError, err != nil)
		assert.Equal(t, tt.want.hasHPA, autoscaler != nil)

		if autoscaler == nil {
			continue
		}

		assert.Equal(t, *autoscaler.TargetCPUUtilizationPercentage, int32(commons.TargetCPUUtilizationPercentage))
		assert.Equal(t, autoscaler.MaxReplicas, igd.Spec.Deployment.Autoscaling.MaxPods)
		assert.Equal(t, *autoscaler.MinReplicas, igd.Spec.Deployment.Autoscaling.MinPods)
		AfterTest(t)
	}
}
