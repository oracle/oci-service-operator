/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package maskingcolumn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MaskingColumnMaskingPolicyIDAnnotation = "datasafe.oracle.com/masking-policy-id"

	maskingColumnLegacyMaskingPolicyIDAnnotation = "datasafe.oracle.com/maskingPolicyId"
	maskingColumnSyntheticKeyPrefix              = "masking-column:"
	maskingColumnDeletePendingMessage            = "OCI MaskingColumn delete is in progress"
)

var maskingColumnWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(datasafesdk.WorkRequestStatusAccepted),
		string(datasafesdk.WorkRequestStatusInProgress),
		string(datasafesdk.WorkRequestStatusCanceling),
		string(datasafesdk.WorkRequestStatusSuspending),
		string(datasafesdk.WorkRequestStatusSuspended),
	},
	SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
	CreateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeCreateMaskingColumn)},
	UpdateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeUpdateMaskingColumn)},
}

type maskingColumnOCIClient interface {
	CreateMaskingColumn(context.Context, datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error)
	GetMaskingColumn(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error)
	ListMaskingColumns(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error)
	UpdateMaskingColumn(context.Context, datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error)
	DeleteMaskingColumn(context.Context, datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type maskingColumnIdentity struct {
	maskingPolicyID string
	key             string
	schemaName      string
	objectName      string
	columnName      string
}

type maskingColumnAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e maskingColumnAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e maskingColumnAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerMaskingColumnRuntimeHooksMutator(func(manager *MaskingColumnServiceManager, hooks *MaskingColumnRuntimeHooks) {
		workRequestClient, initErr := newMaskingColumnWorkRequestClient(manager)
		applyMaskingColumnRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newMaskingColumnWorkRequestClient(manager *MaskingColumnServiceManager) (maskingColumnOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("MaskingColumn service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize MaskingColumn work request OCI client: %w", err)
	}
	return client, nil
}

func applyMaskingColumnRuntimeHooks(
	hooks *MaskingColumnRuntimeHooks,
	workRequestClient maskingColumnOCIClient,
	workRequestClientInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = maskingColumnRuntimeSemantics()
	hooks.BuildCreateBody = buildMaskingColumnCreateBody
	hooks.BuildUpdateBody = buildMaskingColumnUpdateBody
	hooks.Identity.Resolve = resolveMaskingColumnIdentity
	hooks.Identity.RecordPath = recordMaskingColumnPathIdentity
	hooks.Identity.RecordTracked = recordTrackedMaskingColumnIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardMaskingColumnExistingBeforeCreate
	hooks.Identity.SeedSyntheticTrackedID = seedSyntheticMaskingColumnKey
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedMaskingColumnIdentity
	hooks.Async.Adapter = maskingColumnWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getMaskingColumnWorkRequest(ctx, workRequestClient, workRequestClientInitErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveMaskingColumnWorkRequestAction
	hooks.Async.RecoverResourceID = recoverMaskingColumnKeyFromWorkRequest
	hooks.Create.Fields = maskingColumnCreateFields()
	hooks.Get.Fields = maskingColumnKeyedFields()
	hooks.List.Fields = maskingColumnListFields()
	hooks.Update.Fields = append(maskingColumnKeyedFields(), generatedruntime.RequestField{
		FieldName:    "UpdateMaskingColumnDetails",
		RequestName:  "UpdateMaskingColumnDetails",
		Contribution: "body",
	})
	hooks.Delete.Fields = maskingColumnKeyedFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listMaskingColumnsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeMaskingColumnDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateMaskingColumnCreateOnlyDrift
	hooks.DeleteHooks.ConfirmRead = maskingColumnDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleMaskingColumnDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyMaskingColumnDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markMaskingColumnTerminating
	hooks.StatusHooks.MarkDeleted = markMaskingColumnDeleted
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MaskingColumnServiceClient) MaskingColumnServiceClient {
		return maskingColumnDeleteGuardClient{
			MaskingColumnServiceClient: delegate,
			workRequestClient:          workRequestClient,
			workRequestClientInitErr:   workRequestClientInitErr,
			listMaskingColumns:         hooks.List.Call,
		}
	})
}

type maskingColumnDeleteGuardClient struct {
	MaskingColumnServiceClient
	workRequestClient        maskingColumnOCIClient
	workRequestClientInitErr error
	listMaskingColumns       func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error)
}

func (c maskingColumnDeleteGuardClient) Delete(ctx context.Context, resource *datasafev1beta1.MaskingColumn) (bool, error) {
	if handled, err := c.observePendingWriteBeforeDelete(ctx, resource); handled || err != nil {
		return false, err
	}
	ready, err := c.resolveUntrackedKeyBeforeDelete(ctx, resource)
	if err != nil || !ready {
		return false, err
	}
	return c.MaskingColumnServiceClient.Delete(ctx, resource)
}

func (c maskingColumnDeleteGuardClient) observePendingWriteBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.MaskingColumn,
) (bool, error) {
	pending, ok := pendingMaskingColumnWriteWorkRequest(resource)
	if !ok {
		return false, nil
	}
	workRequest, err := getMaskingColumnWorkRequest(
		ctx,
		c.workRequestClient,
		c.workRequestClientInitErr,
		pending.WorkRequestID,
	)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return true, err
	}
	current, err := maskingColumnWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, pending.Phase)
	if err != nil {
		return true, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		markMaskingColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassPending, maskingColumnPendingWriteBeforeDeleteMessage(current))
		return true, nil
	case shared.OSOKAsyncClassSucceeded:
		return false, c.recordSucceededWriteWorkRequestBeforeDelete(resource, workRequest, current)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("MaskingColumn %s work request %s finished with status %s before delete", current.Phase, current.WorkRequestID, current.RawStatus)
		markMaskingColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return true, err
	default:
		err := fmt.Errorf("MaskingColumn %s work request %s projected unsupported async class %s before delete", current.Phase, current.WorkRequestID, current.NormalizedClass)
		markMaskingColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return true, err
	}
}

