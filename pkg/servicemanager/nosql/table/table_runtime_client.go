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

type tableIdentity struct {
	compartmentID string
	name          string
}

type tableRuntimeClient struct {
	log      loggerutil.OSOKLogger
	delegate TableServiceClient
	client   tableOCIClient
	initErr  error
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

var _ TableServiceClient = (*tableRuntimeClient)(nil)

func init() {
	registerTableRuntimeHooksMutator(func(manager *TableServiceManager, hooks *TableRuntimeHooks) {
		client, err := newTableSDKClient(manager)
		applyTableRuntimeHooks(manager, hooks, client, err)
	})
}

func newTableSDKClient(manager *TableServiceManager) (tableOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Table service manager is nil")
	}

	client, err := nosqlsdk.NewNosqlClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTableRuntimeHooks(
	manager *TableServiceManager,
	hooks *TableRuntimeHooks,
	client tableOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}
	hooks.Semantics = nil

	runtimeClient := newTableRuntimeClient(manager, nil, client, initErr)

	hooks.BuildCreateBody = func(_ context.Context, resource *nosqlv1beta1.Table, _ string) (any, error) {
		return buildCreateTableDetails(resource)
	}
	hooks.BuildUpdateBody = runtimeClient.buildGeneratedUpdateBody
	hooks.Identity.Resolve = resolveTableIdentity
	hooks.Identity.LookupExisting = func(ctx context.Context, _ *nosqlv1beta1.Table, identity any) (any, error) {
		return runtimeClient.lookupExisting(ctx, identity)
	}
	hooks.Get.Fields = tableGetFields()
	hooks.Get.Call = wrapTableReadOperationCall(hooks.Get.Call)
	hooks.List.Fields = tableListFields()
	hooks.List.Call = wrapTableListOperationCall(hooks.List.Call)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedTableIdentity
	hooks.StatusHooks.ProjectStatus = runtimeClient.projectStatusFromResponse
	hooks.StatusHooks.ApplyLifecycle = runtimeClient.applyLifecycleFromResponse
	hooks.ParityHooks.RequiresParityHandling = func(_ *nosqlv1beta1.Table, currentResponse any) bool {
		return currentResponse != nil
	}
	hooks.ParityHooks.ApplyParityUpdate = runtimeClient.applyParityUpdate
	hooks.DeleteHooks.ApplyOutcome = runtimeClient.applyDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TableServiceClient) TableServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func newTableRuntimeClient(
	manager *TableServiceManager,
	delegate TableServiceClient,
	client tableOCIClient,
	initErr error,
) *tableRuntimeClient {
	runtimeClient := &tableRuntimeClient{
		delegate: delegate,
		client:   client,
		initErr:  initErr,
	}
	if manager != nil {
		runtimeClient.log = manager.Log
	}
	return runtimeClient
}

func newTableServiceClientWithOCIClient(log loggerutil.OSOKLogger, client tableOCIClient) TableServiceClient {
	manager := &TableServiceManager{Log: log}
	hooks := newTableRuntimeHooksWithOCIClient(client)
	applyTableRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultTableServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*nosqlv1beta1.Table](
			buildTableGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapTableGeneratedClient(hooks, delegate)
}

