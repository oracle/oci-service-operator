/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package containerimagesignature

import (
	"context"
	"fmt"
	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationContainerImageSignatureLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newContainerImageSignatureResource()
	ocimock.InitializeResource(resource, "mock-containerimagesignature")
	resource.Spec = ocimock.MustJSONFixture[artifactsv1beta1.ContainerImageSignatureSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "imageId": "\u003cocid:2\u003e",
  "kmsKeyId": "\u003cocid:3\u003e",
  "kmsKeyVersionId": "\u003cocid:4\u003e",
  "message": "signed-message",
  "signature": "signature-payload",
  "signingAlgorithm": "SHA_256_RSA_PKCS_PSS"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[artifactssdk.CreateContainerImageSignatureDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "imageId": "\u003cocid:2\u003e",
  "kmsKeyId": "\u003cocid:3\u003e",
  "kmsKeyVersionId": "\u003cocid:4\u003e",
  "message": "signed-message",
  "signature": "signature-payload",
  "signingAlgorithm": "SHA_256_RSA_PKCS_PSS"
}`)
	createdState := ocimock.MustOCIResponseFixture[artifactssdk.ContainerImageSignature](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:5\u003e",
  "imageId": "\u003cocid:2\u003e",
  "kmsKeyId": "\u003cocid:3\u003e",
  "kmsKeyVersionId": "\u003cocid:4\u003e",
  "lifecycleState": "AVAILABLE",
  "message": "signed-message",
  "signature": "signature-payload",
  "signingAlgorithm": "SHA_256_RSA_PKCS_PSS"
}`)
	updateRequest := ocimock.MustJSONFixture[artifactssdk.UpdateContainerImageSignatureDetails](t, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[artifactssdk.ContainerImageSignature](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:5\u003e",
  "imageId": "\u003cocid:2\u003e",
  "kmsKeyId": "\u003cocid:3\u003e",
  "kmsKeyVersionId": "\u003cocid:4\u003e",
  "lifecycleState": "AVAILABLE",
  "message": "signed-message",
  "signature": "signature-payload",
  "signingAlgorithm": "SHA_256_RSA_PKCS_PSS"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		artifactssdk.ContainerImageSignature,
		artifactssdk.CreateContainerImageSignatureDetails,
		artifactssdk.UpdateContainerImageSignatureDetails,
	]{
		CollectionPath:    "/20160918/container/imageSignatures",
		ItemPath:          "/20160918/container/imageSignatures/<ocid:5>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ artifactssdk.CreateContainerImageSignatureDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ artifactssdk.ContainerImageSignature) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ContainerImageSignature OCI mock: %v", err)
		}
	})
	sdkClient := artifactssdk.ArtifactsClient{BaseClient: session.BaseClient()}
	manager := &ContainerImageSignatureServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newContainerImageSignatureDefaultRuntimeHooks(sdkClient)
	applyContainerImageSignatureRuntimeHooks(&hooks)
	client := wrapContainerImageSignatureGeneratedClient(hooks, defaultContainerImageSignatureServiceClient{ServiceClient: generatedruntime.NewServiceClient[*artifactsv1beta1.ContainerImageSignature](buildContainerImageSignatureGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*artifactsv1beta1.ContainerImageSignature]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *artifactsv1beta1.ContainerImageSignature) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "AVAILABLE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ImageId, current.Spec.ImageId) ||
				!reflect.DeepEqual(current.Status.KmsKeyId, current.Spec.KmsKeyId) ||
				!reflect.DeepEqual(current.Status.KmsKeyVersionId, current.Spec.KmsKeyVersionId) ||
				!reflect.DeepEqual(current.Status.Message, current.Spec.Message) ||
				!reflect.DeepEqual(current.Status.Signature, current.Spec.Signature) ||
				!reflect.DeepEqual(current.Status.SigningAlgorithm, current.Spec.SigningAlgorithm) {
				return fmt.Errorf("created ContainerImageSignature status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *artifactsv1beta1.ContainerImageSignature) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *artifactsv1beta1.ContainerImageSignature) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "AVAILABLE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated ContainerImageSignature status = %+v", current.Status)
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
