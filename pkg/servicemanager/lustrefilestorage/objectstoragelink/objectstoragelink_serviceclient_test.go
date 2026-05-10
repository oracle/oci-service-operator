package objectstoragelink

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	lustrefilestoragev1beta1 "github.com/oracle/oci-service-operator/api/lustrefilestorage/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestObjectStorageLinkRuntimeHooksConfigureDeleteOnlyWorkRequest(t *testing.T) {
	t.Parallel()

	fake := &fakeObjectStorageLinkWorkRequestClient{
		getWorkRequestFn: func(_ context.Context, request lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error) {
			requireObjectStorageLinkStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-delete-1")
			return lustrefilestoragesdk.GetWorkRequestResponse{
				WorkRequest: makeObjectStorageLinkWorkRequest("wr-delete-1", "ocid1.objectstoragelink.oc1..existing", lustrefilestoragesdk.OperationStatusInProgress),
			}, nil
		},
	}

	hooks := newObjectStorageLinkRuntimeHooksWithWorkRequestClient(fake)
	if hooks.Semantics == nil || hooks.Semantics.Async == nil || hooks.Semantics.Async.WorkRequest == nil {
		t.Fatal("ObjectStorageLink async semantics are incomplete, want delete-only workrequest metadata")
	}
	if !slices.Equal(hooks.Semantics.Async.WorkRequest.Phases, []string{"delete"}) {
		t.Fatalf("Async.WorkRequest.Phases = %v, want [delete]", hooks.Semantics.Async.WorkRequest.Phases)
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("hooks.Async.GetWorkRequest = nil, want Lustre work-request fetcher")
	}
	if !slices.Contains(hooks.Async.Adapter.PendingStatusTokens, string(lustrefilestoragesdk.OperationStatusInProgress)) {
		t.Fatalf("PendingStatusTokens = %v, want IN_PROGRESS", hooks.Async.Adapter.PendingStatusTokens)
	}

	workRequest, err := hooks.Async.GetWorkRequest(context.Background(), "wr-delete-1")
	if err != nil {
		t.Fatalf("hooks.Async.GetWorkRequest() error = %v", err)
	}
	got, ok := workRequest.(lustrefilestoragesdk.WorkRequest)
	if !ok {
		t.Fatalf("hooks.Async.GetWorkRequest() type = %T, want lustrefilestorage.WorkRequest", workRequest)
	}
	requireObjectStorageLinkStringPtr(t, "WorkRequest.Id", got.Id, "wr-delete-1")
	if len(fake.requests) != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", len(fake.requests))
	}
}

