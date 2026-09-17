/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package osn

import (
	"context"
	"fmt"

	blockchainsdk "github.com/oracle/oci-go-sdk/v65/blockchain"
	blockchainv1beta1 "github.com/oracle/oci-service-operator/api/blockchain/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/blockchain/nodeworkrequest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerOsnRuntimeHooksMutator(func(manager *OsnServiceManager, hooks *OsnRuntimeHooks) {
		client, err := newOsnWorkRequestClient(manager)
		applyOsnRuntimeHooks(hooks, client, err)
	})
}

func newOsnWorkRequestClient(manager *OsnServiceManager) (nodeworkrequest.Client, error) {
	if manager == nil {
		return nil, fmt.Errorf("initialize Osn work-request client: service manager is nil")
	}
	client, err := blockchainsdk.NewBlockchainPlatformClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize Osn work-request client: %w", err)
	}
	return client, nil
}

func applyOsnRuntimeHooks(hooks *OsnRuntimeHooks, client nodeworkrequest.Client, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = osnRuntimeSemantics()
	hooks.StatusHooks.ProjectStatus = projectOsnStatus
	hooks.Async.Adapter = nodeworkrequest.Adapter()
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		return nodeworkrequest.Fetch(ctx, client, workRequestID)
	}
	hooks.Async.RecoverResourceID = func(_ *blockchainv1beta1.Osn, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
		return nodeworkrequest.RecoverResourceID(workRequest, phase, "osn", "ordering service node")
	}
	hooks.Async.Message = func(phase shared.OSOKAsyncPhase, workRequest any) string {
		return nodeworkrequest.Message("Osn", phase, workRequest)
	}
}

func osnRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "blockchain", FormalSlug: "osn",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "workrequest", Runtime: "generatedruntime", FormalClassification: "workrequest", WorkRequest: &generatedruntime.WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "update", "delete"}}},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"ad"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"ocpuAllocationParam"}, ForceNew: []string{"blockchainPlatformId", "ad"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-confirm-delete"},
	}
}

func projectOsnStatus(resource *blockchainv1beta1.Osn, response any) error {
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, response, nil); err != nil {
		return err
	}
	if resource.Status.OsnKey != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.OsnKey)
	}
	return nil
}

func newOsnServiceClientWithOCIClient(log loggerutil.OSOKLogger, client blockchainsdk.BlockchainPlatformClient) OsnServiceClient {
	manager := &OsnServiceManager{Log: log}
	hooks := newOsnDefaultRuntimeHooks(client)
	applyOsnRuntimeHooks(&hooks, client, nil)
	delegate := defaultOsnServiceClient{ServiceClient: generatedruntime.NewServiceClient[*blockchainv1beta1.Osn](buildOsnGeneratedRuntimeConfig(manager, hooks))}
	return wrapOsnGeneratedClient(hooks, delegate)
}
