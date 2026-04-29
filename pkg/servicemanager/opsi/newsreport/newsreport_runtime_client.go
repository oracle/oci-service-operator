/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package newsreport

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
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

const newsReportKind = "NewsReport"

var newsReportWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opsisdk.OperationTypeCreateNewsReport)},
	UpdateActionTokens: []string{
		string(opsisdk.OperationTypeUpdateNewsReport),
		string(opsisdk.OperationTypeEnableNewsReport),
		string(opsisdk.OperationTypeDisableNewsReport),
		string(opsisdk.OperationTypeMoveNewsReport),
	},
	DeleteActionTokens: []string{string(opsisdk.OperationTypeDeleteNewsReport)},
}

type newsReportOCIClient interface {
	CreateNewsReport(context.Context, opsisdk.CreateNewsReportRequest) (opsisdk.CreateNewsReportResponse, error)
	GetNewsReport(context.Context, opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error)
	ListNewsReports(context.Context, opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error)
	UpdateNewsReport(context.Context, opsisdk.UpdateNewsReportRequest) (opsisdk.UpdateNewsReportResponse, error)
	DeleteNewsReport(context.Context, opsisdk.DeleteNewsReportRequest) (opsisdk.DeleteNewsReportResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type newsReportWorkRequestClient interface {
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type newsReportRuntimeClient struct {
	delegate           NewsReportServiceClient
	generatedInitErr   error
	hooks              NewsReportRuntimeHooks
	workRequestClient  newsReportWorkRequestClient
	workRequestInitErr error
	log                loggerutil.OSOKLogger
}

type newsReportRuntimeReadResponse struct {
	Body         map[string]any `presentIn:"body"`
	OpcRequestId *string        `presentIn:"header" name:"opc-request-id"`
	Etag         *string        `presentIn:"header" name:"etag"`
}

type newsReportRuntimeListBody struct {
	Items []map[string]any `json:"items"`
}

type newsReportRuntimeListResponse struct {
	Body         newsReportRuntimeListBody `presentIn:"body"`
	OpcRequestId *string                   `presentIn:"header" name:"opc-request-id"`
	OpcNextPage  *string                   `presentIn:"header" name:"opc-next-page"`
}

type newsReportAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e newsReportAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e newsReportAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var _ NewsReportServiceClient = (*newsReportRuntimeClient)(nil)

func init() {
	registerNewsReportRuntimeHooksMutator(func(manager *NewsReportServiceManager, hooks *NewsReportRuntimeHooks) {
		workRequestClient, initErr := newNewsReportWorkRequestClient(manager)
		applyNewsReportRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newNewsReportWorkRequestClient(manager *NewsReportServiceManager) (newsReportWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", newsReportKind)
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyNewsReportRuntimeHooks(
	manager *NewsReportServiceManager,
	hooks *NewsReportRuntimeHooks,
	workRequestClient newsReportWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newNewsReportRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *opsiv1beta1.NewsReport, _ string) (any, error) {
		return buildNewsReportCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *opsiv1beta1.NewsReport,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildNewsReportUpdateBody(resource, currentResponse)
	}
	if hooks.List.Call != nil {
		hooks.List.Call = listNewsReportsAllPages(hooks.List.Call)
	}
	hooks.List.Fields = newsReportListFields()
	hooks.Read.Get = newsReportGetReadOperation(hooks)
	hooks.Read.List = newsReportListReadOperation(hooks)
	hooks.StatusHooks.ProjectStatus = projectNewsReportStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateNewsReportCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleNewsReportDeleteError
	hooks.Async.Adapter = newsReportWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getNewsReportWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveNewsReportWorkRequestAction
	hooks.Async.RecoverResourceID = recoverNewsReportIDFromWorkRequest
	hooks.Async.Message = newsReportWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate NewsReportServiceClient) NewsReportServiceClient {
		runtimeClient := &newsReportRuntimeClient{
			delegate:           delegate,
			generatedInitErr:   newsReportGeneratedDelegateInitError(delegate),
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

func newNewsReportRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "newsreport",
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
			ProvisioningStates: []string{string(opsisdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "onsTopicId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"status",
				"newsFrequency",
				"locale",
				"contentTypes",
				"onsTopicId",
				"freeformTags",
				"definedTags",
				"name",
				"description",
				"dayOfWeek",
				"areChildCompartmentsIncluded",
				"tagFilters",
				"matchRule",
			},
			ForceNew: []string{"compartmentId"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func newNewsReportServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client newsReportOCIClient,
) NewsReportServiceClient {
	manager := &NewsReportServiceManager{Log: log}
	hooks := newNewsReportRuntimeHooksWithOCIClient(client)
	applyNewsReportRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultNewsReportServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.NewsReport](
			buildNewsReportGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapNewsReportGeneratedClient(hooks, delegate)
}

func newNewsReportRuntimeHooksWithOCIClient(client newsReportOCIClient) NewsReportRuntimeHooks {
	return NewsReportRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.NewsReport]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.NewsReport]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.NewsReport]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.NewsReport]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.NewsReport]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.NewsReport]{},
		Create: runtimeOperationHooks[opsisdk.CreateNewsReportRequest, opsisdk.CreateNewsReportResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateNewsReportDetails", RequestName: "CreateNewsReportDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request opsisdk.CreateNewsReportRequest) (opsisdk.CreateNewsReportResponse, error) {
				return client.CreateNewsReport(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetNewsReportRequest, opsisdk.GetNewsReportResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NewsReportId", RequestName: "newsReportId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetNewsReportRequest) (opsisdk.GetNewsReportResponse, error) {
				return client.GetNewsReport(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListNewsReportsRequest, opsisdk.ListNewsReportsResponse]{
			Fields: newsReportListFields(),
			Call: func(ctx context.Context, request opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error) {
				return client.ListNewsReports(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateNewsReportRequest, opsisdk.UpdateNewsReportResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NewsReportId", RequestName: "newsReportId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateNewsReportDetails", RequestName: "UpdateNewsReportDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request opsisdk.UpdateNewsReportRequest) (opsisdk.UpdateNewsReportResponse, error) {
				return client.UpdateNewsReport(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteNewsReportRequest, opsisdk.DeleteNewsReportResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NewsReportId", RequestName: "newsReportId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteNewsReportRequest) (opsisdk.DeleteNewsReportResponse, error) {
				return client.DeleteNewsReport(ctx, request)
			},
		},
		WrapGeneratedClient: []func(NewsReportServiceClient) NewsReportServiceClient{},
	}
}

func (c *newsReportRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", newsReportKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *newsReportRuntimeClient) Delete(ctx context.Context, resource *opsiv1beta1.NewsReport) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("%s runtime client is not configured", newsReportKind)
	}
	if c.generatedInitErr != nil {
		return false, c.generatedInitErr
	}
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", newsReportKind)
	}

	if current := resource.Status.OsokStatus.Async.Current; current != nil && current.Source == shared.OSOKAsyncSourceWorkRequest && current.WorkRequestID != "" {
		switch current.Phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			return c.waitForPendingWriteBeforeDelete(ctx, resource, current)
		case shared.OSOKAsyncPhaseDelete:
			return c.resumeDeleteWorkRequest(ctx, resource, current.WorkRequestID)
		}
	}
	return c.deleteResolvedNewsReport(ctx, resource)
}

func newsReportGeneratedDelegateInitError(delegate NewsReportServiceClient) error {
	if delegate == nil {
		return nil
	}

	var resource *opsiv1beta1.NewsReport
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || newsReportIsNilResourceProbeError(err) {
		return nil
	}
	return err
}

func newsReportIsNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

func (c *newsReportRuntimeClient) waitForPendingWriteBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	tracked *shared.OSOKAsyncOperation,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, tracked.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := newsReportAsyncOperation(&resource.Status.OsokStatus, workRequest, tracked.Phase)
	if err != nil {
		return false, err
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequest(resource, current)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		c.applyWorkRequest(resource, current)
		currentNewsReport, found, err := c.resolveNewsReportAfterSucceededWrite(ctx, resource, workRequest, current)
		if err != nil {
			return false, err
		}
		if !found {
			c.markWriteWorkRequestReadbackPending(resource, current, tracked.WorkRequestID, newsReportIDFromWorkRequest(workRequest, tracked.Phase))
			return false, nil
		}
		if newsReportLifecyclePending(currentNewsReport) {
			c.projectStatus(resource, currentNewsReport)
			c.markTerminating(resource, "OCI write is still settling before delete", currentNewsReport)
			return false, nil
		}
		resource.Status.OsokStatus.Async.Current = nil
		return c.deleteResolvedNewsReport(ctx, resource)
	default:
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s %s work request %s finished with status %s", newsReportKind, current.Phase, tracked.WorkRequestID, current.RawStatus)
	}
}

func (c *newsReportRuntimeClient) resolveNewsReportAfterSucceededWrite(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	workRequest opsisdk.WorkRequest,
	current *shared.OSOKAsyncOperation,
) (opsisdk.NewsReport, bool, error) {
	resourceID := newsReportIDFromWorkRequest(workRequest, current.Phase)
	if resourceID == "" {
		return c.resolveNewsReportForDelete(ctx, resource)
	}

	live, found, err := c.getNewsReportForDelete(ctx, resource, resourceID)
	if err != nil || found {
		return live, found, err
	}
	return opsisdk.NewsReport{}, false, nil
}

func (c *newsReportRuntimeClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	current, err := newsReportAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.applyWorkRequest(resource, current)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		c.applyWorkRequest(resource, current)
		resourceID := currentNewsReportID(resource)
		if resourceID == "" {
			resourceID = newsReportIDFromWorkRequest(workRequest, shared.OSOKAsyncPhaseDelete)
		}
		return c.confirmDeleted(ctx, resource, resourceID, "OCI resource deleted")
	default:
		c.applyWorkRequest(resource, current)
		return false, fmt.Errorf("%s delete work request %s finished with status %s", newsReportKind, workRequestID, current.RawStatus)
	}
}

func (c *newsReportRuntimeClient) deleteResolvedNewsReport(ctx context.Context, resource *opsiv1beta1.NewsReport) (bool, error) {
	current, found, err := c.resolveNewsReportForDelete(ctx, resource)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, fmt.Sprintf("OCI %s no longer exists", newsReportKind))
		return true, nil
	}
	c.projectStatus(resource, current)
	if newsReportLifecycleDeleted(current) {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	if newsReportLifecycleDeleting(current) {
		c.markTerminating(resource, "OCI resource delete is in progress", current)
		return false, nil
	}

	resourceID := newsReportStringValue(current.Id)
	response, err := c.hooks.Delete.Call(ctx, opsisdk.DeleteNewsReportRequest{NewsReportId: common.String(resourceID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if newsReportIsUnambiguousNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, handleNewsReportDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := newsReportStringValue(response.OpcWorkRequestId); workRequestID != "" {
		c.markWorkRequest(resource, workRequestID, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "")
		return false, nil
	}
	return c.confirmDeleted(ctx, resource, resourceID, "OCI resource deleted")
}

func (c *newsReportRuntimeClient) confirmDeleted(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	resourceID string,
	deletedMessage string,
) (bool, error) {
	current, found, err := c.getNewsReportForDelete(ctx, resource, resourceID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, deletedMessage)
		return true, nil
	}
	c.projectStatus(resource, current)
	if newsReportLifecycleDeleted(current) {
		c.markDeleted(resource, deletedMessage)
		return true, nil
	}
	c.markTerminating(resource, "OCI resource delete is in progress", current)
	return false, nil
}

func (c *newsReportRuntimeClient) resolveNewsReportForDelete(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
) (opsisdk.NewsReport, bool, error) {
	if resourceID := currentNewsReportID(resource); resourceID != "" {
		return c.getNewsReportForDelete(ctx, resource, resourceID)
	}

	response, found, err := c.listNewsReportsForDelete(ctx, resource)
	if err != nil || !found {
		return opsisdk.NewsReport{}, false, err
	}
	matches := newsReportSummariesMatchingSpec(response.Items, resource.Spec)
	return c.resolveListedNewsReportForDelete(ctx, resource, matches)
}

func (c *newsReportRuntimeClient) listNewsReportsForDelete(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
) (opsisdk.ListNewsReportsResponse, bool, error) {
	if c.hooks.List.Call == nil {
		return opsisdk.ListNewsReportsResponse{}, false, nil
	}

	response, err := c.hooks.List.Call(ctx, opsisdk.ListNewsReportsRequest{
		CompartmentId: common.String(resource.Spec.CompartmentId),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if newsReportIsUnambiguousNotFound(err) {
			return opsisdk.ListNewsReportsResponse{}, false, nil
		}
		return opsisdk.ListNewsReportsResponse{}, false, handleNewsReportDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response, true, nil
}

func newsReportSummariesMatchingSpec(
	items []opsisdk.NewsReportSummary,
	spec opsiv1beta1.NewsReportSpec,
) []opsisdk.NewsReportSummary {
	matches := make([]opsisdk.NewsReportSummary, 0, len(items))
	for _, item := range items {
		if newsReportSummaryMatchesSpec(item, spec) {
			matches = append(matches, item)
		}
	}
	return matches
}

func (c *newsReportRuntimeClient) resolveListedNewsReportForDelete(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	matches []opsisdk.NewsReportSummary,
) (opsisdk.NewsReport, bool, error) {
	switch len(matches) {
	case 0:
		return opsisdk.NewsReport{}, false, nil
	case 1:
		return c.resolveNewsReportSummaryForDelete(ctx, resource, matches[0])
	default:
		return opsisdk.NewsReport{}, false, fmt.Errorf(
			"multiple OCI %ss matched compartmentId %q, name %q, and onsTopicId %q",
			newsReportKind,
			resource.Spec.CompartmentId,
			resource.Spec.Name,
			resource.Spec.OnsTopicId,
		)
	}
}

func (c *newsReportRuntimeClient) resolveNewsReportSummaryForDelete(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	summary opsisdk.NewsReportSummary,
) (opsisdk.NewsReport, bool, error) {
	id := newsReportStringValue(summary.Id)
	if id == "" || c.hooks.Get.Call == nil {
		return newsReportFromSummary(summary), true, nil
	}
	current, found, err := c.getNewsReportForDelete(ctx, resource, id)
	if err != nil || found {
		return current, found, err
	}
	return newsReportFromSummary(summary), true, nil
}

func (c *newsReportRuntimeClient) getNewsReportForDelete(
	ctx context.Context,
	resource *opsiv1beta1.NewsReport,
	resourceID string,
) (opsisdk.NewsReport, bool, error) {
	if resourceID == "" || c.hooks.Get.Call == nil {
		return opsisdk.NewsReport{}, false, nil
	}
	response, err := c.hooks.Get.Call(ctx, opsisdk.GetNewsReportRequest{NewsReportId: common.String(resourceID)})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if newsReportIsUnambiguousNotFound(err) {
			return opsisdk.NewsReport{}, false, nil
		}
		return opsisdk.NewsReport{}, false, handleNewsReportDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response.NewsReport, true, nil
}

func (c *newsReportRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (opsisdk.WorkRequest, error) {
	return getNewsReportWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, workRequestID)
}

func (c *newsReportRuntimeClient) projectStatus(resource *opsiv1beta1.NewsReport, current opsisdk.NewsReport) {
	_ = projectNewsReportSDKStatus(resource, current)
}

func (c *newsReportRuntimeClient) markWorkRequest(
	resource *opsiv1beta1.NewsReport,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	rawStatus string,
) {
	message := fmt.Sprintf("%s %s work request %s is pending", newsReportKind, phase, workRequestID)
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

func (c *newsReportRuntimeClient) applyWorkRequest(resource *opsiv1beta1.NewsReport, current *shared.OSOKAsyncOperation) {
	if current == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
}

func (c *newsReportRuntimeClient) markWriteWorkRequestReadbackPending(
	resource *opsiv1beta1.NewsReport,
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
	next.Message = newsReportWriteReadbackPendingMessage(current.Phase, workRequestID, resourceID)
	next.UpdatedAt = &now

	if strings.TrimSpace(resourceID) != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, c.log)
}

func newsReportWriteReadbackPendingMessage(
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	resourceID string,
) string {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Sprintf(
			"%s %s work request %s succeeded; waiting for %s readback before delete",
			newsReportKind,
			phase,
			strings.TrimSpace(workRequestID),
			newsReportKind,
		)
	}
	return fmt.Sprintf(
		"%s %s work request %s succeeded; waiting for %s %s to become readable before delete",
		newsReportKind,
		phase,
		strings.TrimSpace(workRequestID),
		newsReportKind,
		strings.TrimSpace(resourceID),
	)
}

func (c *newsReportRuntimeClient) markDeleted(resource *opsiv1beta1.NewsReport, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Async.Current = nil
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, c.log)
}

func (c *newsReportRuntimeClient) markTerminating(
	resource *opsiv1beta1.NewsReport,
	message string,
	current opsisdk.NewsReport,
) {
	rawStatus := strings.ToUpper(string(current.LifecycleState))
	if rawStatus == "" {
		rawStatus = string(opsisdk.LifecycleStateDeleting)
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

func buildNewsReportCreateBody(resource *opsiv1beta1.NewsReport) (opsisdk.CreateNewsReportDetails, error) {
	if resource == nil {
		return opsisdk.CreateNewsReportDetails{}, fmt.Errorf("%s resource is nil", newsReportKind)
	}

	spec := resource.Spec
	details := opsisdk.CreateNewsReportDetails{
		Name:                         common.String(spec.Name),
		NewsFrequency:                opsisdk.NewsFrequencyEnum(strings.TrimSpace(spec.NewsFrequency)),
		Description:                  common.String(spec.Description),
		OnsTopicId:                   common.String(spec.OnsTopicId),
		CompartmentId:                common.String(spec.CompartmentId),
		ContentTypes:                 newsReportContentTypesFromSpec(spec.ContentTypes),
		Locale:                       opsisdk.NewsLocaleEnum(strings.TrimSpace(spec.Locale)),
		FreeformTags:                 maps.Clone(spec.FreeformTags),
		DefinedTags:                  newsReportDefinedTagsFromSpec(spec.DefinedTags),
		AreChildCompartmentsIncluded: common.Bool(spec.AreChildCompartmentsIncluded),
		TagFilters:                   slices.Clone(spec.TagFilters),
	}
	if strings.TrimSpace(spec.Status) != "" {
		details.Status = opsisdk.ResourceStatusEnum(strings.TrimSpace(spec.Status))
	}
	if strings.TrimSpace(spec.DayOfWeek) != "" {
		details.DayOfWeek = opsisdk.DayOfWeekEnum(strings.TrimSpace(spec.DayOfWeek))
	}
	if strings.TrimSpace(spec.MatchRule) != "" {
		details.MatchRule = opsisdk.MatchRuleEnum(strings.TrimSpace(spec.MatchRule))
	}
	return details, nil
}

func buildNewsReportUpdateBody(
	resource *opsiv1beta1.NewsReport,
	currentResponse any,
) (opsisdk.UpdateNewsReportDetails, bool, error) {
	if resource == nil {
		return opsisdk.UpdateNewsReportDetails{}, false, fmt.Errorf("%s resource is nil", newsReportKind)
	}

	current, _, err := newsReportFromCurrentState(resource, currentResponse)
	if err != nil {
		return opsisdk.UpdateNewsReportDetails{}, false, err
	}

	spec := resource.Spec
	builder := newsReportUpdateBuilder{}
	builder.applyScheduleUpdates(spec, current)
	builder.applyContentUpdates(spec, current)
	builder.applyTagUpdates(spec, current)
	builder.applyNotificationUpdates(spec, current)
	return builder.details, builder.updateNeeded, nil
}

type newsReportUpdateBuilder struct {
	details      opsisdk.UpdateNewsReportDetails
	updateNeeded bool
}

func (b *newsReportUpdateBuilder) applyScheduleUpdates(spec opsiv1beta1.NewsReportSpec, current opsisdk.NewsReport) {
	if setEnumUpdate(strings.TrimSpace(spec.Status), string(current.Status)) {
		b.details.Status = opsisdk.ResourceStatusEnum(strings.TrimSpace(spec.Status))
		b.updateNeeded = true
	}
	if setEnumUpdate(strings.TrimSpace(spec.NewsFrequency), string(current.NewsFrequency)) {
		b.details.NewsFrequency = opsisdk.NewsFrequencyEnum(strings.TrimSpace(spec.NewsFrequency))
		b.updateNeeded = true
	}
	if setEnumUpdate(strings.TrimSpace(spec.DayOfWeek), string(current.DayOfWeek)) {
		b.details.DayOfWeek = opsisdk.DayOfWeekEnum(strings.TrimSpace(spec.DayOfWeek))
		b.updateNeeded = true
	}
}

func (b *newsReportUpdateBuilder) applyContentUpdates(spec opsiv1beta1.NewsReportSpec, current opsisdk.NewsReport) {
	if setEnumUpdate(strings.TrimSpace(spec.Locale), string(current.Locale)) {
		b.details.Locale = opsisdk.NewsLocaleEnum(strings.TrimSpace(spec.Locale))
		b.updateNeeded = true
	}
	if desired := newsReportContentTypesFromSpec(spec.ContentTypes); !newsReportContentTypesEqual(desired, current.ContentTypes) {
		b.details.ContentTypes = desired
		b.updateNeeded = true
	}
	if setStringUpdate(spec.Name, newsReportStringValue(current.Name)) {
		b.details.Name = common.String(strings.TrimSpace(spec.Name))
		b.updateNeeded = true
	}
	if setStringUpdate(spec.Description, newsReportStringValue(current.Description)) {
		b.details.Description = common.String(strings.TrimSpace(spec.Description))
		b.updateNeeded = true
	}
}

func (b *newsReportUpdateBuilder) applyTagUpdates(spec opsiv1beta1.NewsReportSpec, current opsisdk.NewsReport) {
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		b.details.FreeformTags = maps.Clone(spec.FreeformTags)
		b.updateNeeded = true
	}
	if spec.DefinedTags != nil {
		b.applyDefinedTags(spec.DefinedTags, current.DefinedTags)
	}
}

func (b *newsReportUpdateBuilder) applyDefinedTags(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) {
	definedTags := newsReportDefinedTagsFromSpec(spec)
	if newsReportDefinedTagsEqual(definedTags, current) {
		return
	}
	b.details.DefinedTags = definedTags
	b.updateNeeded = true
}

func (b *newsReportUpdateBuilder) applyNotificationUpdates(
	spec opsiv1beta1.NewsReportSpec,
	current opsisdk.NewsReport,
) {
	if setStringUpdate(spec.OnsTopicId, newsReportStringValue(current.OnsTopicId)) {
		b.details.OnsTopicId = common.String(strings.TrimSpace(spec.OnsTopicId))
		b.updateNeeded = true
	}
	if current.AreChildCompartmentsIncluded == nil || *current.AreChildCompartmentsIncluded != spec.AreChildCompartmentsIncluded {
		b.details.AreChildCompartmentsIncluded = common.Bool(spec.AreChildCompartmentsIncluded)
		b.updateNeeded = true
	}
	if spec.TagFilters != nil && !slices.Equal(spec.TagFilters, current.TagFilters) {
		b.details.TagFilters = slices.Clone(spec.TagFilters)
		b.updateNeeded = true
	}
	if setEnumUpdate(strings.TrimSpace(spec.MatchRule), string(current.MatchRule)) {
		b.details.MatchRule = opsisdk.MatchRuleEnum(strings.TrimSpace(spec.MatchRule))
		b.updateNeeded = true
	}
}

func setStringUpdate(desired string, current string) bool {
	desired = strings.TrimSpace(desired)
	return desired != "" && desired != strings.TrimSpace(current)
}

func setEnumUpdate(desired string, current string) bool {
	return desired != "" && desired != strings.TrimSpace(current)
}

func validateNewsReportCreateOnlyDriftForResponse(resource *opsiv1beta1.NewsReport, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", newsReportKind)
	}
	current, _, err := newsReportFromCurrentState(resource, currentResponse)
	if err != nil {
		return err
	}

	if strings.TrimSpace(resource.Spec.CompartmentId) == strings.TrimSpace(newsReportStringValue(current.CompartmentId)) {
		return nil
	}
	return fmt.Errorf("%s create-only drift detected for compartmentId; replace the resource or restore the desired spec before update", newsReportKind)
}

func listNewsReportsAllPages(
	call func(context.Context, opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error),
) func(context.Context, opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error) {
	return func(ctx context.Context, request opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error) {
		return listNewsReportPages(ctx, call, request)
	}
}

func listNewsReportPages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListNewsReportsRequest) (opsisdk.ListNewsReportsResponse, error),
	request opsisdk.ListNewsReportsRequest,
) (opsisdk.ListNewsReportsResponse, error) {
	if call == nil {
		return opsisdk.ListNewsReportsResponse{}, fmt.Errorf("%s list operation is not configured", newsReportKind)
	}

	seenPages := map[string]struct{}{}
	var combined opsisdk.ListNewsReportsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return opsisdk.ListNewsReportsResponse{}, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := strings.TrimSpace(newsReportStringValue(response.OpcNextPage))
		if nextPage == "" {
			return combined, nil
		}
		if _, exists := seenPages[nextPage]; exists {
			return opsisdk.ListNewsReportsResponse{}, fmt.Errorf("%s list pagination repeated page token %q", newsReportKind, nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

func newsReportListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
		{FieldName: "NewsReportId", RequestName: "newsReportId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
		{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query", PreferResourceID: false},
	}
}

func newsReportGetReadOperation(hooks *NewsReportRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.Get.Call == nil {
		return nil
	}
	fields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &opsisdk.GetNewsReportRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*opsisdk.GetNewsReportRequest)
			if !ok {
				return nil, fmt.Errorf("expected *opsi.GetNewsReportRequest, got %T", request)
			}
			response, err := hooks.Get.Call(ctx, *typed)
			if err != nil {
				return nil, err
			}
			return newsReportRuntimeReadResponse{
				Body:         newsReportStatusMap(response.NewsReport),
				OpcRequestId: response.OpcRequestId,
				Etag:         response.Etag,
			}, nil
		},
	}
}