func TestObjectStorageLinkCreateRequestMapsSpecAndRereadsActive(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.objectstoragelink.oc1..created"

	resource := newObjectStorageLinkTestResource()
	var createRequest lustrefilestoragesdk.CreateObjectStorageLinkRequest

	cfg := testObjectStorageLinkRuntimeConfig(nil)
	cfg.Create = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.CreateObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			createRequest = *request.(*lustrefilestoragesdk.CreateObjectStorageLinkRequest)
			return lustrefilestoragesdk.CreateObjectStorageLinkResponse{
				ObjectStorageLink: observedObjectStorageLinkFromSpec(createdID, resource.Spec, "CREATING"),
			}, nil
		},
		Fields: objectStorageLinkCreateFields(),
	}
	cfg.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.GetObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			getRequest := *request.(*lustrefilestoragesdk.GetObjectStorageLinkRequest)
			requireObjectStorageLinkStringPtr(t, "GetObjectStorageLinkRequest.ObjectStorageLinkId", getRequest.ObjectStorageLinkId, createdID)
			return lustrefilestoragesdk.GetObjectStorageLinkResponse{
				ObjectStorageLink: observedObjectStorageLinkFromSpec(createdID, resource.Spec, "ACTIVE"),
			}, nil
		},
		Fields: objectStorageLinkGetFields(),
	}

	manager := newObjectStorageLinkRuntimeTestManager(cfg)
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after the ACTIVE reread")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false after the ACTIVE reread")
	}
	requireObjectStorageLinkStringPtr(t, "CreateObjectStorageLinkRequest.CompartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	requireObjectStorageLinkStringPtr(t, "CreateObjectStorageLinkRequest.AvailabilityDomain", createRequest.AvailabilityDomain, resource.Spec.AvailabilityDomain)
	requireObjectStorageLinkStringPtr(t, "CreateObjectStorageLinkRequest.LustreFileSystemId", createRequest.LustreFileSystemId, resource.Spec.LustreFileSystemId)
	requireObjectStorageLinkStringPtr(t, "CreateObjectStorageLinkRequest.FileSystemPath", createRequest.FileSystemPath, resource.Spec.FileSystemPath)
	requireObjectStorageLinkStringPtr(t, "CreateObjectStorageLinkRequest.ObjectStoragePrefix", createRequest.ObjectStoragePrefix, resource.Spec.ObjectStoragePrefix)
	requireObjectStorageLinkStringPtr(t, "CreateObjectStorageLinkRequest.DisplayName", createRequest.DisplayName, resource.Spec.DisplayName)
	if createRequest.IsOverwrite == nil || *createRequest.IsOverwrite != resource.Spec.IsOverwrite {
		t.Fatalf("CreateObjectStorageLinkRequest.IsOverwrite = %v, want %t", createRequest.IsOverwrite, resource.Spec.IsOverwrite)
	}
	if got := createRequest.FreeformTags["team"]; got != resource.Spec.FreeformTags["team"] {
		t.Fatalf("CreateObjectStorageLinkRequest.FreeformTags[team] = %q, want %q", got, resource.Spec.FreeformTags["team"])
	}
}

func TestObjectStorageLinkCreateOrUpdateClassifiesObservedLifecycleStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		lifecycleState string
		wantSuccessful bool
		wantRequeue    bool
		wantCondition  shared.OSOKConditionType
	}{
		{
			name:           "active",
			lifecycleState: "ACTIVE",
			wantSuccessful: true,
			wantRequeue:    false,
			wantCondition:  shared.Active,
		},
		{
			name:           "creating",
			lifecycleState: "CREATING",
			wantSuccessful: true,
			wantRequeue:    true,
			wantCondition:  shared.Provisioning,
		},
		{
			name:           "failed",
			lifecycleState: "FAILED",
			wantSuccessful: false,
			wantRequeue:    false,
			wantCondition:  shared.Failed,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.objectstoragelink.oc1..existing"

			resource := newExistingObjectStorageLinkTestResource(existingID)
			cfg := testObjectStorageLinkRuntimeConfig(nil)
			cfg.Get = &generatedruntime.Operation{
				NewRequest: func() any { return &lustrefilestoragesdk.GetObjectStorageLinkRequest{} },
				Call: func(_ context.Context, request any) (any, error) {
					getRequest := *request.(*lustrefilestoragesdk.GetObjectStorageLinkRequest)
					requireObjectStorageLinkStringPtr(t, "GetObjectStorageLinkRequest.ObjectStorageLinkId", getRequest.ObjectStorageLinkId, existingID)
					return lustrefilestoragesdk.GetObjectStorageLinkResponse{
						ObjectStorageLink: observedObjectStorageLinkFromSpec(existingID, resource.Spec, tc.lifecycleState),
					}, nil
				},
				Fields: objectStorageLinkGetFields(),
			}

			manager := newObjectStorageLinkRuntimeTestManager(cfg)
			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccessful)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tc.wantCondition) {
				t.Fatalf("status.reason = %q, want %q", got, tc.wantCondition)
			}
		})
	}
}

