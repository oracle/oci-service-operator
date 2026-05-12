/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package instanceagentplugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	computeinstanceagentsdk "github.com/oracle/oci-go-sdk/v65/computeinstanceagent"
	computeinstanceagentv1beta1 "github.com/oracle/oci-service-operator/api/computeinstanceagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type instanceAgentPluginOCIClient interface {
	GetInstanceAgentPlugin(context.Context, computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error)
	ListInstanceAgentPlugins(context.Context, computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error)
}

type instanceAgentPluginListCall func(context.Context, computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error)

type instanceAgentPluginRuntimeClient struct {
	delegate InstanceAgentPluginServiceClient
	log      loggerutil.OSOKLogger
}

type projectedInstanceAgentPluginResponse struct {
	Body         projectedInstanceAgentPluginBody `presentIn:"body"`
	OpcRequestId *string                          `presentIn:"header" name:"opc-request-id"`
}

type projectedInstanceAgentPluginBody struct {
	Name               string `json:"name,omitempty"`
	PluginName         string `json:"pluginName,omitempty"`
	SDKStatus          string `json:"sdkStatus,omitempty"`
	LifecycleState     string `json:"lifecycleState,omitempty"`
	TimeLastUpdatedUtc string `json:"timeLastUpdatedUtc,omitempty"`
	Message            string `json:"message,omitempty"`
	InstanceagentId    string `json:"instanceagentId,omitempty"`
	CompartmentId      string `json:"compartmentId,omitempty"`
}

func init() {
	registerInstanceAgentPluginRuntimeHooksMutator(func(manager *InstanceAgentPluginServiceManager, hooks *InstanceAgentPluginRuntimeHooks) {
		applyInstanceAgentPluginRuntimeHooks(manager, hooks)
	})
}

func applyInstanceAgentPluginRuntimeHooks(
	manager *InstanceAgentPluginServiceManager,
	hooks *InstanceAgentPluginRuntimeHooks,
) {
	if hooks == nil {
		return
	}

	hooks.Get.Fields = instanceAgentPluginGetFields()
	hooks.List.Fields = instanceAgentPluginListFields()
	if hooks.Get.Call != nil {
		hooks.Read.Get = instanceAgentPluginGetReadOperation(hooks.Get, hooks.List)
	}
	hooks.Read.List = nil
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate InstanceAgentPluginServiceClient) InstanceAgentPluginServiceClient {
		return instanceAgentPluginRuntimeClient{
			delegate: delegate,
			log:      instanceAgentPluginLogger(manager),
		}
	})
}

func newInstanceAgentPluginServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client instanceAgentPluginOCIClient,
) InstanceAgentPluginServiceClient {
	hooks := newInstanceAgentPluginRuntimeHooksWithOCIClient(client)
	applyInstanceAgentPluginRuntimeHooks(nil, &hooks)
	delegate := defaultInstanceAgentPluginServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*computeinstanceagentv1beta1.InstanceAgentPlugin](
			buildInstanceAgentPluginGeneratedRuntimeConfig(&InstanceAgentPluginServiceManager{Log: log}, hooks),
		),
	}
	return wrapInstanceAgentPluginGeneratedClient(hooks, delegate)
}

func newInstanceAgentPluginRuntimeHooksWithOCIClient(client instanceAgentPluginOCIClient) InstanceAgentPluginRuntimeHooks {
	return InstanceAgentPluginRuntimeHooks{
		Semantics:       newInstanceAgentPluginRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*computeinstanceagentv1beta1.InstanceAgentPlugin]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*computeinstanceagentv1beta1.InstanceAgentPlugin]{},
		StatusHooks:     generatedruntime.StatusHooks[*computeinstanceagentv1beta1.InstanceAgentPlugin]{},
		ParityHooks:     generatedruntime.ParityHooks[*computeinstanceagentv1beta1.InstanceAgentPlugin]{},
		Async:           generatedruntime.AsyncHooks[*computeinstanceagentv1beta1.InstanceAgentPlugin]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*computeinstanceagentv1beta1.InstanceAgentPlugin]{},
		Get: runtimeOperationHooks[computeinstanceagentsdk.GetInstanceAgentPluginRequest, computeinstanceagentsdk.GetInstanceAgentPluginResponse]{
			Fields: instanceAgentPluginGetFields(),
			Call: func(ctx context.Context, request computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
				if client == nil {
					return computeinstanceagentsdk.GetInstanceAgentPluginResponse{}, fmt.Errorf("InstanceAgentPlugin OCI client is not configured")
				}
				return client.GetInstanceAgentPlugin(ctx, request)
			},
		},
		List: runtimeOperationHooks[computeinstanceagentsdk.ListInstanceAgentPluginsRequest, computeinstanceagentsdk.ListInstanceAgentPluginsResponse]{
			Fields: instanceAgentPluginListFields(),
			Call: func(ctx context.Context, request computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error) {
				if client == nil {
					return computeinstanceagentsdk.ListInstanceAgentPluginsResponse{}, fmt.Errorf("InstanceAgentPlugin OCI client is not configured")
				}
				return client.ListInstanceAgentPlugins(ctx, request)
			},
		},
		WrapGeneratedClient: []func(InstanceAgentPluginServiceClient) InstanceAgentPluginServiceClient{},
	}
}

