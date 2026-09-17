/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

package fleetcredential

import (
	"context"
	"fmt"

	fleetappssdk "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	fleetappsv1beta1 "github.com/oracle/oci-service-operator/api/fleetappsmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/fleetappsmanagement/fleetworkrequest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerFleetCredentialRuntimeHooksMutator(func(manager *FleetCredentialServiceManager, hooks *FleetCredentialRuntimeHooks) {
		client, err := newFleetCredentialWorkRequestClient(manager)
		applyFleetCredentialRuntimeHooks(hooks, client, err)
	})
}

func newFleetCredentialWorkRequestClient(manager *FleetCredentialServiceManager) (fleetworkrequest.Client, error) {
	if manager == nil {
		return nil, fmt.Errorf("initialize FleetCredential work-request client: service manager is nil")
	}
	client, err := fleetappssdk.NewFleetAppsManagementWorkRequestClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize FleetCredential work-request client: %w", err)
	}
	return client, nil
}

func applyFleetCredentialRuntimeHooks(hooks *FleetCredentialRuntimeHooks, client fleetworkrequest.Client, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = reviewedFleetCredentialRuntimeSemantics()
	hooks.Async.Adapter = fleetworkrequest.Adapter()
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		return fleetworkrequest.Fetch(ctx, client, workRequestID)
	}
	hooks.Async.RecoverResourceID = func(_ *fleetappsv1beta1.FleetCredential, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
		return fleetworkrequest.RecoverResourceID(workRequest, phase, "fleet credential", "credential")
	}
	hooks.Async.Message = func(phase shared.OSOKAsyncPhase, workRequest any) string {
		return fleetworkrequest.Message("FleetCredential", phase, workRequest)
	}
}

func reviewedFleetCredentialRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "fleetappsmanagement", FormalSlug: "fleetcredential",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "workrequest", Runtime: "generatedruntime", FormalClassification: "workrequest", WorkRequest: &generatedruntime.WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "update", "delete"}}},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "entitySpecifics", "user", "password"}, ForceNew: []string{"fleetId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-confirm-delete"},
	}
}

func newFleetCredentialServiceClientWithOCIClients(log loggerutil.OSOKLogger, client fleetappssdk.FleetAppsManagementClient, workRequestClient fleetappssdk.FleetAppsManagementWorkRequestClient) FleetCredentialServiceClient {
	manager := &FleetCredentialServiceManager{Log: log}
	hooks := newFleetCredentialDefaultRuntimeHooks(FleetCredentialSDKClients{
		fleetAppsManagementClient:            client,
		fleetAppsManagementWorkRequestClient: workRequestClient,
	})
	applyFleetCredentialRuntimeHooks(&hooks, workRequestClient, nil)
	delegate := defaultFleetCredentialServiceClient{ServiceClient: generatedruntime.NewServiceClient[*fleetappsv1beta1.FleetCredential](buildFleetCredentialGeneratedRuntimeConfig(manager, hooks))}
	return wrapFleetCredentialGeneratedClient(hooks, delegate)
}