func (c maskingColumnDeleteGuardClient) recordSucceededWriteWorkRequestBeforeDelete(
	resource *datasafev1beta1.MaskingColumn,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) error {
	key, err := recoverMaskingColumnKeyFromWorkRequest(resource, workRequest, current.Phase)
	if err != nil {
		markMaskingColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return err
	}
	if !isSyntheticMaskingColumnKey(key) {
		identity, err := resolveMaskingColumnIdentity(resource)
		if err != nil {
			markMaskingColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
			return err
		}
		recordTrackedMaskingColumnIdentity(resource, identity, key)
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return nil
}

func (c maskingColumnDeleteGuardClient) resolveUntrackedKeyBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.MaskingColumn,
) (bool, error) {
	if currentMaskingColumnKey(resource) != "" {
		return true, nil
	}
	if c.listMaskingColumns == nil {
		markMaskingColumnTerminating(resource, datasafesdk.MaskingColumn{})
		return false, nil
	}
	policyID := currentMaskingColumnPolicyID(resource)
	if policyID == "" {
		return false, fmt.Errorf("MaskingColumn delete resolution requires %s or recorded status.maskingPolicyId", MaskingColumnMaskingPolicyIDAnnotation)
	}
	response, err := c.listMaskingColumns(ctx, datasafesdk.ListMaskingColumnsRequest{
		MaskingPolicyId: common.String(policyID),
		SchemaName:      []string{resource.Spec.SchemaName},
		ObjectName:      []string{resource.Spec.ObjectName},
		ColumnName:      []string{resource.Spec.ColumnName},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return false, ambiguousMaskingColumnNotFound("delete resolution", err)
		}
		return false, err
	}

	match, err := maskingColumnSelectListItem(resource, response.Items)
	if err != nil {
		if isMaskingColumnListNoMatch(err) {
			markMaskingColumnTerminating(resource, datasafesdk.MaskingColumn{})
			return false, nil
		}
		return false, err
	}
	recordMaskingColumnFromSDK(resource, maskingColumnFromSummary(match))
	return true, nil
}

func maskingColumnRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "maskingcolumn",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update"},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.MaskingColumnLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.MaskingColumnLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.MaskingColumnLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.MaskingColumnLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"maskingPolicyId", "schemaName", "objectName", "columnName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"objectType",
				"maskingColumnGroup",
				"sensitiveTypeId",
				"isMaskingEnabled",
				"maskingFormats",
			},
			ForceNew:      []string{"maskingPolicyId", "schemaName", "objectName", "columnName"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "MaskingColumn", Action: "CreateMaskingColumn"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "MaskingColumn", Action: "UpdateMaskingColumn"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "MaskingColumn", Action: "DeleteMaskingColumn"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "MaskingColumn", Action: "GetMaskingColumn"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "MaskingColumn", Action: "GetMaskingColumn"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "MaskingColumn", Action: "GetMaskingColumn"}},
		},
	}
}

func maskingColumnCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		maskingPolicyIDPathField(),
		{
			FieldName:    "CreateMaskingColumnDetails",
			RequestName:  "CreateMaskingColumnDetails",
			Contribution: "body",
		},
	}
}

func maskingColumnKeyedFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "MaskingColumnKey",
			RequestName:      "maskingColumnKey",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.key", "key"},
		},
		maskingPolicyIDPathField(),
	}
}

func maskingColumnListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		maskingPolicyIDPathField(),
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func maskingPolicyIDPathField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "MaskingPolicyId",
		RequestName:  "maskingPolicyId",
		Contribution: "path",
		LookupPaths:  []string{"status.maskingPolicyId", "maskingPolicyId"},
	}
}

