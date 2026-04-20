/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package schedule

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
)

func init() {
	registerScheduleRuntimeHooksMutator(func(_ *ScheduleServiceManager, hooks *ScheduleRuntimeHooks) {
		applyScheduleRuntimeHooks(hooks)
	})
}

func applyScheduleRuntimeHooks(hooks *ScheduleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Create.Call = wrapScheduleOperationCall(hooks.Create.Call, normalizeCreateScheduleResponse)
	hooks.Get.Call = wrapScheduleOperationCall(hooks.Get.Call, normalizeGetScheduleResponse)
	hooks.Update.Call = wrapScheduleOperationCall(hooks.Update.Call, normalizeUpdateScheduleResponse)
}

func wrapScheduleOperationCall[Req any, Resp any](
	call func(context.Context, Req) (Resp, error),
	normalize func(Resp) Resp,
) func(context.Context, Req) (Resp, error) {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request Req) (Resp, error) {
		resp, err := call(ctx, request)
		return normalize(resp), err
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
