/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsentity

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type logAnalyticsEntityNamespaceClient interface {
	ListNamespaces(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)
}

type logAnalyticsEntityDeleteReadClient interface {
	GetLogAnalyticsEntity(context.Context, loganalyticssdk.GetLogAnalyticsEntityRequest) (loganalyticssdk.GetLogAnalyticsEntityResponse, error)
}

type logAnalyticsEntityRuntimeOCIClient interface {
	logAnalyticsEntityNamespaceClient
	logAnalyticsEntityDeleteReadClient
	CreateLogAnalyticsEntity(context.Context, loganalyticssdk.CreateLogAnalyticsEntityRequest) (loganalyticssdk.CreateLogAnalyticsEntityResponse, error)
	ListLogAnalyticsEntities(context.Context, loganalyticssdk.ListLogAnalyticsEntitiesRequest) (loganalyticssdk.ListLogAnalyticsEntitiesResponse, error)
	UpdateLogAnalyticsEntity(context.Context, loganalyticssdk.UpdateLogAnalyticsEntityRequest) (loganalyticssdk.UpdateLogAnalyticsEntityResponse, error)
	DeleteLogAnalyticsEntity(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityRequest) (loganalyticssdk.DeleteLogAnalyticsEntityResponse, error)
}

type logAnalyticsEntityNamespaceResolver struct {
	client  logAnalyticsEntityNamespaceClient
	initErr error
}

type logAnalyticsEntityNamespaceResolvingClient struct {
	delegate     LogAnalyticsEntityServiceClient
	deleteReader logAnalyticsEntityDeleteReadClient
	resolver     logAnalyticsEntityNamespaceResolver
}

func init() {
	registerLogAnalyticsEntityRuntimeHooksMutator(func(manager *LogAnalyticsEntityServiceManager, hooks *LogAnalyticsEntityRuntimeHooks) {
		namespaceClient, initErr := newLogAnalyticsEntityNamespaceClient(manager)
		applyLogAnalyticsEntityRuntimeHooks(hooks, namespaceClient, initErr)
	})
}

func newLogAnalyticsEntityNamespaceClient(manager *LogAnalyticsEntityServiceManager) (logAnalyticsEntityNamespaceClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("LogAnalyticsEntity service manager is nil")
	}
	return loganalyticssdk.NewLogAnalyticsClientWithConfigurationProvider(manager.Provider)
}

func applyLogAnalyticsEntityRuntimeHooks(
	hooks *LogAnalyticsEntityRuntimeHooks,
	namespaceClient logAnalyticsEntityNamespaceClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	listCall := hooks.List.Call
	hooks.Semantics = newLogAnalyticsEntityRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *loganalyticsv1beta1.LogAnalyticsEntity, _ string) (any, error) {
		return buildLogAnalyticsEntityCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loganalyticsv1beta1.LogAnalyticsEntity,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildLogAnalyticsEntityUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = logAnalyticsEntityCreateFields()
	hooks.Get.Fields = logAnalyticsEntityGetFields()
	hooks.List.Fields = logAnalyticsEntityListFields()
	hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEntitiesRequest) (loganalyticssdk.ListLogAnalyticsEntitiesResponse, error) {
		return listLogAnalyticsEntityPages(ctx, request, listCall)
	}
	hooks.Update.Fields = logAnalyticsEntityUpdateFields()
	hooks.Delete.Fields = logAnalyticsEntityDeleteFields()
	hooks.DeleteHooks.HandleError = handleLogAnalyticsEntityDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleLogAnalyticsEntityDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LogAnalyticsEntityServiceClient) LogAnalyticsEntityServiceClient {
		deleteReader, _ := namespaceClient.(logAnalyticsEntityDeleteReadClient)
		return logAnalyticsEntityNamespaceResolvingClient{
			delegate:     delegate,
			deleteReader: deleteReader,
			resolver: logAnalyticsEntityNamespaceResolver{
				client:  namespaceClient,
				initErr: initErr,
			},
		}
	})
}

