package mediaasset

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	mediaservicessdk "github.com/oracle/oci-go-sdk/v65/mediaservices"
	mediaservicesv1beta1 "github.com/oracle/oci-service-operator/api/mediaservices/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const testMediaAssetID = "ocid1.mediaasset.oc1..runtime"

type fakeMediaAssetOCIClient struct {
	createFn func(context.Context, mediaservicessdk.CreateMediaAssetRequest) (mediaservicessdk.CreateMediaAssetResponse, error)
	getFn    func(context.Context, mediaservicessdk.GetMediaAssetRequest) (mediaservicessdk.GetMediaAssetResponse, error)
	listFn   func(context.Context, mediaservicessdk.ListMediaAssetsRequest) (mediaservicessdk.ListMediaAssetsResponse, error)
	updateFn func(context.Context, mediaservicessdk.UpdateMediaAssetRequest) (mediaservicessdk.UpdateMediaAssetResponse, error)
	deleteFn func(context.Context, mediaservicessdk.DeleteMediaAssetRequest) (mediaservicessdk.DeleteMediaAssetResponse, error)
}

func (f *fakeMediaAssetOCIClient) CreateMediaAsset(
	ctx context.Context,
	req mediaservicessdk.CreateMediaAssetRequest,
) (mediaservicessdk.CreateMediaAssetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return mediaservicessdk.CreateMediaAssetResponse{}, nil
}

func (f *fakeMediaAssetOCIClient) GetMediaAsset(
	ctx context.Context,
	req mediaservicessdk.GetMediaAssetRequest,
) (mediaservicessdk.GetMediaAssetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return mediaservicessdk.GetMediaAssetResponse{}, nil
}

func (f *fakeMediaAssetOCIClient) ListMediaAssets(
	ctx context.Context,
	req mediaservicessdk.ListMediaAssetsRequest,
) (mediaservicessdk.ListMediaAssetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return mediaservicessdk.ListMediaAssetsResponse{}, nil
}

func (f *fakeMediaAssetOCIClient) UpdateMediaAsset(
	ctx context.Context,
	req mediaservicessdk.UpdateMediaAssetRequest,
) (mediaservicessdk.UpdateMediaAssetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return mediaservicessdk.UpdateMediaAssetResponse{}, nil
}

func (f *fakeMediaAssetOCIClient) DeleteMediaAsset(
	ctx context.Context,
	req mediaservicessdk.DeleteMediaAssetRequest,
) (mediaservicessdk.DeleteMediaAssetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return mediaservicessdk.DeleteMediaAssetResponse{}, nil
}

