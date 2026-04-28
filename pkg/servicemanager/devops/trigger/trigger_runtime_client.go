/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package trigger

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
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

const triggerKind = "Trigger"

const triggerRequeueDuration = time.Minute

var triggerWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(devopssdk.OperationStatusAccepted),
		string(devopssdk.OperationStatusInProgress),
		string(devopssdk.OperationStatusCanceling),
		string(devopssdk.OperationStatusWaiting),
	},
	SucceededStatusTokens: []string{string(devopssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(devopssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(devopssdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(devopssdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(devopssdk.OperationTypeCreateTrigger)},
	UpdateActionTokens:    []string{string(devopssdk.OperationTypeUpdateTrigger)},
	DeleteActionTokens:    []string{string(devopssdk.OperationTypeDeleteTrigger)},
}

type triggerOCIClient interface {
	CreateTrigger(context.Context, devopssdk.CreateTriggerRequest) (devopssdk.CreateTriggerResponse, error)
	GetTrigger(context.Context, devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error)
	ListTriggers(context.Context, devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error)
	UpdateTrigger(context.Context, devopssdk.UpdateTriggerRequest) (devopssdk.UpdateTriggerResponse, error)
	DeleteTrigger(context.Context, devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error)
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

type triggerWorkRequestClient interface {
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

type triggerIdentity struct {
	projectID     string
	displayName   string
	triggerSource string
	connectionID  string
	repositoryID  string
}

type triggerRuntimeClient struct {
	delegate TriggerServiceClient
	hooks    TriggerRuntimeHooks
	log      loggerutil.OSOKLogger
}

var _ TriggerServiceClient = (*triggerRuntimeClient)(nil)

func init() {
	registerTriggerRuntimeHooksMutator(func(manager *TriggerServiceManager, hooks *TriggerRuntimeHooks) {
		workRequestClient, initErr := newTriggerWorkRequestClient(manager)
		applyTriggerRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newTriggerWorkRequestClient(manager *TriggerServiceManager) (triggerWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", triggerKind)
	}
	client, err := devopssdk.NewDevopsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTriggerRuntimeHooks(
	manager *TriggerServiceManager,
	hooks *TriggerRuntimeHooks,
	workRequestClient triggerWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newTriggerRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *devopsv1beta1.Trigger, _ string) (any, error) {
		return buildTriggerCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *devopsv1beta1.Trigger,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildTriggerUpdateBody(resource, currentResponse)
	}

	listAllCall := paginateTriggerListCall(hooks.List.Call)
	hooks.List.Call = listAllCall
	hooks.Identity = generatedruntime.IdentityHooks[*devopsv1beta1.Trigger]{
		Resolve: func(resource *devopsv1beta1.Trigger) (any, error) {
			return resolveTriggerIdentity(resource)
		},
		GuardExistingBeforeCreate: func(_ context.Context, resource *devopsv1beta1.Trigger) (generatedruntime.ExistingBeforeCreateDecision, error) {
			identity, err := resolveTriggerIdentity(resource)
			if err != nil {
				return generatedruntime.ExistingBeforeCreateDecisionFail, err
			}
			if !identity.hasStableListMatch() {
				return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
			}
			return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
		},
		LookupExisting: func(ctx context.Context, resource *devopsv1beta1.Trigger, identity any) (any, error) {
			resolved, ok := identity.(triggerIdentity)
			if !ok {
				return nil, fmt.Errorf("unexpected %s identity type %T", triggerKind, identity)
			}
			return lookupExistingTrigger(ctx, resource, resolved, listAllCall)
		},
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateTriggerCreateOnlyDrift
	hooks.Async.Adapter = triggerWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getTriggerWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveTriggerWorkRequestAction
	hooks.Async.RecoverResourceID = recoverTriggerIDFromWorkRequest
	hooks.DeleteHooks.HandleError = handleTriggerDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TriggerServiceClient) TriggerServiceClient {
		runtimeClient := &triggerRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func newTriggerServiceClientWithOCIClient(log loggerutil.OSOKLogger, client triggerOCIClient) TriggerServiceClient {
	hooks := newTriggerRuntimeHooksWithOCIClient(client)
	applyTriggerRuntimeHooks(&TriggerServiceManager{Log: log}, &hooks, client, nil)
	config := buildTriggerGeneratedRuntimeConfig(&TriggerServiceManager{Log: log}, hooks)
	delegate := defaultTriggerServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.Trigger](config),
	}
	return wrapTriggerGeneratedClient(hooks, delegate)
}

func newTriggerRuntimeHooksWithOCIClient(client triggerOCIClient) TriggerRuntimeHooks {
	return TriggerRuntimeHooks{
		Create: runtimeOperationHooks[devopssdk.CreateTriggerRequest, devopssdk.CreateTriggerResponse]{
			Fields: triggerCreateFields(),
			Call: func(ctx context.Context, request devopssdk.CreateTriggerRequest) (devopssdk.CreateTriggerResponse, error) {
				return client.CreateTrigger(ctx, request)
			},
		},
		Get: runtimeOperationHooks[devopssdk.GetTriggerRequest, devopssdk.GetTriggerResponse]{
			Fields: triggerGetFields(),
			Call: func(ctx context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
				return client.GetTrigger(ctx, request)
			},
		},
		List: runtimeOperationHooks[devopssdk.ListTriggersRequest, devopssdk.ListTriggersResponse]{
			Fields: triggerListFields(),
			Call: func(ctx context.Context, request devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error) {
				return client.ListTriggers(ctx, request)
			},
		},
		Update: runtimeOperationHooks[devopssdk.UpdateTriggerRequest, devopssdk.UpdateTriggerResponse]{
			Fields: triggerUpdateFields(),
			Call: func(ctx context.Context, request devopssdk.UpdateTriggerRequest) (devopssdk.UpdateTriggerResponse, error) {
				return client.UpdateTrigger(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[devopssdk.DeleteTriggerRequest, devopssdk.DeleteTriggerResponse]{
			Fields: triggerDeleteFields(),
			Call: func(ctx context.Context, request devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error) {
				return client.DeleteTrigger(ctx, request)
			},
		},
		WrapGeneratedClient: []func(TriggerServiceClient) TriggerServiceClient{},
	}
}

func (c *triggerRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is nil", triggerKind)
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s resource is nil", triggerKind)
	}

	identity, err := resolveTriggerIdentity(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if response, handled, err := c.resumeCurrentWorkRequest(ctx, resource, identity); handled {
		return response, err
	}

	current, found, err := c.resolveCurrent(ctx, resource, identity)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.create(ctx, resource, identity, req)
	}
	return c.reconcileExisting(ctx, resource, identity, current)
}

func (c *triggerRuntimeClient) reconcileExisting(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	identity triggerIdentity,
	current any,
) (servicemanager.OSOKResponse, error) {
	if err := projectTriggerResponse(resource, current); err != nil {
		return c.fail(resource, err)
	}

	if strings.EqualFold(stringValueFromMap(mustTriggerResponseValues(current), "lifecycleState"), string(devopssdk.TriggerLifecycleStateDeleting)) {
		return c.markTerminating(resource, current), nil
	}
	if err := validateTriggerCreateOnlyDrift(resource, current); err != nil {
		return c.fail(resource, err)
	}

	updateDetails, updateNeeded, err := buildTriggerUpdateBody(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if updateNeeded {
		return c.update(ctx, resource, identity, updateDetails)
	}
	return c.finishWithResponse(resource, current, shared.Active)
}

func (c *triggerRuntimeClient) Delete(ctx context.Context, resource *devopsv1beta1.Trigger) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("%s runtime delete client is not configured", triggerKind)
	}
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", triggerKind)
	}
	if workRequestID, phase := currentTriggerWorkRequest(resource); workRequestID != "" {
		switch phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			return c.resumeCreateOrUpdateWorkRequestForDelete(ctx, resource, workRequestID, phase)
		case shared.OSOKAsyncPhaseDelete:
			return c.delegate.Delete(ctx, resource)
		default:
			return false, fmt.Errorf("%s delete cannot resume unsupported work request phase %q", triggerKind, phase)
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *triggerRuntimeClient) resumeCurrentWorkRequest(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	identity triggerIdentity,
) (servicemanager.OSOKResponse, bool, error) {
	workRequestID, phase := currentTriggerWorkRequest(resource)
	if workRequestID == "" {
		return servicemanager.OSOKResponse{}, false, nil
	}

	switch phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		response, err := c.resumeWorkRequest(ctx, resource, identity, workRequestID, phase)
		return response, true, err
	case shared.OSOKAsyncPhaseDelete:
		response, err := c.fail(resource, fmt.Errorf("%s delete work request %s is still active during CreateOrUpdate", triggerKind, workRequestID))
		return response, true, err
	default:
		response, err := c.fail(resource, fmt.Errorf("%s work request %s has unsupported phase %q", triggerKind, workRequestID, phase))
		return response, true, err
	}
}

func (c *triggerRuntimeClient) resolveCurrent(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	identity triggerIdentity,
) (any, bool, error) {
	currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if currentID != "" {
		response, err := c.get(ctx, currentID)
		if err == nil {
			return response, true, nil
		}
		if !isTriggerReadNotFound(err) {
			return nil, false, err
		}
	}

	existing, err := lookupExistingTrigger(ctx, resource, identity, c.hooks.List.Call)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, nil
	}
	existingID := triggerIDFromResponse(existing)
	if existingID == "" {
		return existing, true, nil
	}
	response, err := c.get(ctx, existingID)
	if err != nil {
		return nil, false, err
	}
	return response, true, nil
}

func (c *triggerRuntimeClient) create(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	identity triggerIdentity,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	createDetails, err := buildTriggerCreateBody(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	response, err := c.hooks.Create.Call(ctx, devopssdk.CreateTriggerRequest{
		CreateTriggerDetails: createDetails,
		OpcRetryToken:        optionalString(triggerRetryToken(resource, req)),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	c.seedResponseRequestID(resource, response)
	if workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId)); workRequestID != "" {
		if err := projectTriggerResponse(resource, response); err != nil {
			return c.fail(resource, err)
		}
		c.markWorkRequest(resource, workRequestID, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "", nil)
		return c.resumeWorkRequest(ctx, resource, identity, workRequestID, shared.OSOKAsyncPhaseCreate)
	}

	followUp := any(response)
	if id := triggerIDFromResponse(response); id != "" {
		if refreshed, err := c.get(ctx, id); err == nil {
			followUp = refreshed
		}
	}
	return c.finishWithResponse(resource, followUp, shared.Provisioning)
}

func (c *triggerRuntimeClient) update(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	identity triggerIdentity,
	updateDetails devopssdk.UpdateTriggerDetails,
) (servicemanager.OSOKResponse, error) {
	currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if currentID == "" {
		return c.fail(resource, fmt.Errorf("%s update requires a tracked OCI identifier", triggerKind))
	}
	response, err := c.hooks.Update.Call(ctx, devopssdk.UpdateTriggerRequest{
		TriggerId:            common.String(currentID),
		UpdateTriggerDetails: updateDetails,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	c.seedResponseRequestID(resource, response)
	if workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId)); workRequestID != "" {
		if err := projectTriggerResponse(resource, response); err != nil {
			return c.fail(resource, err)
		}
		c.markWorkRequest(resource, workRequestID, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "", nil)
		return c.resumeWorkRequest(ctx, resource, identity, workRequestID, shared.OSOKAsyncPhaseUpdate)
	}

	followUp := any(response)
	if refreshed, err := c.get(ctx, currentID); err == nil {
		followUp = refreshed
	}
	return c.finishWithResponse(resource, followUp, shared.Updating)
}

func (c *triggerRuntimeClient) get(ctx context.Context, triggerID string) (devopssdk.GetTriggerResponse, error) {
	if c.hooks.Get.Call == nil {
		return devopssdk.GetTriggerResponse{}, fmt.Errorf("%s get operation is not configured", triggerKind)
	}
	return c.hooks.Get.Call(ctx, devopssdk.GetTriggerRequest{TriggerId: common.String(strings.TrimSpace(triggerID))})
}

func (c *triggerRuntimeClient) resumeWorkRequest(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	identity triggerIdentity,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (servicemanager.OSOKResponse, error) {
	if c.hooks.Async.GetWorkRequest == nil {
		return c.fail(resource, fmt.Errorf("%s work request polling is not configured", triggerKind))
	}
	workRequest, err := c.hooks.Async.GetWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, err)
	}
	current, err := buildTriggerAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return c.fail(resource, err)
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.markWorkRequest(resource, workRequestID, current.Phase, shared.OSOKAsyncClassPending, current.RawStatus, current.PercentComplete), nil
	case shared.OSOKAsyncClassSucceeded:
		return c.finishSucceededWorkRequest(ctx, resource, workRequest, current, workRequestID)
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		return c.fail(resource, fmt.Errorf("%s %s work request %s finished with status %s", triggerKind, current.Phase, workRequestID, current.RawStatus))
	default:
		return c.fail(resource, fmt.Errorf("%s work request %s projected unsupported async class %s", triggerKind, workRequestID, current.NormalizedClass))
	}
}

func (c *triggerRuntimeClient) finishSucceededWorkRequest(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (servicemanager.OSOKResponse, error) {
	resourceID, err := recoverTriggerIDFromWorkRequest(resource, workRequest, current.Phase)
	if err != nil {
		return c.fail(resource, err)
	}
	if resourceID == "" {
		resourceID = strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	if resourceID == "" {
		return c.fail(resource, fmt.Errorf("%s %s work request %s did not expose a trigger identifier", triggerKind, current.Phase, workRequestID))
	}
	response, err := c.get(ctx, resourceID)
	if err != nil {
		if current.Phase == shared.OSOKAsyncPhaseCreate && isTriggerReadNotFound(err) {
			return c.markWorkRequestReadbackPending(resource, current, workRequestID, resourceID), nil
		}
		return c.fail(resource, err)
	}
	return c.finishWithResponse(resource, response, fallbackConditionForTriggerPhase(current.Phase))
}

func (c *triggerRuntimeClient) resumeCreateOrUpdateWorkRequestForDelete(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	if c.hooks.Async.GetWorkRequest == nil {
		return false, fmt.Errorf("%s work request polling is not configured", triggerKind)
	}
	workRequest, err := c.hooks.Async.GetWorkRequest(ctx, workRequestID)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	current, err := buildTriggerAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("%s %s work request %s is still in progress; waiting before delete", triggerKind, current.Phase, workRequestID)
		c.markWorkRequestOperation(resource, current, shared.OSOKAsyncClassPending, message)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.deleteAfterSucceededWriteWorkRequest(ctx, resource, workRequest, current, workRequestID)
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("%s %s work request %s finished with status %s before delete", triggerKind, current.Phase, workRequestID, current.RawStatus)
		_, failErr := c.fail(resource, err)
		return false, failErr
	default:
		err := fmt.Errorf("%s %s work request %s projected unsupported async class %s before delete", triggerKind, current.Phase, workRequestID, current.NormalizedClass)
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
}

func (c *triggerRuntimeClient) deleteAfterSucceededWriteWorkRequest(
	ctx context.Context,
	resource *devopsv1beta1.Trigger,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID, err := recoverTriggerIDFromWorkRequest(resource, workRequest, current.Phase)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	response, err := c.get(ctx, resourceID)
	if err != nil {
		if isTriggerReadNotFound(err) {
			c.markWorkRequestReadbackPending(resource, current, workRequestID, resourceID)
			return false, nil
		}
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	if err := projectTriggerResponse(resource, response); err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.delegate.Delete(ctx, resource)
}

func (c *triggerRuntimeClient) markWorkRequestReadbackPending(
	resource *devopsv1beta1.Trigger,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) servicemanager.OSOKResponse {
	now := metav1.Now()
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = fmt.Sprintf(
		"%s %s work request %s succeeded; waiting for %s %s to become readable",
		triggerKind,
		current.Phase,
		strings.TrimSpace(workRequestID),
		triggerKind,
		strings.TrimSpace(resourceID),
	)
	next.UpdatedAt = &now

	if strings.TrimSpace(resourceID) != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: triggerRequeueDuration,
	}
}

func (c *triggerRuntimeClient) markWorkRequestOperation(
	resource *devopsv1beta1.Trigger,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	if current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	now := metav1.Now()
	next.UpdatedAt = &now
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: triggerRequeueDuration,
	}
}

func buildTriggerAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := triggerWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, triggerWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        string(current.OperationType),
		RawOperationType: string(current.OperationType),
		WorkRequestID:    stringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    phase,
	})
}

func (c *triggerRuntimeClient) finishWithResponse(
	resource *devopsv1beta1.Trigger,
	response any,
	fallback shared.OSOKConditionType,
) (servicemanager.OSOKResponse, error) {
	if err := projectTriggerResponse(resource, response); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		status.Ocid = shared.OCID(id)
	}
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	servicemanager.ClearAsyncOperation(status)

	condition, shouldRequeue := triggerConditionForLifecycle(resource.Status.LifecycleState, fallback)
	message := strings.TrimSpace(resource.Status.LifecycleDetails)
	if message == "" {
		message = strings.TrimSpace(resource.Status.DisplayName)
	}
	if message == "" {
		message = string(condition)
	}
	status.Message = message
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatusForTriggerCondition(condition), "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: triggerRequeueDuration,
	}, nil
}

func (c *triggerRuntimeClient) markTerminating(resource *devopsv1beta1.Trigger, response any) servicemanager.OSOKResponse {
	_ = projectTriggerResponse(resource, response)
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         "OCI delete is in progress",
		UpdatedAt:       &now,
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: triggerRequeueDuration,
	}
}

