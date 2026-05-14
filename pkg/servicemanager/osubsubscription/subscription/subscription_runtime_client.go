/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	osubsubscriptionsdk "github.com/oracle/oci-go-sdk/v65/osubsubscription"
	osubsubscriptionv1beta1 "github.com/oracle/oci-service-operator/api/osubsubscription/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const subscriptionObserveRequeueDuration = time.Minute

type subscriptionOCIClient interface {
	ListSubscriptions(context.Context, osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error)
}

type subscriptionProjectedListBody struct {
	Items []subscriptionProjectedBody `json:"items,omitempty"`
}

type subscriptionProjectedBody struct {
	CompartmentId      string                                                  `json:"compartmentId,omitempty"`
	PlanNumber         string                                                  `json:"planNumber,omitempty"`
	SubscriptionId     string                                                  `json:"subscriptionId,omitempty"`
	BuyerEmail         string                                                  `json:"buyerEmail,omitempty"`
	SDKStatus          string                                                  `json:"sdkStatus,omitempty"`
	TimeStart          string                                                  `json:"timeStart,omitempty"`
	TimeEnd            string                                                  `json:"timeEnd,omitempty"`
	Currency           osubsubscriptionv1beta1.SubscriptionCurrency            `json:"currency,omitempty"`
	ServiceName        string                                                  `json:"serviceName,omitempty"`
	SubscribedServices []osubsubscriptionv1beta1.SubscriptionSubscribedService `json:"subscribedServices,omitempty"`
}

type subscriptionObservedBody struct {
	Status             string                                                  `json:"status,omitempty"`
	TimeStart          string                                                  `json:"timeStart,omitempty"`
	TimeEnd            string                                                  `json:"timeEnd,omitempty"`
	Currency           osubsubscriptionv1beta1.SubscriptionCurrency            `json:"currency,omitempty"`
	ServiceName        string                                                  `json:"serviceName,omitempty"`
	SubscribedServices []osubsubscriptionv1beta1.SubscriptionSubscribedService `json:"subscribedServices,omitempty"`
}

type subscriptionLifecycleClient struct {
	delegate SubscriptionServiceClient
	log      loggerutil.OSOKLogger
}

func init() {
	registerSubscriptionRuntimeHooksMutator(func(manager *SubscriptionServiceManager, hooks *SubscriptionRuntimeHooks) {
		applySubscriptionRuntimeHooks(manager, hooks)
	})
}

func applySubscriptionRuntimeHooks(
	manager *SubscriptionServiceManager,
	hooks *SubscriptionRuntimeHooks,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedSubscriptionRuntimeSemantics()
	if hooks.List.Call != nil {
		hooks.List.Call = listSubscriptionsAllPages(hooks.List.Call)
	}
	hooks.List.Fields = subscriptionReadFields()
	hooks.Read.List = subscriptionProjectedListReadOperation(hooks.List)
	hooks.StatusHooks.ApplyLifecycle = func(
		resource *osubsubscriptionv1beta1.Subscription,
		response any,
	) (servicemanager.OSOKResponse, error) {
		current, ok := subscriptionProjectedBodyFromAny(response)
		if !ok {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("unexpected Subscription lifecycle response type %T", response)
		}
		return applySubscriptionObservedLifecycle(resource, current, subscriptionLogger(manager))
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SubscriptionServiceClient) SubscriptionServiceClient {
		return subscriptionLifecycleClient{
			delegate: delegate,
			log:      subscriptionLogger(manager),
		}
	})
}

func subscriptionLogger(manager *SubscriptionServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func newSubscriptionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client subscriptionOCIClient,
) SubscriptionServiceClient {
	hooks := newSubscriptionRuntimeHooksWithOCIClient(client)
	applySubscriptionRuntimeHooks(nil, &hooks)
	delegate := defaultSubscriptionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osubsubscriptionv1beta1.Subscription](
			buildSubscriptionGeneratedRuntimeConfig(&SubscriptionServiceManager{Log: log}, hooks),
		),
	}
	return wrapSubscriptionGeneratedClient(hooks, delegate)
}

func newSubscriptionRuntimeHooksWithOCIClient(client subscriptionOCIClient) SubscriptionRuntimeHooks {
	return SubscriptionRuntimeHooks{
		Semantics:       reviewedSubscriptionRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*osubsubscriptionv1beta1.Subscription]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osubsubscriptionv1beta1.Subscription]{},
		StatusHooks:     generatedruntime.StatusHooks[*osubsubscriptionv1beta1.Subscription]{},
		ParityHooks:     generatedruntime.ParityHooks[*osubsubscriptionv1beta1.Subscription]{},
		Async:           generatedruntime.AsyncHooks[*osubsubscriptionv1beta1.Subscription]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osubsubscriptionv1beta1.Subscription]{},
		List: runtimeOperationHooks[osubsubscriptionsdk.ListSubscriptionsRequest, osubsubscriptionsdk.ListSubscriptionsResponse]{
			Fields: subscriptionReadFields(),
			Call: func(ctx context.Context, request osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
				if client == nil {
					return osubsubscriptionsdk.ListSubscriptionsResponse{}, fmt.Errorf("subscription OCI client is nil")
				}
				return client.ListSubscriptions(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SubscriptionServiceClient) SubscriptionServiceClient{},
	}
}

func reviewedSubscriptionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "osubsubscription",
		FormalSlug:    "subscription",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "none",
			Runtime:              "generatedruntime",
			FormalClassification: "none",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "none",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{},
			UpdatingStates:     []string{},
			ActiveStates:       []string{},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "best-effort",
			PendingStates:  []string{},
			TerminalStates: []string{"LOCAL_DELETE"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"buyerEmail", "compartmentId", "planNumber", "subscriptionId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{},
			ForceNew:      []string{"buyerEmail", "compartmentId", "isCommitInfoRequired", "planNumber", "sortBy", "sortOrder", "subscriptionId", "xOneGatewaySubscriptionId", "xOneOriginRegion"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{},
			Update: []generatedruntime.Hook{},
			Delete: []generatedruntime.Hook{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "none",
			Hooks:    []generatedruntime.Hook{},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "none",
			Hooks:    []generatedruntime.Hook{},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func (c subscriptionLifecycleClient) CreateOrUpdate(
	ctx context.Context,
	resource *osubsubscriptionv1beta1.Subscription,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || resource == nil || !response.IsSuccessful {
		return response, err
	}

	return applySubscriptionObservedLifecycle(resource, subscriptionProjectedBody{
		SDKStatus: strings.TrimSpace(resource.Status.Status),
	}, c.log)
}

func (c subscriptionLifecycleClient) Delete(
	_ context.Context,
	resource *osubsubscriptionv1beta1.Subscription,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("Subscription resource must not be nil")
	}

	markSubscriptionDeleted(resource, c.log)
	return true, nil
}

func subscriptionReadFields() []generatedruntime.RequestField {
	// generatedruntime request building skips fields marked as header, so the
	// request field metadata here uses explicit lookups and relies on the OCI SDK
	// struct tags to place the populated values into headers.
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "PlanNumber", RequestName: "planNumber", Contribution: "query"},
		{FieldName: "SubscriptionId", RequestName: "subscriptionId", Contribution: "query"},
		{FieldName: "BuyerEmail", RequestName: "buyerEmail", Contribution: "query"},
		{FieldName: "IsCommitInfoRequired", RequestName: "isCommitInfoRequired", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "XOneGatewaySubscriptionId", RequestName: "xOneGatewaySubscriptionId", Contribution: "query"},
		{FieldName: "XOneOriginRegion", RequestName: "xOneOriginRegion", Contribution: "query"},
	}
}

func subscriptionProjectedListReadOperation(
	list runtimeOperationHooks[osubsubscriptionsdk.ListSubscriptionsRequest, osubsubscriptionsdk.ListSubscriptionsResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &osubsubscriptionsdk.ListSubscriptionsRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), list.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			typed := normalizeSubscriptionListRequest(*request.(*osubsubscriptionsdk.ListSubscriptionsRequest))
			if err := validateSubscriptionListRequest(typed); err != nil {
				return nil, err
			}
			response, err := list.Call(ctx, typed)
			if err != nil {
				return nil, err
			}
			return projectSubscriptionListBody(response, typed)
		},
	}
}

func normalizeSubscriptionListRequest(
	request osubsubscriptionsdk.ListSubscriptionsRequest,
) osubsubscriptionsdk.ListSubscriptionsRequest {
	request.CompartmentId = trimmedOptionalString(request.CompartmentId)
	request.PlanNumber = trimmedOptionalString(request.PlanNumber)
	request.SubscriptionId = trimmedOptionalString(request.SubscriptionId)
	request.BuyerEmail = trimmedOptionalString(request.BuyerEmail)
	request.Page = trimmedOptionalString(request.Page)
	request.OpcRequestId = trimmedOptionalString(request.OpcRequestId)
	request.XOneGatewaySubscriptionId = trimmedOptionalString(request.XOneGatewaySubscriptionId)
	request.XOneOriginRegion = trimmedOptionalString(request.XOneOriginRegion)
	return request
}

func validateSubscriptionListRequest(
	request osubsubscriptionsdk.ListSubscriptionsRequest,
) error {
	if strings.TrimSpace(stringPtrValue(request.CompartmentId)) == "" {
		return fmt.Errorf("spec.compartmentId is required for OSubscription Subscription observe reads")
	}

	selectorCount := 0
	for _, value := range []string{
		stringPtrValue(request.PlanNumber),
		stringPtrValue(request.SubscriptionId),
		stringPtrValue(request.BuyerEmail),
	} {
		if strings.TrimSpace(value) != "" {
			selectorCount++
		}
	}
	if selectorCount != 1 {
		return fmt.Errorf("exactly one of spec.planNumber, spec.subscriptionId, or spec.buyerEmail must be set")
	}

	return nil
}

