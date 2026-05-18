/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package reportdefinition

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
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

const reportDefinitionKind = "ReportDefinition"

type reportDefinitionOCIClient interface {
	CreateReportDefinition(context.Context, datasafesdk.CreateReportDefinitionRequest) (datasafesdk.CreateReportDefinitionResponse, error)
	GetReportDefinition(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error)
	ListReportDefinitions(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error)
	UpdateReportDefinition(context.Context, datasafesdk.UpdateReportDefinitionRequest) (datasafesdk.UpdateReportDefinitionResponse, error)
	DeleteReportDefinition(context.Context, datasafesdk.DeleteReportDefinitionRequest) (datasafesdk.DeleteReportDefinitionResponse, error)
}

type reportDefinitionIdentity struct {
	compartmentID string
	displayName   string
	parentID      string
}

type reportDefinitionRuntimeClient struct {
	delegate ReportDefinitionServiceClient
	get      func(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error)
	list     func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error)
}

type reportDefinitionAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e reportDefinitionAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e reportDefinitionAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerReportDefinitionRuntimeHooksMutator(func(_ *ReportDefinitionServiceManager, hooks *ReportDefinitionRuntimeHooks) {
		applyReportDefinitionRuntimeHooks(hooks)
	})
}

func applyReportDefinitionRuntimeHooks(hooks *ReportDefinitionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reportDefinitionRuntimeSemantics()
	hooks.BuildCreateBody = buildReportDefinitionCreateBody
	hooks.BuildUpdateBody = buildReportDefinitionUpdateBody
	hooks.Identity.Resolve = resolveReportDefinitionIdentity
	hooks.Identity.RecordPath = recordReportDefinitionPathIdentity
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedReportDefinitionIdentity
	hooks.Create.Fields = reportDefinitionCreateFields()
	hooks.Get.Fields = reportDefinitionGetFields()
	hooks.List.Fields = reportDefinitionListFields()
	hooks.Update.Fields = reportDefinitionUpdateFields()
	hooks.Delete.Fields = reportDefinitionDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listReportDefinitionsAllPages(hooks.List.Call)
	}
	hooks.Identity.LookupExisting = lookupReportDefinitionExisting(hooks.List.Call, hooks.Get.Call)
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateReportDefinitionCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = confirmReportDefinitionDeleteRead(hooks.List.Call, hooks.Get.Call)
	hooks.DeleteHooks.HandleError = handleReportDefinitionDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyReportDefinitionDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectReportDefinitionStatus
	hooks.StatusHooks.MarkTerminating = markReportDefinitionTerminating
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ReportDefinitionServiceClient) ReportDefinitionServiceClient {
		return reportDefinitionRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newReportDefinitionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client reportDefinitionOCIClient,
) ReportDefinitionServiceClient {
	hooks := newReportDefinitionRuntimeHooksWithOCIClient(client)
	applyReportDefinitionRuntimeHooks(&hooks)
	manager := &ReportDefinitionServiceManager{Log: log}
	delegate := defaultReportDefinitionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.ReportDefinition](
			buildReportDefinitionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapReportDefinitionGeneratedClient(hooks, delegate)
}

func newReportDefinitionRuntimeHooksWithOCIClient(client reportDefinitionOCIClient) ReportDefinitionRuntimeHooks {
	return ReportDefinitionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.ReportDefinition]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.ReportDefinition]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.ReportDefinition]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.ReportDefinition]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.ReportDefinition]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.ReportDefinition]{},
		Create: runtimeOperationHooks[datasafesdk.CreateReportDefinitionRequest, datasafesdk.CreateReportDefinitionResponse]{
			Fields: reportDefinitionCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateReportDefinitionRequest) (datasafesdk.CreateReportDefinitionResponse, error) {
				if client == nil {
					return datasafesdk.CreateReportDefinitionResponse{}, fmt.Errorf("%s OCI client is nil", reportDefinitionKind)
				}
				return client.CreateReportDefinition(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetReportDefinitionRequest, datasafesdk.GetReportDefinitionResponse]{
			Fields: reportDefinitionGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error) {
				if client == nil {
					return datasafesdk.GetReportDefinitionResponse{}, fmt.Errorf("%s OCI client is nil", reportDefinitionKind)
				}
				return client.GetReportDefinition(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListReportDefinitionsRequest, datasafesdk.ListReportDefinitionsResponse]{
			Fields: reportDefinitionListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error) {
				if client == nil {
					return datasafesdk.ListReportDefinitionsResponse{}, fmt.Errorf("%s OCI client is nil", reportDefinitionKind)
				}
				return client.ListReportDefinitions(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateReportDefinitionRequest, datasafesdk.UpdateReportDefinitionResponse]{
			Fields: reportDefinitionUpdateFields(),
			Call: func(ctx context.Context, request datasafesdk.UpdateReportDefinitionRequest) (datasafesdk.UpdateReportDefinitionResponse, error) {
				if client == nil {
					return datasafesdk.UpdateReportDefinitionResponse{}, fmt.Errorf("%s OCI client is nil", reportDefinitionKind)
				}
				return client.UpdateReportDefinition(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteReportDefinitionRequest, datasafesdk.DeleteReportDefinitionResponse]{
			Fields: reportDefinitionDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteReportDefinitionRequest) (datasafesdk.DeleteReportDefinitionResponse, error) {
				if client == nil {
					return datasafesdk.DeleteReportDefinitionResponse{}, fmt.Errorf("%s OCI client is nil", reportDefinitionKind)
				}
				return client.DeleteReportDefinition(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ReportDefinitionServiceClient) ReportDefinitionServiceClient{},
	}
}

func reportDefinitionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "reportdefinition",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.ReportDefinitionLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.ReportDefinitionLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.ReportDefinitionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(datasafesdk.ReportDefinitionLifecycleStateDeleting),
			},
			TerminalStates: []string{
				string(datasafesdk.ReportDefinitionLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"parentId",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"columnInfo",
				"columnFilters",
				"columnSortings",
				"summary",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"parentId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: reportDefinitionKind, Action: "CreateReportDefinition"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: reportDefinitionKind, Action: "UpdateReportDefinition"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: reportDefinitionKind, Action: "DeleteReportDefinition"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func reportDefinitionCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateReportDefinitionDetails", RequestName: "CreateReportDefinitionDetails", Contribution: "body"},
	}
}

func reportDefinitionGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ReportDefinitionId", RequestName: "reportDefinitionId", Contribution: "path", PreferResourceID: true},
	}
}

func reportDefinitionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
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
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "IsSeeded", RequestName: "isSeeded", Contribution: "query"},
		{FieldName: "DataSource", RequestName: "dataSource", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Category", RequestName: "category", Contribution: "query"},
	}
}

func reportDefinitionUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ReportDefinitionId", RequestName: "reportDefinitionId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateReportDefinitionDetails", RequestName: "UpdateReportDefinitionDetails", Contribution: "body"},
	}
}

func reportDefinitionDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ReportDefinitionId", RequestName: "reportDefinitionId", Contribution: "path", PreferResourceID: true},
	}
}

func buildReportDefinitionCreateBody(
	_ context.Context,
	resource *datasafev1beta1.ReportDefinition,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", reportDefinitionKind)
	}
	if err := validateReportDefinitionSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := datasafesdk.CreateReportDefinitionDetails{
		CompartmentId:  common.String(strings.TrimSpace(spec.CompartmentId)),
		DisplayName:    common.String(strings.TrimSpace(spec.DisplayName)),
		ParentId:       common.String(strings.TrimSpace(spec.ParentId)),
		ColumnInfo:     reportDefinitionColumnsFromSpec(spec.ColumnInfo),
		ColumnFilters:  reportDefinitionColumnFiltersFromSpec(spec.ColumnFilters),
		ColumnSortings: reportDefinitionColumnSortingsFromSpec(spec.ColumnSortings),
		Summary:        reportDefinitionSummariesFromSpec(spec.Summary),
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = reportDefinitionDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildReportDefinitionUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.ReportDefinition,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", reportDefinitionKind)
	}
	if err := validateReportDefinitionSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := reportDefinitionFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a %s body", reportDefinitionKind, reportDefinitionKind)
	}
	if err := validateReportDefinitionCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}

	body, updateNeeded := reportDefinitionMutableUpdateBody(resource.Spec, current)
	return body, updateNeeded, nil
}

func reportDefinitionMutableUpdateBody(
	spec datasafev1beta1.ReportDefinitionSpec,
	current datasafesdk.ReportDefinition,
) (datasafesdk.UpdateReportDefinitionDetails, bool) {
	body := datasafesdk.UpdateReportDefinitionDetails{
		DisplayName:    common.String(strings.TrimSpace(spec.DisplayName)),
		ColumnInfo:     reportDefinitionColumnsFromSpec(spec.ColumnInfo),
		ColumnFilters:  reportDefinitionColumnFiltersFromSpec(spec.ColumnFilters),
		ColumnSortings: reportDefinitionColumnSortingsFromSpec(spec.ColumnSortings),
		Summary:        reportDefinitionSummariesFromSpec(spec.Summary),
		Description:    common.String(spec.Description),
	}
	currentColumnInfo := reportDefinitionColumnsFromSpec(reportDefinitionAPIColumns(current.ColumnInfo))
	currentColumnFilters := reportDefinitionColumnFiltersFromSpec(reportDefinitionAPIColumnFilters(current.ColumnFilters))
	currentColumnSortings := reportDefinitionColumnSortingsFromSpec(reportDefinitionAPIColumnSortings(current.ColumnSortings))
	currentSummary := reportDefinitionSummariesFromSpec(reportDefinitionAPISummaries(current.Summary))

	updateNeeded := !reportDefinitionStringPtrEqual(current.DisplayName, spec.DisplayName)
	updateNeeded = !reportDefinitionJSONEqual(body.ColumnInfo, currentColumnInfo) || updateNeeded
	updateNeeded = !reportDefinitionJSONEqual(body.ColumnFilters, currentColumnFilters) || updateNeeded
	updateNeeded = !reportDefinitionJSONEqual(body.ColumnSortings, currentColumnSortings) || updateNeeded
	updateNeeded = !reportDefinitionJSONEqual(body.Summary, currentSummary) || updateNeeded
	updateNeeded = !reportDefinitionStringPtrEqual(current.Description, spec.Description) || updateNeeded
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = !maps.Equal(spec.FreeformTags, current.FreeformTags) || updateNeeded
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = reportDefinitionDefinedTagsFromSpec(spec.DefinedTags)
		updateNeeded = !reportDefinitionJSONEqual(body.DefinedTags, current.DefinedTags) || updateNeeded
	}
	return body, updateNeeded
}

