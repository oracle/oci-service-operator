/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasafeprivateendpoint

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
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
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

const dataSafePrivateEndpointKind = "DataSafePrivateEndpoint"

var dataSafePrivateEndpointWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(datasafesdk.WorkRequestStatusAccepted),
		string(datasafesdk.WorkRequestStatusInProgress),
		string(datasafesdk.WorkRequestStatusCanceling),
		string(datasafesdk.WorkRequestStatusSuspending),
	},
	SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(datasafesdk.WorkRequestStatusSuspended)},
	CreateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeCreatePrivateEndpoint),
		string(datasafesdk.WorkRequestResourceActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeUpdatePrivateEndpoint),
		string(datasafesdk.WorkRequestOperationTypeChangePrivateEndpointCompartment),
		string(datasafesdk.WorkRequestResourceActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeDeletePrivateEndpoint),
		string(datasafesdk.WorkRequestResourceActionTypeDeleted),
	},
}

type dataSafePrivateEndpointOCIClient interface {
	CreateDataSafePrivateEndpoint(context.Context, datasafesdk.CreateDataSafePrivateEndpointRequest) (datasafesdk.CreateDataSafePrivateEndpointResponse, error)
	GetDataSafePrivateEndpoint(context.Context, datasafesdk.GetDataSafePrivateEndpointRequest) (datasafesdk.GetDataSafePrivateEndpointResponse, error)
	ListDataSafePrivateEndpoints(context.Context, datasafesdk.ListDataSafePrivateEndpointsRequest) (datasafesdk.ListDataSafePrivateEndpointsResponse, error)
	UpdateDataSafePrivateEndpoint(context.Context, datasafesdk.UpdateDataSafePrivateEndpointRequest) (datasafesdk.UpdateDataSafePrivateEndpointResponse, error)
	DeleteDataSafePrivateEndpoint(context.Context, datasafesdk.DeleteDataSafePrivateEndpointRequest) (datasafesdk.DeleteDataSafePrivateEndpointResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type dataSafePrivateEndpointRuntimeClient struct {
	delegate DataSafePrivateEndpointServiceClient
	hooks    DataSafePrivateEndpointRuntimeHooks
	client   dataSafePrivateEndpointOCIClient
	initErr  error
	log      loggerutil.OSOKLogger
}

type dataSafePrivateEndpointAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e dataSafePrivateEndpointAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e dataSafePrivateEndpointAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var _ DataSafePrivateEndpointServiceClient = (*dataSafePrivateEndpointRuntimeClient)(nil)

func init() {
	registerDataSafePrivateEndpointRuntimeHooksMutator(func(manager *DataSafePrivateEndpointServiceManager, hooks *DataSafePrivateEndpointRuntimeHooks) {
		client, initErr := newDataSafePrivateEndpointOCIClient(manager)
		applyDataSafePrivateEndpointRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newDataSafePrivateEndpointOCIClient(manager *DataSafePrivateEndpointServiceManager) (dataSafePrivateEndpointOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", dataSafePrivateEndpointKind)
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyDataSafePrivateEndpointRuntimeHooks(
	manager *DataSafePrivateEndpointServiceManager,
	hooks *DataSafePrivateEndpointRuntimeHooks,
	client dataSafePrivateEndpointOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = dataSafePrivateEndpointRuntimeSemantics()
	hooks.BuildCreateBody = buildDataSafePrivateEndpointCreateBody
	hooks.BuildUpdateBody = buildDataSafePrivateEndpointUpdateBody
	hooks.List.Fields = dataSafePrivateEndpointListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDataSafePrivateEndpointsAllPages(hooks.List.Call)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedDataSafePrivateEndpointIdentity
	hooks.StatusHooks.ProjectStatus = projectDataSafePrivateEndpointStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDataSafePrivateEndpointCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleDataSafePrivateEndpointDeleteError
	hooks.Async.Adapter = dataSafePrivateEndpointWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDataSafePrivateEndpointWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveDataSafePrivateEndpointWorkRequestAction
	hooks.Async.RecoverResourceID = recoverDataSafePrivateEndpointIDFromWorkRequest
	hooks.Async.Message = dataSafePrivateEndpointWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DataSafePrivateEndpointServiceClient) DataSafePrivateEndpointServiceClient {
		runtimeClient := &dataSafePrivateEndpointRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
			client:   client,
			initErr:  initErr,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func dataSafePrivateEndpointRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "datasafeprivateendpoint",
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
			ProvisioningStates: []string{string(datasafesdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"displayName",
				"compartmentId",
				"vcnId",
				"subnetId",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"nsgIds",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"vcnId",
				"subnetId",
				"privateEndpointIp",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetDataSafePrivateEndpoint"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetDataSafePrivateEndpoint"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func newDataSafePrivateEndpointServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client dataSafePrivateEndpointOCIClient,
) DataSafePrivateEndpointServiceClient {
	manager := &DataSafePrivateEndpointServiceManager{Log: log}
	hooks := newDataSafePrivateEndpointRuntimeHooksWithOCIClient(client)
	applyDataSafePrivateEndpointRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultDataSafePrivateEndpointServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.DataSafePrivateEndpoint](
			buildDataSafePrivateEndpointGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDataSafePrivateEndpointGeneratedClient(hooks, delegate)
}

func newDataSafePrivateEndpointRuntimeHooksWithOCIClient(
	client dataSafePrivateEndpointOCIClient,
) DataSafePrivateEndpointRuntimeHooks {
	return DataSafePrivateEndpointRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.DataSafePrivateEndpoint]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.DataSafePrivateEndpoint]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.DataSafePrivateEndpoint]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.DataSafePrivateEndpoint]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.DataSafePrivateEndpoint]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.DataSafePrivateEndpoint]{},
		Create: runtimeOperationHooks[datasafesdk.CreateDataSafePrivateEndpointRequest, datasafesdk.CreateDataSafePrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateDataSafePrivateEndpointDetails", RequestName: "CreateDataSafePrivateEndpointDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request datasafesdk.CreateDataSafePrivateEndpointRequest) (datasafesdk.CreateDataSafePrivateEndpointResponse, error) {
				return client.CreateDataSafePrivateEndpoint(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetDataSafePrivateEndpointRequest, datasafesdk.GetDataSafePrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DataSafePrivateEndpointId", RequestName: "dataSafePrivateEndpointId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request datasafesdk.GetDataSafePrivateEndpointRequest) (datasafesdk.GetDataSafePrivateEndpointResponse, error) {
				return client.GetDataSafePrivateEndpoint(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListDataSafePrivateEndpointsRequest, datasafesdk.ListDataSafePrivateEndpointsResponse]{
			Fields: dataSafePrivateEndpointListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListDataSafePrivateEndpointsRequest) (datasafesdk.ListDataSafePrivateEndpointsResponse, error) {
				return client.ListDataSafePrivateEndpoints(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateDataSafePrivateEndpointRequest, datasafesdk.UpdateDataSafePrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DataSafePrivateEndpointId", RequestName: "dataSafePrivateEndpointId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateDataSafePrivateEndpointDetails", RequestName: "UpdateDataSafePrivateEndpointDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request datasafesdk.UpdateDataSafePrivateEndpointRequest) (datasafesdk.UpdateDataSafePrivateEndpointResponse, error) {
				return client.UpdateDataSafePrivateEndpoint(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteDataSafePrivateEndpointRequest, datasafesdk.DeleteDataSafePrivateEndpointResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DataSafePrivateEndpointId", RequestName: "dataSafePrivateEndpointId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request datasafesdk.DeleteDataSafePrivateEndpointRequest) (datasafesdk.DeleteDataSafePrivateEndpointResponse, error) {
				return client.DeleteDataSafePrivateEndpoint(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DataSafePrivateEndpointServiceClient) DataSafePrivateEndpointServiceClient{},
	}
}

func (c *dataSafePrivateEndpointRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", dataSafePrivateEndpointKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *dataSafePrivateEndpointRuntimeClient) Delete(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", dataSafePrivateEndpointKind)
	}
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", dataSafePrivateEndpointKind)
	}

	if handled, deleted, err := c.deleteTrackedWorkRequest(ctx, resource); handled {
		return deleted, err
	}
	if deleted, err := c.confirmBeforeInitialDelete(ctx, resource); err != nil || deleted {
		return deleted, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *dataSafePrivateEndpointRuntimeClient) deleteTrackedWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
) (bool, bool, error) {
	current := resource.Status.OsokStatus.Async.Current
	if current != nil &&
		current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.WorkRequestID != "" {
		switch current.Phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			deleted, err := c.waitForPendingWriteBeforeDelete(ctx, resource, current)
			return true, deleted, err
		case shared.OSOKAsyncPhaseDelete:
			deleted, err := c.waitForDeleteWorkRequestBeforeFinalizerRelease(ctx, resource, current)
			return true, deleted, err
		}
	}
	return false, false, nil
}

func (c *dataSafePrivateEndpointRuntimeClient) confirmBeforeInitialDelete(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
) (bool, error) {
	resourceID := currentDataSafePrivateEndpointID(resource)
	if resourceID == "" || c.hooks.Get.Call == nil {
		return false, nil
	}
	_, found, err := c.getDataSafePrivateEndpointForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if found {
		return false, nil
	}
	c.markDeleted(resource, "OCI resource deleted")
	return true, nil
}

func (c *dataSafePrivateEndpointRuntimeClient) waitForPendingWriteBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
) (bool, error) {
	workRequest, err := getDataSafePrivateEndpointWorkRequest(ctx, c.client, c.initErr, tracked.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := dataSafePrivateEndpointAsyncOperation(&resource.Status.OsokStatus, workRequest, tracked.Phase)
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
		return false, fmt.Errorf("%s %s work request %s finished with status %s", dataSafePrivateEndpointKind, current.Phase, tracked.WorkRequestID, current.RawStatus)
	}
}

func (c *dataSafePrivateEndpointRuntimeClient) deleteAfterSucceededWrite(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
	current *shared.OSOKAsyncOperation,
	workRequest datasafesdk.WorkRequest,
) (bool, error) {
	resourceID := currentDataSafePrivateEndpointID(resource)
	if resourceID == "" {
		resourceID = dataSafePrivateEndpointIDFromWorkRequest(workRequest, tracked.Phase)
	}
	if resourceID == "" {
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s %s work request %s did not expose a %s identifier", dataSafePrivateEndpointKind, tracked.Phase, tracked.WorkRequestID, dataSafePrivateEndpointKind)
	}
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	return c.deleteAfterWriteReadback(ctx, resource, tracked, current, resourceID)
}

func (c *dataSafePrivateEndpointRuntimeClient) waitForDeleteWorkRequestBeforeFinalizerRelease(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
) (bool, error) {
	workRequest, err := getDataSafePrivateEndpointWorkRequest(ctx, c.client, c.initErr, tracked.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := dataSafePrivateEndpointAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
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
		return false, fmt.Errorf("%s delete work request %s finished with status %s", dataSafePrivateEndpointKind, tracked.WorkRequestID, current.RawStatus)
	}
}

func (c *dataSafePrivateEndpointRuntimeClient) completeDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
	current *shared.OSOKAsyncOperation,
	workRequest datasafesdk.WorkRequest,
) (bool, error) {
	resourceID := currentDataSafePrivateEndpointID(resource)
	if resourceID == "" {
		resourceID = dataSafePrivateEndpointIDFromWorkRequest(workRequest, shared.OSOKAsyncPhaseDelete)
	}
	if resourceID == "" {
		c.markDeleted(resource, fmt.Sprintf("OCI %s delete work request completed", dataSafePrivateEndpointKind))
		return true, nil
	}
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)

	live, found, err := c.getDataSafePrivateEndpointForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if err := projectDataSafePrivateEndpointSDKStatus(resource, live); err != nil {
		return false, err
	}
	if dataSafePrivateEndpointDeleteLifecycleTerminal(live) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if dataSafePrivateEndpointDeleteLifecyclePending(live) {
		c.markDeleteWorkRequestPending(resource, current, tracked.WorkRequestID, resourceID)
		return false, nil
	}
	c.applyWorkRequest(resource, current)
	return false, fmt.Errorf("%s delete work request %s succeeded but %s %s is in lifecycle state %q", dataSafePrivateEndpointKind, tracked.WorkRequestID, dataSafePrivateEndpointKind, resourceID, live.LifecycleState)
}

