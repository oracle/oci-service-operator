/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package hostinsight

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	hostInsightKind = "HostInsight"

	hostInsightEntitySourceMacsManagedCloudHost    = "MACS_MANAGED_CLOUD_HOST"
	hostInsightEntitySourceMacsManagedExternalHost = "MACS_MANAGED_EXTERNAL_HOST"
	hostInsightEntitySourceEmManagedExternalHost   = "EM_MANAGED_EXTERNAL_HOST"
	hostInsightEntitySourceMacsManagedCloudDBHost  = "MACS_MANAGED_CLOUD_DB_HOST"
	hostInsightEntitySourcePeComanagedHost         = "PE_COMANAGED_HOST"

	hostInsightAmbiguousNotFoundCode = "HostInsightAmbiguousNotFound"
)

var hostInsightWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
		string(opsisdk.OperationTypeCreateHostInsight),
		string(opsisdk.OperationTypeEnableHostInsight),
	},
	UpdateActionTokens: []string{string(opsisdk.OperationTypeUpdateHostInsight)},
	DeleteActionTokens: []string{
		string(opsisdk.OperationTypeDeleteHostInsight),
		string(opsisdk.OperationTypeDisableHostInsight),
	},
}

type hostInsightOCIClient interface {
	CreateHostInsight(context.Context, opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error)
	GetHostInsight(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error)
	ListHostInsights(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error)
	UpdateHostInsight(context.Context, opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error)
	DeleteHostInsight(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type hostInsightIdentity struct {
	compartmentID                     string
	entitySource                      string
	computeID                         string
	managementAgentID                 string
	enterpriseManagerIdentifier       string
	enterpriseManagerBridgeID         string
	enterpriseManagerEntityIdentifier string
	exadataInsightID                  string
}

type hostInsightAmbiguousNotFoundError struct {
	HTTPStatusCode int
	ErrorCode      string
	OpcRequestID   string
	message        string
}

type hostInsightAmbiguousDeleteConfirmResponse struct {
	HostInsight opsisdk.HostInsight
	err         error
}

type hostInsightStatusAlias struct {
	opsisdk.HostInsight
}

type hostInsightSummaryStatusAlias struct {
	opsisdk.HostInsightSummary
}

type hostInsightRequestBodyContextKey struct{}

type hostInsightRequestBodies struct {
	create opsisdk.CreateHostInsightDetails
	update opsisdk.UpdateHostInsightDetails
}

func (e hostInsightAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e hostInsightAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.OpcRequestID
}

func init() {
	registerHostInsightRuntimeHooksMutator(func(manager *HostInsightServiceManager, hooks *HostInsightRuntimeHooks) {
		client, initErr := newHostInsightSDKClient(manager)
		applyHostInsightRuntimeHooks(hooks, client, initErr)
	})
}

func newHostInsightSDKClient(manager *HostInsightServiceManager) (hostInsightOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", hostInsightKind)
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyHostInsightRuntimeHooks(
	hooks *HostInsightRuntimeHooks,
	client hostInsightOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newHostInsightRuntimeSemantics()
	applyHostInsightBodyHooks(hooks)
	applyHostInsightOperationHooks(hooks, client, initErr)
	applyHostInsightIdentityHooks(hooks)
	applyHostInsightAsyncHooks(hooks, client, initErr)
	applyHostInsightClientWrappers(hooks, client, initErr)
	wrapHostInsightReadCalls(hooks)
	wrapHostInsightStatusProjection(hooks)
	hooks.DeleteHooks.ConfirmRead = hostInsightDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
}

func applyHostInsightBodyHooks(hooks *HostInsightRuntimeHooks) {
	hooks.BuildCreateBody = func(ctx context.Context, resource *opsiv1beta1.HostInsight, _ string) (any, error) {
		body, err := buildHostInsightCreateBody(resource)
		if err != nil {
			return nil, err
		}
		storeHostInsightCreateBody(ctx, body)
		return body, nil
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *opsiv1beta1.HostInsight,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		body, ok, err := buildHostInsightUpdateBody(resource, currentResponse)
		if err != nil {
			return nil, false, err
		}
		if ok {
			storeHostInsightUpdateBody(ctx, body)
		}
		return body, ok, nil
	}
}

func applyHostInsightOperationHooks(
	hooks *HostInsightRuntimeHooks,
	client hostInsightOCIClient,
	initErr error,
) {
	hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
		if initErr != nil {
			return opsisdk.CreateHostInsightResponse{}, fmt.Errorf("initialize %s OCI client: %w", hostInsightKind, initErr)
		}
		if client == nil {
			return opsisdk.CreateHostInsightResponse{}, fmt.Errorf("%s OCI client is not configured", hostInsightKind)
		}
		normalized, err := prepareHostInsightCreateRequest(ctx, request)
		if err != nil {
			return opsisdk.CreateHostInsightResponse{}, err
		}
		return client.CreateHostInsight(ctx, normalized)
	}
	hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
		if initErr != nil {
			return opsisdk.UpdateHostInsightResponse{}, fmt.Errorf("initialize %s OCI client: %w", hostInsightKind, initErr)
		}
		if client == nil {
			return opsisdk.UpdateHostInsightResponse{}, fmt.Errorf("%s OCI client is not configured", hostInsightKind)
		}
		normalized, err := prepareHostInsightUpdateRequest(ctx, request)
		if err != nil {
			return opsisdk.UpdateHostInsightResponse{}, err
		}
		return client.UpdateHostInsight(ctx, normalized)
	}
}

func applyHostInsightIdentityHooks(hooks *HostInsightRuntimeHooks) {
	hooks.Identity.Resolve = resolveHostInsightIdentity
	hooks.Identity.LookupExisting = func(ctx context.Context, resource *opsiv1beta1.HostInsight, identity any) (any, error) {
		return lookupExistingHostInsight(ctx, resource, identity, hooks.List.Call)
	}
	hooks.Create.Fields = hostInsightCreateFields()
	hooks.Get.Fields = hostInsightGetFields()
	hooks.List.Fields = hostInsightListFields()
	hooks.Update.Fields = hostInsightUpdateFields()
	hooks.Delete.Fields = hostInsightDeleteFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedHostInsightIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateHostInsightCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleHostInsightDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyHostInsightDeleteOutcome
}

func applyHostInsightAsyncHooks(
	hooks *HostInsightRuntimeHooks,
	client hostInsightOCIClient,
	initErr error,
) {
	hooks.Async.Adapter = hostInsightWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getHostInsightWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveHostInsightWorkRequestAction
	hooks.Async.ResolvePhase = resolveHostInsightWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverHostInsightIDFromWorkRequest
	hooks.Async.Message = hostInsightWorkRequestMessage
}

func applyHostInsightClientWrappers(
	hooks *HostInsightRuntimeHooks,
	client hostInsightOCIClient,
	initErr error,
) {
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate HostInsightServiceClient) HostInsightServiceClient {
		return hostInsightRequestBodyClient{delegate: delegate}
	})
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate HostInsightServiceClient) HostInsightServiceClient {
		return hostInsightDeleteWorkRequestClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	})
}

func newHostInsightServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client hostInsightOCIClient,
) HostInsightServiceClient {
	hooks := newHostInsightRuntimeHooksWithOCIClient(client)
	applyHostInsightRuntimeHooks(&hooks, client, nil)
	manager := &HostInsightServiceManager{Log: log}
	delegate := defaultHostInsightServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.HostInsight](
			buildHostInsightGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapHostInsightGeneratedClient(hooks, delegate)
}