func newMediaAssetTestClient(fake *fakeMediaAssetOCIClient) MediaAssetServiceClient {
	return newMediaAssetServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func newMediaAssetTestResource() *mediaservicesv1beta1.MediaAsset {
	return &mediaservicesv1beta1.MediaAsset{
		Spec: mediaservicesv1beta1.MediaAssetSpec{
			CompartmentId:              "ocid1.compartment.oc1..example",
			Type:                       "VIDEO",
			DisplayName:                "asset-alpha",
			SourceMediaWorkflowId:      "ocid1.mediaworkflow.oc1..workflow",
			MediaWorkflowJobId:         "ocid1.mediaworkflowjob.oc1..job",
			SourceMediaWorkflowVersion: 7,
			ParentMediaAssetId:         "ocid1.mediaasset.oc1..parent",
			MasterMediaAssetId:         "ocid1.mediaasset.oc1..master",
			BucketName:                 "media-bucket",
			NamespaceName:              "object-namespace",
			ObjectName:                 "video.mp4",
			ObjectEtag:                 "etag-value",
			SegmentRangeStartIndex:     1,
			SegmentRangeEndIndex:       10,
			Metadata: []mediaservicesv1beta1.MediaAssetMetadata{{
				Metadata: `{"codec":"h264"}`,
			}},
			MediaAssetTags: []mediaservicesv1beta1.MediaAssetTag{{
				Value: "ingest",
				Type:  "USER",
			}},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			Locks: []mediaservicesv1beta1.MediaAssetLock{{
				Type:              "DELETE",
				CompartmentId:     "ocid1.compartment.oc1..example",
				RelatedResourceId: "ocid1.locksource.oc1..example",
				Message:           "managed lock",
			}},
		},
	}
}

func trackMediaAsset(resource *mediaservicesv1beta1.MediaAsset, mediaAssetID string) {
	resource.Status.Id = mediaAssetID
	resource.Status.OsokStatus.Ocid = shared.OCID(mediaAssetID)
}

func observedMediaAssetFromSpec(
	id string,
	spec mediaservicesv1beta1.MediaAssetSpec,
	state mediaservicessdk.LifecycleStateEnum,
) mediaservicessdk.MediaAsset {
	current := mediaservicessdk.MediaAsset{
		Id:                         common.String(id),
		CompartmentId:              common.String(spec.CompartmentId),
		LifecycleState:             state,
		Type:                       mediaservicessdk.AssetTypeEnum(spec.Type),
		DisplayName:                optionalString(spec.DisplayName),
		SourceMediaWorkflowId:      optionalString(spec.SourceMediaWorkflowId),
		MediaWorkflowJobId:         optionalString(spec.MediaWorkflowJobId),
		ParentMediaAssetId:         optionalString(spec.ParentMediaAssetId),
		MasterMediaAssetId:         optionalString(spec.MasterMediaAssetId),
		BucketName:                 optionalString(spec.BucketName),
		NamespaceName:              optionalString(spec.NamespaceName),
		ObjectName:                 optionalString(spec.ObjectName),
		ObjectEtag:                 optionalString(spec.ObjectEtag),
		Metadata:                   mediaAssetMetadataFromSpec(spec.Metadata),
		MediaAssetTags:             mediaAssetTagsFromSpec(spec.MediaAssetTags),
		FreeformTags:               cloneStringMap(spec.FreeformTags),
		DefinedTags:                mediaAssetDefinedTagsFromSpec(spec.DefinedTags),
		Locks:                      mediaAssetLocksFromSpec(spec.Locks),
		SegmentRangeStartIndex:     optionalInt64(spec.SegmentRangeStartIndex),
		SegmentRangeEndIndex:       optionalInt64(spec.SegmentRangeEndIndex),
		SourceMediaWorkflowVersion: optionalInt64(spec.SourceMediaWorkflowVersion),
	}
	return current
}

func mediaAssetLocksFromSpec(spec []mediaservicesv1beta1.MediaAssetLock) []mediaservicessdk.ResourceLock {
	converted := make([]mediaservicessdk.ResourceLock, 0, len(spec))
	for _, lock := range spec {
		converted = append(converted, mediaservicessdk.ResourceLock{
			Type:              mediaservicessdk.ResourceLockTypeEnum(lock.Type),
			CompartmentId:     common.String(lock.CompartmentId),
			RelatedResourceId: optionalString(lock.RelatedResourceId),
			Message:           optionalString(lock.Message),
		})
	}
	return converted
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func optionalInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func mediaAssetSerializedRequestBody(t *testing.T, request any, method string, path string) string {
	t.Helper()

	ociRequest, ok := request.(interface {
		HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
	})
	if !ok {
		t.Fatalf("request type %T does not implement HTTPRequest", request)
	}

	httpRequest, err := ociRequest.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest(%T) error = %v", request, err)
	}
	if httpRequest.Body == nil {
		return ""
	}
	payload, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("io.ReadAll(%T body) error = %v", request, err)
	}
	return string(payload)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestApplyMediaAssetRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newMediaAssetDefaultRuntimeHooks(mediaservicessdk.MediaServicesClient{})
	applyMediaAssetRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if !reflect.DeepEqual(hooks.List.Fields, mediaAssetListFields()) {
		t.Fatalf("list fields = %#v, want %#v", hooks.List.Fields, mediaAssetListFields())
	}
	if !reflect.DeepEqual(hooks.Update.Fields, mediaAssetUpdateFields()) {
		t.Fatalf("update fields = %#v, want %#v", hooks.Update.Fields, mediaAssetUpdateFields())
	}
	if !reflect.DeepEqual(hooks.Delete.Fields, mediaAssetDeleteFields()) {
		t.Fatalf("delete fields = %#v, want %#v", hooks.Delete.Fields, mediaAssetDeleteFields())
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want create body sanitizer")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want guarded pre-create reuse")
	}
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("hooks.ParityHooks.NormalizeDesiredState = nil, want lock normalization hook")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if got := hooks.Semantics.DeleteFollowUp.Strategy; got != "confirm-delete" {
		t.Fatalf("delete follow-up = %q, want confirm-delete", got)
	}
	if !containsString(hooks.Semantics.Mutation.Mutable, "metadata") {
		t.Fatalf("mutable fields = %#v, want metadata in reviewed mutable surface", hooks.Semantics.Mutation.Mutable)
	}
	if containsString(hooks.Semantics.Mutation.ForceNew, "locks") {
		t.Fatalf("force-new fields = %#v, want locks handled by custom drift logic instead", hooks.Semantics.Mutation.ForceNew)
	}
}

