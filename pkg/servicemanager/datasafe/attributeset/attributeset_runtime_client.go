/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package attributeset

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const attributeSetRequeueDuration = time.Minute

var attributeSetWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(datasafesdk.WorkRequestStatusAccepted),
		string(datasafesdk.WorkRequestStatusInProgress),
		string(datasafesdk.WorkRequestStatusCanceling),
		string(datasafesdk.WorkRequestStatusSuspending),
	},
	SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(datasafesdk.WorkRequestStatusSuspended)},
	CreateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeCreateAttributeSet)},
	UpdateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeUpdateAttributeSet),
		string(datasafesdk.WorkRequestOperationTypeChangeAttributeSetCompartment),
	},
	DeleteActionTokens: []string{string(datasafesdk.WorkRequestOperationTypeDeleteAttributeSet)},
}

type attributeSetOCIClient interface {
	CreateAttributeSet(context.Context, datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error)
	GetAttributeSet(context.Context, datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error)
	ListAttributeSets(context.Context, datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error)
	UpdateAttributeSet(context.Context, datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error)
	DeleteAttributeSet(context.Context, datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error)
	ChangeAttributeSetCompartment(context.Context, datasafesdk.ChangeAttributeSetCompartmentRequest) (datasafesdk.ChangeAttributeSetCompartmentResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

func init() {
	registerAttributeSetRuntimeHooksMutator(func(manager *AttributeSetServiceManager, hooks *AttributeSetRuntimeHooks) {
		client, initErr := newAttributeSetOCIClient(manager)
		applyAttributeSetRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newAttributeSetOCIClient(manager *AttributeSetServiceManager) (attributeSetOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("AttributeSet service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAttributeSetRuntimeHooks(
	manager *AttributeSetServiceManager,
	hooks *AttributeSetRuntimeHooks,
	client attributeSetOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAttributeSetRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *datasafev1beta1.AttributeSet, _ string) (any, error) {
		return buildAttributeSetCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *datasafev1beta1.AttributeSet,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildAttributeSetUpdateBody(resource, currentResponse)
	}
	applyAttributeSetOperationHooks(hooks, client, initErr)
	hooks.List.Fields = attributeSetListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedAttributeSetIdentity
	hooks.ParityHooks.RequiresParityHandling = attributeSetRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *datasafev1beta1.AttributeSet,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyAttributeSetCompartmentMove(ctx, resource, currentResponse, client, initErr)
	}
	hooks.Async.Adapter = attributeSetWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getAttributeSetWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveAttributeSetWorkRequestAction
	hooks.Async.RecoverResourceID = recoverAttributeSetIDFromWorkRequest
	hooks.Async.Message = attributeSetWorkRequestMessage
	hooks.DeleteHooks.HandleError = rejectAttributeSetAuthShapedNotFound
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AttributeSetServiceClient) AttributeSetServiceClient {
		return attributeSetDeleteGuardClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	})
}

func applyAttributeSetOperationHooks(hooks *AttributeSetRuntimeHooks, client attributeSetOCIClient, initErr error) {
	hooks.Create.Call = func(ctx context.Context, request datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error) {
		if err := requireAttributeSetOCIClient(client, initErr); err != nil {
			return datasafesdk.CreateAttributeSetResponse{}, err
		}
		return client.CreateAttributeSet(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
		if err := requireAttributeSetOCIClient(client, initErr); err != nil {
			return datasafesdk.GetAttributeSetResponse{}, err
		}
		return client.GetAttributeSet(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error) {
		return listAttributeSetsAllPages(ctx, client, initErr, request)
	}
	hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error) {
		if err := requireAttributeSetOCIClient(client, initErr); err != nil {
			return datasafesdk.UpdateAttributeSetResponse{}, err
		}
		return client.UpdateAttributeSet(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
		if err := requireAttributeSetOCIClient(client, initErr); err != nil {
			return datasafesdk.DeleteAttributeSetResponse{}, err
		}
		return client.DeleteAttributeSet(ctx, request)
	}
}

func requireAttributeSetOCIClient(client attributeSetOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize AttributeSet OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("AttributeSet OCI client is not configured")
	}
	return nil
}

func newAttributeSetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "attributeset",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{
					"create",
					"update",
					"delete",
				},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.AttributeSetLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.AttributeSetLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.AttributeSetLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.AttributeSetLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"attributeSetType",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"attributeSetValues",
				"compartmentId",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
			},
			ForceNew: []string{
				"attributeSetType",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AttributeSet", Action: "CreateAttributeSet"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AttributeSet", Action: "UpdateAttributeSet"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AttributeSet", Action: "DeleteAttributeSet"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "AttributeSet", Action: "CreateAttributeSet"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "AttributeSet", Action: "UpdateAttributeSet"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "AttributeSet", Action: "DeleteAttributeSet"}},
		},
	}
}

func attributeSetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "status.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"spec.displayName", "status.displayName", "displayName"},
		},
		{
			FieldName:    "AttributeSetType",
			RequestName:  "attributeSetType",
			Contribution: "query",
			LookupPaths:  []string{"spec.attributeSetType", "status.attributeSetType", "attributeSetType"},
		},
		{FieldName: "AttributeSetId", RequestName: "attributeSetId", Contribution: "query", PreferResourceID: true},
	}
}

func buildAttributeSetCreateBody(resource *datasafev1beta1.AttributeSet) (datasafesdk.CreateAttributeSetDetails, error) {
	if resource == nil {
		return datasafesdk.CreateAttributeSetDetails{}, fmt.Errorf("AttributeSet resource is nil")
	}

	details := datasafesdk.CreateAttributeSetDetails{
		DisplayName:        common.String(resource.Spec.DisplayName),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		AttributeSetType:   datasafesdk.AttributeSetAttributeSetTypeEnum(resource.Spec.AttributeSetType),
		AttributeSetValues: cloneAttributeSetValues(resource.Spec.AttributeSetValues),
	}
	if strings.TrimSpace(resource.Spec.Description) != "" {
		details.Description = common.String(resource.Spec.Description)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneAttributeSetStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = attributeSetDefinedTags(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildAttributeSetUpdateBody(
	resource *datasafev1beta1.AttributeSet,
	currentResponse any,
) (datasafesdk.UpdateAttributeSetDetails, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateAttributeSetDetails{}, false, fmt.Errorf("AttributeSet resource is nil")
	}
	current, ok := attributeSetFromResponse(currentResponse)
	if !ok {
		return datasafesdk.UpdateAttributeSetDetails{}, false, fmt.Errorf("current AttributeSet response does not expose an AttributeSet body")
	}

	details := datasafesdk.UpdateAttributeSetDetails{}
	return details, attributeSetAnyUpdateNeeded(
		applyAttributeSetDisplayNameUpdate(&details, resource, current),
		applyAttributeSetDescriptionUpdate(&details, resource, current),
		applyAttributeSetValuesUpdate(&details, resource, current),
		attributeSetCompartmentIDDrift(resource, current),
		applyAttributeSetFreeformTagsUpdate(&details, resource, current),
		applyAttributeSetDefinedTagsUpdate(&details, resource, current),
	), nil
}

func applyAttributeSetDisplayNameUpdate(
	details *datasafesdk.UpdateAttributeSetDetails,
	resource *datasafev1beta1.AttributeSet,
	current datasafesdk.AttributeSet,
) bool {
	if attributeSetStringValue(current.DisplayName) == strings.TrimSpace(resource.Spec.DisplayName) {
		return false
	}
	details.DisplayName = common.String(resource.Spec.DisplayName)
	return true
}

func applyAttributeSetDescriptionUpdate(
	details *datasafesdk.UpdateAttributeSetDetails,
	resource *datasafev1beta1.AttributeSet,
	current datasafesdk.AttributeSet,
) bool {
	desired := resource.Spec.Description
	observed := attributeSetStringValue(current.Description)
	if strings.TrimSpace(desired) == "" {
		return false
	}
	if observed == desired {
		return false
	}
	details.Description = common.String(desired)
	return true
}

func applyAttributeSetValuesUpdate(
	details *datasafesdk.UpdateAttributeSetDetails,
	resource *datasafev1beta1.AttributeSet,
	current datasafesdk.AttributeSet,
) bool {
	if reflect.DeepEqual(current.AttributeSetValues, resource.Spec.AttributeSetValues) {
		return false
	}
	details.AttributeSetValues = cloneAttributeSetValues(resource.Spec.AttributeSetValues)
	return true
}

func applyAttributeSetFreeformTagsUpdate(
	details *datasafesdk.UpdateAttributeSetDetails,
	resource *datasafev1beta1.AttributeSet,
	current datasafesdk.AttributeSet,
) bool {
	if resource.Spec.FreeformTags == nil || reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		return false
	}
	details.FreeformTags = cloneAttributeSetStringMap(resource.Spec.FreeformTags)
	return true
}

func applyAttributeSetDefinedTagsUpdate(
	details *datasafesdk.UpdateAttributeSetDetails,
	resource *datasafev1beta1.AttributeSet,
	current datasafesdk.AttributeSet,
) bool {
	if resource.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := attributeSetDefinedTags(resource.Spec.DefinedTags)
	if reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		return false
	}
	details.DefinedTags = desiredDefinedTags
	return true
}

