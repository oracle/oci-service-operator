/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package profile

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testProfileID          = "ocid1.profile.oc1..profile"
	testCompartmentID      = "ocid1.compartment.oc1..profile"
	testLifecycleStageID   = "ocid1.lifecycle.stage.oc1..stage"
	testManagedInstanceID  = "ocid1.managedinstancegroup.oc1..group"
	testSoftwareSourceID   = "ocid1.softwaresource.oc1..source"
	testSoftwareSourceID2  = "ocid1.softwaresource.oc1..source2"
	testProfileDisplayName = "profile-one"
)

type fakeProfileOCIClient struct {
	createProfile func(context.Context, osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error)
	getProfile    func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error)
	listProfiles  func(context.Context, osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error)
	updateProfile func(context.Context, osmanagementhubsdk.UpdateProfileRequest) (osmanagementhubsdk.UpdateProfileResponse, error)
	deleteProfile func(context.Context, osmanagementhubsdk.DeleteProfileRequest) (osmanagementhubsdk.DeleteProfileResponse, error)

	createRequests []osmanagementhubsdk.CreateProfileRequest
	getRequests    []osmanagementhubsdk.GetProfileRequest
	listRequests   []osmanagementhubsdk.ListProfilesRequest
	updateRequests []osmanagementhubsdk.UpdateProfileRequest
	deleteRequests []osmanagementhubsdk.DeleteProfileRequest
}

func (f *fakeProfileOCIClient) CreateProfile(ctx context.Context, request osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createProfile != nil {
		return f.createProfile(ctx, request)
	}
	return osmanagementhubsdk.CreateProfileResponse{}, nil
}

func (f *fakeProfileOCIClient) GetProfile(ctx context.Context, request osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getProfile != nil {
		return f.getProfile(ctx, request)
	}
	return osmanagementhubsdk.GetProfileResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "profile not found")
}

func (f *fakeProfileOCIClient) ListProfiles(ctx context.Context, request osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listProfiles != nil {
		return f.listProfiles(ctx, request)
	}
	return osmanagementhubsdk.ListProfilesResponse{}, nil
}

func (f *fakeProfileOCIClient) UpdateProfile(ctx context.Context, request osmanagementhubsdk.UpdateProfileRequest) (osmanagementhubsdk.UpdateProfileResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateProfile != nil {
		return f.updateProfile(ctx, request)
	}
	return osmanagementhubsdk.UpdateProfileResponse{}, nil
}

func (f *fakeProfileOCIClient) DeleteProfile(ctx context.Context, request osmanagementhubsdk.DeleteProfileRequest) (osmanagementhubsdk.DeleteProfileResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteProfile != nil {
		return f.deleteProfile(ctx, request)
	}
	return osmanagementhubsdk.DeleteProfileResponse{OpcRequestId: common.String("delete-request")}, nil
}

func TestProfileServiceClientCreatesSoftwareSourceProfile(t *testing.T) {
	resource := testProfileResource()
	client := &fakeProfileOCIClient{}
	client.createProfile = func(_ context.Context, request osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
		assertSoftwareSourceCreateRequest(t, request)
		return osmanagementhubsdk.CreateProfileResponse{
			Profile:      testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateCreating, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			OpcRequestId: common.String("create-request"),
		}, nil
	}
	client.getProfile = func(_ context.Context, request osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		if got := profileString(request.ProfileId); got != testProfileID {
			t.Fatalf("GetProfile profileId = %q, want %q", got, testProfileID)
		}
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateProfile calls = %d, want 1", len(client.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProfileID {
		t.Fatalf("status.ocid = %q, want %q", got, testProfileID)
	}
}

