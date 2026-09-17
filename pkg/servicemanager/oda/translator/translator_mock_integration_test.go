/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package translator

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationTranslatorCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := makeTranslatorResource()
	ocimock.InitializeResource(resource, "mock-translator")
	resource.Annotations[translatorOdaInstanceIDAnnotation] = "<ocid:1>"
	updatedSpec := resource.Spec
	updatedSpec.BaseUrl = "https://translation.example.com/v2"
	updatedSpec.FreeformTags = map[string]string{"env": "prod"}
	createRequest := odasdk.CreateTranslatorDetails{Type: odasdk.TranslationServiceGoogle, BaseUrl: common.String(resource.Spec.BaseUrl), AuthToken: common.String(resource.Spec.AuthToken)}
	created := makeSDKTranslator("<ocid:2>", resource, odasdk.LifecycleStateActive)
	updateRequest := odasdk.UpdateTranslatorDetails{BaseUrl: common.String(updatedSpec.BaseUrl), AuthToken: common.String(updatedSpec.AuthToken), FreeformTags: map[string]string{"env": "prod"}}
	updatedResource := resource.DeepCopy()
	updatedResource.Spec = updatedSpec
	updated := makeSDKTranslator("<ocid:2>", updatedResource, odasdk.LifecycleStateActive)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[odasdk.Translator, odasdk.CreateTranslatorDetails, odasdk.UpdateTranslatorDetails]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/translators", ItemPath: "/20190506/odaInstances/<ocid:1>/translators/<ocid:2>",
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, CreateRequest: &createRequest, CreatedState: &created, ListShape: ocimock.ListShapeItems,
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		UpdateRequest: &updateRequest, UpdatedState: &updated, UpdatedReadStates: []odasdk.Translator{updated}, DeleteEndsNotFound: true,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, CreateStatus: http.StatusCreated, DeleteStatus: http.StatusNoContent, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oda.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, odasdk.ManagementClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.Translator]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.Translator) error {
			if current.Status.Id != "<ocid:2>" || current.Status.Type != current.Spec.Type || current.Status.BaseUrl != current.Spec.BaseUrl || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created Translator status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.Translator) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.Translator) error {
			if current.Status.BaseUrl != current.Spec.BaseUrl || current.Status.FreeformTags["env"] != "prod" || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("updated Translator status = %+v", current.Status)
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
