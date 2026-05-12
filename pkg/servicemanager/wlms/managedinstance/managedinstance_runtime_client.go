/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedinstance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	wlmssdk "github.com/oracle/oci-go-sdk/v65/wlms"
	wlmsv1beta1 "github.com/oracle/oci-service-operator/api/wlms/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

var errManagedInstanceRequiresGetBackedResponse = errors.New("ManagedInstance update assessment requires a GetManagedInstance-backed response")

type managedInstanceOCIClient interface {
	GetManagedInstance(context.Context, wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error)
	ListManagedInstances(context.Context, wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error)
	UpdateManagedInstance(context.Context, wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error)
}

type managedInstanceRuntimeClient struct {
	delegate ManagedInstanceServiceClient
	get      func(context.Context, wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error)
}

type managedInstanceListCall func(context.Context, wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error)

func init() {
	registerManagedInstanceRuntimeHooksMutator(func(_ *ManagedInstanceServiceManager, hooks *ManagedInstanceRuntimeHooks) {
		applyManagedInstanceRuntimeHooks(hooks)
	})
}

func applyManagedInstanceRuntimeHooks(hooks *ManagedInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.List.Fields = managedInstanceListFields()
	hooks.List.Call = wrapManagedInstanceListCall(hooks.List.Call)
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *wlmsv1beta1.ManagedInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildManagedInstanceUpdateBody(resource, currentResponse)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagedInstanceServiceClient) ManagedInstanceServiceClient {
		return managedInstanceRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
		}
	})
}

func newManagedInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client managedInstanceOCIClient,
) ManagedInstanceServiceClient {
	hooks := newManagedInstanceRuntimeHooksWithOCIClient(client)
	applyManagedInstanceRuntimeHooks(&hooks)
	delegate := defaultManagedInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*wlmsv1beta1.ManagedInstance](
			buildManagedInstanceGeneratedRuntimeConfig(&ManagedInstanceServiceManager{Log: log}, hooks),
		),
	}
	return wrapManagedInstanceGeneratedClient(hooks, delegate)
}

func newManagedInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client managedInstanceOCIClient,
) generatedruntime.Config[*wlmsv1beta1.ManagedInstance] {
	hooks := newManagedInstanceRuntimeHooksWithOCIClient(client)
	applyManagedInstanceRuntimeHooks(&hooks)
	return buildManagedInstanceGeneratedRuntimeConfig(&ManagedInstanceServiceManager{Log: log}, hooks)
}

