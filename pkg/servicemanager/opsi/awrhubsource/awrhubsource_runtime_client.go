/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package awrhubsource

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const awrHubSourceKind = "AwrHubSource"

var awrHubSourceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opsisdk.OperationTypeCreateAwrhubSource)},
	UpdateActionTokens:    []string{string(opsisdk.OperationTypeUpdateAwrhubSource)},
	DeleteActionTokens:    []string{string(opsisdk.OperationTypeDeleteAwrhubSource)},
}

type awrHubSourceOCIClient interface {
	CreateAwrHubSource(context.Context, opsisdk.CreateAwrHubSourceRequest) (opsisdk.CreateAwrHubSourceResponse, error)
	GetAwrHubSource(context.Context, opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error)
	ListAwrHubSources(context.Context, opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error)
	UpdateAwrHubSource(context.Context, opsisdk.UpdateAwrHubSourceRequest) (opsisdk.UpdateAwrHubSourceResponse, error)
	DeleteAwrHubSource(context.Context, opsisdk.DeleteAwrHubSourceRequest) (opsisdk.DeleteAwrHubSourceResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type awrHubSourceWorkRequestClient interface {
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type awrHubSourceRuntimeClient struct {
	delegate           AwrHubSourceServiceClient
	hooks              AwrHubSourceRuntimeHooks
	workRequestClient  awrHubSourceWorkRequestClient
	workRequestInitErr error
	log                loggerutil.OSOKLogger
}

type awrHubSourceRuntimeReadResponse struct {
	Body         map[string]any `presentIn:"body"`
	OpcRequestId *string        `presentIn:"header" name:"opc-request-id"`
	Etag         *string        `presentIn:"header" name:"etag"`
}

type awrHubSourceRuntimeListBody struct {
	Items []map[string]any `json:"items"`
}

type awrHubSourceRuntimeListResponse struct {
	Body         awrHubSourceRuntimeListBody `presentIn:"body"`
	OpcRequestId *string                     `presentIn:"header" name:"opc-request-id"`
	OpcNextPage  *string                     `presentIn:"header" name:"opc-next-page"`
}

type awrHubSourceAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e awrHubSourceAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e awrHubSourceAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var _ AwrHubSourceServiceClient = (*awrHubSourceRuntimeClient)(nil)

func init() {
	registerAwrHubSourceRuntimeHooksMutator(func(manager *AwrHubSourceServiceManager, hooks *AwrHubSourceRuntimeHooks) {
		workRequestClient, initErr := newAwrHubSourceWorkRequestClient(manager)
		applyAwrHubSourceRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newAwrHubSourceWorkRequestClient(manager *AwrHubSourceServiceManager) (awrHubSourceWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", awrHubSourceKind)
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAwrHubSourceRuntimeHooks(
	manager *AwrHubSourceServiceManager,
	hooks *AwrHubSourceRuntimeHooks,
	workRequestClient awrHubSourceWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAwrHubSourceRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *opsiv1beta1.AwrHubSource, _ string) (any, error) {
		return buildAwrHubSourceCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *opsiv1beta1.AwrHubSource,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildAwrHubSourceUpdateBody(resource, currentResponse)
	}
	if hooks.List.Call != nil {
		hooks.List.Call = listAwrHubSourcesAllPages(hooks.List.Call)
	}
	hooks.List.Fields = awrHubSourceListFields()
	hooks.Read.Get = awrHubSourceGetReadOperation(hooks)
	hooks.Read.List = awrHubSourceListReadOperation(hooks)
	hooks.StatusHooks.ProjectStatus = projectAwrHubSourceStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateAwrHubSourceCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleAwrHubSourceDeleteError
	hooks.Async.Adapter = awrHubSourceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getAwrHubSourceWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveAwrHubSourceWorkRequestAction
	hooks.Async.RecoverResourceID = recoverAwrHubSourceIDFromWorkRequest
	hooks.Async.Message = awrHubSourceWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AwrHubSourceServiceClient) AwrHubSourceServiceClient {
		runtimeClient := &awrHubSourceRuntimeClient{
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

func newAwrHubSourceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "awrhubsource",
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
			ProvisioningStates: []string{string(opsisdk.AwrHubSourceLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.AwrHubSourceLifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.AwrHubSourceLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.AwrHubSourceLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.AwrHubSourceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"awrHubId", "compartmentId", "name", "type", "associatedResourceId", "associatedOpsiId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{"type", "freeformTags", "definedTags"},
			ForceNew: []string{
				"name",
				"awrHubId",
				"compartmentId",
				"associatedResourceId",
				"associatedOpsiId",
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func newAwrHubSourceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client awrHubSourceOCIClient,
) AwrHubSourceServiceClient {
	manager := &AwrHubSourceServiceManager{Log: log}
	hooks := newAwrHubSourceRuntimeHooksWithOCIClient(client)
	applyAwrHubSourceRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultAwrHubSourceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.AwrHubSource](
			buildAwrHubSourceGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAwrHubSourceGeneratedClient(hooks, delegate)
}

func newAwrHubSourceRuntimeHooksWithOCIClient(client awrHubSourceOCIClient) AwrHubSourceRuntimeHooks {
	return AwrHubSourceRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.AwrHubSource]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.AwrHubSource]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.AwrHubSource]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.AwrHubSource]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.AwrHubSource]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.AwrHubSource]{},
		Create: runtimeOperationHooks[opsisdk.CreateAwrHubSourceRequest, opsisdk.CreateAwrHubSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateAwrHubSourceDetails", RequestName: "CreateAwrHubSourceDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request opsisdk.CreateAwrHubSourceRequest) (opsisdk.CreateAwrHubSourceResponse, error) {
				return client.CreateAwrHubSource(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetAwrHubSourceRequest, opsisdk.GetAwrHubSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "AwrHubSourceId", RequestName: "awrHubSourceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetAwrHubSourceRequest) (opsisdk.GetAwrHubSourceResponse, error) {
				return client.GetAwrHubSource(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListAwrHubSourcesRequest, opsisdk.ListAwrHubSourcesResponse]{
			Fields: awrHubSourceListFields(),
			Call: func(ctx context.Context, request opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error) {
				return client.ListAwrHubSources(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateAwrHubSourceRequest, opsisdk.UpdateAwrHubSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "AwrHubSourceId", RequestName: "awrHubSourceId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateAwrHubSourceDetails", RequestName: "UpdateAwrHubSourceDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request opsisdk.UpdateAwrHubSourceRequest) (opsisdk.UpdateAwrHubSourceResponse, error) {
				return client.UpdateAwrHubSource(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteAwrHubSourceRequest, opsisdk.DeleteAwrHubSourceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "AwrHubSourceId", RequestName: "awrHubSourceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteAwrHubSourceRequest) (opsisdk.DeleteAwrHubSourceResponse, error) {
				return client.DeleteAwrHubSource(ctx, request)
			},
		},
		WrapGeneratedClient: []func(AwrHubSourceServiceClient) AwrHubSourceServiceClient{},
	}
}

func (c *awrHubSourceRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", awrHubSourceKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *awrHubSourceRuntimeClient) Delete(ctx context.Context, resource *opsiv1beta1.AwrHubSource) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", awrHubSourceKind)
	}
	if c == nil {
		return false, fmt.Errorf("%s runtime client is not configured", awrHubSourceKind)
	}

	if current := resource.Status.OsokStatus.Async.Current; current != nil && current.Source == shared.OSOKAsyncSourceWorkRequest && current.WorkRequestID != "" {
		switch current.Phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			return c.waitForPendingWriteBeforeDelete(ctx, resource, current)
		case shared.OSOKAsyncPhaseDelete:
			return c.resumeDeleteWorkRequest(ctx, resource, current.WorkRequestID)
		}
	}
	return c.deleteResolvedAwrHubSource(ctx, resource)
}

func (c *awrHubSourceRuntimeClient) waitForPendingWriteBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	tracked *shared.OSOKAsyncOperation,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, tracked.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := awrHubSourceAsyncOperation(&resource.Status.OsokStatus, workRequest, tracked.Phase)
	if err != nil {
		return false, err
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequest(resource, current)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		c.applyWorkRequest(resource, current)
		currentAwrHubSource, found, err := c.resolveAwrHubSourceAfterSucceededWrite(ctx, resource, workRequest, current)
		if err != nil {
			return false, err
		}
		if !found {
			c.markWriteWorkRequestReadbackPending(resource, current, tracked.WorkRequestID, awrHubSourceIDFromWorkRequest(workRequest, tracked.Phase))
			return false, nil
		}
		if awrHubSourceLifecyclePending(currentAwrHubSource) {
			c.projectStatus(resource, currentAwrHubSource)
			c.markTerminating(resource, "OCI write is still settling before delete", currentAwrHubSource)
			return false, nil
		}
		resource.Status.OsokStatus.Async.Current = nil
		return c.deleteResolvedAwrHubSource(ctx, resource)
	default:
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s %s work request %s finished with status %s", awrHubSourceKind, current.Phase, tracked.WorkRequestID, current.RawStatus)
	}
}

func (c *awrHubSourceRuntimeClient) resolveAwrHubSourceAfterSucceededWrite(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	workRequest opsisdk.WorkRequest,
	current *shared.OSOKAsyncOperation,
) (opsisdk.AwrHubSource, bool, error) {
	resourceID := awrHubSourceIDFromWorkRequest(workRequest, current.Phase)
	if resourceID == "" {
		return c.resolveAwrHubSourceForDelete(ctx, resource)
	}

	live, found, err := c.getAwrHubSourceForDelete(ctx, resource, resourceID)
	if err != nil || found {
		return live, found, err
	}
	return opsisdk.AwrHubSource{}, false, nil
}

func (c *awrHubSourceRuntimeClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := awrHubSourceAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequest(resource, current)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		c.applyWorkRequest(resource, current)
		resourceID := currentAwrHubSourceID(resource)
		if resourceID == "" {
			resourceID = awrHubSourceIDFromWorkRequest(workRequest, shared.OSOKAsyncPhaseDelete)
		}
		return c.confirmDeleted(ctx, resource, resourceID, "OCI resource deleted")
	default:
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s delete work request %s finished with status %s", awrHubSourceKind, workRequestID, current.RawStatus)
	}
}

func (c *awrHubSourceRuntimeClient) deleteResolvedAwrHubSource(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
) (bool, error) {
	current, found, err := c.resolveAwrHubSourceForDelete(ctx, resource)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, fmt.Sprintf("OCI %s no longer exists", awrHubSourceKind))
		return true, nil
	}
	c.projectStatus(resource, current)
	if awrHubSourceLifecycleDeleted(current) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if awrHubSourceLifecycleDeleting(current) {
		c.markTerminating(resource, "OCI resource delete is in progress", current)
		return false, nil
	}

	resourceID := awrHubSourceStringValue(current.Id)
	response, err := c.hooks.Delete.Call(ctx, opsisdk.DeleteAwrHubSourceRequest{AwrHubSourceId: common.String(resourceID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if awrHubSourceIsUnambiguousNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, handleAwrHubSourceDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := awrHubSourceStringValue(response.OpcWorkRequestId); workRequestID != "" {
		c.markWorkRequest(resource, workRequestID, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "")
		return false, nil
	}
	return c.confirmDeleted(ctx, resource, resourceID, "OCI resource deleted")
}

func (c *awrHubSourceRuntimeClient) confirmDeleted(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	resourceID string,
	deletedMessage string,
) (bool, error) {
	current, found, err := c.getAwrHubSourceForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, deletedMessage)
		return true, nil
	}
	c.projectStatus(resource, current)
	if awrHubSourceLifecycleDeleted(current) {
		c.markDeleted(resource, deletedMessage)
		return true, nil
	}
	c.markTerminating(resource, "OCI resource delete is in progress", current)
	return false, nil
}

func (c *awrHubSourceRuntimeClient) resolveAwrHubSourceForDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
) (opsisdk.AwrHubSource, bool, error) {
	if resourceID := currentAwrHubSourceID(resource); resourceID != "" {
		return c.getAwrHubSourceForDelete(ctx, resource, resourceID)
	}

	response, found, err := c.listAwrHubSourcesForDelete(ctx, resource)
	if err != nil || !found {
		return opsisdk.AwrHubSource{}, false, err
	}
	matches := awrHubSourceSummariesMatchingSpec(response.Items, resource.Spec)
	return c.resolveListedAwrHubSourceForDelete(ctx, resource, matches)
}

