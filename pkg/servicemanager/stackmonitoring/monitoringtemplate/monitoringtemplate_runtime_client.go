/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoringtemplate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const monitoringTemplateKind = "MonitoringTemplate"

type monitoringTemplateListCall func(context.Context, stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error)

type monitoringTemplateAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type monitoringTemplateDeletePreflightClient struct {
	delegate MonitoringTemplateServiceClient
	get      func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error)
	list     monitoringTemplateListCall
}

type monitoringTemplateStatusProjection struct {
	Id                          string                                            `json:"id,omitempty"`
	DisplayName                 string                                            `json:"displayName,omitempty"`
	TenantId                    string                                            `json:"tenantId,omitempty"`
	CompartmentId               string                                            `json:"compartmentId,omitempty"`
	Status                      string                                            `json:"sdkStatus,omitempty"`
	LifecycleState              string                                            `json:"lifecycleState,omitempty"`
	Destinations                []string                                          `json:"destinations,omitempty"`
	Members                     []stackmonitoringv1beta1.MonitoringTemplateMember `json:"members,omitempty"`
	TotalAlarmConditions        float32                                           `json:"totalAlarmConditions,omitempty"`
	TotalAppliedAlarmConditions float32                                           `json:"totalAppliedAlarmConditions,omitempty"`
	TimeCreated                 string                                            `json:"timeCreated,omitempty"`
	TimeUpdated                 string                                            `json:"timeUpdated,omitempty"`
	Description                 string                                            `json:"description,omitempty"`
	IsAlarmsEnabled             bool                                              `json:"isAlarmsEnabled,omitempty"`
	IsSplitNotificationEnabled  bool                                              `json:"isSplitNotificationEnabled,omitempty"`
	RepeatNotificationDuration  string                                            `json:"repeatNotificationDuration,omitempty"`
	MessageFormat               string                                            `json:"messageFormat,omitempty"`
	FreeformTags                map[string]string                                 `json:"freeformTags,omitempty"`
	DefinedTags                 map[string]shared.MapValue                        `json:"definedTags,omitempty"`
	SystemTags                  map[string]shared.MapValue                        `json:"systemTags,omitempty"`
}

type monitoringTemplateProjectedResponse struct {
	MonitoringTemplate monitoringTemplateStatusProjection `presentIn:"body"`
	OpcRequestId       *string                            `presentIn:"header" name:"opc-request-id"`
}

type monitoringTemplateProjectedCollection struct {
	Items []monitoringTemplateStatusProjection `json:"items,omitempty"`
}

type monitoringTemplateProjectedListResponse struct {
	MonitoringTemplateCollection monitoringTemplateProjectedCollection `presentIn:"body"`
	OpcRequestId                 *string                               `presentIn:"header" name:"opc-request-id"`
	OpcNextPage                  *string                               `presentIn:"header" name:"opc-next-page"`
}

func (e monitoringTemplateAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e monitoringTemplateAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerMonitoringTemplateRuntimeHooksMutator(func(manager *MonitoringTemplateServiceManager, hooks *MonitoringTemplateRuntimeHooks) {
		applyMonitoringTemplateRuntimeHooks(manager, hooks)
	})
}

func applyMonitoringTemplateRuntimeHooks(_ *MonitoringTemplateServiceManager, hooks *MonitoringTemplateRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newMonitoringTemplateRuntimeSemantics()
	hooks.BuildCreateBody = buildMonitoringTemplateCreateBody
	hooks.BuildUpdateBody = buildMonitoringTemplateUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardMonitoringTemplateExistingBeforeCreate
	hooks.Create.Fields = monitoringTemplateCreateFields()
	hooks.Get.Fields = monitoringTemplateGetFields()
	hooks.List.Fields = monitoringTemplateListFields()
	hooks.Update.Fields = monitoringTemplateUpdateFields()
	hooks.Delete.Fields = monitoringTemplateDeleteFields()
	wrapMonitoringTemplateReadAndDeleteCalls(hooks)
	installMonitoringTemplateProjectedReadOperations(hooks)
	hooks.StatusHooks.ProjectStatus = projectMonitoringTemplateStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateMonitoringTemplateCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleMonitoringTemplateDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyMonitoringTemplateDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MonitoringTemplateServiceClient) MonitoringTemplateServiceClient {
		return monitoringTemplateDeletePreflightClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newMonitoringTemplateRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "monitoringtemplate",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesCreating)},
			UpdatingStates:     []string{string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesUpdating)},
			ActiveStates: []string{
				string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive),
				string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"destinations",
				"isAlarmsEnabled",
				"isSplitNotificationEnabled",
				"members",
				"repeatNotificationDuration",
				"messageFormat",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: monitoringTemplateKind, Action: "CreateMonitoringTemplate"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: monitoringTemplateKind, Action: "UpdateMonitoringTemplate"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: monitoringTemplateKind, Action: "DeleteMonitoringTemplate"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func monitoringTemplateCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateMonitoringTemplateDetails", RequestName: "CreateMonitoringTemplateDetails", Contribution: "body"},
		{FieldName: "OpcRequestId", RequestName: "opc-request-id", Contribution: "header"},
	}
}

func monitoringTemplateGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitoringTemplateId", RequestName: "monitoringTemplateId", Contribution: "path", PreferResourceID: true},
		{FieldName: "OpcRequestId", RequestName: "opc-request-id", Contribution: "header"},
	}
}

func monitoringTemplateListFields() []generatedruntime.RequestField {
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
		{FieldName: "MonitoringTemplateId", RequestName: "monitoringTemplateId", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func monitoringTemplateUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitoringTemplateId", RequestName: "monitoringTemplateId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateMonitoringTemplateDetails", RequestName: "UpdateMonitoringTemplateDetails", Contribution: "body"},
		{FieldName: "OpcRequestId", RequestName: "opc-request-id", Contribution: "header"},
	}
}

func monitoringTemplateDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MonitoringTemplateId", RequestName: "monitoringTemplateId", Contribution: "path", PreferResourceID: true},
		{FieldName: "OpcRequestId", RequestName: "opc-request-id", Contribution: "header"},
	}
}

func buildMonitoringTemplateCreateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("MonitoringTemplate resource is nil")
	}
	if err := validateMonitoringTemplateSpec(resource.Spec); err != nil {
		return nil, err
	}
	members, err := monitoringTemplateMembers(resource.Spec.Members)
	if err != nil {
		return nil, err
	}
	body := stackmonitoringsdk.CreateMonitoringTemplateDetails{
		DisplayName:                common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		CompartmentId:              common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Destinations:               cloneMonitoringTemplateStringSlice(resource.Spec.Destinations),
		Members:                    members,
		IsAlarmsEnabled:            common.Bool(resource.Spec.IsAlarmsEnabled),
		IsSplitNotificationEnabled: common.Bool(resource.Spec.IsSplitNotificationEnabled),
		FreeformTags:               cloneMonitoringTemplateStringMap(resource.Spec.FreeformTags),
		DefinedTags:                monitoringTemplateDefinedTags(resource.Spec.DefinedTags),
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
	}
	if duration := strings.TrimSpace(resource.Spec.RepeatNotificationDuration); duration != "" {
		body.RepeatNotificationDuration = common.String(duration)
	}
	if messageFormat, ok, err := monitoringTemplateMessageFormat(resource.Spec.MessageFormat); err != nil {
		return nil, err
	} else if ok {
		body.MessageFormat = messageFormat
	}
	return body, nil
}

func buildMonitoringTemplateUpdateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	_ string,
	currentResponse any,
) (any, bool, error) {
	current, err := monitoringTemplateCurrentForUpdate(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}

	body, updateNeeded, err := monitoringTemplateUpdateDetails(resource.Spec, current)
	if err != nil {
		return nil, false, err
	}
	if !updateNeeded {
		return nil, false, nil
	}
	return body, true, nil
}

func monitoringTemplateCurrentForUpdate(
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	currentResponse any,
) (stackmonitoringsdk.MonitoringTemplate, error) {
	if resource == nil {
		return stackmonitoringsdk.MonitoringTemplate{}, fmt.Errorf("MonitoringTemplate resource is nil")
	}
	if err := validateMonitoringTemplateSpec(resource.Spec); err != nil {
		return stackmonitoringsdk.MonitoringTemplate{}, err
	}
	current, ok := monitoringTemplateFromResponse(currentResponse)
	if !ok {
		return stackmonitoringsdk.MonitoringTemplate{}, fmt.Errorf("current MonitoringTemplate response does not expose a MonitoringTemplate body")
	}
	if err := validateMonitoringTemplateCreateOnlyDrift(resource.Spec, current); err != nil {
		return stackmonitoringsdk.MonitoringTemplate{}, err
	}
	return current, nil
}

func monitoringTemplateUpdateDetails(
	spec stackmonitoringv1beta1.MonitoringTemplateSpec,
	current stackmonitoringsdk.MonitoringTemplate,
) (stackmonitoringsdk.UpdateMonitoringTemplateDetails, bool, error) {
	members, err := monitoringTemplateMembers(spec.Members)
	if err != nil {
		return stackmonitoringsdk.UpdateMonitoringTemplateDetails{}, false, err
	}
	messageFormat, messageFormatSet, err := monitoringTemplateMessageFormat(spec.MessageFormat)
	if err != nil {
		return stackmonitoringsdk.UpdateMonitoringTemplateDetails{}, false, err
	}
	body := stackmonitoringsdk.UpdateMonitoringTemplateDetails{}
	updateNeeded := applyMonitoringTemplateStringUpdates(&body, spec, current)
	if applyMonitoringTemplateBoolUpdates(&body, spec, current) {
		updateNeeded = true
	}
	if applyMonitoringTemplateCollectionUpdates(&body, spec, current, members, messageFormat, messageFormatSet) {
		updateNeeded = true
	}
	return body, updateNeeded, nil
}

