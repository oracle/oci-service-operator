/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package softwaresource

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type softwareSourceOCIClient interface {
	CreateSoftwareSource(context.Context, osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error)
	GetSoftwareSource(context.Context, osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error)
	ListSoftwareSources(context.Context, osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error)
	UpdateSoftwareSource(context.Context, osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error)
	DeleteSoftwareSource(context.Context, osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error)
}

type softwareSourceIdentity struct {
	compartmentID string
	displayName   string
	sourceType    string
}

type softwareSourceResourceContextKey struct{}

type softwareSourceResourceContextClient struct {
	delegate SoftwareSourceServiceClient
}

type softwareSourceAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e softwareSourceAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e softwareSourceAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSoftwareSourceRuntimeHooksMutator(func(_ *SoftwareSourceServiceManager, hooks *SoftwareSourceRuntimeHooks) {
		applySoftwareSourceRuntimeHookSettings(hooks)
	})
}

func applySoftwareSourceRuntimeHookSettings(hooks *SoftwareSourceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedSoftwareSourceRuntimeSemantics()
	hooks.Create.Fields = softwareSourceCreateFields()
	hooks.List.Fields = softwareSourceListFields()
	hooks.Update.Fields = softwareSourceUpdateFields()
	hooks.BuildCreateBody = func(_ context.Context, resource *osmanagementhubv1beta1.SoftwareSource, _ string) (any, error) {
		return buildSoftwareSourceCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *osmanagementhubv1beta1.SoftwareSource,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildSoftwareSourceUpdateBody(resource, currentResponse)
	}
	hooks.Identity.Resolve = resolveSoftwareSourceIdentity
	hooks.Identity.RecordPath = recordSoftwareSourcePathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardSoftwareSourceExistingBeforeCreate
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateSoftwareSourceCreateOnlyDriftForResponse
	wrapSoftwareSourceRequestCalls(hooks)
	if hooks.List.Call != nil {
		hooks.List.Call = listSoftwareSourcesAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleSoftwareSourceDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SoftwareSourceServiceClient) SoftwareSourceServiceClient {
		return softwareSourceResourceContextClient{delegate: delegate}
	})
}

func reviewedSoftwareSourceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "osmanagementhub",
		FormalSlug:        "softwaresource",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(osmanagementhubsdk.SoftwareSourceLifecycleStateCreating)},
			UpdatingStates:     []string{string(osmanagementhubsdk.SoftwareSourceLifecycleStateUpdating)},
			ActiveStates: []string{
				string(osmanagementhubsdk.SoftwareSourceLifecycleStateActive),
				string(osmanagementhubsdk.SoftwareSourceLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(osmanagementhubsdk.SoftwareSourceLifecycleStateDeleting)},
			TerminalStates: []string{string(osmanagementhubsdk.SoftwareSourceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "softwareSourceType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
				"vendorSoftwareSources",
				"customSoftwareSourceFilter",
				"isAutomaticallyUpdated",
				"isAutoResolveDependencies",
				"isLatestContentOnly",
				"url",
				"gpgKeyUrl",
				"isGpgCheckEnabled",
				"isSslVerifyEnabled",
				"advancedRepoOptions",
				"isMirrorSyncAllowed",
			},
			ForceNew: []string{
				"compartmentId",
				"softwareSourceType",
				"originSoftwareSourceId",
				"osFamily",
				"archType",
				"softwareSourceVersion",
				"softwareSourceSubType",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func softwareSourceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpcRequestId", RequestName: "opcRequestId", Contribution: "header"},
	}
}

func softwareSourceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func softwareSourceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SoftwareSourceId", RequestName: "softwareSourceId", Contribution: "path", PreferResourceID: true},
	}
}

func wrapSoftwareSourceRequestCalls(hooks *SoftwareSourceRuntimeHooks) {
	wrapSoftwareSourceCreateCall(hooks)
	wrapSoftwareSourceListCall(hooks)
	wrapSoftwareSourceUpdateCall(hooks)
	wrapSoftwareSourceDeleteCall(hooks)
}

func wrapSoftwareSourceCreateCall(hooks *SoftwareSourceRuntimeHooks) {
	createSoftwareSource := hooks.Create.Call
	if createSoftwareSource == nil {
		return
	}
	hooks.Create.Call = func(ctx context.Context, request osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error) {
		resource, err := softwareSourceResourceFromContext(ctx)
		if err != nil {
			return osmanagementhubsdk.CreateSoftwareSourceResponse{}, err
		}
		details, err := createSoftwareSourceDetails(resource)
		if err != nil {
			return osmanagementhubsdk.CreateSoftwareSourceResponse{}, err
		}
		request.CreateSoftwareSourceDetails = details
		return createSoftwareSource(ctx, request)
	}
}

func createSoftwareSourceDetails(resource *osmanagementhubv1beta1.SoftwareSource) (osmanagementhubsdk.CreateSoftwareSourceDetails, error) {
	body, err := buildSoftwareSourceCreateBody(resource)
	if err != nil {
		return nil, err
	}
	details, ok := body.(osmanagementhubsdk.CreateSoftwareSourceDetails)
	if !ok {
		return nil, fmt.Errorf("build SoftwareSource create body: %T does not implement CreateSoftwareSourceDetails", body)
	}
	return details, nil
}

