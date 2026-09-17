/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package organizationsubscription

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	osuborganizationsubscriptionsdk "github.com/oracle/oci-go-sdk/v65/osuborganizationsubscription"
	osuborganizationsubscriptionv1beta1 "github.com/oracle/oci-service-operator/api/osuborganizationsubscription/v1beta1"
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

const (
	OrganizationSubscriptionCompartmentIDAnnotation   = "osuborganizationsubscription.oracle.com/compartment-id"
	OrganizationSubscriptionSubscriptionIDAnnotation  = "osuborganizationsubscription.oracle.com/subscription-id"
	OrganizationSubscriptionSubscriptionIDsAnnotation = "osuborganizationsubscription.oracle.com/subscription-ids"
	OrganizationSubscriptionOriginRegionAnnotation    = "osuborganizationsubscription.oracle.com/origin-region"

	organizationSubscriptionLegacyCompartmentIDAnnotation  = "osuborganizationsubscription.oracle.com/compartmentId"
	organizationSubscriptionLegacySubscriptionIDAnnotation = "osuborganizationsubscription.oracle.com/subscriptionId"
	organizationSubscriptionListLimit                      = 50
)

type organizationSubscriptionOCIClient interface {
	ListOrganizationSubscriptions(
		context.Context,
		osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest,
	) (osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsResponse, error)
}

type organizationSubscriptionRuntimeClient struct {
	client  organizationSubscriptionOCIClient
	log     loggerutil.OSOKLogger
	initErr error
}

type organizationSubscriptionLookup struct {
	compartmentID  string
	subscriptionID string
	originRegion   string
}

func init() {
	registerOrganizationSubscriptionRuntimeHooksMutator(func(manager *OrganizationSubscriptionServiceManager, hooks *OrganizationSubscriptionRuntimeHooks) {
		client, err := newOrganizationSubscriptionOCIClient(manager)
		applyOrganizationSubscriptionRuntimeHooks(manager, hooks, client, err)
	})
}

func newOrganizationSubscriptionOCIClient(manager *OrganizationSubscriptionServiceManager) (organizationSubscriptionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("OrganizationSubscription service manager is nil")
	}
	client, err := osuborganizationsubscriptionsdk.NewOrganizationSubscriptionClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOrganizationSubscriptionRuntimeHooks(
	manager *OrganizationSubscriptionServiceManager,
	hooks *OrganizationSubscriptionRuntimeHooks,
	client organizationSubscriptionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = organizationSubscriptionRuntimeSemantics()
	hooks.List.Fields = organizationSubscriptionListFields()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(OrganizationSubscriptionServiceClient) OrganizationSubscriptionServiceClient {
		log := loggerutil.OSOKLogger{}
		if manager != nil {
			log = manager.Log
		}
		return &organizationSubscriptionRuntimeClient{
			client:  client,
			log:     log,
			initErr: initErr,
		}
	})
}

func newOrganizationSubscriptionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client organizationSubscriptionOCIClient,
) OrganizationSubscriptionServiceClient {
	return &organizationSubscriptionRuntimeClient{
		client: client,
		log:    log,
	}
}

func organizationSubscriptionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "osuborganizationsubscription",
		FormalSlug:        "organizationsubscription",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "unbind-only-read-only-resource",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "none",
			Runtime:              "generatedruntime",
			FormalClassification: "none",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "best-effort",
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew: []string{"id"},
		},
		Unsupported: []generatedruntime.UnsupportedSemantic{
			{
				Category:      "sdk-surface",
				StopCondition: "OCI Go SDK osuborganizationsubscription exposes ListOrganizationSubscriptions only; create/update/delete remain unsupported until the SDK/API publishes mutation methods",
			},
			{
				Category:      "crd-shape",
				StopCondition: "compartment and subscription ids are supplied through metadata annotations because the v1beta1 OrganizationSubscription spec has no identity fields",
			},
		},
	}
}

func organizationSubscriptionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths: []string{
				"metadata.annotations." + OrganizationSubscriptionCompartmentIDAnnotation,
				"metadata.annotations." + organizationSubscriptionLegacyCompartmentIDAnnotation,
			},
		},
		{
			FieldName:    "SubscriptionIds",
			RequestName:  "subscriptionIds",
			Contribution: "query",
			LookupPaths: []string{
				"metadata.annotations." + OrganizationSubscriptionSubscriptionIDAnnotation,
				"metadata.annotations." + OrganizationSubscriptionSubscriptionIDsAnnotation,
				"status.status.ocid",
			},
		},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func (c *organizationSubscriptionRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("OrganizationSubscription resource is nil")
	}
	if c.initErr != nil {
		return c.fail(resource, fmt.Errorf("initialize OrganizationSubscription OCI client: %w", c.initErr))
	}
	if c.client == nil {
		return c.fail(resource, fmt.Errorf("OrganizationSubscription OCI client is not configured"))
	}

	lookup, err := organizationSubscriptionLookupFromResource(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	summary, requestID, err := c.findOrganizationSubscription(ctx, lookup)
	if err != nil {
		return c.fail(resource, err)
	}
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, summary, map[string]string{"status": "sdkStatus"}); err != nil {
		return c.fail(resource, err)
	}
	return c.markActive(resource, summary, requestID), nil
}

func (c *organizationSubscriptionRuntimeClient) Delete(
	_ context.Context,
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("OrganizationSubscription resource is nil")
	}
	c.markDeleted(resource, "OCI OrganizationSubscription delete is not supported; removed Kubernetes binding only")
	return true, nil
}

func organizationSubscriptionLookupFromResource(
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
) (organizationSubscriptionLookup, error) {
	if resource == nil {
		return organizationSubscriptionLookup{}, fmt.Errorf("OrganizationSubscription resource is nil")
	}

	compartmentID := organizationSubscriptionAnnotation(
		resource,
		OrganizationSubscriptionCompartmentIDAnnotation,
		organizationSubscriptionLegacyCompartmentIDAnnotation,
	)
	if compartmentID == "" {
		return organizationSubscriptionLookup{}, fmt.Errorf("OrganizationSubscription requires metadata annotation %q with the compartment OCID for ListOrganizationSubscriptions", OrganizationSubscriptionCompartmentIDAnnotation)
	}

	subscriptionID, err := organizationSubscriptionID(resource)
	if err != nil {
		return organizationSubscriptionLookup{}, err
	}

	return organizationSubscriptionLookup{
		compartmentID:  compartmentID,
		subscriptionID: subscriptionID,
		originRegion:   organizationSubscriptionAnnotation(resource, OrganizationSubscriptionOriginRegionAnnotation),
	}, nil
}

func organizationSubscriptionID(resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription) (string, error) {
	desiredID, hasDesiredID, err := annotatedOrganizationSubscriptionID(resource)
	if err != nil {
		return "", err
	}
	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if hasDesiredID {
		if trackedID != "" && trackedID != desiredID {
			return "", fmt.Errorf("OrganizationSubscription tracked subscription id %q differs from annotation value %q; create a replacement resource instead", trackedID, desiredID)
		}
		return desiredID, nil
	}
	if trackedID == "" {
		return "", fmt.Errorf("OrganizationSubscription requires metadata annotation %q with exactly one subscription id", OrganizationSubscriptionSubscriptionIDAnnotation)
	}
	return trackedID, nil
}

func annotatedOrganizationSubscriptionID(
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
) (string, bool, error) {
	singular := organizationSubscriptionAnnotation(
		resource,
		OrganizationSubscriptionSubscriptionIDAnnotation,
		organizationSubscriptionLegacySubscriptionIDAnnotation,
	)
	plural := organizationSubscriptionAnnotation(resource, OrganizationSubscriptionSubscriptionIDsAnnotation)

	switch {
	case singular != "" && plural != "":
		parsedPlural, err := singleOrganizationSubscriptionID(plural, OrganizationSubscriptionSubscriptionIDsAnnotation)
		if err != nil {
			return "", false, err
		}
		if singular != parsedPlural {
			return "", false, fmt.Errorf("OrganizationSubscription annotations %q and %q disagree; use one subscription id", OrganizationSubscriptionSubscriptionIDAnnotation, OrganizationSubscriptionSubscriptionIDsAnnotation)
		}
		return singular, true, nil
	case singular != "":
		return singular, true, nil
	case plural != "":
		parsedPlural, err := singleOrganizationSubscriptionID(plural, OrganizationSubscriptionSubscriptionIDsAnnotation)
		if err != nil {
			return "", false, err
		}
		return parsedPlural, true, nil
	default:
		return "", false, nil
	}
}

func singleOrganizationSubscriptionID(raw string, annotation string) (string, error) {
	parts := strings.Split(raw, ",")
	var ids []string
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) != 1 {
		return "", fmt.Errorf("OrganizationSubscription annotation %q must contain exactly one subscription id, got %d", annotation, len(ids))
	}
	return ids[0], nil
}

