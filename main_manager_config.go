package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	controllerconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/yaml"
)

const defaultLeaderElectionID = "40558063.oci"

const (
	expectedControllerManagerConfigAPIVersion = "controller-runtime.sigs.k8s.io/v1alpha1"
	expectedControllerManagerConfigKind       = "ControllerManagerConfig"
)

type startupFlags struct {
	configFile           string
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
	initOSOKResources    bool
	zapOptions           zap.Options
}

type controllerManagerConfigFile struct {
	APIVersion              string                    `json:"apiVersion,omitempty"`
	Kind                    string                    `json:"kind,omitempty"`
	SyncPeriod              *metav1.Duration          `json:"syncPeriod,omitempty"`
	CacheNamespace          string                    `json:"cacheNamespace,omitempty"`
	GracefulShutdownTimeout *metav1.Duration          `json:"gracefulShutDown,omitempty"`
	LeaderElection          *leaderElectionConfigFile `json:"leaderElection,omitempty"`
	Controller              *controllerConfigFile     `json:"controller,omitempty"`
	Metrics                 metricsConfigFile         `json:"metrics,omitempty"`
	Health                  healthConfigFile          `json:"health,omitempty"`
	Webhook                 webhookConfigFile         `json:"webhook,omitempty"`
}

type leaderElectionConfigFile struct {
	LeaderElect       *bool            `json:"leaderElect,omitempty"`
	LeaseDuration     *metav1.Duration `json:"leaseDuration,omitempty"`
	RenewDeadline     *metav1.Duration `json:"renewDeadline,omitempty"`
	RetryPeriod       *metav1.Duration `json:"retryPeriod,omitempty"`
	ResourceLock      string           `json:"resourceLock,omitempty"`
	ResourceName      string           `json:"resourceName,omitempty"`
	ResourceNamespace string           `json:"resourceNamespace,omitempty"`
}

type controllerConfigFile struct {
	GroupKindConcurrency map[string]int   `json:"groupKindConcurrency,omitempty"`
	CacheSyncTimeout     *metav1.Duration `json:"cacheSyncTimeout,omitempty"`
	RecoverPanic         *bool            `json:"recoverPanic,omitempty"`
}

type metricsConfigFile struct {
	BindAddress string `json:"bindAddress,omitempty"`
}

type healthConfigFile struct {
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`
	ReadinessEndpointName  string `json:"readinessEndpointName,omitempty"`
	LivenessEndpointName   string `json:"livenessEndpointName,omitempty"`
}

type webhookConfigFile struct {
	Port    *int   `json:"port,omitempty"`
	Host    string `json:"host,omitempty"`
	CertDir string `json:"certDir,omitempty"`
}

func parseStartupFlags() startupFlags {
	flags := startupFlags{
		zapOptions: zap.Options{
			Development: true,
		},
	}

	flag.StringVar(&flags.configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"When this flag is set, manager options come from this file instead of the default flag values.")
	flag.StringVar(&flags.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&flags.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&flags.enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&flags.initOSOKResources, "init-osok-resources", false,
		"Install OSOK prerequisites like CRDs and Webhooks at manager bootup")
	flags.zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&flags.zapOptions)))
	return flags
}

func resolveManagerOptions(flags startupFlags) (ctrl.Options, error) {
	if flags.configFile == "" {
		setupLog.InfoLog("Loading the configuration from the command arguments")
		return defaultManagerOptions(flags), nil
	}

	setupLog.InfoLog("Loading the configuration from the ControllerManagerConfig configMap")
	options, err := loadManagerOptionsFromFile(flags.configFile)
	if err != nil {
		return ctrl.Options{}, fmt.Errorf("unable to load the config file: %w", err)
	}
	return options, nil
}

func defaultManagerOptions(flags startupFlags) ctrl.Options {
	return ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: flags.metricsAddr},
		HealthProbeBindAddress: flags.probeAddr,
		LeaderElection:         flags.enableLeaderElection,
		LeaderElectionID:       defaultLeaderElectionID,
	}
}

func loadManagerOptionsFromFile(path string) (ctrl.Options, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ctrl.Options{}, err
	}

	cfg := controllerManagerConfigFile{}
	if err := yaml.UnmarshalStrict(content, &cfg); err != nil {
		return ctrl.Options{}, err
	}
	if err := validateControllerManagerConfigTypeMeta(cfg); err != nil {
		return ctrl.Options{}, err
	}
	return cfg.toOptions(), nil
}

func validateControllerManagerConfigTypeMeta(cfg controllerManagerConfigFile) error {
	if cfg.APIVersion != expectedControllerManagerConfigAPIVersion {
		return fmt.Errorf("controller manager config apiVersion = %q, want %q", cfg.APIVersion, expectedControllerManagerConfigAPIVersion)
	}
	if cfg.Kind != expectedControllerManagerConfigKind {
		return fmt.Errorf("controller manager config kind = %q, want %q", cfg.Kind, expectedControllerManagerConfigKind)
	}
	return nil
}

func (cfg controllerManagerConfigFile) toOptions() ctrl.Options {
	options := ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: cfg.Metrics.BindAddress},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    intValue(cfg.Webhook.Port),
			Host:    cfg.Webhook.Host,
			CertDir: cfg.Webhook.CertDir,
		}),
	}

	if cfg.SyncPeriod != nil {
		options.Cache.SyncPeriod = durationValuePtr(cfg.SyncPeriod)
	}
	if cfg.CacheNamespace != "" {
		options.Cache.DefaultNamespaces = map[string]cache.Config{
			cfg.CacheNamespace: {},
		}
	}
	if cfg.GracefulShutdownTimeout != nil {
		options.GracefulShutdownTimeout = durationValuePtr(cfg.GracefulShutdownTimeout)
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
	if cfg.Controller != nil {
		options.Controller = controllerconfig.Controller{
			GroupKindConcurrency: cfg.Controller.GroupKindConcurrency,
			RecoverPanic:         cfg.Controller.RecoverPanic,
		}
		if cfg.Controller.CacheSyncTimeout != nil {
			options.Controller.CacheSyncTimeout = cfg.Controller.CacheSyncTimeout.Duration
		}
	}
	if cfg.LeaderElection != nil {
		applyLeaderElectionConfig(&options, cfg.LeaderElection)
	}
	return options
}

func applyLeaderElectionConfig(options *ctrl.Options, cfg *leaderElectionConfigFile) {
	if cfg.LeaderElect != nil {
		options.LeaderElection = *cfg.LeaderElect
	}
	if cfg.ResourceLock != "" {
		options.LeaderElectionResourceLock = cfg.ResourceLock
	}
	if cfg.ResourceNamespace != "" {
		options.LeaderElectionNamespace = cfg.ResourceNamespace
	}
	if cfg.ResourceName != "" {
		options.LeaderElectionID = cfg.ResourceName
	}
	if cfg.LeaseDuration != nil {
		options.LeaseDuration = durationValuePtr(cfg.LeaseDuration)
	}
	if cfg.RenewDeadline != nil {
		options.RenewDeadline = durationValuePtr(cfg.RenewDeadline)
	}
	if cfg.RetryPeriod != nil {
		options.RetryPeriod = durationValuePtr(cfg.RetryPeriod)
	}
}

func durationValuePtr(duration *metav1.Duration) *time.Duration {
	if duration == nil {
		return nil
	}
	value := duration.Duration
	return &value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