func (c *triggerRuntimeClient) markWorkRequest(
	resource *devopsv1beta1.Trigger,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	rawStatus string,
	percentComplete *float32,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            phase,
		WorkRequestID:    strings.TrimSpace(workRequestID),
		RawStatus:        strings.TrimSpace(rawStatus),
		NormalizedClass:  class,
		PercentComplete:  percentComplete,
		Message:          fmt.Sprintf("%s %s work request %s is %s", triggerKind, phase, workRequestID, rawStatus),
		UpdatedAt:        &now,
		RawOperationType: string(triggerOperationTypeForPhase(phase)),
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: triggerRequeueDuration,
	}
}

func (c *triggerRuntimeClient) seedResponseRequestID(resource *devopsv1beta1.Trigger, response any) {
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
}

func (c *triggerRuntimeClient) fail(resource *devopsv1beta1.Trigger, err error) (servicemanager.OSOKResponse, error) {
	if resource == nil || err == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func newTriggerRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "devops",
		FormalSlug:    "trigger",
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
			ActiveStates: []string{string(devopssdk.TriggerLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(devopssdk.TriggerLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"projectId", "displayName", "triggerSource", "connectionId", "repositoryId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"jsonData",
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
				"connectionId",
				"repositoryId",
			},
			ForceNew:      []string{"projectId", "triggerSource"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Trigger", Action: "CreateTrigger"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Trigger", Action: "UpdateTrigger"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Trigger", Action: "DeleteTrigger"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Trigger", Action: "GetTrigger"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func triggerCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateTriggerDetails", RequestName: "CreateTriggerDetails", Contribution: "body"},
	}
}

func triggerGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TriggerId", RequestName: "triggerId", Contribution: "path", PreferResourceID: true},
	}
}

func triggerListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func triggerUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TriggerId", RequestName: "triggerId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateTriggerDetails", RequestName: "UpdateTriggerDetails", Contribution: "body"},
	}
}

func triggerDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TriggerId", RequestName: "triggerId", Contribution: "path", PreferResourceID: true},
	}
}

func buildTriggerCreateBody(resource *devopsv1beta1.Trigger) (devopssdk.CreateTriggerDetails, error) {
	payload, err := desiredTriggerPayload(resource, true)
	if err != nil {
		return nil, err
	}
	if !payloadHasNonEmptyArray(payload, "actions") {
		return nil, fmt.Errorf("spec.jsonData.actions is required for %s create", triggerKind)
	}
	details, err := createTriggerDetailsFromPayload(payload)
	if err != nil {
		return nil, err
	}
	if details.GetProjectId() == nil || strings.TrimSpace(*details.GetProjectId()) == "" {
		return nil, fmt.Errorf("spec.projectId is required for %s create", triggerKind)
	}
	if len(details.GetActions()) == 0 {
		return nil, fmt.Errorf("spec.jsonData.actions is required for %s create", triggerKind)
	}
	return details, nil
}

func buildTriggerUpdateBody(resource *devopsv1beta1.Trigger, currentResponse any) (devopssdk.UpdateTriggerDetails, bool, error) {
	currentValues, err := triggerResponseValues(currentResponse)
	if err != nil {
		return nil, false, err
	}

	payload, err := desiredTriggerUpdatePayload(resource, currentValues)
	if err != nil {
		return nil, false, err
	}
	if !triggerUpdateNeeded(payload, currentValues) {
		return nil, false, nil
	}

	details, err := updateTriggerDetailsFromPayload(payload)
	if err != nil {
		return nil, false, err
	}
	return details, true, nil
}

