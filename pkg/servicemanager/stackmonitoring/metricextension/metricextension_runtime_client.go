/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package metricextension

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
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

type metricExtensionOCIClient interface {
	CreateMetricExtension(context.Context, stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error)
	GetMetricExtension(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error)
	ListMetricExtensions(context.Context, stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error)
	UpdateMetricExtension(context.Context, stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error)
	DeleteMetricExtension(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error)
}

type metricExtensionListCall func(context.Context, stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error)

type metricExtensionRuntimeClient struct {
	hooks   MetricExtensionRuntimeHooks
	list    metricExtensionListCall
	initErr error
}

var _ MetricExtensionServiceClient = (*metricExtensionRuntimeClient)(nil)

type metricExtensionAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e metricExtensionAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e metricExtensionAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type metricExtensionAuthShapedConfirmRead struct {
	err error
}

func (e metricExtensionAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("MetricExtension delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e metricExtensionAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func init() {
	registerMetricExtensionRuntimeHooksMutator(func(manager *MetricExtensionServiceManager, hooks *MetricExtensionRuntimeHooks) {
		applyMetricExtensionRuntimeHooks(manager, hooks)
	})
}

func applyMetricExtensionRuntimeHooks(manager *MetricExtensionServiceManager, hooks *MetricExtensionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newMetricExtensionRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *stackmonitoringv1beta1.MetricExtension, namespace string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("MetricExtension resource is nil")
		}
		resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, metricExtensionCredentialClient(manager), namespace)
		if err != nil {
			return nil, err
		}
		return metricExtensionResolvedSpecValue(resource, resolved)
	}
	hooks.BuildUpdateBody = func(ctx context.Context, resource *stackmonitoringv1beta1.MetricExtension, namespace string, current any) (any, bool, error) {
		return buildMetricExtensionUpdateBody(ctx, resource, namespace, current, metricExtensionCredentialClient(manager))
	}
	hooks.List.Fields = metricExtensionListFields()
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *stackmonitoringv1beta1.MetricExtension, currentID string) (any, error) {
		return confirmMetricExtensionDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleMetricExtensionDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleMetricExtensionDeleteConfirmReadOutcome
	if hooks.List.Call != nil {
		hooks.List.Call = listMetricExtensionsAllPages(hooks.List.Call)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MetricExtensionServiceClient) MetricExtensionServiceClient {
		return &metricExtensionRuntimeClient{
			hooks:   *hooks,
			list:    hooks.List.Call,
			initErr: metricExtensionGeneratedDelegateInitError(delegate),
		}
	})
}

func newMetricExtensionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client metricExtensionOCIClient,
) MetricExtensionServiceClient {
	hooks := newMetricExtensionRuntimeHooksWithOCIClient(client)
	applyMetricExtensionRuntimeHooks(&MetricExtensionServiceManager{Log: log}, &hooks)
	manager := &MetricExtensionServiceManager{Log: log}
	delegate := defaultMetricExtensionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.MetricExtension](
			buildMetricExtensionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapMetricExtensionGeneratedClient(hooks, delegate)
}

func metricExtensionGeneratedDelegateInitError(delegate MetricExtensionServiceClient) error {
	if delegate == nil {
		return nil
	}

	var resource *stackmonitoringv1beta1.MetricExtension
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || isMetricExtensionNilResourceProbeError(err) {
		return nil
	}
	return err
}

func isMetricExtensionNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

