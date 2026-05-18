/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package librarymaskingformat

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerLibraryMaskingFormatRuntimeHooksMutator(func(_ *LibraryMaskingFormatServiceManager, hooks *LibraryMaskingFormatRuntimeHooks) {
		applyLibraryMaskingFormatRuntimeHooks(hooks)
	})
}

func applyLibraryMaskingFormatRuntimeHooks(hooks *LibraryMaskingFormatRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = libraryMaskingFormatRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *datasafev1beta1.LibraryMaskingFormat, _ string) (any, error) {
		return buildLibraryMaskingFormatCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *datasafev1beta1.LibraryMaskingFormat,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildLibraryMaskingFormatUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardLibraryMaskingFormatExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedLibraryMaskingFormatIdentity
	hooks.DeleteHooks.HandleError = handleLibraryMaskingFormatDeleteError

	list := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListLibraryMaskingFormatsRequest) (datasafesdk.ListLibraryMaskingFormatsResponse, error) {
		return listAllLibraryMaskingFormats(ctx, request, list)
	}
	wrapLibraryMaskingFormatDeleteConfirmation(hooks)
}

type libraryMaskingFormatDeleteConfirmationClient struct {
	delegate                LibraryMaskingFormatServiceClient
	getLibraryMaskingFormat func(context.Context, datasafesdk.GetLibraryMaskingFormatRequest) (datasafesdk.GetLibraryMaskingFormatResponse, error)
}

func (c libraryMaskingFormatDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.LibraryMaskingFormat,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c libraryMaskingFormatDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *datasafev1beta1.LibraryMaskingFormat,
) (bool, error) {
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func wrapLibraryMaskingFormatDeleteConfirmation(hooks *LibraryMaskingFormatRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getLibraryMaskingFormat := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LibraryMaskingFormatServiceClient) LibraryMaskingFormatServiceClient {
		return libraryMaskingFormatDeleteConfirmationClient{
			delegate:                delegate,
			getLibraryMaskingFormat: getLibraryMaskingFormat,
		}
	})
}

func (c libraryMaskingFormatDeleteConfirmationClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.LibraryMaskingFormat,
) error {
	currentID := currentLibraryMaskingFormatID(resource)
	if currentID == "" || c.getLibraryMaskingFormat == nil {
		return nil
	}
	_, err := c.getLibraryMaskingFormat(ctx, datasafesdk.GetLibraryMaskingFormatRequest{
		LibraryMaskingFormatId: common.String(currentID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("LibraryMaskingFormat delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func currentLibraryMaskingFormatID(resource *datasafev1beta1.LibraryMaskingFormat) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func libraryMaskingFormatRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datasafe",
		FormalSlug:    "librarymaskingformat",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.MaskingLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.MaskingLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.MaskingLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(datasafesdk.MaskingLifecycleStateDeleting),
			},
			TerminalStates: []string{
				string(datasafesdk.MaskingLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"sensitiveTypeIds",
				"formatEntries",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
			},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LibraryMaskingFormat", Action: "CreateLibraryMaskingFormat"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LibraryMaskingFormat", Action: "UpdateLibraryMaskingFormat"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LibraryMaskingFormat", Action: "DeleteLibraryMaskingFormat"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LibraryMaskingFormat", Action: "GetLibraryMaskingFormat"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LibraryMaskingFormat", Action: "GetLibraryMaskingFormat"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LibraryMaskingFormat", Action: "GetLibraryMaskingFormat"}},
		},
	}
}

