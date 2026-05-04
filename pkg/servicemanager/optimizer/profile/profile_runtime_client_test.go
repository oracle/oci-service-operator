/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package profile

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	optimizersdk "github.com/oracle/oci-go-sdk/v65/optimizer"
	optimizerv1beta1 "github.com/oracle/oci-service-operator/api/optimizer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testProfileCompartmentID = "ocid1.tenancy.oc1..profilecompartment"
	testProfileID            = "ocid1.optimizerprofile.oc1..profile"
	testProfileName          = "profile-alpha"
)

type fakeProfileOCIClient struct {
	createFunc func(context.Context, optimizersdk.CreateProfileRequest) (optimizersdk.CreateProfileResponse, error)
	getFunc    func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error)
	listFunc   func(context.Context, optimizersdk.ListProfilesRequest) (optimizersdk.ListProfilesResponse, error)
	updateFunc func(context.Context, optimizersdk.UpdateProfileRequest) (optimizersdk.UpdateProfileResponse, error)
	deleteFunc func(context.Context, optimizersdk.DeleteProfileRequest) (optimizersdk.DeleteProfileResponse, error)

	createRequests []optimizersdk.CreateProfileRequest
	getRequests    []optimizersdk.GetProfileRequest
	listRequests   []optimizersdk.ListProfilesRequest
	updateRequests []optimizersdk.UpdateProfileRequest
	deleteRequests []optimizersdk.DeleteProfileRequest
}

func (f *fakeProfileOCIClient) CreateProfile(
	ctx context.Context,
	request optimizersdk.CreateProfileRequest,
) (optimizersdk.CreateProfileResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return optimizersdk.CreateProfileResponse{}, nil
}

func (f *fakeProfileOCIClient) GetProfile(
	ctx context.Context,
	request optimizersdk.GetProfileRequest,
) (optimizersdk.GetProfileResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return optimizersdk.GetProfileResponse{}, nil
}

func (f *fakeProfileOCIClient) ListProfiles(
	ctx context.Context,
	request optimizersdk.ListProfilesRequest,
) (optimizersdk.ListProfilesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return optimizersdk.ListProfilesResponse{}, nil
}

func (f *fakeProfileOCIClient) UpdateProfile(
	ctx context.Context,
	request optimizersdk.UpdateProfileRequest,
) (optimizersdk.UpdateProfileResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return optimizersdk.UpdateProfileResponse{}, nil
}

func (f *fakeProfileOCIClient) DeleteProfile(
	ctx context.Context,
	request optimizersdk.DeleteProfileRequest,
) (optimizersdk.DeleteProfileResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return optimizersdk.DeleteProfileResponse{}, nil
}

func TestProfileRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newProfileDefaultRuntimeHooks(optimizersdk.OptimizerClient{})
	applyProfileRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.Semantics.List == nil || !reflect.DeepEqual(hooks.Semantics.List.MatchFields, []string{"compartmentId", "name", "id"}) {
		t.Fatalf("hooks.Semantics.List = %#v, want compartment/name/id matching", hooks.Semantics.List)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create body")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update body")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want pre-create guard")
	}
	if hooks.Read.List == nil {
		t.Fatal("hooks.Read.List = nil, want paginated list read")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete errors")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("hooks.WrapGeneratedClient is empty, want delete confirmation guard")
	}
}