func instanceAgentPluginLogger(manager *InstanceAgentPluginServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func projectInstanceAgentPluginResponse(
	response computeinstanceagentsdk.GetInstanceAgentPluginResponse,
	request computeinstanceagentsdk.GetInstanceAgentPluginRequest,
) projectedInstanceAgentPluginResponse {
	name := stringPtrValue(response.Name)
	rawStatus := strings.TrimSpace(string(response.Status))
	return projectedInstanceAgentPluginResponse{
		Body: projectedInstanceAgentPluginBody{
			Name:               name,
			PluginName:         firstNonEmptyTrim(name, stringPtrValue(request.PluginName)),
			SDKStatus:          rawStatus,
			LifecycleState:     rawStatus,
			TimeLastUpdatedUtc: sdkTimeString(response.TimeLastUpdatedUtc),
			Message:            stringPtrValue(response.Message),
			InstanceagentId:    stringPtrValue(request.InstanceagentId),
			CompartmentId:      stringPtrValue(request.CompartmentId),
		},
		OpcRequestId: response.OpcRequestId,
	}
}

func instanceAgentPluginGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "InstanceagentId",
			RequestName:  "instanceagentId",
			Contribution: "path",
			LookupPaths:  []string{"status.instanceagentId", "spec.instanceagentId", "instanceagentId"},
		},
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "PluginName",
			RequestName:  "pluginName",
			Contribution: "path",
			LookupPaths:  []string{"status.name", "spec.pluginName", "pluginName"},
		},
	}
}

func instanceAgentPluginListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "InstanceagentId",
			RequestName:  "instanceagentId",
			Contribution: "path",
			LookupPaths:  []string{"status.instanceagentId", "spec.instanceagentId", "instanceagentId"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths:  []string{"status.name", "spec.pluginName", "pluginName"},
		},
		{
			FieldName:    "Page",
			RequestName:  "page",
			Contribution: "query",
		},
	}
}

func instanceAgentPluginGetReadOperation(
	get runtimeOperationHooks[computeinstanceagentsdk.GetInstanceAgentPluginRequest, computeinstanceagentsdk.GetInstanceAgentPluginResponse],
	list runtimeOperationHooks[computeinstanceagentsdk.ListInstanceAgentPluginsRequest, computeinstanceagentsdk.ListInstanceAgentPluginsResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &computeinstanceagentsdk.GetInstanceAgentPluginRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			typed := *request.(*computeinstanceagentsdk.GetInstanceAgentPluginRequest)
			response, err := get.Call(ctx, typed)
			switch {
			case err == nil:
				return projectInstanceAgentPluginResponse(response, typed), nil
			case !servicemanager.IsNotFoundServiceError(err) || list.Call == nil:
				return response, err
			}

			confirmed, confirmErr := confirmInstanceAgentPluginAbsence(ctx, list.Call, typed)
			switch {
			case confirmErr != nil:
				return nil, confirmErr
			case !confirmed:
				return nil, fmt.Errorf(
					"InstanceAgentPlugin GetInstanceAgentPlugin no longer returns %q, but ListInstanceAgentPlugins still finds a matching plugin",
					stringPtrValue(typed.PluginName),
				)
			default:
				return nil, err
			}
		},
	}
}

func confirmInstanceAgentPluginAbsence(
	ctx context.Context,
	listCall instanceAgentPluginListCall,
	getRequest computeinstanceagentsdk.GetInstanceAgentPluginRequest,
) (bool, error) {
	if listCall == nil {
		return true, nil
	}

	request := computeinstanceagentsdk.ListInstanceAgentPluginsRequest{
		CompartmentId:   getRequest.CompartmentId,
		InstanceagentId: getRequest.InstanceagentId,
		Name:            getRequest.PluginName,
	}

	for {
		response, err := listCall(ctx, request)
		if err != nil {
			if servicemanager.IsNotFoundServiceError(err) {
				return true, nil
			}
			return false, err
		}
		if instanceAgentPluginListContainsName(response.Items, getRequest) {
			return false, nil
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return true, nil
		}
		request.Page = response.OpcNextPage
	}
}

