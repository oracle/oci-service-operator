/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementdashboard

import (
	"context"
	"fmt"
	"strings"

	managementdashboardsdk "github.com/oracle/oci-go-sdk/v65/managementdashboard"
	managementdashboardv1beta1 "github.com/oracle/oci-service-operator/api/managementdashboard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type managementDashboardOCIClient interface {
	CreateManagementDashboard(context.Context, managementdashboardsdk.CreateManagementDashboardRequest) (managementdashboardsdk.CreateManagementDashboardResponse, error)
	GetManagementDashboard(context.Context, managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error)
	ListManagementDashboards(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error)
	UpdateManagementDashboard(context.Context, managementdashboardsdk.UpdateManagementDashboardRequest) (managementdashboardsdk.UpdateManagementDashboardResponse, error)
	DeleteManagementDashboard(context.Context, managementdashboardsdk.DeleteManagementDashboardRequest) (managementdashboardsdk.DeleteManagementDashboardResponse, error)
}

type managementDashboardListCall func(context.Context, managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error)

type managementDashboardAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e managementDashboardAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e managementDashboardAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type managementDashboardAuthShapedConfirmRead struct {
	err error
}

func (e managementDashboardAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("managementdashboard delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e managementDashboardAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func init() {
	registerManagementDashboardRuntimeHooksMutator(func(manager *ManagementDashboardServiceManager, hooks *ManagementDashboardRuntimeHooks) {
		applyManagementDashboardRuntimeHooks(manager, hooks)
	})
}

func applyManagementDashboardRuntimeHooks(manager *ManagementDashboardServiceManager, hooks *ManagementDashboardRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newManagementDashboardRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *managementdashboardv1beta1.ManagementDashboard, namespace string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("managementdashboard resource is nil")
		}
		credentialClient := managerCredentialClient(manager)
		return generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credentialClient, namespace)
	}
	hooks.Create.Fields = managementDashboardCreateFields()
	hooks.Get.Fields = managementDashboardGetFields()
	hooks.List.Fields = managementDashboardListFields()
	hooks.Update.Fields = managementDashboardUpdateFields()
	hooks.Delete.Fields = managementDashboardDeleteFields()
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *managementdashboardv1beta1.ManagementDashboard, currentID string) (any, error) {
		return confirmManagementDashboardDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleManagementDashboardDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleManagementDashboardDeleteConfirmReadOutcome
	if hooks.List.Call != nil {
		hooks.List.Call = listManagementDashboardsAllPages(hooks.List.Call)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagementDashboardServiceClient) ManagementDashboardServiceClient {
		return managementDashboardDeleteFallbackClient{
			delegate: delegate,
			list:     hooks.List.Call,
		}
	})
}

type managementDashboardDeleteFallbackClient struct {
	delegate ManagementDashboardServiceClient
	list     managementDashboardListCall
}

