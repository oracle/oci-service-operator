/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networkloadbalancer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"reflect"
	"sort"
	"strings"

	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type networkLoadBalancerRuntimeOCIClient interface {
	CreateNetworkLoadBalancer(context.Context, networkloadbalancersdk.CreateNetworkLoadBalancerRequest) (networkloadbalancersdk.CreateNetworkLoadBalancerResponse, error)
	GetNetworkLoadBalancer(context.Context, networkloadbalancersdk.GetNetworkLoadBalancerRequest) (networkloadbalancersdk.GetNetworkLoadBalancerResponse, error)
	ListNetworkLoadBalancers(context.Context, networkloadbalancersdk.ListNetworkLoadBalancersRequest) (networkloadbalancersdk.ListNetworkLoadBalancersResponse, error)
	UpdateNetworkLoadBalancer(context.Context, networkloadbalancersdk.UpdateNetworkLoadBalancerRequest) (networkloadbalancersdk.UpdateNetworkLoadBalancerResponse, error)
	UpdateNetworkSecurityGroups(context.Context, networkloadbalancersdk.UpdateNetworkSecurityGroupsRequest) (networkloadbalancersdk.UpdateNetworkSecurityGroupsResponse, error)
	DeleteNetworkLoadBalancer(context.Context, networkloadbalancersdk.DeleteNetworkLoadBalancerRequest) (networkloadbalancersdk.DeleteNetworkLoadBalancerResponse, error)
	GetWorkRequest(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

type networkLoadBalancerWorkRequestClient interface {
	GetWorkRequest(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

type networkLoadBalancerNetworkSecurityGroupsClient interface {
	UpdateNetworkSecurityGroups(context.Context, networkloadbalancersdk.UpdateNetworkSecurityGroupsRequest) (networkloadbalancersdk.UpdateNetworkSecurityGroupsResponse, error)
}

type networkLoadBalancerRuntimeSupportClient interface {
	networkLoadBalancerWorkRequestClient
	networkLoadBalancerNetworkSecurityGroupsClient
}

var networkLoadBalancerWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(networkloadbalancersdk.OperationStatusAccepted),
		string(networkloadbalancersdk.OperationStatusInProgress),
		string(networkloadbalancersdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(networkloadbalancersdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(networkloadbalancersdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(networkloadbalancersdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(networkloadbalancersdk.OperationTypeCreateNetworkLoadBalancer),
	},
	UpdateActionTokens: []string{
		string(networkloadbalancersdk.OperationTypeUpdateNetworkLoadBalancer),
		string(networkloadbalancersdk.OperationTypeUpdateNsgs),
	},
	DeleteActionTokens: []string{
		string(networkloadbalancersdk.OperationTypeDeleteNetworkLoadBalancer),
	},
}

func init() {
	registerNetworkLoadBalancerRuntimeHooksMutator(func(manager *NetworkLoadBalancerServiceManager, hooks *NetworkLoadBalancerRuntimeHooks) {
		runtimeClient, initErr := newNetworkLoadBalancerRuntimeSupportClient(manager)
		applyNetworkLoadBalancerRuntimeHooks(hooks, runtimeClient, initErr)
	})
}

func newNetworkLoadBalancerRuntimeSupportClient(manager *NetworkLoadBalancerServiceManager) (networkLoadBalancerRuntimeSupportClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("NetworkLoadBalancer service manager is nil")
	}
	client, err := networkloadbalancersdk.NewNetworkLoadBalancerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyNetworkLoadBalancerRuntimeHooks(
	hooks *NetworkLoadBalancerRuntimeHooks,
	runtimeClient networkLoadBalancerRuntimeSupportClient,
	runtimeClientInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newReviewedNetworkLoadBalancerRuntimeSemantics()
	hooks.Async.Adapter = networkLoadBalancerWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getNetworkLoadBalancerWorkRequest(ctx, runtimeClient, runtimeClientInitErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveNetworkLoadBalancerWorkRequestAction
	hooks.Async.RecoverResourceID = recoverNetworkLoadBalancerIDFromWorkRequest
	hooks.Async.Message = networkLoadBalancerWorkRequestMessage
	hooks.BuildCreateBody = buildNetworkLoadBalancerCreateBody
	hooks.BuildUpdateBody = buildNetworkLoadBalancerUpdateBody
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateNetworkLoadBalancerCreateOnlyDrift
	hooks.Create.Fields = networkLoadBalancerCreateFields()
	hooks.Get.Fields = networkLoadBalancerGetFields()
	hooks.List.Fields = networkLoadBalancerListFields()
	hooks.Update.Fields = networkLoadBalancerUpdateFields()
	hooks.Delete.Fields = networkLoadBalancerDeleteFields()
	hooks.DeleteHooks.HandleError = handleNetworkLoadBalancerDeleteError
	wrapNetworkLoadBalancerDeleteConfirmation(hooks, runtimeClient, runtimeClientInitErr)
	wrapNetworkLoadBalancerNetworkSecurityGroupUpdates(hooks, runtimeClient, runtimeClientInitErr)
	if hooks.List.Call != nil {
		hooks.List.Call = listNetworkLoadBalancersAllPages(hooks.List.Call)
	}
}

func newReviewedNetworkLoadBalancerRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "networkloadbalancer",
		FormalSlug:    "networkloadbalancer",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"displayName", "compartmentId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"assignedIpv6",
				"definedTags",
				"displayName",
				"freeformTags",
				"isPreserveSourceDestination",
				"isSymmetricHashEnabled",
				"networkSecurityGroupIds",
				"nlbIpVersion",
				"reservedIpv6Id",
				"securityAttributes",
				"subnetIpv6Cidr",
			},
			ForceNew: []string{
				"assignedPrivateIpv4",
				"backendSets",
				"compartmentId",
				"isPrivate",
				"listeners",
				"reservedIps",
				"subnetId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func getNetworkLoadBalancerWorkRequest(
	ctx context.Context,
	client networkLoadBalancerWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize NetworkLoadBalancer work request OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("NetworkLoadBalancer work request OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, networkloadbalancersdk.GetWorkRequestRequest{
		WorkRequestId: stringPointer(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveNetworkLoadBalancerWorkRequestAction(workRequest any) (string, error) {
	current, err := networkLoadBalancerWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func recoverNetworkLoadBalancerIDFromWorkRequest(
	_ *networkloadbalancerv1beta1.NetworkLoadBalancer,
	workRequest any,
	_ shared.OSOKAsyncPhase,
) (string, error) {
	current, err := networkLoadBalancerWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	for _, resource := range current.Resources {
		identifier := strings.TrimSpace(networkLoadBalancerStringValue(resource.Identifier))
		if identifier == "" {
			continue
		}
		entityType := strings.ToLower(strings.TrimSpace(networkLoadBalancerStringValue(resource.EntityType)))
		if entityType == "" || strings.Contains(entityType, "networkloadbalancer") || strings.Contains(entityType, "network-load-balancer") {
			return identifier, nil
		}
	}
	return "", nil
}

func networkLoadBalancerWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := networkLoadBalancerWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	workRequestID := strings.TrimSpace(networkLoadBalancerStringValue(current.Id))
	status := strings.TrimSpace(string(current.Status))
	if workRequestID == "" || status == "" {
		return ""
	}
	return fmt.Sprintf("NetworkLoadBalancer %s work request %s is %s", phase, workRequestID, status)
}

func networkLoadBalancerWorkRequestFromAny(workRequest any) (networkloadbalancersdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case networkloadbalancersdk.WorkRequest:
		return current, nil
	case *networkloadbalancersdk.WorkRequest:
		if current == nil {
			return networkloadbalancersdk.WorkRequest{}, fmt.Errorf("NetworkLoadBalancer work request is nil")
		}
		return *current, nil
	default:
		return networkloadbalancersdk.WorkRequest{}, fmt.Errorf("expected NetworkLoadBalancer work request, got %T", workRequest)
	}
}

func buildNetworkLoadBalancerCreateBody(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	namespace string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("NetworkLoadBalancer resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return nil, fmt.Errorf("marshal resolved NetworkLoadBalancer spec: %w", err)
	}

	var details networkloadbalancersdk.CreateNetworkLoadBalancerDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return nil, fmt.Errorf("decode NetworkLoadBalancer create body: %w", err)
	}
	return details, nil
}

func buildNetworkLoadBalancerUpdateBody(
	_ context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return networkloadbalancersdk.UpdateNetworkLoadBalancerDetails{}, false, fmt.Errorf("NetworkLoadBalancer resource is nil")
	}

	current, err := networkLoadBalancerRuntimeBody(currentResponse)
	if err != nil {
		return networkloadbalancersdk.UpdateNetworkLoadBalancerDetails{}, false, err
	}

	details := networkloadbalancersdk.UpdateNetworkLoadBalancerDetails{}
	updateNeeded := applyNetworkLoadBalancerScalarUpdates(&details, resource.Spec, current)

	ipv6Updated, err := applyNetworkLoadBalancerIPv6Updates(&details, resource.Spec, current)
	if err != nil {
		return networkloadbalancersdk.UpdateNetworkLoadBalancerDetails{}, false, err
	}
	updateNeeded = ipv6Updated || updateNeeded
	updateNeeded = applyNetworkLoadBalancerTagUpdates(&details, resource.Spec, current) || updateNeeded

	if !updateNeeded {
		return networkloadbalancersdk.UpdateNetworkLoadBalancerDetails{}, false, nil
	}
	return details, true, nil
}

func applyNetworkLoadBalancerScalarUpdates(
	details *networkloadbalancersdk.UpdateNetworkLoadBalancerDetails,
	spec networkloadbalancerv1beta1.NetworkLoadBalancerSpec,
	current networkloadbalancersdk.NetworkLoadBalancer,
) bool {
	updateNeeded := false
	if stringPointerNeedsUpdate(spec.DisplayName, current.DisplayName) {
		details.DisplayName = &spec.DisplayName
		updateNeeded = true
	}
	if boolPointerNeedsUpdate(spec.IsPreserveSourceDestination, current.IsPreserveSourceDestination) {
		details.IsPreserveSourceDestination = &spec.IsPreserveSourceDestination
		updateNeeded = true
	}
	if boolPointerNeedsUpdate(spec.IsSymmetricHashEnabled, current.IsSymmetricHashEnabled) {
		details.IsSymmetricHashEnabled = &spec.IsSymmetricHashEnabled
		updateNeeded = true
	}
	if spec.NlbIpVersion != "" && string(current.NlbIpVersion) != spec.NlbIpVersion {
		details.NlbIpVersion = networkloadbalancersdk.NlbIpVersionEnum(spec.NlbIpVersion)
		updateNeeded = true
	}
	return updateNeeded
}

func applyNetworkLoadBalancerIPv6Updates(
	details *networkloadbalancersdk.UpdateNetworkLoadBalancerDetails,
	spec networkloadbalancerv1beta1.NetworkLoadBalancerSpec,
	current networkloadbalancersdk.NetworkLoadBalancer,
) (bool, error) {
	updateNeeded := false
	if spec.SubnetIpv6Cidr != "" {
		hasIPv6InCIDR, err := networkLoadBalancerHasIPv6InCIDR(current.IpAddresses, spec.SubnetIpv6Cidr)
		if err != nil {
			return false, err
		}
		if !hasIPv6InCIDR {
			details.SubnetIpv6Cidr = &spec.SubnetIpv6Cidr
			updateNeeded = true
		}
	}
	if spec.AssignedIpv6 != "" && !networkLoadBalancerHasIPv6Address(current.IpAddresses, spec.AssignedIpv6) {
		details.AssignedIpv6 = &spec.AssignedIpv6
		updateNeeded = true
	}
	if spec.ReservedIpv6Id != "" && !networkLoadBalancerHasReservedIPv6(current.IpAddresses, spec.ReservedIpv6Id) {
		details.ReservedIpv6Id = &spec.ReservedIpv6Id
		updateNeeded = true
	}
	return updateNeeded, nil
}

func applyNetworkLoadBalancerTagUpdates(
	details *networkloadbalancersdk.UpdateNetworkLoadBalancerDetails,
	spec networkloadbalancerv1beta1.NetworkLoadBalancerSpec,
	current networkloadbalancersdk.NetworkLoadBalancer,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, spec.FreeformTags) {
		details.FreeformTags = copyNetworkLoadBalancerStringMap(spec.FreeformTags)
		updateNeeded = true
	}
	desiredDefinedTags := networkLoadBalancerDefinedTags(spec.DefinedTags)
	if desiredDefinedTags != nil && !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		details.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	desiredSecurityAttributes := networkLoadBalancerDefinedTags(spec.SecurityAttributes)
	if desiredSecurityAttributes != nil && !reflect.DeepEqual(current.SecurityAttributes, desiredSecurityAttributes) {
		details.SecurityAttributes = desiredSecurityAttributes
		updateNeeded = true
	}
	return updateNeeded
}

func validateNetworkLoadBalancerCreateOnlyDrift(
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("NetworkLoadBalancer resource is nil")
	}

	current, err := networkLoadBalancerRuntimeBody(currentResponse)
	if err != nil {
		return err
	}

	var driftPaths []string
	if assignedPrivateIPv4 := strings.TrimSpace(resource.Spec.AssignedPrivateIpv4); assignedPrivateIPv4 != "" &&
		!networkLoadBalancerHasIPv4Address(current.IpAddresses, assignedPrivateIPv4) {
		driftPaths = append(driftPaths, "assignedPrivateIpv4")
	}
	if len(missingNetworkLoadBalancerReservedIPIDs(resource.Spec.ReservedIps, current.IpAddresses)) > 0 {
		driftPaths = append(driftPaths, "reservedIps")
	}
	if len(driftPaths) == 0 {
		return nil
	}
	return fmt.Errorf("NetworkLoadBalancer formal semantics require replacement when %s changes", strings.Join(driftPaths, ", "))
}

func networkLoadBalancerNetworkSecurityGroupsNeedUpdate(
	desired []string,
	current []string,
) bool {
	if desired == nil {
		return false
	}
	return !reflect.DeepEqual(
		normalizedNetworkLoadBalancerNetworkSecurityGroupIDs(desired),
		normalizedNetworkLoadBalancerNetworkSecurityGroupIDs(current),
	)
}

func normalizedNetworkLoadBalancerNetworkSecurityGroupIDs(input []string) []string {
	if input == nil {
		return nil
	}
	output := make([]string, 0, len(input))
	for _, id := range input {
		if id := strings.TrimSpace(id); id != "" {
			output = append(output, id)
		}
	}
	sort.Strings(output)
	return output
}

func networkLoadBalancerRuntimeBody(currentResponse any) (networkloadbalancersdk.NetworkLoadBalancer, error) {
	if current, ok, err := networkLoadBalancerRuntimePointerBody(currentResponse); ok || err != nil {
		return current, err
	}

	switch current := currentResponse.(type) {
	case networkloadbalancersdk.NetworkLoadBalancer:
		return current, nil
	case networkloadbalancersdk.NetworkLoadBalancerSummary:
		return networkLoadBalancerFromSummary(current), nil
	case networkloadbalancersdk.CreateNetworkLoadBalancerResponse:
		return current.NetworkLoadBalancer, nil
	case networkloadbalancersdk.GetNetworkLoadBalancerResponse:
		return current.NetworkLoadBalancer, nil
	default:
		return networkloadbalancersdk.NetworkLoadBalancer{}, fmt.Errorf("unexpected current NetworkLoadBalancer response type %T", currentResponse)
	}
}

func networkLoadBalancerRuntimePointerBody(currentResponse any) (networkloadbalancersdk.NetworkLoadBalancer, bool, error) {
	switch current := currentResponse.(type) {
	case *networkloadbalancersdk.NetworkLoadBalancer:
		if current == nil {
			return networkloadbalancersdk.NetworkLoadBalancer{}, true, fmt.Errorf("current NetworkLoadBalancer response is nil")
		}
		return *current, true, nil
	case *networkloadbalancersdk.NetworkLoadBalancerSummary:
		if current == nil {
			return networkloadbalancersdk.NetworkLoadBalancer{}, true, fmt.Errorf("current NetworkLoadBalancer response is nil")
		}
		return networkLoadBalancerFromSummary(*current), true, nil
	case *networkloadbalancersdk.CreateNetworkLoadBalancerResponse:
		if current == nil {
			return networkloadbalancersdk.NetworkLoadBalancer{}, true, fmt.Errorf("current NetworkLoadBalancer response is nil")
		}
		return current.NetworkLoadBalancer, true, nil
	case *networkloadbalancersdk.GetNetworkLoadBalancerResponse:
		if current == nil {
			return networkloadbalancersdk.NetworkLoadBalancer{}, true, fmt.Errorf("current NetworkLoadBalancer response is nil")
		}
		return current.NetworkLoadBalancer, true, nil
	default:
		return networkloadbalancersdk.NetworkLoadBalancer{}, false, nil
	}
}

func networkLoadBalancerFromSummary(summary networkloadbalancersdk.NetworkLoadBalancerSummary) networkloadbalancersdk.NetworkLoadBalancer {
	return networkloadbalancersdk.NetworkLoadBalancer{
		Id:                          summary.Id,
		CompartmentId:               summary.CompartmentId,
		DisplayName:                 summary.DisplayName,
		LifecycleState:              summary.LifecycleState,
		TimeCreated:                 summary.TimeCreated,
		IpAddresses:                 append([]networkloadbalancersdk.IpAddress(nil), summary.IpAddresses...),
		SubnetId:                    summary.SubnetId,
		LifecycleDetails:            summary.LifecycleDetails,
		NlbIpVersion:                summary.NlbIpVersion,
		TimeUpdated:                 summary.TimeUpdated,
		IsPrivate:                   summary.IsPrivate,
		IsPreserveSourceDestination: summary.IsPreserveSourceDestination,
		IsSymmetricHashEnabled:      summary.IsSymmetricHashEnabled,
		NetworkSecurityGroupIds:     append([]string(nil), summary.NetworkSecurityGroupIds...),
		Listeners:                   summary.Listeners,
		BackendSets:                 summary.BackendSets,
		FreeformTags:                copyNetworkLoadBalancerStringMap(summary.FreeformTags),
		SecurityAttributes:          copyNetworkLoadBalancerNestedMap(summary.SecurityAttributes),
		DefinedTags:                 copyNetworkLoadBalancerNestedMap(summary.DefinedTags),
		SystemTags:                  copyNetworkLoadBalancerNestedMap(summary.SystemTags),
	}
}

func stringPointerNeedsUpdate(desired string, current *string) bool {
	if desired == "" {
		return false
	}
	return current == nil || *current != desired
}

func boolPointerNeedsUpdate(desired bool, current *bool) bool {
	return current == nil || *current != desired
}

func networkLoadBalancerHasIPv4Address(ipAddresses []networkloadbalancersdk.IpAddress, assignedIPv4 string) bool {
	for _, ipAddress := range ipAddresses {
		if ipAddress.IpAddress == nil || strings.TrimSpace(*ipAddress.IpAddress) != assignedIPv4 {
			continue
		}
		if ipAddress.IpVersion == networkloadbalancersdk.IpVersionIpv6 {
			continue
		}
		return true
	}
	return false
}

func networkLoadBalancerHasIPv6InCIDR(ipAddresses []networkloadbalancersdk.IpAddress, subnetIpv6Cidr string) (bool, error) {
	prefix, err := netip.ParsePrefix(subnetIpv6Cidr)
	if err != nil {
		return false, fmt.Errorf("invalid subnetIpv6Cidr %q: %w", subnetIpv6Cidr, err)
	}
	if !prefix.Addr().Is6() {
		return false, fmt.Errorf("invalid subnetIpv6Cidr %q: expected IPv6 CIDR", subnetIpv6Cidr)
	}

	for _, ipAddress := range ipAddresses {
		if ipAddress.IpVersion != "" && ipAddress.IpVersion != networkloadbalancersdk.IpVersionIpv6 {
			continue
		}
		if ipAddress.IpAddress == nil {
			continue
		}
		addr, err := netip.ParseAddr(*ipAddress.IpAddress)
		if err != nil || !addr.Is6() {
			continue
		}
		if prefix.Contains(addr) {
			return true, nil
		}
	}
	return false, nil
}

func networkLoadBalancerHasIPv6Address(ipAddresses []networkloadbalancersdk.IpAddress, assignedIPv6 string) bool {
	for _, ipAddress := range ipAddresses {
		if ipAddress.IpAddress == nil || *ipAddress.IpAddress != assignedIPv6 {
			continue
		}
		if ipAddress.IpVersion == "" || ipAddress.IpVersion == networkloadbalancersdk.IpVersionIpv6 {
			return true
		}
	}
	return false
}

func missingNetworkLoadBalancerReservedIPIDs(
	desired []networkloadbalancerv1beta1.NetworkLoadBalancerReservedIp,
	ipAddresses []networkloadbalancersdk.IpAddress,
) []string {
	if len(desired) == 0 {
		return nil
	}

	currentIDs := map[string]struct{}{}
	for _, ipAddress := range ipAddresses {
		if ipAddress.ReservedIp == nil || ipAddress.ReservedIp.Id == nil {
			continue
		}
		if id := strings.TrimSpace(*ipAddress.ReservedIp.Id); id != "" {
			currentIDs[id] = struct{}{}
		}
	}

	var missing []string
	seenDesired := map[string]struct{}{}
	for _, reservedIP := range desired {
		id := strings.TrimSpace(reservedIP.Id)
		if id == "" {
			continue
		}
		if _, seen := seenDesired[id]; seen {
			continue
		}
		seenDesired[id] = struct{}{}
		if _, ok := currentIDs[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}

func networkLoadBalancerHasReservedIPv6(ipAddresses []networkloadbalancersdk.IpAddress, reservedIPv6ID string) bool {
	for _, ipAddress := range ipAddresses {
		if ipAddress.IpVersion != "" && ipAddress.IpVersion != networkloadbalancersdk.IpVersionIpv6 {
			continue
		}
		if ipAddress.ReservedIp == nil || ipAddress.ReservedIp.Id == nil {
			continue
		}
		if *ipAddress.ReservedIp.Id == reservedIPv6ID {
			return true
		}
	}
	return false
}

func networkLoadBalancerDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		output[namespace] = converted
	}
	return output
}

func copyNetworkLoadBalancerStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func copyNetworkLoadBalancerNestedMap(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		output[namespace] = converted
	}
	return output
}

func stringPointer(value string) *string {
	return &value
}

func networkLoadBalancerStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func handleNetworkLoadBalancerDeleteError(resource *networkloadbalancerv1beta1.NetworkLoadBalancer, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return networkLoadBalancerAmbiguousNotFoundError{
		message:      "NetworkLoadBalancer delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

type networkLoadBalancerAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e networkLoadBalancerAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e networkLoadBalancerAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func wrapNetworkLoadBalancerDeleteConfirmation(
	hooks *NetworkLoadBalancerRuntimeHooks,
	workRequestClient networkLoadBalancerWorkRequestClient,
	workRequestInitErr error,
) {
	if hooks == nil {
		return
	}
	getNetworkLoadBalancer := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate NetworkLoadBalancerServiceClient) NetworkLoadBalancerServiceClient {
		return networkLoadBalancerDeleteConfirmationClient{
			delegate:               delegate,
			getNetworkLoadBalancer: getNetworkLoadBalancer,
			workRequestClient:      workRequestClient,
			workRequestInitErr:     workRequestInitErr,
		}
	})
}

func wrapNetworkLoadBalancerNetworkSecurityGroupUpdates(
	hooks *NetworkLoadBalancerRuntimeHooks,
	updateClient networkLoadBalancerNetworkSecurityGroupsClient,
	updateClientInitErr error,
) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate NetworkLoadBalancerServiceClient) NetworkLoadBalancerServiceClient {
		return networkLoadBalancerNetworkSecurityGroupUpdateClient{
			delegate:            delegate,
			updateClient:        updateClient,
			updateClientInitErr: updateClientInitErr,
		}
	})
}

