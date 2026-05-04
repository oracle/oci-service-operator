/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alarmcondition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
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
	alarmConditionKind = "AlarmCondition"

	alarmConditionMonitoringTemplateIDAnnotation = "stackmonitoring.oracle.com/monitoring-template-id"
)

type alarmConditionOCIClient interface {
	CreateAlarmCondition(context.Context, stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error)
	GetAlarmCondition(context.Context, stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error)
	ListAlarmConditions(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error)
	UpdateAlarmCondition(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error)
	DeleteAlarmCondition(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error)
}

type alarmConditionIdentity struct {
	monitoringTemplateID string
	namespace            string
	resourceType         string
	metricName           string
	conditionType        string
	compositeType        string
}

type alarmConditionDeleteGuardClient struct {
	delegate AlarmConditionServiceClient
	get      func(context.Context, stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error)
	list     func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error)
}

type alarmConditionAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type alarmConditionRuntimeReadResponse struct {
	Body         map[string]any `presentIn:"body"`
	OpcRequestId *string        `presentIn:"header" name:"opc-request-id"`
	Etag         *string        `presentIn:"header" name:"etag"`
}

type alarmConditionRuntimeListBody struct {
	Items []map[string]any `json:"items"`
}

type alarmConditionRuntimeListResponse struct {
	Body         alarmConditionRuntimeListBody `presentIn:"body"`
	OpcRequestId *string                       `presentIn:"header" name:"opc-request-id"`
	OpcNextPage  *string                       `presentIn:"header" name:"opc-next-page"`
}

type alarmConditionCreateFollowUpContextKey struct{}

type alarmConditionIdentityContextKey struct{}

type alarmConditionCreateFollowUpState struct {
	response stackmonitoringsdk.CreateAlarmConditionResponse
	created  bool
}

func (e alarmConditionAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e alarmConditionAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerAlarmConditionRuntimeHooksMutator(func(_ *AlarmConditionServiceManager, hooks *AlarmConditionRuntimeHooks) {
		applyAlarmConditionRuntimeHooks(hooks)
	})
}

func applyAlarmConditionRuntimeHooks(hooks *AlarmConditionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = alarmConditionRuntimeSemantics()
	hooks.BuildCreateBody = buildAlarmConditionCreateBody
	hooks.BuildUpdateBody = buildAlarmConditionUpdateBody
	hooks.Identity.Resolve = resolveAlarmConditionIdentity
	hooks.Identity.RecordPath = recordAlarmConditionPathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardAlarmConditionExistingBeforeCreate
	hooks.List.Fields = alarmConditionListFields()
	hooks.List.Call = paginatedAlarmConditionListCall(hooks.List.Call)
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateAlarmConditionCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleAlarmConditionDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyAlarmConditionDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectAlarmConditionStatus
	hooks.StatusHooks.MarkTerminating = markAlarmConditionTerminating
	wrapAlarmConditionCreateFollowUp(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AlarmConditionServiceClient) AlarmConditionServiceClient {
		return alarmConditionIdentityContextClient{delegate: delegate}
	})
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AlarmConditionServiceClient) AlarmConditionServiceClient {
		return alarmConditionCreateFollowUpClient{delegate: delegate}
	})
	if hooks.Get.Call != nil {
		get := hooks.Get.Call
		list := hooks.List.Call
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AlarmConditionServiceClient) AlarmConditionServiceClient {
			return alarmConditionDeleteGuardClient{delegate: delegate, get: get, list: list}
		})
	}
	wrapAlarmConditionReadAndDeleteCalls(hooks)
	hooks.Identity.LookupExisting = alarmConditionLookupExistingByIdentity(hooks.List.Call)
	hooks.DeleteHooks.ConfirmRead = alarmConditionDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	configureAlarmConditionReadHooks(hooks)
}

func alarmConditionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "alarmcondition",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(stackmonitoringsdk.AlarmConditionLifeCycleStatesCreating)},
			UpdatingStates:     []string{string(stackmonitoringsdk.AlarmConditionLifeCycleStatesUpdating)},
			ActiveStates: []string{
				string(stackmonitoringsdk.AlarmConditionLifeCycleStatesActive),
				string(stackmonitoringsdk.AlarmConditionLifeCycleStatesInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(stackmonitoringsdk.AlarmConditionLifeCycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"monitoringTemplateId",
				"namespace",
				"resourceType",
				"metricName",
				"conditionType",
				"compositeType",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"namespace",
				"resourceType",
				"metricName",
				"conditionType",
				"conditions",
				"compositeType",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"monitoringTemplateId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AlarmCondition", Action: "CreateAlarmCondition"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AlarmCondition", Action: "UpdateAlarmCondition"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AlarmCondition", Action: "DeleteAlarmCondition"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "AlarmCondition", Action: "GetAlarmCondition"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "AlarmCondition", Action: "GetAlarmCondition"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "AlarmCondition", Action: "GetAlarmCondition"}},
		},
		// AlarmCondition is an exception to recording the annotation-only parent
		// as UnsupportedSemantic: this package fully handles monitoringTemplateId
		// from metadata, while UnsupportedSemantic blocks generatedruntime CRUD.
	}
}

func alarmConditionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "MonitoringTemplateId",
			RequestName:  "monitoringTemplateId",
			Contribution: "path",
			LookupPaths:  []string{"status.monitoringTemplateId", "monitoringTemplateId"},
		},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func resolveAlarmConditionIdentity(resource *stackmonitoringv1beta1.AlarmCondition) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", alarmConditionKind)
	}
	identity, err := alarmConditionIdentityFromResource(resource)
	if err != nil {
		return nil, err
	}
	if tracked := strings.TrimSpace(resource.Status.MonitoringTemplateId); tracked != "" &&
		identity.monitoringTemplateID != "" &&
		tracked != identity.monitoringTemplateID {
		return nil, fmt.Errorf("%s formal semantics require replacement when monitoringTemplateId changes", alarmConditionKind)
	}
	return identity, nil
}