func applyMonitoringTemplateStringUpdates(
	body *stackmonitoringsdk.UpdateMonitoringTemplateDetails,
	spec stackmonitoringv1beta1.MonitoringTemplateSpec,
	current stackmonitoringsdk.MonitoringTemplate,
) bool {
	return anyMonitoringTemplateUpdate(
		setMonitoringTemplateStringUpdate(&body.DisplayName, current.DisplayName, spec.DisplayName, false),
		setMonitoringTemplateStringUpdate(&body.Description, current.Description, spec.Description, true),
		setMonitoringTemplateStringUpdate(&body.RepeatNotificationDuration, current.RepeatNotificationDuration, spec.RepeatNotificationDuration, true),
	)
}

func applyMonitoringTemplateBoolUpdates(
	body *stackmonitoringsdk.UpdateMonitoringTemplateDetails,
	spec stackmonitoringv1beta1.MonitoringTemplateSpec,
	current stackmonitoringsdk.MonitoringTemplate,
) bool {
	return anyMonitoringTemplateUpdate(
		setMonitoringTemplateBoolUpdate(&body.IsAlarmsEnabled, current.IsAlarmsEnabled, spec.IsAlarmsEnabled),
		setMonitoringTemplateBoolUpdate(&body.IsSplitNotificationEnabled, current.IsSplitNotificationEnabled, spec.IsSplitNotificationEnabled),
	)
}

func applyMonitoringTemplateCollectionUpdates(
	body *stackmonitoringsdk.UpdateMonitoringTemplateDetails,
	spec stackmonitoringv1beta1.MonitoringTemplateSpec,
	current stackmonitoringsdk.MonitoringTemplate,
	members []stackmonitoringsdk.MemberReference,
	messageFormat stackmonitoringsdk.MessageFormatEnum,
	messageFormatSet bool,
) bool {
	return anyMonitoringTemplateUpdate(
		setMonitoringTemplateDestinationsUpdate(&body.Destinations, current.Destinations, spec.Destinations),
		setMonitoringTemplateMembersUpdate(&body.Members, current.Members, members),
		setMonitoringTemplateMessageFormatUpdate(&body.MessageFormat, current.MessageFormat, messageFormat, messageFormatSet),
		setMonitoringTemplateFreeformTagsUpdate(&body.FreeformTags, current.FreeformTags, spec.FreeformTags),
		setMonitoringTemplateDefinedTagsUpdate(&body.DefinedTags, current.DefinedTags, spec.DefinedTags),
	)
}

func anyMonitoringTemplateUpdate(updates ...bool) bool {
	for _, update := range updates {
		if update {
			return true
		}
	}
	return false
}

func setMonitoringTemplateStringUpdate(target **string, current *string, desired string, omitEmpty bool) bool {
	trimmed := strings.TrimSpace(desired)
	if omitEmpty && trimmed == "" {
		return false
	}
	if monitoringTemplateStringPtrEqual(current, trimmed) {
		return false
	}
	*target = common.String(trimmed)
	return true
}

func setMonitoringTemplateDestinationsUpdate(target *[]string, current []string, desired []string) bool {
	desiredClone := cloneMonitoringTemplateStringSlice(desired)
	if reflect.DeepEqual(cloneMonitoringTemplateStringSlice(current), desiredClone) {
		return false
	}
	*target = desiredClone
	return true
}

func setMonitoringTemplateBoolUpdate(target **bool, current *bool, desired bool) bool {
	if current != nil && *current == desired {
		return false
	}
	*target = common.Bool(desired)
	return true
}

func setMonitoringTemplateMembersUpdate(
	target *[]stackmonitoringsdk.MemberReference,
	current []stackmonitoringsdk.MemberReference,
	desired []stackmonitoringsdk.MemberReference,
) bool {
	if monitoringTemplateValuesEqual(desired, current) {
		return false
	}
	*target = desired
	return true
}

func setMonitoringTemplateMessageFormatUpdate(
	target *stackmonitoringsdk.MessageFormatEnum,
	current stackmonitoringsdk.MessageFormatEnum,
	desired stackmonitoringsdk.MessageFormatEnum,
	desiredSet bool,
) bool {
	if !desiredSet || current == desired {
		return false
	}
	*target = desired
	return true
}

func setMonitoringTemplateFreeformTagsUpdate(target *map[string]string, current map[string]string, desired map[string]string) bool {
	if desired == nil {
		return false
	}
	desiredClone := cloneMonitoringTemplateStringMap(desired)
	if reflect.DeepEqual(current, desiredClone) {
		return false
	}
	*target = desiredClone
	return true
}