func (c managementDashboardDeleteFallbackClient) CreateOrUpdate(
	ctx context.Context,
	resource *managementdashboardv1beta1.ManagementDashboard,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c managementDashboardDeleteFallbackClient) Delete(
	ctx context.Context,
	resource *managementdashboardv1beta1.ManagementDashboard,
) (bool, error) {
	if !managementDashboardDeleteFallbackEnabled(resource, c.list) {
		return c.delegate.Delete(ctx, resource)
	}

	summary, found, err := resolveManagementDashboardDeleteFallbackSummary(ctx, c.list, resource)
	if err != nil {
		return false, err
	}
	if !found {
		markManagementDashboardDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}

	currentID := managementDashboardSummaryID(summary)
	if currentID == "" {
		return false, fmt.Errorf("managementdashboard delete fallback could not resolve a resource OCID")
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
	resource.Status.Id = currentID
	return c.delegate.Delete(ctx, resource)
}

func managementDashboardDeleteFallbackEnabled(
	resource *managementdashboardv1beta1.ManagementDashboard,
	list managementDashboardListCall,
) bool {
	return resource != nil &&
		managementDashboardRecordedID(resource) == "" &&
		list != nil
}

func resolveManagementDashboardDeleteFallbackSummary(
	ctx context.Context,
	list managementDashboardListCall,
	resource *managementdashboardv1beta1.ManagementDashboard,
) (managementdashboardsdk.ManagementDashboardSummary, bool, error) {
	response, err := list(ctx, managementdashboardsdk.ListManagementDashboardsRequest{
		CompartmentId: managementDashboardStringPointer(resource.Spec.CompartmentId),
		DisplayName:   managementDashboardStringPointer(resource.Spec.DisplayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return managementdashboardsdk.ManagementDashboardSummary{}, false, managementDashboardAuthShapedConfirmRead{err: err}
		}
		return managementdashboardsdk.ManagementDashboardSummary{}, false, err
	}

	matches := make([]managementdashboardsdk.ManagementDashboardSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if managementDashboardSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return managementdashboardsdk.ManagementDashboardSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return managementdashboardsdk.ManagementDashboardSummary{}, false, fmt.Errorf("managementdashboard delete fallback found multiple matching resources for compartmentId %q and displayName %q", resource.Spec.CompartmentId, resource.Spec.DisplayName)
	}
}

func markManagementDashboardDeleted(resource *managementdashboardv1beta1.ManagementDashboard, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
}

func managerCredentialClient(manager *ManagementDashboardServiceManager) credhelper.CredentialClient {
	if manager == nil {
		return nil
	}
	return manager.CredentialClient
}

func newManagementDashboardRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "managementdashboard",
		FormalSlug:    "managementdashboard",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(managementdashboardsdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "best-effort",
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"providerId",
				"providerName",
				"providerVersion",
				"type",
				"isOobDashboard",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"providerId",
				"providerName",
				"providerVersion",
				"tiles",
				"displayName",
				"description",
				"isShowInHome",
				"metadataVersion",
				"isShowDescription",
				"screenImage",
				"nls",
				"uiConfig",
				"dataConfig",
				"type",
				"isFavorite",
				"parametersConfig",
				"featuresConfig",
				"drilldownConfig",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"dashboardId",
				"isOobDashboard",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ManagementDashboard", Action: "CreateManagementDashboard"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ManagementDashboard", Action: "UpdateManagementDashboard"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ManagementDashboard", Action: "DeleteManagementDashboard"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ManagementDashboard", Action: "GetManagementDashboard"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ManagementDashboard", Action: "GetManagementDashboard"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ManagementDashboard", Action: "GetManagementDashboard"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func newManagementDashboardServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client managementDashboardOCIClient,
) ManagementDashboardServiceClient {
	manager := &ManagementDashboardServiceManager{Log: log}
	hooks := newManagementDashboardRuntimeHooksWithOCIClient(client)
	applyManagementDashboardRuntimeHooks(manager, &hooks)
	delegate := defaultManagementDashboardServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*managementdashboardv1beta1.ManagementDashboard](
			buildManagementDashboardGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapManagementDashboardGeneratedClient(hooks, delegate)
}

func newManagementDashboardRuntimeHooksWithOCIClient(client managementDashboardOCIClient) ManagementDashboardRuntimeHooks {
	return ManagementDashboardRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*managementdashboardv1beta1.ManagementDashboard]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*managementdashboardv1beta1.ManagementDashboard]{},
		StatusHooks:     generatedruntime.StatusHooks[*managementdashboardv1beta1.ManagementDashboard]{},
		ParityHooks:     generatedruntime.ParityHooks[*managementdashboardv1beta1.ManagementDashboard]{},
		Async:           generatedruntime.AsyncHooks[*managementdashboardv1beta1.ManagementDashboard]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*managementdashboardv1beta1.ManagementDashboard]{},
		Create: runtimeOperationHooks[managementdashboardsdk.CreateManagementDashboardRequest, managementdashboardsdk.CreateManagementDashboardResponse]{
			Fields: managementDashboardCreateFields(),
			Call: func(ctx context.Context, request managementdashboardsdk.CreateManagementDashboardRequest) (managementdashboardsdk.CreateManagementDashboardResponse, error) {
				if client == nil {
					return managementdashboardsdk.CreateManagementDashboardResponse{}, fmt.Errorf("managementdashboard OCI client is nil")
				}
				return client.CreateManagementDashboard(ctx, request)
			},
		},
		Get: runtimeOperationHooks[managementdashboardsdk.GetManagementDashboardRequest, managementdashboardsdk.GetManagementDashboardResponse]{
			Fields: managementDashboardGetFields(),
			Call: func(ctx context.Context, request managementdashboardsdk.GetManagementDashboardRequest) (managementdashboardsdk.GetManagementDashboardResponse, error) {
				if client == nil {
					return managementdashboardsdk.GetManagementDashboardResponse{}, fmt.Errorf("managementdashboard OCI client is nil")
				}
				return client.GetManagementDashboard(ctx, request)
			},
		},
		List: runtimeOperationHooks[managementdashboardsdk.ListManagementDashboardsRequest, managementdashboardsdk.ListManagementDashboardsResponse]{
			Fields: managementDashboardListFields(),
			Call: func(ctx context.Context, request managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
				if client == nil {
					return managementdashboardsdk.ListManagementDashboardsResponse{}, fmt.Errorf("managementdashboard OCI client is nil")
				}
				return client.ListManagementDashboards(ctx, request)
			},
		},
		Update: runtimeOperationHooks[managementdashboardsdk.UpdateManagementDashboardRequest, managementdashboardsdk.UpdateManagementDashboardResponse]{
			Fields: managementDashboardUpdateFields(),
			Call: func(ctx context.Context, request managementdashboardsdk.UpdateManagementDashboardRequest) (managementdashboardsdk.UpdateManagementDashboardResponse, error) {
				if client == nil {
					return managementdashboardsdk.UpdateManagementDashboardResponse{}, fmt.Errorf("managementdashboard OCI client is nil")
				}
				return client.UpdateManagementDashboard(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[managementdashboardsdk.DeleteManagementDashboardRequest, managementdashboardsdk.DeleteManagementDashboardResponse]{
			Fields: managementDashboardDeleteFields(),
			Call: func(ctx context.Context, request managementdashboardsdk.DeleteManagementDashboardRequest) (managementdashboardsdk.DeleteManagementDashboardResponse, error) {
				if client == nil {
					return managementdashboardsdk.DeleteManagementDashboardResponse{}, fmt.Errorf("managementdashboard OCI client is nil")
				}
				return client.DeleteManagementDashboard(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ManagementDashboardServiceClient) ManagementDashboardServiceClient{},
	}
}

func managementDashboardCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateManagementDashboardDetails", RequestName: "CreateManagementDashboardDetails", Contribution: "body"},
	}
}

func managementDashboardGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagementDashboardId", RequestName: "managementDashboardId", Contribution: "path", PreferResourceID: true},
	}
}

func managementDashboardListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
	}
}

func managementDashboardUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagementDashboardId", RequestName: "managementDashboardId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateManagementDashboardDetails", RequestName: "UpdateManagementDashboardDetails", Contribution: "body"},
	}
}

func managementDashboardDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagementDashboardId", RequestName: "managementDashboardId", Contribution: "path", PreferResourceID: true},
	}
}

func listManagementDashboardsAllPages(call managementDashboardListCall) managementDashboardListCall {
	return func(ctx context.Context, request managementdashboardsdk.ListManagementDashboardsRequest) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
		if call == nil {
			return managementdashboardsdk.ListManagementDashboardsResponse{}, fmt.Errorf("managementdashboard list operation is not configured")
		}
		return collectManagementDashboardListPages(ctx, call, request)
	}
}

func collectManagementDashboardListPages(
	ctx context.Context,
	call managementDashboardListCall,
	request managementdashboardsdk.ListManagementDashboardsRequest,
) (managementdashboardsdk.ListManagementDashboardsResponse, error) {
	seenPages := map[string]struct{}{}
	var combined managementdashboardsdk.ListManagementDashboardsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return managementdashboardsdk.ListManagementDashboardsResponse{}, err
		}
		appendManagementDashboardListPage(&combined, response)

		nextPage := managementDashboardStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if err := rememberManagementDashboardNextPage(seenPages, nextPage); err != nil {
			return managementdashboardsdk.ListManagementDashboardsResponse{}, err
		}
		request.Page = managementDashboardStringPointer(nextPage)
	}
}

