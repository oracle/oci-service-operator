/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsobjectcollectionrule

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type logAnalyticsObjectCollectionRuleOCIClient interface {
	CreateLogAnalyticsObjectCollectionRule(context.Context, loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error)
	GetLogAnalyticsObjectCollectionRule(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error)
	ListLogAnalyticsObjectCollectionRules(context.Context, loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error)
	UpdateLogAnalyticsObjectCollectionRule(context.Context, loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error)
	DeleteLogAnalyticsObjectCollectionRule(context.Context, loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error)
}

func init() {
	registerLogAnalyticsObjectCollectionRuleRuntimeHooksMutator(func(_ *LogAnalyticsObjectCollectionRuleServiceManager, hooks *LogAnalyticsObjectCollectionRuleRuntimeHooks) {
		applyLogAnalyticsObjectCollectionRuleRuntimeHooks(hooks)
	})
}

func applyLogAnalyticsObjectCollectionRuleRuntimeHooks(hooks *LogAnalyticsObjectCollectionRuleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = logAnalyticsObjectCollectionRuleRuntimeSemantics()
	hooks.BuildCreateBody = buildLogAnalyticsObjectCollectionRuleCreateBody
	hooks.BuildUpdateBody = buildLogAnalyticsObjectCollectionRuleUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardLogAnalyticsObjectCollectionRuleExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedLogAnalyticsObjectCollectionRuleIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateLogAnalyticsObjectCollectionRuleCreateOnlyDriftForResponse
	hooks.StatusHooks.MarkTerminating = markLogAnalyticsObjectCollectionRuleTerminating
	hooks.DeleteHooks.HandleError = handleLogAnalyticsObjectCollectionRuleDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyLogAnalyticsObjectCollectionRuleDeleteOutcome
	hooks.Create.Fields = logAnalyticsObjectCollectionRuleCreateFields()
	hooks.Get.Fields = logAnalyticsObjectCollectionRuleGetFields()
	hooks.List.Fields = logAnalyticsObjectCollectionRuleListFields()
	hooks.Update.Fields = logAnalyticsObjectCollectionRuleUpdateFields()
	hooks.Delete.Fields = logAnalyticsObjectCollectionRuleDeleteFields()
	wrapLogAnalyticsObjectCollectionRuleListCall(hooks)
}

func newLogAnalyticsObjectCollectionRuleServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client logAnalyticsObjectCollectionRuleOCIClient,
) LogAnalyticsObjectCollectionRuleServiceClient {
	hooks := newLogAnalyticsObjectCollectionRuleRuntimeHooksWithOCIClient(client)
	applyLogAnalyticsObjectCollectionRuleRuntimeHooks(&hooks)
	manager := &LogAnalyticsObjectCollectionRuleServiceManager{Log: log}
	delegate := defaultLogAnalyticsObjectCollectionRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule](
			buildLogAnalyticsObjectCollectionRuleGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapLogAnalyticsObjectCollectionRuleGeneratedClient(hooks, delegate)
}

