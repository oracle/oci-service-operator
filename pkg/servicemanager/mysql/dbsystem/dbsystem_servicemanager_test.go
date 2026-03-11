/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem_test

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCredentialClient struct {
	createCalled bool
	getSecretFn  func(ctx context.Context, name, ns string) (map[string][]byte, error)
}

func (f *fakeCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	f.createCalled = true
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
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

func TestCreateDbSystem_OmitsOptionalFieldsWhenEmpty(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	var captured mysql.CreateDbSystemRequest

	ExportSetClientForTest(mgr, &mockOciDbSystemClient{
		createFn: func(_ context.Context, req mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
			captured = req
			return mysql.CreateDbSystemResponse{}, nil
		},
	})

	dbSystem := ociv1beta1.MySqlDbSystem{
		Spec: ociv1beta1.MySqlDbSystemSpec{
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

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec = ociv1beta1.MySqlDbSystemSpec{
		CompartmentId:        "ocid1.compartment.oc1..example",
		ShapeName:            "MySQL.VM.Standard.E4.1.8GB",
		AvailabilityDomain:   "AD-1",
		FaultDomain:          "FAULT-DOMAIN-1",
		DataStorageSizeInGBs: 50,
		SubnetId:             "ocid1.subnet.oc1..example",
		DisplayName:          "test-dbsystem",
		AdminUsername:        ociv1beta1.UsernameSource{Secret: ociv1beta1.SecretSource{SecretName: "admin-user"}},
		AdminPassword:        ociv1beta1.PasswordSource{Secret: ociv1beta1.SecretSource{SecretName: "admin-password"}},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, strings.Contains(err.Error(), "waiting for ACTIVE"))
	assert.False(t, credClient.createCalled, "secret creation should wait until ACTIVE")
}