func desiredTriggerUpdatePayload(resource *devopsv1beta1.Trigger, currentValues map[string]any) (map[string]any, error) {
	payload, err := desiredTriggerPayload(resource, false)
	if err != nil {
		return nil, err
	}
	delete(payload, "projectId")

	desiredSource := triggerSourceFromPayload(payload)
	currentSource := normalizeTriggerSource(stringValueFromMap(currentValues, "triggerSource"))
	switch {
	case desiredSource == "" && currentSource != "":
		payload["triggerSource"] = currentSource
	case desiredSource != "" && currentSource != "" && desiredSource != currentSource:
		return nil, fmt.Errorf("%s triggerSource drift requires replacement: desired %q, observed %q", triggerKind, desiredSource, currentSource)
	case desiredSource == "":
		return nil, fmt.Errorf("triggerSource is required to build %s update body", triggerKind)
	}
	return payload, nil
}

func triggerUpdateNeeded(payload map[string]any, currentValues map[string]any) bool {
	for _, field := range triggerUpdateCompareFields() {
		desired, ok := payload[field]
		if !ok || !meaningfulTriggerValue(desired) {
			continue
		}
		current, currentOK := currentValues[field]
		if !currentOK || !triggerJSONValuesEqual(desired, current) {
			return true
		}
	}
	return false
}

