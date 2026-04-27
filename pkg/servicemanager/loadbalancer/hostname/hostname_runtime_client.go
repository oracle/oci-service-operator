/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package hostname

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
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

const (
	hostnameLoadBalancerIDAnnotation = "loadbalancer.oracle.com/load-balancer-id"

	hostnameActiveMessage        = "OCI hostname is active"
	hostnameCreatePendingMessage = "OCI hostname create is in progress"
	hostnameUpdatePendingMessage = "OCI hostname update is in progress"
	hostnameDeletePendingMessage = "OCI hostname delete is in progress"

	hostnameCreatePendingState = "CREATE_ACCEPTED"
	hostnameUpdatePendingState = "UPDATE_ACCEPTED"
	hostnameDeletePendingState = "DELETE_ACCEPTED"

	hostnameRequeueDuration = time.Minute
)

type hostnameRuntimeOCIClient interface {
	CreateHostname(context.Context, loadbalancersdk.CreateHostnameRequest) (loadbalancersdk.CreateHostnameResponse, error)
	GetHostname(context.Context, loadbalancersdk.GetHostnameRequest) (loadbalancersdk.GetHostnameResponse, error)
	ListHostnames(context.Context, loadbalancersdk.ListHostnamesRequest) (loadbalancersdk.ListHostnamesResponse, error)
	UpdateHostname(context.Context, loadbalancersdk.UpdateHostnameRequest) (loadbalancersdk.UpdateHostnameResponse, error)
	DeleteHostname(context.Context, loadbalancersdk.DeleteHostnameRequest) (loadbalancersdk.DeleteHostnameResponse, error)
	GetWorkRequest(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

type hostnameWorkRequestClient interface {
	GetWorkRequest(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

type hostnameRuntimeHookOCIClient struct {
	create             func(context.Context, loadbalancersdk.CreateHostnameRequest) (loadbalancersdk.CreateHostnameResponse, error)
	get                func(context.Context, loadbalancersdk.GetHostnameRequest) (loadbalancersdk.GetHostnameResponse, error)
	list               func(context.Context, loadbalancersdk.ListHostnamesRequest) (loadbalancersdk.ListHostnamesResponse, error)
	update             func(context.Context, loadbalancersdk.UpdateHostnameRequest) (loadbalancersdk.UpdateHostnameResponse, error)
	delete             func(context.Context, loadbalancersdk.DeleteHostnameRequest) (loadbalancersdk.DeleteHostnameResponse, error)
	workRequest        hostnameWorkRequestClient
	workRequestInitErr error
}

type hostnameIdentity struct {
	loadBalancerID string
	name           string
}

type hostnameRuntimeClient struct {
	client hostnameRuntimeOCIClient
	log    loggerutil.OSOKLogger
}

var _ HostnameServiceClient = (*hostnameRuntimeClient)(nil)

var hostnameWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		string(loadbalancersdk.WorkRequestLifecycleStateInProgress),
	},
	SucceededStatusTokens: []string{string(loadbalancersdk.WorkRequestLifecycleStateSucceeded)},
	FailedStatusTokens:    []string{string(loadbalancersdk.WorkRequestLifecycleStateFailed)},
}

func init() {
	registerHostnameRuntimeHooksMutator(func(manager *HostnameServiceManager, hooks *HostnameRuntimeHooks) {
		applyHostnameRuntimeHooks(manager, hooks)
	})
}

func applyHostnameRuntimeHooks(manager *HostnameServiceManager, hooks *HostnameRuntimeHooks) {
	if hooks == nil {
		return
	}

	workRequestClient, workRequestInitErr := newHostnameWorkRequestClient(manager)
	runtimeClient := hostnameRuntimeHookOCIClient{
		create:             hooks.Create.Call,
		get:                hooks.Get.Call,
		list:               hooks.List.Call,
		update:             hooks.Update.Call,
		delete:             hooks.Delete.Call,
		workRequest:        workRequestClient,
		workRequestInitErr: workRequestInitErr,
	}
	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}

	hooks.Semantics = newHostnameRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(_ HostnameServiceClient) HostnameServiceClient {
		return newHostnameRuntimeClient(runtimeClient, log)
	})
}

func newHostnameWorkRequestClient(manager *HostnameServiceManager) (hostnameWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Hostname service manager is nil")
	}
	client, err := loadbalancersdk.NewLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newHostnameRuntimeClient(client hostnameRuntimeOCIClient, log loggerutil.OSOKLogger) HostnameServiceClient {
	return &hostnameRuntimeClient{
		client: client,
		log:    log,
	}
}

func newHostnameRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loadbalancer",
		FormalSlug:    "hostname",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "handwritten",
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
			ProvisioningStates: []string{hostnameCreatePendingState},
			UpdatingStates:     []string{hostnameUpdatePendingState},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{hostnameDeletePendingState},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"loadBalancerId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"hostname"},
			ForceNew:      []string{"loadBalancerId", "name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local hostname runtime"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local hostname runtime"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local hostname runtime"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local hostname runtime", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local hostname runtime", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local hostname runtime", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{{
			Phase:            "create/update/delete",
			MethodName:       "GetWorkRequest",
			RequestTypeName:  "GetWorkRequestRequest",
			ResponseTypeName: "GetWorkRequestResponse",
		}},
		Unsupported: []generatedruntime.UnsupportedSemantic{{
			Category:      "crd-shape",
			StopCondition: "loadBalancerId is supplied through metadata annotation because the v1beta1 Hostname spec has no parent load balancer field",
		}},
	}
}

func (c *hostnameRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *loadbalancerv1beta1.Hostname,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	identity, err := resolveHostnameIdentity(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if c.client == nil {
		err := fmt.Errorf("Hostname OCI client is not configured")
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}

	pendingPhase, workRequestID, pendingRawState, writePending := hostnamePendingWrite(resource)
	if pendingPhase == shared.OSOKAsyncPhaseDelete {
		err := fmt.Errorf("Hostname delete is still active during CreateOrUpdate")
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if writePending {
		response, handled, err := c.observePendingWorkRequest(ctx, resource, identity, pendingPhase, workRequestID)
		if handled || err != nil {
			return response, err
		}
	}

	current, err := c.lookupHostname(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if writePending {
		if current == nil || stringValue(current.Hostname) != strings.TrimSpace(resource.Spec.Hostname) {
			return c.markPending(resource, identity, pendingPhase, pendingRawState, workRequestID), nil
		}
		return c.markActive(resource, identity, *current), nil
	}
	if current == nil {
		return c.createHostname(ctx, resource, identity)
	}
	if stringValue(current.Hostname) != strings.TrimSpace(resource.Spec.Hostname) {
		return c.updateHostname(ctx, resource, identity)
	}
	return c.markActive(resource, identity, *current), nil
}

func (c *hostnameRuntimeClient) Delete(ctx context.Context, resource *loadbalancerv1beta1.Hostname) (bool, error) {
	identity, err := resolveHostnameDeleteIdentity(resource)
	if err != nil {
		return false, c.fail(resource, err)
	}
	if c.client == nil {
		return false, c.fail(resource, fmt.Errorf("Hostname OCI client is not configured"))
	}

	if hostnameDeleteAlreadyPending(resource) {
		workRequestID := strings.TrimSpace(hostnameCurrentWorkRequestID(resource))
		if workRequestID != "" {
			handled, err := c.observePendingDeleteWorkRequest(ctx, resource, identity, workRequestID)
			if handled || err != nil {
				return false, err
			}
		}
		current, err := c.lookupHostnameForDelete(ctx, identity)
		if err != nil {
			return false, c.fail(resource, err)
		}
		if current == nil {
			c.markDeleted(resource, "OCI hostname deleted")
			return true, nil
		}
		c.markTerminating(resource, identity, strings.TrimSpace(hostnameCurrentWorkRequestID(resource)))
		return false, nil
	}

	response, err := c.client.DeleteHostname(ctx, loadbalancersdk.DeleteHostnameRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		Name:           common.String(identity.name),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if hostnameIsDeleteNotFound(err) {
			c.markDeleted(resource, "OCI hostname deleted")
			return true, nil
		}
		return false, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.markTerminating(resource, identity, stringValue(response.OpcWorkRequestId))
	return false, nil
}

func (c *hostnameRuntimeClient) createHostname(
	ctx context.Context,
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
) (servicemanager.OSOKResponse, error) {
	response, err := c.client.CreateHostname(ctx, loadbalancersdk.CreateHostnameRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		CreateHostnameDetails: loadbalancersdk.CreateHostnameDetails{
			Name:     common.String(identity.name),
			Hostname: common.String(strings.TrimSpace(resource.Spec.Hostname)),
		},
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current, err := c.lookupHostname(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if current == nil || stringValue(current.Hostname) != strings.TrimSpace(resource.Spec.Hostname) {
		return c.markPending(resource, identity, shared.OSOKAsyncPhaseCreate, hostnameCreatePendingState, stringValue(response.OpcWorkRequestId)), nil
	}
	return c.markActive(resource, identity, *current), nil
}

func (c *hostnameRuntimeClient) updateHostname(
	ctx context.Context,
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
) (servicemanager.OSOKResponse, error) {
	response, err := c.client.UpdateHostname(ctx, loadbalancersdk.UpdateHostnameRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		Name:           common.String(identity.name),
		UpdateHostnameDetails: loadbalancersdk.UpdateHostnameDetails{
			Hostname: common.String(strings.TrimSpace(resource.Spec.Hostname)),
		},
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current, err := c.lookupHostname(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if current == nil || stringValue(current.Hostname) != strings.TrimSpace(resource.Spec.Hostname) {
		return c.markPending(resource, identity, shared.OSOKAsyncPhaseUpdate, hostnameUpdatePendingState, stringValue(response.OpcWorkRequestId)), nil
	}
	return c.markActive(resource, identity, *current), nil
}

func (c *hostnameRuntimeClient) observePendingWorkRequest(
	ctx context.Context,
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) (servicemanager.OSOKResponse, bool, error) {
	current, err := c.buildWorkRequestOperation(ctx, resource, phase, workRequestID)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, true, c.failPendingWorkRequestObservation(resource, err)
	}
	if current == nil {
		return servicemanager.OSOKResponse{}, false, nil
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.markWorkRequestOperation(resource, identity, current), true, nil
	case shared.OSOKAsyncClassSucceeded:
		return servicemanager.OSOKResponse{}, false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("Hostname %s work request %s finished with status %s", current.Phase, workRequestID, current.RawStatus)
		response := c.markFailedWorkRequestOperation(resource, identity, current, err)
		return response, true, err
	default:
		err := fmt.Errorf("Hostname %s work request %s projected unsupported async class %s", current.Phase, workRequestID, current.NormalizedClass)
		response := c.markFailedWorkRequestOperation(resource, identity, current, err)
		return response, true, err
	}
}

func (c *hostnameRuntimeClient) observePendingDeleteWorkRequest(
	ctx context.Context,
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	workRequestID string,
) (bool, error) {
	response, handled, err := c.observePendingWorkRequest(ctx, resource, identity, shared.OSOKAsyncPhaseDelete, workRequestID)
	_ = response
	return handled, err
}

func (c *hostnameRuntimeClient) buildWorkRequestOperation(
	ctx context.Context,
	resource *loadbalancerv1beta1.Hostname,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) (*shared.OSOKAsyncOperation, error) {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return nil, nil
	}

	response, err := c.client.GetWorkRequest(ctx, loadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return nil, err
	}

	workRequest := response.WorkRequest
	return servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, hostnameWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.LifecycleState),
		RawOperationType: stringValue(workRequest.Type),
		WorkRequestID:    firstNonEmptyTrim(stringValue(workRequest.Id), workRequestID),
		Message:          stringValue(workRequest.Message),
		FallbackPhase:    phase,
	})
}

func (c *hostnameRuntimeClient) lookupHostname(ctx context.Context, identity hostnameIdentity) (*loadbalancersdk.Hostname, error) {
	return c.lookupHostnameWithNotFound(ctx, identity, hostnameIsReadNotFound)
}

func (c *hostnameRuntimeClient) lookupHostnameForDelete(ctx context.Context, identity hostnameIdentity) (*loadbalancersdk.Hostname, error) {
	return c.lookupHostnameWithNotFound(ctx, identity, hostnameIsDeleteNotFound)
}

func (c *hostnameRuntimeClient) lookupHostnameWithNotFound(
	ctx context.Context,
	identity hostnameIdentity,
	isNotFound func(error) bool,
) (*loadbalancersdk.Hostname, error) {
	response, err := c.client.GetHostname(ctx, loadbalancersdk.GetHostnameRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
		Name:           common.String(identity.name),
	})
	if err == nil {
		current := response.Hostname
		return &current, nil
	}
	if !isNotFound(err) {
		return nil, err
	}

	listResponse, err := c.client.ListHostnames(ctx, loadbalancersdk.ListHostnamesRequest{
		LoadBalancerId: common.String(identity.loadBalancerID),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, item := range listResponse.Items {
		if stringValue(item.Name) == identity.name {
			current := item
			return &current, nil
		}
	}
	return nil, nil
}

func resolveHostnameIdentity(resource *loadbalancerv1beta1.Hostname) (hostnameIdentity, error) {
	if resource == nil {
		return hostnameIdentity{}, fmt.Errorf("Hostname resource is nil")
	}
	normalizeHostnameSpec(resource)
	if resource.Spec.Name == "" {
		return hostnameIdentity{}, fmt.Errorf("Hostname spec.name is required")
	}
	if resource.Spec.Hostname == "" {
		return hostnameIdentity{}, fmt.Errorf("Hostname spec.hostname is required")
	}

	statusName := strings.TrimSpace(resource.Status.Name)
	if statusName != "" && statusName != resource.Spec.Name {
		return hostnameIdentity{}, fmt.Errorf("Hostname formal semantics require replacement when name changes")
	}

	trackedLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationLoadBalancerID := hostnameAnnotationLoadBalancerID(resource)
	if trackedLoadBalancerID != "" &&
		annotationLoadBalancerID != "" &&
		trackedLoadBalancerID != annotationLoadBalancerID {
		return hostnameIdentity{}, fmt.Errorf("Hostname formal semantics require replacement when loadBalancerId changes")
	}

	loadBalancerID := firstNonEmptyTrim(trackedLoadBalancerID, annotationLoadBalancerID)
	if loadBalancerID == "" {
		return hostnameIdentity{}, fmt.Errorf("Hostname metadata annotation %q is required because the CRD has no spec loadBalancerId field", hostnameLoadBalancerIDAnnotation)
	}

	return hostnameIdentity{
		loadBalancerID: loadBalancerID,
		name:           firstNonEmptyTrim(statusName, resource.Spec.Name),
	}, nil
}

func resolveHostnameDeleteIdentity(resource *loadbalancerv1beta1.Hostname) (hostnameIdentity, error) {
	if resource == nil {
		return hostnameIdentity{}, fmt.Errorf("Hostname resource is nil")
	}
	trackedLoadBalancerID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	trackedName := strings.TrimSpace(resource.Status.Name)
	if trackedLoadBalancerID != "" && trackedName != "" {
		return hostnameIdentity{loadBalancerID: trackedLoadBalancerID, name: trackedName}, nil
	}

	normalizeHostnameSpec(resource)
	loadBalancerID := firstNonEmptyTrim(trackedLoadBalancerID, hostnameAnnotationLoadBalancerID(resource))
	name := firstNonEmptyTrim(trackedName, resource.Spec.Name)
	if loadBalancerID == "" {
		return hostnameIdentity{}, fmt.Errorf("Hostname metadata annotation %q is required because the CRD has no spec loadBalancerId field", hostnameLoadBalancerIDAnnotation)
	}
	if name == "" {
		return hostnameIdentity{}, fmt.Errorf("Hostname spec.name is required")
	}
	return hostnameIdentity{loadBalancerID: loadBalancerID, name: name}, nil
}

func normalizeHostnameSpec(resource *loadbalancerv1beta1.Hostname) {
	if resource == nil {
		return
	}
	resource.Spec.Name = strings.TrimSpace(resource.Spec.Name)
	resource.Spec.Hostname = strings.TrimSpace(resource.Spec.Hostname)
}

func hostnameAnnotationLoadBalancerID(resource *loadbalancerv1beta1.Hostname) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.GetAnnotations()[hostnameLoadBalancerIDAnnotation])
}

func (c *hostnameRuntimeClient) markActive(
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	current loadbalancersdk.Hostname,
) servicemanager.OSOKResponse {
	recordHostnamePathIdentity(resource, identity)
	resource.Status.Hostname = stringValue(current.Hostname)

	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = hostnameActiveMessage
	status.Reason = string(shared.Active)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", hostnameActiveMessage, c.log)

	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *hostnameRuntimeClient) markPending(
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	phase shared.OSOKAsyncPhase,
	rawState string,
	workRequestID string,
) servicemanager.OSOKResponse {
	recordHostnamePathIdentity(resource, identity)

	message := hostnameCreatePendingMessage
	if phase == shared.OSOKAsyncPhaseUpdate {
		message = hostnameUpdatePendingMessage
	}
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		RawStatus:       strings.TrimSpace(rawState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    true,
		ShouldRequeue:   true,
		RequeueDuration: hostnameRequeueDuration,
	}
}

func (c *hostnameRuntimeClient) markTerminating(
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	workRequestID string,
) {
	recordHostnamePathIdentity(resource, identity)
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		RawStatus:       hostnameDeletePendingState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         hostnameDeletePendingMessage,
		UpdatedAt:       &now,
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *hostnameRuntimeClient) markWorkRequestOperation(
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	recordHostnamePathIdentity(resource, identity)
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: hostnameRequeueDuration,
	}
}

func (c *hostnameRuntimeClient) markFailedWorkRequestOperation(
	resource *loadbalancerv1beta1.Hostname,
	identity hostnameIdentity,
	current *shared.OSOKAsyncOperation,
	err error,
) servicemanager.OSOKResponse {
	if current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassFailed
	if err != nil {
		next.Message = err.Error()
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return c.markWorkRequestOperation(resource, identity, &next)
}

func (c *hostnameRuntimeClient) markDeleted(resource *loadbalancerv1beta1.Hostname, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *hostnameRuntimeClient) fail(resource *loadbalancerv1beta1.Hostname, err error) error {
	if err == nil || resource == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

func (c *hostnameRuntimeClient) failPendingWorkRequestObservation(resource *loadbalancerv1beta1.Hostname, err error) error {
	if err == nil || resource == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

func recordHostnamePathIdentity(resource *loadbalancerv1beta1.Hostname, identity hostnameIdentity) {
	if resource == nil {
		return
	}
	// OCI Hostname has no OCID of its own. Store the parent load balancer OCID
	// as the tracked path identity so update/delete retries do not depend on annotations.
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.loadBalancerID)
	resource.Status.Name = identity.name
}

func hostnamePendingWrite(resource *loadbalancerv1beta1.Hostname) (shared.OSOKAsyncPhase, string, string, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", "", "", false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return "", "", "", false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncPhaseDelete:
		return current.Phase, strings.TrimSpace(current.WorkRequestID), strings.TrimSpace(current.RawStatus), true
	default:
		return "", "", "", false
	}
}

func hostnameDeleteAlreadyPending(resource *loadbalancerv1beta1.Hostname) bool {
	phase, _, _, pending := hostnamePendingWrite(resource)
	return pending && phase == shared.OSOKAsyncPhaseDelete
}

func hostnameCurrentWorkRequestID(resource *loadbalancerv1beta1.Hostname) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	return resource.Status.OsokStatus.Async.Current.WorkRequestID
}

func hostnameIsReadNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func hostnameIsDeleteNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func (c hostnameRuntimeHookOCIClient) CreateHostname(ctx context.Context, request loadbalancersdk.CreateHostnameRequest) (loadbalancersdk.CreateHostnameResponse, error) {
	if c.create == nil {
		return loadbalancersdk.CreateHostnameResponse{}, fmt.Errorf("Hostname create hook is not configured")
	}
	return c.create(ctx, request)
}

func (c hostnameRuntimeHookOCIClient) GetHostname(ctx context.Context, request loadbalancersdk.GetHostnameRequest) (loadbalancersdk.GetHostnameResponse, error) {
	if c.get == nil {
		return loadbalancersdk.GetHostnameResponse{}, fmt.Errorf("Hostname get hook is not configured")
	}
	return c.get(ctx, request)
}

func (c hostnameRuntimeHookOCIClient) ListHostnames(ctx context.Context, request loadbalancersdk.ListHostnamesRequest) (loadbalancersdk.ListHostnamesResponse, error) {
	if c.list == nil {
		return loadbalancersdk.ListHostnamesResponse{}, fmt.Errorf("Hostname list hook is not configured")
	}
	return c.list(ctx, request)
}

func (c hostnameRuntimeHookOCIClient) UpdateHostname(ctx context.Context, request loadbalancersdk.UpdateHostnameRequest) (loadbalancersdk.UpdateHostnameResponse, error) {
	if c.update == nil {
		return loadbalancersdk.UpdateHostnameResponse{}, fmt.Errorf("Hostname update hook is not configured")
	}
	return c.update(ctx, request)
}

func (c hostnameRuntimeHookOCIClient) DeleteHostname(ctx context.Context, request loadbalancersdk.DeleteHostnameRequest) (loadbalancersdk.DeleteHostnameResponse, error) {
	if c.delete == nil {
		return loadbalancersdk.DeleteHostnameResponse{}, fmt.Errorf("Hostname delete hook is not configured")
	}
	return c.delete(ctx, request)
}

func (c hostnameRuntimeHookOCIClient) GetWorkRequest(ctx context.Context, request loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
	if c.workRequestInitErr != nil {
		return loadbalancersdk.GetWorkRequestResponse{}, fmt.Errorf("initialize Hostname OCI client: %w", c.workRequestInitErr)
	}
	if c.workRequest == nil {
		return loadbalancersdk.GetWorkRequestResponse{}, fmt.Errorf("Hostname work request client is not configured")
	}
	return c.workRequest.GetWorkRequest(ctx, request)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
