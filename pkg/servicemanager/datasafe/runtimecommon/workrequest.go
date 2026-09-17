/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package runtimecommon

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// WorkRequestClient is the common Data Safe work-request lookup surface.
type WorkRequestClient interface {
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

// ConfigureWorkRequest installs the shared Data Safe status normalization and
// lookup hook. Resources opt in explicitly and retain their own lifecycle and
// mutation semantics.
func ConfigureWorkRequest[T any](hooks *generatedruntime.AsyncHooks[T], client WorkRequestClient, initErr error, kind string) {
	if hooks == nil {
		return
	}
	hooks.Adapter = WorkRequestAdapter()
	hooks.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, fmt.Errorf("initialize %s OCI client: %w", kind, initErr)
		}
		if client == nil {
			return nil, fmt.Errorf("%s work request client is not configured", kind)
		}
		response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{
			WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
		})
		if err != nil {
			return nil, err
		}
		return response.WorkRequest, nil
	}
}

// WorkRequestAdapter returns the Data Safe service-wide status vocabulary.
func WorkRequestAdapter() servicemanager.WorkRequestAsyncAdapter {
	return servicemanager.WorkRequestAsyncAdapter{
		PendingStatusTokens: []string{
			string(datasafesdk.WorkRequestStatusAccepted),
			string(datasafesdk.WorkRequestStatusInProgress),
			string(datasafesdk.WorkRequestStatusCanceling),
			string(datasafesdk.WorkRequestStatusSuspending),
		},
		SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
		FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
		CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
		AttentionStatusTokens: []string{string(datasafesdk.WorkRequestStatusSuspended)},
	}
}
