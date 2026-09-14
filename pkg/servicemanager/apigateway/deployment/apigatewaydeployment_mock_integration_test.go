/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package deployment

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

func TestMockIntegrationApiGatewayDeploymentWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	const (
		deploymentID = "ocid1.apideployment.oc1..mock"
		gatewayID    = "ocid1.apigateway.oc1..mock"
		createWR     = "ocid1.apiworkrequest.oc1..deployment-create"
		updateWR     = "ocid1.apiworkrequest.oc1..deployment-update"
		deleteWR     = "ocid1.apiworkrequest.oc1..deployment-delete"
	)
	resource := apiGatewayDeploymentTestResource()
	ocimock.InitializeResource(resource, "mock-api-gateway-deployment")
	resource.Spec.GatewayId = gatewayID
	resource.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(true)
	createDetails, err := buildApiGatewayDeploymentCreateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, resource, "default")
	if err != nil {
		t.Fatal(err)
	}
	created := apigatewaysdk.Deployment{
		Id: common.String(deploymentID), GatewayId: common.String(gatewayID), CompartmentId: createDetails.CompartmentId,
		PathPrefix: createDetails.PathPrefix, Specification: createDetails.Specification,
		DisplayName: createDetails.DisplayName, FreeformTags: createDetails.FreeformTags,
		Endpoint: common.String("https://mock.example.com/mock"), LifecycleState: apigatewaysdk.DeploymentLifecycleStateActive,
	}
	resource.Spec.DisplayName = "mock-deployment-updated"
	resource.Spec.Routes[0].Backend.Body = "updated"
	resource.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
	resource.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(false)
	updateDetails, _, err := buildApiGatewayDeploymentUpdateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, resource, "default", created)
	if err != nil {
		t.Fatal(err)
	}
	resource.Spec.DisplayName = "mock-deployment"
	resource.Spec.Routes[0].Backend.Body = "created"
	resource.Spec.FreeformTags = map[string]string{"osok-mock": "create"}
	resource.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(true)
	updated := created
	updated.DisplayName = updateDetails.DisplayName
	updated.Specification = updateDetails.Specification
	updated.FreeformTags = updateDetails.FreeformTags

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[apigatewaysdk.Deployment, apigatewaysdk.CreateDeploymentDetails, apigatewaysdk.UpdateDeploymentDetails]{
		CollectionPath: "/20190501/deployments", ItemPath: "/20190501/deployments/" + deploymentID,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createDetails, CreatedState: &created, UpdateRequest: &updateDetails, UpdatedState: &updated,
		CompareCreate: func(got, want apigatewaysdk.CreateDeploymentDetails) error {
			if got.GatewayId == nil || want.GatewayId == nil || *got.GatewayId != *want.GatewayId ||
				got.CompartmentId == nil || want.CompartmentId == nil || *got.CompartmentId != *want.CompartmentId ||
				got.PathPrefix == nil || want.PathPrefix == nil || *got.PathPrefix != *want.PathPrefix ||
				got.DisplayName == nil || want.DisplayName == nil || *got.DisplayName != *want.DisplayName ||
				got.Specification == nil || len(got.Specification.Routes) != 1 ||
				got.FreeformTags["osok-mock"] != want.FreeformTags["osok-mock"] {
				return fmt.Errorf("Deployment create details = %+v, want %+v", got, want)
			}
			return validateApiGatewayDeploymentAccessLog(got.Specification, true)
		},
		CompareUpdate: func(got, want apigatewaysdk.UpdateDeploymentDetails) error {
			if got.DisplayName == nil || want.DisplayName == nil || *got.DisplayName != *want.DisplayName ||
				got.Specification == nil || len(got.Specification.Routes) != 1 ||
				got.FreeformTags["osok-mock"] != want.FreeformTags["osok-mock"] {
				return fmt.Errorf("Deployment update details = %+v, want %+v", got, want)
			}
			return validateApiGatewayDeploymentAccessLog(got.Specification, false)
		},
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: http.StatusAccepted, UpdateStatus: http.StatusAccepted, DeleteStatus: http.StatusAccepted,
		NotFoundCode:  "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{createWR}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{updateWR}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{deleteWR}},
		ValidateCreate: func(request ocimock.Request, _ apigatewaysdk.CreateDeploymentDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			apiGatewayDeploymentWorkRequestRoute(t, createWR, deploymentID, apigatewaysdk.WorkRequestOperationTypeCreateDeployment, apigatewaysdk.WorkRequestResourceActionTypeCreated),
			apiGatewayDeploymentWorkRequestRoute(t, updateWR, deploymentID, apigatewaysdk.WorkRequestOperationTypeUpdateDeployment, apigatewaysdk.WorkRequestResourceActionTypeUpdated),
			apiGatewayDeploymentWorkRequestRoute(t, deleteWR, deploymentID, apigatewaysdk.WorkRequestOperationTypeDeleteDeployment, apigatewaysdk.WorkRequestResourceActionTypeDeleted),
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
	manager := &ApiGatewayDeploymentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newApiGatewayDeploymentDefaultRuntimeHooks(ApiGatewayDeploymentSDKClients{
		deploymentClient:   apigatewaysdk.DeploymentClient{BaseClient: session.BaseClient()},
		workRequestsClient: apigatewaysdk.WorkRequestsClient{BaseClient: session.BaseClient()},
	})
	applyApiGatewayDeploymentRuntimeHooks(manager, &hooks)
	client := wrapApiGatewayDeploymentGeneratedClient(hooks, defaultApiGatewayDeploymentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apigatewayv1beta1.ApiGatewayDeployment](buildApiGatewayDeploymentGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apigatewayv1beta1.ApiGatewayDeployment]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreated: func(current *apigatewayv1beta1.ApiGatewayDeployment) error {
			if current.Status.Id != deploymentID || current.Status.DisplayName != "mock-deployment" || len(current.Status.Specification.Routes) != 1 ||
				current.Status.Specification.LoggingPolicies.AccessLog.IsEnabled == nil || !*current.Status.Specification.LoggingPolicies.AccessLog.IsEnabled {
				return fmt.Errorf("created ApiGatewayDeployment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apigatewayv1beta1.ApiGatewayDeployment) {
			current.Spec.DisplayName = "mock-deployment-updated"
			current.Spec.Routes[0].Backend.Body = "updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
			current.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(false)
		},
		ValidateUpdated: func(current *apigatewayv1beta1.ApiGatewayDeployment) error {
			if current.Status.DisplayName != "mock-deployment-updated" || current.Status.Specification.Routes[0].Backend.Body != "updated" ||
				current.Status.Specification.LoggingPolicies.AccessLog.IsEnabled == nil || *current.Status.Specification.LoggingPolicies.AccessLog.IsEnabled {
				return fmt.Errorf("updated ApiGatewayDeployment status = %+v", current.Status)
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

func validateApiGatewayDeploymentAccessLog(specification *apigatewaysdk.ApiSpecification, enabled bool) error {
	if specification == nil || specification.LoggingPolicies == nil || specification.LoggingPolicies.AccessLog == nil ||
		specification.LoggingPolicies.AccessLog.IsEnabled == nil || *specification.LoggingPolicies.AccessLog.IsEnabled != enabled {
		return fmt.Errorf("Deployment accessLog.isEnabled = %+v, want %t", specification, enabled)
	}
	return nil
}

func apiGatewayDeploymentWorkRequestRoute(
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
			EntityType: common.String("ApiGatewayDeployment"), ActionType: action, Identifier: common.String(resourceID),
		}},
	}
	return ocimock.Route{
		Name: workRequestID, Method: http.MethodGet, Path: "/20190501/workRequests/" + workRequestID, MinimumCalls: 2,
		Respond: ocimock.NewWorkRequestResponseSequence(t, string(apigatewaysdk.WorkRequestStatusInProgress), workRequest),
	}
}
