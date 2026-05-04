/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const listenerNetworkLoadBalancerIDAnnotation = "networkloadbalancer.oracle.com/network-load-balancer-id"

var listenerWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens:   []string{string(networkloadbalancersdk.OperationStatusAccepted), string(networkloadbalancersdk.OperationStatusInProgress), string(networkloadbalancersdk.OperationStatusCanceling)},
	SucceededStatusTokens: []string{string(networkloadbalancersdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(networkloadbalancersdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(networkloadbalancersdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(networkloadbalancersdk.OperationTypeCreateListener)},
	UpdateActionTokens:    []string{string(networkloadbalancersdk.OperationTypeUpdateListener)},
	DeleteActionTokens:    []string{string(networkloadbalancersdk.OperationTypeDeleteListener)},
}

type listenerRuntimeOCIClient interface {
	CreateListener(context.Context, networkloadbalancersdk.CreateListenerRequest) (networkloadbalancersdk.CreateListenerResponse, error)
	GetListener(context.Context, networkloadbalancersdk.GetListenerRequest) (networkloadbalancersdk.GetListenerResponse, error)
	ListListeners(context.Context, networkloadbalancersdk.ListListenersRequest) (networkloadbalancersdk.ListListenersResponse, error)
	UpdateListener(context.Context, networkloadbalancersdk.UpdateListenerRequest) (networkloadbalancersdk.UpdateListenerResponse, error)
	DeleteListener(context.Context, networkloadbalancersdk.DeleteListenerRequest) (networkloadbalancersdk.DeleteListenerResponse, error)
	GetWorkRequest(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

type listenerIdentity struct {
	networkLoadBalancerID string
	listenerName          string
}

type listenerRuntimeView struct {
	networkloadbalancersdk.Listener
	Ocid                  string `json:"ocid,omitempty"`
	NetworkLoadBalancerId string `json:"networkLoadBalancerId,omitempty"`
	LifecycleState        string `json:"lifecycleState,omitempty"`
}

type listenerListResult struct {
	Items []listenerRuntimeView `json:"items"`
}

func init() {
	registerListenerRuntimeHooksMutator(func(manager *ListenerServiceManager, hooks *ListenerRuntimeHooks) {
		client, initErr := newListenerSDKClient(manager)
		applyListenerRuntimeHooks(hooks, client, initErr)
	})
}

func applyListenerRuntimeHooks(
	hooks *ListenerRuntimeHooks,
	client listenerRuntimeOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = listenerRuntimeSemantics()
	applyListenerBodyHooks(hooks)
	applyListenerIdentityHooks(hooks)
	applyListenerReadHooks(hooks, client, initErr)
	applyListenerAsyncHooks(hooks, client, initErr)
	applyListenerDeleteHooks(hooks, client, initErr)
	applyListenerOperationFields(hooks)
}

func applyListenerBodyHooks(hooks *ListenerRuntimeHooks) {
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *networkloadbalancerv1beta1.Listener,
		_ string,
	) (any, error) {
		return buildListenerCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *networkloadbalancerv1beta1.Listener,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildListenerUpdateBody(resource, currentResponse)
	}
}

func applyListenerIdentityHooks(hooks *ListenerRuntimeHooks) {
	hooks.Identity = generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.Listener]{
		Resolve: func(resource *networkloadbalancerv1beta1.Listener) (any, error) {
			return resolveListenerIdentity(resource)
		},
		RecordPath: func(resource *networkloadbalancerv1beta1.Listener, identity any) {
			recordListenerPathIdentity(resource, identity.(listenerIdentity))
		},
		RecordTracked: func(resource *networkloadbalancerv1beta1.Listener, identity any, _ string) {
			recordListenerTrackedIdentity(resource, identity.(listenerIdentity))
		},
		LookupExisting: func(context.Context, *networkloadbalancerv1beta1.Listener, any) (any, error) {
			return nil, nil
		},
	}
}

func applyListenerReadHooks(
	hooks *ListenerRuntimeHooks,
	client listenerRuntimeOCIClient,
	initErr error,
) {
	hooks.Read = generatedruntime.ReadHooks{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &networkloadbalancersdk.GetListenerRequest{} },
			Fields:     listenerGetFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				if initErr != nil {
					return nil, initErr
				}
				if client == nil {
					return nil, errors.New("listener OCI client is nil")
				}
				getRequest := *request.(*networkloadbalancersdk.GetListenerRequest)
				response, err := client.GetListener(ctx, getRequest)
				if err != nil {
					return nil, listenerFatalAuthShapedNotFoundError(err)
				}
				return listenerRuntimeViewFromListener(response.Listener, stringValue(getRequest.NetworkLoadBalancerId)), nil
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &networkloadbalancersdk.ListListenersRequest{} },
			Fields:     listenerListFields(),
			Call: func(ctx context.Context, request any) (any, error) {
				if initErr != nil {
					return nil, initErr
				}
				if client == nil {
					return nil, errors.New("listener OCI client is nil")
				}
				result, err := listListenerRuntimeViews(ctx, client, *request.(*networkloadbalancersdk.ListListenersRequest))
				if err != nil {
					return nil, listenerFatalAuthShapedNotFoundError(err)
				}
				return result, nil
			},
		},
	}
}

