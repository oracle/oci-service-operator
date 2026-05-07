/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package wlsdomain

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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

type wlsDomainOCIClient interface {
	GetWlsDomain(context.Context, wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error)
	ListWlsDomains(context.Context, wlmssdk.ListWlsDomainsRequest) (wlmssdk.ListWlsDomainsResponse, error)
	UpdateWlsDomain(context.Context, wlmssdk.UpdateWlsDomainRequest) (wlmssdk.UpdateWlsDomainResponse, error)
	DeleteWlsDomain(context.Context, wlmssdk.DeleteWlsDomainRequest) (wlmssdk.DeleteWlsDomainResponse, error)
}

type wlsDomainRuntimeClient struct {
	delegate WlsDomainServiceClient
}

type wlsDomainListCall func(context.Context, wlmssdk.ListWlsDomainsRequest) (wlmssdk.ListWlsDomainsResponse, error)

func init() {
	registerWlsDomainRuntimeHooksMutator(func(_ *WlsDomainServiceManager, hooks *WlsDomainRuntimeHooks) {
		applyWlsDomainRuntimeHooks(hooks)
	})
}

func applyWlsDomainRuntimeHooks(hooks *WlsDomainRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedWlsDomainRuntimeSemantics()
	hooks.List.Fields = wlsDomainListFields()
	hooks.List.Call = wrapWlsDomainListCall(hooks.List.Call)
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *wlmsv1beta1.WlsDomain,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildWlsDomainUpdateBody(resource, currentResponse)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate WlsDomainServiceClient) WlsDomainServiceClient {
		return wlsDomainRuntimeClient{delegate: delegate}
	})
}

func newWlsDomainServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client wlsDomainOCIClient,
) WlsDomainServiceClient {
	hooks := newWlsDomainRuntimeHooksWithOCIClient(client)
	applyWlsDomainRuntimeHooks(&hooks)
	delegate := defaultWlsDomainServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*wlmsv1beta1.WlsDomain](
			buildWlsDomainGeneratedRuntimeConfig(&WlsDomainServiceManager{Log: log}, hooks),
		),
	}
	return wrapWlsDomainGeneratedClient(hooks, delegate)
}

func newWlsDomainRuntimeConfig(
	log loggerutil.OSOKLogger,
	client wlsDomainOCIClient,
) generatedruntime.Config[*wlmsv1beta1.WlsDomain] {
	hooks := newWlsDomainRuntimeHooksWithOCIClient(client)
	applyWlsDomainRuntimeHooks(&hooks)
	return buildWlsDomainGeneratedRuntimeConfig(&WlsDomainServiceManager{Log: log}, hooks)
}

func newWlsDomainRuntimeHooksWithOCIClient(client wlsDomainOCIClient) WlsDomainRuntimeHooks {
	return WlsDomainRuntimeHooks{
		Semantics:       reviewedWlsDomainRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*wlmsv1beta1.WlsDomain]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*wlmsv1beta1.WlsDomain]{},
		StatusHooks:     generatedruntime.StatusHooks[*wlmsv1beta1.WlsDomain]{},
		ParityHooks:     generatedruntime.ParityHooks[*wlmsv1beta1.WlsDomain]{},
		Async:           generatedruntime.AsyncHooks[*wlmsv1beta1.WlsDomain]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*wlmsv1beta1.WlsDomain]{},
		Get: runtimeOperationHooks[wlmssdk.GetWlsDomainRequest, wlmssdk.GetWlsDomainResponse]{
			Fields: wlsDomainGetFields(),
			Call: func(ctx context.Context, request wlmssdk.GetWlsDomainRequest) (wlmssdk.GetWlsDomainResponse, error) {
				return client.GetWlsDomain(ctx, request)
			},
		},
		List: runtimeOperationHooks[wlmssdk.ListWlsDomainsRequest, wlmssdk.ListWlsDomainsResponse]{
			Fields: wlsDomainListFields(),
			Call: func(ctx context.Context, request wlmssdk.ListWlsDomainsRequest) (wlmssdk.ListWlsDomainsResponse, error) {
				return client.ListWlsDomains(ctx, request)
			},
		},
		Update: runtimeOperationHooks[wlmssdk.UpdateWlsDomainRequest, wlmssdk.UpdateWlsDomainResponse]{
			Fields: wlsDomainUpdateFields(),
			Call: func(ctx context.Context, request wlmssdk.UpdateWlsDomainRequest) (wlmssdk.UpdateWlsDomainResponse, error) {
				return client.UpdateWlsDomain(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[wlmssdk.DeleteWlsDomainRequest, wlmssdk.DeleteWlsDomainResponse]{
			Fields: wlsDomainDeleteFields(),
			Call: func(ctx context.Context, request wlmssdk.DeleteWlsDomainRequest) (wlmssdk.DeleteWlsDomainResponse, error) {
				return client.DeleteWlsDomain(ctx, request)
			},
		},
		WrapGeneratedClient: []func(WlsDomainServiceClient) WlsDomainServiceClient{},
	}
}

func reviewedWlsDomainRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newWlsDomainRuntimeSemantics()
	semantics.AuxiliaryOperations = nil
	return semantics
}

func wlsDomainGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "WlsDomainId", RequestName: "wlsDomainId", Contribution: "path", PreferResourceID: true},
	}
}

func wlsDomainListFields() []generatedruntime.RequestField {
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
			FieldName:    "MiddlewareType",
			RequestName:  "middlewareType",
			Contribution: "query",
			LookupPaths:  []string{"status.middlewareType", "spec.middlewareType", "middlewareType"},
		},
		{
			FieldName:    "WeblogicVersion",
			RequestName:  "weblogicVersion",
			Contribution: "query",
			LookupPaths:  []string{"status.weblogicVersion", "spec.weblogicVersion", "weblogicVersion"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func wlsDomainUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "WlsDomainId", RequestName: "wlsDomainId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateWlsDomainDetails", RequestName: "UpdateWlsDomainDetails", Contribution: "body"},
	}
}

func wlsDomainDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "WlsDomainId", RequestName: "wlsDomainId", Contribution: "path", PreferResourceID: true},
	}
}

func wrapWlsDomainListCall(call wlsDomainListCall) wlsDomainListCall {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request wlmssdk.ListWlsDomainsRequest) (wlmssdk.ListWlsDomainsResponse, error) {
		return listWlsDomainPages(ctx, call, request)
	}
}

func listWlsDomainPages(
	ctx context.Context,
	call wlsDomainListCall,
	request wlmssdk.ListWlsDomainsRequest,
) (wlmssdk.ListWlsDomainsResponse, error) {
	if call == nil {
		return wlmssdk.ListWlsDomainsResponse{}, fmt.Errorf("WlsDomain list call is not configured")
	}

	var combined wlmssdk.ListWlsDomainsResponse
	seenPages := map[string]struct{}{}
	for {
		pageToken := ""
		if request.Page != nil {
			pageToken = strings.TrimSpace(*request.Page)
		}
		if _, seen := seenPages[pageToken]; seen {
			return wlmssdk.ListWlsDomainsResponse{}, fmt.Errorf("WlsDomain list pagination repeated page %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return wlmssdk.ListWlsDomainsResponse{}, err
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

func buildWlsDomainUpdateBody(
	resource *wlmsv1beta1.WlsDomain,
	currentResponse any,
) (wlmssdk.UpdateWlsDomainDetails, bool, error) {
	if resource == nil {
		return wlmssdk.UpdateWlsDomainDetails{}, false, fmt.Errorf("WlsDomain resource is nil")
	}

	current, err := wlsDomainFromResponse(currentResponse)
	if err != nil {
		return wlmssdk.UpdateWlsDomainDetails{}, false, err
	}

	details := wlmssdk.UpdateWlsDomainDetails{}
	updateNeeded := false

	if desired, ok, err := wlsDomainDesiredConfigurationUpdate(resource.Spec.Configuration, current.Configuration); err != nil {
		return wlmssdk.UpdateWlsDomainDetails{}, false, err
	} else if ok {
		details.Configuration = desired
		updateNeeded = true
	}
	if desired, ok := wlsDomainDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := wlsDomainDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func wlsDomainFromResponse(currentResponse any) (wlmssdk.WlsDomain, error) {
	switch response := currentResponse.(type) {
	case wlmssdk.GetWlsDomainResponse:
		return response.WlsDomain, nil
	case *wlmssdk.GetWlsDomainResponse:
		if response == nil {
			break
		}
		return response.WlsDomain, nil
	case wlmssdk.UpdateWlsDomainResponse:
		return response.WlsDomain, nil
	case *wlmssdk.UpdateWlsDomainResponse:
		if response == nil {
			break
		}
		return response.WlsDomain, nil
	case wlmssdk.WlsDomain:
		return response, nil
	case *wlmssdk.WlsDomain:
		if response == nil {
			break
		}
		return *response, nil
	}
	return wlmssdk.WlsDomain{}, fmt.Errorf("current WlsDomain response does not expose a WlsDomain body")
}

func wlsDomainDesiredConfigurationUpdate(
	spec map[string]shared.JSONValue,
	current *wlmssdk.WlsDomainConfiguration,
) (*wlmssdk.UpdateWlsDomainConfigurationDetails, bool, error) {
	if spec == nil || len(spec) == 0 {
		return nil, false, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, false, fmt.Errorf("marshal WlsDomain configuration spec: %w", err)
	}

	var desired wlmssdk.UpdateWlsDomainConfigurationDetails
	if err := json.Unmarshal(payload, &desired); err != nil {
		return nil, false, fmt.Errorf("decode WlsDomain configuration update body: %w", err)
	}

	desiredValues, err := wlsDomainJSONMap(desired)
	if err != nil {
		return nil, false, fmt.Errorf("project desired WlsDomain configuration: %w", err)
	}
	currentValues, err := wlsDomainJSONMap(wlsDomainCurrentConfigurationForUpdate(current))
	if err != nil {
		return nil, false, fmt.Errorf("project current WlsDomain configuration: %w", err)
	}
	if wlsDomainJSONEqual(desiredValues, currentValues) {
		return nil, false, nil
	}
	return &desired, true, nil
}

func wlsDomainCurrentConfigurationForUpdate(
	current *wlmssdk.WlsDomainConfiguration,
) wlmssdk.UpdateWlsDomainConfigurationDetails {
	if current == nil {
		return wlmssdk.UpdateWlsDomainConfigurationDetails{}
	}

	return wlmssdk.UpdateWlsDomainConfigurationDetails{
		IsPatchEnabled:               current.IsPatchEnabled,
		IsRollbackOnFailure:          current.IsRollbackOnFailure,
		ServersShutdownTimeout:       current.ServersShutdownTimeout,
		AdminServerControlMode:       current.AdminServerControlMode,
		ManagedServerControlMode:     current.ManagedServerControlMode,
		AdminServerStartScriptPath:   current.AdminServerStartScriptPath,
		AdminServerStopScriptPath:    current.AdminServerStopScriptPath,
		ManagedServerStartScriptPath: current.ManagedServerStartScriptPath,
		ManagedServerStopScriptPath:  current.ManagedServerStopScriptPath,
	}
}

func wlsDomainDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func wlsDomainDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := wlsDomainDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if wlsDomainJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func wlsDomainDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func wlsDomainJSONMap(value any) (map[string]any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if string(payload) == "null" {
		return nil, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func wlsDomainJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func (c wlsDomainRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *wlmsv1beta1.WlsDomain,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WlsDomain generated runtime delegate is not configured")
	}
	if err := validateWlsDomainBinding(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	hadTrackedIdentity := trackedWlsDomainID(resource) != ""
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || hadTrackedIdentity || strings.TrimSpace(resource.Spec.Id) != "" || trackedWlsDomainID(resource) == "" {
		return response, err
	}

	// With Create=nil, the first untracked reconcile binds from ListWlsDomains.
	// Run one live-ID-backed delegate pass immediately afterwards so status
	// projection and mutable-drift handling use GetWlsDomain instead of the
	// summary payload from the list response.
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c wlsDomainRuntimeClient) Delete(
	ctx context.Context,
	resource *wlmsv1beta1.WlsDomain,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("WlsDomain generated runtime delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func validateWlsDomainBinding(resource *wlmsv1beta1.WlsDomain) error {
	if resource == nil {
		return fmt.Errorf("WlsDomain resource is nil")
	}
	if trackedWlsDomainID(resource) != "" {
		return nil
	}
	if strings.TrimSpace(resource.Spec.Id) != "" {
		return nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) != "" && strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return nil
	}
	return fmt.Errorf("WlsDomain manage-existing flow requires spec.id or spec.compartmentId plus spec.displayName")
}

func trackedWlsDomainID(resource *wlmsv1beta1.WlsDomain) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}
