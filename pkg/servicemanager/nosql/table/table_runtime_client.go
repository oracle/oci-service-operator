/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package table

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	nosqlsdk "github.com/oracle/oci-go-sdk/v65/nosql"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const tableRequeueDuration = time.Minute

var errTableNotFound = errors.New("table not found")

type tableOCIClient interface {
	CreateTable(ctx context.Context, request nosqlsdk.CreateTableRequest) (nosqlsdk.CreateTableResponse, error)
	GetTable(ctx context.Context, request nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error)
	ListTables(ctx context.Context, request nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error)
	UpdateTable(ctx context.Context, request nosqlsdk.UpdateTableRequest) (nosqlsdk.UpdateTableResponse, error)
	ChangeTableCompartment(ctx context.Context, request nosqlsdk.ChangeTableCompartmentRequest) (nosqlsdk.ChangeTableCompartmentResponse, error)
	DeleteTable(ctx context.Context, request nosqlsdk.DeleteTableRequest) (nosqlsdk.DeleteTableResponse, error)
}

type explicitTableServiceClient struct {
	log     loggerutil.OSOKLogger
	oci     tableOCIClient
	initErr error
}

var _ TableServiceClient = (*explicitTableServiceClient)(nil)

func newExplicitTableServiceClient(manager *TableServiceManager) TableServiceClient {
	client, err := nosqlsdk.NewNosqlClientWithConfigurationProvider(manager.Provider)
	explicit := &explicitTableServiceClient{
		log: manager.Log,
		oci: client,
	}
	if err != nil {
		explicit.initErr = fmt.Errorf("initialize Table OCI client: %w", err)
	}
	return explicit
}

func newExplicitTableServiceClientWithOCIClient(log loggerutil.OSOKLogger, oci tableOCIClient) *explicitTableServiceClient {
	return &explicitTableServiceClient{
		log: log,
		oci: oci,
	}
}

func (c *explicitTableServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	existing, err := c.resolveExistingTable(ctx, resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if existing == nil {
		return c.createTable(ctx, resource)
	}

	if err := c.projectPayload(resource, existing.payload); err != nil {
		return c.fail(resource, err)
	}

	state := normalizeLifecycle(existing.lifecycleState)
	switch state {
	case string(nosqlsdk.TableLifecycleStateDeleted):
		return c.finishWithLifecycle(resource, existing, shared.OSOKAsyncPhaseDelete), nil
	case string(nosqlsdk.TableLifecycleStateFailed):
		return c.finishWithLifecycle(resource, existing, currentTableAsyncPhase(resource, shared.OSOKAsyncPhaseCreate)), nil
	case string(nosqlsdk.TableLifecycleStateCreating),
		string(nosqlsdk.TableLifecycleStateUpdating),
		string(nosqlsdk.TableLifecycleStateDeleting):
		return c.finishWithLifecycle(resource, existing, tableLifecyclePhase(state)), nil
	}

	if err := c.validateForceNewFields(resource, existing); err != nil {
		return c.fail(resource, err)
	}

	if existing.compartmentID != "" && resource.Spec.CompartmentId != "" && resource.Spec.CompartmentId != existing.compartmentID {
		if err := c.changeTableCompartment(ctx, existing.id, existing.compartmentID, resource.Spec.CompartmentId); err != nil {
			return c.fail(resource, err)
		}

		refreshed, err := c.readTable(ctx, resource, existing.id)
		if err != nil {
			if errors.Is(err, errTableNotFound) {
				return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseUpdate, "OCI table compartment move request accepted", ""), nil
			}
			return c.fail(resource, fmt.Errorf("confirm Table compartment change: %w", err))
		}
		if err := c.projectPayload(resource, refreshed.payload); err != nil {
			return c.fail(resource, err)
		}

		existing = refreshed
		if normalizeLifecycle(existing.lifecycleState) != string(nosqlsdk.TableLifecycleStateActive) || existing.compartmentID != resource.Spec.CompartmentId {
			return c.finishWithLifecycle(resource, existing, shared.OSOKAsyncPhaseUpdate), nil
		}
	}

	updateDetails, updateNeeded := buildUpdateDetails(resource, existing)
	if !updateNeeded {
		return c.finishWithLifecycle(resource, existing, shared.OSOKAsyncPhaseUpdate), nil
	}

	if err := c.updateTable(ctx, existing.id, updateDetails); err != nil {
		return c.fail(resource, err)
	}

	refreshed, err := c.readTable(ctx, resource, existing.id)
	if err != nil {
		if errors.Is(err, errTableNotFound) {
			return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseUpdate, "OCI table update request accepted", ""), nil
		}
		return c.fail(resource, fmt.Errorf("confirm Table update: %w", err))
	}
	if err := c.projectPayload(resource, refreshed.payload); err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, refreshed, shared.OSOKAsyncPhaseUpdate), nil
}

