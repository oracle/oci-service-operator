/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inclusioncriterion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	governancerulescontrolplanesdk "github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type inclusionCriterionIdentity struct {
	governanceRuleID string
	criterionType    string
	associationType  string
	tenancyID        string
}

type inclusionCriterionAssociationKey struct {
	associationType string
	tenancyID       string
}

type ambiguousInclusionCriterionNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousInclusionCriterionNotFoundError) Error() string {
	return e.message
}

func (e ambiguousInclusionCriterionNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerInclusionCriterionRuntimeHooksMutator(func(_ *InclusionCriterionServiceManager, hooks *InclusionCriterionRuntimeHooks) {
		applyInclusionCriterionRuntimeHooks(hooks)
	})
}

func applyInclusionCriterionRuntimeHooks(hooks *InclusionCriterionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = inclusionCriterionRuntimeSemantics()
	hooks.BuildCreateBody = buildInclusionCriterionCreateBody
	hooks.Identity.Resolve = resolveInclusionCriterionIdentity
	hooks.Identity.RecordPath = recordInclusionCriterionPathIdentity
	hooks.Identity.RecordTracked = recordTrackedInclusionCriterionIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardInclusionCriterionExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedInclusionCriterionIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateInclusionCriterionCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = inclusionCriterionDeleteConfirmRead(hooks.Get.Call)
	hooks.DeleteHooks.HandleError = rejectInclusionCriterionAuthShapedNotFound
	hooks.DeleteHooks.ApplyOutcome = applyInclusionCriterionDeleteOutcome
}

func inclusionCriterionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "governancerulescontrolplane",
		FormalSlug:        "inclusioncriterion",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(governancerulescontrolplanesdk.InclusionCriterionLifecycleStateDeleted)},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"governanceRuleId"},
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
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func buildInclusionCriterionCreateBody(
	_ context.Context,
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	_ string,
) (any, error) {
	if resource == nil {
		return governancerulescontrolplanesdk.CreateInclusionCriterionDetails{}, fmt.Errorf("inclusion criterion resource is nil")
	}
	return inclusionCriterionCreateDetailsFromSpec(resource.Spec)
}

func inclusionCriterionCreateDetailsFromSpec(
	spec governancerulescontrolplanev1beta1.InclusionCriterionSpec,
) (governancerulescontrolplanesdk.CreateInclusionCriterionDetails, error) {
	criterionType, err := normalizeInclusionCriterionType(spec.Type)
	if err != nil {
		return governancerulescontrolplanesdk.CreateInclusionCriterionDetails{}, err
	}
	association, err := inclusionCriterionAssociationFromSpec(criterionType, spec.Association)
	if err != nil {
		return governancerulescontrolplanesdk.CreateInclusionCriterionDetails{}, err
	}
	if err := validateInclusionCriterionRequiredSpec(spec, criterionType, association); err != nil {
		return governancerulescontrolplanesdk.CreateInclusionCriterionDetails{}, err
	}

	return governancerulescontrolplanesdk.CreateInclusionCriterionDetails{
		GovernanceRuleId: common.String(strings.TrimSpace(spec.GovernanceRuleId)),
		Type:             criterionType,
		Association:      association,
	}, nil
}

