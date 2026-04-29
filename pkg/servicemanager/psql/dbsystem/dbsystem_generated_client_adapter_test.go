/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	psqlsdk "github.com/oracle/oci-go-sdk/v65/psql"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestManualDbSystemServiceClientCreatesWhenNoExistingDbSystemMatches(t *testing.T) {
	t.Parallel()

	var createRequest psqlsdk.CreateDbSystemRequest

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			listDbSystems: func(_ context.Context, request psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
				if got := stringValue(request.CompartmentId); got != "ocid1.compartment.oc1..example" {
					t.Fatalf("ListDbSystemsRequest.CompartmentId = %q, want ocid1.compartment.oc1..example", got)
				}
				if got := stringValue(request.DisplayName); got != "sample-db" {
					t.Fatalf("ListDbSystemsRequest.DisplayName = %q, want sample-db", got)
				}
				return psqlsdk.ListDbSystemsResponse{
					DbSystemCollection: psqlsdk.DbSystemCollection{},
				}, nil
			},
			createDbSystem: func(_ context.Context, request psqlsdk.CreateDbSystemRequest) (psqlsdk.CreateDbSystemResponse, error) {
				createRequest = request
				return psqlsdk.CreateDbSystemResponse{
					OpcRequestId: common.String("opc-create-1"),
					DbSystem:     sdkDbSystem("ocid1.dbsystem.oc1..create", "sample-db", psqlsdk.DbSystemLifecycleStateCreating),
				}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeueing create", response)
	}
	if got := stringValue(createRequest.CreateDbSystemDetails.DisplayName); got != "sample-db" {
		t.Fatalf("CreateDbSystemDetails.DisplayName = %q, want sample-db", got)
	}
	if got := stringValue(createRequest.CreateDbSystemDetails.CompartmentId); got != "ocid1.compartment.oc1..example" {
		t.Fatalf("CreateDbSystemDetails.CompartmentId = %q, want ocid1.compartment.oc1..example", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.dbsystem.oc1..create" {
		t.Fatalf("status.status.ocid = %q, want created OCID", got)
	}
	if createRequest.CreateDbSystemDetails.Credentials == nil || stringValue(createRequest.CreateDbSystemDetails.Credentials.Username) != "postgres" {
		t.Fatalf("CreateDbSystemDetails.Credentials = %#v, want inline postgres credentials", createRequest.CreateDbSystemDetails.Credentials)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Provisioning {
		t.Fatalf("status conditions = %#v, want trailing Provisioning condition", resource.Status.OsokStatus.Conditions)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.AdminUsernameSource != (shared.UsernameSource{}) {
		t.Fatalf("status.adminUsernameSource = %#v, want zero value", resource.Status.AdminUsernameSource)
	}
	if resource.Status.AdminPasswordSource != (shared.PasswordSource{}) {
		t.Fatalf("status.adminPasswordSource = %#v, want zero value", resource.Status.AdminPasswordSource)
	}
}

func TestManualDbSystemServiceClientBindsExistingDbSystemBeforeCreate(t *testing.T) {
	t.Parallel()

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			listDbSystems: func(_ context.Context, _ psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
				return psqlsdk.ListDbSystemsResponse{
					DbSystemCollection: psqlsdk.DbSystemCollection{
						Items: []psqlsdk.DbSystemSummary{
							sdkDbSystemSummary("ocid1.dbsystem.oc1..bound", "sample-db", psqlsdk.DbSystemLifecycleStateActive),
						},
					},
				}, nil
			},
			getDbSystem: func(_ context.Context, request psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				if got := stringValue(request.DbSystemId); got != "ocid1.dbsystem.oc1..bound" {
					t.Fatalf("GetDbSystemRequest.DbSystemId = %q, want ocid1.dbsystem.oc1..bound", got)
				}
				return psqlsdk.GetDbSystemResponse{
					DbSystem: sdkDbSystem("ocid1.dbsystem.oc1..bound", "sample-db", psqlsdk.DbSystemLifecycleStateActive),
				}, nil
			},
			createDbSystem: func(_ context.Context, _ psqlsdk.CreateDbSystemRequest) (psqlsdk.CreateDbSystemResponse, error) {
				t.Fatal("CreateDbSystem() should not be called when bind lookup finds an existing DbSystem")
				return psqlsdk.CreateDbSystemResponse{}, nil
			},
			updateDbSystem: func(_ context.Context, _ psqlsdk.UpdateDbSystemRequest) (psqlsdk.UpdateDbSystemResponse, error) {
				t.Fatal("UpdateDbSystem() should not be called when no mutable drift exists")
				return psqlsdk.UpdateDbSystemResponse{}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.dbsystem.oc1..bound" {
		t.Fatalf("status.status.ocid = %q, want bound OCID", got)
	}
}

func TestManualDbSystemServiceClientCreatesUsingAdminSecrets(t *testing.T) {
	t.Parallel()

	var createRequest psqlsdk.CreateDbSystemRequest

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			listDbSystems: func(_ context.Context, _ psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
				return psqlsdk.ListDbSystemsResponse{
					DbSystemCollection: psqlsdk.DbSystemCollection{},
				}, nil
			},
			createDbSystem: func(_ context.Context, request psqlsdk.CreateDbSystemRequest) (psqlsdk.CreateDbSystemResponse, error) {
				createRequest = request
				return psqlsdk.CreateDbSystemResponse{
					DbSystem: sdkDbSystem("ocid1.dbsystem.oc1..create", "sample-db", psqlsdk.DbSystemLifecycleStateCreating),
				}, nil
			},
		},
		credentialClient: fakeDbSystemCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("dbadmin"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()
	resource.Namespace = "default"
	resource.Spec.Credentials = psqlv1beta1.DbSystemCredentials{}
	resource.Spec.AdminUsername = usernameSecretSource("admin-secret")
	resource.Spec.AdminPassword = passwordSecretSource("admin-secret")

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeueing create", response)
	}
	if createRequest.CreateDbSystemDetails.Credentials == nil {
		t.Fatal("CreateDbSystemDetails.Credentials = nil, want secret-backed credentials")
	}
	if got := stringValue(createRequest.CreateDbSystemDetails.Credentials.Username); got != "dbadmin" {
		t.Fatalf("CreateDbSystemDetails.Credentials.Username = %q, want dbadmin", got)
	}
	passwordDetails, ok := createRequest.CreateDbSystemDetails.Credentials.PasswordDetails.(psqlsdk.PlainTextPasswordDetails)
	if !ok {
		t.Fatalf("CreateDbSystemDetails.Credentials.PasswordDetails type = %T, want PlainTextPasswordDetails", createRequest.CreateDbSystemDetails.Credentials.PasswordDetails)
	}
	if got := stringValue(passwordDetails.Password); got != "ChangeMe123!!" {
		t.Fatalf("CreateDbSystemDetails.Credentials.PasswordDetails.Password = %q, want ChangeMe123!!", got)
	}
	if got := resource.Status.AdminUsernameSource.Secret.SecretName; got != "admin-secret" {
		t.Fatalf("status.adminUsernameSource.secret.secretName = %q, want admin-secret", got)
	}
	if got := resource.Status.AdminPasswordSource.Secret.SecretName; got != "admin-secret" {
		t.Fatalf("status.adminPasswordSource.secret.secretName = %q, want admin-secret", got)
	}
}

func TestManualDbSystemServiceClientRejectsImmutableShapeDrift(t *testing.T) {
	t.Parallel()

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			listDbSystems: func(_ context.Context, _ psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
				return psqlsdk.ListDbSystemsResponse{
					DbSystemCollection: psqlsdk.DbSystemCollection{
						Items: []psqlsdk.DbSystemSummary{
							sdkDbSystemSummary("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive),
						},
					},
				}, nil
			},
			getDbSystem: func(_ context.Context, _ psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				current := sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive)
				current.Shape = common.String("VM.Standard3.Flex")
				return psqlsdk.GetDbSystemResponse{DbSystem: current}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "shape") {
		t.Fatalf("CreateOrUpdate() error = %v, want immutable shape drift failure", err)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Failed {
		t.Fatalf("status conditions = %#v, want trailing Failed condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestValidateImmutableDriftAcceptsPostCreateReadbackNormalization(t *testing.T) {
	t.Parallel()

	resource := testDbSystemResource()
	resource.Spec.Shape = "PostgreSQL.VM.Standard.E5.Flex"
	observed := psqlv1beta1.DbSystemStatus{
		CompartmentId:           resource.Spec.CompartmentId,
		DbVersion:               "14.17",
		Shape:                   "VM.Standard.E5.Flex",
		InstanceOcpuCount:       2,
		InstanceMemorySizeInGBs: 16,
		InstanceCount:           1,
		StorageDetails: psqlv1beta1.DbSystemStorageDetails{
			SystemType:          resource.Spec.StorageDetails.SystemType,
			IsRegionallyDurable: resource.Spec.StorageDetails.IsRegionallyDurable,
			Iops:                resource.Spec.StorageDetails.Iops,
		},
		NetworkDetails: psqlv1beta1.DbSystemNetworkDetails{
			SubnetId:                   resource.Spec.NetworkDetails.SubnetId,
			PrimaryDbEndpointPrivateIp: "10.0.10.110",
		},
		AdminUsername: resource.Spec.Credentials.Username,
	}

	if err := validateImmutableDrift(resource.Spec, observed); err != nil {
		t.Fatalf("validateImmutableDrift() error = %v, want nil", err)
	}
}

func TestValidateImmutableDriftRejectsDbVersionMajorChange(t *testing.T) {
	t.Parallel()

	resource := testDbSystemResource()
	observed := psqlv1beta1.DbSystemStatus{
		CompartmentId: resource.Spec.CompartmentId,
		DbVersion:     "15.1",
		Shape:         resource.Spec.Shape,
		StorageDetails: psqlv1beta1.DbSystemStorageDetails{
			SystemType:          resource.Spec.StorageDetails.SystemType,
			IsRegionallyDurable: resource.Spec.StorageDetails.IsRegionallyDurable,
		},
		NetworkDetails: resource.Spec.NetworkDetails,
		AdminUsername:  resource.Spec.Credentials.Username,
	}

	err := validateImmutableDrift(resource.Spec, observed)
	if err == nil || !strings.Contains(err.Error(), "dbVersion") {
		t.Fatalf("validateImmutableDrift() error = %v, want dbVersion drift", err)
	}
}

func TestManualDbSystemServiceClientUpdatesMutableDescriptionAndRequeues(t *testing.T) {
	t.Parallel()

	var (
		updateRequest psqlsdk.UpdateDbSystemRequest
		getCalls      int
	)

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			listDbSystems: func(_ context.Context, _ psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
				return psqlsdk.ListDbSystemsResponse{
					DbSystemCollection: psqlsdk.DbSystemCollection{
						Items: []psqlsdk.DbSystemSummary{
							sdkDbSystemSummary("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive),
						},
					},
				}, nil
			},
			getDbSystem: func(_ context.Context, _ psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				getCalls++
				current := sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive)
				if getCalls == 1 {
					current.Description = common.String("old description")
					return psqlsdk.GetDbSystemResponse{DbSystem: current}, nil
				}
				current.Description = common.String("new description")
				return psqlsdk.GetDbSystemResponse{DbSystem: current}, nil
			},
			updateDbSystem: func(_ context.Context, request psqlsdk.UpdateDbSystemRequest) (psqlsdk.UpdateDbSystemResponse, error) {
				updateRequest = request
				return psqlsdk.UpdateDbSystemResponse{OpcRequestId: common.String("opc-update-1")}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()
	resource.Spec.Description = "new description"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeueing update", response)
	}
	if got := stringValue(updateRequest.DbSystemId); got != "ocid1.dbsystem.oc1..existing" {
		t.Fatalf("UpdateDbSystemRequest.DbSystemId = %q, want ocid1.dbsystem.oc1..existing", got)
	}
	if got := stringValue(updateRequest.UpdateDbSystemDetails.Description); got != "new description" {
		t.Fatalf("UpdateDbSystemDetails.Description = %q, want new description", got)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Updating {
		t.Fatalf("status conditions = %#v, want trailing Updating condition", resource.Status.OsokStatus.Conditions)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestManualDbSystemServiceClientDeleteRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	var (
		deleteRequest psqlsdk.DeleteDbSystemRequest
		getCalls      int
	)

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			getDbSystem: func(_ context.Context, request psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				getCalls++
				if got := stringValue(request.DbSystemId); got != "ocid1.dbsystem.oc1..existing" {
					t.Fatalf("GetDbSystemRequest.DbSystemId = %q, want ocid1.dbsystem.oc1..existing", got)
				}
				lifecycle := psqlsdk.DbSystemLifecycleStateActive
				if getCalls > 1 {
					lifecycle = psqlsdk.DbSystemLifecycleStateDeleting
				}
				return psqlsdk.GetDbSystemResponse{
					DbSystem: sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", lifecycle),
				}, nil
			},
			deleteDbSystem: func(_ context.Context, request psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error) {
				deleteRequest = request
				return psqlsdk.DeleteDbSystemResponse{OpcRequestId: common.String("opc-delete-1")}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()
	resource.Status.OsokStatus.Ocid = "ocid1.dbsystem.oc1..existing"
	resource.Status.Id = "ocid1.dbsystem.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while OCI delete is still in progress")
	}
	if got := stringValue(deleteRequest.DbSystemId); got != "ocid1.dbsystem.oc1..existing" {
		t.Fatalf("DeleteDbSystemRequest.DbSystemId = %q, want ocid1.dbsystem.oc1..existing", got)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func TestManualDbSystemServiceClientDeleteTreatsConfirmedNotFoundAsDeleted(t *testing.T) {
	t.Parallel()

	var getCalls int

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			getDbSystem: func(_ context.Context, _ psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				getCalls++
				if getCalls == 1 {
					return psqlsdk.GetDbSystemResponse{
						DbSystem: sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive),
					}, nil
				}
				return psqlsdk.GetDbSystemResponse{}, fakeDbSystemServiceError{
					code:       "NotAuthorizedOrNotFound",
					message:    "dbsystem not found",
					statusCode: 404,
				}
			},
			deleteDbSystem: func(_ context.Context, _ psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error) {
				return psqlsdk.DeleteDbSystemResponse{}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()
	resource.Status.OsokStatus.Ocid = "ocid1.dbsystem.oc1..existing"
	resource.Status.Id = "ocid1.dbsystem.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success when GetDbSystem confirms NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
}

func TestManualDbSystemServiceClientDeleteTreatsDeletedLifecycleAsDeleted(t *testing.T) {
	t.Parallel()

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			getDbSystem: func(_ context.Context, _ psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				return psqlsdk.GetDbSystemResponse{
					DbSystem: sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateDeleted),
				}, nil
			},
			deleteDbSystem: func(_ context.Context, _ psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error) {
				t.Fatal("DeleteDbSystem() should not be called when OCI already reports DELETED")
				return psqlsdk.DeleteDbSystemResponse{}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()
	resource.Status.OsokStatus.Ocid = "ocid1.dbsystem.oc1..existing"
	resource.Status.Id = "ocid1.dbsystem.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success when OCI lifecycle is already DELETED")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after deleted lifecycle is observed")
	}
}

func TestManualDbSystemServiceClientDeleteSkipsRedundantDeleteWhileDeleting(t *testing.T) {
	t.Parallel()

	client := manualDbSystemServiceClient{
		sdk: fakeDbSystemOCIClient{
			getDbSystem: func(_ context.Context, _ psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
				return psqlsdk.GetDbSystemResponse{
					DbSystem: sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateDeleting),
				}, nil
			},
			deleteDbSystem: func(_ context.Context, _ psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error) {
				t.Fatal("DeleteDbSystem() should not be called when OCI is already deleting the DbSystem")
				return psqlsdk.DeleteDbSystemResponse{}, nil
			},
		},
		log: discardDbSystemLogger(),
	}

	resource := testDbSystemResource()
	resource.Status.OsokStatus.Ocid = "ocid1.dbsystem.oc1..existing"
	resource.Status.Id = "ocid1.dbsystem.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while OCI lifecycle is DELETING")
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestDbSystemRuntimeHooksWrapGeneratedClientWithManualAdapter(t *testing.T) {
	t.Parallel()

	hooks := DbSystemRuntimeHooks{Semantics: newDbSystemRuntimeSemantics()}
	manager := &DbSystemServiceManager{
		Provider: common.NewRawConfigurationProvider("", "", "", "", "", nil),
		Log:      discardDbSystemLogger(),
	}

	applyDbSystemRuntimeHooks(manager, &hooks)
	if got, want := hooks.Semantics.Lifecycle.UpdatingStates, []string{"UPDATING"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Semantics.Lifecycle.UpdatingStates = %#v, want %#v", got, want)
	}
	if got, want := hooks.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE", "NEEDS_ATTENTION"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Semantics.Lifecycle.ActiveStates = %#v, want %#v", got, want)
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}

	wrapped := hooks.WrapGeneratedClient[0](manualDbSystemServiceClient{})
	if _, ok := wrapped.(manualDbSystemServiceClient); !ok {
		t.Fatalf("wrapped client type = %T, want manualDbSystemServiceClient", wrapped)
	}
}

func TestClassifyDbSystemLifecycleMapsFormalStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		state          psqlsdk.DbSystemLifecycleStateEnum
		fallback       shared.OSOKConditionType
		wantCondition  shared.OSOKConditionType
		wantRequeue    bool
		wantSuccessful bool
	}{
		{name: "creating", state: psqlsdk.DbSystemLifecycleStateCreating, fallback: shared.Active, wantCondition: shared.Provisioning, wantRequeue: true, wantSuccessful: true},
		{name: "updating", state: psqlsdk.DbSystemLifecycleStateUpdating, fallback: shared.Active, wantCondition: shared.Updating, wantRequeue: true, wantSuccessful: true},
		{name: "deleting", state: psqlsdk.DbSystemLifecycleStateDeleting, fallback: shared.Active, wantCondition: shared.Terminating, wantRequeue: true, wantSuccessful: true},
		{name: "active", state: psqlsdk.DbSystemLifecycleStateActive, fallback: shared.Active, wantCondition: shared.Active, wantRequeue: false, wantSuccessful: true},
		{name: "inactive", state: psqlsdk.DbSystemLifecycleStateInactive, fallback: shared.Active, wantCondition: shared.Active, wantRequeue: false, wantSuccessful: true},
		{name: "needs attention", state: psqlsdk.DbSystemLifecycleStateNeedsAttention, fallback: shared.Active, wantCondition: shared.Active, wantRequeue: false, wantSuccessful: true},
		{name: "active update follow-up", state: psqlsdk.DbSystemLifecycleStateActive, fallback: shared.Updating, wantCondition: shared.Updating, wantRequeue: true, wantSuccessful: true},
		{name: "failed", state: psqlsdk.DbSystemLifecycleStateFailed, fallback: shared.Active, wantCondition: shared.Failed, wantRequeue: false, wantSuccessful: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			condition, shouldRequeue, _ := classifyDbSystemLifecycle(
				sdkDbSystem("ocid1.dbsystem.oc1..lifecycle", "sample-db", test.state),
				test.fallback,
			)
			if condition != test.wantCondition {
				t.Fatalf("condition = %s, want %s", condition, test.wantCondition)
			}
			if shouldRequeue != test.wantRequeue {
				t.Fatalf("shouldRequeue = %t, want %t", shouldRequeue, test.wantRequeue)
			}
			if successful := condition != shared.Failed; successful != test.wantSuccessful {
				t.Fatalf("successful = %t, want %t", successful, test.wantSuccessful)
			}
		})
	}
}