type networkLoadBalancerDeleteConfirmationClient struct {
	delegate               NetworkLoadBalancerServiceClient
	getNetworkLoadBalancer func(context.Context, networkloadbalancersdk.GetNetworkLoadBalancerRequest) (networkloadbalancersdk.GetNetworkLoadBalancerResponse, error)
	workRequestClient      networkLoadBalancerWorkRequestClient
	workRequestInitErr     error
}

func (c networkLoadBalancerDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c networkLoadBalancerDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
) (bool, error) {
	if networkLoadBalancerHasPendingWriteWorkRequest(resource) {
		response, err := c.delegate.CreateOrUpdate(ctx, resource, ctrl.Request{})
		if err != nil {
			return false, err
		}
		if networkLoadBalancerHasPendingWriteWorkRequest(resource) || response.ShouldRequeue {
			return false, nil
		}
	}
	if err := c.rejectAmbiguousSucceededDeleteWorkRequestConfirmation(ctx, resource); err != nil {
		return false, err
	}
	if networkLoadBalancerPendingDeleteWorkRequestID(resource) == "" {
		if err := c.rejectAmbiguousDeleteConfirmation(ctx, resource); err != nil {
			return false, err
		}
	}
	return c.delegate.Delete(ctx, resource)
}

type networkLoadBalancerNetworkSecurityGroupUpdateClient struct {
	delegate            NetworkLoadBalancerServiceClient
	updateClient        networkLoadBalancerNetworkSecurityGroupsClient
	updateClientInitErr error
}

