/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"testing"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	smanager "github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCredentialClient struct {
	createSecretFn func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	deleteSecretFn func(context.Context, string, string) (bool, error)
	getSecretFn    func(context.Context, string, string) (map[string][]byte, error)
	updateSecretFn func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
}

func (f *fakeCredentialClient) CreateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name string, namespace string) (bool, error) {
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name string, namespace string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, namespace)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

type mockGatewayClient struct {
	createGatewayFn            func(context.Context, apigatewaysdk.CreateGatewayRequest) (apigatewaysdk.CreateGatewayResponse, error)
	getGatewayFn               func(context.Context, apigatewaysdk.GetGatewayRequest) (apigatewaysdk.GetGatewayResponse, error)
	listGatewaysFn             func(context.Context, apigatewaysdk.ListGatewaysRequest) (apigatewaysdk.ListGatewaysResponse, error)
	changeGatewayCompartmentFn func(context.Context, apigatewaysdk.ChangeGatewayCompartmentRequest) (apigatewaysdk.ChangeGatewayCompartmentResponse, error)
	updateGatewayFn            func(context.Context, apigatewaysdk.UpdateGatewayRequest) (apigatewaysdk.UpdateGatewayResponse, error)
	deleteGatewayFn            func(context.Context, apigatewaysdk.DeleteGatewayRequest) (apigatewaysdk.DeleteGatewayResponse, error)
}

func (m *mockGatewayClient) CreateGateway(ctx context.Context, req apigatewaysdk.CreateGatewayRequest) (apigatewaysdk.CreateGatewayResponse, error) {
	if m.createGatewayFn != nil {
		return m.createGatewayFn(ctx, req)
	}
	return apigatewaysdk.CreateGatewayResponse{}, nil
}

func (m *mockGatewayClient) GetGateway(ctx context.Context, req apigatewaysdk.GetGatewayRequest) (apigatewaysdk.GetGatewayResponse, error) {
	if m.getGatewayFn != nil {
		return m.getGatewayFn(ctx, req)
	}
	return apigatewaysdk.GetGatewayResponse{}, nil
}

func (m *mockGatewayClient) ListGateways(ctx context.Context, req apigatewaysdk.ListGatewaysRequest) (apigatewaysdk.ListGatewaysResponse, error) {
	if m.listGatewaysFn != nil {
		return m.listGatewaysFn(ctx, req)
	}
	return apigatewaysdk.ListGatewaysResponse{}, nil
}

func (m *mockGatewayClient) ChangeGatewayCompartment(ctx context.Context, req apigatewaysdk.ChangeGatewayCompartmentRequest) (apigatewaysdk.ChangeGatewayCompartmentResponse, error) {
	if m.changeGatewayCompartmentFn != nil {
		return m.changeGatewayCompartmentFn(ctx, req)
	}
	return apigatewaysdk.ChangeGatewayCompartmentResponse{}, nil
}

func (m *mockGatewayClient) UpdateGateway(ctx context.Context, req apigatewaysdk.UpdateGatewayRequest) (apigatewaysdk.UpdateGatewayResponse, error) {
	if m.updateGatewayFn != nil {
		return m.updateGatewayFn(ctx, req)
	}
	return apigatewaysdk.UpdateGatewayResponse{}, nil
}

func (m *mockGatewayClient) DeleteGateway(ctx context.Context, req apigatewaysdk.DeleteGatewayRequest) (apigatewaysdk.DeleteGatewayResponse, error) {
	if m.deleteGatewayFn != nil {
		return m.deleteGatewayFn(ctx, req)
	}
	return apigatewaysdk.DeleteGatewayResponse{}, nil
}

