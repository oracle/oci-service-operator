/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
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

type dataSourceOCIClient interface {
	CreateDataSource(context.Context, cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error)
	GetDataSource(context.Context, cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error)
	ListDataSources(context.Context, cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error)
	UpdateDataSource(context.Context, cloudguardsdk.UpdateDataSourceRequest) (cloudguardsdk.UpdateDataSourceResponse, error)
	DeleteDataSource(context.Context, cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error)
	GetWorkRequest(context.Context, cloudguardsdk.GetWorkRequestRequest) (cloudguardsdk.GetWorkRequestResponse, error)
}

type dataSourceWorkRequestClient interface {
	GetWorkRequest(context.Context, cloudguardsdk.GetWorkRequestRequest) (cloudguardsdk.GetWorkRequestResponse, error)
}

func init() {
	registerDataSourceRuntimeHooksMutator(func(manager *DataSourceServiceManager, hooks *DataSourceRuntimeHooks) {
		applyDataSourceRuntimeHooks(manager, hooks)
	})
}

type dataSourceRuntimeClient struct {
	delegate DataSourceServiceClient
	hooks    DataSourceRuntimeHooks
	log      loggerutil.OSOKLogger
}

var _ DataSourceServiceClient = (*dataSourceRuntimeClient)(nil)

func applyDataSourceRuntimeHooks(manager *DataSourceServiceManager, hooks *DataSourceRuntimeHooks) {
	workRequestClient, initErr := newDataSourceWorkRequestClient(manager)
	applyDataSourceRuntimeHooksWithWorkRequestClient(manager, hooks, workRequestClient, initErr)
}

func newDataSourceWorkRequestClient(manager *DataSourceServiceManager) (dataSourceWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("DataSource service manager is nil")
	}
	return cloudguardsdk.NewCloudGuardClientWithConfigurationProvider(manager.Provider)
}

