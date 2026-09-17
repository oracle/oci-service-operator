/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securityattribute

import (
	"context"
	"fmt"
	"testing"

	securityattributesdk "github.com/oracle/oci-go-sdk/v65/securityattribute"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Explicit typed service-manager lifecycle; mock evidence is authoring reference only.
func TestMockIntegrationSecurityAttributeCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &securityattributev1beta1.SecurityAttribute{}
	ocimock.InitializeResource(resource, "mock-security-attribute")
	resource.Spec = ocimock.MustJSONFixture[securityattributev1beta1.SecurityAttributeSpec](t, `{
  "securityAttributeNamespaceId":"<ocid:1>","name":"osok_mock_attribute","description":"mock create"
}`)
	updatedSpec := resource.Spec
	updatedSpec.Description = "mock update"
	updatedSpec.IsRetired = true
	createRequest := ocimock.MustJSONFixture[securityattributesdk.CreateSecurityAttributeDetails](t, `{
  "name":"osok_mock_attribute","description":"mock create"
}`)
	createdState := ocimock.MustOCIResponseFixture[securityattributesdk.SecurityAttribute](t, `{
  "id":"<ocid:2>","compartmentId":"<ocid:3>","securityAttributeNamespaceId":"<ocid:1>",
  "securityAttributeNamespaceName":"osok_mock_namespace","name":"osok_mock_attribute",
  "description":"mock create","isRetired":false,"lifecycleState":"ACTIVE","type":"STRING"
}`)
	updateRequest := ocimock.MustJSONFixture[securityattributesdk.UpdateSecurityAttributeDetails](t, `{
  "description":"mock update","isRetired":true
}`)
	updatedState := ocimock.MustOCIResponseFixture[securityattributesdk.SecurityAttribute](t, `{
  "id":"<ocid:2>","compartmentId":"<ocid:3>","securityAttributeNamespaceId":"<ocid:1>",
  "securityAttributeNamespaceName":"osok_mock_namespace","name":"osok_mock_attribute",
  "description":"mock update","isRetired":true,"lifecycleState":"ACTIVE","type":"STRING"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		securityattributesdk.SecurityAttribute,
		securityattributesdk.CreateSecurityAttributeDetails,
		securityattributesdk.UpdateSecurityAttributeDetails,
	]{
		CollectionPath: "/20240815/securityAttributeNamespaces/<ocid:1>/securityAttributes",
		ItemPath:       "/20240815/securityAttributeNamespaces/<ocid:1>/securityAttributes/osok_mock_attribute",
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest: &createRequest, CreatedState: &createdState, ListShape: ocimock.ListShapeArray,
		UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 200, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://security-attribute.mock.invalid", BasePath: "20240815", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := securityattributesdk.SecurityAttributeClient{BaseClient: session.BaseClient()}
	manager := &SecurityAttributeServiceManager{}
	hooks := newSecurityAttributeRuntimeHooks(manager, sdkClient)
	client := wrapSecurityAttributeGeneratedClient(hooks, defaultSecurityAttributeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*securityattributev1beta1.SecurityAttribute](buildSecurityAttributeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*securityattributev1beta1.SecurityAttribute]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *securityattributev1beta1.SecurityAttribute) error {
			if current.Status.Id != "<ocid:2>" || current.Status.SecurityAttributeNamespaceId != current.Spec.SecurityAttributeNamespaceId || current.Status.Name != current.Spec.Name || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("created SecurityAttribute status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *securityattributev1beta1.SecurityAttribute) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *securityattributev1beta1.SecurityAttribute) error {
			if current.Status.Description != current.Spec.Description || current.Status.IsRetired != current.Spec.IsRetired {
				return fmt.Errorf("updated SecurityAttribute status = %+v", current.Status)
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
