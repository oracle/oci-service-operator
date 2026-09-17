/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package peer

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
	registerPeerRuntimeHooksMutator(func(manager *PeerServiceManager, hooks *PeerRuntimeHooks) {
		client, err := newPeerWorkRequestClient(manager)
		applyPeerRuntimeHooks(hooks, client, err)
	})
}

func newPeerWorkRequestClient(manager *PeerServiceManager) (nodeworkrequest.Client, error) {
	if manager == nil {
		return nil, fmt.Errorf("initialize Peer work-request client: service manager is nil")
	}
	client, err := blockchainsdk.NewBlockchainPlatformClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize Peer work-request client: %w", err)
	}
	return client, nil
}

func applyPeerRuntimeHooks(hooks *PeerRuntimeHooks, client nodeworkrequest.Client, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = peerRuntimeSemantics()
	hooks.StatusHooks.ProjectStatus = projectPeerStatus
	hooks.Async.Adapter = nodeworkrequest.Adapter()
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		return nodeworkrequest.Fetch(ctx, client, workRequestID)
	}
	hooks.Async.RecoverResourceID = func(_ *blockchainv1beta1.Peer, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
		return nodeworkrequest.RecoverResourceID(workRequest, phase, "peer")
	}
	hooks.Async.Message = func(phase shared.OSOKAsyncPhase, workRequest any) string {
		return nodeworkrequest.Message("Peer", phase, workRequest)
	}
}

func peerRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "blockchain", FormalSlug: "peer",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "workrequest", Runtime: "generatedruntime", FormalClassification: "workrequest", WorkRequest: &generatedruntime.WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "update", "delete"}}},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"role", "ad", "alias"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"ocpuAllocationParam"}, ForceNew: []string{"blockchainPlatformId", "role", "ad", "alias"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-read"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "work-request-then-confirm-delete"},
	}
}

func projectPeerStatus(resource *blockchainv1beta1.Peer, response any) error {
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, response, nil); err != nil {
		return err
	}
	if resource.Status.PeerKey != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.PeerKey)
	}
	return nil
}

func newPeerServiceClientWithOCIClient(log loggerutil.OSOKLogger, client blockchainsdk.BlockchainPlatformClient) PeerServiceClient {
	manager := &PeerServiceManager{Log: log}
	hooks := newPeerDefaultRuntimeHooks(client)
	applyPeerRuntimeHooks(&hooks, client, nil)
	delegate := defaultPeerServiceClient{ServiceClient: generatedruntime.NewServiceClient[*blockchainv1beta1.Peer](buildPeerGeneratedRuntimeConfig(manager, hooks))}
	return wrapPeerGeneratedClient(hooks, delegate)
}
