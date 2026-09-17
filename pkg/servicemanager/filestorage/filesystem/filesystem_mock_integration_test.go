/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package filesystem

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
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockFileSystemID = "ocid1.filesystem.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/filestorage/filesystem.json
//   - repo-authored runtime: formal/controllers/filestorage/filesystem/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/file_storage/file_storage_file_system_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/filestorage
func TestMockIntegrationFileSystemLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &filestoragev1beta1.FileSystem{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-file-system", Namespace: "default", UID: types.UID("mock-file-system-uid")},
		Spec: filestoragev1beta1.FileSystemSpec{
			AvailabilityDomain: "mock:AD-1",
			CompartmentId:      "ocid1.compartment.oc1..mock",
			DisplayName:        "mock-file-system",
			FreeformTags:       map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newFileSystemMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://filestorage.mock.invalid", BasePath: "20171215", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close FileSystem OCI mock: %v", err)
		}
	})

	sdkClient := filestoragesdk.FileStorageClient{BaseClient: session.BaseClient()}
	manager := &FileSystemServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFileSystemRuntimeHooks(manager, sdkClient)
	client := wrapFileSystemGeneratedClient(hooks, defaultFileSystemServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*filestoragev1beta1.FileSystem](buildFileSystemGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*filestoragev1beta1.FileSystem]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *filestoragev1beta1.FileSystem) error {
			if current.Status.Id != mockFileSystemID ||
				current.Status.DisplayName != "mock-file-system" ||
				current.Status.LifecycleState != string(filestoragesdk.FileSystemLifecycleStateActive) {
				return fmt.Errorf("created FileSystem status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *filestoragev1beta1.FileSystem) {
			current.Spec.DisplayName = "mock-file-system-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *filestoragev1beta1.FileSystem) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(filestoragesdk.FileSystemLifecycleStateActive) {
				return fmt.Errorf("updated FileSystem status = %+v", current.Status)
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

func newFileSystemMockResponder(resource *filestoragev1beta1.FileSystem) (*ocimock.CRUDResponder[filestoragesdk.FileSystem], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[filestoragesdk.FileSystem]{
		CollectionPath:         "/20171215/fileSystems",
		ItemPath:               "/20171215/fileSystems/" + mockFileSystemID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (filestoragesdk.FileSystem, ocimock.Response, error) {
			var details filestoragesdk.CreateFileSystemDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.FileSystem{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return filestoragesdk.FileSystem{}, ocimock.Response{}, err
			}
			if details.AvailabilityDomain == nil || *details.AvailabilityDomain != resource.Spec.AvailabilityDomain ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.FreeformTags["osok-mock"] != "create" {
				return filestoragesdk.FileSystem{}, ocimock.Response{}, fmt.Errorf("unexpected create FileSystem details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return filestoragesdk.FileSystem{}, ocimock.Response{}, fmt.Errorf("create FileSystem opc-retry-token is empty")
			}
			state := filestoragesdk.FileSystem{
				AvailabilityDomain: details.AvailabilityDomain,
				CompartmentId:      details.CompartmentId,
				DisplayName:        details.DisplayName,
				FreeformTags:       details.FreeformTags,
				Id:                 common.String(mockFileSystemID),
				TimeCreated:        &createdAt,
				LifecycleState:     filestoragesdk.FileSystemLifecycleStateCreating,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state filestoragesdk.FileSystem) (filestoragesdk.FileSystem, ocimock.Response, error) {
			switch state.LifecycleState {
			case filestoragesdk.FileSystemLifecycleStateCreating, filestoragesdk.FileSystemLifecycleStateUpdating:
				state.LifecycleState = filestoragesdk.FileSystemLifecycleStateActive
			case filestoragesdk.FileSystemLifecycleStateDeleting:
				state.LifecycleState = filestoragesdk.FileSystemLifecycleStateDeleted
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state filestoragesdk.FileSystem) (filestoragesdk.FileSystem, ocimock.Response, error) {
			var details filestoragesdk.UpdateFileSystemDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.FileSystem{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-file-system-updated" ||
				details.FreeformTags["osok-mock"] != "update" {
				return filestoragesdk.FileSystem{}, ocimock.Response{}, fmt.Errorf("unexpected update FileSystem details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = filestoragesdk.FileSystemLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state filestoragesdk.FileSystem) (filestoragesdk.FileSystem, ocimock.Response, error) {
			if state.LifecycleState == filestoragesdk.FileSystemLifecycleStateDeleted {
				response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
				return state, response, err
			}
			state.LifecycleState = filestoragesdk.FileSystemLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
