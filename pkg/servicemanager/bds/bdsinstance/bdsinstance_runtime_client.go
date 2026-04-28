/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package bdsinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	bdssdk "github.com/oracle/oci-go-sdk/v65/bds"
	"github.com/oracle/oci-go-sdk/v65/common"
	bdsv1beta1 "github.com/oracle/oci-service-operator/api/bds/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerBdsInstanceRuntimeHooksMutator(func(_ *BdsInstanceServiceManager, hooks *BdsInstanceRuntimeHooks) {
		applyBdsInstanceRuntimeHooks(hooks)
	})
}

func applyBdsInstanceRuntimeHooks(hooks *BdsInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedBdsInstanceRuntimeSemantics()
	hooks.StatusHooks.ProjectStatus = projectBdsInstanceStatus
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *bdsv1beta1.BdsInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildBdsInstanceUpdateBody(resource, currentResponse)
	}
}

func reviewedBdsInstanceRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newBdsInstanceRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
	}
	semantics.AuxiliaryOperations = nil
	semantics.Mutation = generatedruntime.MutationSemantics{
		// Cover the full observed create surface so the generic runtime does not
		// reject BDS node and network projections before the reviewed update
		// builder can apply the service-specific mutable-vs-immutable rules.
		Mutable: []string{
			"compartmentId",
			"displayName",
			"clusterVersion",
			"isHighAvailability",
			"isSecure",
			"nodes",
			"secretId",
			"isSecretReused",
			"networkConfig",
			"bootstrapScriptUrl",
			"freeformTags",
			"definedTags",
			"kmsKeyId",
			"clusterProfile",
			"bdsClusterVersionSummary",
		},
		ConflictsWith: map[string][]string{},
	}
	return semantics
}

func buildBdsInstanceUpdateBody(
	resource *bdsv1beta1.BdsInstance,
	currentResponse any,
) (bdssdk.UpdateBdsInstanceDetails, bool, error) {
	if resource == nil {
		return bdssdk.UpdateBdsInstanceDetails{}, false, fmt.Errorf("bdsinstance resource is nil")
	}

	current, err := bdsInstanceRuntimeBody(currentResponse)
	if err != nil {
		return bdssdk.UpdateBdsInstanceDetails{}, false, err
	}
	if err := validateBdsInstanceObservedCreateOnlyDrift(resource.Spec, current); err != nil {
		return bdssdk.UpdateBdsInstanceDetails{}, false, err
	}

	updateDetails := bdssdk.UpdateBdsInstanceDetails{}
	updateNeeded := false

	if resource.Spec.DisplayName != stringValue(current.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.BootstrapScriptUrl != stringValue(current.BootstrapScriptUrl) {
		updateDetails.BootstrapScriptUrl = common.String(resource.Spec.BootstrapScriptUrl)
		updateNeeded = true
	}

	desiredFreeformTags := normalizedBdsFreeformTags(resource.Spec.FreeformTags)
	if !reflect.DeepEqual(desiredFreeformTags, normalizedBdsFreeformTags(current.FreeformTags)) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags, err := bdsDefinedTagsFromSpec(resource.Spec.DefinedTags)
	if err != nil {
		return bdssdk.UpdateBdsInstanceDetails{}, false, err
	}
	if !reflect.DeepEqual(desiredDefinedTags, normalizedBdsDefinedTags(current.DefinedTags)) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if resource.Spec.KmsKeyId != stringValue(current.KmsKeyId) {
		updateDetails.KmsKeyId = common.String(resource.Spec.KmsKeyId)
		updateNeeded = true
	}

	return updateDetails, updateNeeded, nil
}

func projectBdsInstanceStatus(resource *bdsv1beta1.BdsInstance, response any) error {
	if resource == nil {
		return fmt.Errorf("bdsinstance resource is nil")
	}

	body, err := bdsInstanceStatusBody(response)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal current BdsInstance response: %w", err)
	}

	status := bdsv1beta1.BdsInstanceStatus{
		OsokStatus: resource.Status.OsokStatus,
	}
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("project BdsInstance status: %w", err)
	}

	resource.Status = status
	return nil
}

func bdsInstanceStatusBody(currentResponse any) (any, error) {
	switch current := currentResponse.(type) {
	case bdssdk.BdsInstance:
		return current, nil
	case *bdssdk.BdsInstance:
		if current == nil {
			return nil, fmt.Errorf("current BdsInstance response is nil")
		}
		return *current, nil
	case bdssdk.BdsInstanceSummary:
		return current, nil
	case *bdssdk.BdsInstanceSummary:
		if current == nil {
			return nil, fmt.Errorf("current BdsInstance response is nil")
		}
		return *current, nil
	case bdssdk.GetBdsInstanceResponse:
		return current.BdsInstance, nil
	case *bdssdk.GetBdsInstanceResponse:
		if current == nil {
			return nil, fmt.Errorf("current BdsInstance response is nil")
		}
		return current.BdsInstance, nil
	default:
		return nil, fmt.Errorf("unsupported current BdsInstance response type %T", currentResponse)
	}
}

