/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package repository

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
	testRepositoryID     = "ocid1.artifactrepository.oc1..example"
	testRepositoryOther  = "ocid1.artifactrepository.oc1..other"
	testRepositoryCompID = "ocid1.compartment.oc1..example"
)

type fakeRepositoryOCIClient struct {
	createFn func(context.Context, artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error)
	getFn    func(context.Context, artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error)
	listFn   func(context.Context, artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error)
	updateFn func(context.Context, artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error)
	deleteFn func(context.Context, artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error)
}

func (f *fakeRepositoryOCIClient) CreateRepository(
	ctx context.Context,
	request artifactssdk.CreateRepositoryRequest,
) (artifactssdk.CreateRepositoryResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return artifactssdk.CreateRepositoryResponse{}, nil
}

func (f *fakeRepositoryOCIClient) GetRepository(
	ctx context.Context,
	request artifactssdk.GetRepositoryRequest,
) (artifactssdk.GetRepositoryResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return artifactssdk.GetRepositoryResponse{}, nil
}

func (f *fakeRepositoryOCIClient) ListRepositories(
	ctx context.Context,
	request artifactssdk.ListRepositoriesRequest,
) (artifactssdk.ListRepositoriesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return artifactssdk.ListRepositoriesResponse{}, nil
}

func (f *fakeRepositoryOCIClient) UpdateRepository(
	ctx context.Context,
	request artifactssdk.UpdateRepositoryRequest,
) (artifactssdk.UpdateRepositoryResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return artifactssdk.UpdateRepositoryResponse{}, nil
}

func (f *fakeRepositoryOCIClient) DeleteRepository(
	ctx context.Context,
	request artifactssdk.DeleteRepositoryRequest,
) (artifactssdk.DeleteRepositoryResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return artifactssdk.DeleteRepositoryResponse{}, nil
}

