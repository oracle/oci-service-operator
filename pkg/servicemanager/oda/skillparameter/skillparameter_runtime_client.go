/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package skillparameter

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// The OCI API requires parent path identifiers that the generated SkillParameter
	// CRD does not expose, so this resource-local wrapper reads them from annotations
	// and tracks the resolved path identity synthetically in shared status.
	skillParameterOdaInstanceIDAnnotation = "oda.oracle.com/oda-instance-id"
	skillParameterSkillIDAnnotation       = "oda.oracle.com/skill-id"

	skillParameterLegacyOdaInstanceIDAnnotation = "oda.oracle.com/odaInstanceId"
	skillParameterLegacySkillIDAnnotation       = "oda.oracle.com/skillId"

	skillParameterSyntheticIDPrefix  = "skillparameter/"
	skillParameterSyntheticIDVersion = "v1/"
	skillParameterRequeueDuration    = time.Minute
)

type skillParameterOCIClient interface {
	CreateSkillParameter(context.Context, odasdk.CreateSkillParameterRequest) (odasdk.CreateSkillParameterResponse, error)
	GetSkillParameter(context.Context, odasdk.GetSkillParameterRequest) (odasdk.GetSkillParameterResponse, error)
	ListSkillParameters(context.Context, odasdk.ListSkillParametersRequest) (odasdk.ListSkillParametersResponse, error)
	UpdateSkillParameter(context.Context, odasdk.UpdateSkillParameterRequest) (odasdk.UpdateSkillParameterResponse, error)
	DeleteSkillParameter(context.Context, odasdk.DeleteSkillParameterRequest) (odasdk.DeleteSkillParameterResponse, error)
}

type skillParameterIdentity struct {
	odaInstanceID string
	skillID       string
	name          string
}

type skillParameterSnapshot struct {
	name           string
	displayName    string
	parameterType  string
	value          string
	lifecycleState string
	description    string
}

type skillParameterRuntimeClient struct {
	delegate SkillParameterServiceClient
	client   skillParameterOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

var _ SkillParameterServiceClient = (*skillParameterRuntimeClient)(nil)

func init() {
	registerSkillParameterRuntimeHooksMutator(func(manager *SkillParameterServiceManager, hooks *SkillParameterRuntimeHooks) {
		client, err := newSkillParameterSDKClient(manager)
		applySkillParameterRuntimeHooks(manager, hooks, client, err)
	})
}

func newSkillParameterSDKClient(manager *SkillParameterServiceManager) (skillParameterOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("SkillParameter service manager is nil")
	}

	client, err := odasdk.NewManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySkillParameterRuntimeHooks(
	manager *SkillParameterServiceManager,
	hooks *SkillParameterRuntimeHooks,
	client skillParameterOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newSkillParameterRuntimeSemantics()
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SkillParameterServiceClient) SkillParameterServiceClient {
		return newSkillParameterRuntimeClient(manager, delegate, client, initErr)
	})
}

func newSkillParameterRuntimeClient(
	manager *SkillParameterServiceManager,
	delegate SkillParameterServiceClient,
	client skillParameterOCIClient,
	initErr error,
) *skillParameterRuntimeClient {
	runtimeClient := &skillParameterRuntimeClient{
		delegate: delegate,
		client:   client,
		initErr:  initErr,
	}
	if manager != nil {
		runtimeClient.log = manager.Log
	}
	return runtimeClient
}

func newSkillParameterServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client skillParameterOCIClient,
) SkillParameterServiceClient {
	manager := &SkillParameterServiceManager{Log: log}
	hooks := SkillParameterRuntimeHooks{}
	applySkillParameterRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultSkillParameterServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.SkillParameter](
			buildSkillParameterGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSkillParameterGeneratedClient(hooks, delegate)
}

func newSkillParameterRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "skillparameter",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE", "INACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name", "lifecycleState"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"description", "displayName", "value"},
			Mutable:         []string{"description", "displayName", "value"},
			ForceNew:        []string{"name", "type"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func (c *skillParameterRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *odav1beta1.SkillParameter,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.ensureClient(); err != nil {
		return c.fail(resource, err)
	}

	identity, err := resolveDesiredSkillParameterIdentity(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if err := validateTrackedSkillParameterIdentity(resource, identity); err != nil {
		return c.fail(resource, err)
	}
	if err := validateDesiredSkillParameterType(resource); err != nil {
		return c.fail(resource, err)
	}

	current, err := c.readSkillParameter(ctx, identity)
	if err != nil {
		return c.fail(resource, fmt.Errorf("read SkillParameter %q: %w", identity.name, err))
	}
	if current == nil {
		return c.createSkillParameter(ctx, resource, identity)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	currentState := normalizeSkillParameterLifecycle(current.lifecycleState)
	if currentState != "" && !isSkillParameterSteadyLifecycle(currentState) {
		return c.finishWithLifecycle(resource, identity, current, shared.OSOKAsyncPhaseUpdate), nil
	}

	if err := validateSkillParameterCreateOnlyDrift(resource, current); err != nil {
		return c.fail(resource, err)
	}
	updateDetails, updateNeeded := buildSkillParameterUpdateDetails(resource, current)
	if !updateNeeded {
		return c.finishWithLifecycle(resource, identity, current, shared.OSOKAsyncPhaseUpdate), nil
	}

	response, err := c.client.UpdateSkillParameter(ctx, odasdk.UpdateSkillParameterRequest{
		OdaInstanceId:               stringPtr(identity.odaInstanceID),
		SkillId:                     stringPtr(identity.skillID),
		ParameterName:               stringPtr(identity.name),
		UpdateSkillParameterDetails: updateDetails,
	})
	if err != nil {
		return c.fail(resource, fmt.Errorf("update SkillParameter %q: %w", identity.name, err))
	}
	c.recordResponseRequestID(resource, response)

	refreshed, err := c.getSkillParameter(ctx, identity)
	if err != nil {
		if skillParameterIsNotFound(err) {
			return c.markPendingLifecycleOperation(resource, identity, shared.OSOKAsyncPhaseUpdate, "OCI SkillParameter update request accepted", ""), nil
		}
		return c.fail(resource, fmt.Errorf("confirm SkillParameter %q update: %w", identity.name, err))
	}
	return c.finishWithLifecycle(resource, identity, refreshed, shared.OSOKAsyncPhaseUpdate), nil
}

func (c *skillParameterRuntimeClient) Delete(ctx context.Context, resource *odav1beta1.SkillParameter) (bool, error) {
	if err := c.ensureClient(); err != nil {
		return false, err
	}

	identity, shouldDeleteOCI, err := resolveDeleteSkillParameterIdentity(resource)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	if !shouldDeleteOCI {
		// Without a persisted tracked identity or parent annotations, this CR never
		// recorded an OCI target that can be safely deleted.
		c.markDeleted(resource, "No tracked OCI SkillParameter identity recorded; skipping OCI delete")
		return true, nil
	}

	current, err := c.getSkillParameter(ctx, identity)
	if err != nil {
		if skillParameterIsNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI SkillParameter deleted")
			return true, nil
		}
		return false, fmt.Errorf("read SkillParameter %q before delete: %w", identity.name, err)
	}
	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}

	switch normalizeSkillParameterLifecycle(current.lifecycleState) {
	case string(odasdk.LifecycleStateDeleted):
		c.markDeleted(resource, "OCI SkillParameter deleted")
		return true, nil
	case string(odasdk.LifecycleStateDeleting):
		return false, c.markTerminating(resource, identity, "OCI SkillParameter delete is in progress", current.lifecycleState)
	}

	response, err := c.client.DeleteSkillParameter(ctx, odasdk.DeleteSkillParameterRequest{
		OdaInstanceId: stringPtr(identity.odaInstanceID),
		SkillId:       stringPtr(identity.skillID),
		ParameterName: stringPtr(identity.name),
	})
	if err != nil {
		if skillParameterIsNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI SkillParameter deleted")
			return true, nil
		}
		return false, fmt.Errorf("delete SkillParameter %q: %w", identity.name, err)
	}
	c.recordResponseRequestID(resource, response)

	refreshed, err := c.getSkillParameter(ctx, identity)
	if err != nil {
		if skillParameterIsNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI SkillParameter deleted")
			return true, nil
		}
		return false, fmt.Errorf("confirm SkillParameter %q delete: %w", identity.name, err)
	}
	if err := c.projectStatus(resource, refreshed); err != nil {
		return false, err
	}
	if normalizeSkillParameterLifecycle(refreshed.lifecycleState) == string(odasdk.LifecycleStateDeleted) {
		c.markDeleted(resource, "OCI SkillParameter deleted")
		return true, nil
	}

	return false, c.markTerminating(resource, identity, "OCI SkillParameter delete is in progress", refreshed.lifecycleState)
}

