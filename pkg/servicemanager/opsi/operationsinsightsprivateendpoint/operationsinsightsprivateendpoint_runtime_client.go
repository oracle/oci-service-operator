/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operationsinsightsprivateendpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
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

const operationsInsightsPrivateEndpointKind = "OperationsInsightsPrivateEndpoint"

var operationsInsightsPrivateEndpointWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(opsisdk.OperationTypeCreatePrivateEndpoint),
		string(opsisdk.ActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(opsisdk.OperationTypeUpdatePrivateEndpoint),
		string(opsisdk.ActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(opsisdk.OperationTypeDeletePrivateEndpoint),
		string(opsisdk.ActionTypeDeleted),
	},
}

type operationsInsightsPrivateEndpointOCIClient interface {
	CreateOperationsInsightsPrivateEndpoint(context.Context, opsisdk.CreateOperationsInsightsPrivateEndpointRequest) (opsisdk.CreateOperationsInsightsPrivateEndpointResponse, error)
	GetOperationsInsightsPrivateEndpoint(context.Context, opsisdk.GetOperationsInsightsPrivateEndpointRequest) (opsisdk.GetOperationsInsightsPrivateEndpointResponse, error)
	ListOperationsInsightsPrivateEndpoints(context.Context, opsisdk.ListOperationsInsightsPrivateEndpointsRequest) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error)
	UpdateOperationsInsightsPrivateEndpoint(context.Context, opsisdk.UpdateOperationsInsightsPrivateEndpointRequest) (opsisdk.UpdateOperationsInsightsPrivateEndpointResponse, error)
	DeleteOperationsInsightsPrivateEndpoint(context.Context, opsisdk.DeleteOperationsInsightsPrivateEndpointRequest) (opsisdk.DeleteOperationsInsightsPrivateEndpointResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type operationsInsightsPrivateEndpointWorkRequestClient interface {
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type operationsInsightsPrivateEndpointRuntimeClient struct {
	delegate           OperationsInsightsPrivateEndpointServiceClient
	hooks              OperationsInsightsPrivateEndpointRuntimeHooks
	workRequestClient  operationsInsightsPrivateEndpointWorkRequestClient
	workRequestInitErr error
	log                loggerutil.OSOKLogger
}

type operationsInsightsPrivateEndpointAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e operationsInsightsPrivateEndpointAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e operationsInsightsPrivateEndpointAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var _ OperationsInsightsPrivateEndpointServiceClient = (*operationsInsightsPrivateEndpointRuntimeClient)(nil)

func init() {
	registerOperationsInsightsPrivateEndpointRuntimeHooksMutator(func(manager *OperationsInsightsPrivateEndpointServiceManager, hooks *OperationsInsightsPrivateEndpointRuntimeHooks) {
		workRequestClient, initErr := newOperationsInsightsPrivateEndpointWorkRequestClient(manager)
		applyOperationsInsightsPrivateEndpointRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newOperationsInsightsPrivateEndpointWorkRequestClient(
	manager *OperationsInsightsPrivateEndpointServiceManager,
) (operationsInsightsPrivateEndpointWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", operationsInsightsPrivateEndpointKind)
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOperationsInsightsPrivateEndpointRuntimeHooks(
	manager *OperationsInsightsPrivateEndpointServiceManager,
	hooks *OperationsInsightsPrivateEndpointRuntimeHooks,
	workRequestClient operationsInsightsPrivateEndpointWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = operationsInsightsPrivateEndpointRuntimeSemantics()
	hooks.BuildCreateBody = buildOperationsInsightsPrivateEndpointCreateBody
	hooks.BuildUpdateBody = buildOperationsInsightsPrivateEndpointUpdateBody
	if hooks.Get.Call != nil {
		hooks.Get.Call = getOperationsInsightsPrivateEndpointWithDefaults(hooks.Get.Call)
	}
	if hooks.List.Call != nil {
		hooks.List.Call = listOperationsInsightsPrivateEndpointsAllPages(hooks.List.Call)
	}
	hooks.List.Fields = operationsInsightsPrivateEndpointListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedOperationsInsightsPrivateEndpointIdentity
	hooks.StatusHooks.ProjectStatus = projectOperationsInsightsPrivateEndpointStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateOperationsInsightsPrivateEndpointCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleOperationsInsightsPrivateEndpointDeleteError
	hooks.Async.Adapter = operationsInsightsPrivateEndpointWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOperationsInsightsPrivateEndpointWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveOperationsInsightsPrivateEndpointWorkRequestAction
	hooks.Async.RecoverResourceID = recoverOperationsInsightsPrivateEndpointIDFromWorkRequest
	hooks.Async.Message = operationsInsightsPrivateEndpointWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OperationsInsightsPrivateEndpointServiceClient) OperationsInsightsPrivateEndpointServiceClient {
		runtimeClient := &operationsInsightsPrivateEndpointRuntimeClient{
			delegate:           delegate,
			hooks:              *hooks,
			workRequestClient:  workRequestClient,
			workRequestInitErr: initErr,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func operationsInsightsPrivateEndpointRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "operationsinsightsprivateendpoint",
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
			ProvisioningStates: []string{string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateUpdating)},
			ActiveStates: []string{
				string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive),
				string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateNeedsAttention),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"id",
				"displayName",
				"compartmentId",
				"vcnId",
				"subnetId",
				"isUsedForRacDbs",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{"displayName", "description", "nsgIds", "freeformTags", "definedTags"},
			ForceNew: []string{
				"compartmentId",
				"vcnId",
				"subnetId",
				"isUsedForRacDbs",
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetOperationsInsightsPrivateEndpoint"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetOperationsInsightsPrivateEndpoint"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client operationsInsightsPrivateEndpointOCIClient,
) OperationsInsightsPrivateEndpointServiceClient {
	manager := &OperationsInsightsPrivateEndpointServiceManager{Log: log}
	hooks := newOperationsInsightsPrivateEndpointRuntimeHooksWithOCIClient(client)
	applyOperationsInsightsPrivateEndpointRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultOperationsInsightsPrivateEndpointServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.OperationsInsightsPrivateEndpoint](
			buildOperationsInsightsPrivateEndpointGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapOperationsInsightsPrivateEndpointGeneratedClient(hooks, delegate)
}

func newOperationsInsightsPrivateEndpointRuntimeHooksWithOCIClient(
	client operationsInsightsPrivateEndpointOCIClient,
) OperationsInsightsPrivateEndpointRuntimeHooks {
	return OperationsInsightsPrivateEndpointRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{},
		Create: runtimeOperationHooks[opsisdk.CreateOperationsInsightsPrivateEndpointRequest, opsisdk.CreateOperationsInsightsPrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOperationsInsightsPrivateEndpointDetails", RequestName: "CreateOperationsInsightsPrivateEndpointDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request opsisdk.CreateOperationsInsightsPrivateEndpointRequest) (opsisdk.CreateOperationsInsightsPrivateEndpointResponse, error) {
				return client.CreateOperationsInsightsPrivateEndpoint(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetOperationsInsightsPrivateEndpointRequest, opsisdk.GetOperationsInsightsPrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsPrivateEndpointId", RequestName: "operationsInsightsPrivateEndpointId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetOperationsInsightsPrivateEndpointRequest) (opsisdk.GetOperationsInsightsPrivateEndpointResponse, error) {
				return client.GetOperationsInsightsPrivateEndpoint(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListOperationsInsightsPrivateEndpointsRequest, opsisdk.ListOperationsInsightsPrivateEndpointsResponse]{
			Fields: operationsInsightsPrivateEndpointListFields(),
			Call: func(ctx context.Context, request opsisdk.ListOperationsInsightsPrivateEndpointsRequest) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error) {
				return client.ListOperationsInsightsPrivateEndpoints(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateOperationsInsightsPrivateEndpointRequest, opsisdk.UpdateOperationsInsightsPrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsPrivateEndpointId", RequestName: "operationsInsightsPrivateEndpointId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateOperationsInsightsPrivateEndpointDetails", RequestName: "UpdateOperationsInsightsPrivateEndpointDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request opsisdk.UpdateOperationsInsightsPrivateEndpointRequest) (opsisdk.UpdateOperationsInsightsPrivateEndpointResponse, error) {
				return client.UpdateOperationsInsightsPrivateEndpoint(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteOperationsInsightsPrivateEndpointRequest, opsisdk.DeleteOperationsInsightsPrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsPrivateEndpointId", RequestName: "operationsInsightsPrivateEndpointId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteOperationsInsightsPrivateEndpointRequest) (opsisdk.DeleteOperationsInsightsPrivateEndpointResponse, error) {
				return client.DeleteOperationsInsightsPrivateEndpoint(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OperationsInsightsPrivateEndpointServiceClient) OperationsInsightsPrivateEndpointServiceClient{},
	}
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", operationsInsightsPrivateEndpointKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", operationsInsightsPrivateEndpointKind)
	}
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", operationsInsightsPrivateEndpointKind)
	}

	current := resource.Status.OsokStatus.Async.Current
	if current != nil &&
		current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.WorkRequestID != "" {
		switch current.Phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			return c.waitForPendingWriteBeforeDelete(ctx, resource, current)
		case shared.OSOKAsyncPhaseDelete:
			return c.waitForDeleteWorkRequestBeforeFinalizerRelease(ctx, resource, current)
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) waitForPendingWriteBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
) (bool, error) {
	workRequest, err := getOperationsInsightsPrivateEndpointWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, tracked.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := operationsInsightsPrivateEndpointAsyncOperation(&resource.Status.OsokStatus, workRequest, tracked.Phase)
	if err != nil {
		return false, err
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequest(resource, current)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.deleteAfterSucceededWrite(ctx, resource, tracked, current, workRequest)
	default:
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s %s work request %s finished with status %s", operationsInsightsPrivateEndpointKind, current.Phase, tracked.WorkRequestID, current.RawStatus)
	}
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) deleteAfterSucceededWrite(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
	current *shared.OSOKAsyncOperation,
	workRequest opsisdk.WorkRequest,
) (bool, error) {
	resourceID := currentOperationsInsightsPrivateEndpointID(resource)
	if resourceID == "" {
		resourceID = operationsInsightsPrivateEndpointIDFromWorkRequest(workRequest, tracked.Phase)
	}
	if resourceID == "" {
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s %s work request %s did not expose a %s identifier", operationsInsightsPrivateEndpointKind, tracked.Phase, tracked.WorkRequestID, operationsInsightsPrivateEndpointKind)
	}
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	return c.deleteAfterWriteReadback(ctx, resource, tracked, current, resourceID)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) waitForDeleteWorkRequestBeforeFinalizerRelease(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
) (bool, error) {
	workRequest, err := getOperationsInsightsPrivateEndpointWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, tracked.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := operationsInsightsPrivateEndpointAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequest(resource, current)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.completeDeleteWorkRequest(ctx, resource, tracked, current, workRequest)
	default:
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s delete work request %s finished with status %s", operationsInsightsPrivateEndpointKind, tracked.WorkRequestID, current.RawStatus)
	}
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) completeDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
	current *shared.OSOKAsyncOperation,
	workRequest opsisdk.WorkRequest,
) (bool, error) {
	resourceID := currentOperationsInsightsPrivateEndpointID(resource)
	if resourceID == "" {
		resourceID = operationsInsightsPrivateEndpointIDFromWorkRequest(workRequest, shared.OSOKAsyncPhaseDelete)
	}
	if resourceID == "" {
		c.markDeleted(resource, fmt.Sprintf("OCI %s delete work request completed", operationsInsightsPrivateEndpointKind))
		return true, nil
	}
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)

	live, found, err := c.getOperationsInsightsPrivateEndpointForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if err := projectOperationsInsightsPrivateEndpointSDKStatus(resource, live); err != nil {
		return false, err
	}
	if operationsInsightsPrivateEndpointDeleteLifecycleTerminal(live) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if operationsInsightsPrivateEndpointDeleteLifecyclePending(live) {
		c.markDeleteWorkRequestPending(resource, current, tracked.WorkRequestID, resourceID)
		return false, nil
	}
	c.applyWorkRequest(resource, current)
	return false, fmt.Errorf("%s delete work request %s succeeded but %s %s is in lifecycle state %q", operationsInsightsPrivateEndpointKind, tracked.WorkRequestID, operationsInsightsPrivateEndpointKind, resourceID, live.LifecycleState)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) deleteAfterWriteReadback(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
	current *shared.OSOKAsyncOperation,
	resourceID string,
) (bool, error) {
	live, found, err := c.getOperationsInsightsPrivateEndpointForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if !found || operationsInsightsPrivateEndpointWriteLifecyclePending(live) {
		if found {
			_ = projectOperationsInsightsPrivateEndpointSDKStatus(resource, live)
		}
		c.markWriteReadbackPending(resource, current, tracked.WorkRequestID, resourceID)
		return false, nil
	}
	_ = projectOperationsInsightsPrivateEndpointSDKStatus(resource, live)
	resource.Status.OsokStatus.Async.Current = nil
	return c.delegate.Delete(ctx, resource)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) getOperationsInsightsPrivateEndpointForDelete(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	resourceID string,
) (opsisdk.OperationsInsightsPrivateEndpoint, bool, error) {
	if strings.TrimSpace(resourceID) == "" || c.hooks.Get.Call == nil {
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false, nil
	}
	response, err := c.hooks.Get.Call(ctx, opsisdk.GetOperationsInsightsPrivateEndpointRequest{
		OperationsInsightsPrivateEndpointId: common.String(strings.TrimSpace(resourceID)),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if operationsInsightsPrivateEndpointIsUnambiguousNotFound(err) {
			return opsisdk.OperationsInsightsPrivateEndpoint{}, false, nil
		}
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false, handleOperationsInsightsPrivateEndpointDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response.OperationsInsightsPrivateEndpoint, true, nil
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) applyWorkRequest(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	current *shared.OSOKAsyncOperation,
) {
	if current == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) markWriteReadbackPending(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) {
	if current == nil {
		return
	}
	now := metav1.Now()
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = fmt.Sprintf(
		"%s %s work request %s succeeded; waiting for %s %s to become deleteable",
		operationsInsightsPrivateEndpointKind,
		current.Phase,
		strings.TrimSpace(workRequestID),
		operationsInsightsPrivateEndpointKind,
		strings.TrimSpace(resourceID),
	)
	next.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) markDeleteWorkRequestPending(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) {
	if current == nil {
		return
	}
	now := metav1.Now()
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = fmt.Sprintf(
		"%s delete work request %s succeeded; waiting for %s %s deletion to be confirmed",
		operationsInsightsPrivateEndpointKind,
		strings.TrimSpace(workRequestID),
		operationsInsightsPrivateEndpointKind,
		strings.TrimSpace(resourceID),
	)
	next.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func (c *operationsInsightsPrivateEndpointRuntimeClient) markDeleted(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Async.Current = nil
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func operationsInsightsPrivateEndpointListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"spec.compartmentId", "status.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"spec.displayName", "status.displayName", "displayName"}},
		{FieldName: "OpsiPrivateEndpointId", RequestName: "opsiPrivateEndpointId", Contribution: "query", PreferResourceID: true, LookupPaths: []string{"status.id", "status.status.ocid", "id"}},
		{FieldName: "IsUsedForRacDbs", RequestName: "isUsedForRacDbs", Contribution: "query", LookupPaths: []string{"spec.isUsedForRacDbs", "status.isUsedForRacDbs", "isUsedForRacDbs"}},
		{FieldName: "VcnId", RequestName: "vcnId", Contribution: "query", LookupPaths: []string{"spec.vcnId", "status.vcnId", "vcnId"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
	}
}

func buildOperationsInsightsPrivateEndpointCreateBody(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", operationsInsightsPrivateEndpointKind)
	}
	if err := validateOperationsInsightsPrivateEndpointSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := opsisdk.CreateOperationsInsightsPrivateEndpointDetails{
		DisplayName:     common.String(strings.TrimSpace(spec.DisplayName)),
		CompartmentId:   common.String(strings.TrimSpace(spec.CompartmentId)),
		VcnId:           common.String(strings.TrimSpace(spec.VcnId)),
		SubnetId:        common.String(strings.TrimSpace(spec.SubnetId)),
		IsUsedForRacDbs: common.Bool(spec.IsUsedForRacDbs),
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if spec.NsgIds != nil {
		body.NsgIds = cloneOperationsInsightsPrivateEndpointStringSlice(spec.NsgIds)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = operationsInsightsPrivateEndpointDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildOperationsInsightsPrivateEndpointUpdateBody(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", operationsInsightsPrivateEndpointKind)
	}
	if err := validateOperationsInsightsPrivateEndpointSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := operationsInsightsPrivateEndpointFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a %s body", operationsInsightsPrivateEndpointKind, operationsInsightsPrivateEndpointKind)
	}
	if err := validateOperationsInsightsPrivateEndpointCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}

	body, updateNeeded := operationsInsightsPrivateEndpointMutableUpdateBody(resource.Spec, current)
	return body, updateNeeded, nil
}

func operationsInsightsPrivateEndpointMutableUpdateBody(
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
	current opsisdk.OperationsInsightsPrivateEndpoint,
) (opsisdk.UpdateOperationsInsightsPrivateEndpointDetails, bool) {
	body := opsisdk.UpdateOperationsInsightsPrivateEndpointDetails{}
	updateNeeded := false
	updateNeeded = setOperationsInsightsPrivateEndpointDisplayName(&body, current, spec) || updateNeeded
	updateNeeded = setOperationsInsightsPrivateEndpointDescription(&body, current, spec) || updateNeeded
	updateNeeded = setOperationsInsightsPrivateEndpointNsgIds(&body, current, spec) || updateNeeded
	updateNeeded = setOperationsInsightsPrivateEndpointFreeformTags(&body, current, spec) || updateNeeded
	updateNeeded = setOperationsInsightsPrivateEndpointDefinedTags(&body, current, spec) || updateNeeded
	return body, updateNeeded
}

func setOperationsInsightsPrivateEndpointDisplayName(
	body *opsisdk.UpdateOperationsInsightsPrivateEndpointDetails,
	current opsisdk.OperationsInsightsPrivateEndpoint,
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
) bool {
	if operationsInsightsPrivateEndpointStringPtrEqual(current.DisplayName, spec.DisplayName) {
		return false
	}
	body.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	return true
}

func setOperationsInsightsPrivateEndpointDescription(
	body *opsisdk.UpdateOperationsInsightsPrivateEndpointDetails,
	current opsisdk.OperationsInsightsPrivateEndpoint,
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
) bool {
	if operationsInsightsPrivateEndpointStringPtrEqual(current.Description, spec.Description) {
		return false
	}
	body.Description = common.String(spec.Description)
	return true
}

func setOperationsInsightsPrivateEndpointNsgIds(
	body *opsisdk.UpdateOperationsInsightsPrivateEndpointDetails,
	current opsisdk.OperationsInsightsPrivateEndpoint,
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
) bool {
	if spec.NsgIds == nil || operationsInsightsPrivateEndpointStringSlicesEqual(spec.NsgIds, current.NsgIds) {
		return false
	}
	body.NsgIds = cloneOperationsInsightsPrivateEndpointStringSlice(spec.NsgIds)
	return true
}

func setOperationsInsightsPrivateEndpointFreeformTags(
	body *opsisdk.UpdateOperationsInsightsPrivateEndpointDetails,
	current opsisdk.OperationsInsightsPrivateEndpoint,
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	body.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func setOperationsInsightsPrivateEndpointDefinedTags(
	body *opsisdk.UpdateOperationsInsightsPrivateEndpointDetails,
	current opsisdk.OperationsInsightsPrivateEndpoint,
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := operationsInsightsPrivateEndpointDefinedTagsFromSpec(spec.DefinedTags)
	if operationsInsightsPrivateEndpointJSONEqual(desired, current.DefinedTags) {
		return false
	}
	body.DefinedTags = desired
	return true
}

func validateOperationsInsightsPrivateEndpointSpec(spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec) error {
	var missing []string
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.VcnId) == "" {
		missing = append(missing, "vcnId")
	}
	if strings.TrimSpace(spec.SubnetId) == "" {
		missing = append(missing, "subnetId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", operationsInsightsPrivateEndpointKind, strings.Join(missing, ", "))
}

func validateOperationsInsightsPrivateEndpointCreateOnlyDriftForResponse(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", operationsInsightsPrivateEndpointKind)
	}
	current, ok := operationsInsightsPrivateEndpointFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateOperationsInsightsPrivateEndpointCreateOnlyDrift(resource.Spec, current)
}

func validateOperationsInsightsPrivateEndpointCreateOnlyDrift(
	spec opsiv1beta1.OperationsInsightsPrivateEndpointSpec,
	current opsisdk.OperationsInsightsPrivateEndpoint,
) error {
	var drift []string
	if !operationsInsightsPrivateEndpointStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !operationsInsightsPrivateEndpointStringPtrEqual(current.VcnId, spec.VcnId) {
		drift = append(drift, "vcnId")
	}
	if !operationsInsightsPrivateEndpointStringPtrEqual(current.SubnetId, spec.SubnetId) {
		drift = append(drift, "subnetId")
	}
	if !operationsInsightsPrivateEndpointRacFlagMatches(current.IsUsedForRacDbs, spec.IsUsedForRacDbs) {
		drift = append(drift, "isUsedForRacDbs")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only drift detected for %s; replace the resource or restore the desired spec before update", operationsInsightsPrivateEndpointKind, strings.Join(drift, ", "))
}

func operationsInsightsPrivateEndpointRacFlagMatches(current *bool, desired bool) bool {
	if current == nil {
		return !desired
	}
	return *current == desired
}

func getOperationsInsightsPrivateEndpointWithDefaults(
	call func(context.Context, opsisdk.GetOperationsInsightsPrivateEndpointRequest) (opsisdk.GetOperationsInsightsPrivateEndpointResponse, error),
) func(context.Context, opsisdk.GetOperationsInsightsPrivateEndpointRequest) (opsisdk.GetOperationsInsightsPrivateEndpointResponse, error) {
	return func(ctx context.Context, request opsisdk.GetOperationsInsightsPrivateEndpointRequest) (opsisdk.GetOperationsInsightsPrivateEndpointResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		normalizeOperationsInsightsPrivateEndpointDefaults(&response.OperationsInsightsPrivateEndpoint)
		return response, nil
	}
}

func listOperationsInsightsPrivateEndpointsAllPages(
	call func(context.Context, opsisdk.ListOperationsInsightsPrivateEndpointsRequest) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error),
) func(context.Context, opsisdk.ListOperationsInsightsPrivateEndpointsRequest) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error) {
	return func(ctx context.Context, request opsisdk.ListOperationsInsightsPrivateEndpointsRequest) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error) {
		if call == nil {
			return opsisdk.ListOperationsInsightsPrivateEndpointsResponse{}, fmt.Errorf("%s list operation is not configured", operationsInsightsPrivateEndpointKind)
		}
		return collectOperationsInsightsPrivateEndpointListPages(ctx, call, request)
	}
}

func collectOperationsInsightsPrivateEndpointListPages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListOperationsInsightsPrivateEndpointsRequest) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error),
	request opsisdk.ListOperationsInsightsPrivateEndpointsRequest,
) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error) {
	seenPages := map[string]struct{}{}
	var combined opsisdk.ListOperationsInsightsPrivateEndpointsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return opsisdk.ListOperationsInsightsPrivateEndpointsResponse{}, err
		}
		normalizeOperationsInsightsPrivateEndpointListDefaults(&response)
		appendOperationsInsightsPrivateEndpointListPage(&combined, response)

		nextPage, ok, err := nextOperationsInsightsPrivateEndpointListPage(response, seenPages)
		if err != nil || !ok {
			return combined, err
		}
		request.Page = common.String(nextPage)
	}
}