func applyDataSourceRuntimeHooksWithWorkRequestClient(
	manager *DataSourceServiceManager,
	hooks *DataSourceRuntimeHooks,
	workRequestClient dataSourceWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = dataSourceRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *cloudguardv1beta1.DataSource, _ string) (any, error) {
		return buildDataSourceCreateDetails(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *cloudguardv1beta1.DataSource, _ string, currentResponse any) (any, bool, error) {
		return buildDataSourceUpdateDetails(resource, currentResponse)
	}
	hooks.Read.Get = dataSourceStatusGetOperation(hooks)
	hooks.Read.List = dataSourcePaginatedListReadOperation(hooks)
	hooks.Async.Adapter = dataSourceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDataSourceWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveDataSourceWorkRequestAction
	hooks.Async.ResolvePhase = resolveDataSourceWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverDataSourceIDFromWorkRequest
	hooks.Async.Message = dataSourceWorkRequestMessage
	hooks.DeleteHooks.HandleError = rejectDataSourceAuthShapedNotFound
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DataSourceServiceClient) DataSourceServiceClient {
		runtimeClient := &dataSourceRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

var dataSourceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(cloudguardsdk.OperationStatusAccepted),
		string(cloudguardsdk.OperationStatusInProgress),
		string(cloudguardsdk.OperationStatusWaiting),
		string(cloudguardsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(cloudguardsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(cloudguardsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(cloudguardsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(cloudguardsdk.OperationTypeCreate), string(cloudguardsdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(cloudguardsdk.OperationTypeUpdate), string(cloudguardsdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(cloudguardsdk.OperationTypeDelete), string(cloudguardsdk.ActionTypeDeleted)},
}

func dataSourceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "cloudguard",
		FormalSlug:        "datasource",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update"},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(cloudguardsdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(cloudguardsdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(cloudguardsdk.LifecycleStateActive), string(cloudguardsdk.LifecycleStateInactive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(cloudguardsdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(cloudguardsdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "dataSourceFeedProvider"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"dataSourceDetails", "definedTags", "displayName", "freeformTags", "status"},
			ForceNew:      []string{"compartmentId", "dataSourceFeedProvider"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func (c *dataSourceRuntimeClient) CreateOrUpdate(ctx context.Context, resource *cloudguardv1beta1.DataSource, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("datasource runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

//nolint:gocyclo // Delete sequencing keeps pre-read, delete request, and confirm-read decisions together.
func (c *dataSourceRuntimeClient) Delete(ctx context.Context, resource *cloudguardv1beta1.DataSource) (bool, error) {
	if err := c.validateDataSourceDeleteRequest(resource); err != nil {
		return false, err
	}

	current, found, err := c.resolveDataSourceForDelete(ctx, resource)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if err := projectDataSourceStatus(resource, current); err != nil {
		return false, err
	}

	currentID := stringValue(current.Id)
	if dataSourceLifecycleState(current) == string(cloudguardsdk.LifecycleStateDeleted) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}

	if handled, err := c.handleDataSourceTrackedDeleteWorkRequest(ctx, resource, currentID); handled || err != nil {
		return false, err
	}

	if c.handleDataSourceDeleteLifecycle(resource, current) {
		return false, nil
	}

	response, err := c.hooks.Delete.Call(ctx, cloudguardsdk.DeleteDataSourceRequest{DataSourceId: ociString(currentID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if dataSourceIsUnambiguousNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, rejectDataSourceAuthShapedNotFound(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	confirmed, found, err := c.getDataSourceForDelete(ctx, resource, currentID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if err := projectDataSourceStatus(resource, confirmed); err != nil {
		return false, err
	}
	if strings.EqualFold(string(confirmed.LifecycleState), string(cloudguardsdk.LifecycleStateDeleted)) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}

	c.markTerminating(resource, "OCI resource delete is in progress", stringValue(response.OpcWorkRequestId))
	return false, nil
}

func (c *dataSourceRuntimeClient) validateDataSourceDeleteRequest(resource *cloudguardv1beta1.DataSource) error {
	if resource == nil {
		return fmt.Errorf("datasource resource is nil")
	}
	if c == nil {
		return fmt.Errorf("datasource runtime client is not configured")
	}
	return nil
}

func (c *dataSourceRuntimeClient) handleDataSourceDeleteLifecycle(
	resource *cloudguardv1beta1.DataSource,
	current cloudguardsdk.DataSource,
) bool {
	switch dataSourceLifecycleState(current) {
	case string(cloudguardsdk.LifecycleStateDeleting):
		c.markTerminating(resource, "OCI resource delete is in progress", "")
		return true
	case string(cloudguardsdk.LifecycleStateCreating), string(cloudguardsdk.LifecycleStateUpdating):
		c.markPendingWriteBeforeDelete(resource, current)
		return true
	default:
		return false
	}
}

func (c *dataSourceRuntimeClient) handleDataSourceTrackedDeleteWorkRequest(
	ctx context.Context,
	resource *cloudguardv1beta1.DataSource,
	dataSourceID string,
) (bool, error) {
	workRequestID, class, ok := dataSourceTrackedDeleteWorkRequest(resource)
	if !ok {
		return false, nil
	}
	if class == shared.OSOKAsyncClassSucceeded {
		c.markDeleteWorkRequestSucceeded(resource, workRequestID, dataSourceID)
		return true, nil
	}

	currentAsync, err := c.observeDataSourceDeleteWorkRequest(ctx, resource, workRequestID, dataSourceID)
	if err != nil {
		return true, err
	}
	if currentAsync == nil {
		return true, fmt.Errorf("DataSource delete work request %s did not project async status", workRequestID)
	}
	return true, nil
}

func (c *dataSourceRuntimeClient) observeDataSourceDeleteWorkRequest(
	ctx context.Context,
	resource *cloudguardv1beta1.DataSource,
	workRequestID string,
	dataSourceID string,
) (*shared.OSOKAsyncOperation, error) {
	if c.hooks.Async.GetWorkRequest == nil {
		return nil, fmt.Errorf("DataSource delete work request observation is not configured")
	}
	workRequest, err := c.hooks.Async.GetWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil, err
	}
	current, err := buildDataSourceWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return nil, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyDataSourceDeleteWorkRequestOperation(resource, current)
		return current, nil
	case shared.OSOKAsyncClassSucceeded:
		c.markObservedDeleteWorkRequestSucceeded(resource, current, dataSourceID)
		return current, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("DataSource delete work request %s finished with status %s", workRequestID, current.RawStatus)
		failed := *current
		failed.Message = err.Error()
		c.applyDataSourceDeleteWorkRequestOperation(resource, &failed)
		return &failed, err
	default:
		err := fmt.Errorf("DataSource delete work request %s projected unsupported async class %s", workRequestID, current.NormalizedClass)
		failed := *current
		failed.NormalizedClass = shared.OSOKAsyncClassFailed
		failed.Message = err.Error()
		c.applyDataSourceDeleteWorkRequestOperation(resource, &failed)
		return &failed, err
	}
}

func (c *dataSourceRuntimeClient) markObservedDeleteWorkRequestSucceeded(
	resource *cloudguardv1beta1.DataSource,
	current *shared.OSOKAsyncOperation,
	dataSourceID string,
) {
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassSucceeded
	next.Message = dataSourceDeleteWorkRequestSucceededMessage(next.WorkRequestID, dataSourceID)
	c.applyDataSourceDeleteWorkRequestOperation(resource, &next)
}

func (c *dataSourceRuntimeClient) markDeleteWorkRequestSucceeded(
	resource *cloudguardv1beta1.DataSource,
	workRequestID string,
	dataSourceID string,
) {
	c.applyDataSourceDeleteWorkRequestOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		RawStatus:       string(cloudguardsdk.OperationStatusSucceeded),
		NormalizedClass: shared.OSOKAsyncClassSucceeded,
		Message:         dataSourceDeleteWorkRequestSucceededMessage(workRequestID, dataSourceID),
	})
}

func (c *dataSourceRuntimeClient) applyDataSourceDeleteWorkRequestOperation(
	resource *cloudguardv1beta1.DataSource,
	current *shared.OSOKAsyncOperation,
) {
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func dataSourceDeleteWorkRequestSucceededMessage(workRequestID string, dataSourceID string) string {
	workRequestID = strings.TrimSpace(workRequestID)
	dataSourceID = strings.TrimSpace(dataSourceID)
	if dataSourceID == "" {
		return fmt.Sprintf("DataSource delete work request %s succeeded; waiting for delete confirmation", workRequestID)
	}
	return fmt.Sprintf("DataSource delete work request %s succeeded; waiting for DataSource %s to be deleted", workRequestID, dataSourceID)
}

func (c *dataSourceRuntimeClient) resolveDataSourceForDelete(
	ctx context.Context,
	resource *cloudguardv1beta1.DataSource,
) (cloudguardsdk.DataSource, bool, error) {
	if currentID := currentDataSourceID(resource); currentID != "" {
		return c.getDataSourceForDelete(ctx, resource, currentID)
	}
	return c.lookupDataSourceForDelete(ctx, resource)
}

func (c *dataSourceRuntimeClient) lookupDataSourceForDelete(
	ctx context.Context,
	resource *cloudguardv1beta1.DataSource,
) (cloudguardsdk.DataSource, bool, error) {
	if c.hooks.List.Call == nil {
		return cloudguardsdk.DataSource{}, false, nil
	}
	provider, err := dataSourceListFeedProvider(resource.Spec.DataSourceFeedProvider)
	if err != nil {
		return cloudguardsdk.DataSource{}, false, err
	}
	response, err := listDataSourcePages(ctx, c.hooks.List.Call, cloudguardsdk.ListDataSourcesRequest{
		CompartmentId:          ociString(resource.Spec.CompartmentId),
		DisplayName:            ociString(resource.Spec.DisplayName),
		DataSourceFeedProvider: provider,
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return cloudguardsdk.DataSource{}, false, rejectDataSourceAuthShapedNotFound(resource, err)
	}

	matches := make([]cloudguardsdk.DataSourceSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if dataSourceSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return cloudguardsdk.DataSource{}, false, nil
	case 1:
		return c.getDataSourceForDelete(ctx, resource, stringValue(matches[0].Id))
	default:
		return cloudguardsdk.DataSource{}, false, fmt.Errorf(
			"multiple OCI DataSources matched compartmentId %q, displayName %q, and dataSourceFeedProvider %q",
			resource.Spec.CompartmentId,
			resource.Spec.DisplayName,
			resource.Spec.DataSourceFeedProvider,
		)
	}
}

func (c *dataSourceRuntimeClient) getDataSourceForDelete(
	ctx context.Context,
	resource *cloudguardv1beta1.DataSource,
	dataSourceID string,
) (cloudguardsdk.DataSource, bool, error) {
	if dataSourceID == "" || c.hooks.Get.Call == nil {
		return cloudguardsdk.DataSource{}, false, nil
	}
	response, err := c.hooks.Get.Call(ctx, cloudguardsdk.GetDataSourceRequest{DataSourceId: ociString(dataSourceID)})
	if err != nil {
		if dataSourceIsUnambiguousNotFound(err) {
			return cloudguardsdk.DataSource{}, false, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return cloudguardsdk.DataSource{}, false, rejectDataSourceAuthShapedNotFound(resource, err)
	}
	return response.DataSource, true, nil
}

func (c *dataSourceRuntimeClient) markDeleted(resource *cloudguardv1beta1.DataSource, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *dataSourceRuntimeClient) markTerminating(resource *cloudguardv1beta1.DataSource, message string, workRequestID string) {
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	if strings.TrimSpace(workRequestID) != "" {
		current.Source = shared.OSOKAsyncSourceWorkRequest
		current.WorkRequestID = strings.TrimSpace(workRequestID)
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *dataSourceRuntimeClient) markPendingWriteBeforeDelete(
	resource *cloudguardv1beta1.DataSource,
	current cloudguardsdk.DataSource,
) {
	now := metav1.Now()
	lifecycleState := strings.ToUpper(string(current.LifecycleState))
	phase := shared.OSOKAsyncPhaseCreate
	if lifecycleState == string(cloudguardsdk.LifecycleStateUpdating) {
		phase = shared.OSOKAsyncPhaseUpdate
	}
	message := fmt.Sprintf("OCI resource is %s; retaining finalizer until the write completes", lifecycleState)
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       lifecycleState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}, c.log)
}

func dataSourceStatusGetOperation(hooks *DataSourceRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.Get.Call == nil {
		return nil
	}
	getCall := hooks.Get.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &cloudguardsdk.GetDataSourceRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*cloudguardsdk.GetDataSourceRequest)
			if !ok {
				return nil, fmt.Errorf("expected *cloudguard.GetDataSourceRequest, got %T", request)
			}
			response, err := getCall(ctx, *typed)
			if err != nil {
				return nil, err
			}
			return dataSourceReadResponse{
				DataSource:   dataSourceStatusAdapter{DataSource: response.DataSource},
				Etag:         response.Etag,
				OpcRequestId: response.OpcRequestId,
			}, nil
		},
	}
}

func dataSourcePaginatedListReadOperation(hooks *DataSourceRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}
	listCall := hooks.List.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &cloudguardsdk.ListDataSourcesRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*cloudguardsdk.ListDataSourcesRequest)
			if !ok {
				return nil, fmt.Errorf("expected *cloudguard.ListDataSourcesRequest, got %T", request)
			}
			response, err := listDataSourcePages(ctx, listCall, *typed)
			if err != nil {
				return nil, err
			}
			return adaptDataSourceListResponse(response), nil
		},
	}
}

type dataSourceReadResponse struct {
	DataSource   dataSourceStatusAdapter `presentIn:"body"`
	Etag         *string                 `presentIn:"header" name:"etag"`
	OpcRequestId *string                 `presentIn:"header" name:"opc-request-id"`
}

type dataSourceListResponse struct {
	DataSourceCollection dataSourceCollectionStatusAdapter `presentIn:"body"`
	OpcRequestId         *string                           `presentIn:"header" name:"opc-request-id"`
	OpcNextPage          *string                           `presentIn:"header" name:"opc-next-page"`
}

type dataSourceCollectionStatusAdapter struct {
	Items []dataSourceSummaryStatusAdapter `json:"items"`
	Locks []cloudguardsdk.ResourceLock     `json:"locks,omitempty"`
}

type dataSourceStatusAdapter struct {
	cloudguardsdk.DataSource
}

func (a dataSourceStatusAdapter) MarshalJSON() ([]byte, error) {
	return dataSourcePayloadWithSDKStatus(a.DataSource)
}

type dataSourceSummaryStatusAdapter struct {
	cloudguardsdk.DataSourceSummary
}

func (a dataSourceSummaryStatusAdapter) MarshalJSON() ([]byte, error) {
	return dataSourcePayloadWithSDKStatus(a.DataSourceSummary)
}

func adaptDataSourceListResponse(response cloudguardsdk.ListDataSourcesResponse) dataSourceListResponse {
	items := make([]dataSourceSummaryStatusAdapter, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, dataSourceSummaryStatusAdapter{DataSourceSummary: item})
	}
	return dataSourceListResponse{
		DataSourceCollection: dataSourceCollectionStatusAdapter{
			Items: items,
			Locks: response.Locks,
		},
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
}

func listDataSourcePages(
	ctx context.Context,
	call func(context.Context, cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error),
	request cloudguardsdk.ListDataSourcesRequest,
) (cloudguardsdk.ListDataSourcesResponse, error) {
	if call == nil {
		return cloudguardsdk.ListDataSourcesResponse{}, fmt.Errorf("datasource list operation is not configured")
	}
	if request.DataSourceFeedProvider != "" {
		provider, err := dataSourceListFeedProvider(string(request.DataSourceFeedProvider))
		if err != nil {
			return cloudguardsdk.ListDataSourcesResponse{}, err
		}
		request.DataSourceFeedProvider = provider
	}
	if request.LifecycleState != "" {
		return listDataSourcePagesForState(ctx, call, request)
	}

	var combined cloudguardsdk.ListDataSourcesResponse
	for _, state := range dataSourceLookupLifecycleStates() {
		stateRequest := request
		stateRequest.LifecycleState = state
		response, err := listDataSourcePagesForState(ctx, call, stateRequest)
		if err != nil {
			return cloudguardsdk.ListDataSourcesResponse{}, err
		}
		appendDataSourceListResponse(&combined, response)
	}
	return combined, nil
}

func listDataSourcePagesForState(
	ctx context.Context,
	call func(context.Context, cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error),
	request cloudguardsdk.ListDataSourcesRequest,
) (cloudguardsdk.ListDataSourcesResponse, error) {
	seenPages := map[string]struct{}{}
	var combined cloudguardsdk.ListDataSourcesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return cloudguardsdk.ListDataSourcesResponse{}, err
		}
		appendDataSourceListResponse(&combined, response)

		nextPage := stringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return cloudguardsdk.ListDataSourcesResponse{}, fmt.Errorf("datasource list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

func appendDataSourceListResponse(combined *cloudguardsdk.ListDataSourcesResponse, response cloudguardsdk.ListDataSourcesResponse) {
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
	combined.Locks = append(combined.Locks, response.Locks...)
}

func dataSourceLookupLifecycleStates() []cloudguardsdk.ListDataSourcesLifecycleStateEnum {
	return []cloudguardsdk.ListDataSourcesLifecycleStateEnum{
		cloudguardsdk.ListDataSourcesLifecycleStateCreating,
		cloudguardsdk.ListDataSourcesLifecycleStateUpdating,
		cloudguardsdk.ListDataSourcesLifecycleStateActive,
		cloudguardsdk.ListDataSourcesLifecycleStateInactive,
		cloudguardsdk.ListDataSourcesLifecycleStateDeleting,
		cloudguardsdk.ListDataSourcesLifecycleStateFailed,
	}
}

func buildDataSourceCreateDetails(resource *cloudguardv1beta1.DataSource) (cloudguardsdk.CreateDataSourceDetails, error) {
	if resource == nil {
		return cloudguardsdk.CreateDataSourceDetails{}, fmt.Errorf("datasource resource is nil")
	}
	spec := resource.Spec
	provider, err := dataSourceFeedProvider(spec.DataSourceFeedProvider)
	if err != nil {
		return cloudguardsdk.CreateDataSourceDetails{}, err
	}
	status, err := dataSourceStatus(spec.Status)
	if err != nil {
		return cloudguardsdk.CreateDataSourceDetails{}, err
	}
	details, err := buildDataSourceDetails(spec.DataSourceDetails, provider)
	if err != nil {
		return cloudguardsdk.CreateDataSourceDetails{}, err
	}
	return cloudguardsdk.CreateDataSourceDetails{
		DisplayName:            ociString(spec.DisplayName),
		CompartmentId:          ociString(spec.CompartmentId),
		DataSourceFeedProvider: provider,
		Status:                 status,
		DataSourceDetails:      details,
		FreeformTags:           maps.Clone(spec.FreeformTags),
		DefinedTags:            dataSourceDefinedTagsFromSpec(spec.DefinedTags),
	}, nil
}

//nolint:gocognit,gocyclo // Mutable field shaping intentionally mirrors the SDK update model for reviewability.
func buildDataSourceUpdateDetails(
	resource *cloudguardv1beta1.DataSource,
	currentResponse any,
) (cloudguardsdk.UpdateDataSourceDetails, bool, error) {
	if resource == nil {
		return cloudguardsdk.UpdateDataSourceDetails{}, false, fmt.Errorf("datasource resource is nil")
	}

	current, err := dataSourceRuntimeBody(currentResponse)
	if err != nil {
		return cloudguardsdk.UpdateDataSourceDetails{}, false, err
	}

	spec := resource.Spec
	details := cloudguardsdk.UpdateDataSourceDetails{}
	updateNeeded := false

	if spec.DisplayName != stringValue(current.DisplayName) {
		details.DisplayName = ociString(spec.DisplayName)
		updateNeeded = true
	}
	if strings.TrimSpace(spec.Status) != "" {
		status, err := dataSourceStatus(spec.Status)
		if err != nil {
			return cloudguardsdk.UpdateDataSourceDetails{}, false, err
		}
		if status != current.Status {
			details.Status = status
			updateNeeded = true
		}
	}
	if dataSourceDetailsMeaningful(spec.DataSourceDetails) {
		provider, err := dataSourceFeedProvider(spec.DataSourceFeedProvider)
		if err != nil {
			return cloudguardsdk.UpdateDataSourceDetails{}, false, err
		}
		desired, err := buildDataSourceDetails(spec.DataSourceDetails, provider)
		if err != nil {
			return cloudguardsdk.UpdateDataSourceDetails{}, false, err
		}
		if desired != nil && !dataSourceJSONEqual(desired, current.DataSourceDetails) {
			details.DataSourceDetails = desired
			updateNeeded = true
		}
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := dataSourceDefinedTagsFromSpec(spec.DefinedTags)
		if !reflect.DeepEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}

	return details, updateNeeded, nil
}

func buildDataSourceDetails(
	spec cloudguardv1beta1.DataSourceDetails,
	provider cloudguardsdk.DataSourceFeedProviderEnum,
) (cloudguardsdk.DataSourceDetails, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return dataSourceDetailsFromJSON(spec.JsonData, provider)
	}
	if strings.TrimSpace(spec.DataSourceFeedProvider) != "" {
		detailProvider, err := dataSourceFeedProvider(spec.DataSourceFeedProvider)
		if err != nil {
			return nil, err
		}
		if detailProvider != provider {
			return nil, fmt.Errorf("dataSourceDetails.dataSourceFeedProvider %q does not match dataSourceFeedProvider %q", spec.DataSourceFeedProvider, provider)
		}
	}

	switch provider {
	case cloudguardsdk.DataSourceFeedProviderLoggingquery:
		return buildLoggingQueryDataSourceDetails(spec)
	case cloudguardsdk.DataSourceFeedProviderScheduledquery:
		return buildScheduledQueryDataSourceDetails(spec), nil
	default:
		return nil, fmt.Errorf("dataSourceFeedProvider %q is unsupported", provider)
	}
}

func buildLoggingQueryDataSourceDetails(spec cloudguardv1beta1.DataSourceDetails) (cloudguardsdk.DataSourceDetails, error) {
	if !dataSourceDetailsMeaningful(spec) {
		return nil, nil
	}
	queryStartTime, err := buildDataSourceQueryStartTime(spec.QueryStartTime)
	if err != nil {
		return nil, err
	}
	loggingQueryDetails, err := buildDataSourceLoggingQueryDetails(spec.LoggingQueryDetails)
	if err != nil {
		return nil, err
	}
	operator, err := dataSourceLoggingQueryOperator(spec.Operator)
	if err != nil {
		return nil, err
	}
	loggingQueryType, err := dataSourceLoggingQueryType(spec.LoggingQueryType)
	if err != nil {
		return nil, err
	}
	return cloudguardsdk.LoggingQueryDataSourceDetails{
		Regions:                 append([]string(nil), spec.Regions...),
		Query:                   optionalString(spec.Query),
		IntervalInMinutes:       optionalInt(spec.IntervalInMinutes),
		Threshold:               optionalInt(spec.Threshold),
		QueryStartTime:          queryStartTime,
		AdditionalEntitiesCount: optionalInt(spec.AdditionalEntitiesCount),
		LoggingQueryDetails:     loggingQueryDetails,
		Operator:                operator,
		LoggingQueryType:        loggingQueryType,
	}, nil
}

func buildScheduledQueryDataSourceDetails(spec cloudguardv1beta1.DataSourceDetails) cloudguardsdk.DataSourceDetails {
	if !dataSourceDetailsMeaningful(spec) {
		return nil
	}
	return cloudguardsdk.ScheduledQueryDataSourceObjDetails{
		Query:                      optionalString(spec.Query),
		Description:                optionalString(spec.Description),
		IntervalInSeconds:          optionalInt(spec.IntervalInSeconds),
		ScheduledQueryScopeDetails: scheduledQueryScopeDetailsFromSpec(spec.ScheduledQueryScopeDetails),
	}
}

func buildDataSourceQueryStartTime(spec cloudguardv1beta1.DataSourceDetailsQueryStartTime) (cloudguardsdk.ContinuousQueryStartPolicy, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return dataSourceQueryStartTimeFromJSON(spec.JsonData)
	}
	policyType := strings.TrimSpace(spec.StartPolicyType)
	if policyType == "" && strings.TrimSpace(spec.QueryStartTime) != "" {
		policyType = string(cloudguardsdk.ContinuousQueryStartPolicyStartPolicyTypeAbsoluteTimeStartPolicy)
	}
	if policyType == "" {
		return nil, nil
	}
	policy, ok := cloudguardsdk.GetMappingContinuousQueryStartPolicyStartPolicyTypeEnum(policyType)
	if !ok {
		return nil, fmt.Errorf("queryStartTime.startPolicyType %q is unsupported", spec.StartPolicyType)
	}
	switch policy {
	case cloudguardsdk.ContinuousQueryStartPolicyStartPolicyTypeNoDelayStartPolicy:
		return cloudguardsdk.NoDelayStartPolicy{}, nil
	case cloudguardsdk.ContinuousQueryStartPolicyStartPolicyTypeAbsoluteTimeStartPolicy:
		parsed, err := optionalSDKTime("queryStartTime.queryStartTime", spec.QueryStartTime)
		if err != nil {
			return nil, err
		}
		return cloudguardsdk.AbsoluteTimeStartPolicy{QueryStartTime: parsed}, nil
	default:
		return nil, fmt.Errorf("queryStartTime.startPolicyType %q is unsupported", spec.StartPolicyType)
	}
}

func buildDataSourceLoggingQueryDetails(
	spec cloudguardv1beta1.DataSourceDetailsLoggingQueryDetails,
) (cloudguardsdk.LoggingQueryDetails, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return dataSourceLoggingQueryDetailsFromJSON(spec.JsonData)
	}
	loggingQueryType := strings.TrimSpace(spec.LoggingQueryType)
	if loggingQueryType == "" && spec.KeyEntitiesCount > 0 {
		loggingQueryType = string(cloudguardsdk.LoggingQueryTypeInsight)
	}
	if loggingQueryType == "" {
		return nil, nil
	}
	queryType, err := dataSourceLoggingQueryType(loggingQueryType)
	if err != nil {
		return nil, err
	}
	switch queryType {
	case cloudguardsdk.LoggingQueryTypeInsight:
		return cloudguardsdk.InsightTypeLoggingQueryDetails{KeyEntitiesCount: optionalInt(spec.KeyEntitiesCount)}, nil
	default:
		return nil, fmt.Errorf("loggingQueryDetails.loggingQueryType %q is unsupported", spec.LoggingQueryType)
	}
}