func newLogAnalyticsEntityRuntimeHooksWithOCIClient(client logAnalyticsEntityRuntimeOCIClient) LogAnalyticsEntityRuntimeHooks {
	hooks := newLogAnalyticsEntityDefaultRuntimeHooks(loganalyticssdk.LogAnalyticsClient{})
	hooks.Create.Call = func(ctx context.Context, request loganalyticssdk.CreateLogAnalyticsEntityRequest) (loganalyticssdk.CreateLogAnalyticsEntityResponse, error) {
		return client.CreateLogAnalyticsEntity(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request loganalyticssdk.GetLogAnalyticsEntityRequest) (loganalyticssdk.GetLogAnalyticsEntityResponse, error) {
		return client.GetLogAnalyticsEntity(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEntitiesRequest) (loganalyticssdk.ListLogAnalyticsEntitiesResponse, error) {
		return client.ListLogAnalyticsEntities(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request loganalyticssdk.UpdateLogAnalyticsEntityRequest) (loganalyticssdk.UpdateLogAnalyticsEntityResponse, error) {
		return client.UpdateLogAnalyticsEntity(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request loganalyticssdk.DeleteLogAnalyticsEntityRequest) (loganalyticssdk.DeleteLogAnalyticsEntityResponse, error) {
		return client.DeleteLogAnalyticsEntity(ctx, request)
	}
	return hooks
}

func newLogAnalyticsEntityRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:       "loganalytics",
		FormalSlug:          "loganalyticsentity",
		StatusProjection:    "required",
		SecretSideEffects:   "none",
		FinalizerPolicy:     "retain-until-confirmed-delete",
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(loganalyticssdk.EntityLifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(loganalyticssdk.EntityLifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "entityTypeName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"name",
				"managementAgentId",
				"cloudResourceId",
				"timezoneRegion",
				"hostname",
				"properties",
				"freeformTags",
				"definedTags",
				"timeLastDiscovered",
				"metadata",
			},
			ForceNew:      []string{"compartmentId", "entityTypeName", "sourceId"},
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
	}
}

func logAnalyticsEntityCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "CreateLogAnalyticsEntityDetails", RequestName: "CreateLogAnalyticsEntityDetails", Contribution: "body"},
	}
}

func logAnalyticsEntityGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "LogAnalyticsEntityId", RequestName: "logAnalyticsEntityId", Contribution: "path", PreferResourceID: true},
		{FieldName: "IsShowAssociatedSourcesCount", RequestName: "isShowAssociatedSourcesCount", Contribution: "query"},
	}
}

func logAnalyticsEntityListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "CloudResourceId", RequestName: "cloudResourceId", Contribution: "query"},
		{FieldName: "Hostname", RequestName: "hostname", Contribution: "query"},
		{FieldName: "SourceId", RequestName: "sourceId", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "IsShowAssociatedSourcesCount", RequestName: "isShowAssociatedSourcesCount", Contribution: "query"},
	}
}

func logAnalyticsEntityUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "LogAnalyticsEntityId", RequestName: "logAnalyticsEntityId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateLogAnalyticsEntityDetails", RequestName: "UpdateLogAnalyticsEntityDetails", Contribution: "body"},
	}
}

func logAnalyticsEntityDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "LogAnalyticsEntityId", RequestName: "logAnalyticsEntityId", Contribution: "path", PreferResourceID: true},
		{FieldName: "IsForceDelete", RequestName: "isForceDelete", Contribution: "query"},
	}
}

func (c logAnalyticsEntityNamespaceResolvingClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	runtimeResource, err := c.resourceWithResolvedNamespace(ctx, resource)
	if err != nil {
		return markLogAnalyticsEntityFailure(resource, err), err
	}
	req.Namespace = runtimeResource.Namespace
	response, err := c.delegate.CreateOrUpdate(ctx, runtimeResource, req)
	copyLogAnalyticsEntityStatus(resource, runtimeResource)
	return response, err
}

