/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package distributeddatabase

import (
	"context"
	"fmt"
	distributeddatabasesdk "github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	distributeddatabasev1beta1 "github.com/oracle/oci-service-operator/api/distributeddatabase/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDistributedDatabaseLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newTestDistributedDatabaseResource()
	ocimock.InitializeResource(resource, "mock-distributeddatabase")
	resource.Spec = ocimock.MustJSONFixture[distributeddatabasev1beta1.DistributedDatabaseSpec](t, `{
  "catalogDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "availabilityDomain": "Uocm:PHX-AD-1",
      "dbStorageVaultDetails": {
        "highCapacityDatabaseStorage": 128
      },
      "shardSpace": "CATALOG",
      "source": "NEW_VAULT_AND_CLUSTER",
      "vmClusterDetails": {
        "backupNetworkNsgIds": [
          "\u003cocid:1\u003e"
        ],
        "backupSubnetId": "\u003cocid:2\u003e",
        "enabledECpuCount": 16,
        "isDiagnosticsEventsEnabled": false,
        "isHealthMonitoringEnabled": false,
        "isIncidentLogsEnabled": false,
        "nsgIds": [
          "\u003cocid:3\u003e"
        ],
        "privateZoneId": "\u003cocid:4\u003e",
        "sshPublicKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"
        ],
        "subnetId": "\u003cocid:5\u003e",
        "totalECpuCount": 32,
        "vmFileSystemStorageSize": 2048
      }
    }
  ],
  "characterSet": "AL32UTF8",
  "compartmentId": "\u003cocid:6\u003e",
  "databaseVersion": "23ai",
  "dbDeploymentType": "EXADB_XS",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "ddb-runtime",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "listenerPort": 1521,
  "ncharacterSet": "AL16UTF16",
  "onsPortLocal": 6234,
  "onsPortRemote": 6235,
  "prefix": "ddb123",
  "privateEndpointIds": [
    "\u003cocid:7\u003e"
  ],
  "shardDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "peerVmClusterIds": [
        "\u003cocid:8\u003e"
      ],
      "shardSpace": "PRIMARY",
      "source": "EXADB_XS",
      "vmClusterId": "\u003cocid:9\u003e"
    }
  ],
  "shardingMethod": "USER"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "ddb-runtime-updated"
}`)
	createRequest := ocimock.MustJSONFixture[distributeddatabasesdk.CreateDistributedDatabaseDetails](t, `{
  "catalogDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "availabilityDomain": "Uocm:PHX-AD-1",
      "dbStorageVaultDetails": {
        "highCapacityDatabaseStorage": 128
      },
      "shardSpace": "CATALOG",
      "source": "NEW_VAULT_AND_CLUSTER",
      "vmClusterDetails": {
        "backupNetworkNsgIds": [
          "\u003cocid:1\u003e"
        ],
        "backupSubnetId": "\u003cocid:2\u003e",
        "enabledECpuCount": 16,
        "isDiagnosticsEventsEnabled": false,
        "isHealthMonitoringEnabled": false,
        "isIncidentLogsEnabled": false,
        "nsgIds": [
          "\u003cocid:3\u003e"
        ],
        "privateZoneId": "\u003cocid:4\u003e",
        "sshPublicKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"
        ],
        "subnetId": "\u003cocid:5\u003e",
        "totalECpuCount": 32,
        "vmFileSystemStorageSize": 2048
      }
    }
  ],
  "characterSet": "AL32UTF8",
  "compartmentId": "\u003cocid:6\u003e",
  "databaseVersion": "23ai",
  "dbDeploymentType": "EXADB_XS",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "ddb-runtime",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "listenerPort": 1521,
  "ncharacterSet": "AL16UTF16",
  "onsPortLocal": 6234,
  "onsPortRemote": 6235,
  "prefix": "ddb123",
  "privateEndpointIds": [
    "\u003cocid:7\u003e"
  ],
  "shardDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "peerVmClusterIds": [
        "\u003cocid:8\u003e"
      ],
      "shardSpace": "PRIMARY",
      "source": "EXADB_XS",
      "vmClusterId": "\u003cocid:9\u003e"
    }
  ],
  "shardingMethod": "USER"
}`)
	createdState := ocimock.MustOCIResponseFixture[distributeddatabasesdk.DistributedDatabase](t, `{
  "catalogDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "availabilityDomain": "Uocm:PHX-AD-1",
      "dbStorageVaultDetails": {
        "highCapacityDatabaseStorage": 128
      },
      "shardSpace": "CATALOG",
      "source": "NEW_VAULT_AND_CLUSTER",
      "vmClusterDetails": {
        "backupNetworkNsgIds": [
          "\u003cocid:1\u003e"
        ],
        "backupSubnetId": "\u003cocid:2\u003e",
        "enabledECpuCount": 16,
        "nsgIds": [
          "\u003cocid:3\u003e"
        ],
        "privateZoneId": "\u003cocid:4\u003e",
        "sshPublicKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"
        ],
        "subnetId": "\u003cocid:5\u003e",
        "totalECpuCount": 32,
        "vmFileSystemStorageSize": 2048
      }
    }
  ],
  "characterSet": "AL32UTF8",
  "compartmentId": "\u003cocid:6\u003e",
  "databaseVersion": "23ai",
  "dbDeploymentType": "EXADB_XS",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "ddb-runtime",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "\u003cocid:10\u003e",
  "lifecycleState": "ACTIVE",
  "listenerPort": 1521,
  "ncharacterSet": "AL16UTF16",
  "onsPortLocal": 6234,
  "onsPortRemote": 6235,
  "prefix": "ddb123",
  "privateEndpointIds": [
    "\u003cocid:7\u003e"
  ],
  "shardDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "peerVmClusterIds": [
        "\u003cocid:8\u003e"
      ],
      "shardSpace": "PRIMARY",
      "source": "EXADB_XS",
      "vmClusterId": "\u003cocid:9\u003e"
    }
  ],
  "shardingMethod": "USER"
}`)
	createdReadStates := []distributeddatabasesdk.DistributedDatabase{
		ocimock.MustOCIResponseFixture[distributeddatabasesdk.DistributedDatabase](t, `{
  "catalogDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "availabilityDomain": "Uocm:PHX-AD-1",
      "dbStorageVaultDetails": {
        "highCapacityDatabaseStorage": 128
      },
      "shardSpace": "CATALOG",
      "source": "NEW_VAULT_AND_CLUSTER",
      "vmClusterDetails": {
        "backupNetworkNsgIds": [
          "\u003cocid:1\u003e"
        ],
        "backupSubnetId": "\u003cocid:2\u003e",
        "enabledECpuCount": 16,
        "nsgIds": [
          "\u003cocid:3\u003e"
        ],
        "privateZoneId": "\u003cocid:4\u003e",
        "sshPublicKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"
        ],
        "subnetId": "\u003cocid:5\u003e",
        "totalECpuCount": 32,
        "vmFileSystemStorageSize": 2048
      }
    }
  ],
  "characterSet": "AL32UTF8",
  "compartmentId": "\u003cocid:6\u003e",
  "databaseVersion": "23ai",
  "dbDeploymentType": "EXADB_XS",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "ddb-runtime",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "\u003cocid:10\u003e",
  "lifecycleState": "ACTIVE",
  "listenerPort": 1521,
  "ncharacterSet": "AL16UTF16",
  "onsPortLocal": 6234,
  "onsPortRemote": 6235,
  "prefix": "ddb123",
  "privateEndpointIds": [
    "\u003cocid:7\u003e"
  ],
  "shardDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "peerVmClusterIds": [
        "\u003cocid:8\u003e"
      ],
      "shardSpace": "PRIMARY",
      "source": "EXADB_XS",
      "vmClusterId": "\u003cocid:9\u003e"
    }
  ],
  "shardingMethod": "USER"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[distributeddatabasesdk.UpdateDistributedDatabaseDetails](t, `{
  "displayName": "ddb-runtime-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[distributeddatabasesdk.DistributedDatabase](t, `{
  "catalogDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "availabilityDomain": "Uocm:PHX-AD-1",
      "dbStorageVaultDetails": {
        "highCapacityDatabaseStorage": 128
      },
      "shardSpace": "CATALOG",
      "source": "NEW_VAULT_AND_CLUSTER",
      "vmClusterDetails": {
        "backupNetworkNsgIds": [
          "\u003cocid:1\u003e"
        ],
        "backupSubnetId": "\u003cocid:2\u003e",
        "enabledECpuCount": 16,
        "nsgIds": [
          "\u003cocid:3\u003e"
        ],
        "privateZoneId": "\u003cocid:4\u003e",
        "sshPublicKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"
        ],
        "subnetId": "\u003cocid:5\u003e",
        "totalECpuCount": 32,
        "vmFileSystemStorageSize": 2048
      }
    }
  ],
  "characterSet": "AL32UTF8",
  "compartmentId": "\u003cocid:6\u003e",
  "databaseVersion": "23ai",
  "dbDeploymentType": "EXADB_XS",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "ddb-runtime-updated",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "\u003cocid:10\u003e",
  "lifecycleState": "ACTIVE",
  "listenerPort": 1521,
  "ncharacterSet": "AL16UTF16",
  "onsPortLocal": 6234,
  "onsPortRemote": 6235,
  "prefix": "ddb123",
  "privateEndpointIds": [
    "\u003cocid:7\u003e"
  ],
  "shardDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "peerVmClusterIds": [
        "\u003cocid:8\u003e"
      ],
      "shardSpace": "PRIMARY",
      "source": "EXADB_XS",
      "vmClusterId": "\u003cocid:9\u003e"
    }
  ],
  "shardingMethod": "USER"
}`)
	updatedReadStates := []distributeddatabasesdk.DistributedDatabase{
		ocimock.MustOCIResponseFixture[distributeddatabasesdk.DistributedDatabase](t, `{
  "catalogDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "availabilityDomain": "Uocm:PHX-AD-1",
      "dbStorageVaultDetails": {
        "highCapacityDatabaseStorage": 128
      },
      "shardSpace": "CATALOG",
      "source": "NEW_VAULT_AND_CLUSTER",
      "vmClusterDetails": {
        "backupNetworkNsgIds": [
          "\u003cocid:1\u003e"
        ],
        "backupSubnetId": "\u003cocid:2\u003e",
        "enabledECpuCount": 16,
        "nsgIds": [
          "\u003cocid:3\u003e"
        ],
        "privateZoneId": "\u003cocid:4\u003e",
        "sshPublicKeys": [
          "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"
        ],
        "subnetId": "\u003cocid:5\u003e",
        "totalECpuCount": 32,
        "vmFileSystemStorageSize": 2048
      }
    }
  ],
  "characterSet": "AL32UTF8",
  "compartmentId": "\u003cocid:6\u003e",
  "databaseVersion": "23ai",
  "dbDeploymentType": "EXADB_XS",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "ddb-runtime-updated",
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "\u003cocid:10\u003e",
  "lifecycleState": "ACTIVE",
  "listenerPort": 1521,
  "ncharacterSet": "AL16UTF16",
  "onsPortLocal": 6234,
  "onsPortRemote": 6235,
  "prefix": "ddb123",
  "privateEndpointIds": [
    "\u003cocid:7\u003e"
  ],
  "shardDetails": [
    {
      "adminPassword": "\u003credacted\u003e",
      "peerVmClusterIds": [
        "\u003cocid:8\u003e"
      ],
      "shardSpace": "PRIMARY",
      "source": "EXADB_XS",
      "vmClusterId": "\u003cocid:9\u003e"
    }
  ],
  "shardingMethod": "USER"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		distributeddatabasesdk.DistributedDatabase,
		distributeddatabasesdk.CreateDistributedDatabaseDetails,
		distributeddatabasesdk.UpdateDistributedDatabaseDetails,
	]{
		CollectionPath:     "/20250101/distributedDatabases",
		ItemPath:           "/20250101/distributedDatabases/<ocid:10>",
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
		ValidateCreate: func(request ocimock.Request, _ distributeddatabasesdk.CreateDistributedDatabaseDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ distributeddatabasesdk.DistributedDatabase) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20250101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DistributedDatabase OCI mock: %v", err)
		}
	})
	sdkClient := distributeddatabasesdk.DistributedDbServiceClient{BaseClient: session.BaseClient()}
	manager := &DistributedDatabaseServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newDistributedDatabaseDefaultRuntimeHooks(sdkClient)
	applyDistributedDatabaseRuntimeHooks(manager, &hooks)
	client := wrapDistributedDatabaseGeneratedClient(hooks, defaultDistributedDatabaseServiceClient{ServiceClient: generatedruntime.NewServiceClient[*distributeddatabasev1beta1.DistributedDatabase](buildDistributedDatabaseGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*distributeddatabasev1beta1.DistributedDatabase]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *distributeddatabasev1beta1.DistributedDatabase) error {
			if current.Status.Id != "<ocid:10>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:10>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CharacterSet, current.Spec.CharacterSet) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DatabaseVersion, current.Spec.DatabaseVersion) ||
				!reflect.DeepEqual(current.Status.DbDeploymentType, current.Spec.DbDeploymentType) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ListenerPort, current.Spec.ListenerPort) ||
				!reflect.DeepEqual(current.Status.NcharacterSet, current.Spec.NcharacterSet) ||
				!reflect.DeepEqual(current.Status.OnsPortLocal, current.Spec.OnsPortLocal) ||
				!reflect.DeepEqual(current.Status.OnsPortRemote, current.Spec.OnsPortRemote) ||
				!reflect.DeepEqual(current.Status.Prefix, current.Spec.Prefix) ||
				!reflect.DeepEqual(current.Status.PrivateEndpointIds, current.Spec.PrivateEndpointIds) ||
				!reflect.DeepEqual(current.Status.ShardingMethod, current.Spec.ShardingMethod) ||
				len(current.Status.ShardDetails) != 1 ||
				current.Status.ShardDetails[0].Source != "EXADB_XS" ||
				current.Status.ShardDetails[0].VmClusterId != "<ocid:9>" ||
				len(current.Status.CatalogDetails) != 1 ||
				current.Status.CatalogDetails[0].Source != "NEW_VAULT_AND_CLUSTER" ||
				current.Status.CatalogDetails[0].AvailabilityDomain != "Uocm:PHX-AD-1" {
				return fmt.Errorf("created DistributedDatabase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *distributeddatabasev1beta1.DistributedDatabase) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *distributeddatabasev1beta1.DistributedDatabase) error {
			if current.Status.Id != "<ocid:10>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:10>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated DistributedDatabase status = %+v", current.Status)
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