func recordAlarmConditionPathIdentity(resource *stackmonitoringv1beta1.AlarmCondition, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(alarmConditionIdentity)
	if !ok {
		return
	}
	if typed.monitoringTemplateID != "" {
		resource.Status.MonitoringTemplateId = typed.monitoringTemplateID
	}
}

func guardAlarmConditionExistingBeforeCreate(
	_ context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if _, err := alarmConditionIdentityFromResource(resource); err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func alarmConditionIdentityFromResource(resource *stackmonitoringv1beta1.AlarmCondition) (alarmConditionIdentity, error) {
	if resource == nil {
		return alarmConditionIdentity{}, fmt.Errorf("%s resource is nil", alarmConditionKind)
	}

	var problems []string
	monitoringTemplateID := alarmConditionMonitoringTemplateIdentity(resource, &problems)
	namespace := alarmConditionRequiredSpecString("namespace", resource.Spec.Namespace, &problems)
	resourceType := alarmConditionRequiredSpecString("resourceType", resource.Spec.ResourceType, &problems)
	metricName := alarmConditionRequiredSpecString("metricName", resource.Spec.MetricName, &problems)
	conditionType := alarmConditionRequiredSpecString("conditionType", resource.Spec.ConditionType, &problems)
	if conditionType != "" {
		normalized, err := alarmConditionConditionType(conditionType)
		if err != nil {
			problems = append(problems, err.Error())
		} else {
			conditionType = string(normalized)
		}
	}
	alarmConditionRequireConditions(resource.Spec.Conditions, &problems)
	if len(problems) != 0 {
		return alarmConditionIdentity{}, fmt.Errorf("%s spec is invalid: %s", alarmConditionKind, strings.Join(problems, "; "))
	}
	return alarmConditionIdentity{
		monitoringTemplateID: monitoringTemplateID,
		namespace:            namespace,
		resourceType:         resourceType,
		metricName:           metricName,
		conditionType:        conditionType,
		compositeType:        strings.TrimSpace(resource.Spec.CompositeType),
	}, nil
}

func alarmConditionMonitoringTemplateIdentity(resource *stackmonitoringv1beta1.AlarmCondition, problems *[]string) string {
	annotation := alarmConditionAnnotation(resource, alarmConditionMonitoringTemplateIDAnnotation)
	tracked := strings.TrimSpace(resource.Status.MonitoringTemplateId)
	if tracked != "" && annotation != "" && annotation != tracked {
		*problems = append(*problems, "metadata annotation monitoringTemplateId changes are not supported")
	}
	monitoringTemplateID := firstNonEmptyTrimAlarmCondition(tracked, annotation)
	if monitoringTemplateID == "" {
		*problems = append(
			*problems,
			fmt.Sprintf("metadata annotation %q is required because the CRD has no spec monitoringTemplateId field", alarmConditionMonitoringTemplateIDAnnotation),
		)
	}
	return monitoringTemplateID
}

func alarmConditionRequiredSpecString(name string, value string, problems *[]string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		*problems = append(*problems, fmt.Sprintf("spec.%s is required", name))
	}
	return trimmed
}

func alarmConditionRequireConditions(
	conditions []stackmonitoringv1beta1.AlarmConditionCondition,
	problems *[]string,
) {
	if len(conditions) == 0 {
		*problems = append(*problems, "spec.conditions is required")
	}
}

func buildAlarmConditionCreateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
	_ string,
) (any, error) {
	if _, err := alarmConditionIdentityFromResource(resource); err != nil {
		return nil, err
	}
	conditionType, err := alarmConditionConditionType(resource.Spec.ConditionType)
	if err != nil {
		return nil, err
	}
	conditions, err := alarmConditionConditionsFromSpec(resource.Spec.Conditions)
	if err != nil {
		return nil, err
	}

	details := stackmonitoringsdk.CreateAlarmConditionDetails{
		Namespace:     common.String(strings.TrimSpace(resource.Spec.Namespace)),
		ResourceType:  common.String(strings.TrimSpace(resource.Spec.ResourceType)),
		MetricName:    common.String(strings.TrimSpace(resource.Spec.MetricName)),
		ConditionType: conditionType,
		Conditions:    conditions,
	}
	if compositeType := strings.TrimSpace(resource.Spec.CompositeType); compositeType != "" {
		details.CompositeType = common.String(compositeType)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = alarmConditionDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildAlarmConditionUpdateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if _, err := alarmConditionIdentityFromResource(resource); err != nil {
		return stackmonitoringsdk.UpdateAlarmConditionDetails{}, false, err
	}
	current, ok := alarmConditionFromResponse(currentResponse)
	if !ok {
		return stackmonitoringsdk.UpdateAlarmConditionDetails{}, false, fmt.Errorf("current %s response does not expose an alarm condition body", alarmConditionKind)
	}
	if err := validateAlarmConditionCreateOnlyDrift(resource, current); err != nil {
		return stackmonitoringsdk.UpdateAlarmConditionDetails{}, false, err
	}

	details := stackmonitoringsdk.UpdateAlarmConditionDetails{}
	updateNeeded, err := applyAlarmConditionUpdateDetails(&details, resource, current)
	if err != nil {
		return stackmonitoringsdk.UpdateAlarmConditionDetails{}, false, err
	}
	return details, updateNeeded, nil
}

func applyAlarmConditionUpdateDetails(
	details *stackmonitoringsdk.UpdateAlarmConditionDetails,
	resource *stackmonitoringv1beta1.AlarmCondition,
	current stackmonitoringsdk.AlarmCondition,
) (bool, error) {
	updateNeeded, err := applyAlarmConditionStringUpdateDetails(details, resource, current)
	if err != nil {
		return false, err
	}
	updated, err := applyAlarmConditionTypedUpdateDetails(details, resource, current)
	if err != nil {
		return false, err
	}
	updateNeeded = updateNeeded || updated
	if desired, ok := alarmConditionFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := alarmConditionDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	return updateNeeded, nil
}

func applyAlarmConditionStringUpdateDetails(
	details *stackmonitoringsdk.UpdateAlarmConditionDetails,
	resource *stackmonitoringv1beta1.AlarmCondition,
	current stackmonitoringsdk.AlarmCondition,
) (bool, error) {
	updateNeeded := false
	updates := []struct {
		name     string
		desired  string
		current  *string
		required bool
		clear    bool
		apply    func(*string)
	}{
		{name: "namespace", desired: resource.Spec.Namespace, current: current.Namespace, required: true, apply: func(value *string) { details.Namespace = value }},
		{name: "resourceType", desired: resource.Spec.ResourceType, current: current.ResourceType, required: true, apply: func(value *string) { details.ResourceType = value }},
		{name: "metricName", desired: resource.Spec.MetricName, current: current.MetricName, required: true, apply: func(value *string) { details.MetricName = value }},
		{name: "compositeType", desired: resource.Spec.CompositeType, current: current.CompositeType, clear: true, apply: func(value *string) { details.CompositeType = value }},
	}
	for _, update := range updates {
		updated, err := applyAlarmConditionStringUpdate(update.name, update.desired, update.current, update.required, update.clear, update.apply)
		if err != nil {
			return false, err
		}
		updateNeeded = updateNeeded || updated
	}
	return updateNeeded, nil
}

func applyAlarmConditionStringUpdate(
	name string,
	desired string,
	current *string,
	required bool,
	clear bool,
	apply func(*string),
) (bool, error) {
	value, ok, err := alarmConditionStringUpdate(name, desired, current, required, clear)
	if err != nil || !ok {
		return false, err
	}
	apply(value)
	return true, nil
}

func applyAlarmConditionTypedUpdateDetails(
	details *stackmonitoringsdk.UpdateAlarmConditionDetails,
	resource *stackmonitoringv1beta1.AlarmCondition,
	current stackmonitoringsdk.AlarmCondition,
) (bool, error) {
	updateNeeded := false
	if desired, ok, err := alarmConditionConditionTypeUpdate(resource.Spec.ConditionType, current.ConditionType); err != nil {
		return false, err
	} else if ok {
		details.ConditionType = desired
		updateNeeded = true
	}
	if desired, ok, err := alarmConditionConditionsUpdate(resource.Spec.Conditions, current.Conditions); err != nil {
		return false, err
	} else if ok {
		details.Conditions = desired
		updateNeeded = true
	}
	return updateNeeded, nil
}

func alarmConditionStringUpdate(name string, desired string, current *string, required bool, clear bool) (*string, bool, error) {
	trimmed := strings.TrimSpace(desired)
	currentValue := alarmConditionStringValue(current)
	if trimmed == "" {
		if required {
			return nil, false, fmt.Errorf("%s spec.%s is required", alarmConditionKind, name)
		}
		if clear && currentValue != "" {
			return common.String(""), true, nil
		}
		return nil, false, nil
	}
	if trimmed == currentValue {
		return nil, false, nil
	}
	return common.String(trimmed), true, nil
}

func alarmConditionConditionTypeUpdate(
	desired string,
	current stackmonitoringsdk.ConditionTypeEnum,
) (stackmonitoringsdk.ConditionTypeEnum, bool, error) {
	conditionType, err := alarmConditionConditionType(desired)
	if err != nil {
		return "", false, err
	}
	if conditionType == current {
		return "", false, nil
	}
	return conditionType, true, nil
}

func alarmConditionConditionsUpdate(
	desired []stackmonitoringv1beta1.AlarmConditionCondition,
	current []stackmonitoringsdk.Condition,
) ([]stackmonitoringsdk.Condition, bool, error) {
	conditions, err := alarmConditionConditionsFromSpec(desired)
	if err != nil {
		return nil, false, err
	}
	if alarmConditionConditionsEqual(conditions, current) {
		return nil, false, nil
	}
	return conditions, true, nil
}

func alarmConditionFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil || maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func alarmConditionDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := alarmConditionDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func validateAlarmConditionCreateOnlyDriftForResponse(
	resource *stackmonitoringv1beta1.AlarmCondition,
	currentResponse any,
) error {
	current, ok := alarmConditionFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateAlarmConditionCreateOnlyDrift(resource, current)
}

func validateAlarmConditionCreateOnlyDrift(
	resource *stackmonitoringv1beta1.AlarmCondition,
	current stackmonitoringsdk.AlarmCondition,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", alarmConditionKind)
	}
	desired := firstNonEmptyTrimAlarmCondition(
		resource.Status.MonitoringTemplateId,
		alarmConditionAnnotation(resource, alarmConditionMonitoringTemplateIDAnnotation),
	)
	currentParent := alarmConditionStringValue(current.MonitoringTemplateId)
	if desired != "" && currentParent != "" && desired != currentParent {
		return fmt.Errorf("%s create-only field drift is not supported: monitoringTemplateId", alarmConditionKind)
	}
	return nil
}

type alarmConditionIdentityContextClient struct {
	delegate AlarmConditionServiceClient
}

func (c alarmConditionIdentityContextClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", alarmConditionKind)
	}
	identity, err := alarmConditionIdentityFromResource(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, alarmConditionIdentityContextKey{}, identity)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c alarmConditionIdentityContextClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", alarmConditionKind)
	}
	return c.delegate.Delete(ctx, resource)
}

