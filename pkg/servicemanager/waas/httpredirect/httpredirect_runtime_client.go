/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package httpredirect

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

const httpRedirectDeletePendingMessage = "OCI HttpRedirect delete is in progress"

var httpRedirectWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(waassdk.WorkRequestStatusValuesAccepted),
		string(waassdk.WorkRequestStatusValuesInProgress),
		string(waassdk.WorkRequestStatusValuesCanceling),
	},
	SucceededStatusTokens: []string{string(waassdk.WorkRequestStatusValuesSucceeded)},
	FailedStatusTokens:    []string{string(waassdk.WorkRequestStatusValuesFailed)},
	CanceledStatusTokens:  []string{string(waassdk.WorkRequestStatusValuesCanceled)},
	CreateActionTokens:    []string{string(waassdk.WorkRequestOperationTypesCreateHttpRedirect)},
	UpdateActionTokens:    []string{string(waassdk.WorkRequestOperationTypesUpdateHttpRedirect)},
	DeleteActionTokens:    []string{string(waassdk.WorkRequestOperationTypesDeleteHttpRedirect)},
}

type httpRedirectOCIClient interface {
	CreateHttpRedirect(context.Context, waassdk.CreateHttpRedirectRequest) (waassdk.CreateHttpRedirectResponse, error)
	GetHttpRedirect(context.Context, waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error)
	ListHttpRedirects(context.Context, waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error)
	UpdateHttpRedirect(context.Context, waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error)
	DeleteHttpRedirect(context.Context, waassdk.DeleteHttpRedirectRequest) (waassdk.DeleteHttpRedirectResponse, error)
}

type httpRedirectWorkRequestClient interface {
	GetWorkRequest(context.Context, waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error)
}

type httpRedirectAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e httpRedirectAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e httpRedirectAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerHttpRedirectRuntimeHooksMutator(func(manager *HttpRedirectServiceManager, hooks *HttpRedirectRuntimeHooks) {
		workRequestClient, initErr := newHttpRedirectWorkRequestClient(manager)
		applyHttpRedirectRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newHttpRedirectWorkRequestClient(manager *HttpRedirectServiceManager) (httpRedirectWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("HttpRedirect service manager is nil")
	}
	client, err := waassdk.NewWaasClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyHttpRedirectRuntimeHooks(
	hooks *HttpRedirectRuntimeHooks,
	workRequestClient httpRedirectWorkRequestClient,
	workRequestInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = httpRedirectRuntimeSemantics()
	hooks.BuildCreateBody = buildHttpRedirectCreateBody
	hooks.BuildUpdateBody = buildHttpRedirectUpdateBody
	hooks.List.Fields = httpRedirectListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listHttpRedirectsAllPages(hooks.List.Call)
	}
	hooks.Async.Adapter = httpRedirectWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getHttpRedirectWorkRequest(ctx, workRequestClient, workRequestInitErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveHttpRedirectWorkRequestAction
	hooks.Async.ResolvePhase = resolveHttpRedirectWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverHttpRedirectIDFromWorkRequest
	hooks.Async.Message = httpRedirectWorkRequestMessage
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateHttpRedirectCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleHttpRedirectDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyHttpRedirectDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markHttpRedirectTerminating
	wrapHttpRedirectDeleteConfirmation(hooks)
}

func newHttpRedirectServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client httpRedirectOCIClient,
) HttpRedirectServiceClient {
	manager := &HttpRedirectServiceManager{Log: log}
	hooks := newHttpRedirectRuntimeHooksWithOCIClient(client)
	applyHttpRedirectRuntimeHooks(&hooks, httpRedirectWorkRequestClientFromOCI(client), nil)
	delegate := defaultHttpRedirectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.HttpRedirect](
			buildHttpRedirectGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapHttpRedirectGeneratedClient(hooks, delegate)
}

func httpRedirectWorkRequestClientFromOCI(client httpRedirectOCIClient) httpRedirectWorkRequestClient {
	workRequestClient, _ := client.(httpRedirectWorkRequestClient)
	return workRequestClient
}

func newHttpRedirectRuntimeHooksWithOCIClient(client httpRedirectOCIClient) HttpRedirectRuntimeHooks {
	hooks := newHttpRedirectDefaultRuntimeHooks(waassdk.RedirectClient{})
	hooks.Create.Call = func(ctx context.Context, request waassdk.CreateHttpRedirectRequest) (waassdk.CreateHttpRedirectResponse, error) {
		if client == nil {
			return waassdk.CreateHttpRedirectResponse{}, fmt.Errorf("HttpRedirect OCI client is not configured")
		}
		return client.CreateHttpRedirect(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
		if client == nil {
			return waassdk.GetHttpRedirectResponse{}, fmt.Errorf("HttpRedirect OCI client is not configured")
		}
		return client.GetHttpRedirect(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error) {
		if client == nil {
			return waassdk.ListHttpRedirectsResponse{}, fmt.Errorf("HttpRedirect OCI client is not configured")
		}
		return client.ListHttpRedirects(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request waassdk.UpdateHttpRedirectRequest) (waassdk.UpdateHttpRedirectResponse, error) {
		if client == nil {
			return waassdk.UpdateHttpRedirectResponse{}, fmt.Errorf("HttpRedirect OCI client is not configured")
		}
		return client.UpdateHttpRedirect(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request waassdk.DeleteHttpRedirectRequest) (waassdk.DeleteHttpRedirectResponse, error) {
		if client == nil {
			return waassdk.DeleteHttpRedirectResponse{}, fmt.Errorf("HttpRedirect OCI client is not configured")
		}
		return client.DeleteHttpRedirect(ctx, request)
	}
	return hooks
}

func httpRedirectRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "waas",
		FormalSlug:    "httpredirect",
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
			ProvisioningStates: []string{string(waassdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(waassdk.LifecycleStatesUpdating)},
			ActiveStates:       []string{string(waassdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(waassdk.LifecycleStatesDeleting)},
			TerminalStates: []string{string(waassdk.LifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "domain"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "target", "responseCode", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "domain"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "HttpRedirect", Action: "CreateHttpRedirect"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "HttpRedirect", Action: "UpdateHttpRedirect"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "HttpRedirect", Action: "DeleteHttpRedirect"}},
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

func httpRedirectListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildHttpRedirectCreateBody(_ context.Context, resource *waasv1beta1.HttpRedirect, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("HttpRedirect resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("compartmentId is required")
	}
	if strings.TrimSpace(resource.Spec.Domain) == "" {
		return nil, fmt.Errorf("domain is required")
	}
	target, err := httpRedirectTargetFromSpec(resource.Spec.Target)
	if err != nil {
		return nil, err
	}

	body := waassdk.CreateHttpRedirectDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Domain:        common.String(strings.TrimSpace(resource.Spec.Domain)),
		Target:        target,
	}
	setHttpRedirectMutableCreateFields(&body, resource.Spec)
	return body, nil
}

func setHttpRedirectMutableCreateFields(
	body *waassdk.CreateHttpRedirectDetails,
	spec waasv1beta1.HttpRedirectSpec,
) {
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if spec.ResponseCode != 0 {
		body.ResponseCode = common.Int(spec.ResponseCode)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneHttpRedirectStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = httpRedirectDefinedTags(spec.DefinedTags)
	}
}

func buildHttpRedirectUpdateBody(
	_ context.Context,
	resource *waasv1beta1.HttpRedirect,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("HttpRedirect resource is nil")
	}
	current, ok := httpRedirectBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current HttpRedirect response does not expose an HttpRedirect body")
	}

	body := waassdk.UpdateHttpRedirectDetails{}
	updateNeeded := false
	setHttpRedirectStringUpdate(&body.DisplayName, &updateNeeded, resource.Spec.DisplayName, current.DisplayName)
	if err := setHttpRedirectTargetUpdate(&body, &updateNeeded, resource.Spec.Target, current.Target); err != nil {
		return nil, false, err
	}
	setHttpRedirectIntUpdate(&body.ResponseCode, &updateNeeded, resource.Spec.ResponseCode, current.ResponseCode)
	if resource.Spec.FreeformTags != nil {
		desired := cloneHttpRedirectStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := httpRedirectDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func setHttpRedirectStringUpdate(target **string, updateNeeded *bool, desired string, current *string) {
	value := strings.TrimSpace(desired)
	if value == "" {
		return
	}
	*target = common.String(value)
	if current == nil || strings.TrimSpace(*current) != value {
		*updateNeeded = true
	}
}

func setHttpRedirectIntUpdate(target **int, updateNeeded *bool, desired int, current *int) {
	if desired == 0 {
		return
	}
	*target = common.Int(desired)
	if current == nil || *current != desired {
		*updateNeeded = true
	}
}

func setHttpRedirectTargetUpdate(
	body *waassdk.UpdateHttpRedirectDetails,
	updateNeeded *bool,
	spec waasv1beta1.HttpRedirectTarget,
	current *waassdk.HttpRedirectTarget,
) error {
	desired, err := httpRedirectTargetFromSpec(spec)
	if err != nil {
		return err
	}
	body.Target = desired
	if !httpRedirectTargetsEqual(desired, current) {
		*updateNeeded = true
	}
	return nil
}

func httpRedirectTargetFromSpec(spec waasv1beta1.HttpRedirectTarget) (*waassdk.HttpRedirectTarget, error) {
	protocol := strings.ToUpper(strings.TrimSpace(spec.Protocol))
	if protocol == "" {
		return nil, fmt.Errorf("target.protocol is required")
	}
	mappedProtocol, ok := waassdk.GetMappingHttpRedirectTargetProtocolEnum(protocol)
	if !ok {
		return nil, fmt.Errorf("target.protocol %q is not supported", spec.Protocol)
	}
	if strings.TrimSpace(spec.Host) == "" {
		return nil, fmt.Errorf("target.host is required")
	}
	if spec.Port < 0 || spec.Port > 65535 {
		return nil, fmt.Errorf("target.port %d is outside the supported range", spec.Port)
	}

	target := &waassdk.HttpRedirectTarget{
		Protocol: mappedProtocol,
		Host:     common.String(strings.TrimSpace(spec.Host)),
		Path:     common.String(strings.TrimSpace(spec.Path)),
		Query:    common.String(strings.TrimSpace(spec.Query)),
	}
	if spec.Port != 0 {
		target.Port = common.Int(spec.Port)
	}
	return target, nil
}

func httpRedirectTargetsEqual(desired *waassdk.HttpRedirectTarget, current *waassdk.HttpRedirectTarget) bool {
	if desired == nil || current == nil {
		return desired == nil && current == nil
	}
	return desired.Protocol == current.Protocol &&
		stringPtrValue(desired.Host) == stringPtrValue(current.Host) &&
		stringPtrValue(desired.Path) == stringPtrValue(current.Path) &&
		stringPtrValue(desired.Query) == stringPtrValue(current.Query) &&
		intPtrValue(desired.Port) == intPtrValue(current.Port)
}

func validateHttpRedirectCreateOnlyDrift(
	resource *waasv1beta1.HttpRedirect,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("HttpRedirect resource is nil")
	}
	current, ok := httpRedirectBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	if err := rejectHttpRedirectCreateOnlyDrift(
		"compartmentId",
		resource.Spec.CompartmentId,
		current.CompartmentId,
		resource.Status.CompartmentId,
	); err != nil {
		return err
	}
	return rejectHttpRedirectCreateOnlyDrift(
		"domain",
		resource.Spec.Domain,
		current.Domain,
		resource.Status.Domain,
	)
}

func rejectHttpRedirectCreateOnlyDrift(field string, desired string, observed *string, recorded string) error {
	desired = strings.TrimSpace(desired)
	current := strings.TrimSpace(recorded)
	if observed != nil && strings.TrimSpace(*observed) != "" {
		current = strings.TrimSpace(*observed)
	}
	if desired == "" && current != "" {
		return fmt.Errorf("HttpRedirect formal semantics require replacement when %s changes", field)
	}
	if desired != "" && current != "" && desired != current {
		return fmt.Errorf("HttpRedirect formal semantics require replacement when %s changes", field)
	}
	return nil
}

func httpRedirectBodyFromResponse(response any) (waassdk.HttpRedirect, bool) {
	switch current := response.(type) {
	case waassdk.GetHttpRedirectResponse:
		return current.HttpRedirect, true
	case *waassdk.GetHttpRedirectResponse:
		return httpRedirectGetResponseBody(current)
	case waassdk.HttpRedirect:
		return current, true
	case *waassdk.HttpRedirect:
		return httpRedirectPointerBody(current)
	case waassdk.HttpRedirectSummary:
		return httpRedirectFromSummary(current), true
	case *waassdk.HttpRedirectSummary:
		if current == nil {
			return waassdk.HttpRedirect{}, false
		}
		return httpRedirectFromSummary(*current), true
	default:
		return waassdk.HttpRedirect{}, false
	}
}

func httpRedirectGetResponseBody(response *waassdk.GetHttpRedirectResponse) (waassdk.HttpRedirect, bool) {
	if response == nil {
		return waassdk.HttpRedirect{}, false
	}
	return response.HttpRedirect, true
}

func httpRedirectPointerBody(response *waassdk.HttpRedirect) (waassdk.HttpRedirect, bool) {
	if response == nil {
		return waassdk.HttpRedirect{}, false
	}
	return *response, true
}

func httpRedirectFromSummary(summary waassdk.HttpRedirectSummary) waassdk.HttpRedirect {
	return waassdk.HttpRedirect{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		Domain:         summary.Domain,
		Target:         summary.Target,
		ResponseCode:   summary.ResponseCode,
		TimeCreated:    summary.TimeCreated,
		LifecycleState: summary.LifecycleState,
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
	}
}

func listHttpRedirectsAllPages(
	call func(context.Context, waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error),
) func(context.Context, waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error) {
	return func(ctx context.Context, request waassdk.ListHttpRedirectsRequest) (waassdk.ListHttpRedirectsResponse, error) {
		var combined waassdk.ListHttpRedirectsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return waassdk.ListHttpRedirectsResponse{}, err
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

func getHttpRedirectWorkRequest(
	ctx context.Context,
	client httpRedirectWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize HttpRedirect OCI work request client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("HttpRedirect OCI work request client is not configured")
	}
	if strings.TrimSpace(workRequestID) == "" {
		return nil, fmt.Errorf("HttpRedirect work request id is empty")
	}
	response, err := client.GetWorkRequest(ctx, waassdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveHttpRedirectWorkRequestAction(workRequest any) (string, error) {
	current, err := httpRedirectWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveHttpRedirectWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := httpRedirectWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	return httpRedirectWorkRequestPhaseFromOperationType(current.OperationType)
}

func httpRedirectWorkRequestPhaseFromOperationType(operation waassdk.WorkRequestOperationTypesEnum) (shared.OSOKAsyncPhase, bool, error) {
	switch operation {
	case waassdk.WorkRequestOperationTypesCreateHttpRedirect:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case waassdk.WorkRequestOperationTypesUpdateHttpRedirect:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case waassdk.WorkRequestOperationTypesDeleteHttpRedirect:
		return shared.OSOKAsyncPhaseDelete, true, nil
	case "":
		return "", false, nil
	default:
		return "", false, fmt.Errorf("HttpRedirect work request operation %q is not modeled", operation)
	}
}

func recoverHttpRedirectIDFromWorkRequest(
	_ *waasv1beta1.HttpRedirect,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := httpRedirectWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if resolvedPhase, ok, err := httpRedirectWorkRequestPhaseFromOperationType(current.OperationType); err != nil {
		return "", err
	} else if ok && resolvedPhase != phase {
		return "", fmt.Errorf("HttpRedirect work request operation %q does not match phase %q", current.OperationType, phase)
	}
	for _, resource := range current.Resources {
		if !httpRedirectWorkRequestResourceIsRedirect(resource) {
			continue
		}
		if id := strings.TrimSpace(stringPtrValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	for _, resource := range current.Resources {
		if id := strings.TrimSpace(stringPtrValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("HttpRedirect work request %s did not expose an HttpRedirect identifier", stringPtrValue(current.Id))
}

func httpRedirectWorkRequestResourceIsRedirect(resource waassdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.NewReplacer(" ", "", "_", "", "-", "").Replace(stringPtrValue(resource.EntityType)))
	return strings.Contains(entityType, "httpredirect")
}

func httpRedirectWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := httpRedirectWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	workRequestID := strings.TrimSpace(stringPtrValue(current.Id))
	status := strings.TrimSpace(string(current.Status))
	if workRequestID == "" || status == "" {
		return ""
	}
	return fmt.Sprintf("HttpRedirect %s work request %s is %s", phase, workRequestID, status)
}

func httpRedirectWorkRequestFromAny(workRequest any) (waassdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case waassdk.WorkRequest:
		return current, nil
	case *waassdk.WorkRequest:
		if current == nil {
			return waassdk.WorkRequest{}, fmt.Errorf("HttpRedirect work request is nil")
		}
		return *current, nil
	default:
		return waassdk.WorkRequest{}, fmt.Errorf("expected HttpRedirect work request, got %T", workRequest)
	}
}

func handleHttpRedirectDeleteError(resource *waasv1beta1.HttpRedirect, err error) error {
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
	return httpRedirectAmbiguousNotFoundError{
		message:      "HttpRedirect delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapHttpRedirectDeleteConfirmation(hooks *HttpRedirectRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getHttpRedirect := hooks.Get.Call
	hooks.Get.Call = rejectHttpRedirectAuthShapedGet(getHttpRedirect)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate HttpRedirectServiceClient) HttpRedirectServiceClient {
		return httpRedirectDeleteConfirmationClient{
			delegate:        delegate,
			getHttpRedirect: getHttpRedirect,
		}
	})
}

func rejectHttpRedirectAuthShapedGet(
	call func(context.Context, waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error),
) func(context.Context, waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
	return func(ctx context.Context, request waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error) {
		response, err := call(ctx, request)
		if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return response, err
		}
		return response, httpRedirectAmbiguousNotFoundError{
			message:      "HttpRedirect read returned ambiguous 404 NotAuthorizedOrNotFound",
			opcRequestID: errorutil.OpcRequestID(err),
		}
	}
}

type httpRedirectDeleteConfirmationClient struct {
	delegate        HttpRedirectServiceClient
	getHttpRedirect func(context.Context, waassdk.GetHttpRedirectRequest) (waassdk.GetHttpRedirectResponse, error)
}

func (c httpRedirectDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *waasv1beta1.HttpRedirect,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c httpRedirectDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *waasv1beta1.HttpRedirect,
) (bool, error) {
	if httpRedirectPendingDeleteWorkRequestID(resource) != "" {
		deleted, err := c.delegate.Delete(ctx, resource)
		if httpRedirectShouldWaitForLiveDeleteReadback(resource, err) {
			markHttpRedirectTerminatingFromStatus(resource)
			return false, nil
		}
		return deleted, err
	}
	if err := c.rejectAuthShapedPreDeleteConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c httpRedirectDeleteConfirmationClient) rejectAuthShapedPreDeleteConfirmRead(
	ctx context.Context,
	resource *waasv1beta1.HttpRedirect,
) error {
	if c.getHttpRedirect == nil || resource == nil {
		return nil
	}
	redirectID := trackedHttpRedirectID(resource)
	if redirectID == "" {
		return nil
	}
	_, err := c.getHttpRedirect(ctx, waassdk.GetHttpRedirectRequest{HttpRedirectId: common.String(redirectID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("HttpRedirect delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedHttpRedirectID(resource *waasv1beta1.HttpRedirect) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func httpRedirectPendingDeleteWorkRequestID(resource *waasv1beta1.HttpRedirect) string {
	if resource == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func httpRedirectShouldWaitForLiveDeleteReadback(resource *waasv1beta1.HttpRedirect, err error) bool {
	if err == nil || resource == nil {
		return false
	}
	if !strings.Contains(err.Error(), "HttpRedirect delete confirmation returned unexpected lifecycle state") {
		return false
	}
	return httpRedirectRetainFinalizerState(resource.Status.LifecycleState)
}

func httpRedirectRetainFinalizerState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case string(waassdk.LifecycleStatesActive),
		string(waassdk.LifecycleStatesCreating),
		string(waassdk.LifecycleStatesUpdating):
		return true
	default:
		return false
	}
}

func applyHttpRedirectDeleteOutcome(
	resource *waasv1beta1.HttpRedirect,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := httpRedirectLifecycleState(response)
	if state == "" || state == string(waassdk.LifecycleStatesDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !httpRedirectDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markHttpRedirectTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func httpRedirectDeleteAlreadyPending(resource *waasv1beta1.HttpRedirect) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markHttpRedirectTerminating(resource *waasv1beta1.HttpRedirect, response any) {
	markHttpRedirectTerminatingWithRawState(resource, httpRedirectLifecycleState(response))
}

func markHttpRedirectTerminatingFromStatus(resource *waasv1beta1.HttpRedirect) {
	if resource == nil {
		return
	}
	markHttpRedirectTerminatingWithRawState(resource, resource.Status.LifecycleState)
}

func markHttpRedirectTerminatingWithRawState(resource *waasv1beta1.HttpRedirect, rawStatus string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = httpRedirectDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = httpRedirectTerminatingAsync(status.Async.Current, rawStatus, now)
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		httpRedirectDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func httpRedirectTerminatingAsync(
	current *shared.OSOKAsyncOperation,
	rawStatus string,
	now metav1.Time,
) *shared.OSOKAsyncOperation {
	if current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending {
		next := *current
		if current.Source != shared.OSOKAsyncSourceWorkRequest {
			next.RawStatus = rawStatus
		}
		next.Message = httpRedirectDeletePendingMessage
		next.UpdatedAt = &now
		return &next
	}
	return &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         httpRedirectDeletePendingMessage,
		UpdatedAt:       &now,
	}
}

func httpRedirectLifecycleState(response any) string {
	current, ok := httpRedirectBodyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func cloneHttpRedirectStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func httpRedirectDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
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

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intPtrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
