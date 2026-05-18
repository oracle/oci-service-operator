/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mediaasset

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	mediaservicessdk "github.com/oracle/oci-go-sdk/v65/mediaservices"
	mediaservicesv1beta1 "github.com/oracle/oci-service-operator/api/mediaservices/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type mediaAssetOCIClient interface {
	CreateMediaAsset(context.Context, mediaservicessdk.CreateMediaAssetRequest) (mediaservicessdk.CreateMediaAssetResponse, error)
	GetMediaAsset(context.Context, mediaservicessdk.GetMediaAssetRequest) (mediaservicessdk.GetMediaAssetResponse, error)
	ListMediaAssets(context.Context, mediaservicessdk.ListMediaAssetsRequest) (mediaservicessdk.ListMediaAssetsResponse, error)
	UpdateMediaAsset(context.Context, mediaservicessdk.UpdateMediaAssetRequest) (mediaservicessdk.UpdateMediaAssetResponse, error)
	DeleteMediaAsset(context.Context, mediaservicessdk.DeleteMediaAssetRequest) (mediaservicessdk.DeleteMediaAssetResponse, error)
}

func init() {
	registerMediaAssetRuntimeHooksMutator(func(_ *MediaAssetServiceManager, hooks *MediaAssetRuntimeHooks) {
		applyMediaAssetRuntimeHooks(hooks)
	})
}

func applyMediaAssetRuntimeHooks(hooks *MediaAssetRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedMediaAssetRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardMediaAssetExistingBeforeCreate
	hooks.ParityHooks.NormalizeDesiredState = normalizeMediaAssetDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateMediaAssetCreateOnlyDrift
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *mediaservicesv1beta1.MediaAsset,
		namespace string,
	) (any, error) {
		return buildMediaAssetCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *mediaservicesv1beta1.MediaAsset,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildMediaAssetUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = mediaAssetCreateFields()
	hooks.Get.Fields = mediaAssetGetFields()
	hooks.List.Fields = mediaAssetListFields()
	hooks.Update.Fields = mediaAssetUpdateFields()
	hooks.Delete.Fields = mediaAssetDeleteFields()
}

func newMediaAssetServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client mediaAssetOCIClient,
) MediaAssetServiceClient {
	hooks := newMediaAssetRuntimeHooksWithOCIClient(client)
	applyMediaAssetRuntimeHooks(&hooks)
	delegate := defaultMediaAssetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*mediaservicesv1beta1.MediaAsset](
			buildMediaAssetGeneratedRuntimeConfig(&MediaAssetServiceManager{Log: log}, hooks),
		),
	}
	return wrapMediaAssetGeneratedClient(hooks, delegate)
}

