/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type mockOciDbClient struct {
	createFn func(context.Context, database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error)
}

func (m *mockOciDbClient) CreateAutonomousDatabase(ctx context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return database.CreateAutonomousDatabaseResponse{}, nil
}

func (m *mockOciDbClient) ListAutonomousDatabases(context.Context, database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
	return database.ListAutonomousDatabasesResponse{}, nil
}

func (m *mockOciDbClient) GetAutonomousDatabase(context.Context, database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
	return database.GetAutonomousDatabaseResponse{}, nil
}

func (m *mockOciDbClient) UpdateAutonomousDatabase(context.Context, database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
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

	adb := ociv1beta1.AutonomousDatabases{
		Spec: ociv1beta1.AutonomousDatabasesSpec{
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

	adb := ociv1beta1.AutonomousDatabases{
		Spec: ociv1beta1.AutonomousDatabasesSpec{
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