func bdsInstanceRuntimeBody(currentResponse any) (bdssdk.BdsInstance, error) {
	switch current := currentResponse.(type) {
	case bdssdk.BdsInstance:
		return current, nil
	case *bdssdk.BdsInstance:
		if current == nil {
			return bdssdk.BdsInstance{}, fmt.Errorf("current BdsInstance response is nil")
		}
		return *current, nil
	case bdssdk.BdsInstanceSummary:
		return bdssdk.BdsInstance{
			Id:                                      current.Id,
			CompartmentId:                           current.CompartmentId,
			DisplayName:                             current.DisplayName,
			LifecycleState:                          current.LifecycleState,
			IsHighAvailability:                      current.IsHighAvailability,
			IsSecure:                                current.IsSecure,
			IsCloudSqlConfigured:                    current.IsCloudSqlConfigured,
			IsKafkaConfigured:                       current.IsKafkaConfigured,
			NumberOfNodes:                           current.NumberOfNodes,
			TimeCreated:                             current.TimeCreated,
			NumberOfNodesRequiringMaintenanceReboot: current.NumberOfNodesRequiringMaintenanceReboot,
			ClusterVersion:                          current.ClusterVersion,
			ClusterProfile:                          current.ClusterProfile,
			TimeEarliestCertificateExpiration:       current.TimeEarliestCertificateExpiration,
			FreeformTags:                            current.FreeformTags,
			DefinedTags:                             current.DefinedTags,
		}, nil
	case *bdssdk.BdsInstanceSummary:
		if current == nil {
			return bdssdk.BdsInstance{}, fmt.Errorf("current BdsInstance response is nil")
		}
		return bdsInstanceRuntimeBody(*current)
	case bdssdk.GetBdsInstanceResponse:
		return current.BdsInstance, nil
	case *bdssdk.GetBdsInstanceResponse:
		if current == nil {
			return bdssdk.BdsInstance{}, fmt.Errorf("current BdsInstance response is nil")
		}
		return current.BdsInstance, nil
	default:
		return bdssdk.BdsInstance{}, fmt.Errorf("unsupported current BdsInstance response type %T", currentResponse)
	}
}

func validateBdsInstanceObservedCreateOnlyDrift(spec bdsv1beta1.BdsInstanceSpec, current bdssdk.BdsInstance) error {
	if spec.CompartmentId != stringValue(current.CompartmentId) {
		return fmt.Errorf("BdsInstance requires replacement when compartmentId changes")
	}
	if spec.ClusterVersion != string(current.ClusterVersion) {
		return fmt.Errorf("BdsInstance requires replacement when clusterVersion changes")
	}
	if spec.IsHighAvailability != boolValue(current.IsHighAvailability) {
		return fmt.Errorf("BdsInstance requires replacement when isHighAvailability changes")
	}
	if spec.IsSecure != boolValue(current.IsSecure) {
		return fmt.Errorf("BdsInstance requires replacement when isSecure changes")
	}
	if spec.ClusterProfile != "" && spec.ClusterProfile != string(current.ClusterProfile) {
		return fmt.Errorf("BdsInstance requires replacement when clusterProfile changes")
	}
	if spec.SecretId != "" && spec.SecretId != stringValue(current.SecretId) {
		return fmt.Errorf("BdsInstance requires replacement when secretId changes")
	}
	if spec.IsSecretReused && spec.IsSecretReused != boolValue(current.IsSecretReused) {
		return fmt.Errorf("BdsInstance requires replacement when isSecretReused changes")
	}
	if hasMeaningfulBdsClusterVersionSummary(spec.BdsClusterVersionSummary) &&
		!matchesBdsClusterVersionSummary(spec.BdsClusterVersionSummary, current.BdsClusterVersionSummary) {
		return fmt.Errorf("BdsInstance requires replacement when bdsClusterVersionSummary changes")
	}
	if !matchesBdsNetworkConfig(spec.NetworkConfig, current.NetworkConfig) {
		return fmt.Errorf("BdsInstance requires replacement when networkConfig changes")
	}
	if !matchesBdsNodes(spec.Nodes, current.Nodes) {
		return fmt.Errorf("BdsInstance requires replacement when nodes change")
	}
	return nil
}

