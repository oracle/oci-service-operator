/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functional

import (
	"context"
	"log"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	"github.com/go-logr/logr"
	api "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	servicemeshoci "github.com/oracle/oci-service-operator/controllers/servicemesh.oci"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/accesspolicy"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/ingressgateway"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/ingressgatewaydeployment"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/ingressgatewayroutetable"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/injectproxy"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/cache"
	customControllers "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/controllers"
	meshwebhook "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/webhook"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/mesh"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/serviceupdate"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/upgradeproxy"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualdeployment"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualdeploymentbinding"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualservice"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualserviceroutetable"
	mocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	testK8s "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	stopChan      context.Context
	cancel        context.CancelFunc
	testEnv       *envtest.Environment
	setupLog      = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup")}
	metricsClient *metrics.Metrics
)

type TestEnvFramework interface {
	SetupTestEnv() (*envtest.Environment, *rest.Config)
	CleanUpTestEnv(testEnv *envtest.Environment)
	SetupTestFramework(t *testing.T, config *rest.Config) *Framework
	CleanUpTestFramework(framework *Framework)
}

type defaultTestEnvFramework struct{}

func NewDefaultTestEnvFramework() TestEnvFramework {
	return &defaultTestEnvFramework{}
}

func (f *defaultTestEnvFramework) SetupTestEnv() (*envtest.Environment, *rest.Config) {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(api.AddToScheme(scheme.Scheme))
	utilruntime.Must(servicemeshapi.AddToScheme(scheme.Scheme))

	// Get the root of the current file to use in CRD paths
	_, filename, _, _ := goruntime.Caller(0) //nolint
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..")

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join(root, "config", "crd", "bases")},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join(root, "config", "webhook")},
		},
		ErrorIfCRDPathMissing: true,
	}

	config := testK8s.SetupTestEnv(testEnv)
	return testEnv, config
}

