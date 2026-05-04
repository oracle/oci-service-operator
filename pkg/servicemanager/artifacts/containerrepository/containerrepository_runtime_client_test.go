/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerrepository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testContainerRepositoryID     = "ocid1.containerrepo.oc1..example"
	testContainerRepositoryOther  = "ocid1.containerrepo.oc1..other"
	testContainerRepositoryCompID = "ocid1.compartment.oc1..example"
)

type fakeContainerRepositoryOCIClient struct {
	createFn func(context.Context, artifactssdk.CreateContainerRepositoryRequest) (artifactssdk.CreateContainerRepositoryResponse, error)
	getFn    func(context.Context, artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error)
	listFn   func(context.Context, artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error)
	updateFn func(context.Context, artifactssdk.UpdateContainerRepositoryRequest) (artifactssdk.UpdateContainerRepositoryResponse, error)
	deleteFn func(context.Context, artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error)
}

func (f *fakeContainerRepositoryOCIClient) CreateContainerRepository(
	ctx context.Context,
	request artifactssdk.CreateContainerRepositoryRequest,
) (artifactssdk.CreateContainerRepositoryResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return artifactssdk.CreateContainerRepositoryResponse{}, nil
}

func (f *fakeContainerRepositoryOCIClient) GetContainerRepository(
	ctx context.Context,
	request artifactssdk.GetContainerRepositoryRequest,
) (artifactssdk.GetContainerRepositoryResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return artifactssdk.GetContainerRepositoryResponse{}, nil
}

func (f *fakeContainerRepositoryOCIClient) ListContainerRepositories(
	ctx context.Context,
	request artifactssdk.ListContainerRepositoriesRequest,
) (artifactssdk.ListContainerRepositoriesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return artifactssdk.ListContainerRepositoriesResponse{}, nil
}

func (f *fakeContainerRepositoryOCIClient) UpdateContainerRepository(
	ctx context.Context,
	request artifactssdk.UpdateContainerRepositoryRequest,
) (artifactssdk.UpdateContainerRepositoryResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return artifactssdk.UpdateContainerRepositoryResponse{}, nil
}

func (f *fakeContainerRepositoryOCIClient) DeleteContainerRepository(
	ctx context.Context,
	request artifactssdk.DeleteContainerRepositoryRequest,
) (artifactssdk.DeleteContainerRepositoryResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return artifactssdk.DeleteContainerRepositoryResponse{}, nil
}

