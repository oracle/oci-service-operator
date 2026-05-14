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

	"github.com/oracle/oci-go-sdk/v65/common"
	onesubscriptionsdk "github.com/oracle/oci-go-sdk/v65/onesubscription"
	onesubscriptionv1beta1 "github.com/oracle/oci-service-operator/api/onesubscription/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const subscriptionLocalDeleteMessage = "OneSubscription subscription observation released from Kubernetes control; OCI subscription unchanged"

type subscriptionOCIClient interface {
	ListSubscriptions(context.Context, onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error)
}

type subscriptionListCall func(context.Context, onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error)

type subscriptionRuntimeClient struct {
	delegate SubscriptionServiceClient
	log      loggerutil.OSOKLogger
}

type projectedSubscriptionResponse struct {
	Body         projectedSubscriptionBody `presentIn:"body"`
	OpcRequestId *string                   `presentIn:"header" name:"opc-request-id"`
}

type projectedSubscriptionBody struct {
	Status             string                                                 `json:"sdkStatus"`
	TimeStart          string                                                 `json:"timeStart"`
	TimeEnd            string                                                 `json:"timeEnd"`
	Currency           onesubscriptionv1beta1.SubscriptionCurrency            `json:"currency"`
	ServiceName        string                                                 `json:"serviceName"`
	HoldReason         string                                                 `json:"holdReason"`
	TimeHoldReleaseEta string                                                 `json:"timeHoldReleaseEta"`
	SubscribedServices []onesubscriptionv1beta1.SubscriptionSubscribedService `json:"subscribedServices"`
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

	hooks.List.Fields = subscriptionQueryFields()
	hooks.Read.Get = subscriptionQueryReadOperation(hooks.List.Call)
	hooks.Read.List = nil
	hooks.StatusHooks.ProjectStatus = projectSubscriptionStatus
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SubscriptionServiceClient) SubscriptionServiceClient {
		return subscriptionRuntimeClient{
			delegate: delegate,
			log:      subscriptionLogger(manager),
		}
	})
}

func newSubscriptionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client subscriptionOCIClient,
) SubscriptionServiceClient {
	hooks := newSubscriptionRuntimeHooksWithOCIClient(client)
	applySubscriptionRuntimeHooks(nil, &hooks)
	delegate := defaultSubscriptionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*onesubscriptionv1beta1.Subscription](
			buildSubscriptionGeneratedRuntimeConfig(&SubscriptionServiceManager{Log: log}, hooks),
		),
	}
	return wrapSubscriptionGeneratedClient(hooks, delegate)
}

func newSubscriptionRuntimeHooksWithOCIClient(client subscriptionOCIClient) SubscriptionRuntimeHooks {
	return SubscriptionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*onesubscriptionv1beta1.Subscription]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*onesubscriptionv1beta1.Subscription]{},
		StatusHooks:     generatedruntime.StatusHooks[*onesubscriptionv1beta1.Subscription]{},
		ParityHooks:     generatedruntime.ParityHooks[*onesubscriptionv1beta1.Subscription]{},
		Async:           generatedruntime.AsyncHooks[*onesubscriptionv1beta1.Subscription]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*onesubscriptionv1beta1.Subscription]{},
		List: runtimeOperationHooks[onesubscriptionsdk.ListSubscriptionsRequest, onesubscriptionsdk.ListSubscriptionsResponse]{
			Fields: subscriptionQueryFields(),
			Call: func(ctx context.Context, request onesubscriptionsdk.ListSubscriptionsRequest) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
				if client == nil {
					return onesubscriptionsdk.ListSubscriptionsResponse{}, fmt.Errorf("Subscription OCI client is not configured")
				}
				return client.ListSubscriptions(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SubscriptionServiceClient) SubscriptionServiceClient{},
	}
}

func subscriptionLogger(manager *SubscriptionServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func subscriptionQueryFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "PlanNumber",
			RequestName:  "planNumber",
			Contribution: "query",
			LookupPaths:  []string{"spec.planNumber", "planNumber"},
		},
		{
			FieldName:    "SubscriptionId",
			RequestName:  "subscriptionId",
			Contribution: "query",
			LookupPaths:  []string{"spec.subscriptionId", "subscriptionId"},
		},
		{
			FieldName:    "BuyerEmail",
			RequestName:  "buyerEmail",
			Contribution: "query",
			LookupPaths:  []string{"spec.buyerEmail", "buyerEmail"},
		},
		{
			FieldName:    "IsCommitInfoRequired",
			RequestName:  "isCommitInfoRequired",
			Contribution: "query",
			LookupPaths:  []string{"spec.isCommitInfoRequired", "isCommitInfoRequired"},
		},
	}
}

func subscriptionQueryReadOperation(call subscriptionListCall) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &onesubscriptionsdk.ListSubscriptionsRequest{} },
		Fields:     subscriptionQueryFields(),
		Call: func(ctx context.Context, request any) (any, error) {
			return querySingleSubscription(ctx, call, *request.(*onesubscriptionsdk.ListSubscriptionsRequest))
		},
	}
}