func applyListenerAsyncHooks(
	hooks *ListenerRuntimeHooks,
	client listenerRuntimeOCIClient,
	initErr error,
) {
	hooks.Async = generatedruntime.AsyncHooks[*networkloadbalancerv1beta1.Listener]{
		Adapter: listenerWorkRequestAsyncAdapter,
		GetWorkRequest: func(ctx context.Context, workRequestID string) (any, error) {
			if initErr != nil {
				return nil, initErr
			}
			if client == nil {
				return nil, errors.New("listener OCI client is nil")
			}
			response, err := client.GetWorkRequest(ctx, networkloadbalancersdk.GetWorkRequestRequest{
				WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
			})
			if err != nil {
				return nil, err
			}
			return response.WorkRequest, nil
		},
		ResolveAction: func(workRequest any) (string, error) {
			return listenerWorkRequestOperationType(workRequest), nil
		},
		RecoverResourceID: func(resource *networkloadbalancerv1beta1.Listener, _ any, _ shared.OSOKAsyncPhase) (string, error) {
			return recoverListenerNetworkLoadBalancerID(resource), nil
		},
	}
}

func applyListenerDeleteHooks(
	hooks *ListenerRuntimeHooks,
	client listenerRuntimeOCIClient,
	initErr error,
) {
	hooks.DeleteHooks.HandleError = handleListenerDeleteError
	hooks.DeleteHooks.ConfirmRead = func(
		ctx context.Context,
		resource *networkloadbalancerv1beta1.Listener,
		currentID string,
	) (any, error) {
		return confirmListenerDeleteRead(ctx, client, initErr, resource, currentID)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ListenerServiceClient) ListenerServiceClient {
		return listenerDeleteGuardClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	})
}

func applyListenerOperationFields(hooks *ListenerRuntimeHooks) {
	hooks.Create.Fields = listenerCreateFields()
	hooks.Get.Fields = listenerGetFields()
	hooks.List.Fields = listenerListFields()
	hooks.Update.Fields = listenerUpdateFields()
	hooks.Delete.Fields = listenerDeleteFields()
}

func newListenerSDKClient(manager *ListenerServiceManager) (listenerRuntimeOCIClient, error) {
	if manager == nil {
		return nil, errors.New("listener service manager is nil")
	}
	client, err := networkloadbalancersdk.NewNetworkLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize Listener OCI client: %w", err)
	}
	return client, nil
}

