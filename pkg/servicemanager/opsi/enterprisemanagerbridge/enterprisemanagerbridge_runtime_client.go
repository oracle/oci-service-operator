/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package enterprisemanagerbridge

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var enterpriseManagerBridgeWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens:   []string{string(opsisdk.OperationStatusAccepted), string(opsisdk.OperationStatusInProgress), string(opsisdk.OperationStatusWaiting), string(opsisdk.OperationStatusCanceling)},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opsisdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(opsisdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(opsisdk.ActionTypeDeleted)},
}

type enterpriseManagerBridgeOCIClient interface {
	CreateEnterpriseManagerBridge(context.Context, opsisdk.CreateEnterpriseManagerBridgeRequest) (opsisdk.CreateEnterpriseManagerBridgeResponse, error)
	GetEnterpriseManagerBridge(context.Context, opsisdk.GetEnterpriseManagerBridgeRequest) (opsisdk.GetEnterpriseManagerBridgeResponse, error)
	ListEnterpriseManagerBridges(context.Context, opsisdk.ListEnterpriseManagerBridgesRequest) (opsisdk.ListEnterpriseManagerBridgesResponse, error)
	UpdateEnterpriseManagerBridge(context.Context, opsisdk.UpdateEnterpriseManagerBridgeRequest) (opsisdk.UpdateEnterpriseManagerBridgeResponse, error)
	DeleteEnterpriseManagerBridge(context.Context, opsisdk.DeleteEnterpriseManagerBridgeRequest) (opsisdk.DeleteEnterpriseManagerBridgeResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type enterpriseManagerBridgeDeleteGuardClient struct {
	delegate       EnterpriseManagerBridgeServiceClient
	get            func(context.Context, opsisdk.GetEnterpriseManagerBridgeRequest) (opsisdk.GetEnterpriseManagerBridgeResponse, error)
	getWorkRequest func(context.Context, string) (any, error)
}

func init() {
	registerEnterpriseManagerBridgeRuntimeHooksMutator(func(manager *EnterpriseManagerBridgeServiceManager, hooks *EnterpriseManagerBridgeRuntimeHooks) {
		client, initErr := newEnterpriseManagerBridgeSDKClient(manager)
		applyEnterpriseManagerBridgeRuntimeHooks(hooks, client, initErr)
	})
}

func newEnterpriseManagerBridgeSDKClient(manager *EnterpriseManagerBridgeServiceManager) (enterpriseManagerBridgeOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("EnterpriseManagerBridge service manager is nil")
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize EnterpriseManagerBridge OCI client: %w", err)
	}
	return client, nil
}

func applyEnterpriseManagerBridgeRuntimeHooks(
	hooks *EnterpriseManagerBridgeRuntimeHooks,
	client enterpriseManagerBridgeOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = enterpriseManagerBridgeRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *opsiv1beta1.EnterpriseManagerBridge, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("EnterpriseManagerBridge resource is nil")
		}
		return buildEnterpriseManagerBridgeCreateDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *opsiv1beta1.EnterpriseManagerBridge,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildEnterpriseManagerBridgeUpdateDetails(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardEnterpriseManagerBridgeExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedEnterpriseManagerBridgeIdentity
	hooks.StatusHooks.ProjectStatus = enterpriseManagerBridgeStatusFromResponse
	hooks.DeleteHooks.HandleError = rejectEnterpriseManagerBridgeAuthShapedNotFound
	hooks.Async.Adapter = enterpriseManagerBridgeWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getEnterpriseManagerBridgeWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveEnterpriseManagerBridgeGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveEnterpriseManagerBridgeGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverEnterpriseManagerBridgeIDFromGeneratedWorkRequest
	hooks.Async.Message = enterpriseManagerBridgeGeneratedWorkRequestMessage

	hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateEnterpriseManagerBridgeRequest) (opsisdk.CreateEnterpriseManagerBridgeResponse, error) {
		if err := enterpriseManagerBridgeClientReady(client, initErr); err != nil {
			return opsisdk.CreateEnterpriseManagerBridgeResponse{}, err
		}
		return client.CreateEnterpriseManagerBridge(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request opsisdk.GetEnterpriseManagerBridgeRequest) (opsisdk.GetEnterpriseManagerBridgeResponse, error) {
		if err := enterpriseManagerBridgeClientReady(client, initErr); err != nil {
			return opsisdk.GetEnterpriseManagerBridgeResponse{}, err
		}
		return client.GetEnterpriseManagerBridge(ctx, request)
	}
	hooks.List.Fields = enterpriseManagerBridgeListFields()
	hooks.List.Call = func(ctx context.Context, request opsisdk.ListEnterpriseManagerBridgesRequest) (opsisdk.ListEnterpriseManagerBridgesResponse, error) {
		return listEnterpriseManagerBridgesAllPages(ctx, client, initErr, request)
	}
	hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateEnterpriseManagerBridgeRequest) (opsisdk.UpdateEnterpriseManagerBridgeResponse, error) {
		if err := enterpriseManagerBridgeClientReady(client, initErr); err != nil {
			return opsisdk.UpdateEnterpriseManagerBridgeResponse{}, err
		}
		return client.UpdateEnterpriseManagerBridge(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request opsisdk.DeleteEnterpriseManagerBridgeRequest) (opsisdk.DeleteEnterpriseManagerBridgeResponse, error) {
		if err := enterpriseManagerBridgeClientReady(client, initErr); err != nil {
			return opsisdk.DeleteEnterpriseManagerBridgeResponse{}, err
		}
		return client.DeleteEnterpriseManagerBridge(ctx, request)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapEnterpriseManagerBridgeDeleteGuard(hooks.Get.Call, hooks.Async.GetWorkRequest))
}

func enterpriseManagerBridgeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "enterprisemanagerbridge",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(opsisdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "objectStorageBucketName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "description", "displayName", "freeformTags"},
			Mutable:         []string{"definedTags", "description", "displayName", "freeformTags"},
			ForceNew:        []string{"compartmentId", "objectStorageBucketName"},
			ConflictsWith:   map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func enterpriseManagerBridgeListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func enterpriseManagerBridgeClientReady(client enterpriseManagerBridgeOCIClient, initErr error) error {
	if initErr != nil {
		return initErr
	}
	if client == nil {
		return fmt.Errorf("EnterpriseManagerBridge OCI client is not configured")
	}
	return nil
}

func buildEnterpriseManagerBridgeCreateDetails(spec opsiv1beta1.EnterpriseManagerBridgeSpec) opsisdk.CreateEnterpriseManagerBridgeDetails {
	body := opsisdk.CreateEnterpriseManagerBridgeDetails{
		CompartmentId:           common.String(spec.CompartmentId),
		DisplayName:             common.String(spec.DisplayName),
		ObjectStorageBucketName: common.String(spec.ObjectStorageBucketName),
	}
	if spec.Description != "" {
		body.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return body
}

func buildEnterpriseManagerBridgeUpdateDetails(
	resource *opsiv1beta1.EnterpriseManagerBridge,
	currentResponse any,
) (opsisdk.UpdateEnterpriseManagerBridgeDetails, bool, error) {
	if resource == nil {
		return opsisdk.UpdateEnterpriseManagerBridgeDetails{}, false, fmt.Errorf("EnterpriseManagerBridge resource is nil")
	}
	current, ok := enterpriseManagerBridgeFromResponse(currentResponse)
	if !ok {
		return opsisdk.UpdateEnterpriseManagerBridgeDetails{}, false, fmt.Errorf("current EnterpriseManagerBridge response does not expose an EnterpriseManagerBridge body")
	}
	if err := validateEnterpriseManagerBridgeCreateOnlyDrift(resource, current); err != nil {
		return opsisdk.UpdateEnterpriseManagerBridgeDetails{}, false, err
	}

	var details opsisdk.UpdateEnterpriseManagerBridgeDetails
	updateNeeded := false
	if resource.Spec.DisplayName != stringValue(current.DisplayName) {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.Description != "" && resource.Spec.Description != stringValue(current.Description) {
		details.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}
	if tags, ok := desiredEnterpriseManagerBridgeFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = tags
		updateNeeded = true
	}
	if tags, ok := desiredEnterpriseManagerBridgeDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = tags
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func guardEnterpriseManagerBridgeExistingBeforeCreate(
	_ context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("EnterpriseManagerBridge resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" ||
		strings.TrimSpace(resource.Spec.ObjectStorageBucketName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func clearTrackedEnterpriseManagerBridgeIdentity(resource *opsiv1beta1.EnterpriseManagerBridge) {
	if resource == nil {
		return
	}
	resource.Status = opsiv1beta1.EnterpriseManagerBridgeStatus{}
}

func validateEnterpriseManagerBridgeCreateOnlyDrift(
	resource *opsiv1beta1.EnterpriseManagerBridge,
	current opsisdk.EnterpriseManagerBridge,
) error {
	if resource == nil {
		return fmt.Errorf("EnterpriseManagerBridge resource is nil")
	}
	var drift []string
	if hasStringCreateOnlyDrift(resource.Spec.CompartmentId, stringValue(current.CompartmentId)) {
		drift = append(drift, "compartmentId")
	}
	if hasStringCreateOnlyDrift(resource.Spec.ObjectStorageBucketName, stringValue(current.ObjectStorageBucketName)) {
		drift = append(drift, "objectStorageBucketName")
	}
	if len(drift) != 0 {
		return fmt.Errorf("EnterpriseManagerBridge create-only drift detected for %s; replace the resource or restore the desired spec before update", strings.Join(drift, ", "))
	}
	return nil
}

func hasStringCreateOnlyDrift(desired, current string) bool {
	return strings.TrimSpace(desired) != "" && strings.TrimSpace(current) != "" && desired != current
}

func desiredEnterpriseManagerBridgeFreeformTagsForUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	desired := maps.Clone(spec)
	if maps.Equal(desired, current) {
		return nil, false
	}
	return desired, true
}

func desiredEnterpriseManagerBridgeDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := *util.ConvertToOciDefinedTags(&spec)
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func enterpriseManagerBridgeStatusFromResponse(resource *opsiv1beta1.EnterpriseManagerBridge, response any) error {
	current, ok := enterpriseManagerBridgeFromResponse(response)
	if !ok {
		return nil
	}
	projectEnterpriseManagerBridgeStatus(resource, current)
	return nil
}

func projectEnterpriseManagerBridgeStatus(resource *opsiv1beta1.EnterpriseManagerBridge, current opsisdk.EnterpriseManagerBridge) {
	if resource == nil {
		return
	}
	resource.Status.Id = stringValue(current.Id)
	resource.Status.CompartmentId = stringValue(current.CompartmentId)
	resource.Status.DisplayName = stringValue(current.DisplayName)
	resource.Status.ObjectStorageNamespaceName = stringValue(current.ObjectStorageNamespaceName)
	resource.Status.ObjectStorageBucketName = stringValue(current.ObjectStorageBucketName)
	resource.Status.FreeformTags = cloneStringMap(current.FreeformTags)
	resource.Status.DefinedTags = statusDefinedTagsFromSDK(current.DefinedTags)
	resource.Status.TimeCreated = sdkTimeString(current.TimeCreated)
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.Description = stringValue(current.Description)
	resource.Status.ObjectStorageBucketStatusDetails = stringValue(current.ObjectStorageBucketStatusDetails)
	resource.Status.SystemTags = statusDefinedTagsFromSDK(current.SystemTags)
	resource.Status.TimeUpdated = sdkTimeString(current.TimeUpdated)
	resource.Status.LifecycleDetails = stringValue(current.LifecycleDetails)
}

func rejectEnterpriseManagerBridgeAuthShapedNotFound(resource *opsiv1beta1.EnterpriseManagerBridge, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return fmt.Errorf("EnterpriseManagerBridge delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted")
}

func getEnterpriseManagerBridgeWorkRequest(
	ctx context.Context,
	client enterpriseManagerBridgeOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if err := enterpriseManagerBridgeClientReady(client, initErr); err != nil {
		return nil, err
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveEnterpriseManagerBridgeGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := enterpriseManagerBridgeWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveEnterpriseManagerBridgeWorkRequestAction(current)
}

func resolveEnterpriseManagerBridgeGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := enterpriseManagerBridgeWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := enterpriseManagerBridgeWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverEnterpriseManagerBridgeIDFromGeneratedWorkRequest(
	_ *opsiv1beta1.EnterpriseManagerBridge,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := enterpriseManagerBridgeWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveEnterpriseManagerBridgeIDFromWorkRequest(current, enterpriseManagerBridgeWorkRequestActionForPhase(phase))
}

func resolveEnterpriseManagerBridgeWorkRequestAction(workRequest opsisdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isEnterpriseManagerBridgeWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || strings.EqualFold(candidate, string(opsisdk.ActionTypeInProgress)) || strings.EqualFold(candidate, string(opsisdk.ActionTypeRelated)) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("EnterpriseManagerBridge work request %s exposes conflicting EnterpriseManagerBridge action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func enterpriseManagerBridgeWorkRequestPhaseFromOperationType(operationType opsisdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch strings.ToUpper(strings.TrimSpace(string(operationType))) {
	case string(opsisdk.OperationTypeCreateEnterpriseManagerBridge):
		return shared.OSOKAsyncPhaseCreate, true
	case string(opsisdk.OperationTypeUdpateEnterpriseManagerBridge), "UPDATE_ENTERPRISE_MANAGER_BRIDGE":
		return shared.OSOKAsyncPhaseUpdate, true
	case string(opsisdk.OperationTypeDeleteEnterpriseManagerBridge):
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func enterpriseManagerBridgeWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return opsisdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return opsisdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return opsisdk.ActionTypeDeleted
	default:
		return ""
	}
}

func enterpriseManagerBridgeGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := enterpriseManagerBridgeWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("EnterpriseManagerBridge %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func enterpriseManagerBridgeWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("EnterpriseManagerBridge work request is nil")
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("unexpected EnterpriseManagerBridge work request type %T", workRequest)
	}
}

func resolveEnterpriseManagerBridgeIDFromWorkRequest(workRequest opsisdk.WorkRequest, action opsisdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveEnterpriseManagerBridgeIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveEnterpriseManagerBridgeIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("EnterpriseManagerBridge work request %s does not expose an EnterpriseManagerBridge identifier", stringValue(workRequest.Id))
}

func resolveEnterpriseManagerBridgeIDFromResources(
	resources []opsisdk.WorkRequestResource,
	action opsisdk.ActionTypeEnum,
	preferEnterpriseManagerBridgeOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferEnterpriseManagerBridgeOnly && !isEnterpriseManagerBridgeWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func isEnterpriseManagerBridgeWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := normalizeEnterpriseManagerBridgeWorkRequestText(stringValue(resource.EntityType))
	if entityType != "" {
		return strings.Contains(entityType, "enterprisemanagerbridge")
	}
	entityURI := normalizeEnterpriseManagerBridgeWorkRequestText(stringValue(resource.EntityUri))
	if strings.Contains(entityURI, "enterprisemanagerbridges") {
		return true
	}
	return strings.Contains(normalizeEnterpriseManagerBridgeWorkRequestText(stringValue(resource.Identifier)), "enterprisemanagerbridge")
}

func normalizeEnterpriseManagerBridgeWorkRequestText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(value)
}

func listEnterpriseManagerBridgesAllPages(
	ctx context.Context,
	client enterpriseManagerBridgeOCIClient,
	initErr error,
	request opsisdk.ListEnterpriseManagerBridgesRequest,
) (opsisdk.ListEnterpriseManagerBridgesResponse, error) {
	if err := enterpriseManagerBridgeClientReady(client, initErr); err != nil {
		return opsisdk.ListEnterpriseManagerBridgesResponse{}, err
	}

	var combined opsisdk.ListEnterpriseManagerBridgesResponse
	for {
		response, err := client.ListEnterpriseManagerBridges(ctx, request)
		if err != nil {
			return opsisdk.ListEnterpriseManagerBridgesResponse{}, err
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == opsisdk.LifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		page := nextPage(response.OpcNextPage)
		if page == nil {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = page
		combined.OpcNextPage = page
	}
}

func wrapEnterpriseManagerBridgeDeleteGuard(
	get func(context.Context, opsisdk.GetEnterpriseManagerBridgeRequest) (opsisdk.GetEnterpriseManagerBridgeResponse, error),
	getWorkRequest func(context.Context, string) (any, error),
) func(EnterpriseManagerBridgeServiceClient) EnterpriseManagerBridgeServiceClient {
	return func(delegate EnterpriseManagerBridgeServiceClient) EnterpriseManagerBridgeServiceClient {
		return enterpriseManagerBridgeDeleteGuardClient{
			delegate:       delegate,
			get:            get,
			getWorkRequest: getWorkRequest,
		}
	}
}

func (c enterpriseManagerBridgeDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("EnterpriseManagerBridge runtime delegate is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c enterpriseManagerBridgeDeleteGuardClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("EnterpriseManagerBridge runtime delegate is not configured")
	}
	if deleted, handled, err := c.resumeWriteWorkRequestBeforeDelete(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedSucceededDeleteWorkRequestConfirmation(ctx, resource); err != nil {
		return false, err
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c enterpriseManagerBridgeDeleteGuardClient) resumeWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
) (bool, bool, error) {
	workRequestID, phase := currentEnterpriseManagerBridgeWriteWorkRequest(resource)
	if workRequestID == "" {
		return false, false, nil
	}
	if c.getWorkRequest == nil {
		return false, true, fmt.Errorf("EnterpriseManagerBridge work request polling is not configured")
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return false, true, err
	}
	current, err := buildEnterpriseManagerBridgeAsyncOperation(resource, workRequest, phase)
	if err != nil {
		return false, true, c.failWriteWorkRequestForDelete(resource, nil, err)
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("EnterpriseManagerBridge %s work request %s is still in progress; waiting before delete", current.Phase, workRequestID)
		markEnterpriseManagerBridgeWorkRequestOperation(resource, current, shared.OSOKAsyncClassPending, message)
		return false, true, nil
	case shared.OSOKAsyncClassSucceeded:
		deleted, err := c.deleteAfterSucceededWriteWorkRequest(ctx, resource, workRequest, current, workRequestID)
		return deleted, true, err
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("EnterpriseManagerBridge %s work request %s finished with status %s before delete", current.Phase, workRequestID, current.RawStatus)
		return false, true, c.failWriteWorkRequestForDelete(resource, current, err)
	default:
		err := fmt.Errorf("EnterpriseManagerBridge %s work request %s projected unsupported async class %s before delete", current.Phase, workRequestID, current.NormalizedClass)
		return false, true, c.failWriteWorkRequestForDelete(resource, current, err)
	}
}

func (c enterpriseManagerBridgeDeleteGuardClient) deleteAfterSucceededWriteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID := currentEnterpriseManagerBridgeID(resource)
	if resourceID == "" {
		recoveredID, err := recoverEnterpriseManagerBridgeIDFromGeneratedWorkRequest(resource, workRequest, current.Phase)
		if err != nil {
			return false, c.failWriteWorkRequestForDelete(resource, current, err)
		}
		resourceID = strings.TrimSpace(recoveredID)
	}
	if resourceID == "" {
		return false, c.failWriteWorkRequestForDelete(resource, current, fmt.Errorf("EnterpriseManagerBridge %s work request %s did not expose an EnterpriseManagerBridge identifier", current.Phase, workRequestID))
	}
	if c.get == nil {
		return false, c.failWriteWorkRequestForDelete(resource, current, fmt.Errorf("EnterpriseManagerBridge readback is not configured"))
	}

	response, err := c.get(ctx, opsisdk.GetEnterpriseManagerBridgeRequest{EnterpriseManagerBridgeId: common.String(resourceID)})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		if classification.IsAuthShapedNotFound() {
			if resource != nil {
				servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			}
			return false, fmt.Errorf("EnterpriseManagerBridge write work request readback returned ambiguous 404 NotAuthorizedOrNotFound before delete: %v", err)
		}
		if classification.IsUnambiguousNotFound() {
			message := fmt.Sprintf("EnterpriseManagerBridge %s work request %s succeeded; waiting for EnterpriseManagerBridge %s to become readable before delete", current.Phase, workRequestID, resourceID)
			markEnterpriseManagerBridgeWorkRequestOperation(resource, current, shared.OSOKAsyncClassPending, message)
			return false, nil
		}
		return false, c.failWriteWorkRequestForDelete(resource, current, err)
	}

	projectEnterpriseManagerBridgeStatus(resource, response.EnterpriseManagerBridge)
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.delegate.Delete(ctx, resource)
}

func (c enterpriseManagerBridgeDeleteGuardClient) rejectAuthShapedSucceededDeleteWorkRequestConfirmation(
	ctx context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
) error {
	workRequestID := currentEnterpriseManagerBridgeDeleteWorkRequestID(resource)
	if workRequestID == "" {
		return nil
	}
	if c.getWorkRequest == nil {
		return fmt.Errorf("EnterpriseManagerBridge work request polling is not configured")
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return err
	}
	current, err := buildEnterpriseManagerBridgeAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return err
	}
	if current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
		return nil
	}
	return c.rejectAuthShapedConfirmRead(ctx, resource)
}

func (c enterpriseManagerBridgeDeleteGuardClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *opsiv1beta1.EnterpriseManagerBridge,
) error {
	if c.get == nil || resource == nil {
		return nil
	}
	resourceID := currentEnterpriseManagerBridgeID(resource)
	if resourceID == "" {
		return nil
	}
	_, err := c.get(ctx, opsisdk.GetEnterpriseManagerBridgeRequest{EnterpriseManagerBridgeId: common.String(resourceID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("EnterpriseManagerBridge delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func buildEnterpriseManagerBridgeAsyncOperation(
	resource *opsiv1beta1.EnterpriseManagerBridge,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := enterpriseManagerBridgeWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	derivedPhase, ok := enterpriseManagerBridgeWorkRequestPhaseFromOperationType(current.OperationType)
	if ok {
		if phase != "" && phase != derivedPhase {
			return nil, fmt.Errorf("EnterpriseManagerBridge work request %s exposes phase %q while delete expected %q", stringValue(current.Id), derivedPhase, phase)
		}
		phase = derivedPhase
	}
	action, err := resolveEnterpriseManagerBridgeWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, enterpriseManagerBridgeWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        action,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    stringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    phase,
	})
}

func currentEnterpriseManagerBridgeWriteWorkRequest(resource *opsiv1beta1.EnterpriseManagerBridge) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return strings.TrimSpace(current.WorkRequestID), current.Phase
	default:
		return "", ""
	}
}

func currentEnterpriseManagerBridgeDeleteWorkRequestID(resource *opsiv1beta1.EnterpriseManagerBridge) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func markEnterpriseManagerBridgeWorkRequestOperation(
	resource *opsiv1beta1.EnterpriseManagerBridge,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) {
	if resource == nil || current == nil {
		return
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	now := metav1.Now()
	next.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, loggerutil.OSOKLogger{})
}

func (c enterpriseManagerBridgeDeleteGuardClient) failWriteWorkRequestForDelete(
	resource *opsiv1beta1.EnterpriseManagerBridge,
	current *shared.OSOKAsyncOperation,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if current == nil {
		return err
	}
	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}
	markEnterpriseManagerBridgeWorkRequestOperation(resource, current, class, err.Error())
	return err
}

func enterpriseManagerBridgeFromResponse(response any) (opsisdk.EnterpriseManagerBridge, bool) {
	if current, ok := enterpriseManagerBridgeFromOperationResponse(response); ok {
		return current, true
	}
	return enterpriseManagerBridgeFromResourcePayload(response)
}

func enterpriseManagerBridgeFromOperationResponse(response any) (opsisdk.EnterpriseManagerBridge, bool) {
	switch current := response.(type) {
	case opsisdk.CreateEnterpriseManagerBridgeResponse:
		return current.EnterpriseManagerBridge, true
	case *opsisdk.CreateEnterpriseManagerBridgeResponse:
		if current == nil {
			return opsisdk.EnterpriseManagerBridge{}, false
		}
		return current.EnterpriseManagerBridge, true
	case opsisdk.GetEnterpriseManagerBridgeResponse:
		return current.EnterpriseManagerBridge, true
	case *opsisdk.GetEnterpriseManagerBridgeResponse:
		if current == nil {
			return opsisdk.EnterpriseManagerBridge{}, false
		}
		return current.EnterpriseManagerBridge, true
	default:
		return opsisdk.EnterpriseManagerBridge{}, false
	}
}

func enterpriseManagerBridgeFromResourcePayload(response any) (opsisdk.EnterpriseManagerBridge, bool) {
	switch current := response.(type) {
	case opsisdk.EnterpriseManagerBridge:
		return current, true
	case *opsisdk.EnterpriseManagerBridge:
		if current == nil {
			return opsisdk.EnterpriseManagerBridge{}, false
		}
		return *current, true
	case opsisdk.EnterpriseManagerBridgeSummary:
		return enterpriseManagerBridgeFromSummary(current), true
	case *opsisdk.EnterpriseManagerBridgeSummary:
		if current == nil {
			return opsisdk.EnterpriseManagerBridge{}, false
		}
		return enterpriseManagerBridgeFromSummary(*current), true
	default:
		return opsisdk.EnterpriseManagerBridge{}, false
	}
}

func enterpriseManagerBridgeFromSummary(summary opsisdk.EnterpriseManagerBridgeSummary) opsisdk.EnterpriseManagerBridge {
	return opsisdk.EnterpriseManagerBridge{
		Id:                               summary.Id,
		CompartmentId:                    summary.CompartmentId,
		DisplayName:                      summary.DisplayName,
		ObjectStorageNamespaceName:       summary.ObjectStorageNamespaceName,
		ObjectStorageBucketName:          summary.ObjectStorageBucketName,
		FreeformTags:                     summary.FreeformTags,
		DefinedTags:                      summary.DefinedTags,
		TimeCreated:                      summary.TimeCreated,
		LifecycleState:                   summary.LifecycleState,
		ObjectStorageBucketStatusDetails: summary.ObjectStorageBucketStatusDetails,
		SystemTags:                       summary.SystemTags,
		TimeUpdated:                      summary.TimeUpdated,
		LifecycleDetails:                 summary.LifecycleDetails,
	}
}

func currentEnterpriseManagerBridgeID(resource *opsiv1beta1.EnterpriseManagerBridge) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func statusDefinedTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = fmt.Sprint(value)
		}
		converted[namespace] = children
	}
	return converted
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	return maps.Clone(input)
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nextPage(page *string) *string {
	if strings.TrimSpace(stringValue(page)) == "" {
		return nil
	}
	return page
}

func newEnterpriseManagerBridgeServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client enterpriseManagerBridgeOCIClient,
) EnterpriseManagerBridgeServiceClient {
	hooks := newEnterpriseManagerBridgeRuntimeHooksWithOCIClient(client)
	applyEnterpriseManagerBridgeRuntimeHooks(&hooks, client, nil)
	manager := &EnterpriseManagerBridgeServiceManager{Log: log}
	return wrapEnterpriseManagerBridgeGeneratedClient(
		hooks,
		defaultEnterpriseManagerBridgeServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.EnterpriseManagerBridge](
				buildEnterpriseManagerBridgeGeneratedRuntimeConfig(manager, hooks),
			),
		},
	)
}

