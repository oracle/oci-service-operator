/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoredresourcetype

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const monitoredResourceTypeKind = "MonitoredResourceType"

type monitoredResourceTypeIdentity struct {
	compartmentID string
	name          string
}

type monitoredResourceTypeAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type monitoredResourceTypeDeletePreflightClient struct {
	delegate MonitoredResourceTypeServiceClient
	get      func(context.Context, stackmonitoringsdk.GetMonitoredResourceTypeRequest) (stackmonitoringsdk.GetMonitoredResourceTypeResponse, error)
	list     func(context.Context, stackmonitoringsdk.ListMonitoredResourceTypesRequest) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error)
}

func (e monitoredResourceTypeAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e monitoredResourceTypeAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerMonitoredResourceTypeRuntimeHooksMutator(func(_ *MonitoredResourceTypeServiceManager, hooks *MonitoredResourceTypeRuntimeHooks) {
		applyMonitoredResourceTypeRuntimeHooks(hooks)
	})
}

func applyMonitoredResourceTypeRuntimeHooks(hooks *MonitoredResourceTypeRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newMonitoredResourceTypeRuntimeSemantics()
	hooks.BuildCreateBody = buildMonitoredResourceTypeCreateBody
	hooks.BuildUpdateBody = buildMonitoredResourceTypeUpdateBody
	hooks.Identity.Resolve = func(resource *stackmonitoringv1beta1.MonitoredResourceType) (any, error) {
		return resolveMonitoredResourceTypeIdentity(resource)
	}
	hooks.List.Fields = monitoredResourceTypeListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listMonitoredResourceTypesAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleMonitoredResourceTypeDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MonitoredResourceTypeServiceClient) MonitoredResourceTypeServiceClient {
		return monitoredResourceTypeDeletePreflightClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newMonitoredResourceTypeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "monitoredresourcetype",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(stackmonitoringsdk.ResourceTypeLifecycleStateCreating)},
			UpdatingStates:     []string{string(stackmonitoringsdk.ResourceTypeLifecycleStateUpdating)},
			ActiveStates:       []string{string(stackmonitoringsdk.ResourceTypeLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(stackmonitoringsdk.ResourceTypeLifecycleStateDeleting)},
			TerminalStates: []string{string(stackmonitoringsdk.ResourceTypeLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"metadata",
				"metricNamespace",
				"resourceCategory",
				"sourceType",
			},
			ForceNew: []string{
				"compartmentId",
				"name",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: monitoredResourceTypeKind, Action: "CreateMonitoredResourceType"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: monitoredResourceTypeKind, Action: "UpdateMonitoredResourceType"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: monitoredResourceTypeKind, Action: "DeleteMonitoredResourceType"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: monitoredResourceTypeKind, Action: "GetMonitoredResourceType"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: monitoredResourceTypeKind, Action: "GetMonitoredResourceType"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: monitoredResourceTypeKind, Action: "GetMonitoredResourceType"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func monitoredResourceTypeListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "status.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths:  []string{"spec.name", "status.name", "name"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func resolveMonitoredResourceTypeIdentity(resource *stackmonitoringv1beta1.MonitoredResourceType) (monitoredResourceTypeIdentity, error) {
	if resource == nil {
		return monitoredResourceTypeIdentity{}, fmt.Errorf("monitored resource type resource is nil")
	}
	if err := validateMonitoredResourceTypeSpec(resource.Spec); err != nil {
		return monitoredResourceTypeIdentity{}, err
	}
	return monitoredResourceTypeIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		name:          strings.TrimSpace(resource.Spec.Name),
	}, nil
}

func buildMonitoredResourceTypeCreateBody(_ context.Context, resource *stackmonitoringv1beta1.MonitoredResourceType, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("monitored resource type resource is nil")
	}
	if err := validateMonitoredResourceTypeSpec(resource.Spec); err != nil {
		return nil, err
	}

	sourceType, err := monitoredResourceTypeSourceType(resource.Spec.SourceType)
	if err != nil {
		return nil, err
	}
	resourceCategory, err := monitoredResourceTypeResourceCategory(resource.Spec.ResourceCategory)
	if err != nil {
		return nil, err
	}
	metadata, err := monitoredResourceTypeMetadata(resource.Spec.Metadata)
	if err != nil {
		return nil, err
	}

	return stackmonitoringsdk.CreateMonitoredResourceTypeDetails{
		Name:             common.String(strings.TrimSpace(resource.Spec.Name)),
		CompartmentId:    common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:      monitoredResourceTypeOptionalString(resource.Spec.DisplayName),
		Description:      monitoredResourceTypeOptionalString(resource.Spec.Description),
		MetricNamespace:  monitoredResourceTypeOptionalString(resource.Spec.MetricNamespace),
		SourceType:       sourceType,
		ResourceCategory: resourceCategory,
		Metadata:         metadata,
		FreeformTags:     cloneMonitoredResourceTypeStringMap(resource.Spec.FreeformTags),
		DefinedTags:      monitoredResourceTypeDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func buildMonitoredResourceTypeUpdateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("monitored resource type resource is nil")
	}
	if err := validateMonitoredResourceTypeSpec(resource.Spec); err != nil {
		return nil, false, err
	}

	current, err := monitoredResourceTypeResponseMap(currentResponse)
	if err != nil {
		return nil, false, err
	}
	if currentName := monitoredResourceTypeString(current, "name"); currentName != "" && currentName != strings.TrimSpace(resource.Spec.Name) {
		return nil, false, fmt.Errorf("%s formal semantics require replacement when name changes", monitoredResourceTypeKind)
	}
	if currentCompartmentID := monitoredResourceTypeString(current, "compartmentId"); currentCompartmentID != "" && currentCompartmentID != strings.TrimSpace(resource.Spec.CompartmentId) {
		return nil, false, fmt.Errorf("%s formal semantics require replacement when compartmentId changes", monitoredResourceTypeKind)
	}

	body, desired, err := buildMonitoredResourceTypeUpdateDetails(resource.Spec)
	if err != nil {
		return nil, false, err
	}
	return body, monitoredResourceTypeUpdateNeeded(desired, current), nil
}

