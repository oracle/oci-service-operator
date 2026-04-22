/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nodepool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerNodePoolRuntimeHooksMutator(func(_ *NodePoolServiceManager, hooks *NodePoolRuntimeHooks) {
		applyNodePoolRuntimeHooks(hooks)
	})
}

func applyNodePoolRuntimeHooks(hooks *NodePoolRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newNodePoolRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *containerenginev1beta1.NodePool, namespace string) (any, error) {
		return buildNodePoolCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *containerenginev1beta1.NodePool,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildNodePoolUpdateBody(ctx, resource, namespace, currentResponse)
	}
	hooks.Create.Call = wrapNodePoolOperationCall(hooks.Create.Call, sanitizeCreateNodePoolRequest)
	hooks.Update.Call = wrapNodePoolOperationCall(hooks.Update.Call, sanitizeUpdateNodePoolRequest)
}

func wrapNodePoolOperationCall[Req any, Resp any](
	call func(context.Context, Req) (Resp, error),
	mutate func(*Req),
) func(context.Context, Req) (Resp, error) {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request Req) (Resp, error) {
		if mutate != nil {
			mutate(&request)
		}
		return call(ctx, request)
	}
}

func buildNodePoolCreateDetails(
	ctx context.Context,
	resource *containerenginev1beta1.NodePool,
	namespace string,
) (containerenginesdk.CreateNodePoolDetails, error) {
	if resource == nil {
		return containerenginesdk.CreateNodePoolDetails{}, fmt.Errorf("nodepool resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return containerenginesdk.CreateNodePoolDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return containerenginesdk.CreateNodePoolDetails{}, fmt.Errorf("marshal resolved nodepool spec: %w", err)
	}

	var details containerenginesdk.CreateNodePoolDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return containerenginesdk.CreateNodePoolDetails{}, fmt.Errorf("decode nodepool create request body: %w", err)
	}
	applyNodePoolSpecPolymorphicValues(resource.Spec, &details, nil)
	normalizeNodePoolCreateDetails(resource.Spec, &details)
	if err := validateNodePoolCreateDetails(details); err != nil {
		return containerenginesdk.CreateNodePoolDetails{}, err
	}

	return details, nil
}

func sanitizeCreateNodePoolRequest(req *containerenginesdk.CreateNodePoolRequest) {
	if req == nil {
		return
	}

	if req.CreateNodePoolDetails.NodeConfigDetails != nil {
		req.CreateNodePoolDetails.SubnetIds = nil
	}
}

func sanitizeUpdateNodePoolRequest(req *containerenginesdk.UpdateNodePoolRequest) {
	if req == nil {
		return
	}

	if req.UpdateNodePoolDetails.NodeConfigDetails != nil {
		req.UpdateNodePoolDetails.SubnetIds = nil
	}
}

func buildNodePoolUpdateBody(
	ctx context.Context,
	resource *containerenginev1beta1.NodePool,
	namespace string,
	currentResponse any,
) (containerenginesdk.UpdateNodePoolDetails, bool, error) {
	details, err := buildNodePoolUpdateDetails(ctx, resource, namespace)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, false, err
	}

	preservedEmptyArrays := nodePoolPreservedEmptyArrayPaths(resource.Spec)

	desiredValues, err := nodePoolJSONMap(details, preservedEmptyArrays)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, false, fmt.Errorf("project desired nodepool update body: %w", err)
	}
	if len(desiredValues) == 0 {
		return details, false, nil
	}

	currentDetails, err := buildCurrentNodePoolUpdateDetails(currentResponse)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, false, err
	}
	currentValues, err := nodePoolJSONMap(currentDetails, preservedEmptyArrays)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, false, fmt.Errorf("project current nodepool update body: %w", err)
	}

	return details, !nodePoolMapSubsetEqual(desiredValues, currentValues), nil
}

