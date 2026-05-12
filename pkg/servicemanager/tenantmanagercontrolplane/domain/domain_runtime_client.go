/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package domain

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	tenantmanagercontrolplanesdk "github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane"
	tenantmanagercontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/tenantmanagercontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const domainDeletePendingMessage = "OCI Domain delete is in progress"

var domainWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(tenantmanagercontrolplanesdk.OperationStatusAccepted),
		string(tenantmanagercontrolplanesdk.OperationStatusInProgress),
		string(tenantmanagercontrolplanesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(tenantmanagercontrolplanesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(tenantmanagercontrolplanesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(tenantmanagercontrolplanesdk.OperationStatusCanceled)},
}

type domainOCIClient interface {
	CreateDomain(context.Context, tenantmanagercontrolplanesdk.CreateDomainRequest) (tenantmanagercontrolplanesdk.CreateDomainResponse, error)
	GetDomain(context.Context, tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error)
	ListDomains(context.Context, tenantmanagercontrolplanesdk.ListDomainsRequest) (tenantmanagercontrolplanesdk.ListDomainsResponse, error)
	UpdateDomain(context.Context, tenantmanagercontrolplanesdk.UpdateDomainRequest) (tenantmanagercontrolplanesdk.UpdateDomainResponse, error)
	DeleteDomain(context.Context, tenantmanagercontrolplanesdk.DeleteDomainRequest) (tenantmanagercontrolplanesdk.DeleteDomainResponse, error)
}

type domainWorkRequestClient interface {
	GetWorkRequest(context.Context, tenantmanagercontrolplanesdk.GetWorkRequestRequest) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error)
}

type domainRuntimeClient struct {
	delegate DomainServiceClient
}

type domainProjectedStatus struct {
	Id             string                     `json:"id,omitempty"`
	DomainName     string                     `json:"domainName,omitempty"`
	OwnerId        string                     `json:"ownerId,omitempty"`
	LifecycleState string                     `json:"lifecycleState,omitempty"`
	Status         string                     `json:"sdkStatus,omitempty"`
	TxtRecord      string                     `json:"txtRecord,omitempty"`
	TimeCreated    string                     `json:"timeCreated,omitempty"`
	TimeUpdated    string                     `json:"timeUpdated,omitempty"`
	FreeformTags   map[string]string          `json:"freeformTags,omitempty"`
	DefinedTags    map[string]shared.MapValue `json:"definedTags,omitempty"`
}

type domainListCall func(context.Context, tenantmanagercontrolplanesdk.ListDomainsRequest) (tenantmanagercontrolplanesdk.ListDomainsResponse, error)

func init() {
	registerDomainRuntimeHooksMutator(func(manager *DomainServiceManager, hooks *DomainRuntimeHooks) {
		domainClient, workRequestClient, domainInitErr, workRequestInitErr := newDomainRuntimeClients(manager)
		applyDomainRuntimeHooks(hooks, domainClient, workRequestClient, domainInitErr, workRequestInitErr)
	})
}

func newDomainRuntimeClients(
	manager *DomainServiceManager,
) (domainOCIClient, domainWorkRequestClient, error, error) {
	if manager == nil {
		err := fmt.Errorf("Domain service manager is nil")
		return nil, nil, err, err
	}

	domainClient, domainErr := tenantmanagercontrolplanesdk.NewDomainClientWithConfigurationProvider(manager.Provider)
	workRequestClient, workRequestErr := tenantmanagercontrolplanesdk.NewWorkRequestClientWithConfigurationProvider(manager.Provider)
	if domainErr != nil {
		domainClient = tenantmanagercontrolplanesdk.DomainClient{}
	}
	if workRequestErr != nil {
		workRequestClient = tenantmanagercontrolplanesdk.WorkRequestClient{}
	}
	return domainClient, workRequestClient, domainErr, workRequestErr
}

