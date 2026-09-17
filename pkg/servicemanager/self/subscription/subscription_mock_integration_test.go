/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	selfsdk "github.com/oracle/oci-go-sdk/v65/self"
	selfv1beta1 "github.com/oracle/oci-service-operator/api/self/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationSubscriptionWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &selfv1beta1.Subscription{}
	ocimock.InitializeResource(resource, "mock-subscription")
	resource.Spec = ocimock.MustJSONFixture[selfv1beta1.SubscriptionSpec](t, `{
  "additionalDetails": [
    {
      "key": "contract",
      "value": "gold"
    }
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "self-subscription",
  "freeformTags": {
    "env": "dev"
  },
  "productId": "<ocid:2>",
  "realm": "OC1",
  "region": "us-chicago-1",
  "sellerId": "<ocid:3>",
  "sourceType": "OCI_NATIVE",
  "subscriptionDetails": {
    "billingDetails": {
      "meters": [
        {
          "name": "meter-a",
          "rateAllocation": 100
        }
      ],
      "metricType": "OCPU_HOURS",
      "rateAllocation": 100,
      "sku": "gold-sku"
    },
    "partnerRegistrationUrl": "https://partner.example.com/register",
    "pricingPlan": {
      "billingFrequency": "YEARLY",
      "planName": "gold-plan",
      "planType": "FIXED",
      "rates": [
        {
          "currency": "USD",
          "rate": 42.5
        }
      ]
    }
  },
  "tenantId": "<ocid:4>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "self-subscription-updated"
}`)

	createRequest := ocimock.MustJSONFixture[selfsdk.CreateSubscriptionDetails](t, `{
  "additionalDetails": [
    {
      "key": "contract",
      "value": "gold"
    }
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "self-subscription",
  "freeformTags": {
    "env": "dev"
  },
  "productId": "<ocid:2>",
  "realm": "OC1",
  "region": "us-chicago-1",
  "sellerId": "<ocid:3>",
  "sourceType": "OCI_NATIVE",
  "subscriptionDetails": {
    "billingDetails": {
      "meters": [
        {
          "name": "meter-a",
          "rateAllocation": 100
        }
      ],
      "metricType": "OCPU_HOURS",
      "rateAllocation": 100,
      "sku": "gold-sku"
    },
    "partnerRegistrationUrl": "https://partner.example.com/register",
    "pricingPlan": {
      "billingFrequency": "YEARLY",
      "planName": "gold-plan",
      "planType": "FIXED",
      "rates": [
        {
          "currency": "USD",
          "rate": 42.5
        }
      ]
    }
  },
  "tenantId": "<ocid:4>"
}`)
	updateRequest := ocimock.MustJSONFixture[selfsdk.UpdateSubscriptionDetails](t, `{
  "displayName": "self-subscription-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[selfsdk.Subscription](t, `{
  "additionalDetails": [
    {
      "key": "contract",
      "value": "gold"
    }
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "self-subscription",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:6>",
  "lifecycleDetails": "ACTIVE",
  "lifecycleState": "ACTIVE",
  "productId": "<ocid:2>",
  "realm": "OC1",
  "region": "us-chicago-1",
  "sellerId": "<ocid:3>",
  "sourceType": "OCI_NATIVE",
  "subscriptionDetails": {
    "amount": null,
    "billingDetails": {
      "hasGovSku": null,
      "meters": [
        {
          "extendedMetadata": null,
          "name": "meter-a",
          "rateAllocation": 100
        }
      ],
      "metricType": "OCPU_HOURS",
      "rateAllocation": 100,
      "sku": "gold-sku"
    },
    "currency": null,
    "isAutoRenew": null,
    "partnerRegistrationUrl": "https://partner.example.com/register",
    "pricingPlan": {
      "billingFrequency": "YEARLY",
      "planDescription": null,
      "planName": "gold-plan",
      "planType": "FIXED",
      "rates": [
        {
          "currency": "USD",
          "rate": 42.5
        }
      ]
    }
  },
  "systemTags": null,
  "tenantId": "<ocid:4>",
  "timeCreated": null,
  "timeEnded": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	updatedState := ocimock.MustOCIResponseFixture[selfsdk.Subscription](t, `{
  "additionalDetails": [
    {
      "key": "contract",
      "value": "gold"
    }
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "self-subscription-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:6>",
  "lifecycleDetails": "ACTIVE",
  "lifecycleState": "ACTIVE",
  "productId": "<ocid:2>",
  "realm": "OC1",
  "region": "us-chicago-1",
  "sellerId": "<ocid:3>",
  "sourceType": "OCI_NATIVE",
  "subscriptionDetails": {
    "amount": null,
    "billingDetails": {
      "hasGovSku": null,
      "meters": [
        {
          "extendedMetadata": null,
          "name": "meter-a",
          "rateAllocation": 100
        }
      ],
      "metricType": "OCPU_HOURS",
      "rateAllocation": 100,
      "sku": "gold-sku"
    },
    "currency": null,
    "isAutoRenew": null,
    "partnerRegistrationUrl": "https://partner.example.com/register",
    "pricingPlan": {
      "billingFrequency": "YEARLY",
      "planDescription": null,
      "planName": "gold-plan",
      "planType": "FIXED",
      "rates": [
        {
          "currency": "USD",
          "rate": 42.5
        }
      ]
    }
  },
  "systemTags": null,
  "tenantId": "<ocid:4>",
  "timeCreated": null,
  "timeEnded": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[selfsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "CREATE_SUBSCRIPTION",
  "percentComplete": 42,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "subscription",
      "entityUri": "/subscriptions/<ocid:6>",
      "identifier": "<ocid:6>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[selfsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "wr-update",
  "operationType": "UPDATE_SUBSCRIPTION",
  "percentComplete": 42,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "subscription",
      "entityUri": "/subscriptions/<ocid:6>",
      "identifier": "<ocid:6>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[selfsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:7>",
  "operationType": "DELETE_SUBSCRIPTION",
  "percentComplete": 42,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "subscription",
      "entityUri": "/subscriptions/<ocid:6>",
      "identifier": "<ocid:6>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[selfsdk.Subscription, selfsdk.CreateSubscriptionDetails, selfsdk.UpdateSubscriptionDetails]{
		CollectionPath:     "/20260129/subscriptions",
		ItemPath:           "/20260129/subscriptions/<ocid:6>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		ValidateCreate: func(request ocimock.Request, _ selfsdk.CreateSubscriptionDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20260129/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20260129/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20260129/workRequests/<ocid:7>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://self.us-ashburn-1.oci.oraclecloud.com", BasePath: "20260129", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := selfsdk.SubscriptionClient{BaseClient: session.BaseClient()}
	client := newSubscriptionServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*selfv1beta1.Subscription]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *selfv1beta1.Subscription) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Subscription status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *selfv1beta1.Subscription) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *selfv1beta1.Subscription) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Subscription status = %+v", current.Status)
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
