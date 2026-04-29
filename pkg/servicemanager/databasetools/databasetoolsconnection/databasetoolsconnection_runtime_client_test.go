/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package databasetoolsconnection

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
	databasetoolsv1beta1 "github.com/oracle/oci-service-operator/api/databasetools/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDatabaseToolsConnectionOCIClient struct {
	createFn func(context.Context, databasetoolssdk.CreateDatabaseToolsConnectionRequest) (databasetoolssdk.CreateDatabaseToolsConnectionResponse, error)
	getFn    func(context.Context, databasetoolssdk.GetDatabaseToolsConnectionRequest) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error)
	listFn   func(context.Context, databasetoolssdk.ListDatabaseToolsConnectionsRequest) (databasetoolssdk.ListDatabaseToolsConnectionsResponse, error)
	updateFn func(context.Context, databasetoolssdk.UpdateDatabaseToolsConnectionRequest) (databasetoolssdk.UpdateDatabaseToolsConnectionResponse, error)
	deleteFn func(context.Context, databasetoolssdk.DeleteDatabaseToolsConnectionRequest) (databasetoolssdk.DeleteDatabaseToolsConnectionResponse, error)
}

func (f *fakeDatabaseToolsConnectionOCIClient) CreateDatabaseToolsConnection(
	ctx context.Context,
	req databasetoolssdk.CreateDatabaseToolsConnectionRequest,
) (databasetoolssdk.CreateDatabaseToolsConnectionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return databasetoolssdk.CreateDatabaseToolsConnectionResponse{}, nil
}

func (f *fakeDatabaseToolsConnectionOCIClient) GetDatabaseToolsConnection(
	ctx context.Context,
	req databasetoolssdk.GetDatabaseToolsConnectionRequest,
) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return databasetoolssdk.GetDatabaseToolsConnectionResponse{}, nil
}

func (f *fakeDatabaseToolsConnectionOCIClient) ListDatabaseToolsConnections(
	ctx context.Context,
	req databasetoolssdk.ListDatabaseToolsConnectionsRequest,
) (databasetoolssdk.ListDatabaseToolsConnectionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return databasetoolssdk.ListDatabaseToolsConnectionsResponse{}, nil
}

func (f *fakeDatabaseToolsConnectionOCIClient) UpdateDatabaseToolsConnection(
	ctx context.Context,
	req databasetoolssdk.UpdateDatabaseToolsConnectionRequest,
) (databasetoolssdk.UpdateDatabaseToolsConnectionResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return databasetoolssdk.UpdateDatabaseToolsConnectionResponse{}, nil
}

func (f *fakeDatabaseToolsConnectionOCIClient) DeleteDatabaseToolsConnection(
	ctx context.Context,
	req databasetoolssdk.DeleteDatabaseToolsConnectionRequest,
) (databasetoolssdk.DeleteDatabaseToolsConnectionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return databasetoolssdk.DeleteDatabaseToolsConnectionResponse{}, nil
}

func testDatabaseToolsConnectionClient(fake *fakeDatabaseToolsConnectionOCIClient) DatabaseToolsConnectionServiceClient {
	return newDatabaseToolsConnectionServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		nil,
		fake,
	)
}

