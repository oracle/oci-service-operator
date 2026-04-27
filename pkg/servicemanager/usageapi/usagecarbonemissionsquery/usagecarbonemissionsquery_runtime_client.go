/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package usagecarbonemissionsquery

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type synchronousUsageCarbonEmissionsQueryServiceClient struct {
	delegate UsageCarbonEmissionsQueryServiceClient
	log      loggerutil.OSOKLogger
}

func init() {
	registerUsageCarbonEmissionsQueryRuntimeHooksMutator(func(manager *UsageCarbonEmissionsQueryServiceManager, hooks *UsageCarbonEmissionsQueryRuntimeHooks) {
		applyUsageCarbonEmissionsQueryRuntimeHooks(manager, hooks)
	})
}

func applyUsageCarbonEmissionsQueryRuntimeHooks(manager *UsageCarbonEmissionsQueryServiceManager, hooks *UsageCarbonEmissionsQueryRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.Semantics = newUsageCarbonEmissionsQueryRuntimeSemantics()
	hooks.BuildCreateBody = buildUsageCarbonEmissionsQueryCreateBody
	hooks.BuildUpdateBody = buildUsageCarbonEmissionsQueryUpdateBody
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate UsageCarbonEmissionsQueryServiceClient) UsageCarbonEmissionsQueryServiceClient {
		return &synchronousUsageCarbonEmissionsQueryServiceClient{
			delegate: delegate,
			log:      manager.Log,
		}
	})
}

func newUsageCarbonEmissionsQueryRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "usageapi",
		FormalSlug:    "usagecarbonemissionsquery",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"PROVISIONING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"queryDefinition.costAnalysisUi.graph",
				"queryDefinition.costAnalysisUi.isCumulativeGraph",
				"queryDefinition.displayName",
				"queryDefinition.reportQuery.compartmentDepth",
				"queryDefinition.reportQuery.dateRangeName",
				"queryDefinition.reportQuery.filter.dimensions",
				"queryDefinition.reportQuery.filter.operator",
				"queryDefinition.reportQuery.filter.tags",
				"queryDefinition.reportQuery.groupBy",
				"queryDefinition.reportQuery.groupByTag",
				"queryDefinition.reportQuery.isAggregateByTime",
				"queryDefinition.reportQuery.tenantId",
				"queryDefinition.reportQuery.timeUsageEnded",
				"queryDefinition.reportQuery.timeUsageStarted",
				"queryDefinition.version",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"queryDefinition"},
			ForceNew:      []string{"compartmentId"},
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

func buildUsageCarbonEmissionsQueryCreateBody(
	_ context.Context,
	resource *usageapiv1beta1.UsageCarbonEmissionsQuery,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("UsageCarbonEmissionsQuery resource is nil")
	}

	queryDefinition, err := usageCarbonEmissionsQueryDefinition(resource.Spec.QueryDefinition, nil)
	if err != nil {
		return nil, err
	}
	return usageapisdk.CreateUsageCarbonEmissionsQueryDetails{
		CompartmentId:   common.String(resource.Spec.CompartmentId),
		QueryDefinition: queryDefinition,
	}, nil
}

func buildUsageCarbonEmissionsQueryUpdateBody(
	_ context.Context,
	resource *usageapiv1beta1.UsageCarbonEmissionsQuery,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("UsageCarbonEmissionsQuery resource is nil")
	}

	currentDefinition := currentUsageCarbonEmissionsQueryDefinition(currentResponse)
	queryDefinition, err := usageCarbonEmissionsQueryDefinition(resource.Spec.QueryDefinition, currentDefinition)
	if err != nil {
		return nil, false, err
	}
	if reflect.DeepEqual(queryDefinition, currentDefinition) {
		return nil, false, nil
	}
	return usageapisdk.UpdateUsageCarbonEmissionsQueryDetails{
		QueryDefinition: queryDefinition,
	}, true, nil
}