func newLogAnalyticsObjectCollectionRuleRuntimeHooksWithOCIClient(client logAnalyticsObjectCollectionRuleOCIClient) LogAnalyticsObjectCollectionRuleRuntimeHooks {
	return LogAnalyticsObjectCollectionRuleRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{},
		StatusHooks:     generatedruntime.StatusHooks[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{},
		ParityHooks:     generatedruntime.ParityHooks[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{},
		Async:           generatedruntime.AsyncHooks[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*loganalyticsv1beta1.LogAnalyticsObjectCollectionRule]{},
		Create: runtimeOperationHooks[loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest, loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse]{
			Fields: logAnalyticsObjectCollectionRuleCreateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse{}, fmt.Errorf("LogAnalyticsObjectCollectionRule OCI client is nil")
				}
				return client.CreateLogAnalyticsObjectCollectionRule(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse]{
			Fields: logAnalyticsObjectCollectionRuleGetFields(),
			Call: func(ctx context.Context, request loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{}, fmt.Errorf("LogAnalyticsObjectCollectionRule OCI client is nil")
				}
				return client.GetLogAnalyticsObjectCollectionRule(ctx, request)
			},
		},
		List: runtimeOperationHooks[loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest, loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse]{
			Fields: logAnalyticsObjectCollectionRuleListFields(),
			Call: func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
				if client == nil {
					return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{}, fmt.Errorf("LogAnalyticsObjectCollectionRule OCI client is nil")
				}
				return client.ListLogAnalyticsObjectCollectionRules(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest, loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse]{
			Fields: logAnalyticsObjectCollectionRuleUpdateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse{}, fmt.Errorf("LogAnalyticsObjectCollectionRule OCI client is nil")
				}
				return client.UpdateLogAnalyticsObjectCollectionRule(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest, loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse]{
			Fields: logAnalyticsObjectCollectionRuleDeleteFields(),
			Call: func(ctx context.Context, request loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error) {
				if client == nil {
					return loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse{}, fmt.Errorf("LogAnalyticsObjectCollectionRule OCI client is nil")
				}
				return client.DeleteLogAnalyticsObjectCollectionRule(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LogAnalyticsObjectCollectionRuleServiceClient) LogAnalyticsObjectCollectionRuleServiceClient{},
	}
}

func logAnalyticsObjectCollectionRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "loganalytics",
		FormalSlug:        "loganalyticsobjectcollectionrule",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive),
				string(loganalyticssdk.ObjectCollectionRuleLifecycleStatesInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(loganalyticssdk.ObjectCollectionRuleLifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "osNamespace"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"charEncoding",
				"definedTags",
				"description",
				"entityId",
				"freeformTags",
				"isEnabled",
				"logGroupId",
				"logSet",
				"logSetExtRegex",
				"logSetKey",
				"logSourceName",
				"objectNameFilters",
				"overrides",
				"streamCursorTime",
				"streamCursorType",
				"streamId",
				"timezone",
			},
			Mutable: []string{
				"charEncoding",
				"definedTags",
				"description",
				"entityId",
				"freeformTags",
				"isEnabled",
				"logGroupId",
				"logSet",
				"logSetExtRegex",
				"logSetKey",
				"logSourceName",
				"objectNameFilters",
				"overrides",
				"streamCursorTime",
				"streamCursorType",
				"streamId",
				"timezone",
			},
			ForceNew: []string{
				"collectionType",
				"compartmentId",
				"isForceHistoricCollection",
				"logType",
				"name",
				"osBucketName",
				"osNamespace",
				"pollSince",
				"pollTill",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func logAnalyticsObjectCollectionRuleCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path", LookupPaths: logAnalyticsObjectCollectionRuleNamespaceLookupPaths()},
		{FieldName: "CreateLogAnalyticsObjectCollectionRuleDetails", RequestName: "CreateLogAnalyticsObjectCollectionRuleDetails", Contribution: "body"},
	}
}

func logAnalyticsObjectCollectionRuleGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path", LookupPaths: logAnalyticsObjectCollectionRuleNamespaceLookupPaths()},
		{FieldName: "LogAnalyticsObjectCollectionRuleId", RequestName: "logAnalyticsObjectCollectionRuleId", Contribution: "path", PreferResourceID: true},
	}
}

func logAnalyticsObjectCollectionRuleListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path", LookupPaths: logAnalyticsObjectCollectionRuleNamespaceLookupPaths()},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "name", "metadataName"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func logAnalyticsObjectCollectionRuleUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path", LookupPaths: logAnalyticsObjectCollectionRuleNamespaceLookupPaths()},
		{FieldName: "LogAnalyticsObjectCollectionRuleId", RequestName: "logAnalyticsObjectCollectionRuleId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateLogAnalyticsObjectCollectionRuleDetails", RequestName: "UpdateLogAnalyticsObjectCollectionRuleDetails", Contribution: "body"},
	}
}

func logAnalyticsObjectCollectionRuleDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path", LookupPaths: logAnalyticsObjectCollectionRuleNamespaceLookupPaths()},
		{FieldName: "LogAnalyticsObjectCollectionRuleId", RequestName: "logAnalyticsObjectCollectionRuleId", Contribution: "path", PreferResourceID: true},
	}
}

func logAnalyticsObjectCollectionRuleNamespaceLookupPaths() []string {
	return []string{"status.osNamespace", "spec.osNamespace", "osNamespace"}
}

