/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwininstance

import (
	"context"
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

const digitalTwinInstanceDeletePendingMessage = "OCI DigitalTwinInstance delete is in progress"

type digitalTwinInstanceOCIClient interface {
	CreateDigitalTwinInstance(context.Context, iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error)
	GetDigitalTwinInstance(context.Context, iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error)
	ListDigitalTwinInstances(context.Context, iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error)
	UpdateDigitalTwinInstance(context.Context, iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error)
	DeleteDigitalTwinInstance(context.Context, iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error)
}

type digitalTwinInstanceAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e digitalTwinInstanceAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e digitalTwinInstanceAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerDigitalTwinInstanceRuntimeHooksMutator(func(_ *DigitalTwinInstanceServiceManager, hooks *DigitalTwinInstanceRuntimeHooks) {
		applyDigitalTwinInstanceRuntimeHooks(hooks)
	})
}

func applyDigitalTwinInstanceRuntimeHooks(hooks *DigitalTwinInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = digitalTwinInstanceRuntimeSemantics()
	hooks.BuildCreateBody = buildDigitalTwinInstanceCreateBody
	hooks.BuildUpdateBody = buildDigitalTwinInstanceUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardDigitalTwinInstanceExistingBeforeCreate
	hooks.List.Fields = digitalTwinInstanceListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDigitalTwinInstancesAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDigitalTwinInstanceCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleDigitalTwinInstanceDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDigitalTwinInstanceDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markDigitalTwinInstanceTerminating
	wrapDigitalTwinInstanceDeleteConfirmation(hooks)
}

func newDigitalTwinInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client digitalTwinInstanceOCIClient,
) DigitalTwinInstanceServiceClient {
	manager := &DigitalTwinInstanceServiceManager{Log: log}
	hooks := newDigitalTwinInstanceRuntimeHooksWithOCIClient(client)
	applyDigitalTwinInstanceRuntimeHooks(&hooks)
	delegate := defaultDigitalTwinInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*iotv1beta1.DigitalTwinInstance](
			buildDigitalTwinInstanceGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDigitalTwinInstanceGeneratedClient(hooks, delegate)
}