func newHostInsightRuntimeHooksWithOCIClient(client hostInsightOCIClient) HostInsightRuntimeHooks {
	return HostInsightRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.HostInsight]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.HostInsight]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.HostInsight]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.HostInsight]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.HostInsight]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.HostInsight]{},
		Create: runtimeOperationHooks[opsisdk.CreateHostInsightRequest, opsisdk.CreateHostInsightResponse]{
			Fields: hostInsightCreateFields(),
			Call: func(ctx context.Context, request opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
				return client.CreateHostInsight(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetHostInsightRequest, opsisdk.GetHostInsightResponse]{
			Fields: hostInsightGetFields(),
			Call: func(ctx context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
				return client.GetHostInsight(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListHostInsightsRequest, opsisdk.ListHostInsightsResponse]{
			Fields: hostInsightListFields(),
			Call: func(ctx context.Context, request opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
				return client.ListHostInsights(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateHostInsightRequest, opsisdk.UpdateHostInsightResponse]{
			Fields: hostInsightUpdateFields(),
			Call: func(ctx context.Context, request opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
				return client.UpdateHostInsight(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteHostInsightRequest, opsisdk.DeleteHostInsightResponse]{
			Fields: hostInsightDeleteFields(),
			Call: func(ctx context.Context, request opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
				return client.DeleteHostInsight(ctx, request)
			},
		},
		WrapGeneratedClient: []func(HostInsightServiceClient) HostInsightServiceClient{},
	}
}

type hostInsightRequestBodyClient struct {
	delegate HostInsightServiceClient
}

var _ HostInsightServiceClient = hostInsightRequestBodyClient{}

func (c hostInsightRequestBodyClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	ctx = context.WithValue(ctx, hostInsightRequestBodyContextKey{}, &hostInsightRequestBodies{})
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c hostInsightRequestBodyClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

type hostInsightDeleteWorkRequestClient struct {
	delegate HostInsightServiceClient
	client   hostInsightOCIClient
	initErr  error
}

var _ HostInsightServiceClient = hostInsightDeleteWorkRequestClient{}

func (c hostInsightDeleteWorkRequestClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c hostInsightDeleteWorkRequestClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
) (bool, error) {
	if hostInsightSpecJSONData(resource) != "" {
		return c.deleteHostInsightWithUnsupportedJSON(ctx, resource)
	}
	handled, deleted, err := c.confirmSucceededDeleteWorkRequestNotFound(ctx, resource)
	if handled {
		return deleted, err
	}
	handled, deleted, err = c.confirmNoTrackedHostInsightDeleted(ctx, resource)
	if handled {
		return deleted, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c hostInsightDeleteWorkRequestClient) deleteHostInsightWithUnsupportedJSON(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
) (bool, error) {
	currentID := trackedHostInsightID(resource)
	if currentID == "" {
		markHostInsightDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	if c.initErr != nil {
		return false, fmt.Errorf("initialize %s OCI client: %w", hostInsightKind, c.initErr)
	}
	if c.client == nil {
		return false, fmt.Errorf("%s OCI client is not configured", hostInsightKind)
	}
	if workRequestID := currentHostInsightDeleteWorkRequestID(resource); workRequestID != "" {
		return c.resumeTrackedHostInsightDeleteWorkRequest(ctx, resource, workRequestID)
	}
	if handled, deleted, err := c.confirmTrackedHostInsightBeforeDelete(ctx, resource, currentID); handled {
		return deleted, err
	}
	response, err := c.client.DeleteHostInsight(ctx, opsisdk.DeleteHostInsightRequest{
		HostInsightId: common.String(currentID),
	})
	if err != nil {
		return handleTrackedHostInsightDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := hostInsightString(response.OpcWorkRequestId); workRequestID != "" {
		markHostInsightDeleteWorkRequestPending(resource, currentID, workRequestID)
		return false, nil
	}
	return c.confirmTrackedHostInsightAfterDelete(ctx, resource, currentID)
}

func (c hostInsightDeleteWorkRequestClient) resumeTrackedHostInsightDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	workRequestID string,
) (bool, error) {
	response, err := c.client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	current, err := buildHostInsightDeleteWorkRequestStatus(resource, response.WorkRequest)
	if err != nil {
		return false, err
	}
	if current.NormalizedClass == shared.OSOKAsyncClassSucceeded {
		_, deleted, err := c.confirmHostInsightDeletedAfterSucceededWorkRequest(ctx, resource, response.WorkRequest)
		return deleted, err
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
	if current.NormalizedClass == shared.OSOKAsyncClassPending {
		return false, nil
	}
	return false, fmt.Errorf("%s delete work request %s is %s", hostInsightKind, current.WorkRequestID, current.RawStatus)
}

func (c hostInsightDeleteWorkRequestClient) confirmTrackedHostInsightBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	currentID string,
) (bool, bool, error) {
	response, err := c.client.GetHostInsight(ctx, opsisdk.GetHostInsightRequest{
		HostInsightId: common.String(currentID),
	})
	if err != nil {
		return handleTrackedHostInsightConfirmError(resource, err)
	}
	return handleTrackedHostInsightConfirmResponse(resource, response)
}

func (c hostInsightDeleteWorkRequestClient) confirmTrackedHostInsightAfterDelete(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	currentID string,
) (bool, error) {
	response, err := c.client.GetHostInsight(ctx, opsisdk.GetHostInsightRequest{
		HostInsightId: common.String(currentID),
	})
	if err != nil {
		handled, deleted, handledErr := handleTrackedHostInsightConfirmError(resource, err)
		if handled {
			return deleted, handledErr
		}
		return false, err
	}
	if handled, deleted, err := handleTrackedHostInsightConfirmResponse(resource, response); handled {
		return deleted, err
	}
	markHostInsightDeletePending(resource, currentID, "OCI delete request accepted")
	return false, nil
}

func handleTrackedHostInsightConfirmResponse(
	resource *opsiv1beta1.HostInsight,
	response opsisdk.GetHostInsightResponse,
) (bool, bool, error) {
	response.HostInsight = hostInsightWithSDKStatusAlias(response.HostInsight)
	switch {
	case hostInsightResponseDeleted(response):
		markHostInsightDeleted(resource, "OCI resource deleted")
		return true, true, nil
	case hostInsightResponseDeletePending(response):
		currentID := trackedHostInsightID(resource)
		markHostInsightDeletePending(resource, currentID, "OCI resource delete is pending")
		return true, false, nil
	default:
		return false, false, nil
	}
}

func handleTrackedHostInsightConfirmError(
	resource *opsiv1beta1.HostInsight,
	err error,
) (bool, bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markHostInsightDeleted(resource, "OCI resource deleted")
		return true, true, nil
	case classification.IsAuthShapedNotFound(), isHostInsightAmbiguousNotFound(err):
		return true, false, handleHostInsightDeleteError(resource, err)
	default:
		return false, false, nil
	}
}

func handleTrackedHostInsightDeleteError(resource *opsiv1beta1.HostInsight, err error) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsUnambiguousNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markHostInsightDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	return false, handleHostInsightDeleteError(resource, err)
}

func (c hostInsightDeleteWorkRequestClient) confirmSucceededDeleteWorkRequestNotFound(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
) (bool, bool, error) {
	workRequest, ok := c.succeededHostInsightDeleteWorkRequest(ctx, resource)
	if !ok {
		return false, false, nil
	}
	return c.confirmHostInsightDeletedAfterSucceededWorkRequest(ctx, resource, workRequest)
}

func (c hostInsightDeleteWorkRequestClient) confirmNoTrackedHostInsightDeleted(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
) (bool, bool, error) {
	if trackedHostInsightID(resource) != "" || c.initErr != nil || c.client == nil {
		return false, false, nil
	}
	identityAny, err := resolveHostInsightIdentity(resource)
	if err != nil {
		return false, false, nil
	}
	identity := identityAny.(hostInsightIdentity)
	response, err := listHostInsightsAllPages(ctx, hostInsightListRequestForIdentity(identity), c.client.ListHostInsights)
	if err != nil {
		return true, false, err
	}
	matches := hostInsightMatchingSummaries(identity, response.Items)
	switch len(matches) {
	case 0:
		if opcRequestID := hostInsightString(response.OpcRequestId); opcRequestID != "" {
			resource.Status.OsokStatus.OpcRequestID = opcRequestID
		}
		markHostInsightDeleted(resource, "OCI resource no longer exists")
		return true, true, nil
	case 1:
		return false, false, nil
	default:
		return true, false, fmt.Errorf("%s delete confirmation found %d matching host insights", hostInsightKind, len(matches))
	}
}

func (c hostInsightDeleteWorkRequestClient) succeededHostInsightDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
) (opsisdk.WorkRequest, bool) {
	workRequestID := currentHostInsightDeleteWorkRequestID(resource)
	if workRequestID == "" || c.initErr != nil || c.client == nil {
		return opsisdk.WorkRequest{}, false
	}
	workRequestResponse, err := c.client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return opsisdk.WorkRequest{}, false
	}
	workRequest := workRequestResponse.WorkRequest
	if !isSucceededHostInsightDeleteWorkRequest(workRequest) {
		return opsisdk.WorkRequest{}, false
	}
	return workRequest, true
}

func (c hostInsightDeleteWorkRequestClient) confirmHostInsightDeletedAfterSucceededWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	workRequest opsisdk.WorkRequest,
) (bool, bool, error) {
	currentID := currentHostInsightIDForDelete(resource, workRequest)
	if currentID == "" {
		markHostInsightDeleted(resource, "OCI HostInsight delete work request completed")
		return true, true, nil
	}
	response, err := c.client.GetHostInsight(ctx, opsisdk.GetHostInsightRequest{
		HostInsightId: common.String(currentID),
	})
	if err == nil {
		return handleSucceededHostInsightDeleteReadback(resource, response)
	}
	return handleSucceededHostInsightDeleteReadError(resource, err)
}

func handleSucceededHostInsightDeleteReadback(
	resource *opsiv1beta1.HostInsight,
	response opsisdk.GetHostInsightResponse,
) (bool, bool, error) {
	if !hostInsightResponseDeleted(response) {
		return false, false, nil
	}
	markHostInsightDeleted(resource, "OCI resource deleted")
	return true, true, nil
}

func handleSucceededHostInsightDeleteReadError(
	resource *opsiv1beta1.HostInsight,
	err error,
) (bool, bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markHostInsightDeleted(resource, "OCI resource deleted")
		return true, true, nil
	case classification.IsAuthShapedNotFound():
		return true, false, handleHostInsightDeleteError(resource, err)
	default:
		return false, false, nil
	}
}

func currentHostInsightDeleteWorkRequestID(resource *opsiv1beta1.HostInsight) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func isSucceededHostInsightDeleteWorkRequest(workRequest opsisdk.WorkRequest) bool {
	phase, ok, err := resolveHostInsightWorkRequestPhase(workRequest)
	if err != nil || !ok || phase != shared.OSOKAsyncPhaseDelete {
		return false
	}
	class, err := hostInsightWorkRequestAsyncAdapter.Normalize(string(workRequest.Status))
	return err == nil && class == shared.OSOKAsyncClassSucceeded
}

func currentHostInsightIDForDelete(resource *opsiv1beta1.HostInsight, workRequest opsisdk.WorkRequest) string {
	if resource == nil {
		return ""
	}
	if currentID := trackedHostInsightID(resource); currentID != "" {
		return currentID
	}
	currentID, err := recoverHostInsightIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(currentID)
}

func trackedHostInsightID(resource *opsiv1beta1.HostInsight) string {
	if resource == nil {
		return ""
	}
	if currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); currentID != "" {
		return currentID
	}
	return strings.TrimSpace(resource.Status.Id)
}

func markHostInsightDeleted(resource *opsiv1beta1.HostInsight, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		status.Message,
		loggerutil.OSOKLogger{},
	)
}

func markHostInsightDeletePending(resource *opsiv1beta1.HostInsight, currentID string, message string) {
	if resource == nil {
		return
	}
	currentID = strings.TrimSpace(currentID)
	if currentID != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       string(opsisdk.LifecycleStateDeleting),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	}, loggerutil.OSOKLogger{})
}

