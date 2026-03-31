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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
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

const defaultLeaderElectionID = "40558063.oci"

type managerFlags struct {
	configFile           string
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
	initOSOKResources    bool
}

type managerConfigFile struct {
	metav1.TypeMeta         `json:",inline"`
	SyncPeriod              *metav1.Duration       `json:"syncPeriod,omitempty"`
	LeaderElection          *managerLeaderElection `json:"leaderElection,omitempty"`
	CacheNamespace          string                 `json:"cacheNamespace,omitempty"`
	GracefulShutdownTimeout *metav1.Duration       `json:"gracefulShutDown,omitempty"`
	Controller              *managerController     `json:"controller,omitempty"`
	Metrics                 managerMetrics         `json:"metrics,omitempty"`
	Health                  managerHealth          `json:"health,omitempty"`
	Webhook                 managerWebhook         `json:"webhook,omitempty"`
}

type managerLeaderElection struct {
	LeaderElect       *bool           `json:"leaderElect,omitempty"`
	ResourceLock      string          `json:"resourceLock,omitempty"`
	ResourceNamespace string          `json:"resourceNamespace,omitempty"`
	ResourceName      string          `json:"resourceName,omitempty"`
	LeaseDuration     metav1.Duration `json:"leaseDuration,omitempty"`
	RenewDeadline     metav1.Duration `json:"renewDeadline,omitempty"`
	RetryPeriod       metav1.Duration `json:"retryPeriod,omitempty"`
}

type managerController struct {
	GroupKindConcurrency map[string]int   `json:"groupKindConcurrency,omitempty"`
	CacheSyncTimeout     *metav1.Duration `json:"cacheSyncTimeout,omitempty"`
	RecoverPanic         *bool            `json:"recoverPanic,omitempty"`
}

type managerMetrics struct {
	BindAddress string `json:"bindAddress,omitempty"`
}