func validateReportDefinitionSpec(spec datasafev1beta1.ReportDefinitionSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.ParentId) == "" {
		missing = append(missing, "parentId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", reportDefinitionKind, strings.Join(missing, ", "))
}

func resolveReportDefinitionIdentity(resource *datasafev1beta1.ReportDefinition) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", reportDefinitionKind)
	}
	return reportDefinitionIdentityFromResource(resource), nil
}

func reportDefinitionIdentityFromResource(resource *datasafev1beta1.ReportDefinition) reportDefinitionIdentity {
	if resource == nil {
		return reportDefinitionIdentity{}
	}
	return reportDefinitionIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
		parentID:      strings.TrimSpace(resource.Spec.ParentId),
	}
}

func recordReportDefinitionPathIdentity(resource *datasafev1beta1.ReportDefinition, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(reportDefinitionIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if typed.displayName != "" {
		resource.Status.DisplayName = typed.displayName
	}
	if typed.parentID != "" {
		resource.Status.ParentId = typed.parentID
	}
}

func lookupReportDefinitionExisting(
	list func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error),
	get func(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error),
) func(context.Context, *datasafev1beta1.ReportDefinition, any) (any, error) {
	return func(ctx context.Context, resource *datasafev1beta1.ReportDefinition, identity any) (any, error) {
		if list == nil || get == nil {
			return nil, nil
		}
		typed, ok := reportDefinitionLookupIdentity(identity)
		if !ok {
			return nil, nil
		}
		matches, err := findReportDefinitionMatches(ctx, resource, typed, list, get)
		if err != nil {
			return nil, err
		}
		return singleReportDefinitionLookupMatch(matches)
	}
}

func reportDefinitionLookupIdentity(identity any) (reportDefinitionIdentity, bool) {
	typed, ok := identity.(reportDefinitionIdentity)
	if !ok || !reportDefinitionIdentityReady(typed) {
		return reportDefinitionIdentity{}, false
	}
	return typed, true
}

func reportDefinitionIdentityReady(identity reportDefinitionIdentity) bool {
	return identity.compartmentID != "" && identity.displayName != "" && identity.parentID != ""
}

func findReportDefinitionMatches(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
	identity reportDefinitionIdentity,
	list func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error),
	get func(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error),
) ([]datasafesdk.GetReportDefinitionResponse, error) {
	response, err := listReportDefinitionCandidates(ctx, identity, list)
	if err != nil {
		return nil, err
	}
	if resource != nil {
		servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	}
	return collectReportDefinitionMatches(ctx, resource, identity, response.Items, get)
}

func listReportDefinitionCandidates(
	ctx context.Context,
	identity reportDefinitionIdentity,
	list func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error),
) (datasafesdk.ListReportDefinitionsResponse, error) {
	return list(ctx, datasafesdk.ListReportDefinitionsRequest{
		CompartmentId: common.String(identity.compartmentID),
		DisplayName:   common.String(identity.displayName),
		IsSeeded:      common.Bool(false),
	})
}

func collectReportDefinitionMatches(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
	identity reportDefinitionIdentity,
	items []datasafesdk.ReportDefinitionSummary,
	get func(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error),
) ([]datasafesdk.GetReportDefinitionResponse, error) {
	var matches []datasafesdk.GetReportDefinitionResponse
	for _, item := range items {
		current, matched, err := getReportDefinitionIfMatching(ctx, resource, identity, item, get)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, current)
		}
	}
	return matches, nil
}

func getReportDefinitionIfMatching(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
	identity reportDefinitionIdentity,
	item datasafesdk.ReportDefinitionSummary,
	get func(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error),
) (datasafesdk.GetReportDefinitionResponse, bool, error) {
	if !reportDefinitionSummaryMayMatchIdentity(item, identity) {
		return datasafesdk.GetReportDefinitionResponse{}, false, nil
	}
	current, err := get(ctx, datasafesdk.GetReportDefinitionRequest{
		ReportDefinitionId: item.Id,
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return datasafesdk.GetReportDefinitionResponse{}, false, nil
		}
		return datasafesdk.GetReportDefinitionResponse{}, false, err
	}
	if resource != nil {
		servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, current)
	}
	return current, reportDefinitionMatchesIdentity(current.ReportDefinition, identity), nil
}

func singleReportDefinitionLookupMatch(matches []datasafesdk.GetReportDefinitionResponse) (any, error) {
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%s lookup found multiple OCI report definitions matching compartmentId, displayName, and parentId", reportDefinitionKind)
	}
}

func reportDefinitionSummaryMayMatchIdentity(summary datasafesdk.ReportDefinitionSummary, identity reportDefinitionIdentity) bool {
	if !reportDefinitionStringPtrEqual(summary.CompartmentId, identity.compartmentID) {
		return false
	}
	if !reportDefinitionStringPtrEqual(summary.DisplayName, identity.displayName) {
		return false
	}
	state := strings.ToUpper(string(summary.LifecycleState))
	return state != string(datasafesdk.ReportDefinitionLifecycleStateDeleting) &&
		state != string(datasafesdk.ReportDefinitionLifecycleStateDeleted)
}