func markHostInsightDeleteWorkRequestPending(resource *opsiv1beta1.HostInsight, currentID string, workRequestID string) {
	if resource == nil {
		return
	}
	currentID = strings.TrimSpace(currentID)
	if currentID != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
	}
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}, loggerutil.OSOKLogger{})
}

func newHostInsightRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "opsi",
		FormalSlug:    "hostinsight",
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
			ProvisioningStates: []string{string(opsisdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.LifecycleStateUpdating)},
			ActiveStates: []string{
				string(opsisdk.LifecycleStateActive),
				string(opsisdk.ResourceStatusEnabled),
				string(opsisdk.ResourceStatusDisabled),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(opsisdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{
				string(opsisdk.LifecycleStateDeleted),
				string(opsisdk.ResourceStatusTerminated),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"entitySource",
				"computeId",
				"managementAgentId",
				"enterpriseManagerIdentifier",
				"enterpriseManagerBridgeId",
				"enterpriseManagerEntityIdentifier",
				"exadataInsightId",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
			},
			ForceNew: []string{
				"compartmentId",
				"entitySource",
				"computeId",
				"managementAgentId",
				"enterpriseManagerIdentifier",
				"enterpriseManagerBridgeId",
				"enterpriseManagerEntityIdentifier",
				"exadataInsightId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: hostInsightKind, Action: "CreateHostInsight"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: hostInsightKind, Action: "UpdateHostInsight"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: hostInsightKind, Action: "DeleteHostInsight"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetHostInsight",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: hostInsightKind, Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetHostInsight",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: hostInsightKind, Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: hostInsightKind, Action: "GetWorkRequest"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func hostInsightCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpcRetryToken", RequestName: "opcRetryToken", Contribution: "header"},
	}
}

func hostInsightGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "HostInsightId", RequestName: "hostInsightId", Contribution: "path", PreferResourceID: true},
	}
}

func hostInsightListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "EnterpriseManagerBridgeId", RequestName: "enterpriseManagerBridgeId", Contribution: "query", LookupPaths: []string{"status.enterpriseManagerBridgeId", "spec.enterpriseManagerBridgeId", "enterpriseManagerBridgeId"}},
		{FieldName: "ExadataInsightId", RequestName: "exadataInsightId", Contribution: "query", LookupPaths: []string{"status.exadataInsightId", "spec.exadataInsightId", "exadataInsightId"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func hostInsightUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "HostInsightId", RequestName: "hostInsightId", Contribution: "path", PreferResourceID: true},
	}
}

func hostInsightDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "HostInsightId", RequestName: "hostInsightId", Contribution: "path", PreferResourceID: true},
	}
}