func (c *explicitTableServiceClient) Delete(ctx context.Context, resource *nosqlv1beta1.Table) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	target, err := c.resolveDeleteTarget(ctx, resource)
	if err != nil {
		return false, err
	}
	if target == nil {
		c.markDeleted(resource, "OCI table no longer exists")
		return true, nil
	}

	if err := c.projectPayload(resource, target.payload); err != nil {
		return false, err
	}

	switch normalizeLifecycle(target.lifecycleState) {
	case string(nosqlsdk.TableLifecycleStateDeleted):
		c.markDeleted(resource, "OCI table deleted")
		return true, nil
	case string(nosqlsdk.TableLifecycleStateDeleting):
		c.finishWithLifecycle(resource, target, shared.OSOKAsyncPhaseDelete)
		return false, nil
	}

	if target.id == "" {
		return false, fmt.Errorf("resolve Table delete target: missing table OCID")
	}
	if err := c.deleteTable(ctx, target.id); err != nil {
		if errors.Is(err, errTableNotFound) {
			c.markDeleted(resource, "OCI table no longer exists")
			return true, nil
		}
		return false, err
	}

	refreshed, err := c.readTable(ctx, resource, target.id)
	if err != nil {
		if errors.Is(err, errTableNotFound) {
			c.markDeleted(resource, "OCI table deleted")
			return true, nil
		}
		return false, fmt.Errorf("confirm Table delete: %w", err)
	}
	if err := c.projectPayload(resource, refreshed.payload); err != nil {
		return false, err
	}

	if normalizeLifecycle(refreshed.lifecycleState) == string(nosqlsdk.TableLifecycleStateDeleted) {
		c.markDeleted(resource, "OCI table deleted")
		return true, nil
	}

	c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseDelete, "OCI table delete request accepted", refreshed.lifecycleState)
	return false, nil
}

type tableSnapshot struct {
	id                string
	name              string
	compartmentID     string
	lifecycleState    string
	lifecycleDetails  string
	ddlStatement      string
	isAutoReclaimable bool
	tableLimits       *nosqlsdk.TableLimits
	freeformTags      map[string]string
	definedTags       map[string]map[string]interface{}
	payload           any
}

func snapshotFromTable(table nosqlsdk.Table) *tableSnapshot {
	return &tableSnapshot{
		id:                stringPointerValue(table.Id),
		name:              stringPointerValue(table.Name),
		compartmentID:     stringPointerValue(table.CompartmentId),
		lifecycleState:    string(table.LifecycleState),
		lifecycleDetails:  stringPointerValue(table.LifecycleDetails),
		ddlStatement:      stringPointerValue(table.DdlStatement),
		isAutoReclaimable: boolPointerValue(table.IsAutoReclaimable),
		tableLimits:       table.TableLimits,
		freeformTags:      table.FreeformTags,
		definedTags:       table.DefinedTags,
		payload:           table,
	}
}

func snapshotFromSummary(summary nosqlsdk.TableSummary) *tableSnapshot {
	return &tableSnapshot{
		id:                stringPointerValue(summary.Id),
		name:              stringPointerValue(summary.Name),
		compartmentID:     stringPointerValue(summary.CompartmentId),
		lifecycleState:    string(summary.LifecycleState),
		lifecycleDetails:  stringPointerValue(summary.LifecycleDetails),
		isAutoReclaimable: boolPointerValue(summary.IsAutoReclaimable),
		tableLimits:       summary.TableLimits,
		freeformTags:      summary.FreeformTags,
		definedTags:       summary.DefinedTags,
		payload:           summary,
	}
}

func (c *explicitTableServiceClient) createTable(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
) (servicemanager.OSOKResponse, error) {
	if _, err := c.oci.CreateTable(ctx, buildCreateRequest(resource)); err != nil {
		if isNotFoundErr(err) {
			return c.fail(resource, errTableNotFound)
		}
		return c.fail(resource, fmt.Errorf("create Table: %w", err))
	}

	refreshed, err := c.readTable(ctx, resource, "")
	if err != nil {
		if errors.Is(err, errTableNotFound) {
			return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseCreate, "OCI table create request accepted", ""), nil
		}
		return c.fail(resource, fmt.Errorf("confirm Table create: %w", err))
	}
	if err := c.projectPayload(resource, refreshed.payload); err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, refreshed, shared.OSOKAsyncPhaseCreate), nil
}