func (c alarmConditionDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", alarmConditionKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c alarmConditionDeleteGuardClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", alarmConditionKind)
	}
	if c.get == nil {
		return c.delegate.Delete(ctx, resource)
	}
	currentID := alarmConditionTrackedID(resource)
	monitoringTemplateID := alarmConditionMonitoringTemplateID(resource)
	if currentID == "" && monitoringTemplateID == "" {
		markAlarmConditionDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	if currentID == "" {
		return c.deleteWithoutTrackedID(ctx, resource)
	}
	if monitoringTemplateID == "" {
		return c.delegate.Delete(ctx, resource)
	}
	return c.deleteWithTrackedID(ctx, resource, currentID, monitoringTemplateID)
}

func (c alarmConditionDeleteGuardClient) deleteWithoutTrackedID(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
) (bool, error) {
	if c.list == nil {
		return c.delegate.Delete(ctx, resource)
	}
	identity, err := alarmConditionIdentityFromResource(resource)
	if err != nil {
		return false, err
	}
	response, err := c.list(ctx, stackmonitoringsdk.ListAlarmConditionsRequest{
		MonitoringTemplateId: common.String(identity.monitoringTemplateID),
	})
	if err != nil {
		return false, err
	}
	match, err := selectAlarmConditionDeleteListMatch(response.Items, identity)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			markAlarmConditionDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	if err := projectAlarmConditionStatus(resource, stackmonitoringsdk.GetAlarmConditionResponse{
		AlarmCondition: alarmConditionFromSummary(match),
		OpcRequestId:   response.OpcRequestId,
	}); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c alarmConditionDeleteGuardClient) deleteWithTrackedID(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
	currentID string,
	monitoringTemplateID string,
) (bool, error) {
	response, err := c.get(ctx, stackmonitoringsdk.GetAlarmConditionRequest{
		AlarmConditionId:     common.String(currentID),
		MonitoringTemplateId: common.String(monitoringTemplateID),
	})
	if err == nil {
		handled, handleErr := waitForAlarmConditionWriteBeforeDelete(resource, response)
		if handled || handleErr != nil {
			return false, handleErr
		}
		return c.delegate.Delete(ctx, resource)
	}
	if isAlarmConditionAmbiguousNotFound(err) || errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, fmt.Errorf("%s delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", alarmConditionKind, err)
	}
	return c.delegate.Delete(ctx, resource)
}

func alarmConditionDeleteConfirmRead(
	get func(context.Context, stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error),
	list func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error),
) func(context.Context, *stackmonitoringv1beta1.AlarmCondition, string) (any, error) {
	return func(ctx context.Context, resource *stackmonitoringv1beta1.AlarmCondition, currentID string) (any, error) {
		monitoringTemplateID := alarmConditionMonitoringTemplateID(resource)
		if strings.TrimSpace(currentID) != "" {
			if get == nil {
				return nil, fmt.Errorf("%s generated runtime has no get operation", alarmConditionKind)
			}
			return get(ctx, stackmonitoringsdk.GetAlarmConditionRequest{
				AlarmConditionId:     common.String(strings.TrimSpace(currentID)),
				MonitoringTemplateId: common.String(monitoringTemplateID),
			})
		}

		if list == nil {
			return nil, fmt.Errorf("%s generated runtime has no list operation", alarmConditionKind)
		}
		identity, err := alarmConditionIdentityFromResource(resource)
		if err != nil {
			return nil, err
		}
		response, err := list(ctx, stackmonitoringsdk.ListAlarmConditionsRequest{
			MonitoringTemplateId: common.String(identity.monitoringTemplateID),
		})
		if err != nil {
			return response, err
		}
		match, err := selectAlarmConditionDeleteListMatch(response.Items, identity)
		if err != nil {
			return nil, err
		}
		return stackmonitoringsdk.GetAlarmConditionResponse{
			AlarmCondition: alarmConditionFromSummary(match),
			OpcRequestId:   response.OpcRequestId,
		}, nil
	}
}

func alarmConditionLookupExistingByIdentity(
	list func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error),
) func(context.Context, *stackmonitoringv1beta1.AlarmCondition, any) (any, error) {
	if list == nil {
		return nil
	}
	return func(ctx context.Context, _ *stackmonitoringv1beta1.AlarmCondition, identity any) (any, error) {
		typed, ok := identity.(alarmConditionIdentity)
		if !ok {
			return nil, fmt.Errorf("%s identity has unexpected type %T", alarmConditionKind, identity)
		}
		response, err := list(ctx, stackmonitoringsdk.ListAlarmConditionsRequest{
			MonitoringTemplateId: common.String(typed.monitoringTemplateID),
		})
		if err != nil {
			return nil, err
		}
		match, err := selectAlarmConditionDeleteListMatch(response.Items, typed)
		if err != nil {
			if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
				return stackmonitoringsdk.GetAlarmConditionResponse{}, nil
			}
			return nil, err
		}
		return stackmonitoringsdk.GetAlarmConditionResponse{
			AlarmCondition: alarmConditionFromSummary(match),
			OpcRequestId:   response.OpcRequestId,
		}, nil
	}
}

func selectAlarmConditionDeleteListMatch(
	items []stackmonitoringsdk.AlarmConditionSummary,
	identity alarmConditionIdentity,
) (stackmonitoringsdk.AlarmConditionSummary, error) {
	var matches []stackmonitoringsdk.AlarmConditionSummary
	for _, item := range items {
		if alarmConditionSummaryMatchesIdentity(item, identity) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return stackmonitoringsdk.AlarmConditionSummary{}, alarmConditionListNotFoundError()
	case 1:
		return matches[0], nil
	default:
		return stackmonitoringsdk.AlarmConditionSummary{}, fmt.Errorf("%s delete list returned multiple matching resources", alarmConditionKind)
	}
}

func alarmConditionSummaryMatchesIdentity(
	item stackmonitoringsdk.AlarmConditionSummary,
	identity alarmConditionIdentity,
) bool {
	if alarmConditionStringValue(item.MonitoringTemplateId) != identity.monitoringTemplateID ||
		alarmConditionStringValue(item.Namespace) != identity.namespace ||
		alarmConditionStringValue(item.ResourceType) != identity.resourceType ||
		alarmConditionStringValue(item.MetricName) != identity.metricName ||
		string(item.ConditionType) != identity.conditionType {
		return false
	}
	return alarmConditionStringValue(item.CompositeType) == identity.compositeType
}

