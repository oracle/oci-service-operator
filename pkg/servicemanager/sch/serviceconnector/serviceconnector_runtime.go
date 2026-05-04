/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceconnector

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	schsdk "github.com/oracle/oci-go-sdk/v65/sch"
	schv1beta1 "github.com/oracle/oci-service-operator/api/sch/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

var serviceConnectorWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(schsdk.OperationStatusAccepted),
		string(schsdk.OperationStatusInProgress),
		string(schsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(schsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(schsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(schsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(schsdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(schsdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(schsdk.ActionTypeDeleted)},
}

type serviceConnectorOCIClient interface {
	CreateServiceConnector(context.Context, schsdk.CreateServiceConnectorRequest) (schsdk.CreateServiceConnectorResponse, error)
	GetServiceConnector(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error)
	ListServiceConnectors(context.Context, schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error)
	UpdateServiceConnector(context.Context, schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error)
	DeleteServiceConnector(context.Context, schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error)
	GetWorkRequest(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error)
}

type serviceConnectorWorkRequestClient interface {
	GetWorkRequest(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error)
}

func init() {
	registerServiceConnectorRuntimeHooksMutator(func(manager *ServiceConnectorServiceManager, hooks *ServiceConnectorRuntimeHooks) {
		workRequestClient, initErr := newServiceConnectorWorkRequestClient(manager)
		applyServiceConnectorRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newServiceConnectorWorkRequestClient(manager *ServiceConnectorServiceManager) (serviceConnectorWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ServiceConnector service manager is nil")
	}
	client, err := schsdk.NewServiceConnectorClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyServiceConnectorRuntimeHooks(
	manager *ServiceConnectorServiceManager,
	hooks *ServiceConnectorRuntimeHooks,
	workRequestClient serviceConnectorWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = serviceConnectorRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *schv1beta1.ServiceConnector, namespace string) (any, error) {
		return buildServiceConnectorCreateDetails(ctx, manager, resource, namespace)
	}
	hooks.BuildUpdateBody = func(ctx context.Context, resource *schv1beta1.ServiceConnector, namespace string, currentResponse any) (any, bool, error) {
		return buildServiceConnectorUpdateDetails(ctx, manager, resource, namespace, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardServiceConnectorExistingBeforeCreate
	hooks.DeleteHooks.HandleError = rejectServiceConnectorAuthShapedNotFound
	hooks.Async.Adapter = serviceConnectorWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getServiceConnectorWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveServiceConnectorGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveServiceConnectorGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverServiceConnectorIDFromGeneratedWorkRequest
	hooks.Async.Message = serviceConnectorGeneratedWorkRequestMessage

	wrapServiceConnectorListPagination(hooks)
	wrapServiceConnectorDeleteConfirmation(manager, hooks)
}

func serviceConnectorRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "sch",
		FormalSlug:        "serviceconnector",
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
			ProvisioningStates: []string{string(schsdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(schsdk.LifecycleStateUpdating)},
			ActiveStates: []string{
				string(schsdk.LifecycleStateActive),
				string(schsdk.LifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(schsdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(schsdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "description", "displayName", "freeformTags", "source", "target", "tasks"},
			Mutable:         []string{"definedTags", "description", "displayName", "freeformTags", "source", "target", "tasks"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func newServiceConnectorServiceClientWithOCIClient(log loggerutil.OSOKLogger, client serviceConnectorOCIClient) ServiceConnectorServiceClient {
	hooks := newServiceConnectorRuntimeHooksWithOCIClient(client)
	applyServiceConnectorRuntimeHooks(&ServiceConnectorServiceManager{Log: log}, &hooks, client, nil)
	delegate := defaultServiceConnectorServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*schv1beta1.ServiceConnector](
			buildServiceConnectorGeneratedRuntimeConfig(&ServiceConnectorServiceManager{Log: log}, hooks),
		),
	}
	return wrapServiceConnectorGeneratedClient(hooks, delegate)
}

func newServiceConnectorRuntimeHooksWithOCIClient(client serviceConnectorOCIClient) ServiceConnectorRuntimeHooks {
	if client == nil {
		return ServiceConnectorRuntimeHooks{}
	}
	return ServiceConnectorRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*schv1beta1.ServiceConnector]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*schv1beta1.ServiceConnector]{},
		StatusHooks:     generatedruntime.StatusHooks[*schv1beta1.ServiceConnector]{},
		ParityHooks:     generatedruntime.ParityHooks[*schv1beta1.ServiceConnector]{},
		Async:           generatedruntime.AsyncHooks[*schv1beta1.ServiceConnector]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*schv1beta1.ServiceConnector]{},
		Create: runtimeOperationHooks[schsdk.CreateServiceConnectorRequest, schsdk.CreateServiceConnectorResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateServiceConnectorDetails", RequestName: "CreateServiceConnectorDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request schsdk.CreateServiceConnectorRequest) (schsdk.CreateServiceConnectorResponse, error) {
				return client.CreateServiceConnector(ctx, request)
			},
		},
		Get: runtimeOperationHooks[schsdk.GetServiceConnectorRequest, schsdk.GetServiceConnectorResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ServiceConnectorId", RequestName: "serviceConnectorId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
				return client.GetServiceConnector(ctx, request)
			},
		},
		List: runtimeOperationHooks[schsdk.ListServiceConnectorsRequest, schsdk.ListServiceConnectorsResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
			},
			Call: func(ctx context.Context, request schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
				return client.ListServiceConnectors(ctx, request)
			},
		},
		Update: runtimeOperationHooks[schsdk.UpdateServiceConnectorRequest, schsdk.UpdateServiceConnectorResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ServiceConnectorId", RequestName: "serviceConnectorId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateServiceConnectorDetails", RequestName: "UpdateServiceConnectorDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error) {
				return client.UpdateServiceConnector(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[schsdk.DeleteServiceConnectorRequest, schsdk.DeleteServiceConnectorResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ServiceConnectorId", RequestName: "serviceConnectorId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
				return client.DeleteServiceConnector(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ServiceConnectorServiceClient) ServiceConnectorServiceClient{},
	}
}

func buildServiceConnectorCreateDetails(
	ctx context.Context,
	manager *ServiceConnectorServiceManager,
	resource *schv1beta1.ServiceConnector,
	namespace string,
) (schsdk.CreateServiceConnectorDetails, error) {
	if resource == nil {
		return schsdk.CreateServiceConnectorDetails{}, fmt.Errorf("ServiceConnector resource is nil")
	}
	desired, err := serviceConnectorDesiredValues(ctx, manager, resource, namespace)
	if err != nil {
		return schsdk.CreateServiceConnectorDetails{}, err
	}
	var details schsdk.CreateServiceConnectorDetails
	if err := decodeServiceConnectorDetails(desired, &details); err != nil {
		return schsdk.CreateServiceConnectorDetails{}, err
	}
	return details, nil
}

func buildServiceConnectorUpdateDetails(
	ctx context.Context,
	manager *ServiceConnectorServiceManager,
	resource *schv1beta1.ServiceConnector,
	namespace string,
	currentResponse any,
) (schsdk.UpdateServiceConnectorDetails, bool, error) {
	if resource == nil {
		return schsdk.UpdateServiceConnectorDetails{}, false, fmt.Errorf("ServiceConnector resource is nil")
	}
	current, ok := serviceConnectorFromResponse(currentResponse)
	if !ok {
		return schsdk.UpdateServiceConnectorDetails{}, false, fmt.Errorf("current ServiceConnector response does not expose a ServiceConnector body")
	}
	if err := validateServiceConnectorCreateOnlyDrift(resource, current); err != nil {
		return schsdk.UpdateServiceConnectorDetails{}, false, err
	}

	desiredValues, err := serviceConnectorDesiredValues(ctx, manager, resource, namespace)
	if err != nil {
		return schsdk.UpdateServiceConnectorDetails{}, false, err
	}
	var desiredDetails schsdk.UpdateServiceConnectorDetails
	if err := decodeServiceConnectorDetails(desiredValues, &desiredDetails); err != nil {
		return schsdk.UpdateServiceConnectorDetails{}, false, err
	}
	currentValues, err := serviceConnectorCurrentValues(current)
	if err != nil {
		return schsdk.UpdateServiceConnectorDetails{}, false, err
	}

	details := schsdk.UpdateServiceConnectorDetails{}
	updateNeeded := populateServiceConnectorScalarUpdates(&details, current, resource)
	updateNeeded = populateServiceConnectorPolymorphicUpdates(&details, desiredDetails, desiredValues, currentValues) || updateNeeded
	updateNeeded = populateServiceConnectorTagUpdates(&details, current, resource) || updateNeeded
	return details, updateNeeded, nil
}

func populateServiceConnectorScalarUpdates(
	details *schsdk.UpdateServiceConnectorDetails,
	current schsdk.ServiceConnector,
	resource *schv1beta1.ServiceConnector,
) bool {
	updateNeeded := false
	if stringValue(current.DisplayName) != resource.Spec.DisplayName {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if stringValue(current.Description) != resource.Spec.Description {
		details.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}
	return updateNeeded
}

func populateServiceConnectorPolymorphicUpdates(
	details *schsdk.UpdateServiceConnectorDetails,
	desiredDetails schsdk.UpdateServiceConnectorDetails,
	desiredValues map[string]any,
	currentValues map[string]any,
) bool {
	updateNeeded := false
	if serviceConnectorDesiredFieldDiffers(desiredValues, currentValues, "source") {
		details.Source = desiredDetails.Source
		updateNeeded = true
	}
	if serviceConnectorDesiredFieldDiffers(desiredValues, currentValues, "target") {
		details.Target = desiredDetails.Target
		updateNeeded = true
	}
	if serviceConnectorDesiredFieldDiffers(desiredValues, currentValues, "tasks") {
		details.Tasks = desiredDetails.Tasks
		updateNeeded = true
	}
	return updateNeeded
}

func populateServiceConnectorTagUpdates(
	details *schsdk.UpdateServiceConnectorDetails,
	current schsdk.ServiceConnector,
	resource *schv1beta1.ServiceConnector,
) bool {
	updateNeeded := false
	if resource.Spec.FreeformTags != nil && !maps.Equal(resource.Spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := serviceConnectorDefinedTagsFromSpec(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(desiredDefinedTags, current.DefinedTags) {
			details.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	return updateNeeded
}

func serviceConnectorDesiredValues(
	ctx context.Context,
	manager *ServiceConnectorServiceManager,
	resource *schv1beta1.ServiceConnector,
	namespace string,
) (map[string]any, error) {
	var credentialClient credhelper.CredentialClient
	if manager != nil {
		credentialClient = manager.CredentialClient
	}
	resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credentialClient, namespace)
	if err != nil {
		return nil, err
	}
	values, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("resolved ServiceConnector spec has type %T, want map[string]any", resolved)
	}
	return values, nil
}

func decodeServiceConnectorDetails(source map[string]any, target any) error {
	payload, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("marshal ServiceConnector details: %w", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode ServiceConnector details: %w", err)
	}
	return nil
}

func validateServiceConnectorCreateOnlyDrift(resource *schv1beta1.ServiceConnector, current schsdk.ServiceConnector) error {
	if stringValue(current.CompartmentId) == resource.Spec.CompartmentId {
		return nil
	}
	return fmt.Errorf(
		"ServiceConnector create-only drift detected for compartmentId; replace the resource or restore the desired spec before update",
	)
}

func serviceConnectorFromResponse(response any) (schsdk.ServiceConnector, bool) {
	switch current := response.(type) {
	case schsdk.ServiceConnector:
		return current, true
	case *schsdk.ServiceConnector:
		if current == nil {
			return schsdk.ServiceConnector{}, false
		}
		return *current, true
	case schsdk.GetServiceConnectorResponse:
		return current.ServiceConnector, true
	case *schsdk.GetServiceConnectorResponse:
		if current == nil {
			return schsdk.ServiceConnector{}, false
		}
		return current.ServiceConnector, true
	default:
		return schsdk.ServiceConnector{}, false
	}
}

func serviceConnectorCurrentValues(current schsdk.ServiceConnector) (map[string]any, error) {
	payload, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("marshal current ServiceConnector: %w", err)
	}
	values := map[string]any{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode current ServiceConnector: %w", err)
	}
	return values, nil
}

func serviceConnectorDesiredFieldDiffers(desiredValues map[string]any, currentValues map[string]any, field string) bool {
	desired, ok := desiredValues[field]
	if !ok {
		return false
	}
	current := currentValues[field]
	return !serviceConnectorDesiredMatchesCurrent(desired, current)
}

func serviceConnectorDesiredMatchesCurrent(desired any, current any) bool {
	desired = serviceConnectorComparableValue(desired)
	current = serviceConnectorComparableValue(current)

	switch desiredValue := desired.(type) {
	case map[string]any:
		return serviceConnectorDesiredMapMatchesCurrent(desiredValue, current)
	case []any:
		return serviceConnectorDesiredSliceMatchesCurrent(desiredValue, current)
	default:
		return serviceConnectorDesiredScalarMatchesCurrent(desired, current)
	}
}

func serviceConnectorComparableValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return serviceConnectorComparableMap(typed)
	case []any:
		return serviceConnectorComparableSlice(typed)
	default:
		return typed
	}
}

func serviceConnectorDesiredMapMatchesCurrent(desired map[string]any, current any) bool {
	currentMap, ok := current.(map[string]any)
	if !ok {
		return serviceConnectorZeroComparable(desired)
	}
	for key, childDesired := range desired {
		childCurrent, exists := currentMap[key]
		if !exists && serviceConnectorZeroComparable(childDesired) {
			continue
		}
		if !serviceConnectorDesiredMatchesCurrent(childDesired, childCurrent) {
			return false
		}
	}
	return true
}

func serviceConnectorDesiredSliceMatchesCurrent(desired []any, current any) bool {
	if len(desired) == 0 {
		return true
	}
	currentSlice, ok := current.([]any)
	if !ok || len(desired) != len(currentSlice) {
		return false
	}
	for i := range desired {
		if !serviceConnectorDesiredMatchesCurrent(desired[i], currentSlice[i]) {
			return false
		}
	}
	return true
}

func serviceConnectorDesiredScalarMatchesCurrent(desired any, current any) bool {
	if serviceConnectorZeroComparable(desired) && serviceConnectorZeroComparable(current) {
		return true
	}
	return reflect.DeepEqual(desired, current)
}

func serviceConnectorComparableMap(value map[string]any) map[string]any {
	normalized := make(map[string]any, len(value))
	for key, child := range value {
		if key == "privateEndpointMetadata" {
			continue
		}
		child = serviceConnectorComparableValue(child)
		if serviceConnectorPrunableComparable(child) {
			continue
		}
		normalized[key] = child
	}
	return normalized
}

func serviceConnectorComparableSlice(value []any) []any {
	normalized := make([]any, 0, len(value))
	for _, child := range value {
		child = serviceConnectorComparableValue(child)
		if serviceConnectorPrunableComparable(child) {
			continue
		}
		normalized = append(normalized, child)
	}
	return normalized
}

func serviceConnectorZeroComparable(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case bool:
		return !typed
	case float64:
		return typed == 0
	case map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func serviceConnectorPrunableComparable(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func serviceConnectorDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		nested := make(map[string]interface{}, len(values))
		for key, value := range values {
			nested[key] = value
		}
		converted[namespace] = nested
	}
	return converted
}

func guardServiceConnectorExistingBeforeCreate(_ context.Context, resource *schv1beta1.ServiceConnector) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ServiceConnector resource is nil")
	}
	if resource.Spec.CompartmentId == "" || resource.Spec.DisplayName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapServiceConnectorListPagination(hooks *ServiceConnectorRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	listCall := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
		return listServiceConnectorPages(ctx, listCall, request)
	}
}

func listServiceConnectorPages(
	ctx context.Context,
	call func(context.Context, schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error),
	request schsdk.ListServiceConnectorsRequest,
) (schsdk.ListServiceConnectorsResponse, error) {
	seenPages := map[string]struct{}{}
	var combined schsdk.ListServiceConnectorsResponse
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

		nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return schsdk.ListServiceConnectorsResponse{}, fmt.Errorf("ServiceConnector list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

func getServiceConnectorWorkRequest(
	ctx context.Context,
	client serviceConnectorWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize ServiceConnector OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("ServiceConnector OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, schsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveServiceConnectorGeneratedWorkRequestAction(workRequest any) (string, error) {
	serviceConnectorWorkRequest, err := serviceConnectorWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveServiceConnectorWorkRequestAction(serviceConnectorWorkRequest)
}

func resolveServiceConnectorGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	serviceConnectorWorkRequest, err := serviceConnectorWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := serviceConnectorWorkRequestPhaseFromOperationType(serviceConnectorWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverServiceConnectorIDFromGeneratedWorkRequest(
	_ *schv1beta1.ServiceConnector,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	serviceConnectorWorkRequest, err := serviceConnectorWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveServiceConnectorIDFromWorkRequest(serviceConnectorWorkRequest, serviceConnectorWorkRequestActionForPhase(phase))
}

func serviceConnectorGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	serviceConnectorWorkRequest, err := serviceConnectorWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("ServiceConnector %s work request %s is %s", phase, stringValue(serviceConnectorWorkRequest.Id), serviceConnectorWorkRequest.Status)
}

func serviceConnectorWorkRequestFromAny(workRequest any) (schsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case schsdk.WorkRequest:
		return current, nil
	case *schsdk.WorkRequest:
		if current == nil {
			return schsdk.WorkRequest{}, fmt.Errorf("ServiceConnector work request is nil")
		}
		return *current, nil
	default:
		return schsdk.WorkRequest{}, fmt.Errorf("unexpected ServiceConnector work request type %T", workRequest)
	}
}

func serviceConnectorWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) schsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return schsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return schsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return schsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func serviceConnectorWorkRequestPhaseFromOperationType(operationType schsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case schsdk.OperationTypeCreateServiceConnector:
		return shared.OSOKAsyncPhaseCreate, true
	case schsdk.OperationTypeUpdateServiceConnector:
		return shared.OSOKAsyncPhaseUpdate, true
	case schsdk.OperationTypeDeleteServiceConnector:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func resolveServiceConnectorIDFromWorkRequest(workRequest schsdk.WorkRequest, action schsdk.ActionTypeEnum) (string, error) {
	for _, resource := range workRequest.Resources {
		if !isServiceConnectorWorkRequestResource(resource) {
			continue
		}
		if action != "" && resource.ActionType != action {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	for _, resource := range workRequest.Resources {
		if !isServiceConnectorWorkRequestResource(resource) {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("ServiceConnector work request %s does not expose a ServiceConnector identifier", stringValue(workRequest.Id))
}

func resolveServiceConnectorWorkRequestAction(workRequest schsdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isServiceConnectorWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("ServiceConnector work request %s exposes conflicting ServiceConnector action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func isServiceConnectorWorkRequestResource(resource schsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	entityType = strings.NewReplacer("_", "", "-", "", " ", "").Replace(entityType)
	if entityType == "serviceconnector" || entityType == "serviceconnectors" {
		return true
	}
	if entityType != "" {
		return false
	}
	return serviceConnectorLooksLikeServiceConnectorID(resource.Identifier) ||
		strings.Contains(normalizedServiceConnectorWorkRequestText(resource.EntityUri), "serviceconnectors")
}

func serviceConnectorLooksLikeServiceConnectorID(identifier *string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(stringValue(identifier))), "ocid1.serviceconnector.")
}

func normalizedServiceConnectorWorkRequestText(value *string) string {
	text := strings.ToLower(strings.TrimSpace(stringValue(value)))
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(text)
}

func rejectServiceConnectorAuthShapedNotFound(resource *schv1beta1.ServiceConnector, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return fmt.Errorf("ServiceConnector delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted")
}

func wrapServiceConnectorDeleteConfirmation(manager *ServiceConnectorServiceManager, hooks *ServiceConnectorRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	getServiceConnector := hooks.Get.Call
	getWorkRequest := hooks.Async.GetWorkRequest
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ServiceConnectorServiceClient) ServiceConnectorServiceClient {
		return serviceConnectorDeleteConfirmationClient{
			delegate:            delegate,
			getServiceConnector: getServiceConnector,
			getWorkRequest:      getWorkRequest,
			log:                 log,
		}
	})
}

type serviceConnectorDeleteConfirmationClient struct {
	delegate            ServiceConnectorServiceClient
	getServiceConnector func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error)
	getWorkRequest      func(context.Context, string) (any, error)
	log                 loggerutil.OSOKLogger
}

func (c serviceConnectorDeleteConfirmationClient) CreateOrUpdate(ctx context.Context, resource *schv1beta1.ServiceConnector, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c serviceConnectorDeleteConfirmationClient) Delete(ctx context.Context, resource *schv1beta1.ServiceConnector) (bool, error) {
	if deleted, handled, err := c.resumeDeleteWorkRequestForDelete(ctx, resource); handled {
		return deleted, err
	}
	if deleted, handled, err := c.resumeWriteWorkRequestForDelete(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c serviceConnectorDeleteConfirmationClient) resumeDeleteWorkRequestForDelete(ctx context.Context, resource *schv1beta1.ServiceConnector) (bool, bool, error) {
	workRequestID := currentServiceConnectorDeleteWorkRequest(resource)
	if workRequestID == "" {
		return false, false, nil
	}
	if c.getWorkRequest == nil {
		return false, true, fmt.Errorf("ServiceConnector work request polling is not configured")
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, true, err
	}
	current, err := buildServiceConnectorWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, true, c.failWriteWorkRequestForDelete(resource, nil, err)
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markDeleteWorkRequestPending(resource, current, workRequestID)
		return false, true, nil
	case shared.OSOKAsyncClassSucceeded:
		deleted, err := c.confirmSucceededDeleteWorkRequest(ctx, resource, workRequest, current, workRequestID)
		return deleted, true, err
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("ServiceConnector delete work request %s finished with status %s", workRequestID, current.RawStatus)
		return false, true, c.failWriteWorkRequestForDelete(resource, current, err)
	default:
		err := fmt.Errorf("ServiceConnector delete work request %s projected unsupported async class %s", workRequestID, current.NormalizedClass)
		return false, true, c.failWriteWorkRequestForDelete(resource, current, err)
	}
}

func (c serviceConnectorDeleteConfirmationClient) resumeWriteWorkRequestForDelete(ctx context.Context, resource *schv1beta1.ServiceConnector) (bool, bool, error) {
	workRequestID, phase := currentServiceConnectorWriteWorkRequest(resource)
	if workRequestID == "" {
		return false, false, nil
	}
	if c.getWorkRequest == nil {
		return false, true, fmt.Errorf("ServiceConnector work request polling is not configured")
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, true, err
	}
	current, err := buildServiceConnectorWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return false, true, c.failWriteWorkRequestForDelete(resource, nil, err)
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("ServiceConnector %s work request %s is still in progress; waiting before delete", current.Phase, workRequestID)
		c.markWriteWorkRequestForDelete(resource, current, shared.OSOKAsyncClassPending, message)
		return false, true, nil
	case shared.OSOKAsyncClassSucceeded:
		deleted, err := c.deleteAfterSucceededWriteWorkRequest(ctx, resource, workRequest, current, workRequestID)
		return deleted, true, err
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("ServiceConnector %s work request %s finished with status %s before delete", current.Phase, workRequestID, current.RawStatus)
		return false, true, c.failWriteWorkRequestForDelete(resource, current, err)
	default:
		err := fmt.Errorf("ServiceConnector %s work request %s projected unsupported async class %s before delete", current.Phase, workRequestID, current.NormalizedClass)
		return false, true, c.failWriteWorkRequestForDelete(resource, current, err)
	}
}

func currentServiceConnectorWriteWorkRequest(resource *schv1beta1.ServiceConnector) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		if workRequestID := strings.TrimSpace(current.WorkRequestID); workRequestID != "" {
			return workRequestID, current.Phase
		}
	}
	return "", ""
}

func currentServiceConnectorDeleteWorkRequest(resource *schv1beta1.ServiceConnector) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func buildServiceConnectorWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	serviceConnectorWorkRequest, err := serviceConnectorWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	derivedPhase, ok := serviceConnectorWorkRequestPhaseFromOperationType(serviceConnectorWorkRequest.OperationType)
	if ok {
		if phase != "" && phase != derivedPhase {
			return nil, fmt.Errorf(
				"ServiceConnector work request %s exposes phase %q while delete expected %q",
				stringValue(serviceConnectorWorkRequest.Id),
				derivedPhase,
				phase,
			)
		}
		phase = derivedPhase
	}
	action, err := resolveServiceConnectorWorkRequestAction(serviceConnectorWorkRequest)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, serviceConnectorWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(serviceConnectorWorkRequest.Status),
		RawAction:        action,
		RawOperationType: string(serviceConnectorWorkRequest.OperationType),
		WorkRequestID:    stringValue(serviceConnectorWorkRequest.Id),
		PercentComplete:  serviceConnectorWorkRequest.PercentComplete,
		FallbackPhase:    phase,
	})
}

