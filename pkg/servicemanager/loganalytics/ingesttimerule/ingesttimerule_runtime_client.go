/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingesttimerule

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ingestTimeRuleKindField              = "FIELD"
	ingestTimeRuleActionMetricExtraction = "METRIC_EXTRACTION"
)

type ingestTimeRuleOCIClient interface {
	CreateIngestTimeRule(context.Context, loganalyticssdk.CreateIngestTimeRuleRequest) (loganalyticssdk.CreateIngestTimeRuleResponse, error)
	GetIngestTimeRule(context.Context, loganalyticssdk.GetIngestTimeRuleRequest) (loganalyticssdk.GetIngestTimeRuleResponse, error)
	ListIngestTimeRules(context.Context, loganalyticssdk.ListIngestTimeRulesRequest) (loganalyticssdk.ListIngestTimeRulesResponse, error)
	UpdateIngestTimeRule(context.Context, loganalyticssdk.UpdateIngestTimeRuleRequest) (loganalyticssdk.UpdateIngestTimeRuleResponse, error)
	DeleteIngestTimeRule(context.Context, loganalyticssdk.DeleteIngestTimeRuleRequest) (loganalyticssdk.DeleteIngestTimeRuleResponse, error)
	ListNamespaces(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)
}

type ingestTimeRuleNamespaceLister interface {
	ListNamespaces(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)
}

type ingestTimeRuleDeleteGuardReader interface {
	GetIngestTimeRule(context.Context, loganalyticssdk.GetIngestTimeRuleRequest) (loganalyticssdk.GetIngestTimeRuleResponse, error)
}

type namespaceResolvingIngestTimeRuleServiceClient struct {
	delegate        IngestTimeRuleServiceClient
	namespaceLister ingestTimeRuleNamespaceLister
	deleteReader    ingestTimeRuleDeleteGuardReader
}

func init() {
	registerIngestTimeRuleRuntimeHooksMutator(func(manager *IngestTimeRuleServiceManager, hooks *IngestTimeRuleRuntimeHooks) {
		applyIngestTimeRuleRuntimeHooks(hooks)
		appendIngestTimeRuleNamespaceRuntimeWrapper(manager, hooks)
	})
}

func applyIngestTimeRuleRuntimeHooks(hooks *IngestTimeRuleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = ingestTimeRuleRuntimeSemantics()
	hooks.BuildCreateBody = buildIngestTimeRuleCreateBody
	hooks.BuildUpdateBody = buildIngestTimeRuleUpdateBody
	hooks.Create.Fields = ingestTimeRuleCreateFields()
	hooks.Get.Fields = ingestTimeRuleGetFields()
	hooks.List.Fields = ingestTimeRuleListFields()
	hooks.Update.Fields = ingestTimeRuleUpdateFields()
	hooks.Delete.Fields = ingestTimeRuleDeleteFields()
	hooks.Identity.Resolve = resolveIngestTimeRuleIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardIngestTimeRuleExistingBeforeCreate
	hooks.Identity.LookupExisting = lookupIngestTimeRuleExistingByID(hooks)
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateIngestTimeRuleCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleIngestTimeRuleDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyIngestTimeRuleDeleteOutcome
	wrapIngestTimeRuleListCall(hooks)
}

func appendIngestTimeRuleNamespaceRuntimeWrapper(manager *IngestTimeRuleServiceManager, hooks *IngestTimeRuleRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate IngestTimeRuleServiceClient) IngestTimeRuleServiceClient {
		client := &namespaceResolvingIngestTimeRuleServiceClient{delegate: delegate}
		sdkClient, err := loganalyticssdk.NewLogAnalyticsClientWithConfigurationProvider(manager.Provider)
		if err == nil {
			client.namespaceLister = sdkClient
			client.deleteReader = sdkClient
		}
		return client
	})
}

func newIngestTimeRuleServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client ingestTimeRuleOCIClient,
) IngestTimeRuleServiceClient {
	hooks := newIngestTimeRuleRuntimeHooksWithOCIClient(client)
	applyIngestTimeRuleRuntimeHooks(&hooks)
	manager := &IngestTimeRuleServiceManager{Log: log}
	delegate := defaultIngestTimeRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.IngestTimeRule](
			buildIngestTimeRuleGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return &namespaceResolvingIngestTimeRuleServiceClient{
		delegate:        wrapIngestTimeRuleGeneratedClient(hooks, delegate),
		namespaceLister: client,
		deleteReader:    client,
	}
}