func (c networkLoadBalancerNetworkSecurityGroupUpdateClient) CreateOrUpdate(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	hadPendingWriteWorkRequest := networkLoadBalancerHasPendingWriteWorkRequest(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || response.ShouldRequeue || resource == nil || hadPendingWriteWorkRequest {
		return response, err
	}
	if !networkLoadBalancerNetworkSecurityGroupsNeedUpdate(resource.Spec.NetworkSecurityGroupIds, resource.Status.NetworkSecurityGroupIds) {
		return response, nil
	}
	return c.updateNetworkSecurityGroups(ctx, resource, req)
}

func (c networkLoadBalancerNetworkSecurityGroupUpdateClient) Delete(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c networkLoadBalancerNetworkSecurityGroupUpdateClient) updateNetworkSecurityGroups(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.updateClientInitErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("initialize NetworkLoadBalancer OCI client for network security group update: %w", c.updateClientInitErr)
	}
	if c.updateClient == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("NetworkLoadBalancer OCI client for network security group update is not configured")
	}
	networkLoadBalancerID := trackedNetworkLoadBalancerID(resource)
	if networkLoadBalancerID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("NetworkLoadBalancer network security group update requires a tracked networkLoadBalancerId")
	}

	updateResponse, err := c.updateClient.UpdateNetworkSecurityGroups(ctx, networkloadbalancersdk.UpdateNetworkSecurityGroupsRequest{
		NetworkLoadBalancerId: stringPointer(networkLoadBalancerID),
		UpdateNetworkSecurityGroupsDetails: networkloadbalancersdk.UpdateNetworkSecurityGroupsDetails{
			NetworkSecurityGroupIds: append([]string(nil), resource.Spec.NetworkSecurityGroupIds...),
		},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, updateResponse)

	workRequestID := strings.TrimSpace(networkLoadBalancerStringValue(updateResponse.OpcWorkRequestId))
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("NetworkLoadBalancer network security group update did not return an opc-work-request-id")
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         fmt.Sprintf("NetworkLoadBalancer update work request %s is pending", workRequestID),
	}, loggerutil.OSOKLogger{})
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c networkLoadBalancerDeleteConfirmationClient) rejectAmbiguousSucceededDeleteWorkRequestConfirmation(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
) error {
	workRequestID := networkLoadBalancerPendingDeleteWorkRequestID(resource)
	if workRequestID == "" {
		return nil
	}

	workRequest, err := getNetworkLoadBalancerWorkRequest(ctx, c.workRequestClient, c.workRequestInitErr, workRequestID)
	if err != nil {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return err
	}
	current, err := networkLoadBalancerWorkRequestFromAny(workRequest)
	if err != nil {
		return err
	}
	normalizedClass, err := networkLoadBalancerWorkRequestAsyncAdapter.Normalize(string(current.Status))
	if err != nil {
		return err
	}
	if normalizedClass != shared.OSOKAsyncClassSucceeded {
		return nil
	}
	return c.rejectAmbiguousDeleteConfirmation(ctx, resource)
}