func newTableRuntimeHooksWithOCIClient(client tableOCIClient) TableRuntimeHooks {
	return TableRuntimeHooks{
		Create: runtimeOperationHooks[nosqlsdk.CreateTableRequest, nosqlsdk.CreateTableResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateTableDetails", RequestName: "CreateTableDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request nosqlsdk.CreateTableRequest) (nosqlsdk.CreateTableResponse, error) {
				return client.CreateTable(ctx, request)
			},
		},
		Get: runtimeOperationHooks[nosqlsdk.GetTableRequest, nosqlsdk.GetTableResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "TableNameOrId", RequestName: "tableNameOrId", Contribution: "path", PreferResourceID: true},
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
			},
			Call: func(ctx context.Context, request nosqlsdk.GetTableRequest) (nosqlsdk.GetTableResponse, error) {
				return client.GetTable(ctx, request)
			},
		},
		List: runtimeOperationHooks[nosqlsdk.ListTablesRequest, nosqlsdk.ListTablesResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
			},
			Call: func(ctx context.Context, request nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error) {
				return client.ListTables(ctx, request)
			},
		},
		Update: runtimeOperationHooks[nosqlsdk.UpdateTableRequest, nosqlsdk.UpdateTableResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "TableNameOrId", RequestName: "tableNameOrId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateTableDetails", RequestName: "UpdateTableDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request nosqlsdk.UpdateTableRequest) (nosqlsdk.UpdateTableResponse, error) {
				return client.UpdateTable(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[nosqlsdk.DeleteTableRequest, nosqlsdk.DeleteTableResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "TableNameOrId", RequestName: "tableNameOrId", Contribution: "path", PreferResourceID: true},
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "IsIfExists", RequestName: "isIfExists", Contribution: "query"},
			},
			Call: func(ctx context.Context, request nosqlsdk.DeleteTableRequest) (nosqlsdk.DeleteTableResponse, error) {
				return client.DeleteTable(ctx, request)
			},
		},
	}
}

func tableGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "TableNameOrId",
			RequestName:      "tableNameOrId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.ocid", "spec.name"},
		},
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "status.compartmentId"},
		},
	}
}

func tableListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "status.compartmentId"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths:  []string{"spec.name", "status.name"},
		},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func wrapTableReadOperationCall[Req any, Resp any](
	call func(context.Context, Req) (Resp, error),
) func(context.Context, Req) (Resp, error) {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request Req) (Resp, error) {
		response, err := call(ctx, request)
		if err != nil && isNotFoundErr(err) {
			var zero Resp
			return zero, errorutil.NotFoundOciError{
				HTTPStatusCode: 404,
				ErrorCode:      errorutil.NotFound,
				Description:    "Table resource not found",
			}
		}
		return response, err
	}
}

func wrapTableListOperationCall(
	call func(context.Context, nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error),
) func(context.Context, nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error) {
	call = wrapTableReadOperationCall(call)
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request nosqlsdk.ListTablesRequest) (nosqlsdk.ListTablesResponse, error) {
		if request.LifecycleState == "" {
			request.LifecycleState = nosqlsdk.ListTablesLifecycleStateAll
		}
		return call(ctx, request)
	}
}

func (c *tableRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return c.fail(resource, fmt.Errorf("Table generated delegate is not configured"))
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || resource == nil {
		return response, err
	}
	if !response.IsSuccessful || !response.ShouldRequeue || resource.Status.OsokStatus.Async.Current != nil {
		return response, nil
	}
	if currentTableID(resource) != "" {
		return response, nil
	}
	if lastTableCondition(resource) != shared.Provisioning {
		return response, nil
	}

	return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseCreate, "OCI table create request accepted", ""), nil
}