func (c *dataSafePrivateEndpointRuntimeClient) deleteAfterWriteReadback(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	tracked *shared.OSOKAsyncOperation,
	current *shared.OSOKAsyncOperation,
	resourceID string,
) (bool, error) {
	live, found, err := c.getDataSafePrivateEndpointForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if !found || dataSafePrivateEndpointWriteLifecyclePending(live) {
		if found {
			_ = projectDataSafePrivateEndpointSDKStatus(resource, live)
		}
		c.markWriteReadbackPending(resource, current, tracked.WorkRequestID, resourceID)
		return false, nil
	}
	_ = projectDataSafePrivateEndpointSDKStatus(resource, live)
	resource.Status.OsokStatus.Async.Current = nil
	return c.delegate.Delete(ctx, resource)
}

func (c *dataSafePrivateEndpointRuntimeClient) getDataSafePrivateEndpointForDelete(
	ctx context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	resourceID string,
) (datasafesdk.DataSafePrivateEndpoint, bool, error) {
	if strings.TrimSpace(resourceID) == "" || c.hooks.Get.Call == nil {
		return datasafesdk.DataSafePrivateEndpoint{}, false, nil
	}
	response, err := c.hooks.Get.Call(ctx, datasafesdk.GetDataSafePrivateEndpointRequest{
		DataSafePrivateEndpointId: common.String(strings.TrimSpace(resourceID)),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if dataSafePrivateEndpointIsUnambiguousNotFound(err) {
			return datasafesdk.DataSafePrivateEndpoint{}, false, nil
		}
		return datasafesdk.DataSafePrivateEndpoint{}, false, handleDataSafePrivateEndpointDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response.DataSafePrivateEndpoint, true, nil
}

func (c *dataSafePrivateEndpointRuntimeClient) applyWorkRequest(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	current *shared.OSOKAsyncOperation,
) {
	if current == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *dataSafePrivateEndpointRuntimeClient) markWriteReadbackPending(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
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
		dataSafePrivateEndpointKind,
		current.Phase,
		strings.TrimSpace(workRequestID),
		dataSafePrivateEndpointKind,
		strings.TrimSpace(resourceID),
	)
	next.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func (c *dataSafePrivateEndpointRuntimeClient) markDeleteWorkRequestPending(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
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
		dataSafePrivateEndpointKind,
		strings.TrimSpace(workRequestID),
		dataSafePrivateEndpointKind,
		strings.TrimSpace(resourceID),
	)
	next.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func (c *dataSafePrivateEndpointRuntimeClient) markDeleted(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
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

func dataSafePrivateEndpointListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"spec.compartmentId", "status.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"spec.displayName", "status.displayName", "displayName"}},
		{FieldName: "VcnId", RequestName: "vcnId", Contribution: "query", LookupPaths: []string{"spec.vcnId", "status.vcnId", "vcnId"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
	}
}

func buildDataSafePrivateEndpointCreateBody(
	_ context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", dataSafePrivateEndpointKind)
	}
	if err := validateDataSafePrivateEndpointSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := datasafesdk.CreateDataSafePrivateEndpointDetails{
		DisplayName:   common.String(strings.TrimSpace(spec.DisplayName)),
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
		VcnId:         common.String(strings.TrimSpace(spec.VcnId)),
		SubnetId:      common.String(strings.TrimSpace(spec.SubnetId)),
	}
	if strings.TrimSpace(spec.PrivateEndpointIp) != "" {
		body.PrivateEndpointIp = common.String(strings.TrimSpace(spec.PrivateEndpointIp))
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if spec.NsgIds != nil {
		body.NsgIds = cloneDataSafePrivateEndpointStringSlice(spec.NsgIds)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = dataSafePrivateEndpointDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildDataSafePrivateEndpointUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", dataSafePrivateEndpointKind)
	}
	if err := validateDataSafePrivateEndpointSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := dataSafePrivateEndpointFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a %s body", dataSafePrivateEndpointKind, dataSafePrivateEndpointKind)
	}
	if err := validateDataSafePrivateEndpointCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}

	body, updateNeeded := dataSafePrivateEndpointMutableUpdateBody(resource.Spec, current)
	return body, updateNeeded, nil
}

func dataSafePrivateEndpointMutableUpdateBody(
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
	current datasafesdk.DataSafePrivateEndpoint,
) (datasafesdk.UpdateDataSafePrivateEndpointDetails, bool) {
	body := datasafesdk.UpdateDataSafePrivateEndpointDetails{}
	updateNeeded := false
	updateNeeded = setDataSafePrivateEndpointDisplayName(&body, current, spec) || updateNeeded
	updateNeeded = setDataSafePrivateEndpointDescription(&body, current, spec) || updateNeeded
	updateNeeded = setDataSafePrivateEndpointNsgIds(&body, current, spec) || updateNeeded
	updateNeeded = setDataSafePrivateEndpointFreeformTags(&body, current, spec) || updateNeeded
	updateNeeded = setDataSafePrivateEndpointDefinedTags(&body, current, spec) || updateNeeded
	return body, updateNeeded
}

func setDataSafePrivateEndpointDisplayName(
	body *datasafesdk.UpdateDataSafePrivateEndpointDetails,
	current datasafesdk.DataSafePrivateEndpoint,
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
) bool {
	if dataSafePrivateEndpointStringPtrEqual(current.DisplayName, spec.DisplayName) {
		return false
	}
	body.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	return true
}

func setDataSafePrivateEndpointDescription(
	body *datasafesdk.UpdateDataSafePrivateEndpointDetails,
	current datasafesdk.DataSafePrivateEndpoint,
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
) bool {
	if dataSafePrivateEndpointStringPtrEqual(current.Description, spec.Description) {
		return false
	}
	body.Description = common.String(spec.Description)
	return true
}

func setDataSafePrivateEndpointNsgIds(
	body *datasafesdk.UpdateDataSafePrivateEndpointDetails,
	current datasafesdk.DataSafePrivateEndpoint,
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
) bool {
	if spec.NsgIds == nil || dataSafePrivateEndpointStringSlicesEqual(spec.NsgIds, current.NsgIds) {
		return false
	}
	body.NsgIds = cloneDataSafePrivateEndpointStringSlice(spec.NsgIds)
	return true
}

func setDataSafePrivateEndpointFreeformTags(
	body *datasafesdk.UpdateDataSafePrivateEndpointDetails,
	current datasafesdk.DataSafePrivateEndpoint,
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	body.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func setDataSafePrivateEndpointDefinedTags(
	body *datasafesdk.UpdateDataSafePrivateEndpointDetails,
	current datasafesdk.DataSafePrivateEndpoint,
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := dataSafePrivateEndpointDefinedTagsFromSpec(spec.DefinedTags)
	if dataSafePrivateEndpointJSONEqual(desired, current.DefinedTags) {
		return false
	}
	body.DefinedTags = desired
	return true
}

func validateDataSafePrivateEndpointSpec(spec datasafev1beta1.DataSafePrivateEndpointSpec) error {
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
	return fmt.Errorf("%s spec is missing required field(s): %s", dataSafePrivateEndpointKind, strings.Join(missing, ", "))
}

func validateDataSafePrivateEndpointCreateOnlyDriftForResponse(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", dataSafePrivateEndpointKind)
	}
	current, ok := dataSafePrivateEndpointFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateDataSafePrivateEndpointCreateOnlyDrift(resource.Spec, current)
}

func validateDataSafePrivateEndpointCreateOnlyDrift(
	spec datasafev1beta1.DataSafePrivateEndpointSpec,
	current datasafesdk.DataSafePrivateEndpoint,
) error {
	var drift []string
	if !dataSafePrivateEndpointStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !dataSafePrivateEndpointStringPtrEqual(current.VcnId, spec.VcnId) {
		drift = append(drift, "vcnId")
	}
	if !dataSafePrivateEndpointStringPtrEqual(current.SubnetId, spec.SubnetId) {
		drift = append(drift, "subnetId")
	}
	if strings.TrimSpace(spec.PrivateEndpointIp) != "" &&
		!dataSafePrivateEndpointStringPtrEqual(current.PrivateEndpointIp, spec.PrivateEndpointIp) {
		drift = append(drift, "privateEndpointIp")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only drift detected for %s; replace the resource or restore the desired spec before update", dataSafePrivateEndpointKind, strings.Join(drift, ", "))
}

func listDataSafePrivateEndpointsAllPages(
	call func(context.Context, datasafesdk.ListDataSafePrivateEndpointsRequest) (datasafesdk.ListDataSafePrivateEndpointsResponse, error),
) func(context.Context, datasafesdk.ListDataSafePrivateEndpointsRequest) (datasafesdk.ListDataSafePrivateEndpointsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListDataSafePrivateEndpointsRequest) (datasafesdk.ListDataSafePrivateEndpointsResponse, error) {
		if call == nil {
			return datasafesdk.ListDataSafePrivateEndpointsResponse{}, fmt.Errorf("%s list operation is not configured", dataSafePrivateEndpointKind)
		}
		return collectDataSafePrivateEndpointListPages(ctx, call, request)
	}
}

func collectDataSafePrivateEndpointListPages(
	ctx context.Context,
	call func(context.Context, datasafesdk.ListDataSafePrivateEndpointsRequest) (datasafesdk.ListDataSafePrivateEndpointsResponse, error),
	request datasafesdk.ListDataSafePrivateEndpointsRequest,
) (datasafesdk.ListDataSafePrivateEndpointsResponse, error) {
	seenPages := map[string]struct{}{}
	var combined datasafesdk.ListDataSafePrivateEndpointsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return datasafesdk.ListDataSafePrivateEndpointsResponse{}, err
		}
		appendDataSafePrivateEndpointListPage(&combined, response)

		nextPage, ok, err := nextDataSafePrivateEndpointListPage(response, seenPages)
		if err != nil || !ok {
			return combined, err
		}
		request.Page = common.String(nextPage)
	}
}