func resolveMaskingColumnIdentity(resource *datasafev1beta1.MaskingColumn) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("MaskingColumn resource is nil")
	}
	annotatedPolicyID := maskingColumnPolicyIDAnnotation(resource)
	recordedPolicyID := strings.TrimSpace(resource.Status.MaskingPolicyId)
	if annotatedPolicyID != "" && recordedPolicyID != "" && annotatedPolicyID != recordedPolicyID {
		return nil, fmt.Errorf("MaskingColumn create-only parent masking policy annotation %q changed; create a replacement resource instead", MaskingColumnMaskingPolicyIDAnnotation)
	}
	policyID := firstNonEmptyMaskingColumnString(annotatedPolicyID, recordedPolicyID)
	if policyID == "" && currentMaskingColumnKey(resource) == "" {
		return nil, fmt.Errorf("MaskingColumn requires metadata annotation %q with the parent masking policy OCID because spec.maskingPolicyId is not available", MaskingColumnMaskingPolicyIDAnnotation)
	}

	return maskingColumnIdentity{
		maskingPolicyID: policyID,
		key:             currentMaskingColumnKey(resource),
		schemaName:      strings.TrimSpace(firstNonEmptyMaskingColumnString(resource.Status.SchemaName, resource.Spec.SchemaName)),
		objectName:      strings.TrimSpace(firstNonEmptyMaskingColumnString(resource.Status.ObjectName, resource.Spec.ObjectName)),
		columnName:      strings.TrimSpace(firstNonEmptyMaskingColumnString(resource.Status.ColumnName, resource.Spec.ColumnName)),
	}, nil
}

func recordMaskingColumnPathIdentity(resource *datasafev1beta1.MaskingColumn, identity any) {
	if resource == nil {
		return
	}
	resolved, ok := identity.(maskingColumnIdentity)
	if !ok {
		return
	}
	if resource.Status.MaskingPolicyId == "" {
		resource.Status.MaskingPolicyId = resolved.maskingPolicyID
	}
	if resource.Status.SchemaName == "" {
		resource.Status.SchemaName = resolved.schemaName
	}
	if resource.Status.ObjectName == "" {
		resource.Status.ObjectName = resolved.objectName
	}
	if resource.Status.ColumnName == "" {
		resource.Status.ColumnName = resolved.columnName
	}
}

func recordTrackedMaskingColumnIdentity(resource *datasafev1beta1.MaskingColumn, identity any, resourceID string) {
	if resource == nil {
		return
	}
	recordMaskingColumnPathIdentity(resource, identity)
	recordedID := strings.TrimSpace(resourceID)
	if isSyntheticMaskingColumnKey(recordedID) {
		recordedID = ""
	}
	key := firstNonEmptyMaskingColumnString(recordedID, resource.Status.Key)
	if key == "" {
		return
	}
	resource.Status.Key = key
	resource.Status.OsokStatus.Ocid = shared.OCID(key)
	if resource.Status.OsokStatus.CreatedAt == nil {
		now := metav1.Now()
		resource.Status.OsokStatus.CreatedAt = &now
	}
}

func clearTrackedMaskingColumnIdentity(resource *datasafev1beta1.MaskingColumn) {
	if resource == nil {
		return
	}
	resource.Status = datasafev1beta1.MaskingColumnStatus{}
}

func guardMaskingColumnExistingBeforeCreate(
	_ context.Context,
	resource *datasafev1beta1.MaskingColumn,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveMaskingColumnIdentity(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	resolved := identity.(maskingColumnIdentity)
	if resolved.maskingPolicyID == "" || resolved.schemaName == "" || resolved.objectName == "" || resolved.columnName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("MaskingColumn requires masking policy, schemaName, objectName, and columnName before create")
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func seedSyntheticMaskingColumnKey(resource *datasafev1beta1.MaskingColumn, identity any) func() {
	if resource == nil {
		return nil
	}
	if key := strings.TrimSpace(resource.Status.Key); key != "" {
		previous := resource.Status.OsokStatus.Ocid
		resource.Status.OsokStatus.Ocid = shared.OCID(key)
		return func() {
			resource.Status.OsokStatus.Ocid = previous
		}
	}
	if resource.DeletionTimestamp != nil && !resource.DeletionTimestamp.IsZero() {
		return nil
	}
	resolved, ok := identity.(maskingColumnIdentity)
	if !ok {
		return nil
	}
	synthetic := syntheticMaskingColumnKey(resolved)
	if synthetic == "" {
		return nil
	}
	previous := resource.Status.OsokStatus.Ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(synthetic)
	return func() {
		resource.Status.OsokStatus.Ocid = previous
	}
}

func buildMaskingColumnCreateBody(
	_ context.Context,
	resource *datasafev1beta1.MaskingColumn,
	_ string,
) (any, error) {
	if resource == nil {
		return datasafesdk.CreateMaskingColumnDetails{}, fmt.Errorf("MaskingColumn resource is nil")
	}
	normalizeMaskingColumnSpec(resource)
	if err := validateMaskingColumnCreateSpec(resource.Spec); err != nil {
		return datasafesdk.CreateMaskingColumnDetails{}, err
	}
	formats, err := maskingColumnMaskingFormatsFromSpec(resource.Spec.MaskingFormats)
	if err != nil {
		return datasafesdk.CreateMaskingColumnDetails{}, err
	}

	details := datasafesdk.CreateMaskingColumnDetails{
		SchemaName:       common.String(resource.Spec.SchemaName),
		ObjectName:       common.String(resource.Spec.ObjectName),
		ColumnName:       common.String(resource.Spec.ColumnName),
		IsMaskingEnabled: common.Bool(resource.Spec.IsMaskingEnabled),
	}
	if objectType, err := maskingColumnObjectType(resource.Spec.ObjectType); err != nil {
		return datasafesdk.CreateMaskingColumnDetails{}, err
	} else if objectType != "" {
		details.ObjectType = objectType
	}
	if value := strings.TrimSpace(resource.Spec.MaskingColumnGroup); value != "" {
		details.MaskingColumnGroup = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.SensitiveTypeId); value != "" {
		details.SensitiveTypeId = common.String(value)
	}
	if len(formats) > 0 {
		details.MaskingFormats = formats
	}
	return details, nil
}

func buildMaskingColumnUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.MaskingColumn,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateMaskingColumnDetails{}, false, fmt.Errorf("MaskingColumn resource is nil")
	}
	normalizeMaskingColumnSpec(resource)
	current, ok := maskingColumnFromResponse(currentResponse)
	if !ok {
		return datasafesdk.UpdateMaskingColumnDetails{}, false, fmt.Errorf("current MaskingColumn response does not expose a MaskingColumn body")
	}
	if err := validateMaskingColumnCreateOnlyDriftForCurrent(resource, current); err != nil {
		return datasafesdk.UpdateMaskingColumnDetails{}, false, err
	}
	return buildMaskingColumnMutableUpdateDetails(resource.Spec, current)
}

