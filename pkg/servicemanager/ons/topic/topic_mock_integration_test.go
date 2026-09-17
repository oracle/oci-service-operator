/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package topic

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	onssdk "github.com/oracle/oci-go-sdk/v65/ons"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockTopicID = "ocid1.onstopic.oc1..mock"

// Contract evidence: recorded topic CRUD, the resource-local runtime, OCI SDK, and pinned provider.
func TestMockIntegrationTopicLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &onsv1beta1.Topic{ObjectMeta: metav1.ObjectMeta{Name: "mock-topic", Namespace: "default", UID: types.UID("mock-topic-uid")}, Spec: onsv1beta1.TopicSpec{
		Name: "mock-topic", CompartmentId: "ocid1.compartment.oc1..mock", Description: "mock create", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newTopicMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://notification.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Topic OCI mock: %v", err)
		}
	})
	client := newTopicServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, onssdk.NotificationControlPlaneClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*onsv1beta1.Topic]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *onsv1beta1.Topic) error {
			if current.Status.OsokStatus.Ocid != shared.OCID(mockTopicID) || current.Status.OsokStatus.Reason != string(shared.Active) ||
				current.Status.TopicId != mockTopicID || current.Status.Name != "mock-topic" ||
				current.Status.Description != "mock create" || current.Status.LifecycleState != string(onssdk.NotificationTopicLifecycleStateActive) {
				return fmt.Errorf("created Topic status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *onsv1beta1.Topic) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *onsv1beta1.Topic) error {
			if current.Status.OsokStatus.Reason != string(shared.Active) || current.Status.Description != "mock update" ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Topic status = %+v", current.Status)
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

func newTopicMockResponder(resource *onsv1beta1.Topic) (*ocimock.CRUDResponder[onssdk.NotificationTopic], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[onssdk.NotificationTopic]{
		CollectionPath: "/20181201/topics", ItemPath: "/20181201/topics/" + mockTopicID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (onssdk.NotificationTopic, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero onssdk.NotificationTopic
				return zero, ocimock.Response{}, err
			}
			var details onssdk.CreateTopicDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return onssdk.NotificationTopic{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return onssdk.NotificationTopic{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name || details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId {
				return onssdk.NotificationTopic{}, ocimock.Response{}, fmt.Errorf("unexpected CreateTopic details: %+v", details)
			}
			state := onssdk.NotificationTopic{Name: details.Name, TopicId: common.String(mockTopicID), CompartmentId: details.CompartmentId,
				LifecycleState: onssdk.NotificationTopicLifecycleStateActive, TimeCreated: &now, ApiEndpoint: common.String("https://notification.mock.invalid"),
				Description: details.Description, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state onssdk.NotificationTopic) (onssdk.NotificationTopic, ocimock.Response, error) {
			if state.LifecycleState == onssdk.NotificationTopicLifecycleStateDeleting && deleteRead {
				response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
				return state, response, err
			}
			if state.LifecycleState == onssdk.NotificationTopicLifecycleStateDeleting {
				deleteRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state onssdk.NotificationTopic) (onssdk.NotificationTopic, ocimock.Response, error) {
			var details onssdk.TopicAttributesDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return onssdk.NotificationTopic{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return onssdk.NotificationTopic{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return onssdk.NotificationTopic{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateTopic details: %+v", details)
			}
			state.Description, state.FreeformTags = details.Description, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state onssdk.NotificationTopic) (onssdk.NotificationTopic, ocimock.Response, error) {
			state.LifecycleState = onssdk.NotificationTopicLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