func wrapSoftwareSourceListCall(hooks *SoftwareSourceRuntimeHooks) {
	listSoftwareSources := hooks.List.Call
	if listSoftwareSources == nil {
		return
	}
	hooks.List.Call = func(ctx context.Context, request osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
		sourceType, err := softwareSourceTypeFromContext(ctx)
		if err != nil {
			return osmanagementhubsdk.ListSoftwareSourcesResponse{}, err
		}
		return listSoftwareSources(ctx, softwareSourceListRequestWithType(request, sourceType))
	}
}

func softwareSourceListRequestWithType(
	request osmanagementhubsdk.ListSoftwareSourcesRequest,
	sourceType string,
) osmanagementhubsdk.ListSoftwareSourcesRequest {
	if sourceType != "" && len(request.SoftwareSourceType) == 0 {
		request.SoftwareSourceType = []osmanagementhubsdk.SoftwareSourceTypeEnum{osmanagementhubsdk.SoftwareSourceTypeEnum(sourceType)}
	}
	return request
}

func wrapSoftwareSourceUpdateCall(hooks *SoftwareSourceRuntimeHooks) {
	updateSoftwareSource := hooks.Update.Call
	if updateSoftwareSource == nil {
		return
	}
	hooks.Update.Call = func(ctx context.Context, request osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
		resource, err := softwareSourceResourceFromContext(ctx)
		if err != nil {
			return osmanagementhubsdk.UpdateSoftwareSourceResponse{}, err
		}
		details, err := updateSoftwareSourceDetails(resource)
		if err != nil {
			return osmanagementhubsdk.UpdateSoftwareSourceResponse{}, err
		}
		request.UpdateSoftwareSourceDetails = details
		return updateSoftwareSource(ctx, request)
	}
}

func updateSoftwareSourceDetails(resource *osmanagementhubv1beta1.SoftwareSource) (osmanagementhubsdk.UpdateSoftwareSourceDetails, error) {
	body, updateNeeded, err := buildSoftwareSourceUpdateBody(resource, nil)
	if err != nil {
		return nil, err
	}
	if !updateNeeded {
		return nil, fmt.Errorf("build SoftwareSource update body: no mutable update body produced")
	}
	details, ok := body.(osmanagementhubsdk.UpdateSoftwareSourceDetails)
	if !ok {
		return nil, fmt.Errorf("build SoftwareSource update body: %T does not implement UpdateSoftwareSourceDetails", body)
	}
	return details, nil
}

func wrapSoftwareSourceDeleteCall(hooks *SoftwareSourceRuntimeHooks) {
	deleteSoftwareSource := hooks.Delete.Call
	getSoftwareSource := hooks.Get.Call
	if deleteSoftwareSource == nil || getSoftwareSource == nil {
		return
	}
	hooks.Delete.Call = func(ctx context.Context, request osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error) {
		if strings.TrimSpace(softwareSourceStringPtrValue(request.SoftwareSourceId)) != "" {
			if _, err := getSoftwareSource(ctx, osmanagementhubsdk.GetSoftwareSourceRequest{
				SoftwareSourceId: request.SoftwareSourceId,
			}); err != nil {
				return osmanagementhubsdk.DeleteSoftwareSourceResponse{}, err
			}
		}
		return deleteSoftwareSource(ctx, request)
	}
}

func softwareSourceStringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (c softwareSourceResourceContextClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.SoftwareSource,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("SoftwareSource runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(context.WithValue(ctx, softwareSourceResourceContextKey{}, resource), resource, req)
}

func (c softwareSourceResourceContextClient) Delete(
	ctx context.Context,
	resource *osmanagementhubv1beta1.SoftwareSource,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("SoftwareSource runtime client is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func softwareSourceResourceFromContext(ctx context.Context) (*osmanagementhubv1beta1.SoftwareSource, error) {
	if ctx == nil {
		return nil, fmt.Errorf("SoftwareSource runtime context is nil")
	}
	resource, ok := ctx.Value(softwareSourceResourceContextKey{}).(*osmanagementhubv1beta1.SoftwareSource)
	if !ok || resource == nil {
		return nil, fmt.Errorf("SoftwareSource runtime resource missing from context")
	}
	return resource, nil
}

func buildSoftwareSourceCreateBody(resource *osmanagementhubv1beta1.SoftwareSource) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("SoftwareSource resource is nil")
	}
	values, err := softwareSourceCreateValues(resource.Spec)
	if err != nil {
		return nil, err
	}
	if err := validateSoftwareSourceCreateValues(values); err != nil {
		return nil, err
	}
	return decodeSoftwareSourceCreateDetails(values)
}

func buildSoftwareSourceUpdateBody(
	resource *osmanagementhubv1beta1.SoftwareSource,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("SoftwareSource resource is nil")
	}
	sourceType, err := softwareSourceUpdateType(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}
	if err := validateSoftwareSourceCreateOnlyDriftForResponse(resource, currentResponse); err != nil {
		return nil, false, err
	}
	values, err := softwareSourceUpdateValues(resource.Spec, sourceType)
	if err != nil {
		return nil, false, err
	}
	values = filterSoftwareSourceUpdateValues(values, sourceType)
	if !softwareSourceHasUpdateFields(values) {
		return nil, false, nil
	}
	updateNeeded, err := softwareSourceUpdateValuesDiffer(values, currentResponse)
	if err != nil {
		return nil, false, err
	}
	if !updateNeeded {
		return nil, false, nil
	}
	body, err := decodeSoftwareSourceUpdateDetails(values)
	if err != nil {
		return nil, false, err
	}
	return body, true, nil
}

