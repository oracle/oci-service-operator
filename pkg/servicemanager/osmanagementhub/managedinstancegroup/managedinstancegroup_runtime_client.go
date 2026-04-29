/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedinstancegroup

import (
	"context"
	"crypto/sha256"
	"fmt"
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
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type managedInstanceGroupOCIClient interface {
	CreateManagedInstanceGroup(context.Context, osmanagementhubsdk.CreateManagedInstanceGroupRequest) (osmanagementhubsdk.CreateManagedInstanceGroupResponse, error)
	GetManagedInstanceGroup(context.Context, osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error)
	ListManagedInstanceGroups(context.Context, osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error)
	UpdateManagedInstanceGroup(context.Context, osmanagementhubsdk.UpdateManagedInstanceGroupRequest) (osmanagementhubsdk.UpdateManagedInstanceGroupResponse, error)
	DeleteManagedInstanceGroup(context.Context, osmanagementhubsdk.DeleteManagedInstanceGroupRequest) (osmanagementhubsdk.DeleteManagedInstanceGroupResponse, error)
	AttachSoftwareSourcesToManagedInstanceGroup(context.Context, osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupRequest) (osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse, error)
	DetachSoftwareSourcesFromManagedInstanceGroup(context.Context, osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupRequest) (osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupResponse, error)
	AttachManagedInstancesToManagedInstanceGroup(context.Context, osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupRequest) (osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse, error)
	DetachManagedInstancesFromManagedInstanceGroup(context.Context, osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupRequest) (osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupResponse, error)
}

type managedInstanceGroupAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e managedInstanceGroupAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e managedInstanceGroupAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerManagedInstanceGroupRuntimeHooksMutator(func(_ *ManagedInstanceGroupServiceManager, hooks *ManagedInstanceGroupRuntimeHooks) {
		applyManagedInstanceGroupRuntimeHooks(hooks)
	})
	newManagedInstanceGroupServiceClient = newManagedInstanceGroupRuntimeServiceClient
}

func applyManagedInstanceGroupRuntimeHooks(hooks *ManagedInstanceGroupRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = managedInstanceGroupRuntimeSemantics()
	hooks.BuildCreateBody = buildManagedInstanceGroupCreateBody
	hooks.BuildUpdateBody = buildManagedInstanceGroupUpdateBody
	hooks.List.Fields = managedInstanceGroupListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listManagedInstanceGroupsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateManagedInstanceGroupCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleManagedInstanceGroupDeleteError
	wrapManagedInstanceGroupDeleteConfirmation(hooks)
}

func newManagedInstanceGroupServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client managedInstanceGroupOCIClient,
) ManagedInstanceGroupServiceClient {
	hooks := newManagedInstanceGroupRuntimeHooksWithOCIClient(client)
	applyManagedInstanceGroupRuntimeHooks(&hooks)
	delegate := defaultManagedInstanceGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.ManagedInstanceGroup](
			buildManagedInstanceGroupGeneratedRuntimeConfig(&ManagedInstanceGroupServiceManager{Log: log}, hooks),
		),
	}
	return wrapManagedInstanceGroupMembershipClient(client, wrapManagedInstanceGroupGeneratedClient(hooks, delegate))
}

func newManagedInstanceGroupRuntimeServiceClient(manager *ManagedInstanceGroupServiceManager) ManagedInstanceGroupServiceClient {
	sdkClient, err := osmanagementhubsdk.NewManagedInstanceGroupClientWithConfigurationProvider(manager.Provider)
	hooks := newManagedInstanceGroupRuntimeHooks(manager, sdkClient)
	config := buildManagedInstanceGroupGeneratedRuntimeConfig(manager, hooks)
	if err != nil {
		config.InitError = fmt.Errorf("initialize ManagedInstanceGroup OCI client: %w", err)
	}
	delegate := defaultManagedInstanceGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.ManagedInstanceGroup](config),
	}
	return wrapManagedInstanceGroupMembershipClient(sdkClient, wrapManagedInstanceGroupGeneratedClient(hooks, delegate))
}

func wrapManagedInstanceGroupMembershipClient(
	client managedInstanceGroupOCIClient,
	delegate ManagedInstanceGroupServiceClient,
) ManagedInstanceGroupServiceClient {
	if client == nil {
		return delegate
	}
	return managedInstanceGroupMembershipClient{
		delegate: delegate,
		client:   client,
	}
}

