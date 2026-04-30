/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package script

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	apmsyntheticssdk "github.com/oracle/oci-go-sdk/v65/apmsynthetics"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmsyntheticsv1beta1 "github.com/oracle/oci-service-operator/api/apmsynthetics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type scriptOCIClient interface {
	CreateScript(context.Context, apmsyntheticssdk.CreateScriptRequest) (apmsyntheticssdk.CreateScriptResponse, error)
	GetScript(context.Context, apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error)
	ListScripts(context.Context, apmsyntheticssdk.ListScriptsRequest) (apmsyntheticssdk.ListScriptsResponse, error)
	UpdateScript(context.Context, apmsyntheticssdk.UpdateScriptRequest) (apmsyntheticssdk.UpdateScriptResponse, error)
	DeleteScript(context.Context, apmsyntheticssdk.DeleteScriptRequest) (apmsyntheticssdk.DeleteScriptResponse, error)
}

type synchronousScriptServiceClient struct {
	delegate ScriptServiceClient
	log      loggerutil.OSOKLogger
}

type scriptStatusMirrorClient struct {
	delegate ScriptServiceClient
}

type scriptParameterComparable struct {
	ParamName  string
	ParamValue string
	IsSecret   bool
}

func init() {
	registerScriptRuntimeHooksMutator(func(manager *ScriptServiceManager, hooks *ScriptRuntimeHooks) {
		applyScriptRuntimeHooks(manager, hooks)
	})
}

func applyScriptRuntimeHooks(manager *ScriptServiceManager, hooks *ScriptRuntimeHooks) {
	if hooks == nil {
		return
	}

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}

	hooks.Semantics = reviewedScriptRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardScriptExistingBeforeCreate
	hooks.Create.Fields = scriptCreateFields()
	hooks.Get.Fields = scriptGetFields()
	hooks.List.Fields = scriptListFields()
	hooks.Update.Fields = scriptUpdateFields()
	hooks.Delete.Fields = scriptDeleteFields()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *apmsyntheticsv1beta1.Script,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildScriptUpdateDetails(resource, currentResponse)
	}
	hooks.WrapGeneratedClient = append(
		hooks.WrapGeneratedClient,
		func(delegate ScriptServiceClient) ScriptServiceClient {
			return &synchronousScriptServiceClient{
				delegate: delegate,
				log:      log,
			}
		},
		wrapScriptStatusMirrorClient,
	)
}

func newScriptServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client scriptOCIClient,
) ScriptServiceClient {
	hooks := newScriptRuntimeHooksWithOCIClient(client)
	applyScriptRuntimeHooks(&ScriptServiceManager{Log: log}, &hooks)
	delegate := defaultScriptServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apmsyntheticsv1beta1.Script](
			buildScriptGeneratedRuntimeConfig(&ScriptServiceManager{Log: log}, hooks),
		),
	}
	return wrapScriptGeneratedClient(hooks, delegate)
}