func appendOperationsInsightsPrivateEndpointListPage(
	combined *opsisdk.ListOperationsInsightsPrivateEndpointsResponse,
	response opsisdk.ListOperationsInsightsPrivateEndpointsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
	combined.OpcNextPage = response.OpcNextPage
}

func nextOperationsInsightsPrivateEndpointListPage(
	response opsisdk.ListOperationsInsightsPrivateEndpointsResponse,
	seenPages map[string]struct{},
) (string, bool, error) {
	nextPage := strings.TrimSpace(operationsInsightsPrivateEndpointStringValue(response.OpcNextPage))
	if nextPage == "" {
		return "", false, nil
	}
	if _, exists := seenPages[nextPage]; exists {
		return "", false, fmt.Errorf("%s list pagination repeated page token %q", operationsInsightsPrivateEndpointKind, nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nextPage, true, nil
}

func normalizeOperationsInsightsPrivateEndpointListDefaults(response *opsisdk.ListOperationsInsightsPrivateEndpointsResponse) {
	if response == nil {
		return
	}
	for i := range response.Items {
		normalizeOperationsInsightsPrivateEndpointSummaryDefaults(&response.Items[i])
	}
}

func projectOperationsInsightsPrivateEndpointStatus(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", operationsInsightsPrivateEndpointKind)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current, ok := operationsInsightsPrivateEndpointFromResponse(response)
	if !ok {
		return nil
	}
	return projectOperationsInsightsPrivateEndpointSDKStatus(resource, current)
}

func projectOperationsInsightsPrivateEndpointSDKStatus(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	current opsisdk.OperationsInsightsPrivateEndpoint,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", operationsInsightsPrivateEndpointKind)
	}

	status := &resource.Status
	status.Id = operationsInsightsPrivateEndpointStringValue(current.Id)
	status.DisplayName = operationsInsightsPrivateEndpointStringValue(current.DisplayName)
	status.CompartmentId = operationsInsightsPrivateEndpointStringValue(current.CompartmentId)
	status.VcnId = operationsInsightsPrivateEndpointStringValue(current.VcnId)
	status.SubnetId = operationsInsightsPrivateEndpointStringValue(current.SubnetId)
	status.LifecycleState = string(current.LifecycleState)
	status.PrivateIp = operationsInsightsPrivateEndpointStringValue(current.PrivateIp)
	status.Description = operationsInsightsPrivateEndpointStringValue(current.Description)
	status.TimeCreated = operationsInsightsPrivateEndpointTimeString(current.TimeCreated)
	status.LifecycleDetails = operationsInsightsPrivateEndpointStringValue(current.LifecycleDetails)
	status.PrivateEndpointStatusDetails = operationsInsightsPrivateEndpointStringValue(current.PrivateEndpointStatusDetails)
	status.IsUsedForRacDbs = operationsInsightsPrivateEndpointBoolValue(current.IsUsedForRacDbs)
	status.NsgIds = cloneOperationsInsightsPrivateEndpointStringSlice(current.NsgIds)
	status.FreeformTags = maps.Clone(current.FreeformTags)
	status.DefinedTags = operationsInsightsPrivateEndpointStatusTagsFromSDK(current.DefinedTags)
	status.SystemTags = operationsInsightsPrivateEndpointStatusTagsFromSDK(current.SystemTags)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
	return nil
}

func operationsInsightsPrivateEndpointFromResponse(response any) (opsisdk.OperationsInsightsPrivateEndpoint, bool) {
	response = operationsInsightsPrivateEndpointDereference(response)
	switch current := response.(type) {
	case nil:
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false
	case opsisdk.OperationsInsightsPrivateEndpoint:
		return current, true
	case opsisdk.OperationsInsightsPrivateEndpointSummary:
		return operationsInsightsPrivateEndpointFromSummary(current), true
	case opsisdk.CreateOperationsInsightsPrivateEndpointResponse:
		return current.OperationsInsightsPrivateEndpoint, true
	case opsisdk.GetOperationsInsightsPrivateEndpointResponse:
		return current.OperationsInsightsPrivateEndpoint, true
	case map[string]any:
		return operationsInsightsPrivateEndpointFromStatusMap(current)
	default:
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false
	}
}

func operationsInsightsPrivateEndpointDereference(response any) any {
	value := reflect.ValueOf(response)
	if !value.IsValid() || value.Kind() != reflect.Pointer {
		return response
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}

func operationsInsightsPrivateEndpointFromStatusMap(values map[string]any) (opsisdk.OperationsInsightsPrivateEndpoint, bool) {
	if len(values) == 0 {
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false
	}
	var status opsiv1beta1.OperationsInsightsPrivateEndpointStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return opsisdk.OperationsInsightsPrivateEndpoint{}, false
	}
	return operationsInsightsPrivateEndpointFromStatus(status), true
}

func operationsInsightsPrivateEndpointFromSummary(
	summary opsisdk.OperationsInsightsPrivateEndpointSummary,
) opsisdk.OperationsInsightsPrivateEndpoint {
	current := opsisdk.OperationsInsightsPrivateEndpoint{
		Id:                           summary.Id,
		DisplayName:                  summary.DisplayName,
		CompartmentId:                summary.CompartmentId,
		VcnId:                        summary.VcnId,
		SubnetId:                     summary.SubnetId,
		LifecycleState:               summary.LifecycleState,
		Description:                  summary.Description,
		TimeCreated:                  summary.TimeCreated,
		LifecycleDetails:             summary.LifecycleDetails,
		PrivateEndpointStatusDetails: summary.PrivateEndpointStatusDetails,
		IsUsedForRacDbs:              summary.IsUsedForRacDbs,
		FreeformTags:                 summary.FreeformTags,
		DefinedTags:                  summary.DefinedTags,
		SystemTags:                   summary.SystemTags,
	}
	normalizeOperationsInsightsPrivateEndpointDefaults(&current)
	return current
}

func operationsInsightsPrivateEndpointFromStatus(
	status opsiv1beta1.OperationsInsightsPrivateEndpointStatus,
) opsisdk.OperationsInsightsPrivateEndpoint {
	return opsisdk.OperationsInsightsPrivateEndpoint{
		Id:                           common.String(status.Id),
		DisplayName:                  common.String(status.DisplayName),
		CompartmentId:                common.String(status.CompartmentId),
		VcnId:                        common.String(status.VcnId),
		SubnetId:                     common.String(status.SubnetId),
		LifecycleState:               opsisdk.OperationsInsightsPrivateEndpointLifecycleStateEnum(status.LifecycleState),
		PrivateIp:                    common.String(status.PrivateIp),
		Description:                  common.String(status.Description),
		LifecycleDetails:             common.String(status.LifecycleDetails),
		PrivateEndpointStatusDetails: common.String(status.PrivateEndpointStatusDetails),
		IsUsedForRacDbs:              common.Bool(status.IsUsedForRacDbs),
		NsgIds:                       cloneOperationsInsightsPrivateEndpointStringSlice(status.NsgIds),
		FreeformTags:                 maps.Clone(status.FreeformTags),
		DefinedTags:                  operationsInsightsPrivateEndpointDefinedTagsFromStatus(status.DefinedTags),
		SystemTags:                   operationsInsightsPrivateEndpointDefinedTagsFromStatus(status.SystemTags),
	}
}

func clearTrackedOperationsInsightsPrivateEndpointIdentity(resource *opsiv1beta1.OperationsInsightsPrivateEndpoint) {
	if resource == nil {
		return
	}
	status := resource.Status.OsokStatus
	status.Ocid = ""
	resource.Status = opsiv1beta1.OperationsInsightsPrivateEndpointStatus{OsokStatus: status}
}

func currentOperationsInsightsPrivateEndpointID(resource *opsiv1beta1.OperationsInsightsPrivateEndpoint) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func getOperationsInsightsPrivateEndpointWorkRequest(
	ctx context.Context,
	client operationsInsightsPrivateEndpointWorkRequestClient,
	initErr error,
	workRequestID string,
) (opsisdk.WorkRequest, error) {
	if initErr != nil {
		return opsisdk.WorkRequest{}, fmt.Errorf("initialize %s OCI client: %w", operationsInsightsPrivateEndpointKind, initErr)
	}
	if client == nil {
		return opsisdk.WorkRequest{}, fmt.Errorf("%s work request client is not configured", operationsInsightsPrivateEndpointKind)
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return opsisdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func operationsInsightsPrivateEndpointAsyncOperation(
	status *shared.OSOKStatus,
	workRequest opsisdk.WorkRequest,
	fallback shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	rawAction, err := resolveOperationsInsightsPrivateEndpointWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, operationsInsightsPrivateEndpointWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        rawAction,
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    operationsInsightsPrivateEndpointStringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallback,
	})
}

func resolveOperationsInsightsPrivateEndpointWorkRequestAction(workRequest any) (string, error) {
	current, ok := operationsInsightsPrivateEndpointWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", operationsInsightsPrivateEndpointKind, workRequest)
	}
	if current.OperationType != "" {
		return string(current.OperationType), nil
	}

	var action string
	for _, resource := range current.Resources {
		if !isOperationsInsightsPrivateEndpointWorkRequestResource(resource) || operationsInsightsPrivateEndpointIgnorableAction(resource.ActionType) {
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
			return "", fmt.Errorf("%s work request %s exposes conflicting action types %q and %q", operationsInsightsPrivateEndpointKind, operationsInsightsPrivateEndpointStringValue(current.Id), action, candidate)
		}
	}
	return action, nil
}