func testRepositoryClient(fake *fakeRepositoryOCIClient) RepositoryServiceClient {
	return newRepositoryServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeRepositoryResource() *artifactsv1beta1.Repository {
	return &artifactsv1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-alpha",
			Namespace: "default",
			UID:       types.UID("repository-uid"),
		},
		Spec: artifactsv1beta1.RepositorySpec{
			CompartmentId:  testRepositoryCompID,
			DisplayName:    "repo-alpha",
			Description:    "generic artifact repository",
			IsImmutable:    true,
			RepositoryType: "GENERIC",
			FreeformTags:   map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKRepository(
	id string,
	compartmentID string,
	displayName string,
	description string,
	immutable bool,
	state artifactssdk.RepositoryLifecycleStateEnum,
) artifactssdk.GenericRepository {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return artifactssdk.GenericRepository{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		Description:    common.String(description),
		IsImmutable:    common.Bool(immutable),
		LifecycleState: state,
		TimeCreated:    &created,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKRepositorySummary(
	id string,
	compartmentID string,
	displayName string,
	description string,
	immutable bool,
	state artifactssdk.RepositoryLifecycleStateEnum,
) artifactssdk.GenericRepositorySummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return artifactssdk.GenericRepositorySummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		Description:    common.String(description),
		IsImmutable:    common.Bool(immutable),
		LifecycleState: state,
		TimeCreated:    &created,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func TestRepositoryRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := reviewedRepositoryRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedRepositoryRuntimeSemantics() = nil")
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
	requireRepositoryStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	requireRepositoryStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"})
	requireRepositoryStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "isImmutable", "repositoryType"})
}

func TestRepositoryServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	var createRequest artifactssdk.CreateRepositoryRequest
	var listRequest artifactssdk.ListRepositoriesRequest
	var getRequest artifactssdk.GetRepositoryRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		listFn: func(_ context.Context, request artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			listCalls++
			listRequest = request
			return artifactssdk.ListRepositoriesResponse{
				RepositoryCollection: artifactssdk.RepositoryCollection{},
				OpcRequestId:         common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error) {
			createCalls++
			createRequest = request
			return artifactssdk.CreateRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			getRequest = request
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
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
	assertRepositoryCreateRequest(t, resource, listRequest, createRequest)
	requireRepositoryStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	requireRepositoryStringPtr(t, "get repositoryId", getRequest.RepositoryId, testRepositoryID)
	assertCreatedRepositoryStatus(t, resource)
}

func assertRepositoryCreateRequest(
	t *testing.T,
	resource *artifactsv1beta1.Repository,
	listRequest artifactssdk.ListRepositoriesRequest,
	createRequest artifactssdk.CreateRepositoryRequest,
) {
	t.Helper()
	requireRepositoryStringPtr(t, "list compartmentId", listRequest.CompartmentId, testRepositoryCompID)
	requireRepositoryStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	createDetails, ok := createRequest.CreateRepositoryDetails.(artifactssdk.CreateGenericRepositoryDetails)
	if !ok {
		t.Fatalf("CreateRepositoryDetails = %T, want CreateGenericRepositoryDetails", createRequest.CreateRepositoryDetails)
	}
	requireRepositoryStringPtr(t, "create compartmentId", createDetails.CompartmentId, testRepositoryCompID)
	requireRepositoryStringPtr(t, "create displayName", createDetails.DisplayName, resource.Spec.DisplayName)
	requireRepositoryStringPtr(t, "create description", createDetails.Description, resource.Spec.Description)
	requireRepositoryBoolPtr(t, "create isImmutable", createDetails.IsImmutable, true)
	if !reflect.DeepEqual(createDetails.FreeformTags, map[string]string{"env": "dev"}) {
		t.Fatalf("create freeformTags = %#v, want env=dev", createDetails.FreeformTags)
	}
}

func assertCreatedRepositoryStatus(t *testing.T, resource *artifactsv1beta1.Repository) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testRepositoryID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testRepositoryID)
	}
	if resource.Status.Id != testRepositoryID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testRepositoryID)
	}
	if resource.Status.RepositoryType != string(artifactssdk.RepositoryRepositoryTypeGeneric) {
		t.Fatalf("status.repositoryType = %q, want GENERIC", resource.Status.RepositoryType)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func TestRepositoryServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.IsImmutable = false
	var pages []string
	listCalls := 0
	getCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		listFn: func(_ context.Context, request artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			listCalls++
			pages = append(pages, repositoryStringValue(request.Page))
			if listCalls == 1 {
				return artifactssdk.ListRepositoriesResponse{
					RepositoryCollection: artifactssdk.RepositoryCollection{
						Items: []artifactssdk.RepositorySummary{
							makeSDKRepositorySummary(
								testRepositoryOther,
								testRepositoryCompID,
								"other-repo",
								"other",
								true,
								artifactssdk.RepositoryLifecycleStateAvailable,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return artifactssdk.ListRepositoriesResponse{
				RepositoryCollection: artifactssdk.RepositoryCollection{
					Items: []artifactssdk.RepositorySummary{
						makeSDKRepositorySummary(
							testRepositoryID,
							testRepositoryCompID,
							resource.Spec.DisplayName,
							resource.Spec.Description,
							resource.Spec.IsImmutable,
							artifactssdk.RepositoryLifecycleStateAvailable,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		createFn: func(context.Context, artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error) {
			t.Fatal("CreateRepository() called for existing repository")
			return artifactssdk.CreateRepositoryResponse{}, nil
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
	if got := string(resource.Status.OsokStatus.Ocid); got != testRepositoryID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testRepositoryID)
	}
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func TestRepositoryServiceClientBindsUsingJsonDataIdentityWhenSpecDisplayNameEmpty(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.DisplayName = ""
	resource.Spec.Description = ""
	resource.Spec.JsonData = `{"repositoryType":"GENERIC","displayName":"repo-json","description":"json repository"}`
	var listRequest artifactssdk.ListRepositoriesRequest
	listCalls := 0
	getCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		listFn: func(_ context.Context, request artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			listCalls++
			listRequest = request
			return artifactssdk.ListRepositoriesResponse{
				RepositoryCollection: artifactssdk.RepositoryCollection{
					Items: []artifactssdk.RepositorySummary{
						makeSDKRepositorySummary(
							testRepositoryID,
							testRepositoryCompID,
							"repo-json",
							"json repository",
							resource.Spec.IsImmutable,
							artifactssdk.RepositoryLifecycleStateAvailable,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					"repo-json",
					"json repository",
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		createFn: func(context.Context, artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error) {
			t.Fatal("CreateRepository() called for existing jsonData-identified repository")
			return artifactssdk.CreateRepositoryResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 1/1", listCalls, getCalls)
	}
	requireRepositoryStringPtr(t, "list compartmentId", listRequest.CompartmentId, testRepositoryCompID)
	requireRepositoryStringPtr(t, "list displayName", listRequest.DisplayName, "repo-json")
	if got := string(resource.Status.OsokStatus.Ocid); got != testRepositoryID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testRepositoryID)
	}
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func TestRepositoryServiceClientRejectsConflictingJsonDataIdentityBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.JsonData = `{"repositoryType":"GENERIC","compartmentId":"` +
		testRepositoryCompID + `","displayName":"repo-json","isImmutable":true}`
	listCalls := 0
	createCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		listFn: func(context.Context, artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			listCalls++
			t.Fatal("ListRepositories() called for conflicting jsonData identity")
			return artifactssdk.ListRepositoriesResponse{}, nil
		},
		createFn: func(context.Context, artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error) {
			createCalls++
			t.Fatal("CreateRepository() called for conflicting jsonData identity")
			return artifactssdk.CreateRepositoryResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want conflicting jsonData identity error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if listCalls != 0 || createCalls != 0 {
		t.Fatalf("call counts list/create = %d/%d, want 0/0", listCalls, createCalls)
	}
	if !strings.Contains(err.Error(), "repository jsonData identity conflicts") ||
		!strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName identity conflict", err)
	}
	requireRepositoryLastCondition(t, resource, shared.Failed)
}

func TestRepositoryServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	getCalls := 0
	updateCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		updateFn: func(context.Context, artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
			updateCalls++
			return artifactssdk.UpdateRepositoryResponse{}, nil
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
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func TestRepositoryServiceClientDoesNotClearOmittedTags(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	getCalls := 0
	updateCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		updateFn: func(context.Context, artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
			updateCalls++
			t.Fatal("UpdateRepository() called when tag fields are omitted")
			return artifactssdk.UpdateRepositoryResponse{}, nil
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
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func TestRepositoryServiceClientClearsExplicitEmptyTags(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	getCalls := 0
	updateCalls := 0
	var updateRequest artifactssdk.UpdateRepositoryRequest

	client := newRepositoryClearTagsTestClient(t, resource, &getCalls, &updateCalls, &updateRequest)

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
	assertRepositoryClearTagsUpdate(t, resource, updateRequest)
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func newRepositoryClearTagsTestClient(
	t *testing.T,
	resource *artifactsv1beta1.Repository,
	getCalls *int,
	updateCalls *int,
	updateRequest *artifactssdk.UpdateRepositoryRequest,
) RepositoryServiceClient {
	t.Helper()
	return testRepositoryClient(&fakeRepositoryOCIClient{
		getFn:    repositoryClearTagsGetFn(t, resource, getCalls),
		updateFn: repositoryClearTagsUpdateFn(resource, updateCalls, updateRequest),
	})
}

func repositoryClearTagsGetFn(
	t *testing.T,
	resource *artifactsv1beta1.Repository,
	getCalls *int,
) func(context.Context, artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
	t.Helper()
	return func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
		(*getCalls)++
		requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
		current := makeSDKRepository(
			testRepositoryID,
			testRepositoryCompID,
			resource.Spec.DisplayName,
			resource.Spec.Description,
			resource.Spec.IsImmutable,
			artifactssdk.RepositoryLifecycleStateAvailable,
		)
		if *getCalls > 1 {
			current.FreeformTags = map[string]string{}
			current.DefinedTags = map[string]map[string]interface{}{}
		}
		return artifactssdk.GetRepositoryResponse{Repository: current}, nil
	}
}

func repositoryClearTagsUpdateFn(
	resource *artifactsv1beta1.Repository,
	updateCalls *int,
	updateRequest *artifactssdk.UpdateRepositoryRequest,
) func(context.Context, artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
	return func(_ context.Context, request artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
		(*updateCalls)++
		*updateRequest = request
		updated := makeSDKRepository(
			testRepositoryID,
			testRepositoryCompID,
			resource.Spec.DisplayName,
			resource.Spec.Description,
			resource.Spec.IsImmutable,
			artifactssdk.RepositoryLifecycleStateAvailable,
		)
		updated.FreeformTags = map[string]string{}
		updated.DefinedTags = map[string]map[string]interface{}{}
		return artifactssdk.UpdateRepositoryResponse{
			Repository:   updated,
			OpcRequestId: common.String("opc-clear-tags"),
		}, nil
	}
}

func assertRepositoryClearTagsUpdate(
	t *testing.T,
	resource *artifactsv1beta1.Repository,
	updateRequest artifactssdk.UpdateRepositoryRequest,
) {
	t.Helper()
	updateDetails, ok := updateRequest.UpdateRepositoryDetails.(artifactssdk.UpdateGenericRepositoryDetails)
	if !ok {
		t.Fatalf("UpdateRepositoryDetails = %T, want UpdateGenericRepositoryDetails", updateRequest.UpdateRepositoryDetails)
	}
	if updateDetails.FreeformTags == nil || len(updateDetails.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map", updateDetails.FreeformTags)
	}
	if updateDetails.DefinedTags == nil || len(updateDetails.DefinedTags) != 0 {
		t.Fatalf("update definedTags = %#v, want explicit empty map", updateDetails.DefinedTags)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-clear-tags" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-clear-tags", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestRepositoryServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	resource.Spec.DisplayName = "repo-beta"
	resource.Spec.Description = "updated repository"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	updateCalls := 0
	var updateRequest artifactssdk.UpdateRepositoryRequest

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			current := makeSDKRepository(
				testRepositoryID,
				testRepositoryCompID,
				"repo-alpha",
				"old repository",
				resource.Spec.IsImmutable,
				artifactssdk.RepositoryLifecycleStateAvailable,
			)
			if getCalls > 1 {
				current.DisplayName = common.String(resource.Spec.DisplayName)
				current.Description = common.String(resource.Spec.Description)
				current.FreeformTags = map[string]string{"env": "prod"}
				current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			}
			return artifactssdk.GetRepositoryResponse{Repository: current}, nil
		},
		updateFn: func(_ context.Context, request artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
			updateCalls++
			updateRequest = request
			updated := makeSDKRepository(
				testRepositoryID,
				testRepositoryCompID,
				resource.Spec.DisplayName,
				resource.Spec.Description,
				resource.Spec.IsImmutable,
				artifactssdk.RepositoryLifecycleStateAvailable,
			)
			updated.FreeformTags = map[string]string{"env": "prod"}
			updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			return artifactssdk.UpdateRepositoryResponse{
				Repository:   updated,
				OpcRequestId: common.String("opc-update"),
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
	requireRepositoryStringPtr(t, "update repositoryId", updateRequest.RepositoryId, testRepositoryID)
	updateDetails, ok := updateRequest.UpdateRepositoryDetails.(artifactssdk.UpdateGenericRepositoryDetails)
	if !ok {
		t.Fatalf("UpdateRepositoryDetails = %T, want UpdateGenericRepositoryDetails", updateRequest.UpdateRepositoryDetails)
	}
	requireRepositoryStringPtr(t, "update displayName", updateDetails.DisplayName, resource.Spec.DisplayName)
	requireRepositoryStringPtr(t, "update description", updateDetails.Description, resource.Spec.Description)
	if !reflect.DeepEqual(updateDetails.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateDetails.FreeformTags)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	requireRepositoryLastCondition(t, resource, shared.Active)
}

func TestRepositoryServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	updateCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					"ocid1.compartment.oc1..different",
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		updateFn: func(context.Context, artifactssdk.UpdateRepositoryRequest) (artifactssdk.UpdateRepositoryResponse, error) {
			updateCalls++
			return artifactssdk.UpdateRepositoryResponse{}, nil
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
		t.Fatalf("UpdateRepository() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	requireRepositoryLastCondition(t, resource, shared.Failed)
}

func TestRepositoryServiceClientRejectsUnsupportedRepositoryTypeBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.RepositoryType = "CONTAINER"

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		listFn: func(context.Context, artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			t.Fatal("ListRepositories() should not be called for unsupported repositoryType")
			return artifactssdk.ListRepositoriesResponse{}, nil
		},
		createFn: func(context.Context, artifactssdk.CreateRepositoryRequest) (artifactssdk.CreateRepositoryResponse, error) {
			t.Fatal("CreateRepository() should not be called for unsupported repositoryType")
			return artifactssdk.CreateRepositoryResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported repositoryType error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "unsupported repositoryType") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported repositoryType", err)
	}
	requireRepositoryLastCondition(t, resource, shared.Failed)
}

func TestRepositoryServiceClientRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	getCalls := 0
	deleteCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			getCalls++
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			state := artifactssdk.RepositoryLifecycleStateAvailable
			if getCalls > 1 {
				state = artifactssdk.RepositoryLifecycleStateDeleting
			}
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
			deleteCalls++
			requireRepositoryStringPtr(t, "delete repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.DeleteRepositoryResponse{OpcRequestId: common.String("opc-delete")}, nil
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
	if resource.Status.LifecycleState != string(artifactssdk.RepositoryLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	requireRepositoryLastCondition(t, resource, shared.Terminating)
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", current)
	}
}

func TestRepositoryServiceClientMarksDeletedAfterUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	deleteCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(context.Context, artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			return artifactssdk.GetRepositoryResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "repository is gone")
		},
		deleteFn: func(context.Context, artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
			deleteCalls++
			return artifactssdk.DeleteRepositoryResponse{}, nil
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
		t.Fatalf("DeleteRepository() calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireRepositoryLastCondition(t, resource, shared.Terminating)
}

func TestRepositoryServiceClientRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	deleteCalls := 0
	getErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	getErr.OpcRequestID = "opc-get-predelete"

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(_ context.Context, request artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			requireRepositoryStringPtr(t, "get repositoryId", request.RepositoryId, testRepositoryID)
			return artifactssdk.GetRepositoryResponse{}, getErr
		},
		deleteFn: func(context.Context, artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
			deleteCalls++
			return artifactssdk.DeleteRepositoryResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteRepository() calls = %d, want 0 after ambiguous pre-delete read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous pre-delete confirm-read classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-get-predelete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-get-predelete", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set, want pending deletion status")
	}
	if resource.Status.LifecycleState != "" {
		t.Fatalf("status.lifecycleState = %q, want unchanged", resource.Status.LifecycleState)
	}
}

func TestRepositoryServiceClientRejectsDeleteWithoutRecordedIDOrDisplayName(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Spec.DisplayName = ""
	listCalls := 0
	deleteCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		listFn: func(context.Context, artifactssdk.ListRepositoriesRequest) (artifactssdk.ListRepositoriesResponse, error) {
			listCalls++
			return artifactssdk.ListRepositoriesResponse{}, nil
		},
		deleteFn: func(context.Context, artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
			deleteCalls++
			return artifactssdk.DeleteRepositoryResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want missing identity error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false without recorded identity")
	}
	if listCalls != 0 {
		t.Fatalf("ListRepositories() calls = %d, want 0 without stable delete identity", listCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteRepository() calls = %d, want 0 without stable delete identity", deleteCalls)
	}
	if !strings.Contains(err.Error(), "repository identity is not recorded") {
		t.Fatalf("Delete() error = %v, want missing identity message", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set, want finalizer retained")
	}
}

func TestRepositoryServiceClientTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeRepositoryResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRepositoryID)
	deleteCalls := 0

	client := testRepositoryClient(&fakeRepositoryOCIClient{
		getFn: func(context.Context, artifactssdk.GetRepositoryRequest) (artifactssdk.GetRepositoryResponse, error) {
			return artifactssdk.GetRepositoryResponse{
				Repository: makeSDKRepository(
					testRepositoryID,
					testRepositoryCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					resource.Spec.IsImmutable,
					artifactssdk.RepositoryLifecycleStateAvailable,
				),
			}, nil
		},
		deleteFn: func(context.Context, artifactssdk.DeleteRepositoryRequest) (artifactssdk.DeleteRepositoryResponse, error) {
			deleteCalls++
			return artifactssdk.DeleteRepositoryResponse{}, errortest.NewServiceError(
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
		t.Fatalf("DeleteRepository() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func requireRepositoryStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireRepositoryBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func requireRepositoryLastCondition(
	t *testing.T,
	resource *artifactsv1beta1.Repository,
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

func requireRepositoryStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
