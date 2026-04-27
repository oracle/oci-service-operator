/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const channelDefaultRequeueDuration = time.Minute

var channelMutableSpecFields = []string{
	"name",
	"description",
	"sessionExpiryDurationInMilliseconds",
	"freeformTags",
	"definedTags",
	"msaAppId",
	"msaAppPassword",
	"botId",
	"isClientAuthenticationEnabled",
	"maxTokenExpirationTimeInMinutes",
	"allowedDomains",
	"appSecret",
	"pageAccessToken",
	"isAuthenticatedUserId",
	"outboundUrl",
	"domainName",
	"hostNamePrefix",
	"userName",
	"password",
	"clientType",
	"clientId",
	"signingSecret",
	"clientSecret",
	"authSuccessUrl",
	"authErrorUrl",
	"host",
	"port",
	"totalSessionCount",
	"authenticationProviderName",
	"channelService",
	"eventSinkBotIds",
	"inboundMessageTopic",
	"outboundMessageTopic",
	"bootstrapServers",
	"securityProtocol",
	"saslMechanism",
	"tenancyName",
	"streamPoolId",
	"authToken",
	"accountSID",
	"phoneNumber",
	"isMmsEnabled",
	"originalConnectorsUrl",
	"payloadVersion",
}

func init() {
	registerChannelRuntimeHooksMutator(applyChannelRuntimeHooks)
}

func applyChannelRuntimeHooks(manager *ChannelServiceManager, hooks *ChannelRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = channelRuntimeSemantics()
	hooks.Create.Fields = channelCreateFields()
	hooks.Get.Fields = channelGetFields()
	hooks.List.Fields = channelListFields()
	hooks.Update.Fields = channelUpdateFields()
	hooks.Delete.Fields = channelDeleteFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ChannelServiceClient) ChannelServiceClient {
		return channelRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
			log:      manager.Log,
		}
	})
}

func channelRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "oda",
		FormalSlug:        "channel",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(odasdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(odasdk.LifecycleStateActive), string(odasdk.LifecycleStateInactive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(odasdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  append([]string(nil), channelMutableSpecFields...),
			ForceNew: []string{"odaInstanceId", "type"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func channelOdaInstanceIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "OdaInstanceId",
		RequestName:      "odaInstanceId",
		Contribution:     "path",
		PreferResourceID: false,
		LookupPaths:      []string{"spec.odaInstanceId", "odaInstanceId"},
	}
}

func channelIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "ChannelId",
		RequestName:      "channelId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.id", "status.status.ocid", "id", "ocid"},
	}
}

func channelCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		channelOdaInstanceIDField(),
		{FieldName: "CreateChannelDetails", RequestName: "CreateChannelDetails", Contribution: "body", PreferResourceID: false},
	}
}

func channelGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		channelOdaInstanceIDField(),
		channelIDField(),
	}
}

func channelListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		channelOdaInstanceIDField(),
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: false},
		{FieldName: "Name", RequestName: "name", Contribution: "query", PreferResourceID: false, LookupPaths: []string{"spec.name", "name"}},
		{FieldName: "Category", RequestName: "category", Contribution: "query", PreferResourceID: false},
		{FieldName: "Type", RequestName: "type", Contribution: "query", PreferResourceID: false, LookupPaths: []string{"spec.type", "type"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
		{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
	}
}

func channelUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		channelOdaInstanceIDField(),
		channelIDField(),
		{FieldName: "UpdateChannelDetails", RequestName: "UpdateChannelDetails", Contribution: "body", PreferResourceID: false},
	}
}

func channelDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		channelOdaInstanceIDField(),
		channelIDField(),
	}
}

type channelRuntimeClient struct {
	delegate ChannelServiceClient
	hooks    ChannelRuntimeHooks
	log      loggerutil.OSOKLogger
}

var _ ChannelServiceClient = channelRuntimeClient{}