//nolint:gocognit,gocyclo // The create request and status assertions document the runtime contract.
func TestProfileServiceClientCreateOrUpdateCreatesProfileAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	fake := &fakeProfileOCIClient{}
	fake.listFunc = func(context.Context, optimizersdk.ListProfilesRequest) (optimizersdk.ListProfilesResponse, error) {
		return optimizersdk.ListProfilesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request optimizersdk.CreateProfileRequest) (optimizersdk.CreateProfileResponse, error) {
		assertProfileCreateRequest(t, request, newProfileRuntimeTestResource().Spec)
		return optimizersdk.CreateProfileResponse{
			OpcRequestId: common.String("opc-create-1"),
			Profile:      profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateCreating),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		if got := profileStringValue(request.ProfileId); got != testProfileID {
			t.Fatalf("get profileId = %q, want %q", got, testProfileID)
		}
		return optimizersdk.GetProfileResponse{
			Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateActive),
		}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE follow-up")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListProfiles() calls = %d, want 1 pre-create lookup", len(fake.listRequests))
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateProfile() calls = %d, want 1", len(fake.createRequests))
	}
	if got := resource.Status.Id; got != testProfileID {
		t.Fatalf("status.id = %q, want %q", got, testProfileID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProfileID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProfileID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := resource.Status.LifecycleState; got != string(optimizersdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil for ACTIVE", resource.Status.OsokStatus.Async.Current)
	}
}

//nolint:gocognit,gocyclo // The paginated bind sequence is clearer with the assertions kept together.
func TestProfileServiceClientCreateOrUpdateBindsExistingProfileFromLaterListPage(t *testing.T) {
	t.Parallel()

	fake := &fakeProfileOCIClient{}
	fake.listFunc = func(_ context.Context, request optimizersdk.ListProfilesRequest) (optimizersdk.ListProfilesResponse, error) {
		if got := profileStringValue(request.CompartmentId); got != testProfileCompartmentID {
			t.Fatalf("list compartmentId = %q, want %q", got, testProfileCompartmentID)
		}
		if got := profileStringValue(request.Name); got != testProfileName {
			t.Fatalf("list name = %q, want %q", got, testProfileName)
		}
		switch len(fake.listRequests) {
		case 1:
			if request.Page != nil {
				t.Fatalf("first list page = %q, want nil", profileStringValue(request.Page))
			}
			otherSpec := newProfileRuntimeTestResource().Spec
			otherSpec.Name = "profile-other"
			return optimizersdk.ListProfilesResponse{
				ProfileCollection: optimizersdk.ProfileCollection{
					Items: []optimizersdk.ProfileSummary{
						profileSummaryFromSpec("ocid1.optimizerprofile.oc1..other", otherSpec, optimizersdk.LifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			if got := profileStringValue(request.Page); got != "page-2" {
				t.Fatalf("second list page = %q, want page-2", got)
			}
			return optimizersdk.ListProfilesResponse{
				ProfileCollection: optimizersdk.ProfileCollection{
					Items: []optimizersdk.ProfileSummary{
						profileSummaryFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected ListProfiles() call %d", len(fake.listRequests))
			return optimizersdk.ListProfilesResponse{}, nil
		}
	}
	fake.getFunc = func(_ context.Context, request optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		if got := profileStringValue(request.ProfileId); got != testProfileID {
			t.Fatalf("get profileId = %q, want %q", got, testProfileID)
		}
		return optimizersdk.GetProfileResponse{
			Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateActive),
		}, nil
	}
	fake.createFunc = func(context.Context, optimizersdk.CreateProfileRequest) (optimizersdk.CreateProfileResponse, error) {
		t.Fatal("CreateProfile() called; want existing profile bind")
		return optimizersdk.CreateProfileResponse{}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if got := len(fake.listRequests); got != 2 {
		t.Fatalf("ListProfiles() calls = %d, want 2 paginated calls", got)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateProfile() calls = %d, want 0", len(fake.createRequests))
	}
	if got := resource.Status.Id; got != testProfileID {
		t.Fatalf("status.id = %q, want bound profile ID", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProfileID {
		t.Fatalf("status.status.ocid = %q, want bound profile ID", got)
	}
}

func TestProfileServiceClientCreateOrUpdateSkipsUpdateWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		return optimizersdk.GetProfileResponse{
			Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, optimizersdk.UpdateProfileRequest) (optimizersdk.UpdateProfileResponse, error) {
		t.Fatal("UpdateProfile() called; want no-op reconcile")
		return optimizersdk.UpdateProfileResponse{}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	trackProfileID(resource, testProfileID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateProfile() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestProfileServiceClientCreateOrUpdateSkipsUpdateForLowercaseTargetTagValueType(t *testing.T) {
	t.Parallel()

	desired := newProfileRuntimeTestResource()
	desired.Spec.TargetTags.Items[0].TagValueType = "value"

	currentSpec := newProfileRuntimeTestResource().Spec
	currentSpec.TargetTags.Items[0].TagValueType = string(optimizersdk.TagValueTypeValue)

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		return optimizersdk.GetProfileResponse{
			Profile: profileFromSpec(testProfileID, currentSpec, optimizersdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, optimizersdk.UpdateProfileRequest) (optimizersdk.UpdateProfileResponse, error) {
		t.Fatal("UpdateProfile() called; want targetTags.tagValueType case-insensitive no-op")
		return optimizersdk.UpdateProfileResponse{}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	trackProfileID(desired, testProfileID)
	response, err := client.CreateOrUpdate(context.Background(), desired, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateProfile() calls = %d, want 0", len(fake.updateRequests))
	}
}

//nolint:gocognit,gocyclo // The update request shape is the behavior under test.
func TestProfileServiceClientCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	desired := newProfileRuntimeTestResource()
	currentSpec := newProfileRuntimeTestResource().Spec
	currentSpec.Name = "profile-old"
	currentSpec.Description = "old description"
	currentSpec.AggregationIntervalInDays = 30
	currentSpec.FreeformTags = map[string]string{"env": "old"}
	currentSpec.LevelsConfiguration.Items[0].Level = "LOW"

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(_ context.Context, request optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		if got := profileStringValue(request.ProfileId); got != testProfileID {
			t.Fatalf("get profileId = %q, want %q", got, testProfileID)
		}
		switch len(fake.getRequests) {
		case 1:
			return optimizersdk.GetProfileResponse{
				Profile: profileFromSpec(testProfileID, currentSpec, optimizersdk.LifecycleStateActive),
			}, nil
		case 2:
			return optimizersdk.GetProfileResponse{
				Profile: profileFromSpec(testProfileID, desired.Spec, optimizersdk.LifecycleStateActive),
			}, nil
		default:
			t.Fatalf("unexpected GetProfile() call %d", len(fake.getRequests))
			return optimizersdk.GetProfileResponse{}, nil
		}
	}
	fake.updateFunc = func(_ context.Context, request optimizersdk.UpdateProfileRequest) (optimizersdk.UpdateProfileResponse, error) {
		if got := profileStringValue(request.ProfileId); got != testProfileID {
			t.Fatalf("update profileId = %q, want %q", got, testProfileID)
		}
		if got := profileStringValue(request.Name); got != testProfileName {
			t.Fatalf("update name = %q, want %q", got, testProfileName)
		}
		if got := profileStringValue(request.Description); got != desired.Spec.Description {
			t.Fatalf("update description = %q, want desired description", got)
		}
		if request.AggregationIntervalInDays == nil || *request.AggregationIntervalInDays != desired.Spec.AggregationIntervalInDays {
			t.Fatalf("update aggregationIntervalInDays = %v, want %d", request.AggregationIntervalInDays, desired.Spec.AggregationIntervalInDays)
		}
		if got := request.FreeformTags; !reflect.DeepEqual(got, desired.Spec.FreeformTags) {
			t.Fatalf("update freeformTags = %#v, want %#v", got, desired.Spec.FreeformTags)
		}
		if request.LevelsConfiguration == nil || len(request.LevelsConfiguration.Items) != 1 ||
			profileStringValue(request.LevelsConfiguration.Items[0].Level) != "HIGH" {
			t.Fatalf("update levelsConfiguration = %#v, want HIGH level", request.LevelsConfiguration)
		}
		return optimizersdk.UpdateProfileResponse{
			OpcRequestId: common.String("opc-update-1"),
			Profile:      profileFromSpec(testProfileID, desired.Spec, optimizersdk.LifecycleStateUpdating),
		}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	trackProfileID(desired, testProfileID)
	response, err := client.CreateOrUpdate(context.Background(), desired, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("UpdateProfile() calls = %d, want 1", got)
	}
	if got := desired.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
	if got := desired.Status.LifecycleState; got != string(optimizersdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE after follow-up", got)
	}
}

func TestProfileServiceClientCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		currentSpec := newProfileRuntimeTestResource().Spec
		currentSpec.CompartmentId = "ocid1.tenancy.oc1..observed"
		return optimizersdk.GetProfileResponse{
			Profile: profileFromSpec(testProfileID, currentSpec, optimizersdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, optimizersdk.UpdateProfileRequest) (optimizersdk.UpdateProfileResponse, error) {
		t.Fatal("UpdateProfile() called; want create-only drift rejection")
		return optimizersdk.UpdateProfileResponse{}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	trackProfileID(resource, testProfileID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for create-only drift")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateProfile() calls = %d, want 0", len(fake.updateRequests))
	}
}

//nolint:gocognit,gocyclo // Delete lifecycle sequencing is the contract under test.
func TestProfileServiceClientDeleteRetainsFinalizerUntilLifecycleDeleteConfirmed(t *testing.T) {
	t.Parallel()

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(_ context.Context, request optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		if got := profileStringValue(request.ProfileId); got != testProfileID {
			t.Fatalf("get profileId = %q, want %q", got, testProfileID)
		}
		switch len(fake.getRequests) {
		case 1, 2:
			return optimizersdk.GetProfileResponse{
				Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateActive),
			}, nil
		case 3:
			return optimizersdk.GetProfileResponse{
				Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateDeleting),
			}, nil
		case 4, 5:
			return optimizersdk.GetProfileResponse{
				Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateDeleted),
			}, nil
		default:
			t.Fatalf("unexpected GetProfile() call %d", len(fake.getRequests))
			return optimizersdk.GetProfileResponse{}, nil
		}
	}
	fake.deleteFunc = func(_ context.Context, request optimizersdk.DeleteProfileRequest) (optimizersdk.DeleteProfileResponse, error) {
		if got := profileStringValue(request.ProfileId); got != testProfileID {
			t.Fatalf("delete profileId = %q, want %q", got, testProfileID)
		}
		return optimizersdk.DeleteProfileResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	trackProfileID(resource, testProfileID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call deleted = true, want finalizer retained while DELETING")
	}
	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("DeleteProfile() calls after first delete = %d, want 1", got)
	}
	if got := resource.Status.LifecycleState; got != string(optimizersdk.LifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want delete lifecycle tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current.phase = %q, want delete", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call deleted = false, want finalizer release after DELETED")
	}
	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("DeleteProfile() calls after second delete = %d, want no reissue", got)
	}
	if got := resource.Status.LifecycleState; got != string(optimizersdk.LifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState after second delete = %q, want DELETED", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
}

func TestProfileServiceClientDeleteRejectsAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-delete-error-1"

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		return optimizersdk.GetProfileResponse{
			Profile: profileFromSpec(testProfileID, newProfileRuntimeTestResource().Spec, optimizersdk.LifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, optimizersdk.DeleteProfileRequest) (optimizersdk.DeleteProfileResponse, error) {
		return optimizersdk.DeleteProfileResponse{}, serviceErr
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	trackProfileID(resource, testProfileID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-error-1", got)
	}
}

func TestProfileServiceClientDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	fake := &fakeProfileOCIClient{}
	fake.getFunc = func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error) {
		return optimizersdk.GetProfileResponse{}, serviceErr
	}
	fake.deleteFunc = func(context.Context, optimizersdk.DeleteProfileRequest) (optimizersdk.DeleteProfileResponse, error) {
		t.Fatal("DeleteProfile() called; want pre-delete auth-shaped read to stop delete")
		return optimizersdk.DeleteProfileResponse{}, nil
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	trackProfileID(resource, testProfileID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteProfile() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-pre-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-pre-error-1", got)
	}
}

func TestProfileServiceClientCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	serviceErr.OpcRequestID = "opc-create-error-1"

	fake := &fakeProfileOCIClient{}
	fake.listFunc = func(context.Context, optimizersdk.ListProfilesRequest) (optimizersdk.ListProfilesResponse, error) {
		return optimizersdk.ListProfilesResponse{}, nil
	}
	fake.createFunc = func(context.Context, optimizersdk.CreateProfileRequest) (optimizersdk.CreateProfileResponse, error) {
		return optimizersdk.CreateProfileResponse{}, serviceErr
	}
	client := newProfileRuntimeTestClient(fake)

	resource := newProfileRuntimeTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error-1", got)
	}
}

func newProfileRuntimeTestClient(fake *fakeProfileOCIClient) ProfileServiceClient {
	hooks := newProfileDefaultRuntimeHooks(optimizersdk.OptimizerClient{})
	hooks.Create.Call = fake.CreateProfile
	hooks.Get.Call = fake.GetProfile
	hooks.List.Call = fake.ListProfiles
	hooks.Update.Call = fake.UpdateProfile
	hooks.Delete.Call = fake.DeleteProfile
	applyProfileRuntimeHooks(&hooks)

	manager := &ProfileServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	delegate := defaultProfileServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*optimizerv1beta1.Profile](
			buildProfileGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapProfileGeneratedClient(hooks, delegate)
}

func newProfileRuntimeTestResource() *optimizerv1beta1.Profile {
	return &optimizerv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-sample",
			Namespace: "default",
			UID:       types.UID("profile-uid"),
		},
		Spec: optimizerv1beta1.ProfileSpec{
			CompartmentId:             testProfileCompartmentID,
			Name:                      testProfileName,
			Description:               "runtime profile",
			AggregationIntervalInDays: 60,
			LevelsConfiguration: optimizerv1beta1.ProfileLevelsConfiguration{
				Items: []optimizerv1beta1.ProfileLevelsConfigurationItem{
					{
						RecommendationId: "ocid1.optimizerrecommendation.oc1..recommendation",
						Level:            "HIGH",
					},
				},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			TargetCompartments: optimizerv1beta1.ProfileTargetCompartments{
				Items: []string{"ocid1.compartment.oc1..target"},
			},
			TargetTags: optimizerv1beta1.ProfileTargetTags{
				Items: []optimizerv1beta1.ProfileTargetTagsItem{
					{
						TagNamespaceName:  "Operations",
						TagDefinitionName: "CostCenter",
						TagValueType:      string(optimizersdk.TagValueTypeValue),
						TagValues:         []string{"42"},
					},
				},
			},
		},
	}
}

func trackProfileID(resource *optimizerv1beta1.Profile, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func profileFromSpec(
	id string,
	spec optimizerv1beta1.ProfileSpec,
	state optimizersdk.LifecycleStateEnum,
) optimizersdk.Profile {
	return optimizersdk.Profile{
		Id:                        common.String(id),
		CompartmentId:             common.String(spec.CompartmentId),
		Name:                      common.String(spec.Name),
		Description:               common.String(spec.Description),
		LifecycleState:            state,
		AggregationIntervalInDays: common.Int(spec.AggregationIntervalInDays),
		DefinedTags:               profileDefinedTagsFromSpec(spec.DefinedTags),
		FreeformTags:              mapsClone(spec.FreeformTags),
		SystemTags:                profileDefinedTagsFromSpec(spec.SystemTags),
		LevelsConfiguration:       profileLevelsConfigurationFromSpec(spec.LevelsConfiguration),
		TargetCompartments:        profileTargetCompartmentsFromSpecForTest(spec.TargetCompartments),
		TargetTags:                profileTargetTagsFromSpecForTest(spec.TargetTags),
	}
}

func profileSummaryFromSpec(
	id string,
	spec optimizerv1beta1.ProfileSpec,
	state optimizersdk.LifecycleStateEnum,
) optimizersdk.ProfileSummary {
	return optimizersdk.ProfileSummary{
		Id:                        common.String(id),
		CompartmentId:             common.String(spec.CompartmentId),
		Name:                      common.String(spec.Name),
		Description:               common.String(spec.Description),
		LifecycleState:            state,
		AggregationIntervalInDays: common.Int(spec.AggregationIntervalInDays),
		DefinedTags:               profileDefinedTagsFromSpec(spec.DefinedTags),
		FreeformTags:              mapsClone(spec.FreeformTags),
		SystemTags:                profileDefinedTagsFromSpec(spec.SystemTags),
		LevelsConfiguration:       profileLevelsConfigurationFromSpec(spec.LevelsConfiguration),
		TargetCompartments:        profileTargetCompartmentsFromSpecForTest(spec.TargetCompartments),
		TargetTags:                profileTargetTagsFromSpecForTest(spec.TargetTags),
	}
}

func profileTargetCompartmentsFromSpecForTest(
	spec optimizerv1beta1.ProfileTargetCompartments,
) *optimizersdk.TargetCompartments {
	converted, _ := profileTargetCompartmentsFromSpec(spec)
	return converted
}

func profileTargetTagsFromSpecForTest(spec optimizerv1beta1.ProfileTargetTags) *optimizersdk.TargetTags {
	converted, _ := profileTargetTagsFromSpec(spec)
	return converted
}

func assertProfileCreateRequest(
	t *testing.T,
	request optimizersdk.CreateProfileRequest,
	spec optimizerv1beta1.ProfileSpec,
) {
	t.Helper()

	assertProfileCreateIdentity(t, request, spec)
	assertProfileCreateDetails(t, request, spec)
	assertProfileCreateTargets(t, request, spec)
	assertProfileCreateRetryToken(t, request)
}

func assertProfileCreateIdentity(
	t *testing.T,
	request optimizersdk.CreateProfileRequest,
	spec optimizerv1beta1.ProfileSpec,
) {
	t.Helper()

	if got := profileStringValue(request.CompartmentId); got != spec.CompartmentId {
		t.Fatalf("create compartmentId = %q, want %q", got, spec.CompartmentId)
	}
	if got := profileStringValue(request.Name); got != spec.Name {
		t.Fatalf("create name = %q, want %q", got, spec.Name)
	}
	if got := profileStringValue(request.Description); got != spec.Description {
		t.Fatalf("create description = %q, want %q", got, spec.Description)
	}
}

func assertProfileCreateDetails(
	t *testing.T,
	request optimizersdk.CreateProfileRequest,
	spec optimizerv1beta1.ProfileSpec,
) {
	t.Helper()

	if request.LevelsConfiguration == nil || len(request.LevelsConfiguration.Items) != 1 {
		t.Fatalf("create levelsConfiguration = %#v, want one item", request.LevelsConfiguration)
	}
	if request.AggregationIntervalInDays == nil || *request.AggregationIntervalInDays != spec.AggregationIntervalInDays {
		t.Fatalf("create aggregationIntervalInDays = %v, want %d", request.AggregationIntervalInDays, spec.AggregationIntervalInDays)
	}
	if got := request.FreeformTags; !reflect.DeepEqual(got, spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", got, spec.FreeformTags)
	}
}

func assertProfileCreateTargets(
	t *testing.T,
	request optimizersdk.CreateProfileRequest,
	spec optimizerv1beta1.ProfileSpec,
) {
	t.Helper()

	if request.TargetCompartments == nil || !reflect.DeepEqual(request.TargetCompartments.Items, spec.TargetCompartments.Items) {
		t.Fatalf("create targetCompartments = %#v, want %#v", request.TargetCompartments, spec.TargetCompartments.Items)
	}
	if request.TargetTags == nil || len(request.TargetTags.Items) != 1 ||
		request.TargetTags.Items[0].TagValueType != optimizersdk.TagValueTypeValue {
		t.Fatalf("create targetTags = %#v, want VALUE target tag", request.TargetTags)
	}
}

func assertProfileCreateRetryToken(t *testing.T, request optimizersdk.CreateProfileRequest) {
	t.Helper()

	if request.OpcRetryToken == nil || *request.OpcRetryToken != string(types.UID("profile-uid")) {
		t.Fatalf("create retry token = %v, want resource UID", request.OpcRetryToken)
	}
}

func mapsClone(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