func projectSubscriptionListBody(
	response osubsubscriptionsdk.ListSubscriptionsResponse,
	request osubsubscriptionsdk.ListSubscriptionsRequest,
) (subscriptionProjectedListBody, error) {
	projected := subscriptionProjectedListBody{
		Items: make([]subscriptionProjectedBody, 0, len(response.Items)),
	}
	for _, summary := range response.Items {
		item, err := projectSubscriptionSummary(summary, request)
		if err != nil {
			return subscriptionProjectedListBody{}, err
		}
		projected.Items = append(projected.Items, item)
	}
	return projected, nil
}

func projectSubscriptionSummary(
	summary osubsubscriptionsdk.SubscriptionSummary,
	request osubsubscriptionsdk.ListSubscriptionsRequest,
) (subscriptionProjectedBody, error) {
	observed, err := convertThroughJSON[subscriptionObservedBody](summary)
	if err != nil {
		return subscriptionProjectedBody{}, err
	}
	return subscriptionProjectedBody{
		CompartmentId:      stringPtrValue(request.CompartmentId),
		PlanNumber:         stringPtrValue(request.PlanNumber),
		SubscriptionId:     stringPtrValue(request.SubscriptionId),
		BuyerEmail:         stringPtrValue(request.BuyerEmail),
		SDKStatus:          strings.TrimSpace(observed.Status),
		TimeStart:          observed.TimeStart,
		TimeEnd:            observed.TimeEnd,
		Currency:           observed.Currency,
		ServiceName:        observed.ServiceName,
		SubscribedServices: observed.SubscribedServices,
	}, nil
}

func subscriptionProjectedBodyFromAny(response any) (subscriptionProjectedBody, bool) {
	switch current := response.(type) {
	case subscriptionProjectedBody:
		return current, true
	case *subscriptionProjectedBody:
		if current == nil {
			return subscriptionProjectedBody{}, false
		}
		return *current, true
	default:
		return subscriptionProjectedBody{}, false
	}
}

func applySubscriptionObservedLifecycle(
	resource *osubsubscriptionv1beta1.Subscription,
	current subscriptionProjectedBody,
	log loggerutil.OSOKLogger,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("subscription resource is nil")
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Ocid = ""
	status.Async.Current = nil

	if current.SDKStatus == "" {
		message := "Observed OSubscription subscription without sdkStatus; retrying for a complete projection"
		status.Message = message
		status.Reason = string(shared.Updating)
		*status = dropSubscriptionStaleConditions(*status, shared.Updating)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Updating, corev1.ConditionTrue, "", message, log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    true,
			ShouldRequeue:   true,
			RequeueDuration: subscriptionObserveRequeueDuration,
		}, nil
	}

	message := fmt.Sprintf("Observed OSubscription subscription status %q", current.SDKStatus)
	status.Message = message
	status.Reason = string(shared.Active)
	*status = dropSubscriptionStaleConditions(*status, shared.Active)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, corev1.ConditionTrue, "", message, log)
	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func markSubscriptionDeleted(
	resource *osubsubscriptionv1beta1.Subscription,
	log loggerutil.OSOKLogger,
) {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)

	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Ocid = ""
	status.Message = subscriptionDeleteMessage(resource)
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, log)
}

func dropSubscriptionStaleConditions(
	status shared.OSOKStatus,
	current shared.OSOKConditionType,
) shared.OSOKStatus {
	if len(status.Conditions) == 0 {
		return status
	}

	conditions := status.Conditions[:0]
	for _, condition := range status.Conditions {
		switch {
		case condition.Type == shared.Failed:
			continue
		case current == shared.Active && condition.Type == shared.Updating:
			continue
		case current == shared.Updating && condition.Type == shared.Active:
			continue
		default:
			conditions = append(conditions, condition)
		}
	}
	status.Conditions = conditions
	return status
}

func subscriptionDeleteMessage(resource *osubsubscriptionv1beta1.Subscription) string {
	selector := firstNonEmptyTrim(
		resource.Status.ServiceName,
		resource.Spec.SubscriptionId,
		resource.Spec.PlanNumber,
		resource.Spec.BuyerEmail,
	)
	if selector == "" {
		selector = "Subscription"
	}
	return fmt.Sprintf("%s was released from Kubernetes control", selector)
}

func listSubscriptionsAllPages(
	call func(context.Context, osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error),
) func(context.Context, osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
	return func(ctx context.Context, request osubsubscriptionsdk.ListSubscriptionsRequest) (osubsubscriptionsdk.ListSubscriptionsResponse, error) {
		var combined osubsubscriptionsdk.ListSubscriptionsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return osubsubscriptionsdk.ListSubscriptionsResponse{}, err
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

func convertThroughJSON[T any](input any) (T, error) {
	var converted T
	if input == nil {
		return converted, nil
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return converted, fmt.Errorf("marshal subscription projection source: %w", err)
	}
	if err := json.Unmarshal(payload, &converted); err != nil {
		return converted, fmt.Errorf("unmarshal subscription projection target: %w", err)
	}
	return converted, nil
}

func trimmedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