func setMonitoringTemplateDefinedTagsUpdate(
	target *map[string]map[string]interface{},
	current map[string]map[string]interface{},
	desired map[string]shared.MapValue,
) bool {
	if desired == nil {
		return false
	}
	desiredTags := monitoringTemplateDefinedTags(desired)
	if reflect.DeepEqual(current, desiredTags) {
		return false
	}
	*target = desiredTags
	return true
}

func validateMonitoringTemplateSpec(spec stackmonitoringv1beta1.MonitoringTemplateSpec) error {
	var missing []string
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(spec.Destinations) == 0 {
		missing = append(missing, "destinations")
	}
	if len(spec.Members) == 0 {
		missing = append(missing, "members")
	}
	if len(missing) != 0 {
		return fmt.Errorf("MonitoringTemplate spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	for index, destination := range spec.Destinations {
		if strings.TrimSpace(destination) == "" {
			return fmt.Errorf("MonitoringTemplate spec.destinations[%d] is empty", index)
		}
	}
	if _, err := monitoringTemplateMembers(spec.Members); err != nil {
		return err
	}
	if _, _, err := monitoringTemplateMessageFormat(spec.MessageFormat); err != nil {
		return err
	}
	return nil
}

func monitoringTemplateMembers(members []stackmonitoringv1beta1.MonitoringTemplateMember) ([]stackmonitoringsdk.MemberReference, error) {
	if members == nil {
		return nil, nil
	}
	converted := make([]stackmonitoringsdk.MemberReference, 0, len(members))
	for index, member := range members {
		id := strings.TrimSpace(member.Id)
		if id == "" {
			return nil, fmt.Errorf("MonitoringTemplate spec.members[%d] is missing required field: id", index)
		}
		memberType, ok := stackmonitoringsdk.GetMappingMemberReferenceTypeEnum(strings.TrimSpace(member.Type))
		if !ok {
			return nil, fmt.Errorf("unsupported MonitoringTemplate spec.members[%d].type %q; supported values: %s", index, member.Type, strings.Join(stackmonitoringsdk.GetMemberReferenceTypeEnumStringValues(), ", "))
		}
		converted = append(converted, stackmonitoringsdk.MemberReference{
			Id:            common.String(id),
			Type:          memberType,
			CompositeType: monitoringTemplateOptionalString(member.CompositeType),
		})
	}
	return converted, nil
}

func monitoringTemplateMessageFormat(value string) (stackmonitoringsdk.MessageFormatEnum, bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false, nil
	}
	messageFormat, ok := stackmonitoringsdk.GetMappingMessageFormatEnum(trimmed)
	if !ok {
		return "", false, fmt.Errorf("unsupported MonitoringTemplate messageFormat %q; supported values: %s", value, strings.Join(stackmonitoringsdk.GetMessageFormatEnumStringValues(), ", "))
	}
	return messageFormat, true, nil
}

func guardMonitoringTemplateExistingBeforeCreate(_ context.Context, resource *stackmonitoringv1beta1.MonitoringTemplate) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("MonitoringTemplate resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapMonitoringTemplateReadAndDeleteCalls(hooks *MonitoringTemplateRuntimeHooks) {
	if hooks.Get.Call != nil {
		get := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			response, err := get(ctx, request)
			return response, conservativeMonitoringTemplateNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		list := hooks.List.Call
		hooks.List.Call = listMonitoringTemplatesAllPages(func(ctx context.Context, request stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error) {
			response, err := list(ctx, request)
			return response, conservativeMonitoringTemplateNotFoundError(err, "list")
		})
	}
	if hooks.Delete.Call != nil {
		deleteCall := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request stackmonitoringsdk.DeleteMonitoringTemplateRequest) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeMonitoringTemplateNotFoundError(err, "delete")
		}
	}
}

func installMonitoringTemplateProjectedReadOperations(hooks *MonitoringTemplateRuntimeHooks) {
	if hooks.Get.Call != nil {
		getFields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
		hooks.Read.Get = &generatedruntime.Operation{
			NewRequest: func() any { return &stackmonitoringsdk.GetMonitoringTemplateRequest{} },
			Fields:     getFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*stackmonitoringsdk.GetMonitoringTemplateRequest))
				if err != nil {
					return nil, err
				}
				return monitoringTemplateProjectedResponseFromSDK(response.MonitoringTemplate, response.OpcRequestId), nil
			},
		}
	}
	if hooks.List.Call != nil {
		listFields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
		hooks.Read.List = &generatedruntime.Operation{
			NewRequest: func() any { return &stackmonitoringsdk.ListMonitoringTemplatesRequest{} },
			Fields:     listFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*stackmonitoringsdk.ListMonitoringTemplatesRequest))
				if err != nil {
					return nil, err
				}
				return monitoringTemplateProjectedListResponseFromSDK(response), nil
			},
		}
	}
}

