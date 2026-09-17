/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package containerrepository

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

const mockContainerRepositoryID = "ocid1.containerrepo.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - repo-authored runtime: containerrepository_runtime_client.go
//   - OCI SDK and pinned Terraform provider ArtifactsContainerRepository resource
func TestMockIntegrationContainerRepositoryLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &artifactsv1beta1.ContainerRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-container-repository", Namespace: "default", UID: types.UID("mock-container-repository-uid")},
		Spec: artifactsv1beta1.ContainerRepositorySpec{
			CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock/container-repository",
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newContainerRepositoryMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://artifacts.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ContainerRepository OCI mock: %v", err)
		}
	})

	client := newContainerRepositoryServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()}, artifactssdk.ArtifactsClient{BaseClient: session.BaseClient()},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*artifactsv1beta1.ContainerRepository]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *artifactsv1beta1.ContainerRepository) error {
			if current.Status.Id != mockContainerRepositoryID ||
				current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.LifecycleState != string(artifactssdk.ContainerRepositoryLifecycleStateAvailable) ||
				current.Status.IsPublic {
				return fmt.Errorf("created ContainerRepository status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *artifactsv1beta1.ContainerRepository) {
			current.Spec.IsPublic = true
			current.Spec.Readme = artifactsv1beta1.ContainerRepositoryReadme{Content: "Mock lifecycle repository", Format: "text/markdown"}
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *artifactsv1beta1.ContainerRepository) error {
			if !current.Status.IsPublic || current.Status.Readme.Content != current.Spec.Readme.Content ||
				current.Status.Readme.Format != string(artifactssdk.ContainerRepositoryReadmeFormatMarkdown) ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated ContainerRepository status = %+v", current.Status)
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

func newContainerRepositoryMockResponder(resource *artifactsv1beta1.ContainerRepository) (*ocimock.CRUDResponder[artifactssdk.ContainerRepository], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[artifactssdk.ContainerRepository]{
		CollectionPath: "/20160918/container/repositories", ItemPath: "/20160918/container/repositories/" + mockContainerRepositoryID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(request ocimock.Request, present bool, state artifactssdk.ContainerRepository) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("compartmentId") != resource.Spec.CompartmentId || query.Get("displayName") != resource.Spec.DisplayName {
				return ocimock.Response{}, fmt.Errorf("unexpected ListContainerRepositories query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []artifactssdk.ContainerRepository{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []artifactssdk.ContainerRepository{state}})
		},
		Create: func(request ocimock.Request) (artifactssdk.ContainerRepository, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero artifactssdk.ContainerRepository
				return zero, ocimock.Response{}, err
			}
			var details artifactssdk.CreateContainerRepositoryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return artifactssdk.ContainerRepository{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return artifactssdk.ContainerRepository{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.IsPublic == nil || *details.IsPublic || details.IsImmutable == nil || *details.IsImmutable {
				return artifactssdk.ContainerRepository{}, ocimock.Response{}, fmt.Errorf("unexpected CreateContainerRepository details: %+v", details)
			}
			state := artifactssdk.ContainerRepository{
				CompartmentId: details.CompartmentId, CreatedBy: common.String("ocid1.user.oc1..mock"), DisplayName: details.DisplayName,
				Id: common.String(mockContainerRepositoryID), ImageCount: common.Int(0), IsImmutable: details.IsImmutable,
				IsPublic: details.IsPublic, LayerCount: common.Int(0), LayersSizeInBytes: common.Int64(0),
				LifecycleState: artifactssdk.ContainerRepositoryLifecycleStateAvailable, TimeCreated: &createdAt,
				BillableSizeInGBs: common.Int64(0), Namespace: common.String("mocknamespace"), FreeformTags: details.FreeformTags,
				DefinedTags: map[string]map[string]interface{}{}, SystemTags: map[string]map[string]interface{}{}, Readme: details.Readme,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state artifactssdk.ContainerRepository) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state artifactssdk.ContainerRepository) (artifactssdk.ContainerRepository, ocimock.Response, error) {
			var details artifactssdk.UpdateContainerRepositoryDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return artifactssdk.ContainerRepository{}, ocimock.Response{}, err
			}
			if details.IsPublic == nil || !*details.IsPublic || details.Readme == nil || details.Readme.Content == nil ||
				*details.Readme.Content != "Mock lifecycle repository" || details.Readme.Format != artifactssdk.ContainerRepositoryReadmeFormatMarkdown ||
				details.FreeformTags["osok-mock"] != "update" {
				return artifactssdk.ContainerRepository{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateContainerRepository details: %+v", details)
			}
			state.IsPublic, state.Readme, state.FreeformTags = details.IsPublic, details.Readme, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ artifactssdk.ContainerRepository) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(_ ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusNotFound, map[string]any{
				"code": "REPO_ID_UNKNOWN", "message": "Repository Id Unknown",
				"detail": map[string]string{"userSuppliedRepoId": mockContainerRepositoryID},
			})
		},
	})
}