func TestDropDbSystemStaleConditionsOnSuccessfulProjection(t *testing.T) {
	t.Parallel()

	status := shared.OSOKStatus{
		Conditions: []shared.OSOKCondition{
			{Type: shared.Provisioning},
			{Type: shared.Active},
			{Type: shared.Failed},
			{Type: shared.Updating},
			{Type: shared.Terminating},
			{Type: shared.Failed},
		},
	}

	got := dropDbSystemStaleConditions(status, shared.Active)
	if len(got.Conditions) != 1 {
		t.Fatalf("conditions = %#v, want only active condition", got.Conditions)
	}
	if got.Conditions[0].Type != shared.Active {
		t.Fatalf("conditions = %#v, want Active", got.Conditions)
	}
}

func TestDropDbSystemStaleConditionsPreservesCurrentTransientCondition(t *testing.T) {
	t.Parallel()

	status := shared.OSOKStatus{
		Conditions: []shared.OSOKCondition{
			{Type: shared.Active},
			{Type: shared.Failed},
			{Type: shared.Updating},
		},
	}

	got := dropDbSystemStaleConditions(status, shared.Updating)
	if len(got.Conditions) != 2 {
		t.Fatalf("conditions = %#v, want active and updating conditions", got.Conditions)
	}
	if got.Conditions[0].Type != shared.Active || got.Conditions[1].Type != shared.Updating {
		t.Fatalf("conditions = %#v, want Active then Updating", got.Conditions)
	}
}