func TestObjectStorageLinkCreateOrUpdateBindsExistingExactListMatch(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.objectstoragelink.oc1..seeded"

	resource := newObjectStorageLinkTestResource()
	createCalled := false
	var listRequest lustrefilestoragesdk.ListObjectStorageLinksRequest

	cfg := testObjectStorageLinkRuntimeConfig(nil)
	cfg.Create = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.CreateObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			createCalled = true
			return lustrefilestoragesdk.CreateObjectStorageLinkResponse{}, nil
		},
		Fields: objectStorageLinkCreateFields(),
	}
	cfg.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.GetObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			getRequest := *request.(*lustrefilestoragesdk.GetObjectStorageLinkRequest)
			requireObjectStorageLinkStringPtr(t, "GetObjectStorageLinkRequest.ObjectStorageLinkId", getRequest.ObjectStorageLinkId, existingID)
			return lustrefilestoragesdk.GetObjectStorageLinkResponse{
				ObjectStorageLink: observedObjectStorageLinkFromSpec(existingID, resource.Spec, "ACTIVE"),
			}, nil
		},
		Fields: objectStorageLinkGetFields(),
	}
	cfg.List = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.ListObjectStorageLinksRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			listRequest = *request.(*lustrefilestoragesdk.ListObjectStorageLinksRequest)
			return lustrefilestoragesdk.ListObjectStorageLinksResponse{
				ObjectStorageLinkCollection: lustrefilestoragesdk.ObjectStorageLinkCollection{
					Items: []lustrefilestoragesdk.ObjectStorageLinkSummary{
						observedObjectStorageLinkSummaryFromSpec(existingID, resource.Spec, "ACTIVE"),
					},
				},
			}, nil
		},
		Fields: objectStorageLinkListFields(),
	}

	manager := newObjectStorageLinkRuntimeTestManager(cfg)
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when the exact list match is reusable")
	}
	if createCalled {
		t.Fatal("CreateObjectStorageLink() should not be called when list lookup found a reusable exact match")
	}
	requireObjectStorageLinkStringPtr(t, "ListObjectStorageLinksRequest.CompartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireObjectStorageLinkStringPtr(t, "ListObjectStorageLinksRequest.AvailabilityDomain", listRequest.AvailabilityDomain, resource.Spec.AvailabilityDomain)
	requireObjectStorageLinkStringPtr(t, "ListObjectStorageLinksRequest.LustreFileSystemId", listRequest.LustreFileSystemId, resource.Spec.LustreFileSystemId)
	requireObjectStorageLinkStringPtr(t, "ListObjectStorageLinksRequest.DisplayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestObjectStorageLinkCreateOrUpdateRejectsReplacementOnlyCompartmentDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.objectstoragelink.oc1..existing"

	current := newExistingObjectStorageLinkTestResource(existingID)
	resource := newExistingObjectStorageLinkTestResource(existingID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
	updateCalls := 0

	cfg := testObjectStorageLinkRuntimeConfig(nil)
	cfg.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.GetObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			getRequest := *request.(*lustrefilestoragesdk.GetObjectStorageLinkRequest)
			requireObjectStorageLinkStringPtr(t, "GetObjectStorageLinkRequest.ObjectStorageLinkId", getRequest.ObjectStorageLinkId, existingID)
			return lustrefilestoragesdk.GetObjectStorageLinkResponse{
				ObjectStorageLink: observedObjectStorageLinkFromSpec(existingID, current.Spec, "ACTIVE"),
			}, nil
		},
		Fields: objectStorageLinkGetFields(),
	}
	cfg.Update = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.UpdateObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			updateCalls++
			return lustrefilestoragesdk.UpdateObjectStorageLinkResponse{}, nil
		},
		Fields: objectStorageLinkUpdateFields(),
	}

	manager := newObjectStorageLinkRuntimeTestManager(cfg)
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want replacement-only compartment drift failure")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId detail", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when replacement-only compartment drift is detected")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateObjectStorageLink() called %d times, want 0 after replacement-only compartment drift", updateCalls)
	}
}

