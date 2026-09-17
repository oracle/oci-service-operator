/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package backendset

import (
	"context"
	"fmt"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"net/http"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationBackendSetCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loadbalancerv1beta1.BackendSet{}
	ocimock.InitializeResource(resource, "mock-backendset")
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.BackendSetSpec](t, `{
  "backends": [
    {
      "ipAddress": "10.0.0.3",
      "port": 8080,
      "weight": 1
    }
  ],
  "healthChecker": {
    "intervalInMillis": 10000,
    "port": 8080,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000
  },
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN"
}`)
	resource.Spec.LoadBalancerId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": false,
    "port": 8081,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000
  },
  "policy": "ROUND_ROBIN"
}`)
	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreateBackendSetDetails](t, `{
  "backends": [
    {
      "ipAddress": "10.0.0.3",
      "port": 8080,
      "weight": 1
    }
  ],
  "healthChecker": {
    "intervalInMillis": 10000,
    "port": 8080,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000
  },
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN"
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.BackendSet](t, `{
  "backendMaxConnections": null,
  "backendRetryPolicy": null,
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "maxConnections": null,
      "name": "10.0.0.3:8080",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "connectionEstablishmentTimeoutInSeconds": null,
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": null,
    "port": 8080,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "keepAliveTimeoutInSeconds": null,
  "lbCookieSessionPersistenceConfiguration": null,
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN",
  "sessionPersistenceConfiguration": null,
  "sslConfiguration": null
}`)
	createdReadStates := []loadbalancersdk.BackendSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.BackendSet](t, `{
  "backendMaxConnections": null,
  "backendRetryPolicy": null,
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "maxConnections": null,
      "name": "10.0.0.3:8080",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "connectionEstablishmentTimeoutInSeconds": null,
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": null,
    "port": 8080,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "keepAliveTimeoutInSeconds": null,
  "lbCookieSessionPersistenceConfiguration": null,
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN",
  "sessionPersistenceConfiguration": null,
  "sslConfiguration": null
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdateBackendSetDetails](t, `{
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": false,
    "port": 8081,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000
  },
  "policy": "ROUND_ROBIN"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.BackendSet](t, `{
  "backendMaxConnections": null,
  "backendRetryPolicy": null,
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "maxConnections": null,
      "name": "10.0.0.3:8080",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "connectionEstablishmentTimeoutInSeconds": null,
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": false,
    "port": 8081,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "keepAliveTimeoutInSeconds": null,
  "lbCookieSessionPersistenceConfiguration": null,
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN",
  "sessionPersistenceConfiguration": null,
  "sslConfiguration": null
}`)
	updatedReadStates := []loadbalancersdk.BackendSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.BackendSet](t, `{
  "backendMaxConnections": null,
  "backendRetryPolicy": null,
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "maxConnections": null,
      "name": "10.0.0.3:8080",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "connectionEstablishmentTimeoutInSeconds": null,
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": false,
    "port": 8081,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "keepAliveTimeoutInSeconds": null,
  "lbCookieSessionPersistenceConfiguration": null,
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN",
  "sessionPersistenceConfiguration": null,
  "sslConfiguration": null
}`),
	}
	deletedReadStates := []loadbalancersdk.BackendSet{
		ocimock.MustOCIResponseFixture[loadbalancersdk.BackendSet](t, `{
  "backendMaxConnections": null,
  "backendRetryPolicy": null,
  "backends": [
    {
      "backup": false,
      "drain": false,
      "ipAddress": "10.0.0.3",
      "maxConnections": null,
      "name": "10.0.0.3:8080",
      "offline": false,
      "port": 8080,
      "weight": 1
    }
  ],
  "connectionEstablishmentTimeoutInSeconds": null,
  "healthChecker": {
    "intervalInMillis": 10000,
    "isForcePlainText": false,
    "port": 8081,
    "protocol": "TCP",
    "responseBodyRegex": ".*",
    "retries": 3,
    "returnCode": 200,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "keepAliveTimeoutInSeconds": null,
  "lbCookieSessionPersistenceConfiguration": null,
  "name": "osok_mock_backend_set_v1",
  "policy": "ROUND_ROBIN",
  "sessionPersistenceConfiguration": null,
  "sslConfiguration": null
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loadbalancersdk.BackendSet,
		loadbalancersdk.CreateBackendSetDetails,
		loadbalancersdk.UpdateBackendSetDetails,
	]{
		CollectionPath:    "/20170115/loadBalancers/<ocid:1>/backendSets",
		ItemPath:          "/20170115/loadBalancers/<ocid:1>/backendSets/osok_mock_backend_set_v1",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      204,
		UpdateStatus:      204,
		DeleteStatus:      204,
		DeleteStatuses:    []int{http.StatusNoContent, http.StatusNotFound},
		NotFoundCode:      "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreateBackendSetDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loadbalancersdk.BackendSet) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20170115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close BackendSet OCI mock: %v", err)
		}
	})
	sdkClient := loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()}
	hooks := newBackendSetRuntimeHooksWithOCIClient(sdkClient)
	applyBackendSetRuntimeHooks(&hooks)
	client := defaultBackendSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.BackendSet](buildBackendSetGeneratedRuntimeConfig(&BackendSetServiceManager{}, hooks)),
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.BackendSet]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loadbalancerv1beta1.BackendSet) error {
			if !reflect.DeepEqual(current.Status.Backends, current.Spec.Backends) ||
				!reflect.DeepEqual(current.Status.HealthChecker, current.Spec.HealthChecker) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Policy, current.Spec.Policy) {
				return fmt.Errorf("created BackendSet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.BackendSet) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.BackendSet) error {
			if !reflect.DeepEqual(current.Status.Backends, current.Spec.Backends) ||
				!reflect.DeepEqual(current.Status.HealthChecker, current.Spec.HealthChecker) ||
				!reflect.DeepEqual(current.Status.Policy, current.Spec.Policy) {
				return fmt.Errorf("updated BackendSet status = %+v", current.Status)
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