func (c channelRuntimeClient) CreateOrUpdate(ctx context.Context, resource *odav1beta1.Channel, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Channel resource is nil")
	}
	if c.hooks.Create.Call == nil || c.hooks.Get.Call == nil || c.hooks.List.Call == nil || c.hooks.Update.Call == nil {
		return c.fail(resource, fmt.Errorf("Channel runtime OCI operation hooks are not configured"))
	}

	odaInstanceID := strings.TrimSpace(resource.Spec.OdaInstanceId)
	if odaInstanceID == "" {
		return c.fail(resource, fmt.Errorf("Channel spec.odaInstanceId is required"))
	}
	if strings.TrimSpace(resource.Spec.Name) == "" {
		return c.fail(resource, fmt.Errorf("Channel spec.name is required"))
	}

	currentID := channelCurrentID(resource)
	var current any
	if currentID != "" {
		read, err := c.get(ctx, odaInstanceID, currentID)
		if err != nil {
			if isChannelNotFound(err) {
				return c.fail(resource, fmt.Errorf("tracked Channel %q was not found under odaInstanceId %q; refusing to create a replacement because odaInstanceId is create-only", currentID, odaInstanceID))
			}
			return c.fail(resource, fmt.Errorf("get Channel %q: %w", currentID, err))
		}
		current = read
	} else {
		listItem, found, err := c.findByName(ctx, resource)
		if err != nil {
			return c.fail(resource, err)
		}
		if found {
			currentID = channelResponseID(listItem)
			current = listItem
			if currentID != "" {
				read, err := c.get(ctx, odaInstanceID, currentID)
				if err != nil && !isChannelNotFound(err) {
					return c.fail(resource, fmt.Errorf("get listed Channel %q: %w", currentID, err))
				}
				if err == nil {
					current = read
				}
			}
		}
	}

	if currentID == "" {
		return c.create(ctx, resource, req.Namespace, odaInstanceID)
	}
	return c.reconcileExisting(ctx, resource, req.Namespace, odaInstanceID, currentID, current)
}

func (c channelRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.Channel) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c channelRuntimeClient) get(ctx context.Context, odaInstanceID string, channelID string) (odasdk.GetChannelResponse, error) {
	return c.hooks.Get.Call(ctx, odasdk.GetChannelRequest{
		OdaInstanceId: common.String(odaInstanceID),
		ChannelId:     common.String(channelID),
	})
}

func (c channelRuntimeClient) findByName(ctx context.Context, resource *odav1beta1.Channel) (odasdk.ChannelSummary, bool, error) {
	response, err := c.hooks.List.Call(ctx, odasdk.ListChannelsRequest{
		OdaInstanceId: common.String(strings.TrimSpace(resource.Spec.OdaInstanceId)),
		Name:          common.String(strings.TrimSpace(resource.Spec.Name)),
	})
	if err != nil {
		return odasdk.ChannelSummary{}, false, fmt.Errorf("list Channels in odaInstanceId %q: %w", resource.Spec.OdaInstanceId, err)
	}

	var matches []odasdk.ChannelSummary
	for _, item := range response.Items {
		if strings.EqualFold(strings.TrimSpace(channelStringValue(item.Name)), strings.TrimSpace(resource.Spec.Name)) &&
			!channelLifecycleIsDeleted(string(item.LifecycleState)) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return odasdk.ChannelSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return odasdk.ChannelSummary{}, false, fmt.Errorf("found %d Channels named %q in odaInstanceId %q", len(matches), resource.Spec.Name, resource.Spec.OdaInstanceId)
	}
}