func (c *awrHubSourceRuntimeClient) listAwrHubSourcesForDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
) (opsisdk.ListAwrHubSourcesResponse, bool, error) {
	if c.hooks.List.Call == nil {
		return opsisdk.ListAwrHubSourcesResponse{}, false, nil
	}

	response, err := c.hooks.List.Call(ctx, opsisdk.ListAwrHubSourcesRequest{
		AwrHubId:      common.String(resource.Spec.AwrHubId),
		CompartmentId: common.String(resource.Spec.CompartmentId),
		Name:          common.String(resource.Spec.Name),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if awrHubSourceIsUnambiguousNotFound(err) {
			return opsisdk.ListAwrHubSourcesResponse{}, false, nil
		}
		return opsisdk.ListAwrHubSourcesResponse{}, false, handleAwrHubSourceDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response, true, nil
}

func awrHubSourceSummariesMatchingSpec(
	items []opsisdk.AwrHubSourceSummary,
	spec opsiv1beta1.AwrHubSourceSpec,
) []opsisdk.AwrHubSourceSummary {
	matches := make([]opsisdk.AwrHubSourceSummary, 0, len(items))
	for _, item := range items {
		if awrHubSourceSummaryMatchesSpec(item, spec) {
			matches = append(matches, item)
		}
	}
	return matches
}

func (c *awrHubSourceRuntimeClient) resolveListedAwrHubSourceForDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	matches []opsisdk.AwrHubSourceSummary,
) (opsisdk.AwrHubSource, bool, error) {
	switch len(matches) {
	case 0:
		return opsisdk.AwrHubSource{}, false, nil
	case 1:
		return c.resolveAwrHubSourceSummaryForDelete(ctx, resource, matches[0])
	default:
		return opsisdk.AwrHubSource{}, false, fmt.Errorf(
			"multiple OCI %ss matched awrHubId %q, compartmentId %q, name %q, and type %q",
			awrHubSourceKind,
			resource.Spec.AwrHubId,
			resource.Spec.CompartmentId,
			resource.Spec.Name,
			resource.Spec.Type,
		)
	}
}

func (c *awrHubSourceRuntimeClient) resolveAwrHubSourceSummaryForDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	summary opsisdk.AwrHubSourceSummary,
) (opsisdk.AwrHubSource, bool, error) {
	id := awrHubSourceStringValue(summary.Id)
	if id == "" || c.hooks.Get.Call == nil {
		return awrHubSourceFromSummary(summary), true, nil
	}
	current, found, err := c.getAwrHubSourceForDelete(ctx, resource, id)
	if err != nil || found {
		return current, found, err
	}
	return awrHubSourceFromSummary(summary), true, nil
}

