/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package chargebackplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const defaultChargebackPlanRequeueDuration = time.Minute

type chargebackPlanListCall func(context.Context, opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error)

type chargebackPlanRuntimeClient struct {
	delegate ChargebackPlanServiceClient
	hooks    ChargebackPlanRuntimeHooks
}

type chargebackPlanDesiredCreateFields struct {
	FromJSON        bool
	JSONFields      map[string]bool
	CompartmentId   string
	PlanName        string
	PlanType        string
	EntitySource    string
	PlanDescription string
	FreeformTags    map[string]string
	DefinedTags     map[string]map[string]interface{}
	PlanCustomItems []opsisdk.CreatePlanCustomItemDetails
}

type chargebackPlanAuthShapedNotFoundError struct {
	err error
}

func (e chargebackPlanAuthShapedNotFoundError) Error() string {
	return fmt.Sprintf("ChargebackPlan delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e chargebackPlanAuthShapedNotFoundError) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func init() {
	registerChargebackPlanRuntimeHooksMutator(func(_ *ChargebackPlanServiceManager, hooks *ChargebackPlanRuntimeHooks) {
		applyChargebackPlanRuntimeHooks(hooks)
	})
}

func applyChargebackPlanRuntimeHooks(hooks *ChargebackPlanRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newChargebackPlanRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *opsiv1beta1.ChargebackPlan, _ string) (any, error) {
		return buildChargebackPlanCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *opsiv1beta1.ChargebackPlan, _ string, currentResponse any) (any, bool, error) {
		return buildChargebackPlanUpdateBody(resource, currentResponse)
	}
	hooks.List.Fields = chargebackPlanListFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateChargebackPlanCreateOnlyDrift
	hooks.DeleteHooks.HandleError = rejectChargebackPlanAuthShapedNotFound
	if hooks.List.Call != nil {
		hooks.List.Call = listChargebackPlanPages(hooks.List.Call)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ChargebackPlanServiceClient) ChargebackPlanServiceClient {
		return chargebackPlanRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
		}
	})
}

func (c chargebackPlanRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlan,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ChargebackPlan resource is nil")
	}

	currentID := chargebackPlanRecordedID(resource)
	if currentID != "" {
		current, found, err := c.getChargebackPlan(ctx, resource, currentID)
		if err != nil {
			return chargebackPlanFail(resource, err)
		}
		if found {
			return c.reconcileChargebackPlan(ctx, resource, currentID, current)
		}
		resource.Status.OsokStatus.Ocid = ""
		resource.Status.Id = ""
	}

	existing, found, err := c.lookupExistingChargebackPlan(ctx, resource)
	if err != nil {
		return chargebackPlanFail(resource, err)
	}
	if found {
		currentID = chargebackPlanString(existing.Id)
		if currentID == "" {
			return chargebackPlanFail(resource, fmt.Errorf("ChargebackPlan list response did not expose a resource OCID"))
		}
		chargebackPlanProjectSDK(resource, existing)
		resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
		current, currentFound, err := c.getChargebackPlan(ctx, resource, currentID)
		if err != nil {
			return chargebackPlanFail(resource, err)
		}
		if currentFound {
			return c.reconcileChargebackPlan(ctx, resource, currentID, current)
		}
	}

	return c.createChargebackPlan(ctx, resource, req)
}