func (c channelRuntimeClient) create(ctx context.Context, resource *odav1beta1.Channel, namespace string, odaInstanceID string) (servicemanager.OSOKResponse, error) {
	body, err := channelCreateDetails(ctx, resource, namespace)
	if err != nil {
		return c.fail(resource, err)
	}

	request := odasdk.CreateChannelRequest{
		OdaInstanceId:        common.String(odaInstanceID),
		CreateChannelDetails: body,
	}
	if retryToken := strings.TrimSpace(string(resource.UID)); retryToken != "" {
		request.OpcRetryToken = common.String(retryToken)
	}

	response, err := c.hooks.Create.Call(ctx, request)
	if err != nil {
		return c.fail(resource, fmt.Errorf("create Channel %q: %w", resource.Spec.Name, err))
	}

	followUp := any(response)
	if channelID := channelResponseID(response); channelID != "" {
		read, err := c.get(ctx, odaInstanceID, channelID)
		if err != nil && !isChannelNotFound(err) {
			return c.fail(resource, fmt.Errorf("read Channel %q after create: %w", channelID, err))
		}
		if err == nil {
			followUp = read
		}
	}
	return c.applyObservation(resource, followUp, shared.Provisioning)
}

func (c channelRuntimeClient) reconcileExisting(ctx context.Context, resource *odav1beta1.Channel, namespace string, odaInstanceID string, channelID string, current any) (servicemanager.OSOKResponse, error) {
	state := channelLifecycleState(current)
	switch {
	case channelLifecycleIsProvisioning(state):
		return c.applyObservation(resource, current, shared.Provisioning)
	case channelLifecycleIsUpdating(state):
		return c.applyObservation(resource, current, shared.Updating)
	case channelLifecycleIsDeleting(state):
		return c.applyObservation(resource, current, shared.Terminating)
	case channelLifecycleIsTerminalDeleted(state):
		return c.fail(resource, fmt.Errorf("tracked Channel %q is %s", channelID, state))
	case channelLifecycleIsFailed(state):
		if err := c.projectStatus(resource, current); err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		c.markFailed(resource, fmt.Sprintf("Channel %q is %s", resource.Spec.Name, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	body, updateNeeded, err := channelUpdateDetails(ctx, resource, namespace, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.applyObservation(resource, current, shared.Active)
	}

	response, err := c.hooks.Update.Call(ctx, odasdk.UpdateChannelRequest{
		OdaInstanceId:        common.String(odaInstanceID),
		ChannelId:            common.String(channelID),
		UpdateChannelDetails: body,
	})
	if err != nil {
		return c.fail(resource, fmt.Errorf("update Channel %q: %w", channelID, err))
	}

	followUp := any(response)
	read, err := c.get(ctx, odaInstanceID, channelID)
	if err != nil && !isChannelNotFound(err) {
		return c.fail(resource, fmt.Errorf("read Channel %q after update: %w", channelID, err))
	}
	if err == nil {
		followUp = read
	}
	return c.applyObservation(resource, followUp, shared.Updating)
}

func (c channelRuntimeClient) applyObservation(resource *odav1beta1.Channel, response any, fallback shared.OSOKConditionType) (servicemanager.OSOKResponse, error) {
	if err := c.projectStatus(resource, response); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	status := &resource.Status.OsokStatus
	servicemanager.RecordResponseOpcRequestID(status, response)
	resourceID := channelResponseID(response)
	if resourceID == "" {
		resourceID = channelCurrentID(resource)
	}
	if resourceID != "" {
		status.Ocid = shared.OCID(resourceID)
	}

	now := metav1.Now()
	if resourceID != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}

	state := strings.ToUpper(channelLifecycleState(response))
	message := channelLifecycleMessage(resource, state, fallback)
	switch {
	case state == "":
		return c.markCondition(resource, fallback, message), nil
	case channelLifecycleIsProvisioning(state):
		return c.markLifecycleAsync(resource, state, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, message), nil
	case channelLifecycleIsUpdating(state):
		return c.markLifecycleAsync(resource, state, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, message), nil
	case channelLifecycleIsDeleting(state):
		return c.markLifecycleAsync(resource, state, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, message), nil
	case channelLifecycleIsTerminalDeleted(state):
		return c.markLifecycleAsync(resource, state, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded, message), nil
	case channelLifecycleIsActive(state):
		return c.markCondition(resource, shared.Active, message), nil
	case channelLifecycleIsFailed(state):
		c.markFailed(resource, message)
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	default:
		c.markFailed(resource, fmt.Sprintf("formal lifecycle state %q is not modeled: %s", state, message))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}
}

func (c channelRuntimeClient) projectStatus(resource *odav1beta1.Channel, response any) error {
	body := channelResponseBody(response)
	if body == nil {
		return nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal Channel response body: %w", err)
	}
	if err := json.Unmarshal(payload, &resource.Status); err != nil {
		return fmt.Errorf("project Channel response body into status: %w", err)
	}
	if id := channelResponseID(body); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	return nil
}

func (c channelRuntimeClient) markCondition(resource *odav1beta1.Channel, condition shared.OSOKConditionType, message string) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatusForChannel(condition), "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   channelShouldRequeue(condition),
		RequeueDuration: channelDefaultRequeueDuration,
	}
}

