/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package httpmonitor

import (
	"context"
	"fmt"
	"strings"

	healthcheckssdk "github.com/oracle/oci-go-sdk/v65/healthchecks"
	healthchecksv1beta1 "github.com/oracle/oci-service-operator/api/healthchecks/v1beta1"
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

type httpMonitorOCIClient interface {
	CreateHttpMonitor(context.Context, healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error)
	GetHttpMonitor(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error)
	ListHttpMonitors(context.Context, healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error)
	UpdateHttpMonitor(context.Context, healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error)
	DeleteHttpMonitor(context.Context, healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error)
}

func init() {
	registerHttpMonitorRuntimeHooksMutator(func(_ *HttpMonitorServiceManager, hooks *HttpMonitorRuntimeHooks) {
		applyHttpMonitorRuntimeHooks(hooks)
	})
}

func applyHttpMonitorRuntimeHooks(hooks *HttpMonitorRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newHttpMonitorRuntimeSemantics()
	hooks.List.Fields = httpMonitorListFields()
	hooks.List.Call = paginatedHttpMonitorListCall(hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleHttpMonitorDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyHttpMonitorDeleteOutcome
	if hooks.Get.Call != nil {
		get := hooks.Get.Call
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate HttpMonitorServiceClient) HttpMonitorServiceClient {
			return httpMonitorDeleteGuardClient{delegate: delegate, get: get}
		})
	}
}

func newHttpMonitorRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:       "healthchecks",
		FormalSlug:          "httpmonitor",
		StatusProjection:    "required",
		SecretSideEffects:   "none",
		FinalizerPolicy:     "retain-until-confirmed-delete",
		Lifecycle:           generatedruntime.LifecycleSemantics{},
		Delete:              generatedruntime.DeleteSemantics{Policy: "best-effort"},
		List:                &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"compartmentId", "displayName", "protocol", "id"}},
		Mutation:            httpMonitorMutationSemantics(),
		Hooks:               httpMonitorHookSet(),
		CreateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write", Hooks: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}}},
		UpdateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write", Hooks: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}}},
		DeleteFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "confirm-delete", Hooks: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}}},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func httpMonitorMutationSemantics() generatedruntime.MutationSemantics {
	return generatedruntime.MutationSemantics{
		Mutable: []string{
			"targets",
			"vantagePointNames",
			"port",
			"timeoutInSeconds",
			"protocol",
			"method",
			"path",
			"headers",
			"displayName",
			"intervalInSeconds",
			"isEnabled",
			"freeformTags",
			"definedTags",
		},
		ForceNew:      []string{"compartmentId"},
		ConflictsWith: map[string][]string{},
	}
}

func httpMonitorHookSet() generatedruntime.HookSet {
	return generatedruntime.HookSet{
		Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
	}
}

func httpMonitorCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateHttpMonitorDetails", RequestName: "CreateHttpMonitorDetails", Contribution: "body"},
	}
}

func httpMonitorGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitorId", RequestName: "monitorId", Contribution: "path", PreferResourceID: true},
	}
}

func httpMonitorListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "HomeRegion", RequestName: "homeRegion", Contribution: "query", LookupPaths: []string{"status.homeRegion", "homeRegion"}},
	}
}

func httpMonitorUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitorId", RequestName: "monitorId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateHttpMonitorDetails", RequestName: "UpdateHttpMonitorDetails", Contribution: "body"},
	}
}

func httpMonitorDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitorId", RequestName: "monitorId", Contribution: "path", PreferResourceID: true},
	}
}