func attributeSetAnyUpdateNeeded(changes ...bool) bool {
	for _, changed := range changes {
		if changed {
			return true
		}
	}
	return false
}

func listAttributeSetsAllPages(
	ctx context.Context,
	client attributeSetOCIClient,
	initErr error,
	request datasafesdk.ListAttributeSetsRequest,
) (datasafesdk.ListAttributeSetsResponse, error) {
	if err := requireAttributeSetOCIClient(client, initErr); err != nil {
		return datasafesdk.ListAttributeSetsResponse{}, err
	}
	if request.IsUserDefined == nil {
		request.IsUserDefined = common.Bool(true)
	}

	seenPages := map[string]struct{}{}
	var combined datasafesdk.ListAttributeSetsResponse
	for {
		response, err := client.ListAttributeSets(ctx, request)
		if err != nil {
			return datasafesdk.ListAttributeSetsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := attributeSetStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return datasafesdk.ListAttributeSetsResponse{}, fmt.Errorf("AttributeSet list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func attributeSetRequiresCompartmentMove(resource *datasafev1beta1.AttributeSet, currentResponse any) bool {
	if resource == nil {
		return false
	}
	current, ok := attributeSetFromResponse(currentResponse)
	if !ok {
		return false
	}
	return attributeSetCompartmentIDDrift(resource, current)
}

func attributeSetCompartmentIDDrift(resource *datasafev1beta1.AttributeSet, current datasafesdk.AttributeSet) bool {
	if resource == nil {
		return false
	}
	desired := strings.TrimSpace(resource.Spec.CompartmentId)
	observed := strings.TrimSpace(attributeSetStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyAttributeSetCompartmentMove(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
	currentResponse any,
	client attributeSetOCIClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AttributeSet resource is nil")
	}
	if err := requireAttributeSetOCIClient(client, initErr); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	current, ok := attributeSetFromResponse(currentResponse)
	if !ok {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("current AttributeSet response does not expose an AttributeSet body")
	}
	attributeSetID := strings.TrimSpace(attributeSetStringValue(current.Id))
	if attributeSetID == "" {
		attributeSetID = currentAttributeSetID(resource)
	}
	if attributeSetID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AttributeSet compartment move requires a tracked AttributeSet id")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AttributeSet compartment move requires spec.compartmentId")
	}

	response, err := client.ChangeAttributeSetCompartment(ctx, datasafesdk.ChangeAttributeSetCompartmentRequest{
		AttributeSetId: common.String(attributeSetID),
		ChangeAttributeSetCompartmentDetails: datasafesdk.ChangeAttributeSetCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
		OpcRetryToken: attributeSetCompartmentMoveRetryToken(resource, compartmentID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	workRequestID := strings.TrimSpace(attributeSetStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AttributeSet compartment move did not return an opc-work-request-id")
	}

	resource.Status.OsokStatus.Ocid = shared.OCID(attributeSetID)
	currentAsync, err := servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, attributeSetWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawAction:        string(datasafesdk.WorkRequestOperationTypeChangeAttributeSetCompartment),
		RawOperationType: string(datasafesdk.WorkRequestOperationTypeChangeAttributeSetCompartment),
		WorkRequestID:    workRequestID,
		FallbackPhase:    shared.OSOKAsyncPhaseUpdate,
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: attributeSetRequeueDuration,
	}, nil
}

func getAttributeSetWorkRequest(
	ctx context.Context,
	client attributeSetOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if err := requireAttributeSetOCIClient(client, initErr); err != nil {
		return nil, err
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{WorkRequestId: common.String(workRequestID)})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveAttributeSetWorkRequestAction(workRequest any) (string, error) {
	current, ok := attributeSetWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("AttributeSet work request response did not expose a WorkRequest body")
	}
	return string(current.OperationType), nil
}

func recoverAttributeSetIDFromWorkRequest(
	_ *datasafev1beta1.AttributeSet,
	workRequest any,
	_ shared.OSOKAsyncPhase,
) (string, error) {
	current, ok := attributeSetWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("AttributeSet work request response did not expose a WorkRequest body")
	}
	for _, resource := range current.Resources {
		identifier := strings.TrimSpace(attributeSetStringValue(resource.Identifier))
		if identifier == "" {
			continue
		}
		entityType := strings.ToLower(strings.TrimSpace(attributeSetStringValue(resource.EntityType)))
		if entityType == "" || strings.Contains(entityType, "attributeset") || strings.Contains(entityType, "attribute_set") {
			return identifier, nil
		}
	}
	return "", nil
}

func attributeSetWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, ok := attributeSetWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	workRequestID := strings.TrimSpace(attributeSetStringValue(current.Id))
	status := strings.TrimSpace(string(current.Status))
	if workRequestID == "" || status == "" {
		return ""
	}
	return fmt.Sprintf("AttributeSet %s work request %s is %s", phase, workRequestID, status)
}

func attributeSetWorkRequestFromAny(value any) (datasafesdk.WorkRequest, bool) {
	switch typed := value.(type) {
	case datasafesdk.GetWorkRequestResponse:
		return typed.WorkRequest, true
	case datasafesdk.WorkRequest:
		return typed, true
	default:
		return datasafesdk.WorkRequest{}, false
	}
}

type attributeSetDeleteGuardClient struct {
	delegate AttributeSetServiceClient
	client   attributeSetOCIClient
	initErr  error
}

var _ AttributeSetServiceClient = attributeSetDeleteGuardClient{}

func (c attributeSetDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, request)
}

func (c attributeSetDeleteGuardClient) Delete(ctx context.Context, resource *datasafev1beta1.AttributeSet) (bool, error) {
	if resource == nil {
		return c.delegate.Delete(ctx, resource)
	}
	if workRequestID, phase := pendingAttributeSetWriteWorkRequest(resource); workRequestID != "" {
		resolved, err := c.resumeWriteWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
		if err != nil || !resolved {
			return false, err
		}
	}
	if workRequestID := pendingAttributeSetDeleteWorkRequestID(resource); workRequestID != "" {
		return c.resumeDeleteWorkRequest(ctx, resource, workRequestID)
	}
	currentID := currentAttributeSetID(resource)
	if currentID == "" {
		return c.delegate.Delete(ctx, resource)
	}
	if handled, deleted, err := c.confirmPreDeleteRead(ctx, resource, currentID); handled || err != nil {
		return deleted, err
	}
	return c.startDeleteWorkRequest(ctx, resource, currentID)
}

func (c attributeSetDeleteGuardClient) resumeWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	workRequest, err := getAttributeSetWorkRequest(ctx, c.client, c.initErr, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	currentAsync, err := buildAttributeSetWorkRequestAsyncOperation(resource, workRequest, workRequestID, phase)
	if err != nil {
		return false, err
	}
	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		recordAttributeSetIDFromWorkRequest(resource, workRequest, phase)
		return true, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())
		return false, fmt.Errorf("AttributeSet %s work request %s finished with status %s", phase, workRequestID, currentAsync.RawStatus)
	default:
		return false, fmt.Errorf("AttributeSet %s work request %s projected unsupported async class %s", phase, workRequestID, currentAsync.NormalizedClass)
	}
}