func buildMaskingColumnMutableUpdateDetails(
	spec datasafev1beta1.MaskingColumnSpec,
	current datasafesdk.MaskingColumn,
) (datasafesdk.UpdateMaskingColumnDetails, bool, error) {
	details := datasafesdk.UpdateMaskingColumnDetails{}
	updateNeeded := false
	changed, err := applyMaskingColumnObjectTypeUpdate(&details, spec.ObjectType, current.ObjectType)
	if err != nil {
		return datasafesdk.UpdateMaskingColumnDetails{}, false, err
	}
	updateNeeded = updateNeeded || changed
	updateNeeded = applyMaskingColumnStringUpdate(&details.MaskingColumnGroup, spec.MaskingColumnGroup, current.MaskingColumnGroup) || updateNeeded
	updateNeeded = applyMaskingColumnStringUpdate(&details.SensitiveTypeId, spec.SensitiveTypeId, current.SensitiveTypeId) || updateNeeded
	updateNeeded = applyMaskingColumnBoolUpdate(&details.IsMaskingEnabled, spec.IsMaskingEnabled, current.IsMaskingEnabled) || updateNeeded
	changed, err = applyMaskingColumnFormatsUpdate(&details, spec.MaskingFormats, current.MaskingFormats)
	if err != nil {
		return datasafesdk.UpdateMaskingColumnDetails{}, false, err
	}
	updateNeeded = updateNeeded || changed
	return details, updateNeeded, nil
}

func applyMaskingColumnObjectTypeUpdate(
	details *datasafesdk.UpdateMaskingColumnDetails,
	desiredValue string,
	currentValue datasafesdk.ObjectTypeEnum,
) (bool, error) {
	objectType, err := maskingColumnObjectType(desiredValue)
	if err != nil {
		return false, err
	}
	if objectType == "" || objectType == currentValue {
		return false, nil
	}
	details.ObjectType = objectType
	return true, nil
}

func applyMaskingColumnStringUpdate(field **string, specValue string, currentValue *string) bool {
	desired, ok := maskingColumnOptionalStringUpdate(specValue, currentValue)
	if !ok {
		return false
	}
	*field = desired
	return true
}

func applyMaskingColumnBoolUpdate(field **bool, desiredValue bool, currentValue *bool) bool {
	if currentValue != nil && desiredValue == *currentValue {
		return false
	}
	*field = common.Bool(desiredValue)
	return true
}

func applyMaskingColumnFormatsUpdate(
	details *datasafesdk.UpdateMaskingColumnDetails,
	specFormats []datasafev1beta1.MaskingColumnMaskingFormat,
	currentFormats []datasafesdk.MaskingFormat,
) (bool, error) {
	if len(specFormats) == 0 {
		return false, nil
	}
	formats, err := maskingColumnMaskingFormatsFromSpec(specFormats)
	if err != nil {
		return false, err
	}
	if maskingColumnMaskingFormatsEqual(formats, currentFormats) {
		return false, nil
	}
	details.MaskingFormats = formats
	return true, nil
}

func validateMaskingColumnCreateOnlyDrift(
	resource *datasafev1beta1.MaskingColumn,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("MaskingColumn resource is nil")
	}
	current, ok := maskingColumnFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current MaskingColumn response does not expose a MaskingColumn body")
	}
	return validateMaskingColumnCreateOnlyDriftForCurrent(resource, current)
}

