/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package peertargetdatabase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	peerTargetDatabaseTargetDatabaseIDAnnotation = "datasafe.oracle.com/target-database-id"
	peerTargetDatabaseKeyAnnotation              = "datasafe.oracle.com/peer-target-database-key"
	peerTargetDatabaseRequeueDuration            = time.Minute
)

type peerTargetDatabaseOCIClient interface {
	CreatePeerTargetDatabase(context.Context, datasafesdk.CreatePeerTargetDatabaseRequest) (datasafesdk.CreatePeerTargetDatabaseResponse, error)
	GetPeerTargetDatabase(context.Context, datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error)
	ListPeerTargetDatabases(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error)
	UpdatePeerTargetDatabase(context.Context, datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error)
	DeletePeerTargetDatabase(context.Context, datasafesdk.DeletePeerTargetDatabaseRequest) (datasafesdk.DeletePeerTargetDatabaseResponse, error)
}

type peerTargetDatabaseIdentity struct {
	targetDatabaseID string
	key              int
}

type peerTargetDatabaseRuntimeClient struct {
	client peerTargetDatabaseOCIClient
	log    loggerutil.OSOKLogger
}

var _ PeerTargetDatabaseServiceClient = (*peerTargetDatabaseRuntimeClient)(nil)

func init() {
	registerPeerTargetDatabaseRuntimeHooksMutator(func(manager *PeerTargetDatabaseServiceManager, hooks *PeerTargetDatabaseRuntimeHooks) {
		applyPeerTargetDatabaseRuntimeHooks(manager, hooks)
	})
}

func applyPeerTargetDatabaseRuntimeHooks(manager *PeerTargetDatabaseServiceManager, hooks *PeerTargetDatabaseRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = peerTargetDatabaseRuntimeSemantics()
	runtimeClient := peerTargetDatabaseHookOCIClient{
		create: hooks.Create.Call,
		get:    hooks.Get.Call,
		list:   listPeerTargetDatabasesAllPages(hooks.List.Call),
		update: hooks.Update.Call,
		delete: hooks.Delete.Call,
	}
	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(_ PeerTargetDatabaseServiceClient) PeerTargetDatabaseServiceClient {
		return &peerTargetDatabaseRuntimeClient{
			client: runtimeClient,
			log:    log,
		}
	})
}

type peerTargetDatabaseHookOCIClient struct {
	create func(context.Context, datasafesdk.CreatePeerTargetDatabaseRequest) (datasafesdk.CreatePeerTargetDatabaseResponse, error)
	get    func(context.Context, datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error)
	list   func(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error)
	update func(context.Context, datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error)
	delete func(context.Context, datasafesdk.DeletePeerTargetDatabaseRequest) (datasafesdk.DeletePeerTargetDatabaseResponse, error)
}

func (c peerTargetDatabaseHookOCIClient) CreatePeerTargetDatabase(ctx context.Context, request datasafesdk.CreatePeerTargetDatabaseRequest) (datasafesdk.CreatePeerTargetDatabaseResponse, error) {
	if c.create == nil {
		return datasafesdk.CreatePeerTargetDatabaseResponse{}, fmt.Errorf("PeerTargetDatabase create hook is not configured")
	}
	return c.create(ctx, request)
}

func (c peerTargetDatabaseHookOCIClient) GetPeerTargetDatabase(ctx context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
	if c.get == nil {
		return datasafesdk.GetPeerTargetDatabaseResponse{}, fmt.Errorf("PeerTargetDatabase get hook is not configured")
	}
	return c.get(ctx, request)
}

func (c peerTargetDatabaseHookOCIClient) ListPeerTargetDatabases(ctx context.Context, request datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
	if c.list == nil {
		return datasafesdk.ListPeerTargetDatabasesResponse{}, fmt.Errorf("PeerTargetDatabase list hook is not configured")
	}
	return c.list(ctx, request)
}

func (c peerTargetDatabaseHookOCIClient) UpdatePeerTargetDatabase(ctx context.Context, request datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error) {
	if c.update == nil {
		return datasafesdk.UpdatePeerTargetDatabaseResponse{}, fmt.Errorf("PeerTargetDatabase update hook is not configured")
	}
	return c.update(ctx, request)
}

func (c peerTargetDatabaseHookOCIClient) DeletePeerTargetDatabase(ctx context.Context, request datasafesdk.DeletePeerTargetDatabaseRequest) (datasafesdk.DeletePeerTargetDatabaseResponse, error) {
	if c.delete == nil {
		return datasafesdk.DeletePeerTargetDatabaseResponse{}, fmt.Errorf("PeerTargetDatabase delete hook is not configured")
	}
	return c.delete(ctx, request)
}

func newPeerTargetDatabaseServiceClientWithOCIClient(client peerTargetDatabaseOCIClient) PeerTargetDatabaseServiceClient {
	return &peerTargetDatabaseRuntimeClient{client: client}
}

func peerTargetDatabaseRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datasafe",
		FormalSlug:    "peertargetdatabase",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.TargetDatabaseLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.TargetDatabaseLifecycleStateUpdating)},
			ActiveStates: []string{
				string(datasafesdk.TargetDatabaseLifecycleStateActive),
				string(datasafesdk.TargetDatabaseLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.TargetDatabaseLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.TargetDatabaseLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"dataguardAssociationId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"displayName", "description", "databaseDetails", "tlsConfig"},
			ForceNew: []string{"targetDatabaseId", "dataguardAssociationId"},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "resource-local PeerTargetDatabase runtime", Action: "CreatePeerTargetDatabase"}},
			Update: []generatedruntime.Hook{{Helper: "resource-local PeerTargetDatabase runtime", Action: "UpdatePeerTargetDatabase"}},
			Delete: []generatedruntime.Hook{{Helper: "resource-local PeerTargetDatabase runtime", Action: "DeletePeerTargetDatabase"}},
		},
		CreateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "create-response"},
		UpdateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func (c *peerTargetDatabaseRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	identity, err := resolvePeerTargetDatabaseIdentity(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if c.client == nil {
		return c.fail(resource, fmt.Errorf("PeerTargetDatabase OCI client is not configured"))
	}

	current, found, err := c.findPeerTargetDatabase(ctx, resource, identity)
	if err != nil {
		return c.fail(resource, err)
	}
	if !found {
		return c.createPeerTargetDatabase(ctx, resource, identity)
	}

	projectPeerTargetDatabaseStatus(resource, identity.targetDatabaseID, current)
	if isPeerTargetDatabaseWritePending(current) {
		return c.success(resource, current, shared.Updating)
	}
	return c.updatePeerTargetDatabaseIfNeeded(ctx, resource, identity, current)
}

func (c *peerTargetDatabaseRuntimeClient) updatePeerTargetDatabaseIfNeeded(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	identity peerTargetDatabaseIdentity,
	current datasafesdk.PeerTargetDatabase,
) (servicemanager.OSOKResponse, error) {
	if err := validatePeerTargetDatabaseCreateOnly(resource, current); err != nil {
		return c.fail(resource, err)
	}
	details, updateNeeded, err := buildPeerTargetDatabaseUpdateBody(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.success(resource, current, shared.Active)
	}

	response, err := c.client.UpdatePeerTargetDatabase(ctx, datasafesdk.UpdatePeerTargetDatabaseRequest{
		TargetDatabaseId:                common.String(identity.targetDatabaseID),
		PeerTargetDatabaseId:            common.Int(currentPeerTargetDatabaseKey(resource, current)),
		UpdatePeerTargetDatabaseDetails: details,
		OpcRetryToken:                   common.String(peerTargetDatabaseRetryToken(resource)),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	recordPeerTargetDatabaseWorkRequest(resource, response.OpcWorkRequestId, shared.OSOKAsyncPhaseUpdate)

	refreshed, err := c.getPeerTargetDatabase(ctx, identity.targetDatabaseID, currentPeerTargetDatabaseKey(resource, current))
	if err != nil {
		return c.fail(resource, err)
	}
	projectPeerTargetDatabaseStatus(resource, identity.targetDatabaseID, refreshed)
	return c.success(resource, refreshed, shared.Updating)
}

func (c *peerTargetDatabaseRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.PeerTargetDatabase) (bool, error) {
	identity, recorded, err := c.resolveDeleteIdentity(resource)
	if err != nil {
		return false, err
	}
	if !recorded {
		markPeerTargetDatabaseDeleted(resource, "OCI peer target database was never recorded")
		return true, nil
	}

	current, found, err := c.findPeerTargetDatabase(ctx, resource, identity)
	if err != nil {
		return false, c.markDeleteFailure(resource, err)
	}
	if deleted, handled := handlePeerTargetDatabaseObservedDeleteState(resource, identity.targetDatabaseID, current, found); handled {
		return deleted, nil
	}

	return c.deleteAndConfirmPeerTargetDatabase(ctx, resource, identity.targetDatabaseID, currentPeerTargetDatabaseKey(resource, current))
}

func (c *peerTargetDatabaseRuntimeClient) resolveDeleteIdentity(resource *datasafev1beta1.PeerTargetDatabase) (peerTargetDatabaseIdentity, bool, error) {
	if resource == nil {
		return peerTargetDatabaseIdentity{}, false, fmt.Errorf("PeerTargetDatabase resource is nil")
	}
	key, err := peerTargetDatabaseDeleteKey(resource)
	if err != nil {
		return peerTargetDatabaseIdentity{}, false, err
	}
	if key == 0 {
		return peerTargetDatabaseIdentity{}, false, nil
	}
	if c.client == nil {
		return peerTargetDatabaseIdentity{}, false, fmt.Errorf("PeerTargetDatabase OCI client is not configured")
	}
	identity, err := resolvePeerTargetDatabaseIdentity(resource)
	if err != nil {
		return peerTargetDatabaseIdentity{}, false, err
	}
	return identity, true, nil
}

func handlePeerTargetDatabaseObservedDeleteState(
	resource *datasafev1beta1.PeerTargetDatabase,
	targetDatabaseID string,
	current datasafesdk.PeerTargetDatabase,
	found bool,
) (bool, bool) {
	if !found {
		markPeerTargetDatabaseDeleted(resource, "OCI peer target database is not recorded")
		return true, true
	}
	projectPeerTargetDatabaseStatus(resource, targetDatabaseID, current)
	if isPeerTargetDatabaseDeleted(current) {
		markPeerTargetDatabaseDeleted(resource, "OCI peer target database is deleted")
		return true, true
	}
	if isPeerTargetDatabaseDeletePending(current) {
		markPeerTargetDatabaseTerminating(resource, current)
		return false, true
	}
	return false, false
}

func (c *peerTargetDatabaseRuntimeClient) deleteAndConfirmPeerTargetDatabase(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	targetDatabaseID string,
	key int,
) (bool, error) {
	response, err := c.client.DeletePeerTargetDatabase(ctx, datasafesdk.DeletePeerTargetDatabaseRequest{
		TargetDatabaseId:     common.String(targetDatabaseID),
		PeerTargetDatabaseId: common.Int(key),
	})
	if err != nil {
		if deleted, handled := c.handleDeleteCallError(resource, err); handled {
			return deleted, nil
		}
		return false, c.markDeleteFailure(resource, peerTargetDatabaseConservativeDeleteCallError(err))
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	recordPeerTargetDatabaseWorkRequest(resource, response.OpcWorkRequestId, shared.OSOKAsyncPhaseDelete)

	confirmed, err := c.getPeerTargetDatabase(ctx, targetDatabaseID, key)
	if err != nil {
		if deleted, handled := c.handleDeleteConfirmError(resource, err); handled {
			return deleted, nil
		}
		return false, c.markDeleteFailure(resource, peerTargetDatabaseConservativeConfirmError(err))
	}
	projectPeerTargetDatabaseStatus(resource, targetDatabaseID, confirmed)
	if isPeerTargetDatabaseDeleted(confirmed) {
		markPeerTargetDatabaseDeleted(resource, "OCI peer target database is deleted")
		return true, nil
	}
	markPeerTargetDatabaseTerminating(resource, confirmed)
	return false, nil
}

func (c *peerTargetDatabaseRuntimeClient) createPeerTargetDatabase(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	identity peerTargetDatabaseIdentity,
) (servicemanager.OSOKResponse, error) {
	details, err := buildPeerTargetDatabaseCreateBody(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	response, err := c.client.CreatePeerTargetDatabase(ctx, datasafesdk.CreatePeerTargetDatabaseRequest{
		TargetDatabaseId:                common.String(identity.targetDatabaseID),
		CreatePeerTargetDatabaseDetails: details,
		OpcRetryToken:                   common.String(peerTargetDatabaseRetryToken(resource)),
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	recordPeerTargetDatabaseWorkRequest(resource, response.OpcWorkRequestId, shared.OSOKAsyncPhaseCreate)
	projectPeerTargetDatabaseStatus(resource, identity.targetDatabaseID, response.PeerTargetDatabase)
	return c.success(resource, response.PeerTargetDatabase, shared.Provisioning)
}

func (c *peerTargetDatabaseRuntimeClient) findPeerTargetDatabase(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	identity peerTargetDatabaseIdentity,
) (datasafesdk.PeerTargetDatabase, bool, error) {
	if identity.key != 0 {
		return c.findPeerTargetDatabaseByKey(ctx, identity.targetDatabaseID, identity.key)
	}
	return c.findPeerTargetDatabaseBySummary(ctx, resource, identity.targetDatabaseID)
}

func (c *peerTargetDatabaseRuntimeClient) findPeerTargetDatabaseByKey(
	ctx context.Context,
	targetDatabaseID string,
	key int,
) (datasafesdk.PeerTargetDatabase, bool, error) {
	current, err := c.getPeerTargetDatabase(ctx, targetDatabaseID, key)
	if err != nil {
		return peerTargetDatabaseFromRecordedKeyReadError(err)
	}
	return current, true, nil
}

func (c *peerTargetDatabaseRuntimeClient) findPeerTargetDatabaseBySummary(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	targetDatabaseID string,
) (datasafesdk.PeerTargetDatabase, bool, error) {
	summary, found, err := c.findPeerTargetDatabaseSummary(ctx, resource, targetDatabaseID)
	if err != nil {
		return datasafesdk.PeerTargetDatabase{}, false, peerTargetDatabaseConservativeReadError(err)
	}
	if !found {
		return datasafesdk.PeerTargetDatabase{}, false, nil
	}
	key, err := peerTargetDatabaseSummaryKey(summary)
	if err != nil {
		return datasafesdk.PeerTargetDatabase{}, false, err
	}
	current, err := c.getPeerTargetDatabase(ctx, targetDatabaseID, key)
	if err != nil {
		return datasafesdk.PeerTargetDatabase{}, false, peerTargetDatabaseConservativeSummaryReadError(err)
	}
	return current, true, nil
}

func peerTargetDatabaseFromRecordedKeyReadError(err error) (datasafesdk.PeerTargetDatabase, bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsAuthShapedNotFound() {
		return datasafesdk.PeerTargetDatabase{}, false, peerTargetDatabaseAmbiguousNotFoundError("read", err)
	}
	if classification.IsUnambiguousNotFound() {
		return datasafesdk.PeerTargetDatabase{}, false, nil
	}
	return datasafesdk.PeerTargetDatabase{}, false, err
}

func peerTargetDatabaseSummaryKey(summary datasafesdk.PeerTargetDatabaseSummary) (int, error) {
	if summary.Key == nil || *summary.Key == 0 {
		return 0, fmt.Errorf("PeerTargetDatabase list match did not include key")
	}
	return *summary.Key, nil
}

func peerTargetDatabaseConservativeSummaryReadError(err error) error {
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return peerTargetDatabaseAmbiguousNotFoundError("read", err)
	}
	return err
}

func (c *peerTargetDatabaseRuntimeClient) getPeerTargetDatabase(
	ctx context.Context,
	targetDatabaseID string,
	key int,
) (datasafesdk.PeerTargetDatabase, error) {
	response, err := c.client.GetPeerTargetDatabase(ctx, datasafesdk.GetPeerTargetDatabaseRequest{
		TargetDatabaseId:     common.String(targetDatabaseID),
		PeerTargetDatabaseId: common.Int(key),
	})
	if err != nil {
		return datasafesdk.PeerTargetDatabase{}, err
	}
	return response.PeerTargetDatabase, nil
}

func (c *peerTargetDatabaseRuntimeClient) listPeerTargetDatabases(
	ctx context.Context,
	request datasafesdk.ListPeerTargetDatabasesRequest,
) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
	return listPeerTargetDatabasesAllPages(c.client.ListPeerTargetDatabases)(ctx, request)
}

func (c *peerTargetDatabaseRuntimeClient) findPeerTargetDatabaseSummary(
	ctx context.Context,
	resource *datasafev1beta1.PeerTargetDatabase,
	targetDatabaseID string,
) (datasafesdk.PeerTargetDatabaseSummary, bool, error) {
	if err := requirePeerTargetDatabaseBindCriteria(resource); err != nil {
		return datasafesdk.PeerTargetDatabaseSummary{}, false, err
	}
	response, err := c.listPeerTargetDatabases(ctx, datasafesdk.ListPeerTargetDatabasesRequest{
		TargetDatabaseId: common.String(targetDatabaseID),
	})
	if err != nil {
		return datasafesdk.PeerTargetDatabaseSummary{}, false, err
	}

	var matches []datasafesdk.PeerTargetDatabaseSummary
	for _, item := range response.Items {
		if peerTargetDatabaseSummaryMatches(resource, item) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return datasafesdk.PeerTargetDatabaseSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return datasafesdk.PeerTargetDatabaseSummary{}, false, fmt.Errorf("PeerTargetDatabase list returned multiple matches for the desired identity")
	}
}

func (c *peerTargetDatabaseRuntimeClient) success(
	resource *datasafev1beta1.PeerTargetDatabase,
	current datasafesdk.PeerTargetDatabase,
	fallback shared.OSOKConditionType,
) (servicemanager.OSOKResponse, error) {
	condition, requeue, message := peerTargetDatabaseCondition(current, fallback)
	applyPeerTargetDatabaseCondition(resource, condition, message, lifecyclePhase(condition), lifecycleRawState(current), c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   requeue,
		RequeueDuration: peerTargetDatabaseRequeue(requeue),
	}, nil
}

func (c *peerTargetDatabaseRuntimeClient) fail(
	resource *datasafev1beta1.PeerTargetDatabase,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		applyPeerTargetDatabaseCondition(resource, shared.Failed, err.Error(), "", "", c.log)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *peerTargetDatabaseRuntimeClient) markDeleteFailure(resource *datasafev1beta1.PeerTargetDatabase, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		applyPeerTargetDatabaseCondition(resource, shared.Failed, err.Error(), shared.OSOKAsyncPhaseDelete, "", c.log)
	}
	return err
}

func (c *peerTargetDatabaseRuntimeClient) handleDeleteCallError(resource *datasafev1beta1.PeerTargetDatabase, err error) (bool, bool) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsAuthShapedNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, false
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markPeerTargetDatabaseDeleted(resource, "OCI peer target database no longer exists")
		return true, true
	default:
		return false, false
	}
}

func (c *peerTargetDatabaseRuntimeClient) handleDeleteConfirmError(resource *datasafev1beta1.PeerTargetDatabase, err error) (bool, bool) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsAuthShapedNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, false
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markPeerTargetDatabaseDeleted(resource, "OCI peer target database is deleted")
		return true, true
	default:
		return false, false
	}
}