func TestProfileCreateBuildsGroupBody(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.ProfileType = string(osmanagementhubsdk.ProfileTypeGroup)
	resource.Spec.ManagedInstanceGroupId = testManagedInstanceID
	resource.Spec.SoftwareSourceIds = nil
	resource.Spec.VendorName = ""
	resource.Spec.OsFamily = ""
	resource.Spec.ArchType = ""

	body := buildTestProfileCreateBody(t, resource)
	typed, ok := body.(osmanagementhubsdk.CreateGroupProfileDetails)
	if !ok {
		t.Fatalf("CreateProfileDetails = %T, want CreateGroupProfileDetails", body)
	}
	assertCommonCreateDetails(t, typed, osmanagementhubsdk.ProfileRegistrationTypeOciLinux)
	if got := profileString(typed.ManagedInstanceGroupId); got != testManagedInstanceID {
		t.Fatalf("create managedInstanceGroupId = %q, want %q", got, testManagedInstanceID)
	}
}

func TestProfileCreateBuildsLifecycleBody(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.ProfileType = string(osmanagementhubsdk.ProfileTypeLifecycle)
	resource.Spec.LifecycleStageId = testLifecycleStageID
	resource.Spec.SoftwareSourceIds = nil
	resource.Spec.VendorName = ""
	resource.Spec.OsFamily = ""
	resource.Spec.ArchType = ""

	body := buildTestProfileCreateBody(t, resource)
	typed, ok := body.(osmanagementhubsdk.CreateLifecycleProfileDetails)
	if !ok {
		t.Fatalf("CreateProfileDetails = %T, want CreateLifecycleProfileDetails", body)
	}
	assertCommonCreateDetails(t, typed, osmanagementhubsdk.ProfileRegistrationTypeOciLinux)
	if got := profileString(typed.LifecycleStageId); got != testLifecycleStageID {
		t.Fatalf("create lifecycleStageId = %q, want %q", got, testLifecycleStageID)
	}
}

func TestProfileCreateBuildsStationBody(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.ProfileType = string(osmanagementhubsdk.ProfileTypeStation)
	resource.Spec.SoftwareSourceIds = nil

	body := buildTestProfileCreateBody(t, resource)
	typed, ok := body.(osmanagementhubsdk.CreateStationProfileDetails)
	if !ok {
		t.Fatalf("CreateProfileDetails = %T, want CreateStationProfileDetails", body)
	}
	assertCommonCreateDetails(t, typed, osmanagementhubsdk.ProfileRegistrationTypeOciLinux)
	assertCreatePlatform(t, typed.VendorName, typed.OsFamily, typed.ArchType)
}

func TestProfileCreateBuildsWindowsStandaloneBody(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.ProfileType = string(osmanagementhubsdk.ProfileTypeWindowsStandalone)
	resource.Spec.RegistrationType = string(osmanagementhubsdk.ProfileRegistrationTypeOciWindows)
	resource.Spec.VendorName = string(osmanagementhubsdk.VendorNameMicrosoft)
	resource.Spec.OsFamily = string(osmanagementhubsdk.OsFamilyWindowsServer2022)
	resource.Spec.ArchType = string(osmanagementhubsdk.ArchTypeX8664)
	resource.Spec.SoftwareSourceIds = nil

	body := buildTestProfileCreateBody(t, resource)
	typed, ok := body.(osmanagementhubsdk.CreateWindowsStandAloneProfileDetails)
	if !ok {
		t.Fatalf("CreateProfileDetails = %T, want CreateWindowsStandAloneProfileDetails", body)
	}
	assertCommonCreateDetails(t, typed, osmanagementhubsdk.ProfileRegistrationTypeOciWindows)
	if got := typed.RegistrationType; got != osmanagementhubsdk.ProfileRegistrationTypeOciWindows {
		t.Fatalf("create registrationType = %q, want %q", got, osmanagementhubsdk.ProfileRegistrationTypeOciWindows)
	}
	if got := typed.VendorName; got != osmanagementhubsdk.VendorNameMicrosoft {
		t.Fatalf("create vendorName = %q, want %q", got, osmanagementhubsdk.VendorNameMicrosoft)
	}
	if got := typed.OsFamily; got != osmanagementhubsdk.OsFamilyWindowsServer2022 {
		t.Fatalf("create osFamily = %q, want %q", got, osmanagementhubsdk.OsFamilyWindowsServer2022)
	}
	if got := typed.ArchType; got != osmanagementhubsdk.ArchTypeX8664 {
		t.Fatalf("create archType = %q, want %q", got, osmanagementhubsdk.ArchTypeX8664)
	}
}