func makeGenericJDBCResource() *databasetoolsv1beta1.DatabaseToolsConnection {
	return &databasetoolsv1beta1.DatabaseToolsConnection{
		Spec: databasetoolsv1beta1.DatabaseToolsConnectionSpec{
			Type:          "GENERIC_JDBC",
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "jdbc-primary",
			Url:           "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
			UserName:      "app-user",
			UserPassword: databasetoolsv1beta1.DatabaseToolsConnectionUserPassword{
				ValueType: "SECRETID",
				SecretId:  "ocid1.secret.oc1..db-password",
			},
			AdvancedProperties: map[string]string{"sslMode": "REQUIRED"},
			KeyStores: []databasetoolsv1beta1.DatabaseToolsConnectionKeyStore{
				{
					KeyStoreType: "JAVA_TRUST_STORE",
					KeyStoreContent: databasetoolsv1beta1.DatabaseToolsConnectionKeyStoreKeyStoreContent{
						ValueType: "SECRETID",
						SecretId:  "ocid1.secret.oc1..truststore",
					},
					KeyStorePassword: databasetoolsv1beta1.DatabaseToolsConnectionKeyStoreKeyStorePassword{
						ValueType: "SECRETID",
						SecretId:  "ocid1.secret.oc1..truststore-password",
					},
				},
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			RuntimeSupport: "SUPPORTED",
		},
	}
}

func makePostgresqlResource() *databasetoolsv1beta1.DatabaseToolsConnection {
	return &databasetoolsv1beta1.DatabaseToolsConnection{
		Spec: databasetoolsv1beta1.DatabaseToolsConnectionSpec{
			Type:             "POSTGRESQL",
			CompartmentId:    "ocid1.compartment.oc1..example",
			DisplayName:      "shared-display-name",
			ConnectionString: "postgresql://db.example.com:5432/service_a",
			UserName:         "postgres-user",
			UserPassword: databasetoolsv1beta1.DatabaseToolsConnectionUserPassword{
				ValueType: "SECRETID",
				SecretId:  "ocid1.secret.oc1..postgres-password",
			},
			RelatedResource: databasetoolsv1beta1.DatabaseToolsConnectionRelatedResource{
				EntityType: "POSTGRESQLDBSYSTEM",
				Identifier: "ocid1.postgresqldbsystem.oc1..service-a",
			},
			PrivateEndpointId: "ocid1.databasetoolsprivateendpoint.oc1..example",
			RuntimeSupport:    "SUPPORTED",
		},
	}
}

func makeOracleResource() *databasetoolsv1beta1.DatabaseToolsConnection {
	return &databasetoolsv1beta1.DatabaseToolsConnection{
		Spec: databasetoolsv1beta1.DatabaseToolsConnectionSpec{
			Type:             "ORACLE_DATABASE",
			CompartmentId:    "ocid1.compartment.oc1..example",
			DisplayName:      "oracle-primary",
			ConnectionString: "tcps://db.example.com:1521/service_high",
			UserName:         "admin-user",
			UserPassword: databasetoolsv1beta1.DatabaseToolsConnectionUserPassword{
				ValueType: "SECRETID",
				SecretId:  "ocid1.secret.oc1..oracle-password",
			},
			AdvancedProperties: map[string]string{"oracle.net.ssl_server_dn_match": "true"},
			RelatedResource: databasetoolsv1beta1.DatabaseToolsConnectionRelatedResource{
				EntityType: "DATABASE",
				Identifier: "ocid1.database.oc1..oracle",
			},
			PrivateEndpointId: "ocid1.databasetoolsprivateendpoint.oc1..oracle",
			ProxyClient: databasetoolsv1beta1.DatabaseToolsConnectionProxyClient{
				ProxyAuthenticationType: "USER_NAME",
				UserName:                "proxy-user",
				UserPassword: databasetoolsv1beta1.DatabaseToolsConnectionProxyClientUserPassword{
					ValueType: "SECRETID",
					SecretId:  "ocid1.secret.oc1..proxy-password",
				},
				Roles: []string{"CONNECT"},
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "24"},
			},
			RuntimeSupport: "SUPPORTED",
		},
	}
}

