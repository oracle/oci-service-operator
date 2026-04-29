/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsembridge

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type logAnalyticsEmBridgeOCIClient interface {
	CreateLogAnalyticsEmBridge(context.Context, loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error)
	GetLogAnalyticsEmBridge(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error)
	ListLogAnalyticsEmBridges(context.Context, loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error)
	UpdateLogAnalyticsEmBridge(context.Context, loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest) (loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse, error)
	DeleteLogAnalyticsEmBridge(context.Context, loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error)
	ListNamespaces(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)
}

const logAnalyticsEmBridgeNamespaceAnnotation = "loganalytics.oracle.com/namespace-name"

type logAnalyticsEmBridgeNamespaceContextKey struct{}

type logAnalyticsEmBridgeNamespaceLister interface {
	ListNamespaces(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)
}

type logAnalyticsEmBridgeNamespaceResolver struct {
	provider common.ConfigurationProvider
	lister   logAnalyticsEmBridgeNamespaceLister
}

type logAnalyticsEmBridgeAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e logAnalyticsEmBridgeAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e logAnalyticsEmBridgeAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerLogAnalyticsEmBridgeRuntimeHooksMutator(func(manager *LogAnalyticsEmBridgeServiceManager, hooks *LogAnalyticsEmBridgeRuntimeHooks) {
		provider := logAnalyticsEmBridgeProvider(manager)
		applyLogAnalyticsEmBridgeRuntimeHooks(hooks)
		appendLogAnalyticsEmBridgeNamespaceResolverWrapper(
			hooks,
			provider,
			newLogAnalyticsEmBridgeNamespaceLister(provider),
		)
	})
}

func applyLogAnalyticsEmBridgeRuntimeHooks(hooks *LogAnalyticsEmBridgeRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedLogAnalyticsEmBridgeRuntimeSemantics()
	hooks.Create.Fields = logAnalyticsEmBridgeCreateFields()
	hooks.Get.Fields = logAnalyticsEmBridgeGetFields()
	hooks.List.Fields = logAnalyticsEmBridgeListFields()
	hooks.Update.Fields = logAnalyticsEmBridgeUpdateFields()
	hooks.Delete.Fields = logAnalyticsEmBridgeDeleteFields()
	wrapLogAnalyticsEmBridgeNamespaceRequests(hooks)
	hooks.BuildCreateBody = buildLogAnalyticsEmBridgeCreateBody
	hooks.BuildUpdateBody = buildLogAnalyticsEmBridgeUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardLogAnalyticsEmBridgeExistingBeforeCreate
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateLogAnalyticsEmBridgeCreateOnlyDriftForResponse
	if hooks.List.Call != nil {
		hooks.List.Call = listLogAnalyticsEmBridgesAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleLogAnalyticsEmBridgeDeleteError
	if hooks.Get.Call != nil {
		wrapLogAnalyticsEmBridgeDeleteConfirmation(hooks)
	}
}

func newLogAnalyticsEmBridgeNamespaceLister(provider common.ConfigurationProvider) logAnalyticsEmBridgeNamespaceLister {
	if provider == nil {
		return nil
	}
	client, err := loganalyticssdk.NewLogAnalyticsClientWithConfigurationProvider(provider)
	if err != nil {
		return nil
	}
	return client
}

func logAnalyticsEmBridgeProvider(manager *LogAnalyticsEmBridgeServiceManager) common.ConfigurationProvider {
	if manager == nil {
		return nil
	}
	return manager.Provider
}

func reviewedLogAnalyticsEmBridgeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loganalytics",
		FormalSlug:    "loganalyticsembridge",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(loganalyticssdk.EmBridgeLifecycleStatesCreating)},
			ActiveStates: []string{
				string(loganalyticssdk.EmBridgeLifecycleStatesActive),
				string(loganalyticssdk.EmBridgeLifecycleStatesNeedsAttention),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "best-effort",
			TerminalStates: []string{string(loganalyticssdk.EmBridgeLifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"bucketName",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId", "emEntitiesCompartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LogAnalyticsEmBridge", Action: "CreateLogAnalyticsEmBridge"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LogAnalyticsEmBridge", Action: "UpdateLogAnalyticsEmBridge"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LogAnalyticsEmBridge", Action: "DeleteLogAnalyticsEmBridge"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LogAnalyticsEmBridge", Action: "GetLogAnalyticsEmBridge"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LogAnalyticsEmBridge", Action: "GetLogAnalyticsEmBridge"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LogAnalyticsEmBridge", Action: "GetLogAnalyticsEmBridge"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func logAnalyticsEmBridgeCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		logAnalyticsEmBridgeNamespaceField(),
		{FieldName: "CreateLogAnalyticsEmBridgeDetails", RequestName: "CreateLogAnalyticsEmBridgeDetails", Contribution: "body"},
	}
}

func logAnalyticsEmBridgeGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		logAnalyticsEmBridgeNamespaceField(),
		{FieldName: "LogAnalyticsEmBridgeId", RequestName: "logAnalyticsEmBridgeId", Contribution: "path", PreferResourceID: true},
	}
}

func logAnalyticsEmBridgeListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		logAnalyticsEmBridgeNamespaceField(),
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "LifecycleDetailsContains", RequestName: "lifecycleDetailsContains", Contribution: "query"},
		{FieldName: "ImportStatus", RequestName: "importStatus", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func logAnalyticsEmBridgeUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		logAnalyticsEmBridgeNamespaceField(),
		{FieldName: "LogAnalyticsEmBridgeId", RequestName: "logAnalyticsEmBridgeId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateLogAnalyticsEmBridgeDetails", RequestName: "UpdateLogAnalyticsEmBridgeDetails", Contribution: "body"},
	}
}

func logAnalyticsEmBridgeDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		logAnalyticsEmBridgeNamespaceField(),
		{FieldName: "LogAnalyticsEmBridgeId", RequestName: "logAnalyticsEmBridgeId", Contribution: "path", PreferResourceID: true},
		{FieldName: "IsDeleteEntities", RequestName: "isDeleteEntities", Contribution: "query"},
	}
}

func logAnalyticsEmBridgeNamespaceField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "logAnalyticsNamespaceName",
		Contribution: "path",
	}
}

func wrapLogAnalyticsEmBridgeNamespaceRequests(hooks *LogAnalyticsEmBridgeRuntimeHooks) {
	if hooks == nil {
		return
	}
	if createLogAnalyticsEmBridge := hooks.Create.Call; createLogAnalyticsEmBridge != nil {
		hooks.Create.Call = func(ctx context.Context, request loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error) {
			request.NamespaceName = logAnalyticsEmBridgeRequestNamespace(ctx, request.NamespaceName)
			return createLogAnalyticsEmBridge(ctx, request)
		}
	}
	if getLogAnalyticsEmBridge := hooks.Get.Call; getLogAnalyticsEmBridge != nil {
		hooks.Get.Call = func(ctx context.Context, request loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
			request.NamespaceName = logAnalyticsEmBridgeRequestNamespace(ctx, request.NamespaceName)
			return getLogAnalyticsEmBridge(ctx, request)
		}
	}
	if listLogAnalyticsEmBridges := hooks.List.Call; listLogAnalyticsEmBridges != nil {
		hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
			request.NamespaceName = logAnalyticsEmBridgeRequestNamespace(ctx, request.NamespaceName)
			return listLogAnalyticsEmBridges(ctx, request)
		}
	}
	if updateLogAnalyticsEmBridge := hooks.Update.Call; updateLogAnalyticsEmBridge != nil {
		hooks.Update.Call = func(ctx context.Context, request loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest) (loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse, error) {
			request.NamespaceName = logAnalyticsEmBridgeRequestNamespace(ctx, request.NamespaceName)
			return updateLogAnalyticsEmBridge(ctx, request)
		}
	}
	if deleteLogAnalyticsEmBridge := hooks.Delete.Call; deleteLogAnalyticsEmBridge != nil {
		hooks.Delete.Call = func(ctx context.Context, request loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error) {
			request.NamespaceName = logAnalyticsEmBridgeRequestNamespace(ctx, request.NamespaceName)
			return deleteLogAnalyticsEmBridge(ctx, request)
		}
	}
}

