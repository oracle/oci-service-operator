/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package processset

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

const mockProcessSetID = "ocid1.processset.oc1..mock"

// Contract evidence: recorded process-set CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationProcessSetLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &stackmonitoringv1beta1.ProcessSet{Spec: stackmonitoringv1beta1.ProcessSetSpec{CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-process-set", Specification: stackmonitoringv1beta1.ProcessSetSpecification{Items: []stackmonitoringv1beta1.ProcessSetSpecificationItem{{Label: "java", ProcessCommand: "java", ProcessUser: "opc", ProcessLineRegexPattern: ".*java.*"}}}, FreeformTags: map[string]string{"mock": "create"}}}
	ocimock.InitializeResource(resource, "mock-process-set")
	responder, err := newProcessSetMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://stack-monitoring.mock.invalid", BasePath: "20210330", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ProcessSet OCI mock: %v", err)
		}
	})
	client := newProcessSetServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.ProcessSet]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.ProcessSet) error {
			if current.Status.Id != mockProcessSetID || current.Status.DisplayName != resource.Spec.DisplayName {
				return fmt.Errorf("created ProcessSet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.ProcessSet) {
			current.Spec.Specification.Items[0].ProcessLineRegexPattern = ".*java.*-jar.*"
			current.Spec.FreeformTags = map[string]string{"mock": "update"}
		},
		ValidateUpdated: func(current *stackmonitoringv1beta1.ProcessSet) error {
			if current.Status.FreeformTags["mock"] != "update" || current.Status.Specification.Items[0].ProcessLineRegexPattern != ".*java.*-jar.*" {
				return fmt.Errorf("updated ProcessSet status = %+v", current.Status)
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

func newProcessSetMockResponder(resource *stackmonitoringv1beta1.ProcessSet) (*ocimock.CRUDResponder[stackmonitoringsdk.ProcessSet], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[stackmonitoringsdk.ProcessSet]{
		CollectionPath: "/20210330/processSets", ItemPath: "/20210330/processSets/" + mockProcessSetID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state stackmonitoringsdk.ProcessSet) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []stackmonitoringsdk.ProcessSet{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []stackmonitoringsdk.ProcessSet{state}})
		},
		Create: func(request ocimock.Request) (stackmonitoringsdk.ProcessSet, ocimock.Response, error) {
			if got := request.Header.Get("opc-retry-token"); got != string(resource.UID) {
				return stackmonitoringsdk.ProcessSet{}, ocimock.Response{}, fmt.Errorf("CreateProcessSet retry token = %q, want resource UID %q", got, resource.UID)
			}
			var details stackmonitoringsdk.CreateProcessSetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return stackmonitoringsdk.ProcessSet{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return stackmonitoringsdk.ProcessSet{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.Specification == nil || len(details.Specification.Items) != 1 {
				return stackmonitoringsdk.ProcessSet{}, ocimock.Response{}, fmt.Errorf("unexpected CreateProcessSet details: %+v", details)
			}
			state := stackmonitoringsdk.ProcessSet{Id: common.String(mockProcessSetID), CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, Specification: details.Specification, LifecycleState: stackmonitoringsdk.LifecycleStateActive,
				TimeCreated: &now, TimeUpdated: &now, Revision: common.String("1"), FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state stackmonitoringsdk.ProcessSet) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state stackmonitoringsdk.ProcessSet) (stackmonitoringsdk.ProcessSet, ocimock.Response, error) {
			var details stackmonitoringsdk.UpdateProcessSetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.Specification == nil || len(details.Specification.Items) != 1 || details.Specification.Items[0].ProcessLineRegexPattern == nil || *details.Specification.Items[0].ProcessLineRegexPattern != ".*java.*-jar.*" || details.FreeformTags["mock"] != "update" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected UpdateProcessSet details: %+v", details)
			}
			state.DisplayName, state.Specification, state.FreeformTags, state.DefinedTags, state.Revision, state.TimeUpdated = details.DisplayName, details.Specification, details.FreeformTags, details.DefinedTags, common.String("2"), &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ stackmonitoringsdk.ProcessSet) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
