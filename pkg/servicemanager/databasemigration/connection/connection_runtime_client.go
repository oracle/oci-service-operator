/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var connectionWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(databasemigrationsdk.OperationStatusAccepted),
		string(databasemigrationsdk.OperationStatusInProgress),
		string(databasemigrationsdk.OperationStatusWaiting),
		string(databasemigrationsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(databasemigrationsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(databasemigrationsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(databasemigrationsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(databasemigrationsdk.OperationTypesCreateConnection)},
	UpdateActionTokens:    []string{string(databasemigrationsdk.OperationTypesUpdateConnection)},
	DeleteActionTokens:    []string{string(databasemigrationsdk.OperationTypesDeleteConnection)},
}

type connectionOCIClient interface {
	CreateConnection(context.Context, databasemigrationsdk.CreateConnectionRequest) (databasemigrationsdk.CreateConnectionResponse, error)
	GetConnection(context.Context, databasemigrationsdk.GetConnectionRequest) (databasemigrationsdk.GetConnectionResponse, error)
	ListConnections(context.Context, databasemigrationsdk.ListConnectionsRequest) (databasemigrationsdk.ListConnectionsResponse, error)
	UpdateConnection(context.Context, databasemigrationsdk.UpdateConnectionRequest) (databasemigrationsdk.UpdateConnectionResponse, error)
	DeleteConnection(context.Context, databasemigrationsdk.DeleteConnectionRequest) (databasemigrationsdk.DeleteConnectionResponse, error)
	GetWorkRequest(context.Context, databasemigrationsdk.GetWorkRequestRequest) (databasemigrationsdk.GetWorkRequestResponse, error)
}

func init() {
	registerConnectionRuntimeHooksMutator(func(manager *ConnectionServiceManager, hooks *ConnectionRuntimeHooks) {
		client, initErr := newConnectionSDKClient(manager)
		applyConnectionRuntimeHooks(hooks, client, initErr)
	})
}

func newConnectionSDKClient(manager *ConnectionServiceManager) (connectionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Connection service manager is nil")
	}

	client, err := databasemigrationsdk.NewDatabaseMigrationClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyConnectionRuntimeHooks(
	hooks *ConnectionRuntimeHooks,
	client connectionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedConnectionRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardConnectionExistingBeforeCreate
	hooks.Create.Fields = connectionCreateFields()
	hooks.Get.Fields = connectionGetFields()
	hooks.List.Fields = connectionListFields()
	hooks.Update.Fields = connectionUpdateFields()
	hooks.Delete.Fields = connectionDeleteFields()
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *databasemigrationv1beta1.Connection,
		namespace string,
	) (any, error) {
		return buildConnectionCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *databasemigrationv1beta1.Connection,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildConnectionUpdateDetails(ctx, resource, namespace, currentResponse)
	}
	hooks.Async.Adapter = connectionWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getConnectionWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveConnectionGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveConnectionGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverConnectionIDFromGeneratedWorkRequest
	hooks.Async.Message = connectionGeneratedWorkRequestMessage
}

func newConnectionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client connectionOCIClient,
) ConnectionServiceClient {
	return defaultConnectionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*databasemigrationv1beta1.Connection](
			newConnectionRuntimeConfig(log, client),
		),
	}
}

func newConnectionRuntimeConfig(
	log loggerutil.OSOKLogger,
	client connectionOCIClient,
) generatedruntime.Config[*databasemigrationv1beta1.Connection] {
	hooks := newConnectionRuntimeHooksWithOCIClient(client)
	applyConnectionRuntimeHooks(&hooks, client, nil)
	return buildConnectionGeneratedRuntimeConfig(&ConnectionServiceManager{Log: log}, hooks)
}