func logAnalyticsEmBridgeRequestNamespace(ctx context.Context, existing *string) *string {
	namespaceName := logAnalyticsEmBridgeNamespaceFromContext(ctx)
	if namespaceName == "" {
		return existing
	}
	return common.String(namespaceName)
}

func buildLogAnalyticsEmBridgeCreateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("LogAnalyticsEmBridge resource is nil")
	}
	if err := validateLogAnalyticsEmBridgeSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := loganalyticssdk.CreateLogAnalyticsEmBridgeDetails{
		DisplayName:             common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		CompartmentId:           common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		EmEntitiesCompartmentId: common.String(strings.TrimSpace(resource.Spec.EmEntitiesCompartmentId)),
		BucketName:              common.String(strings.TrimSpace(resource.Spec.BucketName)),
	}
	if strings.TrimSpace(resource.Spec.Description) != "" {
		body.Description = common.String(resource.Spec.Description)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneLogAnalyticsEmBridgeStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = logAnalyticsEmBridgeDefinedTagsForOCI(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildLogAnalyticsEmBridgeUpdateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails{}, false, fmt.Errorf("LogAnalyticsEmBridge resource is nil")
	}
	if err := validateLogAnalyticsEmBridgeSpec(resource.Spec); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails{}, false, err
	}

	current, ok := logAnalyticsEmBridgeFromResponse(currentResponse)
	if !ok {
		return loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails{}, false, fmt.Errorf("current LogAnalyticsEmBridge response does not expose a LogAnalyticsEmBridge body")
	}
	if err := validateLogAnalyticsEmBridgeCreateOnlyDrift(resource.Spec, current); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails{}, false, err
	}

	updateDetails, updateNeeded := desiredLogAnalyticsEmBridgeUpdateDetails(resource.Spec, current)
	if !updateNeeded {
		return loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func desiredLogAnalyticsEmBridgeUpdateDetails(
	spec loganalyticsv1beta1.LogAnalyticsEmBridgeSpec,
	current loganalyticssdk.LogAnalyticsEmBridge,
) (loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails, bool) {
	updateDetails := loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails{}
	updated := false
	if desired, ok := desiredLogAnalyticsEmBridgeRequiredStringForUpdate(spec.DisplayName, current.DisplayName); ok {
		updateDetails.DisplayName = desired
		updated = true
	}
	if desired, ok := desiredLogAnalyticsEmBridgeOptionalStringForUpdate(spec.Description, current.Description); ok {
		updateDetails.Description = desired
		updated = true
	}
	if desired, ok := desiredLogAnalyticsEmBridgeRequiredStringForUpdate(spec.BucketName, current.BucketName); ok {
		updateDetails.BucketName = desired
		updated = true
	}
	if desired, ok := desiredLogAnalyticsEmBridgeFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags); ok {
		updateDetails.FreeformTags = desired
		updated = true
	}
	if desired, ok := desiredLogAnalyticsEmBridgeDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags); ok {
		updateDetails.DefinedTags = desired
		updated = true
	}

	return updateDetails, updated
}