func guardLogAnalyticsObjectCollectionRuleExistingBeforeCreate(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("LogAnalyticsObjectCollectionRule resource is nil")
	}
	if strings.TrimSpace(resource.Spec.OsNamespace) == "" ||
		strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.Name) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildLogAnalyticsObjectCollectionRuleCreateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
	_ string,
) (any, error) {
	if resource == nil {
		return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleDetails{}, fmt.Errorf("LogAnalyticsObjectCollectionRule resource is nil")
	}
	if err := validateLogAnalyticsObjectCollectionRuleSpec(resource.Spec); err != nil {
		return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleDetails{}, err
	}

	details := loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleDetails{
		Name:                      stringPtr(resource.Spec.Name),
		CompartmentId:             stringPtr(resource.Spec.CompartmentId),
		OsNamespace:               stringPtr(resource.Spec.OsNamespace),
		OsBucketName:              stringPtr(resource.Spec.OsBucketName),
		LogGroupId:                stringPtr(resource.Spec.LogGroupId),
		IsEnabled:                 common.Bool(resource.Spec.IsEnabled),
		IsForceHistoricCollection: common.Bool(resource.Spec.IsForceHistoricCollection),
		Overrides:                 propertyOverridesFromSpec(resource.Spec.Overrides),
		ObjectNameFilters:         copyStrings(resource.Spec.ObjectNameFilters),
		DefinedTags:               definedTagsFromSpec(resource.Spec.DefinedTags),
		FreeformTags:              maps.Clone(resource.Spec.FreeformTags),
		CollectionType:            loganalyticssdk.ObjectCollectionRuleCollectionTypesEnum(strings.TrimSpace(resource.Spec.CollectionType)),
		LogSetKey:                 loganalyticssdk.LogSetKeyTypesEnum(strings.TrimSpace(resource.Spec.LogSetKey)),
		LogType:                   loganalyticssdk.LogTypesEnum(strings.TrimSpace(resource.Spec.LogType)),
		StreamCursorType:          loganalyticssdk.StreamCursorTypesEnum(strings.TrimSpace(resource.Spec.StreamCursorType)),
	}
	assignOptionalCreateStrings(&details, resource.Spec)
	streamCursorTime, err := sdkTimeFromRFC3339("streamCursorTime", resource.Spec.StreamCursorTime)
	if err != nil {
		return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleDetails{}, err
	}
	details.StreamCursorTime = streamCursorTime
	return details, nil
}

func assignOptionalCreateStrings(
	details *loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
) {
	details.Description = optionalStringPtr(spec.Description)
	details.PollSince = optionalStringPtr(spec.PollSince)
	details.PollTill = optionalStringPtr(spec.PollTill)
	details.LogSourceName = optionalStringPtr(spec.LogSourceName)
	details.EntityId = optionalStringPtr(spec.EntityId)
	details.CharEncoding = optionalStringPtr(spec.CharEncoding)
	details.Timezone = optionalStringPtr(spec.Timezone)
	details.LogSet = optionalStringPtr(spec.LogSet)
	details.LogSetExtRegex = optionalStringPtr(spec.LogSetExtRegex)
	details.StreamId = optionalStringPtr(spec.StreamId)
}

func buildLogAnalyticsObjectCollectionRuleUpdateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails{}, false, fmt.Errorf("LogAnalyticsObjectCollectionRule resource is nil")
	}
	if err := validateLogAnalyticsObjectCollectionRuleSpec(resource.Spec); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails{}, false, err
	}
	current, ok := logAnalyticsObjectCollectionRuleFromResponse(currentResponse)
	if !ok {
		return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails{}, false, fmt.Errorf("current LogAnalyticsObjectCollectionRule response does not expose a LogAnalyticsObjectCollectionRule body")
	}
	if err := validateLogAnalyticsObjectCollectionRuleCreateOnlyDrift(resource.Spec, current); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails{}, false, err
	}

	details := loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails{}
	updateNeeded, err := applyLogAnalyticsObjectCollectionRuleUpdates(&details, resource.Spec, current)
	if err != nil {
		return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails{}, false, err
	}
	return details, updateNeeded, nil
}

func applyLogAnalyticsObjectCollectionRuleUpdates(
	details *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) (bool, error) {
	updateNeeded := applyLogAnalyticsObjectCollectionRuleStringUpdates(details, spec, current)
	updateNeeded = applyLogAnalyticsObjectCollectionRuleBoolUpdate(details, spec, current) || updateNeeded
	updateNeeded = applyLogAnalyticsObjectCollectionRuleEnumUpdates(details, spec, current) || updateNeeded
	updateNeeded = applyLogAnalyticsObjectCollectionRuleCollectionUpdates(details, spec, current) || updateNeeded
	timeUpdateNeeded, err := applyLogAnalyticsObjectCollectionRuleTimeUpdate(details, spec, current)
	if err != nil {
		return false, err
	}
	updateNeeded = timeUpdateNeeded || updateNeeded
	return updateNeeded, nil
}

