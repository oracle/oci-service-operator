/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewaydeployment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	ns "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
	"github.com/oracle/oci-service-operator/test/servicemesh/functional"
)

var (
	testEnvFramework functional.TestEnvFramework
	testEnv          *envtest.Environment
	config           *rest.Config
	ctx              context.Context
)

func beforeEach(t *testing.T) *functional.Framework {
	ctx = context.Background()
	testEnvFramework = functional.NewDefaultTestEnvFramework()
	testEnv, config = testEnvFramework.SetupTestEnv()
	framework := testEnvFramework.SetupTestFramework(t, config)
	framework.CreateNamespace(ctx, "test-namespace")
	return framework
}

func afterEach(f *functional.Framework) {
	testEnvFramework.CleanUpTestFramework(f)
	testEnvFramework.CleanUpTestEnv(testEnv)
}

func TestIngressGatewayDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		hasHPA              bool
		deleteHPA           bool
		hasUpdate           bool
		hasService          bool
		serviceAnnotations  map[string]string
		serviceLabels       map[string]string
		deleteService       bool
		servicePortCount    int
		totalPortCount      int
		differentProtocols  bool
		secrets             []string
		hasDeployment       bool
		minReplicas         int
		maxReplicas         int
		hasResourceRequests bool
	}

	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create ingressGatewayDeployment with service",
			args: args{
				hasService: true,
				serviceAnnotations: map[string]string{
					"oci.oraclecloud.com/load-balancer-type":                "lb",
					"service.beta.kubernetes.io/oci-load-balancer-internal": "true",
				},
				serviceLabels: map[string]string{
					"some-key": "some-value",
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
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
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
				MeshId:             "my-mesh-id",
				IngressGatewayId:   "my-ingressgateway-id",
				IngressGatewayName: "my-ingressgateway",
			},
		},
		{
			name: "create ingressGatewayDeployment with secrets",
			args: args{
				secrets: []string{"bookinfo-cabundle-secret"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
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
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
				MeshId:             "my-mesh-id",
				IngressGatewayId:   "my-ingressgateway-id",
				IngressGatewayName: "my-ingressgateway",
			},
		},
		{
			name: "create ingressGatewayDeployment with deployment",
			args: args{
				hasDeployment: true,
				minReplicas:   3,
				maxReplicas:   5,
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
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
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
				MeshId:             "my-mesh-id",
				IngressGatewayId:   "my-ingressgateway-id",
				IngressGatewayName: "my-ingressgateway",
			},
		},
		{
			name: "create ingressGatewayDeployment with min replicas > max replicas",
			args: args{
				hasDeployment: true,
				minReplicas:   5,
				maxReplicas:   3,
			},
			expectedErr: errors.New("admission webhook \"igd-validator.servicemesh.oci.oracle.cloud.com\" denied the request: Failed to create Resource for Kind: IngressGatewayDeployment, Name: my-ingressgatewaydeployment, Namespace: test-namespace, Error: spec.deployment.autoscaling maxPods cannot be less than minPods."),
		},
		{
			name: "create ingressGatewayDeployment with multiple ports and different protocols",
			args: args{
				differentProtocols: true,
			},
			expectedErr: errors.New("admission webhook \"igd-validator.servicemesh.oci.oracle.cloud.com\" denied the request: Failed to create Resource for Kind: IngressGatewayDeployment, Name: my-ingressgatewaydeployment, Namespace: test-namespace, Error: ingressgatewaydeployment.spec cannot have multiple protocols."),
		},
		{
			name: "update ingressGatewayDeployment",
			args: args{
				hasUpdate: true,
			},
		},
		{
			name: "delete HPA after ingressGatewayDeployment creation",
			args: args{
				hasHPA:    true,
				deleteHPA: true,
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
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
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
				MeshId:             "my-mesh-id",
				IngressGatewayId:   "my-ingressgateway-id",
				IngressGatewayName: "my-ingressgateway",
			},
		},
		{
			name: "delete service after ingressGatewayDeployment creation",
			args: args{
				deleteService: true,
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
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
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
				MeshId:             "my-mesh-id",
				IngressGatewayId:   "my-ingressgateway-id",
				IngressGatewayName: "my-ingressgateway",
			},
		},
		{
			name: "create ingressGatewayDeployment with Resource requests",
			args: args{
				hasResourceRequests: true,
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
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
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
				MeshId:             "my-mesh-id",
				IngressGatewayId:   "my-ingressgateway-id",
				IngressGatewayName: "my-ingressgateway",
			},
		},
	}

	for _, tt := range tests {
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
			// Create config map
			framework.CreateNamespace(ctx, commons.OsokNamespace)
			configmap := functional.GetConfigMap()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, configmap))

			// Create the mesh
			mesh := functional.GetApiMesh()
			framework.MeshClient.EXPECT().CreateMesh(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkMesh(sdk.MeshLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, mesh))

			// Create the ingressGateway
			ingressGateway := functional.GetApiIngressGateway()
			framework.MeshClient.EXPECT().CreateIngressGateway(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGateway))

			// Create the ingressGatewayDeployment
			ingressGatewayDeployment := functional.GetApiIngressGatewayDeployment()

			// service, if any
			if tt.args.hasService {
				ingressGatewayDeployment.Spec.Service = &servicemeshapi.IngressGatewayService{
					Type: corev1.ServiceTypeLoadBalancer,
					Annotations: tt.args.serviceAnnotations,
					Labels: tt.args.serviceLabels,
				}
			}

			if tt.args.hasResourceRequests {
				ingressGatewayDeployment.Spec.Deployment.Autoscaling.Resources = &corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPURequestSize)),
						corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryRequestSize)),
					},
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPULimitSize)),
						corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryLimitSize)),
					},
				}

			}

			// secrets, if any
			for _, secret := range tt.args.secrets {
				ingressGatewayDeployment.Spec.Secrets = append(ingressGatewayDeployment.Spec.Secrets, servicemeshapi.SecretReference{
					SecretName: secret,
				})
			}

			namespace := functional.GetSidecarInjectNamespace()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, namespace))

			service := functional.GetService()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, service))

			pod := functional.GetPod()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, pod))

			if tt.expectedErr != nil {
				if tt.args.hasUpdate {
					// Create
					assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayDeployment))
					oldIngressGatewayDeployment := ingressGatewayDeployment.DeepCopy()
					ingressGatewayDeployment.Spec.Service = &servicemeshapi.IngressGatewayService{
						Type: corev1.ServiceTypeNodePort,
					}
					// Update
					assert.NoError(t, framework.K8sAPIs.Update(ctx, ingressGatewayDeployment, oldIngressGatewayDeployment))
				} else {
					// min > max replicas
					if tt.args.hasDeployment && tt.args.minReplicas > tt.args.maxReplicas {
						ingressGatewayDeployment.Spec.Deployment.Autoscaling = &servicemeshapi.Autoscaling{MinPods: int32(tt.args.minReplicas), MaxPods: int32(tt.args.maxReplicas)}
					}
					// multiple ports and different protocols
					if tt.args.differentProtocols {
						ports := make([]servicemeshapi.GatewayListener, 0)
						pNumber := int32(0)
						sport := int32(0)
						ports = append(ports, servicemeshapi.GatewayListener{
							Protocol:    corev1.ProtocolTCP,
							Port:        &pNumber,
							ServicePort: &sport,
						})
						pNumber = int32(1)
						sport = int32(1)
						ports = append(ports, servicemeshapi.GatewayListener{
							Protocol:    corev1.ProtocolUDP,
							Port:        &pNumber,
							ServicePort: &sport,
						})
						ingressGatewayDeployment.Spec.Ports = ports
					}
					err := framework.K8sAPIs.Create(ctx, ingressGatewayDeployment)
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
			} else {
				if tt.args.hasHPA {
					// create IGD
					assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayDeployment))

					// create HPA
					maxReplicas := ingressGatewayDeployment.Spec.Deployment.Autoscaling.MaxPods
					minReplicas := ingressGatewayDeployment.Spec.Deployment.Autoscaling.MinPods
					hpa := autoscalingv1.HorizontalPodAutoscaler{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ingressGatewayDeployment.Namespace,
							Name:      ingressGatewayDeployment.Name + string(commons.NativeHorizontalPodAutoScalar),
						},
					}
					targetCPUUtilizationPercentage := int32(commons.TargetCPUUtilizationPercentage)
					hpa.Spec = autoscalingv1.HorizontalPodAutoscalerSpec{
						TargetCPUUtilizationPercentage: &targetCPUUtilizationPercentage,
						MaxReplicas:                    maxReplicas,
						MinReplicas:                    &minReplicas,
						ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
							Kind:       "Deployment",
							Name:       "test",
							APIVersion: commons.DeploymentAPIVersion,
						},
					}
					framework.K8sAPIs.Create(ctx, &hpa)

					// Get HPA and validate
					herr := framework.K8sAPIs.Get(ctx, types.NamespacedName{Namespace: ingressGatewayDeployment.Namespace, Name: ingressGatewayDeployment.Name + string(commons.NativeHorizontalPodAutoScalar)}, &hpa)
					assert.Equal(t, tt.args.deleteHPA, herr == nil)

					curIngressGatewayDeployment := &servicemeshapi.IngressGatewayDeployment{}
					key := ns.NewNamespacedName(ingressGatewayDeployment)
					assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGatewayDeployment))
					assert.Equal(t, curIngressGatewayDeployment.Spec.Deployment.Autoscaling.MaxPods, hpa.Spec.MaxReplicas)
					assert.Equal(t, curIngressGatewayDeployment.Spec.Deployment.Autoscaling.MinPods, *hpa.Spec.MinReplicas)

					// Delete HPA and validate
					framework.K8sAPIs.Delete(ctx, &hpa)
					herr = framework.K8sAPIs.Get(ctx, types.NamespacedName{Namespace: ingressGatewayDeployment.Namespace, Name: ingressGatewayDeployment.Name + string(commons.NativeHorizontalPodAutoScalar)}, &hpa)
					assert.Equal(t, tt.args.deleteHPA, herr != nil)

					curIngressGatewayDeployment = &servicemeshapi.IngressGatewayDeployment{}
					assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGatewayDeployment))
					assert.Equal(t, curIngressGatewayDeployment.Spec.Deployment.Autoscaling.MaxPods, maxReplicas)
					assert.Equal(t, curIngressGatewayDeployment.Spec.Deployment.Autoscaling.MinPods, minReplicas)
					opts := equality.IgnoreFakeClientPopulatedFields()
					if tt.expectedStatus != nil {
						assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGatewayDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGatewayDeployment.Status, opts))
					}
				} else if tt.args.deleteService {
					// create IGD
					assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayDeployment))

					// create service
					service := corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ingressGatewayDeployment.Namespace,
							Name:      ingressGatewayDeployment.Name + string(commons.NativeService),
						},
					}
					portMap := make(map[corev1.Protocol][]corev1.ServicePort)
					for _, port := range ingressGatewayDeployment.Spec.Ports {
						if port.ServicePort == nil {
							continue
						}
						if portMap[port.Protocol] == nil {
							portMap[port.Protocol] = make([]corev1.ServicePort, 0)
						}

						portMap[port.Protocol] = append(portMap[port.Protocol], corev1.ServicePort{
							Protocol: port.Protocol,
							Name:     port.Name,
							Port:     *port.ServicePort,
							TargetPort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: *port.Port,
							},
						})
					}
					portsSlice := make([]corev1.ServicePort, 0)
					for _, portsArr := range portMap {
						portsSlice = append(portsSlice, portsArr...)
					}
					service.Spec.Ports = portsSlice
					service.Spec.Type = ingressGatewayDeployment.Spec.Service.Type
					service.Spec.Selector = map[string]string{
						commons.IngressName: ingressGatewayDeployment.Name,
					}
					framework.K8sAPIs.Create(ctx, &service)

					// Get service and validate
					serr := framework.K8sAPIs.Get(ctx, types.NamespacedName{Namespace: ingressGatewayDeployment.Namespace, Name: ingressGatewayDeployment.Name + string(commons.NativeService)}, &service)
					assert.Equal(t, tt.args.deleteService, serr == nil)

					curIngressGatewayDeployment := &servicemeshapi.IngressGatewayDeployment{}
					key := ns.NewNamespacedName(ingressGatewayDeployment)
					assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGatewayDeployment))
					assert.Equal(t, curIngressGatewayDeployment.Spec.Service.Type, service.Spec.Type)

					// Delete service and validate
					framework.K8sAPIs.Delete(ctx, &service)
					serr = framework.K8sAPIs.Get(ctx, types.NamespacedName{Namespace: ingressGatewayDeployment.Namespace, Name: ingressGatewayDeployment.Name + string(commons.NativeService)}, &service)
					assert.Equal(t, tt.args.deleteService, serr != nil)

					curIngressGatewayDeployment = &servicemeshapi.IngressGatewayDeployment{}
					assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGatewayDeployment))
					assert.Equal(t, curIngressGatewayDeployment.Spec.Service.Type, corev1.ServiceTypeLoadBalancer)
					opts := equality.IgnoreFakeClientPopulatedFields()
					if tt.expectedStatus != nil {
						assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGatewayDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGatewayDeployment.Status, opts))
					}
				} else {
					assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayDeployment))
					curIngressGatewayDeployment := &servicemeshapi.IngressGatewayDeployment{}
					opts := equality.IgnoreFakeClientPopulatedFields()
					key := ns.NewNamespacedName(ingressGatewayDeployment)
					assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGatewayDeployment))

					if tt.args.hasService {
						service := corev1.Service{}
						assert.NoError(t, framework.K8sAPIs.Get(ctx, types.NamespacedName{Namespace: ingressGatewayDeployment.Namespace, Name: ingressGatewayDeployment.Name + string(commons.NativeService)}, &service))
						assert.Equal(t, tt.args.serviceLabels, service.Labels)
						assert.Equal(t, tt.args.serviceAnnotations, service.Annotations)
					}

					if tt.expectedStatus != nil {
						assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGatewayDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGatewayDeployment.Status, opts))
					}
				}
			}

			assert.NoError(t, framework.K8sAPIs.Delete(ctx, service))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, pod))
			if tt.expectedErr == nil {
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGatewayDeployment))
			}
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGateway))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, mesh))
		})
		afterEach(framework)
	}
}