func recoverOperationsInsightsPrivateEndpointIDFromWorkRequest(
	_ *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, ok := operationsInsightsPrivateEndpointWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", operationsInsightsPrivateEndpointKind, workRequest)
	}
	resourceID := operationsInsightsPrivateEndpointIDFromWorkRequest(current, phase)
	if resourceID == "" {
		return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", operationsInsightsPrivateEndpointKind, phase, operationsInsightsPrivateEndpointStringValue(current.Id), operationsInsightsPrivateEndpointKind)
	}
	return resourceID, nil
}

func operationsInsightsPrivateEndpointIDFromWorkRequest(
	workRequest opsisdk.WorkRequest,
	phase shared.OSOKAsyncPhase,
) string {
	action := operationsInsightsPrivateEndpointActionForPhase(phase)
	if id, ok := operationsInsightsPrivateEndpointIDFromWorkRequestResources(workRequest.Resources, action, true); ok {
		return id
	}
	id, _ := operationsInsightsPrivateEndpointIDFromWorkRequestResources(workRequest.Resources, "", false)
	return id
}

func operationsInsightsPrivateEndpointIDFromWorkRequestResources(
	resources []opsisdk.WorkRequestResource,
	action opsisdk.ActionTypeEnum,
	requireUnique bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		id, ok := operationsInsightsPrivateEndpointIDFromWorkRequestResource(resource, action)
		if !ok {
			continue
		}
		if !requireUnique {
			return id, true
		}
		if candidate != "" && candidate != id {
			return "", false
		}
		candidate = id
	}
	return candidate, candidate != ""
}

