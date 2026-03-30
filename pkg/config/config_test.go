package config

import (
	"testing"

	"github.com/go-logr/logr"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

func TestGetConfigDetailsReadsSecurityTokenEnv(t *testing.T) {
	t.Setenv("AUTH_TYPE", AuthTypeSecurityToken)
	t.Setenv("OCI_CONFIG_FILE_PATH", "/etc/oci/custom-config")
	t.Setenv("OCI_CONFIG_PROFILE", "WORKLOAD")
	t.Setenv("PASSPHRASE", "secret-passphrase")

	configDetails = osokConfig{}
	t.Cleanup(func() {
		configDetails = osokConfig{}
	})

	cfg := GetConfigDetails(loggerutil.OSOKLogger{Logger: logr.Discard()})

	if got := cfg.Auth().AuthType; got != AuthTypeSecurityToken {
		t.Fatalf("unexpected auth type: got %q want %q", got, AuthTypeSecurityToken)
	}
	if got := cfg.Auth().ConfigFilePath; got != "/etc/oci/custom-config" {
		t.Fatalf("unexpected config file path: got %q", got)
	}
	if got := cfg.Auth().ConfigFileProfile; got != "WORKLOAD" {
		t.Fatalf("unexpected config file profile: got %q", got)
	}
	if got := cfg.Auth().Passphrase; got != "secret-passphrase" {
		t.Fatalf("unexpected passphrase: got %q", got)
	}
}
