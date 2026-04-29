/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package connection

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeConnectionOCIClient struct {
	createFn      func(context.Context, databasemigrationsdk.CreateConnectionRequest) (databasemigrationsdk.CreateConnectionResponse, error)
	getFn         func(context.Context, databasemigrationsdk.GetConnectionRequest) (databasemigrationsdk.GetConnectionResponse, error)
	listFn        func(context.Context, databasemigrationsdk.ListConnectionsRequest) (databasemigrationsdk.ListConnectionsResponse, error)
	updateFn      func(context.Context, databasemigrationsdk.UpdateConnectionRequest) (databasemigrationsdk.UpdateConnectionResponse, error)
	deleteFn      func(context.Context, databasemigrationsdk.DeleteConnectionRequest) (databasemigrationsdk.DeleteConnectionResponse, error)
	workRequestFn func(context.Context, databasemigrationsdk.GetWorkRequestRequest) (databasemigrationsdk.GetWorkRequestResponse, error)
}

func (f *fakeConnectionOCIClient) CreateConnection(
	ctx context.Context,
	req databasemigrationsdk.CreateConnectionRequest,
) (databasemigrationsdk.CreateConnectionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return databasemigrationsdk.CreateConnectionResponse{}, nil
}

func (f *fakeConnectionOCIClient) GetConnection(
	ctx context.Context,
	req databasemigrationsdk.GetConnectionRequest,
) (databasemigrationsdk.GetConnectionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return databasemigrationsdk.GetConnectionResponse{}, nil
}

func (f *fakeConnectionOCIClient) ListConnections(
	ctx context.Context,
	req databasemigrationsdk.ListConnectionsRequest,
) (databasemigrationsdk.ListConnectionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return databasemigrationsdk.ListConnectionsResponse{}, nil
}

func (f *fakeConnectionOCIClient) UpdateConnection(
	ctx context.Context,
	req databasemigrationsdk.UpdateConnectionRequest,
) (databasemigrationsdk.UpdateConnectionResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return databasemigrationsdk.UpdateConnectionResponse{}, nil
}

func (f *fakeConnectionOCIClient) DeleteConnection(
	ctx context.Context,
	req databasemigrationsdk.DeleteConnectionRequest,
) (databasemigrationsdk.DeleteConnectionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return databasemigrationsdk.DeleteConnectionResponse{}, nil
}

func (f *fakeConnectionOCIClient) GetWorkRequest(
	ctx context.Context,
	req databasemigrationsdk.GetWorkRequestRequest,
) (databasemigrationsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return databasemigrationsdk.GetWorkRequestResponse{}, nil
}

type connectionRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func newTestConnectionClient(fake *fakeConnectionOCIClient) ConnectionServiceClient {
	return newConnectionServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func TestReviewedConnectionRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedConnectionRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedConnectionRuntimeSemantics() = nil")
	}

	if got.FormalService != "databasemigration" {
		t.Fatalf("FormalService = %q, want databasemigration", got.FormalService)
	}
	if got.FormalSlug != "connection" {
		t.Fatalf("FormalSlug = %q, want connection", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertConnectionStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertConnectionStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertConnectionStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertConnectionStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertConnectionStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertConnectionStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertConnectionStringSliceEqual(
		t,
		"List.MatchFields",
		got.List.MatchFields,
		[]string{"compartmentId", "displayName", "connectionType", "technologyType", "databaseId", "databaseName", "dbSystemId", "host", "port", "connectionString"},
	)
	assertConnectionStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "connectionType", "technologyType"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetConnection" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want workrequest-backed follow-up", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetConnection" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want workrequest-backed follow-up", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetConnection/ListConnections confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardConnectionExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeMySQLConnectionResource()
	resource.Spec.DisplayName = ""

	decision, err := guardConnectionExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardConnectionExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardConnectionExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "mysql-source"
	resource.Spec.ConnectionType = ""
	decision, err = guardConnectionExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardConnectionExistingBeforeCreate(empty connectionType) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardConnectionExistingBeforeCreate(empty connectionType) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.ConnectionType = "MYSQL"
	resource.Spec.TechnologyType = ""
	decision, err = guardConnectionExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardConnectionExistingBeforeCreate(empty technologyType) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardConnectionExistingBeforeCreate(empty technologyType) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.TechnologyType = "OCI_MYSQL"
	decision, err = guardConnectionExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardConnectionExistingBeforeCreate(valid identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardConnectionExistingBeforeCreate(valid identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildConnectionCreateDetailsUsesConcretePolymorphicBodies(t *testing.T) {
	t.Parallel()

	t.Run("MYSQL", func(t *testing.T) {
		t.Parallel()

		resource := makeMySQLConnectionResource()

		details, err := buildConnectionCreateDetails(context.Background(), resource, resource.Namespace)
		if err != nil {
			t.Fatalf("buildConnectionCreateDetails(MYSQL) error = %v", err)
		}

		createDetails, ok := details.(databasemigrationsdk.CreateMysqlConnectionDetails)
		if !ok {
			t.Fatalf("create body type = %T, want databasemigration.CreateMysqlConnectionDetails", details)
		}
		requireConnectionStringPtr(t, "databaseName", createDetails.DatabaseName, resource.Spec.DatabaseName)
		requireConnectionStringPtr(t, "host", createDetails.Host, resource.Spec.Host)
		requireConnectionIntPtr(t, "port", createDetails.Port, resource.Spec.Port)
		if createDetails.TechnologyType != databasemigrationsdk.MysqlConnectionTechnologyTypeOciMysql {
			t.Fatalf("technologyType = %q, want %q", createDetails.TechnologyType, databasemigrationsdk.MysqlConnectionTechnologyTypeOciMysql)
		}
		if createDetails.SecurityProtocol != databasemigrationsdk.MysqlConnectionSecurityProtocolTls {
			t.Fatalf("securityProtocol = %q, want %q", createDetails.SecurityProtocol, databasemigrationsdk.MysqlConnectionSecurityProtocolTls)
		}

		body := connectionSerializedRequestBody(t, databasemigrationsdk.CreateConnectionRequest{CreateConnectionDetails: details}, http.MethodPost, "/connections")
		for _, want := range []string{
			`"connectionType":"MYSQL"`,
			`"technologyType":"OCI_MYSQL"`,
			`"securityProtocol":"TLS"`,
			`"databaseName":"appdb"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("request body %s does not contain %s", body, want)
			}
		}
	})

	t.Run("ORACLE", func(t *testing.T) {
		t.Parallel()

		resource := makeOracleConnectionResource()

		details, err := buildConnectionCreateDetails(context.Background(), resource, resource.Namespace)
		if err != nil {
			t.Fatalf("buildConnectionCreateDetails(ORACLE) error = %v", err)
		}

		createDetails, ok := details.(databasemigrationsdk.CreateOracleConnectionDetails)
		if !ok {
			t.Fatalf("create body type = %T, want databasemigration.CreateOracleConnectionDetails", details)
		}
		requireConnectionStringPtr(t, "connectionString", createDetails.ConnectionString, resource.Spec.ConnectionString)
		requireConnectionStringPtr(t, "databaseId", createDetails.DatabaseId, resource.Spec.DatabaseId)
		if createDetails.TechnologyType != databasemigrationsdk.OracleConnectionTechnologyTypeOracleDatabase {
			t.Fatalf("technologyType = %q, want %q", createDetails.TechnologyType, databasemigrationsdk.OracleConnectionTechnologyTypeOracleDatabase)
		}

		body := connectionSerializedRequestBody(t, databasemigrationsdk.CreateConnectionRequest{CreateConnectionDetails: details}, http.MethodPost, "/connections")
		for _, want := range []string{
			`"connectionType":"ORACLE"`,
			`"technologyType":"ORACLE_DATABASE"`,
			`"connectionString":"dbhost.example.com:1521/appsvc"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("request body %s does not contain %s", body, want)
			}
		}
	})
}

func TestBuildConnectionCreateDetailsRejectsCrossTypeFields(t *testing.T) {
	t.Parallel()

	mysqlResource := makeMySQLConnectionResource()
	mysqlResource.Spec.ConnectionString = "dbhost.example.com:1521/not-allowed"
	if _, err := buildConnectionCreateDetails(context.Background(), mysqlResource, mysqlResource.Namespace); err == nil {
		t.Fatal("buildConnectionCreateDetails(MYSQL with oracle fields) error = nil, want rejection")
	} else if !strings.Contains(err.Error(), "spec.connectionString") {
		t.Fatalf("buildConnectionCreateDetails(MYSQL with oracle fields) error = %v, want spec.connectionString rejection", err)
	}

	oracleResource := makeOracleConnectionResource()
	oracleResource.Spec.Host = "mysql.example.com"
	if _, err := buildConnectionCreateDetails(context.Background(), oracleResource, oracleResource.Namespace); err == nil {
		t.Fatal("buildConnectionCreateDetails(ORACLE with mysql fields) error = nil, want rejection")
	} else if !strings.Contains(err.Error(), "spec.host") {
		t.Fatalf("buildConnectionCreateDetails(ORACLE with mysql fields) error = %v, want spec.host rejection", err)
	}
}

func TestBuildConnectionUpdateDetailsUsesConcretePolymorphicBody(t *testing.T) {
	t.Parallel()

	resource := makeOracleConnectionResource()
	resource.Spec.DisplayName = "oracle-source-updated"
	resource.Spec.ConnectionString = "dbhost.example.com:1521/updatedsvc"
	resource.Spec.Description = ""
	resource.Spec.FreeformTags = map[string]string{}

	details, updateNeeded, err := buildConnectionUpdateDetails(
		context.Background(),
		resource,
		resource.Namespace,
		databasemigrationsdk.GetConnectionResponse{
			Connection: makeOracleSDKConnection("ocid1.connection.oc1..oracle", makeOracleConnectionResource(), databasemigrationsdk.ConnectionLifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildConnectionUpdateDetails() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildConnectionUpdateDetails() updateNeeded = false, want true after mutable drift")
	}

	updateDetails, ok := details.(databasemigrationsdk.UpdateOracleConnectionDetails)
	if !ok {
		t.Fatalf("update body type = %T, want databasemigration.UpdateOracleConnectionDetails", details)
	}
	requireConnectionStringPtr(t, "displayName", updateDetails.DisplayName, resource.Spec.DisplayName)
	requireConnectionStringPtr(t, "connectionString", updateDetails.ConnectionString, resource.Spec.ConnectionString)
	if len(updateDetails.FreeformTags) != 0 {
		t.Fatalf("freeformTags = %#v, want empty map clear", updateDetails.FreeformTags)
	}

	body := connectionSerializedRequestBody(
		t,
		databasemigrationsdk.UpdateConnectionRequest{
			ConnectionId:            common.String("ocid1.connection.oc1..oracle"),
			UpdateConnectionDetails: details,
		},
		http.MethodPut,
		"/connections/ocid1.connection.oc1..oracle",
	)
	for _, want := range []string{
		`"connectionType":"ORACLE"`,
		`"displayName":"oracle-source-updated"`,
		`"connectionString":"dbhost.example.com:1521/updatedsvc"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestRecoverConnectionIDFromGeneratedWorkRequest(t *testing.T) {
	t.Parallel()

	workRequest := databasemigrationsdk.WorkRequest{
		Id:            common.String("wr-connection-create"),
		OperationType: databasemigrationsdk.OperationTypesCreateConnection,
		Status:        databasemigrationsdk.OperationStatusSucceeded,
		Resources: []databasemigrationsdk.WorkRequestResource{
			{
				ActionType: databasemigrationsdk.WorkRequestResourceActionTypeRelated,
				EntityType: common.String("migration"),
				Identifier: common.String("ocid1.migration.oc1..ignore"),
			},
			{
				ActionType: databasemigrationsdk.WorkRequestResourceActionTypeCreated,
				EntityType: common.String("Connection"),
				Identifier: common.String("ocid1.connection.oc1..created"),
			},
		},
	}

	id, err := recoverConnectionIDFromGeneratedWorkRequest(nil, workRequest, shared.OSOKAsyncPhaseCreate)
	if err != nil {
		t.Fatalf("recoverConnectionIDFromGeneratedWorkRequest() error = %v", err)
	}
	if id != "ocid1.connection.oc1..created" {
		t.Fatalf("recoverConnectionIDFromGeneratedWorkRequest() = %q, want %q", id, "ocid1.connection.oc1..created")
	}
}

func TestConnectionServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.connection.oc1..created"
		workRequestID = "wr-connection-create"
	)

	resource := makeMySQLConnectionResource()
	workRequests := map[string]databasemigrationsdk.WorkRequest{
		workRequestID: makeConnectionWorkRequest(
			workRequestID,
			databasemigrationsdk.OperationTypesCreateConnection,
			databasemigrationsdk.OperationStatusInProgress,
			databasemigrationsdk.WorkRequestResourceActionTypeInProgress,
			"",
		),
	}

	var createRequest databasemigrationsdk.CreateConnectionRequest
	var listRequest databasemigrationsdk.ListConnectionsRequest
	getCalls := 0

	client := newTestConnectionClient(&fakeConnectionOCIClient{
		listFn: func(_ context.Context, req databasemigrationsdk.ListConnectionsRequest) (databasemigrationsdk.ListConnectionsResponse, error) {
			listRequest = req
			return databasemigrationsdk.ListConnectionsResponse{}, nil
		},
		createFn: func(_ context.Context, req databasemigrationsdk.CreateConnectionRequest) (databasemigrationsdk.CreateConnectionResponse, error) {
			createRequest = req
			return databasemigrationsdk.CreateConnectionResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-connection"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req databasemigrationsdk.GetWorkRequestRequest) (databasemigrationsdk.GetWorkRequestResponse, error) {
			requireConnectionStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return databasemigrationsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req databasemigrationsdk.GetConnectionRequest) (databasemigrationsdk.GetConnectionResponse, error) {
			getCalls++
			requireConnectionStringPtr(t, "get connectionId", req.ConnectionId, createdID)
			return databasemigrationsdk.GetConnectionResponse{
				Connection: makeMySQLSDKConnection(createdID, resource, databasemigrationsdk.ConnectionLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	requireConnectionStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireConnectionStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if len(listRequest.ConnectionType) != 0 {
		t.Fatalf("list connectionType = %#v, want empty reviewed lookup filter", listRequest.ConnectionType)
	}
	if len(listRequest.TechnologyType) != 0 {
		t.Fatalf("list technologyType = %#v, want empty reviewed lookup filter", listRequest.TechnologyType)
	}
	if listRequest.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", listRequest.LifecycleState)
	}
	if getCalls != 0 {
		t.Fatalf("GetConnection() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireConnectionAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-connection" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-connection", got)
	}

	body := connectionSerializedRequestBody(t, createRequest, http.MethodPost, "/connections")
	for _, want := range []string{
		`"connectionType":"MYSQL"`,
		`"technologyType":"OCI_MYSQL"`,
		`"databaseName":"appdb"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("create request body %s does not contain %s", body, want)
		}
	}

	workRequests[workRequestID] = makeConnectionWorkRequest(
		workRequestID,
		databasemigrationsdk.OperationTypesCreateConnection,
		databasemigrationsdk.OperationStatusSucceeded,
		databasemigrationsdk.WorkRequestResourceActionTypeCreated,
		createdID,
	)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetConnection() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(databasemigrationsdk.ConnectionLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, databasemigrationsdk.ConnectionLifecycleStateActive)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after successful reconciliation", resource.Status.OsokStatus.Async.Current)
	}
}

func makeMySQLConnectionResource() *databasemigrationv1beta1.Connection {
	return &databasemigrationv1beta1.Connection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-connection",
			Namespace: "default",
		},
		Spec: databasemigrationv1beta1.ConnectionSpec{
			DisplayName:         "mysql-source",
			CompartmentId:       "ocid1.compartment.oc1..mysql",
			VaultId:             "ocid1.vault.oc1..mysql",
			KeyId:               "ocid1.key.oc1..mysql",
			Username:            "migration-user",
			Password:            "SuperSecret123",
			ReplicationUsername: "replication-user",
			ReplicationPassword: "ReplicationSecret123",
			ConnectionType:      "MYSQL",
			TechnologyType:      "OCI_MYSQL",
			SecurityProtocol:    "TLS",
			SslMode:             "REQUIRED",
			DatabaseName:        "appdb",
			Host:                "mysql.example.com",
			Port:                3306,
			SubnetId:            "ocid1.subnet.oc1..mysql",
			NsgIds:              []string{"ocid1.nsg.oc1..mysql"},
			Description:         "mysql source connection",
			FreeformTags:        map[string]string{"env": "test"},
			DefinedTags:         map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			AdditionalAttributes: []databasemigrationv1beta1.ConnectionAdditionalAttribute{
				{Name: "tlsVersion", Value: "TLSv1.2"},
			},
		},
	}
}

func makeOracleConnectionResource() *databasemigrationv1beta1.Connection {
	return &databasemigrationv1beta1.Connection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oracle-connection",
			Namespace: "default",
		},
		Spec: databasemigrationv1beta1.ConnectionSpec{
			DisplayName:         "oracle-source",
			CompartmentId:       "ocid1.compartment.oc1..oracle",
			VaultId:             "ocid1.vault.oc1..oracle",
			KeyId:               "ocid1.key.oc1..oracle",
			Username:            "migration-user",
			Password:            "OracleSecret123",
			ReplicationUsername: "replication-user",
			ReplicationPassword: "OracleReplicationSecret123",
			ConnectionType:      "ORACLE",
			TechnologyType:      "ORACLE_DATABASE",
			ConnectionString:    "dbhost.example.com:1521/appsvc",
			DatabaseId:          "ocid1.database.oc1..oracle",
			SshHost:             "bastion.example.com",
			SshUser:             "opc",
			SshSudoLocation:     "/usr/bin/sudo",
			Description:         "oracle source connection",
			FreeformTags:        map[string]string{"env": "test"},
			DefinedTags:         map[string]shared.MapValue{"Operations": {"CostCenter": "84"}},
		},
	}
}