func (c *awrHubSourceRuntimeClient) getAwrHubSourceForDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHubSource,
	resourceID string,
) (opsisdk.AwrHubSource, bool, error) {
	if resourceID == "" || c.hooks.Get.Call == nil {
		return opsisdk.AwrHubSource{}, false, nil
	}
	response, err := c.hooks.Get.Call(ctx, opsisdk.GetAwrHubSourceRequest{AwrHubSourceId: common.String(resourceID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if awrHubSourceIsUnambiguousNotFound(err) {
			return opsisdk.AwrHubSource{}, false, nil
		}
		return opsisdk.AwrHubSource{}, false, handleAwrHubSourceDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response.AwrHubSource, true, nil
}

func (c *awrHubSourceRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (opsisdk.WorkRequest, error) {
	return getAwrHubSourceWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, workRequestID)
}

func (c *awrHubSourceRuntimeClient) projectStatus(resource *opsiv1beta1.AwrHubSource, current opsisdk.AwrHubSource) {
	_ = projectAwrHubSourceSDKStatus(resource, current)
}

func (c *awrHubSourceRuntimeClient) markWorkRequest(
	resource *opsiv1beta1.AwrHubSource,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	rawStatus string,
) {
	message := fmt.Sprintf("%s %s work request %s is pending", awrHubSourceKind, phase, workRequestID)
	if rawStatus == "" {
		rawStatus = string(opsisdk.OperationStatusAccepted)
	}
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		RawStatus:       rawStatus,
		NormalizedClass: class,
		Message:         message,
	}
	c.applyWorkRequest(resource, current)
}

func (c *awrHubSourceRuntimeClient) applyWorkRequest(resource *opsiv1beta1.AwrHubSource, current *shared.OSOKAsyncOperation) {
	if current == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *awrHubSourceRuntimeClient) markWriteWorkRequestReadbackPending(
	resource *opsiv1beta1.AwrHubSource,
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
	next.Message = awrHubSourceWriteReadbackPendingMessage(current.Phase, workRequestID, resourceID)
	next.UpdatedAt = &now

	if strings.TrimSpace(resourceID) != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func awrHubSourceWriteReadbackPendingMessage(
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	resourceID string,
) string {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Sprintf(
			"%s %s work request %s succeeded; waiting for %s readback before delete",
			awrHubSourceKind,
			phase,
			strings.TrimSpace(workRequestID),
			awrHubSourceKind,
		)
	}
	return fmt.Sprintf(
		"%s %s work request %s succeeded; waiting for %s %s to become readable before delete",
		awrHubSourceKind,
		phase,
		strings.TrimSpace(workRequestID),
		awrHubSourceKind,
		strings.TrimSpace(resourceID),
	)
}

func (c *awrHubSourceRuntimeClient) markDeleted(resource *opsiv1beta1.AwrHubSource, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Async.Current = nil
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, c.log)
}

func (c *awrHubSourceRuntimeClient) markTerminating(
	resource *opsiv1beta1.AwrHubSource,
	message string,
	current opsisdk.AwrHubSource,
) {
	rawStatus := strings.ToUpper(string(current.LifecycleState))
	if rawStatus == "" {
		rawStatus = string(opsisdk.AwrHubSourceLifecycleStateDeleting)
	}
	async := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, async, c.log)
}

