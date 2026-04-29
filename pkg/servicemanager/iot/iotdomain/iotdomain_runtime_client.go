/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package iotdomain

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

const iotDomainDeletePendingMessage = "OCI IotDomain delete is in progress"

var iotDomainWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(iotsdk.OperationStatusAccepted),
		string(iotsdk.OperationStatusInProgress),
		string(iotsdk.OperationStatusWaiting),
	},
	SucceededStatusTokens: []string{string(iotsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(iotsdk.OperationStatusFailed)},
	AttentionStatusTokens: []string{string(iotsdk.OperationStatusNeedsAttention)},
	CreateActionTokens: []string{
		string(iotsdk.OperationTypeCreateIotDomain),
		string(iotsdk.ActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(iotsdk.OperationTypeUpdateIotDomain),
		string(iotsdk.ActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(iotsdk.OperationTypeDeleteIotDomain),
		string(iotsdk.ActionTypeDeleted),
	},
}

type iotDomainOCIClient interface {
	CreateIotDomain(context.Context, iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error)
	GetIotDomain(context.Context, iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error)
	ListIotDomains(context.Context, iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error)
	UpdateIotDomain(context.Context, iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error)
	DeleteIotDomain(context.Context, iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error)
	GetWorkRequest(context.Context, iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error)
}

type iotDomainWorkRequestClient interface {
	GetWorkRequest(context.Context, iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error)
}

type iotDomainAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e iotDomainAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e iotDomainAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerIotDomainRuntimeHooksMutator(func(manager *IotDomainServiceManager, hooks *IotDomainRuntimeHooks) {
		client, initErr := newIotDomainOCIClient(manager)
		applyIotDomainRuntimeHooks(hooks, client, initErr)
	})
}

func newIotDomainOCIClient(manager *IotDomainServiceManager) (iotDomainOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("IotDomain service manager is nil")
	}
	client, err := iotsdk.NewIotClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyIotDomainRuntimeHooks(
	hooks *IotDomainRuntimeHooks,
	client iotDomainOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = iotDomainRuntimeSemantics()
	hooks.BuildCreateBody = buildIotDomainCreateBody
	hooks.BuildUpdateBody = buildIotDomainUpdateBody
	hooks.List.Fields = iotDomainListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listIotDomainsAllPages(hooks.List.Call)
	}
	hooks.Async.Adapter = iotDomainWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getIotDomainWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveIotDomainWorkRequestAction
	hooks.Async.ResolvePhase = resolveIotDomainWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverIotDomainIDFromWorkRequest
	hooks.Async.Message = iotDomainWorkRequestMessage
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateIotDomainCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleIotDomainDeleteError
	hooks.StatusHooks.MarkTerminating = markIotDomainTerminating
	wrapIotDomainDeleteConfirmation(hooks, client, initErr)
}

func newIotDomainServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client iotDomainOCIClient,
) IotDomainServiceClient {
	manager := &IotDomainServiceManager{Log: log}
	hooks := newIotDomainRuntimeHooksWithOCIClient(client)
	applyIotDomainRuntimeHooks(&hooks, client, nil)
	delegate := defaultIotDomainServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*iotv1beta1.IotDomain](
			buildIotDomainGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapIotDomainGeneratedClient(hooks, delegate)
}