func triggerUpdateCompareFields() []string {
	return []string{
		"displayName",
		"description",
		"actions",
		"freeformTags",
		"definedTags",
		"connectionId",
		"repositoryId",
	}
}

func validateTriggerCreateOnlyDrift(resource *devopsv1beta1.Trigger, currentResponse any) error {
	currentValues, err := triggerResponseValues(currentResponse)
	if err != nil {
		return err
	}
	desired, err := desiredTriggerPayload(resource, false)
	if err != nil {
		return err
	}

	for _, field := range []string{"projectId", "triggerSource"} {
		desiredValue, desiredOK := desired[field]
		currentValue, currentOK := currentValues[field]
		if !desiredOK || !currentOK || !meaningfulTriggerValue(desiredValue) || !meaningfulTriggerValue(currentValue) {
			continue
		}
		if !triggerJSONValuesEqual(desiredValue, currentValue) {
			return fmt.Errorf("%s %s drift requires replacement: desired %v, observed %v", triggerKind, field, desiredValue, currentValue)
		}
	}
	return nil
}

func currentTriggerWorkRequest(resource *devopsv1beta1.Trigger) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || strings.TrimSpace(current.WorkRequestID) == "" {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func projectTriggerResponse(resource *devopsv1beta1.Trigger, response any) error {
	body := triggerResponseBody(response)
	if body == nil {
		return nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal %s response body: %w", triggerKind, err)
	}
	if err := json.Unmarshal(payload, &resource.Status); err != nil {
		return fmt.Errorf("project %s response body into status: %w", triggerKind, err)
	}
	if id := triggerIDFromResponse(response); id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	return nil
}

func triggerIDFromResponse(response any) string {
	values, err := triggerResponseValues(response)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stringValueFromMap(values, "id"))
}

func mustTriggerResponseValues(response any) map[string]any {
	values, err := triggerResponseValues(response)
	if err != nil {
		return map[string]any{}
	}
	return values
}

func isTriggerReadNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func triggerRetryToken(resource *devopsv1beta1.Trigger, req ctrl.Request) string {
	if resource == nil {
		return ""
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return uid
	}
	namespace := strings.TrimSpace(resource.Namespace)
	if namespace == "" {
		namespace = strings.TrimSpace(req.Namespace)
	}
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func fallbackConditionForTriggerPhase(phase shared.OSOKAsyncPhase) shared.OSOKConditionType {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return shared.Provisioning
	case shared.OSOKAsyncPhaseUpdate:
		return shared.Updating
	default:
		return shared.Active
	}
}

func triggerConditionForLifecycle(lifecycle string, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool) {
	switch strings.ToUpper(strings.TrimSpace(lifecycle)) {
	case string(devopssdk.TriggerLifecycleStateActive):
		return shared.Active, false
	case string(devopssdk.TriggerLifecycleStateDeleting):
		return shared.Terminating, true
	}
	switch fallback {
	case shared.Provisioning, shared.Updating, shared.Terminating:
		return fallback, true
	case "":
		return shared.Active, false
	default:
		return fallback, false
	}
}

func conditionStatusForTriggerCondition(condition shared.OSOKConditionType) v1.ConditionStatus {
	if condition == shared.Failed {
		return v1.ConditionFalse
	}
	return v1.ConditionTrue
}

func triggerOperationTypeForPhase(phase shared.OSOKAsyncPhase) devopssdk.OperationTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return devopssdk.OperationTypeCreateTrigger
	case shared.OSOKAsyncPhaseUpdate:
		return devopssdk.OperationTypeUpdateTrigger
	case shared.OSOKAsyncPhaseDelete:
		return devopssdk.OperationTypeDeleteTrigger
	default:
		return ""
	}
}

func desiredTriggerPayload(resource *devopsv1beta1.Trigger, requireProject bool) (map[string]any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", triggerKind)
	}

	payload, err := triggerJSONDataPayload(resource.Spec.JsonData)
	if err != nil {
		return nil, err
	}
	if err := applyTriggerSpecPayload(resource, payload, requireProject); err != nil {
		return nil, err
	}
	if source := triggerSourceFromPayload(payload); source != "" {
		payload["triggerSource"] = source
	} else {
		return nil, fmt.Errorf("spec.triggerSource or jsonData.triggerSource is required for %s", triggerKind)
	}
	return payload, nil
}

func applyTriggerSpecPayload(resource *devopsv1beta1.Trigger, payload map[string]any, requireProject bool) error {
	if projectID := strings.TrimSpace(resource.Spec.ProjectId); projectID != "" {
		payload["projectId"] = projectID
	} else if requireProject && !meaningfulTriggerValue(payload["projectId"]) {
		return fmt.Errorf("spec.projectId is required for %s", triggerKind)
	}
	if source := normalizeTriggerSource(resource.Spec.TriggerSource); source != "" {
		payload["triggerSource"] = source
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		payload["displayName"] = displayName
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		payload["description"] = description
	}
	if err := applyTriggerTagPayload(resource, payload); err != nil {
		return err
	}
	applyTriggerSourceIdentityPayload(resource, payload)
	return nil
}

func applyTriggerTagPayload(resource *devopsv1beta1.Trigger, payload map[string]any) error {
	if resource.Spec.FreeformTags != nil {
		payload["freeformTags"] = resource.Spec.FreeformTags
	}
	if resource.Spec.DefinedTags != nil {
		definedTags, err := jsonCompatibleValue(resource.Spec.DefinedTags)
		if err != nil {
			return fmt.Errorf("convert spec.definedTags: %w", err)
		}
		payload["definedTags"] = definedTags
	}
	return nil
}

func applyTriggerSourceIdentityPayload(resource *devopsv1beta1.Trigger, payload map[string]any) {
	if connectionID := strings.TrimSpace(resource.Spec.ConnectionId); connectionID != "" {
		payload["connectionId"] = connectionID
	}
	if repositoryID := strings.TrimSpace(resource.Spec.RepositoryId); repositoryID != "" {
		payload["repositoryId"] = repositoryID
	}
}

func triggerJSONDataPayload(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("parse spec.jsonData: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("spec.jsonData must be a JSON object")
	}
	return payload, nil
}

func jsonCompatibleValue(value any) (any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func triggerSourceFromPayload(payload map[string]any) string {
	source := normalizeTriggerSource(stringValueFromMap(payload, "triggerSource"))
	if source != "" {
		return source
	}
	if strings.TrimSpace(stringValueFromMap(payload, "repositoryId")) != "" {
		return string(devopssdk.TriggerTriggerSourceDevopsCodeRepository)
	}
	return ""
}

func normalizeTriggerSource(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if enum, ok := devopssdk.GetMappingTriggerTriggerSourceEnum(value); ok {
		return string(enum)
	}
	return strings.ToUpper(value)
}

func createTriggerDetailsFromPayload(payload map[string]any) (devopssdk.CreateTriggerDetails, error) {
	source := triggerSourceFromPayload(payload)
	if source == "" {
		return nil, fmt.Errorf("triggerSource is required")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	switch source {
	case string(devopssdk.TriggerTriggerSourceGithub):
		var details devopssdk.CreateGithubTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceGitlab):
		var details devopssdk.CreateGitlabTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceGitlabServer):
		var details devopssdk.CreateGitlabServerTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceBitbucketCloud):
		var details devopssdk.CreateBitbucketCloudTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceBitbucketServer):
		var details devopssdk.CreateBitbucketServerTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceVbs):
		var details devopssdk.CreateVbsTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceDevopsCodeRepository):
		var details devopssdk.CreateDevopsCodeRepositoryTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported triggerSource %q", source)
	}
}