type mockDeploymentClient struct {
	createDeploymentFn            func(context.Context, apigatewaysdk.CreateDeploymentRequest) (apigatewaysdk.CreateDeploymentResponse, error)
	getDeploymentFn               func(context.Context, apigatewaysdk.GetDeploymentRequest) (apigatewaysdk.GetDeploymentResponse, error)
	listDeploymentsFn             func(context.Context, apigatewaysdk.ListDeploymentsRequest) (apigatewaysdk.ListDeploymentsResponse, error)
	changeDeploymentCompartmentFn func(context.Context, apigatewaysdk.ChangeDeploymentCompartmentRequest) (apigatewaysdk.ChangeDeploymentCompartmentResponse, error)
	updateDeploymentFn            func(context.Context, apigatewaysdk.UpdateDeploymentRequest) (apigatewaysdk.UpdateDeploymentResponse, error)
	deleteDeploymentFn            func(context.Context, apigatewaysdk.DeleteDeploymentRequest) (apigatewaysdk.DeleteDeploymentResponse, error)
}

func (m *mockDeploymentClient) CreateDeployment(ctx context.Context, req apigatewaysdk.CreateDeploymentRequest) (apigatewaysdk.CreateDeploymentResponse, error) {
	if m.createDeploymentFn != nil {
		return m.createDeploymentFn(ctx, req)
	}
	return apigatewaysdk.CreateDeploymentResponse{}, nil
}

func (m *mockDeploymentClient) GetDeployment(ctx context.Context, req apigatewaysdk.GetDeploymentRequest) (apigatewaysdk.GetDeploymentResponse, error) {
	if m.getDeploymentFn != nil {
		return m.getDeploymentFn(ctx, req)
	}
	return apigatewaysdk.GetDeploymentResponse{}, nil
}

func (m *mockDeploymentClient) ListDeployments(ctx context.Context, req apigatewaysdk.ListDeploymentsRequest) (apigatewaysdk.ListDeploymentsResponse, error) {
	if m.listDeploymentsFn != nil {
		return m.listDeploymentsFn(ctx, req)
	}
	return apigatewaysdk.ListDeploymentsResponse{}, nil
}

func (m *mockDeploymentClient) ChangeDeploymentCompartment(ctx context.Context, req apigatewaysdk.ChangeDeploymentCompartmentRequest) (apigatewaysdk.ChangeDeploymentCompartmentResponse, error) {
	if m.changeDeploymentCompartmentFn != nil {
		return m.changeDeploymentCompartmentFn(ctx, req)
	}
	return apigatewaysdk.ChangeDeploymentCompartmentResponse{}, nil
}

func (m *mockDeploymentClient) UpdateDeployment(ctx context.Context, req apigatewaysdk.UpdateDeploymentRequest) (apigatewaysdk.UpdateDeploymentResponse, error) {
	if m.updateDeploymentFn != nil {
		return m.updateDeploymentFn(ctx, req)
	}
	return apigatewaysdk.UpdateDeploymentResponse{}, nil
}

func (m *mockDeploymentClient) DeleteDeployment(ctx context.Context, req apigatewaysdk.DeleteDeploymentRequest) (apigatewaysdk.DeleteDeploymentResponse, error) {
	if m.deleteDeploymentFn != nil {
		return m.deleteDeploymentFn(ctx, req)
	}
	return apigatewaysdk.DeleteDeploymentResponse{}, nil
}

func makeLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
}

func makeGatewayManager(client GatewayClientInterface, credClient *fakeCredentialClient) *GatewayServiceManager {
	manager := NewGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), credClient, runtime.NewScheme(), makeLogger())
	manager.ociClient = client
	return manager
}

func makeDeploymentManager(client DeploymentClientInterface) *DeploymentServiceManager {
	manager := NewDeploymentServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, runtime.NewScheme(), makeLogger())
	manager.ociClient = client
	return manager
}