func validateInclusionCriterionRequiredSpec(
	spec governancerulescontrolplanev1beta1.InclusionCriterionSpec,
	criterionType governancerulescontrolplanesdk.InclusionCriterionTypeEnum,
	association governancerulescontrolplanesdk.Association,
) error {
	var missing []string
	if strings.TrimSpace(spec.GovernanceRuleId) == "" {
		missing = append(missing, "governanceRuleId")
	}
	if strings.TrimSpace(spec.Type) == "" {
		missing = append(missing, "type")
	}
	if criterionType == governancerulescontrolplanesdk.InclusionCriterionTypeTenancy {
		key, err := inclusionCriterionAssociationKeyFromSDK(association)
		if err != nil {
			return err
		}
		if strings.TrimSpace(key.tenancyID) == "" {
			missing = append(missing, "association.tenancyId")
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("inclusion criterion spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func normalizeInclusionCriterionType(value string) (governancerulescontrolplanesdk.InclusionCriterionTypeEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("inclusion criterion type is required")
	}
	criterionType, ok := governancerulescontrolplanesdk.GetMappingInclusionCriterionTypeEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported inclusion criterion type %q", value)
	}
	return criterionType, nil
}

func inclusionCriterionAssociationFromSpec(
	criterionType governancerulescontrolplanesdk.InclusionCriterionTypeEnum,
	association governancerulescontrolplanev1beta1.InclusionCriterionAssociation,
) (governancerulescontrolplanesdk.Association, error) {
	if criterionType == governancerulescontrolplanesdk.InclusionCriterionTypeAll {
		if inclusionCriterionAssociationSpecIsZero(association) {
			return nil, nil
		}
		return nil, fmt.Errorf("ALL inclusion criteria do not support an association payload")
	}
	if criterionType != governancerulescontrolplanesdk.InclusionCriterionTypeTenancy {
		return nil, fmt.Errorf("unsupported inclusion criterion type %q", criterionType)
	}
	if raw := strings.TrimSpace(association.JsonData); raw != "" && raw != "null" {
		return inclusionCriterionAssociationFromJSON(raw, string(criterionType))
	}

	associationType, err := normalizeInclusionCriterionAssociationType(association.Type, string(criterionType))
	if err != nil {
		return nil, err
	}
	if associationType != string(governancerulescontrolplanesdk.InclusionCriterionTypeTenancy) {
		return nil, fmt.Errorf("unsupported inclusion criterion association type %q", association.Type)
	}
	return governancerulescontrolplanesdk.TenancyAssociation{
		TenancyId: common.String(strings.TrimSpace(association.TenancyId)),
	}, nil
}

func inclusionCriterionAssociationFromJSON(raw string, fallbackType string) (governancerulescontrolplanesdk.Association, error) {
	var discriminator struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode inclusion criterion association discriminator: %w", err)
	}
	associationType, err := normalizeInclusionCriterionAssociationType(discriminator.Type, fallbackType)
	if err != nil {
		return nil, err
	}
	switch associationType {
	case string(governancerulescontrolplanesdk.InclusionCriterionTypeTenancy):
		var tenancy governancerulescontrolplanesdk.TenancyAssociation
		if err := json.Unmarshal([]byte(raw), &tenancy); err != nil {
			return nil, fmt.Errorf("decode TENANCY inclusion criterion association: %w", err)
		}
		return tenancy, nil
	default:
		return nil, fmt.Errorf("unsupported inclusion criterion association type %q", associationType)
	}
}

func normalizeInclusionCriterionAssociationType(value string, fallback string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		normalized = strings.ToUpper(strings.TrimSpace(fallback))
	}
	if normalized == "" {
		return "", fmt.Errorf("inclusion criterion association type is required")
	}
	if normalized != string(governancerulescontrolplanesdk.InclusionCriterionTypeTenancy) {
		return "", fmt.Errorf("unsupported inclusion criterion association type %q", value)
	}
	return normalized, nil
}

func inclusionCriterionAssociationSpecIsZero(
	association governancerulescontrolplanev1beta1.InclusionCriterionAssociation,
) bool {
	raw := strings.TrimSpace(association.JsonData)
	return (raw == "" || raw == "null") &&
		strings.TrimSpace(association.Type) == "" &&
		strings.TrimSpace(association.TenancyId) == ""
}

