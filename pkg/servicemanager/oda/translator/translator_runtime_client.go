/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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

const translatorOdaInstanceIDAnnotation = "oda.oracle.com/oda-instance-id"

type translatorOCIClient interface {
	CreateTranslator(context.Context, odasdk.CreateTranslatorRequest) (odasdk.CreateTranslatorResponse, error)
	GetTranslator(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error)
	ListTranslators(context.Context, odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error)
	UpdateTranslator(context.Context, odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error)
	DeleteTranslator(context.Context, odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error)
}

type translatorRuntimeClient struct {
	hooks TranslatorRuntimeHooks
	log   loggerutil.OSOKLogger
}

var _ TranslatorServiceClient = (*translatorRuntimeClient)(nil)

func init() {
	registerTranslatorRuntimeHooksMutator(func(manager *TranslatorServiceManager, hooks *TranslatorRuntimeHooks) {
		applyTranslatorRuntimeHooks(manager, hooks)
	})
}

func applyTranslatorRuntimeHooks(manager *TranslatorServiceManager, hooks *TranslatorRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newTranslatorRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(TranslatorServiceClient) TranslatorServiceClient {
		runtimeClient := &translatorRuntimeClient{
			hooks: *hooks,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func newTranslatorServiceClientWithOCIClient(log loggerutil.OSOKLogger, client translatorOCIClient) TranslatorServiceClient {
	hooks := newTranslatorRuntimeHooksWithOCIClient(client)
	applyTranslatorRuntimeHooks(&TranslatorServiceManager{Log: log}, &hooks)
	delegate := defaultTranslatorServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.Translator](
			buildTranslatorGeneratedRuntimeConfig(&TranslatorServiceManager{Log: log}, hooks),
		),
	}
	return wrapTranslatorGeneratedClient(hooks, delegate)
}

func newTranslatorRuntimeHooksWithOCIClient(client translatorOCIClient) TranslatorRuntimeHooks {
	return TranslatorRuntimeHooks{
		Create: runtimeOperationHooks[odasdk.CreateTranslatorRequest, odasdk.CreateTranslatorResponse]{
			Call: func(ctx context.Context, request odasdk.CreateTranslatorRequest) (odasdk.CreateTranslatorResponse, error) {
				return client.CreateTranslator(ctx, request)
			},
		},
		Get: runtimeOperationHooks[odasdk.GetTranslatorRequest, odasdk.GetTranslatorResponse]{
			Call: func(ctx context.Context, request odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
				return client.GetTranslator(ctx, request)
			},
		},
		List: runtimeOperationHooks[odasdk.ListTranslatorsRequest, odasdk.ListTranslatorsResponse]{
			Call: func(ctx context.Context, request odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
				return client.ListTranslators(ctx, request)
			},
		},
		Update: runtimeOperationHooks[odasdk.UpdateTranslatorRequest, odasdk.UpdateTranslatorResponse]{
			Call: func(ctx context.Context, request odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error) {
				return client.UpdateTranslator(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[odasdk.DeleteTranslatorRequest, odasdk.DeleteTranslatorResponse]{
			Call: func(ctx context.Context, request odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error) {
				return client.DeleteTranslator(ctx, request)
			},
		},
	}
}

func newTranslatorRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "translator",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(odasdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(odasdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(odasdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"type"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"baseUrl", "properties", "freeformTags", "definedTags"},
			ForceNew: []string{"type"},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Translator", Action: "CreateTranslator"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Translator", Action: "UpdateTranslator"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Translator", Action: "DeleteTranslator"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Translator", Action: "GetTranslator"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Translator", Action: "GetTranslator"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "resource-local", EntityType: "Translator", Action: "GetTranslator"}},
		},
		Unsupported: []generatedruntime.UnsupportedSemantic{
			{
				Category:      "authToken-update-drift",
				StopCondition: "authToken is write-only in OCI responses; OSOK sends it on create and alongside detected mutable updates, but cannot detect authToken-only drift until the API exposes a last-applied secret hash or secret reference status.",
			},
		},
	}
}

func (c *translatorRuntimeClient) CreateOrUpdate(ctx context.Context, resource *odav1beta1.Translator, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Translator resource is nil")
	}
	if c == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Translator runtime client is not configured")
	}
	if err := validateTranslatorDesiredSpec(resource); err != nil {
		return c.fail(resource, err)
	}

	odaInstanceID, err := translatorOdaInstanceID(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.create(ctx, resource, odaInstanceID)
	}

	if state := normalizedTranslatorLifecycle(current.LifecycleState); translatorLifecycleBlocksMutation(state) {
		return c.finishWithLifecycle(resource, current), nil
	}
	if err := validateTranslatorCreateOnlyDrift(resource, current); err != nil {
		return c.fail(resource, err)
	}

	updateDetails, updateNeeded := buildTranslatorUpdateDetails(resource, current)
	if updateNeeded {
		return c.update(ctx, resource, odaInstanceID, translatorIDFromSDK(current), updateDetails)
	}

	return c.finishWithLifecycle(resource, current), nil
}

func (c *translatorRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.Translator) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("Translator resource is nil")
	}
	if c == nil {
		return false, fmt.Errorf("Translator runtime client is not configured")
	}

	odaInstanceID, parentErr := translatorOdaInstanceID(resource)
	currentID := currentTranslatorID(resource)
	if parentErr != nil {
		if currentID == "" {
			c.markDeleted(resource, "Translator has no tracked OCI identity")
			return true, nil
		}
		c.markFailure(resource, parentErr)
		return false, parentErr
	}

	current, found, err := c.resolveCurrent(ctx, resource, odaInstanceID)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	if !found {
		c.markDeleted(resource, "Translator has no tracked OCI identity")
		return true, nil
	}

	currentID = translatorIDFromSDK(current)
	if currentID == "" {
		err := fmt.Errorf("Translator delete could not resolve OCI resource ID")
		return false, c.markFailure(resource, err)
	}

	switch normalizedTranslatorLifecycle(current.LifecycleState) {
	case string(odasdk.LifecycleStateDeleting):
		c.projectTranslatorStatus(resource, current)
		c.markCondition(resource, shared.Terminating, string(current.LifecycleState), "Translator delete is in progress", true)
		return false, nil
	case string(odasdk.LifecycleStateDeleted):
		c.markDeleted(resource, "OCI Translator deleted")
		return true, nil
	}

	response, err := c.hooks.Delete.Call(ctx, odasdk.DeleteTranslatorRequest{
		OdaInstanceId: common.String(odaInstanceID),
		TranslatorId:  common.String(currentID),
	})
	if err != nil {
		if isTranslatorReadNotFound(err) {
			c.markDeleted(resource, "OCI Translator is already deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	confirm, err := c.read(ctx, odaInstanceID, currentID)
	if err != nil {
		if isTranslatorReadNotFound(err) {
			c.markDeleted(resource, "OCI Translator deleted")
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	if normalizedTranslatorLifecycle(confirm.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
		c.markDeleted(resource, "OCI Translator deleted")
		return true, nil
	}

	c.projectTranslatorStatus(resource, confirm)
	c.markCondition(resource, shared.Terminating, string(confirm.LifecycleState), "Translator delete is in progress", true)
	return false, nil
}

func (c *translatorRuntimeClient) create(ctx context.Context, resource *odav1beta1.Translator, odaInstanceID string) (servicemanager.OSOKResponse, error) {
	response, err := c.hooks.Create.Call(ctx, odasdk.CreateTranslatorRequest{
		OdaInstanceId:           common.String(odaInstanceID),
		CreateTranslatorDetails: buildTranslatorCreateDetails(resource),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current := response.Translator
	currentID := translatorIDFromSDK(current)
	if currentID == "" {
		return c.fail(resource, fmt.Errorf("Translator create response did not include an OCI resource ID"))
	}

	followUp, err := c.read(ctx, odaInstanceID, currentID)
	if err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, followUp), nil
}

func (c *translatorRuntimeClient) update(
	ctx context.Context,
	resource *odav1beta1.Translator,
	odaInstanceID string,
	translatorID string,
	details odasdk.UpdateTranslatorDetails,
) (servicemanager.OSOKResponse, error) {
	if translatorID == "" {
		return c.fail(resource, fmt.Errorf("Translator update could not resolve OCI resource ID"))
	}
	response, err := c.hooks.Update.Call(ctx, odasdk.UpdateTranslatorRequest{
		OdaInstanceId:           common.String(odaInstanceID),
		TranslatorId:            common.String(translatorID),
		UpdateTranslatorDetails: details,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	followUp, err := c.read(ctx, odaInstanceID, translatorID)
	if err != nil {
		return c.fail(resource, err)
	}
	return c.finishWithLifecycle(resource, followUp), nil
}

func (c *translatorRuntimeClient) resolveCurrent(
	ctx context.Context,
	resource *odav1beta1.Translator,
	odaInstanceID string,
) (odasdk.Translator, bool, error) {
	if currentID := currentTranslatorID(resource); currentID != "" {
		current, err := c.read(ctx, odaInstanceID, currentID)
		if err != nil {
			if isTranslatorReadNotFound(err) {
				return odasdk.Translator{}, false, nil
			}
			return odasdk.Translator{}, false, err
		}
		return current, true, nil
	}
	return c.resolveByList(ctx, resource, odaInstanceID)
}

func (c *translatorRuntimeClient) resolveByList(
	ctx context.Context,
	resource *odav1beta1.Translator,
	odaInstanceID string,
) (odasdk.Translator, bool, error) {
	translatorType, err := normalizedTranslatorType(resource.Spec.Type)
	if err != nil {
		return odasdk.Translator{}, false, err
	}
	request := odasdk.ListTranslatorsRequest{
		OdaInstanceId: common.String(odaInstanceID),
		Type:          odasdk.ListTranslatorsTypeEnum(translatorType),
	}

	var matchedID string
	for {
		response, err := c.hooks.List.Call(ctx, request)
		if err != nil {
			return odasdk.Translator{}, false, err
		}

		for _, item := range response.Items {
			if !translatorSummaryMatches(resource, item) {
				continue
			}
			if normalizedTranslatorLifecycle(item.LifecycleState) == string(odasdk.LifecycleStateDeleted) {
				continue
			}
			itemID := stringValue(item.Id)
			if itemID == "" {
				continue
			}
			if matchedID != "" && matchedID != itemID {
				return odasdk.Translator{}, false, fmt.Errorf("multiple OCI Translators matched type %q", resource.Spec.Type)
			}
			matchedID = itemID
		}

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}
	if matchedID == "" {
		return odasdk.Translator{}, false, nil
	}

	current, err := c.read(ctx, odaInstanceID, matchedID)
	if err != nil {
		if isTranslatorReadNotFound(err) {
			return odasdk.Translator{}, false, nil
		}
		return odasdk.Translator{}, false, err
	}
	return current, true, nil
}

func (c *translatorRuntimeClient) read(ctx context.Context, odaInstanceID string, translatorID string) (odasdk.Translator, error) {
	response, err := c.hooks.Get.Call(ctx, odasdk.GetTranslatorRequest{
		OdaInstanceId: common.String(odaInstanceID),
		TranslatorId:  common.String(translatorID),
	})
	if err != nil {
		return odasdk.Translator{}, err
	}
	return response.Translator, nil
}

func (c *translatorRuntimeClient) finishWithLifecycle(resource *odav1beta1.Translator, current odasdk.Translator) servicemanager.OSOKResponse {
	c.projectTranslatorStatus(resource, current)

	state := normalizedTranslatorLifecycle(current.LifecycleState)
	message := translatorLifecycleMessage(current, "Translator lifecycle state "+state)
	switch state {
	case string(odasdk.LifecycleStateCreating):
		return c.markCondition(resource, shared.Provisioning, state, message, true)
	case string(odasdk.LifecycleStateUpdating):
		return c.markCondition(resource, shared.Updating, state, message, true)
	case string(odasdk.LifecycleStateDeleting):
		return c.markCondition(resource, shared.Terminating, state, message, true)
	case string(odasdk.LifecycleStateActive):
		return c.markCondition(resource, shared.Active, state, message, false)
	case string(odasdk.LifecycleStateDeleted),
		string(odasdk.LifecycleStateFailed),
		string(odasdk.LifecycleStateInactive):
		return c.markCondition(resource, shared.Failed, state, message, false)
	default:
		return c.markCondition(resource, shared.Failed, state, fmt.Sprintf("formal lifecycle state %q is not modeled: %s", state, message), false)
	}
}

func (c *translatorRuntimeClient) projectTranslatorStatus(resource *odav1beta1.Translator, current odasdk.Translator) {
	status := &resource.Status
	status.Id = stringValue(current.Id)
	status.Type = string(current.Type)
	status.Name = stringValue(current.Name)
	status.BaseUrl = stringValue(current.BaseUrl)
	status.LifecycleState = string(current.LifecycleState)
	status.TimeCreated = translatorSDKTimeString(current.TimeCreated)
	status.TimeUpdated = translatorSDKTimeString(current.TimeUpdated)
	status.Properties = cloneStringMap(current.Properties)
	status.FreeformTags = cloneStringMap(current.FreeformTags)
	status.DefinedTags = translatorStatusDefinedTags(current.DefinedTags)

	now := metav1.Now()
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
		if status.OsokStatus.CreatedAt == nil {
			status.OsokStatus.CreatedAt = &now
		}
	}
}

func (c *translatorRuntimeClient) fail(resource *odav1beta1.Translator, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}

func (c *translatorRuntimeClient) markFailure(resource *odav1beta1.Translator, err error) error {
	if err == nil {
		return nil
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		NormalizedClass: shared.OSOKAsyncClassFailed,
		Message:         err.Error(),
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

func (c *translatorRuntimeClient) markDeleted(resource *odav1beta1.Translator, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *translatorRuntimeClient) markCondition(
	resource *odav1beta1.Translator,
	condition shared.OSOKConditionType,
	rawState string,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = message
	status.Reason = string(condition)
	status.UpdatedAt = &now

	if condition == shared.Provisioning || condition == shared.Updating || (condition == shared.Terminating && shouldRequeue) {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           asyncPhaseForTranslatorCondition(condition),
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
	} else if condition == shared.Failed {
		status.Async.Current = &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			RawStatus:       rawState,
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
			UpdatedAt:       &now,
		}
	} else {
		status.Async.Current = nil
	}

	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: time.Minute,
	}
}

func validateTranslatorDesiredSpec(resource *odav1beta1.Translator) error {
	if _, err := normalizedTranslatorType(resource.Spec.Type); err != nil {
		return err
	}
	if strings.TrimSpace(resource.Spec.BaseUrl) == "" {
		return fmt.Errorf("Translator spec.baseUrl is required")
	}
	if strings.TrimSpace(resource.Spec.AuthToken) == "" {
		return fmt.Errorf("Translator spec.authToken is required")
	}
	return nil
}

func translatorOdaInstanceID(resource *odav1beta1.Translator) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("Translator resource is nil")
	}
	odaInstanceID := strings.TrimSpace(resource.Annotations[translatorOdaInstanceIDAnnotation])
	if odaInstanceID == "" {
		return "", fmt.Errorf("Translator requires metadata annotation %q with the parent ODA instance OCID", translatorOdaInstanceIDAnnotation)
	}
	return odaInstanceID, nil
}

func currentTranslatorID(resource *odav1beta1.Translator) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func buildTranslatorCreateDetails(resource *odav1beta1.Translator) odasdk.CreateTranslatorDetails {
	translatorType, _ := normalizedTranslatorType(resource.Spec.Type)
	details := odasdk.CreateTranslatorDetails{
		Type:      odasdk.TranslationServiceEnum(translatorType),
		BaseUrl:   common.String(strings.TrimSpace(resource.Spec.BaseUrl)),
		AuthToken: common.String(resource.Spec.AuthToken),
	}
	if resource.Spec.Properties != nil {
		details.Properties = maps.Clone(resource.Spec.Properties)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = translatorDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return details
}

func buildTranslatorUpdateDetails(resource *odav1beta1.Translator, current odasdk.Translator) (odasdk.UpdateTranslatorDetails, bool) {
	spec := resource.Spec
	details := odasdk.UpdateTranslatorDetails{}
	updateNeeded := false

	if strings.TrimSpace(spec.BaseUrl) != "" && strings.TrimSpace(spec.BaseUrl) != stringValue(current.BaseUrl) {
		details.BaseUrl = common.String(strings.TrimSpace(spec.BaseUrl))
		updateNeeded = true
	}
	if spec.Properties != nil && !maps.Equal(spec.Properties, current.Properties) {
		details.Properties = maps.Clone(spec.Properties)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := translatorDefinedTagsFromSpec(spec.DefinedTags)
		if !translatorJSONEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	if updateNeeded && strings.TrimSpace(spec.AuthToken) != "" {
		details.AuthToken = common.String(spec.AuthToken)
	}
	return details, updateNeeded
}

func validateTranslatorCreateOnlyDrift(resource *odav1beta1.Translator, current odasdk.Translator) error {
	desiredType, err := normalizedTranslatorType(resource.Spec.Type)
	if err != nil {
		return err
	}
	if current.Type != "" && desiredType != string(current.Type) {
		return fmt.Errorf("Translator create-only field drift detected for type; recreate the resource instead of updating immutable fields")
	}
	return nil
}

func translatorSummaryMatches(resource *odav1beta1.Translator, item odasdk.TranslatorSummary) bool {
	if resource == nil {
		return false
	}
	desiredType, err := normalizedTranslatorType(resource.Spec.Type)
	if err != nil {
		return false
	}
	return desiredType == string(item.Type)
}

func translatorLifecycleBlocksMutation(state string) bool {
	switch state {
	case string(odasdk.LifecycleStateCreating),
		string(odasdk.LifecycleStateUpdating),
		string(odasdk.LifecycleStateDeleting),
		string(odasdk.LifecycleStateDeleted),
		string(odasdk.LifecycleStateFailed),
		string(odasdk.LifecycleStateInactive):
		return true
	default:
		return false
	}
}

func isTranslatorReadNotFound(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func translatorIDFromSDK(current odasdk.Translator) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func translatorLifecycleMessage(current odasdk.Translator, fallback string) string {
	name := strings.TrimSpace(stringValue(current.Name))
	if name == "" {
		name = strings.TrimSpace(string(current.Type))
	}
	if name != "" {
		return fmt.Sprintf("Translator %s is %s", name, current.LifecycleState)
	}
	return fallback
}

func normalizedTranslatorLifecycle(state odasdk.LifecycleStateEnum) string {
	return strings.ToUpper(strings.TrimSpace(string(state)))
}

func normalizedTranslatorType(value string) (string, error) {
	translatorType := strings.ToUpper(strings.TrimSpace(value))
	if translatorType == "" {
		return "", fmt.Errorf("Translator spec.type is required")
	}
	if _, ok := odasdk.GetMappingTranslationServiceEnum(translatorType); !ok {
		return "", fmt.Errorf("Translator spec.type %q is not supported", value)
	}
	return translatorType, nil
}

func asyncPhaseForTranslatorCondition(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	switch condition {
	case shared.Provisioning:
		return shared.OSOKAsyncPhaseCreate
	case shared.Updating:
		return shared.OSOKAsyncPhaseUpdate
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return ""
	}
}

func translatorDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func translatorStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(input) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if len(values) == 0 {
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	if len(converted) == 0 {
		return nil
	}
	return converted
}

func translatorJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func translatorSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	return maps.Clone(input)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
