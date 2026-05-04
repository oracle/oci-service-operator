/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networkaddresslist

import (
	"context"
	"errors"
	"fmt"
	"maps"
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

const (
	networkAddressListTypeAddresses    = "ADDRESSES"
	networkAddressListTypeVcnAddresses = "VCN_ADDRESSES"
)

type networkAddressListListCall func(context.Context, wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error)
type networkAddressListGetCall func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error)

type networkAddressListRuntimeClient struct {
	delegate    NetworkAddressListServiceClient
	get         networkAddressListGetCall
	confirmRead func(context.Context, *wafv1beta1.NetworkAddressList, string) (any, error)
}

type networkAddressListRequestBodyContextKey struct{}
type networkAddressListResourceContextKey struct{}

type networkAddressListRequestBodies struct {
	create wafsdk.CreateNetworkAddressListDetails
	update wafsdk.UpdateNetworkAddressListDetails
}

type networkAddressListObserved struct {
	Id               string
	DisplayName      string
	CompartmentId    string
	LifecycleState   string
	LifecycleDetails string
	FreeformTags     map[string]string
	DefinedTags      map[string]map[string]interface{}
	SystemTags       map[string]map[string]interface{}
	Type             string
	Addresses        []string
	VcnAddresses     []wafv1beta1.NetworkAddressListVcnAddress
}

type networkAddressListAuthShapedConfirmRead struct {
	err error
}

type networkAddressListNoMatchConfirmRead struct {
	message string
}

func (e networkAddressListAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("NetworkAddressList delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e networkAddressListAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func (e networkAddressListNoMatchConfirmRead) Error() string {
	return e.message
}

var _ NetworkAddressListServiceClient = (*networkAddressListRuntimeClient)(nil)

func init() {
	registerNetworkAddressListRuntimeHooksMutator(func(_ *NetworkAddressListServiceManager, hooks *NetworkAddressListRuntimeHooks) {
		applyNetworkAddressListRuntimeHooks(hooks)
	})
}

func applyNetworkAddressListRuntimeHooks(hooks *NetworkAddressListRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = networkAddressListRuntimeSemantics()
	hooks.BuildCreateBody = buildNetworkAddressListCreateBodyHook
	hooks.BuildUpdateBody = buildNetworkAddressListUpdateBodyHook
	hooks.Create.Fields = []generatedruntime.RequestField{{FieldName: "RequestMetadata", Contribution: "header"}}
	hooks.Create.Call = injectNetworkAddressListCreateBody(hooks.Create.Call)
	hooks.Update.Fields = []generatedruntime.RequestField{
		{FieldName: "NetworkAddressListId", RequestName: "networkAddressListId", Contribution: "path", PreferResourceID: true},
	}
	hooks.Update.Call = injectNetworkAddressListUpdateBody(hooks.Update.Call)
	hooks.List.Fields = networkAddressListListFields()
	hooks.List.Call = listNetworkAddressListsAllPages(hooks.List.Call)
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *wafv1beta1.NetworkAddressList, currentID string) (any, error) {
		return confirmNetworkAddressListDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleNetworkAddressListDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyNetworkAddressListDeleteOutcome
	hooks.ParityHooks.NormalizeDesiredState = normalizeNetworkAddressListDesiredState
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate NetworkAddressListServiceClient) NetworkAddressListServiceClient {
		return &networkAddressListRuntimeClient{
			delegate:    delegate,
			get:         hooks.Get.Call,
			confirmRead: hooks.DeleteHooks.ConfirmRead,
		}
	})
}

func buildNetworkAddressListCreateBodyHook(
	ctx context.Context,
	resource *wafv1beta1.NetworkAddressList,
	namespace string,
) (any, error) {
	body, err := buildNetworkAddressListCreateBody(ctx, resource, namespace)
	if err != nil {
		return nil, err
	}
	typed, ok := body.(wafsdk.CreateNetworkAddressListDetails)
	if !ok {
		return nil, fmt.Errorf("NetworkAddressList create body has unsupported type %T", body)
	}
	networkAddressListRequestBodiesFromContext(ctx).create = typed
	return body, nil
}

func buildNetworkAddressListUpdateBodyHook(
	ctx context.Context,
	resource *wafv1beta1.NetworkAddressList,
	namespace string,
	currentResponse any,
) (any, bool, error) {
	body, updateNeeded, err := buildNetworkAddressListUpdateBody(ctx, resource, namespace, currentResponse)
	if err != nil || !updateNeeded {
		return body, updateNeeded, err
	}
	typed, ok := body.(wafsdk.UpdateNetworkAddressListDetails)
	if !ok {
		return nil, false, fmt.Errorf("NetworkAddressList update body has unsupported type %T", body)
	}
	networkAddressListRequestBodiesFromContext(ctx).update = typed
	return body, updateNeeded, nil
}