func (c logAnalyticsEntityNamespaceResolvingClient) Delete(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) (bool, error) {
	runtimeResource, err := c.resourceWithResolvedNamespace(ctx, resource)
	if err != nil {
		return false, err
	}
	if err := c.guardDeleteAgainstAuthShapedReadback(ctx, runtimeResource); err != nil {
		copyLogAnalyticsEntityStatus(resource, runtimeResource)
		return false, err
	}
	deleted, err := c.delegate.Delete(ctx, runtimeResource)
	copyLogAnalyticsEntityStatus(resource, runtimeResource)
	return deleted, err
}

func (c logAnalyticsEntityNamespaceResolvingClient) guardDeleteAgainstAuthShapedReadback(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) error {
	if c.deleteReader == nil || resource == nil {
		return nil
	}
	currentID := firstNonEmptyTrim(string(resource.Status.OsokStatus.Ocid), resource.Status.Id)
	if currentID == "" {
		return nil
	}
	_, err := c.deleteReader.GetLogAnalyticsEntity(ctx, loganalyticssdk.GetLogAnalyticsEntityRequest{
		NamespaceName:                common.String(resource.Namespace),
		LogAnalyticsEntityId:         common.String(currentID),
		IsShowAssociatedSourcesCount: common.Bool(true),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	return handleLogAnalyticsEntityDeleteError(resource, err)
}

func (c logAnalyticsEntityNamespaceResolvingClient) resourceWithResolvedNamespace(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) (*loganalyticsv1beta1.LogAnalyticsEntity, error) {
	namespaceName, err := c.resolver.resolve(ctx, resource)
	if err != nil {
		return nil, err
	}
	runtimeResource := resource.DeepCopy()
	runtimeResource.Namespace = namespaceName
	return runtimeResource, nil
}

func (r logAnalyticsEntityNamespaceResolver) resolve(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace: resource is nil")
	}
	if r.initErr != nil {
		return "", fmt.Errorf("initialize LogAnalyticsEntity namespace resolver: %w", r.initErr)
	}
	if r.client == nil {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace: namespace client is nil")
	}

	compartmentID := firstNonEmptyTrim(resource.Status.CompartmentId, resource.Spec.CompartmentId)
	if compartmentID == "" {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace: spec.compartmentId is required")
	}

	response, err := r.client.ListNamespaces(ctx, loganalyticssdk.ListNamespacesRequest{
		CompartmentId: common.String(compartmentID),
	})
	if err != nil {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace for compartment %q: %w", compartmentID, err)
	}
	return logAnalyticsNamespaceName(response, compartmentID)
}

func logAnalyticsNamespaceName(response loganalyticssdk.ListNamespacesResponse, compartmentID string) (string, error) {
	items := response.Items
	if len(items) == 0 {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace: no Log Analytics namespace found for compartment %q", compartmentID)
	}
	if len(items) > 1 {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace: expected one namespace for compartment %q, got %d", compartmentID, len(items))
	}

	item := items[0]
	namespaceName := strings.TrimSpace(stringValue(item.NamespaceName))
	if namespaceName == "" {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace: namespace response for compartment %q did not include namespaceName", compartmentID)
	}
	if item.IsOnboarded != nil && !*item.IsOnboarded {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace %q: tenancy is not onboarded to Log Analytics", namespaceName)
	}
	if item.LifecycleState == loganalyticssdk.NamespaceSummaryLifecycleStateInactive {
		return "", fmt.Errorf("resolve LogAnalyticsEntity namespace %q: namespace is INACTIVE", namespaceName)
	}
	return namespaceName, nil
}

func buildLogAnalyticsEntityCreateBody(resource *loganalyticsv1beta1.LogAnalyticsEntity) (loganalyticssdk.CreateLogAnalyticsEntityDetails, error) {
	if resource == nil {
		return loganalyticssdk.CreateLogAnalyticsEntityDetails{}, fmt.Errorf("LogAnalyticsEntity resource is nil")
	}

	name, err := requiredLogAnalyticsEntityString("spec.name", resource.Spec.Name)
	if err != nil {
		return loganalyticssdk.CreateLogAnalyticsEntityDetails{}, err
	}
	compartmentID, err := requiredLogAnalyticsEntityString("spec.compartmentId", resource.Spec.CompartmentId)
	if err != nil {
		return loganalyticssdk.CreateLogAnalyticsEntityDetails{}, err
	}
	entityTypeName, err := requiredLogAnalyticsEntityString("spec.entityTypeName", resource.Spec.EntityTypeName)
	if err != nil {
		return loganalyticssdk.CreateLogAnalyticsEntityDetails{}, err
	}

	details := loganalyticssdk.CreateLogAnalyticsEntityDetails{
		Name:           common.String(name),
		CompartmentId:  common.String(compartmentID),
		EntityTypeName: common.String(entityTypeName),
	}
	if err := applyLogAnalyticsEntityCreateOptionalFields(&details, resource); err != nil {
		return loganalyticssdk.CreateLogAnalyticsEntityDetails{}, err
	}
	return details, nil
}

func applyLogAnalyticsEntityCreateOptionalFields(
	details *loganalyticssdk.CreateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) error {
	applyLogAnalyticsEntityCreateStringFields(details, resource)
	applyLogAnalyticsEntityCreateMapFields(details, resource)
	return applyLogAnalyticsEntityCreateStructuredFields(details, resource)
}

func applyLogAnalyticsEntityCreateStringFields(
	details *loganalyticssdk.CreateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) {
	if value := strings.TrimSpace(resource.Spec.ManagementAgentId); value != "" {
		details.ManagementAgentId = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.CloudResourceId); value != "" {
		details.CloudResourceId = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.TimezoneRegion); value != "" {
		details.TimezoneRegion = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.Hostname); value != "" {
		details.Hostname = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.SourceId); value != "" {
		details.SourceId = common.String(value)
	}
}

func applyLogAnalyticsEntityCreateMapFields(
	details *loganalyticssdk.CreateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) {
	if resource.Spec.Properties != nil {
		details.Properties = maps.Clone(resource.Spec.Properties)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = logAnalyticsEntityDefinedTags(resource.Spec.DefinedTags)
	}
}

func applyLogAnalyticsEntityCreateStructuredFields(
	details *loganalyticssdk.CreateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
) error {
	if value := strings.TrimSpace(resource.Spec.TimeLastDiscovered); value != "" {
		parsed, err := parseLogAnalyticsEntitySDKTime(value)
		if err != nil {
			return err
		}
		details.TimeLastDiscovered = parsed
	}
	if metadata := logAnalyticsEntityMetadataDetails(resource.Spec.Metadata); metadata != nil {
		details.Metadata = metadata
	}
	return nil
}

func buildLogAnalyticsEntityUpdateBody(
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	currentResponse any,
) (loganalyticssdk.UpdateLogAnalyticsEntityDetails, bool, error) {
	if resource == nil {
		return loganalyticssdk.UpdateLogAnalyticsEntityDetails{}, false, fmt.Errorf("LogAnalyticsEntity resource is nil")
	}
	current, err := logAnalyticsEntityRuntimeBody(currentResponse)
	if err != nil {
		return loganalyticssdk.UpdateLogAnalyticsEntityDetails{}, false, err
	}

	details := loganalyticssdk.UpdateLogAnalyticsEntityDetails{}
	updateNeeded := false
	if err := applyLogAnalyticsEntityStringUpdates(&details, resource, current, &updateNeeded); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsEntityDetails{}, false, err
	}
	if err := applyLogAnalyticsEntityCollectionUpdates(&details, resource, current, &updateNeeded); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsEntityDetails{}, false, err
	}
	return details, updateNeeded, nil
}