func (c channelRuntimeClient) markLifecycleAsync(resource *odav1beta1.Channel, state string, phase shared.OSOKAsyncPhase, class shared.OSOKAsyncNormalizedClass, message string) servicemanager.OSOKResponse {
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           servicemanager.ResolveAsyncPhase(&resource.Status.OsokStatus, phase),
		RawStatus:       strings.TrimSpace(state),
		NormalizedClass: class,
		Message:         message,
		UpdatedAt:       &now,
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: channelDefaultRequeueDuration,
	}
}

func (c channelRuntimeClient) markFailed(resource *odav1beta1.Channel, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", status.Message, c.log)
}

func (c channelRuntimeClient) fail(resource *odav1beta1.Channel, err error) (servicemanager.OSOKResponse, error) {
	if resource != nil && err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markFailed(resource, err.Error())
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func channelCreateDetails(ctx context.Context, resource *odav1beta1.Channel, namespace string) (odasdk.CreateChannelDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("Channel resource is nil")
	}
	resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return nil, fmt.Errorf("resolve Channel spec values: %w", err)
	}
	channelType := normalizeChannelType(resource.Spec.Type)
	if channelType == "" {
		return nil, fmt.Errorf("Channel spec.type is required")
	}
	body, err := createChannelDetailsForType(channelType, resolved)
	if err != nil {
		return nil, err
	}
	if err := validateChannelSpecSupportedByBody(resource, channelType, body); err != nil {
		return nil, err
	}
	return body, nil
}

func channelUpdateDetails(ctx context.Context, resource *odav1beta1.Channel, namespace string, current any) (odasdk.UpdateChannelDetails, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("Channel resource is nil")
	}

	currentType := normalizeChannelType(channelResponseType(current))
	desiredType := normalizeChannelType(resource.Spec.Type)
	if desiredType == "" {
		desiredType = currentType
	}
	if desiredType == "" {
		return nil, false, fmt.Errorf("Channel type is required for update")
	}
	if currentType != "" && desiredType != currentType {
		return nil, false, fmt.Errorf("Channel type is create-only: desired %q, current %q", desiredType, currentType)
	}

	resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return nil, false, fmt.Errorf("resolve Channel spec values: %w", err)
	}
	desired, err := updateChannelDetailsForType(desiredType, resolved)
	if err != nil {
		return nil, false, err
	}
	if err := validateChannelSpecSupportedByBody(resource, desiredType, desired); err != nil {
		return nil, false, err
	}
	currentUpdate, err := updateChannelDetailsForType(desiredType, channelResponseBody(current))
	if err != nil {
		return nil, false, fmt.Errorf("decode current Channel %q details: %w", desiredType, err)
	}
	updateNeeded, err := channelUpdateNeeded(desired, currentUpdate)
	if err != nil {
		return nil, false, err
	}
	return desired, updateNeeded, nil
}