func alarmConditionListNotFoundError() error {
	_, err := errorutil.NewServiceFailureFromResponse(
		errorutil.NotFound,
		404,
		"",
		"alarm condition not found in parent monitoring template",
	)
	return err
}

func wrapAlarmConditionReadAndDeleteCalls(hooks *AlarmConditionRuntimeHooks) {
	if hooks.Get.Call != nil {
		get := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			response, err := get(ctx, request)
			return response, conservativeAlarmConditionNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		list := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			response, err := list(ctx, request)
			return response, conservativeAlarmConditionNotFoundError(err, "list")
		}
	}
	if hooks.Delete.Call != nil {
		del := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			response, err := del(ctx, request)
			return response, conservativeAlarmConditionNotFoundError(err, "delete")
		}
	}
}

func wrapAlarmConditionCreateFollowUp(hooks *AlarmConditionRuntimeHooks) {
	if hooks == nil {
		return
	}
	wrapAlarmConditionCreateCall(hooks)
	wrapAlarmConditionCreateFollowUpGetCall(hooks)
}

func wrapAlarmConditionCreateCall(hooks *AlarmConditionRuntimeHooks) {
	if hooks.Create.Call == nil {
		return
	}
	create := hooks.Create.Call
	hooks.Create.Call = func(ctx context.Context, request stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
		response, err := create(ctx, request)
		if err == nil {
			recordAlarmConditionCreateFollowUpState(ctx, response)
		}
		return response, err
	}
}

func wrapAlarmConditionCreateFollowUpGetCall(hooks *AlarmConditionRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	get := hooks.Get.Call
	hooks.Get.Call = func(ctx context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
		response, err := get(ctx, request)
		if err == nil {
			return response, nil
		}
		fallback, ok := alarmConditionCreateFollowUpResponse(ctx, err)
		if !ok {
			return response, err
		}
		return fallback, nil
	}
}

func recordAlarmConditionCreateFollowUpState(ctx context.Context, response stackmonitoringsdk.CreateAlarmConditionResponse) {
	state := alarmConditionCreateFollowUpStateFromContext(ctx)
	if state == nil {
		return
	}
	state.response = response
	state.created = true
}

func alarmConditionCreateFollowUpResponse(
	ctx context.Context,
	err error,
) (stackmonitoringsdk.GetAlarmConditionResponse, bool) {
	state := alarmConditionCreateFollowUpStateFromContext(ctx)
	if state == nil || !state.created || !errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return stackmonitoringsdk.GetAlarmConditionResponse{}, false
	}
	return stackmonitoringsdk.GetAlarmConditionResponse{
		AlarmCondition: state.response.AlarmCondition,
		OpcRequestId:   state.response.OpcRequestId,
		Etag:           state.response.Etag,
	}, true
}

func alarmConditionCreateFollowUpStateFromContext(ctx context.Context) *alarmConditionCreateFollowUpState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(alarmConditionCreateFollowUpContextKey{}).(*alarmConditionCreateFollowUpState)
	return state
}

func paginatedAlarmConditionListCall(
	call func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error),
) func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
		var combined stackmonitoringsdk.ListAlarmConditionsResponse
		nextPage := request.Page
		for {
			pageRequest := request
			pageRequest.Page = nextPage
			response, err := call(ctx, pageRequest)
			if err != nil {
				return response, err
			}
			mergeAlarmConditionListPage(&combined, response)
			if !alarmConditionHasNextPage(response) {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func mergeAlarmConditionListPage(
	combined *stackmonitoringsdk.ListAlarmConditionsResponse,
	response stackmonitoringsdk.ListAlarmConditionsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func alarmConditionHasNextPage(response stackmonitoringsdk.ListAlarmConditionsResponse) bool {
	return response.OpcNextPage != nil && strings.TrimSpace(*response.OpcNextPage) != ""
}

func configureAlarmConditionReadHooks(hooks *AlarmConditionRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Get.Call != nil {
		hooks.Read.Get = alarmConditionGetReadOperation(hooks)
	}
	if hooks.List.Call != nil {
		hooks.Read.List = alarmConditionListReadOperation(hooks)
	}
}

func alarmConditionGetReadOperation(hooks *AlarmConditionRuntimeHooks) *generatedruntime.Operation {
	fields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &stackmonitoringsdk.GetAlarmConditionRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*stackmonitoringsdk.GetAlarmConditionRequest)
			if !ok {
				return nil, fmt.Errorf("expected *stackmonitoring.GetAlarmConditionRequest, got %T", request)
			}
			response, err := hooks.Get.Call(ctx, *typed)
			if err != nil {
				return nil, err
			}
			return alarmConditionRuntimeReadResponse{
				Body:         alarmConditionStatusMap(response.AlarmCondition),
				OpcRequestId: response.OpcRequestId,
				Etag:         response.Etag,
			}, nil
		},
	}
}

func alarmConditionListReadOperation(hooks *AlarmConditionRuntimeHooks) *generatedruntime.Operation {
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &stackmonitoringsdk.ListAlarmConditionsRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*stackmonitoringsdk.ListAlarmConditionsRequest)
			if !ok {
				return nil, fmt.Errorf("expected *stackmonitoring.ListAlarmConditionsRequest, got %T", request)
			}
			response, err := hooks.List.Call(ctx, *typed)
			if err != nil {
				return nil, err
			}
			filterAlarmConditionListResponseForIdentity(ctx, &response)
			items := make([]map[string]any, 0, len(response.Items))
			for _, item := range response.Items {
				items = append(items, alarmConditionStatusMap(alarmConditionFromSummary(item)))
			}
			return alarmConditionRuntimeListResponse{
				Body:         alarmConditionRuntimeListBody{Items: items},
				OpcRequestId: response.OpcRequestId,
				OpcNextPage:  response.OpcNextPage,
			}, nil
		},
	}
}

