/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

// Package nodeworkrequest adapts Blockchain Platform node work requests to
// the shared generatedruntime async contract.
package nodeworkrequest

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/blockchain"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type Client interface {
	GetWorkRequest(context.Context, blockchain.GetWorkRequestRequest) (blockchain.GetWorkRequestResponse, error)
}

func Adapter() servicemanager.WorkRequestAsyncAdapter {
	return servicemanager.WorkRequestAsyncAdapter{
		PendingStatusTokens:   []string{string(blockchain.WorkRequestStatusAccepted), string(blockchain.WorkRequestStatusInProgress), string(blockchain.WorkRequestStatusCanceling)},
		SucceededStatusTokens: []string{string(blockchain.WorkRequestStatusSucceeded)},
		FailedStatusTokens:    []string{string(blockchain.WorkRequestStatusFailed)},
		CanceledStatusTokens:  []string{string(blockchain.WorkRequestStatusCanceled)},
	}
}

func Fetch(ctx context.Context, client Client, workRequestID string) (any, error) {
	if client == nil {
		return nil, fmt.Errorf("Blockchain node work-request client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, blockchain.GetWorkRequestRequest{WorkRequestId: common.String(strings.TrimSpace(workRequestID))})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func RecoverResourceID(workRequest any, phase shared.OSOKAsyncPhase, entityNames ...string) (string, error) {
	current, err := fromAny(workRequest)
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
		identifier := strings.TrimSpace(stringValue(resource.Identifier))
		if identifier == "" {
			continue
		}
		if candidate != "" && candidate != identifier {
			return "", fmt.Errorf("Blockchain node work request %q exposes multiple resource identifiers", stringValue(current.Id))
		}
		candidate = identifier
	}
	if candidate == "" {
		return "", fmt.Errorf("Blockchain node work request %q does not expose the %s resource identifier", stringValue(current.Id), phase)
	}
	return candidate, nil
}

func Message(kind string, phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := fromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", kind, phase, stringValue(current.Id), current.Status)
}

func fromAny(workRequest any) (blockchain.WorkRequest, error) {
	switch current := workRequest.(type) {
	case blockchain.WorkRequest:
		return current, nil
	case *blockchain.WorkRequest:
		if current != nil {
			return *current, nil
		}
	}
	return blockchain.WorkRequest{}, fmt.Errorf("unexpected Blockchain work request type %T", workRequest)
}

func actionForPhase(phase shared.OSOKAsyncPhase) blockchain.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return blockchain.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return blockchain.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return blockchain.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func matchesEntity(resource blockchain.WorkRequestResource, names []string) bool {
	entity := normalize(stringValue(resource.EntityType))
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