func (c *skillParameterRuntimeClient) createSkillParameter(
	ctx context.Context,
	resource *odav1beta1.SkillParameter,
	identity skillParameterIdentity,
) (servicemanager.OSOKResponse, error) {
	response, err := c.client.CreateSkillParameter(ctx, odasdk.CreateSkillParameterRequest{
		OdaInstanceId: stringPtr(identity.odaInstanceID),
		SkillId:       stringPtr(identity.skillID),
		CreateSkillParameterDetails: odasdk.CreateSkillParameterDetails{
			Name:        stringPtr(identity.name),
			DisplayName: stringPtr(resource.Spec.DisplayName),
			Type:        odasdk.ParameterTypeEnum(normalizeSkillParameterType(resource.Spec.Type)),
			Value:       stringPtr(resource.Spec.Value),
			Description: stringPtr(resource.Spec.Description),
		},
	})
	if err != nil {
		return c.fail(resource, fmt.Errorf("create SkillParameter %q: %w", identity.name, err))
	}
	c.recordResponseRequestID(resource, response)

	refreshed, err := c.getSkillParameter(ctx, identity)
	if err != nil {
		if skillParameterIsNotFound(err) {
			snapshot := skillParameterSnapshotFromSDK(response.SkillParameter)
			return c.finishWithLifecycle(resource, identity, snapshot, shared.OSOKAsyncPhaseCreate), nil
		}
		return c.fail(resource, fmt.Errorf("confirm SkillParameter %q create: %w", identity.name, err))
	}
	return c.finishWithLifecycle(resource, identity, refreshed, shared.OSOKAsyncPhaseCreate), nil
}

