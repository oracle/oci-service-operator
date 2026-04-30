/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package baselineablemetric

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type baselineableMetricOCIClient interface {
	CreateBaselineableMetric(context.Context, stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error)
	GetBaselineableMetric(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error)
	ListBaselineableMetrics(context.Context, stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error)
	UpdateBaselineableMetric(context.Context, stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error)
	DeleteBaselineableMetric(context.Context, stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error)
}

type ambiguousBaselineableMetricNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousBaselineableMetricNotFoundError) Error() string {
	return e.message
}

func (e ambiguousBaselineableMetricNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type baselineableMetricRuntimeClient struct {
	delegate BaselineableMetricServiceClient
	get      func(context.Context, stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error)
}

type baselineableMetricIdentity struct {
	id string
}

func init() {
	registerBaselineableMetricRuntimeHooksMutator(func(manager *BaselineableMetricServiceManager, hooks *BaselineableMetricRuntimeHooks) {
		client, initErr := newBaselineableMetricSDKClient(manager)
		applyBaselineableMetricRuntimeHooks(hooks, client, initErr)
	})
}

func newBaselineableMetricSDKClient(manager *BaselineableMetricServiceManager) (baselineableMetricOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("BaselineableMetric service manager is nil")
	}
	client, err := stackmonitoringsdk.NewStackMonitoringClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyBaselineableMetricRuntimeHooks(
	hooks *BaselineableMetricRuntimeHooks,
	client baselineableMetricOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = baselineableMetricRuntimeSemantics()
	hooks.BuildCreateBody = buildBaselineableMetricCreateBody
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *stackmonitoringv1beta1.BaselineableMetric,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildBaselineableMetricUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardBaselineableMetricExistingBeforeCreate
	hooks.Create.Fields = baselineableMetricCreateFields()
	hooks.Create.Call = func(ctx context.Context, request stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
		if err := baselineableMetricClientReady(client, initErr); err != nil {
			return stackmonitoringsdk.CreateBaselineableMetricResponse{}, err
		}
		return client.CreateBaselineableMetric(ctx, request)
	}
	hooks.Get.Fields = baselineableMetricGetFields()
	hooks.Get.Call = func(ctx context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
		if err := baselineableMetricClientReady(client, initErr); err != nil {
			return stackmonitoringsdk.GetBaselineableMetricResponse{}, err
		}
		response, err := client.GetBaselineableMetric(ctx, request)
		return response, conservativeBaselineableMetricNotFoundError(err, "read")
	}
	hooks.List.Fields = baselineableMetricListFields()
	hooks.List.Call = func(ctx context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
		return listBaselineableMetricPages(ctx, client, initErr, request)
	}
	hooks.Identity.Resolve = resolveBaselineableMetricIdentity
	hooks.Identity.LookupExisting = lookupBaselineableMetricExistingByID(hooks.List.Call)
	hooks.Update.Fields = baselineableMetricUpdateFields()
	hooks.Update.Call = func(ctx context.Context, request stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error) {
		if err := baselineableMetricClientReady(client, initErr); err != nil {
			return stackmonitoringsdk.UpdateBaselineableMetricResponse{}, err
		}
		return client.UpdateBaselineableMetric(ctx, request)
	}
	hooks.Delete.Fields = baselineableMetricDeleteFields()
	hooks.Delete.Call = func(ctx context.Context, request stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
		if err := baselineableMetricClientReady(client, initErr); err != nil {
			return stackmonitoringsdk.DeleteBaselineableMetricResponse{}, err
		}
		response, err := client.DeleteBaselineableMetric(ctx, request)
		return response, conservativeBaselineableMetricNotFoundError(err, "delete")
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedBaselineableMetricIdentity
	hooks.StatusHooks.ProjectStatus = projectBaselineableMetricStatus
	hooks.ParityHooks.NormalizeDesiredState = normalizeBaselineableMetricDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateBaselineableMetricCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleBaselineableMetricDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate BaselineableMetricServiceClient) BaselineableMetricServiceClient {
		return baselineableMetricRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
		}
	})
}