func peerTargetDatabaseConservativeReadError(err error) error {
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return peerTargetDatabaseAmbiguousNotFoundError("read", err)
	}
	return err
}

func peerTargetDatabaseConservativeDeleteCallError(err error) error {
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return peerTargetDatabaseAmbiguousNotFoundError("delete", err)
	}
	return err
}

func peerTargetDatabaseConservativeConfirmError(err error) error {
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return peerTargetDatabaseAmbiguousNotFoundError("delete confirmation", err)
	}
	return err
}

func peerTargetDatabaseAmbiguousNotFoundError(action string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "ambiguous 404 "+errorutil.NotAuthorizedOrNotFound) {
		return err
	}
	return fmt.Errorf("PeerTargetDatabase %s returned ambiguous 404 %s; refusing to treat it as deleted: %w", action, errorutil.NotAuthorizedOrNotFound, err)
}

func resolvePeerTargetDatabaseIdentity(resource *datasafev1beta1.PeerTargetDatabase) (peerTargetDatabaseIdentity, error) {
	if resource == nil {
		return peerTargetDatabaseIdentity{}, fmt.Errorf("PeerTargetDatabase resource is nil")
	}

	annotationTargetID := strings.TrimSpace(resource.GetAnnotations()[peerTargetDatabaseTargetDatabaseIDAnnotation])
	recordedTargetID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if annotationTargetID != "" && recordedTargetID != "" && annotationTargetID != recordedTargetID {
		return peerTargetDatabaseIdentity{}, fmt.Errorf(
			"PeerTargetDatabase %s changed from recorded targetDatabaseId %q to %q",
			peerTargetDatabaseTargetDatabaseIDAnnotation,
			recordedTargetID,
			annotationTargetID,
		)
	}
	targetID := firstNonEmptyString(recordedTargetID, annotationTargetID)
	if targetID == "" {
		return peerTargetDatabaseIdentity{}, fmt.Errorf(
			"PeerTargetDatabase requires metadata annotation %q with the parent target database OCID",
			peerTargetDatabaseTargetDatabaseIDAnnotation,
		)
	}

	annotationKey, err := peerTargetDatabaseAnnotationKey(resource)
	if err != nil {
		return peerTargetDatabaseIdentity{}, err
	}
	statusKey := resource.Status.Key
	if annotationKey != 0 && statusKey != 0 && annotationKey != statusKey {
		return peerTargetDatabaseIdentity{}, fmt.Errorf(
			"PeerTargetDatabase %s changed from recorded key %d to %d",
			peerTargetDatabaseKeyAnnotation,
			statusKey,
			annotationKey,
		)
	}
	return peerTargetDatabaseIdentity{
		targetDatabaseID: targetID,
		key:              firstNonZeroInt(statusKey, annotationKey),
	}, nil
}