func (c *skillParameterRuntimeClient) readSkillParameter(
	ctx context.Context,
	identity skillParameterIdentity,
) (*skillParameterSnapshot, error) {
	response, err := c.client.ListSkillParameters(ctx, odasdk.ListSkillParametersRequest{
		OdaInstanceId: stringPtr(identity.odaInstanceID),
		SkillId:       stringPtr(identity.skillID),
		Name:          stringPtr(identity.name),
	})
	if err != nil {
		if skillParameterIsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var matched *skillParameterSnapshot
	for _, item := range response.Items {
		snapshot := skillParameterSnapshotFromSummary(item)
		if snapshot.name != identity.name {
			continue
		}
		if normalizeSkillParameterLifecycle(snapshot.lifecycleState) == string(odasdk.LifecycleStateDeleted) {
			continue
		}
		if matched != nil {
			return nil, fmt.Errorf("list SkillParameter %q returned multiple matches", identity.name)
		}
		copy := snapshot
		matched = &copy
	}

	if matched == nil {
		return c.getSkillParameterIfExists(ctx, identity)
	}

	refreshed, err := c.getSkillParameter(ctx, identity)
	if err != nil {
		if skillParameterIsNotFound(err) {
			return matched, nil
		}
		return nil, err
	}
	return refreshed, nil
}

func (c *skillParameterRuntimeClient) getSkillParameterIfExists(
	ctx context.Context,
	identity skillParameterIdentity,
) (*skillParameterSnapshot, error) {
	snapshot, err := c.getSkillParameter(ctx, identity)
	if err != nil {
		if skillParameterIsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if normalizeSkillParameterLifecycle(snapshot.lifecycleState) == string(odasdk.LifecycleStateDeleted) {
		return nil, nil
	}
	return snapshot, nil
}

func (c *skillParameterRuntimeClient) getSkillParameter(
	ctx context.Context,
	identity skillParameterIdentity,
) (*skillParameterSnapshot, error) {
	response, err := c.client.GetSkillParameter(ctx, odasdk.GetSkillParameterRequest{
		OdaInstanceId: stringPtr(identity.odaInstanceID),
		SkillId:       stringPtr(identity.skillID),
		ParameterName: stringPtr(identity.name),
	})
	if err != nil {
		return nil, err
	}
	return skillParameterSnapshotFromSDK(response.SkillParameter), nil
}

func (c *skillParameterRuntimeClient) finishWithLifecycle(
	resource *odav1beta1.SkillParameter,
	identity skillParameterIdentity,
	snapshot *skillParameterSnapshot,
	fallbackPhase shared.OSOKAsyncPhase,
) servicemanager.OSOKResponse {
	if snapshot == nil {
		return c.markCondition(resource, identity, shared.Active, "OCI SkillParameter is active", false)
	}
	if err := c.projectStatus(resource, snapshot); err != nil {
		response, _ := c.fail(resource, err)
		return response
	}

	state := normalizeSkillParameterLifecycle(snapshot.lifecycleState)
	message := skillParameterLifecycleMessage(snapshot)
	switch state {
	case string(odasdk.LifecycleStateActive), string(odasdk.LifecycleStateInactive):
		return c.markCondition(resource, identity, shared.Active, message, false)
	case string(odasdk.LifecycleStateCreating):
		return c.markPendingLifecycleOperation(resource, identity, shared.OSOKAsyncPhaseCreate, message, state)
	case string(odasdk.LifecycleStateUpdating):
		return c.markPendingLifecycleOperation(resource, identity, shared.OSOKAsyncPhaseUpdate, message, state)
	case string(odasdk.LifecycleStateDeleting):
		return c.markPendingLifecycleOperation(resource, identity, shared.OSOKAsyncPhaseDelete, message, state)
	case string(odasdk.LifecycleStateDeleted):
		return c.markAsyncOperation(resource, identity, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassSucceeded,
			Message:         message,
		})
	case string(odasdk.LifecycleStateFailed):
		return c.markAsyncOperation(resource, identity, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           fallbackPhase,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
		})
	default:
		return c.markCondition(
			resource,
			identity,
			shared.Failed,
			fmt.Sprintf("SkillParameter lifecycle state %q is not modeled", snapshot.lifecycleState),
			false,
		)
	}
}

func (c *skillParameterRuntimeClient) markCondition(
	resource *odav1beta1.SkillParameter,
	identity skillParameterIdentity,
	condition shared.OSOKConditionType,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: condition != shared.Failed}
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Ocid = identity.syntheticOCID()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	}

	conditionStatus := corev1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = corev1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: skillParameterRequeueDuration,
	}
}

func (c *skillParameterRuntimeClient) markAsyncOperation(
	resource *odav1beta1.SkillParameter,
	identity skillParameterIdentity,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Ocid = identity.syntheticOCID()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: skillParameterRequeueDuration,
	}
}

func (c *skillParameterRuntimeClient) markPendingLifecycleOperation(
	resource *odav1beta1.SkillParameter,
	identity skillParameterIdentity,
	phase shared.OSOKAsyncPhase,
	message string,
	rawStatus string,
) servicemanager.OSOKResponse {
	return c.markAsyncOperation(resource, identity, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           phase,
		RawStatus:       normalizeSkillParameterLifecycle(rawStatus),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *skillParameterRuntimeClient) markTerminating(
	resource *odav1beta1.SkillParameter,
	identity skillParameterIdentity,
	message string,
	rawStatus string,
) error {
	c.markPendingLifecycleOperation(resource, identity, shared.OSOKAsyncPhaseDelete, message, rawStatus)
	return nil
}

func (c *skillParameterRuntimeClient) markDeleted(resource *odav1beta1.SkillParameter, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, c.log)
}

