/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOCIResponse struct {
	httpResp *http.Response
}

func (f *fakeOCIResponse) HTTPResponse() *http.Response { return f.httpResp }

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

func makeActiveAdb(id, displayName string) database.AutonomousDatabase {
	return database.AutonomousDatabase{
		Id:                   common.String(id),
		DisplayName:          common.String(displayName),
		DbName:               common.String("testdb"),
		CpuCoreCount:         common.Int(2),
		DataStorageSizeInTBs: common.Int(1),
		DbVersion:            common.String("19c"),
		DbWorkload:           database.AutonomousDatabaseDbWorkloadOltp,
		IsAutoScalingEnabled: common.Bool(false),
		IsFreeTier:           common.Bool(false),
		LicenseModel:         database.AutonomousDatabaseLicenseModelLicenseIncluded,
	}
}

func newTestManagerWithCreds(credClient *fakeCredentialClient) *AdbServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewAdbServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), credClient, nil, log)
}

func newTestManager() *AdbServiceManager {
	return newTestManagerWithCreds(nil)
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

func TestCreateAdb_MapsAdditionalOptionalFields(t *testing.T) {
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
			CompartmentId:                     "ocid1.compartment.oc1..example",
			DisplayName:                       "test-adb",
			DbName:                            "TESTADB",
			DbWorkload:                        "OLTP",
			DataStorageSizeInTBs:              1,
			CpuCoreCount:                      2,
			CharacterSet:                      "AL32UTF8",
			BackupRetentionPeriodInDays:       30,
			DataStorageSizeInGBs:              512,
			OcpuCount:                         3.5,
			KmsKeyId:                          "ocid1.key.oc1..example",
			VaultId:                           "ocid1.vault.oc1..example",
			SubnetId:                          "ocid1.subnet.oc1..example",
			NsgIds:                            []string{"ocid1.nsg.oc1..example"},
			PrivateEndpointLabel:              "adb-private",
			PrivateEndpointIp:                 "10.0.0.10",
			IsAccessControlEnabled:            true,
			WhitelistedIps:                    []string{"10.0.0.0/24"},
			CustomerContacts:                  []databasev1beta1.AutonomousDatabasesCustomerContact{{Email: "dba@example.com"}},
			ResourcePoolLeaderId:              "ocid1.autonomousdatabase.oc1..leader",
			ResourcePoolSummary:               databasev1beta1.AutonomousDatabasesResourcePoolSummary{PoolSize: 4, IsDisabled: true},
			ScheduledOperations:               []databasev1beta1.AutonomousDatabasesScheduledOperation{{DayOfWeek: databasev1beta1.AutonomousDatabasesScheduledOperationDayOfWeek{Name: "FRIDAY"}, ScheduledStartTime: "09:00", ScheduledStopTime: "18:00"}},
			IsAutoScalingForStorageEnabled:    true,
			DbToolsDetails:                    []databasev1beta1.AutonomousDatabasesDbToolsDetail{{Name: "ORDS", IsEnabled: true, ComputeCount: 1.5, MaxIdleTimeInMinutes: 60}},
			SecretId:                          "ocid1.vaultsecret.oc1..example",
			SecretVersionNumber:               2,
			DatabaseEdition:                   "ENTERPRISE_EDITION",
			AutonomousMaintenanceScheduleType: "REGULAR",
		},
	}

	_, err := mgr.CreateAdb(context.Background(), adb, "")
	assert.NoError(t, err)

	details := captured.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Nil(t, details.AdminPassword)
	assert.Equal(t, common.String("AL32UTF8"), details.CharacterSet)
	assert.Equal(t, common.Int(30), details.BackupRetentionPeriodInDays)
	assert.Equal(t, common.Int(512), details.DataStorageSizeInGBs)
	assert.Equal(t, common.Float32(3.5), details.OcpuCount)
	assert.Equal(t, common.String("ocid1.key.oc1..example"), details.KmsKeyId)
	assert.Equal(t, common.String("ocid1.vault.oc1..example"), details.VaultId)
	assert.Equal(t, common.String("ocid1.subnet.oc1..example"), details.SubnetId)
	assert.Equal(t, []string{"ocid1.nsg.oc1..example"}, details.NsgIds)
	assert.Equal(t, common.String("adb-private"), details.PrivateEndpointLabel)
	assert.Equal(t, common.String("10.0.0.10"), details.PrivateEndpointIp)
	assert.Equal(t, common.Bool(true), details.IsAccessControlEnabled)
	assert.Equal(t, []string{"10.0.0.0/24"}, details.WhitelistedIps)
	assert.Len(t, details.CustomerContacts, 1)
	assert.Equal(t, common.String("dba@example.com"), details.CustomerContacts[0].Email)
	if assert.NotNil(t, details.ResourcePoolSummary) {
		assert.Equal(t, common.Int(4), details.ResourcePoolSummary.PoolSize)
		assert.Equal(t, common.Bool(true), details.ResourcePoolSummary.IsDisabled)
	}
	assert.Len(t, details.ScheduledOperations, 1)
	if assert.NotNil(t, details.ScheduledOperations[0].DayOfWeek) {
		assert.Equal(t, database.DayOfWeekNameEnum("FRIDAY"), details.ScheduledOperations[0].DayOfWeek.Name)
	}
	assert.Equal(t, common.String("09:00"), details.ScheduledOperations[0].ScheduledStartTime)
	assert.Equal(t, common.String("18:00"), details.ScheduledOperations[0].ScheduledStopTime)
	assert.Equal(t, common.Bool(true), details.IsAutoScalingForStorageEnabled)
	assert.Len(t, details.DbToolsDetails, 1)
	assert.Equal(t, database.DatabaseToolNameEnum("ORDS"), details.DbToolsDetails[0].Name)
	assert.Equal(t, common.Bool(true), details.DbToolsDetails[0].IsEnabled)
	assert.Equal(t, common.Float32(1.5), details.DbToolsDetails[0].ComputeCount)
	assert.Equal(t, common.Int(60), details.DbToolsDetails[0].MaxIdleTimeInMinutes)
	assert.Equal(t, common.String("ocid1.vaultsecret.oc1..example"), details.SecretId)
	assert.Equal(t, common.Int(2), details.SecretVersionNumber)
	assert.Equal(t, database.AutonomousDatabaseSummaryDatabaseEditionEnum("ENTERPRISE_EDITION"), details.DatabaseEdition)
	assert.Equal(t, database.CreateAutonomousDatabaseBaseAutonomousMaintenanceScheduleTypeEnum("REGULAR"), details.AutonomousMaintenanceScheduleType)
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

	dbSystem := &mysqlv1beta1.DbSystem{}
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

	dbSystem := &mysqlv1beta1.DbSystem{}
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