func (c networkLoadBalancerDeleteConfirmationClient) rejectAmbiguousDeleteConfirmation(
	ctx context.Context,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
) error {
	if c.getNetworkLoadBalancer == nil || resource == nil {
		return nil
	}
	networkLoadBalancerID := trackedNetworkLoadBalancerID(resource)
	if networkLoadBalancerID == "" {
		return nil
	}
	_, err := c.getNetworkLoadBalancer(ctx, networkloadbalancersdk.GetNetworkLoadBalancerRequest{
		NetworkLoadBalancerId: stringPointer(networkLoadBalancerID),
	})
	return networkLoadBalancerAmbiguousDeleteConfirmationError(resource, err)
}

func networkLoadBalancerAmbiguousDeleteConfirmationError(
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	err error,
) error {
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("NetworkLoadBalancer delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func trackedNetworkLoadBalancerID(resource *networkloadbalancerv1beta1.NetworkLoadBalancer) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func networkLoadBalancerPendingDeleteWorkRequestID(resource *networkloadbalancerv1beta1.NetworkLoadBalancer) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func networkLoadBalancerHasPendingWriteWorkRequest(resource *networkloadbalancerv1beta1.NetworkLoadBalancer) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func newNetworkLoadBalancerServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client networkLoadBalancerRuntimeOCIClient,
) NetworkLoadBalancerServiceClient {
	manager := &NetworkLoadBalancerServiceManager{Log: log}
	hooks := newNetworkLoadBalancerRuntimeHooksWithOCIClient(client)
	applyNetworkLoadBalancerRuntimeHooks(&hooks, client, nil)
	delegate := defaultNetworkLoadBalancerServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkloadbalancerv1beta1.NetworkLoadBalancer](
			buildNetworkLoadBalancerGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapNetworkLoadBalancerGeneratedClient(hooks, delegate)
}

func newNetworkLoadBalancerRuntimeHooksWithOCIClient(client networkLoadBalancerRuntimeOCIClient) NetworkLoadBalancerRuntimeHooks {
	return NetworkLoadBalancerRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*networkloadbalancerv1beta1.NetworkLoadBalancer]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*networkloadbalancerv1beta1.NetworkLoadBalancer]{},
		StatusHooks:     generatedruntime.StatusHooks[*networkloadbalancerv1beta1.NetworkLoadBalancer]{},
		ParityHooks:     generatedruntime.ParityHooks[*networkloadbalancerv1beta1.NetworkLoadBalancer]{},
		Async:           generatedruntime.AsyncHooks[*networkloadbalancerv1beta1.NetworkLoadBalancer]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*networkloadbalancerv1beta1.NetworkLoadBalancer]{},
		Create: runtimeOperationHooks[networkloadbalancersdk.CreateNetworkLoadBalancerRequest, networkloadbalancersdk.CreateNetworkLoadBalancerResponse]{
			Fields: networkLoadBalancerCreateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.CreateNetworkLoadBalancerRequest) (networkloadbalancersdk.CreateNetworkLoadBalancerResponse, error) {
				if client == nil {
					return networkloadbalancersdk.CreateNetworkLoadBalancerResponse{}, fmt.Errorf("NetworkLoadBalancer OCI client is nil")
				}
				return client.CreateNetworkLoadBalancer(ctx, request)
			},
		},
		Get: runtimeOperationHooks[networkloadbalancersdk.GetNetworkLoadBalancerRequest, networkloadbalancersdk.GetNetworkLoadBalancerResponse]{
			Fields: networkLoadBalancerGetFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.GetNetworkLoadBalancerRequest) (networkloadbalancersdk.GetNetworkLoadBalancerResponse, error) {
				if client == nil {
					return networkloadbalancersdk.GetNetworkLoadBalancerResponse{}, fmt.Errorf("NetworkLoadBalancer OCI client is nil")
				}
				return client.GetNetworkLoadBalancer(ctx, request)
			},
		},
		List: runtimeOperationHooks[networkloadbalancersdk.ListNetworkLoadBalancersRequest, networkloadbalancersdk.ListNetworkLoadBalancersResponse]{
			Fields: networkLoadBalancerListFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.ListNetworkLoadBalancersRequest) (networkloadbalancersdk.ListNetworkLoadBalancersResponse, error) {
				if client == nil {
					return networkloadbalancersdk.ListNetworkLoadBalancersResponse{}, fmt.Errorf("NetworkLoadBalancer OCI client is nil")
				}
				return client.ListNetworkLoadBalancers(ctx, request)
			},
		},
		Update: runtimeOperationHooks[networkloadbalancersdk.UpdateNetworkLoadBalancerRequest, networkloadbalancersdk.UpdateNetworkLoadBalancerResponse]{
			Fields: networkLoadBalancerUpdateFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.UpdateNetworkLoadBalancerRequest) (networkloadbalancersdk.UpdateNetworkLoadBalancerResponse, error) {
				if client == nil {
					return networkloadbalancersdk.UpdateNetworkLoadBalancerResponse{}, fmt.Errorf("NetworkLoadBalancer OCI client is nil")
				}
				return client.UpdateNetworkLoadBalancer(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[networkloadbalancersdk.DeleteNetworkLoadBalancerRequest, networkloadbalancersdk.DeleteNetworkLoadBalancerResponse]{
			Fields: networkLoadBalancerDeleteFields(),
			Call: func(ctx context.Context, request networkloadbalancersdk.DeleteNetworkLoadBalancerRequest) (networkloadbalancersdk.DeleteNetworkLoadBalancerResponse, error) {
				if client == nil {
					return networkloadbalancersdk.DeleteNetworkLoadBalancerResponse{}, fmt.Errorf("NetworkLoadBalancer OCI client is nil")
				}
				return client.DeleteNetworkLoadBalancer(ctx, request)
			},
		},
		WrapGeneratedClient: []func(NetworkLoadBalancerServiceClient) NetworkLoadBalancerServiceClient{},
	}
}

func networkLoadBalancerCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateNetworkLoadBalancerDetails", RequestName: "CreateNetworkLoadBalancerDetails", Contribution: "body"},
	}
}

func networkLoadBalancerGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		networkLoadBalancerIDField(),
	}
}

func networkLoadBalancerListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		networkLoadBalancerCompartmentIDField(),
		networkLoadBalancerDisplayNameField(),
	}
}

func networkLoadBalancerUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "UpdateNetworkLoadBalancerDetails", RequestName: "UpdateNetworkLoadBalancerDetails", Contribution: "body"},
		networkLoadBalancerIDField(),
	}
}

func networkLoadBalancerDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		networkLoadBalancerIDField(),
	}
}

func networkLoadBalancerIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "NetworkLoadBalancerId",
		RequestName:      "networkLoadBalancerId",
		Contribution:     "path",
		PreferResourceID: true,
		LookupPaths:      []string{"status.id", "status.status.ocid"},
	}
}

func networkLoadBalancerCompartmentIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "CompartmentId",
		RequestName:  "compartmentId",
		Contribution: "query",
		LookupPaths:  []string{"status.compartmentId", "spec.compartmentId"},
	}
}

func networkLoadBalancerDisplayNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "DisplayName",
		RequestName:  "displayName",
		Contribution: "query",
		LookupPaths:  []string{"status.displayName", "spec.displayName"},
	}
}

func listNetworkLoadBalancersAllPages(
	call func(context.Context, networkloadbalancersdk.ListNetworkLoadBalancersRequest) (networkloadbalancersdk.ListNetworkLoadBalancersResponse, error),
) func(context.Context, networkloadbalancersdk.ListNetworkLoadBalancersRequest) (networkloadbalancersdk.ListNetworkLoadBalancersResponse, error) {
	return func(ctx context.Context, request networkloadbalancersdk.ListNetworkLoadBalancersRequest) (networkloadbalancersdk.ListNetworkLoadBalancersResponse, error) {
		var combined networkloadbalancersdk.ListNetworkLoadBalancersResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return networkloadbalancersdk.ListNetworkLoadBalancersResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}
