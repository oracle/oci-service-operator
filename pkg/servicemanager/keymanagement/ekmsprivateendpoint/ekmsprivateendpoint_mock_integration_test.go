/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package ekmsprivateendpoint

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	keymanagementsdk "github.com/oracle/oci-go-sdk/v65/keymanagement"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockEkmsPrivateEndpointID = "ocid1.ekmsprivateendpoint.oc1..mock"

// Contract evidence: vendored SDK request/response types and the handwritten EKMS runtime contract.
func TestMockIntegrationEkmsPrivateEndpointLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := makeSpecEkmsPrivateEndpoint()
	ocimock.InitializeResource(resource, "mock-ekmsprivateendpoint")
	responder, err := newEkmsPrivateEndpointMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://kms.mock.invalid", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &EkmsPrivateEndpointServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	sdkClient := keymanagementsdk.EkmClient{BaseClient: session.BaseClient()}
	hooks := newEkmsPrivateEndpointDefaultRuntimeHooks(sdkClient)
	applyEkmsPrivateEndpointRuntimeHooks(manager, &hooks)
	delegate := defaultEkmsPrivateEndpointServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*keymanagementv1beta1.EkmsPrivateEndpoint](buildEkmsPrivateEndpointGeneratedRuntimeConfig(manager, hooks)),
	}
	client := newEkmsPrivateEndpointRuntimeClient(manager, delegate)
	client.sdk = sdkClient
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*keymanagementv1beta1.EkmsPrivateEndpoint]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *keymanagementv1beta1.EkmsPrivateEndpoint) error {
			if current.Status.Id != mockEkmsPrivateEndpointID || current.Status.LifecycleState != string(keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive) {
				return fmt.Errorf("created EKMS private endpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *keymanagementv1beta1.EkmsPrivateEndpoint) {
			current.Spec.DisplayName = "mock-ekms-private-endpoint-updated"
			current.Spec.FreeformTags = map[string]string{"phase": "updated"}
		},
		ValidateUpdated: func(current *keymanagementv1beta1.EkmsPrivateEndpoint) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["phase"] != "updated" {
				return fmt.Errorf("updated EKMS private endpoint status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func newEkmsPrivateEndpointMockResponder(resource *keymanagementv1beta1.EkmsPrivateEndpoint) (*ocimock.CRUDResponder[keymanagementsdk.EkmsPrivateEndpoint], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[keymanagementsdk.EkmsPrivateEndpoint]{
		CollectionPath: "/20180608/ekmsPrivateEndpoints",
		ItemPath:       "/20180608/ekmsPrivateEndpoints/" + mockEkmsPrivateEndpointID,
		ExpectedOperations: []ocimock.Operation{
			ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete,
		},
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		Create: func(request ocimock.Request) (keymanagementsdk.EkmsPrivateEndpoint, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero keymanagementsdk.EkmsPrivateEndpoint
				return zero, ocimock.Response{}, err
			}
			var details keymanagementsdk.CreateEkmsPrivateEndpointDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return keymanagementsdk.EkmsPrivateEndpoint{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return keymanagementsdk.EkmsPrivateEndpoint{}, ocimock.Response{}, err
			}
			state := makeSDKEkmsPrivateEndpoint(mockEkmsPrivateEndpointID, keymanagementsdk.EkmsPrivateEndpointLifecycleStateCreating, *details.DisplayName, details.FreeformTags)
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state keymanagementsdk.EkmsPrivateEndpoint) (keymanagementsdk.EkmsPrivateEndpoint, ocimock.Response, error) {
			if state.LifecycleState == keymanagementsdk.EkmsPrivateEndpointLifecycleStateCreating {
				state.LifecycleState = keymanagementsdk.EkmsPrivateEndpointLifecycleStateActive
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state keymanagementsdk.EkmsPrivateEndpoint) (keymanagementsdk.EkmsPrivateEndpoint, ocimock.Response, error) {
			var details keymanagementsdk.UpdateEkmsPrivateEndpointDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-ekms-private-endpoint-updated" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected EKMS update details: %+v", details)
			}
			state.DisplayName, state.FreeformTags = details.DisplayName, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ keymanagementsdk.EkmsPrivateEndpoint) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(_ ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotFound", "message": "resource deleted"})
		},
	})
}
