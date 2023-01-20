/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"flag"
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/oracle/oci-go-sdk/v65/common"

	servicemeshcontrollers "github.com/oracle/oci-service-operator/controllers/servicemesh.oci"
	"github.com/oracle/oci-service-operator/go_ensurefips"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/injectproxy"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/serviceupdate"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/updateconfigmap"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/upgradeproxy"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	accessPolicy "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/accesspolicy"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/ingressgateway"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/ingressgatewaydeployment"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/cache"
	customControllers "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/controllers"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualdeploymentbinding"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/ingressgatewayroutetable"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualdeployment"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualservice"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualserviceroutetable"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
	meshwebhook "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/webhook"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/mesh"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/controllers"
	"github.com/oracle/oci-service-operator/pkg/authhelper"
	"github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/credhelper/kubesecret"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/streams"
	"github.com/oracle/oci-service-operator/pkg/util"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup")}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ociv1beta1.AddToScheme(scheme))
	utilruntime.Must(servicemeshapi.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	// Check for fips compliance
	go_ensurefips.Compliant()

	// Allow OCI go sdk to use instance metadata service for region lookup
	common.EnableInstanceMetadataServiceLookup()

	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var initOSOKResources bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&initOSOKResources, "init-osok-resources", false,
		"Install OSOK prerequisites like CRDs and Webhooks at manager bootup")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	operatorNamespace := os.Getenv("POD_NAMESPACE")
	setupLog.InfoLog("Namespace of manager Pod", "namespace", operatorNamespace)
	operatorConditionName := os.Getenv("OPERATOR_CONDITION_NAME")
	setupLog.InfoLog("Operator condition name of manager Pod", "OPERATOR_CONDITION_NAME", operatorConditionName)
	var err error
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		setupLog.InfoLog("Loading the configuration from the ControllerManagerConfig configMap")
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.ErrorLog(err, "unable to load the config file")
			os.Exit(1)
		}
	} else {
		setupLog.InfoLog("Loading the configuration from the command arguments")
		options = ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     metricsAddr,
			Port:                   9443,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "40558063.oci",
		}
	}
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		setupLog.ErrorLog(err, "Can not get kubernetes config")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		setupLog.ErrorLog(err, "Can not create kubernetes client")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.ErrorLog(err, "unable to start manager")
		os.Exit(1)
	}

	if initOSOKResources {
		util.InitOSOK(mgr.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK")})
	}

	setupLog.InfoLog("Getting the config details")
	osokCfg := config.GetConfigDetails(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")})

	authConfigProvider := &authhelper.AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")}}

	provider, err := authConfigProvider.GetAuthProvider(osokCfg)
	if err != nil {
		setupLog.ErrorLog(err, "unable to get the oci configuration provider. Exiting setup")
		os.Exit(1)
	}

	metricsClient := metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})

	credClient := &kubesecret.KubeSecretClient{
		Client:  mgr.GetClient(),
		Log:     loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient")},
		Metrics: metricsClient,
	}

	if err = (&controllers.AutonomousDatabasesReconciler{
		Reconciler: &core.BaseReconciler{
			Client:             mgr.GetClient(),
			OSOKServiceManager: adb.NewAdbServiceManager(provider, credClient, scheme, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AutonomousDatabases")}),
			Finalizer:          core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("AutonomousDatabases")},
			Metrics:            metricsClient,
			Recorder:           mgr.GetEventRecorderFor("AutonomousDatabases"),
			Scheme:             scheme,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "AutonomousDatabases")
		os.Exit(1)
	}
	//if err = (&ociv1beta1.AutonomousDatabases{}).SetupWebhookWithManager(mgr); err != nil {
	//	setupLog.Error(err, "unable to create webhook", "webhook", "AutonomousDatabases")
	//	os.Exit(1)
	//}

	if err = (&controllers.StreamReconciler{
		Reconciler: &core.BaseReconciler{
			Client: mgr.GetClient(),
			OSOKServiceManager: streams.NewStreamServiceManager(provider, credClient, scheme, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Streams")},
				metricsClient),
			Finalizer: core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("Streams")},
			Metrics:   metricsClient,
			Recorder:  mgr.GetEventRecorderFor("Streams"),
			Scheme:    scheme,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "Streams")
		os.Exit(1)
	}
	if err = (&controllers.MySqlDBsystemReconciler{
		Reconciler: &core.BaseReconciler{
			Client:             mgr.GetClient(),
			OSOKServiceManager: dbsystem.NewDbSystemServiceManager(provider, credClient, scheme, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("MySqlDbSystem")}),
			Finalizer:          core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("MySqlDbSystem")},
			Metrics:            metricsClient,
			Recorder:           mgr.GetEventRecorderFor("MySqlDbSystem"),
			Scheme:             scheme,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "MySqlDbSystem")
		os.Exit(1)
	}

	serviceMeshClient, err := services.NewServiceMeshClient(provider, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Service Mesh").WithName("ServiceMeshClient")}, operatorConditionName)
	if err != nil {
		setupLog.ErrorLog(err, "unable to create mesh control plane client")
		os.Exit(1)
	}

	updateMeshConfigmapResourceHandler := updateconfigmap.NewDefaultResourceHandler(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("UpdateMeshConfigMap")})
	if err = customControllers.NewUpdateConfigMapController(mgr.GetClient(), updateMeshConfigmapResourceHandler,
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("UpdateConfigMap")}, operatorNamespace).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "UpdateConfigMap")
		os.Exit(1)
	}

	//Setup Cache needed for Mesh
	cacheLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}
	customCachesConfig := cache.CustomCacheConfig{ResyncPeriod: 10 * time.Minute, ClientSet: clientset, Log: cacheLogger}

	meshCacheManager := customCachesConfig.NewSharedCaches()
	meshCacheManager.SetupWithManager(mgr, cacheLogger)

	referenceResolver := references.NewDefaultResolver(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, meshCacheManager, mgr.GetAPIReader())
	hookServer := mgr.GetWebhookServer()

	meshResourceManager := mesh.NewMeshResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")})
	meshServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")}, meshResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   meshServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("Mesh")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("Mesh"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.MeshFinalizer},
		},
		ResourceObject: new(servicemeshapi.Mesh),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "Mesh")
		os.Exit(1)
	}

	meshValidator := mesh.NewMeshValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})
	meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})

	setupLog.InfoLog("registering mesh validator to the webhook server")

	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-mesh",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Client:            mgr.GetClient(),
			Reader:            mgr.GetAPIReader(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("Mesh")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("MeshValidator"),
			ValidationManager: meshServiceValidationManager,
		}})

	virtualServiceResourceManager := virtualservice.NewVirtualServiceResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")}, referenceResolver)
	virtualServiceServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")}, virtualServiceResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   virtualServiceServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualService")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("VirtualService"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualServiceFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualService),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualService")
		os.Exit(1)
	}

	virtualServiceValidator := virtualservice.NewVirtualServiceValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualService")})
	virtualServiceServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualServiceValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualService")})
	setupLog.InfoLog("registering virtual service validator to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualservice",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualService")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("VirtualServiceValidator"),
			ValidationManager: virtualServiceServiceValidationManager,
		}})

	virtualDeploymentResourceManager := virtualdeployment.NewVirtualDeploymentResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")}, referenceResolver)
	virtualDeploymentServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualDeployment")}, virtualDeploymentResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   virtualDeploymentServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualDeployment")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("VirtualDeployment"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualDeploymentFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualDeployment),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualDeployment")
		os.Exit(1)
	}
	virtualDeploymentValidator := virtualdeployment.NewVirtualDeploymentValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeployment")})
	virtualDeploymentServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualDeploymentValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeployment")})
	setupLog.InfoLog("registering virtual deployment validator to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualdeployment",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualDeployment")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("VirtualDeploymentValidator"),
			ValidationManager: virtualDeploymentServiceValidationManager,
		}})

	virtualServiceRouteTableResourceManager := virtualserviceroutetable.NewVirtualServiceRouteTableResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualServiceRouteTable")}, referenceResolver)
	virtualServiceRouteTableServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualServiceRouteTable")}, virtualServiceRouteTableResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   virtualServiceRouteTableServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualServiceRouteTable")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("VirtualServiceRouteTable"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualServiceRouteTableFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualServiceRouteTable),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualServiceRouteTable")
		os.Exit(1)
	}
	virtualServiceRouteTableValidator := virtualserviceroutetable.NewVirtualServiceRouteTableValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualServiceRouteTable")})
	virtualServiceRouteTableServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualServiceRouteTableValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualServiceRouteTable")})
	setupLog.InfoLog("registering virtual service route table validator to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualserviceroutetable",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualServiceRouteTable")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("VirtualServiceRouteTableValidator"),
			ValidationManager: virtualServiceRouteTableServiceValidationManager,
		}})

	accessPolicyResourceManager := accessPolicy.NewAccessPolicyResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AccessPolicy")}, referenceResolver)
	accessPolicyServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AccessPolicy")}, accessPolicyResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   accessPolicyServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("AccessPolicy")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("AccessPolicy"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.AccessPolicyFinalizer},
		},
		ResourceObject: new(servicemeshapi.AccessPolicy),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "AccessPolicy")
		os.Exit(1)
	}
	accessPolicyValidator := accessPolicy.NewAccessPolicyValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AccessPolicy")})
	accessPolicyServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(accessPolicyValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AccessPolicy")})
	setupLog.InfoLog("registering access policy to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-accesspolicy",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("AccessPolicy")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("AccessPolicyValidator"),
			ValidationManager: accessPolicyServiceValidationManager,
		}})

	ingressGatewayResourceManager := ingressgateway.NewIngressGatewayResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGateway")}, referenceResolver)
	ingressGatewayServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGateway")}, ingressGatewayResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   ingressGatewayServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("IngressGateway")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("IngressGateway"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.IngressGatewayFinalizer},
		},
		ResourceObject: new(servicemeshapi.IngressGateway),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "IngressGateway")
		os.Exit(1)
	}
	ingressGatewayValidator := ingressgateway.NewIngressGatewayValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGateway")})
	ingressGatewayServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(ingressGatewayValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGateway")})
	setupLog.InfoLog("registering ingress gateway to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-ingressgateway",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("IngressGateway")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("IngressGatewayValidator"),
			ValidationManager: ingressGatewayServiceValidationManager,
		}})

	virtualDeploymentBindingResourceManager := virtualdeploymentbinding.NewVirtualDeploymentBindingServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualDeploymentBinding")}, clientset, referenceResolver, serviceMeshClient)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   virtualDeploymentBindingResourceManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("VirtualDeploymentBinding")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("VirtualDeploymentBinding"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.VirtualDeploymentBindingFinalizer},
		},
		ResourceObject: new(servicemeshapi.VirtualDeploymentBinding),
	}).SetupWithManagerWithMaxDelay(mgr, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "VirtualDeploymentBinding")
		os.Exit(1)
	}
	virtualDeploymentBindingValidator := virtualdeploymentbinding.NewVirtualDeploymentBindingValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeploymentBinding")})
	virtualDeploymentBindingServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(virtualDeploymentBindingValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VirtualDeploymentBinding")})
	setupLog.InfoLog("registering virtual deployment binding to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-virtualdeploymentbinding",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("VirtualDeploymentBinding")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("VirtualDeploymentBinding"),
			ValidationManager: virtualDeploymentBindingServiceValidationManager,
		}})

	ingressGatewayRouteTableResourceManager := ingressgatewayroutetable.NewIngressGatewayRouteTableResourceManager(mgr.GetClient(), serviceMeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGatewayRouteTable")}, referenceResolver)
	ingressGatewayRouteTableServiceMeshManager := serviceMeshManager.NewServiceMeshServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGatewayRouteTable")}, ingressGatewayRouteTableResourceManager)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   ingressGatewayRouteTableServiceMeshManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("IngressGatewayRouteTable")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("IngressGatewayRouteTable"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.IngressGatewayRouteTableFinalizer},
		},
		ResourceObject: new(servicemeshapi.IngressGatewayRouteTable),
	}).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "IngressGatewayRouteTable")
		os.Exit(1)
	}
	ingressGatewayRouteTableValidator := ingressgatewayroutetable.NewIngressGatewayRouteTableValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayRouteTable")})
	ingressGatewayRouteTableServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(ingressGatewayRouteTableValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayRouteTable")})
	setupLog.InfoLog("registering ingress gateway route table to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-ingressgatewayroutetable",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("IngressGatewayRouteTable")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("IngressGatewayRouteTableValidator"),
			ValidationManager: ingressGatewayRouteTableServiceValidationManager,
		}})

	enqueueRequestsForConfigmapEvents := ingressgatewaydeployment.NewEnqueueRequestsForConfigmapEvents(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("IngressGatewayDeployment"), operatorNamespace)
	igdCustomWatches := []servicemeshcontrollers.CustomWatch{
		ingressgatewaydeployment.GetIngressGatewayDeploymentOwnerWatch(&appsv1.Deployment{}),
		ingressgatewaydeployment.GetIngressGatewayDeploymentOwnerWatch(&corev1.Service{}),
		ingressgatewaydeployment.GetIngressGatewayDeploymentOwnerWatch(&autoscalingv1.HorizontalPodAutoscaler{}),
		{
			Src:          &source.Kind{Type: &corev1.ConfigMap{}},
			EventHandler: enqueueRequestsForConfigmapEvents,
		},
	}
	ingressGatewayDeploymentResourceManager := ingressgatewaydeployment.NewIngressGatewayDeploymentServiceManager(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IngressGatewayDeployment")}, clientset, referenceResolver, meshCacheManager, operatorNamespace)
	if err = (&servicemeshcontrollers.ServiceMeshReconciler{
		Reconciler: &core.BaseReconciler{
			Client:               mgr.GetClient(),
			OSOKServiceManager:   ingressGatewayDeploymentResourceManager,
			Finalizer:            core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                  loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("IngressGatewayDeployment")},
			Metrics:              metricsClient,
			Recorder:             mgr.GetEventRecorderFor("IngressGatewayDeployment"),
			Scheme:               scheme,
			AdditionalFinalizers: []string{meshCommons.IngressGatewayDeploymentFinalizer},
		},
		ResourceObject: new(servicemeshapi.IngressGatewayDeployment),
		CustomWatches:  igdCustomWatches,
	}).SetupWithManagerWithMaxDelay(mgr, meshCommons.MaxDelay); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "IngressGatewayDeployment")
		os.Exit(1)
	}
	ingressGatewayDeploymentValidator := ingressgatewaydeployment.NewIngressGatewayDeploymentValidator(referenceResolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayDeployment")})
	ingressGatewayDeploymentServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(ingressGatewayDeploymentValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IngressGatewayDeployment")})
	setupLog.InfoLog("registering ingress gateway deployment to the webhook server")
	hookServer.Register("/validate-servicemesh-oci-oracle-com-v1beta1-ingressgatewaydeployment",
		&webhook.Admission{Handler: &serviceMeshManager.BaseValidator{
			Reader:            mgr.GetAPIReader(),
			Client:            mgr.GetClient(),
			Log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Validator").WithName("IngressGatewayDeployment")},
			Metrics:           metricsClient,
			Recorder:          mgr.GetEventRecorderFor("IngressGatewayDeployment"),
			ValidationManager: ingressGatewayDeploymentServiceValidationManager,
		}})

	serviceUpdateResourceHandler := serviceupdate.NewDefaultResourceHandler(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("Services")}, clientset)
	if err = customControllers.NewServiceReconciler(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("Services")},
		serviceUpdateResourceHandler,
	).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "Services")
		os.Exit(1)
	}

	injectProxyResourceHandler := injectproxy.NewDefaultResourceHandler(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("InjectProxy")}, clientset, operatorNamespace)
	if err = customControllers.NewInjectProxyReconciler(
		mgr.GetClient(),
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("InjectProxy")},
		injectProxyResourceHandler,
	).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "InjectProxy")
		os.Exit(1)
	}

	upgradeProxyResourceHandler := upgradeproxy.NewDefaultResourceHandler(mgr.GetClient(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Resource Handler").WithName("UpgradeProxy")}, clientset)
	if err = customControllers.NewUpgradeProxyReconciler(
		mgr.GetClient(),
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("UpgradeProxy")},
		upgradeProxyResourceHandler,
		operatorNamespace,
	).SetupWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create controller", "controller", "UpgradeProxy")
		os.Exit(1)
	}

	setupLog.InfoLog("registering webhooks to the webhook server")
	podMutatorHandler := meshwebhook.NewDefaultPodMutatorHandler(
		mgr.GetAPIReader(),
		meshCacheManager,
		operatorNamespace+"/"+meshCommons.MeshConfigMapName,
	)
	hookServer.Register("/mutate-pod", &webhook.Admission{Handler: podMutatorHandler})

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.InfoLog("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.ErrorLog(err, "problem running manager")
		os.Exit(1)
	}
}