func buildTestProfileCreateBody(t *testing.T, resource *osmanagementhubv1beta1.Profile) any {
	t.Helper()
	body, err := buildProfileCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildProfileCreateBody() error = %v", err)
	}
	return body
}

type testProfileCommonCreateDetails interface {
	GetDisplayName() *string
	GetCompartmentId() *string
	GetDescription() *string
	GetRegistrationType() osmanagementhubsdk.ProfileRegistrationTypeEnum
	GetIsDefaultProfile() *bool
	GetFreeformTags() map[string]string
	GetDefinedTags() map[string]map[string]interface{}
}

func assertCommonCreateDetails(
	t *testing.T,
	body testProfileCommonCreateDetails,
	wantRegistrationType osmanagementhubsdk.ProfileRegistrationTypeEnum,
) {
	t.Helper()
	if got := profileString(body.GetDisplayName()); got != testProfileDisplayName {
		t.Fatalf("create displayName = %q, want %q", got, testProfileDisplayName)
	}
	if got := profileString(body.GetCompartmentId()); got != testCompartmentID {
		t.Fatalf("create compartmentId = %q, want %q", got, testCompartmentID)
	}
	if got := profileString(body.GetDescription()); got != "profile description" {
		t.Fatalf("create description = %q, want profile description", got)
	}
	if got := body.GetRegistrationType(); got != wantRegistrationType {
		t.Fatalf("create registrationType = %q, want %q", got, wantRegistrationType)
	}
	if got := body.GetIsDefaultProfile(); got == nil || *got {
		t.Fatalf("create isDefaultProfile = %v, want false pointer", got)
	}
	if got := body.GetFreeformTags(); got["owner"] != "osok" {
		t.Fatalf("create freeformTags[owner] = %q, want osok", got["owner"])
	}
	if got := body.GetDefinedTags()["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %v, want 42", got)
	}
}

func assertCreatePlatform(
	t *testing.T,
	vendorName osmanagementhubsdk.VendorNameEnum,
	osFamily osmanagementhubsdk.OsFamilyEnum,
	archType osmanagementhubsdk.ArchTypeEnum,
) {
	t.Helper()
	if vendorName != osmanagementhubsdk.VendorNameOracle {
		t.Fatalf("create vendorName = %q, want %q", vendorName, osmanagementhubsdk.VendorNameOracle)
	}
	if osFamily != osmanagementhubsdk.OsFamilyOracleLinux8 {
		t.Fatalf("create osFamily = %q, want %q", osFamily, osmanagementhubsdk.OsFamilyOracleLinux8)
	}
	if archType != osmanagementhubsdk.ArchTypeX8664 {
		t.Fatalf("create archType = %q, want %q", archType, osmanagementhubsdk.ArchTypeX8664)
	}
}

func TestProfileServiceClientBindsExistingAcrossListPages(t *testing.T) {
	resource := testProfileResource()
	client := &fakeProfileOCIClient{}
	client.listProfiles = func(_ context.Context, request osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error) {
		if profileString(request.Page) == "" {
			return osmanagementhubsdk.ListProfilesResponse{
				ProfileCollection: osmanagementhubsdk.ProfileCollection{Items: nil},
				OpcNextPage:       common.String("next"),
			}, nil
		}
		return osmanagementhubsdk.ListProfilesResponse{
			ProfileCollection: osmanagementhubsdk.ProfileCollection{Items: []osmanagementhubsdk.ProfileSummary{
				testProfileSummary(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive),
			}},
		}, nil
	}
	client.getProfile = func(_ context.Context, request osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		if got := profileString(request.ProfileId); got != testProfileID {
			t.Fatalf("GetProfile profileId = %q, want %q", got, testProfileID)
		}
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}

	if _, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource)); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListProfiles calls = %d, want 2", len(client.listRequests))
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateProfile calls = %d, want 0 for bind", len(client.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProfileID {
		t.Fatalf("status.ocid = %q, want bound OCID %q", got, testProfileID)
	}
}

