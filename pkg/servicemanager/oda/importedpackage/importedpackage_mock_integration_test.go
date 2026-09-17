/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package importedpackage

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func TestMockIntegrationImportedPackageCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := makeImportedPackageResource("<ocid:1>", "<ocid:2>")
	ocimock.InitializeResource(resource, "mock-imported-package")
	resource.Spec.ParameterValues = map[string]string{"mode": "create"}
	updatedSpec := resource.Spec
	updatedSpec.ParameterValues = map[string]string{"mode": "update"}
	updatedSpec.FreeformTags = map[string]string{"env": "prod"}
	createRequest := odasdk.CreateImportedPackageDetails{CurrentPackageId: stringPtrImported("<ocid:2>"), ParameterValues: map[string]string{"mode": "create"}}
	created := makeSDKImportedPackage("<ocid:1>", "<ocid:2>", resource, odasdk.ImportedPackageStatusReady)
	updateRequest := odasdk.UpdateImportedPackageDetails{CurrentPackageId: stringPtrImported("<ocid:2>"), ParameterValues: map[string]string{"mode": "update"}, FreeformTags: map[string]string{"env": "prod"}}
	updatedResource := resource.DeepCopy()
	updatedResource.Spec = updatedSpec
	updated := makeSDKImportedPackage("<ocid:1>", "<ocid:2>", updatedResource, odasdk.ImportedPackageStatusReady)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[odasdk.ImportedPackage, odasdk.CreateImportedPackageDetails, odasdk.UpdateImportedPackageDetails]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/importedPackages", ItemPath: "/20190506/odaInstances/<ocid:1>/importedPackages/<ocid:2>",
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, CreateRequest: &createRequest, CreatedState: &created,
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		UpdateRequest: &updateRequest, UpdatedState: &updated, UpdatedReadStates: []odasdk.ImportedPackage{updated}, ListShape: ocimock.ListShapeArray,
		DeleteEndsNotFound: true, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, CreateStatus: http.StatusCreated, DeleteStatus: http.StatusNoContent, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oda-package.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newTestImportedPackageClient(odasdk.OdapackageClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.ImportedPackage]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.ImportedPackage) error {
			if string(current.Status.OsokStatus.Ocid) != "<ocid:2>" || current.Status.CurrentPackageId != current.Spec.CurrentPackageId || !reflect.DeepEqual(current.Status.ParameterValues, current.Spec.ParameterValues) {
				return fmt.Errorf("created ImportedPackage status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.ImportedPackage) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.ImportedPackage) error {
			if current.Status.CurrentPackageId != current.Spec.CurrentPackageId || !reflect.DeepEqual(current.Status.ParameterValues, current.Spec.ParameterValues) || current.Status.FreeformTags["env"] != "prod" {
				return fmt.Errorf("updated ImportedPackage status = %+v", current.Status)
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

func stringPtrImported(value string) *string { return &value }