func peerTargetDatabaseAnnotationKey(resource *datasafev1beta1.PeerTargetDatabase) (int, error) {
	raw := strings.TrimSpace(resource.GetAnnotations()[peerTargetDatabaseKeyAnnotation])
	if raw == "" {
		return 0, nil
	}
	var key int
	if _, err := fmt.Sscanf(raw, "%d", &key); err != nil || key <= 0 {
		return 0, fmt.Errorf("PeerTargetDatabase annotation %q must be a positive integer", peerTargetDatabaseKeyAnnotation)
	}
	return key, nil
}

func peerTargetDatabaseDeleteKey(resource *datasafev1beta1.PeerTargetDatabase) (int, error) {
	annotationKey, err := peerTargetDatabaseAnnotationKey(resource)
	if err != nil {
		return 0, err
	}
	statusKey := resource.Status.Key
	if annotationKey != 0 && statusKey != 0 && annotationKey != statusKey {
		return 0, fmt.Errorf(
			"PeerTargetDatabase %s changed from recorded key %d to %d",
			peerTargetDatabaseKeyAnnotation,
			statusKey,
			annotationKey,
		)
	}
	return firstNonZeroInt(statusKey, annotationKey), nil
}

func requirePeerTargetDatabaseBindCriteria(resource *datasafev1beta1.PeerTargetDatabase) error {
	if resource == nil {
		return fmt.Errorf("PeerTargetDatabase resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DataguardAssociationId) != "" {
		return nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return nil
	}
	return fmt.Errorf("PeerTargetDatabase requires spec.dataguardAssociationId or spec.displayName for idempotent create-or-bind")
}

func peerTargetDatabaseSummaryMatches(resource *datasafev1beta1.PeerTargetDatabase, item datasafesdk.PeerTargetDatabaseSummary) bool {
	if desired := strings.TrimSpace(resource.Spec.DataguardAssociationId); desired != "" {
		return desired == stringValue(item.DataguardAssociationId)
	}
	if desired := strings.TrimSpace(resource.Spec.DisplayName); desired != "" {
		return desired == stringValue(item.DisplayName)
	}
	return false
}

func validatePeerTargetDatabaseCreateOnly(
	resource *datasafev1beta1.PeerTargetDatabase,
	current datasafesdk.PeerTargetDatabase,
) error {
	if desired := strings.TrimSpace(resource.Spec.DataguardAssociationId); desired != "" && desired != stringValue(current.DataguardAssociationId) {
		return fmt.Errorf("PeerTargetDatabase spec.dataguardAssociationId is create-only and cannot change from %q to %q", stringValue(current.DataguardAssociationId), desired)
	}
	return nil
}

func buildPeerTargetDatabaseCreateBody(resource *datasafev1beta1.PeerTargetDatabase) (datasafesdk.CreatePeerTargetDatabaseDetails, error) {
	if resource == nil {
		return datasafesdk.CreatePeerTargetDatabaseDetails{}, fmt.Errorf("PeerTargetDatabase resource is nil")
	}
	databaseDetails, err := peerTargetDatabaseDatabaseDetails(resource.Spec.DatabaseDetails)
	if err != nil {
		return datasafesdk.CreatePeerTargetDatabaseDetails{}, fmt.Errorf("databaseDetails: %w", err)
	}
	tlsConfig, err := peerTargetDatabaseTLSConfig(resource.Spec.TlsConfig)
	if err != nil {
		return datasafesdk.CreatePeerTargetDatabaseDetails{}, fmt.Errorf("tlsConfig: %w", err)
	}

	details := datasafesdk.CreatePeerTargetDatabaseDetails{
		DatabaseDetails: databaseDetails,
	}
	if value := strings.TrimSpace(resource.Spec.DisplayName); value != "" {
		details.DisplayName = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.Description); value != "" {
		details.Description = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.DataguardAssociationId); value != "" {
		details.DataguardAssociationId = common.String(value)
	}
	details.TlsConfig = tlsConfig
	return details, nil
}

func buildPeerTargetDatabaseUpdateBody(
	resource *datasafev1beta1.PeerTargetDatabase,
	current datasafesdk.PeerTargetDatabase,
) (datasafesdk.UpdatePeerTargetDatabaseDetails, bool, error) {
	if resource == nil {
		return datasafesdk.UpdatePeerTargetDatabaseDetails{}, false, fmt.Errorf("PeerTargetDatabase resource is nil")
	}
	desired, err := desiredPeerTargetDatabaseUpdateFields(resource)
	if err != nil {
		return datasafesdk.UpdatePeerTargetDatabaseDetails{}, false, err
	}

	details := datasafesdk.UpdatePeerTargetDatabaseDetails{}
	updateNeeded := applyPeerTargetDatabaseScalarUpdates(resource, current, &details)
	updateNeeded = applyPeerTargetDatabaseDatabaseDetailsUpdate(desired.databaseDetails, current, &details) || updateNeeded
	updateNeeded = applyPeerTargetDatabaseTLSConfigUpdate(desired.tlsConfig, current, &details) || updateNeeded
	return details, updateNeeded, nil
}

type peerTargetDatabaseUpdateFields struct {
	databaseDetails datasafesdk.DatabaseDetails
	tlsConfig       *datasafesdk.TlsConfig
}

func desiredPeerTargetDatabaseUpdateFields(resource *datasafev1beta1.PeerTargetDatabase) (peerTargetDatabaseUpdateFields, error) {
	databaseDetails, err := peerTargetDatabaseDatabaseDetails(resource.Spec.DatabaseDetails)
	if err != nil {
		return peerTargetDatabaseUpdateFields{}, fmt.Errorf("databaseDetails: %w", err)
	}
	tlsConfig, err := peerTargetDatabaseTLSConfig(resource.Spec.TlsConfig)
	if err != nil {
		return peerTargetDatabaseUpdateFields{}, fmt.Errorf("tlsConfig: %w", err)
	}
	return peerTargetDatabaseUpdateFields{
		databaseDetails: databaseDetails,
		tlsConfig:       tlsConfig,
	}, nil
}

func applyPeerTargetDatabaseScalarUpdates(
	resource *datasafev1beta1.PeerTargetDatabase,
	current datasafesdk.PeerTargetDatabase,
	details *datasafesdk.UpdatePeerTargetDatabaseDetails,
) bool {
	updateNeeded := false
	if value := strings.TrimSpace(resource.Spec.DisplayName); value != "" && value != stringValue(current.DisplayName) {
		details.DisplayName = common.String(value)
		updateNeeded = true
	}
	if value := strings.TrimSpace(resource.Spec.Description); value != "" && value != stringValue(current.Description) {
		details.Description = common.String(value)
		updateNeeded = true
	}
	return updateNeeded
}

func applyPeerTargetDatabaseDatabaseDetailsUpdate(
	databaseDetails datasafesdk.DatabaseDetails,
	current datasafesdk.PeerTargetDatabase,
	details *datasafesdk.UpdatePeerTargetDatabaseDetails,
) bool {
	if !jsonEqual(databaseDetails, current.DatabaseDetails) {
		details.DatabaseDetails = databaseDetails
		return true
	}
	return false
}

func applyPeerTargetDatabaseTLSConfigUpdate(
	tlsConfig *datasafesdk.TlsConfig,
	current datasafesdk.PeerTargetDatabase,
	details *datasafesdk.UpdatePeerTargetDatabaseDetails,
) bool {
	if tlsConfig != nil && !jsonEqual(peerTargetDatabaseComparableTLSConfig(tlsConfig), peerTargetDatabaseComparableTLSConfig(current.TlsConfig)) {
		details.TlsConfig = tlsConfig
		return true
	}
	return false
}

func peerTargetDatabaseDatabaseDetails(spec datasafev1beta1.PeerTargetDatabaseDatabaseDetails) (datasafesdk.DatabaseDetails, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return peerTargetDatabaseDatabaseDetailsFromJSON([]byte(spec.JsonData))
	}
	infrastructureType := datasafesdk.InfrastructureTypeEnum(strings.ToUpper(strings.TrimSpace(spec.InfrastructureType)))
	if infrastructureType == "" {
		return nil, fmt.Errorf("infrastructureType is required")
	}
	switch peerTargetDatabaseDatabaseType(spec) {
	case string(datasafesdk.DatabaseTypeAutonomousDatabase):
		if strings.TrimSpace(spec.AutonomousDatabaseId) == "" {
			return nil, fmt.Errorf("autonomousDatabaseId is required for AUTONOMOUS_DATABASE")
		}
		return datasafesdk.AutonomousDatabaseDetails{
			InfrastructureType:   infrastructureType,
			AutonomousDatabaseId: common.String(strings.TrimSpace(spec.AutonomousDatabaseId)),
		}, nil
	case string(datasafesdk.DatabaseTypeDatabaseCloudService):
		return datasafesdk.DatabaseCloudServiceDetails{
			InfrastructureType:  infrastructureType,
			VmClusterId:         stringPointer(spec.VmClusterId),
			DbSystemId:          stringPointer(spec.DbSystemId),
			PluggableDatabaseId: stringPointer(spec.PluggableDatabaseId),
			ListenerPort:        intPointer(spec.ListenerPort),
			ServiceName:         stringPointer(spec.ServiceName),
		}, nil
	case string(datasafesdk.DatabaseTypeInstalledDatabase):
		if spec.ListenerPort == 0 {
			return nil, fmt.Errorf("listenerPort is required for INSTALLED_DATABASE")
		}
		if strings.TrimSpace(spec.ServiceName) == "" {
			return nil, fmt.Errorf("serviceName is required for INSTALLED_DATABASE")
		}
		return datasafesdk.InstalledDatabaseDetails{
			InfrastructureType: infrastructureType,
			ListenerPort:       common.Int(spec.ListenerPort),
			ServiceName:        common.String(strings.TrimSpace(spec.ServiceName)),
			InstanceId:         stringPointer(spec.InstanceId),
			IpAddresses:        cloneStringSlice(spec.IpAddresses),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported databaseType %q", spec.DatabaseType)
	}
}

func peerTargetDatabaseDatabaseDetailsFromJSON(payload []byte) (datasafesdk.DatabaseDetails, error) {
	var discriminator struct {
		DatabaseType string `json:"databaseType"`
	}
	if err := json.Unmarshal(payload, &discriminator); err != nil {
		return nil, fmt.Errorf("decode databaseType: %w", err)
	}
	switch strings.ToUpper(strings.TrimSpace(discriminator.DatabaseType)) {
	case string(datasafesdk.DatabaseTypeAutonomousDatabase):
		var details datasafesdk.AutonomousDatabaseDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode AutonomousDatabaseDetails: %w", err)
		}
		return details, nil
	case string(datasafesdk.DatabaseTypeDatabaseCloudService):
		var details datasafesdk.DatabaseCloudServiceDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode DatabaseCloudServiceDetails: %w", err)
		}
		return details, nil
	case string(datasafesdk.DatabaseTypeInstalledDatabase):
		var details datasafesdk.InstalledDatabaseDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("decode InstalledDatabaseDetails: %w", err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("unsupported databaseType %q", discriminator.DatabaseType)
	}
}

