/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"flag"
	"os"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/go_ensurefips"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	databasecontrollers "github.com/oracle/oci-service-operator/controllers/database"
	mysqlcontrollers "github.com/oracle/oci-service-operator/controllers/mysql"
	streamingcontrollers "github.com/oracle/oci-service-operator/controllers/streaming"
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

	utilruntime.Must(databasev1beta1.AddToScheme(scheme))
	utilruntime.Must(mysqlv1beta1.AddToScheme(scheme))
	utilruntime.Must(streamingv1beta1.AddToScheme(scheme))
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
			Metrics:                metricsserver.Options{BindAddress: metricsAddr},
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "40558063.oci",
		}
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

	if err = (&databasecontrollers.AutonomousDatabasesReconciler{
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
	if err = (&databasev1beta1.AutonomousDatabases{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.ErrorLog(err, "unable to create webhook", "webhook", "AutonomousDatabases")
		os.Exit(1)
	}

	if err = (&streamingcontrollers.StreamReconciler{
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
	if err = (&mysqlcontrollers.MySqlDBsystemReconciler{
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