func newsReportListReadOperation(hooks *NewsReportRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &opsisdk.ListNewsReportsRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*opsisdk.ListNewsReportsRequest)
			if !ok {
				return nil, fmt.Errorf("expected *opsi.ListNewsReportsRequest, got %T", request)
			}
			response, err := hooks.List.Call(ctx, *typed)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(response.Items))
			for _, item := range response.Items {
				items = append(items, newsReportStatusMap(newsReportFromSummary(item)))
			}
			return newsReportRuntimeListResponse{
				Body:         newsReportRuntimeListBody{Items: items},
				OpcRequestId: response.OpcRequestId,
				OpcNextPage:  response.OpcNextPage,
			}, nil
		},
	}
}

func projectNewsReportStatus(resource *opsiv1beta1.NewsReport, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", newsReportKind)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current, ok, err := newsReportFromResponse(response)
	if err != nil || !ok {
		return err
	}
	return projectNewsReportSDKStatus(resource, current)
}

func projectNewsReportSDKStatus(resource *opsiv1beta1.NewsReport, current opsisdk.NewsReport) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", newsReportKind)
	}

	status := &resource.Status
	status.NewsFrequency = string(current.NewsFrequency)
	status.ContentTypes = newsReportContentTypesStatusFromSDK(current.ContentTypes)
	status.Id = newsReportStringValue(current.Id)
	status.CompartmentId = newsReportStringValue(current.CompartmentId)
	status.OnsTopicId = newsReportStringValue(current.OnsTopicId)
	status.Locale = string(current.Locale)
	status.Description = newsReportStringValue(current.Description)
	status.Name = newsReportStringValue(current.Name)
	status.FreeformTags = maps.Clone(current.FreeformTags)
	status.DefinedTags = newsReportStatusTagsFromSDK(current.DefinedTags)
	status.SystemTags = newsReportStatusTagsFromSDK(current.SystemTags)
	status.Status = string(current.Status)
	status.TimeCreated = newsReportTimeString(current.TimeCreated)
	status.TimeUpdated = newsReportTimeString(current.TimeUpdated)
	status.LifecycleState = string(current.LifecycleState)
	status.LifecycleDetails = newsReportStringValue(current.LifecycleDetails)
	status.DayOfWeek = string(current.DayOfWeek)
	status.AreChildCompartmentsIncluded = newsReportBoolValue(current.AreChildCompartmentsIncluded)
	status.TagFilters = slices.Clone(current.TagFilters)
	status.MatchRule = string(current.MatchRule)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
	return nil
}

