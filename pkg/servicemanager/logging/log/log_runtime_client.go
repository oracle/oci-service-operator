/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package log

import (
	"context"
	"fmt"

	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	newLogServiceClient = func(manager *LogServiceManager) LogServiceClient {
		sdkClient, err := loggingsdk.NewLoggingManagementClientWithConfigurationProvider(manager.Provider)
		config := generatedruntime.Config[*loggingv1beta1.Log]{
			Kind:      "Log",
			SDKName:   "Log",
			Log:       manager.Log,
			Semantics: newLogRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &loggingsdk.CreateLogRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.CreateLog(ctx, *request.(*loggingsdk.CreateLogRequest))
				},
				Fields: logCreateFields(),
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &loggingsdk.GetLogRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.GetLog(ctx, *request.(*loggingsdk.GetLogRequest))
				},
				Fields: logGetFields(),
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &loggingsdk.ListLogsRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.ListLogs(ctx, *request.(*loggingsdk.ListLogsRequest))
				},
				Fields: logListFields(),
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &loggingsdk.UpdateLogRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.UpdateLog(ctx, *request.(*loggingsdk.UpdateLogRequest))
				},
				Fields: logUpdateFields(),
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &loggingsdk.DeleteLogRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.DeleteLog(ctx, *request.(*loggingsdk.DeleteLogRequest))
				},
				Fields: logDeleteFields(),
			},
		}
		if err != nil {
			config.InitError = fmt.Errorf("initialize Log OCI client: %w", err)
		}
		return defaultLogServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.Log](config),
		}
	}
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
