/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	getSecretFn    func(ctx context.Context, name, ns string) (map[string][]byte, error)
	updateSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

type mockOciDbSystemClient struct {
	createFn func(context.Context, mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error)
	getFn    func(context.Context, mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error)
	listFn   func(context.Context, mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error)
	updateFn func(context.Context, mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error)
}

func (m *mockOciDbSystemClient) CreateDbSystem(ctx context.Context, req mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return mysql.CreateDbSystemResponse{}, nil
}

func (m *mockOciDbSystemClient) GetDbSystem(ctx context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return mysql.GetDbSystemResponse{}, nil
}

func (m *mockOciDbSystemClient) ListDbSystems(ctx context.Context, req mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return mysql.ListDbSystemsResponse{}, nil
}

func (m *mockOciDbSystemClient) UpdateDbSystem(ctx context.Context, req mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return mysql.UpdateDbSystemResponse{}, nil
}

func newTestManager(credClient *fakeCredentialClient) *DbSystemServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewDbSystemServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), credClient, nil, log)
}

func makeActiveDbSystem(id, displayName string) mysql.DbSystem {
	port := 3306
	portX := 33060
	desc := "test description"
	hostname := "mysql.example.com"
	ip := "10.0.0.1"
	az := "AD-1"
	fd := "FAULT-DOMAIN-1"
	cfgID := "ocid1.mysqlconfiguration.oc1..xxx"
	return mysql.DbSystem{
		Id:                 common.String(id),
		DisplayName:        common.String(displayName),
		Description:        &desc,
		LifecycleState:     mysql.DbSystemLifecycleStateActive,
		Port:               &port,
		PortX:              &portX,
		HostnameLabel:      &hostname,
		IpAddress:          &ip,
		AvailabilityDomain: &az,
		FaultDomain:        &fd,
		ConfigurationId:    &cfgID,
		CompartmentId:      common.String("ocid1.compartment.oc1..xxx"),
	}
}

func TestCreateDbSystem_OmitsOptionalFieldsWhenEmpty(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	var captured mysql.CreateDbSystemRequest

	ExportSetClientForTest(mgr, &mockOciDbSystemClient{
		createFn: func(_ context.Context, req mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
			captured = req
			return mysql.CreateDbSystemResponse{}, nil
		},
	})

	dbSystem := mysqlv1beta1.MySqlDbSystem{
		Spec: mysqlv1beta1.MySqlDbSystemSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			ShapeName:            "MySQL.VM.Standard.E4.1.8GB",
			AvailabilityDomain:   "AD-1",
			FaultDomain:          "FAULT-DOMAIN-1",
			DataStorageSizeInGBs: 50,
			SubnetId:             "ocid1.subnet.oc1..example",
			DisplayName:          "test-dbsystem",
		},
	}

	_, err := mgr.CreateDbSystem(context.Background(), dbSystem, "admin", "password")
	assert.NoError(t, err)
	assert.Nil(t, captured.CreateDbSystemDetails.Description)
	assert.Nil(t, captured.CreateDbSystemDetails.Port)
	assert.Nil(t, captured.CreateDbSystemDetails.PortX)
	assert.Nil(t, captured.CreateDbSystemDetails.ConfigurationId)
}

func TestCreateOrUpdate_LifecycleProvisioningWaitsForActive(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, name, _ string) (map[string][]byte, error) {
			if name == "admin-user" {
				return map[string][]byte{"username": []byte("admin")}, nil
			}
			return map[string][]byte{"password": []byte("password")}, nil
		},
	}
	mgr := newTestManager(credClient)

	const dbSystemID = "ocid1.mysqldbsystem.oc1..example"
	ExportSetClientForTest(mgr, &mockOciDbSystemClient{
		createFn: func(_ context.Context, _ mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
			return mysql.CreateDbSystemResponse{
				DbSystem: mysql.DbSystem{Id: common.String(dbSystemID)},
			}, nil
		},
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: mysql.DbSystem{
					Id:             common.String(dbSystemID),
					DisplayName:    common.String("test-dbsystem"),
					LifecycleState: mysql.DbSystemLifecycleStateCreating,
				},
			}, nil
		},
	})

	dbSystem := &mysqlv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec = mysqlv1beta1.MySqlDbSystemSpec{
		CompartmentId:        "ocid1.compartment.oc1..example",
		ShapeName:            "MySQL.VM.Standard.E4.1.8GB",
		AvailabilityDomain:   "AD-1",
		FaultDomain:          "FAULT-DOMAIN-1",
		DataStorageSizeInGBs: 50,
		SubnetId:             "ocid1.subnet.oc1..example",
		DisplayName:          "test-dbsystem",
		AdminUsername:        shared.UsernameSource{Secret: shared.SecretSource{SecretName: "admin-user"}},
		AdminPassword:        shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-password"}},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, strings.Contains(err.Error(), "waiting for ACTIVE"))
	assert.False(t, credClient.createCalled, "secret creation should wait until ACTIVE")
}