func newsReportStatusMap(current opsisdk.NewsReport) map[string]any {
	body := map[string]any{
		"newsFrequency":                string(current.NewsFrequency),
		"contentTypes":                 newsReportContentTypesStatusFromSDK(current.ContentTypes),
		"id":                           newsReportStringValue(current.Id),
		"compartmentId":                newsReportStringValue(current.CompartmentId),
		"onsTopicId":                   newsReportStringValue(current.OnsTopicId),
		"locale":                       string(current.Locale),
		"description":                  newsReportStringValue(current.Description),
		"name":                         newsReportStringValue(current.Name),
		"freeformTags":                 maps.Clone(current.FreeformTags),
		"definedTags":                  newsReportStatusTagsFromSDK(current.DefinedTags),
		"systemTags":                   newsReportStatusTagsFromSDK(current.SystemTags),
		"sdkStatus":                    string(current.Status),
		"timeCreated":                  newsReportTimeString(current.TimeCreated),
		"timeUpdated":                  newsReportTimeString(current.TimeUpdated),
		"lifecycleState":               string(current.LifecycleState),
		"lifecycleDetails":             newsReportStringValue(current.LifecycleDetails),
		"dayOfWeek":                    string(current.DayOfWeek),
		"areChildCompartmentsIncluded": newsReportBoolValue(current.AreChildCompartmentsIncluded),
		"tagFilters":                   slices.Clone(current.TagFilters),
		"matchRule":                    string(current.MatchRule),
	}
	for key, value := range body {
		if !newsReportMeaningfulStatusValue(value) {
			delete(body, key)
		}
	}
	return body
}