func newManagedInstanceRuntimeHooksWithOCIClient(client managedInstanceOCIClient) ManagedInstanceRuntimeHooks {
	return ManagedInstanceRuntimeHooks{
		Semantics:       newManagedInstanceRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*wlmsv1beta1.ManagedInstance]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*wlmsv1beta1.ManagedInstance]{},
		StatusHooks:     generatedruntime.StatusHooks[*wlmsv1beta1.ManagedInstance]{},
		ParityHooks:     generatedruntime.ParityHooks[*wlmsv1beta1.ManagedInstance]{},
		Async:           generatedruntime.AsyncHooks[*wlmsv1beta1.ManagedInstance]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*wlmsv1beta1.ManagedInstance]{},
		Get: runtimeOperationHooks[wlmssdk.GetManagedInstanceRequest, wlmssdk.GetManagedInstanceResponse]{
			Fields: managedInstanceGetFields(),
			Call: func(ctx context.Context, request wlmssdk.GetManagedInstanceRequest) (wlmssdk.GetManagedInstanceResponse, error) {
				return client.GetManagedInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[wlmssdk.ListManagedInstancesRequest, wlmssdk.ListManagedInstancesResponse]{
			Fields: managedInstanceListFields(),
			Call: func(ctx context.Context, request wlmssdk.ListManagedInstancesRequest) (wlmssdk.ListManagedInstancesResponse, error) {
				return client.ListManagedInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[wlmssdk.UpdateManagedInstanceRequest, wlmssdk.UpdateManagedInstanceResponse]{
			Fields: managedInstanceUpdateFields(),
			Call: func(ctx context.Context, request wlmssdk.UpdateManagedInstanceRequest) (wlmssdk.UpdateManagedInstanceResponse, error) {
				return client.UpdateManagedInstance(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ManagedInstanceServiceClient) ManagedInstanceServiceClient{},
	}
}

func managedInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagedInstanceId", RequestName: "managedInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func managedInstanceListFields() []generatedruntime.RequestField {
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
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{
			FieldName:    "PluginStatus",
			RequestName:  "pluginStatus",
			Contribution: "query",
			LookupPaths:  []string{"status.pluginStatus", "spec.pluginStatus", "pluginStatus"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func managedInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagedInstanceId", RequestName: "managedInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateManagedInstanceDetails", RequestName: "UpdateManagedInstanceDetails", Contribution: "body"},
	}
}

func wrapManagedInstanceListCall(call managedInstanceListCall) managedInstanceListCall {
	if call == nil {
		return nil
	}

	return func(
		ctx context.Context,
		request wlmssdk.ListManagedInstancesRequest,
	) (wlmssdk.ListManagedInstancesResponse, error) {
		return listManagedInstancePages(ctx, call, request)
	}
}

func listManagedInstancePages(
	ctx context.Context,
	call managedInstanceListCall,
	request wlmssdk.ListManagedInstancesRequest,
) (wlmssdk.ListManagedInstancesResponse, error) {
	if call == nil {
		return wlmssdk.ListManagedInstancesResponse{}, fmt.Errorf("ManagedInstance list call is not configured")
	}

	var combined wlmssdk.ListManagedInstancesResponse
	seenPages := map[string]struct{}{}
	for {
		pageToken := ""
		if request.Page != nil {
			pageToken = strings.TrimSpace(*request.Page)
		}
		if _, seen := seenPages[pageToken]; seen {
			return wlmssdk.ListManagedInstancesResponse{}, fmt.Errorf("ManagedInstance list pagination repeated page %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return wlmssdk.ListManagedInstancesResponse{}, err
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

func buildManagedInstanceUpdateBody(
	resource *wlmsv1beta1.ManagedInstance,
	currentResponse any,
) (wlmssdk.UpdateManagedInstanceDetails, bool, error) {
	if resource == nil {
		return wlmssdk.UpdateManagedInstanceDetails{}, false, fmt.Errorf("ManagedInstance resource is nil")
	}

	current, err := managedInstanceFromResponse(currentResponse)
	if err != nil {
		return wlmssdk.UpdateManagedInstanceDetails{}, false, err
	}

	details := wlmssdk.UpdateManagedInstanceDetails{}
	updateNeeded := false

	if desired, ok, err := managedInstanceDesiredConfigurationUpdate(resource.Spec.Configuration, current.Configuration); err != nil {
		return wlmssdk.UpdateManagedInstanceDetails{}, false, err
	} else if ok {
		details.Configuration = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func managedInstanceFromResponse(currentResponse any) (wlmssdk.ManagedInstance, error) {
	switch response := currentResponse.(type) {
	case wlmssdk.GetManagedInstanceResponse:
		return response.ManagedInstance, nil
	case *wlmssdk.GetManagedInstanceResponse:
		if response == nil {
			break
		}
		return response.ManagedInstance, nil
	case wlmssdk.UpdateManagedInstanceResponse:
		return response.ManagedInstance, nil
	case *wlmssdk.UpdateManagedInstanceResponse:
		if response == nil {
			break
		}
		return response.ManagedInstance, nil
	case wlmssdk.ManagedInstance:
		return response, nil
	case *wlmssdk.ManagedInstance:
		if response == nil {
			break
		}
		return *response, nil
	}
	return wlmssdk.ManagedInstance{}, fmt.Errorf("%w: current ManagedInstance response does not expose a ManagedInstance body", errManagedInstanceRequiresGetBackedResponse)
}

func managedInstanceDesiredConfigurationUpdate(
	spec map[string]shared.JSONValue,
	current *wlmssdk.ManagedInstanceConfiguration,
) (*wlmssdk.UpdateManagedInstanceConfigurationDetails, bool, error) {
	if spec == nil || len(spec) == 0 {
		return nil, false, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, false, fmt.Errorf("marshal ManagedInstance configuration spec: %w", err)
	}

	var desired wlmssdk.UpdateManagedInstanceConfigurationDetails
	if err := json.Unmarshal(payload, &desired); err != nil {
		return nil, false, fmt.Errorf("decode ManagedInstance configuration update body: %w", err)
	}

	desiredValues, err := managedInstanceJSONBytesToMap(payload)
	if err != nil {
		return nil, false, fmt.Errorf("project desired ManagedInstance configuration: %w", err)
	}
	if len(desiredValues) == 0 {
		return nil, false, nil
	}

	currentValues, err := managedInstanceJSONMap(managedInstanceCurrentConfigurationForUpdate(current))
	if err != nil {
		return nil, false, fmt.Errorf("project current ManagedInstance configuration: %w", err)
	}
	if managedInstanceJSONEqual(desiredValues, managedInstanceSubsetJSONMap(currentValues, desiredValues)) {
		return nil, false, nil
	}

	return &desired, true, nil
}

func managedInstanceCurrentConfigurationForUpdate(
	current *wlmssdk.ManagedInstanceConfiguration,
) wlmssdk.UpdateManagedInstanceConfigurationDetails {
	if current == nil {
		return wlmssdk.UpdateManagedInstanceConfigurationDetails{}
	}

	return wlmssdk.UpdateManagedInstanceConfigurationDetails{
		DiscoveryInterval: current.DiscoveryInterval,
		DomainSearchPaths: append([]string{}, current.DomainSearchPaths...),
	}
}

func managedInstanceJSONMap(value any) (map[string]any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return managedInstanceJSONBytesToMap(payload)
}

func managedInstanceJSONBytesToMap(payload []byte) (map[string]any, error) {
	if string(payload) == "null" {
		return nil, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func managedInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func managedInstanceSubsetJSONMap(values map[string]any, requested map[string]any) map[string]any {
	if len(values) == 0 || len(requested) == 0 {
		return map[string]any{}
	}

	subset := make(map[string]any, len(requested))
	for key := range requested {
		if value, ok := values[key]; ok {
			subset[key] = value
		}
	}
	return subset
}

func (c managedInstanceRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *wlmsv1beta1.ManagedInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ManagedInstance generated runtime delegate is not configured")
	}
	if err := validateManagedInstanceBinding(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	currentTrackedManagedInstanceID := trackedManagedInstanceID(resource)
	hadTrackedIdentity := currentTrackedManagedInstanceID != ""
	explicitManagedInstanceID := strings.TrimSpace(resource.Spec.Id)
	if explicitManagedInstanceID != "" {
		if err := c.preflightManagedInstanceDirectBind(ctx, resource, explicitManagedInstanceID); err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil && hadTrackedIdentity && managedInstanceNeedsGetBackedRebind(resource) && errors.Is(err, errManagedInstanceRequiresGetBackedResponse) {
		reboundManagedInstanceID := strings.TrimSpace(resource.Status.Id)
		if reboundManagedInstanceID != "" && reboundManagedInstanceID != currentTrackedManagedInstanceID {
			seedManagedInstanceTrackedIdentity(resource, reboundManagedInstanceID)
			return c.delegate.CreateOrUpdate(ctx, resource, req)
		}

		clearManagedInstanceTrackedIdentity(resource)
		response, err = c.delegate.CreateOrUpdate(ctx, resource, req)
		if err != nil || trackedManagedInstanceID(resource) == "" {
			return response, err
		}

		// The rebound pass can only relink through ListManagedInstances because
		// Create=nil. Rerun immediately so mutable-drift evaluation uses a
		// GetManagedInstance payload for the replacement binding.
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}
	if err != nil || hadTrackedIdentity || explicitManagedInstanceID != "" || trackedManagedInstanceID(resource) == "" {
		return response, err
	}

	// With Create=nil, the first untracked reconcile binds from ListManagedInstances.
	// Run one live-ID-backed delegate pass immediately afterwards so status
	// projection and mutable-drift handling use GetManagedInstance instead of
	// the summary payload from the list response.
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c managedInstanceRuntimeClient) Delete(
	_ context.Context,
	_ *wlmsv1beta1.ManagedInstance,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("ManagedInstance generated runtime delegate is not configured")
	}
	return true, nil
}

func (c managedInstanceRuntimeClient) preflightManagedInstanceDirectBind(
	ctx context.Context,
	resource *wlmsv1beta1.ManagedInstance,
	managedInstanceID string,
) error {
	if c.get == nil {
		return fmt.Errorf("ManagedInstance get hook is not configured")
	}
	if _, err := c.get(ctx, wlmssdk.GetManagedInstanceRequest{
		ManagedInstanceId: common.String(managedInstanceID),
	}); err != nil {
		return err
	}
	seedManagedInstanceTrackedIdentity(resource, managedInstanceID)
	return nil
}

func validateManagedInstanceBinding(resource *wlmsv1beta1.ManagedInstance) error {
	if resource == nil {
		return fmt.Errorf("ManagedInstance resource is nil")
	}
	if trackedManagedInstanceID(resource) != "" {
		return nil
	}
	if strings.TrimSpace(resource.Spec.Id) != "" {
		return nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) != "" && strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return nil
	}
	return fmt.Errorf("ManagedInstance bind-existing flow requires spec.id or spec.compartmentId plus spec.displayName")
}

func trackedManagedInstanceID(resource *wlmsv1beta1.ManagedInstance) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func managedInstanceNeedsGetBackedRebind(resource *wlmsv1beta1.ManagedInstance) bool {
	return resource != nil && len(resource.Spec.Configuration) > 0
}

func seedManagedInstanceTrackedIdentity(resource *wlmsv1beta1.ManagedInstance, managedInstanceID string) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(managedInstanceID))
	resource.Status.Id = strings.TrimSpace(managedInstanceID)
}

func clearManagedInstanceTrackedIdentity(resource *wlmsv1beta1.ManagedInstance) {
	if resource == nil {
		return
	}
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
}