func (c chargebackPlanRuntimeClient) Delete(ctx context.Context, resource *opsiv1beta1.ChargebackPlan) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("ChargebackPlan runtime client is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c chargebackPlanRuntimeClient) reconcileChargebackPlan(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlan,
	currentID string,
	current opsisdk.ChargebackPlan,
) (servicemanager.OSOKResponse, error) {
	if chargebackPlanLifecycleInProgress(current.LifecycleState) {
		return chargebackPlanProjectSuccess(resource, current, "", "")
	}
	if err := validateChargebackPlanImmutableFields(resource, current); err != nil {
		return chargebackPlanFail(resource, err)
	}

	body, updateNeeded, err := buildChargebackPlanUpdateBody(resource, current)
	if err != nil {
		return chargebackPlanFail(resource, err)
	}
	if !updateNeeded {
		return chargebackPlanProjectSuccess(resource, current, "", "")
	}

	response, err := c.hooks.Update.Call(ctx, opsisdk.UpdateChargebackPlanRequest{
		ChargebackplanId:            common.String(currentID),
		UpdateChargebackPlanDetails: body,
	})
	if err != nil {
		return chargebackPlanFail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	requestID := chargebackPlanString(response.OpcRequestId)
	workRequestID := chargebackPlanString(response.OpcWorkRequestId)

	refreshed, found, err := c.getChargebackPlan(ctx, resource, currentID)
	if err != nil {
		return chargebackPlanFail(resource, err)
	}
	if !found {
		result, err := chargebackPlanProjectSuccess(resource, current, shared.Updating, requestID)
		chargebackPlanSeedWorkRequest(resource, workRequestID)
		return result, err
	}
	result, err := chargebackPlanProjectSuccess(resource, refreshed, shared.Updating, requestID)
	chargebackPlanSeedWorkRequest(resource, workRequestID)
	return result, err
}

func (c chargebackPlanRuntimeClient) createChargebackPlan(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlan,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	body, err := buildChargebackPlanCreateBody(resource)
	if err != nil {
		return chargebackPlanFail(resource, err)
	}
	response, err := c.hooks.Create.Call(ctx, opsisdk.CreateChargebackPlanRequest{
		CreateChargebackPlanDetails: body,
		OpcRetryToken:               common.String(chargebackPlanRetryToken(resource, req.Namespace)),
	})
	if err != nil {
		return chargebackPlanFail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	requestID := chargebackPlanString(response.OpcRequestId)
	workRequestID := chargebackPlanString(response.OpcWorkRequestId)

	current := response.ChargebackPlan
	currentID := chargebackPlanString(current.Id)
	if currentID != "" {
		if refreshed, found, err := c.getChargebackPlan(ctx, resource, currentID); err != nil {
			return chargebackPlanFail(resource, err)
		} else if found {
			current = refreshed
		}
	}
	result, err := chargebackPlanProjectSuccess(resource, current, shared.Provisioning, requestID)
	chargebackPlanSeedWorkRequest(resource, workRequestID)
	return result, err
}

func (c chargebackPlanRuntimeClient) getChargebackPlan(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlan,
	currentID string,
) (opsisdk.ChargebackPlan, bool, error) {
	response, err := c.hooks.Get.Call(ctx, opsisdk.GetChargebackPlanRequest{
		ChargebackplanId: common.String(currentID),
	})
	if err != nil {
		err = rejectChargebackPlanAuthShapedNotFound(resource, err)
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return opsisdk.ChargebackPlan{}, false, nil
		}
		return opsisdk.ChargebackPlan{}, false, err
	}
	return response.ChargebackPlan, true, nil
}

func (c chargebackPlanRuntimeClient) lookupExistingChargebackPlan(
	ctx context.Context,
	resource *opsiv1beta1.ChargebackPlan,
) (opsisdk.ChargebackPlan, bool, error) {
	desired, err := chargebackPlanDesiredCreateFieldsForResource(resource)
	if err != nil {
		return opsisdk.ChargebackPlan{}, false, err
	}
	if desired.CompartmentId == "" || desired.PlanName == "" || desired.PlanType == "" || c.hooks.List.Call == nil {
		return opsisdk.ChargebackPlan{}, false, nil
	}

	response, err := c.hooks.List.Call(ctx, opsisdk.ListChargebackPlansRequest{
		CompartmentId: common.String(desired.CompartmentId),
	})
	if err != nil {
		err = rejectChargebackPlanAuthShapedNotFound(resource, err)
		return opsisdk.ChargebackPlan{}, false, err
	}

	matches := chargebackPlanMatchingExistingPlans(response.Items, desired)
	switch len(matches) {
	case 0:
		return opsisdk.ChargebackPlan{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return opsisdk.ChargebackPlan{}, false, fmt.Errorf("ChargebackPlan list response returned multiple matching resources for compartmentId %q and planName %q", desired.CompartmentId, desired.PlanName)
	}
}

func newChargebackPlanRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "chargebackplan",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(opsisdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "planName", "planType", "entitySource"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
				"planCustomItems",
				"planDescription",
				"planName",
			},
			ForceNew:      []string{"compartmentId", "entitySource", "jsonData", "planType"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ChargebackPlan", Action: "CreateChargebackPlan"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ChargebackPlan", Action: "UpdateChargebackPlan"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ChargebackPlan", Action: "DeleteChargebackPlan"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ChargebackPlan", Action: "GetChargebackPlan"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ChargebackPlan", Action: "GetChargebackPlan"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ChargebackPlan", Action: "GetChargebackPlan"}},
		},
	}
}

func chargebackPlanListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "ChargebackplanId", RequestName: "chargebackplanId", Contribution: "query", PreferResourceID: true},
	}
}

func chargebackPlanRecordedID(resource *opsiv1beta1.ChargebackPlan) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func chargebackPlanMatchesDesired(plan opsisdk.ChargebackPlan, desired chargebackPlanDesiredCreateFields) bool {
	if chargebackPlanString(plan.CompartmentId) != desired.CompartmentId {
		return false
	}
	if chargebackPlanString(plan.PlanName) != desired.PlanName {
		return false
	}
	if chargebackPlanString(plan.PlanType) != desired.PlanType {
		return false
	}
	if desired.EntitySource != "" && string(plan.EntitySource) != desired.EntitySource {
		return false
	}
	return true
}

func chargebackPlanMatchingExistingPlans(
	items []opsisdk.ChargebackPlanSummary,
	desired chargebackPlanDesiredCreateFields,
) []opsisdk.ChargebackPlan {
	var matches []opsisdk.ChargebackPlan
	for _, item := range items {
		current := chargebackPlanFromSummary(item)
		if !chargebackPlanCanBindExisting(current) || !chargebackPlanMatchesDesired(current, desired) {
			continue
		}
		matches = append(matches, current)
	}
	return matches
}