func newListenerRuntimeHooksWithOCIClient(client listenerRuntimeOCIClient) ListenerRuntimeHooks {
	return ListenerRuntimeHooks{
		Semantics:       listenerRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.Listener]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*networkloadbalancerv1beta1.Listener]{},
		StatusHooks:     generatedruntime.StatusHooks[*networkloadbalancerv1beta1.Listener]{},
		ParityHooks:     generatedruntime.ParityHooks[*networkloadbalancerv1beta1.Listener]{},
		Async: generatedruntime.AsyncHooks[*networkloadbalancerv1beta1.Listener]{
			Adapter: listenerWorkRequestAsyncAdapter,
		},
		DeleteHooks: generatedruntime.DeleteHooks[*networkloadbalancerv1beta1.Listener]{},
		Create: runtimeOperationHooks[networkloadbalancersdk.CreateListenerRequest, networkloadbalancersdk.CreateListenerResponse]{
			Fields: listenerCreateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.CreateListenerRequest) (networkloadbalancersdk.CreateListenerResponse, error) {
				return client.CreateListener(ctx, request)
			},
		},
		Get: runtimeOperationHooks[networkloadbalancersdk.GetListenerRequest, networkloadbalancersdk.GetListenerResponse]{
			Fields: listenerGetFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.GetListenerRequest) (networkloadbalancersdk.GetListenerResponse, error) {
				return client.GetListener(ctx, request)
			},
		},
		List: runtimeOperationHooks[networkloadbalancersdk.ListListenersRequest, networkloadbalancersdk.ListListenersResponse]{
			Fields: listenerListFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.ListListenersRequest) (networkloadbalancersdk.ListListenersResponse, error) {
				return client.ListListeners(ctx, request)
			},
		},
		Update: runtimeOperationHooks[networkloadbalancersdk.UpdateListenerRequest, networkloadbalancersdk.UpdateListenerResponse]{
			Fields: listenerUpdateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.UpdateListenerRequest) (networkloadbalancersdk.UpdateListenerResponse, error) {
				return client.UpdateListener(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[networkloadbalancersdk.DeleteListenerRequest, networkloadbalancersdk.DeleteListenerResponse]{
			Fields: listenerDeleteFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.DeleteListenerRequest) (networkloadbalancersdk.DeleteListenerResponse, error) {
				return client.DeleteListener(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ListenerServiceClient) ListenerServiceClient{},
	}
}

type listenerDeleteGuardClient struct {
	delegate ListenerServiceClient
	client   listenerRuntimeOCIClient
	initErr  error
}

func (c listenerDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Listener,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c listenerDeleteGuardClient) Delete(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Listener,
) (bool, error) {
	if handled, deleted, err := c.handlePendingWriteWorkRequestForDelete(ctx, resource); handled {
		return deleted, err
	}
	if handled, deleted, err := c.rejectAuthShapedPreDeleteRead(ctx, resource); handled {
		return deleted, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c listenerDeleteGuardClient) handlePendingWriteWorkRequestForDelete(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Listener,
) (bool, bool, error) {
	workRequestID, phase := currentListenerWorkRequest(resource)
	switch {
	case workRequestID == "":
		return false, false, nil
	case phase == shared.OSOKAsyncPhaseDelete:
		return false, false, nil
	case phase != shared.OSOKAsyncPhaseCreate && phase != shared.OSOKAsyncPhaseUpdate:
		err := fmt.Errorf("listener delete cannot resume %s work request %s from delete path", phase, workRequestID)
		markListenerFailed(resource, err)
		return true, false, err
	}

	workRequest, err := c.fetchWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markListenerFailed(resource, err)
		return true, false, err
	}
	currentAsync, err := buildListenerWorkRequestOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		markListenerFailed(resource, err)
		return true, false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("Listener %s work request %s is still in progress; waiting before delete", phase, workRequestID)
		applyListenerWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
		return true, false, nil
	case shared.OSOKAsyncClassSucceeded:
		servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
		return false, false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("listener %s work request %s finished with status %s", phase, workRequestID, currentAsync.RawStatus)
		applyListenerWorkRequestOperation(resource, currentAsync)
		return true, false, err
	default:
		err := fmt.Errorf("listener %s work request %s projected unsupported async class %s", phase, workRequestID, currentAsync.NormalizedClass)
		markListenerFailed(resource, err)
		return true, false, err
	}
}