func buildMonitoredResourceTypeUpdateDetails(
	spec stackmonitoringv1beta1.MonitoredResourceTypeSpec,
) (stackmonitoringsdk.UpdateMonitoredResourceTypeDetails, map[string]any, error) {
	body := stackmonitoringsdk.UpdateMonitoredResourceTypeDetails{}
	desired := map[string]any{}

	addMonitoredResourceTypeStringUpdateDetails(spec, &body, desired)
	if err := addMonitoredResourceTypeEnumUpdateDetails(spec, &body, desired); err != nil {
		return stackmonitoringsdk.UpdateMonitoredResourceTypeDetails{}, nil, err
	}
	if err := addMonitoredResourceTypeMetadataUpdateDetails(spec, &body, desired); err != nil {
		return stackmonitoringsdk.UpdateMonitoredResourceTypeDetails{}, nil, err
	}
	addMonitoredResourceTypeTagUpdateDetails(spec, &body, desired)

	return body, desired, nil
}

func addMonitoredResourceTypeStringUpdateDetails(
	spec stackmonitoringv1beta1.MonitoredResourceTypeSpec,
	body *stackmonitoringsdk.UpdateMonitoredResourceTypeDetails,
	desired map[string]any,
) {
	if strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(spec.DisplayName)
		desired["displayName"] = spec.DisplayName
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
		desired["description"] = spec.Description
	}
	if strings.TrimSpace(spec.MetricNamespace) != "" {
		body.MetricNamespace = common.String(spec.MetricNamespace)
		desired["metricNamespace"] = spec.MetricNamespace
	}
}

func addMonitoredResourceTypeEnumUpdateDetails(
	spec stackmonitoringv1beta1.MonitoredResourceTypeSpec,
	body *stackmonitoringsdk.UpdateMonitoredResourceTypeDetails,
	desired map[string]any,
) error {
	if strings.TrimSpace(spec.SourceType) != "" {
		sourceType, err := monitoredResourceTypeSourceType(spec.SourceType)
		if err != nil {
			return err
		}
		body.SourceType = sourceType
		desired["sourceType"] = string(sourceType)
	}
	if strings.TrimSpace(spec.ResourceCategory) != "" {
		resourceCategory, err := monitoredResourceTypeResourceCategory(spec.ResourceCategory)
		if err != nil {
			return err
		}
		body.ResourceCategory = resourceCategory
		desired["resourceCategory"] = string(resourceCategory)
	}
	return nil
}

