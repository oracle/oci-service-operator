package config

import (
	"testing"

	"github.com/go-logr/logr"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

func TestGetConfigDetailsReadsSecurityTokenEnv(t *testing.T) {
	resetConfigDetails(t)

	t.Setenv("AUTH_TYPE", AuthTypeSecurityToken)
	t.Setenv("OCI_CONFIG_FILE_PATH", "/etc/oci/custom-config")
	t.Setenv("OCI_CONFIG_PROFILE", "WORKLOAD")
	t.Setenv("PASSPHRASE", "secret-passphrase")

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

func TestGetConfigDetailsNormalizesAuthTypeSelection(t *testing.T) {
	testCases := []struct {
		name     string
		env      map[string]string
		wantType string
	}{
		{
			name: "explicit auth type wins",
			env: map[string]string{
				"AUTH_TYPE": AuthTypeResourcePrincipal,
			},
			wantType: AuthTypeResourcePrincipal,
		},
		{
			name: "sdk typo normalizes to operator string",
			env: map[string]string{
				"AUTH_TYPE": "instance_principle_delegation_token",
			},
			wantType: AuthTypeInstancePrincipalDelegationToken,
		},
		{
			name: "raw user principal defaults to user principal mode",
			env: map[string]string{
				"USER":        "ocid1.user.oc1..example",
				"TENANCY":     "ocid1.tenancy.oc1..example",
				"REGION":      "us-ashburn-1",
				"FINGERPRINT": "20:3b:97:13:55:1c",
				"PRIVATEKEY":  "PEM",
			},
			wantType: AuthTypeUserPrincipal,
		},
		{
			name: "config file inputs default to user principal mode",
			env: map[string]string{
				"OCI_CONFIG_PROFILE": "WORKLOAD",
			},
			wantType: AuthTypeUserPrincipal,
		},
		{
			name: "legacy instance principal flag remains supported",
			env: map[string]string{
				"USEINSTANCEPRINCIPAL": "true",
			},
			wantType: AuthTypeInstancePrincipal,
		},
		{
			name:     "empty config still defaults to instance principal",
			wantType: AuthTypeInstancePrincipal,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resetConfigDetails(t)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			cfg := GetConfigDetails(loggerutil.OSOKLogger{Logger: logr.Discard()})
			if got := cfg.Auth().AuthType; got != tc.wantType {
				t.Fatalf("unexpected auth type: got %q want %q", got, tc.wantType)
			}
		})
	}
}

func TestGetConfigDetailsReadsAdvancedAuthEnv(t *testing.T) {
	resetConfigDetails(t)

	t.Setenv("AUTH_TYPE", AuthTypeInstancePrincipalWithCerts)
	t.Setenv("INSTANCE_PRINCIPAL_DELEGATION_TOKEN", "instance-obo-token")
	t.Setenv("RESOURCE_PRINCIPAL_DELEGATION_TOKEN", "resource-obo-token")
	t.Setenv("INSTANCE_PRINCIPAL_LEAF_CERTIFICATE_PATH", "/etc/oci/instance-principal.crt")
	t.Setenv("INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PATH", "/etc/oci/instance-principal.key")
	t.Setenv("INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PASSPHRASE", "leaf-passphrase")
	t.Setenv("INSTANCE_PRINCIPAL_INTERMEDIATE_CERTIFICATE_PATHS", "/etc/oci/int-a.pem,/etc/oci/int-b.pem")

	cfg := GetConfigDetails(loggerutil.OSOKLogger{Logger: logr.Discard()})

	if got := cfg.Auth().InstancePrincipalDelegationToken; got != "instance-obo-token" {
		t.Fatalf("unexpected instance delegation token: got %q", got)
	}
	if got := cfg.Auth().ResourcePrincipalDelegationToken; got != "resource-obo-token" {
		t.Fatalf("unexpected resource delegation token: got %q", got)
	}
	if got := cfg.Auth().InstancePrincipalLeafCertificatePath; got != "/etc/oci/instance-principal.crt" {
		t.Fatalf("unexpected leaf certificate path: got %q", got)
	}
	if got := cfg.Auth().InstancePrincipalLeafPrivateKeyPath; got != "/etc/oci/instance-principal.key" {
		t.Fatalf("unexpected leaf private key path: got %q", got)
	}
	if got := cfg.Auth().InstancePrincipalLeafPrivateKeyPassphrase; got != "leaf-passphrase" {
		t.Fatalf("unexpected leaf private key passphrase: got %q", got)
	}
	if got := cfg.Auth().IntermediateCertificatePaths(); len(got) != 2 || got[0] != "/etc/oci/int-a.pem" || got[1] != "/etc/oci/int-b.pem" {
		t.Fatalf("unexpected intermediate certificate paths: got %#v", got)
	}
}

func resetConfigDetails(t *testing.T) {
	t.Helper()

	for _, envKey := range []string{
		"AUTH_TYPE",
		"USER",
		"TENANCY",
		"REGION",
		"FINGERPRINT",
		"PASSPHRASE",
		"PRIVATEKEY",
		"OCI_CONFIG_FILE_PATH",
		"OCI_CONFIG_PROFILE",
		"USEINSTANCEPRINCIPAL",
		"INSTANCE_PRINCIPAL_DELEGATION_TOKEN",
		"RESOURCE_PRINCIPAL_DELEGATION_TOKEN",
		"INSTANCE_PRINCIPAL_LEAF_CERTIFICATE_PATH",
		"INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PATH",
		"INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PASSPHRASE",
		"INSTANCE_PRINCIPAL_INTERMEDIATE_CERTIFICATE_PATHS",
	} {
		t.Setenv(envKey, "")
	}

	configDetails = osokConfig{}
	t.Cleanup(func() {
		configDetails = osokConfig{}
	})
}