func (c *explicitTableServiceClient) resolveExistingTable(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
) (*tableSnapshot, error) {
	if currentID := currentTableID(resource); currentID != "" {
		table, err := c.getTableByID(ctx, currentID)
		if err == nil {
			return table, nil
		}
		if !errors.Is(err, errTableNotFound) {
			return nil, fmt.Errorf("get Table %q: %w", currentID, err)
		}
	}

	summary, err := c.findTableByName(ctx, resource.Spec.CompartmentId, resource.Spec.Name)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}
	if summary.Id == nil || *summary.Id == "" {
		return snapshotFromSummary(*summary), nil
	}

	table, err := c.getTableByID(ctx, *summary.Id)
	if err == nil {
		return table, nil
	}
	if errors.Is(err, errTableNotFound) {
		return snapshotFromSummary(*summary), nil
	}
	return nil, fmt.Errorf("get Table %q: %w", *summary.Id, err)
}

func (c *explicitTableServiceClient) resolveDeleteTarget(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
) (*tableSnapshot, error) {
	if currentID := currentTableID(resource); currentID != "" {
		table, err := c.getTableByID(ctx, currentID)
		if err == nil {
			return table, nil
		}
		if errors.Is(err, errTableNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get Table %q: %w", currentID, err)
	}

	summary, err := c.findTableByName(ctx, resource.Spec.CompartmentId, resource.Spec.Name)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}
	return snapshotFromSummary(*summary), nil
}

func (c *explicitTableServiceClient) readTable(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	preferredID string,
) (*tableSnapshot, error) {
	if preferredID != "" {
		table, err := c.getTableByID(ctx, preferredID)
		if err == nil {
			return table, nil
		}
		if !errors.Is(err, errTableNotFound) {
			return nil, err
		}
	}

	if resource.Spec.Name != "" && resource.Spec.CompartmentId != "" {
		table, err := c.getTableByName(ctx, resource.Spec.CompartmentId, resource.Spec.Name)
		if err == nil {
			return table, nil
		}
		if !errors.Is(err, errTableNotFound) {
			return nil, err
		}
	}

	summary, err := c.findTableByName(ctx, resource.Spec.CompartmentId, resource.Spec.Name)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, errTableNotFound
	}
	return snapshotFromSummary(*summary), nil
}

func (c *explicitTableServiceClient) getTableByID(ctx context.Context, id string) (*tableSnapshot, error) {
	response, err := c.oci.GetTable(ctx, nosqlsdk.GetTableRequest{
		TableNameOrId: common.String(id),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, errTableNotFound
		}
		return nil, err
	}
	return snapshotFromTable(response.Table), nil
}

func (c *explicitTableServiceClient) getTableByName(
	ctx context.Context,
	compartmentID string,
	name string,
) (*tableSnapshot, error) {
	response, err := c.oci.GetTable(ctx, nosqlsdk.GetTableRequest{
		TableNameOrId: common.String(name),
		CompartmentId: common.String(compartmentID),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, errTableNotFound
		}
		return nil, err
	}
	return snapshotFromTable(response.Table), nil
}