type fakeDbSystemOCIClient struct {
	createDbSystem func(context.Context, psqlsdk.CreateDbSystemRequest) (psqlsdk.CreateDbSystemResponse, error)
	getDbSystem    func(context.Context, psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error)
	listDbSystems  func(context.Context, psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error)
	updateDbSystem func(context.Context, psqlsdk.UpdateDbSystemRequest) (psqlsdk.UpdateDbSystemResponse, error)
	deleteDbSystem func(context.Context, psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error)
}

func (f fakeDbSystemOCIClient) CreateDbSystem(ctx context.Context, request psqlsdk.CreateDbSystemRequest) (psqlsdk.CreateDbSystemResponse, error) {
	if f.createDbSystem == nil {
		return psqlsdk.CreateDbSystemResponse{}, errors.New("unexpected CreateDbSystem call")
	}
	return f.createDbSystem(ctx, request)
}

func (f fakeDbSystemOCIClient) GetDbSystem(ctx context.Context, request psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
	if f.getDbSystem == nil {
		return psqlsdk.GetDbSystemResponse{}, errors.New("unexpected GetDbSystem call")
	}
	return f.getDbSystem(ctx, request)
}

func (f fakeDbSystemOCIClient) ListDbSystems(ctx context.Context, request psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
	if f.listDbSystems == nil {
		return psqlsdk.ListDbSystemsResponse{}, errors.New("unexpected ListDbSystems call")
	}
	return f.listDbSystems(ctx, request)
}