func TestCreateOrUpdate_BindExistingAdb_NothingToUpdate(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	adbID := "ocid1.autonomousdatabase.oc1..xxx"
	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "test-adb"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.OCID(adbID), adb.Status.OsokStatus.Ocid)
}

func TestCreateOrUpdate_BindExistingAdb_UpdateNeeded(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	adbID := "ocid1.autonomousdatabase.oc1..yyy"
	updateCalled := false
	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "new-name"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

func TestCreateOrUpdate_BindExistingAdb_UpdateMultipleFields(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	adbID := "ocid1.autonomousdatabase.oc1..multi"
	updateCalled := false
	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "new-name"
	adb.Spec.CpuCoreCount = 4
	adb.Spec.DataStorageSizeInTBs = 2
	adb.Spec.IsAutoScalingEnabled = true

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

func TestCreateOrUpdate_FindExistingAdb(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	adbID := "ocid1.autonomousdatabase.oc1..found"
	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{
				Items: []database.AutonomousDatabaseSummary{
					{
						Id:             common.String(adbID),
						LifecycleState: database.AutonomousDatabaseSummaryLifecycleStateAvailable,
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "my-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.OCID(adbID), adb.Status.OsokStatus.Ocid)
}

func TestCreateOrUpdate_OciGetError(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{}, errors.New("OCI API error")
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = "ocid1.autonomousdatabase.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_OciListError(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, errors.New("list API error")
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_CreateNewAdb(t *testing.T) {
	newAdbID := "ocid1.autonomousdatabase.oc1..new"
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	createCalled := false
	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, _ database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			createCalled = true
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{Id: common.String(newAdbID)},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbID, "new-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "new-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled)
	assert.Equal(t, shared.OCID(newAdbID), adb.Status.OsokStatus.Ocid)
}

func TestCreateOrUpdate_CreateNewAdb_WithSecretIdSkipsAdminPasswordSecret(t *testing.T) {
	newAdbID := "ocid1.autonomousdatabase.oc1..vaultsecret"
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, errors.New("admin password secret should not be fetched")
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	var capturedReq database.CreateAutonomousDatabaseRequest
	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			capturedReq = req
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{Id: common.String(newAdbID)},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbID, "test-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.SecretId = "ocid1.vaultsecret.oc1..example"
	adb.Spec.SecretVersionNumber = 7
	adb.Spec.CpuCoreCount = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedReq.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Nil(t, details.AdminPassword)
	assert.Equal(t, common.String("ocid1.vaultsecret.oc1..example"), details.SecretId)
	assert.Equal(t, common.Int(7), details.SecretVersionNumber)
	assert.Equal(t, "ocid1.vaultsecret.oc1..example", adb.Status.LastAppliedSecretId)
	assert.Equal(t, 7, adb.Status.LastAppliedSecretVersionNumber)
}

func TestCreateOrUpdate_CreateNewAdb_GetSecretError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, errors.New("secret not found")
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_WithWallet_AlreadyExists(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..wallet"
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Name = "test-adb"
	adb.Namespace = "default"
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.Wallet.WalletPassword.Secret.SecretName = "wallet-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_WithWallet_PasswordSecretError(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..wallerr"
	callCount := 0
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("not found")
			}
			return nil, errors.New("wallet password secret not found")
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Name = "test-adb"
	adb.Namespace = "default"
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.Wallet.WalletPassword.Secret.SecretName = "wallet-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestDeleteAdb(t *testing.T) {
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ocid, err := mgr.DeleteAdb()
	assert.NoError(t, err)
	assert.Equal(t, "", ocid)
}

func TestCreateOrUpdate_BindExistingAdb_DefinedTagsChange(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..deftags"
	updateCalled := false
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.DefinedTags = map[string]shared.MapValue{
		"ns1": {"key1": "val1"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

func TestCreateOrUpdate_UpdateAdb_AdditionalFields(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..addfields"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DbWorkload = "DW"
	adb.Spec.IsFreeTier = true
	adb.Spec.LicenseModel = "BRING_YOUR_OWN_LICENSE"
	adb.Spec.DbVersion = "21c"
	adb.Spec.FreeFormTags = map[string]string{"env": "prod"}

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedUpdate.UpdateAutonomousDatabaseDetails
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsDbWorkloadEnum("DW"), details.DbWorkload)
	assert.Equal(t, common.Bool(true), details.IsFreeTier)
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsLicenseModelEnum("BRING_YOUR_OWN_LICENSE"), details.LicenseModel)
	assert.Equal(t, common.String("21c"), details.DbVersion)
	assert.Equal(t, map[string]string{"env": "prod"}, details.FreeformTags)
}

func TestCreateOrUpdate_BindExistingAdb_AdditionalOptionalFields(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..advanced"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	existing := makeActiveAdb(adbID, "test-adb")
	existing.PrivateEndpointLabel = common.String("old-private")
	existing.IsMtlsConnectionRequired = common.Bool(false)
	existing.NsgIds = []string{"ocid1.nsg.oc1..old"}
	existing.CustomerContacts = []database.CustomerContact{{Email: common.String("old@example.com")}}
	existing.ResourcePoolSummary = &database.ResourcePoolSummary{PoolSize: common.Int(1), IsDisabled: common.Bool(false)}
	existing.ScheduledOperations = []database.ScheduledOperationDetails{
		{
			DayOfWeek:          &database.DayOfWeek{Name: database.DayOfWeekNameEnum("MONDAY")},
			ScheduledStartTime: common.String("08:00"),
			ScheduledStopTime:  common.String("17:00"),
		},
	}
	existing.IsAutoScalingForStorageEnabled = common.Bool(false)
	existing.DbToolsDetails = []database.DatabaseTool{{Name: database.DatabaseToolNameEnum("ORDS"), IsEnabled: common.Bool(false)}}
	existing.LongTermBackupSchedule = &database.LongTermBackUpScheduleDetails{
		RepeatCadence:         database.LongTermBackUpScheduleDetailsRepeatCadenceEnum("WEEKLY"),
		RetentionPeriodInDays: common.Int(7),
	}
	existing.OpenMode = database.AutonomousDatabaseOpenModeEnum("READ_ONLY")
	existing.PermissionLevel = database.AutonomousDatabasePermissionLevelEnum("RESTRICTED")
	existing.IsRefreshableClone = common.Bool(false)
	existing.RefreshableMode = database.AutonomousDatabaseRefreshableModeEnum("MANUAL")
	existing.PeerDbIds = []string{"ocid1.autonomousdatabase.oc1..oldpeer"}

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: existing,
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.PrivateEndpointLabel = "new-private"
	adb.Spec.IsMtlsConnectionRequired = true
	adb.Spec.NsgIds = []string{"ocid1.nsg.oc1..new"}
	adb.Spec.CustomerContacts = []databasev1beta1.AutonomousDatabasesCustomerContact{{Email: "new@example.com"}}
	adb.Spec.ResourcePoolSummary = databasev1beta1.AutonomousDatabasesResourcePoolSummary{PoolSize: 3, IsDisabled: true}
	adb.Spec.ScheduledOperations = []databasev1beta1.AutonomousDatabasesScheduledOperation{
		{
			DayOfWeek:          databasev1beta1.AutonomousDatabasesScheduledOperationDayOfWeek{Name: "FRIDAY"},
			ScheduledStartTime: "09:00",
			ScheduledStopTime:  "18:00",
		},
	}
	adb.Spec.IsAutoScalingForStorageEnabled = true
	adb.Spec.DbToolsDetails = []databasev1beta1.AutonomousDatabasesDbToolsDetail{{Name: "ORDS", IsEnabled: true, ComputeCount: 2.5, MaxIdleTimeInMinutes: 45}}
	adb.Spec.LongTermBackupSchedule = databasev1beta1.AutonomousDatabasesLongTermBackupSchedule{
		RepeatCadence:         "MONTHLY",
		RetentionPeriodInDays: 30,
	}
	adb.Spec.OpenMode = "READ_WRITE"
	adb.Spec.PermissionLevel = "UNRESTRICTED"
	adb.Spec.IsRefreshableClone = true
	adb.Spec.RefreshableMode = "AUTOMATIC"
	adb.Spec.PeerDbId = "ocid1.autonomousdatabase.oc1..newpeer"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedUpdate.UpdateAutonomousDatabaseDetails
	assert.Equal(t, common.String("new-private"), details.PrivateEndpointLabel)
	assert.Equal(t, common.Bool(true), details.IsMtlsConnectionRequired)
	assert.Equal(t, []string{"ocid1.nsg.oc1..new"}, details.NsgIds)
	assert.Len(t, details.CustomerContacts, 1)
	assert.Equal(t, common.String("new@example.com"), details.CustomerContacts[0].Email)
	if assert.NotNil(t, details.ResourcePoolSummary) {
		assert.Equal(t, common.Int(3), details.ResourcePoolSummary.PoolSize)
		assert.Equal(t, common.Bool(true), details.ResourcePoolSummary.IsDisabled)
	}
	assert.Len(t, details.ScheduledOperations, 1)
	if assert.NotNil(t, details.ScheduledOperations[0].DayOfWeek) {
		assert.Equal(t, database.DayOfWeekNameEnum("FRIDAY"), details.ScheduledOperations[0].DayOfWeek.Name)
	}
	assert.Equal(t, common.String("09:00"), details.ScheduledOperations[0].ScheduledStartTime)
	assert.Equal(t, common.String("18:00"), details.ScheduledOperations[0].ScheduledStopTime)
	assert.Equal(t, common.Bool(true), details.IsAutoScalingForStorageEnabled)
	assert.Len(t, details.DbToolsDetails, 1)
	assert.Equal(t, database.DatabaseToolNameEnum("ORDS"), details.DbToolsDetails[0].Name)
	assert.Equal(t, common.Bool(true), details.DbToolsDetails[0].IsEnabled)
	assert.Equal(t, common.Float32(2.5), details.DbToolsDetails[0].ComputeCount)
	assert.Equal(t, common.Int(45), details.DbToolsDetails[0].MaxIdleTimeInMinutes)
	if assert.NotNil(t, details.LongTermBackupSchedule) {
		assert.Equal(t, database.LongTermBackUpScheduleDetailsRepeatCadenceEnum("MONTHLY"), details.LongTermBackupSchedule.RepeatCadence)
		assert.Equal(t, common.Int(30), details.LongTermBackupSchedule.RetentionPeriodInDays)
	}
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsOpenModeEnum("READ_WRITE"), details.OpenMode)
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsPermissionLevelEnum("UNRESTRICTED"), details.PermissionLevel)
	assert.Equal(t, common.Bool(true), details.IsRefreshableClone)
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsRefreshableModeEnum("AUTOMATIC"), details.RefreshableMode)
	assert.Equal(t, common.String("ocid1.autonomousdatabase.oc1..newpeer"), details.PeerDbId)
}

func TestCreateOrUpdate_BindExistingAdb_NoUpdateWhenAdditionalFieldsMatch(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..match"
	updateCalled := false
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	existing := makeActiveAdb(adbID, "test-adb")
	existing.PrivateEndpointLabel = common.String("shared-private")
	existing.NsgIds = []string{"ocid1.nsg.oc1..shared"}
	existing.CustomerContacts = []database.CustomerContact{{Email: common.String("same@example.com")}}
	existing.ResourcePoolSummary = &database.ResourcePoolSummary{PoolSize: common.Int(2), IsDisabled: common.Bool(false)}

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: existing,
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.PrivateEndpointLabel = "shared-private"
	adb.Spec.NsgIds = []string{"ocid1.nsg.oc1..shared"}
	adb.Spec.CustomerContacts = []databasev1beta1.AutonomousDatabasesCustomerContact{{Email: "same@example.com"}}
	adb.Spec.ResourcePoolSummary = databasev1beta1.AutonomousDatabasesResourcePoolSummary{PoolSize: 2, IsDisabled: false}

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled)
}

func TestCreateOrUpdate_BindExistingAdb_SecretReferenceUpdateWhenStatusMissing(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..secretmissing"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.SecretId = "ocid1.vaultsecret.oc1..desired"
	adb.Spec.SecretVersionNumber = 4

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, common.String("ocid1.vaultsecret.oc1..desired"), capturedUpdate.UpdateAutonomousDatabaseDetails.SecretId)
	assert.Equal(t, common.Int(4), capturedUpdate.UpdateAutonomousDatabaseDetails.SecretVersionNumber)
	assert.Equal(t, "ocid1.vaultsecret.oc1..desired", adb.Status.LastAppliedSecretId)
	assert.Equal(t, 4, adb.Status.LastAppliedSecretVersionNumber)
}