type managedInstanceGroupMembershipClient struct {
	delegate ManagedInstanceGroupServiceClient
	client   managedInstanceGroupOCIClient
}

func (c managedInstanceGroupMembershipClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || response.ShouldRequeue {
		return response, err
	}
	return c.reconcileMembership(ctx, resource, response)
}

func (c managedInstanceGroupMembershipClient) Delete(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c managedInstanceGroupMembershipClient) reconcileMembership(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	response servicemanager.OSOKResponse,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return response, nil
	}
	managedInstanceGroupID := trackedManagedInstanceGroupID(resource)
	if managedInstanceGroupID == "" {
		return response, nil
	}

	var result managedInstanceGroupMembershipActionResult
	var err error
	if resource.Spec.SoftwareSourceIds != nil {
		result, err = c.reconcileSoftwareSources(ctx, resource, managedInstanceGroupID)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}
	if resource.Spec.ManagedInstanceIds != nil {
		managedInstanceResult, err := c.reconcileManagedInstances(ctx, resource, managedInstanceGroupID)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		result = result.merge(managedInstanceResult)
	}
	if !result.changed {
		return response, nil
	}
	markManagedInstanceGroupMembershipUpdate(resource, result)
	return servicemanager.OSOKResponse{
		IsSuccessful:    true,
		ShouldRequeue:   true,
		RequeueDuration: time.Minute,
	}, nil
}

func (c managedInstanceGroupMembershipClient) reconcileSoftwareSources(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	managedInstanceGroupID string,
) (managedInstanceGroupMembershipActionResult, error) {
	toAttach, toDetach := managedInstanceGroupMembershipDiff(
		resource.Spec.SoftwareSourceIds,
		managedInstanceGroupStatusSoftwareSourceIDs(resource.Status),
	)

	var result managedInstanceGroupMembershipActionResult
	if len(toDetach) > 0 {
		response, err := c.client.DetachSoftwareSourcesFromManagedInstanceGroup(ctx, osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupRequest{
			ManagedInstanceGroupId: common.String(managedInstanceGroupID),
			DetachSoftwareSourcesFromManagedInstanceGroupDetails: osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupDetails{
				SoftwareSources: toDetach,
			},
			OpcRetryToken: managedInstanceGroupRetryToken(resource, "detach-software-sources", toDetach),
		})
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return result, err
		}
		result = result.merge(managedInstanceGroupSoftwareSourceActionResult(response, "detached software sources"))
	}
	if len(toAttach) > 0 {
		response, err := c.client.AttachSoftwareSourcesToManagedInstanceGroup(ctx, osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupRequest{
			ManagedInstanceGroupId: common.String(managedInstanceGroupID),
			AttachSoftwareSourcesToManagedInstanceGroupDetails: osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupDetails{
				SoftwareSources: toAttach,
			},
			OpcRetryToken: managedInstanceGroupRetryToken(resource, "attach-software-sources", toAttach),
		})
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return result, err
		}
		result = result.merge(managedInstanceGroupSoftwareSourceActionResult(response, "attached software sources"))
	}
	return result, nil
}

func (c managedInstanceGroupMembershipClient) reconcileManagedInstances(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	managedInstanceGroupID string,
) (managedInstanceGroupMembershipActionResult, error) {
	toAttach, toDetach := managedInstanceGroupMembershipDiff(
		resource.Spec.ManagedInstanceIds,
		resource.Status.ManagedInstanceIds,
	)

	var result managedInstanceGroupMembershipActionResult
	if len(toDetach) > 0 {
		response, err := c.client.DetachManagedInstancesFromManagedInstanceGroup(ctx, osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupRequest{
			ManagedInstanceGroupId: common.String(managedInstanceGroupID),
			DetachManagedInstancesFromManagedInstanceGroupDetails: osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupDetails{
				ManagedInstances: toDetach,
			},
			OpcRetryToken: managedInstanceGroupRetryToken(resource, "detach-managed-instances", toDetach),
		})
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return result, err
		}
		result = result.merge(managedInstanceGroupManagedInstanceActionResult(response, "detached managed instances"))
	}
	if len(toAttach) > 0 {
		response, err := c.client.AttachManagedInstancesToManagedInstanceGroup(ctx, osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupRequest{
			ManagedInstanceGroupId: common.String(managedInstanceGroupID),
			AttachManagedInstancesToManagedInstanceGroupDetails: osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupDetails{
				ManagedInstances: toAttach,
			},
			OpcRetryToken: managedInstanceGroupRetryToken(resource, "attach-managed-instances", toAttach),
		})
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return result, err
		}
		result = result.merge(managedInstanceGroupManagedInstanceActionResult(response, "attached managed instances"))
	}
	return result, nil
}

