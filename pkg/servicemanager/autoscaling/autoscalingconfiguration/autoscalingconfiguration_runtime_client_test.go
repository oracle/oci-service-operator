/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package autoscalingconfiguration

import (
	"context"
	"reflect"
	"testing"

	autoscalingsdk "github.com/oracle/oci-go-sdk/v65/autoscaling"
	"github.com/oracle/oci-go-sdk/v65/common"
	autoscalingv1beta1 "github.com/oracle/oci-service-operator/api/autoscaling/v1beta1"
)

func TestBuildAutoScalingConfigurationUpdateBodyKeepsDisabledState(t *testing.T) {
	resource := &autoscalingv1beta1.AutoScalingConfiguration{
		Spec: autoscalingv1beta1.AutoScalingConfigurationSpec{
			DisplayName:       "updated",
			CoolDownInSeconds: 600,
			IsEnabled:         common.Bool(false),
			FreeformTags:      map[string]string{"phase": "update"},
		},
	}
	current := autoscalingsdk.GetAutoScalingConfigurationResponse{
		AutoScalingConfiguration: autoscalingsdk.AutoScalingConfiguration{
			DisplayName:       common.String("create"),
			CoolDownInSeconds: common.Int(300),
			IsEnabled:         common.Bool(false),
			FreeformTags:      map[string]string{"phase": "create"},
		},
	}

	body, update, err := buildAutoScalingConfigurationUpdateBody(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if !update {
		t.Fatal("update = false, want true")
	}
	details, ok := body.(autoscalingsdk.UpdateAutoScalingConfigurationDetails)
	if !ok {
		t.Fatalf("body type = %T", body)
	}
	if details.IsEnabled == nil || *details.IsEnabled {
		t.Fatalf("isEnabled = %#v, want explicit false", details.IsEnabled)
	}
	if stringValue(details.DisplayName) != "updated" || intValue(details.CoolDownInSeconds) != 600 ||
		!reflect.DeepEqual(details.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update details = %+v", details)
	}
}

func TestBuildAutoScalingConfigurationUpdateBodySkipsConvergedState(t *testing.T) {
	resource := &autoscalingv1beta1.AutoScalingConfiguration{
		Spec: autoscalingv1beta1.AutoScalingConfigurationSpec{
			DisplayName:       "converged",
			CoolDownInSeconds: 300,
			IsEnabled:         common.Bool(false),
			FreeformTags:      map[string]string{"phase": "current"},
		},
	}
	current := autoscalingsdk.AutoScalingConfiguration{
		DisplayName:       common.String("converged"),
		CoolDownInSeconds: common.Int(300),
		IsEnabled:         common.Bool(false),
		FreeformTags:      map[string]string{"phase": "current"},
	}

	body, update, err := buildAutoScalingConfigurationUpdateBody(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil || update || body != nil {
		t.Fatalf("body=%#v update=%t err=%v, want no update", body, update, err)
	}
}