func (c serviceConnectorDeleteConfirmationClient) deleteAfterSucceededWriteWorkRequest(
	ctx context.Context,
	resource *schv1beta1.ServiceConnector,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID := trackedServiceConnectorID(resource)
	if resourceID == "" {
		recoveredID, err := recoverServiceConnectorIDFromGeneratedWorkRequest(resource, workRequest, current.Phase)
		if err != nil {
			return false, c.failWriteWorkRequestForDelete(resource, current, err)
		}
		resourceID = strings.TrimSpace(recoveredID)
	}
	if resourceID == "" {
		err := fmt.Errorf("ServiceConnector %s work request %s did not expose a ServiceConnector identifier", current.Phase, workRequestID)
		return false, c.failWriteWorkRequestForDelete(resource, current, err)
	}
	if c.getServiceConnector == nil {
		return false, c.failWriteWorkRequestForDelete(resource, current, fmt.Errorf("ServiceConnector readback is not configured"))
	}

	_, err := c.getServiceConnector(ctx, schsdk.GetServiceConnectorRequest{ServiceConnectorId: common.String(resourceID)})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		if classification.IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, fmt.Errorf("ServiceConnector write work request readback returned ambiguous 404 NotAuthorizedOrNotFound before delete: %w", err)
		}
		if classification.IsUnambiguousNotFound() {
			c.markWriteWorkRequestReadbackPending(resource, current, workRequestID, resourceID)
			return false, nil
		}
		return false, c.failWriteWorkRequestForDelete(resource, current, err)
	}

	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.delegate.Delete(ctx, resource)
}