func (c *tableRuntimeClient) Delete(ctx context.Context, resource *nosqlv1beta1.Table) (bool, error) {
	if err := c.ensureClient(); err != nil {
		return false, err
	}

	target, err := c.resolveDeleteTarget(ctx, resource)
	if err != nil {
		c.recordErrorRequestID(resource, err)
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
	if err := c.deleteTable(ctx, resource, target.id); err != nil {
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

func (c *tableRuntimeClient) buildGeneratedUpdateBody(
	_ context.Context,
	resource *nosqlv1beta1.Table,
	_ string,
	currentResponse any,
) (any, bool, error) {
	current, err := tableSnapshotFromResponse(currentResponse)
	if err != nil {
		return nil, false, err
	}
	if current == nil {
		return nil, false, nil
	}
	if err := validateTableForceNewFields(resource, current); err != nil {
		return nil, false, err
	}

	details, _ := buildUpdateDetails(resource, current)
	return details, true, nil
}

func (c *tableRuntimeClient) lookupExisting(ctx context.Context, identity any) (any, error) {
	if err := c.ensureClient(); err != nil {
		return nil, err
	}

	tableIdentity, ok := identity.(tableIdentity)
	if !ok {
		return nil, fmt.Errorf("resolve Table identity: unexpected type %T", identity)
	}

	summary, err := c.findTableByName(ctx, tableIdentity.compartmentID, tableIdentity.name)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}
	if summary.Id == nil || strings.TrimSpace(*summary.Id) == "" {
		return *summary, nil
	}

	table, err := c.getTableByID(ctx, *summary.Id)
	if err == nil {
		return table, nil
	}
	if errors.Is(err, errTableNotFound) {
		return *summary, nil
	}
	return nil, fmt.Errorf("get Table %q: %w", *summary.Id, err)
}

func (c *tableRuntimeClient) projectStatusFromResponse(resource *nosqlv1beta1.Table, response any) error {
	snapshot, err := tableSnapshotFromResponse(response)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil
	}
	return c.projectPayload(resource, snapshot.payload)
}

func (c *tableRuntimeClient) applyLifecycleFromResponse(
	resource *nosqlv1beta1.Table,
	response any,
) (servicemanager.OSOKResponse, error) {
	snapshot, err := tableSnapshotFromResponse(response)
	if err != nil {
		return c.fail(resource, err)
	}
	if snapshot == nil {
		return c.markCondition(resource, shared.Active, "OCI table is active", false), nil
	}

	fallbackPhase := currentTableAsyncPhase(resource, tableLifecyclePhase(normalizeLifecycle(snapshot.lifecycleState)))
	return c.finishWithLifecycle(resource, snapshot, fallbackPhase), nil
}

func (c *tableRuntimeClient) applyParityUpdate(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	currentResponse any,
) (servicemanager.OSOKResponse, error) {
	if err := c.ensureClient(); err != nil {
		return c.fail(resource, err)
	}

	current, err := tableSnapshotFromResponse(currentResponse)
	if err != nil {
		return c.fail(resource, err)
	}
	if current == nil {
		return c.fail(resource, fmt.Errorf("current Table response is empty"))
	}
	if err := c.projectPayload(resource, current.payload); err != nil {
		return c.fail(resource, err)
	}

	switch normalizeLifecycle(current.lifecycleState) {
	case string(nosqlsdk.TableLifecycleStateDeleted),
		string(nosqlsdk.TableLifecycleStateFailed),
		string(nosqlsdk.TableLifecycleStateCreating),
		string(nosqlsdk.TableLifecycleStateUpdating),
		string(nosqlsdk.TableLifecycleStateDeleting):
		return c.finishWithLifecycle(resource, current, currentTableAsyncPhase(resource, tableLifecyclePhase(normalizeLifecycle(current.lifecycleState)))), nil
	}

	if err := validateTableForceNewFields(resource, current); err != nil {
		return c.fail(resource, err)
	}
	if strings.TrimSpace(current.id) == "" {
		return c.fail(resource, fmt.Errorf("resolve Table update target: missing table OCID"))
	}

	if current.compartmentID != "" &&
		strings.TrimSpace(resource.Spec.CompartmentId) != "" &&
		current.compartmentID != resource.Spec.CompartmentId {
		if err := c.changeTableCompartment(ctx, resource, current.id, current.compartmentID, resource.Spec.CompartmentId); err != nil {
			return c.fail(resource, err)
		}

		refreshed, err := c.readTable(ctx, resource, current.id)
		if err != nil {
			if errors.Is(err, errTableNotFound) {
				return c.markPendingLifecycleOperation(resource, shared.OSOKAsyncPhaseUpdate, "OCI table compartment move request accepted", ""), nil
			}
			return c.fail(resource, fmt.Errorf("confirm Table compartment change: %w", err))
		}
		if err := c.projectPayload(resource, refreshed.payload); err != nil {
			return c.fail(resource, err)
		}

		current = refreshed
		if normalizeLifecycle(current.lifecycleState) != string(nosqlsdk.TableLifecycleStateActive) ||
			current.compartmentID != resource.Spec.CompartmentId {
			return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseUpdate), nil
		}
	}

	updateDetails, updateNeeded := buildUpdateDetails(resource, current)
	if !updateNeeded {
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseUpdate), nil
	}

	if err := c.updateTable(ctx, resource, current.id, updateDetails); err != nil {
		return c.fail(resource, err)
	}

	refreshed, err := c.readTable(ctx, resource, current.id)
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