func reportDefinitionMatchesIdentity(current datasafesdk.ReportDefinition, identity reportDefinitionIdentity) bool {
	return reportDefinitionStringPtrEqual(current.CompartmentId, identity.compartmentID) &&
		reportDefinitionStringPtrEqual(current.DisplayName, identity.displayName) &&
		reportDefinitionStringPtrEqual(current.ParentId, identity.parentID)
}

func listReportDefinitionsAllPages(
	call func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error),
) func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error) {
		var combined datasafesdk.ListReportDefinitionsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
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

func validateReportDefinitionCreateOnlyDriftForResponse(
	resource *datasafev1beta1.ReportDefinition,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", reportDefinitionKind)
	}
	current, ok := reportDefinitionFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateReportDefinitionCreateOnlyDrift(resource.Spec, current)
}

func validateReportDefinitionCreateOnlyDrift(
	spec datasafev1beta1.ReportDefinitionSpec,
	current datasafesdk.ReportDefinition,
) error {
	checks := []struct {
		field   string
		current *string
		desired string
	}{
		{field: "compartmentId", current: current.CompartmentId, desired: spec.CompartmentId},
		{field: "parentId", current: current.ParentId, desired: spec.ParentId},
	}
	for _, check := range checks {
		if reportDefinitionStringValue(check.current) == "" || strings.TrimSpace(check.desired) == "" {
			continue
		}
		if !reportDefinitionStringPtrEqual(check.current, check.desired) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", reportDefinitionKind, check.field)
		}
	}
	return nil
}