func makeGenericJDBCSDKConnection(
	id string,
	state databasetoolssdk.LifecycleStateEnum,
) databasetoolssdk.DatabaseToolsConnectionGenericJdbc {
	return databasetoolssdk.DatabaseToolsConnectionGenericJdbc{
		Id:                 common.String(id),
		DisplayName:        common.String("jdbc-primary"),
		CompartmentId:      common.String("ocid1.compartment.oc1..example"),
		Url:                common.String("jdbc:oracle:thin:@tcp://db.example.com:1521/service"),
		UserName:           common.String("app-user"),
		UserPassword:       databasetoolssdk.DatabaseToolsUserPasswordSecretId{SecretId: common.String("ocid1.secret.oc1..db-password")},
		AdvancedProperties: map[string]string{"sslMode": "REQUIRED"},
		KeyStores: []databasetoolssdk.DatabaseToolsKeyStoreGenericJdbc{
			{
				KeyStoreType:    databasetoolssdk.KeyStoreTypeGenericJdbcJavaTrustStore,
				KeyStoreContent: databasetoolssdk.DatabaseToolsKeyStoreContentSecretIdGenericJdbc{SecretId: common.String("ocid1.secret.oc1..truststore")},
				KeyStorePassword: databasetoolssdk.DatabaseToolsKeyStorePasswordSecretIdGenericJdbc{
					SecretId: common.String("ocid1.secret.oc1..truststore-password"),
				},
			},
		},
		LifecycleState: state,
		RuntimeSupport: databasetoolssdk.RuntimeSupportSupported,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makePostgresqlSDKConnectionSummary(
	id string,
	connectionString string,
	relatedIdentifier string,
	state databasetoolssdk.LifecycleStateEnum,
) databasetoolssdk.DatabaseToolsConnectionPostgresqlSummary {
	return databasetoolssdk.DatabaseToolsConnectionPostgresqlSummary{
		Id:               common.String(id),
		DisplayName:      common.String("shared-display-name"),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		ConnectionString: common.String(connectionString),
		RelatedResource: &databasetoolssdk.DatabaseToolsRelatedResourcePostgresql{
			EntityType: databasetoolssdk.RelatedResourceEntityTypePostgresqlPostgresqldbsystem,
			Identifier: common.String(relatedIdentifier),
		},
		UserName:          common.String("postgres-user"),
		PrivateEndpointId: common.String("ocid1.databasetoolsprivateendpoint.oc1..example"),
		LifecycleState:    state,
		RuntimeSupport:    databasetoolssdk.RuntimeSupportSupported,
	}
}

func makePostgresqlSDKConnection(
	id string,
	state databasetoolssdk.LifecycleStateEnum,
) databasetoolssdk.DatabaseToolsConnectionPostgresql {
	return databasetoolssdk.DatabaseToolsConnectionPostgresql{
		Id:               common.String(id),
		DisplayName:      common.String("shared-display-name"),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		ConnectionString: common.String("postgresql://db.example.com:5432/service_a"),
		RelatedResource: &databasetoolssdk.DatabaseToolsRelatedResourcePostgresql{
			EntityType: databasetoolssdk.RelatedResourceEntityTypePostgresqlPostgresqldbsystem,
			Identifier: common.String("ocid1.postgresqldbsystem.oc1..service-a"),
		},
		UserName:          common.String("postgres-user"),
		UserPassword:      databasetoolssdk.DatabaseToolsUserPasswordSecretId{SecretId: common.String("ocid1.secret.oc1..postgres-password")},
		PrivateEndpointId: common.String("ocid1.databasetoolsprivateendpoint.oc1..example"),
		LifecycleState:    state,
		RuntimeSupport:    databasetoolssdk.RuntimeSupportSupported,
	}
}

func makeOracleSDKConnection(
	id string,
	connectionString string,
	proxyClient databasetoolssdk.DatabaseToolsConnectionOracleDatabaseProxyClient,
	state databasetoolssdk.LifecycleStateEnum,
) databasetoolssdk.DatabaseToolsConnectionOracleDatabase {
	return databasetoolssdk.DatabaseToolsConnectionOracleDatabase{
		Id:               common.String(id),
		DisplayName:      common.String("oracle-primary"),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		ConnectionString: common.String(connectionString),
		RelatedResource: &databasetoolssdk.DatabaseToolsRelatedResource{
			EntityType: databasetoolssdk.RelatedResourceEntityTypeDatabase,
			Identifier: common.String("ocid1.database.oc1..oracle"),
		},
		UserName:          common.String("admin-user"),
		UserPassword:      databasetoolssdk.DatabaseToolsUserPasswordSecretId{SecretId: common.String("ocid1.secret.oc1..oracle-password")},
		PrivateEndpointId: common.String("ocid1.databasetoolsprivateendpoint.oc1..oracle"),
		ProxyClient:       proxyClient,
		AdvancedProperties: map[string]string{
			"oracle.net.ssl_server_dn_match": "true",
		},
		LifecycleState: state,
		RuntimeSupport: databasetoolssdk.RuntimeSupportSupported,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "24"}},
	}
}