func (f *defaultTestEnvFramework) SetupTestFramework(t *testing.T, config *rest.Config) *Framework {
	k8sClientset := testK8s.NewTestEnvK8sClientSet(config)

	k8sManager, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
		Port:               testEnv.WebhookInstallOptions.LocalServingPort,
		CertDir:            testEnv.WebhookInstallOptions.LocalServingCertDir,
		Host:               testEnv.WebhookInstallOptions.LocalServingHost,
	})

	if err != nil {
		log.Fatalf("Failed to create k8sManager: %v", err)
	}
	k8sClient := k8sManager.GetClient()

	metricsClient := getMetricsClient()

	meshClient := testK8s.NewFakeMeshClient(t)

	cacheLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("integration-test").WithName("Cache")}
	customCachesConfig := cache.CustomCacheConfig{ResyncPeriod: 10 * time.Minute, ClientSet: k8sClientset, Log: cacheLogger}

	meshCacheManager := customCachesConfig.NewSharedCaches()
	meshCacheManager.SetupWithManager(k8sManager, cacheLogger)

	referenceLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("integration-test").WithName("Resolver")}
	referenceResolver := references.NewDefaultResolver(k8sClient, meshClient, referenceLogger, meshCacheManager, k8sManager.GetAPIReader())
	hookServer := k8sManager.GetWebhookServer()

	// Register controllers
	meshLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")}
	meshResourceManager := mesh.NewMeshResourceManager(k8sClient, meshClient, meshLogger)
	meshServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, meshLogger, meshResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   meshServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  meshLogger,
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("Mesh"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.MeshFinalizer},
		},
		ResourceObject: new(servicemeshapi.Mesh),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "Mesh")
		os.Exit(1)
	}

	meshValidatorLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")}
	meshValidator := mesh.NewMeshValidator(referenceResolver, meshValidatorLogger)
	meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, meshValidatorLogger)

	setupLog.InfoLog("registering mesh validator to the webhook server")

	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-mesh",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               meshLogger,
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("MeshValidator"),
			ValidationManager: meshServiceValidationManager,
		}})

	virtualServiceResourceManager := virtualservice.NewVirtualServiceResourceManager(k8sClient, meshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")}, referenceResolver)
	virtualServiceServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")}, virtualServiceResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   virtualServiceServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualService")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("VirtualService"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualServiceFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualService),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualService")
		os.Exit(1)
	}

	virtualServiceValidator := virtualservice.NewVirtualServiceValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualService")})
	virtualServiceServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualServiceValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualService")})
	setupLog.InfoLog("registering virtual service validator to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualservice",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualService")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("VirtualServiceValidator"),
			ValidationManager: virtualServiceServiceValidationManager,
		}})

	virtualDeploymentResourceManager := virtualdeployment.NewVirtualDeploymentResourceManager(k8sClient, meshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")}, referenceResolver)
	virtualDeploymentServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualDeployment")}, virtualDeploymentResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   virtualDeploymentServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualDeployment")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("VirtualDeployment"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualDeploymentFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualDeployment),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualDeployment")
		os.Exit(1)
	}
	virtualDeploymentValidator := virtualdeployment.NewVirtualDeploymentValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeployment")})
	virtualDeploymentServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualDeploymentValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeployment")})
	setupLog.InfoLog("registering virtual deployment validator to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualdeployment",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualDeployment")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("VirtualDeploymentValidator"),
			ValidationManager: virtualDeploymentServiceValidationManager,
		}})

	virtualServiceRouteTableResourceManager := virtualserviceroutetable.NewVirtualServiceRouteTableResourceManager(k8sClient, meshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualServiceRouteTable")}, referenceResolver)
	virtualServiceRouteTableServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualServiceRouteTable")}, virtualServiceRouteTableResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   virtualServiceRouteTableServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualServiceRouteTable")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("VirtualServiceRouteTable"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualServiceRouteTableFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualServiceRouteTable),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualServiceRouteTable")
		os.Exit(1)
	}
	virtualServiceRouteTableValidator := virtualserviceroutetable.NewVirtualServiceRouteTableValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualServiceRouteTable")})
	virtualServiceRouteTableServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualServiceRouteTableValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualServiceRouteTable")})
	setupLog.InfoLog("registering virtual service route table validator to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualserviceroutetable",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualServiceRouteTable")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("VirtualServiceRouteTableValidator"),
			ValidationManager: virtualServiceRouteTableServiceValidationManager,
		}})

	accessPolicyResourceManager := accesspolicy.NewAccessPolicyResourceManager(k8sClient, meshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AccessPolicy")}, referenceResolver)
	accessPolicyServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AccessPolicy")}, accessPolicyResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   accessPolicyServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("AccessPolicy")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("AccessPolicy"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.AccessPolicyFinalizer},
		},
		ResourceObject: new(servicemeshapi.AccessPolicy),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "AccessPolicy")
		os.Exit(1)
	}
	accessPolicyValidator := accesspolicy.NewAccessPolicyValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AccessPolicy")})
	accessPolicyServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(accessPolicyValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AccessPolicy")})
	setupLog.InfoLog("registering access policy to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-accesspolicy",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("AccessPolicy")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("AccessPolicyValidator"),
			ValidationManager: accessPolicyServiceValidationManager,
		}})

	ingressGatewayResourceManager := ingressgateway.NewIngressGatewayResourceManager(k8sClient, meshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGateway")}, referenceResolver)
	ingressGatewayServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGateway")}, ingressGatewayResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   ingressGatewayServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("IngressGateway")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("IngressGateway"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.IngressGatewayFinalizer},
		},
		ResourceObject: new(servicemeshapi.IngressGateway),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "IngressGateway")
		os.Exit(1)
	}
	ingressGatewayValidator := ingressgateway.NewIngressGatewayValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGateway")})
	ingressGatewayServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(ingressGatewayValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGateway")})
	setupLog.InfoLog("registering ingress gateway to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-ingressgateway",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("IngressGateway")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("IngressGatewayValidator"),
			ValidationManager: ingressGatewayServiceValidationManager,
		}})

	// TODO: Pass the OSOKServiceManager during implementation of VDB CRD
	virtualDeploymentBindingResourceManager := virtualdeploymentbinding.NewVirtualDeploymentBindingServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualDeploymentBinding")}, k8sClientset, referenceResolver, meshClient)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   virtualDeploymentBindingResourceManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualDeploymentBinding")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("VirtualDeploymentBinding"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualDeploymentBindingFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualDeploymentBinding),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualDeploymentBinding")
		os.Exit(1)
	}
	virtualDeploymentBindingValidator := virtualdeploymentbinding.NewVirtualDeploymentBindingValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeploymentBinding")})
	virtualDeploymentBindingServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualDeploymentBindingValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeploymentBinding")})
	setupLog.InfoLog("registering virtual deployment binding to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualdeploymentbinding",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualDeploymentBinding")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("VirtualDeploymentBinding"),
			ValidationManager: virtualDeploymentBindingServiceValidationManager,
		}})

	ingressGatewayRouteTableResourceManager := ingressgatewayroutetable.NewIngressGatewayRouteTableResourceManager(k8sClient, meshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGatewayRouteTable")}, referenceResolver)
	ingressGatewayRouteTableServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGatewayRouteTable")}, ingressGatewayRouteTableResourceManager)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   ingressGatewayRouteTableServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("IngressGatewayRouteTable")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("IngressGatewayRouteTable"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.IngressGatewayRouteTableFinalizer},
		},
		ResourceObject: new(servicemeshapi.IngressGatewayRouteTable),
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "IngressGatewayRouteTable")
		os.Exit(1)
	}
	ingressGatewayRouteTableValidator := ingressgatewayroutetable.NewIngressGatewayRouteTableValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayRouteTable")})
	ingressGatewayRouteTableServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(ingressGatewayRouteTableValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayRouteTable")})
	setupLog.InfoLog("registering ingress gateway route table to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-ingressgatewayroutetable",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("IngressGatewayRouteTable")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("IngressGatewayRouteTableValidator"),
			ValidationManager: ingressGatewayRouteTableServiceValidationManager,
		}})

	enqueueRequestsForConfigmapEvents := ingressgatewaydeployment.NewEnqueueRequestsForConfigmapEvents(k8sClient, ctrl.Log.WithName("controllers").WithName("IngressGatewayDeployment"), meshCommons.OsokNamespace)
	igdCustomWatches := []servicemeshoci.CustomWatch{
		ingressgatewaydeployment.GetIngressGatewayDeploymentOwnerWatch(&appsv1.Deployment{}),
		ingressgatewaydeployment.GetIngressGatewayDeploymentOwnerWatch(&corev1.Service{}),
		ingressgatewaydeployment.GetIngressGatewayDeploymentOwnerWatch(&autoscalingv1.HorizontalPodAutoscaler{}),
		{
			Src:          &source.Kind{Type: &corev1.ConfigMap{}},
			EventHandler: enqueueRequestsForConfigmapEvents,
		},
	}
	ingressGatewayDeploymentResourceManager := ingressgatewaydeployment.NewIngressGatewayDeploymentServiceManager(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGatewayDeployment")}, k8sClientset, referenceResolver, meshCacheManager, meshCommons.OsokNamespace)
	if err = (&servicemeshoci.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               k8sClient,
			OSOKServiceManager:   ingressGatewayDeploymentResourceManager,
			Finalizer:            core.NewBaseFinalizer(k8sClient, ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("IngressGatewayDeployment")},
			Metrics:              metricsClient,
			Recorder:             k8sManager.GetEventRecorderFor("IngressGatewayDeployment"),
			Scheme:               scheme.Scheme,
			AdditionalFinalizers: []string{meshCommons.IngressGatewayDeploymentFinalizer},
		},
		ResourceObject: new(servicemeshapi.IngressGatewayDeployment),
		CustomWatches:  igdCustomWatches,
	}).SetupWithManagerWithMaxDelay(k8sManager, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "IngressGatewayDeployment")
		os.Exit(1)
	}
	ingressGatewayDeploymentValidator := ingressgatewaydeployment.NewIngressGatewayDeploymentValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayDeployment")})
	ingressGatewayDeploymentServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(ingressGatewayDeploymentValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayDeployment")})
	setupLog.InfoLog("registering ingress gateway deployment to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-ingressgatewaydeployment",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            k8sClient,
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("IngressGatewayDeployment")},
			Metrics:           metricsClient,
			Recorder:          k8sManager.GetEventRecorderFor("IngressGatewayDeployment"),
			ValidationManager: ingressGatewayDeploymentServiceValidationManager,
		}})

	injectProxyResourceHandler := injectproxy.NewDefaultResourceHandler(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("InjectProxy")}, k8sClientset, meshCommons.OsokNamespace)
	if err = customControllers.NewInjectProxyReconciler(
		k8sClient,
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("InjectProxy")},
		injectProxyResourceHandler,
	).SetupWithManager(k8sManager); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "InjectProxy")
		os.Exit(1)
	}

	upgradeProxyResourceHandler := upgradeproxy.NewDefaultResourceHandler(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("UpgradeProxy")}, k8sClientset)
	if err = customControllers.NewUpgradeProxyReconciler(
		k8sClient,
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("UpgradeProxy")},
		upgradeProxyResourceHandler,
		meshCommons.OsokNamespace,
	).SetupWithManager(k8sManager); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "UpgradeProxy")
		os.Exit(1)
	}

	serviceUpdateResourceHandler := serviceupdate.NewDefaultResourceHandler(k8sClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("Services")}, k8sClientset)
	if err = customControllers.NewServiceReconciler(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("Services")},
		serviceUpdateResourceHandler,
	).SetupWithManager(k8sManager); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "Services")
		os.Exit(1)
	}

	setupLog.InfoLog("registering webhooks to the webhook server")
	podMutatorHandler := meshwebhook.NewDefaultPodMutatorHandler(
		k8sClient,
		meshCacheManager,
		meshCommons.GlobalConfigMap)
	hookServer.Register("/mutate-pod", &webhook.Admission{Handler: podMutatorHandler})

	if err := k8sManager.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := k8sManager.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start the manager and upcoming requests will trigger corresponding reconciler
	stopChan, cancel = context.WithCancel(context.Background())
	go func() {
		err = k8sManager.Start(stopChan)
		if err != nil {
			log.Fatalf("Failed to run the operator: %v", err)
		}
	}()

	return &Framework{
		K8sClient:    k8sClient,
		K8sClientset: k8sClientset,
		K8sAPIs:      testK8s.NewDefaultK8sAPIs(k8sClient),
		MeshClient:   meshClient,
		Log:          ctrl.Log.WithName("testFramework"),
	}
}