func projectReportDefinitionStatus(resource *datasafev1beta1.ReportDefinition, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", reportDefinitionKind)
	}
	projected, ok := reportDefinitionStatusProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.ReportDefinitionStatus{
		OsokStatus:                   osokStatus,
		DisplayName:                  projected.DisplayName,
		Id:                           projected.Id,
		CompartmentId:                projected.CompartmentId,
		LifecycleState:               projected.LifecycleState,
		ParentId:                     projected.ParentId,
		Category:                     projected.Category,
		Description:                  projected.Description,
		DataSource:                   projected.DataSource,
		IsSeeded:                     projected.IsSeeded,
		DisplayOrder:                 projected.DisplayOrder,
		TimeCreated:                  projected.TimeCreated,
		TimeUpdated:                  projected.TimeUpdated,
		ScimFilter:                   projected.ScimFilter,
		ColumnInfo:                   reportDefinitionAPIColumns(projected.ColumnInfo),
		ColumnFilters:                reportDefinitionAPIColumnFilters(projected.ColumnFilters),
		ColumnSortings:               reportDefinitionAPIColumnSortings(projected.ColumnSortings),
		Summary:                      reportDefinitionAPISummaries(projected.Summary),
		Schedule:                     projected.Schedule,
		ScheduledReportMimeType:      projected.ScheduledReportMimeType,
		ScheduledReportRowLimit:      projected.ScheduledReportRowLimit,
		ScheduledReportName:          projected.ScheduledReportName,
		ScheduledReportCompartmentId: projected.ScheduledReportCompartmentId,
		RecordTimeSpan:               projected.RecordTimeSpan,
		ComplianceStandards:          reportDefinitionStringSlice(projected.ComplianceStandards),
		LifecycleDetails:             projected.LifecycleDetails,
		FreeformTags:                 reportDefinitionStringMap(projected.FreeformTags),
		DefinedTags:                  reportDefinitionSharedTags(projected.DefinedTags),
		SystemTags:                   reportDefinitionSharedTags(projected.SystemTags),
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

type reportDefinitionStatusProjection struct {
	DisplayName                  string
	Id                           string
	CompartmentId                string
	LifecycleState               string
	ParentId                     string
	Category                     string
	Description                  string
	DataSource                   string
	IsSeeded                     bool
	DisplayOrder                 int
	TimeCreated                  string
	TimeUpdated                  string
	ScimFilter                   string
	ColumnInfo                   []datasafesdk.Column
	ColumnFilters                []datasafesdk.ColumnFilter
	ColumnSortings               []datasafesdk.ColumnSorting
	Summary                      []datasafesdk.Summary
	Schedule                     string
	ScheduledReportMimeType      string
	ScheduledReportRowLimit      int
	ScheduledReportName          string
	ScheduledReportCompartmentId string
	RecordTimeSpan               string
	ComplianceStandards          []string
	LifecycleDetails             string
	FreeformTags                 map[string]string
	DefinedTags                  map[string]map[string]interface{}
	SystemTags                   map[string]map[string]interface{}
}

func reportDefinitionStatusProjectionFromResponse(response any) (reportDefinitionStatusProjection, bool) {
	if current, ok := reportDefinitionFromResponse(response); ok {
		return reportDefinitionStatusProjectionFromSDK(current), true
	}
	if summary, ok := reportDefinitionSummaryFromResponse(response); ok {
		return reportDefinitionStatusProjectionFromSummary(summary), true
	}
	return reportDefinitionStatusProjection{}, false
}

func reportDefinitionFromResponse(response any) (datasafesdk.ReportDefinition, bool) {
	switch current := response.(type) {
	case datasafesdk.GetReportDefinitionResponse:
		return current.ReportDefinition, true
	case *datasafesdk.GetReportDefinitionResponse:
		if current == nil {
			return datasafesdk.ReportDefinition{}, false
		}
		return current.ReportDefinition, true
	case datasafesdk.CreateReportDefinitionResponse:
		return current.ReportDefinition, true
	case *datasafesdk.CreateReportDefinitionResponse:
		if current == nil {
			return datasafesdk.ReportDefinition{}, false
		}
		return current.ReportDefinition, true
	case datasafesdk.ReportDefinition:
		return current, true
	case *datasafesdk.ReportDefinition:
		if current == nil {
			return datasafesdk.ReportDefinition{}, false
		}
		return *current, true
	default:
		return datasafesdk.ReportDefinition{}, false
	}
}

func reportDefinitionSummaryFromResponse(response any) (datasafesdk.ReportDefinitionSummary, bool) {
	switch current := response.(type) {
	case datasafesdk.ReportDefinitionSummary:
		return current, true
	case *datasafesdk.ReportDefinitionSummary:
		if current == nil {
			return datasafesdk.ReportDefinitionSummary{}, false
		}
		return *current, true
	default:
		return datasafesdk.ReportDefinitionSummary{}, false
	}
}

func reportDefinitionStatusProjectionFromSDK(current datasafesdk.ReportDefinition) reportDefinitionStatusProjection {
	return reportDefinitionStatusProjection{
		DisplayName:                  reportDefinitionStringValue(current.DisplayName),
		Id:                           reportDefinitionStringValue(current.Id),
		CompartmentId:                reportDefinitionStringValue(current.CompartmentId),
		LifecycleState:               string(current.LifecycleState),
		ParentId:                     reportDefinitionStringValue(current.ParentId),
		Category:                     string(current.Category),
		Description:                  reportDefinitionStringValue(current.Description),
		DataSource:                   string(current.DataSource),
		IsSeeded:                     reportDefinitionBoolValue(current.IsSeeded),
		DisplayOrder:                 reportDefinitionIntValue(current.DisplayOrder),
		TimeCreated:                  reportDefinitionSDKTimeString(current.TimeCreated),
		TimeUpdated:                  reportDefinitionSDKTimeString(current.TimeUpdated),
		ScimFilter:                   reportDefinitionStringValue(current.ScimFilter),
		ColumnInfo:                   reportDefinitionSDKColumns(current.ColumnInfo),
		ColumnFilters:                reportDefinitionSDKColumnFilters(current.ColumnFilters),
		ColumnSortings:               reportDefinitionSDKColumnSortings(current.ColumnSortings),
		Summary:                      reportDefinitionSDKSummaries(current.Summary),
		Schedule:                     reportDefinitionStringValue(current.Schedule),
		ScheduledReportMimeType:      string(current.ScheduledReportMimeType),
		ScheduledReportRowLimit:      reportDefinitionIntValue(current.ScheduledReportRowLimit),
		ScheduledReportName:          reportDefinitionStringValue(current.ScheduledReportName),
		ScheduledReportCompartmentId: reportDefinitionStringValue(current.ScheduledReportCompartmentId),
		RecordTimeSpan:               reportDefinitionStringValue(current.RecordTimeSpan),
		ComplianceStandards:          reportDefinitionStringSlice(current.ComplianceStandards),
		LifecycleDetails:             reportDefinitionStringValue(current.LifecycleDetails),
		FreeformTags:                 reportDefinitionStringMap(current.FreeformTags),
		DefinedTags:                  reportDefinitionDefinedTagsFromSDK(current.DefinedTags),
		SystemTags:                   reportDefinitionDefinedTagsFromSDK(current.SystemTags),
	}
}

func reportDefinitionStatusProjectionFromSummary(current datasafesdk.ReportDefinitionSummary) reportDefinitionStatusProjection {
	return reportDefinitionStatusProjection{
		DisplayName:         reportDefinitionStringValue(current.DisplayName),
		Id:                  reportDefinitionStringValue(current.Id),
		CompartmentId:       reportDefinitionStringValue(current.CompartmentId),
		LifecycleState:      string(current.LifecycleState),
		Category:            string(current.Category),
		Description:         reportDefinitionStringValue(current.Description),
		DataSource:          string(current.DataSource),
		IsSeeded:            reportDefinitionBoolValue(current.IsSeeded),
		DisplayOrder:        reportDefinitionIntValue(current.DisplayOrder),
		TimeCreated:         reportDefinitionSDKTimeString(current.TimeCreated),
		TimeUpdated:         reportDefinitionSDKTimeString(current.TimeUpdated),
		Schedule:            reportDefinitionStringValue(current.Schedule),
		ComplianceStandards: reportDefinitionStringSlice(current.ComplianceStandards),
		FreeformTags:        reportDefinitionStringMap(current.FreeformTags),
		DefinedTags:         reportDefinitionDefinedTagsFromSDK(current.DefinedTags),
	}
}

func (c reportDefinitionRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", reportDefinitionKind)
	}
	if err := validateReportDefinitionCreateOrUpdateIdentity(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, markReportDefinitionFailed(resource, err)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func validateReportDefinitionCreateOrUpdateIdentity(resource *datasafev1beta1.ReportDefinition) error {
	if trackedReportDefinitionID(resource) == "" || resource == nil {
		return nil
	}
	identity := reportDefinitionIdentityFromResource(resource)
	checks := []struct {
		field   string
		tracked string
		desired string
	}{
		{field: "compartmentId", tracked: resource.Status.CompartmentId, desired: identity.compartmentID},
		{field: "parentId", tracked: resource.Status.ParentId, desired: identity.parentID},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.tracked) == "" || strings.TrimSpace(check.desired) == "" {
			continue
		}
		if strings.TrimSpace(check.tracked) != strings.TrimSpace(check.desired) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", reportDefinitionKind, check.field)
		}
	}
	return nil
}

