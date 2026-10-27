/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authhelper

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
)

// Exercise the pinned OCI SDK's TLS provider and transport with the Go defaults
// used by the controller images. A successful TLS 1.3 handshake alone does not
// prove PQC support: check the negotiated group as well as the protocol version.
func TestOCISDKTLSNegotiation(t *testing.T) {
	t.Setenv("OCI_DEFAULT_CLIENT_CERTS_PATH", "")
	t.Setenv("OCI_DEFAULT_CLIENT_CERTS_PRIVATE_KEY_PATH", "")

	for _, useCustomBundle := range []bool{false, true} {
		bundleName := "default TLS config"
		if useCustomBundle {
			bundleName = "custom CA bundle"
		}
		t.Run(bundleName, func(t *testing.T) {
			for _, tc := range []struct {
				name       string
				minVersion uint16
				maxVersion uint16
				groups     []tls.CurveID
				wantGroup  tls.CurveID
			}{
				{"prefers hybrid TLS 1.3", tls.VersionTLS12, tls.VersionTLS13, []tls.CurveID{tls.X25519MLKEM768, tls.CurveP256}, tls.X25519MLKEM768},
				{"P-256 hybrid TLS 1.3", tls.VersionTLS13, tls.VersionTLS13, []tls.CurveID{tls.SecP256r1MLKEM768}, tls.SecP256r1MLKEM768},
				{"classical TLS 1.3 peer", tls.VersionTLS13, tls.VersionTLS13, []tls.CurveID{tls.CurveP256}, tls.CurveP256},
				{"TLS 1.2 peer", tls.VersionTLS12, tls.VersionTLS12, []tls.CurveID{tls.CurveP256}, tls.CurveP256},
			} {
				t.Run(tc.name, func(t *testing.T) {
					server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNoContent)
					}))
					server.TLS = &tls.Config{
						MinVersion:       tc.minVersion,
						MaxVersion:       tc.maxVersion,
						CurvePreferences: tc.groups,
					}
					server.StartTLS()
					defer server.Close()

					t.Setenv("OCI_DEFAULT_CERTS_PATH", "")
					if useCustomBundle {
						bundlePath := filepath.Join(t.TempDir(), "ca-bundle.pem")
						bundle := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
						if err := os.WriteFile(bundlePath, bundle, 0600); err != nil {
							t.Fatal(err)
						}
						t.Setenv("OCI_DEFAULT_CERTS_PATH", bundlePath)
					}
					tlsConfig, err := common.GetTLSConfigTemplateForTransport().NewOrDefault()
					if err != nil {
						t.Fatal(err)
					}
					if !useCustomBundle {
						// Trust only the loopback test server in place of system roots.
						tlsConfig.RootCAs = x509.NewCertPool()
						tlsConfig.RootCAs.AddCert(server.Certificate())
					}
					transport, err := common.DefaultTransport(tlsConfig)
					if err != nil {
						t.Fatal(err)
					}
					defer transport.CloseIdleConnections()
					client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
					response, err := client.Get(server.URL)
					if err != nil {
						t.Fatalf("OCI SDK TLS handshake: %v", err)
					}
					defer response.Body.Close()
					if response.TLS == nil {
						t.Fatal("response did not use TLS")
					}
					state := response.TLS
					if state.Version != tc.maxVersion || state.CurveID != tc.wantGroup {
						t.Fatalf("negotiated TLS %#x, group %s; want TLS %#x, group %s", state.Version, state.CurveID, tc.maxVersion, tc.wantGroup)
					}
					if len(state.VerifiedChains) == 0 || state.DidResume {
						t.Fatal("expected a fresh handshake with a verified certificate chain")
					}
					t.Logf("TLS %#x, negotiated group %s", state.Version, state.CurveID)
				})
			}
		})
	}
}