func appendDataSafePrivateEndpointListPage(
	combined *datasafesdk.ListDataSafePrivateEndpointsResponse,
	response datasafesdk.ListDataSafePrivateEndpointsResponse,
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

func nextDataSafePrivateEndpointListPage(
	response datasafesdk.ListDataSafePrivateEndpointsResponse,
	seenPages map[string]struct{},
) (string, bool, error) {
	nextPage := strings.TrimSpace(dataSafePrivateEndpointStringValue(response.OpcNextPage))
	if nextPage == "" {
		return "", false, nil
	}
	if _, exists := seenPages[nextPage]; exists {
		return "", false, fmt.Errorf("%s list pagination repeated page token %q", dataSafePrivateEndpointKind, nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nextPage, true, nil
}

func projectDataSafePrivateEndpointStatus(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", dataSafePrivateEndpointKind)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current, ok := dataSafePrivateEndpointFromResponse(response)
	if !ok {
		return nil
	}
	return projectDataSafePrivateEndpointSDKStatus(resource, current)
}

func projectDataSafePrivateEndpointSDKStatus(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	current datasafesdk.DataSafePrivateEndpoint,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", dataSafePrivateEndpointKind)
	}

	status := &resource.Status
	status.Id = dataSafePrivateEndpointStringValue(current.Id)
	status.DisplayName = dataSafePrivateEndpointStringValue(current.DisplayName)
	status.CompartmentId = dataSafePrivateEndpointStringValue(current.CompartmentId)
	status.VcnId = dataSafePrivateEndpointStringValue(current.VcnId)
	status.SubnetId = dataSafePrivateEndpointStringValue(current.SubnetId)
	status.PrivateEndpointId = dataSafePrivateEndpointStringValue(current.PrivateEndpointId)
	status.PrivateEndpointIp = dataSafePrivateEndpointStringValue(current.PrivateEndpointIp)
	status.EndpointFqdn = dataSafePrivateEndpointStringValue(current.EndpointFqdn)
	status.Description = dataSafePrivateEndpointStringValue(current.Description)
	status.TimeCreated = dataSafePrivateEndpointTimeString(current.TimeCreated)
	status.LifecycleState = string(current.LifecycleState)
	status.NsgIds = cloneDataSafePrivateEndpointStringSlice(current.NsgIds)
	status.FreeformTags = maps.Clone(current.FreeformTags)
	status.DefinedTags = dataSafePrivateEndpointStatusTagsFromSDK(current.DefinedTags)
	status.SystemTags = dataSafePrivateEndpointStatusTagsFromSDK(current.SystemTags)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
	return nil
}

func dataSafePrivateEndpointFromResponse(response any) (datasafesdk.DataSafePrivateEndpoint, bool) {
	response = dataSafePrivateEndpointDereference(response)
	switch current := response.(type) {
	case nil:
		return datasafesdk.DataSafePrivateEndpoint{}, false
	case datasafesdk.DataSafePrivateEndpoint:
		return current, true
	case datasafesdk.DataSafePrivateEndpointSummary:
		return dataSafePrivateEndpointFromSummary(current), true
	case datasafesdk.CreateDataSafePrivateEndpointResponse:
		return current.DataSafePrivateEndpoint, true
	case datasafesdk.GetDataSafePrivateEndpointResponse:
		return current.DataSafePrivateEndpoint, true
	case map[string]any:
		return dataSafePrivateEndpointFromStatusMap(current)
	default:
		return datasafesdk.DataSafePrivateEndpoint{}, false
	}
}

func dataSafePrivateEndpointDereference(response any) any {
	value := reflect.ValueOf(response)
	if !value.IsValid() || value.Kind() != reflect.Pointer {
		return response
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}

func dataSafePrivateEndpointFromStatusMap(values map[string]any) (datasafesdk.DataSafePrivateEndpoint, bool) {
	if len(values) == 0 {
		return datasafesdk.DataSafePrivateEndpoint{}, false
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return datasafesdk.DataSafePrivateEndpoint{}, false
	}
	var status datasafev1beta1.DataSafePrivateEndpointStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return datasafesdk.DataSafePrivateEndpoint{}, false
	}
	return dataSafePrivateEndpointFromStatus(status), true
}

func dataSafePrivateEndpointFromSummary(
	summary datasafesdk.DataSafePrivateEndpointSummary,
) datasafesdk.DataSafePrivateEndpoint {
	return datasafesdk.DataSafePrivateEndpoint{
		Id:                summary.Id,
		DisplayName:       summary.DisplayName,
		CompartmentId:     summary.CompartmentId,
		VcnId:             summary.VcnId,
		SubnetId:          summary.SubnetId,
		PrivateEndpointId: summary.PrivateEndpointId,
		Description:       summary.Description,
		TimeCreated:       summary.TimeCreated,
		LifecycleState:    summary.LifecycleState,
		FreeformTags:      summary.FreeformTags,
		DefinedTags:       summary.DefinedTags,
		SystemTags:        summary.SystemTags,
	}
}

func dataSafePrivateEndpointFromStatus(
	status datasafev1beta1.DataSafePrivateEndpointStatus,
) datasafesdk.DataSafePrivateEndpoint {
	return datasafesdk.DataSafePrivateEndpoint{
		Id:                common.String(status.Id),
		DisplayName:       common.String(status.DisplayName),
		CompartmentId:     common.String(status.CompartmentId),
		VcnId:             common.String(status.VcnId),
		SubnetId:          common.String(status.SubnetId),
		PrivateEndpointId: common.String(status.PrivateEndpointId),
		PrivateEndpointIp: common.String(status.PrivateEndpointIp),
		EndpointFqdn:      common.String(status.EndpointFqdn),
		Description:       common.String(status.Description),
		LifecycleState:    datasafesdk.LifecycleStateEnum(status.LifecycleState),
		NsgIds:            cloneDataSafePrivateEndpointStringSlice(status.NsgIds),
		FreeformTags:      maps.Clone(status.FreeformTags),
		DefinedTags:       dataSafePrivateEndpointDefinedTagsFromStatus(status.DefinedTags),
		SystemTags:        dataSafePrivateEndpointDefinedTagsFromStatus(status.SystemTags),
	}
}

func clearTrackedDataSafePrivateEndpointIdentity(resource *datasafev1beta1.DataSafePrivateEndpoint) {
	if resource == nil {
		return
	}
	status := resource.Status.OsokStatus
	status.Ocid = ""
	resource.Status = datasafev1beta1.DataSafePrivateEndpointStatus{OsokStatus: status}
}

func currentDataSafePrivateEndpointID(resource *datasafev1beta1.DataSafePrivateEndpoint) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func getDataSafePrivateEndpointWorkRequest(
	ctx context.Context,
	client dataSafePrivateEndpointOCIClient,
	initErr error,
	workRequestID string,
) (datasafesdk.WorkRequest, error) {
	if initErr != nil {
		return datasafesdk.WorkRequest{}, fmt.Errorf("initialize %s OCI client: %w", dataSafePrivateEndpointKind, initErr)
	}
	if client == nil {
		return datasafesdk.WorkRequest{}, fmt.Errorf("%s work request client is not configured", dataSafePrivateEndpointKind)
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return datasafesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func dataSafePrivateEndpointAsyncOperation(
	status *shared.OSOKStatus,
	workRequest datasafesdk.WorkRequest,
	fallback shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	rawAction, err := resolveDataSafePrivateEndpointWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, dataSafePrivateEndpointWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        rawAction,
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    dataSafePrivateEndpointStringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallback,
	})
}

func resolveDataSafePrivateEndpointWorkRequestAction(workRequest any) (string, error) {
	current, ok := dataSafePrivateEndpointWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", dataSafePrivateEndpointKind, workRequest)
	}
	if current.OperationType != "" {
		return string(current.OperationType), nil
	}

	var action string
	for _, resource := range current.Resources {
		if !isDataSafePrivateEndpointWorkRequestResource(resource) || dataSafePrivateEndpointIgnorableAction(resource.ActionType) {
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
			return "", fmt.Errorf("%s work request %s exposes conflicting action types %q and %q", dataSafePrivateEndpointKind, dataSafePrivateEndpointStringValue(current.Id), action, candidate)
		}
	}
	return action, nil
}

func recoverDataSafePrivateEndpointIDFromWorkRequest(
	_ *datasafev1beta1.DataSafePrivateEndpoint,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, ok := dataSafePrivateEndpointWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", dataSafePrivateEndpointKind, workRequest)
	}
	resourceID := dataSafePrivateEndpointIDFromWorkRequest(current, phase)
	if resourceID == "" {
		return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", dataSafePrivateEndpointKind, phase, dataSafePrivateEndpointStringValue(current.Id), dataSafePrivateEndpointKind)
	}
	return resourceID, nil
}

