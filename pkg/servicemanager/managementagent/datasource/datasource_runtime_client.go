/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasource

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
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
	dataSourceManagementAgentIDAnnotation       = "managementagent.oracle.com/management-agent-id"
	dataSourceLegacyManagementAgentIDAnnotation = "managementagent.oracle.com/managementAgentId"

	dataSourceActiveMessage        = "OCI data source is active"
	dataSourceCreatePendingMessage = "OCI data source create is in progress"
	dataSourceUpdatePendingMessage = "OCI data source update is in progress"
	dataSourceDeletePendingMessage = "OCI data source delete is in progress"

	dataSourceRequeueDuration = time.Minute
)

type dataSourceRuntimeOCIClient interface {
	CreateDataSource(context.Context, managementagentsdk.CreateDataSourceRequest) (managementagentsdk.CreateDataSourceResponse, error)
	GetDataSource(context.Context, managementagentsdk.GetDataSourceRequest) (managementagentsdk.GetDataSourceResponse, error)
	ListDataSources(context.Context, managementagentsdk.ListDataSourcesRequest) (managementagentsdk.ListDataSourcesResponse, error)
	UpdateDataSource(context.Context, managementagentsdk.UpdateDataSourceRequest) (managementagentsdk.UpdateDataSourceResponse, error)
	DeleteDataSource(context.Context, managementagentsdk.DeleteDataSourceRequest) (managementagentsdk.DeleteDataSourceResponse, error)
	GetWorkRequest(context.Context, managementagentsdk.GetWorkRequestRequest) (managementagentsdk.GetWorkRequestResponse, error)
}

type dataSourceSDKOCIClient struct {
	client  managementagentsdk.ManagementAgentClient
	initErr error
}

type dataSourceRuntimeClient struct {
	client dataSourceRuntimeOCIClient
	log    loggerutil.OSOKLogger
}

type dataSourceIdentity struct {
	managementAgentID string
	key               string
	name              string
	dataSourceType    string
}

type dataSourceSnapshot struct {
	key               string
	name              string
	compartmentID     string
	state             string
	timeCreated       *common.SDKTime
	timeUpdated       *common.SDKTime
	dataSourceType    string
	namespace         string
	isDaemonSet       bool
	url               string
	allowMetrics      string
	proxyURL          string
	connectionTimeout int
	readTimeout       int
	readDataLimit     int
	scheduleMins      int
	resourceGroup     string
	metricDimensions  []managementagentv1beta1.DataSourceMetricDimension
}

var _ DataSourceServiceClient = (*dataSourceRuntimeClient)(nil)

var dataSourceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(managementagentsdk.OperationStatusCreated),
		string(managementagentsdk.OperationStatusAccepted),
		string(managementagentsdk.OperationStatusInProgress),
		string(managementagentsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(managementagentsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(managementagentsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(managementagentsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(managementagentsdk.OperationTypesCreateDataSource)},
	UpdateActionTokens:    []string{string(managementagentsdk.OperationTypesUpdateDataSource)},
	DeleteActionTokens:    []string{string(managementagentsdk.OperationTypesDeleteDataSource)},
}

func init() {
	registerDataSourceRuntimeHooksMutator(func(manager *DataSourceServiceManager, hooks *DataSourceRuntimeHooks) {
		applyDataSourceRuntimeHooks(manager, hooks)
	})
}

func applyDataSourceRuntimeHooks(manager *DataSourceServiceManager, hooks *DataSourceRuntimeHooks) {
	if hooks == nil {
		return
	}

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}

	hooks.Semantics = newDataSourceRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(_ DataSourceServiceClient) DataSourceServiceClient {
		return newDataSourceRuntimeClient(newDataSourceSDKOCIClient(manager), log)
	})
}

func newDataSourceSDKOCIClient(manager *DataSourceServiceManager) dataSourceSDKOCIClient {
	if manager == nil {
		return dataSourceSDKOCIClient{initErr: fmt.Errorf("DataSource service manager is nil")}
	}
	client, err := managementagentsdk.NewManagementAgentClientWithConfigurationProvider(manager.Provider)
	return dataSourceSDKOCIClient{
		client:  client,
		initErr: err,
	}
}

func newDataSourceRuntimeClient(client dataSourceRuntimeOCIClient, log loggerutil.OSOKLogger) DataSourceServiceClient {
	return &dataSourceRuntimeClient{
		client: client,
		log:    log,
	}
}

func newDataSourceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "managementagent",
		FormalSlug:        "datasource",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "handwritten",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(managementagentsdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(managementagentsdk.LifecycleStatesUpdating)},
			ActiveStates: []string{
				string(managementagentsdk.LifecycleStatesActive),
				string(managementagentsdk.LifecycleStatesInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(managementagentsdk.LifecycleStatesDeleting)},
			TerminalStates: []string{string(managementagentsdk.LifecycleStatesDeleted), string(managementagentsdk.LifecycleStatesTerminated)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"managementAgentId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"url",
				"allowMetrics",
				"proxyUrl",
				"connectionTimeout",
				"readTimeout",
				"readDataLimitInKilobytes",
				"scheduleMins",
				"resourceGroup",
				"metricDimensions",
			},
			ForceNew: []string{"managementAgentId", "name", "compartmentId", "type", "namespace"},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local DataSource runtime"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local DataSource runtime"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local DataSource runtime"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> paginated ListDataSources/GetDataSource readback",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local DataSource runtime", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetDataSource readback",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local DataSource runtime", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> conservative confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local DataSource runtime", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{{
			Phase:            "create/update/delete",
			MethodName:       "GetWorkRequest",
			RequestTypeName:  "GetWorkRequestRequest",
			ResponseTypeName: "GetWorkRequestResponse",
		}},
		Unsupported: []generatedruntime.UnsupportedSemantic{{
			Category:      "crd-shape",
			StopCondition: "managementAgentId is supplied through metadata annotation because the v1beta1 DataSource spec has no parent management agent field",
		}, {
			Category:      "polymorphic-body",
			StopCondition: "only PROMETHEUS_EMITTER create/update bodies are exposed by the OCI SDK",
		}},
	}
}

