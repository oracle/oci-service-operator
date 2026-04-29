/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwinmodel

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

const digitalTwinModelDeletePendingMessage = "OCI DigitalTwinModel delete is in progress"

type digitalTwinModelOCIClient interface {
	CreateDigitalTwinModel(context.Context, iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error)
	GetDigitalTwinModel(context.Context, iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error)
	GetDigitalTwinModelSpec(context.Context, iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error)
	ListDigitalTwinModels(context.Context, iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error)
	UpdateDigitalTwinModel(context.Context, iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error)
	DeleteDigitalTwinModel(context.Context, iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error)
}

type digitalTwinModelSpecGetter func(context.Context, iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error)

type digitalTwinModelAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e digitalTwinModelAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e digitalTwinModelAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerDigitalTwinModelRuntimeHooksMutator(func(manager *DigitalTwinModelServiceManager, hooks *DigitalTwinModelRuntimeHooks) {
		applyDigitalTwinModelRuntimeHooks(hooks, digitalTwinModelSpecGetterFromManager(manager))
	})
}

func applyDigitalTwinModelRuntimeHooks(hooks *DigitalTwinModelRuntimeHooks, getSpec digitalTwinModelSpecGetter) {
	if hooks == nil {
		return
	}

	hooks.Semantics = digitalTwinModelRuntimeSemantics()
	hooks.BuildCreateBody = buildDigitalTwinModelCreateBody
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *iotv1beta1.DigitalTwinModel,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildDigitalTwinModelUpdateBody(ctx, resource, namespace, currentResponse, getSpec)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardDigitalTwinModelExistingBeforeCreate
	hooks.List.Fields = digitalTwinModelListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDigitalTwinModelsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDigitalTwinModelCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleDigitalTwinModelDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDigitalTwinModelDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markDigitalTwinModelTerminating
	wrapDigitalTwinModelDeleteConfirmation(hooks)
}

func digitalTwinModelSpecGetterFromManager(manager *DigitalTwinModelServiceManager) digitalTwinModelSpecGetter {
	if manager == nil || manager.Provider == nil {
		return nil
	}
	sdkClient, err := iotsdk.NewIotClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return func(context.Context, iotsdk.GetDigitalTwinModelSpecRequest) (iotsdk.GetDigitalTwinModelSpecResponse, error) {
			return iotsdk.GetDigitalTwinModelSpecResponse{}, fmt.Errorf("initialize DigitalTwinModel OCI client for spec read: %w", err)
		}
	}
	return sdkClient.GetDigitalTwinModelSpec
}

func newDigitalTwinModelServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client digitalTwinModelOCIClient,
) DigitalTwinModelServiceClient {
	manager := &DigitalTwinModelServiceManager{Log: log}
	hooks := newDigitalTwinModelRuntimeHooksWithOCIClient(client)
	var getSpec digitalTwinModelSpecGetter
	if client != nil {
		getSpec = client.GetDigitalTwinModelSpec
	}
	applyDigitalTwinModelRuntimeHooks(&hooks, getSpec)
	delegate := defaultDigitalTwinModelServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*iotv1beta1.DigitalTwinModel](
			buildDigitalTwinModelGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDigitalTwinModelGeneratedClient(hooks, delegate)
}

