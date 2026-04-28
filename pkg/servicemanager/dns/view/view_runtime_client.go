/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package view

import (
	"context"
	"fmt"
	"strings"

	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type viewOCIClient interface {
	CreateView(context.Context, dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error)
	GetView(context.Context, dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error)
	ListViews(context.Context, dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error)
	UpdateView(context.Context, dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error)
	DeleteView(context.Context, dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error)
}

func init() {
	registerViewRuntimeHooksMutator(func(_ *ViewServiceManager, hooks *ViewRuntimeHooks) {
		applyViewRuntimeHooks(hooks)
	})
}

func applyViewRuntimeHooks(hooks *ViewRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = viewRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardViewExistingBeforeCreate
	hooks.List.Fields = viewListFields()
	hooks.DeleteHooks.HandleError = handleViewDeleteError
	forcePrivateViewScope(hooks)
	wrapViewDeleteConfirmation(hooks)
}

func newViewServiceClientWithOCIClient(log loggerutil.OSOKLogger, client viewOCIClient) ViewServiceClient {
	hooks := newViewRuntimeHooksWithOCIClient(client)
	applyViewRuntimeHooks(&hooks)
	delegate := defaultViewServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dnsv1beta1.View](
			buildViewGeneratedRuntimeConfig(&ViewServiceManager{Log: log}, hooks),
		),
	}
	return wrapViewGeneratedClient(hooks, delegate)
}

func newViewRuntimeHooksWithOCIClient(client viewOCIClient) ViewRuntimeHooks {
	return ViewRuntimeHooks{
		Create: runtimeOperationHooks[dnssdk.CreateViewRequest, dnssdk.CreateViewResponse]{
			Fields: viewCreateFields(),
			Call: func(ctx context.Context, request dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
				return client.CreateView(ctx, request)
			},
		},
		Get: runtimeOperationHooks[dnssdk.GetViewRequest, dnssdk.GetViewResponse]{
			Fields: viewGetFields(),
			Call: func(ctx context.Context, request dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
				return client.GetView(ctx, request)
			},
		},
		List: runtimeOperationHooks[dnssdk.ListViewsRequest, dnssdk.ListViewsResponse]{
			Fields: viewListFields(),
			Call: func(ctx context.Context, request dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error) {
				return client.ListViews(ctx, request)
			},
		},
		Update: runtimeOperationHooks[dnssdk.UpdateViewRequest, dnssdk.UpdateViewResponse]{
			Fields: viewUpdateFields(),
			Call: func(ctx context.Context, request dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error) {
				return client.UpdateView(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[dnssdk.DeleteViewRequest, dnssdk.DeleteViewResponse]{
			Fields: viewDeleteFields(),
			Call: func(ctx context.Context, request dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
				return client.DeleteView(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ViewServiceClient) ViewServiceClient{},
	}
}

func viewRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dns",
		FormalSlug:    "view",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			UpdatingStates: []string{"UPDATING"},
			ActiveStates:   []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "displayName", "freeformTags"},
			Mutable:         []string{"definedTags", "displayName", "freeformTags"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func viewCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateViewDetails", RequestName: "CreateViewDetails", Contribution: "body"},
	}
}

func viewGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "path", PreferResourceID: true},
	}
}

func viewListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func viewUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateViewDetails", RequestName: "UpdateViewDetails", Contribution: "body"},
	}
}

func viewDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ViewId", RequestName: "viewId", Contribution: "path", PreferResourceID: true},
	}
}

func guardViewExistingBeforeCreate(_ context.Context, resource *dnsv1beta1.View) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("view resource is nil")
	}
	if resource.Spec.CompartmentId == "" || resource.Spec.DisplayName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func forcePrivateViewScope(hooks *ViewRuntimeHooks) {
	if hooks.Create.Call != nil {
		call := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request dnssdk.CreateViewRequest) (dnssdk.CreateViewResponse, error) {
			request.Scope = dnssdk.CreateViewScopePrivate
			return call(ctx, request)
		}
	}
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error) {
			request.Scope = dnssdk.GetViewScopePrivate
			return call(ctx, request)
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error) {
			return listPrivateViewPages(ctx, call, request)
		}
	}
	if hooks.Update.Call != nil {
		call := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request dnssdk.UpdateViewRequest) (dnssdk.UpdateViewResponse, error) {
			request.Scope = dnssdk.UpdateViewScopePrivate
			return call(ctx, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request dnssdk.DeleteViewRequest) (dnssdk.DeleteViewResponse, error) {
			request.Scope = dnssdk.DeleteViewScopePrivate
			return call(ctx, request)
		}
	}
}

func listPrivateViewPages(
	ctx context.Context,
	call func(context.Context, dnssdk.ListViewsRequest) (dnssdk.ListViewsResponse, error),
	request dnssdk.ListViewsRequest,
) (dnssdk.ListViewsResponse, error) {
	request.Scope = dnssdk.ListViewsScopePrivate
	var combined dnssdk.ListViewsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || *response.OpcNextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func handleViewDeleteError(resource *dnsv1beta1.View, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("view delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapViewDeleteConfirmation(hooks *ViewRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getView := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ViewServiceClient) ViewServiceClient {
		return viewDeleteConfirmationClient{
			delegate: delegate,
			getView:  getView,
		}
	})
}

type viewDeleteConfirmationClient struct {
	delegate ViewServiceClient
	getView  func(context.Context, dnssdk.GetViewRequest) (dnssdk.GetViewResponse, error)
}

func (c viewDeleteConfirmationClient) CreateOrUpdate(ctx context.Context, resource *dnsv1beta1.View, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c viewDeleteConfirmationClient) Delete(ctx context.Context, resource *dnsv1beta1.View) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c viewDeleteConfirmationClient) rejectAuthShapedConfirmRead(ctx context.Context, resource *dnsv1beta1.View) error {
	if c.getView == nil || resource == nil {
		return nil
	}
	viewID := trackedViewID(resource)
	if viewID == "" {
		return nil
	}
	_, err := c.getView(ctx, dnssdk.GetViewRequest{ViewId: &viewID})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("view delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedViewID(resource *dnsv1beta1.View) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}
