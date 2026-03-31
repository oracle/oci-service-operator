package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const sampleManagerConfigYAML = `apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
syncPeriod: 30m
cacheNamespace: tenant-system
gracefulShutDown: 45s
leaderElection:
  leaderElect: true
  resourceLock: configmapsleases
  resourceNamespace: osok-system
  resourceName: runtime.oci
  leaseDuration: 20s
  renewDeadline: 15s
  retryPeriod: 5s
controller:
  groupKindConcurrency:
    AutonomousDatabase.database.oracle.com: 3
  cacheSyncTimeout: 90s
metrics:
  bindAddress: 127.0.0.1:9090
health:
  healthProbeBindAddress: :9091
  readinessEndpointName: ready
  livenessEndpointName: live
webhook:
  port: 9444
  host: 127.0.0.1
  certDir: /tmp/osok-webhook-certs
`

func TestManagerOptionsWithoutConfigUsesCommandArguments(t *testing.T) {
	t.Parallel()

	flags := managerFlags{
		metricsAddr:          ":18080",
		probeAddr:            ":18081",
		enableLeaderElection: false,
	}

	got, err := managerOptions(flags)
	if err != nil {
		t.Fatalf("managerOptions() error = %v", err)
	}

	if got.Scheme != scheme {
		t.Fatalf("Scheme = %p, want %p", got.Scheme, scheme)
	}
	if got.Metrics.BindAddress != flags.metricsAddr {
		t.Fatalf("Metrics.BindAddress = %q, want %q", got.Metrics.BindAddress, flags.metricsAddr)
	}
	if got.HealthProbeBindAddress != flags.probeAddr {
		t.Fatalf("HealthProbeBindAddress = %q, want %q", got.HealthProbeBindAddress, flags.probeAddr)
	}
	if got.LeaderElection != flags.enableLeaderElection {
		t.Fatalf("LeaderElection = %t, want %t", got.LeaderElection, flags.enableLeaderElection)
	}
	if got.LeaderElectionID != defaultLeaderElectionID {
		t.Fatalf("LeaderElectionID = %q, want %q", got.LeaderElectionID, defaultLeaderElectionID)
	}
	if got.WebhookServer != nil {
		t.Fatalf("WebhookServer = %T, want nil", got.WebhookServer)
	}
}

func TestManagerOptionsWithCheckedInConfigMatchesControllerRuntime(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join("config", "manager", "controller_manager_config.yaml")

	got, err := managerOptions(managerFlags{configFile: configPath})
	if err != nil {
		t.Fatalf("managerOptions() error = %v", err)
	}

	want, err := loadControllerRuntimeManagerOptions(t, configPath, ctrl.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("loadControllerRuntimeManagerOptions() error = %v", err)
	}

	assertManagerOptionsEqual(t, got, want)
}

func TestLoadManagerOptionsFromConfigAppliesSupportedFields(t *testing.T) {
	t.Parallel()

	got, err := loadManagerOptionsFromConfig(writeSampleManagerConfig(t), ctrl.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("loadManagerOptionsFromConfig() error = %v", err)
	}

	assertSampleManagerConfigApplied(t, got)
}

func TestLoadManagerOptionsFromConfigPreservesExistingValues(t *testing.T) {
	t.Parallel()

	existingWebhook := webhook.NewServer(webhook.Options{
		Port:    8443,
		Host:    "0.0.0.0",
		CertDir: "/tmp/existing-webhook",
	})

	got, err := loadManagerOptionsFromConfig(writeSampleManagerConfig(t), presetManagerOptions(existingWebhook))
	if err != nil {
		t.Fatalf("loadManagerOptionsFromConfig() error = %v", err)
	}

	assertPreservedManagerOptions(t, got, existingWebhook)
}

func TestAddManagerHealthChecksAddsHealthAndReadyChecks(t *testing.T) {
	t.Parallel()

	mgr := &fakeHealthCheckManager{}
	if err := addManagerHealthChecks(mgr); err != nil {
		t.Fatalf("addManagerHealthChecks() error = %v", err)
	}

	if len(mgr.healthChecks) != 1 {
		t.Fatalf("healthChecks = %d, want %d", len(mgr.healthChecks), 1)
	}
	if len(mgr.readyChecks) != 1 {
		t.Fatalf("readyChecks = %d, want %d", len(mgr.readyChecks), 1)
	}
	if mgr.healthChecks[0].name != "health" {
		t.Fatalf("healthChecks[0].name = %q, want %q", mgr.healthChecks[0].name, "health")
	}
	if mgr.readyChecks[0].name != "check" {
		t.Fatalf("readyChecks[0].name = %q, want %q", mgr.readyChecks[0].name, "check")
	}
}

