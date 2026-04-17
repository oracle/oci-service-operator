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
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type bdsInstanceOCIClient interface {
	CreateBdsInstance(context.Context, bdssdk.CreateBdsInstanceRequest) (bdssdk.CreateBdsInstanceResponse, error)
	GetBdsInstance(context.Context, bdssdk.GetBdsInstanceRequest) (bdssdk.GetBdsInstanceResponse, error)
	ListBdsInstances(context.Context, bdssdk.ListBdsInstancesRequest) (bdssdk.ListBdsInstancesResponse, error)
	UpdateBdsInstance(context.Context, bdssdk.UpdateBdsInstanceRequest) (bdssdk.UpdateBdsInstanceResponse, error)
	DeleteBdsInstance(context.Context, bdssdk.DeleteBdsInstanceRequest) (bdssdk.DeleteBdsInstanceResponse, error)
}

func init() {
	newBdsInstanceServiceClient = func(manager *BdsInstanceServiceManager) BdsInstanceServiceClient {
		sdkClient, err := bdssdk.NewBdsClientWithConfigurationProvider(manager.Provider)
		config := newBdsInstanceRuntimeConfig(manager.Log, sdkClient)
		if err != nil {
			config.InitError = fmt.Errorf("initialize BdsInstance OCI client: %w", err)
		}
		return defaultBdsInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*bdsv1beta1.BdsInstance](config),
		}
	}
}

func newBdsInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	sdkClient bdsInstanceOCIClient,
) generatedruntime.Config[*bdsv1beta1.BdsInstance] {
	return generatedruntime.Config[*bdsv1beta1.BdsInstance]{
		Kind:      "BdsInstance",
		SDKName:   "BdsInstance",
		Log:       log,
		Semantics: reviewedBdsInstanceRuntimeSemantics(),
		BuildUpdateBody: func(
			_ context.Context,
			resource *bdsv1beta1.BdsInstance,
			_ string,
			currentResponse any,
		) (any, bool, error) {
			return buildBdsInstanceUpdateBody(resource, currentResponse)
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &bdssdk.CreateBdsInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.CreateBdsInstance(ctx, *request.(*bdssdk.CreateBdsInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateBdsInstanceDetails", RequestName: "CreateBdsInstanceDetails", Contribution: "body"},
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &bdssdk.GetBdsInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.GetBdsInstance(ctx, *request.(*bdssdk.GetBdsInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "BdsInstanceId", RequestName: "bdsInstanceId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &bdssdk.ListBdsInstancesRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.ListBdsInstances(ctx, *request.(*bdssdk.ListBdsInstancesRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &bdssdk.UpdateBdsInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.UpdateBdsInstance(ctx, *request.(*bdssdk.UpdateBdsInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "BdsInstanceId", RequestName: "bdsInstanceId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateBdsInstanceDetails", RequestName: "UpdateBdsInstanceDetails", Contribution: "body"},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &bdssdk.DeleteBdsInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.DeleteBdsInstance(ctx, *request.(*bdssdk.DeleteBdsInstanceRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "BdsInstanceId", RequestName: "bdsInstanceId", Contribution: "path", PreferResourceID: true},
			},
		},
	}
}

func reviewedBdsInstanceRuntimeSemantics() *generatedruntime.Semantics {
	createHooks := []generatedruntime.Hook{
		{Helper: "tfresource.CreateResource"},
	}
	updateHooks := []generatedruntime.Hook{
		{Helper: "tfresource.UpdateResource"},
	}
	deleteHooks := []generatedruntime.Hook{
		{Helper: "tfresource.DeleteResource"},
	}

	return &generatedruntime.Semantics{
		FormalService: "bds",
		FormalSlug:    "bdsinstance",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING", "SUSPENDING", "RESUMING"},
			ActiveStates:       []string{"ACTIVE", "INACTIVE", "SUSPENDED"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
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
				"networkConfig",
				"bootstrapScriptUrl",
				"freeformTags",
				"definedTags",
				"kmsKeyId",
				"clusterProfile",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: createHooks,
			Update: updateHooks,
			Delete: deleteHooks,
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    createHooks,
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    updateHooks,
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    deleteHooks,
		},
	}
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