func TestProfileServiceClientCreatesWhenSameNameSoftwareSourceHasDifferentSources(t *testing.T) {
	const mismatchedProfileID = "ocid1.profile.oc1..mismatched"
	resource := testProfileResource()
	client := &fakeProfileOCIClient{}
	client.listProfiles = func(context.Context, osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error) {
		return osmanagementhubsdk.ListProfilesResponse{
			ProfileCollection: osmanagementhubsdk.ProfileCollection{Items: []osmanagementhubsdk.ProfileSummary{
				testProfileSummary(mismatchedProfileID, osmanagementhubsdk.ProfileLifecycleStateActive),
			}},
		}, nil
	}
	client.getProfile = func(_ context.Context, request osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		switch got := profileString(request.ProfileId); got {
		case mismatchedProfileID:
			return osmanagementhubsdk.GetProfileResponse{
				Profile: testSoftwareSourceProfile(mismatchedProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, []string{testSoftwareSourceID2}),
			}, nil
		case testProfileID:
			return osmanagementhubsdk.GetProfileResponse{
				Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			}, nil
		default:
			t.Fatalf("GetProfile profileId = %q, want %q or %q", got, mismatchedProfileID, testProfileID)
		}
		return osmanagementhubsdk.GetProfileResponse{}, nil
	}
	client.createProfile = func(_ context.Context, request osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
		assertSoftwareSourceCreateRequest(t, request)
		return osmanagementhubsdk.CreateProfileResponse{
			Profile:      testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateCreating, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			OpcRequestId: common.String("create-request"),
		}, nil
	}

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateProfile calls = %d, want 1 after mismatched list candidate", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateProfile calls = %d, want 0 after mismatched list candidate", len(client.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProfileID {
		t.Fatalf("status.ocid = %q, want created OCID %q", got, testProfileID)
	}
}

func TestProfileServiceClientNoOpReconcileDoesNotUpdate(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}

	if _, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource)); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateProfile calls = %d, want 0", len(client.updateRequests))
	}
}

func TestProfileServiceClientRecreatesTrackedDeletedProfile(t *testing.T) {
	const replacementProfileID = "ocid1.profile.oc1..replacement"
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	client := &fakeProfileOCIClient{}
	client.getProfile = func(_ context.Context, request osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		switch got := profileString(request.ProfileId); got {
		case testProfileID:
			return osmanagementhubsdk.GetProfileResponse{
				Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateDeleted, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			}, nil
		case replacementProfileID:
			return osmanagementhubsdk.GetProfileResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "profile not found")
		default:
			t.Fatalf("GetProfile profileId = %q, want %q or %q", got, testProfileID, replacementProfileID)
		}
		return osmanagementhubsdk.GetProfileResponse{}, nil
	}
	client.createProfile = func(context.Context, osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
		return osmanagementhubsdk.CreateProfileResponse{
			Profile:      testSoftwareSourceProfile(replacementProfileID, osmanagementhubsdk.ProfileLifecycleStateCreating, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			OpcRequestId: common.String("create-request"),
		}, nil
	}

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true for replacement create")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateProfile calls = %d, want 1 after tracked DELETED readback", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateProfile calls = %d, want 0 after tracked DELETED readback", len(client.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != replacementProfileID {
		t.Fatalf("status.ocid = %q, want replacement OCID %q", got, replacementProfileID)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Provisioning) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Provisioning)
	}
}

func TestProfileServiceClientMutableUpdateShapesUpdateBody(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.Description = "new description"
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	getResponses := []osmanagementhubsdk.GetProfileResponse{
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, "old description", resource.Spec.SoftwareSourceIds)},
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, "new description", resource.Spec.SoftwareSourceIds)},
	}
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		if len(getResponses) == 0 {
			t.Fatal("GetProfile called more times than expected")
		}
		response := getResponses[0]
		getResponses = getResponses[1:]
		return response, nil
	}
	client.updateProfile = func(_ context.Context, request osmanagementhubsdk.UpdateProfileRequest) (osmanagementhubsdk.UpdateProfileResponse, error) {
		if got := profileString(request.ProfileId); got != testProfileID {
			t.Fatalf("UpdateProfile profileId = %q, want %q", got, testProfileID)
		}
		if got := profileString(request.Description); got != "new description" {
			t.Fatalf("update description = %q, want %q", got, "new description")
		}
		return osmanagementhubsdk.UpdateProfileResponse{
			Profile:      testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateUpdating, "new description", resource.Spec.SoftwareSourceIds),
			OpcRequestId: common.String("update-request"),
		}, nil
	}

	if _, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource)); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateProfile calls = %d, want 1", len(client.updateRequests))
	}
}