func usageCarbonEmissionsQueryDefinition(
	spec usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinition,
	current *usageapisdk.UsageCarbonEmissionsQueryDefinition,
) (*usageapisdk.UsageCarbonEmissionsQueryDefinition, error) {
	reportQuery, err := usageCarbonEmissionsReportQuery(spec.ReportQuery, currentReportQuery(current))
	if err != nil {
		return nil, err
	}
	return &usageapisdk.UsageCarbonEmissionsQueryDefinition{
		DisplayName:    common.String(spec.DisplayName),
		ReportQuery:    reportQuery,
		CostAnalysisUI: usageCarbonEmissionsCostAnalysisUI(spec.CostAnalysisUI, currentCostAnalysisUI(current)),
		Version:        common.Int(spec.Version),
	}, nil
}

func usageCarbonEmissionsReportQuery(
	spec usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionReportQuery,
	current *usageapisdk.UsageCarbonEmissionsReportQuery,
) (*usageapisdk.UsageCarbonEmissionsReportQuery, error) {
	reportQuery := &usageapisdk.UsageCarbonEmissionsReportQuery{
		TenantId: common.String(spec.TenantId),
	}
	if strings.TrimSpace(spec.TimeUsageStarted) != "" {
		value, err := sdkTimeFromRFC3339("queryDefinition.reportQuery.timeUsageStarted", spec.TimeUsageStarted)
		if err != nil {
			return nil, err
		}
		reportQuery.TimeUsageStarted = value
	}
	if strings.TrimSpace(spec.TimeUsageEnded) != "" {
		value, err := sdkTimeFromRFC3339("queryDefinition.reportQuery.timeUsageEnded", spec.TimeUsageEnded)
		if err != nil {
			return nil, err
		}
		reportQuery.TimeUsageEnded = value
	}
	if spec.IsAggregateByTime || currentBool(currentIsAggregateByTime(current)) {
		reportQuery.IsAggregateByTime = common.Bool(spec.IsAggregateByTime)
	}
	if len(spec.GroupBy) > 0 {
		reportQuery.GroupBy = append([]string(nil), spec.GroupBy...)
	}
	if len(spec.GroupByTag) > 0 {
		reportQuery.GroupByTag = usageCarbonEmissionsTags(spec.GroupByTag)
	}
	if spec.CompartmentDepth != 0 {
		reportQuery.CompartmentDepth = common.Int(spec.CompartmentDepth)
	}
	if filter := usageCarbonEmissionsFilter(spec.Filter); filter != nil {
		reportQuery.Filter = filter
	}
	if strings.TrimSpace(spec.DateRangeName) != "" {
		reportQuery.DateRangeName = usageapisdk.UsageCarbonEmissionsReportQueryDateRangeNameEnum(spec.DateRangeName)
	}
	return reportQuery, nil
}

func usageCarbonEmissionsCostAnalysisUI(
	spec usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionCostAnalysisUI,
	current *usageapisdk.CostAnalysisUi,
) *usageapisdk.CostAnalysisUi {
	costAnalysisUI := &usageapisdk.CostAnalysisUi{}
	if strings.TrimSpace(spec.Graph) != "" {
		costAnalysisUI.Graph = usageapisdk.CostAnalysisUiGraphEnum(spec.Graph)
	}
	if spec.IsCumulativeGraph || currentBool(currentIsCumulativeGraph(current)) {
		costAnalysisUI.IsCumulativeGraph = common.Bool(spec.IsCumulativeGraph)
	}
	return costAnalysisUI
}

func usageCarbonEmissionsFilter(spec usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionReportQueryFilter) *usageapisdk.Filter {
	filter := usageapisdk.Filter{}
	if strings.TrimSpace(spec.Operator) != "" {
		filter.Operator = usageapisdk.FilterOperatorEnum(spec.Operator)
	}
	if len(spec.Dimensions) > 0 {
		filter.Dimensions = make([]usageapisdk.Dimension, 0, len(spec.Dimensions))
		for _, dimension := range spec.Dimensions {
			filter.Dimensions = append(filter.Dimensions, usageapisdk.Dimension{
				Key:   common.String(dimension.Key),
				Value: common.String(dimension.Value),
			})
		}
	}
	if len(spec.Tags) > 0 {
		filter.Tags = usageCarbonEmissionsFilterTags(spec.Tags)
	}
	if filter.Operator == "" && len(filter.Dimensions) == 0 && len(filter.Tags) == 0 {
		return nil
	}
	return &filter
}

