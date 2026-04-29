/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alarmsuppression

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	monitoringsdk "github.com/oracle/oci-go-sdk/v65/monitoring"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const alarmSuppressionDeletePendingMessage = "OCI alarm suppression delete is in progress"

func init() {
	registerAlarmSuppressionRuntimeHooksMutator(func(_ *AlarmSuppressionServiceManager, hooks *AlarmSuppressionRuntimeHooks) {
		hooks.Semantics = alarmSuppressionRuntimeSemantics()
		hooks.BuildCreateBody = buildAlarmSuppressionCreateBody
		hooks.Identity.GuardExistingBeforeCreate = guardAlarmSuppressionExistingBeforeCreate
		hooks.List.Fields = alarmSuppressionListFields()
		hooks.ParityHooks.NormalizeDesiredState = normalizeAlarmSuppressionDesiredState
		hooks.StatusHooks.MarkTerminating = markAlarmSuppressionTerminating
		hooks.DeleteHooks.ApplyOutcome = applyAlarmSuppressionDeleteOutcome
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapAlarmSuppressionCanonicalizingClient)
	})
}

func alarmSuppressionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "monitoring",
		FormalSlug:        "alarmsuppression",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"alarmSuppressionTarget.alarmId",
				"displayName",
				"dimensions",
				"timeSuppressFrom",
				"timeSuppressUntil",
				"description",
				"freeformTags",
				"definedTags",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew: []string{
				"alarmSuppressionTarget",
				"displayName",
				"dimensions",
				"timeSuppressFrom",
				"timeSuppressUntil",
				"description",
				"freeformTags",
				"definedTags",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
	}
}

func alarmSuppressionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "AlarmId",
			RequestName:  "alarmId",
			Contribution: "query",
			LookupPaths: []string{
				"alarmSuppressionTarget.alarmId",
				"spec.alarmSuppressionTarget.alarmId",
			},
		},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildAlarmSuppressionCreateBody(_ context.Context, resource *monitoringv1beta1.AlarmSuppression, _ string) (any, error) {
	target, err := alarmSuppressionSDKTarget(resource.Spec.AlarmSuppressionTarget)
	if err != nil {
		return nil, err
	}
	timeSuppressFrom, err := sdkTimeFromRFC3339("timeSuppressFrom", resource.Spec.TimeSuppressFrom)
	if err != nil {
		return nil, err
	}
	timeSuppressUntil, err := sdkTimeFromRFC3339("timeSuppressUntil", resource.Spec.TimeSuppressUntil)
	if err != nil {
		return nil, err
	}

	return monitoringsdk.CreateAlarmSuppressionDetails{
		AlarmSuppressionTarget: target,
		DisplayName:            common.String(resource.Spec.DisplayName),
		Dimensions:             cloneStringMap(resource.Spec.Dimensions),
		TimeSuppressFrom:       timeSuppressFrom,
		TimeSuppressUntil:      timeSuppressUntil,
		Description:            optionalString(resource.Spec.Description),
		FreeformTags:           cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:            sdkDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func guardAlarmSuppressionExistingBeforeCreate(_ context.Context, resource *monitoringv1beta1.AlarmSuppression) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if err := normalizeAlarmSuppressionSpec(resource); err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func normalizeAlarmSuppressionDesiredState(resource *monitoringv1beta1.AlarmSuppression, _ any) {
	if resource == nil {
		return
	}
	_ = normalizeAlarmSuppressionSpec(resource)
}

func normalizeAlarmSuppressionSpec(resource *monitoringv1beta1.AlarmSuppression) error {
	target, err := normalizeAlarmSuppressionTarget(resource.Spec.AlarmSuppressionTarget)
	if err != nil {
		return err
	}
	timeSuppressFrom, err := canonicalAlarmSuppressionTime("timeSuppressFrom", resource.Spec.TimeSuppressFrom)
	if err != nil {
		return err
	}
	timeSuppressUntil, err := canonicalAlarmSuppressionTime("timeSuppressUntil", resource.Spec.TimeSuppressUntil)
	if err != nil {
		return err
	}

	resource.Spec.AlarmSuppressionTarget = target
	resource.Spec.TimeSuppressFrom = timeSuppressFrom
	resource.Spec.TimeSuppressUntil = timeSuppressUntil
	return nil
}

type alarmSuppressionCanonicalizingClient struct {
	delegate AlarmSuppressionServiceClient
}

func wrapAlarmSuppressionCanonicalizingClient(delegate AlarmSuppressionServiceClient) AlarmSuppressionServiceClient {
	return alarmSuppressionCanonicalizingClient{delegate: delegate}
}

func (c alarmSuppressionCanonicalizingClient) CreateOrUpdate(
	ctx context.Context,
	resource *monitoringv1beta1.AlarmSuppression,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	normalizeAlarmSuppressionTimesIfValid(resource)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c alarmSuppressionCanonicalizingClient) Delete(ctx context.Context, resource *monitoringv1beta1.AlarmSuppression) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func normalizeAlarmSuppressionTimesIfValid(resource *monitoringv1beta1.AlarmSuppression) {
	if resource == nil {
		return
	}
	if value, err := canonicalAlarmSuppressionTime("timeSuppressFrom", resource.Spec.TimeSuppressFrom); err == nil {
		resource.Spec.TimeSuppressFrom = value
	}
	if value, err := canonicalAlarmSuppressionTime("timeSuppressUntil", resource.Spec.TimeSuppressUntil); err == nil {
		resource.Spec.TimeSuppressUntil = value
	}
}

func normalizeAlarmSuppressionTarget(target monitoringv1beta1.AlarmSuppressionTarget) (monitoringv1beta1.AlarmSuppressionTarget, error) {
	targetType := strings.ToUpper(strings.TrimSpace(target.TargetType))
	alarmID := strings.TrimSpace(target.AlarmId)

	if alarmID == "" && strings.TrimSpace(target.JsonData) != "" {
		var payload struct {
			TargetType string `json:"targetType"`
			AlarmID    string `json:"alarmId"`
		}
		if err := json.Unmarshal([]byte(target.JsonData), &payload); err != nil {
			return target, fmt.Errorf("parse alarmSuppressionTarget.jsonData: %w", err)
		}
		targetType = strings.ToUpper(strings.TrimSpace(payload.TargetType))
		alarmID = strings.TrimSpace(payload.AlarmID)
	}

	if targetType == "" {
		targetType = string(monitoringsdk.AlarmSuppressionTargetTargetTypeAlarm)
	}
	if targetType != string(monitoringsdk.AlarmSuppressionTargetTargetTypeAlarm) {
		return target, fmt.Errorf("unsupported alarmSuppressionTarget.targetType %q", targetType)
	}
	if alarmID == "" {
		return target, fmt.Errorf("alarmSuppressionTarget.alarmId is required")
	}

	target.TargetType = targetType
	target.AlarmId = alarmID
	target.JsonData = ""
	return target, nil
}

func alarmSuppressionSDKTarget(target monitoringv1beta1.AlarmSuppressionTarget) (monitoringsdk.AlarmSuppressionTarget, error) {
	normalized, err := normalizeAlarmSuppressionTarget(target)
	if err != nil {
		return nil, err
	}
	return monitoringsdk.AlarmSuppressionAlarmTarget{AlarmId: common.String(normalized.AlarmId)}, nil
}

func sdkTimeFromRFC3339(fieldName string, value string) (*common.SDKTime, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339: %w", fieldName, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func canonicalAlarmSuppressionTime(fieldName string, value string) (string, error) {
	sdkTime, err := sdkTimeFromRFC3339(fieldName, value)
	if err != nil {
		return "", err
	}
	return sdkTime.Time.Format(time.RFC3339Nano), nil
}

func applyAlarmSuppressionDeleteOutcome(
	resource *monitoringv1beta1.AlarmSuppression,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	lifecycleState := strings.ToUpper(alarmSuppressionLifecycleState(response))
	if lifecycleState != string(monitoringsdk.AlarmSuppressionLifecycleStateActive) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	if stage == generatedruntime.DeleteConfirmStageAlreadyPending &&
		!alarmSuppressionDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	if stage == generatedruntime.DeleteConfirmStageAfterRequest ||
		stage == generatedruntime.DeleteConfirmStageAlreadyPending {
		markAlarmSuppressionTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func alarmSuppressionDeleteAlreadyPending(resource *monitoringv1beta1.AlarmSuppression) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markAlarmSuppressionTerminating(resource *monitoringv1beta1.AlarmSuppression, response any) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = alarmSuppressionDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       alarmSuppressionLifecycleState(response),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         alarmSuppressionDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		alarmSuppressionDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func alarmSuppressionLifecycleState(response any) string {
	switch typed := response.(type) {
	case monitoringsdk.GetAlarmSuppressionResponse:
		return string(typed.AlarmSuppression.LifecycleState)
	case *monitoringsdk.GetAlarmSuppressionResponse:
		if typed == nil {
			return ""
		}
		return string(typed.AlarmSuppression.LifecycleState)
	case monitoringsdk.CreateAlarmSuppressionResponse:
		return string(typed.AlarmSuppression.LifecycleState)
	case *monitoringsdk.CreateAlarmSuppressionResponse:
		if typed == nil {
			return ""
		}
		return string(typed.AlarmSuppression.LifecycleState)
	case monitoringsdk.AlarmSuppression:
		return string(typed.LifecycleState)
	case *monitoringsdk.AlarmSuppression:
		if typed == nil {
			return ""
		}
		return string(typed.LifecycleState)
	case monitoringsdk.AlarmSuppressionSummary:
		return string(typed.LifecycleState)
	case *monitoringsdk.AlarmSuppressionSummary:
		if typed == nil {
			return ""
		}
		return string(typed.LifecycleState)
	default:
		return ""
	}
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sdkDefinedTags(in map[string]shared.MapValue) map[string]map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(in))
	for namespace, values := range in {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		out[namespace] = converted
	}
	return out
}
