/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package export

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

const mockExportID = "ocid1.export.oc1..mock"

// Contract evidence: recorded Export CRUD, resource-local mutation semantics, OCI SDK, and pinned provider.
func TestMockIntegrationExportLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &filestoragev1beta1.Export{ObjectMeta: metav1.ObjectMeta{Name: "mock-export", Namespace: "default", UID: types.UID("mock-export-uid")}, Spec: filestoragev1beta1.ExportSpec{
		ExportSetId: "ocid1.exportset.oc1..mock", FileSystemId: "ocid1.filesystem.oc1..mock", Path: "/mock-export",
		ExportOptions: []filestoragev1beta1.ExportOption{{Source: "10.0.0.0/16", RequirePrivilegedSourcePort: false, Access: "READ_WRITE", IdentitySquash: "NONE", AnonymousUid: 65534, AnonymousGid: 65534, AllowedAuth: []string{"SYS"}}},
	}}
	responder, err := newExportMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://filestorage.mock.invalid", BasePath: "20171215", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Export OCI mock: %v", err)
		}
	})
	client := newMockExportClient(filestoragesdk.FileStorageClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*filestoragev1beta1.Export]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *filestoragev1beta1.Export) error {
			if current.Status.Id != mockExportID || current.Status.Path != resource.Spec.Path || current.Status.LifecycleState != string(filestoragesdk.ExportLifecycleStateActive) {
				return fmt.Errorf("created Export status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *filestoragev1beta1.Export) { current.Spec.ExportOptions[0].Access = "READ_ONLY" },
		ValidateUpdated: func(current *filestoragev1beta1.Export) error {
			if len(current.Status.ExportOptions) != 1 || current.Status.ExportOptions[0].Access != "READ_ONLY" {
				return fmt.Errorf("updated Export status = %+v", current.Status)
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

func newExportMockResponder(resource *filestoragev1beta1.Export) (*ocimock.CRUDResponder[filestoragesdk.Export], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, deleteRead := false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[filestoragesdk.Export]{
		CollectionPath: "/20171215/exports", ItemPath: "/20171215/exports/" + mockExportID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (filestoragesdk.Export, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero filestoragesdk.Export
				return zero, ocimock.Response{}, err
			}
			var details filestoragesdk.CreateExportDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.Export{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return filestoragesdk.Export{}, ocimock.Response{}, err
			}
			if details.ExportSetId == nil || *details.ExportSetId != resource.Spec.ExportSetId || details.FileSystemId == nil || *details.FileSystemId != resource.Spec.FileSystemId || details.Path == nil || *details.Path != resource.Spec.Path || len(details.ExportOptions) != 1 || details.ExportOptions[0].Access != filestoragesdk.ClientOptionsAccessWrite {
				return filestoragesdk.Export{}, ocimock.Response{}, fmt.Errorf("unexpected CreateExport details: %+v", details)
			}
			state := filestoragesdk.Export{ExportOptions: details.ExportOptions, ExportSetId: details.ExportSetId, FileSystemId: details.FileSystemId,
				Id: common.String(mockExportID), LifecycleState: filestoragesdk.ExportLifecycleStateCreating, Path: details.Path, TimeCreated: &now, IsIdmapGroupsForSysAuth: details.IsIdmapGroupsForSysAuth}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state filestoragesdk.Export) (filestoragesdk.Export, ocimock.Response, error) {
			switch state.LifecycleState {
			case filestoragesdk.ExportLifecycleStateCreating:
				if createRead {
					state.LifecycleState = filestoragesdk.ExportLifecycleStateActive
				} else {
					createRead = true
				}
			case filestoragesdk.ExportLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = filestoragesdk.ExportLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state filestoragesdk.Export) (filestoragesdk.Export, ocimock.Response, error) {
			var details filestoragesdk.UpdateExportDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.Export{}, ocimock.Response{}, err
			}
			if len(details.ExportOptions) != 1 {
				return filestoragesdk.Export{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateExport details: %+v", details)
			}
			switch details.ExportOptions[0].Access {
			case filestoragesdk.ClientOptionsAccessWrite:
				// OCI normalizes omitted ClientOptions booleans to explicit false;
				// the runtime performs one convergence update before steady state.
			case filestoragesdk.ClientOptionsAccessOnly:
			default:
				return filestoragesdk.Export{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateExport access: %+v", details)
			}
			state.ExportOptions = details.ExportOptions
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state filestoragesdk.Export) (filestoragesdk.Export, ocimock.Response, error) {
			state.LifecycleState = filestoragesdk.ExportLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