func applyLogAnalyticsObjectCollectionRuleStringUpdates(
	details *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) bool {
	updateNeeded := false
	if desired, ok := desiredStringUpdate(spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.LogGroupId, current.LogGroupId); ok {
		details.LogGroupId = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.LogSourceName, current.LogSourceName); ok {
		details.LogSourceName = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.EntityId, current.EntityId); ok {
		details.EntityId = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.CharEncoding, current.CharEncoding); ok {
		details.CharEncoding = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.Timezone, current.Timezone); ok {
		details.Timezone = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.LogSet, current.LogSet); ok {
		details.LogSet = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.LogSetExtRegex, current.LogSetExtRegex); ok {
		details.LogSetExtRegex = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(spec.StreamId, current.StreamId); ok {
		details.StreamId = desired
		updateNeeded = true
	}
	return updateNeeded
}

func applyLogAnalyticsObjectCollectionRuleBoolUpdate(
	details *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) bool {
	if current.IsEnabled != nil && *current.IsEnabled == spec.IsEnabled {
		return false
	}
	details.IsEnabled = common.Bool(spec.IsEnabled)
	return true
}

func applyLogAnalyticsObjectCollectionRuleEnumUpdates(
	details *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) bool {
	updateNeeded := false
	if desired, ok := desiredLogSetKeyUpdate(spec.LogSetKey, current.LogSetKey); ok {
		details.LogSetKey = desired
		updateNeeded = true
	}
	if desired, ok := desiredStreamCursorTypeUpdate(spec.StreamCursorType, current.StreamCursorType); ok {
		details.StreamCursorType = desired
		updateNeeded = true
	}
	return updateNeeded
}

func applyLogAnalyticsObjectCollectionRuleCollectionUpdates(
	details *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) bool {
	updateNeeded := false
	if spec.Overrides != nil {
		desired := propertyOverridesFromSpec(spec.Overrides)
		if !reflect.DeepEqual(desired, current.Overrides) {
			details.Overrides = desired
			updateNeeded = true
		}
	}
	if spec.ObjectNameFilters != nil && !reflect.DeepEqual(spec.ObjectNameFilters, current.ObjectNameFilters) {
		details.ObjectNameFilters = copyStrings(spec.ObjectNameFilters)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := definedTagsFromSpec(spec.DefinedTags)
		if !reflect.DeepEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	return updateNeeded
}

func applyLogAnalyticsObjectCollectionRuleTimeUpdate(
	details *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) (bool, error) {
	desired, err := sdkTimeFromRFC3339("streamCursorTime", spec.StreamCursorTime)
	if err != nil || desired == nil {
		return false, err
	}
	if current.StreamCursorTime != nil && current.StreamCursorTime.Equal(desired.Time) {
		return false, nil
	}
	details.StreamCursorTime = desired
	return true, nil
}

func validateLogAnalyticsObjectCollectionRuleSpec(spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec) error {
	var problems []string
	for field, value := range map[string]string{
		"name":          spec.Name,
		"compartmentId": spec.CompartmentId,
		"osNamespace":   spec.OsNamespace,
		"osBucketName":  spec.OsBucketName,
		"logGroupId":    spec.LogGroupId,
	} {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("%s is required", field))
		}
	}
	if _, err := sdkTimeFromRFC3339("streamCursorTime", spec.StreamCursorTime); err != nil {
		problems = append(problems, err.Error())
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("LogAnalyticsObjectCollectionRule spec is invalid: %s", strings.Join(problems, "; "))
}

func validateLogAnalyticsObjectCollectionRuleCreateOnlyDriftForResponse(
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("LogAnalyticsObjectCollectionRule resource is nil")
	}
	current, ok := logAnalyticsObjectCollectionRuleFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current LogAnalyticsObjectCollectionRule response does not expose a LogAnalyticsObjectCollectionRule body")
	}
	return validateLogAnalyticsObjectCollectionRuleCreateOnlyDrift(resource.Spec, current)
}

