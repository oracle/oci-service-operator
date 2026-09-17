/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package mounttarget

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

const mockMountTargetID = "ocid1.mounttarget.oc1..mock"

// Contract evidence: recorded MountTarget CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationMountTargetLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &filestoragev1beta1.MountTarget{ObjectMeta: metav1.ObjectMeta{Name: "mock-mount-target", Namespace: "default", UID: types.UID("mock-mount-target-uid")}, Spec: filestoragev1beta1.MountTargetSpec{
		AvailabilityDomain: "example:US-ASHBURN-AD-1", CompartmentId: "ocid1.compartment.oc1..mock", SubnetId: "ocid1.subnet.oc1..mock",
		DisplayName: "mock-mount-target", HostnameLabel: "mockmt", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newMountTargetMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://filestorage.mock.invalid", BasePath: "20171215", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MountTarget OCI mock: %v", err)
		}
	})
	client := newMockMountTargetClient(filestoragesdk.FileStorageClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*filestoragev1beta1.MountTarget]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *filestoragev1beta1.MountTarget) error {
			if current.Status.Id != mockMountTargetID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.ExportSetId == "" || current.Status.LifecycleState != string(filestoragesdk.MountTargetLifecycleStateActive) {
				return fmt.Errorf("created MountTarget status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *filestoragev1beta1.MountTarget) {
			current.Spec.DisplayName = "mock-mount-target-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *filestoragev1beta1.MountTarget) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated MountTarget status = %+v", current.Status)
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

func newMountTargetMockResponder(resource *filestoragev1beta1.MountTarget) (*ocimock.CRUDResponder[filestoragesdk.MountTarget], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, deleteRead := false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[filestoragesdk.MountTarget]{
		CollectionPath: "/20171215/mountTargets", ItemPath: "/20171215/mountTargets/" + mockMountTargetID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (filestoragesdk.MountTarget, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero filestoragesdk.MountTarget
				return zero, ocimock.Response{}, err
			}
			var details filestoragesdk.CreateMountTargetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.MountTarget{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return filestoragesdk.MountTarget{}, ocimock.Response{}, err
			}
			if details.AvailabilityDomain == nil || *details.AvailabilityDomain != resource.Spec.AvailabilityDomain || details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.SubnetId == nil || *details.SubnetId != resource.Spec.SubnetId || details.HostnameLabel == nil || *details.HostnameLabel != resource.Spec.HostnameLabel {
				return filestoragesdk.MountTarget{}, ocimock.Response{}, fmt.Errorf("unexpected CreateMountTarget details: %+v", details)
			}
			state := filestoragesdk.MountTarget{CompartmentId: details.CompartmentId, DisplayName: details.DisplayName, Id: common.String(mockMountTargetID),
				LifecycleDetails: common.String(""), LifecycleState: filestoragesdk.MountTargetLifecycleStateCreating, PrivateIpIds: []string{"ocid1.privateip.oc1..mock"},
				SubnetId: details.SubnetId, TimeCreated: &now, AvailabilityDomain: details.AvailabilityDomain, ExportSetId: common.String("ocid1.exportset.oc1..mock"),
				IdmapType: details.IdmapType, NsgIds: details.NsgIds, FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{}, SecurityAttributes: details.SecurityAttributes}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state filestoragesdk.MountTarget) (filestoragesdk.MountTarget, ocimock.Response, error) {
			switch state.LifecycleState {
			case filestoragesdk.MountTargetLifecycleStateCreating:
				if createRead {
					state.LifecycleState = filestoragesdk.MountTargetLifecycleStateActive
				} else {
					createRead = true
				}
			case filestoragesdk.MountTargetLifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = filestoragesdk.MountTargetLifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state filestoragesdk.MountTarget) (filestoragesdk.MountTarget, ocimock.Response, error) {
			var details filestoragesdk.UpdateMountTargetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return filestoragesdk.MountTarget{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-mount-target-updated" || details.FreeformTags["osok-mock"] != "update" {
				return filestoragesdk.MountTarget{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateMountTarget details: %+v", details)
			}
			state.DisplayName, state.FreeformTags = details.DisplayName, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state filestoragesdk.MountTarget) (filestoragesdk.MountTarget, ocimock.Response, error) {
			state.LifecycleState = filestoragesdk.MountTargetLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
