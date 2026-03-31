/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/registrations"
	"github.com/oracle/oci-service-operator/pkg/authhelper"
	"github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/credhelper/kubesecret"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/yaml"
)

const leaderElectionID = "40558063.oci"

type controllerManagerConfigFile struct {
	APIVersion              string                                 `json:"apiVersion,omitempty"`
	Kind                    string                                 `json:"kind,omitempty"`
	SyncPeriod              *metav1.Duration                       `json:"syncPeriod,omitempty"`
	CacheNamespace          string                                 `json:"cacheNamespace,omitempty"`
	GracefulShutdownTimeout *metav1.Duration                       `json:"gracefulShutDown,omitempty"`
	Controller              *controllerManagerControllerConfig     `json:"controller,omitempty"`
	Metrics                 controllerManagerMetricsConfig         `json:"metrics,omitempty"`
	Health                  controllerManagerHealthConfig          `json:"health,omitempty"`
	Webhook                 controllerManagerWebhookConfig         `json:"webhook,omitempty"`
	LeaderElection          *controllerManagerLeaderElectionConfig `json:"leaderElection,omitempty"`
}

type controllerManagerControllerConfig struct {
	GroupKindConcurrency map[string]int   `json:"groupKindConcurrency,omitempty"`
	CacheSyncTimeout     *metav1.Duration `json:"cacheSyncTimeout,omitempty"`
	RecoverPanic         *bool            `json:"recoverPanic,omitempty"`
}

type controllerManagerMetricsConfig struct {
	BindAddress string `json:"bindAddress,omitempty"`
}

