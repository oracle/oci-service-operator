/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package profile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const profileRequeueDuration = time.Minute

type profileOCIClient interface {
	CreateProfile(context.Context, osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error)
	GetProfile(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error)
	ListProfiles(context.Context, osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error)
	UpdateProfile(context.Context, osmanagementhubsdk.UpdateProfileRequest) (osmanagementhubsdk.UpdateProfileResponse, error)
	DeleteProfile(context.Context, osmanagementhubsdk.DeleteProfileRequest) (osmanagementhubsdk.DeleteProfileResponse, error)
}

type profileListCall func(context.Context, osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error)
type profileGetCall func(context.Context, osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error)

type profileRuntimeClient struct {
	delegate ProfileServiceClient
	hooks    ProfileRuntimeHooks
}

var _ ProfileServiceClient = (*profileRuntimeClient)(nil)

type profileDesiredFields struct {
	DisplayName            string
	CompartmentID          string
	Description            string
	ManagementStationID    string
	RegistrationType       string
	IsDefaultProfile       bool
	HasIsDefaultProfile    bool
	FreeformTags           map[string]string
	HasFreeformTags        bool
	DefinedTags            map[string]map[string]interface{}
	HasDefinedTags         bool
	ProfileType            string
	ManagedInstanceGroupID string
	VendorName             string
	OsFamily               string
	ArchType               string
	SoftwareSourceIDs      []string
	LifecycleStageID       string
}

type profileJSONFields struct {
	DisplayName            *string                           `json:"displayName"`
	CompartmentID          *string                           `json:"compartmentId"`
	Description            *string                           `json:"description"`
	ManagementStationID    *string                           `json:"managementStationId"`
	RegistrationType       *string                           `json:"registrationType"`
	IsDefaultProfile       *bool                             `json:"isDefaultProfile"`
	FreeformTags           map[string]string                 `json:"freeformTags"`
	DefinedTags            map[string]map[string]interface{} `json:"definedTags"`
	ProfileType            *string                           `json:"profileType"`
	ManagedInstanceGroupID *string                           `json:"managedInstanceGroupId"`
	VendorName             *string                           `json:"vendorName"`
	OsFamily               *string                           `json:"osFamily"`
	ArchType               *string                           `json:"archType"`
	SoftwareSourceIDs      []string                          `json:"softwareSourceIds"`
	LifecycleStageID       *string                           `json:"lifecycleStageId"`
}

type profileAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e profileAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e profileAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerProfileRuntimeHooksMutator(func(_ *ProfileServiceManager, hooks *ProfileRuntimeHooks) {
		applyProfileRuntimeHooks(hooks)
	})
}

func applyProfileRuntimeHooks(hooks *ProfileRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedProfileRuntimeSemantics()
	hooks.BuildCreateBody = buildProfileCreateBody
	hooks.BuildUpdateBody = buildProfileUpdateBody
	hooks.List.Fields = profileListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listProfilePages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateProfileCreateOnlyDrift
	hooks.DeleteHooks.ConfirmRead = profileDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleProfileDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyProfileDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProfileServiceClient) ProfileServiceClient {
		return &profileRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
		}
	})
}