func buildHostInsightCreateBody(resource *opsiv1beta1.HostInsight) (opsisdk.CreateHostInsightDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", hostInsightKind)
	}
	payload, err := hostInsightCreatePayload(resource.Spec)
	if err != nil {
		return nil, err
	}
	entitySource := hostInsightStringFromPayload(payload, "entitySource")
	switch entitySource {
	case hostInsightEntitySourceMacsManagedCloudHost:
		if err := requireHostInsightPayloadStrings("spec", payload, "compartmentId", "entitySource", "computeId"); err != nil {
			return nil, err
		}
		var details opsisdk.CreateMacsManagedCloudHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceMacsManagedExternalHost:
		if err := requireHostInsightPayloadStrings("spec", payload, "compartmentId", "entitySource", "managementAgentId"); err != nil {
			return nil, err
		}
		var details opsisdk.CreateMacsManagedExternalHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceEmManagedExternalHost:
		if err := requireHostInsightPayloadStrings(
			"spec",
			payload,
			"compartmentId",
			"entitySource",
			"enterpriseManagerIdentifier",
			"enterpriseManagerBridgeId",
			"enterpriseManagerEntityIdentifier",
		); err != nil {
			return nil, err
		}
		var details opsisdk.CreateEmManagedExternalHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceMacsManagedCloudDBHost, hostInsightEntitySourcePeComanagedHost:
		return nil, fmt.Errorf("%s spec.entitySource %q is read-only for OSOK create; bind an existing HostInsight by status.ocid instead", hostInsightKind, entitySource)
	default:
		return nil, fmt.Errorf("%s spec.entitySource %q is not supported", hostInsightKind, entitySource)
	}
}

func normalizeHostInsightCreateRequest(
	request opsisdk.CreateHostInsightRequest,
) (opsisdk.CreateHostInsightRequest, error) {
	details, err := normalizeHostInsightCreateDetails(request.CreateHostInsightDetails)
	if err != nil {
		return request, fmt.Errorf("normalize %s create body: %w", hostInsightKind, err)
	}
	request.CreateHostInsightDetails = details
	return request, nil
}

func prepareHostInsightCreateRequest(
	ctx context.Context,
	request opsisdk.CreateHostInsightRequest,
) (opsisdk.CreateHostInsightRequest, error) {
	if body, ok := hostInsightCreateBodyFromContext(ctx); ok {
		request.CreateHostInsightDetails = body
		return request, nil
	}
	return normalizeHostInsightCreateRequest(request)
}

func normalizeHostInsightCreateDetails(raw any) (opsisdk.CreateHostInsightDetails, error) {
	if raw == nil {
		return nil, fmt.Errorf("%s create body is nil", hostInsightKind)
	}
	if details, ok := raw.(opsisdk.CreateHostInsightDetails); ok {
		return details, nil
	}
	payload, err := hostInsightPolymorphicPayload(raw, "create")
	if err != nil {
		return nil, err
	}
	switch entitySource := normalizeHostInsightEntitySource(hostInsightStringFromMap(payload, "entitySource")); entitySource {
	case hostInsightEntitySourceMacsManagedCloudHost:
		var details opsisdk.CreateMacsManagedCloudHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceMacsManagedExternalHost:
		var details opsisdk.CreateMacsManagedExternalHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceEmManagedExternalHost:
		var details opsisdk.CreateEmManagedExternalHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	default:
		return nil, fmt.Errorf("%s create entitySource %q is not supported", hostInsightKind, entitySource)
	}
}

func buildHostInsightUpdateBody(
	resource *opsiv1beta1.HostInsight,
	currentResponse any,
) (opsisdk.UpdateHostInsightDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", hostInsightKind)
	}
	if err := validateHostInsightCreateOnlyDrift(resource, currentResponse); err != nil {
		return nil, false, err
	}
	current, ok := hostInsightBodyMap(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a body", hostInsightKind)
	}

	updateNeeded := false
	freeformTags, freeformChanged := desiredHostInsightFreeformTagsUpdate(resource.Spec.FreeformTags, current["freeformTags"])
	if freeformChanged {
		updateNeeded = true
	}
	definedTags, definedChanged := desiredHostInsightDefinedTagsUpdate(resource.Spec.DefinedTags, current["definedTags"])
	if definedChanged {
		updateNeeded = true
	}
	if !updateNeeded {
		return nil, false, nil
	}

	entitySource := normalizeHostInsightEntitySource(hostInsightStringFromMap(current, "entitySource"))
	if entitySource == "" {
		entitySource = normalizeHostInsightEntitySource(resource.Spec.EntitySource)
	}
	if entitySource == "" {
		return nil, false, fmt.Errorf("%s update cannot resolve current entitySource", hostInsightKind)
	}
	return hostInsightUpdateDetailsForSource(entitySource, freeformTags, definedTags)
}

func normalizeHostInsightUpdateRequest(
	request opsisdk.UpdateHostInsightRequest,
) (opsisdk.UpdateHostInsightRequest, error) {
	details, err := normalizeHostInsightUpdateDetails(request.UpdateHostInsightDetails)
	if err != nil {
		return request, fmt.Errorf("normalize %s update body: %w", hostInsightKind, err)
	}
	request.UpdateHostInsightDetails = details
	return request, nil
}

func prepareHostInsightUpdateRequest(
	ctx context.Context,
	request opsisdk.UpdateHostInsightRequest,
) (opsisdk.UpdateHostInsightRequest, error) {
	if body, ok := hostInsightUpdateBodyFromContext(ctx); ok {
		request.UpdateHostInsightDetails = body
		return request, nil
	}
	return normalizeHostInsightUpdateRequest(request)
}

func normalizeHostInsightUpdateDetails(raw any) (opsisdk.UpdateHostInsightDetails, error) {
	if raw == nil {
		return nil, fmt.Errorf("%s update body is nil", hostInsightKind)
	}
	if details, ok := raw.(opsisdk.UpdateHostInsightDetails); ok {
		return details, nil
	}
	payload, err := hostInsightPolymorphicPayload(raw, "update")
	if err != nil {
		return nil, err
	}
	switch entitySource := normalizeHostInsightEntitySource(hostInsightStringFromMap(payload, "entitySource")); entitySource {
	case hostInsightEntitySourceMacsManagedCloudHost:
		var details opsisdk.UpdateMacsManagedCloudHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceMacsManagedExternalHost:
		var details opsisdk.UpdateMacsManagedExternalHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceEmManagedExternalHost:
		var details opsisdk.UpdateEmManagedExternalHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourceMacsManagedCloudDBHost:
		var details opsisdk.UpdateMacsManagedCloudDatabaseHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	case hostInsightEntitySourcePeComanagedHost:
		var details opsisdk.UpdatePeComanagedHostInsightDetails
		return details, unmarshalHostInsightPayload(payload, &details)
	default:
		return nil, fmt.Errorf("%s update entitySource %q is not supported", hostInsightKind, entitySource)
	}
}

func hostInsightPolymorphicPayload(raw any, operation string) (map[string]any, error) {
	payload, ok := hostInsightBodyMap(raw)
	if !ok {
		return nil, fmt.Errorf("%s %s body has unexpected type %T", hostInsightKind, strings.TrimSpace(operation), raw)
	}
	if normalizeHostInsightEntitySource(hostInsightStringFromMap(payload, "entitySource")) == "" {
		return nil, fmt.Errorf("%s %s body is missing entitySource discriminator", hostInsightKind, strings.TrimSpace(operation))
	}
	return payload, nil
}

func storeHostInsightCreateBody(ctx context.Context, body opsisdk.CreateHostInsightDetails) {
	if bodies, ok := hostInsightRequestBodiesFromContext(ctx); ok {
		bodies.create = body
	}
}

func storeHostInsightUpdateBody(ctx context.Context, body opsisdk.UpdateHostInsightDetails) {
	if bodies, ok := hostInsightRequestBodiesFromContext(ctx); ok {
		bodies.update = body
	}
}

func hostInsightCreateBodyFromContext(ctx context.Context) (opsisdk.CreateHostInsightDetails, bool) {
	bodies, ok := hostInsightRequestBodiesFromContext(ctx)
	return bodies.create, ok && bodies.create != nil
}

func hostInsightUpdateBodyFromContext(ctx context.Context) (opsisdk.UpdateHostInsightDetails, bool) {
	bodies, ok := hostInsightRequestBodiesFromContext(ctx)
	return bodies.update, ok && bodies.update != nil
}