func listMonitoringTemplatesAllPages(
	call func(context.Context, stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error),
) func(context.Context, stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error) {
		var combined stackmonitoringsdk.ListMonitoringTemplatesResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return stackmonitoringsdk.ListMonitoringTemplatesResponse{}, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.RawResponse = response.RawResponse
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

func (c monitoringTemplateDeletePreflightClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("MonitoringTemplate runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c monitoringTemplateDeletePreflightClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("MonitoringTemplate runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c monitoringTemplateDeletePreflightClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
) error {
	if resource == nil {
		return nil
	}
	if currentID := monitoringTemplateTrackedID(resource); currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c monitoringTemplateDeletePreflightClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, stackmonitoringsdk.GetMonitoringTemplateRequest{MonitoringTemplateId: common.String(currentID)})
	return monitoringTemplateAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c monitoringTemplateDeletePreflightClient) rejectAuthShapedList(
	ctx context.Context,
	resource *stackmonitoringv1beta1.MonitoringTemplate,
) error {
	if c.list == nil {
		return nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return nil
	}
	_, err := c.list(ctx, stackmonitoringsdk.ListMonitoringTemplatesRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
	})
	return monitoringTemplateAmbiguousDeleteError(resource, err, "pre-delete list")
}

func handleMonitoringTemplateDeleteError(resource *stackmonitoringv1beta1.MonitoringTemplate, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := monitoringTemplateAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func applyMonitoringTemplateDeleteOutcome(
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	current, ok := monitoringTemplateFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	lifecycleState := strings.ToUpper(string(current.LifecycleState))
	if lifecycleState == string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesDeleted) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending &&
		!monitoringTemplateDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if (stage == generatedruntime.DeleteConfirmStageAfterRequest ||
		stage == generatedruntime.DeleteConfirmStageAlreadyPending) &&
		monitoringTemplateReadbackRetainsDeleteFinalizer(lifecycleState) {
		markMonitoringTemplateTerminating(resource, "OCI resource delete is in progress")
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func monitoringTemplateReadbackRetainsDeleteFinalizer(lifecycleState string) bool {
	switch lifecycleState {
	case string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive),
		string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesInactive),
		string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesUpdating),
		string(stackmonitoringsdk.MonitoringTemplateLifeCycleStatesCreating):
		return true
	default:
		return false
	}
}