func peerTargetDatabaseDatabaseType(spec datasafev1beta1.PeerTargetDatabaseDatabaseDetails) string {
	if value := strings.ToUpper(strings.TrimSpace(spec.DatabaseType)); value != "" {
		return value
	}
	switch {
	case strings.TrimSpace(spec.AutonomousDatabaseId) != "":
		return string(datasafesdk.DatabaseTypeAutonomousDatabase)
	case strings.TrimSpace(spec.VmClusterId) != "" || strings.TrimSpace(spec.DbSystemId) != "" || strings.TrimSpace(spec.PluggableDatabaseId) != "":
		return string(datasafesdk.DatabaseTypeDatabaseCloudService)
	default:
		return string(datasafesdk.DatabaseTypeInstalledDatabase)
	}
}

func peerTargetDatabaseTLSConfig(spec datasafev1beta1.PeerTargetDatabaseTlsConfig) (*datasafesdk.TlsConfig, error) {
	if spec == (datasafev1beta1.PeerTargetDatabaseTlsConfig{}) {
		return nil, nil
	}
	if strings.TrimSpace(spec.Status) == "" {
		return nil, fmt.Errorf("status is required when tlsConfig is set")
	}
	return &datasafesdk.TlsConfig{
		Status:               datasafesdk.TlsConfigStatusEnum(strings.ToUpper(strings.TrimSpace(spec.Status))),
		CertificateStoreType: datasafesdk.TlsConfigCertificateStoreTypeEnum(strings.ToUpper(strings.TrimSpace(spec.CertificateStoreType))),
		StorePassword:        stringPointer(spec.StorePassword),
		TrustStoreContent:    stringPointer(spec.TrustStoreContent),
		KeyStoreContent:      stringPointer(spec.KeyStoreContent),
	}, nil
}

