/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dbsystem

import (
	"context"
	"fmt"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDbSystemLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &mysqlv1beta1.DbSystem{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDbSystemName,
			Namespace: "default",
			UID:       types.UID("synthetic-dbsystem-uid"),
		},
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId:        "ocid1.compartment.oc1..mock",
			ShapeName:            "MySQL.VM.Standard.E4.1.8GB",
			SubnetId:             "ocid1.subnet.oc1..mock",
			DisplayName:          mockDbSystemName,
			Description:          "synthetic create",
			AdminUsername:        mockDbSystemUsernameSource("mysql-admin"),
			AdminPassword:        mockDbSystemPasswordSource("mysql-admin"),
			DataStorageSizeInGBs: 50,
			FreeformTags: map[string]string{
				"osok-mock": "create",
			},
		},
	}
	ocimock.InitializeResource(resource, "mock-dbsystem")
	resource.Spec.CompartmentId = "<ocid:1>"
	resource.Spec.SubnetId = "<ocid:2>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "synthetic update",
  "displayName": "osok-mock-mysql-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[mysqlsdk.CreateDbSystemDetails](t, `{
  "adminPassword": "MockPass123!",
  "adminUsername": "mockadmin",
  "compartmentId": "\u003cocid:1\u003e",
  "customerContacts": [],
  "dataStorageSizeInGBs": 50,
  "description": "synthetic create",
  "displayName": "osok-mock-mysql-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isHighlyAvailable": false,
  "nsgIds": [],
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[mysqlsdk.DbSystem](t, `{
  "availabilityDomain": "US-ASHBURN-AD-1",
  "backupPolicy": {
    "isEnabled": false,
    "pitrPolicy": {
      "isEnabled": false
    }
  },
  "compartmentId": "<ocid:1>",
  "dataStorage": {
    "dataStorageSizeInGBs": 50,
    "isAutoExpandStorageEnabled": false
  },
  "dataStorageSizeInGBs": 50,
  "deletionPolicy": {
    "isDeleteProtected": false
  },
  "description": "synthetic create",
  "displayName": "osok-mock-mysql-v1",
  "endpoints": [
    {
      "hostname": "mysql-mock.example.internal",
      "ipAddress": "10.0.0.10",
      "modes": [
        "READ",
        "WRITE"
      ],
      "port": 3306,
      "portX": 33060,
      "status": "ACTIVE"
    }
  ],
  "faultDomain": "FAULT-DOMAIN-1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hostnameLabel": "mysql-mock",
  "id": "<ocid:3>",
  "ipAddress": "10.0.0.10",
  "isHighlyAvailable": false,
  "lifecycleState": "ACTIVE",
  "mysqlVersion": "8.4.0",
  "port": 3306,
  "portX": 33060,
  "readEndpoint": {
    "isEnabled": false
  },
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:01:00Z"
}`)
	createdReadStates := []mysqlsdk.DbSystem{
		ocimock.MustOCIResponseFixture[mysqlsdk.DbSystem](t, `{
  "availabilityDomain": "US-ASHBURN-AD-1",
  "backupPolicy": {
    "isEnabled": false,
    "pitrPolicy": {
      "isEnabled": false
    }
  },
  "compartmentId": "<ocid:1>",
  "dataStorage": {
    "dataStorageSizeInGBs": 50,
    "isAutoExpandStorageEnabled": false
  },
  "dataStorageSizeInGBs": 50,
  "deletionPolicy": {
    "isDeleteProtected": false
  },
  "description": "synthetic create",
  "displayName": "osok-mock-mysql-v1",
  "endpoints": [
    {
      "hostname": "mysql-mock.example.internal",
      "ipAddress": "10.0.0.10",
      "modes": [
        "READ",
        "WRITE"
      ],
      "port": 3306,
      "portX": 33060,
      "status": "ACTIVE"
    }
  ],
  "faultDomain": "FAULT-DOMAIN-1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hostnameLabel": "mysql-mock",
  "id": "<ocid:3>",
  "ipAddress": "10.0.0.10",
  "isHighlyAvailable": false,
  "lifecycleState": "ACTIVE",
  "mysqlVersion": "8.4.0",
  "port": 3306,
  "portX": 33060,
  "readEndpoint": {
    "isEnabled": false
  },
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:01:00Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[mysqlsdk.UpdateDbSystemDetails](t, `{
  "description": "synthetic update",
  "displayName": "osok-mock-mysql-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[mysqlsdk.DbSystem](t, `{
  "availabilityDomain": "US-ASHBURN-AD-1",
  "backupPolicy": {
    "isEnabled": false,
    "pitrPolicy": {
      "isEnabled": false
    }
  },
  "compartmentId": "<ocid:1>",
  "dataStorage": {
    "dataStorageSizeInGBs": 50,
    "isAutoExpandStorageEnabled": false
  },
  "dataStorageSizeInGBs": 50,
  "deletionPolicy": {
    "isDeleteProtected": false
  },
  "description": "synthetic update",
  "displayName": "osok-mock-mysql-v1-updated",
  "endpoints": [
    {
      "hostname": "mysql-mock.example.internal",
      "ipAddress": "10.0.0.10",
      "modes": [
        "READ",
        "WRITE"
      ],
      "port": 3306,
      "portX": 33060,
      "status": "ACTIVE"
    }
  ],
  "faultDomain": "FAULT-DOMAIN-1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostnameLabel": "mysql-mock",
  "id": "<ocid:3>",
  "ipAddress": "10.0.0.10",
  "isHighlyAvailable": false,
  "lifecycleState": "ACTIVE",
  "mysqlVersion": "8.4.0",
  "port": 3306,
  "portX": 33060,
  "readEndpoint": {
    "isEnabled": false
  },
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:02:00Z"
}`)
	updatedReadStates := []mysqlsdk.DbSystem{
		ocimock.MustOCIResponseFixture[mysqlsdk.DbSystem](t, `{
  "availabilityDomain": "US-ASHBURN-AD-1",
  "backupPolicy": {
    "isEnabled": false,
    "pitrPolicy": {
      "isEnabled": false
    }
  },
  "compartmentId": "<ocid:1>",
  "dataStorage": {
    "dataStorageSizeInGBs": 50,
    "isAutoExpandStorageEnabled": false
  },
  "dataStorageSizeInGBs": 50,
  "deletionPolicy": {
    "isDeleteProtected": false
  },
  "description": "synthetic update",
  "displayName": "osok-mock-mysql-v1-updated",
  "endpoints": [
    {
      "hostname": "mysql-mock.example.internal",
      "ipAddress": "10.0.0.10",
      "modes": [
        "READ",
        "WRITE"
      ],
      "port": 3306,
      "portX": 33060,
      "status": "ACTIVE"
    }
  ],
  "faultDomain": "FAULT-DOMAIN-1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostnameLabel": "mysql-mock",
  "id": "<ocid:3>",
  "ipAddress": "10.0.0.10",
  "isHighlyAvailable": false,
  "lifecycleState": "ACTIVE",
  "mysqlVersion": "8.4.0",
  "port": 3306,
  "portX": 33060,
  "readEndpoint": {
    "isEnabled": false
  },
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:02:00Z"
}`),
	}
	deletedReadStates := []mysqlsdk.DbSystem{
		ocimock.MustOCIResponseFixture[mysqlsdk.DbSystem](t, `{
  "backupPolicy": {
    "isEnabled": false,
    "pitrPolicy": {
      "isEnabled": false
    }
  },
  "compartmentId": "<ocid:1>",
  "dataStorage": {
    "dataStorageSizeInGBs": 50,
    "isAutoExpandStorageEnabled": false
  },
  "dataStorageSizeInGBs": 50,
  "deletionPolicy": {
    "isDeleteProtected": false
  },
  "description": "synthetic update",
  "displayName": "osok-mock-mysql-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isHighlyAvailable": false,
  "lifecycleState": "DELETING",
  "mysqlVersion": "8.4.0",
  "readEndpoint": {
    "isEnabled": false
  },
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:03:00Z"
}`),
		ocimock.MustOCIResponseFixture[mysqlsdk.DbSystem](t, `{
  "backupPolicy": {
    "isEnabled": false,
    "pitrPolicy": {
      "isEnabled": false
    }
  },
  "compartmentId": "<ocid:1>",
  "dataStorage": {
    "dataStorageSizeInGBs": 50,
    "isAutoExpandStorageEnabled": false
  },
  "dataStorageSizeInGBs": 50,
  "deletionPolicy": {
    "isDeleteProtected": false
  },
  "description": "synthetic update",
  "displayName": "osok-mock-mysql-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isHighlyAvailable": false,
  "lifecycleState": "DELETED",
  "mysqlVersion": "8.4.0",
  "readEndpoint": {
    "isEnabled": false
  },
  "shapeName": "MySQL.VM.Standard.E4.1.8GB",
  "subnetId": "<ocid:2>",
  "timeCreated": "2026-08-31T12:00:00Z",
  "timeUpdated": "2026-08-31T12:04:00Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		mysqlsdk.DbSystem,
		mysqlsdk.CreateDbSystemDetails,
		mysqlsdk.UpdateDbSystemDetails,
	]{
		CollectionPath:    "/20190415/dbSystems",
		ItemPath:          "/20190415/dbSystems/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: append(ocimock.LifecycleStates(t, createdState, "CREATING", "UPDATING"), createdReadStates...),
		UpdatedReadStates: append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      204,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ mysqlsdk.CreateDbSystemDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ mysqlsdk.DbSystem) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20190415", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DbSystem OCI mock: %v", err)
		}
	})
	sdkClient := mysqlsdk.DbSystemClient{BaseClient: session.BaseClient()}
	credentials := &mockDbSystemCredentialClient{
		secrets: map[string]map[string][]byte{
			"mysql-admin": {
				"username": []byte("mockadmin"),
				"password": []byte("MockPass123!"),
			},
		},
	}
	client := newMockDbSystemClient(sdkClient, credentials)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*mysqlv1beta1.DbSystem]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *mysqlv1beta1.DbSystem) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DataStorageSizeInGBs, current.Spec.DataStorageSizeInGBs) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsHighlyAvailable, current.Spec.IsHighlyAvailable) ||
				!reflect.DeepEqual(current.Status.ShapeName, current.Spec.ShapeName) ||
				!reflect.DeepEqual(current.Status.SubnetId, current.Spec.SubnetId) {
				return fmt.Errorf("created DbSystem status = %+v", current.Status)
			}
			return validateMockDbSystemEndpointSecret(credentials, current)
		},
		Mutate: func(current *mysqlv1beta1.DbSystem) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *mysqlv1beta1.DbSystem) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated DbSystem status = %+v", current.Status)
			}
			return validateMockDbSystemEndpointSecret(credentials, current)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if credentials.hasEndpointSecret(resource.Name) {
		t.Fatal("deleted DbSystem retained its generated endpoint Secret")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func validateMockDbSystemEndpointSecret(
	credentials *mockDbSystemCredentialClient,
	resource *mysqlv1beta1.DbSystem,
) error {
	record, exists := credentials.records[resource.Name]
	if !exists {
		return fmt.Errorf("active DbSystem did not create its generated endpoint Secret")
	}
	if got := record.Labels[dbSystemEndpointSecretOwnerUIDLabel]; got != string(resource.UID) {
		return fmt.Errorf("endpoint Secret owner UID = %q, want %q", got, resource.UID)
	}
	wantData, err := dbSystemEndpointSecretData(resource)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(record.Data, wantData) {
		return fmt.Errorf("endpoint Secret data = %#v, want %#v", record.Data, wantData)
	}
	return nil
}