func TestAddManagerHealthChecksReturnsHealthError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("health check failed")
	mgr := &fakeHealthCheckManager{healthErr: wantErr}
	if err := addManagerHealthChecks(mgr); !errors.Is(err, wantErr) {
		t.Fatalf("addManagerHealthChecks() error = %v, want %v", err, wantErr)
	}
	if len(mgr.readyChecks) != 0 {
		t.Fatalf("readyChecks = %d, want %d", len(mgr.readyChecks), 0)
	}
}

func TestAddManagerHealthChecksReturnsReadyError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("ready check failed")
	mgr := &fakeHealthCheckManager{readyErr: wantErr}
	if err := addManagerHealthChecks(mgr); !errors.Is(err, wantErr) {
		t.Fatalf("addManagerHealthChecks() error = %v, want %v", err, wantErr)
	}
	if len(mgr.healthChecks) != 1 {
		t.Fatalf("healthChecks = %d, want %d", len(mgr.healthChecks), 1)
	}
}

func mustWriteMainTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeSampleManagerConfig(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "manager.yaml")
	mustWriteMainTestFile(t, path, sampleManagerConfigYAML)
	return path
}

func presetManagerOptions(existingWebhook webhook.Server) ctrl.Options {
	options := ctrl.Options{
		Scheme:                     scheme,
		Cache:                      cache.Options{SyncPeriod: durationPointer(10 * time.Minute), DefaultNamespaces: map[string]cache.Config{"preset-system": {}}},
		GracefulShutdownTimeout:    durationPointer(15 * time.Second),
		HealthProbeBindAddress:     ":10081",
		ReadinessEndpointName:      "preset-ready",
		LivenessEndpointName:       "preset-live",
		LeaderElection:             true,
		LeaderElectionResourceLock: "leases",
		LeaderElectionNamespace:    "preset-system",
		LeaderElectionID:           "preset.oci",
		LeaseDuration:              durationPointer(11 * time.Second),
		RenewDeadline:              durationPointer(7 * time.Second),
		RetryPeriod:                durationPointer(3 * time.Second),
		WebhookServer:              existingWebhook,
	}
	options.Metrics.BindAddress = "127.0.0.1:8088"
	options.Controller.CacheSyncTimeout = 40 * time.Second
	options.Controller.GroupKindConcurrency = map[string]int{"Preset.database.oracle.com": 1}
	return options
}

func loadControllerRuntimeManagerOptions(t *testing.T, path string, options ctrl.Options) (ctrl.Options, error) {
	t.Helper()

	compatiblePath := path
	content, err := os.ReadFile(path)
	if err != nil {
		return options, err
	}

	normalized := strings.Replace(string(content), "kind: ControllerManagerConfig", "kind: ControllerManagerConfiguration", 1)
	if normalized != string(content) {
		compatiblePath = filepath.Join(t.TempDir(), filepath.Base(path))
		if err := os.WriteFile(compatiblePath, []byte(normalized), 0o644); err != nil {
			return options, err
		}
	}

	return options.AndFrom(ctrl.ConfigFile().AtPath(compatiblePath)) //nolint:staticcheck // Regression coverage compares the hand-rolled loader to the previous controller-runtime path.
}

func assertSampleManagerConfigApplied(t *testing.T, got ctrl.Options) {
	t.Helper()

	assertManagerCoreOptionsEqual(t, got, sampleAppliedCoreOptions())
	assertManagerLeaderElectionOptionsEqual(t, got, sampleAppliedLeaderElectionOptions())
	assertManagerControllerOptionsEqual(t, got, sampleAppliedControllerOptions())
	assertManagerWebhookOptionsEqual(t, got.WebhookServer, sampleWebhookServer())
}

func assertPreservedManagerOptions(t *testing.T, got ctrl.Options, existingWebhook webhook.Server) {
	t.Helper()

	assertManagerCoreOptionsEqual(t, got, presetManagerOptions(existingWebhook))
	assertManagerLeaderElectionOptionsEqual(t, got, presetManagerOptions(existingWebhook))
	assertManagerControllerOptionsEqual(t, got, presetManagerOptions(existingWebhook))
	if got.WebhookServer != existingWebhook {
		t.Fatalf("WebhookServer = %p, want %p", got.WebhookServer, existingWebhook)
	}
}

