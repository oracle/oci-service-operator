// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/servicemanager/servicemesh/services/servicemesh_client.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	servicemesh "github.com/oracle/oci-go-sdk/v65/servicemesh"
	v1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
)

// MockServiceMeshClient is a mock of ServiceMeshClient interface.
type MockServiceMeshClient struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMeshClientMockRecorder
}

// MockServiceMeshClientMockRecorder is the mock recorder for MockServiceMeshClient.
type MockServiceMeshClientMockRecorder struct {
	mock *MockServiceMeshClient
}

// NewMockServiceMeshClient creates a new mock instance.
func NewMockServiceMeshClient(ctrl *gomock.Controller) *MockServiceMeshClient {
	mock := &MockServiceMeshClient{ctrl: ctrl}
	mock.recorder = &MockServiceMeshClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceMeshClient) EXPECT() *MockServiceMeshClientMockRecorder {
	return m.recorder
}

// ChangeAccessPolicyCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeAccessPolicyCompartment(ctx context.Context, accessPolicyId, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeAccessPolicyCompartment", ctx, accessPolicyId, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeAccessPolicyCompartment indicates an expected call of ChangeAccessPolicyCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeAccessPolicyCompartment(ctx, accessPolicyId, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeAccessPolicyCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeAccessPolicyCompartment), ctx, accessPolicyId, compartmentId)
}

// ChangeIngressGatewayCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeIngressGatewayCompartment(ctx context.Context, ingressGatewayId, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeIngressGatewayCompartment", ctx, ingressGatewayId, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeIngressGatewayCompartment indicates an expected call of ChangeIngressGatewayCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeIngressGatewayCompartment(ctx, ingressGatewayId, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeIngressGatewayCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeIngressGatewayCompartment), ctx, ingressGatewayId, compartmentId)
}

// ChangeIngressGatewayRouteTableCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeIngressGatewayRouteTableCompartment(ctx context.Context, igrtId, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeIngressGatewayRouteTableCompartment", ctx, igrtId, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeIngressGatewayRouteTableCompartment indicates an expected call of ChangeIngressGatewayRouteTableCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeIngressGatewayRouteTableCompartment(ctx, igrtId, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeIngressGatewayRouteTableCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeIngressGatewayRouteTableCompartment), ctx, igrtId, compartmentId)
}

// ChangeMeshCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeMeshCompartment(ctx context.Context, meshId, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeMeshCompartment", ctx, meshId, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeMeshCompartment indicates an expected call of ChangeMeshCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeMeshCompartment(ctx, meshId, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeMeshCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeMeshCompartment), ctx, meshId, compartmentId)
}

// ChangeVirtualDeploymentCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeVirtualDeploymentCompartment(ctx context.Context, vd, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeVirtualDeploymentCompartment", ctx, vd, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeVirtualDeploymentCompartment indicates an expected call of ChangeVirtualDeploymentCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeVirtualDeploymentCompartment(ctx, vd, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeVirtualDeploymentCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeVirtualDeploymentCompartment), ctx, vd, compartmentId)
}

// ChangeVirtualServiceCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeVirtualServiceCompartment(ctx context.Context, virtualServiceId, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeVirtualServiceCompartment", ctx, virtualServiceId, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeVirtualServiceCompartment indicates an expected call of ChangeVirtualServiceCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeVirtualServiceCompartment(ctx, virtualServiceId, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeVirtualServiceCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeVirtualServiceCompartment), ctx, virtualServiceId, compartmentId)
}