func projectPeerTargetDatabaseStatus(
	resource *datasafev1beta1.PeerTargetDatabase,
	targetDatabaseID string,
	current datasafesdk.PeerTargetDatabase,
) {
	if resource == nil {
		return
	}
	status := &resource.Status
	status.DisplayName = stringValue(current.DisplayName)
	status.Key = intValue(current.Key)
	status.DataguardAssociationId = stringValue(current.DataguardAssociationId)
	status.TimeCreated = sdkTimeString(current.TimeCreated)
	status.DatabaseDetails = peerTargetDatabaseDatabaseDetailsStatus(current.DatabaseDetails)
	status.LifecycleState = string(current.LifecycleState)
	status.Description = stringValue(current.Description)
	status.Role = stringValue(current.Role)
	status.DatabaseUniqueName = stringValue(current.DatabaseUniqueName)
	status.TlsConfig = peerTargetDatabaseTLSConfigStatus(current.TlsConfig)
	status.LifecycleDetails = stringValue(current.LifecycleDetails)
	status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(targetDatabaseID))
	if status.Key != 0 && status.OsokStatus.CreatedAt == nil {
		now := metav1.Now()
		status.OsokStatus.CreatedAt = &now
	}
}

func peerTargetDatabaseDatabaseDetailsStatus(details datasafesdk.DatabaseDetails) datasafev1beta1.PeerTargetDatabaseDatabaseDetails {
	if details == nil {
		return datasafev1beta1.PeerTargetDatabaseDatabaseDetails{}
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return datasafev1beta1.PeerTargetDatabaseDatabaseDetails{}
	}
	var status datasafev1beta1.PeerTargetDatabaseDatabaseDetails
	if err := json.Unmarshal(payload, &status); err != nil {
		return datasafev1beta1.PeerTargetDatabaseDatabaseDetails{}
	}
	status.JsonData = ""
	return status
}

