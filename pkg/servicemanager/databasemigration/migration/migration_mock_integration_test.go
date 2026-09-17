/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package migration

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
func TestMockIntegrationMigrationWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &databasemigrationv1beta1.Migration{}
	ocimock.InitializeResource(resource, "mock-migration")
	resource.Spec = ocimock.MustJSONFixture[databasemigrationv1beta1.MigrationSpec](t, `{
  "assessmentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "dataTransferMediumDetails": {
    "objectStorageBucket": {
      "bucketName": "mysql-dumps",
      "namespaceName": "migration-ns"
    },
    "type": "OBJECT_STORAGE"
  },
  "databaseCombination": "MYSQL",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql migration",
  "displayName": "mysql-migration",
  "excludeObjects": [

  ],
  "freeformTags": {
    "env": "test"
  },
  "hubDetails": {
    "acceptableLag": 30,
    "keyId": "<ocid:3>",
    "restAdminCredentials": {
      "password": "<redacted>",
      "username": "ggadmin"
    },
    "url": "https://gg.mysql.example.com",
    "vaultId": "<ocid:4>"
  },
  "includeObjects": [

  ],
  "sourceDatabaseConnectionId": "<ocid:5>",
  "targetDatabaseConnectionId": "<ocid:6>",
  "type": "ONLINE"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "migration updated"
}`)

	createRequest := ocimock.MustJSONFixture[databasemigrationsdk.CreateMySqlMigrationDetails](t, `{
  "assessmentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "dataTransferMediumDetails": {
    "objectStorageBucket": {
      "bucketName": "mysql-dumps",
      "namespaceName": "migration-ns"
    },
    "type": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql migration",
  "displayName": "mysql-migration",
  "excludeObjects": [

  ],
  "freeformTags": {
    "env": "test"
  },
  "hubDetails": {
    "acceptableLag": 30,
    "keyId": "<ocid:3>",
    "restAdminCredentials": {
      "password": "<redacted>",
      "username": "ggadmin"
    },
    "url": "https://gg.mysql.example.com",
    "vaultId": "<ocid:4>"
  },
  "includeObjects": [

  ],
  "sourceDatabaseConnectionId": "<ocid:5>",
  "targetDatabaseConnectionId": "<ocid:6>",
  "type": "ONLINE"
}`)
	updateRequest := ocimock.MustJSONFixture[databasemigrationsdk.UpdateMySqlMigrationDetails](t, `{
  "dataTransferMediumDetails": {
    "objectStorageBucket": {
      "bucketName": "mysql-dumps",
      "namespaceName": "migration-ns"
    },
    "type": "OBJECT_STORAGE"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "migration updated",
  "displayName": "mysql-migration",
  "freeformTags": {
    "env": "test"
  },
  "hubDetails": {
    "acceptableLag": 30,
    "keyId": "<ocid:3>",
    "restAdminCredentials": {
      "password": "<redacted>",
      "username": "ggadmin"
    },
    "url": "https://gg.mysql.example.com",
    "vaultId": "<ocid:4>"
  },
  "sourceDatabaseConnectionId": "<ocid:5>",
  "targetDatabaseConnectionId": "<ocid:6>",
  "type": "ONLINE"
}`)
	createdState := ocimock.MustOCIResponseFixture[databasemigrationsdk.MySqlMigration](t, `{
  "assessmentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "dataTransferMediumDetails": {
    "objectStorageBucket": {
      "bucketName": "mysql-dumps",
      "namespaceName": "migration-ns"
    },
    "type": "OBJECT_STORAGE"
  },
  "databaseCombination": "MYSQL",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql migration",
  "displayName": "mysql-migration",
  "freeformTags": {
    "env": "test"
  },
  "hubDetails": {
    "acceptableLag": 30,
    "keyId": "<ocid:3>",
    "restAdminCredentials": {
      "password": "<redacted>",
      "username": "ggadmin"
    },
    "url": "https://gg.mysql.example.com",
    "vaultId": "<ocid:4>"
  },
  "id": "<ocid:8>",
  "lifecycleState": "ACTIVE",
  "sourceDatabaseConnectionId": "<ocid:5>",
  "targetDatabaseConnectionId": "<ocid:6>",
  "type": "ONLINE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[databasemigrationsdk.MySqlMigration](t, `{
  "assessmentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "dataTransferMediumDetails": {
    "objectStorageBucket": {
      "bucketName": "mysql-dumps",
      "namespaceName": "migration-ns"
    },
    "type": "OBJECT_STORAGE"
  },
  "databaseCombination": "MYSQL",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "migration updated",
  "displayName": "mysql-migration",
  "freeformTags": {
    "env": "test"
  },
  "hubDetails": {
    "acceptableLag": 30,
    "keyId": "<ocid:3>",
    "restAdminCredentials": {
      "password": "<redacted>",
      "username": "ggadmin"
    },
    "url": "https://gg.mysql.example.com",
    "vaultId": "<ocid:4>"
  },
  "id": "<ocid:8>",
  "lifecycleState": "ACTIVE",
  "sourceDatabaseConnectionId": "<ocid:5>",
  "targetDatabaseConnectionId": "<ocid:6>",
  "type": "ONLINE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:9>",
  "id": "<ocid:7>",
  "operationType": "CREATE_MIGRATION",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "Migration",
      "entityUri": null,
      "identifier": "<ocid:8>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:9>",
  "id": "wr-update",
  "operationType": "UPDATE_MIGRATION",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "Migration",
      "entityUri": null,
      "identifier": "<ocid:8>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:9>",
  "id": "<ocid:10>",
  "operationType": "DELETE_MIGRATION",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "Migration",
      "entityUri": null,
      "identifier": "<ocid:8>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[databasemigrationsdk.MySqlMigration, databasemigrationsdk.CreateMySqlMigrationDetails, databasemigrationsdk.UpdateMySqlMigrationDetails]{
		CollectionPath: "/20230518/migrations", ItemPath: "/20230518/migrations/<ocid:8>",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: http.StatusAccepted, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:10>"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "databaseCombination", "MYSQL", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "databaseCombination", "MYSQL", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/<ocid:7>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/wr-update", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/<ocid:10>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newMigrationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, databasemigrationsdk.DatabaseMigrationClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*databasemigrationv1beta1.Migration]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *databasemigrationv1beta1.Migration) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Migration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *databasemigrationv1beta1.Migration) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *databasemigrationv1beta1.Migration) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Migration status = %+v", current.Status)
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