func querySingleSubscription(
	ctx context.Context,
	call subscriptionListCall,
	request onesubscriptionsdk.ListSubscriptionsRequest,
) (projectedSubscriptionResponse, error) {
	if err := validateSubscriptionQueryRequest(request); err != nil {
		return projectedSubscriptionResponse{}, err
	}

	response, err := listSubscriptionsAllPages(ctx, call, request)
	if err != nil {
		return projectedSubscriptionResponse{}, err
	}

	switch len(response.Items) {
	case 0:
		return projectedSubscriptionResponse{}, fmt.Errorf("Subscription query returned no matches; provide compartmentId with a filter that resolves to exactly one summary")
	case 1:
		return projectSubscriptionResponse(response.Items[0], response.OpcRequestId)
	default:
		return projectedSubscriptionResponse{}, fmt.Errorf("Subscription query returned %d matches; this observe-only resource requires exactly one summary", len(response.Items))
	}
}

func validateSubscriptionQueryRequest(request onesubscriptionsdk.ListSubscriptionsRequest) error {
	if strings.TrimSpace(stringPtrValue(request.CompartmentId)) == "" {
		return fmt.Errorf("Subscription query requires compartmentId")
	}

	filterCount := 0
	for _, candidate := range []string{
		stringPtrValue(request.PlanNumber),
		stringPtrValue(request.SubscriptionId),
		stringPtrValue(request.BuyerEmail),
	} {
		if strings.TrimSpace(candidate) != "" {
			filterCount++
		}
	}
	if filterCount != 1 {
		return fmt.Errorf("Subscription query requires exactly one of planNumber, subscriptionId, or buyerEmail")
	}
	return nil
}

func listSubscriptionsAllPages(
	ctx context.Context,
	call subscriptionListCall,
	request onesubscriptionsdk.ListSubscriptionsRequest,
) (onesubscriptionsdk.ListSubscriptionsResponse, error) {
	if call == nil {
		return onesubscriptionsdk.ListSubscriptionsResponse{}, fmt.Errorf("Subscription list call is not configured")
	}

	var combined onesubscriptionsdk.ListSubscriptionsResponse
	seenPages := map[string]struct{}{}
	for {
		pageToken := strings.TrimSpace(stringPtrValue(request.Page))
		if _, seen := seenPages[pageToken]; seen {
			return onesubscriptionsdk.ListSubscriptionsResponse{}, fmt.Errorf("Subscription list pagination repeated page %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return onesubscriptionsdk.ListSubscriptionsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := strings.TrimSpace(stringPtrValue(response.OpcNextPage))
		if nextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}

		combined.OpcNextPage = common.String(nextPage)
		request.Page = common.String(nextPage)
	}
}

func projectSubscriptionResponse(
	item onesubscriptionsdk.SubscriptionSummary,
	opcRequestID *string,
) (projectedSubscriptionResponse, error) {
	currency, err := convertThroughJSON[onesubscriptionv1beta1.SubscriptionCurrency](item.Currency)
	if err != nil {
		return projectedSubscriptionResponse{}, fmt.Errorf("project subscription currency: %w", err)
	}
	subscribedServices, err := convertThroughJSON[[]onesubscriptionv1beta1.SubscriptionSubscribedService](item.SubscribedServices)
	if err != nil {
		return projectedSubscriptionResponse{}, fmt.Errorf("project subscribed services: %w", err)
	}

	return projectedSubscriptionResponse{
		Body: projectedSubscriptionBody{
			Status:             stringPtrValue(item.Status),
			TimeStart:          sdkTimeString(item.TimeStart),
			TimeEnd:            sdkTimeString(item.TimeEnd),
			Currency:           currency,
			ServiceName:        stringPtrValue(item.ServiceName),
			HoldReason:         stringPtrValue(item.HoldReason),
			TimeHoldReleaseEta: sdkTimeString(item.TimeHoldReleaseEta),
			SubscribedServices: subscribedServices,
		},
		OpcRequestId: opcRequestID,
	}, nil
}

func projectSubscriptionStatus(resource *onesubscriptionv1beta1.Subscription, response any) error {
	if resource == nil {
		return fmt.Errorf("Subscription resource is nil")
	}

	body, opcRequestID, err := projectedSubscriptionBodyFromResponse(response)
	if err != nil {
		return err
	}

	osokStatus := resource.Status.OsokStatus
	if opcRequestID != "" {
		osokStatus.OpcRequestID = opcRequestID
	}
	resource.Status = onesubscriptionv1beta1.SubscriptionStatus{
		OsokStatus:         osokStatus,
		Status:             body.Status,
		TimeStart:          body.TimeStart,
		TimeEnd:            body.TimeEnd,
		Currency:           body.Currency,
		ServiceName:        body.ServiceName,
		HoldReason:         body.HoldReason,
		TimeHoldReleaseEta: body.TimeHoldReleaseEta,
		SubscribedServices: append([]onesubscriptionv1beta1.SubscriptionSubscribedService(nil), body.SubscribedServices...),
	}
	return nil
}

func projectedSubscriptionBodyFromResponse(response any) (projectedSubscriptionBody, string, error) {
	switch typed := response.(type) {
	case projectedSubscriptionResponse:
		return typed.Body, stringPtrValue(typed.OpcRequestId), nil
	case *projectedSubscriptionResponse:
		if typed == nil {
			return projectedSubscriptionBody{}, "", fmt.Errorf("projected Subscription response is nil")
		}
		return typed.Body, stringPtrValue(typed.OpcRequestId), nil
	case projectedSubscriptionBody:
		return typed, "", nil
	case *projectedSubscriptionBody:
		if typed == nil {
			return projectedSubscriptionBody{}, "", fmt.Errorf("projected Subscription body is nil")
		}
		return *typed, "", nil
	default:
		return projectedSubscriptionBody{}, "", fmt.Errorf("unexpected Subscription projection response %T", response)
	}
}

func (c subscriptionRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *onesubscriptionv1beta1.Subscription,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Subscription generated runtime delegate is not configured")
	}
	if err := validateSubscriptionResource(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || resource == nil {
		return response, err
	}

	normalizeSubscriptionObservation(resource, c.log, &response)
	return response, nil
}

func validateSubscriptionResource(resource *onesubscriptionv1beta1.Subscription) error {
	if resource == nil {
		return fmt.Errorf("Subscription resource is nil")
	}

	request := onesubscriptionsdk.ListSubscriptionsRequest{
		CompartmentId:        common.String(resource.Spec.CompartmentId),
		PlanNumber:           optionalString(resource.Spec.PlanNumber),
		SubscriptionId:       optionalString(resource.Spec.SubscriptionId),
		BuyerEmail:           optionalString(resource.Spec.BuyerEmail),
		IsCommitInfoRequired: optionalBool(resource.Spec.IsCommitInfoRequired),
	}
	return validateSubscriptionQueryRequest(request)
}

func (c subscriptionRuntimeClient) Delete(
	_ context.Context,
	resource *onesubscriptionv1beta1.Subscription,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("Subscription resource is nil")
	}

	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Ocid = ""
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = subscriptionLocalDeleteMessage
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		subscriptionLocalDeleteMessage,
		c.log,
	)
	return true, nil
}