func requireAsyncCurrent(
	t *testing.T,
	resource *databasetoolsv1beta1.DatabaseToolsConnection,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want active async tracker")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func TestDatabaseToolsConnectionServiceClientCreateOrUpdateCreatesGenericJDBCAndRequeues(t *testing.T) {
	t.Parallel()

	var createRequest databasetoolssdk.CreateDatabaseToolsConnectionRequest
	var getRequest databasetoolssdk.GetDatabaseToolsConnectionRequest

	client := testDatabaseToolsConnectionClient(&fakeDatabaseToolsConnectionOCIClient{
		createFn: func(_ context.Context, req databasetoolssdk.CreateDatabaseToolsConnectionRequest) (databasetoolssdk.CreateDatabaseToolsConnectionResponse, error) {
			createRequest = req
			return databasetoolssdk.CreateDatabaseToolsConnectionResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				DatabaseToolsConnection: makeGenericJDBCSDKConnection(
					"ocid1.databasetoolsconnection.oc1..created",
					databasetoolssdk.LifecycleStateCreating,
				),
			}, nil
		},
		getFn: func(_ context.Context, req databasetoolssdk.GetDatabaseToolsConnectionRequest) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error) {
			getRequest = req
			return databasetoolssdk.GetDatabaseToolsConnectionResponse{
				DatabaseToolsConnection: makeGenericJDBCSDKConnection(
					"ocid1.databasetoolsconnection.oc1..created",
					databasetoolssdk.LifecycleStateCreating,
				),
			}, nil
		},
	})

	resource := makeGenericJDBCResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create is in progress")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the connection remains CREATING")
	}

	details, ok := createRequest.CreateDatabaseToolsConnectionDetails.(databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreateDatabaseToolsConnectionGenericJdbcDetails", createRequest.CreateDatabaseToolsConnectionDetails)
	}
	if details.Url == nil || *details.Url != resource.Spec.Url {
		t.Fatalf("create url = %v, want %q", details.Url, resource.Spec.Url)
	}
	if _, ok := details.UserPassword.(databasetoolssdk.DatabaseToolsUserPasswordSecretIdDetails); !ok {
		t.Fatalf("create userPassword type = %T, want DatabaseToolsUserPasswordSecretIdDetails", details.UserPassword)
	}
	if len(details.KeyStores) != 1 {
		t.Fatalf("create keyStores len = %d, want 1", len(details.KeyStores))
	}
	if _, ok := details.KeyStores[0].KeyStoreContent.(databasetoolssdk.DatabaseToolsKeyStoreContentSecretIdGenericJdbcDetails); !ok {
		t.Fatalf("create keyStoreContent type = %T, want DatabaseToolsKeyStoreContentSecretIdGenericJdbcDetails", details.KeyStores[0].KeyStoreContent)
	}
	if _, ok := details.KeyStores[0].KeyStorePassword.(databasetoolssdk.DatabaseToolsKeyStorePasswordSecretIdGenericJdbcDetails); !ok {
		t.Fatalf("create keyStorePassword type = %T, want DatabaseToolsKeyStorePasswordSecretIdGenericJdbcDetails", details.KeyStores[0].KeyStorePassword)
	}

	if getRequest.DatabaseToolsConnectionId == nil || *getRequest.DatabaseToolsConnectionId != "ocid1.databasetoolsconnection.oc1..created" {
		t.Fatalf("get databaseToolsConnectionId = %v, want created connection ID", getRequest.DatabaseToolsConnectionId)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.databasetoolsconnection.oc1..created" {
		t.Fatalf("status.ocid = %q, want created connection ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "CREATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "CREATING")
	}
}