func validateLogAnalyticsEmBridgeSpec(spec loganalyticsv1beta1.LogAnalyticsEmBridgeSpec) error {
	var missing []string
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.EmEntitiesCompartmentId) == "" {
		missing = append(missing, "emEntitiesCompartmentId")
	}
	if strings.TrimSpace(spec.BucketName) == "" {
		missing = append(missing, "bucketName")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("LogAnalyticsEmBridge spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func guardLogAnalyticsEmBridgeExistingBeforeCreate(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("LogAnalyticsEmBridge resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateLogAnalyticsEmBridgeCreateOnlyDriftForResponse(
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("LogAnalyticsEmBridge resource is nil")
	}
	current, ok := logAnalyticsEmBridgeFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current LogAnalyticsEmBridge response does not expose a LogAnalyticsEmBridge body")
	}
	return validateLogAnalyticsEmBridgeCreateOnlyDrift(resource.Spec, current)
}

func validateLogAnalyticsEmBridgeCreateOnlyDrift(
	spec loganalyticsv1beta1.LogAnalyticsEmBridgeSpec,
	current loganalyticssdk.LogAnalyticsEmBridge,
) error {
	var drift []string
	if !logAnalyticsEmBridgeStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !logAnalyticsEmBridgeStringPtrEqual(current.EmEntitiesCompartmentId, spec.EmEntitiesCompartmentId) {
		drift = append(drift, "emEntitiesCompartmentId")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("LogAnalyticsEmBridge create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func appendLogAnalyticsEmBridgeNamespaceResolverWrapper(
	hooks *LogAnalyticsEmBridgeRuntimeHooks,
	provider common.ConfigurationProvider,
	lister logAnalyticsEmBridgeNamespaceLister,
) {
	if hooks == nil || provider == nil || lister == nil {
		return
	}
	resolver := logAnalyticsEmBridgeNamespaceResolver{provider: provider, lister: lister}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LogAnalyticsEmBridgeServiceClient) LogAnalyticsEmBridgeServiceClient {
		return logAnalyticsEmBridgeNamespaceResolvingClient{
			delegate: delegate,
			resolver: resolver,
		}
	})
}

type logAnalyticsEmBridgeNamespaceResolvingClient struct {
	delegate LogAnalyticsEmBridgeServiceClient
	resolver logAnalyticsEmBridgeNamespaceResolver
}

func (c logAnalyticsEmBridgeNamespaceResolvingClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	ctx, err := c.contextWithNamespace(ctx, resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c logAnalyticsEmBridgeNamespaceResolvingClient) Delete(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) (bool, error) {
	ctx, err := c.contextWithNamespace(ctx, resource)
	if err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c logAnalyticsEmBridgeNamespaceResolvingClient) contextWithNamespace(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) (context.Context, error) {
	namespaceName, err := c.resolver.Resolve(ctx, resource)
	if err != nil {
		return ctx, err
	}
	return logAnalyticsEmBridgeContextWithNamespace(ctx, namespaceName), nil
}

func (r logAnalyticsEmBridgeNamespaceResolver) Resolve(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("LogAnalyticsEmBridge resource is nil")
	}
	if namespaceName := logAnalyticsEmBridgeNamespaceAnnotationValue(resource); namespaceName != "" {
		return namespaceName, nil
	}
	if r.provider == nil {
		return "", fmt.Errorf("resolve LogAnalyticsEmBridge namespaceName: OCI configuration provider is nil")
	}
	if r.lister == nil {
		return "", fmt.Errorf("resolve LogAnalyticsEmBridge namespaceName: Log Analytics namespace lister is nil")
	}
	tenancyID, err := r.provider.TenancyOCID()
	if err != nil {
		return "", fmt.Errorf("resolve LogAnalyticsEmBridge namespaceName tenancy: %w", err)
	}
	tenancyID = strings.TrimSpace(tenancyID)
	if tenancyID == "" {
		return "", fmt.Errorf("resolve LogAnalyticsEmBridge namespaceName: tenancy OCID is empty")
	}
	response, err := r.lister.ListNamespaces(ctx, loganalyticssdk.ListNamespacesRequest{
		CompartmentId: common.String(tenancyID),
	})
	if err != nil {
		return "", fmt.Errorf("lookup LogAnalyticsEmBridge namespaceName: %w", err)
	}
	namespaceName := logAnalyticsEmBridgeNamespaceFromList(response)
	if namespaceName == "" {
		return "", fmt.Errorf("lookup LogAnalyticsEmBridge namespaceName: OCI returned no namespace for tenancy")
	}
	return namespaceName, nil
}