func dataSourceDetailsFromJSON(raw string, provider cloudguardsdk.DataSourceFeedProviderEnum) (cloudguardsdk.DataSourceDetails, error) {
	switch provider {
	case cloudguardsdk.DataSourceFeedProviderLoggingquery:
		var details cloudguardsdk.LoggingQueryDataSourceDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode logging query dataSourceDetails jsonData: %w", err)
		}
		return details, nil
	case cloudguardsdk.DataSourceFeedProviderScheduledquery:
		var details cloudguardsdk.ScheduledQueryDataSourceObjDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode scheduled query dataSourceDetails jsonData: %w", err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("dataSourceFeedProvider %q is unsupported", provider)
	}
}

func dataSourceQueryStartTimeFromJSON(raw string) (cloudguardsdk.ContinuousQueryStartPolicy, error) {
	var probe struct {
		StartPolicyType string `json:"startPolicyType"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("decode queryStartTime jsonData discriminator: %w", err)
	}
	policy, ok := cloudguardsdk.GetMappingContinuousQueryStartPolicyStartPolicyTypeEnum(probe.StartPolicyType)
	if !ok {
		return nil, fmt.Errorf("queryStartTime.startPolicyType %q is unsupported", probe.StartPolicyType)
	}
	switch policy {
	case cloudguardsdk.ContinuousQueryStartPolicyStartPolicyTypeNoDelayStartPolicy:
		var details cloudguardsdk.NoDelayStartPolicy
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode no-delay queryStartTime jsonData: %w", err)
		}
		return details, nil
	case cloudguardsdk.ContinuousQueryStartPolicyStartPolicyTypeAbsoluteTimeStartPolicy:
		var details cloudguardsdk.AbsoluteTimeStartPolicy
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode absolute queryStartTime jsonData: %w", err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("queryStartTime.startPolicyType %q is unsupported", probe.StartPolicyType)
	}
}

func dataSourceLoggingQueryDetailsFromJSON(raw string) (cloudguardsdk.LoggingQueryDetails, error) {
	var probe struct {
		LoggingQueryType string `json:"loggingQueryType"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("decode loggingQueryDetails jsonData discriminator: %w", err)
	}
	queryType, err := dataSourceLoggingQueryType(probe.LoggingQueryType)
	if err != nil {
		return nil, err
	}
	switch queryType {
	case cloudguardsdk.LoggingQueryTypeInsight:
		var details cloudguardsdk.InsightTypeLoggingQueryDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode insight loggingQueryDetails jsonData: %w", err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("loggingQueryDetails.loggingQueryType %q is unsupported", probe.LoggingQueryType)
	}
}

