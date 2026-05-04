/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package artifact

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testArtifactID       = "ocid1.marketplaceartifact.oc1..artifact"
	testArtifactOtherID  = "ocid1.marketplaceartifact.oc1..other"
	testArtifactCompID   = "ocid1.compartment.oc1..artifact"
	testArtifactSourceID = "ocid1.containerrepo.oc1..source"
)

type fakeArtifactOCIClient struct {
	createRequests      []marketplacepublishersdk.CreateArtifactRequest
	getRequests         []marketplacepublishersdk.GetArtifactRequest
	listRequests        []marketplacepublishersdk.ListArtifactsRequest
	updateRequests      []marketplacepublishersdk.UpdateArtifactRequest
	deleteRequests      []marketplacepublishersdk.DeleteArtifactRequest
	workRequestRequests []marketplacepublishersdk.GetWorkRequestRequest

	createFn      func(context.Context, marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error)
	getFn         func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error)
	listFn        func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error)
	updateFn      func(context.Context, marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error)
	deleteFn      func(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error)
	workRequestFn func(context.Context, marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error)
}

func (f *fakeArtifactOCIClient) CreateArtifact(
	ctx context.Context,
	request marketplacepublishersdk.CreateArtifactRequest,
) (marketplacepublishersdk.CreateArtifactResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return marketplacepublishersdk.CreateArtifactResponse{}, nil
}

func (f *fakeArtifactOCIClient) GetArtifact(
	ctx context.Context,
	request marketplacepublishersdk.GetArtifactRequest,
) (marketplacepublishersdk.GetArtifactResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return marketplacepublishersdk.GetArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "artifact not found")
}

func (f *fakeArtifactOCIClient) ListArtifacts(
	ctx context.Context,
	request marketplacepublishersdk.ListArtifactsRequest,
) (marketplacepublishersdk.ListArtifactsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return marketplacepublishersdk.ListArtifactsResponse{}, nil
}

func (f *fakeArtifactOCIClient) UpdateArtifact(
	ctx context.Context,
	request marketplacepublishersdk.UpdateArtifactRequest,
) (marketplacepublishersdk.UpdateArtifactResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return marketplacepublishersdk.UpdateArtifactResponse{}, nil
}

func (f *fakeArtifactOCIClient) DeleteArtifact(
	ctx context.Context,
	request marketplacepublishersdk.DeleteArtifactRequest,
) (marketplacepublishersdk.DeleteArtifactResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return marketplacepublishersdk.DeleteArtifactResponse{}, nil
}

func (f *fakeArtifactOCIClient) GetWorkRequest(
	ctx context.Context,
	request marketplacepublishersdk.GetWorkRequestRequest,
) (marketplacepublishersdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return marketplacepublishersdk.GetWorkRequestResponse{}, nil
}

func newArtifactTestClient(fake *fakeArtifactOCIClient) ArtifactServiceClient {
	return newArtifactServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func newArtifactResource() *marketplacepublisherv1beta1.Artifact {
	return &marketplacepublisherv1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "artifact",
			Namespace: "default",
			UID:       types.UID("artifact-uid"),
		},
		Spec: marketplacepublisherv1beta1.ArtifactSpec{
			CompartmentId: testArtifactCompID,
			DisplayName:   "artifact",
			ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumContainerImage),
			ContainerImage: marketplacepublisherv1beta1.ArtifactContainerImage{
				SourceRegistryId:  testArtifactSourceID,
				SourceRegistryUrl: "iad.ocir.io/example/image:1.0.0",
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func newHelmChartArtifactSpec() marketplacepublisherv1beta1.ArtifactSpec {
	return marketplacepublisherv1beta1.ArtifactSpec{
		CompartmentId: testArtifactCompID,
		DisplayName:   "helm-artifact",
		ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumHelmChart),
		HelmChart: marketplacepublisherv1beta1.ArtifactHelmChart{
			SourceRegistryId:            testArtifactSourceID,
			SourceRegistryUrl:           "iad.ocir.io/example/chart:1.0.0",
			SupportedKubernetesVersions: []string{"v1.29.1", "v1.30.0"},
		},
		ContainerImageArtifactIds: []string{"ocid1.marketplaceartifact.oc1..image1"},
	}
}

func newStackArtifactSpec() marketplacepublisherv1beta1.ArtifactSpec {
	return marketplacepublisherv1beta1.ArtifactSpec{
		CompartmentId: testArtifactCompID,
		DisplayName:   "stack-artifact",
		ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumStack),
		Stack: marketplacepublisherv1beta1.ArtifactStack{
			SourceStackId:   "ocid1.ormstack.oc1..source",
			ImageListingIds: []string{"ocid1.marketplacelisting.oc1..image"},
		},
	}
}

