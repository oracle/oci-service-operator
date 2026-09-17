/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package skill

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

func TestMockIntegrationSkillCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := testSkill()
	resource.Annotations[skillOdaInstanceIDAnnotation] = "<ocid:1>"
	updatedSpec := resource.Spec
	updatedSpec.Category = "updated-category"
	updatedSpec.Description = "updated description"
	updatedSpec.FreeformTags = map[string]string{"env": "test"}
	createDetails := odasdk.CreateNewSkillDetails{Name: common.String(resource.Spec.Name), DisplayName: common.String(resource.Spec.DisplayName), Version: common.String(resource.Spec.Version), Category: common.String(resource.Spec.Category), Description: common.String(resource.Spec.Description)}
	created := skillSDK("<ocid:2>", resource.Spec.Name, resource.Spec.Version, resource.Spec.DisplayName, odasdk.LifecycleStateActive, resource.Spec.Category, resource.Spec.Description)
	updateDetails := odasdk.UpdateSkillDetails{Category: common.String(updatedSpec.Category), Description: common.String(updatedSpec.Description), FreeformTags: updatedSpec.FreeformTags}
	updated := skillSDK("<ocid:2>", updatedSpec.Name, updatedSpec.Version, updatedSpec.DisplayName, odasdk.LifecycleStateActive, updatedSpec.Category, updatedSpec.Description)
	updated.FreeformTags = updatedSpec.FreeformTags

	responder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[odasdk.Skill]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/skills", ItemPath: "/20190506/odaInstances/<ocid:1>/skills/<ocid:2>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		AdditionalRoutes: []ocimock.Route{{Name: "create-work-request", Method: http.MethodGet, Path: "/20190506/workRequests/wr-skill-create", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, odasdk.WorkRequest{Id: common.String("wr-skill-create"), OdaInstanceId: common.String("<ocid:1>"), ResourceId: common.String("<ocid:2>"), RequestAction: odasdk.WorkRequestRequestActionCreateSkill, Status: odasdk.WorkRequestStatusSucceeded, PercentComplete: float32Ptr(100)})
		}}},
		List: func(_ ocimock.Request, present bool, state odasdk.Skill) (ocimock.Response, error) {
			items := []odasdk.Skill{}
			if present {
				items = append(items, state)
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": items})
		},
		Create: func(request ocimock.Request) (odasdk.Skill, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero odasdk.Skill
				return zero, ocimock.Response{}, err
			}
			if err := ocimock.ValidateDiscriminatedJSONRequest(request, "kind", "NEW", createDetails); err != nil {
				return odasdk.Skill{}, ocimock.Response{}, err
			}
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"wr-skill-create"}, "Opc-Request-Id": []string{"req-skill-create"}}
			return created, response, nil
		},
		Read: func(_ ocimock.Request, state odasdk.Skill) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, _ odasdk.Skill) (odasdk.Skill, ocimock.Response, error) {
			if err := ocimock.ValidateJSONRequest(request, updateDetails); err != nil {
				return odasdk.Skill{}, ocimock.Response{}, err
			}
			response, err := ocimock.JSONResponse(http.StatusOK, updated)
			return updated, response, err
		},
		Delete: func(_ ocimock.Request, _ odasdk.Skill) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oda.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	base := session.BaseClient()
	client := newSkillServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, skillSDKClients{management: odasdk.ManagementClient{BaseClient: base}, oda: odasdk.OdaClient{BaseClient: base}})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.Skill]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.Skill) error {
			if current.Status.Id != "<ocid:2>" || current.Status.Name != current.Spec.Name || current.Status.Version != current.Spec.Version || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created Skill status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.Skill) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.Skill) error {
			if current.Status.Category != current.Spec.Category || current.Status.Description != current.Spec.Description || current.Status.FreeformTags["env"] != "test" {
				return fmt.Errorf("updated Skill status = %+v", current.Status)
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