func markReportDefinitionFailed(resource *datasafev1beta1.ReportDefinition, err error) error {
	if resource == nil || err == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		status.Async.Current = &current
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
	return err
}

func (c reportDefinitionRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.ReportDefinition) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", reportDefinitionKind)
	}
	if deleted, handled, err := c.confirmUntrackedReportDefinitionDelete(ctx, resource); handled || err != nil {
		return deleted, err
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c reportDefinitionRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
) error {
	if resource == nil {
		return nil
	}
	currentID := trackedReportDefinitionID(resource)
	if currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return nil
}

func (c reportDefinitionRuntimeClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, datasafesdk.GetReportDefinitionRequest{ReportDefinitionId: common.String(currentID)})
	return reportDefinitionAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c reportDefinitionRuntimeClient) confirmUntrackedReportDefinitionDelete(
	ctx context.Context,
	resource *datasafev1beta1.ReportDefinition,
) (bool, bool, error) {
	if resource == nil || trackedReportDefinitionID(resource) != "" || c.list == nil || c.get == nil {
		return false, false, nil
	}
	identity := reportDefinitionIdentityFromResource(resource)
	if !reportDefinitionIdentityReady(identity) {
		return false, false, nil
	}

	response, err := lookupReportDefinitionExisting(c.list, c.get)(ctx, resource, identity)
	if err != nil {
		return false, true, handleReportDefinitionDeleteError(resource, err)
	}
	if response == nil {
		markReportDefinitionDeleted(resource, "OCI resource no longer exists")
		return true, true, nil
	}
	if err := projectReportDefinitionStatus(resource, response); err != nil {
		return false, true, err
	}
	return false, false, nil
}

func confirmReportDefinitionDeleteRead(
	list func(context.Context, datasafesdk.ListReportDefinitionsRequest) (datasafesdk.ListReportDefinitionsResponse, error),
	get func(context.Context, datasafesdk.GetReportDefinitionRequest) (datasafesdk.GetReportDefinitionResponse, error),
) func(context.Context, *datasafev1beta1.ReportDefinition, string) (any, error) {
	lookupExisting := lookupReportDefinitionExisting(list, get)
	return func(ctx context.Context, resource *datasafev1beta1.ReportDefinition, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID != "" {
			if get == nil {
				return nil, fmt.Errorf("%s delete confirmation requires a Get operation", reportDefinitionKind)
			}
			return get(ctx, datasafesdk.GetReportDefinitionRequest{ReportDefinitionId: common.String(currentID)})
		}

		response, err := lookupExisting(ctx, resource, reportDefinitionIdentityFromResource(resource))
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, reportDefinitionNotFoundError("delete confirmation found no matching OCI report definition")
		}
		return response, nil
	}
}

func reportDefinitionNotFoundError(message string) error {
	return errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    message,
	}
}

