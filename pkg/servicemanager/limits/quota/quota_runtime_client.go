/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package quota

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	limitssdk "github.com/oracle/oci-go-sdk/v65/limits"
	limitsv1beta1 "github.com/oracle/oci-service-operator/api/limits/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerQuotaRuntimeHooksMutator(func(manager *QuotaServiceManager, hooks *QuotaRuntimeHooks) {
		applyQuotaRuntimeHooks(manager, hooks)
	})
}

type quotaRuntimeClient struct {
	delegate QuotaServiceClient
	hooks    QuotaRuntimeHooks
	log      loggerutil.OSOKLogger
}

var _ QuotaServiceClient = (*quotaRuntimeClient)(nil)

func applyQuotaRuntimeHooks(manager *QuotaServiceManager, hooks *QuotaRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newQuotaRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *limitsv1beta1.Quota, _ string) (any, error) {
		return buildQuotaCreateDetails(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *limitsv1beta1.Quota, _ string, currentResponse any) (any, bool, error) {
		return buildQuotaUpdateDetails(resource, currentResponse)
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeQuotaDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateQuotaCreateOnlyDrift
	hooks.DeleteHooks.HandleError = rejectQuotaAuthShapedNotFound
	hooks.Read.List = quotaPaginatedListReadOperation(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate QuotaServiceClient) QuotaServiceClient {
		runtimeClient := &quotaRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func newQuotaRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "limits",
		FormalSlug:        "quota",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(limitssdk.QuotaLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "best-effort",
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"definedTags", "description", "freeformTags", "statements"},
			ForceNew: []string{"compartmentId", "name"},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
		},
	}
}