func organizationSubscriptionAnnotation(
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
	keys ...string,
) string {
	for _, key := range keys {
		if resource == nil || resource.Annotations == nil {
			return ""
		}
		if value := strings.TrimSpace(resource.Annotations[key]); value != "" {
			return value
		}
	}
	return ""
}

func (c *organizationSubscriptionRuntimeClient) findOrganizationSubscription(
	ctx context.Context,
	lookup organizationSubscriptionLookup,
) (osuborganizationsubscriptionsdk.SubscriptionSummary, string, error) {
	seenPages := map[string]struct{}{}
	var nextPage *string
	var lastRequestID string

	for {
		response, err := c.client.ListOrganizationSubscriptions(ctx, organizationSubscriptionListRequest(lookup, nextPage))
		if err != nil {
			return osuborganizationsubscriptionsdk.SubscriptionSummary{}, "", normalizeOrganizationSubscriptionOCIError(err)
		}
		if requestID := organizationSubscriptionString(response.OpcRequestId); requestID != "" {
			lastRequestID = requestID
		}

		for _, item := range response.Items {
			if organizationSubscriptionString(item.Id) == lookup.subscriptionID {
				return item, lastRequestID, nil
			}
		}

		page := organizationSubscriptionString(response.OpcNextPage)
		if page == "" {
			break
		}
		if _, seen := seenPages[page]; seen {
			return osuborganizationsubscriptionsdk.SubscriptionSummary{}, lastRequestID, fmt.Errorf("OrganizationSubscription list pagination repeated page token %q", page)
		}
		seenPages[page] = struct{}{}
		nextPage = common.String(page)
	}

	return osuborganizationsubscriptionsdk.SubscriptionSummary{}, lastRequestID, fmt.Errorf("OrganizationSubscription subscription id %q was not found in compartment %q", lookup.subscriptionID, lookup.compartmentID)
}

func organizationSubscriptionListRequest(
	lookup organizationSubscriptionLookup,
	page *string,
) osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest {
	request := osuborganizationsubscriptionsdk.ListOrganizationSubscriptionsRequest{
		CompartmentId:   common.String(lookup.compartmentID),
		SubscriptionIds: common.String(lookup.subscriptionID),
		Limit:           common.Int(organizationSubscriptionListLimit),
		Page:            page,
	}
	if lookup.originRegion != "" {
		request.XOneOriginRegion = common.String(lookup.originRegion)
	}
	return request
}

func normalizeOrganizationSubscriptionOCIError(err error) error {
	if err == nil {
		return nil
	}
	var serviceErr interface {
		common.ServiceError
		error
	}
	if errors.As(err, &serviceErr) {
		if _, normalized := errorutil.OciErrorTypeResponse(serviceErr); normalized != nil {
			return normalized
		}
	}
	return err
}

func (c *organizationSubscriptionRuntimeClient) markActive(
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
	summary osuborganizationsubscriptionsdk.SubscriptionSummary,
	requestID string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	servicemanager.SetOpcRequestID(status, requestID)

	subscriptionID := organizationSubscriptionString(summary.Id)
	status.Ocid = shared.OCID(subscriptionID)

	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.DeletedAt = nil
	status.Reason = string(shared.Active)
	status.Message = organizationSubscriptionStatusMessage(summary)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, corev1.ConditionTrue, "", status.Message, c.log)

	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func organizationSubscriptionStatusMessage(summary osuborganizationsubscriptionsdk.SubscriptionSummary) string {
	subscriptionID := organizationSubscriptionString(summary.Id)
	serviceName := organizationSubscriptionString(summary.ServiceName)
	planStatus := organizationSubscriptionString(summary.Status)

	parts := []string{"OCI OrganizationSubscription"}
	if subscriptionID != "" {
		parts = append(parts, subscriptionID)
	}
	if serviceName != "" {
		parts = append(parts, serviceName)
	}
	if planStatus != "" {
		parts = append(parts, "status "+planStatus)
	}
	if len(parts) == 1 {
		return "OCI OrganizationSubscription was observed"
	}
	return strings.Join(parts, " ")
}

func organizationSubscriptionString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (c *organizationSubscriptionRuntimeClient) fail(
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *organizationSubscriptionRuntimeClient) markDeleted(
	resource *osuborganizationsubscriptionv1beta1.OrganizationSubscription,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Reason = string(shared.Terminating)
	status.Message = message
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, c.log)
}