func operationsInsightsPrivateEndpointIDFromWorkRequestResource(
	resource opsisdk.WorkRequestResource,
	action opsisdk.ActionTypeEnum,
) (string, bool) {
	if !isOperationsInsightsPrivateEndpointWorkRequestResource(resource) {
		return "", false
	}
	if !operationsInsightsPrivateEndpointWorkRequestActionMatches(resource.ActionType, action) {
		return "", false
	}
	id := strings.TrimSpace(operationsInsightsPrivateEndpointStringValue(resource.Identifier))
	return id, id != ""
}

func operationsInsightsPrivateEndpointWorkRequestActionMatches(
	resourceAction opsisdk.ActionTypeEnum,
	action opsisdk.ActionTypeEnum,
) bool {
	if action == "" {
		return !operationsInsightsPrivateEndpointIgnorableAction(resourceAction)
	}
	return resourceAction == action || resourceAction == opsisdk.ActionTypeInProgress
}

func operationsInsightsPrivateEndpointActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
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

func operationsInsightsPrivateEndpointWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, bool) {
	workRequest = operationsInsightsPrivateEndpointDereference(workRequest)
	current, ok := workRequest.(opsisdk.WorkRequest)
	return current, ok
}

func isOperationsInsightsPrivateEndpointWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := normalizeOperationsInsightsPrivateEndpointEntityType(operationsInsightsPrivateEndpointStringValue(resource.EntityType))
	switch entityType {
	case "operationsinsightsprivateendpoint", "opsiprivateendpoint", "privateendpoint":
		return true
	default:
		return strings.Contains(entityType, "privateendpoint") && strings.Contains(entityType, "ops")
	}
}