func newBaselineableMetricServiceClientWithOCIClient(client baselineableMetricOCIClient) BaselineableMetricServiceClient {
	manager := &BaselineableMetricServiceManager{}
	hooks := newBaselineableMetricRuntimeHooksWithOCIClient(client)
	applyBaselineableMetricRuntimeHooks(&hooks, client, nil)
	delegate := defaultBaselineableMetricServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.BaselineableMetric](
			buildBaselineableMetricGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapBaselineableMetricGeneratedClient(hooks, delegate)
}

func newBaselineableMetricRuntimeHooksWithOCIClient(client baselineableMetricOCIClient) BaselineableMetricRuntimeHooks {
	return BaselineableMetricRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*stackmonitoringv1beta1.BaselineableMetric]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*stackmonitoringv1beta1.BaselineableMetric]{},
		StatusHooks:     generatedruntime.StatusHooks[*stackmonitoringv1beta1.BaselineableMetric]{},
		ParityHooks:     generatedruntime.ParityHooks[*stackmonitoringv1beta1.BaselineableMetric]{},
		Async:           generatedruntime.AsyncHooks[*stackmonitoringv1beta1.BaselineableMetric]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*stackmonitoringv1beta1.BaselineableMetric]{},
		Create: runtimeOperationHooks[stackmonitoringsdk.CreateBaselineableMetricRequest, stackmonitoringsdk.CreateBaselineableMetricResponse]{
			Fields: baselineableMetricCreateFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.CreateBaselineableMetricRequest) (stackmonitoringsdk.CreateBaselineableMetricResponse, error) {
				return client.CreateBaselineableMetric(ctx, request)
			},
		},
		Get: runtimeOperationHooks[stackmonitoringsdk.GetBaselineableMetricRequest, stackmonitoringsdk.GetBaselineableMetricResponse]{
			Fields: baselineableMetricGetFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.GetBaselineableMetricRequest) (stackmonitoringsdk.GetBaselineableMetricResponse, error) {
				return client.GetBaselineableMetric(ctx, request)
			},
		},
		List: runtimeOperationHooks[stackmonitoringsdk.ListBaselineableMetricsRequest, stackmonitoringsdk.ListBaselineableMetricsResponse]{
			Fields: baselineableMetricListFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
				return client.ListBaselineableMetrics(ctx, request)
			},
		},
		Update: runtimeOperationHooks[stackmonitoringsdk.UpdateBaselineableMetricRequest, stackmonitoringsdk.UpdateBaselineableMetricResponse]{
			Fields: baselineableMetricUpdateFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.UpdateBaselineableMetricRequest) (stackmonitoringsdk.UpdateBaselineableMetricResponse, error) {
				return client.UpdateBaselineableMetric(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[stackmonitoringsdk.DeleteBaselineableMetricRequest, stackmonitoringsdk.DeleteBaselineableMetricResponse]{
			Fields: baselineableMetricDeleteFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.DeleteBaselineableMetricRequest) (stackmonitoringsdk.DeleteBaselineableMetricResponse, error) {
				return client.DeleteBaselineableMetric(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BaselineableMetricServiceClient) BaselineableMetricServiceClient{},
	}
}

func baselineableMetricRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "stackmonitoring",
		FormalSlug:    "baselineablemetric",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(stackmonitoringsdk.BaselineableMetricLifeCycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(stackmonitoringsdk.BaselineableMetricLifeCycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"compartmentId",
				"column",
				"namespace",
				"name",
				"resourceGroup",
				"resourceType",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"id",
				"lifecycleState",
				"tenancyId",
				"systemTags",
			},
			ConflictsWith: map[string][]string{},
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

func baselineableMetricCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateBaselineableMetricDetails", RequestName: "CreateBaselineableMetricDetails", Contribution: "body"},
	}
}

func baselineableMetricGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "BaselineableMetricId", RequestName: "baselineableMetricId", Contribution: "path", PreferResourceID: true},
	}
}

func baselineableMetricListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "BaselineableMetricId", RequestName: "baselineableMetricId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func baselineableMetricUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "BaselineableMetricId", RequestName: "baselineableMetricId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateBaselineableMetricDetails", RequestName: "UpdateBaselineableMetricDetails", Contribution: "body"},
	}
}

func baselineableMetricDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "BaselineableMetricId", RequestName: "baselineableMetricId", Contribution: "path", PreferResourceID: true},
	}
}

func buildBaselineableMetricCreateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.BaselineableMetric,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("BaselineableMetric resource is nil")
	}
	if err := validateBaselineableMetricRequiredSpec(resource.Spec); err != nil {
		return nil, err
	}
	if resource.Spec.IsOutOfBox {
		return nil, fmt.Errorf("BaselineableMetric spec.isOutOfBox=true is not supported during create because OCI CreateBaselineableMetricDetails does not expose isOutOfBox")
	}

	body := stackmonitoringsdk.CreateBaselineableMetricDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Column:        common.String(strings.TrimSpace(resource.Spec.Column)),
		Namespace:     common.String(strings.TrimSpace(resource.Spec.Namespace)),
	}
	if value := strings.TrimSpace(resource.Spec.Name); value != "" {
		body.Name = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.ResourceGroup); value != "" {
		body.ResourceGroup = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.ResourceType); value != "" {
		body.ResourceType = common.String(value)
	}
	return body, nil
}

func buildBaselineableMetricUpdateBody(
	resource *stackmonitoringv1beta1.BaselineableMetric,
	currentResponse any,
) (stackmonitoringsdk.UpdateBaselineableMetricDetails, bool, error) {
	if resource == nil {
		return stackmonitoringsdk.UpdateBaselineableMetricDetails{}, false, fmt.Errorf("BaselineableMetric resource is nil")
	}
	if err := validateBaselineableMetricRequiredSpec(resource.Spec); err != nil {
		return stackmonitoringsdk.UpdateBaselineableMetricDetails{}, false, err
	}
	current, err := baselineableMetricResponseBody(currentResponse)
	if err != nil {
		return stackmonitoringsdk.UpdateBaselineableMetricDetails{}, false, err
	}
	if err := validateBaselineableMetricCreateOnlyDrift(resource, current); err != nil {
		return stackmonitoringsdk.UpdateBaselineableMetricDetails{}, false, err
	}

	body := baselineableMetricUpdateBodyFromCurrent(current)
	updateNeeded := applyBaselineableMetricRequiredStringUpdates(&body, resource.Spec, current)
	updateNeeded = applyBaselineableMetricOptionalStringUpdates(&body, resource.Spec, current) || updateNeeded
	updateNeeded = applyBaselineableMetricTagUpdates(&body, resource.Spec, current) || updateNeeded
	return body, updateNeeded, nil
}

func baselineableMetricResponseBody(response any) (stackmonitoringsdk.BaselineableMetric, error) {
	current, ok := baselineableMetricFromResponse(response)
	if !ok {
		return stackmonitoringsdk.BaselineableMetric{}, fmt.Errorf("current BaselineableMetric response does not expose a BaselineableMetric body")
	}
	return current, nil
}

func applyBaselineableMetricRequiredStringUpdates(
	body *stackmonitoringsdk.UpdateBaselineableMetricDetails,
	spec stackmonitoringv1beta1.BaselineableMetricSpec,
	current stackmonitoringsdk.BaselineableMetric,
) bool {
	updateNeeded := false
	if changed := applyBaselineableMetricStringUpdate(&body.CompartmentId, spec.CompartmentId, current.CompartmentId); changed {
		updateNeeded = true
	}
	if changed := applyBaselineableMetricStringUpdate(&body.Column, spec.Column, current.Column); changed {
		updateNeeded = true
	}
	if changed := applyBaselineableMetricStringUpdate(&body.Namespace, spec.Namespace, current.Namespace); changed {
		updateNeeded = true
	}
	return updateNeeded
}

