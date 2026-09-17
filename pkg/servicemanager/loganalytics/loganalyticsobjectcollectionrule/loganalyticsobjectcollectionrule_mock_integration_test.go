/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package loganalyticsobjectcollectionrule

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationLogAnalyticsObjectCollectionRuleCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := makeLogAnalyticsObjectCollectionRuleResource()
	ocimock.InitializeResource(resource, "mock-object-collection-rule")
	updatedSpec := resource.Spec
	updatedSpec.Description = "updated collection rule"
	updatedSpec.IsEnabled = false
	createdState := makeSDKLogAnalyticsObjectCollectionRule("<ocid:2>", resource.Spec, loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive)
	updatedState := makeSDKLogAnalyticsObjectCollectionRule("<ocid:2>", updatedSpec, loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive)
	deletedState := makeSDKLogAnalyticsObjectCollectionRule("<ocid:2>", updatedSpec, loganalyticssdk.ObjectCollectionRuleLifecycleStatesDeleted)
	responder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[loganalyticssdk.LogAnalyticsObjectCollectionRule]{
		CollectionPath: "/20200601/namespaces/" + resource.Spec.OsNamespace + "/logAnalyticsObjectCollectionRules", ItemPath: "/20200601/namespaces/" + resource.Spec.OsNamespace + "/logAnalyticsObjectCollectionRules/<ocid:2>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		List: func(_ ocimock.Request, present bool, state loganalyticssdk.LogAnalyticsObjectCollectionRule) (ocimock.Response, error) {
			items := []loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary{}
			if present {
				items = append(items, makeSDKLogAnalyticsObjectCollectionRuleSummary("<ocid:2>", updatedSpec, state.LifecycleState))
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": items})
		},
		Create: func(request ocimock.Request) (loganalyticssdk.LogAnalyticsObjectCollectionRule, ocimock.Response, error) {
			var details loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name || details.LogGroupId == nil || *details.LogGroupId != resource.Spec.LogGroupId {
				return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, ocimock.Response{}, fmt.Errorf("create object collection rule details = %+v", details)
			}
			response, err := ocimock.JSONResponse(http.StatusCreated, createdState)
			return createdState, response, err
		},
		Read: func(_ ocimock.Request, state loganalyticssdk.LogAnalyticsObjectCollectionRule) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, _ loganalyticssdk.LogAnalyticsObjectCollectionRule) (loganalyticssdk.LogAnalyticsObjectCollectionRule, ocimock.Response, error) {
			var details loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != updatedSpec.Description || details.IsEnabled == nil || *details.IsEnabled {
				return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, ocimock.Response{}, fmt.Errorf("update object collection rule details = %+v", details)
			}
			response, err := ocimock.JSONResponse(http.StatusOK, updatedState)
			return updatedState, response, err
		},
		DeleteTransition: func(_ ocimock.Request, _ loganalyticssdk.LogAnalyticsObjectCollectionRule) (loganalyticssdk.LogAnalyticsObjectCollectionRule, ocimock.Response, error) {
			return deletedState, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://loganalytics.mock.invalid", BasePath: "20200601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newLogAnalyticsObjectCollectionRuleServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule) error {
			if current.Status.Id != "<ocid:2>" || current.Status.Name != current.Spec.Name || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("created object collection rule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule) error {
			if current.Status.Description != current.Spec.Description || current.Status.IsEnabled != current.Spec.IsEnabled {
				return fmt.Errorf("updated object collection rule status = %+v", current.Status)
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
