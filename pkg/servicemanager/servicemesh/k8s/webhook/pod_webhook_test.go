/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"

	"github.com/stretchr/testify/assert"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	. "github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

var testNamespace = NewNamespace("product", map[string]string{})
var globalConfigMap = NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName,
	map[string]string{
		commons.ProxyLabelInMeshConfigMap:  "iad.ocir.io/idotidhmwg5o/awille/envoy:1.17.0.1",
		commons.MdsEndpointInMeshConfigMap: "http://144.25.97.129:443",
	})

var (
	ctx                             = context.Background()
	testFramework                   *framework.Framework
	podMutatorHandler               *PodMutatorHandler
	productVirtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding
	productService                  *corev1.Service
	activeVDBService                *corev1.Service
)

var serviceName = types.NamespacedName{
	Name:      "test",
	Namespace: "product",
}

var activeVDBServiceName = types.NamespacedName{
	Name:      "active",
	Namespace: "product",
}

func BeforeSuite(t *testing.T) {
	testFramework = framework.NewFakeClientFramework(t)
	podMutatorHandler = NewDefaultPodMutatorHandler(testFramework.K8sClient, testFramework.Cache, commons.GlobalConfigMap)
	decoder, _ := admission.NewDecoder(runtime.NewScheme())
	podMutatorHandler.InjectDecoder(decoder)
}