//nolint:gocyclo // DataSourceDetails spans both Cloud Guard provider variants and nested discriminator fields.
func dataSourceDetailsMeaningful(spec cloudguardv1beta1.DataSourceDetails) bool {
	return strings.TrimSpace(spec.JsonData) != "" ||
		strings.TrimSpace(spec.Query) != "" ||
		strings.TrimSpace(spec.Description) != "" ||
		spec.IntervalInSeconds != 0 ||
		len(spec.ScheduledQueryScopeDetails) > 0 ||
		len(spec.Regions) > 0 ||
		spec.IntervalInMinutes != 0 ||
		spec.Threshold != 0 ||
		dataSourceQueryStartTimeMeaningful(spec.QueryStartTime) ||
		spec.AdditionalEntitiesCount != 0 ||
		dataSourceLoggingQueryDetailsMeaningful(spec.LoggingQueryDetails) ||
		strings.TrimSpace(spec.Operator) != "" ||
		strings.TrimSpace(spec.LoggingQueryType) != ""
}

func dataSourceQueryStartTimeMeaningful(spec cloudguardv1beta1.DataSourceDetailsQueryStartTime) bool {
	return strings.TrimSpace(spec.JsonData) != "" ||
		strings.TrimSpace(spec.StartPolicyType) != "" ||
		strings.TrimSpace(spec.QueryStartTime) != ""
}

