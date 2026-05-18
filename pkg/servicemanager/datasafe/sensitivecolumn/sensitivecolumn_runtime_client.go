/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivecolumn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

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
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	SensitiveColumnSensitiveDataModelIDAnnotation = "datasafe.oracle.com/sensitive-data-model-id"

	sensitiveColumnLegacySensitiveDataModelIDAnnotation = "datasafe.oracle.com/sensitiveDataModelId"
	sensitiveColumnSyntheticKeyPrefix                   = "sensitive-column:"
	sensitiveColumnDeletePendingMessage                 = "OCI SensitiveColumn delete is in progress"
)

var sensitiveColumnWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
	CreateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeCreateSensitiveColumn)},
	UpdateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeUpdateSensitiveColumn)},
}

type sensitiveColumnOCIClient interface {
	CreateSensitiveColumn(context.Context, datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error)
	GetSensitiveColumn(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error)
	ListSensitiveColumns(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error)
	UpdateSensitiveColumn(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error)
	DeleteSensitiveColumn(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type sensitiveColumnIdentity struct {
	sensitiveDataModelID string
	key                  string
	schemaName           string
	objectName           string
	columnName           string
}

type sensitiveColumnAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e sensitiveColumnAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e sensitiveColumnAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSensitiveColumnRuntimeHooksMutator(func(manager *SensitiveColumnServiceManager, hooks *SensitiveColumnRuntimeHooks) {
		workRequestClient, initErr := newSensitiveColumnWorkRequestClient(manager)
		applySensitiveColumnRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newSensitiveColumnWorkRequestClient(manager *SensitiveColumnServiceManager) (sensitiveColumnOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("SensitiveColumn service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize SensitiveColumn work request OCI client: %w", err)
	}
	return client, nil
}

func applySensitiveColumnRuntimeHooks(
	hooks *SensitiveColumnRuntimeHooks,
	workRequestClient sensitiveColumnOCIClient,
	workRequestClientInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = sensitiveColumnRuntimeSemantics()
	hooks.BuildCreateBody = buildSensitiveColumnCreateBody
	hooks.BuildUpdateBody = buildSensitiveColumnUpdateBody
	hooks.Identity.Resolve = resolveSensitiveColumnIdentity
	hooks.Identity.RecordPath = recordSensitiveColumnPathIdentity
	hooks.Identity.RecordTracked = recordTrackedSensitiveColumnIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardSensitiveColumnExistingBeforeCreate
	hooks.Identity.SeedSyntheticTrackedID = seedSyntheticSensitiveColumnKey
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedSensitiveColumnIdentity
	hooks.Async.Adapter = sensitiveColumnWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getSensitiveColumnWorkRequest(ctx, workRequestClient, workRequestClientInitErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveSensitiveColumnWorkRequestAction
	hooks.Async.RecoverResourceID = recoverSensitiveColumnKeyFromWorkRequest
	hooks.Create.Fields = sensitiveColumnCreateFields()
	hooks.Get.Fields = sensitiveColumnKeyedFields()
	hooks.List.Fields = sensitiveColumnListFields()
	hooks.Update.Fields = append(sensitiveColumnKeyedFields(), generatedruntime.RequestField{
		FieldName:    "UpdateSensitiveColumnDetails",
		RequestName:  "UpdateSensitiveColumnDetails",
		Contribution: "body",
	})
	hooks.Delete.Fields = sensitiveColumnKeyedFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listSensitiveColumnsAllPages(hooks.List.Call)
	}
	installSensitiveColumnProjectedReadOperations(hooks)
	hooks.ParityHooks.NormalizeDesiredState = normalizeSensitiveColumnDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateSensitiveColumnCreateOnlyDrift
	hooks.StatusHooks.ProjectStatus = projectSensitiveColumnStatus
	hooks.DeleteHooks.ConfirmRead = sensitiveColumnDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleSensitiveColumnDeleteError
	hooks.DeleteHooks.ApplyOutcome = applySensitiveColumnDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markSensitiveColumnTerminating
	hooks.StatusHooks.MarkDeleted = markSensitiveColumnDeleted
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SensitiveColumnServiceClient) SensitiveColumnServiceClient {
		return sensitiveColumnDeleteGuardClient{
			SensitiveColumnServiceClient: delegate,
			workRequestClient:            workRequestClient,
			workRequestClientInitErr:     workRequestClientInitErr,
			listSensitiveColumns:         hooks.List.Call,
		}
	})
}

type sensitiveColumnDeleteGuardClient struct {
	SensitiveColumnServiceClient
	workRequestClient        sensitiveColumnOCIClient
	workRequestClientInitErr error
	listSensitiveColumns     func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error)
}

func (c sensitiveColumnDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveColumn,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.SensitiveColumnServiceClient.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || resource == nil {
		return response, err
	}
	if resource.Status.Status != "" || resource.Status.OsokStatus.Async.Current != nil {
		return response, nil
	}
	c.refreshObservedStatus(ctx, resource)
	return response, nil
}

func (c sensitiveColumnDeleteGuardClient) Delete(ctx context.Context, resource *datasafev1beta1.SensitiveColumn) (bool, error) {
	if handled, err := c.observePendingWriteBeforeDelete(ctx, resource); handled || err != nil {
		return false, err
	}
	ready, deleted, err := c.resolveUntrackedKeyBeforeDelete(ctx, resource)
	if err != nil || deleted || !ready {
		return deleted, err
	}
	restoreAnnotation := preferRecordedSensitiveColumnModelAnnotationForDelete(resource)
	defer restoreAnnotation()
	return c.SensitiveColumnServiceClient.Delete(ctx, resource)
}

func preferRecordedSensitiveColumnModelAnnotationForDelete(resource *datasafev1beta1.SensitiveColumn) func() {
	if resource == nil || currentSensitiveColumnKey(resource) == "" {
		return func() {}
	}
	recordedModelID := strings.TrimSpace(resource.Status.SensitiveDataModelId)
	annotatedModelID := sensitiveColumnModelIDAnnotation(resource)
	if recordedModelID == "" || annotatedModelID == "" || annotatedModelID == recordedModelID {
		return func() {}
	}

	annotations := resource.Annotations
	primary, hadPrimary := annotations[SensitiveColumnSensitiveDataModelIDAnnotation]
	legacy, hadLegacy := annotations[sensitiveColumnLegacySensitiveDataModelIDAnnotation]
	resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation] = recordedModelID
	return func() {
		if hadPrimary {
			resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation] = primary
		} else {
			delete(resource.Annotations, SensitiveColumnSensitiveDataModelIDAnnotation)
		}
		if hadLegacy {
			resource.Annotations[sensitiveColumnLegacySensitiveDataModelIDAnnotation] = legacy
		} else {
			delete(resource.Annotations, sensitiveColumnLegacySensitiveDataModelIDAnnotation)
		}
	}
}

func (c sensitiveColumnDeleteGuardClient) refreshObservedStatus(ctx context.Context, resource *datasafev1beta1.SensitiveColumn) {
	if c.workRequestClientInitErr != nil || c.workRequestClient == nil {
		return
	}
	modelID := currentSensitiveColumnModelID(resource)
	key := currentSensitiveColumnKey(resource)
	if modelID == "" || key == "" || isSyntheticSensitiveColumnKey(key) {
		return
	}
	response, err := c.workRequestClient.GetSensitiveColumn(ctx, datasafesdk.GetSensitiveColumnRequest{
		SensitiveDataModelId: common.String(modelID),
		SensitiveColumnKey:   common.String(key),
	})
	if err != nil {
		return
	}
	recordSensitiveColumnFromSDK(resource, response.SensitiveColumn)
}