func addMonitoredResourceTypeMetadataUpdateDetails(
	spec stackmonitoringv1beta1.MonitoredResourceTypeSpec,
	body *stackmonitoringsdk.UpdateMonitoredResourceTypeDetails,
	desired map[string]any,
) error {
	if monitoredResourceTypeMetadataSet(spec.Metadata) {
		metadata, err := monitoredResourceTypeMetadata(spec.Metadata)
		if err != nil {
			return err
		}
		body.Metadata = metadata
		desired["metadata"] = metadata
	}
	return nil
}

func addMonitoredResourceTypeTagUpdateDetails(
	spec stackmonitoringv1beta1.MonitoredResourceTypeSpec,
	body *stackmonitoringsdk.UpdateMonitoredResourceTypeDetails,
	desired map[string]any,
) {
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneMonitoredResourceTypeStringMap(spec.FreeformTags)
		desired["freeformTags"] = body.FreeformTags
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = monitoredResourceTypeDefinedTags(spec.DefinedTags)
		desired["definedTags"] = body.DefinedTags
	}
}

func validateMonitoredResourceTypeSpec(spec stackmonitoringv1beta1.MonitoredResourceTypeSpec) error {
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("%s spec is missing required field: name", monitoredResourceTypeKind)
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("%s spec is missing required field: compartmentId", monitoredResourceTypeKind)
	}
	if _, err := monitoredResourceTypeSourceType(spec.SourceType); err != nil {
		return err
	}
	if _, err := monitoredResourceTypeResourceCategory(spec.ResourceCategory); err != nil {
		return err
	}
	if _, err := monitoredResourceTypeMetadata(spec.Metadata); err != nil {
		return err
	}
	return nil
}

func monitoredResourceTypeSourceType(raw string) (stackmonitoringsdk.SourceTypeEnum, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	sourceType, ok := stackmonitoringsdk.GetMappingSourceTypeEnum(value)
	if !ok {
		return "", fmt.Errorf("unsupported %s sourceType %q; supported values: %s", monitoredResourceTypeKind, raw, strings.Join(stackmonitoringsdk.GetSourceTypeEnumStringValues(), ", "))
	}
	return sourceType, nil
}

func monitoredResourceTypeResourceCategory(raw string) (stackmonitoringsdk.ResourceCategoryEnum, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	resourceCategory, ok := stackmonitoringsdk.GetMappingResourceCategoryEnum(value)
	if !ok {
		return "", fmt.Errorf("unsupported %s resourceCategory %q; supported values: %s", monitoredResourceTypeKind, raw, strings.Join(stackmonitoringsdk.GetResourceCategoryEnumStringValues(), ", "))
	}
	return resourceCategory, nil
}

func monitoredResourceTypeMetadata(
	metadata stackmonitoringv1beta1.MonitoredResourceTypeMetadata,
) (stackmonitoringsdk.ResourceTypeMetadataDetails, error) {
	if !monitoredResourceTypeMetadataSet(metadata) {
		return nil, nil
	}
	if err := validateMonitoredResourceTypeMetadataFormat(metadata.Format); err != nil {
		return nil, err
	}
	if strings.TrimSpace(metadata.JsonData) != "" {
		return monitoredResourceTypeMetadataFromJSON(metadata.JsonData)
	}

	return stackmonitoringsdk.SystemFormatResourceTypeMetadataDetails{
		RequiredProperties:       cloneMonitoredResourceTypeStringSlice(metadata.RequiredProperties),
		AgentProperties:          cloneMonitoredResourceTypeStringSlice(metadata.AgentProperties),
		ValidPropertiesForCreate: cloneMonitoredResourceTypeStringSlice(metadata.ValidPropertiesForCreate),
		ValidPropertiesForUpdate: cloneMonitoredResourceTypeStringSlice(metadata.ValidPropertiesForUpdate),
		UniquePropertySets:       monitoredResourceTypeUniquePropertySets(metadata.UniquePropertySets),
		ValidPropertyValues:      cloneMonitoredResourceTypeStringSliceMap(metadata.ValidPropertyValues),
		ValidSubResourceTypes:    cloneMonitoredResourceTypeStringSlice(metadata.ValidSubResourceTypes),
	}, nil
}

