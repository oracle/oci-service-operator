/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package subscription

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	onssdk "github.com/oracle/oci-go-sdk/v65/ons"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockSubscriptionID = "ocid1.onssubscription.oc1..mock"

// Contract evidence: recorded subscription CRUD, the resource-local runtime, OCI SDK, and pinned provider.
func TestMockIntegrationSubscriptionLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &onsv1beta1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "mock-subscription", Namespace: "default", UID: types.UID("mock-subscription-uid")}, Spec: onsv1beta1.SubscriptionSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", TopicId: "ocid1.onstopic.oc1..mock", Protocol: "ORACLE_FUNCTIONS",
		Endpoint: "ocid1.fnfunc.oc1..mock", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newSubscriptionMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://notification.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Subscription OCI mock: %v", err)
		}
	})
	client := newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, onssdk.NotificationDataPlaneClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*onsv1beta1.Subscription]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *onsv1beta1.Subscription) error {
			if current.Status.Id != mockSubscriptionID || current.Status.TopicId != current.Spec.TopicId || current.Status.Protocol != current.Spec.Protocol || current.Status.Endpoint != current.Spec.Endpoint || current.Status.LifecycleState != string(onssdk.SubscriptionLifecycleStateActive) {
				return fmt.Errorf("created Subscription status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *onsv1beta1.Subscription) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *onsv1beta1.Subscription) error {
			if current.Status.FreeformTags["osok-mock"] != "update" || current.Status.LifecycleState != string(onssdk.SubscriptionLifecycleStateActive) {
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

func newSubscriptionMockResponder(resource *onsv1beta1.Subscription) (*ocimock.CRUDResponder[onssdk.Subscription], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[onssdk.Subscription]{
		CollectionPath: "/20181201/subscriptions", ItemPath: "/20181201/subscriptions/" + mockSubscriptionID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(request ocimock.Request, present bool, state onssdk.Subscription) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("compartmentId") != resource.Spec.CompartmentId || query.Get("topicId") != resource.Spec.TopicId {
				return ocimock.Response{}, fmt.Errorf("unexpected ListSubscriptions query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []onssdk.Subscription{})
			}
			return ocimock.JSONResponse(http.StatusOK, []onssdk.Subscription{state})
		},
		Create: func(request ocimock.Request) (onssdk.Subscription, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero onssdk.Subscription
				return zero, ocimock.Response{}, err
			}
			var details onssdk.CreateSubscriptionDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return onssdk.Subscription{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return onssdk.Subscription{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.TopicId == nil || *details.TopicId != resource.Spec.TopicId || details.Protocol == nil || *details.Protocol != resource.Spec.Protocol || details.Endpoint == nil || *details.Endpoint != resource.Spec.Endpoint {
				return onssdk.Subscription{}, ocimock.Response{}, fmt.Errorf("unexpected CreateSubscription details: %+v", details)
			}
			state := onssdk.Subscription{Id: common.String(mockSubscriptionID), TopicId: details.TopicId, Protocol: details.Protocol, Endpoint: details.Endpoint,
				LifecycleState: onssdk.SubscriptionLifecycleStateActive, CompartmentId: details.CompartmentId, CreatedTime: common.Int64(1), FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusAccepted, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state onssdk.Subscription) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state onssdk.Subscription) (onssdk.Subscription, ocimock.Response, error) {
			var details onssdk.UpdateSubscriptionDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return onssdk.Subscription{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" || details.DeliveryPolicy != nil {
				return onssdk.Subscription{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateSubscription details: %+v", details)
			}
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, details)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ onssdk.Subscription) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