func (c serviceConnectorDeleteConfirmationClient) confirmSucceededDeleteWorkRequest(
	ctx context.Context,
	resource *schv1beta1.ServiceConnector,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID := trackedServiceConnectorID(resource)
	if resourceID == "" {
		recoveredID, err := recoverServiceConnectorIDFromGeneratedWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err != nil {
			return false, c.failWriteWorkRequestForDelete(resource, current, err)
		}
		resourceID = strings.TrimSpace(recoveredID)
	}
	if resourceID == "" || c.getServiceConnector == nil {
		return c.delegate.Delete(ctx, resource)
	}

	_, err := c.getServiceConnector(ctx, schsdk.GetServiceConnectorRequest{ServiceConnectorId: common.String(resourceID)})
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return c.delegate.Delete(ctx, resource)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		err = fmt.Errorf(
			"ServiceConnector delete work request %s succeeded but confirmation read returned ambiguous 404 NotAuthorizedOrNotFound: %w",
			strings.TrimSpace(workRequestID),
			err,
		)
		return false, c.failWriteWorkRequestForDelete(resource, current, err)
	}
	return false, c.failWriteWorkRequestForDelete(resource, current, err)
}

func (c serviceConnectorDeleteConfirmationClient) markDeleteWorkRequestPending(
	resource *schv1beta1.ServiceConnector,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) {
	message := fmt.Sprintf("ServiceConnector delete work request %s is still in progress", strings.TrimSpace(workRequestID))
	c.markWriteWorkRequestForDelete(resource, current, shared.OSOKAsyncClassPending, message)
}

