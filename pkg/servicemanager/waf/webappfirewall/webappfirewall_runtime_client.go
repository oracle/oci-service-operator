/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappfirewall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const webAppFirewallBackendTypeLoadBalancer = "LOAD_BALANCER"

type webAppFirewallOCIClient interface {
	CreateWebAppFirewall(context.Context, wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error)
	GetWebAppFirewall(context.Context, wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error)
	ListWebAppFirewalls(context.Context, wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error)
	UpdateWebAppFirewall(context.Context, wafsdk.UpdateWebAppFirewallRequest) (wafsdk.UpdateWebAppFirewallResponse, error)
	DeleteWebAppFirewall(context.Context, wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error)
}

type webAppFirewallDeleteGuardClient struct {
	delegate WebAppFirewallServiceClient
	create   func(context.Context, wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error)
	get      func(context.Context, wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error)
	list     func(context.Context, wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error)
	log      loggerutil.OSOKLogger
}

type webAppFirewallAmbiguousNotFoundError struct {
	err error
}

func (e webAppFirewallAmbiguousNotFoundError) Error() string {
	return fmt.Sprintf("WebAppFirewall delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e webAppFirewallAmbiguousNotFoundError) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

var _ WebAppFirewallServiceClient = (*webAppFirewallDeleteGuardClient)(nil)

func init() {
	registerWebAppFirewallRuntimeHooksMutator(func(_ *WebAppFirewallServiceManager, hooks *WebAppFirewallRuntimeHooks) {
		applyWebAppFirewallRuntimeHooks(hooks)
	})
}

func applyWebAppFirewallRuntimeHooks(hooks *WebAppFirewallRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = webAppFirewallRuntimeSemantics()
	hooks.BuildCreateBody = buildWebAppFirewallCreateBody
	hooks.BuildUpdateBody = buildWebAppFirewallUpdateBody
	hooks.List.Fields = webAppFirewallListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listWebAppFirewallsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleWebAppFirewallDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate WebAppFirewallServiceClient) WebAppFirewallServiceClient {
		return &webAppFirewallDeleteGuardClient{
			delegate: delegate,
			create:   hooks.Create.Call,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newWebAppFirewallServiceClientWithOCIClient(log loggerutil.OSOKLogger, client webAppFirewallOCIClient) WebAppFirewallServiceClient {
	manager := &WebAppFirewallServiceManager{Log: log}
	hooks := newWebAppFirewallRuntimeHooksWithOCIClient(client)
	applyWebAppFirewallRuntimeHooks(&hooks)
	delegate := defaultWebAppFirewallServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*wafv1beta1.WebAppFirewall](
			buildWebAppFirewallGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapWebAppFirewallGeneratedClient(hooks, delegate)
}

func newWebAppFirewallRuntimeHooksWithOCIClient(client webAppFirewallOCIClient) WebAppFirewallRuntimeHooks {
	return WebAppFirewallRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*wafv1beta1.WebAppFirewall]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*wafv1beta1.WebAppFirewall]{},
		StatusHooks:     generatedruntime.StatusHooks[*wafv1beta1.WebAppFirewall]{},
		ParityHooks:     generatedruntime.ParityHooks[*wafv1beta1.WebAppFirewall]{},
		Async:           generatedruntime.AsyncHooks[*wafv1beta1.WebAppFirewall]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*wafv1beta1.WebAppFirewall]{},
		Create: runtimeOperationHooks[wafsdk.CreateWebAppFirewallRequest, wafsdk.CreateWebAppFirewallResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateWebAppFirewallDetails", RequestName: "CreateWebAppFirewallDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request wafsdk.CreateWebAppFirewallRequest) (wafsdk.CreateWebAppFirewallResponse, error) {
				if client == nil {
					return wafsdk.CreateWebAppFirewallResponse{}, fmt.Errorf("WebAppFirewall OCI client is nil")
				}
				return client.CreateWebAppFirewall(ctx, request)
			},
		},
		Get: runtimeOperationHooks[wafsdk.GetWebAppFirewallRequest, wafsdk.GetWebAppFirewallResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "WebAppFirewallId", RequestName: "webAppFirewallId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request wafsdk.GetWebAppFirewallRequest) (wafsdk.GetWebAppFirewallResponse, error) {
				if client == nil {
					return wafsdk.GetWebAppFirewallResponse{}, fmt.Errorf("WebAppFirewall OCI client is nil")
				}
				return client.GetWebAppFirewall(ctx, request)
			},
		},
		List: runtimeOperationHooks[wafsdk.ListWebAppFirewallsRequest, wafsdk.ListWebAppFirewallsResponse]{
			Fields: webAppFirewallListFields(),
			Call: func(ctx context.Context, request wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error) {
				if client == nil {
					return wafsdk.ListWebAppFirewallsResponse{}, fmt.Errorf("WebAppFirewall OCI client is nil")
				}
				return client.ListWebAppFirewalls(ctx, request)
			},
		},
		Update: runtimeOperationHooks[wafsdk.UpdateWebAppFirewallRequest, wafsdk.UpdateWebAppFirewallResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "WebAppFirewallId", RequestName: "webAppFirewallId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateWebAppFirewallDetails", RequestName: "UpdateWebAppFirewallDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request wafsdk.UpdateWebAppFirewallRequest) (wafsdk.UpdateWebAppFirewallResponse, error) {
				if client == nil {
					return wafsdk.UpdateWebAppFirewallResponse{}, fmt.Errorf("WebAppFirewall OCI client is nil")
				}
				return client.UpdateWebAppFirewall(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[wafsdk.DeleteWebAppFirewallRequest, wafsdk.DeleteWebAppFirewallResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "WebAppFirewallId", RequestName: "webAppFirewallId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request wafsdk.DeleteWebAppFirewallRequest) (wafsdk.DeleteWebAppFirewallResponse, error) {
				if client == nil {
					return wafsdk.DeleteWebAppFirewallResponse{}, fmt.Errorf("WebAppFirewall OCI client is nil")
				}
				return client.DeleteWebAppFirewall(ctx, request)
			},
		},
		WrapGeneratedClient: []func(WebAppFirewallServiceClient) WebAppFirewallServiceClient{},
	}
}

func webAppFirewallRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "waf",
		FormalSlug:        "webappfirewall",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(wafsdk.WebAppFirewallLifecycleStateCreating)},
			UpdatingStates:     []string{string(wafsdk.WebAppFirewallLifecycleStateUpdating)},
			ActiveStates:       []string{string(wafsdk.WebAppFirewallLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(wafsdk.WebAppFirewallLifecycleStateDeleting)},
			TerminalStates: []string{string(wafsdk.WebAppFirewallLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "loadBalancerId", "backendType"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "webAppFirewallPolicyId", "freeformTags", "definedTags", "systemTags"},
			Mutable:         []string{"displayName", "webAppFirewallPolicyId", "freeformTags", "definedTags", "systemTags"},
			ForceNew:        []string{"compartmentId", "backendType", "loadBalancerId"},
			ConflictsWith:   map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
	}
}

func webAppFirewallListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func buildWebAppFirewallCreateBody(
	_ context.Context,
	resource *wafv1beta1.WebAppFirewall,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("WebAppFirewall resource is nil")
	}
	spec := resource.Spec
	if err := validateWebAppFirewallCreateSpec(spec); err != nil {
		return nil, err
	}

	body := wafsdk.CreateWebAppFirewallLoadBalancerDetails{
		CompartmentId:          common.String(spec.CompartmentId),
		WebAppFirewallPolicyId: common.String(spec.WebAppFirewallPolicyId),
		LoadBalancerId:         common.String(spec.LoadBalancerId),
	}
	applyWebAppFirewallCreateOptionalFields(&body, spec)
	return body, nil
}

func validateWebAppFirewallCreateSpec(spec wafv1beta1.WebAppFirewallSpec) error {
	backendType := normalizeWebAppFirewallBackendType(spec.BackendType)
	if backendType == "" && strings.TrimSpace(spec.LoadBalancerId) != "" {
		backendType = webAppFirewallBackendTypeLoadBalancer
	}
	if backendType != webAppFirewallBackendTypeLoadBalancer {
		return fmt.Errorf("WebAppFirewall backendType %q is not supported by this runtime", spec.BackendType)
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("WebAppFirewall create requires spec.compartmentId")
	}
	if strings.TrimSpace(spec.WebAppFirewallPolicyId) == "" {
		return fmt.Errorf("WebAppFirewall create requires spec.webAppFirewallPolicyId")
	}
	if strings.TrimSpace(spec.LoadBalancerId) == "" {
		return fmt.Errorf("WebAppFirewall create requires spec.loadBalancerId for LOAD_BALANCER backend")
	}
	return nil
}

