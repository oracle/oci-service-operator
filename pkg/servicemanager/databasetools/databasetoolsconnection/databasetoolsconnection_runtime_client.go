/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package databasetoolsconnection

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
	databasetoolsv1beta1 "github.com/oracle/oci-service-operator/api/databasetools/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type databaseToolsConnectionOCIClient interface {
	CreateDatabaseToolsConnection(context.Context, databasetoolssdk.CreateDatabaseToolsConnectionRequest) (databasetoolssdk.CreateDatabaseToolsConnectionResponse, error)
	GetDatabaseToolsConnection(context.Context, databasetoolssdk.GetDatabaseToolsConnectionRequest) (databasetoolssdk.GetDatabaseToolsConnectionResponse, error)
	ListDatabaseToolsConnections(context.Context, databasetoolssdk.ListDatabaseToolsConnectionsRequest) (databasetoolssdk.ListDatabaseToolsConnectionsResponse, error)
	UpdateDatabaseToolsConnection(context.Context, databasetoolssdk.UpdateDatabaseToolsConnectionRequest) (databasetoolssdk.UpdateDatabaseToolsConnectionResponse, error)
	DeleteDatabaseToolsConnection(context.Context, databasetoolssdk.DeleteDatabaseToolsConnectionRequest) (databasetoolssdk.DeleteDatabaseToolsConnectionResponse, error)
}

func init() {
	newDatabaseToolsConnectionServiceClient = func(manager *DatabaseToolsConnectionServiceManager) DatabaseToolsConnectionServiceClient {
		sdkClient, err := databasetoolssdk.NewDatabaseToolsClientWithConfigurationProvider(manager.Provider)
		config := newDatabaseToolsConnectionRuntimeConfig(manager.Log, manager.CredentialClient, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize DatabaseToolsConnection OCI client: %w", err)
		}
		return defaultDatabaseToolsConnectionServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*databasetoolsv1beta1.DatabaseToolsConnection](config),
		}
	}
}

func newDatabaseToolsConnectionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	credentialClient credhelper.CredentialClient,
	client databaseToolsConnectionOCIClient,
) DatabaseToolsConnectionServiceClient {
	return defaultDatabaseToolsConnectionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*databasetoolsv1beta1.DatabaseToolsConnection](
			newDatabaseToolsConnectionRuntimeConfig(log, credentialClient, client),
		),
	}
}

func newDatabaseToolsConnectionRuntimeConfig(
	log loggerutil.OSOKLogger,
	credentialClient credhelper.CredentialClient,
	client databaseToolsConnectionOCIClient,
) generatedruntime.Config[*databasetoolsv1beta1.DatabaseToolsConnection] {
	return generatedruntime.Config[*databasetoolsv1beta1.DatabaseToolsConnection]{
		Kind:             "DatabaseToolsConnection",
		SDKName:          "DatabaseToolsConnection",
		Log:              log,
		CredentialClient: credentialClient,
		Semantics:        databaseToolsConnectionRuntimeSemantics(),
		BuildCreateBody: func(
			ctx context.Context,
			resource *databasetoolsv1beta1.DatabaseToolsConnection,
			namespace string,
		) (any, error) {
			return buildDatabaseToolsConnectionCreateDetails(ctx, credentialClient, resource, namespace)
		},
		BuildUpdateBody: func(
			ctx context.Context,
			resource *databasetoolsv1beta1.DatabaseToolsConnection,
			namespace string,
			currentResponse any,
		) (any, bool, error) {
			return buildDatabaseToolsConnectionUpdateDetails(ctx, credentialClient, resource, namespace, currentResponse)
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &databasetoolssdk.CreateDatabaseToolsConnectionRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateDatabaseToolsConnection(ctx, *request.(*databasetoolssdk.CreateDatabaseToolsConnectionRequest))
			},
			Fields: databaseToolsConnectionCreateFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &databasetoolssdk.GetDatabaseToolsConnectionRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetDatabaseToolsConnection(ctx, *request.(*databasetoolssdk.GetDatabaseToolsConnectionRequest))
			},
			Fields: databaseToolsConnectionGetFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &databasetoolssdk.ListDatabaseToolsConnectionsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListDatabaseToolsConnections(ctx, *request.(*databasetoolssdk.ListDatabaseToolsConnectionsRequest))
			},
			Fields: databaseToolsConnectionListFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &databasetoolssdk.UpdateDatabaseToolsConnectionRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateDatabaseToolsConnection(ctx, *request.(*databasetoolssdk.UpdateDatabaseToolsConnectionRequest))
			},
			Fields: databaseToolsConnectionUpdateFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &databasetoolssdk.DeleteDatabaseToolsConnectionRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteDatabaseToolsConnection(ctx, *request.(*databasetoolssdk.DeleteDatabaseToolsConnectionRequest))
			},
			Fields: databaseToolsConnectionDeleteFields(),
		},
	}
}

func databaseToolsConnectionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "databasetools",
		FormalSlug:    "databasetoolsconnection",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE", "INACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"type",
				"connectionString",
				"url",
				"relatedResource.identifier",
				"userName",
				"privateEndpointId",
				"runtimeSupport",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"advancedProperties",
				"connectionString",
				"definedTags",
				"displayName",
				"freeformTags",
				"keyStores",
				"privateEndpointId",
				"proxyClient",
				"relatedResource",
				"url",
				"userName",
				"userPassword",
			},
			ForceNew:      []string{"compartmentId", "runtimeSupport", "type"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func databaseToolsConnectionCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDatabaseToolsConnectionDetails", RequestName: "CreateDatabaseToolsConnectionDetails", Contribution: "body"},
	}
}

func databaseToolsConnectionGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DatabaseToolsConnectionId", RequestName: "databaseToolsConnectionId", Contribution: "path", PreferResourceID: true},
	}
}

func databaseToolsConnectionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"spec.compartmentId", "status.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"spec.displayName", "status.displayName", "displayName"}},
		{FieldName: "RelatedResourceIdentifier", RequestName: "relatedResourceIdentifier", Contribution: "query", LookupPaths: []string{"spec.relatedResource.identifier", "status.relatedResource.identifier", "relatedResource.identifier"}},
	}
}

func databaseToolsConnectionUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DatabaseToolsConnectionId", RequestName: "databaseToolsConnectionId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateDatabaseToolsConnectionDetails", RequestName: "UpdateDatabaseToolsConnectionDetails", Contribution: "body"},
	}
}

func databaseToolsConnectionDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DatabaseToolsConnectionId", RequestName: "databaseToolsConnectionId", Contribution: "path", PreferResourceID: true},
	}
}

func buildDatabaseToolsConnectionCreateDetails(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *databasetoolsv1beta1.DatabaseToolsConnection,
	namespace string,
) (databasetoolssdk.CreateDatabaseToolsConnectionDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("databasetoolsconnection resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, credentialClient, namespace)
	if err != nil {
		return nil, err
	}

	connectionType, err := databaseToolsConnectionTypeForResource(resource, nil)
	if err != nil {
		return nil, err
	}

	specValues, err := databaseToolsConnectionJSONMap(resolvedSpec)
	if err != nil {
		return nil, fmt.Errorf("project DatabaseToolsConnection create spec: %w", err)
	}
	if err := validateDatabaseToolsConnectionTypeSpecificFields(connectionType, specValues); err != nil {
		return nil, err
	}

	return decodeDatabaseToolsConnectionCreateDetails(connectionType, resolvedSpec)
}

func buildDatabaseToolsConnectionUpdateDetails(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *databasetoolsv1beta1.DatabaseToolsConnection,
	namespace string,
	currentResponse any,
) (databasetoolssdk.UpdateDatabaseToolsConnectionDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("databasetoolsconnection resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, credentialClient, namespace)
	if err != nil {
		return nil, false, err
	}

	connectionType, err := databaseToolsConnectionTypeForResource(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}

	specValues, err := databaseToolsConnectionJSONMap(resolvedSpec)
	if err != nil {
		return nil, false, fmt.Errorf("project DatabaseToolsConnection update spec: %w", err)
	}
	if err := validateDatabaseToolsConnectionTypeSpecificFields(connectionType, specValues); err != nil {
		return nil, false, err
	}

	desired, err := decodeDatabaseToolsConnectionUpdateDetails(connectionType, resolvedSpec)
	if err != nil {
		return nil, false, err
	}
	desiredValues, err := databaseToolsConnectionJSONMap(desired)
	if err != nil {
		return nil, false, fmt.Errorf("project desired DatabaseToolsConnection update body: %w", err)
	}
	if len(desiredValues) == 0 {
		return desired, false, nil
	}

	currentDetails, err := databaseToolsConnectionCurrentUpdateDetails(connectionType, currentResponse)
	if err != nil {
		return nil, false, err
	}
	currentValues, err := databaseToolsConnectionJSONMap(currentDetails)
	if err != nil {
		return nil, false, fmt.Errorf("project current DatabaseToolsConnection update body: %w", err)
	}

	return desired, !databaseToolsConnectionMapSubsetEqual(desiredValues, currentValues), nil
}

func databaseToolsConnectionCurrentUpdateDetails(
	connectionType string,
	currentResponse any,
) (databasetoolssdk.UpdateDatabaseToolsConnectionDetails, error) {
	body, err := databaseToolsConnectionRuntimeBody(currentResponse)
	if err != nil {
		return nil, err
	}
	return decodeDatabaseToolsConnectionUpdateDetails(connectionType, body)
}

func databaseToolsConnectionTypeForResource(
	resource *databasetoolsv1beta1.DatabaseToolsConnection,
	currentResponse any,
) (string, error) {
	if resource != nil {
		if connectionType, ok := normalizeDatabaseToolsConnectionType(resource.Spec.Type); ok {
			return connectionType, nil
		}
	}

	if currentResponse != nil {
		body, err := databaseToolsConnectionRuntimeBody(currentResponse)
		if err != nil {
			return "", err
		}
		if connectionType, ok := normalizeDatabaseToolsConnectionType(databaseToolsConnectionTypeFromRuntimeBody(body)); ok {
			return connectionType, nil
		}
	}

	return "", fmt.Errorf("DatabaseToolsConnection spec.type must be set to a supported value")
}

func normalizeDatabaseToolsConnectionType(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	connectionType, ok := databasetoolssdk.GetMappingConnectionTypeEnum(raw)
	if !ok {
		return "", false
	}
	return string(connectionType), true
}

func decodeDatabaseToolsConnectionCreateDetails(
	connectionType string,
	raw any,
) (databasetoolssdk.CreateDatabaseToolsConnectionDetails, error) {
	switch connectionType {
	case "GENERIC_JDBC":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails](raw)
		return details, err
	case "POSTGRESQL":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.CreateDatabaseToolsConnectionPostgresqlDetails](raw)
		return details, err
	case "MYSQL":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.CreateDatabaseToolsConnectionMySqlDetails](raw)
		return details, err
	case "ORACLE_DATABASE":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.CreateDatabaseToolsConnectionOracleDatabaseDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported DatabaseToolsConnection create type %q", connectionType)
	}
}