func newDigitalTwinInstanceRuntimeHooksWithOCIClient(client digitalTwinInstanceOCIClient) DigitalTwinInstanceRuntimeHooks {
	hooks := newDigitalTwinInstanceDefaultRuntimeHooks(iotsdk.IotClient{})
	hooks.Create.Call = func(ctx context.Context, request iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error) {
		if client == nil {
			return iotsdk.CreateDigitalTwinInstanceResponse{}, fmt.Errorf("DigitalTwinInstance OCI client is not configured")
		}
		return client.CreateDigitalTwinInstance(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
		if client == nil {
			return iotsdk.GetDigitalTwinInstanceResponse{}, fmt.Errorf("DigitalTwinInstance OCI client is not configured")
		}
		return client.GetDigitalTwinInstance(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
		if client == nil {
			return iotsdk.ListDigitalTwinInstancesResponse{}, fmt.Errorf("DigitalTwinInstance OCI client is not configured")
		}
		return client.ListDigitalTwinInstances(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error) {
		if client == nil {
			return iotsdk.UpdateDigitalTwinInstanceResponse{}, fmt.Errorf("DigitalTwinInstance OCI client is not configured")
		}
		return client.UpdateDigitalTwinInstance(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error) {
		if client == nil {
			return iotsdk.DeleteDigitalTwinInstanceResponse{}, fmt.Errorf("DigitalTwinInstance OCI client is not configured")
		}
		return client.DeleteDigitalTwinInstance(ctx, request)
	}
	return hooks
}

func digitalTwinInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "iot",
		FormalSlug:    "digitaltwininstance",
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
			MatchFields: []string{
				"iotDomainId",
				"authId",
				"externalKey",
				"displayName",
				"digitalTwinAdapterId",
				"digitalTwinModelId",
				"digitalTwinModelSpecUri",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"authId",
				"externalKey",
				"displayName",
				"description",
				"digitalTwinAdapterId",
				"digitalTwinModelId",
				"digitalTwinModelSpecUri",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"iotDomainId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinInstance", Action: "CreateDigitalTwinInstance"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinInstance", Action: "UpdateDigitalTwinInstance"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinInstance", Action: "DeleteDigitalTwinInstance"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinInstance", Action: "GetDigitalTwinInstance"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinInstance", Action: "GetDigitalTwinInstance"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinInstance", Action: "GetDigitalTwinInstance"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func digitalTwinInstanceListFields() []generatedruntime.RequestField {
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

func guardDigitalTwinInstanceExistingBeforeCreate(
	_ context.Context,
	resource *iotv1beta1.DigitalTwinInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if trackedDigitalTwinInstanceID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.IotDomainId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	for _, value := range []string{
		resource.Spec.AuthId,
		resource.Spec.ExternalKey,
		resource.Spec.DisplayName,
		resource.Spec.DigitalTwinAdapterId,
		resource.Spec.DigitalTwinModelId,
		resource.Spec.DigitalTwinModelSpecUri,
	} {
		if strings.TrimSpace(value) != "" {
			return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
		}
	}
	return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
}

func buildDigitalTwinInstanceCreateBody(_ context.Context, resource *iotv1beta1.DigitalTwinInstance, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("DigitalTwinInstance resource is nil")
	}
	if strings.TrimSpace(resource.Spec.IotDomainId) == "" {
		return nil, fmt.Errorf("iotDomainId is required")
	}

	body := iotsdk.CreateDigitalTwinInstanceDetails{
		IotDomainId: common.String(strings.TrimSpace(resource.Spec.IotDomainId)),
	}
	setDigitalTwinInstanceMutableCreateFields(&body, resource.Spec)
	return body, nil
}

func setDigitalTwinInstanceMutableCreateFields(
	body *iotsdk.CreateDigitalTwinInstanceDetails,
	spec iotv1beta1.DigitalTwinInstanceSpec,
) {
	if value := strings.TrimSpace(spec.AuthId); value != "" {
		body.AuthId = common.String(value)
	}
	if value := strings.TrimSpace(spec.ExternalKey); value != "" {
		body.ExternalKey = common.String(value)
	}
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.Description); value != "" {
		body.Description = common.String(value)
	}
	if value := strings.TrimSpace(spec.DigitalTwinAdapterId); value != "" {
		body.DigitalTwinAdapterId = common.String(value)
	}
	if value := strings.TrimSpace(spec.DigitalTwinModelId); value != "" {
		body.DigitalTwinModelId = common.String(value)
	}
	if value := strings.TrimSpace(spec.DigitalTwinModelSpecUri); value != "" {
		body.DigitalTwinModelSpecUri = common.String(value)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneDigitalTwinInstanceStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = digitalTwinInstanceDefinedTags(spec.DefinedTags)
	}
}

func buildDigitalTwinInstanceUpdateBody(
	_ context.Context,
	resource *iotv1beta1.DigitalTwinInstance,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("DigitalTwinInstance resource is nil")
	}
	current, ok := digitalTwinInstanceBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current DigitalTwinInstance response does not expose a DigitalTwinInstance body")
	}

	body := iotsdk.UpdateDigitalTwinInstanceDetails{}
	updateNeeded := false
	setDigitalTwinInstanceStringUpdate(&body.AuthId, &updateNeeded, resource.Spec.AuthId, current.AuthId)
	setDigitalTwinInstanceStringUpdate(&body.ExternalKey, &updateNeeded, resource.Spec.ExternalKey, current.ExternalKey)
	setDigitalTwinInstanceStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	setDigitalTwinInstanceStringUpdate(&body.Description, &updateNeeded, resource.Spec.Description, current.Description)
	setDigitalTwinInstanceStringUpdate(&body.DigitalTwinAdapterId, &updateNeeded, resource.Spec.DigitalTwinAdapterId, current.DigitalTwinAdapterId)
	setDigitalTwinInstanceStringUpdate(&body.DigitalTwinModelId, &updateNeeded, resource.Spec.DigitalTwinModelId, current.DigitalTwinModelId)
	setDigitalTwinInstanceStringUpdate(&body.DigitalTwinModelSpecUri, &updateNeeded, resource.Spec.DigitalTwinModelSpecUri, current.DigitalTwinModelSpecUri)
	if resource.Spec.FreeformTags != nil {
		desired := cloneDigitalTwinInstanceStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := digitalTwinInstanceDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateDigitalTwinInstanceCreateOnlyDrift(
	resource *iotv1beta1.DigitalTwinInstance,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("digitalTwinInstance resource is nil")
	}
	current, ok := digitalTwinInstanceBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return rejectDigitalTwinInstanceCreateOnlyDrift(
		"iotDomainId",
		resource.Spec.IotDomainId,
		current.IotDomainId,
		resource.Status.IotDomainId,
	)
}

func rejectDigitalTwinInstanceCreateOnlyDrift(field string, desired string, observed *string, recorded string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(recorded)
	if observed != nil && strings.TrimSpace(*observed) != "" {
		current = strings.TrimSpace(*observed)
	}
	if desired == "" && current != "" {
		return fmt.Errorf("DigitalTwinInstance formal semantics require replacement when %s changes", field)
	}
	if desired != "" && current != "" && desired != current {
		return fmt.Errorf("DigitalTwinInstance formal semantics require replacement when %s changes", field)
	}
	return nil
}

func setDigitalTwinInstanceStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func digitalTwinInstanceBodyFromResponse(response any) (iotsdk.DigitalTwinInstance, bool) {
	if current, ok := digitalTwinInstanceMutationResponseBody(response); ok {
		return current, true
	}
	if current, ok := digitalTwinInstanceReadResponseBody(response); ok {
		return current, true
	}
	return digitalTwinInstanceSummaryResponseBody(response)
}

func digitalTwinInstanceMutationResponseBody(response any) (iotsdk.DigitalTwinInstance, bool) {
	switch current := response.(type) {
	case iotsdk.CreateDigitalTwinInstanceResponse:
		return current.DigitalTwinInstance, true
	case *iotsdk.CreateDigitalTwinInstanceResponse:
		return digitalTwinInstanceCreateResponseBody(current)
	case iotsdk.UpdateDigitalTwinInstanceResponse:
		return current.DigitalTwinInstance, true
	case *iotsdk.UpdateDigitalTwinInstanceResponse:
		return digitalTwinInstanceUpdateResponseBody(current)
	default:
		return iotsdk.DigitalTwinInstance{}, false
	}
}

func digitalTwinInstanceReadResponseBody(response any) (iotsdk.DigitalTwinInstance, bool) {
	switch current := response.(type) {
	case iotsdk.GetDigitalTwinInstanceResponse:
		return current.DigitalTwinInstance, true
	case *iotsdk.GetDigitalTwinInstanceResponse:
		return digitalTwinInstanceGetResponseBody(current)
	case iotsdk.DigitalTwinInstance:
		return current, true
	case *iotsdk.DigitalTwinInstance:
		return digitalTwinInstancePointerBody(current)
	default:
		return iotsdk.DigitalTwinInstance{}, false
	}
}

func digitalTwinInstanceSummaryResponseBody(response any) (iotsdk.DigitalTwinInstance, bool) {
	switch current := response.(type) {
	case iotsdk.DigitalTwinInstanceSummary:
		return digitalTwinInstanceFromSummary(current), true
	case *iotsdk.DigitalTwinInstanceSummary:
		return digitalTwinInstanceSummaryPointerBody(current)
	default:
		return iotsdk.DigitalTwinInstance{}, false
	}
}

func digitalTwinInstanceCreateResponseBody(
	response *iotsdk.CreateDigitalTwinInstanceResponse,
) (iotsdk.DigitalTwinInstance, bool) {
	if response == nil {
		return iotsdk.DigitalTwinInstance{}, false
	}
	return response.DigitalTwinInstance, true
}

func digitalTwinInstanceGetResponseBody(
	response *iotsdk.GetDigitalTwinInstanceResponse,
) (iotsdk.DigitalTwinInstance, bool) {
	if response == nil {
		return iotsdk.DigitalTwinInstance{}, false
	}
	return response.DigitalTwinInstance, true
}

func digitalTwinInstanceUpdateResponseBody(
	response *iotsdk.UpdateDigitalTwinInstanceResponse,
) (iotsdk.DigitalTwinInstance, bool) {
	if response == nil {
		return iotsdk.DigitalTwinInstance{}, false
	}
	return response.DigitalTwinInstance, true
}

func digitalTwinInstancePointerBody(response *iotsdk.DigitalTwinInstance) (iotsdk.DigitalTwinInstance, bool) {
	if response == nil {
		return iotsdk.DigitalTwinInstance{}, false
	}
	return *response, true
}

func digitalTwinInstanceSummaryPointerBody(response *iotsdk.DigitalTwinInstanceSummary) (iotsdk.DigitalTwinInstance, bool) {
	if response == nil {
		return iotsdk.DigitalTwinInstance{}, false
	}
	return digitalTwinInstanceFromSummary(*response), true
}

func digitalTwinInstanceFromSummary(summary iotsdk.DigitalTwinInstanceSummary) iotsdk.DigitalTwinInstance {
	return iotsdk.DigitalTwinInstance{
		Id:                      summary.Id,
		IotDomainId:             summary.IotDomainId,
		AuthId:                  summary.AuthId,
		ExternalKey:             summary.ExternalKey,
		DisplayName:             summary.DisplayName,
		Description:             summary.Description,
		DigitalTwinAdapterId:    summary.DigitalTwinAdapterId,
		DigitalTwinModelId:      summary.DigitalTwinModelId,
		DigitalTwinModelSpecUri: summary.DigitalTwinModelSpecUri,
		LifecycleState:          summary.LifecycleState,
		TimeCreated:             summary.TimeCreated,
		FreeformTags:            cloneDigitalTwinInstanceStringMap(summary.FreeformTags),
		DefinedTags:             cloneDigitalTwinInstanceDefinedTagMap(summary.DefinedTags),
		SystemTags:              cloneDigitalTwinInstanceDefinedTagMap(summary.SystemTags),
		TimeUpdated:             summary.TimeUpdated,
	}
}

func listDigitalTwinInstancesAllPages(
	call func(context.Context, iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error),
) func(context.Context, iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
	return func(ctx context.Context, request iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
		var combined iotsdk.ListDigitalTwinInstancesResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return iotsdk.ListDigitalTwinInstancesResponse{}, err
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

func handleDigitalTwinInstanceDeleteError(resource *iotv1beta1.DigitalTwinInstance, err error) error {
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
	return digitalTwinInstanceAmbiguousNotFoundError{
		message:      "DigitalTwinInstance delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapDigitalTwinInstanceDeleteConfirmation(hooks *DigitalTwinInstanceRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getDigitalTwinInstance := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DigitalTwinInstanceServiceClient) DigitalTwinInstanceServiceClient {
		return digitalTwinInstanceDeleteConfirmationClient{
			delegate:               delegate,
			getDigitalTwinInstance: getDigitalTwinInstance,
		}
	})
}

type digitalTwinInstanceDeleteConfirmationClient struct {
	delegate               DigitalTwinInstanceServiceClient
	getDigitalTwinInstance func(context.Context, iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error)
}

func (c digitalTwinInstanceDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c digitalTwinInstanceDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinInstance,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c digitalTwinInstanceDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinInstance,
) error {
	if c.getDigitalTwinInstance == nil || resource == nil {
		return nil
	}
	instanceID := trackedDigitalTwinInstanceID(resource)
	if instanceID == "" {
		return nil
	}
	_, err := c.getDigitalTwinInstance(ctx, iotsdk.GetDigitalTwinInstanceRequest{DigitalTwinInstanceId: common.String(instanceID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("DigitalTwinInstance delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedDigitalTwinInstanceID(resource *iotv1beta1.DigitalTwinInstance) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func applyDigitalTwinInstanceDeleteOutcome(
	resource *iotv1beta1.DigitalTwinInstance,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := digitalTwinInstanceLifecycleState(response)
	if state == "" || state == string(iotsdk.LifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if state != string(iotsdk.LifecycleStateActive) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !digitalTwinInstanceDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markDigitalTwinInstanceTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func digitalTwinInstanceDeleteAlreadyPending(resource *iotv1beta1.DigitalTwinInstance) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markDigitalTwinInstanceTerminating(resource *iotv1beta1.DigitalTwinInstance, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = digitalTwinInstanceDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := digitalTwinInstanceLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         digitalTwinInstanceDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		digitalTwinInstanceDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func digitalTwinInstanceLifecycleState(response any) string {
	current, ok := digitalTwinInstanceBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneDigitalTwinInstanceStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func digitalTwinInstanceDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
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

func cloneDigitalTwinInstanceDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
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
