/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pingmonitor

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
)

type pingMonitorOCIClient interface {
	CreatePingMonitor(context.Context, healthcheckssdk.CreatePingMonitorRequest) (healthcheckssdk.CreatePingMonitorResponse, error)
	GetPingMonitor(context.Context, healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error)
	ListPingMonitors(context.Context, healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error)
	UpdatePingMonitor(context.Context, healthcheckssdk.UpdatePingMonitorRequest) (healthcheckssdk.UpdatePingMonitorResponse, error)
	DeletePingMonitor(context.Context, healthcheckssdk.DeletePingMonitorRequest) (healthcheckssdk.DeletePingMonitorResponse, error)
}

func init() {
	registerPingMonitorRuntimeHooksMutator(func(_ *PingMonitorServiceManager, hooks *PingMonitorRuntimeHooks) {
		applyPingMonitorRuntimeHooks(hooks)
	})
}

func applyPingMonitorRuntimeHooks(hooks *PingMonitorRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newPingMonitorRuntimeSemantics()
	hooks.DeleteHooks.HandleError = handlePingMonitorDeleteError
	if hooks.List.Call != nil {
		hooks.List.Call = listPingMonitorsAllPages(hooks.List.Call)
	}
}

func newPingMonitorRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "healthchecks",
		FormalSlug:    "pingmonitor",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{},
			UpdatingStates:     []string{},
			ActiveStates:       []string{},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "protocol"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"targets",
				"vantagePointNames",
				"port",
				"timeoutInSeconds",
				"protocol",
				"displayName",
				"intervalInSeconds",
				"isEnabled",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "PingMonitor", Action: "CreatePingMonitor"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "PingMonitor", Action: "UpdatePingMonitor"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "PingMonitor", Action: "DeletePingMonitor"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "PingMonitor", Action: "GetPingMonitor"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "PingMonitor", Action: "GetPingMonitor"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "PingMonitor", Action: "GetPingMonitor"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func handlePingMonitorDeleteError(resource *healthchecksv1beta1.PingMonitor, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return pingMonitorAmbiguousNotFoundError{
		message:      "PingMonitor delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

type pingMonitorAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e pingMonitorAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e pingMonitorAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func newPingMonitorServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client pingMonitorOCIClient,
) PingMonitorServiceClient {
	manager := &PingMonitorServiceManager{Log: log}
	hooks := newPingMonitorRuntimeHooksWithOCIClient(client)
	applyPingMonitorRuntimeHooks(&hooks)
	delegate := defaultPingMonitorServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*healthchecksv1beta1.PingMonitor](
			buildPingMonitorGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapPingMonitorGeneratedClient(hooks, delegate)
}

func newPingMonitorRuntimeHooksWithOCIClient(client pingMonitorOCIClient) PingMonitorRuntimeHooks {
	return PingMonitorRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*healthchecksv1beta1.PingMonitor]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*healthchecksv1beta1.PingMonitor]{},
		StatusHooks:     generatedruntime.StatusHooks[*healthchecksv1beta1.PingMonitor]{},
		ParityHooks:     generatedruntime.ParityHooks[*healthchecksv1beta1.PingMonitor]{},
		Async:           generatedruntime.AsyncHooks[*healthchecksv1beta1.PingMonitor]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*healthchecksv1beta1.PingMonitor]{},
		Create: runtimeOperationHooks[healthcheckssdk.CreatePingMonitorRequest, healthcheckssdk.CreatePingMonitorResponse]{
			Fields: pingMonitorCreateFields(),
			Call: func(ctx context.Context, request healthcheckssdk.CreatePingMonitorRequest) (healthcheckssdk.CreatePingMonitorResponse, error) {
				if client == nil {
					return healthcheckssdk.CreatePingMonitorResponse{}, fmt.Errorf("PingMonitor OCI client is nil")
				}
				return client.CreatePingMonitor(ctx, request)
			},
		},
		Get: runtimeOperationHooks[healthcheckssdk.GetPingMonitorRequest, healthcheckssdk.GetPingMonitorResponse]{
			Fields: pingMonitorGetFields(),
			Call: func(ctx context.Context, request healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
				if client == nil {
					return healthcheckssdk.GetPingMonitorResponse{}, fmt.Errorf("PingMonitor OCI client is nil")
				}
				return client.GetPingMonitor(ctx, request)
			},
		},
		List: runtimeOperationHooks[healthcheckssdk.ListPingMonitorsRequest, healthcheckssdk.ListPingMonitorsResponse]{
			Fields: pingMonitorListFields(),
			Call: func(ctx context.Context, request healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
				if client == nil {
					return healthcheckssdk.ListPingMonitorsResponse{}, fmt.Errorf("PingMonitor OCI client is nil")
				}
				return client.ListPingMonitors(ctx, request)
			},
		},
		Update: runtimeOperationHooks[healthcheckssdk.UpdatePingMonitorRequest, healthcheckssdk.UpdatePingMonitorResponse]{
			Fields: pingMonitorUpdateFields(),
			Call: func(ctx context.Context, request healthcheckssdk.UpdatePingMonitorRequest) (healthcheckssdk.UpdatePingMonitorResponse, error) {
				if client == nil {
					return healthcheckssdk.UpdatePingMonitorResponse{}, fmt.Errorf("PingMonitor OCI client is nil")
				}
				return client.UpdatePingMonitor(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[healthcheckssdk.DeletePingMonitorRequest, healthcheckssdk.DeletePingMonitorResponse]{
			Fields: pingMonitorDeleteFields(),
			Call: func(ctx context.Context, request healthcheckssdk.DeletePingMonitorRequest) (healthcheckssdk.DeletePingMonitorResponse, error) {
				if client == nil {
					return healthcheckssdk.DeletePingMonitorResponse{}, fmt.Errorf("PingMonitor OCI client is nil")
				}
				return client.DeletePingMonitor(ctx, request)
			},
		},
		WrapGeneratedClient: []func(PingMonitorServiceClient) PingMonitorServiceClient{},
	}
}

func pingMonitorCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreatePingMonitorDetails", RequestName: "CreatePingMonitorDetails", Contribution: "body"},
	}
}

func pingMonitorGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitorId", RequestName: "monitorId", Contribution: "path", PreferResourceID: true},
	}
}

func pingMonitorListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "HomeRegion", RequestName: "homeRegion", Contribution: "query"},
	}
}

func pingMonitorUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitorId", RequestName: "monitorId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdatePingMonitorDetails", RequestName: "UpdatePingMonitorDetails", Contribution: "body"},
	}
}

func pingMonitorDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitorId", RequestName: "monitorId", Contribution: "path", PreferResourceID: true},
	}
}

func listPingMonitorsAllPages(
	call func(context.Context, healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error),
) func(context.Context, healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
	return func(ctx context.Context, request healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
		var combined healthcheckssdk.ListPingMonitorsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return healthcheckssdk.ListPingMonitorsResponse{}, err
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
}