func (c *profileRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validateCreateOrUpdateInput(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if response, handled, err := c.reconcileTrackedProfile(ctx, resource); handled || err != nil {
		return response, err
	}
	if response, handled, err := c.reconcileListedProfile(ctx, resource); handled || err != nil {
		return response, err
	}

	return c.createProfile(ctx, resource, req)
}

func (c *profileRuntimeClient) Delete(ctx context.Context, resource *osmanagementhubv1beta1.Profile) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("profile runtime client is not configured")
	}
	if resource == nil {
		return false, fmt.Errorf("profile resource is nil")
	}
	if profileTrackedID(resource) == "" {
		found, err := c.seedProfileDeleteIDFromList(ctx, resource)
		if err != nil {
			return false, err
		}
		if !found {
			return true, nil
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *profileRuntimeClient) validateCreateOrUpdateInput(resource *osmanagementhubv1beta1.Profile) error {
	if c == nil || c.delegate == nil {
		return fmt.Errorf("profile runtime client is not configured")
	}
	if resource == nil {
		return fmt.Errorf("profile resource is nil")
	}
	return nil
}

func (c *profileRuntimeClient) reconcileTrackedProfile(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
) (servicemanager.OSOKResponse, bool, error) {
	currentID := profileTrackedID(resource)
	if currentID == "" {
		return servicemanager.OSOKResponse{}, false, nil
	}
	current, found, err := c.getProfile(ctx, currentID)
	if err != nil {
		response, err := profileFail(resource, err)
		return response, true, err
	}
	if found {
		if profileLifecycleDeleted(current.GetLifecycleState()) {
			clearProfileTrackedID(resource)
			return servicemanager.OSOKResponse{}, false, nil
		}
		response, err := c.reconcileExistingProfile(ctx, resource, currentID, current)
		return response, true, err
	}
	clearProfileTrackedID(resource)
	return servicemanager.OSOKResponse{}, false, nil
}

func (c *profileRuntimeClient) reconcileListedProfile(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
) (servicemanager.OSOKResponse, bool, error) {
	existing, found, err := c.lookupExistingProfile(ctx, resource)
	if err != nil {
		response, err := profileFail(resource, err)
		return response, true, err
	}
	if !found {
		return servicemanager.OSOKResponse{}, false, nil
	}
	response, err := c.reconcileFoundProfile(ctx, resource, existing)
	return response, true, err
}

func (c *profileRuntimeClient) reconcileFoundProfile(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
	existing osmanagementhubsdk.Profile,
) (servicemanager.OSOKResponse, error) {
	existingID := profileString(existing.GetId())
	if existingID == "" {
		return profileFail(resource, fmt.Errorf("profile list response did not expose a resource OCID"))
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID

	current, found, err := c.getProfile(ctx, existingID)
	if err != nil {
		return profileFail(resource, err)
	}
	if found {
		return c.reconcileExistingProfile(ctx, resource, existingID, current)
	}
	return c.reconcileExistingProfile(ctx, resource, existingID, existing)
}

func profileTrackedID(resource *osmanagementhubv1beta1.Profile) string {
	currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if currentID != "" {
		return currentID
	}
	return strings.TrimSpace(resource.Status.Id)
}

func clearProfileTrackedID(resource *osmanagementhubv1beta1.Profile) {
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
}

func (c *profileRuntimeClient) reconcileExistingProfile(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
	currentID string,
	current osmanagementhubsdk.Profile,
) (servicemanager.OSOKResponse, error) {
	if profileLifecycleDeleted(current.GetLifecycleState()) {
		return profileFail(resource, fmt.Errorf("profile %s is already deleted in OCI", currentID))
	}
	if profileLifecycleInProgress(current.GetLifecycleState()) {
		return profileProjectSuccess(resource, current, profileConditionForLifecycle(current.GetLifecycleState()), "")
	}
	if err := validateProfileCreateOnlyDrift(resource, current); err != nil {
		return profileFail(resource, err)
	}

	body, updateNeeded, err := buildProfileUpdateBody(ctx, resource, "", current)
	if err != nil {
		return profileFail(resource, err)
	}
	if !updateNeeded {
		return profileProjectSuccess(resource, current, shared.Active, "")
	}

	response, err := c.hooks.Update.Call(ctx, osmanagementhubsdk.UpdateProfileRequest{
		ProfileId:            common.String(currentID),
		UpdateProfileDetails: body.(osmanagementhubsdk.UpdateProfileDetails),
	})
	if err != nil {
		return profileFail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	requestID := profileString(response.OpcRequestId)
	refreshed := response.Profile
	if refreshedID := profileModelID(refreshed); refreshedID != "" {
		if current, found, err := c.getProfile(ctx, refreshedID); err != nil {
			return profileFail(resource, err)
		} else if found {
			refreshed = current
		}
	}
	return profileProjectSuccess(resource, refreshed, shared.Updating, requestID)
}

func (c *profileRuntimeClient) createProfile(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	body, err := buildProfileCreateBody(ctx, resource, req.Namespace)
	if err != nil {
		return profileFail(resource, err)
	}
	response, err := c.hooks.Create.Call(ctx, osmanagementhubsdk.CreateProfileRequest{
		CreateProfileDetails: body.(osmanagementhubsdk.CreateProfileDetails),
		OpcRetryToken:        common.String(profileRetryToken(resource, req.Namespace)),
	})
	if err != nil {
		return profileFail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	requestID := profileString(response.OpcRequestId)
	current := response.Profile
	if currentID := profileModelID(current); currentID != "" {
		if refreshed, found, err := c.getProfile(ctx, currentID); err != nil {
			return profileFail(resource, err)
		} else if found {
			current = refreshed
		}
	}
	return profileProjectSuccess(resource, current, shared.Provisioning, requestID)
}

func (c *profileRuntimeClient) getProfile(ctx context.Context, currentID string) (osmanagementhubsdk.Profile, bool, error) {
	if c.hooks.Get.Call == nil {
		return nil, false, fmt.Errorf("profile get OCI operation is not configured")
	}
	response, err := c.hooks.Get.Call(ctx, osmanagementhubsdk.GetProfileRequest{ProfileId: common.String(currentID)})
	if err != nil {
		err = conservativeProfileNotFoundError(err, "read")
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return nil, false, nil
		}
		return nil, false, err
	}
	return response.Profile, response.Profile != nil, nil
}

func (c *profileRuntimeClient) lookupExistingProfile(ctx context.Context, resource *osmanagementhubv1beta1.Profile) (osmanagementhubsdk.Profile, bool, error) {
	if c.hooks.List.Call == nil {
		return nil, false, nil
	}
	request, ok, err := profileDeleteListRequest(resource)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	response, err := c.hooks.List.Call(ctx, request)
	if err != nil {
		return nil, false, err
	}
	matches, err := c.profileIdentityMatches(ctx, resource, response.Items)
	if err != nil {
		return nil, false, err
	}
	switch len(matches) {
	case 0:
		return nil, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return nil, false, fmt.Errorf("profile list response returned %d matching resources", len(matches))
	}
}

func (c *profileRuntimeClient) profileIdentityMatches(
	ctx context.Context,
	resource *osmanagementhubv1beta1.Profile,
	items []osmanagementhubsdk.ProfileSummary,
) ([]osmanagementhubsdk.Profile, error) {
	desired, err := desiredProfileFields(resource)
	if err != nil {
		return nil, err
	}
	var matches []osmanagementhubsdk.Profile
	for _, item := range items {
		if !profileSummaryMatches(desired, item) {
			continue
		}
		currentID := profileString(item.Id)
		if currentID == "" {
			return nil, fmt.Errorf("profile list response did not expose a resource OCID")
		}
		current, found, err := c.getProfile(ctx, currentID)
		if err != nil {
			return nil, err
		}
		if !found || !profileFullIdentityMatches(desired, current) {
			continue
		}
		matches = append(matches, current)
	}
	return matches, nil
}

func (c *profileRuntimeClient) seedProfileDeleteIDFromList(ctx context.Context, resource *osmanagementhubv1beta1.Profile) (bool, error) {
	existing, found, err := c.lookupExistingProfile(ctx, resource)
	if err != nil {
		return false, handleProfileDeleteError(resource, err)
	}
	if !found {
		return false, nil
	}
	existingID := profileModelID(existing)
	if existingID == "" {
		return false, fmt.Errorf("profile delete list response did not expose a resource OCID")
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	return true, nil
}

func newProfileServiceClientWithOCIClient(client profileOCIClient) ProfileServiceClient {
	hooks := newProfileRuntimeHooksWithOCIClient(client)
	applyProfileRuntimeHooks(&hooks)
	delegate := defaultProfileServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.Profile](
			buildProfileGeneratedRuntimeConfig(&ProfileServiceManager{}, hooks),
		),
	}
	return wrapProfileGeneratedClient(hooks, delegate)
}

func newProfileRuntimeHooksWithOCIClient(client profileOCIClient) ProfileRuntimeHooks {
	return ProfileRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*osmanagementhubv1beta1.Profile]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osmanagementhubv1beta1.Profile]{},
		StatusHooks:     generatedruntime.StatusHooks[*osmanagementhubv1beta1.Profile]{},
		ParityHooks:     generatedruntime.ParityHooks[*osmanagementhubv1beta1.Profile]{},
		Async:           generatedruntime.AsyncHooks[*osmanagementhubv1beta1.Profile]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osmanagementhubv1beta1.Profile]{},
		Create: runtimeOperationHooks[osmanagementhubsdk.CreateProfileRequest, osmanagementhubsdk.CreateProfileResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateProfileDetails", RequestName: "CreateProfileDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.CreateProfileRequest) (osmanagementhubsdk.CreateProfileResponse, error) {
				if client == nil {
					return osmanagementhubsdk.CreateProfileResponse{}, fmt.Errorf("profile OCI client is nil")
				}
				return client.CreateProfile(ctx, request)
			},
		},
		Get: runtimeOperationHooks[osmanagementhubsdk.GetProfileRequest, osmanagementhubsdk.GetProfileResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProfileId", RequestName: "profileId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.GetProfileRequest) (osmanagementhubsdk.GetProfileResponse, error) {
				if client == nil {
					return osmanagementhubsdk.GetProfileResponse{}, fmt.Errorf("profile OCI client is nil")
				}
				return client.GetProfile(ctx, request)
			},
		},
		List: runtimeOperationHooks[osmanagementhubsdk.ListProfilesRequest, osmanagementhubsdk.ListProfilesResponse]{
			Fields: profileListFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error) {
				if client == nil {
					return osmanagementhubsdk.ListProfilesResponse{}, fmt.Errorf("profile OCI client is nil")
				}
				return client.ListProfiles(ctx, request)
			},
		},
		Update: runtimeOperationHooks[osmanagementhubsdk.UpdateProfileRequest, osmanagementhubsdk.UpdateProfileResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ProfileId", RequestName: "profileId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateProfileDetails", RequestName: "UpdateProfileDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request osmanagementhubsdk.UpdateProfileRequest) (osmanagementhubsdk.UpdateProfileResponse, error) {
				if client == nil {
					return osmanagementhubsdk.UpdateProfileResponse{}, fmt.Errorf("profile OCI client is nil")
				}
				return client.UpdateProfile(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[osmanagementhubsdk.DeleteProfileRequest, osmanagementhubsdk.DeleteProfileResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProfileId", RequestName: "profileId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.DeleteProfileRequest) (osmanagementhubsdk.DeleteProfileResponse, error) {
				if client == nil {
					return osmanagementhubsdk.DeleteProfileResponse{}, fmt.Errorf("profile OCI client is nil")
				}
				return client.DeleteProfile(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ProfileServiceClient) ProfileServiceClient{},
	}
}

func reviewedProfileRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "osmanagementhub",
		FormalSlug:    "profile",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(osmanagementhubsdk.ProfileLifecycleStateCreating)},
			UpdatingStates:     []string{string(osmanagementhubsdk.ProfileLifecycleStateUpdating)},
			ActiveStates: []string{
				string(osmanagementhubsdk.ProfileLifecycleStateActive),
				string(osmanagementhubsdk.ProfileLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(osmanagementhubsdk.ProfileLifecycleStateDeleting)},
			TerminalStates: []string{string(osmanagementhubsdk.ProfileLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"profileType",
				"registrationType",
				"isDefaultProfile",
				"managementStationId",
				"vendorName",
				"osFamily",
				"archType",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"isDefaultProfile",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"profileType",
				"managementStationId",
				"registrationType",
				"managedInstanceGroupId",
				"vendorName",
				"osFamily",
				"archType",
				"softwareSourceIds",
				"lifecycleStageId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Profile", Action: "CreateProfile"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Profile", Action: "UpdateProfile"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Profile", Action: "DeleteProfile"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Profile", Action: "GetProfile"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Profile", Action: "GetProfile"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Profile", Action: "GetProfile"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func profileListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "ProfileId", RequestName: "profileId", Contribution: "query", PreferResourceID: true},
		{FieldName: "OsFamily", RequestName: "osFamily", Contribution: "query", LookupPaths: []string{"status.osFamily", "spec.osFamily", "osFamily"}},
		{FieldName: "ArchType", RequestName: "archType", Contribution: "query", LookupPaths: []string{"status.archType", "spec.archType", "archType"}},
		{FieldName: "IsDefaultProfile", RequestName: "isDefaultProfile", Contribution: "query", LookupPaths: []string{"status.isDefaultProfile", "spec.isDefaultProfile", "isDefaultProfile"}},
		{FieldName: "VendorName", RequestName: "vendorName", Contribution: "query", LookupPaths: []string{"status.vendorName", "spec.vendorName", "vendorName"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func listProfilePages(call profileListCall) profileListCall {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request osmanagementhubsdk.ListProfilesRequest) (osmanagementhubsdk.ListProfilesResponse, error) {
		var combined osmanagementhubsdk.ListProfilesResponse
		seenPages := map[string]bool{}
		nextPage := request.Page
		for {
			pageRequest := request
			pageRequest.Page = nextPage
			if err := recordProfileListPage(pageRequest.Page, seenPages); err != nil {
				return combined, err
			}
			response, err := call(ctx, pageRequest)
			if err != nil {
				return combined, err
			}
			mergeProfileListPage(&combined, response)
			if profileString(response.OpcNextPage) == "" {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func recordProfileListPage(pageToken *string, seenPages map[string]bool) error {
	page := profileString(pageToken)
	if page == "" {
		return nil
	}
	if seenPages[page] {
		return fmt.Errorf("profile list pagination repeated page token %q", page)
	}
	seenPages[page] = true
	return nil
}

func mergeProfileListPage(combined *osmanagementhubsdk.ListProfilesResponse, response osmanagementhubsdk.ListProfilesResponse) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	if combined.OpcTotalItems == nil {
		combined.OpcTotalItems = response.OpcTotalItems
	}
	combined.Items = append(combined.Items, response.Items...)
}

func buildProfileCreateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.Profile,
	_ string,
) (any, error) {
	desired, err := desiredProfileFields(resource)
	if err != nil {
		return nil, err
	}
	if err := validateProfileCreateFields(desired); err != nil {
		return nil, err
	}
	return profileCreateBodyFromDesired(desired)
}

func buildProfileUpdateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.Profile,
	_ string,
	currentResponse any,
) (any, bool, error) {
	desired, err := desiredProfileFields(resource)
	if err != nil {
		return osmanagementhubsdk.UpdateProfileDetails{}, false, err
	}
	current, ok := profileModel(currentResponse)
	if !ok {
		return osmanagementhubsdk.UpdateProfileDetails{}, false, fmt.Errorf("profile update requires current OCI readback")
	}

	return profileUpdateBodyFromDesired(desired, current), profileUpdateNeeded(desired, current), nil
}

func profileUpdateBodyFromDesired(
	desired profileDesiredFields,
	current osmanagementhubsdk.Profile,
) osmanagementhubsdk.UpdateProfileDetails {
	body := osmanagementhubsdk.UpdateProfileDetails{}
	setProfileUpdateString(&body.DisplayName, desired.DisplayName, profileString(current.GetDisplayName()), false)
	setProfileUpdateString(&body.Description, desired.Description, profileString(current.GetDescription()), true)
	setProfileUpdateBool(&body.IsDefaultProfile, desired.IsDefaultProfile, desired.HasIsDefaultProfile, profileBool(current.GetIsDefaultProfile()))
	setProfileUpdateStringMap(&body.FreeformTags, desired.FreeformTags, desired.HasFreeformTags, current.GetFreeformTags())
	setProfileUpdateDefinedTags(&body.DefinedTags, desired.DefinedTags, desired.HasDefinedTags, current.GetDefinedTags())
	return body
}

func profileUpdateNeeded(desired profileDesiredFields, current osmanagementhubsdk.Profile) bool {
	if desired.DisplayName != "" && desired.DisplayName != profileString(current.GetDisplayName()) {
		return true
	}
	if desired.Description != profileString(current.GetDescription()) {
		return true
	}
	if desired.HasIsDefaultProfile && desired.IsDefaultProfile != profileBool(current.GetIsDefaultProfile()) {
		return true
	}
	if desired.HasFreeformTags && !maps.Equal(desired.FreeformTags, current.GetFreeformTags()) {
		return true
	}
	return desired.HasDefinedTags && !profileJSONEqual(desired.DefinedTags, current.GetDefinedTags())
}

func setProfileUpdateString(target **string, desired string, current string, allowEmpty bool) {
	if desired == current || (!allowEmpty && desired == "") {
		return
	}
	*target = common.String(desired)
}

func setProfileUpdateBool(target **bool, desired bool, hasDesired bool, current bool) {
	if hasDesired && desired != current {
		*target = common.Bool(desired)
	}
}

func setProfileUpdateStringMap(target *map[string]string, desired map[string]string, hasDesired bool, current map[string]string) {
	if hasDesired && !maps.Equal(desired, current) {
		*target = maps.Clone(desired)
	}
}

func setProfileUpdateDefinedTags(
	target *map[string]map[string]interface{},
	desired map[string]map[string]interface{},
	hasDesired bool,
	current map[string]map[string]interface{},
) {
	if hasDesired && !profileJSONEqual(desired, current) {
		*target = cloneProfileDefinedTags(desired)
	}
}

func desiredProfileFields(resource *osmanagementhubv1beta1.Profile) (profileDesiredFields, error) {
	if resource == nil {
		return profileDesiredFields{}, fmt.Errorf("profile resource is nil")
	}
	desired := profileDesiredFields{}
	if raw := strings.TrimSpace(resource.Spec.JsonData); raw != "" {
		if err := applyProfileJSONFields(&desired, raw); err != nil {
			return profileDesiredFields{}, err
		}
	}
	if err := overlayProfileSpecFields(&desired, resource.Spec); err != nil {
		return profileDesiredFields{}, err
	}
	if !desired.HasIsDefaultProfile {
		desired.IsDefaultProfile = resource.Spec.IsDefaultProfile
		desired.HasIsDefaultProfile = true
	}
	return desired, nil
}

func applyProfileJSONFields(desired *profileDesiredFields, raw string) error {
	payload := profileJSONFields{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Errorf("decode profile spec.jsonData: %w", err)
	}
	applyProfileJSONStringFields(desired, payload)
	applyProfileJSONValueFields(desired, payload)
	return nil
}

func applyProfileJSONStringFields(desired *profileDesiredFields, payload profileJSONFields) {
	setProfileJSONString(&desired.DisplayName, payload.DisplayName)
	setProfileJSONString(&desired.CompartmentID, payload.CompartmentID)
	setProfileJSONString(&desired.Description, payload.Description)
	setProfileJSONString(&desired.ManagementStationID, payload.ManagementStationID)
	setProfileJSONString(&desired.RegistrationType, payload.RegistrationType)
	setProfileJSONString(&desired.ProfileType, payload.ProfileType)
	setProfileJSONString(&desired.ManagedInstanceGroupID, payload.ManagedInstanceGroupID)
	setProfileJSONString(&desired.VendorName, payload.VendorName)
	setProfileJSONString(&desired.OsFamily, payload.OsFamily)
	setProfileJSONString(&desired.ArchType, payload.ArchType)
	setProfileJSONString(&desired.LifecycleStageID, payload.LifecycleStageID)
}

func applyProfileJSONValueFields(desired *profileDesiredFields, payload profileJSONFields) {
	setProfileJSONBool(&desired.IsDefaultProfile, &desired.HasIsDefaultProfile, payload.IsDefaultProfile)
	setProfileJSONFreeformTags(desired, payload.FreeformTags)
	setProfileJSONDefinedTags(desired, payload.DefinedTags)
	if payload.SoftwareSourceIDs != nil {
		desired.SoftwareSourceIDs = profileStringSlice(payload.SoftwareSourceIDs)
	}
}

func setProfileJSONString(target *string, value *string) {
	if value != nil {
		*target = strings.TrimSpace(*value)
	}
}

func setProfileJSONBool(target *bool, hasTarget *bool, value *bool) {
	if value != nil {
		*target = *value
		*hasTarget = true
	}
}

func setProfileJSONFreeformTags(desired *profileDesiredFields, tags map[string]string) {
	if tags != nil {
		desired.FreeformTags = maps.Clone(tags)
		desired.HasFreeformTags = true
	}
}

func setProfileJSONDefinedTags(desired *profileDesiredFields, tags map[string]map[string]interface{}) {
	if tags != nil {
		desired.DefinedTags = cloneProfileDefinedTags(tags)
		desired.HasDefinedTags = true
	}
}

func overlayProfileSpecFields(desired *profileDesiredFields, spec osmanagementhubv1beta1.ProfileSpec) error {
	if err := overlayProfileSpecStrings(desired, spec); err != nil {
		return err
	}
	if err := overlayProfileSpecDefaultProfile(desired, spec); err != nil {
		return err
	}
	overlayProfileSpecTags(desired, spec)
	return overlayProfileSpecSoftwareSources(desired, spec)
}

func overlayProfileSpecStrings(desired *profileDesiredFields, spec osmanagementhubv1beta1.ProfileSpec) error {
	fields := []struct {
		target *string
		value  string
		name   string
	}{
		{target: &desired.DisplayName, value: spec.DisplayName, name: "displayName"},
		{target: &desired.CompartmentID, value: spec.CompartmentId, name: "compartmentId"},
		{target: &desired.Description, value: spec.Description, name: "description"},
		{target: &desired.ManagementStationID, value: spec.ManagementStationId, name: "managementStationId"},
		{target: &desired.RegistrationType, value: spec.RegistrationType, name: "registrationType"},
		{target: &desired.ProfileType, value: spec.ProfileType, name: "profileType"},
		{target: &desired.ManagedInstanceGroupID, value: spec.ManagedInstanceGroupId, name: "managedInstanceGroupId"},
		{target: &desired.VendorName, value: spec.VendorName, name: "vendorName"},
		{target: &desired.OsFamily, value: spec.OsFamily, name: "osFamily"},
		{target: &desired.ArchType, value: spec.ArchType, name: "archType"},
		{target: &desired.LifecycleStageID, value: spec.LifecycleStageId, name: "lifecycleStageId"},
	}
	for _, field := range fields {
		if err := overlayProfileString(field.target, field.value, field.name); err != nil {
			return err
		}
	}
	return nil
}

func overlayProfileSpecDefaultProfile(desired *profileDesiredFields, spec osmanagementhubv1beta1.ProfileSpec) error {
	if !spec.IsDefaultProfile {
		return nil
	}
	if desired.HasIsDefaultProfile && !desired.IsDefaultProfile {
		return fmt.Errorf("profile spec.jsonData identity conflicts with spec field isDefaultProfile")
	}
	desired.IsDefaultProfile = true
	desired.HasIsDefaultProfile = true
	return nil
}

func overlayProfileSpecTags(desired *profileDesiredFields, spec osmanagementhubv1beta1.ProfileSpec) {
	if spec.FreeformTags != nil {
		desired.FreeformTags = maps.Clone(spec.FreeformTags)
		desired.HasFreeformTags = true
	}
	if spec.DefinedTags != nil {
		desired.DefinedTags = profileDefinedTagsFromSpec(spec.DefinedTags)
		desired.HasDefinedTags = true
	}
}

func overlayProfileSpecSoftwareSources(desired *profileDesiredFields, spec osmanagementhubv1beta1.ProfileSpec) error {
	if spec.SoftwareSourceIds == nil {
		return nil
	}
	candidate := profileStringSlice(spec.SoftwareSourceIds)
	if len(desired.SoftwareSourceIDs) != 0 && !reflect.DeepEqual(profileSortedStrings(desired.SoftwareSourceIDs), profileSortedStrings(candidate)) {
		return fmt.Errorf("profile spec.jsonData identity conflicts with spec field softwareSourceIds")
	}
	desired.SoftwareSourceIDs = candidate
	return nil
}

func overlayProfileString(target *string, value string, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.TrimSpace(*target) != "" && strings.TrimSpace(*target) != value {
		return fmt.Errorf("profile spec.jsonData identity conflicts with spec field %s", field)
	}
	*target = value
	return nil
}

func validateProfileCreateFields(desired profileDesiredFields) error {
	if err := validateProfileRequiredFields(desired); err != nil {
		return err
	}
	profileType, err := profileType(desired.ProfileType)
	if err != nil {
		return err
	}
	if err := validateProfileRegistrationFields(desired); err != nil {
		return err
	}
	return validateProfileTypeFields(desired, profileType)
}

func validateProfileRequiredFields(desired profileDesiredFields) error {
	if desired.DisplayName == "" {
		return fmt.Errorf("profile create requires displayName")
	}
	if desired.CompartmentID == "" {
		return fmt.Errorf("profile create requires compartmentId")
	}
	return nil
}

func validateProfileRegistrationFields(desired profileDesiredFields) error {
	if desired.RegistrationType == string(osmanagementhubsdk.ProfileRegistrationTypeNonOciLinux) && desired.ManagementStationID == "" {
		return fmt.Errorf("profile create requires managementStationId for NON_OCI_LINUX registrationType")
	}
	return nil
}

func validateProfileTypeFields(desired profileDesiredFields, profileType osmanagementhubsdk.ProfileTypeEnum) error {
	switch profileType {
	case osmanagementhubsdk.ProfileTypeGroup:
		if desired.ManagedInstanceGroupID == "" {
			return fmt.Errorf("profile GROUP create requires managedInstanceGroupId")
		}
	case osmanagementhubsdk.ProfileTypeLifecycle:
		if desired.LifecycleStageID == "" {
			return fmt.Errorf("profile LIFECYCLE create requires lifecycleStageId")
		}
	case osmanagementhubsdk.ProfileTypeSoftwaresource, osmanagementhubsdk.ProfileTypeWindowsStandalone:
		if desired.VendorName == "" || desired.OsFamily == "" || desired.ArchType == "" {
			return fmt.Errorf("profile %s create requires vendorName, osFamily, and archType", profileType)
		}
	case osmanagementhubsdk.ProfileTypeStation:
	default:
		return fmt.Errorf("unsupported Profile profileType %q", desired.ProfileType)
	}
	return nil
}

func profileCreateBodyFromDesired(desired profileDesiredFields) (osmanagementhubsdk.CreateProfileDetails, error) {
	registrationType, err := profileRegistrationType(desired.RegistrationType)
	if err != nil {
		return nil, err
	}
	profileType, err := profileType(desired.ProfileType)
	if err != nil {
		return nil, err
	}
	commonFields := profileCommonCreateFields(desired, registrationType)
	switch profileType {
	case osmanagementhubsdk.ProfileTypeGroup:
		return profileGroupCreateBody(desired, commonFields), nil
	case osmanagementhubsdk.ProfileTypeLifecycle:
		return profileLifecycleCreateBody(desired, commonFields), nil
	case osmanagementhubsdk.ProfileTypeSoftwaresource:
		return profileSoftwareSourceCreateBody(desired, commonFields)
	case osmanagementhubsdk.ProfileTypeStation:
		return profileStationCreateBody(desired, commonFields)
	case osmanagementhubsdk.ProfileTypeWindowsStandalone:
		return profileWindowsStandaloneCreateBody(desired, commonFields)
	default:
		return nil, fmt.Errorf("unsupported Profile profileType %q", desired.ProfileType)
	}
}

func profileGroupCreateBody(
	desired profileDesiredFields,
	commonFields profileCommonCreate,
) osmanagementhubsdk.CreateGroupProfileDetails {
	return osmanagementhubsdk.CreateGroupProfileDetails{
		DisplayName:            commonFields.DisplayName,
		CompartmentId:          commonFields.CompartmentId,
		ManagedInstanceGroupId: common.String(desired.ManagedInstanceGroupID),
		Description:            commonFields.Description,
		ManagementStationId:    commonFields.ManagementStationId,
		IsDefaultProfile:       commonFields.IsDefaultProfile,
		FreeformTags:           commonFields.FreeformTags,
		DefinedTags:            commonFields.DefinedTags,
		RegistrationType:       commonFields.RegistrationType,
	}
}

func profileLifecycleCreateBody(
	desired profileDesiredFields,
	commonFields profileCommonCreate,
) osmanagementhubsdk.CreateLifecycleProfileDetails {
	return osmanagementhubsdk.CreateLifecycleProfileDetails{
		DisplayName:         commonFields.DisplayName,
		CompartmentId:       commonFields.CompartmentId,
		LifecycleStageId:    common.String(desired.LifecycleStageID),
		Description:         commonFields.Description,
		ManagementStationId: commonFields.ManagementStationId,
		IsDefaultProfile:    commonFields.IsDefaultProfile,
		FreeformTags:        commonFields.FreeformTags,
		DefinedTags:         commonFields.DefinedTags,
		RegistrationType:    commonFields.RegistrationType,
	}
}

func profileSoftwareSourceCreateBody(
	desired profileDesiredFields,
	commonFields profileCommonCreate,
) (osmanagementhubsdk.CreateProfileDetails, error) {
	vendorName, osFamily, archType, err := profilePlatformEnums(desired)
	if err != nil {
		return nil, err
	}
	return osmanagementhubsdk.CreateSoftwareSourceProfileDetails{
		DisplayName:         commonFields.DisplayName,
		CompartmentId:       commonFields.CompartmentId,
		Description:         commonFields.Description,
		ManagementStationId: commonFields.ManagementStationId,
		IsDefaultProfile:    commonFields.IsDefaultProfile,
		FreeformTags:        commonFields.FreeformTags,
		DefinedTags:         commonFields.DefinedTags,
		SoftwareSourceIds:   profileStringSlice(desired.SoftwareSourceIDs),
		RegistrationType:    commonFields.RegistrationType,
		VendorName:          vendorName,
		OsFamily:            osFamily,
		ArchType:            archType,
	}, nil
}

func profileStationCreateBody(
	desired profileDesiredFields,
	commonFields profileCommonCreate,
) (osmanagementhubsdk.CreateProfileDetails, error) {
	vendorName, osFamily, archType, err := profileOptionalPlatformEnums(desired)
	if err != nil {
		return nil, err
	}
	return osmanagementhubsdk.CreateStationProfileDetails{
		DisplayName:         commonFields.DisplayName,
		CompartmentId:       commonFields.CompartmentId,
		Description:         commonFields.Description,
		ManagementStationId: commonFields.ManagementStationId,
		IsDefaultProfile:    commonFields.IsDefaultProfile,
		FreeformTags:        commonFields.FreeformTags,
		DefinedTags:         commonFields.DefinedTags,
		RegistrationType:    commonFields.RegistrationType,
		VendorName:          vendorName,
		OsFamily:            osFamily,
		ArchType:            archType,
	}, nil
}

func profileWindowsStandaloneCreateBody(
	desired profileDesiredFields,
	commonFields profileCommonCreate,
) (osmanagementhubsdk.CreateProfileDetails, error) {
	vendorName, osFamily, archType, err := profilePlatformEnums(desired)
	if err != nil {
		return nil, err
	}
	return osmanagementhubsdk.CreateWindowsStandAloneProfileDetails{
		DisplayName:         commonFields.DisplayName,
		CompartmentId:       commonFields.CompartmentId,
		Description:         commonFields.Description,
		ManagementStationId: commonFields.ManagementStationId,
		IsDefaultProfile:    commonFields.IsDefaultProfile,
		FreeformTags:        commonFields.FreeformTags,
		DefinedTags:         commonFields.DefinedTags,
		RegistrationType:    commonFields.RegistrationType,
		VendorName:          vendorName,
		OsFamily:            osFamily,
		ArchType:            archType,
	}, nil
}

type profileCommonCreate struct {
	DisplayName         *string
	CompartmentId       *string
	Description         *string
	ManagementStationId *string
	IsDefaultProfile    *bool
	FreeformTags        map[string]string
	DefinedTags         map[string]map[string]interface{}
	RegistrationType    osmanagementhubsdk.ProfileRegistrationTypeEnum
}

func profileCommonCreateFields(
	desired profileDesiredFields,
	registrationType osmanagementhubsdk.ProfileRegistrationTypeEnum,
) profileCommonCreate {
	fields := profileCommonCreate{
		DisplayName:      common.String(desired.DisplayName),
		CompartmentId:    common.String(desired.CompartmentID),
		IsDefaultProfile: common.Bool(desired.IsDefaultProfile),
		RegistrationType: registrationType,
	}
	if desired.Description != "" {
		fields.Description = common.String(desired.Description)
	}
	if desired.ManagementStationID != "" {
		fields.ManagementStationId = common.String(desired.ManagementStationID)
	}
	if desired.HasFreeformTags {
		fields.FreeformTags = maps.Clone(desired.FreeformTags)
	}
	if desired.HasDefinedTags {
		fields.DefinedTags = cloneProfileDefinedTags(desired.DefinedTags)
	}
	return fields
}

func validateProfileCreateOnlyDrift(resource *osmanagementhubv1beta1.Profile, currentResponse any) error {
	desired, err := desiredProfileFields(resource)
	if err != nil {
		return err
	}
	current, ok := profileModel(currentResponse)
	if !ok {
		return nil
	}
	var drift []string
	drift = appendProfileDrift(drift, "compartmentId", desired.CompartmentID, profileString(current.GetCompartmentId()))
	drift = appendProfileDrift(drift, "profileType", desired.ProfileType, profileModelType(current))
	drift = appendProfileDrift(drift, "managementStationId", desired.ManagementStationID, profileString(current.GetManagementStationId()))
	drift = appendProfileDrift(drift, "registrationType", desired.RegistrationType, string(current.GetRegistrationType()))
	drift = appendProfileDrift(drift, "managedInstanceGroupId", desired.ManagedInstanceGroupID, profileManagedInstanceGroupID(current))
	drift = appendProfileDrift(drift, "vendorName", desired.VendorName, string(current.GetVendorName()))
	drift = appendProfileDrift(drift, "osFamily", desired.OsFamily, string(current.GetOsFamily()))
	drift = appendProfileDrift(drift, "archType", desired.ArchType, string(current.GetArchType()))
	drift = appendProfileDrift(drift, "lifecycleStageId", desired.LifecycleStageID, profileLifecycleStageID(current))
	if !reflect.DeepEqual(profileSortedStrings(desired.SoftwareSourceIDs), profileSortedStrings(profileSoftwareSourceIDs(current))) {
		drift = append(drift, "softwareSourceIds")
	}
	if len(drift) == 0 {
		return nil
	}
	sort.Strings(drift)
	return fmt.Errorf("profile create-only field drift detected: %s", strings.Join(drift, ", "))
}

func appendProfileDrift(drift []string, field string, desired string, current string) []string {
	desired = strings.TrimSpace(desired)
	current = strings.TrimSpace(current)
	if desired == "" && current == "" {
		return drift
	}
	if !strings.EqualFold(desired, current) {
		return append(drift, field)
	}
	return drift
}

func profileDeleteConfirmRead(
	getProfile profileGetCall,
	listProfiles profileListCall,
) func(context.Context, *osmanagementhubv1beta1.Profile, string) (any, error) {
	return func(ctx context.Context, resource *osmanagementhubv1beta1.Profile, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID != "" {
			if getProfile == nil {
				return nil, fmt.Errorf("profile delete confirmation requires get OCI operation")
			}
			response, err := getProfile(ctx, osmanagementhubsdk.GetProfileRequest{ProfileId: common.String(currentID)})
			return profileDeleteConfirmReadResponse(response, err)
		}
		if listProfiles == nil {
			return nil, fmt.Errorf("profile delete confirmation requires list OCI operation when no profile OCID is tracked")
		}
		request, ok, err := profileDeleteListRequest(resource)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("profile delete confirmation requires compartmentId and displayName when no profile OCID is tracked")
		}
		response, err := listProfiles(ctx, request)
		if err != nil {
			return profileDeleteConfirmReadResponse(response, err)
		}
		matches := profileMatchingSummaries(resource, response.Items)
		switch len(matches) {
		case 0:
			return nil, errorutil.NotFoundOciError{
				HTTPStatusCode: 404,
				ErrorCode:      errorutil.NotFound,
				Description:    "Profile delete confirmation did not find a matching OCI profile",
			}
		case 1:
			return matches[0], nil
		default:
			return nil, fmt.Errorf("profile delete confirmation found %d matching OCI profiles", len(matches))
		}
	}
}

func profileDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	converted := conservativeProfileNotFoundError(err, "delete confirmation")
	var ambiguous profileAmbiguousNotFoundError
	if errors.As(converted, &ambiguous) {
		return profileAmbiguousNotFoundError{
			message:      fmt.Sprintf("profile delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound: %s", err.Error()),
			opcRequestID: ambiguous.opcRequestID,
		}, nil
	}
	return nil, converted
}

func profileDeleteListRequest(resource *osmanagementhubv1beta1.Profile) (osmanagementhubsdk.ListProfilesRequest, bool, error) {
	desired, err := desiredProfileFields(resource)
	if err != nil {
		return osmanagementhubsdk.ListProfilesRequest{}, false, err
	}
	request := osmanagementhubsdk.ListProfilesRequest{}
	if desired.CompartmentID != "" {
		request.CompartmentId = common.String(desired.CompartmentID)
	}
	if desired.DisplayName != "" {
		request.DisplayName = []string{desired.DisplayName}
	}
	if desired.ProfileType != "" {
		profileType, err := profileType(desired.ProfileType)
		if err != nil {
			return osmanagementhubsdk.ListProfilesRequest{}, false, err
		}
		request.ProfileType = []osmanagementhubsdk.ProfileTypeEnum{profileType}
	}
	return request, request.CompartmentId != nil && len(request.DisplayName) != 0, nil
}

func profileMatchingSummaries(resource *osmanagementhubv1beta1.Profile, items []osmanagementhubsdk.ProfileSummary) []osmanagementhubsdk.ProfileSummary {
	desired, err := desiredProfileFields(resource)
	if err != nil {
		return nil
	}
	var matches []osmanagementhubsdk.ProfileSummary
	for _, item := range items {
		if profileSummaryMatches(desired, item) {
			matches = append(matches, item)
		}
	}
	return matches
}

func profileSummaryMatches(desired profileDesiredFields, item osmanagementhubsdk.ProfileSummary) bool {
	if item.LifecycleState == osmanagementhubsdk.ProfileLifecycleStateDeleted {
		return false
	}
	return profileSummaryFieldsMatch([]profileSummaryMatchField{
		{desired: desired.CompartmentID, current: profileString(item.CompartmentId)},
		{desired: desired.DisplayName, current: profileString(item.DisplayName)},
		{desired: desired.ProfileType, current: string(item.ProfileType), equalFold: true},
		{desired: desired.ManagementStationID, current: profileString(item.ManagementStationId)},
		{desired: desired.RegistrationType, current: string(item.RegistrationType), equalFold: true},
		{desired: desired.VendorName, current: string(item.VendorName), equalFold: true},
		{desired: desired.OsFamily, current: string(item.OsFamily), equalFold: true},
		{desired: desired.ArchType, current: string(item.ArchType), equalFold: true},
	})
}

func profileFullIdentityMatches(desired profileDesiredFields, current osmanagementhubsdk.Profile) bool {
	if current == nil || profileLifecycleDeleted(current.GetLifecycleState()) {
		return false
	}
	if !profileSummaryFieldsMatch([]profileSummaryMatchField{
		{desired: desired.CompartmentID, current: profileString(current.GetCompartmentId())},
		{desired: desired.DisplayName, current: profileString(current.GetDisplayName())},
		{desired: desired.ProfileType, current: profileModelType(current), equalFold: true},
		{desired: desired.ManagementStationID, current: profileString(current.GetManagementStationId())},
		{desired: desired.RegistrationType, current: string(current.GetRegistrationType()), equalFold: true},
		{desired: desired.ManagedInstanceGroupID, current: profileManagedInstanceGroupID(current)},
		{desired: desired.VendorName, current: string(current.GetVendorName()), equalFold: true},
		{desired: desired.OsFamily, current: string(current.GetOsFamily()), equalFold: true},
		{desired: desired.ArchType, current: string(current.GetArchType()), equalFold: true},
		{desired: desired.LifecycleStageID, current: profileLifecycleStageID(current)},
	}) {
		return false
	}
	if desired.HasIsDefaultProfile && desired.IsDefaultProfile != profileBool(current.GetIsDefaultProfile()) {
		return false
	}
	return reflect.DeepEqual(profileSortedStrings(desired.SoftwareSourceIDs), profileSortedStrings(profileSoftwareSourceIDs(current)))
}

type profileSummaryMatchField struct {
	desired   string
	current   string
	equalFold bool
}

func profileSummaryFieldsMatch(fields []profileSummaryMatchField) bool {
	for _, field := range fields {
		if !profileSummaryFieldMatches(field) {
			return false
		}
	}
	return true
}

func profileSummaryFieldMatches(field profileSummaryMatchField) bool {
	desired := strings.TrimSpace(field.desired)
	if desired == "" {
		return true
	}
	current := strings.TrimSpace(field.current)
	if field.equalFold {
		return strings.EqualFold(current, desired)
	}
	return current == desired
}

func applyProfileDeleteOutcome(
	resource *osmanagementhubv1beta1.Profile,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	var ambiguous profileAmbiguousNotFoundError
	if errors.As(profileResponseAsError(response), &ambiguous) {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && profileLifecycleWritePending(response) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func profileResponseAsError(response any) error {
	if typed, ok := response.(error); ok {
		return typed
	}
	return nil
}

func profileLifecycleWritePending(response any) bool {
	current, ok := profileModel(response)
	if !ok {
		return false
	}
	switch current.GetLifecycleState() {
	case osmanagementhubsdk.ProfileLifecycleStateCreating, osmanagementhubsdk.ProfileLifecycleStateUpdating:
		return true
	default:
		return false
	}
}

func handleProfileDeleteError(resource *osmanagementhubv1beta1.Profile, err error) error {
	normalized := conservativeProfileNotFoundError(err, "delete")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, normalized)
	}
	return normalized
}

func conservativeProfileNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("profile %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	if requestID := errorutil.OpcRequestID(err); requestID != "" {
		return profileAmbiguousNotFoundError{message: message, opcRequestID: requestID}
	}
	return profileAmbiguousNotFoundError{message: message}
}

func profileProjectSuccess(
	resource *osmanagementhubv1beta1.Profile,
	current osmanagementhubsdk.Profile,
	fallback shared.OSOKConditionType,
	requestID string,
) (servicemanager.OSOKResponse, error) {
	if current != nil {
		profileProjectModel(resource, current)
	}
	status := &resource.Status.OsokStatus
	profileProjectIdentityStatus(resource, current)
	profileProjectRequestStatus(status, requestID)
	profileProjectTimestamps(status)
	lifecycleState := profileLifecycleState(current)
	condition := profileSuccessCondition(lifecycleState, fallback)
	message := profileConditionMessage(lifecycleState, condition)
	if async := profileLifecycleAsyncOperation(status, lifecycleState, fallback, message); async != nil {
		projection := servicemanager.ApplyAsyncOperation(status, async, loggerutil.OSOKLogger{})
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: profileRequeueDuration,
		}, nil
	}
	servicemanager.ClearAsyncOperation(status)
	status.Message = message
	status.Reason = string(condition)
	conditionStatus := profileConditionStatus(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating,
		RequeueDuration: profileRequeueDuration,
	}, nil
}

func profileProjectIdentityStatus(resource *osmanagementhubv1beta1.Profile, current osmanagementhubsdk.Profile) {
	if currentID := profileModelID(current); currentID != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
		resource.Status.Id = currentID
	}
}