func makeMySQLSDKConnection(
	id string,
	resource *databasemigrationv1beta1.Connection,
	state databasemigrationsdk.ConnectionLifecycleStateEnum,
) databasemigrationsdk.MysqlConnection {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	return databasemigrationsdk.MysqlConnection{
		Id:                  common.String(id),
		DisplayName:         common.String(resource.Spec.DisplayName),
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		TimeCreated:         now,
		TimeUpdated:         now,
		Username:            common.String(resource.Spec.Username),
		Description:         common.String(resource.Spec.Description),
		FreeformTags:        map[string]string{"env": "test"},
		DefinedTags:         map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		VaultId:             common.String(resource.Spec.VaultId),
		KeyId:               common.String(resource.Spec.KeyId),
		SubnetId:            common.String(resource.Spec.SubnetId),
		NsgIds:              append([]string(nil), resource.Spec.NsgIds...),
		Password:            common.String(resource.Spec.Password),
		ReplicationUsername: common.String(resource.Spec.ReplicationUsername),
		ReplicationPassword: common.String(resource.Spec.ReplicationPassword),
		Host:                common.String(resource.Spec.Host),
		Port:                common.Int(resource.Spec.Port),
		DatabaseName:        common.String(resource.Spec.DatabaseName),
		AdditionalAttributes: []databasemigrationsdk.NameValuePair{
			{Name: common.String("tlsVersion"), Value: common.String("TLSv1.2")},
		},
		TechnologyType:   databasemigrationsdk.MysqlConnectionTechnologyTypeOciMysql,
		SecurityProtocol: databasemigrationsdk.MysqlConnectionSecurityProtocolTls,
		SslMode:          databasemigrationsdk.MysqlConnectionSslModeRequired,
		LifecycleState:   state,
	}
}