func (c serviceConnectorDeleteConfirmationClient) markWriteWorkRequestReadbackPending(
	resource *schv1beta1.ServiceConnector,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) {
	message := fmt.Sprintf(
		"ServiceConnector %s work request %s succeeded; waiting for ServiceConnector %s to become readable before delete",
		current.Phase,
		strings.TrimSpace(workRequestID),
		strings.TrimSpace(resourceID),
	)
	if resourceID := strings.TrimSpace(resourceID); resourceID != "" {
		resource.Status.Id = resourceID
		resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	}
	c.markWriteWorkRequestForDelete(resource, current, shared.OSOKAsyncClassPending, message)
}

func (c serviceConnectorDeleteConfirmationClient) markWriteWorkRequestForDelete(
	resource *schv1beta1.ServiceConnector,
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
	next.UpdatedAt = nil
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func (c serviceConnectorDeleteConfirmationClient) failWriteWorkRequestForDelete(
	resource *schv1beta1.ServiceConnector,
	current *shared.OSOKAsyncOperation,
	err error,
) error {
	if resource == nil || err == nil {
		return err
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if current == nil {
		return err
	}
	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}
	c.markWriteWorkRequestForDelete(resource, current, class, err.Error())
	return err
}

func (c serviceConnectorDeleteConfirmationClient) rejectAuthShapedConfirmRead(ctx context.Context, resource *schv1beta1.ServiceConnector) error {
	if c.getServiceConnector == nil || resource == nil {
		return nil
	}
	serviceConnectorID := trackedServiceConnectorID(resource)
	if serviceConnectorID == "" {
		return nil
	}
	_, err := c.getServiceConnector(ctx, schsdk.GetServiceConnectorRequest{ServiceConnectorId: common.String(serviceConnectorID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("ServiceConnector delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func trackedServiceConnectorID(resource *schv1beta1.ServiceConnector) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