func hostInsightRequestBodiesFromContext(ctx context.Context) (*hostInsightRequestBodies, bool) {
	if ctx == nil {
		return nil, false
	}
	bodies, ok := ctx.Value(hostInsightRequestBodyContextKey{}).(*hostInsightRequestBodies)
	return bodies, ok && bodies != nil
}

func hostInsightUpdateDetailsForSource(
	entitySource string,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
) (opsisdk.UpdateHostInsightDetails, bool, error) {
	switch entitySource {
	case hostInsightEntitySourceMacsManagedCloudHost:
		return opsisdk.UpdateMacsManagedCloudHostInsightDetails{
			FreeformTags: freeformTags,
			DefinedTags:  definedTags,
		}, true, nil
	case hostInsightEntitySourceMacsManagedExternalHost:
		return opsisdk.UpdateMacsManagedExternalHostInsightDetails{
			FreeformTags: freeformTags,
			DefinedTags:  definedTags,
		}, true, nil
	case hostInsightEntitySourceEmManagedExternalHost:
		return opsisdk.UpdateEmManagedExternalHostInsightDetails{
			FreeformTags: freeformTags,
			DefinedTags:  definedTags,
		}, true, nil
	case hostInsightEntitySourceMacsManagedCloudDBHost:
		return opsisdk.UpdateMacsManagedCloudDatabaseHostInsightDetails{
			FreeformTags: freeformTags,
			DefinedTags:  definedTags,
		}, true, nil
	case hostInsightEntitySourcePeComanagedHost:
		return opsisdk.UpdatePeComanagedHostInsightDetails{
			FreeformTags: freeformTags,
			DefinedTags:  definedTags,
		}, true, nil
	default:
		return nil, false, fmt.Errorf("%s update entitySource %q is not supported", hostInsightKind, entitySource)
	}
}

func resolveHostInsightIdentity(resource *opsiv1beta1.HostInsight) (any, error) {
	identity, err := hostInsightIdentityFromResource(resource)
	if err != nil {
		return nil, err
	}
	if err := validateHostInsightIdentity(identity); err != nil {
		return nil, err
	}
	return identity, nil
}

func hostInsightIdentityFromResource(resource *opsiv1beta1.HostInsight) (hostInsightIdentity, error) {
	if resource == nil {
		return hostInsightIdentity{}, fmt.Errorf("%s resource is nil", hostInsightKind)
	}
	payload, err := hostInsightCreatePayload(resource.Spec)
	if err != nil {
		return hostInsightIdentity{}, err
	}
	return hostInsightIdentity{
		compartmentID:                     hostInsightStringFromPayload(payload, "compartmentId"),
		entitySource:                      hostInsightStringFromPayload(payload, "entitySource"),
		computeID:                         hostInsightStringFromPayload(payload, "computeId"),
		managementAgentID:                 hostInsightStringFromPayload(payload, "managementAgentId"),
		enterpriseManagerIdentifier:       hostInsightStringFromPayload(payload, "enterpriseManagerIdentifier"),
		enterpriseManagerBridgeID:         hostInsightStringFromPayload(payload, "enterpriseManagerBridgeId"),
		enterpriseManagerEntityIdentifier: hostInsightStringFromPayload(payload, "enterpriseManagerEntityIdentifier"),
		exadataInsightID:                  hostInsightStringFromPayload(payload, "exadataInsightId"),
	}, nil
}

func validateHostInsightIdentity(identity hostInsightIdentity) error {
	if identity.compartmentID == "" {
		return fmt.Errorf("%s spec.compartmentId is required", hostInsightKind)
	}
	if identity.entitySource == "" {
		return fmt.Errorf("%s spec.entitySource could not be inferred from HostInsight identity fields", hostInsightKind)
	}
	switch identity.entitySource {
	case hostInsightEntitySourceMacsManagedCloudHost:
		return requireHostInsightIdentityField(identity.computeID, "computeId", identity.entitySource)
	case hostInsightEntitySourceMacsManagedExternalHost:
		return requireHostInsightIdentityField(identity.managementAgentID, "managementAgentId", identity.entitySource)
	case hostInsightEntitySourceEmManagedExternalHost:
		return requireEmHostInsightIdentity(identity)
	case hostInsightEntitySourceMacsManagedCloudDBHost, hostInsightEntitySourcePeComanagedHost:
		return nil
	default:
		return fmt.Errorf("%s spec.entitySource %q is not supported", hostInsightKind, identity.entitySource)
	}
}

func requireHostInsightIdentityField(value string, field string, entitySource string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}
	return fmt.Errorf("%s spec.%s is required for entitySource %s", hostInsightKind, field, entitySource)
}

func requireEmHostInsightIdentity(identity hostInsightIdentity) error {
	missing := missingHostInsightIdentityFields([]struct {
		name  string
		value string
	}{
		{name: "enterpriseManagerIdentifier", value: identity.enterpriseManagerIdentifier},
		{name: "enterpriseManagerBridgeId", value: identity.enterpriseManagerBridgeID},
		{name: "enterpriseManagerEntityIdentifier", value: identity.enterpriseManagerEntityIdentifier},
	})
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"%s spec is missing required field(s) for entitySource %s: %s",
		hostInsightKind,
		identity.entitySource,
		strings.Join(missing, ", "),
	)
}

func missingHostInsightIdentityFields(fields []struct {
	name  string
	value string
}) []string {
	var missing []string
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	return missing
}

func lookupExistingHostInsight(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	identity any,
	listHostInsights func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error),
) (any, error) {
	if resource == nil || listHostInsights == nil || identity == nil {
		return nil, nil
	}
	resolved, ok := identity.(hostInsightIdentity)
	if !ok {
		return nil, fmt.Errorf("%s identity has unexpected type %T", hostInsightKind, identity)
	}
	response, err := listHostInsights(ctx, hostInsightListRequestForIdentity(resolved))
	if err != nil {
		return nil, err
	}
	matches := hostInsightMatchingSummaries(resolved, response.Items)
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%s list lookup found %d matching host insights", hostInsightKind, len(matches))
	}
}

func hostInsightListRequestForIdentity(identity hostInsightIdentity) opsisdk.ListHostInsightsRequest {
	request := opsisdk.ListHostInsightsRequest{
		CompartmentId: common.String(identity.compartmentID),
	}
	if identity.enterpriseManagerBridgeID != "" {
		request.EnterpriseManagerBridgeId = common.String(identity.enterpriseManagerBridgeID)
	}
	if identity.exadataInsightID != "" {
		request.ExadataInsightId = common.String(identity.exadataInsightID)
	}
	return request
}

func hostInsightMatchingSummaries(identity hostInsightIdentity, items []opsisdk.HostInsightSummary) []opsisdk.HostInsightSummary {
	matches := make([]opsisdk.HostInsightSummary, 0, len(items))
	for _, item := range items {
		if hostInsightSummaryMatchesIdentity(item, identity) {
			matches = append(matches, item)
		}
	}
	return matches
}

func hostInsightSummaryMatchesIdentity(item opsisdk.HostInsightSummary, identity hostInsightIdentity) bool {
	values, ok := hostInsightBodyMap(item)
	if !ok || !hostInsightSummaryBaseMatchesIdentity(values, identity) {
		return false
	}
	switch identity.entitySource {
	case hostInsightEntitySourceMacsManagedCloudHost:
		return hostInsightStringFromMap(values, "computeId") == identity.computeID
	case hostInsightEntitySourceMacsManagedExternalHost:
		return hostInsightStringFromMap(values, "managementAgentId") == identity.managementAgentID
	case hostInsightEntitySourceEmManagedExternalHost:
		return emHostInsightSummaryMatchesIdentity(values, identity)
	default:
		return false
	}
}

