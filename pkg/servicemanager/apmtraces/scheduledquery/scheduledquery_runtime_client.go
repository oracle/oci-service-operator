/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package scheduledquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apmtracessdk "github.com/oracle/oci-go-sdk/v65/apmtraces"
	apmtracesv1beta1 "github.com/oracle/oci-service-operator/api/apmtraces/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type scheduledQueryOCIClient interface {
	CreateScheduledQuery(context.Context, apmtracessdk.CreateScheduledQueryRequest) (apmtracessdk.CreateScheduledQueryResponse, error)
	GetScheduledQuery(context.Context, apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error)
	ListScheduledQueries(context.Context, apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error)
	UpdateScheduledQuery(context.Context, apmtracessdk.UpdateScheduledQueryRequest) (apmtracessdk.UpdateScheduledQueryResponse, error)
	DeleteScheduledQuery(context.Context, apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error)
}

func init() {
	registerScheduledQueryRuntimeHooksMutator(func(_ *ScheduledQueryServiceManager, hooks *ScheduledQueryRuntimeHooks) {
		applyScheduledQueryRuntimeHooks(hooks)
	})
}

func applyScheduledQueryRuntimeHooks(hooks *ScheduledQueryRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedScheduledQueryRuntimeSemantics()
	hooks.BuildCreateBody = scheduledQueryBuildCreateBody
	hooks.BuildUpdateBody = scheduledQueryBuildUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardScheduledQueryExistingBeforeCreate
	hooks.Create.Fields = scheduledQueryCreateFields()
	hooks.Get.Fields = scheduledQueryGetFields()
	hooks.List.Fields = scheduledQueryListFields()
	hooks.Update.Fields = scheduledQueryUpdateFields()
	hooks.Delete.Fields = scheduledQueryDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapScheduledQueryStatusMirrorClient)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapScheduledQueryUntrackedDeleteClient)
}

func reviewedScheduledQueryRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newScheduledQueryRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"scheduledQueryName"},
	}
	semantics.Unsupported = nil
	return semantics
}

func newScheduledQueryServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client scheduledQueryOCIClient,
) ScheduledQueryServiceClient {
	hooks := newScheduledQueryRuntimeHooksWithOCIClient(client)
	applyScheduledQueryRuntimeHooks(&hooks)
	delegate := defaultScheduledQueryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apmtracesv1beta1.ScheduledQuery](
			buildScheduledQueryGeneratedRuntimeConfig(&ScheduledQueryServiceManager{Log: log}, hooks),
		),
	}
	return wrapScheduledQueryGeneratedClient(hooks, delegate)
}

func newScheduledQueryRuntimeHooksWithOCIClient(client scheduledQueryOCIClient) ScheduledQueryRuntimeHooks {
	return ScheduledQueryRuntimeHooks{
		Semantics:       newScheduledQueryRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*apmtracesv1beta1.ScheduledQuery]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*apmtracesv1beta1.ScheduledQuery]{},
		StatusHooks:     generatedruntime.StatusHooks[*apmtracesv1beta1.ScheduledQuery]{},
		ParityHooks:     generatedruntime.ParityHooks[*apmtracesv1beta1.ScheduledQuery]{},
		Async:           generatedruntime.AsyncHooks[*apmtracesv1beta1.ScheduledQuery]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*apmtracesv1beta1.ScheduledQuery]{},
		Create: runtimeOperationHooks[apmtracessdk.CreateScheduledQueryRequest, apmtracessdk.CreateScheduledQueryResponse]{
			Fields: scheduledQueryCreateFields(),
			Call: func(ctx context.Context, request apmtracessdk.CreateScheduledQueryRequest) (apmtracessdk.CreateScheduledQueryResponse, error) {
				return client.CreateScheduledQuery(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apmtracessdk.GetScheduledQueryRequest, apmtracessdk.GetScheduledQueryResponse]{
			Fields: scheduledQueryGetFields(),
			Call: func(ctx context.Context, request apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error) {
				return client.GetScheduledQuery(ctx, request)
			},
		},
		List: runtimeOperationHooks[apmtracessdk.ListScheduledQueriesRequest, apmtracessdk.ListScheduledQueriesResponse]{
			Fields: scheduledQueryListFields(),
			Call: func(ctx context.Context, request apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error) {
				return client.ListScheduledQueries(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apmtracessdk.UpdateScheduledQueryRequest, apmtracessdk.UpdateScheduledQueryResponse]{
			Fields: scheduledQueryUpdateFields(),
			Call: func(ctx context.Context, request apmtracessdk.UpdateScheduledQueryRequest) (apmtracessdk.UpdateScheduledQueryResponse, error) {
				return client.UpdateScheduledQuery(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apmtracessdk.DeleteScheduledQueryRequest, apmtracessdk.DeleteScheduledQueryResponse]{
			Fields: scheduledQueryDeleteFields(),
			Call: func(ctx context.Context, request apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error) {
				return client.DeleteScheduledQuery(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ScheduledQueryServiceClient) ScheduledQueryServiceClient{},
	}
}

func scheduledQueryCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "CreateScheduledQueryDetails", RequestName: "CreateScheduledQueryDetails", Contribution: "body"},
	}
}

func scheduledQueryGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScheduledQueryId", RequestName: "scheduledQueryId", Contribution: "path", PreferResourceID: true},
	}
}

func scheduledQueryListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.scheduledQueryName", "spec.scheduledQueryName", "scheduledQueryName"},
		},
	}
}

func scheduledQueryUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScheduledQueryId", RequestName: "scheduledQueryId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateScheduledQueryDetails", RequestName: "UpdateScheduledQueryDetails", Contribution: "body"},
	}
}

func scheduledQueryDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScheduledQueryId", RequestName: "scheduledQueryId", Contribution: "path", PreferResourceID: true},
	}
}

func guardScheduledQueryExistingBeforeCreate(
	_ context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ScheduledQuery resource is nil")
	}
	if strings.TrimSpace(resource.Spec.ApmDomainId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ScheduledQuery spec.apmDomainId is required")
	}
	if strings.TrimSpace(resource.Spec.ScheduledQueryName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func scheduledQueryBuildCreateBody(
	_ context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ScheduledQuery resource is nil")
	}
	return scheduledQueryCreateDetailsFromSpec(resource.Spec), nil
}

func scheduledQueryBuildUpdateBody(
	_ context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("ScheduledQuery resource is nil")
	}

	desired := scheduledQueryDesiredMap(resource.Spec)
	current := scheduledQueryCurrentMap(currentResponse)
	changed := map[string]bool{}
	for key, desiredValue := range desired {
		if !scheduledQueryJSONEqual(desiredValue, current[key]) {
			changed[key] = true
		}
	}
	if len(changed) == 0 {
		return nil, false, nil
	}
	return scheduledQueryUpdateDetailsFromSpec(resource.Spec, changed), true, nil
}

