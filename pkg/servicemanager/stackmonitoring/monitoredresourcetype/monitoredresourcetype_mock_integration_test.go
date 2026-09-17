/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package monitoredresourcetype

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockMonitoredResourceTypeID = "ocid1.monitoredresourcetype.oc1..mock"

// Contract evidence: recorded monitored-resource-type CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationMonitoredResourceTypeLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &stackmonitoringv1beta1.MonitoredResourceType{Spec: stackmonitoringv1beta1.MonitoredResourceTypeSpec{Name: "osok_mock_custom_type", CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "OSOK mock custom type", Description: "mock create", MetricNamespace: "osok_mock", SourceType: string(stackmonitoringsdk.SourceTypeSmRepoOnly), ResourceCategory: string(stackmonitoringsdk.ResourceCategoryApplication), FreeformTags: map[string]string{"mock": "create"}}}
	ocimock.InitializeResource(resource, "mock-monitored-resource-type")
	responder, err := newMonitoredResourceTypeMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://stack-monitoring.mock.invalid", BasePath: "20210330", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MonitoredResourceType OCI mock: %v", err)
		}
	})
	manager := &MonitoredResourceTypeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMonitoredResourceTypeDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()})
	applyMonitoredResourceTypeRuntimeHooks(&hooks)
	client := wrapMonitoredResourceTypeGeneratedClient(hooks, defaultMonitoredResourceTypeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.MonitoredResourceType](buildMonitoredResourceTypeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.MonitoredResourceType]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.MonitoredResourceType) error {
			if current.Status.Id != mockMonitoredResourceTypeID || current.Status.Name != resource.Spec.Name {
				return fmt.Errorf("created MonitoredResourceType status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.MonitoredResourceType) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"mock": "update"}
		},
		ValidateUpdated: func(current *stackmonitoringv1beta1.MonitoredResourceType) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["mock"] != "update" {
				return fmt.Errorf("updated MonitoredResourceType status = %+v", current.Status)
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

func newMonitoredResourceTypeMockResponder(resource *stackmonitoringv1beta1.MonitoredResourceType) (*ocimock.CRUDResponder[stackmonitoringsdk.MonitoredResourceType], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[stackmonitoringsdk.MonitoredResourceType]{
		CollectionPath: "/20210330/monitoredResourceTypes", ItemPath: "/20210330/monitoredResourceTypes/" + mockMonitoredResourceTypeID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (stackmonitoringsdk.MonitoredResourceType, ocimock.Response, error) {
			if got := request.Header.Get("opc-retry-token"); got != string(resource.UID) {
				return stackmonitoringsdk.MonitoredResourceType{}, ocimock.Response{}, fmt.Errorf("CreateMonitoredResourceType retry token = %q, want resource UID %q", got, resource.UID)
			}
			var details stackmonitoringsdk.CreateMonitoredResourceTypeDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return stackmonitoringsdk.MonitoredResourceType{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return stackmonitoringsdk.MonitoredResourceType{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name || details.SourceType != stackmonitoringsdk.SourceTypeSmRepoOnly {
				return stackmonitoringsdk.MonitoredResourceType{}, ocimock.Response{}, fmt.Errorf("unexpected CreateMonitoredResourceType details: %+v", details)
			}
			state := stackmonitoringsdk.MonitoredResourceType{Id: common.String(mockMonitoredResourceTypeID), Name: details.Name, CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, Description: details.Description, MetricNamespace: details.MetricNamespace,
				TenancyId: common.String("ocid1.tenancy.oc1..mock"), IsSystemDefined: common.Bool(false), LifecycleState: stackmonitoringsdk.ResourceTypeLifecycleStateActive, SourceType: details.SourceType, ResourceCategory: details.ResourceCategory,
				TimeCreated: &now, TimeUpdated: &now, Metadata: details.Metadata, FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state stackmonitoringsdk.MonitoredResourceType) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state stackmonitoringsdk.MonitoredResourceType) (stackmonitoringsdk.MonitoredResourceType, ocimock.Response, error) {
			var details stackmonitoringsdk.UpdateMonitoredResourceTypeDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["mock"] != "update" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected UpdateMonitoredResourceType details: %+v", details)
			}
			state.DisplayName, state.Description, state.MetricNamespace, state.SourceType, state.ResourceCategory, state.Metadata, state.FreeformTags, state.DefinedTags, state.TimeUpdated = details.DisplayName, details.Description, details.MetricNamespace, details.SourceType, details.ResourceCategory, details.Metadata, details.FreeformTags, details.DefinedTags, &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state stackmonitoringsdk.MonitoredResourceType) (stackmonitoringsdk.MonitoredResourceType, ocimock.Response, error) {
			state.LifecycleState = stackmonitoringsdk.ResourceTypeLifecycleStateDeleted
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
