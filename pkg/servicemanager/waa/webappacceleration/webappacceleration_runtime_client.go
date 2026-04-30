/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappacceleration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	waasdk "github.com/oracle/oci-go-sdk/v65/waa"
	waav1beta1 "github.com/oracle/oci-service-operator/api/waa/v1beta1"
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

const webAppAccelerationLoadBalancerBackendType = string(waasdk.BackendTypeLoadBalancer)

type webAppAccelerationOCIClient interface {
	ChangeWebAppAccelerationCompartment(context.Context, waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error)
	CreateWebAppAcceleration(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error)
	GetWebAppAcceleration(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error)
	GetWorkRequest(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error)
	ListWebAppAccelerations(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error)
	UpdateWebAppAcceleration(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error)
	DeleteWebAppAcceleration(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error)
}

type webAppAccelerationChangeCompartmentCall func(context.Context, waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error)
type webAppAccelerationGetWorkRequestCall func(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error)

type ambiguousWebAppAccelerationNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousWebAppAccelerationNotFoundError) Error() string {
	return e.message
}

func (e ambiguousWebAppAccelerationNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type reviewedWebAppAccelerationServiceClient struct {
	hooks                    WebAppAccelerationRuntimeHooks
	changeCompartment        webAppAccelerationChangeCompartmentCall
	changeCompartmentInitErr error
	getWorkRequest           webAppAccelerationGetWorkRequestCall
	getWorkRequestInitErr    error
	initErr                  error
	log                      loggerutil.OSOKLogger
}

func init() {
	registerWebAppAccelerationRuntimeHooksMutator(func(manager *WebAppAccelerationServiceManager, hooks *WebAppAccelerationRuntimeHooks) {
		applyWebAppAccelerationRuntimeHooks(manager, hooks)
	})
}

func applyWebAppAccelerationRuntimeHooks(manager *WebAppAccelerationServiceManager, hooks *WebAppAccelerationRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedWebAppAccelerationRuntimeSemantics()
	hooks.BuildCreateBody = buildWebAppAccelerationCreateBody
	hooks.BuildUpdateBody = buildWebAppAccelerationUpdateBody
	hooks.List.Fields = webAppAccelerationListFields()
	wrapWebAppAccelerationReadAndDeleteCalls(hooks)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedWebAppAccelerationIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateWebAppAccelerationCreateOnlyDriftForResponse
	hooks.StatusHooks.ProjectStatus = projectWebAppAccelerationStatus
	hooks.DeleteHooks.HandleError = handleWebAppAccelerationDeleteError

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	changeCompartment, changeCompartmentInitErr := newWebAppAccelerationCompartmentMoveCall(manager)
	getWorkRequest, getWorkRequestInitErr := newWebAppAccelerationWorkRequestCall(manager)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate WebAppAccelerationServiceClient) WebAppAccelerationServiceClient {
		return &reviewedWebAppAccelerationServiceClient{
			hooks:                    *hooks,
			changeCompartment:        changeCompartment,
			changeCompartmentInitErr: changeCompartmentInitErr,
			getWorkRequest:           getWorkRequest,
			getWorkRequestInitErr:    getWorkRequestInitErr,
			initErr:                  webAppAccelerationGeneratedDelegateInitError(delegate),
			log:                      log,
		}
	})
}

func webAppAccelerationGeneratedDelegateInitError(delegate WebAppAccelerationServiceClient) error {
	if delegate == nil {
		return nil
	}

	var resource *waav1beta1.WebAppAcceleration
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || isWebAppAccelerationNilResourceProbeError(err) {
		return nil
	}
	return err
}

func isWebAppAccelerationNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

func newWebAppAccelerationCompartmentMoveCall(
	manager *WebAppAccelerationServiceManager,
) (webAppAccelerationChangeCompartmentCall, error) {
	if manager == nil || manager.Provider == nil {
		return nil, nil
	}
	client, err := waasdk.NewWaaClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client.ChangeWebAppAccelerationCompartment, nil
}

func newWebAppAccelerationWorkRequestCall(
	manager *WebAppAccelerationServiceManager,
) (webAppAccelerationGetWorkRequestCall, error) {
	if manager == nil || manager.Provider == nil {
		return nil, nil
	}
	client, err := waasdk.NewWorkRequestClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client.GetWorkRequest, nil
}

func wrapWebAppAccelerationReadAndDeleteCalls(hooks *WebAppAccelerationRuntimeHooks) {
	getCall := hooks.Get.Call
	if getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeWebAppAccelerationNotFoundError(err, "read")
		}
	}

	listCall := hooks.List.Call
	if listCall != nil {
		hooks.List.Call = func(ctx context.Context, request waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			return listWebAppAccelerationsAllPages(ctx, listCall, request)
		}
	}

	deleteCall := hooks.Delete.Call
	if deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeWebAppAccelerationNotFoundError(err, "delete")
		}
	}
}

func newWebAppAccelerationServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client webAppAccelerationOCIClient,
) WebAppAccelerationServiceClient {
	hooks := newWebAppAccelerationRuntimeHooksWithOCIClient(client)
	manager := &WebAppAccelerationServiceManager{Log: log}
	applyWebAppAccelerationRuntimeHooks(manager, &hooks)
	return &reviewedWebAppAccelerationServiceClient{
		hooks:             hooks,
		changeCompartment: client.ChangeWebAppAccelerationCompartment,
		getWorkRequest:    client.GetWorkRequest,
		log:               log,
	}
}

func newWebAppAccelerationRuntimeHooksWithOCIClient(client webAppAccelerationOCIClient) WebAppAccelerationRuntimeHooks {
	return WebAppAccelerationRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*waav1beta1.WebAppAcceleration]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*waav1beta1.WebAppAcceleration]{},
		StatusHooks:     generatedruntime.StatusHooks[*waav1beta1.WebAppAcceleration]{},
		ParityHooks:     generatedruntime.ParityHooks[*waav1beta1.WebAppAcceleration]{},
		Async:           generatedruntime.AsyncHooks[*waav1beta1.WebAppAcceleration]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*waav1beta1.WebAppAcceleration]{},
		Create: runtimeOperationHooks[waasdk.CreateWebAppAccelerationRequest, waasdk.CreateWebAppAccelerationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateWebAppAccelerationDetails", RequestName: "CreateWebAppAccelerationDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
				return client.CreateWebAppAcceleration(ctx, request)
			},
		},
		Get: runtimeOperationHooks[waasdk.GetWebAppAccelerationRequest, waasdk.GetWebAppAccelerationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "WebAppAccelerationId", RequestName: "webAppAccelerationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
				return client.GetWebAppAcceleration(ctx, request)
			},
		},
		List: runtimeOperationHooks[waasdk.ListWebAppAccelerationsRequest, waasdk.ListWebAppAccelerationsResponse]{
			Fields: webAppAccelerationListFields(),
			Call: func(ctx context.Context, request waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
				return client.ListWebAppAccelerations(ctx, request)
			},
		},
		Update: runtimeOperationHooks[waasdk.UpdateWebAppAccelerationRequest, waasdk.UpdateWebAppAccelerationResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "WebAppAccelerationId", RequestName: "webAppAccelerationId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateWebAppAccelerationDetails", RequestName: "UpdateWebAppAccelerationDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
				return client.UpdateWebAppAcceleration(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[waasdk.DeleteWebAppAccelerationRequest, waasdk.DeleteWebAppAccelerationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "WebAppAccelerationId", RequestName: "webAppAccelerationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
				return client.DeleteWebAppAcceleration(ctx, request)
			},
		},
		WrapGeneratedClient: []func(WebAppAccelerationServiceClient) WebAppAccelerationServiceClient{},
	}
}

func reviewedWebAppAccelerationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "waa",
		FormalSlug:    "webappacceleration",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(waasdk.WebAppAccelerationLifecycleStateCreating)},
			UpdatingStates:     []string{string(waasdk.WebAppAccelerationLifecycleStateUpdating)},
			ActiveStates:       []string{string(waasdk.WebAppAccelerationLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(waasdk.WebAppAccelerationLifecycleStateDeleting)},
			TerminalStates: []string{string(waasdk.WebAppAccelerationLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "webAppAccelerationPolicyId", "backendType", "loadBalancerId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "webAppAccelerationPolicyId", "compartmentId", "freeformTags", "definedTags", "systemTags"},
			ForceNew:      []string{"backendType", "loadBalancerId"},
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

func webAppAccelerationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths: []string{
				"status.compartmentId",
				"spec.compartmentId",
				"compartmentId",
			},
		},
		{
			FieldName:    "WebAppAccelerationPolicyId",
			RequestName:  "webAppAccelerationPolicyId",
			Contribution: "query",
			LookupPaths: []string{
				"status.webAppAccelerationPolicyId",
				"spec.webAppAccelerationPolicyId",
				"webAppAccelerationPolicyId",
			},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths: []string{
				"status.displayName",
				"spec.displayName",
				"displayName",
			},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func (c *reviewedWebAppAccelerationServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.failCreateOrUpdate(resource, c.initErr)
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAcceleration resource is nil")
	}
	if err := validateWebAppAccelerationSpec(resource.Spec); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	current, found, err := c.readCurrent(ctx, resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if found {
		return c.reconcileExisting(ctx, resource, current)
	}
	return c.create(ctx, resource, req)
}

func (c *reviewedWebAppAccelerationServiceClient) Delete(ctx context.Context, resource *waav1beta1.WebAppAcceleration) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	if resource == nil {
		return false, fmt.Errorf("WebAppAcceleration resource is nil")
	}
	if handled, err := c.guardPendingWriteBeforeDelete(ctx, resource); handled || err != nil {
		return false, err
	}
	currentID := webAppAccelerationStatusID(resource)
	if currentID == "" {
		c.markDeleteConfirmed(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	if pending, ok := webAppAccelerationPendingDelete(resource); ok && strings.TrimSpace(pending.WorkRequestID) != "" {
		return c.resumePendingDeleteWorkRequest(ctx, resource, currentID, pending)
	}

	return c.deleteTrackedWebAppAcceleration(ctx, resource, currentID)
}

func (c *reviewedWebAppAccelerationServiceClient) deleteTrackedWebAppAcceleration(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
) (bool, error) {
	current, found, err := c.confirmDeleteRead(ctx, resource, currentID)
	if err != nil {
		return false, err
	}
	if handled, deleted, err := c.handleDeleteReadState(
		resource,
		current,
		found,
		"OCI resource no longer exists",
	); handled {
		return deleted, err
	}

	return c.requestWebAppAccelerationDelete(ctx, resource, currentID)
}

func (c *reviewedWebAppAccelerationServiceClient) requestWebAppAccelerationDelete(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
) (bool, error) {
	response, err := c.hooks.Delete.Call(ctx, waasdk.DeleteWebAppAccelerationRequest{
		WebAppAccelerationId: common.String(currentID),
	})
	if err != nil {
		return c.handleDeleteCallError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.seedDeleteWorkRequest(resource, response)

	return c.confirmWebAppAccelerationDeleted(ctx, resource, currentID)
}

func (c *reviewedWebAppAccelerationServiceClient) handleDeleteCallError(
	resource *waav1beta1.WebAppAcceleration,
	err error,
) (bool, error) {
	err = handleWebAppAccelerationDeleteError(resource, err)
	if isWebAppAccelerationUnambiguousNotFound(err) {
		c.markDeleteConfirmed(resource, "OCI resource no longer exists")
		return true, nil
	}
	return false, err
}

func (c *reviewedWebAppAccelerationServiceClient) confirmWebAppAccelerationDeleted(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
) (bool, error) {
	if pending, ok := webAppAccelerationPendingDelete(resource); ok && strings.TrimSpace(pending.WorkRequestID) != "" {
		return c.resumePendingDeleteWorkRequest(ctx, resource, currentID, pending)
	}

	refreshed, found, err := c.confirmDeleteRead(ctx, resource, currentID)
	if err != nil {
		return false, err
	}
	if handled, deleted, err := c.handleDeleteReadState(
		resource,
		refreshed,
		found,
		"OCI resource deleted",
	); handled {
		return deleted, err
	}
	return false, fmt.Errorf("WebAppAcceleration delete confirmation returned unexpected lifecycle state %q", refreshed.LifecycleState)
}

func (c *reviewedWebAppAccelerationServiceClient) resumePendingDeleteWorkRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
	pending shared.OSOKAsyncOperation,
) (bool, error) {
	next, err := c.observePendingWorkRequest(ctx, resource, pending)
	if err != nil {
		_, failErr := c.failPendingWorkRequest(resource, &pending, err)
		return false, failErr
	}
	switch next.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.markObservedWorkRequest(resource, next)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.confirmDeleteAfterSucceededWorkRequest(ctx, resource, currentID, next)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, failErr := c.failObservedWorkRequest(resource, next)
		return false, failErr
	default:
		err := fmt.Errorf(
			"WebAppAcceleration %s work request %s projected unsupported async class %s",
			next.Phase,
			next.WorkRequestID,
			next.NormalizedClass,
		)
		_, failErr := c.failPendingWorkRequest(resource, next, err)
		return false, failErr
	}
}

func (c *reviewedWebAppAccelerationServiceClient) confirmDeleteAfterSucceededWorkRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
	observed *shared.OSOKAsyncOperation,
) (bool, error) {
	refreshed, found, err := c.confirmDeleteRead(ctx, resource, currentID)
	if err != nil {
		_, failErr := c.failPendingWorkRequest(resource, observed, err)
		return false, failErr
	}
	if !found || refreshed.LifecycleState == waasdk.WebAppAccelerationLifecycleStateDeleted {
		c.markDeleteConfirmed(resource, "OCI resource deleted")
		return true, nil
	}
	_ = c.markDeleteWorkRequestWaitingForConfirmation(resource, refreshed, observed)
	return false, nil
}

func (c *reviewedWebAppAccelerationServiceClient) markDeleteWorkRequestWaitingForConfirmation(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	observed *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	next := observed.DeepCopy()
	if next == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = webAppAccelerationWorkRequestSucceededWaitingMessage(next)
	return c.markObservedWorkRequestForCurrent(resource, current, next)
}

func (c *reviewedWebAppAccelerationServiceClient) handleDeleteReadState(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	found bool,
	confirmedMessage string,
) (bool, bool, error) {
	if !found {
		c.markDeleteConfirmed(resource, confirmedMessage)
		return true, true, nil
	}
	switch current.LifecycleState {
	case waasdk.WebAppAccelerationLifecycleStateDeleted:
		c.markDeleteConfirmed(resource, "OCI resource deleted")
		return true, true, nil
	case waasdk.WebAppAccelerationLifecycleStateCreating,
		waasdk.WebAppAccelerationLifecycleStateUpdating:
		err := c.markPendingWriteReadbackBeforeDelete(resource, current)
		return true, false, err
	case waasdk.WebAppAccelerationLifecycleStateDeleting:
		deleted, err := c.markDeletePending(resource, current)
		return true, deleted, err
	default:
		if webAppAccelerationHasPendingDelete(resource) {
			deleted, err := c.markDeletePending(resource, current)
			return true, deleted, err
		}
		return false, false, nil
	}
}

func (c *reviewedWebAppAccelerationServiceClient) confirmDeleteRead(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
) (waasdk.WebAppAccelerationLoadBalancer, bool, error) {
	if c.hooks.Get.Call == nil {
		return waasdk.WebAppAccelerationLoadBalancer{}, false, fmt.Errorf("WebAppAcceleration delete confirmation requires a readable OCI operation")
	}
	response, err := c.hooks.Get.Call(ctx, waasdk.GetWebAppAccelerationRequest{
		WebAppAccelerationId: common.String(currentID),
	})
	if err != nil {
		err = handleWebAppAccelerationDeleteError(resource, err)
		if isWebAppAccelerationUnambiguousNotFound(err) {
			return waasdk.WebAppAccelerationLoadBalancer{}, false, nil
		}
		return waasdk.WebAppAccelerationLoadBalancer{}, false, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current, ok := webAppAccelerationFromResponse(response)
	if !ok {
		return waasdk.WebAppAccelerationLoadBalancer{}, false, fmt.Errorf("delete confirmation WebAppAcceleration response does not expose a load balancer body")
	}
	return current, true, nil
}

func (c *reviewedWebAppAccelerationServiceClient) markDeletePending(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (bool, error) {
	if pending, ok := webAppAccelerationPendingDelete(resource); ok {
		return c.markDeleteWorkRequestPending(resource, current, pending)
	}
	_, err := c.applySuccess(resource, current, shared.Terminating)
	return false, err
}

func (c *reviewedWebAppAccelerationServiceClient) markDeleteWorkRequestPending(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	pending shared.OSOKAsyncOperation,
) (bool, error) {
	if err := projectWebAppAccelerationStatus(resource, current); err != nil {
		return false, err
	}
	now := metav1.Now()
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    strings.TrimSpace(pending.WorkRequestID),
		RawStatus:        string(current.LifecycleState),
		RawOperationType: string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          webAppAccelerationDeletePendingMessage(pending.WorkRequestID),
		UpdatedAt:        &now,
	}, c.log)
	resource.Status.OsokStatus.UpdatedAt = &now
	resource.Status.OsokStatus.Reason = string(projection.Condition)
	return false, nil
}

func (c *reviewedWebAppAccelerationServiceClient) markDeleteConfirmed(
	resource *waav1beta1.WebAppAcceleration,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	if strings.TrimSpace(message) != "" {
		status.Message = message
	}
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.log)
}

func (c *reviewedWebAppAccelerationServiceClient) readCurrent(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
) (waasdk.WebAppAccelerationLoadBalancer, bool, error) {
	currentID := webAppAccelerationStatusID(resource)
	if currentID != "" {
		current, found, err := c.readCurrentByID(ctx, resource, currentID)
		if err != nil || found {
			return current, found, err
		}
		clearTrackedWebAppAccelerationIdentity(resource)
	}

	return c.findExisting(ctx, resource)
}

func (c *reviewedWebAppAccelerationServiceClient) readCurrentByID(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
) (waasdk.WebAppAccelerationLoadBalancer, bool, error) {
	response, err := c.hooks.Get.Call(ctx, waasdk.GetWebAppAccelerationRequest{
		WebAppAccelerationId: common.String(currentID),
	})
	switch {
	case err == nil:
		current, ok := webAppAccelerationFromResponse(response)
		if !ok {
			return waasdk.WebAppAccelerationLoadBalancer{}, false, fmt.Errorf("current WebAppAcceleration response does not expose a load balancer body")
		}
		servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
		if current.LifecycleState == waasdk.WebAppAccelerationLifecycleStateDeleted {
			return waasdk.WebAppAccelerationLoadBalancer{}, false, nil
		}
		return current, true, nil
	case isWebAppAccelerationUnambiguousNotFound(err):
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return waasdk.WebAppAccelerationLoadBalancer{}, false, nil
	default:
		return waasdk.WebAppAccelerationLoadBalancer{}, false, err
	}
}

func (c *reviewedWebAppAccelerationServiceClient) findExisting(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
) (waasdk.WebAppAccelerationLoadBalancer, bool, error) {
	request := waasdk.ListWebAppAccelerationsRequest{
		CompartmentId:              common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		WebAppAccelerationPolicyId: common.String(strings.TrimSpace(resource.Spec.WebAppAccelerationPolicyId)),
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		request.DisplayName = common.String(displayName)
	}

	response, err := c.hooks.List.Call(ctx, request)
	if err != nil {
		return waasdk.WebAppAccelerationLoadBalancer{}, false, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	matches := make([]waasdk.WebAppAccelerationLoadBalancer, 0, len(response.Items))
	for _, item := range response.Items {
		current, ok := webAppAccelerationFromResponse(item)
		if !ok || webAppAccelerationIsDeleted(current.LifecycleState) {
			continue
		}
		matched, err := webAppAccelerationMatchesSpec(resource.Spec, current)
		if err != nil {
			return waasdk.WebAppAccelerationLoadBalancer{}, false, err
		}
		if matched {
			matches = append(matches, current)
		}
	}

	switch len(matches) {
	case 0:
		return waasdk.WebAppAccelerationLoadBalancer{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return waasdk.WebAppAccelerationLoadBalancer{}, false, fmt.Errorf("WebAppAcceleration list response returned multiple matching resources")
	}
}

func (c *reviewedWebAppAccelerationServiceClient) reconcileExisting(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (servicemanager.OSOKResponse, error) {
	if err := projectWebAppAccelerationStatus(resource, current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if err := validateWebAppAccelerationCreateOnlyDrift(resource.Spec, current); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if handled, osokResponse, err := c.resumePendingWriteIfNeeded(ctx, resource, current); handled || err != nil {
		return osokResponse, err
	}
	if webAppAccelerationLifecyclePending(current.LifecycleState) || current.LifecycleState == waasdk.WebAppAccelerationLifecycleStateFailed {
		return c.applySuccess(resource, current, shared.Active)
	}

	current, handled, osokResponse, err := c.applyCompartmentMoveIfNeeded(ctx, resource, current)
	if handled || err != nil {
		return osokResponse, err
	}

	return c.reconcileUpdate(ctx, resource, current)
}

func (c *reviewedWebAppAccelerationServiceClient) reconcileUpdate(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (servicemanager.OSOKResponse, error) {
	if err := validateWebAppAccelerationCreateOnlyDrift(resource.Spec, current); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	updateDetails, updateNeeded, err := requiredWebAppAccelerationUpdateDetails(ctx, resource, current)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if !updateNeeded {
		return c.applySuccess(resource, current, shared.Active)
	}

	refreshed, pending, err := c.applyWebAppAccelerationUpdate(ctx, resource, current, updateDetails)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	osokResponse, pendingHandled, err := c.handleWebAppAccelerationUpdateReadback(ctx, resource, refreshed, pending)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if pendingHandled {
		return osokResponse, nil
	}
	return c.applySuccess(resource, refreshed, shared.Updating)
}

func requiredWebAppAccelerationUpdateDetails(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (waasdk.UpdateWebAppAccelerationDetails, bool, error) {
	body, updateNeeded, err := buildWebAppAccelerationUpdateBody(ctx, resource, "", current)
	if err != nil || !updateNeeded {
		return waasdk.UpdateWebAppAccelerationDetails{}, updateNeeded, err
	}
	updateDetails, ok := body.(waasdk.UpdateWebAppAccelerationDetails)
	if !ok {
		return waasdk.UpdateWebAppAccelerationDetails{}, false, fmt.Errorf("WebAppAcceleration update body has unexpected type %T", body)
	}
	return updateDetails, true, nil
}

func (c *reviewedWebAppAccelerationServiceClient) applyWebAppAccelerationUpdate(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	updateDetails waasdk.UpdateWebAppAccelerationDetails,
) (any, *shared.OSOKAsyncOperation, error) {
	currentID := webAppAccelerationStringValue(current.Id)
	response, err := c.hooks.Update.Call(ctx, waasdk.UpdateWebAppAccelerationRequest{
		WebAppAccelerationId:            common.String(currentID),
		UpdateWebAppAccelerationDetails: updateDetails,
	})
	if err != nil {
		return nil, nil, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	pending := newWebAppAccelerationUpdateWorkRequest(response.OpcWorkRequestId)
	if pending != nil {
		c.applyWriteWorkRequest(resource, pending)
	}
	refreshed, err := c.readWebAppAccelerationAfterUpdate(ctx, resource, currentID, response)
	return refreshed, pending, err
}

func (c *reviewedWebAppAccelerationServiceClient) readWebAppAccelerationAfterUpdate(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	currentID string,
	response waasdk.UpdateWebAppAccelerationResponse,
) (any, error) {
	if currentID != "" {
		if getResponse, getErr := c.hooks.Get.Call(ctx, waasdk.GetWebAppAccelerationRequest{WebAppAccelerationId: common.String(currentID)}); getErr == nil {
			servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, getResponse)
			return getResponse, nil
		} else if !isWebAppAccelerationUnambiguousNotFound(getErr) {
			return nil, getErr
		}
	}
	return response, nil
}

func (c *reviewedWebAppAccelerationServiceClient) handleWebAppAccelerationUpdateReadback(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	refreshed any,
	pending *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, bool, error) {
	if current, ok := webAppAccelerationFromResponse(refreshed); ok {
		stillPending, err := webAppAccelerationWriteReadbackPending(resource, current)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, false, err
		}
		if stillPending {
			if pending == nil {
				pending = newWebAppAccelerationUpdatePendingOperation("")
			}
			response, err := c.handlePendingWriteWorkRequest(ctx, resource, current, pending)
			return response, true, err
		}
	}
	return servicemanager.OSOKResponse{}, false, nil
}

func (c *reviewedWebAppAccelerationServiceClient) resumePendingWriteIfNeeded(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (bool, servicemanager.OSOKResponse, error) {
	if current.LifecycleState == waasdk.WebAppAccelerationLifecycleStateFailed {
		return false, servicemanager.OSOKResponse{}, nil
	}
	pending, ok := webAppAccelerationPendingWrite(resource)
	if !ok {
		return false, servicemanager.OSOKResponse{}, nil
	}
	stillPending, err := webAppAccelerationWriteReadbackPending(resource, current)
	if err != nil {
		osokResponse, failErr := c.failCreateOrUpdate(resource, err)
		return true, osokResponse, failErr
	}
	if !stillPending {
		return false, servicemanager.OSOKResponse{}, nil
	}
	osokResponse, err := c.handlePendingWriteWorkRequest(ctx, resource, current, &pending)
	return true, osokResponse, err
}

func (c *reviewedWebAppAccelerationServiceClient) applyCompartmentMoveIfNeeded(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (waasdk.WebAppAccelerationLoadBalancer, bool, servicemanager.OSOKResponse, error) {
	if !webAppAccelerationCompartmentNeedsChange(resource, current) {
		return current, false, servicemanager.OSOKResponse{}, nil
	}
	if webAppAccelerationHasPendingMove(resource) {
		response, err := c.handlePendingMoveWorkRequest(ctx, resource, current, servicemanager.OSOKResponse{})
		return waasdk.WebAppAccelerationLoadBalancer{}, true, response, err
	}

	refreshed, response, err := c.moveCompartment(ctx, resource, current)
	if err != nil {
		return waasdk.WebAppAccelerationLoadBalancer{}, true, response, err
	}
	if webAppAccelerationCompartmentNeedsChange(resource, refreshed) || webAppAccelerationLifecyclePending(refreshed.LifecycleState) {
		response, err := c.handlePendingMoveWorkRequest(ctx, resource, refreshed, response)
		return waasdk.WebAppAccelerationLoadBalancer{}, true, response, err
	}
	return refreshed, false, servicemanager.OSOKResponse{}, nil
}

func (c *reviewedWebAppAccelerationServiceClient) moveCompartment(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (waasdk.WebAppAccelerationLoadBalancer, servicemanager.OSOKResponse, error) {
	if c.changeCompartmentInitErr != nil {
		err := fmt.Errorf("initialize WebAppAcceleration OCI client for compartment move: %w", c.changeCompartmentInitErr)
		return c.failCompartmentMove(resource, err)
	}
	if c.changeCompartment == nil {
		err := fmt.Errorf("WebAppAcceleration compartment move operation is not configured")
		return c.failCompartmentMove(resource, err)
	}

	resourceID := webAppAccelerationResourceID(resource, current)
	if resourceID == "" {
		err := fmt.Errorf("WebAppAcceleration compartment move requires a tracked resource id")
		return c.failCompartmentMove(resource, err)
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		err := fmt.Errorf("WebAppAcceleration compartment move requires spec.compartmentId")
		return c.failCompartmentMove(resource, err)
	}

	response, err := c.changeCompartment(ctx, waasdk.ChangeWebAppAccelerationCompartmentRequest{
		WebAppAccelerationId: common.String(resourceID),
		ChangeWebAppAccelerationCompartmentDetails: waasdk.ChangeWebAppAccelerationCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
	})
	if err != nil {
		return c.failCompartmentMove(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.seedMoveWorkRequest(resource, response)

	refreshed, err := c.readAfterCompartmentMove(ctx, resourceID)
	if err != nil {
		return c.failCompartmentMove(resource, err)
	}
	return refreshed, servicemanager.OSOKResponse{}, nil
}

func (c *reviewedWebAppAccelerationServiceClient) failCompartmentMove(
	resource *waav1beta1.WebAppAcceleration,
	err error,
) (waasdk.WebAppAccelerationLoadBalancer, servicemanager.OSOKResponse, error) {
	response, err := c.failCreateOrUpdate(resource, err)
	return waasdk.WebAppAccelerationLoadBalancer{}, response, err
}

func (c *reviewedWebAppAccelerationServiceClient) readAfterCompartmentMove(
	ctx context.Context,
	resourceID string,
) (waasdk.WebAppAccelerationLoadBalancer, error) {
	response, err := c.hooks.Get.Call(ctx, waasdk.GetWebAppAccelerationRequest{
		WebAppAccelerationId: common.String(resourceID),
	})
	if err != nil {
		return waasdk.WebAppAccelerationLoadBalancer{}, fmt.Errorf("read WebAppAcceleration after compartment move: %w", err)
	}
	refreshed, ok := webAppAccelerationFromResponse(response)
	if !ok {
		return waasdk.WebAppAccelerationLoadBalancer{}, fmt.Errorf("read WebAppAcceleration after compartment move did not expose a load balancer body")
	}
	return refreshed, nil
}

func (c *reviewedWebAppAccelerationServiceClient) seedDeleteWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	response waasdk.DeleteWebAppAccelerationResponse,
) {
	workRequestID := strings.TrimSpace(webAppAccelerationStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return
	}
	now := metav1.Now()
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    workRequestID,
		RawOperationType: string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          webAppAccelerationDeletePendingMessage(workRequestID),
		UpdatedAt:        &now,
	}, c.log)
}

func (c *reviewedWebAppAccelerationServiceClient) seedMoveWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	response waasdk.ChangeWebAppAccelerationCompartmentResponse,
) {
	workRequestID := strings.TrimSpace(webAppAccelerationStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return
	}
	now := metav1.Now()
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    workRequestID,
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          webAppAccelerationMovePendingMessage(workRequestID),
		UpdatedAt:        &now,
	}, c.log)
}

func newWebAppAccelerationCreateWorkRequest(workRequestID *string) *shared.OSOKAsyncOperation {
	trimmed := strings.TrimSpace(webAppAccelerationStringValue(workRequestID))
	if trimmed == "" {
		return nil
	}
	return newWebAppAccelerationWriteWorkRequest(
		shared.OSOKAsyncPhaseCreate,
		trimmed,
		string(waasdk.WorkRequestOperationTypeCreateWebAppAcceleration),
		webAppAccelerationCreatePendingMessage(trimmed),
	)
}

func newWebAppAccelerationUpdateWorkRequest(workRequestID *string) *shared.OSOKAsyncOperation {
	trimmed := strings.TrimSpace(webAppAccelerationStringValue(workRequestID))
	if trimmed == "" {
		return nil
	}
	return newWebAppAccelerationUpdatePendingOperation(trimmed)
}

func newWebAppAccelerationUpdatePendingOperation(workRequestID string) *shared.OSOKAsyncOperation {
	return newWebAppAccelerationWriteWorkRequest(
		shared.OSOKAsyncPhaseUpdate,
		strings.TrimSpace(workRequestID),
		string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration),
		webAppAccelerationUpdatePendingMessage(workRequestID),
	)
}

func newWebAppAccelerationWriteWorkRequest(
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	rawOperationType string,
	message string,
) *shared.OSOKAsyncOperation {
	source := shared.OSOKAsyncSourceWorkRequest
	if strings.TrimSpace(workRequestID) == "" {
		source = shared.OSOKAsyncSourceLifecycle
	}
	now := metav1.Now()
	return &shared.OSOKAsyncOperation{
		Source:           source,
		Phase:            phase,
		WorkRequestID:    strings.TrimSpace(workRequestID),
		RawOperationType: strings.TrimSpace(rawOperationType),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          strings.TrimSpace(message),
		UpdatedAt:        &now,
	}
}

func (c *reviewedWebAppAccelerationServiceClient) applyWriteWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	current *shared.OSOKAsyncOperation,
) {
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *reviewedWebAppAccelerationServiceClient) observePendingWorkRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	pending shared.OSOKAsyncOperation,
) (*shared.OSOKAsyncOperation, error) {
	workRequest, err := c.fetchPendingWorkRequest(ctx, pending)
	if err != nil {
		return nil, err
	}
	return webAppAccelerationAsyncOperationFromWorkRequest(resource, pending, workRequest)
}

func (c *reviewedWebAppAccelerationServiceClient) fetchPendingWorkRequest(
	ctx context.Context,
	pending shared.OSOKAsyncOperation,
) (waasdk.WorkRequest, error) {
	if c.getWorkRequestInitErr != nil {
		return waasdk.WorkRequest{}, fmt.Errorf("initialize WebAppAcceleration work request OCI client: %w", c.getWorkRequestInitErr)
	}
	if c.getWorkRequest == nil {
		return waasdk.WorkRequest{}, fmt.Errorf("WebAppAcceleration work request operation is not configured")
	}
	workRequestID := strings.TrimSpace(pending.WorkRequestID)
	if workRequestID == "" {
		return waasdk.WorkRequest{}, fmt.Errorf("WebAppAcceleration pending %s work request is missing a work request ID", pending.Phase)
	}
	response, err := c.getWorkRequest(ctx, waasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return waasdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func webAppAccelerationAsyncOperationFromWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	pending shared.OSOKAsyncOperation,
	workRequest waasdk.WorkRequest,
) (*shared.OSOKAsyncOperation, error) {
	if resource == nil {
		return nil, fmt.Errorf("WebAppAcceleration resource is nil")
	}
	workRequestID := strings.TrimSpace(webAppAccelerationStringValue(workRequest.Id))
	if workRequestID == "" {
		workRequestID = strings.TrimSpace(pending.WorkRequestID)
	}
	operationType := strings.TrimSpace(string(workRequest.OperationType))
	if operationType == "" {
		operationType = strings.TrimSpace(pending.RawOperationType)
	}
	current, err := servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, webAppAccelerationWorkRequestAsyncAdapter(), servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        operationType,
		RawOperationType: operationType,
		WorkRequestID:    workRequestID,
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    pending.Phase,
	})
	if err != nil {
		return nil, err
	}
	if current.WorkRequestID == "" {
		current.WorkRequestID = strings.TrimSpace(pending.WorkRequestID)
	}
	if current.RawOperationType == "" {
		current.RawOperationType = strings.TrimSpace(pending.RawOperationType)
	}
	if message := webAppAccelerationWorkRequestMessage(current); message != "" {
		current.Message = message
	}
	return current, nil
}

func webAppAccelerationWorkRequestAsyncAdapter() servicemanager.WorkRequestAsyncAdapter {
	return servicemanager.WorkRequestAsyncAdapter{
		PendingStatusTokens: []string{
			string(waasdk.WorkRequestStatusAccepted),
			string(waasdk.WorkRequestStatusInProgress),
			string(waasdk.WorkRequestStatusCanceling),
		},
		SucceededStatusTokens: []string{string(waasdk.WorkRequestStatusSucceeded)},
		FailedStatusTokens:    []string{string(waasdk.WorkRequestStatusFailed)},
		CanceledStatusTokens:  []string{string(waasdk.WorkRequestStatusCanceled)},
		CreateActionTokens:    []string{string(waasdk.WorkRequestOperationTypeCreateWebAppAcceleration)},
		UpdateActionTokens: []string{
			string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration),
			string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		},
		DeleteActionTokens: []string{string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration)},
	}
}