func softwareSourceCreateValues(spec osmanagementhubv1beta1.SoftwareSourceSpec) (map[string]any, error) {
	values, err := softwareSourceJSONDataValues(spec.JsonData)
	if err != nil {
		return nil, err
	}
	if err := validateSoftwareSourceJSONDataIdentity(spec, values); err != nil {
		return nil, err
	}
	applySoftwareSourceCreateDefaults(values, spec)
	if err := normalizeSoftwareSourceTypeValue(values, true); err != nil {
		return nil, err
	}
	return values, nil
}

func softwareSourceUpdateValues(spec osmanagementhubv1beta1.SoftwareSourceSpec, sourceType string) (map[string]any, error) {
	values, err := softwareSourceJSONDataValues(spec.JsonData)
	if err != nil {
		return nil, err
	}
	if err := validateSoftwareSourceJSONDataIdentity(spec, values); err != nil {
		return nil, err
	}
	applySoftwareSourceUpdateDefaults(values, spec, sourceType)
	values["softwareSourceType"] = sourceType
	return values, nil
}

func applySoftwareSourceCreateDefaults(values map[string]any, spec osmanagementhubv1beta1.SoftwareSourceSpec) {
	applySoftwareSourceCommonDefaults(values, spec)
	switch softwareSourceCreateDefaultType(values) {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		applySoftwareSourceCustomCreateDefaults(values, spec)
	case string(osmanagementhubsdk.SoftwareSourceTypeVendor):
		putStringDefault(values, "originSoftwareSourceId", spec.OriginSoftwareSourceId)
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate), string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		applySoftwareSourcePrivateCreateDefaults(values, spec)
	case string(osmanagementhubsdk.SoftwareSourceTypeVersioned):
		applySoftwareSourceVersionedCreateDefaults(values, spec)
	}
}

func applySoftwareSourceCommonDefaults(values map[string]any, spec osmanagementhubv1beta1.SoftwareSourceSpec) {
	putStringDefault(values, "displayName", spec.DisplayName)
	putStringDefault(values, "description", spec.Description)
	putStringDefault(values, "compartmentId", spec.CompartmentId)
	putStringDefault(values, "softwareSourceType", spec.SoftwareSourceType)
	putStringMapDefault(values, "freeformTags", spec.FreeformTags)
	putDefinedTagsDefault(values, spec.DefinedTags)
}

func softwareSourceCreateDefaultType(values map[string]any) string {
	sourceType, err := softwareSourceTypeFromValues(values, false)
	if err != nil {
		return ""
	}
	return sourceType
}

func applySoftwareSourceCustomCreateDefaults(values map[string]any, spec osmanagementhubv1beta1.SoftwareSourceSpec) {
	putVendorSoftwareSourcesDefault(values, spec.VendorSoftwareSources)
	putJSONDefault(values, "customSoftwareSourceFilter", spec.CustomSoftwareSourceFilter)
	putBoolDefault(values, "isAutomaticallyUpdated", spec.IsAutomaticallyUpdated)
	putBoolDefault(values, "isAutoResolveDependencies", spec.IsAutoResolveDependencies)
	putBoolDefault(values, "isCreatedFromPackageList", spec.IsCreatedFromPackageList)
	putBoolDefault(values, "isLatestContentOnly", spec.IsLatestContentOnly)
	putStringSliceDefault(values, "packages", spec.Packages)
	putStringDefault(values, "softwareSourceSubType", spec.SoftwareSourceSubType)
}

func applySoftwareSourcePrivateCreateDefaults(values map[string]any, spec osmanagementhubv1beta1.SoftwareSourceSpec) {
	putStringDefault(values, "url", spec.Url)
	putStringDefault(values, "gpgKeyUrl", spec.GpgKeyUrl)
	putBoolDefault(values, "isGpgCheckEnabled", spec.IsGpgCheckEnabled)
	putBoolDefault(values, "isSslVerifyEnabled", spec.IsSslVerifyEnabled)
	putStringDefault(values, "advancedRepoOptions", spec.AdvancedRepoOptions)
	putBoolDefault(values, "isMirrorSyncAllowed", spec.IsMirrorSyncAllowed)
	putStringDefault(values, "osFamily", spec.OsFamily)
	putStringDefault(values, "archType", spec.ArchType)
}

func applySoftwareSourceVersionedCreateDefaults(values map[string]any, spec osmanagementhubv1beta1.SoftwareSourceSpec) {
	putVendorSoftwareSourcesDefault(values, spec.VendorSoftwareSources)
	putJSONDefault(values, "customSoftwareSourceFilter", spec.CustomSoftwareSourceFilter)
	putBoolDefault(values, "isAutoResolveDependencies", spec.IsAutoResolveDependencies)
	putBoolDefault(values, "isCreatedFromPackageList", spec.IsCreatedFromPackageList)
	putBoolDefault(values, "isLatestContentOnly", spec.IsLatestContentOnly)
	putStringSliceDefault(values, "packages", spec.Packages)
	putStringDefault(values, "softwareSourceSubType", spec.SoftwareSourceSubType)
	putStringDefault(values, "softwareSourceVersion", spec.SoftwareSourceVersion)
}