type managedInstanceGroupMembershipActionResult struct {
	changed       bool
	message       string
	opcRequestID  string
	workRequestID string
}

func (r managedInstanceGroupMembershipActionResult) merge(next managedInstanceGroupMembershipActionResult) managedInstanceGroupMembershipActionResult {
	if !next.changed {
		return r
	}
	if !r.changed {
		return next
	}
	r.changed = true
	r.message = next.message
	if next.opcRequestID != "" {
		r.opcRequestID = next.opcRequestID
	}
	if next.workRequestID != "" {
		r.workRequestID = next.workRequestID
	}
	return r
}

func managedInstanceGroupSoftwareSourceActionResult(response any, message string) managedInstanceGroupMembershipActionResult {
	return managedInstanceGroupActionResult(response, message, managedInstanceGroupSoftwareSourceWorkRequestID(response))
}

func managedInstanceGroupManagedInstanceActionResult(response any, message string) managedInstanceGroupMembershipActionResult {
	return managedInstanceGroupActionResult(response, message, managedInstanceGroupManagedInstanceWorkRequestID(response))
}

func managedInstanceGroupActionResult(response any, message string, workRequestID string) managedInstanceGroupMembershipActionResult {
	return managedInstanceGroupMembershipActionResult{
		changed:       true,
		message:       message,
		opcRequestID:  servicemanager.ResponseOpcRequestID(response),
		workRequestID: strings.TrimSpace(workRequestID),
	}
}

func newManagedInstanceGroupRuntimeHooksWithOCIClient(client managedInstanceGroupOCIClient) ManagedInstanceGroupRuntimeHooks {
	return ManagedInstanceGroupRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*osmanagementhubv1beta1.ManagedInstanceGroup]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osmanagementhubv1beta1.ManagedInstanceGroup]{},
		StatusHooks:     generatedruntime.StatusHooks[*osmanagementhubv1beta1.ManagedInstanceGroup]{},
		ParityHooks:     generatedruntime.ParityHooks[*osmanagementhubv1beta1.ManagedInstanceGroup]{},
		Async:           generatedruntime.AsyncHooks[*osmanagementhubv1beta1.ManagedInstanceGroup]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osmanagementhubv1beta1.ManagedInstanceGroup]{},
		Create: runtimeOperationHooks[osmanagementhubsdk.CreateManagedInstanceGroupRequest, osmanagementhubsdk.CreateManagedInstanceGroupResponse]{
			Fields: managedInstanceGroupCreateFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.CreateManagedInstanceGroupRequest) (osmanagementhubsdk.CreateManagedInstanceGroupResponse, error) {
				if client == nil {
					return osmanagementhubsdk.CreateManagedInstanceGroupResponse{}, fmt.Errorf("ManagedInstanceGroup OCI client is nil")
				}
				return client.CreateManagedInstanceGroup(ctx, request)
			},
		},
		Get: runtimeOperationHooks[osmanagementhubsdk.GetManagedInstanceGroupRequest, osmanagementhubsdk.GetManagedInstanceGroupResponse]{
			Fields: managedInstanceGroupGetFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
				if client == nil {
					return osmanagementhubsdk.GetManagedInstanceGroupResponse{}, fmt.Errorf("ManagedInstanceGroup OCI client is nil")
				}
				return client.GetManagedInstanceGroup(ctx, request)
			},
		},
		List: runtimeOperationHooks[osmanagementhubsdk.ListManagedInstanceGroupsRequest, osmanagementhubsdk.ListManagedInstanceGroupsResponse]{
			Fields: managedInstanceGroupListFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
				if client == nil {
					return osmanagementhubsdk.ListManagedInstanceGroupsResponse{}, fmt.Errorf("ManagedInstanceGroup OCI client is nil")
				}
				return client.ListManagedInstanceGroups(ctx, request)
			},
		},
		Update: runtimeOperationHooks[osmanagementhubsdk.UpdateManagedInstanceGroupRequest, osmanagementhubsdk.UpdateManagedInstanceGroupResponse]{
			Fields: managedInstanceGroupUpdateFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.UpdateManagedInstanceGroupRequest) (osmanagementhubsdk.UpdateManagedInstanceGroupResponse, error) {
				if client == nil {
					return osmanagementhubsdk.UpdateManagedInstanceGroupResponse{}, fmt.Errorf("ManagedInstanceGroup OCI client is nil")
				}
				return client.UpdateManagedInstanceGroup(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[osmanagementhubsdk.DeleteManagedInstanceGroupRequest, osmanagementhubsdk.DeleteManagedInstanceGroupResponse]{
			Fields: managedInstanceGroupDeleteFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.DeleteManagedInstanceGroupRequest) (osmanagementhubsdk.DeleteManagedInstanceGroupResponse, error) {
				if client == nil {
					return osmanagementhubsdk.DeleteManagedInstanceGroupResponse{}, fmt.Errorf("ManagedInstanceGroup OCI client is nil")
				}
				return client.DeleteManagedInstanceGroup(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ManagedInstanceGroupServiceClient) ManagedInstanceGroupServiceClient{},
	}
}

func managedInstanceGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:       "osmanagementhub",
		FormalSlug:          "managedinstancegroup",
		StatusProjection:    "required",
		SecretSideEffects:   "none",
		FinalizerPolicy:     "retain-until-confirmed-delete",
		Async:               managedInstanceGroupLifecycleAsyncSemantics(),
		Lifecycle:           managedInstanceGroupLifecycleSemantics(),
		Delete:              managedInstanceGroupDeleteSemantics(),
		List:                managedInstanceGroupListSemantics(),
		Mutation:            managedInstanceGroupMutationSemantics(),
		Hooks:               managedInstanceGroupHookSet(),
		CreateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write", Hooks: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}}},
		UpdateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write", Hooks: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}}},
		DeleteFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "confirm-delete", Hooks: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}}},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func managedInstanceGroupLifecycleAsyncSemantics() *generatedruntime.AsyncSemantics {
	return &generatedruntime.AsyncSemantics{
		Strategy:             "lifecycle",
		Runtime:              "generatedruntime",
		FormalClassification: "lifecycle",
	}
}