func TestObjectStorageLinkDeleteTracksDeleteWorkRequestUntilConfirmed(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.objectstoragelink.oc1..existing"
	const workRequestID = "wr-delete-1"

	fake := &fakeObjectStorageLinkWorkRequestClient{
		getWorkRequestFn: func(_ context.Context, request lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error) {
			requireObjectStorageLinkStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, workRequestID)
			return lustrefilestoragesdk.GetWorkRequestResponse{
				WorkRequest: makeObjectStorageLinkWorkRequest(workRequestID, existingID, lustrefilestoragesdk.OperationStatusInProgress),
			}, nil
		},
	}

	cfg := testObjectStorageLinkRuntimeConfig(fake)
	cfg.Delete = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.DeleteObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			deleteRequest := *request.(*lustrefilestoragesdk.DeleteObjectStorageLinkRequest)
			requireObjectStorageLinkStringPtr(t, "DeleteObjectStorageLinkRequest.ObjectStorageLinkId", deleteRequest.ObjectStorageLinkId, existingID)
			return lustrefilestoragesdk.DeleteObjectStorageLinkResponse{
				OpcWorkRequestId: common.String(workRequestID),
			}, nil
		},
		Fields: objectStorageLinkDeleteFields(),
	}

	manager := newObjectStorageLinkRuntimeTestManager(cfg)
	resource := newExistingObjectStorageLinkTestResource(existingID)

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while the delete work request is still pending")
	}
	if len(fake.requests) != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", len(fake.requests))
	}
	assertObjectStorageLinkCurrentAsync(t, resource, shared.OSOKAsyncPhaseDelete, workRequestID, shared.OSOKAsyncClassPending)
}

func TestObjectStorageLinkDeleteConfirmsSucceededWorkRequestViaDeletedRead(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.objectstoragelink.oc1..existing"
	const workRequestID = "wr-delete-1"

	fake := &fakeObjectStorageLinkWorkRequestClient{
		getWorkRequestFn: func(_ context.Context, request lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error) {
			requireObjectStorageLinkStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, workRequestID)
			return lustrefilestoragesdk.GetWorkRequestResponse{
				WorkRequest: makeObjectStorageLinkWorkRequest(workRequestID, existingID, lustrefilestoragesdk.OperationStatusSucceeded),
			}, nil
		},
	}

	cfg := testObjectStorageLinkRuntimeConfig(fake)
	cfg.Delete = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.DeleteObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			return lustrefilestoragesdk.DeleteObjectStorageLinkResponse{
				OpcWorkRequestId: common.String(workRequestID),
			}, nil
		},
		Fields: objectStorageLinkDeleteFields(),
	}
	cfg.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &lustrefilestoragesdk.GetObjectStorageLinkRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			getRequest := *request.(*lustrefilestoragesdk.GetObjectStorageLinkRequest)
			requireObjectStorageLinkStringPtr(t, "GetObjectStorageLinkRequest.ObjectStorageLinkId", getRequest.ObjectStorageLinkId, existingID)
			return lustrefilestoragesdk.GetObjectStorageLinkResponse{
				ObjectStorageLink: observedObjectStorageLinkFromSpec(existingID, newObjectStorageLinkTestResource().Spec, "DELETED"),
			}, nil
		},
		Fields: objectStorageLinkGetFields(),
	}

	manager := newObjectStorageLinkRuntimeTestManager(cfg)
	resource := newExistingObjectStorageLinkTestResource(existingID)

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after the delete work request succeeds and OCI reports DELETED")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp after confirmed delete")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

type fakeObjectStorageLinkWorkRequestClient struct {
	requests         []lustrefilestoragesdk.GetWorkRequestRequest
	getWorkRequestFn func(context.Context, lustrefilestoragesdk.GetWorkRequestRequest) (lustrefilestoragesdk.GetWorkRequestResponse, error)
}