func webAppAccelerationWorkRequestMessage(current *shared.OSOKAsyncOperation) string {
	if current == nil {
		return ""
	}
	label := webAppAccelerationWorkRequestLabel(current)
	switch {
	case current.WorkRequestID != "" && current.RawStatus != "":
		return fmt.Sprintf("WebAppAcceleration %s work request %s is %s", label, current.WorkRequestID, current.RawStatus)
	case current.RawStatus != "":
		return fmt.Sprintf("WebAppAcceleration %s work request is %s", label, current.RawStatus)
	default:
		return ""
	}
}

func webAppAccelerationWorkRequestSucceededWaitingMessage(current *shared.OSOKAsyncOperation) string {
	label := webAppAccelerationWorkRequestLabel(current)
	if current != nil && strings.TrimSpace(current.WorkRequestID) != "" {
		return fmt.Sprintf("WebAppAcceleration %s work request %s succeeded; waiting for OCI readback confirmation", label, current.WorkRequestID)
	}
	return fmt.Sprintf("WebAppAcceleration %s work request succeeded; waiting for OCI readback confirmation", label)
}

func webAppAccelerationWorkRequestLabel(current *shared.OSOKAsyncOperation) string {
	if current == nil {
		return "operation"
	}
	if strings.TrimSpace(current.RawOperationType) == string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration) {
		return "compartment move"
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate:
		return "create"
	case shared.OSOKAsyncPhaseUpdate:
		return "update"
	case shared.OSOKAsyncPhaseDelete:
		return "delete"
	default:
		return "operation"
	}
}