func validateMaskingColumnCreateOnlyDriftForCurrent(
	resource *datasafev1beta1.MaskingColumn,
	current datasafesdk.MaskingColumn,
) error {
	drift, err := maskingColumnCreateOnlyDriftFields(resource, current)
	if err != nil {
		return err
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("MaskingColumn create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func maskingColumnCreateOnlyDriftFields(
	resource *datasafev1beta1.MaskingColumn,
	current datasafesdk.MaskingColumn,
) ([]string, error) {
	identity, err := resolveMaskingColumnIdentity(resource)
	if err != nil {
		return nil, err
	}
	resolved := identity.(maskingColumnIdentity)
	checks := []struct {
		name    string
		desired string
		current string
	}{
		{name: "maskingPolicyId", desired: resolved.maskingPolicyID, current: maskingColumnStringValue(current.MaskingPolicyId)},
		{name: "schemaName", desired: resource.Spec.SchemaName, current: maskingColumnStringValue(current.SchemaName)},
		{name: "objectName", desired: resource.Spec.ObjectName, current: maskingColumnStringValue(current.ObjectName)},
		{name: "columnName", desired: resource.Spec.ColumnName, current: maskingColumnStringValue(current.ColumnName)},
	}
	var drift []string
	for _, check := range checks {
		if desired := strings.TrimSpace(check.desired); desired != "" && desired != check.current {
			drift = append(drift, check.name)
		}
	}
	return drift, nil
}

func getMaskingColumnWorkRequest(
	ctx context.Context,
	client maskingColumnOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("MaskingColumn work request OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveMaskingColumnWorkRequestAction(workRequest any) (string, error) {
	current, ok := maskingColumnWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("work request response does not expose a Data Safe WorkRequest body")
	}
	return string(current.OperationType), nil
}

func recoverMaskingColumnKeyFromWorkRequest(
	resource *datasafev1beta1.MaskingColumn,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if key := currentMaskingColumnKey(resource); key != "" {
		return key, nil
	}
	if key := maskingColumnKeyFromWorkRequestResources(workRequest, phase); key != "" {
		return key, nil
	}
	identity, err := resolveMaskingColumnIdentity(resource)
	if err != nil {
		return "", err
	}
	return syntheticMaskingColumnKey(identity.(maskingColumnIdentity)), nil
}

func maskingColumnKeyFromWorkRequestResources(workRequest any, phase shared.OSOKAsyncPhase) string {
	current, ok := maskingColumnWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	for _, impacted := range current.Resources {
		if !maskingColumnWorkRequestActionMatchesPhase(impacted.ActionType, phase) {
			continue
		}
		if key := maskingColumnStringValue(impacted.Identifier); key != "" {
			return key
		}
	}
	return ""
}

func maskingColumnWorkRequestActionMatchesPhase(
	action datasafesdk.WorkRequestResourceActionTypeEnum,
	phase shared.OSOKAsyncPhase,
) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == datasafesdk.WorkRequestResourceActionTypeCreated ||
			action == datasafesdk.WorkRequestResourceActionTypeInProgress
	case shared.OSOKAsyncPhaseUpdate:
		return action == datasafesdk.WorkRequestResourceActionTypeUpdated ||
			action == datasafesdk.WorkRequestResourceActionTypeInProgress
	default:
		return true
	}
}

func maskingColumnWorkRequestFromAny(value any) (datasafesdk.WorkRequest, bool) {
	switch current := value.(type) {
	case datasafesdk.WorkRequest:
		return current, maskingColumnWorkRequestPresent(current)
	case *datasafesdk.WorkRequest:
		return maskingColumnWorkRequestFromPointer(current)
	case datasafesdk.GetWorkRequestResponse:
		return current.WorkRequest, maskingColumnWorkRequestPresent(current.WorkRequest)
	case *datasafesdk.GetWorkRequestResponse:
		return maskingColumnWorkRequestFromResponsePointer(current)
	default:
		return datasafesdk.WorkRequest{}, false
	}
}

func maskingColumnWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, ok := maskingColumnWorkRequestFromAny(workRequest)
	if !ok {
		return nil, fmt.Errorf("MaskingColumn work request response does not expose a Data Safe WorkRequest body")
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, maskingColumnWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        string(current.OperationType),
		RawOperationType: string(current.OperationType),
		WorkRequestID:    maskingColumnStringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func pendingMaskingColumnWriteWorkRequest(resource *datasafev1beta1.MaskingColumn) (*shared.OSOKAsyncOperation, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return nil, false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != "" && current.Source != shared.OSOKAsyncSourceWorkRequest {
		return nil, false
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return nil, false
	}
	if strings.TrimSpace(current.WorkRequestID) == "" {
		return nil, false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return current, true
	default:
		return nil, false
	}
}

func markMaskingColumnWriteWorkRequestBeforeDelete(
	resource *datasafev1beta1.MaskingColumn,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) {
	if resource == nil || current == nil {
		return
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	_ = servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, loggerutil.OSOKLogger{})
}

func maskingColumnPendingWriteBeforeDeleteMessage(current *shared.OSOKAsyncOperation) string {
	if current == nil {
		return "MaskingColumn write work request is still in progress; retaining finalizer before delete"
	}
	if workRequestID := strings.TrimSpace(current.WorkRequestID); workRequestID != "" {
		return fmt.Sprintf("MaskingColumn %s work request %s is still in progress; retaining finalizer before delete", current.Phase, workRequestID)
	}
	return fmt.Sprintf("MaskingColumn %s work request is still in progress; retaining finalizer before delete", current.Phase)
}

func maskingColumnWorkRequestFromPointer(current *datasafesdk.WorkRequest) (datasafesdk.WorkRequest, bool) {
	if current == nil {
		return datasafesdk.WorkRequest{}, false
	}
	return *current, maskingColumnWorkRequestPresent(*current)
}

func maskingColumnWorkRequestFromResponsePointer(
	current *datasafesdk.GetWorkRequestResponse,
) (datasafesdk.WorkRequest, bool) {
	if current == nil {
		return datasafesdk.WorkRequest{}, false
	}
	return current.WorkRequest, maskingColumnWorkRequestPresent(current.WorkRequest)
}

func maskingColumnWorkRequestPresent(current datasafesdk.WorkRequest) bool {
	return current.Id != nil || current.OperationType != ""
}

func listMaskingColumnsAllPages(
	call func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error),
) func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
		if call == nil {
			return datasafesdk.ListMaskingColumnsResponse{}, fmt.Errorf("MaskingColumn list operation is not configured")
		}
		var combined datasafesdk.ListMaskingColumnsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return combined, err
			}
			combined.Items = append(combined.Items, response.Items...)
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			request.Page = response.OpcNextPage
		}
	}
}