func newScriptRuntimeHooksWithOCIClient(client scriptOCIClient) ScriptRuntimeHooks {
	return ScriptRuntimeHooks{
		Semantics: reviewedScriptRuntimeSemantics(),
		Create: runtimeOperationHooks[apmsyntheticssdk.CreateScriptRequest, apmsyntheticssdk.CreateScriptResponse]{
			Fields: scriptCreateFields(),
			Call: func(ctx context.Context, request apmsyntheticssdk.CreateScriptRequest) (apmsyntheticssdk.CreateScriptResponse, error) {
				return client.CreateScript(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apmsyntheticssdk.GetScriptRequest, apmsyntheticssdk.GetScriptResponse]{
			Fields: scriptGetFields(),
			Call: func(ctx context.Context, request apmsyntheticssdk.GetScriptRequest) (apmsyntheticssdk.GetScriptResponse, error) {
				return client.GetScript(ctx, request)
			},
		},
		List: runtimeOperationHooks[apmsyntheticssdk.ListScriptsRequest, apmsyntheticssdk.ListScriptsResponse]{
			Fields: scriptListFields(),
			Call: func(ctx context.Context, request apmsyntheticssdk.ListScriptsRequest) (apmsyntheticssdk.ListScriptsResponse, error) {
				return client.ListScripts(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apmsyntheticssdk.UpdateScriptRequest, apmsyntheticssdk.UpdateScriptResponse]{
			Fields: scriptUpdateFields(),
			Call: func(ctx context.Context, request apmsyntheticssdk.UpdateScriptRequest) (apmsyntheticssdk.UpdateScriptResponse, error) {
				return client.UpdateScript(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apmsyntheticssdk.DeleteScriptRequest, apmsyntheticssdk.DeleteScriptResponse]{
			Fields: scriptDeleteFields(),
			Call: func(ctx context.Context, request apmsyntheticssdk.DeleteScriptRequest) (apmsyntheticssdk.DeleteScriptResponse, error) {
				return client.DeleteScript(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ScriptServiceClient) ScriptServiceClient{},
	}
}

func reviewedScriptRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newScriptRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"displayName", "contentType"},
	}
	semantics.AuxiliaryOperations = nil
	semantics.Unsupported = nil
	return semantics
}

func scriptCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "CreateScriptDetails", RequestName: "CreateScriptDetails", Contribution: "body"},
	}
}

func scriptGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScriptId", RequestName: "scriptId", Contribution: "path", PreferResourceID: true},
	}
}

func scriptListFields() []generatedruntime.RequestField {
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
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{
			FieldName:    "ContentType",
			RequestName:  "contentType",
			Contribution: "query",
			LookupPaths:  []string{"status.contentType", "spec.contentType", "contentType"},
		},
	}
}

func scriptUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScriptId", RequestName: "scriptId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateScriptDetails", RequestName: "UpdateScriptDetails", Contribution: "body"},
	}
}

func scriptDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ApmDomainId",
			RequestName:  "apmDomainId",
			Contribution: "query",
			LookupPaths:  []string{"status.apmDomainId", "spec.apmDomainId", "apmDomainId"},
		},
		{FieldName: "ScriptId", RequestName: "scriptId", Contribution: "path", PreferResourceID: true},
	}
}

func guardScriptExistingBeforeCreate(
	_ context.Context,
	resource *apmsyntheticsv1beta1.Script,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Script resource is nil")
	}
	if strings.TrimSpace(resource.Spec.ApmDomainId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Script spec.apmDomainId is required")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if strings.TrimSpace(resource.Spec.ContentType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapScriptStatusMirrorClient(delegate ScriptServiceClient) ScriptServiceClient {
	return scriptStatusMirrorClient{delegate: delegate}
}

func (c scriptStatusMirrorClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmsyntheticsv1beta1.Script,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful {
		projectScriptRequestContext(resource)
	}
	return response, err
}

func (c scriptStatusMirrorClient) Delete(ctx context.Context, resource *apmsyntheticsv1beta1.Script) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func projectScriptRequestContext(resource *apmsyntheticsv1beta1.Script) {
	if resource == nil {
		return
	}

	resource.Status.ApmDomainId = strings.TrimSpace(resource.Spec.ApmDomainId)
}

