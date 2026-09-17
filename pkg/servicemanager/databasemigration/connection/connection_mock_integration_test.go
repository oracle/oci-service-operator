/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package connection

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: explicit MYSQL subtype selected from the CR discriminator,
// vendored SDK, production runtime, formal lifecycle, and sanitized evidence.
func TestMockIntegrationConnectionWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &databasemigrationv1beta1.Connection{}
	ocimock.InitializeResource(resource, "mock-connection")
	resource.Spec = ocimock.MustJSONFixture[databasemigrationv1beta1.ConnectionSpec](t, `{
  "additionalAttributes": [
    {
      "name": "tlsVersion",
      "value": "TLSv1.2"
    }
  ],
  "compartmentId": "<ocid:1>",
  "connectionType": "MYSQL",
  "databaseName": "appdb",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql source connection",
  "displayName": "mysql-source",
  "freeformTags": {
    "env": "test"
  },
  "host": "mysql.example.com",
  "keyId": "<ocid:2>",
  "nsgIds": [
    "<ocid:3>"
  ],
  "password": "<redacted>",
  "port": 3306,
  "replicationPassword": "<redacted>",
  "replicationUsername": "replication-user",
  "securityProtocol": "TLS",
  "sslMode": "REQUIRED",
  "subnetId": "<ocid:4>",
  "technologyType": "OCI_MYSQL",
  "username": "migration-user",
  "vaultId": "<ocid:5>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "connection updated"
}`)

	createRequest := ocimock.MustJSONFixture[databasemigrationsdk.CreateMysqlConnectionDetails](t, `{
  "additionalAttributes": [
    {
      "name": "tlsVersion",
      "value": "TLSv1.2"
    }
  ],
  "compartmentId": "<ocid:1>",
  "databaseName": "appdb",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql source connection",
  "displayName": "mysql-source",
  "freeformTags": {
    "env": "test"
  },
  "host": "mysql.example.com",
  "keyId": "<ocid:2>",
  "nsgIds": [
    "<ocid:3>"
  ],
  "password": "<redacted>",
  "port": 3306,
  "replicationPassword": "<redacted>",
  "replicationUsername": "replication-user",
  "securityProtocol": "TLS",
  "sslMode": "REQUIRED",
  "subnetId": "<ocid:4>",
  "technologyType": "OCI_MYSQL",
  "username": "migration-user",
  "vaultId": "<ocid:5>"
}`)
	updateRequest := ocimock.MustJSONFixture[databasemigrationsdk.UpdateMysqlConnectionDetails](t, `{
  "additionalAttributes": [
    {
      "name": "tlsVersion",
      "value": "TLSv1.2"
    }
  ],
  "databaseName": "appdb",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "connection updated",
  "displayName": "mysql-source",
  "freeformTags": {
    "env": "test"
  },
  "host": "mysql.example.com",
  "keyId": "<ocid:2>",
  "nsgIds": [
    "<ocid:3>"
  ],
  "password": "<redacted>",
  "port": 3306,
  "replicationPassword": "<redacted>",
  "replicationUsername": "replication-user",
  "securityProtocol": "TLS",
  "sslMode": "REQUIRED",
  "subnetId": "<ocid:4>",
  "username": "migration-user",
  "vaultId": "<ocid:5>"
}`)
	createdState := ocimock.MustOCIResponseFixture[databasemigrationsdk.MysqlConnection](t, `{
  "additionalAttributes": [
    {
      "name": "tlsVersion",
      "value": "TLSv1.2"
    }
  ],
  "compartmentId": "<ocid:1>",
  "connectionType": "MYSQL",
  "databaseName": "appdb",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql source connection",
  "displayName": "mysql-source",
  "freeformTags": {
    "env": "test"
  },
  "host": "mysql.example.com",
  "id": "<ocid:7>",
  "keyId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "<ocid:3>"
  ],
  "password": "<redacted>",
  "port": 3306,
  "replicationPassword": "<redacted>",
  "replicationUsername": "replication-user",
  "securityProtocol": "TLS",
  "sslMode": "REQUIRED",
  "subnetId": "<ocid:4>",
  "technologyType": "OCI_MYSQL",
  "username": "migration-user",
  "vaultId": "<ocid:5>"
}`)
	updatedState := ocimock.MustOCIResponseFixture[databasemigrationsdk.MysqlConnection](t, `{
  "additionalAttributes": [
    {
      "name": "tlsVersion",
      "value": "TLSv1.2"
    }
  ],
  "compartmentId": "<ocid:1>",
  "connectionType": "MYSQL",
  "databaseName": "appdb",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "connection updated",
  "displayName": "mysql-source",
  "freeformTags": {
    "env": "test"
  },
  "host": "mysql.example.com",
  "id": "<ocid:7>",
  "keyId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "<ocid:3>"
  ],
  "password": "<redacted>",
  "port": 3306,
  "replicationPassword": "<redacted>",
  "replicationUsername": "replication-user",
  "securityProtocol": "TLS",
  "sslMode": "REQUIRED",
  "subnetId": "<ocid:4>",
  "technologyType": "OCI_MYSQL",
  "username": "migration-user",
  "vaultId": "<ocid:5>"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:8>",
  "id": "<ocid:6>",
  "operationType": "CREATE_CONNECTION",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "Connection",
      "entityUri": null,
      "identifier": "<ocid:7>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:8>",
  "id": "wr-update",
  "operationType": "UPDATE_CONNECTION",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "Connection",
      "entityUri": null,
      "identifier": "<ocid:7>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:8>",
  "id": "<ocid:9>",
  "operationType": "DELETE_CONNECTION",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "Connection",
      "entityUri": null,
      "identifier": "<ocid:7>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[databasemigrationsdk.MysqlConnection, databasemigrationsdk.CreateMysqlConnectionDetails, databasemigrationsdk.UpdateMysqlConnectionDetails]{
		CollectionPath: "/20230518/connections", ItemPath: "/20230518/connections/<ocid:7>",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: http.StatusAccepted, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:9>"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "connectionType", "MYSQL", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "connectionType", "MYSQL", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/wr-update", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/<ocid:9>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://database-migration.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230518", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newConnectionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, databasemigrationsdk.DatabaseMigrationClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*databasemigrationv1beta1.Connection]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *databasemigrationv1beta1.Connection) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Connection status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *databasemigrationv1beta1.Connection) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *databasemigrationv1beta1.Connection) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Connection status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