func (c *dataSourceRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	identity, err := c.createOrUpdateIdentity(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if response, handled, err := c.resumePendingDataSourceWrite(ctx, resource, identity); handled || err != nil {
		return response, err
	}

	current, err := c.lookupDataSource(ctx, identity)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	return c.reconcileObservedDataSource(ctx, resource, identity, current)
}

func (c *dataSourceRuntimeClient) createOrUpdateIdentity(
	resource *managementagentv1beta1.DataSource,
) (dataSourceIdentity, error) {
	identity, err := resolveDataSourceIdentity(resource)
	if err != nil {
		return dataSourceIdentity{}, c.fail(resource, err)
	}
	if c.client == nil {
		return dataSourceIdentity{}, c.fail(resource, fmt.Errorf("DataSource OCI client is not configured"))
	}
	return identity, nil
}

func (c *dataSourceRuntimeClient) resumePendingDataSourceWrite(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) (servicemanager.OSOKResponse, bool, error) {
	pendingPhase, workRequestID, writePending := dataSourcePendingWrite(resource)
	if !writePending {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if pendingPhase == shared.OSOKAsyncPhaseDelete {
		err := fmt.Errorf("DataSource delete is still active during CreateOrUpdate")
		return servicemanager.OSOKResponse{IsSuccessful: false}, true, c.fail(resource, err)
	}
	return c.observePendingWorkRequest(ctx, resource, identity, pendingPhase, workRequestID)
}

func (c *dataSourceRuntimeClient) reconcileObservedDataSource(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	current *dataSourceSnapshot,
) (servicemanager.OSOKResponse, error) {
	_, _, writePending := dataSourcePendingWrite(resource)
	if current == nil {
		if writePending {
			return c.resumeMissingPendingDataSource(resource, identity), nil
		}
		return c.createDataSource(ctx, resource, identity)
	}
	identity.key = current.key
	if err := validateDataSourceCreateOnlyDrift(resource, identity, *current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	if dataSourceStateBlocksMutation(current.state) {
		return c.markObserved(resource, identity, *current), nil
	}
	if dataSourceNeedsUpdate(resource, *current) {
		return c.updateDataSource(ctx, resource, identity, *current)
	}
	return c.markObserved(resource, identity, *current), nil
}

func (c *dataSourceRuntimeClient) resumeMissingPendingDataSource(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) servicemanager.OSOKResponse {
	pendingPhase, workRequestID, _ := dataSourcePendingWrite(resource)
	return c.markPending(
		resource,
		identity,
		pendingPhase,
		string(managementagentsdk.OperationStatusAccepted),
		workRequestID,
		shared.OSOKAsyncSourceWorkRequest,
	)
}

func (c *dataSourceRuntimeClient) Delete(ctx context.Context, resource *managementagentv1beta1.DataSource) (bool, error) {
	identity, hasTrackedIdentity, err := c.deleteIdentity(resource)
	if err != nil {
		return false, err
	}
	if !hasTrackedIdentity {
		c.markDeleted(resource, "OCI data source was never tracked")
		return true, nil
	}

	pendingPhase, _, writePending := dataSourcePendingWrite(resource)
	if writePending {
		switch pendingPhase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			ready, err := c.resumePendingDataSourceWriteBeforeDelete(ctx, resource, identity)
			if err != nil || !ready {
				return false, err
			}
		case shared.OSOKAsyncPhaseDelete:
			return c.continueDataSourceDelete(ctx, resource, identity)
		}
	}

	if dataSourceDeleteAlreadyPending(resource) {
		return c.continueDataSourceDelete(ctx, resource, identity)
	}
	return c.startDataSourceDelete(ctx, resource, identity)
}

func (c *dataSourceRuntimeClient) deleteIdentity(resource *managementagentv1beta1.DataSource) (dataSourceIdentity, bool, error) {
	identity, hasTrackedIdentity, err := resolveDataSourceDeleteIdentity(resource)
	if err != nil {
		return dataSourceIdentity{}, false, c.fail(resource, err)
	}
	if !hasTrackedIdentity {
		return dataSourceIdentity{}, false, nil
	}
	if c.client == nil {
		return dataSourceIdentity{}, false, c.fail(resource, fmt.Errorf("DataSource OCI client is not configured"))
	}
	return identity, true, nil
}

func (c *dataSourceRuntimeClient) continueDataSourceDelete(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) (bool, error) {
	workRequestID := strings.TrimSpace(dataSourceCurrentWorkRequestID(resource))
	if workRequestID != "" {
		handled, err := c.observePendingDeleteWorkRequest(ctx, resource, identity, workRequestID)
		if handled || err != nil {
			return false, err
		}
	}
	return c.confirmDataSourceDelete(ctx, resource, identity, workRequestID, c.failPendingObservation)
}

func (c *dataSourceRuntimeClient) startDataSourceDelete(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) (bool, error) {
	current, err := c.lookupDataSourceForDelete(ctx, identity)
	if err != nil {
		return false, c.fail(resource, err)
	}
	if current == nil {
		c.markDeleted(resource, "OCI data source deleted")
		return true, nil
	}
	identity.key = current.key
	if current.state == string(managementagentsdk.LifecycleStatesDeleting) {
		c.markTerminating(resource, identity, "")
		return false, nil
	}

	response, err := c.client.DeleteDataSource(ctx, managementagentsdk.DeleteDataSourceRequest{
		ManagementAgentId: common.String(identity.managementAgentID),
		DataSourceKey:     common.String(identity.key),
	})
	if err != nil {
		return c.handleDataSourceDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	if workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId)); workRequestID != "" {
		c.markTerminating(resource, identity, workRequestID)
		return false, nil
	}

	return c.confirmDataSourceDelete(ctx, resource, identity, "", c.failPendingObservation)
}

func (c *dataSourceRuntimeClient) resumePendingDataSourceWriteBeforeDelete(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) (bool, error) {
	pendingPhase, workRequestID, writePending := dataSourcePendingWrite(resource)
	if !writePending {
		return true, nil
	}
	if pendingPhase == shared.OSOKAsyncPhaseDelete {
		return true, nil
	}
	if strings.TrimSpace(workRequestID) == "" {
		return c.resumeLifecyclePendingDataSourceWriteBeforeDelete(ctx, resource, identity)
	}

	_, handled, err := c.observePendingWorkRequest(ctx, resource, identity, pendingPhase, workRequestID)
	if err != nil {
		return false, err
	}
	return !handled, nil
}

func (c *dataSourceRuntimeClient) resumeLifecyclePendingDataSourceWriteBeforeDelete(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) (bool, error) {
	current, err := c.lookupDataSourceForDelete(ctx, identity)
	if err != nil {
		return false, c.failPendingObservation(resource, err)
	}
	if current == nil {
		return true, nil
	}
	identity.key = current.key
	if dataSourceLifecycleWriteStillPending(current.state) {
		c.markObserved(resource, identity, *current)
		return false, nil
	}
	if current.state == string(managementagentsdk.LifecycleStatesDeleting) {
		c.markTerminating(resource, identity, "")
		return false, nil
	}
	return true, nil
}

func (c *dataSourceRuntimeClient) handleDataSourceDeleteError(
	resource *managementagentv1beta1.DataSource,
	err error,
) (bool, error) {
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		c.markDeleted(resource, "OCI data source deleted")
		return true, nil
	case classification.IsAuthShapedNotFound():
		return false, c.fail(resource, fmt.Errorf("DataSource delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err))
	default:
		return false, c.fail(resource, err)
	}
}

func (c *dataSourceRuntimeClient) confirmDataSourceDelete(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	workRequestID string,
	fail func(*managementagentv1beta1.DataSource, error) error,
) (bool, error) {
	current, err := c.lookupDataSourceForDelete(ctx, identity)
	if err != nil {
		return false, fail(resource, err)
	}
	if current == nil {
		c.markDeleted(resource, "OCI data source deleted")
		return true, nil
	}
	identity.key = current.key
	c.markTerminating(resource, identity, strings.TrimSpace(workRequestID))
	return false, nil
}

func (c *dataSourceRuntimeClient) createDataSource(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
) (servicemanager.OSOKResponse, error) {
	details, err := buildDataSourceCreateDetails(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	response, err := c.client.CreateDataSource(ctx, managementagentsdk.CreateDataSourceRequest{
		ManagementAgentId:       common.String(identity.managementAgentID),
		CreateDataSourceDetails: details,
		OpcRetryToken:           dataSourceRetryToken(resource),
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	return c.markPending(
		resource,
		identity,
		shared.OSOKAsyncPhaseCreate,
		string(managementagentsdk.OperationStatusAccepted),
		stringValue(response.OpcWorkRequestId),
		"",
	), nil
}

func (c *dataSourceRuntimeClient) updateDataSource(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	current dataSourceSnapshot,
) (servicemanager.OSOKResponse, error) {
	details, err := buildDataSourceUpdateDetails(resource, current)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	response, err := c.client.UpdateDataSource(ctx, managementagentsdk.UpdateDataSourceRequest{
		ManagementAgentId:       common.String(identity.managementAgentID),
		DataSourceKey:           common.String(identity.key),
		UpdateDataSourceDetails: details,
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	return c.markPending(
		resource,
		identity,
		shared.OSOKAsyncPhaseUpdate,
		string(managementagentsdk.OperationStatusAccepted),
		stringValue(response.OpcWorkRequestId),
		"",
	), nil
}

func (c *dataSourceRuntimeClient) observePendingWorkRequest(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) (servicemanager.OSOKResponse, bool, error) {
	current, err := c.buildWorkRequestOperation(ctx, resource, phase, workRequestID)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, true, c.failPendingObservation(resource, err)
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
		err := fmt.Errorf("DataSource %s work request %s finished with status %s", current.Phase, workRequestID, current.RawStatus)
		response := c.markFailedWorkRequestOperation(resource, identity, current, err)
		return response, true, err
	default:
		err := fmt.Errorf("DataSource %s work request %s projected unsupported async class %s", current.Phase, workRequestID, current.NormalizedClass)
		response := c.markFailedWorkRequestOperation(resource, identity, current, err)
		return response, true, err
	}
}

func (c *dataSourceRuntimeClient) observePendingDeleteWorkRequest(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	workRequestID string,
) (bool, error) {
	response, handled, err := c.observePendingWorkRequest(ctx, resource, identity, shared.OSOKAsyncPhaseDelete, workRequestID)
	_ = response
	return handled, err
}

func (c *dataSourceRuntimeClient) buildWorkRequestOperation(
	ctx context.Context,
	resource *managementagentv1beta1.DataSource,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) (*shared.OSOKAsyncOperation, error) {
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return nil, nil
	}

	response, err := c.client.GetWorkRequest(ctx, managementagentsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return nil, err
	}

	workRequest := response.WorkRequest
	return servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, dataSourceWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        string(workRequest.OperationType),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    firstNonEmptyTrim(stringValue(workRequest.Id), workRequestID),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    phase,
	})
}

func (c *dataSourceRuntimeClient) lookupDataSource(
	ctx context.Context,
	identity dataSourceIdentity,
) (*dataSourceSnapshot, error) {
	if identity.key != "" {
		current, err := c.getDataSource(ctx, identity)
		if err == nil {
			if current == nil || current.isTerminalDeleted() {
				return nil, nil
			}
			return current, nil
		}
		if !errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return nil, err
		}
	}
	return c.lookupDataSourceByName(ctx, identity, false)
}

func (c *dataSourceRuntimeClient) lookupDataSourceForDelete(
	ctx context.Context,
	identity dataSourceIdentity,
) (*dataSourceSnapshot, error) {
	if identity.key != "" {
		current, err := c.getDataSource(ctx, identity)
		if err == nil {
			if current == nil || current.isTerminalDeleted() {
				return nil, nil
			}
			return current, nil
		}
		classification := errorutil.ClassifyDeleteError(err)
		switch {
		case classification.IsUnambiguousNotFound():
			return nil, nil
		case classification.IsAuthShapedNotFound():
			return nil, fmt.Errorf("DataSource delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
		default:
			return nil, err
		}
	}
	return c.lookupDataSourceByName(ctx, identity, true)
}

func (c *dataSourceRuntimeClient) getDataSource(
	ctx context.Context,
	identity dataSourceIdentity,
) (*dataSourceSnapshot, error) {
	response, err := c.client.GetDataSource(ctx, managementagentsdk.GetDataSourceRequest{
		ManagementAgentId: common.String(identity.managementAgentID),
		DataSourceKey:     common.String(identity.key),
	})
	if err != nil {
		return nil, err
	}
	current, ok := snapshotFromDataSource(response.DataSource)
	if !ok {
		return nil, fmt.Errorf("DataSource get response did not include a data source")
	}
	return &current, nil
}

func (c *dataSourceRuntimeClient) lookupDataSourceByName(
	ctx context.Context,
	identity dataSourceIdentity,
	forDelete bool,
) (*dataSourceSnapshot, error) {
	if strings.TrimSpace(identity.name) == "" {
		return nil, nil
	}

	request := managementagentsdk.ListDataSourcesRequest{
		ManagementAgentId: common.String(identity.managementAgentID),
		Name:              []string{identity.name},
	}
	for {
		response, err := c.listDataSourcePage(ctx, request, forDelete)
		if err != nil {
			return nil, err
		}
		current, matched, err := c.dataSourceFromListPage(ctx, identity, response.Items)
		if err != nil || matched {
			return current, err
		}
		nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
		if nextPage == "" {
			return nil, nil
		}
		request.Page = common.String(nextPage)
	}
}

func (c *dataSourceRuntimeClient) listDataSourcePage(
	ctx context.Context,
	request managementagentsdk.ListDataSourcesRequest,
	forDelete bool,
) (managementagentsdk.ListDataSourcesResponse, error) {
	response, err := c.client.ListDataSources(ctx, request)
	if err == nil {
		return response, nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if forDelete && classification.IsUnambiguousNotFound() {
		return managementagentsdk.ListDataSourcesResponse{}, nil
	}
	if classification.IsAuthShapedNotFound() {
		return managementagentsdk.ListDataSourcesResponse{}, fmt.Errorf("DataSource list returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as not found: %w", err)
	}
	return managementagentsdk.ListDataSourcesResponse{}, err
}

func (c *dataSourceRuntimeClient) dataSourceFromListPage(
	ctx context.Context,
	identity dataSourceIdentity,
	items []managementagentsdk.DataSourceSummary,
) (*dataSourceSnapshot, bool, error) {
	for _, item := range items {
		current, matched, err := c.dataSourceFromListSummary(ctx, identity, item)
		if err != nil || matched {
			return current, matched, err
		}
	}
	return nil, false, nil
}

func (c *dataSourceRuntimeClient) dataSourceFromListSummary(
	ctx context.Context,
	identity dataSourceIdentity,
	item managementagentsdk.DataSourceSummary,
) (*dataSourceSnapshot, bool, error) {
	summary, ok := snapshotFromDataSourceSummary(item)
	if !ok || summary.name != identity.name {
		return nil, false, nil
	}
	if dataSourceSummaryTypeMismatch(identity, summary) {
		return nil, false, nil
	}
	if summary.key == "" {
		return nil, true, fmt.Errorf("DataSource list found %q without a dataSourceKey", identity.name)
	}
	identity.key = summary.key
	current, err := c.getDataSource(ctx, identity)
	if err != nil {
		return nil, false, dataSourceReadAfterListError(err)
	}
	if current == nil || current.isTerminalDeleted() {
		return nil, true, nil
	}
	return current, true, nil
}

func dataSourceSummaryTypeMismatch(identity dataSourceIdentity, summary dataSourceSnapshot) bool {
	return identity.dataSourceType != "" &&
		summary.dataSourceType != "" &&
		summary.dataSourceType != identity.dataSourceType
}

func dataSourceReadAfterListError(err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsUnambiguousNotFound() {
		return nil
	}
	if classification.IsAuthShapedNotFound() {
		return fmt.Errorf("DataSource read after list returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as not found: %w", err)
	}
	return err
}

func resolveDataSourceIdentity(resource *managementagentv1beta1.DataSource) (dataSourceIdentity, error) {
	if resource == nil {
		return dataSourceIdentity{}, fmt.Errorf("DataSource resource is nil")
	}
	dataSourceType, err := validateDataSourceIdentitySpec(resource)
	if err != nil {
		return dataSourceIdentity{}, err
	}

	trackedManagementAgentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	annotationManagementAgentID := dataSourceAnnotationManagementAgentID(resource)
	if err := validateDataSourceTrackedIdentity(resource, trackedManagementAgentID, annotationManagementAgentID, dataSourceType); err != nil {
		return dataSourceIdentity{}, err
	}

	managementAgentID := firstNonEmptyTrim(trackedManagementAgentID, annotationManagementAgentID)
	if managementAgentID == "" {
		return dataSourceIdentity{}, fmt.Errorf("DataSource metadata annotation %q is required because the CRD has no spec managementAgentId field", dataSourceManagementAgentIDAnnotation)
	}

	return dataSourceIdentity{
		managementAgentID: managementAgentID,
		key:               strings.TrimSpace(resource.Status.Key),
		name:              firstNonEmptyTrim(resource.Status.Name, resource.Spec.Name),
		dataSourceType:    dataSourceType,
	}, nil
}

func validateDataSourceIdentitySpec(resource *managementagentv1beta1.DataSource) (string, error) {
	normalizeDataSourceSpec(resource)
	if resource.Spec.Name == "" {
		return "", fmt.Errorf("DataSource spec.name is required")
	}
	if resource.Spec.CompartmentId == "" {
		return "", fmt.Errorf("DataSource spec.compartmentId is required")
	}
	return normalizedDataSourceType(resource.Spec.Type)
}

func validateDataSourceTrackedIdentity(
	resource *managementagentv1beta1.DataSource,
	trackedManagementAgentID string,
	annotationManagementAgentID string,
	dataSourceType string,
) error {
	if trackedManagementAgentID != "" &&
		annotationManagementAgentID != "" &&
		trackedManagementAgentID != annotationManagementAgentID {
		return fmt.Errorf("DataSource formal semantics require replacement when managementAgentId changes")
	}
	if statusName := strings.TrimSpace(resource.Status.Name); statusName != "" && statusName != resource.Spec.Name {
		return fmt.Errorf("DataSource formal semantics require replacement when name changes")
	}
	if statusType := strings.TrimSpace(resource.Status.Type); statusType != "" && statusType != dataSourceType {
		return fmt.Errorf("DataSource formal semantics require replacement when type changes")
	}
	return nil
}

func resolveDataSourceDeleteIdentity(resource *managementagentv1beta1.DataSource) (dataSourceIdentity, bool, error) {
	if resource == nil {
		return dataSourceIdentity{}, false, fmt.Errorf("DataSource resource is nil")
	}
	normalizeDataSourceSpec(resource)

	statusKey := strings.TrimSpace(resource.Status.Key)
	statusName := strings.TrimSpace(resource.Status.Name)
	if statusKey == "" && statusName == "" {
		return dataSourceIdentity{}, false, nil
	}

	managementAgentID := firstNonEmptyTrim(
		string(resource.Status.OsokStatus.Ocid),
		dataSourceAnnotationManagementAgentID(resource),
	)
	if managementAgentID == "" {
		return dataSourceIdentity{}, true, fmt.Errorf("DataSource metadata annotation %q is required because the CRD has no spec managementAgentId field", dataSourceManagementAgentIDAnnotation)
	}
	return dataSourceIdentity{
		managementAgentID: managementAgentID,
		key:               statusKey,
		name:              firstNonEmptyTrim(statusName, resource.Spec.Name),
		dataSourceType:    normalizedDataSourceDeleteType(firstNonEmptyTrim(resource.Status.Type, resource.Spec.Type)),
	}, true, nil
}

func normalizedDataSourceDeleteType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mapped, ok := managementagentsdk.GetMappingDataSourceTypesEnum(value)
	if !ok {
		return ""
	}
	return string(mapped)
}

func normalizeDataSourceSpec(resource *managementagentv1beta1.DataSource) {
	if resource == nil {
		return
	}
	resource.Spec.Name = strings.TrimSpace(resource.Spec.Name)
	resource.Spec.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	resource.Spec.Type = strings.TrimSpace(resource.Spec.Type)
	resource.Spec.Url = strings.TrimSpace(resource.Spec.Url)
	resource.Spec.Namespace = strings.TrimSpace(resource.Spec.Namespace)
	resource.Spec.AllowMetrics = strings.TrimSpace(resource.Spec.AllowMetrics)
	resource.Spec.ProxyUrl = strings.TrimSpace(resource.Spec.ProxyUrl)
	resource.Spec.ResourceGroup = strings.TrimSpace(resource.Spec.ResourceGroup)
	for i := range resource.Spec.MetricDimensions {
		resource.Spec.MetricDimensions[i].Name = strings.TrimSpace(resource.Spec.MetricDimensions[i].Name)
		resource.Spec.MetricDimensions[i].Value = strings.TrimSpace(resource.Spec.MetricDimensions[i].Value)
	}
}

func normalizedDataSourceType(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return string(managementagentsdk.DataSourceTypesPrometheusEmitter), nil
	}
	mapped, ok := managementagentsdk.GetMappingDataSourceTypesEnum(value)
	if !ok {
		return "", fmt.Errorf("DataSource spec.type %q is not supported", value)
	}
	if mapped != managementagentsdk.DataSourceTypesPrometheusEmitter {
		return "", fmt.Errorf("DataSource spec.type %q is not supported by create/update; only PROMETHEUS_EMITTER is supported", value)
	}
	return string(mapped), nil
}

func dataSourceAnnotationManagementAgentID(resource *managementagentv1beta1.DataSource) string {
	if resource == nil {
		return ""
	}
	annotations := resource.GetAnnotations()
	return firstNonEmptyTrim(
		annotations[dataSourceManagementAgentIDAnnotation],
		annotations[dataSourceLegacyManagementAgentIDAnnotation],
	)
}

func validateDataSourceCreateOnlyDrift(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	current dataSourceSnapshot,
) error {
	if current.name != "" && current.name != resource.Spec.Name {
		return fmt.Errorf("DataSource create-only field name changed from %q to %q", current.name, resource.Spec.Name)
	}
	if current.compartmentID != "" && current.compartmentID != resource.Spec.CompartmentId {
		return fmt.Errorf("DataSource create-only field compartmentId changed from %q to %q", current.compartmentID, resource.Spec.CompartmentId)
	}
	if current.dataSourceType != "" && current.dataSourceType != identity.dataSourceType {
		return fmt.Errorf("DataSource create-only field type changed from %q to %q", current.dataSourceType, identity.dataSourceType)
	}
	if (current.namespace != "" || resource.Spec.Namespace != "") && current.namespace != resource.Spec.Namespace {
		return fmt.Errorf("DataSource create-only field namespace changed from %q to %q", current.namespace, resource.Spec.Namespace)
	}
	return nil
}

func buildDataSourceCreateDetails(resource *managementagentv1beta1.DataSource) (managementagentsdk.CreateDataSourceDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("DataSource resource is nil")
	}
	if _, err := normalizedDataSourceType(resource.Spec.Type); err != nil {
		return nil, err
	}
	if resource.Spec.Name == "" {
		return nil, fmt.Errorf("DataSource spec.name is required")
	}
	if resource.Spec.CompartmentId == "" {
		return nil, fmt.Errorf("DataSource spec.compartmentId is required")
	}
	if resource.Spec.Url == "" {
		return nil, fmt.Errorf("DataSource spec.url is required for PROMETHEUS_EMITTER")
	}
	if resource.Spec.Namespace == "" {
		return nil, fmt.Errorf("DataSource spec.namespace is required for PROMETHEUS_EMITTER")
	}

	return managementagentsdk.CreatePrometheusEmitterDataSourceDetails{
		Name:                     common.String(resource.Spec.Name),
		CompartmentId:            common.String(resource.Spec.CompartmentId),
		Url:                      common.String(resource.Spec.Url),
		Namespace:                common.String(resource.Spec.Namespace),
		AllowMetrics:             stringPtrIfNonEmpty(resource.Spec.AllowMetrics),
		ProxyUrl:                 stringPtrIfNonEmpty(resource.Spec.ProxyUrl),
		ConnectionTimeout:        intPtrIfPositive(resource.Spec.ConnectionTimeout),
		ReadTimeout:              intPtrIfPositive(resource.Spec.ReadTimeout),
		ReadDataLimitInKilobytes: intPtrIfPositive(resource.Spec.ReadDataLimitInKilobytes),
		ScheduleMins:             intPtrIfPositive(resource.Spec.ScheduleMins),
		ResourceGroup:            stringPtrIfNonEmpty(resource.Spec.ResourceGroup),
		MetricDimensions:         dataSourceSDKMetricDimensions(resource.Spec.MetricDimensions),
	}, nil
}

func buildDataSourceUpdateDetails(
	resource *managementagentv1beta1.DataSource,
	current dataSourceSnapshot,
) (managementagentsdk.UpdateDataSourceDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("DataSource resource is nil")
	}
	desired := dataSourceDesiredMutableFields(resource, current)
	if desired.url == "" {
		return nil, fmt.Errorf("DataSource spec.url or observed url is required for update")
	}
	return managementagentsdk.UpdatePrometheusEmitterDataSourceDetails{
		Url:                      common.String(desired.url),
		AllowMetrics:             stringPtrIfNonEmpty(desired.allowMetrics),
		ProxyUrl:                 stringPtrIfNonEmpty(desired.proxyURL),
		ConnectionTimeout:        intPtrIfPositive(desired.connectionTimeout),
		ReadTimeout:              intPtrIfPositive(desired.readTimeout),
		ReadDataLimitInKilobytes: intPtrIfPositive(desired.readDataLimitInKilobytes),
		ScheduleMins:             intPtrIfPositive(desired.scheduleMins),
		ResourceGroup:            stringPtrIfNonEmpty(desired.resourceGroup),
		MetricDimensions:         dataSourceSDKMetricDimensions(desired.metricDimensions),
	}, nil
}