func newsReportMeaningfulStatusValue(value any) bool {
	switch current := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(current) != ""
	case bool:
		return current
	case []string:
		return len(current) != 0
	case map[string]string:
		return len(current) != 0
	case map[string]shared.MapValue:
		return len(current) != 0
	case opsiv1beta1.NewsReportContentTypes:
		return newsReportContentTypesMeaningful(current)
	default:
		return true
	}
}

func newsReportContentTypesMeaningful(current opsiv1beta1.NewsReportContentTypes) bool {
	return len(current.CapacityPlanningResources) != 0 ||
		len(current.SqlInsightsFleetAnalysisResources) != 0 ||
		len(current.SqlInsightsPlanChangesResources) != 0 ||
		len(current.SqlInsightsTopDatabasesResources) != 0 ||
		len(current.SqlInsightsTopSqlByInsightsResources) != 0 ||
		len(current.SqlInsightsTopSqlResources) != 0 ||
		len(current.SqlInsightsPerformanceDegradationResources) != 0 ||
		len(current.ActionableInsightsResources) != 0
}

func newsReportFromCurrentState(
	resource *opsiv1beta1.NewsReport,
	currentResponse any,
) (opsisdk.NewsReport, bool, error) {
	if current, ok, err := newsReportFromResponse(currentResponse); err != nil || ok {
		return current, ok, err
	}
	if resource == nil {
		return opsisdk.NewsReport{}, false, nil
	}
	return newsReportFromStatus(resource.Status), true, nil
}

