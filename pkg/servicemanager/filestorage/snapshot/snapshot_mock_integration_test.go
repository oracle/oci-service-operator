/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package snapshot

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockSnapshotID = "ocid1.snapshot.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/filestorage/snapshot.json
//   - repo-authored runtime: formal/controllers/filestorage/snapshot/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/file_storage/file_storage_snapshot_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/filestorage
func TestMockIntegrationSnapshotLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &filestoragev1beta1.Snapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-snapshot", Namespace: "default", UID: types.UID("mock-snapshot-uid")},
		Spec: filestoragev1beta1.SnapshotSpec{
			FileSystemId: "ocid1.filesystem.oc1..mock",
			Name:         "mock-snapshot",
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newSnapshotMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://filestorage.mock.invalid", BasePath: "20171215", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Snapshot OCI mock: %v", err)
		}
	})

	sdkClient := filestoragesdk.FileStorageClient{BaseClient: session.BaseClient()}
	manager := &SnapshotServiceManager{}
	hooks := newSnapshotRuntimeHooks(manager, sdkClient)
	client := wrapSnapshotGeneratedClient(hooks, defaultSnapshotServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*filestoragev1beta1.Snapshot](buildSnapshotGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*filestoragev1beta1.Snapshot]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *filestoragev1beta1.Snapshot) error {
			if current.Status.Id != mockSnapshotID ||
				current.Status.Name != "mock-snapshot" ||
				current.Status.LifecycleState != string(filestoragesdk.SnapshotLifecycleStateActive) {
				return fmt.Errorf("created Snapshot status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *filestoragev1beta1.Snapshot) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *filestoragev1beta1.Snapshot) error {
			if current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(filestoragesdk.SnapshotLifecycleStateActive) {
				return fmt.Errorf("updated Snapshot status = %+v", current.Status)
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

func newSnapshotMockResponder(resource *filestoragev1beta1.Snapshot) (*ocimock.CRUDResponder[filestoragesdk.Snapshot], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[filestoragesdk.Snapshot]{
		CollectionPath:         "/20171215/snapshots",
		ItemPath:               "/20171215/snapshots/" + mockSnapshotID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (filestoragesdk.Snapshot, ocimock.Response, error) {
			var details filestoragesdk.CreateSnapshotDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.Snapshot{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return filestoragesdk.Snapshot{}, ocimock.Response{}, err
			}
			if details.FileSystemId == nil || *details.FileSystemId != resource.Spec.FileSystemId ||
				details.Name == nil || *details.Name != resource.Spec.Name ||
				details.FreeformTags["osok-mock"] != "create" {
				return filestoragesdk.Snapshot{}, ocimock.Response{}, fmt.Errorf("unexpected create Snapshot details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return filestoragesdk.Snapshot{}, ocimock.Response{}, fmt.Errorf("create Snapshot opc-retry-token is empty")
			}
			state := filestoragesdk.Snapshot{
				FileSystemId:   details.FileSystemId,
				Name:           details.Name,
				FreeformTags:   details.FreeformTags,
				Id:             common.String(mockSnapshotID),
				TimeCreated:    &createdAt,
				SnapshotTime:   &createdAt,
				SnapshotType:   filestoragesdk.SnapshotSnapshotTypeUser,
				LifecycleState: filestoragesdk.SnapshotLifecycleStateCreating,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state filestoragesdk.Snapshot) (filestoragesdk.Snapshot, ocimock.Response, error) {
			switch state.LifecycleState {
			case filestoragesdk.SnapshotLifecycleStateCreating:
				state.LifecycleState = filestoragesdk.SnapshotLifecycleStateActive
			case filestoragesdk.SnapshotLifecycleStateDeleting:
				state.LifecycleState = filestoragesdk.SnapshotLifecycleStateDeleted
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state filestoragesdk.Snapshot) (filestoragesdk.Snapshot, ocimock.Response, error) {
			var details filestoragesdk.UpdateSnapshotDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.Snapshot{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" {
				return filestoragesdk.Snapshot{}, ocimock.Response{}, fmt.Errorf("unexpected update Snapshot details: %+v", details)
			}
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state filestoragesdk.Snapshot) (filestoragesdk.Snapshot, ocimock.Response, error) {
			if state.LifecycleState == filestoragesdk.SnapshotLifecycleStateDeleted {
				response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
				return state, response, err
			}
			state.LifecycleState = filestoragesdk.SnapshotLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