func updateTriggerDetailsFromPayload(payload map[string]any) (devopssdk.UpdateTriggerDetails, error) {
	source := triggerSourceFromPayload(payload)
	if source == "" {
		return nil, fmt.Errorf("triggerSource is required")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	switch source {
	case string(devopssdk.TriggerTriggerSourceGithub):
		var details devopssdk.UpdateGithubTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceGitlab):
		var details devopssdk.UpdateGitlabTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceGitlabServer):
		var details devopssdk.UpdateGitlabServerTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceBitbucketCloud):
		var details devopssdk.UpdateBitbucketCloudTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceBitbucketServer):
		var details devopssdk.UpdateBitbucketServerTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceVbs):
		var details devopssdk.UpdateVbsTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	case string(devopssdk.TriggerTriggerSourceDevopsCodeRepository):
		var details devopssdk.UpdateDevopsCodeRepositoryTriggerDetails
		err = json.Unmarshal(body, &details)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported triggerSource %q", source)
	}
}

func triggerResponseValues(response any) (map[string]any, error) {
	body := triggerResponseBody(response)
	if body == nil {
		return map[string]any{}, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal %s response body: %w", triggerKind, err)
	}
	values := map[string]any{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode %s response body: %w", triggerKind, err)
	}
	if source := triggerSourceFromPayload(values); source != "" {
		values["triggerSource"] = source
	}
	return values, nil
}

func triggerResponseBody(response any) any {
	switch current := response.(type) {
	case nil:
		return nil
	case *devopssdk.CreateTriggerResponse:
		if current == nil {
			return nil
		}
		return triggerResponseBody(*current)
	case *devopssdk.GetTriggerResponse:
		if current == nil {
			return nil
		}
		return triggerResponseBody(*current)
	case *devopssdk.UpdateTriggerResponse:
		if current == nil {
			return nil
		}
		return triggerResponseBody(*current)
	default:
		return triggerResponseBodyValue(response)
	}
}

func triggerResponseBodyValue(response any) any {
	switch current := response.(type) {
	case devopssdk.CreateTriggerResponse:
		return current.TriggerCreateResult
	case devopssdk.GetTriggerResponse:
		return current.Trigger
	case devopssdk.UpdateTriggerResponse:
		return current.Trigger
	case devopssdk.TriggerCreateResult:
		return current
	case devopssdk.TriggerSummary:
		return current
	default:
		return current
	}
}

func resolveTriggerIdentity(resource *devopsv1beta1.Trigger) (triggerIdentity, error) {
	payload, err := desiredTriggerPayload(resource, true)
	if err != nil {
		return triggerIdentity{}, err
	}
	return triggerIdentity{
		projectID:     strings.TrimSpace(stringValueFromMap(payload, "projectId")),
		displayName:   strings.TrimSpace(stringValueFromMap(payload, "displayName")),
		triggerSource: strings.TrimSpace(stringValueFromMap(payload, "triggerSource")),
		connectionID:  strings.TrimSpace(stringValueFromMap(payload, "connectionId")),
		repositoryID:  strings.TrimSpace(stringValueFromMap(payload, "repositoryId")),
	}, nil
}

func (i triggerIdentity) hasStableListMatch() bool {
	if i.projectID == "" || i.triggerSource == "" {
		return false
	}
	return i.displayName != "" || i.connectionID != "" || i.repositoryID != ""
}

func lookupExistingTrigger(
	ctx context.Context,
	_ *devopsv1beta1.Trigger,
	identity triggerIdentity,
	listCall func(context.Context, devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error),
) (any, error) {
	if !identity.hasStableListMatch() || listCall == nil {
		return nil, nil
	}

	response, err := listCall(ctx, devopssdk.ListTriggersRequest{
		ProjectId:   common.String(identity.projectID),
		DisplayName: optionalString(identity.displayName),
	})
	if err != nil {
		return nil, err
	}

	var matches []devopssdk.TriggerSummary
	for _, item := range response.Items {
		if item == nil {
			continue
		}
		matched, err := triggerSummaryMatchesIdentity(item, identity)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%s list returned multiple resources matching projectId/displayName/source identity", triggerKind)
	}
}