func buildNodePoolUpdateDetails(
	ctx context.Context,
	resource *containerenginev1beta1.NodePool,
	namespace string,
) (containerenginesdk.UpdateNodePoolDetails, error) {
	if resource == nil {
		return containerenginesdk.UpdateNodePoolDetails{}, fmt.Errorf("nodepool resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, fmt.Errorf("marshal resolved nodepool spec: %w", err)
	}

	var details containerenginesdk.UpdateNodePoolDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, fmt.Errorf("decode nodepool update request body: %w", err)
	}
	applyNodePoolSpecPolymorphicValues(resource.Spec, nil, &details)
	normalizeNodePoolUpdateDetails(resource.Spec, &details)
	if err := validateNodePoolUpdateDetails(details); err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, err
	}

	return details, nil
}

func buildCurrentNodePoolUpdateDetails(currentResponse any) (containerenginesdk.UpdateNodePoolDetails, error) {
	body, err := nodePoolRuntimeResponseBody(currentResponse)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, fmt.Errorf("marshal current nodepool response: %w", err)
	}

	var details containerenginesdk.UpdateNodePoolDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return containerenginesdk.UpdateNodePoolDetails{}, fmt.Errorf("decode current nodepool update body: %w", err)
	}
	normalizeObservedNodePoolUpdateDetails(&details)

	return details, nil
}

func normalizeNodePoolCreateDetails(
	spec containerenginev1beta1.NodePoolSpec,
	details *containerenginesdk.CreateNodePoolDetails,
) {
	if details == nil {
		return
	}

	normalizeNodePoolPlacementFields(
		nodePoolSpecUsesDeprecatedPlacement(spec),
		nodePoolSpecUsesNodeConfigDetails(spec),
		&details.SubnetIds,
		&details.QuantityPerSubnet,
		func() { details.NodeConfigDetails = nil },
	)
	normalizeNodePoolCreateNodeConfigDetails(spec.NodeConfigDetails, details.NodeConfigDetails)
}

func normalizeNodePoolUpdateDetails(
	spec containerenginev1beta1.NodePoolSpec,
	details *containerenginesdk.UpdateNodePoolDetails,
) {
	if details == nil {
		return
	}

	normalizeNodePoolPlacementFields(
		nodePoolSpecUsesDeprecatedPlacement(spec),
		nodePoolSpecUsesNodeConfigDetails(spec),
		&details.SubnetIds,
		&details.QuantityPerSubnet,
		func() { details.NodeConfigDetails = nil },
	)
	normalizeNodePoolUpdateNodeConfigDetails(spec.NodeConfigDetails, details.NodeConfigDetails)
}

func normalizeObservedNodePoolUpdateDetails(details *containerenginesdk.UpdateNodePoolDetails) {
	if details == nil {
		return
	}

	if len(details.SubnetIds) == 0 {
		details.SubnetIds = nil
	}
}

func normalizeNodePoolCreateNodeConfigDetails(
	spec containerenginev1beta1.NodePoolNodeConfigDetails,
	details *containerenginesdk.CreateNodePoolNodeConfigDetails,
) {
	if details == nil {
		return
	}

	normalizeNodePoolNodeConfigDetails(spec, details.PlacementConfigs, &details.NsgIds)
}

func normalizeNodePoolUpdateNodeConfigDetails(
	spec containerenginev1beta1.NodePoolNodeConfigDetails,
	details *containerenginesdk.UpdateNodePoolNodeConfigDetails,
) {
	if details == nil {
		return
	}

	normalizeNodePoolNodeConfigDetails(spec, details.PlacementConfigs, &details.NsgIds)
}

func normalizeNodePoolNodeConfigDetails(
	spec containerenginev1beta1.NodePoolNodeConfigDetails,
	details []containerenginesdk.NodePoolPlacementConfigDetails,
	nsgIDs *[]string,
) {
	if nsgIDs != nil && len(*nsgIDs) == 0 && spec.NsgIds == nil {
		*nsgIDs = nil
	}

	for i := range details {
		if details[i].PreemptibleNodeConfig == nil || details[i].PreemptibleNodeConfig.PreemptionAction != nil {
			continue
		}
		details[i].PreemptibleNodeConfig = nil
	}
}

