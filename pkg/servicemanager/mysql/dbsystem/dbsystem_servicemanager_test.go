/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDbSystemBody struct {
	Id             string `json:"id,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type fakeCreateDbSystemResponse struct {
	DbSystem fakeDbSystemBody `presentIn:"body"`
}

type createOrUpdateSourceVariantTestCase struct {
	name   string
	source mysqlv1beta1.DbSystemSource
	assert func(*testing.T, mysqlsdk.CreateDbSystemSourceDetails)
}

type quickDbSystemSourceCase struct {
	Source mysqlv1beta1.DbSystemSource
}

func (quickDbSystemSourceCase) Generate(rand *rand.Rand, _ int) reflect.Value {
	var source mysqlv1beta1.DbSystemSource

	switch rand.Intn(4) {
	case 0:
		source = mysqlv1beta1.DbSystemSource{
			SourceType: "BACKUP",
			BackupId:   randomOCID(rand, "mysqlbackup"),
		}
	case 1:
		source = mysqlv1beta1.DbSystemSource{
			SourceType:    "PITR",
			DbSystemId:    randomOCID(rand, "mysqldbsystem"),
			RecoveryPoint: randomRecoveryPoint(rand),
		}
	case 2:
		source = mysqlv1beta1.DbSystemSource{
			SourceType: "IMPORTURL",
			SourceUrl:  randomImportURL(rand),
		}
	default:
		source = mysqlv1beta1.DbSystemSource{
			SourceType: "NONE",
		}
	}

	return reflect.ValueOf(quickDbSystemSourceCase{Source: source})
}

func (c quickDbSystemSourceCase) matches(projected mysqlsdk.CreateDbSystemSourceDetails) error {
	switch c.Source.SourceType {
	case "BACKUP":
		return matchBackupSourceDetails(projected, c.Source.BackupId)
	case "PITR":
		return matchPITRSourceDetails(projected, c.Source.DbSystemId, c.Source.RecoveryPoint)
	case "IMPORTURL":
		return matchImportURLSourceDetails(projected, c.Source.SourceUrl)
	case "NONE":
		return matchNoneSourceDetails(projected)
	default:
		return fmt.Errorf("unexpected sourceType %q", c.Source.SourceType)
	}
}

func (c quickDbSystemSourceCase) equal(other quickDbSystemSourceCase) bool {
	return reflect.DeepEqual(c.Source, other.Source)
}

func TestDbSystemServiceManagerCreateOrUpdateProjectsSourceVariants(t *testing.T) {
	t.Parallel()

	tests := []createOrUpdateSourceVariantTestCase{
		{
			name: "backup",
			source: mysqlv1beta1.DbSystemSource{
				SourceType: "BACKUP",
				BackupId:   "ocid1.mysqlbackup.oc1..backup",
			},
			assert: assertBackupSourceDetails,
		},
		{
			name: "pitr",
			source: mysqlv1beta1.DbSystemSource{
				SourceType:    "PITR",
				DbSystemId:    "ocid1.mysqldbsystem.oc1..source",
				RecoveryPoint: "2026-03-01T02:03:04Z",
			},
			assert: assertPITRSourceDetails,
		},
		{
			name: "import-url",
			source: mysqlv1beta1.DbSystemSource{
				SourceType: "IMPORTURL",
				SourceUrl:  "https://objectstorage.example.com/n/tenant/b/bucket/o/import.manifest.json",
			},
			assert: assertImportURLSourceDetails,
		},
		{
			name: "none",
			source: mysqlv1beta1.DbSystemSource{
				SourceType: "NONE",
			},
			assert: assertNoneSourceDetails,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.assert(t, captureCreateDbSystemSource(t, tt.source))
		})
	}
}

func TestDbSystemServiceManagerCreateOrUpdateProjectsSourceVariantsQuick(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(sourceCase quickDbSystemSourceCase) bool {
		projected, projectErr := projectCreateDbSystemSource(sourceCase.Source)
		if projectErr != nil {
			t.Logf("project %q source %#v: %v", sourceCase.Source.SourceType, sourceCase.Source, projectErr)
			return false
		}
		if matchErr := sourceCase.matches(projected); matchErr != nil {
			t.Logf("match %q source %#v: %v", sourceCase.Source.SourceType, sourceCase.Source, matchErr)
			return false
		}
		return true
	}, &quick.Config{MaxCount: 64})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDbSystemServiceManagerCreateOrUpdateRejectsSourceMutationsQuick(t *testing.T) {
	t.Parallel()

	manager := newDbSystemSourceMutationManager()

	err := quick.Check(func(current, desired quickDbSystemSourceCase) bool {
		if current.equal(desired) {
			return true
		}

		resource := newExistingDbSystemWithSource(current.Source, desired.Source)
		_, createOrUpdateErr := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if createOrUpdateErr == nil {
			t.Logf("CreateOrUpdate() error = nil for current=%#v desired=%#v", current.Source, desired.Source)
			return false
		}
		if !strings.Contains(createOrUpdateErr.Error(), "require replacement when source changes") {
			t.Logf("CreateOrUpdate() error = %v for current=%#v desired=%#v", createOrUpdateErr, current.Source, desired.Source)
			return false
		}
		return true
	}, &quick.Config{MaxCount: 64})
	if err != nil {
		t.Fatal(err)
	}
}

func captureCreateDbSystemSource(t *testing.T, source mysqlv1beta1.DbSystemSource) mysqlsdk.CreateDbSystemSourceDetails {
	t.Helper()

	projected, err := projectCreateDbSystemSource(source)
	if err != nil {
		t.Fatal(err)
	}
	return projected
}

func projectCreateDbSystemSource(source mysqlv1beta1.DbSystemSource) (mysqlsdk.CreateDbSystemSourceDetails, error) {
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
			Source:        source,
		},
	}

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		return nil, fmt.Errorf("CreateOrUpdate() error = %w", err)
	}
	if !response.IsSuccessful {
		return nil, fmt.Errorf("CreateOrUpdate() should report success")
	}
	if captured.Source == nil {
		return nil, fmt.Errorf("CreateDbSystemDetails.Source should be projected into the OCI request")
	}

	return captured.Source, nil
}

func assertBackupSourceDetails(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
	t.Helper()

	if err := matchBackupSourceDetails(source, "ocid1.mysqlbackup.oc1..backup"); err != nil {
		t.Fatal(err)
	}
}

func assertPITRSourceDetails(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
	t.Helper()

	if err := matchPITRSourceDetails(source, "ocid1.mysqldbsystem.oc1..source", "2026-03-01T02:03:04Z"); err != nil {
		t.Fatal(err)
	}
}

func assertImportURLSourceDetails(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
	t.Helper()

	if err := matchImportURLSourceDetails(source, "https://objectstorage.example.com/n/tenant/b/bucket/o/import.manifest.json"); err != nil {
		t.Fatal(err)
	}
}

func assertNoneSourceDetails(t *testing.T, source mysqlsdk.CreateDbSystemSourceDetails) {
	t.Helper()

	if err := matchNoneSourceDetails(source); err != nil {
		t.Fatal(err)
	}
}

func matchBackupSourceDetails(source mysqlsdk.CreateDbSystemSourceDetails, wantBackupID string) error {
	backup, ok := source.(mysqlsdk.CreateDbSystemSourceFromBackupDetails)
	if !ok {
		return fmt.Errorf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceFromBackupDetails{})
	}
	if backup.BackupId == nil || *backup.BackupId != wantBackupID {
		return fmt.Errorf("BackupId = %v, want %q", backup.BackupId, wantBackupID)
	}
	return nil
}

func matchPITRSourceDetails(source mysqlsdk.CreateDbSystemSourceDetails, wantDbSystemID, wantRecoveryPoint string) error {
	pitr, ok := source.(mysqlsdk.CreateDbSystemSourceFromPitrDetails)
	if !ok {
		return fmt.Errorf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceFromPitrDetails{})
	}
	if pitr.DbSystemId == nil || *pitr.DbSystemId != wantDbSystemID {
		return fmt.Errorf("DbSystemId = %v, want %q", pitr.DbSystemId, wantDbSystemID)
	}
	if pitr.RecoveryPoint == nil || pitr.RecoveryPoint.Format(time.RFC3339) != wantRecoveryPoint {
		return fmt.Errorf("RecoveryPoint = %v, want %q", pitr.RecoveryPoint, wantRecoveryPoint)
	}
	return nil
}

func matchImportURLSourceDetails(source mysqlsdk.CreateDbSystemSourceDetails, wantSourceURL string) error {
	importURL, ok := source.(mysqlsdk.CreateDbSystemSourceImportFromUrlDetails)
	if !ok {
		return fmt.Errorf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceImportFromUrlDetails{})
	}
	if importURL.SourceUrl == nil || *importURL.SourceUrl != wantSourceURL {
		return fmt.Errorf("SourceUrl = %v, want %q", importURL.SourceUrl, wantSourceURL)
	}
	return nil
}

func matchNoneSourceDetails(source mysqlsdk.CreateDbSystemSourceDetails) error {
	if _, ok := source.(mysqlsdk.CreateDbSystemSourceFromNoneDetails); !ok {
		return fmt.Errorf("CreateDbSystemDetails.Source type = %T, want %T", source, mysqlsdk.CreateDbSystemSourceFromNoneDetails{})
	}
	return nil
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

func newDbSystemSourceMutationManager() *DbSystemServiceManager {
	return &DbSystemServiceManager{
		client: generatedruntime.NewServiceClient[*mysqlv1beta1.DbSystem](generatedruntime.Config[*mysqlv1beta1.DbSystem]{
			Kind:    "DbSystem",
			SDKName: "DbSystem",
			Semantics: &generatedruntime.Semantics{
				Mutation: generatedruntime.MutationSemantics{
					ForceNew: []string{
						"source",
						"source.backupId",
						"source.dbSystemId",
						"source.recoveryPoint",
						"source.sourceType",
						"source.sourceUrl",
					},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &mysqlsdk.UpdateDbSystemRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return nil, fmt.Errorf("Update() should not be called when source changes")
				},
			},
		}),
	}
}

func newExistingDbSystemWithSource(current, desired mysqlv1beta1.DbSystemSource) *mysqlv1beta1.DbSystem {
	return &mysqlv1beta1.DbSystem{
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			ShapeName:     "MySQL.VM.Standard.E3.1.8GB",
			SubnetId:      "ocid1.subnet.oc1..example",
			Source:        desired,
		},
		Status: mysqlv1beta1.DbSystemStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: "ocid1.mysqldbsystem.oc1..existing",
			},
			Source: current,
		},
	}
}

func randomOCID(rand *rand.Rand, resource string) string {
	return fmt.Sprintf("ocid1.%s.oc1..%08x%08x", resource, rand.Uint32(), rand.Uint32())
}

func randomImportURL(rand *rand.Rand) string {
	return fmt.Sprintf(
		"https://objectstorage.example.com/n/tenant/b/bucket/o/%08x/%08x.manifest.json",
		rand.Uint32(),
		rand.Uint32(),
	)
}

func randomRecoveryPoint(rand *rand.Rand) string {
	return time.Unix(1_700_000_000+int64(rand.Int31n(31_536_000)), 0).UTC().Format(time.RFC3339)
}
