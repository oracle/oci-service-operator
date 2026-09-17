/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package deployment

import (
	"context"
	"testing"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
)

func TestBuildApiGatewayDeploymentCreateBodyNormalizesCompatibilityRoutes(t *testing.T) {
	t.Parallel()
	resource := apiGatewayDeploymentTestResource()
	resource.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(false)
	details, err := buildApiGatewayDeploymentCreateBody(
		context.Background(),
		&ApiGatewayDeploymentServiceManager{},
		resource,
		"default",
	)
	if err != nil {
		t.Fatal(err)
	}
	if details.Specification == nil || len(details.Specification.Routes) != 1 {
		t.Fatalf("create specification = %+v", details.Specification)
	}
	backend, ok := details.Specification.Routes[0].Backend.(apigatewaysdk.StockResponseBackend)
	if !ok {
		t.Fatalf("route backend = %T, want apigateway.StockResponseBackend", details.Specification.Routes[0].Backend)
	}
	if backend.Status == nil || *backend.Status != 200 || backend.Body == nil || *backend.Body != "created" {
		t.Fatalf("stock response backend = %+v", backend)
	}
	if details.Specification.LoggingPolicies == nil || details.Specification.LoggingPolicies.AccessLog == nil ||
		details.Specification.LoggingPolicies.AccessLog.IsEnabled == nil || *details.Specification.LoggingPolicies.AccessLog.IsEnabled {
		t.Fatalf("create accessLog.isEnabled = %+v, want explicit false", details.Specification.LoggingPolicies)
	}
}

func TestApiGatewayDeploymentRejectsConflictingRouteForms(t *testing.T) {
	t.Parallel()
	resource := apiGatewayDeploymentTestResource()
	resource.Spec.Specification.Routes = []apigatewayv1beta1.ApiGatewayDeploymentSpecificationRoute{{
		Path: "/other",
		Backend: apigatewayv1beta1.ApiGatewayDeploymentSpecificationRouteBackend{
			Type:   "STOCK_RESPONSE_BACKEND",
			Status: 201,
		},
	}}
	_, err := buildApiGatewayDeploymentCreateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, resource, "default")
	if err == nil {
		t.Fatal("conflicting routes unexpectedly accepted")
	}
}

func TestBuildApiGatewayDeploymentUpdateBodyDetectsRouteDrift(t *testing.T) {
	t.Parallel()
	resource := apiGatewayDeploymentTestResource()
	create, err := buildApiGatewayDeploymentCreateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, resource, "default")
	if err != nil {
		t.Fatal(err)
	}
	current := apigatewaysdk.GetDeploymentResponse{Deployment: apigatewaysdk.Deployment{
		DisplayName:   common.String(resource.Spec.DisplayName),
		Specification: create.Specification,
		FreeformTags:  resource.Spec.FreeformTags,
	}}

	_, needed, err := buildApiGatewayDeploymentUpdateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, resource, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if needed {
		t.Fatal("unchanged deployment unexpectedly requires update")
	}

	resource.Spec.Routes[0].Backend.Body = "updated"
	details, needed, err := buildApiGatewayDeploymentUpdateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, resource, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if !needed || details.Specification == nil {
		t.Fatalf("update = %+v, needed = %t", details, needed)
	}
}

func TestBuildApiGatewayDeploymentUpdateBodyPreservesTrueToFalseBoolean(t *testing.T) {
	t.Parallel()

	currentResource := apiGatewayDeploymentTestResource()
	currentResource.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(true)
	currentDetails, err := buildApiGatewayDeploymentCreateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, currentResource, "default")
	if err != nil {
		t.Fatal(err)
	}
	desired := apiGatewayDeploymentTestResource()
	desired.Spec.Specification.LoggingPolicies.AccessLog.IsEnabled = common.Bool(false)
	current := apigatewaysdk.GetDeploymentResponse{Deployment: apigatewaysdk.Deployment{
		DisplayName:   common.String(desired.Spec.DisplayName),
		Specification: currentDetails.Specification,
		FreeformTags:  desired.Spec.FreeformTags,
	}}

	details, needed, err := buildApiGatewayDeploymentUpdateBody(context.Background(), &ApiGatewayDeploymentServiceManager{}, desired, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if !needed {
		t.Fatal("Deployment true-to-false access log drift was not detected")
	}
	if details.Specification == nil || details.Specification.LoggingPolicies == nil || details.Specification.LoggingPolicies.AccessLog == nil ||
		details.Specification.LoggingPolicies.AccessLog.IsEnabled == nil || *details.Specification.LoggingPolicies.AccessLog.IsEnabled {
		t.Fatalf("update accessLog.isEnabled = %+v, want explicit false", details.Specification)
	}
}

func TestApplyApiGatewayDeploymentRuntimeHooksUsesReviewedSDKShape(t *testing.T) {
	t.Parallel()
	hooks := ApiGatewayDeploymentRuntimeHooks{Semantics: newApiGatewayDeploymentRuntimeSemantics()}
	applyApiGatewayDeploymentRuntimeHooks(&ApiGatewayDeploymentServiceManager{}, &hooks)
	if got := hooks.Semantics.List.ResponseItemsField; got != "Items" {
		t.Fatalf("List response items field = %q, want Items", got)
	}
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("deployment compatibility hooks are incomplete")
	}
}

func apiGatewayDeploymentTestResource() *apigatewayv1beta1.ApiGatewayDeployment {
	return &apigatewayv1beta1.ApiGatewayDeployment{Spec: apigatewayv1beta1.ApiGatewayDeploymentSpec{
		GatewayId:     "ocid1.apigateway.oc1..mock",
		CompartmentId: "ocid1.compartment.oc1..mock",
		PathPrefix:    "/mock",
		DisplayName:   "mock-deployment",
		FreeformTags:  map[string]string{"osok-mock": "create"},
		Routes: []apigatewayv1beta1.ApiGatewayDeploymentSpecificationRoute{{
			Path:    "/hello",
			Methods: []string{"GET"},
			Backend: apigatewayv1beta1.ApiGatewayDeploymentSpecificationRouteBackend{
				Type:   "STOCK_RESPONSE_BACKEND",
				Status: 200,
				Body:   "created",
			},
		}},
	}}
}
