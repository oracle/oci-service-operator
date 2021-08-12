/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"flag"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
	"os"

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
	setupLog = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup"), FixedLogs: make(map[string]string)}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ociv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
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
			MetricsBindAddress:     metricsAddr,
			Port:                   9443,
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
		util.InitOSOK(mgr.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK"), FixedLogs: make(map[string]string)})
	}

	setupLog.InfoLog("Getting the config details")
	osokCfg := config.GetConfigDetails(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config"), FixedLogs: make(map[string]string)})

	authConfigProvider := &authhelper.AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config"), FixedLogs: make(map[string]string)}}

	provider, err := authConfigProvider.GetAuthProvider(osokCfg)
	if err != nil {
		setupLog.ErrorLog(err, "unable to get the oci configuration provider. Exiting setup")
		os.Exit(1)
	}

	metricsClient := metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics"), FixedLogs: make(map[string]string)})

	credClient := &kubesecret.KubeSecretClient{
		Client:  mgr.GetClient(),
		Log:     loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient"), FixedLogs: make(map[string]string)},
		Metrics: metricsClient,
	}

	if err = (&controllers.AutonomousDatabasesReconciler{
		Reconciler: &core.BaseReconciler{
			Client:             mgr.GetClient(),
			OSOKServiceManager: adb.NewAdbServiceManager(provider, credClient, scheme, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AutonomousDatabases"), FixedLogs: make(map[string]string)}),
			Finalizer:          core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("AutonomousDatabases"), FixedLogs: make(map[string]string)},
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
			OSOKServiceManager: streams.NewStreamServiceManager(provider, credClient, scheme, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Streams"), FixedLogs: make(map[string]string)},
				metricsClient),
			Finalizer: core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("Streams"), FixedLogs: make(map[string]string)},
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
			OSOKServiceManager: dbsystem.NewDbSystemServiceManager(provider, credClient, scheme, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("MySqlDbSystem"), FixedLogs: make(map[string]string)}),
			Finalizer:          core.NewBaseFinalizer(mgr.GetClient(), ctrl.Log),
			Log:                loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("MySqlDbSystem"), FixedLogs: make(map[string]string)},
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