func appendManagementDashboardListPage(
	combined *managementdashboardsdk.ListManagementDashboardsResponse,
	response managementdashboardsdk.ListManagementDashboardsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func rememberManagementDashboardNextPage(seenPages map[string]struct{}, nextPage string) error {
	if _, ok := seenPages[nextPage]; ok {
		return fmt.Errorf("managementdashboard list pagination repeated page token %q", nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nil
}

func confirmManagementDashboardDeleteRead(
	ctx context.Context,
	hooks *ManagementDashboardRuntimeHooks,
	resource *managementdashboardv1beta1.ManagementDashboard,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm ManagementDashboard delete: runtime hooks are nil")
	}
	if currentID = strings.TrimSpace(currentID); currentID != "" {
		return confirmManagementDashboardDeleteReadByID(ctx, hooks, currentID)
	}
	return confirmManagementDashboardDeleteReadByIdentity(ctx, hooks, resource)
}

func confirmManagementDashboardDeleteReadByID(
	ctx context.Context,
	hooks *ManagementDashboardRuntimeHooks,
	currentID string,
) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm ManagementDashboard delete: get hook is not configured")
	}
	response, err := hooks.Get.Call(ctx, managementdashboardsdk.GetManagementDashboardRequest{
		ManagementDashboardId: managementDashboardStringPointer(currentID),
	})
	return managementDashboardDeleteConfirmReadResponse(response, err)
}

func confirmManagementDashboardDeleteReadByIdentity(
	ctx context.Context,
	hooks *ManagementDashboardRuntimeHooks,
	resource *managementdashboardv1beta1.ManagementDashboard,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("confirm ManagementDashboard delete: resource is nil")
	}
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm ManagementDashboard delete: list hook is not configured")
	}

	response, err := hooks.List.Call(ctx, managementdashboardsdk.ListManagementDashboardsRequest{
		CompartmentId: managementDashboardStringPointer(resource.Spec.CompartmentId),
		DisplayName:   managementDashboardStringPointer(resource.Spec.DisplayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return nil, managementDashboardAuthShapedConfirmRead{err: err}
		}
		return nil, err
	}

	matches := make([]managementdashboardsdk.ManagementDashboardSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if managementDashboardSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, managementDashboardNotFoundError("ManagementDashboard delete confirmation did not find a matching OCI dashboard")
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("managementdashboard list response returned multiple matching resources for compartmentId %q and displayName %q", resource.Spec.CompartmentId, resource.Spec.DisplayName)
	}
}

func managementDashboardDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return managementDashboardAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func handleManagementDashboardDeleteError(resource *managementdashboardv1beta1.ManagementDashboard, err error) error {
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
	return managementDashboardAmbiguousNotFoundError{
		message:      "managementdashboard delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func handleManagementDashboardDeleteConfirmReadOutcome(
	resource *managementdashboardv1beta1.ManagementDashboard,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case managementDashboardAuthShapedConfirmRead:
		recordManagementDashboardConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *managementDashboardAuthShapedConfirmRead:
		if typed != nil {
			recordManagementDashboardConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func recordManagementDashboardConfirmReadRequestID(
	resource *managementdashboardv1beta1.ManagementDashboard,
	err managementDashboardAuthShapedConfirmRead,
) {
	if resource == nil {
		return
	}
	servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, err.GetOpcRequestID())
}

func managementDashboardSummaryMatchesSpec(
	summary managementdashboardsdk.ManagementDashboardSummary,
	spec managementdashboardv1beta1.ManagementDashboardSpec,
) bool {
	return managementDashboardStringValue(summary.CompartmentId) == spec.CompartmentId &&
		managementDashboardStringValue(summary.DisplayName) == spec.DisplayName &&
		managementDashboardStringValue(summary.ProviderId) == spec.ProviderId &&
		managementDashboardStringValue(summary.ProviderName) == spec.ProviderName &&
		managementDashboardStringValue(summary.ProviderVersion) == spec.ProviderVersion &&
		managementDashboardStringValue(summary.Type) == spec.Type &&
		managementDashboardBoolValue(summary.IsOobDashboard) == spec.IsOobDashboard
}

func managementDashboardSummaryID(summary managementdashboardsdk.ManagementDashboardSummary) string {
	if id := managementDashboardStringValue(summary.Id); id != "" {
		return id
	}
	return managementDashboardStringValue(summary.DashboardId)
}

func managementDashboardRecordedID(resource *managementdashboardv1beta1.ManagementDashboard) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func managementDashboardNotFoundError(message string) error {
	return errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    message,
	}
}

func managementDashboardStringPointer(value string) *string {
	return &value
}

func managementDashboardStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func managementDashboardBoolValue(value *bool) bool {
	return value != nil && *value
}
