/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package detectorrecipedetectorrule

import (
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
)

func TestDetectorRecipeDetectorRuleProjectsKeyedIdentity(t *testing.T) {
	resource := &cloudguardv1beta1.DetectorRecipeDetectorRule{}
	response := cloudguardsdk.GetDetectorRecipeDetectorRuleResponse{DetectorRecipeDetectorRule: cloudguardsdk.DetectorRecipeDetectorRule{
		DetectorRuleId: common.String("detector-rule-id"), LifecycleState: cloudguardsdk.LifecycleStateActive,
	}}
	if err := projectDetectorRecipeDetectorRuleStatus(resource, response); err != nil {
		t.Fatal(err)
	}
	if resource.Status.Id != "detector-rule-id" || string(resource.Status.OsokStatus.Ocid) != "detector-rule-id" {
		t.Fatalf("projected status = %+v", resource.Status)
	}
}
