/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package authenticationprovider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationAuthenticationProviderCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := makeAuthenticationProviderResource()
	ocimock.InitializeResource(resource, "mock-authentication-provider")
	resource.Annotations[authenticationProviderOdaInstanceIDAnnotation] = "<ocid:1>"
	resource.Spec.TokenEndpointUrl = "https://idp.example.com/token"
	updatedSpec := resource.Spec
	updatedSpec.TokenEndpointUrl = "https://idp.example.com/token-v2"
	updatedSpec.Scopes = "openid profile"
	updatedSpec.FreeformTags = map[string]string{"env": "prod"}
	created := makeSDKAuthenticationProvider("<ocid:2>", resource, odasdk.LifecycleStateActive)
	updatedResource := resource.DeepCopy()
	updatedResource.Spec = updatedSpec
	updated := makeSDKAuthenticationProvider("<ocid:2>", updatedResource, odasdk.LifecycleStateActive)

	responder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[odasdk.AuthenticationProvider]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/authenticationProviders", ItemPath: "/20190506/odaInstances/<ocid:1>/authenticationProviders/<ocid:2>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(request ocimock.Request, present bool, state odasdk.AuthenticationProvider) (ocimock.Response, error) {
			if request.URL.Query().Get("identityProvider") != resource.Spec.IdentityProvider || request.URL.Query().Get("name") != resource.Spec.Name {
				return ocimock.Response{}, fmt.Errorf("ListAuthenticationProviders query = %s", request.URL.RawQuery)
			}
			items := []odasdk.AuthenticationProviderSummary{}
			if present {
				items = append(items, makeSDKAuthenticationProviderSummary("<ocid:2>", resource, state.LifecycleState))
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": items})
		},
		Create: func(request ocimock.Request) (odasdk.AuthenticationProvider, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero odasdk.AuthenticationProvider
				return zero, ocimock.Response{}, err
			}
			var details odasdk.CreateAuthenticationProviderDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return odasdk.AuthenticationProvider{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name || details.TokenEndpointUrl == nil || *details.TokenEndpointUrl != resource.Spec.TokenEndpointUrl || details.ClientSecret == nil || *details.ClientSecret != resource.Spec.ClientSecret {
				return odasdk.AuthenticationProvider{}, ocimock.Response{}, fmt.Errorf("CreateAuthenticationProvider details = %+v", details)
			}
			response, err := ocimock.JSONResponse(http.StatusCreated, created)
			return created, response, err
		},
		Read: func(_ ocimock.Request, state odasdk.AuthenticationProvider) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, _ odasdk.AuthenticationProvider) (odasdk.AuthenticationProvider, ocimock.Response, error) {
			var details odasdk.UpdateAuthenticationProviderDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return odasdk.AuthenticationProvider{}, ocimock.Response{}, err
			}
			if details.TokenEndpointUrl == nil || *details.TokenEndpointUrl != updatedSpec.TokenEndpointUrl || details.Scopes == nil || *details.Scopes != updatedSpec.Scopes || details.ClientSecret != nil || details.FreeformTags["env"] != "prod" {
				return odasdk.AuthenticationProvider{}, ocimock.Response{}, fmt.Errorf("UpdateAuthenticationProvider details = %+v", details)
			}
			response, err := ocimock.JSONResponse(http.StatusOK, updated)
			return updated, response, err
		},
		Delete: func(_ ocimock.Request, _ odasdk.AuthenticationProvider) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oda.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, odasdk.ManagementClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.AuthenticationProvider]{
		Resource: resource, Client: client,
		ValidateCreated: func(current *odav1beta1.AuthenticationProvider) error {
			if current.Status.Id != "<ocid:2>" || current.Status.Name != current.Spec.Name || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created AuthenticationProvider status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.AuthenticationProvider) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.AuthenticationProvider) error {
			if current.Status.TokenEndpointUrl != current.Spec.TokenEndpointUrl || current.Status.Scopes != current.Spec.Scopes || current.Status.FreeformTags["env"] != "prod" {
				return fmt.Errorf("updated AuthenticationProvider status = %+v", current.Status)
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