// ChangeVirtualServiceRouteTableCompartment mocks base method.
func (m *MockServiceMeshClient) ChangeVirtualServiceRouteTableCompartment(ctx context.Context, vsrtId, compartmentId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeVirtualServiceRouteTableCompartment", ctx, vsrtId, compartmentId)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeVirtualServiceRouteTableCompartment indicates an expected call of ChangeVirtualServiceRouteTableCompartment.
func (mr *MockServiceMeshClientMockRecorder) ChangeVirtualServiceRouteTableCompartment(ctx, vsrtId, compartmentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeVirtualServiceRouteTableCompartment", reflect.TypeOf((*MockServiceMeshClient)(nil).ChangeVirtualServiceRouteTableCompartment), ctx, vsrtId, compartmentId)
}

// CreateAccessPolicy mocks base method.
func (m *MockServiceMeshClient) CreateAccessPolicy(ctx context.Context, accessPolicy *servicemesh.AccessPolicy, opcRetryToken *string) (*servicemesh.AccessPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccessPolicy", ctx, accessPolicy, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.AccessPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAccessPolicy indicates an expected call of CreateAccessPolicy.
func (mr *MockServiceMeshClientMockRecorder) CreateAccessPolicy(ctx, accessPolicy, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccessPolicy", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateAccessPolicy), ctx, accessPolicy, opcRetryToken)
}

// CreateIngressGateway mocks base method.
func (m *MockServiceMeshClient) CreateIngressGateway(ctx context.Context, ingressGateway *servicemesh.IngressGateway, opcRetryToken *string) (*servicemesh.IngressGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateIngressGateway", ctx, ingressGateway, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.IngressGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateIngressGateway indicates an expected call of CreateIngressGateway.
func (mr *MockServiceMeshClientMockRecorder) CreateIngressGateway(ctx, ingressGateway, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIngressGateway", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateIngressGateway), ctx, ingressGateway, opcRetryToken)
}

// CreateIngressGatewayRouteTable mocks base method.
func (m *MockServiceMeshClient) CreateIngressGatewayRouteTable(ctx context.Context, igrt *servicemesh.IngressGatewayRouteTable, opcRetryToken *string) (*servicemesh.IngressGatewayRouteTable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateIngressGatewayRouteTable", ctx, igrt, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.IngressGatewayRouteTable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateIngressGatewayRouteTable indicates an expected call of CreateIngressGatewayRouteTable.
func (mr *MockServiceMeshClientMockRecorder) CreateIngressGatewayRouteTable(ctx, igrt, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIngressGatewayRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateIngressGatewayRouteTable), ctx, igrt, opcRetryToken)
}

// CreateMesh mocks base method.
func (m *MockServiceMeshClient) CreateMesh(ctx context.Context, mesh *servicemesh.Mesh, opcRetryToken *string) (*servicemesh.Mesh, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMesh", ctx, mesh, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.Mesh)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateMesh indicates an expected call of CreateMesh.
func (mr *MockServiceMeshClientMockRecorder) CreateMesh(ctx, mesh, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMesh", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateMesh), ctx, mesh, opcRetryToken)
}

// CreateVirtualDeployment mocks base method.
func (m *MockServiceMeshClient) CreateVirtualDeployment(ctx context.Context, vd *servicemesh.VirtualDeployment, opcRetryToken *string) (*servicemesh.VirtualDeployment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVirtualDeployment", ctx, vd, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.VirtualDeployment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVirtualDeployment indicates an expected call of CreateVirtualDeployment.
func (mr *MockServiceMeshClientMockRecorder) CreateVirtualDeployment(ctx, vd, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVirtualDeployment", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateVirtualDeployment), ctx, vd, opcRetryToken)
}

// CreateVirtualService mocks base method.
func (m *MockServiceMeshClient) CreateVirtualService(ctx context.Context, virtualService *servicemesh.VirtualService, opcRetryToken *string) (*servicemesh.VirtualService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVirtualService", ctx, virtualService, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.VirtualService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVirtualService indicates an expected call of CreateVirtualService.
func (mr *MockServiceMeshClientMockRecorder) CreateVirtualService(ctx, virtualService, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVirtualService", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateVirtualService), ctx, virtualService, opcRetryToken)
}

// CreateVirtualServiceRouteTable mocks base method.
func (m *MockServiceMeshClient) CreateVirtualServiceRouteTable(ctx context.Context, vsrt *servicemesh.VirtualServiceRouteTable, opcRetryToken *string) (*servicemesh.VirtualServiceRouteTable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVirtualServiceRouteTable", ctx, vsrt, opcRetryToken)
	ret0, _ := ret[0].(*servicemesh.VirtualServiceRouteTable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVirtualServiceRouteTable indicates an expected call of CreateVirtualServiceRouteTable.
func (mr *MockServiceMeshClientMockRecorder) CreateVirtualServiceRouteTable(ctx, vsrt, opcRetryToken interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVirtualServiceRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).CreateVirtualServiceRouteTable), ctx, vsrt, opcRetryToken)
}

// DeleteAccessPolicy mocks base method.
func (m *MockServiceMeshClient) DeleteAccessPolicy(ctx context.Context, accessPolicyId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAccessPolicy", ctx, accessPolicyId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAccessPolicy indicates an expected call of DeleteAccessPolicy.
func (mr *MockServiceMeshClientMockRecorder) DeleteAccessPolicy(ctx, accessPolicyId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAccessPolicy", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteAccessPolicy), ctx, accessPolicyId)
}

// DeleteIngressGateway mocks base method.
func (m *MockServiceMeshClient) DeleteIngressGateway(ctx context.Context, ingressGatewayId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteIngressGateway", ctx, ingressGatewayId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteIngressGateway indicates an expected call of DeleteIngressGateway.
func (mr *MockServiceMeshClientMockRecorder) DeleteIngressGateway(ctx, ingressGatewayId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteIngressGateway", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteIngressGateway), ctx, ingressGatewayId)
}

// DeleteIngressGatewayRouteTable mocks base method.
func (m *MockServiceMeshClient) DeleteIngressGatewayRouteTable(ctx context.Context, igrtId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteIngressGatewayRouteTable", ctx, igrtId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteIngressGatewayRouteTable indicates an expected call of DeleteIngressGatewayRouteTable.
func (mr *MockServiceMeshClientMockRecorder) DeleteIngressGatewayRouteTable(ctx, igrtId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteIngressGatewayRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteIngressGatewayRouteTable), ctx, igrtId)
}

// DeleteMesh mocks base method.
func (m *MockServiceMeshClient) DeleteMesh(ctx context.Context, meshId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteMesh", ctx, meshId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteMesh indicates an expected call of DeleteMesh.
func (mr *MockServiceMeshClientMockRecorder) DeleteMesh(ctx, meshId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMesh", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteMesh), ctx, meshId)
}

// DeleteVirtualDeployment mocks base method.
func (m *MockServiceMeshClient) DeleteVirtualDeployment(ctx context.Context, vd *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVirtualDeployment", ctx, vd)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVirtualDeployment indicates an expected call of DeleteVirtualDeployment.
func (mr *MockServiceMeshClientMockRecorder) DeleteVirtualDeployment(ctx, vd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVirtualDeployment", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteVirtualDeployment), ctx, vd)
}

// DeleteVirtualService mocks base method.
func (m *MockServiceMeshClient) DeleteVirtualService(ctx context.Context, virtualServiceId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVirtualService", ctx, virtualServiceId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVirtualService indicates an expected call of DeleteVirtualService.
func (mr *MockServiceMeshClientMockRecorder) DeleteVirtualService(ctx, virtualServiceId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVirtualService", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteVirtualService), ctx, virtualServiceId)
}

// DeleteVirtualServiceRouteTable mocks base method.
func (m *MockServiceMeshClient) DeleteVirtualServiceRouteTable(ctx context.Context, vsrtId *v1beta1.OCID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVirtualServiceRouteTable", ctx, vsrtId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVirtualServiceRouteTable indicates an expected call of DeleteVirtualServiceRouteTable.
func (mr *MockServiceMeshClientMockRecorder) DeleteVirtualServiceRouteTable(ctx, vsrtId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVirtualServiceRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).DeleteVirtualServiceRouteTable), ctx, vsrtId)
}

// GetAccessPolicy mocks base method.
func (m *MockServiceMeshClient) GetAccessPolicy(ctx context.Context, accessPolicyId *v1beta1.OCID) (*servicemesh.AccessPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccessPolicy", ctx, accessPolicyId)
	ret0, _ := ret[0].(*servicemesh.AccessPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccessPolicy indicates an expected call of GetAccessPolicy.
func (mr *MockServiceMeshClientMockRecorder) GetAccessPolicy(ctx, accessPolicyId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccessPolicy", reflect.TypeOf((*MockServiceMeshClient)(nil).GetAccessPolicy), ctx, accessPolicyId)
}

// GetIngressGateway mocks base method.
func (m *MockServiceMeshClient) GetIngressGateway(ctx context.Context, ingressGatewayId *v1beta1.OCID) (*servicemesh.IngressGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressGateway", ctx, ingressGatewayId)
	ret0, _ := ret[0].(*servicemesh.IngressGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIngressGateway indicates an expected call of GetIngressGateway.
func (mr *MockServiceMeshClientMockRecorder) GetIngressGateway(ctx, ingressGatewayId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressGateway", reflect.TypeOf((*MockServiceMeshClient)(nil).GetIngressGateway), ctx, ingressGatewayId)
}

// GetIngressGatewayRouteTable mocks base method.
func (m *MockServiceMeshClient) GetIngressGatewayRouteTable(ctx context.Context, igrtId *v1beta1.OCID) (*servicemesh.IngressGatewayRouteTable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressGatewayRouteTable", ctx, igrtId)
	ret0, _ := ret[0].(*servicemesh.IngressGatewayRouteTable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIngressGatewayRouteTable indicates an expected call of GetIngressGatewayRouteTable.
func (mr *MockServiceMeshClientMockRecorder) GetIngressGatewayRouteTable(ctx, igrtId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressGatewayRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).GetIngressGatewayRouteTable), ctx, igrtId)
}

// GetMesh mocks base method.
func (m *MockServiceMeshClient) GetMesh(ctx context.Context, meshId *v1beta1.OCID) (*servicemesh.Mesh, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMesh", ctx, meshId)
	ret0, _ := ret[0].(*servicemesh.Mesh)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMesh indicates an expected call of GetMesh.
func (mr *MockServiceMeshClientMockRecorder) GetMesh(ctx, meshId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMesh", reflect.TypeOf((*MockServiceMeshClient)(nil).GetMesh), ctx, meshId)
}

// GetProxyDetails mocks base method.
func (m *MockServiceMeshClient) GetProxyDetails(ctx context.Context) (*string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProxyDetails", ctx)
	ret0, _ := ret[0].(*string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProxyDetails indicates an expected call of GetProxyDetails.
func (mr *MockServiceMeshClientMockRecorder) GetProxyDetails(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProxyDetails", reflect.TypeOf((*MockServiceMeshClient)(nil).GetProxyDetails), ctx)
}

// GetVirtualDeployment mocks base method.
func (m *MockServiceMeshClient) GetVirtualDeployment(ctx context.Context, virtualDeploymentId *v1beta1.OCID) (*servicemesh.VirtualDeployment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVirtualDeployment", ctx, virtualDeploymentId)
	ret0, _ := ret[0].(*servicemesh.VirtualDeployment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVirtualDeployment indicates an expected call of GetVirtualDeployment.
func (mr *MockServiceMeshClientMockRecorder) GetVirtualDeployment(ctx, virtualDeploymentId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVirtualDeployment", reflect.TypeOf((*MockServiceMeshClient)(nil).GetVirtualDeployment), ctx, virtualDeploymentId)
}

// GetVirtualService mocks base method.
func (m *MockServiceMeshClient) GetVirtualService(ctx context.Context, virtualServiceId *v1beta1.OCID) (*servicemesh.VirtualService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVirtualService", ctx, virtualServiceId)
	ret0, _ := ret[0].(*servicemesh.VirtualService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVirtualService indicates an expected call of GetVirtualService.
func (mr *MockServiceMeshClientMockRecorder) GetVirtualService(ctx, virtualServiceId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVirtualService", reflect.TypeOf((*MockServiceMeshClient)(nil).GetVirtualService), ctx, virtualServiceId)
}

// GetVirtualServiceRouteTable mocks base method.
func (m *MockServiceMeshClient) GetVirtualServiceRouteTable(ctx context.Context, vsrtId *v1beta1.OCID) (*servicemesh.VirtualServiceRouteTable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVirtualServiceRouteTable", ctx, vsrtId)
	ret0, _ := ret[0].(*servicemesh.VirtualServiceRouteTable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVirtualServiceRouteTable indicates an expected call of GetVirtualServiceRouteTable.
func (mr *MockServiceMeshClientMockRecorder) GetVirtualServiceRouteTable(ctx, vsrtId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVirtualServiceRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).GetVirtualServiceRouteTable), ctx, vsrtId)
}

// SetClientHost mocks base method.
func (m *MockServiceMeshClient) SetClientHost(cpEndpoint string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetClientHost", cpEndpoint)
}

// SetClientHost indicates an expected call of SetClientHost.
func (mr *MockServiceMeshClientMockRecorder) SetClientHost(cpEndpoint interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetClientHost", reflect.TypeOf((*MockServiceMeshClient)(nil).SetClientHost), cpEndpoint)
}

// UpdateAccessPolicy mocks base method.
func (m *MockServiceMeshClient) UpdateAccessPolicy(ctx context.Context, accessPolicy *servicemesh.AccessPolicy) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAccessPolicy", ctx, accessPolicy)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateAccessPolicy indicates an expected call of UpdateAccessPolicy.
func (mr *MockServiceMeshClientMockRecorder) UpdateAccessPolicy(ctx, accessPolicy interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAccessPolicy", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateAccessPolicy), ctx, accessPolicy)
}

// UpdateIngressGateway mocks base method.
func (m *MockServiceMeshClient) UpdateIngressGateway(ctx context.Context, ingressGateway *servicemesh.IngressGateway) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateIngressGateway", ctx, ingressGateway)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateIngressGateway indicates an expected call of UpdateIngressGateway.
func (mr *MockServiceMeshClientMockRecorder) UpdateIngressGateway(ctx, ingressGateway interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateIngressGateway", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateIngressGateway), ctx, ingressGateway)
}

// UpdateIngressGatewayRouteTable mocks base method.
func (m *MockServiceMeshClient) UpdateIngressGatewayRouteTable(ctx context.Context, igrt *servicemesh.IngressGatewayRouteTable) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateIngressGatewayRouteTable", ctx, igrt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateIngressGatewayRouteTable indicates an expected call of UpdateIngressGatewayRouteTable.
func (mr *MockServiceMeshClientMockRecorder) UpdateIngressGatewayRouteTable(ctx, igrt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateIngressGatewayRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateIngressGatewayRouteTable), ctx, igrt)
}

// UpdateMesh mocks base method.
func (m *MockServiceMeshClient) UpdateMesh(ctx context.Context, mesh *servicemesh.Mesh) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMesh", ctx, mesh)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateMesh indicates an expected call of UpdateMesh.
func (mr *MockServiceMeshClientMockRecorder) UpdateMesh(ctx, mesh interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMesh", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateMesh), ctx, mesh)
}

// UpdateVirtualDeployment mocks base method.
func (m *MockServiceMeshClient) UpdateVirtualDeployment(ctx context.Context, vd *servicemesh.VirtualDeployment) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateVirtualDeployment", ctx, vd)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateVirtualDeployment indicates an expected call of UpdateVirtualDeployment.
func (mr *MockServiceMeshClientMockRecorder) UpdateVirtualDeployment(ctx, vd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateVirtualDeployment", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateVirtualDeployment), ctx, vd)
}

// UpdateVirtualService mocks base method.
func (m *MockServiceMeshClient) UpdateVirtualService(ctx context.Context, virtualService *servicemesh.VirtualService) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateVirtualService", ctx, virtualService)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateVirtualService indicates an expected call of UpdateVirtualService.
func (mr *MockServiceMeshClientMockRecorder) UpdateVirtualService(ctx, virtualService interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateVirtualService", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateVirtualService), ctx, virtualService)
}

// UpdateVirtualServiceRouteTable mocks base method.
func (m *MockServiceMeshClient) UpdateVirtualServiceRouteTable(ctx context.Context, vsrt *servicemesh.VirtualServiceRouteTable) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateVirtualServiceRouteTable", ctx, vsrt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateVirtualServiceRouteTable indicates an expected call of UpdateVirtualServiceRouteTable.
func (mr *MockServiceMeshClientMockRecorder) UpdateVirtualServiceRouteTable(ctx, vsrt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateVirtualServiceRouteTable", reflect.TypeOf((*MockServiceMeshClient)(nil).UpdateVirtualServiceRouteTable), ctx, vsrt)
}