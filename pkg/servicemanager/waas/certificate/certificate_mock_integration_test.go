/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package certificate

import (
	"context"
	"fmt"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationCertificateEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &waasv1beta1.Certificate{}
	ocimock.InitializeResource(resource, "mock-certificate")
	resource.Spec = ocimock.MustJSONFixture[waasv1beta1.CertificateSpec](t, `{
  "certificateData": "\u003cbinding:certificate\u003e",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waas-certificate",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isTrustVerificationDisabled": true,
  "privateKeyData": "\u003credacted\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-waas-certificate-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[waassdk.CreateCertificateDetails](t, `{
  "certificateData": "\u003cbinding:certificate\u003e",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waas-certificate",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isTrustVerificationDisabled": true,
  "privateKeyData": "\u003credacted\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[waassdk.Certificate](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:36:03.579Z"
    }
  },
  "displayName": "osok-mock-waas-certificate",
  "extensions": null,
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "issuedBy": "osok-mock-waas.example.com",
  "issuerName": {
    "commonName": "osok-mock-waas.example.com"
  },
  "lifecycleState": "ACTIVE",
  "publicKeyInfo": {
    "algorithm": "RSA",
    "exponent": 65537,
    "keySize": 2048
  },
  "serialNumber": "13993365216209283207",
  "signatureAlgorithm": "",
  "subjectName": {
    "commonName": "osok-mock-waas.example.com"
  },
  "timeCreated": "2026-09-02T03:36:04.193Z",
  "timeNotValidAfter": "2026-09-04T03:31:27.000Z",
  "timeNotValidBefore": "2026-09-02T03:31:27.000Z",
  "version": 0
}`)
	updateRequest := ocimock.MustJSONFixture[waassdk.UpdateCertificateDetails](t, `{
  "displayName": "osok-mock-waas-certificate-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[waassdk.Certificate](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:36:03.579Z"
    }
  },
  "displayName": "osok-mock-waas-certificate-updated",
  "extensions": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "issuedBy": "osok-mock-waas.example.com",
  "issuerName": {
    "commonName": "osok-mock-waas.example.com"
  },
  "lifecycleState": "ACTIVE",
  "publicKeyInfo": {
    "algorithm": "RSA",
    "exponent": 65537,
    "keySize": 2048
  },
  "serialNumber": "13993365216209283207",
  "signatureAlgorithm": "",
  "subjectName": {
    "commonName": "osok-mock-waas.example.com"
  },
  "timeCreated": "2026-09-02T03:36:04.193Z",
  "timeNotValidAfter": "2026-09-04T03:31:27.000Z",
  "timeNotValidBefore": "2026-09-02T03:31:27.000Z",
  "version": 0
}`)
	deletedState := ocimock.MustOCIResponseFixture[waassdk.Certificate](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:36:03.579Z"
    }
  },
  "displayName": "osok-mock-waas-certificate-updated",
  "extensions": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "issuedBy": "osok-mock-waas.example.com",
  "issuerName": {
    "commonName": "osok-mock-waas.example.com"
  },
  "lifecycleState": "DELETED",
  "publicKeyInfo": {
    "algorithm": "RSA",
    "exponent": 65537,
    "keySize": 2048
  },
  "serialNumber": "13993365216209283207",
  "signatureAlgorithm": "",
  "subjectName": {
    "commonName": "osok-mock-waas.example.com"
  },
  "timeCreated": "2026-09-02T03:36:04.193Z",
  "timeNotValidAfter": "2026-09-04T03:31:27.000Z",
  "timeNotValidBefore": "2026-09-02T03:31:27.000Z",
  "version": 0
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		waassdk.Certificate,
		waassdk.CreateCertificateDetails,
		waassdk.UpdateCertificateDetails,
	]{
		CollectionPath:    "/20181116/certificates",
		ItemPath:          "/20181116/certificates/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ waassdk.CreateCertificateDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ waassdk.Certificate) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181116", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Certificate OCI mock: %v", err)
		}
	})
	sdkClient := waassdk.WaasClient{BaseClient: session.BaseClient()}
	client := newCertificateServiceClientWithOCIClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waasv1beta1.Certificate]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waasv1beta1.Certificate) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created Certificate status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *waasv1beta1.Certificate) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *waasv1beta1.Certificate) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated Certificate status = %+v", current.Status)
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