func (c *quotaRuntimeClient) CreateOrUpdate(ctx context.Context, resource *limitsv1beta1.Quota, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("quota runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *quotaRuntimeClient) Delete(ctx context.Context, resource *limitsv1beta1.Quota) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("quota resource is nil")
	}
	if c == nil {
		return false, fmt.Errorf("quota runtime client is not configured")
	}

	current, found, err := c.resolveQuotaForDelete(ctx, resource)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	c.projectStatus(resource, current)

	currentID := stringValue(current.Id)
	response, err := c.hooks.Delete.Call(ctx, limitssdk.DeleteQuotaRequest{QuotaId: commonString(currentID)})
	if err != nil {
		if quotaIsUnambiguousNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, rejectQuotaAuthShapedNotFound(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current, found, err = c.getQuotaForDelete(ctx, resource, currentID)
	if err != nil {
		return false, err
	}
	if !found {
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	c.projectStatus(resource, current)
	c.markTerminating(resource, "OCI resource delete is in progress")
	return false, nil
}

func (c *quotaRuntimeClient) resolveQuotaForDelete(ctx context.Context, resource *limitsv1beta1.Quota) (limitssdk.Quota, bool, error) {
	if currentID := currentQuotaID(resource); currentID != "" {
		return c.getQuotaForDelete(ctx, resource, currentID)
	}

	if c.hooks.List.Call == nil {
		return limitssdk.Quota{}, false, nil
	}
	response, err := listQuotaPages(ctx, c.hooks.List.Call, limitssdk.ListQuotasRequest{
		CompartmentId: commonString(resource.Spec.CompartmentId),
		Name:          commonString(resource.Spec.Name),
	})
	if err != nil {
		if quotaIsUnambiguousNotFound(err) {
			return limitssdk.Quota{}, false, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return limitssdk.Quota{}, false, rejectQuotaAuthShapedNotFound(resource, err)
	}

	matches := make([]limitssdk.QuotaSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if quotaSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return limitssdk.Quota{}, false, nil
	case 1:
		return c.getQuotaForDelete(ctx, resource, stringValue(matches[0].Id))
	default:
		return limitssdk.Quota{}, false, fmt.Errorf("multiple OCI Quotas matched compartmentId %q and name %q", resource.Spec.CompartmentId, resource.Spec.Name)
	}
}

func (c *quotaRuntimeClient) getQuotaForDelete(ctx context.Context, resource *limitsv1beta1.Quota, quotaID string) (limitssdk.Quota, bool, error) {
	if quotaID == "" || c.hooks.Get.Call == nil {
		return limitssdk.Quota{}, false, nil
	}
	response, err := c.hooks.Get.Call(ctx, limitssdk.GetQuotaRequest{QuotaId: commonString(quotaID)})
	if err != nil {
		if quotaIsUnambiguousNotFound(err) {
			return limitssdk.Quota{}, false, nil
		}
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return limitssdk.Quota{}, false, rejectQuotaAuthShapedNotFound(resource, err)
	}
	return response.Quota, true, nil
}

func currentQuotaID(resource *limitsv1beta1.Quota) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func quotaSummaryMatchesSpec(summary limitssdk.QuotaSummary, spec limitsv1beta1.QuotaSpec) bool {
	return stringValue(summary.CompartmentId) == spec.CompartmentId && stringValue(summary.Name) == spec.Name
}

func quotaIsUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func (c *quotaRuntimeClient) projectStatus(resource *limitsv1beta1.Quota, current limitssdk.Quota) {
	if resource == nil {
		return
	}
	resource.Status.Id = stringValue(current.Id)
	resource.Status.CompartmentId = stringValue(current.CompartmentId)
	resource.Status.Name = stringValue(current.Name)
	resource.Status.Statements = append([]string(nil), current.Statements...)
	resource.Status.Description = stringValue(current.Description)
	resource.Status.Locks = quotaStatusLocksFromSDK(current.Locks)
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.FreeformTags = maps.Clone(current.FreeformTags)
	resource.Status.DefinedTags = quotaStatusDefinedTagsFromSDK(current.DefinedTags)
	if current.TimeCreated != nil {
		resource.Status.TimeCreated = current.TimeCreated.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func (c *quotaRuntimeClient) markDeleted(resource *limitsv1beta1.Quota, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Async.Current = nil
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *quotaRuntimeClient) markTerminating(resource *limitsv1beta1.Quota, message string) {
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
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func quotaPaginatedListReadOperation(hooks *QuotaRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}

	listCall := hooks.List.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &limitssdk.ListQuotasRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*limitssdk.ListQuotasRequest)
			if !ok {
				return nil, fmt.Errorf("expected *limits.ListQuotasRequest, got %T", request)
			}
			return listQuotaPages(ctx, listCall, *typed)
		},
	}
}

func listQuotaPages(
	ctx context.Context,
	call func(context.Context, limitssdk.ListQuotasRequest) (limitssdk.ListQuotasResponse, error),
	request limitssdk.ListQuotasRequest,
) (limitssdk.ListQuotasResponse, error) {
	if call == nil {
		return limitssdk.ListQuotasResponse{}, fmt.Errorf("quota list operation is not configured")
	}

	seenPages := map[string]struct{}{}
	var combined limitssdk.ListQuotasResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return limitssdk.ListQuotasResponse{}, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := stringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return limitssdk.ListQuotasResponse{}, fmt.Errorf("quota list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

func buildQuotaCreateDetails(resource *limitsv1beta1.Quota) (limitssdk.CreateQuotaDetails, error) {
	if resource == nil {
		return limitssdk.CreateQuotaDetails{}, fmt.Errorf("quota resource is nil")
	}

	spec := resource.Spec
	details := limitssdk.CreateQuotaDetails{
		CompartmentId: commonString(spec.CompartmentId),
		Description:   commonString(spec.Description),
		Name:          commonString(spec.Name),
		Statements:    append([]string(nil), spec.Statements...),
		FreeformTags:  maps.Clone(spec.FreeformTags),
		DefinedTags:   quotaDefinedTagsFromSpec(spec.DefinedTags),
	}
	if spec.Locks != nil {
		details.Locks = quotaAddLocksFromSpec(spec.Locks)
	}
	return details, nil
}

func buildQuotaUpdateDetails(
	resource *limitsv1beta1.Quota,
	currentResponse any,
) (limitssdk.UpdateQuotaDetails, bool, error) {
	if resource == nil {
		return limitssdk.UpdateQuotaDetails{}, false, fmt.Errorf("quota resource is nil")
	}

	current, err := quotaRuntimeBody(currentResponse)
	if err != nil {
		return limitssdk.UpdateQuotaDetails{}, false, err
	}

	spec := resource.Spec
	details := limitssdk.UpdateQuotaDetails{}
	updateNeeded := false

	if desired, ok := quotaDesiredStringUpdate(spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := quotaDesiredStatementsUpdate(spec.Statements, current.Statements); ok {
		details.Statements = desired
		updateNeeded = true
	}
	if desired, ok := quotaDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := quotaDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func quotaRuntimeBody(currentResponse any) (limitssdk.Quota, error) {
	currentResponse, err := dereferenceQuotaRuntimeBody(currentResponse)
	if err != nil {
		return limitssdk.Quota{}, err
	}

	switch current := currentResponse.(type) {
	case limitssdk.Quota:
		return current, nil
	case limitssdk.QuotaSummary:
		return quotaFromSummary(current), nil
	case limitssdk.CreateQuotaResponse:
		return current.Quota, nil
	case limitssdk.GetQuotaResponse:
		return current.Quota, nil
	case limitssdk.UpdateQuotaResponse:
		return current.Quota, nil
	default:
		return limitssdk.Quota{}, fmt.Errorf("unexpected current quota response type %T", currentResponse)
	}
}

func dereferenceQuotaRuntimeBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case *limitssdk.Quota:
		return quotaDereferenceRuntimeBody(current)
	case *limitssdk.QuotaSummary:
		return quotaDereferenceRuntimeBody(current)
	case *limitssdk.CreateQuotaResponse:
		return quotaDereferenceRuntimeBody(current)
	case *limitssdk.GetQuotaResponse:
		return quotaDereferenceRuntimeBody(current)
	case *limitssdk.UpdateQuotaResponse:
		return quotaDereferenceRuntimeBody(current)
	default:
		return currentResponse, nil
	}
}

func quotaDereferenceRuntimeBody[T any](current *T) (T, error) {
	if current == nil {
		var zero T
		return zero, fmt.Errorf("current quota response is nil")
	}
	return *current, nil
}

func quotaFromSummary(summary limitssdk.QuotaSummary) limitssdk.Quota {
	return limitssdk.Quota{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		Name:           summary.Name,
		Description:    summary.Description,
		TimeCreated:    summary.TimeCreated,
		Locks:          summary.Locks,
		LifecycleState: limitssdk.QuotaLifecycleStateEnum(summary.LifecycleState),
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
	}
}

func normalizeQuotaDesiredState(resource *limitsv1beta1.Quota, currentResponse any) {
	if resource == nil || resource.Spec.Locks == nil {
		return
	}
	current, err := quotaRuntimeBody(currentResponse)
	if err != nil {
		return
	}
	if quotaLocksEqual(resource.Spec.Locks, current.Locks) {
		resource.Spec.Locks = nil
	}
}

func validateQuotaCreateOnlyDrift(resource *limitsv1beta1.Quota, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("quota resource is nil")
	}
	current, err := quotaRuntimeBody(currentResponse)
	if err != nil {
		return err
	}

	drift := quotaCreateOnlyDriftFields(resource.Spec, current)
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("quota create-only drift detected for %s; replace the resource or restore the desired spec before update", strings.Join(drift, ", "))
}

func quotaCreateOnlyDriftFields(spec limitsv1beta1.QuotaSpec, current limitssdk.Quota) []string {
	checks := []struct {
		name    string
		desired string
		current *string
	}{
		{name: "compartmentId", desired: spec.CompartmentId, current: current.CompartmentId},
		{name: "name", desired: spec.Name, current: current.Name},
	}

	var drift []string
	for _, check := range checks {
		if quotaHasStringCreateOnlyDrift(check.desired, stringValue(check.current)) {
			drift = append(drift, check.name)
		}
	}
	if quotaHasLocksCreateOnlyDrift(spec.Locks, current.Locks) {
		drift = append(drift, "locks")
	}
	return drift
}

func quotaHasStringCreateOnlyDrift(desired, current string) bool {
	return desired != "" && current != "" && desired != current
}

func quotaHasLocksCreateOnlyDrift(desired []limitsv1beta1.QuotaLock, current []limitssdk.ResourceLock) bool {
	return desired != nil && !quotaLocksEqual(desired, current)
}

func rejectQuotaAuthShapedNotFound(_ *limitsv1beta1.Quota, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return fmt.Errorf("quota delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func quotaDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := stringValue(current)
	if spec == currentValue {
		return nil, false
	}
	return commonString(spec), true
}

func quotaDesiredStatementsUpdate(spec []string, current []string) ([]string, bool) {
	if slices.Equal(spec, current) {
		return nil, false
	}
	return append([]string(nil), spec...), true
}

func quotaDesiredFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func quotaDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := quotaDefinedTagsFromSpec(spec)
	if quotaDefinedTagsEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func quotaDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func quotaDefinedTagsEqual(left map[string]map[string]interface{}, right map[string]map[string]interface{}) bool {
	if len(left) != len(right) {
		return false
	}
	for namespace, leftValues := range left {
		rightValues, ok := right[namespace]
		if !ok || len(leftValues) != len(rightValues) {
			return false
		}
		for key, leftValue := range leftValues {
			if fmt.Sprint(leftValue) != fmt.Sprint(rightValues[key]) {
				return false
			}
		}
	}
	return true
}

func quotaAddLocksFromSpec(locks []limitsv1beta1.QuotaLock) []limitssdk.AddLockDetails {
	converted := make([]limitssdk.AddLockDetails, 0, len(locks))
	for _, lock := range locks {
		converted = append(converted, limitssdk.AddLockDetails{
			Type:              limitssdk.AddLockDetailsTypeEnum(lock.Type),
			RelatedResourceId: optionalString(lock.RelatedResourceId),
			Message:           optionalString(lock.Message),
		})
	}
	return converted
}

func quotaStatusLocksFromSDK(locks []limitssdk.ResourceLock) []limitsv1beta1.QuotaLock {
	converted := make([]limitsv1beta1.QuotaLock, 0, len(locks))
	for _, lock := range locks {
		converted = append(converted, limitsv1beta1.QuotaLock{
			Type:              string(lock.Type),
			RelatedResourceId: stringValue(lock.RelatedResourceId),
			Message:           stringValue(lock.Message),
		})
	}
	return converted
}

func quotaLocksEqual(spec []limitsv1beta1.QuotaLock, current []limitssdk.ResourceLock) bool {
	if len(spec) != len(current) {
		return false
	}
	for index, lock := range spec {
		if lock.Type != string(current[index].Type) ||
			lock.RelatedResourceId != stringValue(current[index].RelatedResourceId) ||
			lock.Message != stringValue(current[index].Message) {
			return false
		}
	}
	return true
}

func quotaStatusDefinedTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
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

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return commonString(value)
}

func commonString(value string) *string {
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
