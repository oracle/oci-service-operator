/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	onssdk "github.com/oracle/oci-go-sdk/v65/ons"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	subscriptionDeletePendingState   = "DELETE_ACCEPTED"
	subscriptionDeletePendingMessage = "OCI subscription delete is in progress"

	subscriptionMetadataFingerprintKey = "osokMetadataSHA256="
)

type subscriptionOCIClient interface {
	CreateSubscription(context.Context, onssdk.CreateSubscriptionRequest) (onssdk.CreateSubscriptionResponse, error)
	GetSubscription(context.Context, onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error)
	ListSubscriptions(context.Context, onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error)
	UpdateSubscription(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error)
	DeleteSubscription(context.Context, onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error)
}

type subscriptionAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e subscriptionAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e subscriptionAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerSubscriptionRuntimeHooksMutator(func(_ *SubscriptionServiceManager, hooks *SubscriptionRuntimeHooks) {
		applySubscriptionRuntimeHooks(hooks)
	})
}

func applySubscriptionRuntimeHooks(hooks *SubscriptionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = subscriptionRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *onsv1beta1.Subscription,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildSubscriptionUpdateBody(resource, currentResponse)
	}
	hooks.List.Fields = subscriptionListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listSubscriptionsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateSubscriptionMetadataDrift
	hooks.DeleteHooks.HandleError = handleSubscriptionDeleteError
	hooks.DeleteHooks.ApplyOutcome = applySubscriptionDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markSubscriptionTerminating
	wrapSubscriptionDeleteConfirmation(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SubscriptionServiceClient) SubscriptionServiceClient {
		return subscriptionMetadataTrackingClient{delegate: delegate}
	})
}

type subscriptionMetadataTrackingClient struct {
	delegate SubscriptionServiceClient
}