func handleReportDefinitionDeleteError(resource *datasafev1beta1.ReportDefinition, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := reportDefinitionAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func reportDefinitionAmbiguousDeleteError(
	resource *datasafev1beta1.ReportDefinition,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return reportDefinitionAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", reportDefinitionKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func applyReportDefinitionDeleteOutcome(
	resource *datasafev1beta1.ReportDefinition,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := reportDefinitionLifecycleState(response)
	if state == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if strings.EqualFold(state, string(datasafesdk.ReportDefinitionLifecycleStateDeleted)) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markReportDefinitionTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markReportDefinitionDeleted(resource *datasafev1beta1.ReportDefinition, message string) {
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

func markReportDefinitionTerminating(resource *datasafev1beta1.ReportDefinition, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = "OCI resource delete is in progress"
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         status.Message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, loggerutil.OSOKLogger{})
}

func reportDefinitionLifecycleState(response any) string {
	if current, ok := reportDefinitionFromResponse(response); ok {
		return strings.ToUpper(string(current.LifecycleState))
	}
	if summary, ok := reportDefinitionSummaryFromResponse(response); ok {
		return strings.ToUpper(string(summary.LifecycleState))
	}
	return ""
}

func clearTrackedReportDefinitionIdentity(resource *datasafev1beta1.ReportDefinition) {
	if resource == nil {
		return
	}
	status := resource.Status.OsokStatus
	status.Ocid = ""
	resource.Status = datasafev1beta1.ReportDefinitionStatus{OsokStatus: status}
}

func trackedReportDefinitionID(resource *datasafev1beta1.ReportDefinition) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func reportDefinitionColumnsFromSpec(source []datasafev1beta1.ReportDefinitionColumnInfo) []datasafesdk.Column {
	if len(source) == 0 {
		return nil
	}
	columns := make([]datasafesdk.Column, 0, len(source))
	for _, item := range source {
		columns = append(columns, datasafesdk.Column{
			DisplayName:         common.String(strings.TrimSpace(item.DisplayName)),
			FieldName:           common.String(strings.TrimSpace(item.FieldName)),
			IsHidden:            common.Bool(item.IsHidden),
			DisplayOrder:        common.Int(item.DisplayOrder),
			DataType:            reportDefinitionOptionalString(item.DataType),
			IsVirtual:           common.Bool(item.IsVirtual),
			ApplicableOperators: reportDefinitionColumnOperators(item.ApplicableOperators),
		})
	}
	return columns
}

func reportDefinitionColumnFiltersFromSpec(source []datasafev1beta1.ReportDefinitionColumnFilter) []datasafesdk.ColumnFilter {
	if len(source) == 0 {
		return nil
	}
	filters := make([]datasafesdk.ColumnFilter, 0, len(source))
	for _, item := range source {
		filters = append(filters, datasafesdk.ColumnFilter{
			FieldName:   common.String(strings.TrimSpace(item.FieldName)),
			Operator:    datasafesdk.ColumnFilterOperatorEnum(strings.TrimSpace(item.Operator)),
			Expressions: reportDefinitionStringSlice(item.Expressions),
			IsEnabled:   common.Bool(item.IsEnabled),
			IsHidden:    common.Bool(item.IsHidden),
		})
	}
	return filters
}

func reportDefinitionColumnSortingsFromSpec(source []datasafev1beta1.ReportDefinitionColumnSorting) []datasafesdk.ColumnSorting {
	if len(source) == 0 {
		return nil
	}
	sortings := make([]datasafesdk.ColumnSorting, 0, len(source))
	for _, item := range source {
		sortings = append(sortings, datasafesdk.ColumnSorting{
			FieldName:    common.String(strings.TrimSpace(item.FieldName)),
			IsAscending:  common.Bool(item.IsAscending),
			SortingOrder: common.Int(item.SortingOrder),
		})
	}
	return sortings
}

func reportDefinitionSummariesFromSpec(source []datasafev1beta1.ReportDefinitionSummary) []datasafesdk.Summary {
	if len(source) == 0 {
		return nil
	}
	summaries := make([]datasafesdk.Summary, 0, len(source))
	for _, item := range source {
		summaries = append(summaries, datasafesdk.Summary{
			Name:             common.String(strings.TrimSpace(item.Name)),
			DisplayOrder:     common.Int(item.DisplayOrder),
			IsHidden:         common.Bool(item.IsHidden),
			GroupByFieldName: reportDefinitionOptionalString(item.GroupByFieldName),
			CountOf:          reportDefinitionOptionalString(item.CountOf),
			ScimFilter:       reportDefinitionOptionalString(item.ScimFilter),
		})
	}
	return summaries
}

func reportDefinitionAPIColumns(source []datasafesdk.Column) []datasafev1beta1.ReportDefinitionColumnInfo {
	if len(source) == 0 {
		return nil
	}
	columns := make([]datasafev1beta1.ReportDefinitionColumnInfo, 0, len(source))
	for _, item := range source {
		columns = append(columns, datasafev1beta1.ReportDefinitionColumnInfo{
			DisplayName:         reportDefinitionStringValue(item.DisplayName),
			FieldName:           reportDefinitionStringValue(item.FieldName),
			IsHidden:            reportDefinitionBoolValue(item.IsHidden),
			DisplayOrder:        reportDefinitionIntValue(item.DisplayOrder),
			DataType:            reportDefinitionStringValue(item.DataType),
			IsVirtual:           reportDefinitionBoolValue(item.IsVirtual),
			ApplicableOperators: reportDefinitionColumnOperatorStrings(item.ApplicableOperators),
		})
	}
	return columns
}

func reportDefinitionAPIColumnFilters(source []datasafesdk.ColumnFilter) []datasafev1beta1.ReportDefinitionColumnFilter {
	if len(source) == 0 {
		return nil
	}
	filters := make([]datasafev1beta1.ReportDefinitionColumnFilter, 0, len(source))
	for _, item := range source {
		filters = append(filters, datasafev1beta1.ReportDefinitionColumnFilter{
			FieldName:   reportDefinitionStringValue(item.FieldName),
			Operator:    string(item.Operator),
			Expressions: reportDefinitionStringSlice(item.Expressions),
			IsEnabled:   reportDefinitionBoolValue(item.IsEnabled),
			IsHidden:    reportDefinitionBoolValue(item.IsHidden),
		})
	}
	return filters
}

func reportDefinitionAPIColumnSortings(source []datasafesdk.ColumnSorting) []datasafev1beta1.ReportDefinitionColumnSorting {
	if len(source) == 0 {
		return nil
	}
	sortings := make([]datasafev1beta1.ReportDefinitionColumnSorting, 0, len(source))
	for _, item := range source {
		sortings = append(sortings, datasafev1beta1.ReportDefinitionColumnSorting{
			FieldName:    reportDefinitionStringValue(item.FieldName),
			IsAscending:  reportDefinitionBoolValue(item.IsAscending),
			SortingOrder: reportDefinitionIntValue(item.SortingOrder),
		})
	}
	return sortings
}

func reportDefinitionAPISummaries(source []datasafesdk.Summary) []datasafev1beta1.ReportDefinitionSummary {
	if len(source) == 0 {
		return nil
	}
	summaries := make([]datasafev1beta1.ReportDefinitionSummary, 0, len(source))
	for _, item := range source {
		summaries = append(summaries, datasafev1beta1.ReportDefinitionSummary{
			Name:             reportDefinitionStringValue(item.Name),
			DisplayOrder:     reportDefinitionIntValue(item.DisplayOrder),
			IsHidden:         reportDefinitionBoolValue(item.IsHidden),
			GroupByFieldName: reportDefinitionStringValue(item.GroupByFieldName),
			CountOf:          reportDefinitionStringValue(item.CountOf),
			ScimFilter:       reportDefinitionStringValue(item.ScimFilter),
		})
	}
	return summaries
}

func reportDefinitionSDKColumns(source []datasafesdk.Column) []datasafesdk.Column {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]datasafesdk.Column, 0, len(source))
	for _, item := range source {
		cloned = append(cloned, datasafesdk.Column{
			DisplayName:         reportDefinitionOptionalString(reportDefinitionStringValue(item.DisplayName)),
			FieldName:           reportDefinitionOptionalString(reportDefinitionStringValue(item.FieldName)),
			IsHidden:            common.Bool(reportDefinitionBoolValue(item.IsHidden)),
			DisplayOrder:        common.Int(reportDefinitionIntValue(item.DisplayOrder)),
			DataType:            reportDefinitionOptionalString(reportDefinitionStringValue(item.DataType)),
			IsVirtual:           common.Bool(reportDefinitionBoolValue(item.IsVirtual)),
			ApplicableOperators: append([]datasafesdk.ColumnApplicableOperatorsEnum(nil), item.ApplicableOperators...),
		})
	}
	return cloned
}