func normalizeSubscriptionObservation(
	resource *onesubscriptionv1beta1.Subscription,
	log loggerutil.OSOKLogger,
	response *servicemanager.OSOKResponse,
) {
	status := &resource.Status.OsokStatus
	status.Ocid = ""
	servicemanager.ClearAsyncOperation(status)

	condition, shouldRequeue := classifySubscriptionCondition(resource.Status.Status)
	message := subscriptionObservationMessage(resource.Status.ServiceName, resource.Status.Status)
	now := metav1.Now()
	status.Message = message
	status.Reason = string(condition)
	status.UpdatedAt = &now
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		condition,
		conditionStatusForCondition(condition),
		"",
		message,
		log,
	)

	response.IsSuccessful = condition != shared.Failed
	response.ShouldRequeue = shouldRequeue
	if shouldRequeue {
		if response.RequeueDuration == 0 {
			response.RequeueDuration = time.Minute
		}
	} else {
		response.RequeueDuration = 0
	}
}

func classifySubscriptionCondition(rawStatus string) (shared.OSOKConditionType, bool) {
	status := strings.ToUpper(strings.TrimSpace(rawStatus))
	switch {
	case status == "":
		return shared.Active, false
	case strings.Contains(status, "FAIL"), strings.Contains(status, "ERROR"):
		return shared.Failed, false
	case strings.Contains(status, "UPDAT"), strings.Contains(status, "MODIFY"), strings.Contains(status, "PATCH"):
		return shared.Updating, true
	case strings.Contains(status, "CREATE"),
		strings.Contains(status, "PROVISION"),
		strings.Contains(status, "PENDING"),
		strings.Contains(status, "IN_PROGRESS"),
		strings.Contains(status, "ACCEPT"),
		strings.Contains(status, "START"),
		strings.Contains(status, "ACTIVAT"):
		return shared.Provisioning, true
	default:
		return shared.Active, false
	}
}

func subscriptionObservationMessage(serviceName string, rawStatus string) string {
	message := strings.TrimSpace(serviceName)
	if message == "" {
		message = "Observed OneSubscription subscription"
	}

	status := strings.TrimSpace(rawStatus)
	if status == "" {
		return message
	}
	return fmt.Sprintf("%s (%s)", message, status)
}

func conditionStatusForCondition(condition shared.OSOKConditionType) corev1.ConditionStatus {
	if condition == shared.Failed {
		return corev1.ConditionFalse
	}
	return corev1.ConditionTrue
}

func convertThroughJSON[T any](input any) (T, error) {
	var output T
	if input == nil {
		return output, nil
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return output, err
	}
	if err := json.Unmarshal(payload, &output); err != nil {
		return output, err
	}
	return output, nil
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(strings.TrimSpace(value))
}

func optionalBool(value bool) *bool {
	if !value {
		return nil
	}
	return common.Bool(value)
}
