/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package privateapplication

import (
	"context"
	"fmt"
	servicecatalogsdk "github.com/oracle/oci-go-sdk/v65/servicecatalog"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"net/http"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationPrivateApplicationLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makePrivateApplicationResource()
	ocimock.InitializeResource(resource, "mock-privateapplication")
	resource.Spec = ocimock.MustJSONFixture[servicecatalogv1beta1.PrivateApplicationSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "private-app-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "longDescription": "long description",
  "packageDetails": {
    "packageType": "STACK",
    "version": "1.0.0",
    "zipFileBase64Encoded": "zip-payload"
  },
  "shortDescription": "short description"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "private-app-alpha-updated"
}`)
	createRequest := ocimock.MustJSONFixture[servicecatalogsdk.CreatePrivateApplicationDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "private-app-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "longDescription": "long description",
  "packageDetails": {
    "packageType": "STACK",
    "version": "1.0.0",
    "zipFileBase64Encoded": "zip-payload"
  },
  "shortDescription": "short description"
}`)
	createdState := ocimock.MustOCIResponseFixture[servicecatalogsdk.PrivateApplication](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "private-app-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "longDescription": "long description",
  "packageDetails": {
    "packageType": "STACK",
    "version": "1.0.0",
    "zipFileBase64Encoded": "zip-payload"
  },
  "shortDescription": "short description"
}`)
	createdReadStates := []servicecatalogsdk.PrivateApplication{
		ocimock.MustOCIResponseFixture[servicecatalogsdk.PrivateApplication](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "private-app-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "longDescription": "long description",
  "packageDetails": {
    "packageType": "STACK",
    "version": "1.0.0",
    "zipFileBase64Encoded": "zip-payload"
  },
  "shortDescription": "short description"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[servicecatalogsdk.UpdatePrivateApplicationDetails](t, `{
  "displayName": "private-app-alpha-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[servicecatalogsdk.PrivateApplication](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "private-app-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "longDescription": "long description",
  "packageDetails": {
    "packageType": "STACK",
    "version": "1.0.0",
    "zipFileBase64Encoded": "zip-payload"
  },
  "shortDescription": "short description"
}`)
	updatedReadStates := []servicecatalogsdk.PrivateApplication{
		ocimock.MustOCIResponseFixture[servicecatalogsdk.PrivateApplication](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "private-app-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "longDescription": "long description",
  "packageDetails": {
    "packageType": "STACK",
    "version": "1.0.0",
    "zipFileBase64Encoded": "zip-payload"
  },
  "shortDescription": "short description"
}`),
	}
	packageState := ocimock.MustOCIResponseFixture[servicecatalogsdk.PrivateApplicationPackageCollection](t, `{
	  "items":[{"id":"<ocid:3>","packageType":"STACK","privateApplicationId":"<ocid:2>","version":"1.0.0"}]
	}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		servicecatalogsdk.PrivateApplication,
		servicecatalogsdk.CreatePrivateApplicationDetails,
		servicecatalogsdk.UpdatePrivateApplicationDetails,
	]{
		CollectionPath:     "/20210527/privateApplications",
		ItemPath:           "/20210527/privateApplications/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ servicecatalogsdk.CreatePrivateApplicationDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ servicecatalogsdk.PrivateApplication) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name:         "list application packages",
				Method:       http.MethodGet,
				Path:         "/20210527/privateApplicationPackages",
				MinimumCalls: 1,
				Respond: func(request ocimock.Request) (ocimock.Response, error) {
					if request.URL.Query().Get("packageType") != "STACK" ||
						request.URL.Query().Get("privateApplicationId") != "<ocid:2>" {
						return ocimock.Response{}, fmt.Errorf("package list query = %s", request.URL.RawQuery)
					}
					return ocimock.JSONResponse(http.StatusOK, packageState)
				},
			},
			{
				Name:         "download package configuration",
				Method:       http.MethodGet,
				Path:         "/20210527/privateApplicationPackages/<ocid:3>/actions/downloadConfig",
				MinimumCalls: 1,
				Respond: func(request ocimock.Request) (ocimock.Response, error) {
					if len(request.Body) != 0 {
						return ocimock.Response{}, fmt.Errorf("package download body = %s", request.Body)
					}
					return ocimock.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
						Body:       []byte("zip-payload"),
					}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210527", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close PrivateApplication OCI mock: %v", err)
		}
	})
	sdkClient := servicecatalogsdk.ServiceCatalogClient{BaseClient: session.BaseClient()}
	client := newPrivateApplicationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*servicecatalogv1beta1.PrivateApplication]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *servicecatalogv1beta1.PrivateApplication) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LongDescription, current.Spec.LongDescription) ||
				!reflect.DeepEqual(current.Status.ShortDescription, current.Spec.ShortDescription) {
				return fmt.Errorf("created PrivateApplication status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *servicecatalogv1beta1.PrivateApplication) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *servicecatalogv1beta1.PrivateApplication) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated PrivateApplication status = %+v", current.Status)
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
