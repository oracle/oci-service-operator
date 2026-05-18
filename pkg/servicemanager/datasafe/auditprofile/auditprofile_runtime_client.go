/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package auditprofile

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type auditProfileOCIClient interface {
	CreateAuditProfile(context.Context, datasafesdk.CreateAuditProfileRequest) (datasafesdk.CreateAuditProfileResponse, error)
	GetAuditProfile(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error)
	ListAuditProfiles(context.Context, datasafesdk.ListAuditProfilesRequest) (datasafesdk.ListAuditProfilesResponse, error)
	UpdateAuditProfile(context.Context, datasafesdk.UpdateAuditProfileRequest) (datasafesdk.UpdateAuditProfileResponse, error)
	DeleteAuditProfile(context.Context, datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error)
}

type auditProfileListCall func(context.Context, datasafesdk.ListAuditProfilesRequest) (datasafesdk.ListAuditProfilesResponse, error)

type auditProfileGetCall func(context.Context, datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error)

type auditProfileAuthShapedNotFound struct {
	err error
}

func (e auditProfileAuthShapedNotFound) Error() string {
	return fmt.Sprintf("datasafe AuditProfile delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e auditProfileAuthShapedNotFound) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

type auditProfileAuthShapedConfirmRead struct {
	err error
}

func (e auditProfileAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("datasafe AuditProfile delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e auditProfileAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func init() {
	registerAuditProfileRuntimeHooksMutator(func(manager *AuditProfileServiceManager, hooks *AuditProfileRuntimeHooks) {
		applyAuditProfileRuntimeHooks(manager, hooks)
	})
}

func applyAuditProfileRuntimeHooks(manager *AuditProfileServiceManager, hooks *AuditProfileRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedAuditProfileRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *datasafev1beta1.AuditProfile, namespace string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("AuditProfile resource is nil")
		}
		return generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, managerCredentialClient(manager), namespace)
	}
	hooks.Create.Fields = auditProfileCreateFields()
	hooks.Get.Fields = auditProfileGetFields()
	hooks.List.Fields = auditProfileListFields()
	hooks.Update.Fields = auditProfileUpdateFields()
	hooks.Delete.Fields = auditProfileDeleteFields()
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, _ *datasafev1beta1.AuditProfile, currentID string) (any, error) {
		return confirmAuditProfileDeleteRead(ctx, hooks.Get.Call, currentID)
	}
	hooks.DeleteHooks.HandleError = handleAuditProfileDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleAuditProfileDeleteConfirmReadOutcome
	if hooks.List.Call != nil {
		hooks.List.Call = listAuditProfilesAllPages(hooks.List.Call)
	}
}

func managerCredentialClient(manager *AuditProfileServiceManager) credhelper.CredentialClient {
	if manager == nil {
		return nil
	}
	return manager.CredentialClient
}

func newAuditProfileServiceClientWithOCIClient(client auditProfileOCIClient) AuditProfileServiceClient {
	manager := &AuditProfileServiceManager{}
	hooks := newAuditProfileRuntimeHooksWithOCIClient(client)
	applyAuditProfileRuntimeHooks(manager, &hooks)
	delegate := defaultAuditProfileServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.AuditProfile](
			buildAuditProfileGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAuditProfileGeneratedClient(hooks, delegate)
}

func newAuditProfileRuntimeHooksWithOCIClient(client auditProfileOCIClient) AuditProfileRuntimeHooks {
	return AuditProfileRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.AuditProfile]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.AuditProfile]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.AuditProfile]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.AuditProfile]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.AuditProfile]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.AuditProfile]{},
		Create: runtimeOperationHooks[datasafesdk.CreateAuditProfileRequest, datasafesdk.CreateAuditProfileResponse]{
			Fields: auditProfileCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateAuditProfileRequest) (datasafesdk.CreateAuditProfileResponse, error) {
				if client == nil {
					return datasafesdk.CreateAuditProfileResponse{}, fmt.Errorf("AuditProfile OCI client is nil")
				}
				return client.CreateAuditProfile(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetAuditProfileRequest, datasafesdk.GetAuditProfileResponse]{
			Fields: auditProfileGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetAuditProfileRequest) (datasafesdk.GetAuditProfileResponse, error) {
				if client == nil {
					return datasafesdk.GetAuditProfileResponse{}, fmt.Errorf("AuditProfile OCI client is nil")
				}
				return client.GetAuditProfile(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListAuditProfilesRequest, datasafesdk.ListAuditProfilesResponse]{
			Fields: auditProfileListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListAuditProfilesRequest) (datasafesdk.ListAuditProfilesResponse, error) {
				if client == nil {
					return datasafesdk.ListAuditProfilesResponse{}, fmt.Errorf("AuditProfile OCI client is nil")
				}
				return client.ListAuditProfiles(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateAuditProfileRequest, datasafesdk.UpdateAuditProfileResponse]{
			Fields: auditProfileUpdateFields(),
			Call: func(ctx context.Context, request datasafesdk.UpdateAuditProfileRequest) (datasafesdk.UpdateAuditProfileResponse, error) {
				if client == nil {
					return datasafesdk.UpdateAuditProfileResponse{}, fmt.Errorf("AuditProfile OCI client is nil")
				}
				return client.UpdateAuditProfile(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteAuditProfileRequest, datasafesdk.DeleteAuditProfileResponse]{
			Fields: auditProfileDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteAuditProfileRequest) (datasafesdk.DeleteAuditProfileResponse, error) {
				if client == nil {
					return datasafesdk.DeleteAuditProfileResponse{}, fmt.Errorf("AuditProfile OCI client is nil")
				}
				return client.DeleteAuditProfile(ctx, request)
			},
		},
		WrapGeneratedClient: []func(AuditProfileServiceClient) AuditProfileServiceClient{},
	}
}

func reviewedAuditProfileRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:       "datasafe",
		FormalSlug:          "auditprofile",
		StatusProjection:    "required",
		SecretSideEffects:   "none",
		FinalizerPolicy:     "retain-until-confirmed-delete",
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.AuditProfileLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.AuditProfileLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.AuditProfileLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.AuditProfileLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.AuditProfileLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "targetId", "targetType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"description",
				"displayName",
				"isPaidUsageEnabled",
				"isOverrideGlobalPaidUsage",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"targetId",
				"targetType",
				"onlineMonths",
				"offlineMonths",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AuditProfile", Action: "CreateAuditProfile"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AuditProfile", Action: "UpdateAuditProfile"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AuditProfile", Action: "DeleteAuditProfile"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AuditProfile", Action: "GetAuditProfile"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AuditProfile", Action: "GetAuditProfile"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AuditProfile", Action: "GetAuditProfile"}},
		},
	}
}

func auditProfileCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateAuditProfileDetails", RequestName: "CreateAuditProfileDetails", Contribution: "body"},
	}
}

func auditProfileGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AuditProfileId", RequestName: "auditProfileId", Contribution: "path", PreferResourceID: true},
	}
}

func auditProfileListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "AuditProfileId", RequestName: "auditProfileId", Contribution: "query", PreferResourceID: true},
		{FieldName: "TargetId", RequestName: "targetId", Contribution: "query", LookupPaths: []string{"status.targetId", "spec.targetId", "targetId"}},
		{FieldName: "TargetType", RequestName: "targetType", Contribution: "query", LookupPaths: []string{"status.targetType", "spec.targetType", "targetType"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func auditProfileUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AuditProfileId", RequestName: "auditProfileId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateAuditProfileDetails", RequestName: "UpdateAuditProfileDetails", Contribution: "body"},
	}
}

func auditProfileDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AuditProfileId", RequestName: "auditProfileId", Contribution: "path", PreferResourceID: true},
	}
}

func listAuditProfilesAllPages(next auditProfileListCall) auditProfileListCall {
	return func(ctx context.Context, request datasafesdk.ListAuditProfilesRequest) (datasafesdk.ListAuditProfilesResponse, error) {
		merged := datasafesdk.ListAuditProfilesResponse{}
		for {
			normalizeAuditProfileListTargetFilter(&request)
			response, err := next(ctx, request)
			if err != nil {
				return datasafesdk.ListAuditProfilesResponse{}, err
			}
			if merged.OpcRequestId == nil {
				merged.OpcRequestId = response.OpcRequestId
			}
			merged.Items = append(merged.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				merged.OpcNextPage = response.OpcNextPage
				return merged, nil
			}
			request.Page = response.OpcNextPage
		}
	}
}

func normalizeAuditProfileListTargetFilter(request *datasafesdk.ListAuditProfilesRequest) {
	if request == nil || request.TargetId == nil || request.TargetDatabaseGroupId != nil {
		return
	}
	if request.TargetType == datasafesdk.ListAuditProfilesTargetTypeDatabaseGroup {
		request.TargetDatabaseGroupId = common.String(strings.TrimSpace(*request.TargetId))
		request.TargetId = nil
	}
}

func confirmAuditProfileDeleteRead(ctx context.Context, get auditProfileGetCall, currentID string) (any, error) {
	if get == nil {
		return nil, fmt.Errorf("confirm AuditProfile delete: get hook is not configured")
	}
	currentID = strings.TrimSpace(currentID)
	if currentID == "" {
		return nil, fmt.Errorf("confirm AuditProfile delete: audit profile OCID is empty")
	}

	response, err := get(ctx, datasafesdk.GetAuditProfileRequest{
		AuditProfileId: common.String(currentID),
	})
	return auditProfileDeleteConfirmReadResponse(response, err)
}

func auditProfileDeleteConfirmReadResponse(response datasafesdk.GetAuditProfileResponse, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return auditProfileAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func handleAuditProfileDeleteError(resource *datasafev1beta1.AuditProfile, err error) error {
	if err == nil {
		return nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return auditProfileAuthShapedNotFound{err: err}
	}
	return err
}

func handleAuditProfileDeleteConfirmReadOutcome(
	resource *datasafev1beta1.AuditProfile,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case auditProfileAuthShapedConfirmRead:
		recordAuditProfileConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *auditProfileAuthShapedConfirmRead:
		if typed != nil {
			recordAuditProfileConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func recordAuditProfileConfirmReadRequestID(resource *datasafev1beta1.AuditProfile, err auditProfileAuthShapedConfirmRead) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}
