/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package managedinstancegroup

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockManagedInstanceGroupID = "ocid1.osmhmanagedinstancegroup.oc1..mock"

// Contract evidence: recorded group CRUD, membership-aware runtime, OCI SDK, and pinned provider.
func TestMockIntegrationManagedInstanceGroupLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &osmanagementhubv1beta1.ManagedInstanceGroup{ObjectMeta: metav1.ObjectMeta{Name: "mock-managed-instance-group", Namespace: "default", UID: types.UID("mock-managed-instance-group-uid")}, Spec: osmanagementhubv1beta1.ManagedInstanceGroupSpec{
		DisplayName: "mock-managed-instance-group", CompartmentId: "ocid1.compartment.oc1..mock", OsFamily: "ORACLE_LINUX_8", ArchType: "X86_64", VendorName: "ORACLE",
		Description: "mock create", Location: "OCI_COMPUTE", SoftwareSourceIds: []string{"ocid1.osmhsoftwaresource.oc1..mock"},
	}}
	responder, err := newManagedInstanceGroupMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://osmh.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ManagedInstanceGroup OCI mock: %v", err)
		}
	})
	client := newManagedInstanceGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{}, osmanagementhubsdk.ManagedInstanceGroupClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*osmanagementhubv1beta1.ManagedInstanceGroup]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *osmanagementhubv1beta1.ManagedInstanceGroup) error {
			if current.Status.Id != mockManagedInstanceGroupID || current.Status.Description != resource.Spec.Description || current.Status.LifecycleState != string(osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive) {
				return fmt.Errorf("created ManagedInstanceGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *osmanagementhubv1beta1.ManagedInstanceGroup) { current.Spec.Description = "mock update" },
		ValidateUpdated: func(current *osmanagementhubv1beta1.ManagedInstanceGroup) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated ManagedInstanceGroup status = %+v", current.Status)
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

func newManagedInstanceGroupMockResponder(resource *osmanagementhubv1beta1.ManagedInstanceGroup) (*ocimock.CRUDResponder[osmanagementhubsdk.ManagedInstanceGroup], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[osmanagementhubsdk.ManagedInstanceGroup]{
		CollectionPath: "/20220901/managedInstanceGroups", ItemPath: "/20220901/managedInstanceGroups/" + mockManagedInstanceGroupID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state osmanagementhubsdk.ManagedInstanceGroup) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.ManagedInstanceGroup{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.ManagedInstanceGroup{state}})
		},
		Create: func(request ocimock.Request) (osmanagementhubsdk.ManagedInstanceGroup, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero osmanagementhubsdk.ManagedInstanceGroup
				return zero, ocimock.Response{}, err
			}
			var details osmanagementhubsdk.CreateManagedInstanceGroupDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return osmanagementhubsdk.ManagedInstanceGroup{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return osmanagementhubsdk.ManagedInstanceGroup{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || len(details.SoftwareSourceIds) != 1 || details.SoftwareSourceIds[0] != resource.Spec.SoftwareSourceIds[0] {
				return osmanagementhubsdk.ManagedInstanceGroup{}, ocimock.Response{}, fmt.Errorf("unexpected CreateManagedInstanceGroup details: %+v", details)
			}
			sources := []osmanagementhubsdk.SoftwareSourceDetails{{Id: common.String(details.SoftwareSourceIds[0])}}
			state := osmanagementhubsdk.ManagedInstanceGroup{Id: common.String(mockManagedInstanceGroupID), CompartmentId: details.CompartmentId, LifecycleState: osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive,
				DisplayName: details.DisplayName, Description: details.Description, TimeCreated: &now, TimeModified: &now, OsFamily: details.OsFamily, ArchType: details.ArchType,
				VendorName: details.VendorName, SoftwareSourceIds: sources, SoftwareSources: sources, ManagedInstanceIds: details.ManagedInstanceIds, ManagedInstanceCount: common.Int(0),
				Location: details.Location, PendingJobCount: common.Int(0), NotificationTopicId: details.NotificationTopicId, FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags}
			response, err := ocimock.JSONResponse(http.StatusAccepted, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state osmanagementhubsdk.ManagedInstanceGroup) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state osmanagementhubsdk.ManagedInstanceGroup) (osmanagementhubsdk.ManagedInstanceGroup, ocimock.Response, error) {
			var details osmanagementhubsdk.UpdateManagedInstanceGroupDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return osmanagementhubsdk.ManagedInstanceGroup{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.DisplayName != nil {
				return osmanagementhubsdk.ManagedInstanceGroup{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateManagedInstanceGroup details: %+v", details)
			}
			state.Description, state.TimeModified = details.Description, &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ osmanagementhubsdk.ManagedInstanceGroup) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