func applyWebAppFirewallCreateOptionalFields(
	body *wafsdk.CreateWebAppFirewallLoadBalancerDetails,
	spec wafv1beta1.WebAppFirewallSpec,
) {
	if body == nil {
		return
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(spec.DisplayName)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneWebAppFirewallFreeformTags(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = webAppFirewallDefinedTagsFromSpec(spec.DefinedTags)
	}
	if spec.SystemTags != nil {
		body.SystemTags = webAppFirewallDefinedTagsFromSpec(spec.SystemTags)
	}
}

func buildWebAppFirewallUpdateBody(
	_ context.Context,
	resource *wafv1beta1.WebAppFirewall,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("WebAppFirewall resource is nil")
	}
	current, err := webAppFirewallObservedFromResponse(currentResponse)
	if err != nil {
		return nil, false, err
	}

	spec := resource.Spec
	builder := webAppFirewallUpdateBodyBuilder{}
	builder.applyString(spec.DisplayName, current.DisplayName, &builder.details.DisplayName)
	builder.applyString(spec.WebAppFirewallPolicyId, current.WebAppFirewallPolicyId, &builder.details.WebAppFirewallPolicyId)
	builder.applyFreeformTags(spec.FreeformTags, current.FreeformTags)
	builder.applyDefinedTags(spec.DefinedTags, current.DefinedTags, &builder.details.DefinedTags)
	builder.applyDefinedTags(spec.SystemTags, current.SystemTags, &builder.details.SystemTags)

	return builder.details, builder.updateNeeded, nil
}

type webAppFirewallUpdateBodyBuilder struct {
	details      wafsdk.UpdateWebAppFirewallDetails
	updateNeeded bool
}

func (b *webAppFirewallUpdateBodyBuilder) applyString(desired string, current string, target **string) {
	if strings.TrimSpace(desired) == "" || desired == current {
		return
	}
	*target = common.String(desired)
	b.updateNeeded = true
}

func (b *webAppFirewallUpdateBodyBuilder) applyFreeformTags(
	desired map[string]string,
	current map[string]string,
) {
	if desired == nil || reflect.DeepEqual(desired, current) {
		return
	}
	b.details.FreeformTags = cloneWebAppFirewallFreeformTags(desired)
	b.updateNeeded = true
}

func (b *webAppFirewallUpdateBodyBuilder) applyDefinedTags(
	desiredSpec map[string]shared.MapValue,
	current map[string]map[string]interface{},
	target *map[string]map[string]interface{},
) {
	if desiredSpec == nil {
		return
	}
	desired := webAppFirewallDefinedTagsFromSpec(desiredSpec)
	if reflect.DeepEqual(desired, current) {
		return
	}
	*target = desired
	b.updateNeeded = true
}

type webAppFirewallObserved struct {
	Id                     string                            `json:"id,omitempty"`
	DisplayName            string                            `json:"displayName,omitempty"`
	CompartmentId          string                            `json:"compartmentId,omitempty"`
	WebAppFirewallPolicyId string                            `json:"webAppFirewallPolicyId,omitempty"`
	LifecycleState         string                            `json:"lifecycleState,omitempty"`
	FreeformTags           map[string]string                 `json:"freeformTags,omitempty"`
	DefinedTags            map[string]map[string]interface{} `json:"definedTags,omitempty"`
	SystemTags             map[string]map[string]interface{} `json:"systemTags,omitempty"`
	BackendType            string                            `json:"backendType,omitempty"`
	LoadBalancerId         string                            `json:"loadBalancerId,omitempty"`
}

func webAppFirewallObservedFromResponse(response any) (webAppFirewallObserved, error) {
	body := response
	switch typed := response.(type) {
	case wafsdk.CreateWebAppFirewallResponse:
		body = typed.WebAppFirewall
	case wafsdk.GetWebAppFirewallResponse:
		body = typed.WebAppFirewall
	case wafsdk.WebAppFirewall:
		body = typed
	}
	if body == nil {
		return webAppFirewallObserved{}, nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return webAppFirewallObserved{}, fmt.Errorf("marshal WebAppFirewall OCI response: %w", err)
	}
	var observed webAppFirewallObserved
	if err := json.Unmarshal(payload, &observed); err != nil {
		return webAppFirewallObserved{}, fmt.Errorf("decode WebAppFirewall OCI response: %w", err)
	}
	return observed, nil
}