func scheduledQueryCreateDetailsFromSpec(spec apmtracesv1beta1.ScheduledQuerySpec) apmtracessdk.CreateScheduledQueryDetails {
	return apmtracessdk.CreateScheduledQueryDetails{
		ScheduledQueryName:                    scheduledQueryStringPtr(spec.ScheduledQueryName),
		ScheduledQueryProcessingType:          apmtracessdk.ScheduledQueryProcessingTypeEnum(strings.TrimSpace(spec.ScheduledQueryProcessingType)),
		ScheduledQueryText:                    scheduledQueryStringPtr(spec.ScheduledQueryText),
		ScheduledQuerySchedule:                scheduledQueryStringPtr(spec.ScheduledQuerySchedule),
		ScheduledQueryDescription:             scheduledQueryStringPtr(spec.ScheduledQueryDescription),
		ScheduledQueryMaximumRuntimeInSeconds: scheduledQueryInt64Ptr(spec.ScheduledQueryMaximumRuntimeInSeconds),
		ScheduledQueryRetentionPeriodInMs:     scheduledQueryRetentionPeriodPtr(spec),
		ScheduledQueryProcessingSubType:       apmtracessdk.ScheduledQueryProcessingSubTypeEnum(strings.TrimSpace(spec.ScheduledQueryProcessingSubType)),
		ScheduledQueryProcessingConfiguration: scheduledQueryProcessingConfigFromSpec(spec.ScheduledQueryProcessingConfiguration),
		ScheduledQueryRetentionCriteria:       apmtracessdk.ScheduledQueryRetentionCriteriaEnum(strings.TrimSpace(spec.ScheduledQueryRetentionCriteria)),
		FreeformTags:                          scheduledQueryStringMap(spec.FreeformTags),
		DefinedTags:                           scheduledQueryDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func scheduledQueryUpdateDetailsFromSpec(
	spec apmtracesv1beta1.ScheduledQuerySpec,
	changed map[string]bool,
) apmtracessdk.UpdateScheduledQueryDetails {
	details := apmtracessdk.UpdateScheduledQueryDetails{}
	if changed["scheduledQueryName"] {
		details.ScheduledQueryName = scheduledQueryStringPtr(spec.ScheduledQueryName)
	}
	if changed["scheduledQueryProcessingType"] {
		details.ScheduledQueryProcessingType = apmtracessdk.ScheduledQueryProcessingTypeEnum(strings.TrimSpace(spec.ScheduledQueryProcessingType))
	}
	if changed["scheduledQueryProcessingSubType"] {
		details.ScheduledQueryProcessingSubType = apmtracessdk.ScheduledQueryProcessingSubTypeEnum(strings.TrimSpace(spec.ScheduledQueryProcessingSubType))
	}
	if changed["scheduledQueryText"] {
		details.ScheduledQueryText = scheduledQueryStringPtr(spec.ScheduledQueryText)
	}
	if changed["scheduledQuerySchedule"] {
		details.ScheduledQuerySchedule = scheduledQueryStringPtr(spec.ScheduledQuerySchedule)
	}
	if changed["scheduledQueryDescription"] {
		details.ScheduledQueryDescription = scheduledQueryStringPtr(spec.ScheduledQueryDescription)
	}
	if changed["scheduledQueryMaximumRuntimeInSeconds"] {
		details.ScheduledQueryMaximumRuntimeInSeconds = scheduledQueryInt64Ptr(spec.ScheduledQueryMaximumRuntimeInSeconds)
	}
	if changed["scheduledQueryRetentionPeriodInMs"] {
		details.ScheduledQueryRetentionPeriodInMs = scheduledQueryRetentionPeriodPtr(spec)
	}
	if changed["scheduledQueryProcessingConfiguration"] {
		details.ScheduledQueryProcessingConfiguration = scheduledQueryProcessingConfigFromSpec(spec.ScheduledQueryProcessingConfiguration)
	}
	if changed["scheduledQueryRetentionCriteria"] {
		details.ScheduledQueryRetentionCriteria = apmtracessdk.ScheduledQueryRetentionCriteriaEnum(strings.TrimSpace(spec.ScheduledQueryRetentionCriteria))
	}
	if changed["freeformTags"] {
		details.FreeformTags = scheduledQueryStringMap(spec.FreeformTags)
	}
	if changed["definedTags"] {
		details.DefinedTags = scheduledQueryDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details
}

func scheduledQueryDesiredMap(spec apmtracesv1beta1.ScheduledQuerySpec) map[string]any {
	desired := map[string]any{}
	if value := strings.TrimSpace(spec.ScheduledQueryName); value != "" {
		desired["scheduledQueryName"] = value
	}
	if value := strings.TrimSpace(spec.ScheduledQueryProcessingType); value != "" {
		desired["scheduledQueryProcessingType"] = value
	}
	if value := strings.TrimSpace(spec.ScheduledQueryText); value != "" {
		desired["scheduledQueryText"] = value
	}
	if value := strings.TrimSpace(spec.ScheduledQuerySchedule); value != "" {
		desired["scheduledQuerySchedule"] = value
	}
	if value := strings.TrimSpace(spec.ScheduledQueryDescription); value != "" {
		desired["scheduledQueryDescription"] = value
	}
	if spec.ScheduledQueryMaximumRuntimeInSeconds != 0 {
		desired["scheduledQueryMaximumRuntimeInSeconds"] = spec.ScheduledQueryMaximumRuntimeInSeconds
	}
	if scheduledQueryRetentionPeriodPtr(spec) != nil {
		desired["scheduledQueryRetentionPeriodInMs"] = spec.ScheduledQueryRetentionPeriodInMs
	}
	if value := strings.TrimSpace(spec.ScheduledQueryProcessingSubType); value != "" {
		desired["scheduledQueryProcessingSubType"] = value
	}
	if value := scheduledQueryProcessingConfigFromSpec(spec.ScheduledQueryProcessingConfiguration); value != nil {
		desired["scheduledQueryProcessingConfiguration"] = value
	}
	if value := strings.TrimSpace(spec.ScheduledQueryRetentionCriteria); value != "" {
		desired["scheduledQueryRetentionCriteria"] = value
	}
	if len(spec.FreeformTags) > 0 {
		desired["freeformTags"] = scheduledQueryStringMap(spec.FreeformTags)
	}
	if len(spec.DefinedTags) > 0 {
		desired["definedTags"] = scheduledQueryDefinedTagsFromSpec(spec.DefinedTags)
	}
	return desired
}

func scheduledQueryCurrentMap(currentResponse any) map[string]any {
	current, ok := scheduledQueryCurrent(currentResponse)
	if !ok {
		return map[string]any{}
	}

	values := map[string]any{}
	if current.ScheduledQueryName != nil {
		values["scheduledQueryName"] = *current.ScheduledQueryName
	}
	if current.ScheduledQueryProcessingType != "" {
		values["scheduledQueryProcessingType"] = string(current.ScheduledQueryProcessingType)
	}
	if current.ScheduledQueryText != nil {
		values["scheduledQueryText"] = *current.ScheduledQueryText
	}
	if current.ScheduledQuerySchedule != nil {
		values["scheduledQuerySchedule"] = *current.ScheduledQuerySchedule
	}
	if current.ScheduledQueryDescription != nil {
		values["scheduledQueryDescription"] = *current.ScheduledQueryDescription
	}
	if current.ScheduledQueryMaximumRuntimeInSeconds != nil {
		values["scheduledQueryMaximumRuntimeInSeconds"] = *current.ScheduledQueryMaximumRuntimeInSeconds
	}
	if current.ScheduledQueryRetentionPeriodInMs != nil {
		values["scheduledQueryRetentionPeriodInMs"] = *current.ScheduledQueryRetentionPeriodInMs
	}
	if current.ScheduledQueryProcessingSubType != "" {
		values["scheduledQueryProcessingSubType"] = string(current.ScheduledQueryProcessingSubType)
	}
	if current.ScheduledQueryProcessingConfiguration != nil {
		values["scheduledQueryProcessingConfiguration"] = current.ScheduledQueryProcessingConfiguration
	}
	if current.ScheduledQueryRetentionCriteria != "" {
		values["scheduledQueryRetentionCriteria"] = string(current.ScheduledQueryRetentionCriteria)
	}
	if len(current.FreeformTags) > 0 {
		values["freeformTags"] = current.FreeformTags
	}
	if len(current.DefinedTags) > 0 {
		values["definedTags"] = current.DefinedTags
	}
	return values
}

func scheduledQueryCurrent(currentResponse any) (apmtracessdk.ScheduledQuery, bool) {
	switch typed := currentResponse.(type) {
	case apmtracessdk.GetScheduledQueryResponse:
		return typed.ScheduledQuery, true
	case *apmtracessdk.GetScheduledQueryResponse:
		if typed == nil {
			return apmtracessdk.ScheduledQuery{}, false
		}
		return typed.ScheduledQuery, true
	case apmtracessdk.CreateScheduledQueryResponse:
		return typed.ScheduledQuery, true
	case *apmtracessdk.CreateScheduledQueryResponse:
		if typed == nil {
			return apmtracessdk.ScheduledQuery{}, false
		}
		return typed.ScheduledQuery, true
	case apmtracessdk.UpdateScheduledQueryResponse:
		return typed.ScheduledQuery, true
	case *apmtracessdk.UpdateScheduledQueryResponse:
		if typed == nil {
			return apmtracessdk.ScheduledQuery{}, false
		}
		return typed.ScheduledQuery, true
	case apmtracessdk.ScheduledQuery:
		return typed, true
	case *apmtracessdk.ScheduledQuery:
		if typed == nil {
			return apmtracessdk.ScheduledQuery{}, false
		}
		return *typed, true
	default:
		return apmtracessdk.ScheduledQuery{}, false
	}
}

func scheduledQueryProcessingConfigFromSpec(
	spec apmtracesv1beta1.ScheduledQueryProcessingConfiguration,
) *apmtracessdk.ScheduledQueryProcessingConfig {
	config := apmtracessdk.ScheduledQueryProcessingConfig{}

	if value := strings.TrimSpace(spec.Streaming.StreamId); value != "" {
		config.Streaming = &apmtracessdk.Streaming{StreamId: &value}
	}
	if objectStorage := scheduledQueryObjectStorageFromSpec(spec.ObjectStorage); objectStorage != nil {
		config.ObjectStorage = objectStorage
	}
	if customMetric := scheduledQueryCustomMetricFromSpec(spec.CustomMetric); customMetric != nil {
		config.CustomMetric = customMetric
	}
	if config.Streaming == nil && config.ObjectStorage == nil && config.CustomMetric == nil {
		return nil
	}
	return &config
}

func scheduledQueryObjectStorageFromSpec(
	spec apmtracesv1beta1.ScheduledQueryProcessingConfigurationObjectStorage,
) *apmtracessdk.ObjectStorage {
	objectStorage := apmtracessdk.ObjectStorage{
		BucketName:       scheduledQueryStringPtr(spec.BucketName),
		NameSpace:        scheduledQueryStringPtr(spec.NameSpace),
		ObjectNamePrefix: scheduledQueryStringPtr(spec.ObjectNamePrefix),
	}
	if objectStorage.BucketName == nil && objectStorage.NameSpace == nil && objectStorage.ObjectNamePrefix == nil {
		return nil
	}
	return &objectStorage
}

func scheduledQueryCustomMetricFromSpec(
	spec apmtracesv1beta1.ScheduledQueryProcessingConfigurationCustomMetric,
) *apmtracessdk.CustomMetric {
	if strings.TrimSpace(spec.Name) == "" {
		return nil
	}
	return &apmtracessdk.CustomMetric{
		Name:                      scheduledQueryStringPtr(spec.Name),
		Namespace:                 scheduledQueryStringPtr(spec.Namespace),
		Description:               scheduledQueryStringPtr(spec.Description),
		ResourceGroup:             scheduledQueryStringPtr(spec.ResourceGroup),
		IsAnomalyDetectionEnabled: &spec.IsAnomalyDetectionEnabled,
		Compartment:               scheduledQueryStringPtr(spec.Compartment),
		Unit:                      scheduledQueryStringPtr(spec.Unit),
		IsMetricPublished:         &spec.IsMetricPublished,
	}
}

func scheduledQueryStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func scheduledQueryInt64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func scheduledQueryRetentionPeriodPtr(spec apmtracesv1beta1.ScheduledQuerySpec) *int64 {
	if strings.EqualFold(strings.TrimSpace(spec.ScheduledQueryRetentionCriteria), string(apmtracessdk.ScheduledQueryRetentionCriteriaUpdate)) {
		return nil
	}
	return scheduledQueryInt64Ptr(spec.ScheduledQueryRetentionPeriodInMs)
}

func scheduledQueryStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func scheduledQueryDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if len(source) == 0 {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		out[namespace] = converted
	}
	return out
}

func scheduledQueryJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

type scheduledQueryStatusMirrorClient struct {
	delegate ScheduledQueryServiceClient
}

func wrapScheduledQueryStatusMirrorClient(delegate ScheduledQueryServiceClient) ScheduledQueryServiceClient {
	return scheduledQueryStatusMirrorClient{delegate: delegate}
}

func (c scheduledQueryStatusMirrorClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful {
		projectScheduledQueryRequestContext(resource)
	}
	return response, err
}

func (c scheduledQueryStatusMirrorClient) Delete(ctx context.Context, resource *apmtracesv1beta1.ScheduledQuery) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

type scheduledQueryUntrackedDeleteClient struct {
	delegate ScheduledQueryServiceClient
}

func wrapScheduledQueryUntrackedDeleteClient(delegate ScheduledQueryServiceClient) ScheduledQueryServiceClient {
	return scheduledQueryUntrackedDeleteClient{delegate: delegate}
}

func (c scheduledQueryUntrackedDeleteClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmtracesv1beta1.ScheduledQuery,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c scheduledQueryUntrackedDeleteClient) Delete(ctx context.Context, resource *apmtracesv1beta1.ScheduledQuery) (bool, error) {
	if scheduledQueryHasNoTrackedIdentity(resource) {
		return true, nil
	}
	return c.delegate.Delete(ctx, resource)
}

func scheduledQueryHasNoTrackedIdentity(resource *apmtracesv1beta1.ScheduledQuery) bool {
	if resource == nil {
		return false
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		return false
	}
	return strings.TrimSpace(resource.Status.Id) == "" &&
		strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) == "" &&
		strings.TrimSpace(resource.Status.ApmDomainId) == ""
}

func projectScheduledQueryRequestContext(resource *apmtracesv1beta1.ScheduledQuery) {
	if resource == nil {
		return
	}

	resource.Status.ApmDomainId = strings.TrimSpace(resource.Spec.ApmDomainId)
	if strings.TrimSpace(resource.Status.ScheduledQueryName) == "" {
		resource.Status.ScheduledQueryName = strings.TrimSpace(resource.Spec.ScheduledQueryName)
	}
}
