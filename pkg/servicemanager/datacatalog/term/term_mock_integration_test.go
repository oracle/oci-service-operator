/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package term

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

func TestMockIntegrationTermCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &datacatalogv1beta1.Term{}
	ocimock.InitializeResource(resource, "mock-term")
	resource.Spec = ocimock.MustJSONFixture[datacatalogv1beta1.TermSpec](t, `{
  "catalogId": "<ocid:1>",
  "glossaryKey": "glossary-key",
  "displayName": "term create",
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[datacatalogsdk.CreateTermDetails](t, `{
  "displayName": "term create",
  "description": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[datacatalogsdk.UpdateTermDetails](t, `{
  "description": "updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[datacatalogsdk.Term](t, `{
  "glossaryKey": "glossary-key",
  "displayName": "term create",
  "description": "create",
  "key": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datacatalogsdk.Term](t, `{
  "glossaryKey": "glossary-key",
  "displayName": "term create",
  "description": "updated",
  "key": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datacatalogsdk.Term, datacatalogsdk.CreateTermDetails, datacatalogsdk.UpdateTermDetails]{
		CollectionPath: "/20190325/catalogs/<ocid:1>/glossaries/glossary-key/terms", ItemPath: "/20190325/catalogs/<ocid:1>/glossaries/glossary-key/terms/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datacatalogsdk.CreateTermDetails) error {
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
	manager := &TermServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTermRuntimeHooks(manager, sdkClient)
	client := wrapTermGeneratedClient(hooks, defaultTermServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datacatalogv1beta1.Term](buildTermGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datacatalogv1beta1.Term]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datacatalogv1beta1.Term) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Term status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datacatalogv1beta1.Term) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datacatalogv1beta1.Term) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Term status = %+v", current.Status)
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