func (c *synchronousScriptServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *apmsyntheticsv1beta1.Script,
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
		status.Message = strings.TrimSpace(resource.Status.DisplayName)
		if strings.TrimSpace(status.Message) == "" {
			status.Message = strings.TrimSpace(resource.Spec.DisplayName)
		}
		if strings.TrimSpace(status.Message) == "" {
			status.Message = "OCI Script is active"
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

func (c *synchronousScriptServiceClient) Delete(ctx context.Context, resource *apmsyntheticsv1beta1.Script) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func buildScriptUpdateDetails(
	resource *apmsyntheticsv1beta1.Script,
	currentResponse any,
) (apmsyntheticssdk.UpdateScriptDetails, bool, error) {
	if resource == nil {
		return apmsyntheticssdk.UpdateScriptDetails{}, false, fmt.Errorf("Script resource is nil")
	}

	current, err := scriptFromResponse(currentResponse)
	if err != nil {
		return apmsyntheticssdk.UpdateScriptDetails{}, false, err
	}

	details := apmsyntheticssdk.UpdateScriptDetails{}
	updateNeeded := false

	if desired, ok := scriptDesiredRequiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := scriptDesiredContentTypeUpdate(resource.Spec.ContentType, current.ContentType); ok {
		details.ContentType = desired
		updateNeeded = true
	}
	if desired, ok := scriptDesiredRequiredStringUpdate(resource.Spec.Content, current.Content); ok {
		details.Content = desired
		updateNeeded = true
	}
	if desired, ok := scriptDesiredOptionalStringUpdate(resource.Spec.ContentFileName, current.ContentFileName); ok {
		details.ContentFileName = desired
		updateNeeded = true
	}
	if desired, ok := scriptDesiredParametersUpdate(resource.Spec.Parameters, current.Parameters); ok {
		details.Parameters = desired
		updateNeeded = true
	}
	if desired, ok := scriptDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok, err := scriptDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); err != nil {
		return apmsyntheticssdk.UpdateScriptDetails{}, false, err
	} else if ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func scriptFromResponse(currentResponse any) (apmsyntheticssdk.Script, error) {
	switch current := currentResponse.(type) {
	case apmsyntheticssdk.Script:
		return current, nil
	case *apmsyntheticssdk.Script:
		if current == nil {
			return apmsyntheticssdk.Script{}, fmt.Errorf("current Script response is nil")
		}
		return *current, nil
	case apmsyntheticssdk.ScriptSummary:
		return apmsyntheticssdk.Script{
			Id:                    current.Id,
			DisplayName:           current.DisplayName,
			ContentType:           current.ContentType,
			MonitorStatusCountMap: current.MonitorStatusCountMap,
			TimeCreated:           current.TimeCreated,
			TimeUpdated:           current.TimeUpdated,
			FreeformTags:          current.FreeformTags,
			DefinedTags:           current.DefinedTags,
		}, nil
	case *apmsyntheticssdk.ScriptSummary:
		if current == nil {
			return apmsyntheticssdk.Script{}, fmt.Errorf("current Script response is nil")
		}
		return scriptFromResponse(*current)
	case apmsyntheticssdk.CreateScriptResponse:
		return current.Script, nil
	case *apmsyntheticssdk.CreateScriptResponse:
		if current == nil {
			return apmsyntheticssdk.Script{}, fmt.Errorf("current Script response is nil")
		}
		return current.Script, nil
	case apmsyntheticssdk.GetScriptResponse:
		return current.Script, nil
	case *apmsyntheticssdk.GetScriptResponse:
		if current == nil {
			return apmsyntheticssdk.Script{}, fmt.Errorf("current Script response is nil")
		}
		return current.Script, nil
	case apmsyntheticssdk.UpdateScriptResponse:
		return current.Script, nil
	case *apmsyntheticssdk.UpdateScriptResponse:
		if current == nil {
			return apmsyntheticssdk.Script{}, fmt.Errorf("current Script response is nil")
		}
		return current.Script, nil
	default:
		return apmsyntheticssdk.Script{}, fmt.Errorf("unexpected current Script response type %T", currentResponse)
	}
}

func scriptDesiredRequiredStringUpdate(spec string, current *string) (*string, bool) {
	if spec == "" {
		return nil, false
	}
	return scriptDesiredOptionalStringUpdate(spec, current)
}

func scriptDesiredOptionalStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func scriptDesiredContentTypeUpdate(
	spec string,
	current apmsyntheticssdk.ContentTypesEnum,
) (apmsyntheticssdk.ContentTypesEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return apmsyntheticssdk.ContentTypesEnum(spec), true
}

func scriptDesiredParametersUpdate(
	spec []apmsyntheticsv1beta1.ScriptParameter,
	current []apmsyntheticssdk.ScriptParameterInfo,
) ([]apmsyntheticssdk.ScriptParameter, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if scriptParametersEqual(spec, current) {
		return nil, false
	}
	return scriptParametersToSDK(spec), true
}

func scriptDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func scriptDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	currentNormalized, err := scriptDefinedTagsFromSDK(current)
	if err != nil {
		return nil, false, err
	}
	if scriptDefinedTagsEqual(spec, currentNormalized) {
		return nil, false, nil
	}

	desired, err := scriptDefinedTagsToSDK(spec)
	if err != nil {
		return nil, false, err
	}
	return desired, true, nil
}

func scriptParametersEqual(
	spec []apmsyntheticsv1beta1.ScriptParameter,
	current []apmsyntheticssdk.ScriptParameterInfo,
) bool {
	return slices.Equal(scriptDesiredParametersComparable(spec), scriptCurrentParametersComparable(current))
}

func scriptDesiredParametersComparable(spec []apmsyntheticsv1beta1.ScriptParameter) []scriptParameterComparable {
	values := make([]scriptParameterComparable, 0, len(spec))
	for _, parameter := range spec {
		values = append(values, scriptParameterComparable{
			ParamName:  strings.TrimSpace(parameter.ParamName),
			ParamValue: parameter.ParamValue,
			IsSecret:   parameter.IsSecret,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].ParamName != values[j].ParamName {
			return values[i].ParamName < values[j].ParamName
		}
		if values[i].ParamValue != values[j].ParamValue {
			return values[i].ParamValue < values[j].ParamValue
		}
		if values[i].IsSecret != values[j].IsSecret {
			return !values[i].IsSecret && values[j].IsSecret
		}
		return false
	})
	return values
}

func scriptCurrentParametersComparable(current []apmsyntheticssdk.ScriptParameterInfo) []scriptParameterComparable {
	values := make([]scriptParameterComparable, 0, len(current))
	for _, parameterInfo := range current {
		if parameterInfo.ScriptParameter == nil {
			continue
		}
		values = append(values, scriptParameterComparable{
			ParamName:  strings.TrimSpace(stringValue(parameterInfo.ScriptParameter.ParamName)),
			ParamValue: stringValue(parameterInfo.ScriptParameter.ParamValue),
			IsSecret:   boolValue(parameterInfo.ScriptParameter.IsSecret),
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].ParamName != values[j].ParamName {
			return values[i].ParamName < values[j].ParamName
		}
		if values[i].ParamValue != values[j].ParamValue {
			return values[i].ParamValue < values[j].ParamValue
		}
		if values[i].IsSecret != values[j].IsSecret {
			return !values[i].IsSecret && values[j].IsSecret
		}
		return false
	})
	return values
}

func scriptParametersToSDK(spec []apmsyntheticsv1beta1.ScriptParameter) []apmsyntheticssdk.ScriptParameter {
	if spec == nil {
		return nil
	}

	values := make([]apmsyntheticssdk.ScriptParameter, len(spec))
	for i, parameter := range spec {
		values[i] = apmsyntheticssdk.ScriptParameter{
			ParamName:  common.String(parameter.ParamName),
			ParamValue: common.String(parameter.ParamValue),
			IsSecret:   common.Bool(parameter.IsSecret),
		}
	}
	return values
}

func scriptDefinedTagsEqual(
	spec map[string]shared.MapValue,
	current map[string]shared.MapValue,
) bool {
	if len(spec) != len(current) {
		return false
	}
	for namespace, desired := range spec {
		live, ok := current[namespace]
		if !ok {
			return false
		}
		if !maps.Equal(map[string]string(desired), map[string]string(live)) {
			return false
		}
	}
	return true
}

func scriptDefinedTagsToSDK(spec map[string]shared.MapValue) (map[string]map[string]interface{}, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal Script definedTags: %w", err)
	}

	out := make(map[string]map[string]interface{})
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("decode Script definedTags into SDK shape: %w", err)
	}
	return out, nil
}

func scriptDefinedTagsFromSDK(current map[string]map[string]interface{}) (map[string]shared.MapValue, error) {
	payload, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("marshal current Script definedTags: %w", err)
	}

	out := make(map[string]shared.MapValue)
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("decode current Script definedTags: %w", err)
	}
	return out, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}