func newEnterpriseManagerBridgeRuntimeHooksWithOCIClient(client enterpriseManagerBridgeOCIClient) EnterpriseManagerBridgeRuntimeHooks {
	return EnterpriseManagerBridgeRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.EnterpriseManagerBridge]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.EnterpriseManagerBridge]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.EnterpriseManagerBridge]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.EnterpriseManagerBridge]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.EnterpriseManagerBridge]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.EnterpriseManagerBridge]{},
		Create: runtimeOperationHooks[opsisdk.CreateEnterpriseManagerBridgeRequest, opsisdk.CreateEnterpriseManagerBridgeResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateEnterpriseManagerBridgeDetails", RequestName: "CreateEnterpriseManagerBridgeDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.CreateEnterpriseManagerBridgeRequest) (opsisdk.CreateEnterpriseManagerBridgeResponse, error) {
				return client.CreateEnterpriseManagerBridge(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetEnterpriseManagerBridgeRequest, opsisdk.GetEnterpriseManagerBridgeResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "EnterpriseManagerBridgeId", RequestName: "enterpriseManagerBridgeId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetEnterpriseManagerBridgeRequest) (opsisdk.GetEnterpriseManagerBridgeResponse, error) {
				return client.GetEnterpriseManagerBridge(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListEnterpriseManagerBridgesRequest, opsisdk.ListEnterpriseManagerBridgesResponse]{
			Fields: enterpriseManagerBridgeListFields(),
			Call: func(ctx context.Context, request opsisdk.ListEnterpriseManagerBridgesRequest) (opsisdk.ListEnterpriseManagerBridgesResponse, error) {
				return client.ListEnterpriseManagerBridges(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateEnterpriseManagerBridgeRequest, opsisdk.UpdateEnterpriseManagerBridgeResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "EnterpriseManagerBridgeId", RequestName: "enterpriseManagerBridgeId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateEnterpriseManagerBridgeDetails", RequestName: "UpdateEnterpriseManagerBridgeDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.UpdateEnterpriseManagerBridgeRequest) (opsisdk.UpdateEnterpriseManagerBridgeResponse, error) {
				return client.UpdateEnterpriseManagerBridge(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteEnterpriseManagerBridgeRequest, opsisdk.DeleteEnterpriseManagerBridgeResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "EnterpriseManagerBridgeId", RequestName: "enterpriseManagerBridgeId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteEnterpriseManagerBridgeRequest) (opsisdk.DeleteEnterpriseManagerBridgeResponse, error) {
				return client.DeleteEnterpriseManagerBridge(ctx, request)
			},
		},
		WrapGeneratedClient: []func(EnterpriseManagerBridgeServiceClient) EnterpriseManagerBridgeServiceClient{},
	}
}