func (c *tableRuntimeClient) applyDeleteOutcome(
	resource *nosqlv1beta1.Table,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	snapshot, err := tableSnapshotFromResponse(response)
	if err != nil {
		return generatedruntime.DeleteOutcome{}, err
	}
	if snapshot == nil {
		c.markDeleted(resource, "OCI table deleted")
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}

	if normalizeLifecycle(snapshot.lifecycleState) == string(nosqlsdk.TableLifecycleStateDeleted) {
		c.markDeleted(resource, "OCI table deleted")
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}

	result := c.markPendingLifecycleOperation(
		resource,
		shared.OSOKAsyncPhaseDelete,
		"OCI table delete request accepted",
		snapshot.lifecycleState,
	)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: !result.ShouldRequeue}, nil
}

func (c *tableRuntimeClient) ensureClient() error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.client == nil {
		return fmt.Errorf("Table OCI client is not configured")
	}
	return nil
}

func (c *tableRuntimeClient) readTable(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	preferredID string,
) (*tableSnapshot, error) {
	if preferredID != "" {
		table, err := c.getTableByID(ctx, preferredID)
		if err == nil {
			return snapshotFromTable(table), nil
		}
		if !errors.Is(err, errTableNotFound) {
			return nil, err
		}
	}

	if resource != nil && strings.TrimSpace(resource.Spec.Name) != "" && strings.TrimSpace(resource.Spec.CompartmentId) != "" {
		table, err := c.getTableByName(ctx, resource.Spec.CompartmentId, resource.Spec.Name)
		if err == nil {
			return snapshotFromTable(table), nil
		}
		if !errors.Is(err, errTableNotFound) {
			return nil, err
		}
	}

	compartmentID := ""
	name := ""
	if resource != nil {
		compartmentID = resource.Spec.CompartmentId
		name = resource.Spec.Name
	}
	summary, err := c.findTableByName(ctx, compartmentID, name)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, errTableNotFound
	}
	return snapshotFromSummary(*summary), nil
}

func (c *tableRuntimeClient) resolveDeleteTarget(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
) (*tableSnapshot, error) {
	if currentID := currentTableID(resource); currentID != "" {
		table, err := c.getTableByID(ctx, currentID)
		if err == nil {
			return snapshotFromTable(table), nil
		}
		if errors.Is(err, errTableNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get Table %q: %w", currentID, err)
	}

	compartmentID := ""
	name := ""
	if resource != nil {
		compartmentID = resource.Spec.CompartmentId
		name = resource.Spec.Name
	}
	summary, err := c.findTableByName(ctx, compartmentID, name)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}
	return snapshotFromSummary(*summary), nil
}

func (c *tableRuntimeClient) getTableByID(ctx context.Context, id string) (nosqlsdk.Table, error) {
	response, err := c.client.GetTable(ctx, nosqlsdk.GetTableRequest{
		TableNameOrId: common.String(id),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nosqlsdk.Table{}, errTableNotFound
		}
		return nosqlsdk.Table{}, err
	}
	return response.Table, nil
}

func (c *tableRuntimeClient) getTableByName(
	ctx context.Context,
	compartmentID string,
	name string,
) (nosqlsdk.Table, error) {
	response, err := c.client.GetTable(ctx, nosqlsdk.GetTableRequest{
		TableNameOrId: common.String(name),
		CompartmentId: common.String(compartmentID),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nosqlsdk.Table{}, errTableNotFound
		}
		return nosqlsdk.Table{}, err
	}
	return response.Table, nil
}