func newIngestTimeRuleRuntimeHooksWithOCIClient(client ingestTimeRuleOCIClient) IngestTimeRuleRuntimeHooks {
	return IngestTimeRuleRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*loganalyticsv1beta1.IngestTimeRule]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*loganalyticsv1beta1.IngestTimeRule]{},
		StatusHooks:     generatedruntime.StatusHooks[*loganalyticsv1beta1.IngestTimeRule]{},
		ParityHooks:     generatedruntime.ParityHooks[*loganalyticsv1beta1.IngestTimeRule]{},
		Async:           generatedruntime.AsyncHooks[*loganalyticsv1beta1.IngestTimeRule]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*loganalyticsv1beta1.IngestTimeRule]{},
		Create: runtimeOperationHooks[loganalyticssdk.CreateIngestTimeRuleRequest, loganalyticssdk.CreateIngestTimeRuleResponse]{
			Fields: ingestTimeRuleCreateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.CreateIngestTimeRuleRequest) (loganalyticssdk.CreateIngestTimeRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.CreateIngestTimeRuleResponse{}, fmt.Errorf("IngestTimeRule OCI client is nil")
				}
				return client.CreateIngestTimeRule(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loganalyticssdk.GetIngestTimeRuleRequest, loganalyticssdk.GetIngestTimeRuleResponse]{
			Fields: ingestTimeRuleGetFields(),
			Call: func(ctx context.Context, request loganalyticssdk.GetIngestTimeRuleRequest) (loganalyticssdk.GetIngestTimeRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.GetIngestTimeRuleResponse{}, fmt.Errorf("IngestTimeRule OCI client is nil")
				}
				return client.GetIngestTimeRule(ctx, request)
			},
		},
		List: runtimeOperationHooks[loganalyticssdk.ListIngestTimeRulesRequest, loganalyticssdk.ListIngestTimeRulesResponse]{
			Fields: ingestTimeRuleListFields(),
			Call: func(ctx context.Context, request loganalyticssdk.ListIngestTimeRulesRequest) (loganalyticssdk.ListIngestTimeRulesResponse, error) {
				if client == nil {
					return loganalyticssdk.ListIngestTimeRulesResponse{}, fmt.Errorf("IngestTimeRule OCI client is nil")
				}
				return client.ListIngestTimeRules(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loganalyticssdk.UpdateIngestTimeRuleRequest, loganalyticssdk.UpdateIngestTimeRuleResponse]{
			Fields: ingestTimeRuleUpdateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.UpdateIngestTimeRuleRequest) (loganalyticssdk.UpdateIngestTimeRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.UpdateIngestTimeRuleResponse{}, fmt.Errorf("IngestTimeRule OCI client is nil")
				}
				return client.UpdateIngestTimeRule(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loganalyticssdk.DeleteIngestTimeRuleRequest, loganalyticssdk.DeleteIngestTimeRuleResponse]{
			Fields: ingestTimeRuleDeleteFields(),
			Call: func(ctx context.Context, request loganalyticssdk.DeleteIngestTimeRuleRequest) (loganalyticssdk.DeleteIngestTimeRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.DeleteIngestTimeRuleResponse{}, fmt.Errorf("IngestTimeRule OCI client is nil")
				}
				return client.DeleteIngestTimeRule(ctx, request)
			},
		},
		WrapGeneratedClient: []func(IngestTimeRuleServiceClient) IngestTimeRuleServiceClient{},
	}
}

func ingestTimeRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loganalytics",
		FormalSlug:    "ingesttimerule",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(loganalyticssdk.ConfigLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(loganalyticssdk.ConfigLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"id",
				"compartmentId",
				"displayName",
				"conditionKind",
				"fieldName",
				"fieldValue",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"id",
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
				"conditions",
				"actions",
				"isEnabled",
				"timeCreated",
				"timeUpdated",
				"lifecycleState",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IngestTimeRule", Action: "CreateIngestTimeRule"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IngestTimeRule", Action: "UpdateIngestTimeRule"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IngestTimeRule", Action: "DeleteIngestTimeRule"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "IngestTimeRule", Action: "GetIngestTimeRule"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "IngestTimeRule", Action: "GetIngestTimeRule"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "IngestTimeRule", Action: "GetIngestTimeRule"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func ingestTimeRuleCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ingestTimeRuleNamespaceField(),
		{FieldName: "CreateIngestTimeRuleDetails", RequestName: "CreateIngestTimeRuleDetails", Contribution: "body"},
	}
}

func ingestTimeRuleGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ingestTimeRuleNamespaceField(),
		{FieldName: "IngestTimeRuleId", RequestName: "ingestTimeRuleId", Contribution: "path", PreferResourceID: true},
	}
}

func ingestTimeRuleListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ingestTimeRuleNamespaceField(),
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"spec.compartmentId", "status.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"spec.displayName", "status.displayName", "displayName"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", LookupPaths: []string{"spec.lifecycleState", "status.lifecycleState", "lifecycleState"}},
		{FieldName: "ConditionKind", RequestName: "conditionKind", Contribution: "query", LookupPaths: []string{"spec.conditions.kind", "status.conditionKind", "conditionKind"}},
		{FieldName: "FieldName", RequestName: "fieldName", Contribution: "query", LookupPaths: []string{"spec.conditions.fieldName", "status.fieldName", "fieldName"}},
		{FieldName: "FieldValue", RequestName: "fieldValue", Contribution: "query", LookupPaths: []string{"spec.conditions.fieldValue", "status.fieldValue", "fieldValue"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func ingestTimeRuleUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ingestTimeRuleNamespaceField(),
		{FieldName: "IngestTimeRuleId", RequestName: "ingestTimeRuleId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateIngestTimeRuleDetails", RequestName: "UpdateIngestTimeRuleDetails", Contribution: "body"},
	}
}

func ingestTimeRuleDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		ingestTimeRuleNamespaceField(),
		{FieldName: "IngestTimeRuleId", RequestName: "ingestTimeRuleId", Contribution: "path", PreferResourceID: true},
	}
}

func ingestTimeRuleNamespaceField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "namespaceName",
		Contribution: "path",
		LookupPaths:  []string{"namespaceName", "namespace"},
	}
}

func (c *namespaceResolvingIngestTimeRuleServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	namespace, err := c.resolveNamespace(ctx, resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return c.withNamespace(resource, namespace, func() (servicemanager.OSOKResponse, error) {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	})
}

func (c *namespaceResolvingIngestTimeRuleServiceClient) Delete(
	ctx context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
) (bool, error) {
	namespace, err := c.resolveNamespace(ctx, resource)
	if err != nil {
		return false, err
	}
	return c.withNamespaceForDelete(resource, namespace, func() (bool, error) {
		if err := c.guardBeforeDelete(ctx, resource); err != nil {
			return false, err
		}
		return c.delegate.Delete(ctx, resource)
	})
}

func (c *namespaceResolvingIngestTimeRuleServiceClient) resolveNamespace(
	ctx context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("IngestTimeRule resource is nil")
	}
	if c.namespaceLister == nil {
		return strings.TrimSpace(resource.Namespace), nil
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return "", nil
	}

	response, err := c.namespaceLister.ListNamespaces(ctx, loganalyticssdk.ListNamespacesRequest{
		CompartmentId: common.String(compartmentID),
	})
	if err != nil {
		return "", fmt.Errorf("lookup IngestTimeRule namespace: %w", err)
	}
	namespace := selectIngestTimeRuleNamespace(response.Items)
	if namespace == "" {
		return "", fmt.Errorf("lookup IngestTimeRule namespace: OCI returned no namespace for compartment %q", compartmentID)
	}
	return namespace, nil
}

func selectIngestTimeRuleNamespace(items []loganalyticssdk.NamespaceSummary) string {
	if namespace := selectActiveIngestTimeRuleNamespace(items); namespace != "" {
		return namespace
	}
	return selectFallbackIngestTimeRuleNamespace(items)
}

func selectActiveIngestTimeRuleNamespace(items []loganalyticssdk.NamespaceSummary) string {
	for _, item := range items {
		namespace := ingestTimeRuleNamespaceName(item)
		if namespace != "" && ingestTimeRuleNamespaceAvailable(item) {
			return namespace
		}
	}
	return ""
}

func selectFallbackIngestTimeRuleNamespace(items []loganalyticssdk.NamespaceSummary) string {
	for _, item := range items {
		if namespace := ingestTimeRuleNamespaceName(item); namespace != "" {
			return namespace
		}
	}
	return ""
}

