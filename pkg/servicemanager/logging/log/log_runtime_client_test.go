/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package log

import (
	"reflect"
	"testing"

	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func TestLogRuntimeSemanticsEncodesBaselineLifecycle(t *testing.T) {
	t.Parallel()

	got := newLogRuntimeSemantics()

	if got == nil {
		t.Fatal("newLogRuntimeSemantics() = nil")
	}
	if got.FormalService != "logging" {
		t.Fatalf("FormalService = %q, want logging", got.FormalService)
	}
	if got.FormalSlug != "log" {
		t.Fatalf("FormalSlug = %q, want log", got.FormalSlug)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}

	assertStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{
		"displayName",
		"lifecycleState",
		"logGroupId",
		"logType",
		"sourceResource",
		"sourceService",
	})
	assertStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"definedTags",
		"displayName",
		"freeformTags",
		"isEnabled",
		"retentionDuration",
	})
	assertStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"configuration",
		"logGroupId",
		"logType",
	})
}

func TestLogRequestFieldsKeepTrackedOperationsScopedToRecordedLogGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  logCreateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LogGroupId",
					RequestName:  "logGroupId",
					Contribution: "path",
					LookupPaths:  []string{"spec.logGroupId", "logGroupId"},
				},
				{
					FieldName:    "CreateLogDetails",
					RequestName:  "CreateLogDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  logGetFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LogGroupId",
					RequestName:  "logGroupId",
					Contribution: "path",
					LookupPaths:  []string{"status.logGroupId", "logGroupId"},
				},
				{
					FieldName:        "LogId",
					RequestName:      "logId",
					Contribution:     "path",
					PreferResourceID: true,
				},
			},
		},
		{
			name: "list",
			got:  logListFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LogGroupId",
					RequestName:  "logGroupId",
					Contribution: "path",
					LookupPaths:  []string{"status.logGroupId", "logGroupId"},
				},
				{
					FieldName:    "LogType",
					RequestName:  "logType",
					Contribution: "query",
				},
				{
					FieldName:    "SourceService",
					RequestName:  "sourceService",
					Contribution: "query",
					LookupPaths:  []string{"spec.configuration.source.service", "configuration.source.service"},
				},
				{
					FieldName:    "SourceResource",
					RequestName:  "sourceResource",
					Contribution: "query",
					LookupPaths:  []string{"spec.configuration.source.resource", "configuration.source.resource"},
				},
				{
					FieldName:    "DisplayName",
					RequestName:  "displayName",
					Contribution: "query",
				},
				{
					FieldName:    "LifecycleState",
					RequestName:  "lifecycleState",
					Contribution: "query",
				},
				{
					FieldName:    "Page",
					RequestName:  "page",
					Contribution: "query",
				},
				{
					FieldName:    "Limit",
					RequestName:  "limit",
					Contribution: "query",
				},
				{
					FieldName:    "SortBy",
					RequestName:  "sortBy",
					Contribution: "query",
				},
				{
					FieldName:    "SortOrder",
					RequestName:  "sortOrder",
					Contribution: "query",
				},
			},
		},
		{
			name: "update",
			got:  logUpdateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LogGroupId",
					RequestName:  "logGroupId",
					Contribution: "path",
					LookupPaths:  []string{"status.logGroupId", "logGroupId"},
				},
				{
					FieldName:        "LogId",
					RequestName:      "logId",
					Contribution:     "path",
					PreferResourceID: true,
				},
				{
					FieldName:    "UpdateLogDetails",
					RequestName:  "UpdateLogDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  logDeleteFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LogGroupId",
					RequestName:  "logGroupId",
					Contribution: "path",
					LookupPaths:  []string{"status.logGroupId", "logGroupId"},
				},
				{
					FieldName:        "LogId",
					RequestName:      "logId",
					Contribution:     "path",
					PreferResourceID: true,
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertRequestFieldsEqual(t, tc.name, tc.got, tc.want)
		})
	}
}

func assertRequestFieldsEqual(t *testing.T, name string, got, want []generatedruntime.RequestField) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s fields = %#v, want %#v", name, got, want)
}

func assertStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