func filterAlarmConditionListResponseForIdentity(
	ctx context.Context,
	response *stackmonitoringsdk.ListAlarmConditionsResponse,
) {
	if response == nil {
		return
	}
	identity, ok := alarmConditionIdentityFromContext(ctx)
	if !ok {
		return
	}
	filtered := make([]stackmonitoringsdk.AlarmConditionSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if alarmConditionSummaryMatchesIdentity(item, identity) {
			filtered = append(filtered, item)
		}
	}
	response.Items = filtered
}

func alarmConditionIdentityFromContext(ctx context.Context) (alarmConditionIdentity, bool) {
	if ctx == nil {
		return alarmConditionIdentity{}, false
	}
	identity, ok := ctx.Value(alarmConditionIdentityContextKey{}).(alarmConditionIdentity)
	return identity, ok
}

func handleAlarmConditionDeleteError(resource *stackmonitoringv1beta1.AlarmCondition, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeAlarmConditionNotFoundError(err, "delete")
}

func conservativeAlarmConditionNotFoundError(err error, operation string) error {
	if err == nil || isAlarmConditionAmbiguousNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return alarmConditionAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", alarmConditionKind, strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isAlarmConditionAmbiguousNotFound(err error) bool {
	var ambiguous alarmConditionAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func applyAlarmConditionDeleteOutcome(
	resource *stackmonitoringv1beta1.AlarmCondition,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	current, ok := alarmConditionFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if current.LifecycleState == stackmonitoringsdk.AlarmConditionLifeCycleStatesDeleted {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending {
		if handled, err := waitForAlarmConditionWriteBeforeDelete(resource, response); handled || err != nil {
			return generatedruntime.DeleteOutcome{Handled: handled, Deleted: false}, err
		}
		if !alarmConditionDeleteIsPending(resource) {
			return generatedruntime.DeleteOutcome{}, nil
		}
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending || stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markAlarmConditionTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func waitForAlarmConditionWriteBeforeDelete(
	resource *stackmonitoringv1beta1.AlarmCondition,
	response any,
) (bool, error) {
	current, ok := alarmConditionFromResponse(response)
	if !ok {
		return false, nil
	}
	if _, _, _, ok := alarmConditionPendingWriteLifecycle(current.LifecycleState); !ok {
		return false, nil
	}
	if err := projectAlarmConditionStatus(resource, response); err != nil {
		return true, err
	}
	markAlarmConditionWritePendingDeleteGuard(resource, current.LifecycleState)
	return true, nil
}

func markAlarmConditionWritePendingDeleteGuard(
	resource *stackmonitoringv1beta1.AlarmCondition,
	state stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum,
) bool {
	phase, condition, operation, ok := alarmConditionPendingWriteLifecycle(state)
	if !ok || resource == nil {
		return false
	}
	now := metav1.Now()
	message := fmt.Sprintf("OCI AlarmCondition %s is in progress; waiting before delete", operation)
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       string(state),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
	return true
}

func alarmConditionPendingWriteLifecycle(
	state stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum,
) (shared.OSOKAsyncPhase, shared.OSOKConditionType, string, bool) {
	switch state {
	case stackmonitoringsdk.AlarmConditionLifeCycleStatesCreating:
		return shared.OSOKAsyncPhaseCreate, shared.Provisioning, "create", true
	case stackmonitoringsdk.AlarmConditionLifeCycleStatesUpdating:
		return shared.OSOKAsyncPhaseUpdate, shared.Updating, "update", true
	default:
		return "", "", "", false
	}
}

func alarmConditionDeleteIsPending(resource *stackmonitoringv1beta1.AlarmCondition) bool {
	return resource != nil &&
		resource.Status.OsokStatus.Async.Current != nil &&
		resource.Status.OsokStatus.Async.Current.Phase == shared.OSOKAsyncPhaseDelete
}

func projectAlarmConditionStatus(
	resource *stackmonitoringv1beta1.AlarmCondition,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", alarmConditionKind)
	}
	current, ok := alarmConditionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = stackmonitoringv1beta1.AlarmConditionStatus{
		OsokStatus:           osokStatus,
		Id:                   alarmConditionStringValue(current.Id),
		MonitoringTemplateId: alarmConditionStringValue(current.MonitoringTemplateId),
		Namespace:            alarmConditionStringValue(current.Namespace),
		ResourceType:         alarmConditionStringValue(current.ResourceType),
		MetricName:           alarmConditionStringValue(current.MetricName),
		ConditionType:        string(current.ConditionType),
		Conditions:           alarmConditionStatusConditions(current.Conditions),
		Status:               string(current.Status),
		LifecycleState:       string(current.LifecycleState),
		CompositeType:        alarmConditionStringValue(current.CompositeType),
		TimeCreated:          alarmConditionSDKTimeString(current.TimeCreated),
		TimeUpdated:          alarmConditionSDKTimeString(current.TimeUpdated),
		FreeformTags:         maps.Clone(current.FreeformTags),
		DefinedTags:          alarmConditionStatusTags(current.DefinedTags),
		SystemTags:           alarmConditionStatusTags(current.SystemTags),
	}
	return nil
}

func markAlarmConditionTerminating(resource *stackmonitoringv1beta1.AlarmCondition, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	message := "OCI AlarmCondition delete is in progress"
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

func markAlarmConditionDeleted(resource *stackmonitoringv1beta1.AlarmCondition, message string) {
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

type alarmConditionCreateFollowUpClient struct {
	delegate AlarmConditionServiceClient
}

func (c alarmConditionCreateFollowUpClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", alarmConditionKind)
	}
	state := &alarmConditionCreateFollowUpState{}
	ctx = context.WithValue(ctx, alarmConditionCreateFollowUpContextKey{}, state)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c alarmConditionCreateFollowUpClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.AlarmCondition,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", alarmConditionKind)
	}
	return c.delegate.Delete(ctx, resource)
}

func alarmConditionFromResponse(response any) (stackmonitoringsdk.AlarmCondition, bool) {
	switch current := alarmConditionDereference(response).(type) {
	case nil:
		return stackmonitoringsdk.AlarmCondition{}, false
	case stackmonitoringsdk.AlarmCondition:
		return current, true
	case stackmonitoringsdk.AlarmConditionSummary:
		return alarmConditionFromSummary(current), true
	case stackmonitoringsdk.CreateAlarmConditionResponse:
		return current.AlarmCondition, true
	case stackmonitoringsdk.GetAlarmConditionResponse:
		return current.AlarmCondition, true
	case stackmonitoringsdk.UpdateAlarmConditionResponse:
		return current.AlarmCondition, true
	case alarmConditionRuntimeReadResponse:
		return alarmConditionFromStatusMap(current.Body)
	case map[string]any:
		return alarmConditionFromStatusMap(current)
	default:
		return stackmonitoringsdk.AlarmCondition{}, false
	}
}

func alarmConditionDereference(response any) any {
	value := reflect.ValueOf(response)
	if !value.IsValid() || value.Kind() != reflect.Pointer {
		return response
	}
	if value.IsNil() {
		return nil
	}
	return value.Elem().Interface()
}

func alarmConditionStatusMap(current stackmonitoringsdk.AlarmCondition) map[string]any {
	return map[string]any{
		"id":                   alarmConditionStringValue(current.Id),
		"monitoringTemplateId": alarmConditionStringValue(current.MonitoringTemplateId),
		"namespace":            alarmConditionStringValue(current.Namespace),
		"resourceType":         alarmConditionStringValue(current.ResourceType),
		"metricName":           alarmConditionStringValue(current.MetricName),
		"conditionType":        string(current.ConditionType),
		"conditions":           alarmConditionStatusConditions(current.Conditions),
		"sdkStatus":            string(current.Status),
		"lifecycleState":       string(current.LifecycleState),
		"compositeType":        alarmConditionStringValue(current.CompositeType),
		"timeCreated":          alarmConditionSDKTimeString(current.TimeCreated),
		"timeUpdated":          alarmConditionSDKTimeString(current.TimeUpdated),
		"freeformTags":         maps.Clone(current.FreeformTags),
		"definedTags":          alarmConditionStatusTags(current.DefinedTags),
		"systemTags":           alarmConditionStatusTags(current.SystemTags),
	}
}

func alarmConditionFromStatusMap(values map[string]any) (stackmonitoringsdk.AlarmCondition, bool) {
	if len(values) == 0 {
		return stackmonitoringsdk.AlarmCondition{}, false
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return stackmonitoringsdk.AlarmCondition{}, false
	}
	status := stackmonitoringv1beta1.AlarmConditionStatus{}
	if err := json.Unmarshal(payload, &status); err != nil {
		return stackmonitoringsdk.AlarmCondition{}, false
	}
	conditions, err := alarmConditionConditionsFromSpec(status.Conditions)
	if err != nil && len(status.Conditions) != 0 {
		return stackmonitoringsdk.AlarmCondition{}, false
	}
	return stackmonitoringsdk.AlarmCondition{
		Id:                   alarmConditionStringPointer(status.Id),
		MonitoringTemplateId: alarmConditionStringPointer(status.MonitoringTemplateId),
		Namespace:            alarmConditionStringPointer(status.Namespace),
		ResourceType:         alarmConditionStringPointer(status.ResourceType),
		MetricName:           alarmConditionStringPointer(status.MetricName),
		ConditionType:        stackmonitoringsdk.ConditionTypeEnum(status.ConditionType),
		Conditions:           conditions,
		Status:               stackmonitoringsdk.AlarmConditionLifeCycleDetailsEnum(status.Status),
		LifecycleState:       stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum(status.LifecycleState),
		CompositeType:        alarmConditionStringPointer(status.CompositeType),
		FreeformTags:         maps.Clone(status.FreeformTags),
		DefinedTags:          alarmConditionDefinedTagsFromSpec(status.DefinedTags),
		SystemTags:           alarmConditionDefinedTagsFromSpec(status.SystemTags),
	}, true
}

func alarmConditionFromSummary(summary stackmonitoringsdk.AlarmConditionSummary) stackmonitoringsdk.AlarmCondition {
	return stackmonitoringsdk.AlarmCondition{
		Id:                   summary.Id,
		MonitoringTemplateId: summary.MonitoringTemplateId,
		Namespace:            summary.Namespace,
		ResourceType:         summary.ResourceType,
		MetricName:           summary.MetricName,
		ConditionType:        summary.ConditionType,
		Conditions:           summary.Conditions,
		Status:               summary.Status,
		LifecycleState:       summary.LifecycleState,
		CompositeType:        summary.CompositeType,
		TimeCreated:          summary.TimeCreated,
		TimeUpdated:          summary.TimeUpdated,
		FreeformTags:         summary.FreeformTags,
		DefinedTags:          summary.DefinedTags,
		SystemTags:           summary.SystemTags,
	}
}

func alarmConditionConditionsFromSpec(
	specConditions []stackmonitoringv1beta1.AlarmConditionCondition,
) ([]stackmonitoringsdk.Condition, error) {
	if len(specConditions) == 0 {
		return nil, fmt.Errorf("%s spec.conditions is required", alarmConditionKind)
	}
	conditions := make([]stackmonitoringsdk.Condition, 0, len(specConditions))
	for index, condition := range specConditions {
		severity, err := alarmConditionSeverity(condition.Severity)
		if err != nil {
			return nil, fmt.Errorf("spec.conditions[%d]: %w", index, err)
		}
		query := strings.TrimSpace(condition.Query)
		if query == "" {
			return nil, fmt.Errorf("spec.conditions[%d].query is required", index)
		}
		converted := stackmonitoringsdk.Condition{
			Severity:         severity,
			Query:            common.String(query),
			ShouldAppendNote: common.Bool(condition.ShouldAppendNote),
			ShouldAppendUrl:  common.Bool(condition.ShouldAppendUrl),
		}
		if body := strings.TrimSpace(condition.Body); body != "" {
			converted.Body = common.String(body)
		}
		if triggerDelay := strings.TrimSpace(condition.TriggerDelay); triggerDelay != "" {
			converted.TriggerDelay = common.String(triggerDelay)
		}
		conditions = append(conditions, converted)
	}
	return conditions, nil
}

func alarmConditionConditionsEqual(left []stackmonitoringsdk.Condition, right []stackmonitoringsdk.Condition) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !alarmConditionConditionEqual(left[index], right[index]) {
			return false
		}
	}
	return true
}

func alarmConditionConditionEqual(left stackmonitoringsdk.Condition, right stackmonitoringsdk.Condition) bool {
	return left.Severity == right.Severity &&
		alarmConditionStringValue(left.Query) == alarmConditionStringValue(right.Query) &&
		alarmConditionStringValue(left.Body) == alarmConditionStringValue(right.Body) &&
		alarmConditionBoolValue(left.ShouldAppendNote) == alarmConditionBoolValue(right.ShouldAppendNote) &&
		alarmConditionBoolValue(left.ShouldAppendUrl) == alarmConditionBoolValue(right.ShouldAppendUrl) &&
		alarmConditionStringValue(left.TriggerDelay) == alarmConditionStringValue(right.TriggerDelay)
}

func alarmConditionStatusConditions(conditions []stackmonitoringsdk.Condition) []stackmonitoringv1beta1.AlarmConditionCondition {
	if len(conditions) == 0 {
		return nil
	}
	statusConditions := make([]stackmonitoringv1beta1.AlarmConditionCondition, 0, len(conditions))
	for _, condition := range conditions {
		statusConditions = append(statusConditions, stackmonitoringv1beta1.AlarmConditionCondition{
			Severity:         string(condition.Severity),
			Query:            alarmConditionStringValue(condition.Query),
			Body:             alarmConditionStringValue(condition.Body),
			ShouldAppendNote: alarmConditionBoolValue(condition.ShouldAppendNote),
			ShouldAppendUrl:  alarmConditionBoolValue(condition.ShouldAppendUrl),
			TriggerDelay:     alarmConditionStringValue(condition.TriggerDelay),
		})
	}
	return statusConditions
}

func alarmConditionConditionType(value string) (stackmonitoringsdk.ConditionTypeEnum, error) {
	conditionType, ok := stackmonitoringsdk.GetMappingConditionTypeEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported conditionType %q", value)
	}
	return conditionType, nil
}

