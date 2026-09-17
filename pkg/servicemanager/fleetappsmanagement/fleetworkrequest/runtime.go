/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

// Package fleetworkrequest adapts Fleet Application Management work requests
// to the shared generatedruntime async contract.
package fleetworkrequest

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	fleetappssdk "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type Client interface {
	GetWorkRequest(context.Context, fleetappssdk.GetWorkRequestRequest) (fleetappssdk.GetWorkRequestResponse, error)
}

func Adapter() servicemanager.WorkRequestAsyncAdapter {
	return servicemanager.WorkRequestAsyncAdapter{
		PendingStatusTokens:   []string{string(fleetappssdk.OperationStatusAccepted), string(fleetappssdk.OperationStatusInProgress), string(fleetappssdk.OperationStatusWaiting), string(fleetappssdk.OperationStatusCanceling)},
		SucceededStatusTokens: []string{string(fleetappssdk.OperationStatusSucceeded)},
		FailedStatusTokens:    []string{string(fleetappssdk.OperationStatusFailed)},
		CanceledStatusTokens:  []string{string(fleetappssdk.OperationStatusCanceled)},
		AttentionStatusTokens: []string{string(fleetappssdk.OperationStatusNeedsAttention)},
	}
}

func Fetch(ctx context.Context, client Client, workRequestID string) (any, error) {
	if client == nil {
		return nil, fmt.Errorf("Fleet Application Management work-request client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, fleetappssdk.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func RecoverResourceID(workRequest any, phase shared.OSOKAsyncPhase, entityNames ...string) (string, error) {
	current, err := FromAny(workRequest)
	if err != nil {
		return "", err
	}
	wantAction := actionForPhase(phase)
	var candidate string
	for _, resource := range current.Resources {
		if wantAction != "" && resource.ActionType != wantAction {
			continue
		}
		if len(entityNames) > 0 && !matchesEntity(resource, entityNames) {
			continue
		}
		identifier := strings.TrimSpace(StringValue(resource.Identifier))
		if identifier == "" {
			continue
		}
		if candidate != "" && candidate != identifier {
			return "", fmt.Errorf("Fleet Application Management work request %q exposes multiple resource identifiers", StringValue(current.Id))
		}
		candidate = identifier
	}
	if candidate == "" {
		return "", fmt.Errorf("Fleet Application Management work request %q does not expose the %s resource identifier", StringValue(current.Id), phase)
	}
	return candidate, nil
}

func Message(kind string, phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := FromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", kind, phase, StringValue(current.Id), current.Status)
}

func FromAny(workRequest any) (fleetappssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case fleetappssdk.WorkRequest:
		return current, nil
	case *fleetappssdk.WorkRequest:
		if current != nil {
			return *current, nil
		}
	}
	return fleetappssdk.WorkRequest{}, fmt.Errorf("unexpected Fleet Application Management work request type %T", workRequest)
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func actionForPhase(phase shared.OSOKAsyncPhase) fleetappssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return fleetappssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return fleetappssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return fleetappssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func matchesEntity(resource fleetappssdk.WorkRequestResource, names []string) bool {
	entity := normalize(StringValue(resource.EntityType))
	for _, name := range names {
		if entity == normalize(name) {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.TrimSpace(value))
}
