/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lockbox

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	lockboxsdk "github.com/oracle/oci-go-sdk/v65/lockbox"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type lockboxOCIClient interface {
	CreateLockbox(context.Context, lockboxsdk.CreateLockboxRequest) (lockboxsdk.CreateLockboxResponse, error)
	GetLockbox(context.Context, lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error)
	ListLockboxes(context.Context, lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error)
	UpdateLockbox(context.Context, lockboxsdk.UpdateLockboxRequest) (lockboxsdk.UpdateLockboxResponse, error)
	DeleteLockbox(context.Context, lockboxsdk.DeleteLockboxRequest) (lockboxsdk.DeleteLockboxResponse, error)
}

type ambiguousLockboxNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousLockboxNotFoundError) Error() string {
	return e.message
}

func (e ambiguousLockboxNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerLockboxRuntimeHooksMutator(func(_ *LockboxServiceManager, hooks *LockboxRuntimeHooks) {
		applyLockboxRuntimeHooks(hooks)
	})
}

func applyLockboxRuntimeHooks(hooks *LockboxRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newLockboxRuntimeSemantics()
	hooks.BuildCreateBody = buildLockboxCreateBody
	hooks.BuildUpdateBody = buildLockboxUpdateBody
	hooks.List.Fields = lockboxListFields()
	hooks.Read.Get = lockboxProjectedGetReadOperation(hooks)
	hooks.Read.List = lockboxPaginatedListReadOperation(hooks)
	hooks.DeleteHooks.HandleError = handleLockboxDeleteError
	wrapLockboxDeleteConfirmation(hooks)
}

func newLockboxServiceClientWithOCIClient(log loggerutil.OSOKLogger, client lockboxOCIClient) LockboxServiceClient {
	hooks := newLockboxRuntimeHooksWithOCIClient(client)
	applyLockboxRuntimeHooks(&hooks)
	delegate := defaultLockboxServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*lockboxv1beta1.Lockbox](
			buildLockboxGeneratedRuntimeConfig(&LockboxServiceManager{Log: log}, hooks),
		),
	}
	return wrapLockboxGeneratedClient(hooks, delegate)
}