func applySoftwareSourceUpdateDefaults(
	values map[string]any,
	spec osmanagementhubv1beta1.SoftwareSourceSpec,
	sourceType string,
) {
	putStringDefault(values, "displayName", spec.DisplayName)
	putStringDefault(values, "description", spec.Description)
	putStringMapDefault(values, "freeformTags", spec.FreeformTags)
	putDefinedTagsDefault(values, spec.DefinedTags)

	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		putVendorSoftwareSourcesDefault(values, spec.VendorSoftwareSources)
		putJSONDefault(values, "customSoftwareSourceFilter", spec.CustomSoftwareSourceFilter)
		putBoolDefault(values, "isAutomaticallyUpdated", spec.IsAutomaticallyUpdated)
		putBoolDefault(values, "isAutoResolveDependencies", spec.IsAutoResolveDependencies)
		putBoolDefault(values, "isLatestContentOnly", spec.IsLatestContentOnly)
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate), string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		putStringDefault(values, "url", spec.Url)
		putStringDefault(values, "gpgKeyUrl", spec.GpgKeyUrl)
		putBoolDefault(values, "isGpgCheckEnabled", spec.IsGpgCheckEnabled)
		putBoolDefault(values, "isSslVerifyEnabled", spec.IsSslVerifyEnabled)
		putStringDefault(values, "advancedRepoOptions", spec.AdvancedRepoOptions)
		putBoolDefault(values, "isMirrorSyncAllowed", spec.IsMirrorSyncAllowed)
	}
}

func validateSoftwareSourceCreateValues(values map[string]any) error {
	sourceType, err := softwareSourceTypeFromValues(values, true)
	if err != nil {
		return err
	}

	var missing []string
	requireStringValue(values, "compartmentId", &missing)
	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		requireSliceValue(values, "vendorSoftwareSources", &missing)
	case string(osmanagementhubsdk.SoftwareSourceTypeVendor):
		requireStringValue(values, "originSoftwareSourceId", &missing)
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate), string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		requireStringValue(values, "url", &missing)
		requireStringValue(values, "osFamily", &missing)
		requireStringValue(values, "archType", &missing)
	case string(osmanagementhubsdk.SoftwareSourceTypeVersioned):
		requireSliceValue(values, "vendorSoftwareSources", &missing)
		requireStringValue(values, "softwareSourceVersion", &missing)
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("SoftwareSource spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func resolveSoftwareSourceIdentity(resource *osmanagementhubv1beta1.SoftwareSource) (any, error) {
	if resource == nil {
		return softwareSourceIdentity{}, fmt.Errorf("SoftwareSource resource is nil")
	}
	sourceType, err := softwareSourceTypeForResource(resource, false)
	if err != nil {
		return softwareSourceIdentity{}, err
	}
	if sourceType != "" && strings.TrimSpace(resource.Spec.SoftwareSourceType) != sourceType {
		resource.Spec.SoftwareSourceType = sourceType
	}
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(resource.Status.DisplayName)
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		compartmentID = strings.TrimSpace(resource.Status.CompartmentId)
	}
	return softwareSourceIdentity{
		compartmentID: compartmentID,
		displayName:   displayName,
		sourceType:    sourceType,
	}, nil
}

func recordSoftwareSourcePathIdentity(resource *osmanagementhubv1beta1.SoftwareSource, identity any) {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	typed, ok := identity.(softwareSourceIdentity)
	if !ok {
		return
	}
	if resource.Status.CompartmentId == "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if resource.Status.DisplayName == "" {
		resource.Status.DisplayName = typed.displayName
	}
	if resource.Status.SoftwareSourceType == "" {
		resource.Status.SoftwareSourceType = typed.sourceType
	}
}