func managedInstanceGroupLifecycleSemantics() generatedruntime.LifecycleSemantics {
	return generatedruntime.LifecycleSemantics{
		ProvisioningStates: []string{string(osmanagementhubsdk.ManagedInstanceGroupLifecycleStateCreating)},
		UpdatingStates:     []string{string(osmanagementhubsdk.ManagedInstanceGroupLifecycleStateUpdating)},
		ActiveStates:       []string{string(osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive)},
	}
}

func managedInstanceGroupDeleteSemantics() generatedruntime.DeleteSemantics {
	return generatedruntime.DeleteSemantics{
		Policy:         "required",
		PendingStates:  []string{string(osmanagementhubsdk.ManagedInstanceGroupLifecycleStateDeleting)},
		TerminalStates: []string{string(osmanagementhubsdk.ManagedInstanceGroupLifecycleStateDeleted)},
	}
}

func managedInstanceGroupListSemantics() *generatedruntime.ListSemantics {
	return &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields: []string{
			"compartmentId",
			"displayName",
			"osFamily",
			"archType",
			"vendorName",
			"location",
		},
	}
}

func managedInstanceGroupMutationSemantics() generatedruntime.MutationSemantics {
	return generatedruntime.MutationSemantics{
		UpdateCandidate: []string{
			"displayName",
			"description",
			"notificationTopicId",
			"autonomousSettings.isDataCollectionAuthorized",
			"freeformTags",
			"definedTags",
		},
		Mutable: []string{
			"displayName",
			"description",
			"notificationTopicId",
			"autonomousSettings.isDataCollectionAuthorized",
			"freeformTags",
			"definedTags",
			"softwareSourceIds",
			"managedInstanceIds",
		},
		ForceNew:      []string{"compartmentId", "osFamily", "archType", "vendorName", "location"},
		ConflictsWith: map[string][]string{},
	}
}

func managedInstanceGroupHookSet() generatedruntime.HookSet {
	return generatedruntime.HookSet{
		Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
	}
}

func managedInstanceGroupCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateManagedInstanceGroupDetails", RequestName: "CreateManagedInstanceGroupDetails", Contribution: "body"},
	}
}

func managedInstanceGroupGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagedInstanceGroupId", RequestName: "managedInstanceGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func managedInstanceGroupListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagedInstanceGroupId", RequestName: "managedInstanceGroupId", Contribution: "query", PreferResourceID: true},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "ArchType", RequestName: "archType", Contribution: "query", LookupPaths: []string{"status.archType", "spec.archType", "archType"}},
		{FieldName: "OsFamily", RequestName: "osFamily", Contribution: "query", LookupPaths: []string{"status.osFamily", "spec.osFamily", "osFamily"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", LookupPaths: []string{"status.lifecycleState", "lifecycleState"}},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func managedInstanceGroupUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagedInstanceGroupId", RequestName: "managedInstanceGroupId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateManagedInstanceGroupDetails", RequestName: "UpdateManagedInstanceGroupDetails", Contribution: "body"},
	}
}

func managedInstanceGroupDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagedInstanceGroupId", RequestName: "managedInstanceGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func buildManagedInstanceGroupCreateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	_ string,
) (any, error) {
	if resource == nil {
		return osmanagementhubsdk.CreateManagedInstanceGroupDetails{}, fmt.Errorf("ManagedInstanceGroup resource is nil")
	}
	spec := resource.Spec
	details := osmanagementhubsdk.CreateManagedInstanceGroupDetails{
		DisplayName:        common.String(strings.TrimSpace(spec.DisplayName)),
		CompartmentId:      common.String(strings.TrimSpace(spec.CompartmentId)),
		OsFamily:           osmanagementhubsdk.OsFamilyEnum(strings.TrimSpace(spec.OsFamily)),
		ArchType:           osmanagementhubsdk.ArchTypeEnum(strings.TrimSpace(spec.ArchType)),
		VendorName:         osmanagementhubsdk.VendorNameEnum(strings.TrimSpace(spec.VendorName)),
		AutonomousSettings: managedInstanceGroupAutonomousSettingsFromSpec(spec.AutonomousSettings),
	}
	if spec.Description != "" {
		details.Description = common.String(strings.TrimSpace(spec.Description))
	}
	if spec.Location != "" {
		details.Location = osmanagementhubsdk.ManagedInstanceLocationEnum(strings.TrimSpace(spec.Location))
	}
	if len(spec.SoftwareSourceIds) > 0 {
		details.SoftwareSourceIds = managedInstanceGroupCloneStrings(spec.SoftwareSourceIds)
	}
	if len(spec.ManagedInstanceIds) > 0 {
		details.ManagedInstanceIds = managedInstanceGroupCloneStrings(spec.ManagedInstanceIds)
	}
	if spec.NotificationTopicId != "" {
		details.NotificationTopicId = common.String(strings.TrimSpace(spec.NotificationTopicId))
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = managedInstanceGroupCloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = managedInstanceGroupDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details, nil
}

func buildManagedInstanceGroupUpdateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return osmanagementhubsdk.UpdateManagedInstanceGroupDetails{}, false, fmt.Errorf("ManagedInstanceGroup resource is nil")
	}
	current, ok := managedInstanceGroupFromResponse(currentResponse)
	if !ok {
		return osmanagementhubsdk.UpdateManagedInstanceGroupDetails{}, false, fmt.Errorf("current ManagedInstanceGroup response does not expose a resource body")
	}
	if err := validateManagedInstanceGroupCreateOnlyDrift(resource, current); err != nil {
		return osmanagementhubsdk.UpdateManagedInstanceGroupDetails{}, false, err
	}

	update := osmanagementhubsdk.UpdateManagedInstanceGroupDetails{}
	updateNeeded := false
	updateNeeded = applyManagedInstanceGroupStringUpdates(&update, resource.Spec, current) || updateNeeded
	updateNeeded = applyManagedInstanceGroupAutonomousSettingsUpdate(&update, resource.Spec, current) || updateNeeded
	updateNeeded = applyManagedInstanceGroupTagUpdates(&update, resource.Spec, current) || updateNeeded
	if !updateNeeded {
		return osmanagementhubsdk.UpdateManagedInstanceGroupDetails{}, false, nil
	}
	return update, true, nil
}