func (c sensitiveColumnDeleteGuardClient) observePendingWriteBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveColumn,
) (bool, error) {
	pending, ok := pendingSensitiveColumnWriteWorkRequest(resource)
	if !ok {
		return false, nil
	}
	workRequest, err := getSensitiveColumnWorkRequest(
		ctx,
		c.workRequestClient,
		c.workRequestClientInitErr,
		pending.WorkRequestID,
	)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return true, err
	}
	current, err := sensitiveColumnWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, pending.Phase)
	if err != nil {
		return true, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		markSensitiveColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassPending, sensitiveColumnPendingWriteBeforeDeleteMessage(current))
		return true, nil
	case shared.OSOKAsyncClassSucceeded:
		return false, c.recordSucceededWriteWorkRequestBeforeDelete(resource, workRequest, current)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("SensitiveColumn %s work request %s finished with status %s before delete", current.Phase, current.WorkRequestID, current.RawStatus)
		markSensitiveColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return true, err
	default:
		err := fmt.Errorf("SensitiveColumn %s work request %s projected unsupported async class %s before delete", current.Phase, current.WorkRequestID, current.NormalizedClass)
		markSensitiveColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return true, err
	}
}

func (c sensitiveColumnDeleteGuardClient) recordSucceededWriteWorkRequestBeforeDelete(
	resource *datasafev1beta1.SensitiveColumn,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) error {
	key, err := recoverSensitiveColumnKeyBeforeDelete(resource, workRequest, current.Phase)
	if err != nil {
		markSensitiveColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return err
	}
	if !isSyntheticSensitiveColumnKey(key) {
		identity, err := resolveSensitiveColumnIdentityBeforeDelete(resource, key)
		if err != nil {
			markSensitiveColumnWriteWorkRequestBeforeDelete(resource, current, shared.OSOKAsyncClassFailed, err.Error())
			return err
		}
		recordTrackedSensitiveColumnIdentity(resource, identity, key)
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return nil
}

func recoverSensitiveColumnKeyBeforeDelete(
	resource *datasafev1beta1.SensitiveColumn,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if key := currentSensitiveColumnKey(resource); key != "" {
		return key, nil
	}
	if key := sensitiveColumnKeyFromWorkRequestResources(workRequest, phase); key != "" {
		return key, nil
	}
	identity, err := resolveSensitiveColumnIdentityBeforeDelete(resource, "")
	if err != nil {
		return "", err
	}
	return syntheticSensitiveColumnKey(identity), nil
}

func resolveSensitiveColumnIdentityBeforeDelete(
	resource *datasafev1beta1.SensitiveColumn,
	key string,
) (sensitiveColumnIdentity, error) {
	if resource == nil {
		return sensitiveColumnIdentity{}, fmt.Errorf("SensitiveColumn resource is nil")
	}
	modelID := firstNonEmptySensitiveColumnString(resource.Status.SensitiveDataModelId, sensitiveColumnModelIDAnnotation(resource))
	if modelID == "" && strings.TrimSpace(key) == "" {
		return sensitiveColumnIdentity{}, fmt.Errorf("SensitiveColumn delete requires %s or recorded status.sensitiveDataModelId", SensitiveColumnSensitiveDataModelIDAnnotation)
	}
	return sensitiveColumnIdentity{
		sensitiveDataModelID: modelID,
		key:                  strings.TrimSpace(firstNonEmptySensitiveColumnString(key, currentSensitiveColumnKey(resource))),
		schemaName:           strings.TrimSpace(firstNonEmptySensitiveColumnString(resource.Status.SchemaName, resource.Spec.SchemaName)),
		objectName:           strings.TrimSpace(firstNonEmptySensitiveColumnString(resource.Status.ObjectName, resource.Spec.ObjectName)),
		columnName:           strings.TrimSpace(firstNonEmptySensitiveColumnString(resource.Status.ColumnName, resource.Spec.ColumnName)),
	}, nil
}

func (c sensitiveColumnDeleteGuardClient) resolveUntrackedKeyBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.SensitiveColumn,
) (bool, bool, error) {
	if currentSensitiveColumnKey(resource) != "" {
		return true, false, nil
	}
	if c.listSensitiveColumns == nil {
		markSensitiveColumnTerminating(resource, datasafesdk.SensitiveColumn{})
		return false, false, nil
	}
	identity, err := sensitiveColumnDeleteListIdentity(resource)
	if err != nil {
		return false, false, err
	}
	response, err := c.listSensitiveColumns(ctx, datasafesdk.ListSensitiveColumnsRequest{
		SensitiveDataModelId: common.String(identity.sensitiveDataModelID),
		SchemaName:           []string{identity.schemaName},
		ObjectName:           []string{identity.objectName},
		ColumnName:           []string{identity.columnName},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return false, false, ambiguousSensitiveColumnNotFound("delete resolution", err)
		}
		return false, false, err
	}

	match, err := sensitiveColumnSelectListItem(resource, response.Items)
	if err != nil {
		if isSensitiveColumnListNoMatch(err) {
			markSensitiveColumnDeleted(resource, "SensitiveColumn delete confirmed absent before OCI delete")
			return false, true, nil
		}
		return false, false, err
	}
	recordSensitiveColumnFromSDK(resource, sensitiveColumnFromSummary(match))
	return true, false, nil
}

func sensitiveColumnRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "sensitivecolumn",
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
			ProvisioningStates: []string{string(datasafesdk.SensitiveColumnLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.SensitiveColumnLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.SensitiveColumnLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.SensitiveColumnLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"sensitiveDataModelId", "schemaName", "objectName", "columnName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"dataType",
				"status",
				"sensitiveTypeId",
				"parentColumnKeys",
				"relationType",
				"appDefinedChildColumnKeys",
				"dbDefinedChildColumnKeys",
			},
			ForceNew: []string{
				"sensitiveDataModelId",
				"schemaName",
				"objectName",
				"columnName",
				"appName",
				"objectType",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "SensitiveColumn", Action: "CreateSensitiveColumn"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "SensitiveColumn", Action: "UpdateSensitiveColumn"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "SensitiveColumn", Action: "DeleteSensitiveColumn"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "SensitiveColumn", Action: "GetSensitiveColumn"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "SensitiveColumn", Action: "GetSensitiveColumn"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "SensitiveColumn", Action: "GetSensitiveColumn"}},
		},
	}
}

func sensitiveColumnCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sensitiveDataModelIDPathField(),
		{
			FieldName:    "CreateSensitiveColumnDetails",
			RequestName:  "CreateSensitiveColumnDetails",
			Contribution: "body",
		},
	}
}

func sensitiveColumnKeyedFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sensitiveDataModelIDPathField(),
		{
			FieldName:        "SensitiveColumnKey",
			RequestName:      "sensitiveColumnKey",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.key", "key"},
		},
	}
}

func sensitiveColumnListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		sensitiveDataModelIDPathField(),
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func sensitiveDataModelIDPathField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "SensitiveDataModelId",
		RequestName:  "sensitiveDataModelId",
		Contribution: "path",
		LookupPaths:  []string{"status.sensitiveDataModelId", "sensitiveDataModelId"},
	}
}

func resolveSensitiveColumnIdentity(resource *datasafev1beta1.SensitiveColumn) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("SensitiveColumn resource is nil")
	}
	annotatedModelID := sensitiveColumnModelIDAnnotation(resource)
	recordedModelID := strings.TrimSpace(resource.Status.SensitiveDataModelId)
	if annotatedModelID != "" && recordedModelID != "" && annotatedModelID != recordedModelID {
		return nil, fmt.Errorf("SensitiveColumn create-only parent sensitive data model annotation %q changed; create a replacement resource instead", SensitiveColumnSensitiveDataModelIDAnnotation)
	}
	modelID := firstNonEmptySensitiveColumnString(annotatedModelID, recordedModelID)
	if modelID == "" && currentSensitiveColumnKey(resource) == "" {
		return nil, fmt.Errorf("SensitiveColumn requires metadata annotation %q with the parent sensitive data model OCID because spec.sensitiveDataModelId is not available", SensitiveColumnSensitiveDataModelIDAnnotation)
	}

	return sensitiveColumnIdentity{
		sensitiveDataModelID: modelID,
		key:                  currentSensitiveColumnKey(resource),
		schemaName:           strings.TrimSpace(firstNonEmptySensitiveColumnString(resource.Status.SchemaName, resource.Spec.SchemaName)),
		objectName:           strings.TrimSpace(firstNonEmptySensitiveColumnString(resource.Status.ObjectName, resource.Spec.ObjectName)),
		columnName:           strings.TrimSpace(firstNonEmptySensitiveColumnString(resource.Status.ColumnName, resource.Spec.ColumnName)),
	}, nil
}