func (c attributeSetDeleteGuardClient) startDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
	attributeSetID string,
) (bool, error) {
	if err := requireAttributeSetOCIClient(c.client, c.initErr); err != nil {
		return false, err
	}

	response, err := c.client.DeleteAttributeSet(ctx, datasafesdk.DeleteAttributeSetRequest{
		AttributeSetId: common.String(attributeSetID),
	})
	if err != nil {
		return handleAttributeSetDeleteRequestError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := strings.TrimSpace(attributeSetStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return false, fmt.Errorf("AttributeSet delete did not return an opc-work-request-id")
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(attributeSetID)
	currentAsync, err := servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, attributeSetWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawAction:        string(datasafesdk.WorkRequestOperationTypeDeleteAttributeSet),
		RawOperationType: string(datasafesdk.WorkRequestOperationTypeDeleteAttributeSet),
		WorkRequestID:    workRequestID,
		FallbackPhase:    shared.OSOKAsyncPhaseDelete,
	})
	if err != nil {
		return false, err
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())

	return c.resumeDeleteWorkRequest(ctx, resource, workRequestID)
}

func handleAttributeSetDeleteRequestError(resource *datasafev1beta1.AttributeSet, err error) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markAttributeSetDeleted(resource, "OCI resource no longer exists")
		return true, nil
	case classification.IsAuthShapedNotFound():
		return false, rejectAttributeSetAuthShapedNotFound(resource, err)
	default:
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
}