func dataSourceLoggingQueryDetailsMeaningful(spec cloudguardv1beta1.DataSourceDetailsLoggingQueryDetails) bool {
	return strings.TrimSpace(spec.JsonData) != "" ||
		strings.TrimSpace(spec.LoggingQueryType) != "" ||
		spec.KeyEntitiesCount != 0
}

func scheduledQueryScopeDetailsFromSpec(
	spec []cloudguardv1beta1.DataSourceDetailsScheduledQueryScopeDetail,
) []cloudguardsdk.ScheduledQueryScopeDetail {
	if spec == nil {
		return nil
	}
	converted := make([]cloudguardsdk.ScheduledQueryScopeDetail, 0, len(spec))
	for _, item := range spec {
		converted = append(converted, cloudguardsdk.ScheduledQueryScopeDetail{
			Region:       optionalString(item.Region),
			ResourceIds:  append([]string(nil), item.ResourceIds...),
			ResourceType: optionalString(item.ResourceType),
		})
	}
	return converted
}

//nolint:gocyclo // Response normalization must accept each SDK wrapper shape used by generatedruntime.
func dataSourceRuntimeBody(currentResponse any) (cloudguardsdk.DataSource, error) {
	switch current := currentResponse.(type) {
	case nil:
		return cloudguardsdk.DataSource{}, nil
	case cloudguardsdk.DataSource:
		return current, nil
	case *cloudguardsdk.DataSource:
		if current == nil {
			return cloudguardsdk.DataSource{}, fmt.Errorf("current datasource response is nil")
		}
		return *current, nil
	case dataSourceStatusAdapter:
		return current.DataSource, nil
	case *dataSourceStatusAdapter:
		if current == nil {
			return cloudguardsdk.DataSource{}, fmt.Errorf("current datasource response is nil")
		}
		return current.DataSource, nil
	case cloudguardsdk.DataSourceSummary:
		return dataSourceFromSummary(current), nil
	case *cloudguardsdk.DataSourceSummary:
		if current == nil {
			return cloudguardsdk.DataSource{}, fmt.Errorf("current datasource summary response is nil")
		}
		return dataSourceFromSummary(*current), nil
	case dataSourceSummaryStatusAdapter:
		return dataSourceFromSummary(current.DataSourceSummary), nil
	case *dataSourceSummaryStatusAdapter:
		if current == nil {
			return cloudguardsdk.DataSource{}, fmt.Errorf("current datasource summary response is nil")
		}
		return dataSourceFromSummary(current.DataSourceSummary), nil
	case cloudguardsdk.GetDataSourceResponse:
		return current.DataSource, nil
	case *cloudguardsdk.GetDataSourceResponse:
		if current == nil {
			return cloudguardsdk.DataSource{}, fmt.Errorf("current get datasource response is nil")
		}
		return current.DataSource, nil
	case dataSourceReadResponse:
		return current.DataSource.DataSource, nil
	case *dataSourceReadResponse:
		if current == nil {
			return cloudguardsdk.DataSource{}, fmt.Errorf("current datasource read response is nil")
		}
		return current.DataSource.DataSource, nil
	default:
		return cloudguardsdk.DataSource{}, fmt.Errorf("unexpected current datasource response type %T", currentResponse)
	}
}