func recordSensitiveColumnPathIdentity(resource *datasafev1beta1.SensitiveColumn, identity any) {
	if resource == nil {
		return
	}
	resolved, ok := identity.(sensitiveColumnIdentity)
	if !ok {
		return
	}
	if resource.Status.SensitiveDataModelId == "" {
		resource.Status.SensitiveDataModelId = resolved.sensitiveDataModelID
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

func recordTrackedSensitiveColumnIdentity(resource *datasafev1beta1.SensitiveColumn, identity any, resourceID string) {
	if resource == nil {
		return
	}
	recordSensitiveColumnPathIdentity(resource, identity)
	recordedID := strings.TrimSpace(resourceID)
	if isSyntheticSensitiveColumnKey(recordedID) {
		recordedID = ""
	}
	key := firstNonEmptySensitiveColumnString(recordedID, resource.Status.Key)
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

func clearTrackedSensitiveColumnIdentity(resource *datasafev1beta1.SensitiveColumn) {
	if resource == nil {
		return
	}
	resource.Status = datasafev1beta1.SensitiveColumnStatus{}
}

func guardSensitiveColumnExistingBeforeCreate(
	_ context.Context,
	resource *datasafev1beta1.SensitiveColumn,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveSensitiveColumnIdentity(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	resolved := identity.(sensitiveColumnIdentity)
	if resolved.sensitiveDataModelID == "" || resolved.schemaName == "" || resolved.objectName == "" || resolved.columnName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("SensitiveColumn requires sensitive data model, schemaName, objectName, and columnName before create")
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func seedSyntheticSensitiveColumnKey(resource *datasafev1beta1.SensitiveColumn, identity any) func() {
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
	resolved, ok := identity.(sensitiveColumnIdentity)
	if !ok {
		return nil
	}
	synthetic := syntheticSensitiveColumnKey(resolved)
	if synthetic == "" {
		return nil
	}
	previous := resource.Status.OsokStatus.Ocid
	resource.Status.OsokStatus.Ocid = shared.OCID(synthetic)
	return func() {
		resource.Status.OsokStatus.Ocid = previous
	}
}

func buildSensitiveColumnCreateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveColumn,
	_ string,
) (any, error) {
	if resource == nil {
		return datasafesdk.CreateSensitiveColumnDetails{}, fmt.Errorf("SensitiveColumn resource is nil")
	}
	normalizeSensitiveColumnSpec(resource)
	if err := validateSensitiveColumnCreateSpec(resource.Spec); err != nil {
		return datasafesdk.CreateSensitiveColumnDetails{}, err
	}

	details := datasafesdk.CreateSensitiveColumnDetails{
		SchemaName: common.String(resource.Spec.SchemaName),
		ObjectName: common.String(resource.Spec.ObjectName),
		ColumnName: common.String(resource.Spec.ColumnName),
	}
	if err := applySensitiveColumnCreateOptionalFields(&details, resource.Spec); err != nil {
		return datasafesdk.CreateSensitiveColumnDetails{}, err
	}
	return details, nil
}

func applySensitiveColumnCreateOptionalFields(
	details *datasafesdk.CreateSensitiveColumnDetails,
	spec datasafev1beta1.SensitiveColumnSpec,
) error {
	applySensitiveColumnCreateStringFields(details, spec)
	if err := applySensitiveColumnCreateEnumFields(details, spec); err != nil {
		return err
	}
	applySensitiveColumnCreateSliceFields(details, spec)
	return nil
}

func applySensitiveColumnCreateStringFields(
	details *datasafesdk.CreateSensitiveColumnDetails,
	spec datasafev1beta1.SensitiveColumnSpec,
) {
	if value := strings.TrimSpace(spec.AppName); value != "" {
		details.AppName = common.String(value)
	}
	if value := strings.TrimSpace(spec.DataType); value != "" {
		details.DataType = common.String(value)
	}
	if value := strings.TrimSpace(spec.SensitiveTypeId); value != "" {
		details.SensitiveTypeId = common.String(value)
	}
}

func applySensitiveColumnCreateEnumFields(
	details *datasafesdk.CreateSensitiveColumnDetails,
	spec datasafev1beta1.SensitiveColumnSpec,
) error {
	if objectType, err := sensitiveColumnCreateObjectType(spec.ObjectType); err != nil {
		return err
	} else if objectType != "" {
		details.ObjectType = objectType
	}
	if status, err := sensitiveColumnCreateStatus(spec.Status); err != nil {
		return err
	} else if status != "" {
		details.Status = status
	}
	if relationType, err := sensitiveColumnCreateRelationType(spec.RelationType); err != nil {
		return err
	} else if relationType != "" {
		details.RelationType = relationType
	}
	return nil
}

func applySensitiveColumnCreateSliceFields(
	details *datasafesdk.CreateSensitiveColumnDetails,
	spec datasafev1beta1.SensitiveColumnSpec,
) {
	if len(spec.ParentColumnKeys) > 0 {
		details.ParentColumnKeys = append([]string(nil), spec.ParentColumnKeys...)
	}
	if len(spec.AppDefinedChildColumnKeys) > 0 {
		details.AppDefinedChildColumnKeys = append([]string(nil), spec.AppDefinedChildColumnKeys...)
	}
	if len(spec.DbDefinedChildColumnKeys) > 0 {
		details.DbDefinedChildColumnKeys = append([]string(nil), spec.DbDefinedChildColumnKeys...)
	}
}

func buildSensitiveColumnUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveColumn,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateSensitiveColumnDetails{}, false, fmt.Errorf("SensitiveColumn resource is nil")
	}
	normalizeSensitiveColumnSpec(resource)
	current, ok := sensitiveColumnFromResponse(currentResponse)
	if !ok {
		return datasafesdk.UpdateSensitiveColumnDetails{}, false, fmt.Errorf("current SensitiveColumn response does not expose a SensitiveColumn body")
	}
	if err := validateSensitiveColumnCreateOnlyDriftForCurrent(resource, current); err != nil {
		return datasafesdk.UpdateSensitiveColumnDetails{}, false, err
	}
	if current.Status == "" && resource.Status.Status != "" {
		current.Status = datasafesdk.SensitiveColumnStatusEnum(resource.Status.Status)
	}
	return buildSensitiveColumnMutableUpdateDetails(resource.Spec, current)
}

func buildSensitiveColumnMutableUpdateDetails(
	spec datasafev1beta1.SensitiveColumnSpec,
	current datasafesdk.SensitiveColumn,
) (datasafesdk.UpdateSensitiveColumnDetails, bool, error) {
	details := datasafesdk.UpdateSensitiveColumnDetails{}
	updateNeeded := false
	updateNeeded = applySensitiveColumnStringUpdate(&details.DataType, spec.DataType, current.DataType) || updateNeeded
	changed, err := applySensitiveColumnStatusUpdate(&details, spec.Status, current.Status)
	if err != nil {
		return datasafesdk.UpdateSensitiveColumnDetails{}, false, err
	}
	updateNeeded = updateNeeded || changed
	updateNeeded = applySensitiveColumnStringUpdate(&details.SensitiveTypeId, spec.SensitiveTypeId, current.SensitiveTypeId) || updateNeeded
	updateNeeded = applySensitiveColumnStringSliceUpdate(&details.ParentColumnKeys, spec.ParentColumnKeys, current.ParentColumnKeys) || updateNeeded
	changed, err = applySensitiveColumnRelationTypeUpdate(&details, spec.RelationType, current.RelationType)
	if err != nil {
		return datasafesdk.UpdateSensitiveColumnDetails{}, false, err
	}
	updateNeeded = updateNeeded || changed
	updateNeeded = applySensitiveColumnStringSliceUpdate(&details.AppDefinedChildColumnKeys, spec.AppDefinedChildColumnKeys, current.AppDefinedChildColumnKeys) || updateNeeded
	updateNeeded = applySensitiveColumnStringSliceUpdate(&details.DbDefinedChildColumnKeys, spec.DbDefinedChildColumnKeys, current.DbDefinedChildColumnKeys) || updateNeeded
	return details, updateNeeded, nil
}

func applySensitiveColumnStatusUpdate(
	details *datasafesdk.UpdateSensitiveColumnDetails,
	desiredValue string,
	currentValue datasafesdk.SensitiveColumnStatusEnum,
) (bool, error) {
	status, err := sensitiveColumnUpdateStatus(desiredValue)
	if err != nil {
		return false, err
	}
	if status == "" || string(status) == string(currentValue) {
		return false, nil
	}
	details.Status = status
	return true, nil
}

func applySensitiveColumnRelationTypeUpdate(
	details *datasafesdk.UpdateSensitiveColumnDetails,
	desiredValue string,
	currentValue datasafesdk.SensitiveColumnRelationTypeEnum,
) (bool, error) {
	relationType, err := sensitiveColumnUpdateRelationType(desiredValue)
	if err != nil {
		return false, err
	}
	if relationType == "" || string(relationType) == string(currentValue) {
		return false, nil
	}
	details.RelationType = relationType
	return true, nil
}

func applySensitiveColumnStringUpdate(field **string, specValue string, currentValue *string) bool {
	desired, ok := sensitiveColumnOptionalStringUpdate(specValue, currentValue)
	if !ok {
		return false
	}
	*field = desired
	return true
}

func applySensitiveColumnStringSliceUpdate(field *[]string, desiredValue []string, currentValue []string) bool {
	if desiredValue == nil || slices.Equal(desiredValue, currentValue) {
		return false
	}
	*field = append([]string(nil), desiredValue...)
	return true
}

func validateSensitiveColumnCreateOnlyDrift(
	resource *datasafev1beta1.SensitiveColumn,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("SensitiveColumn resource is nil")
	}
	current, ok := sensitiveColumnFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current SensitiveColumn response does not expose a SensitiveColumn body")
	}
	return validateSensitiveColumnCreateOnlyDriftForCurrent(resource, current)
}