func monitoringTemplateAmbiguousDeleteError(resource *stackmonitoringv1beta1.MonitoringTemplate, err error, operation string) error {
	if err == nil || !isAmbiguousMonitoringTemplateNotFound(err) {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return monitoringTemplateAmbiguousNotFoundError{
		message:      fmt.Sprintf("MonitoringTemplate %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func conservativeMonitoringTemplateNotFoundError(err error, operation string) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("MonitoringTemplate %s returned ambiguous 404 NotAuthorizedOrNotFound: %v", strings.TrimSpace(operation), err)
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return monitoringTemplateAmbiguousNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return monitoringTemplateAmbiguousNotFoundError{message: message}
}

func isAmbiguousMonitoringTemplateNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous monitoringTemplateAmbiguousNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func projectMonitoringTemplateStatus(resource *stackmonitoringv1beta1.MonitoringTemplate, response any) error {
	if resource == nil {
		return fmt.Errorf("MonitoringTemplate resource is nil")
	}
	projected, ok := monitoringTemplateProjectionFromResponse(response)
	if !ok {
		return nil
	}
	resource.Status = stackmonitoringv1beta1.MonitoringTemplateStatus{
		OsokStatus:                  resource.Status.OsokStatus,
		Id:                          projected.Id,
		DisplayName:                 projected.DisplayName,
		TenantId:                    projected.TenantId,
		CompartmentId:               projected.CompartmentId,
		Status:                      projected.Status,
		LifecycleState:              projected.LifecycleState,
		Destinations:                cloneMonitoringTemplateStringSlice(projected.Destinations),
		Members:                     cloneMonitoringTemplateAPIMembers(projected.Members),
		TotalAlarmConditions:        projected.TotalAlarmConditions,
		TotalAppliedAlarmConditions: projected.TotalAppliedAlarmConditions,
		TimeCreated:                 projected.TimeCreated,
		TimeUpdated:                 projected.TimeUpdated,
		Description:                 projected.Description,
		IsAlarmsEnabled:             projected.IsAlarmsEnabled,
		IsSplitNotificationEnabled:  projected.IsSplitNotificationEnabled,
		RepeatNotificationDuration:  projected.RepeatNotificationDuration,
		MessageFormat:               projected.MessageFormat,
		FreeformTags:                cloneMonitoringTemplateStringMap(projected.FreeformTags),
		DefinedTags:                 cloneMonitoringTemplateSharedTags(projected.DefinedTags),
		SystemTags:                  cloneMonitoringTemplateSharedTags(projected.SystemTags),
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

func monitoringTemplateProjectedResponseFromSDK(
	current stackmonitoringsdk.MonitoringTemplate,
	opcRequestID *string,
) monitoringTemplateProjectedResponse {
	return monitoringTemplateProjectedResponse{
		MonitoringTemplate: monitoringTemplateStatusProjectionFromSDK(current),
		OpcRequestId:       opcRequestID,
	}
}

func monitoringTemplateProjectedListResponseFromSDK(
	response stackmonitoringsdk.ListMonitoringTemplatesResponse,
) monitoringTemplateProjectedListResponse {
	projected := monitoringTemplateProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.MonitoringTemplateCollection.Items = append(projected.MonitoringTemplateCollection.Items, monitoringTemplateStatusProjectionFromSummary(item))
	}
	return projected
}

func monitoringTemplateStatusProjectionFromSDK(current stackmonitoringsdk.MonitoringTemplate) monitoringTemplateStatusProjection {
	return monitoringTemplateStatusProjection{
		Id:                          monitoringTemplateStringValue(current.Id),
		DisplayName:                 monitoringTemplateStringValue(current.DisplayName),
		TenantId:                    monitoringTemplateStringValue(current.TenantId),
		CompartmentId:               monitoringTemplateStringValue(current.CompartmentId),
		Status:                      string(current.Status),
		LifecycleState:              string(current.LifecycleState),
		Destinations:                cloneMonitoringTemplateStringSlice(current.Destinations),
		Members:                     monitoringTemplateAPIMembers(current.Members),
		TotalAlarmConditions:        monitoringTemplateFloat32Value(current.TotalAlarmConditions),
		TotalAppliedAlarmConditions: monitoringTemplateFloat32Value(current.TotalAppliedAlarmConditions),
		TimeCreated:                 monitoringTemplateTimeString(current.TimeCreated),
		TimeUpdated:                 monitoringTemplateTimeString(current.TimeUpdated),
		Description:                 monitoringTemplateStringValue(current.Description),
		IsAlarmsEnabled:             monitoringTemplateBoolValue(current.IsAlarmsEnabled),
		IsSplitNotificationEnabled:  monitoringTemplateBoolValue(current.IsSplitNotificationEnabled),
		RepeatNotificationDuration:  monitoringTemplateStringValue(current.RepeatNotificationDuration),
		MessageFormat:               string(current.MessageFormat),
		FreeformTags:                cloneMonitoringTemplateStringMap(current.FreeformTags),
		DefinedTags:                 monitoringTemplateSharedTags(current.DefinedTags),
		SystemTags:                  monitoringTemplateSharedTags(current.SystemTags),
	}
}

func monitoringTemplateStatusProjectionFromSummary(current stackmonitoringsdk.MonitoringTemplateSummary) monitoringTemplateStatusProjection {
	return monitoringTemplateStatusProjection{
		Id:                          monitoringTemplateStringValue(current.Id),
		DisplayName:                 monitoringTemplateStringValue(current.DisplayName),
		TenantId:                    monitoringTemplateStringValue(current.TenantId),
		CompartmentId:               monitoringTemplateStringValue(current.CompartmentId),
		Status:                      string(current.Status),
		LifecycleState:              string(current.LifecycleState),
		Destinations:                cloneMonitoringTemplateStringSlice(current.Destinations),
		Members:                     monitoringTemplateAPIMembers(current.Members),
		TotalAlarmConditions:        monitoringTemplateFloat32Value(current.TotalAlarmConditions),
		TotalAppliedAlarmConditions: monitoringTemplateFloat32Value(current.TotalAppliedAlarmConditions),
		TimeCreated:                 monitoringTemplateTimeString(current.TimeCreated),
		TimeUpdated:                 monitoringTemplateTimeString(current.TimeUpdated),
		Description:                 monitoringTemplateStringValue(current.Description),
		FreeformTags:                cloneMonitoringTemplateStringMap(current.FreeformTags),
		DefinedTags:                 monitoringTemplateSharedTags(current.DefinedTags),
		SystemTags:                  monitoringTemplateSharedTags(current.SystemTags),
	}
}

func monitoringTemplateProjectionFromResponse(response any) (monitoringTemplateStatusProjection, bool) {
	switch current := response.(type) {
	case monitoringTemplateProjectedResponse:
		return current.MonitoringTemplate, true
	case *monitoringTemplateProjectedResponse:
		if current == nil {
			return monitoringTemplateStatusProjection{}, false
		}
		return current.MonitoringTemplate, true
	case monitoringTemplateStatusProjection:
		return current, true
	case *monitoringTemplateStatusProjection:
		if current == nil {
			return monitoringTemplateStatusProjection{}, false
		}
		return *current, true
	default:
		if template, ok := monitoringTemplateFromResponse(response); ok {
			return monitoringTemplateStatusProjectionFromSDK(template), true
		}
		return monitoringTemplateStatusProjection{}, false
	}
}

func monitoringTemplateFromResponse(response any) (stackmonitoringsdk.MonitoringTemplate, bool) {
	if template, ok := monitoringTemplateFromMutationResponse(response); ok {
		return template, true
	}
	if template, ok := monitoringTemplateFromReadResponse(response); ok {
		return template, true
	}
	if template, ok := monitoringTemplateFromListItem(response); ok {
		return template, true
	}
	return monitoringTemplateFromProjectedValue(response)
}

func monitoringTemplateFromMutationResponse(response any) (stackmonitoringsdk.MonitoringTemplate, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.CreateMonitoringTemplateResponse:
		return current.MonitoringTemplate, true
	case *stackmonitoringsdk.CreateMonitoringTemplateResponse:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return current.MonitoringTemplate, true
	case stackmonitoringsdk.UpdateMonitoringTemplateResponse:
		return current.MonitoringTemplate, true
	case *stackmonitoringsdk.UpdateMonitoringTemplateResponse:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return current.MonitoringTemplate, true
	default:
		return stackmonitoringsdk.MonitoringTemplate{}, false
	}
}

func monitoringTemplateFromReadResponse(response any) (stackmonitoringsdk.MonitoringTemplate, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.GetMonitoringTemplateResponse:
		return current.MonitoringTemplate, true
	case *stackmonitoringsdk.GetMonitoringTemplateResponse:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return current.MonitoringTemplate, true
	case stackmonitoringsdk.MonitoringTemplate:
		return current, true
	case *stackmonitoringsdk.MonitoringTemplate:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return *current, true
	default:
		return stackmonitoringsdk.MonitoringTemplate{}, false
	}
}

func monitoringTemplateFromListItem(response any) (stackmonitoringsdk.MonitoringTemplate, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.MonitoringTemplateSummary:
		return monitoringTemplateFromSummary(current), true
	case *stackmonitoringsdk.MonitoringTemplateSummary:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return monitoringTemplateFromSummary(*current), true
	default:
		return stackmonitoringsdk.MonitoringTemplate{}, false
	}
}

func monitoringTemplateFromProjectedValue(response any) (stackmonitoringsdk.MonitoringTemplate, bool) {
	switch current := response.(type) {
	case monitoringTemplateProjectedResponse:
		return monitoringTemplateFromProjection(current.MonitoringTemplate), true
	case *monitoringTemplateProjectedResponse:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return monitoringTemplateFromProjection(current.MonitoringTemplate), true
	case monitoringTemplateStatusProjection:
		return monitoringTemplateFromProjection(current), true
	case *monitoringTemplateStatusProjection:
		if current == nil {
			return stackmonitoringsdk.MonitoringTemplate{}, false
		}
		return monitoringTemplateFromProjection(*current), true
	default:
		return stackmonitoringsdk.MonitoringTemplate{}, false
	}
}

func monitoringTemplateFromSummary(summary stackmonitoringsdk.MonitoringTemplateSummary) stackmonitoringsdk.MonitoringTemplate {
	return stackmonitoringsdk.MonitoringTemplate{
		Id:                          summary.Id,
		DisplayName:                 summary.DisplayName,
		TenantId:                    summary.TenantId,
		CompartmentId:               summary.CompartmentId,
		Status:                      summary.Status,
		LifecycleState:              summary.LifecycleState,
		Destinations:                cloneMonitoringTemplateStringSlice(summary.Destinations),
		Members:                     cloneMonitoringTemplateMembers(summary.Members),
		TotalAlarmConditions:        summary.TotalAlarmConditions,
		TotalAppliedAlarmConditions: summary.TotalAppliedAlarmConditions,
		TimeCreated:                 summary.TimeCreated,
		TimeUpdated:                 summary.TimeUpdated,
		Description:                 summary.Description,
		FreeformTags:                cloneMonitoringTemplateStringMap(summary.FreeformTags),
		DefinedTags:                 cloneMonitoringTemplateOCITags(summary.DefinedTags),
		SystemTags:                  cloneMonitoringTemplateOCITags(summary.SystemTags),
	}
}

func monitoringTemplateFromProjection(projection monitoringTemplateStatusProjection) stackmonitoringsdk.MonitoringTemplate {
	return stackmonitoringsdk.MonitoringTemplate{
		Id:                          common.String(projection.Id),
		DisplayName:                 common.String(projection.DisplayName),
		TenantId:                    common.String(projection.TenantId),
		CompartmentId:               common.String(projection.CompartmentId),
		Status:                      stackmonitoringsdk.MonitoringTemplateLifeCycleDetailsEnum(projection.Status),
		LifecycleState:              stackmonitoringsdk.MonitoringTemplateLifeCycleStatesEnum(projection.LifecycleState),
		Destinations:                cloneMonitoringTemplateStringSlice(projection.Destinations),
		Members:                     monitoringTemplateSDKMembers(projection.Members),
		TotalAlarmConditions:        common.Float32(projection.TotalAlarmConditions),
		TotalAppliedAlarmConditions: common.Float32(projection.TotalAppliedAlarmConditions),
		Description:                 monitoringTemplateOptionalString(projection.Description),
		IsAlarmsEnabled:             common.Bool(projection.IsAlarmsEnabled),
		IsSplitNotificationEnabled:  common.Bool(projection.IsSplitNotificationEnabled),
		RepeatNotificationDuration:  monitoringTemplateOptionalString(projection.RepeatNotificationDuration),
		MessageFormat:               stackmonitoringsdk.MessageFormatEnum(projection.MessageFormat),
		FreeformTags:                cloneMonitoringTemplateStringMap(projection.FreeformTags),
		DefinedTags:                 monitoringTemplateDefinedTags(projection.DefinedTags),
		SystemTags:                  monitoringTemplateDefinedTags(projection.SystemTags),
	}
}

func validateMonitoringTemplateCreateOnlyDriftForResponse(
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("MonitoringTemplate resource is nil")
	}
	current, ok := monitoringTemplateFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current MonitoringTemplate response does not expose a MonitoringTemplate body")
	}
	return validateMonitoringTemplateCreateOnlyDrift(resource.Spec, current)
}