func (c *reviewedWebAppAccelerationServiceClient) markObservedWorkRequestForCurrent(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	observed *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if err := projectWebAppAccelerationStatus(resource, current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	if id := webAppAccelerationResourceID(resource, current); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	return c.markObservedWorkRequest(resource, observed)
}

func (c *reviewedWebAppAccelerationServiceClient) markObservedWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next := current.DeepCopy()
	if next == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	now := metav1.Now()
	if next.UpdatedAt == nil {
		next.UpdatedAt = &now
	}
	status := &resource.Status.OsokStatus
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, next, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func (c *reviewedWebAppAccelerationServiceClient) failObservedWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	current *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, error) {
	err := fmt.Errorf(
		"WebAppAcceleration %s work request %s finished with status %s",
		current.Phase,
		current.WorkRequestID,
		current.RawStatus,
	)
	return c.failPendingWorkRequest(resource, current, err)
}

func (c *reviewedWebAppAccelerationServiceClient) failPendingWorkRequest(
	resource *waav1beta1.WebAppAcceleration,
	current *shared.OSOKAsyncOperation,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if current == nil {
		return c.failCreateOrUpdate(resource, err)
	}
	next := current.DeepCopy()
	if next == nil {
		return c.failCreateOrUpdate(resource, err)
	}
	switch next.NormalizedClass {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		next.NormalizedClass = shared.OSOKAsyncClassFailed
	}
	next.Message = err.Error()
	return c.markObservedWorkRequest(resource, next), err
}

func (c *reviewedWebAppAccelerationServiceClient) markWriteWorkRequestPending(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	pending *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if err := projectWebAppAccelerationStatus(resource, current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	if id := webAppAccelerationResourceID(resource, current); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}

	next := pending.DeepCopy()
	if next == nil {
		next = newWebAppAccelerationUpdatePendingOperation("")
	}
	next.RawStatus = string(current.LifecycleState)
	next.NormalizedClass = shared.OSOKAsyncClassPending
	if next.RawOperationType == "" {
		next.RawOperationType = webAppAccelerationWriteOperationType(next.Phase)
	}
	if strings.TrimSpace(next.Message) == "" {
		next.Message = webAppAccelerationWritePendingMessage(next.Phase, next.WorkRequestID)
	}
	now := metav1.Now()
	next.UpdatedAt = &now
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, next, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func (c *reviewedWebAppAccelerationServiceClient) handlePendingWriteWorkRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	pending *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, error) {
	if pending == nil || strings.TrimSpace(pending.WorkRequestID) == "" {
		return c.markWriteWorkRequestPending(resource, current, pending), nil
	}
	next, err := c.observePendingWorkRequest(ctx, resource, *pending)
	if err != nil {
		return c.failPendingWorkRequest(resource, pending, err)
	}
	switch next.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.markObservedWorkRequestForCurrent(resource, current, next), nil
	case shared.OSOKAsyncClassSucceeded:
		return c.markWriteWorkRequestWaitingForReadback(resource, current, next), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failObservedWorkRequest(resource, next)
	default:
		err := fmt.Errorf(
			"WebAppAcceleration %s work request %s projected unsupported async class %s",
			next.Phase,
			next.WorkRequestID,
			next.NormalizedClass,
		)
		return c.failPendingWorkRequest(resource, next, err)
	}
}