func guardSoftwareSourceExistingBeforeCreate(
	_ context.Context,
	resource *osmanagementhubv1beta1.SoftwareSource,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveSoftwareSourceIdentity(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	typed := identity.(softwareSourceIdentity)
	if typed.compartmentID == "" || typed.displayName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateSoftwareSourceCreateOnlyDriftForResponse(
	resource *osmanagementhubv1beta1.SoftwareSource,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("SoftwareSource resource is nil")
	}
	if currentResponse == nil {
		return nil
	}
	current, ok := softwareSourceResponseValues(currentResponse)
	if !ok {
		return fmt.Errorf("current SoftwareSource response does not expose a SoftwareSource body")
	}
	desired, err := softwareSourceCreateValues(resource.Spec)
	if err != nil {
		return err
	}
	sourceType, err := softwareSourceTypeFromValues(desired, true)
	if err != nil {
		return err
	}
	drift := softwareSourceCreateOnlyDriftFields(desired, current, sourceType)
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("SoftwareSource create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func softwareSourceCreateOnlyDriftFields(desired map[string]any, current map[string]any, sourceType string) []string {
	var drift []string
	for _, field := range softwareSourceCreateOnlyFields(sourceType) {
		if softwareSourceCreateOnlyFieldDrifted(desired, current, field) {
			drift = append(drift, field)
		}
	}
	return drift
}

func softwareSourceCreateOnlyFieldDrifted(desired map[string]any, current map[string]any, field string) bool {
	desiredValue, desiredOK := desired[field]
	currentValue, currentOK := current[field]
	if desiredOK && softwareSourceFalseBoolMatchesEmptyReadback(desiredValue, currentValue, currentOK) {
		return false
	}
	return desiredOK && currentOK && !softwareSourceJSONEqual(desiredValue, currentValue)
}

func softwareSourceCreateOnlyFields(sourceType string) []string {
	fields := []string{"compartmentId", "softwareSourceType"}
	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		fields = append(fields, "isCreatedFromPackageList", "packages", "softwareSourceSubType")
	case string(osmanagementhubsdk.SoftwareSourceTypeVendor):
		fields = append(fields, "originSoftwareSourceId")
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate), string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		fields = append(fields, "osFamily", "archType")
	case string(osmanagementhubsdk.SoftwareSourceTypeVersioned):
		fields = append(fields,
			"vendorSoftwareSources",
			"customSoftwareSourceFilter",
			"isAutoResolveDependencies",
			"isCreatedFromPackageList",
			"isLatestContentOnly",
			"packages",
			"softwareSourceSubType",
			"softwareSourceVersion",
		)
	}
	return fields
}

func decodeSoftwareSourceCreateDetails(values map[string]any) (osmanagementhubsdk.CreateSoftwareSourceDetails, error) {
	sourceType, err := softwareSourceTypeFromValues(values, true)
	if err != nil {
		return nil, err
	}
	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		var details osmanagementhubsdk.CreateCustomSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypeVendor):
		var details osmanagementhubsdk.CreateVendorSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate):
		var details osmanagementhubsdk.CreatePrivateSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypeVersioned):
		var details osmanagementhubsdk.CreateVersionedCustomSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		var details osmanagementhubsdk.CreateThirdPartySoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	default:
		return nil, fmt.Errorf("unsupported SoftwareSource softwareSourceType %q", sourceType)
	}
}

func decodeSoftwareSourceUpdateDetails(values map[string]any) (osmanagementhubsdk.UpdateSoftwareSourceDetails, error) {
	sourceType, err := softwareSourceTypeFromValues(values, true)
	if err != nil {
		return nil, err
	}
	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		var details osmanagementhubsdk.UpdateCustomSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypeVendor):
		var details osmanagementhubsdk.UpdateVendorSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate):
		var details osmanagementhubsdk.UpdatePrivateSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypeVersioned):
		var details osmanagementhubsdk.UpdateVersionedCustomSoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	case string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		var details osmanagementhubsdk.UpdateThirdPartySoftwareSourceDetails
		return details, decodeSoftwareSourceDetails(values, &details)
	default:
		return nil, fmt.Errorf("unsupported SoftwareSource softwareSourceType %q", sourceType)
	}
}

func decodeSoftwareSourceDetails(values map[string]any, target any) error {
	payload, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal SoftwareSource details: %w", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode SoftwareSource details: %w", err)
	}
	if validator, ok := target.(interface{ ValidateEnumValue() (bool, error) }); ok {
		if _, err := validator.ValidateEnumValue(); err != nil {
			return err
		}
	}
	return nil
}

func listSoftwareSourcesAllPages(
	call func(context.Context, osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error),
) func(context.Context, osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
	return func(ctx context.Context, request osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
		return collectSoftwareSourceListPages(ctx, request, call)
	}
}

func collectSoftwareSourceListPages(
	ctx context.Context,
	request osmanagementhubsdk.ListSoftwareSourcesRequest,
	call func(context.Context, osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error),
) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
	var combined osmanagementhubsdk.ListSoftwareSourcesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return osmanagementhubsdk.ListSoftwareSourcesResponse{}, err
		}
		appendSoftwareSourceListPage(&combined, response)
		nextPage := softwareSourceNextPage(response)
		if nextPage == nil {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = nextPage
		combined.OpcNextPage = nextPage
	}
}

func appendSoftwareSourceListPage(
	combined *osmanagementhubsdk.ListSoftwareSourcesResponse,
	response osmanagementhubsdk.ListSoftwareSourcesResponse,
) {
	combined.RawResponse = response.RawResponse
	combined.OpcRequestId = response.OpcRequestId
	combined.Items = append(combined.Items, activeSoftwareSourceListItems(response.Items)...)
}

func activeSoftwareSourceListItems(items []osmanagementhubsdk.SoftwareSourceSummary) []osmanagementhubsdk.SoftwareSourceSummary {
	active := make([]osmanagementhubsdk.SoftwareSourceSummary, 0, len(items))
	for _, item := range items {
		if softwareSourceListItemActive(item) {
			active = append(active, item)
		}
	}
	return active
}

func softwareSourceListItemActive(item osmanagementhubsdk.SoftwareSourceSummary) bool {
	return item != nil && item.GetLifecycleState() != osmanagementhubsdk.SoftwareSourceLifecycleStateDeleted
}