func logAnalyticsEmBridgeNamespaceFromList(response loganalyticssdk.ListNamespacesResponse) string {
	for _, item := range response.Items {
		namespaceName := strings.TrimSpace(logAnalyticsEmBridgeStringValue(item.NamespaceName))
		if namespaceName == "" {
			continue
		}
		if item.IsOnboarded != nil && !*item.IsOnboarded {
			continue
		}
		if item.LifecycleState != "" && item.LifecycleState != loganalyticssdk.NamespaceSummaryLifecycleStateActive {
			continue
		}
		return namespaceName
	}
	for _, item := range response.Items {
		if namespaceName := strings.TrimSpace(logAnalyticsEmBridgeStringValue(item.NamespaceName)); namespaceName != "" {
			return namespaceName
		}
	}
	return ""
}

func logAnalyticsEmBridgeNamespaceAnnotationValue(resource *loganalyticsv1beta1.LogAnalyticsEmBridge) string {
	if resource == nil || len(resource.Annotations) == 0 {
		return ""
	}
	return strings.TrimSpace(resource.Annotations[logAnalyticsEmBridgeNamespaceAnnotation])
}

func logAnalyticsEmBridgeContextWithNamespace(ctx context.Context, namespaceName string) context.Context {
	return context.WithValue(ctx, logAnalyticsEmBridgeNamespaceContextKey{}, strings.TrimSpace(namespaceName))
}

func logAnalyticsEmBridgeNamespaceFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	namespaceName, _ := ctx.Value(logAnalyticsEmBridgeNamespaceContextKey{}).(string)
	return strings.TrimSpace(namespaceName)
}

func listLogAnalyticsEmBridgesAllPages(
	call func(context.Context, loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error),
) func(context.Context, loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
	return func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
		var combined loganalyticssdk.ListLogAnalyticsEmBridgesResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return loganalyticssdk.ListLogAnalyticsEmBridgesResponse{}, err
			}
			appendLogAnalyticsEmBridgeListPage(&combined, response)
			if logAnalyticsEmBridgeNextPage(response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func appendLogAnalyticsEmBridgeListPage(
	combined *loganalyticssdk.ListLogAnalyticsEmBridgesResponse,
	response loganalyticssdk.ListLogAnalyticsEmBridgesResponse,
) {
	combined.RawResponse = response.RawResponse
	combined.OpcRequestId = response.OpcRequestId
	for _, item := range response.Items {
		if item.LifecycleState == loganalyticssdk.EmBridgeLifecycleStatesDeleted {
			continue
		}
		combined.Items = append(combined.Items, item)
	}
}

func logAnalyticsEmBridgeNextPage(page *string) string {
	if page == nil {
		return ""
	}
	return strings.TrimSpace(*page)
}

func handleLogAnalyticsEmBridgeDeleteError(resource *loganalyticsv1beta1.LogAnalyticsEmBridge, err error) error {
	if err == nil {
		return nil
	}
	if !isAmbiguousLogAnalyticsEmBridgeNotFound(err) {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return logAnalyticsEmBridgeAmbiguousError("delete", err)
}

func wrapLogAnalyticsEmBridgeDeleteConfirmation(hooks *LogAnalyticsEmBridgeRuntimeHooks) {
	getLogAnalyticsEmBridge := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LogAnalyticsEmBridgeServiceClient) LogAnalyticsEmBridgeServiceClient {
		return logAnalyticsEmBridgeDeleteConfirmationClient{
			delegate:                delegate,
			getLogAnalyticsEmBridge: getLogAnalyticsEmBridge,
		}
	})
}

