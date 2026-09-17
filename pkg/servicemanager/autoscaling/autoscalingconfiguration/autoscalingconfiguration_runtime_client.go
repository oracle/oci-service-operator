/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package autoscalingconfiguration

import (
	"context"
	"fmt"
	"reflect"

	autoscalingsdk "github.com/oracle/oci-go-sdk/v65/autoscaling"
	"github.com/oracle/oci-go-sdk/v65/common"
	autoscalingv1beta1 "github.com/oracle/oci-service-operator/api/autoscaling/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func init() {
	registerAutoScalingConfigurationRuntimeHooksMutator(func(
		_ *AutoScalingConfigurationServiceManager,
		hooks *AutoScalingConfigurationRuntimeHooks,
	) {
		hooks.BuildUpdateBody = buildAutoScalingConfigurationUpdateBody
	})
}

func buildAutoScalingConfigurationUpdateBody(
	_ context.Context,
	resource *autoscalingv1beta1.AutoScalingConfiguration,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("autoscaling configuration resource is nil")
	}
	current, err := autoScalingConfigurationFromResponse(currentResponse)
	if err != nil {
		return nil, false, err
	}

	details := autoscalingsdk.UpdateAutoScalingConfigurationDetails{}
	updateNeeded := false
	desiredEnabled := false
	if current.IsEnabled != nil {
		desiredEnabled = *current.IsEnabled
	}
	if resource.Spec.IsEnabled != nil {
		desiredEnabled = *resource.Spec.IsEnabled
		if current.IsEnabled == nil || *current.IsEnabled != desiredEnabled {
			updateNeeded = true
		}
	}
	if resource.Spec.DisplayName != "" && stringValue(current.DisplayName) != resource.Spec.DisplayName {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.CoolDownInSeconds != 0 && intValue(current.CoolDownInSeconds) != resource.Spec.CoolDownInSeconds {
		details.CoolDownInSeconds = common.Int(resource.Spec.CoolDownInSeconds)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		details.FreeformTags = resource.Spec.FreeformTags
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desired := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	if !updateNeeded {
		return nil, false, nil
	}

	// The live Autoscaling API rejects otherwise valid partial updates when
	// isEnabled is omitted. Preserve the desired value on every actual update.
	details.IsEnabled = common.Bool(desiredEnabled)
	return details, true, nil
}

func autoScalingConfigurationFromResponse(response any) (autoscalingsdk.AutoScalingConfiguration, error) {
	switch current := response.(type) {
	case autoscalingsdk.AutoScalingConfiguration:
		return current, nil
	case *autoscalingsdk.AutoScalingConfiguration:
		if current != nil {
			return *current, nil
		}
	case autoscalingsdk.CreateAutoScalingConfigurationResponse:
		return current.AutoScalingConfiguration, nil
	case *autoscalingsdk.CreateAutoScalingConfigurationResponse:
		if current != nil {
			return current.AutoScalingConfiguration, nil
		}
	case autoscalingsdk.GetAutoScalingConfigurationResponse:
		return current.AutoScalingConfiguration, nil
	case *autoscalingsdk.GetAutoScalingConfigurationResponse:
		if current != nil {
			return current.AutoScalingConfiguration, nil
		}
	case autoscalingsdk.UpdateAutoScalingConfigurationResponse:
		return current.AutoScalingConfiguration, nil
	case *autoscalingsdk.UpdateAutoScalingConfigurationResponse:
		if current != nil {
			return current.AutoScalingConfiguration, nil
		}
	}
	return autoscalingsdk.AutoScalingConfiguration{}, fmt.Errorf(
		"unexpected autoscaling configuration response type %T",
		response,
	)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
