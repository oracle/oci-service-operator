package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveManagerOptionsUsesDefaultFlagsWithoutConfigFile(t *testing.T) {
	t.Parallel()

	options, err := resolveManagerOptions(startupFlags{
		metricsAddr:          "127.0.0.1:18080",
		probeAddr:            ":18081",
		enableLeaderElection: false,
	})
	if err != nil {
		t.Fatalf("resolveManagerOptions() error = %v", err)
	}

	if options.Scheme != scheme {
		t.Fatal("resolveManagerOptions() did not reuse the global scheme")
	}
	if options.Metrics.BindAddress != "127.0.0.1:18080" {
		t.Fatalf("options.Metrics.BindAddress = %q, want %q", options.Metrics.BindAddress, "127.0.0.1:18080")
	}
	if options.HealthProbeBindAddress != ":18081" {
		t.Fatalf("options.HealthProbeBindAddress = %q, want %q", options.HealthProbeBindAddress, ":18081")
	}
	if options.LeaderElection {
		t.Fatal("options.LeaderElection = true, want false")
	}
	if options.LeaderElectionID != defaultLeaderElectionID {
		t.Fatalf("options.LeaderElectionID = %q, want %q", options.LeaderElectionID, defaultLeaderElectionID)
	}
}

func TestResolveManagerOptionsLoadsControllerManagerConfigFile(t *testing.T) {
	t.Parallel()

	configPath := writeTempManagerConfig(t, `
apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
syncPeriod: "45s"
cacheNamespace: operator-system
gracefulShutDown: "30s"
leaderElection:
  leaderElect: true
  leaseDuration: "1m"
  renewDeadline: "40s"
  retryPeriod: "10s"
  resourceLock: leases
  resourceName: custom-leader-election
  resourceNamespace: operator-system
controller:
  groupKindConcurrency:
    core/Vcn: 4
  cacheSyncTimeout: "2m"
  recoverPanic: true
metrics:
  bindAddress: 127.0.0.1:28080
health:
  healthProbeBindAddress: :28081
  readinessEndpointName: readyz-custom
  livenessEndpointName: healthz-custom
webhook:
  port: 9443
  host: 127.0.0.1
  certDir: /tmp/osok-webhook-certs
`)

	options, err := resolveManagerOptions(startupFlags{
		configFile:           configPath,
		metricsAddr:          "ignored-by-config",
		probeAddr:            "ignored-by-config",
		enableLeaderElection: false,
	})
	if err != nil {
		t.Fatalf("resolveManagerOptions() error = %v", err)
	}

	if options.Scheme != scheme {
		t.Fatal("resolveManagerOptions() did not reuse the global scheme")
	}
	if options.Metrics.BindAddress != "127.0.0.1:28080" {
		t.Fatalf("options.Metrics.BindAddress = %q, want %q", options.Metrics.BindAddress, "127.0.0.1:28080")
	}
	if options.HealthProbeBindAddress != ":28081" {
		t.Fatalf("options.HealthProbeBindAddress = %q, want %q", options.HealthProbeBindAddress, ":28081")
	}
	if options.ReadinessEndpointName != "readyz-custom" {
		t.Fatalf("options.ReadinessEndpointName = %q, want %q", options.ReadinessEndpointName, "readyz-custom")
	}
	if options.LivenessEndpointName != "healthz-custom" {
		t.Fatalf("options.LivenessEndpointName = %q, want %q", options.LivenessEndpointName, "healthz-custom")
	}
	if options.Cache.SyncPeriod == nil || *options.Cache.SyncPeriod != 45*time.Second {
		t.Fatalf("options.Cache.SyncPeriod = %v, want %v", options.Cache.SyncPeriod, 45*time.Second)
	}
	if _, ok := options.Cache.DefaultNamespaces["operator-system"]; !ok {
		t.Fatalf("options.Cache.DefaultNamespaces = %v, want operator-system entry", options.Cache.DefaultNamespaces)
	}
	if options.GracefulShutdownTimeout == nil || *options.GracefulShutdownTimeout != 30*time.Second {
		t.Fatalf("options.GracefulShutdownTimeout = %v, want %v", options.GracefulShutdownTimeout, 30*time.Second)
	}
	if !options.LeaderElection {
		t.Fatal("options.LeaderElection = false, want true")
	}
	if options.LeaderElectionResourceLock != "leases" {
		t.Fatalf("options.LeaderElectionResourceLock = %q, want %q", options.LeaderElectionResourceLock, "leases")
	}
	if options.LeaderElectionNamespace != "operator-system" {
		t.Fatalf("options.LeaderElectionNamespace = %q, want %q", options.LeaderElectionNamespace, "operator-system")
	}
	if options.LeaderElectionID != "custom-leader-election" {
		t.Fatalf("options.LeaderElectionID = %q, want %q", options.LeaderElectionID, "custom-leader-election")
	}
	if options.LeaseDuration == nil || *options.LeaseDuration != time.Minute {
		t.Fatalf("options.LeaseDuration = %v, want %v", options.LeaseDuration, time.Minute)
	}
	if options.RenewDeadline == nil || *options.RenewDeadline != 40*time.Second {
		t.Fatalf("options.RenewDeadline = %v, want %v", options.RenewDeadline, 40*time.Second)
	}
	if options.RetryPeriod == nil || *options.RetryPeriod != 10*time.Second {
		t.Fatalf("options.RetryPeriod = %v, want %v", options.RetryPeriod, 10*time.Second)
	}
	if options.Controller.CacheSyncTimeout != 2*time.Minute {
		t.Fatalf("options.Controller.CacheSyncTimeout = %v, want %v", options.Controller.CacheSyncTimeout, 2*time.Minute)
	}
	if got := options.Controller.GroupKindConcurrency["core/Vcn"]; got != 4 {
		t.Fatalf("options.Controller.GroupKindConcurrency[core/Vcn] = %d, want %d", got, 4)
	}
	if options.Controller.RecoverPanic == nil || !*options.Controller.RecoverPanic {
		t.Fatalf("options.Controller.RecoverPanic = %v, want true", options.Controller.RecoverPanic)
	}
	if options.WebhookServer == nil {
		t.Fatal("options.WebhookServer = nil, want initialized server")
	}
}