func listWebAppFirewallsAllPages(
	call func(context.Context, wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error),
) func(context.Context, wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error) {
	return func(ctx context.Context, request wafsdk.ListWebAppFirewallsRequest) (wafsdk.ListWebAppFirewallsResponse, error) {
		var combined wafsdk.ListWebAppFirewallsResponse
		seenPages := map[string]struct{}{}
		for {
			response, err := call(ctx, request)
			if err != nil {
				return wafsdk.ListWebAppFirewallsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)

			nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return wafsdk.ListWebAppFirewallsResponse{}, fmt.Errorf("WebAppFirewall list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			combined.OpcNextPage = response.OpcNextPage
			request.Page = response.OpcNextPage
		}
	}
}

func handleWebAppFirewallDeleteError(resource *wafv1beta1.WebAppFirewall, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return webAppFirewallAmbiguousNotFoundError{err: err}
	}
	return err
}

func (c *webAppFirewallDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewall,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppFirewall runtime client is not configured")
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppFirewall resource is nil")
	}
	if currentWebAppFirewallID(resource) == "" {
		if response, err, handled := c.createOrBindInitialResource(ctx, resource, req); handled {
			return response, err
		}
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *webAppFirewallDeleteGuardClient) createOrBindInitialResource(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewall,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error, bool) {
	existing, err := c.lookupExistingWebAppFirewall(ctx, resource)
	if err != nil {
		response, err := c.failCreateOrUpdate(resource, err)
		return response, err, true
	}
	if existing.Id != "" {
		projectWebAppFirewallObservedStatus(resource, existing)
		response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
		return response, err, true
	}

	createResponse, err := c.createWebAppFirewall(ctx, resource)
	if err != nil {
		response, err := c.failCreateOrUpdate(resource, err)
		return response, err, true
	}
	observed, err := webAppFirewallObservedFromResponse(createResponse)
	if err != nil {
		response, err := c.failCreateOrUpdate(resource, err)
		return response, err, true
	}
	if strings.TrimSpace(observed.Id) == "" {
		response, err := c.failCreateOrUpdate(resource, fmt.Errorf("WebAppFirewall create response did not include an OCI identifier"))
		return response, err, true
	}
	projectWebAppFirewallObservedStatus(resource, observed)
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, createResponse)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	return response, err, true
}

func (c *webAppFirewallDeleteGuardClient) lookupExistingWebAppFirewall(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewall,
) (webAppFirewallObserved, error) {
	if c.list == nil {
		return webAppFirewallObserved{}, nil
	}
	response, err := c.list(ctx, wafsdk.ListWebAppFirewallsRequest{
		CompartmentId: common.String(resource.Spec.CompartmentId),
	})
	if err != nil {
		return webAppFirewallObserved{}, err
	}
	var matches []webAppFirewallObserved
	for _, item := range response.Items {
		observed, err := webAppFirewallObservedFromResponse(item)
		if err != nil {
			return webAppFirewallObserved{}, err
		}
		if webAppFirewallObservedMatchesSpec(observed, resource.Spec) {
			matches = append(matches, observed)
		}
	}
	switch len(matches) {
	case 0:
		return webAppFirewallObserved{}, nil
	case 1:
		return matches[0], nil
	default:
		return webAppFirewallObserved{}, fmt.Errorf("WebAppFirewall list response returned multiple matching resources for compartmentId %q and loadBalancerId %q", resource.Spec.CompartmentId, resource.Spec.LoadBalancerId)
	}
}

func (c *webAppFirewallDeleteGuardClient) createWebAppFirewall(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewall,
) (wafsdk.CreateWebAppFirewallResponse, error) {
	if c.create == nil {
		return wafsdk.CreateWebAppFirewallResponse{}, fmt.Errorf("WebAppFirewall create call is not configured")
	}
	body, err := buildWebAppFirewallCreateBody(ctx, resource, resource.Namespace)
	if err != nil {
		return wafsdk.CreateWebAppFirewallResponse{}, err
	}
	createDetails, ok := body.(wafsdk.CreateWebAppFirewallDetails)
	if !ok {
		return wafsdk.CreateWebAppFirewallResponse{}, fmt.Errorf("WebAppFirewall create body type %T does not implement CreateWebAppFirewallDetails", body)
	}
	return c.create(ctx, wafsdk.CreateWebAppFirewallRequest{
		CreateWebAppFirewallDetails: createDetails,
		OpcRetryToken:               common.String(webAppFirewallRetryToken(resource)),
	})
}

