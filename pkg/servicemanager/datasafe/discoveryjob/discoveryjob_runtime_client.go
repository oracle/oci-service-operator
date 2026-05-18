/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package discoveryjob

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
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

const discoveryJobKind = "DiscoveryJob"

type discoveryJobOCIClient interface {
	CreateDiscoveryJob(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error)
	GetDiscoveryJob(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error)
	ListDiscoveryJobs(context.Context, datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error)
	DeleteDiscoveryJob(context.Context, datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error)
}

type discoveryJobIdentity struct {
	compartmentID        string
	sensitiveDataModelID string
	displayName          string
	discoveryType        string
}

type discoveryJobRuntimeClient struct {
	delegate DiscoveryJobServiceClient
	get      func(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error)
	list     func(context.Context, datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error)
}

type discoveryJobAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e discoveryJobAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e discoveryJobAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerDiscoveryJobRuntimeHooksMutator(func(_ *DiscoveryJobServiceManager, hooks *DiscoveryJobRuntimeHooks) {
		applyDiscoveryJobRuntimeHooks(hooks)
	})
}

func applyDiscoveryJobRuntimeHooks(hooks *DiscoveryJobRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = discoveryJobRuntimeSemantics()
	hooks.BuildCreateBody = buildDiscoveryJobCreateBody
	hooks.Identity.Resolve = resolveDiscoveryJobIdentity
	hooks.Identity.RecordPath = recordDiscoveryJobPathIdentity
	hooks.Create.Fields = discoveryJobCreateFields()
	hooks.Get.Fields = discoveryJobGetFields()
	hooks.List.Fields = discoveryJobListFields()
	hooks.Delete.Fields = discoveryJobDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDiscoveryJobsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleDiscoveryJobDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDiscoveryJobDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectDiscoveryJobStatus
	hooks.StatusHooks.MarkTerminating = markDiscoveryJobTerminating
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DiscoveryJobServiceClient) DiscoveryJobServiceClient {
		return discoveryJobRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newDiscoveryJobServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client discoveryJobOCIClient,
) DiscoveryJobServiceClient {
	hooks := newDiscoveryJobRuntimeHooksWithOCIClient(client)
	applyDiscoveryJobRuntimeHooks(&hooks)
	manager := &DiscoveryJobServiceManager{Log: log}
	delegate := defaultDiscoveryJobServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.DiscoveryJob](
			buildDiscoveryJobGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDiscoveryJobGeneratedClient(hooks, delegate)
}