func buildAwrHubSourceCreateBody(resource *opsiv1beta1.AwrHubSource) (opsisdk.CreateAwrHubSourceDetails, error) {
	if resource == nil {
		return opsisdk.CreateAwrHubSourceDetails{}, fmt.Errorf("%s resource is nil", awrHubSourceKind)
	}

	spec := resource.Spec
	details := opsisdk.CreateAwrHubSourceDetails{
		Name:          common.String(spec.Name),
		AwrHubId:      common.String(spec.AwrHubId),
		CompartmentId: common.String(spec.CompartmentId),
		Type:          opsisdk.AwrHubSourceTypeEnum(strings.TrimSpace(spec.Type)),
		FreeformTags:  maps.Clone(spec.FreeformTags),
		DefinedTags:   awrHubSourceDefinedTagsFromSpec(spec.DefinedTags),
	}
	if strings.TrimSpace(spec.AssociatedResourceId) != "" {
		details.AssociatedResourceId = common.String(spec.AssociatedResourceId)
	}
	if strings.TrimSpace(spec.AssociatedOpsiId) != "" {
		details.AssociatedOpsiId = common.String(spec.AssociatedOpsiId)
	}
	return details, nil
}

func buildAwrHubSourceUpdateBody(
	resource *opsiv1beta1.AwrHubSource,
	currentResponse any,
) (opsisdk.UpdateAwrHubSourceDetails, bool, error) {
	if resource == nil {
		return opsisdk.UpdateAwrHubSourceDetails{}, false, fmt.Errorf("%s resource is nil", awrHubSourceKind)
	}

	current, _, err := awrHubSourceFromCurrentState(resource, currentResponse)
	if err != nil {
		return opsisdk.UpdateAwrHubSourceDetails{}, false, err
	}

	spec := resource.Spec
	details := opsisdk.UpdateAwrHubSourceDetails{}
	updateNeeded := false
	desiredType := strings.TrimSpace(spec.Type)
	currentType := strings.TrimSpace(string(current.Type))
	if desiredType != "" && desiredType != currentType {
		details.Type = opsisdk.AwrHubSourceTypeEnum(desiredType)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		definedTags := awrHubSourceDefinedTagsFromSpec(spec.DefinedTags)
		if !awrHubSourceDefinedTagsEqual(definedTags, current.DefinedTags) {
			details.DefinedTags = definedTags
			updateNeeded = true
		}
	}
	return details, updateNeeded, nil
}