func alarmConditionSeverity(value string) (stackmonitoringsdk.AlarmConditionSeverityEnum, error) {
	severity, ok := stackmonitoringsdk.GetMappingAlarmConditionSeverityEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported severity %q", value)
	}
	return severity, nil
}

func alarmConditionDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func alarmConditionStatusTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(tags) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = fmt.Sprint(value)
		}
		converted[namespace] = children
	}
	return converted
}

func alarmConditionTrackedID(resource *stackmonitoringv1beta1.AlarmCondition) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func alarmConditionMonitoringTemplateID(resource *stackmonitoringv1beta1.AlarmCondition) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyTrimAlarmCondition(
		resource.Status.MonitoringTemplateId,
		alarmConditionAnnotation(resource, alarmConditionMonitoringTemplateIDAnnotation),
	)
}

func alarmConditionAnnotation(resource *stackmonitoringv1beta1.AlarmCondition, key string) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.GetAnnotations()[key])
}

func alarmConditionStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func alarmConditionStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return common.String(trimmed)
}

func alarmConditionBoolValue(value *bool) bool {
	return value != nil && *value
}

func alarmConditionSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func firstNonEmptyTrimAlarmCondition(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func newAlarmConditionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client alarmConditionOCIClient,
) AlarmConditionServiceClient {
	hooks := newAlarmConditionRuntimeHooksWithOCIClient(client)
	applyAlarmConditionRuntimeHooks(&hooks)
	manager := &AlarmConditionServiceManager{Log: log}
	delegate := defaultAlarmConditionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.AlarmCondition](
			buildAlarmConditionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAlarmConditionGeneratedClient(hooks, delegate)
}

