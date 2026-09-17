/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package glossary

import (
	"context"
	"fmt"
	"testing"

	datacatalogsdk "github.com/oracle/oci-go-sdk/v65/datacatalog"
	datacatalogv1beta1 "github.com/oracle/oci-service-operator/api/datacatalog/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationGlossaryCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &datacatalogv1beta1.Glossary{}
	ocimock.InitializeResource(resource, "mock-glossary")
	resource.Spec = ocimock.MustJSONFixture[datacatalogv1beta1.GlossarySpec](t, `{
  "catalogId": "<ocid:1>",
  "displayName": "glossary create",
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[datacatalogsdk.CreateGlossaryDetails](t, `{
  "displayName": "glossary create",
  "description": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[datacatalogsdk.UpdateGlossaryDetails](t, `{
  "description": "updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[datacatalogsdk.Glossary](t, `{
  "catalogId": "<ocid:1>",
  "displayName": "glossary create",
  "description": "create",
  "key": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datacatalogsdk.Glossary](t, `{
  "catalogId": "<ocid:1>",
  "displayName": "glossary create",
  "description": "updated",
  "key": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datacatalogsdk.Glossary, datacatalogsdk.CreateGlossaryDetails, datacatalogsdk.UpdateGlossaryDetails]{
		CollectionPath: "/20190325/catalogs/<ocid:1>/glossaries", ItemPath: "/20190325/catalogs/<ocid:1>/glossaries/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datacatalogsdk.CreateGlossaryDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datacatalog.mock.invalid", BasePath: "20190325", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := datacatalogsdk.DataCatalogClient{BaseClient: session.BaseClient()}
	manager := &GlossaryServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newGlossaryRuntimeHooks(manager, sdkClient)
	client := wrapGlossaryGeneratedClient(hooks, defaultGlossaryServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datacatalogv1beta1.Glossary](buildGlossaryGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datacatalogv1beta1.Glossary]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datacatalogv1beta1.Glossary) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Glossary status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datacatalogv1beta1.Glossary) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datacatalogv1beta1.Glossary) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Glossary status = %+v", current.Status)
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
