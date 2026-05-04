/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package databaseinsight

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	databaseInsightDeletePendingMessage      = "OCI DatabaseInsight delete is in progress"
	databaseInsightPendingWriteDeleteMessage = "OCI DatabaseInsight create or update is in progress; waiting before delete"
)

var (
	// The OCI SDK models DatabaseInsight create/update bodies as non-empty
	// polymorphic interfaces, so keep the typed body beside the generated
	// request builder and attach it in the operation wrapper.
	pendingDatabaseInsightCreateBodies sync.Map
	pendingDatabaseInsightUpdateBodies sync.Map
	databaseInsightRequestBodySequence atomic.Uint64
)

type databaseInsightRequestBodyContextKey struct{}

type databaseInsightOCIClient interface {
	CreateDatabaseInsight(context.Context, opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error)
	GetDatabaseInsight(context.Context, opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error)
	ListDatabaseInsights(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error)
	UpdateDatabaseInsight(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error)
	DeleteDatabaseInsight(context.Context, opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error)
}

type databaseInsightListCall func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error)

type databaseInsightAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e databaseInsightAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e databaseInsightAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type databaseInsightNotFoundError struct {
	message string
}

func (e databaseInsightNotFoundError) Error() string {
	return e.message
}

func init() {
	registerDatabaseInsightRuntimeHooksMutator(func(manager *DatabaseInsightServiceManager, hooks *DatabaseInsightRuntimeHooks) {
		applyDatabaseInsightRuntimeHooks(manager, hooks)
	})
}

func applyDatabaseInsightRuntimeHooks(_ *DatabaseInsightServiceManager, hooks *DatabaseInsightRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = databaseInsightRuntimeSemantics()
	hooks.BuildCreateBody = buildDatabaseInsightCreateBody
	hooks.BuildUpdateBody = buildDatabaseInsightUpdateBody
	hooks.Identity.Resolve = resolveDatabaseInsightIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardDatabaseInsightExistingBeforeCreate
	hooks.Identity.LookupExisting = func(ctx context.Context, resource *opsiv1beta1.DatabaseInsight, _ any) (any, error) {
		return lookupExistingDatabaseInsight(ctx, hooks.List.Call, resource)
	}
	hooks.List.Fields = databaseInsightListFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDatabaseInsightCreateOnlyDrift
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *opsiv1beta1.DatabaseInsight, currentID string) (any, error) {
		return confirmDatabaseInsightDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleDatabaseInsightDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDatabaseInsightDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markDatabaseInsightTerminating
	wrapDatabaseInsightRequestBodyContext(hooks)
	wrapDatabaseInsightRequestBodies(hooks)
	wrapDatabaseInsightResponses(hooks)
	wrapDatabaseInsightDeleteConfirmation(hooks)
}

func databaseInsightRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "opsi",
		FormalSlug:    "databaseinsight",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(opsisdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:        "required",
			PendingStates: []string{string(opsisdk.LifecycleStateDeleting)},
			TerminalStates: []string{
				string(opsisdk.LifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"entitySource",
				"databaseId",
				"enterpriseManagerBridgeId",
				"enterpriseManagerIdentifier",
				"enterpriseManagerEntityIdentifier",
				"exadataInsightId",
				"opsiPrivateEndpointId",
				"databaseConnectorId",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"entitySource",
				"databaseId",
				"managementAgentId",
				"connectionDetails",
				"connectionCredentialDetails",
				"databaseResourceType",
				"systemTags",
				"deploymentType",
				"databaseConnectorId",
				"isAdvancedFeaturesEnabled",
				"credentialDetails",
				"opsiPrivateEndpointId",
				"enterpriseManagerIdentifier",
				"enterpriseManagerBridgeId",
				"enterpriseManagerEntityIdentifier",
				"exadataInsightId",
				"serviceName",
				"dbmPrivateEndpointId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DatabaseInsight", Action: "CreateDatabaseInsight"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DatabaseInsight", Action: "UpdateDatabaseInsight"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DatabaseInsight", Action: "DeleteDatabaseInsight"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DatabaseInsight", Action: "GetDatabaseInsight"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DatabaseInsight", Action: "GetDatabaseInsight"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DatabaseInsight", Action: "GetDatabaseInsight"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func newDatabaseInsightServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client databaseInsightOCIClient,
) DatabaseInsightServiceClient {
	manager := &DatabaseInsightServiceManager{Log: log}
	hooks := newDatabaseInsightRuntimeHooksWithOCIClient(client)
	applyDatabaseInsightRuntimeHooks(manager, &hooks)
	delegate := defaultDatabaseInsightServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.DatabaseInsight](
			buildDatabaseInsightGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDatabaseInsightGeneratedClient(hooks, delegate)
}

func newDatabaseInsightRuntimeHooksWithOCIClient(client databaseInsightOCIClient) DatabaseInsightRuntimeHooks {
	hooks := newDatabaseInsightDefaultRuntimeHooks(opsisdk.OperationsInsightsClient{})
	hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
		if client == nil {
			return opsisdk.CreateDatabaseInsightResponse{}, fmt.Errorf("DatabaseInsight OCI client is not configured")
		}
		return client.CreateDatabaseInsight(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
		if client == nil {
			return opsisdk.GetDatabaseInsightResponse{}, fmt.Errorf("DatabaseInsight OCI client is not configured")
		}
		return client.GetDatabaseInsight(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
		if client == nil {
			return opsisdk.ListDatabaseInsightsResponse{}, fmt.Errorf("DatabaseInsight OCI client is not configured")
		}
		return client.ListDatabaseInsights(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
		if client == nil {
			return opsisdk.UpdateDatabaseInsightResponse{}, fmt.Errorf("DatabaseInsight OCI client is not configured")
		}
		return client.UpdateDatabaseInsight(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error) {
		if client == nil {
			return opsisdk.DeleteDatabaseInsightResponse{}, fmt.Errorf("DatabaseInsight OCI client is not configured")
		}
		return client.DeleteDatabaseInsight(ctx, request)
	}
	return hooks
}

func databaseInsightListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "EnterpriseManagerBridgeId", RequestName: "enterpriseManagerBridgeId", Contribution: "query", LookupPaths: []string{"status.enterpriseManagerBridgeId", "spec.enterpriseManagerBridgeId", "enterpriseManagerBridgeId"}},
		{FieldName: "ExadataInsightId", RequestName: "exadataInsightId", Contribution: "query", LookupPaths: []string{"status.exadataInsightId", "spec.exadataInsightId", "exadataInsightId"}},
		{FieldName: "OpsiPrivateEndpointId", RequestName: "opsiPrivateEndpointId", Contribution: "query", LookupPaths: []string{"status.opsiPrivateEndpointId", "spec.opsiPrivateEndpointId", "opsiPrivateEndpointId"}},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardDatabaseInsightExistingBeforeCreate(
	_ context.Context,
	resource *opsiv1beta1.DatabaseInsight,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if trackedDatabaseInsightID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveDatabaseInsightIdentity(resource *opsiv1beta1.DatabaseInsight) (any, error) {
	if resource == nil {
		return nil, nil
	}
	return databaseInsightDesiredBodyMap(resource.Spec)
}

func buildDatabaseInsightCreateBody(ctx context.Context, resource *opsiv1beta1.DatabaseInsight, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("DatabaseInsight resource is nil")
	}
	body, err := databaseInsightDesiredBodyMap(resource.Spec)
	if err != nil {
		return nil, err
	}
	details, err := databaseInsightCreateDetailsFromMap(body)
	if err != nil {
		return nil, err
	}
	if err := stashDatabaseInsightCreateBody(ctx, resource, details); err != nil {
		return nil, err
	}
	return nil, nil
}

func buildDatabaseInsightUpdateBody(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("DatabaseInsight resource is nil")
	}
	current := databaseInsightRawFromResponse(currentResponse)
	if len(current) == 0 {
		return nil, false, fmt.Errorf("current DatabaseInsight response does not expose a DatabaseInsight body")
	}

	desired, err := databaseInsightDesiredBodyMap(resource.Spec)
	if err != nil {
		return nil, false, err
	}
	update := databaseInsightMutableUpdate(resource.Spec, desired, current)
	if len(update) == 0 {
		return nil, false, nil
	}

	entitySource := firstDatabaseInsightString(desired, current, "entitySource")
	if entitySource == "" {
		return nil, false, fmt.Errorf("entitySource is required to build DatabaseInsight update body")
	}
	update["entitySource"] = entitySource
	body, err := databaseInsightUpdateDetailsFromMap(update)
	if err != nil {
		return nil, false, err
	}
	if err := stashDatabaseInsightUpdateBody(ctx, resource, current, body); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

func databaseInsightMutableUpdate(
	spec opsiv1beta1.DatabaseInsightSpec,
	desired map[string]any,
	current map[string]any,
) map[string]any {
	update := map[string]any{}
	for _, field := range []string{"freeformTags", "definedTags"} {
		if !databaseInsightDesiredHasField(spec, desired, field) {
			continue
		}
		if reflect.DeepEqual(normalizedJSONValue(desired[field]), normalizedJSONValue(current[field])) {
			continue
		}
		update[field] = desired[field]
	}
	return update
}

func validateDatabaseInsightCreateOnlyDrift(
	resource *opsiv1beta1.DatabaseInsight,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("DatabaseInsight resource is nil")
	}
	current := databaseInsightRawFromResponse(currentResponse)
	if len(current) == 0 {
		return nil
	}
	desired, err := databaseInsightDesiredBodyMap(resource.Spec)
	if err != nil {
		return err
	}
	for _, field := range []string{
		"compartmentId",
		"entitySource",
		"databaseId",
		"managementAgentId",
		"connectionDetails",
		"connectionCredentialDetails",
		"databaseResourceType",
		"systemTags",
		"deploymentType",
		"databaseConnectorId",
		"isAdvancedFeaturesEnabled",
		"credentialDetails",
		"opsiPrivateEndpointId",
		"enterpriseManagerIdentifier",
		"enterpriseManagerBridgeId",
		"enterpriseManagerEntityIdentifier",
		"exadataInsightId",
		"serviceName",
		"dbmPrivateEndpointId",
	} {
		if err := rejectDatabaseInsightCreateOnlyDrift(field, desired, current); err != nil {
			return err
		}
	}
	return nil
}

func rejectDatabaseInsightCreateOnlyDrift(field string, desired map[string]any, current map[string]any) error {
	desiredValue, desiredOK := meaningfulDatabaseInsightMapValue(desired, field)
	if !desiredOK {
		return nil
	}
	currentValue, currentOK := meaningfulDatabaseInsightMapValue(current, field)
	if !currentOK {
		return fmt.Errorf("DatabaseInsight formal semantics require replacement when %s changes", field)
	}
	if reflect.DeepEqual(normalizedJSONValue(desiredValue), normalizedJSONValue(currentValue)) {
		return nil
	}
	return fmt.Errorf("DatabaseInsight formal semantics require replacement when %s changes", field)
}

func databaseInsightDesiredBodyMap(spec opsiv1beta1.DatabaseInsightSpec) (map[string]any, error) {
	body, err := databaseInsightJSONDataMap(spec.JsonData)
	if err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]any{}
	}
	if err := databaseInsightApplySpecDefaults(body, spec); err != nil {
		return nil, err
	}
	return body, nil
}

func databaseInsightApplySpecDefaults(body map[string]any, spec opsiv1beta1.DatabaseInsightSpec) error {
	defaults := databaseInsightStringDefaults(spec)
	databaseInsightApplyTagDefaults(defaults, spec)
	if err := databaseInsightApplyNestedDefaults(defaults, body, spec); err != nil {
		return err
	}
	databaseInsightApplyAdvancedFeaturesDefault(defaults, body, spec)
	return setDatabaseInsightDefaults(body, defaults)
}

func databaseInsightStringDefaults(spec opsiv1beta1.DatabaseInsightSpec) map[string]any {
	defaults := map[string]any{}
	stringDefaults := []struct {
		key   string
		value string
	}{
		{key: "compartmentId", value: spec.CompartmentId},
		{key: "entitySource", value: spec.EntitySource},
		{key: "databaseId", value: spec.DatabaseId},
		{key: "managementAgentId", value: spec.ManagementAgentId},
		{key: "databaseResourceType", value: spec.DatabaseResourceType},
		{key: "deploymentType", value: spec.DeploymentType},
		{key: "databaseConnectorId", value: spec.DatabaseConnectorId},
		{key: "opsiPrivateEndpointId", value: spec.OpsiPrivateEndpointId},
		{key: "enterpriseManagerIdentifier", value: spec.EnterpriseManagerIdentifier},
		{key: "enterpriseManagerBridgeId", value: spec.EnterpriseManagerBridgeId},
		{key: "enterpriseManagerEntityIdentifier", value: spec.EnterpriseManagerEntityIdentifier},
		{key: "exadataInsightId", value: spec.ExadataInsightId},
		{key: "serviceName", value: spec.ServiceName},
		{key: "dbmPrivateEndpointId", value: spec.DbmPrivateEndpointId},
	}
	for _, field := range stringDefaults {
		setDatabaseInsightStringDefault(defaults, field.key, field.value)
	}
	return defaults
}

func databaseInsightApplyTagDefaults(defaults map[string]any, spec opsiv1beta1.DatabaseInsightSpec) {
	if spec.FreeformTags != nil {
		defaults["freeformTags"] = cloneDatabaseInsightStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		defaults["definedTags"] = databaseInsightDefinedTags(spec.DefinedTags)
	}
	if spec.SystemTags != nil {
		defaults["systemTags"] = databaseInsightDefinedTags(spec.SystemTags)
	}
}

func databaseInsightApplyNestedDefaults(
	defaults map[string]any,
	body map[string]any,
	spec opsiv1beta1.DatabaseInsightSpec,
) error {
	if connection := databaseInsightConnectionDetailsForEntitySource(defaults, body, spec.ConnectionDetails); connection != nil {
		defaults["connectionDetails"] = connection
	}
	credential, err := databaseInsightConnectionCredentialDetails(spec.ConnectionCredentialDetails)
	if err != nil {
		return fmt.Errorf("connectionCredentialDetails.jsonData: %w", err)
	}
	if credential != nil {
		defaults["connectionCredentialDetails"] = credential
	}
	credential, err = databaseInsightCredentialDetails(spec.CredentialDetails)
	if err != nil {
		return fmt.Errorf("credentialDetails.jsonData: %w", err)
	}
	if credential != nil {
		defaults["credentialDetails"] = credential
	}
	return nil
}

func databaseInsightApplyAdvancedFeaturesDefault(
	defaults map[string]any,
	body map[string]any,
	spec opsiv1beta1.DatabaseInsightSpec,
) {
	if spec.IsAdvancedFeaturesEnabled {
		defaults["isAdvancedFeaturesEnabled"] = true
	} else if strings.EqualFold(databaseInsightStringMapValue(defaults, body, "entitySource"), string(databaseInsightEntitySourceAutonomous)) {
		if _, ok := body["isAdvancedFeaturesEnabled"]; !ok {
			defaults["isAdvancedFeaturesEnabled"] = false
		}
	}
}

func setDatabaseInsightDefaults(body map[string]any, defaults map[string]any) error {
	for key, value := range defaults {
		if err := setDatabaseInsightDefault(body, key, value); err != nil {
			return err
		}
	}
	return nil
}

func setDatabaseInsightDefault(body map[string]any, key string, value any) error {
	if key == "isAdvancedFeaturesEnabled" {
		if existing, ok := body[key]; ok {
			if wanted, ok := value.(bool); ok && wanted &&
				!reflect.DeepEqual(normalizedJSONValue(existing), normalizedJSONValue(value)) {
				return fmt.Errorf("DatabaseInsight jsonData field %s conflicts with typed spec field", key)
			}
			return nil
		}
		body[key] = value
		return nil
	}
	if !meaningfulDatabaseInsightValue(value) {
		return nil
	}
	if existing, ok := body[key]; ok && meaningfulDatabaseInsightValue(existing) {
		if !reflect.DeepEqual(normalizedJSONValue(existing), normalizedJSONValue(value)) {
			return fmt.Errorf("DatabaseInsight jsonData field %s conflicts with typed spec field", key)
		}
		return nil
	}
	body[key] = value
	return nil
}

func setDatabaseInsightStringDefault(body map[string]any, key string, value string) {
	if value = strings.TrimSpace(value); value != "" {
		body[key] = value
	}
}

type databaseInsightEntitySource string

const (
	databaseInsightEntitySourceAutonomous            databaseInsightEntitySource = "AUTONOMOUS_DATABASE"
	databaseInsightEntitySourceMacsManagedCloud      databaseInsightEntitySource = "MACS_MANAGED_CLOUD_DATABASE"
	databaseInsightEntitySourceMacsManagedAutonomous databaseInsightEntitySource = "MACS_MANAGED_AUTONOMOUS_DATABASE"
	databaseInsightEntitySourcePeComanaged           databaseInsightEntitySource = "PE_COMANAGED_DATABASE"
	databaseInsightEntitySourceMdsMySQL              databaseInsightEntitySource = "MDS_MYSQL_DATABASE_SYSTEM"
	databaseInsightEntitySourceExternalMySQL         databaseInsightEntitySource = "EXTERNAL_MYSQL_DATABASE_SYSTEM"
	databaseInsightEntitySourceEMManagedExternal     databaseInsightEntitySource = "EM_MANAGED_EXTERNAL_DATABASE"
	databaseInsightEntitySourceMacsManagedExternal   databaseInsightEntitySource = "MACS_MANAGED_EXTERNAL_DATABASE"
)

func databaseInsightCreateDetailsFromMap(body map[string]any) (opsisdk.CreateDatabaseInsightDetails, error) {
	entitySource := databaseInsightEntitySource(strings.ToUpper(strings.TrimSpace(stringValueFromMap(body, "entitySource"))))
	if entitySource == "" {
		return nil, fmt.Errorf("entitySource is required to build DatabaseInsight create body")
	}
	switch entitySource {
	case databaseInsightEntitySourceAutonomous:
		var details opsisdk.CreateAutonomousDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMacsManagedCloud:
		var details opsisdk.CreateMacsManagedCloudDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMacsManagedAutonomous:
		var details opsisdk.CreateMacsManagedAutonomousDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourcePeComanaged:
		var details opsisdk.CreatePeComanagedDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMdsMySQL:
		var details opsisdk.CreateMdsMySqlDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceExternalMySQL:
		var details opsisdk.CreateExternalMysqlDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceEMManagedExternal:
		var details opsisdk.CreateEmManagedExternalDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	default:
		return nil, fmt.Errorf("unsupported DatabaseInsight entitySource %q for create", entitySource)
	}
}

func databaseInsightUpdateDetailsFromMap(body map[string]any) (opsisdk.UpdateDatabaseInsightDetails, error) {
	entitySource := databaseInsightEntitySource(strings.ToUpper(strings.TrimSpace(stringValueFromMap(body, "entitySource"))))
	switch entitySource {
	case databaseInsightEntitySourceAutonomous:
		var details opsisdk.UpdateAutonomousDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMacsManagedCloud:
		var details opsisdk.UpdateMacsManagedCloudDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMacsManagedAutonomous:
		var details opsisdk.UpdateMacsManagedAutonomousDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourcePeComanaged:
		var details opsisdk.UpdatePeComanagedDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMdsMySQL:
		var details opsisdk.UpdateMdsMySqlDatabaseInsight
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceExternalMySQL:
		var details opsisdk.UpdateExternalMysqlDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceEMManagedExternal:
		var details opsisdk.UpdateEmManagedExternalDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	case databaseInsightEntitySourceMacsManagedExternal:
		var details opsisdk.UpdateMacsManagedExternalDatabaseInsightDetails
		return details, decodeDatabaseInsightBody(body, &details)
	default:
		return nil, fmt.Errorf("unsupported DatabaseInsight entitySource %q for update", entitySource)
	}
}

func decodeDatabaseInsightBody(body map[string]any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal DatabaseInsight body: %w", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("unmarshal DatabaseInsight body into %T: %w", target, err)
	}
	return nil
}