func ingestTimeRuleNamespaceName(item loganalyticssdk.NamespaceSummary) string {
	if item.NamespaceName == nil {
		return ""
	}
	return strings.TrimSpace(*item.NamespaceName)
}

func ingestTimeRuleNamespaceAvailable(item loganalyticssdk.NamespaceSummary) bool {
	if item.IsOnboarded != nil && !*item.IsOnboarded {
		return false
	}
	return item.LifecycleState == "" || item.LifecycleState == loganalyticssdk.NamespaceSummaryLifecycleStateActive
}

func (c *namespaceResolvingIngestTimeRuleServiceClient) withNamespace(
	resource *loganalyticsv1beta1.IngestTimeRule,
	namespace string,
	fn func() (servicemanager.OSOKResponse, error),
) (servicemanager.OSOKResponse, error) {
	if strings.TrimSpace(namespace) == "" {
		return fn()
	}
	original := resource.Namespace
	resource.Namespace = namespace
	defer func() {
		resource.Namespace = original
	}()
	return fn()
}

func (c *namespaceResolvingIngestTimeRuleServiceClient) withNamespaceForDelete(
	resource *loganalyticsv1beta1.IngestTimeRule,
	namespace string,
	fn func() (bool, error),
) (bool, error) {
	if strings.TrimSpace(namespace) == "" {
		return fn()
	}
	original := resource.Namespace
	resource.Namespace = namespace
	defer func() {
		resource.Namespace = original
	}()
	return fn()
}

