/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedlist

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockManagedListID = "ocid1.cloudguardmanagedlist.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal managed-list contract, and vendored OCI SDK.
func TestMockIntegrationManagedListLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &cloudguardv1beta1.ManagedList{ObjectMeta: metav1.ObjectMeta{Name: "mock-managed-list", Namespace: "default", UID: types.UID("mock-managed-list-uid")}, Spec: cloudguardv1beta1.ManagedListSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-managed-list", Description: "mock create",
		ListType: "CIDR_BLOCK", ListItems: []string{"192.0.2.0/24"}, FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newManagedListMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard.mock.invalid", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &ManagedListServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newManagedListRuntimeHooks(manager, cloudguardsdk.CloudGuardClient{BaseClient: session.BaseClient()})
	client := wrapManagedListGeneratedClient(hooks, defaultManagedListServiceClient{ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.ManagedList](buildManagedListGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.ManagedList]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.ManagedList) error {
			if current.Status.Id != mockManagedListID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(cloudguardsdk.LifecycleStateActive) {
				return fmt.Errorf("created ManagedList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.ManagedList) {
			current.Spec.Description = "mock update"
			current.Spec.ListItems = []string{"198.51.100.0/24"}
		},
		ValidateUpdated: func(current *cloudguardv1beta1.ManagedList) error {
			if current.Status.Description != current.Spec.Description || len(current.Status.ListItems) != 1 || current.Status.ListItems[0] != "198.51.100.0/24" {
				return fmt.Errorf("updated ManagedList status = %+v", current.Status)
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

func newManagedListMockResponder(resource *cloudguardv1beta1.ManagedList) (*ocimock.CRUDResponder[cloudguardsdk.ManagedList], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, updateRead, deleteRead := false, false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[cloudguardsdk.ManagedList]{
		CollectionPath: "/20200131/managedLists", ItemPath: "/20200131/managedLists/" + mockManagedListID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (cloudguardsdk.ManagedList, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero cloudguardsdk.ManagedList
				return zero, ocimock.Response{}, err
			}
			var details cloudguardsdk.CreateManagedListDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudguardsdk.ManagedList{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return cloudguardsdk.ManagedList{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.ListType != cloudguardsdk.ManagedListTypeCidrBlock || len(details.ListItems) != 1 {
				return cloudguardsdk.ManagedList{}, ocimock.Response{}, fmt.Errorf("unexpected CreateManagedList details: %+v", details)
			}
			state := cloudguardsdk.ManagedList{Id: common.String(mockManagedListID), DisplayName: details.DisplayName, CompartmentId: details.CompartmentId,
				ListType: details.ListType, Description: details.Description, ListItems: details.ListItems, IsEditable: common.Bool(true),
				TimeCreated: &now, TimeUpdated: &now, LifecycleState: cloudguardsdk.LifecycleStateCreating, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state cloudguardsdk.ManagedList) (cloudguardsdk.ManagedList, ocimock.Response, error) {
			switch state.LifecycleState {
			case cloudguardsdk.LifecycleStateCreating:
				if createRead {
					state.LifecycleState = cloudguardsdk.LifecycleStateActive
				} else {
					createRead = true
				}
			case cloudguardsdk.LifecycleStateUpdating:
				if updateRead {
					state.LifecycleState = cloudguardsdk.LifecycleStateActive
				} else {
					updateRead = true
				}
			case cloudguardsdk.LifecycleStateDeleting:
				if deleteRead {
					state.LifecycleState = cloudguardsdk.LifecycleStateDeleted
				} else {
					deleteRead = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state cloudguardsdk.ManagedList) (cloudguardsdk.ManagedList, ocimock.Response, error) {
			var details cloudguardsdk.UpdateManagedListDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return cloudguardsdk.ManagedList{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || len(details.ListItems) != 1 || details.ListItems[0] != "198.51.100.0/24" || details.Group != nil {
				return cloudguardsdk.ManagedList{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateManagedList details: %+v", details)
			}
			state.Description, state.ListItems, state.LifecycleState = details.Description, details.ListItems, cloudguardsdk.LifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state cloudguardsdk.ManagedList) (cloudguardsdk.ManagedList, ocimock.Response, error) {
			state.LifecycleState = cloudguardsdk.LifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
