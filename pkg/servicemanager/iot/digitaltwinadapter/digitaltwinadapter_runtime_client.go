/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwinadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
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

const digitalTwinAdapterDeletePendingMessage = "OCI DigitalTwinAdapter delete is in progress"

type digitalTwinAdapterOCIClient interface {
	CreateDigitalTwinAdapter(context.Context, iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error)
	GetDigitalTwinAdapter(context.Context, iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error)
	ListDigitalTwinAdapters(context.Context, iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error)
	UpdateDigitalTwinAdapter(context.Context, iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error)
	DeleteDigitalTwinAdapter(context.Context, iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error)
}

type digitalTwinAdapterAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e digitalTwinAdapterAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e digitalTwinAdapterAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerDigitalTwinAdapterRuntimeHooksMutator(func(_ *DigitalTwinAdapterServiceManager, hooks *DigitalTwinAdapterRuntimeHooks) {
		applyDigitalTwinAdapterRuntimeHooks(hooks)
	})
}

func applyDigitalTwinAdapterRuntimeHooks(hooks *DigitalTwinAdapterRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = digitalTwinAdapterRuntimeSemantics()
	hooks.BuildCreateBody = buildDigitalTwinAdapterCreateBody
	hooks.BuildUpdateBody = buildDigitalTwinAdapterUpdateBody
	hooks.List.Fields = digitalTwinAdapterListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDigitalTwinAdaptersAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDigitalTwinAdapterCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleDigitalTwinAdapterDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDigitalTwinAdapterDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markDigitalTwinAdapterTerminating
	wrapDigitalTwinAdapterDeleteConfirmation(hooks)
}

func newDigitalTwinAdapterServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client digitalTwinAdapterOCIClient,
) DigitalTwinAdapterServiceClient {
	manager := &DigitalTwinAdapterServiceManager{Log: log}
	hooks := newDigitalTwinAdapterRuntimeHooksWithOCIClient(client)
	applyDigitalTwinAdapterRuntimeHooks(&hooks)
	delegate := defaultDigitalTwinAdapterServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*iotv1beta1.DigitalTwinAdapter](
			buildDigitalTwinAdapterGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDigitalTwinAdapterGeneratedClient(hooks, delegate)
}