func (c *explicitTableServiceClient) findTableByName(
	ctx context.Context,
	compartmentID string,
	name string,
) (*nosqlsdk.TableSummary, error) {
	if compartmentID == "" || name == "" {
		return nil, nil
	}

	response, err := c.oci.ListTables(ctx, nosqlsdk.ListTablesRequest{
		CompartmentId:  common.String(compartmentID),
		Name:           common.String(name),
		LifecycleState: nosqlsdk.ListTablesLifecycleStateAll,
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, errTableNotFound
		}
		return nil, fmt.Errorf("list Table %q: %w", name, err)
	}

	var matches []nosqlsdk.TableSummary
	for _, item := range response.Items {
		if stringPointerValue(item.Name) == name {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("list Table %q returned multiple exact matches", name)
	}
}

func (c *explicitTableServiceClient) changeTableCompartment(
	ctx context.Context,
	tableID string,
	fromCompartmentID string,
	toCompartmentID string,
) error {
	_, err := c.oci.ChangeTableCompartment(ctx, nosqlsdk.ChangeTableCompartmentRequest{
		TableNameOrId: common.String(tableID),
		ChangeTableCompartmentDetails: nosqlsdk.ChangeTableCompartmentDetails{
			ToCompartmentId:   common.String(toCompartmentID),
			FromCompartmentId: optionalString(fromCompartmentID),
		},
	})
	if err != nil && isNotFoundErr(err) {
		return errTableNotFound
	}
	if err != nil {
		return fmt.Errorf("change Table compartment: %w", err)
	}
	return nil
}

func (c *explicitTableServiceClient) updateTable(
	ctx context.Context,
	tableID string,
	details nosqlsdk.UpdateTableDetails,
) error {
	_, err := c.oci.UpdateTable(ctx, nosqlsdk.UpdateTableRequest{
		TableNameOrId:      common.String(tableID),
		UpdateTableDetails: details,
	})
	if err != nil && isNotFoundErr(err) {
		return errTableNotFound
	}
	if err != nil {
		return fmt.Errorf("update Table: %w", err)
	}
	return nil
}

func (c *explicitTableServiceClient) deleteTable(ctx context.Context, tableID string) error {
	_, err := c.oci.DeleteTable(ctx, nosqlsdk.DeleteTableRequest{
		TableNameOrId: common.String(tableID),
		IsIfExists:    common.Bool(true),
	})
	if err != nil && isNotFoundErr(err) {
		return errTableNotFound
	}
	if err != nil {
		return fmt.Errorf("delete Table: %w", err)
	}
	return nil
}

func (c *explicitTableServiceClient) validateForceNewFields(resource *nosqlv1beta1.Table, existing *tableSnapshot) error {
	if existing == nil {
		return nil
	}
	if existing.name != "" && resource.Spec.Name != "" && existing.name != resource.Spec.Name {
		return fmt.Errorf("Table formal semantics require replacement when name changes")
	}
	if existing.isAutoReclaimable != resource.Spec.IsAutoReclaimable {
		return fmt.Errorf("Table formal semantics require replacement when isAutoReclaimable changes")
	}
	return nil
}

func buildCreateRequest(resource *nosqlv1beta1.Table) nosqlsdk.CreateTableRequest {
	request := nosqlsdk.CreateTableRequest{
		CreateTableDetails: nosqlsdk.CreateTableDetails{
			Name:              common.String(resource.Spec.Name),
			CompartmentId:     common.String(resource.Spec.CompartmentId),
			DdlStatement:      common.String(resource.Spec.DdlStatement),
			IsAutoReclaimable: common.Bool(resource.Spec.IsAutoReclaimable),
			FreeformTags:      resource.Spec.FreeformTags,
		},
	}
	if limits := specTableLimits(resource.Spec.TableLimits); limits != nil {
		request.CreateTableDetails.TableLimits = limits
	}
	if resource.Spec.DefinedTags != nil {
		request.CreateTableDetails.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return request
}

func buildUpdateDetails(resource *nosqlv1beta1.Table, existing *tableSnapshot) (nosqlsdk.UpdateTableDetails, bool) {
	var (
		details     nosqlsdk.UpdateTableDetails
		needsUpdate bool
	)

	if isAlterTableDDL(resource.Spec.DdlStatement) {
		// The CR surface stores create-time DDL, but OCI update expects an ALTER statement.
		// Rejecting implicit DDL drift keeps the handwritten path idempotent.
		return details, false
	}

	if limits := specTableLimits(resource.Spec.TableLimits); limits != nil && !sdkTableLimitsEqual(limits, existing.tableLimits) {
		details.TableLimits = limits
		needsUpdate = true
	}

	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(existing.freeformTags, resource.Spec.FreeformTags) {
		details.FreeformTags = resource.Spec.FreeformTags
		needsUpdate = true
	}

	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(existing.definedTags, desiredDefinedTags) {
			details.DefinedTags = desiredDefinedTags
			needsUpdate = true
		}
	}

	return details, needsUpdate
}

func (c *explicitTableServiceClient) projectPayload(resource *nosqlv1beta1.Table, payload any) error {
	if payload == nil {
		return nil
	}

	newStatus := nosqlv1beta1.TableStatus{
		OsokStatus: resource.Status.OsokStatus,
	}
	serialized, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal Table status payload: %w", err)
	}
	if err := json.Unmarshal(serialized, &newStatus); err != nil {
		return fmt.Errorf("project Table status payload: %w", err)
	}
	resource.Status = newStatus
	return nil
}

func (c *explicitTableServiceClient) finishWithLifecycle(resource *nosqlv1beta1.Table, snapshot *tableSnapshot, fallbackPhase shared.OSOKAsyncPhase) servicemanager.OSOKResponse {
	message := lifecycleMessage(snapshot.lifecycleDetails, snapshot.name, "OCI table is active")
	current := servicemanager.NewLifecycleAsyncOperation(&resource.Status.OsokStatus, snapshot.lifecycleState, message, fallbackPhase)
	if current != nil {
		return c.markAsyncOperation(resource, current)
	}
	return c.markCondition(resource, shared.Active, message, false)
}

