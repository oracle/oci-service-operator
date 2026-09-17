/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stack

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourcemanagersdk "github.com/oracle/oci-go-sdk/v65/resourcemanager"
	resourcemanagerv1beta1 "github.com/oracle/oci-service-operator/api/resourcemanager/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockStackID = "ocid1.ormstack.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/resourcemanager
//   - reviewed runtime semantics: stack_runtime_semantics.go
//
// The pinned Terraform provider exposes Stack data sources but no Stack
// resource, so it is not used as mutation or lifecycle evidence here.
func TestMockIntegrationStackLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &resourcemanagerv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-stack", Namespace: "default", UID: types.UID("mock-stack-uid")},
		Spec: resourcemanagerv1beta1.StackSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			ConfigSource: resourcemanagerv1beta1.StackConfigSource{
				ConfigSourceType:     "ZIP_UPLOAD",
				ZipFileBase64Encoded: mockStackZip(t),
			},
			DisplayName:  "mock-stack",
			Description:  "mock create",
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newStackMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://resourcemanager.mock.invalid", BasePath: "20180917", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Stack OCI mock: %v", err)
		}
	})

	sdkClient := resourcemanagersdk.ResourceManagerClient{BaseClient: session.BaseClient()}
	client := newMockStackClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*resourcemanagerv1beta1.Stack]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *resourcemanagerv1beta1.Stack) error {
			if current.Status.Id != mockStackID ||
				current.Status.DisplayName != "mock-stack" ||
				current.Status.LifecycleState != string(resourcemanagersdk.StackLifecycleStateActive) {
				return fmt.Errorf("created Stack status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *resourcemanagerv1beta1.Stack) {
			current.Spec.DisplayName = "mock-stack-updated"
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *resourcemanagerv1beta1.Stack) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.Description != current.Spec.Description ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Stack status = %+v", current.Status)
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

func newStackMockResponder(resource *resourcemanagerv1beta1.Stack) (*ocimock.CRUDResponder[resourcemanagersdk.Stack], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[resourcemanagersdk.Stack]{
		CollectionPath:         "/20180917/stacks",
		ItemPath:               "/20180917/stacks/" + mockStackID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (resourcemanagersdk.Stack, ocimock.Response, error) {
			var details resourcemanagersdk.CreateStackDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return resourcemanagersdk.Stack{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return resourcemanagersdk.Stack{}, ocimock.Response{}, err
			}
			source, ok := details.ConfigSource.(resourcemanagersdk.CreateZipUploadConfigSourceDetails)
			if !ok || source.ZipFileBase64Encoded == nil || *source.ZipFileBase64Encoded != resource.Spec.ConfigSource.ZipFileBase64Encoded ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return resourcemanagersdk.Stack{}, ocimock.Response{}, fmt.Errorf("unexpected create Stack details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return resourcemanagersdk.Stack{}, ocimock.Response{}, fmt.Errorf("create Stack opc-retry-token is empty")
			}
			state := resourcemanagersdk.Stack{
				Id:             common.String(mockStackID),
				CompartmentId:  details.CompartmentId,
				DisplayName:    details.DisplayName,
				Description:    details.Description,
				ConfigSource:   resourcemanagersdk.ZipUploadConfigSource{},
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: resourcemanagersdk.StackLifecycleStateCreating,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state resourcemanagersdk.Stack) (resourcemanagersdk.Stack, ocimock.Response, error) {
			switch state.LifecycleState {
			case resourcemanagersdk.StackLifecycleStateCreating:
				state.LifecycleState = resourcemanagersdk.StackLifecycleStateActive
			case resourcemanagersdk.StackLifecycleStateDeleting:
				state.LifecycleState = resourcemanagersdk.StackLifecycleStateDeleted
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state resourcemanagersdk.Stack) (resourcemanagersdk.Stack, ocimock.Response, error) {
			var details resourcemanagersdk.UpdateStackDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return resourcemanagersdk.Stack{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-stack-updated" ||
				details.Description == nil || *details.Description != "mock update" ||
				details.FreeformTags["osok-mock"] != "update" {
				return resourcemanagersdk.Stack{}, ocimock.Response{}, fmt.Errorf("unexpected update Stack details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.Description = details.Description
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state resourcemanagersdk.Stack) (resourcemanagersdk.Stack, ocimock.Response, error) {
			state.LifecycleState = resourcemanagersdk.StackLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