func profileProjectRequestStatus(status *shared.OSOKStatus, requestID string) {
	if requestID != "" {
		status.OpcRequestID = requestID
	}
}

func profileProjectTimestamps(status *shared.OSOKStatus) {
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
}

func profileLifecycleState(current osmanagementhubsdk.Profile) osmanagementhubsdk.ProfileLifecycleStateEnum {
	if current == nil {
		return ""
	}
	return current.GetLifecycleState()
}

func profileSuccessCondition(
	lifecycleState osmanagementhubsdk.ProfileLifecycleStateEnum,
	fallback shared.OSOKConditionType,
) shared.OSOKConditionType {
	condition := profileConditionForLifecycle(lifecycleState)
	if condition == "" {
		return fallback
	}
	return condition
}

func profileConditionStatus(condition shared.OSOKConditionType) corev1.ConditionStatus {
	if condition == shared.Failed {
		return corev1.ConditionFalse
	}
	return corev1.ConditionTrue
}

func profileLifecycleAsyncOperation(
	status *shared.OSOKStatus,
	state osmanagementhubsdk.ProfileLifecycleStateEnum,
	fallback shared.OSOKConditionType,
	message string,
) *shared.OSOKAsyncOperation {
	phase := profileLifecycleAsyncPhase(status, state, fallback)
	class := profileLifecycleAsyncClass(state)
	if phase == "" || class == "" {
		return nil
	}
	return &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       string(state),
		NormalizedClass: class,
		Message:         message,
	}
}