func dataSourceNeedsUpdate(resource *managementagentv1beta1.DataSource, current dataSourceSnapshot) bool {
	if resource == nil {
		return false
	}
	desired := dataSourceDesiredMutableFields(resource, current)
	observed := dataSourceObservedMutableFields(current)
	return dataSourceStringFieldsNeedUpdate(desired, observed) ||
		dataSourceIntFieldsNeedUpdate(desired, observed) ||
		!slices.EqualFunc(desired.metricDimensions, observed.metricDimensions, dataSourceMetricDimensionEqual)
}

type dataSourceMutableFields struct {
	url                      string
	allowMetrics             string
	proxyURL                 string
	connectionTimeout        int
	readTimeout              int
	readDataLimitInKilobytes int
	scheduleMins             int
	resourceGroup            string
	metricDimensions         []managementagentv1beta1.DataSourceMetricDimension
}

func dataSourceDesiredMutableFields(
	resource *managementagentv1beta1.DataSource,
	current dataSourceSnapshot,
) dataSourceMutableFields {
	return dataSourceMutableFields{
		url:                      firstNonEmptyTrim(resource.Spec.Url, current.url),
		allowMetrics:             firstNonEmptyTrim(resource.Spec.AllowMetrics, current.allowMetrics),
		proxyURL:                 firstNonEmptyTrim(resource.Spec.ProxyUrl, current.proxyURL),
		connectionTimeout:        firstPositiveInt(resource.Spec.ConnectionTimeout, current.connectionTimeout),
		readTimeout:              firstPositiveInt(resource.Spec.ReadTimeout, current.readTimeout),
		readDataLimitInKilobytes: firstPositiveInt(resource.Spec.ReadDataLimitInKilobytes, current.readDataLimit),
		scheduleMins:             firstPositiveInt(resource.Spec.ScheduleMins, current.scheduleMins),
		resourceGroup:            firstNonEmptyTrim(resource.Spec.ResourceGroup, current.resourceGroup),
		metricDimensions:         firstNonEmptyMetricDimensions(resource.Spec.MetricDimensions, current.metricDimensions),
	}
}