func chargebackPlanCanBindExisting(plan opsisdk.ChargebackPlan) bool {
	switch plan.LifecycleState {
	case opsisdk.LifecycleStateDeleting, opsisdk.LifecycleStateDeleted:
		return false
	default:
		return true
	}
}

func validateChargebackPlanImmutableFields(resource *opsiv1beta1.ChargebackPlan, current opsisdk.ChargebackPlan) error {
	if resource == nil {
		return fmt.Errorf("ChargebackPlan resource is nil")
	}
	desired, err := chargebackPlanDesiredCreateFieldsForResource(resource)
	if err != nil {
		return err
	}
	checks := []struct {
		field string
		want  string
		have  string
	}{
		{field: desired.fieldPath("compartmentId"), want: desired.CompartmentId, have: chargebackPlanString(current.CompartmentId)},
		{field: desired.fieldPath("planType"), want: desired.PlanType, have: chargebackPlanString(current.PlanType)},
	}
	if desired.EntitySource != "" {
		checks = append(checks, struct {
			field string
			want  string
			have  string
		}{field: desired.fieldPath("entitySource"), want: desired.EntitySource, have: string(current.EntitySource)})
	}
	for _, check := range checks {
		if check.want != "" && check.have != "" && check.want != check.have {
			return fmt.Errorf("ChargebackPlan formal semantics require replacement when %s changes", check.field)
		}
	}
	return validateChargebackPlanCreateOnlyDrift(resource, current)
}

func chargebackPlanProjectSuccess(
	resource *opsiv1beta1.ChargebackPlan,
	plan opsisdk.ChargebackPlan,
	fallback shared.OSOKConditionType,
	requestID string,
) (servicemanager.OSOKResponse, error) {
	chargebackPlanProjectSDK(resource, plan)
	status := &resource.Status.OsokStatus
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	servicemanager.SetOpcRequestID(status, requestID)

	condition, shouldRequeue := chargebackPlanCondition(plan.LifecycleState, fallback)
	message := chargebackPlanString(plan.LifecycleDetails)
	if message == "" {
		message = chargebackPlanDefaultMessage(condition)
	}

	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           chargebackPlanAsyncPhase(condition),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	} else {
		status.Async.Current = nil
	}

	conditionStatus := corev1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = corev1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, servicemanager.RuntimeDeps{}.Log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: defaultChargebackPlanRequeueDuration,
	}, nil
}

func chargebackPlanSeedWorkRequest(resource *opsiv1beta1.ChargebackPlan, workRequestID string) {
	if resource == nil || strings.TrimSpace(workRequestID) == "" || resource.Status.OsokStatus.Async.Current == nil {
		return
	}
	current := *resource.Status.OsokStatus.Async.Current
	current.Source = shared.OSOKAsyncSourceWorkRequest
	current.WorkRequestID = strings.TrimSpace(workRequestID)
	resource.Status.OsokStatus.Async.Current = &current
}

func chargebackPlanFail(resource *opsiv1beta1.ChargebackPlan, err error) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), servicemanager.RuntimeDeps{}.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func chargebackPlanCondition(state opsisdk.LifecycleStateEnum, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool) {
	switch state {
	case opsisdk.LifecycleStateCreating:
		return shared.Provisioning, true
	case opsisdk.LifecycleStateUpdating:
		return shared.Updating, true
	case opsisdk.LifecycleStateDeleting:
		return shared.Terminating, true
	case opsisdk.LifecycleStateFailed, opsisdk.LifecycleStateNeedsAttention:
		return shared.Failed, false
	case opsisdk.LifecycleStateActive:
		return shared.Active, false
	case opsisdk.LifecycleStateDeleted:
		return shared.Terminating, false
	default:
		if fallback != "" {
			return fallback, fallback == shared.Provisioning || fallback == shared.Updating || fallback == shared.Terminating
		}
		return shared.Active, false
	}
}

func chargebackPlanLifecycleInProgress(state opsisdk.LifecycleStateEnum) bool {
	switch state {
	case opsisdk.LifecycleStateCreating, opsisdk.LifecycleStateUpdating, opsisdk.LifecycleStateDeleting:
		return true
	default:
		return false
	}
}

func chargebackPlanAsyncPhase(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	switch condition {
	case shared.Provisioning:
		return shared.OSOKAsyncPhaseCreate
	case shared.Updating:
		return shared.OSOKAsyncPhaseUpdate
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return ""
	}
}

func chargebackPlanDefaultMessage(condition shared.OSOKConditionType) string {
	switch condition {
	case shared.Provisioning:
		return "OCI resource is provisioning"
	case shared.Updating:
		return "OCI resource is updating"
	case shared.Terminating:
		return "OCI resource delete is in progress"
	case shared.Failed:
		return "OCI resource is failed"
	default:
		return "OCI resource is active"
	}
}