func (c *reviewedWebAppAccelerationServiceClient) markWriteWorkRequestWaitingForReadback(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	observed *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	next := observed.DeepCopy()
	if next == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = webAppAccelerationWorkRequestSucceededWaitingMessage(next)
	return c.markObservedWorkRequestForCurrent(resource, current, next)
}

func (c *reviewedWebAppAccelerationServiceClient) guardPendingWriteBeforeDelete(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
) (bool, error) {
	pending, ok := webAppAccelerationPendingWriteBeforeDelete(resource)
	if !ok {
		return false, nil
	}
	if strings.TrimSpace(pending.WorkRequestID) == "" {
		return c.guardPendingLifecycleWriteBeforeDelete(ctx, resource, pending)
	}

	next, err := c.observePendingWorkRequest(ctx, resource, pending)
	if err != nil {
		_, failErr := c.failPendingWorkRequest(resource, &pending, err)
		return true, failErr
	}
	switch next.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.markObservedWorkRequest(resource, next)
		return true, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.guardSucceededWriteBeforeDelete(ctx, resource, next)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, failErr := c.failObservedWorkRequest(resource, next)
		return true, failErr
	default:
		err := fmt.Errorf(
			"WebAppAcceleration %s work request %s projected unsupported async class %s",
			next.Phase,
			next.WorkRequestID,
			next.NormalizedClass,
		)
		_, failErr := c.failPendingWorkRequest(resource, next, err)
		return true, failErr
	}
}

func (c *reviewedWebAppAccelerationServiceClient) guardPendingLifecycleWriteBeforeDelete(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	pending shared.OSOKAsyncOperation,
) (bool, error) {
	current, found, err := c.readPendingWriteBeforeDelete(ctx, resource)
	if err != nil {
		return true, err
	}
	if found {
		return c.guardWriteReadbackBeforeDelete(resource, current)
	}
	_ = c.markPendingWriteWithoutReadback(resource, &pending)
	return true, nil
}

func (c *reviewedWebAppAccelerationServiceClient) guardSucceededWriteBeforeDelete(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	pending *shared.OSOKAsyncOperation,
) (bool, error) {
	current, found, err := c.readPendingWriteBeforeDelete(ctx, resource)
	if err != nil {
		_, failErr := c.failPendingWorkRequest(resource, pending, err)
		return true, failErr
	}
	if !found {
		servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
		return false, nil
	}
	handled, err := c.guardWriteReadbackBeforeDelete(resource, current)
	if handled || err != nil {
		return handled, err
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return false, nil
}

func (c *reviewedWebAppAccelerationServiceClient) readPendingWriteBeforeDelete(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
) (waasdk.WebAppAccelerationLoadBalancer, bool, error) {
	currentID := webAppAccelerationStatusID(resource)
	if currentID == "" {
		return waasdk.WebAppAccelerationLoadBalancer{}, false, nil
	}
	return c.confirmDeleteRead(ctx, resource, currentID)
}

func (c *reviewedWebAppAccelerationServiceClient) guardWriteReadbackBeforeDelete(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (bool, error) {
	switch current.LifecycleState {
	case waasdk.WebAppAccelerationLifecycleStateCreating,
		waasdk.WebAppAccelerationLifecycleStateUpdating:
		return true, c.markPendingWriteReadbackBeforeDelete(resource, current)
	case waasdk.WebAppAccelerationLifecycleStateDeleting:
		_, err := c.markDeletePending(resource, current)
		return true, err
	default:
		return false, nil
	}
}

func (c *reviewedWebAppAccelerationServiceClient) markPendingWriteWithoutReadback(
	resource *waav1beta1.WebAppAcceleration,
	pending *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || pending == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next := pending.DeepCopy()
	if next == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	if strings.TrimSpace(string(next.Source)) == "" {
		next.Source = shared.OSOKAsyncSourceLifecycle
	}
	next.NormalizedClass = shared.OSOKAsyncClassPending
	if strings.TrimSpace(next.RawOperationType) == "" {
		next.RawOperationType = webAppAccelerationWriteOperationType(next.Phase)
	}
	next.Message = webAppAccelerationPendingWriteBeforeDeleteMessage(next)
	now := metav1.Now()
	next.UpdatedAt = &now
	return c.markObservedWorkRequest(resource, next)
}

func (c *reviewedWebAppAccelerationServiceClient) markPendingWriteReadbackBeforeDelete(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) error {
	message := fmt.Sprintf(
		"WebAppAcceleration lifecycle state %s is still pending; retaining finalizer before delete",
		current.LifecycleState,
	)
	phase := shared.OSOKAsyncPhaseUpdate
	if current.LifecycleState == waasdk.WebAppAccelerationLifecycleStateCreating {
		phase = shared.OSOKAsyncPhaseCreate
	}
	async := newWebAppAccelerationLifecycleAsync(message, string(current.LifecycleState), phase, shared.OSOKAsyncClassPending)
	response := c.markObservedWorkRequestForCurrent(resource, current, async)
	if !response.IsSuccessful {
		return fmt.Errorf("project WebAppAcceleration pending write before delete")
	}
	return nil
}

func webAppAccelerationWriteReadbackPending(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) (bool, error) {
	if webAppAccelerationLifecyclePending(current.LifecycleState) {
		return true, nil
	}
	matches, err := webAppAccelerationMatchesSpec(resource.Spec, current)
	if err != nil {
		return false, err
	}
	if !matches {
		return true, nil
	}
	_, updateNeeded, err := buildWebAppAccelerationUpdateBody(context.Background(), resource, "", current)
	if err != nil {
		return false, err
	}
	return updateNeeded, nil
}

func (c *reviewedWebAppAccelerationServiceClient) markMovePending(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	response servicemanager.OSOKResponse,
) servicemanager.OSOKResponse {
	if err := projectWebAppAccelerationStatus(resource, current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	workRequestID := ""
	hasPendingMove := false
	if current, ok := webAppAccelerationPendingMove(resource); ok {
		workRequestID = strings.TrimSpace(current.WorkRequestID)
		hasPendingMove = true
	}
	if !hasPendingMove {
		return c.mustApplySuccess(resource, current, shared.Updating)
	}
	now := metav1.Now()
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    workRequestID,
		RawStatus:        string(current.LifecycleState),
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          webAppAccelerationMovePendingMessage(workRequestID),
		UpdatedAt:        &now,
	}, c.log)
	response.IsSuccessful = projection.Condition != shared.Failed
	response.ShouldRequeue = projection.ShouldRequeue
	response.RequeueDuration = time.Minute
	return response
}

func (c *reviewedWebAppAccelerationServiceClient) handlePendingMoveWorkRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	response servicemanager.OSOKResponse,
) (servicemanager.OSOKResponse, error) {
	pending, ok := webAppAccelerationPendingMove(resource)
	if !ok || strings.TrimSpace(pending.WorkRequestID) == "" {
		return c.markMovePending(resource, current, response), nil
	}
	next, err := c.observePendingWorkRequest(ctx, resource, pending)
	if err != nil {
		return c.failPendingWorkRequest(resource, &pending, err)
	}
	switch next.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.markObservedWorkRequestForCurrent(resource, current, next), nil
	case shared.OSOKAsyncClassSucceeded:
		if webAppAccelerationCompartmentNeedsChange(resource, current) || webAppAccelerationLifecyclePending(current.LifecycleState) {
			return c.markMoveWorkRequestWaitingForReadback(resource, current, next), nil
		}
		return c.applySuccess(resource, current, shared.Updating)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failObservedWorkRequest(resource, next)
	default:
		err := fmt.Errorf(
			"WebAppAcceleration %s work request %s projected unsupported async class %s",
			next.Phase,
			next.WorkRequestID,
			next.NormalizedClass,
		)
		return c.failPendingWorkRequest(resource, next, err)
	}
}

func (c *reviewedWebAppAccelerationServiceClient) markMoveWorkRequestWaitingForReadback(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
	observed *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	next := observed.DeepCopy()
	if next == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = webAppAccelerationWorkRequestSucceededWaitingMessage(next)
	return c.markObservedWorkRequestForCurrent(resource, current, next)
}

func (c *reviewedWebAppAccelerationServiceClient) mustApplySuccess(
	resource *waav1beta1.WebAppAcceleration,
	response any,
	fallback shared.OSOKConditionType,
) servicemanager.OSOKResponse {
	osokResponse, err := c.applySuccess(resource, response, fallback)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	return osokResponse
}

func (c *reviewedWebAppAccelerationServiceClient) create(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	request, err := webAppAccelerationCreateRequest(ctx, resource, req)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	response, err := c.hooks.Create.Call(ctx, request)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	refreshed, pending, err := c.handleWebAppAccelerationCreateResponse(ctx, resource, response)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	osokResponse, pendingHandled, err := c.handleWebAppAccelerationCreateReadback(ctx, resource, refreshed, pending)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if pendingHandled {
		return osokResponse, nil
	}
	return c.applySuccess(resource, refreshed, shared.Provisioning)
}

func webAppAccelerationCreateRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	req ctrl.Request,
) (waasdk.CreateWebAppAccelerationRequest, error) {
	body, err := buildWebAppAccelerationCreateBody(ctx, resource, req.Namespace)
	if err != nil {
		return waasdk.CreateWebAppAccelerationRequest{}, err
	}
	request := waasdk.CreateWebAppAccelerationRequest{
		CreateWebAppAccelerationDetails: body.(waasdk.CreateWebAppAccelerationDetails),
	}
	if resource.UID != "" {
		request.OpcRetryToken = common.String(string(resource.UID))
	}
	return request, nil
}

func (c *reviewedWebAppAccelerationServiceClient) handleWebAppAccelerationCreateResponse(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	response waasdk.CreateWebAppAccelerationResponse,
) (any, *shared.OSOKAsyncOperation, error) {
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	pending := newWebAppAccelerationCreateWorkRequest(response.OpcWorkRequestId)
	if pending != nil {
		c.applyWriteWorkRequest(resource, pending)
	}
	refreshed, err := c.readWebAppAccelerationAfterCreate(ctx, resource, response)
	return refreshed, pending, err
}