func validateAwrHubSourceCreateOnlyDriftForResponse(resource *opsiv1beta1.AwrHubSource, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", awrHubSourceKind)
	}
	current, _, err := awrHubSourceFromCurrentState(resource, currentResponse)
	if err != nil {
		return err
	}

	drift := awrHubSourceCreateOnlyDriftFields(resource.Spec, current)
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only drift detected for %s; replace the resource or restore the desired spec before update", awrHubSourceKind, strings.Join(drift, ", "))
}

func awrHubSourceCreateOnlyDriftFields(spec opsiv1beta1.AwrHubSourceSpec, current opsisdk.AwrHubSource) []string {
	checks := []struct {
		name    string
		desired string
		current string
	}{
		{name: "name", desired: spec.Name, current: awrHubSourceStringValue(current.Name)},
		{name: "awrHubId", desired: spec.AwrHubId, current: awrHubSourceStringValue(current.AwrHubId)},
		{name: "compartmentId", desired: spec.CompartmentId, current: awrHubSourceStringValue(current.CompartmentId)},
		{name: "associatedResourceId", desired: spec.AssociatedResourceId, current: awrHubSourceStringValue(current.AssociatedResourceId)},
		{name: "associatedOpsiId", desired: spec.AssociatedOpsiId, current: awrHubSourceStringValue(current.AssociatedOpsiId)},
	}

	drift := make([]string, 0, len(checks))
	for _, check := range checks {
		if awrHubSourceHasStringCreateOnlyDrift(check.desired, check.current) {
			drift = append(drift, check.name)
		}
	}
	return drift
}

func awrHubSourceHasStringCreateOnlyDrift(desired string, current string) bool {
	desired = strings.TrimSpace(desired)
	current = strings.TrimSpace(current)
	return desired != current
}

func listAwrHubSourcesAllPages(
	call func(context.Context, opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error),
) func(context.Context, opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error) {
	return func(ctx context.Context, request opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error) {
		return listAwrHubSourcePages(ctx, call, request)
	}
}

func listAwrHubSourcePages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListAwrHubSourcesRequest) (opsisdk.ListAwrHubSourcesResponse, error),
	request opsisdk.ListAwrHubSourcesRequest,
) (opsisdk.ListAwrHubSourcesResponse, error) {
	if call == nil {
		return opsisdk.ListAwrHubSourcesResponse{}, fmt.Errorf("%s list operation is not configured", awrHubSourceKind)
	}

	seenPages := map[string]struct{}{}
	var combined opsisdk.ListAwrHubSourcesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return opsisdk.ListAwrHubSourcesResponse{}, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := strings.TrimSpace(awrHubSourceStringValue(response.OpcNextPage))
		if nextPage == "" {
			return combined, nil
		}
		if _, exists := seenPages[nextPage]; exists {
			return opsisdk.ListAwrHubSourcesResponse{}, fmt.Errorf("%s list pagination repeated page token %q", awrHubSourceKind, nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

func awrHubSourceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AwrHubId", RequestName: "awrHubId", Contribution: "query", PreferResourceID: false},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
		{FieldName: "AwrHubSourceId", RequestName: "awrHubSourceId", Contribution: "query", PreferResourceID: false},
		{FieldName: "Name", RequestName: "name", Contribution: "query", PreferResourceID: false},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
		{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
	}
}

func awrHubSourceGetReadOperation(hooks *AwrHubSourceRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.Get.Call == nil {
		return nil
	}
	fields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &opsisdk.GetAwrHubSourceRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*opsisdk.GetAwrHubSourceRequest)
			if !ok {
				return nil, fmt.Errorf("expected *opsi.GetAwrHubSourceRequest, got %T", request)
			}
			response, err := hooks.Get.Call(ctx, *typed)
			if err != nil {
				return nil, err
			}
			return awrHubSourceRuntimeReadResponse{
				Body:         awrHubSourceStatusMap(response.AwrHubSource),
				OpcRequestId: response.OpcRequestId,
				Etag:         response.Etag,
			}, nil
		},
	}
}

func awrHubSourceListReadOperation(hooks *AwrHubSourceRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &opsisdk.ListAwrHubSourcesRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*opsisdk.ListAwrHubSourcesRequest)
			if !ok {
				return nil, fmt.Errorf("expected *opsi.ListAwrHubSourcesRequest, got %T", request)
			}
			response, err := hooks.List.Call(ctx, *typed)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(response.Items))
			for _, item := range response.Items {
				items = append(items, awrHubSourceStatusMap(awrHubSourceFromSummary(item)))
			}
			return awrHubSourceRuntimeListResponse{
				Body:         awrHubSourceRuntimeListBody{Items: items},
				OpcRequestId: response.OpcRequestId,
				OpcNextPage:  response.OpcNextPage,
			}, nil
		},
	}
}

func projectAwrHubSourceStatus(resource *opsiv1beta1.AwrHubSource, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", awrHubSourceKind)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current, ok, err := awrHubSourceFromResponse(response)
	if err != nil || !ok {
		return err
	}
	return projectAwrHubSourceSDKStatus(resource, current)
}