func newLockboxRuntimeHooksWithOCIClient(client lockboxOCIClient) LockboxRuntimeHooks {
	return LockboxRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*lockboxv1beta1.Lockbox]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*lockboxv1beta1.Lockbox]{},
		StatusHooks:     generatedruntime.StatusHooks[*lockboxv1beta1.Lockbox]{},
		ParityHooks:     generatedruntime.ParityHooks[*lockboxv1beta1.Lockbox]{},
		Async:           generatedruntime.AsyncHooks[*lockboxv1beta1.Lockbox]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*lockboxv1beta1.Lockbox]{},
		Create: runtimeOperationHooks[lockboxsdk.CreateLockboxRequest, lockboxsdk.CreateLockboxResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateLockboxDetails", RequestName: "CreateLockboxDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request lockboxsdk.CreateLockboxRequest) (lockboxsdk.CreateLockboxResponse, error) {
				return client.CreateLockbox(ctx, request)
			},
		},
		Get: runtimeOperationHooks[lockboxsdk.GetLockboxRequest, lockboxsdk.GetLockboxResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LockboxId", RequestName: "lockboxId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error) {
				return client.GetLockbox(ctx, request)
			},
		},
		List: runtimeOperationHooks[lockboxsdk.ListLockboxesRequest, lockboxsdk.ListLockboxesResponse]{
			Fields: lockboxListFields(),
			Call: func(ctx context.Context, request lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error) {
				return client.ListLockboxes(ctx, request)
			},
		},
		Update: runtimeOperationHooks[lockboxsdk.UpdateLockboxRequest, lockboxsdk.UpdateLockboxResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LockboxId", RequestName: "lockboxId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateLockboxDetails", RequestName: "UpdateLockboxDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request lockboxsdk.UpdateLockboxRequest) (lockboxsdk.UpdateLockboxResponse, error) {
				return client.UpdateLockbox(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[lockboxsdk.DeleteLockboxRequest, lockboxsdk.DeleteLockboxResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LockboxId", RequestName: "lockboxId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request lockboxsdk.DeleteLockboxRequest) (lockboxsdk.DeleteLockboxResponse, error) {
				return client.DeleteLockbox(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LockboxServiceClient) LockboxServiceClient{},
	}
}

func newLockboxRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "lockbox",
		FormalSlug:    "lockbox",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(lockboxsdk.LockboxLifecycleStateCreating)},
			UpdatingStates:     []string{string(lockboxsdk.LockboxLifecycleStateUpdating)},
			ActiveStates:       []string{string(lockboxsdk.LockboxLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(lockboxsdk.LockboxLifecycleStateDeleting)},
			TerminalStates: []string{string(lockboxsdk.LockboxLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "resourceId", "lockboxPartner", "partnerId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"approvalTemplateId",
				"maxAccessDuration",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"resourceId",
				"compartmentId",
				"accessContextAttributes",
				"lockboxPartner",
				"partnerId",
				"partnerCompartmentId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func lockboxListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true, LookupPaths: []string{"status.id", "status.status.ocid", "id", "ocid"}},
		{FieldName: "ResourceId", RequestName: "resourceId", Contribution: "query", LookupPaths: []string{"status.resourceId", "spec.resourceId", "resourceId"}},
		{FieldName: "LockboxPartner", RequestName: "lockboxPartner", Contribution: "query", LookupPaths: []string{"status.lockboxPartner", "spec.lockboxPartner", "lockboxPartner"}},
		{FieldName: "PartnerId", RequestName: "partnerId", Contribution: "query", LookupPaths: []string{"status.partnerId", "spec.partnerId", "partnerId"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildLockboxCreateBody(_ context.Context, resource *lockboxv1beta1.Lockbox, _ string) (any, error) {
	if resource == nil {
		return lockboxsdk.CreateLockboxDetails{}, fmt.Errorf("lockbox resource is nil")
	}
	if err := validateLockboxSpec(resource.Spec); err != nil {
		return lockboxsdk.CreateLockboxDetails{}, err
	}

	spec := resource.Spec
	body := lockboxsdk.CreateLockboxDetails{
		ResourceId:              common.String(strings.TrimSpace(spec.ResourceId)),
		CompartmentId:           common.String(strings.TrimSpace(spec.CompartmentId)),
		AccessContextAttributes: lockboxAccessContextAttributesFromSpec(spec.AccessContextAttributes),
	}
	applyLockboxCreateOptionalFields(&body, spec)
	return body, nil
}

func applyLockboxCreateOptionalFields(
	body *lockboxsdk.CreateLockboxDetails,
	spec lockboxv1beta1.LockboxSpec,
) {
	if value := lockboxOptionalString(spec.DisplayName); value != nil {
		body.DisplayName = value
	}
	if value := strings.TrimSpace(spec.LockboxPartner); value != "" {
		body.LockboxPartner = lockboxsdk.LockboxPartnerEnum(value)
	}
	if value := lockboxOptionalString(spec.PartnerId); value != nil {
		body.PartnerId = value
	}
	if value := lockboxOptionalString(spec.PartnerCompartmentId); value != nil {
		body.PartnerCompartmentId = value
	}
	if value := lockboxOptionalString(spec.ApprovalTemplateId); value != nil {
		body.ApprovalTemplateId = value
	}
	if value := lockboxOptionalString(spec.MaxAccessDuration); value != nil {
		body.MaxAccessDuration = value
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
}

func buildLockboxUpdateBody(
	_ context.Context,
	resource *lockboxv1beta1.Lockbox,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return lockboxsdk.UpdateLockboxDetails{}, false, fmt.Errorf("lockbox resource is nil")
	}
	if err := validateLockboxSpec(resource.Spec); err != nil {
		return lockboxsdk.UpdateLockboxDetails{}, false, err
	}

	current, ok := lockboxProjectionFromResponse(currentResponse)
	if !ok {
		return lockboxsdk.UpdateLockboxDetails{}, false, fmt.Errorf("current Lockbox response does not expose a Lockbox body")
	}

	spec := resource.Spec
	body := lockboxsdk.UpdateLockboxDetails{}
	updateNeeded := false
	if desired, ok := lockboxDesiredStringUpdate(spec.DisplayName, current.DisplayName); ok {
		body.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := lockboxDesiredStringUpdate(spec.ApprovalTemplateId, current.ApprovalTemplateId); ok {
		body.ApprovalTemplateId = desired
		updateNeeded = true
	}
	if desired, ok := lockboxDesiredStringUpdate(spec.MaxAccessDuration, current.MaxAccessDuration); ok {
		body.MaxAccessDuration = desired
		updateNeeded = true
	}
	if desired, ok := lockboxDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		body.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := lockboxDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		body.DefinedTags = desired
		updateNeeded = true
	}

	return body, updateNeeded, nil
}

func validateLockboxSpec(spec lockboxv1beta1.LockboxSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ResourceId) == "" {
		missing = append(missing, "resourceId")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(spec.AccessContextAttributes.Items) == 0 {
		missing = append(missing, "accessContextAttributes.items")
	}
	for index, item := range spec.AccessContextAttributes.Items {
		if strings.TrimSpace(item.Name) == "" {
			missing = append(missing, fmt.Sprintf("accessContextAttributes.items[%d].name", index))
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("lockbox spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if value := strings.TrimSpace(spec.LockboxPartner); value != "" {
		if _, ok := lockboxsdk.GetMappingLockboxPartnerEnum(value); !ok {
			return fmt.Errorf("unsupported enum value for LockboxPartner: %s", value)
		}
	}
	return nil
}

func lockboxAccessContextAttributesFromSpec(
	spec lockboxv1beta1.LockboxAccessContextAttributes,
) *lockboxsdk.AccessContextAttributeCollection {
	items := make([]lockboxsdk.AccessContextAttribute, 0, len(spec.Items))
	for _, item := range spec.Items {
		items = append(items, lockboxsdk.AccessContextAttribute{
			Name:         common.String(strings.TrimSpace(item.Name)),
			Description:  lockboxOptionalString(item.Description),
			DefaultValue: lockboxOptionalString(item.DefaultValue),
			Values:       append([]string(nil), item.Values...),
		})
	}
	return &lockboxsdk.AccessContextAttributeCollection{Items: items}
}

func lockboxAccessContextAttributesFromSDK(
	current *lockboxsdk.AccessContextAttributeCollection,
) lockboxv1beta1.LockboxAccessContextAttributes {
	if current == nil {
		return lockboxv1beta1.LockboxAccessContextAttributes{}
	}

	items := make([]lockboxv1beta1.LockboxAccessContextAttributesItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, lockboxv1beta1.LockboxAccessContextAttributesItem{
			Name:         lockboxStringValue(item.Name),
			Description:  lockboxStringValue(item.Description),
			DefaultValue: lockboxStringValue(item.DefaultValue),
			Values:       append([]string(nil), item.Values...),
		})
	}
	return lockboxv1beta1.LockboxAccessContextAttributes{Items: items}
}

func lockboxProjectedGetReadOperation(hooks *LockboxRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.Get.Call == nil {
		return nil
	}

	getCall := hooks.Get.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &lockboxsdk.GetLockboxRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*lockboxsdk.GetLockboxRequest)
			if !ok {
				return nil, fmt.Errorf("expected *lockbox.GetLockboxRequest, got %T", request)
			}
			response, err := getCall(ctx, *typed)
			if err != nil {
				return nil, err
			}
			return lockboxProjectedResponseFromSDK(response.Lockbox, response.OpcRequestId), nil
		},
	}
}

func lockboxPaginatedListReadOperation(hooks *LockboxRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}

	listCall := hooks.List.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &lockboxsdk.ListLockboxesRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*lockboxsdk.ListLockboxesRequest)
			if !ok {
				return nil, fmt.Errorf("expected *lockbox.ListLockboxesRequest, got %T", request)
			}
			response, err := listLockboxPages(ctx, listCall, *typed)
			if err != nil {
				return nil, err
			}
			return lockboxProjectedListResponseFromSDK(response), nil
		},
	}
}

