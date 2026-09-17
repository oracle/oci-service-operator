/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securityattributenamespace

import (
	"context"
	"fmt"
	securityattributesdk "github.com/oracle/oci-go-sdk/v65/securityattribute"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationSecurityAttributeNamespaceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &securityattributev1beta1.SecurityAttributeNamespace{Spec: securityattributev1beta1.SecurityAttributeNamespaceSpec{
		CompartmentId: "ocid1.tenancy.oc1..mock", Name: mockSecurityAttributeNamespaceName, Description: "synthetic create", FreeformTags: map[string]string{"osok-mock": "synthetic"},
	}}
	ocimock.InitializeResource(resource, "mock-securityattributenamespace")
	resource.Spec = ocimock.MustJSONFixture[securityattributev1beta1.SecurityAttributeNamespaceSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic create",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "name": "osok_mock_security_namespace_v1"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "synthetic update"
}`)
	createRequest := ocimock.MustJSONFixture[securityattributesdk.CreateSecurityAttributeNamespaceDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic create",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "name": "osok_mock_security_namespace_v1"
}`)
	createdState := ocimock.MustOCIResponseFixture[securityattributesdk.SecurityAttributeNamespace](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic create",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "mode": [
    "ENFORCE"
  ],
  "name": "osok_mock_security_namespace_v1"
}`)
	updateRequest := ocimock.MustJSONFixture[securityattributesdk.UpdateSecurityAttributeNamespaceDetails](t, `{
  "description": "synthetic update"
}`)
	updatedState := ocimock.MustOCIResponseFixture[securityattributesdk.SecurityAttributeNamespace](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic update",
  "freeformTags": {
    "osok-mock": "synthetic"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "mode": [
    "ENFORCE"
  ],
  "name": "osok_mock_security_namespace_v1"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		securityattributesdk.SecurityAttributeNamespace,
		securityattributesdk.CreateSecurityAttributeNamespaceDetails,
		securityattributesdk.UpdateSecurityAttributeNamespaceDetails,
	]{
		CollectionPath:    "/20240815/securityAttributeNamespaces",
		ItemPath:          "/20240815/securityAttributeNamespaces/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ securityattributesdk.CreateSecurityAttributeNamespaceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ securityattributesdk.SecurityAttributeNamespace) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20240815", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SecurityAttributeNamespace OCI mock: %v", err)
		}
	})
	sdkClient := securityattributesdk.
		SecurityAttributeClient{BaseClient: session.BaseClient()}
	client := newSecurityAttributeNamespaceServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*securityattributev1beta1.SecurityAttributeNamespace]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *securityattributev1beta1.SecurityAttributeNamespace) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created SecurityAttributeNamespace status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *securityattributev1beta1.SecurityAttributeNamespace) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *securityattributev1beta1.SecurityAttributeNamespace) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated SecurityAttributeNamespace status = %+v", current.Status)
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