func softwareSourceNextPage(response osmanagementhubsdk.ListSoftwareSourcesResponse) *string {
	if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
		return nil
	}
	return response.OpcNextPage
}

func handleSoftwareSourceDeleteError(resource *osmanagementhubv1beta1.SoftwareSource, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return softwareSourceAmbiguousNotFoundError{
		message:      fmt.Sprintf("SoftwareSource delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func softwareSourceTypeFromContext(ctx context.Context) (string, error) {
	resource, err := softwareSourceResourceFromContext(ctx)
	if err != nil {
		return "", nil
	}
	return softwareSourceTypeForResource(resource, false)
}

func softwareSourceTypeForResource(resource *osmanagementhubv1beta1.SoftwareSource, required bool) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("SoftwareSource resource is nil")
	}
	if sourceType := strings.TrimSpace(resource.Spec.SoftwareSourceType); sourceType != "" {
		return normalizeSoftwareSourceType(sourceType, required)
	}
	if sourceType := softwareSourceTypeFromJSONData(resource.Spec.JsonData); sourceType != "" {
		return normalizeSoftwareSourceType(sourceType, required)
	}
	if sourceType := strings.TrimSpace(resource.Status.SoftwareSourceType); sourceType != "" {
		return normalizeSoftwareSourceType(sourceType, required)
	}
	return normalizeSoftwareSourceType("", required)
}

func softwareSourceUpdateType(resource *osmanagementhubv1beta1.SoftwareSource, currentResponse any) (string, error) {
	if sourceType, err := softwareSourceTypeForResource(resource, false); err != nil || sourceType != "" {
		return sourceType, err
	}
	if values, ok := softwareSourceResponseValues(currentResponse); ok {
		return softwareSourceTypeFromValues(values, true)
	}
	return "", fmt.Errorf("SoftwareSource softwareSourceType is required for update")
}

func softwareSourceTypeFromJSONData(raw string) string {
	values, err := softwareSourceJSONDataValues(raw)
	if err != nil {
		return ""
	}
	sourceType, _ := softwareSourceStringValue(values["softwareSourceType"])
	return sourceType
}

func softwareSourceTypeFromValues(values map[string]any, required bool) (string, error) {
	sourceType, _ := softwareSourceStringValue(values["softwareSourceType"])
	normalized, err := normalizeSoftwareSourceType(sourceType, required)
	if err != nil {
		return "", err
	}
	if normalized != "" {
		values["softwareSourceType"] = normalized
	}
	return normalized, nil
}

func normalizeSoftwareSourceTypeValue(values map[string]any, required bool) error {
	_, err := softwareSourceTypeFromValues(values, required)
	return err
}

func normalizeSoftwareSourceType(sourceType string, required bool) (string, error) {
	sourceType = strings.ToUpper(strings.TrimSpace(sourceType))
	if sourceType == "" {
		if required {
			return "", fmt.Errorf("SoftwareSource softwareSourceType is required")
		}
		return "", nil
	}
	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom),
		string(osmanagementhubsdk.SoftwareSourceTypeVendor),
		string(osmanagementhubsdk.SoftwareSourceTypePrivate),
		string(osmanagementhubsdk.SoftwareSourceTypeVersioned),
		string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		return sourceType, nil
	default:
		return "", fmt.Errorf("unsupported SoftwareSource softwareSourceType %q", sourceType)
	}
}

func softwareSourceJSONDataValues(raw string) (map[string]any, error) {
	values := map[string]any{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return values, nil
	}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode SoftwareSource spec.jsonData: %w", err)
	}
	delete(values, "jsonData")
	return values, nil
}