func databaseInsightConnectionDetailsForEntitySource(
	defaults map[string]any,
	body map[string]any,
	details opsiv1beta1.DatabaseInsightConnectionDetails,
) any {
	entitySource := databaseInsightEntitySource(strings.ToUpper(strings.TrimSpace(databaseInsightStringMapValue(defaults, body, "entitySource"))))
	if entitySource == databaseInsightEntitySourcePeComanaged {
		connection := databaseInsightPeComanagedConnectionDetails(details)
		if connection == nil {
			return nil
		}
		return connection
	}
	connection := databaseInsightConnectionDetails(details)
	if connection == nil {
		return nil
	}
	return connection
}

func databaseInsightConnectionDetails(details opsiv1beta1.DatabaseInsightConnectionDetails) *opsisdk.ConnectionDetails {
	if strings.TrimSpace(details.HostName) == "" &&
		strings.TrimSpace(details.Protocol) == "" &&
		details.Port == 0 &&
		strings.TrimSpace(details.ServiceName) == "" {
		return nil
	}
	return &opsisdk.ConnectionDetails{
		HostName:    databaseInsightStringPointer(details.HostName),
		Protocol:    opsisdk.ConnectionDetailsProtocolEnum(strings.TrimSpace(details.Protocol)),
		Port:        databaseInsightIntPointer(details.Port),
		ServiceName: databaseInsightStringPointer(details.ServiceName),
	}
}