func newIotDomainRuntimeHooksWithOCIClient(client iotDomainOCIClient) IotDomainRuntimeHooks {
	hooks := newIotDomainDefaultRuntimeHooks(iotsdk.IotClient{})
	hooks.Create.Call = func(ctx context.Context, request iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error) {
		if client == nil {
			return iotsdk.CreateIotDomainResponse{}, fmt.Errorf("IotDomain OCI client is not configured")
		}
		return client.CreateIotDomain(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
		if client == nil {
			return iotsdk.GetIotDomainResponse{}, fmt.Errorf("IotDomain OCI client is not configured")
		}
		return client.GetIotDomain(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
		if client == nil {
			return iotsdk.ListIotDomainsResponse{}, fmt.Errorf("IotDomain OCI client is not configured")
		}
		return client.ListIotDomains(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error) {
		if client == nil {
			return iotsdk.UpdateIotDomainResponse{}, fmt.Errorf("IotDomain OCI client is not configured")
		}
		return client.UpdateIotDomain(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
		if client == nil {
			return iotsdk.DeleteIotDomainResponse{}, fmt.Errorf("IotDomain OCI client is not configured")
		}
		return client.DeleteIotDomain(ctx, request)
	}
	return hooks
}

func iotDomainRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "iot",
		FormalSlug:    "iotdomain",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(iotsdk.IotDomainLifecycleStateCreating)},
			UpdatingStates:     []string{string(iotsdk.IotDomainLifecycleStateUpdating)},
			ActiveStates:       []string{string(iotsdk.IotDomainLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(iotsdk.IotDomainLifecycleStateDeleting)},
			TerminalStates: []string{string(iotsdk.IotDomainLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "iotDomainGroupId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"iotDomainGroupId", "compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IotDomain", Action: "CreateIotDomain"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IotDomain", Action: "UpdateIotDomain"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IotDomain", Action: "DeleteIotDomain"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func iotDomainListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "IotDomainGroupId",
			RequestName:  "iotDomainGroupId",
			Contribution: "query",
			LookupPaths:  []string{"status.iotDomainGroupId", "spec.iotDomainGroupId", "iotDomainGroupId"},
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

func buildIotDomainCreateBody(_ context.Context, resource *iotv1beta1.IotDomain, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("IotDomain resource is nil")
	}
	if strings.TrimSpace(resource.Spec.IotDomainGroupId) == "" {
		return nil, fmt.Errorf("iotDomainGroupId is required")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("compartmentId is required")
	}

	body := iotsdk.CreateIotDomainDetails{
		IotDomainGroupId: common.String(strings.TrimSpace(resource.Spec.IotDomainGroupId)),
		CompartmentId:    common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
	}
	setIotDomainMutableCreateFields(&body, resource.Spec)
	return body, nil
}

func setIotDomainMutableCreateFields(
	body *iotsdk.CreateIotDomainDetails,
	spec iotv1beta1.IotDomainSpec,
) {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(spec.Description); value != "" {
		body.Description = common.String(value)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneIotDomainStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = iotDomainDefinedTags(spec.DefinedTags)
	}
}

func buildIotDomainUpdateBody(
	_ context.Context,
	resource *iotv1beta1.IotDomain,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("IotDomain resource is nil")
	}
	current, ok := iotDomainBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current IotDomain response does not expose an IotDomain body")
	}

	body := iotsdk.UpdateIotDomainDetails{}
	updateNeeded := false
	setIotDomainStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	setIotDomainStringUpdate(&body.Description, &updateNeeded, resource.Spec.Description, current.Description)
	if resource.Spec.FreeformTags != nil {
		desired := cloneIotDomainStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := iotDomainDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func setIotDomainStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func validateIotDomainCreateOnlyDrift(resource *iotv1beta1.IotDomain, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("IotDomain resource is nil")
	}
	current, ok := iotDomainBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	if err := rejectIotDomainCreateOnlyDrift(
		"iotDomainGroupId",
		resource.Spec.IotDomainGroupId,
		current.IotDomainGroupId,
		resource.Status.IotDomainGroupId,
	); err != nil {
		return err
	}
	return rejectIotDomainCreateOnlyDrift(
		"compartmentId",
		resource.Spec.CompartmentId,
		current.CompartmentId,
		resource.Status.CompartmentId,
	)
}

func rejectIotDomainCreateOnlyDrift(field string, desired string, observed *string, recorded string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(recorded)
	if observed != nil && strings.TrimSpace(*observed) != "" {
		current = strings.TrimSpace(*observed)
	}
	if desired == "" && current != "" {
		return fmt.Errorf("IotDomain formal semantics require replacement when %s changes", field)
	}
	if desired != "" && current != "" && desired != current {
		return fmt.Errorf("IotDomain formal semantics require replacement when %s changes", field)
	}
	return nil
}

func iotDomainBodyFromResponse(response any) (iotsdk.IotDomain, bool) {
	switch current := response.(type) {
	case iotsdk.CreateIotDomainResponse:
		return current.IotDomain, true
	case *iotsdk.CreateIotDomainResponse:
		return iotDomainCreateResponseBody(current)
	case iotsdk.GetIotDomainResponse:
		return current.IotDomain, true
	case *iotsdk.GetIotDomainResponse:
		return iotDomainGetResponseBody(current)
	case iotsdk.IotDomain:
		return current, true
	case *iotsdk.IotDomain:
		return iotDomainPointerBody(current)
	default:
		return iotsdk.IotDomain{}, false
	}
}

func iotDomainCreateResponseBody(response *iotsdk.CreateIotDomainResponse) (iotsdk.IotDomain, bool) {
	if response == nil {
		return iotsdk.IotDomain{}, false
	}
	return response.IotDomain, true
}

func iotDomainGetResponseBody(response *iotsdk.GetIotDomainResponse) (iotsdk.IotDomain, bool) {
	if response == nil {
		return iotsdk.IotDomain{}, false
	}
	return response.IotDomain, true
}

func iotDomainPointerBody(response *iotsdk.IotDomain) (iotsdk.IotDomain, bool) {
	if response == nil {
		return iotsdk.IotDomain{}, false
	}
	return *response, true
}

func listIotDomainsAllPages(
	call func(context.Context, iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error),
) func(context.Context, iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
	return func(ctx context.Context, request iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
		var combined iotsdk.ListIotDomainsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return iotsdk.ListIotDomainsResponse{}, err
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

func getIotDomainWorkRequest(
	ctx context.Context,
	client iotDomainWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize IotDomain OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("IotDomain OCI client is not configured")
	}
	if strings.TrimSpace(workRequestID) == "" {
		return nil, fmt.Errorf("IotDomain work request id is empty")
	}
	response, err := client.GetWorkRequest(ctx, iotsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveIotDomainWorkRequestAction(workRequest any) (string, error) {
	current, err := iotDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveIotDomainWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := iotDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	return iotDomainWorkRequestPhaseFromOperationType(current.OperationType)
}

func iotDomainWorkRequestPhaseFromOperationType(operation iotsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool, error) {
	switch operation {
	case iotsdk.OperationTypeCreateIotDomain:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case iotsdk.OperationTypeUpdateIotDomain:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case iotsdk.OperationTypeDeleteIotDomain:
		return shared.OSOKAsyncPhaseDelete, true, nil
	case "":
		return "", false, nil
	default:
		return "", false, fmt.Errorf("IotDomain work request operation %q is not modeled", operation)
	}
}

func recoverIotDomainIDFromWorkRequest(
	_ *iotv1beta1.IotDomain,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := iotDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if resolvedPhase, ok, err := iotDomainWorkRequestPhaseFromOperationType(current.OperationType); err != nil {
		return "", err
	} else if ok && resolvedPhase != phase {
		return "", fmt.Errorf("IotDomain work request operation %q does not match phase %q", current.OperationType, phase)
	}
	for _, resource := range current.Resources {
		if !iotDomainWorkRequestResourceIsDomain(resource) {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	for _, resource := range current.Resources {
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("IotDomain work request %s did not expose an IotDomain identifier", stringValue(current.Id))
}

func iotDomainWorkRequestResourceIsDomain(resource iotsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(stringValue(resource.EntityType)), " ", ""))
	return strings.Contains(entityType, "iotdomain") && !strings.Contains(entityType, "iotdomaingroup")
}

func iotDomainWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := iotDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	workRequestID := strings.TrimSpace(stringValue(current.Id))
	status := strings.TrimSpace(string(current.Status))
	if workRequestID == "" || status == "" {
		return ""
	}
	return fmt.Sprintf("IotDomain %s work request %s is %s", phase, workRequestID, status)
}

func iotDomainWorkRequestFromAny(workRequest any) (iotsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case iotsdk.WorkRequest:
		return current, nil
	case *iotsdk.WorkRequest:
		if current == nil {
			return iotsdk.WorkRequest{}, fmt.Errorf("IotDomain work request is nil")
		}
		return *current, nil
	default:
		return iotsdk.WorkRequest{}, fmt.Errorf("expected IotDomain work request, got %T", workRequest)
	}
}

func handleIotDomainDeleteError(resource *iotv1beta1.IotDomain, err error) error {
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
	return iotDomainAmbiguousNotFoundError{
		message:      "IotDomain delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapIotDomainDeleteConfirmation(
	hooks *IotDomainRuntimeHooks,
	workRequestClient iotDomainWorkRequestClient,
	workRequestInitErr error,
) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getIotDomain := hooks.Get.Call
	hooks.Get.Call = rejectIotDomainAuthShapedGet(getIotDomain)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate IotDomainServiceClient) IotDomainServiceClient {
		return iotDomainDeleteConfirmationClient{
			delegate:           delegate,
			getIotDomain:       getIotDomain,
			workRequestClient:  workRequestClient,
			workRequestInitErr: workRequestInitErr,
		}
	})
}

func rejectIotDomainAuthShapedGet(
	call func(context.Context, iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error),
) func(context.Context, iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
	return func(ctx context.Context, request iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
		response, err := call(ctx, request)
		if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return response, err
		}
		return response, iotDomainAmbiguousNotFoundError{
			message:      "IotDomain read returned ambiguous 404 NotAuthorizedOrNotFound",
			opcRequestID: errorutil.OpcRequestID(err),
		}
	}
}

type iotDomainDeleteConfirmationClient struct {
	delegate           IotDomainServiceClient
	getIotDomain       func(context.Context, iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error)
	workRequestClient  iotDomainWorkRequestClient
	workRequestInitErr error
}

func (c iotDomainDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *iotv1beta1.IotDomain,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c iotDomainDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *iotv1beta1.IotDomain,
) (bool, error) {
	if iotDomainPendingDeleteWorkRequestID(resource) != "" {
		if err := c.rejectAuthShapedSucceededDeleteWorkRequest(ctx, resource); err != nil {
			return false, err
		}
		return c.delegate.Delete(ctx, resource)
	}
	if err := c.rejectAuthShapedPreDeleteConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c iotDomainDeleteConfirmationClient) rejectAuthShapedPreDeleteConfirmRead(
	ctx context.Context,
	resource *iotv1beta1.IotDomain,
) error {
	if c.getIotDomain == nil || resource == nil {
		return nil
	}
	domainID := trackedIotDomainID(resource)
	if domainID == "" {
		return nil
	}
	_, err := c.getIotDomain(ctx, iotsdk.GetIotDomainRequest{IotDomainId: common.String(domainID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("IotDomain delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func (c iotDomainDeleteConfirmationClient) rejectAuthShapedSucceededDeleteWorkRequest(
	ctx context.Context,
	resource *iotv1beta1.IotDomain,
) error {
	workRequestID := iotDomainPendingDeleteWorkRequestID(resource)
	if workRequestID == "" || c.workRequestClient == nil {
		return nil
	}
	if !c.deleteWorkRequestSucceeded(ctx, workRequestID) {
		return nil
	}
	return c.rejectAuthShapedSucceededDeleteRead(ctx, resource, workRequestID)
}

func (c iotDomainDeleteConfirmationClient) deleteWorkRequestSucceeded(
	ctx context.Context,
	workRequestID string,
) bool {
	workRequest, err := getIotDomainWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, workRequestID)
	if err != nil {
		return false
	}
	currentWorkRequest, err := iotDomainWorkRequestFromAny(workRequest)
	if err != nil {
		return false
	}
	class, err := iotDomainWorkRequestAsyncAdapter.Normalize(string(currentWorkRequest.Status))
	return err == nil && class == shared.OSOKAsyncClassSucceeded
}

func (c iotDomainDeleteConfirmationClient) rejectAuthShapedSucceededDeleteRead(
	ctx context.Context,
	resource *iotv1beta1.IotDomain,
	workRequestID string,
) error {
	if c.getIotDomain == nil {
		return nil
	}
	domainID := trackedIotDomainID(resource)
	if domainID == "" {
		return nil
	}
	_, err := c.getIotDomain(ctx, iotsdk.GetIotDomainRequest{IotDomainId: common.String(domainID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("IotDomain delete work request %s succeeded but confirmation read returned ambiguous 404 NotAuthorizedOrNotFound: %w", workRequestID, err)
}

func trackedIotDomainID(resource *iotv1beta1.IotDomain) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func iotDomainPendingDeleteWorkRequestID(resource *iotv1beta1.IotDomain) string {
	if resource == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func markIotDomainTerminating(resource *iotv1beta1.IotDomain, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = iotDomainDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := iotDomainLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         iotDomainDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		iotDomainDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func iotDomainLifecycleState(response any) string {
	current, ok := iotDomainBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneIotDomainStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func iotDomainDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