func applyBaselineableMetricOptionalStringUpdates(
	body *stackmonitoringsdk.UpdateBaselineableMetricDetails,
	spec stackmonitoringv1beta1.BaselineableMetricSpec,
	current stackmonitoringsdk.BaselineableMetric,
) bool {
	updateNeeded := false
	if changed := applyOptionalBaselineableMetricStringUpdate(&body.Name, spec.Name, current.Name); changed {
		updateNeeded = true
	}
	if changed := applyOptionalBaselineableMetricStringUpdate(&body.ResourceGroup, spec.ResourceGroup, current.ResourceGroup); changed {
		updateNeeded = true
	}
	if changed := applyOptionalBaselineableMetricStringUpdate(&body.ResourceType, spec.ResourceType, current.ResourceType); changed {
		updateNeeded = true
	}
	return updateNeeded
}

func applyBaselineableMetricTagUpdates(
	body *stackmonitoringsdk.UpdateBaselineableMetricDetails,
	spec stackmonitoringv1beta1.BaselineableMetricSpec,
	current stackmonitoringsdk.BaselineableMetric,
) bool {
	updateNeeded := false
	if desired, changed := desiredBaselineableMetricFreeformTags(spec.FreeformTags, current.FreeformTags); changed {
		body.FreeformTags = desired
		updateNeeded = true
	}
	if desired, changed := desiredBaselineableMetricDefinedTags(spec.DefinedTags, current.DefinedTags); changed {
		body.DefinedTags = desired
		updateNeeded = true
	}
	return updateNeeded
}

func baselineableMetricUpdateBodyFromCurrent(current stackmonitoringsdk.BaselineableMetric) stackmonitoringsdk.UpdateBaselineableMetricDetails {
	return stackmonitoringsdk.UpdateBaselineableMetricDetails{
		Id:             current.Id,
		Name:           current.Name,
		Column:         current.Column,
		Namespace:      current.Namespace,
		ResourceGroup:  current.ResourceGroup,
		IsOutOfBox:     current.IsOutOfBox,
		LifecycleState: current.LifecycleState,
		TenancyId:      current.TenancyId,
		CompartmentId:  current.CompartmentId,
		ResourceType:   current.ResourceType,
		FreeformTags:   cloneBaselineableMetricStringMap(current.FreeformTags),
		DefinedTags:    cloneBaselineableMetricDefinedTags(current.DefinedTags),
		SystemTags:     cloneBaselineableMetricDefinedTags(current.SystemTags),
	}
}

func validateBaselineableMetricRequiredSpec(spec stackmonitoringv1beta1.BaselineableMetricSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.Column) == "" {
		missing = append(missing, "column")
	}
	if strings.TrimSpace(spec.Namespace) == "" {
		missing = append(missing, "namespace")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("BaselineableMetric spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func guardBaselineableMetricExistingBeforeCreate(
	_ context.Context,
	resource *stackmonitoringv1beta1.BaselineableMetric,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("BaselineableMetric resource is nil")
	}
	if strings.TrimSpace(resource.Spec.Id) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveBaselineableMetricIdentity(resource *stackmonitoringv1beta1.BaselineableMetric) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("BaselineableMetric resource is nil")
	}
	id := strings.TrimSpace(resource.Spec.Id)
	if id == "" {
		return nil, nil
	}
	return baselineableMetricIdentity{id: id}, nil
}

func lookupBaselineableMetricExistingByID(
	listBaselineableMetrics func(context.Context, stackmonitoringsdk.ListBaselineableMetricsRequest) (stackmonitoringsdk.ListBaselineableMetricsResponse, error),
) func(context.Context, *stackmonitoringv1beta1.BaselineableMetric, any) (any, error) {
	if listBaselineableMetrics == nil {
		return nil
	}
	return func(
		ctx context.Context,
		resource *stackmonitoringv1beta1.BaselineableMetric,
		identity any,
	) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("BaselineableMetric resource is nil")
		}
		id := strings.TrimSpace(resource.Spec.Id)
		if id == "" {
			id = baselineableMetricIdentityID(identity)
		}
		if id == "" {
			return nil, nil
		}
		response, err := listBaselineableMetrics(ctx, stackmonitoringsdk.ListBaselineableMetricsRequest{
			BaselineableMetricId: common.String(id),
		})
		if err != nil {
			return nil, err
		}
		if summary, found := baselineableMetricSummaryByID(response.Items, id); found {
			return summary, nil
		}
		return nil, fmt.Errorf("BaselineableMetric spec.id %q was not found; refusing to create because OCI CreateBaselineableMetricDetails does not accept id", id)
	}
}