func validateSensitiveColumnCreateOnlyDriftForCurrent(
	resource *datasafev1beta1.SensitiveColumn,
	current datasafesdk.SensitiveColumn,
) error {
	drift, err := sensitiveColumnCreateOnlyDriftFields(resource, current)
	if err != nil {
		return err
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("SensitiveColumn create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func sensitiveColumnCreateOnlyDriftFields(
	resource *datasafev1beta1.SensitiveColumn,
	current datasafesdk.SensitiveColumn,
) ([]string, error) {
	identity, err := resolveSensitiveColumnIdentity(resource)
	if err != nil {
		return nil, err
	}
	resolved := identity.(sensitiveColumnIdentity)
	checks := []struct {
		name    string
		desired string
		current string
	}{
		{name: "sensitiveDataModelId", desired: resolved.sensitiveDataModelID, current: sensitiveColumnStringValue(current.SensitiveDataModelId)},
		{name: "schemaName", desired: resource.Spec.SchemaName, current: sensitiveColumnStringValue(current.SchemaName)},
		{name: "objectName", desired: resource.Spec.ObjectName, current: sensitiveColumnStringValue(current.ObjectName)},
		{name: "columnName", desired: resource.Spec.ColumnName, current: sensitiveColumnStringValue(current.ColumnName)},
		{name: "appName", desired: resource.Spec.AppName, current: sensitiveColumnStringValue(current.AppName)},
		{name: "objectType", desired: resource.Spec.ObjectType, current: string(current.ObjectType)},
	}
	var drift []string
	for _, check := range checks {
		desired := strings.ToUpper(strings.TrimSpace(check.desired))
		current := strings.ToUpper(strings.TrimSpace(check.current))
		if desired != "" && desired != current {
			drift = append(drift, check.name)
		}
	}
	return drift, nil
}

func getSensitiveColumnWorkRequest(
	ctx context.Context,
	client sensitiveColumnOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("SensitiveColumn work request OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveSensitiveColumnWorkRequestAction(workRequest any) (string, error) {
	current, ok := sensitiveColumnWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("work request response does not expose a Data Safe WorkRequest body")
	}
	return string(current.OperationType), nil
}

func recoverSensitiveColumnKeyFromWorkRequest(
	resource *datasafev1beta1.SensitiveColumn,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if key := currentSensitiveColumnKey(resource); key != "" {
		return key, nil
	}
	if key := sensitiveColumnKeyFromWorkRequestResources(workRequest, phase); key != "" {
		return key, nil
	}
	identity, err := resolveSensitiveColumnIdentity(resource)
	if err != nil {
		return "", err
	}
	return syntheticSensitiveColumnKey(identity.(sensitiveColumnIdentity)), nil
}

func sensitiveColumnKeyFromWorkRequestResources(workRequest any, phase shared.OSOKAsyncPhase) string {
	current, ok := sensitiveColumnWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	for _, impacted := range current.Resources {
		if !sensitiveColumnWorkRequestActionMatchesPhase(impacted.ActionType, phase) {
			continue
		}
		if key := sensitiveColumnStringValue(impacted.Identifier); key != "" {
			return key
		}
	}
	return ""
}

func sensitiveColumnWorkRequestActionMatchesPhase(
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

func sensitiveColumnWorkRequestFromAny(value any) (datasafesdk.WorkRequest, bool) {
	switch current := value.(type) {
	case datasafesdk.WorkRequest:
		return current, sensitiveColumnWorkRequestPresent(current)
	case *datasafesdk.WorkRequest:
		return sensitiveColumnWorkRequestFromPointer(current)
	case datasafesdk.GetWorkRequestResponse:
		return current.WorkRequest, sensitiveColumnWorkRequestPresent(current.WorkRequest)
	case *datasafesdk.GetWorkRequestResponse:
		return sensitiveColumnWorkRequestFromResponsePointer(current)
	default:
		return datasafesdk.WorkRequest{}, false
	}
}

func sensitiveColumnWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, ok := sensitiveColumnWorkRequestFromAny(workRequest)
	if !ok {
		return nil, fmt.Errorf("SensitiveColumn work request response does not expose a Data Safe WorkRequest body")
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, sensitiveColumnWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        string(current.OperationType),
		RawOperationType: string(current.OperationType),
		WorkRequestID:    sensitiveColumnStringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func pendingSensitiveColumnWriteWorkRequest(resource *datasafev1beta1.SensitiveColumn) (*shared.OSOKAsyncOperation, bool) {
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

func markSensitiveColumnWriteWorkRequestBeforeDelete(
	resource *datasafev1beta1.SensitiveColumn,
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

func sensitiveColumnPendingWriteBeforeDeleteMessage(current *shared.OSOKAsyncOperation) string {
	if current == nil {
		return "SensitiveColumn write work request is still in progress; retaining finalizer before delete"
	}
	if workRequestID := strings.TrimSpace(current.WorkRequestID); workRequestID != "" {
		return fmt.Sprintf("SensitiveColumn %s work request %s is still in progress; retaining finalizer before delete", current.Phase, workRequestID)
	}
	return fmt.Sprintf("SensitiveColumn %s work request is still in progress; retaining finalizer before delete", current.Phase)
}

func sensitiveColumnWorkRequestFromPointer(current *datasafesdk.WorkRequest) (datasafesdk.WorkRequest, bool) {
	if current == nil {
		return datasafesdk.WorkRequest{}, false
	}
	return *current, sensitiveColumnWorkRequestPresent(*current)
}

func sensitiveColumnWorkRequestFromResponsePointer(
	current *datasafesdk.GetWorkRequestResponse,
) (datasafesdk.WorkRequest, bool) {
	if current == nil {
		return datasafesdk.WorkRequest{}, false
	}
	return current.WorkRequest, sensitiveColumnWorkRequestPresent(current.WorkRequest)
}

func sensitiveColumnWorkRequestPresent(current datasafesdk.WorkRequest) bool {
	return current.Id != nil || current.OperationType != ""
}

type sensitiveColumnProjectedListResponse struct {
	Items []map[string]any
}

func installSensitiveColumnProjectedReadOperations(hooks *SensitiveColumnRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Get.Call != nil {
		getCall := hooks.Get.Call
		hooks.Read.Get = &generatedruntime.Operation{
			NewRequest: func() any { return &datasafesdk.GetSensitiveColumnRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Get.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := getCall(ctx, *request.(*datasafesdk.GetSensitiveColumnRequest))
				if err != nil {
					return nil, err
				}
				return sensitiveColumnStatusProjection(response.SensitiveColumn), nil
			},
		}
	}
	if hooks.List.Call != nil {
		listCall := hooks.List.Call
		hooks.Read.List = &generatedruntime.Operation{
			NewRequest: func() any { return &datasafesdk.ListSensitiveColumnsRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.List.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := listCall(ctx, *request.(*datasafesdk.ListSensitiveColumnsRequest))
				if err != nil {
					return nil, err
				}
				items := make([]map[string]any, 0, len(response.Items))
				for _, item := range response.Items {
					items = append(items, sensitiveColumnStatusProjection(sensitiveColumnFromSummary(item)))
				}
				return sensitiveColumnProjectedListResponse{Items: items}, nil
			},
		}
	}
}

func listSensitiveColumnsAllPages(
	call func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error),
) func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
		if call == nil {
			return datasafesdk.ListSensitiveColumnsResponse{}, fmt.Errorf("SensitiveColumn list operation is not configured")
		}
		var combined datasafesdk.ListSensitiveColumnsResponse
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

func sensitiveColumnDeleteConfirmRead(
	get func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error),
	list func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error),
) func(context.Context, *datasafev1beta1.SensitiveColumn, string) (any, error) {
	return func(ctx context.Context, resource *datasafev1beta1.SensitiveColumn, currentID string) (any, error) {
		key := firstNonEmptySensitiveColumnString(currentID, currentSensitiveColumnKey(resource))
		identity, err := sensitiveColumnDeleteConfirmIdentity(resource, key)
		if err != nil {
			return nil, err
		}
		if sensitiveColumnShouldConfirmDeleteByKey(key) {
			return sensitiveColumnConfirmDeleteByKey(ctx, get, identity, key)
		}
		return sensitiveColumnConfirmDeleteByList(ctx, list, resource)
	}
}

func sensitiveColumnDeleteConfirmIdentity(
	resource *datasafev1beta1.SensitiveColumn,
	key string,
) (sensitiveColumnIdentity, error) {
	identity, err := resolveSensitiveColumnIdentityBeforeDelete(resource, key)
	if err != nil {
		return sensitiveColumnIdentity{}, err
	}
	if identity.sensitiveDataModelID == "" {
		return sensitiveColumnIdentity{}, fmt.Errorf("SensitiveColumn delete confirmation requires %s or recorded status.sensitiveDataModelId", SensitiveColumnSensitiveDataModelIDAnnotation)
	}
	return identity, nil
}

func sensitiveColumnShouldConfirmDeleteByKey(key string) bool {
	key = strings.TrimSpace(key)
	return key != "" && !isSyntheticSensitiveColumnKey(key)
}

func sensitiveColumnConfirmDeleteByKey(
	ctx context.Context,
	get func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error),
	identity sensitiveColumnIdentity,
	key string,
) (any, error) {
	if get == nil {
		return nil, fmt.Errorf("SensitiveColumn delete confirmation requires a Get operation")
	}
	response, err := get(ctx, datasafesdk.GetSensitiveColumnRequest{
		SensitiveDataModelId: common.String(identity.sensitiveDataModelID),
		SensitiveColumnKey:   common.String(key),
	})
	return sensitiveColumnDeleteConfirmReadResponse(response, err)
}

func sensitiveColumnConfirmDeleteByList(
	ctx context.Context,
	list func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error),
	resource *datasafev1beta1.SensitiveColumn,
) (any, error) {
	listIdentity, err := sensitiveColumnDeleteListIdentity(resource)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("SensitiveColumn delete confirmation requires a List operation")
	}
	response, err := list(ctx, datasafesdk.ListSensitiveColumnsRequest{
		SensitiveDataModelId: common.String(listIdentity.sensitiveDataModelID),
		SchemaName:           []string{listIdentity.schemaName},
		ObjectName:           []string{listIdentity.objectName},
		ColumnName:           []string{listIdentity.columnName},
	})
	if err != nil {
		return sensitiveColumnDeleteConfirmReadResponse(response, err)
	}
	return sensitiveColumnSelectListItem(resource, response.Items)
}