func sampleAppliedCoreOptions() ctrl.Options {
	options := ctrl.Options{
		Scheme:                  scheme,
		Cache:                   cache.Options{SyncPeriod: durationPointer(30 * time.Minute), DefaultNamespaces: map[string]cache.Config{"tenant-system": {}}},
		GracefulShutdownTimeout: durationPointer(45 * time.Second),
		HealthProbeBindAddress:  ":9091",
		ReadinessEndpointName:   "ready",
		LivenessEndpointName:    "live",
	}
	options.Metrics.BindAddress = "127.0.0.1:9090"
	return options
}

func sampleAppliedLeaderElectionOptions() ctrl.Options {
	return ctrl.Options{
		LeaderElection:             true,
		LeaderElectionResourceLock: "configmapsleases",
		LeaderElectionNamespace:    "osok-system",
		LeaderElectionID:           "runtime.oci",
		LeaseDuration:              durationPointer(20 * time.Second),
		RenewDeadline:              durationPointer(15 * time.Second),
		RetryPeriod:                durationPointer(5 * time.Second),
	}
}

func sampleAppliedControllerOptions() ctrl.Options {
	options := ctrl.Options{}
	options.Controller.CacheSyncTimeout = 90 * time.Second
	options.Controller.GroupKindConcurrency = map[string]int{"AutonomousDatabase.database.oracle.com": 3}
	return options
}

func sampleWebhookServer() webhook.Server {
	return webhook.NewServer(webhook.Options{
		Port:    9444,
		Host:    "127.0.0.1",
		CertDir: "/tmp/osok-webhook-certs",
	})
}

func assertManagerOptionsEqual(t *testing.T, got ctrl.Options, want ctrl.Options) {
	t.Helper()

	assertManagerCoreOptionsEqual(t, got, want)
	assertManagerLeaderElectionOptionsEqual(t, got, want)
	assertManagerControllerOptionsEqual(t, got, want)
	assertManagerWebhookOptionsEqual(t, got.WebhookServer, want.WebhookServer)
}

func assertManagerCoreOptionsEqual(t *testing.T, got ctrl.Options, want ctrl.Options) {
	t.Helper()

	if got.Scheme != want.Scheme {
		t.Fatalf("Scheme = %p, want %p", got.Scheme, want.Scheme)
	}
	assertDurationPointer(t, "Cache.SyncPeriod", got.Cache.SyncPeriod, want.Cache.SyncPeriod)
	if !reflect.DeepEqual(got.Cache.DefaultNamespaces, want.Cache.DefaultNamespaces) {
		t.Fatalf("Cache.DefaultNamespaces = %#v, want %#v", got.Cache.DefaultNamespaces, want.Cache.DefaultNamespaces)
	}
	assertDurationPointer(t, "GracefulShutdownTimeout", got.GracefulShutdownTimeout, want.GracefulShutdownTimeout)
	if got.Metrics.BindAddress != want.Metrics.BindAddress {
		t.Fatalf("Metrics.BindAddress = %q, want %q", got.Metrics.BindAddress, want.Metrics.BindAddress)
	}
	if got.HealthProbeBindAddress != want.HealthProbeBindAddress {
		t.Fatalf("HealthProbeBindAddress = %q, want %q", got.HealthProbeBindAddress, want.HealthProbeBindAddress)
	}
	if got.ReadinessEndpointName != want.ReadinessEndpointName {
		t.Fatalf("ReadinessEndpointName = %q, want %q", got.ReadinessEndpointName, want.ReadinessEndpointName)
	}
	if got.LivenessEndpointName != want.LivenessEndpointName {
		t.Fatalf("LivenessEndpointName = %q, want %q", got.LivenessEndpointName, want.LivenessEndpointName)
	}
}

