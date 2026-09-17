/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package decryptionprofile

import (
	"context"
	"fmt"
	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDecryptionProfileCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &networkfirewallv1beta1.DecryptionProfile{}
	ocimock.InitializeResource(resource, "mock-decryptionprofile")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.DecryptionProfileSpec](t, `{
  "description": "OSOK recorded decryption profile",
  "name": "osok_mock_decrypt_profile",
  "type": "SSL_FORWARD_PROXY"
}`)
	resource.Spec.NetworkFirewallPolicyId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded DecryptionProfile updated",
  "type": "SSL_FORWARD_PROXY"
}`)
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateSslForwardProxyProfileDetails](t, `{
  "description": "OSOK recorded decryption profile",
  "name": "osok_mock_decrypt_profile"
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.SslForwardProxyProfile](t, `{
  "areCertificateExtensionsRestricted": false,
  "description": "OSOK recorded decryption profile",
  "isAutoIncludeAltName": false,
  "isExpiredCertificateBlocked": false,
  "isOutOfCapacityBlocked": false,
  "isRevocationStatusTimeoutBlocked": false,
  "isUnknownRevocationStatusBlocked": false,
  "isUnsupportedCipherBlocked": false,
  "isUnsupportedVersionBlocked": false,
  "isUntrustedIssuerBlocked": false,
  "name": "osok_mock_decrypt_profile",
  "parentResourceId": "<ocid:1>",
  "type": "SSL_FORWARD_PROXY"
}`)
	createdReadStates := []networkfirewallsdk.SslForwardProxyProfile{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.SslForwardProxyProfile](t, `{
  "areCertificateExtensionsRestricted": false,
  "description": "OSOK recorded decryption profile",
  "isAutoIncludeAltName": false,
  "isExpiredCertificateBlocked": false,
  "isOutOfCapacityBlocked": false,
  "isRevocationStatusTimeoutBlocked": false,
  "isUnknownRevocationStatusBlocked": false,
  "isUnsupportedCipherBlocked": false,
  "isUnsupportedVersionBlocked": false,
  "isUntrustedIssuerBlocked": false,
  "name": "osok_mock_decrypt_profile",
  "parentResourceId": "<ocid:1>",
  "type": "SSL_FORWARD_PROXY"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateSslForwardProxyProfileDetails](t, `{
  "description": "OSOK recorded DecryptionProfile updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.SslForwardProxyProfile](t, `{
  "areCertificateExtensionsRestricted": false,
  "description": "OSOK recorded DecryptionProfile updated",
  "isAutoIncludeAltName": false,
  "isExpiredCertificateBlocked": false,
  "isOutOfCapacityBlocked": false,
  "isRevocationStatusTimeoutBlocked": false,
  "isUnknownRevocationStatusBlocked": false,
  "isUnsupportedCipherBlocked": false,
  "isUnsupportedVersionBlocked": false,
  "isUntrustedIssuerBlocked": false,
  "name": "osok_mock_decrypt_profile",
  "parentResourceId": "<ocid:1>",
  "type": "SSL_FORWARD_PROXY"
}`)
	updatedReadStates := []networkfirewallsdk.SslForwardProxyProfile{
		ocimock.MustOCIResponseFixture[networkfirewallsdk.SslForwardProxyProfile](t, `{
  "areCertificateExtensionsRestricted": false,
  "description": "OSOK recorded DecryptionProfile updated",
  "isAutoIncludeAltName": false,
  "isExpiredCertificateBlocked": false,
  "isOutOfCapacityBlocked": false,
  "isRevocationStatusTimeoutBlocked": false,
  "isUnknownRevocationStatusBlocked": false,
  "isUnsupportedCipherBlocked": false,
  "isUnsupportedVersionBlocked": false,
  "isUntrustedIssuerBlocked": false,
  "name": "osok_mock_decrypt_profile",
  "parentResourceId": "<ocid:1>",
  "type": "SSL_FORWARD_PROXY"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.SslForwardProxyProfile,
		networkfirewallsdk.CreateSslForwardProxyProfileDetails,
		networkfirewallsdk.UpdateSslForwardProxyProfileDetails,
	]{
		CollectionPath:     "/20230501/networkFirewallPolicies/<ocid:1>/decryptionProfiles",
		ItemPath:           "/20230501/networkFirewallPolicies/<ocid:1>/decryptionProfiles/osok_mock_decrypt_profile",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:       &createdState,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "SSL_FORWARD_PROXY", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "SSL_FORWARD_PROXY", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ networkfirewallsdk.SslForwardProxyProfile) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20230501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DecryptionProfile OCI mock: %v", err)
		}
	})
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &DecryptionProfileServiceManager{}
	hooks := newDecryptionProfileRuntimeHooks(manager, sdkClient)
	client := wrapDecryptionProfileGeneratedClient(hooks, defaultDecryptionProfileServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.DecryptionProfile](buildDecryptionProfileGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.DecryptionProfile]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.DecryptionProfile) error {
			if !reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created DecryptionProfile status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.DecryptionProfile) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.DecryptionProfile) error {
			if !reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("updated DecryptionProfile status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