func (c *skillParameterRuntimeClient) fail(
	resource *odav1beta1.SkillParameter,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil && err != nil {
		status := &resource.Status.OsokStatus
		servicemanager.RecordErrorOpcRequestID(status, err)
		now := metav1.Now()
		status.UpdatedAt = &now
		status.Message = err.Error()
		status.Reason = string(shared.Failed)
		if status.Async.Current != nil {
			current := *status.Async.Current
			current.NormalizedClass = shared.OSOKAsyncClassFailed
			current.Message = err.Error()
			current.UpdatedAt = &now
			_ = servicemanager.ApplyAsyncOperation(status, &current, c.log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), c.log)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *skillParameterRuntimeClient) projectStatus(
	resource *odav1beta1.SkillParameter,
	snapshot *skillParameterSnapshot,
) error {
	if resource == nil {
		return fmt.Errorf("SkillParameter resource is nil")
	}
	if snapshot == nil {
		return nil
	}
	resource.Status.Name = snapshot.name
	resource.Status.DisplayName = snapshot.displayName
	resource.Status.Type = snapshot.parameterType
	resource.Status.Value = snapshot.value
	resource.Status.LifecycleState = snapshot.lifecycleState
	resource.Status.Description = snapshot.description
	return nil
}

func (c *skillParameterRuntimeClient) recordResponseRequestID(resource *odav1beta1.SkillParameter, response any) {
	if resource == nil {
		return
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
}

func (c *skillParameterRuntimeClient) recordErrorRequestID(resource *odav1beta1.SkillParameter, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func (c *skillParameterRuntimeClient) ensureClient() error {
	if c.initErr != nil {
		return fmt.Errorf("initialize SkillParameter OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return fmt.Errorf("SkillParameter OCI client is not configured")
	}
	return nil
}

func resolveDesiredSkillParameterIdentity(resource *odav1beta1.SkillParameter) (skillParameterIdentity, error) {
	if resource == nil {
		return skillParameterIdentity{}, fmt.Errorf("SkillParameter resource is nil")
	}
	identity := skillParameterIdentity{
		odaInstanceID: annotationValue(resource, skillParameterOdaInstanceIDAnnotation, skillParameterLegacyOdaInstanceIDAnnotation),
		skillID:       annotationValue(resource, skillParameterSkillIDAnnotation, skillParameterLegacySkillIDAnnotation),
		name:          strings.TrimSpace(resource.Spec.Name),
	}
	if identity.name == "" {
		identity.name = strings.TrimSpace(resource.Status.Name)
	}
	if identity.name == "" {
		identity.name = strings.TrimSpace(resource.Name)
	}
	return identity, identity.validate()
}

func resolveDeleteSkillParameterIdentity(resource *odav1beta1.SkillParameter) (skillParameterIdentity, bool, error) {
	if identity, ok := legacyTrackedSkillParameterIdentity(resource); ok {
		return identity, true, nil
	}
	identity, err := resolveDesiredSkillParameterIdentity(resource)
	if err == nil {
		if err := validateTrackedSkillParameterIdentity(resource, identity); err != nil {
			return skillParameterIdentity{}, true, err
		}
		return identity, true, nil
	}
	if _, ok := trackedSkillParameterFingerprint(resource); ok {
		return skillParameterIdentity{}, true, fmt.Errorf(
			"SkillParameter delete requires %s and %s because status only records a bounded identity fingerprint; restore the parent annotations before deleting: %w",
			skillParameterOdaInstanceIDAnnotation,
			skillParameterSkillIDAnnotation,
			err,
		)
	}
	return skillParameterIdentity{}, false, nil
}

func validateTrackedSkillParameterIdentity(
	resource *odav1beta1.SkillParameter,
	desired skillParameterIdentity,
) error {
	trackedFingerprint, ok := trackedSkillParameterFingerprint(resource)
	if !ok {
		return nil
	}
	desiredFingerprint := desired.fingerprint()
	if trackedFingerprint == desiredFingerprint {
		return nil
	}
	return fmt.Errorf(
		"SkillParameter identity is immutable: tracked fingerprint %q, desired odaInstanceId/skillId/name %q/%q/%q fingerprint %q",
		trackedFingerprint,
		desired.odaInstanceID,
		desired.skillID,
		desired.name,
		desiredFingerprint,
	)
}

func (identity skillParameterIdentity) validate() error {
	var missing []string
	if strings.TrimSpace(identity.odaInstanceID) == "" {
		missing = append(missing, skillParameterOdaInstanceIDAnnotation)
	}
	if strings.TrimSpace(identity.skillID) == "" {
		missing = append(missing, skillParameterSkillIDAnnotation)
	}
	if strings.TrimSpace(identity.name) == "" {
		missing = append(missing, "spec.name")
	}
	if len(missing) > 0 {
		return fmt.Errorf("SkillParameter requires %s because the generated API omits OCI parent path fields", strings.Join(missing, ", "))
	}
	return nil
}

func annotationValue(resource *odav1beta1.SkillParameter, keys ...string) string {
	if resource == nil {
		return ""
	}
	for _, key := range keys {
		value := strings.TrimSpace(resource.Annotations[key])
		if value != "" {
			return value
		}
	}
	return ""
}

func validateDesiredSkillParameterType(resource *odav1beta1.SkillParameter) error {
	if resource == nil {
		return fmt.Errorf("SkillParameter resource is nil")
	}
	parameterType := normalizeSkillParameterType(resource.Spec.Type)
	if parameterType == "" {
		return fmt.Errorf("SkillParameter spec.type is required")
	}
	if _, ok := odasdk.GetMappingParameterTypeEnum(parameterType); !ok {
		return fmt.Errorf("unsupported SkillParameter type %q", resource.Spec.Type)
	}
	return nil
}

func validateSkillParameterCreateOnlyDrift(
	resource *odav1beta1.SkillParameter,
	current *skillParameterSnapshot,
) error {
	if resource == nil || current == nil {
		return nil
	}
	if current.name != "" && strings.TrimSpace(resource.Spec.Name) != current.name {
		return fmt.Errorf("SkillParameter name is immutable: current %q, desired %q", current.name, resource.Spec.Name)
	}
	currentType := normalizeSkillParameterType(current.parameterType)
	desiredType := normalizeSkillParameterType(resource.Spec.Type)
	if currentType != "" && desiredType != "" && currentType != desiredType {
		return fmt.Errorf("SkillParameter type is immutable: current %q, desired %q", current.parameterType, resource.Spec.Type)
	}
	return nil
}

func buildSkillParameterUpdateDetails(
	resource *odav1beta1.SkillParameter,
	current *skillParameterSnapshot,
) (odasdk.UpdateSkillParameterDetails, bool) {
	if resource == nil || current == nil {
		return odasdk.UpdateSkillParameterDetails{}, false
	}

	var details odasdk.UpdateSkillParameterDetails
	updateNeeded := false
	if resource.Spec.DisplayName != current.displayName {
		details.DisplayName = stringPtr(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.Description != current.description {
		details.Description = stringPtr(resource.Spec.Description)
		updateNeeded = true
	}
	if resource.Spec.Value != current.value {
		details.Value = stringPtr(resource.Spec.Value)
		updateNeeded = true
	}
	return details, updateNeeded
}

func skillParameterSnapshotFromSDK(parameter odasdk.SkillParameter) *skillParameterSnapshot {
	return &skillParameterSnapshot{
		name:           stringValue(parameter.Name),
		displayName:    stringValue(parameter.DisplayName),
		parameterType:  normalizeSkillParameterType(string(parameter.Type)),
		value:          stringValue(parameter.Value),
		lifecycleState: normalizeSkillParameterLifecycle(string(parameter.LifecycleState)),
		description:    stringValue(parameter.Description),
	}
}

func skillParameterSnapshotFromSummary(parameter odasdk.SkillParameterSummary) skillParameterSnapshot {
	return skillParameterSnapshot{
		name:           stringValue(parameter.Name),
		displayName:    stringValue(parameter.DisplayName),
		parameterType:  normalizeSkillParameterType(string(parameter.Type)),
		value:          stringValue(parameter.Value),
		lifecycleState: normalizeSkillParameterLifecycle(string(parameter.LifecycleState)),
		description:    stringValue(parameter.Description),
	}
}

func skillParameterLifecycleMessage(snapshot *skillParameterSnapshot) string {
	if snapshot == nil {
		return "OCI SkillParameter is active"
	}
	if snapshot.displayName != "" {
		return snapshot.displayName
	}
	if snapshot.name != "" {
		return snapshot.name
	}
	return "OCI SkillParameter is active"
}

func normalizeSkillParameterLifecycle(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isSkillParameterSteadyLifecycle(value string) bool {
	switch normalizeSkillParameterLifecycle(value) {
	case string(odasdk.LifecycleStateActive), string(odasdk.LifecycleStateInactive):
		return true
	default:
		return false
	}
}

func normalizeSkillParameterType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func skillParameterIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return servicemanager.IsNotFoundServiceError(err) ||
		servicemanager.IsNotFoundErrorString(err) ||
		errors.Is(err, errSkillParameterNotFound)
}

var errSkillParameterNotFound = errors.New("skillparameter not found")

func stringPtr(value string) *string {
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (identity skillParameterIdentity) syntheticOCID() shared.OCID {
	return shared.OCID(skillParameterSyntheticIDPrefix + skillParameterSyntheticIDVersion + identity.fingerprint())
}

func (identity skillParameterIdentity) fingerprint() string {
	hash := sha256.New()
	for _, value := range []string{identity.odaInstanceID, identity.skillID, identity.name} {
		_, _ = hash.Write([]byte(strings.TrimSpace(value)))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func trackedSkillParameterFingerprint(resource *odav1beta1.SkillParameter) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if !strings.HasPrefix(raw, skillParameterSyntheticIDPrefix) {
		return "", false
	}

	suffix := strings.TrimPrefix(raw, skillParameterSyntheticIDPrefix)
	if strings.HasPrefix(suffix, skillParameterSyntheticIDVersion) {
		fingerprint := strings.TrimPrefix(suffix, skillParameterSyntheticIDVersion)
		if isSkillParameterFingerprint(fingerprint) {
			return fingerprint, true
		}
		return "", false
	}

	identity, ok := legacyTrackedSkillParameterIdentity(resource)
	if !ok {
		return "", false
	}
	return identity.fingerprint(), true
}

func legacyTrackedSkillParameterIdentity(resource *odav1beta1.SkillParameter) (skillParameterIdentity, bool) {
	if resource == nil {
		return skillParameterIdentity{}, false
	}
	raw := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if !strings.HasPrefix(raw, skillParameterSyntheticIDPrefix) {
		return skillParameterIdentity{}, false
	}

	suffix := strings.TrimPrefix(raw, skillParameterSyntheticIDPrefix)
	if strings.HasPrefix(suffix, skillParameterSyntheticIDVersion) {
		return skillParameterIdentity{}, false
	}

	parts := strings.Split(suffix, "/")
	if len(parts) != 3 {
		return skillParameterIdentity{}, false
	}
	odaInstanceID, ok := decodeSkillParameterIdentityPart(parts[0])
	if !ok {
		return skillParameterIdentity{}, false
	}
	skillID, ok := decodeSkillParameterIdentityPart(parts[1])
	if !ok {
		return skillParameterIdentity{}, false
	}
	name, ok := decodeSkillParameterIdentityPart(parts[2])
	if !ok {
		return skillParameterIdentity{}, false
	}

	identity := skillParameterIdentity{
		odaInstanceID: odaInstanceID,
		skillID:       skillID,
		name:          name,
	}
	return identity, identity.validate() == nil
}

func legacySyntheticSkillParameterOCID(identity skillParameterIdentity) shared.OCID {
	return shared.OCID(skillParameterSyntheticIDPrefix +
		legacyEncodeSkillParameterIdentityPart(identity.odaInstanceID) + "/" +
		legacyEncodeSkillParameterIdentityPart(identity.skillID) + "/" +
		legacyEncodeSkillParameterIdentityPart(identity.name))
}

func legacyEncodeSkillParameterIdentityPart(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(value)))
}

func decodeSkillParameterIdentityPart(value string) (string, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}

func isSkillParameterFingerprint(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
