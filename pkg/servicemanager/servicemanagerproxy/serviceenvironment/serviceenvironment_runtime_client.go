/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceenvironment

import (
	"context"
	"fmt"
	"strings"

	servicemanagerproxysdk "github.com/oracle/oci-go-sdk/v65/servicemanagerproxy"
	servicemanagerproxyv1beta1 "github.com/oracle/oci-service-operator/api/servicemanagerproxy/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type serviceEnvironmentOCIClient interface {
	GetServiceEnvironment(context.Context, servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error)
	ListServiceEnvironments(context.Context, servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error)
}

type serviceEnvironmentListCall func(context.Context, servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error)

type serviceEnvironmentRuntimeClient struct {
	delegate ServiceEnvironmentServiceClient
	log      loggerutil.OSOKLogger
}

type projectedServiceEnvironmentResponse struct {
	Body         projectedServiceEnvironmentBody `presentIn:"body"`
	OpcRequestId *string                         `presentIn:"header" name:"opc-request-id"`
}

type projectedServiceEnvironmentBody struct {
	Id                          string                                                         `json:"id,omitempty"`
	SubscriptionId              string                                                         `json:"subscriptionId,omitempty"`
	Status                      string                                                         `json:"sdkStatus,omitempty"`
	CompartmentId               string                                                         `json:"compartmentId,omitempty"`
	ServiceDefinition           servicemanagerproxyv1beta1.ServiceEnvironmentServiceDefinition `json:"serviceDefinition,omitempty"`
	ConsoleUrl                  string                                                         `json:"consoleUrl,omitempty"`
	ServiceEnvironmentEndpoints []servicemanagerproxyv1beta1.ServiceEnvironmentEndpoint        `json:"serviceEnvironmentEndpoints,omitempty"`
	DefinedTags                 map[string]shared.MapValue                                     `json:"definedTags,omitempty"`
	FreeformTags                map[string]string                                              `json:"freeformTags,omitempty"`
}

func init() {
	registerServiceEnvironmentRuntimeHooksMutator(func(manager *ServiceEnvironmentServiceManager, hooks *ServiceEnvironmentRuntimeHooks) {
		applyServiceEnvironmentRuntimeHooks(manager, hooks)
	})
}

func applyServiceEnvironmentRuntimeHooks(
	manager *ServiceEnvironmentServiceManager,
	hooks *ServiceEnvironmentRuntimeHooks,
) {
	if hooks == nil {
		return
	}

	hooks.Get.Fields = serviceEnvironmentGetFields()
	hooks.List.Fields = serviceEnvironmentListFields()
	if hooks.Get.Call != nil {
		hooks.Read.Get = serviceEnvironmentGetReadOperation(hooks.Get, hooks.List)
	}
	hooks.Read.List = nil
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ServiceEnvironmentServiceClient) ServiceEnvironmentServiceClient {
		return serviceEnvironmentRuntimeClient{
			delegate: delegate,
			log:      serviceEnvironmentLogger(manager),
		}
	})
}

func newServiceEnvironmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client serviceEnvironmentOCIClient,
) ServiceEnvironmentServiceClient {
	hooks := newServiceEnvironmentRuntimeHooksWithOCIClient(client)
	applyServiceEnvironmentRuntimeHooks(nil, &hooks)
	delegate := defaultServiceEnvironmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*servicemanagerproxyv1beta1.ServiceEnvironment](
			buildServiceEnvironmentGeneratedRuntimeConfig(&ServiceEnvironmentServiceManager{Log: log}, hooks),
		),
	}
	return wrapServiceEnvironmentGeneratedClient(hooks, delegate)
}

