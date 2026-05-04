/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwinrelationship

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

const digitalTwinRelationshipDeletePendingMessage = "OCI DigitalTwinRelationship delete is in progress"

type digitalTwinRelationshipOCIClient interface {
	CreateDigitalTwinRelationship(context.Context, iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error)
	GetDigitalTwinRelationship(context.Context, iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error)
	ListDigitalTwinRelationships(context.Context, iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error)
	UpdateDigitalTwinRelationship(context.Context, iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error)
	DeleteDigitalTwinRelationship(context.Context, iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error)
}

type digitalTwinRelationshipAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e digitalTwinRelationshipAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e digitalTwinRelationshipAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerDigitalTwinRelationshipRuntimeHooksMutator(func(_ *DigitalTwinRelationshipServiceManager, hooks *DigitalTwinRelationshipRuntimeHooks) {
		applyDigitalTwinRelationshipRuntimeHooks(hooks)
	})
}

func applyDigitalTwinRelationshipRuntimeHooks(hooks *DigitalTwinRelationshipRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = digitalTwinRelationshipRuntimeSemantics()
	hooks.BuildCreateBody = buildDigitalTwinRelationshipCreateBody
	hooks.BuildUpdateBody = buildDigitalTwinRelationshipUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardDigitalTwinRelationshipExistingBeforeCreate
	hooks.List.Fields = digitalTwinRelationshipListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDigitalTwinRelationshipsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDigitalTwinRelationshipCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleDigitalTwinRelationshipDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDigitalTwinRelationshipDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markDigitalTwinRelationshipTerminating
	wrapDigitalTwinRelationshipDeleteConfirmation(hooks)
}

func newDigitalTwinRelationshipServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client digitalTwinRelationshipOCIClient,
) DigitalTwinRelationshipServiceClient {
	manager := &DigitalTwinRelationshipServiceManager{Log: log}
	hooks := newDigitalTwinRelationshipRuntimeHooksWithOCIClient(client)
	applyDigitalTwinRelationshipRuntimeHooks(&hooks)
	delegate := defaultDigitalTwinRelationshipServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*iotv1beta1.DigitalTwinRelationship](
			buildDigitalTwinRelationshipGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDigitalTwinRelationshipGeneratedClient(hooks, delegate)
}