func (c attributeSetDeleteGuardClient) confirmPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
	currentID string,
) (bool, bool, error) {
	if err := requireAttributeSetOCIClient(c.client, c.initErr); err != nil {
		return true, false, err
	}
	response, err := c.client.GetAttributeSet(ctx, datasafesdk.GetAttributeSetRequest{AttributeSetId: common.String(currentID)})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		switch {
		case classification.IsUnambiguousNotFound():
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markAttributeSetDeleted(resource, "OCI resource no longer exists")
			return true, true, nil
		case classification.IsAuthShapedNotFound():
			return true, false, rejectAttributeSetAuthShapedNotFound(resource, err)
		default:
			return false, false, nil
		}
	}

	switch strings.ToUpper(string(response.LifecycleState)) {
	case string(datasafesdk.AttributeSetLifecycleStateCreating),
		string(datasafesdk.AttributeSetLifecycleStateUpdating):
		if err := projectAttributeSetStatus(resource, response.AttributeSet); err != nil {
			return true, false, err
		}
		markAttributeSetWritePending(resource, response.LifecycleState)
		return true, false, nil
	case string(datasafesdk.AttributeSetLifecycleStateDeleting):
		if err := projectAttributeSetStatus(resource, response.AttributeSet); err != nil {
			return true, false, err
		}
		markAttributeSetTerminating(resource, "OCI resource delete is in progress")
		return true, false, nil
	default:
		return false, false, nil
	}
}

func (c attributeSetDeleteGuardClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
	workRequestID string,
) (bool, error) {
	workRequest, err := getAttributeSetWorkRequest(ctx, c.client, c.initErr, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	currentAsync, err := buildAttributeSetDeleteAsyncOperation(resource, workRequest, workRequestID)
	if err != nil {
		return false, err
	}
	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.confirmDeleteAfterSucceededWorkRequest(ctx, resource)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())
		return false, fmt.Errorf("AttributeSet delete work request %s finished with status %s", workRequestID, currentAsync.RawStatus)
	default:
		return false, fmt.Errorf("AttributeSet delete work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass)
	}
}