func (f *fakeObjectStorageLinkWorkRequestClient) GetWorkRequest(
	ctx context.Context,
	request lustrefilestoragesdk.GetWorkRequestRequest,
) (lustrefilestoragesdk.GetWorkRequestResponse, error) {
	f.requests = append(f.requests, request)
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return lustrefilestoragesdk.GetWorkRequestResponse{}, nil
}

func newObjectStorageLinkRuntimeTestManager(
	cfg generatedruntime.Config[*lustrefilestoragev1beta1.ObjectStorageLink],
) *ObjectStorageLinkServiceManager {
	return &ObjectStorageLinkServiceManager{
		client: defaultObjectStorageLinkServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*lustrefilestoragev1beta1.ObjectStorageLink](cfg),
		},
	}
}

func testObjectStorageLinkRuntimeConfig(
	workRequestClient objectStorageLinkWorkRequestClient,
) generatedruntime.Config[*lustrefilestoragev1beta1.ObjectStorageLink] {
	hooks := newObjectStorageLinkRuntimeHooksWithWorkRequestClient(workRequestClient)
	return generatedruntime.Config[*lustrefilestoragev1beta1.ObjectStorageLink]{
		Kind:      "ObjectStorageLink",
		SDKName:   "ObjectStorageLink",
		Semantics: hooks.Semantics,
		Async:     hooks.Async,
	}
}

func newObjectStorageLinkTestResource() *lustrefilestoragev1beta1.ObjectStorageLink {
	return &lustrefilestoragev1beta1.ObjectStorageLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "objectstoragelink-sample",
			Namespace: "default",
		},
		Spec: lustrefilestoragev1beta1.ObjectStorageLinkSpec{
			CompartmentId:       "ocid1.compartment.oc1..objectstoragelink",
			AvailabilityDomain:  "Uocm:PHX-AD-1",
			LustreFileSystemId:  "ocid1.lustrefilesystem.oc1..filesystem",
			FileSystemPath:      "/mnt/lustre/link",
			ObjectStoragePrefix: "namespace:/bucket/prefix",
			IsOverwrite:         true,
			DisplayName:         "sample-object-storage-link",
			FreeformTags: map[string]string{
				"team": "storage",
			},
		},
	}
}

func newExistingObjectStorageLinkTestResource(existingID string) *lustrefilestoragev1beta1.ObjectStorageLink {
	resource := newObjectStorageLinkTestResource()
	resource.Status = lustrefilestoragev1beta1.ObjectStorageLinkStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedObjectStorageLinkFromSpec(
	id string,
	spec lustrefilestoragev1beta1.ObjectStorageLinkSpec,
	lifecycleState string,
) lustrefilestoragesdk.ObjectStorageLink {
	now := common.SDKTime{Time: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)}
	return lustrefilestoragesdk.ObjectStorageLink{
		Id:                  common.String(id),
		CompartmentId:       common.String(spec.CompartmentId),
		AvailabilityDomain:  common.String(spec.AvailabilityDomain),
		DisplayName:         common.String(spec.DisplayName),
		TimeCreated:         &now,
		TimeUpdated:         &now,
		LifecycleState:      lustrefilestoragesdk.ObjectStorageLinkLifecycleStateEnum(lifecycleState),
		FreeformTags:        spec.FreeformTags,
		SystemTags:          map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		LustreFileSystemId:  common.String(spec.LustreFileSystemId),
		FileSystemPath:      common.String(spec.FileSystemPath),
		ObjectStoragePrefix: common.String(spec.ObjectStoragePrefix),
		IsOverwrite:         common.Bool(spec.IsOverwrite),
		LifecycleDetails:    common.String("lifecycle detail"),
		CurrentJobId:        common.String("ocid1.syncjob.oc1..current"),
		LastJobId:           common.String("ocid1.syncjob.oc1..last"),
	}
}