func assertManagerLeaderElectionOptionsEqual(t *testing.T, got ctrl.Options, want ctrl.Options) {
	t.Helper()

	if got.LeaderElection != want.LeaderElection {
		t.Fatalf("LeaderElection = %t, want %t", got.LeaderElection, want.LeaderElection)
	}
	if got.LeaderElectionResourceLock != want.LeaderElectionResourceLock {
		t.Fatalf("LeaderElectionResourceLock = %q, want %q", got.LeaderElectionResourceLock, want.LeaderElectionResourceLock)
	}
	if got.LeaderElectionNamespace != want.LeaderElectionNamespace {
		t.Fatalf("LeaderElectionNamespace = %q, want %q", got.LeaderElectionNamespace, want.LeaderElectionNamespace)
	}
	if got.LeaderElectionID != want.LeaderElectionID {
		t.Fatalf("LeaderElectionID = %q, want %q", got.LeaderElectionID, want.LeaderElectionID)
	}
	assertDurationPointer(t, "LeaseDuration", got.LeaseDuration, want.LeaseDuration)
	assertDurationPointer(t, "RenewDeadline", got.RenewDeadline, want.RenewDeadline)
	assertDurationPointer(t, "RetryPeriod", got.RetryPeriod, want.RetryPeriod)
}

func assertManagerControllerOptionsEqual(t *testing.T, got ctrl.Options, want ctrl.Options) {
	t.Helper()

	if got.Controller.CacheSyncTimeout != want.Controller.CacheSyncTimeout {
		t.Fatalf("Controller.CacheSyncTimeout = %s, want %s", got.Controller.CacheSyncTimeout, want.Controller.CacheSyncTimeout)
	}
	if !reflect.DeepEqual(got.Controller.GroupKindConcurrency, want.Controller.GroupKindConcurrency) {
		t.Fatalf("Controller.GroupKindConcurrency = %#v, want %#v", got.Controller.GroupKindConcurrency, want.Controller.GroupKindConcurrency)
	}
	assertBoolPointer(t, "Controller.RecoverPanic", got.Controller.RecoverPanic, want.Controller.RecoverPanic)
}

func assertManagerWebhookOptionsEqual(t *testing.T, got webhook.Server, want webhook.Server) {
	t.Helper()

	gotWebhook := webhookOptionsFromServer(t, got)
	wantWebhook := webhookOptionsFromServer(t, want)
	if gotWebhook.Port != wantWebhook.Port {
		t.Fatalf("WebhookServer.Port = %d, want %d", gotWebhook.Port, wantWebhook.Port)
	}
	if gotWebhook.Host != wantWebhook.Host {
		t.Fatalf("WebhookServer.Host = %q, want %q", gotWebhook.Host, wantWebhook.Host)
	}
	if gotWebhook.CertDir != wantWebhook.CertDir {
		t.Fatalf("WebhookServer.CertDir = %q, want %q", gotWebhook.CertDir, wantWebhook.CertDir)
	}
	if got.NeedLeaderElection() != want.NeedLeaderElection() {
		t.Fatalf("WebhookServer.NeedLeaderElection() = %t, want %t", got.NeedLeaderElection(), want.NeedLeaderElection())
	}
}

func assertDurationPointer(t *testing.T, field string, got *time.Duration, want *time.Duration) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", field, got, want)
	case *got != *want:
		t.Fatalf("%s = %s, want %s", field, *got, *want)
	}
}

func assertBoolPointer(t *testing.T, field string, got *bool, want *bool) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", field, got, want)
	case *got != *want:
		t.Fatalf("%s = %t, want %t", field, *got, *want)
	}
}

func durationPointer(value time.Duration) *time.Duration {
	return &value
}

func webhookOptionsFromServer(t *testing.T, server webhook.Server) webhook.Options {
	t.Helper()

	defaultServer, ok := server.(*webhook.DefaultServer)
	if !ok {
		t.Fatalf("WebhookServer = %T, want *webhook.DefaultServer", server)
	}
	return defaultServer.Options
}

type recordedHealthCheck struct {
	name string
	fn   healthz.Checker
}

type fakeHealthCheckManager struct {
	healthErr    error
	readyErr     error
	healthChecks []recordedHealthCheck
	readyChecks  []recordedHealthCheck
}

func (f *fakeHealthCheckManager) AddHealthzCheck(name string, check healthz.Checker) error {
	f.healthChecks = append(f.healthChecks, recordedHealthCheck{name: name, fn: check})
	return f.healthErr
}

func (f *fakeHealthCheckManager) AddReadyzCheck(name string, check healthz.Checker) error {
	f.readyChecks = append(f.readyChecks, recordedHealthCheck{name: name, fn: check})
	return f.readyErr
}