func newConnectionRuntimeHooksWithOCIClient(client connectionOCIClient) ConnectionRuntimeHooks {
	return ConnectionRuntimeHooks{
		Semantics: reviewedConnectionRuntimeSemantics(),
		Create: runtimeOperationHooks[databasemigrationsdk.CreateConnectionRequest, databasemigrationsdk.CreateConnectionResponse]{
			Fields: connectionCreateFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.CreateConnectionRequest) (databasemigrationsdk.CreateConnectionResponse, error) {
				return client.CreateConnection(ctx, request)
			},
		},
		Get: runtimeOperationHooks[databasemigrationsdk.GetConnectionRequest, databasemigrationsdk.GetConnectionResponse]{
			Fields: connectionGetFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.GetConnectionRequest) (databasemigrationsdk.GetConnectionResponse, error) {
				return client.GetConnection(ctx, request)
			},
		},
		List: runtimeOperationHooks[databasemigrationsdk.ListConnectionsRequest, databasemigrationsdk.ListConnectionsResponse]{
			Fields: connectionListFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.ListConnectionsRequest) (databasemigrationsdk.ListConnectionsResponse, error) {
				return client.ListConnections(ctx, request)
			},
		},
		Update: runtimeOperationHooks[databasemigrationsdk.UpdateConnectionRequest, databasemigrationsdk.UpdateConnectionResponse]{
			Fields: connectionUpdateFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.UpdateConnectionRequest) (databasemigrationsdk.UpdateConnectionResponse, error) {
				return client.UpdateConnection(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[databasemigrationsdk.DeleteConnectionRequest, databasemigrationsdk.DeleteConnectionResponse]{
			Fields: connectionDeleteFields(),
			Call: func(ctx context.Context, request databasemigrationsdk.DeleteConnectionRequest) (databasemigrationsdk.DeleteConnectionResponse, error) {
				return client.DeleteConnection(ctx, request)
			},
		},
	}
}

func reviewedConnectionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "databasemigration",
		FormalSlug:    "connection",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(databasemigrationsdk.ConnectionLifecycleStateCreating)},
			UpdatingStates:     []string{string(databasemigrationsdk.ConnectionLifecycleStateUpdating)},
			ActiveStates: []string{
				string(databasemigrationsdk.ConnectionLifecycleStateActive),
				string(databasemigrationsdk.ConnectionLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(databasemigrationsdk.ConnectionLifecycleStateDeleting)},
			TerminalStates: []string{string(databasemigrationsdk.ConnectionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"connectionType",
				"technologyType",
				"databaseId",
				"databaseName",
				"dbSystemId",
				"host",
				"port",
				"connectionString",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"additionalAttributes",
				"connectionString",
				"databaseId",
				"databaseName",
				"dbSystemId",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"host",
				"keyId",
				"nsgIds",
				"password",
				"port",
				"replicationPassword",
				"replicationUsername",
				"securityProtocol",
				"sshHost",
				"sshKey",
				"sshSudoLocation",
				"sshUser",
				"sslCa",
				"sslCert",
				"sslCrl",
				"sslKey",
				"sslMode",
				"subnetId",
				"username",
				"vaultId",
				"wallet",
			},
			ForceNew:      []string{"compartmentId", "connectionType", "technologyType"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "connection", Action: "CREATED"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "connection", Action: "UPDATED"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "connection", Action: "DELETED"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetConnection",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "connection", Action: "CREATED"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetConnection",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "connection", Action: "UPDATED"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetConnection/ListConnections confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "connection", Action: "DELETED"}},
		},
		AuxiliaryOperations: nil,
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func connectionCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateConnectionDetails", RequestName: "CreateConnectionDetails", Contribution: "body"},
	}
}

func connectionGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ConnectionId", RequestName: "connectionId", Contribution: "path", PreferResourceID: true},
	}
}

func connectionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
	}
}

func connectionUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ConnectionId", RequestName: "connectionId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateConnectionDetails", RequestName: "UpdateConnectionDetails", Contribution: "body"},
	}
}

func connectionDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ConnectionId", RequestName: "connectionId", Contribution: "path", PreferResourceID: true},
	}
}