func dataSourceObservedMutableFields(current dataSourceSnapshot) dataSourceMutableFields {
	return dataSourceMutableFields{
		url:                      current.url,
		allowMetrics:             current.allowMetrics,
		proxyURL:                 current.proxyURL,
		connectionTimeout:        current.connectionTimeout,
		readTimeout:              current.readTimeout,
		readDataLimitInKilobytes: current.readDataLimit,
		scheduleMins:             current.scheduleMins,
		resourceGroup:            current.resourceGroup,
		metricDimensions:         cloneDataSourceMetricDimensions(current.metricDimensions),
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyMetricDimensions(values ...[]managementagentv1beta1.DataSourceMetricDimension) []managementagentv1beta1.DataSourceMetricDimension {
	for _, value := range values {
		if len(value) > 0 {
			return cloneDataSourceMetricDimensions(value)
		}
	}
	return nil
}

func dataSourceStringFieldsNeedUpdate(desired dataSourceMutableFields, observed dataSourceMutableFields) bool {
	return desired.url != observed.url ||
		desired.allowMetrics != observed.allowMetrics ||
		desired.proxyURL != observed.proxyURL ||
		desired.resourceGroup != observed.resourceGroup
}

func dataSourceIntFieldsNeedUpdate(desired dataSourceMutableFields, observed dataSourceMutableFields) bool {
	return desired.connectionTimeout != observed.connectionTimeout ||
		desired.readTimeout != observed.readTimeout ||
		desired.readDataLimitInKilobytes != observed.readDataLimitInKilobytes ||
		desired.scheduleMins != observed.scheduleMins
}

func dataSourceStateBlocksMutation(state string) bool {
	switch state {
	case string(managementagentsdk.LifecycleStatesCreating),
		string(managementagentsdk.LifecycleStatesUpdating),
		string(managementagentsdk.LifecycleStatesDeleting),
		string(managementagentsdk.LifecycleStatesFailed):
		return true
	default:
		return false
	}
}

func dataSourceLifecycleWriteStillPending(state string) bool {
	switch state {
	case string(managementagentsdk.LifecycleStatesCreating), string(managementagentsdk.LifecycleStatesUpdating):
		return true
	default:
		return false
	}
}

func (c *dataSourceRuntimeClient) markObserved(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	current dataSourceSnapshot,
) servicemanager.OSOKResponse {
	identity.key = firstNonEmptyTrim(current.key, identity.key)
	recordDataSourcePathIdentity(resource, identity)
	projectDataSourceStatus(resource, current)

	switch current.state {
	case string(managementagentsdk.LifecycleStatesCreating):
		return c.markPending(resource, identity, shared.OSOKAsyncPhaseCreate, current.state, "", shared.OSOKAsyncSourceLifecycle)
	case string(managementagentsdk.LifecycleStatesUpdating):
		return c.markPending(resource, identity, shared.OSOKAsyncPhaseUpdate, current.state, "", shared.OSOKAsyncSourceLifecycle)
	case string(managementagentsdk.LifecycleStatesDeleting):
		c.markTerminating(resource, identity, "")
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: dataSourceRequeueDuration}
	case string(managementagentsdk.LifecycleStatesFailed):
		err := fmt.Errorf("DataSource is in OCI lifecycle state FAILED")
		_ = c.fail(resource, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	default:
		return c.markActive(resource, identity, current)
	}
}

func (c *dataSourceRuntimeClient) markActive(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	current dataSourceSnapshot,
) servicemanager.OSOKResponse {
	recordDataSourcePathIdentity(resource, identity)
	projectDataSourceStatus(resource, current)

	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = dataSourceActiveMessage
	status.Reason = string(shared.Active)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", dataSourceActiveMessage, c.log)

	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *dataSourceRuntimeClient) markPending(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	phase shared.OSOKAsyncPhase,
	rawState string,
	workRequestID string,
	source shared.OSOKAsyncSource,
) servicemanager.OSOKResponse {
	recordDataSourcePathIdentity(resource, identity)
	if source == "" {
		source = shared.OSOKAsyncSourceLifecycle
		if workRequestID != "" {
			source = shared.OSOKAsyncSourceWorkRequest
		}
	}

	message := dataSourceCreatePendingMessage
	if phase == shared.OSOKAsyncPhaseUpdate {
		message = dataSourceUpdatePendingMessage
	}
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          source,
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
		RequeueDuration: dataSourceRequeueDuration,
	}
}

func (c *dataSourceRuntimeClient) markTerminating(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	workRequestID string,
) {
	recordDataSourcePathIdentity(resource, identity)
	source := shared.OSOKAsyncSourceWorkRequest
	rawStatus := string(managementagentsdk.OperationStatusAccepted)
	if strings.TrimSpace(workRequestID) == "" {
		source = shared.OSOKAsyncSourceLifecycle
		rawStatus = string(managementagentsdk.LifecycleStatesDeleting)
	}
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          source,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         dataSourceDeletePendingMessage,
		UpdatedAt:       &now,
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *dataSourceRuntimeClient) markWorkRequestOperation(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	recordDataSourcePathIdentity(resource, identity)
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: dataSourceRequeueDuration,
	}
}