func hostInsightSummaryBaseMatchesIdentity(values map[string]any, identity hostInsightIdentity) bool {
	if hostInsightItemDeleted(values) {
		return false
	}
	if hostInsightStringFromMap(values, "compartmentId") != identity.compartmentID {
		return false
	}
	source := normalizeHostInsightEntitySource(hostInsightStringFromMap(values, "entitySource"))
	return source != "" && source == identity.entitySource
}

func emHostInsightSummaryMatchesIdentity(values map[string]any, identity hostInsightIdentity) bool {
	if hostInsightStringFromMap(values, "enterpriseManagerIdentifier") != identity.enterpriseManagerIdentifier {
		return false
	}
	if hostInsightStringFromMap(values, "enterpriseManagerBridgeId") != identity.enterpriseManagerBridgeID {
		return false
	}
	if hostInsightStringFromMap(values, "enterpriseManagerEntityIdentifier") != identity.enterpriseManagerEntityIdentifier {
		return false
	}
	return identity.exadataInsightID == "" || hostInsightStringFromMap(values, "exadataInsightId") == identity.exadataInsightID
}

func validateHostInsightCreateOnlyDrift(resource *opsiv1beta1.HostInsight, currentResponse any) error {
	if resource == nil || currentResponse == nil {
		return nil
	}
	identityAny, err := resolveHostInsightIdentity(resource)
	if err != nil {
		return err
	}
	identity := identityAny.(hostInsightIdentity)
	current, ok := hostInsightBodyMap(currentResponse)
	if !ok {
		return nil
	}

	checks := []struct {
		field string
		want  string
	}{
		{field: "compartmentId", want: identity.compartmentID},
		{field: "entitySource", want: identity.entitySource},
		{field: "computeId", want: identity.computeID},
		{field: "managementAgentId", want: identity.managementAgentID},
		{field: "enterpriseManagerIdentifier", want: identity.enterpriseManagerIdentifier},
		{field: "enterpriseManagerBridgeId", want: identity.enterpriseManagerBridgeID},
		{field: "enterpriseManagerEntityIdentifier", want: identity.enterpriseManagerEntityIdentifier},
		{field: "exadataInsightId", want: identity.exadataInsightID},
	}
	for _, check := range checks {
		if check.want == "" {
			continue
		}
		got := hostInsightStringFromMap(current, check.field)
		if got == "" {
			continue
		}
		if normalizeHostInsightComparable(check.field, got) != normalizeHostInsightComparable(check.field, check.want) {
			return fmt.Errorf("%s create-only field %s changed from %q to %q", hostInsightKind, check.field, got, check.want)
		}
	}
	return nil
}

func clearTrackedHostInsightIdentity(resource *opsiv1beta1.HostInsight) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
}

func wrapHostInsightReadCalls(hooks *HostInsightRuntimeHooks) {
	if hooks.Get.Call != nil {
		getHostInsight := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			response, err := getHostInsight(ctx, request)
			return response, conservativeHostInsightNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		listHostInsights := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			return listHostInsightsAllPages(ctx, request, listHostInsights)
		}
	}
}

func wrapHostInsightStatusProjection(hooks *HostInsightRuntimeHooks) {
	if hooks.Create.Call != nil {
		createHostInsight := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
			response, err := createHostInsight(ctx, request)
			response.HostInsight = hostInsightWithSDKStatusAlias(response.HostInsight)
			return response, err
		}
	}
	if hooks.Get.Call != nil {
		getHostInsight := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			response, err := getHostInsight(ctx, request)
			response.HostInsight = hostInsightWithSDKStatusAlias(response.HostInsight)
			return response, err
		}
	}
	if hooks.List.Call != nil {
		listHostInsights := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			response, err := listHostInsights(ctx, request)
			for index, item := range response.Items {
				response.Items[index] = hostInsightSummaryWithSDKStatusAlias(item)
			}
			return response, err
		}
	}
}

func listHostInsightsAllPages(
	ctx context.Context,
	request opsisdk.ListHostInsightsRequest,
	listHostInsights func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error),
) (opsisdk.ListHostInsightsResponse, error) {
	var combined opsisdk.ListHostInsightsResponse
	for {
		response, err := listHostInsights(ctx, request)
		if err != nil {
			return opsisdk.ListHostInsightsResponse{}, conservativeHostInsightNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		for _, item := range response.Items {
			values, ok := hostInsightBodyMap(item)
			if ok && hostInsightItemDeleted(values) {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleHostInsightDeleteError(resource *opsiv1beta1.HostInsight, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if isHostInsightAmbiguousNotFound(err) {
		return err
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return hostInsightAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      hostInsightAmbiguousNotFoundCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"%s delete returned ambiguous 404 NotAuthorizedOrNotFound; retaining finalizer until OCI deletion is confirmed",
			hostInsightKind,
		),
	}
}

func hostInsightDeleteConfirmRead(
	getHostInsight func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error),
	listHostInsights func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error),
) func(context.Context, *opsiv1beta1.HostInsight, string) (any, error) {
	return func(ctx context.Context, resource *opsiv1beta1.HostInsight, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID == "" {
			return hostInsightDeleteConfirmReadByList(ctx, resource, listHostInsights)
		}
		if getHostInsight == nil {
			return nil, fmt.Errorf("%s delete confirmation requires a readable OCI operation", hostInsightKind)
		}
		response, err := getHostInsight(ctx, opsisdk.GetHostInsightRequest{
			HostInsightId: common.String(currentID),
		})
		if err == nil {
			return response, nil
		}
		if !isHostInsightAmbiguousNotFound(err) && !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return nil, err
		}
		handledErr := handleHostInsightDeleteError(resource, err)
		if handledErr == nil {
			handledErr = err
		}
		return hostInsightAmbiguousDeleteConfirmResponse{
			HostInsight: hostInsightWithSDKStatusAlias(hostInsightDeletePlaceholder(resource, currentID)),
			err:         handledErr,
		}, nil
	}
}

func hostInsightDeleteConfirmReadByList(
	ctx context.Context,
	resource *opsiv1beta1.HostInsight,
	listHostInsights func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error),
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", hostInsightKind)
	}
	if listHostInsights == nil {
		return nil, fmt.Errorf("%s delete confirmation requires a tracked OCI identifier", hostInsightKind)
	}
	identity, err := resolveHostInsightIdentity(resource)
	if err != nil {
		return nil, err
	}
	resolved := identity.(hostInsightIdentity)
	response, err := listHostInsights(ctx, opsisdk.ListHostInsightsRequest{
		CompartmentId: common.String(resolved.compartmentID),
	})
	if err != nil {
		return nil, err
	}
	matches := hostInsightMatchingSummaries(resolved, response.Items)
	switch len(matches) {
	case 0:
		return nil, errorutil.NotFoundOciError(errorutil.OciErrors{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			OpcRequestID:   hostInsightString(response.OpcRequestId),
			Description:    "host insight not found during delete confirmation",
		})
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%s delete confirmation found %d matching host insights", hostInsightKind, len(matches))
	}
}