func TestProfileServiceClientProjectsCreateLifecycleAsync(t *testing.T) {
	resource := testProfileResource()
	client := &fakeProfileOCIClient{}
	client.createProfile = func(context.Context, osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
		return osmanagementhubsdk.CreateProfileResponse{
			Profile:      testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateCreating, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			OpcRequestId: common.String("create-request"),
		}, nil
	}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateCreating, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true for CREATING lifecycle")
	}
	requireProfileLifecycleAsync(t, resource, shared.OSOKAsyncPhaseCreate, string(osmanagementhubsdk.ProfileLifecycleStateCreating), shared.OSOKAsyncClassPending)
}

func TestProfileServiceClientProjectsUpdateLifecycleAsync(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.Description = "new description"
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	getResponses := []osmanagementhubsdk.GetProfileResponse{
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, "old description", resource.Spec.SoftwareSourceIds)},
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateUpdating, "new description", resource.Spec.SoftwareSourceIds)},
	}
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		if len(getResponses) == 0 {
			t.Fatal("GetProfile called more times than expected")
		}
		response := getResponses[0]
		getResponses = getResponses[1:]
		return response, nil
	}
	client.updateProfile = func(context.Context, osmanagementhubsdk.UpdateProfileRequest) (osmanagementhubsdk.UpdateProfileResponse, error) {
		return osmanagementhubsdk.UpdateProfileResponse{
			Profile:      testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateUpdating, "new description", resource.Spec.SoftwareSourceIds),
			OpcRequestId: common.String("update-request"),
		}, nil
	}

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true for UPDATING lifecycle")
	}
	requireProfileLifecycleAsync(t, resource, shared.OSOKAsyncPhaseUpdate, string(osmanagementhubsdk.ProfileLifecycleStateUpdating), shared.OSOKAsyncClassPending)
}

func TestProfileServiceClientClearsLifecycleAsyncWhenActive(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		RawStatus:       string(osmanagementhubsdk.ProfileLifecycleStateUpdating),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = true, want false for ACTIVE lifecycle")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after ACTIVE readback", resource.Status.OsokStatus.Async.Current)
	}
}

func TestProfileServiceClientProjectsDeletingLifecycleAsync(t *testing.T) {
	resource := testTrackedProfileResource()
	client := testProfileLifecycleClient(resource, osmanagementhubsdk.ProfileLifecycleStateDeleting)

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	requireProfileLifecycleAsync(t, resource, shared.OSOKAsyncPhaseDelete, string(osmanagementhubsdk.ProfileLifecycleStateDeleting), shared.OSOKAsyncClassPending)
}

func TestProfileServiceClientProjectsFailedLifecycleAsyncPreservesPendingUpdatePhase(t *testing.T) {
	resource := testTrackedProfileResource()
	seedProfileLifecycleAsync(resource, shared.OSOKAsyncPhaseUpdate)
	client := testProfileLifecycleClient(resource, osmanagementhubsdk.ProfileLifecycleStateFailed)

	response, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want status failure response without returned error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false for FAILED lifecycle")
	}
	requireProfileLifecycleAsync(t, resource, shared.OSOKAsyncPhaseUpdate, string(osmanagementhubsdk.ProfileLifecycleStateFailed), shared.OSOKAsyncClassFailed)
}

func testTrackedProfileResource() *osmanagementhubv1beta1.Profile {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	return resource
}

func seedProfileLifecycleAsync(resource *osmanagementhubv1beta1.Profile, phase shared.OSOKAsyncPhase) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       string(osmanagementhubsdk.ProfileLifecycleStateUpdating),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func testProfileLifecycleClient(
	resource *osmanagementhubv1beta1.Profile,
	state osmanagementhubsdk.ProfileLifecycleStateEnum,
) *fakeProfileOCIClient {
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, state, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}
	return client
}

