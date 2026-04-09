/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"fmt"
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
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/yaml"

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

type managerFlags = startupFlags

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	for _, registration := range registrations.All() {
		utilruntime.Must(registration.AddToScheme(scheme))
	}
	// +kubebuilder:scaffold:scheme
}

func main() {
	if err := run(); err != nil {
		setupLog.ErrorLog(err, "manager exited")
		os.Exit(1)
	}
}

func run() error {
	go_ensurefips.Compliant()
	common.EnableInstanceMetadataServiceLookup()

	startup := parseStartupFlags()

	options, err := resolveManagerOptions(startup)
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	if err := setupRegistrations(mgr, startup.initOSOKResources); err != nil {
		return err
	}

	// +kubebuilder:scaffold:builder

	if err := addManagerHealthChecks(mgr); err != nil {
		return err
	}

	setupLog.InfoLog("starting manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}

func managerOptions(flags managerFlags) (ctrl.Options, error) {
	return resolveManagerOptions(flags)
}

func loadManagerOptionsFromConfig(path string, options ctrl.Options) (ctrl.Options, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return options, err
	}

	cfg := controllerManagerConfigFile{}
	if err := yaml.UnmarshalStrict(content, &cfg); err != nil {
		return options, err
	}
	if err := validateControllerManagerConfigTypeMeta(cfg); err != nil {
		return options, err
	}
	return applyManagerConfigOverlay(options, cfg), nil
}

func applyManagerConfigOverlay(options ctrl.Options, cfg controllerManagerConfigFile) ctrl.Options {
	options = applyLeaderElectionConfigOverlay(options, cfg.LeaderElection)
	options = applyGeneralManagerConfigOverlay(options, cfg)
	options = applyHealthConfigOverlay(options, cfg.Health)
	options = applyWebhookConfigOverlay(options, cfg.Webhook)
	if cfg.Controller != nil {
		options = applyControllerConfigOverlay(options, *cfg.Controller)
	}
	return options
}

func applyLeaderElectionConfigOverlay(options ctrl.Options, cfg *leaderElectionConfigFile) ctrl.Options {
	if cfg == nil {
		return options
	}
	options = applyLeaderElectionIdentityConfigOverlay(options, cfg)
	options = applyLeaderElectionDurationConfigOverlay(options, cfg)
	return options
}

func applyLeaderElectionIdentityConfigOverlay(options ctrl.Options, cfg *leaderElectionConfigFile) ctrl.Options {
	if !options.LeaderElection && cfg.LeaderElect != nil {
		options.LeaderElection = *cfg.LeaderElect
	}
	if options.LeaderElectionResourceLock == "" && cfg.ResourceLock != "" {
		options.LeaderElectionResourceLock = cfg.ResourceLock
	}
	if options.LeaderElectionNamespace == "" && cfg.ResourceNamespace != "" {
		options.LeaderElectionNamespace = cfg.ResourceNamespace
	}
	if options.LeaderElectionID == "" && cfg.ResourceName != "" {
		options.LeaderElectionID = cfg.ResourceName
	}
	return options
}

func applyLeaderElectionDurationConfigOverlay(options ctrl.Options, cfg *leaderElectionConfigFile) ctrl.Options {
	if options.LeaseDuration == nil && cfg.LeaseDuration != nil {
		leaseDuration := cfg.LeaseDuration.Duration
		options.LeaseDuration = &leaseDuration
	}
	if options.RenewDeadline == nil && cfg.RenewDeadline != nil {
		renewDeadline := cfg.RenewDeadline.Duration
		options.RenewDeadline = &renewDeadline
	}
	if options.RetryPeriod == nil && cfg.RetryPeriod != nil {
		retryPeriod := cfg.RetryPeriod.Duration
		options.RetryPeriod = &retryPeriod
	}
	return options
}

func applyGeneralManagerConfigOverlay(options ctrl.Options, cfg controllerManagerConfigFile) ctrl.Options {
	if options.Cache.SyncPeriod == nil && cfg.SyncPeriod != nil {
		syncPeriod := cfg.SyncPeriod.Duration
		options.Cache.SyncPeriod = &syncPeriod
	}
	if len(options.Cache.DefaultNamespaces) == 0 && cfg.CacheNamespace != "" {
		options.Cache.DefaultNamespaces = map[string]cache.Config{cfg.CacheNamespace: {}}
	}
	if options.GracefulShutdownTimeout == nil && cfg.GracefulShutdownTimeout != nil {
		timeout := cfg.GracefulShutdownTimeout.Duration
		options.GracefulShutdownTimeout = &timeout
	}
	if options.Metrics.BindAddress == "" && cfg.Metrics.BindAddress != "" {
		options.Metrics.BindAddress = cfg.Metrics.BindAddress
	}
	return options
}

func applyHealthConfigOverlay(options ctrl.Options, cfg healthConfigFile) ctrl.Options {
	if options.HealthProbeBindAddress == "" && cfg.HealthProbeBindAddress != "" {
		options.HealthProbeBindAddress = cfg.HealthProbeBindAddress
	}
	if options.ReadinessEndpointName == "" && cfg.ReadinessEndpointName != "" {
		options.ReadinessEndpointName = cfg.ReadinessEndpointName
	}
	if options.LivenessEndpointName == "" && cfg.LivenessEndpointName != "" {
		options.LivenessEndpointName = cfg.LivenessEndpointName
	}
	return options
}

func applyWebhookConfigOverlay(options ctrl.Options, cfg webhookConfigFile) ctrl.Options {
	if options.WebhookServer != nil {
		return options
	}

	port := 0
	if cfg.Port != nil {
		port = *cfg.Port
	}
	options.WebhookServer = webhook.NewServer(webhook.Options{
		Port:    port,
		Host:    cfg.Host,
		CertDir: cfg.CertDir,
	})
	return options
}

func applyControllerConfigOverlay(options ctrl.Options, cfg controllerConfigFile) ctrl.Options {
	if options.Controller.CacheSyncTimeout == 0 && cfg.CacheSyncTimeout != nil {
		options.Controller.CacheSyncTimeout = cfg.CacheSyncTimeout.Duration
	}
	if len(options.Controller.GroupKindConcurrency) == 0 && len(cfg.GroupKindConcurrency) > 0 {
		options.Controller.GroupKindConcurrency = cfg.GroupKindConcurrency
	}
	if options.Controller.RecoverPanic == nil && cfg.RecoverPanic != nil {
		options.Controller.RecoverPanic = cfg.RecoverPanic
	}
	return options
}

type credentialClientManager interface {
	GetClient() ctrlclient.Client
	GetAPIReader() ctrlclient.Reader
}

func newCredentialClient(mgr credentialClientManager, metricsClient *metrics.Metrics) *kubesecret.KubeSecretClient {
	return kubesecret.NewWithReader(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient")},
		metricsClient,
	)
}

func setupRegistrations(mgr ctrl.Manager, initOSOKResources bool) error {
	if initOSOKResources {
		util.InitOSOK(mgr.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK")})
	}

	runtimeDeps, err := buildRuntimeDeps(mgr)
	if err != nil {
		return err
	}

	registrationContext := registrations.NewContext(mgr, runtimeDeps)
	for _, registration := range registrations.All() {
		if err := registration.SetupWithManager(registrationContext); err != nil {
			return err
		}
	}
	for _, webhook := range registrations.ManualWebhooks() {
		if err := webhook.SetupWithManager(mgr); err != nil {
			return err
		}
	}
	return nil
}

func buildRuntimeDeps(mgr ctrl.Manager) (servicemanager.RuntimeDeps, error) {
	setupLog.InfoLog("Getting the config details")
	configLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")}
	osokCfg := config.GetConfigDetails(configLogger)

	authConfigProvider := &authhelper.AuthConfigProvider{Log: configLogger}
	provider, err := authConfigProvider.GetAuthProvider(osokCfg)
	if err != nil {
		return servicemanager.RuntimeDeps{}, fmt.Errorf("unable to get the oci configuration provider: %w", err)
	}

	metricsClient := metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})
	credClient := newCredentialClient(mgr, metricsClient)

	return servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           mgr.GetScheme(),
		Metrics:          metricsClient,
	}, nil
}

type managerHealthChecker interface {
	AddHealthzCheck(name string, check healthz.Checker) error
	AddReadyzCheck(name string, check healthz.Checker) error
}

func addManagerHealthChecks(mgr managerHealthChecker) error {
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return err
	}
	return mgr.AddReadyzCheck("check", healthz.Ping)
}