func TestCreateOrUpdate_BindExistingAdb_NoUpdateWhenSecretReferenceAlreadyApplied(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..secretmatch"
	updateCalled := false
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.SecretId = "ocid1.vaultsecret.oc1..desired"
	adb.Spec.SecretVersionNumber = 4
	adb.Status.LastAppliedSecretId = "ocid1.vaultsecret.oc1..desired"
	adb.Status.LastAppliedSecretVersionNumber = 4

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled)
	assert.Equal(t, "ocid1.vaultsecret.oc1..desired", adb.Status.LastAppliedSecretId)
	assert.Equal(t, 4, adb.Status.LastAppliedSecretVersionNumber)
}

func TestCreateOrUpdate_BindExistingAdb_SecretReferenceChangeTriggersUpdate(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..secretchange"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.SecretId = "ocid1.vaultsecret.oc1..new"
	adb.Spec.SecretVersionNumber = 2
	adb.Status.LastAppliedSecretId = "ocid1.vaultsecret.oc1..old"
	adb.Status.LastAppliedSecretVersionNumber = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, common.String("ocid1.vaultsecret.oc1..new"), capturedUpdate.UpdateAutonomousDatabaseDetails.SecretId)
	assert.Equal(t, common.Int(2), capturedUpdate.UpdateAutonomousDatabaseDetails.SecretVersionNumber)
	assert.Equal(t, "ocid1.vaultsecret.oc1..new", adb.Status.LastAppliedSecretId)
	assert.Equal(t, 2, adb.Status.LastAppliedSecretVersionNumber)
}