func guardConnectionExistingBeforeCreate(
	_ context.Context,
	resource *databasemigrationv1beta1.Connection,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if strings.TrimSpace(resource.Spec.ConnectionType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if strings.TrimSpace(resource.Spec.TechnologyType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildConnectionCreateDetails(
	ctx context.Context,
	resource *databasemigrationv1beta1.Connection,
	namespace string,
) (databasemigrationsdk.CreateConnectionDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("Connection resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return nil, err
	}

	connectionType, err := connectionTypeForResource(resource, nil)
	if err != nil {
		return nil, err
	}

	specValues, err := connectionJSONMap(resolvedSpec)
	if err != nil {
		return nil, fmt.Errorf("project Connection create spec: %w", err)
	}
	if err := validateConnectionRequiredFields(connectionType, specValues); err != nil {
		return nil, err
	}
	if err := validateConnectionTypeSpecificFields(connectionType, specValues); err != nil {
		return nil, err
	}

	return decodeConnectionCreateDetails(connectionType, resolvedSpec)
}

func buildConnectionUpdateDetails(
	ctx context.Context,
	resource *databasemigrationv1beta1.Connection,
	namespace string,
	currentResponse any,
) (databasemigrationsdk.UpdateConnectionDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("Connection resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return nil, false, err
	}

	connectionType, err := connectionTypeForResource(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}

	specValues, err := connectionJSONMap(resolvedSpec)
	if err != nil {
		return nil, false, fmt.Errorf("project Connection update spec: %w", err)
	}
	if err := validateConnectionTypeSpecificFields(connectionType, specValues); err != nil {
		return nil, false, err
	}

	desired, err := decodeConnectionUpdateDetails(connectionType, resolvedSpec)
	if err != nil {
		return nil, false, err
	}
	desiredValues, err := connectionJSONMap(desired)
	if err != nil {
		return nil, false, fmt.Errorf("project desired Connection update body: %w", err)
	}
	if len(desiredValues) == 0 {
		return desired, false, nil
	}

	currentDetails, err := connectionCurrentUpdateDetails(connectionType, currentResponse)
	if err != nil {
		return nil, false, err
	}
	currentValues, err := connectionJSONMap(currentDetails)
	if err != nil {
		return nil, false, fmt.Errorf("project current Connection update body: %w", err)
	}

	return desired, !connectionMapSubsetEqual(desiredValues, currentValues), nil
}

func connectionCurrentUpdateDetails(
	connectionType string,
	currentResponse any,
) (databasemigrationsdk.UpdateConnectionDetails, error) {
	body, err := connectionRuntimeBody(currentResponse)
	if err != nil {
		return nil, err
	}
	return decodeConnectionUpdateDetails(connectionType, body)
}

func connectionTypeForResource(
	resource *databasemigrationv1beta1.Connection,
	currentResponse any,
) (string, error) {
	if resource != nil {
		if connectionType, ok := normalizeConnectionType(resource.Spec.ConnectionType); ok {
			return connectionType, nil
		}
	}

	if currentResponse != nil {
		body, err := connectionRuntimeBody(currentResponse)
		if err != nil {
			return "", err
		}
		if connectionType, ok := normalizeConnectionType(connectionTypeFromRuntimeBody(body)); ok {
			return connectionType, nil
		}
	}

	return "", fmt.Errorf("Connection spec.connectionType must be set to MYSQL or ORACLE")
}

func normalizeConnectionType(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	connectionType, ok := databasemigrationsdk.GetMappingConnectionTypeEnum(raw)
	if !ok {
		return "", false
	}
	return string(connectionType), true
}

func decodeConnectionCreateDetails(
	connectionType string,
	raw any,
) (databasemigrationsdk.CreateConnectionDetails, error) {
	switch connectionType {
	case "MYSQL":
		details, err := decodeConnectionConcrete[databasemigrationsdk.CreateMysqlConnectionDetails](raw)
		return details, err
	case "ORACLE":
		details, err := decodeConnectionConcrete[databasemigrationsdk.CreateOracleConnectionDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported Connection create type %q", connectionType)
	}
}

func decodeConnectionUpdateDetails(
	connectionType string,
	raw any,
) (databasemigrationsdk.UpdateConnectionDetails, error) {
	switch connectionType {
	case "MYSQL":
		details, err := decodeConnectionConcrete[databasemigrationsdk.UpdateMysqlConnectionDetails](raw)
		return details, err
	case "ORACLE":
		details, err := decodeConnectionConcrete[databasemigrationsdk.UpdateOracleConnectionDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported Connection update type %q", connectionType)
	}
}

func decodeConnectionConcrete[T any](raw any) (T, error) {
	var decoded T

	payload, err := json.Marshal(raw)
	if err != nil {
		return decoded, fmt.Errorf("marshal Connection payload: %w", err)
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return decoded, fmt.Errorf("unmarshal Connection payload: %w", err)
	}
	return decoded, nil
}

func connectionRuntimeBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case databasemigrationsdk.CreateConnectionResponse:
		return current.Connection, nil
	case *databasemigrationsdk.CreateConnectionResponse:
		if current == nil {
			return nil, fmt.Errorf("current Connection response is nil")
		}
		return current.Connection, nil
	case databasemigrationsdk.GetConnectionResponse:
		return current.Connection, nil
	case *databasemigrationsdk.GetConnectionResponse:
		if current == nil {
			return nil, fmt.Errorf("current Connection response is nil")
		}
		return current.Connection, nil
	case databasemigrationsdk.Connection:
		if current == nil {
			return nil, fmt.Errorf("current Connection body is nil")
		}
		return current, nil
	case databasemigrationsdk.ConnectionSummary:
		if current == nil {
			return nil, fmt.Errorf("current Connection summary is nil")
		}
		return current, nil
	case databasemigrationsdk.MysqlConnection,
		databasemigrationsdk.OracleConnection,
		databasemigrationsdk.MysqlConnectionSummary,
		databasemigrationsdk.OracleConnectionSummary:
		return current, nil
	default:
		return nil, fmt.Errorf("unsupported current Connection payload type %T", currentResponse)
	}
}

func connectionTypeFromRuntimeBody(body any) string {
	switch current := body.(type) {
	case databasemigrationsdk.MysqlConnection, databasemigrationsdk.MysqlConnectionSummary:
		return "MYSQL"
	case databasemigrationsdk.OracleConnection, databasemigrationsdk.OracleConnectionSummary:
		return "ORACLE"
	case databasemigrationsdk.Connection, databasemigrationsdk.ConnectionSummary:
		return connectionTypeFromRuntimeBodyValue(current)
	default:
		return ""
	}
}

func connectionTypeFromRuntimeBodyValue(body any) string {
	values, err := connectionJSONMap(body)
	if err != nil {
		return ""
	}
	rawType, ok := values["connectionType"].(string)
	if !ok {
		return ""
	}
	return rawType
}

func validateConnectionRequiredFields(connectionType string, specValues map[string]any) error {
	required := []string{"displayName", "compartmentId", "vaultId", "keyId", "username", "password"}
	switch connectionType {
	case "MYSQL":
		required = append(required, "databaseName", "technologyType", "securityProtocol")
	case "ORACLE":
		required = append(required, "technologyType")
	}

	var missing []string
	for _, field := range required {
		if !connectionHasMeaningfulValue(specValues, field) {
			missing = append(missing, "spec."+field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("Connection requires %s for %s connections", strings.Join(missing, ", "), connectionType)
	}
	return nil
}

func validateConnectionTypeSpecificFields(connectionType string, specValues map[string]any) error {
	if len(specValues) == 0 {
		return nil
	}

	var unsupported []string
	switch connectionType {
	case "MYSQL":
		unsupported = connectionUnsupportedFields(
			specValues,
			"connectionString",
			"databaseId",
			"sshHost",
			"sshKey",
			"sshSudoLocation",
			"sshUser",
			"wallet",
		)
	case "ORACLE":
		unsupported = connectionUnsupportedFields(
			specValues,
			"additionalAttributes",
			"databaseName",
			"dbSystemId",
			"host",
			"port",
			"securityProtocol",
			"sslCa",
			"sslCert",
			"sslCrl",
			"sslKey",
			"sslMode",
		)
	default:
		return fmt.Errorf("unsupported Connection type %q", connectionType)
	}
	if len(unsupported) == 0 {
		return nil
	}

	return fmt.Errorf("Connection type %s does not support %s", connectionType, strings.Join(unsupported, ", "))
}

func connectionUnsupportedFields(specValues map[string]any, fieldNames ...string) []string {
	var unsupported []string
	for _, fieldName := range fieldNames {
		if connectionHasMeaningfulValue(specValues, fieldName) {
			unsupported = append(unsupported, "spec."+fieldName)
		}
	}
	return unsupported
}

func connectionHasMeaningfulValue(specValues map[string]any, field string) bool {
	value, ok := specValues[field]
	if !ok || value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case float64:
		return concrete != 0
	case []any:
		return len(concrete) > 0
	case map[string]any:
		return len(concrete) > 0
	default:
		return true
	}
}

func connectionJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func connectionMapSubsetEqual(desired map[string]any, current map[string]any) bool {
	for key, desiredValue := range desired {
		currentValue, ok := current[key]
		if !ok || !connectionValuesEqual(desiredValue, currentValue) {
			return false
		}
	}
	return true
}

func connectionValuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return string(leftPayload) == string(rightPayload)
}

func getConnectionWorkRequest(
	ctx context.Context,
	client connectionOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Connection OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Connection OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, databasemigrationsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveConnectionGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := connectionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveConnectionGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := connectionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := connectionWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverConnectionIDFromGeneratedWorkRequest(
	_ *databasemigrationv1beta1.Connection,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := connectionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := connectionWorkRequestActionForPhase(phase)
	if id, ok := resolveConnectionIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveConnectionIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Connection work request %s does not expose a connection identifier", stringValue(current.Id))
}

func connectionGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := connectionWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Connection %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func connectionWorkRequestFromAny(workRequest any) (databasemigrationsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case databasemigrationsdk.WorkRequest:
		return current, nil
	case *databasemigrationsdk.WorkRequest:
		if current == nil {
			return databasemigrationsdk.WorkRequest{}, fmt.Errorf("Connection work request is nil")
		}
		return *current, nil
	default:
		return databasemigrationsdk.WorkRequest{}, fmt.Errorf("unexpected Connection work request type %T", workRequest)
	}
}

func connectionWorkRequestPhaseFromOperationType(operationType databasemigrationsdk.OperationTypesEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case databasemigrationsdk.OperationTypesCreateConnection:
		return shared.OSOKAsyncPhaseCreate, true
	case databasemigrationsdk.OperationTypesUpdateConnection:
		return shared.OSOKAsyncPhaseUpdate, true
	case databasemigrationsdk.OperationTypesDeleteConnection:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func connectionWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) databasemigrationsdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return databasemigrationsdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return databasemigrationsdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return databasemigrationsdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func resolveConnectionIDFromResources(
	resources []databasemigrationsdk.WorkRequestResource,
	action databasemigrationsdk.WorkRequestResourceActionTypeEnum,
	preferConnectionOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferConnectionOnly && !isConnectionWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func isConnectionWorkRequestResource(resource databasemigrationsdk.WorkRequestResource) bool {
	return normalizeConnectionWorkRequestToken(stringValue(resource.EntityType)) == "connection"
}

func normalizeConnectionWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
