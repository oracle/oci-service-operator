/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package domain

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	tenantmanagercontrolplanesdk "github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane"
	tenantmanagercontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/tenantmanagercontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationDomainWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &tenantmanagercontrolplanev1beta1.Domain{}
	ocimock.InitializeResource(resource, "mock-domain")
	resource.Spec = ocimock.MustJSONFixture[tenantmanagercontrolplanev1beta1.DomainSpec](t, `{
  "compartmentId": "<ocid:1>",
  "domainName": "example.com",
  "subscriptionEmail": "admin@example.com"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[tenantmanagercontrolplanesdk.CreateDomainDetails](t, `{
  "compartmentId": "<ocid:1>",
  "domainName": "example.com",
  "subscriptionEmail": "admin@example.com"
}`)
	updateRequest := ocimock.MustJSONFixture[tenantmanagercontrolplanesdk.UpdateDomainDetails](t, `{
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[tenantmanagercontrolplanesdk.Domain](t, `{
  "definedTags": null,
  "domainName": "example.com",
  "freeformTags": null,
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "ownerId": "<ocid:4>",
  "status": "ACTIVE",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "txtRecord": "txt-value"
}`)
	updatedState := ocimock.MustOCIResponseFixture[tenantmanagercontrolplanesdk.Domain](t, `{
  "definedTags": null,
  "domainName": "example.com",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "ownerId": "<ocid:4>",
  "status": "ACTIVE",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "txtRecord": "txt-value"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[tenantmanagercontrolplanesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "REGISTER_DOMAIN",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "Domain",
      "entityUri": null,
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[tenantmanagercontrolplanesdk.Domain, tenantmanagercontrolplanesdk.CreateDomainDetails, tenantmanagercontrolplanesdk.UpdateDomainDetails]{
		CollectionPath: "/20230401/domains", ItemPath: "/20230401/domains/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: nil, DeleteHeaders: nil,
		ValidateCreate: func(request ocimock.Request, _ tenantmanagercontrolplanesdk.CreateDomainDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20230401/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://organizations.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230401", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	base := session.BaseClient()
	client := newDomainServiceClientWithClients(log,
		tenantmanagercontrolplanesdk.DomainClient{BaseClient: base},
		tenantmanagercontrolplanesdk.WorkRequestClient{BaseClient: base},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*tenantmanagercontrolplanev1beta1.Domain]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *tenantmanagercontrolplanev1beta1.Domain) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DomainName != resource.Spec.DomainName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Domain status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *tenantmanagercontrolplanev1beta1.Domain) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *tenantmanagercontrolplanev1beta1.Domain) error {
			if current.Status.FreeformTags["osok-mock"] != "update" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Domain status = %+v", current.Status)
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
