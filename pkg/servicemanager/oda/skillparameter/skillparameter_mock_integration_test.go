/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package skillparameter

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

func TestMockIntegrationSkillParameterCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := testSkillParameter()
	ocimock.InitializeResource(resource, "mock-skill-parameter")
	updatedSpec := resource.Spec
	updatedSpec.DisplayName = "Updated Display"
	updatedSpec.Description = "updated description"
	updatedSpec.Value = "updated value"
	createRequest := odasdk.CreateSkillParameterDetails{Name: common.String(resource.Spec.Name), DisplayName: common.String(resource.Spec.DisplayName), Type: odasdk.ParameterTypeEnum(resource.Spec.Type), Value: common.String(resource.Spec.Value), Description: common.String(resource.Spec.Description)}
	created := skillParameter(resource.Spec.Name, resource.Spec.DisplayName, resource.Spec.Type, resource.Spec.Value, odasdk.LifecycleStateActive, resource.Spec.Description)
	updateRequest := odasdk.UpdateSkillParameterDetails{DisplayName: common.String(updatedSpec.DisplayName), Value: common.String(updatedSpec.Value), Description: common.String(updatedSpec.Description)}
	updated := skillParameter(updatedSpec.Name, updatedSpec.DisplayName, updatedSpec.Type, updatedSpec.Value, odasdk.LifecycleStateActive, updatedSpec.Description)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[odasdk.SkillParameter, odasdk.CreateSkillParameterDetails, odasdk.UpdateSkillParameterDetails]{
		CollectionPath: "/20190506/odaInstances/oda-1/skills/skill-1/parameters", ItemPath: "/20190506/odaInstances/oda-1/skills/skill-1/parameters/param",
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, CreateRequest: &createRequest, CreatedState: &created, ListShape: ocimock.ListShapeItems,
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		UpdateRequest: &updateRequest, UpdatedState: &updated, UpdatedReadStates: []odasdk.SkillParameter{updated}, DeleteEndsNotFound: true,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteStatus: http.StatusNoContent, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oda.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newSkillParameterServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, odasdk.ManagementClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.SkillParameter]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.SkillParameter) error {
			if current.Status.Name != current.Spec.Name || current.Status.DisplayName != current.Spec.DisplayName || current.Status.Value != current.Spec.Value || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created SkillParameter status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.SkillParameter) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.SkillParameter) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description || current.Status.Value != current.Spec.Value {
				return fmt.Errorf("updated SkillParameter status = %+v", current.Status)
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