func decodeDatabaseToolsConnectionUpdateDetails(
	connectionType string,
	raw any,
) (databasetoolssdk.UpdateDatabaseToolsConnectionDetails, error) {
	switch connectionType {
	case "GENERIC_JDBC":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.UpdateDatabaseToolsConnectionGenericJdbcDetails](raw)
		return details, err
	case "POSTGRESQL":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.UpdateDatabaseToolsConnectionPostgresqlDetails](raw)
		return details, err
	case "MYSQL":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.UpdateDatabaseToolsConnectionMySqlDetails](raw)
		return details, err
	case "ORACLE_DATABASE":
		details, err := decodeDatabaseToolsConnectionConcrete[databasetoolssdk.UpdateDatabaseToolsConnectionOracleDatabaseDetails](raw)
		return details, err
	default:
		return nil, fmt.Errorf("unsupported DatabaseToolsConnection update type %q", connectionType)
	}
}

func decodeDatabaseToolsConnectionConcrete[T any](raw any) (T, error) {
	var decoded T

	payload, err := json.Marshal(raw)
	if err != nil {
		return decoded, fmt.Errorf("marshal DatabaseToolsConnection payload: %w", err)
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return decoded, fmt.Errorf("unmarshal DatabaseToolsConnection payload: %w", err)
	}

	return decoded, nil
}

func databaseToolsConnectionRuntimeBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case databasetoolssdk.CreateDatabaseToolsConnectionResponse:
		return current.DatabaseToolsConnection, nil
	case *databasetoolssdk.CreateDatabaseToolsConnectionResponse:
		if current == nil {
			return nil, fmt.Errorf("current DatabaseToolsConnection response is nil")
		}
		return current.DatabaseToolsConnection, nil
	case databasetoolssdk.GetDatabaseToolsConnectionResponse:
		return current.DatabaseToolsConnection, nil
	case *databasetoolssdk.GetDatabaseToolsConnectionResponse:
		if current == nil {
			return nil, fmt.Errorf("current DatabaseToolsConnection response is nil")
		}
		return current.DatabaseToolsConnection, nil
	case databasetoolssdk.DatabaseToolsConnection:
		if current == nil {
			return nil, fmt.Errorf("current DatabaseToolsConnection body is nil")
		}
		return current, nil
	case databasetoolssdk.DatabaseToolsConnectionSummary:
		if current == nil {
			return nil, fmt.Errorf("current DatabaseToolsConnection summary is nil")
		}
		return current, nil
	case databasetoolssdk.DatabaseToolsConnectionOracleDatabase,
		databasetoolssdk.DatabaseToolsConnectionMySql,
		databasetoolssdk.DatabaseToolsConnectionPostgresql,
		databasetoolssdk.DatabaseToolsConnectionGenericJdbc,
		databasetoolssdk.DatabaseToolsConnectionOracleDatabaseSummary,
		databasetoolssdk.DatabaseToolsConnectionMySqlSummary,
		databasetoolssdk.DatabaseToolsConnectionPostgresqlSummary,
		databasetoolssdk.DatabaseToolsConnectionGenericJdbcSummary:
		return current, nil
	default:
		return nil, fmt.Errorf("unsupported current DatabaseToolsConnection payload type %T", currentResponse)
	}
}

