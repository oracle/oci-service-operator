/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cabundle

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	certificatesmanagementsdk "github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
	certificatesmanagementv1beta1 "github.com/oracle/oci-service-operator/api/certificatesmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCaBundleID            = "ocid1.cabundle.oc1..example"
	testOtherCaBundleID       = "ocid1.cabundle.oc1..other"
	testCaBundleCompartmentID = "ocid1.compartment.oc1..example"
	testCaBundlePEM           = "-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----"
	testUpdatedCaBundlePEM    = "-----BEGIN CERTIFICATE-----\nupdated\n-----END CERTIFICATE-----"
)

type fakeCaBundleOCIClient struct {
	createFn func(context.Context, certificatesmanagementsdk.CreateCaBundleRequest) (certificatesmanagementsdk.CreateCaBundleResponse, error)
	getFn    func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error)
	listFn   func(context.Context, certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error)
	updateFn func(context.Context, certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error)
	deleteFn func(context.Context, certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error)
}

func (f *fakeCaBundleOCIClient) CreateCaBundle(
	ctx context.Context,
	request certificatesmanagementsdk.CreateCaBundleRequest,
) (certificatesmanagementsdk.CreateCaBundleResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return certificatesmanagementsdk.CreateCaBundleResponse{}, nil
}

func (f *fakeCaBundleOCIClient) GetCaBundle(
	ctx context.Context,
	request certificatesmanagementsdk.GetCaBundleRequest,
) (certificatesmanagementsdk.GetCaBundleResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return certificatesmanagementsdk.GetCaBundleResponse{}, nil
}

func (f *fakeCaBundleOCIClient) ListCaBundles(
	ctx context.Context,
	request certificatesmanagementsdk.ListCaBundlesRequest,
) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return certificatesmanagementsdk.ListCaBundlesResponse{}, nil
}

func (f *fakeCaBundleOCIClient) UpdateCaBundle(
	ctx context.Context,
	request certificatesmanagementsdk.UpdateCaBundleRequest,
) (certificatesmanagementsdk.UpdateCaBundleResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return certificatesmanagementsdk.UpdateCaBundleResponse{}, nil
}

func (f *fakeCaBundleOCIClient) DeleteCaBundle(
	ctx context.Context,
	request certificatesmanagementsdk.DeleteCaBundleRequest,
) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return certificatesmanagementsdk.DeleteCaBundleResponse{}, nil
}

func testCaBundleClient(fake *fakeCaBundleOCIClient) CaBundleServiceClient {
	return newCaBundleServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeCaBundleResource() *certificatesmanagementv1beta1.CaBundle {
	return &certificatesmanagementv1beta1.CaBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bundle-alpha",
			Namespace: "default",
			UID:       types.UID("cabundle-uid"),
		},
		Spec: certificatesmanagementv1beta1.CaBundleSpec{
			Name:          "bundle-alpha",
			CompartmentId: testCaBundleCompartmentID,
			CaBundlePem:   testCaBundlePEM,
			Description:   "initial bundle",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKCaBundle(
	id string,
	compartmentID string,
	name string,
	description string,
	state certificatesmanagementsdk.CaBundleLifecycleStateEnum,
) certificatesmanagementsdk.CaBundle {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return certificatesmanagementsdk.CaBundle{
		Id:               common.String(id),
		Name:             common.String(name),
		TimeCreated:      &created,
		LifecycleState:   state,
		CompartmentId:    common.String(compartmentID),
		Description:      common.String(description),
		LifecycleDetails: common.String("ready"),
		FreeformTags:     caBundleFreeformTagsFromSpec(map[string]string{"env": "dev"}, testCaBundlePEM),
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKCaBundleSummary(
	id string,
	compartmentID string,
	name string,
	description string,
	state certificatesmanagementsdk.CaBundleLifecycleStateEnum,
) certificatesmanagementsdk.CaBundleSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
	return certificatesmanagementsdk.CaBundleSummary{
		Id:               common.String(id),
		Name:             common.String(name),
		TimeCreated:      &created,
		LifecycleState:   state,
		CompartmentId:    common.String(compartmentID),
		Description:      common.String(description),
		LifecycleDetails: common.String("ready"),
		FreeformTags:     caBundleFreeformTagsFromSpec(map[string]string{"env": "dev"}, testCaBundlePEM),
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func TestCaBundleRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := reviewedCaBundleRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedCaBundleRuntimeSemantics() = nil")
	}
	if got.FormalService != "certificatesmanagement" || got.FormalSlug != "cabundle" {
		t.Fatalf("formal binding = %s/%s, want certificatesmanagement/cabundle", got.FormalService, got.FormalSlug)
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
	requireCaBundleStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "id"})
	requireCaBundleStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"caBundlePem", "description", "freeformTags", "definedTags"})
	requireCaBundleStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "name"})
}

func TestCaBundleServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0
	var createRequest certificatesmanagementsdk.CreateCaBundleRequest
	var listRequest certificatesmanagementsdk.ListCaBundlesRequest

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		listFn: func(_ context.Context, request certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
			listCalls++
			listRequest = request
			return certificatesmanagementsdk.ListCaBundlesResponse{}, nil
		},
		createFn: func(_ context.Context, request certificatesmanagementsdk.CreateCaBundleRequest) (certificatesmanagementsdk.CreateCaBundleResponse, error) {
			createCalls++
			createRequest = request
			return certificatesmanagementsdk.CreateCaBundleResponse{
				CaBundle:     makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, request certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			getCalls++
			requireStringPtr(t, "get caBundleId", request.CaBundleId, testCaBundleID)
			return certificatesmanagementsdk.GetCaBundleResponse{
				CaBundle: makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
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
		t.Fatalf("calls list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	requireStringPtr(t, "list compartmentId", listRequest.CompartmentId, testCaBundleCompartmentID)
	requireStringPtr(t, "list name", listRequest.Name, resource.Spec.Name)
	requireStringPtr(t, "create name", createRequest.Name, resource.Spec.Name)
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, testCaBundleCompartmentID)
	requireStringPtr(t, "create caBundlePem", createRequest.CaBundlePem, testCaBundlePEM)
	requireStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	requireCaBundlePEMFingerprintTag(t, createRequest.FreeformTags, testCaBundlePEM)
	assertCreatedCaBundleStatus(t, resource)
}

func TestCaBundleServiceClientBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	listCalls := 0
	getCalls := 0
	createCalls := 0
	var pages []string

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		listFn: func(_ context.Context, request certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
			listCalls++
			pages = append(pages, stringValue(request.Page))
			if listCalls == 1 {
				return certificatesmanagementsdk.ListCaBundlesResponse{
					CaBundleCollection: certificatesmanagementsdk.CaBundleCollection{
						Items: []certificatesmanagementsdk.CaBundleSummary{
							makeSDKCaBundleSummary(testOtherCaBundleID, testCaBundleCompartmentID, "other-bundle", "other", certificatesmanagementsdk.CaBundleLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return certificatesmanagementsdk.ListCaBundlesResponse{
				CaBundleCollection: certificatesmanagementsdk.CaBundleCollection{
					Items: []certificatesmanagementsdk.CaBundleSummary{
						makeSDKCaBundleSummary(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			getCalls++
			requireStringPtr(t, "get caBundleId", request.CaBundleId, testCaBundleID)
			return certificatesmanagementsdk.GetCaBundleResponse{
				CaBundle: makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, certificatesmanagementsdk.CreateCaBundleRequest) (certificatesmanagementsdk.CreateCaBundleResponse, error) {
			createCalls++
			return certificatesmanagementsdk.CreateCaBundleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 2 || getCalls != 1 || createCalls != 0 {
		t.Fatalf("calls list/get/create = %d/%d/%d, want 2/1/0", listCalls, getCalls, createCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testCaBundleID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testCaBundleID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestCaBundleServiceClientRejectsDuplicateListMatchesAcrossPages(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	listCalls := 0
	createCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		listFn: func(_ context.Context, request certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
			listCalls++
			if listCalls == 1 {
				return certificatesmanagementsdk.ListCaBundlesResponse{
					CaBundleCollection: certificatesmanagementsdk.CaBundleCollection{
						Items: []certificatesmanagementsdk.CaBundleSummary{
							makeSDKCaBundleSummary(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second list page", request.Page, "page-2")
			return certificatesmanagementsdk.ListCaBundlesResponse{
				CaBundleCollection: certificatesmanagementsdk.CaBundleCollection{
					Items: []certificatesmanagementsdk.CaBundleSummary{
						makeSDKCaBundleSummary("ocid1.cabundle.oc1..duplicate", testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
					},
				},
			}, nil
		},
		createFn: func(context.Context, certificatesmanagementsdk.CreateCaBundleRequest) (certificatesmanagementsdk.CreateCaBundleResponse, error) {
			createCalls++
			return certificatesmanagementsdk.CreateCaBundleResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want duplicate list match error")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %q, want multiple matching resources", err.Error())
	}
	if createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", createCalls)
	}
}

func TestCaBundleServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	updateCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(_ context.Context, request certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			requireStringPtr(t, "get caBundleId", request.CaBundleId, testCaBundleID)
			return certificatesmanagementsdk.GetCaBundleResponse{
				CaBundle: makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error) {
			updateCalls++
			return certificatesmanagementsdk.UpdateCaBundleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestCaBundleServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	setCaBundleMutableUpdateSpec(resource)

	getCalls := 0
	updateCalls := 0
	var updateRequest certificatesmanagementsdk.UpdateCaBundleRequest
	client := newCaBundleMutableUpdateTestClient(t, resource, &getCalls, &updateCalls, &updateRequest)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("calls get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	requireCaBundleMutableUpdateRequest(t, updateRequest, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func setCaBundleMutableUpdateSpec(resource *certificatesmanagementv1beta1.CaBundle) {
	resource.Spec.Description = "updated bundle"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
}

func newCaBundleMutableUpdateTestClient(
	t *testing.T,
	resource *certificatesmanagementv1beta1.CaBundle,
	getCalls *int,
	updateCalls *int,
	updateRequest *certificatesmanagementsdk.UpdateCaBundleRequest,
) CaBundleServiceClient {
	t.Helper()
	return testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			(*getCalls)++
			return certificatesmanagementsdk.GetCaBundleResponse{CaBundle: caBundleForMutableUpdate(resource, *getCalls)}, nil
		},
		updateFn: func(_ context.Context, request certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error) {
			(*updateCalls)++
			*updateRequest = request
			return certificatesmanagementsdk.UpdateCaBundleResponse{
				CaBundle:     updatedCaBundleForMutableUpdate(resource),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})
}

func caBundleForMutableUpdate(
	resource *certificatesmanagementv1beta1.CaBundle,
	getCalls int,
) certificatesmanagementsdk.CaBundle {
	if getCalls == 1 {
		return makeSDKCaBundle(
			testCaBundleID,
			testCaBundleCompartmentID,
			resource.Spec.Name,
			"initial bundle",
			certificatesmanagementsdk.CaBundleLifecycleStateActive,
		)
	}
	return updatedCaBundleForMutableUpdate(resource)
}

func updatedCaBundleForMutableUpdate(
	resource *certificatesmanagementv1beta1.CaBundle,
) certificatesmanagementsdk.CaBundle {
	updated := makeSDKCaBundle(
		testCaBundleID,
		testCaBundleCompartmentID,
		resource.Spec.Name,
		resource.Spec.Description,
		certificatesmanagementsdk.CaBundleLifecycleStateActive,
	)
	updated.FreeformTags = caBundleFreeformTagsFromSpec(map[string]string{"env": "prod"}, resource.Spec.CaBundlePem)
	updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
	return updated
}

func requireCaBundleMutableUpdateRequest(
	t *testing.T,
	updateRequest certificatesmanagementsdk.UpdateCaBundleRequest,
	resource *certificatesmanagementv1beta1.CaBundle,
) {
	t.Helper()
	requireStringPtr(t, "update caBundleId", updateRequest.CaBundleId, testCaBundleID)
	requireStringPtr(t, "update description", updateRequest.Description, resource.Spec.Description)
	requireCaBundleUserFreeformTags(t, updateRequest.FreeformTags, map[string]string{"env": "prod"})
	requireCaBundlePEMFingerprintTag(t, updateRequest.FreeformTags, resource.Spec.CaBundlePem)
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update defined tag CostCenter = %#v, want 84", got)
	}
	if updateRequest.CaBundlePem != nil {
		t.Fatalf("update caBundlePem = %#v, want nil when PEM fingerprint already matches", updateRequest.CaBundlePem)
	}
}

func TestCaBundleServiceClientUpdatesPEMWhenTrackedFingerprintDiffers(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	resource.Spec.CaBundlePem = testUpdatedCaBundlePEM

	updateCalls := 0
	var updateRequest certificatesmanagementsdk.UpdateCaBundleRequest
	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			current := makeSDKCaBundle(
				testCaBundleID,
				testCaBundleCompartmentID,
				resource.Spec.Name,
				resource.Spec.Description,
				certificatesmanagementsdk.CaBundleLifecycleStateActive,
			)
			if updateCalls > 0 {
				current.FreeformTags = caBundleFreeformTagsFromSpec(resource.Spec.FreeformTags, resource.Spec.CaBundlePem)
			}
			return certificatesmanagementsdk.GetCaBundleResponse{CaBundle: current}, nil
		},
		updateFn: func(_ context.Context, request certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error) {
			updateCalls++
			updateRequest = request
			updated := makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive)
			updated.FreeformTags = caBundleFreeformTagsFromSpec(resource.Spec.FreeformTags, resource.Spec.CaBundlePem)
			return certificatesmanagementsdk.UpdateCaBundleResponse{CaBundle: updated}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", updateCalls)
	}
	requireStringPtr(t, "update caBundlePem", updateRequest.CaBundlePem, resource.Spec.CaBundlePem)
	requireCaBundlePEMFingerprintTag(t, updateRequest.FreeformTags, resource.Spec.CaBundlePem)
	requireCaBundlePEMFingerprintTag(t, resource.Status.FreeformTags, resource.Spec.CaBundlePem)
	requireLastCondition(t, resource, shared.Active)
}

func TestCaBundleServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	updateCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			current := makeSDKCaBundle(testCaBundleID, "ocid1.compartment.oc1..different", resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive)
			return certificatesmanagementsdk.GetCaBundleResponse{CaBundle: current}, nil
		},
		updateFn: func(context.Context, certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error) {
			updateCalls++
			return certificatesmanagementsdk.UpdateCaBundleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId drift", err.Error())
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestCaBundleServiceClientRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	getCalls := 0
	deleteCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(_ context.Context, request certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			getCalls++
			requireStringPtr(t, "get caBundleId", request.CaBundleId, testCaBundleID)
			return certificatesmanagementsdk.GetCaBundleResponse{
				CaBundle: makeSDKCaBundle(
					testCaBundleID,
					testCaBundleCompartmentID,
					resource.Spec.Name,
					resource.Spec.Description,
					caBundleDeleteStateForCall(getCalls),
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete caBundleId", request.CaBundleId, testCaBundleID)
			return certificatesmanagementsdk.DeleteCaBundleResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	requireCaBundleDeletePending(t, deleted, resource)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	requireCaBundleDeleteConfirmed(t, deleted, deleteCalls)
}

func caBundleDeleteStateForCall(call int) certificatesmanagementsdk.CaBundleLifecycleStateEnum {
	states := []certificatesmanagementsdk.CaBundleLifecycleStateEnum{
		certificatesmanagementsdk.CaBundleLifecycleStateActive,
		certificatesmanagementsdk.CaBundleLifecycleStateActive,
		certificatesmanagementsdk.CaBundleLifecycleStateDeleting,
		certificatesmanagementsdk.CaBundleLifecycleStateDeleted,
	}
	if call <= 0 {
		return certificatesmanagementsdk.CaBundleLifecycleStateActive
	}
	if call > len(states) {
		return states[len(states)-1]
	}
	return states[call-1]
}

func requireCaBundleDeletePending(t *testing.T, deleted bool, resource *certificatesmanagementv1beta1.CaBundle) {
	t.Helper()
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while OCI is DELETING")
	}
	if got := resource.Status.LifecycleState; got != string(certificatesmanagementsdk.CaBundleLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func requireCaBundleDeleteConfirmed(t *testing.T, deleted bool, deleteCalls int) {
	t.Helper()
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after OCI is DELETED")
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", deleteCalls)
	}
}

func TestCaBundleServiceClientTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	deleteCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			return certificatesmanagementsdk.GetCaBundleResponse{
				CaBundle: makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
			deleteCalls++
			return certificatesmanagementsdk.DeleteCaBundleResponse{}, errortest.NewServiceError(
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
		t.Fatalf("delete calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous auth-shaped 404", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestCaBundleServiceClientRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	deleteCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			return certificatesmanagementsdk.GetCaBundleResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"authorization or existence is ambiguous",
			)
		},
		deleteFn: func(context.Context, certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
			deleteCalls++
			return certificatesmanagementsdk.DeleteCaBundleResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if deleteCalls != 0 {
		t.Fatalf("delete calls = %d, want 0 after auth-shaped pre-delete read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous confirm-read error", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestCaBundleServiceClientRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeCaBundleResource()
	resource.Status.Id = testCaBundleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testCaBundleID)
	getCalls := 0
	deleteCalls := 0

	client := testCaBundleClient(&fakeCaBundleOCIClient{
		getFn: func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			getCalls++
			if getCalls > 2 {
				return certificatesmanagementsdk.GetCaBundleResponse{}, errortest.NewServiceError(
					404,
					errorutil.NotAuthorizedOrNotFound,
					"authorization or existence is ambiguous",
				)
			}
			return certificatesmanagementsdk.GetCaBundleResponse{
				CaBundle: makeSDKCaBundle(testCaBundleID, testCaBundleCompartmentID, resource.Spec.Name, resource.Spec.Description, certificatesmanagementsdk.CaBundleLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
			deleteCalls++
			return certificatesmanagementsdk.DeleteCaBundleResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete read")
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1 before auth-shaped post-delete read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous auth-shaped 404", err.Error())
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func assertCreatedCaBundleStatus(t *testing.T, resource *certificatesmanagementv1beta1.CaBundle) {
	t.Helper()
	if got := resource.Status.Id; got != testCaBundleID {
		t.Fatalf("status.id = %q, want %q", got, testCaBundleID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testCaBundleID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testCaBundleID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := resource.Status.LifecycleState; got != string(certificatesmanagementsdk.CaBundleLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	requireCaBundleUserFreeformTags(t, resource.Status.FreeformTags, map[string]string{"env": "dev"})
	requireLastCondition(t, resource, shared.Active)
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

func requireLastCondition(
	t *testing.T,
	resource *certificatesmanagementv1beta1.CaBundle,
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

func requireCaBundleStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireCaBundlePEMFingerprintTag(t *testing.T, tags map[string]string, caBundlePEM string) {
	t.Helper()
	want := caBundlePEMFingerprint(caBundlePEM)
	if got := tags[caBundlePEMFingerprintTag]; got != want {
		t.Fatalf("freeformTags[%q] = %q, want %q", caBundlePEMFingerprintTag, got, want)
	}
}

func requireCaBundleUserFreeformTags(t *testing.T, got map[string]string, want map[string]string) {
	t.Helper()
	filtered := caBundleStatusFreeformTags(got)
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("user freeformTags = %#v, want %#v", filtered, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