func monitoredResourceTypeMetadataFromJSON(raw string) (stackmonitoringsdk.ResourceTypeMetadataDetails, error) {
	var discriminator struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("%s metadata.jsonData is not valid JSON: %w", monitoredResourceTypeKind, err)
	}
	if err := validateMonitoredResourceTypeMetadataFormat(discriminator.Format); err != nil {
		return nil, err
	}
	var metadata stackmonitoringsdk.SystemFormatResourceTypeMetadataDetails
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("%s metadata.jsonData does not match SYSTEM_FORMAT metadata: %w", monitoredResourceTypeKind, err)
	}
	return metadata, nil
}

func validateMonitoredResourceTypeMetadataFormat(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if _, ok := stackmonitoringsdk.GetMappingResourceTypeMetadataDetailsFormatEnum(value); !ok {
		return fmt.Errorf("unsupported %s metadata.format %q; supported values: %s", monitoredResourceTypeKind, raw, strings.Join(stackmonitoringsdk.GetResourceTypeMetadataDetailsFormatEnumStringValues(), ", "))
	}
	return nil
}

func monitoredResourceTypeMetadataSet(metadata stackmonitoringv1beta1.MonitoredResourceTypeMetadata) bool {
	return strings.TrimSpace(metadata.JsonData) != "" ||
		strings.TrimSpace(metadata.Format) != "" ||
		len(metadata.RequiredProperties) > 0 ||
		len(metadata.AgentProperties) > 0 ||
		len(metadata.ValidPropertiesForCreate) > 0 ||
		len(metadata.ValidPropertiesForUpdate) > 0 ||
		len(metadata.UniquePropertySets) > 0 ||
		len(metadata.ValidPropertyValues) > 0 ||
		len(metadata.ValidSubResourceTypes) > 0
}

func monitoredResourceTypeUniquePropertySets(
	values []stackmonitoringv1beta1.MonitoredResourceTypeMetadataUniquePropertySet,
) []stackmonitoringsdk.UniquePropertySet {
	if values == nil {
		return nil
	}
	converted := make([]stackmonitoringsdk.UniquePropertySet, 0, len(values))
	for _, value := range values {
		converted = append(converted, stackmonitoringsdk.UniquePropertySet{
			Properties: cloneMonitoredResourceTypeStringSlice(value.Properties),
		})
	}
	return converted
}