func applyManagedInstanceGroupStringUpdates(
	update *osmanagementhubsdk.UpdateManagedInstanceGroupDetails,
	spec osmanagementhubv1beta1.ManagedInstanceGroupSpec,
	current osmanagementhubsdk.ManagedInstanceGroup,
) bool {
	updateNeeded := false
	if desired := strings.TrimSpace(spec.DisplayName); desired != "" && desired != managedInstanceGroupString(current.DisplayName) {
		update.DisplayName = common.String(desired)
		updateNeeded = true
	}
	if desired := strings.TrimSpace(spec.Description); desired != "" && desired != managedInstanceGroupString(current.Description) {
		update.Description = common.String(desired)
		updateNeeded = true
	}
	if desired := strings.TrimSpace(spec.NotificationTopicId); desired != "" && desired != managedInstanceGroupString(current.NotificationTopicId) {
		update.NotificationTopicId = common.String(desired)
		updateNeeded = true
	}
	return updateNeeded
}

func applyManagedInstanceGroupAutonomousSettingsUpdate(
	update *osmanagementhubsdk.UpdateManagedInstanceGroupDetails,
	spec osmanagementhubv1beta1.ManagedInstanceGroupSpec,
	current osmanagementhubsdk.ManagedInstanceGroup,
) bool {
	desired := spec.AutonomousSettings.IsDataCollectionAuthorized
	currentValue := false
	if current.AutonomousSettings != nil && current.AutonomousSettings.IsDataCollectionAuthorized != nil {
		currentValue = *current.AutonomousSettings.IsDataCollectionAuthorized
	}
	if desired == currentValue {
		return false
	}
	update.AutonomousSettings = managedInstanceGroupAutonomousSettingsFromSpec(spec.AutonomousSettings)
	return true
}

func applyManagedInstanceGroupTagUpdates(
	update *osmanagementhubsdk.UpdateManagedInstanceGroupDetails,
	spec osmanagementhubv1beta1.ManagedInstanceGroupSpec,
	current osmanagementhubsdk.ManagedInstanceGroup,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !reflect.DeepEqual(spec.FreeformTags, current.FreeformTags) {
		update.FreeformTags = managedInstanceGroupCloneStringMap(spec.FreeformTags)
		updateNeeded = true
	}

	desiredDefinedTags := managedInstanceGroupDefinedTagsFromSpec(spec.DefinedTags)
	if spec.DefinedTags != nil && !reflect.DeepEqual(desiredDefinedTags, current.DefinedTags) {
		update.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	return updateNeeded
}

func managedInstanceGroupAutonomousSettingsFromSpec(
	settings osmanagementhubv1beta1.ManagedInstanceGroupAutonomousSettings,
) *osmanagementhubsdk.UpdatableAutonomousSettings {
	return &osmanagementhubsdk.UpdatableAutonomousSettings{
		IsDataCollectionAuthorized: common.Bool(settings.IsDataCollectionAuthorized),
	}
}

func validateManagedInstanceGroupCreateOnlyDriftForResponse(
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	currentResponse any,
) error {
	current, ok := managedInstanceGroupFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateManagedInstanceGroupCreateOnlyDrift(resource, current)
}

func validateManagedInstanceGroupCreateOnlyDrift(
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	_ osmanagementhubsdk.ManagedInstanceGroup,
) error {
	if resource == nil {
		return fmt.Errorf("ManagedInstanceGroup resource is nil")
	}
	return nil
}

func managedInstanceGroupFromResponse(response any) (osmanagementhubsdk.ManagedInstanceGroup, bool) {
	if current, ok := managedInstanceGroupBodyFromResponse(response); ok {
		return current, true
	}
	return managedInstanceGroupSummaryFromResponse(response)
}

func managedInstanceGroupBodyFromResponse(response any) (osmanagementhubsdk.ManagedInstanceGroup, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.ManagedInstanceGroup:
		return current, true
	case osmanagementhubsdk.CreateManagedInstanceGroupResponse:
		return current.ManagedInstanceGroup, true
	case osmanagementhubsdk.GetManagedInstanceGroupResponse:
		return current.ManagedInstanceGroup, true
	case osmanagementhubsdk.UpdateManagedInstanceGroupResponse:
		return current.ManagedInstanceGroup, true
	default:
		return managedInstanceGroupBodyFromPointerResponse(response)
	}
}