func (c *namespaceResolvingIngestTimeRuleServiceClient) guardBeforeDelete(
	ctx context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
) error {
	if c.deleteReader == nil || resource == nil {
		return nil
	}
	namespace := strings.TrimSpace(resource.Namespace)
	ruleID := ingestTimeRuleResourceID(resource)
	if namespace == "" || ruleID == "" {
		return nil
	}
	_, err := c.deleteReader.GetIngestTimeRule(ctx, loganalyticssdk.GetIngestTimeRuleRequest{
		NamespaceName:    common.String(namespace),
		IngestTimeRuleId: common.String(ruleID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("IngestTimeRule delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func resolveIngestTimeRuleIdentity(resource *loganalyticsv1beta1.IngestTimeRule) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("IngestTimeRule resource is nil")
	}
	if statusID := ingestTimeRuleStatusID(resource); statusID != "" {
		return statusID, nil
	}
	id := strings.TrimSpace(resource.Spec.Id)
	if id == "" {
		return nil, nil
	}
	return id, nil
}

func guardIngestTimeRuleExistingBeforeCreate(
	_ context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("IngestTimeRule resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupIngestTimeRuleExistingByID(
	hooks *IngestTimeRuleRuntimeHooks,
) func(context.Context, *loganalyticsv1beta1.IngestTimeRule, any) (any, error) {
	return func(ctx context.Context, resource *loganalyticsv1beta1.IngestTimeRule, identity any) (any, error) {
		if hooks == nil || hooks.Get.Call == nil || resource == nil {
			return nil, nil
		}
		id, ok := identity.(string)
		if !ok || strings.TrimSpace(id) == "" {
			return nil, nil
		}
		namespace := strings.TrimSpace(resource.Namespace)
		if namespace == "" {
			return nil, nil
		}
		response, err := hooks.Get.Call(ctx, loganalyticssdk.GetIngestTimeRuleRequest{
			NamespaceName:    common.String(namespace),
			IngestTimeRuleId: common.String(strings.TrimSpace(id)),
		})
		if err != nil {
			return nil, err
		}
		return response, nil
	}
}

func buildIngestTimeRuleCreateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("IngestTimeRule resource is nil")
	}
	if err := validateIngestTimeRuleSpec(resource.Spec); err != nil {
		return nil, err
	}

	conditions, err := ingestTimeRuleConditionFromSpec(resource.Spec.Conditions)
	if err != nil {
		return nil, err
	}
	actions, err := ingestTimeRuleActionsFromSpec(resource.Spec.Actions)
	if err != nil {
		return nil, err
	}

	details := loganalyticssdk.CreateIngestTimeRuleDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		Conditions:    conditions,
		Actions:       actions,
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		details.Description = common.String(description)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildIngestTimeRuleUpdateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.IngestTimeRule,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return loganalyticssdk.IngestTimeRule{}, false, fmt.Errorf("IngestTimeRule resource is nil")
	}
	if err := validateIngestTimeRuleSpec(resource.Spec); err != nil {
		return loganalyticssdk.IngestTimeRule{}, false, err
	}
	current, ok := ingestTimeRuleFromResponse(currentResponse)
	if !ok {
		return loganalyticssdk.IngestTimeRule{}, false, fmt.Errorf("current IngestTimeRule response does not expose an IngestTimeRule body")
	}
	if err := validateIngestTimeRuleCreateOnlyDrift(resource.Spec, current); err != nil {
		return loganalyticssdk.IngestTimeRule{}, false, err
	}

	desired, err := desiredIngestTimeRule(resource.Spec, current)
	if err != nil {
		return loganalyticssdk.IngestTimeRule{}, false, err
	}
	updateNeeded := ingestTimeRuleUpdateNeeded(resource.Spec, current, desired)
	return desired, updateNeeded, nil
}

func desiredIngestTimeRule(
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
	current loganalyticssdk.IngestTimeRule,
) (loganalyticssdk.IngestTimeRule, error) {
	conditions, err := ingestTimeRuleConditionFromSpec(spec.Conditions)
	if err != nil {
		return loganalyticssdk.IngestTimeRule{}, err
	}
	actions, err := ingestTimeRuleActionsFromSpec(spec.Actions)
	if err != nil {
		return loganalyticssdk.IngestTimeRule{}, err
	}

	id := stringPtrValue(current.Id)
	if id == "" {
		id = strings.TrimSpace(spec.Id)
	}
	compartmentID := stringPtrValue(current.CompartmentId)
	if compartmentID == "" {
		compartmentID = strings.TrimSpace(spec.CompartmentId)
	}
	desired := loganalyticssdk.IngestTimeRule{
		Id:             optionalString(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(strings.TrimSpace(spec.DisplayName)),
		Description:    current.Description,
		FreeformTags:   current.FreeformTags,
		DefinedTags:    current.DefinedTags,
		LifecycleState: current.LifecycleState,
		IsEnabled:      common.Bool(spec.IsEnabled),
		Conditions:     conditions,
		Actions:        actions,
	}
	if strings.TrimSpace(spec.Description) != "" {
		desired.Description = common.String(strings.TrimSpace(spec.Description))
	}
	if spec.FreeformTags != nil {
		desired.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		desired.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return desired, nil
}

func ingestTimeRuleUpdateNeeded(
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
	current loganalyticssdk.IngestTimeRule,
	desired loganalyticssdk.IngestTimeRule,
) bool {
	checks := []func() bool{
		func() bool { return stringPtrValue(current.DisplayName) != stringPtrValue(desired.DisplayName) },
		func() bool { return ingestTimeRuleDescriptionUpdateNeeded(spec, current, desired) },
		func() bool { return ingestTimeRuleFreeformTagUpdateNeeded(spec, current, desired) },
		func() bool { return ingestTimeRuleDefinedTagUpdateNeeded(spec, current, desired) },
		func() bool { return ingestTimeRuleBoolUpdateNeeded(spec.IsEnabled, current.IsEnabled) },
		func() bool { return !jsonEqual(current.Conditions, desired.Conditions) },
		func() bool { return !jsonEqual(current.Actions, desired.Actions) },
	}
	for _, check := range checks {
		if check() {
			return true
		}
	}
	return false
}

func ingestTimeRuleDescriptionUpdateNeeded(
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
	current loganalyticssdk.IngestTimeRule,
	desired loganalyticssdk.IngestTimeRule,
) bool {
	return strings.TrimSpace(spec.Description) != "" &&
		stringPtrValue(current.Description) != stringPtrValue(desired.Description)
}

func ingestTimeRuleFreeformTagUpdateNeeded(
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
	current loganalyticssdk.IngestTimeRule,
	desired loganalyticssdk.IngestTimeRule,
) bool {
	return spec.FreeformTags != nil && !jsonEqual(current.FreeformTags, desired.FreeformTags)
}

func ingestTimeRuleDefinedTagUpdateNeeded(
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
	current loganalyticssdk.IngestTimeRule,
	desired loganalyticssdk.IngestTimeRule,
) bool {
	return spec.DefinedTags != nil && !jsonEqual(current.DefinedTags, desired.DefinedTags)
}

func ingestTimeRuleBoolUpdateNeeded(spec bool, current *bool) bool {
	return current == nil || spec != *current
}

func validateIngestTimeRuleSpec(spec loganalyticsv1beta1.IngestTimeRuleSpec) error {
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("IngestTimeRule spec.compartmentId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		return fmt.Errorf("IngestTimeRule spec.displayName is required")
	}
	if _, err := ingestTimeRuleConditionFromSpec(spec.Conditions); err != nil {
		return err
	}
	if _, err := ingestTimeRuleActionsFromSpec(spec.Actions); err != nil {
		return err
	}
	return nil
}

func ingestTimeRuleConditionFromSpec(
	spec loganalyticsv1beta1.IngestTimeRuleConditions,
) (loganalyticssdk.IngestTimeRuleCondition, error) {
	kind := strings.ToUpper(strings.TrimSpace(spec.Kind))
	if kind == "" {
		kind = ingestTimeRuleKindField
	}
	if kind != ingestTimeRuleKindField {
		return nil, fmt.Errorf("IngestTimeRule conditions.kind %q is not supported", spec.Kind)
	}
	if strings.TrimSpace(spec.FieldName) == "" {
		return nil, fmt.Errorf("IngestTimeRule conditions.fieldName is required")
	}
	if strings.TrimSpace(spec.FieldValue) == "" {
		return nil, fmt.Errorf("IngestTimeRule conditions.fieldValue is required")
	}
	if strings.TrimSpace(spec.FieldOperator) == "" {
		return nil, fmt.Errorf("IngestTimeRule conditions.fieldOperator is required")
	}
	operator, ok := loganalyticssdk.GetMappingIngestTimeRuleFieldConditionFieldOperatorEnum(spec.FieldOperator)
	if !ok {
		return nil, fmt.Errorf("IngestTimeRule conditions.fieldOperator %q is not supported", spec.FieldOperator)
	}

	additionalConditions, err := ingestTimeRuleAdditionalConditionsFromSpec(spec.AdditionalConditions)
	if err != nil {
		return nil, err
	}

	return loganalyticssdk.IngestTimeRuleFieldCondition{
		FieldName:            common.String(strings.TrimSpace(spec.FieldName)),
		FieldValue:           common.String(strings.TrimSpace(spec.FieldValue)),
		FieldOperator:        operator,
		AdditionalConditions: additionalConditions,
	}, nil
}

func ingestTimeRuleAdditionalConditionsFromSpec(
	spec []loganalyticsv1beta1.IngestTimeRuleConditionsAdditionalCondition,
) ([]loganalyticssdk.IngestTimeRuleAdditionalFieldCondition, error) {
	if len(spec) == 0 {
		return nil, nil
	}
	conditions := make([]loganalyticssdk.IngestTimeRuleAdditionalFieldCondition, 0, len(spec))
	for index, item := range spec {
		if strings.TrimSpace(item.ConditionField) == "" {
			return nil, fmt.Errorf("IngestTimeRule conditions.additionalConditions[%d].conditionField is required", index)
		}
		if strings.TrimSpace(item.ConditionOperator) == "" {
			return nil, fmt.Errorf("IngestTimeRule conditions.additionalConditions[%d].conditionOperator is required", index)
		}
		if strings.TrimSpace(item.ConditionValue) == "" {
			return nil, fmt.Errorf("IngestTimeRule conditions.additionalConditions[%d].conditionValue is required", index)
		}
		operator, ok := loganalyticssdk.GetMappingIngestTimeRuleAdditionalFieldConditionConditionOperatorEnum(item.ConditionOperator)
		if !ok {
			return nil, fmt.Errorf("IngestTimeRule conditions.additionalConditions[%d].conditionOperator %q is not supported", index, item.ConditionOperator)
		}
		conditions = append(conditions, loganalyticssdk.IngestTimeRuleAdditionalFieldCondition{
			ConditionField:    common.String(strings.TrimSpace(item.ConditionField)),
			ConditionOperator: operator,
			ConditionValue:    common.String(strings.TrimSpace(item.ConditionValue)),
		})
	}
	return conditions, nil
}

func ingestTimeRuleActionsFromSpec(
	spec []loganalyticsv1beta1.IngestTimeRuleAction,
) ([]loganalyticssdk.IngestTimeRuleAction, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("IngestTimeRule spec.actions must contain at least one action")
	}
	actions := make([]loganalyticssdk.IngestTimeRuleAction, 0, len(spec))
	for index, item := range spec {
		action, err := ingestTimeRuleActionFromSpec(index, item)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func ingestTimeRuleActionFromSpec(
	index int,
	spec loganalyticsv1beta1.IngestTimeRuleAction,
) (loganalyticssdk.IngestTimeRuleAction, error) {
	actionType := strings.ToUpper(strings.TrimSpace(spec.Type))
	if actionType == "" {
		actionType = ingestTimeRuleActionMetricExtraction
	}
	if actionType != ingestTimeRuleActionMetricExtraction {
		return nil, fmt.Errorf("IngestTimeRule actions[%d].type %q is not supported", index, spec.Type)
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return nil, fmt.Errorf("IngestTimeRule actions[%d].compartmentId is required", index)
	}
	if strings.TrimSpace(spec.Namespace) == "" {
		return nil, fmt.Errorf("IngestTimeRule actions[%d].namespace is required", index)
	}
	if strings.TrimSpace(spec.MetricName) == "" {
		return nil, fmt.Errorf("IngestTimeRule actions[%d].metricName is required", index)
	}

	action := loganalyticssdk.IngestTimeRuleMetricExtractionAction{
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
		Namespace:     common.String(strings.TrimSpace(spec.Namespace)),
		MetricName:    common.String(strings.TrimSpace(spec.MetricName)),
	}
	if strings.TrimSpace(spec.ResourceGroup) != "" {
		action.ResourceGroup = common.String(strings.TrimSpace(spec.ResourceGroup))
	}
	if len(spec.Dimensions) != 0 {
		action.Dimensions = append([]string(nil), spec.Dimensions...)
	}
	return action, nil
}

func validateIngestTimeRuleCreateOnlyDriftForResponse(
	resource *loganalyticsv1beta1.IngestTimeRule,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("IngestTimeRule resource is nil")
	}
	current, ok := ingestTimeRuleFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateIngestTimeRuleCreateOnlyDrift(resource.Spec, current)
}

func validateIngestTimeRuleCreateOnlyDrift(
	spec loganalyticsv1beta1.IngestTimeRuleSpec,
	current loganalyticssdk.IngestTimeRule,
) error {
	currentCompartmentID := stringPtrValue(current.CompartmentId)
	if currentCompartmentID != "" && strings.TrimSpace(spec.CompartmentId) != currentCompartmentID {
		return fmt.Errorf("IngestTimeRule compartmentId is create-only; desired %q does not match OCI %q", strings.TrimSpace(spec.CompartmentId), currentCompartmentID)
	}
	if strings.TrimSpace(spec.TimeCreated) != "" && strings.TrimSpace(spec.TimeCreated) != sdkTimeString(current.TimeCreated) {
		return fmt.Errorf("IngestTimeRule timeCreated is observed-only and cannot be updated")
	}
	if strings.TrimSpace(spec.TimeUpdated) != "" && strings.TrimSpace(spec.TimeUpdated) != sdkTimeString(current.TimeUpdated) {
		return fmt.Errorf("IngestTimeRule timeUpdated is observed-only and cannot be updated")
	}
	if strings.TrimSpace(spec.LifecycleState) != "" &&
		!strings.EqualFold(strings.TrimSpace(spec.LifecycleState), string(current.LifecycleState)) {
		return fmt.Errorf("IngestTimeRule lifecycleState is observed-only and cannot be updated")
	}
	return nil
}

func wrapIngestTimeRuleListCall(hooks *IngestTimeRuleRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	list := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListIngestTimeRulesRequest) (loganalyticssdk.ListIngestTimeRulesResponse, error) {
		return listAllIngestTimeRulePages(ctx, list, request)
	}
}

func listAllIngestTimeRulePages(
	ctx context.Context,
	list func(context.Context, loganalyticssdk.ListIngestTimeRulesRequest) (loganalyticssdk.ListIngestTimeRulesResponse, error),
	request loganalyticssdk.ListIngestTimeRulesRequest,
) (loganalyticssdk.ListIngestTimeRulesResponse, error) {
	combined := loganalyticssdk.ListIngestTimeRulesResponse{}
	seenPages := map[string]struct{}{}
	for {
		response, err := list(ctx, request)
		if err != nil {
			return loganalyticssdk.ListIngestTimeRulesResponse{}, err
		}
		mergeIngestTimeRuleListPage(&combined, response)
		nextPage, ok, err := nextIngestTimeRuleListPage(response, seenPages)
		if err != nil || !ok {
			return combined, err
		}
		request.Page = common.String(nextPage)
		combined.OpcNextPage = common.String(nextPage)
	}
}

func mergeIngestTimeRuleListPage(
	combined *loganalyticssdk.ListIngestTimeRulesResponse,
	response loganalyticssdk.ListIngestTimeRulesResponse,
) {
	combined.RawResponse = response.RawResponse
	combined.OpcRequestId = response.OpcRequestId
	for _, item := range response.Items {
		if item.LifecycleState != loganalyticssdk.ConfigLifecycleStateDeleted {
			combined.Items = append(combined.Items, item)
		}
	}
}

func nextIngestTimeRuleListPage(
	response loganalyticssdk.ListIngestTimeRulesResponse,
	seenPages map[string]struct{},
) (string, bool, error) {
	if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
		return "", false, nil
	}
	nextPage := strings.TrimSpace(*response.OpcNextPage)
	if _, seen := seenPages[nextPage]; seen {
		return "", false, fmt.Errorf("IngestTimeRule list pagination repeated page token %q", nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return nextPage, true, nil
}

func handleIngestTimeRuleDeleteError(resource *loganalyticsv1beta1.IngestTimeRule, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return ingestTimeRuleAmbiguousNotFoundError{
		message: fmt.Sprintf(
			"IngestTimeRule delete returned ambiguous not-found response (HTTP %s, code %s); retaining finalizer until OCI deletion is confirmed",
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func applyIngestTimeRuleDeleteOutcome(
	resource *loganalyticsv1beta1.IngestTimeRule,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	current, ok := ingestTimeRuleFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	switch current.LifecycleState {
	case loganalyticssdk.ConfigLifecycleStateDeleted:
		markIngestTimeRuleDeleted(resource, "OCI resource deleted")
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	case loganalyticssdk.ConfigLifecycleStateActive, "":
		if stage == generatedruntime.DeleteConfirmStageAfterRequest {
			markIngestTimeRuleTerminating(resource, "OCI resource delete is in progress")
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markIngestTimeRuleDeleted(resource *loganalyticsv1beta1.IngestTimeRule, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func markIngestTimeRuleTerminating(resource *loganalyticsv1beta1.IngestTimeRule, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func ingestTimeRuleFromResponse(response any) (loganalyticssdk.IngestTimeRule, bool) {
	switch current := response.(type) {
	case loganalyticssdk.IngestTimeRule:
		return current, true
	case *loganalyticssdk.IngestTimeRule:
		if current == nil {
			return loganalyticssdk.IngestTimeRule{}, false
		}
		return *current, true
	case loganalyticssdk.IngestTimeRuleSummary:
		return ingestTimeRuleFromSummary(current), true
	case *loganalyticssdk.IngestTimeRuleSummary:
		if current == nil {
			return loganalyticssdk.IngestTimeRule{}, false
		}
		return ingestTimeRuleFromSummary(*current), true
	case loganalyticssdk.CreateIngestTimeRuleResponse:
		return current.IngestTimeRule, true
	case loganalyticssdk.GetIngestTimeRuleResponse:
		return current.IngestTimeRule, true
	case loganalyticssdk.UpdateIngestTimeRuleResponse:
		return current.IngestTimeRule, true
	default:
		return loganalyticssdk.IngestTimeRule{}, false
	}
}

func ingestTimeRuleFromSummary(summary loganalyticssdk.IngestTimeRuleSummary) loganalyticssdk.IngestTimeRule {
	return loganalyticssdk.IngestTimeRule{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		Description:    summary.Description,
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
		TimeCreated:    summary.TimeCreated,
		TimeUpdated:    summary.TimeUpdated,
		LifecycleState: summary.LifecycleState,
		IsEnabled:      summary.IsEnabled,
	}
}

func ingestTimeRuleResourceID(resource *loganalyticsv1beta1.IngestTimeRule) string {
	if statusID := ingestTimeRuleStatusID(resource); statusID != "" {
		return statusID
	}
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Spec.Id)
}

func ingestTimeRuleStatusID(resource *loganalyticsv1beta1.IngestTimeRule) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(strings.TrimSpace(value))
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02T15:04:05Z07:00")
}

func jsonEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprintf("%#v", left) == fmt.Sprintf("%#v", right)
	}
	return string(leftPayload) == string(rightPayload)
}

type ingestTimeRuleAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ingestTimeRuleAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e ingestTimeRuleAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var _ error = ingestTimeRuleAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = ingestTimeRuleAmbiguousNotFoundError{}
