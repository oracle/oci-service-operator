/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoredinstance

import (
	"context"
	"fmt"
	"strings"

	appmgmtcontrolsdk "github.com/oracle/oci-go-sdk/v65/appmgmtcontrol"
	"github.com/oracle/oci-go-sdk/v65/common"
	appmgmtcontrolv1beta1 "github.com/oracle/oci-service-operator/api/appmgmtcontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type monitoredInstanceOCIClient interface {
	GetMonitoredInstance(context.Context, appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error)
	ListMonitoredInstances(context.Context, appmgmtcontrolsdk.ListMonitoredInstancesRequest) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error)
}

type monitoredInstanceRuntimeClient struct {
	delegate MonitoredInstanceServiceClient
	get      func(context.Context, appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error)
}

type monitoredInstanceListCall func(context.Context, appmgmtcontrolsdk.ListMonitoredInstancesRequest) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error)

func init() {
	registerMonitoredInstanceRuntimeHooksMutator(func(_ *MonitoredInstanceServiceManager, hooks *MonitoredInstanceRuntimeHooks) {
		applyMonitoredInstanceRuntimeHooks(hooks)
	})
}

func applyMonitoredInstanceRuntimeHooks(hooks *MonitoredInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Get.Fields = monitoredInstanceGetFields()
	hooks.List.Fields = monitoredInstanceListFields()
	hooks.List.Call = wrapMonitoredInstanceListCall(hooks.List.Call)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MonitoredInstanceServiceClient) MonitoredInstanceServiceClient {
		return monitoredInstanceRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
		}
	})
}

func newMonitoredInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client monitoredInstanceOCIClient,
) MonitoredInstanceServiceClient {
	hooks := newMonitoredInstanceRuntimeHooksWithOCIClient(client)
	applyMonitoredInstanceRuntimeHooks(&hooks)
	delegate := defaultMonitoredInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*appmgmtcontrolv1beta1.MonitoredInstance](
			buildMonitoredInstanceGeneratedRuntimeConfig(&MonitoredInstanceServiceManager{Log: log}, hooks),
		),
	}
	return wrapMonitoredInstanceGeneratedClient(hooks, delegate)
}

func newMonitoredInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client monitoredInstanceOCIClient,
) generatedruntime.Config[*appmgmtcontrolv1beta1.MonitoredInstance] {
	hooks := newMonitoredInstanceRuntimeHooksWithOCIClient(client)
	applyMonitoredInstanceRuntimeHooks(&hooks)
	return buildMonitoredInstanceGeneratedRuntimeConfig(&MonitoredInstanceServiceManager{Log: log}, hooks)
}

func newMonitoredInstanceRuntimeHooksWithOCIClient(client monitoredInstanceOCIClient) MonitoredInstanceRuntimeHooks {
	return MonitoredInstanceRuntimeHooks{
		Semantics:       newMonitoredInstanceRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*appmgmtcontrolv1beta1.MonitoredInstance]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*appmgmtcontrolv1beta1.MonitoredInstance]{},
		StatusHooks:     generatedruntime.StatusHooks[*appmgmtcontrolv1beta1.MonitoredInstance]{},
		ParityHooks:     generatedruntime.ParityHooks[*appmgmtcontrolv1beta1.MonitoredInstance]{},
		Async:           generatedruntime.AsyncHooks[*appmgmtcontrolv1beta1.MonitoredInstance]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*appmgmtcontrolv1beta1.MonitoredInstance]{},
		Get: runtimeOperationHooks[appmgmtcontrolsdk.GetMonitoredInstanceRequest, appmgmtcontrolsdk.GetMonitoredInstanceResponse]{
			Fields: monitoredInstanceGetFields(),
			Call: func(ctx context.Context, request appmgmtcontrolsdk.GetMonitoredInstanceRequest) (appmgmtcontrolsdk.GetMonitoredInstanceResponse, error) {
				if client == nil {
					return appmgmtcontrolsdk.GetMonitoredInstanceResponse{}, fmt.Errorf("MonitoredInstance OCI client is not configured")
				}
				return client.GetMonitoredInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[appmgmtcontrolsdk.ListMonitoredInstancesRequest, appmgmtcontrolsdk.ListMonitoredInstancesResponse]{
			Fields: monitoredInstanceListFields(),
			Call: func(ctx context.Context, request appmgmtcontrolsdk.ListMonitoredInstancesRequest) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error) {
				if client == nil {
					return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, fmt.Errorf("MonitoredInstance OCI client is not configured")
				}
				return client.ListMonitoredInstances(ctx, request)
			},
		},
		WrapGeneratedClient: []func(MonitoredInstanceServiceClient) MonitoredInstanceServiceClient{},
	}
}

func monitoredInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "MonitoredInstanceId",
			RequestName:      "monitoredInstanceId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.instanceId", "spec.instanceId", "instanceId"},
		},
	}
}

func monitoredInstanceListFields() []generatedruntime.RequestField {
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
		{
			FieldName:    "Page",
			RequestName:  "page",
			Contribution: "query",
		},
	}
}

func wrapMonitoredInstanceListCall(call monitoredInstanceListCall) monitoredInstanceListCall {
	if call == nil {
		return nil
	}

	return func(
		ctx context.Context,
		request appmgmtcontrolsdk.ListMonitoredInstancesRequest,
	) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error) {
		if stringPtrValue(request.CompartmentId) == "" || stringPtrValue(request.DisplayName) == "" {
			return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, fmt.Errorf("MonitoredInstance list bind requires compartmentId plus displayName")
		}
		return listMonitoredInstancePages(ctx, call, request)
	}
}