func newDiscoveryJobRuntimeHooksWithOCIClient(client discoveryJobOCIClient) DiscoveryJobRuntimeHooks {
	return DiscoveryJobRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.DiscoveryJob]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.DiscoveryJob]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.DiscoveryJob]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.DiscoveryJob]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.DiscoveryJob]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.DiscoveryJob]{},
		Create: runtimeOperationHooks[datasafesdk.CreateDiscoveryJobRequest, datasafesdk.CreateDiscoveryJobResponse]{
			Fields: discoveryJobCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
				if client == nil {
					return datasafesdk.CreateDiscoveryJobResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.CreateDiscoveryJob(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetDiscoveryJobRequest, datasafesdk.GetDiscoveryJobResponse]{
			Fields: discoveryJobGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
				if client == nil {
					return datasafesdk.GetDiscoveryJobResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.GetDiscoveryJob(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListDiscoveryJobsRequest, datasafesdk.ListDiscoveryJobsResponse]{
			Fields: discoveryJobListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error) {
				if client == nil {
					return datasafesdk.ListDiscoveryJobsResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.ListDiscoveryJobs(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteDiscoveryJobRequest, datasafesdk.DeleteDiscoveryJobResponse]{
			Fields: discoveryJobDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error) {
				if client == nil {
					return datasafesdk.DeleteDiscoveryJobResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.DeleteDiscoveryJob(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DiscoveryJobServiceClient) DiscoveryJobServiceClient{},
	}
}

func discoveryJobRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "discoveryjob",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.DiscoveryLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.DiscoveryLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.DiscoveryLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:        "required",
			PendingStates: []string{string(datasafesdk.DiscoveryLifecycleStateDeleting)},
			TerminalStates: []string{
				string(datasafesdk.DiscoveryLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"sensitiveDataModelId",
				"displayName",
				"discoveryType",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew: []string{
				"sensitiveDataModelId",
				"compartmentId",
				"discoveryType",
				"displayName",
				"schemasForDiscovery",
				"tablesForDiscovery",
				"sensitiveTypeIdsForDiscovery",
				"sensitiveTypeGroupIdsForDiscovery",
				"isSampleDataCollectionEnabled",
				"isAppDefinedRelationDiscoveryEnabled",
				"isIncludeAllSchemas",
				"isIncludeAllSensitiveTypes",
				"freeformTags",
				"definedTags",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: discoveryJobKind, Action: "CreateDiscoveryJob"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: discoveryJobKind, Action: "DeleteDiscoveryJob"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func discoveryJobCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDiscoveryJobDetails", RequestName: "CreateDiscoveryJobDetails", Contribution: "body"},
	}
}

func discoveryJobGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DiscoveryJobId", RequestName: "discoveryJobId", Contribution: "path", PreferResourceID: true},
	}
}

func discoveryJobListFields() []generatedruntime.RequestField {
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
		{
			FieldName:    "SensitiveDataModelId",
			RequestName:  "sensitiveDataModelId",
			Contribution: "query",
			LookupPaths:  []string{"status.sensitiveDataModelId", "spec.sensitiveDataModelId", "sensitiveDataModelId"},
		},
		{FieldName: "DiscoveryJobId", RequestName: "discoveryJobId", Contribution: "query", PreferResourceID: true},
		{FieldName: "TargetId", RequestName: "targetId", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func discoveryJobDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DiscoveryJobId", RequestName: "discoveryJobId", Contribution: "path", PreferResourceID: true},
	}
}

func buildDiscoveryJobCreateBody(_ context.Context, resource *datasafev1beta1.DiscoveryJob, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", discoveryJobKind)
	}
	return datasafesdk.CreateDiscoveryJobDetails{
		SensitiveDataModelId:                 common.String(strings.TrimSpace(resource.Spec.SensitiveDataModelId)),
		CompartmentId:                        common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DiscoveryType:                        datasafesdk.DiscoveryJobDiscoveryTypeEnum(strings.TrimSpace(resource.Spec.DiscoveryType)),
		DisplayName:                          discoveryJobOptionalString(resource.Spec.DisplayName),
		SchemasForDiscovery:                  discoveryJobStringSlice(resource.Spec.SchemasForDiscovery),
		TablesForDiscovery:                   discoveryJobTablesForDiscoveryFromSpec(resource.Spec.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         discoveryJobStringSlice(resource.Spec.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    discoveryJobStringSlice(resource.Spec.SensitiveTypeGroupIdsForDiscovery),
		IsSampleDataCollectionEnabled:        common.Bool(resource.Spec.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: common.Bool(resource.Spec.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  common.Bool(resource.Spec.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           common.Bool(resource.Spec.IsIncludeAllSensitiveTypes),
		FreeformTags:                         discoveryJobStringMap(resource.Spec.FreeformTags),
		DefinedTags:                          discoveryJobDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func discoveryJobTablesForDiscoveryFromSpec(spec []datasafev1beta1.DiscoveryJobTablesForDiscovery) []datasafesdk.TablesForDiscovery {
	if len(spec) == 0 {
		return nil
	}
	tables := make([]datasafesdk.TablesForDiscovery, 0, len(spec))
	for _, table := range spec {
		tables = append(tables, datasafesdk.TablesForDiscovery{
			SchemaName: discoveryJobOptionalString(table.SchemaName),
			TableNames: discoveryJobStringSlice(table.TableNames),
		})
	}
	return tables
}

func resolveDiscoveryJobIdentity(resource *datasafev1beta1.DiscoveryJob) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", discoveryJobKind)
	}
	return discoveryJobIdentityFromResource(resource), nil
}

func discoveryJobIdentityFromResource(resource *datasafev1beta1.DiscoveryJob) discoveryJobIdentity {
	if resource == nil {
		return discoveryJobIdentity{}
	}
	return discoveryJobIdentity{
		compartmentID:        strings.TrimSpace(resource.Spec.CompartmentId),
		sensitiveDataModelID: strings.TrimSpace(resource.Spec.SensitiveDataModelId),
		displayName:          strings.TrimSpace(resource.Spec.DisplayName),
		discoveryType:        strings.TrimSpace(resource.Spec.DiscoveryType),
	}
}

func recordDiscoveryJobPathIdentity(resource *datasafev1beta1.DiscoveryJob, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(discoveryJobIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if typed.sensitiveDataModelID != "" {
		resource.Status.SensitiveDataModelId = typed.sensitiveDataModelID
	}
	if typed.displayName != "" {
		resource.Status.DisplayName = typed.displayName
	}
	if typed.discoveryType != "" {
		resource.Status.DiscoveryType = typed.discoveryType
	}
}

func listDiscoveryJobsAllPages(
	call func(context.Context, datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error),
) func(context.Context, datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error) {
		var combined datasafesdk.ListDiscoveryJobsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleDiscoveryJobDeleteError(resource *datasafev1beta1.DiscoveryJob, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := discoveryJobAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func applyDiscoveryJobDeleteOutcome(
	resource *datasafev1beta1.DiscoveryJob,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := discoveryJobLifecycleState(response)
	if state == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if discoveryJobTerminalDeleteState(state) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markDiscoveryJobTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markDiscoveryJobTerminating(resource *datasafev1beta1.DiscoveryJob, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = "OCI resource delete is in progress"
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         status.Message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, loggerutil.OSOKLogger{})
}

func discoveryJobLifecycleState(response any) string {
	if current, ok := discoveryJobFromResponse(response); ok {
		return strings.ToUpper(string(current.LifecycleState))
	}
	if summary, ok := discoveryJobSummaryFromResponse(response); ok {
		return strings.ToUpper(string(summary.LifecycleState))
	}
	return ""
}

func discoveryJobTerminalDeleteState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), string(datasafesdk.DiscoveryLifecycleStateDeleted))
}

func projectDiscoveryJobStatus(resource *datasafev1beta1.DiscoveryJob, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", discoveryJobKind)
	}
	projected, ok := discoveryJobStatusProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.DiscoveryJobStatus{
		OsokStatus:                           osokStatus,
		Id:                                   projected.Id,
		DiscoveryType:                        projected.DiscoveryType,
		DisplayName:                          projected.DisplayName,
		CompartmentId:                        projected.CompartmentId,
		TimeStarted:                          projected.TimeStarted,
		TimeFinished:                         projected.TimeFinished,
		LifecycleState:                       projected.LifecycleState,
		SensitiveDataModelId:                 projected.SensitiveDataModelId,
		TargetId:                             projected.TargetId,
		IsSampleDataCollectionEnabled:        projected.IsSampleDataCollectionEnabled,
		IsAppDefinedRelationDiscoveryEnabled: projected.IsAppDefinedRelationDiscoveryEnabled,
		IsIncludeAllSchemas:                  projected.IsIncludeAllSchemas,
		IsIncludeAllSensitiveTypes:           projected.IsIncludeAllSensitiveTypes,
		TotalSchemasScanned:                  projected.TotalSchemasScanned,
		TotalObjectsScanned:                  projected.TotalObjectsScanned,
		TotalColumnsScanned:                  projected.TotalColumnsScanned,
		TotalNewSensitiveColumns:             projected.TotalNewSensitiveColumns,
		TotalModifiedSensitiveColumns:        projected.TotalModifiedSensitiveColumns,
		TotalDeletedSensitiveColumns:         projected.TotalDeletedSensitiveColumns,
		SchemasForDiscovery:                  discoveryJobStringSlice(projected.SchemasForDiscovery),
		TablesForDiscovery:                   discoveryJobAPITablesForDiscovery(projected.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         discoveryJobStringSlice(projected.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    discoveryJobStringSlice(projected.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         discoveryJobStringMap(projected.FreeformTags),
		DefinedTags:                          discoveryJobCloneSharedTags(projected.DefinedTags),
		SystemTags:                           discoveryJobCloneSharedTags(projected.SystemTags),
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

type discoveryJobStatusProjection struct {
	Id                                   string
	DiscoveryType                        string
	DisplayName                          string
	CompartmentId                        string
	TimeStarted                          string
	TimeFinished                         string
	LifecycleState                       string
	SensitiveDataModelId                 string
	TargetId                             string
	IsSampleDataCollectionEnabled        bool
	IsAppDefinedRelationDiscoveryEnabled bool
	IsIncludeAllSchemas                  bool
	IsIncludeAllSensitiveTypes           bool
	TotalSchemasScanned                  int64
	TotalObjectsScanned                  int64
	TotalColumnsScanned                  int64
	TotalNewSensitiveColumns             int64
	TotalModifiedSensitiveColumns        int64
	TotalDeletedSensitiveColumns         int64
	SchemasForDiscovery                  []string
	TablesForDiscovery                   []datasafesdk.TablesForDiscovery
	SensitiveTypeIdsForDiscovery         []string
	SensitiveTypeGroupIdsForDiscovery    []string
	FreeformTags                         map[string]string
	DefinedTags                          map[string]shared.MapValue
	SystemTags                           map[string]shared.MapValue
}

func discoveryJobStatusProjectionFromResponse(response any) (discoveryJobStatusProjection, bool) {
	if current, ok := discoveryJobFromResponse(response); ok {
		return discoveryJobStatusProjectionFromSDK(current), true
	}
	if summary, ok := discoveryJobSummaryFromResponse(response); ok {
		return discoveryJobStatusProjectionFromSummary(summary), true
	}
	return discoveryJobStatusProjection{}, false
}

func discoveryJobFromResponse(response any) (datasafesdk.DiscoveryJob, bool) {
	switch current := response.(type) {
	case datasafesdk.GetDiscoveryJobResponse:
		return current.DiscoveryJob, true
	case *datasafesdk.GetDiscoveryJobResponse:
		if current == nil {
			return datasafesdk.DiscoveryJob{}, false
		}
		return current.DiscoveryJob, true
	case datasafesdk.CreateDiscoveryJobResponse:
		return current.DiscoveryJob, true
	case *datasafesdk.CreateDiscoveryJobResponse:
		if current == nil {
			return datasafesdk.DiscoveryJob{}, false
		}
		return current.DiscoveryJob, true
	case datasafesdk.DiscoveryJob:
		return current, true
	case *datasafesdk.DiscoveryJob:
		if current == nil {
			return datasafesdk.DiscoveryJob{}, false
		}
		return *current, true
	default:
		return datasafesdk.DiscoveryJob{}, false
	}
}

func discoveryJobSummaryFromResponse(response any) (datasafesdk.DiscoveryJobSummary, bool) {
	switch current := response.(type) {
	case datasafesdk.DiscoveryJobSummary:
		return current, true
	case *datasafesdk.DiscoveryJobSummary:
		if current == nil {
			return datasafesdk.DiscoveryJobSummary{}, false
		}
		return *current, true
	default:
		return datasafesdk.DiscoveryJobSummary{}, false
	}
}

func discoveryJobStatusProjectionFromSDK(current datasafesdk.DiscoveryJob) discoveryJobStatusProjection {
	return discoveryJobStatusProjection{
		Id:                                   discoveryJobStringValue(current.Id),
		DiscoveryType:                        string(current.DiscoveryType),
		DisplayName:                          discoveryJobStringValue(current.DisplayName),
		CompartmentId:                        discoveryJobStringValue(current.CompartmentId),
		TimeStarted:                          discoveryJobSDKTimeString(current.TimeStarted),
		TimeFinished:                         discoveryJobSDKTimeString(current.TimeFinished),
		LifecycleState:                       string(current.LifecycleState),
		SensitiveDataModelId:                 discoveryJobStringValue(current.SensitiveDataModelId),
		TargetId:                             discoveryJobStringValue(current.TargetId),
		IsSampleDataCollectionEnabled:        discoveryJobBoolValue(current.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: discoveryJobBoolValue(current.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  discoveryJobBoolValue(current.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           discoveryJobBoolValue(current.IsIncludeAllSensitiveTypes),
		TotalSchemasScanned:                  discoveryJobInt64Value(current.TotalSchemasScanned),
		TotalObjectsScanned:                  discoveryJobInt64Value(current.TotalObjectsScanned),
		TotalColumnsScanned:                  discoveryJobInt64Value(current.TotalColumnsScanned),
		TotalNewSensitiveColumns:             discoveryJobInt64Value(current.TotalNewSensitiveColumns),
		TotalModifiedSensitiveColumns:        discoveryJobInt64Value(current.TotalModifiedSensitiveColumns),
		TotalDeletedSensitiveColumns:         discoveryJobInt64Value(current.TotalDeletedSensitiveColumns),
		SchemasForDiscovery:                  discoveryJobStringSlice(current.SchemasForDiscovery),
		TablesForDiscovery:                   discoveryJobTablesForDiscovery(current.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         discoveryJobStringSlice(current.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    discoveryJobStringSlice(current.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         discoveryJobStringMap(current.FreeformTags),
		DefinedTags:                          discoveryJobSharedTags(current.DefinedTags),
		SystemTags:                           discoveryJobSharedTags(current.SystemTags),
	}
}

func discoveryJobStatusProjectionFromSummary(current datasafesdk.DiscoveryJobSummary) discoveryJobStatusProjection {
	return discoveryJobStatusProjection{
		Id:                   discoveryJobStringValue(current.Id),
		DiscoveryType:        string(current.DiscoveryType),
		DisplayName:          discoveryJobStringValue(current.DisplayName),
		CompartmentId:        discoveryJobStringValue(current.CompartmentId),
		TimeStarted:          discoveryJobSDKTimeString(current.TimeStarted),
		TimeFinished:         discoveryJobSDKTimeString(current.TimeFinished),
		LifecycleState:       string(current.LifecycleState),
		SensitiveDataModelId: discoveryJobStringValue(current.SensitiveDataModelId),
		TargetId:             discoveryJobStringValue(current.TargetId),
		FreeformTags:         discoveryJobStringMap(current.FreeformTags),
		DefinedTags:          discoveryJobSharedTags(current.DefinedTags),
	}
}

func (c discoveryJobRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.DiscoveryJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DiscoveryJob runtime client is not configured")
	}
	if err := validateDiscoveryJobCreateOrUpdateIdentity(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, markDiscoveryJobFailed(resource, err)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func validateDiscoveryJobCreateOrUpdateIdentity(resource *datasafev1beta1.DiscoveryJob) error {
	if trackedDiscoveryJobID(resource) == "" || resource == nil {
		return nil
	}
	identity := discoveryJobIdentityFromResource(resource)
	checks := []struct {
		field   string
		tracked string
		desired string
	}{
		{field: "compartmentId", tracked: resource.Status.CompartmentId, desired: identity.compartmentID},
		{field: "sensitiveDataModelId", tracked: resource.Status.SensitiveDataModelId, desired: identity.sensitiveDataModelID},
		{field: "displayName", tracked: resource.Status.DisplayName, desired: identity.displayName},
		{field: "discoveryType", tracked: resource.Status.DiscoveryType, desired: identity.discoveryType},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.tracked) == "" || strings.TrimSpace(check.desired) == "" {
			continue
		}
		if strings.TrimSpace(check.tracked) != strings.TrimSpace(check.desired) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", discoveryJobKind, check.field)
		}
	}
	return nil
}

func markDiscoveryJobFailed(resource *datasafev1beta1.DiscoveryJob, err error) error {
	if resource == nil || err == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		status.Async.Current = &current
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
	return err
}

func (c discoveryJobRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.DiscoveryJob) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("DiscoveryJob runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c discoveryJobRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.DiscoveryJob,
) error {
	if resource == nil {
		return nil
	}
	currentID := trackedDiscoveryJobID(resource)
	if currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c discoveryJobRuntimeClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *datasafev1beta1.DiscoveryJob,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, datasafesdk.GetDiscoveryJobRequest{DiscoveryJobId: common.String(currentID)})
	return discoveryJobAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c discoveryJobRuntimeClient) rejectAuthShapedList(
	ctx context.Context,
	resource *datasafev1beta1.DiscoveryJob,
) error {
	if c.list == nil {
		return nil
	}
	identity := discoveryJobIdentityFromResource(resource)
	if identity.compartmentID == "" || identity.sensitiveDataModelID == "" {
		return nil
	}
	_, err := c.list(ctx, datasafesdk.ListDiscoveryJobsRequest{
		CompartmentId:        common.String(identity.compartmentID),
		SensitiveDataModelId: common.String(identity.sensitiveDataModelID),
		DisplayName:          discoveryJobOptionalString(identity.displayName),
	})
	return discoveryJobAmbiguousDeleteError(resource, err, "pre-delete list")
}

func discoveryJobAmbiguousDeleteError(
	resource *datasafev1beta1.DiscoveryJob,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return discoveryJobAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", discoveryJobKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func trackedDiscoveryJobID(resource *datasafev1beta1.DiscoveryJob) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func discoveryJobOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func discoveryJobStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func discoveryJobBoolValue(value *bool) bool {
	return value != nil && *value
}

func discoveryJobInt64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func discoveryJobSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func discoveryJobStringSlice(source []string) []string {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]string, len(source))
	copy(cloned, source)
	return cloned
}

func discoveryJobTablesForDiscovery(source []datasafesdk.TablesForDiscovery) []datasafesdk.TablesForDiscovery {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]datasafesdk.TablesForDiscovery, 0, len(source))
	for _, table := range source {
		cloned = append(cloned, datasafesdk.TablesForDiscovery{
			SchemaName: discoveryJobOptionalString(discoveryJobStringValue(table.SchemaName)),
			TableNames: discoveryJobStringSlice(table.TableNames),
		})
	}
	return cloned
}

func discoveryJobAPITablesForDiscovery(source []datasafesdk.TablesForDiscovery) []datasafev1beta1.DiscoveryJobTablesForDiscovery {
	if len(source) == 0 {
		return nil
	}
	tables := make([]datasafev1beta1.DiscoveryJobTablesForDiscovery, 0, len(source))
	for _, table := range source {
		tables = append(tables, datasafev1beta1.DiscoveryJobTablesForDiscovery{
			SchemaName: discoveryJobStringValue(table.SchemaName),
			TableNames: discoveryJobStringSlice(table.TableNames),
		})
	}
	return tables
}

func discoveryJobStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func discoveryJobSharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		converted[namespace] = make(shared.MapValue, len(values))
		for key, value := range values {
			converted[namespace][key] = fmt.Sprint(value)
		}
	}
	return converted
}

func discoveryJobCloneSharedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		cloned[namespace] = make(shared.MapValue, len(values))
		for key, value := range values {
			cloned[namespace][key] = value
		}
	}
	return cloned
}

func discoveryJobDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

var _ interface{ GetOpcRequestID() string } = discoveryJobAmbiguousNotFoundError{}