func makeOracleSDKConnection(
	id string,
	resource *databasemigrationv1beta1.Connection,
	state databasemigrationsdk.ConnectionLifecycleStateEnum,
) databasemigrationsdk.OracleConnection {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	return databasemigrationsdk.OracleConnection{
		Id:                  common.String(id),
		DisplayName:         common.String(resource.Spec.DisplayName),
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		TimeCreated:         now,
		TimeUpdated:         now,
		Username:            common.String(resource.Spec.Username),
		Description:         common.String(resource.Spec.Description),
		FreeformTags:        map[string]string{"env": "test"},
		DefinedTags:         map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}},
		VaultId:             common.String(resource.Spec.VaultId),
		KeyId:               common.String(resource.Spec.KeyId),
		Password:            common.String(resource.Spec.Password),
		ReplicationUsername: common.String(resource.Spec.ReplicationUsername),
		ReplicationPassword: common.String(resource.Spec.ReplicationPassword),
		ConnectionString:    common.String(resource.Spec.ConnectionString),
		DatabaseId:          common.String(resource.Spec.DatabaseId),
		SshHost:             common.String(resource.Spec.SshHost),
		SshUser:             common.String(resource.Spec.SshUser),
		SshSudoLocation:     common.String(resource.Spec.SshSudoLocation),
		TechnologyType:      databasemigrationsdk.OracleConnectionTechnologyTypeOracleDatabase,
		LifecycleState:      state,
	}
}