func projectAwrHubSourceSDKStatus(resource *opsiv1beta1.AwrHubSource, current opsisdk.AwrHubSource) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", awrHubSourceKind)
	}

	status := &resource.Status
	status.Name = awrHubSourceStringValue(current.Name)
	status.AwrHubId = awrHubSourceStringValue(current.AwrHubId)
	status.CompartmentId = awrHubSourceStringValue(current.CompartmentId)
	status.Type = string(current.Type)
	status.Id = awrHubSourceStringValue(current.Id)
	status.AwrHubOpsiSourceId = awrHubSourceStringValue(current.AwrHubOpsiSourceId)
	status.SourceMailBoxUrl = awrHubSourceStringValue(current.SourceMailBoxUrl)
	status.TimeCreated = awrHubSourceTimeString(current.TimeCreated)
	status.LifecycleState = string(current.LifecycleState)
	status.Status = string(current.Status)
	status.AssociatedResourceId = awrHubSourceStringValue(current.AssociatedResourceId)
	status.AssociatedOpsiId = awrHubSourceStringValue(current.AssociatedOpsiId)
	status.TimeUpdated = awrHubSourceTimeString(current.TimeUpdated)
	status.FreeformTags = maps.Clone(current.FreeformTags)
	status.DefinedTags = awrHubSourceStatusTagsFromSDK(current.DefinedTags)
	status.SystemTags = awrHubSourceStatusTagsFromSDK(current.SystemTags)
	status.IsRegisteredWithAwrHub = awrHubSourceBoolValue(current.IsRegisteredWithAwrHub)
	status.AwrSourceDatabaseId = awrHubSourceStringValue(current.AwrSourceDatabaseId)
	status.MinSnapshotIdentifier = awrHubSourceFloat32Value(current.MinSnapshotIdentifier)
	status.MaxSnapshotIdentifier = awrHubSourceFloat32Value(current.MaxSnapshotIdentifier)
	status.TimeFirstSnapshotGenerated = awrHubSourceTimeString(current.TimeFirstSnapshotGenerated)
	status.TimeLastSnapshotGenerated = awrHubSourceTimeString(current.TimeLastSnapshotGenerated)
	status.HoursSinceLastImport = awrHubSourceFloat64Value(current.HoursSinceLastImport)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
	return nil
}

func awrHubSourceStatusMap(current opsisdk.AwrHubSource) map[string]any {
	body := map[string]any{
		"name":                       awrHubSourceStringValue(current.Name),
		"awrHubId":                   awrHubSourceStringValue(current.AwrHubId),
		"compartmentId":              awrHubSourceStringValue(current.CompartmentId),
		"type":                       string(current.Type),
		"id":                         awrHubSourceStringValue(current.Id),
		"awrHubOpsiSourceId":         awrHubSourceStringValue(current.AwrHubOpsiSourceId),
		"sourceMailBoxUrl":           awrHubSourceStringValue(current.SourceMailBoxUrl),
		"timeCreated":                awrHubSourceTimeString(current.TimeCreated),
		"lifecycleState":             string(current.LifecycleState),
		"sdkStatus":                  string(current.Status),
		"associatedResourceId":       awrHubSourceStringValue(current.AssociatedResourceId),
		"associatedOpsiId":           awrHubSourceStringValue(current.AssociatedOpsiId),
		"timeUpdated":                awrHubSourceTimeString(current.TimeUpdated),
		"freeformTags":               maps.Clone(current.FreeformTags),
		"definedTags":                awrHubSourceStatusTagsFromSDK(current.DefinedTags),
		"systemTags":                 awrHubSourceStatusTagsFromSDK(current.SystemTags),
		"isRegisteredWithAwrHub":     awrHubSourceBoolValue(current.IsRegisteredWithAwrHub),
		"awrSourceDatabaseId":        awrHubSourceStringValue(current.AwrSourceDatabaseId),
		"minSnapshotIdentifier":      awrHubSourceFloat32Value(current.MinSnapshotIdentifier),
		"maxSnapshotIdentifier":      awrHubSourceFloat32Value(current.MaxSnapshotIdentifier),
		"timeFirstSnapshotGenerated": awrHubSourceTimeString(current.TimeFirstSnapshotGenerated),
		"timeLastSnapshotGenerated":  awrHubSourceTimeString(current.TimeLastSnapshotGenerated),
		"hoursSinceLastImport":       awrHubSourceFloat64Value(current.HoursSinceLastImport),
	}
	for key, value := range body {
		if !awrHubSourceMeaningfulStatusValue(value) {
			delete(body, key)
		}
	}
	return body
}

func awrHubSourceMeaningfulStatusValue(value any) bool {
	switch current := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(current) != ""
	case bool:
		return current
	case float32:
		return current != 0
	case float64:
		return current != 0
	case map[string]string:
		return len(current) != 0
	case map[string]shared.MapValue:
		return len(current) != 0
	default:
		return true
	}
}

func awrHubSourceFromCurrentState(
	resource *opsiv1beta1.AwrHubSource,
	currentResponse any,
) (opsisdk.AwrHubSource, bool, error) {
	if current, ok, err := awrHubSourceFromResponse(currentResponse); err != nil || ok {
		return current, ok, err
	}
	if resource == nil {
		return opsisdk.AwrHubSource{}, false, nil
	}
	return awrHubSourceFromStatus(resource.Status), true, nil
}