func newMetricExtensionRuntimeHooksWithOCIClient(client metricExtensionOCIClient) MetricExtensionRuntimeHooks {
	return MetricExtensionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*stackmonitoringv1beta1.MetricExtension]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*stackmonitoringv1beta1.MetricExtension]{},
		StatusHooks:     generatedruntime.StatusHooks[*stackmonitoringv1beta1.MetricExtension]{},
		ParityHooks:     generatedruntime.ParityHooks[*stackmonitoringv1beta1.MetricExtension]{},
		Async:           generatedruntime.AsyncHooks[*stackmonitoringv1beta1.MetricExtension]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*stackmonitoringv1beta1.MetricExtension]{},
		Create: runtimeOperationHooks[stackmonitoringsdk.CreateMetricExtensionRequest, stackmonitoringsdk.CreateMetricExtensionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateMetricExtensionDetails", RequestName: "CreateMetricExtensionDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error) {
				if client == nil {
					return stackmonitoringsdk.CreateMetricExtensionResponse{}, fmt.Errorf("MetricExtension OCI client is nil")
				}
				return client.CreateMetricExtension(ctx, request)
			},
		},
		Get: runtimeOperationHooks[stackmonitoringsdk.GetMetricExtensionRequest, stackmonitoringsdk.GetMetricExtensionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "MetricExtensionId", RequestName: "metricExtensionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
				if client == nil {
					return stackmonitoringsdk.GetMetricExtensionResponse{}, fmt.Errorf("MetricExtension OCI client is nil")
				}
				return client.GetMetricExtension(ctx, request)
			},
		},
		List: runtimeOperationHooks[stackmonitoringsdk.ListMetricExtensionsRequest, stackmonitoringsdk.ListMetricExtensionsResponse]{
			Fields: metricExtensionListFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
				if client == nil {
					return stackmonitoringsdk.ListMetricExtensionsResponse{}, fmt.Errorf("MetricExtension OCI client is nil")
				}
				return client.ListMetricExtensions(ctx, request)
			},
		},
		Update: runtimeOperationHooks[stackmonitoringsdk.UpdateMetricExtensionRequest, stackmonitoringsdk.UpdateMetricExtensionResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "MetricExtensionId", RequestName: "metricExtensionId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateMetricExtensionDetails", RequestName: "UpdateMetricExtensionDetails", Contribution: "body", PreferResourceID: false},
			},
			Call: func(ctx context.Context, request stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error) {
				if client == nil {
					return stackmonitoringsdk.UpdateMetricExtensionResponse{}, fmt.Errorf("MetricExtension OCI client is nil")
				}
				return client.UpdateMetricExtension(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[stackmonitoringsdk.DeleteMetricExtensionRequest, stackmonitoringsdk.DeleteMetricExtensionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "MetricExtensionId", RequestName: "metricExtensionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
				if client == nil {
					return stackmonitoringsdk.DeleteMetricExtensionResponse{}, fmt.Errorf("MetricExtension OCI client is nil")
				}
				return client.DeleteMetricExtension(ctx, request)
			},
		},
		WrapGeneratedClient: []func(MetricExtensionServiceClient) MetricExtensionServiceClient{},
	}
}

func newMetricExtensionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "metricextension",
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
				string(stackmonitoringsdk.MetricExtensionLifeCycleStatesActive),
				string(stackmonitoringsdk.MetricExtensionLifeCycleDetailsDraft),
				string(stackmonitoringsdk.MetricExtensionLifeCycleDetailsPublished),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(stackmonitoringsdk.MetricExtensionLifeCycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "resourceType", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "description", "collectionRecurrences", "metricList", "queryProperties"},
			Mutable:         []string{"displayName", "description", "collectionRecurrences", "metricList", "queryProperties"},
			ForceNew:        []string{"name", "resourceType", "compartmentId"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func metricExtensionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
		{FieldName: "ResourceType", RequestName: "resourceType", Contribution: "query", PreferResourceID: false},
		{FieldName: "Name", RequestName: "name", Contribution: "query", PreferResourceID: false},
		{FieldName: "MetricExtensionId", RequestName: "metricExtensionId", Contribution: "query", PreferResourceID: true},
	}
}

func buildMetricExtensionUpdateBody(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
	namespace string,
	current any,
	credentialClient credhelper.CredentialClient,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("MetricExtension resource is nil")
	}
	resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credentialClient, namespace)
	if err != nil {
		return nil, false, err
	}
	resolved, err = metricExtensionResolvedSpecValue(resource, resolved)
	if err != nil {
		return nil, false, err
	}
	specValues, err := metricExtensionJSONMap(resolved)
	if err != nil {
		return nil, false, fmt.Errorf("decode MetricExtension desired state: %w", err)
	}
	currentValues, err := metricExtensionJSONMap(metricExtensionResponseBody(current))
	if err != nil {
		return nil, false, fmt.Errorf("decode MetricExtension observed state: %w", err)
	}

	body := metricExtensionMutableUpdateBody(specValues, currentValues)
	if len(body) == 0 {
		return nil, false, nil
	}
	return body, true, nil
}

func metricExtensionMutableUpdateBody(specValues map[string]any, currentValues map[string]any) map[string]any {
	body := make(map[string]any)
	for _, field := range metricExtensionMutableFields() {
		value, changed := metricExtensionMutableFieldChange(specValues, currentValues, field)
		if changed {
			body[field] = value
		}
	}
	return body
}

func metricExtensionMutableFieldChange(specValues map[string]any, currentValues map[string]any, field string) (any, bool) {
	specValue, ok := specValues[field]
	if !ok {
		return nil, false
	}
	specComparable, specMeaningful := metricExtensionComparableValue(specValue)
	if !specMeaningful {
		return nil, false
	}
	if currentValue, currentOK := currentValues[field]; currentOK {
		currentComparable, _ := metricExtensionComparableValue(currentValue)
		if reflect.DeepEqual(specComparable, currentComparable) {
			return nil, false
		}
	}
	return specComparable, true
}

func metricExtensionComparableValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		return metricExtensionComparableMap(typed)
	case []any:
		return metricExtensionComparableSlice(typed)
	case string:
		return metricExtensionComparableString(typed)
	default:
		return typed, true
	}
}

func metricExtensionComparableMap(values map[string]any) (any, bool) {
	result := make(map[string]any, len(values))
	for key, child := range values {
		normalized, ok := metricExtensionComparableValue(child)
		if ok {
			result[key] = normalized
		}
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func metricExtensionComparableSlice(values []any) (any, bool) {
	result := make([]any, 0, len(values))
	for _, child := range values {
		normalized, ok := metricExtensionComparableValue(child)
		if ok {
			result = append(result, normalized)
		}
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func metricExtensionComparableString(value string) (any, bool) {
	if strings.TrimSpace(value) == "" {
		return nil, false
	}
	return value, true
}

func metricExtensionMutableFields() []string {
	return []string{"displayName", "description", "collectionRecurrences", "metricList", "queryProperties"}
}

func metricExtensionResolvedSpecValue(
	resource *stackmonitoringv1beta1.MetricExtension,
	resolved any,
) (map[string]any, error) {
	if err := validateMetricExtensionQueryPropertiesJSONData(resource); err != nil {
		return nil, err
	}
	values, err := metricExtensionJSONMap(resolved)
	if err != nil {
		return nil, err
	}
	metricItems, _ := values["metricList"].([]any)
	for index := range metricItems {
		if index >= len(resource.Spec.MetricList) {
			break
		}
		item, ok := metricItems[index].(map[string]any)
		if !ok {
			continue
		}
		item["isDimension"] = resource.Spec.MetricList[index].IsDimension
		item["isHidden"] = resource.Spec.MetricList[index].IsHidden
	}
	query, _ := values["queryProperties"].(map[string]any)
	if query != nil {
		query["isMetricServiceEnabled"] = resource.Spec.QueryProperties.IsMetricServiceEnabled
	}
	return values, nil
}

func validateMetricExtensionQueryPropertiesJSONData(resource *stackmonitoringv1beta1.MetricExtension) error {
	if resource == nil {
		return fmt.Errorf("MetricExtension resource is nil")
	}
	if strings.TrimSpace(resource.Spec.QueryProperties.JsonData) == "" {
		return nil
	}
	return fmt.Errorf("MetricExtension spec.queryProperties.jsonData is not supported; use typed queryProperties fields instead")
}

func metricExtensionJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

func metricExtensionResponseBody(response any) any {
	switch typed := response.(type) {
	case stackmonitoringsdk.CreateMetricExtensionResponse:
		return typed.MetricExtension
	case stackmonitoringsdk.GetMetricExtensionResponse:
		return typed.MetricExtension
	case stackmonitoringsdk.UpdateMetricExtensionResponse:
		return typed.MetricExtension
	case stackmonitoringsdk.MetricExtension, stackmonitoringsdk.MetricExtensionSummary:
		return response
	default:
		return metricExtensionPointerResponseBody(response)
	}
}

func metricExtensionPointerResponseBody(response any) any {
	if metricExtensionAnyNil(response) {
		return nil
	}
	switch typed := response.(type) {
	case *stackmonitoringsdk.CreateMetricExtensionResponse:
		return typed.MetricExtension
	case *stackmonitoringsdk.GetMetricExtensionResponse:
		return typed.MetricExtension
	case *stackmonitoringsdk.UpdateMetricExtensionResponse:
		return typed.MetricExtension
	case *stackmonitoringsdk.MetricExtension:
		return *typed
	case *stackmonitoringsdk.MetricExtensionSummary:
		return *typed
	default:
		return response
	}
}

func metricExtensionAnyNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Ptr && reflected.IsNil()
}

func metricExtensionCreateDetails(body any) (stackmonitoringsdk.CreateMetricExtensionDetails, error) {
	var details stackmonitoringsdk.CreateMetricExtensionDetails
	if err := metricExtensionDecodeBody(body, &details); err != nil {
		return details, fmt.Errorf("decode MetricExtension create body: %w", err)
	}
	return details, nil
}

func metricExtensionUpdateDetails(body any) (stackmonitoringsdk.UpdateMetricExtensionDetails, error) {
	var details stackmonitoringsdk.UpdateMetricExtensionDetails
	if err := metricExtensionDecodeBody(body, &details); err != nil {
		return details, fmt.Errorf("decode MetricExtension update body: %w", err)
	}
	return details, nil
}

func metricExtensionDecodeBody(body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}

func validateMetricExtensionCreateOnlyDrift(
	resource *stackmonitoringv1beta1.MetricExtension,
	current stackmonitoringsdk.MetricExtension,
) error {
	checks := []struct {
		field string
		want  string
		got   string
	}{
		{field: "name", want: resource.Spec.Name, got: metricExtensionStringValue(current.Name)},
		{field: "resourceType", want: resource.Spec.ResourceType, got: metricExtensionStringValue(current.ResourceType)},
		{field: "compartmentId", want: resource.Spec.CompartmentId, got: metricExtensionStringValue(current.CompartmentId)},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.want) != strings.TrimSpace(check.got) {
			return fmt.Errorf("MetricExtension formal semantics require replacement when %s changes", check.field)
		}
	}
	return nil
}

func projectMetricExtensionStatus(resource *stackmonitoringv1beta1.MetricExtension, current stackmonitoringsdk.MetricExtension) {
	if resource == nil {
		return
	}
	recordMetricExtensionID(resource, metricExtensionID(current))
	resource.Status.Name = metricExtensionStringValue(current.Name)
	resource.Status.DisplayName = metricExtensionStringValue(current.DisplayName)
	resource.Status.ResourceType = metricExtensionStringValue(current.ResourceType)
	resource.Status.CompartmentId = metricExtensionStringValue(current.CompartmentId)
	resource.Status.TenantId = metricExtensionStringValue(current.TenantId)
	resource.Status.CollectionMethod = metricExtensionStringValue(current.CollectionMethod)
	resource.Status.Status = string(current.Status)
	resource.Status.CollectionRecurrences = metricExtensionStringValue(current.CollectionRecurrences)
	resource.Status.MetricList = metricExtensionStatusMetricList(current.MetricList)
	resource.Status.QueryProperties = metricExtensionStatusQueryProperties(current.QueryProperties)
	resource.Status.Description = metricExtensionStringValue(current.Description)
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.CreatedBy = metricExtensionStringValue(current.CreatedBy)
	resource.Status.LastUpdatedBy = metricExtensionStringValue(current.LastUpdatedBy)
	resource.Status.TimeCreated = metricExtensionSDKTimeString(current.TimeCreated)
	resource.Status.TimeUpdated = metricExtensionSDKTimeString(current.TimeUpdated)
	resource.Status.EnabledOnResources = metricExtensionStatusEnabledResources(current.EnabledOnResources)
	if current.EnabledOnResourcesCount != nil {
		resource.Status.EnabledOnResourcesCount = *current.EnabledOnResourcesCount
	}
	resource.Status.ResourceUri = metricExtensionStringValue(current.ResourceUri)
}

func metricExtensionStatusMetricList(items []stackmonitoringsdk.Metric) []stackmonitoringv1beta1.MetricExtensionMetricList {
	if len(items) == 0 {
		return nil
	}
	result := make([]stackmonitoringv1beta1.MetricExtensionMetricList, 0, len(items))
	for _, item := range items {
		result = append(result, stackmonitoringv1beta1.MetricExtensionMetricList{
			Name:              metricExtensionStringValue(item.Name),
			DataType:          string(item.DataType),
			DisplayName:       metricExtensionStringValue(item.DisplayName),
			IsDimension:       metricExtensionBoolValue(item.IsDimension),
			ComputeExpression: metricExtensionStringValue(item.ComputeExpression),
			IsHidden:          metricExtensionBoolValue(item.IsHidden),
			MetricCategory:    string(item.MetricCategory),
			Unit:              metricExtensionStringValue(item.Unit),
		})
	}
	return result
}

func metricExtensionStatusQueryProperties(query stackmonitoringsdk.MetricExtensionQueryProperties) stackmonitoringv1beta1.MetricExtensionQueryProperties {
	var status stackmonitoringv1beta1.MetricExtensionQueryProperties
	if query == nil {
		return status
	}
	payload, err := json.Marshal(query)
	if err != nil {
		return status
	}
	_ = json.Unmarshal(payload, &status)
	return status
}

func metricExtensionStatusEnabledResources(items []stackmonitoringsdk.EnabledResourceDetails) []stackmonitoringv1beta1.MetricExtensionEnabledOnResource {
	if len(items) == 0 {
		return nil
	}
	result := make([]stackmonitoringv1beta1.MetricExtensionEnabledOnResource, 0, len(items))
	for _, item := range items {
		result = append(result, stackmonitoringv1beta1.MetricExtensionEnabledOnResource{
			ResourceId: metricExtensionStringValue(item.ResourceId),
		})
	}
	return result
}

func markMetricExtensionSuccess(
	resource *stackmonitoringv1beta1.MetricExtension,
	current stackmonitoringsdk.MetricExtension,
	fallback shared.OSOKConditionType,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	condition := metricExtensionCondition(current, fallback)
	message := metricExtensionConditionMessage(current, condition)
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, condition, corev1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
	return servicemanager.OSOKResponse{IsSuccessful: condition != shared.Failed}
}

func markMetricExtensionFailure(resource *stackmonitoringv1beta1.MetricExtension, err error) {
	if resource == nil || err == nil {
		return
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), servicemanager.RuntimeDeps{}.Log)
}

func metricExtensionCondition(
	current stackmonitoringsdk.MetricExtension,
	fallback shared.OSOKConditionType,
) shared.OSOKConditionType {
	state := metricExtensionSDKLifecycleState(current.LifecycleState, current.Status)
	switch {
	case metricExtensionTerminalDeleteState(state):
		return shared.Terminating
	case state == "" || state == string(stackmonitoringsdk.MetricExtensionLifeCycleStatesActive) ||
		state == string(stackmonitoringsdk.MetricExtensionLifeCycleDetailsDraft) ||
		state == string(stackmonitoringsdk.MetricExtensionLifeCycleDetailsPublished):
		return shared.Active
	default:
		return fallback
	}
}

func metricExtensionConditionMessage(
	current stackmonitoringsdk.MetricExtension,
	condition shared.OSOKConditionType,
) string {
	if displayName := metricExtensionStringValue(current.DisplayName); displayName != "" {
		return displayName
	}
	switch condition {
	case shared.Terminating:
		return "OCI resource delete is in progress"
	case shared.Active:
		return "OCI resource is active"
	default:
		return "OCI resource state was observed"
	}
}

func metricExtensionRetryToken(resource *stackmonitoringv1beta1.MetricExtension) *string {
	if resource == nil {
		return nil
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return &uid
	}
	seed := strings.TrimSpace(resource.Namespace) + "/" + strings.TrimSpace(resource.Name)
	if strings.Trim(seed, "/") == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(seed))
	token := fmt.Sprintf("%x", sum[:16])
	return &token
}

