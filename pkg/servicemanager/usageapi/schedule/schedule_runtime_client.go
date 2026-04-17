/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package schedule

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type scheduleOCIClient interface {
	CreateSchedule(context.Context, usageapisdk.CreateScheduleRequest) (usageapisdk.CreateScheduleResponse, error)
	GetSchedule(context.Context, usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error)
	ListSchedules(context.Context, usageapisdk.ListSchedulesRequest) (usageapisdk.ListSchedulesResponse, error)
	UpdateSchedule(context.Context, usageapisdk.UpdateScheduleRequest) (usageapisdk.UpdateScheduleResponse, error)
	DeleteSchedule(context.Context, usageapisdk.DeleteScheduleRequest) (usageapisdk.DeleteScheduleResponse, error)
}

func init() {
	newScheduleServiceClient = func(manager *ScheduleServiceManager) ScheduleServiceClient {
		sdkClient, err := usageapisdk.NewUsageapiClientWithConfigurationProvider(manager.Provider)
		initErr := error(nil)
		if err != nil {
			initErr = fmt.Errorf("initialize Schedule OCI client: %w", err)
		}
		return defaultScheduleServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.Schedule](
				newScheduleGeneratedRuntimeConfig(manager.Log, sdkClient, initErr),
			),
		}
	}
}

func newScheduleGeneratedRuntimeConfig(
	log loggerutil.OSOKLogger,
	client scheduleOCIClient,
	initErr error,
) generatedruntime.Config[*usageapiv1beta1.Schedule] {
	return generatedruntime.Config[*usageapiv1beta1.Schedule]{
		Kind:      "Schedule",
		SDKName:   "Schedule",
		Log:       log,
		InitError: initErr,
		Semantics: newScheduleRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &usageapisdk.CreateScheduleRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				resp, err := client.CreateSchedule(ctx, *request.(*usageapisdk.CreateScheduleRequest))
				return normalizeCreateScheduleResponse(resp), err
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CreateScheduleDetails", RequestName: "CreateScheduleDetails", Contribution: "body", PreferResourceID: false}},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &usageapisdk.GetScheduleRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				resp, err := client.GetSchedule(ctx, *request.(*usageapisdk.GetScheduleRequest))
				return normalizeGetScheduleResponse(resp), err
			},
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "path", PreferResourceID: true}},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &usageapisdk.ListSchedulesRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListSchedules(ctx, *request.(*usageapisdk.ListSchedulesRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false}, {FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false}, {FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false}, {FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false}, {FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false}, {FieldName: "Name", RequestName: "name", Contribution: "query", PreferResourceID: false}},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &usageapisdk.UpdateScheduleRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				resp, err := client.UpdateSchedule(ctx, *request.(*usageapisdk.UpdateScheduleRequest))
				return normalizeUpdateScheduleResponse(resp), err
			},
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateScheduleDetails", RequestName: "UpdateScheduleDetails", Contribution: "body", PreferResourceID: false}},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &usageapisdk.DeleteScheduleRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteSchedule(ctx, *request.(*usageapisdk.DeleteScheduleRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "path", PreferResourceID: true}},
		},
	}
}

func normalizeCreateScheduleResponse(resp usageapisdk.CreateScheduleResponse) usageapisdk.CreateScheduleResponse {
	normalizeScheduleQueryProperties(&resp.Schedule)
	return resp
}

func normalizeGetScheduleResponse(resp usageapisdk.GetScheduleResponse) usageapisdk.GetScheduleResponse {
	normalizeScheduleQueryProperties(&resp.Schedule)
	return resp
}

func normalizeUpdateScheduleResponse(resp usageapisdk.UpdateScheduleResponse) usageapisdk.UpdateScheduleResponse {
	normalizeScheduleQueryProperties(&resp.Schedule)
	return resp
}

func normalizeScheduleQueryProperties(schedule *usageapisdk.Schedule) {
	if schedule == nil || schedule.QueryProperties != nil {
		return
	}
	schedule.QueryProperties = &usageapisdk.QueryProperties{
		IsAggregateByTime: common.Bool(false),
	}
}