type controllerManagerHealthConfig struct {
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`
	ReadinessEndpointName  string `json:"readinessEndpointName,omitempty"`
	LivenessEndpointName   string `json:"livenessEndpointName,omitempty"`
}

type controllerManagerWebhookConfig struct {
	Port    *int   `json:"port,omitempty"`
	Host    string `json:"host,omitempty"`
	CertDir string `json:"certDir,omitempty"`
}

type controllerManagerLeaderElectionConfig struct {
	LeaderElect       *bool            `json:"leaderElect,omitempty"`
	LeaseDuration     *metav1.Duration `json:"leaseDuration,omitempty"`
	RenewDeadline     *metav1.Duration `json:"renewDeadline,omitempty"`
	RetryPeriod       *metav1.Duration `json:"retryPeriod,omitempty"`
	ResourceLock      string           `json:"resourceLock,omitempty"`
	ResourceName      string           `json:"resourceName,omitempty"`
	ResourceNamespace string           `json:"resourceNamespace,omitempty"`
}

func mustManagerOptions(configFile string, metricsAddr string, probeAddr string, enableLeaderElection bool) ctrl.Options {
	options, err := managerOptionsFromInputs(configFile, metricsAddr, probeAddr, enableLeaderElection)
	if err != nil {
		setupLog.ErrorLog(err, "unable to load the config file")
		os.Exit(1)
	}
	return options
}

func managerOptionsFromInputs(configFile string, metricsAddr string, probeAddr string, enableLeaderElection bool) (ctrl.Options, error) {
	if configFile == "" {
		setupLog.InfoLog("Loading the configuration from the command arguments")
		return defaultManagerOptions(metricsAddr, probeAddr, enableLeaderElection), nil
	}

	setupLog.InfoLog("Loading the configuration from the ControllerManagerConfig configMap")
	return loadManagerOptionsFromFile(configFile)
}

func defaultManagerOptions(metricsAddr string, probeAddr string, enableLeaderElection bool) ctrl.Options {
	return ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
	}
}

func loadManagerOptionsFromFile(configFile string) (ctrl.Options, error) {
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return ctrl.Options{}, fmt.Errorf("read manager config file %q: %w", configFile, err)
	}

	var cfg controllerManagerConfigFile
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return ctrl.Options{}, fmt.Errorf("decode manager config file %q: %w", configFile, err)
	}

	return managerOptionsFromConfig(cfg), nil
}

func managerOptionsFromConfig(cfg controllerManagerConfigFile) ctrl.Options {
	options := ctrl.Options{Scheme: scheme}

	if cfg.SyncPeriod != nil {
		options.Cache.SyncPeriod = &cfg.SyncPeriod.Duration
	}
	if cfg.CacheNamespace != "" {
		options.Cache.DefaultNamespaces = map[string]cache.Config{
			cfg.CacheNamespace: {},
		}
	}
	if cfg.Metrics.BindAddress != "" {
		options.Metrics = metricsserver.Options{BindAddress: cfg.Metrics.BindAddress}
	}
	if cfg.Health.HealthProbeBindAddress != "" {
		options.HealthProbeBindAddress = cfg.Health.HealthProbeBindAddress
	}
	if cfg.Health.ReadinessEndpointName != "" {
		options.ReadinessEndpointName = cfg.Health.ReadinessEndpointName
	}
	if cfg.Health.LivenessEndpointName != "" {
		options.LivenessEndpointName = cfg.Health.LivenessEndpointName
	}
	if hasWebhookServerConfig(cfg.Webhook) {
		options.WebhookServer = webhook.NewServer(webhook.Options{
			Host:    cfg.Webhook.Host,
			CertDir: cfg.Webhook.CertDir,
			Port:    webhookPort(cfg.Webhook.Port),
		})
	}
	applyLeaderElectionConfig(&options, cfg.LeaderElection)
	applyControllerConfig(&options, cfg.Controller)
	if cfg.GracefulShutdownTimeout != nil {
		options.GracefulShutdownTimeout = &cfg.GracefulShutdownTimeout.Duration
	}

	return options
}

func hasWebhookServerConfig(cfg controllerManagerWebhookConfig) bool {
	return cfg.Port != nil || cfg.Host != "" || cfg.CertDir != ""
}

func webhookPort(port *int) int {
	if port == nil {
		return 0
	}
	return *port
}

func applyLeaderElectionConfig(options *ctrl.Options, cfg *controllerManagerLeaderElectionConfig) {
	if cfg == nil {
		return
	}
	if cfg.LeaderElect != nil {
		options.LeaderElection = *cfg.LeaderElect
	}
	if cfg.ResourceLock != "" {
		options.LeaderElectionResourceLock = cfg.ResourceLock
	}
	if cfg.ResourceName != "" {
		options.LeaderElectionID = cfg.ResourceName
	}
	if cfg.ResourceNamespace != "" {
		options.LeaderElectionNamespace = cfg.ResourceNamespace
	}
	if cfg.LeaseDuration != nil {
		options.LeaseDuration = &cfg.LeaseDuration.Duration
	}
	if cfg.RenewDeadline != nil {
		options.RenewDeadline = &cfg.RenewDeadline.Duration
	}
	if cfg.RetryPeriod != nil {
		options.RetryPeriod = &cfg.RetryPeriod.Duration
	}
}

func applyControllerConfig(options *ctrl.Options, cfg *controllerManagerControllerConfig) {
	if cfg == nil {
		return
	}
	if cfg.CacheSyncTimeout != nil {
		options.Controller.CacheSyncTimeout = cfg.CacheSyncTimeout.Duration
	}
	if len(cfg.GroupKindConcurrency) > 0 {
		options.Controller.GroupKindConcurrency = cfg.GroupKindConcurrency
	}
	if cfg.RecoverPanic != nil {
		options.Controller.RecoverPanic = cfg.RecoverPanic
	}
}

func mustNewManager(options ctrl.Options) ctrl.Manager {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.ErrorLog(err, "unable to start manager")
		os.Exit(1)
	}
	return mgr
}

func maybeInitOSOKResources(mgr ctrl.Manager, initOSOKResources bool) {
	if !initOSOKResources {
		return
	}
	util.InitOSOK(mgr.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK")})
}

func mustSetupRegistrations(mgr ctrl.Manager) {
	setupLog.InfoLog("Getting the config details")
	osokCfg := config.GetConfigDetails(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")})

	authConfigProvider := &authhelper.AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")},
	}
	provider, err := authConfigProvider.GetAuthProvider(osokCfg)
	if err != nil {
		setupLog.ErrorLog(err, "unable to get the oci configuration provider. Exiting setup")
		os.Exit(1)
	}

	metricsClient := metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})
	credClient := kubesecret.NewWithReader(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient")},
		metricsClient,
	)

	registrationContext := registrations.NewContext(mgr, servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           mgr.GetScheme(),
		Metrics:          metricsClient,
	})

	for _, registration := range registrations.All() {
		if err := registration.SetupWithManager(registrationContext); err != nil {
			setupLog.ErrorLog(err, "unable to create controller registration", "group", registration.Group)
			os.Exit(1)
		}
	}
}

func mustSetupManualWebhooks(mgr ctrl.Manager) {
	for _, webhookRegistration := range registrations.ManualWebhooks() {
		if err := webhookRegistration.SetupWithManager(mgr); err != nil {
			setupLog.ErrorLog(err, "unable to create webhook", "webhook", webhookRegistration.Name)
			os.Exit(1)
		}
	}
}

func mustAddHealthChecks(mgr ctrl.Manager) {
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up ready check")
		os.Exit(1)
	}
}

func mustStartManager(mgr ctrl.Manager) {
	setupLog.InfoLog("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.ErrorLog(err, "problem running manager")
		os.Exit(1)
	}
}