func makeConnectionWorkRequest(
	id string,
	operation databasemigrationsdk.OperationTypesEnum,
	status databasemigrationsdk.OperationStatusEnum,
	action databasemigrationsdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) databasemigrationsdk.WorkRequest {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	resources := []databasemigrationsdk.WorkRequestResource{
		{
			ActionType: action,
			EntityType: common.String("Connection"),
			Identifier: common.String(resourceID),
		},
	}
	return databasemigrationsdk.WorkRequest{
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..workrequest"),
		OperationType:   operation,
		Status:          status,
		Resources:       resources,
		PercentComplete: common.Float32(25),
		TimeAccepted:    now,
	}
}

func connectionSerializedRequestBody(
	t *testing.T,
	request connectionRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request body) error = %v", err)
	}
	return string(body)
}

func requireConnectionStringPtr(t *testing.T, field string, actual *string, want string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *actual != want {
		t.Fatalf("%s = %q, want %q", field, *actual, want)
	}
}

func requireConnectionIntPtr(t *testing.T, field string, actual *int, want int) {
	t.Helper()
	if actual == nil {
		t.Fatalf("%s = nil, want %d", field, want)
	}
	if *actual != want {
		t.Fatalf("%s = %d, want %d", field, *actual, want)
	}
}

func requireConnectionAsyncCurrent(
	t *testing.T,
	resource *databasemigrationv1beta1.Connection,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func assertConnectionStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}