func applyLogAnalyticsEntityStringUpdates(
	details *loganalyticssdk.UpdateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	current loganalyticssdk.LogAnalyticsEntity,
	updateNeeded *bool,
) error {
	desiredName, err := requiredLogAnalyticsEntityString("spec.name", resource.Spec.Name)
	if err != nil {
		return err
	}
	if desired, ok := desiredRequiredStringUpdate(desiredName, current.Name); ok {
		details.Name = desired
		*updateNeeded = true
	}
	if desired, ok := desiredOptionalStringUpdate(resource.Spec.ManagementAgentId, current.ManagementAgentId); ok {
		details.ManagementAgentId = desired
		*updateNeeded = true
	}
	if desired, ok := desiredOptionalStringUpdate(resource.Spec.TimezoneRegion, current.TimezoneRegion); ok {
		details.TimezoneRegion = desired
		*updateNeeded = true
	}
	if desired, ok := desiredOptionalStringUpdate(resource.Spec.Hostname, current.Hostname); ok {
		details.Hostname = desired
		*updateNeeded = true
	}
	if desired, ok := desiredOptionalStringUpdate(resource.Spec.CloudResourceId, current.CloudResourceId); ok {
		details.CloudResourceId = desired
		*updateNeeded = true
	}
	if desired, ok, err := desiredSDKTimeUpdate(resource.Spec.TimeLastDiscovered, current.TimeLastDiscovered); err != nil {
		return err
	} else if ok {
		details.TimeLastDiscovered = desired
		*updateNeeded = true
	}
	return nil
}