func TestCreateOrUpdate_WalletPassword_MissingKey(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..walpwd"
	callCount := 0
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("wallet not found")
			}
			return map[string][]byte{"wrongkey": []byte("value")}, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Name = "test-adb"
	adb.Namespace = "default"
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.Wallet.WalletPassword.Secret.SecretName = "wallet-pwd-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "walletPassword")
	assert.False(t, resp.IsSuccessful)
}

type fakeServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f *fakeServiceError) GetHTTPStatusCode() int  { return f.statusCode }
func (f *fakeServiceError) GetMessage() string      { return f.message }
func (f *fakeServiceError) GetCode() string         { return f.code }
func (f *fakeServiceError) GetOpcRequestID() string { return "" }
func (f *fakeServiceError) Error() string {
	return fmt.Sprintf("%d %s: %s", f.statusCode, f.code, f.message)
}

func TestCreateOrUpdate_CreateNewAdb_MissingPasswordKey(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"wrongkey": []byte("value")}, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password key")
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_CreateNewAdb_WithVersionAndLicense(t *testing.T) {
	newAdbID := "ocid1.autonomousdatabase.oc1..verlic"
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	var capturedReq database.CreateAutonomousDatabaseRequest
	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			capturedReq = req
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{Id: common.String(newAdbID)},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbID, "test-adb"),
			}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 2
	adb.Spec.DbVersion = "21c"
	adb.Spec.LicenseModel = "BRING_YOUR_OWN_LICENSE"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedReq.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Equal(t, common.String("21c"), details.DbVersion)
	assert.Equal(t, database.CreateAutonomousDatabaseBaseLicenseModelEnum("BRING_YOUR_OWN_LICENSE"), details.LicenseModel)
}