func peerTargetDatabaseTLSConfigStatus(config *datasafesdk.TlsConfig) datasafev1beta1.PeerTargetDatabaseTlsConfig {
	if config == nil {
		return datasafev1beta1.PeerTargetDatabaseTlsConfig{}
	}
	return datasafev1beta1.PeerTargetDatabaseTlsConfig{
		Status:               string(config.Status),
		CertificateStoreType: string(config.CertificateStoreType),
	}
}

func peerTargetDatabaseComparableTLSConfig(config *datasafesdk.TlsConfig) datasafesdk.TlsConfig {
	if config == nil {
		return datasafesdk.TlsConfig{}
	}
	return datasafesdk.TlsConfig{
		Status:               config.Status,
		CertificateStoreType: config.CertificateStoreType,
	}
}

func markPeerTargetDatabaseDeleted(resource *datasafev1beta1.PeerTargetDatabase, message string) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func markPeerTargetDatabaseTerminating(resource *datasafev1beta1.PeerTargetDatabase, current datasafesdk.PeerTargetDatabase) {
	applyPeerTargetDatabaseCondition(
		resource,
		shared.Terminating,
		firstNonEmptyString(stringValue(current.LifecycleDetails), "OCI peer target database delete is in progress"),
		shared.OSOKAsyncPhaseDelete,
		lifecycleRawState(current),
		loggerutil.OSOKLogger{},
	)
}

func applyPeerTargetDatabaseCondition(
	resource *datasafev1beta1.PeerTargetDatabase,
	condition shared.OSOKConditionType,
	message string,
	phase shared.OSOKAsyncPhase,
	rawState string,
	log loggerutil.OSOKLogger,
) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = message
	status.Reason = string(condition)
	status.UpdatedAt = &now
	if shouldTrackPeerTargetDatabaseAsync(condition) && phase != "" {
		current := &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			RawStatus:       rawState,
			NormalizedClass: peerTargetDatabaseAsyncClass(condition),
			Message:         message,
			UpdatedAt:       &now,
		}
		if status.Async.Current != nil &&
			status.Async.Current.Source == shared.OSOKAsyncSourceWorkRequest &&
			status.Async.Current.Phase == phase &&
			status.Async.Current.WorkRequestID != "" {
			current.Source = shared.OSOKAsyncSourceWorkRequest
			current.WorkRequestID = status.Async.Current.WorkRequestID
		}
		status.Async.Current = current
	} else {
		status.Async.Current = nil
	}
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, log)
}