func newDigitalTwinAdapterRuntimeHooksWithOCIClient(client digitalTwinAdapterOCIClient) DigitalTwinAdapterRuntimeHooks {
	hooks := newDigitalTwinAdapterDefaultRuntimeHooks(iotsdk.IotClient{})
	hooks.Create.Call = func(ctx context.Context, request iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error) {
		if client == nil {
			return iotsdk.CreateDigitalTwinAdapterResponse{}, fmt.Errorf("DigitalTwinAdapter OCI client is not configured")
		}
		return client.CreateDigitalTwinAdapter(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
		if client == nil {
			return iotsdk.GetDigitalTwinAdapterResponse{}, fmt.Errorf("DigitalTwinAdapter OCI client is not configured")
		}
		return client.GetDigitalTwinAdapter(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
		if client == nil {
			return iotsdk.ListDigitalTwinAdaptersResponse{}, fmt.Errorf("DigitalTwinAdapter OCI client is not configured")
		}
		return client.ListDigitalTwinAdapters(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
		if client == nil {
			return iotsdk.UpdateDigitalTwinAdapterResponse{}, fmt.Errorf("DigitalTwinAdapter OCI client is not configured")
		}
		return client.UpdateDigitalTwinAdapter(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error) {
		if client == nil {
			return iotsdk.DeleteDigitalTwinAdapterResponse{}, fmt.Errorf("DigitalTwinAdapter OCI client is not configured")
		}
		return client.DeleteDigitalTwinAdapter(ctx, request)
	}
	return hooks
}

func digitalTwinAdapterRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "iot",
		FormalSlug:    "digitaltwinadapter",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(iotsdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(iotsdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"iotDomainId", "digitalTwinModelId", "digitalTwinModelSpecUri", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "inboundEnvelope", "inboundRoutes", "freeformTags", "definedTags"},
			ForceNew:      []string{"iotDomainId", "digitalTwinModelId", "digitalTwinModelSpecUri"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinAdapter", Action: "CreateDigitalTwinAdapter"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinAdapter", Action: "UpdateDigitalTwinAdapter"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinAdapter", Action: "DeleteDigitalTwinAdapter"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinAdapter", Action: "GetDigitalTwinAdapter"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinAdapter", Action: "GetDigitalTwinAdapter"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinAdapter", Action: "GetDigitalTwinAdapter"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func digitalTwinAdapterListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "IotDomainId",
			RequestName:  "iotDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.iotDomainId", "spec.iotDomainId", "iotDomainId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "DigitalTwinModelSpecUri",
			RequestName:  "digitalTwinModelSpecUri",
			Contribution: "query",
			LookupPaths:  []string{"status.digitalTwinModelSpecUri", "spec.digitalTwinModelSpecUri", "digitalTwinModelSpecUri"},
		},
		{
			FieldName:    "DigitalTwinModelId",
			RequestName:  "digitalTwinModelId",
			Contribution: "query",
			LookupPaths:  []string{"status.digitalTwinModelId", "spec.digitalTwinModelId", "digitalTwinModelId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func buildDigitalTwinAdapterCreateBody(_ context.Context, resource *iotv1beta1.DigitalTwinAdapter, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("DigitalTwinAdapter resource is nil")
	}
	if strings.TrimSpace(resource.Spec.IotDomainId) == "" {
		return nil, fmt.Errorf("iotDomainId is required")
	}

	body := iotsdk.CreateDigitalTwinAdapterDetails{
		IotDomainId: common.String(strings.TrimSpace(resource.Spec.IotDomainId)),
	}
	setDigitalTwinAdapterCreateOnlyFields(&body, resource.Spec)
	if err := setDigitalTwinAdapterMutableCreateFields(&body, resource.Spec); err != nil {
		return nil, err
	}
	return body, nil
}

func setDigitalTwinAdapterCreateOnlyFields(
	body *iotsdk.CreateDigitalTwinAdapterDetails,
	spec iotv1beta1.DigitalTwinAdapterSpec,
) {
	if value := strings.TrimSpace(spec.DigitalTwinModelId); value != "" {
		body.DigitalTwinModelId = common.String(value)
	}
	if value := strings.TrimSpace(spec.DigitalTwinModelSpecUri); value != "" {
		body.DigitalTwinModelSpecUri = common.String(value)
	}
}

func setDigitalTwinAdapterMutableCreateFields(
	body *iotsdk.CreateDigitalTwinAdapterDetails,
	spec iotv1beta1.DigitalTwinAdapterSpec,
) error {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.Description); value != "" {
		body.Description = common.String(value)
	}
	envelope, ok, err := digitalTwinAdapterInboundEnvelope(spec.InboundEnvelope)
	if err != nil {
		return err
	}
	if ok {
		body.InboundEnvelope = envelope
	}
	routes, ok, err := digitalTwinAdapterInboundRoutes(spec.InboundRoutes)
	if err != nil {
		return err
	}
	if ok {
		body.InboundRoutes = routes
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneDigitalTwinAdapterStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = digitalTwinAdapterDefinedTags(spec.DefinedTags)
	}
	return nil
}

func buildDigitalTwinAdapterUpdateBody(
	_ context.Context,
	resource *iotv1beta1.DigitalTwinAdapter,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("DigitalTwinAdapter resource is nil")
	}
	current, ok := digitalTwinAdapterBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current DigitalTwinAdapter response does not expose a DigitalTwinAdapter body")
	}

	body := iotsdk.UpdateDigitalTwinAdapterDetails{}
	updateNeeded := false
	setDigitalTwinAdapterStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	setDigitalTwinAdapterStringUpdate(&body.Description, &updateNeeded, resource.Spec.Description, current.Description)
	if err := setDigitalTwinAdapterEnvelopeUpdate(&body, &updateNeeded, resource.Spec.InboundEnvelope, current.InboundEnvelope); err != nil {
		return nil, false, err
	}
	if err := setDigitalTwinAdapterRoutesUpdate(&body, &updateNeeded, resource.Spec.InboundRoutes, current.InboundRoutes); err != nil {
		return nil, false, err
	}
	if resource.Spec.FreeformTags != nil {
		desired := cloneDigitalTwinAdapterStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := digitalTwinAdapterDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateDigitalTwinAdapterCreateOnlyDrift(
	resource *iotv1beta1.DigitalTwinAdapter,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("digitalTwinAdapter resource is nil")
	}
	current, ok := digitalTwinAdapterBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	if err := rejectDigitalTwinAdapterCreateOnlyDrift(
		"digitalTwinModelId",
		resource.Spec.DigitalTwinModelId,
		current.DigitalTwinModelId,
		resource.Status.DigitalTwinModelId,
	); err != nil {
		return err
	}
	return rejectDigitalTwinAdapterCreateOnlyDrift(
		"digitalTwinModelSpecUri",
		resource.Spec.DigitalTwinModelSpecUri,
		current.DigitalTwinModelSpecUri,
		resource.Status.DigitalTwinModelSpecUri,
	)
}

func rejectDigitalTwinAdapterCreateOnlyDrift(field string, desired string, observed *string, recorded string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(recorded)
	if observed != nil && strings.TrimSpace(*observed) != "" {
		current = strings.TrimSpace(*observed)
	}
	if desired == "" && current != "" {
		return fmt.Errorf("DigitalTwinAdapter formal semantics require replacement when %s changes", field)
	}
	if desired != "" && current != "" && desired != current {
		return fmt.Errorf("DigitalTwinAdapter formal semantics require replacement when %s changes", field)
	}
	return nil
}

func setDigitalTwinAdapterStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func setDigitalTwinAdapterEnvelopeUpdate(
	body *iotsdk.UpdateDigitalTwinAdapterDetails,
	updateNeeded *bool,
	spec iotv1beta1.DigitalTwinAdapterInboundEnvelope,
	current *iotsdk.DigitalTwinAdapterInboundEnvelope,
) error {
	desired, ok, err := digitalTwinAdapterInboundEnvelope(spec)
	if err != nil || !ok {
		return err
	}
	body.InboundEnvelope = desired
	if !reflect.DeepEqual(current, desired) {
		*updateNeeded = true
	}
	return nil
}

func setDigitalTwinAdapterRoutesUpdate(
	body *iotsdk.UpdateDigitalTwinAdapterDetails,
	updateNeeded *bool,
	spec []iotv1beta1.DigitalTwinAdapterInboundRoute,
	current []iotsdk.DigitalTwinAdapterInboundRoute,
) error {
	desired, ok, err := digitalTwinAdapterInboundRoutes(spec)
	if err != nil || !ok {
		return err
	}
	body.InboundRoutes = desired
	if !reflect.DeepEqual(current, desired) {
		*updateNeeded = true
	}
	return nil
}

func digitalTwinAdapterInboundEnvelope(
	spec iotv1beta1.DigitalTwinAdapterInboundEnvelope,
) (*iotsdk.DigitalTwinAdapterInboundEnvelope, bool, error) {
	payload, hasPayload, err := digitalTwinAdapterEnvelopePayload(spec.ReferencePayload)
	if err != nil {
		return nil, false, err
	}
	hasMapping := strings.TrimSpace(spec.EnvelopeMapping.TimeObserved) != ""
	hasEndpoint := strings.TrimSpace(spec.ReferenceEndpoint) != ""
	if !hasPayload && !hasMapping && !hasEndpoint {
		return nil, false, nil
	}
	if !hasEndpoint {
		return nil, false, fmt.Errorf("inboundEnvelope.referenceEndpoint is required when inboundEnvelope is set")
	}

	envelope := &iotsdk.DigitalTwinAdapterInboundEnvelope{
		ReferenceEndpoint: common.String(strings.TrimSpace(spec.ReferenceEndpoint)),
	}
	if hasPayload {
		envelope.ReferencePayload = payload
	}
	if hasMapping {
		envelope.EnvelopeMapping = &iotsdk.DigitalTwinAdapterEnvelopeMapping{
			TimeObserved: common.String(strings.TrimSpace(spec.EnvelopeMapping.TimeObserved)),
		}
	}
	return envelope, true, nil
}

func digitalTwinAdapterInboundRoutes(
	spec []iotv1beta1.DigitalTwinAdapterInboundRoute,
) ([]iotsdk.DigitalTwinAdapterInboundRoute, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	routes := make([]iotsdk.DigitalTwinAdapterInboundRoute, 0, len(spec))
	for index, route := range spec {
		if strings.TrimSpace(route.Condition) == "" {
			return nil, false, fmt.Errorf("inboundRoutes[%d].condition is required", index)
		}
		payload, hasPayload, err := digitalTwinAdapterRoutePayload(route.ReferencePayload)
		if err != nil {
			return nil, false, fmt.Errorf("inboundRoutes[%d].referencePayload: %w", index, err)
		}
		converted := iotsdk.DigitalTwinAdapterInboundRoute{
			Condition: common.String(strings.TrimSpace(route.Condition)),
		}
		if hasPayload {
			converted.ReferencePayload = payload
		}
		if route.PayloadMapping != nil {
			converted.PayloadMapping = cloneDigitalTwinAdapterStringMap(route.PayloadMapping)
		}
		if value := strings.TrimSpace(route.Description); value != "" {
			converted.Description = common.String(value)
		}
		routes = append(routes, converted)
	}
	return routes, true, nil
}

func digitalTwinAdapterEnvelopePayload(
	spec iotv1beta1.DigitalTwinAdapterInboundEnvelopeReferencePayload,
) (iotsdk.DigitalTwinAdapterPayload, bool, error) {
	return digitalTwinAdapterPayload(spec.JsonData, spec.DataFormat, spec.Data)
}

func digitalTwinAdapterRoutePayload(
	spec iotv1beta1.DigitalTwinAdapterInboundRouteReferencePayload,
) (iotsdk.DigitalTwinAdapterPayload, bool, error) {
	return digitalTwinAdapterPayload(spec.JsonData, spec.DataFormat, spec.Data)
}

func digitalTwinAdapterPayload(
	jsonData string,
	dataFormat string,
	data map[string]shared.JSONValue,
) (iotsdk.DigitalTwinAdapterPayload, bool, error) {
	decoded, ok, err := digitalTwinAdapterPayloadData(jsonData, data)
	if err != nil || !ok {
		return nil, false, err
	}
	format := strings.ToUpper(strings.TrimSpace(dataFormat))
	if format == "" {
		format = string(iotsdk.DigitalTwinAdapterPayloadDataFormatJson)
	}
	if format != string(iotsdk.DigitalTwinAdapterPayloadDataFormatJson) {
		return nil, false, fmt.Errorf("unsupported referencePayload.dataFormat %q", dataFormat)
	}
	return iotsdk.DigitalTwinAdapterJsonPayload{Data: decoded}, true, nil
}

func digitalTwinAdapterPayloadData(
	jsonData string,
	data map[string]shared.JSONValue,
) (map[string]interface{}, bool, error) {
	raw := strings.TrimSpace(jsonData)
	if raw != "" && len(data) > 0 {
		return nil, false, fmt.Errorf("referencePayload cannot set both jsonData and data")
	}
	if raw != "" {
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			return nil, false, fmt.Errorf("decode referencePayload.jsonData: %w", err)
		}
		return decoded, true, nil
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	decoded := make(map[string]interface{}, len(data))
	for key, value := range data {
		var item interface{}
		if len(value.Raw) > 0 {
			if err := json.Unmarshal(value.Raw, &item); err != nil {
				return nil, false, fmt.Errorf("decode referencePayload.data[%s]: %w", key, err)
			}
		}
		decoded[key] = item
	}
	return decoded, true, nil
}

func digitalTwinAdapterBodyFromResponse(response any) (iotsdk.DigitalTwinAdapter, bool) {
	switch current := response.(type) {
	case iotsdk.CreateDigitalTwinAdapterResponse:
		return current.DigitalTwinAdapter, true
	case *iotsdk.CreateDigitalTwinAdapterResponse:
		return digitalTwinAdapterCreateResponseBody(current)
	case iotsdk.GetDigitalTwinAdapterResponse:
		return current.DigitalTwinAdapter, true
	case *iotsdk.GetDigitalTwinAdapterResponse:
		return digitalTwinAdapterGetResponseBody(current)
	case iotsdk.UpdateDigitalTwinAdapterResponse:
		return current.DigitalTwinAdapter, true
	case *iotsdk.UpdateDigitalTwinAdapterResponse:
		return digitalTwinAdapterUpdateResponseBody(current)
	case iotsdk.DigitalTwinAdapter:
		return current, true
	case *iotsdk.DigitalTwinAdapter:
		return digitalTwinAdapterPointerBody(current)
	default:
		return iotsdk.DigitalTwinAdapter{}, false
	}
}

func digitalTwinAdapterCreateResponseBody(
	response *iotsdk.CreateDigitalTwinAdapterResponse,
) (iotsdk.DigitalTwinAdapter, bool) {
	if response == nil {
		return iotsdk.DigitalTwinAdapter{}, false
	}
	return response.DigitalTwinAdapter, true
}

func digitalTwinAdapterGetResponseBody(
	response *iotsdk.GetDigitalTwinAdapterResponse,
) (iotsdk.DigitalTwinAdapter, bool) {
	if response == nil {
		return iotsdk.DigitalTwinAdapter{}, false
	}
	return response.DigitalTwinAdapter, true
}

func digitalTwinAdapterUpdateResponseBody(
	response *iotsdk.UpdateDigitalTwinAdapterResponse,
) (iotsdk.DigitalTwinAdapter, bool) {
	if response == nil {
		return iotsdk.DigitalTwinAdapter{}, false
	}
	return response.DigitalTwinAdapter, true
}

func digitalTwinAdapterPointerBody(response *iotsdk.DigitalTwinAdapter) (iotsdk.DigitalTwinAdapter, bool) {
	if response == nil {
		return iotsdk.DigitalTwinAdapter{}, false
	}
	return *response, true
}

func listDigitalTwinAdaptersAllPages(
	call func(context.Context, iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error),
) func(context.Context, iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
	return func(ctx context.Context, request iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
		var combined iotsdk.ListDigitalTwinAdaptersResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return iotsdk.ListDigitalTwinAdaptersResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
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

func handleDigitalTwinAdapterDeleteError(resource *iotv1beta1.DigitalTwinAdapter, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return digitalTwinAdapterAmbiguousNotFoundError{
		message:      "DigitalTwinAdapter delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapDigitalTwinAdapterDeleteConfirmation(hooks *DigitalTwinAdapterRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getDigitalTwinAdapter := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DigitalTwinAdapterServiceClient) DigitalTwinAdapterServiceClient {
		return digitalTwinAdapterDeleteConfirmationClient{
			delegate:              delegate,
			getDigitalTwinAdapter: getDigitalTwinAdapter,
		}
	})
}

type digitalTwinAdapterDeleteConfirmationClient struct {
	delegate              DigitalTwinAdapterServiceClient
	getDigitalTwinAdapter func(context.Context, iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error)
}

func (c digitalTwinAdapterDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinAdapter,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c digitalTwinAdapterDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinAdapter,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c digitalTwinAdapterDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinAdapter,
) error {
	if c.getDigitalTwinAdapter == nil || resource == nil {
		return nil
	}
	adapterID := trackedDigitalTwinAdapterID(resource)
	if adapterID == "" {
		return nil
	}
	_, err := c.getDigitalTwinAdapter(ctx, iotsdk.GetDigitalTwinAdapterRequest{DigitalTwinAdapterId: common.String(adapterID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("DigitalTwinAdapter delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedDigitalTwinAdapterID(resource *iotv1beta1.DigitalTwinAdapter) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func applyDigitalTwinAdapterDeleteOutcome(
	resource *iotv1beta1.DigitalTwinAdapter,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := digitalTwinAdapterLifecycleState(response)
	if state == "" || state == string(iotsdk.LifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if state != string(iotsdk.LifecycleStateActive) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !digitalTwinAdapterDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markDigitalTwinAdapterTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func digitalTwinAdapterDeleteAlreadyPending(resource *iotv1beta1.DigitalTwinAdapter) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markDigitalTwinAdapterTerminating(resource *iotv1beta1.DigitalTwinAdapter, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = digitalTwinAdapterDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := digitalTwinAdapterLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         digitalTwinAdapterDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		digitalTwinAdapterDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func digitalTwinAdapterLifecycleState(response any) string {
	current, ok := digitalTwinAdapterBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneDigitalTwinAdapterStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func digitalTwinAdapterDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}