func metricExtensionID(current stackmonitoringsdk.MetricExtension) string {
	return strings.TrimSpace(metricExtensionStringValue(current.Id))
}

func metricExtensionSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func metricExtensionBoolValue(value *bool) bool {
	return value != nil && *value
}

func (c *metricExtensionRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.failCreateOrUpdate(resource, c.initErr)
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("MetricExtension resource is nil")
	}

	current, found, err := c.resolveCurrentMetricExtension(ctx, resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	namespace := resource.Namespace
	if strings.TrimSpace(namespace) == "" {
		namespace = req.Namespace
	}
	if found {
		return c.reconcileCurrentMetricExtension(ctx, resource, namespace, current)
	}
	return c.createMetricExtension(ctx, resource, namespace)
}

func (c *metricExtensionRuntimeClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	if resource == nil {
		return false, fmt.Errorf("MetricExtension resource is nil")
	}

	currentID, err := c.resolveMetricExtensionDeleteID(ctx, resource)
	if err != nil {
		return false, err
	}
	if currentID == "" {
		markMetricExtensionDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}

	current, err := c.getMetricExtension(ctx, currentID)
	if err != nil {
		return c.handleMetricExtensionDeleteReadError(resource, err)
	}
	projectMetricExtensionStatus(resource, current)
	if metricExtensionTerminalDeleteState(metricExtensionSDKLifecycleState(current.LifecycleState, current.Status)) {
		markMetricExtensionDeleted(resource, "OCI resource already deleted")
		return true, nil
	}

	response, err := c.hooks.Delete.Call(ctx, stackmonitoringsdk.DeleteMetricExtensionRequest{
		MetricExtensionId: metricExtensionString(currentID),
	})
	if err != nil {
		return c.handleMetricExtensionDeleteCallError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	confirmed, err := c.getMetricExtension(ctx, currentID)
	if err != nil {
		return c.handleMetricExtensionPostDeleteReadError(resource, err)
	}
	projectMetricExtensionStatus(resource, confirmed)
	if metricExtensionTerminalDeleteState(metricExtensionSDKLifecycleState(confirmed.LifecycleState, confirmed.Status)) {
		markMetricExtensionDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	markMetricExtensionTerminating(resource, "OCI resource delete is in progress")
	return false, nil
}

func (c *metricExtensionRuntimeClient) resolveMetricExtensionDeleteID(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
) (string, error) {
	if currentID := metricExtensionRecordedID(resource); currentID != "" {
		return currentID, nil
	}

	summary, found, err := resolveMetricExtensionDeleteFallbackSummary(ctx, c.list, resource)
	if err != nil || !found {
		return "", err
	}
	currentID := metricExtensionSummaryID(summary)
	if currentID == "" {
		return "", fmt.Errorf("MetricExtension delete fallback could not resolve a resource OCID")
	}
	recordMetricExtensionID(resource, currentID)
	return currentID, nil
}

func (c *metricExtensionRuntimeClient) handleMetricExtensionDeleteCallError(
	resource *stackmonitoringv1beta1.MetricExtension,
	err error,
) (bool, error) {
	err = handleMetricExtensionDeleteError(resource, err)
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		markMetricExtensionDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	return false, err
}

func (c *metricExtensionRuntimeClient) resolveCurrentMetricExtension(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
) (stackmonitoringsdk.MetricExtension, bool, error) {
	currentID := metricExtensionRecordedID(resource)
	if currentID != "" {
		current, err := c.getMetricExtension(ctx, currentID)
		if err == nil {
			if !metricExtensionResourceTerminalDeleted(current) {
				return current, true, nil
			}
			// A terminal DELETED readback is a stale identity, not current state.
			// Fall through to list fallback so an active replacement can bind or
			// a new resource can be created without sending an unsafe update.
		} else if !errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return stackmonitoringsdk.MetricExtension{}, false, err
		}
	}

	summary, found, err := resolveMetricExtensionDeleteFallbackSummary(ctx, c.list, resource)
	if err != nil {
		return stackmonitoringsdk.MetricExtension{}, false, err
	}
	if !found {
		return stackmonitoringsdk.MetricExtension{}, false, nil
	}

	resolvedID := metricExtensionSummaryID(summary)
	if resolvedID == "" {
		return stackmonitoringsdk.MetricExtension{}, false, fmt.Errorf("MetricExtension list lookup could not resolve a resource OCID")
	}
	current, err := c.getMetricExtension(ctx, resolvedID)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return stackmonitoringsdk.MetricExtension{}, false, nil
		}
		return stackmonitoringsdk.MetricExtension{}, false, err
	}
	return current, true, nil
}

