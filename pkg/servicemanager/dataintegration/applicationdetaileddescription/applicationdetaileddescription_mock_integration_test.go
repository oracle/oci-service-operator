/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package applicationdetaileddescription

import (
	"context"
	"fmt"
	"testing"

	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationApplicationDetailedDescriptionCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.ApplicationDetailedDescription{}
	ocimock.InitializeResource(resource, "mock-applicationdetaileddescription")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.ApplicationDetailedDescriptionSpec](t, `{
  "workspaceId": "<ocid:1>",
  "applicationKey": "application-key",
  "logo": "bG9nbw==",
  "detailedDescription": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"detailedDescription":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateDetailedDescriptionDetails](t, `{
  "logo": "bG9nbw==",
  "detailedDescription": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateDetailedDescriptionDetails](t, `{
  "detailedDescription": "updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.DetailedDescription](t, `{
  "logo": "bG9nbw==",
  "detailedDescription": "create"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.DetailedDescription](t, `{
  "logo": "bG9nbw==",
  "detailedDescription": "updated"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.DetailedDescription, dataintegrationsdk.CreateDetailedDescriptionDetails, dataintegrationsdk.UpdateDetailedDescriptionDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/applications/application-key/detailedDescription", ItemPath: "/20200430/workspaces/<ocid:1>/applications/application-key/detailedDescription",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeNone,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateDetailedDescriptionDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dataintegration.mock.invalid", BasePath: "20200430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := dataintegrationsdk.DataIntegrationClient{BaseClient: session.BaseClient()}
	manager := &ApplicationDetailedDescriptionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newApplicationDetailedDescriptionRuntimeHooks(manager, sdkClient)
	client := wrapApplicationDetailedDescriptionGeneratedClient(hooks, defaultApplicationDetailedDescriptionServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.ApplicationDetailedDescription](buildApplicationDetailedDescriptionGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.ApplicationDetailedDescription]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.ApplicationDetailedDescription) error {
			if current.Status.OsokStatus.Ocid != "<ocid:1>/applications/application-key/detailedDescription" ||
				current.Status.OsokStatus.Reason != string(shared.Active) || current.Status.Logo != "bG9nbw==" ||
				current.Status.DetailedDescription != "create" {
				return fmt.Errorf("created ApplicationDetailedDescription status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.ApplicationDetailedDescription) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.ApplicationDetailedDescription) error {
			if current.Status.OsokStatus.Ocid != "<ocid:1>/applications/application-key/detailedDescription" ||
				current.Status.OsokStatus.Reason != string(shared.Active) || current.Status.DetailedDescription != "updated" {
				return fmt.Errorf("updated ApplicationDetailedDescription status = %+v", current.Status)
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