func (c listenerDeleteGuardClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.Listener,
) (bool, bool, error) {
	identity, err := resolveListenerIdentity(resource)
	if err != nil {
		return false, false, nil
	}
	if c.initErr != nil || c.client == nil {
		return false, false, nil
	}

	_, err = c.client.GetListener(ctx, networkloadbalancersdk.GetListenerRequest{
		NetworkLoadBalancerId: common.String(identity.networkLoadBalancerID),
		ListenerName:          common.String(identity.listenerName),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return false, false, nil
	}

	fatalErr := listenerFatalAuthShapedNotFoundError(err)
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, fatalErr)
	markListenerFailed(resource, fatalErr)
	return true, false, fatalErr
}

func (c listenerDeleteGuardClient) fetchWorkRequest(
	ctx context.Context,
	workRequestID string,
) (networkloadbalancersdk.WorkRequest, error) {
	if c.initErr != nil {
		return networkloadbalancersdk.WorkRequest{}, c.initErr
	}
	if c.client == nil {
		return networkloadbalancersdk.WorkRequest{}, errors.New("listener OCI client is nil")
	}
	response, err := c.client.GetWorkRequest(ctx, networkloadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return networkloadbalancersdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func confirmListenerDeleteRead(
	ctx context.Context,
	client listenerRuntimeOCIClient,
	initErr error,
	resource *networkloadbalancerv1beta1.Listener,
	currentID string,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, errors.New("listener OCI client is nil")
	}
	identity, err := resolveListenerIdentity(resource)
	if err != nil {
		return nil, err
	}
	networkLoadBalancerID := firstNonEmptyTrim(currentID, identity.networkLoadBalancerID)
	response, err := client.GetListener(ctx, networkloadbalancersdk.GetListenerRequest{
		NetworkLoadBalancerId: common.String(networkLoadBalancerID),
		ListenerName:          common.String(identity.listenerName),
	})
	if err != nil {
		return nil, err
	}
	return listenerRuntimeViewFromListener(response.Listener, networkLoadBalancerID), nil
}

type listenerAuthShapedNotFoundError struct {
	message      string
	opcRequestID string
}

func (e listenerAuthShapedNotFoundError) Error() string {
	return e.message
}

func (e listenerAuthShapedNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func handleListenerDeleteError(resource *networkloadbalancerv1beta1.Listener, err error) error {
	if err == nil {
		return nil
	}
	fatalErr := listenerFatalAuthShapedNotFoundError(err)
	if _, ok := fatalErr.(listenerAuthShapedNotFoundError); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, fatalErr)
		}
		markListenerFailed(resource, fatalErr)
	}
	return fatalErr
}

func listenerFatalAuthShapedNotFoundError(err error) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return listenerAuthShapedNotFoundError{
		message:      fmt.Sprintf("Listener received ambiguous OCI 404 NotAuthorizedOrNotFound during delete; retaining finalizer: %v", err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func currentListenerWorkRequest(resource *networkloadbalancerv1beta1.Listener) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func buildListenerWorkRequestOperation(
	status *shared.OSOKStatus,
	workRequest networkloadbalancersdk.WorkRequest,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	return servicemanager.BuildWorkRequestAsyncOperation(status, listenerWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        string(workRequest.OperationType),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    stringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func applyListenerWorkRequestOperation(
	resource *networkloadbalancerv1beta1.Listener,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	if current.WorkRequestID == "" && resource.Status.OsokStatus.Async.Current != nil {
		current.WorkRequestID = resource.Status.OsokStatus.Async.Current.WorkRequestID
	}
	now := metav1.Now()
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: 0,
	}
}

func applyListenerWorkRequestOperationAs(
	resource *networkloadbalancerv1beta1.Listener,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	if current == nil {
		return applyListenerWorkRequestOperation(resource, current)
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	return applyListenerWorkRequestOperation(resource, &next)
}

func markListenerFailed(resource *networkloadbalancerv1beta1.Listener, err error) {
	if resource == nil || err == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
}

func listenerRuntimeSemantics() *generatedruntime.Semantics {
	workRequestPendingStates := []string{
		string(networkloadbalancersdk.OperationStatusAccepted),
		string(networkloadbalancersdk.OperationStatusInProgress),
		string(networkloadbalancersdk.OperationStatusCanceling),
	}
	return &generatedruntime.Semantics{
		FormalService: "networkloadbalancer",
		FormalSlug:    "listener",
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
			ProvisioningStates: workRequestPendingStates,
			UpdatingStates:     workRequestPendingStates,
			ActiveStates:       []string{string(networkloadbalancersdk.OperationStatusSucceeded), "ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  append([]string{"DELETING"}, workRequestPendingStates...),
			TerminalStates: []string{"DELETED", string(networkloadbalancersdk.OperationStatusSucceeded)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"defaultBackendSetName",
				"ipVersion",
				"isPpv2Enabled",
				"l3IpIdleTimeout",
				"port",
				"protocol",
				"tcpIdleTimeout",
				"udpIdleTimeout",
			},
			ForceNew:      []string{"name", "networkLoadBalancerId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func listenerCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerNetworkLoadBalancerIDField(),
		{
			FieldName:    "CreateListenerDetails",
			RequestName:  "CreateListenerDetails",
			Contribution: "body",
		},
	}
}

func listenerGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerNetworkLoadBalancerIDField(),
		listenerNameField(),
	}
}

func listenerListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerNetworkLoadBalancerIDField(),
	}
}

func listenerUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerNetworkLoadBalancerIDField(),
		listenerNameField(),
		{
			FieldName:    "UpdateListenerDetails",
			RequestName:  "UpdateListenerDetails",
			Contribution: "body",
		},
	}
}

func listenerDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		listenerNetworkLoadBalancerIDField(),
		listenerNameField(),
	}
}

func listenerNetworkLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "NetworkLoadBalancerId",
		RequestName:      "networkLoadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.status.ocid"},
	}
}

func listenerNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "ListenerName",
		RequestName:  "listenerName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func buildListenerCreateBody(resource *networkloadbalancerv1beta1.Listener) (networkloadbalancersdk.CreateListenerDetails, error) {
	if resource == nil {
		return networkloadbalancersdk.CreateListenerDetails{}, fmt.Errorf("listener resource is nil")
	}
	return networkloadbalancersdk.CreateListenerDetails{
		Name:                  stringPointer(firstNonEmptyTrim(resource.Spec.Name, resource.Name)),
		DefaultBackendSetName: stringPointer(resource.Spec.DefaultBackendSetName),
		Port:                  common.Int(resource.Spec.Port),
		Protocol:              networkloadbalancersdk.ListenerProtocolsEnum(resource.Spec.Protocol),
		IpVersion:             networkloadbalancersdk.IpVersionEnum(strings.TrimSpace(resource.Spec.IpVersion)),
		IsPpv2Enabled:         common.Bool(resource.Spec.IsPpv2Enabled),
		TcpIdleTimeout:        intPointer(resource.Spec.TcpIdleTimeout),
		UdpIdleTimeout:        intPointer(resource.Spec.UdpIdleTimeout),
		L3IpIdleTimeout:       intPointer(resource.Spec.L3IpIdleTimeout),
	}, nil
}

func buildListenerUpdateBody(
	resource *networkloadbalancerv1beta1.Listener,
	currentResponse any,
) (networkloadbalancersdk.UpdateListenerDetails, bool, error) {
	if resource == nil {
		return networkloadbalancersdk.UpdateListenerDetails{}, false, fmt.Errorf("listener resource is nil")
	}

	current, err := listenerCurrentValue(currentResponse)
	if err != nil {
		return networkloadbalancersdk.UpdateListenerDetails{}, false, err
	}
	if !listenerNeedsUpdate(resource, current) {
		return networkloadbalancersdk.UpdateListenerDetails{}, false, nil
	}
	return networkloadbalancersdk.UpdateListenerDetails{
		DefaultBackendSetName: stringPointer(resource.Spec.DefaultBackendSetName),
		Port:                  common.Int(resource.Spec.Port),
		Protocol:              networkloadbalancersdk.ListenerProtocolsEnum(resource.Spec.Protocol),
		IpVersion:             networkloadbalancersdk.IpVersionEnum(strings.TrimSpace(resource.Spec.IpVersion)),
		IsPpv2Enabled:         common.Bool(resource.Spec.IsPpv2Enabled),
		TcpIdleTimeout:        intPointer(resource.Spec.TcpIdleTimeout),
		UdpIdleTimeout:        intPointer(resource.Spec.UdpIdleTimeout),
		L3IpIdleTimeout:       intPointer(resource.Spec.L3IpIdleTimeout),
	}, true, nil
}