func listLockboxPages(
	ctx context.Context,
	call func(context.Context, lockboxsdk.ListLockboxesRequest) (lockboxsdk.ListLockboxesResponse, error),
	request lockboxsdk.ListLockboxesRequest,
) (lockboxsdk.ListLockboxesResponse, error) {
	if call == nil {
		return lockboxsdk.ListLockboxesResponse{}, fmt.Errorf("lockbox list operation is not configured")
	}

	seenPages := map[string]struct{}{}
	var combined lockboxsdk.ListLockboxesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return lockboxsdk.ListLockboxesResponse{}, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := lockboxStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return lockboxsdk.ListLockboxesResponse{}, fmt.Errorf("lockbox list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

type lockboxStatusProjection struct {
	Id                      string                                        `json:"id,omitempty"`
	DisplayName             string                                        `json:"displayName,omitempty"`
	CompartmentId           string                                        `json:"compartmentId,omitempty"`
	ResourceId              string                                        `json:"resourceId,omitempty"`
	TimeCreated             string                                        `json:"timeCreated,omitempty"`
	LifecycleState          string                                        `json:"lifecycleState,omitempty"`
	FreeformTags            map[string]string                             `json:"freeformTags,omitempty"`
	DefinedTags             map[string]shared.MapValue                    `json:"definedTags,omitempty"`
	PartnerId               string                                        `json:"partnerId,omitempty"`
	ParentLockboxId         string                                        `json:"parentLockboxId,omitempty"`
	PartnerCompartmentId    string                                        `json:"partnerCompartmentId,omitempty"`
	LockboxPartner          string                                        `json:"lockboxPartner,omitempty"`
	TimeUpdated             string                                        `json:"timeUpdated,omitempty"`
	AccessContextAttributes lockboxv1beta1.LockboxAccessContextAttributes `json:"accessContextAttributes,omitempty"`
	ApprovalTemplateId      string                                        `json:"approvalTemplateId,omitempty"`
	MaxAccessDuration       string                                        `json:"maxAccessDuration,omitempty"`
	LifecycleDetails        string                                        `json:"lifecycleDetails,omitempty"`
	SystemTags              map[string]shared.MapValue                    `json:"systemTags,omitempty"`
}

type lockboxProjectedResponse struct {
	Lockbox      lockboxStatusProjection `presentIn:"body"`
	OpcRequestId *string                 `presentIn:"header" name:"opc-request-id"`
}

type lockboxProjectedCollection struct {
	Items []lockboxStatusProjection `json:"items,omitempty"`
}

type lockboxProjectedListResponse struct {
	LockboxCollection lockboxProjectedCollection `presentIn:"body"`
	OpcRequestId      *string                    `presentIn:"header" name:"opc-request-id"`
	OpcNextPage       *string                    `presentIn:"header" name:"opc-next-page"`
}

func lockboxProjectedResponseFromSDK(
	current lockboxsdk.Lockbox,
	opcRequestID *string,
) lockboxProjectedResponse {
	return lockboxProjectedResponse{
		Lockbox:      lockboxStatusProjectionFromSDK(current),
		OpcRequestId: opcRequestID,
	}
}

func lockboxProjectedListResponseFromSDK(
	response lockboxsdk.ListLockboxesResponse,
) lockboxProjectedListResponse {
	projected := lockboxProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.LockboxCollection.Items = append(
			projected.LockboxCollection.Items,
			lockboxStatusProjectionFromSummary(item),
		)
	}
	return projected
}

func lockboxStatusProjectionFromSDK(current lockboxsdk.Lockbox) lockboxStatusProjection {
	return lockboxStatusProjection{
		Id:                      lockboxStringValue(current.Id),
		DisplayName:             lockboxStringValue(current.DisplayName),
		CompartmentId:           lockboxStringValue(current.CompartmentId),
		ResourceId:              lockboxStringValue(current.ResourceId),
		TimeCreated:             lockboxSDKTimeString(current.TimeCreated),
		LifecycleState:          string(current.LifecycleState),
		FreeformTags:            maps.Clone(current.FreeformTags),
		DefinedTags:             lockboxStatusTagsFromSDK(current.DefinedTags),
		PartnerId:               lockboxStringValue(current.PartnerId),
		ParentLockboxId:         lockboxStringValue(current.ParentLockboxId),
		PartnerCompartmentId:    lockboxStringValue(current.PartnerCompartmentId),
		LockboxPartner:          string(current.LockboxPartner),
		TimeUpdated:             lockboxSDKTimeString(current.TimeUpdated),
		AccessContextAttributes: lockboxAccessContextAttributesFromSDK(current.AccessContextAttributes),
		ApprovalTemplateId:      lockboxStringValue(current.ApprovalTemplateId),
		MaxAccessDuration:       lockboxStringValue(current.MaxAccessDuration),
		LifecycleDetails:        lockboxStringValue(current.LifecycleDetails),
		SystemTags:              lockboxStatusTagsFromSDK(current.SystemTags),
	}
}

func lockboxStatusProjectionFromSummary(current lockboxsdk.LockboxSummary) lockboxStatusProjection {
	return lockboxStatusProjection{
		Id:                   lockboxStringValue(current.Id),
		DisplayName:          lockboxStringValue(current.DisplayName),
		CompartmentId:        lockboxStringValue(current.CompartmentId),
		ResourceId:           lockboxStringValue(current.ResourceId),
		TimeCreated:          lockboxSDKTimeString(current.TimeCreated),
		LifecycleState:       string(current.LifecycleState),
		FreeformTags:         maps.Clone(current.FreeformTags),
		DefinedTags:          lockboxStatusTagsFromSDK(current.DefinedTags),
		PartnerId:            lockboxStringValue(current.PartnerId),
		PartnerCompartmentId: lockboxStringValue(current.PartnerCompartmentId),
		LockboxPartner:       string(current.LockboxPartner),
		TimeUpdated:          lockboxSDKTimeString(current.TimeUpdated),
		ApprovalTemplateId:   lockboxStringValue(current.ApprovalTemplateId),
		MaxAccessDuration:    lockboxStringValue(current.MaxAccessDuration),
		LifecycleDetails:     lockboxStringValue(current.LifecycleDetails),
		SystemTags:           lockboxStatusTagsFromSDK(current.SystemTags),
	}
}

func lockboxProjectionFromResponse(response any) (lockboxStatusProjection, bool) {
	response = lockboxDereferenceRuntimeBody(response)
	switch current := response.(type) {
	case lockboxProjectedResponse:
		return current.Lockbox, true
	case lockboxStatusProjection:
		return current, true
	case lockboxsdk.CreateLockboxResponse:
		return lockboxStatusProjectionFromSDK(current.Lockbox), true
	case lockboxsdk.GetLockboxResponse:
		return lockboxStatusProjectionFromSDK(current.Lockbox), true
	case lockboxsdk.UpdateLockboxResponse:
		return lockboxStatusProjectionFromSDK(current.Lockbox), true
	case lockboxsdk.Lockbox:
		return lockboxStatusProjectionFromSDK(current), true
	case lockboxsdk.LockboxSummary:
		return lockboxStatusProjectionFromSummary(current), true
	default:
		return lockboxStatusProjection{}, false
	}
}

func lockboxDereferenceRuntimeBody(response any) any {
	value := reflect.ValueOf(response)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return nil
	}
	return value.Interface()
}