func (c *metricExtensionRuntimeClient) reconcileCurrentMetricExtension(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
	namespace string,
	current stackmonitoringsdk.MetricExtension,
) (servicemanager.OSOKResponse, error) {
	projectMetricExtensionStatus(resource, current)
	if err := validateMetricExtensionCreateOnlyDrift(resource, current); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	updateBody, ok, err := buildMetricExtensionUpdateBody(ctx, resource, namespace, current, nil)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if !ok {
		return markMetricExtensionSuccess(resource, current, shared.Active), nil
	}

	details, err := metricExtensionUpdateDetails(updateBody)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	currentID := metricExtensionRecordedID(resource)
	response, err := c.hooks.Update.Call(ctx, stackmonitoringsdk.UpdateMetricExtensionRequest{
		MetricExtensionId:            metricExtensionString(currentID),
		UpdateMetricExtensionDetails: details,
	})
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	updated := response.MetricExtension
	if refreshed, err := c.getMetricExtension(ctx, currentID); err == nil {
		updated = refreshed
	} else if !errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return c.failCreateOrUpdate(resource, err)
	}
	projectMetricExtensionStatus(resource, updated)
	return markMetricExtensionSuccess(resource, updated, shared.Updating), nil
}

func (c *metricExtensionRuntimeClient) createMetricExtension(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MetricExtension,
	namespace string,
) (servicemanager.OSOKResponse, error) {
	body, err := c.hooks.BuildCreateBody(ctx, resource, namespace)
	if err != nil {
		return c.failCreateOrUpdate(resource, fmt.Errorf("build MetricExtension create body: %w", err))
	}
	details, err := metricExtensionCreateDetails(body)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	response, err := c.hooks.Create.Call(ctx, stackmonitoringsdk.CreateMetricExtensionRequest{
		CreateMetricExtensionDetails: details,
		OpcRetryToken:                metricExtensionRetryToken(resource),
	})
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	created := response.MetricExtension
	currentID := metricExtensionID(created)
	if currentID != "" {
		if refreshed, err := c.getMetricExtension(ctx, currentID); err == nil {
			created = refreshed
		} else if !errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return c.failCreateOrUpdate(resource, err)
		}
	}
	projectMetricExtensionStatus(resource, created)
	return markMetricExtensionSuccess(resource, created, shared.Provisioning), nil
}