func profileLifecycleAsyncPhase(
	status *shared.OSOKStatus,
	state osmanagementhubsdk.ProfileLifecycleStateEnum,
	fallback shared.OSOKConditionType,
) shared.OSOKAsyncPhase {
	switch state {
	case osmanagementhubsdk.ProfileLifecycleStateCreating:
		return shared.OSOKAsyncPhaseCreate
	case osmanagementhubsdk.ProfileLifecycleStateUpdating:
		return shared.OSOKAsyncPhaseUpdate
	case osmanagementhubsdk.ProfileLifecycleStateDeleting:
		return shared.OSOKAsyncPhaseDelete
	case osmanagementhubsdk.ProfileLifecycleStateFailed:
		return profileFailedLifecycleAsyncPhase(status, fallback)
	default:
		return ""
	}
}

func profileFailedLifecycleAsyncPhase(status *shared.OSOKStatus, condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	if status != nil && status.Async.Current != nil && status.Async.Current.Phase != "" {
		return status.Async.Current.Phase
	}
	switch condition {
	case shared.Updating:
		return shared.OSOKAsyncPhaseUpdate
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return shared.OSOKAsyncPhaseCreate
	}
}

func profileLifecycleAsyncClass(state osmanagementhubsdk.ProfileLifecycleStateEnum) shared.OSOKAsyncNormalizedClass {
	switch state {
	case osmanagementhubsdk.ProfileLifecycleStateCreating,
		osmanagementhubsdk.ProfileLifecycleStateUpdating,
		osmanagementhubsdk.ProfileLifecycleStateDeleting:
		return shared.OSOKAsyncClassPending
	case osmanagementhubsdk.ProfileLifecycleStateFailed:
		return shared.OSOKAsyncClassFailed
	default:
		return ""
	}
}