func sensitiveColumnDeleteListIdentity(resource *datasafev1beta1.SensitiveColumn) (sensitiveColumnIdentity, error) {
	identity, err := resolveSensitiveColumnIdentityBeforeDelete(resource, "")
	if err != nil {
		return sensitiveColumnIdentity{}, err
	}
	if identity.sensitiveDataModelID == "" {
		return sensitiveColumnIdentity{}, fmt.Errorf("SensitiveColumn delete list resolution requires %s or recorded status.sensitiveDataModelId", SensitiveColumnSensitiveDataModelIDAnnotation)
	}
	if identity.schemaName == "" || identity.objectName == "" || identity.columnName == "" {
		return sensitiveColumnIdentity{}, fmt.Errorf("SensitiveColumn delete list resolution requires recorded or desired schemaName, objectName, and columnName")
	}
	return identity, nil
}

func sensitiveColumnDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return ambiguousSensitiveColumnNotFound("delete confirmation", err), nil
	}
	return nil, err
}

func sensitiveColumnSelectListItem(
	resource *datasafev1beta1.SensitiveColumn,
	items []datasafesdk.SensitiveColumnSummary,
) (datasafesdk.SensitiveColumnSummary, error) {
	identity, err := sensitiveColumnDeleteListIdentity(resource)
	if err != nil {
		return datasafesdk.SensitiveColumnSummary{}, err
	}
	var matches []datasafesdk.SensitiveColumnSummary
	for _, item := range items {
		if sensitiveColumnSummaryMatchesIdentity(identity, item) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return datasafesdk.SensitiveColumnSummary{}, errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			Description:    "SensitiveColumn delete confirmation found no matching sensitive column",
		}
	case 1:
		return matches[0], nil
	default:
		return datasafesdk.SensitiveColumnSummary{}, fmt.Errorf("SensitiveColumn delete confirmation found multiple matching sensitive columns")
	}
}

func sensitiveColumnSummaryMatchesIdentity(
	identity sensitiveColumnIdentity,
	item datasafesdk.SensitiveColumnSummary,
) bool {
	if identity.sensitiveDataModelID != "" && identity.sensitiveDataModelID != sensitiveColumnStringValue(item.SensitiveDataModelId) {
		return false
	}
	return strings.TrimSpace(identity.schemaName) == sensitiveColumnStringValue(item.SchemaName) &&
		strings.TrimSpace(identity.objectName) == sensitiveColumnStringValue(item.ObjectName) &&
		strings.TrimSpace(identity.columnName) == sensitiveColumnStringValue(item.ColumnName)
}

func handleSensitiveColumnDeleteError(resource *datasafev1beta1.SensitiveColumn, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return ambiguousSensitiveColumnNotFound("delete path", err)
}