type logAnalyticsEmBridgeDeleteConfirmationClient struct {
	delegate                LogAnalyticsEmBridgeServiceClient
	getLogAnalyticsEmBridge func(context.Context, loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error)
}

func (c logAnalyticsEmBridgeDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c logAnalyticsEmBridgeDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c logAnalyticsEmBridgeDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEmBridge,
) error {
	if c.getLogAnalyticsEmBridge == nil || resource == nil {
		return nil
	}
	namespaceName := logAnalyticsEmBridgeNamespaceFromContext(ctx)
	if namespaceName == "" {
		namespaceName = logAnalyticsEmBridgeNamespaceAnnotationValue(resource)
	}
	resourceID := trackedLogAnalyticsEmBridgeID(resource)
	if namespaceName == "" || resourceID == "" {
		if resourceID == "" {
			return nil
		}
		return fmt.Errorf("LogAnalyticsEmBridge namespaceName is required for delete confirmation")
	}

	_, err := c.getLogAnalyticsEmBridge(ctx, loganalyticssdk.GetLogAnalyticsEmBridgeRequest{
		NamespaceName:          common.String(namespaceName),
		LogAnalyticsEmBridgeId: common.String(resourceID),
	})
	if err == nil || !isAmbiguousLogAnalyticsEmBridgeNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return logAnalyticsEmBridgeAmbiguousError("delete confirmation", err)
}

func trackedLogAnalyticsEmBridgeID(resource *loganalyticsv1beta1.LogAnalyticsEmBridge) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func isAmbiguousLogAnalyticsEmBridgeNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous logAnalyticsEmBridgeAmbiguousNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func logAnalyticsEmBridgeAmbiguousError(operation string, err error) logAnalyticsEmBridgeAmbiguousNotFoundError {
	requestID := errorutil.OpcRequestID(err)
	return logAnalyticsEmBridgeAmbiguousNotFoundError{
		message: fmt.Sprintf(
			"LogAnalyticsEmBridge %s returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed: %v",
			strings.TrimSpace(operation),
			err,
		),
		opcRequestID: requestID,
	}
}

func logAnalyticsEmBridgeFromResponse(response any) (loganalyticssdk.LogAnalyticsEmBridge, bool) {
	if current, ok := logAnalyticsEmBridgeFromWriteResponse(response); ok {
		return current, true
	}
	if current, ok := logAnalyticsEmBridgeFromReadResponse(response); ok {
		return current, true
	}
	return logAnalyticsEmBridgeFromListItem(response)
}

func logAnalyticsEmBridgeFromWriteResponse(response any) (loganalyticssdk.LogAnalyticsEmBridge, bool) {
	switch current := response.(type) {
	case loganalyticssdk.CreateLogAnalyticsEmBridgeResponse:
		return current.LogAnalyticsEmBridge, true
	case *loganalyticssdk.CreateLogAnalyticsEmBridgeResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEmBridge{}, false
		}
		return current.LogAnalyticsEmBridge, true
	case loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse:
		return current.LogAnalyticsEmBridge, true
	case *loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEmBridge{}, false
		}
		return current.LogAnalyticsEmBridge, true
	default:
		return loganalyticssdk.LogAnalyticsEmBridge{}, false
	}
}

func logAnalyticsEmBridgeFromReadResponse(response any) (loganalyticssdk.LogAnalyticsEmBridge, bool) {
	switch current := response.(type) {
	case loganalyticssdk.GetLogAnalyticsEmBridgeResponse:
		return current.LogAnalyticsEmBridge, true
	case *loganalyticssdk.GetLogAnalyticsEmBridgeResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEmBridge{}, false
		}
		return current.LogAnalyticsEmBridge, true
	case loganalyticssdk.LogAnalyticsEmBridge:
		return current, true
	case *loganalyticssdk.LogAnalyticsEmBridge:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEmBridge{}, false
		}
		return *current, true
	default:
		return loganalyticssdk.LogAnalyticsEmBridge{}, false
	}
}