func profileFail(resource *osmanagementhubv1beta1.Profile, err error) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func profileProjectModel(resource *osmanagementhubv1beta1.Profile, current osmanagementhubsdk.Profile) {
	if resource == nil || current == nil {
		return
	}
	resource.Status.Id = profileString(current.GetId())
	resource.Status.CompartmentId = profileString(current.GetCompartmentId())
	resource.Status.DisplayName = profileString(current.GetDisplayName())
	resource.Status.Description = profileString(current.GetDescription())
	resource.Status.ManagementStationId = profileString(current.GetManagementStationId())
	resource.Status.TimeCreated = profileTimeString(current.GetTimeCreated())
	resource.Status.TimeModified = profileTimeString(current.GetTimeModified())
	resource.Status.ProfileVersion = profileString(current.GetProfileVersion())
	resource.Status.LifecycleState = string(current.GetLifecycleState())
	resource.Status.RegistrationType = string(current.GetRegistrationType())
	resource.Status.IsDefaultProfile = profileBool(current.GetIsDefaultProfile())
	resource.Status.IsServiceProvidedProfile = profileBool(current.GetIsServiceProvidedProfile())
	resource.Status.FreeformTags = maps.Clone(current.GetFreeformTags())
	resource.Status.DefinedTags = profileStatusDefinedTags(current.GetDefinedTags())
	resource.Status.SystemTags = profileStatusDefinedTags(current.GetSystemTags())
	resource.Status.VendorName = string(current.GetVendorName())
	resource.Status.OsFamily = string(current.GetOsFamily())
	resource.Status.ArchType = string(current.GetArchType())
	resource.Status.ProfileType = profileModelType(current)
	resource.Status.ManagedInstanceGroup = osmanagementhubv1beta1.ProfileManagedInstanceGroup{}
	resource.Status.LifecycleStage = osmanagementhubv1beta1.ProfileLifecycleStage{}
	resource.Status.LifecycleEnvironment = osmanagementhubv1beta1.ProfileLifecycleEnvironment{}
	resource.Status.SoftwareSources = nil
	switch typed := current.(type) {
	case osmanagementhubsdk.GroupProfile:
		if typed.ManagedInstanceGroup != nil {
			resource.Status.ManagedInstanceGroup = osmanagementhubv1beta1.ProfileManagedInstanceGroup{
				Id:          profileString(typed.ManagedInstanceGroup.Id),
				DisplayName: profileString(typed.ManagedInstanceGroup.DisplayName),
			}
		}
	case osmanagementhubsdk.LifecycleProfile:
		if typed.LifecycleStage != nil {
			resource.Status.LifecycleStage = osmanagementhubv1beta1.ProfileLifecycleStage{
				Id:          profileString(typed.LifecycleStage.Id),
				DisplayName: profileString(typed.LifecycleStage.DisplayName),
			}
		}
		if typed.LifecycleEnvironment != nil {
			resource.Status.LifecycleEnvironment = osmanagementhubv1beta1.ProfileLifecycleEnvironment{
				Id:          profileString(typed.LifecycleEnvironment.Id),
				DisplayName: profileString(typed.LifecycleEnvironment.DisplayName),
			}
		}
	case osmanagementhubsdk.SoftwareSourceProfile:
		resource.Status.SoftwareSources = profileStatusSoftwareSources(typed.SoftwareSources)
	}
	if payload, err := json.Marshal(current); err == nil {
		resource.Status.JsonData = string(payload)
	}
}