func nodePoolSpecPreemptibleNodeConfigIsEmpty(
	spec containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfigPreemptibleNodeConfig,
) bool {
	action := spec.PreemptionAction
	return strings.TrimSpace(action.Type) == "" && !action.IsPreserveBootVolume
}

func normalizeNodePoolPlacementFields(
	hasDeprecatedPlacement bool,
	hasNodeConfigDetails bool,
	subnetIDs *[]string,
	quantityPerSubnet **int,
	clearNodeConfigDetails func(),
) {
	if subnetIDs != nil && len(*subnetIDs) == 0 {
		*subnetIDs = nil
	}

	switch {
	case hasDeprecatedPlacement && !hasNodeConfigDetails:
		if clearNodeConfigDetails != nil {
			clearNodeConfigDetails()
		}
	case hasNodeConfigDetails && !hasDeprecatedPlacement:
		if subnetIDs != nil {
			*subnetIDs = nil
		}
		if quantityPerSubnet != nil {
			*quantityPerSubnet = nil
		}
	case !hasDeprecatedPlacement && !hasNodeConfigDetails:
		if clearNodeConfigDetails != nil {
			clearNodeConfigDetails()
		}
	}
}

func nodePoolSpecUsesDeprecatedPlacement(spec containerenginev1beta1.NodePoolSpec) bool {
	return len(spec.SubnetIds) > 0 || spec.QuantityPerSubnet != 0
}

func nodePoolSpecUsesNodeConfigDetails(spec containerenginev1beta1.NodePoolSpec) bool {
	details := spec.NodeConfigDetails

	if details.Size != 0 ||
		len(details.PlacementConfigs) > 0 ||
		len(details.NsgIds) > 0 ||
		strings.TrimSpace(details.KmsKeyId) != "" ||
		details.IsPvEncryptionInTransitEnabled ||
		len(details.FreeformTags) > 0 ||
		len(details.DefinedTags) > 0 {
		return true
	}

	return nodePoolSpecUsesPodNetworkOptionDetails(details.NodePoolPodNetworkOptionDetails)
}

func nodePoolSpecUsesPodNetworkOptionDetails(
	details containerenginev1beta1.NodePoolNodeConfigDetailsNodePoolPodNetworkOptionDetails,
) bool {
	return strings.TrimSpace(details.CniType) != "" ||
		len(details.PodSubnetIds) > 0 ||
		details.MaxPodsPerNode != 0 ||
		len(details.PodNsgIds) > 0
}

func nodePoolRuntimeResponseBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case containerenginesdk.NodePool:
		return current, nil
	case *containerenginesdk.NodePool:
		if current == nil {
			return nil, fmt.Errorf("current NodePool response is nil")
		}
		return *current, nil
	case containerenginesdk.NodePoolSummary:
		return current, nil
	case *containerenginesdk.NodePoolSummary:
		if current == nil {
			return nil, fmt.Errorf("current NodePool response is nil")
		}
		return *current, nil
	case containerenginesdk.GetNodePoolResponse:
		return current.NodePool, nil
	case *containerenginesdk.GetNodePoolResponse:
		if current == nil {
			return nil, fmt.Errorf("current NodePool response is nil")
		}
		return current.NodePool, nil
	default:
		return nil, fmt.Errorf("unexpected current NodePool response type %T", currentResponse)
	}
}