func buildAttributeSetDeleteAsyncOperation(
	resource *datasafev1beta1.AttributeSet,
	workRequest any,
	workRequestID string,
) (*shared.OSOKAsyncOperation, error) {
	return buildAttributeSetWorkRequestAsyncOperation(resource, workRequest, workRequestID, shared.OSOKAsyncPhaseDelete)
}

func buildAttributeSetWorkRequestAsyncOperation(
	resource *datasafev1beta1.AttributeSet,
	workRequest any,
	workRequestID string,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, ok := attributeSetWorkRequestFromAny(workRequest)
	if !ok {
		return nil, fmt.Errorf("AttributeSet work request response did not expose a WorkRequest body")
	}
	if strings.TrimSpace(attributeSetStringValue(current.Id)) == "" {
		current.Id = common.String(workRequestID)
	}
	return servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, attributeSetWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        string(current.OperationType),
		RawOperationType: string(current.OperationType),
		WorkRequestID:    attributeSetStringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func (c attributeSetDeleteGuardClient) confirmDeleteAfterSucceededWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.AttributeSet,
) (bool, error) {
	currentID := currentAttributeSetID(resource)
	if currentID == "" {
		markAttributeSetDeleted(resource, "OCI AttributeSet delete work request completed")
		return true, nil
	}
	if err := requireAttributeSetOCIClient(c.client, c.initErr); err != nil {
		return false, err
	}

	response, err := c.client.GetAttributeSet(ctx, datasafesdk.GetAttributeSetRequest{AttributeSetId: common.String(currentID)})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		switch {
		case classification.IsUnambiguousNotFound():
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markAttributeSetDeleted(resource, "OCI resource deleted")
			return true, nil
		case classification.IsAuthShapedNotFound():
			return false, rejectAttributeSetAuthShapedNotFound(resource, err)
		default:
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, err
		}
	}
	if err := projectAttributeSetStatus(resource, response.AttributeSet); err != nil {
		return false, err
	}

	switch strings.ToUpper(string(response.LifecycleState)) {
	case string(datasafesdk.AttributeSetLifecycleStateDeleting), "":
		markAttributeSetTerminating(resource, "OCI resource delete is in progress")
		return false, nil
	default:
		return false, fmt.Errorf("AttributeSet delete work request succeeded but readback lifecycle state is %q", response.LifecycleState)
	}
}

func pendingAttributeSetDeleteWorkRequestID(resource *datasafev1beta1.AttributeSet) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func pendingAttributeSetWriteWorkRequest(resource *datasafev1beta1.AttributeSet) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
	default:
		return "", ""
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassSucceeded, shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled:
		return "", ""
	default:
		return strings.TrimSpace(current.WorkRequestID), current.Phase
	}
}

func recordAttributeSetIDFromWorkRequest(
	resource *datasafev1beta1.AttributeSet,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) {
	if resource == nil || currentAttributeSetID(resource) != "" {
		return
	}
	resourceID, err := recoverAttributeSetIDFromWorkRequest(resource, workRequest, phase)
	if err != nil || strings.TrimSpace(resourceID) == "" {
		return
	}
	resource.Status.Id = strings.TrimSpace(resourceID)
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
}

func rejectAttributeSetAuthShapedNotFound(resource *datasafev1beta1.AttributeSet, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return fmt.Errorf("AttributeSet delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted")
}

func attributeSetFromResponse(response any) (datasafesdk.AttributeSet, bool) {
	switch typed := response.(type) {
	case datasafesdk.CreateAttributeSetResponse:
		return typed.AttributeSet, true
	case datasafesdk.GetAttributeSetResponse:
		return typed.AttributeSet, true
	case datasafesdk.AttributeSet:
		return typed, true
	case datasafesdk.AttributeSetSummary:
		return attributeSetFromSummary(typed), true
	default:
		return datasafesdk.AttributeSet{}, false
	}
}

