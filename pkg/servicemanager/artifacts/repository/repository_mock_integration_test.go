/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package repository

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockRepositoryID = "ocid1.artifactrepository.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - repo-authored runtime: repository_runtime_client.go
//   - OCI SDK and pinned Terraform provider ArtifactsRepository resource
func TestMockIntegrationRepositoryLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &artifactsv1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-repository", Namespace: "default", UID: types.UID("mock-repository-uid")},
		Spec: artifactsv1beta1.RepositorySpec{
			CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-repository", Description: "mock create",
			RepositoryType: "GENERIC", FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newRepositoryMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://artifacts.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Repository OCI mock: %v", err)
		}
	})
	client := newRepositoryServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, artifactssdk.ArtifactsClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*artifactsv1beta1.Repository]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *artifactsv1beta1.Repository) error {
			if current.Status.Id != mockRepositoryID || current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.LifecycleState != string(artifactssdk.RepositoryLifecycleStateAvailable) {
				return fmt.Errorf("created Repository status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *artifactsv1beta1.Repository) {
			current.Spec.DisplayName = "mock-repository-updated"
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *artifactsv1beta1.Repository) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Repository status = %+v", current.Status)
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

func newRepositoryMockResponder(resource *artifactsv1beta1.Repository) (*ocimock.CRUDResponder[artifactssdk.GenericRepository], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[artifactssdk.GenericRepository]{
		CollectionPath: "/20160918/repositories", ItemPath: "/20160918/repositories/" + mockRepositoryID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (artifactssdk.GenericRepository, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero artifactssdk.GenericRepository
				return zero, ocimock.Response{}, err
			}
			var envelope struct {
				artifactssdk.CreateGenericRepositoryDetails
				RepositoryType string `json:"repositoryType"`
			}
			if err := ocimock.DecodeJSONRequest(request, &envelope); err != nil {
				return artifactssdk.GenericRepository{}, ocimock.Response{}, err
			}
			details := envelope.CreateGenericRepositoryDetails
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return artifactssdk.GenericRepository{}, ocimock.Response{}, err
			}
			if envelope.RepositoryType != "GENERIC" || details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.IsImmutable == nil || *details.IsImmutable {
				return artifactssdk.GenericRepository{}, ocimock.Response{}, fmt.Errorf("unexpected CreateRepository details: %+v", envelope)
			}
			state := artifactssdk.GenericRepository{Id: common.String(mockRepositoryID), DisplayName: details.DisplayName, CompartmentId: details.CompartmentId,
				Description: details.Description, IsImmutable: details.IsImmutable, FreeformTags: details.FreeformTags,
				DefinedTags: map[string]map[string]interface{}{}, TimeCreated: &createdAt, LifecycleState: artifactssdk.RepositoryLifecycleStateAvailable}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state artifactssdk.GenericRepository) (artifactssdk.GenericRepository, ocimock.Response, error) {
			if state.LifecycleState == artifactssdk.RepositoryLifecycleStateDeleting {
				if deleteReadObserved {
					state.LifecycleState = artifactssdk.RepositoryLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state artifactssdk.GenericRepository) (artifactssdk.GenericRepository, ocimock.Response, error) {
			var envelope struct {
				artifactssdk.UpdateGenericRepositoryDetails
				RepositoryType string `json:"repositoryType"`
			}
			if err := ocimock.DecodeJSONRequest(request, &envelope); err != nil {
				return artifactssdk.GenericRepository{}, ocimock.Response{}, err
			}
			details := envelope.UpdateGenericRepositoryDetails
			if envelope.RepositoryType != "GENERIC" || details.DisplayName == nil || *details.DisplayName != "mock-repository-updated" ||
				details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" {
				return artifactssdk.GenericRepository{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateRepository details: %+v", envelope)
			}
			state.DisplayName, state.Description, state.FreeformTags = details.DisplayName, details.Description, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state artifactssdk.GenericRepository) (artifactssdk.GenericRepository, ocimock.Response, error) {
			state.LifecycleState = artifactssdk.RepositoryLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