func newAlarmConditionRuntimeHooksWithOCIClient(client alarmConditionOCIClient) AlarmConditionRuntimeHooks {
	hooks := newAlarmConditionDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	hooks.Create.Call = func(ctx context.Context, request stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
		if client == nil {
			return stackmonitoringsdk.CreateAlarmConditionResponse{}, fmt.Errorf("%s OCI client is not configured", alarmConditionKind)
		}
		return client.CreateAlarmCondition(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
		if client == nil {
			return stackmonitoringsdk.GetAlarmConditionResponse{}, fmt.Errorf("%s OCI client is not configured", alarmConditionKind)
		}
		return client.GetAlarmCondition(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
		if client == nil {
			return stackmonitoringsdk.ListAlarmConditionsResponse{}, fmt.Errorf("%s OCI client is not configured", alarmConditionKind)
		}
		return client.ListAlarmConditions(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
		if client == nil {
			return stackmonitoringsdk.UpdateAlarmConditionResponse{}, fmt.Errorf("%s OCI client is not configured", alarmConditionKind)
		}
		return client.UpdateAlarmCondition(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
		if client == nil {
			return stackmonitoringsdk.DeleteAlarmConditionResponse{}, fmt.Errorf("%s OCI client is not configured", alarmConditionKind)
		}
		return client.DeleteAlarmCondition(ctx, request)
	}
	return hooks
}