func TestCreateOrUpdate_BindExistingAdb_DbNameChange(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..dbname"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "new-name"
	adb.Spec.DbName = "newdb"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, common.String("newdb"), capturedUpdate.UpdateAutonomousDatabaseDetails.DbName)
}

func TestCreateOrUpdate_CreateNewAdb_InvalidParameterError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, _ database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			return database.CreateAutonomousDatabaseResponse{},
				&fakeServiceError{statusCode: 400, code: "InvalidParameter", message: "bad param"}
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_CreateNewAdb_OciCreateError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManagerWithCreds(credClient)

	ExportSetClientForTest(mgr, &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, _ database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			return database.CreateAutonomousDatabaseResponse{},
				&fakeServiceError{statusCode: 500, code: "InternalServerError", message: "server error"}
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_BindExistingAdb_UpdateNeeded_WithCreatedAt(t *testing.T) {
	adbID := "ocid1.autonomousdatabase.oc1..creat"
	mgr := newTestManagerWithCreds(&fakeCredentialClient{})

	ExportSetClientForTest(mgr, &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbID, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	})

	adb := &databasev1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = shared.OCID(adbID)
	adb.Spec.DisplayName = "new-name"
	ts := metav1.NewTime(time.Now())
	adb.Status.OsokStatus.CreatedAt = &ts

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

func TestGetCredentialMap_Valid(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("tnsnames.ora")
	assert.NoError(t, err)
	_, err = fw.Write([]byte("MY_SERVICE = (DESCRIPTION=...)"))
	assert.NoError(t, err)
	assert.NoError(t, zw.Close())

	resp := database.GenerateAutonomousDatabaseWalletResponse{
		Content: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}

	credMap, err := ExportGetCredentialMapForTest("test-adb", resp)
	assert.NoError(t, err)
	assert.Contains(t, credMap, "tnsnames.ora")
	assert.Equal(t, []byte("MY_SERVICE = (DESCRIPTION=...)"), credMap["tnsnames.ora"])
}

func TestGetCredentialMap_MultipleFiles(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"tnsnames.ora", "sqlnet.ora", "cwallet.sso"} {
		fw, err := zw.Create(name)
		assert.NoError(t, err)
		_, err = fw.Write([]byte("content of " + name))
		assert.NoError(t, err)
	}
	assert.NoError(t, zw.Close())

	resp := database.GenerateAutonomousDatabaseWalletResponse{
		Content: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}

	credMap, err := ExportGetCredentialMapForTest("test-adb", resp)
	assert.NoError(t, err)
	assert.Len(t, credMap, 3)
	assert.Equal(t, []byte("content of tnsnames.ora"), credMap["tnsnames.ora"])
	assert.Equal(t, []byte("content of sqlnet.ora"), credMap["sqlnet.ora"])
	assert.Equal(t, []byte("content of cwallet.sso"), credMap["cwallet.sso"])
}