func databaseInsightPeComanagedConnectionDetails(
	details opsiv1beta1.DatabaseInsightConnectionDetails,
) *opsisdk.PeComanagedDatabaseConnectionDetails {
	if strings.TrimSpace(details.HostName) == "" &&
		strings.TrimSpace(details.Protocol) == "" &&
		details.Port == 0 &&
		strings.TrimSpace(details.ServiceName) == "" {
		return nil
	}
	connection := &opsisdk.PeComanagedDatabaseConnectionDetails{
		Protocol:    opsisdk.PeComanagedDatabaseConnectionDetailsProtocolEnum(strings.TrimSpace(details.Protocol)),
		ServiceName: databaseInsightStringPointer(details.ServiceName),
	}
	host := opsisdk.PeComanagedDatabaseHostDetails{
		HostIp: databaseInsightStringPointer(details.HostName),
		Port:   databaseInsightIntPointer(details.Port),
	}
	if host.HostIp != nil || host.Port != nil {
		connection.Hosts = []opsisdk.PeComanagedDatabaseHostDetails{host}
	}
	return connection
}

func databaseInsightConnectionCredentialDetails(
	details opsiv1beta1.DatabaseInsightConnectionCredentialDetails,
) (opsisdk.CredentialDetails, error) {
	return databaseInsightCredentialDetails(opsiv1beta1.DatabaseInsightCredentialDetails(details))
}

func databaseInsightCredentialDetails(details opsiv1beta1.DatabaseInsightCredentialDetails) (opsisdk.CredentialDetails, error) {
	if raw := strings.TrimSpace(details.JsonData); raw != "" {
		credential, err := databaseInsightCredentialDetailsFromJSONData(raw)
		if err != nil {
			return nil, err
		}
		return credential, nil
	}
	return databaseInsightCredentialDetailsFromTypedFields(details), nil
}

func databaseInsightCredentialDetailsFromTypedFields(details opsiv1beta1.DatabaseInsightCredentialDetails) opsisdk.CredentialDetails {
	switch strings.ToUpper(strings.TrimSpace(details.CredentialType)) {
	case string(opsisdk.CredentialDetailsCredentialTypeSource):
		return opsisdk.CredentialsBySource{
			CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
		}
	case string(opsisdk.CredentialDetailsCredentialTypeNamedCreds):
		return opsisdk.CredentialByNamedCredentials{
			CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
			NamedCredentialId:    databaseInsightStringPointer(details.NamedCredentialId),
		}
	case string(opsisdk.CredentialDetailsCredentialTypeVault):
		return opsisdk.CredentialByVault{
			CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
			UserName:             databaseInsightStringPointer(details.UserName),
			PasswordSecretId:     databaseInsightStringPointer(details.PasswordSecretId),
			WalletSecretId:       databaseInsightStringPointer(details.WalletSecretId),
			Role:                 opsisdk.CredentialByVaultRoleEnum(strings.TrimSpace(details.Role)),
		}
	case string(opsisdk.CredentialDetailsCredentialTypeIam):
		return opsisdk.CredentialByIam{
			CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
		}
	default:
		if !databaseInsightHasTypedCredentialFields(details) {
			return nil
		}
		if strings.TrimSpace(details.NamedCredentialId) != "" {
			return opsisdk.CredentialByNamedCredentials{
				CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
				NamedCredentialId:    databaseInsightStringPointer(details.NamedCredentialId),
			}
		}
		if databaseInsightHasOnlyCredentialSourceName(details) {
			return opsisdk.CredentialsBySource{
				CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
			}
		}
		return opsisdk.CredentialByVault{
			CredentialSourceName: databaseInsightStringPointer(details.CredentialSourceName),
			UserName:             databaseInsightStringPointer(details.UserName),
			PasswordSecretId:     databaseInsightStringPointer(details.PasswordSecretId),
			WalletSecretId:       databaseInsightStringPointer(details.WalletSecretId),
			Role:                 opsisdk.CredentialByVaultRoleEnum(strings.TrimSpace(details.Role)),
		}
	}
}