func validateChannelSpecSupportedByBody(resource *odav1beta1.Channel, channelType string, body any) error {
	specValues, err := channelJSONObject(resource.Spec)
	if err != nil {
		return fmt.Errorf("marshal Channel spec for %s compatibility: %w", channelType, err)
	}
	bodyValues, err := channelJSONObject(body)
	if err != nil {
		return fmt.Errorf("marshal Channel %s request body: %w", channelType, err)
	}

	unsupported := make([]string, 0)
	for key, value := range specValues {
		switch key {
		case "odaInstanceId", "type":
			continue
		}
		if !channelMeaningfulValue(value) {
			continue
		}
		if _, ok := bodyValues[key]; !ok {
			unsupported = append(unsupported, key)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Strings(unsupported)
	return fmt.Errorf("Channel type %q does not support spec field(s): %s", channelType, strings.Join(unsupported, ", "))
}

func createChannelDetailsForType(channelType string, value any) (odasdk.CreateChannelDetails, error) {
	switch channelType {
	case string(odasdk.ChannelTypeAndroid):
		return decodeChannelDetails[odasdk.CreateAndroidChannelDetails](value)
	case string(odasdk.ChannelTypeAppevent):
		return decodeChannelDetails[odasdk.CreateAppEventChannelDetails](value)
	case string(odasdk.ChannelTypeApplication):
		return decodeChannelDetails[odasdk.CreateApplicationChannelDetails](value)
	case string(odasdk.ChannelTypeCortana):
		return decodeChannelDetails[odasdk.CreateCortanaChannelDetails](value)
	case string(odasdk.ChannelTypeFacebook):
		return decodeChannelDetails[odasdk.CreateFacebookChannelDetails](value)
	case string(odasdk.ChannelTypeIos):
		return decodeChannelDetails[odasdk.CreateIosChannelDetails](value)
	case string(odasdk.ChannelTypeMsteams):
		return decodeChannelDetails[odasdk.CreateMsTeamsChannelDetails](value)
	case string(odasdk.ChannelTypeOss):
		return decodeChannelDetails[odasdk.CreateOssChannelDetails](value)
	case string(odasdk.ChannelTypeOsvc):
		return decodeChannelDetails[odasdk.CreateOsvcChannelDetails](value)
	case string(odasdk.ChannelTypeServicecloud):
		return decodeChannelDetails[odasdk.CreateServiceCloudChannelDetails](value)
	case string(odasdk.ChannelTypeSlack):
		return decodeChannelDetails[odasdk.CreateSlackChannelDetails](value)
	case string(odasdk.ChannelTypeTwilio):
		return decodeChannelDetails[odasdk.CreateTwilioChannelDetails](value)
	case string(odasdk.ChannelTypeWeb):
		return decodeChannelDetails[odasdk.CreateWebChannelDetails](value)
	case string(odasdk.ChannelTypeWebhook):
		return decodeChannelDetails[odasdk.CreateWebhookChannelDetails](value)
	case string(odasdk.ChannelTypeTest):
		return nil, fmt.Errorf("Channel type %q is read-only in the OCI SDK and cannot be created by OSOK", channelType)
	default:
		return nil, fmt.Errorf("unsupported Channel type %q", channelType)
	}
}

func updateChannelDetailsForType(channelType string, value any) (odasdk.UpdateChannelDetails, error) {
	switch channelType {
	case string(odasdk.ChannelTypeAndroid):
		return decodeChannelDetails[odasdk.UpdateAndroidChannelDetails](value)
	case string(odasdk.ChannelTypeAppevent):
		return decodeChannelDetails[odasdk.UpdateAppEventChannelDetails](value)
	case string(odasdk.ChannelTypeApplication):
		return decodeChannelDetails[odasdk.UpdateApplicationChannelDetails](value)
	case string(odasdk.ChannelTypeCortana):
		return decodeChannelDetails[odasdk.UpdateCortanaChannelDetails](value)
	case string(odasdk.ChannelTypeFacebook):
		return decodeChannelDetails[odasdk.UpdateFacebookChannelDetails](value)
	case string(odasdk.ChannelTypeIos):
		return decodeChannelDetails[odasdk.UpdateIosChannelDetails](value)
	case string(odasdk.ChannelTypeMsteams):
		return decodeChannelDetails[odasdk.UpdateMsTeamsChannelDetails](value)
	case string(odasdk.ChannelTypeOss):
		return decodeChannelDetails[odasdk.UpdateOssChannelDetails](value)
	case string(odasdk.ChannelTypeOsvc):
		return decodeChannelDetails[odasdk.UpdateOsvcChannelDetails](value)
	case string(odasdk.ChannelTypeServicecloud):
		return decodeChannelDetails[odasdk.UpdateServiceCloudChannelDetails](value)
	case string(odasdk.ChannelTypeSlack):
		return decodeChannelDetails[odasdk.UpdateSlackChannelDetails](value)
	case string(odasdk.ChannelTypeTwilio):
		return decodeChannelDetails[odasdk.UpdateTwilioChannelDetails](value)
	case string(odasdk.ChannelTypeWeb):
		return decodeChannelDetails[odasdk.UpdateWebChannelDetails](value)
	case string(odasdk.ChannelTypeWebhook):
		return decodeChannelDetails[odasdk.UpdateWebhookChannelDetails](value)
	case string(odasdk.ChannelTypeTest):
		return nil, fmt.Errorf("Channel type %q is read-only in the OCI SDK and cannot be updated by OSOK", channelType)
	default:
		return nil, fmt.Errorf("unsupported Channel type %q", channelType)
	}
}

func decodeChannelDetails[T any](value any) (T, error) {
	var decoded T
	payload, err := json.Marshal(value)
	if err != nil {
		return decoded, fmt.Errorf("marshal Channel details: %w", err)
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return decoded, fmt.Errorf("decode Channel details into %T: %w", decoded, err)
	}
	return decoded, nil
}

func channelUpdateNeeded(desired odasdk.UpdateChannelDetails, current odasdk.UpdateChannelDetails) (bool, error) {
	desiredMap, err := channelJSONObject(desired)
	if err != nil {
		return false, fmt.Errorf("marshal desired Channel update details: %w", err)
	}
	currentMap, err := channelJSONObject(current)
	if err != nil {
		return false, fmt.Errorf("marshal current Channel update details: %w", err)
	}

	for key, desiredValue := range desiredMap {
		if key == "type" || !channelMeaningfulValue(desiredValue) {
			continue
		}
		if !reflect.DeepEqual(desiredValue, currentMap[key]) {
			return true, nil
		}
	}
	return false, nil
}

func channelResponseBody(response any) any {
	switch typed := response.(type) {
	case nil:
		return nil
	case odasdk.CreateChannelResponse:
		return typed.CreateChannelResult
	case *odasdk.CreateChannelResponse:
		if typed == nil {
			return nil
		}
		return typed.CreateChannelResult
	case odasdk.GetChannelResponse:
		return typed.Channel
	case *odasdk.GetChannelResponse:
		if typed == nil {
			return nil
		}
		return typed.Channel
	case odasdk.UpdateChannelResponse:
		return typed.Channel
	case *odasdk.UpdateChannelResponse:
		if typed == nil {
			return nil
		}
		return typed.Channel
	default:
		return response
	}
}

func channelResponseID(response any) string {
	body := channelResponseBody(response)
	switch typed := body.(type) {
	case odasdk.Channel:
		return channelStringValue(typed.GetId())
	case odasdk.CreateChannelResult:
		return channelStringValue(typed.GetId())
	case odasdk.ChannelSummary:
		return channelStringValue(typed.Id)
	}
	values, _ := channelJSONObject(body)
	return firstNonEmptyChannelString(values, "id", "ocid")
}

func channelResponseName(response any) string {
	body := channelResponseBody(response)
	switch typed := body.(type) {
	case odasdk.Channel:
		return channelStringValue(typed.GetName())
	case odasdk.CreateChannelResult:
		return channelStringValue(typed.GetName())
	case odasdk.ChannelSummary:
		return channelStringValue(typed.Name)
	}
	values, _ := channelJSONObject(body)
	return firstNonEmptyChannelString(values, "name", "displayName")
}

func channelResponseType(response any) string {
	body := channelResponseBody(response)
	switch typed := body.(type) {
	case odasdk.ChannelSummary:
		return string(typed.Type)
	}
	values, _ := channelJSONObject(body)
	return firstNonEmptyChannelString(values, "type")
}

func channelLifecycleState(response any) string {
	body := channelResponseBody(response)
	switch typed := body.(type) {
	case odasdk.Channel:
		return strings.ToUpper(string(typed.GetLifecycleState()))
	case odasdk.CreateChannelResult:
		return strings.ToUpper(string(typed.GetLifecycleState()))
	case odasdk.ChannelSummary:
		return strings.ToUpper(string(typed.LifecycleState))
	}
	values, _ := channelJSONObject(body)
	return strings.ToUpper(firstNonEmptyChannelString(values, "lifecycleState", "status"))
}

func channelCurrentID(resource *odav1beta1.Channel) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func channelLifecycleMessage(resource *odav1beta1.Channel, state string, fallback shared.OSOKConditionType) string {
	name := strings.TrimSpace(channelResponseName(resource.Status))
	if name == "" && resource != nil {
		name = strings.TrimSpace(resource.Spec.Name)
	}
	if state != "" && name != "" {
		return fmt.Sprintf("Channel %q is %s", name, state)
	}
	switch fallback {
	case shared.Provisioning:
		return "Channel provisioning is in progress"
	case shared.Updating:
		return "Channel update is in progress"
	case shared.Terminating:
		return "Channel delete is in progress"
	default:
		return "Channel is active"
	}
}

func channelLifecycleIsProvisioning(state string) bool {
	return strings.EqualFold(state, string(odasdk.LifecycleStateCreating))
}

func channelLifecycleIsUpdating(state string) bool {
	return strings.EqualFold(state, string(odasdk.LifecycleStateUpdating))
}

func channelLifecycleIsActive(state string) bool {
	return strings.EqualFold(state, string(odasdk.LifecycleStateActive)) ||
		strings.EqualFold(state, string(odasdk.LifecycleStateInactive))
}

func channelLifecycleIsDeleting(state string) bool {
	return strings.EqualFold(state, string(odasdk.LifecycleStateDeleting))
}

func channelLifecycleIsDeleted(state string) bool {
	return channelLifecycleIsDeleting(state) || channelLifecycleIsTerminalDeleted(state)
}

func channelLifecycleIsTerminalDeleted(state string) bool {
	return strings.EqualFold(state, string(odasdk.LifecycleStateDeleted))
}

func channelLifecycleIsFailed(state string) bool {
	return strings.EqualFold(state, string(odasdk.LifecycleStateFailed))
}

func isChannelNotFound(err error) bool {
	if err == nil {
		return false
	}
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func normalizeChannelType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func channelJSONObject(value any) (map[string]any, error) {
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

func firstNonEmptyChannelString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		default:
			text := strings.TrimSpace(fmt.Sprint(typed))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func channelMeaningfulValue(value any) bool {
	if value == nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case bool:
		return true
	case float64:
		return typed != 0
	case map[string]any:
		for _, nested := range typed {
			if channelMeaningfulValue(nested) {
				return true
			}
		}
		return false
	case []any:
		return len(typed) > 0
	default:
		return !reflect.ValueOf(value).IsZero()
	}
}

func channelShouldRequeue(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func conditionStatusForChannel(condition shared.OSOKConditionType) v1.ConditionStatus {
	if condition == shared.Failed {
		return v1.ConditionFalse
	}
	return v1.ConditionTrue
}

func channelStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