func observedObjectStorageLinkSummaryFromSpec(
	id string,
	spec lustrefilestoragev1beta1.ObjectStorageLinkSpec,
	lifecycleState string,
) lustrefilestoragesdk.ObjectStorageLinkSummary {
	now := common.SDKTime{Time: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)}
	return lustrefilestoragesdk.ObjectStorageLinkSummary{
		Id:                  common.String(id),
		CompartmentId:       common.String(spec.CompartmentId),
		AvailabilityDomain:  common.String(spec.AvailabilityDomain),
		DisplayName:         common.String(spec.DisplayName),
		TimeCreated:         &now,
		TimeUpdated:         &now,
		LifecycleState:      lustrefilestoragesdk.ObjectStorageLinkLifecycleStateEnum(lifecycleState),
		FreeformTags:        spec.FreeformTags,
		SystemTags:          map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		LustreFileSystemId:  common.String(spec.LustreFileSystemId),
		FileSystemPath:      common.String(spec.FileSystemPath),
		ObjectStoragePrefix: common.String(spec.ObjectStoragePrefix),
		IsOverwrite:         common.Bool(spec.IsOverwrite),
		LifecycleDetails:    common.String("lifecycle detail"),
		CurrentJobId:        common.String("ocid1.syncjob.oc1..current"),
		LastJobId:           common.String("ocid1.syncjob.oc1..last"),
	}
}

func makeObjectStorageLinkWorkRequest(
	workRequestID string,
	resourceID string,
	status lustrefilestoragesdk.OperationStatusEnum,
) lustrefilestoragesdk.WorkRequest {
	now := common.SDKTime{Time: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)}
	action := lustrefilestoragesdk.ActionTypeDeleted
	if status == lustrefilestoragesdk.OperationStatusInProgress || status == lustrefilestoragesdk.OperationStatusAccepted || status == lustrefilestoragesdk.OperationStatusWaiting {
		action = lustrefilestoragesdk.ActionTypeInProgress
	}
	return lustrefilestoragesdk.WorkRequest{
		OperationType: lustrefilestoragesdk.OperationTypeDeleteObjectStorageLink,
		Status:        status,
		Id:            common.String(workRequestID),
		CompartmentId: common.String("ocid1.compartment.oc1..objectstoragelink"),
		Resources: []lustrefilestoragesdk.WorkRequestResource{
			{
				EntityType: common.String("objectstoragelink"),
				ActionType: action,
				Identifier: common.String(resourceID),
			},
		},
		PercentComplete: objectStorageLinkFloat32Ptr(50),
		TimeAccepted:    &now,
	}
}

func objectStorageLinkCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateObjectStorageLinkDetails", RequestName: "CreateObjectStorageLinkDetails", Contribution: "body"},
	}
}

func objectStorageLinkGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ObjectStorageLinkId", RequestName: "objectStorageLinkId", Contribution: "path", PreferResourceID: true},
	}
}

func objectStorageLinkListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
		{FieldName: "AvailabilityDomain", RequestName: "availabilityDomain", Contribution: "query", PreferResourceID: false},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: false},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
		{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
		{FieldName: "LustreFileSystemId", RequestName: "lustreFileSystemId", Contribution: "query", PreferResourceID: false},
	}
}

func objectStorageLinkUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ObjectStorageLinkId", RequestName: "objectStorageLinkId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateObjectStorageLinkDetails", RequestName: "UpdateObjectStorageLinkDetails", Contribution: "body", PreferResourceID: false},
	}
}

func objectStorageLinkDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ObjectStorageLinkId", RequestName: "objectStorageLinkId", Contribution: "path", PreferResourceID: true},
	}
}

func assertObjectStorageLinkCurrentAsync(
	t *testing.T,
	resource *lustrefilestoragev1beta1.ObjectStorageLink,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked delete work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func requireObjectStorageLinkStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func objectStorageLinkFloat32Ptr(value float32) *float32 {
	return &value
}