func newServiceEnvironmentRuntimeHooksWithOCIClient(client serviceEnvironmentOCIClient) ServiceEnvironmentRuntimeHooks {
	return ServiceEnvironmentRuntimeHooks{
		Semantics:       newServiceEnvironmentRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*servicemanagerproxyv1beta1.ServiceEnvironment]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*servicemanagerproxyv1beta1.ServiceEnvironment]{},
		StatusHooks:     generatedruntime.StatusHooks[*servicemanagerproxyv1beta1.ServiceEnvironment]{},
		ParityHooks:     generatedruntime.ParityHooks[*servicemanagerproxyv1beta1.ServiceEnvironment]{},
		Async:           generatedruntime.AsyncHooks[*servicemanagerproxyv1beta1.ServiceEnvironment]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*servicemanagerproxyv1beta1.ServiceEnvironment]{},
		Get: runtimeOperationHooks[servicemanagerproxysdk.GetServiceEnvironmentRequest, servicemanagerproxysdk.GetServiceEnvironmentResponse]{
			Fields: serviceEnvironmentGetFields(),
			Call: func(ctx context.Context, request servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
				if client == nil {
					return servicemanagerproxysdk.GetServiceEnvironmentResponse{}, fmt.Errorf("ServiceEnvironment OCI client is not configured")
				}
				return client.GetServiceEnvironment(ctx, request)
			},
		},
		List: runtimeOperationHooks[servicemanagerproxysdk.ListServiceEnvironmentsRequest, servicemanagerproxysdk.ListServiceEnvironmentsResponse]{
			Fields: serviceEnvironmentListFields(),
			Call: func(ctx context.Context, request servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error) {
				if client == nil {
					return servicemanagerproxysdk.ListServiceEnvironmentsResponse{}, fmt.Errorf("ServiceEnvironment OCI client is not configured")
				}
				return client.ListServiceEnvironments(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ServiceEnvironmentServiceClient) ServiceEnvironmentServiceClient{},
	}
}

func serviceEnvironmentLogger(manager *ServiceEnvironmentServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func projectServiceEnvironmentResponse(
	response servicemanagerproxysdk.GetServiceEnvironmentResponse,
) projectedServiceEnvironmentResponse {
	return projectedServiceEnvironmentResponse{
		Body: projectedServiceEnvironmentBody{
			Id:                          stringPtrValue(response.Id),
			SubscriptionId:              stringPtrValue(response.SubscriptionId),
			Status:                      strings.TrimSpace(string(response.Status)),
			CompartmentId:               stringPtrValue(response.CompartmentId),
			ServiceDefinition:           projectServiceEnvironmentDefinition(response.ServiceDefinition),
			ConsoleUrl:                  stringPtrValue(response.ConsoleUrl),
			ServiceEnvironmentEndpoints: projectServiceEnvironmentEndpoints(response.ServiceEnvironmentEndpoints),
		},
		OpcRequestId: response.OpcRequestId,
	}
}

func projectServiceEnvironmentDefinition(
	definition *servicemanagerproxysdk.ServiceDefinition,
) servicemanagerproxyv1beta1.ServiceEnvironmentServiceDefinition {
	if definition == nil {
		return servicemanagerproxyv1beta1.ServiceEnvironmentServiceDefinition{}
	}
	return servicemanagerproxyv1beta1.ServiceEnvironmentServiceDefinition{
		Type:             stringPtrValue(definition.Type),
		DisplayName:      stringPtrValue(definition.DisplayName),
		ShortDisplayName: stringPtrValue(definition.ShortDisplayName),
	}
}

func projectServiceEnvironmentEndpoints(
	endpoints []servicemanagerproxysdk.ServiceEnvironmentEndPointOverview,
) []servicemanagerproxyv1beta1.ServiceEnvironmentEndpoint {
	if len(endpoints) == 0 {
		return nil
	}
	projected := make([]servicemanagerproxyv1beta1.ServiceEnvironmentEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		projected = append(projected, servicemanagerproxyv1beta1.ServiceEnvironmentEndpoint{
			EnvironmentType: strings.TrimSpace(string(endpoint.EnvironmentType)),
			Url:             stringPtrValue(endpoint.Url),
			Description:     stringPtrValue(endpoint.Description),
		})
	}
	return projected
}

func serviceEnvironmentGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "ServiceEnvironmentId",
			RequestName:      "serviceEnvironmentId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "spec.serviceEnvironmentId", "serviceEnvironmentId"},
		},
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
	}
}

func serviceEnvironmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:        "ServiceEnvironmentId",
			RequestName:      "serviceEnvironmentId",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "spec.serviceEnvironmentId", "serviceEnvironmentId"},
		},
		{
			FieldName:    "Page",
			RequestName:  "page",
			Contribution: "query",
		},
	}
}

func serviceEnvironmentGetReadOperation(
	get runtimeOperationHooks[servicemanagerproxysdk.GetServiceEnvironmentRequest, servicemanagerproxysdk.GetServiceEnvironmentResponse],
	list runtimeOperationHooks[servicemanagerproxysdk.ListServiceEnvironmentsRequest, servicemanagerproxysdk.ListServiceEnvironmentsResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &servicemanagerproxysdk.GetServiceEnvironmentRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			typed := *request.(*servicemanagerproxysdk.GetServiceEnvironmentRequest)
			response, err := get.Call(ctx, typed)
			switch {
			case err == nil:
				return projectServiceEnvironmentResponse(response), nil
			case !servicemanager.IsNotFoundServiceError(err) || list.Call == nil:
				return response, err
			}

			confirmed, confirmErr := confirmServiceEnvironmentAbsence(ctx, list.Call, typed)
			switch {
			case confirmErr != nil:
				return nil, confirmErr
			case !confirmed:
				return nil, fmt.Errorf(
					"ServiceEnvironment GetServiceEnvironment no longer returns %q, but ListServiceEnvironments still finds a matching environment",
					stringPtrValue(typed.ServiceEnvironmentId),
				)
			default:
				return nil, err
			}
		},
	}
}

func confirmServiceEnvironmentAbsence(
	ctx context.Context,
	listCall serviceEnvironmentListCall,
	getRequest servicemanagerproxysdk.GetServiceEnvironmentRequest,
) (bool, error) {
	if listCall == nil {
		return true, nil
	}

	request := servicemanagerproxysdk.ListServiceEnvironmentsRequest{
		CompartmentId:        getRequest.CompartmentId,
		ServiceEnvironmentId: getRequest.ServiceEnvironmentId,
	}

	for {
		response, err := listCall(ctx, request)
		if err != nil {
			if servicemanager.IsNotFoundServiceError(err) {
				return true, nil
			}
			return false, err
		}
		if serviceEnvironmentListContainsID(response.Items, getRequest) {
			return false, nil
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return true, nil
		}
		request.Page = response.OpcNextPage
	}
}

func serviceEnvironmentListContainsID(
	items []servicemanagerproxysdk.ServiceEnvironmentSummary,
	request servicemanagerproxysdk.GetServiceEnvironmentRequest,
) bool {
	serviceEnvironmentID := stringPtrValue(request.ServiceEnvironmentId)
	compartmentID := stringPtrValue(request.CompartmentId)
	for _, item := range items {
		if stringPtrValue(item.Id) == serviceEnvironmentID && stringPtrValue(item.CompartmentId) == compartmentID {
			return true
		}
	}
	return false
}

func (c serviceEnvironmentRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *servicemanagerproxyv1beta1.ServiceEnvironment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ServiceEnvironment delegate is not configured")
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		return response, err
	}
	return projectServiceEnvironmentLifecycle(resource, c.log), nil
}

func (c serviceEnvironmentRuntimeClient) Delete(
	_ context.Context,
	resource *servicemanagerproxyv1beta1.ServiceEnvironment,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("ServiceEnvironment resource must not be nil")
	}

	markServiceEnvironmentDeleted(resource, c.log)
	return true, nil
}