func chargebackPlanProjectSDK(resource *opsiv1beta1.ChargebackPlan, plan opsisdk.ChargebackPlan) {
	resource.Status.Id = chargebackPlanString(plan.Id)
	resource.Status.CompartmentId = chargebackPlanString(plan.CompartmentId)
	resource.Status.PlanName = chargebackPlanString(plan.PlanName)
	resource.Status.PlanType = chargebackPlanString(plan.PlanType)
	resource.Status.LifecycleState = string(plan.LifecycleState)
	resource.Status.PlanDescription = chargebackPlanString(plan.PlanDescription)
	resource.Status.PlanCategory = string(plan.PlanCategory)
	if plan.IsCustomizable != nil {
		resource.Status.IsCustomizable = *plan.IsCustomizable
	}
	resource.Status.EntitySource = string(plan.EntitySource)
	resource.Status.LifecycleDetails = chargebackPlanString(plan.LifecycleDetails)
	resource.Status.FreeformTags = cloneChargebackPlanStringMap(plan.FreeformTags)
	resource.Status.DefinedTags = chargebackPlanStatusDefinedTags(plan.DefinedTags)
	resource.Status.SystemTags = chargebackPlanStatusDefinedTags(plan.SystemTags)
	resource.Status.PlanCustomItems = chargebackPlanStatusCustomItems(plan.PlanCustomItems)
	if plan.TimeCreated != nil {
		resource.Status.TimeCreated = plan.TimeCreated.String()
	}
	if plan.TimeUpdated != nil {
		resource.Status.TimeUpdated = plan.TimeUpdated.String()
	}
}

func chargebackPlanRetryToken(resource *opsiv1beta1.ChargebackPlan, namespace string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(namespace))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(resource.Name))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(resource.UID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(resource.Spec.CompartmentId))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(resource.Spec.PlanName))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(resource.Spec.PlanType))
	return hex.EncodeToString(h.Sum(nil))
}

func buildChargebackPlanCreateBody(resource *opsiv1beta1.ChargebackPlan) (opsisdk.CreateChargebackPlanDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("ChargebackPlan resource is nil")
	}

	if raw := strings.TrimSpace(resource.Spec.JsonData); raw != "" {
		details, err := chargebackPlanCreateDetailsFromJSON(raw)
		if err != nil {
			return nil, err
		}
		return chargebackPlanApplyCreateSpecDefaults(details, resource.Spec)
	}

	if _, err := normalizeChargebackPlanEntitySource(resource.Spec.EntitySource); err != nil {
		return nil, err
	}

	details := opsisdk.CreateChargebackPlanExadataDetails{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		PlanName:      common.String(resource.Spec.PlanName),
		PlanType:      common.String(resource.Spec.PlanType),
	}
	if resource.Spec.PlanDescription != "" {
		details.PlanDescription = common.String(resource.Spec.PlanDescription)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneChargebackPlanStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = chargebackPlanDefinedTags(resource.Spec.DefinedTags)
	}
	if resource.Spec.PlanCustomItems != nil {
		details.PlanCustomItems = chargebackPlanCustomItems(resource.Spec.PlanCustomItems)
	}
	if err := validateChargebackPlanCreateDetails(details); err != nil {
		return nil, err
	}
	return details, nil
}

func chargebackPlanApplyCreateSpecDefaults(
	details opsisdk.CreateChargebackPlanDetails,
	spec opsiv1beta1.ChargebackPlanSpec,
) (opsisdk.CreateChargebackPlanDetails, error) {
	typed, ok := details.(opsisdk.CreateChargebackPlanExadataDetails)
	if !ok {
		return nil, fmt.Errorf("unsupported ChargebackPlan create body %T", details)
	}
	if err := validateChargebackPlanJSONSpecConflicts(spec, typed); err != nil {
		return nil, err
	}
	chargebackPlanDefaultCreateIdentity(&typed, spec)
	chargebackPlanDefaultCreateMutableFields(&typed, spec)
	if err := validateChargebackPlanCreateDetails(typed); err != nil {
		return nil, err
	}
	return typed, nil
}

func chargebackPlanDefaultCreateIdentity(details *opsisdk.CreateChargebackPlanExadataDetails, spec opsiv1beta1.ChargebackPlanSpec) {
	if details.CompartmentId == nil {
		details.CompartmentId = common.String(spec.CompartmentId)
	}
	if details.PlanName == nil {
		details.PlanName = common.String(spec.PlanName)
	}
	if details.PlanType == nil {
		details.PlanType = common.String(spec.PlanType)
	}
}