func (f *defaultTestEnvFramework) CleanUpTestEnv(testEnv *envtest.Environment) {
	if err := testEnv.Stop(); err != nil {
		panic(err)
	}
}

func (f *defaultTestEnvFramework) CleanUpTestFramework(framework *Framework) {
	framework.Cleanup()
}

// Framework supports common operations used by integration tests. It comes with a K8sClient,
// K8sClientset and K8sAPIs. It uses a mock MeshClient(ControlPlane) by default.
type Framework struct {
	// k8s controller-runtime client is used in most of the controller operations.
	K8sClient client.Client
	// k8s client-go client set is needed when additional k8s operations are involved in tests.
	K8sClientset kubernetes.Interface
	// k8sAPIs supports kube-apiserver CRUD operations for each kind of resource.
	K8sAPIs    testK8s.K8sAPIs
	MeshClient *mocks.MockServiceMeshClient
	Log        logr.Logger
}

func (f *Framework) CreateNamespace(ctx context.Context, namespaceName string) {
	objectkey := types.NamespacedName{Name: namespaceName}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespaceName,
			Namespace: "",
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
	}

	err := f.K8sClient.Get(ctx, objectkey, ns)
	if err != nil {
		if err = f.K8sClient.Create(ctx, ns); err != nil {
			log.Fatalf("Failed to create test namespace: %+v", err)
		}
	}
}

// DeleteNamespace TODO: Fix the namespace deletion error. Namespace is in terminating state after deletion because it fails
// to delete the finalizer "kubernetes"
func (f *Framework) DeleteNamespace(ctx context.Context, namespaceName string) {
	key := types.NamespacedName{Name: namespaceName}
	observedObj := &corev1.Namespace{}
	if err := f.K8sClient.Get(ctx, key, observedObj); err != nil {
		log.Fatalf("Namespace not found: %+v", err)
	}

	oldObj := observedObj.DeepCopy()
	finalizers := observedObj.Spec.Finalizers

	for _, finalizer := range finalizers {
		controllerutil.RemoveFinalizer(observedObj, string(finalizer))
	}

	if err := f.K8sClient.Patch(ctx, observedObj, client.MergeFrom(oldObj)); err != nil {
		log.Fatalf("Namespace not found: %+v", err)
	}

	if err := f.K8sClient.Delete(ctx, observedObj); err != nil {
		log.Fatalf("Failed to delete test namespace: %+v", err)
	}
}

func getMetricsClient() *metrics.Metrics {
	if metricsClient == nil {
		metricsClient = metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})
	}
	return metricsClient
}

func (f *Framework) Cleanup() {
	// Stop the k8sManager
	cancel()
}