func maskingColumnDeleteConfirmRead(
	get func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error),
	list func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error),
) func(context.Context, *datasafev1beta1.MaskingColumn, string) (any, error) {
	return func(ctx context.Context, resource *datasafev1beta1.MaskingColumn, currentID string) (any, error) {
		policyID := currentMaskingColumnPolicyID(resource)
		if policyID == "" {
			return nil, fmt.Errorf("MaskingColumn delete confirmation requires %s or recorded status.maskingPolicyId", MaskingColumnMaskingPolicyIDAnnotation)
		}
		key := firstNonEmptyMaskingColumnString(currentID, currentMaskingColumnKey(resource))
		if key != "" && !isSyntheticMaskingColumnKey(key) {
			if get == nil {
				return nil, fmt.Errorf("MaskingColumn delete confirmation requires a Get operation")
			}
			response, err := get(ctx, datasafesdk.GetMaskingColumnRequest{
				MaskingPolicyId:  common.String(policyID),
				MaskingColumnKey: common.String(key),
			})
			return maskingColumnDeleteConfirmReadResponse(response, err)
		}
		if list == nil {
			return nil, fmt.Errorf("MaskingColumn delete confirmation requires a List operation")
		}
		response, err := list(ctx, datasafesdk.ListMaskingColumnsRequest{
			MaskingPolicyId: common.String(policyID),
			SchemaName:      []string{resource.Spec.SchemaName},
			ObjectName:      []string{resource.Spec.ObjectName},
			ColumnName:      []string{resource.Spec.ColumnName},
		})
		if err != nil {
			return maskingColumnDeleteConfirmReadResponse(response, err)
		}
		return maskingColumnSelectListItem(resource, response.Items)
	}
}

func maskingColumnDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return ambiguousMaskingColumnNotFound("delete confirmation", err), nil
	}
	return nil, err
}

func maskingColumnSelectListItem(
	resource *datasafev1beta1.MaskingColumn,
	items []datasafesdk.MaskingColumnSummary,
) (datasafesdk.MaskingColumnSummary, error) {
	var matches []datasafesdk.MaskingColumnSummary
	for _, item := range items {
		if maskingColumnSummaryMatchesResource(resource, item) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return datasafesdk.MaskingColumnSummary{}, errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			Description:    "MaskingColumn delete confirmation found no matching masking column",
		}
	case 1:
		return matches[0], nil
	default:
		return datasafesdk.MaskingColumnSummary{}, fmt.Errorf("MaskingColumn delete confirmation found multiple matching masking columns")
	}
}

func maskingColumnSummaryMatchesResource(
	resource *datasafev1beta1.MaskingColumn,
	item datasafesdk.MaskingColumnSummary,
) bool {
	if resource == nil {
		return false
	}
	if policyID := currentMaskingColumnPolicyID(resource); policyID != "" && policyID != maskingColumnStringValue(item.MaskingPolicyId) {
		return false
	}
	return strings.TrimSpace(resource.Spec.SchemaName) == maskingColumnStringValue(item.SchemaName) &&
		strings.TrimSpace(resource.Spec.ObjectName) == maskingColumnStringValue(item.ObjectName) &&
		strings.TrimSpace(resource.Spec.ColumnName) == maskingColumnStringValue(item.ColumnName)
}

func handleMaskingColumnDeleteError(resource *datasafev1beta1.MaskingColumn, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return ambiguousMaskingColumnNotFound("delete path", err)
}