func chargebackPlanDefaultCreateMutableFields(details *opsisdk.CreateChargebackPlanExadataDetails, spec opsiv1beta1.ChargebackPlanSpec) {
	if details.PlanDescription == nil && spec.PlanDescription != "" {
		details.PlanDescription = common.String(spec.PlanDescription)
	}
	if details.FreeformTags == nil && spec.FreeformTags != nil {
		details.FreeformTags = cloneChargebackPlanStringMap(spec.FreeformTags)
	}
	if details.DefinedTags == nil && spec.DefinedTags != nil {
		details.DefinedTags = chargebackPlanDefinedTags(spec.DefinedTags)
	}
	if details.PlanCustomItems == nil && spec.PlanCustomItems != nil {
		details.PlanCustomItems = chargebackPlanCustomItems(spec.PlanCustomItems)
	}
}

func validateChargebackPlanJSONSpecConflicts(
	spec opsiv1beta1.ChargebackPlanSpec,
	details opsisdk.CreateChargebackPlanExadataDetails,
) error {
	conflicts, err := chargebackPlanJSONSpecIdentityConflicts(spec, details)
	if err != nil {
		return err
	}
	conflicts = append(conflicts, chargebackPlanJSONSpecMutableConflicts(spec, details)...)
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("ChargebackPlan jsonData conflicts with spec field(s): %s", strings.Join(conflicts, ", "))
}

func chargebackPlanJSONSpecIdentityConflicts(
	spec opsiv1beta1.ChargebackPlanSpec,
	details opsisdk.CreateChargebackPlanExadataDetails,
) ([]string, error) {
	conflicts := []string{}
	for _, check := range []struct {
		field   string
		spec    string
		current *string
	}{
		{field: "compartmentId", spec: spec.CompartmentId, current: details.CompartmentId},
		{field: "planName", spec: spec.PlanName, current: details.PlanName},
		{field: "planType", spec: spec.PlanType, current: details.PlanType},
	} {
		conflicts = chargebackPlanAppendPointerStringConflict(conflicts, check.field, check.spec, check.current)
	}
	sourceConflict, err := chargebackPlanJSONSpecEntitySourceConflict(spec.EntitySource)
	if err != nil {
		return nil, err
	}
	if sourceConflict != "" {
		conflicts = append(conflicts, sourceConflict)
	}
	return conflicts, nil
}

func chargebackPlanJSONSpecEntitySourceConflict(specEntitySource string) (string, error) {
	if strings.TrimSpace(specEntitySource) == "" {
		return "", nil
	}
	source, err := normalizeChargebackPlanEntitySource(specEntitySource)
	if err != nil {
		return "", err
	}
	if source != opsisdk.ChargebackPlanEntitySourceChargebackExadata {
		return "entitySource", nil
	}
	return "", nil
}

func chargebackPlanJSONSpecMutableConflicts(
	spec opsiv1beta1.ChargebackPlanSpec,
	details opsisdk.CreateChargebackPlanExadataDetails,
) []string {
	conflicts := chargebackPlanAppendPointerStringConflict(nil, "planDescription", spec.PlanDescription, details.PlanDescription)
	conflicts = chargebackPlanAppendMapConflict(conflicts, "freeformTags", spec.FreeformTags, details.FreeformTags)
	conflicts = chargebackPlanAppendJSONConflict(conflicts, "definedTags", chargebackPlanDefinedTags(spec.DefinedTags), details.DefinedTags)
	return chargebackPlanAppendJSONConflict(conflicts, "planCustomItems", chargebackPlanCustomItems(spec.PlanCustomItems), details.PlanCustomItems)
}

func chargebackPlanAppendPointerStringConflict(conflicts []string, field string, spec string, current *string) []string {
	if current != nil && spec != "" && chargebackPlanString(current) != spec {
		return append(conflicts, field)
	}
	return conflicts
}

func chargebackPlanAppendMapConflict(conflicts []string, field string, spec map[string]string, current map[string]string) []string {
	if current != nil && spec != nil && !reflect.DeepEqual(current, spec) {
		return append(conflicts, field)
	}
	return conflicts
}

func chargebackPlanAppendJSONConflict(conflicts []string, field string, spec any, current any) []string {
	if !chargebackPlanIsNil(current) && !chargebackPlanIsNil(spec) && !chargebackPlanJSONEqual(current, spec) {
		return append(conflicts, field)
	}
	return conflicts
}