func (c *metricExtensionRuntimeClient) getMetricExtension(ctx context.Context, currentID string) (stackmonitoringsdk.MetricExtension, error) {
	response, err := c.hooks.Get.Call(ctx, stackmonitoringsdk.GetMetricExtensionRequest{
		MetricExtensionId: metricExtensionString(currentID),
	})
	if err != nil {
		return stackmonitoringsdk.MetricExtension{}, err
	}
	return response.MetricExtension, nil
}

func (c *metricExtensionRuntimeClient) handleMetricExtensionDeleteReadError(
	resource *stackmonitoringv1beta1.MetricExtension,
	err error,
) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markMetricExtensionDeleted(resource, "OCI resource no longer exists")
		return true, nil
	case classification.IsAuthShapedNotFound():
		ambiguous := metricExtensionAmbiguousNotFound("pre-delete read", err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		return false, ambiguous
	default:
		return false, err
	}
}

func (c *metricExtensionRuntimeClient) handleMetricExtensionPostDeleteReadError(
	resource *stackmonitoringv1beta1.MetricExtension,
	err error,
) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markMetricExtensionDeleted(resource, "OCI resource deleted")
		return true, nil
	case classification.IsAuthShapedNotFound():
		ambiguous := metricExtensionAmbiguousNotFound("delete confirmation", err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		return false, ambiguous
	default:
		return false, err
	}
}