func newsReportFromResponse(response any) (opsisdk.NewsReport, bool, error) {
	response = newsReportDereference(response)
	switch current := response.(type) {
	case nil:
		return opsisdk.NewsReport{}, false, nil
	case opsisdk.NewsReport:
		return current, true, nil
	case opsisdk.NewsReportSummary:
		return newsReportFromSummary(current), true, nil
	case newsReportRuntimeReadResponse:
		return newsReportFromStatusMap(current.Body)
	case newsReportRuntimeListResponse:
		return opsisdk.NewsReport{}, false, nil
	case map[string]any:
		return newsReportFromStatusMap(current)
	case opsisdk.CreateNewsReportResponse:
		return current.NewsReport, true, nil
	case opsisdk.GetNewsReportResponse:
		return current.NewsReport, true, nil
	case opsisdk.UpdateNewsReportResponse, opsisdk.DeleteNewsReportResponse, opsisdk.GetWorkRequestResponse, opsisdk.WorkRequest:
		return opsisdk.NewsReport{}, false, nil
	default:
		return opsisdk.NewsReport{}, false, fmt.Errorf("unexpected %s response type %T", newsReportKind, response)
	}
}

func newsReportFromStatusMap(values map[string]any) (opsisdk.NewsReport, bool, error) {
	if len(values) == 0 {
		return opsisdk.NewsReport{}, false, nil
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return opsisdk.NewsReport{}, false, fmt.Errorf("marshal %s status map: %w", newsReportKind, err)
	}
	var status opsiv1beta1.NewsReportStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return opsisdk.NewsReport{}, false, fmt.Errorf("unmarshal %s status map: %w", newsReportKind, err)
	}
	return newsReportFromStatus(status), true, nil
}