func recordPeerTargetDatabaseWorkRequest(
	resource *datasafev1beta1.PeerTargetDatabase,
	workRequestID *string,
	phase shared.OSOKAsyncPhase,
) {
	if resource == nil || workRequestID == nil || strings.TrimSpace(*workRequestID) == "" {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   strings.TrimSpace(*workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
}

func peerTargetDatabaseCondition(
	current datasafesdk.PeerTargetDatabase,
	fallback shared.OSOKConditionType,
) (shared.OSOKConditionType, bool, string) {
	state := strings.ToUpper(string(current.LifecycleState))
	message := firstNonEmptyString(stringValue(current.LifecycleDetails), state)
	switch state {
	case string(datasafesdk.TargetDatabaseLifecycleStateCreating):
		return shared.Provisioning, true, firstNonEmptyString(message, "OCI peer target database create is in progress")
	case string(datasafesdk.TargetDatabaseLifecycleStateUpdating):
		return shared.Updating, true, firstNonEmptyString(message, "OCI peer target database update is in progress")
	case string(datasafesdk.TargetDatabaseLifecycleStateDeleting):
		return shared.Terminating, true, firstNonEmptyString(message, "OCI peer target database delete is in progress")
	case string(datasafesdk.TargetDatabaseLifecycleStateFailed),
		string(datasafesdk.TargetDatabaseLifecycleStateNeedsAttention):
		return shared.Failed, false, firstNonEmptyString(message, "OCI peer target database needs attention")
	case string(datasafesdk.TargetDatabaseLifecycleStateDeleted):
		return shared.Terminating, false, firstNonEmptyString(message, "OCI peer target database is deleted")
	case string(datasafesdk.TargetDatabaseLifecycleStateActive),
		string(datasafesdk.TargetDatabaseLifecycleStateInactive):
		return shared.Active, false, firstNonEmptyString(message, "OCI peer target database is active")
	default:
		return fallback, fallback == shared.Provisioning || fallback == shared.Updating || fallback == shared.Terminating, firstNonEmptyString(message, "OCI peer target database state observed")
	}
}

func lifecyclePhase(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
	switch condition {
	case shared.Provisioning:
		return shared.OSOKAsyncPhaseCreate
	case shared.Updating:
		return shared.OSOKAsyncPhaseUpdate
	case shared.Terminating:
		return shared.OSOKAsyncPhaseDelete
	default:
		return ""
	}
}

func lifecycleRawState(current datasafesdk.PeerTargetDatabase) string {
	return strings.ToUpper(string(current.LifecycleState))
}

func peerTargetDatabaseAsyncClass(condition shared.OSOKConditionType) shared.OSOKAsyncNormalizedClass {
	switch condition {
	case shared.Failed:
		return shared.OSOKAsyncClassFailed
	case shared.Provisioning, shared.Updating, shared.Terminating:
		return shared.OSOKAsyncClassPending
	default:
		return shared.OSOKAsyncClassSucceeded
	}
}

func shouldTrackPeerTargetDatabaseAsync(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating || condition == shared.Failed
}

func isPeerTargetDatabaseDeleted(current datasafesdk.PeerTargetDatabase) bool {
	return strings.EqualFold(string(current.LifecycleState), string(datasafesdk.TargetDatabaseLifecycleStateDeleted))
}

func isPeerTargetDatabaseDeletePending(current datasafesdk.PeerTargetDatabase) bool {
	return strings.EqualFold(string(current.LifecycleState), string(datasafesdk.TargetDatabaseLifecycleStateDeleting))
}

func isPeerTargetDatabaseWritePending(current datasafesdk.PeerTargetDatabase) bool {
	switch strings.ToUpper(string(current.LifecycleState)) {
	case string(datasafesdk.TargetDatabaseLifecycleStateCreating),
		string(datasafesdk.TargetDatabaseLifecycleStateUpdating),
		string(datasafesdk.TargetDatabaseLifecycleStateDeleting):
		return true
	default:
		return false
	}
}

func currentPeerTargetDatabaseKey(resource *datasafev1beta1.PeerTargetDatabase, current datasafesdk.PeerTargetDatabase) int {
	if key := intValue(current.Key); key != 0 {
		return key
	}
	return resource.Status.Key
}

func listPeerTargetDatabasesAllPages(
	next func(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error),
) func(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
	if next == nil {
		return nil
	}
	return func(ctx context.Context, request datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
		collector := peerTargetDatabasePageCollector{next: next}
		return collector.collect(ctx, request)
	}
}

type peerTargetDatabasePageCollector struct {
	next func(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error)
}

func (c peerTargetDatabasePageCollector) collect(
	ctx context.Context,
	request datasafesdk.ListPeerTargetDatabasesRequest,
) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
	merged := datasafesdk.ListPeerTargetDatabasesResponse{}
	for {
		response, err := c.next(ctx, request)
		if err != nil {
			return datasafesdk.ListPeerTargetDatabasesResponse{}, err
		}
		mergePeerTargetDatabaseListPage(&merged, response)
		if !peerTargetDatabaseHasNextPage(response) {
			merged.OpcNextPage = response.OpcNextPage
			return merged, nil
		}
		request.Page = response.OpcNextPage
	}
}

func mergePeerTargetDatabaseListPage(
	merged *datasafesdk.ListPeerTargetDatabasesResponse,
	response datasafesdk.ListPeerTargetDatabasesResponse,
) {
	if merged.OpcRequestId == nil {
		merged.OpcRequestId = response.OpcRequestId
	}
	if merged.CompartmentId == nil {
		merged.CompartmentId = response.CompartmentId
	}
	if merged.TargetDatabaseId == nil {
		merged.TargetDatabaseId = response.TargetDatabaseId
	}
	merged.Items = append(merged.Items, response.Items...)
}

func peerTargetDatabaseHasNextPage(response datasafesdk.ListPeerTargetDatabasesResponse) bool {
	return response.OpcNextPage != nil && strings.TrimSpace(*response.OpcNextPage) != ""
}

func peerTargetDatabaseRetryToken(resource *datasafev1beta1.PeerTargetDatabase) string {
	if resource == nil {
		return ""
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return uid
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(resource.Namespace) + "/" + strings.TrimSpace(resource.Name)))
	return fmt.Sprintf("%x", sum[:16])
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339Nano)
}

func jsonEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	var leftValue any
	var rightValue any
	if json.Unmarshal(leftPayload, &leftValue) != nil || json.Unmarshal(rightPayload, &rightValue) != nil {
		return string(leftPayload) == string(rightPayload)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func stringPointer(value string) *string {
	if value = strings.TrimSpace(value); value != "" {
		return common.String(value)
	}
	return nil
}

func intPointer(value int) *int {
	if value == 0 {
		return nil
	}
	return common.Int(value)
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZeroInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func peerTargetDatabaseRequeue(requeue bool) time.Duration {
	if !requeue {
		return 0
	}
	return peerTargetDatabaseRequeueDuration
}