func validateMonitoringTemplateCreateOnlyDrift(
	spec stackmonitoringv1beta1.MonitoringTemplateSpec,
	current stackmonitoringsdk.MonitoringTemplate,
) error {
	if monitoringTemplateStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("MonitoringTemplate create-only field drift is not supported: compartmentId")
}

func markMonitoringTemplateTerminating(resource *stackmonitoringv1beta1.MonitoringTemplate, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
}

func monitoringTemplateDeleteAlreadyPending(resource *stackmonitoringv1beta1.MonitoringTemplate) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func monitoringTemplateTrackedID(resource *stackmonitoringv1beta1.MonitoringTemplate) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func monitoringTemplateDefinedTags(values map[string]shared.MapValue) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&values)
}

func monitoringTemplateSharedTags(values map[string]map[string]interface{}) map[string]shared.MapValue {
	if values == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(values))
	for namespace, entries := range values {
		converted[namespace] = make(shared.MapValue, len(entries))
		for key, value := range entries {
			converted[namespace][key] = fmt.Sprint(value)
		}
	}
	return converted
}

func cloneMonitoringTemplateSharedTags(values map[string]shared.MapValue) map[string]shared.MapValue {
	if values == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(values))
	for namespace, entries := range values {
		cloned[namespace] = make(shared.MapValue, len(entries))
		for key, value := range entries {
			cloned[namespace][key] = value
		}
	}
	return cloned
}