func projectServiceEnvironmentLifecycle(
	resource *servicemanagerproxyv1beta1.ServiceEnvironment,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)

	now := metav1.Now()
	if serviceEnvironmentID := firstNonEmptyTrim(resource.Status.Id, string(status.Ocid), resource.Spec.ServiceEnvironmentId); serviceEnvironmentID != "" {
		status.Ocid = shared.OCID(serviceEnvironmentID)
		if status.CreatedAt == nil {
			status.CreatedAt = &now
		}
	}

	rawStatus := strings.ToUpper(firstNonEmptyTrim(resource.Status.Status))
	message := serviceEnvironmentStatusMessage(resource, rawStatus)
	status.Message = message
	status.UpdatedAt = &now

	switch {
	case serviceEnvironmentIsProvisioning(rawStatus):
		status.Reason = string(shared.Provisioning)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Provisioning, v1.ConditionTrue, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true}
	case serviceEnvironmentIsUpdating(rawStatus):
		status.Reason = string(shared.Updating)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Updating, v1.ConditionTrue, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true}
	case serviceEnvironmentIsStable(rawStatus):
		status.Reason = string(shared.Active)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		status.Reason = string(shared.Failed)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}

func markServiceEnvironmentDeleted(
	resource *servicemanagerproxyv1beta1.ServiceEnvironment,
	log loggerutil.OSOKLogger,
) {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)

	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = serviceEnvironmentDeleteMessage(resource)
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, log)
}

func serviceEnvironmentStatusMessage(resource *servicemanagerproxyv1beta1.ServiceEnvironment, rawStatus string) string {
	subject := firstNonEmptyTrim(
		resource.Status.ServiceDefinition.ShortDisplayName,
		resource.Status.ServiceDefinition.DisplayName,
		resource.Status.Id,
		resource.Spec.ServiceEnvironmentId,
	)
	if subject == "" {
		subject = "ServiceEnvironment"
	}
	if rawStatus == "" {
		return fmt.Sprintf("%s status is not reported", subject)
	}
	return fmt.Sprintf("%s status is %s", subject, rawStatus)
}

func serviceEnvironmentDeleteMessage(resource *servicemanagerproxyv1beta1.ServiceEnvironment) string {
	subject := firstNonEmptyTrim(
		resource.Status.ServiceDefinition.ShortDisplayName,
		resource.Status.ServiceDefinition.DisplayName,
		resource.Status.Id,
		resource.Spec.ServiceEnvironmentId,
	)
	if subject == "" {
		subject = "ServiceEnvironment"
	}
	return fmt.Sprintf("%s was released from Kubernetes control", subject)
}

func serviceEnvironmentIsProvisioning(rawStatus string) bool {
	switch rawStatus {
	case "INITIALIZED", "BEGIN_ACTIVATION":
		return true
	default:
		return false
	}
}

func serviceEnvironmentIsUpdating(rawStatus string) bool {
	switch rawStatus {
	case "BEGIN_SOFT_TERMINATION",
		"BEGIN_TERMINATION",
		"BEGIN_DISABLING",
		"BEGIN_ENABLING",
		"BEGIN_MIGRATION",
		"BEGIN_SUSPENSION",
		"BEGIN_RESUMPTION",
		"BEGIN_LOCK_RELOCATION",
		"BEGIN_RELOCATION",
		"BEGIN_UNLOCK_RELOCATION",
		"BEGIN_DISABLING_ACCESS",
		"BEGIN_ENABLING_ACCESS":
		return true
	default:
		return false
	}
}

func serviceEnvironmentIsStable(rawStatus string) bool {
	switch rawStatus {
	case "ACTIVE",
		"SOFT_TERMINATED",
		"CANCELED",
		"TERMINATED",
		"DISABLED",
		"SUSPENDED",
		"LOCKED_RELOCATION",
		"RELOCATED",
		"UNLOCKED_RELOCATION",
		"ACCESS_DISABLED":
		return true
	default:
		return false
	}
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ ServiceEnvironmentServiceClient = serviceEnvironmentRuntimeClient{}

var _ serviceEnvironmentOCIClient = (*servicemanagerproxysdk.ServiceManagerProxyClient)(nil)