func applyLogAnalyticsEntityCollectionUpdates(
	details *loganalyticssdk.UpdateLogAnalyticsEntityDetails,
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	current loganalyticssdk.LogAnalyticsEntity,
	updateNeeded *bool,
) error {
	if desired, ok := desiredStringMapUpdate(resource.Spec.Properties, current.Properties); ok {
		details.Properties = desired
		*updateNeeded = true
	}
	if desired, ok := desiredStringMapUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		*updateNeeded = true
	}
	if desired, ok := desiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		*updateNeeded = true
	}
	if desired, ok := desiredMetadataUpdate(resource.Spec.Metadata, current.Metadata); ok {
		details.Metadata = desired
		*updateNeeded = true
	}
	return nil
}

func listLogAnalyticsEntityPages(
	ctx context.Context,
	request loganalyticssdk.ListLogAnalyticsEntitiesRequest,
	call func(context.Context, loganalyticssdk.ListLogAnalyticsEntitiesRequest) (loganalyticssdk.ListLogAnalyticsEntitiesResponse, error),
) (loganalyticssdk.ListLogAnalyticsEntitiesResponse, error) {
	if call == nil {
		return loganalyticssdk.ListLogAnalyticsEntitiesResponse{}, fmt.Errorf("LogAnalyticsEntity list operation is nil")
	}

	var combined loganalyticssdk.ListLogAnalyticsEntitiesResponse
	nextRequest := request
	for {
		response, err := call(ctx, nextRequest)
		if err != nil {
			return loganalyticssdk.ListLogAnalyticsEntitiesResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return combined, nil
		}
		nextRequest.Page = response.OpcNextPage
	}
}