func newsReportDereference(response any) any {
	value := reflect.ValueOf(response)
	if !value.IsValid() || value.Kind() != reflect.Pointer {
		return response
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}

func newsReportFromSummary(summary opsisdk.NewsReportSummary) opsisdk.NewsReport {
	return opsisdk.NewsReport{
		NewsFrequency:                summary.NewsFrequency,
		ContentTypes:                 summary.ContentTypes,
		Id:                           summary.Id,
		CompartmentId:                summary.CompartmentId,
		OnsTopicId:                   summary.OnsTopicId,
		Locale:                       summary.Locale,
		Description:                  summary.Description,
		Name:                         summary.Name,
		FreeformTags:                 summary.FreeformTags,
		DefinedTags:                  summary.DefinedTags,
		SystemTags:                   summary.SystemTags,
		Status:                       summary.Status,
		TimeCreated:                  summary.TimeCreated,
		TimeUpdated:                  summary.TimeUpdated,
		LifecycleState:               summary.LifecycleState,
		LifecycleDetails:             summary.LifecycleDetails,
		DayOfWeek:                    summary.DayOfWeek,
		AreChildCompartmentsIncluded: summary.AreChildCompartmentsIncluded,
		TagFilters:                   summary.TagFilters,
		MatchRule:                    summary.MatchRule,
	}
}

func newsReportFromStatus(status opsiv1beta1.NewsReportStatus) opsisdk.NewsReport {
	return opsisdk.NewsReport{
		NewsFrequency:                opsisdk.NewsFrequencyEnum(status.NewsFrequency),
		ContentTypes:                 newsReportContentTypesFromStatus(status.ContentTypes),
		Id:                           common.String(status.Id),
		CompartmentId:                common.String(status.CompartmentId),
		OnsTopicId:                   common.String(status.OnsTopicId),
		Locale:                       opsisdk.NewsLocaleEnum(status.Locale),
		Description:                  common.String(status.Description),
		Name:                         common.String(status.Name),
		FreeformTags:                 maps.Clone(status.FreeformTags),
		DefinedTags:                  newsReportDefinedTagsFromStatus(status.DefinedTags),
		SystemTags:                   newsReportDefinedTagsFromStatus(status.SystemTags),
		Status:                       opsisdk.ResourceStatusEnum(status.Status),
		LifecycleState:               opsisdk.LifecycleStateEnum(status.LifecycleState),
		LifecycleDetails:             common.String(status.LifecycleDetails),
		DayOfWeek:                    opsisdk.DayOfWeekEnum(status.DayOfWeek),
		AreChildCompartmentsIncluded: common.Bool(status.AreChildCompartmentsIncluded),
		TagFilters:                   slices.Clone(status.TagFilters),
		MatchRule:                    opsisdk.MatchRuleEnum(status.MatchRule),
	}
}

func newsReportSummaryMatchesSpec(summary opsisdk.NewsReportSummary, spec opsiv1beta1.NewsReportSpec) bool {
	checks := []struct {
		desired string
		current string
	}{
		{desired: spec.CompartmentId, current: newsReportStringValue(summary.CompartmentId)},
		{desired: spec.Name, current: newsReportStringValue(summary.Name)},
		{desired: spec.OnsTopicId, current: newsReportStringValue(summary.OnsTopicId)},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.desired) != "" && strings.TrimSpace(check.desired) != strings.TrimSpace(check.current) {
			return false
		}
	}
	return true
}