func usageCarbonEmissionsTags(tags []usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionReportQueryGroupByTag) []usageapisdk.Tag {
	sdkTags := make([]usageapisdk.Tag, 0, len(tags))
	for _, tag := range tags {
		sdkTags = append(sdkTags, usageCarbonEmissionsTag(tag.Namespace, tag.Key, tag.Value))
	}
	return sdkTags
}

func usageCarbonEmissionsFilterTags(tags []usageapiv1beta1.UsageCarbonEmissionsQueryQueryDefinitionReportQueryFilterTag) []usageapisdk.Tag {
	sdkTags := make([]usageapisdk.Tag, 0, len(tags))
	for _, tag := range tags {
		sdkTags = append(sdkTags, usageCarbonEmissionsTag(tag.Namespace, tag.Key, tag.Value))
	}
	return sdkTags
}

func usageCarbonEmissionsTag(namespace string, key string, value string) usageapisdk.Tag {
	tag := usageapisdk.Tag{}
	if strings.TrimSpace(namespace) != "" {
		tag.Namespace = common.String(namespace)
	}
	if strings.TrimSpace(key) != "" {
		tag.Key = common.String(key)
	}
	if strings.TrimSpace(value) != "" {
		tag.Value = common.String(value)
	}
	return tag
}

func sdkTimeFromRFC3339(fieldName string, value string) (*common.SDKTime, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339: %w", fieldName, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func currentUsageCarbonEmissionsQueryDefinition(response any) *usageapisdk.UsageCarbonEmissionsQueryDefinition {
	switch current := response.(type) {
	case usageapisdk.CreateUsageCarbonEmissionsQueryResponse:
		return current.UsageCarbonEmissionsQuery.QueryDefinition
	case usageapisdk.GetUsageCarbonEmissionsQueryResponse:
		return current.UsageCarbonEmissionsQuery.QueryDefinition
	case usageapisdk.UpdateUsageCarbonEmissionsQueryResponse:
		return current.UsageCarbonEmissionsQuery.QueryDefinition
	case usageapisdk.UsageCarbonEmissionsQuery:
		return current.QueryDefinition
	case usageapisdk.UsageCarbonEmissionsQuerySummary:
		return current.QueryDefinition
	default:
		return nil
	}
}

func currentReportQuery(current *usageapisdk.UsageCarbonEmissionsQueryDefinition) *usageapisdk.UsageCarbonEmissionsReportQuery {
	if current == nil {
		return nil
	}
	return current.ReportQuery
}

func currentCostAnalysisUI(current *usageapisdk.UsageCarbonEmissionsQueryDefinition) *usageapisdk.CostAnalysisUi {
	if current == nil {
		return nil
	}
	return current.CostAnalysisUI
}

func currentIsAggregateByTime(current *usageapisdk.UsageCarbonEmissionsReportQuery) *bool {
	if current == nil {
		return nil
	}
	return current.IsAggregateByTime
}

func currentIsCumulativeGraph(current *usageapisdk.CostAnalysisUi) *bool {
	if current == nil {
		return nil
	}
	return current.IsCumulativeGraph
}

func currentBool(value *bool) bool {
	return value != nil
}

func (c *synchronousUsageCarbonEmissionsQueryServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *usageapiv1beta1.UsageCarbonEmissionsQuery,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !response.ShouldRequeue || resource == nil {
		return response, err
	}

	status := &resource.Status.OsokStatus
	if status.Async.Current != nil {
		return response, err
	}
	if status.Reason != string(shared.Provisioning) && status.Reason != string(shared.Updating) {
		return response, err
	}

	now := metav1.Now()
	servicemanager.ClearAsyncOperation(status)
	status.Reason = string(shared.Active)
	status.UpdatedAt = &now
	if strings.TrimSpace(status.Message) == "" {
		status.Message = resource.Status.QueryDefinition.DisplayName
		if strings.TrimSpace(status.Message) == "" {
			status.Message = resource.Spec.QueryDefinition.DisplayName
		}
	}
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Active,
		corev1.ConditionTrue,
		"",
		status.Message,
		c.log,
	)

	response.ShouldRequeue = false
	response.RequeueDuration = 0
	return response, nil
}

func (c *synchronousUsageCarbonEmissionsQueryServiceClient) Delete(ctx context.Context, resource *usageapiv1beta1.UsageCarbonEmissionsQuery) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