func TestGatewayCreateOrUpdateCreateSuccess(t *testing.T) {
	const gatewayID = "ocid1.apigateway.oc1..example"
	secretCreated := false

	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, name, namespace string, labels map[string]string, data map[string][]byte) (bool, error) {
			secretCreated = true
			assert.Equal(t, "test-gw", name)
			assert.Equal(t, "default", namespace)
			assert.Equal(t, smanager.ManagedSecretLabelValue, labels[smanager.ManagedSecretLabelKey])
			assert.Equal(t, "test-gw.example.com", string(data["hostname"]))
			assert.True(t, smanager.SecretOwnedBy(data, "ApiGateway", "test-gw"))
			return true, nil
		},
	}

	manager := makeGatewayManager(&mockGatewayClient{
		listGatewaysFn: func(_ context.Context, _ apigatewaysdk.ListGatewaysRequest) (apigatewaysdk.ListGatewaysResponse, error) {
			return apigatewaysdk.ListGatewaysResponse{}, nil
		},
		createGatewayFn: func(_ context.Context, _ apigatewaysdk.CreateGatewayRequest) (apigatewaysdk.CreateGatewayResponse, error) {
			return apigatewaysdk.CreateGatewayResponse{Gateway: apigatewaysdk.Gateway{Id: common.String(gatewayID)}}, nil
		},
		getGatewayFn: func(_ context.Context, _ apigatewaysdk.GetGatewayRequest) (apigatewaysdk.GetGatewayResponse, error) {
			return apigatewaysdk.GetGatewayResponse{
				Gateway: apigatewaysdk.Gateway{
					Id:             common.String(gatewayID),
					DisplayName:    common.String("test-gw"),
					Hostname:       common.String("test-gw.example.com"),
					LifecycleState: apigatewaysdk.GatewayLifecycleStateActive,
				},
			}, nil
		},
	}, credClient)

	resource := &apigatewayv1beta1.ApiGateway{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw", Namespace: "default"},
		Spec: apigatewayv1beta1.ApiGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "test-gw",
			EndpointType:  "PUBLIC",
			SubnetId:      "ocid1.subnet.oc1..example",
		},
	}

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.Equal(t, shared.OCID(gatewayID), resource.Status.OsokStatus.Ocid)
	assert.True(t, secretCreated)
}

func TestGatewayCreateOrUpdateBindByID(t *testing.T) {
	const gatewayID = "ocid1.apigateway.oc1..bound"

	manager := makeGatewayManager(&mockGatewayClient{
		getGatewayFn: func(_ context.Context, _ apigatewaysdk.GetGatewayRequest) (apigatewaysdk.GetGatewayResponse, error) {
			return apigatewaysdk.GetGatewayResponse{
				Gateway: apigatewaysdk.Gateway{
					Id:             common.String(gatewayID),
					DisplayName:    common.String("bound-gw"),
					Hostname:       common.String("bound-gw.example.com"),
					LifecycleState: apigatewaysdk.GatewayLifecycleStateActive,
				},
			}, nil
		},
		updateGatewayFn: func(_ context.Context, _ apigatewaysdk.UpdateGatewayRequest) (apigatewaysdk.UpdateGatewayResponse, error) {
			return apigatewaysdk.UpdateGatewayResponse{}, nil
		},
	}, &fakeCredentialClient{})

	resource := &apigatewayv1beta1.ApiGateway{
		ObjectMeta: metav1.ObjectMeta{Name: "bound-gw", Namespace: "default"},
		Spec: apigatewayv1beta1.ApiGatewaySpec{
			ApiGatewayId: "ocid1.apigateway.oc1..bound",
			DisplayName:  "bound-gw",
			EndpointType: "PUBLIC",
			SubnetId:     "ocid1.subnet.oc1..example",
		},
	}

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.Equal(t, shared.OCID(gatewayID), resource.Status.OsokStatus.Ocid)
}

func TestGatewayDeleteUsesSpecID(t *testing.T) {
	const gatewayID = "ocid1.apigateway.oc1..delete"
	deletedID := ""

	manager := makeGatewayManager(&mockGatewayClient{
		deleteGatewayFn: func(_ context.Context, req apigatewaysdk.DeleteGatewayRequest) (apigatewaysdk.DeleteGatewayResponse, error) {
			deletedID = *req.GatewayId
			return apigatewaysdk.DeleteGatewayResponse{}, nil
		},
		getGatewayFn: func(_ context.Context, req apigatewaysdk.GetGatewayRequest) (apigatewaysdk.GetGatewayResponse, error) {
			return apigatewaysdk.GetGatewayResponse{
				Gateway: apigatewaysdk.Gateway{
					Id:             req.GatewayId,
					LifecycleState: apigatewaysdk.GatewayLifecycleStateDeleted,
				},
			}, nil
		},
	}, &fakeCredentialClient{})

	resource := &apigatewayv1beta1.ApiGateway{
		Spec: apigatewayv1beta1.ApiGatewaySpec{
			ApiGatewayId: gatewayID,
		},
	}

	done, err := manager.Delete(context.Background(), resource)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, gatewayID, deletedID)
}