func (c *reviewedWebAppAccelerationServiceClient) readWebAppAccelerationAfterCreate(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	response waasdk.CreateWebAppAccelerationResponse,
) (any, error) {
	if currentID := webAppAccelerationResponseID(response); currentID != "" {
		if getResponse, getErr := c.hooks.Get.Call(ctx, waasdk.GetWebAppAccelerationRequest{WebAppAccelerationId: common.String(currentID)}); getErr == nil {
			servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, getResponse)
			return getResponse, nil
		} else if !isWebAppAccelerationUnambiguousNotFound(getErr) {
			return nil, getErr
		}
	}
	return response, nil
}

func (c *reviewedWebAppAccelerationServiceClient) handleWebAppAccelerationCreateReadback(
	ctx context.Context,
	resource *waav1beta1.WebAppAcceleration,
	refreshed any,
	pending *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, bool, error) {
	if pending != nil {
		if current, ok := webAppAccelerationFromResponse(refreshed); ok {
			stillPending, err := webAppAccelerationWriteReadbackPending(resource, current)
			if err != nil {
				return servicemanager.OSOKResponse{IsSuccessful: false}, false, err
			}
			if stillPending {
				response, err := c.handlePendingWriteWorkRequest(ctx, resource, current, pending)
				return response, true, err
			}
		}
	}
	return servicemanager.OSOKResponse{}, false, nil
}

func (c *reviewedWebAppAccelerationServiceClient) applySuccess(
	resource *waav1beta1.WebAppAcceleration,
	response any,
	fallback shared.OSOKConditionType,
) (servicemanager.OSOKResponse, error) {
	if err := projectWebAppAccelerationStatus(resource, response); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	status := &resource.Status.OsokStatus
	if id := webAppAccelerationResponseID(response); id != "" {
		status.Ocid = shared.OCID(id)
	}
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}

	state := strings.ToUpper(strings.TrimSpace(resource.Status.LifecycleState))
	message := strings.TrimSpace(resource.Status.LifecycleDetails)
	if message == "" {
		message = strings.TrimSpace(resource.Status.DisplayName)
	}

	if async, ok := webAppAccelerationLifecycleAsync(state, message, fallback); ok {
		async.UpdatedAt = &now
		projection := servicemanager.ApplyAsyncOperation(status, async, c.log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: time.Minute,
		}, nil
	}

	servicemanager.ClearAsyncOperation(status)
	condition, requeue, message := webAppAccelerationCondition(state, message, fallback)
	status.Message = message
	status.Reason = string(condition)
	status.UpdatedAt = &now
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   requeue,
		RequeueDuration: time.Minute,
	}, nil
}

func (c *reviewedWebAppAccelerationServiceClient) failCreateOrUpdate(
	resource *waav1beta1.WebAppAcceleration,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil {
		status := &resource.Status.OsokStatus
		servicemanager.RecordErrorOpcRequestID(status, err)
		status.Message = err.Error()
		status.Reason = string(shared.Failed)
		now := metav1.Now()
		status.UpdatedAt = &now
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func buildWebAppAccelerationCreateBody(
	_ context.Context,
	resource *waav1beta1.WebAppAcceleration,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("WebAppAcceleration resource is nil")
	}
	return webAppAccelerationCreateDetailsFromSpec(resource.Spec)
}

func webAppAccelerationCreateDetailsFromSpec(
	spec waav1beta1.WebAppAccelerationSpec,
) (waasdk.CreateWebAppAccelerationDetails, error) {
	if err := validateWebAppAccelerationSpec(spec); err != nil {
		return nil, err
	}

	body := waasdk.CreateWebAppAccelerationLoadBalancerDetails{}
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		var err error
		body, err = webAppAccelerationCreateDetailsFromJSON(raw)
		if err != nil {
			return nil, err
		}
		if err := validateWebAppAccelerationCreateIdentityOverrides(spec, body); err != nil {
			return nil, err
		}
	}

	applyWebAppAccelerationCreateSpecDefaults(&body, spec)
	if err := validateWebAppAccelerationCreateDetails(body); err != nil {
		return nil, err
	}
	return body, nil
}

func webAppAccelerationCreateDetailsFromJSON(raw string) (waasdk.CreateWebAppAccelerationLoadBalancerDetails, error) {
	backendType, err := normalizedWebAppAccelerationBackendType("", raw)
	if err != nil {
		return waasdk.CreateWebAppAccelerationLoadBalancerDetails{}, err
	}
	if backendType != webAppAccelerationLoadBalancerBackendType {
		return waasdk.CreateWebAppAccelerationLoadBalancerDetails{}, fmt.Errorf("unsupported WebAppAcceleration backendType %q", backendType)
	}

	var details waasdk.CreateWebAppAccelerationLoadBalancerDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return waasdk.CreateWebAppAccelerationLoadBalancerDetails{}, fmt.Errorf("decode WebAppAcceleration jsonData: %w", err)
	}
	return details, nil
}

func validateWebAppAccelerationCreateIdentityOverrides(
	spec waav1beta1.WebAppAccelerationSpec,
	body waasdk.CreateWebAppAccelerationLoadBalancerDetails,
) error {
	var conflicts []string
	conflicts = appendWebAppAccelerationStringConflict(conflicts, "compartmentId", body.CompartmentId, spec.CompartmentId, true)
	conflicts = appendWebAppAccelerationStringConflict(conflicts, "webAppAccelerationPolicyId", body.WebAppAccelerationPolicyId, spec.WebAppAccelerationPolicyId, true)
	conflicts = appendWebAppAccelerationStringConflict(conflicts, "loadBalancerId", body.LoadBalancerId, spec.LoadBalancerId, false)
	conflicts = appendWebAppAccelerationStringConflict(conflicts, "displayName", body.DisplayName, spec.DisplayName, false)
	if backendType := strings.TrimSpace(spec.BackendType); backendType != "" &&
		!strings.EqualFold(backendType, webAppAccelerationLoadBalancerBackendType) {
		conflicts = append(conflicts, "backendType")
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("WebAppAcceleration jsonData identity conflicts with spec field(s): %s", strings.Join(conflicts, ", "))
}

func appendWebAppAccelerationStringConflict(
	conflicts []string,
	name string,
	current *string,
	desired string,
	requireDesired bool,
) []string {
	desired = strings.TrimSpace(desired)
	if current == nil || (desired == "" && !requireDesired) || webAppAccelerationStringPtrEqual(current, desired) {
		return conflicts
	}
	return append(conflicts, name)
}

func applyWebAppAccelerationCreateSpecDefaults(
	body *waasdk.CreateWebAppAccelerationLoadBalancerDetails,
	spec waav1beta1.WebAppAccelerationSpec,
) {
	applyWebAppAccelerationStringDefault(&body.CompartmentId, spec.CompartmentId, true)
	applyWebAppAccelerationStringDefault(&body.WebAppAccelerationPolicyId, spec.WebAppAccelerationPolicyId, true)
	applyWebAppAccelerationStringDefault(&body.LoadBalancerId, spec.LoadBalancerId, false)
	applyWebAppAccelerationStringDefault(&body.DisplayName, spec.DisplayName, false)
	applyWebAppAccelerationFreeformTagDefault(&body.FreeformTags, spec.FreeformTags)
	applyWebAppAccelerationDefinedTagDefault(&body.DefinedTags, spec.DefinedTags)
	applyWebAppAccelerationDefinedTagDefault(&body.SystemTags, spec.SystemTags)
}

func applyWebAppAccelerationStringDefault(target **string, value string, required bool) {
	value = strings.TrimSpace(value)
	if *target == nil && (value != "" || required) {
		*target = common.String(value)
	}
}

func applyWebAppAccelerationFreeformTagDefault(target *map[string]string, source map[string]string) {
	if *target == nil && source != nil {
		*target = maps.Clone(source)
	}
}

func applyWebAppAccelerationDefinedTagDefault(
	target *map[string]map[string]interface{},
	source map[string]shared.MapValue,
) {
	if *target == nil && source != nil {
		*target = *util.ConvertToOciDefinedTags(&source)
	}
}

func validateWebAppAccelerationCreateDetails(body waasdk.CreateWebAppAccelerationLoadBalancerDetails) error {
	var missing []string
	if strings.TrimSpace(webAppAccelerationStringValue(body.CompartmentId)) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(webAppAccelerationStringValue(body.WebAppAccelerationPolicyId)) == "" {
		missing = append(missing, "webAppAccelerationPolicyId")
	}
	if strings.TrimSpace(webAppAccelerationStringValue(body.LoadBalancerId)) == "" {
		missing = append(missing, "loadBalancerId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("WebAppAcceleration create body is missing required field(s): %s", strings.Join(missing, ", "))
}

func buildWebAppAccelerationUpdateBody(
	_ context.Context,
	resource *waav1beta1.WebAppAcceleration,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return waasdk.UpdateWebAppAccelerationDetails{}, false, fmt.Errorf("WebAppAcceleration resource is nil")
	}
	if err := validateWebAppAccelerationSpec(resource.Spec); err != nil {
		return waasdk.UpdateWebAppAccelerationDetails{}, false, err
	}

	current, ok := webAppAccelerationFromResponse(currentResponse)
	if !ok {
		return waasdk.UpdateWebAppAccelerationDetails{}, false, fmt.Errorf("current WebAppAcceleration response does not expose a load balancer body")
	}
	desired, err := webAppAccelerationUpdateDetailsFromSpec(resource.Spec)
	if err != nil {
		return waasdk.UpdateWebAppAccelerationDetails{}, false, err
	}

	updateDetails := waasdk.UpdateWebAppAccelerationDetails{}
	updateNeeded := false
	markWebAppAccelerationUpdateNeeded(&updateNeeded, applyWebAppAccelerationStringUpdate(&updateDetails.DisplayName, current.DisplayName, desired.DisplayName))
	markWebAppAccelerationUpdateNeeded(&updateNeeded, applyWebAppAccelerationStringUpdate(&updateDetails.WebAppAccelerationPolicyId, current.WebAppAccelerationPolicyId, desired.WebAppAccelerationPolicyId))
	markWebAppAccelerationUpdateNeeded(&updateNeeded, applyWebAppAccelerationFreeformTagUpdate(&updateDetails.FreeformTags, current.FreeformTags, desired.FreeformTags))
	markWebAppAccelerationUpdateNeeded(&updateNeeded, applyWebAppAccelerationDefinedTagUpdate(&updateDetails.DefinedTags, current.DefinedTags, desired.DefinedTags))
	markWebAppAccelerationUpdateNeeded(&updateNeeded, applyWebAppAccelerationDefinedTagUpdate(&updateDetails.SystemTags, current.SystemTags, desired.SystemTags))
	if !updateNeeded {
		return waasdk.UpdateWebAppAccelerationDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func markWebAppAccelerationUpdateNeeded(updateNeeded *bool, changed bool) {
	if changed {
		*updateNeeded = true
	}
}

func applyWebAppAccelerationStringUpdate(target **string, current *string, desired *string) bool {
	if desired == nil || webAppAccelerationStringPtrEqual(current, *desired) {
		return false
	}
	*target = common.String(strings.TrimSpace(*desired))
	return true
}

func applyWebAppAccelerationFreeformTagUpdate(
	target *map[string]string,
	current map[string]string,
	desired map[string]string,
) bool {
	if desired == nil || maps.Equal(current, desired) {
		return false
	}
	*target = maps.Clone(desired)
	return true
}

func applyWebAppAccelerationDefinedTagUpdate(
	target *map[string]map[string]interface{},
	current map[string]map[string]interface{},
	desired map[string]map[string]interface{},
) bool {
	if desired == nil || reflect.DeepEqual(current, desired) {
		return false
	}
	*target = cloneWebAppAccelerationDefinedTags(desired)
	return true
}

func webAppAccelerationUpdateDetailsFromSpec(
	spec waav1beta1.WebAppAccelerationSpec,
) (waasdk.UpdateWebAppAccelerationDetails, error) {
	body := waasdk.UpdateWebAppAccelerationDetails{}
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return waasdk.UpdateWebAppAccelerationDetails{}, fmt.Errorf("decode WebAppAcceleration update jsonData: %w", err)
		}
	}

	applyWebAppAccelerationStringDefault(&body.DisplayName, spec.DisplayName, false)
	applyWebAppAccelerationStringDefault(&body.WebAppAccelerationPolicyId, spec.WebAppAccelerationPolicyId, true)
	applyWebAppAccelerationFreeformTagDefault(&body.FreeformTags, spec.FreeformTags)
	applyWebAppAccelerationDefinedTagDefault(&body.DefinedTags, spec.DefinedTags)
	applyWebAppAccelerationDefinedTagDefault(&body.SystemTags, spec.SystemTags)
	return body, nil
}

func validateWebAppAccelerationSpec(spec waav1beta1.WebAppAccelerationSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.WebAppAccelerationPolicyId) == "" {
		missing = append(missing, "webAppAccelerationPolicyId")
	}
	loadBalancerID, err := desiredWebAppAccelerationLoadBalancerID(spec)
	if err != nil {
		return err
	}
	if loadBalancerID == "" {
		missing = append(missing, "loadBalancerId")
	}
	if _, err := normalizedWebAppAccelerationBackendType(spec.BackendType, spec.JsonData); err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("WebAppAcceleration spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func normalizedWebAppAccelerationBackendType(specValue string, rawJSON string) (string, error) {
	if backendType := strings.TrimSpace(specValue); backendType != "" {
		mapped, ok := waasdk.GetMappingBackendTypeEnum(backendType)
		if !ok {
			return "", fmt.Errorf("unsupported WebAppAcceleration backendType %q", specValue)
		}
		return string(mapped), nil
	}

	if raw := strings.TrimSpace(rawJSON); raw != "" {
		var discriminator struct {
			BackendType string `json:"backendType"`
		}
		if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
			return "", fmt.Errorf("decode WebAppAcceleration jsonData backendType: %w", err)
		}
		if backendType := strings.TrimSpace(discriminator.BackendType); backendType != "" {
			mapped, ok := waasdk.GetMappingBackendTypeEnum(backendType)
			if !ok {
				return "", fmt.Errorf("unsupported WebAppAcceleration backendType %q", discriminator.BackendType)
			}
			return string(mapped), nil
		}
	}

	return webAppAccelerationLoadBalancerBackendType, nil
}

func desiredWebAppAccelerationLoadBalancerID(spec waav1beta1.WebAppAccelerationSpec) (string, error) {
	if loadBalancerID := strings.TrimSpace(spec.LoadBalancerId); loadBalancerID != "" {
		return loadBalancerID, nil
	}
	if strings.TrimSpace(spec.JsonData) == "" {
		return "", nil
	}
	body, err := webAppAccelerationCreateDetailsFromJSON(spec.JsonData)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(webAppAccelerationStringValue(body.LoadBalancerId)), nil
}

func validateWebAppAccelerationCreateOnlyDriftForResponse(
	resource *waav1beta1.WebAppAcceleration,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("WebAppAcceleration resource is nil")
	}
	current, ok := webAppAccelerationFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current WebAppAcceleration response does not expose a load balancer body")
	}
	return validateWebAppAccelerationCreateOnlyDrift(resource.Spec, current)
}