func newMachineImageArtifactSpec() marketplacepublisherv1beta1.ArtifactSpec {
	return marketplacepublisherv1beta1.ArtifactSpec{
		CompartmentId: testArtifactCompID,
		DisplayName:   "machine-image-artifact",
		ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumMachineImage),
		MachineImage: marketplacepublisherv1beta1.ArtifactMachineImage{
			SourceImageId:     "ocid1.image.oc1..source",
			IsSnapshotAllowed: false,
			Username:          "opc",
			ImageShapeCompatibilityEntries: []marketplacepublisherv1beta1.ArtifactMachineImageImageShapeCompatibilityEntry{
				{
					Shape: "VM.Standard.E4.Flex",
					MemoryConstraints: marketplacepublisherv1beta1.ArtifactMachineImageImageShapeCompatibilityEntryMemoryConstraints{
						MinInGBs: 8,
						MaxInGBs: 32,
					},
					OcpuConstraints: marketplacepublisherv1beta1.ArtifactMachineImageImageShapeCompatibilityEntryOcpuConstraints{
						Min: 1,
						Max: 4,
					},
				},
			},
		},
	}
}

func newExistingArtifactResource(id string) *marketplacepublisherv1beta1.Artifact {
	resource := newArtifactResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func sdkArtifactFromSpec(
	id string,
	spec marketplacepublisherv1beta1.ArtifactSpec,
	state marketplacepublishersdk.ArtifactLifecycleStateEnum,
) marketplacepublishersdk.ContainerImageArtifact {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.ContainerImageArtifact{
		Id:             common.String(id),
		DisplayName:    common.String(spec.DisplayName),
		TimeCreated:    &now,
		CompartmentId:  common.String(spec.CompartmentId),
		PublisherId:    common.String("ocid1.publisher.oc1..artifact"),
		TimeUpdated:    &now,
		ContainerImage: sdkContainerImageFromSpec(spec.ContainerImage),
		FreeformTags:   artifactCloneStringMap(spec.FreeformTags),
		DefinedTags:    artifactDefinedTagsFromSpec(spec.DefinedTags),
		Status:         marketplacepublishersdk.ArtifactStatusAvailable,
		LifecycleState: state,
	}
}

func sdkContainerImageFromSpec(
	spec marketplacepublisherv1beta1.ArtifactContainerImage,
) *marketplacepublishersdk.ContainerImageDetails {
	return &marketplacepublishersdk.ContainerImageDetails{
		SourceRegistryId:  common.String(spec.SourceRegistryId),
		SourceRegistryUrl: common.String(spec.SourceRegistryUrl),
		ValidationStatus:  marketplacepublishersdk.ValidationStatusValidationCompleted,
		PublicationStatus: marketplacepublishersdk.PublicationStatusPublicationCompleted,
	}
}

func sdkArtifactSummaryFromSpec(
	id string,
	spec marketplacepublisherv1beta1.ArtifactSpec,
	state marketplacepublishersdk.ArtifactLifecycleStateEnum,
) marketplacepublishersdk.ArtifactSummary {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.ArtifactSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		ArtifactType:   marketplacepublishersdk.ArtifactTypeEnumContainerImage,
		LifecycleState: state,
		Status:         marketplacepublishersdk.ArtifactStatusAvailable,
		TimeCreated:    &now,
		TimeUpdated:    &now,
		FreeformTags:   artifactCloneStringMap(spec.FreeformTags),
		DefinedTags:    artifactDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func artifactWorkRequest(
	id string,
	operation marketplacepublishersdk.OperationTypeEnum,
	status marketplacepublishersdk.OperationStatusEnum,
	action marketplacepublishersdk.ActionTypeEnum,
	artifactID string,
) marketplacepublishersdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := marketplacepublishersdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if artifactID != "" {
		workRequest.Resources = []marketplacepublishersdk.WorkRequestResource{
			{
				EntityType: common.String("artifact"),
				ActionType: action,
				Identifier: common.String(artifactID),
				EntityUri:  common.String("/20220901/artifacts/" + artifactID),
			},
		}
	}
	return workRequest
}

func TestArtifactRuntimeSemanticsEncodesWorkRequestAndDeleteContracts(t *testing.T) {
	t.Parallel()

	got := reviewedArtifactRuntimeSemantics()
	if got.FormalService != "marketplacepublisher" || got.FormalSlug != "artifact" {
		t.Fatalf("formal identity = %s/%s, want marketplacepublisher/artifact", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	assertArtifactStrings(t, "work request phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v follow-up %#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertArtifactStrings(t, "list match fields", got.List.MatchFields, []string{"compartmentId", "displayName", "artifactType", "id"})
	assertArtifactStrings(t, "force-new fields", got.Mutation.ForceNew, []string{"compartmentId", "artifactType"})
}

func TestArtifactCreateUsesContainerImageBodyAndTracksWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newArtifactResource()
	workRequestStatus := marketplacepublishersdk.OperationStatusInProgress
	createCalls := 0
	getCalls := 0
	var createRequest marketplacepublishersdk.CreateArtifactRequest

	client := newArtifactTestClient(&fakeArtifactOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
			requireArtifactStringPtr(t, "list compartmentId", request.CompartmentId, testArtifactCompID)
			requireArtifactStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			return marketplacepublishersdk.ListArtifactsResponse{}, nil
		},
		createFn: func(_ context.Context, request marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error) {
			createCalls++
			createRequest = request
			return marketplacepublishersdk.CreateArtifactResponse{
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error) {
			requireArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
			action := marketplacepublishersdk.ActionTypeInProgress
			if workRequestStatus == marketplacepublishersdk.OperationStatusSucceeded {
				action = marketplacepublishersdk.ActionTypeCreated
			}
			return marketplacepublishersdk.GetWorkRequestResponse{
				WorkRequest: artifactWorkRequest(
					"wr-create",
					marketplacepublishersdk.OperationTypeCreateArtifact,
					workRequestStatus,
					action,
					testArtifactID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			getCalls++
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.GetArtifactResponse{
				Artifact: sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireArtifactNoError(t, "CreateOrUpdate()", err)
	requireArtifactRequeueResponse(t, response, true, "while work request is pending")
	requireArtifactCreateRequest(t, createRequest, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireArtifactAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)

	workRequestStatus = marketplacepublishersdk.OperationStatusSucceeded
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireArtifactNoError(t, "CreateOrUpdate() after work request success", err)
	requireArtifactRequeueResponse(t, response, false, "after completed create work request")
	requireArtifactCallCount(t, "CreateArtifact()", createCalls, 1)
	requireArtifactCallCount(t, "GetArtifact()", getCalls, 1)
	requireArtifactIdentity(t, resource, testArtifactID)
	if got := resource.Status.Status; got != string(marketplacepublishersdk.ArtifactStatusAvailable) {
		t.Fatalf("status.sdkStatus = %q, want AVAILABLE", got)
	}
	requireArtifactCondition(t, resource, shared.Active)
	requireArtifactNoCurrentAsync(t, resource)
}

func TestArtifactCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := newArtifactResource()
	otherSpec := resource.Spec
	otherSpec.DisplayName = "other"
	fake := &fakeArtifactOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
			switch artifactString(request.Page) {
			case "":
				return marketplacepublishersdk.ListArtifactsResponse{
					ArtifactCollection: marketplacepublishersdk.ArtifactCollection{Items: []marketplacepublishersdk.ArtifactSummary{
						sdkArtifactSummaryFromSpec(testArtifactOtherID, otherSpec, marketplacepublishersdk.ArtifactLifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return marketplacepublishersdk.ListArtifactsResponse{
					ArtifactCollection: marketplacepublishersdk.ArtifactCollection{Items: []marketplacepublishersdk.ArtifactSummary{
						sdkArtifactSummaryFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
					}},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", artifactString(request.Page))
				return marketplacepublishersdk.ListArtifactsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.GetArtifactResponse{
				Artifact: sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error) {
			t.Fatal("CreateArtifact() should not be called when list resolves an existing artifact")
			return marketplacepublishersdk.CreateArtifactResponse{}, nil
		},
	}
	client := newArtifactTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireArtifactNoError(t, "CreateOrUpdate()", err)
	requireArtifactRequeueResponse(t, response, false, "after binding existing artifact")
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListArtifacts() calls = %d, want 2 paginated calls", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateArtifact() calls = %d, want 0", len(fake.createRequests))
	}
	requireArtifactIdentity(t, resource, testArtifactID)
	if got := resource.Status.Status; got != string(marketplacepublishersdk.ArtifactStatusAvailable) {
		t.Fatalf("status.sdkStatus = %q, want AVAILABLE", got)
	}
}

func TestArtifactCreateOrUpdateSkipsUpdateWhenReadbackHasOnlyComputedNestedFields(t *testing.T) {
	t.Parallel()

	resource := newExistingArtifactResource(testArtifactID)
	client := newArtifactTestClient(&fakeArtifactOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.GetArtifactResponse{
				Artifact: sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error) {
			t.Fatal("UpdateArtifact() should not be called when desired and mutable readback match")
			return marketplacepublishersdk.UpdateArtifactResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireArtifactNoError(t, "CreateOrUpdate()", err)
	requireArtifactRequeueResponse(t, response, false, "after no-op readback")
}

func TestArtifactCreateOrUpdateUpdatesMutableFieldsAndCompletesWorkRequest(t *testing.T) {
	t.Parallel()

	original := newArtifactResource()
	resource := newExistingArtifactResource(testArtifactID)
	resource.Spec.DisplayName = "artifact-updated"
	resource.Spec.ContainerImage.SourceRegistryUrl = "iad.ocir.io/example/image:2.0.0"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	getCalls := 0
	var updateRequest marketplacepublishersdk.UpdateArtifactRequest
	client := newArtifactTestClient(&fakeArtifactOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			getCalls++
			spec := original.Spec
			if getCalls > 1 {
				spec = resource.Spec
			}
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.GetArtifactResponse{
				Artifact: sdkArtifactFromSpec(testArtifactID, spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error) {
			updateRequest = request
			return marketplacepublishersdk.UpdateArtifactResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error) {
			requireArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update")
			return marketplacepublishersdk.GetWorkRequestResponse{
				WorkRequest: artifactWorkRequest(
					"wr-update",
					marketplacepublishersdk.OperationTypeUpdateArtifact,
					marketplacepublishersdk.OperationStatusSucceeded,
					marketplacepublishersdk.ActionTypeUpdated,
					testArtifactID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireArtifactNoError(t, "CreateOrUpdate()", err)
	requireArtifactRequeueResponse(t, response, false, "after completed update work request")
	requireArtifactStringPtr(t, "update artifactId", updateRequest.ArtifactId, testArtifactID)
	details, ok := updateRequest.UpdateArtifactDetails.(marketplacepublishersdk.UpdateContainerImageArtifactDetails)
	if !ok {
		t.Fatalf("update details type = %T, want UpdateContainerImageArtifactDetails", updateRequest.UpdateArtifactDetails)
	}
	if details.CompartmentId != nil {
		t.Fatalf("update compartmentId = %q, want nil because compartmentId is create-only", *details.CompartmentId)
	}
	requireArtifactStringPtr(t, "update displayName", details.DisplayName, resource.Spec.DisplayName)
	if details.ContainerImage == nil {
		t.Fatal("update containerImage = nil, want updated container image")
	}
	requireArtifactStringPtr(t, "update sourceRegistryUrl", details.ContainerImage.SourceRegistryUrl, resource.Spec.ContainerImage.SourceRegistryUrl)
	if got := details.FreeformTags["env"]; got != "prod" {
		t.Fatalf("update freeformTags[env] = %q, want prod", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireArtifactCondition(t, resource, shared.Active)
}

func TestArtifactCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newExistingArtifactResource(testArtifactID)
	current := sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive)
	current.CompartmentId = common.String("ocid1.compartment.oc1..different")
	fake := &fakeArtifactOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			return marketplacepublishersdk.GetArtifactResponse{Artifact: current}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error) {
			t.Fatal("UpdateArtifact() should not be called when compartmentId drifts")
			return marketplacepublishersdk.UpdateArtifactResponse{}, nil
		},
	}
	client := newArtifactTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId detail", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateArtifact() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestArtifactCreateOrUpdateRejectsJsonDataBeforeOCICalls(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		resource *marketplacepublisherv1beta1.Artifact
	}{
		{
			name: "create",
			resource: func() *marketplacepublisherv1beta1.Artifact {
				resource := newArtifactResource()
				resource.Spec.JsonData = `{"displayName":"json-artifact"}`
				return resource
			}(),
		},
		{
			name: "update",
			resource: func() *marketplacepublisherv1beta1.Artifact {
				resource := newExistingArtifactResource(testArtifactID)
				resource.Spec.JsonData = `{"displayName":"json-artifact"}`
				return resource
			}(),
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeArtifactOCIClient{
				listFn: func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
					t.Fatal("ListArtifacts() should not be called when spec.jsonData is rejected")
					return marketplacepublishersdk.ListArtifactsResponse{}, nil
				},
				getFn: func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
					t.Fatal("GetArtifact() should not be called when spec.jsonData is rejected")
					return marketplacepublishersdk.GetArtifactResponse{}, nil
				},
				createFn: func(context.Context, marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error) {
					t.Fatal("CreateArtifact() should not be called when spec.jsonData is rejected")
					return marketplacepublishersdk.CreateArtifactResponse{}, nil
				},
				updateFn: func(context.Context, marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error) {
					t.Fatal("UpdateArtifact() should not be called when spec.jsonData is rejected")
					return marketplacepublishersdk.UpdateArtifactResponse{}, nil
				},
			}
			client := newArtifactTestClient(fake)

			response, err := client.CreateOrUpdate(context.Background(), tt.resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want spec.jsonData rejection")
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
			}
			if !strings.Contains(err.Error(), "spec.jsonData") || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("CreateOrUpdate() error = %v, want spec.jsonData unsupported detail", err)
			}
			if len(fake.listRequests) != 0 || len(fake.getRequests) != 0 ||
				len(fake.createRequests) != 0 || len(fake.updateRequests) != 0 {
				t.Fatalf("OCI calls = list:%d get:%d create:%d update:%d, want none",
					len(fake.listRequests),
					len(fake.getRequests),
					len(fake.createRequests),
					len(fake.updateRequests),
				)
			}
		})
	}
}

func TestArtifactHelmChartCreateAndUpdateBodies(t *testing.T) {
	t.Parallel()

	createDetails, updateDetails := requireArtifactCreateAndUpdateDetails(t, newHelmChartArtifactSpec())
	createHelm, ok := createDetails.(marketplacepublishersdk.CreateKubernetesImageArtifactDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateKubernetesImageArtifactDetails", createDetails)
	}
	if createHelm.HelmChart == nil {
		t.Fatal("create helmChart = nil, want helm chart details")
	}
	requireArtifactStringPtr(t, "create helm sourceRegistryId", createHelm.HelmChart.SourceRegistryId, testArtifactSourceID)
	requireArtifactStringPtr(t, "create helm sourceRegistryUrl", createHelm.HelmChart.SourceRegistryUrl, "iad.ocir.io/example/chart:1.0.0")
	assertArtifactStrings(t, "create helm supportedKubernetesVersions", createHelm.HelmChart.SupportedKubernetesVersions, []string{"v1.29.1", "v1.30.0"})
	assertArtifactStrings(t, "create helm containerImageArtifactIds", createHelm.ContainerImageArtifactIds, []string{"ocid1.marketplaceartifact.oc1..image1"})

	updateHelm, ok := updateDetails.(marketplacepublishersdk.UpdateKubernetesImageArtifactDetails)
	if !ok {
		t.Fatalf("update details type = %T, want UpdateKubernetesImageArtifactDetails", updateDetails)
	}
	if updateHelm.CompartmentId != nil {
		t.Fatalf("update helm compartmentId = %q, want nil", *updateHelm.CompartmentId)
	}
	if updateHelm.HelmChart == nil {
		t.Fatal("update helmChart = nil, want helm chart details")
	}
	requireArtifactStringPtr(t, "update helm sourceRegistryId", updateHelm.HelmChart.SourceRegistryId, testArtifactSourceID)
	requireArtifactStringPtr(t, "update helm sourceRegistryUrl", updateHelm.HelmChart.SourceRegistryUrl, "iad.ocir.io/example/chart:1.0.0")
	assertArtifactStrings(t, "update helm containerImageArtifactIds", updateHelm.ContainerImageArtifactIds, []string{"ocid1.marketplaceartifact.oc1..image1"})
}

func TestArtifactStackCreateAndUpdateBodies(t *testing.T) {
	t.Parallel()

	createDetails, updateDetails := requireArtifactCreateAndUpdateDetails(t, newStackArtifactSpec())
	createStack, ok := createDetails.(marketplacepublishersdk.CreateStackArtifactDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateStackArtifactDetails", createDetails)
	}
	if createStack.Stack == nil {
		t.Fatal("create stack = nil, want stack details")
	}
	requireArtifactStringPtr(t, "create stack sourceStackId", createStack.Stack.SourceStackId, "ocid1.ormstack.oc1..source")
	assertArtifactStrings(t, "create stack imageListingIds", createStack.Stack.ImageListingIds, []string{"ocid1.marketplacelisting.oc1..image"})

	updateStack, ok := updateDetails.(marketplacepublishersdk.UpdateStackArtifactDetails)
	if !ok {
		t.Fatalf("update details type = %T, want UpdateStackArtifactDetails", updateDetails)
	}
	if updateStack.CompartmentId != nil {
		t.Fatalf("update stack compartmentId = %q, want nil", *updateStack.CompartmentId)
	}
	if updateStack.Stack == nil {
		t.Fatal("update stack = nil, want stack details")
	}
	requireArtifactStringPtr(t, "update stack sourceStackId", updateStack.Stack.SourceStackId, "ocid1.ormstack.oc1..source")
	assertArtifactStrings(t, "update stack imageListingIds", updateStack.Stack.ImageListingIds, []string{"ocid1.marketplacelisting.oc1..image"})
}

func TestArtifactMachineImageCreateAndUpdateBodies(t *testing.T) {
	t.Parallel()

	createDetails, updateDetails := requireArtifactCreateAndUpdateDetails(t, newMachineImageArtifactSpec())
	createImage, ok := createDetails.(marketplacepublishersdk.CreateMachineImageArtifactDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateMachineImageArtifactDetails", createDetails)
	}
	if createImage.MachineImage == nil {
		t.Fatal("create machineImage = nil, want machine image details")
	}
	requireArtifactStringPtr(t, "create machine sourceImageId", createImage.MachineImage.SourceImageId, "ocid1.image.oc1..source")
	requireArtifactBoolPtr(t, "create machine isSnapshotAllowed", createImage.MachineImage.IsSnapshotAllowed, false)
	requireArtifactShapeCompatibility(t, "create machine", createImage.MachineImage.ImageShapeCompatibilityEntries)

	updateImage, ok := updateDetails.(marketplacepublishersdk.UpdateMachineImageArtifactDetails)
	if !ok {
		t.Fatalf("update details type = %T, want UpdateMachineImageArtifactDetails", updateDetails)
	}
	if updateImage.CompartmentId != nil {
		t.Fatalf("update machine compartmentId = %q, want nil", *updateImage.CompartmentId)
	}
	if updateImage.MachineImage == nil {
		t.Fatal("update machineImage = nil, want machine image details")
	}
	requireArtifactStringPtr(t, "update machine sourceImageId", updateImage.MachineImage.SourceImageId, "ocid1.image.oc1..source")
	requireArtifactBoolPtr(t, "update machine isSnapshotAllowed", updateImage.MachineImage.IsSnapshotAllowed, false)
	requireArtifactShapeCompatibility(t, "update machine", updateImage.MachineImage.ImageShapeCompatibilityEntries)
}

func TestArtifactPolymorphicCreateRequiresTypeSpecificFields(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		spec marketplacepublisherv1beta1.ArtifactSpec
		want string
	}{
		{
			name: "helm chart source registry url",
			spec: marketplacepublisherv1beta1.ArtifactSpec{
				CompartmentId: testArtifactCompID,
				ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumHelmChart),
				HelmChart: marketplacepublisherv1beta1.ArtifactHelmChart{
					SourceRegistryId: testArtifactSourceID,
				},
			},
			want: "helmChart.sourceRegistryUrl",
		},
		{
			name: "stack source stack id",
			spec: marketplacepublisherv1beta1.ArtifactSpec{
				CompartmentId: testArtifactCompID,
				ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumStack),
				Stack:         marketplacepublisherv1beta1.ArtifactStack{},
			},
			want: "stack.sourceStackId",
		},
		{
			name: "machine image shape compatibility",
			spec: marketplacepublisherv1beta1.ArtifactSpec{
				CompartmentId: testArtifactCompID,
				ArtifactType:  string(marketplacepublishersdk.ArtifactTypeEnumMachineImage),
				MachineImage: marketplacepublisherv1beta1.ArtifactMachineImage{
					SourceImageId:     "ocid1.image.oc1..source",
					IsSnapshotAllowed: true,
				},
			},
			want: "machineImage.imageShapeCompatibilityEntries",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := artifactCreateDetailsFromSpec(tt.spec)
			if err == nil {
				t.Fatal("artifactCreateDetailsFromSpec() error = nil, want required-field failure")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("artifactCreateDetailsFromSpec() error = %v, want %s", err, tt.want)
			}
		})
	}
}

func TestArtifactDeletePollsWorkRequestAndConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := newExistingArtifactResource(testArtifactID)
	getCalls := 0
	client := newArtifactTestClient(&fakeArtifactOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			getCalls++
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			if getCalls == 1 {
				return marketplacepublishersdk.GetArtifactResponse{
					Artifact: sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(_ context.Context, request marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			requireArtifactStringPtr(t, "delete artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.DeleteArtifactResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error) {
			requireArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return marketplacepublishersdk.GetWorkRequestResponse{
				WorkRequest: artifactWorkRequest(
					"wr-delete",
					marketplacepublishersdk.OperationTypeDeleteArtifact,
					marketplacepublishersdk.OperationStatusSucceeded,
					marketplacepublishersdk.ActionTypeDeleted,
					testArtifactID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after work request and not-found confirmation")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestArtifactDeleteWithoutRecordedOCIDConfirmsNoMatchByList(t *testing.T) {
	t.Parallel()

	resource := newArtifactResource()
	fake := &fakeArtifactOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
			requireArtifactStringPtr(t, "list compartmentId", request.CompartmentId, testArtifactCompID)
			requireArtifactStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			return marketplacepublishersdk.ListArtifactsResponse{}, nil
		},
		getFn: func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			t.Fatal("GetArtifact() should not be called when delete confirms missing artifact by list")
			return marketplacepublishersdk.GetArtifactResponse{}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			t.Fatal("DeleteArtifact() should not be called when delete confirms missing artifact by list")
			return marketplacepublishersdk.DeleteArtifactResponse{}, nil
		},
	}
	client := newArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after no-match list confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireArtifactCondition(t, resource, shared.Terminating)
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListArtifacts() calls = %d, want 1", len(fake.listRequests))
	}
	if len(fake.getRequests) != 0 || len(fake.deleteRequests) != 0 {
		t.Fatalf("OCI calls = get:%d delete:%d, want none", len(fake.getRequests), len(fake.deleteRequests))
	}
}

type artifactWriteWorkRequestDeleteCase struct {
	name  string
	phase shared.OSOKAsyncPhase
	op    marketplacepublishersdk.OperationTypeEnum
}

func artifactWriteWorkRequestDeleteCases() []artifactWriteWorkRequestDeleteCase {
	return []artifactWriteWorkRequestDeleteCase{
		{
			name:  "create",
			phase: shared.OSOKAsyncPhaseCreate,
			op:    marketplacepublishersdk.OperationTypeCreateArtifact,
		},
		{
			name:  "update",
			phase: shared.OSOKAsyncPhaseUpdate,
			op:    marketplacepublishersdk.OperationTypeUpdateArtifact,
		},
	}
}

func (tc artifactWriteWorkRequestDeleteCase) workRequestID() string {
	return "wr-" + tc.name
}

func TestArtifactDeleteWaitsForPendingCreateOrUpdateWorkRequestWithoutTrackedID(t *testing.T) {
	t.Parallel()

	for _, tc := range artifactWriteWorkRequestDeleteCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runArtifactPendingWriteDeleteCase(t, tc)
		})
	}
}

func runArtifactPendingWriteDeleteCase(t *testing.T, tc artifactWriteWorkRequestDeleteCase) {
	t.Helper()

	resource := newArtifactResource()
	seedArtifactCurrentWorkRequest(resource, tc.phase, tc.workRequestID())
	workRequestCalls := 0
	fake := newArtifactPendingWriteDeleteFake(t, tc, &workRequestCalls)
	client := newArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireArtifactNoError(t, "Delete()", err)
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while %s work request is pending", tc.phase)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1 pending %s poll", workRequestCalls, tc.phase)
	}
	requireArtifactNoDeleteOCIRequests(t, fake)
	requireArtifactPendingWriteStatus(t, resource, tc)
}

func newArtifactPendingWriteDeleteFake(
	t *testing.T,
	tc artifactWriteWorkRequestDeleteCase,
	workRequestCalls *int,
) *fakeArtifactOCIClient {
	t.Helper()

	return &fakeArtifactOCIClient{
		listFn: func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
			t.Fatal("ListArtifacts() should not be called while a write work request is pending")
			return marketplacepublishersdk.ListArtifactsResponse{}, nil
		},
		getFn: func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			t.Fatal("GetArtifact() should not be called while a write work request is pending")
			return marketplacepublishersdk.GetArtifactResponse{}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			t.Fatal("DeleteArtifact() should not be called while a write work request is pending")
			return marketplacepublishersdk.DeleteArtifactResponse{}, nil
		},
		workRequestFn: func(_ context.Context, request marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error) {
			(*workRequestCalls)++
			requireArtifactStringPtr(t, "workRequestId", request.WorkRequestId, tc.workRequestID())
			return marketplacepublishersdk.GetWorkRequestResponse{
				WorkRequest: artifactWorkRequest(
					tc.workRequestID(),
					tc.op,
					marketplacepublishersdk.OperationStatusInProgress,
					marketplacepublishersdk.ActionTypeInProgress,
					"",
				),
			}, nil
		},
	}
}

func requireArtifactNoDeleteOCIRequests(t *testing.T, fake *fakeArtifactOCIClient) {
	t.Helper()

	if len(fake.listRequests) != 0 || len(fake.getRequests) != 0 || len(fake.deleteRequests) != 0 {
		t.Fatalf("OCI calls = list:%d get:%d delete:%d, want only GetWorkRequest",
			len(fake.listRequests),
			len(fake.getRequests),
			len(fake.deleteRequests),
		)
	}
}

func requireArtifactPendingWriteStatus(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Artifact,
	tc artifactWriteWorkRequestDeleteCase,
) {
	t.Helper()

	current := requireArtifactAsyncCurrent(t, resource, tc.phase, tc.workRequestID())
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if !strings.Contains(current.Message, "waiting before delete") {
		t.Fatalf("status.async.current.message = %q, want waiting-before-delete detail", current.Message)
	}
	if got := artifactTrackedID(resource); got != "" {
		t.Fatalf("tracked artifact ID = %q, want empty while pending write has not resolved identity", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set, want finalizer retained while pending write is unresolved")
	}
}

func TestArtifactDeleteWithTrackedIDIgnoresCreateOnlySpecValidation(t *testing.T) {
	t.Parallel()

	resource := newExistingArtifactResource(testArtifactID)
	resource.Spec.JsonData = `{"displayName":"delete-only"}`
	resource.Spec.ArtifactType = "NOT_A_SUPPORTED_ARTIFACT_TYPE"
	getCalls := 0
	client := newArtifactTestClient(&fakeArtifactOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			getCalls++
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			if getCalls == 1 {
				return marketplacepublishersdk.GetArtifactResponse{
					Artifact: sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(_ context.Context, request marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			requireArtifactStringPtr(t, "delete artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.DeleteArtifactResponse{
				OpcWorkRequestId: common.String("wr-delete-invalid-spec"),
				OpcRequestId:     common.String("opc-delete-invalid-spec"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error) {
			requireArtifactStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete-invalid-spec")
			return marketplacepublishersdk.GetWorkRequestResponse{
				WorkRequest: artifactWorkRequest(
					"wr-delete-invalid-spec",
					marketplacepublishersdk.OperationTypeDeleteArtifact,
					marketplacepublishersdk.OperationStatusSucceeded,
					marketplacepublishersdk.ActionTypeDeleted,
					testArtifactID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	requireArtifactNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for tracked OCID despite invalid create/update spec")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-invalid-spec" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-invalid-spec", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestArtifactDeleteWithRecordedPathIdentityIgnoresCreateOnlySpecValidation(t *testing.T) {
	t.Parallel()

	resource := newArtifactResource()
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.ArtifactType = string(marketplacepublishersdk.ArtifactTypeEnumContainerImage)
	resource.Spec.JsonData = `{"displayName":"delete-only"}`
	resource.Spec.ArtifactType = "NOT_A_SUPPORTED_ARTIFACT_TYPE"
	fake := &fakeArtifactOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
			requireArtifactStringPtr(t, "list compartmentId", request.CompartmentId, testArtifactCompID)
			requireArtifactStringPtr(t, "list displayName", request.DisplayName, resource.Status.DisplayName)
			return marketplacepublishersdk.ListArtifactsResponse{}, nil
		},
		getFn: func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			t.Fatal("GetArtifact() should not be called when recorded path identity confirms no matching artifact")
			return marketplacepublishersdk.GetArtifactResponse{}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			t.Fatal("DeleteArtifact() should not be called when recorded path identity confirms no matching artifact")
			return marketplacepublishersdk.DeleteArtifactResponse{}, nil
		},
	}
	client := newArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireArtifactNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after recorded path identity confirms no match")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListArtifacts() calls = %d, want 1", len(fake.listRequests))
	}
}

func TestArtifactDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := newExistingArtifactResource(testArtifactID)
	fake := &fakeArtifactOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			return marketplacepublishersdk.GetArtifactResponse{
				Artifact: sdkArtifactFromSpec(testArtifactID, resource.Spec, marketplacepublishersdk.ArtifactLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			return marketplacepublishersdk.DeleteArtifactResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
		},
	}
	client := newArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not-found")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteArtifact() calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestArtifactDeleteTreatsPreReadAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := newExistingArtifactResource(testArtifactID)
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous pre-read")
	serviceErr.OpcRequestID = "opc-pre-delete-auth"
	fake := &fakeArtifactOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
			requireArtifactStringPtr(t, "get artifactId", request.ArtifactId, testArtifactID)
			return marketplacepublishersdk.GetArtifactResponse{}, serviceErr
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
			t.Fatal("DeleteArtifact() should not be called after ambiguous pre-delete read")
			return marketplacepublishersdk.DeleteArtifactResponse{}, nil
		},
	}
	client := newArtifactTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-read to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteArtifact() calls = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-pre-delete-auth" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-pre-delete-auth", got)
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Delete() error = %v, want ambiguous detail", err)
	}
}

func TestArtifactCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := newArtifactResource()
	client := newArtifactTestClient(&fakeArtifactOCIClient{
		createFn: func(context.Context, marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error) {
			return marketplacepublishersdk.CreateArtifactResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "conflict")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireArtifactCondition(t, resource, shared.Failed)
}

func requireArtifactNoError(t *testing.T, label string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s error = %v", label, err)
	}
}

func requireArtifactRequeueResponse(
	t *testing.T,
	response servicemanager.OSOKResponse,
	shouldRequeue bool,
	context string,
) {
	t.Helper()

	if !response.IsSuccessful || response.ShouldRequeue != shouldRequeue {
		t.Fatalf("response = %#v, want successful requeue=%t %s", response, shouldRequeue, context)
	}
}

func requireArtifactCallCount(t *testing.T, label string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", label, got, want)
	}
}

func requireArtifactCreateRequest(
	t *testing.T,
	request marketplacepublishersdk.CreateArtifactRequest,
	resource *marketplacepublisherv1beta1.Artifact,
) {
	t.Helper()

	details, ok := request.CreateArtifactDetails.(marketplacepublishersdk.CreateContainerImageArtifactDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateContainerImageArtifactDetails", request.CreateArtifactDetails)
	}
	requireArtifactStringPtr(t, "create compartmentId", details.CompartmentId, testArtifactCompID)
	requireArtifactStringPtr(t, "create displayName", details.DisplayName, resource.Spec.DisplayName)
	if details.ContainerImage == nil {
		t.Fatal("create containerImage = nil, want container image details")
	}
	requireArtifactStringPtr(t, "create sourceRegistryId", details.ContainerImage.SourceRegistryId, resource.Spec.ContainerImage.SourceRegistryId)
	requireArtifactStringPtr(t, "create sourceRegistryUrl", details.ContainerImage.SourceRegistryUrl, resource.Spec.ContainerImage.SourceRegistryUrl)
	requireArtifactStringPtr(t, "create opc retry token", request.OpcRetryToken, string(resource.UID))
}