func (c *dataSourceRuntimeClient) markFailedWorkRequestOperation(
	resource *managementagentv1beta1.DataSource,
	identity dataSourceIdentity,
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

func (c *dataSourceRuntimeClient) markDeleted(resource *managementagentv1beta1.DataSource, message string) {
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

func (c *dataSourceRuntimeClient) fail(resource *managementagentv1beta1.DataSource, err error) error {
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

func (c *dataSourceRuntimeClient) failPendingObservation(resource *managementagentv1beta1.DataSource, err error) error {
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

func recordDataSourcePathIdentity(resource *managementagentv1beta1.DataSource, identity dataSourceIdentity) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(identity.managementAgentID)
	if identity.key != "" {
		resource.Status.Key = identity.key
	}
	if identity.name != "" {
		resource.Status.Name = identity.name
	}
	if identity.dataSourceType != "" {
		resource.Status.Type = identity.dataSourceType
	}
}

func projectDataSourceStatus(resource *managementagentv1beta1.DataSource, current dataSourceSnapshot) {
	if resource == nil {
		return
	}
	resource.Status.Key = current.key
	resource.Status.Name = current.name
	resource.Status.CompartmentId = current.compartmentID
	resource.Status.State = current.state
	resource.Status.TimeCreated = sdkTimeString(current.timeCreated)
	resource.Status.TimeUpdated = sdkTimeString(current.timeUpdated)
	resource.Status.Type = current.dataSourceType
	resource.Status.Namespace = current.namespace
	resource.Status.IsDaemonSet = current.isDaemonSet
	resource.Status.Url = current.url
	resource.Status.AllowMetrics = current.allowMetrics
	resource.Status.ProxyUrl = current.proxyURL
	resource.Status.ConnectionTimeout = current.connectionTimeout
	resource.Status.ReadTimeout = current.readTimeout
	resource.Status.ReadDataLimit = current.readDataLimit
	resource.Status.ScheduleMins = current.scheduleMins
	resource.Status.ResourceGroup = current.resourceGroup
	resource.Status.MetricDimensions = cloneDataSourceMetricDimensions(current.metricDimensions)
}

func snapshotFromDataSource(current managementagentsdk.DataSource) (dataSourceSnapshot, bool) {
	if current == nil {
		return dataSourceSnapshot{}, false
	}
	switch value := current.(type) {
	case managementagentsdk.PrometheusEmitterDataSource:
		return snapshotFromPrometheusEmitter(value), true
	case *managementagentsdk.PrometheusEmitterDataSource:
		if value == nil {
			return dataSourceSnapshot{}, false
		}
		return snapshotFromPrometheusEmitter(*value), true
	case managementagentsdk.KubernetesClusterDataSource:
		return snapshotFromKubernetesCluster(value), true
	case *managementagentsdk.KubernetesClusterDataSource:
		if value == nil {
			return dataSourceSnapshot{}, false
		}
		return snapshotFromKubernetesCluster(*value), true
	default:
		return dataSourceSnapshot{
			key:           stringValue(current.GetKey()),
			name:          stringValue(current.GetName()),
			compartmentID: stringValue(current.GetCompartmentId()),
			state:         string(current.GetState()),
			timeCreated:   current.GetTimeCreated(),
			timeUpdated:   current.GetTimeUpdated(),
		}, true
	}
}

func snapshotFromPrometheusEmitter(current managementagentsdk.PrometheusEmitterDataSource) dataSourceSnapshot {
	return dataSourceSnapshot{
		key:               stringValue(current.Key),
		name:              stringValue(current.Name),
		compartmentID:     stringValue(current.CompartmentId),
		state:             string(current.State),
		timeCreated:       current.TimeCreated,
		timeUpdated:       current.TimeUpdated,
		dataSourceType:    string(managementagentsdk.DataSourceTypesPrometheusEmitter),
		namespace:         stringValue(current.Namespace),
		url:               stringValue(current.Url),
		allowMetrics:      stringValue(current.AllowMetrics),
		proxyURL:          stringValue(current.ProxyUrl),
		connectionTimeout: intValue(current.ConnectionTimeout),
		readTimeout:       intValue(current.ReadTimeout),
		readDataLimit:     intValue(current.ReadDataLimit),
		scheduleMins:      intValue(current.ScheduleMins),
		resourceGroup:     stringValue(current.ResourceGroup),
		metricDimensions:  dataSourceStatusMetricDimensions(current.MetricDimensions),
	}
}

func snapshotFromKubernetesCluster(current managementagentsdk.KubernetesClusterDataSource) dataSourceSnapshot {
	return dataSourceSnapshot{
		key:            stringValue(current.Key),
		name:           stringValue(current.Name),
		compartmentID:  stringValue(current.CompartmentId),
		state:          string(current.State),
		timeCreated:    current.TimeCreated,
		timeUpdated:    current.TimeUpdated,
		dataSourceType: string(managementagentsdk.DataSourceTypesKubernetesCluster),
		namespace:      stringValue(current.Namespace),
		isDaemonSet:    boolValue(current.IsDaemonSet),
	}
}

func snapshotFromDataSourceSummary(current managementagentsdk.DataSourceSummary) (dataSourceSnapshot, bool) {
	if current == nil {
		return dataSourceSnapshot{}, false
	}
	switch value := current.(type) {
	case managementagentsdk.PrometheusEmitterDataSourceSummary:
		return dataSourceSnapshot{
			key:            stringValue(value.Key),
			name:           stringValue(value.Name),
			dataSourceType: string(managementagentsdk.DataSourceTypesPrometheusEmitter),
		}, true
	case *managementagentsdk.PrometheusEmitterDataSourceSummary:
		if value == nil {
			return dataSourceSnapshot{}, false
		}
		return dataSourceSnapshot{
			key:            stringValue(value.Key),
			name:           stringValue(value.Name),
			dataSourceType: string(managementagentsdk.DataSourceTypesPrometheusEmitter),
		}, true
	case managementagentsdk.KubernetesClusterDataSourceSummary:
		return dataSourceSnapshot{
			key:            stringValue(value.Key),
			name:           stringValue(value.Name),
			dataSourceType: string(managementagentsdk.DataSourceTypesKubernetesCluster),
		}, true
	case *managementagentsdk.KubernetesClusterDataSourceSummary:
		if value == nil {
			return dataSourceSnapshot{}, false
		}
		return dataSourceSnapshot{
			key:            stringValue(value.Key),
			name:           stringValue(value.Name),
			dataSourceType: string(managementagentsdk.DataSourceTypesKubernetesCluster),
		}, true
	default:
		return dataSourceSnapshot{
			key:  stringValue(current.GetKey()),
			name: stringValue(current.GetName()),
		}, true
	}
}

func (s dataSourceSnapshot) isTerminalDeleted() bool {
	switch s.state {
	case string(managementagentsdk.LifecycleStatesDeleted), string(managementagentsdk.LifecycleStatesTerminated):
		return true
	default:
		return false
	}
}

func dataSourcePendingWrite(resource *managementagentv1beta1.DataSource) (shared.OSOKAsyncPhase, string, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", "", false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return "", "", false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncPhaseDelete:
		return current.Phase, strings.TrimSpace(current.WorkRequestID), true
	default:
		return "", "", false
	}
}

func dataSourceDeleteAlreadyPending(resource *managementagentv1beta1.DataSource) bool {
	phase, _, pending := dataSourcePendingWrite(resource)
	return pending && phase == shared.OSOKAsyncPhaseDelete
}

func dataSourceCurrentWorkRequestID(resource *managementagentv1beta1.DataSource) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	return resource.Status.OsokStatus.Async.Current.WorkRequestID
}

func dataSourceRetryToken(resource *managementagentv1beta1.DataSource) *string {
	if resource == nil {
		return nil
	}
	uid := strings.TrimSpace(string(resource.GetUID()))
	if uid == "" {
		return nil
	}
	return common.String(uid)
}

func dataSourceSDKMetricDimensions(input []managementagentv1beta1.DataSourceMetricDimension) []managementagentsdk.MetricDimension {
	if len(input) == 0 {
		return nil
	}
	output := make([]managementagentsdk.MetricDimension, 0, len(input))
	for _, item := range input {
		name := strings.TrimSpace(item.Name)
		value := strings.TrimSpace(item.Value)
		if name == "" || value == "" {
			continue
		}
		output = append(output, managementagentsdk.MetricDimension{
			Name:  common.String(name),
			Value: common.String(value),
		})
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func dataSourceStatusMetricDimensions(input []managementagentsdk.MetricDimension) []managementagentv1beta1.DataSourceMetricDimension {
	if len(input) == 0 {
		return nil
	}
	output := make([]managementagentv1beta1.DataSourceMetricDimension, 0, len(input))
	for _, item := range input {
		output = append(output, managementagentv1beta1.DataSourceMetricDimension{
			Name:  stringValue(item.Name),
			Value: stringValue(item.Value),
		})
	}
	return output
}

func cloneDataSourceMetricDimensions(input []managementagentv1beta1.DataSourceMetricDimension) []managementagentv1beta1.DataSourceMetricDimension {
	if len(input) == 0 {
		return nil
	}
	return append([]managementagentv1beta1.DataSourceMetricDimension(nil), input...)
}

func dataSourceMetricDimensionEqual(a, b managementagentv1beta1.DataSourceMetricDimension) bool {
	return strings.TrimSpace(a.Name) == strings.TrimSpace(b.Name) &&
		strings.TrimSpace(a.Value) == strings.TrimSpace(b.Value)
}

func (c dataSourceSDKOCIClient) CreateDataSource(
	ctx context.Context,
	request managementagentsdk.CreateDataSourceRequest,
) (managementagentsdk.CreateDataSourceResponse, error) {
	if c.initErr != nil {
		return managementagentsdk.CreateDataSourceResponse{}, fmt.Errorf("initialize DataSource OCI client: %w", c.initErr)
	}
	return c.client.CreateDataSource(ctx, request)
}

func (c dataSourceSDKOCIClient) GetDataSource(
	ctx context.Context,
	request managementagentsdk.GetDataSourceRequest,
) (managementagentsdk.GetDataSourceResponse, error) {
	if c.initErr != nil {
		return managementagentsdk.GetDataSourceResponse{}, fmt.Errorf("initialize DataSource OCI client: %w", c.initErr)
	}
	return c.client.GetDataSource(ctx, request)
}

func (c dataSourceSDKOCIClient) ListDataSources(
	ctx context.Context,
	request managementagentsdk.ListDataSourcesRequest,
) (managementagentsdk.ListDataSourcesResponse, error) {
	if c.initErr != nil {
		return managementagentsdk.ListDataSourcesResponse{}, fmt.Errorf("initialize DataSource OCI client: %w", c.initErr)
	}
	return c.client.ListDataSources(ctx, request)
}

func (c dataSourceSDKOCIClient) UpdateDataSource(
	ctx context.Context,
	request managementagentsdk.UpdateDataSourceRequest,
) (managementagentsdk.UpdateDataSourceResponse, error) {
	if c.initErr != nil {
		return managementagentsdk.UpdateDataSourceResponse{}, fmt.Errorf("initialize DataSource OCI client: %w", c.initErr)
	}
	return c.client.UpdateDataSource(ctx, request)
}

func (c dataSourceSDKOCIClient) DeleteDataSource(
	ctx context.Context,
	request managementagentsdk.DeleteDataSourceRequest,
) (managementagentsdk.DeleteDataSourceResponse, error) {
	if c.initErr != nil {
		return managementagentsdk.DeleteDataSourceResponse{}, fmt.Errorf("initialize DataSource OCI client: %w", c.initErr)
	}
	return c.client.DeleteDataSource(ctx, request)
}

func (c dataSourceSDKOCIClient) GetWorkRequest(
	ctx context.Context,
	request managementagentsdk.GetWorkRequestRequest,
) (managementagentsdk.GetWorkRequestResponse, error) {
	if c.initErr != nil {
		return managementagentsdk.GetWorkRequestResponse{}, fmt.Errorf("initialize DataSource OCI client: %w", c.initErr)
	}
	return c.client.GetWorkRequest(ctx, request)
}

func stringPtrIfNonEmpty(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func intPtrIfPositive(value int) *int {
	if value <= 0 {
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
	if value == nil {
		return false
	}
	return *value
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}
