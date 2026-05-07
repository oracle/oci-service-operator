/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rovercluster

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	roversdk "github.com/oracle/oci-go-sdk/v65/rover"
	roverv1beta1 "github.com/oracle/oci-service-operator/api/rover/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerRoverClusterRuntimeHooksMutator(func(_ *RoverClusterServiceManager, hooks *RoverClusterRuntimeHooks) {
		applyRoverClusterRuntimeHooks(hooks)
	})
}

func applyRoverClusterRuntimeHooks(hooks *RoverClusterRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedRoverClusterRuntimeSemantics()
	hooks.List.Fields = reviewedRoverClusterListFields()
	hooks.ParityHooks.NormalizeDesiredState = normalizeRoverClusterDesiredState
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *roverv1beta1.RoverCluster,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildRoverClusterUpdateBody(resource, currentResponse)
	}
}

func reviewedRoverClusterRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newRoverClusterRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"clusterSize", "clusterType", "compartmentId", "displayName"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func reviewedRoverClusterListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "ClusterType", RequestName: "clusterType", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func normalizeRoverClusterDesiredState(resource *roverv1beta1.RoverCluster, _ any) {
	if resource == nil {
		return
	}

	resource.Spec.LifecycleState = ""
	resource.Spec.LifecycleStateDetails = ""
	resource.Spec.SystemTags = nil
	for i := range resource.Spec.ClusterWorkloads {
		resource.Spec.ClusterWorkloads[i].WorkRequestId = ""
	}
}