func validateSoftwareSourceJSONDataIdentity(
	spec osmanagementhubv1beta1.SoftwareSourceSpec,
	values map[string]any,
) error {
	var conflicts []string
	if hasConflictingString(values, "compartmentId", spec.CompartmentId, false) {
		conflicts = append(conflicts, "compartmentId")
	}
	if hasConflictingString(values, "displayName", spec.DisplayName, false) {
		conflicts = append(conflicts, "displayName")
	}
	if hasConflictingString(values, "softwareSourceType", spec.SoftwareSourceType, true) {
		conflicts = append(conflicts, "softwareSourceType")
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("SoftwareSource spec.jsonData identity conflicts with spec field(s): %s", strings.Join(conflicts, ", "))
}

func hasConflictingString(values map[string]any, key string, typedValue string, caseInsensitive bool) bool {
	typedValue = strings.TrimSpace(typedValue)
	if typedValue == "" {
		return false
	}
	raw, ok := softwareSourceStringValue(values[key])
	if !ok || raw == "" {
		return false
	}
	if caseInsensitive {
		return !strings.EqualFold(raw, typedValue)
	}
	return raw != typedValue
}

func filterSoftwareSourceUpdateValues(values map[string]any, sourceType string) map[string]any {
	allowed := map[string]bool{"softwareSourceType": true}
	for _, field := range softwareSourceUpdateFieldsForType(sourceType) {
		allowed[field] = true
	}
	filtered := make(map[string]any, len(values))
	for key, value := range values {
		if allowed[key] {
			filtered[key] = value
		}
	}
	return filtered
}

func softwareSourceUpdateFieldsForType(sourceType string) []string {
	commonFields := []string{"displayName", "description", "freeformTags", "definedTags"}
	switch sourceType {
	case string(osmanagementhubsdk.SoftwareSourceTypeCustom):
		return append(commonFields,
			"vendorSoftwareSources",
			"customSoftwareSourceFilter",
			"isAutomaticallyUpdated",
			"isAutoResolveDependencies",
			"isLatestContentOnly",
		)
	case string(osmanagementhubsdk.SoftwareSourceTypePrivate), string(osmanagementhubsdk.SoftwareSourceTypeThirdParty):
		return append(commonFields,
			"url",
			"gpgKeyUrl",
			"isGpgCheckEnabled",
			"isSslVerifyEnabled",
			"advancedRepoOptions",
			"isMirrorSyncAllowed",
		)
	default:
		return commonFields
	}
}

func softwareSourceHasUpdateFields(values map[string]any) bool {
	for key := range values {
		if key != "softwareSourceType" {
			return true
		}
	}
	return false
}

func softwareSourceUpdateValuesDiffer(values map[string]any, currentResponse any) (bool, error) {
	current, ok := softwareSourceResponseValues(currentResponse)
	if !ok {
		return true, nil
	}
	for key, desired := range values {
		if key == "softwareSourceType" {
			continue
		}
		currentValue, currentOK := current[key]
		if softwareSourceFalseBoolMatchesEmptyReadback(desired, currentValue, currentOK) {
			continue
		}
		if !currentOK || !softwareSourceJSONEqual(desired, currentValue) {
			return true, nil
		}
	}
	return false, nil
}

func softwareSourceFalseBoolMatchesEmptyReadback(desired any, current any, currentOK bool) bool {
	desiredBool, ok := desired.(bool)
	if !ok || desiredBool {
		return false
	}
	return !currentOK || current == nil
}

func softwareSourceResponseValues(response any) (map[string]any, bool) {
	body, ok := softwareSourceResponseBody(response)
	if !ok || body == nil {
		return nil, false
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, false
	}
	return values, true
}

func softwareSourceResponseBody(response any) (any, bool) {
	switch typed := response.(type) {
	case osmanagementhubsdk.CreateSoftwareSourceResponse:
		return typed.SoftwareSource, typed.SoftwareSource != nil
	case osmanagementhubsdk.GetSoftwareSourceResponse:
		return typed.SoftwareSource, typed.SoftwareSource != nil
	case osmanagementhubsdk.UpdateSoftwareSourceResponse:
		return typed.SoftwareSource, typed.SoftwareSource != nil
	case osmanagementhubsdk.SoftwareSource:
		return typed, typed != nil
	case osmanagementhubsdk.SoftwareSourceSummary:
		return typed, typed != nil
	default:
		return softwareSourceReflectResponseBody(response)
	}
}

func softwareSourceReflectResponseBody(response any) (any, bool) {
	value, ok := indirectSoftwareSourceReflectValue(response)
	if !ok {
		return nil, false
	}
	if value.Kind() == reflect.Struct && strings.HasSuffix(value.Type().Name(), "Response") {
		return softwareSourcePresentBodyField(value)
	}
	return response, true
}

func indirectSoftwareSourceReflectValue(response any) (reflect.Value, bool) {
	value := reflect.ValueOf(response)
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func softwareSourcePresentBodyField(value reflect.Value) (any, bool) {
	for i := 0; i < value.NumField(); i++ {
		fieldType := value.Type().Field(i)
		if !fieldType.IsExported() || fieldType.Tag.Get("presentIn") != "body" {
			continue
		}
		return softwareSourceBodyFieldValue(value.Field(i))
	}
	return nil, false
}

func softwareSourceBodyFieldValue(field reflect.Value) (any, bool) {
	if field.Kind() == reflect.Interface && field.IsNil() {
		return nil, false
	}
	return field.Interface(), true
}

func softwareSourceJSONEqual(left any, right any) bool {
	left = normalizeSoftwareSourceJSONValue(left)
	right = normalizeSoftwareSourceJSONValue(right)
	return reflect.DeepEqual(left, right)
}

func normalizeSoftwareSourceJSONValue(value any) any {
	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return value
	}
	return decoded
}

func putStringDefault(values map[string]any, key string, value string) {
	if _, ok := values[key]; ok {
		return
	}
	value = strings.TrimSpace(value)
	if value != "" {
		values[key] = value
	}
}

func putBoolDefault(values map[string]any, key string, value bool) {
	if _, ok := values[key]; ok {
		return
	}
	values[key] = value
}

func putStringMapDefault(values map[string]any, key string, value map[string]string) {
	if _, ok := values[key]; ok || value == nil {
		return
	}
	cloned := make(map[string]string, len(value))
	for k, v := range value {
		cloned[k] = v
	}
	values[key] = cloned
}

func putDefinedTagsDefault(values map[string]any, tags map[string]shared.MapValue) {
	if _, ok := values["definedTags"]; ok || tags == nil {
		return
	}
	values["definedTags"] = *util.ConvertToOciDefinedTags(&tags)
}

func putStringSliceDefault(values map[string]any, key string, value []string) {
	if _, ok := values[key]; ok || len(value) == 0 {
		return
	}
	values[key] = append([]string(nil), value...)
}

func putVendorSoftwareSourcesDefault(values map[string]any, value []osmanagementhubv1beta1.SoftwareSourceVendorSoftwareSource) {
	if _, ok := values["vendorSoftwareSources"]; ok || len(value) == 0 {
		return
	}
	putJSONDefault(values, "vendorSoftwareSources", value)
}

func putJSONDefault(values map[string]any, key string, value any) {
	if _, ok := values[key]; ok {
		return
	}
	normalized := normalizeSoftwareSourceJSONValue(value)
	if !softwareSourceMeaningfulJSONValue(normalized) {
		return
	}
	values[key] = normalized
}

func softwareSourceMeaningfulJSONValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case map[string]any:
		return len(typed) > 0
	case []any:
		return len(typed) > 0
	default:
		return true
	}
}