func baselineableMetricIdentityID(identity any) string {
	switch value := identity.(type) {
	case baselineableMetricIdentity:
		return strings.TrimSpace(value.id)
	case *baselineableMetricIdentity:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(value.id)
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func baselineableMetricSummaryByID(
	items []stackmonitoringsdk.BaselineableMetricSummary,
	id string,
) (stackmonitoringsdk.BaselineableMetricSummary, bool) {
	for _, item := range items {
		if strings.TrimSpace(stringValue(item.Id)) == id {
			return item, true
		}
	}
	return stackmonitoringsdk.BaselineableMetricSummary{}, false
}

func validateBaselineableMetricCreateOnlyDriftForResponse(
	resource *stackmonitoringv1beta1.BaselineableMetric,
	currentResponse any,
) error {
	current, ok := baselineableMetricFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current BaselineableMetric response does not expose a BaselineableMetric body")
	}
	return validateBaselineableMetricCreateOnlyDrift(resource, current)
}

func normalizeBaselineableMetricDesiredState(
	resource *stackmonitoringv1beta1.BaselineableMetric,
	currentResponse any,
) {
	if resource == nil {
		return
	}
	current, ok := baselineableMetricFromResponse(currentResponse)
	if !ok {
		return
	}
	if id := strings.TrimSpace(stringValue(current.Id)); id != "" && strings.TrimSpace(resource.Spec.Id) == "" {
		resource.Spec.Id = id
	}
}

func validateBaselineableMetricCreateOnlyDrift(
	resource *stackmonitoringv1beta1.BaselineableMetric,
	current stackmonitoringsdk.BaselineableMetric,
) error {
	if resource == nil {
		return fmt.Errorf("BaselineableMetric resource is nil")
	}
	spec := resource.Spec
	var drift []string
	drift = appendOptionalStringDrift(drift, "id", spec.Id, current.Id)
	drift = appendOptionalStringDrift(drift, "lifecycleState", spec.LifecycleState, common.String(string(current.LifecycleState)))
	drift = appendOptionalStringDrift(drift, "tenancyId", spec.TenancyId, current.TenancyId)
	if spec.SystemTags != nil && !reflect.DeepEqual(baselineableMetricDefinedTags(spec.SystemTags), current.SystemTags) {
		drift = append(drift, "systemTags")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("BaselineableMetric create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func appendOptionalStringDrift(drift []string, fieldName string, desired string, current *string) []string {
	if strings.TrimSpace(desired) == "" {
		return drift
	}
	if !stringPtrEqual(current, desired) {
		return append(drift, fieldName)
	}
	return drift
}

func applyBaselineableMetricStringUpdate(target **string, desired string, current *string) bool {
	if stringPtrEqual(current, desired) {
		return false
	}
	*target = common.String(strings.TrimSpace(desired))
	return true
}

func applyOptionalBaselineableMetricStringUpdate(target **string, desired string, current *string) bool {
	if strings.TrimSpace(desired) == "" || stringPtrEqual(current, desired) {
		return false
	}
	*target = common.String(strings.TrimSpace(desired))
	return true
}

func projectBaselineableMetricStatus(resource *stackmonitoringv1beta1.BaselineableMetric, response any) error {
	if resource == nil {
		return fmt.Errorf("BaselineableMetric resource is nil")
	}
	current, ok := baselineableMetricFromResponse(response)
	if !ok {
		return nil
	}
	resource.Status = stackmonitoringv1beta1.BaselineableMetricStatus{
		OsokStatus:      resource.Status.OsokStatus,
		Id:              stringValue(current.Id),
		Name:            stringValue(current.Name),
		Column:          stringValue(current.Column),
		Namespace:       stringValue(current.Namespace),
		ResourceGroup:   stringValue(current.ResourceGroup),
		IsOutOfBox:      boolValue(current.IsOutOfBox),
		LifecycleState:  string(current.LifecycleState),
		TenancyId:       stringValue(current.TenancyId),
		CompartmentId:   stringValue(current.CompartmentId),
		ResourceType:    stringValue(current.ResourceType),
		CreatedBy:       stringValue(current.CreatedBy),
		LastUpdatedBy:   stringValue(current.LastUpdatedBy),
		TimeCreated:     sdkTimeString(current.TimeCreated),
		TimeLastUpdated: sdkTimeString(current.TimeLastUpdated),
		FreeformTags:    cloneBaselineableMetricStringMap(current.FreeformTags),
		DefinedTags:     baselineableMetricStatusDefinedTags(current.DefinedTags),
		SystemTags:      baselineableMetricStatusDefinedTags(current.SystemTags),
	}
	return nil
}

func clearTrackedBaselineableMetricIdentity(resource *stackmonitoringv1beta1.BaselineableMetric) {
	if resource == nil {
		return
	}
	resource.Status = stackmonitoringv1beta1.BaselineableMetricStatus{}
}

func listBaselineableMetricPages(
	ctx context.Context,
	client baselineableMetricOCIClient,
	initErr error,
	request stackmonitoringsdk.ListBaselineableMetricsRequest,
) (stackmonitoringsdk.ListBaselineableMetricsResponse, error) {
	if err := baselineableMetricClientReady(client, initErr); err != nil {
		return stackmonitoringsdk.ListBaselineableMetricsResponse{}, err
	}

	var combined stackmonitoringsdk.ListBaselineableMetricsResponse
	for {
		response, err := client.ListBaselineableMetrics(ctx, request)
		if err != nil {
			return stackmonitoringsdk.ListBaselineableMetricsResponse{}, conservativeBaselineableMetricNotFoundError(err, "list")
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		if combined.RetryAfter == nil {
			combined.RetryAfter = response.RetryAfter
		}
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return combined, nil
		}
		nextPage := strings.TrimSpace(*response.OpcNextPage)
		request.Page = common.String(nextPage)
		combined.OpcNextPage = common.String(nextPage)
	}
}

func (c baselineableMetricRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.BaselineableMetric,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("BaselineableMetric generated runtime delegate is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c baselineableMetricRuntimeClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.BaselineableMetric,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("BaselineableMetric generated runtime delegate is not configured")
	}
	currentID := trackedBaselineableMetricID(resource)
	if currentID == "" {
		return true, nil
	}
	if c.get != nil {
		_, err := c.get(ctx, stackmonitoringsdk.GetBaselineableMetricRequest{
			BaselineableMetricId: common.String(currentID),
		})
		if isAmbiguousBaselineableMetricNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, fmt.Errorf("BaselineableMetric delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func trackedBaselineableMetricID(resource *stackmonitoringv1beta1.BaselineableMetric) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func handleBaselineableMetricDeleteError(
	resource *stackmonitoringv1beta1.BaselineableMetric,
	err error,
) error {
	if err == nil {
		return nil
	}
	err = conservativeBaselineableMetricNotFoundError(err, "delete")
	if isAmbiguousBaselineableMetricNotFound(err) && resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeBaselineableMetricNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !isAmbiguousBaselineableMetricNotFound(err) {
		return err
	}
	return ambiguousBaselineableMetricNotFoundError{
		message:      fmt.Sprintf("BaselineableMetric %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isAmbiguousBaselineableMetricNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousBaselineableMetricNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func baselineableMetricClientReady(client baselineableMetricOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize BaselineableMetric OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("BaselineableMetric OCI client is not configured")
	}
	return nil
}

func baselineableMetricFromResponse(response any) (stackmonitoringsdk.BaselineableMetric, bool) {
	if metric, ok := baselineableMetricFromResponseValue(response); ok {
		return metric, true
	}
	return baselineableMetricFromResponsePointer(response)
}

func baselineableMetricFromResponseValue(response any) (stackmonitoringsdk.BaselineableMetric, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.BaselineableMetric:
		return current, true
	case stackmonitoringsdk.BaselineableMetricSummary:
		return baselineableMetricFromSummary(current), true
	case stackmonitoringsdk.CreateBaselineableMetricResponse:
		return current.BaselineableMetric, true
	case stackmonitoringsdk.GetBaselineableMetricResponse:
		return current.BaselineableMetric, true
	case stackmonitoringsdk.UpdateBaselineableMetricResponse:
		return current.BaselineableMetric, true
	default:
		return stackmonitoringsdk.BaselineableMetric{}, false
	}
}

func baselineableMetricFromResponsePointer(response any) (stackmonitoringsdk.BaselineableMetric, bool) {
	switch current := response.(type) {
	case *stackmonitoringsdk.BaselineableMetric:
		return baselineableMetricFromOptional(current, func(metric stackmonitoringsdk.BaselineableMetric) stackmonitoringsdk.BaselineableMetric {
			return metric
		})
	case *stackmonitoringsdk.BaselineableMetricSummary:
		return baselineableMetricFromOptional(current, baselineableMetricFromSummary)
	case *stackmonitoringsdk.CreateBaselineableMetricResponse:
		return baselineableMetricFromOptional(current, func(response stackmonitoringsdk.CreateBaselineableMetricResponse) stackmonitoringsdk.BaselineableMetric {
			return response.BaselineableMetric
		})
	case *stackmonitoringsdk.GetBaselineableMetricResponse:
		return baselineableMetricFromOptional(current, func(response stackmonitoringsdk.GetBaselineableMetricResponse) stackmonitoringsdk.BaselineableMetric {
			return response.BaselineableMetric
		})
	case *stackmonitoringsdk.UpdateBaselineableMetricResponse:
		return baselineableMetricFromOptional(current, func(response stackmonitoringsdk.UpdateBaselineableMetricResponse) stackmonitoringsdk.BaselineableMetric {
			return response.BaselineableMetric
		})
	default:
		return stackmonitoringsdk.BaselineableMetric{}, false
	}
}

func baselineableMetricFromOptional[T any](
	value *T,
	convert func(T) stackmonitoringsdk.BaselineableMetric,
) (stackmonitoringsdk.BaselineableMetric, bool) {
	if value == nil {
		return stackmonitoringsdk.BaselineableMetric{}, false
	}
	return convert(*value), true
}

func baselineableMetricFromSummary(summary stackmonitoringsdk.BaselineableMetricSummary) stackmonitoringsdk.BaselineableMetric {
	return stackmonitoringsdk.BaselineableMetric{
		Id:             summary.Id,
		Name:           summary.Name,
		Column:         summary.Column,
		Namespace:      summary.Namespace,
		ResourceGroup:  summary.ResourceGroup,
		IsOutOfBox:     summary.IsOutOfBox,
		LifecycleState: summary.LifecycleState,
		TenancyId:      summary.TenancyId,
		CompartmentId:  summary.CompartmentId,
		ResourceType:   summary.ResourceType,
		FreeformTags:   cloneBaselineableMetricStringMap(summary.FreeformTags),
		DefinedTags:    cloneBaselineableMetricDefinedTags(summary.DefinedTags),
		SystemTags:     cloneBaselineableMetricDefinedTags(summary.SystemTags),
	}
}

func desiredBaselineableMetricFreeformTags(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec != nil {
		desired := cloneBaselineableMetricStringMap(spec)
		return desired, !reflect.DeepEqual(current, desired)
	}
	return nil, false
}

func desiredBaselineableMetricDefinedTags(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec != nil {
		desired := baselineableMetricDefinedTags(spec)
		return desired, !reflect.DeepEqual(current, desired)
	}
	return nil, false
}

func baselineableMetricDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func baselineableMetricStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func cloneBaselineableMetricStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneBaselineableMetricDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		cloned[namespace] = child
	}
	return cloned
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func stringPtrEqual(value *string, desired string) bool {
	return strings.TrimSpace(stringValue(value)) == strings.TrimSpace(desired)
}