func TestResolveManagerOptionsWrapsConfigLoadErrors(t *testing.T) {
	t.Parallel()

	missingPath := filepath.Join(t.TempDir(), "missing-controller-manager-config.yaml")
	_, err := resolveManagerOptions(startupFlags{configFile: missingPath})
	if err == nil {
		t.Fatal("resolveManagerOptions() error = nil, want load failure")
	}
	if !strings.Contains(err.Error(), "unable to load the config file") {
		t.Fatalf("resolveManagerOptions() error = %q, want wrapper prefix", err)
	}
	if !strings.Contains(err.Error(), missingPath) {
		t.Fatalf("resolveManagerOptions() error = %q, want missing path", err)
	}
}

func TestLoadManagerOptionsFromFileRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	configPath := writeTempManagerConfig(t, `
apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
metrics:
  bindAddress: 127.0.0.1:38080
unexpectedField: true
`)

	_, err := loadManagerOptionsFromFile(configPath)
	if err == nil {
		t.Fatal("loadManagerOptionsFromFile() error = nil, want strict unmarshal failure")
	}
	if !strings.Contains(err.Error(), "unexpectedField") {
		t.Fatalf("loadManagerOptionsFromFile() error = %q, want unexpectedField details", err)
	}
}

func TestLoadManagerOptionsFromFileRejectsInvalidTypeMeta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		apiVersionLine string
		kindLine       string
		want           string
	}{
		{
			name:     "missing apiVersion",
			kindLine: "kind: " + expectedControllerManagerConfigKind,
			want:     `controller manager config apiVersion = "", want "` + expectedControllerManagerConfigAPIVersion + `"`,
		},
		{
			name:           "missing kind",
			apiVersionLine: "apiVersion: " + expectedControllerManagerConfigAPIVersion,
			want:           `controller manager config kind = "", want "` + expectedControllerManagerConfigKind + `"`,
		},
		{
			name:           "wrong apiVersion",
			apiVersionLine: "apiVersion: wrong.example/v1",
			kindLine:       "kind: " + expectedControllerManagerConfigKind,
			want:           `controller manager config apiVersion = "wrong.example/v1", want "` + expectedControllerManagerConfigAPIVersion + `"`,
		},
		{
			name:           "wrong kind",
			apiVersionLine: "apiVersion: " + expectedControllerManagerConfigAPIVersion,
			kindLine:       "kind: WrongControllerManagerConfig",
			want:           `controller manager config kind = "WrongControllerManagerConfig", want "` + expectedControllerManagerConfigKind + `"`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			lines := make([]string, 0, 4)
			if test.apiVersionLine != "" {
				lines = append(lines, test.apiVersionLine)
			}
			if test.kindLine != "" {
				lines = append(lines, test.kindLine)
			}
			lines = append(lines,
				"metrics:",
				"  bindAddress: 127.0.0.1:48080",
			)

			configPath := writeTempManagerConfig(t, strings.Join(lines, "\n"))

			_, err := loadManagerOptionsFromFile(configPath)
			if err == nil {
				t.Fatal("loadManagerOptionsFromFile() error = nil, want invalid type metadata failure")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadManagerOptionsFromFile() error = %q, want substring %q", err, test.want)
			}
		})
	}
}

func writeTempManagerConfig(t *testing.T, content string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "controller_manager_config.yaml")
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}
	return configPath
}