func validateLogAnalyticsObjectCollectionRuleCreateOnlyDrift(
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) error {
	drift := logAnalyticsObjectCollectionRuleCreateOnlyStringDrift(spec, current)
	if current.IsForceHistoricCollection != nil && spec.IsForceHistoricCollection != *current.IsForceHistoricCollection {
		drift = append(drift, "isForceHistoricCollection")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("LogAnalyticsObjectCollectionRule create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func logAnalyticsObjectCollectionRuleCreateOnlyStringDrift(
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	current loganalyticssdk.LogAnalyticsObjectCollectionRule,
) []string {
	checks := []struct {
		name     string
		desired  string
		current  string
		required bool
	}{
		{name: "name", desired: spec.Name, current: stringPtrValue(current.Name), required: true},
		{name: "compartmentId", desired: spec.CompartmentId, current: stringPtrValue(current.CompartmentId), required: true},
		{name: "osNamespace", desired: spec.OsNamespace, current: stringPtrValue(current.OsNamespace), required: true},
		{name: "osBucketName", desired: spec.OsBucketName, current: stringPtrValue(current.OsBucketName), required: true},
		{name: "collectionType", desired: spec.CollectionType, current: string(current.CollectionType)},
		{name: "pollSince", desired: spec.PollSince, current: stringPtrValue(current.PollSince)},
		{name: "pollTill", desired: spec.PollTill, current: stringPtrValue(current.PollTill)},
		{name: "logType", desired: spec.LogType, current: string(current.LogType)},
	}

	var drift []string
	for _, check := range checks {
		desired := strings.TrimSpace(check.desired)
		if desired == "" && !check.required {
			continue
		}
		if desired != check.current {
			drift = append(drift, check.name)
		}
	}
	return drift
}

func wrapLogAnalyticsObjectCollectionRuleListCall(hooks *LogAnalyticsObjectCollectionRuleRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
		return listLogAnalyticsObjectCollectionRulePages(ctx, call, request)
	}
}

func listLogAnalyticsObjectCollectionRulePages(
	ctx context.Context,
	call func(context.Context, loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error),
	request loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest,
) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
	var combined loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse
	seenPages := map[string]struct{}{}
	for {
		response, err := call(ctx, request)
		if err != nil {
			return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{}, err
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.Items = append(combined.Items, response.Items...)

		nextPage := stringPtrValue(response.OpcNextPage)
		if nextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{}, fmt.Errorf("LogAnalyticsObjectCollectionRule list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = common.String(nextPage)
		combined.OpcNextPage = common.String(nextPage)
	}
}

func applyLogAnalyticsObjectCollectionRuleDeleteOutcome(
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !hasPendingLogAnalyticsObjectCollectionRuleDelete(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage != generatedruntime.DeleteConfirmStageAfterRequest &&
		stage != generatedruntime.DeleteConfirmStageAlreadyPending {
		return generatedruntime.DeleteOutcome{}, nil
	}
	current, ok := logAnalyticsObjectCollectionRuleFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	switch current.LifecycleState {
	case loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive, loganalyticssdk.ObjectCollectionRuleLifecycleStatesInactive, "":
		markLogAnalyticsObjectCollectionRuleTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func hasPendingLogAnalyticsObjectCollectionRuleDelete(resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markLogAnalyticsObjectCollectionRuleTerminating(
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
	response any,
) {
	if resource == nil {
		return
	}
	message := "OCI resource delete is in progress"
	rawStatus := ""
	if current, ok := logAnalyticsObjectCollectionRuleFromResponse(response); ok {
		rawStatus = string(current.LifecycleState)
		if detail := stringPtrValue(current.LifecycleDetails); detail != "" {
			message = detail
		}
	}

	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func handleLogAnalyticsObjectCollectionRuleDeleteError(
	resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return logAnalyticsObjectCollectionRuleAmbiguousNotFoundError{
		message:      fmt.Sprintf("LogAnalyticsObjectCollectionRule delete returned ambiguous 404 NotAuthorizedOrNotFound: %s", err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

type logAnalyticsObjectCollectionRuleAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e logAnalyticsObjectCollectionRuleAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e logAnalyticsObjectCollectionRuleAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func clearTrackedLogAnalyticsObjectCollectionRuleIdentity(resource *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule) {
	if resource == nil {
		return
	}
	resource.Status = loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleStatus{}
}

func logAnalyticsObjectCollectionRuleFromResponse(response any) (loganalyticssdk.LogAnalyticsObjectCollectionRule, bool) {
	if current, ok := logAnalyticsObjectCollectionRuleFromBody(response); ok {
		return current, true
	}
	return logAnalyticsObjectCollectionRuleFromOperationResponse(response)
}

func logAnalyticsObjectCollectionRuleFromBody(response any) (loganalyticssdk.LogAnalyticsObjectCollectionRule, bool) {
	switch current := response.(type) {
	case loganalyticssdk.LogAnalyticsObjectCollectionRule:
		return current, true
	case *loganalyticssdk.LogAnalyticsObjectCollectionRule:
		if current == nil {
			return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
		}
		return *current, true
	case loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary:
		return logAnalyticsObjectCollectionRuleFromSummary(current), true
	case *loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary:
		if current == nil {
			return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
		}
		return logAnalyticsObjectCollectionRuleFromSummary(*current), true
	default:
		return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
	}
}

func logAnalyticsObjectCollectionRuleFromOperationResponse(response any) (loganalyticssdk.LogAnalyticsObjectCollectionRule, bool) {
	switch current := response.(type) {
	case loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse:
		return current.LogAnalyticsObjectCollectionRule, true
	case *loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
		}
		return current.LogAnalyticsObjectCollectionRule, true
	case loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse:
		return current.LogAnalyticsObjectCollectionRule, true
	case *loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
		}
		return current.LogAnalyticsObjectCollectionRule, true
	case loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse:
		return current.LogAnalyticsObjectCollectionRule, true
	case *loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
		}
		return current.LogAnalyticsObjectCollectionRule, true
	default:
		return loganalyticssdk.LogAnalyticsObjectCollectionRule{}, false
	}
}

func logAnalyticsObjectCollectionRuleFromSummary(summary loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary) loganalyticssdk.LogAnalyticsObjectCollectionRule {
	return loganalyticssdk.LogAnalyticsObjectCollectionRule{
		Id:                summary.Id,
		Name:              summary.Name,
		CompartmentId:     summary.CompartmentId,
		OsNamespace:       summary.OsNamespace,
		OsBucketName:      summary.OsBucketName,
		CollectionType:    summary.CollectionType,
		LifecycleState:    summary.LifecycleState,
		TimeCreated:       summary.TimeCreated,
		TimeUpdated:       summary.TimeUpdated,
		IsEnabled:         summary.IsEnabled,
		Description:       summary.Description,
		LifecycleDetails:  summary.LifecycleDetails,
		ObjectNameFilters: summary.ObjectNameFilters,
		LogType:           summary.LogType,
		StreamId:          summary.StreamId,
		DefinedTags:       summary.DefinedTags,
		FreeformTags:      summary.FreeformTags,
	}
}

func propertyOverridesFromSpec(
	spec map[string][]loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleOverrides,
) map[string][]loganalyticssdk.PropertyOverride {
	if spec == nil {
		return nil
	}
	converted := make(map[string][]loganalyticssdk.PropertyOverride, len(spec))
	for key, overrides := range spec {
		converted[key] = make([]loganalyticssdk.PropertyOverride, 0, len(overrides))
		for _, override := range overrides {
			converted[key] = append(converted[key], loganalyticssdk.PropertyOverride{
				MatchType:     optionalStringPtr(override.MatchType),
				MatchValue:    optionalStringPtr(override.MatchValue),
				PropertyName:  optionalStringPtr(override.PropertyName),
				PropertyValue: optionalStringPtr(override.PropertyValue),
			})
		}
	}
	return converted
}

func definedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func desiredStringUpdate(spec string, current *string) (*string, bool) {
	desired := strings.TrimSpace(spec)
	currentValue := stringPtrValue(current)
	if desired == currentValue {
		return nil, false
	}
	if desired == "" && current == nil {
		return nil, false
	}
	return common.String(desired), true
}

func desiredLogSetKeyUpdate(spec string, current loganalyticssdk.LogSetKeyTypesEnum) (loganalyticssdk.LogSetKeyTypesEnum, bool) {
	desired := strings.TrimSpace(spec)
	if desired == "" || desired == string(current) {
		return "", false
	}
	return loganalyticssdk.LogSetKeyTypesEnum(desired), true
}

func desiredStreamCursorTypeUpdate(spec string, current loganalyticssdk.StreamCursorTypesEnum) (loganalyticssdk.StreamCursorTypesEnum, bool) {
	desired := strings.TrimSpace(spec)
	if desired == "" || desired == string(current) {
		return "", false
	}
	return loganalyticssdk.StreamCursorTypesEnum(desired), true
}

func sdkTimeFromRFC3339(fieldName string, value string) (*common.SDKTime, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339 timestamp: %w", fieldName, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func copyStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func optionalStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func stringPtr(value string) *string {
	return common.String(strings.TrimSpace(value))
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ error = logAnalyticsObjectCollectionRuleAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = logAnalyticsObjectCollectionRuleAmbiguousNotFoundError{}