func applyHostInsightDeleteOutcome(
	resource *opsiv1beta1.HostInsight,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if err, ok := hostInsightAmbiguousDeleteConfirmError(response); ok && err != nil {
		return generatedruntime.DeleteOutcome{Handled: true}, err
	}
	if hostInsightResponseDeleted(response) {
		markHostInsightDeleted(resource, "OCI resource deleted")
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func hostInsightAmbiguousDeleteConfirmError(response any) (error, bool) {
	switch typed := response.(type) {
	case hostInsightAmbiguousDeleteConfirmResponse:
		return typed.err, true
	case *hostInsightAmbiguousDeleteConfirmResponse:
		if typed == nil {
			return nil, false
		}
		return typed.err, true
	default:
		return nil, false
	}
}

func hostInsightResponseDeleted(response any) bool {
	values, ok := hostInsightBodyMap(response)
	return ok && hostInsightItemDeleted(values)
}

func hostInsightResponseDeletePending(response any) bool {
	values, ok := hostInsightBodyMap(response)
	return ok && strings.ToUpper(hostInsightStringFromMap(values, "lifecycleState")) == string(opsisdk.LifecycleStateDeleting)
}

func hostInsightDeletePlaceholder(resource *opsiv1beta1.HostInsight, currentID string) opsisdk.HostInsight {
	compartmentID := ""
	computeID := ""
	managementAgentID := ""
	if resource != nil {
		compartmentID = strings.TrimSpace(resource.Spec.CompartmentId)
		computeID = strings.TrimSpace(resource.Spec.ComputeId)
		managementAgentID = strings.TrimSpace(resource.Spec.ManagementAgentId)
	}
	return opsisdk.MacsManagedCloudHostInsight{
		Id:                common.String(currentID),
		CompartmentId:     common.String(compartmentID),
		HostName:          common.String("unknown"),
		FreeformTags:      map[string]string{},
		DefinedTags:       map[string]map[string]interface{}{},
		ComputeId:         common.String(computeID),
		ManagementAgentId: common.String(managementAgentID),
		Status:            opsisdk.ResourceStatusEnabled,
		LifecycleState:    opsisdk.LifecycleStateActive,
	}
}

func conservativeHostInsightNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return hostInsightAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      hostInsightAmbiguousNotFoundCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted",
			hostInsightKind,
			strings.TrimSpace(operation),
		),
	}
}

func isHostInsightAmbiguousNotFound(err error) bool {
	if err == nil {
		return false
	}
	switch typed := err.(type) {
	case hostInsightAmbiguousNotFoundError:
		return typed.ErrorCode == hostInsightAmbiguousNotFoundCode
	case *hostInsightAmbiguousNotFoundError:
		return typed != nil && typed.ErrorCode == hostInsightAmbiguousNotFoundCode
	default:
		return false
	}
}

func getHostInsightWorkRequest(
	ctx context.Context,
	client hostInsightOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", hostInsightKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", hostInsightKind)
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveHostInsightWorkRequestAction(workRequest any) (string, error) {
	current, err := hostInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveHostInsightWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := hostInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case opsisdk.OperationTypeCreateHostInsight, opsisdk.OperationTypeEnableHostInsight:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case opsisdk.OperationTypeUpdateHostInsight:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case opsisdk.OperationTypeDeleteHostInsight, opsisdk.OperationTypeDisableHostInsight:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverHostInsightIDFromWorkRequest(
	_ *opsiv1beta1.HostInsight,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := hostInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if id, ok := resolveHostInsightIDFromWorkRequestResources(current.Resources, phase, true); ok {
		return id, nil
	}
	if id, ok := resolveHostInsightIDFromWorkRequestResources(current.Resources, phase, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a host insight identifier", hostInsightKind, hostInsightString(current.Id))
}

func resolveHostInsightIDFromWorkRequestResources(
	resources []opsisdk.WorkRequestResource,
	phase shared.OSOKAsyncPhase,
	requireActionMatch bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if !isHostInsightWorkRequestResource(resource) {
			continue
		}
		if requireActionMatch && !hostInsightWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		id := strings.TrimSpace(hostInsightString(resource.Identifier))
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

func isHostInsightWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(hostInsightString(resource.EntityType)))
	if strings.Contains(entityType, "host") && strings.Contains(entityType, "insight") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(hostInsightString(resource.EntityUri)))
	return strings.Contains(entityURI, "/hostinsights/")
}

func hostInsightWorkRequestActionMatchesPhase(action opsisdk.ActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == opsisdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return action == opsisdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return action == opsisdk.ActionTypeDeleted
	default:
		return false
	}
}

func hostInsightWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := hostInsightWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", hostInsightKind, phase, hostInsightString(current.Id), current.Status)
}

func buildHostInsightDeleteWorkRequestStatus(
	resource *opsiv1beta1.HostInsight,
	workRequest opsisdk.WorkRequest,
) (*shared.OSOKAsyncOperation, error) {
	if phase, ok, err := resolveHostInsightWorkRequestPhase(workRequest); err != nil {
		return nil, err
	} else if ok && phase != shared.OSOKAsyncPhaseDelete {
		return nil, fmt.Errorf("%s work request %s is for phase %q, want delete", hostInsightKind, hostInsightString(workRequest.Id), phase)
	}
	return servicemanager.BuildWorkRequestAsyncOperation(
		&resource.Status.OsokStatus,
		hostInsightWorkRequestAsyncAdapter,
		servicemanager.WorkRequestAsyncInput{
			RawStatus:        string(workRequest.Status),
			RawAction:        string(workRequest.OperationType),
			RawOperationType: string(workRequest.OperationType),
			WorkRequestID:    hostInsightString(workRequest.Id),
			PercentComplete:  workRequest.PercentComplete,
			Message:          hostInsightWorkRequestMessage(shared.OSOKAsyncPhaseDelete, workRequest),
			FallbackPhase:    shared.OSOKAsyncPhaseDelete,
		},
	)
}

func hostInsightWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("%s work request is nil", hostInsightKind)
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", hostInsightKind, workRequest)
	}
}

func hostInsightCreatePayload(spec opsiv1beta1.HostInsightSpec) (map[string]any, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return nil, fmt.Errorf("%s spec.jsonData is not supported; use structured HostInsight spec fields", hostInsightKind)
	}
	payload := map[string]any{}
	setStringPayload(payload, "compartmentId", spec.CompartmentId)
	setStringPayload(payload, "entitySource", spec.EntitySource)
	setStringPayload(payload, "computeId", spec.ComputeId)
	setStringPayload(payload, "managementAgentId", spec.ManagementAgentId)
	setStringPayload(payload, "enterpriseManagerIdentifier", spec.EnterpriseManagerIdentifier)
	setStringPayload(payload, "enterpriseManagerBridgeId", spec.EnterpriseManagerBridgeId)
	setStringPayload(payload, "enterpriseManagerEntityIdentifier", spec.EnterpriseManagerEntityIdentifier)
	setStringPayload(payload, "exadataInsightId", spec.ExadataInsightId)
	if spec.FreeformTags != nil {
		payload["freeformTags"] = hostInsightCloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		payload["definedTags"] = hostInsightDefinedTagsFromSpec(spec.DefinedTags)
	}
	source := normalizeHostInsightEntitySource(hostInsightStringFromPayload(payload, "entitySource"))
	if source == "" {
		source = inferHostInsightEntitySource(payload)
	}
	payload["entitySource"] = source
	return payload, nil
}

func hostInsightSpecJSONData(resource *opsiv1beta1.HostInsight) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Spec.JsonData)
}

func inferHostInsightEntitySource(payload map[string]any) string {
	switch {
	case hostInsightStringFromPayload(payload, "computeId") != "":
		return hostInsightEntitySourceMacsManagedCloudHost
	case hostInsightStringFromPayload(payload, "enterpriseManagerIdentifier") != "" ||
		hostInsightStringFromPayload(payload, "enterpriseManagerBridgeId") != "" ||
		hostInsightStringFromPayload(payload, "enterpriseManagerEntityIdentifier") != "":
		return hostInsightEntitySourceEmManagedExternalHost
	case hostInsightStringFromPayload(payload, "managementAgentId") != "":
		return hostInsightEntitySourceMacsManagedExternalHost
	default:
		return ""
	}
}

