/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package databasetoolsconnection

import (
	"context"
	"fmt"
	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
	databasetoolsv1beta1 "github.com/oracle/oci-service-operator/api/databasetools/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDatabaseToolsConnectionLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeGenericJDBCResource()
	ocimock.InitializeResource(resource, "mock-databasetoolsconnection")
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary-updated",
  "freeformTags": {
    "env": "test"
  },
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`)
	createRequest := ocimock.MustJSONFixture[databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails](t, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "compartmentId": "ocid1.compartment.oc1..example",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary",
  "freeformTags": {
    "env": "test"
  },
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "locks": [],
  "runtimeSupport": "SUPPORTED",
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[databasetoolssdk.DatabaseToolsConnectionGenericJdbc](t, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "compartmentId": "ocid1.compartment.oc1..example",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "lifecycleState": "ACTIVE",
  "locks": [],
  "runtimeSupport": "SUPPORTED",
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`)
	createdReadStates := []databasetoolssdk.DatabaseToolsConnectionGenericJdbc{
		ocimock.MustOCIResponseFixture[databasetoolssdk.DatabaseToolsConnectionGenericJdbc](t, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "compartmentId": "ocid1.compartment.oc1..example",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "lifecycleState": "ACTIVE",
  "locks": [],
  "runtimeSupport": "SUPPORTED",
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`),
	}
	updateRequest := ocimock.MustJSONFixture[databasetoolssdk.UpdateDatabaseToolsConnectionGenericJdbcDetails](t, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary-updated",
  "freeformTags": {
    "env": "test"
  },
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[databasetoolssdk.DatabaseToolsConnectionGenericJdbc](t, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "compartmentId": "ocid1.compartment.oc1..example",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary-updated",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "lifecycleState": "ACTIVE",
  "locks": [],
  "runtimeSupport": "SUPPORTED",
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`)
	updatedReadStates := []databasetoolssdk.DatabaseToolsConnectionGenericJdbc{
		ocimock.MustOCIResponseFixture[databasetoolssdk.DatabaseToolsConnectionGenericJdbc](t, `{
  "advancedProperties": {
    "sslMode": "REQUIRED"
  },
  "compartmentId": "ocid1.compartment.oc1..example",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "jdbc-primary-updated",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "keyStores": [
    {
      "keyStoreContent": {
        "secretId": "ocid1.secret.oc1..truststore",
        "valueType": "SECRETID"
      },
      "keyStorePassword": {
        "secretId": "ocid1.secret.oc1..truststore-password",
        "valueType": "SECRETID"
      },
      "keyStoreType": "JAVA_TRUST_STORE"
    }
  ],
  "lifecycleState": "ACTIVE",
  "locks": [],
  "runtimeSupport": "SUPPORTED",
  "type": "GENERIC_JDBC",
  "url": "jdbc:oracle:thin:@tcp://db.example.com:1521/service",
  "userName": "app-user",
  "userPassword": {
    "secretId": "ocid1.secret.oc1..db-password",
    "valueType": "SECRETID"
  }
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		databasetoolssdk.DatabaseToolsConnectionGenericJdbc,
		databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails,
		databasetoolssdk.UpdateDatabaseToolsConnectionGenericJdbcDetails,
	]{
		CollectionPath:     "/20201005/databaseToolsConnections",
		ItemPath:           "/20201005/databaseToolsConnections/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		CreatedReadStates:  append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...),
		UpdatedReadStates:  append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ databasetoolssdk.DatabaseToolsConnectionGenericJdbc) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20201005", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DatabaseToolsConnection OCI mock: %v", err)
		}
	})
	sdkClient := databasetoolssdk.DatabaseToolsClient{BaseClient: session.BaseClient()}
	client := newDatabaseToolsConnectionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, nil, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*databasetoolsv1beta1.DatabaseToolsConnection]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *databasetoolsv1beta1.DatabaseToolsConnection) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AdvancedProperties, current.Spec.AdvancedProperties) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.RuntimeSupport, current.Spec.RuntimeSupport) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) ||
				!reflect.DeepEqual(current.Status.Url, current.Spec.Url) ||
				!reflect.DeepEqual(current.Status.UserName, current.Spec.UserName) ||
				len(current.Status.KeyStores) != 1 ||
				current.Status.KeyStores[0].KeyStoreType != "JAVA_TRUST_STORE" ||
				current.Status.UserPassword.ValueType == "" {
				return fmt.Errorf("created DatabaseToolsConnection status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *databasetoolsv1beta1.DatabaseToolsConnection) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *databasetoolsv1beta1.DatabaseToolsConnection) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AdvancedProperties, current.Spec.AdvancedProperties) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) ||
				!reflect.DeepEqual(current.Status.Url, current.Spec.Url) ||
				!reflect.DeepEqual(current.Status.UserName, current.Spec.UserName) ||
				len(current.Status.KeyStores) != 1 ||
				current.Status.UserPassword.ValueType == "" {
				return fmt.Errorf("updated DatabaseToolsConnection status = %+v", current.Status)
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