func TestDeploymentCreateOrUpdateCreateSuccess(t *testing.T) {
	const deploymentID = "ocid1.apigatewaydeployment.oc1..example"

	manager := makeDeploymentManager(&mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigatewaysdk.ListDeploymentsRequest) (apigatewaysdk.ListDeploymentsResponse, error) {
			return apigatewaysdk.ListDeploymentsResponse{}, nil
		},
		createDeploymentFn: func(_ context.Context, _ apigatewaysdk.CreateDeploymentRequest) (apigatewaysdk.CreateDeploymentResponse, error) {
			return apigatewaysdk.CreateDeploymentResponse{Deployment: apigatewaysdk.Deployment{Id: common.String(deploymentID)}}, nil
		},
		getDeploymentFn: func(_ context.Context, _ apigatewaysdk.GetDeploymentRequest) (apigatewaysdk.GetDeploymentResponse, error) {
			return apigatewaysdk.GetDeploymentResponse{
				Deployment: apigatewaysdk.Deployment{
					Id:             common.String(deploymentID),
					DisplayName:    common.String("test-deployment"),
					LifecycleState: apigatewaysdk.DeploymentLifecycleStateActive,
				},
			}, nil
		},
	})

	resource := &apigatewayv1beta1.ApiGatewayDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
		Spec: apigatewayv1beta1.ApiGatewayDeploymentSpec{
			GatewayId:     "ocid1.apigateway.oc1..example",
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "test-deployment",
			PathPrefix:    "/hello",
		},
	}

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.Equal(t, shared.OCID(deploymentID), resource.Status.OsokStatus.Ocid)
}

func TestDeploymentDeleteUsesSpecID(t *testing.T) {
	const deploymentID = "ocid1.apigatewaydeployment.oc1..delete"
	deletedID := ""

	manager := makeDeploymentManager(&mockDeploymentClient{
		deleteDeploymentFn: func(_ context.Context, req apigatewaysdk.DeleteDeploymentRequest) (apigatewaysdk.DeleteDeploymentResponse, error) {
			deletedID = *req.DeploymentId
			return apigatewaysdk.DeleteDeploymentResponse{}, nil
		},
		getDeploymentFn: func(_ context.Context, req apigatewaysdk.GetDeploymentRequest) (apigatewaysdk.GetDeploymentResponse, error) {
			return apigatewaysdk.GetDeploymentResponse{
				Deployment: apigatewaysdk.Deployment{
					Id:             req.DeploymentId,
					LifecycleState: apigatewaysdk.DeploymentLifecycleStateDeleted,
				},
			}, nil
		},
	})

	resource := &apigatewayv1beta1.ApiGatewayDeployment{
		Spec: apigatewayv1beta1.ApiGatewayDeploymentSpec{
			DeploymentId: deploymentID,
		},
	}

	done, err := manager.Delete(context.Background(), resource)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, deploymentID, deletedID)
}

func TestGetCrdStatusWrongType(t *testing.T) {
	gatewayManager := makeGatewayManager(&mockGatewayClient{}, &fakeCredentialClient{})
	_, err := gatewayManager.GetCrdStatus(&apigatewayv1beta1.ApiGatewayDeployment{})
	assert.Error(t, err)

	deploymentManager := makeDeploymentManager(&mockDeploymentClient{})
	_, err = deploymentManager.GetCrdStatus(&apigatewayv1beta1.ApiGateway{})
	assert.Error(t, err)
}