func guardInclusionCriterionExistingBeforeCreate(
	_ context.Context,
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveInclusionCriterionIdentityValue(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if identity.governanceRuleID == "" || identity.criterionType == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveInclusionCriterionIdentity(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
) (any, error) {
	return resolveInclusionCriterionIdentityValue(resource)
}

func resolveInclusionCriterionIdentityValue(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
) (inclusionCriterionIdentity, error) {
	if resource == nil {
		return inclusionCriterionIdentity{}, fmt.Errorf("inclusion criterion resource is nil")
	}
	details, err := inclusionCriterionCreateDetailsFromSpec(resource.Spec)
	if err != nil {
		return inclusionCriterionIdentity{}, err
	}
	key, err := inclusionCriterionAssociationKeyFromSDK(details.Association)
	if err != nil {
		return inclusionCriterionIdentity{}, err
	}
	return inclusionCriterionIdentity{
		governanceRuleID: strings.TrimSpace(inclusionCriterionStringValue(details.GovernanceRuleId)),
		criterionType:    string(details.Type),
		associationType:  key.associationType,
		tenancyID:        key.tenancyID,
	}, nil
}

func recordInclusionCriterionPathIdentity(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	identity any,
) {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	typed, ok := identity.(inclusionCriterionIdentity)
	if !ok {
		return
	}
	if resource.Status.GovernanceRuleId == "" {
		resource.Status.GovernanceRuleId = typed.governanceRuleID
	}
	if resource.Status.Type == "" {
		resource.Status.Type = typed.criterionType
	}
	if resource.Status.Association.Type == "" {
		resource.Status.Association.Type = typed.associationType
	}
	if resource.Status.Association.TenancyId == "" {
		resource.Status.Association.TenancyId = typed.tenancyID
	}
}

func recordTrackedInclusionCriterionIdentity(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	identity any,
	resourceID string,
) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(resourceID) != "" {
		resource.Status.Id = strings.TrimSpace(resourceID)
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	recordInclusionCriterionPathIdentity(resource, identity)
}

func clearTrackedInclusionCriterionIdentity(resource *governancerulescontrolplanev1beta1.InclusionCriterion) {
	if resource == nil {
		return
	}
	resource.Status = governancerulescontrolplanev1beta1.InclusionCriterionStatus{}
}

func validateInclusionCriterionCreateOnlyDriftForResponse(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("inclusion criterion resource is nil")
	}
	current, ok := inclusionCriterionFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current inclusion criterion response does not expose an inclusion criterion body")
	}
	return validateInclusionCriterionCreateOnlyDrift(resource.Spec, current)
}

func validateInclusionCriterionCreateOnlyDrift(
	spec governancerulescontrolplanev1beta1.InclusionCriterionSpec,
	current governancerulescontrolplanesdk.InclusionCriterion,
) error {
	desired, err := inclusionCriterionCreateDetailsFromSpec(spec)
	if err != nil {
		return err
	}
	desiredAssociation, err := inclusionCriterionAssociationKeyFromSDK(desired.Association)
	if err != nil {
		return err
	}
	currentAssociation, err := inclusionCriterionAssociationKeyFromSDK(current.Association)
	if err != nil {
		return err
	}

	var drift []string
	if !inclusionCriterionStringPtrEqual(current.GovernanceRuleId, inclusionCriterionStringValue(desired.GovernanceRuleId)) {
		drift = append(drift, "governanceRuleId")
	}
	if current.Type != "" && current.Type != desired.Type {
		drift = append(drift, "type")
	}
	if desiredAssociation.associationType != currentAssociation.associationType {
		drift = append(drift, "association.type")
	}
	if desiredAssociation.tenancyID != currentAssociation.tenancyID {
		drift = append(drift, "association.tenancyId")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf(
		"InclusionCriterion has no OCI update operation; create-only field drift is not supported: %s",
		strings.Join(drift, ", "),
	)
}

func inclusionCriterionDeleteConfirmRead(
	get func(context.Context, governancerulescontrolplanesdk.GetInclusionCriterionRequest) (governancerulescontrolplanesdk.GetInclusionCriterionResponse, error),
) func(context.Context, *governancerulescontrolplanev1beta1.InclusionCriterion, string) (any, error) {
	return func(ctx context.Context, resource *governancerulescontrolplanev1beta1.InclusionCriterion, currentID string) (any, error) {
		resourceID := strings.TrimSpace(currentID)
		if resourceID == "" {
			resourceID = currentInclusionCriterionID(resource)
		}
		if resourceID == "" {
			return nil, errorutil.NotFoundOciError{
				HTTPStatusCode: 404,
				ErrorCode:      errorutil.NotFound,
				Description:    "InclusionCriterion delete confirmation has no recorded OCI identifier",
			}
		}
		if get == nil {
			return nil, fmt.Errorf("InclusionCriterion delete confirmation requires a Get operation")
		}
		response, err := get(ctx, governancerulescontrolplanesdk.GetInclusionCriterionRequest{
			InclusionCriterionId: common.String(resourceID),
		})
		return inclusionCriterionDeleteConfirmReadResponse(response, err)
	}
}

func inclusionCriterionDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return ambiguousInclusionCriterionNotFound("delete confirmation", err), nil
	}
	return nil, err
}

func rejectInclusionCriterionAuthShapedNotFound(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	err error,
) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return ambiguousInclusionCriterionNotFound("delete path", err)
}