func logAnalyticsEmBridgeFromListItem(response any) (loganalyticssdk.LogAnalyticsEmBridge, bool) {
	switch current := response.(type) {
	case loganalyticssdk.LogAnalyticsEmBridgeSummary:
		return logAnalyticsEmBridgeFromSummary(current), true
	case *loganalyticssdk.LogAnalyticsEmBridgeSummary:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEmBridge{}, false
		}
		return logAnalyticsEmBridgeFromSummary(*current), true
	default:
		return loganalyticssdk.LogAnalyticsEmBridge{}, false
	}
}

func logAnalyticsEmBridgeFromSummary(summary loganalyticssdk.LogAnalyticsEmBridgeSummary) loganalyticssdk.LogAnalyticsEmBridge {
	return loganalyticssdk.LogAnalyticsEmBridge{
		Id:                          summary.Id,
		DisplayName:                 summary.DisplayName,
		CompartmentId:               summary.CompartmentId,
		EmEntitiesCompartmentId:     summary.EmEntitiesCompartmentId,
		BucketName:                  summary.BucketName,
		TimeCreated:                 summary.TimeCreated,
		TimeUpdated:                 summary.TimeUpdated,
		LifecycleState:              summary.LifecycleState,
		LastImportProcessingStatus:  summary.LastImportProcessingStatus,
		Description:                 summary.Description,
		LifecycleDetails:            summary.LifecycleDetails,
		LastImportProcessingDetails: summary.LastImportProcessingDetails,
		TimeImportLastProcessed:     summary.TimeImportLastProcessed,
		TimeEmDataLastExtracted:     summary.TimeEmDataLastExtracted,
		FreeformTags:                cloneLogAnalyticsEmBridgeStringMap(summary.FreeformTags),
		DefinedTags:                 cloneLogAnalyticsEmBridgeDefinedTags(summary.DefinedTags),
	}
}

func desiredLogAnalyticsEmBridgeRequiredStringForUpdate(spec string, current *string) (*string, bool) {
	trimmedSpec := strings.TrimSpace(spec)
	if trimmedSpec == "" || trimmedSpec == strings.TrimSpace(logAnalyticsEmBridgeStringValue(current)) {
		return nil, false
	}
	return common.String(trimmedSpec), true
}

func desiredLogAnalyticsEmBridgeOptionalStringForUpdate(spec string, current *string) (*string, bool) {
	trimmedSpec := strings.TrimSpace(spec)
	if trimmedSpec == "" || trimmedSpec == strings.TrimSpace(logAnalyticsEmBridgeStringValue(current)) {
		return nil, false
	}
	return common.String(spec), true
}

func desiredLogAnalyticsEmBridgeFreeformTagsForUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	desired := cloneLogAnalyticsEmBridgeStringMap(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func desiredLogAnalyticsEmBridgeDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := logAnalyticsEmBridgeDefinedTagsForOCI(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func cloneLogAnalyticsEmBridgeStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneLogAnalyticsEmBridgeDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func logAnalyticsEmBridgeDefinedTagsForOCI(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func logAnalyticsEmBridgeStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func logAnalyticsEmBridgeStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(logAnalyticsEmBridgeStringValue(current)) == strings.TrimSpace(desired)
}

func newLogAnalyticsEmBridgeServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	provider common.ConfigurationProvider,
	client logAnalyticsEmBridgeOCIClient,
) LogAnalyticsEmBridgeServiceClient {
	hooks := newLogAnalyticsEmBridgeRuntimeHooksWithOCIClient(client)
	applyLogAnalyticsEmBridgeRuntimeHooks(&hooks)
	appendLogAnalyticsEmBridgeNamespaceResolverWrapper(&hooks, provider, client)
	delegate := defaultLogAnalyticsEmBridgeServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.LogAnalyticsEmBridge](
			buildLogAnalyticsEmBridgeGeneratedRuntimeConfig(&LogAnalyticsEmBridgeServiceManager{Log: log}, hooks),
		),
	}
	return wrapLogAnalyticsEmBridgeGeneratedClient(hooks, delegate)
}