func matchesBdsNetworkConfig(
	spec bdsv1beta1.BdsInstanceNetworkConfig,
	current *bdssdk.NetworkConfig,
) bool {
	if spec.CidrBlock == "" && !spec.IsNatGatewayRequired {
		return true
	}
	if current == nil {
		return false
	}
	if spec.CidrBlock != "" && spec.CidrBlock != stringValue(current.CidrBlock) {
		return false
	}
	if spec.IsNatGatewayRequired && spec.IsNatGatewayRequired != boolValue(current.IsNatGatewayRequired) {
		return false
	}
	return true
}

func matchesBdsNodes(specNodes []bdsv1beta1.BdsInstanceNode, currentNodes []bdssdk.Node) bool {
	if len(specNodes) != len(currentNodes) {
		return false
	}

	sortedSpecNodes := append([]bdsv1beta1.BdsInstanceNode(nil), specNodes...)
	sort.Slice(sortedSpecNodes, func(i int, j int) bool {
		return bdsSpecNodeSortKey(sortedSpecNodes[i]) < bdsSpecNodeSortKey(sortedSpecNodes[j])
	})

	sortedCurrentNodes := append([]bdssdk.Node(nil), currentNodes...)
	sort.Slice(sortedCurrentNodes, func(i int, j int) bool {
		return bdsCurrentNodeSortKey(sortedCurrentNodes[i]) < bdsCurrentNodeSortKey(sortedCurrentNodes[j])
	})

	for index := range sortedSpecNodes {
		if !matchesBdsNode(sortedSpecNodes[index], sortedCurrentNodes[index]) {
			return false
		}
	}
	return true
}

func hasMeaningfulBdsClusterVersionSummary(spec bdsv1beta1.BdsInstanceBdsClusterVersionSummary) bool {
	return spec.BdsVersion != "" || spec.OdhVersion != ""
}

func matchesBdsClusterVersionSummary(
	spec bdsv1beta1.BdsInstanceBdsClusterVersionSummary,
	current *bdssdk.BdsClusterVersionSummary,
) bool {
	if !hasMeaningfulBdsClusterVersionSummary(spec) {
		return true
	}
	if current == nil {
		return false
	}
	if spec.BdsVersion != stringValue(current.BdsVersion) {
		return false
	}
	if spec.OdhVersion != "" && spec.OdhVersion != stringValue(current.OdhVersion) {
		return false
	}
	return true
}

func bdsSpecNodeSortKey(node bdsv1beta1.BdsInstanceNode) string {
	return node.NodeType + "|" + node.Shape + "|" + node.SubnetId
}

func bdsCurrentNodeSortKey(node bdssdk.Node) string {
	return string(node.NodeType) + "|" + stringValue(node.Shape) + "|" + stringValue(node.SubnetId)
}

func matchesBdsNode(spec bdsv1beta1.BdsInstanceNode, current bdssdk.Node) bool {
	if spec.NodeType != string(current.NodeType) {
		return false
	}
	if spec.Shape != stringValue(current.Shape) {
		return false
	}
	if spec.SubnetId != stringValue(current.SubnetId) {
		return false
	}
	if spec.BlockVolumeSizeInGBs != currentBdsNodeBlockVolumeSize(current) {
		return false
	}
	if spec.ShapeConfig.Ocpus != 0 && spec.ShapeConfig.Ocpus != intValue(current.Ocpus) {
		return false
	}
	if spec.ShapeConfig.MemoryInGBs != 0 && spec.ShapeConfig.MemoryInGBs != intValue(current.MemoryInGBs) {
		return false
	}
	if spec.ShapeConfig.Nvmes != 0 && spec.ShapeConfig.Nvmes != intValue(current.Nvmes) {
		return false
	}
	return true
}

func currentBdsNodeBlockVolumeSize(node bdssdk.Node) int64 {
	var total int64
	for _, attachment := range node.AttachedBlockVolumes {
		total += int64Value(attachment.VolumeSizeInGBs)
	}
	return total
}

func normalizedBdsFreeformTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(tags))
	for key, value := range tags {
		cloned[key] = value
	}
	return cloned
}

func bdsDefinedTagsFromSpec(tags map[string]shared.MapValue) (map[string]map[string]interface{}, error) {
	if len(tags) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	payload, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf("marshal BdsInstance defined tags: %w", err)
	}

	var decoded map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode BdsInstance defined tags: %w", err)
	}

	return normalizedBdsDefinedTags(decoded), nil
}

func normalizedBdsDefinedTags(tags map[string]map[string]interface{}) map[string]map[string]interface{} {
	if len(tags) == 0 {
		return map[string]map[string]interface{}{}
	}

	cloned := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		cloned[namespace] = inner
	}
	return cloned
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