func injectNetworkAddressListCreateBody(
	call func(context.Context, wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error),
) func(context.Context, wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error) {
	return func(ctx context.Context, request wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error) {
		if call == nil {
			return wafsdk.CreateNetworkAddressListResponse{}, fmt.Errorf("NetworkAddressList create hook is not configured")
		}
		body := networkAddressListRequestBodiesFromContext(ctx).create
		if body == nil {
			return wafsdk.CreateNetworkAddressListResponse{}, fmt.Errorf("NetworkAddressList create body was not prepared")
		}
		request.CreateNetworkAddressListDetails = body
		return call(ctx, request)
	}
}

func injectNetworkAddressListUpdateBody(
	call func(context.Context, wafsdk.UpdateNetworkAddressListRequest) (wafsdk.UpdateNetworkAddressListResponse, error),
) func(context.Context, wafsdk.UpdateNetworkAddressListRequest) (wafsdk.UpdateNetworkAddressListResponse, error) {
	return func(ctx context.Context, request wafsdk.UpdateNetworkAddressListRequest) (wafsdk.UpdateNetworkAddressListResponse, error) {
		if call == nil {
			return wafsdk.UpdateNetworkAddressListResponse{}, fmt.Errorf("NetworkAddressList update hook is not configured")
		}
		body := networkAddressListRequestBodiesFromContext(ctx).update
		if body == nil {
			return wafsdk.UpdateNetworkAddressListResponse{}, fmt.Errorf("NetworkAddressList update body was not prepared")
		}
		request.UpdateNetworkAddressListDetails = body
		return call(ctx, request)
	}
}

func networkAddressListRequestBodiesFromContext(ctx context.Context) *networkAddressListRequestBodies {
	if ctx != nil {
		if bodies, ok := ctx.Value(networkAddressListRequestBodyContextKey{}).(*networkAddressListRequestBodies); ok && bodies != nil {
			return bodies
		}
	}
	return &networkAddressListRequestBodies{}
}

func contextWithNetworkAddressListRequestBodies(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(networkAddressListRequestBodyContextKey{}).(*networkAddressListRequestBodies); ok {
		return ctx
	}
	return context.WithValue(ctx, networkAddressListRequestBodyContextKey{}, &networkAddressListRequestBodies{})
}

func contextWithNetworkAddressListResource(ctx context.Context, resource *wafv1beta1.NetworkAddressList) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if resource == nil {
		return ctx
	}
	return context.WithValue(ctx, networkAddressListResourceContextKey{}, resource)
}

func networkAddressListResourceFromContext(ctx context.Context) *wafv1beta1.NetworkAddressList {
	if ctx == nil {
		return nil
	}
	resource, _ := ctx.Value(networkAddressListResourceContextKey{}).(*wafv1beta1.NetworkAddressList)
	return resource
}

func networkAddressListRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "waf",
		FormalSlug:        "networkaddresslist",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(wafsdk.NetworkAddressListLifecycleStateCreating)},
			UpdatingStates:     []string{string(wafsdk.NetworkAddressListLifecycleStateUpdating)},
			ActiveStates:       []string{string(wafsdk.NetworkAddressListLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(wafsdk.NetworkAddressListLifecycleStateCreating),
				string(wafsdk.NetworkAddressListLifecycleStateUpdating),
				string(wafsdk.NetworkAddressListLifecycleStateDeleting),
			},
			TerminalStates: []string{string(wafsdk.NetworkAddressListLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"displayName",
				"freeformTags",
				"definedTags",
				"systemTags",
				"addresses",
				"vcnAddresses",
			},
			Mutable: []string{
				"displayName",
				"freeformTags",
				"definedTags",
				"systemTags",
				"addresses",
				"vcnAddresses",
			},
			ForceNew:      []string{"compartmentId", "type"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "NetworkAddressList", Action: "CreateNetworkAddressList"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "NetworkAddressList", Action: "UpdateNetworkAddressList"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "NetworkAddressList", Action: "DeleteNetworkAddressList"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func networkAddressListListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildNetworkAddressListCreateBody(
	_ context.Context,
	resource *wafv1beta1.NetworkAddressList,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("NetworkAddressList resource is nil")
	}
	if strings.TrimSpace(resource.Spec.JsonData) != "" {
		return nil, fmt.Errorf("NetworkAddressList spec.jsonData is not supported; use typed type, addresses, and vcnAddresses fields")
	}
	desiredType, err := desiredNetworkAddressListType(resource.Spec, "")
	if err != nil {
		return nil, err
	}
	if err := validateNetworkAddressListCreateSpec(resource.Spec, desiredType); err != nil {
		return nil, err
	}

	switch desiredType {
	case networkAddressListTypeAddresses:
		return wafsdk.CreateNetworkAddressListAddressesDetails{
			CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
			Addresses:     cloneNetworkAddressListStrings(resource.Spec.Addresses),
			DisplayName:   optionalNetworkAddressListString(resource.Spec.DisplayName),
			FreeformTags:  maps.Clone(resource.Spec.FreeformTags),
			DefinedTags:   networkAddressListDefinedTagsFromSpec(resource.Spec.DefinedTags),
			SystemTags:    networkAddressListDefinedTagsFromSpec(resource.Spec.SystemTags),
		}, nil
	case networkAddressListTypeVcnAddresses:
		vcnAddresses, err := networkAddressListVcnAddressesFromSpec(resource.Spec.VcnAddresses)
		if err != nil {
			return nil, err
		}
		return wafsdk.CreateNetworkAddressListVcnAddressesDetails{
			CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
			VcnAddresses:  vcnAddresses,
			DisplayName:   optionalNetworkAddressListString(resource.Spec.DisplayName),
			FreeformTags:  maps.Clone(resource.Spec.FreeformTags),
			DefinedTags:   networkAddressListDefinedTagsFromSpec(resource.Spec.DefinedTags),
			SystemTags:    networkAddressListDefinedTagsFromSpec(resource.Spec.SystemTags),
		}, nil
	default:
		return nil, fmt.Errorf("NetworkAddressList type %q is unsupported", desiredType)
	}
}

func buildNetworkAddressListUpdateBody(
	_ context.Context,
	resource *wafv1beta1.NetworkAddressList,
	_ string,
	currentResponse any,
) (any, bool, error) {
	inputs, err := newNetworkAddressListUpdateInputs(resource, currentResponse)
	if err != nil {
		return nil, false, err
	}

	switch inputs.desiredType {
	case networkAddressListTypeAddresses:
		return buildNetworkAddressListAddressesUpdateBody(inputs.spec, inputs.current)
	case networkAddressListTypeVcnAddresses:
		return buildNetworkAddressListVcnAddressesUpdateBody(inputs.spec, inputs.current)
	default:
		return nil, false, fmt.Errorf("NetworkAddressList type %q is unsupported", inputs.desiredType)
	}
}

type networkAddressListUpdateInputs struct {
	spec        wafv1beta1.NetworkAddressListSpec
	current     networkAddressListObserved
	desiredType string
}

func newNetworkAddressListUpdateInputs(
	resource *wafv1beta1.NetworkAddressList,
	currentResponse any,
) (networkAddressListUpdateInputs, error) {
	spec, err := networkAddressListUpdateSpec(resource)
	if err != nil {
		return networkAddressListUpdateInputs{}, err
	}
	current, err := currentNetworkAddressListObserved(currentResponse)
	if err != nil {
		return networkAddressListUpdateInputs{}, err
	}
	desiredType, err := compatibleNetworkAddressListUpdateType(spec, current)
	if err != nil {
		return networkAddressListUpdateInputs{}, err
	}

	return networkAddressListUpdateInputs{
		spec:        spec,
		current:     current,
		desiredType: desiredType,
	}, nil
}

func networkAddressListUpdateSpec(resource *wafv1beta1.NetworkAddressList) (wafv1beta1.NetworkAddressListSpec, error) {
	if resource == nil {
		return wafv1beta1.NetworkAddressListSpec{}, fmt.Errorf("NetworkAddressList resource is nil")
	}
	if strings.TrimSpace(resource.Spec.JsonData) != "" {
		return wafv1beta1.NetworkAddressListSpec{}, fmt.Errorf("NetworkAddressList spec.jsonData is not supported; use typed type, addresses, and vcnAddresses fields")
	}
	return resource.Spec, nil
}

func currentNetworkAddressListObserved(currentResponse any) (networkAddressListObserved, error) {
	current, ok := networkAddressListObservedFromResponse(currentResponse)
	if !ok {
		return networkAddressListObserved{}, fmt.Errorf("current NetworkAddressList response does not expose a NetworkAddressList body")
	}
	return current, nil
}

func compatibleNetworkAddressListUpdateType(
	spec wafv1beta1.NetworkAddressListSpec,
	current networkAddressListObserved,
) (string, error) {
	if err := rejectNetworkAddressListForceNewDrift(spec, current); err != nil {
		return "", err
	}
	desiredType, err := desiredNetworkAddressListType(spec, current.Type)
	if err != nil {
		return "", err
	}
	if current.Type != "" && desiredType != current.Type {
		return "", fmt.Errorf("NetworkAddressList formal semantics require replacement when type changes")
	}
	if err := validateNetworkAddressListUpdateSpec(spec, desiredType); err != nil {
		return "", err
	}
	return desiredType, nil
}

func rejectNetworkAddressListForceNewDrift(spec wafv1beta1.NetworkAddressListSpec, current networkAddressListObserved) error {
	if current.CompartmentId == "" || strings.TrimSpace(spec.CompartmentId) == "" {
		return nil
	}
	if current.CompartmentId != strings.TrimSpace(spec.CompartmentId) {
		return fmt.Errorf("NetworkAddressList formal semantics require replacement when compartmentId changes")
	}
	return nil
}

func buildNetworkAddressListAddressesUpdateBody(
	spec wafv1beta1.NetworkAddressListSpec,
	current networkAddressListObserved,
) (wafsdk.UpdateNetworkAddressListAddressesDetails, bool, error) {
	details := wafsdk.UpdateNetworkAddressListAddressesDetails{}
	updateNeeded := applyNetworkAddressListCommonUpdates(&details, spec, current)
	if spec.Addresses != nil && !reflect.DeepEqual(spec.Addresses, current.Addresses) {
		details.Addresses = cloneNetworkAddressListStrings(spec.Addresses)
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func buildNetworkAddressListVcnAddressesUpdateBody(
	spec wafv1beta1.NetworkAddressListSpec,
	current networkAddressListObserved,
) (wafsdk.UpdateNetworkAddressListVcnAddressesDetails, bool, error) {
	details := wafsdk.UpdateNetworkAddressListVcnAddressesDetails{}
	updateNeeded := applyNetworkAddressListCommonUpdates(&details, spec, current)
	desired, err := networkAddressListVcnAddressesFromSpec(spec.VcnAddresses)
	if err != nil {
		return details, false, err
	}
	if spec.VcnAddresses != nil && !reflect.DeepEqual(spec.VcnAddresses, current.VcnAddresses) {
		details.VcnAddresses = desired
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func applyNetworkAddressListCommonUpdates(details any, spec wafv1beta1.NetworkAddressListSpec, current networkAddressListObserved) bool {
	updateNeeded := false
	if desired := strings.TrimSpace(spec.DisplayName); desired != "" && desired != current.DisplayName {
		setNetworkAddressListUpdateDisplayName(details, desired)
		updateNeeded = true
	}
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		setNetworkAddressListUpdateFreeformTags(details, maps.Clone(spec.FreeformTags))
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := networkAddressListDefinedTagsFromSpec(spec.DefinedTags)
		if !reflect.DeepEqual(desired, current.DefinedTags) {
			setNetworkAddressListUpdateDefinedTags(details, desired)
			updateNeeded = true
		}
	}
	if spec.SystemTags != nil {
		desired := networkAddressListDefinedTagsFromSpec(spec.SystemTags)
		if !reflect.DeepEqual(desired, current.SystemTags) {
			setNetworkAddressListUpdateSystemTags(details, desired)
			updateNeeded = true
		}
	}
	return updateNeeded
}

func setNetworkAddressListUpdateDisplayName(details any, value string) {
	switch typed := details.(type) {
	case *wafsdk.UpdateNetworkAddressListAddressesDetails:
		typed.DisplayName = common.String(value)
	case *wafsdk.UpdateNetworkAddressListVcnAddressesDetails:
		typed.DisplayName = common.String(value)
	}
}

func setNetworkAddressListUpdateFreeformTags(details any, value map[string]string) {
	switch typed := details.(type) {
	case *wafsdk.UpdateNetworkAddressListAddressesDetails:
		typed.FreeformTags = value
	case *wafsdk.UpdateNetworkAddressListVcnAddressesDetails:
		typed.FreeformTags = value
	}
}

func setNetworkAddressListUpdateDefinedTags(details any, value map[string]map[string]interface{}) {
	switch typed := details.(type) {
	case *wafsdk.UpdateNetworkAddressListAddressesDetails:
		typed.DefinedTags = value
	case *wafsdk.UpdateNetworkAddressListVcnAddressesDetails:
		typed.DefinedTags = value
	}
}

func setNetworkAddressListUpdateSystemTags(details any, value map[string]map[string]interface{}) {
	switch typed := details.(type) {
	case *wafsdk.UpdateNetworkAddressListAddressesDetails:
		typed.SystemTags = value
	case *wafsdk.UpdateNetworkAddressListVcnAddressesDetails:
		typed.SystemTags = value
	}
}

func validateNetworkAddressListCreateSpec(spec wafv1beta1.NetworkAddressListSpec, desiredType string) error {
	problems := networkAddressListBaseSpecProblems(spec)
	switch desiredType {
	case networkAddressListTypeAddresses:
		if len(spec.Addresses) == 0 {
			problems = append(problems, "addresses is required when type is ADDRESSES")
		}
	case networkAddressListTypeVcnAddresses:
		if len(spec.VcnAddresses) == 0 {
			problems = append(problems, "vcnAddresses is required when type is VCN_ADDRESSES")
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("NetworkAddressList spec is invalid: %s", strings.Join(problems, "; "))
}

func validateNetworkAddressListUpdateSpec(spec wafv1beta1.NetworkAddressListSpec, desiredType string) error {
	problems := networkAddressListBaseSpecProblems(spec)
	if desiredType == networkAddressListTypeVcnAddresses {
		if _, err := networkAddressListVcnAddressesFromSpec(spec.VcnAddresses); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("NetworkAddressList spec is invalid: %s", strings.Join(problems, "; "))
}

func networkAddressListBaseSpecProblems(spec wafv1beta1.NetworkAddressListSpec) []string {
	var problems []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if spec.Addresses != nil && spec.VcnAddresses != nil {
		problems = append(problems, "addresses and vcnAddresses cannot both be set")
	}
	if spec.Type != "" {
		if _, err := normalizeNetworkAddressListType(spec.Type); err != nil {
			problems = append(problems, err.Error())
		}
	}
	return problems
}

func desiredNetworkAddressListType(spec wafv1beta1.NetworkAddressListSpec, fallback string) (string, error) {
	if desired := strings.TrimSpace(spec.Type); desired != "" {
		return desiredExplicitNetworkAddressListType(spec, desired)
	}
	return inferredNetworkAddressListType(spec, fallback)
}

func desiredExplicitNetworkAddressListType(spec wafv1beta1.NetworkAddressListSpec, desired string) (string, error) {
	normalized, err := normalizeNetworkAddressListType(desired)
	if err != nil {
		return "", err
	}
	if err := validateExplicitNetworkAddressListType(spec, normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func validateExplicitNetworkAddressListType(spec wafv1beta1.NetworkAddressListSpec, normalized string) error {
	switch normalized {
	case networkAddressListTypeAddresses:
		if spec.VcnAddresses != nil {
			return fmt.Errorf("vcnAddresses cannot be set when type is ADDRESSES")
		}
	case networkAddressListTypeVcnAddresses:
		if spec.Addresses != nil {
			return fmt.Errorf("addresses cannot be set when type is VCN_ADDRESSES")
		}
	}
	return nil
}

func inferredNetworkAddressListType(spec wafv1beta1.NetworkAddressListSpec, fallback string) (string, error) {
	switch {
	case spec.Addresses != nil && spec.VcnAddresses == nil:
		return networkAddressListTypeAddresses, nil
	case spec.VcnAddresses != nil && spec.Addresses == nil:
		return networkAddressListTypeVcnAddresses, nil
	case strings.TrimSpace(fallback) != "":
		return normalizeNetworkAddressListType(fallback)
	default:
		return "", fmt.Errorf("NetworkAddressList type is required when neither addresses nor vcnAddresses is set")
	}
}

func normalizeNetworkAddressListType(value string) (string, error) {
	normalized, ok := wafsdk.GetMappingNetworkAddressListTypeEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("NetworkAddressList type %q is unsupported", value)
	}
	return string(normalized), nil
}

func normalizeNetworkAddressListDesiredState(resource *wafv1beta1.NetworkAddressList, _ any) {
	if resource == nil {
		return
	}
	normalized, err := normalizeNetworkAddressListType(resource.Spec.Type)
	if err != nil {
		return
	}
	resource.Spec.Type = normalized
}

func listNetworkAddressListsAllPages(call networkAddressListListCall) networkAddressListListCall {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error) {
		combined := wafsdk.ListNetworkAddressListsResponse{}
		for {
			response, err := call(ctx, request)
			if err != nil {
				return combined, err
			}
			combined.Items = append(combined.Items, response.Items...)
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				if resource := networkAddressListResourceFromContext(ctx); resource != nil {
					combined.Items = filterNetworkAddressListSummariesForSpec(combined.Items, resource.Spec)
				}
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func (c *networkAddressListRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *wafv1beta1.NetworkAddressList,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("NetworkAddressList runtime client is not configured")
	}
	ctx = contextWithNetworkAddressListRequestBodies(ctx)
	ctx = contextWithNetworkAddressListResource(ctx, resource)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *networkAddressListRuntimeClient) Delete(ctx context.Context, resource *wafv1beta1.NetworkAddressList) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("NetworkAddressList runtime client is not configured")
	}
	ctx = contextWithNetworkAddressListResource(ctx, resource)
	if deleted, err, handled := c.deleteUntrackedNoMatch(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *networkAddressListRuntimeClient) deleteUntrackedNoMatch(
	ctx context.Context,
	resource *wafv1beta1.NetworkAddressList,
) (bool, error, bool) {
	if currentNetworkAddressListID(resource) != "" || c.confirmRead == nil {
		return false, nil, false
	}
	response, err := c.confirmRead(ctx, resource, "")
	if err == nil {
		if authErr, ok := networkAddressListConfirmReadAuthError(response); ok {
			recordNetworkAddressListConfirmReadRequestID(resource, authErr)
			return false, authErr, true
		}
		return false, nil, false
	}
	var noMatch networkAddressListNoMatchConfirmRead
	if !errors.As(err, &noMatch) {
		return false, err, true
	}
	markNetworkAddressListDeleted(resource, noMatch.message)
	return true, nil, true
}

func (c *networkAddressListRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *wafv1beta1.NetworkAddressList,
) error {
	currentID := currentNetworkAddressListID(resource)
	if currentID == "" || c.get == nil {
		return nil
	}
	_, err := c.get(ctx, wafsdk.GetNetworkAddressListRequest{NetworkAddressListId: common.String(currentID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("NetworkAddressList delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
}

func confirmNetworkAddressListDeleteRead(
	ctx context.Context,
	hooks *NetworkAddressListRuntimeHooks,
	resource *wafv1beta1.NetworkAddressList,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm NetworkAddressList delete: runtime hooks are nil")
	}
	if currentID = strings.TrimSpace(currentID); currentID != "" {
		return confirmNetworkAddressListDeleteReadByID(ctx, hooks, currentID)
	}
	return confirmNetworkAddressListDeleteReadByIdentity(ctx, hooks, resource)
}

func confirmNetworkAddressListDeleteReadByID(
	ctx context.Context,
	hooks *NetworkAddressListRuntimeHooks,
	currentID string,
) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm NetworkAddressList delete: get hook is not configured")
	}
	response, err := hooks.Get.Call(ctx, wafsdk.GetNetworkAddressListRequest{
		NetworkAddressListId: common.String(currentID),
	})
	if err != nil && errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return networkAddressListAuthShapedConfirmRead{err: err}, nil
	}
	return response, err
}

func confirmNetworkAddressListDeleteReadByIdentity(
	ctx context.Context,
	hooks *NetworkAddressListRuntimeHooks,
	resource *wafv1beta1.NetworkAddressList,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("confirm NetworkAddressList delete: resource is nil")
	}
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm NetworkAddressList delete: list hook is not configured")
	}

	response, err := hooks.List.Call(ctx, wafsdk.ListNetworkAddressListsRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   optionalNetworkAddressListString(resource.Spec.DisplayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return networkAddressListAuthShapedConfirmRead{err: err}, nil
		}
		return nil, err
	}

	matches := make([]wafsdk.NetworkAddressListSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if networkAddressListSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, networkAddressListNoMatchConfirmRead{message: "NetworkAddressList delete confirmation did not find a matching OCI network address list"}
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("NetworkAddressList list response returned multiple matching resources for compartmentId %q and displayName %q", resource.Spec.CompartmentId, resource.Spec.DisplayName)
	}
}

func applyNetworkAddressListDeleteOutcome(
	resource *wafv1beta1.NetworkAddressList,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case networkAddressListAuthShapedConfirmRead:
		recordNetworkAddressListConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *networkAddressListAuthShapedConfirmRead:
		if typed != nil {
			recordNetworkAddressListConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func handleNetworkAddressListDeleteError(resource *wafv1beta1.NetworkAddressList, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("NetworkAddressList delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func networkAddressListSummaryMatchesSpec(summary wafsdk.NetworkAddressListSummary, spec wafv1beta1.NetworkAddressListSpec) bool {
	if summary == nil {
		return false
	}
	observed, ok := networkAddressListObservedFromResponse(summary)
	if !ok {
		return false
	}
	if strings.TrimSpace(spec.CompartmentId) != "" && observed.CompartmentId != strings.TrimSpace(spec.CompartmentId) {
		return false
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		return observed.DisplayName == strings.TrimSpace(spec.DisplayName)
	}
	return networkAddressListPayloadMatchesObserved(spec, observed)
}

func filterNetworkAddressListSummariesForSpec(
	items []wafsdk.NetworkAddressListSummary,
	spec wafv1beta1.NetworkAddressListSpec,
) []wafsdk.NetworkAddressListSummary {
	if len(items) == 0 {
		return items
	}
	filtered := make([]wafsdk.NetworkAddressListSummary, 0, len(items))
	for _, item := range items {
		if networkAddressListSummaryMatchesSpec(item, spec) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func networkAddressListPayloadMatchesObserved(
	spec wafv1beta1.NetworkAddressListSpec,
	observed networkAddressListObserved,
) bool {
	desiredType, err := desiredNetworkAddressListType(spec, "")
	if err != nil {
		return false
	}
	if observed.Type != "" && observed.Type != desiredType {
		return false
	}

	switch desiredType {
	case networkAddressListTypeAddresses:
		return spec.Addresses != nil && reflect.DeepEqual(spec.Addresses, observed.Addresses)
	case networkAddressListTypeVcnAddresses:
		return spec.VcnAddresses != nil && reflect.DeepEqual(spec.VcnAddresses, observed.VcnAddresses)
	default:
		return false
	}
}

func recordNetworkAddressListConfirmReadRequestID(resource *wafv1beta1.NetworkAddressList, err networkAddressListAuthShapedConfirmRead) {
	if resource == nil {
		return
	}
	servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, err.GetOpcRequestID())
}

func networkAddressListConfirmReadAuthError(response any) (networkAddressListAuthShapedConfirmRead, bool) {
	switch typed := response.(type) {
	case networkAddressListAuthShapedConfirmRead:
		return typed, true
	case *networkAddressListAuthShapedConfirmRead:
		if typed != nil {
			return *typed, true
		}
	}
	return networkAddressListAuthShapedConfirmRead{}, false
}

func currentNetworkAddressListID(resource *wafv1beta1.NetworkAddressList) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func markNetworkAddressListDeleted(resource *wafv1beta1.NetworkAddressList, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func networkAddressListObservedFromResponse(response any) (networkAddressListObserved, bool) {
	return networkAddressListObservedFromBody(networkAddressListResponseBody(response))
}

func networkAddressListResponseBody(response any) any {
	switch current := response.(type) {
	case wafsdk.CreateNetworkAddressListResponse:
		return current.NetworkAddressList
	case *wafsdk.CreateNetworkAddressListResponse:
		if current == nil {
			return nil
		}
		return current.NetworkAddressList
	case wafsdk.GetNetworkAddressListResponse:
		return current.NetworkAddressList
	case *wafsdk.GetNetworkAddressListResponse:
		if current == nil {
			return nil
		}
		return current.NetworkAddressList
	default:
		return response
	}
}

func networkAddressListObservedFromBody(response any) (networkAddressListObserved, bool) {
	if observed, ok := networkAddressListObservedFromAddressesBody(response); ok {
		return observed, true
	}
	if observed, ok := networkAddressListObservedFromVcnAddressesBody(response); ok {
		return observed, true
	}
	return networkAddressListObservedFromCommonBody(response)
}

func networkAddressListObservedFromAddressesBody(response any) (networkAddressListObserved, bool) {
	switch current := response.(type) {
	case wafsdk.NetworkAddressListAddresses:
		return networkAddressListObservedFromAddresses(current), true
	case *wafsdk.NetworkAddressListAddresses:
		if current == nil {
			return networkAddressListObserved{}, false
		}
		return networkAddressListObservedFromAddresses(*current), true
	case wafsdk.NetworkAddressListAddressesSummary:
		return networkAddressListObservedFromAddressesSummary(current), true
	case *wafsdk.NetworkAddressListAddressesSummary:
		if current == nil {
			return networkAddressListObserved{}, false
		}
		return networkAddressListObservedFromAddressesSummary(*current), true
	default:
		return networkAddressListObserved{}, false
	}
}

func networkAddressListObservedFromVcnAddressesBody(response any) (networkAddressListObserved, bool) {
	switch current := response.(type) {
	case wafsdk.NetworkAddressListVcnAddresses:
		return networkAddressListObservedFromVcnAddresses(current), true
	case *wafsdk.NetworkAddressListVcnAddresses:
		if current == nil {
			return networkAddressListObserved{}, false
		}
		return networkAddressListObservedFromVcnAddresses(*current), true
	case wafsdk.NetworkAddressListVcnAddressesSummary:
		return networkAddressListObservedFromVcnAddressesSummary(current), true
	case *wafsdk.NetworkAddressListVcnAddressesSummary:
		if current == nil {
			return networkAddressListObserved{}, false
		}
		return networkAddressListObservedFromVcnAddressesSummary(*current), true
	default:
		return networkAddressListObserved{}, false
	}
}

func networkAddressListObservedFromCommonBody(response any) (networkAddressListObserved, bool) {
	switch current := response.(type) {
	case wafsdk.NetworkAddressList:
		if current == nil {
			return networkAddressListObserved{}, false
		}
		return networkAddressListObservedFromBase(current), true
	default:
		return networkAddressListObserved{}, false
	}
}

func networkAddressListObservedFromAddresses(current wafsdk.NetworkAddressListAddresses) networkAddressListObserved {
	observed := networkAddressListObservedFromBase(current)
	observed.Type = networkAddressListTypeAddresses
	observed.Addresses = cloneNetworkAddressListStrings(current.Addresses)
	return observed
}

func networkAddressListObservedFromVcnAddresses(current wafsdk.NetworkAddressListVcnAddresses) networkAddressListObserved {
	observed := networkAddressListObservedFromBase(current)
	observed.Type = networkAddressListTypeVcnAddresses
	observed.VcnAddresses = networkAddressListVcnAddressesToSpec(current.VcnAddresses)
	return observed
}

func networkAddressListObservedFromAddressesSummary(current wafsdk.NetworkAddressListAddressesSummary) networkAddressListObserved {
	observed := networkAddressListObservedFromSummary(current)
	observed.Type = networkAddressListTypeAddresses
	observed.Addresses = cloneNetworkAddressListStrings(current.Addresses)
	return observed
}

func networkAddressListObservedFromVcnAddressesSummary(current wafsdk.NetworkAddressListVcnAddressesSummary) networkAddressListObserved {
	observed := networkAddressListObservedFromSummary(current)
	observed.Type = networkAddressListTypeVcnAddresses
	observed.VcnAddresses = networkAddressListVcnAddressesToSpec(current.VcnAddresses)
	return observed
}

func networkAddressListObservedFromBase(current wafsdk.NetworkAddressList) networkAddressListObserved {
	return networkAddressListObserved{
		Id:               stringValue(current.GetId()),
		DisplayName:      stringValue(current.GetDisplayName()),
		CompartmentId:    stringValue(current.GetCompartmentId()),
		LifecycleState:   string(current.GetLifecycleState()),
		LifecycleDetails: stringValue(current.GetLifecycleDetails()),
		FreeformTags:     maps.Clone(current.GetFreeformTags()),
		DefinedTags:      cloneNetworkAddressListDefinedTags(current.GetDefinedTags()),
		SystemTags:       cloneNetworkAddressListDefinedTags(current.GetSystemTags()),
	}
}

func networkAddressListObservedFromSummary(current wafsdk.NetworkAddressListSummary) networkAddressListObserved {
	return networkAddressListObserved{
		Id:               stringValue(current.GetId()),
		DisplayName:      stringValue(current.GetDisplayName()),
		CompartmentId:    stringValue(current.GetCompartmentId()),
		LifecycleState:   string(current.GetLifecycleState()),
		LifecycleDetails: stringValue(current.GetLifecycleDetails()),
		FreeformTags:     maps.Clone(current.GetFreeformTags()),
		DefinedTags:      cloneNetworkAddressListDefinedTags(current.GetDefinedTags()),
		SystemTags:       cloneNetworkAddressListDefinedTags(current.GetSystemTags()),
	}
}

func networkAddressListVcnAddressesFromSpec(addresses []wafv1beta1.NetworkAddressListVcnAddress) ([]wafsdk.PrivateAddresses, error) {
	if addresses == nil {
		return nil, nil
	}
	converted := make([]wafsdk.PrivateAddresses, 0, len(addresses))
	for index, address := range addresses {
		if strings.TrimSpace(address.VcnId) == "" {
			return nil, fmt.Errorf("vcnAddresses[%d].vcnId is required", index)
		}
		if strings.TrimSpace(address.Addresses) == "" {
			return nil, fmt.Errorf("vcnAddresses[%d].addresses is required", index)
		}
		converted = append(converted, wafsdk.PrivateAddresses{
			VcnId:     common.String(strings.TrimSpace(address.VcnId)),
			Addresses: common.String(strings.TrimSpace(address.Addresses)),
		})
	}
	return converted, nil
}

func networkAddressListVcnAddressesToSpec(addresses []wafsdk.PrivateAddresses) []wafv1beta1.NetworkAddressListVcnAddress {
	if addresses == nil {
		return nil
	}
	converted := make([]wafv1beta1.NetworkAddressListVcnAddress, 0, len(addresses))
	for _, address := range addresses {
		converted = append(converted, wafv1beta1.NetworkAddressListVcnAddress{
			VcnId:     stringValue(address.VcnId),
			Addresses: stringValue(address.Addresses),
		})
	}
	return converted
}

func networkAddressListDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func cloneNetworkAddressListDefinedTags(tags map[string]map[string]interface{}) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		cloned[namespace] = maps.Clone(values)
	}
	return cloned
}

func cloneNetworkAddressListStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func optionalNetworkAddressListString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