func (c *webAppFirewallDeleteGuardClient) failCreateOrUpdate(
	resource *wafv1beta1.WebAppFirewall,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil && err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		resource.Status.OsokStatus.Message = err.Error()
		resource.Status.OsokStatus.Reason = string(shared.Failed)
		now := metav1.Now()
		resource.Status.OsokStatus.UpdatedAt = &now
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
			resource.Status.OsokStatus,
			shared.Failed,
			v1.ConditionFalse,
			"",
			err.Error(),
			c.log,
		)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *webAppFirewallDeleteGuardClient) Delete(ctx context.Context, resource *wafv1beta1.WebAppFirewall) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("WebAppFirewall runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *webAppFirewallDeleteGuardClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewall,
) error {
	currentID := currentWebAppFirewallID(resource)
	if currentID == "" || c.get == nil {
		return nil
	}
	_, err := c.get(ctx, wafsdk.GetWebAppFirewallRequest{WebAppFirewallId: common.String(currentID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("WebAppFirewall delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func currentWebAppFirewallID(resource *wafv1beta1.WebAppFirewall) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func projectWebAppFirewallObservedStatus(resource *wafv1beta1.WebAppFirewall, observed webAppFirewallObserved) {
	if resource == nil {
		return
	}
	projectWebAppFirewallObservedIdentity(resource, observed)
	projectWebAppFirewallObservedCore(resource, observed)
	projectWebAppFirewallObservedTags(resource, observed)
	projectWebAppFirewallObservedBackend(resource, observed)
}

func projectWebAppFirewallObservedIdentity(resource *wafv1beta1.WebAppFirewall, observed webAppFirewallObserved) {
	if observed.Id != "" {
		resource.Status.Id = observed.Id
		resource.Status.OsokStatus.Ocid = shared.OCID(observed.Id)
	}
}

func projectWebAppFirewallObservedCore(resource *wafv1beta1.WebAppFirewall, observed webAppFirewallObserved) {
	if observed.DisplayName != "" {
		resource.Status.DisplayName = observed.DisplayName
	}
	if observed.CompartmentId != "" {
		resource.Status.CompartmentId = observed.CompartmentId
	}
	if observed.WebAppFirewallPolicyId != "" {
		resource.Status.WebAppFirewallPolicyId = observed.WebAppFirewallPolicyId
	}
	if observed.LifecycleState != "" {
		resource.Status.LifecycleState = observed.LifecycleState
	}
}

func projectWebAppFirewallObservedTags(resource *wafv1beta1.WebAppFirewall, observed webAppFirewallObserved) {
	if observed.FreeformTags != nil {
		resource.Status.FreeformTags = cloneWebAppFirewallFreeformTags(observed.FreeformTags)
	}
	if observed.DefinedTags != nil {
		resource.Status.DefinedTags = webAppFirewallSharedTagsFromSDK(observed.DefinedTags)
	}
	if observed.SystemTags != nil {
		resource.Status.SystemTags = webAppFirewallSharedTagsFromSDK(observed.SystemTags)
	}
}

func projectWebAppFirewallObservedBackend(resource *wafv1beta1.WebAppFirewall, observed webAppFirewallObserved) {
	if observed.BackendType != "" {
		resource.Status.BackendType = observed.BackendType
	}
	if observed.LoadBalancerId != "" {
		resource.Status.LoadBalancerId = observed.LoadBalancerId
	}
}

func webAppFirewallObservedMatchesSpec(observed webAppFirewallObserved, spec wafv1beta1.WebAppFirewallSpec) bool {
	if strings.TrimSpace(observed.Id) == "" {
		return false
	}
	if observed.CompartmentId != spec.CompartmentId {
		return false
	}
	if observed.LoadBalancerId != spec.LoadBalancerId {
		return false
	}
	backendType := normalizeWebAppFirewallBackendType(spec.BackendType)
	if backendType != "" && normalizeWebAppFirewallBackendType(observed.BackendType) != backendType {
		return false
	}
	return true
}

func webAppFirewallRetryToken(resource *wafv1beta1.WebAppFirewall) string {
	if resource == nil {
		return ""
	}
	identity := strings.Join([]string{
		resource.Namespace,
		resource.Name,
		string(resource.UID),
		resource.Spec.CompartmentId,
		resource.Spec.LoadBalancerId,
	}, "/")
	sum := sha256.Sum256([]byte(identity))
	return "osok-webappfirewall-" + hex.EncodeToString(sum[:16])
}

func normalizeWebAppFirewallBackendType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func cloneWebAppFirewallFreeformTags(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func webAppFirewallDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}

func webAppFirewallSharedTagsFromSDK(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		children := make(shared.MapValue, len(values))
		for key, value := range values {
			children[key] = fmt.Sprint(value)
		}
		converted[namespace] = children
	}
	return converted
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