func databaseToolsConnectionTypeFromRuntimeBody(body any) string {
	switch current := body.(type) {
	case databasetoolssdk.DatabaseToolsConnectionOracleDatabase,
		databasetoolssdk.DatabaseToolsConnectionOracleDatabaseSummary:
		return "ORACLE_DATABASE"
	case databasetoolssdk.DatabaseToolsConnectionMySql,
		databasetoolssdk.DatabaseToolsConnectionMySqlSummary:
		return "MYSQL"
	case databasetoolssdk.DatabaseToolsConnectionPostgresql,
		databasetoolssdk.DatabaseToolsConnectionPostgresqlSummary:
		return "POSTGRESQL"
	case databasetoolssdk.DatabaseToolsConnectionGenericJdbc,
		databasetoolssdk.DatabaseToolsConnectionGenericJdbcSummary:
		return "GENERIC_JDBC"
	case databasetoolssdk.DatabaseToolsConnection:
		return databaseToolsConnectionTypeFromRuntimeBodyValue(current)
	case databasetoolssdk.DatabaseToolsConnectionSummary:
		return databaseToolsConnectionTypeFromRuntimeBodyValue(current)
	default:
		return ""
	}
}

func databaseToolsConnectionTypeFromRuntimeBodyValue(body any) string {
	values, err := databaseToolsConnectionJSONMap(body)
	if err != nil {
		return ""
	}

	rawType, ok := databaseToolsConnectionLookupValue(values, "type")
	if !ok {
		return ""
	}
	connectionType, _ := rawType.(string)
	return connectionType
}

func validateDatabaseToolsConnectionTypeSpecificFields(connectionType string, specValues map[string]any) error {
	if len(specValues) == 0 {
		return nil
	}

	var unsupported []string
	switch connectionType {
	case "GENERIC_JDBC":
		unsupported = databaseToolsConnectionUnsupportedFields(
			specValues,
			"connectionString",
			"privateEndpointId",
			"proxyClient",
			"relatedResource",
		)
	case "MYSQL", "POSTGRESQL":
		unsupported = databaseToolsConnectionUnsupportedFields(specValues, "proxyClient", "url")
	case "ORACLE_DATABASE":
		unsupported = databaseToolsConnectionUnsupportedFields(specValues, "url")
	default:
		return fmt.Errorf("unsupported DatabaseToolsConnection type %q", connectionType)
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("DatabaseToolsConnection type %s does not support fields: %s", connectionType, strings.Join(unsupported, ", "))
}

func databaseToolsConnectionUnsupportedFields(specValues map[string]any, paths ...string) []string {
	unsupported := make([]string, 0, len(paths))
	for _, path := range paths {
		if value, ok := databaseToolsConnectionLookupValue(specValues, path); ok && databaseToolsConnectionMeaningfulValue(value) {
			unsupported = append(unsupported, path)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}

func databaseToolsConnectionJSONMap(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func databaseToolsConnectionMapSubsetEqual(want map[string]any, have map[string]any) bool {
	for key, wantValue := range want {
		if !databaseToolsConnectionMeaningfulValue(wantValue) {
			continue
		}

		haveValue, ok := databaseToolsConnectionLookupMapValue(have, key)
		if !ok {
			return false
		}

		wantMap, wantIsMap := wantValue.(map[string]any)
		haveMap, haveIsMap := haveValue.(map[string]any)
		if wantIsMap && haveIsMap {
			if !databaseToolsConnectionMapSubsetEqual(wantMap, haveMap) {
				return false
			}
			continue
		}

		if !databaseToolsConnectionValuesEqual(wantValue, haveValue) {
			return false
		}
	}
	return true
}

func databaseToolsConnectionLookupValue(values map[string]any, path string) (any, bool) {
	if values == nil {
		return nil, false
	}

	current := any(values)
	for _, segment := range strings.Split(strings.TrimSpace(path), ".") {
		if segment == "" {
			return nil, false
		}

		valueMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		next, ok := databaseToolsConnectionLookupMapValue(valueMap, segment)
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func databaseToolsConnectionLookupMapValue(values map[string]any, key string) (any, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}

	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	for candidate, value := range values {
		if strings.ToLower(strings.TrimSpace(candidate)) == normalizedKey {
			return value, true
		}
	}
	return nil, false
}

func databaseToolsConnectionMeaningfulValue(value any) bool {
	if value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case []any:
		for _, item := range concrete {
			if databaseToolsConnectionMeaningfulValue(item) {
				return true
			}
		}
		return false
	case map[string]any:
		for _, item := range concrete {
			if databaseToolsConnectionMeaningfulValue(item) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func databaseToolsConnectionValuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