func applyInclusionCriterionDeleteOutcome(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if ambiguous, ok := inclusionCriterionAmbiguousNotFoundResponse(response); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}
	current, ok := inclusionCriterionFromResponse(response)
	if !ok || current.LifecycleState != governancerulescontrolplanesdk.InclusionCriterionLifecycleStateActive {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !inclusionCriterionDeleteIsPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	markInclusionCriterionDeletePending(resource, inclusionCriterionStringValue(current.Id), string(current.LifecycleState))
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func inclusionCriterionDeleteIsPending(resource *governancerulescontrolplanev1beta1.InclusionCriterion) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete && current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markInclusionCriterionDeletePending(
	resource *governancerulescontrolplanev1beta1.InclusionCriterion,
	resourceID string,
	rawState string,
) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(resourceID) != "" {
		resource.Status.Id = strings.TrimSpace(resourceID)
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	now := metav1.Now()
	message := "OCI InclusionCriterion delete is in progress"
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       strings.TrimSpace(rawState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func currentInclusionCriterionID(resource *governancerulescontrolplanev1beta1.InclusionCriterion) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func inclusionCriterionFromResponse(response any) (governancerulescontrolplanesdk.InclusionCriterion, bool) {
	switch current := response.(type) {
	case governancerulescontrolplanesdk.CreateInclusionCriterionResponse:
		return current.InclusionCriterion, current.Id != nil
	case *governancerulescontrolplanesdk.CreateInclusionCriterionResponse:
		if current == nil {
			return governancerulescontrolplanesdk.InclusionCriterion{}, false
		}
		return current.InclusionCriterion, current.Id != nil
	case governancerulescontrolplanesdk.GetInclusionCriterionResponse:
		return current.InclusionCriterion, current.Id != nil
	case *governancerulescontrolplanesdk.GetInclusionCriterionResponse:
		if current == nil {
			return governancerulescontrolplanesdk.InclusionCriterion{}, false
		}
		return current.InclusionCriterion, current.Id != nil
	case governancerulescontrolplanesdk.InclusionCriterion:
		return current, current.Id != nil
	case *governancerulescontrolplanesdk.InclusionCriterion:
		if current == nil {
			return governancerulescontrolplanesdk.InclusionCriterion{}, false
		}
		return *current, current.Id != nil
	default:
		return governancerulescontrolplanesdk.InclusionCriterion{}, false
	}
}

func inclusionCriterionAssociationKeyFromSDK(
	association governancerulescontrolplanesdk.Association,
) (inclusionCriterionAssociationKey, error) {
	if association == nil {
		return inclusionCriterionAssociationKey{}, nil
	}
	payload, err := json.Marshal(association)
	if err != nil {
		return inclusionCriterionAssociationKey{}, fmt.Errorf("marshal inclusion criterion association: %w", err)
	}
	var decoded struct {
		Type      string `json:"type"`
		TenancyId string `json:"tenancyId"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return inclusionCriterionAssociationKey{}, fmt.Errorf("decode inclusion criterion association: %w", err)
	}
	return inclusionCriterionAssociationKey{
		associationType: strings.ToUpper(strings.TrimSpace(decoded.Type)),
		tenancyID:       strings.TrimSpace(decoded.TenancyId),
	}, nil
}

func ambiguousInclusionCriterionNotFound(operation string, err error) ambiguousInclusionCriterionNotFoundError {
	message := fmt.Sprintf(
		"InclusionCriterion %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s",
		strings.TrimSpace(operation),
		err.Error(),
	)
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousInclusionCriterionNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousInclusionCriterionNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func inclusionCriterionAmbiguousNotFoundResponse(value any) (ambiguousInclusionCriterionNotFoundError, bool) {
	switch typed := value.(type) {
	case ambiguousInclusionCriterionNotFoundError:
		return typed, true
	case *ambiguousInclusionCriterionNotFoundError:
		if typed == nil {
			return ambiguousInclusionCriterionNotFoundError{}, false
		}
		return *typed, true
	default:
		return ambiguousInclusionCriterionNotFoundError{}, false
	}
}

func inclusionCriterionStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func inclusionCriterionStringPtrEqual(value *string, want string) bool {
	if value == nil {
		return strings.TrimSpace(want) == ""
	}
	return strings.TrimSpace(*value) == strings.TrimSpace(want)
}