func (c subscriptionMetadataTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *onsv1beta1.Subscription,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	recorded, hasRecorded := subscriptionRecordedMetadataFingerprint(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	switch {
	case hasRecorded && subscriptionHasTrackedIdentity(resource):
		setSubscriptionMetadataFingerprint(resource, recorded)
	case err == nil && response.IsSuccessful && subscriptionHasTrackedIdentity(resource):
		recordSubscriptionMetadataFingerprint(resource)
	}
	return response, err
}

func (c subscriptionMetadataTrackingClient) Delete(
	ctx context.Context,
	resource *onsv1beta1.Subscription,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

type subscriptionDeleteConfirmationClient struct {
	delegate        SubscriptionServiceClient
	getSubscription func(context.Context, onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error)
}

func (c subscriptionDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *onsv1beta1.Subscription,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c subscriptionDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *onsv1beta1.Subscription,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func subscriptionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "ons",
		FormalSlug:    "subscription",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(onssdk.SubscriptionLifecycleStatePending)},
			UpdatingStates:     []string{},
			ActiveStates:       []string{string(onssdk.SubscriptionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{},
			TerminalStates: []string{string(onssdk.SubscriptionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "topicId", "protocol", "endpoint", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"deliveryPolicy", "freeformTags", "definedTags"},
			Mutable:         []string{"deliveryPolicy", "freeformTags", "definedTags"},
			ForceNew:        []string{"topicId", "compartmentId", "protocol", "endpoint", "metadata"},
			ConflictsWith:   map[string][]string{},
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
			Strategy: "response",
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

func subscriptionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "TopicId", RequestName: "topicId", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func newSubscriptionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client subscriptionOCIClient,
) SubscriptionServiceClient {
	manager := &SubscriptionServiceManager{Log: log}
	hooks := newSubscriptionRuntimeHooksWithOCIClient(client)
	applySubscriptionRuntimeHooks(&hooks)
	delegate := defaultSubscriptionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*onsv1beta1.Subscription](
			buildSubscriptionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSubscriptionGeneratedClient(hooks, delegate)
}

func newSubscriptionRuntimeHooksWithOCIClient(client subscriptionOCIClient) SubscriptionRuntimeHooks {
	return SubscriptionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*onsv1beta1.Subscription]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*onsv1beta1.Subscription]{},
		StatusHooks:     generatedruntime.StatusHooks[*onsv1beta1.Subscription]{},
		ParityHooks:     generatedruntime.ParityHooks[*onsv1beta1.Subscription]{},
		Async:           generatedruntime.AsyncHooks[*onsv1beta1.Subscription]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*onsv1beta1.Subscription]{},
		Create: runtimeOperationHooks[onssdk.CreateSubscriptionRequest, onssdk.CreateSubscriptionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateSubscriptionDetails", RequestName: "CreateSubscriptionDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request onssdk.CreateSubscriptionRequest) (onssdk.CreateSubscriptionResponse, error) {
				if client == nil {
					return onssdk.CreateSubscriptionResponse{}, fmt.Errorf("subscription OCI client is nil")
				}
				return client.CreateSubscription(ctx, request)
			},
		},
		Get: runtimeOperationHooks[onssdk.GetSubscriptionRequest, onssdk.GetSubscriptionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SubscriptionId", RequestName: "subscriptionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
				if client == nil {
					return onssdk.GetSubscriptionResponse{}, fmt.Errorf("subscription OCI client is nil")
				}
				return client.GetSubscription(ctx, request)
			},
		},
		List: runtimeOperationHooks[onssdk.ListSubscriptionsRequest, onssdk.ListSubscriptionsResponse]{
			Fields: subscriptionListFields(),
			Call: func(ctx context.Context, request onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
				if client == nil {
					return onssdk.ListSubscriptionsResponse{}, fmt.Errorf("subscription OCI client is nil")
				}
				return client.ListSubscriptions(ctx, request)
			},
		},
		Update: runtimeOperationHooks[onssdk.UpdateSubscriptionRequest, onssdk.UpdateSubscriptionResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "SubscriptionId", RequestName: "subscriptionId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateSubscriptionDetails", RequestName: "UpdateSubscriptionDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
				if client == nil {
					return onssdk.UpdateSubscriptionResponse{}, fmt.Errorf("subscription OCI client is nil")
				}
				return client.UpdateSubscription(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[onssdk.DeleteSubscriptionRequest, onssdk.DeleteSubscriptionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SubscriptionId", RequestName: "subscriptionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error) {
				if client == nil {
					return onssdk.DeleteSubscriptionResponse{}, fmt.Errorf("subscription OCI client is nil")
				}
				return client.DeleteSubscription(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SubscriptionServiceClient) SubscriptionServiceClient{},
	}
}

func listSubscriptionsAllPages(
	call func(context.Context, onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error),
) func(context.Context, onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
	return func(ctx context.Context, request onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
		var combined onssdk.ListSubscriptionsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return onssdk.ListSubscriptionsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
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

func buildSubscriptionUpdateBody(resource *onsv1beta1.Subscription, currentResponse any) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("subscription resource is nil")
	}

	current := observedSubscriptionState(resource, currentResponse)
	update := onssdk.UpdateSubscriptionDetails{}
	updateNeeded := false

	if resource.Spec.FreeformTags != nil && !jsonEqual(resource.Spec.FreeformTags, current.freeformTags) {
		update.FreeformTags = cloneStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}

	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := subscriptionDefinedTags(resource.Spec.DefinedTags)
		if !jsonEqual(desiredDefinedTags, current.definedTags) {
			update.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}

	desiredDeliveryPolicy, hasDeliveryPolicy, err := subscriptionSDKDeliveryPolicy(resource.Spec.DeliveryPolicy)
	if err != nil {
		return nil, false, err
	}
	if hasDeliveryPolicy && !jsonEqual(desiredDeliveryPolicy, current.deliveryPolicy) {
		update.DeliveryPolicy = desiredDeliveryPolicy
		updateNeeded = true
	}

	return update, updateNeeded, nil
}

func validateSubscriptionMetadataDrift(resource *onsv1beta1.Subscription, _ any) error {
	if resource == nil {
		return fmt.Errorf("subscription resource is nil")
	}

	recorded, ok := subscriptionRecordedMetadataFingerprint(resource)
	if !ok {
		if subscriptionHasTrackedIdentity(resource) && resource.Status.OsokStatus.CreatedAt != nil && resource.Spec.Metadata != "" {
			return fmt.Errorf("subscription create-only metadata cannot be validated because the original metadata fingerprint is missing; recreate the resource instead of changing metadata")
		}
		return nil
	}
	if desired := subscriptionMetadataFingerprint(resource.Spec.Metadata); desired != recorded {
		return fmt.Errorf("subscription formal semantics require replacement when metadata changes")
	}
	return nil
}

func wrapSubscriptionDeleteConfirmation(hooks *SubscriptionRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getSubscription := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SubscriptionServiceClient) SubscriptionServiceClient {
		return subscriptionDeleteConfirmationClient{
			delegate:        delegate,
			getSubscription: getSubscription,
		}
	})
}

