/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package tenancyattachment

import (
	"context"
	"fmt"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationTenancyAttachmentLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newTenancyAttachmentTestResource()
	ocimock.InitializeResource(resource, "mock-tenancyattachment")
	resource.Spec = ocimock.MustJSONFixture[resourceanalyticsv1beta1.TenancyAttachmentSpec](t, `{
  "description": "desired description",
  "resourceAnalyticsInstanceId": "\u003cocid:1\u003e",
  "tenancyId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "desired description-updated"
}`)
	createRequest := ocimock.MustJSONFixture[resourceanalyticssdk.CreateTenancyAttachmentDetails](t, `{
  "description": "desired description",
  "resourceAnalyticsInstanceId": "\u003cocid:1\u003e",
  "tenancyId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[resourceanalyticssdk.TenancyAttachment](t, `{
  "description": "desired description",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "resourceAnalyticsInstanceId": "\u003cocid:1\u003e",
  "tenancyId": "\u003cocid:2\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[resourceanalyticssdk.UpdateTenancyAttachmentDetails](t, `{
  "description": "desired description-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[resourceanalyticssdk.TenancyAttachment](t, `{
  "description": "desired description-updated",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "resourceAnalyticsInstanceId": "\u003cocid:1\u003e",
  "tenancyId": "\u003cocid:2\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		resourceanalyticssdk.TenancyAttachment,
		resourceanalyticssdk.CreateTenancyAttachmentDetails,
		resourceanalyticssdk.UpdateTenancyAttachmentDetails,
	]{
		CollectionPath:    "/20241031/tenancyAttachments",
		ItemPath:          "/20241031/tenancyAttachments/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ resourceanalyticssdk.CreateTenancyAttachmentDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ resourceanalyticssdk.TenancyAttachment) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20241031", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close TenancyAttachment OCI mock: %v", err)
		}
	})
	sdkClient := resourceanalyticssdk.TenancyAttachmentClient{BaseClient: session.BaseClient()}
	client := newTenancyAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*resourceanalyticsv1beta1.TenancyAttachment]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *resourceanalyticsv1beta1.TenancyAttachment) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.ResourceAnalyticsInstanceId, current.Spec.ResourceAnalyticsInstanceId) ||
				!reflect.DeepEqual(current.Status.TenancyId, current.Spec.TenancyId) {
				return fmt.Errorf("created TenancyAttachment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *resourceanalyticsv1beta1.TenancyAttachment) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *resourceanalyticsv1beta1.TenancyAttachment) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated TenancyAttachment status = %+v", current.Status)
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