func BeforeTest(t *testing.T) {
	productService = NewKubernetesService(serviceName.Name, serviceName.Namespace)
	activeVDBService = NewKubernetesService(activeVDBServiceName.Name, activeVDBServiceName.Namespace)

	productVirtualDeploymentBinding = NewVdbWithVdRef("product-v1-binding", "product", "product", "test")
	productVirtualDeploymentBinding1 := NewVdbWithVdRef("product-v2-binding", "product", "product", "active")
	productVirtualDeploymentBinding1.Status = servicemeshapi.ServiceMeshStatus{
		VirtualDeploymentId: "123",
		MeshId:              "123",
		Conditions: []servicemeshapi.ServiceMeshCondition{
			{
				Type: servicemeshapi.ServiceMeshDependenciesActive,
				ResourceCondition: servicemeshapi.ResourceCondition{
					Status: metav1.ConditionTrue,
				},
			},
			{
				Type: servicemeshapi.ServiceMeshActive,
				ResourceCondition: servicemeshapi.ResourceCondition{
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	err1 := CreateKubernetesService(ctx,
		testFramework.K8sClient,
		types.NamespacedName{Namespace: serviceName.Namespace, Name: serviceName.Name}, productService)
	assert.NoError(t, err1)
	if err1 != nil {
		t.Fatal("Failed to create product service", err1)
	}

	err2 := CreateKubernetesService(ctx,
		testFramework.K8sClient,
		types.NamespacedName{Namespace: activeVDBServiceName.Namespace, Name: activeVDBServiceName.Name}, activeVDBService)
	assert.NoError(t, err2)
	if err2 != nil {
		t.Fatal("Failed to create product service", err2)
	}

	err := CreateVirtualDeploymentBinding(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "product", Name: productVirtualDeploymentBinding.Name}, productVirtualDeploymentBinding)
	assert.NoError(t, err)

	verr := CreateVirtualDeploymentBinding(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "product", Name: productVirtualDeploymentBinding1.Name}, productVirtualDeploymentBinding1)
	assert.NoError(t, verr)

}
func AfterTest(t *testing.T) {
	err := DeleteNamespace(context.Background(), testFramework.K8sClient, testNamespace)
	if err != nil {
		t.Fatal("Failed to delete product namespace", err)
	}
}
func TestMutateWebhook(t *testing.T) {

	type args struct {
		namespaceLabel       string
		podLabel             string
		matchServiceSelector bool
		createConfigMap      bool
		activeVDB            bool
		typeOfProbe          string
	}
	type expected struct {
		hasSidecar       bool
		hasAnnotation    bool
		hasInitContainer bool
		isAllowed        bool
		hasScheme        bool
		hasPath          bool
		hasPort          bool
		hasHttpHeaders   bool
		patchsize        int
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "TCP probe",
			args: args{
				namespaceLabel:       commons.Enabled,
				podLabel:             commons.Enabled,
				matchServiceSelector: true,
				createConfigMap:      true,
				activeVDB:            true,
				typeOfProbe:          commons.Tcp,
			},
			want: expected{
				hasSidecar:       true,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: true,
				hasScheme:        false,
				hasPath:          false,
				hasPort:          false,
				hasHttpHeaders:   false,
				patchsize:        7,
			},
		},
		{
			name: "Create patch",
			args: args{
				namespaceLabel:       commons.Enabled,
				podLabel:             commons.Enabled,
				matchServiceSelector: true,
				createConfigMap:      true,
				activeVDB:            true,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       true,
				hasAnnotation:    true,
				isAllowed:        true,
				hasInitContainer: true,
				hasScheme:        true,
				hasPath:          true,
				hasPort:          true,
				hasHttpHeaders:   true,
				patchsize:        11,
			},
		},
		{
			name: "VDB is not active",
			args: args{
				namespaceLabel:       commons.Enabled,
				podLabel:             commons.Enabled,
				matchServiceSelector: true,
				createConfigMap:      false,
				activeVDB:            false,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       false,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: false,
				patchsize:        0,
			},
		},

		{
			name: "VDB match fail",
			args: args{
				namespaceLabel:       commons.Enabled,
				podLabel:             commons.Enabled,
				matchServiceSelector: false,
				createConfigMap:      false,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       false,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: false,
				patchsize:        0,
			},
		},

		{
			name: " Do not add patch podLabel disabled",
			args: args{
				namespaceLabel:       commons.Enabled,
				podLabel:             commons.Disabled,
				matchServiceSelector: true,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       false,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: false,
				patchsize:        0,
			},
		},

		{
			name: " Do not add patch podLabel disabled and namespace disabled",
			args: args{
				namespaceLabel:       commons.Disabled,
				podLabel:             commons.Disabled,
				matchServiceSelector: true,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       false,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: false,
				patchsize:        0,
			},
		},

		{
			name: " pod label is enabled",
			args: args{
				namespaceLabel:       commons.Disabled,
				podLabel:             commons.Enabled,
				matchServiceSelector: true,
				createConfigMap:      true,
				activeVDB:            true,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       true,
				hasAnnotation:    true,
				isAllowed:        true,
				hasInitContainer: true,
				hasScheme:        true,
				hasPath:          true,
				hasPort:          true,
				hasHttpHeaders:   true,
				patchsize:        11,
			},
		},

		{
			name: " pod label is enabled and active VDB is false",
			args: args{
				namespaceLabel:       commons.Disabled,
				podLabel:             commons.Enabled,
				matchServiceSelector: true,
				createConfigMap:      false,
				activeVDB:            false,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       false,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: false,
				patchsize:        0,
			},
		},

		{
			name: " No namespace label",
			args: args{
				podLabel:             commons.Disabled,
				matchServiceSelector: true,
				typeOfProbe:          commons.Http,
			},
			want: expected{
				hasSidecar:       false,
				hasAnnotation:    false,
				isAllowed:        true,
				hasInitContainer: false,
				patchsize:        0,
			},
		},
	}
	BeforeSuite(t)
	for _, tt := range tests {
		BeforeTest(t)
		var namespace = testNamespace
		if tt.args.namespaceLabel != "" {
			namespace = NewNamespace("product", map[string]string{
				commons.ProxyInjectionLabel: tt.args.namespaceLabel,
			})
		}
		err := CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: "product"}, namespace)
		if err != nil {
			t.Fatal("Failed to create product namespace", err)
		}

		testFramework.Cache.EXPECT().GetNamespaceByKey("product").Return(namespace, nil)
		if tt.args.createConfigMap {
			testFramework.Cache.EXPECT().GetConfigMapByKey(commons.GlobalConfigMap).Return(globalConfigMap, nil)
		}

		pod := NewPodWithoutServiceMeshProxy("test", "product")
		pod.Labels = map[string]string{}
		if tt.args.podLabel != "" {
			pod.Labels[commons.ProxyInjectionLabel] = tt.args.podLabel
		}
		if tt.args.matchServiceSelector {
			if tt.args.activeVDB {
				pod.Labels["app"] = "active"
			} else {
				pod.Labels["app"] = "test"
			}
		}

		if tt.args.typeOfProbe == commons.Http {
			setHTTPProbes(pod)
		} else if tt.args.typeOfProbe == commons.Tcp {
			setTCPProbes(pod)
		}

		podRaw, rawErr := json.Marshal(pod)
		assert.NoError(t, rawErr)

		admissionReq := v1.AdmissionRequest{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Operation: v1.Create,
			Object: runtime.RawExtension{
				Raw: podRaw,
			},
		}
		response := podMutatorHandler.Handle(ctx, admission.Request{AdmissionRequest: admissionReq})

		patchMap := make(map[string]jsonpatch.JsonPatchOperation)
		for _, patch := range response.Patches {
			patchMap[patch.Path] = patch
		}
		assert.Equal(t, tt.want.isAllowed, response.Allowed)

		_, hasInitContainer := patchMap["/spec/initContainers"]
		assert.Equal(t, tt.want.hasInitContainer, hasInitContainer)
		_, hasSidecarContainer := patchMap["/spec/containers/1"]
		assert.Equal(t, tt.want.hasSidecar, hasSidecarContainer)
		_, hasAnnotation := patchMap["/metadata/annotations"]
		assert.Equal(t, tt.want.hasSidecar, hasAnnotation)
		_, hasScheme := patchMap["/spec/containers/0/livenessProbe/httpGet/scheme"]
		assert.Equal(t, tt.want.hasScheme, hasScheme)
		_, hasPath := patchMap["/spec/containers/0/livenessProbe/httpGet/path"]
		assert.Equal(t, tt.want.hasPath, hasPath)
		_, hasPort := patchMap["/spec/containers/0/livenessProbe/httpGet/port"]
		assert.Equal(t, tt.want.hasPort, hasPort)
		_, hasHttpHeaders := patchMap["/spec/containers/0/livenessProbe/httpGet/httpHeaders"]
		assert.Equal(t, tt.want.hasHttpHeaders, hasHttpHeaders)
		_, hasScheme = patchMap["/spec/containers/0/readinessProbe/httpGet/scheme"]
		assert.Equal(t, tt.want.hasScheme, hasScheme)
		_, hasPath = patchMap["/spec/containers/0/readinessProbe/httpGet/path"]
		assert.Equal(t, tt.want.hasPath, hasPath)
		_, hasPort = patchMap["/spec/containers/0/readinessProbe/httpGet/port"]
		assert.Equal(t, tt.want.hasPort, hasPort)
		_, hasHttpHeaders = patchMap["/spec/containers/0/readinessProbe/httpGet/httpHeaders"]
		assert.Equal(t, tt.want.hasHttpHeaders, hasHttpHeaders)
		if hasHttpHeaders {
			fmt.Println("start")
			for s, _ := range patchMap {
				fmt.Println(s)
			}
			fmt.Println("end")
			fmt.Println(response.Patches)
		}
		assert.Equal(t, tt.want.patchsize, len(response.Patches))
		AfterTest(t)
	}
}

func setTCPProbes(pod *corev1.Pod) {
	for i, container := range pod.Spec.Containers {
		if container.Name == "test" {
			pod.Spec.Containers[i].Ports = []corev1.ContainerPort{{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP}}
			pod.Spec.Containers[i].LivenessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 8080,
						},
						Host: "localhost",
					},
				},
				InitialDelaySeconds: commons.LivenessProbeInitialDelaySeconds,
			}
			pod.Spec.Containers[i].ReadinessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "tcp",
						},
						Host: "localhost",
					},
				},
			}
		}
	}
}

func setHTTPProbes(pod *corev1.Pod) {
	for i, container := range pod.Spec.Containers {
		if container.Name == "test" {
			pod.Spec.Containers[i].Ports = []corev1.ContainerPort{{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolHTTP}}
			pod.Spec.Containers[i].LivenessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: commons.LivenessProbeEndpointPath,
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 8080,
						},
					},
				},
				InitialDelaySeconds: commons.LivenessProbeInitialDelaySeconds,
			}
			pod.Spec.Containers[i].ReadinessProbe = &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/ready",
						Port: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "http",
						},
					},
				},
			}
		}
	}
}