func cloneMonitoringTemplateOCITags(values map[string]map[string]interface{}) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(values))
	for namespace, entries := range values {
		if entries == nil {
			cloned[namespace] = nil
			continue
		}
		cloned[namespace] = make(map[string]interface{}, len(entries))
		for key, value := range entries {
			cloned[namespace][key] = value
		}
	}
	return cloned
}

func monitoringTemplateAPIMembers(values []stackmonitoringsdk.MemberReference) []stackmonitoringv1beta1.MonitoringTemplateMember {
	if values == nil {
		return nil
	}
	members := make([]stackmonitoringv1beta1.MonitoringTemplateMember, 0, len(values))
	for _, value := range values {
		members = append(members, stackmonitoringv1beta1.MonitoringTemplateMember{
			Id:            monitoringTemplateStringValue(value.Id),
			Type:          string(value.Type),
			CompositeType: monitoringTemplateStringValue(value.CompositeType),
		})
	}
	return members
}

func cloneMonitoringTemplateAPIMembers(values []stackmonitoringv1beta1.MonitoringTemplateMember) []stackmonitoringv1beta1.MonitoringTemplateMember {
	if values == nil {
		return nil
	}
	return append([]stackmonitoringv1beta1.MonitoringTemplateMember(nil), values...)
}

func monitoringTemplateSDKMembers(values []stackmonitoringv1beta1.MonitoringTemplateMember) []stackmonitoringsdk.MemberReference {
	if values == nil {
		return nil
	}
	members, _ := monitoringTemplateMembers(values)
	return members
}

func cloneMonitoringTemplateMembers(values []stackmonitoringsdk.MemberReference) []stackmonitoringsdk.MemberReference {
	if values == nil {
		return nil
	}
	return append([]stackmonitoringsdk.MemberReference(nil), values...)
}

func cloneMonitoringTemplateStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneMonitoringTemplateStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func monitoringTemplateOptionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(strings.TrimSpace(value))
}

func monitoringTemplateStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func monitoringTemplateStringPtrEqual(current *string, desired string) bool {
	return monitoringTemplateStringValue(current) == strings.TrimSpace(desired)
}

func monitoringTemplateBoolValue(value *bool) bool {
	return value != nil && *value
}

func monitoringTemplateFloat32Value(value *float32) float32 {
	if value == nil {
		return 0
	}
	return *value
}

func monitoringTemplateTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func monitoringTemplateValuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}
