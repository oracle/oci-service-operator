package mediaworkflow

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

const testMediaWorkflowID = "ocid1.mediaworkflow.oc1..runtime"

type fakeMediaWorkflowOCIClient struct {
	createFn func(context.Context, mediaservicessdk.CreateMediaWorkflowRequest) (mediaservicessdk.CreateMediaWorkflowResponse, error)
	getFn    func(context.Context, mediaservicessdk.GetMediaWorkflowRequest) (mediaservicessdk.GetMediaWorkflowResponse, error)
	listFn   func(context.Context, mediaservicessdk.ListMediaWorkflowsRequest) (mediaservicessdk.ListMediaWorkflowsResponse, error)
	updateFn func(context.Context, mediaservicessdk.UpdateMediaWorkflowRequest) (mediaservicessdk.UpdateMediaWorkflowResponse, error)
	deleteFn func(context.Context, mediaservicessdk.DeleteMediaWorkflowRequest) (mediaservicessdk.DeleteMediaWorkflowResponse, error)
}

func (f *fakeMediaWorkflowOCIClient) CreateMediaWorkflow(
	ctx context.Context,
	req mediaservicessdk.CreateMediaWorkflowRequest,
) (mediaservicessdk.CreateMediaWorkflowResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return mediaservicessdk.CreateMediaWorkflowResponse{}, nil
}

func (f *fakeMediaWorkflowOCIClient) GetMediaWorkflow(
	ctx context.Context,
	req mediaservicessdk.GetMediaWorkflowRequest,
) (mediaservicessdk.GetMediaWorkflowResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return mediaservicessdk.GetMediaWorkflowResponse{}, nil
}

func (f *fakeMediaWorkflowOCIClient) ListMediaWorkflows(
	ctx context.Context,
	req mediaservicessdk.ListMediaWorkflowsRequest,
) (mediaservicessdk.ListMediaWorkflowsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return mediaservicessdk.ListMediaWorkflowsResponse{}, nil
}

func (f *fakeMediaWorkflowOCIClient) UpdateMediaWorkflow(
	ctx context.Context,
	req mediaservicessdk.UpdateMediaWorkflowRequest,
) (mediaservicessdk.UpdateMediaWorkflowResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return mediaservicessdk.UpdateMediaWorkflowResponse{}, nil
}

func (f *fakeMediaWorkflowOCIClient) DeleteMediaWorkflow(
	ctx context.Context,
	req mediaservicessdk.DeleteMediaWorkflowRequest,
) (mediaservicessdk.DeleteMediaWorkflowResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return mediaservicessdk.DeleteMediaWorkflowResponse{}, nil
}