func TestDatabaseToolsConnectionServiceClientCreateOrUpdateResolvesExistingUsingFormalMatchFields(t *testing.T) {
	t.Parallel()

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := testDatabaseToolsConnectionClient(&fakeDatabaseToolsConnectionOCIClient{
		createFn: func(_ context.Context, _ databasetoolssdk.CreateDatabaseToolsConnectionRequest) (databasetoolssdk.CreateDatabaseToolsConnectionResponse, error) {
			createCalls++
			return databasetoolssdk.CreateDatabaseToolsConnectionResponse{}, nil
		},
		listFn: func(_ context.Context, req databasetoolssdk.ListDatabaseToolsConnectionsRequest) (databasetoolssdk.ListDatabaseToolsConnectionsResponse, error) {
			listCalls++
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..example" {
				t.Fatalf("list compartmentId = %v, want spec compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "shared-display-name" {
				t.Fatalf("list displayName = %v, want spec displayName", req.DisplayName)
			}
			if req.RelatedResourceIdentifier == nil || *req.RelatedResourceIdentifier != "ocid1.postgresqldbsystem.oc1..service-a" {
				t.Fatalf("list relatedResourceIdentifier = %v, want spec related resource identifier", req.RelatedResourceIdentifier)
			}
			if len(req.Type) != 0 {
				t.Fatalf("list type = %#v, want omitted reviewed request field", req.Type)
			}
			if len(req.RuntimeSupport) != 0 {
				t.Fatalf("list runtimeSupport = %#v, want omitted reviewed request field", req.RuntimeSupport)
			}
			return databasetoolssdk.ListDatabaseToolsConnectionsResponse{
				DatabaseToolsConnectionCollection: databasetoolssdk.DatabaseToolsConnectionCollection{
					Items: []databasetoolssdk.DatabaseToolsConnectionSummary{
						makePostgresqlSDKConnectionSummary(
							"ocid1.databasetoolsconnection.oc1..wrong",
							"postgresql://db.example.com:5432/service_b",
							"ocid1.postgresqldbsystem.oc1..service-b",
							databasetoolssdk.LifecycleStateActive,
						),
						makePostgresqlSDKConnectionSummary(
							"ocid1.databasetoolsconnection.oc1..existing",
							"postgresql://db.example.com:5432/service_a",
							"ocid1.postgresqldbsystem.oc1..service-a",
							databasetoolssdk.LifecycleStateInactive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req databasetoolssdk.GetDatabaseToolsConnectionRequest) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error) {
			getCalls++
			if req.DatabaseToolsConnectionId == nil || *req.DatabaseToolsConnectionId != "ocid1.databasetoolsconnection.oc1..existing" {
				t.Fatalf("get databaseToolsConnectionId = %v, want resolved connection ID", req.DatabaseToolsConnectionId)
			}
			return databasetoolssdk.GetDatabaseToolsConnectionResponse{
				DatabaseToolsConnection: makePostgresqlSDKConnection(
					"ocid1.databasetoolsconnection.oc1..existing",
					databasetoolssdk.LifecycleStateInactive,
				),
			}, nil
		},
	})

	resource := makePostgresqlResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for steady INACTIVE lifecycle")
	}
	if createCalls != 0 {
		t.Fatalf("CreateDatabaseToolsConnection() calls = %d, want 0", createCalls)
	}
	if listCalls != 1 {
		t.Fatalf("ListDatabaseToolsConnections() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDatabaseToolsConnection() calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != "ocid1.databasetoolsconnection.oc1..existing" {
		t.Fatalf("status.id = %q, want resolved connection ID", resource.Status.Id)
	}
	if resource.Status.LifecycleState != "INACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "INACTIVE")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for steady INACTIVE lifecycle", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDatabaseToolsConnectionServiceClientCreateOrUpdateUpdatesOracleConnection(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest databasetoolssdk.UpdateDatabaseToolsConnectionRequest

	client := testDatabaseToolsConnectionClient(&fakeDatabaseToolsConnectionOCIClient{
		getFn: func(_ context.Context, req databasetoolssdk.GetDatabaseToolsConnectionRequest) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error) {
			getCalls++
			if req.DatabaseToolsConnectionId == nil || *req.DatabaseToolsConnectionId != "ocid1.databasetoolsconnection.oc1..existing" {
				t.Fatalf("get databaseToolsConnectionId = %v, want tracked connection ID", req.DatabaseToolsConnectionId)
			}
			switch getCalls {
			case 1:
				return databasetoolssdk.GetDatabaseToolsConnectionResponse{
					DatabaseToolsConnection: makeOracleSDKConnection(
						"ocid1.databasetoolsconnection.oc1..existing",
						"tcps://db.example.com:1521/service_low",
						databasetoolssdk.DatabaseToolsConnectionOracleDatabaseProxyClientNoProxy{},
						databasetoolssdk.LifecycleStateActive,
					),
				}, nil
			case 2:
				return databasetoolssdk.GetDatabaseToolsConnectionResponse{
					DatabaseToolsConnection: makeOracleSDKConnection(
						"ocid1.databasetoolsconnection.oc1..existing",
						"tcps://db.example.com:1521/service_high",
						databasetoolssdk.DatabaseToolsConnectionOracleDatabaseProxyClientUserName{
							UserName:     common.String("proxy-user"),
							UserPassword: databasetoolssdk.DatabaseToolsUserPasswordSecretId{SecretId: common.String("ocid1.secret.oc1..proxy-password")},
							Roles:        []string{"CONNECT"},
						},
						databasetoolssdk.LifecycleStateUpdating,
					),
				}, nil
			default:
				t.Fatalf("unexpected GetDatabaseToolsConnection() call %d", getCalls)
				return databasetoolssdk.GetDatabaseToolsConnectionResponse{}, nil
			}
		},
		updateFn: func(_ context.Context, req databasetoolssdk.UpdateDatabaseToolsConnectionRequest) (databasetoolssdk.UpdateDatabaseToolsConnectionResponse, error) {
			updateRequest = req
			return databasetoolssdk.UpdateDatabaseToolsConnectionResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
	})

	resource := makeOracleResource()
	resource.Status.Id = "ocid1.databasetoolsconnection.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while the follow-up read reports UPDATING")
	}
	if getCalls != 2 {
		t.Fatalf("GetDatabaseToolsConnection() calls = %d, want 2", getCalls)
	}
	if updateRequest.DatabaseToolsConnectionId == nil || *updateRequest.DatabaseToolsConnectionId != "ocid1.databasetoolsconnection.oc1..existing" {
		t.Fatalf("update databaseToolsConnectionId = %v, want tracked connection ID", updateRequest.DatabaseToolsConnectionId)
	}
	if updateRequest.IsLockOverride != nil {
		t.Fatalf("update isLockOverride = %#v, want reviewed hook field omission", updateRequest.IsLockOverride)
	}

	details, ok := updateRequest.UpdateDatabaseToolsConnectionDetails.(databasetoolssdk.UpdateDatabaseToolsConnectionOracleDatabaseDetails)
	if !ok {
		t.Fatalf("update body type = %T, want UpdateDatabaseToolsConnectionOracleDatabaseDetails", updateRequest.UpdateDatabaseToolsConnectionDetails)
	}
	if details.ConnectionString == nil || *details.ConnectionString != resource.Spec.ConnectionString {
		t.Fatalf("update connectionString = %v, want %q", details.ConnectionString, resource.Spec.ConnectionString)
	}
	proxyDetails, ok := details.ProxyClient.(databasetoolssdk.DatabaseToolsConnectionOracleDatabaseProxyClientUserNameDetails)
	if !ok {
		t.Fatalf("update proxyClient type = %T, want DatabaseToolsConnectionOracleDatabaseProxyClientUserNameDetails", details.ProxyClient)
	}
	if proxyDetails.UserName == nil || *proxyDetails.UserName != "proxy-user" {
		t.Fatalf("update proxyClient.userName = %v, want %q", proxyDetails.UserName, "proxy-user")
	}
	if _, ok := proxyDetails.UserPassword.(databasetoolssdk.DatabaseToolsUserPasswordSecretIdDetails); !ok {
		t.Fatalf("update proxyClient.userPassword type = %T, want DatabaseToolsUserPasswordSecretIdDetails", proxyDetails.UserPassword)
	}

	if resource.Status.ConnectionString != resource.Spec.ConnectionString {
		t.Fatalf("status.connectionString = %q, want %q", resource.Status.ConnectionString, resource.Spec.ConnectionString)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1")
	if resource.Status.OsokStatus.Async.Current.RawStatus != "UPDATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "UPDATING")
	}
}

func TestDatabaseToolsConnectionServiceClientDeleteConfirmsLifecycleDelete(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0

	client := testDatabaseToolsConnectionClient(&fakeDatabaseToolsConnectionOCIClient{
		getFn: func(_ context.Context, req databasetoolssdk.GetDatabaseToolsConnectionRequest) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error) {
			getCalls++
			if req.DatabaseToolsConnectionId == nil || *req.DatabaseToolsConnectionId != "ocid1.databasetoolsconnection.oc1..existing" {
				t.Fatalf("get databaseToolsConnectionId = %v, want tracked connection ID", req.DatabaseToolsConnectionId)
			}
			state := databasetoolssdk.LifecycleStateActive
			if getCalls > 1 {
				state = databasetoolssdk.LifecycleStateDeleting
			}
			return databasetoolssdk.GetDatabaseToolsConnectionResponse{
				DatabaseToolsConnection: makeGenericJDBCSDKConnection(
					"ocid1.databasetoolsconnection.oc1..existing",
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, req databasetoolssdk.DeleteDatabaseToolsConnectionRequest) (databasetoolssdk.DeleteDatabaseToolsConnectionResponse, error) {
			deleteCalls++
			if req.DatabaseToolsConnectionId == nil || *req.DatabaseToolsConnectionId != "ocid1.databasetoolsconnection.oc1..existing" {
				t.Fatalf("delete databaseToolsConnectionId = %v, want tracked connection ID", req.DatabaseToolsConnectionId)
			}
			if req.IsLockOverride != nil {
				t.Fatalf("delete isLockOverride = %#v, want reviewed hook field omission", req.IsLockOverride)
			}
			return databasetoolssdk.DeleteDatabaseToolsConnectionResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	resource := makeGenericJDBCResource()
	resource.Status.Id = "ocid1.databasetoolsconnection.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should report in-progress delete while OCI still returns DELETING")
	}
	if getCalls != 2 {
		t.Fatalf("GetDatabaseToolsConnection() calls = %d, want 2", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDatabaseToolsConnection() calls = %d, want 1", deleteCalls)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "DELETING")
	}
}

func TestDatabaseToolsConnectionServiceClientRejectsUnsupportedFieldsForGenericJDBC(t *testing.T) {
	t.Parallel()

	createCalls := 0
	client := testDatabaseToolsConnectionClient(&fakeDatabaseToolsConnectionOCIClient{
		createFn: func(_ context.Context, _ databasetoolssdk.CreateDatabaseToolsConnectionRequest) (databasetoolssdk.CreateDatabaseToolsConnectionResponse, error) {
			createCalls++
			return databasetoolssdk.CreateDatabaseToolsConnectionResponse{}, nil
		},
	})

	resource := makeGenericJDBCResource()
	resource.Spec.PrivateEndpointId = "ocid1.databasetoolsprivateendpoint.oc1..unexpected"
	resource.Spec.ProxyClient = databasetoolsv1beta1.DatabaseToolsConnectionProxyClient{
		ProxyAuthenticationType: "USER_NAME",
		UserName:                "proxy-user",
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported field failure")
	}
	if createCalls != 0 {
		t.Fatalf("CreateDatabaseToolsConnection() calls = %d, want 0", createCalls)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when the selected type rejects fields")
	}
	if !strings.Contains(err.Error(), "DatabaseToolsConnection type GENERIC_JDBC does not support fields: privateEndpointId, proxyClient") {
		t.Fatalf("CreateOrUpdate() error = %q, want unsupported field failure", err.Error())
	}
}