func applySensitiveColumnDeleteOutcome(
	resource *datasafev1beta1.SensitiveColumn,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if ambiguous, ok := sensitiveColumnAmbiguousNotFoundResponse(response); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}
	current, ok := sensitiveColumnFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	recordSensitiveColumnFromSDK(resource, current)
	if current.LifecycleState == datasafesdk.SensitiveColumnLifecycleStateActive {
		if stage == generatedruntime.DeleteConfirmStageAfterRequest || sensitiveColumnDeleteIsPending(resource) {
			markSensitiveColumnTerminating(resource, current)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markSensitiveColumnTerminating(resource *datasafev1beta1.SensitiveColumn, response any) {
	if resource == nil {
		return
	}
	if current, ok := sensitiveColumnFromResponse(response); ok {
		recordSensitiveColumnFromSDK(resource, current)
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = sensitiveColumnDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	rawState := strings.TrimSpace(resource.Status.LifecycleState)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         sensitiveColumnDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", sensitiveColumnDeletePendingMessage, loggerutil.OSOKLogger{})
}

func markSensitiveColumnDeleted(resource *datasafev1beta1.SensitiveColumn, message string) {
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

func sensitiveColumnDeleteIsPending(resource *datasafev1beta1.SensitiveColumn) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete && current.NormalizedClass == shared.OSOKAsyncClassPending
}

func projectSensitiveColumnStatus(resource *datasafev1beta1.SensitiveColumn, response any) error {
	if resource == nil {
		return fmt.Errorf("SensitiveColumn resource is nil")
	}
	current, ok := sensitiveColumnFromResponse(response)
	if !ok {
		return nil
	}
	status := datasafev1beta1.SensitiveColumnStatus{OsokStatus: resource.Status.OsokStatus}
	payload, err := json.Marshal(sensitiveColumnStatusProjection(current))
	if err != nil {
		return fmt.Errorf("marshal SensitiveColumn status projection: %w", err)
	}
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("project SensitiveColumn status: %w", err)
	}
	resource.Status = status
	return nil
}

func sensitiveColumnStatusProjection(current datasafesdk.SensitiveColumn) map[string]any {
	values := make(map[string]any)
	if key := sensitiveColumnStringValue(current.Key); key != "" {
		values["id"] = key
		values["ocid"] = key
		values["key"] = key
	}
	setSensitiveColumnProjectionString(values, "sensitiveDataModelId", current.SensitiveDataModelId)
	setSensitiveColumnProjectionTime(values, "timeCreated", current.TimeCreated)
	setSensitiveColumnProjectionTime(values, "timeUpdated", current.TimeUpdated)
	setSensitiveColumnProjectionEnum(values, "lifecycleState", string(current.LifecycleState))
	setSensitiveColumnProjectionString(values, "appName", current.AppName)
	setSensitiveColumnProjectionString(values, "schemaName", current.SchemaName)
	setSensitiveColumnProjectionString(values, "objectName", current.ObjectName)
	setSensitiveColumnProjectionString(values, "columnName", current.ColumnName)
	setSensitiveColumnProjectionEnum(values, "objectType", string(current.ObjectType))
	setSensitiveColumnProjectionString(values, "dataType", current.DataType)
	setSensitiveColumnProjectionEnum(values, "sdkStatus", string(current.Status))
	setSensitiveColumnProjectionEnum(values, "source", string(current.Source))
	setSensitiveColumnProjectionEnum(values, "relationType", string(current.RelationType))
	if current.EstimatedDataValueCount != nil {
		values["estimatedDataValueCount"] = *current.EstimatedDataValueCount
	}
	setSensitiveColumnProjectionString(values, "lifecycleDetails", current.LifecycleDetails)
	setSensitiveColumnProjectionString(values, "sensitiveTypeId", current.SensitiveTypeId)
	if current.ParentColumnKeys != nil {
		values["parentColumnKeys"] = append([]string(nil), current.ParentColumnKeys...)
	}
	setSensitiveColumnProjectionEnum(values, "confidenceLevel", string(current.ConfidenceLevel))
	if current.ConfidenceLevelDetails != nil {
		values["confidenceLevelDetails"] = append([]interface{}(nil), current.ConfidenceLevelDetails...)
	}
	if current.AppDefinedChildColumnKeys != nil {
		values["appDefinedChildColumnKeys"] = append([]string(nil), current.AppDefinedChildColumnKeys...)
	}
	if current.DbDefinedChildColumnKeys != nil {
		values["dbDefinedChildColumnKeys"] = append([]string(nil), current.DbDefinedChildColumnKeys...)
	}
	if current.ColumnGroups != nil {
		values["columnGroups"] = append([]string(nil), current.ColumnGroups...)
	}
	return values
}

func setSensitiveColumnProjectionString(values map[string]any, key string, value *string) {
	if value := sensitiveColumnStringValue(value); value != "" {
		values[key] = value
	}
}

func setSensitiveColumnProjectionEnum(values map[string]any, key string, value string) {
	if value := strings.TrimSpace(value); value != "" {
		values[key] = value
	}
}

func setSensitiveColumnProjectionTime(values map[string]any, key string, value *common.SDKTime) {
	if value := sensitiveColumnSDKTimeString(value); value != "" {
		values[key] = value
	}
}

func recordSensitiveColumnFromSDK(resource *datasafev1beta1.SensitiveColumn, current datasafesdk.SensitiveColumn) {
	if resource == nil {
		return
	}
	recordSensitiveColumnIdentityStatus(resource, current)
	recordSensitiveColumnNameStatus(resource, current)
	recordSensitiveColumnLifecycleStatus(resource, current)
	recordSensitiveColumnClassificationStatus(resource, current)
	recordSensitiveColumnRelationStatus(resource, current)
}

func recordSensitiveColumnIdentityStatus(resource *datasafev1beta1.SensitiveColumn, current datasafesdk.SensitiveColumn) {
	if key := sensitiveColumnStringValue(current.Key); key != "" {
		resource.Status.Key = key
		resource.Status.OsokStatus.Ocid = shared.OCID(key)
	}
	if value := sensitiveColumnStringValue(current.SensitiveDataModelId); value != "" {
		resource.Status.SensitiveDataModelId = value
	}
	if value := sensitiveColumnStringValue(current.SensitiveTypeId); value != "" {
		resource.Status.SensitiveTypeId = value
	}
}

func recordSensitiveColumnLifecycleStatus(resource *datasafev1beta1.SensitiveColumn, current datasafesdk.SensitiveColumn) {
	if state := strings.TrimSpace(string(current.LifecycleState)); state != "" {
		resource.Status.LifecycleState = state
	}
	if value := sensitiveColumnSDKTimeString(current.TimeCreated); value != "" {
		resource.Status.TimeCreated = value
	}
	if value := sensitiveColumnSDKTimeString(current.TimeUpdated); value != "" {
		resource.Status.TimeUpdated = value
	}
	if value := sensitiveColumnStringValue(current.LifecycleDetails); value != "" {
		resource.Status.LifecycleDetails = value
	}
}

func recordSensitiveColumnNameStatus(resource *datasafev1beta1.SensitiveColumn, current datasafesdk.SensitiveColumn) {
	if value := sensitiveColumnStringValue(current.AppName); value != "" {
		resource.Status.AppName = value
	}
	if value := sensitiveColumnStringValue(current.SchemaName); value != "" {
		resource.Status.SchemaName = value
	}
	if value := sensitiveColumnStringValue(current.ObjectName); value != "" {
		resource.Status.ObjectName = value
	}
	if value := sensitiveColumnStringValue(current.ColumnName); value != "" {
		resource.Status.ColumnName = value
	}
	if value := strings.TrimSpace(string(current.ObjectType)); value != "" {
		resource.Status.ObjectType = value
	}
	if value := sensitiveColumnStringValue(current.DataType); value != "" {
		resource.Status.DataType = value
	}
}

func recordSensitiveColumnClassificationStatus(resource *datasafev1beta1.SensitiveColumn, current datasafesdk.SensitiveColumn) {
	if sdkStatus := strings.TrimSpace(string(current.Status)); sdkStatus != "" {
		resource.Status.Status = sdkStatus
	}
	if value := strings.TrimSpace(string(current.Source)); value != "" {
		resource.Status.Source = value
	}
	if current.EstimatedDataValueCount != nil {
		resource.Status.EstimatedDataValueCount = *current.EstimatedDataValueCount
	}
	if value := strings.TrimSpace(string(current.ConfidenceLevel)); value != "" {
		resource.Status.ConfidenceLevel = value
	}
	if current.ConfidenceLevelDetails != nil {
		resource.Status.ConfidenceLevelDetails = sensitiveColumnJSONValues(current.ConfidenceLevelDetails)
	}
}

func recordSensitiveColumnRelationStatus(resource *datasafev1beta1.SensitiveColumn, current datasafesdk.SensitiveColumn) {
	if value := strings.TrimSpace(string(current.RelationType)); value != "" {
		resource.Status.RelationType = value
	}
	if current.ParentColumnKeys != nil {
		resource.Status.ParentColumnKeys = append([]string(nil), current.ParentColumnKeys...)
	}
	if current.AppDefinedChildColumnKeys != nil {
		resource.Status.AppDefinedChildColumnKeys = append([]string(nil), current.AppDefinedChildColumnKeys...)
	}
	if current.DbDefinedChildColumnKeys != nil {
		resource.Status.DbDefinedChildColumnKeys = append([]string(nil), current.DbDefinedChildColumnKeys...)
	}
	if current.ColumnGroups != nil {
		resource.Status.ColumnGroups = append([]string(nil), current.ColumnGroups...)
	}
}

func sensitiveColumnJSONValues(values []interface{}) []shared.JSONValue {
	if values == nil {
		return nil
	}
	out := make([]shared.JSONValue, 0, len(values))
	for _, value := range values {
		payload, err := json.Marshal(value)
		if err != nil {
			payload = []byte("null")
		}
		out = append(out, shared.JSONValue{Raw: append([]byte(nil), payload...)})
	}
	return out
}

func sensitiveColumnSDKTimeString(value *common.SDKTime) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func validateSensitiveColumnCreateSpec(spec datasafev1beta1.SensitiveColumnSpec) error {
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
		return fmt.Errorf("SensitiveColumn spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func normalizeSensitiveColumnDesiredState(resource *datasafev1beta1.SensitiveColumn, _ any) {
	normalizeSensitiveColumnSpec(resource)
}

func normalizeSensitiveColumnSpec(resource *datasafev1beta1.SensitiveColumn) {
	if resource == nil {
		return
	}
	resource.Spec.SchemaName = strings.TrimSpace(resource.Spec.SchemaName)
	resource.Spec.ObjectName = strings.TrimSpace(resource.Spec.ObjectName)
	resource.Spec.ColumnName = strings.TrimSpace(resource.Spec.ColumnName)
	resource.Spec.AppName = strings.TrimSpace(resource.Spec.AppName)
	resource.Spec.ObjectType = strings.ToUpper(strings.TrimSpace(resource.Spec.ObjectType))
	resource.Spec.DataType = strings.TrimSpace(resource.Spec.DataType)
	resource.Spec.Status = strings.ToUpper(strings.TrimSpace(resource.Spec.Status))
	resource.Spec.SensitiveTypeId = strings.TrimSpace(resource.Spec.SensitiveTypeId)
	resource.Spec.RelationType = strings.ToUpper(strings.TrimSpace(resource.Spec.RelationType))
	resource.Spec.ParentColumnKeys = trimSensitiveColumnStringSlice(resource.Spec.ParentColumnKeys)
	resource.Spec.AppDefinedChildColumnKeys = trimSensitiveColumnStringSlice(resource.Spec.AppDefinedChildColumnKeys)
	resource.Spec.DbDefinedChildColumnKeys = trimSensitiveColumnStringSlice(resource.Spec.DbDefinedChildColumnKeys)
}

func sensitiveColumnCreateObjectType(value string) (datasafesdk.CreateSensitiveColumnDetailsObjectTypeEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	objectType, ok := datasafesdk.GetMappingCreateSensitiveColumnDetailsObjectTypeEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported SensitiveColumn objectType %q", value)
	}
	return objectType, nil
}

func sensitiveColumnCreateStatus(value string) (datasafesdk.CreateSensitiveColumnDetailsStatusEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	status, ok := datasafesdk.GetMappingCreateSensitiveColumnDetailsStatusEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported SensitiveColumn status %q", value)
	}
	return status, nil
}

func sensitiveColumnUpdateStatus(value string) (datasafesdk.UpdateSensitiveColumnDetailsStatusEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	status, ok := datasafesdk.GetMappingUpdateSensitiveColumnDetailsStatusEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported SensitiveColumn status %q", value)
	}
	return status, nil
}

