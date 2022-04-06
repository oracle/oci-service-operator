/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package upgradeproxy

import (
	"context"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	testProxyController "github.com/oracle/oci-service-operator/test/servicemesh/proxycontroller"
)

var (
	testFramework               *framework.Framework
	ctx                         context.Context
	upgradeProxyResourceHandler ResourceHandler
	PodUpgradeEnabled           = true
	PodUpgradeDisabled          = false
)

func BeforeSuite(t *testing.T) {
	ctx = context.Background()
	testFramework = framework.NewTestEnvClientFramework(t)
	upgradeProxyResourceHandler = NewDefaultResourceHandler(testFramework.K8sClient,
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("UpgradeProxyResourceHandler"), FixedLogs: make(map[string]string)},
		testFramework.K8sClientset)
}

func AfterSuite(t *testing.T) {
	err := DeleteNamespace(context.Background(), testFramework.K8sClient, testProxyController.TestNamespace)
	if err != nil {
		t.Fatal("Failed to delete test namespace", err)
	}
	err = DeleteNamespace(context.Background(), testFramework.K8sClient, testProxyController.MeshOperatorNamespace)
	if err != nil {
		t.Fatal("Failed to delete mesh operator namespace", err)
	}
	testFramework.Cleanup()
}

func BeforeEach(ctx context.Context, t *testing.T) {
	testProxyController.Initialize(true)
	err := CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: testProxyController.TestNamespace.Name}, testProxyController.TestNamespace)
	if err != nil {
		t.Fatal("Failed to create test namespace", err)
	}
	err = CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testProxyController.TestPod.Namespace, Name: testProxyController.TestPod.Name}, testProxyController.TestPod)
	if err != nil {
		t.Fatal("Failed to create test pod", err)
	}
	err = CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: commons.OsokNamespace}, testProxyController.MeshOperatorNamespace)
	if err != nil {
		t.Fatal("Failed to create mesh operator namespace", err)
	}
	err = CreateConfigMap(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testProxyController.MeshConfigMap.Namespace, Name: testProxyController.MeshConfigMap.Name}, testProxyController.MeshConfigMap)
	if err != nil {
		t.Fatal("Failed to create test config map", err)
	}
}

func TestUpgradeProxyReconcile(t *testing.T) {
	tests := []struct {
		name string
		args testProxyController.ProxyControllerArgs
		want testProxyController.Result
	}{
		{
			name: "ProxyInjectionLabel:[Namespace:disabled]",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel: commons.Disabled,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled]",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel: commons.Enabled,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], PodVDBRefAnnotation:empty Set Outdated Proxy Annotation",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "",
				UpdateConfigMap:     true,
			},
			want: testProxyController.Result{
				OutdatedProxyAnnotation: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], PodVDBRefAnnotation:mismatch Set Outdated Proxy Annotation",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "test/randomvdbbinding",
				UpdateConfigMap:     true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], VDB:NotFound",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "test/product-vdb",
				UpdateConfigMap:     true,
				CreateVDB:           false,
			},
			want: testProxyController.Result{
				OutdatedProxyAnnotation: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], VDB:Match PodUpgradeEnabled:false Set Outdated Proxy Annotation",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "test/product-vdb",
				UpdateConfigMap:     true,
				CreateService:       true,
				CreateVDB:           true,
			},
			want: testProxyController.Result{
				OutdatedProxyAnnotation: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], VDB match, PodUpgradeEnabled: true, Evict pod",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "test/product-vdb",
				UpdateConfigMap:     true,
				CreateService:       true,
				CreateVDB:           true,
			},
			want: testProxyController.Result{
				PodEvicted: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], VDB matched, podUpgradeEnabled:true, Respects PDB and does not evict",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "test/product-vdb",
				UpdateConfigMap:     true,
				CreateService:       true,
				CreateVDB:           true,
				CreatePDB:           true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled], VDB matched, podUpgradeEnabled:true, Respects PDB and evicts pod in subsequent runs",
			args: testProxyController.ProxyControllerArgs{
				NamespaceLabel:      commons.Enabled,
				VDBRefPodAnnotation: "test/product-vdb",
				CreateService:       true,
				CreateVDB:           true,
				CreatePDB:           true,
				DeletePDB:           true,
			},
			want: testProxyController.Result{
				PodEvicted: true,
			},
		},
	}
	BeforeSuite(t)
	for _, tt := range tests {
		BeforeEach(ctx, t)
		t.Run(tt.name, func(t *testing.T) {
			testProxyController.UpdateResources(ctx, testFramework, tt.args, t)
			times := 1
			if tt.args.DeletePDB {
				times = 2
			}
			var err error
			for i := 0; i < times; i++ {
				err = upgradeProxyResourceHandler.Reconcile(ctx, testProxyController.MeshConfigMap)
				if err != nil {
					_, err = merrors.HandleErrorAndRequeue(err, testFramework.Log)
				}
				if tt.args.DeletePDB && i == 0 {
					if err != nil {
						break
					}
					err = DeletePodDisruptionBudget(ctx, testFramework.K8sClient, testProxyController.TestPDB)
					if err != nil {
						t.Fatal("Failed to delete test pdb", err)
					}
				}
			}
		})
	}
	AfterSuite(t)
}