func assertSoftwareSourceCreateRequest(t *testing.T, request osmanagementhubsdk.CreateProfileRequest) {
	t.Helper()
	body, ok := request.CreateProfileDetails.(osmanagementhubsdk.CreateSoftwareSourceProfileDetails)
	if !ok {
		t.Fatalf("CreateProfileDetails = %T, want CreateSoftwareSourceProfileDetails", request.CreateProfileDetails)
	}
	if got := profileString(body.DisplayName); got != testProfileDisplayName {
		t.Fatalf("create displayName = %q, want %q", got, testProfileDisplayName)
	}
	if got := profileString(body.CompartmentId); got != testCompartmentID {
		t.Fatalf("create compartmentId = %q, want %q", got, testCompartmentID)
	}
	if got := body.SoftwareSourceIds; len(got) != 1 || got[0] != testSoftwareSourceID {
		t.Fatalf("create softwareSourceIds = %#v, want [%q]", got, testSoftwareSourceID)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateProfile OpcRetryToken is empty, want deterministic retry token")
	}
}

func TestProfileRetryTokenUsesUIDBeforeNamespacedNameAndSpecFields(t *testing.T) {
	resource := testProfileResource()
	resource.UID = types.UID("profile-uid")
	got := profileRetryToken(resource, "request-namespace")

	resource.Namespace = "changed-namespace"
	resource.Name = "changed-name"
	resource.Spec.DisplayName = "changed-display-name"
	resource.Spec.ProfileType = string(osmanagementhubsdk.ProfileTypeGroup)
	if changed := profileRetryToken(resource, "changed-request-namespace"); changed != got {
		t.Fatalf("profileRetryToken() changed after UID-backed resource fields mutated: got %q, want %q", changed, got)
	}

	fallback := testProfileResource()
	fallbackToken := profileRetryToken(fallback, "request-namespace")
	fallback.Spec.DisplayName = "changed-display-name"
	fallback.Spec.ProfileType = string(osmanagementhubsdk.ProfileTypeGroup)
	if changed := profileRetryToken(fallback, "request-namespace"); changed != fallbackToken {
		t.Fatalf("profileRetryToken() fallback changed after spec fields mutated: got %q, want %q", changed, fallbackToken)
	}
	fallback.Name = "changed-name"
	if changed := profileRetryToken(fallback, "request-namespace"); changed == fallbackToken {
		t.Fatalf("profileRetryToken() fallback stayed %q after name changed, want namespace/name fallback", changed)
	}
}

func requireProfileLifecycleAsync(
	t *testing.T,
	resource *osmanagementhubv1beta1.Profile,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle operation")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.Message != "OCI Profile lifecycle state is "+rawStatus {
		t.Fatalf("status.async.current.message = %q, want lifecycle message for %s", current.Message, rawStatus)
	}
	if current.UpdatedAt == nil {
		t.Fatal("status.async.current.updatedAt = nil, want timestamp")
	}
}

func TestProfileServiceClientRejectsSoftwareSourcesDriftBeforeUpdate(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.SoftwareSourceIds = []string{testSoftwareSourceID2}
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, []string{testSoftwareSourceID}),
		}, nil
	}

	_, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift")
	}
	if !strings.Contains(err.Error(), "softwareSourceIds") {
		t.Fatalf("CreateOrUpdate() error = %v, want softwareSourceIds drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateProfile calls = %d, want 0", len(client.updateRequests))
	}
}

