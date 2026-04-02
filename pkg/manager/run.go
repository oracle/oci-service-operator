package manager

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/go_ensurefips"
	"github.com/oracle/oci-service-operator/pkg/authhelper"
	"github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/credhelper/kubesecret"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/yaml"
)

// RegisterFunc wires controllers, webhooks, and supporting components into the shared manager instance.
type RegisterFunc func(ctrl.Manager, *Dependencies) error

// Dependencies bundles common clients initialised by Run that individual services can reuse.
type Dependencies struct {
	Provider   common.ConfigurationProvider
	CredClient *kubesecret.KubeSecretClient
	Metrics    *metrics.Metrics
	Scheme     *runtime.Scheme
}

// Options configure shared manager behaviour.
type Options struct {
	// Scheme is the runtime scheme populated by the caller with the APIs served by this binary.
	Scheme *runtime.Scheme
	// MetricsServiceName is used when initialising the metrics collector for this manager.
	MetricsServiceName string
	// LeaderElectionID identifies the leader election record used by this manager.
	LeaderElectionID string
	// SkipFIPS disables FIPS checks when running without a BoringCrypto-enabled binary.
	SkipFIPS bool
}

const (
	defaultLeaderElectionID  = "40558063.oci"
	defaultMetricsService    = "osok"
	expectedConfigAPIVersion = "controller-runtime.sigs.k8s.io/v1alpha1"
	expectedConfigKind       = "ControllerManagerConfiguration"
	skipFIPSEnvVar          = "OSOK_SKIP_FIPS"
)

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

// Run bootstraps the shared controller-runtime manager and delegates controller registration to the supplied hooks.
func Run(opts Options, registrars ...RegisterFunc) error {
	if opts.Scheme == nil {
		return fmt.Errorf("manager: scheme must be provided")
	}
	if opts.LeaderElectionID == "" {
		opts.LeaderElectionID = defaultLeaderElectionID
	}
	if opts.MetricsServiceName == "" {
		opts.MetricsServiceName = defaultMetricsService
	}

	skipFIPS := opts.SkipFIPS
	if raw, ok := os.LookupEnv(skipFIPSEnvVar); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("manager: parse %s: %w", skipFIPSEnvVar, err)
		}
		skipFIPS = parsed
	}

	if !skipFIPS {
		go_ensurefips.Compliant()
	}
	common.EnableInstanceMetadataServiceLookup()

	var (
		configFile           string
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		initOSOKResources    bool
	)

	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&initOSOKResources, "init-osok-resources", false,
		"Install OSOK prerequisites like CRDs and Webhooks at manager bootup")

	zapOpts := zap.Options{
		Development: true,
	}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	setupLog := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup")}

	var (
		err     error
		options ctrl.Options
	)

	options = ctrl.Options{
		Scheme:                 opts.Scheme,
		Metrics:                server.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       opts.LeaderElectionID,
	}

	if configFile != "" {
		setupLog.InfoLog("Loading the configuration from the ControllerManagerConfiguration configMap")
		options, err = loadManagerOptionsFromFile(configFile, options)
		if err != nil {
			setupLog.ErrorLog(err, "unable to load the config file")
			return err
		}
	} else {
		setupLog.InfoLog("Loading the configuration from the command arguments")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.ErrorLog(err, "unable to start manager")
		return err
	}

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
		setupLog.ErrorLog(err, "unable to get the oci configuration provider. Exiting setup")
		return err
	}

	metricsClient := metrics.Init(opts.MetricsServiceName, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})

	credClient := &kubesecret.KubeSecretClient{
		Client:  mgr.GetClient(),
		Log:     loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient")},
		Metrics: metricsClient,
	}

	deps := &Dependencies{
		Provider:   provider,
		CredClient: credClient,
		Metrics:    metricsClient,
		Scheme:     opts.Scheme,
	}

	for _, register := range registrars {
		if register == nil {
			continue
		}
		if err := register(mgr, deps); err != nil {
			setupLog.ErrorLog(err, "unable to register controller")
			return err
		}
	}

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.ErrorLog(err, "unable to set up ready check")
		return err
	}

	setupLog.InfoLog("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.ErrorLog(err, "problem running manager")
		return err
	}
	return nil
}

func loadManagerOptionsFromFile(path string, options ctrl.Options) (ctrl.Options, error) {
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

func validateControllerManagerConfigTypeMeta(cfg controllerManagerConfigFile) error {
	if cfg.APIVersion != expectedConfigAPIVersion {
		return fmt.Errorf("controller manager config apiVersion = %q, want %q", cfg.APIVersion, expectedConfigAPIVersion)
	}
	if cfg.Kind != expectedConfigKind {
		return fmt.Errorf("controller manager config kind = %q, want %q", cfg.Kind, expectedConfigKind)
	}
	return nil
}

func applyManagerConfigOverlay(options ctrl.Options, cfg controllerManagerConfigFile) ctrl.Options {
	if options.Cache.SyncPeriod == nil && cfg.SyncPeriod != nil {
		options.Cache.SyncPeriod = durationValuePtr(cfg.SyncPeriod)
	}
	if len(options.Cache.DefaultNamespaces) == 0 && cfg.CacheNamespace != "" {
		options.Cache.DefaultNamespaces = map[string]cache.Config{cfg.CacheNamespace: {}}
	}
	if options.GracefulShutdownTimeout == nil && cfg.GracefulShutdownTimeout != nil {
		options.GracefulShutdownTimeout = durationValuePtr(cfg.GracefulShutdownTimeout)
	}
	if options.Metrics.BindAddress == "" && cfg.Metrics.BindAddress != "" {
		options.Metrics.BindAddress = cfg.Metrics.BindAddress
	}
	if options.HealthProbeBindAddress == "" && cfg.Health.HealthProbeBindAddress != "" {
		options.HealthProbeBindAddress = cfg.Health.HealthProbeBindAddress
	}
	if options.ReadinessEndpointName == "" && cfg.Health.ReadinessEndpointName != "" {
		options.ReadinessEndpointName = cfg.Health.ReadinessEndpointName
	}
	if options.LivenessEndpointName == "" && cfg.Health.LivenessEndpointName != "" {
		options.LivenessEndpointName = cfg.Health.LivenessEndpointName
	}
	if options.WebhookServer == nil && (cfg.Webhook.Port != nil || cfg.Webhook.Host != "" || cfg.Webhook.CertDir != "") {
		options.WebhookServer = webhook.NewServer(webhook.Options{
			Port:    intValue(cfg.Webhook.Port),
			Host:    cfg.Webhook.Host,
			CertDir: cfg.Webhook.CertDir,
		})
	}
	if cfg.Controller != nil {
		if len(options.Controller.GroupKindConcurrency) == 0 {
			options.Controller.GroupKindConcurrency = cfg.Controller.GroupKindConcurrency
		}
		if options.Controller.CacheSyncTimeout == 0 && cfg.Controller.CacheSyncTimeout != nil {
			options.Controller.CacheSyncTimeout = cfg.Controller.CacheSyncTimeout.Duration
		}
		if options.Controller.RecoverPanic == nil && cfg.Controller.RecoverPanic != nil {
			options.Controller.RecoverPanic = cfg.Controller.RecoverPanic
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

func intValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func durationValuePtr(v *metav1.Duration) *time.Duration {
	if v == nil {
		return nil
	}
	duration := v.Duration
	return &duration
}