func validateWebAppAccelerationCreateOnlyDrift(
	spec waav1beta1.WebAppAccelerationSpec,
	current waasdk.WebAppAccelerationLoadBalancer,
) error {
	var drift []string
	if backendType := strings.TrimSpace(spec.BackendType); backendType != "" &&
		!strings.EqualFold(backendType, webAppAccelerationLoadBalancerBackendType) {
		drift = append(drift, "backendType")
	}
	loadBalancerID, err := desiredWebAppAccelerationLoadBalancerID(spec)
	if err != nil {
		return err
	}
	if !webAppAccelerationStringPtrEqual(current.LoadBalancerId, loadBalancerID) {
		drift = append(drift, "loadBalancerId")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("WebAppAcceleration create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func listWebAppAccelerationsAllPages(
	ctx context.Context,
	list func(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error),
	request waasdk.ListWebAppAccelerationsRequest,
) (waasdk.ListWebAppAccelerationsResponse, error) {
	var combined waasdk.ListWebAppAccelerationsResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return waasdk.ListWebAppAccelerationsResponse{}, conservativeWebAppAccelerationNotFoundError(err, "list")
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

func projectWebAppAccelerationStatus(resource *waav1beta1.WebAppAcceleration, response any) error {
	if resource == nil {
		return fmt.Errorf("WebAppAcceleration resource is nil")
	}
	current, ok := webAppAccelerationFromResponse(response)
	if !ok {
		return nil
	}
	payload, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("marshal WebAppAcceleration response body: %w", err)
	}
	if err := json.Unmarshal(payload, &resource.Status); err != nil {
		return fmt.Errorf("project WebAppAcceleration response body into status: %w", err)
	}
	resource.Status.JsonData = string(payload)
	return nil
}

func handleWebAppAccelerationDeleteError(resource *waav1beta1.WebAppAcceleration, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeWebAppAccelerationNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("WebAppAcceleration %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousWebAppAccelerationNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousWebAppAccelerationNotFoundError{message: message}
}

func isWebAppAccelerationUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func clearTrackedWebAppAccelerationIdentity(resource *waav1beta1.WebAppAcceleration) {
	if resource == nil {
		return
	}
	resource.Status = waav1beta1.WebAppAccelerationStatus{}
}

func webAppAccelerationFromResponse(response any) (waasdk.WebAppAccelerationLoadBalancer, bool) {
	if current, ok := webAppAccelerationFromOperationResponse(response); ok {
		return current, true
	}
	return webAppAccelerationFromModelResponse(response)
}

func webAppAccelerationFromOperationResponse(response any) (waasdk.WebAppAccelerationLoadBalancer, bool) {
	switch current := response.(type) {
	case waasdk.CreateWebAppAccelerationResponse:
		return webAppAccelerationFromResponse(current.WebAppAcceleration)
	case *waasdk.CreateWebAppAccelerationResponse:
		if current == nil {
			return waasdk.WebAppAccelerationLoadBalancer{}, false
		}
		return webAppAccelerationFromResponse(current.WebAppAcceleration)
	case waasdk.GetWebAppAccelerationResponse:
		return webAppAccelerationFromResponse(current.WebAppAcceleration)
	case *waasdk.GetWebAppAccelerationResponse:
		if current == nil {
			return waasdk.WebAppAccelerationLoadBalancer{}, false
		}
		return webAppAccelerationFromResponse(current.WebAppAcceleration)
	default:
		return waasdk.WebAppAccelerationLoadBalancer{}, false
	}
}

func webAppAccelerationFromModelResponse(response any) (waasdk.WebAppAccelerationLoadBalancer, bool) {
	switch current := response.(type) {
	case waasdk.WebAppAccelerationLoadBalancer:
		return current, true
	case *waasdk.WebAppAccelerationLoadBalancer:
		if current == nil {
			return waasdk.WebAppAccelerationLoadBalancer{}, false
		}
		return *current, true
	case waasdk.WebAppAccelerationLoadBalancerSummary:
		return webAppAccelerationFromSummary(current), true
	case *waasdk.WebAppAccelerationLoadBalancerSummary:
		if current == nil {
			return waasdk.WebAppAccelerationLoadBalancer{}, false
		}
		return webAppAccelerationFromSummary(*current), true
	default:
		return waasdk.WebAppAccelerationLoadBalancer{}, false
	}
}

func webAppAccelerationFromSummary(summary waasdk.WebAppAccelerationLoadBalancerSummary) waasdk.WebAppAccelerationLoadBalancer {
	return waasdk.WebAppAccelerationLoadBalancer{
		Id:                         summary.Id,
		DisplayName:                summary.DisplayName,
		CompartmentId:              summary.CompartmentId,
		WebAppAccelerationPolicyId: summary.WebAppAccelerationPolicyId,
		TimeCreated:                summary.TimeCreated,
		FreeformTags:               maps.Clone(summary.FreeformTags),
		DefinedTags:                cloneWebAppAccelerationDefinedTags(summary.DefinedTags),
		SystemTags:                 cloneWebAppAccelerationDefinedTags(summary.SystemTags),
		LoadBalancerId:             summary.LoadBalancerId,
		TimeUpdated:                summary.TimeUpdated,
		LifecycleDetails:           summary.LifecycleDetails,
		LifecycleState:             summary.LifecycleState,
	}
}

func webAppAccelerationMatchesSpec(
	spec waav1beta1.WebAppAccelerationSpec,
	current waasdk.WebAppAccelerationLoadBalancer,
) (bool, error) {
	if !webAppAccelerationStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		return false, nil
	}
	if !webAppAccelerationStringPtrEqual(current.WebAppAccelerationPolicyId, spec.WebAppAccelerationPolicyId) {
		return false, nil
	}
	loadBalancerID, err := desiredWebAppAccelerationLoadBalancerID(spec)
	if err != nil {
		return false, err
	}
	if !webAppAccelerationStringPtrEqual(current.LoadBalancerId, loadBalancerID) {
		return false, nil
	}
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" &&
		!webAppAccelerationStringPtrEqual(current.DisplayName, displayName) {
		return false, nil
	}
	if backendType := strings.TrimSpace(spec.BackendType); backendType != "" &&
		!strings.EqualFold(backendType, webAppAccelerationLoadBalancerBackendType) {
		return false, nil
	}
	return true, nil
}

func webAppAccelerationLifecyclePending(state waasdk.WebAppAccelerationLifecycleStateEnum) bool {
	switch state {
	case waasdk.WebAppAccelerationLifecycleStateCreating,
		waasdk.WebAppAccelerationLifecycleStateUpdating,
		waasdk.WebAppAccelerationLifecycleStateDeleting:
		return true
	default:
		return false
	}
}

func webAppAccelerationIsDeleted(state waasdk.WebAppAccelerationLifecycleStateEnum) bool {
	return state == waasdk.WebAppAccelerationLifecycleStateDeleted
}

func webAppAccelerationLifecycleAsync(
	state string,
	message string,
	fallback shared.OSOKConditionType,
) (*shared.OSOKAsyncOperation, bool) {
	switch state {
	case string(waasdk.WebAppAccelerationLifecycleStateCreating):
		return newWebAppAccelerationLifecycleAsync(message, state, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending), true
	case string(waasdk.WebAppAccelerationLifecycleStateUpdating):
		return newWebAppAccelerationLifecycleAsync(message, state, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending), true
	case string(waasdk.WebAppAccelerationLifecycleStateDeleting):
		return newWebAppAccelerationLifecycleAsync(message, state, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending), true
	case string(waasdk.WebAppAccelerationLifecycleStateDeleted):
		return newWebAppAccelerationLifecycleAsync(message, state, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded), true
	case string(waasdk.WebAppAccelerationLifecycleStateFailed):
		phase := shared.OSOKAsyncPhase("")
		switch fallback {
		case shared.Provisioning:
			phase = shared.OSOKAsyncPhaseCreate
		case shared.Updating:
			phase = shared.OSOKAsyncPhaseUpdate
		case shared.Terminating:
			phase = shared.OSOKAsyncPhaseDelete
		}
		if phase != "" {
			return newWebAppAccelerationLifecycleAsync(message, state, phase, shared.OSOKAsyncClassFailed), true
		}
	}
	return nil, false
}

func newWebAppAccelerationLifecycleAsync(
	message string,
	rawStatus string,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
) *shared.OSOKAsyncOperation {
	if strings.TrimSpace(message) == "" {
		message = servicemanager.ProjectAsyncCondition(class, phase).DefaultMessage
	}
	return &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       rawStatus,
		NormalizedClass: class,
		Message:         message,
	}
}

func webAppAccelerationCondition(
	state string,
	message string,
	fallback shared.OSOKConditionType,
) (shared.OSOKConditionType, bool, string) {
	if strings.TrimSpace(message) == "" {
		message = "OCI resource is active"
	}
	switch state {
	case "", string(waasdk.WebAppAccelerationLifecycleStateActive):
		return shared.Active, false, message
	case string(waasdk.WebAppAccelerationLifecycleStateFailed):
		return shared.Failed, false, message
	default:
		if fallback == shared.Provisioning || fallback == shared.Updating || fallback == shared.Terminating {
			return fallback, true, message
		}
		return shared.Failed, false, fmt.Sprintf("WebAppAcceleration lifecycle state %q is not modeled: %s", state, message)
	}
}

func webAppAccelerationStatusID(resource *waav1beta1.WebAppAcceleration) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func webAppAccelerationResourceID(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) string {
	if id := strings.TrimSpace(webAppAccelerationStringValue(current.Id)); id != "" {
		return id
	}
	return webAppAccelerationStatusID(resource)
}

func webAppAccelerationCompartmentNeedsChange(
	resource *waav1beta1.WebAppAcceleration,
	current waasdk.WebAppAccelerationLoadBalancer,
) bool {
	if resource == nil {
		return false
	}
	desired := strings.TrimSpace(resource.Spec.CompartmentId)
	observed := strings.TrimSpace(webAppAccelerationStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func webAppAccelerationHasPendingDelete(resource *waav1beta1.WebAppAcceleration) bool {
	_, ok := webAppAccelerationPendingDelete(resource)
	return ok
}

func webAppAccelerationPendingDelete(resource *waav1beta1.WebAppAcceleration) (shared.OSOKAsyncOperation, bool) {
	return webAppAccelerationPendingAsync(resource, shared.OSOKAsyncPhaseDelete, "")
}

func webAppAccelerationPendingWrite(resource *waav1beta1.WebAppAcceleration) (shared.OSOKAsyncOperation, bool) {
	if pending, ok := webAppAccelerationPendingCreate(resource); ok {
		return pending, true
	}
	return webAppAccelerationPendingUpdate(resource)
}

func webAppAccelerationPendingWriteBeforeDelete(resource *waav1beta1.WebAppAcceleration) (shared.OSOKAsyncOperation, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return shared.OSOKAsyncOperation{}, false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return shared.OSOKAsyncOperation{}, false
	}
	rawOperationType := strings.TrimSpace(current.RawOperationType)
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate:
		if rawOperationType == "" || rawOperationType == string(waasdk.WorkRequestOperationTypeCreateWebAppAcceleration) {
			return *current.DeepCopy(), true
		}
	case shared.OSOKAsyncPhaseUpdate:
		switch rawOperationType {
		case "", string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration), string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration):
			return *current.DeepCopy(), true
		}
	}
	return shared.OSOKAsyncOperation{}, false
}

