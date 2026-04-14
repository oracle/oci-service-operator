/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nodepool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	newNodePoolServiceClient = func(manager *NodePoolServiceManager) NodePoolServiceClient {
		sdkClient, err := containerenginesdk.NewContainerEngineClientWithConfigurationProvider(manager.Provider)
		config := generatedruntime.Config[*containerenginev1beta1.NodePool]{
			Kind:    "NodePool",
			SDKName: "NodePool",
			Log:     manager.Log,
			BuildCreateBody: func(ctx context.Context, resource *containerenginev1beta1.NodePool, namespace string) (any, error) {
				details, err := buildNodePoolCreateDetails(ctx, resource, namespace)
				if err != nil {
					return details, err
				}
				logNodePoolSDKBody(manager.Log, "create", resource, details)
				return details, nil
			},
			BuildUpdateBody: func(
				ctx context.Context,
				resource *containerenginev1beta1.NodePool,
				namespace string,
				currentResponse any,
			) (any, bool, error) {
				details, updateNeeded, err := buildNodePoolUpdateBody(ctx, resource, namespace, currentResponse)
				if err != nil {
					return details, updateNeeded, err
				}
				logNodePoolSDKBody(manager.Log, "update", resource, details)
				return details, updateNeeded, nil
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
						"sshPublicKey",
					},
					ForceNew: []string{
						"clusterId",
						"compartmentId",
						"nodeImageName",
					},
					ConflictsWith: map[string][]string{},
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
	logNodePoolNormalizeState(managerLogForResource(resource), "pre-normalize", "create", resource.Name, details.SubnetIds, createNodeConfigNsgIDs(details.NodeConfigDetails))
	normalizeNodePoolCreateDetails(resource.Spec, &details)
	logNodePoolNormalizeState(managerLogForResource(resource), "post-normalize", "create", resource.Name, details.SubnetIds, createNodeConfigNsgIDs(details.NodeConfigDetails))
	if err := validateNodePoolCreateDetails(details); err != nil {
		return containerenginesdk.CreateNodePoolDetails{}, err
	}

	return details, nil
}

func logNodePoolSDKBody(log loggerutil.OSOKLogger, operation string, resource *containerenginev1beta1.NodePool, body any) {
	payload, err := nodePoolSDKHTTPRequestBody(operation, body)
	if err != nil {
		log.ErrorLog(err, "failed to build NodePool SDK HTTP request body", "operation", operation)
		return
	}

	resourceName := ""
	if resource != nil {
		resourceName = resource.Name
	}

	log.InfoLog(
		"prepared NodePool SDK HTTP request body",
		"operation", operation,
		"name", resourceName,
		"body", payload,
	)
}

func logNodePoolNormalizeState(log loggerutil.OSOKLogger, stage string, operation string, resourceName string, subnetIDs []string, nsgIDs []string) {
	log.InfoLog(
		"NodePool request state",
		"stage", stage,
		"operation", operation,
		"name", resourceName,
		"subnetIdsIsNil", fmt.Sprintf("%t", subnetIDs == nil),
		"subnetIdsLen", fmt.Sprintf("%d", len(subnetIDs)),
		"nsgIdsIsNil", fmt.Sprintf("%t", nsgIDs == nil),
		"nsgIdsLen", fmt.Sprintf("%d", len(nsgIDs)),
	)
}

func managerLogForResource(resource *containerenginev1beta1.NodePool) loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("nodepool-runtime")}
}

func sdkRequestLog() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("nodepool-sdk-request")}
}

func createNodeConfigNsgIDs(details *containerenginesdk.CreateNodePoolNodeConfigDetails) []string {
	if details == nil {
		return nil
	}
	return details.NsgIds
}

func updateNodeConfigNsgIDs(details *containerenginesdk.UpdateNodePoolNodeConfigDetails) []string {
	if details == nil {
		return nil
	}
	return details.NsgIds
}

func nodePoolSDKHTTPRequestBody(operation string, body any) (string, error) {
	const debugNodePoolID = "ocid1.nodepool.oc1..debug"

	var (
		httpRequest http.Request
		err         error
	)

	switch operation {
	case "create":
		request := containerenginesdk.CreateNodePoolRequest{
			CreateNodePoolDetails: body.(containerenginesdk.CreateNodePoolDetails),
		}
		logNodePoolNormalizeState(sdkRequestLog(), "request-struct", "create", "", request.CreateNodePoolDetails.SubnetIds, createNodeConfigNsgIDs(request.CreateNodePoolDetails.NodeConfigDetails))
		httpRequest, err = request.HTTPRequest(http.MethodPost, "/nodePools", nil, nil)
	case "update":
		request := containerenginesdk.UpdateNodePoolRequest{
			NodePoolId:            common.String(debugNodePoolID),
			UpdateNodePoolDetails: body.(containerenginesdk.UpdateNodePoolDetails),
		}
		logNodePoolNormalizeState(sdkRequestLog(), "request-struct", "update", "", request.UpdateNodePoolDetails.SubnetIds, updateNodeConfigNsgIDs(request.UpdateNodePoolDetails.NodeConfigDetails))
		httpRequest, err = request.HTTPRequest(http.MethodPut, "/nodePools/"+debugNodePoolID, nil, nil)
	default:
		return "", fmt.Errorf("unsupported NodePool SDK body log operation %q", operation)
	}
	if err != nil {
		return "", err
	}
	if httpRequest.Body == nil {
		return "", nil
	}

	payload, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		return "", err
	}
	return string(payload), nil
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
	logNodePoolNormalizeState(managerLogForResource(resource), "pre-normalize", "update", resource.Name, details.SubnetIds, updateNodeConfigNsgIDs(details.NodeConfigDetails))
	normalizeNodePoolUpdateDetails(resource.Spec, &details)
	logNodePoolNormalizeState(managerLogForResource(resource), "post-normalize", "update", resource.Name, details.SubnetIds, updateNodeConfigNsgIDs(details.NodeConfigDetails))
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

	normalizeNodePoolPlacementFields(&details.SubnetIds, &details.QuantityPerSubnet)
	normalizeNodePoolCreateNodeConfigDetails(spec.NodeConfigDetails, details.NodeConfigDetails)
}

func normalizeNodePoolUpdateDetails(
	spec containerenginev1beta1.NodePoolSpec,
	details *containerenginesdk.UpdateNodePoolDetails,
) {
	if details == nil {
		return
	}

	normalizeNodePoolPlacementFields(&details.SubnetIds, &details.QuantityPerSubnet)
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

func normalizeNodePoolPlacementFields(subnetIDs *[]string, quantityPerSubnet **int) {
	if subnetIDs != nil && len(*subnetIDs) == 0 {
		*subnetIDs = nil
	}
	if subnetIDs != nil {
		*subnetIDs = nil
	}
	if quantityPerSubnet != nil {
		*quantityPerSubnet = nil
	}
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
	hasNodeConfig := details.NodeConfigDetails != nil &&
		details.NodeConfigDetails.Size != nil &&
		len(details.NodeConfigDetails.PlacementConfigs) > 0
	if !hasNodeConfig {
		return fmt.Errorf("nodepool create request must set nodeConfigDetails; deprecated subnetIds placement is no longer supported")
	}

	return validateNodePoolCommonDetails(details.NodeSourceDetails, details.NodeConfigDetails)
}

func validateNodePoolUpdateDetails(details containerenginesdk.UpdateNodePoolDetails) error {
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
