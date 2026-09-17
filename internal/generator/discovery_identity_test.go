/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/ocisdk"
)

func TestShouldPreferResourceIDUsesFinalNestedKey(t *testing.T) {
	tests := []struct {
		name         string
		field        ocisdk.Field
		pathIdentity string
		want         bool
	}{
		{name: "parent key", field: ocisdk.Field{Name: "AttributeKey", RequestName: "attributeKey", Contribution: ocisdk.FieldContributionPath}, want: false},
		{name: "resource key", field: ocisdk.Field{Name: "TagKey", RequestName: "tagKey", Contribution: ocisdk.FieldContributionPath}, pathIdentity: "tagKey", want: true},
		{name: "abbreviated resource id", field: ocisdk.Field{Name: "DetectorRuleId", RequestName: "detectorRuleId", Contribution: ocisdk.FieldContributionPath}, pathIdentity: "detectorRuleId", want: true},
		{name: "struct-order parent id", field: ocisdk.Field{Name: "MonitoringTemplateId", RequestName: "monitoringTemplateId", Contribution: ocisdk.FieldContributionPath}, pathIdentity: "alarmConditionId", want: false},
		{name: "list parent id", field: ocisdk.Field{Name: "TargetId", RequestName: "targetId", Contribution: ocisdk.FieldContributionPath}, pathIdentity: "targetId", want: false},
		{name: "query key", field: ocisdk.Field{Name: "TagKey", RequestName: "tagKey", Contribution: ocisdk.FieldContributionQuery}, pathIdentity: "tagKey", want: false},
		{name: "create key", field: ocisdk.Field{Name: "TagKey", RequestName: "tagKey", Contribution: ocisdk.FieldContributionPath}, pathIdentity: "tagKey", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := "Get"
			if test.name == "create key" {
				operation = "Create"
			} else if test.name == "list parent id" {
				operation = "List"
			}
			if got := shouldPreferResourceID(operation, "AttributeTag", test.field, 5, test.pathIdentity); got != test.want {
				t.Fatalf("shouldPreferResourceID() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestTerminalPathParameter(t *testing.T) {
	if got := terminalPathParameter("/monitoringTemplates/{monitoringTemplateId}/alarmConditions/{alarmConditionId}"); got != "alarmConditionId" {
		t.Fatalf("terminalPathParameter() = %q", got)
	}
}