func managedInstanceGroupBodyFromPointerResponse(response any) (osmanagementhubsdk.ManagedInstanceGroup, bool) {
	switch current := response.(type) {
	case *osmanagementhubsdk.ManagedInstanceGroup:
		if current == nil {
			return osmanagementhubsdk.ManagedInstanceGroup{}, false
		}
		return *current, true
	case *osmanagementhubsdk.CreateManagedInstanceGroupResponse:
		if current == nil {
			return osmanagementhubsdk.ManagedInstanceGroup{}, false
		}
		return current.ManagedInstanceGroup, true
	case *osmanagementhubsdk.GetManagedInstanceGroupResponse:
		if current == nil {
			return osmanagementhubsdk.ManagedInstanceGroup{}, false
		}
		return current.ManagedInstanceGroup, true
	case *osmanagementhubsdk.UpdateManagedInstanceGroupResponse:
		if current == nil {
			return osmanagementhubsdk.ManagedInstanceGroup{}, false
		}
		return current.ManagedInstanceGroup, true
	default:
		return osmanagementhubsdk.ManagedInstanceGroup{}, false
	}
}

func managedInstanceGroupSummaryFromResponse(response any) (osmanagementhubsdk.ManagedInstanceGroup, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.ManagedInstanceGroupSummary:
		return managedInstanceGroupFromSummary(current), true
	case *osmanagementhubsdk.ManagedInstanceGroupSummary:
		if current == nil {
			return osmanagementhubsdk.ManagedInstanceGroup{}, false
		}
		return managedInstanceGroupFromSummary(*current), true
	default:
		return osmanagementhubsdk.ManagedInstanceGroup{}, false
	}
}

func managedInstanceGroupFromSummary(summary osmanagementhubsdk.ManagedInstanceGroupSummary) osmanagementhubsdk.ManagedInstanceGroup {
	return osmanagementhubsdk.ManagedInstanceGroup{
		Id:                         summary.Id,
		CompartmentId:              summary.CompartmentId,
		LifecycleState:             summary.LifecycleState,
		DisplayName:                summary.DisplayName,
		Description:                summary.Description,
		ManagedInstanceCount:       summary.ManagedInstanceCount,
		Location:                   summary.Location,
		TimeCreated:                summary.TimeCreated,
		TimeModified:               summary.TimeModified,
		OsFamily:                   summary.OsFamily,
		ArchType:                   summary.ArchType,
		VendorName:                 summary.VendorName,
		NotificationTopicId:        summary.NotificationTopicId,
		AutonomousSettings:         summary.AutonomousSettings,
		IsManagedByAutonomousLinux: summary.IsManagedByAutonomousLinux,
		FreeformTags:               summary.FreeformTags,
		DefinedTags:                summary.DefinedTags,
		SystemTags:                 summary.SystemTags,
	}
}