func buildLibraryMaskingFormatCreateBody(resource *datasafev1beta1.LibraryMaskingFormat) (datasafesdk.CreateLibraryMaskingFormatDetails, error) {
	if resource == nil {
		return datasafesdk.CreateLibraryMaskingFormatDetails{}, fmt.Errorf("LibraryMaskingFormat resource is nil")
	}

	entries, err := libraryMaskingFormatEntriesForOCI(resource.Spec.FormatEntries)
	if err != nil {
		return datasafesdk.CreateLibraryMaskingFormatDetails{}, err
	}
	if len(entries) == 0 {
		return datasafesdk.CreateLibraryMaskingFormatDetails{}, fmt.Errorf("LibraryMaskingFormat requires at least one format entry")
	}

	details := datasafesdk.CreateLibraryMaskingFormatDetails{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		FormatEntries: entries,
	}
	if resource.Spec.DisplayName != "" {
		details.DisplayName = common.String(resource.Spec.DisplayName)
	}
	if resource.Spec.Description != "" {
		details.Description = common.String(resource.Spec.Description)
	}
	if resource.Spec.SensitiveTypeIds != nil {
		details.SensitiveTypeIds = cloneLibraryMaskingFormatStringSlice(resource.Spec.SensitiveTypeIds)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneLibraryMaskingFormatStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = libraryMaskingFormatDefinedTags(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildLibraryMaskingFormatUpdateBody(
	resource *datasafev1beta1.LibraryMaskingFormat,
	currentResponse any,
) (datasafesdk.UpdateLibraryMaskingFormatDetails, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateLibraryMaskingFormatDetails{}, false, fmt.Errorf("LibraryMaskingFormat resource is nil")
	}

	current, ok := libraryMaskingFormatFromResponse(currentResponse)
	if !ok {
		return datasafesdk.UpdateLibraryMaskingFormatDetails{}, false, fmt.Errorf("current LibraryMaskingFormat response does not expose a LibraryMaskingFormat body")
	}

	details := datasafesdk.UpdateLibraryMaskingFormatDetails{}
	updateNeeded := applyLibraryMaskingFormatTextUpdates(&details, resource, current)
	updateNeeded = applyLibraryMaskingFormatSensitiveTypeIdsUpdate(&details, resource, current) || updateNeeded
	formatEntriesChanged, err := applyLibraryMaskingFormatEntriesUpdate(&details, resource, current)
	if err != nil {
		return datasafesdk.UpdateLibraryMaskingFormatDetails{}, false, err
	}
	updateNeeded = formatEntriesChanged || updateNeeded
	updateNeeded = applyLibraryMaskingFormatTagUpdates(&details, resource, current) || updateNeeded
	return details, updateNeeded, nil
}

func applyLibraryMaskingFormatTextUpdates(
	details *datasafesdk.UpdateLibraryMaskingFormatDetails,
	resource *datasafev1beta1.LibraryMaskingFormat,
	current datasafesdk.LibraryMaskingFormat,
) bool {
	updateNeeded := false
	if resource.Spec.DisplayName != "" && stringPointerValue(current.DisplayName) != resource.Spec.DisplayName {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.Description != "" && stringPointerValue(current.Description) != resource.Spec.Description {
		details.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}
	return updateNeeded
}

func applyLibraryMaskingFormatSensitiveTypeIdsUpdate(
	details *datasafesdk.UpdateLibraryMaskingFormatDetails,
	resource *datasafev1beta1.LibraryMaskingFormat,
	current datasafesdk.LibraryMaskingFormat,
) bool {
	if resource.Spec.SensitiveTypeIds == nil || reflect.DeepEqual(current.SensitiveTypeIds, resource.Spec.SensitiveTypeIds) {
		return false
	}
	details.SensitiveTypeIds = cloneLibraryMaskingFormatStringSlice(resource.Spec.SensitiveTypeIds)
	return true
}

func applyLibraryMaskingFormatEntriesUpdate(
	details *datasafesdk.UpdateLibraryMaskingFormatDetails,
	resource *datasafev1beta1.LibraryMaskingFormat,
	current datasafesdk.LibraryMaskingFormat,
) (bool, error) {
	if resource.Spec.FormatEntries == nil {
		return false, nil
	}
	entries, err := libraryMaskingFormatEntriesForOCI(resource.Spec.FormatEntries)
	if err != nil {
		return false, err
	}
	if libraryMaskingFormatEntriesEqual(current.FormatEntries, entries) {
		return false, nil
	}
	details.FormatEntries = entries
	return true, nil
}

func applyLibraryMaskingFormatTagUpdates(
	details *datasafesdk.UpdateLibraryMaskingFormatDetails,
	resource *datasafev1beta1.LibraryMaskingFormat,
	current datasafesdk.LibraryMaskingFormat,
) bool {
	freeformTagsChanged := applyLibraryMaskingFormatFreeformTagsUpdate(details, resource, current)
	definedTagsChanged := applyLibraryMaskingFormatDefinedTagsUpdate(details, resource, current)
	return freeformTagsChanged || definedTagsChanged
}

func applyLibraryMaskingFormatFreeformTagsUpdate(
	details *datasafesdk.UpdateLibraryMaskingFormatDetails,
	resource *datasafev1beta1.LibraryMaskingFormat,
	current datasafesdk.LibraryMaskingFormat,
) bool {
	if resource.Spec.FreeformTags == nil {
		return false
	}
	desiredTags := cloneLibraryMaskingFormatStringMap(resource.Spec.FreeformTags)
	if reflect.DeepEqual(current.FreeformTags, desiredTags) {
		return false
	}
	details.FreeformTags = desiredTags
	return true
}

func applyLibraryMaskingFormatDefinedTagsUpdate(
	details *datasafesdk.UpdateLibraryMaskingFormatDetails,
	resource *datasafev1beta1.LibraryMaskingFormat,
	current datasafesdk.LibraryMaskingFormat,
) bool {
	if resource.Spec.DefinedTags == nil {
		return false
	}
	desiredTags := libraryMaskingFormatDefinedTags(resource.Spec.DefinedTags)
	if reflect.DeepEqual(current.DefinedTags, desiredTags) {
		return false
	}
	details.DefinedTags = desiredTags
	return true
}

func guardLibraryMaskingFormatExistingBeforeCreate(
	_ context.Context,
	resource *datasafev1beta1.LibraryMaskingFormat,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("LibraryMaskingFormat resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func clearTrackedLibraryMaskingFormatIdentity(resource *datasafev1beta1.LibraryMaskingFormat) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
}

func listAllLibraryMaskingFormats(
	ctx context.Context,
	request datasafesdk.ListLibraryMaskingFormatsRequest,
	list func(context.Context, datasafesdk.ListLibraryMaskingFormatsRequest) (datasafesdk.ListLibraryMaskingFormatsResponse, error),
) (datasafesdk.ListLibraryMaskingFormatsResponse, error) {
	if list == nil {
		return datasafesdk.ListLibraryMaskingFormatsResponse{}, fmt.Errorf("LibraryMaskingFormat list operation is not configured")
	}

	request.LibraryMaskingFormatSource = datasafesdk.ListLibraryMaskingFormatsLibraryMaskingFormatSourceUser
	var aggregate datasafesdk.ListLibraryMaskingFormatsResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return datasafesdk.ListLibraryMaskingFormatsResponse{}, err
		}
		if aggregate.OpcRequestId == nil {
			aggregate.OpcRequestId = response.OpcRequestId
		}
		aggregate.Items = append(aggregate.Items, response.Items...)
		if response.OpcNextPage == nil || *response.OpcNextPage == "" {
			return aggregate, nil
		}
		request.Page = response.OpcNextPage
	}
}

func handleLibraryMaskingFormatDeleteError(resource *datasafev1beta1.LibraryMaskingFormat, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("LibraryMaskingFormat delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound: %s", err.Error())
}

func libraryMaskingFormatEntriesForOCI(entries []datasafev1beta1.LibraryMaskingFormatFormatEntry) ([]datasafesdk.FormatEntry, error) {
	if entries == nil {
		return nil, nil
	}

	converted := make([]datasafesdk.FormatEntry, 0, len(entries))
	for index, entry := range entries {
		ociEntry, err := libraryMaskingFormatEntryForOCI(entry)
		if err != nil {
			return nil, fmt.Errorf("formatEntries[%d]: %w", index, err)
		}
		converted = append(converted, ociEntry)
	}
	return converted, nil
}

func libraryMaskingFormatEntryForOCI(entry datasafev1beta1.LibraryMaskingFormatFormatEntry) (datasafesdk.FormatEntry, error) {
	payload, err := libraryMaskingFormatEntryPayload(entry)
	if err != nil {
		return nil, err
	}
	entryType := strings.TrimSpace(fmt.Sprint(payload["type"]))
	canonicalType, ok := datasafesdk.GetMappingFormatEntryTypeEnum(entryType)
	if !ok {
		return nil, fmt.Errorf("unsupported format entry type %q", entryType)
	}
	payload["type"] = string(canonicalType)

	wrapper := struct {
		FormatEntries []map[string]any `json:"formatEntries"`
	}{
		FormatEntries: []map[string]any{payload},
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("marshal format entry: %w", err)
	}

	var details datasafesdk.CreateLibraryMaskingFormatDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, fmt.Errorf("decode format entry as OCI SDK model: %w", err)
	}
	if len(details.FormatEntries) != 1 || details.FormatEntries[0] == nil {
		return nil, fmt.Errorf("format entry type %q did not resolve to an OCI SDK model", entryType)
	}
	return details.FormatEntries[0], nil
}

func libraryMaskingFormatEntryPayload(entry datasafev1beta1.LibraryMaskingFormatFormatEntry) (map[string]any, error) {
	payload := map[string]any{}
	if strings.TrimSpace(entry.JsonData) != "" {
		if err := json.Unmarshal([]byte(entry.JsonData), &payload); err != nil {
			return nil, fmt.Errorf("decode jsonData: %w", err)
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("marshal CRD format entry: %w", err)
	}
	var structured map[string]any
	if err := json.Unmarshal(data, &structured); err != nil {
		return nil, fmt.Errorf("decode CRD format entry: %w", err)
	}
	delete(structured, "jsonData")
	for key, value := range structured {
		if _, exists := payload[key]; !exists {
			payload[key] = value
		}
	}
	if strings.TrimSpace(entry.Type) != "" {
		payload["type"] = entry.Type
	}
	return payload, nil
}

//nolint:gocyclo // Response normalization must handle each OCI wrapper shape generated for this resource.
func libraryMaskingFormatFromResponse(response any) (datasafesdk.LibraryMaskingFormat, bool) {
	switch typed := response.(type) {
	case datasafesdk.LibraryMaskingFormat:
		return typed, true
	case *datasafesdk.LibraryMaskingFormat:
		if typed == nil {
			return datasafesdk.LibraryMaskingFormat{}, false
		}
		return *typed, true
	case datasafesdk.LibraryMaskingFormatSummary:
		return libraryMaskingFormatFromSummary(typed), true
	case *datasafesdk.LibraryMaskingFormatSummary:
		if typed == nil {
			return datasafesdk.LibraryMaskingFormat{}, false
		}
		return libraryMaskingFormatFromSummary(*typed), true
	case datasafesdk.CreateLibraryMaskingFormatResponse:
		return typed.LibraryMaskingFormat, true
	case *datasafesdk.CreateLibraryMaskingFormatResponse:
		if typed == nil {
			return datasafesdk.LibraryMaskingFormat{}, false
		}
		return typed.LibraryMaskingFormat, true
	case datasafesdk.GetLibraryMaskingFormatResponse:
		return typed.LibraryMaskingFormat, true
	case *datasafesdk.GetLibraryMaskingFormatResponse:
		if typed == nil {
			return datasafesdk.LibraryMaskingFormat{}, false
		}
		return typed.LibraryMaskingFormat, true
	default:
		return datasafesdk.LibraryMaskingFormat{}, false
	}
}

func libraryMaskingFormatFromSummary(summary datasafesdk.LibraryMaskingFormatSummary) datasafesdk.LibraryMaskingFormat {
	return datasafesdk.LibraryMaskingFormat{
		Id:               summary.Id,
		CompartmentId:    summary.CompartmentId,
		DisplayName:      summary.DisplayName,
		TimeCreated:      summary.TimeCreated,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleState:   summary.LifecycleState,
		Source:           summary.Source,
		Description:      summary.Description,
		SensitiveTypeIds: summary.SensitiveTypeIds,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
	}
}

func libraryMaskingFormatEntriesEqual(left []datasafesdk.FormatEntry, right []datasafesdk.FormatEntry) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func cloneLibraryMaskingFormatStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func cloneLibraryMaskingFormatStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func libraryMaskingFormatDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
