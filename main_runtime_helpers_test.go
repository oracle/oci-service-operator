/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func TestDefaultManagerOptions(t *testing.T) {
	t.Parallel()

	options := defaultManagerOptions(":8080", ":8081", true)
	if options.Scheme != scheme {
		t.Fatal("default manager options should use the shared scheme")
	}
	if options.Metrics.BindAddress != ":8080" {
		t.Fatalf("metrics bind address = %q, want :8080", options.Metrics.BindAddress)
	}
	if options.HealthProbeBindAddress != ":8081" {
		t.Fatalf("health probe bind address = %q, want :8081", options.HealthProbeBindAddress)
	}
	if !options.LeaderElection {
		t.Fatal("leader election should be enabled in the default options")
	}
	if options.LeaderElectionID != leaderElectionID {
		t.Fatalf("leader election ID = %q, want %q", options.LeaderElectionID, leaderElectionID)
	}
}

func TestManagerOptionsFromInputsUsesConfigFile(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "controller_manager_config.yaml")
	configContents := `apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
cacheNamespace: system
syncPeriod: 1m0s
gracefulShutDown: 30s
metrics:
  bindAddress: 127.0.0.1:8080
health:
  healthProbeBindAddress: :9090
  readinessEndpointName: ready
  livenessEndpointName: live
webhook:
  host: 127.0.0.1
  port: 9443
  certDir: /certs
leaderElection:
  leaderElect: true
  resourceName: test-lock
  resourceNamespace: system
  resourceLock: leases
  leaseDuration: 15s
  renewDeadline: 10s
  retryPeriod: 2s
controller:
  cacheSyncTimeout: 45s
  groupKindConcurrency:
    Stream.streaming.oracle.com: 3
`
	if err := os.WriteFile(configPath, []byte(configContents), 0o600); err != nil {
		t.Fatalf("write manager config file: %v", err)
	}

	options, err := managerOptionsFromInputs(configPath, ":1111", ":2222", false)
	if err != nil {
		t.Fatalf("managerOptionsFromInputs() error = %v", err)
	}

	assertManagerOptionsScheme(t, options)
	assertManagerOptionsCache(t, options)
	assertManagerOptionsHealth(t, options)
	assertManagerOptionsLeaderElection(t, options)
	assertManagerOptionsLeaderElectionTiming(t, options)
	assertManagerOptionsController(t, options)
	assertManagerOptionsWebhook(t, options)
}

func assertManagerOptionsScheme(t *testing.T, options ctrl.Options) {
	t.Helper()
	if options.Scheme != scheme {
		t.Fatal("config-backed manager options should use the shared scheme")
	}
	if options.Metrics.BindAddress != "127.0.0.1:8080" {
		t.Fatalf("metrics bind address = %q, want 127.0.0.1:8080", options.Metrics.BindAddress)
	}
}

func assertManagerOptionsCache(t *testing.T, options ctrl.Options) {
	t.Helper()
	if options.Cache.SyncPeriod == nil || *options.Cache.SyncPeriod != time.Minute {
		t.Fatalf("sync period = %v, want %v", options.Cache.SyncPeriod, time.Minute)
	}
	if _, ok := options.Cache.DefaultNamespaces["system"]; !ok {
		t.Fatalf("default namespaces = %#v, want system", options.Cache.DefaultNamespaces)
	}
	if options.GracefulShutdownTimeout == nil || *options.GracefulShutdownTimeout != 30*time.Second {
		t.Fatalf("graceful shutdown timeout = %v, want %v", options.GracefulShutdownTimeout, 30*time.Second)
	}
}

func assertManagerOptionsHealth(t *testing.T, options ctrl.Options) {
	t.Helper()
	if options.HealthProbeBindAddress != ":9090" {
		t.Fatalf("health probe bind address = %q, want :9090", options.HealthProbeBindAddress)
	}
	if options.ReadinessEndpointName != "ready" {
		t.Fatalf("readiness endpoint name = %q, want ready", options.ReadinessEndpointName)
	}
	if options.LivenessEndpointName != "live" {
		t.Fatalf("liveness endpoint name = %q, want live", options.LivenessEndpointName)
	}
}

func assertManagerOptionsLeaderElection(t *testing.T, options ctrl.Options) {
	t.Helper()
	if !options.LeaderElection {
		t.Fatal("leader election should be enabled from the config file")
	}
	if options.LeaderElectionID != "test-lock" {
		t.Fatalf("leader election ID = %q, want test-lock", options.LeaderElectionID)
	}
	if options.LeaderElectionNamespace != "system" {
		t.Fatalf("leader election namespace = %q, want system", options.LeaderElectionNamespace)
	}
	if options.LeaderElectionResourceLock != "leases" {
		t.Fatalf("leader election resource lock = %q, want leases", options.LeaderElectionResourceLock)
	}
}

func assertManagerOptionsLeaderElectionTiming(t *testing.T, options ctrl.Options) {
	t.Helper()
	if options.LeaseDuration == nil || *options.LeaseDuration != 15*time.Second {
		t.Fatalf("lease duration = %v, want %v", options.LeaseDuration, 15*time.Second)
	}
	if options.RenewDeadline == nil || *options.RenewDeadline != 10*time.Second {
		t.Fatalf("renew deadline = %v, want %v", options.RenewDeadline, 10*time.Second)
	}
	if options.RetryPeriod == nil || *options.RetryPeriod != 2*time.Second {
		t.Fatalf("retry period = %v, want %v", options.RetryPeriod, 2*time.Second)
	}
}

func assertManagerOptionsController(t *testing.T, options ctrl.Options) {
	t.Helper()
	if options.Controller.CacheSyncTimeout != 45*time.Second {
		t.Fatalf("controller cache sync timeout = %v, want %v", options.Controller.CacheSyncTimeout, 45*time.Second)
	}
	if got := options.Controller.GroupKindConcurrency["Stream.streaming.oracle.com"]; got != 3 {
		t.Fatalf("stream concurrency = %d, want 3", got)
	}
}

func assertManagerOptionsWebhook(t *testing.T, options ctrl.Options) {
	t.Helper()
	server, ok := options.WebhookServer.(*webhook.DefaultServer)
	if !ok {
		t.Fatalf("webhook server = %T, want *webhook.DefaultServer", options.WebhookServer)
	}
	if server.Options.Host != "127.0.0.1" {
		t.Fatalf("webhook host = %q, want 127.0.0.1", server.Options.Host)
	}
	if server.Options.Port != 9443 {
		t.Fatalf("webhook port = %d, want 9443", server.Options.Port)
	}
	if server.Options.CertDir != "/certs" {
		t.Fatalf("webhook cert dir = %q, want /certs", server.Options.CertDir)
	}
}