func sensitiveColumnCreateRelationType(value string) (datasafesdk.CreateSensitiveColumnDetailsRelationTypeEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	relationType, ok := datasafesdk.GetMappingCreateSensitiveColumnDetailsRelationTypeEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported SensitiveColumn relationType %q", value)
	}
	return relationType, nil
}

func sensitiveColumnUpdateRelationType(value string) (datasafesdk.UpdateSensitiveColumnDetailsRelationTypeEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	relationType, ok := datasafesdk.GetMappingUpdateSensitiveColumnDetailsRelationTypeEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported SensitiveColumn relationType %q", value)
	}
	return relationType, nil
}

func sensitiveColumnOptionalStringUpdate(specValue string, currentValue *string) (*string, bool) {
	desired := strings.TrimSpace(specValue)
	if desired == "" {
		return nil, false
	}
	if desired == sensitiveColumnStringValue(currentValue) {
		return nil, false
	}
	return common.String(desired), true
}

func sensitiveColumnFromResponse(response any) (datasafesdk.SensitiveColumn, bool) {
	switch current := response.(type) {
	case datasafesdk.GetSensitiveColumnResponse:
		return sensitiveColumnPresent(current.SensitiveColumn)
	case *datasafesdk.GetSensitiveColumnResponse:
		return sensitiveColumnFromGetResponsePointer(current)
	case datasafesdk.SensitiveColumn:
		return sensitiveColumnPresent(current)
	case *datasafesdk.SensitiveColumn:
		return sensitiveColumnFromPointer(current)
	case datasafesdk.SensitiveColumnSummary:
		return sensitiveColumnPresent(sensitiveColumnFromSummary(current))
	case *datasafesdk.SensitiveColumnSummary:
		return sensitiveColumnFromSummaryPointer(current)
	case map[string]any:
		return sensitiveColumnFromProjectionMap(current)
	default:
		return datasafesdk.SensitiveColumn{}, false
	}
}

func sensitiveColumnFromGetResponsePointer(
	current *datasafesdk.GetSensitiveColumnResponse,
) (datasafesdk.SensitiveColumn, bool) {
	if current == nil {
		return datasafesdk.SensitiveColumn{}, false
	}
	return sensitiveColumnPresent(current.SensitiveColumn)
}

func sensitiveColumnFromPointer(current *datasafesdk.SensitiveColumn) (datasafesdk.SensitiveColumn, bool) {
	if current == nil {
		return datasafesdk.SensitiveColumn{}, false
	}
	return sensitiveColumnPresent(*current)
}

func sensitiveColumnFromSummaryPointer(current *datasafesdk.SensitiveColumnSummary) (datasafesdk.SensitiveColumn, bool) {
	if current == nil {
		return datasafesdk.SensitiveColumn{}, false
	}
	return sensitiveColumnPresent(sensitiveColumnFromSummary(*current))
}

func sensitiveColumnPresent(current datasafesdk.SensitiveColumn) (datasafesdk.SensitiveColumn, bool) {
	return current, current.Key != nil
}

