/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package framework

import (
	"path/filepath"
	goruntime "runtime"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	api "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	"github.com/oracle/oci-service-operator/test/servicemesh/k8s"
)

var (
	scheme  = runtime.NewScheme()
	testEnv *envtest.Environment
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(api.AddToScheme(scheme))
	utilruntime.Must(servicemeshapi.AddToScheme(scheme))

	// Get the root of the current file to use in CRD paths
	_, filename, _, _ := goruntime.Caller(0) //nolint
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join(root, "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
}

// Framework supports common operations used by unit tests. It comes with a K8sClient,
// K8sClientset and K8sAPIs.
type Framework struct {
	// k8s controller-runtime client is used in most of the controller operations.
	K8sClient client.Client
	// k8s client-go client set is needed when additional k8s operations are involved in tests.
	K8sClientset kubernetes.Interface
	// k8sAPIs supports kube-apiserver CRUD operations for each kind of resource.
	K8sAPIs    k8s.K8sAPIs
	MeshClient *meshMocks.MockServiceMeshClient
	Resolver   *meshMocks.MockResolver
	Cache      *meshMocks.MockCacheMapClient
	Log        loggerutil.OSOKLogger
	// useTestEnv indicates the test framework uses a testEnv which should be clear after tests.
	useTestEnv bool
}

func NewDefaultFramework(k8sClient client.Client,
	k8sClientset kubernetes.Interface, t *testing.T) *Framework {
	return &Framework{
		K8sClient:    k8sClient,
		K8sClientset: k8sClientset,
		K8sAPIs:      k8s.NewDefaultK8sAPIs(k8sClient),
		MeshClient:   k8s.NewFakeMeshClient(t),
		Resolver:     k8s.NewFakeResolver(t),
		Log:          loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("testFramework").WithName("ServiceMesh")},
		Cache:        k8s.NewFakeCache(t),
	}
}

// NewFakeClientFramework creates a test framework with fake k8sClient and fake k8sClientSet
// You won't need to create a test namespace using fake clients
func NewFakeClientFramework(t *testing.T) *Framework {
	k8sClient := k8s.NewFakeK8sClient(scheme)
	k8sClientSet := k8s.NewFakeK8sClientSet()
	return NewDefaultFramework(k8sClient, k8sClientSet, t)
}

// NewTestEnvClientFramework creates a test framework with 8sClient and k8sClientSet using test environment in /testbin
// You will need to create a test namespace for your test cases
func NewTestEnvClientFramework(t *testing.T) *Framework {
	cfg := k8s.SetupTestEnv(testEnv)
	k8sClient := k8s.NewTestEnvK8sClient(cfg, scheme)
	k8sClientSet := k8s.NewTestEnvK8sClientSet(cfg)
	f := NewDefaultFramework(k8sClient, k8sClientSet, t)
	f.useTestEnv = true
	return f
}

func (f *Framework) Cleanup() {
	if f.useTestEnv {
		if err := testEnv.Stop(); err != nil {
			panic(err)
		}
	}
}