func handleLogAnalyticsEntityDeleteError(resource *loganalyticsv1beta1.LogAnalyticsEntity, err error) error {
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("LogAnalyticsEntity delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func handleLogAnalyticsEntityDeleteOutcome(
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if stage != generatedruntime.DeleteConfirmStageAfterRequest {
		return generatedruntime.DeleteOutcome{}, nil
	}
	current, err := logAnalyticsEntityRuntimeBody(response)
	if err != nil {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if current.LifecycleState != loganalyticssdk.EntityLifecycleStatesActive {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markLogAnalyticsEntityDeletePending(resource, string(current.LifecycleState))
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func markLogAnalyticsEntityDeletePending(resource *loganalyticsv1beta1.LogAnalyticsEntity, rawStatus string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	message := "OCI resource delete is in progress"
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
}

func logAnalyticsEntityRuntimeBody(currentResponse any) (loganalyticssdk.LogAnalyticsEntity, error) {
	extractors := []func(any) (loganalyticssdk.LogAnalyticsEntity, bool, error){
		logAnalyticsEntityDirectRuntimeBody,
		logAnalyticsEntityCreateResponseBody,
		logAnalyticsEntityGetResponseBody,
		logAnalyticsEntityUpdateResponseBody,
	}
	for _, extract := range extractors {
		current, ok, err := extract(currentResponse)
		if ok || err != nil {
			return current, err
		}
	}
	return loganalyticssdk.LogAnalyticsEntity{}, fmt.Errorf("unexpected current LogAnalyticsEntity response type %T", currentResponse)
}

func logAnalyticsEntityDirectRuntimeBody(currentResponse any) (loganalyticssdk.LogAnalyticsEntity, bool, error) {
	switch current := currentResponse.(type) {
	case loganalyticssdk.LogAnalyticsEntity:
		return current, true, nil
	case *loganalyticssdk.LogAnalyticsEntity:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntity{}, true, fmt.Errorf("current LogAnalyticsEntity response is nil")
		}
		return *current, true, nil
	case loganalyticssdk.LogAnalyticsEntitySummary:
		return logAnalyticsEntityFromSummary(current), true, nil
	case *loganalyticssdk.LogAnalyticsEntitySummary:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntity{}, true, fmt.Errorf("current LogAnalyticsEntity response is nil")
		}
		return logAnalyticsEntityFromSummary(*current), true, nil
	default:
		return loganalyticssdk.LogAnalyticsEntity{}, false, nil
	}
}

func logAnalyticsEntityCreateResponseBody(currentResponse any) (loganalyticssdk.LogAnalyticsEntity, bool, error) {
	switch current := currentResponse.(type) {
	case loganalyticssdk.CreateLogAnalyticsEntityResponse:
		return current.LogAnalyticsEntity, true, nil
	case *loganalyticssdk.CreateLogAnalyticsEntityResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntity{}, true, fmt.Errorf("current LogAnalyticsEntity response is nil")
		}
		return current.LogAnalyticsEntity, true, nil
	default:
		return loganalyticssdk.LogAnalyticsEntity{}, false, nil
	}
}

func logAnalyticsEntityGetResponseBody(currentResponse any) (loganalyticssdk.LogAnalyticsEntity, bool, error) {
	switch current := currentResponse.(type) {
	case loganalyticssdk.GetLogAnalyticsEntityResponse:
		return current.LogAnalyticsEntity, true, nil
	case *loganalyticssdk.GetLogAnalyticsEntityResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntity{}, true, fmt.Errorf("current LogAnalyticsEntity response is nil")
		}
		return current.LogAnalyticsEntity, true, nil
	default:
		return loganalyticssdk.LogAnalyticsEntity{}, false, nil
	}
}

func logAnalyticsEntityUpdateResponseBody(currentResponse any) (loganalyticssdk.LogAnalyticsEntity, bool, error) {
	switch current := currentResponse.(type) {
	case loganalyticssdk.UpdateLogAnalyticsEntityResponse:
		return current.LogAnalyticsEntity, true, nil
	case *loganalyticssdk.UpdateLogAnalyticsEntityResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntity{}, true, fmt.Errorf("current LogAnalyticsEntity response is nil")
		}
		return current.LogAnalyticsEntity, true, nil
	default:
		return loganalyticssdk.LogAnalyticsEntity{}, false, nil
	}
}

func logAnalyticsEntityFromSummary(summary loganalyticssdk.LogAnalyticsEntitySummary) loganalyticssdk.LogAnalyticsEntity {
	return loganalyticssdk.LogAnalyticsEntity{
		Id:                     summary.Id,
		Name:                   summary.Name,
		CompartmentId:          summary.CompartmentId,
		EntityTypeName:         summary.EntityTypeName,
		EntityTypeInternalName: summary.EntityTypeInternalName,
		LifecycleState:         summary.LifecycleState,
		LifecycleDetails:       summary.LifecycleDetails,
		TimeCreated:            summary.TimeCreated,
		TimeUpdated:            summary.TimeUpdated,
		ManagementAgentId:      summary.ManagementAgentId,
		CloudResourceId:        summary.CloudResourceId,
		TimezoneRegion:         summary.TimezoneRegion,
		TimeLastDiscovered:     summary.TimeLastDiscovered,
		Metadata:               metadataSummaryFromCollection(summary.Metadata),
		AreLogsCollected:       summary.AreLogsCollected,
		SourceId:               summary.SourceId,
		CreationSource:         summary.CreationSource,
		FreeformTags:           summary.FreeformTags,
		DefinedTags:            summary.DefinedTags,
		AssociatedSourcesCount: summary.AssociatedSourcesCount,
	}
}