func (f fakeDbSystemOCIClient) UpdateDbSystem(ctx context.Context, request psqlsdk.UpdateDbSystemRequest) (psqlsdk.UpdateDbSystemResponse, error) {
	if f.updateDbSystem == nil {
		return psqlsdk.UpdateDbSystemResponse{}, errors.New("unexpected UpdateDbSystem call")
	}
	return f.updateDbSystem(ctx, request)
}

func (f fakeDbSystemOCIClient) DeleteDbSystem(ctx context.Context, request psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error) {
	if f.deleteDbSystem == nil {
		return psqlsdk.DeleteDbSystemResponse{}, errors.New("unexpected DeleteDbSystem call")
	}
	return f.deleteDbSystem(ctx, request)
}

type fakeDbSystemServiceError struct {
	code       string
	message    string
	statusCode int
	opcID      string
}

func (f fakeDbSystemServiceError) Error() string          { return f.message }
func (f fakeDbSystemServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeDbSystemServiceError) GetMessage() string     { return f.message }
func (f fakeDbSystemServiceError) GetCode() string        { return f.code }
func (f fakeDbSystemServiceError) GetOpcRequestID() string {
	return f.opcID
}

type fakeDbSystemCredentialClient struct {
	secrets  map[string]map[string][]byte
	getCalls []string
}