func (c subscriptionDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *onsv1beta1.Subscription,
) error {
	if c.getSubscription == nil || resource == nil {
		return nil
	}
	subscriptionID := trackedSubscriptionID(resource)
	if subscriptionID == "" {
		return nil
	}
	_, err := c.getSubscription(ctx, onssdk.GetSubscriptionRequest{SubscriptionId: common.String(subscriptionID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("subscription delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func trackedSubscriptionID(resource *onsv1beta1.Subscription) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func recordSubscriptionMetadataFingerprint(resource *onsv1beta1.Subscription) {
	if resource == nil {
		return
	}
	setSubscriptionMetadataFingerprint(resource, subscriptionMetadataFingerprint(resource.Spec.Metadata))
}

func setSubscriptionMetadataFingerprint(resource *onsv1beta1.Subscription, fingerprint string) {
	if resource == nil {
		return
	}
	base := stripSubscriptionMetadataFingerprint(resource.Status.OsokStatus.Message)
	marker := subscriptionMetadataFingerprintKey + fingerprint
	if base == "" {
		resource.Status.OsokStatus.Message = marker
		return
	}
	resource.Status.OsokStatus.Message = base + "; " + marker
}

func subscriptionHasTrackedIdentity(resource *onsv1beta1.Subscription) bool {
	if resource == nil {
		return false
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" ||
		strings.TrimSpace(resource.Status.Id) != ""
}

func subscriptionMetadataFingerprint(metadata string) string {
	sum := sha256.Sum256([]byte(metadata))
	return hex.EncodeToString(sum[:])
}

func subscriptionRecordedMetadataFingerprint(resource *onsv1beta1.Subscription) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, subscriptionMetadataFingerprintKey)
	if index < 0 {
		return "", false
	}
	start := index + len(subscriptionMetadataFingerprintKey)
	end := start
	for end < len(raw) && isHexDigit(raw[end]) {
		end++
	}
	fingerprint := raw[start:end]
	if len(fingerprint) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", false
	}
	return fingerprint, true
}

func stripSubscriptionMetadataFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, subscriptionMetadataFingerprintKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(subscriptionMetadataFingerprintKey)
	end := start
	for end < len(raw) && isHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func isHexDigit(value byte) bool {
	return ('0' <= value && value <= '9') ||
		('a' <= value && value <= 'f') ||
		('A' <= value && value <= 'F')
}

type subscriptionObservedState struct {
	freeformTags   map[string]string
	definedTags    map[string]map[string]interface{}
	deliveryPolicy *onssdk.DeliveryPolicy
}

func observedSubscriptionState(resource *onsv1beta1.Subscription, response any) subscriptionObservedState {
	state := subscriptionObservedState{}
	if resource != nil {
		state.freeformTags = cloneStringMap(resource.Status.FreeformTags)
		state.definedTags = subscriptionDefinedTags(resource.Status.DefinedTags)
		if policy, ok, _ := subscriptionSDKDeliveryPolicy(resource.Status.DeliveryPolicy); ok {
			state.deliveryPolicy = policy
		} else if parsed := subscriptionDeliveryPolicyFromJSON(resource.Status.DeliverPolicy); parsed != nil {
			state.deliveryPolicy = parsed
		}
	}
	state.applyResponse(response)
	return state
}

func (s *subscriptionObservedState) applyResponse(response any) {
	if subscription, ok := subscriptionFromResponse(response); ok {
		s.applySubscription(subscription)
		return
	}
	if summary, ok := subscriptionSummaryFromResponse(response); ok {
		s.applySubscriptionSummary(summary)
		return
	}
	if details, ok := subscriptionUpdateDetailsFromResponse(response); ok {
		s.applyUpdateDetails(details)
	}
}

func subscriptionFromResponse(response any) (onssdk.Subscription, bool) {
	switch concrete := response.(type) {
	case onssdk.GetSubscriptionResponse:
		return concrete.Subscription, true
	case *onssdk.GetSubscriptionResponse:
		if concrete != nil {
			return concrete.Subscription, true
		}
	case onssdk.CreateSubscriptionResponse:
		return concrete.Subscription, true
	case *onssdk.CreateSubscriptionResponse:
		if concrete != nil {
			return concrete.Subscription, true
		}
	case onssdk.Subscription:
		return concrete, true
	case *onssdk.Subscription:
		if concrete != nil {
			return *concrete, true
		}
	}
	return onssdk.Subscription{}, false
}

func subscriptionSummaryFromResponse(response any) (onssdk.SubscriptionSummary, bool) {
	switch concrete := response.(type) {
	case onssdk.SubscriptionSummary:
		return concrete, true
	case *onssdk.SubscriptionSummary:
		if concrete != nil {
			return *concrete, true
		}
	}
	return onssdk.SubscriptionSummary{}, false
}

func subscriptionUpdateDetailsFromResponse(response any) (onssdk.UpdateSubscriptionDetails, bool) {
	switch concrete := response.(type) {
	case onssdk.UpdateSubscriptionResponse:
		return concrete.UpdateSubscriptionDetails, true
	case *onssdk.UpdateSubscriptionResponse:
		if concrete != nil {
			return concrete.UpdateSubscriptionDetails, true
		}
	case onssdk.UpdateSubscriptionDetails:
		return concrete, true
	case *onssdk.UpdateSubscriptionDetails:
		if concrete != nil {
			return *concrete, true
		}
	}
	return onssdk.UpdateSubscriptionDetails{}, false
}

func (s *subscriptionObservedState) applySubscription(subscription onssdk.Subscription) {
	s.freeformTags = cloneStringMap(subscription.FreeformTags)
	s.definedTags = cloneInterfaceTags(subscription.DefinedTags)
	if policy := subscriptionDeliveryPolicyFromJSON(stringValue(subscription.DeliverPolicy)); policy != nil {
		s.deliveryPolicy = policy
	}
}

func (s *subscriptionObservedState) applySubscriptionSummary(summary onssdk.SubscriptionSummary) {
	s.freeformTags = cloneStringMap(summary.FreeformTags)
	s.definedTags = cloneInterfaceTags(summary.DefinedTags)
	if summary.DeliveryPolicy != nil {
		policy := *summary.DeliveryPolicy
		s.deliveryPolicy = &policy
	}
}

func (s *subscriptionObservedState) applyUpdateDetails(details onssdk.UpdateSubscriptionDetails) {
	if details.FreeformTags != nil {
		s.freeformTags = cloneStringMap(details.FreeformTags)
	}
	if details.DefinedTags != nil {
		s.definedTags = cloneInterfaceTags(details.DefinedTags)
	}
	if details.DeliveryPolicy != nil {
		policy := *details.DeliveryPolicy
		s.deliveryPolicy = &policy
	}
}

func subscriptionDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		convertedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			convertedValues[key] = value
		}
		converted[namespace] = convertedValues
	}
	return converted
}

