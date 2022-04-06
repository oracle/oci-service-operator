/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package proxycontroller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	. "github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

type ProxyControllerArgs struct {
	NamespaceLabel                     string
	PodLabel                           string
	CreatePod                          bool
	CreateService                      bool
	CreateVDB                          bool
	CreatePDB                          bool
	DeletePDB                          bool
	VDBRefPodAnnotation                string
	UpdateConfigMap                    bool
	CreateConfigMap                    bool
	CreateConfigMapWithoutSidecarImage bool
}

type Result struct {
	PodEvicted              bool
	OutdatedProxyAnnotation bool
	Err                     error
}

var (
	TestNamespace         *corev1.Namespace
	TestPod               *corev1.Pod
	TestService           *corev1.Service
	TestVDB               *servicemeshapi.VirtualDeploymentBinding
	TestPDB               *v1beta1.PodDisruptionBudget
	MeshConfigMap         *corev1.ConfigMap
	MeshOperatorNamespace *corev1.Namespace
)

func Initialize(podWithProxy bool) {
	TestNamespace = NewNamespace("test", map[string]string{})
	TestService = NewKubernetesService("product", "test")
	TestVDB = NewVdbWithVdRef("product-vdb", "test", "product", "product")
	TestPDB = NewPodDisruptionBudget("product-pdb", "test", 1, "product")
	MeshConfigMap = NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName, map[string]string{
		commons.ProxyLabelInMeshConfigMap: "sm-proxy-image",
	})
	MeshOperatorNamespace = NewNamespace(commons.OsokNamespace, map[string]string{})
	if podWithProxy {
		TestPod = NewPodWithServiceMeshProxy("product", "test")
	} else {
		TestPod = NewPodWithoutServiceMeshProxy("product", "test")
	}
}

func UpdateResources(ctx context.Context, testFramework *framework.Framework, args ProxyControllerArgs, t *testing.T) {
	if args.NamespaceLabel != "" {
		err := UpdateProxyInjectionNamespaceLabel(ctx, testFramework.K8sClient, TestNamespace, args.NamespaceLabel)
		if err != nil {
			t.Fatal("Failed to update proxy injection namespace labels", err)
		}
	}

	if args.VDBRefPodAnnotation != "" {
		err := UpdateVDBRefPodAnnotation(ctx, testFramework.K8sClient, TestPod, args.VDBRefPodAnnotation)
		if err != nil {
			t.Fatal("Failed to update vdb ref annotation in pod", err)
		}
	}

	if args.UpdateConfigMap {
		MeshConfigMap.Data[commons.ProxyLabelInMeshConfigMap] = "sm-proxy-image-1"
		err := UpdateConfigMap(ctx, testFramework.K8sClient, MeshConfigMap)
		if err != nil {
			t.Fatal("Failed to update proxy version in config map", err)
		}
	}

	if args.CreateConfigMap {
		MeshConfigMap.Data[commons.ProxyLabelInMeshConfigMap] = "sm-proxy-image"
		if UpdateConfigMap(ctx, testFramework.K8sClient, MeshConfigMap) != nil {
			err := CreateConfigMap(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: MeshConfigMap.Namespace, Name: MeshConfigMap.Name}, MeshConfigMap)
			if err != nil {
				t.Fatal("Failed to create config map", err)
			}
		}
	}

	if args.CreateConfigMapWithoutSidecarImage {
		MeshConfigMap.Data[commons.ProxyLabelInMeshConfigMap] = ""
		if UpdateConfigMap(ctx, testFramework.K8sClient, MeshConfigMap) != nil {
			err := CreateConfigMap(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: MeshConfigMap.Namespace, Name: MeshConfigMap.Name}, MeshConfigMap)
			if err != nil {
				t.Fatal("Failed to create config map without sidecar image", err)
			}
		}
	}

	if args.CreatePod {
		err := CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: TestPod.Name}, TestPod)
		if err != nil {
			t.Fatal("Failed to create product pod", err)
		}
		if args.PodLabel != "" {
			err = UpdateProxyInjectionPodLabel(ctx, testFramework.K8sClient, TestPod, args.PodLabel)
			if err != nil {
				t.Fatal("Failed to update proxy injection pod labels", err)
			}
		}
	}

	if args.CreateService {
		err := CreateKubernetesService(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: TestService.Name}, TestService)
		if err != nil {
			t.Fatal("Failed to create product service", err)
		}
	}

	if args.CreateVDB {
		err := CreateVirtualDeploymentBinding(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: TestVDB.Name}, TestVDB)
		if err != nil {
			t.Fatal("Failed to create product vdb", err)
		}
		err = UpdateVirtualDeploymentBinding(ctx, testFramework.K8sClient, TestVDB)
		if err != nil {
			t.Fatal("Failed to update product vdb", err)
		}
	}

	if args.CreatePDB {
		AddPodConditionReady(TestPod)
		if _, err := testFramework.K8sClientset.CoreV1().Pods("test").UpdateStatus(ctx, TestPod, metav1.UpdateOptions{}); err != nil {
			t.Fatal("Failed to update status for product")
		}
		err := CreatePodDisruptionBudget(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: TestPDB.Name}, TestPDB)
		if err != nil {
			t.Fatal("Failed to create product pdb", err)
		}
	}
}

func ValidateResult(err error, args ProxyControllerArgs, want Result, ctx context.Context,
	testFramework *framework.Framework, t *testing.T) {

	if err != nil {
		assert.EqualError(t, want.Err, err.Error())
	} else {
		assert.NoError(t, err)
	}

	if args.CreatePod {
		err = GetPod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: TestPod.Name}, TestPod)
		if want.PodEvicted && err == nil {
			t.Fatal("Pod should have been evicted", err)
		} else if !want.PodEvicted && err != nil {
			t.Fatal("Pod should exist", err)
		}
	}

	if want.OutdatedProxyAnnotation {
		val, ok := TestPod.Annotations[commons.OutdatedProxyAnnotation]
		if !ok || val != "true" {
			t.Fatal("Pod should contain outdated proxy annotation")
		}
	}
}