func newMediaAssetRuntimeHooksWithOCIClient(client mediaAssetOCIClient) MediaAssetRuntimeHooks {
	return MediaAssetRuntimeHooks{
		Create: runtimeOperationHooks[mediaservicessdk.CreateMediaAssetRequest, mediaservicessdk.CreateMediaAssetResponse]{
			Fields: mediaAssetCreateFields(),
			Call: func(ctx context.Context, request mediaservicessdk.CreateMediaAssetRequest) (mediaservicessdk.CreateMediaAssetResponse, error) {
				return client.CreateMediaAsset(ctx, request)
			},
		},
		Get: runtimeOperationHooks[mediaservicessdk.GetMediaAssetRequest, mediaservicessdk.GetMediaAssetResponse]{
			Fields: mediaAssetGetFields(),
			Call: func(ctx context.Context, request mediaservicessdk.GetMediaAssetRequest) (mediaservicessdk.GetMediaAssetResponse, error) {
				return client.GetMediaAsset(ctx, request)
			},
		},
		List: runtimeOperationHooks[mediaservicessdk.ListMediaAssetsRequest, mediaservicessdk.ListMediaAssetsResponse]{
			Fields: mediaAssetListFields(),
			Call: func(ctx context.Context, request mediaservicessdk.ListMediaAssetsRequest) (mediaservicessdk.ListMediaAssetsResponse, error) {
				return client.ListMediaAssets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[mediaservicessdk.UpdateMediaAssetRequest, mediaservicessdk.UpdateMediaAssetResponse]{
			Fields: mediaAssetUpdateFields(),
			Call: func(ctx context.Context, request mediaservicessdk.UpdateMediaAssetRequest) (mediaservicessdk.UpdateMediaAssetResponse, error) {
				return client.UpdateMediaAsset(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[mediaservicessdk.DeleteMediaAssetRequest, mediaservicessdk.DeleteMediaAssetResponse]{
			Fields: mediaAssetDeleteFields(),
			Call: func(ctx context.Context, request mediaservicessdk.DeleteMediaAssetRequest) (mediaservicessdk.DeleteMediaAssetResponse, error) {
				return client.DeleteMediaAsset(ctx, request)
			},
		},
	}
}

func reviewedMediaAssetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "mediaservices",
		FormalSlug:    "mediaasset",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(mediaservicessdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(mediaservicessdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(mediaservicessdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(mediaservicessdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(mediaservicessdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"bucketName",
				"compartmentId",
				"displayName",
				"id",
				"masterMediaAssetId",
				"mediaWorkflowJobId",
				"objectName",
				"parentMediaAssetId",
				"sourceMediaWorkflowId",
				"sourceMediaWorkflowVersion",
				"type",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"displayName",
				"freeformTags",
				"masterMediaAssetId",
				"mediaAssetTags",
				"metadata",
				"parentMediaAssetId",
				"type",
			},
			ForceNew: []string{
				"bucketName",
				"compartmentId",
				"mediaWorkflowJobId",
				"namespaceName",
				"objectEtag",
				"objectName",
				"segmentRangeEndIndex",
				"segmentRangeStartIndex",
				"sourceMediaWorkflowId",
				"sourceMediaWorkflowVersion",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "none",
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "none",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: nil,
		Unsupported:         nil,
	}
}

func mediaAssetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateMediaAssetDetails", RequestName: "CreateMediaAssetDetails", Contribution: "body"},
	}
}

func mediaAssetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MediaAssetId", RequestName: "mediaAssetId", Contribution: "path", PreferResourceID: true},
	}
}

func mediaAssetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Type", RequestName: "type", Contribution: "query", LookupPaths: []string{"status.type", "spec.type", "type"}},
		{FieldName: "BucketName", RequestName: "bucketName", Contribution: "query", LookupPaths: []string{"status.bucketName", "spec.bucketName", "bucketName"}},
		{FieldName: "ObjectName", RequestName: "objectName", Contribution: "query", LookupPaths: []string{"status.objectName", "spec.objectName", "objectName"}},
		{FieldName: "MediaWorkflowJobId", RequestName: "mediaWorkflowJobId", Contribution: "query", LookupPaths: []string{"status.mediaWorkflowJobId", "spec.mediaWorkflowJobId", "mediaWorkflowJobId"}},
		{FieldName: "SourceMediaWorkflowId", RequestName: "sourceMediaWorkflowId", Contribution: "query", LookupPaths: []string{"status.sourceMediaWorkflowId", "spec.sourceMediaWorkflowId", "sourceMediaWorkflowId"}},
		{FieldName: "SourceMediaWorkflowVersion", RequestName: "sourceMediaWorkflowVersion", Contribution: "query", LookupPaths: []string{"status.sourceMediaWorkflowVersion", "spec.sourceMediaWorkflowVersion", "sourceMediaWorkflowVersion"}},
		{FieldName: "ParentMediaAssetId", RequestName: "parentMediaAssetId", Contribution: "query", LookupPaths: []string{"status.parentMediaAssetId", "spec.parentMediaAssetId", "parentMediaAssetId"}},
		{FieldName: "MasterMediaAssetId", RequestName: "masterMediaAssetId", Contribution: "query", LookupPaths: []string{"status.masterMediaAssetId", "spec.masterMediaAssetId", "masterMediaAssetId"}},
	}
}

func mediaAssetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MediaAssetId", RequestName: "mediaAssetId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateMediaAssetDetails", RequestName: "UpdateMediaAssetDetails", Contribution: "body"},
	}
}

func mediaAssetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MediaAssetId", RequestName: "mediaAssetId", Contribution: "path", PreferResourceID: true},
	}
}

func guardMediaAssetExistingBeforeCreate(
	_ context.Context,
	resource *mediaservicesv1beta1.MediaAsset,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("MediaAsset resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("MediaAsset spec.compartmentId is required")
	}
	if strings.TrimSpace(resource.Spec.MediaWorkflowJobId) != "" && strings.TrimSpace(resource.Spec.SourceMediaWorkflowId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if resource.Spec.SourceMediaWorkflowVersion != 0 && strings.TrimSpace(resource.Spec.SourceMediaWorkflowId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if !mediaAssetHasReusableLookupIdentity(resource.Spec) {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func mediaAssetHasReusableLookupIdentity(spec mediaservicesv1beta1.MediaAssetSpec) bool {
	switch {
	case strings.TrimSpace(spec.DisplayName) != "":
		return true
	case strings.TrimSpace(spec.BucketName) != "" && strings.TrimSpace(spec.ObjectName) != "":
		return true
	case strings.TrimSpace(spec.SourceMediaWorkflowId) != "" &&
		(strings.TrimSpace(spec.MediaWorkflowJobId) != "" || spec.SourceMediaWorkflowVersion != 0):
		return true
	case strings.TrimSpace(spec.ParentMediaAssetId) != "":
		return true
	case strings.TrimSpace(spec.MasterMediaAssetId) != "":
		return true
	default:
		return false
	}
}

func buildMediaAssetCreateDetails(
	ctx context.Context,
	resource *mediaservicesv1beta1.MediaAsset,
	namespace string,
) (mediaservicessdk.CreateMediaAssetDetails, error) {
	if resource == nil {
		return mediaservicessdk.CreateMediaAssetDetails{}, fmt.Errorf("mediaasset resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return mediaservicessdk.CreateMediaAssetDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return mediaservicessdk.CreateMediaAssetDetails{}, fmt.Errorf("marshal resolved mediaasset spec: %w", err)
	}

	var details mediaservicessdk.CreateMediaAssetDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return mediaservicessdk.CreateMediaAssetDetails{}, fmt.Errorf("decode mediaasset create request body: %w", err)
	}
	for index := range details.Locks {
		details.Locks[index].TimeCreated = nil
	}

	return details, nil
}

func buildMediaAssetUpdateBody(
	resource *mediaservicesv1beta1.MediaAsset,
	currentResponse any,
) (mediaservicessdk.UpdateMediaAssetDetails, bool, error) {
	if resource == nil {
		return mediaservicessdk.UpdateMediaAssetDetails{}, false, fmt.Errorf("mediaasset resource is nil")
	}

	current, err := mediaAssetRuntimeBody(currentResponse)
	if err != nil {
		return mediaservicessdk.UpdateMediaAssetDetails{}, false, err
	}

	spec := resource.Spec
	details := mediaservicessdk.UpdateMediaAssetDetails{}
	updateNeeded := false

	if desired, ok := mediaAssetDesiredStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredTypeUpdate(spec.Type, current.Type); ok {
		details.Type = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredStringUpdate(spec.ParentMediaAssetId, current.ParentMediaAssetId); ok {
		details.ParentMediaAssetId = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredStringUpdate(spec.MasterMediaAssetId, current.MasterMediaAssetId); ok {
		details.MasterMediaAssetId = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredMetadataUpdate(spec.Metadata, current.Metadata); ok {
		details.Metadata = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredTagsUpdate(spec.MediaAssetTags, current.MediaAssetTags); ok {
		details.MediaAssetTags = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := mediaAssetDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func mediaAssetRuntimeBody(currentResponse any) (mediaservicessdk.MediaAsset, error) {
	switch current := currentResponse.(type) {
	case mediaservicessdk.MediaAsset:
		return current, nil
	case *mediaservicessdk.MediaAsset:
		if current == nil {
			return mediaservicessdk.MediaAsset{}, fmt.Errorf("current MediaAsset response is nil")
		}
		return *current, nil
	case mediaservicessdk.MediaAssetSummary:
		return mediaservicessdk.MediaAsset{
			Id:                 current.Id,
			CompartmentId:      current.CompartmentId,
			LifecycleState:     current.LifecycleState,
			Type:               current.Type,
			DisplayName:        current.DisplayName,
			TimeCreated:        current.TimeCreated,
			TimeUpdated:        current.TimeUpdated,
			MasterMediaAssetId: current.MasterMediaAssetId,
			ParentMediaAssetId: current.ParentMediaAssetId,
			FreeformTags:       current.FreeformTags,
			DefinedTags:        current.DefinedTags,
			SystemTags:         current.SystemTags,
			Locks:              current.Locks,
		}, nil
	case *mediaservicessdk.MediaAssetSummary:
		if current == nil {
			return mediaservicessdk.MediaAsset{}, fmt.Errorf("current MediaAsset response is nil")
		}
		return mediaAssetRuntimeBody(*current)
	case mediaservicessdk.CreateMediaAssetResponse:
		return current.MediaAsset, nil
	case *mediaservicessdk.CreateMediaAssetResponse:
		if current == nil {
			return mediaservicessdk.MediaAsset{}, fmt.Errorf("current MediaAsset response is nil")
		}
		return current.MediaAsset, nil
	case mediaservicessdk.GetMediaAssetResponse:
		return current.MediaAsset, nil
	case *mediaservicessdk.GetMediaAssetResponse:
		if current == nil {
			return mediaservicessdk.MediaAsset{}, fmt.Errorf("current MediaAsset response is nil")
		}
		return current.MediaAsset, nil
	case mediaservicessdk.UpdateMediaAssetResponse:
		return current.MediaAsset, nil
	case *mediaservicessdk.UpdateMediaAssetResponse:
		if current == nil {
			return mediaservicessdk.MediaAsset{}, fmt.Errorf("current MediaAsset response is nil")
		}
		return current.MediaAsset, nil
	default:
		return mediaservicessdk.MediaAsset{}, fmt.Errorf("unexpected current MediaAsset response type %T", currentResponse)
	}
}

func normalizeMediaAssetDesiredState(resource *mediaservicesv1beta1.MediaAsset, currentResponse any) {
	if resource == nil || resource.Spec.Locks == nil {
		return
	}
	current, err := mediaAssetRuntimeBody(currentResponse)
	if err != nil {
		return
	}
	if mediaAssetLocksEqual(resource.Spec.Locks, current.Locks) {
		resource.Spec.Locks = nil
	}
}

func validateMediaAssetCreateOnlyDrift(resource *mediaservicesv1beta1.MediaAsset, currentResponse any) error {
	if resource == nil || resource.Spec.Locks == nil {
		return nil
	}
	current, err := mediaAssetRuntimeBody(currentResponse)
	if err != nil {
		return err
	}
	if mediaAssetLocksEqual(resource.Spec.Locks, current.Locks) {
		return nil
	}
	return fmt.Errorf("mediaasset create-only drift detected for locks; replace the resource or restore the desired spec before update")
}

func mediaAssetDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func mediaAssetDesiredTypeUpdate(
	spec string,
	current mediaservicessdk.AssetTypeEnum,
) (mediaservicessdk.AssetTypeEnum, bool) {
	if strings.TrimSpace(spec) == "" || spec == string(current) {
		return "", false
	}
	return mediaservicessdk.AssetTypeEnum(spec), true
}

func mediaAssetDesiredMetadataUpdate(
	spec []mediaservicesv1beta1.MediaAssetMetadata,
	current []mediaservicessdk.Metadata,
) ([]mediaservicessdk.Metadata, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}

	desired := mediaAssetMetadataFromSpec(spec)
	if mediaAssetMetadataEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func mediaAssetDesiredTagsUpdate(
	spec []mediaservicesv1beta1.MediaAssetTag,
	current []mediaservicessdk.MediaAssetTag,
) ([]mediaservicessdk.MediaAssetTag, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}

	desired := mediaAssetTagsFromSpec(spec)
	if mediaAssetTagsEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func mediaAssetDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func mediaAssetDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}

	desired := mediaAssetDefinedTagsFromSpec(spec)
	if mediaAssetDefinedTagsEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func mediaAssetMetadataFromSpec(spec []mediaservicesv1beta1.MediaAssetMetadata) []mediaservicessdk.Metadata {
	converted := make([]mediaservicessdk.Metadata, 0, len(spec))
	for _, item := range spec {
		converted = append(converted, mediaservicessdk.Metadata{
			Metadata: common.String(item.Metadata),
		})
	}
	return converted
}

func mediaAssetMetadataEqual(left []mediaservicessdk.Metadata, right []mediaservicessdk.Metadata) bool {
	return slices.EqualFunc(left, right, func(left mediaservicessdk.Metadata, right mediaservicessdk.Metadata) bool {
		return mediaAssetStringValue(left.Metadata) == mediaAssetStringValue(right.Metadata)
	})
}

func mediaAssetTagsFromSpec(spec []mediaservicesv1beta1.MediaAssetTag) []mediaservicessdk.MediaAssetTag {
	converted := make([]mediaservicessdk.MediaAssetTag, 0, len(spec))
	for _, item := range spec {
		converted = append(converted, mediaservicessdk.MediaAssetTag{
			Value: common.String(item.Value),
			Type:  mediaservicessdk.MediaAssetTagTypeEnum(item.Type),
		})
	}
	return converted
}

func mediaAssetTagsEqual(left []mediaservicessdk.MediaAssetTag, right []mediaservicessdk.MediaAssetTag) bool {
	return slices.EqualFunc(left, right, func(left mediaservicessdk.MediaAssetTag, right mediaservicessdk.MediaAssetTag) bool {
		return mediaAssetStringValue(left.Value) == mediaAssetStringValue(right.Value) &&
			left.Type == right.Type
	})
}

func mediaAssetDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func mediaAssetDefinedTagsEqual(left map[string]map[string]interface{}, right map[string]map[string]interface{}) bool {
	if len(left) != len(right) {
		return false
	}
	for namespace, leftValues := range left {
		rightValues, ok := right[namespace]
		if !ok || len(leftValues) != len(rightValues) {
			return false
		}
		for key, leftValue := range leftValues {
			if fmt.Sprint(leftValue) != fmt.Sprint(rightValues[key]) {
				return false
			}
		}
	}
	return true
}

func mediaAssetLocksEqual(spec []mediaservicesv1beta1.MediaAssetLock, current []mediaservicessdk.ResourceLock) bool {
	if len(spec) != len(current) {
		return false
	}
	for index, lock := range spec {
		if lock.Type != string(current[index].Type) ||
			lock.CompartmentId != mediaAssetStringValue(current[index].CompartmentId) ||
			lock.RelatedResourceId != mediaAssetStringValue(current[index].RelatedResourceId) ||
			lock.Message != mediaAssetStringValue(current[index].Message) {
			return false
		}
	}
	return true
}

func mediaAssetStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