func validateNodePoolCreateDetails(details containerenginesdk.CreateNodePoolDetails) error {
	hasSubnetIDs := len(details.SubnetIds) > 0
	hasNodeConfig := details.NodeConfigDetails != nil
	switch {
	case hasSubnetIDs && hasNodeConfig:
		return fmt.Errorf("nodepool create request must not set both subnetIds and nodeConfigDetails")
	case !hasSubnetIDs && !hasNodeConfig:
		return fmt.Errorf("nodepool create request must set either subnetIds or nodeConfigDetails")
	case details.QuantityPerSubnet != nil && !hasSubnetIDs:
		return fmt.Errorf("nodepool create request requires subnetIds when quantityPerSubnet is set")
	}

	return validateNodePoolCommonDetails(details.NodeSourceDetails, details.NodeConfigDetails)
}

func validateNodePoolUpdateDetails(details containerenginesdk.UpdateNodePoolDetails) error {
	if len(details.SubnetIds) > 0 && details.NodeConfigDetails != nil {
		return fmt.Errorf("nodepool update request must not set both subnetIds and nodeConfigDetails")
	}
	if details.QuantityPerSubnet != nil && details.NodeConfigDetails != nil {
		return fmt.Errorf("nodepool update request must not set both quantityPerSubnet and nodeConfigDetails")
	}

	return validateNodePoolCommonDetails(details.NodeSourceDetails, details.NodeConfigDetails)
}

func validateNodePoolCommonDetails(nodeSourceDetails containerenginesdk.NodeSourceDetails, nodeConfigDetails any) error {
	switch nodeSourceDetails.(type) {
	case nil, containerenginesdk.NodeSourceViaImageDetails:
	default:
		return fmt.Errorf("unsupported nodeSourceDetails type %T", nodeSourceDetails)
	}

	switch details := nodeConfigDetails.(type) {
	case nil:
		return nil
	case *containerenginesdk.CreateNodePoolNodeConfigDetails:
		if details == nil {
			return nil
		}
		if err := validateNodePoolPodNetworkOptionDetails(details.NodePoolPodNetworkOptionDetails); err != nil {
			return err
		}
		return validateNodePoolPlacementConfigs(details.PlacementConfigs)
	case *containerenginesdk.UpdateNodePoolNodeConfigDetails:
		if details == nil {
			return nil
		}
		if err := validateNodePoolPodNetworkOptionDetails(details.NodePoolPodNetworkOptionDetails); err != nil {
			return err
		}
		return validateNodePoolPlacementConfigs(details.PlacementConfigs)
	default:
		return fmt.Errorf("unexpected nodeConfigDetails type %T", nodeConfigDetails)
	}
}

func validateNodePoolPodNetworkOptionDetails(details containerenginesdk.NodePoolPodNetworkOptionDetails) error {
	switch details.(type) {
	case nil,
		containerenginesdk.FlannelOverlayNodePoolPodNetworkOptionDetails,
		containerenginesdk.OciVcnIpNativeNodePoolPodNetworkOptionDetails:
		return nil
	default:
		return fmt.Errorf("unsupported nodeConfigDetails.nodePoolPodNetworkOptionDetails type %T", details)
	}
}

func applyNodePoolSpecPolymorphicValues(
	spec containerenginev1beta1.NodePoolSpec,
	createDetails *containerenginesdk.CreateNodePoolDetails,
	updateDetails *containerenginesdk.UpdateNodePoolDetails,
) {
	switch {
	case createDetails != nil && createDetails.NodeConfigDetails != nil:
		applyNodePoolPlacementConfigSpecValues(spec.NodeConfigDetails.PlacementConfigs, createDetails.NodeConfigDetails.PlacementConfigs)
	case updateDetails != nil && updateDetails.NodeConfigDetails != nil:
		applyNodePoolPlacementConfigSpecValues(spec.NodeConfigDetails.PlacementConfigs, updateDetails.NodeConfigDetails.PlacementConfigs)
	}
}