func newLogAnalyticsEmBridgeRuntimeHooksWithOCIClient(client logAnalyticsEmBridgeOCIClient) LogAnalyticsEmBridgeRuntimeHooks {
	return LogAnalyticsEmBridgeRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*loganalyticsv1beta1.LogAnalyticsEmBridge]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*loganalyticsv1beta1.LogAnalyticsEmBridge]{},
		StatusHooks:     generatedruntime.StatusHooks[*loganalyticsv1beta1.LogAnalyticsEmBridge]{},
		ParityHooks:     generatedruntime.ParityHooks[*loganalyticsv1beta1.LogAnalyticsEmBridge]{},
		Async:           generatedruntime.AsyncHooks[*loganalyticsv1beta1.LogAnalyticsEmBridge]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*loganalyticsv1beta1.LogAnalyticsEmBridge]{},
		Create: runtimeOperationHooks[loganalyticssdk.CreateLogAnalyticsEmBridgeRequest, loganalyticssdk.CreateLogAnalyticsEmBridgeResponse]{
			Fields: logAnalyticsEmBridgeCreateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.CreateLogAnalyticsEmBridgeRequest) (loganalyticssdk.CreateLogAnalyticsEmBridgeResponse, error) {
				if client == nil {
					return loganalyticssdk.CreateLogAnalyticsEmBridgeResponse{}, fmt.Errorf("LogAnalyticsEmBridge OCI client is nil")
				}
				return client.CreateLogAnalyticsEmBridge(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loganalyticssdk.GetLogAnalyticsEmBridgeRequest, loganalyticssdk.GetLogAnalyticsEmBridgeResponse]{
			Fields: logAnalyticsEmBridgeGetFields(),
			Call: func(ctx context.Context, request loganalyticssdk.GetLogAnalyticsEmBridgeRequest) (loganalyticssdk.GetLogAnalyticsEmBridgeResponse, error) {
				if client == nil {
					return loganalyticssdk.GetLogAnalyticsEmBridgeResponse{}, fmt.Errorf("LogAnalyticsEmBridge OCI client is nil")
				}
				return client.GetLogAnalyticsEmBridge(ctx, request)
			},
		},
		List: runtimeOperationHooks[loganalyticssdk.ListLogAnalyticsEmBridgesRequest, loganalyticssdk.ListLogAnalyticsEmBridgesResponse]{
			Fields: logAnalyticsEmBridgeListFields(),
			Call: func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEmBridgesRequest) (loganalyticssdk.ListLogAnalyticsEmBridgesResponse, error) {
				if client == nil {
					return loganalyticssdk.ListLogAnalyticsEmBridgesResponse{}, fmt.Errorf("LogAnalyticsEmBridge OCI client is nil")
				}
				return client.ListLogAnalyticsEmBridges(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest, loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse]{
			Fields: logAnalyticsEmBridgeUpdateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.UpdateLogAnalyticsEmBridgeRequest) (loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse, error) {
				if client == nil {
					return loganalyticssdk.UpdateLogAnalyticsEmBridgeResponse{}, fmt.Errorf("LogAnalyticsEmBridge OCI client is nil")
				}
				return client.UpdateLogAnalyticsEmBridge(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest, loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse]{
			Fields: logAnalyticsEmBridgeDeleteFields(),
			Call: func(ctx context.Context, request loganalyticssdk.DeleteLogAnalyticsEmBridgeRequest) (loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse, error) {
				if client == nil {
					return loganalyticssdk.DeleteLogAnalyticsEmBridgeResponse{}, fmt.Errorf("LogAnalyticsEmBridge OCI client is nil")
				}
				return client.DeleteLogAnalyticsEmBridge(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LogAnalyticsEmBridgeServiceClient) LogAnalyticsEmBridgeServiceClient{},
	}
}