func normalizeHostInsightEntitySource(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value)
}

func normalizeHostInsightComparable(field string, value string) string {
	if field == "entitySource" {
		return normalizeHostInsightEntitySource(value)
	}
	return strings.TrimSpace(value)
}

func unmarshalHostInsightPayload(payload map[string]any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func requireHostInsightPayloadStrings(prefix string, payload map[string]any, fields ...string) error {
	var missing []string
	for _, field := range fields {
		if strings.TrimSpace(hostInsightStringFromPayload(payload, field)) == "" {
			missing = append(missing, prefix+"."+field)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", hostInsightKind, strings.Join(missing, ", "))
}

func setStringPayload(payload map[string]any, field string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		payload[field] = value
	}
}

func desiredHostInsightFreeformTagsUpdate(spec map[string]string, current any) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	currentTags := hostInsightStringMapFromAny(current)
	if len(spec) == 0 && len(currentTags) == 0 {
		return nil, false
	}
	if reflect.DeepEqual(spec, currentTags) {
		return nil, false
	}
	return hostInsightCloneStringMap(spec), true
}

func desiredHostInsightDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current any,
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := hostInsightDefinedTagsFromSpec(spec)
	currentTags := hostInsightDefinedTagsFromAny(current)
	if len(desired) == 0 && len(currentTags) == 0 {
		return nil, false
	}
	if hostInsightJSONEqual(desired, currentTags) {
		return nil, false
	}
	return desired, true
}

func hostInsightBodyMap(value any) (map[string]any, bool) {
	body := hostInsightBodyValue(value)
	if body == nil {
		return nil, false
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false
	}
	if len(decoded) == 0 {
		return nil, false
	}
	preserveHostInsightEntitySource(decoded, body)
	if _, hasID := decoded["id"]; hasID {
		return decoded, true
	}
	if _, hasSource := decoded["entitySource"]; hasSource {
		return decoded, true
	}
	return decoded, decoded["freeformTags"] != nil || decoded["definedTags"] != nil
}

func preserveHostInsightEntitySource(values map[string]any, value any) {
	if values == nil {
		return
	}
	source := normalizeHostInsightEntitySource(hostInsightStringFromMap(values, "entitySource"))
	if source == "" {
		source = hostInsightEntitySourceFromSDKValue(value)
	}
	if source != "" {
		values["entitySource"] = source
	}
}

func hostInsightEntitySourceFromSDKValue(value any) string {
	switch typed := hostInsightIndirect(value).(type) {
	case hostInsightStatusAlias:
		return hostInsightEntitySourceFromSDKValue(typed.HostInsight)
	case hostInsightSummaryStatusAlias:
		return hostInsightEntitySourceFromSDKValue(typed.HostInsightSummary)
	case opsisdk.MacsManagedCloudHostInsight, opsisdk.MacsManagedCloudHostInsightSummary:
		return hostInsightEntitySourceMacsManagedCloudHost
	case opsisdk.MacsManagedExternalHostInsight, opsisdk.MacsManagedExternalHostInsightSummary:
		return hostInsightEntitySourceMacsManagedExternalHost
	case opsisdk.EmManagedExternalHostInsight, opsisdk.EmManagedExternalHostInsightSummary:
		return hostInsightEntitySourceEmManagedExternalHost
	case opsisdk.MacsManagedCloudDatabaseHostInsight, opsisdk.MacsManagedCloudDatabaseHostInsightSummary:
		return hostInsightEntitySourceMacsManagedCloudDBHost
	case opsisdk.PeComanagedHostInsight, opsisdk.PeComanagedHostInsightSummary:
		return hostInsightEntitySourcePeComanagedHost
	default:
		return ""
	}
}

func hostInsightBodyValue(value any) any {
	value = hostInsightIndirect(value)
	switch typed := value.(type) {
	case opsisdk.CreateHostInsightResponse:
		return typed.HostInsight
	case opsisdk.GetHostInsightResponse:
		return typed.HostInsight
	case opsisdk.HostInsightSummary:
		return typed
	case opsisdk.HostInsight:
		return typed
	default:
		return value
	}
}

func hostInsightWithSDKStatusAlias(item opsisdk.HostInsight) opsisdk.HostInsight {
	if item == nil {
		return nil
	}
	switch item.(type) {
	case hostInsightStatusAlias, *hostInsightStatusAlias:
		return item
	default:
		return hostInsightStatusAlias{HostInsight: item}
	}
}

func hostInsightSummaryWithSDKStatusAlias(item opsisdk.HostInsightSummary) opsisdk.HostInsightSummary {
	if item == nil {
		return nil
	}
	switch item.(type) {
	case hostInsightSummaryStatusAlias, *hostInsightSummaryStatusAlias:
		return item
	default:
		return hostInsightSummaryStatusAlias{HostInsightSummary: item}
	}
}

func (item hostInsightStatusAlias) MarshalJSON() ([]byte, error) {
	if item.HostInsight == nil {
		return []byte("null"), nil
	}
	return hostInsightSDKStatusAliasPayload(item.HostInsight)
}

func (item hostInsightSummaryStatusAlias) MarshalJSON() ([]byte, error) {
	if item.HostInsightSummary == nil {
		return []byte("null"), nil
	}
	return hostInsightSDKStatusAliasPayload(item.HostInsightSummary)
}

func hostInsightSDKStatusAliasPayload(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return rewriteHostInsightSDKStatusPayload(payload), nil
}

func rewriteHostInsightSDKStatusPayload(payload []byte) []byte {
	values := map[string]json.RawMessage{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return payload
	}
	rawStatus, ok := values["status"]
	if !ok || len(rawStatus) == 0 {
		return payload
	}
	trimmed := strings.TrimSpace(string(rawStatus))
	if trimmed == "" || strings.HasPrefix(trimmed, "{") {
		return payload
	}
	if _, exists := values["sdkStatus"]; !exists {
		values["sdkStatus"] = rawStatus
	}
	delete(values, "status")
	rewritten, err := json.Marshal(values)
	if err != nil {
		return payload
	}
	return rewritten
}

func hostInsightIndirect(value any) any {
	if value == nil {
		return nil
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() != reflect.Pointer || reflected.IsNil() {
		return value
	}
	return reflected.Elem().Interface()
}

func hostInsightItemDeleted(values map[string]any) bool {
	state := strings.ToUpper(hostInsightStringFromMap(values, "lifecycleState"))
	status := strings.ToUpper(hostInsightStatusFromMap(values))
	return state == string(opsisdk.LifecycleStateDeleted) || status == string(opsisdk.ResourceStatusTerminated)
}

func hostInsightStatusFromMap(values map[string]any) string {
	if status := hostInsightStringFromMap(values, "status"); status != "" {
		return status
	}
	return hostInsightStringFromMap(values, "sdkStatus")
}

func hostInsightStringFromPayload(payload map[string]any, field string) string {
	if payload == nil {
		return ""
	}
	return hostInsightStringValue(payload[field])
}

func hostInsightStringFromMap(values map[string]any, field string) string {
	if values == nil {
		return ""
	}
	return hostInsightStringValue(values[field])
}

func hostInsightStringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func hostInsightString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func hostInsightStringMapFromAny(value any) map[string]string {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	decoded := map[string]string{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func hostInsightDefinedTagsFromAny(value any) map[string]map[string]interface{} {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	decoded := map[string]map[string]interface{}{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func hostInsightDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func hostInsightCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func hostInsightJSONEqual(left any, right any) bool {
	leftValue, leftErr := normalizedHostInsightJSONValue(left)
	rightValue, rightErr := normalizedHostInsightJSONValue(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func normalizedHostInsightJSONValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