func (f fakeDbSystemCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return false, nil
}

func (f fakeDbSystemCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return false, nil
}

func (f fakeDbSystemCredentialClient) GetSecret(_ context.Context, name, _ string) (map[string][]byte, error) {
	if secret, ok := f.secrets[name]; ok {
		return secret, nil
	}
	return nil, fmt.Errorf("secret %q not found", name)
}

func (f fakeDbSystemCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return false, nil
}

func testDbSystemResource() *psqlv1beta1.DbSystem {
	return &psqlv1beta1.DbSystem{
		Spec: psqlv1beta1.DbSystemSpec{
			DisplayName:   "sample-db",
			CompartmentId: "ocid1.compartment.oc1..example",
			DbVersion:     "14",
			Shape:         "VM.Standard.E4.Flex",
			StorageDetails: psqlv1beta1.DbSystemStorageDetails{
				SystemType:          "OCI_OPTIMIZED_STORAGE",
				IsRegionallyDurable: true,
				Iops:                10,
			},
			NetworkDetails: psqlv1beta1.DbSystemNetworkDetails{
				SubnetId: "ocid1.subnet.oc1..example",
			},
			Credentials: psqlv1beta1.DbSystemCredentials{
				Username: "postgres",
				PasswordDetails: psqlv1beta1.DbSystemCredentialsPasswordDetails{
					PasswordType: "PLAIN_TEXT",
					Password:     "ChangeMe123!!",
				},
			},
		},
	}
}

