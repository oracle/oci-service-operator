/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package bastion

import (
	"context"
	"fmt"
	"strings"

	bastionsdk "github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/common"
	bastionv1beta1 "github.com/oracle/oci-service-operator/api/bastion/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

var bastionWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(bastionsdk.OperationStatusAccepted),
		string(bastionsdk.OperationStatusInProgress),
		string(bastionsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(bastionsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(bastionsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(bastionsdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(bastionsdk.OperationTypeCreateBastion),
		string(bastionsdk.ActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(bastionsdk.OperationTypeUpdateBastion),
		string(bastionsdk.ActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(bastionsdk.OperationTypeDeleteBastion),
		string(bastionsdk.ActionTypeDeleted),
	},
}

type bastionOCIClient interface {
	CreateBastion(context.Context, bastionsdk.CreateBastionRequest) (bastionsdk.CreateBastionResponse, error)
	GetBastion(context.Context, bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error)
	ListBastions(context.Context, bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error)
	UpdateBastion(context.Context, bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error)
	DeleteBastion(context.Context, bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error)
	GetWorkRequest(context.Context, bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error)
}

type bastionWorkRequestClient interface {
	GetWorkRequest(context.Context, bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error)
}

type bastionListCall func(context.Context, bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error)

type bastionAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e bastionAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e bastionAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerBastionRuntimeHooksMutator(func(manager *BastionServiceManager, hooks *BastionRuntimeHooks) {
		client, initErr := newBastionOCIClient(manager)
		applyBastionRuntimeHooks(hooks, client, initErr)
	})
}

func newBastionOCIClient(manager *BastionServiceManager) (bastionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("bastion service manager is nil")
	}
	client, err := bastionsdk.NewBastionClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyBastionRuntimeHooks(
	hooks *BastionRuntimeHooks,
	client bastionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newBastionRuntimeSemantics()
	hooks.Async.Adapter = bastionWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getBastionWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveBastionGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveBastionGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverBastionIDFromGeneratedWorkRequest
	hooks.Async.Message = bastionGeneratedWorkRequestMessage
	hooks.DeleteHooks.HandleError = handleBastionDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapBastionDeleteWorkRequestConfirmation(client, initErr))
	if hooks.List.Call != nil {
		hooks.List.Call = listBastionsAllPages(hooks.List.Call)
	}
}

func newBastionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "bastion",
		FormalSlug:    "bastion",
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
			ProvisioningStates: []string{string(bastionsdk.BastionLifecycleStateCreating)},
			UpdatingStates:     []string{string(bastionsdk.BastionLifecycleStateUpdating)},
			ActiveStates:       []string{string(bastionsdk.BastionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(bastionsdk.BastionLifecycleStateDeleting)},
			TerminalStates: []string{string(bastionsdk.BastionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "targetSubnetId", "bastionType"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"maxSessionTtlInSeconds",
				"staticJumpHostIpAddresses",
				"clientCidrBlockAllowList",
				"freeformTags",
				"definedTags",
				"securityAttributes",
			},
			ForceNew: []string{
				"bastionType",
				"compartmentId",
				"targetSubnetId",
				"name",
				"phoneBookEntry",
				"dnsProxyStatus",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func newBastionServiceClientWithOCIClient(log loggerutil.OSOKLogger, client bastionOCIClient) BastionServiceClient {
	manager := &BastionServiceManager{Log: log}
	hooks := newBastionRuntimeHooksWithOCIClient(client)
	applyBastionRuntimeHooks(&hooks, client, nil)
	delegate := defaultBastionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*bastionv1beta1.Bastion](
			buildBastionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapBastionGeneratedClient(hooks, delegate)
}

func newBastionRuntimeHooksWithOCIClient(client bastionOCIClient) BastionRuntimeHooks {
	return BastionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*bastionv1beta1.Bastion]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*bastionv1beta1.Bastion]{},
		StatusHooks:     generatedruntime.StatusHooks[*bastionv1beta1.Bastion]{},
		ParityHooks:     generatedruntime.ParityHooks[*bastionv1beta1.Bastion]{},
		Async:           generatedruntime.AsyncHooks[*bastionv1beta1.Bastion]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*bastionv1beta1.Bastion]{},
		Create: runtimeOperationHooks[bastionsdk.CreateBastionRequest, bastionsdk.CreateBastionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateBastionDetails", RequestName: "CreateBastionDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request bastionsdk.CreateBastionRequest) (bastionsdk.CreateBastionResponse, error) {
				if client == nil {
					return bastionsdk.CreateBastionResponse{}, fmt.Errorf("bastion OCI client is nil")
				}
				return client.CreateBastion(ctx, request)
			},
		},
		Get: runtimeOperationHooks[bastionsdk.GetBastionRequest, bastionsdk.GetBastionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "BastionId", RequestName: "bastionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
				if client == nil {
					return bastionsdk.GetBastionResponse{}, fmt.Errorf("bastion OCI client is nil")
				}
				return client.GetBastion(ctx, request)
			},
		},
		List: runtimeOperationHooks[bastionsdk.ListBastionsRequest, bastionsdk.ListBastionsResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "BastionLifecycleState", RequestName: "bastionLifecycleState", Contribution: "query"},
				{FieldName: "BastionId", RequestName: "bastionId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
			},
			Call: func(ctx context.Context, request bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error) {
				if client == nil {
					return bastionsdk.ListBastionsResponse{}, fmt.Errorf("bastion OCI client is nil")
				}
				return client.ListBastions(ctx, request)
			},
		},
		Update: runtimeOperationHooks[bastionsdk.UpdateBastionRequest, bastionsdk.UpdateBastionResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "BastionId", RequestName: "bastionId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateBastionDetails", RequestName: "UpdateBastionDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error) {
				if client == nil {
					return bastionsdk.UpdateBastionResponse{}, fmt.Errorf("bastion OCI client is nil")
				}
				return client.UpdateBastion(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[bastionsdk.DeleteBastionRequest, bastionsdk.DeleteBastionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "BastionId", RequestName: "bastionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error) {
				if client == nil {
					return bastionsdk.DeleteBastionResponse{}, fmt.Errorf("bastion OCI client is nil")
				}
				return client.DeleteBastion(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BastionServiceClient) BastionServiceClient{},
	}
}

func getBastionWorkRequest(
	ctx context.Context,
	client bastionWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize bastion OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("bastion OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, bastionsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveBastionGeneratedWorkRequestAction(workRequest any) (string, error) {
	bastionWorkRequest, err := bastionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(bastionWorkRequest.OperationType), nil
}

func resolveBastionGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	bastionWorkRequest, err := bastionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := bastionWorkRequestPhaseFromOperationType(bastionWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverBastionIDFromGeneratedWorkRequest(
	_ *bastionv1beta1.Bastion,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	bastionWorkRequest, err := bastionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveBastionIDFromWorkRequest(bastionWorkRequest, bastionWorkRequestActionForPhase(phase))
}

func bastionWorkRequestFromAny(workRequest any) (bastionsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case bastionsdk.WorkRequest:
		return current, nil
	case *bastionsdk.WorkRequest:
		if current == nil {
			return bastionsdk.WorkRequest{}, fmt.Errorf("bastion work request is nil")
		}
		return *current, nil
	default:
		return bastionsdk.WorkRequest{}, fmt.Errorf("unexpected bastion work request type %T", workRequest)
	}
}

func bastionWorkRequestPhaseFromOperationType(operationType bastionsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case bastionsdk.OperationTypeCreateBastion:
		return shared.OSOKAsyncPhaseCreate, true
	case bastionsdk.OperationTypeUpdateBastion:
		return shared.OSOKAsyncPhaseUpdate, true
	case bastionsdk.OperationTypeDeleteBastion:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func bastionWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) bastionsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return bastionsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return bastionsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return bastionsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveBastionIDFromWorkRequest(workRequest bastionsdk.WorkRequest, action bastionsdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveBastionIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveBastionIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("bastion work request %s does not expose a bastion identifier", stringValue(workRequest.Id))
}

func resolveBastionIDFromResources(
	resources []bastionsdk.WorkRequestResource,
	action bastionsdk.ActionTypeEnum,
	preferBastionOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferBastionOnly && !isBastionWorkRequestResource(resource) {
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

func bastionGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	bastionWorkRequest, err := bastionWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Bastion %s work request %s is %s", phase, stringValue(bastionWorkRequest.Id), bastionWorkRequest.Status)
}

func isBastionWorkRequestResource(resource bastionsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "bastion", "bastions":
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/bastions/")
}

func listBastionsAllPages(call bastionListCall) bastionListCall {
	return func(ctx context.Context, request bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error) {
		if call == nil {
			return bastionsdk.ListBastionsResponse{}, fmt.Errorf("bastion list operation is not configured")
		}

		accumulator := newBastionListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return bastionsdk.ListBastionsResponse{}, err
			}
			accumulator.append(response)

			nextPage := stringValue(response.OpcNextPage)
			if nextPage == "" {
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return bastionsdk.ListBastionsResponse{}, err
			}
		}
	}
}

type bastionListAccumulator struct {
	response  bastionsdk.ListBastionsResponse
	seenPages map[string]struct{}
}

func newBastionListAccumulator() bastionListAccumulator {
	return bastionListAccumulator{seenPages: map[string]struct{}{}}
}

func (a *bastionListAccumulator) append(response bastionsdk.ListBastionsResponse) {
	if a.response.RawResponse == nil {
		a.response.RawResponse = response.RawResponse
	}
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.OpcNextPage = response.OpcNextPage
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *bastionListAccumulator) advance(request *bastionsdk.ListBastionsRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("bastion list pagination repeated page token %q", nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return nil
}

func wrapBastionDeleteWorkRequestConfirmation(
	client bastionOCIClient,
	initErr error,
) func(BastionServiceClient) BastionServiceClient {
	return func(delegate BastionServiceClient) BastionServiceClient {
		return bastionDeleteWorkRequestConfirmationClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	}
}

type bastionDeleteWorkRequestConfirmationClient struct {
	delegate BastionServiceClient
	client   bastionOCIClient
	initErr  error
}

func (c bastionDeleteWorkRequestConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *bastionv1beta1.Bastion,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c bastionDeleteWorkRequestConfirmationClient) Delete(
	ctx context.Context,
	resource *bastionv1beta1.Bastion,
) (bool, error) {
	if err := c.rejectAuthShapedCompletedDelete(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c bastionDeleteWorkRequestConfirmationClient) rejectAuthShapedCompletedDelete(
	ctx context.Context,
	resource *bastionv1beta1.Bastion,
) error {
	if c.initErr != nil || c.client == nil || resource == nil {
		return nil
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return nil
	}
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if workRequestID == "" {
		return nil
	}
	workRequest, err := c.client.GetWorkRequest(ctx, bastionsdk.GetWorkRequestRequest{WorkRequestId: common.String(workRequestID)})
	if err != nil || !bastionDeleteWorkRequestSucceeded(workRequest.WorkRequest) {
		return nil
	}
	return c.rejectAuthShapedConfirmRead(ctx, resource)
}

func (c bastionDeleteWorkRequestConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *bastionv1beta1.Bastion,
) error {
	bastionID := trackedBastionID(resource)
	if bastionID == "" {
		return nil
	}
	_, err := c.client.GetBastion(ctx, bastionsdk.GetBastionRequest{BastionId: common.String(bastionID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("bastion delete work request completed but confirmation returned authorization-shaped not found; refusing to confirm deletion: %w", err)
}

func bastionDeleteWorkRequestSucceeded(workRequest bastionsdk.WorkRequest) bool {
	if workRequest.OperationType != bastionsdk.OperationTypeDeleteBastion {
		return false
	}
	class, err := bastionWorkRequestAsyncAdapter.Normalize(string(workRequest.Status))
	return err == nil && class == shared.OSOKAsyncClassSucceeded
}

func trackedBastionID(resource *bastionv1beta1.Bastion) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func handleBastionDeleteError(resource *bastionv1beta1.Bastion, err error) error {
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
	return bastionAmbiguousNotFoundError{
		message:      "bastion delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