func dataSafePrivateEndpointIDFromWorkRequest(
	workRequest datasafesdk.WorkRequest,
	phase shared.OSOKAsyncPhase,
) string {
	action := dataSafePrivateEndpointActionForPhase(phase)
	if id, ok := dataSafePrivateEndpointIDFromWorkRequestResources(workRequest.Resources, action, true); ok {
		return id
	}
	id, _ := dataSafePrivateEndpointIDFromWorkRequestResources(workRequest.Resources, "", false)
	return id
}

func dataSafePrivateEndpointIDFromWorkRequestResources(
	resources []datasafesdk.WorkRequestResource,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
	requireUnique bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		id, ok := dataSafePrivateEndpointIDFromWorkRequestResource(resource, action)
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

func dataSafePrivateEndpointIDFromWorkRequestResource(
	resource datasafesdk.WorkRequestResource,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
) (string, bool) {
	if !isDataSafePrivateEndpointWorkRequestResource(resource) {
		return "", false
	}
	if !dataSafePrivateEndpointWorkRequestActionMatches(resource.ActionType, action) {
		return "", false
	}
	id := strings.TrimSpace(dataSafePrivateEndpointStringValue(resource.Identifier))
	return id, id != ""
}

func dataSafePrivateEndpointWorkRequestActionMatches(
	resourceAction datasafesdk.WorkRequestResourceActionTypeEnum,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
) bool {
	if action == "" {
		return !dataSafePrivateEndpointIgnorableAction(resourceAction)
	}
	return resourceAction == action || resourceAction == datasafesdk.WorkRequestResourceActionTypeInProgress
}

func dataSafePrivateEndpointActionForPhase(phase shared.OSOKAsyncPhase) datasafesdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return datasafesdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return datasafesdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return datasafesdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func dataSafePrivateEndpointWorkRequestFromAny(workRequest any) (datasafesdk.WorkRequest, bool) {
	workRequest = dataSafePrivateEndpointDereference(workRequest)
	current, ok := workRequest.(datasafesdk.WorkRequest)
	return current, ok
}

func isDataSafePrivateEndpointWorkRequestResource(resource datasafesdk.WorkRequestResource) bool {
	entityType := normalizeDataSafePrivateEndpointEntityType(dataSafePrivateEndpointStringValue(resource.EntityType))
	switch entityType {
	case "datasafeprivateendpoint", "privateendpoint":
		return true
	default:
		return strings.Contains(entityType, "privateendpoint") && strings.Contains(entityType, "datasafe")
	}
}

func normalizeDataSafePrivateEndpointEntityType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", ".", "", "/", "")
	return replacer.Replace(value)
}