func buildRoverClusterUpdateBody(
	resource *roverv1beta1.RoverCluster,
	currentResponse any,
) (roversdk.UpdateRoverClusterDetails, bool, error) {
	if resource == nil {
		return roversdk.UpdateRoverClusterDetails{}, false, fmt.Errorf("rovercluster resource is nil")
	}

	current, err := roverClusterRuntimeBody(currentResponse)
	if err != nil {
		return roversdk.UpdateRoverClusterDetails{}, false, err
	}

	details := roversdk.UpdateRoverClusterDetails{}
	updateNeeded := false

	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredIntUpdate(resource.Spec.ClusterSize, current.ClusterSize); ok {
		details.ClusterSize = desired
		updateNeeded = true
	}
	if desired, ok, err := roverClusterDesiredShippingAddressUpdate(resource.Spec.CustomerShippingAddress, current.CustomerShippingAddress); err != nil {
		return roversdk.UpdateRoverClusterDetails{}, false, err
	} else if ok {
		details.CustomerShippingAddress = desired
		updateNeeded = true
	}
	if desired, ok, err := roverClusterDesiredWorkloadsUpdate(resource.Spec.ClusterWorkloads, current.ClusterWorkloads); err != nil {
		return roversdk.UpdateRoverClusterDetails{}, false, err
	} else if ok {
		details.ClusterWorkloads = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.SuperUserPassword, current.SuperUserPassword); ok {
		details.SuperUserPassword = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredEnclosureTypeUpdate(resource.Spec.EnclosureType, current.EnclosureType); ok {
		details.EnclosureType = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.UnlockPassphrase, current.UnlockPassphrase); ok {
		details.UnlockPassphrase = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.PointOfContact, current.PointOfContact); ok {
		details.PointOfContact = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.PointOfContactPhoneNumber, current.PointOfContactPhoneNumber); ok {
		details.PointOfContactPhoneNumber = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredShippingPreferenceUpdate(resource.Spec.ShippingPreference, current.ShippingPreference); ok {
		details.ShippingPreference = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.OracleShippingTrackingUrl, current.OracleShippingTrackingUrl); ok {
		details.OracleShippingTrackingUrl = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.SubscriptionId, current.SubscriptionId); ok {
		details.SubscriptionId = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.ShippingVendor, current.ShippingVendor); ok {
		details.ShippingVendor = desired
		updateNeeded = true
	}
	if desired, ok, err := roverClusterDesiredTimeUpdate(resource.Spec.TimePickupExpected, current.TimePickupExpected); err != nil {
		return roversdk.UpdateRoverClusterDetails{}, false, err
	} else if ok {
		details.TimePickupExpected = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredBoolUpdate(resource.Spec.IsImportRequested, current.IsImportRequested); ok {
		details.IsImportRequested = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.ImportCompartmentId, current.ImportCompartmentId); ok {
		details.ImportCompartmentId = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.ImportFileBucket, current.ImportFileBucket); ok {
		details.ImportFileBucket = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredStringUpdate(resource.Spec.DataValidationCode, current.DataValidationCode); ok {
		details.DataValidationCode = desired
		updateNeeded = true
	}
	if desired, ok := roverClusterDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok, err := roverClusterDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); err != nil {
		return roversdk.UpdateRoverClusterDetails{}, false, err
	} else if ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func roverClusterRuntimeBody(currentResponse any) (roversdk.RoverCluster, error) {
	switch current := currentResponse.(type) {
	case roversdk.RoverCluster:
		return current, nil
	case *roversdk.RoverCluster:
		if current == nil {
			return roversdk.RoverCluster{}, fmt.Errorf("current RoverCluster response is nil")
		}
		return *current, nil
	case roversdk.RoverClusterSummary:
		return roverClusterFromSummary(current), nil
	case *roversdk.RoverClusterSummary:
		if current == nil {
			return roversdk.RoverCluster{}, fmt.Errorf("current RoverCluster response is nil")
		}
		return roverClusterFromSummary(*current), nil
	case roversdk.CreateRoverClusterResponse:
		return current.RoverCluster, nil
	case *roversdk.CreateRoverClusterResponse:
		if current == nil {
			return roversdk.RoverCluster{}, fmt.Errorf("current RoverCluster response is nil")
		}
		return current.RoverCluster, nil
	case roversdk.GetRoverClusterResponse:
		return current.RoverCluster, nil
	case *roversdk.GetRoverClusterResponse:
		if current == nil {
			return roversdk.RoverCluster{}, fmt.Errorf("current RoverCluster response is nil")
		}
		return current.RoverCluster, nil
	case roversdk.UpdateRoverClusterResponse:
		return current.RoverCluster, nil
	case *roversdk.UpdateRoverClusterResponse:
		if current == nil {
			return roversdk.RoverCluster{}, fmt.Errorf("current RoverCluster response is nil")
		}
		return current.RoverCluster, nil
	default:
		return roversdk.RoverCluster{}, fmt.Errorf("unexpected current RoverCluster response type %T", currentResponse)
	}
}

func roverClusterFromSummary(summary roversdk.RoverClusterSummary) roversdk.RoverCluster {
	return roversdk.RoverCluster{
		Id:                    summary.Id,
		CompartmentId:         summary.CompartmentId,
		DisplayName:           summary.DisplayName,
		ClusterSize:           summary.ClusterSize,
		LifecycleState:        summary.LifecycleState,
		TimeCreated:           summary.TimeCreated,
		LifecycleStateDetails: summary.LifecycleStateDetails,
		ClusterType:           summary.ClusterType,
		FreeformTags:          summary.FreeformTags,
		DefinedTags:           summary.DefinedTags,
		SystemTags:            summary.SystemTags,
	}
}

func roverClusterDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func roverClusterDesiredIntUpdate(spec int, current *int) (*int, bool) {
	if spec == 0 {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Int(spec), true
}

func roverClusterDesiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	currentValue := false
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	return common.Bool(spec), true
}

func roverClusterDesiredEnclosureTypeUpdate(
	spec string,
	current roversdk.EnclosureTypeEnum,
) (roversdk.EnclosureTypeEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return roversdk.EnclosureTypeEnum(spec), true
}

func roverClusterDesiredShippingPreferenceUpdate(
	spec string,
	current roversdk.RoverClusterShippingPreferenceEnum,
) (roversdk.UpdateRoverClusterDetailsShippingPreferenceEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return roversdk.UpdateRoverClusterDetailsShippingPreferenceEnum(spec), true
}

func roverClusterDesiredTimeUpdate(spec string, current *common.SDKTime) (*common.SDKTime, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, false, nil
	}

	parsed, err := time.Parse(time.RFC3339, spec)
	if err != nil {
		return nil, false, fmt.Errorf("parse timePickupExpected: %w", err)
	}
	parsed = parsed.UTC()
	if current != nil && current.Time.Equal(parsed) {
		return nil, false, nil
	}

	return &common.SDKTime{Time: parsed}, true, nil
}

func roverClusterDesiredFreeformTagsUpdate(
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

func roverClusterDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := roverClusterDefinedTagsFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	if roverClusterJSONEqual(desired, current) {
		return nil, false, nil
	}
	return desired, true, nil
}

func roverClusterDesiredShippingAddressUpdate(
	spec roverv1beta1.RoverClusterCustomerShippingAddress,
	current *roversdk.ShippingAddress,
) (*roversdk.ShippingAddress, bool, error) {
	if spec == (roverv1beta1.RoverClusterCustomerShippingAddress{}) {
		return nil, false, nil
	}

	desired, err := roverClusterShippingAddressFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	if current != nil && roverClusterJSONEqual(desired, current) {
		return nil, false, nil
	}
	return desired, true, nil
}

func roverClusterDesiredWorkloadsUpdate(
	spec []roverv1beta1.RoverClusterClusterWorkload,
	current []roversdk.RoverWorkload,
) ([]roversdk.RoverWorkload, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := roverClusterWorkloadsFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	if roverClusterJSONEqual(desired, current) {
		return nil, false, nil
	}
	return desired, true, nil
}

func roverClusterShippingAddressFromSpec(
	spec roverv1beta1.RoverClusterCustomerShippingAddress,
) (*roversdk.ShippingAddress, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal RoverCluster customerShippingAddress: %w", err)
	}

	var address roversdk.ShippingAddress
	if err := json.Unmarshal(payload, &address); err != nil {
		return nil, fmt.Errorf("decode RoverCluster customerShippingAddress: %w", err)
	}
	return &address, nil
}

func roverClusterWorkloadsFromSpec(
	spec []roverv1beta1.RoverClusterClusterWorkload,
) ([]roversdk.RoverWorkload, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal RoverCluster clusterWorkloads: %w", err)
	}

	var workloads []roversdk.RoverWorkload
	if err := json.Unmarshal(payload, &workloads); err != nil {
		return nil, fmt.Errorf("decode RoverCluster clusterWorkloads: %w", err)
	}
	return workloads, nil
}

func roverClusterDefinedTagsFromSpec(
	spec map[string]shared.MapValue,
) (map[string]map[string]interface{}, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal RoverCluster definedTags: %w", err)
	}

	var tags map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &tags); err != nil {
		return nil, fmt.Errorf("decode RoverCluster definedTags: %w", err)
	}
	return tags, nil
}

func roverClusterJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return string(leftPayload) == string(rightPayload)
}