func newDigitalTwinModelRuntimeHooksWithOCIClient(client digitalTwinModelOCIClient) DigitalTwinModelRuntimeHooks {
	hooks := newDigitalTwinModelDefaultRuntimeHooks(iotsdk.IotClient{})
	hooks.Create.Call = func(ctx context.Context, request iotsdk.CreateDigitalTwinModelRequest) (iotsdk.CreateDigitalTwinModelResponse, error) {
		if client == nil {
			return iotsdk.CreateDigitalTwinModelResponse{}, fmt.Errorf("DigitalTwinModel OCI client is not configured")
		}
		return client.CreateDigitalTwinModel(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error) {
		if client == nil {
			return iotsdk.GetDigitalTwinModelResponse{}, fmt.Errorf("DigitalTwinModel OCI client is not configured")
		}
		return client.GetDigitalTwinModel(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error) {
		if client == nil {
			return iotsdk.ListDigitalTwinModelsResponse{}, fmt.Errorf("DigitalTwinModel OCI client is not configured")
		}
		return client.ListDigitalTwinModels(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request iotsdk.UpdateDigitalTwinModelRequest) (iotsdk.UpdateDigitalTwinModelResponse, error) {
		if client == nil {
			return iotsdk.UpdateDigitalTwinModelResponse{}, fmt.Errorf("DigitalTwinModel OCI client is not configured")
		}
		return client.UpdateDigitalTwinModel(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request iotsdk.DeleteDigitalTwinModelRequest) (iotsdk.DeleteDigitalTwinModelResponse, error) {
		if client == nil {
			return iotsdk.DeleteDigitalTwinModelResponse{}, fmt.Errorf("DigitalTwinModel OCI client is not configured")
		}
		return client.DeleteDigitalTwinModel(ctx, request)
	}
	return hooks
}

func digitalTwinModelRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "iot",
		FormalSlug:    "digitaltwinmodel",
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
			MatchFields:        []string{"iotDomainId", "specUri", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"iotDomainId", "spec"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinModel", Action: "CreateDigitalTwinModel"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinModel", Action: "UpdateDigitalTwinModel"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinModel", Action: "DeleteDigitalTwinModel"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinModel", Action: "GetDigitalTwinModel"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinModel", Action: "GetDigitalTwinModel"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinModel", Action: "GetDigitalTwinModel"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func digitalTwinModelListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "IotDomainId",
			RequestName:  "iotDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.iotDomainId", "spec.iotDomainId", "iotDomainId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "SpecUriStartsWith",
			RequestName:  "specUriStartsWith",
			Contribution: "query",
			LookupPaths:  []string{"status.specUri", "spec.spec.@id", "specUri"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func guardDigitalTwinModelExistingBeforeCreate(
	_ context.Context,
	resource *iotv1beta1.DigitalTwinModel,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if trackedDigitalTwinModelID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.IotDomainId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	specURI, err := desiredDigitalTwinModelSpecURI(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if specURI == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildDigitalTwinModelCreateBody(_ context.Context, resource *iotv1beta1.DigitalTwinModel, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("DigitalTwinModel resource is nil")
	}
	if strings.TrimSpace(resource.Spec.IotDomainId) == "" {
		return nil, fmt.Errorf("iotDomainId is required")
	}
	spec, _, err := digitalTwinModelDesiredSpec(resource)
	if err != nil {
		return nil, err
	}

	body := iotsdk.CreateDigitalTwinModelDetails{
		IotDomainId: common.String(strings.TrimSpace(resource.Spec.IotDomainId)),
		Spec:        spec,
	}
	setDigitalTwinModelMutableCreateFields(&body, resource.Spec)
	return body, nil
}

func setDigitalTwinModelMutableCreateFields(
	body *iotsdk.CreateDigitalTwinModelDetails,
	spec iotv1beta1.DigitalTwinModelSpec,
) {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.Description); value != "" {
		body.Description = common.String(value)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneDigitalTwinModelStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = digitalTwinModelDefinedTags(spec.DefinedTags)
	}
}

func buildDigitalTwinModelUpdateBody(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinModel,
	_ string,
	currentResponse any,
	getSpec digitalTwinModelSpecGetter,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("DigitalTwinModel resource is nil")
	}
	current, ok := digitalTwinModelBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current DigitalTwinModel response does not expose a DigitalTwinModel body")
	}
	desiredSpec, desiredSpecURI, err := digitalTwinModelDesiredSpec(resource)
	if err != nil {
		return nil, false, err
	}
	if err := validateDigitalTwinModelSpecDrift(ctx, desiredSpec, desiredSpecURI, current, getSpec); err != nil {
		return nil, false, err
	}

	body := iotsdk.UpdateDigitalTwinModelDetails{}
	updateNeeded := false
	setDigitalTwinModelStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	setDigitalTwinModelStringUpdate(&body.Description, &updateNeeded, resource.Spec.Description, current.Description)
	if resource.Spec.FreeformTags != nil {
		desired := cloneDigitalTwinModelStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := digitalTwinModelDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateDigitalTwinModelCreateOnlyDrift(
	resource *iotv1beta1.DigitalTwinModel,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("DigitalTwinModel resource is nil")
	}
	current, ok := digitalTwinModelBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return rejectDigitalTwinModelCreateOnlyDrift(
		"iotDomainId",
		resource.Spec.IotDomainId,
		current.IotDomainId,
		resource.Status.IotDomainId,
	)
}

func rejectDigitalTwinModelCreateOnlyDrift(field string, desired string, observed *string, recorded string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(recorded)
	if observed != nil && strings.TrimSpace(*observed) != "" {
		current = strings.TrimSpace(*observed)
	}
	if desired == "" && current != "" {
		return fmt.Errorf("DigitalTwinModel formal semantics require replacement when %s changes", field)
	}
	if desired != "" && current != "" && desired != current {
		return fmt.Errorf("DigitalTwinModel formal semantics require replacement when %s changes", field)
	}
	return nil
}

func validateDigitalTwinModelSpecDrift(
	ctx context.Context,
	desiredSpec map[string]interface{},
	desiredSpecURI string,
	current iotsdk.DigitalTwinModel,
	getSpec digitalTwinModelSpecGetter,
) error {
	currentSpecURI := stringPointerValue(current.SpecUri)
	if currentSpecURI != "" && desiredSpecURI != currentSpecURI {
		return fmt.Errorf("DigitalTwinModel formal semantics require replacement when spec changes")
	}
	currentID := stringPointerValue(current.Id)
	if currentID == "" {
		return nil
	}
	if getSpec == nil {
		return fmt.Errorf("DigitalTwinModel spec read is not configured")
	}
	response, err := getSpec(ctx, iotsdk.GetDigitalTwinModelSpecRequest{DigitalTwinModelId: common.String(currentID)})
	if err != nil {
		return fmt.Errorf("read DigitalTwinModel spec %q: %w", currentID, err)
	}
	if response.Object == nil {
		return fmt.Errorf("read DigitalTwinModel spec %q returned empty spec", currentID)
	}
	if !digitalTwinModelSpecsEqual(desiredSpec, response.Object) {
		return fmt.Errorf("DigitalTwinModel formal semantics require replacement when spec changes")
	}
	return nil
}

func setDigitalTwinModelStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func digitalTwinModelDesiredSpec(resource *iotv1beta1.DigitalTwinModel) (map[string]interface{}, string, error) {
	spec, err := digitalTwinModelSpecMap(resource.Spec.Spec)
	if err != nil {
		return nil, "", err
	}
	specURI, err := digitalTwinModelSpecURI(spec)
	if err != nil {
		return nil, "", err
	}
	return spec, specURI, nil
}

func desiredDigitalTwinModelSpecURI(resource *iotv1beta1.DigitalTwinModel) (string, error) {
	_, specURI, err := digitalTwinModelDesiredSpec(resource)
	return specURI, err
}

func digitalTwinModelSpecMap(spec map[string]shared.JSONValue) (map[string]interface{}, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("spec is required")
	}
	converted := make(map[string]interface{}, len(spec))
	for key, value := range spec {
		var item interface{}
		if len(value.Raw) > 0 {
			if err := json.Unmarshal(value.Raw, &item); err != nil {
				return nil, fmt.Errorf("decode spec[%s]: %w", key, err)
			}
		}
		converted[key] = item
	}
	return converted, nil
}

func digitalTwinModelSpecURI(spec map[string]interface{}) (string, error) {
	raw, ok := spec["@id"]
	if !ok {
		return "", fmt.Errorf("spec.@id is required")
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("spec.@id must be a string")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("spec.@id is required")
	}
	return value, nil
}

func digitalTwinModelSpecsEqual(left map[string]interface{}, right map[string]interface{}) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func digitalTwinModelBodyFromResponse(response any) (iotsdk.DigitalTwinModel, bool) {
	if current, ok := digitalTwinModelMutationResponseBody(response); ok {
		return current, true
	}
	if current, ok := digitalTwinModelReadResponseBody(response); ok {
		return current, true
	}
	return digitalTwinModelSummaryResponseBody(response)
}

func digitalTwinModelMutationResponseBody(response any) (iotsdk.DigitalTwinModel, bool) {
	switch current := response.(type) {
	case iotsdk.CreateDigitalTwinModelResponse:
		return current.DigitalTwinModel, true
	case *iotsdk.CreateDigitalTwinModelResponse:
		return digitalTwinModelCreateResponseBody(current)
	case iotsdk.UpdateDigitalTwinModelResponse:
		return current.DigitalTwinModel, true
	case *iotsdk.UpdateDigitalTwinModelResponse:
		return digitalTwinModelUpdateResponseBody(current)
	default:
		return iotsdk.DigitalTwinModel{}, false
	}
}

func digitalTwinModelReadResponseBody(response any) (iotsdk.DigitalTwinModel, bool) {
	switch current := response.(type) {
	case iotsdk.GetDigitalTwinModelResponse:
		return current.DigitalTwinModel, true
	case *iotsdk.GetDigitalTwinModelResponse:
		return digitalTwinModelGetResponseBody(current)
	case iotsdk.DigitalTwinModel:
		return current, true
	case *iotsdk.DigitalTwinModel:
		return digitalTwinModelPointerBody(current)
	default:
		return iotsdk.DigitalTwinModel{}, false
	}
}

func digitalTwinModelSummaryResponseBody(response any) (iotsdk.DigitalTwinModel, bool) {
	switch current := response.(type) {
	case iotsdk.DigitalTwinModelSummary:
		return digitalTwinModelFromSummary(current), true
	case *iotsdk.DigitalTwinModelSummary:
		return digitalTwinModelSummaryPointerBody(current)
	default:
		return iotsdk.DigitalTwinModel{}, false
	}
}

func digitalTwinModelCreateResponseBody(
	response *iotsdk.CreateDigitalTwinModelResponse,
) (iotsdk.DigitalTwinModel, bool) {
	if response == nil {
		return iotsdk.DigitalTwinModel{}, false
	}
	return response.DigitalTwinModel, true
}

func digitalTwinModelGetResponseBody(
	response *iotsdk.GetDigitalTwinModelResponse,
) (iotsdk.DigitalTwinModel, bool) {
	if response == nil {
		return iotsdk.DigitalTwinModel{}, false
	}
	return response.DigitalTwinModel, true
}

func digitalTwinModelUpdateResponseBody(
	response *iotsdk.UpdateDigitalTwinModelResponse,
) (iotsdk.DigitalTwinModel, bool) {
	if response == nil {
		return iotsdk.DigitalTwinModel{}, false
	}
	return response.DigitalTwinModel, true
}

func digitalTwinModelPointerBody(response *iotsdk.DigitalTwinModel) (iotsdk.DigitalTwinModel, bool) {
	if response == nil {
		return iotsdk.DigitalTwinModel{}, false
	}
	return *response, true
}

func digitalTwinModelSummaryPointerBody(response *iotsdk.DigitalTwinModelSummary) (iotsdk.DigitalTwinModel, bool) {
	if response == nil {
		return iotsdk.DigitalTwinModel{}, false
	}
	return digitalTwinModelFromSummary(*response), true
}

func digitalTwinModelFromSummary(summary iotsdk.DigitalTwinModelSummary) iotsdk.DigitalTwinModel {
	return iotsdk.DigitalTwinModel{
		Id:             summary.Id,
		IotDomainId:    summary.IotDomainId,
		SpecUri:        summary.SpecUri,
		DisplayName:    summary.DisplayName,
		Description:    summary.Description,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
		FreeformTags:   cloneDigitalTwinModelStringMap(summary.FreeformTags),
		DefinedTags:    cloneDigitalTwinModelDefinedTagMap(summary.DefinedTags),
		SystemTags:     cloneDigitalTwinModelDefinedTagMap(summary.SystemTags),
		TimeUpdated:    summary.TimeUpdated,
	}
}

func listDigitalTwinModelsAllPages(
	call func(context.Context, iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error),
) func(context.Context, iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error) {
	return func(ctx context.Context, request iotsdk.ListDigitalTwinModelsRequest) (iotsdk.ListDigitalTwinModelsResponse, error) {
		var combined iotsdk.ListDigitalTwinModelsResponse
		exactSpecURI := strings.TrimSpace(stringPointerValue(request.SpecUriStartsWith))
		for {
			response, err := call(ctx, request)
			if err != nil {
				return iotsdk.ListDigitalTwinModelsResponse{}, err
			}
			appendMatchingDigitalTwinModelListPage(&combined, response, exactSpecURI)
			if !digitalTwinModelListHasNextPage(response) {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func appendMatchingDigitalTwinModelListPage(
	combined *iotsdk.ListDigitalTwinModelsResponse,
	response iotsdk.ListDigitalTwinModelsResponse,
	exactSpecURI string,
) {
	combined.RawResponse = response.RawResponse
	combined.OpcRequestId = response.OpcRequestId
	for _, item := range response.Items {
		if digitalTwinModelListItemMatchesSpecURI(item, exactSpecURI) {
			combined.Items = append(combined.Items, item)
		}
	}
}

func digitalTwinModelListItemMatchesSpecURI(item iotsdk.DigitalTwinModelSummary, exactSpecURI string) bool {
	if exactSpecURI == "" {
		return true
	}
	return strings.TrimSpace(stringPointerValue(item.SpecUri)) == exactSpecURI
}

func digitalTwinModelListHasNextPage(response iotsdk.ListDigitalTwinModelsResponse) bool {
	return response.OpcNextPage != nil && strings.TrimSpace(*response.OpcNextPage) != ""
}

func handleDigitalTwinModelDeleteError(resource *iotv1beta1.DigitalTwinModel, err error) error {
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
	return digitalTwinModelAmbiguousNotFoundError{
		message:      "DigitalTwinModel delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapDigitalTwinModelDeleteConfirmation(hooks *DigitalTwinModelRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getDigitalTwinModel := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DigitalTwinModelServiceClient) DigitalTwinModelServiceClient {
		return digitalTwinModelDeleteConfirmationClient{
			delegate:            delegate,
			getDigitalTwinModel: getDigitalTwinModel,
		}
	})
}

type digitalTwinModelDeleteConfirmationClient struct {
	delegate            DigitalTwinModelServiceClient
	getDigitalTwinModel func(context.Context, iotsdk.GetDigitalTwinModelRequest) (iotsdk.GetDigitalTwinModelResponse, error)
}

func (c digitalTwinModelDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinModel,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c digitalTwinModelDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinModel,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c digitalTwinModelDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinModel,
) error {
	if c.getDigitalTwinModel == nil || resource == nil {
		return nil
	}
	modelID := trackedDigitalTwinModelID(resource)
	if modelID == "" {
		return nil
	}
	_, err := c.getDigitalTwinModel(ctx, iotsdk.GetDigitalTwinModelRequest{DigitalTwinModelId: common.String(modelID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("DigitalTwinModel delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedDigitalTwinModelID(resource *iotv1beta1.DigitalTwinModel) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func applyDigitalTwinModelDeleteOutcome(
	resource *iotv1beta1.DigitalTwinModel,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := digitalTwinModelLifecycleState(response)
	if state == "" || state == string(iotsdk.LifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if state != string(iotsdk.LifecycleStateActive) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !digitalTwinModelDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markDigitalTwinModelTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func digitalTwinModelDeleteAlreadyPending(resource *iotv1beta1.DigitalTwinModel) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markDigitalTwinModelTerminating(resource *iotv1beta1.DigitalTwinModel, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = digitalTwinModelDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := digitalTwinModelLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         digitalTwinModelDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		digitalTwinModelDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func digitalTwinModelLifecycleState(response any) string {
	current, ok := digitalTwinModelBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cloneDigitalTwinModelStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func digitalTwinModelDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
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

func cloneDigitalTwinModelDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		clone[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			clone[namespace][key] = value
		}
	}
	return clone
}