func triggerSummaryMatchesIdentity(summary devopssdk.TriggerSummary, identity triggerIdentity) (bool, error) {
	values, err := triggerResponseValues(summary)
	if err != nil {
		return false, err
	}
	return triggerValuesMatchIdentity(values, identity), nil
}

func triggerValuesMatchIdentity(values map[string]any, identity triggerIdentity) bool {
	if strings.TrimSpace(stringValueFromMap(values, "projectId")) != identity.projectID {
		return false
	}
	if normalizeTriggerSource(stringValueFromMap(values, "triggerSource")) != identity.triggerSource {
		return false
	}
	for _, field := range []struct {
		name string
		want string
	}{
		{name: "displayName", want: identity.displayName},
		{name: "connectionId", want: identity.connectionID},
		{name: "repositoryId", want: identity.repositoryID},
	} {
		if field.want == "" {
			continue
		}
		if strings.TrimSpace(stringValueFromMap(values, field.name)) != field.want {
			return false
		}
	}
	return true
}

func paginateTriggerListCall(
	listCall func(context.Context, devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error),
) func(context.Context, devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error) {
	if listCall == nil {
		return nil
	}
	return func(ctx context.Context, request devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error) {
		var combined devopssdk.ListTriggersResponse
		page := request.Page
		firstPage := true
		for {
			nextRequest := request
			nextRequest.Page = page
			response, err := listCall(ctx, nextRequest)
			if err != nil {
				return response, err
			}
			if firstPage {
				combined = response
				combined.Items = nil
				firstPage = false
			}
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			page = response.OpcNextPage
		}
	}
}

func getTriggerWorkRequest(
	ctx context.Context,
	client triggerWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", triggerKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", triggerKind)
	}

	response, err := client.GetWorkRequest(ctx, devopssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveTriggerWorkRequestAction(workRequest any) (string, error) {
	current, err := triggerWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func recoverTriggerIDFromWorkRequest(
	resource *devopsv1beta1.Trigger,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if resource != nil && strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)), nil
	}
	current, err := triggerWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	actionType := triggerWorkRequestActionForPhase(phase)
	for _, item := range current.Resources {
		identifier := strings.TrimSpace(stringValue(item.Identifier))
		if identifier == "" {
			continue
		}
		if item.ActionType == actionType {
			return identifier, nil
		}
	}
	for _, item := range current.Resources {
		identifier := strings.TrimSpace(stringValue(item.Identifier))
		if identifier != "" {
			return identifier, nil
		}
	}
	return "", fmt.Errorf("%s work request %s did not expose a trigger identifier", triggerKind, stringValue(current.Id))
}

func triggerWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) devopssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return devopssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return devopssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return devopssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func triggerWorkRequestFromAny(workRequest any) (devopssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case devopssdk.WorkRequest:
		return current, nil
	case *devopssdk.WorkRequest:
		if current == nil {
			return devopssdk.WorkRequest{}, fmt.Errorf("%s work request is nil", triggerKind)
		}
		return *current, nil
	default:
		return devopssdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", triggerKind, workRequest)
	}
}

func handleTriggerDeleteError(resource *devopsv1beta1.Trigger, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("%s delete received ambiguous not-found response: %s", triggerKind, err.Error())
	}
	return err
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func stringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func stringValueFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func payloadHasNonEmptyArray(payload map[string]any, key string) bool {
	value, ok := payload[key]
	if !ok {
		return false
	}
	switch current := value.(type) {
	case []any:
		return len(current) > 0
	case []devopssdk.TriggerAction:
		return len(current) > 0
	default:
		return meaningfulTriggerValue(current)
	}
}

func meaningfulTriggerValue(value any) bool {
	switch current := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(current) != ""
	case []any:
		return len(current) > 0
	case map[string]any:
		return len(current) > 0
	default:
		return true
	}
}

func triggerJSONValuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}

	var leftValue any
	var rightValue any
	if err := json.Unmarshal(leftPayload, &leftValue); err != nil {
		return string(leftPayload) == string(rightPayload)
	}
	if err := json.Unmarshal(rightPayload, &rightValue); err != nil {
		return string(leftPayload) == string(rightPayload)
	}
	leftValue = normalizeTriggerJSONValue(leftValue)
	rightValue = normalizeTriggerJSONValue(rightValue)
	normalizedLeft, leftErr := json.Marshal(leftValue)
	normalizedRight, rightErr := json.Marshal(rightValue)
	if leftErr != nil || rightErr != nil {
		return string(leftPayload) == string(rightPayload)
	}
	return string(normalizedLeft) == string(normalizedRight)
}

func normalizeTriggerJSONValue(value any) any {
	switch current := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(current))
		for key, raw := range current {
			normalized := normalizeTriggerJSONValue(raw)
			if normalized == nil {
				continue
			}
			out[key] = normalized
		}
		return out
	case []any:
		out := make([]any, 0, len(current))
		for _, raw := range current {
			out = append(out, normalizeTriggerJSONValue(raw))
		}
		return out
	default:
		return current
	}
}