func (c *metricExtensionRuntimeClient) failCreateOrUpdate(
	resource *stackmonitoringv1beta1.MetricExtension,
	err error,
) (servicemanager.OSOKResponse, error) {
	markMetricExtensionFailure(resource, err)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func resolveMetricExtensionDeleteFallbackSummary(
	ctx context.Context,
	list metricExtensionListCall,
	resource *stackmonitoringv1beta1.MetricExtension,
) (stackmonitoringsdk.MetricExtensionSummary, bool, error) {
	response, err := list(ctx, stackmonitoringsdk.ListMetricExtensionsRequest{
		CompartmentId: metricExtensionString(resource.Spec.CompartmentId),
		ResourceType:  metricExtensionString(resource.Spec.ResourceType),
		Name:          metricExtensionString(resource.Spec.Name),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			ambiguous := metricExtensionAmbiguousNotFound("delete fallback list", err)
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
			return stackmonitoringsdk.MetricExtensionSummary{}, false, ambiguous
		}
		return stackmonitoringsdk.MetricExtensionSummary{}, false, err
	}

	matches := make([]stackmonitoringsdk.MetricExtensionSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if metricExtensionSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return stackmonitoringsdk.MetricExtensionSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return stackmonitoringsdk.MetricExtensionSummary{}, false, fmt.Errorf("MetricExtension delete fallback found multiple matching resources for compartmentId %q, resourceType %q, and name %q", resource.Spec.CompartmentId, resource.Spec.ResourceType, resource.Spec.Name)
	}
}

func confirmMetricExtensionDeleteRead(
	ctx context.Context,
	hooks *MetricExtensionRuntimeHooks,
	resource *stackmonitoringv1beta1.MetricExtension,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("MetricExtension runtime hooks are nil")
	}
	response, err := readMetricExtensionForDelete(ctx, hooks, resource, currentID)
	return metricExtensionDeleteConfirmReadResponse(response, err)
}

func readMetricExtensionForDelete(
	ctx context.Context,
	hooks *MetricExtensionRuntimeHooks,
	resource *stackmonitoringv1beta1.MetricExtension,
	currentID string,
) (any, error) {
	currentID = strings.TrimSpace(currentID)
	if currentID != "" {
		return hooks.Get.Call(ctx, stackmonitoringsdk.GetMetricExtensionRequest{
			MetricExtensionId: metricExtensionString(currentID),
		})
	}
	if hooks.List.Call == nil {
		return nil, metricExtensionNotFound("MetricExtension delete confirmation has no list operation")
	}
	summary, found, err := resolveMetricExtensionDeleteFallbackSummary(ctx, hooks.List.Call, resource)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, metricExtensionNotFound("MetricExtension delete confirmation did not find a matching OCI resource")
	}
	return summary, nil
}

func metricExtensionDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return metricExtensionAuthShapedConfirmRead{err: err}, nil
	}
	return response, err
}

func handleMetricExtensionDeleteError(resource *stackmonitoringv1beta1.MetricExtension, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	ambiguous := metricExtensionAmbiguousNotFound("delete", err)
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
	return ambiguous
}

func handleMetricExtensionDeleteConfirmReadOutcome(
	resource *stackmonitoringv1beta1.MetricExtension,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case metricExtensionAuthShapedConfirmRead:
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, typed)
		return generatedruntime.DeleteOutcome{Handled: true}, typed
	case *metricExtensionAuthShapedConfirmRead:
		if typed != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, *typed)
			return generatedruntime.DeleteOutcome{Handled: true}, *typed
		}
	}

	if stage != generatedruntime.DeleteConfirmStageAfterRequest {
		return generatedruntime.DeleteOutcome{}, nil
	}
	state := metricExtensionLifecycleState(response)
	if state == "" || metricExtensionTerminalDeleteState(state) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markMetricExtensionTerminating(resource, "OCI resource delete is in progress")
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func listMetricExtensionsAllPages(call metricExtensionListCall) metricExtensionListCall {
	return func(ctx context.Context, request stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
		var combined stackmonitoringsdk.ListMetricExtensionsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			combined.Items = append(combined.Items, response.Items...)
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			request.Page = response.OpcNextPage
		}
	}
}