func (c *tableRuntimeClient) findTableByName(
	ctx context.Context,
	compartmentID string,
	name string,
) (*nosqlsdk.TableSummary, error) {
	if strings.TrimSpace(compartmentID) == "" || strings.TrimSpace(name) == "" {
		return nil, nil
	}

	response, err := c.client.ListTables(ctx, nosqlsdk.ListTablesRequest{
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

func (c *tableRuntimeClient) changeTableCompartment(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	tableID string,
	fromCompartmentID string,
	toCompartmentID string,
) error {
	response, err := c.client.ChangeTableCompartment(ctx, nosqlsdk.ChangeTableCompartmentRequest{
		TableNameOrId: common.String(tableID),
		ChangeTableCompartmentDetails: nosqlsdk.ChangeTableCompartmentDetails{
			ToCompartmentId:   common.String(toCompartmentID),
			FromCompartmentId: optionalString(fromCompartmentID),
		},
	})
	if err != nil {
		c.recordErrorRequestID(resource, err)
		if isNotFoundErr(err) {
			return errTableNotFound
		}
		return fmt.Errorf("change Table compartment: %w", err)
	}
	c.recordResponseRequestID(resource, response)
	return nil
}

func (c *tableRuntimeClient) updateTable(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	tableID string,
	details nosqlsdk.UpdateTableDetails,
) error {
	response, err := c.client.UpdateTable(ctx, nosqlsdk.UpdateTableRequest{
		TableNameOrId:      common.String(tableID),
		UpdateTableDetails: details,
	})
	if err != nil {
		c.recordErrorRequestID(resource, err)
		if isNotFoundErr(err) {
			return errTableNotFound
		}
		return fmt.Errorf("update Table: %w", err)
	}
	c.recordResponseRequestID(resource, response)
	return nil
}

func (c *tableRuntimeClient) deleteTable(
	ctx context.Context,
	resource *nosqlv1beta1.Table,
	tableID string,
) error {
	response, err := c.client.DeleteTable(ctx, nosqlsdk.DeleteTableRequest{
		TableNameOrId: common.String(tableID),
		IsIfExists:    common.Bool(true),
	})
	if err != nil {
		c.recordErrorRequestID(resource, err)
		if isNotFoundErr(err) {
			return errTableNotFound
		}
		return fmt.Errorf("delete Table: %w", err)
	}
	c.recordResponseRequestID(resource, response)
	return nil
}

func resolveTableIdentity(resource *nosqlv1beta1.Table) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("resolve Table identity: resource is nil")
	}

	identity := tableIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		name:          strings.TrimSpace(resource.Spec.Name),
	}
	if identity.compartmentID == "" {
		return nil, fmt.Errorf("resolve Table identity: compartmentId is empty")
	}
	if identity.name == "" {
		return nil, fmt.Errorf("resolve Table identity: name is empty")
	}
	return identity, nil
}

func clearTrackedTableIdentity(resource *nosqlv1beta1.Table) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}