func newsReportLifecyclePending(current opsisdk.NewsReport) bool {
	state := strings.ToUpper(string(current.LifecycleState))
	return state == string(opsisdk.LifecycleStateCreating) ||
		state == string(opsisdk.LifecycleStateUpdating) ||
		state == string(opsisdk.LifecycleStateDeleting)
}

func newsReportLifecycleDeleting(current opsisdk.NewsReport) bool {
	return strings.ToUpper(string(current.LifecycleState)) == string(opsisdk.LifecycleStateDeleting)
}

func newsReportLifecycleDeleted(current opsisdk.NewsReport) bool {
	return strings.ToUpper(string(current.LifecycleState)) == string(opsisdk.LifecycleStateDeleted) ||
		strings.ToUpper(string(current.Status)) == string(opsisdk.ResourceStatusTerminated)
}

func currentNewsReportID(resource *opsiv1beta1.NewsReport) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func handleNewsReportDeleteError(resource *opsiv1beta1.NewsReport, err error) error {
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
	return newsReportAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s delete path returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed", newsReportKind),
		opcRequestID: requestID,
	}
}

func newsReportIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func getNewsReportWorkRequest(
	ctx context.Context,
	client newsReportWorkRequestClient,
	initErr error,
	workRequestID string,
) (opsisdk.WorkRequest, error) {
	if initErr != nil {
		return opsisdk.WorkRequest{}, fmt.Errorf("initialize %s OCI client: %w", newsReportKind, initErr)
	}
	if client == nil {
		return opsisdk.WorkRequest{}, fmt.Errorf("%s work request client is not configured", newsReportKind)
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{WorkRequestId: common.String(workRequestID)})
	if err != nil {
		return opsisdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveNewsReportWorkRequestAction(workRequest any) (string, error) {
	current, err := newsReportWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func recoverNewsReportIDFromWorkRequest(
	_ *opsiv1beta1.NewsReport,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := newsReportWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	resourceID := newsReportIDFromWorkRequest(current, phase)
	if resourceID == "" {
		return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", newsReportKind, phase, newsReportStringValue(current.Id), newsReportKind)
	}
	return resourceID, nil
}

func newsReportIDFromWorkRequest(workRequest opsisdk.WorkRequest, phase shared.OSOKAsyncPhase) string {
	action := newsReportWorkRequestActionForPhase(phase)
	var candidate string
	for _, resource := range workRequest.Resources {
		if !isNewsReportWorkRequestResource(resource) {
			continue
		}
		if action != "" && resource.ActionType != action && resource.ActionType != opsisdk.ActionTypeInProgress {
			continue
		}
		id := strings.TrimSpace(newsReportStringValue(resource.Identifier))
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

func newsReportWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
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

func isNewsReportWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(newsReportStringValue(resource.EntityType)))
	normalizedEntityType := strings.NewReplacer("_", "", "-", "", " ", "").Replace(entityType)
	if strings.Contains(normalizedEntityType, "newsreport") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(newsReportStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/newsreports/")
}

func newsReportAsyncOperation(
	status *shared.OSOKStatus,
	workRequest opsisdk.WorkRequest,
	fallback shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	return servicemanager.BuildWorkRequestAsyncOperation(status, newsReportWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        string(workRequest.OperationType),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    newsReportStringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		Message:          newsReportWorkRequestMessage(fallback, workRequest),
		FallbackPhase:    fallback,
	})
}

func newsReportWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := newsReportWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", newsReportKind, phase, newsReportStringValue(current.Id), current.Status)
}

func newsReportWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	workRequest = newsReportDereference(workRequest)
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case opsisdk.GetWorkRequestResponse:
		return current.WorkRequest, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", newsReportKind, workRequest)
	}
}

