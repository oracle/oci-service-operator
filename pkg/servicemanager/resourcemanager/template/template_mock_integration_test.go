/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package template

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourcemanagersdk "github.com/oracle/oci-go-sdk/v65/resourcemanager"
	resourcemanagerv1beta1 "github.com/oracle/oci-service-operator/api/resourcemanager/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockTemplateID = "ocid1.ormtemplate.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/resourcemanager
//   - reviewed runtime semantics: template_runtime_semantics.go
//
// The pinned Terraform provider exposes no Template resource.
func TestMockIntegrationTemplateImmediateCRUD(t *testing.T) {
	t.Parallel()

	resource := &resourcemanagerv1beta1.Template{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-template", Namespace: "default", UID: types.UID("mock-template-uid")},
		Spec: resourcemanagerv1beta1.TemplateSpec{
			CompartmentId:   "ocid1.compartment.oc1..mock",
			DisplayName:     "mock-template",
			Description:     "mock create",
			LongDescription: "mock long description",
			TemplateConfigSource: resourcemanagerv1beta1.TemplateConfigSource{
				TemplateConfigSourceType: "ZIP_UPLOAD",
				ZipFileBase64Encoded:     mockTemplateZip(t),
			},
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newTemplateMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://resourcemanager.mock.invalid", BasePath: "20180917", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Template OCI mock: %v", err)
		}
	})

	sdkClient := resourcemanagersdk.ResourceManagerClient{BaseClient: session.BaseClient()}
	manager := &TemplateServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTemplateRuntimeHooks(manager, sdkClient)
	client := wrapTemplateGeneratedClient(hooks, defaultTemplateServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourcemanagerv1beta1.Template](buildTemplateGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*resourcemanagerv1beta1.Template]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *resourcemanagerv1beta1.Template) error {
			if current.Status.Id != mockTemplateID ||
				current.Status.DisplayName != "mock-template" ||
				current.Status.LifecycleState != string(resourcemanagersdk.TemplateLifecycleStateActive) {
				return fmt.Errorf("created Template status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *resourcemanagerv1beta1.Template) {
			current.Spec.DisplayName = "mock-template-updated"
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *resourcemanagerv1beta1.Template) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.Description != current.Spec.Description ||
				current.Status.LongDescription != "mock long description" ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Template status = %+v", current.Status)
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

func newTemplateMockResponder(resource *resourcemanagerv1beta1.Template) (*ocimock.CRUDResponder[resourcemanagersdk.Template], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[resourcemanagersdk.Template]{
		CollectionPath:     "/20180917/templates",
		ItemPath:           "/20180917/templates/" + mockTemplateID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		Create: func(request ocimock.Request) (resourcemanagersdk.Template, ocimock.Response, error) {
			var details resourcemanagersdk.CreateTemplateDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return resourcemanagersdk.Template{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return resourcemanagersdk.Template{}, ocimock.Response{}, err
			}
			source, ok := details.TemplateConfigSource.(resourcemanagersdk.CreateTemplateZipUploadConfigSourceDetails)
			if !ok || source.ZipFileBase64Encoded == nil || *source.ZipFileBase64Encoded != resource.Spec.TemplateConfigSource.ZipFileBase64Encoded ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return resourcemanagersdk.Template{}, ocimock.Response{}, fmt.Errorf("unexpected create Template details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return resourcemanagersdk.Template{}, ocimock.Response{}, fmt.Errorf("create Template opc-retry-token is empty")
			}
			state := resourcemanagersdk.Template{
				Id:                   common.String(mockTemplateID),
				CompartmentId:        details.CompartmentId,
				CategoryId:           common.String("3"),
				DisplayName:          details.DisplayName,
				Description:          details.Description,
				LongDescription:      details.LongDescription,
				TemplateConfigSource: resourcemanagersdk.TemplateZipUploadConfigSource{},
				FreeformTags:         details.FreeformTags,
				TimeCreated:          &createdAt,
				LifecycleState:       resourcemanagersdk.TemplateLifecycleStateActive,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state resourcemanagersdk.Template) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state resourcemanagersdk.Template) (resourcemanagersdk.Template, ocimock.Response, error) {
			var details resourcemanagersdk.UpdateTemplateDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return resourcemanagersdk.Template{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-template-updated" ||
				details.Description == nil || *details.Description != "mock update" ||
				details.FreeformTags["osok-mock"] != "update" {
				return resourcemanagersdk.Template{}, ocimock.Response{}, fmt.Errorf("unexpected update Template details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.Description = details.Description
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ resourcemanagersdk.Template) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