func testContainerRepositoryClient(fake *fakeContainerRepositoryOCIClient) ContainerRepositoryServiceClient {
	return newContainerRepositoryServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeContainerRepositoryResource() *artifactsv1beta1.ContainerRepository {
	return &artifactsv1beta1.ContainerRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-alpha",
			Namespace: "default",
			UID:       types.UID("containerrepository-uid"),
		},
		Spec: artifactsv1beta1.ContainerRepositorySpec{
			CompartmentId: testContainerRepositoryCompID,
			DisplayName:   "repo-alpha",
			IsImmutable:   true,
			IsPublic:      true,
			Readme: artifactsv1beta1.ContainerRepositoryReadme{
				Content: "hello registry",
				Format:  "text/markdown",
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKContainerRepository(
	id string,
	compartmentID string,
	displayName string,
	immutable bool,
	public bool,
	state artifactssdk.ContainerRepositoryLifecycleStateEnum,
) artifactssdk.ContainerRepository {
	imageCount := 1
	layerCount := 2
	layerBytes := int64(128)
	billableGB := int64(1)
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return artifactssdk.ContainerRepository{
		CompartmentId:     common.String(compartmentID),
		CreatedBy:         common.String("ocid1.user.oc1..example"),
		DisplayName:       common.String(displayName),
		Id:                common.String(id),
		ImageCount:        common.Int(imageCount),
		IsImmutable:       common.Bool(immutable),
		IsPublic:          common.Bool(public),
		LayerCount:        common.Int(layerCount),
		LayersSizeInBytes: common.Int64(layerBytes),
		LifecycleState:    state,
		TimeCreated:       &created,
		BillableSizeInGBs: common.Int64(billableGB),
		Namespace:         common.String("tenantns"),
		FreeformTags:      map[string]string{"env": "dev"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:        map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		Readme: &artifactssdk.ContainerRepositoryReadme{
			Content: common.String("hello registry"),
			Format:  artifactssdk.ContainerRepositoryReadmeFormatMarkdown,
		},
	}
}

func makeSDKContainerRepositorySummary(
	id string,
	compartmentID string,
	displayName string,
	public bool,
	state artifactssdk.ContainerRepositoryLifecycleStateEnum,
) artifactssdk.ContainerRepositorySummary {
	imageCount := 1
	layerCount := 2
	layerBytes := int64(128)
	billableGB := int64(1)
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return artifactssdk.ContainerRepositorySummary{
		CompartmentId:     common.String(compartmentID),
		DisplayName:       common.String(displayName),
		Id:                common.String(id),
		ImageCount:        common.Int(imageCount),
		IsPublic:          common.Bool(public),
		LayerCount:        common.Int(layerCount),
		LayersSizeInBytes: common.Int64(layerBytes),
		LifecycleState:    state,
		TimeCreated:       &created,
		BillableSizeInGBs: common.Int64(billableGB),
		Namespace:         common.String("tenantns"),
		FreeformTags:      map[string]string{"env": "dev"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:        map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func TestContainerRepositoryRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := reviewedContainerRepositoryRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedContainerRepositoryRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	requireStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"isImmutable", "isPublic", "readme", "freeformTags", "definedTags"})
	requireStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "displayName"})
}

func TestContainerRepositoryServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	var createRequest artifactssdk.CreateContainerRepositoryRequest
	var listRequest artifactssdk.ListContainerRepositoriesRequest
	var getRequest artifactssdk.GetContainerRepositoryRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		listFn: func(_ context.Context, request artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error) {
			listCalls++
			listRequest = request
			return artifactssdk.ListContainerRepositoriesResponse{
				ContainerRepositoryCollection: artifactssdk.ContainerRepositoryCollection{},
				OpcRequestId:                  common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request artifactssdk.CreateContainerRepositoryRequest) (artifactssdk.CreateContainerRepositoryResponse, error) {
			createCalls++
			createRequest = request
			return artifactssdk.CreateContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					testContainerRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					artifactssdk.ContainerRepositoryLifecycleStateAvailable,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			getCalls++
			getRequest = request
			return artifactssdk.GetContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					testContainerRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					artifactssdk.ContainerRepositoryLifecycleStateAvailable,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 1 || createCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	assertContainerRepositoryCreateRequest(t, resource, listRequest, createRequest)
	requireStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	requireStringPtr(t, "get repositoryId", getRequest.RepositoryId, testContainerRepositoryID)
	assertCreatedContainerRepositoryStatus(t, resource)
}

func assertContainerRepositoryCreateRequest(
	t *testing.T,
	resource *artifactsv1beta1.ContainerRepository,
	listRequest artifactssdk.ListContainerRepositoriesRequest,
	createRequest artifactssdk.CreateContainerRepositoryRequest,
) {
	t.Helper()
	requireStringPtr(t, "list compartmentId", listRequest.CompartmentId, testContainerRepositoryCompID)
	requireStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, testContainerRepositoryCompID)
	requireStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireBoolPtr(t, "create isImmutable", createRequest.IsImmutable, true)
	requireBoolPtr(t, "create isPublic", createRequest.IsPublic, true)
	requireStringPtr(t, "create readme content", createRequest.Readme.Content, resource.Spec.Readme.Content)
	if createRequest.Readme.Format != artifactssdk.ContainerRepositoryReadmeFormatMarkdown {
		t.Fatalf("create readme format = %q, want %q", createRequest.Readme.Format, artifactssdk.ContainerRepositoryReadmeFormatMarkdown)
	}
}

func assertCreatedContainerRepositoryStatus(t *testing.T, resource *artifactsv1beta1.ContainerRepository) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testContainerRepositoryID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testContainerRepositoryID)
	}
	if resource.Status.Id != testContainerRepositoryID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testContainerRepositoryID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestContainerRepositoryServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Spec.IsImmutable = false
	var pages []string
	listCalls := 0
	getCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		listFn: func(_ context.Context, request artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error) {
			listCalls++
			pages = append(pages, stringValue(request.Page))
			if listCalls == 1 {
				return artifactssdk.ListContainerRepositoriesResponse{
					ContainerRepositoryCollection: artifactssdk.ContainerRepositoryCollection{
						Items: []artifactssdk.ContainerRepositorySummary{
							makeSDKContainerRepositorySummary(
								testContainerRepositoryOther,
								testContainerRepositoryCompID,
								"other-repo",
								true,
								artifactssdk.ContainerRepositoryLifecycleStateAvailable,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return artifactssdk.ListContainerRepositoriesResponse{
				ContainerRepositoryCollection: artifactssdk.ContainerRepositoryCollection{
					Items: []artifactssdk.ContainerRepositorySummary{
						makeSDKContainerRepositorySummary(
							testContainerRepositoryID,
							testContainerRepositoryCompID,
							resource.Spec.DisplayName,
							resource.Spec.IsPublic,
							artifactssdk.ContainerRepositoryLifecycleStateAvailable,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			getCalls++
			requireStringPtr(t, "get repositoryId", request.RepositoryId, testContainerRepositoryID)
			return artifactssdk.GetContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					testContainerRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					artifactssdk.ContainerRepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		createFn: func(context.Context, artifactssdk.CreateContainerRepositoryRequest) (artifactssdk.CreateContainerRepositoryResponse, error) {
			t.Fatal("CreateContainerRepository() called for existing repository")
			return artifactssdk.CreateContainerRepositoryResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 2/1", listCalls, getCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testContainerRepositoryID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testContainerRepositoryID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestContainerRepositoryServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerRepositoryID)
	getCalls := 0
	updateCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			getCalls++
			requireStringPtr(t, "get repositoryId", request.RepositoryId, testContainerRepositoryID)
			return artifactssdk.GetContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					testContainerRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					artifactssdk.ContainerRepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		updateFn: func(context.Context, artifactssdk.UpdateContainerRepositoryRequest) (artifactssdk.UpdateContainerRepositoryResponse, error) {
			updateCalls++
			return artifactssdk.UpdateContainerRepositoryResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts get/update = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestContainerRepositoryServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerRepositoryID)
	resource.Spec.IsImmutable = false
	resource.Spec.IsPublic = true
	resource.Spec.Readme = artifactsv1beta1.ContainerRepositoryReadme{Content: "updated readme", Format: "text/plain"}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	updateCalls := 0
	var updateRequest artifactssdk.UpdateContainerRepositoryRequest

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			getCalls++
			requireStringPtr(t, "get repositoryId", request.RepositoryId, testContainerRepositoryID)
			current := makeSDKContainerRepository(
				testContainerRepositoryID,
				testContainerRepositoryCompID,
				resource.Spec.DisplayName,
				false,
				false,
				artifactssdk.ContainerRepositoryLifecycleStateAvailable,
			)
			if getCalls > 1 {
				current.IsPublic = common.Bool(true)
				current.Readme = &artifactssdk.ContainerRepositoryReadme{
					Content: common.String(resource.Spec.Readme.Content),
					Format:  artifactssdk.ContainerRepositoryReadmeFormatPlain,
				}
				current.FreeformTags = map[string]string{"env": "prod"}
				current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			}
			return artifactssdk.GetContainerRepositoryResponse{ContainerRepository: current}, nil
		},
		updateFn: func(_ context.Context, request artifactssdk.UpdateContainerRepositoryRequest) (artifactssdk.UpdateContainerRepositoryResponse, error) {
			updateCalls++
			updateRequest = request
			updated := makeSDKContainerRepository(
				testContainerRepositoryID,
				testContainerRepositoryCompID,
				resource.Spec.DisplayName,
				false,
				true,
				artifactssdk.ContainerRepositoryLifecycleStateAvailable,
			)
			updated.Readme = &artifactssdk.ContainerRepositoryReadme{
				Content: common.String(resource.Spec.Readme.Content),
				Format:  artifactssdk.ContainerRepositoryReadmeFormatPlain,
			}
			updated.FreeformTags = map[string]string{"env": "prod"}
			updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			return artifactssdk.UpdateContainerRepositoryResponse{
				ContainerRepository: updated,
				OpcRequestId:        common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	requireStringPtr(t, "update repositoryId", updateRequest.RepositoryId, testContainerRepositoryID)
	requireBoolPtr(t, "update isPublic", updateRequest.IsPublic, true)
	requireStringPtr(t, "update readme content", updateRequest.Readme.Content, resource.Spec.Readme.Content)
	if updateRequest.Readme.Format != artifactssdk.ContainerRepositoryReadmeFormatPlain {
		t.Fatalf("update readme format = %q, want %q", updateRequest.Readme.Format, artifactssdk.ContainerRepositoryReadmeFormatPlain)
	}
	if !reflect.DeepEqual(updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestContainerRepositoryServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerRepositoryID)
	updateCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			requireStringPtr(t, "get repositoryId", request.RepositoryId, testContainerRepositoryID)
			return artifactssdk.GetContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					"ocid1.compartment.oc1..different",
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					artifactssdk.ContainerRepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		updateFn: func(context.Context, artifactssdk.UpdateContainerRepositoryRequest) (artifactssdk.UpdateContainerRepositoryResponse, error) {
			updateCalls++
			return artifactssdk.UpdateContainerRepositoryResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateContainerRepository() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestContainerRepositoryServiceClientRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerRepositoryID)
	getCalls := 0
	deleteCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			getCalls++
			requireStringPtr(t, "get repositoryId", request.RepositoryId, testContainerRepositoryID)
			state := artifactssdk.ContainerRepositoryLifecycleStateAvailable
			if getCalls > 1 {
				state = artifactssdk.ContainerRepositoryLifecycleStateDeleting
			}
			return artifactssdk.GetContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					testContainerRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete repositoryId", request.RepositoryId, testContainerRepositoryID)
			return artifactssdk.DeleteContainerRepositoryResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 2/1", getCalls, deleteCalls)
	}
	if resource.Status.LifecycleState != string(artifactssdk.ContainerRepositoryLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Terminating)
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", current)
	}
}

func TestContainerRepositoryServiceClientMarksDeletedAfterUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerRepositoryID)
	deleteCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		getFn: func(context.Context, artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			return artifactssdk.GetContainerRepositoryResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "repository is gone")
		},
		deleteFn: func(context.Context, artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error) {
			deleteCalls++
			return artifactssdk.DeleteContainerRepositoryResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteContainerRepository() calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestContainerRepositoryServiceClientTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeContainerRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerRepositoryID)
	deleteCalls := 0

	client := testContainerRepositoryClient(&fakeContainerRepositoryOCIClient{
		getFn: func(context.Context, artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			return artifactssdk.GetContainerRepositoryResponse{
				ContainerRepository: makeSDKContainerRepository(
					testContainerRepositoryID,
					testContainerRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.IsImmutable,
					resource.Spec.IsPublic,
					artifactssdk.ContainerRepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		deleteFn: func(context.Context, artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error) {
			deleteCalls++
			return artifactssdk.DeleteContainerRepositoryResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"authorization or existence is ambiguous",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteContainerRepository() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func requireLastCondition(
	t *testing.T,
	resource *artifactsv1beta1.ContainerRepository,
	want shared.OSOKConditionType,
) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil")
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