func listMonitoredInstancePages(
	ctx context.Context,
	call monitoredInstanceListCall,
	request appmgmtcontrolsdk.ListMonitoredInstancesRequest,
) (appmgmtcontrolsdk.ListMonitoredInstancesResponse, error) {
	if call == nil {
		return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, fmt.Errorf("MonitoredInstance list call is not configured")
	}

	var combined appmgmtcontrolsdk.ListMonitoredInstancesResponse
	seenPages := map[string]struct{}{}
	for {
		pageToken := stringPtrValue(request.Page)
		if _, seen := seenPages[pageToken]; seen {
			return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, fmt.Errorf("MonitoredInstance list pagination repeated page %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return appmgmtcontrolsdk.ListMonitoredInstancesResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		if stringPtrValue(response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}

		nextPage := stringPtrValue(response.OpcNextPage)
		combined.OpcNextPage = common.String(nextPage)
		request.Page = common.String(nextPage)
	}
}

func (c monitoredInstanceRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *appmgmtcontrolv1beta1.MonitoredInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("MonitoredInstance generated runtime delegate is not configured")
	}
	if err := validateMonitoredInstanceBinding(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	hadTrackedIdentity := trackedMonitoredInstanceID(resource) != ""
	explicitInstanceID := strings.TrimSpace(resource.Spec.InstanceId)
	restoreDirectBindFields, restoreStatusFields := clearDirectBindLookupFields(resource, explicitInstanceID)
	defer restoreDirectBindFields()

	if !hadTrackedIdentity && explicitInstanceID != "" {
		if err := c.preflightMonitoredInstanceDirectBind(ctx, resource, explicitInstanceID); err != nil {
			restoreStatusFields()
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		restoreStatusFields()
		return response, err
	}
	ensureMonitoredInstanceTrackedIdentity(resource)
	if hadTrackedIdentity || explicitInstanceID != "" || trackedMonitoredInstanceID(resource) == "" {
		return response, nil
	}

	// With Create=nil, the first untracked reconcile binds from
	// ListMonitoredInstances. Run one live-ID-backed delegate pass immediately
	// afterwards so status projection uses GetMonitoredInstance rather than the
	// summary payload from the list response.
	response, err = c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		return response, err
	}
	ensureMonitoredInstanceTrackedIdentity(resource)
	return response, nil
}

func (c monitoredInstanceRuntimeClient) Delete(
	_ context.Context,
	_ *appmgmtcontrolv1beta1.MonitoredInstance,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("MonitoredInstance generated runtime delegate is not configured")
	}
	return true, nil
}

func (c monitoredInstanceRuntimeClient) preflightMonitoredInstanceDirectBind(
	ctx context.Context,
	resource *appmgmtcontrolv1beta1.MonitoredInstance,
	instanceID string,
) error {
	if c.get == nil {
		return fmt.Errorf("MonitoredInstance get hook is not configured")
	}
	if _, err := c.get(ctx, appmgmtcontrolsdk.GetMonitoredInstanceRequest{
		MonitoredInstanceId: common.String(instanceID),
	}); err != nil {
		return err
	}
	seedMonitoredInstanceTrackedIdentity(resource, instanceID)
	return nil
}

func validateMonitoredInstanceBinding(resource *appmgmtcontrolv1beta1.MonitoredInstance) error {
	if resource == nil {
		return fmt.Errorf("MonitoredInstance resource is nil")
	}

	trackedID := trackedMonitoredInstanceID(resource)
	explicitInstanceID := strings.TrimSpace(resource.Spec.InstanceId)
	if trackedID != "" && explicitInstanceID != "" && explicitInstanceID != trackedID {
		return fmt.Errorf("MonitoredInstance formal semantics require replacement when instanceId changes")
	}
	if trackedID != "" {
		return nil
	}
	if explicitInstanceID != "" {
		return nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) != "" && strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return nil
	}
	return fmt.Errorf("MonitoredInstance bind-existing flow requires spec.instanceId or spec.compartmentId plus spec.displayName")
}

func trackedMonitoredInstanceID(resource *appmgmtcontrolv1beta1.MonitoredInstance) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.InstanceId)
}

func seedMonitoredInstanceTrackedIdentity(resource *appmgmtcontrolv1beta1.MonitoredInstance, instanceID string) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(instanceID))
	resource.Status.InstanceId = strings.TrimSpace(instanceID)
}

func ensureMonitoredInstanceTrackedIdentity(resource *appmgmtcontrolv1beta1.MonitoredInstance) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	if instanceID := strings.TrimSpace(resource.Status.InstanceId); instanceID != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(instanceID)
	}
}

func clearDirectBindLookupFields(
	resource *appmgmtcontrolv1beta1.MonitoredInstance,
	explicitInstanceID string,
) (func(), func()) {
	if resource == nil || strings.TrimSpace(explicitInstanceID) == "" {
		return func() {}, func() {}
	}

	originalSpecCompartmentID := resource.Spec.CompartmentId
	originalSpecDisplayName := resource.Spec.DisplayName
	resource.Spec.CompartmentId = ""
	resource.Spec.DisplayName = ""

	originalStatusCompartmentID := resource.Status.CompartmentId
	originalStatusDisplayName := resource.Status.DisplayName
	resource.Status.CompartmentId = ""
	resource.Status.DisplayName = ""

	return func() {
			resource.Spec.CompartmentId = originalSpecCompartmentID
			resource.Spec.DisplayName = originalSpecDisplayName
		}, func() {
			resource.Status.CompartmentId = originalStatusCompartmentID
			resource.Status.DisplayName = originalStatusDisplayName
		}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ MonitoredInstanceServiceClient = monitoredInstanceRuntimeClient{}
var _ monitoredInstanceOCIClient = (*appmgmtcontrolsdk.AppmgmtControlClient)(nil)