func profileConditionForLifecycle(state osmanagementhubsdk.ProfileLifecycleStateEnum) shared.OSOKConditionType {
	switch state {
	case osmanagementhubsdk.ProfileLifecycleStateCreating:
		return shared.Provisioning
	case osmanagementhubsdk.ProfileLifecycleStateUpdating:
		return shared.Updating
	case osmanagementhubsdk.ProfileLifecycleStateDeleting:
		return shared.Terminating
	case osmanagementhubsdk.ProfileLifecycleStateFailed:
		return shared.Failed
	case osmanagementhubsdk.ProfileLifecycleStateActive, osmanagementhubsdk.ProfileLifecycleStateInactive:
		return shared.Active
	case "":
		return ""
	default:
		return shared.Updating
	}
}

func profileConditionMessage(state osmanagementhubsdk.ProfileLifecycleStateEnum, condition shared.OSOKConditionType) string {
	if state != "" {
		return fmt.Sprintf("OCI Profile lifecycle state is %s", state)
	}
	return fmt.Sprintf("OCI Profile reconciled as %s", condition)
}

func profileLifecycleInProgress(state osmanagementhubsdk.ProfileLifecycleStateEnum) bool {
	switch state {
	case osmanagementhubsdk.ProfileLifecycleStateCreating,
		osmanagementhubsdk.ProfileLifecycleStateUpdating,
		osmanagementhubsdk.ProfileLifecycleStateDeleting:
		return true
	default:
		return false
	}
}

