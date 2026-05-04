/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package analyticsinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	analyticssdk "github.com/oracle/oci-go-sdk/v65/analytics"
	"github.com/oracle/oci-go-sdk/v65/common"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerAnalyticsInstanceRuntimeHooksMutator(func(_ *AnalyticsInstanceServiceManager, hooks *AnalyticsInstanceRuntimeHooks) {
		applyAnalyticsInstanceRuntimeHooks(hooks)
	})
}

func applyAnalyticsInstanceRuntimeHooks(hooks *AnalyticsInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedAnalyticsInstanceRuntimeSemantics()
	hooks.ParityHooks.NormalizeDesiredState = normalizeAnalyticsInstanceDesiredState
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *analyticsv1beta1.AnalyticsInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildAnalyticsInstanceUpdateBody(resource, currentResponse)
	}
}

func reviewedAnalyticsInstanceRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newAnalyticsInstanceRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{
		ProvisioningStates: []string{"CREATING"},
		UpdatingStates:     []string{"UPDATING"},
		ActiveStates:       []string{"ACTIVE", "INACTIVE"},
	}
	semantics.Mutation = generatedruntime.MutationSemantics{
		Mutable:       []string{"definedTags", "description", "emailNotification", "freeformTags", "licenseType", "updateChannel"},
		ForceNew:      []string{"capacity.capacityType", "featureSet", "name", "networkEndpointDetails.networkEndpointType"},
		ConflictsWith: map[string][]string{},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func normalizeAnalyticsInstanceDesiredState(
	resource *analyticsv1beta1.AnalyticsInstance,
	currentResponse any,
) {
	if resource == nil || currentResponse == nil {
		return
	}

	// OCI does not echo these create-time inputs back on AnalyticsInstance.
	resource.Spec.AdminUser = ""
	resource.Spec.IdcsAccessToken = ""
}

func analyticsInstanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateAnalyticsInstanceDetails", RequestName: "CreateAnalyticsInstanceDetails", Contribution: "body"},
	}
}

func analyticsInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AnalyticsInstanceId", RequestName: "analyticsInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func analyticsInstanceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "CapacityType", RequestName: "capacityType", Contribution: "query"},
		{FieldName: "FeatureSet", RequestName: "featureSet", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func analyticsInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AnalyticsInstanceId", RequestName: "analyticsInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateAnalyticsInstanceDetails", RequestName: "UpdateAnalyticsInstanceDetails", Contribution: "body"},
	}
}

func analyticsInstanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AnalyticsInstanceId", RequestName: "analyticsInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func buildAnalyticsInstanceUpdateBody(
	resource *analyticsv1beta1.AnalyticsInstance,
	currentResponse any,
) (analyticssdk.UpdateAnalyticsInstanceDetails, bool, error) {
	if resource == nil {
		return analyticssdk.UpdateAnalyticsInstanceDetails{}, false, fmt.Errorf("analyticsinstance resource is nil")
	}

	current, err := analyticsInstanceRuntimeBody(currentResponse)
	if err != nil {
		return analyticssdk.UpdateAnalyticsInstanceDetails{}, false, err
	}

	details := analyticssdk.UpdateAnalyticsInstanceDetails{}
	updateNeeded := false

	if desired, ok := analyticsDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := analyticsDesiredStringUpdate(resource.Spec.EmailNotification, current.EmailNotification); ok {
		details.EmailNotification = desired
		updateNeeded = true
	}
	if desired, ok := analyticsDesiredLicenseTypeUpdate(resource.Spec.LicenseType, current.LicenseType); ok {
		details.LicenseType = desired
		updateNeeded = true
	}
	if desired, ok := analyticsDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := analyticsDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if desired, ok := analyticsDesiredUpdateChannelUpdate(resource.Spec.UpdateChannel, current.UpdateChannel); ok {
		details.UpdateChannel = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func analyticsInstanceRuntimeBody(currentResponse any) (analyticssdk.AnalyticsInstance, error) {
	switch current := currentResponse.(type) {
	case analyticssdk.AnalyticsInstance:
		return current, nil
	case *analyticssdk.AnalyticsInstance:
		if current == nil {
			return analyticssdk.AnalyticsInstance{}, fmt.Errorf("current AnalyticsInstance response is nil")
		}
		return *current, nil
	case analyticssdk.AnalyticsInstanceSummary:
		return analyticssdk.AnalyticsInstance{
			Id:                     current.Id,
			Name:                   current.Name,
			CompartmentId:          current.CompartmentId,
			LifecycleState:         current.LifecycleState,
			FeatureSet:             current.FeatureSet,
			Capacity:               current.Capacity,
			NetworkEndpointDetails: current.NetworkEndpointDetails,
			TimeCreated:            current.TimeCreated,
			Description:            current.Description,
			LicenseType:            current.LicenseType,
			EmailNotification:      current.EmailNotification,
			DefinedTags:            current.DefinedTags,
			FreeformTags:           current.FreeformTags,
			SystemTags:             current.SystemTags,
			ServiceUrl:             current.ServiceUrl,
			TimeUpdated:            current.TimeUpdated,
		}, nil
	case *analyticssdk.AnalyticsInstanceSummary:
		if current == nil {
			return analyticssdk.AnalyticsInstance{}, fmt.Errorf("current AnalyticsInstance response is nil")
		}
		return analyticsInstanceRuntimeBody(*current)
	case analyticssdk.CreateAnalyticsInstanceResponse:
		return current.AnalyticsInstance, nil
	case *analyticssdk.CreateAnalyticsInstanceResponse:
		if current == nil {
			return analyticssdk.AnalyticsInstance{}, fmt.Errorf("current AnalyticsInstance response is nil")
		}
		return current.AnalyticsInstance, nil
	case analyticssdk.GetAnalyticsInstanceResponse:
		return current.AnalyticsInstance, nil
	case *analyticssdk.GetAnalyticsInstanceResponse:
		if current == nil {
			return analyticssdk.AnalyticsInstance{}, fmt.Errorf("current AnalyticsInstance response is nil")
		}
		return current.AnalyticsInstance, nil
	case analyticssdk.UpdateAnalyticsInstanceResponse:
		return current.AnalyticsInstance, nil
	case *analyticssdk.UpdateAnalyticsInstanceResponse:
		if current == nil {
			return analyticssdk.AnalyticsInstance{}, fmt.Errorf("current AnalyticsInstance response is nil")
		}
		return current.AnalyticsInstance, nil
	default:
		return analyticssdk.AnalyticsInstance{}, fmt.Errorf("unexpected current AnalyticsInstance response type %T", currentResponse)
	}
}

func analyticsDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func analyticsDesiredLicenseTypeUpdate(
	spec string,
	current analyticssdk.LicenseTypeEnum,
) (analyticssdk.LicenseTypeEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return analyticssdk.LicenseTypeEnum(spec), true
}

func analyticsDesiredUpdateChannelUpdate(
	spec string,
	current analyticssdk.UpdateChannelEnum,
) (analyticssdk.UpdateChannelEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return analyticssdk.UpdateChannelEnum(spec), true
}

func analyticsDesiredFreeformTagsUpdate(
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

func analyticsDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := analyticsDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if analyticsJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func analyticsDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func analyticsJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