type managerHealth struct {
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`
	ReadinessEndpointName  string `json:"readinessEndpointName,omitempty"`
	LivenessEndpointName   string `json:"livenessEndpointName,omitempty"`
}

type managerWebhook struct {
	Port    *int   `json:"port,omitempty"`
	Host    string `json:"host,omitempty"`
	CertDir string `json:"certDir,omitempty"`
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	for _, registration := range registrations.All() {
		utilruntime.Must(registration.AddToScheme(scheme))
	}
	// +kubebuilder:scaffold:scheme
}

func main() {
	go_ensurefips.Compliant()
	common.EnableInstanceMetadataServiceLookup()

	flags, zapOptions := parseManagerFlags()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	mgr, err := buildManager(flags)
	exitOnSetupError(err, "unable to start manager")

	exitOnSetupError(setupRegistrations(mgr, flags.initOSOKResources), "unable to configure manager dependencies")
	// +kubebuilder:scaffold:builder

	exitOnSetupError(addManagerHealthChecks(mgr), "unable to add manager health checks")

	setupLog.InfoLog("starting manager")
	exitOnSetupError(mgr.Start(ctrl.SetupSignalHandler()), "problem running manager")
}

func parseManagerFlags() (managerFlags, zap.Options) {
	flags := managerFlags{}
	flag.StringVar(&flags.configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.StringVar(&flags.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&flags.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&flags.enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&flags.initOSOKResources, "init-osok-resources", false,
		"Install OSOK prerequisites like CRDs and supporting manifests at manager bootup")

	zapOptions := zap.Options{Development: true}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()
	return flags, zapOptions
}

func buildManager(flags managerFlags) (ctrl.Manager, error) {
	options, err := managerOptions(flags)
	if err != nil {
		return nil, err
	}
	return ctrl.NewManager(ctrl.GetConfigOrDie(), options)
}

func managerOptions(flags managerFlags) (ctrl.Options, error) {
	if flags.configFile == "" {
		setupLog.InfoLog("Loading the configuration from the command arguments")
		return ctrl.Options{
			Scheme:                 scheme,
			Metrics:                metricsserver.Options{BindAddress: flags.metricsAddr},
			HealthProbeBindAddress: flags.probeAddr,
			LeaderElection:         flags.enableLeaderElection,
			LeaderElectionID:       defaultLeaderElectionID,
		}, nil
	}

	setupLog.InfoLog("Loading the configuration from the ControllerManagerConfig configMap")
	return loadManagerOptionsFromConfig(flags.configFile, ctrl.Options{Scheme: scheme})
}

func loadManagerOptionsFromConfig(path string, options ctrl.Options) (ctrl.Options, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return options, err
	}

	var configFile managerConfigFile
	if err := yaml.UnmarshalStrict(content, &configFile); err != nil {
		return options, err
	}
	return applyManagerConfig(options, configFile), nil
}

func applyManagerConfig(options ctrl.Options, configFile managerConfigFile) ctrl.Options {
	options = applyLeaderElectionConfig(options, configFile.LeaderElection)
	options = applyGeneralManagerConfig(options, configFile)
	options = applyHealthConfig(options, configFile.Health)
	options = applyWebhookConfig(options, configFile.Webhook)
	if configFile.Controller != nil {
		options = applyControllerConfig(options, *configFile.Controller)
	}
	return options
}

func applyLeaderElectionConfig(options ctrl.Options, leaderElection *managerLeaderElection) ctrl.Options {
	if leaderElection == nil {
		return options
	}
	options = applyLeaderElectionBool(options, leaderElection)
	options = applyLeaderElectionStrings(options, leaderElection)
	return applyLeaderElectionDurations(options, leaderElection)
}

func applyGeneralManagerConfig(options ctrl.Options, configFile managerConfigFile) ctrl.Options {
	if options.Cache.SyncPeriod == nil && configFile.SyncPeriod != nil {
		syncPeriod := configFile.SyncPeriod.Duration
		options.Cache.SyncPeriod = &syncPeriod
	}
	if len(options.Cache.DefaultNamespaces) == 0 && configFile.CacheNamespace != "" {
		options.Cache.DefaultNamespaces = map[string]cache.Config{configFile.CacheNamespace: {}}
	}
	if options.GracefulShutdownTimeout == nil && configFile.GracefulShutdownTimeout != nil {
		timeout := configFile.GracefulShutdownTimeout.Duration
		options.GracefulShutdownTimeout = &timeout
	}
	if options.Metrics.BindAddress == "" && configFile.Metrics.BindAddress != "" {
		options.Metrics.BindAddress = configFile.Metrics.BindAddress
	}
	return options
}

func applyHealthConfig(options ctrl.Options, healthConfig managerHealth) ctrl.Options {
	if options.HealthProbeBindAddress == "" && healthConfig.HealthProbeBindAddress != "" {
		options.HealthProbeBindAddress = healthConfig.HealthProbeBindAddress
	}
	if options.ReadinessEndpointName == "" && healthConfig.ReadinessEndpointName != "" {
		options.ReadinessEndpointName = healthConfig.ReadinessEndpointName
	}
	if options.LivenessEndpointName == "" && healthConfig.LivenessEndpointName != "" {
		options.LivenessEndpointName = healthConfig.LivenessEndpointName
	}
	return options
}

func applyWebhookConfig(options ctrl.Options, webhookConfig managerWebhook) ctrl.Options {
	if options.WebhookServer != nil {
		return options
	}
	port := 0
	if webhookConfig.Port != nil {
		port = *webhookConfig.Port
	}
	options.WebhookServer = webhook.NewServer(webhook.Options{
		Port:    port,
		Host:    webhookConfig.Host,
		CertDir: webhookConfig.CertDir,
	})
	return options
}

func applyLeaderElectionBool(options ctrl.Options, leaderElection *managerLeaderElection) ctrl.Options {
	if !options.LeaderElection && leaderElection.LeaderElect != nil {
		options.LeaderElection = *leaderElection.LeaderElect
	}
	return options
}

func applyLeaderElectionStrings(options ctrl.Options, leaderElection *managerLeaderElection) ctrl.Options {
	if options.LeaderElectionResourceLock == "" && leaderElection.ResourceLock != "" {
		options.LeaderElectionResourceLock = leaderElection.ResourceLock
	}
	if options.LeaderElectionNamespace == "" && leaderElection.ResourceNamespace != "" {
		options.LeaderElectionNamespace = leaderElection.ResourceNamespace
	}
	if options.LeaderElectionID == "" && leaderElection.ResourceName != "" {
		options.LeaderElectionID = leaderElection.ResourceName
	}
	return options
}

func applyLeaderElectionDurations(options ctrl.Options, leaderElection *managerLeaderElection) ctrl.Options {
	if options.LeaseDuration == nil && leaderElection.LeaseDuration.Duration != 0 {
		leaseDuration := leaderElection.LeaseDuration.Duration
		options.LeaseDuration = &leaseDuration
	}
	if options.RenewDeadline == nil && leaderElection.RenewDeadline.Duration != 0 {
		renewDeadline := leaderElection.RenewDeadline.Duration
		options.RenewDeadline = &renewDeadline
	}
	if options.RetryPeriod == nil && leaderElection.RetryPeriod.Duration != 0 {
		retryPeriod := leaderElection.RetryPeriod.Duration
		options.RetryPeriod = &retryPeriod
	}
	return options
}

func applyControllerConfig(options ctrl.Options, controllerConfig managerController) ctrl.Options {
	if options.Controller.CacheSyncTimeout == 0 && controllerConfig.CacheSyncTimeout != nil {
		options.Controller.CacheSyncTimeout = controllerConfig.CacheSyncTimeout.Duration
	}
	if len(options.Controller.GroupKindConcurrency) == 0 && len(controllerConfig.GroupKindConcurrency) > 0 {
		options.Controller.GroupKindConcurrency = controllerConfig.GroupKindConcurrency
	}
	if options.Controller.RecoverPanic == nil && controllerConfig.RecoverPanic != nil {
		options.Controller.RecoverPanic = controllerConfig.RecoverPanic
	}
	return options
}

func setupRegistrations(mgr ctrl.Manager, initOSOKResources bool) error {
	if initOSOKResources {
		util.InitOSOK(mgr.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK")})
	}

	setupLog.InfoLog("Getting the config details")
	osokCfg := config.GetConfigDetails(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")})

	authConfigProvider := &authhelper.AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")},
	}

	provider, err := authConfigProvider.GetAuthProvider(osokCfg)
	if err != nil {
		return err
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
		if err := registration.SetupWithManager(registrationContext); err != nil {
			return err
		}
	}
	return nil
}

func addManagerHealthChecks(mgr ctrl.Manager) error {
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return err
	}
	return mgr.AddReadyzCheck("check", healthz.Ping)
}

func exitOnSetupError(err error, message string, keysAndValues ...interface{}) {
	if err == nil {
		return
	}
	setupLog.ErrorLog(err, message, keysAndValues...)
	os.Exit(1)
}
