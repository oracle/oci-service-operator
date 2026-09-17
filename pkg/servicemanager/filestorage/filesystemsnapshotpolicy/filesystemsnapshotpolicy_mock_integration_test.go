/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package filesystemsnapshotpolicy

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockFilesystemSnapshotPolicyID = "ocid1.filesystemsnapshotpolicy.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/filestorage/filesystemsnapshotpolicy.json
//   - repo-authored runtime: formal/controllers/filestorage/filesystemsnapshotpolicy/diagrams/runtime-lifecycle.yaml
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/file_storage/file_storage_filesystem_snapshot_policy_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/filestorage
func TestMockIntegrationFilesystemSnapshotPolicyLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &filestoragev1beta1.FilesystemSnapshotPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-snapshot-policy", Namespace: "default", UID: types.UID("mock-snapshot-policy-uid")},
		Spec: filestoragev1beta1.FilesystemSnapshotPolicySpec{
			AvailabilityDomain: "mock:AD-1",
			CompartmentId:      "ocid1.compartment.oc1..mock",
			DisplayName:        "mock-snapshot-policy",
			PolicyPrefix:       "mock",
			FreeformTags:       map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newFilesystemSnapshotPolicyMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://filestorage.mock.invalid", BasePath: "20171215", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close FilesystemSnapshotPolicy OCI mock: %v", err)
		}
	})

	sdkClient := filestoragesdk.FileStorageClient{BaseClient: session.BaseClient()}
	manager := &FilesystemSnapshotPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newFilesystemSnapshotPolicyRuntimeHooks(manager, sdkClient)
	client := wrapFilesystemSnapshotPolicyGeneratedClient(hooks, defaultFilesystemSnapshotPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*filestoragev1beta1.FilesystemSnapshotPolicy](buildFilesystemSnapshotPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*filestoragev1beta1.FilesystemSnapshotPolicy]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *filestoragev1beta1.FilesystemSnapshotPolicy) error {
			if current.Status.Id != mockFilesystemSnapshotPolicyID ||
				current.Status.DisplayName != "mock-snapshot-policy" ||
				current.Status.PolicyPrefix != "mock" ||
				current.Status.LifecycleState != string(filestoragesdk.FilesystemSnapshotPolicyLifecycleStateActive) {
				return fmt.Errorf("created FilesystemSnapshotPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *filestoragev1beta1.FilesystemSnapshotPolicy) {
			current.Spec.DisplayName = "mock-snapshot-policy-updated"
			current.Spec.PolicyPrefix = "updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *filestoragev1beta1.FilesystemSnapshotPolicy) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.PolicyPrefix != current.Spec.PolicyPrefix ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated FilesystemSnapshotPolicy status = %+v", current.Status)
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

func newFilesystemSnapshotPolicyMockResponder(resource *filestoragev1beta1.FilesystemSnapshotPolicy) (*ocimock.CRUDResponder[filestoragesdk.FilesystemSnapshotPolicy], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[filestoragesdk.FilesystemSnapshotPolicy]{
		CollectionPath:         "/20171215/filesystemSnapshotPolicies",
		ItemPath:               "/20171215/filesystemSnapshotPolicies/" + mockFilesystemSnapshotPolicyID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (filestoragesdk.FilesystemSnapshotPolicy, ocimock.Response, error) {
			var details filestoragesdk.CreateFilesystemSnapshotPolicyDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.FilesystemSnapshotPolicy{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return filestoragesdk.FilesystemSnapshotPolicy{}, ocimock.Response{}, err
			}
			if details.AvailabilityDomain == nil || *details.AvailabilityDomain != resource.Spec.AvailabilityDomain ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.PolicyPrefix == nil || *details.PolicyPrefix != resource.Spec.PolicyPrefix {
				return filestoragesdk.FilesystemSnapshotPolicy{}, ocimock.Response{}, fmt.Errorf("unexpected create FilesystemSnapshotPolicy details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return filestoragesdk.FilesystemSnapshotPolicy{}, ocimock.Response{}, fmt.Errorf("create FilesystemSnapshotPolicy opc-retry-token is empty")
			}
			state := filestoragesdk.FilesystemSnapshotPolicy{
				AvailabilityDomain: details.AvailabilityDomain,
				CompartmentId:      details.CompartmentId,
				DisplayName:        details.DisplayName,
				PolicyPrefix:       details.PolicyPrefix,
				FreeformTags:       details.FreeformTags,
				Id:                 common.String(mockFilesystemSnapshotPolicyID),
				TimeCreated:        &createdAt,
				LifecycleState:     filestoragesdk.FilesystemSnapshotPolicyLifecycleStateCreating,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state filestoragesdk.FilesystemSnapshotPolicy) (filestoragesdk.FilesystemSnapshotPolicy, ocimock.Response, error) {
			switch state.LifecycleState {
			case filestoragesdk.FilesystemSnapshotPolicyLifecycleStateCreating:
				state.LifecycleState = filestoragesdk.FilesystemSnapshotPolicyLifecycleStateActive
			case filestoragesdk.FilesystemSnapshotPolicyLifecycleStateDeleting:
				state.LifecycleState = filestoragesdk.FilesystemSnapshotPolicyLifecycleStateDeleted
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state filestoragesdk.FilesystemSnapshotPolicy) (filestoragesdk.FilesystemSnapshotPolicy, ocimock.Response, error) {
			var details filestoragesdk.UpdateFilesystemSnapshotPolicyDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.FilesystemSnapshotPolicy{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-snapshot-policy-updated" ||
				details.PolicyPrefix == nil || *details.PolicyPrefix != "updated" ||
				details.FreeformTags["osok-mock"] != "update" {
				return filestoragesdk.FilesystemSnapshotPolicy{}, ocimock.Response{}, fmt.Errorf("unexpected update FilesystemSnapshotPolicy details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.PolicyPrefix = details.PolicyPrefix
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state filestoragesdk.FilesystemSnapshotPolicy) (filestoragesdk.FilesystemSnapshotPolicy, ocimock.Response, error) {
			if state.LifecycleState == filestoragesdk.FilesystemSnapshotPolicyLifecycleStateDeleted {
				response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
				return state, response, err
			}
			state.LifecycleState = filestoragesdk.FilesystemSnapshotPolicyLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