func requireStringValue(values map[string]any, key string, missing *[]string) {
	value, ok := softwareSourceStringValue(values[key])
	if !ok || value == "" {
		*missing = append(*missing, key)
	}
}

func requireSliceValue(values map[string]any, key string, missing *[]string) {
	value, ok := values[key]
	if !ok {
		*missing = append(*missing, key)
		return
	}
	switch typed := value.(type) {
	case []any:
		if len(typed) == 0 {
			*missing = append(*missing, key)
		}
	case []string:
		if len(typed) == 0 {
			*missing = append(*missing, key)
		}
	default:
		normalized := normalizeSoftwareSourceJSONValue(value)
		if slice, ok := normalized.([]any); !ok || len(slice) == 0 {
			*missing = append(*missing, key)
		}
	}
}

func softwareSourceStringValue(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case fmt.Stringer:
		return strings.TrimSpace(typed.String()), true
	default:
		return "", false
	}
}

func newSoftwareSourceServiceClientWithOCIClient(client softwareSourceOCIClient) SoftwareSourceServiceClient {
	hooks := newSoftwareSourceRuntimeHooksWithOCIClient(client)
	applySoftwareSourceRuntimeHookSettings(&hooks)
	manager := &SoftwareSourceServiceManager{}
	return softwareSourceResourceContextClient{
		delegate: defaultSoftwareSourceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.SoftwareSource](
				buildSoftwareSourceGeneratedRuntimeConfig(manager, hooks),
			),
		},
	}
}

func newSoftwareSourceRuntimeHooksWithOCIClient(client softwareSourceOCIClient) SoftwareSourceRuntimeHooks {
	return SoftwareSourceRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*osmanagementhubv1beta1.SoftwareSource]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osmanagementhubv1beta1.SoftwareSource]{},
		StatusHooks:     generatedruntime.StatusHooks[*osmanagementhubv1beta1.SoftwareSource]{},
		ParityHooks:     generatedruntime.ParityHooks[*osmanagementhubv1beta1.SoftwareSource]{},
		Async:           generatedruntime.AsyncHooks[*osmanagementhubv1beta1.SoftwareSource]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osmanagementhubv1beta1.SoftwareSource]{},
		Create: runtimeOperationHooks[osmanagementhubsdk.CreateSoftwareSourceRequest, osmanagementhubsdk.CreateSoftwareSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateSoftwareSourceDetails", RequestName: "CreateSoftwareSourceDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.CreateSoftwareSourceRequest) (osmanagementhubsdk.CreateSoftwareSourceResponse, error) {
				return client.CreateSoftwareSource(ctx, request)
			},
		},
		Get: runtimeOperationHooks[osmanagementhubsdk.GetSoftwareSourceRequest, osmanagementhubsdk.GetSoftwareSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SoftwareSourceId", RequestName: "softwareSourceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.GetSoftwareSourceRequest) (osmanagementhubsdk.GetSoftwareSourceResponse, error) {
				return client.GetSoftwareSource(ctx, request)
			},
		},
		List: runtimeOperationHooks[osmanagementhubsdk.ListSoftwareSourcesRequest, osmanagementhubsdk.ListSoftwareSourcesResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"}, {FieldName: "Page", RequestName: "page", Contribution: "query"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.ListSoftwareSourcesRequest) (osmanagementhubsdk.ListSoftwareSourcesResponse, error) {
				return client.ListSoftwareSources(ctx, request)
			},
		},
		Update: runtimeOperationHooks[osmanagementhubsdk.UpdateSoftwareSourceRequest, osmanagementhubsdk.UpdateSoftwareSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SoftwareSourceId", RequestName: "softwareSourceId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateSoftwareSourceDetails", RequestName: "UpdateSoftwareSourceDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.UpdateSoftwareSourceRequest) (osmanagementhubsdk.UpdateSoftwareSourceResponse, error) {
				return client.UpdateSoftwareSource(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[osmanagementhubsdk.DeleteSoftwareSourceRequest, osmanagementhubsdk.DeleteSoftwareSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SoftwareSourceId", RequestName: "softwareSourceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.DeleteSoftwareSourceRequest) (osmanagementhubsdk.DeleteSoftwareSourceResponse, error) {
				return client.DeleteSoftwareSource(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SoftwareSourceServiceClient) SoftwareSourceServiceClient{},
	}
}