func dataSourceFromSummary(summary cloudguardsdk.DataSourceSummary) cloudguardsdk.DataSource {
	return cloudguardsdk.DataSource{
		Id:                     summary.Id,
		DisplayName:            summary.DisplayName,
		DataSourceFeedProvider: summary.DataSourceFeedProvider,
		CompartmentId:          summary.CompartmentId,
		TimeCreated:            summary.TimeCreated,
		TimeUpdated:            summary.TimeUpdated,
		Status:                 summary.Status,
		LifecycleState:         summary.LifecycleState,
		FreeformTags:           summary.FreeformTags,
		DefinedTags:            summary.DefinedTags,
		SystemTags:             summary.SystemTags,
	}
}

func projectDataSourceStatus(resource *cloudguardv1beta1.DataSource, response any) error {
	if resource == nil {
		return fmt.Errorf("datasource resource is nil")
	}
	payload, ok, err := dataSourceStatusPayload(response)
	if err != nil || !ok {
		return err
	}
	projected := resource.Status
	projected.OsokStatus = resource.Status.OsokStatus
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal datasource status payload: %w", err)
	}
	if err := json.Unmarshal(raw, &projected); err != nil {
		return fmt.Errorf("project datasource status: %w", err)
	}
	projected.OsokStatus = resource.Status.OsokStatus
	if projected.Id != "" {
		projected.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	resource.Status = projected
	return nil
}