func instanceAgentPluginListContainsName(
	items []computeinstanceagentsdk.InstanceAgentPluginSummary,
	request computeinstanceagentsdk.GetInstanceAgentPluginRequest,
) bool {
	pluginName := stringPtrValue(request.PluginName)
	for _, item := range items {
		if stringPtrValue(item.Name) == pluginName {
			return true
		}
	}
	return false
}

func (c instanceAgentPluginRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *computeinstanceagentv1beta1.InstanceAgentPlugin,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("InstanceAgentPlugin delegate is not configured")
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		return response, err
	}
	return projectInstanceAgentPluginLifecycle(resource, c.log), nil
}

func (c instanceAgentPluginRuntimeClient) Delete(
	_ context.Context,
	resource *computeinstanceagentv1beta1.InstanceAgentPlugin,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("InstanceAgentPlugin resource must not be nil")
	}

	markInstanceAgentPluginDeleted(resource, c.log)
	return true, nil
}

func projectInstanceAgentPluginLifecycle(
	resource *computeinstanceagentv1beta1.InstanceAgentPlugin,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)

	now := metav1.Now()
	if trackedID := instanceAgentPluginTrackedID(resource); trackedID != "" {
		status.Ocid = shared.OCID(trackedID)
		if status.CreatedAt == nil {
			status.CreatedAt = &now
		}
	}

	rawStatus := strings.ToUpper(firstNonEmptyTrim(resource.Status.Status))
	message := instanceAgentPluginStatusMessage(resource, rawStatus)
	status.Message = message
	status.UpdatedAt = &now

	switch {
	case instanceAgentPluginIsStable(rawStatus):
		status.Reason = string(shared.Active)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		status.Reason = string(shared.Failed)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}

func markInstanceAgentPluginDeleted(
	resource *computeinstanceagentv1beta1.InstanceAgentPlugin,
	log loggerutil.OSOKLogger,
) {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)

	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = instanceAgentPluginDeleteMessage(resource)
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, log)
}

func instanceAgentPluginTrackedID(resource *computeinstanceagentv1beta1.InstanceAgentPlugin) string {
	if resource == nil {
		return ""
	}

	instanceID := firstNonEmptyTrim(resource.Status.InstanceagentId, resource.Spec.InstanceagentId)
	compartmentID := firstNonEmptyTrim(resource.Status.CompartmentId, resource.Spec.CompartmentId)
	pluginName := firstNonEmptyTrim(resource.Status.Name, resource.Spec.PluginName)
	if instanceID == "" || compartmentID == "" || pluginName == "" {
		return ""
	}
	return strings.Join([]string{instanceID, compartmentID, pluginName}, "|")
}

func instanceAgentPluginStatusMessage(resource *computeinstanceagentv1beta1.InstanceAgentPlugin, rawStatus string) string {
	subject := firstNonEmptyTrim(resource.Status.Name, resource.Spec.PluginName)
	if subject == "" {
		subject = "InstanceAgentPlugin"
	}
	detail := firstNonEmptyTrim(resource.Status.Message)
	switch {
	case rawStatus == "":
		return fmt.Sprintf("%s status is not reported", subject)
	case detail != "":
		return fmt.Sprintf("%s status is %s: %s", subject, rawStatus, detail)
	default:
		return fmt.Sprintf("%s status is %s", subject, rawStatus)
	}
}

func instanceAgentPluginDeleteMessage(resource *computeinstanceagentv1beta1.InstanceAgentPlugin) string {
	subject := firstNonEmptyTrim(resource.Status.Name, resource.Spec.PluginName)
	if subject == "" {
		subject = "InstanceAgentPlugin"
	}
	return fmt.Sprintf("%s was released from Kubernetes control", subject)
}

func instanceAgentPluginIsStable(rawStatus string) bool {
	switch rawStatus {
	case "RUNNING", "STOPPED", "NOT_SUPPORTED", "INVALID":
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

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339Nano)
}

var _ InstanceAgentPluginServiceClient = instanceAgentPluginRuntimeClient{}

var _ instanceAgentPluginOCIClient = (*computeinstanceagentsdk.PluginClient)(nil)