func listenerNeedsUpdate(resource *networkloadbalancerv1beta1.Listener, current networkloadbalancersdk.Listener) bool {
	return listenerRequiredFieldsNeedUpdate(resource, current) ||
		listenerOptionalFieldsNeedUpdate(resource, current)
}

func listenerRequiredFieldsNeedUpdate(resource *networkloadbalancerv1beta1.Listener, current networkloadbalancersdk.Listener) bool {
	if strings.TrimSpace(resource.Spec.DefaultBackendSetName) != stringValue(current.DefaultBackendSetName) {
		return true
	}
	if resource.Spec.Port != intValue(current.Port) {
		return true
	}
	if strings.TrimSpace(resource.Spec.Protocol) != strings.TrimSpace(string(current.Protocol)) {
		return true
	}
	if resource.Spec.IsPpv2Enabled != boolValue(current.IsPpv2Enabled) {
		return true
	}
	return false
}

func listenerOptionalFieldsNeedUpdate(resource *networkloadbalancerv1beta1.Listener, current networkloadbalancersdk.Listener) bool {
	if strings.TrimSpace(resource.Spec.IpVersion) != "" &&
		strings.TrimSpace(resource.Spec.IpVersion) != strings.TrimSpace(string(current.IpVersion)) {
		return true
	}
	if resource.Spec.TcpIdleTimeout != 0 && resource.Spec.TcpIdleTimeout != intValue(current.TcpIdleTimeout) {
		return true
	}
	if resource.Spec.UdpIdleTimeout != 0 && resource.Spec.UdpIdleTimeout != intValue(current.UdpIdleTimeout) {
		return true
	}
	return resource.Spec.L3IpIdleTimeout != 0 && resource.Spec.L3IpIdleTimeout != intValue(current.L3IpIdleTimeout)
}

func listenerCurrentValue(currentResponse any) (networkloadbalancersdk.Listener, error) {
	if currentResponse == nil || reflect.ValueOf(currentResponse).Kind() == reflect.Pointer && reflect.ValueOf(currentResponse).IsNil() {
		return networkloadbalancersdk.Listener{}, fmt.Errorf("current Listener response is nil")
	}
	payload, err := json.Marshal(currentResponse)
	if err != nil {
		return networkloadbalancersdk.Listener{}, fmt.Errorf("marshal current Listener response: %w", err)
	}
	var decoded networkloadbalancersdk.Listener
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return networkloadbalancersdk.Listener{}, fmt.Errorf("decode current Listener response: %w", err)
	}
	return decoded, nil
}

func resolveListenerIdentity(resource *networkloadbalancerv1beta1.Listener) (listenerIdentity, error) {
	if resource == nil {
		return listenerIdentity{}, fmt.Errorf("resolve Listener identity: resource is nil")
	}

	statusNetworkLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationNetworkLoadBalancerID := strings.TrimSpace(resource.Annotations[listenerNetworkLoadBalancerIDAnnotation])
	if statusNetworkLoadBalancerID != "" && annotationNetworkLoadBalancerID != "" && statusNetworkLoadBalancerID != annotationNetworkLoadBalancerID {
		return listenerIdentity{}, fmt.Errorf(
			"resolve Listener identity: %s changed from recorded networkLoadBalancerId %q to %q",
			listenerNetworkLoadBalancerIDAnnotation,
			statusNetworkLoadBalancerID,
			annotationNetworkLoadBalancerID,
		)
	}

	statusListenerName := strings.TrimSpace(resource.Status.Name)
	specListenerName := firstNonEmptyTrim(resource.Spec.Name, resource.Name)
	if statusListenerName != "" && specListenerName != "" && statusListenerName != specListenerName {
		return listenerIdentity{}, fmt.Errorf(
			"resolve Listener identity: formal semantics require replacement when name changes from %q to %q",
			statusListenerName,
			specListenerName,
		)
	}

	identity := listenerIdentity{
		networkLoadBalancerID: firstNonEmptyTrim(statusNetworkLoadBalancerID, annotationNetworkLoadBalancerID),
		listenerName:          firstNonEmptyTrim(statusListenerName, specListenerName),
	}
	if identity.networkLoadBalancerID == "" {
		return listenerIdentity{}, fmt.Errorf("resolve Listener identity: %s annotation is required", listenerNetworkLoadBalancerIDAnnotation)
	}
	if identity.listenerName == "" {
		return listenerIdentity{}, fmt.Errorf("resolve Listener identity: listener name is empty")
	}
	return identity, nil
}