func reportDefinitionSDKColumnFilters(source []datasafesdk.ColumnFilter) []datasafesdk.ColumnFilter {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]datasafesdk.ColumnFilter, 0, len(source))
	for _, item := range source {
		cloned = append(cloned, datasafesdk.ColumnFilter{
			FieldName:   reportDefinitionOptionalString(reportDefinitionStringValue(item.FieldName)),
			Operator:    item.Operator,
			Expressions: reportDefinitionStringSlice(item.Expressions),
			IsEnabled:   common.Bool(reportDefinitionBoolValue(item.IsEnabled)),
			IsHidden:    common.Bool(reportDefinitionBoolValue(item.IsHidden)),
		})
	}
	return cloned
}

func reportDefinitionSDKColumnSortings(source []datasafesdk.ColumnSorting) []datasafesdk.ColumnSorting {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]datasafesdk.ColumnSorting, 0, len(source))
	for _, item := range source {
		cloned = append(cloned, datasafesdk.ColumnSorting{
			FieldName:    reportDefinitionOptionalString(reportDefinitionStringValue(item.FieldName)),
			IsAscending:  common.Bool(reportDefinitionBoolValue(item.IsAscending)),
			SortingOrder: common.Int(reportDefinitionIntValue(item.SortingOrder)),
		})
	}
	return cloned
}

func reportDefinitionSDKSummaries(source []datasafesdk.Summary) []datasafesdk.Summary {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]datasafesdk.Summary, 0, len(source))
	for _, item := range source {
		cloned = append(cloned, datasafesdk.Summary{
			Name:             reportDefinitionOptionalString(reportDefinitionStringValue(item.Name)),
			DisplayOrder:     common.Int(reportDefinitionIntValue(item.DisplayOrder)),
			IsHidden:         common.Bool(reportDefinitionBoolValue(item.IsHidden)),
			GroupByFieldName: reportDefinitionOptionalString(reportDefinitionStringValue(item.GroupByFieldName)),
			CountOf:          reportDefinitionOptionalString(reportDefinitionStringValue(item.CountOf)),
			ScimFilter:       reportDefinitionOptionalString(reportDefinitionStringValue(item.ScimFilter)),
		})
	}
	return cloned
}

func reportDefinitionColumnOperators(source []string) []datasafesdk.ColumnApplicableOperatorsEnum {
	if len(source) == 0 {
		return nil
	}
	operators := make([]datasafesdk.ColumnApplicableOperatorsEnum, 0, len(source))
	for _, item := range source {
		if value := strings.TrimSpace(item); value != "" {
			operators = append(operators, datasafesdk.ColumnApplicableOperatorsEnum(value))
		}
	}
	return operators
}

func reportDefinitionColumnOperatorStrings(source []datasafesdk.ColumnApplicableOperatorsEnum) []string {
	if len(source) == 0 {
		return nil
	}
	operators := make([]string, 0, len(source))
	for _, item := range source {
		if value := strings.TrimSpace(string(item)); value != "" {
			operators = append(operators, value)
		}
	}
	return operators
}

func reportDefinitionOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func reportDefinitionStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(reportDefinitionStringValue(current)) == strings.TrimSpace(desired)
}

func reportDefinitionStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func reportDefinitionBoolValue(value *bool) bool {
	return value != nil && *value
}

func reportDefinitionIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func reportDefinitionSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func reportDefinitionStringSlice(source []string) []string {
	if source == nil {
		return nil
	}
	cloned := make([]string, len(source))
	copy(cloned, source)
	return cloned
}

func reportDefinitionStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	return maps.Clone(source)
}

func reportDefinitionDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func reportDefinitionDefinedTagsFromSDK(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func reportDefinitionSharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		converted[namespace] = make(shared.MapValue, len(values))
		for key, value := range values {
			converted[namespace][key] = fmt.Sprint(value)
		}
	}
	return converted
}

func reportDefinitionJSONEqual(left any, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftBytes) == string(rightBytes)
}

var _ interface{ GetOpcRequestID() string } = reportDefinitionAmbiguousNotFoundError{}
