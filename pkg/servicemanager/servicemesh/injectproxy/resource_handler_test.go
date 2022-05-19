/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package injectproxy

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	testProxyController "github.com/oracle/oci-service-operator/test/servicemesh/proxycontroller"
)

var (
	testFramework              *framework.Framework
	ctx                        context.Context
	injectProxyResourceHandler ResourceHandler
	PodUpgradeEnabled          = true
	PodUpgradeDisabled         = false
)

func BeforeSuite(t *testing.T) {
	ctx = context.Background()
	testFramework = framework.NewTestEnvClientFramework(t)
	injectProxyResourceHandler = NewDefaultResourceHandler(testFramework.K8sClient,
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("InjectProxyResourceHandler")},
		testFramework.K8sClientset, commons.OsokNamespace)
}

func AfterSuite(t *testing.T) {
	err := DeleteNamespace(context.Background(), testFramework.K8sClient, testProxyController.TestNamespace)
	if err != nil {
		t.Fatal("Failed to delete test namespace", err)
	}
	testFramework.Cleanup()
}

func BeforeEach(ctx context.Context, t *testing.T) {
	testProxyController.Initialize(false)
	err := CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: testProxyController.TestNamespace.Name}, testProxyController.TestNamespace)
	if err != nil {
		t.Fatal("Failed to create test namespace", err)
	}
	err = CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: commons.OsokNamespace}, testProxyController.MeshOperatorNamespace)
	if err != nil {
		t.Fatal("Failed to create mesh operator namespace", err)
	}
}

func TestInjectProxyReconcile(t *testing.T) {
	tests := []struct {
		name string
		args testProxyController.ProxyControllerArgs
		want testProxyController.Result
	}{
		{
			name: "No config map found",
		},
		{
			name: "No SIDECAR_IMAGE in the config map",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMapWithoutSidecarImage: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:notset]",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:invalid]",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled],PodsList:Empty",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        "",
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled,Pod:notset],VDBList:Empty",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				CreatePod:       true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled,Pod:invalid], VDB not matched",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        "invalid",
				CreatePod:       true,
				CreateVDB:       true,
			},
		},
		{
			name: "ProxyInjectionLabel [Namespace:disabled Pod:disabled], VDB not matched",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        Disabled,
				CreatePod:       true,
				CreateVDB:       true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled,Pod:enabled], VDB not matched",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        Enabled,
				CreatePod:       true,
				CreateVDB:       true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled,Pod:enabled], VDB matched, Evict pod",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        Enabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
			},
			want: testProxyController.Result{
				PodEvicted: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled,Pod:enabled], VDB matched, Respects PDB and does not evict",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        Enabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
				CreatePDB:       true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:disabled,Pod:enabled], VDB matched, Respects PDB and evicts in subsequent runs",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Disabled,
				PodLabel:        Enabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
				CreatePDB:       true,
				DeletePDB:       true,
			},
			want: testProxyController.Result{
				PodEvicted: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled,Pod:notset], VDB match, Evict pod",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Enabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
			},
			want: testProxyController.Result{
				PodEvicted: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled,Pod:enabled], VDB matched, Respects PDB and does not evict",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Enabled,
				PodLabel:        Enabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
				CreatePDB:       true,
			},
			want: testProxyController.Result{
				Err: merrors.NewRequeueAfter(time.Minute),
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled,Pod:enabled], VDB matched, Respects PDB and evicts pod in subsequent runs",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Enabled,
				PodLabel:        Enabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
				CreatePDB:       true,
				DeletePDB:       true,
			},
			want: testProxyController.Result{
				PodEvicted: true,
			},
		},
		{
			name: "ProxyInjectionLabel:[Namespace:enabled,Pod:disabled], VDB match, Not evict pod",
			args: testProxyController.ProxyControllerArgs{
				CreateConfigMap: true,
				NamespaceLabel:  Enabled,
				PodLabel:        Disabled,
				CreatePod:       true,
				CreateVDB:       true,
				CreateService:   true,
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
				err = injectProxyResourceHandler.Reconcile(ctx, testProxyController.TestNamespace)
				if err != nil {
					_, err = merrors.HandleErrorAndRequeue(ctx, err, testFramework.Log)
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
			testProxyController.ValidateResult(err, tt.args, tt.want, ctx, testFramework, t)
		})
	}
	AfterSuite(t)
}
