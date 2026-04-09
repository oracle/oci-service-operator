package authhelper

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"

	configpkg "github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

type stubOsokConfig struct {
	auth                  configpkg.UserAuthConfig
	useInstancePrincipals bool
	vaultDetails          string
}

func (s stubOsokConfig) Auth() configpkg.UserAuthConfig {
	return s.auth
}

func (s stubOsokConfig) UseInstancePrincipals() bool {
	return s.useInstancePrincipals
}

func (s stubOsokConfig) VaultDetails() string {
	return s.vaultDetails
}

func TestGetAuthProviderWithSecurityToken(t *testing.T) {
	profile := "WORKLOAD"
	configPath := writeSecurityTokenConfigFixture(t, profile)

	provider, err := (&AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
		Validator: func(_ context.Context, _ common.ConfigurationProvider, _ configpkg.OsokConfig) bool {
			return true
		},
	}).GetAuthProvider(stubOsokConfig{
		auth: configpkg.UserAuthConfig{
			AuthType:          configpkg.AuthTypeSecurityToken,
			ConfigFilePath:    configPath,
			ConfigFileProfile: profile,
		},
	})
	if err != nil {
		t.Fatalf("get auth provider: %v", err)
	}

	assertProviderTenancy(t, provider, "ocid1.tenancy.oc1..exampleuniqueID")
	assertProviderRegion(t, provider, "us-ashburn-1")
	assertProviderKeyIDPrefix(t, provider, "ST$session-token")
	assertProviderLoadsPrivateKey(t, provider)
}

func TestGetAuthProviderRejectsIncompleteUserPrincipalConfig(t *testing.T) {
	_, err := (&AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}).GetAuthProvider(stubOsokConfig{
		auth: configpkg.UserAuthConfig{
			User: "ocid1.user.oc1..example",
		},
	})
	if err == nil {
		t.Fatal("expected incomplete user principal configuration error")
	}
	if !strings.Contains(err.Error(), "incomplete user principal configuration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAuthProviderUsesRawUserPrincipal(t *testing.T) {
	privateKey := string(generateRSAPrivateKeyPEM(t))

	provider, err := (&AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
		Validator: func(_ context.Context, _ common.ConfigurationProvider, _ configpkg.OsokConfig) bool {
			return true
		},
	}).GetAuthProvider(stubOsokConfig{
		auth: configpkg.UserAuthConfig{
			Tenancy:     "ocid1.tenancy.oc1..example",
			User:        "ocid1.user.oc1..example",
			Region:      "us-ashburn-1",
			Fingerprint: "20:3b:97:13:55:1c",
			PrivateKey:  privateKey,
		},
	})
	if err != nil {
		t.Fatalf("get auth provider: %v", err)
	}

	if user, err := provider.UserOCID(); err != nil || user != "ocid1.user.oc1..example" {
		t.Fatalf("unexpected user: got %q err=%v", user, err)
	}
	if keyID, err := provider.KeyID(); err != nil || keyID != "ocid1.tenancy.oc1..example/ocid1.user.oc1..example/20:3b:97:13:55:1c" {
		t.Fatalf("unexpected key id: got %q err=%v", keyID, err)
	}
	if _, err := provider.PrivateRSAKey(); err != nil {
		t.Fatalf("load private key: %v", err)
	}
}

func TestResolveValidationTenancyUsesProviderTenancyWhenConfigIsEmpty(t *testing.T) {
	privateKey := string(generateRSAPrivateKeyPEM(t))
	provider := common.NewRawConfigurationProvider(
		"ocid1.tenancy.oc1..example",
		"ocid1.user.oc1..example",
		"us-ashburn-1",
		"20:3b:97:13:55:1c",
		privateKey,
		nil,
	)

	tenancy, err := resolveValidationTenancy(provider, stubOsokConfig{})
	if err != nil {
		t.Fatalf("resolve validation tenancy: %v", err)
	}
	if tenancy != "ocid1.tenancy.oc1..example" {
		t.Fatalf("unexpected tenancy: got %q", tenancy)
	}
}

func generateRSAPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func writeSecurityTokenConfigFixture(t *testing.T, profile string) string {
	t.Helper()

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "privatekey")
	tokenPath := filepath.Join(dir, "security_token")
	configPath := filepath.Join(dir, "config")

	mustWriteFile(t, keyPath, generateRSAPrivateKeyPEM(t))
	mustWriteFile(t, tokenPath, []byte("session-token"))

	configFile := strings.Join([]string{
		"[" + profile + "]",
		"tenancy=ocid1.tenancy.oc1..exampleuniqueID",
		"region=us-ashburn-1",
		"fingerprint=20:3b:97:13:55:1c",
		"key_file=" + keyPath,
		"security_token_file=" + tokenPath,
	}, "\n")
	mustWriteFile(t, configPath, []byte(configFile))

	return configPath
}

func mustWriteFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertProviderTenancy(t *testing.T, provider common.ConfigurationProvider, want string) {
	t.Helper()

	got, err := provider.TenancyOCID()
	if err != nil || got != want {
		t.Fatalf("unexpected tenancy: got %q err=%v", got, err)
	}
}

func assertProviderRegion(t *testing.T, provider common.ConfigurationProvider, want string) {
	t.Helper()

	got, err := provider.Region()
	if err != nil || got != want {
		t.Fatalf("unexpected region: got %q err=%v", got, err)
	}
}

func assertProviderKeyIDPrefix(t *testing.T, provider common.ConfigurationProvider, prefix string) {
	t.Helper()

	keyID, err := provider.KeyID()
	if err != nil {
		t.Fatalf("get key id: %v", err)
	}
	if !strings.HasPrefix(keyID, prefix) {
		t.Fatalf("expected session token key id prefix, got %q", keyID)
	}
}

func assertProviderLoadsPrivateKey(t *testing.T, provider common.ConfigurationProvider) {
	t.Helper()

	if _, err := provider.PrivateRSAKey(); err != nil {
		t.Fatalf("load private key: %v", err)
	}
}