func requireArtifactCreateAndUpdateDetails(
	t *testing.T,
	spec marketplacepublisherv1beta1.ArtifactSpec,
) (marketplacepublishersdk.CreateArtifactDetails, marketplacepublishersdk.UpdateArtifactDetails) {
	t.Helper()

	createDetails, err := artifactCreateDetailsFromSpec(spec)
	requireArtifactNoError(t, "artifactCreateDetailsFromSpec()", err)

	resource := &marketplacepublisherv1beta1.Artifact{Spec: spec}
	updateDetails, err := artifactUpdateDetailsFromDesired(resource)
	requireArtifactNoError(t, "artifactUpdateDetailsFromDesired()", err)
	return createDetails, updateDetails
}

func requireArtifactIdentity(t *testing.T, resource *marketplacepublisherv1beta1.Artifact, want string) {
	t.Helper()

	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func requireArtifactAsync(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Artifact,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Phase != phase || current.WorkRequestID != workRequestID || current.NormalizedClass != class {
		t.Fatalf("status.async.current = %#v, want phase=%q workRequestID=%q class=%q", current, phase, workRequestID, class)
	}
}

func requireArtifactAsyncCurrent(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Artifact,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) *shared.OSOKAsyncOperation {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Phase != phase || current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current = %#v, want phase=%q workRequestID=%q", current, phase, workRequestID)
	}
	return current
}