func dataSourceStatusPayload(response any) (map[string]any, bool, error) {
	body, ok, err := dataSourceStatusBody(response)
	if err != nil || !ok {
		return nil, ok, err
	}
	raw, err := dataSourcePayloadWithSDKStatus(body)
	if err != nil {
		return nil, true, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, true, fmt.Errorf("decode datasource status payload: %w", err)
	}
	return payload, true, nil
}

//nolint:gocyclo // Status projection must accept each SDK wrapper shape used by generatedruntime.
func dataSourceStatusBody(response any) (any, bool, error) {
	switch current := response.(type) {
	case nil:
		return nil, false, nil
	case cloudguardsdk.DataSource, cloudguardsdk.DataSourceSummary, dataSourceStatusAdapter, dataSourceSummaryStatusAdapter:
		return current, true, nil
	case *cloudguardsdk.DataSource:
		if current == nil {
			return nil, false, fmt.Errorf("datasource status body is nil")
		}
		return *current, true, nil
	case *cloudguardsdk.DataSourceSummary:
		if current == nil {
			return nil, false, fmt.Errorf("datasource summary status body is nil")
		}
		return *current, true, nil
	case *dataSourceStatusAdapter:
		if current == nil {
			return nil, false, fmt.Errorf("datasource status body is nil")
		}
		return *current, true, nil
	case *dataSourceSummaryStatusAdapter:
		if current == nil {
			return nil, false, fmt.Errorf("datasource summary status body is nil")
		}
		return *current, true, nil
	case cloudguardsdk.GetDataSourceResponse:
		return current.DataSource, true, nil
	case *cloudguardsdk.GetDataSourceResponse:
		if current == nil {
			return nil, false, fmt.Errorf("get datasource response is nil")
		}
		return current.DataSource, true, nil
	case dataSourceReadResponse:
		return current.DataSource, true, nil
	case *dataSourceReadResponse:
		if current == nil {
			return nil, false, fmt.Errorf("datasource read response is nil")
		}
		return current.DataSource, true, nil
	default:
		return nil, false, fmt.Errorf("unexpected datasource status response type %T", response)
	}
}

func dataSourcePayloadWithSDKStatus(body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal datasource status source: %w", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode datasource status source: %w", err)
	}
	if sdkStatus, ok := payload["status"]; ok {
		payload["sdkStatus"] = sdkStatus
		delete(payload, "status")
	}
	return json.Marshal(payload)
}

func currentDataSourceID(resource *cloudguardv1beta1.DataSource) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func dataSourceTrackedDeleteWorkRequest(
	resource *cloudguardv1beta1.DataSource,
) (string, shared.OSOKAsyncNormalizedClass, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", "", false
	}
	current := resource.Status.OsokStatus.Async.Current
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		workRequestID == "" {
		return "", "", false
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending, shared.OSOKAsyncClassSucceeded:
		return workRequestID, current.NormalizedClass, true
	default:
		return "", current.NormalizedClass, false
	}
}

func getDataSourceWorkRequest(
	ctx context.Context,
	client dataSourceWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize DataSource work request OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("DataSource work request OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, cloudguardsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func buildDataSourceWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := dataSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}

	derivedPhase, ok, err := resolveDataSourceWorkRequestPhase(current)
	if err != nil {
		return nil, err
	}
	if ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf(
				"DataSource work request %s exposes phase %q while delete expected %q",
				stringValue(current.Id),
				derivedPhase,
				fallbackPhase,
			)
		}
		fallbackPhase = derivedPhase
	}
	rawAction, err := resolveDataSourceWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	operation, err := servicemanager.BuildWorkRequestAsyncOperation(status, dataSourceWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        rawAction,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    stringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := dataSourceWorkRequestMessage(operation.Phase, current); message != "" {
		operation.Message = message
	}
	return operation, nil
}