func TestProfileServiceClientRejectsOptionalForceNewClearBeforeUpdate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*osmanagementhubv1beta1.Profile)
		wantField string
	}{
		{
			name: "omitted softwareSourceIds",
			mutate: func(resource *osmanagementhubv1beta1.Profile) {
				resource.Spec.SoftwareSourceIds = nil
			},
			wantField: "softwareSourceIds",
		},
		{
			name: "cleared softwareSourceIds",
			mutate: func(resource *osmanagementhubv1beta1.Profile) {
				resource.Spec.SoftwareSourceIds = []string{}
			},
			wantField: "softwareSourceIds",
		},
		{
			name: "omitted vendorName",
			mutate: func(resource *osmanagementhubv1beta1.Profile) {
				resource.Spec.VendorName = ""
			},
			wantField: "vendorName",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource := testProfileResource()
			test.mutate(resource)
			resource.Spec.Description = "new description"
			resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
			client := &fakeProfileOCIClient{}
			client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
				return osmanagementhubsdk.GetProfileResponse{
					Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, "old description", []string{testSoftwareSourceID}),
				}, nil
			}

			_, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want create-only drift")
			}
			if !strings.Contains(err.Error(), test.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want %s drift", err, test.wantField)
			}
			if len(client.updateRequests) != 0 {
				t.Fatalf("UpdateProfile calls = %d, want 0", len(client.updateRequests))
			}
		})
	}
}

func TestProfileServiceClientRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	resource := testProfileResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..other"
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}

	_, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want compartmentId create-only drift")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateProfile calls = %d, want 0", len(client.updateRequests))
	}
}

func TestProfileDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	getResponses := []osmanagementhubsdk.GetProfileResponse{
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds)},
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateDeleting, resource.Spec.Description, resource.Spec.SoftwareSourceIds)},
	}
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		if len(getResponses) == 0 {
			t.Fatal("GetProfile called more times than expected")
		}
		response := getResponses[0]
		getResponses = getResponses[1:]
		return response, nil
	}

	deleted, err := newProfileServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteProfile calls = %d, want 1", len(client.deleteRequests))
	}
}

func TestProfileDeleteReleasesFinalizerAfterPostDeleteNotFound(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	getCalls := 0
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		getCalls++
		if getCalls == 1 {
			return osmanagementhubsdk.GetProfileResponse{
				Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
			}, nil
		}
		return osmanagementhubsdk.GetProfileResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "profile not found")
	}

	deleted, err := newProfileServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after post-delete NotFound")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteProfile calls = %d, want 1", len(client.deleteRequests))
	}
	if getCalls != 2 {
		t.Fatalf("GetProfile calls = %d, want 2", getCalls)
	}
}

func TestProfileDeleteReleasesFinalizerAfterDeletedReadback(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	getResponses := []osmanagementhubsdk.GetProfileResponse{
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds)},
		{Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateDeleted, resource.Spec.Description, resource.Spec.SoftwareSourceIds)},
	}
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		if len(getResponses) == 0 {
			t.Fatal("GetProfile called more times than expected")
		}
		response := getResponses[0]
		getResponses = getResponses[1:]
		return response, nil
	}

	deleted, err := newProfileServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteProfile calls = %d, want 1", len(client.deleteRequests))
	}
}

func TestProfileDeleteRecordsDirectDeleteErrorRequestID(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{
			Profile: testSoftwareSourceProfile(testProfileID, osmanagementhubsdk.ProfileLifecycleStateActive, resource.Spec.Description, resource.Spec.SoftwareSourceIds),
		}, nil
	}
	client.deleteProfile = func(context.Context, osmanagementhubsdk.DeleteProfileRequest) (osmanagementhubsdk.DeleteProfileResponse, error) {
		return osmanagementhubsdk.DeleteProfileResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "delete failed")
	}

	deleted, err := newProfileServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want service error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteProfile calls = %d, want 1", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want service error request id", got)
	}
}

func TestProfileDeleteRejectsAuthShapedConfirmReadBeforeDelete(t *testing.T) {
	resource := testProfileResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProfileID)
	client := &fakeProfileOCIClient{}
	client.getProfile = func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
		return osmanagementhubsdk.GetProfileResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newProfileServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteProfile calls = %d, want 0 after ambiguous confirm read", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want service error request id", got)
	}
}

