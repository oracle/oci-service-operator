/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

package fleetresource

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
	registerFleetResourceRuntimeHooksMutator(func(manager *FleetResourceServiceManager, hooks *FleetResourceRuntimeHooks) {
		client, err := newFleetResourceWorkRequestClient(manager)
		applyFleetResourceRuntimeHooks(hooks, client, err)
	})
}

func newFleetResourceWorkRequestClient(manager *FleetResourceServiceManager) (fleetworkrequest.Client, error) {
	if manager == nil {
		return nil, fmt.Errorf("initialize FleetResource work-request client: service manager is nil")
	}
	client, err := fleetappssdk.NewFleetAppsManagementWorkRequestClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize FleetResource work-request client: %w", err)
	}
	return client, nil
}

func applyFleetResourceRuntimeHooks(hooks *FleetResourceRuntimeHooks, client fleetworkrequest.Client, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = reviewedFleetResourceRuntimeSemantics()
	hooks.Async.Adapter = fleetworkrequest.Adapter()
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		return fleetworkrequest.Fetch(ctx, client, workRequestID)
	}
	hooks.Async.RecoverResourceID = func(_ *fleetappsv1beta1.FleetResource, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
		return fleetworkrequest.RecoverResourceID(workRequest, phase, "fleet resource")
	}
	hooks.Async.Message = func(phase shared.OSOKAsyncPhase, workRequest any) string {
		return fleetworkrequest.Message("FleetResource", phase, workRequest)
	}
}

func reviewedFleetResourceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "fleetappsmanagement", FormalSlug: "fleetresource",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "workrequest", Runtime: "generatedruntime", FormalClassification: "workrequest", WorkRequest: &generatedruntime.WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "update", "delete"}}},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"resourceId"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"tenancyId", "compartmentId"}, ForceNew: []string{"resourceId", "resourceRegion", "resourceType", "fleetId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-confirm-delete"},
	}
}

func newFleetResourceServiceClientWithOCIClients(log loggerutil.OSOKLogger, client fleetappssdk.FleetAppsManagementClient, workRequestClient fleetappssdk.FleetAppsManagementWorkRequestClient) FleetResourceServiceClient {
	manager := &FleetResourceServiceManager{Log: log}
	hooks := newFleetResourceDefaultRuntimeHooks(FleetResourceSDKClients{
		fleetAppsManagementClient:            client,
		fleetAppsManagementWorkRequestClient: workRequestClient,
	})
	applyFleetResourceRuntimeHooks(&hooks, workRequestClient, nil)
	delegate := defaultFleetResourceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*fleetappsv1beta1.FleetResource](buildFleetResourceGeneratedRuntimeConfig(manager, hooks))}
	return wrapFleetResourceGeneratedClient(hooks, delegate)
}
