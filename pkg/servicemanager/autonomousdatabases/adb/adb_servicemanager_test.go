/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOCIResponse struct {
	httpResp *http.Response
}

func (f *fakeOCIResponse) HTTPResponse() *http.Response { return f.httpResp }

type mockOciDbClient struct {
	createFn func(context.Context, database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error)
	listFn   func(context.Context, database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error)
	getFn    func(context.Context, database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)
	updateFn func(context.Context, database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)
}

func (m *mockOciDbClient) CreateAutonomousDatabase(ctx context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return database.CreateAutonomousDatabaseResponse{}, nil
}

func (m *mockOciDbClient) ListAutonomousDatabases(ctx context.Context, req database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return database.ListAutonomousDatabasesResponse{}, nil
}

func (m *mockOciDbClient) GetAutonomousDatabase(ctx context.Context, req database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return database.GetAutonomousDatabaseResponse{}, nil
}

func (m *mockOciDbClient) UpdateAutonomousDatabase(ctx context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return database.UpdateAutonomousDatabaseResponse{}, nil
}

func newTestManager() *AdbServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewAdbServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log)
}

func TestCreateAdb_ECPUUsesComputeFields(t *testing.T) {
	mgr := newTestManager()
	var captured database.CreateAutonomousDatabaseRequest

	ExportSetClientForTest(mgr, &mockOciDbClient{
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			captured = req
			return database.CreateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := databasev1beta1.AutonomousDatabases{
		Spec: databasev1beta1.AutonomousDatabasesSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "test-adb",
			DbName:               "TESTADB",
			DbWorkload:           "OLTP",
			DataStorageSizeInTBs: 1,
			ComputeModel:         "ECPU",
			ComputeCount:         2.0,
		},
	}

	_, err := mgr.CreateAdb(context.Background(), adb, "password")
	assert.NoError(t, err)
	details := captured.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Equal(t, database.CreateAutonomousDatabaseBaseComputeModelEnum("ECPU"), details.ComputeModel)
	assert.Equal(t, common.Float32(2.0), details.ComputeCount)
	assert.Nil(t, details.CpuCoreCount)
}

func TestCreateAdb_OCPUUsesCpuCoreCount(t *testing.T) {
	mgr := newTestManager()
	var captured database.CreateAutonomousDatabaseRequest

	ExportSetClientForTest(mgr, &mockOciDbClient{
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			captured = req
			return database.CreateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := databasev1beta1.AutonomousDatabases{
		Spec: databasev1beta1.AutonomousDatabasesSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "test-adb",
			DbName:               "TESTADB",
			DbWorkload:           "OLTP",
			DataStorageSizeInTBs: 1,
			CpuCoreCount:         2,
		},
	}

	_, err := mgr.CreateAdb(context.Background(), adb, "password")
	assert.NoError(t, err)
	details := captured.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Equal(t, common.Int(2), details.CpuCoreCount)
	assert.Empty(t, string(details.ComputeModel))
	assert.Nil(t, details.ComputeCount)
}

func TestGetCrdStatus_Happy(t *testing.T) {
	mgr := newTestManager()

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Status.OsokStatus.Ocid = "ocid1.autonomousdatabase.oc1..xxx"

	status, err := mgr.GetCrdStatus(adb)
	assert.NoError(t, err)
	assert.Equal(t, "ocid1.autonomousdatabase.oc1..xxx", string(status.Ocid))
}

func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := newTestManager()

	dbSystem := &mysqlv1beta1.MySqlDbSystem{}
	_, err := mgr.GetCrdStatus(dbSystem)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert the type assertion for Autonomous Databases")
}

func TestDelete_NoOcid(t *testing.T) {
	mgr := newTestManager()

	adb := &databasev1beta1.AutonomousDatabases{}
	done, err := mgr.Delete(context.Background(), adb)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := newTestManager()

	dbSystem := &mysqlv1beta1.MySqlDbSystem{}
	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestAdbRetryPolicy_Provisioning(t *testing.T) {
	mgr := newTestManager()
	shouldRetry := ExportAdbRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: database.GetAutonomousDatabaseResponse{
			AutonomousDatabase: database.AutonomousDatabase{LifecycleState: "PROVISIONING"},
		},
	}
	assert.True(t, shouldRetry(resp))
}

func TestAdbRetryPolicy_Available(t *testing.T) {
	mgr := newTestManager()
	shouldRetry := ExportAdbRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: database.GetAutonomousDatabaseResponse{
			AutonomousDatabase: database.AutonomousDatabase{LifecycleState: "AVAILABLE"},
		},
	}
	assert.False(t, shouldRetry(resp))
}

func TestAdbRetryPolicy_NonResponse(t *testing.T) {
	mgr := newTestManager()
	shouldRetry := ExportAdbRetryPredicate(mgr)

	assert.True(t, shouldRetry(common.OCIOperationResponse{}))
}

func TestAdbRetryNextDuration(t *testing.T) {
	mgr := newTestManager()
	nextDuration := ExportAdbRetryNextDuration(mgr)

	assert.Equal(t, 1*time.Second, nextDuration(common.OCIOperationResponse{AttemptNumber: 1}))
}

func TestExponentialBackoffPolicy_SuccessResponse(t *testing.T) {
	mgr := newTestManager()
	shouldRetry := ExportExponentialBackoffPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: &fakeOCIResponse{httpResp: &http.Response{StatusCode: 200}},
	}
	assert.False(t, shouldRetry(resp))
}

func TestExponentialBackoffPolicy_ErrorResponse(t *testing.T) {
	mgr := newTestManager()
	shouldRetry := ExportExponentialBackoffPredicate(mgr)

	resp := common.OCIOperationResponse{Error: errors.New("network error")}
	assert.True(t, shouldRetry(resp))
}

func TestExponentialBackoffNextDuration(t *testing.T) {
	mgr := newTestManager()
	nextDuration := ExportExponentialBackoffNextDuration(mgr)

	assert.Equal(t, 1*time.Second, nextDuration(common.OCIOperationResponse{AttemptNumber: 1}))
}
