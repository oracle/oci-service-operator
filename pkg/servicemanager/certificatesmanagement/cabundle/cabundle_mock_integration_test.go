/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package cabundle

import (
	"context"
	"fmt"
	certificatesmanagementsdk "github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	certificatesmanagementv1beta1 "github.com/oracle/oci-service-operator/api/certificatesmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationCaBundleEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &certificatesmanagementv1beta1.CaBundle{}
	ocimock.InitializeResource(resource, "mock-cabundle")
	resource.Spec = ocimock.MustJSONFixture[certificatesmanagementv1beta1.CaBundleSpec](t, `{
  "caBundlePem": "-----BEGIN CERTIFICATE-----\nMIICuDCCAaACCQCswlehgd76VjANBgkqhkiG9w0BAQsFADAeMRwwGgYDVQQDDBNv\nc29rLXJlcGxheS5pbnZhbGlkMB4XDTI2MDkwMTA1NDgyNVoXDTM2MDgyOTA1NDgy\nNVowHjEcMBoGA1UEAwwTb3Nvay1yZXBsYXkuaW52YWxpZDCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBALzZjMNUR5SvwVaOFTw8eDJSMcdapOu7D+MQw2MS\nA+1YAF5OQ3gwoMPmTLrGBte9hxWv2N/gYyM5tyDhoBsRKbHPrDz03Z5AEz/riGWp\nYwpoTP/zcQ1SQzbV5p/+O2MvbyzuVZ5kixuFZh78PrjBDkcMGNH0uA0NmdH9lKii\nROOFd0aKMCA5L/dn8ESLzUekNghcKuUAu2Gc61J0v+0/3LOe7aPH0v28ossf2ACu\nTmLoIGirLaOJynksSpQIG0SVA93oc2CyltFxLinx6Ec0NKb/7qUyqwRYJatSpQcK\nGUFPBDs0+reOI4n7tuyhvBBiXrrqbkMJc8WyPs8ClXJlzb0CAwEAATANBgkqhkiG\n9w0BAQsFAAOCAQEALCaJLZARLaB9PpH4JURzQV9Xe/3r+2aWjuneSILxxUTvyJ61\nj0i59UVbmCaA8hZXt7QhU3D3K8mJoTBMGAdvYSjLm+scYB8V6MbvJ5kHA7bGIvZL\nT+qcCWe0ElyYnvoLpqbki9JJI/zgcXYDY5BwypLhpNykxPQeGjqg88TszrRi7o1C\nSXIaSw21TuZVNm42tZPkWBqKWPECONg1i9pkoIDZVNaUt3z4nP6LpRAMrNWHoJQW\nfovQtpvBhD2r6F7PHqfkymKTMs+76niaMQV13iQKEFLgaaYbkVYk52eUmjMGAGoL\n3TftBbW3NhwflEB5+PAD/jpg5EvXLoQINFdW+w==\n-----END CERTIFICATE-----",
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "name": "osok-mock-common-ca-bundle-v1"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  }
}`)
	createRequest := ocimock.MustJSONFixture[certificatesmanagementsdk.CreateCaBundleDetails](t, `{
  "caBundlePem": "-----BEGIN CERTIFICATE-----\nMIICuDCCAaACCQCswlehgd76VjANBgkqhkiG9w0BAQsFADAeMRwwGgYDVQQDDBNv\nc29rLXJlcGxheS5pbnZhbGlkMB4XDTI2MDkwMTA1NDgyNVoXDTM2MDgyOTA1NDgy\nNVowHjEcMBoGA1UEAwwTb3Nvay1yZXBsYXkuaW52YWxpZDCCASIwDQYJKoZIhvcN\nAQEBBQADggEPADCCAQoCggEBALzZjMNUR5SvwVaOFTw8eDJSMcdapOu7D+MQw2MS\nA+1YAF5OQ3gwoMPmTLrGBte9hxWv2N/gYyM5tyDhoBsRKbHPrDz03Z5AEz/riGWp\nYwpoTP/zcQ1SQzbV5p/+O2MvbyzuVZ5kixuFZh78PrjBDkcMGNH0uA0NmdH9lKii\nROOFd0aKMCA5L/dn8ESLzUekNghcKuUAu2Gc61J0v+0/3LOe7aPH0v28ossf2ACu\nTmLoIGirLaOJynksSpQIG0SVA93oc2CyltFxLinx6Ec0NKb/7qUyqwRYJatSpQcK\nGUFPBDs0+reOI4n7tuyhvBBiXrrqbkMJc8WyPs8ClXJlzb0CAwEAATANBgkqhkiG\n9w0BAQsFAAOCAQEALCaJLZARLaB9PpH4JURzQV9Xe/3r+2aWjuneSILxxUTvyJ61\nj0i59UVbmCaA8hZXt7QhU3D3K8mJoTBMGAdvYSjLm+scYB8V6MbvJ5kHA7bGIvZL\nT+qcCWe0ElyYnvoLpqbki9JJI/zgcXYDY5BwypLhpNykxPQeGjqg88TszrRi7o1C\nSXIaSw21TuZVNm42tZPkWBqKWPECONg1i9pkoIDZVNaUt3z4nP6LpRAMrNWHoJQW\nfovQtpvBhD2r6F7PHqfkymKTMs+76niaMQV13iQKEFLgaaYbkVYk52eUmjMGAGoL\n3TftBbW3NhwflEB5+PAD/jpg5EvXLoQINFdW+w==\n-----END CERTIFICATE-----",
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "name": "osok-mock-common-ca-bundle-v1"
}`)
	createdState := ocimock.MustOCIResponseFixture[certificatesmanagementsdk.CaBundle](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-common-ca-bundle-v1",
  "timeCreated": "2026-09-01T05:56:00.996Z"
}`)
	createdReadStates := []certificatesmanagementsdk.CaBundle{
		ocimock.MustOCIResponseFixture[certificatesmanagementsdk.CaBundle](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-common-ca-bundle-v1",
  "timeCreated": "2026-09-01T05:56:00.996Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[certificatesmanagementsdk.UpdateCaBundleDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[certificatesmanagementsdk.CaBundle](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-common-ca-bundle-v1",
  "timeCreated": "2026-09-01T05:56:00.996Z"
}`)
	updatedReadStates := []certificatesmanagementsdk.CaBundle{
		ocimock.MustOCIResponseFixture[certificatesmanagementsdk.CaBundle](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-common-ca-bundle-v1",
  "timeCreated": "2026-09-01T05:56:00.996Z"
}`),
	}
	deletedReadStates := []certificatesmanagementsdk.CaBundle{
		ocimock.MustOCIResponseFixture[certificatesmanagementsdk.CaBundle](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETING",
  "name": "osok-mock-common-ca-bundle-v1",
  "timeCreated": "2026-09-01T05:56:00.996Z"
}`),
		ocimock.MustOCIResponseFixture[certificatesmanagementsdk.CaBundle](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update",
    "osokCaBundlePemSha256": "eb67e294add6b04ce0798f3c28492f41be14a5abcbe27a1417743a411e984628"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "name": "osok-mock-common-ca-bundle-v1",
  "timeCreated": "2026-09-01T05:56:00.996Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		certificatesmanagementsdk.CaBundle,
		certificatesmanagementsdk.CreateCaBundleDetails,
		certificatesmanagementsdk.UpdateCaBundleDetails,
	]{
		CollectionPath:    "/20210224/caBundles",
		ItemPath:          "/20210224/caBundles/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ certificatesmanagementsdk.CreateCaBundleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ certificatesmanagementsdk.CaBundle) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210224", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close CaBundle OCI mock: %v", err)
		}
	})
	sdkClient := certificatesmanagementsdk.CertificatesManagementClient{BaseClient: session.BaseClient()}
	client := newCaBundleServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*certificatesmanagementv1beta1.CaBundle]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *certificatesmanagementv1beta1.CaBundle) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created CaBundle status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *certificatesmanagementv1beta1.CaBundle) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *certificatesmanagementv1beta1.CaBundle) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated CaBundle status = %+v", current.Status)
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
