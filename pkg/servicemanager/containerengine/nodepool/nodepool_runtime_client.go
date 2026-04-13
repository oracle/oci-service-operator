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
	newNodePoolServiceClient = func(manager *NodePoolServiceManager) NodePoolServiceClient {
		sdkClient, err := containerenginesdk.NewContainerEngineClientWithConfigurationProvider(manager.Provider)
		config := generatedruntime.Config[*containerenginev1beta1.NodePool]{
			Kind:    "NodePool",
			SDKName: "NodePool",
			Log:     manager.Log,
			BuildCreateBody: func(ctx context.Context, resource *containerenginev1beta1.NodePool, namespace string) (any, error) {
				return buildNodePoolCreateDetails(ctx, resource, namespace)
			},
			BuildUpdateBody: func(
				ctx context.Context,
				resource *containerenginev1beta1.NodePool,
				namespace string,
				currentResponse any,
			) (any, bool, error) {
				return buildNodePoolUpdateBody(ctx, resource, namespace, currentResponse)
			},
			Semantics: &generatedruntime.Semantics{
				FormalService:     "containerengine",
				FormalSlug:        "nodepool",
				StatusProjection:  "required",
				SecretSideEffects: "none",
				FinalizerPolicy:   "retain-until-confirmed-delete",
				Lifecycle: generatedruntime.LifecycleSemantics{
					ProvisioningStates: []string{"CREATING"},
					UpdatingStates:     []string{"UPDATING"},
					ActiveStates:       []string{"ACTIVE", "INACTIVE", "NEEDS_ATTENTION"},
				},
				Delete: generatedruntime.DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{"DELETING"},
					TerminalStates: []string{"DELETED"},
				},
				List: &generatedruntime.ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"compartmentId", "clusterId", "name", "lifecycleState"},
				},
				Mutation: generatedruntime.MutationSemantics{
					Mutable: []string{
						"definedTags",
						"freeformTags",
						"initialNodeLabels",
						"kubernetesVersion",
						"name",
						"nodeConfigDetails",
						"nodeEvictionNodePoolSettings",
						"nodeMetadata",
						"nodePoolCyclingDetails",
						"nodeShape",
						"nodeShapeConfig",
						"nodeSourceDetails",
						"quantityPerSubnet",
						"sshPublicKey",
						"subnetIds",
					},
					ForceNew: []string{
						"clusterId",
						"compartmentId",
						"nodeImageName",
					},
					ConflictsWith: map[string][]string{
						"nodeConfigDetails": []string{"quantityPerSubnet", "subnetIds"},
						"quantityPerSubnet": []string{"nodeConfigDetails"},
						"subnetIds":         []string{"nodeConfigDetails"},
					},
				},
				Hooks: generatedruntime.HookSet{
					Create: []generatedruntime.Hook{
						{Helper: "tfresource.CreateResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "nodepool", Action: "CREATED"},
					},
					Update: []generatedruntime.Hook{
						{Helper: "tfresource.UpdateResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "nodepool", Action: "UPDATED"},
					},
					Delete: []generatedruntime.Hook{
						{Helper: "tfresource.DeleteResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "nodepool", Action: "DELETED"},
					},
				},
				CreateFollowUp: generatedruntime.FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks: []generatedruntime.Hook{
						{Helper: "tfresource.CreateResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "nodepool", Action: "CREATED"},
					},
				},
				UpdateFollowUp: generatedruntime.FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks: []generatedruntime.Hook{
						{Helper: "tfresource.UpdateResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "nodepool", Action: "UPDATED"},
					},
				},
				DeleteFollowUp: generatedruntime.FollowUpSemantics{
					Strategy: "confirm-delete",
					Hooks: []generatedruntime.Hook{
						{Helper: "tfresource.DeleteResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "nodepool", Action: "DELETED"},
					},
				},
				AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
				Unsupported:         []generatedruntime.UnsupportedSemantic{},
			},
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.CreateNodePoolRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.CreateNodePool(ctx, *request.(*containerenginesdk.CreateNodePoolRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateNodePoolDetails", RequestName: "CreateNodePoolDetails", Contribution: "body"},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.GetNodePoolRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.GetNodePool(ctx, *request.(*containerenginesdk.GetNodePoolRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "NodePoolId", RequestName: "nodePoolId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.ListNodePoolsRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.ListNodePools(ctx, *request.(*containerenginesdk.ListNodePoolsRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
					{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "query"},
					{FieldName: "Name", RequestName: "name", Contribution: "query"},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
					{FieldName: "Page", RequestName: "page", Contribution: "query"},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
					{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.UpdateNodePoolRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.UpdateNodePool(ctx, *request.(*containerenginesdk.UpdateNodePoolRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "NodePoolId", RequestName: "nodePoolId", Contribution: "path", PreferResourceID: true},
					{FieldName: "OverrideEvictionGraceDuration", RequestName: "overrideEvictionGraceDuration", Contribution: "query"},
					{FieldName: "IsForceDeletionAfterOverrideGraceDuration", RequestName: "isForceDeletionAfterOverrideGraceDuration", Contribution: "query"},
					{FieldName: "UpdateNodePoolDetails", RequestName: "UpdateNodePoolDetails", Contribution: "body"},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &containerenginesdk.DeleteNodePoolRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.DeleteNodePool(ctx, *request.(*containerenginesdk.DeleteNodePoolRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "NodePoolId", RequestName: "nodePoolId", Contribution: "path", PreferResourceID: true},
					{FieldName: "OverrideEvictionGraceDuration", RequestName: "overrideEvictionGraceDuration", Contribution: "query"},
					{FieldName: "IsForceDeletionAfterOverrideGraceDuration", RequestName: "isForceDeletionAfterOverrideGraceDuration", Contribution: "query"},
				},
			},
		}
		if err != nil {
			config.InitError = fmt.Errorf("initialize NodePool OCI client: %w", err)
		}
		return defaultNodePoolServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*containerenginev1beta1.NodePool](config),
		}
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
	if err := validateNodePoolCreateDetails(details); err != nil {
		return containerenginesdk.CreateNodePoolDetails{}, err
	}

	return details, nil
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

	desiredValues, err := nodePoolJSONMap(details)
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
	currentValues, err := nodePoolJSONMap(currentDetails)
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

	return details, nil
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

func nodePoolJSONMap(value any) (map[string]any, error) {
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

	pruned, ok := pruneNodePoolJSONValue(decoded)
	if !ok {
		return nil, nil
	}

	values, ok := pruned.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("nodepool JSON projection is %T, want map[string]any", pruned)
	}
	return values, nil
}

func pruneNodePoolJSONValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			prunedChild, ok := pruneNodePoolJSONValue(child)
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
			prunedChild, ok := pruneNodePoolJSONValue(child)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
		}
		if len(pruned) == 0 {
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