func paginatedHttpMonitorListCall(
	call func(context.Context, healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error),
) func(context.Context, healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error) {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error) {
		var combined healthcheckssdk.ListHttpMonitorsResponse
		nextPage := request.Page
		for {
			response, err := call(ctx, httpMonitorListPageRequest(request, nextPage))
			if err != nil {
				return response, err
			}
			mergeHttpMonitorListPage(&combined, response)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func httpMonitorListPageRequest(request healthcheckssdk.ListHttpMonitorsRequest, nextPage *string) healthcheckssdk.ListHttpMonitorsRequest {
	pageRequest := request
	pageRequest.Page = nextPage
	return pageRequest
}

func mergeHttpMonitorListPage(combined *healthcheckssdk.ListHttpMonitorsResponse, response healthcheckssdk.ListHttpMonitorsResponse) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func handleHttpMonitorDeleteError(resource *healthchecksv1beta1.HttpMonitor, err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}

	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("HttpMonitor delete returned ambiguous %s %s; retaining finalizer",
		classification.HTTPStatusCodeString(), classification.ErrorCodeString())
}

func applyHttpMonitorDeleteOutcome(resource *healthchecksv1beta1.HttpMonitor, response any, stage generatedruntime.DeleteConfirmStage) (generatedruntime.DeleteOutcome, error) {
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if httpMonitorResponseID(response) == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}

	markHttpMonitorTerminating(resource, "OCI resource delete is in progress")
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func httpMonitorResponseID(response any) string {
	switch typed := response.(type) {
	case healthcheckssdk.GetHttpMonitorResponse:
		return httpMonitorStringValue(typed.Id)
	case *healthcheckssdk.GetHttpMonitorResponse:
		if typed == nil {
			return ""
		}
		return httpMonitorStringValue(typed.Id)
	case healthcheckssdk.HttpMonitor:
		return httpMonitorStringValue(typed.Id)
	case *healthcheckssdk.HttpMonitor:
		if typed == nil {
			return ""
		}
		return httpMonitorStringValue(typed.Id)
	}
	return ""
}

func httpMonitorStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func markHttpMonitorTerminating(resource *healthchecksv1beta1.HttpMonitor, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

type httpMonitorDeleteGuardClient struct {
	delegate HttpMonitorServiceClient
	get      func(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error)
}

func (c httpMonitorDeleteGuardClient) CreateOrUpdate(ctx context.Context, resource *healthchecksv1beta1.HttpMonitor, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c httpMonitorDeleteGuardClient) Delete(ctx context.Context, resource *healthchecksv1beta1.HttpMonitor) (bool, error) {
	currentID := httpMonitorTrackedID(resource)
	if currentID == "" || c.get == nil {
		return c.delegate.Delete(ctx, resource)
	}

	_, err := c.get(ctx, healthcheckssdk.GetHttpMonitorRequest{MonitorId: &currentID})
	if err != nil && errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return false, handleHttpMonitorDeleteError(resource, err)
	}
	return c.delegate.Delete(ctx, resource)
}

func httpMonitorTrackedID(resource *healthchecksv1beta1.HttpMonitor) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func newHttpMonitorServiceClientWithOCIClient(log loggerutil.OSOKLogger, client httpMonitorOCIClient) HttpMonitorServiceClient {
	hooks := newHttpMonitorRuntimeHooksWithOCIClient(client)
	applyHttpMonitorRuntimeHooks(&hooks)
	delegate := defaultHttpMonitorServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*healthchecksv1beta1.HttpMonitor](
			buildHttpMonitorGeneratedRuntimeConfig(&HttpMonitorServiceManager{Log: log}, hooks),
		),
	}
	return wrapHttpMonitorGeneratedClient(hooks, delegate)
}

func newHttpMonitorRuntimeHooksWithOCIClient(client httpMonitorOCIClient) HttpMonitorRuntimeHooks {
	return HttpMonitorRuntimeHooks{
		Create: runtimeOperationHooks[healthcheckssdk.CreateHttpMonitorRequest, healthcheckssdk.CreateHttpMonitorResponse]{
			Fields: httpMonitorCreateFields(),
			Call: func(ctx context.Context, request healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error) {
				return client.CreateHttpMonitor(ctx, request)
			},
		},
		Get: runtimeOperationHooks[healthcheckssdk.GetHttpMonitorRequest, healthcheckssdk.GetHttpMonitorResponse]{
			Fields: httpMonitorGetFields(),
			Call: func(ctx context.Context, request healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
				return client.GetHttpMonitor(ctx, request)
			},
		},
		List: runtimeOperationHooks[healthcheckssdk.ListHttpMonitorsRequest, healthcheckssdk.ListHttpMonitorsResponse]{
			Fields: httpMonitorListFields(),
			Call: func(ctx context.Context, request healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error) {
				return client.ListHttpMonitors(ctx, request)
			},
		},
		Update: runtimeOperationHooks[healthcheckssdk.UpdateHttpMonitorRequest, healthcheckssdk.UpdateHttpMonitorResponse]{
			Fields: httpMonitorUpdateFields(),
			Call: func(ctx context.Context, request healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error) {
				return client.UpdateHttpMonitor(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[healthcheckssdk.DeleteHttpMonitorRequest, healthcheckssdk.DeleteHttpMonitorResponse]{
			Fields: httpMonitorDeleteFields(),
			Call: func(ctx context.Context, request healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error) {
				return client.DeleteHttpMonitor(ctx, request)
			},
		},
		WrapGeneratedClient: []func(HttpMonitorServiceClient) HttpMonitorServiceClient{},
	}
}
