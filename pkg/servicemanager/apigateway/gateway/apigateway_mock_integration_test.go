/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationApiGatewayWorkRequestCRUDAndEndpointSecret(t *testing.T) {
	t.Parallel()
	const (
		gatewayID = "ocid1.apigateway.oc1..mock"
		createWR  = "ocid1.apiworkrequest.oc1..gateway-create"
		updateWR  = "ocid1.apiworkrequest.oc1..gateway-update"
		deleteWR  = "ocid1.apiworkrequest.oc1..gateway-delete"
	)

	resource := &apigatewayv1beta1.ApiGateway{}
	ocimock.InitializeResource(resource, "mock-api-gateway")
	resource.Spec = apigatewayv1beta1.ApiGatewaySpec{
		CompartmentId: "ocid1.compartment.oc1..mock",
		EndpointType:  "PRIVATE",
		SubnetId:      "ocid1.subnet.oc1..mock",
		DisplayName:   "mock-api-gateway",
		FreeformTags:  map[string]string{"osok-mock": "create"},
		ResponseCacheDetails: apigatewayv1beta1.ApiGatewayResponseCacheDetails{
			Type: "EXTERNAL_RESP_CACHE",
			Servers: []apigatewayv1beta1.ApiGatewayResponseCacheDetailsServer{{
				Host: "cache.example.com",
				Port: 6379,
			}},
			AuthenticationSecretId:            "ocid1.vaultsecret.oc1..mock",
			AuthenticationSecretVersionNumber: 1,
			IsSslEnabled:                      common.Bool(true),
		},
	}
	createdCache := apiGatewayMockResponseCache(true)
	updatedCache := apiGatewayMockResponseCache(false)
	createDetails := apigatewaysdk.CreateGatewayDetails{
		CompartmentId:        common.String(resource.Spec.CompartmentId),
		EndpointType:         apigatewaysdk.GatewayEndpointTypePrivate,
		SubnetId:             common.String(resource.Spec.SubnetId),
		DisplayName:          common.String(resource.Spec.DisplayName),
		FreeformTags:         resource.Spec.FreeformTags,
		ResponseCacheDetails: createdCache,
	}
	updateDetails := apigatewaysdk.UpdateGatewayDetails{
		DisplayName:          common.String("mock-api-gateway-updated"),
		FreeformTags:         map[string]string{"osok-mock": "update"},
		ResponseCacheDetails: updatedCache,
	}
	created := apigatewaysdk.Gateway{
		Id: common.String(gatewayID), CompartmentId: createDetails.CompartmentId,
		EndpointType: createDetails.EndpointType, SubnetId: createDetails.SubnetId,
		DisplayName: createDetails.DisplayName, FreeformTags: createDetails.FreeformTags,
		ResponseCacheDetails: createdCache,
		Hostname:             common.String("mock.example.com"), LifecycleState: apigatewaysdk.GatewayLifecycleStateActive,
	}
	updated := created
	updated.DisplayName = updateDetails.DisplayName
	updated.FreeformTags = updateDetails.FreeformTags
	updated.ResponseCacheDetails = updatedCache

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[apigatewaysdk.Gateway, apigatewaysdk.CreateGatewayDetails, apigatewaysdk.UpdateGatewayDetails]{
		CollectionPath: "/20190501/gateways",
		ItemPath:       "/20190501/gateways/" + gatewayID,
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:  &createDetails,
		CreatedState:   &created,
		UpdateRequest:  &updateDetails,
		UpdatedState:   &updated,
		CompareCreate: func(got, want apigatewaysdk.CreateGatewayDetails) error {
			if got.CompartmentId == nil || want.CompartmentId == nil || *got.CompartmentId != *want.CompartmentId ||
				got.SubnetId == nil || want.SubnetId == nil || *got.SubnetId != *want.SubnetId ||
				got.DisplayName == nil || want.DisplayName == nil || *got.DisplayName != *want.DisplayName ||
				got.EndpointType != want.EndpointType || got.FreeformTags["osok-mock"] != want.FreeformTags["osok-mock"] {
				return fmt.Errorf("Gateway create details = %+v, want %+v", got, want)
			}
			return validateApiGatewayMockResponseCache(got.ResponseCacheDetails, true)
		},
		CompareUpdate: func(got, want apigatewaysdk.UpdateGatewayDetails) error {
			if got.DisplayName == nil || want.DisplayName == nil || *got.DisplayName != *want.DisplayName ||
				got.FreeformTags["osok-mock"] != want.FreeformTags["osok-mock"] {
				return fmt.Errorf("Gateway update details = %+v, want %+v", got, want)
			}
			return validateApiGatewayMockResponseCache(got.ResponseCacheDetails, false)
		},
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: http.StatusAccepted, UpdateStatus: http.StatusAccepted, DeleteStatus: http.StatusAccepted,
		NotFoundCode:  "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{createWR}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{updateWR}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{deleteWR}},
		ValidateCreate: func(request ocimock.Request, _ apigatewaysdk.CreateGatewayDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			apiGatewayWorkRequestRoute(t, createWR, gatewayID, apigatewaysdk.WorkRequestOperationTypeCreateGateway, apigatewaysdk.WorkRequestResourceActionTypeCreated),
			apiGatewayWorkRequestRoute(t, updateWR, gatewayID, apigatewaysdk.WorkRequestOperationTypeUpdateGateway, apigatewaysdk.WorkRequestResourceActionTypeUpdated),
			apiGatewayWorkRequestRoute(t, deleteWR, gatewayID, apigatewaysdk.WorkRequestOperationTypeDeleteGateway, apigatewaysdk.WorkRequestResourceActionTypeDeleted),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://apigateway.mock.invalid", BasePath: "20190501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	credentials := &apiGatewayTestCredentialClient{}
	manager := &ApiGatewayServiceManager{CredentialClient: credentials, Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newApiGatewayDefaultRuntimeHooks(ApiGatewaySDKClients{
		gatewayClient:      apigatewaysdk.GatewayClient{BaseClient: session.BaseClient()},
		workRequestsClient: apigatewaysdk.WorkRequestsClient{BaseClient: session.BaseClient()},
	})
	applyApiGatewayRuntimeHooks(manager, &hooks)
	client := wrapApiGatewayGeneratedClient(hooks, defaultApiGatewayServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apigatewayv1beta1.ApiGateway](buildApiGatewayGeneratedRuntimeConfig(manager, hooks)),
	})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apigatewayv1beta1.ApiGateway]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreated: func(current *apigatewayv1beta1.ApiGateway) error {
			if current.Status.Id != gatewayID || current.Status.Hostname != "mock.example.com" || credentials.createCall == 0 ||
				current.Status.ResponseCacheDetails.IsSslEnabled == nil || !*current.Status.ResponseCacheDetails.IsSslEnabled {
				return fmt.Errorf("created ApiGateway status = %+v, secret creates = %d", current.Status, credentials.createCall)
			}
			return nil
		},
		Mutate: func(current *apigatewayv1beta1.ApiGateway) {
			current.Spec.DisplayName = "mock-api-gateway-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
			current.Spec.ResponseCacheDetails.IsSslEnabled = common.Bool(false)
		},
		ValidateUpdated: func(current *apigatewayv1beta1.ApiGateway) error {
			if current.Status.DisplayName != "mock-api-gateway-updated" || current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.ResponseCacheDetails.IsSslEnabled == nil || *current.Status.ResponseCacheDetails.IsSslEnabled {
				return fmt.Errorf("updated ApiGateway status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if credentials.deleteCall != 1 {
		t.Fatalf("endpoint Secret deletes = %d, want 1", credentials.deleteCall)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func apiGatewayMockResponseCache(enabled bool) apigatewaysdk.ExternalRespCache {
	return apigatewaysdk.ExternalRespCache{
		Servers: []apigatewaysdk.ResponseCacheRespServer{{
			Host: common.String("cache.example.com"),
			Port: common.Int(6379),
		}},
		AuthenticationSecretId:            common.String("ocid1.vaultsecret.oc1..mock"),
		AuthenticationSecretVersionNumber: common.Int64(1),
		IsSslEnabled:                      common.Bool(enabled),
	}
}

func validateApiGatewayMockResponseCache(value apigatewaysdk.ResponseCacheDetails, enabled bool) error {
	cache, ok := value.(apigatewaysdk.ExternalRespCache)
	if !ok {
		return fmt.Errorf("Gateway responseCacheDetails = %T, want apigateway.ExternalRespCache", value)
	}
	if cache.IsSslEnabled == nil || *cache.IsSslEnabled != enabled {
		return fmt.Errorf("Gateway responseCacheDetails.isSslEnabled = %v, want %t", cache.IsSslEnabled, enabled)
	}
	if cache.IsSslVerifyDisabled != nil {
		return fmt.Errorf("Gateway omitted responseCacheDetails.isSslVerifyDisabled = %v, want nil", cache.IsSslVerifyDisabled)
	}
	return nil
}

func apiGatewayWorkRequestRoute(
	t *testing.T,
	workRequestID string,
	resourceID string,
	operation apigatewaysdk.WorkRequestOperationTypeEnum,
	action apigatewaysdk.WorkRequestResourceActionTypeEnum,
) ocimock.Route {
	t.Helper()
	workRequest := apigatewaysdk.WorkRequest{
		Id: common.String(workRequestID), OperationType: operation,
		Status:        apigatewaysdk.WorkRequestStatusSucceeded,
		CompartmentId: common.String("ocid1.compartment.oc1..mock"), PercentComplete: common.Float32(100),
		Resources: []apigatewaysdk.WorkRequestResource{{
			EntityType: common.String("ApiGateway"), ActionType: action, Identifier: common.String(resourceID),
		}},
	}
	return ocimock.Route{
		Name: workRequestID, Method: http.MethodGet, Path: "/20190501/workRequests/" + workRequestID, MinimumCalls: 2,
		Respond: ocimock.NewWorkRequestResponseSequence(t, string(apigatewaysdk.WorkRequestStatusInProgress), workRequest),
	}
}