func sdkDbSystem(id string, displayName string, lifecycle psqlsdk.DbSystemLifecycleStateEnum) psqlsdk.DbSystem {
	return psqlsdk.DbSystem{
		Id:                      common.String(id),
		DisplayName:             common.String(displayName),
		CompartmentId:           common.String("ocid1.compartment.oc1..example"),
		LifecycleState:          lifecycle,
		SystemType:              psqlsdk.DbSystemSystemTypeEnum("OCI_OPTIMIZED_STORAGE"),
		DbVersion:               common.String("14"),
		Shape:                   common.String("VM.Standard.E4.Flex"),
		InstanceOcpuCount:       common.Int(2),
		InstanceMemorySizeInGBs: common.Int(16),
		InstanceCount:           common.Int(1),
		StorageDetails: psqlsdk.OciOptimizedStorageDetails{
			IsRegionallyDurable: common.Bool(true),
			Iops:                common.Int64(10),
		},
		NetworkDetails: &psqlsdk.NetworkDetails{
			SubnetId: common.String("ocid1.subnet.oc1..example"),
		},
		AdminUsername: common.String("postgres"),
		ConfigId:      common.String("ocid1.configuration.oc1..example"),
		FreeformTags:  map[string]string{},
		DefinedTags:   map[string]map[string]interface{}{},
		SystemTags:    map[string]map[string]interface{}{},
		Instances:     []psqlsdk.DbInstance{},
	}
}

func sdkDbSystemSummary(id string, displayName string, lifecycle psqlsdk.DbSystemLifecycleStateEnum) psqlsdk.DbSystemSummary {
	return psqlsdk.DbSystemSummary{
		Id:                      common.String(id),
		DisplayName:             common.String(displayName),
		CompartmentId:           common.String("ocid1.compartment.oc1..example"),
		LifecycleState:          lifecycle,
		SystemType:              psqlsdk.DbSystemSystemTypeEnum("OCI_OPTIMIZED_STORAGE"),
		InstanceCount:           common.Int(1),
		InstanceOcpuCount:       common.Int(2),
		InstanceMemorySizeInGBs: common.Int(16),
		DbVersion:               common.String("14"),
		FreeformTags:            map[string]string{},
		DefinedTags:             map[string]map[string]interface{}{},
	}
}

func discardDbSystemLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: logr.Discard()}
}

func usernameSecretSource(name string) shared.UsernameSource {
	return shared.UsernameSource{Secret: shared.SecretSource{SecretName: name}}
}

func passwordSecretSource(name string) shared.PasswordSource {
	return shared.PasswordSource{Secret: shared.SecretSource{SecretName: name}}
}