func TestProfileDeleteReleasesFinalizerWhenUntrackedListMisses(t *testing.T) {
	resource := testProfileResource()
	client := &fakeProfileOCIClient{}

	deleted, err := newProfileServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after untracked list miss")
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("ListProfiles calls = %d, want 1", len(client.listRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("GetProfile calls = %d, want 0 after untracked list miss", len(client.getRequests))
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteProfile calls = %d, want 0 after untracked list miss", len(client.deleteRequests))
	}
}

func TestProfileCreateRecordsOCIErrorRequestID(t *testing.T) {
	resource := testProfileResource()
	client := &fakeProfileOCIClient{}
	client.createProfile = func(context.Context, osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
		return osmanagementhubsdk.CreateProfileResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	}

	_, err := newProfileServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, testProfileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want service error")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want service error request id", got)
	}
}

func testProfileRequest(resource *osmanagementhubv1beta1.Profile) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func testProfileResource() *osmanagementhubv1beta1.Profile {
	return &osmanagementhubv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile",
			Namespace: "default",
		},
		Spec: osmanagementhubv1beta1.ProfileSpec{
			DisplayName:       testProfileDisplayName,
			CompartmentId:     testCompartmentID,
			Description:       "profile description",
			ProfileType:       string(osmanagementhubsdk.ProfileTypeSoftwaresource),
			RegistrationType:  string(osmanagementhubsdk.ProfileRegistrationTypeOciLinux),
			IsDefaultProfile:  false,
			VendorName:        string(osmanagementhubsdk.VendorNameOracle),
			OsFamily:          string(osmanagementhubsdk.OsFamilyOracleLinux8),
			ArchType:          string(osmanagementhubsdk.ArchTypeX8664),
			SoftwareSourceIds: []string{testSoftwareSourceID},
			FreeformTags: map[string]string{
				"owner": "osok",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func testProfileSummary(id string, state osmanagementhubsdk.ProfileLifecycleStateEnum) osmanagementhubsdk.ProfileSummary {
	return osmanagementhubsdk.ProfileSummary{
		Id:                       common.String(id),
		CompartmentId:            common.String(testCompartmentID),
		DisplayName:              common.String(testProfileDisplayName),
		Description:              common.String("profile description"),
		ProfileType:              osmanagementhubsdk.ProfileTypeSoftwaresource,
		RegistrationType:         osmanagementhubsdk.ProfileRegistrationTypeOciLinux,
		VendorName:               osmanagementhubsdk.VendorNameOracle,
		OsFamily:                 osmanagementhubsdk.OsFamilyOracleLinux8,
		ArchType:                 osmanagementhubsdk.ArchTypeX8664,
		LifecycleState:           state,
		IsDefaultProfile:         common.Bool(false),
		IsServiceProvidedProfile: common.Bool(false),
		FreeformTags: map[string]string{
			"owner": "osok",
		},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {
				"CostCenter": "42",
			},
		},
	}
}

func testSoftwareSourceProfile(
	id string,
	state osmanagementhubsdk.ProfileLifecycleStateEnum,
	description string,
	softwareSourceIDs []string,
) osmanagementhubsdk.SoftwareSourceProfile {
	sources := make([]osmanagementhubsdk.SoftwareSourceDetails, 0, len(softwareSourceIDs))
	for _, sourceID := range softwareSourceIDs {
		sources = append(sources, osmanagementhubsdk.SoftwareSourceDetails{Id: common.String(sourceID)})
	}
	return osmanagementhubsdk.SoftwareSourceProfile{
		Id:                       common.String(id),
		CompartmentId:            common.String(testCompartmentID),
		DisplayName:              common.String(testProfileDisplayName),
		SoftwareSources:          sources,
		Description:              common.String(description),
		LifecycleState:           state,
		RegistrationType:         osmanagementhubsdk.ProfileRegistrationTypeOciLinux,
		IsDefaultProfile:         common.Bool(false),
		IsServiceProvidedProfile: common.Bool(false),
		FreeformTags: map[string]string{
			"owner": "osok",
		},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {
				"CostCenter": "42",
			},
		},
		VendorName: osmanagementhubsdk.VendorNameOracle,
		OsFamily:   osmanagementhubsdk.OsFamilyOracleLinux8,
		ArchType:   osmanagementhubsdk.ArchTypeX8664,
	}
}