func requireArtifactNoCurrentAsync(t *testing.T, resource *marketplacepublisherv1beta1.Artifact) {
	t.Helper()

	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil", resource.Status.OsokStatus.Async.Current)
	}
}

func seedArtifactCurrentWorkRequest(
	resource *marketplacepublisherv1beta1.Artifact,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func requireArtifactCondition(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Artifact,
	want shared.OSOKConditionType,
) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want %q", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireArtifactStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if strings.TrimSpace(*got) != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireArtifactBoolPtr(t *testing.T, label string, got *bool, want bool) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %t", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", label, *got, want)
	}
}

func requireArtifactIntPtr(t *testing.T, label string, got *int, want int) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %d", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", label, *got, want)
	}
}

func requireArtifactShapeCompatibility(
	t *testing.T,
	label string,
	got []marketplacepublishersdk.ImageShapeCompatibility,
) {
	t.Helper()

	if len(got) != 1 {
		t.Fatalf("%s shape compatibility count = %d, want 1", label, len(got))
	}
	requireArtifactStringPtr(t, label+" shape", got[0].Shape, "VM.Standard.E4.Flex")
	if got[0].MemoryConstraints == nil {
		t.Fatalf("%s memoryConstraints = nil, want constraints", label)
	}
	requireArtifactIntPtr(t, label+" memory min", got[0].MemoryConstraints.MinInGBs, 8)
	requireArtifactIntPtr(t, label+" memory max", got[0].MemoryConstraints.MaxInGBs, 32)
	if got[0].OcpuConstraints == nil {
		t.Fatalf("%s ocpuConstraints = nil, want constraints", label)
	}
	requireArtifactIntPtr(t, label+" ocpu min", got[0].OcpuConstraints.Min, 1)
	requireArtifactIntPtr(t, label+" ocpu max", got[0].OcpuConstraints.Max, 4)
}

func assertArtifactStrings(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d (%v)", label, len(got), len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q (got %v)", label, i, got[i], want[i], got)
		}
	}
}