func recordListenerPathIdentity(resource *networkloadbalancerv1beta1.Listener, identity listenerIdentity) {
	if resource == nil {
		return
	}
	resource.Status.Name = identity.listenerName
	// Listener has no child OCID in the Network Load Balancer API, so the
	// parent networkLoadBalancerId is the stable path identity for requests.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.networkLoadBalancerID)
}

func recordListenerTrackedIdentity(resource *networkloadbalancerv1beta1.Listener, identity listenerIdentity) {
	recordListenerPathIdentity(resource, identity)
}

func recoverListenerNetworkLoadBalancerID(resource *networkloadbalancerv1beta1.Listener) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyTrim(
		string(resource.Status.OsokStatus.Ocid),
		resource.Annotations[listenerNetworkLoadBalancerIDAnnotation],
	)
}

func listListenerRuntimeViews(
	ctx context.Context,
	client listenerRuntimeOCIClient,
	request networkloadbalancersdk.ListListenersRequest,
) (listenerListResult, error) {
	parentID := stringValue(request.NetworkLoadBalancerId)
	var items []listenerRuntimeView
	for {
		response, err := client.ListListeners(ctx, request)
		if err != nil {
			return listenerListResult{}, err
		}
		for _, summary := range response.Items {
			items = append(items, listenerRuntimeViewFromSummary(summary, parentID))
		}
		if stringValue(response.OpcNextPage) == "" {
			return listenerListResult{Items: items}, nil
		}
		request.Page = response.OpcNextPage
	}
}

func listenerRuntimeViewFromListener(listener networkloadbalancersdk.Listener, networkLoadBalancerID string) listenerRuntimeView {
	return listenerRuntimeView{
		Listener:              listener,
		Ocid:                  strings.TrimSpace(networkLoadBalancerID),
		NetworkLoadBalancerId: strings.TrimSpace(networkLoadBalancerID),
		LifecycleState:        "ACTIVE",
	}
}

func listenerRuntimeViewFromSummary(summary networkloadbalancersdk.ListenerSummary, networkLoadBalancerID string) listenerRuntimeView {
	return listenerRuntimeView{
		Listener:              listenerFromSummary(summary),
		NetworkLoadBalancerId: strings.TrimSpace(networkLoadBalancerID),
		LifecycleState:        "ACTIVE",
	}
}

func listenerFromSummary(summary networkloadbalancersdk.ListenerSummary) networkloadbalancersdk.Listener {
	return networkloadbalancersdk.Listener(summary)
}

func listenerWorkRequestOperationType(workRequest any) string {
	switch wr := workRequest.(type) {
	case networkloadbalancersdk.WorkRequest:
		return strings.TrimSpace(string(wr.OperationType))
	case *networkloadbalancersdk.WorkRequest:
		if wr == nil {
			return ""
		}
		return strings.TrimSpace(string(wr.OperationType))
	default:
		return ""
	}
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringPointer(value string) *string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return common.String(trimmed)
	}
	return nil
}

func intPointer(value int) *int {
	if value == 0 {
		return nil
	}
	return common.Int(value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}
