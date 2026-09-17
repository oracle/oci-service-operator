/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package pattern

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

func TestMockIntegrationPatternCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &datacatalogv1beta1.Pattern{}
	ocimock.InitializeResource(resource, "mock-pattern")
	resource.Spec = ocimock.MustJSONFixture[datacatalogv1beta1.PatternSpec](t, `{
  "catalogId": "<ocid:1>",
  "displayName": "pattern create",
  "description": "create",
  "expression": ".*"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[datacatalogsdk.CreatePatternDetails](t, `{
  "displayName": "pattern create",
  "description": "create",
  "expression": ".*"
}`)
	updateRequest := ocimock.MustJSONFixture[datacatalogsdk.UpdatePatternDetails](t, `{
  "description": "updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[datacatalogsdk.Pattern](t, `{
  "catalogId": "<ocid:1>",
  "displayName": "pattern create",
  "description": "create",
  "expression": ".*",
  "key": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datacatalogsdk.Pattern](t, `{
  "catalogId": "<ocid:1>",
  "displayName": "pattern create",
  "description": "updated",
  "expression": ".*",
  "key": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datacatalogsdk.Pattern, datacatalogsdk.CreatePatternDetails, datacatalogsdk.UpdatePatternDetails]{
		CollectionPath: "/20190325/catalogs/<ocid:1>/patterns", ItemPath: "/20190325/catalogs/<ocid:1>/patterns/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ datacatalogsdk.CreatePatternDetails) error {
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
	manager := &PatternServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newPatternRuntimeHooks(manager, sdkClient)
	client := wrapPatternGeneratedClient(hooks, defaultPatternServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datacatalogv1beta1.Pattern](buildPatternGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datacatalogv1beta1.Pattern]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datacatalogv1beta1.Pattern) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Pattern status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datacatalogv1beta1.Pattern) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datacatalogv1beta1.Pattern) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Pattern status = %+v", current.Status)
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