func listMonitoredResourceTypesAllPages(
	call func(context.Context, stackmonitoringsdk.ListMonitoredResourceTypesRequest) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error),
) func(context.Context, stackmonitoringsdk.ListMonitoredResourceTypesRequest) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.ListMonitoredResourceTypesRequest) (stackmonitoringsdk.ListMonitoredResourceTypesResponse, error) {
		var combined stackmonitoringsdk.ListMonitoredResourceTypesResponse
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

func (c monitoredResourceTypeDeletePreflightClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("monitored resource type runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c monitoredResourceTypeDeletePreflightClient) Delete(ctx context.Context, resource *stackmonitoringv1beta1.MonitoredResourceType) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("monitored resource type runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c monitoredResourceTypeDeletePreflightClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
) error {
	if resource == nil {
		return nil
	}
	if currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c monitoredResourceTypeDeletePreflightClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, stackmonitoringsdk.GetMonitoredResourceTypeRequest{MonitoredResourceTypeId: common.String(currentID)})
	return monitoredResourceTypeAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c monitoredResourceTypeDeletePreflightClient) rejectAuthShapedList(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoredResourceType,
) error {
	if c.list == nil {
		return nil
	}
	identity, err := resolveMonitoredResourceTypeIdentity(resource)
	if err != nil {
		return err
	}
	_, err = c.list(ctx, stackmonitoringsdk.ListMonitoredResourceTypesRequest{
		CompartmentId: common.String(identity.compartmentID),
		Name:          common.String(identity.name),
	})
	return monitoredResourceTypeAmbiguousDeleteError(resource, err, "pre-delete list")
}

func handleMonitoredResourceTypeDeleteError(resource *stackmonitoringv1beta1.MonitoredResourceType, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := monitoredResourceTypeAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func monitoredResourceTypeAmbiguousDeleteError(resource *stackmonitoringv1beta1.MonitoredResourceType, err error, operation string) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return monitoredResourceTypeAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", monitoredResourceTypeKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func monitoredResourceTypeResponseMap(response any) (map[string]any, error) {
	body, ok := monitoredResourceTypeResponseBody(response)
	if !ok || body == nil {
		return nil, fmt.Errorf("current %s response does not expose a resource body", monitoredResourceTypeKind)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal current %s body: %w", monitoredResourceTypeKind, err)
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode current %s body: %w", monitoredResourceTypeKind, err)
	}
	return values, nil
}

func monitoredResourceTypeResponseBody(response any) (any, bool) {
	if body, handled, ok := monitoredResourceTypeCreateResponseBody(response); handled {
		return body, ok
	}
	if body, handled, ok := monitoredResourceTypeGetResponseBody(response); handled {
		return body, ok
	}
	if body, handled, ok := monitoredResourceTypeUpdateResponseBody(response); handled {
		return body, ok
	}
	return monitoredResourceTypeStandaloneResponseBody(response)
}

func monitoredResourceTypeCreateResponseBody(response any) (any, bool, bool) {
	switch concrete := response.(type) {
	case stackmonitoringsdk.CreateMonitoredResourceTypeResponse:
		return concrete.MonitoredResourceType, true, true
	case *stackmonitoringsdk.CreateMonitoredResourceTypeResponse:
		if concrete == nil {
			return nil, true, false
		}
		return concrete.MonitoredResourceType, true, true
	default:
		return nil, false, false
	}
}

func monitoredResourceTypeGetResponseBody(response any) (any, bool, bool) {
	switch concrete := response.(type) {
	case stackmonitoringsdk.GetMonitoredResourceTypeResponse:
		return concrete.MonitoredResourceType, true, true
	case *stackmonitoringsdk.GetMonitoredResourceTypeResponse:
		if concrete == nil {
			return nil, true, false
		}
		return concrete.MonitoredResourceType, true, true
	default:
		return nil, false, false
	}
}

func monitoredResourceTypeUpdateResponseBody(response any) (any, bool, bool) {
	switch concrete := response.(type) {
	case stackmonitoringsdk.UpdateMonitoredResourceTypeResponse:
		return concrete.MonitoredResourceType, true, true
	case *stackmonitoringsdk.UpdateMonitoredResourceTypeResponse:
		if concrete == nil {
			return nil, true, false
		}
		return concrete.MonitoredResourceType, true, true
	default:
		return nil, false, false
	}
}

func monitoredResourceTypeStandaloneResponseBody(response any) (any, bool) {
	switch concrete := response.(type) {
	case nil:
		return nil, false
	case stackmonitoringsdk.MonitoredResourceType:
		return concrete, true
	case stackmonitoringsdk.MonitoredResourceTypeSummary:
		return concrete, true
	default:
		return response, true
	}
}

func monitoredResourceTypeUpdateNeeded(desired map[string]any, current map[string]any) bool {
	for key, desiredValue := range desired {
		currentValue, ok := monitoredResourceTypeMapValue(current, key)
		if !ok || !reflect.DeepEqual(monitoredResourceTypeComparableValue(desiredValue), monitoredResourceTypeComparableValue(currentValue)) {
			return true
		}
	}
	return false
}

func monitoredResourceTypeMapValue(values map[string]any, key string) (any, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}
	normalized := strings.ToLower(key)
	for candidate, value := range values {
		if strings.ToLower(candidate) == normalized {
			return value, true
		}
	}
	return nil, false
}

func monitoredResourceTypeString(values map[string]any, key string) string {
	value, ok := monitoredResourceTypeMapValue(values, key)
	if !ok || value == nil {
		return ""
	}
	if concrete, ok := value.(string); ok {
		return strings.TrimSpace(concrete)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func monitoredResourceTypeComparableValue(value any) any {
	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var comparable any
	if err := json.Unmarshal(payload, &comparable); err != nil {
		return value
	}
	return comparable
}

func monitoredResourceTypeOptionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func cloneMonitoredResourceTypeStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneMonitoredResourceTypeStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneMonitoredResourceTypeStringSliceMap(values map[string][]string) map[string][]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string][]string, len(values))
	for key, value := range values {
		cloned[key] = cloneMonitoredResourceTypeStringSlice(value)
	}
	return cloned
}

func monitoredResourceTypeDefinedTags(values map[string]shared.MapValue) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(values))
	for namespace, entries := range values {
		converted[namespace] = make(map[string]interface{}, len(entries))
		for key, value := range entries {
			converted[namespace][key] = value
		}
	}
	return converted
}