func normalizeOperationsInsightsPrivateEndpointEntityType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", ".", "", "/", "")
	return replacer.Replace(value)
}

func operationsInsightsPrivateEndpointIgnorableAction(action opsisdk.ActionTypeEnum) bool {
	return action == "" || action == opsisdk.ActionTypeRelated
}

func operationsInsightsPrivateEndpointWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, ok := operationsInsightsPrivateEndpointWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	workRequestID := operationsInsightsPrivateEndpointStringValue(current.Id)
	rawStatus := strings.TrimSpace(string(current.Status))
	if phase == "" || workRequestID == "" || rawStatus == "" {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", operationsInsightsPrivateEndpointKind, phase, workRequestID, rawStatus)
}

func handleOperationsInsightsPrivateEndpointDeleteError(
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
	err error,
) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return operationsInsightsPrivateEndpointAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s delete path returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed", operationsInsightsPrivateEndpointKind),
		opcRequestID: requestID,
	}
}

func operationsInsightsPrivateEndpointIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func operationsInsightsPrivateEndpointWriteLifecyclePending(current opsisdk.OperationsInsightsPrivateEndpoint) bool {
	state := strings.ToUpper(string(current.LifecycleState))
	return state == string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateCreating) ||
		state == string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateUpdating)
}