func validateChargebackPlanCreateDetails(details opsisdk.CreateChargebackPlanExadataDetails) error {
	var missing []string
	if chargebackPlanString(details.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if chargebackPlanString(details.PlanName) == "" {
		missing = append(missing, "planName")
	}
	if chargebackPlanString(details.PlanType) == "" {
		missing = append(missing, "planType")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("ChargebackPlan create details missing required field(s): %s", strings.Join(missing, ", "))
}

func chargebackPlanCreateDetailsFromJSON(raw string) (opsisdk.CreateChargebackPlanDetails, error) {
	var discriminator struct {
		EntitySource string `json:"entitySource"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode ChargebackPlan jsonData discriminator: %w", err)
	}
	if _, err := normalizeChargebackPlanEntitySource(discriminator.EntitySource); err != nil {
		return nil, err
	}

	var details opsisdk.CreateChargebackPlanExadataDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return nil, fmt.Errorf("decode ChargebackPlan CHARGEBACK_EXADATA jsonData: %w", err)
	}
	return details, nil
}

func normalizeChargebackPlanEntitySource(raw string) (opsisdk.ChargebackPlanEntitySourceEnum, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return opsisdk.ChargebackPlanEntitySourceChargebackExadata, nil
	}
	source, ok := opsisdk.GetMappingChargebackPlanEntitySourceEnum(raw)
	if !ok {
		return "", fmt.Errorf("unsupported ChargebackPlan entitySource %q", raw)
	}
	if source != opsisdk.ChargebackPlanEntitySourceChargebackExadata {
		return "", fmt.Errorf("unsupported ChargebackPlan entitySource %q; SDK create details are available only for %s", raw, opsisdk.ChargebackPlanEntitySourceChargebackExadata)
	}
	return source, nil
}

func buildChargebackPlanUpdateBody(
	resource *opsiv1beta1.ChargebackPlan,
	currentResponse any,
) (opsisdk.UpdateChargebackPlanDetails, bool, error) {
	if resource == nil {
		return opsisdk.UpdateChargebackPlanDetails{}, false, fmt.Errorf("ChargebackPlan resource is nil")
	}
	desired, err := chargebackPlanDesiredCreateFieldsForResource(resource)
	if err != nil {
		return opsisdk.UpdateChargebackPlanDetails{}, false, err
	}

	current, ok := chargebackPlanFromResponse(currentResponse)
	if !ok {
		return opsisdk.UpdateChargebackPlanDetails{}, false, fmt.Errorf("current ChargebackPlan response does not expose a ChargebackPlan body")
	}

	details := opsisdk.UpdateChargebackPlanDetails{}
	updateNeeded := false

	updateNeeded = chargebackPlanApplyStringUpdate(&details.PlanDescription, desired.PlanDescription, current.PlanDescription) || updateNeeded
	updateNeeded = chargebackPlanApplyStringUpdate(&details.PlanName, desired.PlanName, current.PlanName) || updateNeeded
	updateNeeded = chargebackPlanApplyTagUpdates(&details, desired, current) || updateNeeded
	updateNeeded = chargebackPlanApplyCustomItemUpdate(&details, desired, current) || updateNeeded

	return details, updateNeeded, nil
}

func chargebackPlanApplyStringUpdate(target **string, spec string, current *string) bool {
	desired, ok := chargebackPlanDesiredStringUpdate(spec, current)
	if !ok {
		return false
	}
	*target = desired
	return true
}

func chargebackPlanApplyTagUpdates(
	details *opsisdk.UpdateChargebackPlanDetails,
	desired chargebackPlanDesiredCreateFields,
	current opsisdk.ChargebackPlan,
) bool {
	updateNeeded := false
	if desired.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, desired.FreeformTags) {
		details.FreeformTags = cloneChargebackPlanStringMap(desired.FreeformTags)
		updateNeeded = true
	}
	if desired.DefinedTags != nil && !reflect.DeepEqual(current.DefinedTags, desired.DefinedTags) {
		details.DefinedTags = desired.DefinedTags
		updateNeeded = true
	}
	return updateNeeded
}

func chargebackPlanApplyCustomItemUpdate(
	details *opsisdk.UpdateChargebackPlanDetails,
	desired chargebackPlanDesiredCreateFields,
	current opsisdk.ChargebackPlan,
) bool {
	if desired.PlanCustomItems == nil || chargebackPlanJSONEqual(current.PlanCustomItems, desired.PlanCustomItems) {
		return false
	}
	details.PlanCustomItems = desired.PlanCustomItems
	return true
}

func chargebackPlanDesiredStringUpdate(spec string, current *string) (*string, bool) {
	if spec == "" {
		return nil, false
	}
	if current == nil || *current != spec {
		return common.String(spec), true
	}
	return nil, false
}

func chargebackPlanCustomItems(items []opsiv1beta1.ChargebackPlanPlanCustomItem) []opsisdk.CreatePlanCustomItemDetails {
	if items == nil {
		return nil
	}
	details := make([]opsisdk.CreatePlanCustomItemDetails, 0, len(items))
	for _, item := range items {
		detail := opsisdk.CreatePlanCustomItemDetails{
			IsCustomizable: common.Bool(item.IsCustomizable),
		}
		if item.Name != "" {
			detail.Name = common.String(item.Name)
		}
		if item.Value != "" {
			detail.Value = common.String(item.Value)
		}
		details = append(details, detail)
	}
	return details
}

func chargebackPlanDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		converted[namespace] = child
	}
	return converted
}

func chargebackPlanStatusDefinedTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		child := make(shared.MapValue, len(values))
		for key, value := range values {
			child[key] = fmt.Sprint(value)
		}
		converted[namespace] = child
	}
	return converted
}

func chargebackPlanStatusCustomItems(items []opsisdk.CreatePlanCustomItemDetails) []opsiv1beta1.ChargebackPlanPlanCustomItem {
	if items == nil {
		return nil
	}
	converted := make([]opsiv1beta1.ChargebackPlanPlanCustomItem, 0, len(items))
	for _, item := range items {
		statusItem := opsiv1beta1.ChargebackPlanPlanCustomItem{
			Name:  chargebackPlanString(item.Name),
			Value: chargebackPlanString(item.Value),
		}
		if item.IsCustomizable != nil {
			statusItem.IsCustomizable = *item.IsCustomizable
		}
		converted = append(converted, statusItem)
	}
	return converted
}

func cloneChargebackPlanStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneChargebackPlanNestedMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		cloned[namespace] = child
	}
	return cloned
}

func cloneChargebackPlanCustomItems(source []opsisdk.CreatePlanCustomItemDetails) []opsisdk.CreatePlanCustomItemDetails {
	if source == nil {
		return nil
	}
	cloned := make([]opsisdk.CreatePlanCustomItemDetails, len(source))
	copy(cloned, source)
	return cloned
}

func chargebackPlanFromResponse(currentResponse any) (opsisdk.ChargebackPlan, bool) {
	if current, ok := chargebackPlanFromDirectResponse(currentResponse); ok {
		return current, true
	}
	return chargebackPlanFromPointerResponse(currentResponse)
}

func chargebackPlanFromDirectResponse(currentResponse any) (opsisdk.ChargebackPlan, bool) {
	switch current := currentResponse.(type) {
	case opsisdk.ChargebackPlan:
		return current, true
	case opsisdk.ChargebackPlanSummary:
		return chargebackPlanFromSummary(current), true
	case opsisdk.CreateChargebackPlanResponse:
		return current.ChargebackPlan, true
	case opsisdk.GetChargebackPlanResponse:
		return current.ChargebackPlan, true
	default:
		return opsisdk.ChargebackPlan{}, false
	}
}

func chargebackPlanFromPointerResponse(currentResponse any) (opsisdk.ChargebackPlan, bool) {
	switch current := currentResponse.(type) {
	case *opsisdk.ChargebackPlan:
		if current == nil {
			return opsisdk.ChargebackPlan{}, false
		}
		return *current, true
	case *opsisdk.ChargebackPlanSummary:
		if current == nil {
			return opsisdk.ChargebackPlan{}, false
		}
		return chargebackPlanFromSummary(*current), true
	case *opsisdk.CreateChargebackPlanResponse:
		if current == nil {
			return opsisdk.ChargebackPlan{}, false
		}
		return current.ChargebackPlan, true
	case *opsisdk.GetChargebackPlanResponse:
		if current == nil {
			return opsisdk.ChargebackPlan{}, false
		}
		return current.ChargebackPlan, true
	default:
		return opsisdk.ChargebackPlan{}, false
	}
}

func chargebackPlanFromSummary(summary opsisdk.ChargebackPlanSummary) opsisdk.ChargebackPlan {
	return opsisdk.ChargebackPlan{
		Id:               summary.Id,
		CompartmentId:    summary.CompartmentId,
		PlanName:         summary.PlanName,
		PlanType:         summary.PlanType,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		PlanDescription:  summary.PlanDescription,
		PlanCategory:     summary.PlanCategory,
		IsCustomizable:   summary.IsCustomizable,
		EntitySource:     summary.EntitySource,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleDetails: summary.LifecycleDetails,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		SystemTags:       summary.SystemTags,
		PlanCustomItems:  summary.PlanCustomItems,
	}
}

func validateChargebackPlanCreateOnlyDrift(resource *opsiv1beta1.ChargebackPlan, currentResponse any) error {
	if resource == nil {
		return nil
	}

	current, ok := chargebackPlanFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current ChargebackPlan response does not expose a ChargebackPlan body")
	}

	desired, err := chargebackPlanDesiredCreateFieldsForResource(resource)
	if err != nil {
		return err
	}

	drift := chargebackPlanCreateOnlyDrift(desired, current)
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("ChargebackPlan create-only field drift detected for %s; recreate the resource instead of updating immutable fields", strings.Join(drift, ", "))
}

func chargebackPlanCreateOnlyDrift(desired chargebackPlanDesiredCreateFields, current opsisdk.ChargebackPlan) []string {
	drift := []string{}
	drift = chargebackPlanAppendStringDrift(drift, desired.fieldPath("compartmentId"), desired.CompartmentId, chargebackPlanString(current.CompartmentId))
	drift = chargebackPlanAppendStringDrift(drift, desired.fieldPath("planType"), desired.PlanType, chargebackPlanString(current.PlanType))
	drift = chargebackPlanAppendStringDrift(drift, desired.fieldPath("entitySource"), desired.EntitySource, string(current.EntitySource))
	if desired.FromJSON {
		drift = append(drift, validateChargebackPlanJSONDrift(desired, current)...)
	}
	return drift
}

func validateChargebackPlanJSONDrift(desired chargebackPlanDesiredCreateFields, current opsisdk.ChargebackPlan) []string {
	drift := []string{}
	for _, check := range []struct {
		field   string
		desired string
		current string
	}{
		{field: "planName", desired: desired.PlanName, current: chargebackPlanString(current.PlanName)},
		{field: "planDescription", desired: desired.PlanDescription, current: chargebackPlanString(current.PlanDescription)},
	} {
		if desired.JSONFields[check.field] {
			drift = chargebackPlanAppendStringDrift(drift, desired.fieldPath(check.field), check.desired, check.current)
		}
	}
	drift = chargebackPlanAppendJSONDrift(drift, desired, "freeformTags", desired.FreeformTags, current.FreeformTags)
	drift = chargebackPlanAppendJSONDrift(drift, desired, "definedTags", desired.DefinedTags, current.DefinedTags)
	drift = chargebackPlanAppendJSONDrift(drift, desired, "planCustomItems", desired.PlanCustomItems, current.PlanCustomItems)
	return drift
}

func chargebackPlanAppendStringDrift(drift []string, field string, desired string, current string) []string {
	if desired != "" && current != "" && desired != current {
		return append(drift, field)
	}
	return drift
}

func chargebackPlanAppendJSONDrift(
	drift []string,
	desired chargebackPlanDesiredCreateFields,
	field string,
	desiredValue any,
	currentValue any,
) []string {
	if !desired.JSONFields[field] || chargebackPlanIsNil(desiredValue) || chargebackPlanIsNil(currentValue) {
		return drift
	}
	if chargebackPlanJSONEqual(desiredValue, currentValue) {
		return drift
	}
	return append(drift, desired.fieldPath(field))
}

func chargebackPlanDesiredCreateFieldsForResource(resource *opsiv1beta1.ChargebackPlan) (chargebackPlanDesiredCreateFields, error) {
	if resource == nil {
		return chargebackPlanDesiredCreateFields{}, fmt.Errorf("ChargebackPlan resource is nil")
	}
	rawJSON := strings.TrimSpace(resource.Spec.JsonData)
	details, err := buildChargebackPlanCreateBody(resource)
	if err != nil {
		return chargebackPlanDesiredCreateFields{}, err
	}
	exadata, ok := details.(opsisdk.CreateChargebackPlanExadataDetails)
	if !ok {
		return chargebackPlanDesiredCreateFields{}, fmt.Errorf("unsupported ChargebackPlan create body %T", details)
	}

	jsonFields, err := chargebackPlanJSONFields(rawJSON)
	if err != nil {
		return chargebackPlanDesiredCreateFields{}, err
	}
	desired := chargebackPlanDesiredCreateFields{
		FromJSON:        rawJSON != "",
		JSONFields:      jsonFields,
		CompartmentId:   chargebackPlanString(exadata.CompartmentId),
		PlanName:        chargebackPlanString(exadata.PlanName),
		PlanType:        chargebackPlanString(exadata.PlanType),
		EntitySource:    string(opsisdk.ChargebackPlanEntitySourceChargebackExadata),
		PlanDescription: chargebackPlanString(exadata.PlanDescription),
		FreeformTags:    cloneChargebackPlanStringMap(exadata.FreeformTags),
		DefinedTags:     cloneChargebackPlanNestedMap(exadata.DefinedTags),
		PlanCustomItems: cloneChargebackPlanCustomItems(exadata.PlanCustomItems),
	}
	return desired, nil
}

func chargebackPlanJSONFields(raw string) (map[string]bool, error) {
	fields := map[string]bool{}
	if strings.TrimSpace(raw) == "" {
		return fields, nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode ChargebackPlan jsonData: %w", err)
	}
	for field := range values {
		fields[field] = true
	}
	return fields, nil
}

func (d chargebackPlanDesiredCreateFields) fieldPath(field string) string {
	if d.FromJSON && d.JSONFields[field] {
		return "jsonData." + field
	}
	return field
}

func rejectChargebackPlanAuthShapedNotFound(resource *opsiv1beta1.ChargebackPlan, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return chargebackPlanAuthShapedNotFoundError{err: err}
}

func listChargebackPlanPages(call chargebackPlanListCall) chargebackPlanListCall {
	return func(ctx context.Context, request opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error) {
		seenPages := map[string]struct{}{}
		var combined opsisdk.ListChargebackPlansResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return opsisdk.ListChargebackPlansResponse{}, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)

			nextPage := chargebackPlanString(response.OpcNextPage)
			if nextPage == "" {
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return opsisdk.ListChargebackPlansResponse{}, fmt.Errorf("ChargebackPlan list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func chargebackPlanString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func chargebackPlanJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}

	var leftDecoded any
	var rightDecoded any
	if err := json.Unmarshal(leftPayload, &leftDecoded); err != nil {
		return reflect.DeepEqual(left, right)
	}
	if err := json.Unmarshal(rightPayload, &rightDecoded); err != nil {
		return reflect.DeepEqual(left, right)
	}
	return reflect.DeepEqual(leftDecoded, rightDecoded)
}

func chargebackPlanIsNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