func applyMaskingColumnDeleteOutcome(
	resource *datasafev1beta1.MaskingColumn,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if ambiguous, ok := maskingColumnAmbiguousNotFoundResponse(response); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}
	current, ok := maskingColumnFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	recordMaskingColumnFromSDK(resource, current)
	if current.LifecycleState == datasafesdk.MaskingColumnLifecycleStateActive {
		if stage == generatedruntime.DeleteConfirmStageAfterRequest || maskingColumnDeleteIsPending(resource) {
			markMaskingColumnTerminating(resource, current)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markMaskingColumnTerminating(resource *datasafev1beta1.MaskingColumn, response any) {
	if resource == nil {
		return
	}
	if current, ok := maskingColumnFromResponse(response); ok {
		recordMaskingColumnFromSDK(resource, current)
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = maskingColumnDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	rawState := strings.TrimSpace(resource.Status.LifecycleState)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         maskingColumnDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", maskingColumnDeletePendingMessage, loggerutil.OSOKLogger{})
}

func markMaskingColumnDeleted(resource *datasafev1beta1.MaskingColumn, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	if message != "" {
		status.Message = message
	}
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, loggerutil.OSOKLogger{})
}

func maskingColumnDeleteIsPending(resource *datasafev1beta1.MaskingColumn) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete && current.NormalizedClass == shared.OSOKAsyncClassPending
}

func recordMaskingColumnFromSDK(resource *datasafev1beta1.MaskingColumn, current datasafesdk.MaskingColumn) {
	if resource == nil {
		return
	}
	if key := maskingColumnStringValue(current.Key); key != "" {
		resource.Status.Key = key
		resource.Status.OsokStatus.Ocid = shared.OCID(key)
	}
	if value := maskingColumnStringValue(current.MaskingPolicyId); value != "" {
		resource.Status.MaskingPolicyId = value
	}
	if state := strings.TrimSpace(string(current.LifecycleState)); state != "" {
		resource.Status.LifecycleState = state
	}
}

func validateMaskingColumnCreateSpec(spec datasafev1beta1.MaskingColumnSpec) error {
	var missing []string
	if strings.TrimSpace(spec.SchemaName) == "" {
		missing = append(missing, "schemaName")
	}
	if strings.TrimSpace(spec.ObjectName) == "" {
		missing = append(missing, "objectName")
	}
	if strings.TrimSpace(spec.ColumnName) == "" {
		missing = append(missing, "columnName")
	}
	if len(missing) > 0 {
		return fmt.Errorf("MaskingColumn spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func normalizeMaskingColumnDesiredState(resource *datasafev1beta1.MaskingColumn, _ any) {
	normalizeMaskingColumnSpec(resource)
}

func normalizeMaskingColumnSpec(resource *datasafev1beta1.MaskingColumn) {
	if resource == nil {
		return
	}
	resource.Spec.SchemaName = strings.TrimSpace(resource.Spec.SchemaName)
	resource.Spec.ObjectName = strings.TrimSpace(resource.Spec.ObjectName)
	resource.Spec.ColumnName = strings.TrimSpace(resource.Spec.ColumnName)
	resource.Spec.ObjectType = strings.ToUpper(strings.TrimSpace(resource.Spec.ObjectType))
	resource.Spec.MaskingColumnGroup = strings.TrimSpace(resource.Spec.MaskingColumnGroup)
	resource.Spec.SensitiveTypeId = strings.TrimSpace(resource.Spec.SensitiveTypeId)
}

func maskingColumnObjectType(value string) (datasafesdk.ObjectTypeEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	objectType, ok := datasafesdk.GetMappingObjectTypeEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported MaskingColumn objectType %q", value)
	}
	return objectType, nil
}

func maskingColumnMaskingFormatsFromSpec(
	formats []datasafev1beta1.MaskingColumnMaskingFormat,
) ([]datasafesdk.MaskingFormat, error) {
	if len(formats) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(formats)
	if err != nil {
		return nil, fmt.Errorf("marshal maskingFormats: %w", err)
	}
	var converted []datasafesdk.MaskingFormat
	if err := json.Unmarshal(payload, &converted); err != nil {
		return nil, fmt.Errorf("decode maskingFormats: %w", err)
	}
	for index, format := range converted {
		if len(format.FormatEntries) == 0 {
			return nil, fmt.Errorf("maskingFormats[%d].formatEntries is required", index)
		}
		for entryIndex, entry := range format.FormatEntries {
			if entry == nil {
				return nil, fmt.Errorf("maskingFormats[%d].formatEntries[%d] is invalid", index, entryIndex)
			}
		}
	}
	return converted, nil
}

func maskingColumnOptionalStringUpdate(specValue string, currentValue *string) (*string, bool) {
	desired := strings.TrimSpace(specValue)
	if desired == "" {
		return nil, false
	}
	if desired == maskingColumnStringValue(currentValue) {
		return nil, false
	}
	return common.String(desired), true
}

func maskingColumnMaskingFormatsEqual(left []datasafesdk.MaskingFormat, right []datasafesdk.MaskingFormat) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && slices.Equal(leftPayload, rightPayload)
}

func maskingColumnFromResponse(response any) (datasafesdk.MaskingColumn, bool) {
	switch current := response.(type) {
	case datasafesdk.GetMaskingColumnResponse:
		return maskingColumnPresent(current.MaskingColumn)
	case *datasafesdk.GetMaskingColumnResponse:
		return maskingColumnFromGetResponsePointer(current)
	case datasafesdk.MaskingColumn:
		return maskingColumnPresent(current)
	case *datasafesdk.MaskingColumn:
		return maskingColumnFromPointer(current)
	case datasafesdk.MaskingColumnSummary:
		return maskingColumnPresent(maskingColumnFromSummary(current))
	case *datasafesdk.MaskingColumnSummary:
		return maskingColumnFromSummaryPointer(current)
	default:
		return datasafesdk.MaskingColumn{}, false
	}
}

func maskingColumnFromGetResponsePointer(
	current *datasafesdk.GetMaskingColumnResponse,
) (datasafesdk.MaskingColumn, bool) {
	if current == nil {
		return datasafesdk.MaskingColumn{}, false
	}
	return maskingColumnPresent(current.MaskingColumn)
}

func maskingColumnFromPointer(current *datasafesdk.MaskingColumn) (datasafesdk.MaskingColumn, bool) {
	if current == nil {
		return datasafesdk.MaskingColumn{}, false
	}
	return maskingColumnPresent(*current)
}

func maskingColumnFromSummaryPointer(current *datasafesdk.MaskingColumnSummary) (datasafesdk.MaskingColumn, bool) {
	if current == nil {
		return datasafesdk.MaskingColumn{}, false
	}
	return maskingColumnPresent(maskingColumnFromSummary(*current))
}

func maskingColumnPresent(current datasafesdk.MaskingColumn) (datasafesdk.MaskingColumn, bool) {
	return current, current.Key != nil
}

func maskingColumnFromSummary(summary datasafesdk.MaskingColumnSummary) datasafesdk.MaskingColumn {
	return datasafesdk.MaskingColumn{
		Key:                summary.Key,
		MaskingPolicyId:    summary.MaskingPolicyId,
		LifecycleState:     summary.LifecycleState,
		TimeCreated:        summary.TimeCreated,
		TimeUpdated:        summary.TimeUpdated,
		SchemaName:         summary.SchemaName,
		ObjectName:         summary.ObjectName,
		ColumnName:         summary.ColumnName,
		IsMaskingEnabled:   summary.IsMaskingEnabled,
		LifecycleDetails:   summary.LifecycleDetails,
		ObjectType:         summary.ObjectType,
		ChildColumns:       append([]string(nil), summary.ChildColumns...),
		MaskingColumnGroup: summary.MaskingColumnGroup,
		SensitiveTypeId:    summary.SensitiveTypeId,
		DataType:           summary.DataType,
		MaskingFormats:     append([]datasafesdk.MaskingFormat(nil), summary.MaskingFormats...),
	}
}

func ambiguousMaskingColumnNotFound(operation string, err error) maskingColumnAmbiguousNotFoundError {
	message := fmt.Sprintf(
		"MaskingColumn %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s",
		strings.TrimSpace(operation),
		err.Error(),
	)
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return maskingColumnAmbiguousNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return maskingColumnAmbiguousNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func maskingColumnAmbiguousNotFoundResponse(value any) (maskingColumnAmbiguousNotFoundError, bool) {
	switch typed := value.(type) {
	case maskingColumnAmbiguousNotFoundError:
		return typed, true
	case *maskingColumnAmbiguousNotFoundError:
		if typed == nil {
			return maskingColumnAmbiguousNotFoundError{}, false
		}
		return *typed, true
	default:
		return maskingColumnAmbiguousNotFoundError{}, false
	}
}

func maskingColumnPolicyIDAnnotation(resource *datasafev1beta1.MaskingColumn) string {
	if resource == nil {
		return ""
	}
	return maskingColumnAnnotationValue(
		resource.Annotations,
		MaskingColumnMaskingPolicyIDAnnotation,
		maskingColumnLegacyMaskingPolicyIDAnnotation,
	)
}

func maskingColumnAnnotationValue(annotations map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func currentMaskingColumnPolicyID(resource *datasafev1beta1.MaskingColumn) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptyMaskingColumnString(maskingColumnPolicyIDAnnotation(resource), resource.Status.MaskingPolicyId)
}

func currentMaskingColumnKey(resource *datasafev1beta1.MaskingColumn) string {
	if resource == nil {
		return ""
	}
	if key := strings.TrimSpace(resource.Status.Key); key != "" {
		return key
	}
	if key := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); key != "" && !isSyntheticMaskingColumnKey(key) {
		return key
	}
	return ""
}

func syntheticMaskingColumnKey(identity maskingColumnIdentity) string {
	parts := []string{
		strings.TrimSpace(identity.maskingPolicyID),
		strings.TrimSpace(identity.schemaName),
		strings.TrimSpace(identity.objectName),
		strings.TrimSpace(identity.columnName),
	}
	for _, part := range parts {
		if part == "" {
			return ""
		}
	}
	return maskingColumnSyntheticKeyPrefix + strings.Join(parts, "/")
}

func isSyntheticMaskingColumnKey(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), maskingColumnSyntheticKeyPrefix)
}

func isMaskingColumnListNoMatch(err error) bool {
	if err == nil {
		return false
	}
	var notFound errorutil.NotFoundOciError
	return errors.As(err, &notFound)
}

func maskingColumnStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyMaskingColumnString(values ...string) string {
	for _, value := range values {
		if value := strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
