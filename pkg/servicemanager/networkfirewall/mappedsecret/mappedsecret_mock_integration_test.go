/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package mappedsecret

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Explicit typed service-manager lifecycle; mock evidence is authoring reference only.
func TestMockIntegrationMappedSecretCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &networkfirewallv1beta1.MappedSecret{}
	ocimock.InitializeResource(resource, "mock-mapped-secret")
	resource.Spec = ocimock.MustJSONFixture[networkfirewallv1beta1.MappedSecretSpec](t, `{
  "networkFirewallPolicyId":"<ocid:1>",
  "name":"osok_mock_mapped_secret",
  "description":"mock create",
  "type":"SSL_FORWARD_PROXY",
  "source":"OCI_VAULT",
  "vaultSecretId":"<ocid:2>",
  "versionNumber":1
}`)
	updatedSpec := resource.Spec
	updatedSpec.Description = "mock update"
	updatedSpec.VersionNumber = 2
	createRequest := ocimock.MustJSONFixture[networkfirewallsdk.CreateVaultMappedSecretDetails](t, `{
  "name":"osok_mock_mapped_secret","description":"mock create","type":"SSL_FORWARD_PROXY",
  "vaultSecretId":"<ocid:2>","versionNumber":1
}`)
	createdState := ocimock.MustOCIResponseFixture[networkfirewallsdk.VaultMappedSecret](t, `{
  "name":"osok_mock_mapped_secret","description":"mock create","type":"SSL_FORWARD_PROXY",
  "source":"OCI_VAULT","vaultSecretId":"<ocid:2>","versionNumber":1,"parentResourceId":"<ocid:1>"
}`)
	updateRequest := ocimock.MustJSONFixture[networkfirewallsdk.UpdateVaultMappedSecretDetails](t, `{
  "description":"mock update","type":"SSL_FORWARD_PROXY",
  "vaultSecretId":"<ocid:2>","versionNumber":2
}`)
	updatedState := ocimock.MustOCIResponseFixture[networkfirewallsdk.VaultMappedSecret](t, `{
  "name":"osok_mock_mapped_secret","description":"mock update","type":"SSL_FORWARD_PROXY",
  "source":"OCI_VAULT","vaultSecretId":"<ocid:2>","versionNumber":2,"parentResourceId":"<ocid:1>"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		networkfirewallsdk.VaultMappedSecret,
		networkfirewallsdk.CreateVaultMappedSecretDetails,
		networkfirewallsdk.UpdateVaultMappedSecretDetails,
	]{
		CollectionPath: "/20230501/networkFirewallPolicies/<ocid:1>/mappedSecrets",
		ItemPath:       "/20230501/networkFirewallPolicies/<ocid:1>/mappedSecrets/osok_mock_mapped_secret",
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:   &createdState, UpdatedState: &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "source", "OCI_VAULT", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "source", "OCI_VAULT", updateRequest)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://network-firewall.mock.invalid", BasePath: "20230501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := networkfirewallsdk.NetworkFirewallClient{BaseClient: session.BaseClient()}
	manager := &MappedSecretServiceManager{}
	hooks := newMappedSecretRuntimeHooks(manager, sdkClient)
	client := wrapMappedSecretGeneratedClient(hooks, defaultMappedSecretServiceClient{ServiceClient: generatedruntime.NewServiceClient[*networkfirewallv1beta1.MappedSecret](buildMappedSecretGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*networkfirewallv1beta1.MappedSecret]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *networkfirewallv1beta1.MappedSecret) error {
			if current.Status.ParentResourceId != current.Spec.NetworkFirewallPolicyId || current.Status.Name != current.Spec.Name || current.Status.VersionNumber != 1 {
				return fmt.Errorf("created MappedSecret status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *networkfirewallv1beta1.MappedSecret) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *networkfirewallv1beta1.MappedSecret) error {
			if current.Status.VersionNumber != current.Spec.VersionNumber || !reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated MappedSecret status = %+v", current.Status)
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
