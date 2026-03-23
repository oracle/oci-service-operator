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
	"github.com/oracle/oci-service-operator/internal/registrations"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"

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

	"github.com/oracle/oci-service-operator/pkg/authhelper"
	"github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/credhelper/kubesecret"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup")}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	for _, registration := range registrations.All() {
		utilruntime.Must(registration.AddToScheme(scheme))
	}
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

	registrationContext := registrations.NewContext(mgr, servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           mgr.GetScheme(),
		Metrics:          metricsClient,
	})

	for _, registration := range registrations.All() {
		if err = registration.SetupWithManager(registrationContext); err != nil {
			setupLog.ErrorLog(err, "unable to create controller registration", "group", registration.Group)
			os.Exit(1)
		}
	}
	for _, webhook := range registrations.ManualWebhooks() {
		if err = webhook.SetupWithManager(mgr); err != nil {
			setupLog.ErrorLog(err, "unable to create webhook", "webhook", webhook.Name)
			os.Exit(1)
		}
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