func operationsInsightsPrivateEndpointDeleteLifecyclePending(current opsisdk.OperationsInsightsPrivateEndpoint) bool {
	state := strings.ToUpper(string(current.LifecycleState))
	return state == "" || state == string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateDeleting)
}

func operationsInsightsPrivateEndpointDeleteLifecycleTerminal(current opsisdk.OperationsInsightsPrivateEndpoint) bool {
	return strings.ToUpper(string(current.LifecycleState)) == string(opsisdk.OperationsInsightsPrivateEndpointLifecycleStateDeleted)
}

func operationsInsightsPrivateEndpointDefinedTagsFromSpec(
	spec map[string]shared.MapValue,
) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	definedTags := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		definedTags[namespace] = converted
	}
	return definedTags
}

func operationsInsightsPrivateEndpointDefinedTagsFromStatus(
	status map[string]shared.MapValue,
) map[string]map[string]interface{} {
	return operationsInsightsPrivateEndpointDefinedTagsFromSpec(status)
}

func operationsInsightsPrivateEndpointStatusTagsFromSDK(
	input map[string]map[string]interface{},
) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func operationsInsightsPrivateEndpointStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(operationsInsightsPrivateEndpointStringValue(current)) == strings.TrimSpace(desired)
}

func operationsInsightsPrivateEndpointStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func operationsInsightsPrivateEndpointBoolValue(value *bool) bool {
	return value != nil && *value
}

func normalizeOperationsInsightsPrivateEndpointDefaults(current *opsisdk.OperationsInsightsPrivateEndpoint) {
	if current != nil && current.IsUsedForRacDbs == nil {
		current.IsUsedForRacDbs = common.Bool(false)
	}
}

func normalizeOperationsInsightsPrivateEndpointSummaryDefaults(current *opsisdk.OperationsInsightsPrivateEndpointSummary) {
	if current != nil && current.IsUsedForRacDbs == nil {
		current.IsUsedForRacDbs = common.Bool(false)
	}
}

func cloneOperationsInsightsPrivateEndpointStringSlice(input []string) []string {
	if input == nil {
		return nil
	}
	output := make([]string, 0, len(input))
	for _, value := range input {
		if value := strings.TrimSpace(value); value != "" {
			output = append(output, value)
		}
	}
	return output
}

func operationsInsightsPrivateEndpointStringSlicesEqual(left []string, right []string) bool {
	left = cloneOperationsInsightsPrivateEndpointStringSlice(left)
	right = cloneOperationsInsightsPrivateEndpointStringSlice(right)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}

func operationsInsightsPrivateEndpointTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func operationsInsightsPrivateEndpointJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
