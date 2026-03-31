/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"strings"
	"testing"
	"time"

	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDbSystemBody struct {
	Id             string `json:"id,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type fakeCreateDbSystemResponse struct {
	DbSystem fakeDbSystemBody `presentIn:"body"`
}

func TestDbSystemServiceManagerCreateOrUpdateProjectsSourceVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source mysqlv1beta1.DbSystemSource
		assert func(*testing.T, mysqlsdk.CreateDbSystemSourceDetails)
	}{
		{
			name: "backup",
			source: mysqlv1beta1.DbSystemSource{
				SourceType: "BACKUP",
				BackupId:   "ocid1.mysqlbackup.oc1..backup",
			},
			assert: func(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
				t.Helper()

				backup, ok := source.(mysqlsdk.CreateDbSystemSourceFromBackupDetails)
				if !ok {
					t.Fatalf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceFromBackupDetails{})
				}
				if backup.BackupId == nil || *backup.BackupId != "ocid1.mysqlbackup.oc1..backup" {
					t.Fatalf("BackupId = %v, want backup OCID", backup.BackupId)
				}
			},
		},
		{
			name: "pitr",
			source: mysqlv1beta1.DbSystemSource{
				SourceType:    "PITR",
				DbSystemId:    "ocid1.mysqldbsystem.oc1..source",
				RecoveryPoint: "2026-03-01T02:03:04Z",
			},
			assert: func(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
				t.Helper()

				pitr, ok := source.(mysqlsdk.CreateDbSystemSourceFromPitrDetails)
				if !ok {
					t.Fatalf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceFromPitrDetails{})
				}
				if pitr.DbSystemId == nil || *pitr.DbSystemId != "ocid1.mysqldbsystem.oc1..source" {
					t.Fatalf("DbSystemId = %v, want source DB System OCID", pitr.DbSystemId)
				}
				if pitr.RecoveryPoint == nil || pitr.RecoveryPoint.Format(time.RFC3339) != "2026-03-01T02:03:04Z" {
					t.Fatalf("RecoveryPoint = %v, want 2026-03-01T02:03:04Z", pitr.RecoveryPoint)
				}
			},
		},
		{
			name: "import-url",
			source: mysqlv1beta1.DbSystemSource{
				SourceType: "IMPORTURL",
				SourceUrl:  "https://objectstorage.example.com/n/tenant/b/bucket/o/import.manifest.json",
			},
			assert: func(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
				t.Helper()

				importURL, ok := source.(mysqlsdk.CreateDbSystemSourceImportFromUrlDetails)
				if !ok {
					t.Fatalf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceImportFromUrlDetails{})
				}
				if importURL.SourceUrl == nil || *importURL.SourceUrl != "https://objectstorage.example.com/n/tenant/b/bucket/o/import.manifest.json" {
					t.Fatalf("SourceUrl = %v, want import URL", importURL.SourceUrl)
				}
			},
		},
		{
			name: "none",
			source: mysqlv1beta1.DbSystemSource{
				SourceType: "NONE",
			},
			assert: func(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
				t.Helper()

				if _, ok := source.(mysqlsdk.CreateDbSystemSourceFromNoneDetails); !ok {
					t.Fatalf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceFromNoneDetails{})
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var captured mysqlsdk.CreateDbSystemRequest
			manager := &DbSystemServiceManager{
				client: generatedruntime.NewServiceClient[*mysqlv1beta1.DbSystem](generatedruntime.Config[*mysqlv1beta1.DbSystem]{
					Kind:    "DbSystem",
					SDKName: "DbSystem",
					Create: &generatedruntime.Operation{
						NewRequest: func() any { return &mysqlsdk.CreateDbSystemRequest{} },
						Call: func(_ context.Context, request any) (any, error) {
							captured = *request.(*mysqlsdk.CreateDbSystemRequest)
							return fakeCreateDbSystemResponse{
								DbSystem: fakeDbSystemBody{
									Id:             "ocid1.mysqldbsystem.oc1..created",
									LifecycleState: "ACTIVE",
								},
							}, nil
						},
					},
				}),
			}

			resource := &mysqlv1beta1.DbSystem{
				Spec: mysqlv1beta1.DbSystemSpec{
					CompartmentId: "ocid1.compartment.oc1..example",
					ShapeName:     "MySQL.VM.Standard.E3.1.8GB",
					SubnetId:      "ocid1.subnet.oc1..example",
					Source:        tt.source,
				},
			}

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatal("CreateOrUpdate() should report success")
			}
			if captured.CreateDbSystemDetails.Source == nil {
				t.Fatal("CreateDbSystemDetails.Source should be projected into the OCI request")
			}
			tt.assert(t, captured.CreateDbSystemDetails.Source)
		})
	}
}

func TestDbSystemServiceManagerCreateOrUpdateRejectsWrongObjectType(t *testing.T) {
	t.Parallel()

	manager := &DbSystemServiceManager{}

	response, err := manager.CreateOrUpdate(context.Background(), &mysqlv1beta1.Backup{}, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want conversion failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for the wrong resource type")
	}
	if !strings.Contains(err.Error(), "expected *mysqlv1beta1.DbSystem") {
		t.Fatalf("CreateOrUpdate() error = %q, want DbSystem conversion failure", err.Error())
	}
}