func tableSnapshotFromResponse(response any) (*tableSnapshot, error) {
	switch typed := response.(type) {
	case nil:
		return nil, nil
	case nosqlsdk.Table:
		return snapshotFromTable(typed), nil
	case *nosqlsdk.Table:
		if typed == nil {
			return nil, nil
		}
		return snapshotFromTable(*typed), nil
	case nosqlsdk.TableSummary:
		return snapshotFromSummary(typed), nil
	case *nosqlsdk.TableSummary:
		if typed == nil {
			return nil, nil
		}
		return snapshotFromSummary(*typed), nil
	case nosqlsdk.GetTableResponse:
		return snapshotFromTable(typed.Table), nil
	case *nosqlsdk.GetTableResponse:
		if typed == nil {
			return nil, nil
		}
		return snapshotFromTable(typed.Table), nil
	default:
		return nil, fmt.Errorf("unexpected Table response type %T", response)
	}
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

func buildCreateTableDetails(resource *nosqlv1beta1.Table) (nosqlsdk.CreateTableDetails, error) {
	if resource == nil {
		return nosqlsdk.CreateTableDetails{}, fmt.Errorf("Table resource is nil")
	}

	details := nosqlsdk.CreateTableDetails{
		Name:              common.String(resource.Spec.Name),
		CompartmentId:     common.String(resource.Spec.CompartmentId),
		DdlStatement:      common.String(resource.Spec.DdlStatement),
		IsAutoReclaimable: common.Bool(resource.Spec.IsAutoReclaimable),
		FreeformTags:      resource.Spec.FreeformTags,
	}
	if limits := specTableLimits(resource.Spec.TableLimits); limits != nil {
		details.TableLimits = limits
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildUpdateDetails(resource *nosqlv1beta1.Table, existing *tableSnapshot) (nosqlsdk.UpdateTableDetails, bool) {
	var (
		details     nosqlsdk.UpdateTableDetails
		needsUpdate bool
	)

	if resource == nil || existing == nil {
		return details, false
	}
	if isAlterTableDDL(resource.Spec.DdlStatement) {
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

func validateTableForceNewFields(resource *nosqlv1beta1.Table, existing *tableSnapshot) error {
	if resource == nil || existing == nil {
		return nil
	}
	if strings.TrimSpace(existing.name) != "" && strings.TrimSpace(resource.Spec.Name) != "" && existing.name != resource.Spec.Name {
		return fmt.Errorf("Table formal semantics require replacement when name changes")
	}
	if existing.isAutoReclaimable != resource.Spec.IsAutoReclaimable {
		return fmt.Errorf("Table formal semantics require replacement when isAutoReclaimable changes")
	}
	return nil
}

func (c *tableRuntimeClient) projectPayload(resource *nosqlv1beta1.Table, payload any) error {
	if resource == nil || payload == nil {
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

func (c *tableRuntimeClient) finishWithLifecycle(
	resource *nosqlv1beta1.Table,
	snapshot *tableSnapshot,
	fallbackPhase shared.OSOKAsyncPhase,
) servicemanager.OSOKResponse {
	if snapshot == nil {
		return c.markCondition(resource, shared.Active, "OCI table is active", false)
	}

	message := lifecycleMessage(snapshot.lifecycleDetails, snapshot.name, "OCI table is active")
	current := servicemanager.NewLifecycleAsyncOperation(&resource.Status.OsokStatus, snapshot.lifecycleState, message, fallbackPhase)
	if current != nil {
		return c.markAsyncOperation(resource, current)
	}
	return c.markCondition(resource, shared.Active, message, false)
}

func (c *tableRuntimeClient) markCondition(
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

func (c *tableRuntimeClient) markAsyncOperation(resource *nosqlv1beta1.Table, current *shared.OSOKAsyncOperation) servicemanager.OSOKResponse {
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

func (c *tableRuntimeClient) markPendingLifecycleOperation(
	resource *nosqlv1beta1.Table,
	phase shared.OSOKAsyncPhase,
	message string,
	rawStatus string,
) servicemanager.OSOKResponse {
	return c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       normalizeLifecycle(rawStatus),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *tableRuntimeClient) markDeleted(resource *nosqlv1beta1.Table, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *tableRuntimeClient) fail(
	resource *nosqlv1beta1.Table,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil {
		status := &resource.Status.OsokStatus
		servicemanager.RecordErrorOpcRequestID(status, err)
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

func (c *tableRuntimeClient) recordResponseRequestID(resource *nosqlv1beta1.Table, response any) {
	if resource == nil {
		return
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
}

func (c *tableRuntimeClient) recordErrorRequestID(resource *nosqlv1beta1.Table, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
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

func lastTableCondition(resource *nosqlv1beta1.Table) shared.OSOKConditionType {
	if resource == nil {
		return ""
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return ""
	}
	return conditions[len(conditions)-1].Type
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