func wrapLockboxDeleteConfirmation(hooks *LockboxRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}

	getLockbox := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LockboxServiceClient) LockboxServiceClient {
		return lockboxDeleteGuardClient{
			delegate:   delegate,
			getLockbox: getLockbox,
		}
	})
}

type lockboxDeleteGuardClient struct {
	delegate   LockboxServiceClient
	getLockbox func(context.Context, lockboxsdk.GetLockboxRequest) (lockboxsdk.GetLockboxResponse, error)
}

func (c lockboxDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *lockboxv1beta1.Lockbox,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("lockbox runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c lockboxDeleteGuardClient) Delete(ctx context.Context, resource *lockboxv1beta1.Lockbox) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("lockbox runtime client is not configured")
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c lockboxDeleteGuardClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *lockboxv1beta1.Lockbox,
) error {
	if c.getLockbox == nil || resource == nil {
		return nil
	}
	lockboxID := trackedLockboxID(resource)
	if lockboxID == "" {
		return nil
	}
	_, err := c.getLockbox(ctx, lockboxsdk.GetLockboxRequest{LockboxId: common.String(lockboxID)})
	if err == nil || !isAmbiguousLockboxNotFound(err) {
		return nil
	}
	err = conservativeLockboxNotFoundError(err, "pre-delete read")
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return err
}

func handleLockboxDeleteError(resource *lockboxv1beta1.Lockbox, err error) error {
	err = conservativeLockboxNotFoundError(err, "delete")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeLockboxNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("Lockbox %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousLockboxNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousLockboxNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func isAmbiguousLockboxNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousLockboxNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func trackedLockboxID(resource *lockboxv1beta1.Lockbox) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func lockboxDesiredStringUpdate(spec string, current string) (*string, bool) {
	if strings.TrimSpace(spec) == "" || spec == current {
		return nil, false
	}
	return common.String(spec), true
}

func lockboxDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil || maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func lockboxDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]shared.MapValue,
) (map[string]map[string]interface{}, bool) {
	if spec == nil || lockboxStatusTagsEqual(spec, current) {
		return nil, false
	}
	return *util.ConvertToOciDefinedTags(&spec), true
}

func lockboxStatusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}

	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = fmt.Sprint(value)
		}
		converted[namespace] = children
	}
	return converted
}

func lockboxStatusTagsEqual(left map[string]shared.MapValue, right map[string]shared.MapValue) bool {
	if len(left) != len(right) {
		return false
	}
	for namespace, leftValues := range left {
		rightValues, ok := right[namespace]
		if !ok || !maps.Equal(leftValues, rightValues) {
			return false
		}
	}
	return true
}

func lockboxOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func lockboxStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func lockboxSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02T15:04:05.999999999Z07:00")
}