func profileLifecycleDeleted(state osmanagementhubsdk.ProfileLifecycleStateEnum) bool {
	return state == osmanagementhubsdk.ProfileLifecycleStateDeleted
}

func profileRetryToken(resource *osmanagementhubv1beta1.Profile, namespace string) string {
	if resource == nil {
		return profileRetryTokenFromParts("profile", "namespace-name", strings.TrimSpace(namespace))
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return profileRetryTokenFromParts("profile", "uid", uid)
	}
	resourceNamespace := strings.TrimSpace(resource.Namespace)
	if resourceNamespace == "" {
		resourceNamespace = strings.TrimSpace(namespace)
	}
	parts := []string{"profile", "namespace-name", resourceNamespace, strings.TrimSpace(resource.Name)}
	return profileRetryTokenFromParts(parts...)
}

func profileRetryTokenFromParts(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "/")))
	return hex.EncodeToString(sum[:])
}

func profileModel(response any) (osmanagementhubsdk.Profile, bool) {
	switch typed := response.(type) {
	case osmanagementhubsdk.CreateProfileResponse:
		return typed.Profile, typed.Profile != nil
	case osmanagementhubsdk.GetProfileResponse:
		return typed.Profile, typed.Profile != nil
	case osmanagementhubsdk.UpdateProfileResponse:
		return typed.Profile, typed.Profile != nil
	case osmanagementhubsdk.Profile:
		return typed, typed != nil
	case osmanagementhubsdk.ProfileSummary:
		return profileFromSummary(typed), true
	default:
		return nil, false
	}
}