func awrHubSourceFromResponse(response any) (opsisdk.AwrHubSource, bool, error) {
	response = awrHubSourceDereference(response)
	switch current := response.(type) {
	case nil:
		return opsisdk.AwrHubSource{}, false, nil
	case opsisdk.AwrHubSource:
		return current, true, nil
	case opsisdk.AwrHubSourceSummary:
		return awrHubSourceFromSummary(current), true, nil
	case awrHubSourceRuntimeReadResponse:
		return awrHubSourceFromStatusMap(current.Body)
	case awrHubSourceRuntimeListResponse:
		return opsisdk.AwrHubSource{}, false, nil
	case map[string]any:
		return awrHubSourceFromStatusMap(current)
	case opsisdk.CreateAwrHubSourceResponse:
		return current.AwrHubSource, true, nil
	case opsisdk.GetAwrHubSourceResponse:
		return current.AwrHubSource, true, nil
	case opsisdk.UpdateAwrHubSourceResponse, opsisdk.DeleteAwrHubSourceResponse, opsisdk.GetWorkRequestResponse, opsisdk.WorkRequest:
		return opsisdk.AwrHubSource{}, false, nil
	default:
		return opsisdk.AwrHubSource{}, false, fmt.Errorf("unexpected %s response type %T", awrHubSourceKind, response)
	}
}

func awrHubSourceFromStatusMap(values map[string]any) (opsisdk.AwrHubSource, bool, error) {
	if len(values) == 0 {
		return opsisdk.AwrHubSource{}, false, nil
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return opsisdk.AwrHubSource{}, false, fmt.Errorf("marshal %s status map: %w", awrHubSourceKind, err)
	}
	var status opsiv1beta1.AwrHubSourceStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return opsisdk.AwrHubSource{}, false, fmt.Errorf("unmarshal %s status map: %w", awrHubSourceKind, err)
	}
	return awrHubSourceFromStatus(status), true, nil
}

func awrHubSourceDereference(response any) any {
	value := reflect.ValueOf(response)
	if !value.IsValid() || value.Kind() != reflect.Pointer {
		return response
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}

func awrHubSourceFromSummary(summary opsisdk.AwrHubSourceSummary) opsisdk.AwrHubSource {
	return opsisdk.AwrHubSource(summary)
}

func awrHubSourceFromStatus(status opsiv1beta1.AwrHubSourceStatus) opsisdk.AwrHubSource {
	return opsisdk.AwrHubSource{
		Name:                   common.String(status.Name),
		AwrHubId:               common.String(status.AwrHubId),
		CompartmentId:          common.String(status.CompartmentId),
		Type:                   opsisdk.AwrHubSourceTypeEnum(status.Type),
		Id:                     common.String(status.Id),
		AwrHubOpsiSourceId:     common.String(status.AwrHubOpsiSourceId),
		SourceMailBoxUrl:       common.String(status.SourceMailBoxUrl),
		LifecycleState:         opsisdk.AwrHubSourceLifecycleStateEnum(status.LifecycleState),
		Status:                 opsisdk.AwrHubSourceStatusEnum(status.Status),
		AssociatedResourceId:   common.String(status.AssociatedResourceId),
		AssociatedOpsiId:       common.String(status.AssociatedOpsiId),
		FreeformTags:           maps.Clone(status.FreeformTags),
		DefinedTags:            awrHubSourceDefinedTagsFromStatus(status.DefinedTags),
		SystemTags:             awrHubSourceDefinedTagsFromStatus(status.SystemTags),
		IsRegisteredWithAwrHub: common.Bool(status.IsRegisteredWithAwrHub),
		AwrSourceDatabaseId:    common.String(status.AwrSourceDatabaseId),
		MinSnapshotIdentifier:  common.Float32(status.MinSnapshotIdentifier),
		MaxSnapshotIdentifier:  common.Float32(status.MaxSnapshotIdentifier),
		HoursSinceLastImport:   common.Float64(status.HoursSinceLastImport),
	}
}

func awrHubSourceSummaryMatchesSpec(summary opsisdk.AwrHubSourceSummary, spec opsiv1beta1.AwrHubSourceSpec) bool {
	checks := []struct {
		desired string
		current string
	}{
		{desired: spec.AwrHubId, current: awrHubSourceStringValue(summary.AwrHubId)},
		{desired: spec.CompartmentId, current: awrHubSourceStringValue(summary.CompartmentId)},
		{desired: spec.Name, current: awrHubSourceStringValue(summary.Name)},
		{desired: spec.Type, current: string(summary.Type)},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.desired) != "" && check.desired != check.current {
			return false
		}
	}
	if strings.TrimSpace(spec.AssociatedResourceId) != "" && spec.AssociatedResourceId != awrHubSourceStringValue(summary.AssociatedResourceId) {
		return false
	}
	if strings.TrimSpace(spec.AssociatedOpsiId) != "" && spec.AssociatedOpsiId != awrHubSourceStringValue(summary.AssociatedOpsiId) {
		return false
	}
	return true
}