func TestBuildMediaAssetCreateDetailsOmitsLockTimeCreated(t *testing.T) {
	t.Parallel()

	resource := newMediaAssetTestResource()
	resource.Spec.Locks[0].TimeCreated = "2026-05-07T12:00:00Z"

	details, err := buildMediaAssetCreateDetails(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildMediaAssetCreateDetails() error = %v", err)
	}
	if len(details.Locks) != 1 {
		t.Fatalf("create locks = %d, want 1", len(details.Locks))
	}
	if details.Locks[0].TimeCreated != nil {
		t.Fatalf("create lock timeCreated = %#v, want omitted from reviewed request", details.Locks[0].TimeCreated)
	}
}

func TestMediaAssetServiceClientUpdatesSupportedMutableDriftAndClearsEmptyValues(t *testing.T) {
	t.Parallel()

	original := newMediaAssetTestResource()
	resource := newMediaAssetTestResource()
	trackMediaAsset(resource, testMediaAssetID)
	resource.Spec.DisplayName = ""
	resource.Spec.ParentMediaAssetId = ""
	resource.Spec.MasterMediaAssetId = ""
	resource.Spec.Metadata = []mediaservicesv1beta1.MediaAssetMetadata{}
	resource.Spec.MediaAssetTags = []mediaservicesv1beta1.MediaAssetTag{}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	var updateRequest mediaservicessdk.UpdateMediaAssetRequest

	client := newMediaAssetTestClient(&fakeMediaAssetOCIClient{
		getFn: func(_ context.Context, req mediaservicessdk.GetMediaAssetRequest) (mediaservicessdk.GetMediaAssetResponse, error) {
			if req.MediaAssetId == nil || *req.MediaAssetId != testMediaAssetID {
				t.Fatalf("get mediaAssetId = %v, want %q", req.MediaAssetId, testMediaAssetID)
			}
			return mediaservicessdk.GetMediaAssetResponse{
				MediaAsset: observedMediaAssetFromSpec(
					testMediaAssetID,
					original.Spec,
					mediaservicessdk.LifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, req mediaservicessdk.UpdateMediaAssetRequest) (mediaservicessdk.UpdateMediaAssetResponse, error) {
			updateRequest = req
			return mediaservicessdk.UpdateMediaAssetResponse{
				OpcRequestId: common.String("opc-update-1"),
				MediaAsset: observedMediaAssetFromSpec(
					testMediaAssetID,
					resource.Spec,
					mediaservicessdk.LifecycleStateActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful update without requeue", response)
	}
	if updateRequest.MediaAssetId == nil || *updateRequest.MediaAssetId != testMediaAssetID {
		t.Fatalf("update mediaAssetId = %v, want %q", updateRequest.MediaAssetId, testMediaAssetID)
	}
	if updateRequest.IsLockOverride != nil {
		t.Fatalf("update isLockOverride = %#v, want reviewed hook field omission", updateRequest.IsLockOverride)
	}
	if updateRequest.UpdateMediaAssetDetails.DisplayName == nil || *updateRequest.UpdateMediaAssetDetails.DisplayName != "" {
		t.Fatalf("update displayName = %#v, want explicit empty-string clear", updateRequest.UpdateMediaAssetDetails.DisplayName)
	}
	if updateRequest.UpdateMediaAssetDetails.ParentMediaAssetId == nil || *updateRequest.UpdateMediaAssetDetails.ParentMediaAssetId != "" {
		t.Fatalf("update parentMediaAssetId = %#v, want explicit empty-string clear", updateRequest.UpdateMediaAssetDetails.ParentMediaAssetId)
	}
	if updateRequest.UpdateMediaAssetDetails.MasterMediaAssetId == nil || *updateRequest.UpdateMediaAssetDetails.MasterMediaAssetId != "" {
		t.Fatalf("update masterMediaAssetId = %#v, want explicit empty-string clear", updateRequest.UpdateMediaAssetDetails.MasterMediaAssetId)
	}
	if updateRequest.UpdateMediaAssetDetails.Metadata == nil || len(updateRequest.UpdateMediaAssetDetails.Metadata) != 0 {
		t.Fatalf("update metadata = %#v, want explicit empty slice clear", updateRequest.UpdateMediaAssetDetails.Metadata)
	}
	if updateRequest.UpdateMediaAssetDetails.MediaAssetTags == nil || len(updateRequest.UpdateMediaAssetDetails.MediaAssetTags) != 0 {
		t.Fatalf("update mediaAssetTags = %#v, want explicit empty slice clear", updateRequest.UpdateMediaAssetDetails.MediaAssetTags)
	}
	if updateRequest.UpdateMediaAssetDetails.FreeformTags == nil || len(updateRequest.UpdateMediaAssetDetails.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map clear", updateRequest.UpdateMediaAssetDetails.FreeformTags)
	}
	if updateRequest.UpdateMediaAssetDetails.DefinedTags == nil || len(updateRequest.UpdateMediaAssetDetails.DefinedTags) != 0 {
		t.Fatalf("update definedTags = %#v, want explicit empty map clear", updateRequest.UpdateMediaAssetDetails.DefinedTags)
	}

	body := mediaAssetSerializedRequestBody(t, mediaservicessdk.UpdateMediaAssetRequest{
		MediaAssetId:            common.String(testMediaAssetID),
		UpdateMediaAssetDetails: updateRequest.UpdateMediaAssetDetails,
	}, http.MethodPut, "/mediaAssets/"+testMediaAssetID)
	for _, want := range []string{
		`"displayName":""`,
		`"parentMediaAssetId":""`,
		`"masterMediaAssetId":""`,
		`"metadata":[]`,
		`"mediaAssetTags":[]`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("update body %s does not contain %s", body, want)
		}
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
}

func TestNormalizeMediaAssetDesiredStateClearsEquivalentLocks(t *testing.T) {
	t.Parallel()

	resource := newMediaAssetTestResource()
	current := observedMediaAssetFromSpec(testMediaAssetID, resource.Spec, mediaservicessdk.LifecycleStateActive)
	now := common.SDKTime{Time: time.Date(2026, time.May, 7, 12, 34, 56, 0, time.UTC)}
	current.Locks[0].TimeCreated = &now

	normalizeMediaAssetDesiredState(resource, current)
	if resource.Spec.Locks != nil {
		t.Fatalf("spec.locks = %#v, want normalized nil after equivalent OCI lock readback", resource.Spec.Locks)
	}
}

func TestValidateMediaAssetCreateOnlyDriftRejectsLockDrift(t *testing.T) {
	t.Parallel()

	resource := newMediaAssetTestResource()
	current := observedMediaAssetFromSpec(testMediaAssetID, resource.Spec, mediaservicessdk.LifecycleStateActive)
	current.Locks[0].Type = mediaservicessdk.ResourceLockTypeFull

	err := validateMediaAssetCreateOnlyDrift(resource, current)
	if err == nil || !strings.Contains(err.Error(), "locks") {
		t.Fatalf("validateMediaAssetCreateOnlyDrift() error = %v, want locks drift failure", err)
	}
}

func TestGuardMediaAssetExistingBeforeCreateRequiresStrongIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*mediaservicesv1beta1.MediaAsset)
		expected generatedruntime.ExistingBeforeCreateDecision
	}{
		{
			name: "skip without identity stronger than compartment and type",
			mutate: func(resource *mediaservicesv1beta1.MediaAsset) {
				resource.Spec.DisplayName = ""
				resource.Spec.BucketName = ""
				resource.Spec.ObjectName = ""
				resource.Spec.MediaWorkflowJobId = ""
				resource.Spec.SourceMediaWorkflowId = ""
				resource.Spec.SourceMediaWorkflowVersion = 0
				resource.Spec.ParentMediaAssetId = ""
				resource.Spec.MasterMediaAssetId = ""
			},
			expected: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name: "allow exact display name reuse",
			mutate: func(resource *mediaservicesv1beta1.MediaAsset) {
				resource.Spec.BucketName = ""
				resource.Spec.ObjectName = ""
				resource.Spec.MediaWorkflowJobId = ""
				resource.Spec.SourceMediaWorkflowId = ""
				resource.Spec.SourceMediaWorkflowVersion = 0
				resource.Spec.ParentMediaAssetId = ""
				resource.Spec.MasterMediaAssetId = ""
			},
			expected: generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
		{
			name: "allow object location reuse",
			mutate: func(resource *mediaservicesv1beta1.MediaAsset) {
				resource.Spec.DisplayName = ""
				resource.Spec.MediaWorkflowJobId = ""
				resource.Spec.SourceMediaWorkflowId = ""
				resource.Spec.SourceMediaWorkflowVersion = 0
				resource.Spec.ParentMediaAssetId = ""
				resource.Spec.MasterMediaAssetId = ""
			},
			expected: generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
		{
			name: "skip workflow job lookup without workflow id",
			mutate: func(resource *mediaservicesv1beta1.MediaAsset) {
				resource.Spec.DisplayName = ""
				resource.Spec.BucketName = ""
				resource.Spec.ObjectName = ""
				resource.Spec.ParentMediaAssetId = ""
				resource.Spec.MasterMediaAssetId = ""
				resource.Spec.SourceMediaWorkflowId = ""
			},
			expected: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := newMediaAssetTestResource()
			tc.mutate(resource)

			decision, err := guardMediaAssetExistingBeforeCreate(context.Background(), resource)
			if err != nil {
				t.Fatalf("guardMediaAssetExistingBeforeCreate() error = %v", err)
			}
			if decision != tc.expected {
				t.Fatalf("guardMediaAssetExistingBeforeCreate() = %q, want %q", decision, tc.expected)
			}
		})
	}
}

func TestMediaAssetServiceClientDeleteOmitsDeleteModeAndConfirmsLifecycleDelete(t *testing.T) {
	t.Parallel()

	resource := newMediaAssetTestResource()
	trackMediaAsset(resource, testMediaAssetID)

	getCalls := 0
	var deleteRequest mediaservicessdk.DeleteMediaAssetRequest

	client := newMediaAssetTestClient(&fakeMediaAssetOCIClient{
		getFn: func(_ context.Context, req mediaservicessdk.GetMediaAssetRequest) (mediaservicessdk.GetMediaAssetResponse, error) {
			getCalls++
			if req.MediaAssetId == nil || *req.MediaAssetId != testMediaAssetID {
				t.Fatalf("get mediaAssetId = %v, want %q", req.MediaAssetId, testMediaAssetID)
			}
			state := mediaservicessdk.LifecycleStateActive
			if getCalls > 1 {
				state = mediaservicessdk.LifecycleStateDeleting
			}
			return mediaservicessdk.GetMediaAssetResponse{
				MediaAsset: observedMediaAssetFromSpec(testMediaAssetID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req mediaservicessdk.DeleteMediaAssetRequest) (mediaservicessdk.DeleteMediaAssetResponse, error) {
			deleteRequest = req
			return mediaservicessdk.DeleteMediaAssetResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want in-progress delete while OCI still returns DELETING")
	}
	if getCalls != 2 {
		t.Fatalf("GetMediaAsset() calls = %d, want 2", getCalls)
	}
	if deleteRequest.MediaAssetId == nil || *deleteRequest.MediaAssetId != testMediaAssetID {
		t.Fatalf("delete mediaAssetId = %v, want %q", deleteRequest.MediaAssetId, testMediaAssetID)
	}
	if deleteRequest.IsLockOverride != nil {
		t.Fatalf("delete isLockOverride = %#v, want reviewed hook field omission", deleteRequest.IsLockOverride)
	}
	if deleteRequest.DeleteMode != "" {
		t.Fatalf("delete deleteMode = %q, want default omitted reviewed policy", deleteRequest.DeleteMode)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	}
}