func databaseInsightHasTypedCredentialFields(details opsiv1beta1.DatabaseInsightCredentialDetails) bool {
	for _, value := range []string{
		details.CredentialSourceName,
		details.NamedCredentialId,
		details.UserName,
		details.PasswordSecretId,
		details.WalletSecretId,
		details.Role,
	} {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func databaseInsightHasOnlyCredentialSourceName(details opsiv1beta1.DatabaseInsightCredentialDetails) bool {
	return strings.TrimSpace(details.CredentialSourceName) != "" &&
		strings.TrimSpace(details.UserName) == "" &&
		strings.TrimSpace(details.PasswordSecretId) == "" &&
		strings.TrimSpace(details.WalletSecretId) == "" &&
		strings.TrimSpace(details.Role) == ""
}

func databaseInsightCredentialDetailsFromJSONData(raw string) (opsisdk.CredentialDetails, error) {
	var discriminator struct {
		CredentialType string `json:"credentialType"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode DatabaseInsight credential jsonData: %w", err)
	}
	switch strings.ToUpper(strings.TrimSpace(discriminator.CredentialType)) {
	case string(opsisdk.CredentialDetailsCredentialTypeSource):
		var details opsisdk.CredentialsBySource
		return details, json.Unmarshal([]byte(raw), &details)
	case string(opsisdk.CredentialDetailsCredentialTypeNamedCreds):
		var details opsisdk.CredentialByNamedCredentials
		return details, json.Unmarshal([]byte(raw), &details)
	case string(opsisdk.CredentialDetailsCredentialTypeVault):
		var details opsisdk.CredentialByVault
		return details, json.Unmarshal([]byte(raw), &details)
	case string(opsisdk.CredentialDetailsCredentialTypeIam):
		var details opsisdk.CredentialByIam
		return details, json.Unmarshal([]byte(raw), &details)
	default:
		return nil, fmt.Errorf("unsupported DatabaseInsight credentialType %q", discriminator.CredentialType)
	}
}

func lookupExistingDatabaseInsight(
	ctx context.Context,
	list databaseInsightListCall,
	resource *opsiv1beta1.DatabaseInsight,
) (any, error) {
	if list == nil || resource == nil {
		return nil, nil
	}
	if !databaseInsightHasListIdentity(resource) {
		return nil, nil
	}
	response, err := list(ctx, databaseInsightListRequest(resource))
	if err != nil {
		return nil, err
	}
	summary, found, err := selectDatabaseInsightSummary(response.Items, resource, false)
	if err != nil {
		return nil, err
	}
	if !found {
		return opsisdk.GetDatabaseInsightResponse{}, nil
	}
	return summary, nil
}

func databaseInsightHasListIdentity(resource *opsiv1beta1.DatabaseInsight) bool {
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return false
	}
	for _, value := range []string{
		resource.Spec.DatabaseId,
		resource.Spec.EnterpriseManagerBridgeId,
		resource.Spec.EnterpriseManagerIdentifier,
		resource.Spec.EnterpriseManagerEntityIdentifier,
		resource.Spec.DatabaseConnectorId,
		resource.Spec.ExadataInsightId,
		resource.Spec.OpsiPrivateEndpointId,
	} {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func databaseInsightListRequest(resource *opsiv1beta1.DatabaseInsight) opsisdk.ListDatabaseInsightsRequest {
	request := opsisdk.ListDatabaseInsightsRequest{
		CompartmentId:             databaseInsightStringPointer(resource.Spec.CompartmentId),
		EnterpriseManagerBridgeId: databaseInsightStringPointer(resource.Spec.EnterpriseManagerBridgeId),
		ExadataInsightId:          databaseInsightStringPointer(resource.Spec.ExadataInsightId),
		OpsiPrivateEndpointId:     databaseInsightStringPointer(resource.Spec.OpsiPrivateEndpointId),
		CompartmentIdInSubtree:    common.Bool(false),
	}
	if value := strings.TrimSpace(resource.Spec.DatabaseId); value != "" {
		request.DatabaseId = []string{value}
	}
	return request
}

func selectDatabaseInsightSummary(
	items []opsisdk.DatabaseInsightSummary,
	resource *opsiv1beta1.DatabaseInsight,
	includeDeleteStates bool,
) (opsisdk.DatabaseInsightSummary, bool, error) {
	matches := make([]opsisdk.DatabaseInsightSummary, 0, len(items))
	for _, item := range items {
		if item == nil || (!includeDeleteStates && databaseInsightSummaryDeleted(item)) {
			continue
		}
		if databaseInsightSummaryMatchesSpec(item, resource) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return nil, false, fmt.Errorf("DatabaseInsight list response returned multiple matching resources")
	}
}

func databaseInsightSummaryMatchesSpec(item opsisdk.DatabaseInsightSummary, resource *opsiv1beta1.DatabaseInsight) bool {
	if item == nil || resource == nil {
		return false
	}
	itemRaw := databaseInsightRawMap(item)
	if id := trackedDatabaseInsightID(resource); id != "" && id == strings.TrimSpace(stringValueFromMap(itemRaw, "id")) {
		return true
	}
	desired, err := databaseInsightDesiredBodyMap(resource.Spec)
	if err != nil {
		return false
	}
	compared := false
	for _, field := range []string{
		"compartmentId",
		"entitySource",
		"databaseId",
		"enterpriseManagerBridgeId",
		"enterpriseManagerIdentifier",
		"enterpriseManagerEntityIdentifier",
		"exadataInsightId",
		"opsiPrivateEndpointId",
		"databaseConnectorId",
	} {
		desiredValue, desiredOK := meaningfulDatabaseInsightMapValue(desired, field)
		if !desiredOK {
			continue
		}
		actualValue, actualOK := meaningfulDatabaseInsightMapValue(itemRaw, field)
		if !actualOK {
			continue
		}
		compared = true
		if !reflect.DeepEqual(normalizedJSONValue(desiredValue), normalizedJSONValue(actualValue)) {
			return false
		}
	}
	return compared
}

func databaseInsightSummaryDeleted(item opsisdk.DatabaseInsightSummary) bool {
	return strings.EqualFold(string(item.GetLifecycleState()), string(opsisdk.LifecycleStateDeleted))
}

func confirmDatabaseInsightDeleteRead(
	ctx context.Context,
	hooks *DatabaseInsightRuntimeHooks,
	resource *opsiv1beta1.DatabaseInsight,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm DatabaseInsight delete: runtime hooks are nil")
	}
	if currentID = strings.TrimSpace(currentID); currentID != "" {
		if hooks.Get.Call == nil {
			return nil, fmt.Errorf("confirm DatabaseInsight delete: get hook is not configured")
		}
		response, err := hooks.Get.Call(ctx, opsisdk.GetDatabaseInsightRequest{
			DatabaseInsightId: databaseInsightStringPointer(currentID),
		})
		return databaseInsightDeleteReadResponse(response, err)
	}
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm DatabaseInsight delete: list hook is not configured")
	}
	response, err := hooks.List.Call(ctx, databaseInsightListRequest(resource))
	if err != nil {
		return nil, err
	}
	summary, found, err := selectDatabaseInsightSummary(response.Items, resource, true)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, databaseInsightNotFoundError{message: "DatabaseInsight delete confirmation did not find a matching OCI resource"}
	}
	return summary, nil
}

func databaseInsightDeleteReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil, databaseInsightNotFoundError{message: err.Error()}
	}
	return nil, err
}

func handleDatabaseInsightDeleteError(resource *opsiv1beta1.DatabaseInsight, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(databaseInsightNotFoundError); ok {
		return errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			Description:    err.Error(),
		}
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return databaseInsightAmbiguousNotFoundError{
		message:      "DatabaseInsight delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapDatabaseInsightDeleteConfirmation(hooks *DatabaseInsightRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getDatabaseInsight := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DatabaseInsightServiceClient) DatabaseInsightServiceClient {
		return databaseInsightDeleteConfirmationClient{
			delegate:           delegate,
			getDatabaseInsight: getDatabaseInsight,
		}
	})
}

type databaseInsightDeleteConfirmationClient struct {
	delegate           DatabaseInsightServiceClient
	getDatabaseInsight func(context.Context, opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error)
}

func (c databaseInsightDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c databaseInsightDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c databaseInsightDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
) error {
	if c.getDatabaseInsight == nil || resource == nil {
		return nil
	}
	databaseInsightID := trackedDatabaseInsightID(resource)
	if databaseInsightID == "" {
		return nil
	}
	_, err := c.getDatabaseInsight(ctx, opsisdk.GetDatabaseInsightRequest{DatabaseInsightId: databaseInsightStringPointer(databaseInsightID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("DatabaseInsight delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func applyDatabaseInsightDeleteOutcome(
	resource *opsiv1beta1.DatabaseInsight,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := strings.ToUpper(databaseInsightLifecycleState(response))
	if databaseInsightDeleteConfirmedState(state) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if databaseInsightShouldGuardPendingWriteDelete(resource, state, stage) {
		markDatabaseInsightPendingWriteDeleteGuard(resource, response, state)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	if !databaseInsightShouldMarkTerminating(resource, state, stage) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markDatabaseInsightTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func databaseInsightDeleteConfirmedState(state string) bool {
	return state == "" || state == string(opsisdk.LifecycleStateDeleted)
}

func databaseInsightShouldGuardPendingWriteDelete(
	resource *opsiv1beta1.DatabaseInsight,
	state string,
	stage generatedruntime.DeleteConfirmStage,
) bool {
	return stage == generatedruntime.DeleteConfirmStageAlreadyPending &&
		databaseInsightPendingWriteState(state) &&
		!databaseInsightDeleteAlreadyPending(resource)
}

func databaseInsightShouldMarkTerminating(
	resource *opsiv1beta1.DatabaseInsight,
	state string,
	stage generatedruntime.DeleteConfirmStage,
) bool {
	if !databaseInsightRetainFinalizerState(state) {
		return false
	}
	return stage != generatedruntime.DeleteConfirmStageAlreadyPending ||
		databaseInsightDeleteAlreadyPending(resource)
}

func databaseInsightRetainFinalizerState(state string) bool {
	return state == string(opsisdk.LifecycleStateActive) ||
		state == string(opsisdk.LifecycleStateUpdating) ||
		state == string(opsisdk.LifecycleStateCreating)
}

func databaseInsightDeleteAlreadyPending(resource *opsiv1beta1.DatabaseInsight) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func databaseInsightPendingWriteState(state string) bool {
	return state == string(opsisdk.LifecycleStateCreating) || state == string(opsisdk.LifecycleStateUpdating)
}

func markDatabaseInsightPendingWriteDeleteGuard(
	resource *opsiv1beta1.DatabaseInsight,
	response any,
	state string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	phase := shared.OSOKAsyncPhaseUpdate
	if state == string(opsisdk.LifecycleStateCreating) {
		phase = shared.OSOKAsyncPhaseCreate
	}
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = databaseInsightPendingWriteDeleteMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       databaseInsightLifecycleState(response),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         databaseInsightPendingWriteDeleteMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		databaseInsightPendingWriteDeleteMessage,
		loggerutil.OSOKLogger{},
	)
}

func markDatabaseInsightTerminating(resource *opsiv1beta1.DatabaseInsight, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = databaseInsightDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := databaseInsightLifecycleState(response)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         databaseInsightDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		databaseInsightDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func trackedDatabaseInsightID(resource *opsiv1beta1.DatabaseInsight) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func wrapDatabaseInsightRequestBodyContext(hooks *DatabaseInsightRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DatabaseInsightServiceClient) DatabaseInsightServiceClient {
		return databaseInsightRequestBodyContextClient{delegate: delegate}
	})
}

type databaseInsightRequestBodyContextClient struct {
	delegate DatabaseInsightServiceClient
}

func (c databaseInsightRequestBodyContextClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	ctx = withDatabaseInsightRequestBodyToken(ctx)
	defer clearDatabaseInsightRequestBodies(ctx)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c databaseInsightRequestBodyContextClient) Delete(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func withDatabaseInsightRequestBodyToken(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if databaseInsightRequestBodyToken(ctx) != "" {
		return ctx
	}
	token := fmt.Sprintf("databaseinsight-request-%d", databaseInsightRequestBodySequence.Add(1))
	return context.WithValue(ctx, databaseInsightRequestBodyContextKey{}, token)
}

func databaseInsightRequestBodyToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(databaseInsightRequestBodyContextKey{}).(string)
	return strings.TrimSpace(value)
}

func clearDatabaseInsightRequestBodies(ctx context.Context) {
	key := databaseInsightRequestBodyToken(ctx)
	if key == "" {
		return
	}
	pendingDatabaseInsightCreateBodies.Delete(key)
	pendingDatabaseInsightUpdateBodies.Delete(key)
}

func wrapDatabaseInsightRequestBodies(hooks *DatabaseInsightRuntimeHooks) {
	if hooks == nil {
		return
	}
	wrapDatabaseInsightCreateRequestBody(hooks)
	wrapDatabaseInsightUpdateRequestBody(hooks)
}

func wrapDatabaseInsightCreateRequestBody(hooks *DatabaseInsightRuntimeHooks) {
	call := hooks.Create.Call
	if call == nil {
		return
	}
	hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
		if request.CreateDatabaseInsightDetails == nil {
			body, err := takeDatabaseInsightCreateBody(ctx, request.OpcRetryToken)
			if err != nil {
				return opsisdk.CreateDatabaseInsightResponse{}, err
			}
			request.CreateDatabaseInsightDetails = body
		}
		return call(ctx, request)
	}
}

func wrapDatabaseInsightUpdateRequestBody(hooks *DatabaseInsightRuntimeHooks) {
	call := hooks.Update.Call
	if call == nil {
		return
	}
	hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
		if request.UpdateDatabaseInsightDetails == nil {
			body, err := takeDatabaseInsightUpdateBody(ctx, request.DatabaseInsightId)
			if err != nil {
				return opsisdk.UpdateDatabaseInsightResponse{}, err
			}
			request.UpdateDatabaseInsightDetails = body
		}
		return call(ctx, request)
	}
}

func stashDatabaseInsightCreateBody(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
	body opsisdk.CreateDatabaseInsightDetails,
) error {
	key := databaseInsightRequestBodyToken(ctx)
	if key == "" {
		key = databaseInsightResourceRetryToken(resource)
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("DatabaseInsight create body cannot be keyed without resource namespace/name or uid")
	}
	pendingDatabaseInsightCreateBodies.Store(key, body)
	return nil
}

func takeDatabaseInsightCreateBody(ctx context.Context, retryToken *string) (opsisdk.CreateDatabaseInsightDetails, error) {
	key := databaseInsightRequestBodyToken(ctx)
	if key == "" {
		key = strings.TrimSpace(databaseInsightStringValue(retryToken))
	}
	if key == "" {
		return nil, fmt.Errorf("DatabaseInsight create body is missing a retry-token key")
	}
	value, ok := pendingDatabaseInsightCreateBodies.LoadAndDelete(key)
	if !ok {
		return nil, fmt.Errorf("DatabaseInsight create body was not prepared for retry-token %q", key)
	}
	body, ok := value.(opsisdk.CreateDatabaseInsightDetails)
	if !ok {
		return nil, fmt.Errorf("prepared DatabaseInsight create body has unexpected type %T", value)
	}
	return body, nil
}

func stashDatabaseInsightUpdateBody(
	ctx context.Context,
	resource *opsiv1beta1.DatabaseInsight,
	current map[string]any,
	body opsisdk.UpdateDatabaseInsightDetails,
) error {
	key := databaseInsightRequestBodyToken(ctx)
	if key == "" {
		key = databaseInsightUpdateBodyKey(resource, current)
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("DatabaseInsight update body cannot be keyed without a resource OCID")
	}
	pendingDatabaseInsightUpdateBodies.Store(key, body)
	return nil
}

func takeDatabaseInsightUpdateBody(ctx context.Context, databaseInsightID *string) (opsisdk.UpdateDatabaseInsightDetails, error) {
	key := databaseInsightRequestBodyToken(ctx)
	if key == "" {
		key = strings.TrimSpace(databaseInsightStringValue(databaseInsightID))
	}
	if key == "" {
		return nil, fmt.Errorf("DatabaseInsight update body is missing a resource OCID key")
	}
	value, ok := pendingDatabaseInsightUpdateBodies.LoadAndDelete(key)
	if !ok {
		return nil, fmt.Errorf("DatabaseInsight update body was not prepared for %q", key)
	}
	body, ok := value.(opsisdk.UpdateDatabaseInsightDetails)
	if !ok {
		return nil, fmt.Errorf("prepared DatabaseInsight update body has unexpected type %T", value)
	}
	return body, nil
}

func databaseInsightUpdateBodyKey(resource *opsiv1beta1.DatabaseInsight, current map[string]any) string {
	if id := strings.TrimSpace(stringValueFromMap(current, "id")); id != "" {
		return id
	}
	return trackedDatabaseInsightID(resource)
}

func databaseInsightResourceRetryToken(resource *opsiv1beta1.DatabaseInsight) string {
	if resource == nil {
		return ""
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return uid
	}
	namespace := strings.TrimSpace(resource.Namespace)
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func wrapDatabaseInsightResponses(hooks *DatabaseInsightRuntimeHooks) {
	if hooks == nil {
		return
	}
	if call := hooks.Create.Call; call != nil {
		hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			response.DatabaseInsight = newDatabaseInsightBody(response.DatabaseInsight)
			return response, nil
		}
	}
	if call := hooks.Get.Call; call != nil {
		hooks.Get.Call = func(ctx context.Context, request opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			response.DatabaseInsight = newDatabaseInsightBody(response.DatabaseInsight)
			return response, nil
		}
	}
	if call := hooks.List.Call; call != nil {
		hooks.List.Call = listDatabaseInsightsAllPages(call)
	}
}

func listDatabaseInsightsAllPages(call databaseInsightListCall) databaseInsightListCall {
	return func(ctx context.Context, request opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
		if call == nil {
			return opsisdk.ListDatabaseInsightsResponse{}, fmt.Errorf("DatabaseInsight list operation is not configured")
		}
		seenPages := map[string]struct{}{}
		var combined opsisdk.ListDatabaseInsightsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return opsisdk.ListDatabaseInsightsResponse{}, err
			}
			appendDatabaseInsightListPage(&combined, response)
			nextPage := strings.TrimSpace(databaseInsightStringValue(response.OpcNextPage))
			if nextPage == "" {
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return opsisdk.ListDatabaseInsightsResponse{}, fmt.Errorf("DatabaseInsight list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = databaseInsightStringPointer(nextPage)
		}
	}
}

func appendDatabaseInsightListPage(
	combined *opsisdk.ListDatabaseInsightsResponse,
	response opsisdk.ListDatabaseInsightsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	for _, item := range response.Items {
		combined.Items = append(combined.Items, newDatabaseInsightSummaryBody(item))
	}
}

type databaseInsightBody struct {
	raw map[string]any
}

func newDatabaseInsightBody(body opsisdk.DatabaseInsight) opsisdk.DatabaseInsight {
	if body == nil {
		return nil
	}
	return databaseInsightBody{raw: databaseInsightRawMap(body)}
}

func (b databaseInsightBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(databaseInsightStatusSafeRaw(b.raw))
}

func (b databaseInsightBody) GetId() *string {
	return databaseInsightRawStringPointer(b.raw, "id")
}

func (b databaseInsightBody) GetCompartmentId() *string {
	return databaseInsightRawStringPointer(b.raw, "compartmentId")
}

func (b databaseInsightBody) GetStatus() opsisdk.ResourceStatusEnum {
	return opsisdk.ResourceStatusEnum(stringValueFromMap(b.raw, "status"))
}

func (b databaseInsightBody) GetFreeformTags() map[string]string {
	return databaseInsightRawStringMap(b.raw, "freeformTags")
}

func (b databaseInsightBody) GetDefinedTags() map[string]map[string]interface{} {
	return databaseInsightRawNestedMap(b.raw, "definedTags")
}

func (b databaseInsightBody) GetTimeCreated() *common.SDKTime {
	return nil
}

func (b databaseInsightBody) GetLifecycleState() opsisdk.LifecycleStateEnum {
	return opsisdk.LifecycleStateEnum(stringValueFromMap(b.raw, "lifecycleState"))
}

func (b databaseInsightBody) GetDatabaseType() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseType")
}

func (b databaseInsightBody) GetDatabaseVersion() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseVersion")
}

func (b databaseInsightBody) GetProcessorCount() *int {
	return databaseInsightRawIntPointer(b.raw, "processorCount")
}

func (b databaseInsightBody) GetSystemTags() map[string]map[string]interface{} {
	return databaseInsightRawNestedMap(b.raw, "systemTags")
}

func (b databaseInsightBody) GetTimeUpdated() *common.SDKTime {
	return nil
}

func (b databaseInsightBody) GetLifecycleDetails() *string {
	return databaseInsightRawStringPointer(b.raw, "lifecycleDetails")
}

func (b databaseInsightBody) GetDatabaseConnectionStatusDetails() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseConnectionStatusDetails")
}

type databaseInsightSummaryBody struct {
	raw map[string]any
}

func newDatabaseInsightSummaryBody(body opsisdk.DatabaseInsightSummary) opsisdk.DatabaseInsightSummary {
	if body == nil {
		return nil
	}
	return databaseInsightSummaryBody{raw: databaseInsightRawMap(body)}
}

func (b databaseInsightSummaryBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(databaseInsightStatusSafeRaw(b.raw))
}

func (b databaseInsightSummaryBody) GetId() *string {
	return databaseInsightRawStringPointer(b.raw, "id")
}

func (b databaseInsightSummaryBody) GetDatabaseId() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseId")
}

func (b databaseInsightSummaryBody) GetCompartmentId() *string {
	return databaseInsightRawStringPointer(b.raw, "compartmentId")
}

func (b databaseInsightSummaryBody) GetDatabaseName() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseName")
}

func (b databaseInsightSummaryBody) GetDatabaseDisplayName() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseDisplayName")
}

func (b databaseInsightSummaryBody) GetDatabaseType() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseType")
}

func (b databaseInsightSummaryBody) GetDatabaseVersion() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseVersion")
}

func (b databaseInsightSummaryBody) GetDatabaseHostNames() []string {
	return databaseInsightRawStringSlice(b.raw, "databaseHostNames")
}

func (b databaseInsightSummaryBody) GetFreeformTags() map[string]string {
	return databaseInsightRawStringMap(b.raw, "freeformTags")
}

func (b databaseInsightSummaryBody) GetDefinedTags() map[string]map[string]interface{} {
	return databaseInsightRawNestedMap(b.raw, "definedTags")
}

func (b databaseInsightSummaryBody) GetSystemTags() map[string]map[string]interface{} {
	return databaseInsightRawNestedMap(b.raw, "systemTags")
}

func (b databaseInsightSummaryBody) GetProcessorCount() *int {
	return databaseInsightRawIntPointer(b.raw, "processorCount")
}

func (b databaseInsightSummaryBody) GetStatus() opsisdk.ResourceStatusEnum {
	return opsisdk.ResourceStatusEnum(stringValueFromMap(b.raw, "status"))
}

func (b databaseInsightSummaryBody) GetTimeCreated() *common.SDKTime {
	return nil
}

func (b databaseInsightSummaryBody) GetTimeUpdated() *common.SDKTime {
	return nil
}

func (b databaseInsightSummaryBody) GetLifecycleState() opsisdk.LifecycleStateEnum {
	return opsisdk.LifecycleStateEnum(stringValueFromMap(b.raw, "lifecycleState"))
}

func (b databaseInsightSummaryBody) GetLifecycleDetails() *string {
	return databaseInsightRawStringPointer(b.raw, "lifecycleDetails")
}

func (b databaseInsightSummaryBody) GetDatabaseConnectionStatusDetails() *string {
	return databaseInsightRawStringPointer(b.raw, "databaseConnectionStatusDetails")
}

func databaseInsightStatusSafeRaw(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	clone := make(map[string]any, len(source)+1)
	for key, value := range source {
		if key == "status" {
			clone["sdkStatus"] = value
			continue
		}
		clone[key] = value
	}
	return clone
}

func databaseInsightRawFromResponse(response any) map[string]any {
	switch typed := response.(type) {
	case opsisdk.CreateDatabaseInsightResponse:
		return databaseInsightRawMap(typed.DatabaseInsight)
	case *opsisdk.CreateDatabaseInsightResponse:
		if typed == nil {
			return nil
		}
		return databaseInsightRawMap(typed.DatabaseInsight)
	case opsisdk.GetDatabaseInsightResponse:
		return databaseInsightRawMap(typed.DatabaseInsight)
	case *opsisdk.GetDatabaseInsightResponse:
		if typed == nil {
			return nil
		}
		return databaseInsightRawMap(typed.DatabaseInsight)
	case opsisdk.DatabaseInsightSummary:
		return databaseInsightRawMap(typed)
	case opsisdk.DatabaseInsight:
		return databaseInsightRawMap(typed)
	default:
		return databaseInsightRawMap(response)
	}
}

func databaseInsightLifecycleState(response any) string {
	return strings.ToUpper(strings.TrimSpace(stringValueFromMap(databaseInsightRawFromResponse(response), "lifecycleState")))
}

func databaseInsightRawMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if body, ok := value.(databaseInsightBody); ok {
		return cloneDatabaseInsightMap(body.raw)
	}
	if body, ok := value.(databaseInsightSummaryBody); ok {
		return cloneDatabaseInsightMap(body.raw)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func databaseInsightJSONDataMap(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("decode DatabaseInsight jsonData: %w", err)
	}
	return decoded, nil
}

func databaseInsightDesiredHasField(spec opsiv1beta1.DatabaseInsightSpec, desired map[string]any, key string) bool {
	switch key {
	case "freeformTags":
		return spec.FreeformTags != nil || mapHasKey(spec.JsonData, key, desired)
	case "definedTags":
		return spec.DefinedTags != nil || mapHasKey(spec.JsonData, key, desired)
	default:
		return mapHasKey(spec.JsonData, key, desired)
	}
}

func mapHasKey(_ string, key string, values map[string]any) bool {
	_, ok := values[key]
	return ok
}

func meaningfulDatabaseInsightMapValue(values map[string]any, key string) (any, bool) {
	if values == nil {
		return nil, false
	}
	value, ok := values[key]
	if ok && key == "isAdvancedFeaturesEnabled" {
		if _, isBool := value.(bool); isBool {
			return value, true
		}
	}
	if !ok || !meaningfulDatabaseInsightValue(value) {
		return nil, false
	}
	return value, true
}

func meaningfulDatabaseInsightValue(value any) bool {
	switch concrete := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(concrete) != ""
	case []byte:
		return len(concrete) > 0
	case []any:
		return len(concrete) > 0
	case map[string]any:
		for _, child := range concrete {
			if meaningfulDatabaseInsightValue(child) {
				return true
			}
		}
		return false
	default:
		return !reflect.ValueOf(value).IsZero()
	}
}

func normalizedJSONValue(value any) any {
	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return value
	}
	return decoded
}

func firstDatabaseInsightString(primary map[string]any, secondary map[string]any, key string) string {
	if value := strings.TrimSpace(stringValueFromMap(primary, key)); value != "" {
		return value
	}
	return strings.TrimSpace(stringValueFromMap(secondary, key))
}

func databaseInsightStringMapValue(defaults map[string]any, body map[string]any, key string) string {
	if value := strings.TrimSpace(stringValueFromMap(defaults, key)); value != "" {
		return value
	}
	return strings.TrimSpace(stringValueFromMap(body, key))
}

func stringValueFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func databaseInsightRawStringPointer(values map[string]any, key string) *string {
	value := strings.TrimSpace(stringValueFromMap(values, key))
	if value == "" {
		return nil
	}
	return common.String(value)
}

func databaseInsightRawIntPointer(values map[string]any, key string) *int {
	switch value := values[key].(type) {
	case int:
		return common.Int(value)
	case float64:
		intValue := int(value)
		return &intValue
	default:
		return nil
	}
}

func databaseInsightRawStringMap(values map[string]any, key string) map[string]string {
	raw, ok := values[key].(map[string]any)
	if !ok {
		return nil
	}
	converted := make(map[string]string, len(raw))
	for k, v := range raw {
		converted[k] = fmt.Sprint(v)
	}
	return converted
}

func databaseInsightRawNestedMap(values map[string]any, key string) map[string]map[string]interface{} {
	raw, ok := values[key].(map[string]any)
	if !ok {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(raw))
	for namespace, values := range raw {
		child, ok := values.(map[string]any)
		if !ok {
			continue
		}
		converted[namespace] = make(map[string]interface{}, len(child))
		for key, value := range child {
			converted[namespace][key] = value
		}
	}
	return converted
}

func databaseInsightRawStringSlice(values map[string]any, key string) []string {
	raw, ok := values[key].([]any)
	if !ok {
		return nil
	}
	converted := make([]string, 0, len(raw))
	for _, value := range raw {
		if stringValue := strings.TrimSpace(fmt.Sprint(value)); stringValue != "" {
			converted = append(converted, stringValue)
		}
	}
	return converted
}

func databaseInsightStringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func databaseInsightIntPointer(value int) *int {
	if value == 0 {
		return nil
	}
	return common.Int(value)
}

func databaseInsightStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneDatabaseInsightStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func databaseInsightDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func cloneDatabaseInsightMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