func awrHubSourceLifecyclePending(current opsisdk.AwrHubSource) bool {
	state := strings.ToUpper(string(current.LifecycleState))
	return state == string(opsisdk.AwrHubSourceLifecycleStateCreating) ||
		state == string(opsisdk.AwrHubSourceLifecycleStateUpdating) ||
		state == string(opsisdk.AwrHubSourceLifecycleStateDeleting)
}

func awrHubSourceLifecycleDeleting(current opsisdk.AwrHubSource) bool {
	return strings.ToUpper(string(current.LifecycleState)) == string(opsisdk.AwrHubSourceLifecycleStateDeleting)
}

func awrHubSourceLifecycleDeleted(current opsisdk.AwrHubSource) bool {
	return strings.ToUpper(string(current.LifecycleState)) == string(opsisdk.AwrHubSourceLifecycleStateDeleted)
}

func currentAwrHubSourceID(resource *opsiv1beta1.AwrHubSource) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func handleAwrHubSourceDeleteError(resource *opsiv1beta1.AwrHubSource, err error) error {
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
	return awrHubSourceAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s delete path returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed", awrHubSourceKind),
		opcRequestID: requestID,
	}
}

func awrHubSourceIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func getAwrHubSourceWorkRequest(
	ctx context.Context,
	client awrHubSourceWorkRequestClient,
	initErr error,
	workRequestID string,
) (opsisdk.WorkRequest, error) {
	if initErr != nil {
		return opsisdk.WorkRequest{}, fmt.Errorf("initialize %s OCI client: %w", awrHubSourceKind, initErr)
	}
	if client == nil {
		return opsisdk.WorkRequest{}, fmt.Errorf("%s work request client is not configured", awrHubSourceKind)
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{WorkRequestId: common.String(workRequestID)})
	if err != nil {
		return opsisdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveAwrHubSourceWorkRequestAction(workRequest any) (string, error) {
	current, err := awrHubSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func recoverAwrHubSourceIDFromWorkRequest(
	_ *opsiv1beta1.AwrHubSource,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := awrHubSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	resourceID := awrHubSourceIDFromWorkRequest(current, phase)
	if resourceID == "" {
		return "", fmt.Errorf("%s %s work request %s did not expose an %s identifier", awrHubSourceKind, phase, awrHubSourceStringValue(current.Id), awrHubSourceKind)
	}
	return resourceID, nil
}

func awrHubSourceIDFromWorkRequest(workRequest opsisdk.WorkRequest, phase shared.OSOKAsyncPhase) string {
	action := awrHubSourceWorkRequestActionForPhase(phase)
	var candidate string
	for _, resource := range workRequest.Resources {
		if !isAwrHubSourceWorkRequestResource(resource) {
			continue
		}
		if action != "" && resource.ActionType != action && resource.ActionType != opsisdk.ActionTypeInProgress {
			continue
		}
		id := strings.TrimSpace(awrHubSourceStringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return ""
		}
	}
	return candidate
}

func awrHubSourceWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
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

func isAwrHubSourceWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(awrHubSourceStringValue(resource.EntityType)))
	normalizedEntityType := strings.NewReplacer("_", "", "-", "", " ", "").Replace(entityType)
	if strings.Contains(normalizedEntityType, "awrhubsource") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(awrHubSourceStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/awrhubsources/")
}

func awrHubSourceAsyncOperation(
	status *shared.OSOKStatus,
	workRequest opsisdk.WorkRequest,
	fallback shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	return servicemanager.BuildWorkRequestAsyncOperation(status, awrHubSourceWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        string(workRequest.OperationType),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    awrHubSourceStringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		Message:          awrHubSourceWorkRequestMessage(fallback, workRequest),
		FallbackPhase:    fallback,
	})
}

func awrHubSourceWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := awrHubSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", awrHubSourceKind, phase, awrHubSourceStringValue(current.Id), current.Status)
}

func awrHubSourceWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	workRequest = awrHubSourceDereference(workRequest)
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case opsisdk.GetWorkRequestResponse:
		return current.WorkRequest, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", awrHubSourceKind, workRequest)
	}
}

func awrHubSourceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		converted[namespace] = inner
	}
	return converted
}

func awrHubSourceDefinedTagsFromStatus(status map[string]shared.MapValue) map[string]map[string]interface{} {
	return awrHubSourceDefinedTagsFromSpec(status)
}

func awrHubSourceStatusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}

	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		inner := make(shared.MapValue, len(values))
		for key, value := range values {
			inner[key] = fmt.Sprint(value)
		}
		converted[namespace] = inner
	}
	return converted
}

func awrHubSourceDefinedTagsEqual(left map[string]map[string]interface{}, right map[string]map[string]interface{}) bool {
	if len(left) != len(right) {
		return false
	}
	for namespace, leftValues := range left {
		rightValues, ok := right[namespace]
		if !ok || len(leftValues) != len(rightValues) {
			return false
		}
		for key, leftValue := range leftValues {
			if fmt.Sprint(leftValue) != fmt.Sprint(rightValues[key]) {
				return false
			}
		}
	}
	return true
}

func awrHubSourceStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func awrHubSourceBoolValue(value *bool) bool {
	return value != nil && *value
}

func awrHubSourceFloat32Value(value *float32) float32 {
	if value == nil {
		return 0
	}
	return *value
}

func awrHubSourceFloat64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func awrHubSourceTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