func applyNodePoolPlacementConfigSpecValues(
	specs []containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfig,
	details []containerenginesdk.NodePoolPlacementConfigDetails,
) {
	for i := 0; i < len(specs) && i < len(details); i++ {
		actionSpec := specs[i].PreemptibleNodeConfig.PreemptionAction
		if !strings.EqualFold(strings.TrimSpace(actionSpec.Type), "TERMINATE") {
			continue
		}
		if details[i].PreemptibleNodeConfig == nil {
			details[i].PreemptibleNodeConfig = &containerenginesdk.PreemptibleNodeConfigDetails{}
		}
		details[i].PreemptibleNodeConfig.PreemptionAction = containerenginesdk.TerminatePreemptionAction{
			IsPreserveBootVolume: common.Bool(actionSpec.IsPreserveBootVolume),
		}
	}
}

func validateNodePoolPlacementConfigs(configs []containerenginesdk.NodePoolPlacementConfigDetails) error {
	for i, config := range configs {
		if config.PreemptibleNodeConfig == nil || config.PreemptibleNodeConfig.PreemptionAction == nil {
			continue
		}
		switch config.PreemptibleNodeConfig.PreemptionAction.(type) {
		case containerenginesdk.TerminatePreemptionAction:
		default:
			return fmt.Errorf(
				"unsupported nodeConfigDetails.placementConfigs[%d].preemptibleNodeConfig.preemptionAction type %T",
				i,
				config.PreemptibleNodeConfig.PreemptionAction,
			)
		}
	}
	return nil
}

func nodePoolJSONMap(value any, preservedEmptyArrays map[string]struct{}) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	pruned, ok := pruneNodePoolJSONValue(decoded, "", preservedEmptyArrays)
	if !ok {
		return nil, nil
	}

	values, ok := pruned.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("nodepool JSON projection is %T, want map[string]any", pruned)
	}
	return values, nil
}

func nodePoolPreservedEmptyArrayPaths(spec containerenginev1beta1.NodePoolSpec) map[string]struct{} {
	if spec.NodeConfigDetails.NsgIds == nil || len(spec.NodeConfigDetails.NsgIds) != 0 {
		return nil
	}

	return map[string]struct{}{
		"nodeConfigDetails.nsgIds": {},
	}
}

func pruneNodePoolJSONValue(value any, path string, preservedEmptyArrays map[string]struct{}) (any, bool) {
	switch concrete := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			prunedChild, ok := pruneNodePoolJSONValue(child, childPath, preservedEmptyArrays)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case []any:
		pruned := make([]any, 0, len(concrete))
		for _, child := range concrete {
			prunedChild, ok := pruneNodePoolJSONValue(child, path, preservedEmptyArrays)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
		}
		if len(pruned) == 0 {
			if nodePoolPreservesEmptyArrayPath(path, preservedEmptyArrays) {
				return []any{}, true
			}
			return nil, false
		}
		return pruned, true
	case string:
		if strings.TrimSpace(concrete) == "" {
			return nil, false
		}
		return concrete, true
	default:
		return concrete, true
	}
}

func nodePoolPreservesEmptyArrayPath(path string, preservedEmptyArrays map[string]struct{}) bool {
	if len(preservedEmptyArrays) == 0 {
		return false
	}
	_, ok := preservedEmptyArrays[path]
	return ok
}

func nodePoolMapSubsetEqual(want map[string]any, got map[string]any) bool {
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			return false
		}
		if !nodePoolJSONValueEqual(wantValue, gotValue) {
			return false
		}
	}
	return true
}

func nodePoolJSONValueEqual(left any, right any) bool {
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	switch {
	case leftIsMap && rightIsMap:
		return nodePoolMapSubsetEqual(leftMap, rightMap)
	case leftIsMap || rightIsMap:
		return false
	}

	leftSlice, leftIsSlice := left.([]any)
	rightSlice, rightIsSlice := right.([]any)
	switch {
	case leftIsSlice && rightIsSlice:
		if len(leftSlice) != len(rightSlice) {
			return false
		}
		for i := range leftSlice {
			if !nodePoolJSONValueEqual(leftSlice[i], rightSlice[i]) {
				return false
			}
		}
		return true
	case leftIsSlice || rightIsSlice:
		return false
	}

	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