func webAppAccelerationPendingCreate(resource *waav1beta1.WebAppAcceleration) (shared.OSOKAsyncOperation, bool) {
	return webAppAccelerationPendingAsync(
		resource,
		shared.OSOKAsyncPhaseCreate,
		string(waasdk.WorkRequestOperationTypeCreateWebAppAcceleration),
	)
}

func webAppAccelerationPendingUpdate(resource *waav1beta1.WebAppAcceleration) (shared.OSOKAsyncOperation, bool) {
	return webAppAccelerationPendingAsync(
		resource,
		shared.OSOKAsyncPhaseUpdate,
		string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration),
	)
}

func webAppAccelerationHasPendingMove(resource *waav1beta1.WebAppAcceleration) bool {
	_, ok := webAppAccelerationPendingMove(resource)
	return ok
}

func webAppAccelerationPendingMove(resource *waav1beta1.WebAppAcceleration) (shared.OSOKAsyncOperation, bool) {
	return webAppAccelerationPendingAsync(
		resource,
		shared.OSOKAsyncPhaseUpdate,
		string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
	)
}

func webAppAccelerationPendingAsync(
	resource *waav1beta1.WebAppAcceleration,
	phase shared.OSOKAsyncPhase,
	rawOperationType string,
) (shared.OSOKAsyncOperation, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return shared.OSOKAsyncOperation{}, false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Phase != phase || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return shared.OSOKAsyncOperation{}, false
	}
	if rawOperationType != "" && strings.TrimSpace(current.RawOperationType) != rawOperationType {
		return shared.OSOKAsyncOperation{}, false
	}
	return *current.DeepCopy(), true
}

func webAppAccelerationDeletePendingMessage(workRequestID string) string {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return "WebAppAcceleration delete is pending"
	}
	return fmt.Sprintf("WebAppAcceleration delete work request %s is pending", workRequestID)
}

func webAppAccelerationCreatePendingMessage(workRequestID string) string {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return "WebAppAcceleration create is pending"
	}
	return fmt.Sprintf("WebAppAcceleration create work request %s is pending", workRequestID)
}

func webAppAccelerationUpdatePendingMessage(workRequestID string) string {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return "WebAppAcceleration update is pending"
	}
	return fmt.Sprintf("WebAppAcceleration update work request %s is pending", workRequestID)
}

func webAppAccelerationMovePendingMessage(workRequestID string) string {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return "WebAppAcceleration compartment move is pending"
	}
	return fmt.Sprintf("WebAppAcceleration compartment move work request %s is pending", workRequestID)
}

func webAppAccelerationWritePendingMessage(phase shared.OSOKAsyncPhase, workRequestID string) string {
	if phase == shared.OSOKAsyncPhaseCreate {
		return webAppAccelerationCreatePendingMessage(workRequestID)
	}
	return webAppAccelerationUpdatePendingMessage(workRequestID)
}

func webAppAccelerationPendingWriteBeforeDeleteMessage(current *shared.OSOKAsyncOperation) string {
	label := webAppAccelerationWorkRequestLabel(current)
	if current != nil && strings.TrimSpace(current.WorkRequestID) != "" {
		return fmt.Sprintf("WebAppAcceleration %s work request %s is pending; retaining finalizer before delete", label, current.WorkRequestID)
	}
	return fmt.Sprintf("WebAppAcceleration %s is pending; retaining finalizer before delete", label)
}

func webAppAccelerationWriteOperationType(phase shared.OSOKAsyncPhase) string {
	if phase == shared.OSOKAsyncPhaseCreate {
		return string(waasdk.WorkRequestOperationTypeCreateWebAppAcceleration)
	}
	return string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration)
}

func webAppAccelerationResponseID(response any) string {
	current, ok := webAppAccelerationFromResponse(response)
	if !ok {
		return ""
	}
	return webAppAccelerationStringValue(current.Id)
}

func webAppAccelerationStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(webAppAccelerationStringValue(current)) == strings.TrimSpace(desired)
}

func webAppAccelerationStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneWebAppAccelerationDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}