func newMediaWorkflowTestClient(fake *fakeMediaWorkflowOCIClient) MediaWorkflowServiceClient {
	return newMediaWorkflowServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func mediaWorkflowJSONValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func newMediaWorkflowTestResource() *mediaservicesv1beta1.MediaWorkflow {
	return &mediaservicesv1beta1.MediaWorkflow{
		Spec: mediaservicesv1beta1.MediaWorkflowSpec{
			DisplayName:   "workflow-alpha",
			CompartmentId: "ocid1.compartment.oc1..example",
			Tasks: []mediaservicesv1beta1.MediaWorkflowTask{{
				Type:    "TRANSCODE_VIDEO",
				Version: 1,
				Key:     "transcode",
				Parameters: map[string]shared.JSONValue{
					"preset":      mediaWorkflowJSONValue(`"HD"`),
					"bitrateKbps": mediaWorkflowJSONValue(`6000`),
				},
				EnableParameterReference: "/flags/transcodeEnabled",
				EnableWhenReferencedParameterEquals: map[string]shared.JSONValue{
					"value": mediaWorkflowJSONValue(`true`),
				},
			}},
			MediaWorkflowConfigurationIds: []string{
				"ocid1.mediaworkflowconfiguration.oc1..config1",
			},
			Parameters: map[string]shared.JSONValue{
				"bucket":     mediaWorkflowJSONValue(`"media-bucket"`),
				"retryCount": mediaWorkflowJSONValue(`3`),
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			Locks: []mediaservicesv1beta1.MediaWorkflowLock{{
				Type:              "DELETE",
				CompartmentId:     "ocid1.compartment.oc1..example",
				RelatedResourceId: "ocid1.locksource.oc1..example",
				Message:           "managed lock",
			}},
		},
	}
}

func trackMediaWorkflow(resource *mediaservicesv1beta1.MediaWorkflow, mediaWorkflowID string) {
	resource.Status.Id = mediaWorkflowID
	resource.Status.OsokStatus.Ocid = shared.OCID(mediaWorkflowID)
}

func observedMediaWorkflowFromSpec(
	id string,
	spec mediaservicesv1beta1.MediaWorkflowSpec,
	state mediaservicessdk.MediaWorkflowLifecycleStateEnum,
) mediaservicessdk.MediaWorkflow {
	tasks, err := mediaWorkflowTasksFromSpec(spec.Tasks)
	if err != nil {
		panic(err)
	}
	parameters, err := mediaWorkflowParametersFromSpec(spec.Parameters)
	if err != nil {
		panic(err)
	}
	definedTags, err := mediaWorkflowDefinedTagsFromSpec(spec.DefinedTags)
	if err != nil {
		panic(err)
	}
	current := mediaservicessdk.MediaWorkflow{
		Id:                            common.String(id),
		DisplayName:                   optionalString(spec.DisplayName),
		CompartmentId:                 common.String(spec.CompartmentId),
		Tasks:                         tasks,
		MediaWorkflowConfigurationIds: cloneStringSlice(spec.MediaWorkflowConfigurationIds),
		Parameters:                    parameters,
		LifecycleState:                state,
		Version:                       optionalInt64(7),
		Locks:                         mediaWorkflowLocksFromSpec(spec.Locks),
		FreeformTags:                  cloneStringMap(spec.FreeformTags),
		DefinedTags:                   definedTags,
	}
	return current
}

func mediaWorkflowLocksFromSpec(spec []mediaservicesv1beta1.MediaWorkflowLock) []mediaservicessdk.ResourceLock {
	converted := make([]mediaservicessdk.ResourceLock, 0, len(spec))
	for _, lock := range spec {
		converted = append(converted, mediaservicessdk.ResourceLock{
			Type:              mediaservicessdk.ResourceLockTypeEnum(lock.Type),
			CompartmentId:     optionalString(lock.CompartmentId),
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

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
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

func mediaWorkflowSerializedRequestBody(t *testing.T, request any, method string, path string) string {
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

func TestApplyMediaWorkflowRuntimeHooksUsesReviewedContract(t *testing.T) {
	t.Parallel()

	hooks := newMediaWorkflowDefaultRuntimeHooks(mediaservicessdk.MediaServicesClient{})
	applyMediaWorkflowRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("hooks.Semantics.Async = %#v, want explicit async.lifecycle contract", hooks.Semantics.Async)
	}
	if !reflect.DeepEqual(hooks.List.Fields, mediaWorkflowListFields()) {
		t.Fatalf("list fields = %#v, want %#v", hooks.List.Fields, mediaWorkflowListFields())
	}
	if !reflect.DeepEqual(hooks.Update.Fields, mediaWorkflowUpdateFields()) {
		t.Fatalf("update fields = %#v, want %#v", hooks.Update.Fields, mediaWorkflowUpdateFields())
	}
	if !reflect.DeepEqual(hooks.Delete.Fields, mediaWorkflowDeleteFields()) {
		t.Fatalf("delete fields = %#v, want %#v", hooks.Delete.Fields, mediaWorkflowDeleteFields())
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want create body sanitizer")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want bounded pre-create reuse")
	}
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("hooks.ParityHooks.NormalizeDesiredState = nil, want lock normalization hook")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if got := hooks.Semantics.DeleteFollowUp.Strategy; got != "confirm-delete" {
		t.Fatalf("delete follow-up = %q, want %q", got, "confirm-delete")
	}
	if got := hooks.Semantics.CreateFollowUp.Strategy; got != "read-after-write" {
		t.Fatalf("create follow-up = %q, want %q", got, "read-after-write")
	}
	if got := hooks.Semantics.UpdateFollowUp.Strategy; got != "read-after-write" {
		t.Fatalf("update follow-up = %q, want %q", got, "read-after-write")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("auxiliary operations = %#v, want ChangeMediaWorkflowCompartment removed", hooks.Semantics.AuxiliaryOperations)
	}
	if !containsString(hooks.Semantics.Mutation.Mutable, "tasks") {
		t.Fatalf("mutable fields = %#v, want tasks in reviewed mutable surface", hooks.Semantics.Mutation.Mutable)
	}
}

func TestBuildMediaWorkflowCreateDetailsOmitsLockTimeCreated(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	resource.Spec.Locks[0].TimeCreated = "2026-05-07T12:00:00Z"

	details, err := buildMediaWorkflowCreateDetails(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildMediaWorkflowCreateDetails() error = %v", err)
	}
	if len(details.Locks) != 1 {
		t.Fatalf("create locks = %d, want 1", len(details.Locks))
	}
	if details.Locks[0].TimeCreated != nil {
		t.Fatalf("create lock timeCreated = %#v, want omitted from reviewed request", details.Locks[0].TimeCreated)
	}
}

func TestMediaWorkflowServiceClientUpdatesSupportedMutableDriftAndClearsEmptyValues(t *testing.T) {
	t.Parallel()

	original := newMediaWorkflowTestResource()
	resource := newMediaWorkflowTestResource()
	trackMediaWorkflow(resource, testMediaWorkflowID)
	resource.Spec.DisplayName = "workflow-beta"
	resource.Spec.Tasks = []mediaservicesv1beta1.MediaWorkflowTask{}
	resource.Spec.MediaWorkflowConfigurationIds = []string{}
	resource.Spec.Parameters = map[string]shared.JSONValue{}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	var updateRequest mediaservicessdk.UpdateMediaWorkflowRequest

	client := newMediaWorkflowTestClient(&fakeMediaWorkflowOCIClient{
		getFn: func(_ context.Context, req mediaservicessdk.GetMediaWorkflowRequest) (mediaservicessdk.GetMediaWorkflowResponse, error) {
			if req.MediaWorkflowId == nil || *req.MediaWorkflowId != testMediaWorkflowID {
				t.Fatalf("get mediaWorkflowId = %v, want %q", req.MediaWorkflowId, testMediaWorkflowID)
			}
			return mediaservicessdk.GetMediaWorkflowResponse{
				MediaWorkflow: observedMediaWorkflowFromSpec(
					testMediaWorkflowID,
					original.Spec,
					mediaservicessdk.MediaWorkflowLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, req mediaservicessdk.UpdateMediaWorkflowRequest) (mediaservicessdk.UpdateMediaWorkflowResponse, error) {
			updateRequest = req
			return mediaservicessdk.UpdateMediaWorkflowResponse{
				OpcRequestId: common.String("opc-update-1"),
				MediaWorkflow: observedMediaWorkflowFromSpec(
					testMediaWorkflowID,
					resource.Spec,
					mediaservicessdk.MediaWorkflowLifecycleStateActive,
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
	if updateRequest.MediaWorkflowId == nil || *updateRequest.MediaWorkflowId != testMediaWorkflowID {
		t.Fatalf("update mediaWorkflowId = %v, want %q", updateRequest.MediaWorkflowId, testMediaWorkflowID)
	}
	if updateRequest.IsLockOverride != nil {
		t.Fatalf("update isLockOverride = %#v, want reviewed hook field omission", updateRequest.IsLockOverride)
	}
	if updateRequest.UpdateMediaWorkflowDetails.DisplayName == nil || *updateRequest.UpdateMediaWorkflowDetails.DisplayName != "workflow-beta" {
		t.Fatalf("update displayName = %#v, want workflow-beta", updateRequest.UpdateMediaWorkflowDetails.DisplayName)
	}
	if updateRequest.UpdateMediaWorkflowDetails.Tasks == nil || len(updateRequest.UpdateMediaWorkflowDetails.Tasks) != 0 {
		t.Fatalf("update tasks = %#v, want explicit empty slice clear", updateRequest.UpdateMediaWorkflowDetails.Tasks)
	}
	if updateRequest.UpdateMediaWorkflowDetails.MediaWorkflowConfigurationIds == nil || len(updateRequest.UpdateMediaWorkflowDetails.MediaWorkflowConfigurationIds) != 0 {
		t.Fatalf("update mediaWorkflowConfigurationIds = %#v, want explicit empty slice clear", updateRequest.UpdateMediaWorkflowDetails.MediaWorkflowConfigurationIds)
	}
	if updateRequest.UpdateMediaWorkflowDetails.Parameters == nil || len(updateRequest.UpdateMediaWorkflowDetails.Parameters) != 0 {
		t.Fatalf("update parameters = %#v, want explicit empty map clear", updateRequest.UpdateMediaWorkflowDetails.Parameters)
	}
	if updateRequest.UpdateMediaWorkflowDetails.FreeformTags == nil || len(updateRequest.UpdateMediaWorkflowDetails.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map clear", updateRequest.UpdateMediaWorkflowDetails.FreeformTags)
	}
	if updateRequest.UpdateMediaWorkflowDetails.DefinedTags == nil || len(updateRequest.UpdateMediaWorkflowDetails.DefinedTags) != 0 {
		t.Fatalf("update definedTags = %#v, want explicit empty map clear", updateRequest.UpdateMediaWorkflowDetails.DefinedTags)
	}

	body := mediaWorkflowSerializedRequestBody(t, mediaservicessdk.UpdateMediaWorkflowRequest{
		MediaWorkflowId:            common.String(testMediaWorkflowID),
		UpdateMediaWorkflowDetails: updateRequest.UpdateMediaWorkflowDetails,
	}, http.MethodPut, "/mediaWorkflows/"+testMediaWorkflowID)
	for _, want := range []string{
		`"displayName":"workflow-beta"`,
		`"tasks":[]`,
		`"mediaWorkflowConfigurationIds":[]`,
		`"parameters":{}`,
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

func TestNormalizeMediaWorkflowDesiredStateClearsEquivalentLocks(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	current := observedMediaWorkflowFromSpec(testMediaWorkflowID, resource.Spec, mediaservicessdk.MediaWorkflowLifecycleStateActive)
	now := common.SDKTime{Time: time.Date(2026, time.May, 7, 12, 34, 56, 0, time.UTC)}
	current.Locks[0].TimeCreated = &now

	normalizeMediaWorkflowDesiredState(resource, current)
	if resource.Spec.Locks != nil {
		t.Fatalf("spec.locks = %#v, want normalized nil after equivalent OCI lock readback", resource.Spec.Locks)
	}
}

func TestValidateMediaWorkflowCreateOnlyDriftRejectsLockDrift(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	current := observedMediaWorkflowFromSpec(testMediaWorkflowID, resource.Spec, mediaservicessdk.MediaWorkflowLifecycleStateActive)
	current.Locks[0].Type = mediaservicessdk.ResourceLockTypeFull

	err := validateMediaWorkflowCreateOnlyDrift(resource, current)
	if err == nil || !strings.Contains(err.Error(), "locks") {
		t.Fatalf("validateMediaWorkflowCreateOnlyDrift() error = %v, want locks drift failure", err)
	}
}

func TestGuardMediaWorkflowExistingBeforeCreateRequiresDisplayName(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	resource.Spec.DisplayName = ""

	decision, err := guardMediaWorkflowExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardMediaWorkflowExistingBeforeCreate() error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardMediaWorkflowExistingBeforeCreate() = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "workflow-alpha"
	decision, err = guardMediaWorkflowExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardMediaWorkflowExistingBeforeCreate() error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardMediaWorkflowExistingBeforeCreate() = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestNewMediaWorkflowServiceClientWithOCIClientReusesPagedDisplayNameMatch(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	existingID := "ocid1.mediaworkflow.oc1..existing"
	createCalled := false
	listCalls := 0
	var getRequest mediaservicessdk.GetMediaWorkflowRequest

	client := newMediaWorkflowTestClient(&fakeMediaWorkflowOCIClient{
		createFn: func(context.Context, mediaservicessdk.CreateMediaWorkflowRequest) (mediaservicessdk.CreateMediaWorkflowResponse, error) {
			createCalled = true
			return mediaservicessdk.CreateMediaWorkflowResponse{}, nil
		},
		listFn: func(_ context.Context, req mediaservicessdk.ListMediaWorkflowsRequest) (mediaservicessdk.ListMediaWorkflowsResponse, error) {
			listCalls++
			switch listCalls {
			case 1:
				if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
					t.Fatalf("first ListMediaWorkflowsRequest.CompartmentId = %#v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
				}
				if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
					t.Fatalf("first ListMediaWorkflowsRequest.DisplayName = %#v, want %q", req.DisplayName, resource.Spec.DisplayName)
				}
				return mediaservicessdk.ListMediaWorkflowsResponse{
					MediaWorkflowCollection: mediaservicessdk.MediaWorkflowCollection{
						Items: []mediaservicessdk.MediaWorkflowSummary{},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			case 2:
				if req.Page == nil || *req.Page != "next-page" {
					t.Fatalf("second ListMediaWorkflowsRequest.Page = %#v, want next-page", req.Page)
				}
				return mediaservicessdk.ListMediaWorkflowsResponse{
					MediaWorkflowCollection: mediaservicessdk.MediaWorkflowCollection{
						Items: []mediaservicessdk.MediaWorkflowSummary{{
							Id:             common.String(existingID),
							CompartmentId:  common.String(resource.Spec.CompartmentId),
							DisplayName:    common.String(resource.Spec.DisplayName),
							LifecycleState: mediaservicessdk.MediaWorkflowLifecycleStateActive,
						}},
					},
				}, nil
			default:
				t.Fatalf("unexpected ListMediaWorkflows call #%d", listCalls)
				return mediaservicessdk.ListMediaWorkflowsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req mediaservicessdk.GetMediaWorkflowRequest) (mediaservicessdk.GetMediaWorkflowResponse, error) {
			getRequest = req
			return mediaservicessdk.GetMediaWorkflowResponse{
				MediaWorkflow: observedMediaWorkflowFromSpec(
					existingID,
					resource.Spec,
					mediaservicessdk.MediaWorkflowLifecycleStateActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateOrUpdate() invoked CreateMediaWorkflow, want existing-before-create reuse")
	}
	if getRequest.MediaWorkflowId == nil || *getRequest.MediaWorkflowId != existingID {
		t.Fatalf("GetMediaWorkflowRequest.MediaWorkflowId = %#v, want %q", getRequest.MediaWorkflowId, existingID)
	}
	if resource.Status.Id != existingID {
		t.Fatalf("resource.Status.Id = %q, want %q", resource.Status.Id, existingID)
	}
}

func TestMediaWorkflowServiceClientDeleteOmitsLockOverrideAndConfirmsDeleted(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	trackMediaWorkflow(resource, testMediaWorkflowID)

	getCalls := 0
	var deleteRequest mediaservicessdk.DeleteMediaWorkflowRequest

	client := newMediaWorkflowTestClient(&fakeMediaWorkflowOCIClient{
		getFn: func(_ context.Context, req mediaservicessdk.GetMediaWorkflowRequest) (mediaservicessdk.GetMediaWorkflowResponse, error) {
			getCalls++
			if req.MediaWorkflowId == nil || *req.MediaWorkflowId != testMediaWorkflowID {
				t.Fatalf("get mediaWorkflowId = %v, want %q", req.MediaWorkflowId, testMediaWorkflowID)
			}
			state := mediaservicessdk.MediaWorkflowLifecycleStateActive
			if getCalls > 1 {
				state = mediaservicessdk.MediaWorkflowLifecycleStateDeleted
			}
			return mediaservicessdk.GetMediaWorkflowResponse{
				MediaWorkflow: observedMediaWorkflowFromSpec(testMediaWorkflowID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req mediaservicessdk.DeleteMediaWorkflowRequest) (mediaservicessdk.DeleteMediaWorkflowResponse, error) {
			deleteRequest = req
			return mediaservicessdk.DeleteMediaWorkflowResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want terminal delete confirmation")
	}
	if getCalls != 2 {
		t.Fatalf("GetMediaWorkflow() calls = %d, want 2", getCalls)
	}
	if deleteRequest.MediaWorkflowId == nil || *deleteRequest.MediaWorkflowId != testMediaWorkflowID {
		t.Fatalf("delete mediaWorkflowId = %v, want %q", deleteRequest.MediaWorkflowId, testMediaWorkflowID)
	}
	if deleteRequest.IsLockOverride != nil {
		t.Fatalf("delete isLockOverride = %#v, want reviewed hook field omission", deleteRequest.IsLockOverride)
	}
	if resource.Status.LifecycleState != "DELETED" {
		t.Fatalf("status.lifecycleState = %q, want DELETED", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed delete timestamp")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	}
}