func dataSafePrivateEndpointIgnorableAction(action datasafesdk.WorkRequestResourceActionTypeEnum) bool {
	return action == "" || action == datasafesdk.WorkRequestResourceActionTypeFailed
}

func dataSafePrivateEndpointWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, ok := dataSafePrivateEndpointWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	workRequestID := dataSafePrivateEndpointStringValue(current.Id)
	rawStatus := strings.TrimSpace(string(current.Status))
	if phase == "" || workRequestID == "" || rawStatus == "" {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", dataSafePrivateEndpointKind, phase, workRequestID, rawStatus)
}

func handleDataSafePrivateEndpointDeleteError(
	resource *datasafev1beta1.DataSafePrivateEndpoint,
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
	return dataSafePrivateEndpointAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s delete path returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed", dataSafePrivateEndpointKind),
		opcRequestID: requestID,
	}
}

func dataSafePrivateEndpointIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func dataSafePrivateEndpointWriteLifecyclePending(current datasafesdk.DataSafePrivateEndpoint) bool {
	state := strings.ToUpper(string(current.LifecycleState))
	return state == string(datasafesdk.LifecycleStateCreating) ||
		state == string(datasafesdk.LifecycleStateUpdating)
}

func dataSafePrivateEndpointDeleteLifecyclePending(current datasafesdk.DataSafePrivateEndpoint) bool {
	state := strings.ToUpper(string(current.LifecycleState))
	return state == "" || state == string(datasafesdk.LifecycleStateDeleting)
}

func dataSafePrivateEndpointDeleteLifecycleTerminal(current datasafesdk.DataSafePrivateEndpoint) bool {
	return strings.ToUpper(string(current.LifecycleState)) == string(datasafesdk.LifecycleStateDeleted)
}

func dataSafePrivateEndpointDefinedTagsFromSpec(
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

func dataSafePrivateEndpointDefinedTagsFromStatus(
	status map[string]shared.MapValue,
) map[string]map[string]interface{} {
	return dataSafePrivateEndpointDefinedTagsFromSpec(status)
}

func dataSafePrivateEndpointStatusTagsFromSDK(
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

func dataSafePrivateEndpointStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(dataSafePrivateEndpointStringValue(current)) == strings.TrimSpace(desired)
}

func dataSafePrivateEndpointStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cloneDataSafePrivateEndpointStringSlice(input []string) []string {
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

func dataSafePrivateEndpointStringSlicesEqual(left []string, right []string) bool {
	left = cloneDataSafePrivateEndpointStringSlice(left)
	right = cloneDataSafePrivateEndpointStringSlice(right)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}

func dataSafePrivateEndpointTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func dataSafePrivateEndpointJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