func sensitiveColumnFromProjectionMap(values map[string]any) (datasafesdk.SensitiveColumn, bool) {
	if values == nil {
		return datasafesdk.SensitiveColumn{}, false
	}
	key := firstNonEmptySensitiveColumnString(
		sensitiveColumnMapString(values, "key"),
		sensitiveColumnMapString(values, "id"),
		sensitiveColumnMapString(values, "ocid"),
	)
	if key == "" {
		return datasafesdk.SensitiveColumn{}, false
	}
	current := datasafesdk.SensitiveColumn{
		Key:                     common.String(key),
		SensitiveDataModelId:    sensitiveColumnMapStringPointer(values, "sensitiveDataModelId"),
		LifecycleState:          datasafesdk.SensitiveColumnLifecycleStateEnum(sensitiveColumnMapString(values, "lifecycleState")),
		AppName:                 sensitiveColumnMapStringPointer(values, "appName"),
		SchemaName:              sensitiveColumnMapStringPointer(values, "schemaName"),
		ObjectName:              sensitiveColumnMapStringPointer(values, "objectName"),
		ColumnName:              sensitiveColumnMapStringPointer(values, "columnName"),
		ObjectType:              datasafesdk.SensitiveColumnObjectTypeEnum(sensitiveColumnMapString(values, "objectType")),
		DataType:                sensitiveColumnMapStringPointer(values, "dataType"),
		Status:                  datasafesdk.SensitiveColumnStatusEnum(sensitiveColumnMapString(values, "sdkStatus")),
		Source:                  datasafesdk.SensitiveColumnSourceEnum(sensitiveColumnMapString(values, "source")),
		RelationType:            datasafesdk.SensitiveColumnRelationTypeEnum(sensitiveColumnMapString(values, "relationType")),
		EstimatedDataValueCount: sensitiveColumnMapInt64Pointer(values, "estimatedDataValueCount"),
		LifecycleDetails:        sensitiveColumnMapStringPointer(values, "lifecycleDetails"),
		SensitiveTypeId:         sensitiveColumnMapStringPointer(values, "sensitiveTypeId"),
		ParentColumnKeys:        sensitiveColumnMapStringSlice(values, "parentColumnKeys"),
		ConfidenceLevel:         datasafesdk.ConfidenceLevelEnumEnum(sensitiveColumnMapString(values, "confidenceLevel")),
		ConfidenceLevelDetails:  sensitiveColumnMapAnySlice(values, "confidenceLevelDetails"),
		AppDefinedChildColumnKeys: sensitiveColumnMapStringSlice(
			values,
			"appDefinedChildColumnKeys",
		),
		DbDefinedChildColumnKeys: sensitiveColumnMapStringSlice(values, "dbDefinedChildColumnKeys"),
		ColumnGroups:             sensitiveColumnMapStringSlice(values, "columnGroups"),
	}
	return current, true
}

func sensitiveColumnFromSummary(summary datasafesdk.SensitiveColumnSummary) datasafesdk.SensitiveColumn {
	return datasafesdk.SensitiveColumn{
		Key:                     summary.Key,
		SensitiveDataModelId:    summary.SensitiveDataModelId,
		LifecycleState:          summary.LifecycleState,
		TimeCreated:             summary.TimeCreated,
		TimeUpdated:             summary.TimeUpdated,
		AppName:                 summary.AppName,
		SchemaName:              summary.SchemaName,
		ObjectName:              summary.ObjectName,
		ColumnName:              summary.ColumnName,
		ObjectType:              datasafesdk.SensitiveColumnObjectTypeEnum(summary.ObjectType),
		DataType:                summary.DataType,
		Status:                  datasafesdk.SensitiveColumnStatusEnum(summary.Status),
		Source:                  datasafesdk.SensitiveColumnSourceEnum(summary.Source),
		RelationType:            datasafesdk.SensitiveColumnRelationTypeEnum(summary.RelationType),
		EstimatedDataValueCount: summary.EstimatedDataValueCount,
		LifecycleDetails:        summary.LifecycleDetails,
		SensitiveTypeId:         summary.SensitiveTypeId,
		ParentColumnKeys:        append([]string(nil), summary.ParentColumnKeys...),
		ConfidenceLevel:         summary.ConfidenceLevel,
	}
}

func ambiguousSensitiveColumnNotFound(operation string, err error) sensitiveColumnAmbiguousNotFoundError {
	message := fmt.Sprintf(
		"SensitiveColumn %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s",
		strings.TrimSpace(operation),
		err.Error(),
	)
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return sensitiveColumnAmbiguousNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return sensitiveColumnAmbiguousNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func sensitiveColumnAmbiguousNotFoundResponse(value any) (sensitiveColumnAmbiguousNotFoundError, bool) {
	switch typed := value.(type) {
	case sensitiveColumnAmbiguousNotFoundError:
		return typed, true
	case *sensitiveColumnAmbiguousNotFoundError:
		if typed == nil {
			return sensitiveColumnAmbiguousNotFoundError{}, false
		}
		return *typed, true
	default:
		return sensitiveColumnAmbiguousNotFoundError{}, false
	}
}

func sensitiveColumnModelIDAnnotation(resource *datasafev1beta1.SensitiveColumn) string {
	if resource == nil {
		return ""
	}
	return sensitiveColumnAnnotationValue(
		resource.Annotations,
		SensitiveColumnSensitiveDataModelIDAnnotation,
		sensitiveColumnLegacySensitiveDataModelIDAnnotation,
	)
}

func sensitiveColumnAnnotationValue(annotations map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func currentSensitiveColumnModelID(resource *datasafev1beta1.SensitiveColumn) string {
	if resource == nil {
		return ""
	}
	return firstNonEmptySensitiveColumnString(resource.Status.SensitiveDataModelId, sensitiveColumnModelIDAnnotation(resource))
}

func currentSensitiveColumnKey(resource *datasafev1beta1.SensitiveColumn) string {
	if resource == nil {
		return ""
	}
	if key := strings.TrimSpace(resource.Status.Key); key != "" {
		return key
	}
	if key := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); key != "" && !isSyntheticSensitiveColumnKey(key) {
		return key
	}
	return ""
}

func syntheticSensitiveColumnKey(identity sensitiveColumnIdentity) string {
	parts := []string{
		strings.TrimSpace(identity.sensitiveDataModelID),
		strings.TrimSpace(identity.schemaName),
		strings.TrimSpace(identity.objectName),
		strings.TrimSpace(identity.columnName),
	}
	for _, part := range parts {
		if part == "" {
			return ""
		}
	}
	return sensitiveColumnSyntheticKeyPrefix + strings.Join(parts, "/")
}

func isSyntheticSensitiveColumnKey(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), sensitiveColumnSyntheticKeyPrefix)
}

func isSensitiveColumnListNoMatch(err error) bool {
	if err == nil {
		return false
	}
	var notFound errorutil.NotFoundOciError
	return errors.As(err, &notFound)
}

func trimSensitiveColumnStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	out := values[:0]
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func sensitiveColumnMapString(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func sensitiveColumnMapStringPointer(values map[string]any, key string) *string {
	if value := sensitiveColumnMapString(values, key); value != "" {
		return common.String(value)
	}
	return nil
}

func sensitiveColumnMapInt64Pointer(values map[string]any, key string) *int64 {
	value, ok := values[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case int64:
		return common.Int64(typed)
	case int:
		return common.Int64(int64(typed))
	case float64:
		return common.Int64(int64(typed))
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return common.Int64(parsed)
		}
	}
	return nil
}

func sensitiveColumnMapStringSlice(values map[string]any, key string) []string {
	value, ok := values[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value := strings.TrimSpace(fmt.Sprint(item)); value != "" {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}

func sensitiveColumnMapAnySlice(values map[string]any, key string) []interface{} {
	value, ok := values[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []interface{}:
		return append([]interface{}(nil), typed...)
	case []shared.JSONValue:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			var decoded interface{}
			if err := json.Unmarshal(item.Raw, &decoded); err == nil {
				out = append(out, decoded)
			}
		}
		return out
	default:
		return nil
	}
}

func sensitiveColumnStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptySensitiveColumnString(values ...string) string {
	for _, value := range values {
		if value := strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