func resolveDataSourceWorkRequestAction(workRequest any) (string, error) {
	current, err := dataSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveDataSourceWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := dataSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case cloudguardsdk.OperationTypeCreate:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case cloudguardsdk.OperationTypeUpdate:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case cloudguardsdk.OperationTypeDelete:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverDataSourceIDFromWorkRequest(
	_ *cloudguardv1beta1.DataSource,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := dataSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if id, ok := resolveDataSourceIDFromWorkRequestResources(current.Resources, phase, true); ok {
		return id, nil
	}
	if id, ok := resolveDataSourceIDFromWorkRequestResources(current.Resources, phase, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("DataSource work request %s does not expose a datasource identifier", stringValue(current.Id))
}

func resolveDataSourceIDFromWorkRequestResources(
	resources []cloudguardsdk.WorkRequestResource,
	phase shared.OSOKAsyncPhase,
	requireActionMatch bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if !isDataSourceWorkRequestResource(resource) {
			continue
		}
		if requireActionMatch && !dataSourceWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		id := stringValue(resource.Identifier)
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

func isDataSourceWorkRequestResource(resource cloudguardsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.ReplaceAll(stringValue(resource.EntityType), " ", ""))
	if strings.Contains(entityType, "datasource") {
		return true
	}
	entityURI := strings.ToLower(stringValue(resource.EntityUri))
	return strings.Contains(entityURI, "/datasources/")
}

func dataSourceWorkRequestActionMatchesPhase(action cloudguardsdk.ActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == cloudguardsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return action == cloudguardsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return action == cloudguardsdk.ActionTypeDeleted
	default:
		return false
	}
}

func dataSourceWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := dataSourceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("DataSource %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func dataSourceWorkRequestFromAny(workRequest any) (cloudguardsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case cloudguardsdk.WorkRequest:
		return current, nil
	case *cloudguardsdk.WorkRequest:
		if current == nil {
			return cloudguardsdk.WorkRequest{}, fmt.Errorf("DataSource work request is nil")
		}
		return *current, nil
	default:
		return cloudguardsdk.WorkRequest{}, fmt.Errorf("unexpected DataSource work request type %T", workRequest)
	}
}

func dataSourceSummaryMatchesSpec(summary cloudguardsdk.DataSourceSummary, spec cloudguardv1beta1.DataSourceSpec) bool {
	provider, err := dataSourceFeedProvider(spec.DataSourceFeedProvider)
	if err != nil {
		return false
	}
	return stringValue(summary.CompartmentId) == spec.CompartmentId &&
		stringValue(summary.DisplayName) == spec.DisplayName &&
		summary.DataSourceFeedProvider == provider
}

func dataSourceLifecycleState(current cloudguardsdk.DataSource) string {
	return strings.ToUpper(string(current.LifecycleState))
}

func rejectDataSourceAuthShapedNotFound(resource *cloudguardv1beta1.DataSource, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("datasource delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted")
}

func dataSourceIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func dataSourceFeedProvider(value string) (cloudguardsdk.DataSourceFeedProviderEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("dataSourceFeedProvider is required")
	}
	provider, ok := cloudguardsdk.GetMappingDataSourceFeedProviderEnum(trimmed)
	if !ok {
		return "", fmt.Errorf("dataSourceFeedProvider %q is unsupported", value)
	}
	return provider, nil
}

func dataSourceListFeedProvider(value string) (cloudguardsdk.ListDataSourcesDataSourceFeedProviderEnum, error) {
	provider, err := dataSourceFeedProvider(value)
	if err != nil {
		return "", err
	}
	return cloudguardsdk.ListDataSourcesDataSourceFeedProviderEnum(provider), nil
}

func dataSourceStatus(value string) (cloudguardsdk.DataSourceStatusEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	status, ok := cloudguardsdk.GetMappingDataSourceStatusEnum(trimmed)
	if !ok {
		return "", fmt.Errorf("status %q is unsupported", value)
	}
	return status, nil
}

func dataSourceLoggingQueryOperator(value string) (cloudguardsdk.LoggingQueryOperatorTypeEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	operator, ok := cloudguardsdk.GetMappingLoggingQueryOperatorTypeEnum(trimmed)
	if !ok {
		return "", fmt.Errorf("operator %q is unsupported", value)
	}
	return operator, nil
}

func dataSourceLoggingQueryType(value string) (cloudguardsdk.LoggingQueryTypeEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	queryType, ok := cloudguardsdk.GetMappingLoggingQueryTypeEnum(trimmed)
	if !ok {
		return "", fmt.Errorf("loggingQueryType %q is unsupported", value)
	}
	return queryType, nil
}

func dataSourceDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		convertedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			convertedValues[key] = value
		}
		converted[namespace] = convertedValues
	}
	return converted
}

func optionalSDKTime(fieldName string, value string) (*common.SDKTime, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339: %w", fieldName, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func dataSourceJSONEqual(left any, right any) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftRaw) == string(rightRaw)
}

func ociString(value string) *string {
	return common.String(value)
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func optionalInt(value int) *int {
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

func newDataSourceRuntimeHooksWithOCIClient(client dataSourceOCIClient) DataSourceRuntimeHooks {
	hooks := newDataSourceDefaultRuntimeHooks(cloudguardsdk.CloudGuardClient{})
	if client == nil {
		return hooks
	}
	hooks.Create.Call = func(ctx context.Context, request cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error) {
		return client.CreateDataSource(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		return client.GetDataSource(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error) {
		return client.ListDataSources(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request cloudguardsdk.UpdateDataSourceRequest) (cloudguardsdk.UpdateDataSourceResponse, error) {
		return client.UpdateDataSource(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error) {
		return client.DeleteDataSource(ctx, request)
	}
	return hooks
}
