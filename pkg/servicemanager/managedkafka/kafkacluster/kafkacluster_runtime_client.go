/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kafkacluster

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	managedkafkasdk "github.com/oracle/oci-go-sdk/v65/managedkafka"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

const kafkaClusterWorkRequestEntityType = "kafkaCluster"

var kafkaClusterWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(managedkafkasdk.OperationStatusAccepted),
		string(managedkafkasdk.OperationStatusInProgress),
		string(managedkafkasdk.OperationStatusWaiting),
		string(managedkafkasdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(managedkafkasdk.OperationStatusSucceeded)},
	FailedStatusTokens: []string{
		string(managedkafkasdk.OperationStatusFailed),
		string(managedkafkasdk.OperationStatusNeedsAttention),
	},
	CanceledStatusTokens: []string{string(managedkafkasdk.OperationStatusCanceled)},
	CreateActionTokens:   []string{string(managedkafkasdk.ActionTypeCreated)},
	UpdateActionTokens:   []string{string(managedkafkasdk.ActionTypeUpdated)},
	DeleteActionTokens:   []string{string(managedkafkasdk.ActionTypeDeleted)},
}

type kafkaClusterWorkRequestClient interface {
	GetWorkRequest(context.Context, managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error)
}

type kafkaClusterPendingWriteDeleteClient struct {
	delegate        KafkaClusterServiceClient
	getKafkaCluster func(context.Context, managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error)
}

type ambiguousKafkaClusterNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousKafkaClusterNotFoundError) Error() string {
	return e.message
}

func (e ambiguousKafkaClusterNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerKafkaClusterRuntimeHooksMutator(func(manager *KafkaClusterServiceManager, hooks *KafkaClusterRuntimeHooks) {
		workRequestClient, initErr := newKafkaClusterWorkRequestClient(manager)
		applyKafkaClusterRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newKafkaClusterWorkRequestClient(manager *KafkaClusterServiceManager) (kafkaClusterWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("KafkaCluster service manager is nil")
	}
	client, err := managedkafkasdk.NewKafkaClusterClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyKafkaClusterRuntimeHooks(
	hooks *KafkaClusterRuntimeHooks,
	workRequestClient kafkaClusterWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = kafkaClusterRuntimeSemantics()
	hooks.BuildCreateBody = buildKafkaClusterCreateBody
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *managedkafkav1beta1.KafkaCluster,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildKafkaClusterUpdateBody(resource, currentResponse)
	}
	getCall := hooks.Get.Call
	hooks.Get.Call = func(ctx context.Context, request managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		if getCall == nil {
			return managedkafkasdk.GetKafkaClusterResponse{}, fmt.Errorf("KafkaCluster GetKafkaCluster call is not configured")
		}
		response, err := getCall(ctx, request)
		return response, conservativeKafkaClusterNotFoundError(err, "get")
	}
	listCall := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request managedkafkasdk.ListKafkaClustersRequest) (managedkafkasdk.ListKafkaClustersResponse, error) {
		return listKafkaClustersAllPages(ctx, listCall, request)
	}
	deleteCall := hooks.Delete.Call
	hooks.Delete.Call = func(ctx context.Context, request managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error) {
		if deleteCall == nil {
			return managedkafkasdk.DeleteKafkaClusterResponse{}, fmt.Errorf("KafkaCluster DeleteKafkaCluster call is not configured")
		}
		response, err := deleteCall(ctx, request)
		return response, conservativeKafkaClusterNotFoundError(err, "delete")
	}
	hooks.Identity.GuardExistingBeforeCreate = guardKafkaClusterExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedKafkaClusterIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateKafkaClusterCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleKafkaClusterDeleteError
	hooks.Async.Adapter = kafkaClusterWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getKafkaClusterWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveKafkaClusterGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveKafkaClusterGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverKafkaClusterIDFromGeneratedWorkRequest
	hooks.Async.Message = kafkaClusterGeneratedWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate KafkaClusterServiceClient) KafkaClusterServiceClient {
		return kafkaClusterPendingWriteDeleteClient{delegate: delegate, getKafkaCluster: hooks.Get.Call}
	})
}

func kafkaClusterRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "managedkafka",
		FormalSlug:    "kafkacluster",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"accessSubnets",
				"brokerShape",
				"clientCertificateBundle",
				"clusterConfigId",
				"clusterConfigVersion",
				"coordinationType",
				"definedTags",
				"displayName",
				"freeformTags",
			},
			ForceNew:      []string{"clusterType", "compartmentId", "kafkaVersion"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: kafkaClusterWorkRequestEntityType, Action: "CREATED"},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: kafkaClusterWorkRequestEntityType, Action: "UPDATED"},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: kafkaClusterWorkRequestEntityType, Action: "DELETED"},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: kafkaClusterWorkRequestEntityType, Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: kafkaClusterWorkRequestEntityType, Action: "UPDATED"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: kafkaClusterWorkRequestEntityType, Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func guardKafkaClusterExistingBeforeCreate(
	_ context.Context,
	resource *managedkafkav1beta1.KafkaCluster,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("KafkaCluster resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func listKafkaClustersAllPages(
	ctx context.Context,
	listCall func(context.Context, managedkafkasdk.ListKafkaClustersRequest) (managedkafkasdk.ListKafkaClustersResponse, error),
	request managedkafkasdk.ListKafkaClustersRequest,
) (managedkafkasdk.ListKafkaClustersResponse, error) {
	if listCall == nil {
		return managedkafkasdk.ListKafkaClustersResponse{}, fmt.Errorf("KafkaCluster ListKafkaClusters call is not configured")
	}

	var combined managedkafkasdk.ListKafkaClustersResponse
	for {
		response, err := listCall(ctx, request)
		if err != nil {
			return managedkafkasdk.ListKafkaClustersResponse{}, conservativeKafkaClusterNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == managedkafkasdk.KafkaClusterLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleKafkaClusterDeleteError(resource *managedkafkav1beta1.KafkaCluster, err error) error {
	err = conservativeKafkaClusterNotFoundError(err, "delete")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeKafkaClusterNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}

	operation = strings.TrimSpace(operation)
	if operation == "" {
		operation = "operation"
	}
	message := fmt.Sprintf("KafkaCluster %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", operation, err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousKafkaClusterNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousKafkaClusterNotFoundError{message: message}
}

func buildKafkaClusterCreateBody(
	_ context.Context,
	resource *managedkafkav1beta1.KafkaCluster,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("KafkaCluster resource is nil")
	}

	spec := resource.Spec
	details := managedkafkasdk.CreateKafkaClusterDetails{
		CompartmentId:        common.String(spec.CompartmentId),
		AccessSubnets:        buildKafkaClusterAccessSubnets(spec.AccessSubnets),
		KafkaVersion:         common.String(spec.KafkaVersion),
		ClusterType:          managedkafkasdk.KafkaClusterClusterTypeEnum(spec.ClusterType),
		BrokerShape:          buildKafkaClusterBrokerShape(spec.BrokerShape),
		ClusterConfigId:      common.String(spec.ClusterConfigId),
		ClusterConfigVersion: common.Int(spec.ClusterConfigVersion),
		CoordinationType:     managedkafkasdk.KafkaClusterCoordinationTypeEnum(spec.CoordinationType),
	}
	if spec.DisplayName != "" {
		details.DisplayName = common.String(spec.DisplayName)
	}
	if spec.ClientCertificateBundle != "" {
		details.ClientCertificateBundle = common.String(spec.ClientCertificateBundle)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneKafkaClusterStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = kafkaClusterDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details, nil
}

func buildKafkaClusterUpdateBody(
	resource *managedkafkav1beta1.KafkaCluster,
	currentResponse any,
) (managedkafkasdk.UpdateKafkaClusterDetails, bool, error) {
	if resource == nil {
		return managedkafkasdk.UpdateKafkaClusterDetails{}, false, fmt.Errorf("KafkaCluster resource is nil")
	}
	current, ok := kafkaClusterFromResponse(currentResponse)
	if !ok {
		return managedkafkasdk.UpdateKafkaClusterDetails{}, false, fmt.Errorf("current KafkaCluster response does not expose a KafkaCluster body")
	}
	if err := validateKafkaClusterCreateOnlyDrift(resource.Spec, current); err != nil {
		return managedkafkasdk.UpdateKafkaClusterDetails{}, false, err
	}

	details, updateNeeded := buildKafkaClusterUpdateDetails(resource.Spec, current)
	if !updateNeeded {
		return managedkafkasdk.UpdateKafkaClusterDetails{}, false, nil
	}
	return details, true, nil
}

func buildKafkaClusterUpdateDetails(
	spec managedkafkav1beta1.KafkaClusterSpec,
	current managedkafkasdk.KafkaCluster,
) (managedkafkasdk.UpdateKafkaClusterDetails, bool) {
	details := managedkafkasdk.UpdateKafkaClusterDetails{}
	updateNeeded := false

	updateNeeded = setKafkaClusterStringUpdate(&details.DisplayName, spec.DisplayName, current.DisplayName) || updateNeeded
	updateNeeded = setKafkaClusterStringUpdate(&details.ClientCertificateBundle, spec.ClientCertificateBundle, current.ClientCertificateBundle) || updateNeeded
	updateNeeded = setKafkaClusterAccessSubnetsUpdate(&details, spec.AccessSubnets, current.AccessSubnets) || updateNeeded
	updateNeeded = setKafkaClusterBrokerShapeUpdate(&details, spec.BrokerShape, current.BrokerShape) || updateNeeded
	updateNeeded = setKafkaClusterStringUpdate(&details.ClusterConfigId, spec.ClusterConfigId, current.ClusterConfigId) || updateNeeded
	updateNeeded = setKafkaClusterIntUpdate(&details.ClusterConfigVersion, spec.ClusterConfigVersion, current.ClusterConfigVersion) || updateNeeded
	updateNeeded = setKafkaClusterCoordinationTypeUpdate(&details, spec.CoordinationType, current.CoordinationType) || updateNeeded
	updateNeeded = setKafkaClusterFreeformTagsUpdate(&details, spec.FreeformTags, current.FreeformTags) || updateNeeded
	updateNeeded = setKafkaClusterDefinedTagsUpdate(&details, spec.DefinedTags, current.DefinedTags) || updateNeeded

	return details, updateNeeded
}

func setKafkaClusterStringUpdate(target **string, spec string, current *string) bool {
	desired, ok := kafkaClusterDesiredStringUpdate(spec, current)
	if !ok {
		return false
	}
	*target = desired
	return true
}

func setKafkaClusterAccessSubnetsUpdate(
	details *managedkafkasdk.UpdateKafkaClusterDetails,
	spec []managedkafkav1beta1.KafkaClusterAccessSubnet,
	current []managedkafkasdk.SubnetSet,
) bool {
	if kafkaClusterAccessSubnetsEqual(current, spec) {
		return false
	}
	details.AccessSubnets = buildKafkaClusterAccessSubnets(spec)
	return true
}

func setKafkaClusterBrokerShapeUpdate(
	details *managedkafkasdk.UpdateKafkaClusterDetails,
	spec managedkafkav1beta1.KafkaClusterBrokerShape,
	current *managedkafkasdk.BrokerShape,
) bool {
	if !kafkaClusterBrokerShapeNeedsUpdate(spec, current) {
		return false
	}
	details.BrokerShape = buildKafkaClusterBrokerShape(spec)
	return true
}

func setKafkaClusterIntUpdate(target **int, spec int, current *int) bool {
	if kafkaClusterIntPtrEqual(current, spec) {
		return false
	}
	*target = common.Int(spec)
	return true
}

func setKafkaClusterCoordinationTypeUpdate(
	details *managedkafkasdk.UpdateKafkaClusterDetails,
	spec string,
	current managedkafkasdk.KafkaClusterCoordinationTypeEnum,
) bool {
	if spec == "" || spec == string(current) {
		return false
	}
	details.CoordinationType = managedkafkasdk.KafkaClusterCoordinationTypeEnum(spec)
	return true
}

func setKafkaClusterFreeformTagsUpdate(
	details *managedkafkasdk.UpdateKafkaClusterDetails,
	spec map[string]string,
	current map[string]string,
) bool {
	if spec == nil || reflect.DeepEqual(current, spec) {
		return false
	}
	details.FreeformTags = cloneKafkaClusterStringMap(spec)
	return true
}

func setKafkaClusterDefinedTagsUpdate(
	details *managedkafkasdk.UpdateKafkaClusterDetails,
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) bool {
	if spec == nil {
		return false
	}
	desired := kafkaClusterDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(current, desired) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func validateKafkaClusterCreateOnlyDriftForResponse(
	resource *managedkafkav1beta1.KafkaCluster,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("KafkaCluster resource is nil")
	}
	current, ok := kafkaClusterFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current KafkaCluster response does not expose a KafkaCluster body")
	}
	return validateKafkaClusterCreateOnlyDrift(resource.Spec, current)
}

func validateKafkaClusterCreateOnlyDrift(
	spec managedkafkav1beta1.KafkaClusterSpec,
	current managedkafkasdk.KafkaCluster,
) error {
	var unsupported []string
	if !kafkaClusterStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !kafkaClusterStringPtrEqual(current.KafkaVersion, spec.KafkaVersion) {
		unsupported = append(unsupported, "kafkaVersion")
	}
	if spec.ClusterType != "" && spec.ClusterType != string(current.ClusterType) {
		unsupported = append(unsupported, "clusterType")
	}
	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("KafkaCluster create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func clearTrackedKafkaClusterIdentity(resource *managedkafkav1beta1.KafkaCluster) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func (c kafkaClusterPendingWriteDeleteClient) CreateOrUpdate(
	ctx context.Context,
	resource *managedkafkav1beta1.KafkaCluster,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c kafkaClusterPendingWriteDeleteClient) Delete(
	ctx context.Context,
	resource *managedkafkav1beta1.KafkaCluster,
) (bool, error) {
	if kafkaClusterHasPendingLifecycleWrite(resource) {
		ready, err := c.kafkaClusterPendingLifecycleWriteReadyForDelete(ctx, resource)
		if err != nil {
			return false, err
		}
		if !ready {
			return false, nil
		}
	} else if kafkaClusterHasPendingWrite(resource) {
		response, err := c.delegate.CreateOrUpdate(ctx, resource, ctrl.Request{})
		if err != nil {
			return false, err
		}
		if kafkaClusterHasPendingWrite(resource) || response.ShouldRequeue {
			return false, nil
		}
	}
	return c.delegate.Delete(ctx, resource)
}

func (c kafkaClusterPendingWriteDeleteClient) kafkaClusterPendingLifecycleWriteReadyForDelete(
	ctx context.Context,
	resource *managedkafkav1beta1.KafkaCluster,
) (bool, error) {
	if c.getKafkaCluster == nil {
		return true, nil
	}
	currentID := kafkaClusterCurrentID(resource)
	if currentID == "" {
		return true, nil
	}

	response, err := c.getKafkaCluster(ctx, managedkafkasdk.GetKafkaClusterRequest{
		KafkaClusterId: common.String(currentID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return true, nil
		}
		return false, err
	}
	if !kafkaClusterLifecycleStateIsPendingWrite(response.LifecycleState) {
		return true, nil
	}

	projectKafkaClusterPendingWriteLifecycle(resource, response.KafkaCluster)
	return false, nil
}

func kafkaClusterHasPendingWrite(resource *managedkafkav1beta1.KafkaCluster) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	return kafkaClusterAsyncIsPendingWrite(resource.Status.OsokStatus.Async.Current)
}

func kafkaClusterHasPendingLifecycleWrite(resource *managedkafkav1beta1.KafkaCluster) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceLifecycle && kafkaClusterAsyncIsPendingWrite(current)
}

func kafkaClusterAsyncIsPendingWrite(current *shared.OSOKAsyncOperation) bool {
	if current == nil || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest && current.Source != shared.OSOKAsyncSourceLifecycle {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func kafkaClusterCurrentID(resource *managedkafkav1beta1.KafkaCluster) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func kafkaClusterLifecycleStateIsPendingWrite(state managedkafkasdk.KafkaClusterLifecycleStateEnum) bool {
	switch state {
	case managedkafkasdk.KafkaClusterLifecycleStateCreating,
		managedkafkasdk.KafkaClusterLifecycleStateUpdating:
		return true
	default:
		return false
	}
}

func projectKafkaClusterPendingWriteLifecycle(
	resource *managedkafkav1beta1.KafkaCluster,
	current managedkafkasdk.KafkaCluster,
) {
	if resource == nil {
		return
	}
	if id := strings.TrimSpace(kafkaClusterStringValue(current.Id)); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	resource.Status.LifecycleState = string(current.LifecycleState)

	phase := kafkaClusterPendingWritePhase(resource, current.LifecycleState)
	if phase == "" {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       string(current.LifecycleState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         kafkaClusterLifecycleMessage(current),
	}, servicemanager.RuntimeDeps{}.Log)
}

func kafkaClusterPendingWritePhase(
	resource *managedkafkav1beta1.KafkaCluster,
	state managedkafkasdk.KafkaClusterLifecycleStateEnum,
) shared.OSOKAsyncPhase {
	if resource != nil && resource.Status.OsokStatus.Async.Current != nil {
		phase := resource.Status.OsokStatus.Async.Current.Phase
		if phase == shared.OSOKAsyncPhaseCreate || phase == shared.OSOKAsyncPhaseUpdate {
			return phase
		}
	}
	switch state {
	case managedkafkasdk.KafkaClusterLifecycleStateCreating:
		return shared.OSOKAsyncPhaseCreate
	case managedkafkasdk.KafkaClusterLifecycleStateUpdating:
		return shared.OSOKAsyncPhaseUpdate
	default:
		return ""
	}
}

func kafkaClusterLifecycleMessage(current managedkafkasdk.KafkaCluster) string {
	if message := strings.TrimSpace(kafkaClusterStringValue(current.LifecycleDetails)); message != "" {
		return message
	}
	state := strings.TrimSpace(string(current.LifecycleState))
	if state == "" {
		return ""
	}
	if id := strings.TrimSpace(kafkaClusterStringValue(current.Id)); id != "" {
		return fmt.Sprintf("KafkaCluster %s is %s", id, state)
	}
	return fmt.Sprintf("KafkaCluster is %s", state)
}

func getKafkaClusterWorkRequest(
	ctx context.Context,
	client kafkaClusterWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize KafkaCluster OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("KafkaCluster OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, managedkafkasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveKafkaClusterGeneratedWorkRequestAction(workRequest any) (string, error) {
	kafkaClusterWorkRequest, err := kafkaClusterWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveKafkaClusterWorkRequestAction(kafkaClusterWorkRequest)
}

func resolveKafkaClusterGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	kafkaClusterWorkRequest, err := kafkaClusterWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := kafkaClusterWorkRequestPhaseFromOperationType(kafkaClusterWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverKafkaClusterIDFromGeneratedWorkRequest(
	_ *managedkafkav1beta1.KafkaCluster,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	kafkaClusterWorkRequest, err := kafkaClusterWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveKafkaClusterIDFromWorkRequest(kafkaClusterWorkRequest, kafkaClusterWorkRequestActionForPhase(phase))
}

func kafkaClusterGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	kafkaClusterWorkRequest, err := kafkaClusterWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("KafkaCluster %s work request %s is %s", phase, kafkaClusterStringValue(kafkaClusterWorkRequest.Id), kafkaClusterWorkRequest.Status)
}

func kafkaClusterWorkRequestFromAny(workRequest any) (managedkafkasdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case managedkafkasdk.WorkRequest:
		return current, nil
	case *managedkafkasdk.WorkRequest:
		if current == nil {
			return managedkafkasdk.WorkRequest{}, fmt.Errorf("KafkaCluster work request is nil")
		}
		return *current, nil
	default:
		return managedkafkasdk.WorkRequest{}, fmt.Errorf("unexpected KafkaCluster work request type %T", workRequest)
	}
}

func kafkaClusterWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) managedkafkasdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return managedkafkasdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return managedkafkasdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return managedkafkasdk.ActionTypeDeleted
	default:
		return ""
	}
}

func kafkaClusterWorkRequestPhaseFromOperationType(operationType managedkafkasdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case managedkafkasdk.OperationTypeCreateKafkaCluster:
		return shared.OSOKAsyncPhaseCreate, true
	case managedkafkasdk.OperationTypeUpdateKafkaCluster:
		return shared.OSOKAsyncPhaseUpdate, true
	case managedkafkasdk.OperationTypeDeleteKafkaCluster:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func resolveKafkaClusterIDFromWorkRequest(
	workRequest managedkafkasdk.WorkRequest,
	action managedkafkasdk.ActionTypeEnum,
) (string, error) {
	if id, ok := kafkaClusterIDFromWorkRequestResources(workRequest.Resources, action); ok {
		return id, nil
	}
	if id, ok := kafkaClusterIDFromWorkRequestResources(workRequest.Resources, ""); ok {
		return id, nil
	}

	return "", fmt.Errorf("KafkaCluster work request %s does not expose a KafkaCluster identifier", kafkaClusterStringValue(workRequest.Id))
}

func kafkaClusterIDFromWorkRequestResources(
	resources []managedkafkasdk.WorkRequestResource,
	action managedkafkasdk.ActionTypeEnum,
) (string, bool) {
	for _, resource := range resources {
		id, ok := kafkaClusterIDFromWorkRequestResource(resource, action)
		if ok {
			return id, true
		}
	}
	return "", false
}

func kafkaClusterIDFromWorkRequestResource(
	resource managedkafkasdk.WorkRequestResource,
	action managedkafkasdk.ActionTypeEnum,
) (string, bool) {
	if !isKafkaClusterWorkRequestResource(resource) {
		return "", false
	}
	if isKafkaClusterIgnorableWorkRequestAction(resource.ActionType) {
		return "", false
	}
	if action != "" && resource.ActionType != action {
		return "", false
	}
	id := strings.TrimSpace(kafkaClusterStringValue(resource.Identifier))
	return id, id != ""
}

func resolveKafkaClusterWorkRequestAction(workRequest managedkafkasdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isKafkaClusterWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || isKafkaClusterIgnorableWorkRequestAction(resource.ActionType) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf(
				"KafkaCluster work request %s exposes conflicting KafkaCluster action types %q and %q",
				kafkaClusterStringValue(workRequest.Id),
				action,
				candidate,
			)
		}
	}
	return action, nil
}

func isKafkaClusterIgnorableWorkRequestAction(action managedkafkasdk.ActionTypeEnum) bool {
	return action == managedkafkasdk.ActionTypeInProgress || action == managedkafkasdk.ActionTypeRelated
}

func isKafkaClusterWorkRequestResource(resource managedkafkasdk.WorkRequestResource) bool {
	return normalizeKafkaClusterWorkRequestEntity(kafkaClusterStringValue(resource.EntityType)) == "kafkacluster"
}

func normalizeKafkaClusterWorkRequestEntity(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func kafkaClusterFromResponse(response any) (managedkafkasdk.KafkaCluster, bool) {
	switch current := response.(type) {
	case managedkafkasdk.KafkaCluster:
		return current, true
	case *managedkafkasdk.KafkaCluster:
		if current == nil {
			return managedkafkasdk.KafkaCluster{}, false
		}
		return *current, true
	case managedkafkasdk.KafkaClusterSummary:
		return kafkaClusterFromSummary(current), true
	case *managedkafkasdk.KafkaClusterSummary:
		if current == nil {
			return managedkafkasdk.KafkaCluster{}, false
		}
		return kafkaClusterFromSummary(*current), true
	default:
		return kafkaClusterFromResponseField(response)
	}
}

func kafkaClusterFromResponseField(response any) (managedkafkasdk.KafkaCluster, bool) {
	value := reflect.ValueOf(response)
	if !value.IsValid() {
		return managedkafkasdk.KafkaCluster{}, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return managedkafkasdk.KafkaCluster{}, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return managedkafkasdk.KafkaCluster{}, false
	}

	field := value.FieldByName("KafkaCluster")
	if !field.IsValid() || !field.CanInterface() {
		return managedkafkasdk.KafkaCluster{}, false
	}
	cluster, ok := field.Interface().(managedkafkasdk.KafkaCluster)
	return cluster, ok
}

func kafkaClusterFromSummary(summary managedkafkasdk.KafkaClusterSummary) managedkafkasdk.KafkaCluster {
	return managedkafkasdk.KafkaCluster{
		Id:                   summary.Id,
		DisplayName:          summary.DisplayName,
		CompartmentId:        summary.CompartmentId,
		TimeCreated:          summary.TimeCreated,
		LifecycleState:       summary.LifecycleState,
		AccessSubnets:        cloneKafkaClusterSDKAccessSubnets(summary.AccessSubnets),
		KafkaVersion:         summary.KafkaVersion,
		ClusterType:          summary.ClusterType,
		BrokerShape:          cloneKafkaClusterSDKBrokerShape(summary.BrokerShape),
		ClusterConfigId:      summary.ClusterConfigId,
		ClusterConfigVersion: summary.ClusterConfigVersion,
		FreeformTags:         cloneKafkaClusterStringMap(summary.FreeformTags),
		DefinedTags:          cloneKafkaClusterDefinedTags(summary.DefinedTags),
		TimeUpdated:          summary.TimeUpdated,
		LifecycleDetails:     summary.LifecycleDetails,
		CoordinationType:     summary.CoordinationType,
		SystemTags:           cloneKafkaClusterDefinedTags(summary.SystemTags),
	}
}

func buildKafkaClusterAccessSubnets(source []managedkafkav1beta1.KafkaClusterAccessSubnet) []managedkafkasdk.SubnetSet {
	if source == nil {
		return nil
	}
	subnets := make([]managedkafkasdk.SubnetSet, 0, len(source))
	for _, subnet := range source {
		subnets = append(subnets, managedkafkasdk.SubnetSet{
			Subnets: append([]string(nil), subnet.Subnets...),
		})
	}
	return subnets
}

func buildKafkaClusterBrokerShape(source managedkafkav1beta1.KafkaClusterBrokerShape) *managedkafkasdk.BrokerShape {
	shape := &managedkafkasdk.BrokerShape{
		NodeCount: common.Int(source.NodeCount),
		OcpuCount: common.Int(source.OcpuCount),
	}
	if source.StorageSizeInGbs != 0 {
		shape.StorageSizeInGbs = common.Int(source.StorageSizeInGbs)
	}
	if source.NodeShape != "" {
		shape.NodeShape = common.String(source.NodeShape)
	}
	return shape
}

func kafkaClusterBrokerShapeNeedsUpdate(
	spec managedkafkav1beta1.KafkaClusterBrokerShape,
	current *managedkafkasdk.BrokerShape,
) bool {
	if current == nil {
		return true
	}
	if !kafkaClusterIntPtrEqual(current.NodeCount, spec.NodeCount) {
		return true
	}
	if !kafkaClusterIntPtrEqual(current.OcpuCount, spec.OcpuCount) {
		return true
	}
	if spec.StorageSizeInGbs != 0 && !kafkaClusterIntPtrEqual(current.StorageSizeInGbs, spec.StorageSizeInGbs) {
		return true
	}
	return spec.NodeShape != "" && !kafkaClusterStringPtrEqual(current.NodeShape, spec.NodeShape)
}

func kafkaClusterAccessSubnetsEqual(
	current []managedkafkasdk.SubnetSet,
	spec []managedkafkav1beta1.KafkaClusterAccessSubnet,
) bool {
	if len(current) != len(spec) {
		return false
	}
	for i := range current {
		if !reflect.DeepEqual(current[i].Subnets, spec[i].Subnets) {
			return false
		}
	}
	return true
}

func kafkaClusterDesiredStringUpdate(spec string, current *string) (*string, bool) {
	if spec == "" {
		return nil, false
	}
	if kafkaClusterStringPtrEqual(current, spec) {
		return nil, false
	}
	return common.String(spec), true
}

func kafkaClusterStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(kafkaClusterStringValue(current)) == strings.TrimSpace(desired)
}

func kafkaClusterIntPtrEqual(current *int, desired int) bool {
	return current != nil && *current == desired
}

func kafkaClusterStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneKafkaClusterStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func kafkaClusterDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}

func cloneKafkaClusterDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		clone[namespace] = inner
	}
	return clone
}

func cloneKafkaClusterSDKAccessSubnets(source []managedkafkasdk.SubnetSet) []managedkafkasdk.SubnetSet {
	if source == nil {
		return nil
	}
	clone := make([]managedkafkasdk.SubnetSet, 0, len(source))
	for _, subnet := range source {
		clone = append(clone, managedkafkasdk.SubnetSet{Subnets: append([]string(nil), subnet.Subnets...)})
	}
	return clone
}

func cloneKafkaClusterSDKBrokerShape(source *managedkafkasdk.BrokerShape) *managedkafkasdk.BrokerShape {
	if source == nil {
		return nil
	}
	return &managedkafkasdk.BrokerShape{
		NodeCount:        source.NodeCount,
		OcpuCount:        source.OcpuCount,
		StorageSizeInGbs: source.StorageSizeInGbs,
		NodeShape:        source.NodeShape,
	}
}
