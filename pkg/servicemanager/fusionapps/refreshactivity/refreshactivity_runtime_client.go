/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

package refreshactivity

import (
	"context"
	"fmt"
	"strings"

	fusionappssdk "github.com/oracle/oci-go-sdk/v65/fusionapps"
	fusionappsv1beta1 "github.com/oracle/oci-service-operator/api/fusionapps/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type refreshActivityOCIClient interface {
	GetWorkRequest(context.Context, fusionappssdk.GetWorkRequestRequest) (fusionappssdk.GetWorkRequestResponse, error)
}

var refreshActivityWorkRequestAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens:   []string{string(fusionappssdk.WorkRequestStatusAccepted), string(fusionappssdk.WorkRequestStatusInProgress), string(fusionappssdk.WorkRequestStatusCanceling)},
	SucceededStatusTokens: []string{string(fusionappssdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(fusionappssdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(fusionappssdk.WorkRequestStatusCanceled)},
}

func init() {
	registerRefreshActivityRuntimeHooksMutator(func(manager *RefreshActivityServiceManager, hooks *RefreshActivityRuntimeHooks) {
		client, err := newRefreshActivityOCIClient(manager)
		applyRefreshActivityRuntimeHooks(hooks, client, err)
	})
}

func newRefreshActivityOCIClient(manager *RefreshActivityServiceManager) (refreshActivityOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("initialize RefreshActivity work-request client: service manager is nil")
	}
	client, err := fusionappssdk.NewFusionApplicationsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize RefreshActivity work-request client: %w", err)
	}
	return client, nil
}

func applyRefreshActivityRuntimeHooks(hooks *RefreshActivityRuntimeHooks, client refreshActivityOCIClient, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = reviewedRefreshActivityRuntimeSemantics()
	hooks.Async.Adapter = refreshActivityWorkRequestAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		if client == nil {
			return nil, fmt.Errorf("RefreshActivity work-request client is not configured")
		}
		response, err := client.GetWorkRequest(ctx, fusionappssdk.GetWorkRequestRequest{WorkRequestId: &workRequestID})
		if err != nil {
			return nil, err
		}
		return response.WorkRequest, nil
	}
	hooks.Async.RecoverResourceID = recoverRefreshActivityID
	hooks.Async.Message = func(phase shared.OSOKAsyncPhase, workRequest any) string {
		current, _ := refreshActivityWorkRequest(workRequest)
		return fmt.Sprintf("RefreshActivity %s work request %s is %s", phase, stringValue(current.Id), current.Status)
	}
}

func reviewedRefreshActivityRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "fusionapps", FormalSlug: "refreshactivity",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "workrequest", Runtime: "generatedruntime", FormalClassification: "workrequest", WorkRequest: &generatedruntime.WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "delete"}}},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"ACCEPTED", "IN_PROGRESS"}, ActiveStates: []string{"SUCCEEDED"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"sourceFusionEnvironmentId", "timeScheduledStart"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"timeScheduledStart"}, ForceNew: []string{"sourceFusionEnvironmentId", "isDataMaskingOpted", "fusionEnvironmentId"}, ZeroValueNullEquivalent: []string{"isDataMaskingOpted"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-confirm-delete"},
	}
}

func recoverRefreshActivityID(_ *fusionappsv1beta1.RefreshActivity, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
	current, err := refreshActivityWorkRequest(workRequest)
	if err != nil {
		return "", err
	}
	wantAction := fusionappssdk.WorkRequestResourceActionTypeCreated
	if phase == shared.OSOKAsyncPhaseDelete {
		wantAction = fusionappssdk.WorkRequestResourceActionTypeDeleted
	}
	for _, resource := range current.Resources {
		if resource.ActionType != wantAction || normalizeResourceType(stringValue(resource.EntityType)) != "refreshactivity" {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("RefreshActivity work request %q does not expose the %s resource identifier", stringValue(current.Id), phase)
}

func refreshActivityWorkRequest(value any) (fusionappssdk.WorkRequest, error) {
	switch current := value.(type) {
	case fusionappssdk.WorkRequest:
		return current, nil
	case *fusionappssdk.WorkRequest:
		if current != nil {
			return *current, nil
		}
	}
	return fusionappssdk.WorkRequest{}, fmt.Errorf("unexpected Fusion Apps work request type %T", value)
}

func normalizeResourceType(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newRefreshActivityServiceClientWithOCIClient(log loggerutil.OSOKLogger, client fusionappssdk.FusionApplicationsClient) RefreshActivityServiceClient {
	manager := &RefreshActivityServiceManager{Log: log}
	hooks := newRefreshActivityDefaultRuntimeHooks(client)
	applyRefreshActivityRuntimeHooks(&hooks, client, nil)
	delegate := defaultRefreshActivityServiceClient{ServiceClient: generatedruntime.NewServiceClient[*fusionappsv1beta1.RefreshActivity](buildRefreshActivityGeneratedRuntimeConfig(manager, hooks))}
	return wrapRefreshActivityGeneratedClient(hooks, delegate)
}