func (c *explicitTableServiceClient) markCondition(
	resource *nosqlv1beta1.Table,
	condition shared.OSOKConditionType,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	}
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: tableRequeueDuration,
	}
}

func (c *explicitTableServiceClient) markAsyncOperation(resource *nosqlv1beta1.Table, current *shared.OSOKAsyncOperation) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: tableRequeueDuration,
	}
}

func (c *explicitTableServiceClient) markPendingLifecycleOperation(resource *nosqlv1beta1.Table, phase shared.OSOKAsyncPhase, message string, rawStatus string) servicemanager.OSOKResponse {
	return c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       normalizeLifecycle(rawStatus),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *explicitTableServiceClient) markDeleted(resource *nosqlv1beta1.Table, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *explicitTableServiceClient) fail(
	resource *nosqlv1beta1.Table,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil {
		status := &resource.Status.OsokStatus
		now := metav1.Now()
		status.UpdatedAt = &now
		status.Message = err.Error()
		status.Reason = string(shared.Failed)
		if status.Async.Current != nil {
			current := *status.Async.Current
			current.NormalizedClass = shared.OSOKAsyncClassFailed
			current.Message = err.Error()
			current.UpdatedAt = &now
			_ = servicemanager.ApplyAsyncOperation(status, &current, c.log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func currentTableID(resource *nosqlv1beta1.Table) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return resource.Status.Id
}

func normalizeLifecycle(state string) string {
	return strings.ToUpper(strings.TrimSpace(state))
}

func lifecycleMessage(detail string, name string, fallback string) string {
	if strings.TrimSpace(detail) != "" {
		return detail
	}
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

func tableLifecyclePhase(state string) shared.OSOKAsyncPhase {
	switch state {
	case string(nosqlsdk.TableLifecycleStateUpdating):
		return shared.OSOKAsyncPhaseUpdate
	case string(nosqlsdk.TableLifecycleStateDeleting),
		string(nosqlsdk.TableLifecycleStateDeleted):
		return shared.OSOKAsyncPhaseDelete
	default:
		return shared.OSOKAsyncPhaseCreate
	}
}

func currentTableAsyncPhase(resource *nosqlv1beta1.Table, fallback shared.OSOKAsyncPhase) shared.OSOKAsyncPhase {
	if resource == nil {
		return fallback
	}
	if resource.Status.OsokStatus.Async.Current != nil && resource.Status.OsokStatus.Async.Current.Phase != "" {
		return resource.Status.OsokStatus.Async.Current.Phase
	}
	return fallback
}

func specTableLimits(spec nosqlv1beta1.TableLimits) *nosqlsdk.TableLimits {
	if !hasMeaningfulTableLimits(spec) {
		return nil
	}

	limits := &nosqlsdk.TableLimits{
		MaxReadUnits:    common.Int(spec.MaxReadUnits),
		MaxWriteUnits:   common.Int(spec.MaxWriteUnits),
		MaxStorageInGBs: common.Int(spec.MaxStorageInGBs),
	}
	if strings.TrimSpace(spec.CapacityMode) != "" {
		limits.CapacityMode = nosqlsdk.TableLimitsCapacityModeEnum(spec.CapacityMode)
	}
	return limits
}

func hasMeaningfulTableLimits(spec nosqlv1beta1.TableLimits) bool {
	return spec.MaxReadUnits != 0 ||
		spec.MaxWriteUnits != 0 ||
		spec.MaxStorageInGBs != 0 ||
		strings.TrimSpace(spec.CapacityMode) != ""
}

func sdkTableLimitsEqual(desired *nosqlsdk.TableLimits, existing *nosqlsdk.TableLimits) bool {
	if desired == nil {
		return true
	}
	if existing == nil {
		return false
	}
	return intPointerValue(desired.MaxReadUnits) == intPointerValue(existing.MaxReadUnits) &&
		intPointerValue(desired.MaxWriteUnits) == intPointerValue(existing.MaxWriteUnits) &&
		intPointerValue(desired.MaxStorageInGBs) == intPointerValue(existing.MaxStorageInGBs) &&
		strings.EqualFold(string(desired.CapacityMode), string(existing.CapacityMode))
}

func isAlterTableDDL(statement string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(statement)), "ALTER TABLE")
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolPointerValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func intPointerValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}

	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.GetHTTPStatusCode() == 404 {
			return true
		}
		switch serviceErr.GetCode() {
		case "NotFound", "NotAuthorizedOrNotFound":
			return true
		}
	}

	message := err.Error()
	return strings.Contains(message, "http status code: 404") ||
		strings.Contains(message, "NotFound") ||
		strings.Contains(message, "NotAuthorizedOrNotFound")
}