func metricExtensionSummaryMatchesSpec(
	summary stackmonitoringsdk.MetricExtensionSummary,
	spec stackmonitoringv1beta1.MetricExtensionSpec,
) bool {
	if metricExtensionTerminalDeleteState(metricExtensionSDKLifecycleState(summary.LifecycleState, summary.Status)) {
		return false
	}
	return strings.TrimSpace(metricExtensionStringValue(summary.CompartmentId)) == strings.TrimSpace(spec.CompartmentId) &&
		strings.TrimSpace(metricExtensionStringValue(summary.ResourceType)) == strings.TrimSpace(spec.ResourceType) &&
		strings.TrimSpace(metricExtensionStringValue(summary.Name)) == strings.TrimSpace(spec.Name)
}

func metricExtensionResourceTerminalDeleted(current stackmonitoringsdk.MetricExtension) bool {
	return metricExtensionTerminalDeleteState(metricExtensionSDKLifecycleState(current.LifecycleState, current.Status))
}

func metricExtensionLifecycleState(response any) string {
	switch typed := response.(type) {
	case stackmonitoringsdk.GetMetricExtensionResponse:
		return metricExtensionSDKLifecycleState(typed.LifecycleState, typed.Status)
	case *stackmonitoringsdk.GetMetricExtensionResponse:
		if typed == nil {
			return ""
		}
		return metricExtensionSDKLifecycleState(typed.LifecycleState, typed.Status)
	case stackmonitoringsdk.MetricExtension:
		return metricExtensionSDKLifecycleState(typed.LifecycleState, typed.Status)
	case *stackmonitoringsdk.MetricExtension:
		if typed == nil {
			return ""
		}
		return metricExtensionSDKLifecycleState(typed.LifecycleState, typed.Status)
	case stackmonitoringsdk.MetricExtensionSummary:
		return metricExtensionSDKLifecycleState(typed.LifecycleState, typed.Status)
	case *stackmonitoringsdk.MetricExtensionSummary:
		if typed == nil {
			return ""
		}
		return metricExtensionSDKLifecycleState(typed.LifecycleState, typed.Status)
	default:
		return ""
	}
}

func metricExtensionSDKLifecycleState(
	lifecycleState stackmonitoringsdk.MetricExtensionLifeCycleStatesEnum,
	status stackmonitoringsdk.MetricExtensionLifeCycleDetailsEnum,
) string {
	if lifecycleState != "" {
		return strings.ToUpper(string(lifecycleState))
	}
	return strings.ToUpper(string(status))
}

func metricExtensionTerminalDeleteState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), string(stackmonitoringsdk.MetricExtensionLifeCycleStatesDeleted))
}

func metricExtensionNotFound(message string) error {
	return errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    message,
	}
}

func metricExtensionAmbiguousNotFound(operation string, err error) metricExtensionAmbiguousNotFoundError {
	message := fmt.Sprintf("MetricExtension %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	return metricExtensionAmbiguousNotFoundError{
		message:      message,
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func metricExtensionRecordedID(resource *stackmonitoringv1beta1.MetricExtension) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func recordMetricExtensionID(resource *stackmonitoringv1beta1.MetricExtension, id string) {
	id = strings.TrimSpace(id)
	if resource == nil || id == "" {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.Id = id
}

func metricExtensionSummaryID(summary stackmonitoringsdk.MetricExtensionSummary) string {
	return strings.TrimSpace(metricExtensionStringValue(summary.Id))
}

func markMetricExtensionDeleted(resource *stackmonitoringv1beta1.MetricExtension, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
}

func markMetricExtensionTerminating(resource *stackmonitoringv1beta1.MetricExtension, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	_ = applyMetricExtensionAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	})
}

func applyMetricExtensionAsyncOperation(status *shared.OSOKStatus, current *shared.OSOKAsyncOperation) error {
	if status == nil {
		return nil
	}
	status.Async.Current = current
	return nil
}

func metricExtensionCredentialClient(manager *MetricExtensionServiceManager) credhelper.CredentialClient {
	if manager == nil {
		return nil
	}
	return manager.CredentialClient
}

func metricExtensionString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func metricExtensionStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ error = metricExtensionAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = metricExtensionAmbiguousNotFoundError{}