func newsReportContentTypesFromSpec(spec opsiv1beta1.NewsReportContentTypes) *opsisdk.NewsContentTypes {
	return &opsisdk.NewsContentTypes{
		CapacityPlanningResources:                  newsReportCapacityPlanningResources(spec.CapacityPlanningResources),
		SqlInsightsFleetAnalysisResources:          newsReportSQLInsightResources(spec.SqlInsightsFleetAnalysisResources),
		SqlInsightsPlanChangesResources:            newsReportSQLInsightResources(spec.SqlInsightsPlanChangesResources),
		SqlInsightsTopDatabasesResources:           newsReportSQLInsightResources(spec.SqlInsightsTopDatabasesResources),
		SqlInsightsTopSqlByInsightsResources:       newsReportSQLInsightResources(spec.SqlInsightsTopSqlByInsightsResources),
		SqlInsightsTopSqlResources:                 newsReportSQLInsightResources(spec.SqlInsightsTopSqlResources),
		SqlInsightsPerformanceDegradationResources: newsReportSQLInsightResources(spec.SqlInsightsPerformanceDegradationResources),
		ActionableInsightsResources:                newsReportActionableInsightResources(spec.ActionableInsightsResources),
	}
}

func newsReportContentTypesFromStatus(status opsiv1beta1.NewsReportContentTypes) *opsisdk.NewsContentTypes {
	return newsReportContentTypesFromSpec(status)
}

func newsReportCapacityPlanningResources(values []string) []opsisdk.NewsContentTypesResourceEnum {
	if values == nil {
		return nil
	}
	converted := make([]opsisdk.NewsContentTypesResourceEnum, 0, len(values))
	for _, value := range values {
		converted = append(converted, opsisdk.NewsContentTypesResourceEnum(strings.TrimSpace(value)))
	}
	return converted
}

func newsReportSQLInsightResources(values []string) []opsisdk.NewsSqlInsightsContentTypesResourceEnum {
	if values == nil {
		return nil
	}
	converted := make([]opsisdk.NewsSqlInsightsContentTypesResourceEnum, 0, len(values))
	for _, value := range values {
		converted = append(converted, opsisdk.NewsSqlInsightsContentTypesResourceEnum(strings.TrimSpace(value)))
	}
	return converted
}

func newsReportActionableInsightResources(values []string) []opsisdk.ActionableInsightsContentTypesResourceEnum {
	if values == nil {
		return nil
	}
	converted := make([]opsisdk.ActionableInsightsContentTypesResourceEnum, 0, len(values))
	for _, value := range values {
		converted = append(converted, opsisdk.ActionableInsightsContentTypesResourceEnum(strings.TrimSpace(value)))
	}
	return converted
}

func newsReportContentTypesStatusFromSDK(current *opsisdk.NewsContentTypes) opsiv1beta1.NewsReportContentTypes {
	if current == nil {
		return opsiv1beta1.NewsReportContentTypes{}
	}
	return opsiv1beta1.NewsReportContentTypes{
		CapacityPlanningResources:                  newsReportCapacityPlanningResourceStrings(current.CapacityPlanningResources),
		SqlInsightsFleetAnalysisResources:          newsReportSQLInsightResourceStrings(current.SqlInsightsFleetAnalysisResources),
		SqlInsightsPlanChangesResources:            newsReportSQLInsightResourceStrings(current.SqlInsightsPlanChangesResources),
		SqlInsightsTopDatabasesResources:           newsReportSQLInsightResourceStrings(current.SqlInsightsTopDatabasesResources),
		SqlInsightsTopSqlByInsightsResources:       newsReportSQLInsightResourceStrings(current.SqlInsightsTopSqlByInsightsResources),
		SqlInsightsTopSqlResources:                 newsReportSQLInsightResourceStrings(current.SqlInsightsTopSqlResources),
		SqlInsightsPerformanceDegradationResources: newsReportSQLInsightResourceStrings(current.SqlInsightsPerformanceDegradationResources),
		ActionableInsightsResources:                newsReportActionableInsightResourceStrings(current.ActionableInsightsResources),
	}
}

func newsReportCapacityPlanningResourceStrings(values []opsisdk.NewsContentTypesResourceEnum) []string {
	if values == nil {
		return nil
	}
	converted := make([]string, 0, len(values))
	for _, value := range values {
		converted = append(converted, string(value))
	}
	return converted
}

func newsReportSQLInsightResourceStrings(values []opsisdk.NewsSqlInsightsContentTypesResourceEnum) []string {
	if values == nil {
		return nil
	}
	converted := make([]string, 0, len(values))
	for _, value := range values {
		converted = append(converted, string(value))
	}
	return converted
}

func newsReportActionableInsightResourceStrings(values []opsisdk.ActionableInsightsContentTypesResourceEnum) []string {
	if values == nil {
		return nil
	}
	converted := make([]string, 0, len(values))
	for _, value := range values {
		converted = append(converted, string(value))
	}
	return converted
}

func newsReportContentTypesEqual(left *opsisdk.NewsContentTypes, right *opsisdk.NewsContentTypes) bool {
	return reflect.DeepEqual(newsReportContentTypesStatusFromSDK(left), newsReportContentTypesStatusFromSDK(right))
}

func newsReportDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func newsReportDefinedTagsFromStatus(status map[string]shared.MapValue) map[string]map[string]interface{} {
	return newsReportDefinedTagsFromSpec(status)
}

func newsReportStatusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
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

func newsReportDefinedTagsEqual(left map[string]map[string]interface{}, right map[string]map[string]interface{}) bool {
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

func newsReportStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func newsReportBoolValue(value *bool) bool {
	return value != nil && *value
}

func newsReportTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