func metadataSummaryFromCollection(collection *loganalyticssdk.LogAnalyticsMetadataCollection) *loganalyticssdk.LogAnalyticsMetadataSummary {
	if collection == nil {
		return nil
	}
	return &loganalyticssdk.LogAnalyticsMetadataSummary{Items: append([]loganalyticssdk.LogAnalyticsMetadata(nil), collection.Items...)}
}

func requiredLogAnalyticsEntityString(field string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("LogAnalyticsEntity %s is required", field)
	}
	return trimmed, nil
}

func desiredRequiredStringUpdate(spec string, current *string) (*string, bool) {
	if spec == stringValue(current) {
		return nil, false
	}
	return common.String(spec), true
}

func desiredOptionalStringUpdate(spec string, current *string) (*string, bool) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" || trimmed == stringValue(current) {
		return nil, false
	}
	return common.String(trimmed), true
}

func desiredStringMapUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil || maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func desiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := logAnalyticsEntityDefinedTags(spec)
	if logAnalyticsEntityJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func desiredMetadataUpdate(
	spec loganalyticsv1beta1.LogAnalyticsEntityMetadata,
	current *loganalyticssdk.LogAnalyticsMetadataSummary,
) (*loganalyticssdk.LogAnalyticsMetadataDetails, bool) {
	if spec.Items == nil {
		return nil, false
	}
	desired := logAnalyticsEntityMetadataDetails(spec)
	if desired == nil {
		return nil, false
	}
	if current == nil && len(desired.Items) == 0 {
		return nil, false
	}
	if current != nil && logAnalyticsEntityJSONEqual(desired.Items, current.Items) {
		return nil, false
	}
	return desired, true
}

func desiredSDKTimeUpdate(spec string, current *common.SDKTime) (*common.SDKTime, bool, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil, false, nil
	}
	desired, err := parseLogAnalyticsEntitySDKTime(trimmed)
	if err != nil {
		return nil, false, err
	}
	if current != nil && desired.Equal(current.Time) {
		return nil, false, nil
	}
	return desired, true, nil
}

func parseLogAnalyticsEntitySDKTime(value string) (*common.SDKTime, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("parse LogAnalyticsEntity timeLastDiscovered: %w", err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func logAnalyticsEntityDefinedTags(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func logAnalyticsEntityMetadataDetails(
	metadata loganalyticsv1beta1.LogAnalyticsEntityMetadata,
) *loganalyticssdk.LogAnalyticsMetadataDetails {
	if metadata.Items == nil {
		return nil
	}

	items := make([]loganalyticssdk.LogAnalyticsMetadata, 0, len(metadata.Items))
	for _, item := range metadata.Items {
		items = append(items, loganalyticssdk.LogAnalyticsMetadata{
			Name:  optionalStringPointer(item.Name),
			Value: optionalStringPointer(item.Value),
			Type:  optionalStringPointer(item.Type),
		})
	}
	return &loganalyticssdk.LogAnalyticsMetadataDetails{Items: items}
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return common.String(trimmed)
}

func logAnalyticsEntityJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func copyLogAnalyticsEntityStatus(dst *loganalyticsv1beta1.LogAnalyticsEntity, src *loganalyticsv1beta1.LogAnalyticsEntity) {
	if dst == nil || src == nil {
		return
	}
	dst.Status = src.Status
}

func markLogAnalyticsEntityFailure(
	resource *loganalyticsv1beta1.LogAnalyticsEntity,
	err error,
) servicemanager.OSOKResponse {
	if resource == nil || err == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), servicemanager.RuntimeDeps{}.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