func profileModelID(current osmanagementhubsdk.Profile) string {
	if current == nil {
		return ""
	}
	return profileString(current.GetId())
}

func profileFromSummary(summary osmanagementhubsdk.ProfileSummary) osmanagementhubsdk.Profile {
	return osmanagementhubsdk.StationProfile{
		Id:                       summary.Id,
		CompartmentId:            summary.CompartmentId,
		DisplayName:              summary.DisplayName,
		Description:              summary.Description,
		ManagementStationId:      summary.ManagementStationId,
		TimeCreated:              summary.TimeCreated,
		LifecycleState:           summary.LifecycleState,
		RegistrationType:         summary.RegistrationType,
		IsDefaultProfile:         summary.IsDefaultProfile,
		IsServiceProvidedProfile: summary.IsServiceProvidedProfile,
		FreeformTags:             summary.FreeformTags,
		DefinedTags:              summary.DefinedTags,
		SystemTags:               summary.SystemTags,
		VendorName:               summary.VendorName,
		OsFamily:                 summary.OsFamily,
		ArchType:                 summary.ArchType,
	}
}

func profileModelType(current osmanagementhubsdk.Profile) string {
	switch current.(type) {
	case osmanagementhubsdk.GroupProfile:
		return string(osmanagementhubsdk.ProfileTypeGroup)
	case osmanagementhubsdk.LifecycleProfile:
		return string(osmanagementhubsdk.ProfileTypeLifecycle)
	case osmanagementhubsdk.SoftwareSourceProfile:
		return string(osmanagementhubsdk.ProfileTypeSoftwaresource)
	case osmanagementhubsdk.StationProfile:
		return string(osmanagementhubsdk.ProfileTypeStation)
	case osmanagementhubsdk.WindowsStandaloneProfile:
		return string(osmanagementhubsdk.ProfileTypeWindowsStandalone)
	default:
		return ""
	}
}

func profileManagedInstanceGroupID(current osmanagementhubsdk.Profile) string {
	group, ok := current.(osmanagementhubsdk.GroupProfile)
	if !ok || group.ManagedInstanceGroup == nil {
		return ""
	}
	return profileString(group.ManagedInstanceGroup.Id)
}

func profileLifecycleStageID(current osmanagementhubsdk.Profile) string {
	lifecycle, ok := current.(osmanagementhubsdk.LifecycleProfile)
	if !ok || lifecycle.LifecycleStage == nil {
		return ""
	}
	return profileString(lifecycle.LifecycleStage.Id)
}

func profileSoftwareSourceIDs(current osmanagementhubsdk.Profile) []string {
	softwareSource, ok := current.(osmanagementhubsdk.SoftwareSourceProfile)
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(softwareSource.SoftwareSources))
	for _, source := range softwareSource.SoftwareSources {
		if id := strings.TrimSpace(profileString(source.Id)); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func profileType(value string) (osmanagementhubsdk.ProfileTypeEnum, error) {
	if profileType, ok := osmanagementhubsdk.GetMappingProfileTypeEnum(strings.TrimSpace(value)); ok {
		return profileType, nil
	}
	return "", fmt.Errorf("unsupported Profile profileType %q", value)
}

func profileRegistrationType(value string) (osmanagementhubsdk.ProfileRegistrationTypeEnum, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	if registrationType, ok := osmanagementhubsdk.GetMappingProfileRegistrationTypeEnum(value); ok {
		return registrationType, nil
	}
	return "", fmt.Errorf("unsupported Profile registrationType %q", value)
}

func profilePlatformEnums(desired profileDesiredFields) (osmanagementhubsdk.VendorNameEnum, osmanagementhubsdk.OsFamilyEnum, osmanagementhubsdk.ArchTypeEnum, error) {
	vendorName, ok := osmanagementhubsdk.GetMappingVendorNameEnum(desired.VendorName)
	if !ok {
		return "", "", "", fmt.Errorf("unsupported Profile vendorName %q", desired.VendorName)
	}
	osFamily, ok := osmanagementhubsdk.GetMappingOsFamilyEnum(desired.OsFamily)
	if !ok {
		return "", "", "", fmt.Errorf("unsupported Profile osFamily %q", desired.OsFamily)
	}
	archType, ok := osmanagementhubsdk.GetMappingArchTypeEnum(desired.ArchType)
	if !ok {
		return "", "", "", fmt.Errorf("unsupported Profile archType %q", desired.ArchType)
	}
	return vendorName, osFamily, archType, nil
}

func profileOptionalPlatformEnums(desired profileDesiredFields) (osmanagementhubsdk.VendorNameEnum, osmanagementhubsdk.OsFamilyEnum, osmanagementhubsdk.ArchTypeEnum, error) {
	var vendorName osmanagementhubsdk.VendorNameEnum
	var osFamily osmanagementhubsdk.OsFamilyEnum
	var archType osmanagementhubsdk.ArchTypeEnum
	if desired.VendorName != "" {
		value, ok := osmanagementhubsdk.GetMappingVendorNameEnum(desired.VendorName)
		if !ok {
			return "", "", "", fmt.Errorf("unsupported Profile vendorName %q", desired.VendorName)
		}
		vendorName = value
	}
	if desired.OsFamily != "" {
		value, ok := osmanagementhubsdk.GetMappingOsFamilyEnum(desired.OsFamily)
		if !ok {
			return "", "", "", fmt.Errorf("unsupported Profile osFamily %q", desired.OsFamily)
		}
		osFamily = value
	}
	if desired.ArchType != "" {
		value, ok := osmanagementhubsdk.GetMappingArchTypeEnum(desired.ArchType)
		if !ok {
			return "", "", "", fmt.Errorf("unsupported Profile archType %q", desired.ArchType)
		}
		archType = value
	}
	return vendorName, osFamily, archType, nil
}

func profileDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		tagValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		converted[namespace] = tagValues
	}
	return converted
}

func profileStatusDefinedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func profileStatusSoftwareSources(source []osmanagementhubsdk.SoftwareSourceDetails) []osmanagementhubv1beta1.ProfileSoftwareSource {
	if len(source) == 0 {
		return nil
	}
	converted := make([]osmanagementhubv1beta1.ProfileSoftwareSource, 0, len(source))
	for _, item := range source {
		converted = append(converted, osmanagementhubv1beta1.ProfileSoftwareSource{
			Id:                            profileString(item.Id),
			DisplayName:                   profileString(item.DisplayName),
			Description:                   profileString(item.Description),
			SoftwareSourceType:            string(item.SoftwareSourceType),
			IsMandatoryForAutonomousLinux: profileBool(item.IsMandatoryForAutonomousLinux),
		})
	}
	return converted
}

func profileTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func cloneProfileDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		cloned[namespace] = maps.Clone(values)
	}
	return cloned
}

func profileJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func profileSortedStrings(values []string) []string {
	cloned := profileStringSlice(values)
	sort.Strings(cloned)
	return cloned
}

func profileStringSlice(values []string) []string {
	cloned := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			cloned = append(cloned, trimmed)
		}
	}
	return cloned
}

func profileString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func profileBool(value *bool) bool {
	return value != nil && *value
}