func newDigitalTwinRelationshipRuntimeHooksWithOCIClient(client digitalTwinRelationshipOCIClient) DigitalTwinRelationshipRuntimeHooks {
	hooks := newDigitalTwinRelationshipDefaultRuntimeHooks(iotsdk.IotClient{})
	hooks.Create.Call = func(ctx context.Context, request iotsdk.CreateDigitalTwinRelationshipRequest) (iotsdk.CreateDigitalTwinRelationshipResponse, error) {
		if client == nil {
			return iotsdk.CreateDigitalTwinRelationshipResponse{}, fmt.Errorf("DigitalTwinRelationship OCI client is not configured")
		}
		return client.CreateDigitalTwinRelationship(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error) {
		if client == nil {
			return iotsdk.GetDigitalTwinRelationshipResponse{}, fmt.Errorf("DigitalTwinRelationship OCI client is not configured")
		}
		return client.GetDigitalTwinRelationship(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
		if client == nil {
			return iotsdk.ListDigitalTwinRelationshipsResponse{}, fmt.Errorf("DigitalTwinRelationship OCI client is not configured")
		}
		return client.ListDigitalTwinRelationships(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request iotsdk.UpdateDigitalTwinRelationshipRequest) (iotsdk.UpdateDigitalTwinRelationshipResponse, error) {
		if client == nil {
			return iotsdk.UpdateDigitalTwinRelationshipResponse{}, fmt.Errorf("DigitalTwinRelationship OCI client is not configured")
		}
		return client.UpdateDigitalTwinRelationship(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request iotsdk.DeleteDigitalTwinRelationshipRequest) (iotsdk.DeleteDigitalTwinRelationshipResponse, error) {
		if client == nil {
			return iotsdk.DeleteDigitalTwinRelationshipResponse{}, fmt.Errorf("DigitalTwinRelationship OCI client is not configured")
		}
		return client.DeleteDigitalTwinRelationship(ctx, request)
	}
	return hooks
}

func digitalTwinRelationshipRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "iot",
		FormalSlug:    "digitaltwinrelationship",
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
				"contentPath",
				"sourceDigitalTwinInstanceId",
				"targetDigitalTwinInstanceId",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"content",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"iotDomainId",
				"contentPath",
				"sourceDigitalTwinInstanceId",
				"targetDigitalTwinInstanceId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinRelationship", Action: "CreateDigitalTwinRelationship"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinRelationship", Action: "UpdateDigitalTwinRelationship"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinRelationship", Action: "DeleteDigitalTwinRelationship"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DigitalTwinRelationship", Action: "GetDigitalTwinRelationship"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DigitalTwinRelationship", Action: "GetDigitalTwinRelationship"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DigitalTwinRelationship", Action: "GetDigitalTwinRelationship"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func digitalTwinRelationshipListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "IotDomainId",
			RequestName:  "iotDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.iotDomainId", "spec.iotDomainId", "iotDomainId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "ContentPath",
			RequestName:  "contentPath",
			Contribution: "query",
			LookupPaths:  []string{"status.contentPath", "spec.contentPath", "contentPath"},
		},
		{
			FieldName:    "SourceDigitalTwinInstanceId",
			RequestName:  "sourceDigitalTwinInstanceId",
			Contribution: "query",
			LookupPaths:  []string{"status.sourceDigitalTwinInstanceId", "spec.sourceDigitalTwinInstanceId", "sourceDigitalTwinInstanceId"},
		},
		{
			FieldName:    "TargetDigitalTwinInstanceId",
			RequestName:  "targetDigitalTwinInstanceId",
			Contribution: "query",
			LookupPaths:  []string{"status.targetDigitalTwinInstanceId", "spec.targetDigitalTwinInstanceId", "targetDigitalTwinInstanceId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func guardDigitalTwinRelationshipExistingBeforeCreate(
	_ context.Context,
	resource *iotv1beta1.DigitalTwinRelationship,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if trackedDigitalTwinRelationshipID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	for _, value := range []string{
		resource.Spec.IotDomainId,
		resource.Spec.ContentPath,
		resource.Spec.SourceDigitalTwinInstanceId,
		resource.Spec.TargetDigitalTwinInstanceId,
	} {
		if strings.TrimSpace(value) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildDigitalTwinRelationshipCreateBody(_ context.Context, resource *iotv1beta1.DigitalTwinRelationship, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("DigitalTwinRelationship resource is nil")
	}
	if strings.TrimSpace(resource.Spec.IotDomainId) == "" {
		return nil, fmt.Errorf("iotDomainId is required")
	}
	if strings.TrimSpace(resource.Spec.ContentPath) == "" {
		return nil, fmt.Errorf("contentPath is required")
	}
	if strings.TrimSpace(resource.Spec.SourceDigitalTwinInstanceId) == "" {
		return nil, fmt.Errorf("sourceDigitalTwinInstanceId is required")
	}
	if strings.TrimSpace(resource.Spec.TargetDigitalTwinInstanceId) == "" {
		return nil, fmt.Errorf("targetDigitalTwinInstanceId is required")
	}

	body := iotsdk.CreateDigitalTwinRelationshipDetails{
		IotDomainId:                 common.String(strings.TrimSpace(resource.Spec.IotDomainId)),
		ContentPath:                 common.String(strings.TrimSpace(resource.Spec.ContentPath)),
		SourceDigitalTwinInstanceId: common.String(strings.TrimSpace(resource.Spec.SourceDigitalTwinInstanceId)),
		TargetDigitalTwinInstanceId: common.String(strings.TrimSpace(resource.Spec.TargetDigitalTwinInstanceId)),
	}
	if err := setDigitalTwinRelationshipMutableCreateFields(&body, resource.Spec); err != nil {
		return nil, err
	}
	return body, nil
}

func setDigitalTwinRelationshipMutableCreateFields(
	body *iotsdk.CreateDigitalTwinRelationshipDetails,
	spec iotv1beta1.DigitalTwinRelationshipSpec,
) error {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.Description); value != "" {
		body.Description = common.String(value)
	}
	content, err := digitalTwinRelationshipContent(spec.Content)
	if err != nil {
		return err
	}
	if content != nil {
		body.Content = content
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneDigitalTwinRelationshipStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = digitalTwinRelationshipDefinedTags(spec.DefinedTags)
	}
	return nil
}

func buildDigitalTwinRelationshipUpdateBody(
	_ context.Context,
	resource *iotv1beta1.DigitalTwinRelationship,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("DigitalTwinRelationship resource is nil")
	}
	current, ok := digitalTwinRelationshipBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current DigitalTwinRelationship response does not expose a DigitalTwinRelationship body")
	}

	body := iotsdk.UpdateDigitalTwinRelationshipDetails{}
	updateNeeded := false
	setDigitalTwinRelationshipStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	setDigitalTwinRelationshipStringUpdate(&body.Description, &updateNeeded, resource.Spec.Description, current.Description)
	if resource.Spec.Content != nil {
		desired, err := digitalTwinRelationshipContent(resource.Spec.Content)
		if err != nil {
			return nil, false, err
		}
		body.Content = desired
		if !reflect.DeepEqual(current.Content, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.FreeformTags != nil {
		desired := cloneDigitalTwinRelationshipStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := digitalTwinRelationshipDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateDigitalTwinRelationshipCreateOnlyDrift(
	resource *iotv1beta1.DigitalTwinRelationship,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("digitalTwinRelationship resource is nil")
	}
	current, ok := digitalTwinRelationshipBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	for _, check := range []struct {
		field    string
		desired  string
		observed *string
		recorded string
	}{
		{"iotDomainId", resource.Spec.IotDomainId, current.IotDomainId, resource.Status.IotDomainId},
		{"contentPath", resource.Spec.ContentPath, current.ContentPath, resource.Status.ContentPath},
		{"sourceDigitalTwinInstanceId", resource.Spec.SourceDigitalTwinInstanceId, current.SourceDigitalTwinInstanceId, resource.Status.SourceDigitalTwinInstanceId},
		{"targetDigitalTwinInstanceId", resource.Spec.TargetDigitalTwinInstanceId, current.TargetDigitalTwinInstanceId, resource.Status.TargetDigitalTwinInstanceId},
	} {
		if err := rejectDigitalTwinRelationshipCreateOnlyDrift(check.field, check.desired, check.observed, check.recorded); err != nil {
			return err
		}
	}
	return nil
}

func rejectDigitalTwinRelationshipCreateOnlyDrift(field string, desired string, observed *string, recorded string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(recorded)
	if observed != nil && strings.TrimSpace(*observed) != "" {
		current = strings.TrimSpace(*observed)
	}
	if desired == "" && current != "" {
		return fmt.Errorf("DigitalTwinRelationship formal semantics require replacement when %s changes", field)
	}
	if desired != "" && current != "" && desired != current {
		return fmt.Errorf("DigitalTwinRelationship formal semantics require replacement when %s changes", field)
	}
	return nil
}

func setDigitalTwinRelationshipStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func digitalTwinRelationshipBodyFromResponse(response any) (iotsdk.DigitalTwinRelationship, bool) {
	if current, ok := digitalTwinRelationshipMutationResponseBody(response); ok {
		return current, true
	}
	if current, ok := digitalTwinRelationshipReadResponseBody(response); ok {
		return current, true
	}
	return digitalTwinRelationshipSummaryResponseBody(response)
}

func digitalTwinRelationshipMutationResponseBody(response any) (iotsdk.DigitalTwinRelationship, bool) {
	switch current := response.(type) {
	case iotsdk.CreateDigitalTwinRelationshipResponse:
		return current.DigitalTwinRelationship, true
	case *iotsdk.CreateDigitalTwinRelationshipResponse:
		return digitalTwinRelationshipCreateResponseBody(current)
	case iotsdk.UpdateDigitalTwinRelationshipResponse:
		return current.DigitalTwinRelationship, true
	case *iotsdk.UpdateDigitalTwinRelationshipResponse:
		return digitalTwinRelationshipUpdateResponseBody(current)
	default:
		return iotsdk.DigitalTwinRelationship{}, false
	}
}

func digitalTwinRelationshipReadResponseBody(response any) (iotsdk.DigitalTwinRelationship, bool) {
	switch current := response.(type) {
	case iotsdk.GetDigitalTwinRelationshipResponse:
		return current.DigitalTwinRelationship, true
	case *iotsdk.GetDigitalTwinRelationshipResponse:
		return digitalTwinRelationshipGetResponseBody(current)
	case iotsdk.DigitalTwinRelationship:
		return current, true
	case *iotsdk.DigitalTwinRelationship:
		return digitalTwinRelationshipPointerBody(current)
	default:
		return iotsdk.DigitalTwinRelationship{}, false
	}
}

func digitalTwinRelationshipSummaryResponseBody(response any) (iotsdk.DigitalTwinRelationship, bool) {
	switch current := response.(type) {
	case iotsdk.DigitalTwinRelationshipSummary:
		return digitalTwinRelationshipFromSummary(current), true
	case *iotsdk.DigitalTwinRelationshipSummary:
		return digitalTwinRelationshipSummaryPointerBody(current)
	default:
		return iotsdk.DigitalTwinRelationship{}, false
	}
}

func digitalTwinRelationshipCreateResponseBody(
	response *iotsdk.CreateDigitalTwinRelationshipResponse,
) (iotsdk.DigitalTwinRelationship, bool) {
	if response == nil {
		return iotsdk.DigitalTwinRelationship{}, false
	}
	return response.DigitalTwinRelationship, true
}

func digitalTwinRelationshipGetResponseBody(
	response *iotsdk.GetDigitalTwinRelationshipResponse,
) (iotsdk.DigitalTwinRelationship, bool) {
	if response == nil {
		return iotsdk.DigitalTwinRelationship{}, false
	}
	return response.DigitalTwinRelationship, true
}

func digitalTwinRelationshipUpdateResponseBody(
	response *iotsdk.UpdateDigitalTwinRelationshipResponse,
) (iotsdk.DigitalTwinRelationship, bool) {
	if response == nil {
		return iotsdk.DigitalTwinRelationship{}, false
	}
	return response.DigitalTwinRelationship, true
}

func digitalTwinRelationshipPointerBody(response *iotsdk.DigitalTwinRelationship) (iotsdk.DigitalTwinRelationship, bool) {
	if response == nil {
		return iotsdk.DigitalTwinRelationship{}, false
	}
	return *response, true
}

func digitalTwinRelationshipSummaryPointerBody(response *iotsdk.DigitalTwinRelationshipSummary) (iotsdk.DigitalTwinRelationship, bool) {
	if response == nil {
		return iotsdk.DigitalTwinRelationship{}, false
	}
	return digitalTwinRelationshipFromSummary(*response), true
}

func digitalTwinRelationshipFromSummary(summary iotsdk.DigitalTwinRelationshipSummary) iotsdk.DigitalTwinRelationship {
	return iotsdk.DigitalTwinRelationship{
		Id:                          summary.Id,
		IotDomainId:                 summary.IotDomainId,
		DisplayName:                 summary.DisplayName,
		ContentPath:                 summary.ContentPath,
		SourceDigitalTwinInstanceId: summary.SourceDigitalTwinInstanceId,
		TargetDigitalTwinInstanceId: summary.TargetDigitalTwinInstanceId,
		LifecycleState:              summary.LifecycleState,
		TimeCreated:                 summary.TimeCreated,
		Description:                 summary.Description,
		FreeformTags:                cloneDigitalTwinRelationshipStringMap(summary.FreeformTags),
		DefinedTags:                 cloneDigitalTwinRelationshipDefinedTagMap(summary.DefinedTags),
		SystemTags:                  cloneDigitalTwinRelationshipDefinedTagMap(summary.SystemTags),
		TimeUpdated:                 summary.TimeUpdated,
	}
}

func listDigitalTwinRelationshipsAllPages(
	call func(context.Context, iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error),
) func(context.Context, iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
	return func(ctx context.Context, request iotsdk.ListDigitalTwinRelationshipsRequest) (iotsdk.ListDigitalTwinRelationshipsResponse, error) {
		var combined iotsdk.ListDigitalTwinRelationshipsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return iotsdk.ListDigitalTwinRelationshipsResponse{}, err
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

func handleDigitalTwinRelationshipDeleteError(resource *iotv1beta1.DigitalTwinRelationship, err error) error {
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
	return digitalTwinRelationshipAmbiguousNotFoundError{
		message:      "DigitalTwinRelationship delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapDigitalTwinRelationshipDeleteConfirmation(hooks *DigitalTwinRelationshipRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getDigitalTwinRelationship := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DigitalTwinRelationshipServiceClient) DigitalTwinRelationshipServiceClient {
		return digitalTwinRelationshipDeleteConfirmationClient{
			delegate:                   delegate,
			getDigitalTwinRelationship: getDigitalTwinRelationship,
		}
	})
}

type digitalTwinRelationshipDeleteConfirmationClient struct {
	delegate                   DigitalTwinRelationshipServiceClient
	getDigitalTwinRelationship func(context.Context, iotsdk.GetDigitalTwinRelationshipRequest) (iotsdk.GetDigitalTwinRelationshipResponse, error)
}

func (c digitalTwinRelationshipDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinRelationship,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c digitalTwinRelationshipDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinRelationship,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c digitalTwinRelationshipDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *iotv1beta1.DigitalTwinRelationship,
) error {
	if c.getDigitalTwinRelationship == nil || resource == nil {
		return nil
	}
	relationshipID := trackedDigitalTwinRelationshipID(resource)
	if relationshipID == "" {
		return nil
	}
	_, err := c.getDigitalTwinRelationship(ctx, iotsdk.GetDigitalTwinRelationshipRequest{DigitalTwinRelationshipId: common.String(relationshipID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("DigitalTwinRelationship delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedDigitalTwinRelationshipID(resource *iotv1beta1.DigitalTwinRelationship) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func applyDigitalTwinRelationshipDeleteOutcome(
	resource *iotv1beta1.DigitalTwinRelationship,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := digitalTwinRelationshipLifecycleState(response)
	if state == "" || state == string(iotsdk.LifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if state != string(iotsdk.LifecycleStateActive) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !digitalTwinRelationshipDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markDigitalTwinRelationshipTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func digitalTwinRelationshipDeleteAlreadyPending(resource *iotv1beta1.DigitalTwinRelationship) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markDigitalTwinRelationshipTerminating(resource *iotv1beta1.DigitalTwinRelationship, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = digitalTwinRelationshipDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := digitalTwinRelationshipLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         digitalTwinRelationshipDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		digitalTwinRelationshipDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func digitalTwinRelationshipLifecycleState(response any) string {
	current, ok := digitalTwinRelationshipBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func digitalTwinRelationshipContent(source map[string]shared.JSONValue) (map[string]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	converted := make(map[string]interface{}, len(source))
	for key, value := range source {
		var decoded interface{}
		if len(value.Raw) > 0 {
			if err := json.Unmarshal(value.Raw, &decoded); err != nil {
				return nil, fmt.Errorf("decode content[%s]: %w", key, err)
			}
		}
		converted[key] = decoded
	}
	return converted, nil
}

func cloneDigitalTwinRelationshipStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func digitalTwinRelationshipDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
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

func cloneDigitalTwinRelationshipDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
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