func applyDomainRuntimeHooks(
	hooks *DomainRuntimeHooks,
	domainClient domainOCIClient,
	workRequestClient domainWorkRequestClient,
	domainInitErr error,
	workRequestInitErr error,
) {
	if hooks == nil {
		return
	}
	if hooks.Semantics != nil {
		if hooks.Semantics.List != nil {
			// ListDomains is already request-scoped by compartmentId, but DomainSummary does not
			// echo compartmentId back in the list item payload, so matching on it prevents truthful adoption.
			hooks.Semantics.List.MatchFields = []string{"domainName", "lifecycleState", "status"}
		}
		hooks.Semantics.DeleteFollowUp.Strategy = "confirm-delete"
	}

	hooks.BuildCreateBody = buildDomainCreateBody
	hooks.BuildUpdateBody = buildDomainUpdateBody
	hooks.ParityHooks.NormalizeDesiredState = normalizeDomainDesiredState
	hooks.Read.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &tenantmanagercontrolplanesdk.GetDomainRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), hooks.Get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := hooks.Get.Call(ctx, *request.(*tenantmanagercontrolplanesdk.GetDomainRequest))
			if err != nil {
				return nil, err
			}
			return domainProjectedStatusFromDomain(response.Domain), nil
		},
	}
	hooks.StatusHooks.ProjectStatus = projectDomainStatus
	hooks.Create.Call = func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.CreateDomainRequest,
	) (tenantmanagercontrolplanesdk.CreateDomainResponse, error) {
		if err := domainRuntimeClientReady(domainClient, domainInitErr); err != nil {
			return tenantmanagercontrolplanesdk.CreateDomainResponse{}, err
		}
		return domainClient.CreateDomain(ctx, request)
	}
	hooks.Get.Call = func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.GetDomainRequest,
	) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
		if err := domainRuntimeClientReady(domainClient, domainInitErr); err != nil {
			return tenantmanagercontrolplanesdk.GetDomainResponse{}, err
		}
		return domainClient.GetDomain(ctx, request)
	}
	hooks.List.Fields = domainListFields()
	hooks.List.Call = wrapDomainListCall(func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.ListDomainsRequest,
	) (tenantmanagercontrolplanesdk.ListDomainsResponse, error) {
		if err := domainRuntimeClientReady(domainClient, domainInitErr); err != nil {
			return tenantmanagercontrolplanesdk.ListDomainsResponse{}, err
		}
		return domainClient.ListDomains(ctx, request)
	})
	hooks.Update.Call = func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.UpdateDomainRequest,
	) (tenantmanagercontrolplanesdk.UpdateDomainResponse, error) {
		if err := domainRuntimeClientReady(domainClient, domainInitErr); err != nil {
			return tenantmanagercontrolplanesdk.UpdateDomainResponse{}, err
		}
		return domainClient.UpdateDomain(ctx, request)
	}
	hooks.Delete.Call = func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.DeleteDomainRequest,
	) (tenantmanagercontrolplanesdk.DeleteDomainResponse, error) {
		if err := domainRuntimeClientReady(domainClient, domainInitErr); err != nil {
			return tenantmanagercontrolplanesdk.DeleteDomainResponse{}, err
		}
		return domainClient.DeleteDomain(ctx, request)
	}
	hooks.Async.Adapter = domainWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDomainWorkRequest(ctx, workRequestClient, workRequestInitErr, workRequestID)
	}
	hooks.Async.RecoverResourceID = recoverDomainIDFromWorkRequest
	hooks.DeleteHooks.ApplyOutcome = applyDomainDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DomainServiceClient) DomainServiceClient {
		return domainRuntimeClient{delegate: delegate}
	})
}

func newDomainServiceClientWithClients(
	log loggerutil.OSOKLogger,
	domainClient domainOCIClient,
	workRequestClient domainWorkRequestClient,
) DomainServiceClient {
	hooks := newDomainRuntimeHooksWithClients(domainClient)
	applyDomainRuntimeHooks(&hooks, domainClient, workRequestClient, nil, nil)
	delegate := defaultDomainServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*tenantmanagercontrolplanev1beta1.Domain](
			buildDomainGeneratedRuntimeConfig(&DomainServiceManager{Log: log}, hooks),
		),
	}
	return wrapDomainGeneratedClient(hooks, delegate)
}

func newDomainRuntimeHooksWithClients(domainClient domainOCIClient) DomainRuntimeHooks {
	_ = domainClient
	return newDomainDefaultRuntimeHooks(tenantmanagercontrolplanesdk.DomainClient{})
}

func domainListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:        "DomainId",
			RequestName:      "domainId",
			Contribution:     "query",
			PreferResourceID: true,
		},
		{
			FieldName:    "LifecycleState",
			RequestName:  "lifecycleState",
			Contribution: "query",
			LookupPaths:  []string{"spec.lifecycleState", "status.lifecycleState", "lifecycleState"},
		},
		{
			FieldName:    "Status",
			RequestName:  "sdkStatus",
			Contribution: "query",
			LookupPaths:  []string{"spec.status", "status.sdkStatus"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths:  []string{"status.domainName", "spec.domainName", "domainName", "name"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func wrapDomainListCall(call domainListCall) domainListCall {
	if call == nil {
		return nil
	}

	return func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.ListDomainsRequest,
	) (tenantmanagercontrolplanesdk.ListDomainsResponse, error) {
		return listDomainPages(ctx, call, request)
	}
}

func listDomainPages(
	ctx context.Context,
	call domainListCall,
	request tenantmanagercontrolplanesdk.ListDomainsRequest,
) (tenantmanagercontrolplanesdk.ListDomainsResponse, error) {
	if call == nil {
		return tenantmanagercontrolplanesdk.ListDomainsResponse{}, fmt.Errorf("Domain list call is not configured")
	}

	var combined tenantmanagercontrolplanesdk.ListDomainsResponse
	seenPages := map[string]struct{}{}
	for {
		pageToken := ""
		if request.Page != nil {
			pageToken = strings.TrimSpace(*request.Page)
		}
		if _, seen := seenPages[pageToken]; seen {
			return tenantmanagercontrolplanesdk.ListDomainsResponse{}, fmt.Errorf("Domain list pagination repeated page %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return tenantmanagercontrolplanesdk.ListDomainsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}

		nextPage := strings.TrimSpace(*response.OpcNextPage)
		combined.OpcNextPage = common.String(nextPage)
		request.Page = common.String(nextPage)
	}
}

func buildDomainCreateBody(
	_ context.Context,
	resource *tenantmanagercontrolplanev1beta1.Domain,
	_ string,
) (any, error) {
	if resource == nil {
		return tenantmanagercontrolplanesdk.CreateDomainDetails{}, fmt.Errorf("Domain resource is nil")
	}
	if err := validateDomainSpec(resource.Spec); err != nil {
		return tenantmanagercontrolplanesdk.CreateDomainDetails{}, err
	}

	details := tenantmanagercontrolplanesdk.CreateDomainDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DomainName:    common.String(strings.TrimSpace(resource.Spec.DomainName)),
	}
	if email := strings.TrimSpace(resource.Spec.SubscriptionEmail); email != "" {
		details.SubscriptionEmail = common.String(email)
	}
	if resource.Spec.IsGovernanceEnabled {
		details.IsGovernanceEnabled = common.Bool(true)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = domainDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildDomainUpdateBody(
	_ context.Context,
	resource *tenantmanagercontrolplanev1beta1.Domain,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return tenantmanagercontrolplanesdk.UpdateDomainDetails{}, false, fmt.Errorf("Domain resource is nil")
	}
	if err := validateDomainSpec(resource.Spec); err != nil {
		return tenantmanagercontrolplanesdk.UpdateDomainDetails{}, false, err
	}
	if err := validateDomainTrackedCreateOnlyDrift(resource); err != nil {
		return tenantmanagercontrolplanesdk.UpdateDomainDetails{}, false, err
	}

	current, ok := domainProjectionFromResponse(currentResponse)
	if !ok {
		return tenantmanagercontrolplanesdk.UpdateDomainDetails{}, false, fmt.Errorf("current Domain response does not expose a readable projected body")
	}

	details := tenantmanagercontrolplanesdk.UpdateDomainDetails{}
	updateNeeded := false
	if resource.Spec.FreeformTags != nil {
		desired := maps.Clone(resource.Spec.FreeformTags)
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			details.FreeformTags = desired
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := cloneDomainDefinedTags(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			details.DefinedTags = domainDefinedTagsFromSpec(resource.Spec.DefinedTags)
			updateNeeded = true
		}
	}
	return details, updateNeeded, nil
}

func normalizeDomainDesiredState(resource *tenantmanagercontrolplanev1beta1.Domain, currentResponse any) {
	if resource == nil {
		return
	}

	resource.Spec.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	resource.Spec.DomainName = strings.TrimSpace(resource.Spec.DomainName)
	resource.Spec.SubscriptionEmail = strings.TrimSpace(resource.Spec.SubscriptionEmail)
	resource.Spec.Status = strings.TrimSpace(resource.Spec.Status)
	resource.Spec.LifecycleState = strings.TrimSpace(resource.Spec.LifecycleState)

	if currentResponse != nil {
		resource.Spec.Status = ""
		resource.Spec.LifecycleState = ""
	}
}

func validateDomainSpec(spec tenantmanagercontrolplanev1beta1.DomainSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DomainName) == "" {
		missing = append(missing, "domainName")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("Domain spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func validateDomainTrackedCreateOnlyDrift(resource *tenantmanagercontrolplanev1beta1.Domain) error {
	if resource == nil {
		return fmt.Errorf("Domain resource is nil")
	}
	if currentCompartmentID := strings.TrimSpace(resource.Status.CompartmentId); currentCompartmentID != "" &&
		currentCompartmentID != strings.TrimSpace(resource.Spec.CompartmentId) {
		return fmt.Errorf("Domain formal semantics require replacement when compartmentId changes")
	}
	if currentEmail := strings.TrimSpace(resource.Status.SubscriptionEmail); currentEmail != "" &&
		currentEmail != strings.TrimSpace(resource.Spec.SubscriptionEmail) {
		return fmt.Errorf("Domain formal semantics require replacement when subscriptionEmail changes")
	}
	if strings.TrimSpace(resource.Status.CompartmentId) != "" &&
		resource.Status.IsGovernanceEnabled != resource.Spec.IsGovernanceEnabled {
		return fmt.Errorf("Domain formal semantics require replacement when isGovernanceEnabled changes")
	}
	return nil
}

func domainDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func getDomainWorkRequest(
	ctx context.Context,
	client domainWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Domain work request client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Domain work request client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, tenantmanagercontrolplanesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func recoverDomainIDFromWorkRequest(
	resource *tenantmanagercontrolplanev1beta1.Domain,
	workRequest any,
	_ shared.OSOKAsyncPhase,
) (string, error) {
	if currentID := trackedDomainID(resource); currentID != "" {
		return currentID, nil
	}

	current, ok := domainWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("Domain work request payload does not expose a WorkRequest body")
	}
	for _, item := range current.Resources {
		if entityType := strings.TrimSpace(stringPtrValue(item.EntityType)); entityType != "" &&
			!strings.EqualFold(entityType, "Domain") {
			continue
		}
		if identifier := strings.TrimSpace(stringPtrValue(item.Identifier)); identifier != "" {
			return identifier, nil
		}
	}
	return "", fmt.Errorf("Domain create work request %s did not expose a Domain resource identifier", strings.TrimSpace(stringPtrValue(current.Id)))
}

func applyDomainDeleteOutcome(
	resource *tenantmanagercontrolplanev1beta1.Domain,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	lifecycleState := strings.ToUpper(domainLifecycleState(response))
	if lifecycleState == "" || lifecycleState == string(tenantmanagercontrolplanesdk.DomainLifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !domainDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest ||
		stage == generatedruntime.DeleteConfirmStageAlreadyPending {
		markDomainTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func domainDeleteAlreadyPending(resource *tenantmanagercontrolplanev1beta1.Domain) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markDomainTerminating(resource *tenantmanagercontrolplanev1beta1.Domain, response any) {
	if resource == nil {
		return
	}

	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = domainDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       domainLifecycleState(response),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         domainDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", domainDeletePendingMessage, loggerutil.OSOKLogger{})
}

func projectDomainStatus(resource *tenantmanagercontrolplanev1beta1.Domain, response any) error {
	if resource == nil {
		return fmt.Errorf("Domain resource is nil")
	}

	projected, ok := domainProjectionFromResponse(response)
	if !ok {
		return fmt.Errorf("Domain response does not expose a readable projected body")
	}

	resource.Status.Id = projected.Id
	resource.Status.DomainName = projected.DomainName
	resource.Status.OwnerId = projected.OwnerId
	resource.Status.LifecycleState = projected.LifecycleState
	resource.Status.Status = projected.Status
	resource.Status.TxtRecord = projected.TxtRecord
	resource.Status.TimeCreated = projected.TimeCreated
	resource.Status.TimeUpdated = projected.TimeUpdated
	resource.Status.FreeformTags = cloneDomainFreeformTags(projected.FreeformTags)
	resource.Status.DefinedTags = cloneDomainDefinedTags(projected.DefinedTags)
	return nil
}

func domainLifecycleState(response any) string {
	if current, ok := domainProjectionFromResponse(response); ok {
		return current.LifecycleState
	}
	return ""
}

func domainProjectionFromResponse(response any) (domainProjectedStatus, bool) {
	switch typed := response.(type) {
	case domainProjectedStatus:
		return typed, true
	case *domainProjectedStatus:
		if typed != nil {
			return *typed, true
		}
	case tenantmanagercontrolplanesdk.Domain:
		return domainProjectedStatusFromDomain(typed), true
	case *tenantmanagercontrolplanesdk.Domain:
		if typed != nil {
			return domainProjectedStatusFromDomain(*typed), true
		}
	case tenantmanagercontrolplanesdk.CreateDomainResponse:
		return domainProjectedStatusFromDomain(typed.Domain), true
	case *tenantmanagercontrolplanesdk.CreateDomainResponse:
		if typed != nil {
			return domainProjectedStatusFromDomain(typed.Domain), true
		}
	case tenantmanagercontrolplanesdk.GetDomainResponse:
		return domainProjectedStatusFromDomain(typed.Domain), true
	case *tenantmanagercontrolplanesdk.GetDomainResponse:
		if typed != nil {
			return domainProjectedStatusFromDomain(typed.Domain), true
		}
	case tenantmanagercontrolplanesdk.UpdateDomainResponse:
		return domainProjectedStatusFromDomain(typed.Domain), true
	case *tenantmanagercontrolplanesdk.UpdateDomainResponse:
		if typed != nil {
			return domainProjectedStatusFromDomain(typed.Domain), true
		}
	case tenantmanagercontrolplanesdk.DomainSummary:
		return domainProjectedStatusFromSummary(typed), true
	case *tenantmanagercontrolplanesdk.DomainSummary:
		if typed != nil {
			return domainProjectedStatusFromSummary(*typed), true
		}
	}
	return domainProjectedStatus{}, false
}

func domainWorkRequestFromAny(workRequest any) (tenantmanagercontrolplanesdk.WorkRequest, bool) {
	switch typed := workRequest.(type) {
	case tenantmanagercontrolplanesdk.WorkRequest:
		return typed, true
	case *tenantmanagercontrolplanesdk.WorkRequest:
		if typed != nil {
			return *typed, true
		}
	}
	return tenantmanagercontrolplanesdk.WorkRequest{}, false
}

func domainRuntimeClientReady(client domainOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize Domain OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("Domain OCI client is not configured")
	}
	return nil
}

func (c domainRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *tenantmanagercontrolplanev1beta1.Domain,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Domain generated runtime delegate is not configured")
	}

	previousTrackedID := trackedDomainID(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil {
		recordDomainTrackedCreateOnlyStatus(resource, previousTrackedID)
	}
	return response, err
}

func (c domainRuntimeClient) Delete(
	ctx context.Context,
	resource *tenantmanagercontrolplanev1beta1.Domain,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("Domain generated runtime delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func trackedDomainID(resource *tenantmanagercontrolplanev1beta1.Domain) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func recordDomainTrackedCreateOnlyStatus(resource *tenantmanagercontrolplanev1beta1.Domain, previousTrackedID string) {
	if resource == nil {
		return
	}
	currentTrackedID := trackedDomainID(resource)
	if currentTrackedID == "" {
		return
	}

	shouldRecord := previousTrackedID == "" ||
		currentTrackedID != previousTrackedID ||
		strings.TrimSpace(resource.Status.CompartmentId) == ""
	if !shouldRecord {
		return
	}

	resource.Status.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	resource.Status.SubscriptionEmail = strings.TrimSpace(resource.Spec.SubscriptionEmail)
	resource.Status.IsGovernanceEnabled = resource.Spec.IsGovernanceEnabled
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func domainProjectedStatusFromDomain(source tenantmanagercontrolplanesdk.Domain) domainProjectedStatus {
	return domainProjectedStatus{
		Id:             strings.TrimSpace(stringPtrValue(source.Id)),
		DomainName:     strings.TrimSpace(stringPtrValue(source.DomainName)),
		OwnerId:        strings.TrimSpace(stringPtrValue(source.OwnerId)),
		LifecycleState: string(source.LifecycleState),
		Status:         string(source.Status),
		TxtRecord:      strings.TrimSpace(stringPtrValue(source.TxtRecord)),
		TimeCreated:    sdkTimeString(source.TimeCreated),
		TimeUpdated:    sdkTimeString(source.TimeUpdated),
		FreeformTags:   cloneDomainFreeformTags(source.FreeformTags),
		DefinedTags:    domainStatusDefinedTags(source.DefinedTags),
	}
}

func domainProjectedStatusFromSummary(source tenantmanagercontrolplanesdk.DomainSummary) domainProjectedStatus {
	return domainProjectedStatus{
		Id:             strings.TrimSpace(stringPtrValue(source.Id)),
		DomainName:     strings.TrimSpace(stringPtrValue(source.DomainName)),
		OwnerId:        strings.TrimSpace(stringPtrValue(source.OwnerId)),
		LifecycleState: string(source.LifecycleState),
		Status:         string(source.Status),
		TxtRecord:      strings.TrimSpace(stringPtrValue(source.TxtRecord)),
		TimeCreated:    sdkTimeString(source.TimeCreated),
		TimeUpdated:    sdkTimeString(source.TimeUpdated),
		FreeformTags:   cloneDomainFreeformTags(source.FreeformTags),
		DefinedTags:    domainStatusDefinedTags(source.DefinedTags),
	}
}

func cloneDomainFreeformTags(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	return maps.Clone(source)
}

func cloneDomainDefinedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		cloned[namespace] = tagValues
	}
	return cloned
}

func domainStatusDefinedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(source) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
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

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}