func managedInstanceGroupStatusSoftwareSourceIDs(status osmanagementhubv1beta1.ManagedInstanceGroupStatus) []string {
	ids := make([]string, 0, len(status.SoftwareSourceIds)+len(status.SoftwareSources))
	for _, source := range status.SoftwareSourceIds {
		if id := strings.TrimSpace(source.Id); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) != 0 {
		return ids
	}
	for _, source := range status.SoftwareSources {
		if id := strings.TrimSpace(source.Id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func managedInstanceGroupMembershipDiff(desired []string, current []string) ([]string, []string) {
	desired = managedInstanceGroupNormalizedStrings(desired)
	current = managedInstanceGroupNormalizedStrings(current)
	desiredSet := managedInstanceGroupStringSet(desired)
	currentSet := managedInstanceGroupStringSet(current)

	toAttach := make([]string, 0)
	for _, id := range desired {
		if _, ok := currentSet[id]; !ok {
			toAttach = append(toAttach, id)
		}
	}
	toDetach := make([]string, 0)
	for _, id := range current {
		if _, ok := desiredSet[id]; !ok {
			toDetach = append(toDetach, id)
		}
	}
	return toAttach, toDetach
}

func managedInstanceGroupStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func managedInstanceGroupRetryToken(
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	action string,
	ids []string,
) *string {
	seed := strings.Join([]string{
		resource.Namespace,
		resource.Name,
		string(resource.UID),
		action,
		strings.Join(managedInstanceGroupNormalizedStrings(ids), ","),
	}, "/")
	sum := sha256.Sum256([]byte(seed))
	return common.String(fmt.Sprintf("osok-%x", sum[:16]))
}

func managedInstanceGroupSoftwareSourceWorkRequestID(response any) string {
	switch current := response.(type) {
	case osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse:
		return managedInstanceGroupString(current.OpcWorkRequestId)
	case *osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse:
		if current == nil {
			return ""
		}
		return managedInstanceGroupString(current.OpcWorkRequestId)
	default:
		return ""
	}
}

func managedInstanceGroupManagedInstanceWorkRequestID(response any) string {
	switch current := response.(type) {
	case osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse:
		return managedInstanceGroupString(current.OpcWorkRequestId)
	case *osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse:
		if current == nil {
			return ""
		}
		return managedInstanceGroupString(current.OpcWorkRequestId)
	default:
		return ""
	}
}

func markManagedInstanceGroupMembershipUpdate(
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	result managedInstanceGroupMembershipActionResult,
) {
	if resource == nil || !result.changed {
		return
	}
	status := &resource.Status.OsokStatus
	servicemanager.SetOpcRequestID(status, result.opcRequestID)
	now := metav1.Now()
	message := strings.TrimSpace(result.message)
	if message == "" {
		message = "managed instance group membership update accepted"
	}
	status.Message = message
	status.Reason = string(shared.Updating)
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          managedInstanceGroupMembershipAsyncSource(result.workRequestID),
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   result.workRequestID,
		RawStatus:       "UPDATING",
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Updating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func managedInstanceGroupMembershipAsyncSource(workRequestID string) shared.OSOKAsyncSource {
	if strings.TrimSpace(workRequestID) != "" {
		return shared.OSOKAsyncSourceWorkRequest
	}
	return shared.OSOKAsyncSourceLifecycle
}

func listManagedInstanceGroupsAllPages(
	call func(context.Context, osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error),
) func(context.Context, osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
	return func(ctx context.Context, request osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
		if call == nil {
			return osmanagementhubsdk.ListManagedInstanceGroupsResponse{}, fmt.Errorf("ManagedInstanceGroup list operation is not configured")
		}
		var combined osmanagementhubsdk.ListManagedInstanceGroupsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.RawResponse = response.RawResponse
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleManagedInstanceGroupDeleteError(
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return managedInstanceGroupAmbiguousNotFoundError{
		message:      fmt.Sprintf("ManagedInstanceGroup delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func wrapManagedInstanceGroupDeleteConfirmation(hooks *ManagedInstanceGroupRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getManagedInstanceGroup := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagedInstanceGroupServiceClient) ManagedInstanceGroupServiceClient {
		return managedInstanceGroupDeleteConfirmationClient{
			delegate:                delegate,
			getManagedInstanceGroup: getManagedInstanceGroup,
		}
	})
}

type managedInstanceGroupDeleteConfirmationClient struct {
	delegate                ManagedInstanceGroupServiceClient
	getManagedInstanceGroup func(context.Context, osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error)
}

func (c managedInstanceGroupDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c managedInstanceGroupDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c managedInstanceGroupDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
) error {
	if c.getManagedInstanceGroup == nil || resource == nil {
		return nil
	}
	managedInstanceGroupID := trackedManagedInstanceGroupID(resource)
	if managedInstanceGroupID == "" {
		return nil
	}
	_, err := c.getManagedInstanceGroup(ctx, osmanagementhubsdk.GetManagedInstanceGroupRequest{
		ManagedInstanceGroupId: common.String(managedInstanceGroupID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return managedInstanceGroupAmbiguousNotFoundError{
		message:      fmt.Sprintf("ManagedInstanceGroup delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func trackedManagedInstanceGroupID(resource *osmanagementhubv1beta1.ManagedInstanceGroup) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func managedInstanceGroupString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func managedInstanceGroupCloneStrings(values []string) []string {
	cloned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cloned = append(cloned, value)
	}
	return cloned
}

func managedInstanceGroupCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func managedInstanceGroupDefinedTagsFromSpec(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		convertedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			convertedValues[key] = value
		}
		converted[namespace] = convertedValues
	}
	return converted
}

func managedInstanceGroupNormalizedStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

var _ interface{ GetOpcRequestID() string } = managedInstanceGroupAmbiguousNotFoundError{}