func attributeSetFromSummary(summary datasafesdk.AttributeSetSummary) datasafesdk.AttributeSet {
	return datasafesdk.AttributeSet{
		Id:               summary.Id,
		CompartmentId:    summary.CompartmentId,
		DisplayName:      summary.DisplayName,
		LifecycleState:   summary.LifecycleState,
		TimeCreated:      summary.TimeCreated,
		AttributeSetType: summary.AttributeSetType,
		Description:      summary.Description,
		TimeUpdated:      summary.TimeUpdated,
		IsUserDefined:    summary.IsUserDefined,
		InUse:            datasafesdk.AttributeSetInUseEnum(summary.InUse),
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		SystemTags:       summary.SystemTags,
	}
}

func projectAttributeSetStatus(resource *datasafev1beta1.AttributeSet, current datasafesdk.AttributeSet) error {
	if resource == nil {
		return fmt.Errorf("AttributeSet resource is nil")
	}
	resource.Status.Id = attributeSetStringValue(current.Id)
	resource.Status.CompartmentId = attributeSetStringValue(current.CompartmentId)
	resource.Status.DisplayName = attributeSetStringValue(current.DisplayName)
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.AttributeSetType = string(current.AttributeSetType)
	resource.Status.AttributeSetValues = cloneAttributeSetValues(current.AttributeSetValues)
	resource.Status.Description = attributeSetStringValue(current.Description)
	resource.Status.IsUserDefined = attributeSetBoolValue(current.IsUserDefined)
	resource.Status.InUse = string(current.InUse)
	resource.Status.FreeformTags = cloneAttributeSetStringMap(current.FreeformTags)
	resource.Status.DefinedTags = attributeSetSharedDefinedTags(current.DefinedTags)
	resource.Status.SystemTags = attributeSetSharedDefinedTags(current.SystemTags)
	if current.TimeCreated != nil {
		resource.Status.TimeCreated = current.TimeCreated.String()
	}
	if current.TimeUpdated != nil {
		resource.Status.TimeUpdated = current.TimeUpdated.String()
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func markAttributeSetDeleted(resource *datasafev1beta1.AttributeSet, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, nilAttributeSetLogger())
}

func markAttributeSetTerminating(resource *datasafev1beta1.AttributeSet, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}, nilAttributeSetLogger())
}

func markAttributeSetWritePending(
	resource *datasafev1beta1.AttributeSet,
	state datasafesdk.AttributeSetLifecycleStateEnum,
) {
	if resource == nil {
		return
	}
	message := fmt.Sprintf("OCI AttributeSet is %s; waiting before delete", state)
	currentAsync := servicemanager.NewLifecycleAsyncOperation(&resource.Status.OsokStatus, string(state), message, "")
	if currentAsync == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilAttributeSetLogger())
}

func currentAttributeSetID(resource *datasafev1beta1.AttributeSet) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func clearTrackedAttributeSetIdentity(resource *datasafev1beta1.AttributeSet) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
}

func attributeSetCompartmentMoveRetryToken(resource *datasafev1beta1.AttributeSet, compartmentID string) *string {
	if resource == nil {
		return nil
	}
	parts := []string{
		string(resource.UID),
		resource.Namespace,
		resource.Name,
		strings.TrimSpace(compartmentID),
	}
	return common.String(strings.Join(parts, ":"))
}

func attributeSetDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func attributeSetSharedDefinedTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		inner := make(shared.MapValue, len(values))
		for key, value := range values {
			inner[key] = fmt.Sprint(value)
		}
		converted[namespace] = inner
	}
	return converted
}

func cloneAttributeSetStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func cloneAttributeSetValues(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func attributeSetStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func attributeSetBoolValue(value *bool) bool {
	return value != nil && *value
}

func nilAttributeSetLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