func cloneInterfaceTags(tags map[string]map[string]interface{}) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func subscriptionSDKDeliveryPolicy(
	policy onsv1beta1.SubscriptionDeliveryPolicy,
) (*onssdk.DeliveryPolicy, bool, error) {
	backoff := policy.BackoffRetryPolicy
	if backoff.MaxRetryDuration == 0 && strings.TrimSpace(backoff.PolicyType) == "" {
		return nil, false, nil
	}
	if strings.TrimSpace(backoff.PolicyType) == "" {
		return nil, false, fmt.Errorf("deliveryPolicy.backoffRetryPolicy.policyType is required when deliveryPolicy is set")
	}
	return &onssdk.DeliveryPolicy{
		BackoffRetryPolicy: &onssdk.BackoffRetryPolicy{
			MaxRetryDuration: common.Int(backoff.MaxRetryDuration),
			PolicyType:       onssdk.BackoffRetryPolicyPolicyTypeEnum(strings.TrimSpace(backoff.PolicyType)),
		},
	}, true, nil
}

func subscriptionDeliveryPolicyFromJSON(raw string) *onssdk.DeliveryPolicy {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var policy onssdk.DeliveryPolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return nil
	}
	return &policy
}

func handleSubscriptionDeleteError(resource *onsv1beta1.Subscription, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return subscriptionAmbiguousNotFoundError{
		message:      "subscription delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func applySubscriptionDeleteOutcome(
	resource *onsv1beta1.Subscription,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if response == nil || subscriptionLifecycleState(response) == string(onssdk.SubscriptionLifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !subscriptionDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest ||
		stage == generatedruntime.DeleteConfirmStageAlreadyPending {
		markSubscriptionTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func subscriptionDeleteAlreadyPending(resource *onsv1beta1.Subscription) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markSubscriptionTerminating(resource *onsv1beta1.Subscription, response any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = subscriptionDeletePendingMessage
	status.Reason = string(shared.Terminating)
	rawStatus := subscriptionLifecycleState(response)
	if rawStatus == "" {
		rawStatus = subscriptionDeletePendingState
	}
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawStatus,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         subscriptionDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		subscriptionDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func subscriptionLifecycleState(response any) string {
	if subscription, ok := subscriptionFromResponse(response); ok {
		return normalizeLifecycleState(string(subscription.LifecycleState))
	}
	if summary, ok := subscriptionSummaryFromResponse(response); ok {
		return normalizeLifecycleState(string(summary.LifecycleState))
	}
	return ""
}

func normalizeLifecycleState(state string) string {
	return strings.ToUpper(strings.TrimSpace(state))
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func jsonEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprintf("%#v", left) == fmt.Sprintf("%#v", right)
	}
	return string(leftPayload) == string(rightPayload)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