func TestGetCrdStatus_Happy(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	dbSystem := &mysqlv1beta1.MySqlDbSystem{}
	dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..xxx"

	status, err := mgr.GetCrdStatus(dbSystem)
	assert.NoError(t, err)
	assert.Equal(t, "ocid1.mysqldbsystem.oc1..xxx", string(status.Ocid))
}

func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	stream := &streamingv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert the type assertion for MySqlDbSystem")
}

func TestDelete_NoOcid(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	dbSystem := &mysqlv1beta1.MySqlDbSystem{}
	done, err := mgr.Delete(context.Background(), dbSystem)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	stream := &streamingv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestGetCredentialMapForDbSystem(t *testing.T) {
	credMap, err := GetCredentialMapForTest(makeActiveDbSystem("ocid1.mysqldbsystem.oc1..xxx", "test-dbsystem"))

	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.1", string(credMap["PrivateIPAddress"]))
	assert.Equal(t, "mysql.example.com", string(credMap["InternalFQDN"]))
	assert.Equal(t, "AD-1", string(credMap["AvailabilityDomain"]))
	assert.Equal(t, "FAULT-DOMAIN-1", string(credMap["FaultDomain"]))
	assert.Equal(t, "3306", string(credMap["MySQLPort"]))
	assert.Equal(t, "33060", string(credMap["MySQLXProtocolPort"]))
	assert.Contains(t, credMap, "Endpoints")
}

func TestGetCredentialMapForDbSystem_NilHostname(t *testing.T) {
	port := 3306
	portX := 33060
	ip := "10.0.0.2"
	az := "AD-2"
	fd := "FAULT-DOMAIN-2"
	dbSystem := mysql.DbSystem{
		Id:                 common.String("ocid1.mysqldbsystem.oc1..yyy"),
		IpAddress:          &ip,
		AvailabilityDomain: &az,
		FaultDomain:        &fd,
		Port:               &port,
		PortX:              &portX,
	}

	credMap, err := GetCredentialMapForTest(dbSystem)
	assert.NoError(t, err)
	assert.Equal(t, "", string(credMap["InternalFQDN"]))
}

func TestDeleteFromSecret(t *testing.T) {
	deleteCalled := false
	credClient := &fakeCredentialClient{
		deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
			deleteCalled = true
			return true, nil
		},
	}
	mgr := newTestManager(credClient)

	ok, err := ExportDeleteFromSecretForTest(mgr, context.Background(), "default", "my-dbsystem")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, deleteCalled)
}

func TestDbSystemRetryPolicy_Creating(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportGetDbSystemRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: mysql.GetDbSystemResponse{DbSystem: mysql.DbSystem{LifecycleState: "CREATING"}},
	}
	assert.True(t, shouldRetry(resp))
}

func TestDbSystemRetryPolicy_Active(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportGetDbSystemRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: mysql.GetDbSystemResponse{DbSystem: mysql.DbSystem{LifecycleState: "ACTIVE"}},
	}
	assert.False(t, shouldRetry(resp))
}

func TestDbSystemRetryPolicy_NonResponse(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportGetDbSystemRetryPredicate(mgr)

	assert.True(t, shouldRetry(common.OCIOperationResponse{}))
}

func TestDbSystemRetryNextDuration(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	nextDuration := ExportGetDbSystemNextDuration(mgr)

	assert.Equal(t, 1*time.Minute, nextDuration(common.OCIOperationResponse{AttemptNumber: 1}))
}
