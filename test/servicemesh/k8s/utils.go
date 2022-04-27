/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package k8s

import (
	"log"
	"testing"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

func NewFakeK8sClient(scheme *runtime.Scheme) client.Client {
	K8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	return K8sClient
}

func NewFakeK8sClientSet() kubernetes.Interface {
	return fakeclientset.NewSimpleClientset()
}

func NewTestEnvK8sClient(cfg *rest.Config, scheme *runtime.Scheme) client.Client {
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("Failed to create test k8sClient: %+v", err)
	}
	return k8sClient
}

func NewTestEnvK8sClientSet(cfg *rest.Config) kubernetes.Interface {
	k8sClientSet := kubernetes.NewForConfigOrDie(cfg)
	if k8sClientSet == nil {
		log.Fatalf("k8sClientSet should be be empty.")
	}
	return k8sClientSet
}

func NewFakeMeshClient(t *testing.T) *meshMocks.MockServiceMeshClient {
	mockCtrl := gomock.NewController(t)
	return meshMocks.NewMockServiceMeshClient(mockCtrl)
}

func NewFakeCache(t *testing.T) *meshMocks.MockCacheMapClient {
	mockCtrl := gomock.NewController(t)
	return meshMocks.NewMockCacheMapClient(mockCtrl)
}

func NewFakeResolver(t *testing.T) *meshMocks.MockResolver {
	mockCtrl := gomock.NewController(t)
	return meshMocks.NewMockResolver(mockCtrl)
}

func SetupTestEnv(testEnv *envtest.Environment) *rest.Config {
	cfg, err := testEnv.Start()
	if err != nil {
		err = kerrors.NewAggregate([]error{err, testEnv.Stop()})
		panic(err)
	}
	if cfg == nil {
		log.Fatalf("Config should be be empty.")
	}
	return cfg
}
