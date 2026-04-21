/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package log

import (
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerLogRuntimeHooksMutator(func(_ *LogServiceManager, hooks *LogRuntimeHooks) {
		applyLogRuntimeHooks(hooks)
	})
}

func applyLogRuntimeHooks(hooks *LogRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Create.Fields = logCreateFields()
	hooks.Get.Fields = logGetFields()
	hooks.List.Fields = logListFields()
	hooks.Update.Fields = logUpdateFields()
	hooks.Delete.Fields = logDeleteFields()
}

func logCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
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
	}
}

func logGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
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
	}
}

func logListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
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
	}
}

func logUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
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
	}
}

func logDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
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
	}
}
